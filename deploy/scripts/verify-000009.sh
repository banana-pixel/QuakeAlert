#!/usr/bin/env bash
# verify-000009.sh — cek baca-saja sebelum/sesudah migrasi 000009 di VPS.
#
# 000009 (D-012) menambah tabel event_near_confirmed: aditif, idempoten, tanpa
# ALTER/DROP. Kosong itu VALID di fleet satu-node (S2) — yang dicek adalah
# keberadaan tabel + kolomnya, bukan isinya.
#
# Tidak menulis apa pun: hanya SELECT + \d lewat `docker exec` ke container
# postgis. Dijalankan di VPS produksi:
#
#   ./scripts/verify-000009.sh pre    # sebelum compose up --build
#   ./scripts/verify-000009.sh post   # sesudahnya
set -euo pipefail

cd "$(dirname "$0")/.."

PG="${PG_CONTAINER:-quakealert-postgis}"
MODE="${1:-}"

if [ "$MODE" != "pre" ] && [ "$MODE" != "post" ]; then
  echo "pakai: $0 pre|post" >&2
  exit 2
fi

q() {
  docker exec -i "$PG" psql -U quakealert -d quakealert -Atc "$1"
}

echo "== migrasi terpasang =="
q "SELECT version FROM schema_migrations ORDER BY version;"

echo "== tabel event_near_confirmed =="
TABLE_EXISTS="$(q "SELECT to_regclass('public.event_near_confirmed');")"
echo "to_regclass: ${TABLE_EXISTS:-NULL}"

if [ "$MODE" = "pre" ]; then
  echo "pre OK (tabel boleh belum ada — yang penting versi tercatat di atas)"
  exit 0
fi

# -- post: tabel WAJIB ada dengan kolom kunci D-012 -------------------------------
if [ "$TABLE_EXISTS" != "event_near_confirmed" ]; then
  echo "POST GAGAL: tabel event_near_confirmed tidak ada setelah migrate up" >&2
  exit 1
fi

for col in first_two_independent_at independent_count_at_peak node_count_at_peak min_independent_cells algo_ver confirmed_at terminal_state terminal_at; do
  if q "SELECT column_name FROM information_schema.columns WHERE table_name='event_near_confirmed' AND column_name='$col';" | grep -q "$col"; then
    echo "kolom OK: $col"
  else
    echo "POST GAGAL: kolom $col hilang" >&2
    exit 1
  fi
done

echo "== isi (kosong = valid di 1 node) =="
q "SELECT count(*) AS rows FROM event_near_confirmed;"
echo "post OK"
