# 15 — Deep pre-execution audit: two-window PRELIM architecture

- Purpose: decide whether the proposed **PRELIM-1 / PRELIM-2 / FINAL** two-window
  architecture is technically, statistically, architecturally, and operationally
  sound **before any execution**. Determine only whether it deserves to proceed
  to an offline experiment.
- Status: **AUDIT ONLY.** No implementation, no deployment, no physical test, no
  production change, no threshold change, no new D-number / algo_ver / contract
  version. Nothing here changes the floor (still 140.0).
- Date: 2026-09-26 UTC.
- Method: three read-only source passes (firmware+timing; server pipeline;
  Android+contracts+decision-docs), cross-checked against direct reads of
  `firmware/src/sensor.cpp`, `config.h`, `mqtt.cpp`. Every claim is labelled
  **SOURCE FACT** (in source) / **OBSERVATION** (my reading of source) /
  **INFERENCE** (deduced) / **UNKNOWN** (not derivable from current data).
- The 1–2 s window is a **hypothesis to audit, not an approved requirement.**
  "Two windows better than one" is **not assumed.**

---

## 1. Executive conclusion

**Verdict: INSUFFICIENT EVIDENCE to implement. NO-GO on firmware / server /
Android / production. CONDITIONAL GO only for the *design* of an offline
experiment — whose first deliverable is data that does not exist yet.**

Three findings drive this:

1. **The historical data cannot evaluate the core premise.** (SOURCE FACT +
   INFERENCE) The stored PRELIM is a **single onset-instant PGA sample** taken
   at the ~300 ms confirmation point (`sensor.cpp:214,227`), and FINAL is the
   event peak (`sensor.cpp:267`). There are exactly **two PGA points per event**
   and **no intra-event growth curve** anywhere in the data. The proposal's
   central claim — that a window at 1–2 s carries meaningfully more actionable
   evidence than 300 ms — is therefore **UNKNOWN and unmeasurable from history.**
   "What happens at 1 s / 2 s / 3 s?" has no answer in the current dataset.

2. **"Two windows" conflates two orthogonal knobs.** (INFERENCE) A *second
   sample time* and a *second decision threshold* are independent scalars.
   Everything the proposal wants from "PRELIM-2 = peak-at-2s" is already
   expressible as a single re-timed, re-thresholded PRELIM. A second window is a
   genuinely new concept **only if PRELIM-2 is a decision** (a persistence /
   stability judgement over accumulated evidence), not merely a later sample —
   and that decision logic is unspecified today (the request's own Q4 lists four
   incompatible candidate meanings).

3. **Every downstream path that would let a provisional warning alarm is
   currently closed by design, and the false-warning evidence to open them
   safely is empty.** (SOURCE FACT) `classify()` ignores phase entirely
   (`classify.go:10-21`); Android drops every pre-confirmation (`ADVISORY` /
   `UNCONFIRMED`) frame before the gate (`AlertRaiser.kt:71`); the false-warning
   metric-D row for the decisive band is empty and, worse, the planned battery
   measures the wrong quantity (FINAL, not the intra-event curve a provisional
   window decides on).

The concept is not disproven — it targets a real defect (FINAL-gating means the
local warning fires at detrigger, after local shaking; SOURCE FACT
`sensor.cpp:255-257`). But it must earn its way forward through a bounded
offline experiment, and that experiment is currently blocked on instrumentation
that does not exist. The Decision Gate at the end lists the exact evidence
required.

## 2. What is already proven (SOURCE FACTs)

Firmware (direct read of `sensor.cpp`, `config.h`, `mqtt.cpp`):
- Trigger: `ratio >= 4.0 && sta >= 1.5 gal` (`sensor.cpp:191`; `config.h:103,105`).
- Confirmation hold = 300 ms (`CONFIRMATION_DURATION_MS`, `config.h:106`;
  `sensor.cpp:209`).
- **PRELIM PGA is a single instantaneous sample at the confirmation instant.**
  `pga = correctedMagnitude` is assigned once at confirmation (`sensor.cpp:214`)
  and copied straight into `pendingPrelim.maxPga` (`sensor.cpp:227`). The
  running peak `pga = max(pga, correctedMagnitude)` only runs *after*
  `eventInProgress` (`sensor.cpp:250-252`), i.e. on later samples — after PRELIM
  is already frozen. So PRELIM ≈ PGA at ~300 ms, **not** peak-so-far.
