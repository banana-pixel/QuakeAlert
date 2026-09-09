package event

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/banana-pixel/quakealert/server/internal/dispatch"
	"github.com/banana-pixel/quakealert/server/internal/store"
)

// Fixture replay Admin Node (D-036 PROPOSED).
//
// Seluruh designasi di berkas ini datang dari PROFIL OPERATOR
// (ReplayProfile.AdminNode) — tidak satu pun dibaca dari baris historis,
// karena tidak ada tabel yang mencatat siapa yang ditunjuk, kapan
// diverifikasi, atau kapan denyut terakhir. Itulah yang diuji di sini:
// "observasi ini, DI BAWAH DESIGNASI INI, menghasilkan keputusan lokal itu".
//
// Hasil AdminOutcomes DILAPORKAN, tidak dibandingkan dengan riwayat (tidak
// ada trusted_local historis sebagai pembanding), dan Compare() tidak
// menyentuhnya.

// ---- pembangun fixture ----------------------------------------------------

const rpAdminNode = "NODE-ADM1"

// rpAdminSingleRows adalah satu observasi milik node admin: bentuk minimal
// yang menghasilkan transisi DETECTED->UNCONFIRMED lalu RESOLVED oleh sweep.
func rpAdminSingleRows(pga float64, phase string) []store.ReplayObservation {
	return rpObs(
		rpRow{id: 101, node: rpAdminNode, phase: phase, pga: pga, durMs: 6000,
			publish: 3006147, received: 3006164, onset: 3000000, upper: 3000009, seq: 500,
			lat: -6.870, lon: 107.540},
	)
}

// rpAdminProfile membangun profil dengan designasi operator eksplisit.
func rpAdminProfile(station string, verified bool, age time.Duration) ReplayProfile {
	p := rpProfile()
	p.AdminNode = &ReplayAdminNode{StationID: station, Verified: verified, HeartbeatAge: age}
	return p
}

// outcomeFor mengembalikan outcome frame UNCONFIRMED revisi pertama — bentuk
// yang dinilai gerbang lokal — beserta snapshot-nya.
func outcomeFor(t *testing.T, res *ReplayResult) (AdminOutcome, Snapshot) {
	t.Helper()
	if len(res.Frames) != len(res.AdminOutcomes) {
		t.Fatalf("frame = %d, outcome = %d; mau sejajar", len(res.Frames), len(res.AdminOutcomes))
	}
	for i, f := range res.Frames {
		if f.To == StateUnconfirmed {
			return res.AdminOutcomes[i], f
		}
	}
	t.Fatalf("tidak ada frame UNCONFIRMED di %v", rpStates(res.Frames))
	return AdminOutcome{}, Snapshot{}
}

// assertSameFrames membuktikan designasi tidak menggeser SATU PUN keputusan
// Tracker: perbandingan keputusan identik dengan TestReplayIsDeterministic.
func assertSameFrames(t *testing.T, a, b []Snapshot) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("jumlah frame = %d vs %d", len(a), len(b))
	}
	for i := range a {
		x, y := a[i], b[i]
		if x.EventID != y.EventID || x.Revision != y.Revision || x.To != y.To ||
			x.From != y.From || x.Reason != y.Reason || x.DecidedAt != y.DecidedAt ||
			x.NodeCount != y.NodeCount || x.IndependentCells != y.IndependentCells ||
			round4(x.PeakPGA) != round4(y.PeakPGA) ||
			string(x.Evidence.JSON()) != string(y.Evidence.JSON()) {
			t.Fatalf("frame[%d] menyimpang:\n a=%+v\n b=%+v", i, x, y)
		}
	}
}

func adminOutcomeKey(out []AdminOutcome) string {
	var b strings.Builder
	for _, o := range out {
		fmt.Fprintf(&b, "%s#%d#%v#%s;", o.EventID, o.Revision, o.Eligible, o.Reason)
	}
	return b.String()
}

// ---- positif: layak -> trusted-local lewat jalur produksi ------------------

