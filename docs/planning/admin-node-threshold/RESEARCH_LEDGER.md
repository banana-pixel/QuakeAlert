# Research ledger (append-only — never rewrite, never delete)

- Purpose: every meaningful discovery, one entry, with evidence + confidence + impact + action. Contradictions APPEND; history is preserved.
- Status: open.
- Last updated: 2026-09-11 UTC.
- Format per entry: ID / timestamp / topic / finding / evidence-source / confidence / impact / action / related experiment / related decision / supersedes.

---

## R-001 — 2026-09-11 — Admin threshold

- Finding: production floor is 140.0 gal compile-time; 100 gal exists nowhere in prod code. 140 has no observed environmental false warnings at current install (small-n).
- Evidence: `server/internal/event/admin_node.go:30`; grep 2026-09-11; LIVE-EQ audit control event.
- Confidence: FACT (source) + observational (false-warning absence).
- Impact: 140 stays the production baseline during research; 100 is benchmark-only.
- Action: keep 140 untouched; build offline floor-parameterized harness.
- Experiment: benchmark matrix (§05). Decision: none (research).
- Supersedes: nothing.

## R-002 — 2026-09-11 — Eligibility

- Finding: floor is 1 of 7 gates (designated, verified, UNCONFIRMED, admin-contrib, FINAL, PGA≥floor, heartbeat≤5m). Any of the other six can be the limiter (e.g., LIVE-EQ stopped before PGA existed).
- Evidence: `admin_node.go:96-119`, reason vocab `:42-50`; LIVE-EQ AUDIT §8 (all PGA gates FAIL-vacuous).
- Confidence: FACT.
- Impact: replay must report WHICH gate fired per candidate, not just eligible counts.
- Action: §06 procedure step 2 records reason-at-X.
- Experiment: replay plan. Decision: none. Supersedes: nothing.

## R-003 — 2026-09-11 — PRELIM vs FINAL asymmetry

- Finding: PRELIM never warns locally by construction (`ADMIN_NOT_FINAL`); PRELIM PGA is peak-so-far (≈300 ms), FINAL is whole-event max. 100 gal not comparable across phases. Live proof: 1.13 → 288.36 gal same episode.
- Evidence: `admin_node.go:110`; `sensor.cpp:209-287`; `firmware.ino:260-308`; LIVE-EQ AUDIT §5 control (obs 90/91).
- Confidence: FACT + OBSERVED.
- Impact: 140→100 gains only via FINAL edge/transition; intra-event 100-crossing instant is UNLOGGED — key instrumentation gap.
- Action: §06 §3 + hardware plan C; consider future PGA-growth logging (separate decision, NOT this research).
- Experiment: replay + timing model. Decision: none. Supersedes: nothing.

## R-004 — 2026-09-11 — PGA meaning

- Finding: firmware PGA = `|vectorMagnitude−baselineEMA|` (gal), STA/LTA-gated, vector = `sqrt(x²+y²+z²)*0.1197`; STA α=1/50, LTA α=1/2000 (frozen in-event), clamp 0.5, trigger 4.0 & STA≥1.5, confirm 300 ms, detrigger 1.5/60 s, cooldown 60 s, warmup 45 s. Wire 4-decimal, server float64 max-ratcheting. Clip ±4 g (≈3923 gal) — no concern at 60–160.
- Evidence: `sensor.cpp:141-287`; `config.h:99-110`; `mqtt.cpp:152-163`; `event/tracker.go:699-740`.
- Confidence: FACT (source).
- Impact: external seismometer PGA ≠ ESP32 PGA — no numeric transfer without transfer-curve work.
- Action: §11-B/C transfer measurements.
- Experiment: hardware battery. Decision: none. Supersedes: nothing.

## R-005 — 2026-09-11 — Dispatch bounds hold at any floor

- Finding: trusted-local = token-only 20 km, never GeoTopic (even severe/no-token/guard-off), WS identical, audit without coords, validity per D-018.
- Evidence: `dispatch/event_frame.go:60-138`; `trusted_local_test.go:22-41`; `severity.go`.
- Confidence: FACT + tested.
- Impact: blast radius bounded — lowering floor scales WHO warns, not HOW FAR.
- Action: none (verify H-5 strings before any recommendation).
- Experiment: safety analysis. Decision: D-036 (existing). Supersedes: nothing.

## R-006 — 2026-09-11 — Positive + negative controls exist

- Finding: 288.36 gal FINAL → live ADMIN_ELIGIBLE + TOKENS_RADIUS_20KM (24 attempted/2 succeeded); 77.89 gal event → no local frame; M5.9 deep → zero rows (threshold-irrelevant denominator).
- Evidence: LIVE-EQ AUDIT §§5,8,13; P4-M6 archive (4fcc3374); emissions 94–96.
- Confidence: OBSERVED.
- Impact: replay has one positive, one boundary negative, one vacuous — enough to TEST the method, not to SET the floor.
- Action: §06 inputs in that order.
- Experiment: replay. Decision: none. Supersedes: nothing.

## R-007 — 2026-09-11 — Sample rate UNKNOWN

- Finding: STA/LTA comments assume 100 Hz but code sets `setRate(99)` + DMP/FIFO/ISR path; TRUE GPIO15/DMP rate never measured. All time constants scale with the true rate.
- Evidence: `config.h:100-101`; `sensor.cpp:65-83,308-326`.
- Confidence: UNKNOWN (explicit, not guessed).
- Impact: latency model + H-6 hang on one measurement.
- Action: §11 diagnostic (ISR counter, laptop-side).
- Experiment: timing diagnostic. Decision: none. Supersedes: nothing.

## R-008 — 2026-09-11 — algo_ver gap

- Finding: `algo_ver phase3-1.1/ic=5` encodes only independence radius, NOT the admin floor. A floor change under the same label would violate V3/V6 interpretability.
- Evidence: `event/store.go:26,71-73`; PROJECT_RULES §11; D-006.
- Confidence: FACT.
- Impact: recommendation must include an explicit version ruling (bump / admin-version / comparability case).
- Action: carry to `10-decision.md` §1 + `08-safety-analysis.md` §4.
- Experiment: none. Decision: BLOCKED on owner version ruling. Supersedes: nothing.

## R-009 — 2026-09-11 — Governance tension recorded

- Finding: D-036 "adopted, NOT activated" (DECISIONS, 3 preconditions unmet 2026-09-09) vs CURRENT_STATE + LIVE-EQ audit (binary 8e93c4d + schema 10 deployed, sole holder NODE-52960B47, live ADMIN_ELIGIBLE 2026-09-11). D-037 edge has code+deploy but no DECISIONS entry found.
- Evidence: `DECISIONS.md:1042-1052`; `CURRENT_STATE.md:48-51,433-445`; LIVE-EQ AUDIT §§4,13; `git log 8e93c4d "feat(d-037)"`.
- Confidence: FACT (documents disagree — reported, not reconciled, per PROJECT_RULES §5).
- Impact: threshold verdict cannot close until activation/decision records are repaired by owner; research proceeds on code-as-deployed.
- Action: owner closeout (activation verification + D-037 entry) outside research.
- Experiment: none. Decision: D-036/D-037 (pending). Supersedes: nothing.

