# 02 — Research questions

- Purpose: fix the questions before answers, so evidence cannot be cherry-picked for 100 gal.
- Status: open; none answered in this file.
- Last updated: 2026-09-11 UTC.
- Sources: task brief §§4–13; PROJECT_RULES §§6,8,10,11.

## Primary (must answer all before any recommendation)

1. RQ-ELIG: for each candidate floor (60/80/100/120/140/160+), which historical observations would have become eligible, at which stage (transition vs D-037 edge), and with what PRELIM→FINAL latency? (Method: `06-replay-plan.md`.)
2. RQ-LAT: how much warning time does 140→100 actually buy? Measure motion→detection→eligibility→dispatch→audible-alarm per §10 file, not just server timestamps.
3. RQ-FP: what realistically produces 20/40/60/80/100/120/140/160+ gal at THIS installation, separated OBSERVED / INFERRED / UNKNOWN? (Method: §5 file + hardware tests §9 file.)
4. RQ-EQ: what acceleration levels do relevant earthquakes produce at THIS sensor (magnitude/distance/depth/site/frequency/duration/installation), and which public datasets can be replayed offline? External seismometer ≠ ESP32 — document the gap.
5. RQ-PGA: what does the firmware PGA value physically mean (raw/filter/baseline/vector/STA-LTA/PRELIM-vs-FINAL/quantization/clipping/mounting/duration), and is 100 gal comparable across phases? (Answer in §4 file: NO across PRELIM/FINAL.)
6. RQ-TIME: what is the TRUE GPIO15/DMP sample rate (never assume 100 Hz), and do STA/LTA time constants mean what comments claim?
7. RQ-SAFE: per candidate, what new false-warning modes appear, what genuine events become detectable / still missed, and are D-003/D-006/D-007/D-036/D-037 + `algo_ver` affected?
8. RQ-CRIT: what objective acceptance criteria (false-warning / detection / latency / robustness / simplicity / trust) must a new value meet — PROPOSED, owner-approved before use?

## Explicitly OUT (research must not drift)

- Changing CONFIRMED quorum, 16.6 gal floor, attach/correlation radii, independence rule, alert radius, delivery tiers, notification policy, contracts, `algo_ver` base — all require separate accepted decisions (D-011 out-of-scope list; PROJECT_RULES §9).
- BMKG/USGS/EMSC as ground truth or realtime dependency (PROJECT_RULES §2 — post-event reference only, structurally incapable of affecting realtime, verified by test per S7).
- Automatic false-positive classification or automatic threshold calibration (Phase 4 out-of-scope).
- Manufacturing a CONFIRMED event in production; any lead-time CLAIM (S2); any population RATE claim from n=1 events (§6).

## Allowed vs forbidden (binding)

- Allowed: docs/notes, offline replay/benchmark scripts + datasets, read-only production SELECTs (server-enforced read-only session + before/after metadata, per M1′/M6′ precedent), laptop-side ESP32 diagnostics/build/upload ONLY if separately asked (VPS never touched).
- Forbidden: changing 140 in production, eligibility, PRELIM/FINAL semantics, Android/dispatch/contracts/`algo_ver`, any deploy, any VPS write, any production DB write.
