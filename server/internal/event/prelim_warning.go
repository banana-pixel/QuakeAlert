// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package event

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/banana-pixel/quakealert/server/internal/dispatch"
)

// --- Peringatan PRA-WUJUD (PRELIM) mode UJI — penadah emisi TEPAT-SEKALI ---
//
// Otorisasi pemilik (S4, kondisi pemilik): tidak ada pengguna publik, jadi
// path kode produksi boleh disentuh, dengan pagar-pagar ini permanen:
//
//   - Topik SATU-SATUNYA yang boleh dipakai adalah test_alerts
//     (testalert.go — di sana penjelasan drillnya); build rilis tidak pernah
//     berlangganan topik itu (PushRegistrar debug-only) dan mapper klien
//     menjatuhkan frame is_test pada build rilis (dua pagar independen).
//     Android TIDAK diubah — frame ini menunggangi jalur alarm drill yang
//     sudah ada.
//   - Gerbang kelayakan: shadowPrelimVerdict APA ADANYA (evaluator FINAL
//     asli; satu-satunya relaksasi veto ADMIN_NOT_FINAL kaki PRELIM; lantai
//     60 gal D-038, verified, designated, heartbeat, UNCONFIRMED tetap).
//   - Validity DPE-6 placeholder 90 s (basis "DPE-6-PLACEHOLDER-90S") —
//     eksplisit hanya uji, diganti angka ukur setelah data produksi ada.
//
// Kedisiplinan penarikan (kondisi pemilik: TIDAK BOLEH men-clear peringatan
// yang lebih baru):
//
//   - SATU outcome per episode (XOR): salinan upgrade FINAL ATAU all-clear,
//     tidak pernah keduanya; tertangani spend-once di bawah mutex.
//   - CONFIRMED pada transisi → BISU: alarm nyata yang lebih baru sedang
//     hidup atau akan diangkat jalur normal; mengirim all-clear di sini = meniup
//     peringatan yang lebih baru. Timer pendamping ikut dibatalkan.
//   - Timer placeholder hanyalah penarik terakhir: ia menangis bila TIDAK ada
//     outcome (upgrade/withdrawal/terminal) sebelum masa berlaku. Ia di-Stop
//     setiap kali persaingan outcome apa pun dimenangkan, sehingga penarikan ganda
//     tidak pernah diemisikan.
//   - Restart proses: peta pending hilang → tidak ada penarikan yang pernah
//     dikirim (perangkat memegang peringatan sampai frame nyata berikutnya
//     atau pengguna). TIDAK ADA re-raise: tepi-PRELIM yang re-fire setelah
//     restart membawa revision yang sama → dedup perangkat (AlertDedup,
//     type+event_id, revision-gated) menanggulkannya.
const (
	// prelimTestWarningValidityMs adalah masa berlaku PLACEHOLDER (DPE-6,
	// basis "DPE-6-PLACEHOLDER-90S") yang dinyatakan pada SETIAP frame
	// peringatan pra-wujud uji. Bukan pengukuran.
	prelimTestWarningValidityMs = 90_000

	// prelimWarningSendTimeout membungkus pengiriman penadah (topik uji)
	// dengan deadline sendiri — DAFTAR OUTCOME yang kehabisan waktu tidak
	// pernah melumpuhkan penarik pukul.
	prelimWarningSendTimeout = 2 * time.Second
)

// PrelimTestWarningSink adalah satu-satunya jalan keluar peringatan uji
// pra-wujud: pola drill (testalert.go) — broadcast hub + FCM topik
// test_alerts SAJA. Tidak ada fallback GeoTopic, tanpa target token, tanpa
// baris ledger. Implementasi produksi: *dispatch.Dispatcher.
type PrelimTestWarningSink interface {
	BroadcastProvisionalTestFrame(ctx context.Context, msg *dispatch.AlertMessage)
}

