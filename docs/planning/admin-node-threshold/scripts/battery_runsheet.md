# Admin Node basic bench test — V1 detector sanity + PRELIM behaviour

- Purpose: basic sanity of the flashed V1 firmware on the bench NODE-52960B47:
  (1) the detector triggers and publishes on real motion, (2) PRELIM carries
  the running peak over the first 1.0 s after confirmation (D-039, not the old
  onset-instant sample), (3) PRELIM and FINAL share one `obs_seq` as exactly
  two publishes per event. This is NOT an exhaustive false-warning
  characterization — the metric-D disturbance battery is deferred.
- Status: PROCEDURE — owner runs it at the bench. Nothing here changes
  production, firmware, or any floor. Read-only capture + analysis.
- Last updated: 2026-09-27 UTC (replaces the metric-D battery run-sheet; see
  RESEARCH_LEDGER R-031 for why).
- Companion: `battery_capture.py` (serial capture), `admin_floor_sweep.go`
  (read-only eligibility analysis over the window).
- Firmware under test: merged V1 + INT_RATE build (R-029), flashed 2026-09-27
  (R-030). `INT_RATE ≈100 Hz` lines confirm the ISR path is alive.

## Bench setup requirements

- Install the Admin Node in a **bright, stable location** free from avoidable
  physical/vibration interference: solid surface (no wobbly shelf), away from
  foot traffic, doors, appliances, speakers, and fans. Bright so the run is
  visually observable and each stimulus can be timestamped unambiguously.
- Remove or switch off avoidable vibration sources for the duration of the run.
- Node runs exactly as it lives (WiFi ON — the FINAL PGA number prints ONLY
  on a successful MQTT publish; WiFi OFF loses it).

### Confirm before starting
- [ ] Location is bright, stable, and cleared of avoidable interference.
- [ ] Node is the bench unit, powered, reachable on the prod broker.
- [ ] You are physically at the bench for the whole run (owner-attended).
- [ ] Serial capture running (below) with 45 s warmup observed after the
  port-open reboot.

## Capture setup

```
python3 docs/planning/admin-node-threshold/scripts/battery_capture.py /dev/ttyUSB0 battery_basic_2026-09-27.log
```
- 115200 baud, line-buffered, epoch-timestamped.
- Opening the port REBOOTS the node — **wait 45 s** before the first stimulus.
- The log's first `epoch=` line is the window start; send start + end epochs
  as FROM_TS / TO_TS afterwards.

## The three stimuli (in order, gentle first)

Rules for every stimulus: **3 repeats**, **≥ 90 s apart** (60 s event cooldown
+ margin). Announce each out loud for the log ("stimulus=light-vib repeat=2").
After any reboot, wait 45 s before resuming.

1. **Light vibration** — tap the mount surface lightly with a fingertip, just
   enough to be felt. Expect: trigger, sometimes confirmation.
2. **Moderate controlled vibration** — firmer tap / rap on the mount surface
   with knuckles, clearly felt but never a slam. Expect: confirmation +
   PRELIM + FINAL publishes.
3. **Light handling** — lift the node enclosure ~1 cm and set it gently back
   down, or rotate it slightly in place. Expect: confirmation + publishes.
   Keep it gentle — handling easily overshoots.

### STOP rules (unchanged thresholds)
- Record **every** stimulus, including ones that trigger nothing.
- FINAL ≥ **70 gal** on any stimulus = flag prominently (would alarm at the
  V1 60 floor once the V1 server binary is deployed; cannot alarm at the
  current live 140 production gate).
- FINAL ≥ **100 gal** = STOP that stimulus class, do NOT repeat it harder.
- Anything approaching ~**130 gal** = STOP the whole test. Re-assess before
  continuing.

## What to validate on serial (per stimulus, in order)

- `[STA/LTA] Trigger! …` → `Event confirmed - PGA tracking started.`
- `Trigger published (PRELIM … pga=X … obs_seq=N …)` — X should reflect
  roughly the first second of shaking (running peak), NOT a near-zero onset
  sample. Note X and the publish timestamp vs the confirm timestamp
  (≈ 1.0 s later confirms the D-039 window).
- `Trigger published (FINAL … pga=Y … obs_seq=N …)` — Y is the whole-event
  peak; **N must equal the PRELIM's `obs_seq`** (exactly two publishes, one
  episode). Y ≥ X is expected; Y ≈ X means the peak fell inside the first
  second.
- `INT_RATE … INT_HZ=…` lines continue ≈100 Hz throughout (ISR path alive).
- `Trigger publish failed!` — should NOT appear with WiFi on; if it does,
  the FINAL number for that stimulus is lost — re-run it.

## Recording template

| wall-clock (announced) | stimulus | repeat | outcome (trigger/cancelled/confirmed) | PRELIM pga | PRELIM delay vs confirm | FINAL pga | obs_seq shared? | INT_RATE ok? | notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |  |  |

## When done — send me

1. The capture log — I read the raw PRELIM/FINAL numbers, timing, and
   `obs_seq` sharing.
2. The window: the `epoch=` start and end lines (→ FROM_TS / TO_TS ms).

I then run the read-only sweep over that window:
```
FROM_TS=<start_ms> TO_TS=<end_ms> ADMIN_NODE_ID=NODE-52960B47 ADMIN_NODE_VERIFIED=true \
DATABASE_URL="postgres://<readonly>@host/quakealert?sslmode=disable" \
go run scripts/admin_floor_sweep.go
```
Note: production still runs the pre-V1 140 binary, so sweep eligibility is
evaluated against the live 140 gate until the V1 server is deployed — state
this alongside any 60-floor comparison.
