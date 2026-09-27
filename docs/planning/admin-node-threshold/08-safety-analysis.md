# 08 — Safety analysis (per candidate + governance)

- Purpose: what each floor newly permits / newly catches / still misses, and what governance it touches. Research only — changes nothing.
- Status: analysis from code + LIVE-EQ audit; disturbance mapping UNKNOWN pending hardware battery.
- Last updated: 2026-09-11 UTC.

## 1. What "100 gal" physically means (summary; full pipeline in `01-current-state.md` §5 + finding R-004)

Detector-derived corrected peak (`|vector−baselineEMA|`, STA/LTA-gated), NOT raw acceleration, NOT seismometer PGA. PRELIM = peak-so-far at confirmation (~300 ms post-trigger, typically small — e.g., 1.13 gal on the event that later hit 288.36); FINAL = whole-event max. The admin gate reads the admin contribution's ratcheted peak (effectively the FINAL value at edge time). Lowering the floor CANNOT produce PRELIM warnings (`ADMIN_NOT_FINAL` stands) — it only admits smaller FINALs and (unlogged-intra-event-growth permitting) possibly earlier edge timing. Do NOT treat 100 gal as "MMI VI shaking" without the MEMS-vs-seismometer caveat.

## 2. False-warning modes by band (installation-specific; OBSERVED / INFERRED / UNKNOWN separated)

| Band | OBSERVED (production) | INFERRED (needs test) | UNKNOWN (must measure) |
| --- | --- | --- | --- |
| 20–60 | below-floor ledger rows exist (e.g., 25 below MinPGAGal 16.6 in M1′ window; 4.94 gal rows during M1′ run) — normal path handles them as DETECTED/advisory | footsteps/doors/appliances/wind likely live here for a fixed install | exact per-source gal at THIS mount; LTA-adapted floor drift |
| 60–100 | 77.89 gal event (4fcc3374, seismic-class per M6, no local frame at 140) — proves the band is REACHABLE by real triggers | furniture impacts, cable yank, accidental knock plausibly reach here | whether any reach ≥80/100 without quake |
| 100–140 | NONE observed (no production trigger 100–140 found to date — small-n) | deliberate kick/table slam, sensor handling, mounting resonance | THE decisive unknown: controlled battery (§9 file) must populate this row |
| 140–160+ | 288.36 gal FINAL (seismic-class, ADMIN_ELIGIBLE, live dispatch) | strong local quake; severe device fault (least likely AND most dangerous — single-sensor cannot distinguish, hence heartbeat + FINAL + designated gates) | I2C/FIFO-fault PGA shape (reset path zeroes STA/LTA — transient on recovery unmeasured) |

Rule: no disturbance is claimed impossible without evidence. An empty OBSERVED cell is not safety.

## 3. What each candidate newly catches / still misses (seismic side)

- 60: catches 4fcc3374-class (77.89); still misses deep-distant M5.9-class (no trigger at ANY floor — detection, not policy, was the limiter, LIVE-EQ audit).
- 80/100/120: catch progressively smaller FINALs; all still miss sub-firmware-floor motion (<1.5 gal STA) and stale-heartbeat windows regardless of floor.
- 160: strictly fewer warnings than 140; no safety upside found (trusted-local has no severe path to protect).
- Fundamental ceiling (S2): one-node fleet, 20 km token radius, indoor audibility — floor tuning moves the ELIGIBILITY instant, never the geometry or the FCM/Doze tail.

## 4. Interaction checklist

- FINAL semantics: floor is evaluated on FINAL-phase contribution; D-037 edge (exactly-once, no revision) is the delivery vehicle for already-UNCONFIRMED events. Lower floor widens WHICH edges fire, never HOW (mechanism unchanged). No D-003 text change needed for a floor-only move; any PRELIM-gating change WOULD need one (out of scope).
- D-036: design adopted at 140; new value needs a NEW accepted decision (this research is its input, not its verdict).
- D-007: floor stays compile-time (research harness parametrizes a COPY only).
- D-006/V3/V6/V7 + `algo_ver`: OPEN — `phase3-1.1/ic=5` does not encode the admin floor; decision must state bump / admin-version / comparability rationale. Replay groups by whatever label is decided (V5).
- D-018/D-019: unaffected (validity + audit apply per frame at any floor). D-009/CONFIRMED path: unaffected (separate gate, separate radius, separate FCM path). Android: NO code change for floor-only (gate is distance-only; copy mentions no number — verify strings before recommendation per H-5).
- S4 (false positives kill the channel) vs S5 (misses leave no artifact): the asymmetry is WHY the denominator (pre-confirmation/near-confirmation records, §S5) must be preserved and why D (false-warning rate) gates any lowering.
