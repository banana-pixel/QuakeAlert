// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package event

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/banana-pixel/quakealert/server/internal/dispatch"
	"github.com/banana-pixel/quakealert/server/internal/ingest"
)

// Fixture tepi-PRELIM Admin Node (keputusan D-043).
//
// Seluruh uji di berkas ini menggerakkan Tracker PRODUKSI lewat harness yang
// sama dengan tepi-FINAL (input deterministik, jam palsu, tanpa I/O). Yang
// dibuktikan: PRELIM pertama milik Admin Node selagi UNCONFIRMED yang tidak
// menghasilkan transisi menghasilkan TEPAT SATU advisory pembaruan bukti —
// tipe EARTHQUAKE_ADVISORY, push=false, tanpa trusted_local, tanpa revisi,
// tanpa baris log — dan tidak ada pola lain yang menghasilkannya. Jalur
// tepi-FINAL, trusted-local, dan D-040 tidak berubah (diuji oleh berkas lain
// yang tidak disentuh di sini; sebagian ditegaskan ulang lewat uji
// koeksistensi di bawah).

// v2admin membangun observasi v2 milik Admin Node: helper v2() yang ada tidak
// disentuh (dipakai ratusan uji lain), lencana admin ditempel di sini supaya
// cakupannya eksplisit per uji.
func v2admin(node string, pga float64, onset int64, phase string, obsSeq int64) Input {
	in := v2(node, pga, onset, phase, obsSeq)
	in.IsAdminNode = true
	return in
}

// lockedSink adalah recSink yang aman-konkuren: emisi tepi-PRELIM bersifat
// async (goroutine di Bridge, seperti tepi-FINAL) sehingga polling uji
// berpacu dengan DispatchEventFrame. recSink milik emit_test (tanpa kunci,
// untuk pemakaian sinkron) tidak disentuh — pembungkus ini lokal berkas ini.
type lockedSink struct {
	mu    sync.Mutex
	msgs  []*dispatch.AlertMessage
	pushs []bool
}

func (s *lockedSink) DispatchEventFrame(_ context.Context, msg *dispatch.AlertMessage, push bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, msg)
	s.pushs = append(s.pushs, push)
}

func (s *lockedSink) snapshot() (msgs []*dispatch.AlertMessage, pushs []bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*dispatch.AlertMessage(nil), s.msgs...),
		append([]bool(nil), s.pushs...)
}

// prelimRig mirrors eligibleEdgeRig but with a race-safe sink for the async
// prelim path. Existing rig/helpers are untouched.
type prelimRig struct {
	sink  *lockedSink
	local *fakeLocalSink
	h     *harness
}

func newPrelimRig(t *testing.T) *prelimRig {
	t.Helper()
	h := newHarness(t)
	sink := &lockedSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)
	h.trk.SetEmitter(b)
	h.node(adminTestStation, -6.87, 107.54)
	return &prelimRig{sink: sink, local: local, h: h}
}

// waitForSinkAdvisory menunggu advisory ke-n pada sink normal.
func waitForSinkAdvisory(t *testing.T, r *prelimRig, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sinkTypeCount(r, dispatch.TypeAdvisory) >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("frame advisory tidak mencapai %d", want)
}

// sinkTypeCount menghitung frame normal per tipe (snapshot berkunci).
func sinkTypeCount(r *prelimRig, typ string) int {
	msgs, _ := r.sink.snapshot()
	n := 0
	for _, m := range msgs {
		if m.Type == typ {
			n++
		}
	}
	return n
}

// prelimAdvisories mengembalikan semua advisory pada sink dalam urutan emisi,
// beserta flag push masing-masing.
func prelimAdvisories(r *prelimRig) (advs []*dispatch.AlertMessage, pushs []bool) {
	msgs, ps := r.sink.snapshot()
	for i, m := range msgs {
		if m.Type == dispatch.TypeAdvisory {
			advs = append(advs, m)
			pushs = append(pushs, ps[i])
		}
	}
	return advs, pushs
}

// ---- 1: matriks kelayakan ---------------------------------------------------

