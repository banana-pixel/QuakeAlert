// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package event

import (
	"context"
	"strings"
	"testing"
	"time"
)

// --- Bench pra-wujud uji (S4 Tambahan, Opsi B): tepi sintetis yang hemat ---
//
// Uji di sini menegaskan lima pagar bench (prelim_bench.go): TIDAK ADA frame
// selain penadahan uji (advisory D-043, trusted_local, sink normal semua
// senyap), gate Admin Node asli tetap menilai (pga < 60 ditolak), mode OFF
// tidak melakukan I/O apa pun, gagal-tertutup admin menjawab inactive, dan
// all-clear placeholder masih bekerja lewat timer yang sama.

// benchIn adalah masukan bench layak (pga 300 → gerbang lolos).
func benchIn(t *testing.T, eventID string, pga float64) PrelimTestInput {
	t.Helper()
	return PrelimTestInput{
		EventID:     eventID,
		PGAGal:      pga,
		CentroidLat: -6.9175,
		CentroidLon: 107.6191,
		Timestamp:   time.Now().UnixMilli(),
	}
}

// TestPrelimBenchRaisesThroughWarnOnly: bench berpengangkat lewat SATU frame
// is_test ke penadahan; sink advisory/coretan lain TIDAK menerima apa pun.
func TestPrelimBenchRaisesThroughWarnOnly(t *testing.T) {
	rec := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(rec)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)
	ps := &fakePrelimSink{}
	b.SetPrelimWarningSink(ps)

	outcome, reason := b.PrelimTestBench(context.Background(), benchIn(t, "test-bench-01", 300))
	// Relaksasi veto: pengangkatan kaki PRELIM membawa alasan penolakan FINAL
	// yang DIRELAKSASIKAN — label gate yang sama persis dengan jalur prod
	// (prelim_shadow.go), bukan label khusus bench.
	if outcome != PrelimBenchRaised || reason != ReasonAdminNodeNotFinal {
		t.Fatalf("outcome = %q reason = %q, mau raised/ADMIN_NOT_FINAL (relaksasi)", outcome, reason)
	}
	waitForPrelim(t, ps, 1)

	frame := ps.at(0)
	if !strings.HasPrefix(frame.EventID, "test-") {
		t.Fatalf("event_id = %q, mau berawalan test-", frame.EventID)
	}
	if frame.Type != "EARTHQUAKE_ALERT" || !frame.IsTest || frame.TrustedLocal {
		t.Fatalf("frame bench salah: type=%q is_test=%v trusted_local=%v",
			frame.Type, frame.IsTest, frame.TrustedLocal)
	}
	if frame.ValidityMs != 90_000 {
		t.Fatalf("validity = %d, mau placeholder 90000", frame.ValidityMs)
	}
	if frame.EventState != string(StateUnconfirmed) {
		t.Fatalf("event_state = %q, mau UNCONFIRMED", frame.EventState)
	}
	// Kontributor yang dinisiasi bench =	node Admin asli, fase PRELIM.
	if c := b.prelimWarning; c == nil {
		t.Fatal("penadahan hilang setelah bench")
	}

	// NOL bocor ke jalur emisi normal: advisory D-043, trusted_local, sensor
	// sink lain — semua senyap (itu inti batasan pemilik).
	time.Sleep(50 * time.Millisecond)
	if rec.count() != 0 {
		t.Fatalf("sink emisi normal menerima %d frame — bench membocorkan jalur", rec.count())
	}
	if local.count() != 0 {
		t.Fatal("bench memancarkan frame trusted_local")
	}
}

// TestPrelimBenchRefusesBelowFloor: gate nyata menilai bench — PGA di bawah
// lantai 60 gal ditolak dan TIDAK ada frame (ini pengamatan, bukan galat).
func TestPrelimBenchRefusesBelowFloor(t *testing.T) {
	b, ps, rec, local := newPrelimBridge(t)

	outcome, reason := b.PrelimTestBench(context.Background(), benchIn(t, "test-bench-02", 40))
	if outcome != PrelimBenchRefused {
		t.Fatalf("outcome = %q, mau refused", outcome)
	}
	if reason != ReasonAdminNodeBelowFloor {
		t.Fatalf("reason = %q, mau BELOW_FLOOR", reason)
	}
	time.Sleep(50 * time.Millisecond)
	if ps.count() != 0 || rec.count() != 0 || local.count() != 0 {
		t.Fatal("penolakan gate tetap mengirim frame — lingkar gate berlubang")
	}
}

