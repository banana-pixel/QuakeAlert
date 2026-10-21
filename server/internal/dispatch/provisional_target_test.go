// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package dispatch

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// --- S4 Tambahan: penadahan pra-wujud mode BERTARGET ---
//
// Ketika SetPrelimTestTargetToken mematok SATU token, SEMUA frame penadahan
// (pengangkatan, upgrade FINAL, penarikan, kedaluwarsa — semuanya mengalir
// lewat BroadcastProvisionalTestFrame) menuju token yang sama, EKSKLUSIF:
// tidak ada satu pun pengiriman topik, dan kegagalan kirim tidak pernah
// memicu fallback. Token tidak pernah masuk frame Maupun log.

// prelimMode fungi kecil: frame pra-wujud apa adanya (memakai pola
// provisional_test.go). Dua tipe frame yang mengalir: TypeAlert
// (pengangkatan/upgrade FINAL) dan TypeResolved (penarikan/kedaluwarsa).
func prelimFrame(t *testing.T, typ, evID string) *AlertMessage {
	t.Helper()
	return &AlertMessage{
		Type:          typ,
		EventID:       evID,
		MMI:           "VII",
		LocationName:  "uji",
		Timestamp:     time.Now().UnixMilli(),
		EventState:    "UNCONFIRMED",
		EventRevision: 3,
	}
}

// TestPrelimTargetTokenExclusiveForEveryOutcome: pengangkatan, upgrade FINAL,
// dan penarikan — seluruhnya ke token yang DIPATOK yang sama, NOL topik.
func TestPrelimTargetTokenExclusiveForEveryOutcome(t *testing.T) {
	h := NewHub(testLogger(), func(*http.Request) bool { return true })
	sender := &fakeFCM{}
	d := NewDispatcher(&fakeSaver{}, h, sender, 90*time.Second, testLogger())

	const pinned = "prelim-token-SEKALI-di-boot"
	d.SetPrelimTestTargetToken(pinned)

	frames := []struct {
		typ  string
		evid string
	}{
		{TypeAlert, "E1000000-0000-4000-8000-000000000001"},    // pengangkatan
		{TypeAlert, "E1000000-0000-4000-8000-000000000002"},    // upgrade FINAL (frame keduanya TypeAlert)
		{TypeResolved, "E1000000-0000-4000-8000-000000000001"}, // penarikan
	}
	for _, f := range frames {
		d.BroadcastProvisionalTestFrame(context.Background(), prelimFrame(t, f.typ, f.evid))
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sender.count() >= len(frames) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := sender.count(); got != len(frames) {
		t.Fatalf("pengiriman = %d, mau tepat %d (semua outcome, tanpa ekstra)", got, len(frames))
	}
	sender.mu.Lock()
	defer sender.mu.Unlock()
	for i, sent := range sender.sends {
		if sent.Token != pinned {
			t.Fatalf("send %d: token = %q, mau token DIPATOK yang sama", i, sent.Token)
		}
		if sent.Topic != "" {
			t.Fatalf("send %d: topic = %q — fallback topik MUNCUL (dilarang)", i, sent.Topic)
		}
	}
}

// TestPrelimTargetSendFailureNeverFallsBackToTopic: token UNREGISTERED
// menggagalkan pengiriman dan HARUS tinggal gagal — tanpa upaya topik, tanpa
// pengiriman kedua ke target lain.
func TestPrelimTargetSendFailureNeverFallsBackToTopic(t *testing.T) {
	h := NewHub(testLogger(), func(*http.Request) bool { return true })
	sender := &failingFCM{failMatch: "unreg-prelim"}
	d := NewDispatcher(&fakeSaver{}, h, sender, 90*time.Second, testLogger())

	const pinned = "unreg-prelim-token"
	d.SetPrelimTestTargetToken(pinned)
	d.BroadcastProvisionalTestFrame(context.Background(), prelimFrame(t, TypeAlert, "E2000000-0000-4000-8000-000000000001"))
	time.Sleep(80 * time.Millisecond)

	if got := sender.count(); got != 1 {
		t.Fatalf("pengiriman = %d, mau tepat 1 (gagal-tetap-gagal, tanpa fallback)", got)
	}
	sender.mu.Lock()
	only := sender.sends[0]
	sender.mu.Unlock()
	if only.Token == "" || only.Topic != "" {
		t.Fatalf("target kirim salah: token=%q topic=%q", only.Token, only.Topic)
	}
}

// TestPrelimEmptyTargetTokenRejected: parameter kosong/spasi ditolak —
// salah konfigurasi tidak boleh diam-diam mematikan mode topik atau
// membuka jalur pengiriman tanpa target.
func TestPrelimEmptyTargetTokenRejected(t *testing.T) {
	h := NewHub(testLogger(), func(*http.Request) bool { return true })
	sender := &fakeFCM{}
	d := NewDispatcher(&fakeSaver{}, h, sender, 90*time.Second, testLogger())

	d.SetPrelimTestTargetToken("")
	d.SetPrelimTestTargetToken("   ")
	d.BroadcastProvisionalTestFrame(context.Background(), prelimFrame(t, TypeAlert, "E3000000-0000-4000-8000-000000000001"))

	deadline := time.Now().Add(2 * time.Second)
	var sent *FCMMessage
	for time.Now().Before(deadline) {
		if sender.count() > 0 {
			sent = sender.sends[0]
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if sent == nil {
		t.Fatal("FCM pra-wujud tidak pernah terkirim")
	}
	time.Sleep(100 * time.Millisecond)
	if sender.count() != 1 {
		t.Fatalf("pengiriman = %d, mau tepat 1", sender.count())
	}
	if sent.Topic != TestAlertsTopic || sent.Token != "" {
		t.Fatalf("tanpa pin = mode topik: topic=%q token=%q", sent.Topic, sent.Token)
	}
}

// TestPrelimTargetNeverLoggedOrInFrame: token dipatok tidak boleh mengafuk ke
// frame yang disiarkan ke hub (TargetToken tetap kosong pada objek frame meski
// field-nya pun memang json:"-").
func TestPrelimTargetNeverLoggedOrInFrame(t *testing.T) {
	h := NewHub(testLogger(), func(*http.Request) bool { return true })
	sender := &fakeFCM{}
	d := NewDispatcher(&fakeSaver{}, h, sender, 90*time.Second, testLogger())

	const pinned = "secret-token-must-not-leak"
	d.SetPrelimTestTargetToken(pinned)
	msg := prelimFrame(t, TypeAlert, "E4000000-0000-4000-8000-000000000001")
	d.BroadcastProvisionalTestFrame(context.Background(), msg)

	if msg.TargetToken != "" {
		t.Fatal("frame membawa TargetToken — kebocoran jalur hub berpotensi")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sender.count() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	sender.mu.Lock()
	defer sender.mu.Unlock()
	if len(sender.sends) == 0 {
		t.Fatal("FCM pra-wujud tidak pernah terkirim")
	}
	if sender.sends[0].Token == "" || sender.sends[0].Topic != "" {
		t.Fatalf("target kirim salah: token=%q topic=%q", sender.sends[0].Token, sender.sends[0].Topic)
	}
}
