# 12 — PRELIM provisional local warning research (RESEARCH ONLY)

- Purpose: answer Research Problem 2 — can Admin Node PRELIM support a bounded provisional local warning before FINAL, and what must FINAL do afterward (confirm / strengthen / revise / cancel)? Decides nothing; maps evidence, tradeoffs, and governance.
- Status: RESEARCH — drafted 2026-09-13 UTC from source + planning evidence. No production change of any kind.
- Last updated: 2026-09-13 UTC.
- Authority note: this directory is a RESEARCH planning area. It does not override `/contracts`, `PROJECT_RULES.md`, `docs/DECISIONS.md`, or `docs/CURRENT_STATE.md` (PROJECT_RULES §5). Any conflict → higher document wins and this is the defect.
- Stop condition (binding): no production code, threshold, dispatch, contract, `algo_ver`, VPS, migration, ESP32-firmware, or Android change under this research. Allowed: this document, ledger/changelog rows, offline replay/benchmark scripts, test datasets, reports.
- Separation (binding): Problem 1 (Admin Node floor, owner target 60 gal) is a *sensitivity* question. Problem 2 (this file) is a *timing/semantics* question. A 60-gal FINAL-only system still waits for FINAL and is still not early warning. 60 gal is used below as an analytical reference only — NOT approved, NOT a prerequisite, NOT a design decision.
- Critical distinction (binding for this file): PRELIM is early evidence, NEVER "earthquake confirmation." The question is only whether that early evidence suffices for a BOUNDED, PROVISIONAL, LOCAL warning within 20 km of one designated node.

## Label legend

- OBSERVED: repository source text, test, or archived field evidence cited with `file:line` or evidence path.
- INFERRED: engineering consequence that follows from OBSERVED facts by stated reasoning.
- HYPOTHETICAL: a design considered but not implemented, measured, or approved.
- UNKNOWN: explicitly not established by any evidence in hand; must not be guessed.

## 0. Prior findings reused (not re-litigated)

- OBSERVED: firmware PRELIM = peak-so-far at ~300 ms confirmation (`firmware/src/sensor.cpp:223-237`); FINAL = whole-event max at detrigger (`sensor.cpp:249-287`); same `obs_seq`, two slots, QoS 0 + retry (`firmware/src/firmware.ino:260-383`, `config.h:127-129`).
- OBSERVED: server one node = one contributor; PGA ratchets up only; FINAL sticks; onset first-bound-wins (`server/internal/event/tracker.go:699-740`); revision++ only on state transition (`server/internal/event/event.go:172-176`); `UNCONFIRMED→UNCONFIRMED` illegal (`event.go:91-117`).
- OBSERVED: D-036 evaluator is 7 gates in fixed order incl. `Phase==FINAL` veto (`server/internal/event/admin_node.go:96-119`); sole frame constructor (`admin_node.go:140-162`); async fail-closed emit (`admin_node.go:176-197`); hook additive after normal emission (`server/internal/event/emit.go:62-75`).
- OBSERVED: D-037 edge is the ONLY non-transition emission: `!wasFinal && UNCONFIRMED && now-FINAL && !adminEdgeFired` → `localPending` snapshot, drained before the empty-transition early-return, no state/revision/log change (`tracker.go:407-427`, `tracker.go:212-235`, `emit.go:77-86`, `event.go:198-206`).
- OBSERVED: dispatch bounds hold at any floor — token-only 20 km, never GeoTopic (`server/internal/dispatch/event_frame.go:60-138`); FCM `"true"`-only (`fcm.go:84-90`); WS `omitempty` (`ws.go:77-81`).
- OBSERVED: Android drops advisories before any gate (`AlertRaiser.kt:71`), dedups per `type:eventId`, routes trusted-local to the distance-only 20 km no-override gate (`AlertGate.kt:138-204`), posts via `WarningNotifier.notify` (`WarningNotifier.kt:104-159`).
- OBSERVED: live anchors — `004c5f65` PRELIM 1.13 → FINAL 288.36 gal eligible (positive control); `4fcc3374` 77.89 gal no local frame (boundary negative); M5.9 deep zero rows (threshold-irrelevant); 2026-09-01 SENSOR onset→decided 6166 ms vs `dur_ms` 6138 (server share 28 ms). Full detail `01-current-state.md:§9`, ledger `R-006`, findings `F-07..F-09`.
- OBSERVED: `algo_ver = phase3-1.1` encodes only independence radius, NOT the admin floor (`server/internal/event/store.go:26`); any eligibility-meaning change forces a V3/V6 version ruling (ledger `R-008`).
- UNKNOWN carried in: true GPIO15/DMP rate (`config.h:100-101` comments assume 100 Hz, `setRate(99)` in code); per-disturbance gal rows; intra-event threshold-crossing instants (UNLOGGED); heartbeat continuity through quake windows; audible-tail distribution; any population rate (ledger `R-007`, findings `F-10`, `09-findings.md` unknowns 1–6).