## R-010 — 2026-09-13 — Problem-2 PRELIM provisional verdict

- Finding: PRELIM is the earliest actionable point (ingested PRELIM reaching UNCONFIRMED) but decides on strictly less evidence than FINAL (peak-so-far vs whole-event max; future peak/duration/outcome inherently unavailable). Same-frame/same-event_id/same-revision provisional with additive marker + short D-018 validity + explicit RESOLVED withdrawal is the minimal D-003/D-018/D-019/D-020-compatible architecture (E2); advisory-tier provisional (E3) cannot warn by deliberate client design; PRELIM-armed early dispatch + crossing-instant logging (E4) is no-regrets instrumentation. 60-gal interaction multiplies sensitivity with FP exposure; phase-incomparability forbids treating a PRELIM bar as a lower FINAL bar. Evidence INSUFFICIENT to implement provisional warning now.
- Evidence: `12-prelim-early-warning-research.md` §§A–I + decision matrix; `admin_node.go:96-119`; `tracker.go:407-427,212-235`; `emit.go:62-119`; `event.go:91-206`; `event_frame.go:60-138`; `AlertRaiser.kt:44-112`; `AlertGate.kt:138-204`; `sensor.cpp:203-287`; ledger R-003/R-005/R-007/R-008; findings F-03/F-04.
- Confidence: FACT (mechanism/blockers) + INFERENCE (E2/E4 recommendability) + UNKNOWN (growth curve, battery, rate, tails — listed §12 unknowns 1–8).
- Impact: no production change; next work is E4 logging + `04/05/06` replay/benchmark at candidate PRELIM bars + `11-A` battery + rate/validity/withdrawal design before any D-036-amendment proposal.
- Action: keep FINAL-only production; run missing measurements; return with KEEP / TEST FURTHER / ADOPT or repeat INSUFFICIENT EVIDENCE.
- Experiment: benchmark matrix + replay with reason-at-PRELIM vs reason-at-FINAL + hardware battery. Decision: BLOCKED on owner criteria (`10-decision.md` §1 adapted), version ruling (V3/V6), contract addendum, withdrawal rule. Supersedes: nothing.

## R-011 — 2026-09-13 — Problem-2 evidence phase scoped

- Finding: eight measurements (M1 sample rate, M2 crossing instants design-only, M3 growth history, M4 ingest distribution, M5 heartbeat continuity, M6 reason-at-X replay, M7 audible tails, M8 armed-dispatch cost) plus bench batteries A–D defined in `13-evidence-plan-prelim-timing.md`; E4 instrumentation specified as serial-only `#ifdef`-gated design (default OFF, zero publish-path contact) with zero-change GPIO15-analyzer path preferred; single-node answerability separated (local now vs opportunistic-quake vs fundamentally-UNKNOWN); E2 gate restated as seven conjunctive conditions.
- Evidence: `13-evidence-plan-prelim-timing.md` §§0–7; `sensor.cpp:141-326`; `firmware.ino:260-319`; `config.h:99-130`; `input.go:49-70`; `tracker.go:212-235,407-427,699-740`; `admin_node.go:96-197`; `event_frame.go:60-138`; `store.go:111-137`; `store/admin_node.go:136-168`; `dispatcher.go:35-37,118-122`; LIVE-EQ AUDIT (344 lines, triple-zero window, prod `8e93c4d`/schema 10).
- Confidence: FACT (paths/pins) + INFERENCE (M-priority, E4 FP-neutrality) + UNKNOWN (all eight M-results — plan only, nothing executed).
- Impact: no production/code/contract/threshold/`algo_ver`/VPS/firmware change; next action is owner re-authorization for read-only SELECTs + bench diagnostic approval + adapted criteria acceptance.
- Action: run §§2–3 in order (M1 → M6-history → M8 → batteries → M2 capture → M7), populate `07-` C/D, then return KEEP / TEST FURTHER / ADOPT or repeat INSUFFICIENT EVIDENCE.
- Experiment: this plan IS the experiment definition. Decision: BLOCKED on seven governance items (`13-` §7). Supersedes: nothing.

## R-012 — 2026-09-13 — M1 CHECKPOINT 1: rate unmeasurable from agent environment

- Finding: M1 (true GPIO15 ISR / DMP packet rate, zero-change analyzer path) could not be executed here: no serial device (`/dev/ttyUSB*`, `/dev/ttyACM*` absent; only console `tty*`), no USB peripheral beyond QEMU tablet/hub (`lsusb`), no PlatformIO/esptool toolchain, no analyzer. No firmware touched, flashed, or read; nothing inferred from `setRate(99)` or "100 Hz" comments per the checkpoint rule.
- Evidence: environment inspection 2026-09-13 (`ls /dev`, `lsusb`, `which platformio/pio/esptool.py`, `adb devices` — all negative); `git status`/`git diff` clean over prod paths (see CHECKPOINT 1 report).
- Confidence: FACT (environment observation) → rate remains UNKNOWN; H-6 unresolved.
- Impact: M1 must run from the laptop + bench side with analyzer/scope on GPIO15 per plan §1-M1; fallback diagnostic-counter build still needs its separate approval and was NOT attempted.
- Action: owner/laptop-side execution of CHECKPOINT 1; this agent proceeds to nothing automatically (STOP observed).
- Experiment: M1 (blocked here). Decision: none. Supersedes: nothing.

## R-014 — 2026-09-13 — CHECKPOINT 2: M3/M4/M6 read-only production results

