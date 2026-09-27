# 04 — Data plan (what counts, where it lives, how grouped)

- Purpose: define the evidence population BEFORE measuring, per PROJECT_RULES §6 (population, not single event) and V5 (group by `algo_ver`).
- Status: plan; no new data collected in this phase beyond source reads + LIVE-EQ audit reuse.
- Last updated: 2026-09-11 UTC.

## 1. Production rows usable for replay (read-only)

- `sensor_observations`: PRELIM/FINAL PGA, `dur_ms`, `publish_ts`/`received_ts`, `onset_ts`+source, `obs_seq`, `attempt_no`, `detrigger_ts`, `verify_result`, `node_location` (NULL ≠ (0,0)), `observation_id` (BIGSERIAL tie-break for canonical order).
- `event_state_log`: revision, from/to, reason, `decided_at`, node/cell counts, `peak_pga`, `evidence_summary` (contributors[] with node/phase/peak/onset/obs_seq/cell), `algo_ver`.
- `alert_emissions`: frame type, event_id+revision link (exact proof `MATCHED_BY_EVENT_ID_AND_REVISION`), audience, FCM attempted/succeeded, `ws_clients`.
- `earthquake_events`: latest shape only (overwritten on escalation — NOT history); `event_state_log` is the history.
- Coverage/incompleteness to record every run: `ledger_drops` UNKNOWN (log-only `ledger_drops_total`), NULL locations, failed verifications, no-onset-anchor rows, schema version, `algo_ver` per row, tolerance provenance. Absence never = proof of absence (D-015).
- Grouping: split by `algo_ver` before ANY comparison (V5). Current label `phase3-1.1/ic=5`; pre-Phase-3 NULLs never mixed. Operator-asserted params (correlation window, attach radius, resolve-after, sweep, max diameter, min cells) printed BEFORE results (D-013); `INDEPENDENCE_CELL_KM` contradiction → reject.

## 2. Anchored production points (to date)

- 004c5f65 (2026-09-11 12:40Z): obs 90 PRELIM 1.1329 gal → obs 91 FINAL 288.3640 gal (same obs_seq, SENSOR, attempt 1); event RESOLVED rev2, 1 node/1 cell, `phase3-1.1/ic=5`; emissions 94 ADVISORY / 95 ALERT `TOKENS_RADIUS_20KM` (24/2) / 96 RESOLVED; `ADMIN_ELIGIBLE`. Positive control for ≥140.
- 4fcc3374 (P4-M6): 77.8888 gal, 1/1, RESOLVED rev2 — negative control (eligible at ≤80 candidates, not at 100+).
- LIVE-EQ M5.9 deep 2026-09-11 21:23Z: triple-zero 20:00–22:00 UTC — no-evidence denominator (S5), threshold-irrelevant.
- P4-M1 window (51 rows, 26 qualifying, 26 traced): population forage ground for 60–160 replay once extracted read-only.

## 3. Non-production inputs

- Firmware host tests (`firmware/test/canonical_host_test.cpp`, `onset_host_test.cpp`): PRELIM/FINAL canonical strings, epoch math — software evidence only.
- Simulation artifacts (`.sim-evidence/`, CI #22): virtual-node rows — never field correlation (D-011 c2, D-014).
- External waveform catalogues (CANDIDATES, not yet pulled): PEER NGA-West2, K-NET/KiK-net, USGS strong-motion, BMKG InaTEWS post-event (reference ONLY per §2). All are external seismometers — document non-equivalence to ESP32 MEMS (bandwidth, noise, clipping, mounting, site) before ANY numeric transfer. No external data enters the realtime path (S7).

## 4. What is NOT data for threshold change

Single events (PROJECT_RULES §6 forbids); reconstructed `evidence_summary` agreement (tautological where scalars were the only capture); simulation passes; drill-path alarms (no `event_state`, no row).