// prelimWarning adalah state penadah per peringatan pra-wujud. Nilai nol
// (tidak pernah diatur) = tidak aktif sama sekali.
type prelimWarning struct {
	sink PrelimTestWarningSink
	log  *slog.Logger

	// validityMs menentukan masa berlaku timer penarik terakhir. Default
	// DPE-6 placeholder; disuntik di uji agar timer tidak harus 90 s.
	validityMs int64

	// targetPseudonym (S4 Tambahan): pseudonim target mode BERTARGET.
	// Kosong = mode topik (perilaku S4 asli). Pseudonim aman untuk log
	// audit; token ASLI tidak pernah berada di event pkg sama sekali — ia
	// dipatok di Dispatcher (lihat SetPrelimTestTargetToken).
	targetPseudonym string

	mu      sync.Mutex
	pending map[string]*prelimPending // event_id -> pengangkatan yang masih menunggu outcome
}

// raiseValidity adalah masa berlaku efektif untuk pengangkatan berikutnya:
// nilai nol terdegradasi ke placeholder DPE-6, bukan "tanpa validity".
func (pw *prelimWarning) raiseValidity() int64 {
	if pw.validityMs > 0 {
		return pw.validityMs
	}
	return prelimTestWarningValidityMs
}

// prelimPending menyimpan SATU pengangkatan yang belum berakhir: frame dasar
// untuk membangun frame penarikan, revision yang diangkat, dan handle timer.
type prelimPending struct {
	frame    *dispatch.AlertMessage // frame PENGANGKATAN dasar
	revision int
	timer    *time.Timer
}

// enabled menyetujui penadah aktif; receiver nil = tidak aktif (pola nil-safe
// yang sama dengan hook Admin Node).
func (pw *prelimWarning) enabled() bool {
	return pw != nil && pw.sink != nil
}

// logger memakai logger bawaan bila tidak disuntik (di produksi handler JSON
// stdout dari cmd/quakealert/main.go).
func (pw *prelimWarning) logger() *slog.Logger {
	if pw != nil && pw.log != nil {
		return pw.log
	}
	return slog.Default()
}

// prelimTestWarningAttrs menyusun label SETIAP baris penadah uji mode topik —
// topik uji saja, bukan jalur produksi.
func prelimTestWarningAttrs() []any {
	return []any{"test_warning", true, "topic", "test_alerts"}
}

// warnAttrs memilih label mode penadahan: pseudonim target (BERTARGET,
// S4 Tambahan) atau topik uji (S4 asli). Keduanya saling-mengecualikan;
// token ASLI tidak pernah nongkol di label apa pun.
func (pw *prelimWarning) warnAttrs() []any {
	if pw != nil && pw.targetPseudonym != "" {
		return []any{"test_warning", true, "target_pseudonym", pw.targetPseudonym}
	}
	return prelimTestWarningAttrs()
}

// prelimTestFrame membangun frame peringatan pra-wujud: bentuk persis
// TrustedLocalFrameFor KECUALI TrustedLocal tetap false (bukan peringatan
// otoritas D-036 — gerbang radiusnya berbeda di perangkat) plus is_test=true
// dan validity placeholder.
func prelimTestFrame(s Snapshot) *dispatch.AlertMessage {
	return &dispatch.AlertMessage{
		Type:                 dispatch.TypeAlert,
		EventID:              s.EventID,
		MMI:                  s.MMIScale,
		IntensityLabel:       s.IntensityLabel,
		PGAGal:               s.PeakPGA,
		CentroidLat:          s.CentroidLat,
		CentroidLon:          s.CentroidLon,
		LocationName:         s.LocationName,
		Timestamp:            s.DecidedAt,
		NodeCount:            s.NodeCount,
		EventState:           string(s.To),
		EventRevision:        s.Revision,
		OriginTS:             s.OriginTS,
		OriginTSSource:       s.OriginTSSource,
		IndependentCellCount: s.IndependentCells,
		IsTest:               true,
		ValidityMs:           prelimTestWarningValidityMs,
	}
}

