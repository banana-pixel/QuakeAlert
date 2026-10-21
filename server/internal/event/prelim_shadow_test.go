// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package event

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/banana-pixel/quakealert/server/internal/dispatch"
)

// --- Uji evaluasi murni bayangan ---------------------------------------------

// Bayangan memakai evaluator FINAL yang ASLI; satu-satunya relaksasi adalah
// veto ADMIN_NOT_FINAL milik kaki PRELIM (DPE-1). Lantai 60 gal (D-038) TIDAK
// direlaksasi: PGA di bawah lantai tetap tidak-layak, bahkan di tepi PRELIM.
func TestShadowPrelimVerdictTable(t *testing.T) {
	stale := adminEligibleState()
	stale.HeartbeatAge = time.Hour

	cases := []struct {
		name       string
		state      AdminNodeState
		phase      string
		pga        float64
		wantWould  bool
		wantReason string
	}{
		{"PRELIM di atas lantai: relaksasi veto saja", adminEligibleState(), PhasePrelim, 100, true, ReasonAdminNodeNotFinal},
		{"PRELIM tepat di lantai: masih layak", adminEligibleState(), PhasePrelim, AdminNodeMinPGAGal, true, ReasonAdminNodeNotFinal},
		{"PRELIM di bawah lantai: lantai TIDAK direlaksasi", adminEligibleState(), PhasePrelim, AdminNodeMinPGAGal - 0.01, false, ReasonAdminNodeBelowFloor},
		// Veto ADMIN_NOT_FINAL berada SEBELUM gate heartbeat pada urutan
		// evaluator asli, jadi PRELIM + heartbeat basi tetap dicatat would=true
		// (shadow meniru urutan evaluator apa adanya). Gate heartbeat kemudian
		// menutup episode pada kaki FINAL:
		{"PRELIM dengan heartbeat basi: gate heartbeat tetap memutus", stale, PhasePrelim, 100, false, ReasonAdminNodeHeartbeatStale},
		{"FINAL heartbeat basi: evaluator apa adanya", stale, PhaseFinal, 100, false, ReasonAdminNodeHeartbeatStale},
		{"FINAL di atas lantai: layak apa adanya", adminEligibleState(), PhaseFinal, 100, true, ReasonAdminNodeEligible},
		{"FINAL di bawah lantai: tetap gagal", adminEligibleState(), PhaseFinal, 59, false, ReasonAdminNodeBelowFloor},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			would, reason := shadowPrelimVerdict(c.state, adminSnap(c.phase, c.pga))
			if would != c.wantWould || reason != c.wantReason {
				t.Fatalf("verdict = (%v, %s), mau (%v, %s)", would, reason, c.wantWould, c.wantReason)
			}
		})
	}

	// Tanpa node designated: kegagalan tertutup sebelum evaluasi apa pun.
	would, reason := shadowPrelimVerdict(AdminNodeState{}, adminSnap(PhasePrelim, 100))
	if would || reason != ReasonAdminNodeNoneDesignated {
		t.Fatalf("tanpa designate = (%v, %s), mau (false, NO_ADMIN_DESIGNATED)", would, reason)
	}
}

// --- Perekam log bayangan ----------------------------------------------------

// syncBuf melindungi buffer log dari goroutine bayangan: logger menulis dari
// goroutine emitTrusted* sementara uji membaca di goroutine utama (-race).
type syncBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// tapSink adalah perekam emisi normal yang aman goroutine: advisory D-043
// (EmitAdminPrelimEdge) berjalan di goroutine emitTrustedPrelim, sedangkan
// EmitTransition menulis sinkron — recSink polos akan data-race di -race.
type tapSink struct {
	mu    sync.Mutex
	msgs  []*dispatch.AlertMessage
	pushs []bool
}

func (s *tapSink) DispatchEventFrame(_ context.Context, msg *dispatch.AlertMessage, push bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, msg)
	s.pushs = append(s.pushs, push)
}

