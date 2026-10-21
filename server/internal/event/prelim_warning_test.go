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

// --- Penadah peringatan pra-wujud UJI (S4): pengangkatan tanpa kebocoran ---
//
// Seluruh uji di sini menegaskan pagar-pagar pemilik: satu topik uji, NOL
// frame ke sink emisi normal selain perilaku yang sudah ada, NOL trusted_local,
// validity placeholder DPE-6 yang eksplisit, dan disiplin penarikan satu-
// outcome (tidak pernah meniup peringatan yang lebih baru). Jalur alarm FINAL
// asli tetap diuji di emit_test/admin_node_test.

type fakePrelimSink struct {
	mu     sync.Mutex
	frames []*dispatch.AlertMessage
}

func (f *fakePrelimSink) BroadcastProvisionalTestFrame(_ context.Context, msg *dispatch.AlertMessage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.frames = append(f.frames, msg)
}

func (f *fakePrelimSink) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.frames)
}

func (f *fakePrelimSink) at(i int) *dispatch.AlertMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i < 0 || i >= len(f.frames) {
		return nil
	}
	return f.frames[i]
}

// waitForPrelim menantikan frame ke-n tiba (pengiriman dapat datang dari
// goroutine emitTrusted* maupun dari timer kedaluwarsa).
func waitForPrelim(t *testing.T, f *fakePrelimSink, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if f.count() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("frame pra-wujud = %d, mau >= %d", f.count(), want)
}

// newPrelimBridge menyiapkan Bridge lengkap: hook Admin Node layak + penadah
// pra-wujud uji. rec memakai tapSink (advisory D-043 datang dari goroutine) —
// recSink polos akan data-race di -race.
func newPrelimBridge(t *testing.T) (*Bridge, *fakePrelimSink, *tapSink, *fakeLocalSink) {
	t.Helper()
	rec := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(rec)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)
	ps := &fakePrelimSink{}
	b.SetPrelimWarningSink(ps)
	return b, ps, rec, local
}

// Pengangkatan tepi-PRELIM yang layak: TEPAT SATU frame ke topik uji dengan
// is_test=true, trusted_local=false, TypeAlert, validity placeholder DPE-6,
// dan NOL kebocoran ke jalur alarm normal (advisory D-043 tetap satu, frame
// lokal tetap nol, localAlarmed tetap kosong).
func TestPrelimWarningRaisesOnPrelimEdge(t *testing.T) {
	b, ps, rec, local := newPrelimBridge(t)

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForPrelim(t, ps, 1)

	frame := ps.at(0)
	if frame.Type != dispatch.TypeAlert || !frame.IsTest {
		t.Fatalf("frame pengangkatan salah: type=%q is_test=%v", frame.Type, frame.IsTest)
	}
	if frame.TrustedLocal {
		t.Fatal("trusted_local bocor ke peringatan pra-wujud (gerbang 20 km asli terganggu)")
	}
	if frame.ValidityMs != 90_000 {
		t.Fatalf("validity = %d, mau placeholder 90000 (DPE-6)", frame.ValidityMs)
	}
	if frame.EventState != string(StateUnconfirmed) || frame.EventRevision != 3 {
		t.Fatalf("metadata siklus hidup salah: state=%q rev=%d", frame.EventState, frame.EventRevision)
	}

	// Pagar non-publik server: jalur emisi normal TIDAK berubah.
	if rec.count() != 1 || rec.typeOf(0) != dispatch.TypeAdvisory || rec.pushOf(0) {
		t.Fatalf("advisory D-043 berubah: n=%d t=%s", rec.count(), rec.typeOf(0))
	}
	time.Sleep(50 * time.Millisecond)
	if local.count() != 0 {
		t.Fatal("peringatan pra-wujud memancarkan frame trusted_local")
	}
}

// Lantai 60 gal TIDAK direlaksasi di tepi PRELIM: PRELIM di bawah lantai
// tidak pernah diangkat — sementara advisory D-043 tetap teremisi (perilaku
// yang sudah ada tidak digerbangi lantai).
func TestPrelimWarningFloorHeld(t *testing.T) {
	b, ps, rec, _ := newPrelimBridge(t)

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, AdminNodeMinPGAGal-0.01))
	time.Sleep(100 * time.Millisecond)

	if ps.count() != 0 {
		t.Fatalf("lantai ikut direlaksasi oleh penadah: %d frame", ps.count())
	}
	if rec.count() != 1 || rec.typeOf(0) != dispatch.TypeAdvisory {
		t.Fatalf("advisory D-043 berubah: n=%d", rec.count())
	}
}

// Tepat-sekali per episode: tepi berulang tidak menggandakan pengangkatan.
func TestPrelimWarningExactlyOnce(t *testing.T) {
	b, ps, rec, _ := newPrelimBridge(t)

	s := adminSnap(PhasePrelim, 100)
	b.EmitAdminPrelimEdge(context.Background(), s)
	waitForPrelim(t, ps, 1)
	b.EmitAdminPrelimEdge(context.Background(), s)
	time.Sleep(100 * time.Millisecond)

	if ps.count() != 1 {
		t.Fatalf("pengangkatan ganda: %d frame", ps.count())
	}
	if rec.count() > 2 {
		t.Fatalf("advisory D-043 berlebih: %d", rec.count())
	}
}

