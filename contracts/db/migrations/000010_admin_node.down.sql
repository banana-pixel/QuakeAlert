-- ============================================================================
-- QuakeAlert — Migration 000010 (DOWN): Batalkan kapabilitas Admin Node
--
-- Melepas indeks unik parsial lalu kolomnya, dalam urutan itu. Tidak ada baris
-- tabel lain yang tersentuh — karena migrasi 000010 (UP) tidak mengubah apa pun
-- di luar kolom dan indeksnya sendiri.
--
-- URUTAN ROLLBACK YANG BENAR (kode LEBIH DULU, lalu skema):
--
--   1. Turunkan biner ke versi pra-D-036, ATAU jalankan biner D-036 dengan basis
--      data yang masih memiliki kolom ini.
--   2. Baru jalankan migrasi turun ini.
--
-- Alasannya: biner D-036 yang berjalan di atas skema yang sudah diturunkan akan
-- gagal pada SETIAP designate/revoke dan pada SETIAP pembacaan designation.
-- Kegagalan itu TERBATAS pada administrasi — ia TIDAK dapat menghalangi emisi
-- peringatan (S1, §9.5): jalur ingest, Tracker lima-state, dispatch normal, dan
-- seluruh lifecycle event tidak membaca kolom ini. Yang hilang hanyalah
-- kemampuan menunjuk/mencabut Admin Node sampai skema dinaikkan kembali.
--
-- YANG HILANG PERMANEN: designation yang sedang aktif — satu station_id yang
-- sedang memegang is_admin_node = TRUE. Setelah rollback tidak ada catatan di
-- tabel mana pun tentang siapa yang pernah ditunjuk; designate ulang eksplisit
-- diperlukan setelah migrate up berikutnya.
--
-- Idempoten: aman dijalankan dua kali.
-- ============================================================================

DROP INDEX IF EXISTS uq_iot_nodes_single_admin;
ALTER TABLE iot_nodes DROP COLUMN IF EXISTS is_admin_node;