func (s *tapSink) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.msgs)
}

func (s *tapSink) typeOf(i int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i < 0 || i >= len(s.msgs) {
		return ""
	}
	return s.msgs[i].Type
}

func (s *tapSink) pushOf(i int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pushs[i]
}

func shadowLogBuffer(b *Bridge) *syncBuf {
	buf := &syncBuf{}
	b.prelimShadow.log = slog.New(slog.NewJSONHandler(buf, nil))
	return buf
}

func shadowRecordCount(buf *syncBuf) int {
	return strings.Count(buf.String(), `"msg":"PRELIM_SHADOW"`)
}

func waitForShadow(t *testing.T, b *Bridge, eventID string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := b.prelimShadow.recorded.Load(eventID); ok {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("catatan bayangan %s tidak pernah muncul", eventID)
}

// --- Tepi PRELIM: satu catatan, NOL emisi ------------------------------------

// Tepi PRELIM yang layak secara bayangan memancarkan PERSIS seperti sebelum
// ada bayangan: satu advisory D-043 (TypeAdvisory, no-push), nol frame
// lokal, localAlarmed kosong — dan TEPAT SATU catatan PRELIM_SHADOW
// berlabel shadow/simulated dengan would_dispatch=true.
func TestShadowPrelimEdgeLogsAndEmitsNothing(t *testing.T) {
	sink := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)
	buf := shadowLogBuffer(b)

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForShadow(t, b, adminSnap(PhasePrelim, 100).EventID)

	if sink.count() != 1 || sink.typeOf(0) != dispatch.TypeAdvisory || sink.pushOf(0) {
		t.Fatalf("advisory D-043 berubah oleh bayangan: %d", sink.count())
	}
	time.Sleep(50 * time.Millisecond)
	if local.count() != 0 {
		t.Fatal("bayangan membangun frame lokal")
	}
	b.localAlarmed.Range(func(key, _ any) bool {
		t.Fatalf("bayangan menandai localAlarmed: %v", key)
		return false
	})
	if shadowRecordCount(buf) != 1 {
		t.Fatalf("catatan bayangan = %d, mau 1:\n%s", shadowRecordCount(buf), buf.String())
	}
	for _, want := range []string{`"shadow":true`, `"simulated":true`, `"would_dispatch":true`} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("catatan kehilangan label %s:\n%s", want, buf.String())
		}
	}
}

// Tepat-sekali per episode: edge PRELIM berulang pada event yang sama tetap
// satu catatan — tetapi sumber status tetap dibaca dua kali (bayangan tidak
// mengubah perilaku kait yang ada).
func TestShadowExactlyOncePerEvent(t *testing.T) {
	sink := &tapSink{}
	local := &fakeLocalSink{}
	src := &fakeAdminSource{state: adminEligibleState(), ok: true}
	b := NewBridge(sink)
	b.SetAdminNodeHook(src, local)
	buf := shadowLogBuffer(b)

	s := adminSnap(PhasePrelim, 100)
	b.EmitAdminPrelimEdge(context.Background(), s)
	waitForShadow(t, b, s.EventID)
	b.EmitAdminPrelimEdge(context.Background(), s)
	time.Sleep(50 * time.Millisecond)

	if n := shadowRecordCount(buf); n != 1 {
		t.Fatalf("catatan bayangan = %d, mau 1 (tepat-sekali)", n)
	}
	if src.callCount() != 2 {
		t.Fatalf("panggilan sumber = %d, mau 2 (perilaku kait tak berubah)", src.callCount())
	}
}

// --- Kaki FINAL: hasil DPE-4 ---------------------------------------------------

