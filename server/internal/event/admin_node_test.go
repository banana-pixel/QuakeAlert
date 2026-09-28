// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package event

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/banana-pixel/quakealert/server/internal/dispatch"
)

// --- Kelayakan Admin Node (D-036 PROPOSED): matriks murni, tanpa Postgres ---
//
// Seluruh kasus di bawah deterministik penuh: evaluator tidak membaca basis
// data, jam, maupun state global — hanya AdminNodeState + Snapshot yang
// diberikan pemanggil.

const adminTestStation = "NODE-AD000001"

func adminEligibleState() AdminNodeState {
	return AdminNodeState{
		Designated:   true,
		StationID:    adminTestStation,
		Verified:     true,
		HeartbeatAge: time.Minute,
	}
}

// adminSnap membangun transisi UNCONFIRMED satu-kontributor milik node admin,
// dengan fase dan PGA yang dapat diatur — bentuk tepat snapshot yang
// dihasilkan Tracker untuk event satu-node.
func adminSnap(phase string, pga float64) Snapshot {
	s := snapTo(StateUnconfirmed, false)
	s.NodeCount = 1
	s.IndependentCells = 1
	s.PeakPGA = pga
	s.Evidence = EvidenceSummary{
		Contributors: []ContributorEvidence{{
			NodeID:      adminTestStation,
			PeakPGA:     pga,
			Phase:       phase,
			OnsetTS:     onsetBase,
			OnsetSource: OnsetSourceSensor,
		}},
		IndependentCells: 1,
		OriginTSSource:   OnsetSourceSensor,
	}
	return s
}

func TestEvaluateAdminNodeEligibility(t *testing.T) {
	eligible := adminEligibleState()
	cases := []struct {
		name   string
		state  AdminNodeState
		snap   Snapshot
		reason string
	}{
		{"positif FINAL 150 gal denyut segar",
			eligible, adminSnap(PhaseFinal, 150), ReasonAdminNodeEligible},
		{"batas PGA tepat 60 gal layak",
			eligible, adminSnap(PhaseFinal, AdminNodeMinPGAGal), ReasonAdminNodeEligible},
		{"batas heartbeat tepat 5 menit layak",
			AdminNodeState{Designated: true, StationID: adminTestStation, Verified: true, HeartbeatAge: AdminNodeHeartbeatMaxAge},
			adminSnap(PhaseFinal, 150), ReasonAdminNodeEligible},
		{"tanpa designate",
			AdminNodeState{}, adminSnap(PhaseFinal, 150), ReasonAdminNodeNoneDesignated},
		{"designate tetapi unverified",
			AdminNodeState{Designated: true, StationID: adminTestStation, HeartbeatAge: time.Minute},
			adminSnap(PhaseFinal, 150), ReasonAdminNodeUnverified},
		{"59.999 gal di bawah lantai",
			eligible, adminSnap(PhaseFinal, 59.999), ReasonAdminNodeBelowFloor},
		{"PRELIM bukan FINAL",
			eligible, adminSnap(PhasePrelim, 300), ReasonAdminNodeNotFinal},
		{"heartbeat basi 5m1s",
			AdminNodeState{Designated: true, StationID: adminTestStation, Verified: true, HeartbeatAge: AdminNodeHeartbeatMaxAge + time.Second},
			adminSnap(PhaseFinal, 150), ReasonAdminNodeHeartbeatStale},
		{"heartbeat basi 1 jam",
			AdminNodeState{Designated: true, StationID: adminTestStation, Verified: true, HeartbeatAge: time.Hour},
			adminSnap(PhaseFinal, 150), ReasonAdminNodeHeartbeatStale},
		{"kontribusi milik node lain",
			eligible, otherNodeSnap(), ReasonAdminNodeNoContribution},
		{"tanpa kontributor",
			eligible, snapTo(StateUnconfirmed, false), ReasonAdminNodeNoContribution},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := EvaluateAdminNodeEligibility(c.state, c.snap)
			want := c.reason == ReasonAdminNodeEligible
			if d.Eligible != want || d.Reason != c.reason {
				t.Errorf("keputusan = (%v,%q), mau (%v,%q)",
					d.Eligible, d.Reason, want, c.reason)
			}
		})
	}
}

