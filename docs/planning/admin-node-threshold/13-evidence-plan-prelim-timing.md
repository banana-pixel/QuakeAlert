# 13 — Evidence phase plan: PRELIM timing + growth + rate (plan — NOT executed)

- Purpose: concrete measurement/replay program that closes the UNKNOWNs blocking any E2 provisional-warning decision (`12-...md` §12 unknowns 1–8, `09-findings.md` unknowns 1–6). Produces evidence only; decides nothing; changes nothing in production.
- Status: plan; NO measurement executed, NO hardware touched, NO code written under this file.
- Last updated: 2026-09-13 UTC.
- Authority note: RESEARCH planning area; does not override `/contracts`, `PROJECT_RULES.md`, `docs/DECISIONS.md`, `docs/CURRENT_STATE.md` (PROJECT_RULES §5). Conflict → higher document wins.
- Stop condition (binding): no production deployment, no VPS access (read or write — the M1′/M6′ read-only precedent is NOT re-authorized here; any future production SELECT needs a separate owner authorization), no firmware flashing, no Android/server/contract/threshold/`algo_ver`/migration change. Firmware work below is DESIGN-ONLY (minimal read-only instrumentation specification); implementation, if ever approved, is a separate laptop-side task, never VPS.
- Separation: Problem 1 (floor value) and Problem 2 (PRELIM timing) stay decoupled; every measurement below is parameterized by candidate bar (60/80/100/120/140/160) so one dataset serves both.

## Label legend

- OBSERVED: source text, test, or archived evidence with `file:line` or evidence path.
- INFERRED: consequence following from OBSERVED by stated reasoning.
- HYPOTHETICAL: design considered, not implemented/approved.
- UNKNOWN: not established; must not be guessed.

## 0. What this phase must deliver (exit checklist)

- [ ] M1 true sample rate measured (rescues or falsifies every STA/LTA constant).
- [ ] M2 intra-event threshold-crossing instants specified (design-only) — the Q4 prerequisite.
- [ ] M3 PRELIM→FINAL growth distribution from history + new captures (two-point today, curve after M2).
- [ ] M4 server-ingest latency distribution beyond healthy (7–14 ms OBSERVED healthy only).
- [ ] M5 heartbeat-continuity method defined + baseline captured without VPS writes.
- [ ] M6 candidate-bar crossing table populated (replay at 60/80/100/120/140/160 with reason-at-PRELIM vs reason-at-FINAL).
- [ ] M7 audible-tail sample extended (locked vs in-use; vendor variance stays UNKNOWN with one device — recorded, not hidden).
- [ ] M8 E4 armed-dispatch cost measured offline (async store budget + token lookup off-path benefit).
- [ ] `07-threshold-comparison.md` C/D columns no longer all-UNKNOWN, or explicitly WHY they remain so.
- [ ] Ledger + changelog rows per run; any HYPOTHESIS falsified → mark SUPERSEDED, never delete (`03-hypotheses.md` rule).

## 1. Measurement matrix (one row per measurement)

