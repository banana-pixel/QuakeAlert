# 06 — Shadow-mode runbook (S1–S3: LOG-ONLY, owner-approved)

- Scope: owner-approved S1–S3 only. A strictly log-only **shadow** of the DPE-1 provisional decision runs on the server: *NO* FCM, *NO* WebSocket frames, *NO* trusted_local, *NO* siren, *NO* change to alarm eligibility, dedup, revisions, emission tables, or any production behavior. This is not a real PRELIM warning and not an adoption of DPE-1..8.
- Implementation: `server/internal/event/prelim_shadow.go` (the entire shadow), hooks placed in `emitTrustedPrelim` (PRELIM edge), `emitTrustedLocal` (FINAL leg), `EmitTransition` (terminal/confirmed) in `server/internal/event/emit.go` + `admin_node.go`. Everything additive; zero-sink behaviour proven by `server/internal/event/prelim_shadow_test.go` (16 assertions, `-race` green, full `go test ./...` green).

## What one record looks like

A JSON line on the server's stdout, first label `PRELIM_SHADOW`:

```json
{"time":"...","level":"INFO","msg":"PRELIM_SHADOW","shadow":true,"simulated":true,"note":"shadow-simulated-bukan-dispatch","event_id":"E0000000-…","revision":3,"would_dispatch":true,"reason":"ADMIN_NOT_FINAL","floor_gal":60,"would_validity_ms":90000,"validity_basis":"DPE-6-PLACEHOLDER-90S","decided_at":1700000004000,"contrib_pga_gal":100,"event_peak_pga_gal":100}
```

- `shadow`/`simulated`/`note`: every line is labelled a research row, never a dispatch.
- `would_dispatch` (PRELIM edge): the hypothetical DPE-1 decision — real evaluator gates verbatim, only the `ADMIN_NOT_FINAL` veto relaxed for PRELIM-phase contributions. `floor_gal` is the real 60-gal floor (D-038); it is **not** relaxed: evaluator gate order puts the veto before the floor/heartbeat gates, so the shadow re-completes those gates against the PRELIM contribution (`ADMIN_PGA_BELOW_FLOOR`, `ADMIN_HEARTBEAT_STALE` — both `would_dispatch=false`).
- `would_outcome` (FINAL leg / terminal): DPE-4 outcome — `absorb` (FINAL eligible; the real trusted_local dispatch still happens exactly as before), `withdraw` (FINAL below floor / stale heartbeat / event ends RESOLVED-CANCELLED without FINAL), or `confirmed` (escalation to CONFIRMED closed the episode before the FINAL leg).
- `would_validity_ms` = 90000 with `validity_basis = DPE-6-PLACEHOLDER-90S`: placeholder, NOT a measurement (production prov-lead distribution is still owed — file 02). No interpretation may also claim the other number.
- Privacy: no coordinates anywhere — no centroid, no lat/lon; intensity only (`contrib_pga_gal`, `event_peak_pga_gal`). XX-privacy posture makes every label unique (D-019/D-023, "WITHOUT INVASION-PROOF …").

## Exactly-once per episode

One `would_dispatch` record per event (reload/re-edge suppressed); the first outcome (`absorb`/`withdraw`/`confirmed`) deletes the entry — retries at the FINAL leg never double-log. Events without a designated admin node are not recorded (noise suppression, same fail-closed posture as the real hooks).

## Retention

- Records go to the server's **stdout only**. Production runs under `deploy/docker-compose.prod.yml` with the `json-file` driver: `max-size 10m`, `max-file 3` — bounded to ≈30 MB, oldest rotated away. There is no dedicated log file and none is created.
- The in-memory would-dispatch map is lost on process restart; an episode spanning a restart can therefore emit a second `would_dispatch` record. Comparable timestamps mark it; treat it as research noise, not a bug.

## Export procedure (operator)

1. List the records:
   `docker logs <quakealert-container> 2>&1 | grep PRELIM_SHADOW > prelim-shadow-$(date +%F).ndjson`
   (or `docker compose -f deploy/docker-compose.prod.yml logs quakealert` with `--since`/`--until` for a window).
2. Hash and archive: `sha256sum prelim-shadow-…ndjson`, then copy the file + hash + a README (with the same SYNTHETIC-fixture disclaimer pattern used by `docs/evidence/prelim-walltime/2026-10-18-synthetic-fixture-demo/`, when run against fixtures) into `docs/evidence/prelim-shadow/<date>/`. The grep output contains only labelled shadow records — production lines from other paths stay out.
3. Correlation with latency windows: `decided_at` is epoch-ms on the same server clock the `prelim_walltime` tool uses, so shadow stamps line up with A1's gain / prov-lead windows.
4. Retention caveat: after log rotation the oldest records are gone — archive before they rotate out if an episode matters.