- FINAL PGA = the true running event peak, written at detrigger
  (`pendingReport.maxPga = pga`, `sensor.cpp:267`).
- Detrigger = `ratio < 1.5` (`STA_LTA_DETRIGGER_RATIO`) or event age > 60 000 ms
  (`sensor.cpp:254-257`; `config.h:104,107`).
- **Exactly two emission slots exist** — `pendingPrelim` and `pendingReport`.
  No intermediate slot, no second-PRELIM path (`sensor.cpp:226-237, 264-278`).
- Cooldown 60 s, LTA warmup 45 s (`config.h:108,109`).

Server (pass ac7b96e8544d3ba5f):
- Phase is a **sticky-FINAL ratchet**; PGA **ratchets up only**
  (`tracker.go:723-728`). `classify()` reads only peak-PGA, contributor count,
  independent cells, and Invalidated — **it never reads phase**
  (`classify.go:10-21`).
- Admin trusted-local eligibility = 7 ordered gates, floor `AdminNodeMinPGAGal
  = 140.0`, one gate being `Phase == FINAL` (`admin_node.go:96-119,30`).
- D-037 FINAL-edge fires **exactly once** on the FINAL flip while UNCONFIRMED,
  bumps neither revision nor state (`tracker.go:406-427`, `event.go:198-206`).
- `algo_ver` encodes only `phase3-1.1` + IndependenceCellKm — **not** the admin
  floor and **not** the phase gate (`store.go:26,71-73`).
- Replay is strictly read-only, params operator-asserted; reconstructs
  per-frame PRELIM/FINAL phase and PGA endpoints but **not** absolute
  timestamps and **not** any intra-event curve; the D-037 edge is **not**
  surfaced by replay (`replay.go:207-211,466-491`).

Android + contracts (pass af6a6626654b876fa):
- Wire `type` enum is **frozen at three** values; unknown type → frame dropped
  (`WsAlertMessage.kt:16-31`, `AlertMappers.kt:44`).
- **Every pre-confirmation frame is dropped before the gate**: `if type ==
  EARTHQUAKE_ADVISORY return` (`AlertRaiser.kt:71`). UNCONFIRMED can never wake
  the device.
- `event_state` is an additive enum; an unknown value parses to null but the
  frame is **kept** (`AlertMappers.kt:88-89`).
- Trusted-local gate is distance-only, 20 km, **no** severity override
  (`AlertGate.kt:182-203`).
- EVENT_RESOLVED bypasses gate + expiry; **all-clear FCM goes only to the
  ever-CONFIRMED audience** (`emit.go:112`).
- Dedup key = `type:eventId`; lower/equal revision suppressed
  (`AlertDedup.kt:43,73`). D-018 validity, D-019 no-position audit, D-020 max-3
  coexistence all confirmed present.
- Trigger schema `phase` enum = exactly `[PRELIM, FINAL]`, "exactly two
  publications per event, no UPDATE" (`trigger.schema.json:53-60`). The
  client-facing event frame has **no** provisional field.

Decision record: stance is **NO-GO / floor stays 140.0** (`10-decision.md`);
metrics A–F essentially empty, the 100–140 gal false-warning cell empty
(`07-threshold-comparison.md`, `08-safety-analysis.md`); six blocking unknowns
stand (`09-findings.md:26`).

## 3. What the two-window concept changes

Relative to today's pipeline the proposal would require, at minimum:
- **Firmware** (SOURCE FACT there is no third slot): a new mid-event snapshot at
  a candidate time, a timer, and a third wire phase value.
- **PRELIM semantics** (SOURCE FACT PRELIM is onset-instant): PRELIM-1 would
  keep the onset-instant meaning, but PRELIM-2 needs a *new, explicit* meaning
  (peak-so-far, or a decision) — this redefines what "PRELIM" is.
- **Server** (SOURCE FACT classify ignores phase): a new phase-gated side
  channel analogous to the D-036 hook / D-037 edge, because a provisional phase
  cannot ride the consensus state machine.
- **Android** (SOURCE FACT advisory is dropped): a raise path for a provisional
  frame — either lifting the advisory drop or adding a `decideFor` branch.