| ID | Measures | Why it matters | Exact timestamp / reference | Collection (no-prod-change) | Informs | Cannot prove |
| --- | --- | --- | --- | --- | --- | --- |
| M1 | True GPIO15 ISR + DMP packet rate (mean/jitter/drift, 60 s window) | Every STA/LTA/baseline constant is stated "at 100 Hz" (`config.h:100-101`) while code sets `setRate(99)` (`R-007`); wrong rate rescales trigger 4.0/300 ms/detrigger 1.5 and all latency claims | ISR edge count vs laptop clock; DMP packet count in `sensorTask` semaphore path (`sensor.cpp:65-83,308-326`) | FIRST: zero-change logic-analyzer/scope on GPIO15, laptop-side. FALLBACK (separate approval): diagnostic counter build flashed from laptop to the bench node ONLY, never production; counter removed afterward | H-6 verdict; re-derivation of T1; validity of every timing number in `11-` | Seismic detection quality; transfer to any other unit |
| M2 | Intra-event instants `t_cross(X)` for X ∈ {60,80,100,120,140} gal + `t_prelim`, `t_final`, peak-so-far@PRELIM, max@FINAL | Q4 prerequisite: replay rows carry only 2 PGA points/episode, so 140→100 gain is unfalsifiable today (`06-...md` §3, `R-003` impact) | `millis()` at first `correctedMagnitude ≥ X` while `eventInProgress`, plus existing `eventStartTime`/`detriggerMs` (`sensor.cpp:210-272`) | DESIGN-ONLY here (§3 spec: serial-log lines, never MQTT/contract). Capture on bench + opportunistic real events after separate approval | Growth curve; T_min gain test (`10-...md` §1 LAT); Q2 split-vs-same evidence | Population curve (one mount/site); seismic equivalence of taps |
| M3 | PRELIM→FINAL growth time per episode: `received_ts(FINAL) − received_ts(PRELIM)` + `dur_ms` delta, grouped by `algo_ver` | Only latency component E2 can remove (T5 wait); bounds the MAXIMUM provisional gain even before M2 | Existing `sensor_observations.received_ts` + `dur_ms` + `obs_seq` pairing (same obs_seq = same episode); canonical order `received_ts, observation_id` (D-013) | Offline replay over extracted windows (§4: P4-M1 51-row set, all UNCONFIRMED on NODE-52960B47 since `phase3-1.1/ic=5`); read-only SELECTs only after owner re-authorizes production reads | `07-` column C scaffold (two-point bound); which episodes had ANY room for gain | Crossing instant within the gap (needs M2); causal attribution (membership-and-time only, D-015) |
| M4 | Server ingest latency: `received_ts − ts` (publish→receipt) + verify→`decided_at` share | Separates network/transit from shake-duration in onset→decided (2026-09-01 case: 6166 ≈ 6138, server share 28 ms OBSERVED — one point) | `sensor_observations` (`ts`, `received_ts`) + `event_state_log.decided_at`; `attempt_no > 1` flagged separately (retry tail) | Same offline windows as M3; no new code | Whether T3/T7 dominate or are negligible vs T5; E4 shave plausibility | Unhealthy distribution (needs opportune captures); FCM/device tail (M7) |
| M5 | Heartbeat continuity through event windows | Gates 7/7 (`HeartbeatAge ≤ 5 min`, `admin_node.go:114-115`); history UNAVAILABLE server-side (single `last_heartbeat = NOW()`, `store.go:111-137`); a stale heartbeat at FINAL silently kills an earned warning | `HEARTBEAT_INTERVAL_MS 60000` (`config.h:130`) vs event `onset_ts…decided_at` windows; `GetAdminNodeStatus` single-read semantics (`store/admin_node.go:136-168`, NULL = stale per `admin_node_test.go:466`) | Laptop-side polling of read-only status surface at 30 s cadence across battery + opportunistic events (no VPS writes; production polling only if owner authorizes reads); bench: unplug/reboot/drop-MQTT cases observe `last_heartbeat` aging | Whether heartbeat is ever the limiter in practice; provisional re-evaluation policy at FINAL (freshness re-check design) | Multi-month continuity (needs longitudinal sampling); broker-outage vs node-death discrimination |
| M6 | Candidate-bar eligibility at PRELIM-instant vs FINAL-instant (reason-at-X) | Fills `07-` A/B/D cells without touching production: which history WOULD have warned at 60/80/100/120/140/160 and at which stage | Per §`06-...md` procedure step 2: rebuild contributor timeline (ratchet/max/sticky per `tracker.go:699-740`), evaluate OFFLINE eligibility copy at X with all other gates as-recorded (heartbeat explicitly assumed-fresh vs unknown — never invented) | Offline harness = parameterized copy of `EvaluateAdminNodeEligibility` (`admin_node.go:96-119`), production const untouched; inputs per `04-...md` §1 (004c5f65, 4fcc3374, P4-M1 window, then all UNCONFIRMED) | Q2 evidence (same-vs-split); F-08 boundary value (77.89 eligible iff floor ≤77.89); false-provisional cell frequency IN HISTORY (not a rate) | Future FP rate (history n≈tens, single site); heartbeat-at-the-time (mostly UNKNOWN — printed per row) |
| M7 | Audible-tail latency: dispatch→audible, locked-Doze vs in-use heads-up+siren | Downstream tail dominates sub-second T1–T4 gains (`10-...md` T_min rationale); only 2 drill points OBSERVED (+467 ms locked, +960 ms FCM-revived) | Device-measured: FCM/WS receipt log → `WarningNotifier.notify` → audible onset (logcat + video timestamp); drill path only (`test-alert`/local-test, no `event_state`, per `AlertRaiser.kt` drill fence) | Same single POCO + any additional owner devices if offered; matrix locked/in-use/killed-app; record `validity_ms` honored/expired per D-018 | Whether provisional gain survives to the ear; T_min defensibility; D-018 short-validity sizing for provisional | Vendor distribution (one vendor = UNKNOWN, stated); release-build behavior (debug-only drills) |
| M8 | E4 armed-dispatch cost: admin-status read (≤2 s budget, `admin_node.go:168-196`) + `FCMTokensWithin(20 km)` lookup | Quantifies the no-regrets E4 shave (prefetch/precompute at PRELIM, alarm still FINAL-gated → zero FP change) | `GetAdminNodeState` duration + `FCMTokensWithin` duration (`store.go:675-692`; boundary-tested `fcm_tokens_radius_test.go:53-92`) measured in test harness with seeded latencies | Unit/benchmark harness only (existing `admin_node_test.go`, `trusted_local_test.go` patterns); never production | E4 implementation proposal sizing; async-budget adequacy | Production DB latency distribution (needs prod telemetry, out of scope) |

