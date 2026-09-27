# QuakeAlert Release Readiness Master Audit

Date: 2026-09-27 UTC. Status: AUDIT / RESEARCH / DECISION-FRAMEWORK ONLY.
Scope note: this document decides nothing operationally on its own. Nothing here was
deployed, merged, pushed, flashed, rotated, or cleaned. It reports the state and names
the decisions the owner must make. Discrepancies between documentation and source are
reported, not silently reconciled.

Evidence labels used throughout:
- [FACT] verified this session by direct read of source or git.
- [MEAS] a measurement (offline replay or a source-cited subagent finding). Subagent
  findings are model output; where I did not re-read the cited source myself I say so.
- [INFER] interpretation built on the labelled facts and measurements.
- [UNKNOWN] not answerable from available evidence.

Public strong-motion PGA is NOT treated as equivalent to QuakeAlert MPU6050 PGA. The
candidate thresholds 60/80/100/120/140 gal are used for COMPARISON ONLY; no floor is
concluded to be safe or optimal or production-ready by this document.

---

## 1. Executive Summary

Overall release decision: **NOT READY**, decomposed into two distinct releases that the
project conflates but that have different blockers:

- **Open-source code publication (making the repo public): NOT READY.** Four hard
  blockers, all verified this session: (B1) real key material sits in an untracked but
  NOT git-ignored working-tree file; (B2) no LICENSE; (B3) no root README; (B4) a
  .gitignore gap that leaves the secret-bearing transcripts one `git add .` away from
  being committed. See sections 11, 13, 14.

- **Operational limited service release (the private single-node VPS release the owner
  already approved on 2026-09-07): READY WITH CONDITIONS.** The service runs, but two
  issues must be resolved or explicitly accepted first: (1) a verified life-safety-adjacent
  defect where a trusted-local Admin Node alarm on a never-CONFIRMED event has no FCM
  all-clear to backgrounded devices (section 6.1, 9); (2) a production domain mismatch
  between the Android release build and every server/deploy default (section 9).

The two algorithm decisions the owner must ratify are framed with evidence matrices in
section 6. My defensible initial-operating recommendations: keep the Admin Node local
floor at 140 gal as a conscious initial operating point (not a default-preserve), and
keep PRELIM timing unchanged for the initial release. Both are reversible and both are
paired with the exact owner decision and the evidence that is missing.

---

## 2. Current Project State

Source of truth: `docs/CURRENT_STATE.md` (currently modified and uncommitted). System is
a single-node early-warning stack on one private VPS (`api.quakealert.web.id` per the app;
see the domain discrepancy in section 9). No CONFIRMED event has ever occurred in
production; no population-scale latency, false-positive, false-negative, or lead-time rate
has ever been measured [FACT, from CURRENT_STATE.md].

**Firmware (ESP32, C++/Arduino).** One board, one location, one build. Detector is
STA/LTA on MPU6050 vector magnitude, confirm at 300 ms, detrigger at ratio < 1.5. PRELIM
is a single onset-instant sample; FINAL is the true running peak at detrigger [FACT,
sensor.cpp:214,227,252]. Sample rate measured 100 Hz (R-013). Two publishes per event,
shared obs_seq, v2 signed canonical string, HMAC-SHA256.

**Server (Go 1.24).** Runs Phase-3 event.Tracker or legacy Phase-2 consensus.Engine,
selected by `EVENT_TRACKER_ENABLED`; both compiled in for one-release rollback [MEAS,
server audit, main.go:134-223]. Five-state lifecycle DETECTED / UNCONFIRMED / CONFIRMED /
RESOLVED / CANCELLED, legal-transition table enforced. Confirmation gate: PGA >= 16.6 gal
AND >= 3 nodes AND >= 2 independent cells within 5 km [FACT, CURRENT_STATE.md]. This gate
is unreachable with one node, so no real CONFIRMED alert can fire in the current fleet
[INFER, stated in CURRENT_STATE.md]. Admin Node local warning (D-036/D-037) gates on the
admin node's own FINAL contribution PeakPGA >= 140 gal (compile-time const, D-007) [FACT,
verified emit.go and admin_node.go this session]. HMAC verification is present and
constant-time; MQTT requires TLS in production; admin routes gated behind `ADMIN_API_KEY`
[MEAS, server audit, source-cited].