## A. Earliest technically available actionable point (trace)

OBSERVED chain with per-hop source:

1. `T0` ground motion at sensor. (Physical fact; no repo timestamp.)
2. DMP sample → `sensorTask` → `processSensorData` (`sensor.cpp:95-288`): `vectorMagnitude = sqrt(x²+y²+z²) * DATA_RATIO` (gal, `DATA_RATIO≈0.1197`, `config.h:110`); `baselineEMA α=0.005`; `corrected=|vector−baseline|` (ALL downstream uses this); `STA α=1/50`, `LTA α=1/2000` frozen in-event; clamp 0.5; `ratio=STA/LTA`. Rates assume 100 Hz (UNKNOWN true rate — `R-007`).
3. Trigger: `ratio ≥ 4.0 AND STA ≥ 1.5 gal` (`sensor.cpp:190-201`); warmup 45 s and 60 s cooldown gate before it.
4. Confirmation: hold 300 ms (`CONFIRMATION_DURATION_MS`, `sensor.cpp:203-247`) → `eventInProgress=true`, `pga=correctedMagnitude` (instantaneous), `obs_seq` minted once, PRELIM slot filled (peak-so-far, running duration, onset, detrigger=0).
5. PRELIM publish: `servePendingSlot(pendingPrelim,false)` first (`firmware.ino:367-383`); QoS 0, 5 s backoff, 36 attempts, 300 s max age (`config.h:127-129`); `onset_ts` converted millis→epoch at each attempt (`firmware.ino:300-308`).
6. Server ingest: verify pipeline → `ObservationFrom` (`server/internal/event/input.go:49-70`: v2 keeps `SENSOR` onset + `ObsSeq`; `PublishTS` preserved separately) → `Tracker.Ingest` (location outside lock, fail-closed) → `newEventLocked` (`tracker.go:631-651`, `DETECTED rev0`) or absorb (`upsertContributorLocked`, `tracker.go:699-740`).
7. Event decision: `classify` (`classify.go:10-21`: `<16.6 → DETECTED` else UNCONFIRMED/CONFIRMED by quorum) → `transitionLocked/forceTransitionLocked` (revision++ only on legal transition). PRELIM ≥ 16.6 ⇒ `DETECTED→UNCONFIRMED FLOOR_MET` + `FrameFor` advisory WS-only no push (`emit.go:98-119`, D-009). PRELIM < 16.6 ⇒ stays in-memory DETECTED, no frame, no persistence (`publish` empty path `tracker.go:244-247`).
8. FINAL path today: detrigger (`ratio<1.5` or 60 s cap) → `pendingReport` whole-event max → same absorb; then either transition case (was DETECTED → `EmitTransition` + admin hook) or edge case (was UNCONFIRMED → `EmitAdminEdge`, `tracker.go:407-427` → `publish` drain `213-235`) → same `emitTrustedLocal` evaluator → `DispatchTrustedLocalEventFrame` (WS always + FCM ≤20 km) → Android raise path.

INFERRED: the earliest point carrying (a) measured onset, (b) episode identity, (c) above-noise motion evidence, and (d) a created/updated event is **the ingested PRELIM that first reaches `UNCONFIRMED`** (step 7). Everything before it is firmware-local (no server identity, no audience, no audit row). Everything after it up to FINAL is waiting for the detrigger condition, which is shake-duration-dependent (seconds; cap 60 s).

UNKNOWN: trigger-to-PRELIM wall time at the true sample rate; MQTT transit distribution unhealthy; per-event `T5` duration distribution; FCM/Doze/audible tail distribution beyond the two drill points (+467 ms locked, +960 ms FCM-revived — OBSERVED drill only, `11-hardware-and-timing.md`).