// prelimTestWithdrawFrame membangun frame PENARIKAN dari frame dasar
// pengangkatan: EventState selalu dari vocab enum kontrak FCM
// {"UNCONFIRMED","CONFIRMED","RESOLVED","CANCELLED"} — penarikan uji memakai
// "CANCELLED" (kontrak FCM: "termasuk penarikan (event_state=CANCELLED)");
// terminal nyata membawa nilai terminalnya sendiri. Revision memakai revision
// PENGANGKATAN, sehingga salinan penarikan yang terduplikasi sampai ke
// perangkat tidak pernah menjadi "berita" lebih baru.
func prelimTestWithdrawFrame(base *dispatch.AlertMessage, state string) *dispatch.AlertMessage {
	out := *base
	out.Type = dispatch.TypeResolved
	out.EventState = state
	out.Timestamp = time.Now().UnixMilli()
	return &out
}

// send menyalurkan frame ke penadah dengan deadline sendiri — pengiriman yang
// gagal tidak pernah melumpuhkan penarik pukul.
func (pw *prelimWarning) send(msg *dispatch.AlertMessage) {
	tctx, cancel := context.WithTimeout(context.Background(), prelimWarningSendTimeout)
	defer cancel()
	pw.sink.BroadcastProvisionalTestFrame(tctx, msg)
}

// observePrelim menilai pengangkatan pada tepi PRELIM (dipanggil dari
// emitTrustedPrelim SETELAH status operator dimuat — nol I/O tambahan).
// Tepat-sekali per event: tepi berulang tidak menggandakan pengangkatan.
// Tanpa designate: kebisingan NO_ADMIN_DESIGNATED per event negara tidak
// bernilai riset, sama seperti kebijakan bayangan.
//
// Kembalian (would, reason) dibutuhkan bench pra-wujud (prelim_bench.go) untuk
// melaporkan hasil gate; pemanggil emisi normal mengabaikannya.
func (pw *prelimWarning) observePrelim(state AdminNodeState, s Snapshot) (bool, string) {
	if !pw.enabled() || !state.Designated {
		return false, ""
	}
	would, reason := shadowPrelimVerdict(state, s)
	if !would {
		pw.logger().Info("prelim test warning: tidak diangkat",
			append(pw.warnAttrs(), "event_id", s.EventID, "reason", reason)...)
		return false, reason
	}
	frame := prelimTestFrame(s)
	pw.mu.Lock()
	if _, exists := pw.pending[s.EventID]; exists {
		pw.mu.Unlock()
		return true, reason // tepat-sekali: episode ini sudah diangkat
	}
	entry := &prelimPending{frame: frame, revision: s.Revision}
	pw.pending[s.EventID] = entry
	entry.timer = time.AfterFunc(time.Duration(pw.raiseValidity())*time.Millisecond, func() {
		pw.expire(s.EventID)
	})
	pw.mu.Unlock()
	pw.send(frame)
	pw.logger().Info("prelim test warning: diangkat",
		append(pw.warnAttrs(),
			"event_id", s.EventID, "revision", s.Revision, "reason", reason,
			"validity_ms", prelimTestWarningValidityMs, "validity_basis", "DPE-6-PLACEHOLDER-90S")...)
	return true, reason
}

