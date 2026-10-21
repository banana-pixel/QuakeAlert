// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package event

import (
	"context"

	"github.com/banana-pixel/quakealert/server/internal/consensus"
)

// --- Bench peringatan pra-wujud uji (S4 Tambahan, Opsi B — otorisasi pemilik) ---
//
// TEPAT-SEKALI satu tepi-PRELIM SINTETIS masuk ke penadahan yang sama dengan
// jalur produksi (emitTrustedPrelim → observePrelim → gate → send), lewat SATU
// metode Bridge itu. Kelima pagar yang membuatnya aman:
//
//   - NOL jalur lain: metode ini TIDAK memanggil FrameFor, TIDAK menyentuh
//     b.sink (advisory D-043 skipped), TIDAK menyentuh trusted_local, TIDAK
//     memanggil persist/RecordEventUnit, TIDAK mengubah state machine — frame
//     is_test satu-satunya lawatannya adalah penadahan uji.
//   - Gate BEARER SAMA: kelayakan dinilai DARI STATE ADMIN NODE asli
//     (adminNodeSource — designated, verified, heartbeat; lantai 60 gal di
//     bawah relaksasi ADMIN_NOT_FINAL). Bila gate menolak, bench melaporkan
//     penolakan — penolakan itu ADALAH hasil yang benar.
//   - Kontributor terpalsu HANYA identitas node Admin asli + PGA masukan:
//     bench tidak bisa mengarang kelayakan di luar state yang server tau.
//   - Mode OFF: penadahan nil → "inactive" SEBELUM source ke mana pun
//     disentuh — nol kirim, nol I/O.
//   - Token TIDAK pernah lewat sini: penadahan mengafolehnya ke Dispatcher
//     yang pinned (SetPrelimTestTargetToken); outcome hanya teks + reason.

// Hasil BENCH yang mungkin. raised/refused keduanya ke test-alert endpoint
// dengan status 200 (penolakan gate adalah pengamatan, bukan galat server);
// inactive untuk mode OFF / kait tidak terpasang diterjemahkan api ke 503.
const (
	PrelimBenchRaised   = "raised"
	PrelimBenchRefused  = "refused"
	PrelimBenchInactive = "inactive"
)

// PrelimTestInput adalah masukan BENCH yang boleh dilayani: identitas drill
// (EventID "test-" dari api), intensitas, dan koordinat. Field siklus hidup
// lain (state, revision, bukti) diisi BENCH sendiri — operator tidak boleh
// bisa memalsukan state machine lewat endpoint uji.
type PrelimTestInput struct {
	// EventID wajib berawalan "test-" (pemanggil api yang membuktikannya):
	// dedup perangkat dan label latihan bergantung padanya.
	EventID      string
	PGAGal       float64
	CentroidLat  float64
	CentroidLon  float64
	LocationName string
	Timestamp    int64 // ms epoch
}

// PrelimTestBench menawarkan SATU tepi-PRELIM sintetis ke penadahan yang
// dipasang (S4 mode test mana pun — topik maupun bertarget). Reason kembalian
// mengikuti label gate bayangan sehingga log dan respons operator bercerita
// sama.
func (b *Bridge) PrelimTestBench(ctx context.Context, in PrelimTestInput) (outcome, reason string) {
	if b == nil {
		return PrelimBenchInactive, "bridge tidak ada"
	}
	// Penadahan diuji DULU: mode OFF harus tidak melakukan I/O apa pun
	// (adminSource tidak disentuh), bukan sekadar tidak mengirim.
	if !b.prelimWarning.enabled() {
		return PrelimBenchInactive, "penadahan pra-wujud tidak aktif (prelim test warning off)"
	}
	src := b.adminSource
	if src == nil {
		return PrelimBenchInactive, "kait Admin Node tidak terpasang"
	}
	lctx, cancel := context.WithTimeout(ctx, adminNodeLoadTimeout)
	defer cancel()
	state, ok := src.GetAdminNodeState(lctx)
	if !ok {
		// Fail-closed yang sama dengan jalur emisi: tanpa status tepercaya,
		// tanpa peringatan — termasuk dari bench.
		return PrelimBenchInactive, "status Admin Node gagal-tertutup (designate/verifikasi/heartbeat tidak terbaca)"
	}
	would, gateReason := b.prelimWarning.observePrelim(state, benchPrelimSnapshot(in, state.StationID))
	return benchOutcome(would, gateReason)
}

// benchPrelimSnapshot mengemas masukan bench menjadi Snapshot yang SETARA
// tepi-PRELIM asli: selalu StateUnconfirmed, kontributor SATU node Admin asli
// ber-fase PRELIM (pga = masukan operator), dan label intensitas dari fungsi
// KONSOL yang sama dengan jalur gempa (consensus.Intensity) — bench tidak
// dapat mengangkat kombinasi gempa nyata tidak pernah menghasilkan. State
// machine tidakusik: Snapshot ini tidak pernah masuk tracker.
func benchPrelimSnapshot(in PrelimTestInput, stationID string) Snapshot {
	mmi, label := consensus.Intensity(in.PGAGal)
	return Snapshot{
		EventID:          in.EventID,
		To:               StateUnconfirmed,
		Reason:           ReasonFirstObservation,
		Revision:         1,
		OriginTS:         in.Timestamp,
		OriginTSSource:   "SERVER",
		DecidedAt:        in.Timestamp,
		CreatedAt:        in.Timestamp,
		CentroidLat:      in.CentroidLat,
		CentroidLon:      in.CentroidLon,
		LocationName:     in.LocationName,
		PeakPGA:          in.PGAGal,
		MMIScale:         mmi,
		IntensityLabel:   label,
		NodeCount:        1,
		IndependentCells: 1,
		Evidence: EvidenceSummary{
			Contributors: []ContributorEvidence{{
				NodeID:      stationID,
				PeakPGA:     in.PGAGal,
				Phase:       PhasePrelim,
				OnsetTS:     in.Timestamp,
				OnsetSource: "SERVER",
			}},
			IndependentCells: 1,
			CellIDs:          []CellID{{}}, // satu sel sintetis — identitas tidak dipakai penadahan
			OriginTSSource:   "SERVER",
		},
	}
}

// benchOutcome menerjemahkan (would, reason) observePrelim ke outcome bench.
func benchOutcome(would bool, reason string) (string, string) {
	// reason kosong seharusnya tidak terjadi; dinyatakan eksplisit supaya
	// tidak ada respons dengan field kosong yang menipu pembaca laporan.
	if reason == "" {
		reason = "UNKNOWN"
	}
	if would {
		return PrelimBenchRaised, reason
	}
	return PrelimBenchRefused, reason
}
