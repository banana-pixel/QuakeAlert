# 09 — Findings (evidence matrix + unknowns)

- Purpose: the running answer sheet. Every row points to a ledger entry; ledger points back. Superseded rows stay with strikethrough + pointer.
- Status: initial (source-read + LIVE-EQ reuse only).
- Last updated: 2026-09-11 UTC.

## Evidence matrix

| ID | Finding | Evidence | Confidence | Impact |
| --- | --- | --- | --- | --- |
| F-01 | Production floor is 140.0 compile-time; 100 exists nowhere in prod code | `admin_node.go:30`, grep 2026-09-11 | FACT (source) | anchors all replay |
| F-02 | Eligibility = designated+verified+UNCONFIRMED+admin-contrib+FINAL+PGA≥floor+heartbeat≤5m, in that gate order | `admin_node.go:96-119` | FACT | floor is 1 of 7 gates — 6 others can be the limiter |
| F-03 | PRELIM never warns locally (`ADMIN_NOT_FINAL`); FINAL-only by construction | `admin_node.go:110`, `tracker.go:407-427` | FACT | 140→100 gains only via FINAL edge/transition |
| F-04 | PRELIM PGA (peak-so-far) ≪ FINAL PGA (whole-event max) for the same episode | `sensor.cpp:209-287`, 004c5f65 1.13→288.36 | FACT + OBSERVED | phases not comparable; intra-event 100-crossing instant unlogged |
| F-05 | Edge path (D-037) fires exactly-once, no revision/log, async, fail-closed | `tracker.go`, `emit.go`, `event.go:198-206` | FACT | latency model must include async store read |
| F-06 | Dispatch is token-only 20 km, never GeoTopic, WS identical, audit without coords | `event_frame.go:82-138` | FACT | blast radius bounded at any floor |
| F-07 | 288.36 gal FINAL → live ADMIN_ELIGIBLE + TOKENS_RADIUS_20KM (24/2) | LIVE-EQ AUDIT §5 control + emissions 94-96 | OBSERVED | positive control ≥140 |
| F-08 | 77.89 gal event produced NO local frame (below floor, else eligible-class) | P4-M6 archive (4fcc3374) | OBSERVED | key negative for 60–100 boundary |
| F-09 | M5.9 deep produced ZERO rows at any PGA — threshold-irrelevant | LIVE-EQ AUDIT §§5-8 | OBSERVED (absence, high conf) + INFERRED (<1.5 gal STA) | denominator, not miss; no calibration weight |
| F-10 | True GPIO15/DMP rate UNMEASURED; 100 Hz assumed in comments, `setRate(99)` in code | `config.h:100-101`, `sensor.cpp:65-70` | UNKNOWN | rescales all STA/LTA timing claims |
| F-11 | `algo_ver phase3-1.1/ic=5` does NOT encode the admin floor | `store.go:26,71-73` | FACT | floor change forces version decision (V3/V6) |
| F-12 | 140 ≈ just above "strong" label 137.2; 100 mid-"moderate" | `centroid.go:97-106` | FACT | UI-copy note for any lowering |

## Key unknowns (block a recommendation until closed or explicitly accepted as residual)

1. Per-disturbance gal at THIS mount (the 100–140 row is empty). 2. Intra-event PGA growth curve (100→140 crossing time). 3. True sample rate. 4. Heartbeat continuity through quake windows. 5. Audible-alarm tail latency distribution (locked vs in-use, vendors). 6. Population denominators for any RATE.
