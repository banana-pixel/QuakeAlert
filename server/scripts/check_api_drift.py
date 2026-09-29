#!/usr/bin/env python3
# Copyright (c) 2026 QuakeAlert contributors.
# SPDX-License-Identifier: AGPL-3.0-or-later
"""check_api_drift.py -- implementation-vs-OpenAPI drift gate (CIT-002).

Compares the authoritative contract (contracts/openapi/openapi.yaml, top of
the PROJECT_RULES.md authority hierarchy) against the Go implementation and
fails loudly on drift of the classes found as API-001..API-004:

  A. route <-> path parity      (API-004: handler without a path)
  B. DTO struct <-> schema property parity, both directions
                                (API-001: stale contract field;
                                 API-002: undeclared implementation field)
  C. Station.status enum covers every status the code can emit
                                (API-003: unrepresentable state)
  D. test-alert documents the 202 the handler actually writes
                                (API-004 companion: undocumented status)

Why Python and not a Go test: the contract is YAML and this repo already
requires PyYAML for the contract gate (check_contracts.sh section 2 installs
it in CI); adding a Go YAML dependency just for the gate would widen
go.mod for tooling. Why structured parsing and not grep: routes are matched
as (method, path) pairs, Go structs are brace-balanced (not line regexes),
and every extractor FAILs when it finds nothing -- nothing passes silently.

Usage (from the repository root):
  python3 server/scripts/check_api_drift.py            # gate the real tree
  python3 server/scripts/check_api_drift.py --selftest # prove the gate bites:
      replays each API-001..API-004 drift class against mutated copies and
      asserts the matching check FAILs, then asserts the real tree PASSes.

Deterministic, offline, secret-free, production-free. Exit 0 iff all pass.
"""
import copy
import re
import subprocess
import sys
import tempfile
from pathlib import Path

try:
    import yaml
except ImportError:
    print("FAIL: modul yaml Python tidak tersedia (lihat check_contracts.sh bagian 2)")
    sys.exit(1)

PASS = 0
FAIL = 0


def ok(msg):
    global PASS
    PASS += 1
    print("PASS: " + msg)


def fail(msg):
    global FAIL
    FAIL += 1
    print("FAIL: " + msg)


def repo_root():
    out = subprocess.run(
        ["git", "rev-parse", "--show-toplevel"],
        capture_output=True, text=True, check=True,
    )
    return Path(out.stdout.strip())


OPENAPI = "contracts/openapi/openapi.yaml"
ROUTER = "server/internal/api/router.go"

# (schema name, Go struct name, Go source file): the DTOs whose wire shape
# the gate pins. Extend this table -- not the check logic -- for new DTOs.
DTO_TABLE = [
    ("RerollResponse", "rerollResponse", "server/internal/api/api.go"),
    ("UpdateLocationResponse", "updateLocationResponse", "server/internal/api/api.go"),
    ("Station", "stationDTO", "server/internal/api/api.go"),
    ("TestAlertResponse", "testAlertResponse", "server/internal/api/testalert.go"),
    ("CreateTestAlertRequest", "createTestAlertRequest", "server/internal/api/testalert.go"),
]

STATUS_CODES = {
    "StatusOK": 200, "StatusCreated": 201, "StatusAccepted": 202,
    "StatusBadRequest": 400, "StatusUnauthorized": 401,
    "StatusForbidden": 403, "StatusNotFound": 404,
    "StatusTooManyRequests": 429, "StatusInternalServerError": 500,
    "StatusServiceUnavailable": 503,
}


def load_openapi(root, rel=OPENAPI):
    with open(root / rel) as f:
        return yaml.safe_load(f)


def extract_routes(root, rel=ROUTER):
    """(method, path) pairs registered on the chi router, e.g. {('POST', '/api/v1/chat/messages')}."""
    src = (root / rel).read_text()
    found = set()
    for m in re.finditer(r'^\s*r\.(Get|Post|Put|Delete|Patch|Head|Options)\(\s*"([^"]+)"', src, re.M):
        found.add((m.group(1).upper(), m.group(2)))
    return found


def check_routes(root):
    doc = load_openapi(root)
    spec_ops = set()
    for path, item in (doc.get("paths") or {}).items():
        for method in ("get", "post", "put", "delete", "patch", "head", "options"):
            if isinstance(item, dict) and method in item:
                spec_ops.add((method.upper(), path))
    code_ops = extract_routes(root)
    if not code_ops:
        fail("router: tidak ada rute terekstraksi (pola pendaftaran berubah?)")
        return
    for op in sorted(code_ops - spec_ops):
        fail("rute %s %s terdaftar di router.go tetapi tidak ada di paths OpenAPI" % op)
    for op in sorted(spec_ops - code_ops):
        fail("path OpenAPI %s %s tidak punya rute di router.go" % op)
    if not (code_ops - spec_ops) and not (spec_ops - code_ops):
        ok("rute router <-> paths OpenAPI selaras (%d operasi)" % len(code_ops))


