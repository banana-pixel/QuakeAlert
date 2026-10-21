// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package dispatch

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// --- Peringatan pra-wujud UJI: satu topik, tanpa fallback, is_test dijamin ---

// TestProvisionalFrameTopicOnly menegaskan pagar-pagar permanen
// BroadcastProvisionalTestFrame: hanya TestAlertsTopic, tanpa target token,
// tanpa fallback GeoTopic (nol pengiriman kedua), IsTest dipaksa true, dan
// validity selalu dinyatakan (DPE-6 placeholder bila pengirim lupa).
func TestProvisionalFrameTopicOnly(t *testing.T) {
	h := NewHub(testLogger(), func(*http.Request) bool { return true })
	sender := &fakeFCM{}
	d := NewDispatcher(&fakeSaver{}, h, sender, 90*time.Second, testLogger())

	msg := &AlertMessage{
		Type:          TypeAlert,
		EventID:       "E0000000-0000-4000-8000-000000000000",
		MMI:           "V",
		LocationName:  "uji",
		Timestamp:     time.Now().UnixMilli(),
		EventState:    "UNCONFIRMED",
		EventRevision: 3,
	}
	d.BroadcastProvisionalTestFrame(context.Background(), msg)

	deadline := time.Now().Add(2 * time.Second)
	var sent *FCMMessage
	for time.Now().Before(deadline) {
		sender.mu.Lock()
		if len(sender.sends) > 0 {
			sent = sender.sends[len(sender.sends)-1]
		}
		sender.mu.Unlock()
		if sent != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if sent == nil {
		t.Fatal("FCM pra-wujud tidak pernah terkirim")
	}
	time.Sleep(100 * time.Millisecond)

	if sent.Topic != TestAlertsTopic || sent.Token != "" || sent.Condition != "" {
		t.Fatalf("topik salah: topic=%q token=%q cond=%q", sent.Topic, sent.Token, sent.Condition)
	}
	if sent.Priority != "HIGH" {
		t.Fatalf("prioritas = %q, mau HIGH (Doze bypass)", sent.Priority)
	}
	if sent.Data["is_test"] != "true" {
		t.Fatalf("is_test hilang dari data: %v", sent.Data)
	}
	if sent.Data["validity_ms"] != "90000" {
		t.Fatalf("validity tidak dinyatakan: %v", sent.Data)
	}
	if _, hasLocal := sent.Data["trusted_local"]; hasLocal {
		t.Fatal("trusted_local bocor ke frame pra-wujud uji")
	}
	sender.mu.Lock()
	n := len(sender.sends)
	sender.mu.Unlock()
	// NOL fallback: satu pengangkatan, satu pengiriman — tidak ada pengiriman
	// kedua ke GeoTopic pada kondisi severe maupun apa pun (pola testalert.go).
	if n != 1 {
		t.Fatalf("pengiriman FCM = %d, pastikan tepat 1 (fallback dilarang)", n)
	}
}