func TestReplayAdmin_EligibleProducesTrustedLocal(t *testing.T) {
	rows := rpAdminSingleRows(150, PhaseFinal)
	res, err := Replay(context.Background(), rows, rpAdminProfile(rpAdminNode, true, time.Minute))
	if err != nil {
		t.Fatalf("Replay galat: %v", err)
	}

	out, snap := outcomeFor(t, res)
	if !out.Eligible || out.Reason != ReasonAdminNodeEligible {
		t.Fatalf("outcome = (%v,%q); mau (true,ADMIN_ELIGIBLE)", out.Eligible, out.Reason)
	}

	// Kawat: frame dibangun lewat choke point produksi yang sama
	// (TrustedLocalFrameFor), bukan konstruksi uji.
	msg, ok := TrustedLocalFrameFor(AdminNodeState{
		Designated: true, StationID: rpAdminNode, Verified: true, HeartbeatAge: time.Minute,
	}, snap)
	if !ok {
		t.Fatal("TrustedLocalFrameFor menolak kasus yang dinilai layak")
	}
	if msg.Type != dispatch.TypeAlert {
		t.Errorf("type = %q; mau EARTHQUAKE_ALERT (tanpa enum baru)", msg.Type)
	}
	if got := dispatch.BuildAlertData(msg)["trusted_local"]; got != "true" {
		t.Errorf("data trusted_local = %q; mau \"true\"", got)
	}
	if msg.EventID != snap.EventID || msg.EventRevision != snap.Revision {
		t.Error("korelasi event_id/revisi rusak pada frame lokal")
	}

	// Designasi tidak menggeser keputusan Tracker: frame identik dengan
	// pemutaran tanpa designasi.
	plain, err := Replay(context.Background(), rows, rpProfile())
	if err != nil {
		t.Fatalf("Replay tanpa designasi galat: %v", err)
	}
	assertSameFrames(t, plain.Frames, res.Frames)
	if len(plain.AdminOutcomes) != 0 {
		t.Fatalf("tanpa designasi outcome = %d; mau kosong (fitur mati, bukan ditolak)",
			len(plain.AdminOutcomes))
	}
}

// ---- batas: 140 gal dan 5 menit --------------------------------------------

func TestReplayAdmin_Boundaries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		pga    float64
		age    time.Duration
		reason string
	}{
		{"tepat 140 gal layak", 140.0, time.Minute, ReasonAdminNodeEligible},
		{"139.999 gal ditolak", 139.999, time.Minute, ReasonAdminNodeBelowFloor},
		{"tepat 5 menit layak", 150, AdminNodeHeartbeatMaxAge, ReasonAdminNodeEligible},
		{"5 menit 1 detik basi", 150, AdminNodeHeartbeatMaxAge + time.Second, ReasonAdminNodeHeartbeatStale},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Replay(context.Background(),
				rpAdminSingleRows(tc.pga, PhaseFinal),
				rpAdminProfile(rpAdminNode, true, tc.age))
			if err != nil {
				t.Fatalf("Replay galat: %v", err)
			}
			out, _ := outcomeFor(t, res)
			want := tc.reason == ReasonAdminNodeEligible
			if out.Eligible != want || out.Reason != tc.reason {
				t.Errorf("outcome = (%v,%q); mau (%v,%q)",
					out.Eligible, out.Reason, want, tc.reason)
			}
		})
	}
}

// ---- negatif: PRELIM, unverified, bukan kontributor, basi -------------------

func TestReplayAdmin_Negatives(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rows     func() []store.ReplayObservation
		station  string
		verified bool
		age      time.Duration
		reason   string
	}{
		{"PRELIM bukan FINAL", func() []store.ReplayObservation {
			return rpAdminSingleRows(300, PhasePrelim)
		}, rpAdminNode, true, time.Minute, ReasonAdminNodeNotFinal},
		{"unverified", func() []store.ReplayObservation {
			return rpAdminSingleRows(150, PhaseFinal)
		}, rpAdminNode, false, time.Minute, ReasonAdminNodeUnverified},
		{"bukan kontributor", func() []store.ReplayObservation {
			return rpAdminSingleRows(150, PhaseFinal)
		}, "NODE-LAIN", true, time.Minute, ReasonAdminNodeNoContribution},
		{"denyut basi 1 jam", func() []store.ReplayObservation {
			return rpAdminSingleRows(150, PhaseFinal)
		}, rpAdminNode, true, time.Hour, ReasonAdminNodeHeartbeatStale},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Replay(context.Background(), tc.rows(), rpAdminProfile(tc.station, tc.verified, tc.age))
			if err != nil {
				t.Fatalf("Replay galat: %v", err)
			}
			out, _ := outcomeFor(t, res)
			if out.Eligible || out.Reason != tc.reason {
				t.Errorf("outcome = (%v,%q); mau (false,%q)",
					out.Eligible, out.Reason, tc.reason)
			}
		})
	}
}

// ---- revoke / unverify: designasi hilang -> tidak ada frame lokal -----------

