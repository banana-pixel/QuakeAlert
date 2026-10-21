// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package dispatch

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// targetDrillMsg menyusun pesan drill latihan yang punya bentuk sama dengan
// hasil HandleCreateTestAlert: is_test, event_id berawalan test-, validitas
// penuh resolusi drill. token hanya diisi pada drill bertarget.
func targetDrillMsg(token string) *AlertMessage {
	return &AlertMessage{
		Type:           TypeAlert,
		EventID:        "test-" + "aabbccdd-0000-0000-0000-000000000000",
		MMI:            "VII",
		IntensityLabel: "strong",
		PGAGal:         300,
		CentroidLat:    -6.9175,
		CentroidLon:    107.6191,
		LocationName:   "LATIHAN — bukan gempa sungguhan",
		Timestamp:      time.Now().UnixMilli(),
		NodeCount:      3,
		TargetToken:    token,
	}
}

// waitForSends yang sudah ada di broadcast_test.go dipakai kembali — asinkron
// pengiriman drill mengikuti pola fanout broadcast.

// Drill bertarget HARUS eksklusif satu token: tanpa topic, tanpa fallback
// broadcast ke jalur alert nyata, dan tanpa menulis satu baris pun ke
// earthquake_events. All-clear 20 dtknya menyambut TOKEN yang sama, bukan
// topic — stand-down perangkat yang diuji, bukan stand-down nasional.
func TestDrillTargetExcludesTopicAndEventWrites(t *testing.T) {
	savedResolveAfter := testAlertResolveAfter
	testAlertResolveAfter = 30 * time.Millisecond
	defer func() { testAlertResolveAfter = savedResolveAfter }()
	// Assertion terhadap penambalahan var di atas: produksi boleh mengandalkan
	// 20 s… tapi tepat dalam test ini pemangkasnya tidak mengubah relevansi.
	if savedResolveAfter != 20*time.Second {
		t.Fatalf("testAlertResolveAfter harus tetap 20 s di init: %v", savedResolveAfter)
	}

	saver := &fakeSaver{}
	h := NewHub(testLogger(), func(*http.Request) bool { return true })
	fcm := &fakeFCM{}
	d := NewDispatcher(saver, h, fcm, time.Minute, testLogger())

	d.DispatchTestAlert(targetDrillMsg("uji-sintetis-token-ABCDEFGHIJKLMNOPQRSTUVWXYZ"))

	waitForSends(t, fcm, 1)
	// tunggu all-clear drill (testAlertResolveAfter yang telah dipangkas)
	waitForSends(t, fcm, 2)
	// TANPA pengiriman ketiga: tidak ada fallback topic setelah sukses token
	time.Sleep(60 * time.Millisecond)
	tokens, topics := fcm.targets()
	if len(tokens) != 2 || len(topics) != 0 {
		t.Fatalf("tokens=%q topics=%q mau: 2 token eksklusif, 0 topic", tokens, topics)
	}
	for _, tok := range tokens {
		if tok != "uji-sintetis-token-ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			t.Fatalf("token asing terkirim: %q", tok)
		}
	}
	// nol penulisan event: jalur drill tidak pernah menyentuh eventSaver.
	saved, resolved := saver.snapshot()
	if len(saved) != 0 || len(resolved) != 0 {
		t.Fatalf("drill menyambar basis data: saved=%d resolved=%d", len(saved), len(resolved))
	}
	// payload: is_test=true + validitas drill
	fcm.mu.Lock()
	first := fcm.sends[0]
	fcm.mu.Unlock()
	if first.Data["is_test"] != "true" {
		t.Fatalf("is_test = %q, mau \"true\"", first.Data["is_test"])
	}
	if first.Data["validity_ms"] == "" {
		t.Fatal("validity_ms absen — validitas terbatas harus dinyatakan")
	}
	if !strings.HasPrefix(first.Data["event_id"], "test-") {
		t.Fatalf("event_id = %q, mau berawalan test-", first.Data["event_id"])
	}
}

// Pengiriman gagal ke token TIDAK BOLEH memicu fallback topic mana pun:
// yang diuji adalah perangkat itu; sebuah UNREGISTERED tidak boleh diam-diam
// berubah menjadi broadcast.
func TestDrillTargetFailureNeverFallsBackToTopic(t *testing.T) {
	// failMatch menyasar token target: pengiriman pertama gagal (UNREGISTERED)
	// lalu tidak ada yang kedua.
	fcm := &failingFCM{failMatch: "uji-sintetis"}
	h := NewHub(testLogger(), func(*http.Request) bool { return true })
	d := NewDispatcher(&fakeSaver{}, h, fcm, time.Minute, testLogger())

	d.dispatchTestFCM(targetDrillMsg("uji-sintetis-token-ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
	// beri waktu goroutine pengiriman berjalan dan gagal
	time.Sleep(80 * time.Millisecond)
	if fcm.count() != 1 {
		t.Fatalf("pengiriman = %d, mau tepat 1 (tanpa fallback topic)", fcm.count())
	}
	fcm.mu.Lock()
	only := fcm.sends[0]
	fcm.mu.Unlock()
	if only.Token == "" || only.Topic != "" {
		t.Fatalf("satusnya token: %+v — tidak boleh topic", only)
	}
}

// TargetToken adalah routing internal: frame WebSocket yang diterima klien
// HARUS tetap tidak membawanya — json:"-" pada field.
func TestDrillTargetHiddenFromWebSocketFrame(t *testing.T) {
	h := NewHub(testLogger(), func(*http.Request) bool { return true })
	fcm := &fakeFCM{}
	d := NewDispatcher(&fakeSaver{}, h, fcm, time.Minute, testLogger())

	client := registerClient(h)
	d.DispatchTestAlert(targetDrillMsg("uji-sintetis-token-ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
	waitForSends(t, fcm, 1)

	frame := readMessage(t, client.send)
	var wire map[string]any
	if err := json.Unmarshal(frame, &wire); err != nil {
		t.Fatal(err)
	}
	raw := string(frame)
	if strings.Contains(raw, "uji-sintetis") || strings.Contains(raw, "TargetToken") {
		t.Fatalf("token routing bocor ke wire: %s", raw)
	}
	// yang tetap harus ada: envelope is_test latihan
	if wire["is_test"] != true {
		t.Fatalf("is_test = %v, mau true", wire["is_test"])
	}
}

// Drill tanpa target mempertahankan perilaku lama: topic test_alerts, tanpa
// token bagi siapa pun.
func TestDrillWithoutTargetKeepsTopic(t *testing.T) {
	h := NewHub(testLogger(), func(*http.Request) bool { return true })
	fcm := &fakeFCM{}
	d := NewDispatcher(&fakeSaver{}, h, fcm, time.Minute, testLogger())

	d.DispatchTestAlert(targetDrillMsg(""))
	waitForSends(t, fcm, 1)
	tokens, topics := fcm.targets()
	if len(topics) != 1 || topics[0] != TestAlertsTopic || len(tokens) != 0 {
		t.Fatalf("tokens=%q topics=%q — drill biasa harus topic-only", tokens, topics)
	}
}
