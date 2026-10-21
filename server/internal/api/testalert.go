// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/banana-pixel/quakealert/server/internal/consensus"
	"github.com/banana-pixel/quakealert/server/internal/store"
)

// Batas PGA yang diterima sebuah drill. Bukan batas fisika, melainkan batas
// kewajaran: nilai di luar rentang ini hampir selalu salah ketik pada skrip
// (satuan g alih-alih gal, atau nol yang terlewat), dan sebuah drill yang
// mengaku MMI XII akan mengukur ketenangan penguji, bukan aplikasinya.
const (
	minTestPGAGal = 1.0
	maxTestPGAGal = 2000.0
)

// defaultTestNodeCount adalah jumlah node yang dilaporkan sebuah drill bila
// tidak disebutkan: 3, ambang CONFIRMED, karena itulah bentuk peringatan yang
// sedang dilatih.
const defaultTestNodeCount = 3

// maxTestNodeCount menjaga angka yang dirender klien tetap masuk akal.
const maxTestNodeCount = 99

// defaultTestLocationName dipakai bila operator tidak menyebut lokasi. Sengaja
// menyebut dirinya latihan: kalaupun sebuah build lolos kedua pagar dan
// menampilkannya, teksnya sendiri masih mengatakan apa itu.
const defaultTestLocationName = "LATIHAN — bukan gempa sungguhan"

// TestAlert adalah peringatan latihan yang siap difanout.
//
// Didefinisikan di paket api dengan alasan yang sama seperti ChatEvent dan
// AdminBroadcast: api dan dispatch tidak saling impor, dan cmd/quakealert yang
// menjembatani keduanya.
type TestAlert struct {
	EventID        string
	MMI            string
	IntensityLabel string
	PGAGal         float64
	Latitude       float64
	Longitude      float64
	LocationName   string
	NodeCount      int
	Timestamp      time.Time
	// TargetToken, hanya terisi bila operator meminta drill bertarget
	// (target_pseudonym): token FCM satu instalasi, dibaca session dari
	// store. Berarti dispatchTestFCM mengirim ke token itu ALIH-ALIH topic
	// test_alerts — eksklusif, tanpa topic, tanpa fallback apa pun. Dikosongkan
	// pada drill biasa. Token TIDAK boleh sampai ke log maupun respons —
	// perangkat target tidak perlu ikut dikenali dari luar.
	TargetToken string
	// TargetPseudonym adalah pseudonim yang diminta operator — untuk log dan
	// respons, bukan untuk target kirim (kirim memakai Token).
	TargetPseudonym string
}

// PseudonymTokenFinder membaca token FCM satu profil anonim dari store, untuk
// drill bertarget per perangkat. Read-only penuh: implementasinya hanya
// SELECT, dan token kembaliannya boleh menuju FCM — bukan log, bukan respons.
type PseudonymTokenFinder interface {
	FcmTokenByPseudonym(ctx context.Context, pseudonym string) (string, error)
}

// maxTargetPseudonymLen menahan input operator di kertas: pseudonim tersimpan
// VARCHAR(32), jadi apa pun yang lebih panjang pasti salah ketik dan tidak
// perlu menyentuh database.
const maxTargetPseudonymLen = 64

// TestAlertFanout menyiarkan peringatan latihan. Opsional seperti
// BroadcastFanout: tanpanya endpoint-nya menjawab 503 alih-alih diam-diam
// menerima drill yang tidak akan sampai ke mana pun.
type TestAlertFanout interface {
	DispatchTestAlert(TestAlert)
}

// SetTestAlertFanout memasang jalur drill. Setter, bukan parameter NewServer,
// karena dispatcher dibangun setelah Server di cmd/quakealert.
func (s *Server) SetTestAlertFanout(f TestAlertFanout) {
	s.testAlerts = f
}

// SetPseudonymTokenFinder memasang pembaca token untuk drill bertarget.
// Setter, bukan parameter NewServer, dengan alasan yang sama dengan
// SetTestAlertFanout. nil (tidak dipasang) membiarkan drill biasa tetap hidup;
// hanya permintaan target yang menjawab 503.
func (s *Server) SetPseudonymTokenFinder(f PseudonymTokenFinder) {
	s.pseudonymTokens = f
}

type createTestAlertRequest struct {
	// PGAGal adalah satu-satunya masukan intensitas: MMI dan intensity_label
	// diturunkan darinya oleh fungsi yang SAMA dengan jalur konsensus, sehingga
	// sebuah drill tidak dapat membawa kombinasi yang gempa sungguhan tidak
	// pernah menghasilkan (mis. MMI VIII berlabel "light").
	PGAGal float64 `json:"pga_gal"`
	// Pointer agar 0,0 yang disengaja dapat dibedakan dari field yang absen —
	// alasan yang sama dengan updateLocationRequest.
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	LocationName string   `json:"location_name"`
	NodeCount    int      `json:"node_count"`
	// TargetPseudonym opsional: bila terisi, drill ini TIDAK masuk topic
	// test_alerts — ia dikirim tepat satu token FCM milik profil pseudonim itu
	// (sekali, tanpa fallback topic/broadcast/jalur alert nyata). Dipakai untuk
	// penguji end-to-end ketika subscription topik perangkat tidak dapat
	// divalidasi dari luar (mis. tanpa adb). Pseudonim tidak rahasia (identity
	// anonim yang dilihat pengguna sendiri), tetapi tokennya: token hanya
	// berpindah store→FCM, tidak pernah ke log/http-log maupun body respons.
	TargetPseudonym string `json:"target_pseudonym"`
}