- **Governance**: a D-number, an algo_ver ruling (phase gate is not encoded), a
  contract amendment (new phase value), possibly a schema migration.

It need **not** change: STA/LTA math, trigger/detrigger logic, `event_id`
assignment, revision-on-transition invariant, or the consensus (`classify`)
gate — and it must not, for the reasons in §14.

## 4. Semantic definition — PRELIM-1 vs PRELIM-2 vs FINAL

| Term | Current meaning (SOURCE FACT) | Under the proposal |
| --- | --- | --- |
| raw signal | DMP linear accel magnitude × `DATA_RATIO` (`sensor.cpp:142-146`) | unchanged |
| correctedMagnitude | gravity/DC-removed magnitude, gal (`sensor.cpp:152-155`) | unchanged |
| STA | ~0.5 s EMA of correctedMagnitude (`sensor.cpp:169`) | unchanged |
| LTA | ~20 s EMA, frozen during event (`sensor.cpp:170-172`) | unchanged |
| trigger | ratio≥4 & STA≥1.5 gal (`sensor.cpp:191`) | = PRELIM-1 onset |
| confirmation | 300 ms hold (`sensor.cpp:209`) | = PRELIM-1 emit point |
| PRELIM (today) | **onset-instant PGA at ~300 ms** (`sensor.cpp:214,227`) | → PRELIM-1 |
| PRELIM-1 (proposed) | — | rapid candidate; onset-instant PGA |
| PRELIM-2 (proposed) | **does not exist** | **UNDERSPECIFIED** (see below) |
| FINAL | true event-peak PGA at detrigger (`sensor.cpp:267`) | unchanged |
| resolution | RESOLVED/CANCELLED transition (`event.go`) | unchanged |

**Finding (INFERENCE):** PRELIM-2 is semantically distinct enough to justify a
separate concept **only if** it is a *decision* — a persistence/stability score
over the samples between 300 ms and T, or a re-detection judgement. If PRELIM-2
is merely "peak-so-far at T=1–2 s," it is **not a new concept**; it is a
re-timed, re-thresholded PRELIM and should be treated as a parameter change to a
single window, which is strictly simpler and answers research-Q1 directly.
**Do not silently redefine PRELIM** from onset-instant (its true current
meaning) to peak-so-far without recording it as a semantic change — note that
`trigger.schema.json:55` and the `sensor.cpp:223-225` comment *already* mislabel
today's PRELIM as "peak-so-far," which is a **pre-existing documentation defect**
to correct before any design (see §20 blocker 2).

## 5. Timing audit

- **Sample rate: UNKNOWN.** `mpu.setRate(99)` with comments asserting 100 Hz
  (`sensor.cpp:69,83`; `config.h:100-101`); no rate diagnostic exists in the
  repo firmware. This **contradicts ledger R-013's claim** that the bench node
  "already prints INT_RATE" — no such print is in `sensor.cpp`, `firmware.ino`,
  or `mqtt.cpp`. Treat the true GPIO15/DMP rate as UNMEASURED (matches
  `09-findings.md` F-10). Every STA/LTA time-constant and the "1–2 s" figure
  ride on this unknown.
- Confirmation 300 ms; PRELIM emitted at that instant. FINAL at detrigger
  (ratio<1.5) or 60 s cap. Cooldown 60 s. (all SOURCE FACT, §2.)
- Publication: QoS 0 (best-effort, PRELIM can be lost); `ts` re-stamped each
  retry, not onset; `onset_ts` = sensor clock; `detrigger_ts` = FINAL only
  (`trigger.schema.json:36-81`; `mqtt.cpp`).
- **Is 1–2 s meaningful or arbitrary? UNKNOWN.** It cannot be validated without
  the intra-event growth curve (§6). R-014's PRELIM→FINAL gap p50 ≈ 5.4 s is an
  *endpoint* gap, not a 300 ms→2 s trajectory — it tells us events last several
  seconds but nothing about how PGA accumulates inside the first 2 s.
- **Timestamps that exist:** `onset_ts`, publish `ts`, `detrigger_ts` (FINAL).
  **Missing:** any intermediate crossing timestamp; the time at which PGA
  crosses a candidate provisional threshold.