## B. What PRELIM contains vs what is inherently missing

OBSERVED present at PRELIM (v2): `node_id`, `pga` (peak-so-far), `dur_ms` (running), `ts` (publish), `onset_ts` SENSOR + `SENSOR` provenance, `obs_seq` (episode identity), `attempt_no`, registered lat/lon/location (server-side join), correlation anchor. (`trigger.go` schema; `input.go:49-70`.)

OBSERVED present only at FINAL: whole-event max PGA, full duration, `detrigger_ts`, closed-episode signal. (`sensor.cpp:249-287`; `input.go:DetriggerTS`.)

INFERRED inherently unavailable at PRELIM, at any threshold: whether the peak has occurred yet; final max and its margin over the floor; total duration; whether a second episode/retrigger follows inside the window; whether the contributor would ever reach FINAL at all (QoS-0 loss, node loss, 60 s cap); network corroboration beyond this one node. No threshold choice recovers these — they are future facts, not hidden parameters.

HYPOTHETICAL mitigations that do NOT cure inherence: lower PRELIM bar (admits earlier, still blind to future); STA-persistence add-ons (more evidence of *started* shaking, zero evidence of *peak* shaking); duration gates (delay toward FINAL, defeating the purpose).

## C. What PRELIM technically is (three semantics separated)

- Sensor semantics — OBSERVED: a confirmed-onset early report (`sensor.cpp:223-225` comment: peak-so-far, running duration, "bukan nilai final"). It asserts *shaking started here at onset_ts with at least this peak so far*. It does not assert magnitude, duration, peak, or cause.
- Server event semantics — OBSERVED: one contributor vote (`event.go:129-131`); sufficient with ≥16.6 gal to create `UNCONFIRMED` (real shaking as far as the network knows, unconfirmed by separated stations). `UNCONFIRMED` is already a public claim (advisory WS frame) but deliberately non-alarming (D-009).
- User-warning semantics — OBSERVED: PRELIM today produces at most a banner; it never produces `trusted_local=true` (`ADMIN_NOT_FINAL`, `admin_node.go:110-111`; advisory dropped pre-gate, `AlertRaiser.kt:71`). Calling it a "warning" today is false; calling it "confirmation" under any architecture would be false. The only honest user-facing reading of a PRELIM-based alert is **provisional local warning: early single-station evidence, bounded to 20 km, subject to confirmation/withdrawal when FINAL arrives**.

## D. Can PRELIM-only warn under the existing model? (exact blockers)

INFERRED: technically routable (the snapshot + hook + dispatch + client path all exist at PRELIM time), but blocked at four independent layers, each sufficient alone:

1. Server gate: `contrib.Phase != PhaseFinal → ADMIN_NOT_FINAL` (`admin_node.go:110-111`). Single-line veto; PGA value never examined afterward.
2. Emission wiring: provisional has no emitter — `EmitTransition` fires the admin hook only on transitions (`emit.go:62-75`); the only non-transition emitter is the FINAL-conditioned D-037 edge (`tracker.go:422-425`). A PRELIM snapshot is never queued to `localPending`.
3. Contract/type: no provisional representation exists — `type` enum frozen 3 values (`ws.go:41-68`, D11 rationale `emit.go:92-97`); `trusted_local` on an advisory is meaningless to every client (dropped at `AlertRaiser.kt:71-72,89-92` by design).
4. Client gate: even a crafted frame would need `EARTHQUAKE_ALERT + trusted_local=true` to reach `decideLocalWarning`; the client cannot distinguish "provisional" from "confirmed-local" today, so any PRELIM reuse without representation work silently upgrades provisional evidence to confirmed-local copy ("LOCAL WARNING" pill, `WarningNotifier.kt:137-145`).

Do NOT read "routable" as "safe": removing veto (1) alone creates exactly the silent-upgrade hazard in (4), with FP exposure bounded below (§F) and no withdrawal path (§G/H).

## E. Architecture comparison

### E1. CURRENT (FINAL-only): PRELIM → UNCONFIRMED → FINAL → warning

