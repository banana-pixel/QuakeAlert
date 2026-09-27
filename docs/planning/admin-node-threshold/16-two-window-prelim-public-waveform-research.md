# Two-Window PRELIM — Public Waveform Research

- Status: COMPLETE (2026-09-27). RESEARCH ONLY. No firmware/server/Android/MQTT/Admin/threshold/PRELIM-FINAL-semantic changes; nothing deployed; no physical testing; no ESP32 flashing; no production access. Verdict: Two-Window PRELIM question answered at **PRELIMINARY / small-sample** strength — a later single window recovers substantially more of FINAL than the onset-instant PRELIM (median 7 %→49 % by +2 s, B), a second decision window is unsupported over re-timing, and the "useful warning time" half is INSUFFICIENT EVIDENCE offline (needs latency + real-device measurement). Floor stays 140 (D-007).
- Scope: determine whether public earthquake WAVEFORM data can give a defensible empirical basis for a "Two-Window PRELIM" (an added early-warning decision window ~1–2 s after onset) — BEFORE any physical ESP32 testing. The goal is NOT to implement Two-Window PRELIM.
- Evidence labels used throughout: **A** = documented dataset fact · **B** = measurement/calculation performed in this research · **C** = interpretation · **D** = unknown. Numerical claims name their source or say "calculated from downloaded waveform".
- Related: [[15-two-window-prelim-audit]] (pre-execution audit; verdict INSUFFICIENT EVIDENCE), [[12-prelim-early-warning-research]], [[13-evidence-plan-prelim-timing]], RESEARCH_LEDGER R-021.

---

## 1. Research Question

Primary: **Does waiting ~1–2 s after initial detection provide substantially more reliable amplitude information than the current early PRELIM measurement, while still leaving useful warning time?** Do not assume the answer is yes.

Secondary: Is a *second* window actually necessary, or would simply **re-timing/redefining the single PRELIM** be technically simpler and sufficient? Compare conceptually: (A) current early PRELIM · (B) a single later PRELIM · (C) two-stage PRELIM · (D) FINAL-only.

Enabling question: Do authoritative public waveform archives exist that can support an **offline replay** of the QuakeAlert amplitude-growth question, and what can such data legitimately establish vs not?

This report answers the enabling and secondary questions from source + public data, and answers the primary question only to the evidence strength the data actually support (stated explicitly, INSUFFICIENT EVIDENCE where warranted).

---

## 2. QuakeAlert Current PRELIM/FINAL Semantics (verified from source, 2026-09-27)

Pipeline, verified line-by-line in `firmware/src/` (not from comments):

`correctedMagnitude → STA/LTA → trigger → confirmation → PRELIM → running PGA → FINAL`

| Stage | Source | Behaviour (SOURCE FACT) |
| --- | --- | --- |
| correctedMagnitude | `sensor.cpp:155` | `fabsf(vectorMagnitude - baselineEMA)` in gal — scalar accel magnitude with an EMA baseline (DC/gravity) removed. |
| STA / LTA | `sensor.cpp:17-18`, upd `:169-172`, clamp `:181` | STA_ALPHA=1/50 (~0.5 s), LTA_ALPHA=1/2000 (~20 s); LTA frozen while `eventInProgress`; `ratio = sta / ltaClamped` (`:182`). |
| trigger | `sensor.cpp:190-201` | `ratio ≥ 4.0 && sta ≥ 1.5 gal` → `potentialEvent=true`, `potentialEventTime=millis()`. |
| confirmation | `sensor.cpp:203-247` | must stay above threshold for `CONFIRMATION_DURATION_MS = 300` ms; on confirm: `eventInProgress=true`, `eventStartTime=potentialEventTime` (`:211`), **`pga = correctedMagnitude`** (`:214`). |
| obs_seq | `sensor.cpp:220-221` | `(bootCount<<16)|inBootSeq`, minted ONCE at confirmation; shared by PRELIM and FINAL. |
| PRELIM | `sensor.cpp:226-237` → `firmware.ino:300` → `mqtt.cpp:157-170` | slot `pendingPrelim.maxPga = pga`; published `phase="PRELIM"`, `dur_ms ≈ 300`, no `detrigger_ts`. |
| running PGA | `sensor.cpp:250-252` | `pga = max(pga, correctedMagnitude)` every sample while `eventInProgress` — genuine running peak. |
| FINAL | `sensor.cpp:254-286` → `firmware.ino:300,308` → `mqtt.cpp` | detrigger on `ratio<1.5` OR age>`60000` ms; `pendingReport.maxPga = pga` (true peak), full `dur_ms`, `detrigger_ts` set; `phase="FINAL"`. |