- **Can we reconstruct onset → PRELIM-1 → PRELIM-2 → FINAL from history? NO.**
  We have onset and the two endpoints; the PRELIM-2 crossing is not recorded and
  cannot be synthesised from two points (SOURCE FACT + INFERENCE).

## 6. Historical-data feasibility

**OBSERVABLE from existing data (per event):** PRELIM PGA (onset-instant), FINAL
PGA (peak), both timestamps, PRELIM→FINAL gap, event duration, `event_id`,
`obs_seq`, `node_id`, phase, state/revision, validity — all reconstructable by
the read-only replay path (pass ac7b96e8544d3ba5f; `replay.go` §9).

**NOT observable — REQUIRES NEW INSTRUMENTATION:** the PGA trajectory between
the two endpoints. With exactly two points per event (the first at ~300 ms, the
second at detrigger), the questions "what would happen at 1 s / 2 s / 3 s?" are
**UNANSWERABLE** from history. This is the single fact that blocks an offline
validation of PRELIM-2 timing (see §20 blocker 1).

Do **not** treat the stored PRELIM/FINAL pair as a growth curve — it is two
disconnected samples, and PRELIM is an onset-instant reading, not even the start
of a monotone curve.

## 7. Threshold / sensitivity analysis

Candidate structures `PRELIM-1 < PRELIM-2 < FINAL` and `PRELIM-1 = detection /
PRELIM-2 = warning / FINAL = confirmation` are coherent, but:
- Without the growth curve we **cannot estimate** how often a real event's PGA
  at T=1–2 s lands between any two candidate thresholds. (UNKNOWN)
- **Sensitivity is a threshold *value*; latency is a window *time*.** Both are
  single scalars already exposed by a single PRELIM. (INFERENCE) A second window
  improves the sensitivity/latency frontier over a single re-timed PRELIM **only
  if** PRELIM-2 adds *information* (a persistence decision that rejects
  transients a single early sample would accept) — not merely a second cut on
  the same signal.
- Because today's PRELIM is onset-instant and systematically **below** the peak,
  any PRELIM-based floor must be set *lower* to admit the same events a 140-gal
  FINAL floor admits — which increases false-warning exposure (see §8).

**Finding:** the two-window architecture does **not** self-evidently beat a
single tuned PRELIM. That comparison is exactly research-Q1 and is currently
undecidable without new data.

## 8. False-warning analysis (critical)

Information available at each decision point, and how FP risk moves:
- **300 ms (PRELIM-1):** onset-instant PGA + STA/LTA ratio. Least information;
  cannot distinguish a sharp transient (door slam, kick) from a quake onset.
  Highest FP exposure. (SOURCE FACT: PRELIM = onset sample.)
- **1–2 s (PRELIM-2):** whatever the window decides on — UNKNOWN today because
  the intra-window samples are not captured.
- **FINAL:** full event peak + duration; lowest FP exposure; but latest (fires
  at detrigger, after local shaking). (SOURCE FACT.)

FP risk rises **monotonically** as the decision moves earlier (INFERENCE:
strictly less information). And because a provisional floor must sit below the
peak to be useful, the earlier decision also runs at a lower gal threshold —
compounding exposure.

**Is Battery A sufficient to validate PRELIM-2? NO — twice over:**
1. The metric-D battery was **never executed** and the decisive 100–140 gal row
   is **empty** (`07`, `08`, `battery_runsheet.md`).
2. Even if run as designed, it records the **FINAL** PGA of each disturbance
   (`battery_runsheet.md:96-98`) — the wrong quantity. Validating a provisional
   window needs each disturbance's **PGA at ~1–2 s**, i.e. the same intra-event
   curve that history lacks (§6).

**Required additional evidence (not invented here):** a disturbance battery run
**with intra-event PGA logging** so each ordinary disturbance yields a curve, not
just a FINAL peak. No false-positive *rate* may be claimed — n is two anchor
events (`04-data-plan.md`); counts only, per `05-experiment-plan.md` §2.

## 9. Event lifecycle analysis (12 paths)

For each: what server/Android should do, whether an alert can fire, and whether
revision/`event_id` stay valid. `event_id` is stable for life
(`tracker.go:635`) and revision rises only on real transitions
(`tracker.go:769`) — both hold across every path below.

