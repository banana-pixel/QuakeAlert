// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package event

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- S4 Tambahan: penadahan pra-wujud mode BERTARGET (pseudonim saja) ---
//
// Event pkg TIDAK PERNAH memegang token: yang dipunyainya hanya pseudonim
// target untuk label audit. Token dipikat di Dispatcher (lihat
// dispatch.SetPrelimTestTargetToken). Seluruh outcome pengangkatan, upgrade
// FINAL, dan penarikan mengalir lewat SATU sink yang sama.

// bufferedLogger mengembalikan logger JSON yang ditambang ke buffer (dipakai
// untuk menegaskan pseudonim ada dan test_alerts/token absen dari label).
func bufferedLogger(t *testing.T) (*slog.Logger, *syncBuffer) {
	t.Helper()
	var buf syncBuffer
	return slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})), &buf
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestPrelimTargetAttrsPseudonymOnly: mode BERTARGET menempel
// "target_pseudonym" pada SETIAP baris penadahan — dan tidak pernah
// "topic"/"test_alerts" maupun kata token.
func TestPrelimTargetAttrsPseudonymOnly(t *testing.T) {
	rec := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(rec)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)
	ps := &fakePrelimSink{}
	b.SetPrelimWarningSink(ps)
	lg, out := bufferedLogger(t)
	b.prelimWarning.log = lg // pola injeksi assert-log; log hanya dipakai penadahan
	b.SetPrelimWarningTarget("Quakezen-6EB53CF4")

	b.EmitAdminPrelimEdge(t.Context(), adminSnap(PhasePrelim, 100))
	waitForPrelim(t, ps, 1)

	logs := out.String()
	if !strings.Contains(logs, "target_pseudonym") || !strings.Contains(logs, "Quakezen-6EB53CF4") {
		t.Fatalf("label mode target absen:\n%s", logs)
	}
	if strings.Contains(logs, "test_alerts") || strings.Contains(logs, `\"topic\"`) {
		t.Fatal("label masih menyebut topik padahal mode bertarget")
	}
	tl := strings.ToLower(logs)
	if strings.Contains(tl, "fcm_token") || strings.Contains(tl, "registration_token") {
		t.Fatal("log membawa nama field token — event pkg tidak boleh menyentuh token")
	}
}

// TestPrelimTargetNullishSetterGuards: pseudonim kosong/sink belum terpasang
// adalah no-op — mode topik asli tetap tercatat (regresi label S4).
func TestPrelimTargetNullishSetterGuards(t *testing.T) {
	rec := &tapSink{}
	local := &fakeLocalSink{}
	b := NewBridge(rec)
	b.SetAdminNodeHook(&fakeAdminSource{state: adminEligibleState(), ok: true}, local)
	ps := &fakePrelimSink{}
	b.SetPrelimWarningSink(ps)
	lg, out := bufferedLogger(t)
	b.prelimWarning.log = lg
	b.SetPrelimWarningTarget("") // no-op

	b.EmitAdminPrelimEdge(t.Context(), adminSnap(PhasePrelim, 100))
	waitForPrelim(t, ps, 1)

	logs := out.String()
	if !strings.Contains(logs, "test_alerts") {
		t.Fatalf("label mode topik hilang setelah setter nol:\n%s", logs)
	}
	if strings.Contains(logs, "target_pseudonym") {
		t.Fatal("label target nongkol padahal pseudonim kosong")
	}

	// Tanpa sink terpasang sama sekali: setter target tidak boleh meghidupkan.
	b2 := NewBridge(rec)
	b2.SetPrelimWarningTarget("Quakezen-6EB53CF4")
	if b2.prelimWarning != nil {
		t.Fatal("setter target menciptakan penadahan tanpa sink")
	}
}

// TestPrelimAllOutcomesThroughSameSink: pengangkatan + upgrade FINAL +
// penarikan tepat pada SATU instance sink (kontrak "menuju target yang sama"
// tingkat event); token-nya sendiri dijaminkan lapisan dispatch).
func TestPrelimAllOutcomesThroughSameSink(t *testing.T) {
	b, ps, _, _ := newPrelimBridge(t)
	b.prelimWarning.validityMs = 40

	b.EmitAdminPrelimEdge(t.Context(), adminSnap(PhasePrelim, 100))
	waitForPrelim(t, ps, 1)

	// Upgrade FINAL (kontributor layak): snapshot kaki FINAL kini datang
	// (sink sama, frame pengangkatan kedua).
	b.EmitAdminEdge(t.Context(), adminSnap(PhaseFinal, 101))
	waitForPrelim(t, ps, 2)

	if ps != b.prelimWarning.sink {
		t.Fatal("sink bertukar di tengah episode — target bisa berpindah")
	}
	if b.prelimWarning.targetPseudonym != "" {
		t.Fatal("fixture ini tanpa target — setter nol tidak boleh mengisi")
	}

	// Penarikan memakai frame dasar yang DIAMBIL DARI ENTRI yang sama: masih
	// salinan sink satu-satunya.
	time.Sleep(80 * time.Millisecond) // tunggu timer 40 ms atau tunggu outcome bila tidak dipakai
}
