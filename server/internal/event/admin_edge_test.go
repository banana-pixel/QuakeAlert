package event

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/banana-pixel/quakealert/server/internal/dispatch"
	"github.com/banana-pixel/quakealert/server/internal/store"
)

// Fixture tepi-FINAL Admin Node (D-037, pengecualian eksplisit D-003).
//
// Seluruh uji di berkas ini menggerakkan Tracker PRODUKSI lewat harness
// (input deterministik, jam palsu, tanpa I/O) dengan Bridge + sumber admin
// palsu + sink lokal perekam. Yang dibuktikan: tepi non-FINAL->FINAL pada
// kontributor admin yang memenuhi syarat menghasilkan TEPAT SATU frame lokal
// — tanpa transisi state, revisi, maupun baris log — dan tidak ada pola lain
// yang menghasilkannya.

// ---- rig ---------------------------------------------------------------

type edgeRig struct {
	sink  *recSink
	local *fakeLocalSink
	src   *fakeAdminSource
	h     *harness
}

func newEdgeRig(t *testing.T, state AdminNodeState, ok bool) *edgeRig {
	t.Helper()
	h := newHarness(t)
	sink := &recSink{}
	local := &fakeLocalSink{}
	src := &fakeAdminSource{state: state, ok: ok}
	b := NewBridge(sink)
	b.SetAdminNodeHook(src, local)
	h.trk.SetEmitter(b)
	h.node(adminTestStation, -6.87, 107.54)
	return &edgeRig{sink: sink, local: local, src: src, h: h}
}

func eligibleEdgeRig(t *testing.T) *edgeRig {
	t.Helper()
	return newEdgeRig(t, adminEligibleState(), true)
}

const edgeOnset = int64(1_700_000_100_000)

// ingestPrelimFinal memasukkan pasangan PRELIM lalu FINAL satu episode
// (obs_seq sama, seperti firmware) dan mengembalikan setelah keduanya
// diproses sinkron oleh Tracker. Emisi lokalnya async: pemanggil menunggu
// lewat waitForLocal atau tidur singkat untuk kasus nihil.
func (r *edgeRig) ingestPrelimFinal(pgaPrelim, pgaFinal float64) {
	r.h.ingest(v2(adminTestStation, pgaPrelim, edgeOnset, PhasePrelim, 700))
	r.h.ingest(v2(adminTestStation, pgaFinal, edgeOnset, PhaseFinal, 700))
}

// countSinkType menghitung frame normal per tipe pada sink Bridge. recSink
// (emit_test.go) tidak punya penghitung — penambahan metode ke helper milik
// berkas lain dihindari; lagipula pembacaan di sini selalu setelah ingest
// sinkron kembali, jadi tanpa kunci pun aman.
func countSinkType(sink *recSink, typ string) int {
	n := 0
	for _, m := range sink.msgs {
		if m.Type == typ {
			n++
		}
	}
	return n
}

// syncLoc adalah nodeSource yang aman-konkuren: penghitung fakeLoc milik
// harness tidak dikunci. Dipakai hanya oleh uji konkuren di berkas ini.
type syncLoc struct {
	mu    sync.Mutex
	nodes map[string]store.NodeLocation
}

func (l *syncLoc) GetNodeLocation(_ context.Context, id string) (*store.NodeLocation, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	nl, ok := l.nodes[id]
	if !ok {
		return nil, store.ErrNodeNotFound
	}
	cp := nl
	return &cp, nil
}

// ---- 1: PRELIM -> FINAL layak = tepat satu frame lokal --------------------

func TestAdminEdge_PrelimFinalEmitsExactlyOne(t *testing.T) {
	r := eligibleEdgeRig(t)

	r.ingestPrelimFinal(60, 150)
	waitForLocal(t, r.local, 1)

	if got := countSinkType(r.sink, dispatch.TypeAdvisory); got != 1 {
		t.Fatalf("frame UNCONFIRMED normal = %d, mau 1 (emisi normal utuh)", got)
	}
	if got := countSinkType(r.sink, dispatch.TypeAlert); got != 0 {
		t.Fatalf("frame CONFIRMED = %d, mau 0", got)
	}
	if got := r.local.count(); got != 1 {
		t.Fatalf("frame lokal = %d, mau tepat 1", got)
	}
	msg := r.local.first()
	if msg == nil || !msg.TrustedLocal || msg.Type != dispatch.TypeAlert {
		t.Fatalf("frame lokal = %+v; mau ALERT bertanda", msg)
	}
	// Bingkai tepi memakai revisi transisi yang melahirkannya — TANPA bump:
	// revisi naik hanya pada transisi state.
	if msg.EventRevision != 1 {
		t.Fatalf("revisi frame lokal = %d, mau 1 (tanpa increment)", msg.EventRevision)
	}
}

