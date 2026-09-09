package dispatch

import (
	"context"
	"time"

	"github.com/banana-pixel/quakealert/server/internal/ledger"
)

// DispatchEventFrame adalah jalur emisi Fase 3: SATU frame yang sudah selesai
// diputuskan oleh event.Tracker, dikirim ke kanal klien apa adanya.
//
// Perbedaannya dari Dispatch bukan kosmetik. Dispatch memutuskan sendiri (status
// -> tipe), mempersistensi event secara SINKRON untuk mendapatkan event_id, dan
// memasang timer resolusinya sendiri. Ketiganya pindah ke Tracker pada Fase 3:
// identitas dibuat di Go sebelum penulisan mana pun (§4.1), persistensi mengikuti
// emisi dan boleh gagal (§9.5), dan resolusi dimiliki satu sweeper (§5.4). Yang
// tersisa untuk dispatcher adalah pengiriman — dan itulah seluruh isi fungsi ini.
//
// push memisahkan kanal, bukan tipe: UNCONFIRMED disiarkan lewat WebSocket dan
// TIDAK mendorong FCM (D10), sedangkan RESOLVED/CANCELLED mendorong hanya bila
// event-nya pernah CONFIRMED (§8.1). Keputusan itu dibuat pemanggil dari
// snapshot; di sini ia hanya dijalankan.
func (d *Dispatcher) DispatchEventFrame(ctx context.Context, msg *AlertMessage, push bool) {
	if msg == nil {
		return
	}

	// Masa berlaku yang dinyatakan pengirim (D-018): frame Fase 3 dibangun
	// Tracker tanpa mengetahui resolve-after dispatcher, jadi transport
	// mengisinya bila pemanggil belum menyatakan. Nilai yang sudah
	// dinyatakan eksplisit tidak pernah ditimpa — pengirim yang tahu
	// (misalnya drill dengan janji 20 detiknya) menang atas default.
	if msg.ValidityMs == 0 {
		msg.ValidityMs = d.resolveAfter.Milliseconds()
	}

	// 1. WebSocket lebih dulu, selalu, untuk setiap transisi. Non-blocking.
	wsCount := d.hub.Broadcast(msg)

	// 2. FCM hanya bila transisi ini berhak atasnya.
	if push {
		d.dispatchFCM(msg, wsCount)
		return
	}

	// Tanpa FCM, baris alert_emissions tetap ditulis: §8.5 mengharuskan setiap
	// frame yang MUNGKIN diterima klien dapat direkonstruksi, dan sebuah frame
	// WebSocket-saja adalah frame yang diterima klien.
	//
	// fcmConfigured mengikuti apakah FCM ADA, bukan apakah ia dipakai. Dengan
	// begitu "dikonfigurasi tetapi sengaja tidak mengirim" tercatat sebagai
	// fcm_attempted = 0 — nol yang teramati, sama seperti guard satu-node —
	// sementara instalasi tanpa kredensial tetap NULL.
	d.recordEmission(msg, ledger.AudienceNone, time.Now().UnixMilli(), delivery{
		wsClients: wsCount, fcmConfigured: d.fcm != nil,
	})
}

// TrustedLocalRadiusKm adalah radius peringatan lokal Admin Node (D-036
// PROPOSED): token dalam 20 km dari centroid menerima frame trusted_local.
// Konstanta sendiri, bukan parameter, dengan alasan yang sama seperti
// AlertRadiusKm — dan NILAINYA berbeda dengan sengaja: peringatan otoritas
// satu-node tidak boleh menjangkau sejauh peringatan terkonfirmasi jaringan.
const TrustedLocalRadiusKm = 20

// DispatchTrustedLocalEventFrame adalah jalur emisi peringatan lokal Admin
// Node: SATU frame EARTHQUAKE_ALERT bertanda trusted_local, disiarkan ke
// WebSocket seperti setiap frame, lalu dikirim FCM hanya ke token dalam
// TrustedLocalRadiusKm.
//
// Berbeda dari dispatchFCM dalam tepat dua hal, dan keduanya disengaja:
//   - radius token 20 km (bukan AlertRadiusKm), dan
//   - TANPA fallback GeoTopic dalam keadaan apa pun — termasuk severe,
//     termasuk guard-nonaktif, termasuk tanpa token. Tanpa token berarti
//     AudienceNone (nol yang teramati bila FCM dikonfigurasi).
//
// singleNodeGeoTopicGuard TIDAK disentuh: jalur ini tidak pernah menyentuh
// GeoTopic, jadi guard tetap berlaku penuh pada jalur normal. Fungsi ini
// sinkron (lookup + kirim + catat); pemanggil (Bridge Admin Node) yang
// menjalankannya di luar jalur publikasi.
func (d *Dispatcher) DispatchTrustedLocalEventFrame(ctx context.Context, msg *AlertMessage) {
	if msg == nil {
		return
	}
	if msg.ValidityMs == 0 {
		msg.ValidityMs = d.resolveAfter.Milliseconds()
	}
	// decidedAt sinkron seperti dispatchFCM: waktu keputusan, bukan waktu
	// pengiriman.
	decidedAt := time.Now().UnixMilli()

	// WebSocket lebih dulu, seperti setiap frame: klien foreground menerima
	// peringatan lokal lewat kanal yang sama dengan peringatan lain.
	wsCount := d.hub.Broadcast(msg)

	if d.fcm == nil {
		d.recordEmission(msg, ledger.AudienceNone, decidedAt, delivery{wsClients: wsCount})
		return
	}
	data := BuildAlertData(msg)
	tokens := d.trustedLocalTokens(ctx, msg)
	if len(tokens) == 0 {
		d.log.Info("peringatan lokal tanpa audiens: tanpa token dalam 20 km, FCM tidak dikirim",
			"event_id", msg.EventID, "type", msg.Type, "nodes", msg.NodeCount)
		d.recordEmission(msg, ledger.AudienceNone, decidedAt, delivery{
			wsClients: wsCount, fcmConfigured: true,
		})
		return
	}
	attempted, succeeded := d.sendToTokens(ctx, tokens, data, msg, TrustedLocalRadiusKm)
	d.recordEmission(msg, ledger.AudienceTokensRadiusLocal, decidedAt, delivery{
		wsClients: wsCount, fcmAttempted: attempted, fcmSucceeded: succeeded,
		fcmConfigured: true,
	})
}

// trustedLocalTokens mengembalikan token dalam TrustedLocalRadiusKm dari
// centroid, atau nil bila store tidak mendukung pencarian / query gagal.
// Kegagalan di sini bukan kegagalan dispatch: pemanggil mencatat AudienceNone
// — TIDAK ADA fallback topik pada jalur ini, itulah seluruh maksudnya.
func (d *Dispatcher) trustedLocalTokens(ctx context.Context, msg *AlertMessage) []string {
	finder, ok := d.saver.(tokenFinder)
	if !ok {
		return nil
	}
	tokens, err := finder.FCMTokensWithin(ctx, msg.CentroidLat, msg.CentroidLon, TrustedLocalRadiusKm)
	if err != nil {
		d.log.Error("gagal cari token FCM lokal", "err", err, "event_id", msg.EventID)
		return nil
	}
	return tokens
}