func TestReplayAdmin_RevokeAndUnverify(t *testing.T) {
	rows := rpAdminSingleRows(150, PhaseFinal)

	designated, err := Replay(context.Background(), rows, rpAdminProfile(rpAdminNode, true, time.Minute))
	if err != nil {
		t.Fatalf("Replay galat: %v", err)
	}
	out, _ := outcomeFor(t, designated)
	if !out.Eligible {
		t.Fatalf("dengan designasi outcome = (%v,%q); mau layak", out.Eligible, out.Reason)
	}

	// Revoke = config dihapus operator: fitur mati, outcome kosong.
	revoked, err := Replay(context.Background(), rows, rpProfile())
	if err != nil {
		t.Fatalf("Replay revoke galat: %v", err)
	}
	if len(revoked.AdminOutcomes) != 0 {
		t.Fatalf("setelah revoke outcome = %d; mau kosong", len(revoked.AdminOutcomes))
	}
	assertSameFrames(t, designated.Frames, revoked.Frames)

	// Unverify = verified=false: dinilai dan ditolak, bukan mati.
	unverified, err := Replay(context.Background(), rows, rpAdminProfile(rpAdminNode, false, time.Minute))
	if err != nil {
		t.Fatalf("Replay unverify galat: %v", err)
	}
	out, _ = outcomeFor(t, unverified)
	if out.Eligible || out.Reason != ReasonAdminNodeUnverified {
		t.Errorf("setelah unverify outcome = (%v,%q); mau (false,ADMIN_UNVERIFIED)",
			out.Eligible, out.Reason)
	}
}

// ---- restart + konkuren: deterministik --------------------------------------

func TestReplayAdmin_RestartAndConcurrent(t *testing.T) {
	rows := rpAdminSingleRows(150, PhaseFinal)
	prof := rpAdminProfile(rpAdminNode, true, time.Minute)

	// Restart = pemutaran baru dengan config yang sama (tidak ada state yang
	// bertahan di dalam mesin): hasil identik.
	first, err := Replay(context.Background(), rows, prof)
	if err != nil {
		t.Fatalf("Replay #1 galat: %v", err)
	}
	second, err := Replay(context.Background(), rows, prof)
	if err != nil {
		t.Fatalf("Replay #2 galat: %v", err)
	}
	want := adminOutcomeKey(first.AdminOutcomes)
	if got := adminOutcomeKey(second.AdminOutcomes); got != want {
		t.Fatalf("restart menyimpang:\n %s\n %s", want, got)
	}

	// Konkuren: 8 pemutaran paralel atas profil bersama — identik semua, dan
	// bersih di bawah -race (profil hanya dibaca, tidak pernah ditulis).
	const n = 8
	keys := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := Replay(context.Background(), rows, prof)
			if err != nil {
				errs[i] = err
				return
			}
			keys[i] = adminOutcomeKey(res.AdminOutcomes)
		}(i)
	}
	wg.Wait()
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("Replay paralel #%d galat: %v", i, errs[i])
		}
		if keys[i] != want {
			t.Fatalf("paralel #%d menyimpang:\n %s\n %s", i, want, keys[i])
		}
	}
}

// ---- regresi: CONFIRMED 3-node/2-sel tidak terusik designasi -----------------

func TestReplayAdmin_DoesNotPerturbConfirmed(t *testing.T) {
	rows := rpMultiNodeRows()

	plain, err := Replay(context.Background(), rows, rpProfile())
	if err != nil {
		t.Fatalf("Replay galat: %v", err)
	}
	if got := rpStates(plain.Frames); len(got) != 3 ||
		got[0] != StateUnconfirmed || got[1] != StateConfirmed || got[2] != StateResolved {
		t.Fatalf("state = %v; mau [UNCONFIRMED CONFIRMED RESOLVED]", got)
	}

	// Designasi salah satu kontributor (NODE-53494D41, FINAL 300 gal, segar):
	// keputusan Tracker harus byte-identik dengan tanpa designasi.
	admin, err := Replay(context.Background(), rows, rpAdminProfile("NODE-53494D41", true, time.Minute))
	if err != nil {
		t.Fatalf("Replay berdesignasi galat: %v", err)
	}
	assertSameFrames(t, plain.Frames, admin.Frames)

	// INVARIAN: frame CONFIRMED tidak pernah memenuhi syarat lokal — gerbang
	// lokal hanya mengenal UNCONFIRMED, dan CONFIRMED memakai jalur normalnya.
	if len(admin.Frames) != len(admin.AdminOutcomes) {
		t.Fatalf("frame = %d, outcome = %d; mau sejajar", len(admin.Frames), len(admin.AdminOutcomes))
	}
	for i, f := range admin.Frames {
		if f.To == StateConfirmed {
			o := admin.AdminOutcomes[i]
			if o.Eligible || o.Reason != ReasonAdminNodeNotUnconfirmed {
				t.Errorf("frame CONFIRMED rev%d outcome = (%v,%q); mau (false,NOT_UNCONFIRMED)",
					f.Revision, o.Eligible, o.Reason)
			}
		}
	}
}