// observeFinalEval menilai outcome kaki FINAL (dipanggil dari emitTrustedLocal
// pada kontribusi FINAL; snapshot kaki PRELIM pasti return duluan). XOR
// satu-outcome: yang pertama masuk menang, yang kedua no-op.
func (pw *prelimWarning) observeFinalEval(state AdminNodeState, s Snapshot) {
	if !pw.enabled() {
		return
	}
	c := adminContribution(state.StationID, s.Evidence.Contributors)
	if c == nil || c.Phase != PhaseFinal {
		return
	}
	entry, ok := pw.spend(s.EventID)
	if !ok {
		return // tidak ada pengangkatan menunggu / episode tidak pernah diangkat
	}
	d := EvaluateAdminNodeEligibility(state, s)
	if d.Eligible {
		// UPGRADE (absorbsi): salinan frame FINAL ke audiens uji dengan
		// revision FINAL — dedup perangkat menganggap ini "berita" (revision
		// lebih tinggi) sehingga notifikasi pengangkatan menegaskan FINAL.
		upgraded := prelimTestFrame(s)
		pw.send(upgraded)
		pw.logger().Info("prelim test warning: upgrade FINAL",
			append(pw.warnAttrs(), "event_id", s.EventID, "reason", d.Reason,
				"revision", s.Revision)...)
		return
	}
	// WITHDRAWAL: relaksasi tidak lagi berlaku — all-clear ke audiens uji.
	withdrawn := prelimTestWithdrawFrame(entry.frame, "CANCELLED")
	pw.send(withdrawn)
	pw.logger().Info("prelim test warning: ditarik pada re-evaluasi FINAL",
		append(pw.warnAttrs(), "event_id", s.EventID, "reason", d.Reason,
			"event_state", "CANCELLED", "revision", entry.revision)...)
}

// observeTerminal menilai hasil terminal (dipanggil dari EmitTransition pada
// CONFIRMED/RESOLVED/CANCELLED). DISIPLIN ANTI-CLEAR-BARU: CONFIRMED BISU —
// alarm nyata yang lebih baru sedang hidup; terminal RESOLVED/CANCELLED
// membawa event_state nyata sebagai justifikasi penarikan.
func (pw *prelimWarning) observeTerminal(s Snapshot) {
	if !pw.enabled() {
		return
	}
	switch s.To {
	case StateConfirmed:
		if entry, ok := pw.spend(s.EventID); ok {
			pw.logger().Info("prelim test warning: absorb CONFIRMED (BISU, tanpa frame dari kami)",
				append(pw.warnAttrs(), "event_id", s.EventID,
					"event_state", string(s.To), "revision", entry.revision)...)
		}
	case StateResolved, StateCancelled:
		if entry, ok := pw.spend(s.EventID); ok {
			withdrawn := prelimTestWithdrawFrame(entry.frame, string(s.To))
			pw.send(withdrawn)
			pw.logger().Info("prelim test warning: penarikan akhir event",
				append(pw.warnAttrs(), "event_id", s.EventID,
					"event_state", string(s.To), "revision", entry.revision)...)
		}
	}
}

// expire adalah penarik terakhir bila tidak ada outcome; event_state
// "CANCELLED" (vocab kontrak untuk penarikan). spend-once mencegah kolisi
// dengan outcome nyata: hanya satu pihak yang boleh mengirim.
func (pw *prelimWarning) expire(eventID string) {
	entry, ok := pw.spend(eventID)
	if !ok {
		return // outcome nyata sudah menang / episode tidak pernah diangkat
	}
	withdrawn := prelimTestWithdrawFrame(entry.frame, "CANCELLED")
	pw.send(withdrawn)
	pw.logger().Info("prelim test warning: kedaluwarsa placeholder",
		append(pw.warnAttrs(), "event_id", eventID, "event_state", "CANCELLED",
			"revision", entry.revision, "validity_ms", prelimTestWarningValidityMs,
			"validity_basis", "DPE-6-PLACEHOLDER-90S")...)
}

// spend menghabiskan SATU pengangkatan yang menunggu tepat-sekali di bawah
// mutex, membatalkan timer-nya, dan mengembalikan entrinya; ok=false berarti
// tidak ada yang perlu dihabiskan (belum diangkat / sudah dipakai).
func (pw *prelimWarning) spend(eventID string) (*prelimPending, bool) {
	pw.mu.Lock()
	entry, ok := pw.pending[eventID]
	if !ok {
		pw.mu.Unlock()
		return nil, false
	}
	delete(pw.pending, eventID)
	timer := entry.timer
	pw.mu.Unlock()
	if timer != nil {
		timer.Stop()
	}
	return entry, true
}