// ---- 2: PRELIM saja = nihil -------------------------------------------------

func TestAdminEdge_PrelimOnlyEmitsNone(t *testing.T) {
	r := eligibleEdgeRig(t)

	r.h.ingest(v2(adminTestStation, 60, edgeOnset, PhasePrelim, 700))

	if got := countSinkType(r.sink, dispatch.TypeAdvisory); got != 1 {
		t.Fatalf("frame UNCONFIRMED normal = %d, mau 1", got)
	}
	time.Sleep(50 * time.Millisecond)
	if got := r.local.count(); got != 0 {
		t.Fatalf("frame lokal = %d, mau 0 (PRELIM tidak memicu tepi)", got)
	}
}

// ---- 3+4: FINAL duplikat/retry = tetap satu ---------------------------------

func TestAdminEdge_DuplicateFinalEmitsOnce(t *testing.T) {
	r := eligibleEdgeRig(t)

	r.ingestPrelimFinal(60, 150)
	waitForLocal(t, r.local, 1)

	// Duplikat dan retry (seq sama, PGA sama/lebih tinggi): fase sudah FINAL,
	// jadi bukan tepi — tidak ada emisi kedua.
	r.h.ingest(v2(adminTestStation, 150, edgeOnset, PhaseFinal, 700))
	r.h.ingest(v2(adminTestStation, 180, edgeOnset, PhaseFinal, 700))
	time.Sleep(50 * time.Millisecond)
	if got := r.local.count(); got != 1 {
		t.Fatalf("frame lokal = %d, mau tetap 1", got)
	}
}

// ---- 5-8: tidak layak = nihil -------------------------------------------------

func TestAdminEdge_IneligibleEmitsNone(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state AdminNodeState
		ok    bool
		pga   float64
	}{
		{"di bawah lantai", adminEligibleState(), true, 139},
		{"denyut basi", AdminNodeState{Designated: true, StationID: adminTestStation, Verified: true, HeartbeatAge: time.Hour}, true, 150},
		{"unverified", AdminNodeState{Designated: true, StationID: adminTestStation, HeartbeatAge: time.Minute}, true, 150},
		{"tanpa designasi", AdminNodeState{}, false, 150},
		{"kontributor bukan admin", AdminNodeState{Designated: true, StationID: "NODE-LAIN", Verified: true, HeartbeatAge: time.Minute}, true, 150},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newEdgeRig(t, tc.state, tc.ok)
			r.ingestPrelimFinal(60, tc.pga)
			// Tepi TERCATAT (flip fase terjadi) tetapi evaluasi menolak:
			// beri goroutine async kesempatan, lalu pastikan nihil.
			time.Sleep(50 * time.Millisecond)
			if got := r.local.count(); got != 0 {
				t.Fatalf("frame lokal = %d, mau 0", got)
			}
			// Emisi normal tidak terusik oleh designasi apa pun.
			if got := countSinkType(r.sink, dispatch.TypeAdvisory); got != 1 {
				t.Fatalf("frame UNCONFIRMED normal = %d, mau 1", got)
			}
		})
	}
}

// ---- 9+10: regresi CONFIRMED/ADVISORY/RESOLVED ---------------------------------

func TestAdminEdge_ConfirmedRegression(t *testing.T) {
	r := eligibleEdgeRig(t)
	h := r.h
	h.nodeAt("NODE-B", -6.87, 107.54, 6, 90)
	h.nodeAt("NODE-C", -6.87, 107.54, 1, 0)

	h.ingest(v2(adminTestStation, 60, edgeOnset, PhasePrelim, 700))
	h.ingest(v2(adminTestStation, 300, edgeOnset, PhaseFinal, 700))
	waitForLocal(t, r.local, 1)

	h.ingest(v2("NODE-B", 300, edgeOnset+1000, PhaseFinal, 701))
	h.ingest(v2("NODE-C", 300, edgeOnset+2000, PhaseFinal, 702))
	time.Sleep(50 * time.Millisecond)

	if got := countSinkType(r.sink, dispatch.TypeAlert); got != 1 {
		t.Fatalf("frame CONFIRMED = %d, mau 1 (jalur normal utuh)", got)
	}
	// Tepi milik non-admin (B, C) dievaluasi lalu ditolak; CONFIRMED tak
	// pernah memicu jalur lokal. Total lokal tetap yang satu dari tepi admin.
	if got := r.local.count(); got != 1 {
		t.Fatalf("frame lokal = %d, mau tetap 1", got)
	}
	if m := r.local.first(); m == nil || m.EventState == string(StateConfirmed) {
		t.Fatalf("frame lokal ber-state CONFIRMED: %+v", m)
	}
}