1. **P1→P2→FINAL (normal):** provisional alarm at P2, UPDATE at FINAL via higher
   revision (PGA ratchets up, `tracker.go:723`). Android re-raises on higher
   revision (`AlertDedup.kt:43`). Valid.
2. **P1→(no P2)→FINAL:** falls back to today's FINAL path. Valid, no regression.
3. **P1→P2→weak FINAL:** provisional alarm fired, FINAL below floor. Withdrawal
   needed. EVENT_RESOLVED+CANCELLED bypasses gate/expiry (`AlertRaiser.kt:51-60`)
   — **BUT** the all-clear FCM goes only to the ever-CONFIRMED audience
   (`emit.go:112`); a provisional-only event never CONFIRMED, so **no FCM
   all-clear is sent** → **stale-warning risk** (§20 blocker 5).
4. **P1→P2→event resolves:** same as (3); same stand-down gap.
5. **P2 qualifies → FINAL grows a lot:** UPDATE via revision works cleanly (peak
   ratchets up, broadcast intensity rises). Valid.
6. **Multiple simultaneous events:** D-020 board holds max 3 slots keyed by
   `event_id` (`ActiveAlertBoard.kt`); provisional slots coexist. Valid, but see
   sound/promotion rules — a silent promotion on stand-down must not sound.
7. **Resolution after provisional notice:** = (3); depends on the all-clear fix.
8. **Duplicate PRELIM frames:** dedup keys on `node_id + phase` server-side and
   `type:eventId` client-side. **Two PRELIM phases sharing the string "PRELIM"
   would dedup-collide** — PRELIM-2 needs a distinct phase value (§21).
9. **Reconnect/retry/reordering:** revision + `obs_seq` ordering handle it;
   QoS 0 means a provisional frame can be lost (arrives late or never). Alarm
   must tolerate loss. Valid with care.
10. **Stale PRELIM:** D-018 `validity_ms` + client `isActionable` gate it
    (`WsAlertMessage.kt:151-160`). A provisional frame **must** carry a short
    validity or it rides the 15-min legacy window — too long for provisional.
11. **Malformed PRELIM:** unknown `type` dropped; unknown `event_state` kept as
    null. A new phase must travel in `event_state` (kept), not `type` (dropped)
    — §K/§12.
12. **Node reboot during event:** `obs_seq = (boot_count<<16)|seq` preserves
    identity across reboot (`trigger.schema.json:61-65`). Valid.

## 10. Server architecture impact

- **PRELIM-2 cannot ride the state machine.** `classify()` ignores phase
  (`classify.go:10-21`); a provisional phase changes no state. To alarm it must
  use a side channel like the D-036 trusted-local hook or the D-037 edge —
  which today are **FINAL-gated** (`admin_node.go:110`; edge on FINAL flip,
  `tracker.go:422-424`). A new provisional edge (phase==PRELIM-2 && threshold &&
  UNCONFIRMED && guard) would be required. (INFERENCE from SOURCE FACTs.)
- **algo_ver does not encode the phase gate** (`store.go:71-73`), so changing
  what phase may alarm needs an algo_ver ruling (R-008 / F-11).
- **Replay is edge-blind:** `replayRecorder` implements only `EmitTransition`,
  not `adminEdgeEmitter` (`replay.go:207-211`), so today's offline harness does
  not even surface the FINAL edge — a provisional edge would be invisible to
  replay until the harness is taught to emit it (read-only change; §18).

## 11. Firmware impact

**Minimum change to *ship* two windows** (SOURCE FACT there are only two slots):
a third pending slot, one timer armed at the candidate T after confirmation to
snapshot peak-so-far, a new wire `phase` value, and possibly an onset-relative
timestamp. **No** change to STA/LTA, trigger/detrigger, or `event_id`/`obs_seq`.

**Can the concept be evaluated offline *first*, without shipping firmware?**
Partially — but only after a **measurement-only** firmware instrumentation
captures the intra-event curve. Per `13-evidence-plan` E4/D1 this is a
serial-only, `#ifdef`-guarded, default-OFF log that leaves production bytes
identical. That instrumentation is the prerequisite even for the offline
experiment, because the curve exists nowhere else (§6). Shipping the production
two-window firmware is a later, separate step and is NO-GO now.

## 12. Android impact