func TestEvaluateAdminNodePrelim(t *testing.T) {
	eligible := adminEligibleState()
	prelimOther := adminSnap(PhasePrelim, 150)
	prelimOther.Evidence.Contributors[0].NodeID = "NODE-0A1B2C3D"
	cases := []struct {
		name   string
		state  AdminNodeState
		snap   Snapshot
		reason string
	}{
		{"positif PRELIM 150 gal denyut segar",
			eligible, adminSnap(PhasePrelim, 150), ReasonAdminNodePrelimEligible},
		// TANPA lantai PGA alarm: 10 gal (< 60) tetap layak sebagai pembaruan
		// bukti. Inilah beda utamanya dengan gerbang alarm.
		{"PRELIM 10 gal di bawah lantai alarm tetap layak",
			eligible, adminSnap(PhasePrelim, 10), ReasonAdminNodePrelimEligible},
		{"tanpa designate",
			AdminNodeState{}, adminSnap(PhasePrelim, 150), ReasonAdminNodeNoneDesignated},
		{"designate tetapi unverified",
			AdminNodeState{Designated: true, StationID: adminTestStation, HeartbeatAge: time.Minute},
			adminSnap(PhasePrelim, 150), ReasonAdminNodeUnverified},
		{"state CONFIRMED bukan UNCONFIRMED",
			eligible, snapTo(StateConfirmed, true), ReasonAdminNodeNotUnconfirmed},
		{"kontribusi milik node lain",
			eligible, prelimOther, ReasonAdminNodeNoContribution},
		{"tanpa kontributor",
			eligible, snapTo(StateUnconfirmed, false), ReasonAdminNodeNoContribution},
		{"kontribusi admin berfase FINAL, bukan PRELIM",
			eligible, adminSnap(PhaseFinal, 150), ReasonAdminNodeNotPrelim},
		{"heartbeat basi 5m1s",
			AdminNodeState{Designated: true, StationID: adminTestStation, Verified: true, HeartbeatAge: AdminNodeHeartbeatMaxAge + time.Second},
			adminSnap(PhasePrelim, 150), ReasonAdminNodeHeartbeatStale},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := EvaluateAdminNodePrelim(c.state, c.snap)
			want := c.reason == ReasonAdminNodePrelimEligible
			if d.Eligible != want || d.Reason != c.reason {
				t.Errorf("keputusan = (%v,%q), mau (%v,%q)",
					d.Eligible, d.Reason, want, c.reason)
			}
		})
	}
}

// ---- 2-8: tepi menyala tepat sekali -----------------------------------------

func TestAdminPrelimEdge_FiresExactlyOnce(t *testing.T) {
	r := newPrelimRig(t)
	r.h.node("NODE-B", -6.87, 107.55)

	// PRELIM non-admin lebih dulu: menciptakan event + advisory transisi #1.
	r.h.ingest(v2("NODE-B", 60, edgeOnset, PhasePrelim, 700))
	waitForSinkAdvisory(t, r, 1)

	// PRELIM pertama milik admin selagi UNCONFIRMED, tanpa transisi: tepi
	// menyala -> advisory pembaruan bukti #2.
	r.h.ingest(v2admin(adminTestStation, 90, edgeOnset, PhasePrelim, 701))
	waitForSinkAdvisory(t, r, 2)

	// PRELIM admin kedua (obs_seq baru): bendera tepat-sekali menahannya.
	r.h.ingest(v2admin(adminTestStation, 95, edgeOnset, PhasePrelim, 702))
	time.Sleep(50 * time.Millisecond)

	advs, pushs := prelimAdvisories(r)
	if len(advs) != 2 {
		t.Fatalf("frame advisory = %d, mau tepat 2 (transisi + tepi)", len(advs))
	}
	first, second := advs[0], advs[1]
	// Tipe advisory, bukan alert — dari kedua sumber.
	if first.Type != dispatch.TypeAdvisory || second.Type != dispatch.TypeAdvisory {
		t.Fatalf("tipe = %q/%q, mau keduanya ADVISORY", first.Type, second.Type)
	}
	// Korelasi: event_id dan revisi yang sama — tepi tidak menaikkan revisi.
	if first.EventID == "" || second.EventID != first.EventID {
		t.Fatalf("event_id = %q/%q, mau sama dan tak-kosong", first.EventID, second.EventID)
	}
	if first.EventRevision != second.EventRevision {
		t.Fatalf("revisi = %d/%d, mau sama (tepi bukan transisi)",
			first.EventRevision, second.EventRevision)
	}
	// Bukti diperbarui: advisory tepi membawa PGA admin yang lebih baru.
	if second.PGAGal < first.PGAGal {
		t.Fatalf("pga advisory tepi = %v < transisi %v, mau bukti lebih baru",
			second.PGAGal, first.PGAGal)
	}
	// push=false di kedua advisory (WS saja, D10): tidak ada yang membangunkan
	// latar. Tidak ada trusted_local di keduanya.
	for i, pushed := range pushs {
		if pushed {
			t.Fatalf("advisory #%d push=true, mau false (WS saja)", i+1)
		}
		if advs[i].TrustedLocal {
			t.Fatalf("advisory #%d bertanda trusted_local", i+1)
		}
	}
	// Jalur lokal diam total: PRELIM tidak pernah menjadi alarm.
	if got := r.local.count(); got != 0 {
		t.Fatalf("frame lokal = %d, mau 0 (PRELIM bukan alarm)", got)
	}
}

