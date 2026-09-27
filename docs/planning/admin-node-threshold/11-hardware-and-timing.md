# 11 — Hardware experiments + timing model (plan — NOT executed)

- Purpose: cover task §§9–11 (controlled tests, PRELIM→warning latency, sensor-timing diagnostic) in one executable plan.
- Status: plan; NO hardware touched; ESP32 work (if any) happens from the laptop in a separate task, never VPS.
- Last updated: 2026-09-11 UTC.

## A. Environmental disturbance battery (goal: fill the 20–160+ gal rows)

For each: fixed mount, log raw `correctedMagnitude`/STA/LTA/PGA if serial available (else server PGA + video log timestamp), 3+ repeats, note LTA state/warmup/cooldown. Classes: footsteps (walk/past/room-above), doors (close/slam), furniture (chair drag/drop, table knock), accidental contact (bump enclosure, cable tug), kick/table slam (CONTROLLED, low force first), appliances (washer/fridge/AC on/off, speaker bass), construction/vehicles (if ambient), wind/structure (window open/closed, fan), handling (pick up/rotate/set down), I2C/cable (wiggle — EXPECT faults; observe reset transient), reboot (power cycle → warmup 45 s + first-minute triggers), electrical (charger plug/unplug, nearby relay). Record peak corrected + reported PRELIM/FINAL PGA per class. STOP rule: any class ≥ 100 gal → H-3 falsified, file ledger, do not repeat harder.

## B. Controlled mechanical excitation (characterize, NOT "simulate a quake")

Calibrated taps/shaker at increasing amplitude on the SAME mount; log input vs reported PGA to build the mount transfer curve + clipping check (approach ±4 g only if rig allows — otherwise assert headroom analytically: 160 gal ≈ 0.16 g ≪ 4 g). Never claim equivalence to seismic loading (frequency/duration differ).

## C. Replayed earthquake signals (offline preferred)

Inject public-waveform-derived acceleration time series (PEER/K-NET/USGS candidates per `04-data-plan.md`) through the OFFLINE detector model (port of `sensor.cpp` STA/LTA + baseline, laptop-side, research-only) to map waveform PGA → reported PRELIM/FINAL PGA + detection latency. On-device injection only if a safe harness exists; never present offline-model output as device behavior without on-device confirmation of at least the transfer points.

## D. Real-earthquake validation (opportunistic, honesty-bound)

Any future felt event: preserve serial/sensor logs + server rows + heartbeat continuity + audible-alarm timestamps; file as one population member (never a threshold justification alone, §6). BMKG catalog May be cited as post-event time/place reference only (§2), never as PGA truth for the ESP32.

## Timing model to populate (motion → audible)

`T0 ground motion at sensor → T1 STA/LTA confirm (+300 ms hold, Idris: growth-dependent) → T2 PRELIM publish (loop + MQTT QoS0, retry 5 s if down) → T3 server ingest (OBSERVED 7–14 ms healthy) → T4 UNCONFIRMED+advisory (server share OBSERVED ~28 ms on 2026-09-01尽可能的 SENSOR event) → T5 event duration until detrigger (seconds; ratio<1.5 or 60 s cap) → T6 FINAL publish → T7 edge/transition evaluation (async, ≤2 s store budget) → T8 DispatchTrustedLocal (WS immediate + FCM sendToTokens) → T9 device audible (OBSERVED drill: +467 ms locked foreground, +960 ms FCM-revived; in-use heads-up + siren per D-017; vendor/Doze variance UNKNOWN)`. The 140→100 gain can ONLY shorten T5→T7 (earlier FINAL crossing IF growth curve crosses 100 earlier — unlogged today) — it never shortens T1–T4 or T8–T9. Quantify, don't assume.

## Sensor-timing diagnostic (the UNKNOWN that rescales everything)

Measure TRUE GPIO15 ISR rate + DMP packet rate (logic analyzer or ISR counter over 60 s, laptop-side, no detector change): report mean/jitter/drift vs the 100 Hz assumption in `config.h:100-101` comments; recompute STA/LTA/baseline time constants at the MEASURED rate; if ≠100 Hz, file ledger + mark H-6 falsified and re-derive the latency model. Until measured: every timing claim carries "at assumed 100 Hz". `setRate(99)` + DLPF_BW_5 + FIFO + semaphore path (`sensor.cpp:65-83,308-326`) means the DMP output rate, not the gyro sample rate, is what matters — measure AT the ISR.