- Finding M3 (growth gaps, 36 v2 episodes: 32 complete pairs + 4 PRELIM-only + 25 v1 FINAL-only rows): `received_ts(FINAL)−received_ts(PRELIM)` min 1145 ms / p50 ≈5440 ms / max 18416 ms (n=32); `dur_ms` delta tracks gap within ~10 ms except 3 late-publish anomalies (ep 1507336: gap 11208 vs dur 1190; ep 1507331: 9614 vs 7128; ep 1507340: 2940 vs 1400 — FINAL publish lagged detrigger by ~1.5–10 s, mechanism UNKNOWN, consistent with offline-buffered first attempt). PRELIM-only (FINAL never received): obs 56 (79.02 gal), obs 63 (207.21 gal), obs 64 (61.42 gal), obs 79 (9.43 gal, below floor, DETECTED-only). Zero duplicated phases; all attempt_no=1.
- Evidence: read-only SELECTs (schema 10, `transaction_read_only=on` verified, pg_stat write counters byte-identical before/after: 96/50/88/93 + 0/50/0/0 upd), canonical order `received_ts, observation_id`; raw CSVs agent-`/tmp/opencode/cp2_{obs,log,emis}.csv` (ephemeral, uncommitted); offline `cp2_analyze.py`.
- Finding M4 (transit, n=93): `received_ts−publish_ts` min 6 / p50 32 / max 10066 ms (max = obs 39 FINAL 4.26 gal, attempt_no=1 — NOT a retry, cause UNKNOWN); attempt_no>1: 0 rows; verify_result≠OK: 0 rows; negative transit: 0 rows.
- Finding M6 (reason-at-X at rev1, 44 events, all `phase3-1.1/ic=5`, all DETECTED→UNCONFIRMED FLOOR_MET → RESOLVED NO_NEW_EVIDENCE): PGA≥X counts at rev1 = 60:28 / 80:22 / 100:17 / 120:13 / 140:12 / 160:10 (counts, NOT rates). EIGHT live trusted-local ALERTs (`TOKENS_RADIUS_20KM`, 23–24 attempted / 1–2 succeeded) prove BOTH paths in production: transition path ×5 (0ac4cea8, d95751d8, 443dcff0, 5c119916, 004c5f65 — rev1 contrib already FINAL, ALERT ≤3 ms after advisory) and D-037 edge path ×3 (edde3105 +15034 ms, 5793b269 +16892 ms, eb89b5da +4421 ms — ALERT lag equals the M3 gap, rev1 contrib PRELIM). Negative controls: e3bc939d FINAL 138.21 <140 → no ALERT (boundary miss 1.79 gal); da6a5cae + 3d872c06 PRELIM-only (79.02 / 207.21 gal, FINAL never arrived) → no ALERT ever — the exact stranded-provisional archetype: at a 60 PRELIM bar, 7 of 11 post-designation PRELIM-phase rev1s would have alarmed, 2 of those 7 (28% IN HISTORY, n=11, NOT a rate) with never any confirmation.
- Confidence: OBSERVED (rows, gaps, emissions) + INFERENCE (stage attribution via decided_at deltas matching M3 gaps; designation split at ≈1788988500000 ~21:15Z Sep 9, approximate — boundary flips flagged none within ±2 h). UNKNOWN: heartbeat-at-the-time for all 44 (no history; the 8 ALERTs prove freshness at those 8 instants only); verified/designated-at-the-time for pre-split rows (taken as-recorded per CURRENT_STATE); causal attribution (membership-and-time, D-015); `ledger_drops` (log-only).
- Impact: M3 PASS (bounded gain now measured: E2 saves 1.1–18.4 s of T5 wait, p50 ≈5.4 s — before T8/T9 tails); M4 PASS (healthy transit p50 32 ms; tail exists, unexplained); M6 PASS (table + both-paths-live proof + stranded-provisional datum). `07-` columns A/B/C fillable from this; D stays UNKNOWN (no battery yet).
- Action: STOP observed; next CHECKPOINT 3 (M8 offline, no approval needed beyond this workflow).
- Experiment: M3/M4/M6. Decision: none (no threshold/policy chosen; counts are not rates). Supersedes: nothing (refines R-006 anchors into a 44-event corpus).

## R-015 — 2026-09-13 — CHECKPOINT 3: M8 offline armed-dispatch cost

- Finding: pure evaluator + frame build cost ≈ microseconds (all `TestEvaluateAdminNodeEligibility`/`TestTrustedLocalFrameFor`/`TestReplayAdmin` subtests 0.00 s); full edge round-trip (ingest→edge→async emit→fake sink) bounded <50 ms in-harness (edge tests pass with 50 ms waits; slowest `TestAdminEdge_IneligibleEmitsNone` 0.25 s = 5 subtests × sleep); dispatch-fake local tests (`TestTrustedLocalUses20KmRadius`, `...SevereSingleNodeNeverGeoTopic`, `...NoTokensMeansNoAudience`, `...WithoutFCMSender`, `...D019OutcomeExactlyOnce`) all PASS 0.00–0.01 s. Store DB tests (`TestGetAdminNodeStatus*`, `TestFCMTokensWithin_*`) SKIP — no local Postgres here, none provisioned (out of scope). Seeded-latency harness NOT built (would require new test files = server change, forbidden this checkpoint).
- Evidence: `go test ./internal/event/ -run 'TestAdminEdge|TestEvaluateAdminNode|TestTrustedLocalFrame|TestReplayAdmin'` → ok 0.552 s; `go test ./internal/dispatch/ -run 'TrustedLocal|TestDispatch'` → ok 0.017 s; store run → 4 SKIP. Zero repo files added/modified (test runs only).
- Confidence: OBSERVED (timings) for in-harness costs. UNKNOWN: production `GetAdminNodeStatus` single-SELECT latency and `FCMTokensWithin` PostGIS 20 km latency (needs prod telemetry — explicitly out of scope per plan §1-M8).
- Impact: 2 s async budget adequate IN-HARNESS with large headroom (slowest measured round-trip 0.25 s incl. deliberate sleeps); E4 prefetch benefit UNQUANTIFIED (saves at most two prod lookups of unknown latency) — E4 sizing stays INSUFFICIENT EVIDENCE, FP-neutrality claim unaffected (E4 never gates the alarm).
- Action: STOP observed; next CHECKPOINT 4 (local batteries — needs bench-node safety re-scope per R-013 flag before any shaking).
- Experiment: M8. Decision: none. Supersedes: nothing.

## R-016 — 2026-09-13 — CHECKPOINT 4 prep: battery safety re-scope (no shaking performed)

- Finding: no bench/offline/dry-run mode exists in firmware (grep bench/offline/dry-run/suppress: nil; publish gated only on WL_CONNECTED + NTP + HMAC — `mqtt.cpp:98-148,377-378`); broker + WiFi + identity are per-node NVS provisioned via wizard/AP-`/config` flow (`network.cpp:116-241`, keys `ssid/password/HMAC/station_id/mqtt_broker/port/tls`). Proposed safe setup: (1) PRIMARY — power bench node where none of its saved SSIDs is visible: zero device change, detection + serial PRELIM/FINAL logic run, publish fails closed with backoff; needs owner confirmation bench ≠ deployment-WiFi coverage. (2) ALWAYS-SAFE — Battery C offline port, no device contact. Approval-gated alternatives (NOT recommended now): NVS re-provisioning (device config change + production-provisioning loss risk), server-side mute (VPS change, forbidden), distinct bench identity (still pollutes prod ledger unless broker also moved).
- Evidence: source grep 2026-09-13 (pins above); M1 wire log (bench unit = NODE-52960B47 on prod broker).
- Confidence: FACT (source) + INFERENCE (option ranking).
- Impact: CHECKPOINT 4 cannot proceed until owner confirms (a) bench-network separation for option 1, or selects option 2, plus (b) identity question: bench unit == relocated production unit, or second unit sharing NODE-52960B47 keys (collision risk on `obs_seq`/dedup if both ever publish). INFERRED honesty note: current "admin heartbeat fresh" may reflect the BENCH, not the deployment site — M5 + the 8 live ALERT freshness claims inherit this location ambiguity until resolved.
- Action: STOP observed; zero shaking performed; await owner answers + explicit CHECKPOINT 4 approval.
- Experiment: none (prep only). Decision: BLOCKED on owner confirmation. Supersedes: nothing.

## R-017 — 2026-09-15 — CHECKPOINT 4 open: Battery A capture running, private-testing framing

