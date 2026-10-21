// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package dispatch

import (
	"context"
	"time"
)

// TestAlertsTopic adalah topic FCM untuk peringatan LATIHAN — terpisah dari
// GeoTopic dan dari UpdatesTopic, dan hanya dilanggani build debug.
//
// Ini adalah pagar pertama dari dua. Sebuah drill yang sampai ke perangkat
// pengguna sungguhan mengajari orang bahwa sirene aplikasi ini kadang tidak
// berarti apa-apa, dan tidak ada rilis berikutnya yang bisa mencabut pelajaran
// itu. Karena itu jalurnya dibuat sedemikian sehingga tidak ada satu pun
// keadaan di mana instalasi produksi *berlangganan* peringatan latihan:
// pemisahan topic membuat pengiriman itu mustahil, bukan sekadar tidak
// diinginkan. Pagar keduanya ada di klien, yang menjatuhkan frame ber-is_test
// pada build rilis (docs/CLIENT_SPEC.md §5.8) — dua pagar independen, sebab
// satu kekeliruan konfigurasi tidak boleh cukup untuk melewati keduanya.
const TestAlertsTopic = "test_alerts"

// testAlertResolveAfter adalah jarak antara drill dan all-clear-nya. var (bukan
// const) HANYA agar uji dapat memangkasnya — target maupun janji D-018 tidak
// berubah: produksi membaca 20 s, dan uji tidak pernah memasang yang lain
// pada jalur produksi (field-nya tidak diubah dari setter mana pun).
//
// Lebih pendek dari defaultResolveAfter (90s): drill tidak punya baris
// earthquake_events untuk ditutup, tidak ada state machine yang bergantung
// padanya, dan penguji yang menunggu satu setengah menit untuk melihat layar
// siaga mati akan mengira fiturnya rusak.
var testAlertResolveAfter = 20 * time.Second

// DispatchTestAlert menyiarkan peringatan LATIHAN.
//
// Tiga hal yang TIDAK dilakukannya, dan masing-masing disengaja:
//
//   - tidak menulis earthquake_events. Drill tidak boleh muncul di riwayat
//     "aktivitas terkini" pengguna maupun menggeser hitungan 30 hari; riwayat
//     gempa yang memuat gempa yang tidak pernah terjadi tidak dapat dipercaya
//     untuk apa pun.
//   - tidak menyentuh consensus maupun state machine resolusi milik dispatcher
//     (map active). Sebuah drill yang menyita entri di sana dapat menahan
//     all-clear gempa sungguhan yang datang setelahnya.
//   - tidak pernah mengirim ke GeoTopic atau ke token bertarget. Hanya
//     TestAlertsTopic. Tidak ada fallback: bila tidak ada satu pun perangkat
//     debug yang berlangganan, drill berhenti — itu hasil yang benar.
//
// All-clear-nya tetap dikirim, dan dengan is_test yang sama, karena layar siaga
// di perangkat penguji perlu dimatikan oleh jalur yang sama seperti aslinya.
func (d *Dispatcher) DispatchTestAlert(msg *AlertMessage) {
	msg.IsTest = true
	if msg.Type == "" {
		msg.Type = TypeAlert
	}
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}
	// Drill memakai resolve-after drill-nya sendiri (20s, bukan 90s): masa
	// berlaku yang dinyatakan harus sama dengan janji all-clear jalur ini
	// (D-018), kalau tidak klien menahan drill 90 detik setelah layarnya mati.
	msg.ValidityMs = testAlertResolveAfter.Milliseconds()

	d.hub.Broadcast(msg)
	d.dispatchTestFCM(msg)

	d.log.Warn("peringatan LATIHAN didispatch (tidak dipersistensi)",
		"event_id", msg.EventID, "mmi", msg.MMI, "topic", TestAlertsTopic)

	go d.resolveTestAlert(*msg)
}

// dispatchTestFCM mengirim drill. Target-nya EKSKLUSIF:
//   - TargetToken kosong (drill biasa) → topic TestAlertsTopic saja.
//   - TargetToken terisi (drill bertarget per perangkat, target_pseudonym) →
//     TEPAT SATU token itu — TANPA topic, tanpa fallback broadcast, dan
//     TANPA jalur alert nyata (fungsi ini tidak pernah menyentuh saver atau
//     state machine; lihat DispatchTestAlert).
//
// Satu kondisi yang sama (msg.TargetToken) menentukan kEDUA sisi eksklusif
// itu, sehingga satu pesan drill tidak pernah bisa mengisi topic dan token
// sekaligus — target perangkat tidak mungkin bocor jadi broadcast nasional,
// dan kesalahan pengiriman ke token tidak pernah memicu fallback ke topic
// mana pun. Priority HIGH: yang sedang diuji justru apakah sebuah peringatan
// mampu membangunkan perangkat dari Doze, dan drill yang dikirim dengan
// prioritas lain tidak menguji hal itu.
func (d *Dispatcher) dispatchTestFCM(msg *AlertMessage) {
	if d.fcm == nil {
		return
	}
	data := BuildAlertData(msg)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), fcmTimeout)
		defer cancel()
		fcmMsg := &FCMMessage{Data: data, Priority: "HIGH"}
		if msg.TargetToken != "" {
			fcmMsg.Token = msg.TargetToken
		} else {
			fcmMsg.Topic = TestAlertsTopic
		}
		if err := d.fcm.Send(ctx, fcmMsg); err != nil {
			d.log.Error("gagal kirim FCM latihan", "err", err, "event_id", msg.EventID)
		}
	}()
}

// resolveTestAlert mengirim all-clear drill setelah testAlertResolveAfter.
// Menyalin pesannya (bukan menyimpan pointer) agar tidak ada yang dibagi
// dengan jalur alert sungguhan.
func (d *Dispatcher) resolveTestAlert(orig AlertMessage) {
	timer := time.NewTimer(testAlertResolveAfter)
	defer timer.Stop()
	<-timer.C

	resolved := orig
	resolved.Type = TypeResolved
	resolved.Timestamp = time.Now().UnixMilli()
	// All-clear drill mewarisi masa berlaku drill (D-018); lihat catatan pada
	// DispatchTestAlert. Jalur stand-down klien tidak digerbangi usia.
	resolved.ValidityMs = testAlertResolveAfter.Milliseconds()

	d.hub.Broadcast(&resolved)
	d.dispatchTestFCM(&resolved)

	d.log.Info("all-clear LATIHAN didispatch", "event_id", resolved.EventID)
}