- OBSERVED behavior: §0 + §A steps 7–8; proven live (004c5f65 eligible; 4fcc3374 correctly silent).
- Latency: INFERRED slowest of the options by exactly the detrigger wait (`T5`: seconds to 60 s cap). 2026-09-01 SENSOR case shows the structural content: onset→decided 6166 ms ≈ `dur_ms` 6138 (server share 28 ms) — the wait IS the shake so far.
- Sensitivity: misses nothing FINAL would catch at the same floor, but cannot warn before the event ends locally — at the detecting site lead time is negative by physics (S1); for 20 km neighbors the wait consumes the S-travel budget.
- FP exposure: OBSERVED smallest — FINAL max + full-duration evidence + exactly-once + 7 gates; zero observed environmental false-eligibles at 140 (small-n, site-specific — NOT a rate).
- Duplicates/revision/stand-down: OBSERVED clean — revision only on transition; edge exactly-once; same-`event_id` dedup; `EVENT_RESOLVED` clears (normal RESOLVED push only if `EverConfirmed`, `emit.go:108-112` — local path records its own emission regardless).
- Complexity/auditability: OBSERVED minimal — one veto line, one edge block, closed reason vocab, one audit line (D-019), validity defaulted (D-018), per-event coexistence (D-020) untouched.
- Governance: OBSERVED heaviest approval already banked (D-036 + deployed D-037 code); `algo_ver` question open (`R-008`) but no new semantics.
- Verdict: INFERRED safest today; INFERRED not early warning for the local audience.

### E2. PROVISIONAL (recommended IF ever built): PRELIM → provisional local warning (same frame shape, same event_id) → FINAL confirms/maintains/upgrades or withdraws

- HYPOTHETICAL mechanism (minimal delta): evaluate the admin gates at the PRELIM-created (or PRELIM-updated) `UNCONFIRMED` snapshot with the FINAL veto replaced by a provisional branch; emit `EARTHQUAKE_ALERT + trusted_local=true + event_state=UNCONFIRMED`, same `event_id`, same `revision` (no bump — mirrors D-037), through the identical `DispatchTrustedLocalEventFrame` path; FINAL arrival re-evaluates (eligible → confirm/maintain silently; stronger peak → same-frame update suppressed as duplicate by design — no re-siren; ineligible → explicit stand-down, §H).
- Latency: INFERRED fastest — saves the full `T5` detrigger wait; T1–T4/T8–T9 unchanged. Magnitude UNKNOWN until growth-curve measured (Q4).
- Sensitivity: INFERRED highest — any episode crossing the PRELIM bar warns even if FINAL is lost (QoS-0), node dies, or event caps at 60 s.
- FP exposure: INFERRED highest — decides on the least-informative peak (peak-so-far systematically understates final; transient 300 ms-confirmable spikes qualify). The 16-gal/6-month zero-FP history is at a different (lower, FINAL-gated, advisory-only-loudness) operating point and does NOT transfer — UNKNOWN until the disturbance battery (§11-A) is run at the PRELIM bar.
- Missing info: §B inherence in full force — every provisional is issued blind to peak/duration/outcome.
- Duplicates: INFERRED manageable — same `event_id` + `markIfNew` suppresses the FINAL confirm as "already handled" (no double siren by construction); risk inverts to *silent* confirmation (user never learns it was confirmed — acceptable) and *stale-provisional* (user keeps a warning FINAL invalidated — must be closed by explicit stand-down, Q3).
- Revision/stand-down: INFERRED representable without new state — no transition, no bump (D-003 preserved, exactly as D-037); withdrawal needs an explicit `EVENT_RESOLVED`-for-this-`event_id` even though the event never CONFIRMED (exception to `emit.go:108-112` all-clear-owed rule — governance item, not code trick).
- Server complexity: INFERRED small (new `localPending`-style queue branch conditioned on PRELIM + exactly-once guard mirroring `adminEdgeFired` + reason-vocab additions e.g. `ADMIN_PRELIMINARY`); async/fail-closed/S1 posture unchanged.
- Android complexity: INFERRED near-zero IF same-frame reuse — no new type, no new gate (existing `decideLocalWarning`), validity/dedup/coexistence/logging apply unchanged; copy must gain a provisional marker to avoid silent-upgrade (additive string/flag — contract governance, §H).
- Auditability: INFERRED preserved — same `trusted-local emitted` line + new reason value; replay must record gate-at-PRELIM vs gate-at-FINAL (procedure change, `06-replay-plan.md` step 2 already requires reason-at-X).
- Failure modes: §12 table worst case is a loud 20 km alarm for a transient that FINAL disproves seconds later (S4 cost: one discredited channel); mitigations are short validity + immediate explicit withdrawal + battery-gated bar (§H/Q3).
- Governance: new/expanded decision required (D-036 amendment or D-number), `algo_ver` ruling (eligibility meaning changes → V3/V6 bump or admin-version), contract addendum for the provisional marker, owner acceptance criteria (`10-decision.md` §1 adapted: FP battery + growth curve + T_min gain + robustness).