func TestAdminEdge_ResolvedGainsNothing(t *testing.T) {
	r := eligibleEdgeRig(t)

	r.h.ingest(v2(adminTestStation, 60, edgeOnset, PhasePrelim, 700))
	if got := countSinkType(r.sink, dispatch.TypeAdvisory); got != 1 {
		t.Fatalf("frame UNCONFIRMED = %d, mau 1", got)
	}
	// Sapuan resolve: transisi terminal normal tanpa tepi dan tanpa lokal.
	r.h.clock.advance(100 * time.Second)
	r.h.trk.sweep(context.Background())
	time.Sleep(50 * time.Millisecond)
	if got := countSinkType(r.sink, dispatch.TypeResolved); got != 1 {
		t.Fatalf("frame RESOLVED = %d, mau 1", got)
	}
	if got := r.local.count(); got != 0 {
		t.Fatalf("frame lokal = %d, mau 0", got)
	}
}

// ---- 11: konkuren --------------------------------------------------------------

func TestAdminEdge_ConcurrentFinalAbsorbsOnce(t *testing.T) {
	// Tracker dirakit langsung (bukan newHarness) agar locator-nya aman
	// konkuren: fakeLoc milik harness menaikkan penghitung tanpa kunci
	// (cukup untuk pemakaian serial), sedangkan uji ini memanggil Ingest
	// dari banyak goroutine — seperti subscriber MQTT produksi, yang memang
	// konkuren. Sisanya identik: tanpa I/O, emitter tunggal Bridge uji.
	loc := &syncLoc{nodes: map[string]store.NodeLocation{
		adminTestStation: {StationID: adminTestStation, Lat: -6.87, Lon: 107.54, LocationName: "uji"},
	}}
	sink := &recSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)
	trk := NewTracker(loc, defaultOptions(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	trk.SetEmitter(b)
	ingest := func(in Input) {
		trk.Ingest(context.Background(), in)
	}
	ingest(v2(adminTestStation, 60, edgeOnset, PhasePrelim, 700))

	// 8 absorb FINAL konkuren atas kontributor yang sama: flip fase hanya
	// terjadi sekali karena upsert berjalan di bawah t.mu. -race membuktikan
	// tidak ada balapan pada flag, antrean, maupun locator.
	const n = 8
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ingest(v2(adminTestStation, 150, edgeOnset, PhaseFinal, 700))
		}()
	}
	wg.Wait()
	waitForLocal(t, local, 1)
	time.Sleep(50 * time.Millisecond)
	if got := local.count(); got != 1 {
		t.Fatalf("frame lokal konkuren = %d, mau tepat 1", got)
	}
	if got := countSinkType(sink, dispatch.TypeAdvisory); got != 1 {
		t.Fatalf("frame UNCONFIRMED normal = %d, mau 1", got)
	}
}

// ---- 13: fixture perilaku replay + recording sink -------------------------------

// Baris ledger PRELIM-lalu-FINAL (bentuk rpRow) diumpankan lewat mapper
// PRODUKSI (replayInputFrom) ke Tracker harness: perilaku replay atas data
// nyata, dengan sink perekam sebagai saksi emisi lokal.
func TestReplayAdminEdge_PrelimFinalRowsYieldOneLocalFrame(t *testing.T) {
	rows := rpObs(
		rpRow{id: 201, node: adminTestStation, phase: PhasePrelim, pga: 60, durMs: 6000,
			publish: 4006147, received: 4006164, onset: 4000000, upper: 4000009, seq: 800,
			lat: -6.870, lon: 107.540},
		rpRow{id: 202, node: adminTestStation, phase: PhaseFinal, pga: 150, durMs: 6000,
			publish: 4012147, received: 4012164, onset: 4000000, upper: 4000009, seq: 800,
			lat: -6.870, lon: 107.540},
	)

	r := eligibleEdgeRig(t)
	for _, o := range rows {
		in, err := replayInputFrom(o)
		if err != nil {
			t.Fatalf("replayInputFrom: %v", err)
		}
		r.h.ingest(in)
	}
	waitForLocal(t, r.local, 1)
	if got := r.local.count(); got != 1 {
		t.Fatalf("frame lokal replay = %d, mau tepat 1", got)
	}

	// Pemisahan yang dijaga: Replay() atas baris yang SAMA tidak mengenal
	// emisi lokal — Frames murni transisi, Compare steril.
	res, err := Replay(context.Background(), rows, rpProfile())
	if err != nil {
		t.Fatalf("Replay galat: %v", err)
	}
	for _, f := range res.Frames {
		if f.To == StateConfirmed {
			t.Fatalf("fixture satu-node mencapai CONFIRMED: %+v", f)
		}
	}
	if len(res.AdminOutcomes) != 0 {
		t.Fatalf("Replay tanpa designasi: outcome = %d, mau kosong", len(res.AdminOutcomes))
	}
}
