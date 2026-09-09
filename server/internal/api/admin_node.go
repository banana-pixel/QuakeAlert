package api

import (
	"errors"
	"net/http"

	"github.com/banana-pixel/quakealert/server/internal/store"
	"github.com/go-chi/chi/v5"
)

// --- Designasi Admin Node (migrasi 000010, D-036 PROPOSED) ---
//
// Rute aksi memakai POST seperti seluruh rute admin lain (broadcasts,
// test-alert, verify): tidak ada DELETE di grup admin, dan POST selaras dengan
// pola "aksi eksplisit operator" yang sudah ada. Auditability mengikuti
// konvensi repo — baris log terstruktur berisi station_id + hasil, tanpa
// koordinat — sama seperti verifikasi node (admin_nodes.go).

type adminNodeResponse struct {
	StationID   string `json:"station_id"`
	IsAdminNode bool   `json:"is_admin_node"`
}

// HandleDesignateAdminNode menunjuk satu node TERVERIFIKASI sebagai Admin Node.
//
// Syarat keberhasilan: station_id berpola benar, baris dikenal,
// verified = TRUE, dan belum ada node lain yang memegang kapabilitas
// (maksimum satu pada deployment saat ini, D-036). Idempoten: menunjuk ulang
// pemegang saat ini menjawab 200 tanpa efek. TIDAK menunjuk siapa pun secara
// otomatis — kapabilitas lahir hanya dari panggilan eksplisit ini.
func (s *Server) HandleDesignateAdminNode(w http.ResponseWriter, r *http.Request) {
	stationID := chi.URLParam(r, "stationID")
	if !stationIDPattern.MatchString(stationID) {
		writeError(w, http.StatusNotFound, "NODE_NOT_FOUND",
			"station_id harus berpola NODE-XXXXXXXX (hex kapital)")
		return
	}

	if err := s.repo.DesignateAdminNode(r.Context(), stationID); err != nil {
		switch {
		case errors.Is(err, store.ErrNodeNotFound):
			writeError(w, http.StatusNotFound, "NODE_NOT_FOUND", "station_id tidak ditemukan")
		case errors.Is(err, store.ErrNodeNotVerified):
			s.log.Warn("designate admin ditolak: node belum terverifikasi", "station_id", stationID)
			writeError(w, http.StatusConflict, "NODE_NOT_VERIFIED",
				"node harus terverifikasi sebelum ditunjuk sebagai Admin Node")
		case errors.Is(err, store.ErrAdminNodeExists):
			s.log.Warn("designate admin ditolak: sudah ada Admin Node lain", "station_id", stationID)
			writeError(w, http.StatusConflict, "ADMIN_NODE_EXISTS",
				"sudah ada Admin Node aktif; cabut dulu sebelum menunjuk yang baru")
		default:
			s.log.Error("gagal designate admin node", "station_id", stationID, "err", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "gagal menyimpan designasi")
		}
		return
	}

	s.log.Info("admin node ditunjuk", "station_id", stationID)
	writeJSON(w, http.StatusOK, adminNodeResponse{StationID: stationID, IsAdminNode: true})
}

// HandleRevokeAdminNode mencabut kapabilitas Admin Node satu node.
//
// Idempoten: mencabut node yang memang bukan admin tetap menjawab 200 dengan
// is_admin_node=false. station_id yang salah bentuk atau tidak dikenal
// menjawab 404 dan tidak pernah menyentuh basis data — mengikuti konvensi
// HandleVerifyNode.
func (s *Server) HandleRevokeAdminNode(w http.ResponseWriter, r *http.Request) {
	stationID := chi.URLParam(r, "stationID")
	if !stationIDPattern.MatchString(stationID) {
		writeError(w, http.StatusNotFound, "NODE_NOT_FOUND",
			"station_id harus berpola NODE-XXXXXXXX (hex kapital)")
		return
	}

	known, err := s.repo.RevokeAdminNode(r.Context(), stationID)
	if err != nil {
		s.log.Error("gagal revoke admin node", "station_id", stationID, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "gagal mencabut designasi")
		return
	}
	if !known {
		writeError(w, http.StatusNotFound, "NODE_NOT_FOUND", "station_id tidak ditemukan")
		return
	}

	s.log.Info("admin node dicabut", "station_id", stationID)
	writeJSON(w, http.StatusOK, adminNodeResponse{StationID: stationID, IsAdminNode: false})
}