### E3. TWO-STAGE: PRELIM → lower-severity local advisory → FINAL → actionable warning

- HYPOTHETICAL: provisional carried as advisory-tier (banner/soundless) then FINAL escalates to alarming `trusted_local`.
- Latency/sensitivity/FP: INFERRED — advisory tier adds no loud coverage beyond today (client drops advisories pre-gate by deliberate safety net, `AlertRaiser.kt:62-71`); FP exposure of the loud path identical to CURRENT; the only gain is an earlier *silent* cue for foreground users.
- INFERRED verdict: strictly dominated for the stated objective (warn nearby sleeping users loudly). It re-spends architecture/contract/governance budget to reproduce CURRENT's loud behavior with an extra quiet step. Viable only if the owner explicitly wants a quiet pre-cue AND accepts a new advisory-local representation + client work — a product decision, not a safety improvement. Not recommended as the answer to Problem 2.

### E4. OTHER (evidence-suggested): PRELIM-gated early dispatch of the SAME FINAL rule (no provisional alarm)

- HYPOTHETICAL, recorded because the evidence suggests it as a cheaper halfway house: on PRELIM, pre-resolve everything *except* the alarm — prefetch admin status/heartbeat, precompute tokens, arm the evaluator — so the FINAL edge fires with minimum added delay; optionally log (not emit) the intra-event threshold-crossing instant for the growth curve.
- INFERRED: zero FP change (alarm still FINAL-gated), small latency shave (async 2 s budget + token lookup off the critical path), zero contract/client/governance cost beyond logging. Recommended as a no-regrets instrumentation step regardless of E2, NOT as an alternative warning architecture.

## F. PRELIM × 60-gal matrix (analytical reference only — no policy chosen)

Distinction enforced throughout: PRELIM bar tests peak-so-far ≈300 ms after onset; FINAL bar tests whole-event max. Same number ≠ same evidence.

- PRELIM ≥ 60 / FINAL ≥ 60 — OBSERVED routable under E2; INFERRED earliest + most sensitive; INFERRED highest FP (decides on least information). Live 004c5f65 shape (1.13 → 288.36) shows how far apart the two readings of "the same episode" can be — a 60 PRELIM bar would have warned on episodes whose FINAL later reads 5× higher AND on transients that never grow.
- PRELIM ≥ 60 / FINAL < 60 — INFERRED the false-provisional archetype: loud 20 km alarm followed by disproving FINAL seconds later. Frequency UNKNOWN (no battery, no growth curve, n≈2). This cell is the entire S4 cost of E2; Q3 withdrawal design exists for exactly this cell.
- PRELIM < 60 / FINAL ≥ 60 — OBSERVED under CURRENT (e.g. 7 → 166; 1.13 → 288.36): warning arrives at FINAL via transition path (was DETECTED) or D-037 edge (was UNCONFIRMED). INFERRED: E2 buys nothing here beyond CURRENT (provisional never fires; FINAL still warns). A PRELIM bar *above* the FINAL bar would enlarge this silent-until-FINAL class — the cost of the safer split-threshold variant (Q2).
- PRELIM < 60 / FINAL < 60 — OBSERVED silent under every architecture at floor 60 (e.g. 4fcc3374 77.89? No — that is ABOVE 60: under a 60 floor it WOULD warn; the true double-below class has no archived member — UNKNOWN rate). Correctly silent for weak episodes; the miss class for genuinely damaging low-PGA-at-sensor events (site/frequency/duration effects — RQ-EQ, UNKNOWN).

60-gal interaction summary — INFERRED: lowering the FINAL floor to 60 expands *which* episodes warn; adding a PRELIM path expands *when* they warn. The two multiply: 60-gal provisional is the most sensitive AND least evidenced combination. Sensitivity-first preference does not supply the missing battery/growth-curve/rate evidence (§8).

## G. Representability audit (supported today vs needs-new)