// ---- deduplikasi terhadap advisory transisi ---------------------------------

func TestAdminPrelimEdge_NoDuplicateWhenInputTransitions(t *testing.T) {
	r := newPrelimRig(t)

	// PRELIM admin SEBAGAI kontribusi pertama: menciptakan event + advisory
	// transisi. Tepi wajib diam — kabarnya sudah dibawa transisi itu.
	r.h.ingest(v2admin(adminTestStation, 90, edgeOnset, PhasePrelim, 700))
	waitForSinkAdvisory(t, r, 1)
	time.Sleep(50 * time.Millisecond)

	if got := sinkTypeCount(r, dispatch.TypeAdvisory); got != 1 {
		t.Fatalf("frame advisory = %d, mau tepat 1 (tanpa duplikat tepi)", got)
	}
	if got := r.local.count(); got != 0 {
		t.Fatalf("frame lokal = %d, mau 0", got)
	}
}

// ---- mata-mata emitter: tanpa transisi, tanpa baris --------------------------

type prelimSpyEmitter struct {
	mu          sync.Mutex
	transitions []Snapshot
	prelimEdges []Snapshot
	finalEdges  []Snapshot
}

func (s *prelimSpyEmitter) EmitTransition(_ context.Context, snap Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transitions = append(s.transitions, snap)
}

func (s *prelimSpyEmitter) EmitAdminEdge(_ context.Context, snap Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.finalEdges = append(s.finalEdges, snap)
}

func (s *prelimSpyEmitter) EmitAdminPrelimEdge(_ context.Context, snap Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prelimEdges = append(s.prelimEdges, snap)
}

func (s *prelimSpyEmitter) counts() (trans, prelim, final int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.transitions), len(s.prelimEdges), len(s.finalEdges)
}

// Tepi-PRELIM bukan transisi: untuk masukan pemicunya, Tracker tidak
// menghasilkan snapshot transisi (sumber satu-satunya baris log dan revisi).
// Sinkron penuh di sini karena mata-mata mencatat di goroutine pemanggil
// (async hanya terjadi di dalam Bridge, yang tidak dipakai di uji ini).
func TestAdminPrelimEdge_EmitsNoTransition(t *testing.T) {
	h := newHarness(t)
	spy := &prelimSpyEmitter{}
	h.trk.SetEmitter(spy)
	h.node("NODE-B", -6.87, 107.55)
	h.node(adminTestStation, -6.87, 107.56)

	h.ingest(v2("NODE-B", 60, edgeOnset, PhasePrelim, 700))
	h.ingest(v2admin(adminTestStation, 90, edgeOnset, PhasePrelim, 701))

	if trans, prelim, final := spy.counts(); trans != 1 || prelim != 1 || final != 0 {
		t.Fatalf("transisi/prelim/final = %d/%d/%d, mau 1/1/0",
			trans, prelim, final)
	}
	// Revisi tidak bergerak oleh tepi: satu-satunya transisi bervisi 1 dan
	// snapshot tepi mewarisinya tanpa increment.
	evs := h.events()
	if len(evs) != 1 {
		t.Fatalf("event terlacak = %d, mau 1", len(evs))
	}
	for _, e := range evs {
		if e.Revision != 1 {
			t.Fatalf("revisi event = %d, mau 1 (tepi bukan transisi)", e.Revision)
		}
		if !e.adminPrelimEdgeFired {
			t.Error("bendera adminPrelimEdgeFired tidak disetel")
		}
	}
}

// ---- koeksistensi dengan tepi-FINAL ------------------------------------------