**Android (Kotlin).** `applicationId id.web.quakealert`, `versionCode 1`, `versionName
1.0` (never bumped for release) [MEAS, build.gradle.kts:43-47]. Release build points at
`https://api.quakealert.web.id/` [FACT, build.gradle.kts:127, verified this session].
Advisory / UNCONFIRMED frames are kept for banner display but deliberately cannot raise a
full-screen alarm; drill (`is_test`) frames are dropped on release builds [MEAS, Android
audit, AlertRaiser.kt:71, AlertMappers.kt:44-45]. Release signing is env-driven; no
keystore is committed. Only `ExampleInstrumentedTest.kt` exists as an instrumented test
[FACT, CURRENT_STATE.md]; unit and lint coverage exists.

**Infrastructure (deploy/).** Docker Compose production stack: Caddy (80/443) and
Mosquitto (8883) exposed; PostGIS, Redis, and the server have no published ports. All
secrets are `${VAR:?}` fail-if-unset; PostGIS pinned by digest; production Mosquitto is
MQTTS-only with anonymous disabled and default-deny ACLs [MEAS, infra audit, source-cited].
Real production secrets are correctly git-ignored and none are committed [MEAS+FACT,
confirmed by both the secret-scan audit and my own git checks].

**Documentation.** Extensive: PROJECT_RULES.md, ROADMAP.md, docs/adr/0001-0004, several
SPEC docs, DECISIONS.md, CURRENT_STATE.md, and the docs/planning/admin-node-threshold
research area (docs 00-16, RESEARCH_LEDGER R-001..R-022, CHANGELOG). No root README,
LICENSE, CONTRIBUTING, SECURITY, or CODE_OF_CONDUCT [FACT, verified this session].

**Testing.** ~90 Go unit test files, byte-exact HMAC golden cross-check with firmware,
CI runs server/firmware-host/Android/release-checks/simulation jobs. CI performs no
deploy (`permissions: contents: read`). No dedicated secret-scanner job. No field
validation of any life-safety claim; simulation is explicitly software-only [MEAS,
CI/infra audit; FACT, CURRENT_STATE.md].

---

## 3. Completed Phases and Decisions

Decisions of record live in `docs/DECISIONS.md` (D-001..D-037) and the threshold research
ledger (R-001..R-022). Load-bearing items:

- **D-007:** the Admin Node PGA floor is a compile-time constant, deliberately not
  runtime-tunable, so it cannot be lowered mid-incident [FACT, admin_node.go:18-20,30].
- **D-036 (ACCEPTED 2026-09-09):** Admin Node local-warning capability adopted. The
  committed decision text in DECISIONS.md still reads "NOT activated" with three activation
  preconditions, and records `NODE-52960B47` failing the 5-minute heartbeat gate as of
  2026-09-09. See the discrepancy in section 4.
- **D-037:** trusted-local FINAL-edge exception (allows a trusted-local frame on the
  FINAL-edge non-transition while the event is UNCONFIRMED).
- **Phase 4 (Self-Measurement / Forensics):** IN_PROGRESS; sub-milestones P4-M1'..M6'
  owner-approved SATISFIED. Phase F (multi-node field validation) BLOCKED on node density.
- **R-022 (2026-09-27):** public-waveform offline replay result, PRELIMINARY, partially
  discharges the R-021 two-window decision gate on public data only.

### 3.1 Consolidated roadmap (A through G)

**A. Already complete.** Firmware detector and two-phase publish; Go five-state lifecycle,
consensus gate, dispatch tier; Android client with advisory-never-wakes enforcement;
MQTT/HMAC/anti-replay ingest; DB migrations 000001-000010; CI (build/test/lint/contract-
drift/sim); Admin Node capability code (D-036) and FINAL-edge (D-037); offline research
harnesses (admin_floor_sweep.go, replay_growth.py); doc 16 public-waveform study.

**B. Complete but requiring final verification.** Admin Node activation state (DECISIONS.md
says NOT activated, CURRENT_STATE.md says live since 2026-09-09T~21:15Z, section 4);
production domain (`web.id` vs `.id`, section 9); the trusted-local all-clear path
(verified defective, section 6.1/9); migration deploy path (`server/migrations/` empty,
real migrations under `contracts/db/migrations/`).

**C. In progress.** Phase 4 self-measurement/forensics; threshold research (population
counts pending the owner-run read-only sweep); PRELIM-timing evidence (offline only).