type testAlertResponse struct {
	EventID        string `json:"event_id"`
	Type           string `json:"type"`
	MMI            string `json:"mmi"`
	IntensityLabel string `json:"intensity_label"`
	IsTest         bool   `json:"is_test"`
	Topic          string `json:"topic"`
	// Target mengidentifikasi mode pengiriman: kosong = topic test_alerts
	// (drill biasa); terisi = pseudonim target drill bertarget. Token TIDAK
	// PERNAH masuk respons — operator mengenal target dari pseudonimnya.
	Target string `json:"target,omitempty"`
}

// HandleCreateTestAlert menyiarkan sebuah peringatan LATIHAN.
//
// Tidak melewati internal/consensus dan tidak menulis satu baris pun ke
// earthquake_events. Keduanya disengaja dan keduanya penting: drill yang
// tersimpan akan muncul di riwayat aktivitas pengguna dan menggeser hitungan 30
// hari, dan riwayat gempa yang memuat gempa yang tidak pernah terjadi tidak
// dapat dipercaya untuk apa pun — termasuk untuk mengevaluasi sistem ini
// sendiri. Drill yang masuk ke engine, lebih buruk lagi, akan memakai cooldown
// dan slot resolusi yang mungkin dibutuhkan gempa sungguhan beberapa detik
// kemudian.
//
// Penargetannya ada di dispatch.DispatchTestAlert: topic test_alerts saja, yang
// hanya dilanggani build debug, tanpa fallback ke topic nasional.
func (s *Server) HandleCreateTestAlert(w http.ResponseWriter, r *http.Request) {
	var req createTestAlertRequest
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
				"jadi drill tanpa koordinat tidak menguji apa pun")
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

	nodeCount := req.NodeCount
	if nodeCount <= 0 {
		nodeCount = defaultTestNodeCount
	}
	if nodeCount > maxTestNodeCount {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT",
			"node_count maksimal 99")
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

	if s.testAlerts == nil {
		// 503, bukan 200: sebuah drill yang diterima tetapi tidak dikirim ke mana
		// pun adalah cara terburuk untuk mengetahui bahwa jalur push mati.
		writeError(w, http.StatusServiceUnavailable, "UNAVAILABLE",
			"jalur siaran belum terpasang di server ini")
		return
	}

	// Target opsional: drill bertarget satu token (tanpa topic sama sekali).
	targetPseudonym := strings.TrimSpace(req.TargetPseudonym)
	targetToken := ""
	if targetPseudonym != "" {
		if len(targetPseudonym) > maxTargetPseudonymLen {
			writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT",
				"target_pseudonym terlalu panjang")
			return
		}
		if s.pseudonymTokens == nil {
			writeError(w, http.StatusServiceUnavailable, "UNAVAILABLE",
				"drill bertarget tidak terpasang di server ini")
			return
		}
		token, err := s.pseudonymTokens.FcmTokenByPseudonym(r.Context(), targetPseudonym)
		if errors.Is(err, store.ErrUserNotFound) {
			// 404 tanpa meng-echo pseudonim maupun bagian token: operator yang
			// salah mengetik cukup melihat nama request-nya.
			s.log.Warn("drill bertarget: pseudonim tidak dikenal atau tanpa token")
			writeError(w, http.StatusNotFound, "NOT_FOUND",
				"target_pseudonym tidak dikenal (atau profil tanpa token FCM)")
			return
		}
		if err != nil {
			// err internal dari store: bisa memuat teks query, tidak pernah
			// memuat nilai token — aman untuk log, alasannya generic di body.
			s.log.Warn("drill bertarget: pembacaan token gagal", "err", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL",
				"gagal membaca target drill")
			return
		}
		targetToken = token
	}

	// event_id bukan UUID dari basis data — tidak ada barisnya — melainkan
	// diberi awalan yang tidak mungkin bertabrakan dengan gen_random_uuid().
	// Dedup di klien tetap bekerja, dan sebuah id yang terbaca di log atau
	// laporan bug langsung mengaku dirinya latihan.
	suffix, err := randomUserID()
	if err != nil {
		s.log.Error("gagal membuat id drill", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "gagal membuat id drill")
		return
	}
	eventID := "test-" + suffix

	mmi, label := consensus.Intensity(req.PGAGal)

	logFields := []any{
		"event_id", eventID, "mmi", mmi, "pga_gal", req.PGAGal, "ip", clientIP(r),
	}
	if targetToken != "" {
		// Token TIDAK pernah masuk log — cukup pseudonimnya (identitas yang
		// sudah diintip pemiliknya di aplikasi).
		logFields = append(logFields, "target_pseudonym", targetPseudonym)
	}
	s.log.Warn("peringatan LATIHAN diminta operator", logFields...)

	s.testAlerts.DispatchTestAlert(TestAlert{
		EventID:         eventID,
		MMI:             mmi,
		IntensityLabel:  label,
		PGAGal:          req.PGAGal,
		Latitude:        *req.Latitude,
		Longitude:       *req.Longitude,
		LocationName:    location,
		NodeCount:       nodeCount,
		Timestamp:       time.Now().UTC(),
		TargetToken:     targetToken,
		TargetPseudonym: targetPseudonym,
	})

	resp := testAlertResponse{
		EventID:        eventID,
		Type:           "EARTHQUAKE_ALERT",
		MMI:            mmi,
		IntensityLabel: label,
		IsTest:         true,
	}
	if targetToken != "" {
		resp.Target = targetPseudonym
	} else {
		resp.Topic = "test_alerts"
	}
	writeJSON(w, http.StatusAccepted, resp)
}