Exactly **two** publishes per event (`firmware.ino:300`, `state.h:49-54`), same `obs_seq`. Wire fields (`mqtt.cpp:157-170`): `proto_ver, node_id, phase, obs_seq, attempt_no, pga, dur_ms, onset_ts, detrigger_ts?(FINAL), ts, signature`.

### 2.1 What PRELIM actually is (SOURCE FACT)

PRELIM `pga` is an **onset-instant single sample**: `sensor.cpp:214` sets `pga = correctedMagnitude` (the one sample at the confirmation instant), and `sensor.cpp:227` copies that into `pendingPrelim.maxPga`. During the 0–300 ms confirmation window `pga` is **not updated at all** — the running `max()` at `sensor.cpp:252` only runs *after* confirmation. So PRELIM is **neither** peak-so-far **nor** an average; it is the instantaneous corrected magnitude at the 300 ms confirmation point.

### 2.2 Discrepancies — reported, NOT corrected (per task instruction)

- **DISC-1 (comment vs code):** `sensor.cpp:223-225` comment says *"pga adalah puncak sejauh ini"* ("pga is the peak so far"). This is **false** for PRELIM — line 214 assigns a single sample; no max is taken before line 252. Cross-ref [[15-two-window-prelim-audit]] / R-021, which first flagged this.
- **DISC-2 (field name vs content):** `EventReport.maxPga` (`state.h:63`) is reused for both slots, but in the PRELIM slot it holds a single sample, not a maximum — the name is misleading for PRELIM.
- **DISC-3 (FINAL peak window):** FINAL `maxPga` IS a true running peak (`:252,:267`), but only over `[confirmation .. detrigger]`. The pre-confirmation 0–300 ms samples were never tracked, so even FINAL omits the earliest ~300 ms of motion.

### 2.3 Consequence for this research (INFERENCE, high confidence)

Firmware/server history stores **exactly two amplitude points per event** — an onset-instant PRELIM and a detrigger-peak FINAL. **No intra-event amplitude time series exists** anywhere in QuakeAlert history. Therefore the checkpoint growth curve (T+0.3/0.5/1.0/1.5/2.0/3.0 s) **cannot be reconstructed from QuakeAlert data**; it can come only from (a) public waveform archives (this report) or (b) new firmware instrumentation (separate growth-curve design). This is the entire motivation for the public-waveform route.

---

## 3. Public Data Sources

Environment feasibility (SOURCE FACT / **B**, checked 2026-09-27 from this VPS): FDSN web services are reachable and open — IRIS/EarthScope `service.iris.edu`→`service.earthscope.org` dataselect (redirect chain) and USGS `earthquake.usgs.gov/fdsnws/event`. A full pipeline was **executed successfully** (§ smoke test): obspy 1.5.1 fetched CI.LRL for Ridgecrest 2019, removed instrument response via StationXML, and produced ~184 gal horizontal at 100 Hz.

Catalog (**A** = from official pages surfaced in the source survey; headless = scriptable from a login-less server; ✗-verified claims marked). Full URLs in §18.

| Dataset | Publisher | Waveforms? | Format | Response meta | Sampling | Headless no-login? |
| --- | --- | --- | --- | --- | --- | --- |
| **FDSN IRIS/EarthScope** | EarthScope (NSF SAGE) | yes, raw counts | miniSEED + StationXML | StationXML/RESP | HN* 100/200 Hz | **YES** ← used here |
| FDSN ORFEUS/EIDA | ORFEUS federation | yes, raw counts | miniSEED + StationXML | StationXML | station-dep | **YES** |
| FDSN RESIF/EPOS-France | EPOS-France | yes, raw counts | miniSEED + StationXML | StationXML | station-dep | **YES** |
| RRSM → FDSN (Europe) | ORFEUS DC | yes, raw | miniSEED/StationXML | StationXML | station-dep | **YES** (portal retired 2025-01-14 → plain FDSN) |
| USGS event/ComCat | USGS EHP | **NO** (catalog only) | QuakeML/GeoJSON | n/a | n/a | YES but no waveforms |
| USGS ShakeMap | USGS + networks | **NO** ("model, not measurement") | HDF/grid.xml | n/a | n/a | YES but no waveforms |
| USGS NSMP (historic) | USGS | yes, raw+processed | SMC/COSMOS text | UNKNOWN | ~200 Hz | YES (legacy static archive) |
| CESMD | USGS+CGS | yes, raw+processed | COSMOS/CSMIP ZIP | StationXML (stations svc) | varies (UNKNOWN) | partial (1× email reg, then scriptable) |
| COSMOS VDC | COSMOS | yes | COSMOS text | UNKNOWN | UNKNOWN | **NO** (interactive portal, no API) |
| **PEER NGA-West2** | PEER (UC Berkeley) | yes, processed | PEER AT2/VT2/DT2 + flatfile | removed (inferred) | per-record (200/100 Hz) | **NO** (account, univ/corp email, weekday login, throttle) |
| PEER NGA-East | PEER | yes, processed | PEER format + flatfile | removed (inferred) | UNKNOWN | **NO** (same gated tool) |
| NIED K-NET/KiK-net | NIED (Japan) | yes, raw counts | K-NET ASCII/bin/CSV | none (flat-to-DC) | 100/200 Hz (lit., unconfirmed on-site) | **NO** (login; redistribution prohibited) |
| ESM (Euro-Med) | INGV Milano | yes, raw+processed | miniSEED/SAC + flatfile | provided | varies | partial (1× reg + token) |