- Finding: owner declares system PRIVATE TESTING (not publicly released) with a planned pre-launch DB clean/reset — NODE-52960B47 battery rows are PRIVATE TEST DATA; R-013/R-016 pollution blocker lifted by owner decision (DB reset covers it; release procedure in new `14-release-readiness-checklist.md`, guideline only, nothing executed). Battery A serial capture live: laptop `/tmp/battA_serial.log` since epoch 1789473197.524 (2026-09-15T11:53:17Z), 40-min window, line-buffered (fixed 0-byte background bug: file objects ignore `python -u`; `buffering=1` required). Node rebooted on capture open (expected per-open behavior; Boots: 55), warmup 45 s → detection live ~11:53:20+45 s; MQTT Connected 11:53:33Z after one NTP retry; INT_RATE 100.02 (M1 reconfirmed). Boot count 44 (M1) → 55 accounting: ~4 agent port-opens + owner-side power/USB cycles (attribution approximate).
- Evidence: capture log (live, laptop-side); `14-release-readiness-checklist.md` (new, §1–§10 guideline); owner private-testing declaration this turn.
- Confidence: FACT (setup/log) + INFERENCE (warmup math) + UNKNOWN (all battery outcomes pending).
- Impact: batteries proceed one class at a time per `11-A` (footsteps → electrical; reboot class DEFERRED to last — unplug kills capture+power); ≥90 s class spacing (60 s firmware cooldown + margin); ≥100 gal STOP rule binding; NTP wobble noted — if NTP never syncs, publish aborts and server rows go missing while serial still logs (watch item for row matching).
- Action: owner performs Battery A classes with wall-clock announcements; agent analyzes per class next turns; STOP after checkpoint.
- Experiment: Battery A (open). Decision: owner framing accepted (private testing + future reset). Supersedes: R-013/R-016 blocker portion (pollution concern) — identity/heartbeat flags stand.

## R-018 — 2026-09-15 — CHECKPOINT 4 PAUSED (owner-action pause, NOT a failure)

- Finding: owner away from bench — no physical stimulus possible. Batteries NOT started; zero classes performed; zero shaking by anyone. Capture disposition: laptop UNREACHABLE at pause verification (Tailscale ping 2/2 timeout — likely asleep/offline), so live capture state could not be confirmed; last confirmed state was healthy logging at 11:53:48Z (`/tmp/battA_serial.log` growing, 40-min window to ~12:33Z). Nothing cleaned, deleted, modified, or restarted by anyone this turn — evidence left exactly as-is for later recovery.
- Evidence: pause order this turn; ping transcript (0/2 replies); prior R-017 window-open record.
- Confidence: FACT (pause + unreachability observed). UNKNOWN: whether the capture ran to its window end or died with laptop sleep — resolvable on resume by reading the log tail (footer `# end epoch` present = clean finish; absent = interrupted, data up to last line still valid).
- Impact: no data loss risk beyond the already-captured idle baseline; resume = verify log tail + (if needed) fresh capture + reboot-aware warmup, NOT a phase restart. M1/M3/M4/M6/M8 results stand unaffected.
- Action: STOP observed; await owner back-at-bench signal. On resume: (1) confirm laptop awake, (2) `wc -l` + tail the log, (3) restart capture only if window expired, (4) continue Battery A class 1 (footsteps).
- Experiment: none (pause). Decision: none. Supersedes: nothing.

## R-013 — 2026-09-13 — M1 CHECKPOINT 1: ISR rate MEASURED 100.0 Hz via pre-existing INT_RATE lines

- Finding: bench ESP32 (`/dev/ttyUSB0`, CH340, owner-confirmed bench sensor) already prints `INT_RATE window_ms≈10000 INT_EDGES=1000/1001 INT_HZ=99.96–100.10` + `INT_DIAG` every ~10 s — zero firmware change needed. 7 consecutive windows post-boot: 100.09, 99.96, 100.10, 100.01, 99.97, 100.05, 100.01 Hz (mean ≈100.03). Firmware identity on wire: `System Ready (7.0.0) [Boots: 44]`, `Station ID: NODE-52960B47`, prod TLS broker, NTP synced, location −6.856209,107.528961. H-6 NOT falsified (100 Hz assumption holds within ~0.1% over 70 s).
- Evidence: raw serial log (75 s, 30 lines, laptop `/tmp/m1_serial.log`, pulled to agent `/tmp/opencode/m1_serial.log` — ephemeral, NOT committed); capture script (agent `/tmp/opencode/m1_capture.py`, laptop copy `/tmp/m1_capture.py`); baud 115200 from `firmware.ino:390` + `platformio.ini:20`.
- Confidence: OBSERVED (wire text) for rate over one 70 s post-boot window, one unit. UNKNOWN: long-term drift/jitter, other units, DMP-packet-vs-ISR 1:1 (edges counted = ISR; packet path assumed same semaphore path per plan).
- Impact: M1 PASS (bounded: single-window, single-unit); T1 constants stand; M2/M3 timing model keeps 100 Hz basis with stated ±0.15 Hz window tolerance.
- Deviation (honest): opening the port rebooted the bench node (boot banner 0.5 s after open, `BOOT millis=2289`, boot count 44) despite `dtr=False/rts=False` — CH340 asserts on open. Unintentional, bench-only, no production contact; warmup restarted, no event in flight (no PRELIM/FINAL lines in window).
- Safety flag for CHECKPOINT 4: bench node identifies as NODE-52960B47 against the PRODUCTION broker — any battery shaking will ingest as production observations. Batteries must be re-scoped (unplug-network/observe-serial-only, or owner-accepted production-window accounting) before CHECKPOINT 4 proceeds.
- Action: STOP observed; next is CHECKPOINT 2 (still blocked on production-SELECT re-authorization).
- Experiment: M1. Decision: none. Supersedes: nothing (R-012 environment-block refined: agent-side unobservable, laptop-side measurable).

## R-019 — 2026-09-26 — Offline floor-sweep harness built + faithfulness proven (owner chose "measure the band first")

