# 07 — Threshold comparison (evidence table — mostly EMPTY today)

- Purpose: one table where every candidate meets the same six metrics. Empty cells are HONEST (UNKNOWN), not zero.
- Status: scaffold; only OBSERVED anchors filled. DO NOT RECOMMEND from this version. Sweep harness READY (R-019) — column A fillable for the population once the owner runs it read-only over the extracted windows.
- Last updated: 2026-09-26 UTC.

## Candidates × metrics (A–F per `05-experiment-plan.md`)

| Floor | A detect | B missed | C latency gain vs 140 | D false-warning | E mount sensitivity | F anomaly robustness |
| --- | --- | --- | --- | --- | --- | --- |
| 60 | POPULATION UNKNOWN (run harness); anchors: 4fcc3374 77.89 → ELIGIBLE (confirmed by R-019 sweep), 004c5f65 → eligible | UNKNOWN | UNKNOWN | UNKNOWN — closest to disturbance band, highest risk | UNKNOWN | UNKNOWN |
| 80 | POPULATION UNKNOWN (run harness); anchors: 4fcc3374 77.89 → BELOW_FLOOR (77.89 < 80, R-019 sharpens the loose "≤80" note), 004c5f65 → eligible | UNKNOWN | UNKNOWN | UNKNOWN | UNKNOWN | UNKNOWN |
| 100 | POPULATION UNKNOWN (run harness); anchors: 004c5f65 288.36 → eligible, 4fcc3374 77.89 → BELOW_FLOOR | UNKNOWN | UNKNOWN (gain mechanism unlogged — see §06 §3) | UNKNOWN — THE question of this research | UNKNOWN | UNKNOWN |
| 120 | UNKNOWN | UNKNOWN | UNKNOWN | UNKNOWN | UNKNOWN | UNKNOWN |
| 140 (baseline) | OBSERVED ≥1 eligible (004c5f65 FINAL 288.36 → ADMIN_ELIGIBLE + TOKENS_RADIUS_20KM) | OBSERVED ≥1 below-floor seismic-class (4fcc3374 77.89, no local frame) | baseline (0 by def) | OBSERVED zero environmental false warnings *at current install, small-n* (observational, NOT a rate) | UNKNOWN | INFERRED guarded (warmup/cooldown/clamp/reset) — UNPROVEN |
| 160 | UNKNOWN (004c5f65 still eligible) | UNKNOWN | UNKNOWN (negative gain vs 140 by construction) | UNKNOWN (lower than 140 by construction, unmeasured) | UNKNOWN | UNKNOWN |

## Notes

- **How to fill column A (R-019):** run the read-only sweep harness over each window and paste its table here. Owner-side, read-only role:
  ```
  FROM_TS=<ms> TO_TS=<ms> ADMIN_NODE_ID=NODE-52960B47 ADMIN_NODE_VERIFIED=true \
  DATABASE_URL="postgres://<readonly>@host/quakealert?sslmode=disable" \
  go run scripts/admin_floor_sweep.go
  ```
  The harness proves its floor-parameterized copy identical to production `EvaluateAdminNodeEligibility` at 140 before reporting (aborts on drift). `SELFTEST=1 go run scripts/admin_floor_sweep.go` runs the faithfulness proof with no DB. It answers DETECTION only — never latency (metric C, F-04) or false-warning (metric D, needs the bench battery).

- 140 sits just above the "strong" label boundary 137.2 gal (`centroid.go:97-106`); 100 sits mid-"moderate" (16.6–137.2). A 100 floor would warn on events the UI calls "moderate" — copy/audit implication for `10-decision.md`, not a blocker.
- Trusted-local has NO severe override (client `decideLocalWarning` distance-only; server no-GeoTopic-ever), so floors ≥250 interact with NOTHING — no reason to benchmark there beyond completeness.
- MMI equivalents (Wald `3.66*log10(PGA)-1.66`, for intuition ONLY — the gate is on gal): 60→~4.9 (V), 80→~5.3 (V), 100→~5.7 (VI), 120→~6.0 (VI), 140→~6.2 (VI), 160→~6.4 (VI). All candidates round to MMI V–VI; none reaches the VII severe override. (Computed 2026-09-11; verify before citing.)