// FINAL yang layak: catatan bayangan menghasilkan outcome absorb, entri
// dihapus, dan — terpenting — dispatch TRUSTED_LOCAL yang ASLI tetap terjadi
// persis seperti tanpa bayangan (perilaku produksi tidak tersentuh).
func TestShadowFinalEdgeWouldAbsorb(t *testing.T) {
	sink := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)
	buf := shadowLogBuffer(b)

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForShadow(t, b, adminSnap(PhasePrelim, 100).EventID)
	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, 100))
	waitForLocal(t, local, 1)

	if got := local.first(); got == nil || got.Type != dispatch.TypeAlert || !got.TrustedLocal {
		t.Fatalf("dispatch lokal asli berubah oleh bayangan: %+v", got)
	}
	if _, ok := b.localAlarmed.Load(local.first().EventID); !ok {
		t.Fatal("localAlarmed asli tidak tercatat — perilaku D-040 berubah")
	}
	line := buf.String()
	if !strings.Contains(line, `"would_outcome":"absorb"`) {
		t.Fatalf("outcome absorb hilang:\n%s", line)
	}
	if _, ok := b.prelimShadow.recorded.Load(local.first().EventID); ok {
		t.Fatal("entri bayangan tidak dihapus pada outcome FINAL")
	}
}

// FINAL yang tidak layak (di bawah lantai): outcome withdraw dicatat dan
// TIDAK ada frame lokal nyata — penarikan bayangan murni catatan.
func TestShadowFinalEdgeWouldWithdraw(t *testing.T) {
	sink := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)
	buf := shadowLogBuffer(b)

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForShadow(t, b, adminSnap(PhasePrelim, 100).EventID)
	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, AdminNodeMinPGAGal-1))

	time.Sleep(50 * time.Millisecond)
	if local.count() != 0 {
		t.Fatal("FINAL di bawah lantai membangun frame lokal nyata")
	}
	if !strings.Contains(buf.String(), `"would_outcome":"withdraw"`) {
		t.Fatalf("outcome withdraw hilang:\n%s", buf.String())
	}
	if _, ok := b.prelimShadow.recorded.Load(adminSnap(PhasePrelim, 100).EventID); ok {
		t.Fatal("entri bayangan tidak dihapus pada outcome FINAL")
	}
}

// Episode would=NO: tidak ada provisional yang pernah ada, jadi outcome
// FINAL/terminal tidak boleh menghasilkan baris withdraw — entri habis diam.
func TestShadowWouldNoNeverWithdraws(t *testing.T) {
	sink := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)
	buf := shadowLogBuffer(b)

	// Tepi PRELIM di bawah lantai: catatan would=false (bukan disingkirkan —
	// hasil risetnya valid: alarm hipotetis tidak akan jalan).
	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, AdminNodeMinPGAGal-1))
	waitForShadow(t, b, adminSnap(PhasePrelim, AdminNodeMinPGAGal-1).EventID)
	if shadowRecordCount(buf) != 1 || strings.Contains(buf.String(), `"would_dispatch":true`) {
		t.Fatalf("catatan would=no salah:\n%s", buf.String())
	}

	// Edge PRELIM berulang bukan outcome DPE-4.
	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, AdminNodeMinPGAGal-1))
	time.Sleep(50 * time.Millisecond)
	// Terminal: entri would=no dihapus tanpa baris withdraw.
	b.EmitTransition(context.Background(), snapTo(StateResolved, false))

	time.Sleep(50 * time.Millisecond)
	if n := shadowRecordCount(buf); n != 1 {
		t.Fatalf("catatan would=no menghasilkan outcome liar: %d:\n%s", n, buf.String())
	}
	if _, ok := b.prelimShadow.recorded.Load(snapTo(StateResolved, false).EventID); ok {
		t.Fatal("entri would=no tidak dihapus pada terminal")
	}
}

// --- Penutup episode tanpa kaki FINAL -----------------------------------------