// otherNodeSnap adalah UNCONFIRMED dari node biasa — admin ditunjuk dan segar,
// tetapi buktinya bukan miliknya.
func otherNodeSnap() Snapshot {
	s := adminSnap(PhaseFinal, 300)
	s.Evidence.Contributors[0].NodeID = "NODE-0A1B2C3D"
	return s
}

// INVARIAN: Admin Node TIDAK PERNAH menghasilkan frame lokal dari state selain
// UNCONFIRMED — khususnya CONFIRMED memakai jalur normalnya yang tidak
// berubah, dan status terminal tidak pernah membangunkan siapa pun.
func TestEvaluateAdminNodeNeverBeyondUnconfirmed(t *testing.T) {
	for _, to := range []State{StateDetected, StateConfirmed, StateResolved, StateCancelled} {
		s := adminSnap(PhaseFinal, 300)
		s.To = to
		d := EvaluateAdminNodeEligibility(adminEligibleState(), s)
		if d.Eligible || d.Reason != ReasonAdminNodeNotUnconfirmed {
			t.Errorf("state %s: keputusan = (%v,%q), mau (false,NOT_UNCONFIRMED)",
				to, d.Eligible, d.Reason)
		}
		if _, ok := TrustedLocalFrameFor(adminEligibleState(), s); ok {
			t.Errorf("state %s: frame lokal terbangun", to)
		}
	}
}

// Frame lokal: tipe tetap EARTHQUAKE_ALERT (tanpa enum/state baru),
// trusted_local=true, korelasi event_id/revisi/timestamp utuh.
func TestTrustedLocalFrameFor(t *testing.T) {
	s := adminSnap(PhaseFinal, 150)
	msg, ok := TrustedLocalFrameFor(adminEligibleState(), s)
	if !ok {
		t.Fatal("kasus positif ok = false")
	}
	if msg.Type != dispatch.TypeAlert {
		t.Errorf("type = %q, mau EARTHQUAKE_ALERT (tanpa enum baru)", msg.Type)
	}
	if !msg.TrustedLocal {
		t.Error("TrustedLocal = false, mau true")
	}
	if msg.EventID != s.EventID || msg.EventRevision != s.Revision || msg.Timestamp != s.DecidedAt {
		t.Errorf("korelasi rusak: id=%q rev=%d ts=%d", msg.EventID, msg.EventRevision, msg.Timestamp)
	}
	if msg.EventState != string(StateUnconfirmed) {
		t.Errorf("event_state = %q, mau UNCONFIRMED yang jujur", msg.EventState)
	}

	if _, ok := TrustedLocalFrameFor(adminEligibleState(), adminSnap(PhaseFinal, 59)); ok {
		t.Error("59 gal membangun frame lokal")
	}
	if _, ok := TrustedLocalFrameFor(AdminNodeState{}, s); ok {
		t.Error("tanpa designate membangun frame lokal")
	}
}

// Data FCM: "trusted_local" = "true" hanya pada frame lokal; frame normal
// (termasuk CONFIRMED) tidak menumbuhkan kunci itu.
func TestTrustedLocalFCMData(t *testing.T) {
	msg, ok := TrustedLocalFrameFor(adminEligibleState(), adminSnap(PhaseFinal, 150))
	if !ok {
		t.Fatal("kasus positif ok = false")
	}
	if got := dispatch.BuildAlertData(msg)["trusted_local"]; got != "true" {
		t.Errorf("data trusted_local = %q, mau \"true\"", got)
	}

	normal, _, ok := FrameFor(snapTo(StateConfirmed, false))
	if !ok {
		t.Fatal("FrameFor(CONFIRMED) ok = false")
	}
	if _, has := dispatch.BuildAlertData(normal)["trusted_local"]; has {
		t.Error("frame CONFIRMED normal membawa trusted_local")
	}
	plain, _, ok := FrameFor(snapTo(StateUnconfirmed, false))
	if !ok {
		t.Fatal("FrameFor(UNCONFIRMED) ok = false")
	}
	if _, has := dispatch.BuildAlertData(plain)["trusted_local"]; has {
		t.Error("frame UNCONFIRMED normal membawa trusted_local")
	}
}