OBSERVED supported today, no contract/state change: same-`event_id` provisional + silent confirm/update (dedup `markIfNew` suppresses re-siren); revision-no-bump provisional and edge (D-003/D-037 precedent); per-event coexistence for concurrent episodes (distinct `event_id`s, D-020 slots); `EVENT_RESOLVED` clearing on the client (`AlertRaiser.kt:51-60`, type-frozen so old builds still clear); validity expiry suppressing stale replays (`WsAlertMessage.kt:151-160`); D-019 event_id+outcome logging both ends.

INFERRED needs-new (governance, then additive contract): a provisional marker perceivable by user/support (else silent upgrade of evidence strength — §D(4)); withdrawal semantics for never-CONFIRMED provisional (exception to `emit.go:108-112` all-clear-owed rule + push-audience ruling for the stand-down itself); reason-vocab additions (e.g. `ADMIN_PRELIMINARY`, `PROVISIONAL_WITHDRAWN`); replay/admin-outcome columns for gate-at-PRELIM; `algo_ver` ruling (V3/V6). Explicitly NOT needed: new event state, new `type` value, quorum/classifier/radius change, heartbeat-as-evidence (heartbeat stays dispatch gate only, `admin_node.go:32-36`).

## H. Notification UX tradeoffs

- ALERT-reuse vs advisory — INFERRED: reuse is the only option that actually warns (advisory path is dropped pre-gate by deliberate double safety net, `AlertRaiser.kt:62-71`; server never FCM-pushes advisories). A "provisional advisory" that wakes nobody answers a different question than Problem 2. Cost of reuse: must carry a provisional marker or the UI claims confirmed-local confidence it never had (copy + `isLocalWarning` pill path, `WarningNotifier.kt:137-145`).
- D-018 validity — OBSERVED mechanism, INFERRED setting: provisional MUST carry a short sender-declared validity (tens of seconds, not the 90 s real-path default) so an un-withdrawn provisional (lost FINAL, dead node, dropped FCM) dies loudly-quiet rather than lingering to the 15-min legacy window (`WsAlertMessage.kt:151-167`). Confirm/upgrade re-arms with the normal validity; withdrawal bypasses expiry (clears are never dangerous).
- D-020 coexistence — OBSERVED: per-`event_id` slots, ≤3 + collapse, newest sounds. INFERRED: same-ID provisional→confirm collapses correctly (no new slot, no double siren); distinct concurrent episodes coexist correctly; beyond-3 collapse must never silently drop a live provisional without representation (existing invariant, unchanged).
- D-019 audit — OBSERVED: one server line (`trusted-local emitted`, event_id+outcome) + one client line per outcome, never position. INFERRED: provisional needs no new shape, only new reason values; the withdrawal line is load-bearing for the F-02-style false-provisional accounting.
- Foreground/background parity — INFERRED: WS-first + FCM-token-only identical to today (`event_frame.go:98-116`); cold-start/FCM-delay/Doze tails (UNKNOWN distribution) dominate any T1–T4 gain below ~0.5–1 s — the `10-decision.md` T_min rationale, carried over.

## I. Q1–Q4 resolutions (evidence → tradeoff → verdict)

**Q1 — same `EARTHQUAKE_ALERT + trusted_local=true`, same event_id/revision, or distinct representation?**
Established (OBSERVED): frozen 3-type enum + advisory-drop + dedup + coexistence + no-new-state rule make same-frame reuse the only compatible carrier. NOT established: user-comprehensible provisional copy (no strings, no tests). Tradeoff: reuse = minimal complexity + silent-upgrade risk; distinct type = honest labeling + contract fork + old-build drop/clear hazards. Provisional research conclusion: **reuse same frame + same event_id + same revision, WITH an additive provisional marker** (string/flag, additive/omitempty like `trusted_local` itself). No new state, no new type.

**Q2 — same 60 gal for PRELIM and FINAL, or split?**
Established (OBSERVED+F-04): phase-incomparable by construction; same number tests strictly less evidence at PRELIM. NOT established: growth-curve slope, battery margins at any bar, rate at any bar. Tradeoff: same-bar = simplest + most sensitive + highest FP; split (PRELIM bar ≥ FINAL bar, or non-PGA persistence add-on) = compensates evidence gap + delays/rare-ifies provisional + needs its own battery. Provisional research conclusion: **do NOT assume same threshold; default to split-or-gated (PRELIM bar never below FINAL bar) pending battery + growth data**. The exact numbers are OPEN and must come from `05/06` matrix + `11-A` battery, not preference.