// TestPrelimBenchInactiveWhenModeOff: mode OFF menjawab inactive SEBELUM
// adminSource/persist ke mana pun disentuh — nol I/O, nol kirim.
func TestPrelimBenchInactiveWhenModeOff(t *testing.T) {
	rec := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(rec)
	// tanpa SetPrelimWarningSink → prelimWarning nil = mode OFF.
	src := &countingAdminSource{inner: &fakeAdminSource{state: adminEligibleState(), ok: true}}
	b.SetAdminNodeHook(src, local)

	outcome, _ := b.PrelimTestBench(context.Background(), benchIn(t, "test-bench-03", 300))
	if outcome != PrelimBenchInactive {
		t.Fatalf("outcome = %q, mau inactive", outcome)
	}
	if src.called {
		t.Fatal("mode OFF masih membaca status Admin (I/O tak perlu)")
	}
	time.Sleep(50 * time.Millisecond)
	if rec.count() != 0 || local.count() != 0 {
		t.Fatal("mode OFF mengirim frame")
	}
}

// TestPrelimBenchInactiveOnAdminFailClosed: status admin gagal → failed-closed
// (tidak "mencoba jalan lain").
func TestPrelimBenchInactiveOnAdminFailClosed(t *testing.T) {
	rec := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(rec)
	// kait Admin ada, tetapi membaca gagal (ok:false)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: false}, local)
	ps := &fakePrelimSink{}
	b.SetPrelimWarningSink(ps)

	outcome, _ := b.PrelimTestBench(context.Background(), benchIn(t, "test-bench-04", 300))
	if outcome != PrelimBenchInactive {
		t.Fatalf("outcome = %q, mau inactive", outcome)
	}
	time.Sleep(50 * time.Millisecond)
	if ps.count() != 0 || rec.count() != 0 || local.count() != 0 {
		t.Fatal("fail-closed masih mengirim")
	}
}

// TestPrelimBenchSameSinkCancelsAtExpiry: pengangkatan bench berutang
// penarikan placeholder yang kembali ke sink YANG SAMA — event_state
// CANCELLED, revision pada pengangkatan.
func TestPrelimBenchSameSinkCancelsAtExpiry(t *testing.T) {
	b, ps, _, _ := newPrelimBridge(t)
	b.prelimWarning.validityMs = 40

	outcome, reason := b.PrelimTestBench(context.Background(), benchIn(t, "test-bench-05", 300))
	if outcome != PrelimBenchRaised || reason != ReasonAdminNodeNotFinal {
		t.Fatalf("outcome = %q reason = %q, mau raised (relaksasi)", outcome, reason)
	}
	waitForPrelim(t, ps, 1) // pengangkatan

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ps.count() >= 2 { // count() ber-mutex — akses langsung ps.frames memicu race
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel := ps.at(1)
	if cancel == nil {
		t.Fatal("penarikan placeholder tidak tiba")
	}
	if cancel.Type != "EVENT_RESOLVED" || cancel.EventState != "CANCELLED" {
		t.Fatalf("penarikan salah: type=%q state=%q", cancel.Type, cancel.EventState)
	}
	if cancel.EventID != "test-bench-05" {
		t.Fatalf("penarikan event_id = %q, mau milik episode yang sama", cancel.EventID)
	}
	if !cancel.IsTest {
		t.Fatal("penarikan bukan is_test — kontrak frame uji berubah diam-diam")
	}
}

// countingAdminSource membungkus AdminNodeSource dan mencatat pemanggilan
// — dipakai menegaskan mode OFF tidak melakukan I/O apa pun.
type countingAdminSource struct {
	inner  AdminNodeSource
	called bool
	mu     countingMu
}

// countingMu diabaikan: pemanggilan bench adalah synchron satu goroutine uji;
// pembeda di sini hanya nama agar tidak bentrok dengan sync.Mutex implisit.
type countingMu struct{}

func (c *countingAdminSource) GetAdminNodeState(ctx context.Context) (AdminNodeState, bool) {
	c.called = true
	return c.inner.GetAdminNodeState(ctx)
}
