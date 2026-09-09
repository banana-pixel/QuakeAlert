package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// --- Integrasi Postgres untuk kapabilitas Admin Node (migrasi 000010, D-036)
// ---
//
// Butuh Postgres NYATA: yang diuji adalah perilaku SKEMA dan KUERI-nya (kolom
// DEFAULT, indeks unik parsial sebagai penegak maksimum-satu, predikat
// verified atomik dalam UPDATE, pembersihan otomatis saat unverify) — bukan
// perilaku Go. Tanpa env TEST_DATABASE_URL seluruh test di berkas ini skip,
// pola yang sama dengan berkas integrasi lain.
//
// Basis data uji BERSAMA antar berkas dan antar paket: seluruh station_id di
// sini berprefix NODE-AD agar tidak bertabrakan dengan berkas lain, dan setiap
// baris yang dibuat dihapus kembali lewat Cleanup.
//
// CATATAN PARALELISME (kelas yang sama dengan TestMigration000009*): uji down
// di bawah melepas kolom is_admin_node sesaat. Paket lain tidak membaca kolom
// itu, tetapi seperti saudaranya ia harus berjalan dengan `-p 1` bila seluruh
// suite dijalankan bersamaan.

const (
	adNodeA     = "NODE-AD000001" // terverifikasi, kandidat designate
	adNodeB     = "NODE-AD000002" // terverifikasi, kandidat designate kedua
	adNodePend  = "NODE-AD000003" // pending (belum terverifikasi)
	adNodeRaceA = "NODE-AD000011" // designate konkuren, sisi A
	adNodeRaceB = "NODE-AD000012" // designate konkuren, sisi B
	adNodeSolo  = "NODE-AD000013" // designate konkuren satu-node (idempoten)
	adNodeRe    = "NODE-AD000021" // restart/persistence
	adNodeFresh = "NODE-AD000031" // status: denyut segar
	adNodeOld   = "NODE-AD000032" // status: denyut 10 menit lalu
	adNodeNull  = "NODE-AD000033" // status: last_heartbeat NULL
	adUnknown   = "NODE-AD009999" // tidak pernah dibuat
)

// isAdminNode membaca flag is_admin_node satu baris — jalur baca mentah untuk
// membuktikan state kolom, bukan perilaku helper produksi.
func isAdminNode(t *testing.T, st *Store, stationID string) bool {
	t.Helper()
	var v bool
	if err := st.pool.QueryRow(context.Background(),
		`SELECT is_admin_node FROM iot_nodes WHERE station_id = $1`, stationID).Scan(&v); err != nil {
		t.Fatalf("baca is_admin_node %s: %v", stationID, err)
	}
	return v
}

