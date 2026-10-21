// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package event

import (
	"log/slog"
	"sync"
)

// --- BAYANGAN tepi-PRELIM (shadow) — HANYA-CATAT, TANPA EMISI ---
//
// Sumber kebenaran: docs/planning/provisional-warning/01-decision-draft-e2.md
// (DPE-1..8) dan 05-sequencing-rollback.md langkah 7 (mode bayangan
// logging-only). Otorisasi pemilik (S1–S3): evaluasi BAYANGAN alarm di tepi
// PRELIM, dilarang keras memancarkan apa pun.
//
// INVARIAN yang dijaga test (prelim_shadow_test.go):
//   - TIDAK PERNAH memanggil sink mana pun — frameSink maupun
//     trustedLocalSink; tidak ada FCM, tidak ada WebSocket, tidak ada
//     trusted_local, tidak ada apa pun yang dapat memicu siren.
//   - TIDAK PERNAH menyentuh localAlarmed, dedup, revisi, tracker, classify —
//     kelayakan alarm asli, dedup, dan revisi tidak berubah satu bit.
//   - Satu-satunya state baru adalah peta in-memory milik bayangan sendiri;
//     hilang saat proses restart (catatan riset, bukan durabilitas).
//   - Setiap catatan terlabel shadow=true + simulated=true: angka di dalamnya
//     adalah HIPOTETIS, bukan dispatch nyata.
//
// Tempat log: logger bawaan paket (slog.Default()) — di produksi
// cmd/quakealert/main.go memasangnya ke JSON handler di os.Stdout, sehingga
// setiap catatan jatuh ke stdout Docker yang sudah dibatasi rotasi
// (json-file, max-size 10m, max-file 3 — deploy/docker-compose.prod.yml).
// Tidak ada berkas log khusus yang diperkenalkan.
const (
	// ShadowPrelimMsg adalah label pertama setiap catatan bayangan, agar
	// `docker logs | grep PRELIM_SHADOW` menarik semuanya sekaligus.
	ShadowPrelimMsg = "PRELIM_SHADOW"

	// shadowValidityPlaceholderMs adalah masa-berlaku HIPOTETIS yang dicatat
	// setiap catatan would-dispatch (DPE-6): valid sampai re-evaluasi FINAL,
	// dengan batas keras jendela alarm 90 s. PLACEHOLDER — distribusi
	// prov-lead produksi belum terukur (leg produksi prelim_walltime
	// terutang); menulis angka lain di sini berarti mengarang pengukuran.
	shadowValidityPlaceholderMs = 90_000

	// shadowValidityBasis menyatakan asal-usul angka validity di atas, sehingga
	// pembaca catatan tidak dapat menyamar angka desain sebagai pengukuran.
	shadowValidityBasis = "DPE-6-PLACEHOLDER-90S"
)

// prelimShadow adalah state BAYANGAN per Bridge. Nilai nol siap pakai.
type prelimShadow struct {
	// recorded: event_id -> would-dispatch (bool) untuk episode yang catatan
	// tepi-PRELIM-nya sudah dicatat. Tepat-sekali per episode; entri dihapus
	// pada outcome pertama (FINAL edge, CONFIRMED, atau terminal).
	recorded sync.Map

	// log opsional untuk uji; nol berarti logger bawaan (produksi: stdout JSON).
	log *slog.Logger
}

// logger mengembalikan logger yang dipakai catatan bayangan.
func (p *prelimShadow) logger() *slog.Logger {
	if p.log != nil {
		return p.log
	}
	return slog.Default()
}

// shadowPrelimVerdict adalah inti evaluasi BAYANGAN: jalankan evaluator
// kelayakan FINAL yang ASLI (EvaluateAdminNodeEligibility), lalu satu-satunya
// penafsiran bayangan: veto ADMIN_NOT_FINAL milik kontribusi PRELIM diabaikan
// (DPE-1 — relaksasi veto pada tepat satu cabang).
//
// KELengkapan GERBANG: pada urutan evaluator asli, veto ADMIN_NOT_FINAL
// MENUTUPI gerbang di belakangnya (lantai PGA, heartbeat). Relaksasi hanya
// pada veto itu sendiri — lantai 60 gal (D-038) dan kesegaran heartbeat
// TETAP dinilai apa adanya terhadap kontribusi PRELIM. Tanpa ini, PGA di
// bawah lantai akan tercatat would=true padahal DPE-1 menjamin sebaliknya.
// Murni: tidak membaca I/O, jam, maupun state global.
func shadowPrelimVerdict(state AdminNodeState, s Snapshot) (wouldDispatch bool, reason string) {
	d := EvaluateAdminNodeEligibility(state, s)
	if d.Eligible {
		return true, d.Reason
	}
	if d.Reason != ReasonAdminNodeNotFinal {
		return false, d.Reason
	}
	c := adminContribution(state.StationID, s.Evidence.Contributors)
	switch {
	case c == nil || c.Phase != PhasePrelim:
		// Sabuk-pengaman: relaksasi hanya berlaku bila kontribusinya memang
		// kaki PRELIM (sampai di sini contrib pasti ada dan fasenya bukan FINAL).
		return false, d.Reason
	case c.PeakPGA < AdminNodeMinPGAGal:
		return false, ReasonAdminNodeBelowFloor
	case state.HeartbeatAge > AdminNodeHeartbeatMaxAge:
		return false, ReasonAdminNodeHeartbeatStale
	}
	return true, d.Reason
}