## 2. Local experiment plan (single bench ESP32 + laptop; never VPS/production)

Setup (fixed for all runs): SAME mount as deployment (photo + torque/fastener note); serial log on; video clock in frame; record LTA state (uptime vs 45 s warmup), cooldown state (60 s since last report), ambient note. Every run logs: peak `correctedMagnitude`, reported PRELIM/FINAL PGA (server rows matched later by `obs_seq`), STA/LTA at trigger/detrigger if serial exposes them, outcome row for `07-` D.

- Battery A (environmental, `11-A` verbatim classes): footsteps/doors/furniture/contact/appliances/wind/handling/I2C-wiggle/reboot/electrical, 3+ repeats, low force first. STOP rule (binding): any class ≥ 100 gal → H-3 falsified, file ledger, do NOT repeat harder. Fills FP denominator context (never a rate until longitudinal).
- Battery B (transfer curve): calibrated taps ascending on SAME mount; input vs reported PGA; clip headroom analytic (160 gal ≈ 0.16 g ≪ 4 g). Characterizes, never "simulates a quake."
- Battery C (offline detector port): public-waveform series through laptop port of `sensor.cpp` STA/LTA+baseline → waveform-PGA → reported-PRELIM/FINAL + detection latency mapping. On-device injection only with a safe harness + separate approval; offline output never presented as device behavior without transfer-point confirmation.
- Battery D (opportunistic real events): preserve serial + server rows + heartbeat track + audible timestamps; one population member each (§6), never a sole justification.
- M1 diagnostic runs inside Battery A idle gaps (60 s ISR counts at rest + post-event LTA recovery watch).

## 3. Replay/benchmark plan (offline; production const untouched)

1. Extract (after owner re-authorizes read-only production SELECTs under M1′/M6′ precedent: `default_transaction_read_only=on`, before/after `pg_stat_user_tables`, metadata archived): P4-M1 window + every UNCONFIRMED on NODE-52960B47 since `phase3-1.1/ic=5`, with §`04-...md` coverage block (drops UNKNOWN, NULLs, failed verifications, schema/`algo_ver` per row).
2. Rebuild timelines per `06-...md` §2 (ratchet/sticky/first-wins), evaluate at X ∈ {60,80,100,120,140,160} at BOTH instants (transition snapshot + D-037 edge instant), recording eligible/stage/`decided_at`/gap/firing-reason — the reason-at-PRELIM vs reason-at-FINAL columns Q2 needs.
3. Tabulate into `07-threshold-comparison.md` (A–F); print §`06` limits block on every output; group by `algo_ver` (V5); operator-asserted params printed first (D-013); `INDEPENDENCE_CELL_KM` contradiction → reject run.
4. Worked predictions pre-registered (not results): 004c5f65 eligible all floors at edge, gain 0 at stage (growth unlogged — M2 closes); 4fcc3374 eligible at 60 only; LIVE-EQ vacuous (denominator).