// ---- Kait Bridge ------------------------------------------------------------

type fakeAdminSource struct {
	mu    sync.Mutex
	state AdminNodeState
	ok    bool
	calls int
}

func (f *fakeAdminSource) GetAdminNodeState(_ context.Context) (AdminNodeState, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.state, f.ok
}

// callCount membaca penghitung lewat kunci yang sama dengan penulisnya:
// Bridge memanggil sumber dari goroutine, jadi pembacaan tanpa kunci adalah
// data race di bawah -race.
func (f *fakeAdminSource) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type fakeLocalSink struct {
	mu   sync.Mutex
	msgs []*dispatch.AlertMessage
}

func (f *fakeLocalSink) DispatchTrustedLocalEventFrame(_ context.Context, msg *dispatch.AlertMessage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.msgs = append(f.msgs, msg)
}

func (f *fakeLocalSink) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.msgs)
}

func (f *fakeLocalSink) first() *dispatch.AlertMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.msgs) == 0 {
		return nil
	}
	return f.msgs[0]
}

func (f *fakeLocalSink) at(i int) *dispatch.AlertMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i < 0 || i >= len(f.msgs) {
		return nil
	}
	return f.msgs[i]
}

func waitForLocal(t *testing.T, f *fakeLocalSink, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if f.count() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("frame lokal = %d, mau >= %d", f.count(), want)
}

// Jalur tunggal terjaga: UNCONFIRMED yang layak menghasilkan frame normal
// (ADVISORY, no-push) DAN frame lokal — emisi normal tidak berubah, frame
// lokal aditif sesudahnya.
func TestBridgeEmitsNormalAndLocal(t *testing.T) {
	sink := &recSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)

	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, 150))

	if len(sink.msgs) != 1 || sink.msgs[0].Type != dispatch.TypeAdvisory || sink.pushs[0] {
		t.Fatalf("emisi normal berubah: %+v push=%v", sink.msgs, sink.pushs)
	}
	waitForLocal(t, local, 1)
	got := local.first()
	if got == nil || got.Type != dispatch.TypeAlert || !got.TrustedLocal {
		t.Fatalf("frame lokal = %+v", got)
	}
}

// Denyut basi menahan frame lokal TETAPI tidak menyentuh emisi normal:
// heartbeat adalah gerbang kelayakan lokal, bukan bukti — UNCONFIRMED normal
// tetap diproses seperti biasa.
func TestBridgeStaleHeartbeatKeepsNormalEmission(t *testing.T) {
	sink := &recSink{}
	local := &fakeLocalSink{}
	stale := adminEligibleState()
	stale.HeartbeatAge = time.Hour
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: stale, ok: true}, local)

	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, 150))

	if len(sink.msgs) != 1 {
		t.Fatalf("emisi normal tertahan oleh heartbeat basi: %d", len(sink.msgs))
	}
	time.Sleep(50 * time.Millisecond)
	if local.count() != 0 {
		t.Fatal("heartbeat basi membangun frame lokal")
	}
}

// Sumber gagal = gagal-tertutup (tanpa frame lokal, emisi normal utuh), dan
// CONFIRMED tidak pernah menyentuh kait lokal sama sekali.
func TestBridgeLocalFailClosedAndConfirmedUntouched(t *testing.T) {
	src := &fakeAdminSource{state: adminEligibleState(), ok: false}
	sink := &recSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(src, local)

	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, 150))
	if len(sink.msgs) != 1 {
		t.Fatalf("emisi normal = %d, mau 1", len(sink.msgs))
	}
	time.Sleep(50 * time.Millisecond)
	if local.count() != 0 {
		t.Fatal("sumber gagal membangun frame lokal")
	}

	before := src.callCount()
	b.EmitTransition(context.Background(), snapTo(StateConfirmed, false))
	if len(sink.msgs) != 2 {
		t.Fatalf("emisi normal CONFIRMED = %d, mau 2", len(sink.msgs))
	}
	time.Sleep(50 * time.Millisecond)
	if local.count() != 0 {
		t.Fatal("CONFIRMED membangun frame lokal")
	}
	if src.callCount() != before {
		t.Fatal("CONFIRMED memicu pembacaan status operator: pra-filter bocor")
	}
}

