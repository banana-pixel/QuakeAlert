#!/usr/bin/env bash
# check_contracts.sh — penjaga contract-first (D-001 / ADR-0004) di CI.
#
# Setiap klaim di bawah diuji sebagai asersi yang GAGAL BERSUARA, bukan lolos
# diam-diam: kontrak yang tidak lagi menggambarkan kode adalah cacat pada
# kontrak (otoritatif), bukan pada kode. Dijalankan dari akar repo:
#   bash server/scripts/check_contracts.sh
#
# Yang dicek: JSON kontrak valid, OpenAPI valid, setiap kode writeError ada di
# enum Error.code, cermin is_test/validity_ms lintas kanal, dan tidak ada
# private key yang terlacak di git.
set -euo pipefail

PASS=0
FAIL=0
ok()   { PASS=$((PASS+1)); echo "PASS: $1"; }
fail() { FAIL=$((FAIL+1)); echo "FAIL: $1"; }

# 1. Semua JSON kontrak harus terurai ------------------------------------------------
json_count=0
while IFS= read -r f; do
  json_count=$((json_count+1))
  if python3 -c "import json,sys; json.load(open(sys.argv[1]))" "$f"; then
    ok "JSON valid: $f"
  else
    fail "JSON rusak: $f"
  fi
done < <(git ls-files 'contracts/**/*.json')

[ "$json_count" -gt 0 ] || { fail "tidak ada JSON kontrak ditemukan"; }
ok "jumlah JSON kontrak diperiksa: $json_count"

# 2. OpenAPI harus terurai sebagai YAML ----------------------------------------------
if python3 -c "import yaml,sys; yaml.safe_load(open('contracts/openapi/openapi.yaml'))" 2>/dev/null; then
  ok "OpenAPI YAML valid"
else
  fail "OpenAPI YAML tidak terurai (butuh modul yaml di CI)"
fi

# 3. Setiap kode writeError harus ada di enum Error.code ------------------------------
# Pola: writeError(w, <status>, "<CODE>", ...) di server/internal/api/*.go
mapfile -t codes < <(git grep -hoE 'writeError\(w, [^,]+, "[A-Z][A-Z_]*"' -- server/internal/api/ \
  | grep -oE '"[A-Z][A-Z_]*"$' | tr -d '"' | sort -u)
[ "${#codes[@]}" -gt 0 ] || { fail "tidak ada kode writeError ditemukan"; }
for code in "${codes[@]}"; do
  if grep -qE "^[[:space:]]+-[[:space:]]+$code$" contracts/openapi/openapi.yaml; then
    ok "Error.code memuat $code"
  else
    fail "Error.code TIDAK memuat $code (handler mengembalikannya)"
  fi
done

# 4. Cermin is_test / validity_ms lintas kanal (D-018) ---------------------------------
for field in is_test validity_ms; do
  if grep -q "\"$field\"" contracts/fcm/alert_payload.json; then
    ok "FCM membawa $field"
  else
    fail "FCM TIDAK membawa $field"
  fi
  if python3 -c "import json,sys; d=json.load(open('contracts/mqtt/alert.schema.json')); sys.exit(0 if '$field' in d['properties'] else 1)"; then
    ok "MQTT alert.schema membawa $field"
  else
    fail "MQTT alert.schema TIDAK membawa $field"
  fi
done
if grep -q 'validity_ms' contracts/openapi/openapi.yaml; then
  ok "OpenAPI mendokumentasikan validity_ms"
else
  fail "OpenAPI tidak menyebut validity_ms sama sekali"
fi

# 5. Tidak ada private key yang terlacak ------------------------------------------------
if git grep -qE 'BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY' -- . ; then
  fail "private key ditemukan di file terlacak"
else
  ok "tidak ada private key di file terlacak"
fi

echo "---"
echo "check_contracts: $PASS PASS / $FAIL FAIL"
[ "$FAIL" -eq 0 ]