func TestAdminPrelimEdge_FinalEdgeStillFires(t *testing.T) {
	r := newPrelimRig(t)
	r.h.node("NODE-B", -6.87, 107.55)

	r.h.ingest(v2("NODE-B", 60, edgeOnset, PhasePrelim, 700))
	r.h.ingest(v2admin(adminTestStation, 90, edgeOnset, PhasePrelim, 701))
	waitForSinkAdvisory(t, r, 2)

	// FINAL admin: jalur tepi-FINAL (D-037) utuh — satu frame lokal bertanda,
	// dan tidak ada advisory ketiga dari tepi mana pun.
	r.h.ingest(v2admin(adminTestStation, 150, edgeOnset, PhaseFinal, 701))
	waitForLocal(t, r.local, 1)
	time.Sleep(50 * time.Millisecond)

	if got := sinkTypeCount(r, dispatch.TypeAdvisory); got != 2 {
		t.Fatalf("frame advisory = %d, mau tetap 2", got)
	}
	if got := r.local.count(); got != 1 {
		t.Fatalf("frame lokal = %d, mau tepat 1 (FINAL)", got)
	}
	if m := r.local.first(); m == nil || !m.TrustedLocal || m.Type != dispatch.TypeAlert {
		t.Fatalf("frame lokal = %+v; mau ALERT bertanda", m)
	}
}

// ---- D-040 tidak berutang pada advisory --------------------------------------

func TestAdminPrelimEdge_ResolveOwesNoAllClear(t *testing.T) {
	r := newPrelimRig(t)
	r.h.node("NODE-B", -6.87, 107.55)

	r.h.ingest(v2("NODE-B", 60, edgeOnset, PhasePrelim, 700))
	r.h.ingest(v2admin(adminTestStation, 90, edgeOnset, PhasePrelim, 701))
	waitForSinkAdvisory(t, r, 2)

	// Sapuan resolve: RESOLVED normal (EverConfirmed=false -> WS saja, §8.1)
	// dan TIDAK ADA all-clear lokal — PRELIM tidak pernah membangunkan
	// audiens 20 km sehingga tidak ada utang (D-040 tak tersentuh).
	r.h.clock.advance(100 * time.Second)
	r.h.trk.sweep(context.Background())
	time.Sleep(50 * time.Millisecond)

	if got := sinkTypeCount(r, dispatch.TypeResolved); got != 1 {
		t.Fatalf("frame RESOLVED = %d, mau 1", got)
	}
	if got := r.local.count(); got != 0 {
		t.Fatalf("frame lokal = %d, mau 0 (tanpa alarm, tanpa all-clear)", got)
	}
}

// ---- replay tetap tuli -------------------------------------------------------

func TestReplayPrelimEdge_Deaf(t *testing.T) {
	// Baris ledger PRELIM-lalu-FINAL milik admin: replay membangun Input
	// TANPA lencana (replayInputFrom tidak mengenalnya) di atas Tracker
	// TANPA emitter tepi — Frames murni transisi, Compare steril.
	rows := rpObs(
		rpRow{id: 301, node: adminTestStation, phase: PhasePrelim, pga: 90, durMs: 6000,
			publish: 4006147, received: 4006164, onset: 4000000, upper: 4000009, seq: 800,
			lat: -6.870, lon: 107.540},
		rpRow{id: 302, node: adminTestStation, phase: PhaseFinal, pga: 150, durMs: 6000,
			publish: 4012147, received: 4012164, onset: 4000000, upper: 4000009, seq: 800,
			lat: -6.870, lon: 107.540},
	)
	res, err := Replay(context.Background(), rows, rpProfile())
	if err != nil {
		t.Fatalf("Replay galat: %v", err)
	}
	advisories := 0
	for _, f := range res.Frames {
		if f.To == StateUnconfirmed {
			advisories++
		}
	}
	if advisories != 1 {
		t.Fatalf("frame advisory replay = %d, mau tepat 1 (transisi saja)", advisories)
	}
}

// ---- plumbing lencana ---------------------------------------------------------

func TestObservationFromCopiesAdminFlag(t *testing.T) {
	tr := &ingest.Trigger{NodeID: "NODE-AD000001", IsAdminNode: true}
	if got := ObservationFrom(tr); !got.IsAdminNode {
		t.Error("ObservationFrom membuang lencana admin")
	}
	tr = &ingest.Trigger{NodeID: "NODE-0A1B2C3D"}
	if got := ObservationFrom(tr); got.IsAdminNode {
		t.Error("ObservationFrom mengarang lencana admin")
	}
}