- Finding: built `server/internal/../scripts/admin_floor_sweep.go` (`//go:build ignore`, `package main`, read-only) — the floor-parameterized replay harness 05-experiment §3 / 10-decision §3 step 3 call for. It reuses the SAME read-only path as `replay_window.go` (`ListObservationsForReplay` → `event.Replay`, Tracker instance without persister) and carries a deliberate line-for-line COPY of `EvaluateAdminNodeEligibility` (`eligibleAtFloor`) whose ONLY difference is `floor` as a parameter instead of the `AdminNodeMinPGAGal=140.0` constant. Production floor and all production code UNTOUCHED (D-007 honored; the const is not read at runtime by the harness, it is copied). Before reporting any sweep the harness runs `assertCopyFaithful`: it evaluates the copy at 140 against `res.AdminOutcomes` (the PRODUCTION evaluator that `Replay` already ran) frame-by-frame and ABORTS on any divergence — a drifted copy cannot report. `SELFTEST=1` mode (no DB) proves the copy reproduces `EvaluateAdminNodeEligibility` exactly across all 11 gate branches (below-floor, ==140, PRELIM/NOT_FINAL, NOT_UNCONFIRMED, NO_CONTRIBUTION, HEARTBEAT_STALE, eligible) and demonstrates the {60,80,100,120,140,160} sweep. Sweep answers DETECTION only.
- Evidence: `go vet scripts/admin_floor_sweep.go` clean; `SELFTEST=1 go run` output 2026-09-26 (11/11 gate cases copy==production; demo sweep: 77.89→eligible only at ≤60 of the candidates, 100→≤100, 288.36→all). Anchored predictions the tool confirms from documented FINAL peaks: 004c5f65 (FINAL 288.36) eligible at every candidate floor; 4fcc3374 (FINAL 77.89) eligible only at floor ≤77.89 → among candidates, 60 alone (this SHARPENS the loose "≤80" wording in `04-data-plan.md`: at floor 80 the 77.89 event is ADMIN_PGA_BELOW_FLOOR).
- Confidence: FACT (harness faithfulness proven by self-check + SELFTEST); the population detection numbers remain UNKNOWN until the harness is run over the extracted windows (owner-gated read-only run — this environment has no production DB).
- Impact: closes 10-decision §3 step 3 "build the offline floor-parameterized harness" as a TOOL; does NOT populate `07-threshold-comparison.md` column A for the population yet (needs the read-only run). Does NOT touch metric C (latency — unmeasurable, F-04 growth curve unlogged) or metric D (false-warning — needs the bench disturbance battery, CHECKPOINT-4 class). Detection is one of six metrics; a floor verdict still requires D + owner criteria approval (10-decision §1). Recommendation stays INSUFFICIENT EVIDENCE / floor 140.
- Action: harness ready. Owner runs it over the M1′ window + all UNCONFIRMED-on-NODE-52960B47 windows under a read-only role (`DATABASE_URL` to a replica or `default_transaction_read_only=on`), pastes output → I fill 07 column A. The decisive 100–140 false-warning row still needs the bench battery (metric D).
- Experiment: floor-sweep (replay side of 05/06). Decision: none — tool only, no threshold change. Supersedes: nothing.

## R-020 — 2026-09-26 — Battery run-sheet written; measurement REQUIRES WiFi ON (resolves the offline-readout question), safe at the live 140 floor

- Finding: resolved the open readout question from CHECKPOINT-4 prep. The FINAL peak PGA is serial-printed ONLY on a successful MQTT publish (`mqtt.cpp:183-189` `Trigger published (FINAL … pga=%.4f …)`); a failed publish prints `Trigger publish failed!` with NO number (`mqtt.cpp:193`), and `pendingReport.maxPga` is never printed elsewhere (`sensor.cpp:267` sets it silently). Therefore the R-016 "run with WiFi OFF" primary would LOSE the metric-D number entirely — the battery MUST run with WiFi ON to read PGA. This is safe at the current production floor because the live trusted-local alarm gate is FINAL ≥ 140 gal (D-037) and the band under measurement for the 70-gal question is [70,140), strictly BELOW the alarm floor: those disturbances are recorded (serial + DB rows) but cannot fire a real alarm. Only a ≥140 gal hit would alarm; STOP rule holds a 40-gal margin (record ≥70, STOP class ≥100, abort battery near ~130).
- Evidence: `mqtt.cpp:150-195` (publish-path print, fail-closed branch); `sensor.cpp:249-288` (detrigger sets `pendingReport.maxPga=pga`, no print); `sensor.cpp:304` (LED = WiFi status). Wrote `scripts/battery_runsheet.md` (safety key, capture setup, class order realistic→stress, STOP rules, serial lines to watch, recording template, send-back → sweep) + committed `scripts/battery_capture.py` (was ephemeral in R-017; now a proper read-only artifact).
- Confidence: FACT (firmware print paths read from source). The disturbance PGA population is UNKNOWN until the owner runs the battery — that is the point of the run.
- Impact: unblocks the owner to execute metric D themselves. Closes the readout ambiguity that R-016/R-017 left open (WiFi-off is NOT usable for the number). The empty 100–140 gal / 70-gal row in `08-safety §2` stays empty until the log arrives; recommendation stays INSUFFICIENT EVIDENCE / floor 140. The sweep harness (R-019) is the analysis step for the battery window (each confirmed disturbance → UNCONFIRMED frame → eligibility at 70/80/…).
- Action: owner runs `battery_capture.py`, performs the classes per the run-sheet, sends the log + FROM_TS/TO_TS. I then read the raw FINAL numbers and run `admin_floor_sweep.go` over the window to tabulate would-be eligibility at 70.
- Experiment: Battery (metric D, CHECKPOINT-4 class). Decision: none — procedure + readout resolution only, no production/firmware/floor change. Supersedes: R-016 "WiFi-OFF primary" readout assumption (WiFi ON required for PGA; safety re-established via the 140-floor margin rather than network isolation).

## R-021 — 2026-09-26 — Deep pre-execution audit of the two-window PRELIM architecture: INSUFFICIENT EVIDENCE, blocked on a growth curve that does not exist