def extract_struct_fields(src, struct_name):
    """json tag names of a Go struct, via brace balancing from its declaration."""
    m = re.search(r"type\s+%s\s+struct\s*\{" % re.escape(struct_name), src)
    if not m:
        return None
    depth, i = 0, m.end() - 1
    while i < len(src):
        if src[i] == "{":
            depth += 1
        elif src[i] == "}":
            depth -= 1
            if depth == 0:
                break
        i += 1
    body = src[m.end():i]
    return set(re.findall(r'json:"([^",]+)"', body))


def check_dtos(root):
    doc = load_openapi(root)
    schemas = (doc.get("components") or {}).get("schemas") or {}
    for schema_name, struct_name, rel in DTO_TABLE:
        schema = schemas.get(schema_name)
        if not isinstance(schema, dict):
            fail("skema OpenAPI %s tidak ditemukan" % schema_name)
            continue
        spec_props = set((schema.get("properties") or {}).keys())
        if not spec_props:
            fail("skema OpenAPI %s tidak punya properties" % schema_name)
            continue
        src = (root / rel).read_text()
        code_fields = extract_struct_fields(src, struct_name)
        if code_fields is None:
            fail("struct Go %s tidak ditemukan di %s" % (struct_name, rel))
            continue
        for name in sorted(code_fields - spec_props):
            fail("%s: field wire %r dikirim %s tetapi tidak ada di skema" % (schema_name, name, struct_name))
        for name in sorted(spec_props - code_fields):
            fail("%s: properti skema %r dijanjikan kontrak tetapi tidak dikirim %s" % (schema_name, name, struct_name))
        required = set(schema.get("required") or [])
        for name in sorted(required - spec_props):
            fail("%s: required %r tidak ada di properties" % (schema_name, name))
        if not (code_fields ^ spec_props) and not (required - spec_props):
            ok("skema %s <-> struct %s selaras (%d field)" % (schema_name, struct_name, len(spec_props)))


def func_body(src, func_name):
    """Body of a top-level Go func, up to (not including) the next one.

    func_name is already a regex fragment (callers pass escaped receivers),
    so it must NOT be re.escape()d here.
    """
    m = re.search(r"\nfunc\s+%s\b" % func_name, src)
    if not m:
        return None
    rest = src[m.end():]
    nxt = re.search(r"\nfunc\s+\w", rest)
    return rest[:nxt.start()] if nxt else rest


def check_station_enum(root):
    doc = load_openapi(root)
    schemas = (doc.get("components") or {}).get("schemas") or {}
    status = ((schemas.get("Station") or {}).get("properties") or {}).get("status") or {}
    enum = status.get("enum") or []
    src = (root / "server/internal/api/api.go").read_text()
    body = func_body(src, r"\(s \*Server\) HandleListSensors")
    if body is None:
        fail("HandleListSensors tidak ditemukan di api.go")
        return
    emitted = set(re.findall(r'\bstatus\s*(?::=|=)\s*"([A-Za-z]+)"', body))
    if not emitted:
        fail("tidak ada literal status terekstraksi dari HandleListSensors")
        return
    for value in sorted(emitted):
        if value not in enum:
            fail("Station.status=%r dikirim HandleListSensors tetapi tidak ada di enum kontrak" % value)
    if all(v in enum for v in emitted):
        ok("enum Station.status mencakup semua status yang dikirim kode (%s)" % ",".join(sorted(emitted)))


def check_test_alert_code(root):
    doc = load_openapi(root)
    path_item = None
    for path, item in (doc.get("paths") or {}).items():
        if isinstance(item, dict) and (item.get("post") or {}).get("operationId") == "createTestAlert":
            path_item = (path, item["post"])
            break
    if path_item is None:
        fail("operasi createTestAlert tidak ada di paths OpenAPI")
        return
    path, op = path_item
    src = (root / "server/internal/api/testalert.go").read_text()
    body = func_body(src, r"\(s \*Server\) HandleCreateTestAlert")
    if body is None:
        fail("HandleCreateTestAlert tidak ditemukan di testalert.go")
        return
    m = re.search(r"writeJSON\(w,\s*http\.(Status\w+),\s*testAlertResponse", body)
    if not m or m.group(1) not in STATUS_CODES:
        fail("kode sukses testAlertResponse tidak terekstraksi dari HandleCreateTestAlert")
        return
    code = str(STATUS_CODES[m.group(1)])
    responses = op.get("responses") or {}
    if code in responses:
        ok("POST %s mendokumentasikan %s seperti yang ditulis handler" % (path, code))
    else:
        fail("POST %s menulis %s tetapi responses kontrak tidak memuatnya" % (path, code))