**Q3 — silent resolve vs explicit withdrawal when FINAL disproves provisional?**
Established (OBSERVED): clears bypass gates/expiry client-side; normal RESOLVED owes audience only to ever-alarmed (local provisional qualifies morally but not under current `EverConfirmed` rule). NOT established: withdrawal copy, push audience for the withdrawal, post-withdrawal re-alarm policy for the same episode. Tradeoff: silent = less alarm-fatigue + stranded siren risk; explicit = closes the loop + one extra alarming-adjacent interruption. Provisional research conclusion: **explicit `EVENT_RESOLVED`-for-this-`event_id` withdrawal is REQUIRED for any provisional that alarmed** (a siren the system raised is a debt the system must retire); silent expiry is only the backstop (short D-018 validity), never the plan.

**Q4 — is measuring intra-event threshold-crossing time a prerequisite?**
Established (OBSERVED): crossing instant UNLOGGED (`R-003` impact); latency gain claim currently unfalsifiable; downstream tails dominate sub-second gains. NOT established: the distribution itself. Tradeoff: measuring delays the decision by instrumentation work vs deciding blind. Provisional research conclusion: **YES — prerequisite (or owner formally accepts the residual as UNKNOWN risk)**. Minimum: growth-curve sampling on replayable episodes + disturbance battery at the candidate PRELIM bar + true sample-rate measurement (`R-007`); otherwise any "earlier warning" claim violates PROJECT_RULES §8 honesty.

## Worked examples (demonstrate / cannot-demonstrate)

- 50 → 200 (PRELIM ≥16.6, FINAL ≥140-class): OBSERVED pattern for D-037 edge taxonomy — demonstrates why transition-only emission goes silent (no revision) and what the edge repairs. Cannot demonstrate rate, timing gain, or FP behavior.
- 120 → 575 (already-UNCONFIRMED): demonstrates the exact pre-D-037 failure (strong FINAL, zero frame) and post-D-037 single local ALERT. Cannot demonstrate provisional value (both readings far above any bar).
- 7 → 166 (PRELIM below floor): demonstrates the D-036 transition path (no edge; warning rides `DETECTED→UNCONFIRMED` with FINAL already present). Under E2 with a 60 PRELIM bar this episode stays silent until FINAL — demonstrates the low→high trap working as designed.
- 1.13 → 288.36 (live 004c5f65): OBSERVED positive control for FINAL-gating; demonstrates phase non-equivalence at its most extreme (two orders of magnitude same episode). Cannot demonstrate that a 60 PRELIM bar would have been either timely or safe.
- 77.89 (4fcc3374, negative): OBSERVED no-frame below 140; under a 60 floor it becomes eligible-class — demonstrates that Problem-1 sensitivity and Problem-2 timing are independent decisions with compounding blast radius. Cannot demonstrate felt outcome or FP status.
- Global caveat — INFERRED: n≈2 eligible-class episodes cannot support any false-positive/confirmation/latency RATE (PROJECT_RULES §6, S9, `02-research-questions.md` out-of-scope). Every rate claim from these examples would be dishonest.

## Missing measurements (all UNKNOWN until run)

1. True DMP/GPIO15 sample rate (rescues every STA/LTA constant). 2. Trigger→PRELIM wall time at measured rate. 3. Server ingest latency distribution unhealthy (7–14 ms healthy OBSERVED only). 4. PRELIM→FINAL growth time per episode + time spent crossing each candidate bar (60/100/140) — needs intra-event PGA logging (new instrumentation decision, NOT this research). 5. Heartbeat continuity through quake windows (no history table). 6. Audible-tail distribution (locked/in-use/vendors/Doze). 7. False-positive population/denominator at the deployment mount (disturbance battery `11-A` empty). 8. Transfer curve mount/waveform→reported-PGA (external seismometer ≠ ESP32, `R-004`).

## Decision matrix