// countAdminNodes menghitung baris TRUE global — invarian maksimum-satu.
func countAdminNodes(t *testing.T, st *Store) int {
	t.Helper()
	var n int
	if err := st.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM iot_nodes WHERE is_admin_node`).Scan(&n); err != nil {
		t.Fatalf("hitung admin node: %v", err)
	}
	return n
}

// 000010 dijalankan DUA KALI harus menjadi no-op, diterapkan pada basis data
// yang skemanya sudah dimigrasi. Bila ADD COLUMN / CREATE INDEX kehilangan
// IF NOT EXISTS, penerapan kedua gagal di sini alih-alih di produksi.
func TestMigration000010IsIdempotent(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	up, err := os.ReadFile(filepath.Join("..", "..", "..",
		"contracts", "db", "migrations", "000010_admin_node.up.sql"))
	if err != nil {
		t.Fatalf("baca migrasi: %v", err)
	}
	for i := 1; i <= 2; i++ {
		if _, err := st.pool.Exec(ctx, string(up)); err != nil {
			t.Fatalf("penerapan migrasi ke-%d gagal (idempotensi rusak): %v", i, err)
		}
	}

	var col, idx int
	if err := st.pool.QueryRow(ctx,
		`SELECT count(*) FROM information_schema.columns
		 WHERE table_schema = 'public' AND table_name = 'iot_nodes'
		   AND column_name = 'is_admin_node'`).Scan(&col); err != nil {
		t.Fatalf("periksa kolom: %v", err)
	}
	if col != 1 {
		t.Errorf("kolom is_admin_node ditemukan %d kali, mau 1", col)
	}
	if err := st.pool.QueryRow(ctx,
		`SELECT count(*) FROM pg_indexes
		 WHERE schemaname = 'public' AND indexname = 'uq_iot_nodes_single_admin'`).Scan(&idx); err != nil {
		t.Fatalf("periksa indeks: %v", err)
	}
	if idx != 1 {
		t.Errorf("indeks uq_iot_nodes_single_admin ditemukan %d kali, mau 1", idx)
	}
}

// § rollback lengkap: down setelah up mengembalikan skema seperti sebelumnya —
// kolom dan indeks hilang — dan pemulihannya cukup menerapkan ulang berkas
// up-nya sendiri di Cleanup, karena 000010 yang terakhir dan tidak menyentuh
// tabel lain.
func TestMigration000010DownRestoresSchema(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	dir := filepath.Join("..", "..", "..", "contracts", "db", "migrations")
	down, err := os.ReadFile(filepath.Join(dir, "000010_admin_node.down.sql"))
	if err != nil {
		t.Fatalf("baca migrasi down: %v", err)
	}
	up, err := os.ReadFile(filepath.Join(dir, "000010_admin_node.up.sql"))
	if err != nil {
		t.Fatalf("baca migrasi up: %v", err)
	}

	t.Cleanup(func() {
		if _, err := st.pool.Exec(ctx, string(up)); err != nil {
			t.Fatalf("gagal menerapkan ulang 000010 setelah rollback: %v", err)
		}
	})

	if _, err := st.pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("rollback gagal: %v", err)
	}

	var col, idx int
	if err := st.pool.QueryRow(ctx,
		`SELECT count(*) FROM information_schema.columns
		 WHERE table_schema = 'public' AND table_name = 'iot_nodes'
		   AND column_name = 'is_admin_node'`).Scan(&col); err != nil {
		t.Fatalf("periksa kolom setelah rollback: %v", err)
	}
	if col != 0 {
		t.Errorf("kolom is_admin_node masih ada setelah rollback (%d)", col)
	}
	if err := st.pool.QueryRow(ctx,
		`SELECT count(*) FROM pg_indexes
		 WHERE schemaname = 'public' AND indexname = 'uq_iot_nodes_single_admin'`).Scan(&idx); err != nil {
		t.Fatalf("periksa indeks setelah rollback: %v", err)
	}
	if idx != 0 {
		t.Errorf("indeks uq_iot_nodes_single_admin masih ada setelah rollback (%d)", idx)
	}
}

// Default semua node = non-Admin: baris yang dibuat setelah migrasi lahir
// dengan FALSE, dan designate tidak terjadi dengan sendirinya. Perbandingan
// pemegang global sebelum/sesudah (bukan asumsi "kosong") agar uji tetap sah
// pada basis data bersama.
func TestAdminNode_DefaultIsNonAdmin(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	before, err := st.GetAdminNodeStationID(ctx)
	if err != nil {
		t.Fatalf("baca pemegang awal: %v", err)
	}

	st.seedNode(t, adNodeA, true, 24*time.Hour, noHeartbeat)
	if isAdminNode(t, st, adNodeA) {
		t.Fatal("node baru lahir sebagai admin: default harus FALSE")
	}

	after, err := st.GetAdminNodeStationID(ctx)
	if err != nil {
		t.Fatalf("baca pemegang akhir: %v", err)
	}
	if after != before {
		t.Fatalf("pemegang berubah %q -> %q tanpa designate", before, after)
	}
}

// Designate mensyaratkan verified: node terverifikasi berhasil, pending dan
// tak dikenal ditolak dengan sentinel yang berbeda.
func TestDesignateAdminNode_RequiresVerified(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	st.seedNode(t, adNodeA, true, 24*time.Hour, noHeartbeat)
	st.seedNode(t, adNodePend, false, 24*time.Hour, noHeartbeat)

	if err := st.DesignateAdminNode(ctx, adNodeA); err != nil {
		t.Fatalf("designate node terverifikasi: %v", err)
	}
	if got, _ := st.GetAdminNodeStationID(ctx); got != adNodeA {
		t.Fatalf("pemegang = %q, mau %q", got, adNodeA)
	}
	if _, err := st.RevokeAdminNode(ctx, adNodeA); err != nil {
		t.Fatalf("bersih revoke: %v", err)
	}

	if err := st.DesignateAdminNode(ctx, adNodePend); !errors.Is(err, ErrNodeNotVerified) {
		t.Fatalf("designate pending = %v, mau ErrNodeNotVerified", err)
	}
	if err := st.DesignateAdminNode(ctx, adUnknown); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("designate tak dikenal = %v, mau ErrNodeNotFound", err)
	}
	if isAdminNode(t, st, adNodePend) {
		t.Fatal("node pending menjadi admin lewat penolakan yang bocor")
	}
}

// Idempoten: designate ulang pemegang saat ini berhasil tanpa efek.
func TestDesignateAdminNode_Idempotent(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	st.seedNode(t, adNodeA, true, 24*time.Hour, noHeartbeat)
	for i := 1; i <= 2; i++ {
		if err := st.DesignateAdminNode(ctx, adNodeA); err != nil {
			t.Fatalf("designate ke-%d: %v", i, err)
		}
	}
	if got, _ := st.GetAdminNodeStationID(ctx); got != adNodeA {
		t.Fatalf("pemegang = %q, mau %q", got, adNodeA)
	}
	if n := countAdminNodes(t, st); n != 1 {
		t.Fatalf("baris TRUE = %d, mau 1", n)
	}
	if _, err := st.RevokeAdminNode(ctx, adNodeA); err != nil {
		t.Fatalf("bersih revoke: %v", err)
	}
}

// Pemegang kedua ditolak: maksimum satu ditegakkan indeks, bukan Go.
func TestDesignateAdminNode_RejectsSecondHolder(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	st.seedNode(t, adNodeA, true, 24*time.Hour, noHeartbeat)
	st.seedNode(t, adNodeB, true, 24*time.Hour, noHeartbeat)

	if err := st.DesignateAdminNode(ctx, adNodeA); err != nil {
		t.Fatalf("designate pertama: %v", err)
	}
	if err := st.DesignateAdminNode(ctx, adNodeB); !errors.Is(err, ErrAdminNodeExists) {
		t.Fatalf("designate kedua = %v, mau ErrAdminNodeExists", err)
	}
	if got, _ := st.GetAdminNodeStationID(ctx); got != adNodeA {
		t.Fatalf("pemegang berpindah ke %q setelah penolakan", got)
	}
	if _, err := st.RevokeAdminNode(ctx, adNodeA); err != nil {
		t.Fatalf("bersih revoke: %v", err)
	}
	// Setelah pemegang dicabut, kursi kosong dan designate berikutnya sah.
	if err := st.DesignateAdminNode(ctx, adNodeB); err != nil {
		t.Fatalf("designate setelah revoke: %v", err)
	}
	if _, err := st.RevokeAdminNode(ctx, adNodeB); err != nil {
		t.Fatalf("bersih revoke: %v", err)
	}
}

// Revoke idempoten dan membedakan tak dikenal: cabut-dua-kali tetap 200 di
// lapisan API (true,true), ID tak dikenal false tanpa galat.
func TestRevokeAdminNode_IdempotentAndUnknown(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	st.seedNode(t, adNodeA, true, 24*time.Hour, noHeartbeat)
	if err := st.DesignateAdminNode(ctx, adNodeA); err != nil {
		t.Fatalf("designate: %v", err)
	}

	for i := 1; i <= 2; i++ {
		known, err := st.RevokeAdminNode(ctx, adNodeA)
		if err != nil || !known {
			t.Fatalf("revoke ke-%d = (%v,%v), mau (true,nil)", i, known, err)
		}
	}
	if got, _ := st.GetAdminNodeStationID(ctx); got != "" {
		t.Fatalf("pemegang = %q setelah revoke, mau kosong", got)
	}

	known, err := st.RevokeAdminNode(ctx, adUnknown)
	if err != nil || known {
		t.Fatalf("revoke tak dikenal = (%v,%v), mau (false,nil)", known, err)
	}
}

// Unverify otomatis mencabut: SetNodeVerified(false) membersihkan flag dalam
// statement yang sama; verify ulang TIDAK mengembalikannya (tidak retroaktif).
func TestUnverifyClearsAdminNode(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	st.seedNode(t, adNodeA, true, 24*time.Hour, noHeartbeat)
	if err := st.DesignateAdminNode(ctx, adNodeA); err != nil {
		t.Fatalf("designate: %v", err)
	}

	ok, err := st.SetNodeVerified(ctx, adNodeA, false)
	if err != nil || !ok {
		t.Fatalf("unverify = (%v,%v), mau (true,nil)", ok, err)
	}
	if isAdminNode(t, st, adNodeA) {
		t.Fatal("is_admin_node masih TRUE setelah unverify")
	}
	if got, _ := st.GetAdminNodeStationID(ctx); got != "" {
		t.Fatalf("pemegang = %q setelah unverify, mau kosong", got)
	}

	ok, err = st.SetNodeVerified(ctx, adNodeA, true)
	if err != nil || !ok {
		t.Fatalf("verify ulang = (%v,%v), mau (true,nil)", ok, err)
	}
	if isAdminNode(t, st, adNodeA) {
		t.Fatal("verify ulang mengembalikan admin secara retroaktif")
	}
	// Designate ulang setelah verifikasi sah.
	if err := st.DesignateAdminNode(ctx, adNodeA); err != nil {
		t.Fatalf("designate ulang: %v", err)
	}
	if _, err := st.RevokeAdminNode(ctx, adNodeA); err != nil {
		t.Fatalf("bersih revoke: %v", err)
	}
}

// Konkuren: 8 designate A + 8 designate B — tepat satu pemegang di akhir, dan
// setiap goroutine hanya boleh melihat nil atau ErrAdminNodeExists (tidak ada
// galat ketiga, tidak ada dua TRUE).
func TestDesignateAdminNode_ConcurrentAllowsExactlyOne(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	st.seedNode(t, adNodeRaceA, true, 24*time.Hour, noHeartbeat)
	st.seedNode(t, adNodeRaceB, true, 24*time.Hour, noHeartbeat)

	const perSide = 8
	errs := make(chan error, 2*perSide)
	var wg sync.WaitGroup
	for i := 0; i < perSide; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); errs <- st.DesignateAdminNode(ctx, adNodeRaceA) }()
		go func() { defer wg.Done(); errs <- st.DesignateAdminNode(ctx, adNodeRaceB) }()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil && !errors.Is(err, ErrAdminNodeExists) {
			t.Fatalf("galat konkuren tak terduga: %v", err)
		}
	}
	if n := countAdminNodes(t, st); n != 1 {
		t.Fatalf("baris TRUE = %d setelah designate konkuren, mau tepat 1", n)
	}
	got, err := st.GetAdminNodeStationID(ctx)
	if err != nil {
		t.Fatalf("baca pemegang: %v", err)
	}
	if got != adNodeRaceA && got != adNodeRaceB {
		t.Fatalf("pemegang = %q, mau salah satu kandidat", got)
	}
	if _, err := st.RevokeAdminNode(ctx, got); err != nil {
		t.Fatalf("bersih revoke: %v", err)
	}

	// Satu-node konkuren idempoten: 8 designate node yang sama, semua nil.
	st.seedNode(t, adNodeSolo, true, 24*time.Hour, noHeartbeat)
	errs2 := make(chan error, perSide)
	for i := 0; i < perSide; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs2 <- st.DesignateAdminNode(ctx, adNodeSolo) }()
	}
	wg.Wait()
	close(errs2)
	for err := range errs2 {
		if err != nil {
			t.Fatalf("designate satu-node konkuren = %v, mau nil", err)
		}
	}
	if _, err := st.RevokeAdminNode(ctx, adNodeSolo); err != nil {
		t.Fatalf("bersih revoke: %v", err)
	}
}

// Restart: designation selamat dari pergantian proses. Tidak ada state admin
// di memori mana pun, jadi "restart" di sini adalah pool baru — dan baris
// harus tetap ada. Pembersihan eksplisit lewat pool baru karena Cleanup bawaan
// seedNode terikat pada pool lama yang sudah ditutup.
func TestAdminNode_PersistsAcrossRestart(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	st.seedNode(t, adNodeRe, true, 24*time.Hour, noHeartbeat)
	if err := st.DesignateAdminNode(ctx, adNodeRe); err != nil {
		t.Fatalf("designate: %v", err)
	}
	st.Close()

	st2 := newTestStore(t)
	got, err := st2.GetAdminNodeStationID(ctx)
	if err != nil {
		t.Fatalf("baca pemegang setelah restart: %v", err)
	}
	if got != adNodeRe {
		t.Fatalf("pemegang setelah restart = %q, mau %q", got, adNodeRe)
	}

	if _, err := st2.pool.Exec(ctx,
		`DELETE FROM iot_nodes WHERE station_id = $1`, adNodeRe); err != nil {
		t.Fatalf("bersih hapus baris: %v", err)
	}
}

// GetAdminNodeStatus membaca designation + verified + umur denyut dalam satu
// baris: denyut segar berumur detik, denyut 10 menit lalu berumur menit,
// heartbeat NULL gagal-tertutup (basi), dan tanpa designate designated=false
// tanpa galat.
func TestGetAdminNodeStatus(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	fresh := time.Now()
	st.seedNode(t, adNodeFresh, true, 24*time.Hour, &fresh)
	old := time.Now().Add(-10 * time.Minute)
	st.seedNode(t, adNodeOld, true, 24*time.Hour, &old)

	if _, designated, err := st.GetAdminNodeStatus(ctx); err != nil || designated {
		t.Fatalf("status tanpa designate = (%v,%v), mau (false,nil)", designated, err)
	}

	if err := st.DesignateAdminNode(ctx, adNodeFresh); err != nil {
		t.Fatalf("designate: %v", err)
	}
	row, designated, err := st.GetAdminNodeStatus(ctx)
	if err != nil || !designated {
		t.Fatalf("status = (%v,%v), mau (true,nil)", designated, err)
	}
	if row.StationID != adNodeFresh || !row.Verified {
		t.Fatalf("baris = %+v, mau {fresh true}", row)
	}
	if row.HeartbeatAge < 0 || row.HeartbeatAge > 5*time.Minute {
		t.Fatalf("umur denyut segar = %v, mau detik-orde", row.HeartbeatAge)
	}
	if _, err := st.RevokeAdminNode(ctx, adNodeFresh); err != nil {
		t.Fatalf("bersih revoke: %v", err)
	}

	if err := st.DesignateAdminNode(ctx, adNodeOld); err != nil {
		t.Fatalf("designate: %v", err)
	}
	row, _, err = st.GetAdminNodeStatus(ctx)
	if err != nil {
		t.Fatalf("baca status basi: %v", err)
	}
	if row.HeartbeatAge < 10*time.Minute {
		t.Fatalf("umur denyut 10-menit = %v, mau >= 10m", row.HeartbeatAge)
	}
	if _, err := st.RevokeAdminNode(ctx, adNodeOld); err != nil {
		t.Fatalf("bersih revoke: %v", err)
	}
}

// Heartbeat NULL tidak pernah terbaca segar: kolomnya nullable di DDL walau
// penulis produksi selalu mengisinya, jadi pembaca gagal-tertutup ke basi.
func TestGetAdminNodeStatus_NullHeartbeatIsStale(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	if _, err := st.pool.Exec(ctx, `
		INSERT INTO iot_nodes (
			station_id, sensor_model, location_name, location,
			secret_key_enc, secret_key_nonce, verified, last_heartbeat
		) VALUES (
			$1, 'MPU 6050', 'Cimahi',
			ST_SetSRID(ST_MakePoint(107.54, -6.87), 4326)::geography,
			'\x00'::bytea, '\x00'::bytea, TRUE, NULL
		)`, adNodeNull); err != nil {
		t.Fatalf("seed NULL heartbeat: %v", err)
	}
	t.Cleanup(func() {
		_, _ = st.pool.Exec(ctx, `DELETE FROM iot_nodes WHERE station_id = $1`, adNodeNull)
	})

	if err := st.DesignateAdminNode(ctx, adNodeNull); err != nil {
		t.Fatalf("designate: %v", err)
	}
	row, designated, err := st.GetAdminNodeStatus(ctx)
	if err != nil || !designated {
		t.Fatalf("status = (%v,%v), mau (true,nil)", designated, err)
	}
	if row.HeartbeatAge < time.Hour {
		t.Fatalf("umur heartbeat NULL = %v, mau gagal-tertutup (>= 1 jam)", row.HeartbeatAge)
	}
	if _, err := st.RevokeAdminNode(ctx, adNodeNull); err != nil {
		t.Fatalf("bersih revoke: %v", err)
	}
}
