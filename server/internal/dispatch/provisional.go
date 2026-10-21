// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package dispatch

import (
	"context"
)

// TestProvisionalValidityMs adalah masa berlaku DPE-6 PLACEHOLDER (basis
// "DPE-6-PLACEHOLDER-90S") untuk label uji pra-wujud yang tidak dinyatakan
// pengirimnya. Angka yang sama dengan constanta pada event/prelim_warning.go
// — keduanya merujuk dokumen yang sama, bukan nilainya yang diukur.
const TestProvisionalValidityMs = 90_000

// BroadcastProvisionalTestFrame menyalurkan frame peringatan pra-wujud UJI
// (PRELIM sebelum FINAL; otorisasi pemilik S4) mengikuti pola drill di
// testalert.go, dengan TIGA pagar yang dipegang di sini:
//
//   - FCM topik TestAlertsTopic ATAU — bila SetPrelimTestTargetToken
//     mematok mode BERTARGET — SATU token, EKSKLUSIF. Dua mode tidak pernah
//     bercampur: cabang di bawah bersifat saling-mengecualikan dan TIDAK ADA
//     fallback menerka (kegagalan kirim ke token = kegagalan penuh, tanpa
//     topik). Token tidak pernah dilog dan tidak pernah masuk frame.
//   - IsTest DIJAMIN true secara defensif: tidak ada satu pun jalur yang
//     boleh menulis frame non-uji ke penadahan ini, dan mapper klien
//     menjatuhkan frame is_test pada build rilis (CLIENT_SPEC §5.8).
//   - TIDAK ADA timer resolusi di sini: penarikan milik event/prelim_warning.go
//     (spend-once, xor satu-outcome). Timer kedua DI SINI berarti dua penarik
//     untuk satu pengangkatan — justru perang yang dilarang pemilik.
//
// Validity: dinyatakan pengirim (event pkg, placeholder DPE-6); dicap bila
// lupa, karena frame uji tanpa validity jatuh ke "jendela legacy" klien —
// tidak pernah kedaluwarsa di luar pengawasan penarik yang menjadi jangkar.
func (d *Dispatcher) BroadcastProvisionalTestFrame(ctx context.Context, msg *AlertMessage) {
	if msg == nil || d == nil {
		return
	}
	msg.IsTest = true
	if msg.ValidityMs == 0 {
		msg.ValidityMs = TestProvisionalValidityMs
	}
	d.hub.Broadcast(msg)
	if d.fcm == nil {
		return
	}
	data := BuildAlertData(msg)
	go func() {
		tctx, cancel := context.WithTimeout(context.Background(), fcmTimeout)
		defer cancel()
		fcmMsg := &FCMMessage{
			Data:     data,
			Priority: "HIGH",
		}
		if d.prelimTestTargetToken != "" {
			// Mode BERTARGET: SATU token, tanpa topik, tanpa fallback.
			fcmMsg.Token = d.prelimTestTargetToken
		} else {
			fcmMsg.Topic = TestAlertsTopic
		}
		if err := d.fcm.Send(tctx, fcmMsg); err != nil {
			d.log.Error("gagal kirim FCM pra-wujud uji", "err", err, "event_id", msg.EventID)
		}
	}()
	if d.prelimTestTargetToken != "" {
		d.log.Info("frame pra-wujud UJI disiarkan (token bertarget)", "event_id", msg.EventID,
			"type", msg.Type, "event_state", msg.EventState,
			"event_revision", msg.EventRevision, "validity_ms", msg.ValidityMs)
		return
	}
	d.log.Info("frame pra-wujud UJI disiarkan (topik uji saja)",
		"event_id", msg.EventID, "type", msg.Type, "event_state", msg.EventState,
		"event_revision", msg.EventRevision, "validity_ms", msg.ValidityMs)
}
