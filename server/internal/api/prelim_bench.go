// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/banana-pixel/quakealert/server/internal/consensus"
)

// --- Bench peringatan pra-wujud uji (S4 Tambahan, Opsi B — otorisasi pemilik) ---
//
// Endpoint admin yang menawarkan SATU tepi-PRELIM sintetis ke penadahan uji
// lewat Bridge (event.PrelimTestBench) — pola drill (testalert.go) versi S4:
// satu kunci admin, tanpa titik selain penadahan. Bentuknya sengaja mirip
// drill:
//   - Karena hanya kunci admin, tidak ada modifikasi router khusus.
//   - is_test=true dan id "test-" dipaksakan handler, bukan peminta.
//   - NOL penulisan earthquake_events/ledger/alert_emissions, nol konsensus,
//     nol state machine, NOL WebSocket advisory — satu-satunya lawatan frame
//     adalah penadahan uji (topik maupun token pinned, tergantung pro  mode).
//   - Token TIDAK PERNAH ada di respons/log: penargetan milik Dispatcher
//     pinned; respons operator cukup menyebut pseudonim-mode.
//
// Jalur yang TIDAK divalidasi bench ini: MQTT → tracker → tepi-PRELIM D-043
// (lihat laporan audit pemicu; armada 1 node tidak pernah menyalakan tepi itu).

// Pisahkan konstanta: event pkg adalah pemilik semantiknya
// (prelim_bench.go); nilai di sini sengaja disinkronkan — api tidak mengimpor
// event (arah import event→dispatch→api tidak ada; main yang menjembatani).
const (
	prelimBenchOutcomeRaised   = "raised"
	prelimBenchOutcomeRefused  = "refused"
	prelimBenchOutcomeInactive = "inactive"
)

// PrelimTestBenchInput adalah masukan valid yang diteruskan ke event.Bridge.
// Field siklus hidup lain diisi Bridge sendiri — operator tidak bisa malsukan
// state machine lewat endpoint uji.
type PrelimTestBenchInput struct {
	// EventID telah berawalan "test-" di handler (pemanggil tidak bisa
	// melewatkannya).
	EventID      string
	PGAGal       float64
	CentroidLat  float64
	CentroidLon  float64
	LocationName string
	Timestamp    int64
}

// PrelimTestBenchFunc melaporkan (outcome, reason) hasil bench: outcome
// "raised"/"refused" (2xx), "inactive" (503). Type sendiri, bukan
// PrelimTestWarningSink: bridge semata tidak ada di api — main yang
// menjembatani, pola yang sama dengan SetChatFanout.
type PrelimTestBenchFunc func(ctx context.Context, in PrelimTestBenchInput) (outcome, reason string)

// PrelimTestBench memasang jembatan bench pra-wujud. nil = endpoint uji tidak
// tersangkut sama sekali: permintaan menjawab 503 (fail-closed, mode OFF).
func (s *Server) SetPrelimTestBench(f PrelimTestBenchFunc) {
	if s == nil || f == nil {
		return
	}
	s.prelimBench = f
}

type prelimBenchRequest struct {
	// Rentang & semantik identik dengan drill (pga_gal 1..2000, koordinat
	// wajib): bench tanpa koordinat tidak menguji apa pun di klien.
	PGAGal       float64  `json:"pga_gal"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	LocationName string   `json:"location_name"`
}

// PrelimTestBenchResponse: outcome and reason bercerita tentang gate Admin
// Node, bukan tentang pengiriman. Token tidak di sini (target milik
// Dispatcher pinned), pseudonim-mode sudah tercatat di log boot.
type PrelimTestBenchResponse struct {
	EventID        string `json:"event_id"`
	Type           string `json:"type"`
	MMI            string `json:"mmi"`
	IntensityLabel string `json:"intensity_label"`
	IsTest         bool   `json:"is_test"`
	Outcome        string `json:"outcome"`
	Reason         string `json:"reason"`
}

// HandlePrelimTestBench POST /api/v1/admin/prelim-test — SATU tepi-PRELIM uji
// melalui gate Admin Node asli (designate/verified/heartbeat/lantai), frame
// is_test ke penadahan uji, penarikan placeholder 90 s milik event pkg
// (all-clear tetap kerja lewat mock... kalau macet? lihat prelim_warning).
func (s *Server) HandlePrelimTestBench(w http.ResponseWriter, r *http.Request) {
	var req prelimBenchRequest
	if !s.decodeBody(w, r, &req) {
		return
	}

	if req.PGAGal < minTestPGAGal || req.PGAGal > maxTestPGAGal {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT",
			"pga_gal wajib, dalam rentang 1..2000 gal")
		return
	}
	if req.Latitude == nil || req.Longitude == nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT",
			"latitude & longitude wajib: klien menyaring alert berdasarkan jarak, "+
				"jadi bench tanpa koordinat tidak menguji apa pun")
		return
	}
	if *req.Latitude < -90 || *req.Latitude > 90 {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "latitude di luar rentang -90..90")
		return
	}
	if *req.Longitude < -180 || *req.Longitude > 180 {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "longitude di luar rentang -180..180")
		return
	}

	// 503 setelah input tervalidasi — penolakan ini tentang server, bukan
	// salah ketik operator.
	if s.prelimBench == nil {
		writeError(w, http.StatusServiceUnavailable, "UNAVAILABLE",
			"penadahan uji pra-wujud belum terpasang di server ini (prelim test warning off)")
		return
	}

	location := strings.Join(strings.Fields(req.LocationName), " ")
	if location == "" {
		location = defaultTestLocationName
	}
	if len(location) > maxLocationNameLen {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT",
			"location_name terlalu panjang")
		return
	}

	// event_id berawalan "test-" seperti drill: dedup perangkat tetap bekerja,
	// dan id yang muncul di log langsung mengaku dirinya uji.
	suffix, err := randomUserID()
	if err != nil {
		s.log.Error("gagal membuat id bench", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "gagal membuat id bench")
		return
	}
	eventID := "test-" + suffix

	mmi, label := consensus.Intensity(req.PGAGal)

	s.log.Warn("prelim latihan uji diminta operator",
		"event_id", eventID, "mmi", mmi, "pga_gal", req.PGAGal,
		"ip", clientIP(r))

	outcome, reason := s.prelimBench(r.Context(), PrelimTestBenchInput{
		EventID:      eventID,
		PGAGal:       req.PGAGal,
		CentroidLat:  *req.Latitude,
		CentroidLon:  *req.Longitude,
		LocationName: location,
		Timestamp:    time.Now().UnixMilli(),
	})

	s.log.Info("prelim latihan uji selesai",
		"event_id", eventID, "outcome", outcome, "reason", reason)

	resp := PrelimTestBenchResponse{
		EventID:        eventID,
		Type:           "EARTHQUAKE_ALERT",
		MMI:            mmi,
		IntensityLabel: label,
		IsTest:         true,
		Outcome:        outcome,
		Reason:         reason,
	}
	if outcome == prelimBenchOutcomeInactive {
		// Mode OFF/fail-closed menjawab 503: bench yang "diterima tapi tidak
		// terpasang" menipu operator (analisis sama dengan drill).
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}
	writeJSON(w, http.StatusOK, resp) // raised & refused sama-sama 200
}