## 4. E4 minimal read-only instrumentation design (DESIGN-ONLY, not implemented)

Problem: intra-event crossing instant unlogged; E4 prefetch has no hook. Constraints: no contract/MQTT-schema change, no new emission, no threshold/decision change, no production behavior change without a follow-up decision.

- D1 serial-only crossing log (bench/diagnostic builds): while `eventInProgress`, on first `correctedMagnitude ≥ X` per X ∈ {60,80,100,120,140}, `Serial.printf("XING ... obs_seq=... xing=X t=<millis> pga=...")`; values kept in a volatile per-event struct, reset on detrigger; zero heap, zero publish path contact (`servePendingSlot` untouched, `firmware.ino:260-319`). Production builds compile it out (`#ifdef PRELIM_XING_LOG`, default OFF) — so the shipped detector is byte-identical in behavior.
- D2 zero-change M1 path: GPIO15 logic-analyzer capture (no firmware delta at all) — preferred first run; D1 only if analyzer unavailable.
- D3 server-side E4 arming (follow-up proposal, not this phase): PRELIM-queued prefetch of `GetAdminNodeStatus` + token-set precompute into a TTL cache keyed `event_id`, consumed (not trusted) by the FINAL edge; alarm still FINAL-gated → FP-neutral. This phase only measures the two lookup costs offline (M8); implementation needs its own D-number + `algo_ver`-neutrality memo (cache must not become evidence).

## 5. What is answerable locally vs what needs quakes vs what stays UNKNOWN (one node)

- Locally answerable now: M1 (rate), M4-healthy/M8 (latencies), M6-history (replay), M7-drill tails, Battery A/B FP-context + transfer, D1-design review (no device needed).
- Needs real earthquake data (opportunistic only): seismic growth curves (M2/M3-seismic), felt-outcome pairing, heartbeat-through-shaking, any CONFIRMED-path behavior (structurally unreachable one-node, S2).
- Fundamentally UNKNOWN with one node/site/vendor: population FP/confirmation/latency RATES (§6/S9); multi-node correlation; vendor/Doze distribution; site transfer to any other installation. Each stays UNKNOWN in `07-` until Phase F — recorded, never zero-filled.

## 6. Evidence required before ANY E2 implementation proposal

Gate (all must hold, else repeat INSUFFICIENT EVIDENCE): (a) M1 rate known + constants re-derived; (b) M2+M3 growth distribution over ≥ the replay corpus + new captures, with T_min gain testable; (c) Battery A complete at the candidate PRELIM bar with zero ≥-bar environmental eligible AND `07-` D non-UNKNOWN; (d) M5 continuity baseline (no silent-limiter surprises); (e) M7 tail sample supporting the D-018 provisional-validity sizing; (f) M6 table showing which history flips at which bar/stage; (g) owner-set T_min + FP/detection/robustness criteria (`10-...md` §1 adapted to provisional) MET on the above. Single events never suffice (§6).

## 7. Remaining governance decisions (none taken here)

1. Owner re-authorization for read-only production SELECTs (scope: windows in §3.1; precedent M1′/M6′ safeguards mandatory).
2. Approval to build (not deploy) the D1 diagnostic firmware variant on the bench node from the laptop.
3. `10-decision.md` §1 criteria adapted to provisional (T_min, FP window/counts, robustness) — PROPOSED→ACCEPTED.
4. D-036 amendment (or new D-number) for any future provisional branch + reason vocab + withdrawal rule + validity policy; D-037-equivalent exactly-once/no-bump text for a PRELIM edge.
5. Contract addendum for the provisional marker (additive/`omitempty`, old-build clear-safe).
6. `algo_ver` ruling V3/V6 (eligibility-meaning change under identical label: bump vs admin-version vs comparability case).
7. D-036/D-037 activation-record repair (ledger `R-009` tension) — independent of, and prior to, any E2 verdict closeout.