// UPGRADE pada kaki FINAL yang layak: salinan frame FINAL dengan revision
// tinggi ke topik uji; dispatch trusted_local yang ASLI tetap terjadi; NOL
// penarikan setelahnya.
func TestPrelimWarningUpgradeOnFinalEdge(t *testing.T) {
	b, ps, rec, local := newPrelimBridge(t)

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForPrelim(t, ps, 1)
	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, 100))
	waitForLocal(t, local, 1)
	waitForPrelim(t, ps, 2)

	upgraded := ps.at(1)
	if upgraded.Type != dispatch.TypeAlert || !upgraded.IsTest {
		t.Fatalf("salinan upgrade salah: type=%q is_test=%v", upgraded.Type, upgraded.IsTest)
	}
	for _, f := range []*dispatch.AlertMessage{ps.at(0), upgraded} {
		if f.Type == dispatch.TypeResolved {
			t.Fatal("penarikan ikut terkirim bersama upgrade (XOR dilanggar)")
		}
	}
	if got := local.first(); got == nil || !got.TrustedLocal {
		t.Fatalf("dispatch lokal asli berubah: %+v", got)
	}
	if rec.count() != 2 || rec.typeOf(1) != dispatch.TypeAdvisory {
		// FINAL leg UNCONFIRMED: emisi normalnya juga advisory (frame lokal
		// trusted_local nyata yang jalan di sink lokal, bukan di rec).
		t.Fatalf("emisi normal FINAL berubah: n=%d t1=%s", rec.count(), rec.typeOf(1))
	}

	// Tidak ada penarik telat: timer dihentikan oleh spend kemenangan upgrade.
	time.Sleep(150 * time.Millisecond)
	if ps.count() != 2 {
		t.Fatalf("frame liar setelah upgrade: %d", ps.count())
	}
}

// WITHDRAWAL pada kaki FINAL yang tidak lagi layak (lantai 59): all-clear
// tepat-sekali ke topik uji dengan event_state vocab kontrak "CANCELLED";
// jalur nyata tidak memancarkan frame lokal apa pun.
func TestPrelimWarningWithdrawOnFinalEdge(t *testing.T) {
	b, ps, rec, local := newPrelimBridge(t)

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForPrelim(t, ps, 1)
	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, AdminNodeMinPGAGal-1))
	waitForPrelim(t, ps, 2)

	withdrawn := ps.at(1)
	if withdrawn.Type != dispatch.TypeResolved || !withdrawn.IsTest {
		t.Fatalf("frame penarikan salah: type=%q is_test=%v", withdrawn.Type, withdrawn.IsTest)
	}
	if withdrawn.EventState != "CANCELLED" {
		t.Fatalf("event_state penarikan = %q, mau CANCELLED (vocab kontrak FCM)", withdrawn.EventState)
	}
	// Perilaku nyata: lantai 59 menahan frame lokal (tier FINAL aslinya utuh).
	time.Sleep(50 * time.Millisecond)
	if local.count() != 0 {
		t.Fatalf("FINAL di bawah lantai memancarkan frame lokal nyata: %d", local.count())
	}
	if rec.count() != 2 {
		t.Fatalf("emisi normal berubah: %d", rec.count())
	}
}

// Timer placeholder DPE-6 adalah penarik terakhir: tanpa outcome apa pun,
// pengangkatan menangis menjadi penarikan; tepat-sekali (satu kali saja).
func TestPrelimWarningExpiryTimer(t *testing.T) {
	b, ps, _, _ := newPrelimBridge(t)
	b.prelimWarning.validityMs = 40 // sungguh-sungguh hanya uji: jendela 40 ms

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForPrelim(t, ps, 1)
	waitForPrelim(t, ps, 2)

	expired := ps.at(1)
	if expired.Type != dispatch.TypeResolved || expired.EventState != "CANCELLED" {
		t.Fatalf("frame kedaluwarsa salah: type=%q state=%q", expired.Type, expired.EventState)
	}
	time.Sleep(150 * time.Millisecond)
	// Tepat-sekali: timer yang sudah menangis tidak pernah menangis lagi, dan
	// tidak ada outcome lain yang menyisakan frame keempat.
	if ps.count() != 2 {
		t.Fatalf("penarikan ganda dari timer: %d", ps.count())
	}
}