- **Cannot alarm on a provisional frame today** — the advisory drop
  (`AlertRaiser.kt:71`) is unconditional. Enabling it requires either lifting
  that drop or adding a `decideFor` branch (like the trusted-local one).
- **A whole new notification architecture is NOT required.** revision-based
  UPDATE, D-018 validity, dedup, and D-020 coexistence already support
  provisional→update→withdraw (pass af6a6626654b876fa bottom-line (a)).
- **Two concrete gaps:** (i) the stand-down all-clear audience is
  ever-CONFIRMED-only (§9 path 3) — a provisional-only alarm would not get its
  FCM all-clear; (ii) a provisional frame must carry a short `validity_ms` or it
  inherits the 15-min legacy window.
- The new phase must travel in the additive `event_state` (unknown value kept,
  `AlertMappers.kt:88-89`), never in the frozen `type` (unknown dropped) — so
  old installs degrade safely.

## 13. Admin Node impact

Options: (A) Admin-only, (B) generic sensor architecture, (C) server-side
generic provisional evidence with Admin Node as first consumer.
- Today's trusted-local path **is already "Admin-only"** (D-036/D-037), 20 km,
  token-only, never GeoTopic (`event_frame.go:82-121`). Starting Admin-only is
  the lowest blast radius and reuses the existing choke point.
- **Risk to avoid (INFERENCE):** baking Admin-only assumptions into the *wire*
  (phase enum, `event_state`, validity) that later block generalisation. The
  contract-level phase should be node-agnostic even if only the Admin path
  consumes it first — i.e. option (C) at the contract layer, (A) at the policy
  layer. Do not assume Admin-only is the final architecture.

## 14. Generic-node / consensus implications

**Keep LOCAL PROVISIONAL WARNING strictly separate from DISTRIBUTED
CONFIRMATION.** (INFERENCE, strongly supported by SOURCE FACTs)
- `classify()` is the consensus gate and reads only peak-PGA + node count +
  independent cells. **PRELIM-2 must not feed `classify`** — letting a single
  early sample move CONFIRMED would let one node's transient trip network-wide
  alarm, defeating the ≥3-node / ≥2-cell quorum.
- PRELIM-2 as *consensus* evidence (multi-node provisional, independence-cell
  interaction, promotion to CONFIRMED) is a **much larger, separate
  architecture** and is **out of scope** for a first experiment.
- A safe first step treats PRELIM-2 only as *local* provisional evidence for the
  single-node trusted-local channel, never as a consensus input.

## 15. EEW lead-time implications (the central question)

The owner's core complaint is SOURCE-FACT correct: the trusted-local warning is
FINAL-gated (`admin_node.go:110`), FINAL fires at detrigger (`sensor.cpp:255`),
and detrigger means the STA/LTA ratio has fallen back below 1.5 — i.e. **local
shaking has largely subsided.** So today's local warning is not "early" for the
node's own location.

Moving the decision to PRELIM-2 (1–2 s) **does reduce the sensor-local decision
latency.** But EEW lead time must be decomposed (do NOT equate the PRELIM→FINAL
gap with lead time):
- **sensor-local decision latency** — improved by an earlier window. (the only
  thing two-window directly changes)
- **server latency** — T3/T4 ~tens of ms (`11-hardware-and-timing.md`).
- **notification latency** — T9 audible tail +467 ms locked / +960 ms FCM-revived
  (`11-hardware-and-timing.md`); a hard floor no window change can beat.
- **target-site seismic arrival** — geometry. **For the single co-located bench
  node the "target" is the node's own site, so lead time ≈ 0 by definition** —
  the shaking is already present when the node decides. (SOURCE FACT: single
  node, 20 km token radius.) Real lead time only appears for targets *away* from
  the source, which requires distant nodes — a multi-node/consensus problem
  (§14), not something the two-window change delivers on its own.

**Finding:** two-window PRELIM can improve *decision* latency and thus help
targets at some distance, but for the current single-node deployment its EEW
lead-time benefit at the protected location is bounded by geometry and the FCM
tail, and is **UNKNOWN in magnitude** without the timing dataset.

## 16. Alternative architectures (described, not ranked)