// Event would-dispatch yang berakhir tanpa kaki FINAL sempat menilai:
// terminal RESOLVED berutang withdraw (DPE-4 baris resolved/expired).
func TestShadowTerminalWithdraw(t *testing.T) {
	sink := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)
	buf := shadowLogBuffer(b)

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForShadow(t, b, adminSnap(PhasePrelim, 100).EventID)
	b.EmitTransition(context.Background(), snapTo(StateResolved, false))

	time.Sleep(50 * time.Millisecond)
	if !strings.Contains(buf.String(), `"would_outcome":"withdraw"`) {
		t.Fatalf("terminal tanpa FINAL tidak mencatat withdraw:\n%s", buf.String())
	}
	if _, ok := b.prelimShadow.recorded.Load(snapTo(StateResolved, false).EventID); ok {
		t.Fatal("entri bayangan tidak dihapus pada terminal")
	}
}

// Escalasi CONFIRMED sebelum kaki FINAL: alarm nyata jalur normal yang
// menutup episode — bayangan mencatat outcome confirmed dan menghapus entri.
func TestShadowTerminalConfirmed(t *testing.T) {
	sink := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(sink)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)
	buf := shadowLogBuffer(b)

	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForShadow(t, b, adminSnap(PhasePrelim, 100).EventID)
	b.EmitTransition(context.Background(), snapTo(StateConfirmed, false))

	time.Sleep(50 * time.Millisecond)
	if !strings.Contains(buf.String(), `"would_outcome":"confirmed"`) {
		t.Fatalf("CONFIRMED tidak mencatat outcome confirmed:\n%s", buf.String())
	}
	if _, ok := b.prelimShadow.recorded.Load(snapTo(StateConfirmed, false).EventID); ok {
		t.Fatal("entri bayangan tidak dihapus pada CONFIRMED")
	}
	if src := b.adminSource.(*fakeAdminSource); src.callCount() < 1 {
		t.Fatal("sanity: tepi PRELIM harusnya membaca status operator")
	}
}

// --- Invarian inti S1 ----------------------------------------------------------

// Bayangan TIDAK PERNAH boleh menyentuh kelayakan alarm asli: dengan hook
// yang menolak semuanya (state tak-verifikasi), satu episode bayangan
// lengkap — tepi PRELIM, kaki FINAL, terminal — tetap menghasilkan NOL frame
// lokal, NOL jejak localAlarmed, dan sumber status dibaca tepat untuk
// kait yang sudah ada (bayangan tidak menambah pembacaan).
func TestShadowNeverAltersAlarmPath(t *testing.T) {
	sink := &tapSink{}
	local := &fakeLocalSink{}
	reject := adminEligibleState()
	reject.Verified = false
	src := &fakeAdminSource{state: reject, ok: true}
	b := NewBridge(sink)
	b.SetAdminNodeHook(src, local)
	buf := shadowLogBuffer(b)

	// Tepi PRELIM: seluruh gerbang gagal (UNVERIFIED) — designated masih true,
	// jadi catatan would=false (UNVERIFIED) muncul untuk riset.
	b.EmitAdminPrelimEdge(context.Background(), adminSnap(PhasePrelim, 100))
	waitForShadow(t, b, adminSnap(PhasePrelim, 100).EventID)

	// Kaki FINAL + terminal: kelayakan lokal gagal semua, bayangan dilarang
	// membantu. Setelah 50 ms, tidak ada jejak alarm lokal sama sekali.
	b.EmitTransition(context.Background(), adminSnap(PhaseFinal, 100))
	b.EmitTransition(context.Background(), snapTo(StateResolved, false))
	time.Sleep(50 * time.Millisecond)

	if local.count() != 0 {
		t.Fatalf("bayangan membangun frame lokal: %d", local.count())
	}
	b.localAlarmed.Range(func(key, _ any) bool {
		t.Fatalf("localAlarmed ternoda: %v", key)
		return false
	})
	if strings.Contains(buf.String(), `"would_dispatch":true`) {
		t.Fatalf("state tak-verifikasi direlaksasi oleh bayangan:\n%s", buf.String())
	}
}