// DISIPLIN ANTI-CLEAR-BARU (kondisi pemilik): transisi CONFIRMED BISU —
// alarm nyata yang lebih baru sedang hidup, jadi penadah tidak pernah mengirim
// all-clear di sana, dan timer-nya mati sehingga TIDAK ADA penarikan telat.
func TestPrelimWarningConfirmedNeverClears(t *testing.T) {
	b, ps, rec, local := newPrelimBridge(t)
	b.prelimWarning.validityMs = 40

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForPrelim(t, ps, 1)
	b.EmitTransition(context.Background(), snapTo(StateConfirmed, false))

	// Emisi normal CONFIRMED adalah alarm NYATA (sinkron di
	// EmitTransition): push=true ke jalur geo/wilayah seperti biasa.
	if rec.count() != 2 || rec.typeOf(1) != dispatch.TypeAlert || !rec.pushOf(1) {
		t.Fatalf("emisi CONFIRMED normal berubah: n=%d t1=%s", rec.count(), rec.typeOf(1))
	}

	time.Sleep(200 * time.Millisecond) // melampaui jendela timer 40 ms
	if ps.count() != 1 {
		t.Fatalf("penadah meniup alarm CONFIRMED: %d frame pra-wujud", ps.count())
	}
	if local.count() != 0 {
		t.Fatalf("CONFIRMED membangun frame lokal: %d", local.count())
	}
}

// Terminal RESOLVED menutup episode: penarikan dengan event_state nyata.
func TestPrelimWarningTerminalResolvedClears(t *testing.T) {
	b, ps, rec, _ := newPrelimBridge(t)

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForPrelim(t, ps, 1)
	b.EmitTransition(context.Background(), snapTo(StateResolved, false))
	waitForPrelim(t, ps, 2)

	if got := ps.at(1); got.Type != dispatch.TypeResolved || got.EventState != string(StateResolved) {
		t.Fatalf("penarikan terminal salah: type=%q state=%q", got.Type, got.EventState)
	}
	if rec.count() != 2 || rec.typeOf(1) != dispatch.TypeResolved {
		t.Fatalf("emisi normal RESOLVED berubah: n=%d t1=%s", rec.count(), rec.typeOf(1))
	}
}

// XOR satu-outcome, deterministik: penarikan kaki FINAL yang menang membuat
// penutup terminal berikutnya menjadi no-op — tidak ada frame ketiga.
func TestPrelimWarningSingleOutcome(t *testing.T) {
	b, ps, _, _ := newPrelimBridge(t)
	b.prelimWarning.validityMs = 40

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForPrelim(t, ps, 1)
	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, AdminNodeMinPGAGal-1)) // kemenangan penarikan
	waitForPrelim(t, ps, 2)
	b.EmitTransition(context.Background(), snapTo(StateResolved, false)) // terminal: harus no-op
	time.Sleep(200 * time.Millisecond)                                   // melampaui jendela timer juga

	if ps.count() != 2 {
		t.Fatalf("outcome ganda: %d frame", ps.count())
	}
}

// Restart proses: peta pending hilang → TIDAK ada penarikan KECUALI episode
// baru, dan tepi berulang post-restart (revision sama) akan ditanggulkan
// dedup perangkat karena revision tidak pernah mundur.
func TestPrelimWarningRestartLosesPending(t *testing.T) {
	bA, psA, _, _ := newPrelimBridge(t)
	bA.prelimWarning.validityMs = 40
	bA.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForPrelim(t, psA, 1)

	// Proses baru: penadah kosong, tetapi pipeline yang sama terpasang.
	rec2 := &tapSink{}
	local2 := &fakeLocalSink{}
	bB := NewBridge(rec2)
	bB.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local2)
	psB := &fakePrelimSink{}
	bB.SetPrelimWarningSink(psB)

	// Kaki FINAL pada proses baru TANPA pengangkatan: tidak ada frame pra-wujud
	// apa pun (tidak ada yang ditarik, tidak ada yang diangkat ulang).
	bB.EmitTransition(context.Background(), adminSnap(PhaseFinal, 100))
	time.Sleep(150 * time.Millisecond)
	if psB.count() != 0 {
		t.Fatalf("proses baru mereplikasi outcome lama: %d frame", psB.count())
	}

	// Tepi-PRELIM re-fire post-restart = episode BARU (revision sama): frame
	// boleh naik; dedup perangkat (type+event_id, revision <=) yang
	// menanggulkannya — uji itu sudah ada di sisi Android.
	bB.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForPrelim(t, psB, 1)
}

// Mode OFF (setPrelimWarningSink tak pernah dipasang): seluruh jalur berjalan
// persis seperti hari ini — bayangan tetap hidup, D-043 advisory tetap, NOL
// frame pra-wujud; pointer penadah tetap nil.
func TestPrelimWarningModeOffInert(t *testing.T) {
	rec := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(rec)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	if b.prelimWarning != nil {
		t.Fatal("penadah terpasang tanpa hook; harus tetap nil")
	}

	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, 100))
	waitForLocal(t, local, 1)

	// D-043 advisory (satu frame WS) + trusted_local FINAL (satu frame lokal)
	// — perilaku pra-S4 terjaga penuh tanpa presensinya.
	time.Sleep(50 * time.Millisecond)
	if local.count() != 1 || !local.first().TrustedLocal {
		t.Fatalf("jalur lokal D-036 berubah tanpa penadah: %+v", local.first())
	}
}