| # | Design | Latency | Evidence quality | FP exposure | Impl. complexity | Compat w/ current | New evidence needed |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | Current FINAL-only | worst (detrigger) | highest (full peak) | lowest | none | native | none |
| 2 | Single PRELIM warning | best (onset ~300 ms) | lowest (onset sample) | highest | firmware phase-gate + Android gate | needs phase gate | growth curve + FP battery |
| 3 | Two-window P1/P2 | mid (P2 at T) | mid (if P2 is a decision) | mid | firmware timer+slot + server edge + Android gate | most changes | growth curve + FP battery + P2 semantics |
| 4 | PRELIM-armed + FINAL confirm | onset arm, FINAL fire | high | low–mid | server arm logic | moderate | arm/confirm rule validation |
| 5 | PRELIM as evidence only, no direct warning | n/a (no earlier alarm) | n/a | none added | instrumentation only | fully | growth curve (for study) |
| 6 | Generic provisional evidence, all nodes | varies | varies | varies (consensus-gated) | largest (consensus change) | new event model risk | multi-node dataset |

(No "best" is declared; ranking is exactly what the offline experiment must
decide. Note design 5 is the no-regrets instrumentation path — it produces the
missing curve without any alarm-semantics change.)

## 17. Minimum offline experiment (if the concept survives)

Ordered, offline-first, no production/firmware ship:
1. **Fix PRELIM-2 semantics on paper** — decide whether it is peak-so-far or a
   decision, and record it (§4). If it is merely peak-so-far, re-scope the whole
   effort to "single re-timed PRELIM" and run design 2's experiment instead.
2. **Build measurement-only instrumentation** — serial, `#ifdef`, default-OFF,
   byte-identical production (`13-evidence-plan` E4/D1) — to log intra-event PGA
   at fixed intervals. This is the only source of the growth curve.
3. **Collect curves** on two sides: (a) opportunistic real seismic events; (b) a
   disturbance battery run *with* the instrumentation (fixes §8's wrong-quantity
   problem).
4. **Offline replay** over a window grid {300 ms, 500 ms, 1 s, 1.5 s, 2 s, 3 s}
   × a threshold grid, using the read-only floor-sweep harness extended to
   evaluate a provisional edge (§18). Candidate values, not approved values.
5. **Metrics:** detection gain vs 140-FINAL baseline; latency gain (onset →
   eligibility) per window; **false-warning count** per (window, threshold);
   report max + n, never a p95 or rate below n (`05-experiment-plan.md` §2).
6. **Stopping criterion:** any (window, threshold) that admits a disturbance at
   or above its provisional floor **fails** that cell.
7. **Acceptance criterion:** a candidate two-window config must beat a single
   re-timed PRELIM (design 2) on the latency/FP frontier by an owner-set margin,
   or the second window is not justified (research-Q1).

## 18. Required new instrumentation

- **Intra-event PGA growth curve** — the one artifact nothing today provides
  (§6). Serial-only, default-OFF, production-identical.
- **Disturbance-curve battery** — the metric-D battery revised to log the curve,
  not just FINAL (§8).
- **Replay extension** — teach the read-only replay harness to emit/evaluate a
  provisional edge (`replayRecorder` currently only does `EmitTransition`,
  `replay.go:207-211`). Read-only, research-only.
- **Sample-rate measurement** — close the F-10 / R-013 discrepancy (§5); the
  "1–2 s" figure is uncalibrated until the true rate is known.

## 19. Required governance (enumerated, none created here)

Per PROJECT_RULES and prior rulings, implementation would require, in order:
1. **Owner decision** that provisional-alarm semantics are acceptable at all
   (PROJECT_RULES §9 — owner owns safety/alert semantics).
2. **A new D-number** for the provisional phase gate and edge.
3. **An algo_ver ruling** (R-008) — the phase gate is not encoded in algo_ver
   today (F-11), so changing what may alarm needs a version decision.
4. **A contract amendment** — a new node-agnostic phase value carried in
   `event_state` (additive), plus a short `validity_ms` policy for provisional.
5. **A schema migration** if provisional observations are persisted distinctly.
6. **Firmware / Android / server change gates** — each its own review.
7. **Battery validation** — the false-warning dataset (§8, §18).
8. **Release gate** — `14-release-readiness-checklist.md`.

## 20. Blocking issues

1. **No intra-event growth curve exists** (§6) — the core premise (1–2 s carries
   more actionable evidence) is unmeasurable from history. *Blocks offline
   validation.*