// shadowAttrs menyusun atribut label yang wajib ada di SETIAP catatan.
func shadowAttrs() []any {
	return []any{"shadow", true, "simulated", true, "note", "shadow-simulated-bukan-dispatch"}
}

// observePrelim mencatat BAYANGAN dispatch provisional untuk satu snapshot
// tepi-PRELIM. Dipanggil dari emitTrustedPrelim SETELAH status operator
// dimuat — evaluasi berjalan di atas state yang SAMA, nol I/O tambahan.
//
// Penyaring: hanya event dengan node admin designated yang dicatat; tanpa
// designate, kebisingan NO_ADMIN_DESIGNATED per-event negara tidak
// bernilai riset. Tepat-sekali per event_id (LoadOrStore): publish ulang
// maupun edge berulang tidak menggandakan catatan.
func (p *prelimShadow) observePrelim(state AdminNodeState, s Snapshot) {
	if !state.Designated {
		return
	}
	would, reason := shadowPrelimVerdict(state, s)
	if _, loaded := p.recorded.LoadOrStore(s.EventID, would); loaded {
		return
	}
	attrs := append(shadowAttrs(),
		"event_id", s.EventID,
		"revision", s.Revision,
		"would_dispatch", would,
		"reason", reason,
		"floor_gal", AdminNodeMinPGAGal,
		"would_validity_ms", shadowValidityPlaceholderMs,
		"validity_basis", shadowValidityBasis,
		"decided_at", s.DecidedAt,
	)
	if c := adminContribution(state.StationID, s.Evidence.Contributors); c != nil {
		attrs = append(attrs, "contrib_pga_gal", c.PeakPGA, "event_peak_pga_gal", s.PeakPGA)
	}
	p.logger().Info(ShadowPrelimMsg, attrs...)
}

// observeLocalEval mencatat HASIL BAYANGAN re-evaluasi FINAL (DPE-4) untuk
// episode yang catatan PRELIM-nya would-dispatch: layak -> absorb (bisu,
// tepat-sekali menyerap), tidak layak -> withdraw (penarikan). Dipanggil
// dari emitTrustedLocal pada state yang sudah dimuat (nol I/O tambahan).
// Snapshot berkontribusi PRELIM lewat sini diabaikan — catatannya ada di
// tepi PRELIM, bukan di kaki transisi.
//
// Satu outcome per episode: entri dihapus pada evaluasi FINAL pertama,
// sehingga FINAL berulang (retry) tidak menggandakan outcome.
func (p *prelimShadow) observeLocalEval(state AdminNodeState, s Snapshot) {
	if !state.Designated {
		return
	}
	c := adminContribution(state.StationID, s.Evidence.Contributors)
	if c == nil || c.Phase != PhaseFinal {
		return
	}
	v, loaded := p.recorded.LoadAndDelete(s.EventID)
	if !loaded {
		return // episode tanpa catatan: tidak ada yang dinilai ulang
	}
	would, isBool := v.(bool)
	if !isBool || !would {
		return // would=no: DPE-4 tidak berlaku (tidak pernah ada provisional)
	}
	d := EvaluateAdminNodeEligibility(state, s)
	outcome := "withdraw"
	if d.Eligible {
		outcome = "absorb"
	}
	p.logger().Info(ShadowPrelimMsg, append(shadowAttrs(),
		"event_id", s.EventID,
		"revision", s.Revision,
		"would_outcome", outcome,
		"reason", d.Reason,
		"floor_gal", AdminNodeMinPGAGal,
		"contrib_pga_gal", c.PeakPGA,
		"event_peak_pga_gal", s.PeakPGA,
		"phase", c.Phase,
		"decided_at", s.DecidedAt,
	)...)
}

// observeTerminal mencatat outcome bayangan ketika event berpindah
// CONFIRMED / RESOLVED / CANCELLED sebelum kaki FINAL sempat menilai
// (DPE-4 baris "resolved/expired"): CONFIRMED -> diserap jalur normal,
// terminal -> withdraw. Murni: membaca peta bayangan sendiri, tidak
// membaca status operator, tidak menyentuh localAlarmed.
func (p *prelimShadow) observeTerminal(s Snapshot) {
	var outcome string
	switch s.To {
	case StateConfirmed:
		outcome = "confirmed"
	case StateResolved, StateCancelled:
		outcome = "withdraw"
	default:
		return // DETECTED/UNCONFIRMED bukan penutup episode bayangan
	}
	v, loaded := p.recorded.LoadAndDelete(s.EventID)
	if !loaded {
		return
	}
	if would, isBool := v.(bool); !isBool || !would {
		return // would=no tidak pernah punya provisional untuk diserap/ditarik
	}
	p.logger().Info(ShadowPrelimMsg, append(shadowAttrs(),
		"event_id", s.EventID,
		"revision", s.Revision,
		"would_outcome", outcome,
		"to", string(s.To),
		"decided_at", s.DecidedAt,
	)...)
}