**D. Required before release.** Open-source: remove secret-bearing transcripts, add
LICENSE, add README, close the .gitignore gap (section 15). Operational: resolve or
explicitly accept the trusted-local all-clear asymmetry; confirm the production domain;
ratify the Admin Node floor and PRELIM-timing decisions.

**E. Recommended but not release-blocking.** CI secret-scanner job; bump Android
versionCode/versionName; correct the PRELIM "peak-so-far" doc/contract text; CONTRIBUTING,
SECURITY, CODE_OF_CONDUCT; repair the D-036/D-037 governance record.

**F. Post-release validation.** Population latency, false-positive/negative and lead-time
observation once real nodes and events exist (Phase F, currently BLOCKED on density).

**G. Future research.** Two-window PRELIM (currently unsupported over re-timing, section
6.3); on-device PGA-growth capture; threshold field tuning with measured false-warning cost.

---

## 4. Current Git State

[FACT, all verified this session unless noted.]

- Current branch: `development`. Clean versus `origin/development` (0 ahead, 0 behind).
- `development` is 74 ahead / 0 behind `origin/main`; main is a strict ancestor, so a
  fast-forward is possible. `phase-1-observation-ledger` is a fully merged stale ancestor
  (0 unique commits). No tags, no releases.
- Modified but uncommitted tracked files: `docs/CURRENT_STATE.md` and `docs/DECISIONS.md`.
- The entire `docs/planning/` tree is UNTRACKED (all threshold and this-audit research
  history is uncommitted). Also untracked: `.hermes/` (empty dirs), `deploy/README.md`,
  `docs/evidence/live-eq-2026-09-12-001/`, `server/scripts/admin_floor_sweep.go`, six
  root `session-ses_*` transcripts, and `firmware/src/2026-09-27-045555-this-session-is-
  being.txt`.

**DISCREPANCY D-A (report, do not reconcile): Admin Node activation state.** The
uncommitted `DECISIONS.md` D-036 entry still literally reads "**NOT activated**" (as of
2026-09-09, listing three unmet activation preconditions, including `NODE-52960B47`
failing the 5-minute heartbeat gate). The uncommitted `CURRENT_STATE.md` states the
opposite: `NODE-52960B47` was "designated sole Admin Node 2026-09-09T~21:15Z (verified,
heartbeat fresh)... Local-warning path active." Both edits are uncommitted. Whether a
later same-day activation superseded the D-036 "NOT activated" text is not recorded in a
way that reconciles the two files. The owner must confirm the governance trail and, if the
node is genuinely live, amend the D-036 record (append-only) rather than leave it reading
"NOT activated."

**DISCREPANCY D-B: uncommitted research history.** Because `docs/planning/` is entirely
untracked, none of the threshold or two-window or public-waveform research (docs 00-16,
the ledger, this audit) is under version control. This is not a code risk but it means the
decision history that justifies the release posture is not yet captured in git.

---

## 5. Branch and Merge Readiness

[FACT.] A `development -> main` fast-forward is mechanically clean (main is a strict
ancestor, 74 commits behind, no divergence). The governance rules for this task forbid
push, merge, branch deletion, and history rewrite, so no merge was performed and none is
recommended as part of this audit.

Readiness caveats before any future merge (owner decisions, not actions I will take):
1. The uncommitted tracked files (`CURRENT_STATE.md`, `DECISIONS.md`) and the untracked
   `docs/planning/` history should be committed deliberately, NOT via `git add .`, because
   a blanket add would also stage the secret-bearing `session-ses_f6d9.md` (section 13).
2. The D-036 activation discrepancy (section 4) should be resolved in the record before it
   is merged, so main does not carry a self-contradicting governance history.
3. No tag or GitHub release exists; creating one is a release step (section 15), not a
   prerequisite for the code being correct.

---

## 6. Critical Algorithm Decisions

### 6.1 Admin Node PGA threshold (initial operational configuration)

**What the threshold actually is.** [FACT, verified emit.go / admin_node.go this session.]
`AdminNodeMinPGAGal = 140.0` is a compile-time const (D-007). It gates the trusted-local
warning on the admin node's OWN FINAL contribution PeakPGA (the true running peak at
detrigger, not the onset-instant PRELIM, and not the multi-node event peak). It is distinct
from the confirmation-detector floor `MinPGAGal` (16.6 gal). When eligible, the path sends
a HIGH-priority `EARTHQUAKE_ALERT` FCM to tokens within 20 km while the event is UNCONFIRMED.