2. **PRELIM is onset-instant, but contract + comments call it "peak-so-far"**
   (`trigger.schema.json:55`, `sensor.cpp:223-225` vs `sensor.cpp:214,227`) — a
   semantic defect that must be corrected before any design, or PRELIM-2 will be
   defined against a wrong baseline.
3. **False-warning evidence is empty AND the planned battery measures the wrong
   quantity** (§8) — provisional FP risk cannot be bounded today.
4. **Sample rate UNKNOWN** (§5; F-10 contradicts R-013) — "1–2 s" is
   uncalibrated.
5. **Provisional-only all-clear gap** — stand-down FCM is ever-CONFIRMED-only
   (`emit.go:112`); a provisional alarm that never CONFIRMs would not be
   withdrawn to its audience (§9 path 3). *Blocks safe alarming.*
6. **PRELIM-2 semantics underspecified** (§4) — cannot design against four
   candidate meanings.

## 21. Non-blocking issues

- **Phase dedup collision** — two PRELIM phases sharing the string "PRELIM"
  would collide on the `node_id+phase` / `type:eventId` dedup keys; fixed by a
  distinct phase enum value (§9 path 8).
- **Replay edge-blindness** — fixable with a read-only harness extension (§18).
- **Admin-only vs generic** — a deferrable design choice; keep the wire
  node-agnostic (§13) and it stays open.
- **revision / event_id semantics** — already sufficient for
  provisional→update→withdraw; no change needed.

## 22. Explicit GO / NO-GO / INSUFFICIENT-EVIDENCE status

| Track | Status | Why |
| --- | --- | --- |
| **Offline analysis** | **CONDITIONAL GO for *design*; INSUFFICIENT-EVIDENCE for a conclusive replay** | The replay grid (§17) cannot run until the growth curve exists; designing the experiment + instrumentation is allowed and useful now. |
| **Physical battery testing** | **NO-GO as designed** | Metric-D battery measures FINAL, not the intra-event curve (§8); a revised, instrumented battery is required first, and the owner has declined the current battery. |
| **Firmware implementation** | **NO-GO** | Premise unvalidated; only measurement-only, default-OFF instrumentation is even a candidate, and only after §17 step 1–2. |
| **Server implementation** | **NO-GO** | Needs a new provisional edge + algo_ver ruling + D-number; none justified without §17 evidence. |
| **Android implementation** | **NO-GO** | Advisory drop + all-clear-audience gap must be resolved by design first (§12, §20 blocker 5). |
| **Production deployment** | **NO-GO** | Every upstream track is NO-GO or conditional; floor stays 140.0 (D-007). |

---

## Decision Gate — exact evidence required before implementing two-window PRELIM

Implementation may not begin until **all** of the following exist, in order:

1. **Semantic ruling (paper).** A written, owner-approved definition of PRELIM-2
   as either (a) peak-so-far — in which case re-scope to a single re-timed
   PRELIM — or (b) a persistence/stability *decision*. Plus correction of the
   PRELIM "peak-so-far" mislabel in the contract and firmware comments.
2. **Measured sample rate.** The true GPIO15/DMP rate, closing F-10 and the
   R-013 discrepancy, so window times are calibrated.
3. **Intra-event PGA growth curves** for an owner-set N of real seismic events
   **and** for the disturbance battery — captured by serial-only, default-OFF,
   production-identical instrumentation. (This dataset does not exist today and
   is the gate's centre of gravity.)
4. **Offline replay result** over the window × threshold grid showing a
   two-window config beats a single re-timed PRELIM on the latency/false-warning
   frontier, with n stated and no rate claimed below n.
5. **False-warning gate met:** zero disturbance reaches or exceeds its candidate
   provisional floor at the candidate window, across the instrumented battery.
6. **Resolved provisional stand-down path:** an all-clear that reaches the
   audience of a provisional-only (never-CONFIRMED) alarm — fixing the
   ever-CONFIRMED-only FCM audience.
7. **Governance approvals** from §19: owner sign-off on provisional-alarm
   semantics, a D-number, an algo_ver ruling, and the contract amendment —
   created only after 1–6 pass.

Until every item is satisfied, the recorded stance stands: **INSUFFICIENT
EVIDENCE — do not implement; floor stays 140.0.**