**Key correction (A):** "USGS" is **not** itself a waveform API — `earthquake.usgs.gov` serves catalog/ShakeMap only; USGS strong-motion waveforms flow to CESMD and to the contributing networks' FDSN dataselect. So the reproducible, no-registration route for this study is **FDSN dataselect** (IRIS/EarthScope/ORFEUS/RESIF), which is exactly what the pilot uses. PEER NGA (the classic engineering flatfiles) is **not** headless-fetchable — login-gated, university-email-only, throttled — so it is out for an automated re-runnable pipeline, though usable manually.

## 4. Dataset Selection Criteria


A candidate dataset/record is usable for this study only if it has: (1) **3-component acceleration** time series (not just PGA summaries); (2) a **known origin time** (for onset reasoning); (3) **instrument response** available (StationXML/RESP) so counts→physical accel is valid; (4) **sampling ≥100 Hz** (to compare against the consumer ~100 Hz path without upsampling artefacts); (5) **FDSN-fetchable without registration** (so the analysis is reproducible on a headless server and re-runnable); (6) **open license**; (7) records spanning a **range of magnitudes and distances** so growth-vs-M/R can be examined. Records failing (2) or where onset cannot be determined reliably are **excluded, not force-aligned**.

## 5. Selected Datasets

**Selected: FDSN dataselect (IRIS/EarthScope), HN? strong-motion accelerometer channels, CI (Southern California Seismic Network).** Chosen because it is the only route meeting all §4 criteria *and* headless-scriptable with no registration (so the run is reproducible). Records are raw counts; instrument response removed per-record via StationXML → acceleration in gal. Events (FDSN-open, well-documented, all-station distance spread within each event's radius):

| Event | Origin (UTC) | Epicenter | Station search radius |
| --- | --- | --- | --- |
| Ridgecrest M7.1 | 2019-07-06T03:19:53 | 35.770, -117.599 | 1.2° |
| Ridgecrest M6.4 | 2019-07-04T17:33:49 | 35.705, -117.504 | 1.2° |
| La Habra M5.1 | 2014-03-29T04:09:42 | 33.932, -117.917 | 0.8° |

Exact retained/excluded station IDs are written to `scripts/out/growth_records.json` for reproducibility. PEER NGA-West2/East, NIED K-NET, and COSMOS VDC were **excluded from the automated pipeline** (login-gated / no API) but remain available for a manual cross-check if warranted.

**Security note (B):** the source survey encountered injected off-topic text on two fetched pages (a K-NET FAQ contact string and spurious meta-commentary in some fetch summaries). These were treated as untrusted data and **disregarded**; none are reflected as facts in this report.

## 6. Data Processing Method


Two deliberately separate tracks, to keep the non-equivalence (§ Critical Requirement) visible rather than hidden:

- **T1 — reference strong-motion:** fetch miniSEED (FDSN dataselect) + StationXML; `remove_response` to acceleration; convert m/s²→gal (×100); standard PGA = peak of horizontal components (and geometric mean). This is the "textbook" record.
- **T2 — QuakeAlert-analog:** from the same record, resample to ~100 Hz, then apply QuakeAlert's OWN logic on a scalar magnitude: EMA baseline subtraction (analog of `baselineEMA`), STA/LTA with the exact alphas (STA 1/50, LTA 1/2000), trigger `ratio≥4.0 & STA≥1.5 gal`, 300 ms confirmation, then running peak. T2 is what QuakeAlist would *approximately* see if fed this ground motion through an ideal sensor.

Only T2's derived series is used for checkpoint/threshold analysis; T1 is the sanity reference. All per-record processing parameters are logged. The analysis script lives in `scripts/` (offline, read-only, `//research-only`), clearly separated from firmware/server code.

## 7. Event-Onset Alignment

T0 is defined as the **QuakeAlert-analog confirmation instant** on each T2 record (i.e. the same onset definition QuakeAlert uses: trigger + 300 ms hold), NOT the seismological P-pick — otherwise checkpoints would be measured from a different clock than the device uses. Catalog origin time / P arrival are recorded for context and to sanity-check that T0 lands in the early P/S coda, but are not the checkpoint zero. Where T2 never confirms (no STA/LTA trigger) or onset is ambiguous, the record is **excluded** and counted as such — no forced alignment.

## 8. Amplitude/PGA Method

For each retained record: checkpoint amplitude at T0+Δ (Δ ∈ {0.3, 0.5, 1.0, 1.5, 2.0, 3.0 s}) = running peak of the T2 corrected magnitude over [T0, T0+Δ]; **final** = running peak over the full strong-motion window. Derived per record: checkpoint/final ratio; time to reach 25/50/75/90 % of final; time to cross 60/80/100/120/140 gal (**RESEARCH COMPARISON VALUES ONLY** — not safe/optimal/production thresholds, per task); count of records crossing each threshold at each checkpoint; count below-at-early-but-exceed-later; distribution of time-to-threshold; growth vs M/R where metadata supports it. Unit conversion validity (gal) is asserted only for response-removed T1/T2; raw-count paths are excluded from threshold-crossing counts.

### 8.1 Critical methodological requirement — Non-equivalence of public PGA vs QuakeAlert PGA (C/D)

A public strong-motion PGA is **not** the number QuakeAlert would produce. Differences, and whether the offline replay can reproduce each:

| Factor | Public strong-motion record | QuakeAlert | Reproducible offline? |
| --- | --- | --- | --- |
| Sensor | FBA/force-balance accelerometer, low noise, flat response | MPU6050 MEMS ±2 g, higher noise floor | **No** — MEMS noise/response not modelled |
| Sampling | 100–200 sps (HN? = 100) | assumed ~100 Hz (DMP rate **UNPROVEN**, F-10 / R-013 dispute) | Partly — matched to 100 Hz; true device rate UNKNOWN |
| Instrument response | removed via StationXML (physical accel) | none removed; raw counts scaled | Replay removes response (T1/T2); device does not — a real gap |
| DC / gravity | ~0 after response removal | ~1 g (981 gal) DC present, removed by slow EMA | **Partly** — EMA analog applied, but device baseline≈981 vs replay≈0 |
| Orientation | 3-comp rotated ZNE; PGA = peak horizontal / geo-mean | scalar `sqrt(ax²+ay²+az²)` magnitude, EMA-subtracted | Replay uses `sqrt(E²+N²+Z²)` analog — includes vertical, unlike standard horizontal PGA |
| Baseline correction / filtering | standard SM processing (bandpass, baseline) | EMA high-pass only | Approximate |
| STA/LTA, trigger, confirm, running-PGA | not part of the archive | QuakeAlert-specific (this replay applies the exact logic) | **Yes** — the one part that transfers cleanly |
| Warm-up | n/a | 45 s boot warm-up + ~20 s LTA settle | **Adapted** — replay uses pre-event noise as warm-up (`WARMUP_S`), not the boot-relative 45 s |
| Mounting / structural coupling | free-field or structure per station | unknown local mount | **No** |

**Therefore:** the replay legitimately studies **how the QuakeAlert *detection logic* responds to real ground-motion time histories** (growth dynamics, relative checkpoint/final ratios, ordering of threshold crossings). It does **not** predict the absolute gal a specific MPU6050 on a specific mount will report. Absolute-threshold conclusions (is 100 vs 140 gal "right") are therefore **out of scope** and remain governed by physical bench testing (D-007, [[13-evidence-plan-prelim-timing]]).

---

## 9. Checkpoint Analysis — method applied

For each retained T2 record: PRELIM-analog = corrected magnitude at the confirmation instant T0; checkpoint PGA(Δ) = running peak of corrected over [T0, T0+Δ]; FINAL = running peak over [T0, detrigger]. Ratios and threshold-crossing times computed per §8. Aggregated as distributions (median, p25, p75, range) — never single anecdotes. Script: `scripts/replay_growth.py` (faithful to `sensor.cpp`/`config.h` constants).

## 10. Results

Run completed 2026-09-27 (`scripts/replay_growth.py`, obspy 1.5.1, IRIS FDSN). All numbers below are **B** (calculated from downloaded waveforms via the T2 QuakeAlert-analog detector) unless marked. Full per-record data: `scripts/out/growth_records.json`; aggregates: `scripts/out/growth_summary.json`.

### 10.1 Population — retained vs excluded (with bias audit)

- **Retained: 52 records** (records where the T2 detector confirmed). Spread — events: Ridgecrest M7.1 = 14, Ridgecrest M6.4 = 14, La Habra M5.1 = 24. Distance min/median/max = **7.1 / 44.0 / 132.1 km**. Final(T2) gal min/median/max = **4.0 / 27.7 / 243.0** — spanning well below to well above all candidate thresholds.
- **Excluded: 326**, of which **296 = `FDSNNoDataException`** (station returned no waveform for the window — an archive-availability gap, **uncorrelated with amplitude/outcome**, so not a bias source) and **30 = "no confirmation (T2 never triggered)"**.
- **Bias caveat (C, important):** the 30 no-confirmation exclusions ARE outcome-correlated — they are predominantly weak/distant records the detector never fired on. The retained set is therefore the **"confirmed-event" population**, not "all shaking." For *this* question (growth curve given the device has confirmed) that is the correct conditioning population, but the ratio distributions below must **not** be read as representing all ground motion — only motion strong/near enough for the analog detector to confirm. Two source regions (Ridgecrest, La Habra — both S. California); no teleseismic or other-tectonic diversity → sample is **small and regionally narrow**.

### 10.2 Checkpoint / FINAL ratio — how much of FINAL is captured by T0+Δ (running peak)

| Δ after T0 | n | median ratio to FINAL | p25 | p75 | min | max |
| --- | --- | --- | --- | --- | --- | --- |
| onset-instant (current PRELIM analog) | 52 | **0.071** | 0.030 | 0.169 | — | — |
| +0.3 s | 52 | 0.273 | 0.127 | 0.553 | 0.033 | 1.0 |
| +0.5 s | 52 | 0.324 | 0.183 | 0.802 | 0.095 | 1.0 |
| +1.0 s | 52 | 0.418 | 0.235 | 0.923 | 0.105 | 1.0 |
| +1.5 s | 52 | 0.469 | 0.281 | 1.0 | 0.124 | 1.0 |
| +2.0 s | 52 | 0.492 | 0.293 | 1.0 | 0.134 | 1.0 |
| +3.0 s | 52 | 0.708 | 0.319 | 1.0 | 0.159 | 1.0 |

Reading (C): the current **onset-instant PRELIM captures a median ~7 % of FINAL**; a running peak by +2.0 s captures ~49 % (≈7× the onset sample), and +3.0 s ~71 %. The gain is **monotonic** and largest in the first ~1 s. But **even at +3.0 s the median is only 0.71** — no early window captures FINAL, consistent with §12.1 (PGA arrives late).

### 10.3 Time to reach a fraction of FINAL (s from T0)

| Fraction of FINAL | median (s) | p25 | p75 | n |
| --- | --- | --- | --- | --- |
| 25 % | 0.245 | 0.01 | 1.08 | 52 |
| 50 % | 2.415 | 0.185 | 5.112 | 52 |
| 75 % | 3.115 | 0.487 | 9.945 | 52 |
| 90 % | 4.355 | 1.153 | 13.43 | 52 |

Reading (C): median record reaches 50 % of FINAL only at ~2.4 s and 90 % at ~4.4 s (p75 out to 13 s) — the peak develops seconds after confirmation. This is the quantitative basis for "a 1–2 s window sees materially more than the onset instant" AND for "no fixed early window is a reliable FINAL proxy."

## 11. Threshold-Crossing Analysis

Thresholds are **RESEARCH COMPARISON VALUES ONLY** — no threshold is endorsed, and none of this bears on the production floor (stays 140, D-007). "reach by Δ" uses the running-peak checkpoint value; "reach ever" uses FINAL; time-to-cross uses the instantaneous corrected series. All **B**.

| Threshold (gal) | reach ever (FINAL≥thr) | by 0.3 s | 0.5 s | 1.0 s | 1.5 s | 2.0 s | 3.0 s | below at 0.3 s but exceed later | time-to-cross median s (p25/p75, n) |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 60 | 15 | 0 | 1 | 2 | 5 | 5 | 8 | 15 | 2.87 (1.20 / 4.29, 15) |
| 80 | 12 | 0 | 1 | 2 | 2 | 3 | 7 | 12 | 2.83 (1.99 / 4.30, 12) |
| 100 | 8 | 0 | 0 | 1 | 1 | 1 | 4 | 8 | 3.27 (2.49 / 5.07, 8) |
| 120 | 7 | 0 | 0 | 1 | 1 | 1 | 3 | 7 | 4.34 (2.72 / 5.15, 7) |
| 140 | 5 | 0 | 0 | 0 | 0 | 0 | 2 | 5 | 4.35 (2.97 / 5.30, 5) |

Reading (C):
- **Zero records cross ANY candidate threshold at the 0.3 s checkpoint.** Every threshold crossing in this sample happens later, so "below at 0.3 s but exceed later" equals the "reach ever" count for every threshold — an early-instant decision at any of these levels would, in this sample, catch none of them at 0.3 s.
- Crossings accrue with the window: e.g. at 60 gal, 1 of 15 by 0.5 s → 5 by 1.5 s → 8 by 3.0 s. Even by 3.0 s a **minority** of eventual crossings have occurred (8/15 at 60 gal; 2/5 at 140 gal).
- Median time to first cross ranges ~2.9 s (60 gal) to ~4.4 s (140 gal). Higher thresholds are crossed later, as expected if PGA builds into the S/surface train.
- **Sample caution:** only 5–15 records reach any given threshold; these counts are illustrative of dynamics, **not** rates or probabilities, and rest on the T2 analog (§8.1 non-equivalence), so no absolute-gal inference is drawn.

## 12. Two-Window PRELIM Analysis

### 12.1 What the EEW literature already establishes (A/C — from the §18 source survey)

The seismological-EEW literature bears directly on "does a ~1–2 s window give substantially more reliable amplitude information":

- **PGA arrives late, in the S/surface-wave train** — not in the first P-wave cycles. The P-wave is low-amplitude and higher-frequency; peak ground acceleration almost always occurs seconds after onset with the S and surface waves. So *any* early window (0.3 s or 2 s) systematically **under-observes** the eventual peak.
- **Early amplitude is probabilistic, not deterministic, for the final size.** Real-time EEW estimators use the first P seconds (Allen & Kanamori 2003 τ_p; Wu & Kanamori 2005 τ_c and Pd) and are genuinely useful, but the "universal onset" results (Rydelek & Horiuchi 2006; Meier, Heaton & Ampuero 2016; Meier et al. 2017; Melgar & Hayes 2019) find that small and large ruptures begin essentially identically — magnitude-diagnostic information emerges only after a substantial fraction of the rupture (~10 s for great events). USGS ShakeAlert's EPIC uses ~0.5–4 s of P data and updates continuously precisely because a single early window is not final.

**Implication for QuakeAlert (C):** re-timing/adding a window changes *how much of the developing S-train the device has integrated when it decides* — a real, monotonic gain in how representative the sample is of FINAL — but the literature says **no early window deterministically predicts FINAL PGA**. This reframes the local question from "predict FINAL" to "capture more of the already-arrived motion before deciding," which is a within-record running-peak question the pilot (§10) measures directly.

### 12.2 Pilot evidence (QuakeAlert-analog replay)

Direct answer to the primary question, from the §10 distributions (all **B**, on the T2 analog):

1. **Does a ~1–2 s window give substantially more amplitude information than the current early PRELIM?** In this sample, **yes, and by a large margin**: the onset-instant PRELIM analog holds a median **7 %** of FINAL, versus **~42 % at +1.0 s** and **~49 % at +2.0 s** — roughly a 6–7× increase in how much of the eventual peak has arrived. The onset instant is, on this evidence, a **poor proxy** for FINAL (median 0.071, p75 0.169). (**C**, PRELIMINARY.)
2. **"…while still leaving useful warning time?"** The FINAL-relevant motion is *still largely in the future* at +2 s — median time-to-50 %-of-FINAL is 2.4 s and to-90 % is 4.4 s (§10.3), and threshold crossings cluster at 2.9–4.4 s (§11). So waiting 1–2 s to *sample* does not "miss the peak"; it samples while the peak is still building. The cost is 1–2 s of consumed lead time, and whether that remaining lead time is **useful** depends on device→user end-to-end latency, which is **D / §14 UNKNOWN** and NOT establishable from waveforms. So this half of the question is **INSUFFICIENT EVIDENCE offline.**
3. **Is a SECOND window necessary, or is re-timing the single PRELIM enough?** The marginal information in the very-early sample is small (7 % of FINAL), so most of the gain a two-stage scheme could offer is already captured by simply **moving the single PRELIM later (option B)**. The data give **no** indication that a *second decision* adds value beyond a single re-timed window — and option C carries the [[15-two-window-prelim-audit]] NO-GO cost (new phase/contract/server/Android). (**C**, PRELIMINARY: this pilot cannot enumerate a case where a first + second decision beats one well-placed decision; absence of such a case here is weak evidence, not proof.)

**Evidence strength for the Two-Window question (stated explicitly, per task):** **PRELIMINARY / small-sample.** Supporting: consistent, monotonic, large checkpoint→FINAL gain across 52 records and 3 events, coherent with independent EEW literature (§12.1). Limiting: n=52 from 2 S. California source regions; the T2 analog is NOT the MPU6050 (§8.1 non-equivalence — absolute gal and true device response unverified); the retained set is conditioned on T2-confirmation (§10.1 bias); the warning-time-usefulness half turns on latency the archive cannot supply. **Remaining unknowns before any design:** real-device growth curve on the MPU6050; end-to-end latency budget; false-positive impact of a later decision; whether the extra 1–2 s is acceptable UX. These are the §14 / physical-bench items.

## 13. Comparison: Current PRELIM vs Later PRELIM vs Two-Window (conceptual + evidence)

| Option | What it is | Complexity | Empirical question it hinges on |
| --- | --- | --- | --- |
| A. Current early PRELIM | onset-instant sample at 300 ms (`sensor.cpp:214,227`) | shipped | Is the onset instant a poor proxy for FINAL? (checkpoint/final ratio at 0.3 s) |
| B. Single later PRELIM | re-time the existing single PRELIM to ~1–2 s | **low** — one timing change, no new slot/phase/contract | Does 1–2 s recover most of FINAL while keeping warning time? |
| C. Two-stage PRELIM | keep early PRELIM + add a second decision window | **high** — new phase semantic, contract, server/Android handling ([[15-two-window-prelim-audit]] NO-GO) | Does a SECOND decision add value beyond B? |
| D. FINAL-only | drop early warning, act on detrigger peak | shipped path (classify uses FINAL) | Is any early window worth it vs waiting? |

Key architectural point (INFERENCE, from [[15-two-window-prelim-audit]]): a "window" is a *sample time* + a *decision threshold*; both are single scalars a single re-timed PRELIM (B) already exposes. C is only genuinely new if the SECOND window drives a distinct DECISION. The data below inform B-vs-A and whether C is warranted.

---

## 14. Limitations — what public-waveform research CANNOT establish for QuakeAlert

Public waveforms cannot establish (unchanged by any amount of archive analysis): MPU6050-specific response/noise; exact ESP32/DMP sample rate (still UNKNOWN, F-10); actual detector behavior under physical vibration; false-positive behavior in the deployment environment; local mounting/structural coupling; end-to-end warning latency; multi-node consensus behavior; Admin Node behavior; user-notification latency. These require the physical bench batteries ([[13-evidence-plan-prelim-timing]], [[11-hardware-and-timing]]) and are **not filled with assumptions here**.

## 15. What the Research Supports

1. **An offline, reproducible, headless replay of the QuakeAlert growth question is feasible** (A/B) — FDSN dataselect + StationXML response removal + the exact-constant T2 detector runs end-to-end from this VPS; 52 records processed across 3 events. This answers the *enabling* question definitively.
2. **The current onset-instant PRELIM is a weak proxy for FINAL** on real ground motion run through QuakeAlert's own logic — median ~7 % of FINAL captured (B, §10.2). (Independently, this confirms the [[15-two-window-prelim-audit]] SOURCE FACT that PRELIM is an onset sample, not peak-so-far — DISC-1.)
3. **A later single window recovers substantially more of FINAL** — monotonic gain to ~42 % (1 s) / ~49 % (2 s) / ~71 % (3 s) medians (B). The bulk of the gain is in the first ~1 s.
4. **Re-timing the single PRELIM (option B) is the simpler lever and captures most of the available gain**; the evidence gives no support for a *second decision window* (option C) beyond it (C, PRELIMINARY) — favouring the low-complexity path over the NO-GO two-stage architecture.
5. **The direction is coherent with the independent EEW literature** (§12.1): early amplitude is informative but late-arriving and non-deterministic for FINAL.

All of the above are stated at **PRELIMINARY** strength and as *dynamics/relative* findings, never absolute-gal or production claims.

## 16. What the Research Does NOT Support

1. **No production threshold change** from public-waveform data alone (pre-committed; the 60–140 values are comparison-only; floor stays 140, D-007).
2. **No absolute-gal claim about the MPU6050** — the T2 analog is not the device (§8.1 non-equivalence: MEMS response/noise, true sample rate F-10, mounting, no response removal on-device). The ratios transfer; the absolute numbers do not.
3. **No conclusion that 1–2 s "leaves useful warning time"** — that depends on end-to-end device→user latency, which waveforms cannot supply (§14, D). This half of the primary question is **INSUFFICIENT EVIDENCE** offline.
4. **No probability/rate claims** — threshold-crossing counts (5–15 records) are illustrative dynamics, not the likelihood a real deployment crosses a level.
5. **No generalization beyond the sampled regime** — 52 records, 2 S. California source regions, confirmed-event-conditioned (§10.1 bias); not teleseismic, not other tectonic settings, not the local deployment site.
6. **No GO for implementing Two-Window PRELIM** — this report does not overturn the [[15-two-window-prelim-audit]] Decision Gate; it informs a *next experiment*, not a build.

## 17. Recommended Next Experiment

The pilot condition is met (a later window materially beats the onset-instant PRELIM in replay, §10/§12.2), so the recommended next step is the one that closes the **non-equivalence gap** rather than any code change: **firmware growth-curve instrumentation** — the separate serial `#ifdef` design ([[13-evidence-plan-prelim-timing]] E4, default OFF) that logs the SAME checkpoints (T0+0.3/0.5/1.0/1.5/2.0/3.0 s running peak) on the real MPU6050 for a handful of controlled physical shakes. That would test whether the replay's *relative* growth dynamics hold on the actual sensor/mount before a single PRELIM re-timing (option B) is even designed. It stays behind D-007 and the [[15-two-window-prelim-audit]] Decision Gate; it is a measurement proposal, not an implementation, and nothing here authorizes it — owner approval + physical bench required. The end-to-end latency budget (§14/§16.3 UNKNOWN) must be measured in parallel, since it governs whether the recovered amplitude is worth the consumed lead time.

## 18. Evidence and Sources

**Category-B artifacts (this session, reproducible):** `scripts/smoke_fdsn.py` (feasibility), `scripts/replay_growth.py` (T2 engine), `scripts/out/growth_summary.json` + `scripts/out/growth_records.json` (results incl. exact retained/excluded station IDs). Toolchain: python 3.11 venv `/home/ubuntu/.venv-waveform`, obspy 1.5.1, numpy 2.4.6, scipy 1.17.1.

**Firmware source (SOURCE FACT, lines cited inline in §2):** `firmware/src/sensor.cpp`, `config.h`, `state.h`, `mqtt.cpp`, `firmware.ino`.

**Data services (A — primary, verified reachable/open from this host 2026-09-27):**
- FDSN web-services spec — https://www.fdsn.org/webservices/
- IRIS/EarthScope FDSN dataselect + station — https://service.iris.edu/ (redirects to https://service.earthscope.org/)
- USGS FDSN event / ComCat (catalog only, NOT waveforms) — https://earthquake.usgs.gov/fdsnws/event/1/
- SCEDC / CI network (Southern California) — https://scedc.caltech.edu/
- ObsPy (fetch + response removal + resample) — https://docs.obspy.org/

**Other archives surveyed (A — not used in the automated pipeline; §3 catalog):**
- ORFEUS / EIDA — https://www.orfeus-eu.org/ · RESIF/EPOS-France — https://www.resif.fr/ · ESM (Euro-Med) — https://esm-db.eu/
- USGS CESMD — https://www.strongmotioncenter.org/ · COSMOS VDC — https://www.strongmotioncenter.org/vdc/ (interactive, no API)
- PEER NGA-West2/East — https://ngawest2.berkeley.edu/ (login-gated, not headless)
- NIED K-NET/KiK-net — https://www.kyoshin.bosai.go.jp/ (login-gated; redistribution restricted)

**Events (A — catalog, cross-check via USGS ComCat):** Ridgecrest M7.1 2019-07-06T03:19:53 (35.770,-117.599); Ridgecrest M6.4 2019-07-04T17:33:49 (35.705,-117.504); La Habra M5.1 2014-03-29T04:09:42 (33.932,-117.917).

**EEW literature (A — §12.1 primary references, cited from the source survey; consult originals before quoting):** Allen & Kanamori 2003 (*Science* 300, τ_p EEW); Wu & Kanamori 2005 (τ_c / Pd); Rydelek & Horiuchi 2006 (*Nature* — onset determinism); Meier, Heaton & Ampuero 2016 and Meier et al. 2017 (*JGR/BSSA* — universal rupture onset); Melgar & Hayes 2019; USGS ShakeAlert / EPIC algorithm documentation (https://www.shakealert.org/). These support the *direction* (early amplitude informative but late-arriving, non-deterministic for FINAL); they are not QuakeAlert-specific.

**Security note (B):** two injected off-topic text fragments encountered while surveying sources (a K-NET FAQ contact string and spurious meta-commentary in fetch summaries) were treated as untrusted data and disregarded; none appear as facts above (see §5).

---
