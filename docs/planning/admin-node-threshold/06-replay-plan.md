# 06 — Replay plan ("if the floor had been X")

- Purpose: answer RQ-ELIG/RQ-LAT from HISTORY without touching production.
- Status: method defined; not executed.
- Last updated: 2026-09-11 UTC.
- Basis: D-013 (F2 bijection, F3 delta, canonical order, operator-asserted params) + D-015 (membership-and-time, non-causal) + `01-current-state.md` §9 anchors.

## 1. Input

Historical windows from `sensor_observations` + `event_state_log` (+ `alert_emissions` for the advisory link), grouped by `algo_ver`, canonical order `received_ts, observation_id`, with the §04 coverage block. Start with: P4-M1 51-row window, events 004c5f65 + 4fcc3374, then every UNCONFIRMED on `NODE-52960B47` since `phase3-1.1/ic=5`.

## 2. Procedure (per event, per candidate floor X)

1. Rebuild the contributor timeline: PRELIM rows (phase, PGA, onset, obs_seq) then FINAL rows; apply server absorb rules (PGA ratchets up, Phase sticks to FINAL, onset first-wins) to get admin-contribution PeakPGA at (a) each transition snapshot and (b) the D-037 edge instant.
2. Evaluate the OFFLINE eligibility copy at X (all other gates as-recorded: designated/verified/heartbeat-at-the-time — heartbeat history is UNAVAILABLE server-side, so record heartbeat-assumed-fresh vs unknown explicitly; never invent freshness).
3. Record: eligible? at which stage (transition rev N vs edge)? `decided_at` of that stage? PRELIM→eligibility gap? which reason would have fired otherwise?
4. Latency decomposition: `onset→PRELIM received`, `PRELIM→FINAL received` (the 140→100 gain lives INSIDE this interval only if PGA growth 100→140 spans it — verify per event, never assume), `eligibility→emission` (async 2 s budget + FCM send), `emission→audible` (device-measured, separate).
5. Repeat for X ∈ {60,80,100,120,140,160}; tabulate in `07-threshold-comparison.md`.

## 3. Worked expectations (PREDICTIONS, not results)

- 004c5f65 (PRELIM 1.13 → FINAL 288.36): eligible at ALL floors 60–160, at the edge (already UNCONFIRMED when FINAL landed). Gain 140→100 for THIS event = 0 at eligibility stage (both fire on the same FINAL); any gain would have to come from an EARLIER FINAL crossing — unknowable from two rows (intra-event PGA growth unlogged). Records the key instrumentation gap: firmware logs only peak-so-far (PRELIM) + peak (FINAL), not the 100-crossing instant.
- 4fcc3374 (77.89): eligible at 60 only — the single most informative negative for the 80/100 boundary.
- LIVE-EQ window: vacuous for all X (no rows) — file as denominator, not as miss.

## 4. Limits (print on every replay output)

Membership-and-time attribution (not causal); `ledger_drops` UNKNOWN; heartbeat-at-event-time mostly UNKNOWN (no history); intra-event growth curve UNKNOWN from current rows (only two PGA points per episode); single-node fleet (CONFIRMED path unexercised); operator-asserted non-`ic` params; F3 relative time only.