**Verified defect that bears directly on the floor choice.** [FACT, verified this session:
emit.go:108-112, admin_node.go:176-179.] The trusted-local alarm fires only on
`To == UNCONFIRMED`. The RESOLVED/CANCELLED stand-down sets `push = s.EverConfirmed`, and
`EverConfirmed` is set true only at CONFIRMED. A single-node trusted-local event never
reaches CONFIRMED, so `EverConfirmed == false`, so **its all-clear is sent over WebSocket
only, never over FCM.** Backgrounded devices within 20 km that received the HIGH-priority
alarm get no FCM all-clear. The in-code comment ("an event that was never CONFIRMED never
woke anyone, so there is no one to reassure") is contradicted by the D-036/D-037 path,
which can wake exactly that audience. This is reported as a discrepancy and is a release
blocker (section 9), not silently reconciled.

**Evidence on the floor value itself.**
- [MEAS, doc 16 / R-022, PRELIMINARY, small sample, confirmed-event-conditioned bias.]
  Offline public-waveform T2 replay (52 records). At the onset-instant PRELIM the median
  PGA is ~7% of FINAL. Reach-ever counts among the 52: floor 60 -> 15 records, 80 -> 12,
  100 -> 8, 120 -> 7, 140 -> 5. Median time to first cross: 2.87 s (60) rising to 4.35 s
  (140). Public strong-motion PGA is not QuakeAlert MPU6050 PGA; these are comparison-only.
- [FACT, doc 07.] 140 gal sits just above the "strong" shaking label boundary (137.2 gal);
  candidates 60-140 all map to MMI V-VI.
- [UNKNOWN.] The false-warning rate at any floor. No CONFIRMED event has ever occurred; no
  population false-positive/negative or latency has been measured. The detection-count
  sweep (admin_floor_sweep.go, R-019) is READY but its population numbers are pending the
  owner-run read-only query; it answers detection only, never false-warning cost.

**Decision I can defensibly make now: keep 140 gal as the initial operating point.** This
is NOT an auto-preserve of the current value; it rests on affirmative reasons:
1. [INFER] While the all-clear defect above is unfixed, every trusted-local alarm on a
   never-CONFIRMED event risks a HIGH-priority wake with no FCM stand-down. A higher floor
   minimizes the count of such un-cleared alarms, so lowering the floor now increases
   exposure to the exact defect that is open.
2. [INFER, on FACT] 140 gal corresponds to genuinely strong shaking (MMI ~VI, above the
   137.2 boundary), where a single trusted node's local self-warning is most defensible.
3. [INFER] Lower floors (60-100) push alarms into MMI V, where the single-node
   false-positive cost is UNKNOWN, and the missing all-clear compounds any false alarm.
4. [FACT] There is no evidence that 140 misses events a lower floor would usefully catch at
   acceptable false-warning cost, because that cost is unmeasured.

**Owner decision still required (offline evidence cannot settle it):** ratify 140 gal as
the initial operating floor, OR choose a lower floor accepting an unquantified single-node
false-warning cost. Evidence matrix:

| Floor (gal) | Reach-ever / 52 (public replay, comparison-only) | MMI band | False-warning cost | Missing evidence to justify lowering |
| --- | --- | --- | --- | --- |
| 60  | 15 | V-VI | UNKNOWN | population FP rate; on-device PGA equivalence |
| 80  | 12 | V-VI | UNKNOWN | same |
| 100 | 8  | V-VI | UNKNOWN | same |
| 120 | 7  | V-VI | UNKNOWN | same |
| 140 (current) | 5 | VI (just above "strong") | UNKNOWN but fewest alarms | none needed to hold; this is the conservative point |

Recommendation: hold 140 for the initial release, fix the all-clear defect first, and
revisit the floor only once a measured false-warning cost exists (Phase F, post-release).

### 6.2 PRELIM timing (initial operational configuration)

[FACT.] PRELIM is a single sample captured at the 300 ms confirmation instant
(sensor.cpp:214), not a peak-so-far, despite the comment and contract text that call it
"peak-so-far" (section 6.3 / DISCREPANCY D-C). [MEAS, doc 16.] A later single window
recovers substantially more of FINAL (median ~7% at onset rising to ~49% at +2 s), but
whether the recovered accuracy still leaves "useful warning time" is INSUFFICIENT EVIDENCE
offline, because device-to-user latency is unmeasured.

**Decision I can defensibly make now: keep PRELIM timing unchanged for the initial release.**
Reasons:
1. [FACT] Changing PRELIM timing is a firmware change requiring a flash, which is out of
   scope for this audit and unvalidated on-device.
2. [FACT, verified this session] The trusted-local life-safety warning gates on the FINAL
   contribution (peak at detrigger), NOT on PRELIM. PRELIM feeds only the UNCONFIRMED
   advisory banner, which is WebSocket-only and never raises an alarm. So PRELIM timing has
   low life-safety leverage and does not block release.
3. [INFER] The simplest architecture supported by evidence is preferred; there is no
   evidence that re-timing PRELIM improves any release-relevant outcome today.

**Owner decision required:** authorize (later, separately) a documentation-only correction
of the PRELIM "peak-so-far" text so the contract and firmware comment match the code
(behavior unchanged). This is recommended, not release-blocking.

### 6.3 Two-Window PRELIM

[MEAS, doc 16 / R-021 / R-022.] A second PRELIM window is NOT supported over simply
re-timing a single window (option B in doc 16). The evidence does not justify introducing a
two-window architecture, and the governance rule against adding architecture that merely
"sounds useful" applies. **Recommendation: do not adopt Two-Window PRELIM.** It stays in
roadmap band G (future research), contingent on on-device growth-curve data that does not
yet exist (the R-021 growth-curve blocker).

**DISCREPANCY D-C (report, do not reconcile):** `contracts/mqtt/trigger.schema.json:55`
and `firmware/src/sensor.cpp:223-225` both describe PRELIM `pga` as "peak-so-far"; the code
captures a single onset-instant sample (sensor.cpp:214). Independently noted in the
untracked in-tree file `firmware/src/2026-09-27-...txt`. Reported, not corrected.

---

## 7. Public Waveform Evidence

Full report: `docs/planning/admin-node-threshold/16-two-window-prelim-public-waveform-
research.md` (COMPLETE 2026-09-27, R-022). Method: offline replay of public FDSN
strong-motion records through a T2 detector carrying the exact config.h constants
(`scripts/replay_growth.py`), compared against reference PGA. Not run against the device.

Population [MEAS]: 52 retained records (Ridgecrest M7.1, M6.4, La Habra M5.1); distances
7.1 / 44.0 / 132.1 km; FINAL PGA 4.0 / 27.7 / 243.0 gal. Excluded 326 = 296 archive-gap
FDSNNoDataException (unbiased) plus 30 no-confirmation (outcome-correlated, so the retained
set is conditioned on "confirmed-event"; this bias is disclosed).

Key results [MEAS, PRELIMINARY]:
- Checkpoint/FINAL median ratio grows: onset-instant 0.071; +0.3 s 0.273; +0.5 s 0.324;
  +1.0 s 0.418; +1.5 s 0.469; +2.0 s 0.492; +3.0 s 0.708.
- Time to reach fraction of FINAL (median): 25% at 0.245 s; 50% at 2.415 s; 75% at 3.115 s;
  90% at 4.355 s.
- Zero records cross any candidate threshold at +0.3 s.

Limits (binding on every use of this evidence): small sample, confirmed-event-conditioned,
public PGA is not QuakeAlert PGA, and the "useful warning time" question is INSUFFICIENT
EVIDENCE offline. This evidence supports "a later single window recovers more of FINAL than
the onset-instant PRELIM" and does NOT support any claim that a specific floor is safe or
that a second window is warranted.

---

## 8. Pre-Release vs Post-Release Validation

**What can be validated pre-release (offline / bench / single node):** code correctness
(unit, HMAC golden, contract-drift), the trusted-local all-clear fix once made, the domain
resolution, migration up/down, drill-path FCM delivery (already demonstrated on two vendors,
debug builds), and the offline threshold sweep counts.

**What CANNOT be validated pre-release [FACT, from CURRENT_STATE.md]:** any real CONFIRMED
event (gate unreachable at one node), population latency, measured false-positive /
false-negative / lead-time rates, in-use-device alert policy (U-012), and any multi-node
consensus behavior. These are structurally post-release / Phase F items and must not be
presented as satisfied. The honest release posture is: ship the operational service as an
observational deployment, not as a validated life-safety guarantee, and say so in the app
and README (life-safety disclaimer, section 12).

---

## 9. Release Blockers

Operational service release blockers:
1. **[BLOCKER, verified this session] Trusted-local all-clear asymmetry.** A never-CONFIRMED
   trusted-local HIGH-priority FCM alarm to the 20 km audience has no FCM all-clear
   (emit.go:112 gates on `EverConfirmed`; emitTrustedLocal fires only on UNCONFIRMED,
   admin_node.go:177). Either add a trusted-local withdrawal FCM path, or (owner decision)
   explicitly accept and document the gap before activating the Admin Node for real users.
2. **[BLOCKER, verified this session] Production domain mismatch.** Android release build
   uses `api.quakealert.web.id` (build.gradle.kts:127, asserted canonical by ADR-0003 and
   QuakeApiConfig.kt) while every server/deploy/contract default uses `api.quakealert.id`
   with no `.web` (openapi.yaml:19, deploy/.env.prod.example:15, docker-compose.prod.yml:42,
   config.go:184, the deploy scripts). [UNKNOWN] whether both names resolve to the same VPS
   (no DNS access here). If they are not aliases, release builds reach a different host than
   deploy configures. Owner must confirm DNS/TLS for `web.id` or align the configs.

Open-source publication blockers (verified this session): B1 real secrets in an untracked,
not-ignored file; B2 no LICENSE; B3 no README; B4 .gitignore gap. Detailed in sections
11, 13, 14.

---

## 10. Non-Blocking Unknowns

- [UNKNOWN] On-device MPU6050 PGA vs public strong-motion PGA equivalence. Does not block
  release; blocks any quantitative floor claim.
- [UNKNOWN] Population false-positive/negative, latency, lead-time (post-release, Phase F).
- [UNKNOWN] Whether the live Firebase `quakealert26` API key has console-side package/API
  restrictions (client keys are designed to ship in apps; low risk if restricted).
- [UNKNOWN] In-use-device alert policy (U-012, pre-existing open question).
- [UNKNOWN] Whether the D-036 governance record was meant to be superseded by a later
  same-day activation (section 4, D-A). Non-blocking for code, blocking for a clean record.

---

## 11. Open-Source Readiness

NOT READY. Four hard blockers (all verified this session):
- **B1.** Real key material sits in `session-ses_f6d9.md` (untracked, NOT git-ignored). The
  secret-scan audit reports a process/env dump containing MASTER_KEY_HEX, JWT_SECRET,
  ADMIN_API_KEY, and MQTT_PASSWORD at generated-key lengths. I did NOT open the file or
  reproduce any value, per the no-secret-exposure rule. See section 13.
- **B2.** No LICENSE anywhere [FACT, git ls-files and disk check both empty]. A public repo
  with no license is all-rights-reserved, not open source.
- **B3.** No root README.md [FACT]. Only `docs/brand/README.md` exists.
- **B4.** `.gitignore` gap [FACT, verified]: it lists `session.txt` and `*-hello.txt` but
  not `session-ses_*`, not `docs/planning/`, not the stray firmware transcript. So the
  secret-bearing file in B1 is one `git add .` away from being committed.

Also absent [FACT]: CONTRIBUTING.md, SECURITY.md, CODE_OF_CONDUCT.md. Present internal
material a public consumer does not need: `.clinerules/` (tracked, internal agent rules),
`docs/TEMP_*` files. Owner decides whether to keep as history or tidy.

I did not run `git add`, create a license, write a README, rotate any key, or delete any
file. Those are release steps (section 15) that require explicit owner authorization.

---

## 12. README Gap Analysis

No root README exists. A public README should add, at minimum: a one-line description
(community earthquake early warning, life-safety context); a life-safety disclaimer ("not a
certified life-safety product; observational"); high-level architecture (ESP32 firmware /
Go server / Android / MQTT / Postgres+PostGIS / FCM); repo layout; per-component build and
run (firmware PlatformIO, server Go, Android Gradle); Docker Compose quick-start and env
setup pointing at the `.example` templates; prerequisites and toolchain versions; hardware
BOM and wiring; testing instructions; and pointers to CONTRIBUTING, SECURITY, and LICENSE.
Per the governance rules I did not write the README; this is the gap list only.

---

## 13. Security and Secret Audit

**Committed (tracked) source: clean.** No real production secret is committed [MEAS,
secret-scan audit; corroborated by my own git checks that all classic secret files are
git-ignored]. Tracked matches are dummies/templates only: CI throwaway values
(ci.yml:212,219), dev-default DSN password `devpassword` (config.go:170 and dev scripts),
test fixtures (`test-key`, sim JWT), and `.example` templates with blank values. Real
hostnames and an ops email are tracked (openapi.yaml, config.go, deploy scripts): not
secret, but they reveal production infrastructure (INFO/LOW).

**Working tree (untracked): the real risk.**
- **[HIGH] `session-ses_f6d9.md`** contains real MASTER_KEY_HEX / JWT_SECRET /
  ADMIN_API_KEY / MQTT_PASSWORD (secret-scan audit finding). These are the crown-jewel
  keys. Because the file is untracked but not git-ignored (B4), it is stageable. Per the
  task rules I did NOT rotate or revoke anything and did NOT print any value. Recommendation
  for the owner: treat these keys as potentially exposed, and consider rotating them as a
  separate authorized action (out of scope here). This recommendation carries no user
  authority; it is my assessment.
- **[MED]** Five other `session-ses_*` transcripts and the stray
  `firmware/src/2026-09-27-...txt`: internal audit logs, no secret assignments detected by
  the scan, but they should not be public.
- **[LOW-MED]** Root `google-services-debug.json` / `google-services-release.json`: real
  `quakealert26` Firebase client config. Both are git-ignored [FACT], so contained; ensure
  they are excluded from any release archive too.

**Server security posture [MEAS, server/infra audits, source-cited]:** HMAC verified
(constant-time `hmac.Equal`); MQTT requires TLS in production (`InsecureSkipVerify` not
set); admin routes gated behind `ADMIN_API_KEY` with a min-length check; drill endpoint
topic-isolated with `is_test=true`; production Mosquitto MQTTS-only, anonymous disabled,
default-deny ACLs. Hygiene flags: `config.go:170` falls back to a dev-password DSN if
`DATABASE_URL` is unset (a misconfigured prod deploy would silently target dev creds); dev
DB-direct provisioning scripts (`sim_setup_nodes.go`) must not ship in prod images; no CI
secret-scanner job exists (recommend adding gitleaks/trufflehog given B4).

---

## 14. Production / Test Data Cleanup Requirements

Identify only; delete nothing (governance rule).
- **MUST clean before public / archive:** the six root `session-ses_*` transcripts (~2.8 MB,
  one holds real secrets), and `firmware/src/2026-09-27-...txt`. Add matching patterns to
  `.gitignore` and remove from the working tree.
- **MUST verify excluded from any release archive:** root `google-services-*.json` (real
  Firebase config; already git-ignored).
- **Owner's call (keep-as-history vs tidy):** `.clinerules/` (tracked internal rules),
  `docs/TEMP_*` files.
- **Review before it lands:** `server/scripts/admin_floor_sweep.go` (untracked new file) and
  the untracked `docs/planning/` and `docs/evidence/` trees, for any embedded identity.
- **Keep:** legitimate tooling (`run_e2e_test.sh`, `deploy/scripts/*.sh`, sim scripts) and
  engineering docs/ADRs; they contain only env-var references, no literal secrets.

---

## 15. Required Release Steps (ordered checklist)

None of these were performed. They require explicit owner authorization; several are
destructive or outward-facing and must not be done blindly.

Operational service release:
1. Fix or explicitly accept the trusted-local all-clear asymmetry (section 9.1). If fixing,
   add a trusted-local withdrawal FCM path and a test asserting the 20 km audience is
   reached on RESOLVED/CANCELLED of a never-CONFIRMED trusted-local event.
2. Resolve the production domain (section 9.2): confirm `web.id` DNS/TLS or align configs.
3. Ratify the Admin Node floor decision (section 6.1) and the PRELIM-timing decision
   (section 6.2).
4. Resolve the D-036 activation-record discrepancy (section 4) in the append-only record.
5. Confirm deploy tooling applies `contracts/db/migrations/` (server/migrations is empty).
6. Smoke-test on the target host; then freeze and monitor.

Open-source publication (do before flipping the repo public):
7. Remove the secret-bearing and stray transcripts from the working tree; add
   `session-ses_*`, the stray firmware `.txt`, and any other transcript patterns to
   `.gitignore` (closes B4).
8. Owner: consider rotating the keys exposed in B1 as a separate authorized action.
9. Add a LICENSE (record the choice in a new ADR under docs/adr).
10. Write the root README (section 12) and add CONTRIBUTING / SECURITY / CODE_OF_CONDUCT.
11. Add a CI secret-scanner job.
12. Commit the uncommitted tracked files and the `docs/planning/` history deliberately
    (never `git add .`), then, if desired, fast-forward `development -> main` and tag.

---

## 16. Post-Release Observational Validation Plan

Once the service is live (still a single node, gate unreachable), collect observationally,
without promising alerts: heartbeat liveness and uptime; any UNCONFIRMED/advisory frames
and their cadence; any trusted-local activations and, critically, whether the all-clear
reached devices (validates the section 9.1 fix in the field); drill-path delivery latency
on real installs; and crash/ANR telemetry. As nodes are added (Phase F, currently BLOCKED
on density), begin measuring the items in section 8 that are structurally post-release:
population latency, false-positive/negative, lead-time, and multi-node consensus behavior.
Every such number stays UNKNOWN until measured; do not backfill from simulation.

---

## 17. Final Decision Matrix

| Area | Current State | Evidence | Blocker? | Required Action | Owner Decision? |
| --- | --- | --- | --- | --- | --- |
| Trusted-local all-clear | No FCM stand-down for never-CONFIRMED alarm | [FACT] emit.go:112, admin_node.go:177 | YES (operational) | Add withdrawal path or accept+document | Yes: fix vs accept |
| Production domain | App `web.id`, deploy `.id` | [FACT] build.gradle.kts:127 vs deploy | YES (operational) | Confirm DNS or align | Yes: which host is canonical |
| Admin Node PGA floor | 140 gal compile-time | [MEAS] doc16/R-022; [UNKNOWN] FP rate | No (hold 140) | Ratify 140 as initial point | Yes: ratify vs lower |
| PRELIM timing | Onset-instant single sample | [FACT] sensor.cpp:214; [MEAS] doc16 | No | Keep; doc-fix later | Yes: authorize doc fix |
| Two-Window PRELIM | Not implemented | [MEAS] doc16/R-021 | No | Do not adopt | Confirm defer |
| PRELIM "peak-so-far" text | Contract/comment wrong | [FACT] schema:55, sensor.cpp:223 | No | Doc-only correction | Authorize |
| D-036 activation record | DECISIONS says NOT activated; STATE says live | [FACT] both uncommitted | No (record only) | Reconcile in record | Yes: confirm live |
| Secrets in transcript | Real keys in untracked file | [MEAS] scan; [FACT] not ignored | YES (open-source) | Remove + gitignore + rotate | Yes: rotate |
| LICENSE | Absent | [FACT] | YES (open-source) | Add license | Yes: which license |
| README | Absent | [FACT] | YES (open-source) | Write README | No (just do it) |
| .gitignore gap | `session-ses_*` not ignored | [FACT] | YES (open-source) | Add patterns | No |
| Migrations path | server/migrations empty | [MEAS] server audit | Verify | Point deploy at contracts/db | Confirm |
| Config DSN fallback | dev password default | [MEAS] config.go:170 | No | Require DATABASE_URL in prod | Confirm |
| Android version | versionCode 1 | [FACT] build.gradle.kts:46 | No | Bump before store release | Yes |

---

## 18. Recommended Immediate Next Actions

1. Owner reads sections 6.1, 9, and 13 first: the all-clear defect, the domain mismatch,
   and the exposed keys are the three items that gate everything else.
2. Decide the two operational blockers (all-clear fix vs accept; canonical domain).
3. Ratify the two algorithm decisions (hold 140; keep PRELIM timing), or override with the
   section 17 matrix in hand.
4. Handle the exposed keys (remove the transcript, close the .gitignore gap, consider
   rotation) before any repo goes public.
5. For open source, add LICENSE and README; everything else in band E follows.

Answering the owner's core question (governance rule 20): QuakeAlert's operational service
is READY WITH CONDITIONS (fix or accept the all-clear defect, confirm the domain, ratify the
two decisions); the open-source repo is NOT READY until the four B-blockers clear. Nothing
here was deployed, merged, pushed, flashed, rotated, or cleaned.