- Finding: audited the proposed PRELIM-1 (~300 ms) / PRELIM-2 (~1–2 s) / FINAL architecture end-to-end via three read-only fact-extraction passes (firmware+timing, server pipeline, Android+contracts+docs), cross-checked against direct reads of `sensor.cpp`/`config.h`/`mqtt.cpp`. Wrote `15-two-window-prelim-audit.md` (22 sections + Decision Gate, every claim labelled FACT/OBSERVATION/INFERENCE/UNKNOWN). Verdict: INSUFFICIENT EVIDENCE to implement; NO-GO on firmware/server/Android/production; conditional-GO only for the *design* of an offline experiment. Three drivers: (1) the stored PRELIM is a single ONSET-INSTANT sample at the 300 ms confirmation (`sensor.cpp:214,227`), FINAL is the peak (`:267`) — exactly two points per event, NO intra-event growth curve exists, so "what happens at 1 s / 2 s / 3 s?" is unanswerable from history; (2) "two windows" conflates a second sample-TIME with a second decision-THRESHOLD — both are single scalars a single re-timed PRELIM already exposes, so PRELIM-2 is a genuinely new concept only if it is a DECISION (persistence/stability), which is unspecified (request Q4 lists 4 candidate meanings); (3) every path that would let a provisional frame alarm is closed by design (`classify()` ignores phase `classify.go:10-21`; Android drops all pre-confirmation frames `AlertRaiser.kt:71`) and the false-warning evidence to open them is empty — worse, the planned metric-D battery measures FINAL not the intra-event curve a provisional window decides on.
- Evidence: `15-two-window-prelim-audit.md` §§1–22; source pins throughout (sensor/config/mqtt firmware; classify/admin_node/tracker/event/emit/event_frame/store/replay server; AlertRaiser/AlertGate/WsAlertMessage/AlertMappers Android; trigger/alert schemas; docs 04–13).
- Flagged discrepancy: NO sample-rate diagnostic (`INT_RATE`) exists anywhere in the repo firmware (`sensor.cpp`/`firmware.ino`/`mqtt.cpp`) — the rate is `setRate(99)` with comments asserting 100 Hz (`config.h:100-101`). This CONTRADICTS R-013's claim that the bench node "already prints INT_RATE." Either R-013 observed a different (uncommitted) build or overstated the source. True GPIO15/DMP rate recorded as UNKNOWN (agrees with F-10), which leaves the "1–2 s" window uncalibrated.
- Also flagged: contract + firmware comments call today's PRELIM "peak-so-far" (`trigger.schema.json:55`, `sensor.cpp:223-225`) but the code assigns an onset-instant sample (`:214,227`) — a pre-existing documentation defect to correct before any PRELIM-2 design. And a provisional-only (never-CONFIRMED) alarm would get NO FCM all-clear (`emit.go:112` sends stand-down only to the ever-CONFIRMED audience) — a stale-warning risk.
- Confidence: FACT for all cited code behaviours; UNKNOWN for the growth curve, sample rate, and any false-warning rate; INFERENCE for the "two knobs" and consensus-separation arguments.
- Impact: does not change production; no D-number/algo_ver/contract created (audit-only per owner). Establishes the Decision Gate: 7 evidence items required before implementation, centred on capturing the intra-event growth curve via serial-only default-OFF instrumentation, an instrumented disturbance battery, and an offline replay showing two-window beats single re-timed PRELIM on the latency/FP frontier. Recommendation stays INSUFFICIENT EVIDENCE / floor 140.
- Action: audit delivered to owner. Next step is owner's — decide whether to fund the offline experiment design (instrumentation + curve capture) or hold. No implementation begins until the Decision Gate is satisfied.
- Experiment: pre-execution audit (Problem-2 architecture). Decision: none — audit only, no production/firmware/threshold/contract change. Supersedes: nothing (extends `12-`/`13-` Problem-2 research; refines R-013's INT_RATE claim as source-unconfirmed).

## R-022 — 2026-09-27 — Public-waveform offline replay of the intra-event growth curve: later window recovers substantially more of FINAL (PRELIMINARY); two-window unsupported over re-timing

- Finding: executed the offline replay that R-021's Decision Gate called for on PUBLIC data (the real-device curve is still owed). Built `scripts/replay_growth.py` — a faithful T2 QuakeAlert-analog detector (exact `config.h` constants: BASELINE_ALPHA 0.005, STA 1/50, LTA 1/2000, clamp 0.5, trig 4.0, detrig 1.5, MIN_STA 1.5 gal, confirm 300 ms, max-event 60 s) fed real FDSN strong-motion (IRIS, CI network, HN? 100 sps, response-removed → gal). 52 records retained across Ridgecrest M7.1/M6.4 2019 + La Habra M5.1 2014; dist 7.1–132.1 km; FINAL(T2) 4–243 gal. RESULT (B, on the analog): the current onset-instant PRELIM holds a median 7 % of FINAL; running peak recovers ~27 % by +0.3 s, ~42 % by +1.0 s, ~49 % by +2.0 s, ~71 % by +3.0 s — monotonic, most gain in the first ~1 s, but even +3 s median <1.0 (PGA arrives late; time-to-90 %-of-FINAL median 4.4 s). Threshold-crossings (60/80/100/120/140 gal, COMPARISON ONLY): ZERO records cross any level at +0.3 s; crossings accrue with the window (median time-to-cross 2.9–4.4 s). Answers: (1) later window gives substantially more amplitude info — YES, PRELIMINARY; (2) still useful warning time — INSUFFICIENT EVIDENCE offline (turns on device→user latency, §14 UNKNOWN); (3) second window necessary — NO indication it beats simply re-timing the single PRELIM (option B), favouring the low-complexity path over the [[15-two-window-prelim-audit]] NO-GO two-stage.
- Evidence: `16-two-window-prelim-public-waveform-research.md` §§10–12,15–18; `scripts/replay_growth.py`, `scripts/smoke_fdsn.py`; `scripts/out/growth_summary.json` + `growth_records.json` (exact retained/excluded station IDs). Toolchain: py3.11 venv `/home/ubuntu/.venv-waveform`, obspy 1.5.1. Feasibility proven end-to-end (smoke test: CI.LRL Ridgecrest, ~184 gal horizontal, 100 Hz).
- Bias/limits (C): retained set is conditioned on T2-confirmation (30 no-confirmation exclusions are outcome-correlated; the other 296 exclusions are `FDSNNoDataException` archive gaps, uncorrelated). n=52 from 2 S. California source regions — small, regionally narrow. The T2 analog is NOT the MPU6050 (§8.1 non-equivalence: MEMS response/noise, true sample rate still UNKNOWN F-10, no on-device response removal, mounting) — relative growth dynamics transfer, absolute gal do NOT.
- Confidence: FACT for the computed distributions on the analog; PRELIMINARY / small-sample for the two-window verdict; INSUFFICIENT EVIDENCE for the warning-time-usefulness half and any absolute-gal / production claim.
- Impact: does NOT change production; no firmware/server/Android/MQTT/threshold/PRELIM-FINAL-semantic change; thresholds remain research-comparison-only; floor stays 140 (D-007). Partially discharges R-021 Decision Gate item "offline replay showing whether a later window beats the onset-instant PRELIM" — on PUBLIC data only; the real-device growth curve (serial `#ifdef` instrumentation, [[13-evidence-plan-prelim-timing]] E4) and the end-to-end latency budget remain owed before any option-B re-timing is designed.
- Action: report delivered. Next step is owner's — decide whether to fund the real-device growth-curve capture (§17) that closes the non-equivalence gap; no implementation begins until the Decision Gate is satisfied.
- Experiment: public-waveform offline replay (enabling + secondary questions of the two-window study). Decision: none — research only, no production/firmware/threshold/contract change. Supersedes: nothing (discharges part of R-021's gate; consistent with R-021 DISC-1 that PRELIM is an onset-instant sample).

## R-023 — 2026-09-27 — Release Readiness Master Audit: project-wide audit synthesised from three read-only source sweeps; verdict NOT READY (open source) / READY WITH CONDITIONS (operational)

- Finding: produced `docs/planning/release-readiness-master-audit.md` (project-wide, one level above this threshold area), an 18-section AUDIT/DECISION-FRAMEWORK synthesising three read-only opus-subagent sweeps (server/event-pipeline/Admin-Node; Android/MQTT/FCM/deploy/CI; secret-scan/README/cleanup/license) plus my own git and source verification. Overall verdict NOT READY, split into two releases: open-source repo publication NOT READY (four hard blockers) and operational single-node service READY WITH CONDITIONS. Decided the initial operational configuration for both algorithm knobs the owner asked about: hold the Admin Node local floor at 140 gal as a conscious initial operating point (not auto-preserve; reasons: strong-shaking semantics + the open all-clear defect argues against more alarms + unmeasured single-node FP cost) and keep PRELIM timing unchanged (life-safety warning gates on FINAL not PRELIM, and re-timing needs a firmware flash out of scope). Both paired with an owner decision + evidence matrix (§6, §17).
- Evidence: `release-readiness-master-audit.md` §§1–18; verified this session: git ignore/track status (session-ses_* + docs/planning NOT ignored; .gitignore only lists session.txt/*-hello.txt), LICENSE/README absent, domain mismatch (build.gradle.kts:127 `api.quakealert.web.id` vs deploy/contract `api.quakealert.id`), and the trusted-local all-clear asymmetry (emit.go:108-112 push=EverConfirmed; admin_node.go:176-179 fires only on UNCONFIRMED).
- Blockers recorded: (operational) trusted-local never-CONFIRMED alarm has NO FCM all-clear [FACT, verified] and production domain mismatch [FACT, DNS unknown]; (open-source) B1 real keys in untracked-not-ignored `session-ses_f6d9.md` (per secret-scan; NOT opened/reproduced/rotated by me), B2 no LICENSE, B3 no README, B4 .gitignore gap. Discrepancies reported not reconciled: D-036 record says "NOT activated" while CURRENT_STATE says live 2026-09-09T~21:15Z; PRELIM "peak-so-far" text vs onset-instant code (consistent with [[15-two-window-prelim-audit]] and R-021/R-022).
- Confidence: FACT for items I re-read this session (git state, domain, all-clear path, license/readme absence); MEAS (source-cited subagent output, not all re-read by me) for the broader server/Android/infra posture; PRELIMINARY for the public-waveform inputs (R-022); UNKNOWN for population FP/latency, DNS aliasing, and Firebase key restrictions.
- Impact: does NOT change production; nothing deployed/merged/pushed/flashed/rotated/cleaned. Audit-only; no D-number/algo_ver/contract created; floor stays 140 (D-007). Recommends owner decisions (§17 matrix) and an ordered release checklist (§15).
- Action: report delivered to owner. Next step is owner's — decide the two operational blockers, ratify the two algorithm decisions, and handle the exposed keys before any repo goes public.
- Experiment: none (project-wide release-readiness audit). Decision: none — audit only. Supersedes: nothing (extends the planning area upward to a project-wide doc; incorporates R-019/R-021/R-022).

## R-024 — 2026-09-27 — V1 floor DECIDED: Admin Node PGA floor set to 60 gal (D-038, sensitivity-first, reversible)

- Finding: owner made the §3 MASTER-prompt decision explicitly ("60 gal" chosen among 60/80/100/120/140). `AdminNodeMinPGAGal` lowered 140.0 → 60.0 in `server/internal/event/admin_node.go:30`, recorded as D-038 (ACCEPTED, owner-approved 2026-09-27). Rationale: sensitivity-first priority; locally-measured moderate shaking (~MMI V+) wakes the 20 km trusted-local audience earlier; false-warning rate monitored post-release rather than pre-empted. Remains compile-time const (D-007 rationale unchanged). Reversal criterion recorded in D-038: raise the floor if post-release trusted-local false-warning rate exceeds owner acceptance. Offline public-waveform replay (R-022, 52 FDSN records, comparison-only per §8.1 non-equivalence) frames the tradeoff: reach-ever counts rise as the floor falls (140→5, 120→7, 100→8, 80→12, 60→15), none crossing before ~1 s after onset.
- Evidence: `server/internal/event/admin_node.go` diff (const + D-038 comment); `docs/DECISIONS.md` D-038; eligibility boundary matrix + replay boundaries updated in `admin_node_test.go` / `replay_admin_test.go` / `admin_edge_test.go`; R-022 counts.
- Confidence: FACT for the code change and decision record; PRELIMINARY for the tradeoff framing (R-022 bias/limits carry over); UNKNOWN for the single-node false-warning rate at 60 (post-release observable).
- Impact: changes the production gate value (first threshold change in the programme). Quorum, independence, radius, CONFIRMED path unchanged. Tests: `go test ./internal/event/ -race` green 2026-09-27 (per D-038 text).
- Action: needs ledger/CHANGELOG catch-up (this entry), deliberate commit, rebuild + redeploy to take effect (const ships with the binary), post-release FP monitoring per reversal criterion.
- Experiment: Admin floor decision (§3 of MASTER prompt). Decision: 60 gal initial operational floor. Supersedes: R-023 §6.1 recommendation to hold 140 (owner override — audit text preserved, this entry records the override, nothing rewritten).

## R-025 — 2026-09-27 — V1 PRELIM timing DECIDED + IMPLEMENTED in source: running peak over first 1.0 s (D-039, single window)

- Finding: owner chose "Peak over first 1.0s" among 300ms/500ms/1s/1.5s/2s/3s (§4 MASTER prompt). PRELIM becomes the running peak of corrected acceleration over [confirmation, confirmation + 1.0 s], published once at window end (D-039, ACCEPTED). Replaces the onset-instant sample at 300 ms confirmation, correcting the code-vs-comment contradiction flagged as DISC-3/DISC-C (now fixed in code + contract rather than only reported). FINAL unchanged (true running peak at detrigger). Still exactly two publishes per event sharing one `obs_seq` (early-detrigger flush preserves the contract). Two-window PRELIM evaluated (docs 15-, 16-) and NOT adopted: R-022 showed a later single window recovers median ~7% → ~42% of FINAL at +1.0 s while a second window showed no advantage over re-timing — simplest supported architecture wins per owner directive. Confirmation timing (300 ms), FINAL semantics, detector constants unchanged.
- Evidence: `firmware/src/sensor.cpp` (`prelimPending` + `prelimWindowEndMillis` + `flushPendingPrelim`); `PRELIM_WINDOW_MS = 1000` in `config.h`; corrected descriptions in `canonical.h`, `state.h`, `firmware.ino`, `contracts/mqtt/trigger.schema.json`; `docs/DECISIONS.md` D-039; `16-two-window-prelim-public-waveform-research.md`.
- Confidence: FACT for source change; PRELIMINARY for the growth-fraction numbers (R-022 public-data limits); on-device validation OWED (see below).
- Impact: firmware source DONE; on-device build/flash/shake verification BLOCKED at implementation time (no ESP32 toolchain in that environment) — owed on the owner's bench. `sensor.cpp` cannot be host-compiled (Arduino/FreeRTOS deps); `canonical-host-test.sh` passes (confirms `canonical.h` still host-compiles byte-identically to the Go signer).
- Action: bench build + flash + shake from the laptop; serial confirmation of window behaviour (PRELIM ≈ running peak over first 1 s, two publishes, shared `obs_seq`).
- Experiment: PRELIM timing decision + implementation (§4 of MASTER prompt). Decision: single 1.0 s window; two-window DEFERRED (reopen only on evidence a second window beats re-timing on the latency/FP frontier). Supersedes: R-023 §6.2 recommendation to keep PRELIM timing (owner override); resolves R-021 DISC-1/DISC-C in code.

## R-026 — 2026-09-27 — Trusted-local all-clear asymmetry FIXED (D-040, release-blocker fix)

- Finding: the §6A release blocker was real and is now fixed in source. A trusted-local `EARTHQUAKE_ALERT` to the 20 km audience on a never-CONFIRMED event previously got no FCM all-clear (`FrameFor` pushes RESOLVED/CANCELLED only when `EverConfirmed`, `emit.go:108-112`; `EverConfirmed` set only at CONFIRMED). `Bridge` now records alarming events (`localAlarmed`, keyed by `event_id`, stored BEFORE dispatch so a found-eligible event always owes withdrawal) and on the terminal transition of a never-CONFIRMED alarmed event dispatches one `EVENT_RESOLVED` with `trusted_local=true` through the same token-only 20 km path. CONFIRMED events use the normal all-clear (200 km contains the 20 km ring; local trace forgotten). Additive + fail-safe (no prior alarm → no-op) + S1-preserving (own goroutine + deadline; slow token lookup delays only the local all-clear).
- Evidence: `server/internal/event/emit.go` (`localAlarmed sync.Map` + `reconcileTrustedLocalAllClear` call); `server/internal/event/admin_node.go` (`Store` before dispatch + `reconcileTrustedLocalAllClear` + `trustedLocalAllClearFrame`); `docs/DECISIONS.md` D-040.
- Confidence: FACT for the code change.
- Impact: closes operational blocker §9.1 of the master audit, pending test proof below. Reversible (remove the reconciliation call). Normal-path all-clear semantics, radius, eligibility unchanged.
- Action: four new Bridge tests added (resolve-never-confirmed, cancel-never-confirmed, no-double-all-clear-when-confirmed, no-all-clear-when-never-alarmed); `go test ./internal/event/ -race` green 2026-09-27 (per D-040 text; independently re-verified this session — see R-028).
- Experiment: release-blocker fix (§6A of MASTER prompt). Decision: IMPLEMENT (was "fix or accept" — fixed). Supersedes: R-023 §6.1/§9.1 blocker entry (fixed in source, not yet deployed).

## R-027 — 2026-09-27 — Canonical domain DECIDED + reconciled to quakealert.web.id (D-041); open-source docs drafted; secret containment done, rotation still OPEN

- Finding: §6B domain mismatch resolved as a bounded operational decision (not BLOCKED): DNS checked this session — `api.quakealert.web.id` + `broker.quakealert.web.id` resolve to production VPS `168.110.217.4`, all bare-`.id` hosts NO-RESOLVE, Android reverse-domain package independently confirms `web.id`. `quakealert.web.id` canonical for API + broker. Reconciled in operator/runtime-facing defaults: `config.go` broker default + comment, `.env.prod.example`, `docker-compose.prod.yml` hints, deploy scripts, `openapi.yaml` URL + example, `config.h` comment, `CLIENT_SPEC.md`, `SYSTEM_SPEC.md`. Deliberately NOT changed: JSON-Schema `$id` URNs (opaque identity, no `$ref`/code comparison), Android test fixtures (arbitrary round-trip data), historical records. `go build ./...` + `go test ./internal/config/` green after the change. Open-source docs (§7 MASTER prompt, partial): root `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md` drafted (untracked); `LICENSE` still missing (owner license choice owed). Secret containment (§7/§13 audit B-blockers, partial): `.gitignore` now excludes `session-ses_*`, the firmware stray transcript, `.hermes/` (verified via `check-ignore`; secret files never tracked per `git ls-files`). Rotation per B1 still OPEN and owner-only (MASTER_KEY_HEX highest severity — re-keys node verification, plan re-provisioning first).
- Evidence: `docs/DECISIONS.md` D-041 + B1; DNS resolution this session (per D-041 text); diff across `server/internal/config/config.go`, `deploy/`, `contracts/`, `firmware/src/config.h`, `docs/`; new untracked doc files; `.gitignore` diff.
- Confidence: FACT for source/doc/ignore changes; DNS per D-041 text (not re-checked this session); secret-file names per key-name scan (values never read/printed).
- Impact: deploy-against-defaults now reaches the live names. No push/deploy/flash/rotation performed.
- Action: owner — choose LICENSE (+ ADR), rotate B1 keys, confirm live DNS/TLS on the VPS.
- Experiment: domain decision + open-source prep (§6B/§7 of MASTER prompt). Decision: `web.id` canonical. Supersedes: R-023 §9 blocker entry for the domain (resolved in source; live-VPS confirmation owed).

## R-028 — 2026-09-27 — Independent verification session (build mode): V1 batch re-verified green; bench reachable; two residual gaps found

- Finding: re-ran the V1 batch proof from a clean checkout state (working tree uncommitted, same content as R-024..R-027): `go build ./...` exit 0; `go test ./internal/event/ -race` ok (6.5 s, incl. all 4 D-040 all-clear tests + 7 admin-edge tests + eligibility tests run individually green); `go test ./internal/config/` ok; `firmware/scripts/canonical-host-test.sh` pass; `trigger.schema.json` + `openapi.yaml` parse ok; `SELFTEST=1 go run scripts/admin_floor_sweep.go` faithfulness proof ok. Bench probe (Tailscale `100.110.141.103`, `~/.ssh/bench_readonly` key — password not needed): repo present at `~/QuakeAlert26`, branch `development`, same HEAD `8e93c4d`; `/dev/ttyUSB0` present; `pio`/`platformio` installed. Gap 1: Fedora carries its OWN uncommitted firmware work — the temporary INT_RATE diagnostic (`sensor.h` +7, `sensor.cpp` +57, `firmware.ino` +7, marked TEMPORARY/remove-after-diagnosis) — in different regions than the V1 PRELIM-window edits, so mergeable but must be reconciled deliberately on the bench, never by blind copy. Gap 2: root transcript `2026-09-27-162414-this-session-is-being-continued-from-a-previous-c.txt` is NOT git-ignored (only the `firmware/src/` pattern exists) — one `git add .` away from staging; ignore fix + ledger catch-up (this entry) in progress. Migration path verified locally, no fix needed: `deploy/docker-compose.prod.yml:184` mounts `../contracts/db/migrations` (000001–000010 present); `server/migrations/` does not exist. D-036 activation still contradictory (D-036 "NOT activated" vs `CURRENT_STATE.md:48-51` "live since 2026-09-09T~21:15Z") — VPS read-only check still owed (no VPS target/credentials this session).
- Evidence: test/build outputs this session; SSH probe transcript (ping ~111 ms, key auth, `git status`/`log`, `ls /dev/ttyUSB*`, `which pio`); `git check-ignore` output; `docker-compose.prod.yml:133-188`; `deploy/README.md:41-46`.
- Confidence: FACT (all directly observed this session).
- Impact: CK3 local-test leg now independently evidenced; no files modified by the checks themselves.
- Action: close Gap 2 now (ignore pattern + commit); resolve Gap 1 on the bench with owner (keep vs remove INT_RATE before V1 flash); VPS check when target/credentials provided.
- Experiment: verification + bench-readiness (§9/§13 of MASTER prompt). Decision: none — verification only. Supersedes: nothing.
