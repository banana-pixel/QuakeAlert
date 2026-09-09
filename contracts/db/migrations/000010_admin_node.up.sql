-- ============================================================================
-- QuakeAlert — Migration 000010 (UP): Kapabilitas Admin Node (D-036, PROPOSED)
--
-- Satu kolom baru di iot_nodes (is_admin_node) plus satu indeks unik parsial
-- yang memastikan MAKSIMUM SATU baris bernilai TRUE pada deployment saat ini.
-- Seluruhnya ADITIF: ADD COLUMN IF NOT EXISTS dengan DEFAULT FALSE, CREATE
-- UNIQUE INDEX IF NOT EXISTS — biner pra-000010 tetap berjalan di atas skema
-- ini tanpa menyadarinya, dan migrasi aman dijalankan dua kali maupun pada
-- basis data berisi data.
--
-- Kolom, dan mengapa bentuknya begini:
--
--   is_admin_node — BOOLEAN NOT NULL DEFAULT FALSE. Kemampuan eksplisit
--       operator, BUKAN pengganti verifikasi: designate mensyaratkan node sudah
--       verified (migrasi 000005), dan unverify otomatis mencabutnya kembali
--       (lihat SetNodeVerified di internal/store). FALSE adalah default yang
--       aman: migrasi ini TIDAK menunjuk siapa pun — termasuk NODE-52960B47 —
--       dan kapabilitas nonaktif sampai designate eksplisit lewat endpoint admin.
--
-- Indeks, dan mengapa unik parsial:
--
--   uq_iot_nodes_single_admin — UNIQUE atas (is_admin_node) dengan predikat
--       WHERE is_admin_node, sehingga hanya baris TRUE yang masuk indeks dan
--       dua baris TRUE tidak dapat berdampingan. Inilah yang membuat designate
--       race-safe: dua designate konkuren untuk node berbeda diserialisasi
--       Postgres pada indeks ini, tepat satu yang menang dan yang kalah
--       menerima unique-violation yang dipetakan ke 409 ADMIN_NODE_EXISTS.
--       Batas "satu" adalah keputusan deployment saat ini (D-036); ekspansi
--       multi-region kelak mengganti predikat/indeks ini, bukan skema kolomnya.
--
-- Yang SENGAJA tidak ada di migrasi ini:
--
--   - TIDAK ada designate awal untuk node mana pun (OFF by default).
--   - TIDAK ada FOREIGN KEY baru dan TIDAK ada tabel audit baru: auditability
--     designate/revoke mengikuti konvensi repo — baris log terstruktur
--     (station_id + hasil, tanpa koordinat), sama seperti verifikasi node.
--   - TIDAK ada perubahan pada kolom verified, is_active, atau ambang
--     konsensus mana pun.
--
-- Aditif & idempoten: aman dijalankan pada basis data berisi data, dan aman
-- dijalankan dua kali.
-- ============================================================================

ALTER TABLE iot_nodes ADD COLUMN IF NOT EXISTS is_admin_node BOOLEAN NOT NULL DEFAULT FALSE;

-- Hanya baris TRUE yang masuk indeks; dua TRUE tidak dapat berdampingan.
CREATE UNIQUE INDEX IF NOT EXISTS uq_iot_nodes_single_admin
    ON iot_nodes (is_admin_node)
    WHERE is_admin_node;