def run_all(root):
    check_routes(root)
    check_dtos(root)
    check_station_enum(root)
    check_test_alert_code(root)


# -- selftest: replay each API-001..API-004 drift class against mutated
# copies and prove the matching check FAILs (fixtures are generated from the
# real files so they cannot rot independently of the contract). ----------

def mutate(path, old, new, count=1):
    text = path.read_text()
    assert old in text, "jangkar mutasi tidak ditemukan di %s" % path
    path.write_text(text.replace(old, new, count))


def selftest_case(name, mutate_fn, expect_in_output):
    global PASS, FAIL
    with tempfile.TemporaryDirectory() as tmp:
        tmp = Path(tmp)
        (tmp / "contracts/openapi").mkdir(parents=True)
        (tmp / "server/internal/api").mkdir(parents=True)
        real = repo_root()
        (tmp / OPENAPI).write_text((real / OPENAPI).read_text())
        for rel in (ROUTER, "server/internal/api/api.go", "server/internal/api/testalert.go"):
            (tmp / rel).write_text((real / rel).read_text())
        mutate_fn(tmp)
        PASS, FAIL = 0, 0
        import io
        from contextlib import redirect_stdout
        buf = io.StringIO()
        with redirect_stdout(buf):
            run_all(tmp)
        out = buf.getvalue()
        if FAIL == 0:
            print("SELFTEST-GAGAL %s: drift tidak tertangkap:\n%s" % (name, out))
            return False
        if expect_in_output not in out:
            print("SELFTEST-GAGAL %s: pesan tak terduga:\n%s" % (name, out))
            return False
        print("SELFTEST-OK %s: drift tertangkap" % name)
        return True


def remove_test_alert_path(tmp):
    doc = load_openapi(tmp)
    del doc["paths"]["/api/v1/admin/test-alert"]
    with open(tmp / OPENAPI, "w") as f:
        yaml.safe_dump(doc, f, allow_unicode=True)


def selftest():
    real = repo_root()
    cases = [
        ("API-004: path test-alert hilang dari kontrak",
         remove_test_alert_path, "tidak ada di paths OpenAPI"),
        ("API-004: rute tanpa path",
         lambda tmp: mutate(tmp / ROUTER,
                             'r.Post("/api/v1/admin/test-alert", s.HandleCreateTestAlert)',
                             'r.Post("/api/v1/admin/test-alert", s.HandleCreateTestAlert)\n\t\tr.Post("/api/v1/admin/phantom", s.HandleCreateTestAlert)'),
         "tidak ada di paths OpenAPI"),
        ("API-002: region_code dihapus dari UpdateLocationResponse",
         lambda tmp: mutate(tmp / OPENAPI,
                             "        region_code:\n          type: [string, \"null\"]\n          maxLength: 50",
                             "        region_code_gone:\n          type: [string, \"null\"]\n          maxLength: 50"),
         "tidak ada di skema"),
        ("API-001: region_code basi di RerollResponse",
         lambda tmp: mutate(tmp / OPENAPI,
                             "        pseudonym:\n          type: string\n          example: Quakezen-7B9A\n",
                             "        pseudonym:\n          type: string\n          example: Quakezen-7B9A\n        region_code:\n          type: [string, \"null\"]\n"),
         "dijanjikan kontrak tetapi tidak dikirim"),
        ("API-003: Pending hilang dari enum",
         lambda tmp: mutate(tmp / OPENAPI,
                             "enum: [Online, Offline, Pending]",
                             "enum: [Online, Offline]"),
         "tidak ada di enum kontrak"),
        ("API-004: handler menulis 201 tetapi kontrak 202",
         lambda tmp: mutate(tmp / "server/internal/api/testalert.go",
                             "writeJSON(w, http.StatusAccepted, testAlertResponse{",
                             "writeJSON(w, http.StatusCreated, testAlertResponse{"),
         "tidak memuatnya"),
    ]
    bad = 0
    for name, fn, expect in cases:
        if not selftest_case(name, fn, expect):
            bad += 1
    # Kasus positif: pohon asli harus lolos bersih.
    global PASS, FAIL
    PASS, FAIL = 0, 0
    run_all(real)
    if FAIL != 0:
        print("SELFTEST-GAGAL pohon-asli: gate menolak tree yang seharusnya lolos")
        bad += 1
    else:
        print("SELFTEST-OK pohon-asli: %d pemeriksaan lolos" % PASS)
    return 1 if bad else 0


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--selftest":
        sys.exit(selftest())
    run_all(repo_root())
    print("---")
    print("check_api_drift: %d PASS / %d FAIL" % (PASS, FAIL))
    sys.exit(1 if FAIL else 0)
