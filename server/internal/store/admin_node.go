package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// --- Kapabilitas Admin Node (migrasi 000010, D-036 PROPOSED) ---
//
// is_admin_node adalah kemampuan eksplisit operator di atas satu node yang
// sudah terverifikasi: designate mensyaratkan verified = TRUE, dan unverify
// otomatis mencabutnya (lihat SetNodeVerified di store.go). Maksimum SATU
// baris TRUE pada deployment saat ini, ditegakkan indeks unik parsial
// uq_iot_nodes_single_admin — bukan oleh Go — sehingga dua designate konkuren
// diserialisasi Postgres dan tepat satu yang menang.
//
// Tidak ada koordinat yang dibaca maupun ditulis di berkas ini: designate dan
// revoke bekerja murni pada station_id + flag.

// ErrNodeNotVerified dikembalikan DesignateAdminNode bila station_id dikenal
// tetapi node-nya belum terverifikasi (migrasi 000005). API memetakannya ke
// HTTP 409: kepercayaan (verified) harus didapat dulu lewat endpoint verify,
// tidak boleh dilompati lewat designate.
var ErrNodeNotVerified = errors.New("node belum terverifikasi")

// ErrAdminNodeExists dikembalikan DesignateAdminNode bila node LAIN sudah
// memegang is_admin_node = TRUE. API memetakannya ke HTTP 409: cabut dulu
// pemegang saat ini sebelum menunjuk yang baru. Inilah permukaan Go dari
// indeks uq_iot_nodes_single_admin — termasuk untuk designate konkuren, di
// mana yang kalah balapan menerima unique-violation ini.
var ErrAdminNodeExists = errors.New("sudah ada Admin Node aktif")

// DesignateAdminNode menunjuk satu node terverifikasi sebagai Admin Node.
//
// Predikat verified ada DI DALAM statement UPDATE yang sama (atomik terhadap
// unverify konkuren): designate yang balapan dengan unverify kalah secara
// benar — tidak pernah ditunjuk — dan designate yang balapan dengan designate
// lain untuk node berbeda diselesaikan indeks unik (satu menang, sisanya
// ErrAdminNodeExists).
//
// Idempoten: menunjuk ulang pemegang saat ini berhasil tanpa efek (UPDATE
// TRUE-ke-TRUE tetap menyentuh satu baris). OFF by default dipertahankan di
// lapisan migrasi: tidak ada pemanggil di sini yang menunjuk siapa pun tanpa
// perintah operator eksplisit.
func (s *Store) DesignateAdminNode(ctx context.Context, stationID string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE iot_nodes SET is_admin_node = TRUE WHERE station_id = $1 AND verified = TRUE`,
		stationID)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: %s", ErrAdminNodeExists, stationID)
		}
		return fmt.Errorf("designate admin node: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}

	// Nol baris tersentuh: tidak dikenal ATAU belum terverifikasi. SELECT
	// lanjutan hanya berjalan di jalur penolakan (bukan jalur panas) untuk
	// membedakan keduanya. Selalu tepat satu baris hasil (EXISTS +
	// COALESCE), jadi tidak ada ErrNoRows yang perlu ditangani.
	var exists, verified bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM iot_nodes WHERE station_id = $1),
		       COALESCE((SELECT verified FROM iot_nodes WHERE station_id = $1), FALSE)`,
		stationID).Scan(&exists, &verified); err != nil {
		return fmt.Errorf("cek kelayakan admin node: %w", err)
	}
	if !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, stationID)
	}
	if !verified {
		return fmt.Errorf("%w: %s", ErrNodeNotVerified, stationID)
	}
	// Baris dikenal dan terverifikasi tetapi UPDATE tidak menyentuh apa pun:
	// hanya mungkin bila verified berubah tepat di antara kedua statement.
	// Tolak ke arah aman (jangan tunjuk) dan laporkan sebagai galat internal.
	return fmt.Errorf("designate admin node %s: baris terverifikasi tidak ter-update", stationID)
}

// RevokeAdminNode mencabut kapabilitas Admin Node satu node.
//
// Idempoten: mencabut node yang memang bukan admin tetap berhasil — tidak ada
// keadaan "sudah dicabut" yang perlu dibedakan. Mengembalikan false bila
// station_id tidak dikenal (pemanggil API memetakannya ke 404, mengikuti
// konvensi SetNodeVerified: ID yang salah adalah kesalahan operator).
func (s *Store) RevokeAdminNode(ctx context.Context, stationID string) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE iot_nodes SET is_admin_node = FALSE WHERE station_id = $1`, stationID)
	if err != nil {
		return false, fmt.Errorf("revoke admin node: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM iot_nodes WHERE station_id = $1)`,
		stationID).Scan(&exists); err != nil {
		return false, fmt.Errorf("cek keberadaan node: %w", err)
	}
	return exists, nil
}

// GetAdminNodeStationID mengembalikan station_id pemegang is_admin_node = TRUE,
// atau "" bila tidak ada yang ditunjuk (keadaan default dan yang diharapkan
// sebelum designate pertama). Selalu tepat satu baris hasil; tidak pernah
// mengembalikan galat "tidak ada baris".
//
// Tidak ada state admin di memori proses mana pun: designation hidup di kolom
// basis data dan dibaca lewat fungsi ini (atau predikat SQL langsung), sehingga
// restart server mempertahankannya tanpa logika pemuatan ulang.
func (s *Store) GetAdminNodeStationID(ctx context.Context) (string, error) {
	var id string
	if err := s.pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT station_id FROM iot_nodes WHERE is_admin_node LIMIT 1), '')`,
	).Scan(&id); err != nil {
		return "", fmt.Errorf("baca admin node: %w", err)
	}
	return id, nil
}

// AdminNodeStatus adalah baris designation + verifikasi + kesegaran denyut
// pemegang is_admin_node, dibaca dalam SATU query. Dipakai gerbang kelayakan
// peringatan lokal (D-036): designation, verified, dan umur heartbeat harus
// berasal dari pembacaan yang sama agar tidak ada keputusan atas campuran dua
// waktu yang berbeda.
type AdminNodeStatus struct {
	StationID string
	Verified  bool
	// HeartbeatAge adalah NOW() - last_heartbeat saat dibaca. Heartbeat yang
	// hilang terbaca sebagai setahun (gagal-tertutup: basi, tidak pernah
	// segar), dan umur negatif (jam node di depan jam server) dijepit ke nol.
	HeartbeatAge time.Duration
}

// GetAdminNodeStatus membaca status pemegang Admin Node, atau designated=false
// bila tidak ada yang ditunjuk. Selalu gagal-tertutup ke arah "tidak layak":
// tanpa baris, tanpa denyut, atau tanpa jawaban basis data tidak ada
// peringatan lokal — dan tidak ada pengaruh apa pun pada jalur normal.
func (s *Store) GetAdminNodeStatus(ctx context.Context) (AdminNodeStatus, bool, error) {
	var st AdminNodeStatus
	var hb *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT station_id, verified, last_heartbeat
		FROM iot_nodes WHERE is_admin_node LIMIT 1`,
	).Scan(&st.StationID, &st.Verified, &hb)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminNodeStatus{}, false, nil
		}
		return AdminNodeStatus{}, false, fmt.Errorf("baca status admin node: %w", err)
	}
	if hb == nil {
		st.HeartbeatAge = 365 * 24 * time.Hour
	} else {
		st.HeartbeatAge = time.Since(*hb)
		if st.HeartbeatAge < 0 {
			st.HeartbeatAge = 0
		}
	}
	return st, true, nil
}