| # | Statement | Class | Evidence |
| --- | --- | --- | --- |
| 1 | PRELIM veto is one line (`ADMIN_NOT_FINAL`); FINAL-only by construction | FACT | `admin_node.go:110-111` |
| 2 | PRELIM peak-so-far ≪ FINAL whole-max same episode | FACT + OBSERVED | `sensor.cpp:209-287`; 1.13→288.36 |
| 3 | Same-`event_id`/no-bump provisional preserves D-003; dedup suppresses double siren | INFERENCE | `event.go:172-206`; `tracker.go:407-427`; `AlertRaiser.kt:78-81` |
| 4 | Advisory-tier provisional cannot warn (dropped pre-gate by design) | FACT | `AlertRaiser.kt:62-72` |
| 5 | Withdrawal for alarming provisional needs explicit RESOLVED (rule exception) | INFERENCE | `emit.go:108-112` vs clear path `AlertRaiser.kt:51-60` |
| 6 | Latency gain = detrigger wait only; magnitude unmeasured | INFERENCE + UNKNOWN | T-model `11-`; 6166≈6138 case |
| 7 | FP rate at any PRELIM bar | UNKNOWN | battery empty; n≈2 |
| 8 | 16-gal/6-month history transfers to 60-PRELIM safety | UNKNOWN (do not assume) | different gate + loudness operating point |
| 9 | Same-frame + marker + short validity + explicit withdrawal is the minimal compatible E2 | RECOMMENDATION (conditional) | §§E–I |
| 10 | Evidence sufficient to implement provisional warning now | RECOMMENDATION: NO (see below) | §§F–I + unknowns 1–8 |

## Research conclusion (6 answers)

1. **Can PRELIM safely be made actionable?** Technically YES (earliest actionable point exists; model can represent it), safely NO on current evidence (FP/growth/rate unknowns empty; withdrawing semantics unapproved; version/contract rulings open).
2. **Recommended architecture?** If pursued after evidence: **E2 PROVISIONAL** — same `EARTHQUAKE_ALERT + trusted_local=true`, same `event_id`, same revision, additive provisional marker, short D-018 validity, explicit `EVENT_RESOLVED` withdrawal, unchanged 20 km token-only dispatch, unchanged gates otherwise. E3 (two-stage advisory) not recommended for the loud-warning objective. E4 (PRELIM-armed early dispatch + crossing-instant logging) recommended as no-regrets instrumentation regardless.
3. **What must FINAL do?** Re-evaluate same gates on the FINAL snapshot: eligible → confirm/maintain (silently absorbed as duplicate — no re-siren); stronger peak → same-frame update (same suppression); ineligible (below bar / unverified / stale heartbeat / terminal) → explicit withdrawal for the provisional `event_id`; resolved/expired → stand-down. Later FINAL duplicates/retries: nothing (exactly-once guard, D-037 precedent).
4. **Does 60-gal make provisional more useful?** It makes it *more frequent*, not proven *more useful*: usefulness = timeliness × correctness, and correctness at 60-PRELIM is UNKNOWN while the high→low false-provisional cell grows monotonically as the bar drops. Keep Problems 1 and 2 decoupled; compounding them without the battery is the highest-risk combination in this file.
5. **Evidence still required:** Q4 growth-curve + §11-A disturbance battery at the candidate PRELIM bar + true sample-rate + heartbeat continuity + audible-tail sample + replay matrix with reason-at-PRELIM vs reason-at-FINAL (`06-replay-plan.md` step 2) + owner T_min + FP/detection/robustness criteria (`10-decision.md` §1 adapted to provisional).
6. **Governance before implementation:** D-036 amendment (or new D-number) covering provisional branch, veto relaxation, reason vocab, withdrawal rule, validity policy; D-037-equivalent exactly-once/no-bump text for the PRELIM edge; contract addendum (provisional marker, additive/omitempty); `algo_ver` ruling V3/V6 (eligibility meaning changes under identical label — bump, admin-version, or comparability case); owner acceptance criteria + activation preconditions; deployment scheduling. None approved here.

**Sufficiency statement: evidence is INSUFFICIENT to implement PRELIM warning now. DO NOT CHANGE PRODUCTION.** Next experiment: E4 instrumentation first (crossing-instant logging + armed dispatch, logging-only decision), then `04/05/06` replay+benchmark at candidate PRELIM bars, then `11-A` battery + rate measurement, then return with KEEP FINAL-only / TEST provisional FURTHER / ADOPT provisional + bar — or repeat INSUFFICIENT EVIDENCE.