// ---- All-clear lokal (D-040) ------------------------------------------------

// Event yang membangunkan audiens 20 km lewat trusted_local lalu RESOLVED TANPA
// pernah CONFIRMED menerima penarikan lokal EVENT_RESOLVED — persis audiens yang
// jalur normal tidak tenangkan (push=EverConfirmed, dan itu false di sini).
func TestBridgeTrustedLocalAllClearOnResolveNeverConfirmed(t *testing.T) {
	sink := &recSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)

	// Alarm lokal saat UNCONFIRMED.
	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, 150))
	waitForLocal(t, local, 1)

	// Event berakhir tanpa pernah CONFIRMED: EverConfirmed=false, event_id sama.
	b.EmitTransition(context.Background(), snapTo(StateResolved, false))
	waitForLocal(t, local, 2)

	got := local.at(1)
	if got == nil || got.Type != dispatch.TypeResolved {
		t.Fatalf("penarikan lokal = %+v, mau EVENT_RESOLVED", got)
	}
	if !got.TrustedLocal {
		t.Error("penarikan lokal TrustedLocal = false, mau true (korelasi audiens)")
	}
	if got.EventID != snapTo(StateResolved, false).EventID {
		t.Errorf("penarikan lokal event_id = %q, mau sama dengan alarm", got.EventID)
	}
	// Emisi normal RESOLVED tidak didorong (EverConfirmed=false): jalur lokallah
	// satu-satunya yang menenangkan audiens ini.
	if len(sink.pushs) < 2 || sink.pushs[1] {
		t.Errorf("push normal RESOLVED = %v, mau elemen kedua false", sink.pushs)
	}
}

// CANCELLED memikul utang yang sama: dua status terminal, satu penarikan lokal.
func TestBridgeTrustedLocalAllClearOnCancelNeverConfirmed(t *testing.T) {
	sink := &recSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)

	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, 150))
	waitForLocal(t, local, 1)
	b.EmitTransition(context.Background(), snapTo(StateCancelled, false))
	waitForLocal(t, local, 2)
	if got := local.at(1); got == nil || got.Type != dispatch.TypeResolved || !got.TrustedLocal {
		t.Fatalf("penarikan lokal CANCELLED = %+v", got)
	}
}

// Event yang PERNAH CONFIRMED tidak menerima penarikan lokal terpisah: jalur
// normal mendorong RESOLVED-nya ke AlertRadiusKm (200 km) yang mencakup 20 km.
// Jejak lokal dilupakan pada CONFIRMED.
func TestBridgeTrustedLocalNoDoubleAllClearWhenConfirmed(t *testing.T) {
	sink := &recSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)

	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, 150))
	waitForLocal(t, local, 1)
	b.EmitTransition(context.Background(), snapTo(StateConfirmed, false))
	b.EmitTransition(context.Background(), snapTo(StateResolved, true))
	time.Sleep(50 * time.Millisecond)
	if local.count() != 1 {
		t.Fatalf("frame lokal = %d, mau 1 (alarm saja; tanpa penarikan lokal ganda)", local.count())
	}
}

// All-clear diutangkan TEPAT kepada yang menerima alarm: event yang tak pernah
// membangunkan audiens lokal (di bawah lantai 60) tidak menghasilkan penarikan
// lokal saat RESOLVED.
func TestBridgeTrustedLocalNoAllClearWhenNeverAlarmed(t *testing.T) {
	sink := &recSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)

	// 30 gal < lantai 60: tidak ada alarm lokal.
	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, 30))
	time.Sleep(50 * time.Millisecond)
	if local.count() != 0 {
		t.Fatalf("alarm lokal di bawah lantai = %d, mau 0", local.count())
	}
	b.EmitTransition(context.Background(), snapTo(StateResolved, false))
	time.Sleep(50 * time.Millisecond)
	if local.count() != 0 {
		t.Fatalf("penarikan lokal tanpa alarm = %d, mau 0", local.count())
	}
}
