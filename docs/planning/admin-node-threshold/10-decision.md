# 10 — Decision (criteria + recommendation slot — NOT A DECISION)

- Purpose: pre-commit to the acceptance bar BEFORE results, so a recommendation cannot be retro-fitted. This file RECOMMENDS NOTHING today.
- Status: criteria PROPOSED — require owner approval before any threshold verdict. Recommendation: INSUFFICIENT EVIDENCE — DO NOT CHANGE PRODUCTION.
- Last updated: 2026-09-11 UTC.
- Governance: per PROJECT_RULES §6 (population evidence) + §9 (owner decides safety/alert/contract semantics). A floor change additionally needs: accepted decision entry, `algo_ver` ruling (V3/V6), const change + tests, replay profile update, deployment scheduling. None approved here.

## 1. Acceptance categories (PROPOSED — owner to approve/adjust)

- FALSE WARNING (PROPOSED): zero OBSERVED environmental/anomaly triggers ≥ candidate floor across the disturbance battery (§5 levels × mounts) + zero production false eligible over a bounded window the owner names; any single ≥-floor non-seismic eligible FAILS the candidate. (Numbers illustrative — owner sets the window/counts.)
- DETECTION (PROPOSED): candidate must demonstrably admit a class of significant events the baseline demonstrably excluded for floor reasons alone (not heartbeat/detection/ingest reasons), with cases listed, not just a higher count.
- LATENCY (PROPOSED): median eligibility gain 140→candidate ≥ T_min (owner sets; suggest 0.5–1 s floor for "meaningful" given the FCM/Doze tail dominates below it), measured onset→audible, not just server-internal.
- ROBUSTNESS (PROPOSED): decision stable across mounts/sites/anomaly injection (E/F metrics); no new spurious-eligible mode introduced.
- SIMPLICITY (PROPOSED): floor-only const change, no architecture/contract/client change (verify H-5), `algo_ver` treatment recorded.
- TRUST (PROPOSED): behavior explainable in one sentence ("designated verified fresh Admin's FINAL ≥ X warns within 20 km"), auditable via existing reason vocab + `trusted-local emitted` line.

## 2. Recommendation (current)

**INSUFFICIENT EVIDENCE — DO NOT CHANGE PRODUCTION.** Floor stays 140.0. Evidence supports: mechanism mapped (F-01–F-06, F-11), one positive control (F-07), one informative negative (F-08), one threshold-irrelevant deep event (F-09). Missing: disturbance battery, growth-curve timing, sample-rate measurement, population denominators. Next step if owner agrees: run `04/05/06` data+benchmark+replay plans + hardware battery (file `11` below is folded into ledger/experiment files — no separate implementation task opened).

## 3. Exact next step (research-only)

1. Owner approves/repairs the PROPOSED criteria above. 2. Extract read-only replay windows (M1′ + all UNCONFIRMED on NODE-52960B47). 3. Build the offline floor-parameterized harness + populate `07-threshold-comparison.md`. 4. Run the §9-file hardware battery from the laptop (never VPS). 5. Measure GPIO15/DMP rate. 6. Return with KEEP 140 / TEST 100 FURTHER / ADOPT 100-or-X + why, or repeat INSUFFICIENT EVIDENCE.
