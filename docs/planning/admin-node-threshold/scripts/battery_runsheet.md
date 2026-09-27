# Battery run-sheet — metric D (false-warning) for the 70-gal question

- Purpose: measure the FINAL PGA that ordinary non-earthquake disturbances reach
  on the single bench node, so the empty 100–140 gal / 70-gal row in
  `08-safety-analysis §2` can be filled with real numbers. This is the evidence
  D-036 requires before any floor can be lowered.
- Status: PROCEDURE — owner runs it at the bench. Nothing here changes
  production, firmware, or the floor (still 140). Read-only capture + analysis.
- Last updated: 2026-09-26 UTC.
- Companion: `battery_capture.py` (serial capture), `admin_floor_sweep.go` (analysis).

## The safety key: run WITH WiFi ON, at the current 140 floor

Two firmware facts decide the whole setup:

1. **The FINAL PGA number is printed ONLY on a successful MQTT publish**
   (`mqtt.cpp:183-189`, `Trigger published (FINAL … pga=%.4f …)`). With WiFi OFF
   the publish fails closed and you get `Trigger publish failed!` with **no
   number** (`mqtt.cpp:193`). `pendingReport.maxPga` is never serial-printed
   anywhere else (`sensor.cpp:267`). **So the measurement REQUIRES WiFi ON.**
   (This supersedes the R-016 "run offline" idea, which was written to avoid data
   pollution — a concern lifted by the private-testing framing in R-017.)

2. **The live alarm gate is FINAL ≥ 140 gal** (`AdminNodeMinPGAGal`, D-037). The
   band we are measuring for the 70-gal question is [70, 140) — **entirely below
   the live alarm floor.** So those disturbances are recorded but CANNOT fire a
   real alarm at the current production floor. Measuring live is inherently safe
   *as long as nothing reaches 140.*

Net: run the node exactly as it lives now (WiFi on, designated as-is). You get
exact PGA numbers on serial AND the rows land in the DB so the sweep harness can
analyze them. The only thing to guard against is an accidental ≥140 gal hit — the
STOP rule below keeps a 40-gal margin.

### Confirm before starting
- [ ] Node is the bench unit and is powered / reachable on the prod broker.
- [ ] You are physically at the bench for the whole run (owner-attended).
- [ ] You accept that during the run this node's *real* local-warning ability
      stays ON (that is the safe default — a genuine quake ≥140 still warns).
      If you would rather it CANNOT alarm at all during the high-force classes,
      temporarily revoke its admin designation for those classes and re-designate
      after (tradeoff: no local warning in that short window).

## Capture setup

```
python3 docs/planning/admin-node-threshold/scripts/battery_capture.py /dev/ttyUSB0 battery_2026-09-26.log
```
- 115200 baud, line-buffered, epoch-timestamped (see the script header).
- Opening the port REBOOTS the CH340 (R-013). **Wait 45 s** (warmup) after it
  starts before the first disturbance.
- Note the boot: the log's first `epoch=` line is your window start; you will
  send me that and the end epoch as FROM_TS / TO_TS.

## The battery — realistic classes first, stress bounds last

Rules for every class:
- **3+ repeats**, **≥ 90 s apart** (60 s event cooldown + margin).
- **Low force first, escalate gently.** Announce each out loud so it lands in the
  timestamped log context ("class=door repeat=2 medium").
- After any reboot, wait 45 s before resuming.

Order (the realistic ones answer "does normal life reach 70?"):
1. Footsteps near the mount — walk past, then stomp.
2. Door — open/close on the same wall, then a slam.
3. Furniture — chair drag, table knock, set an object down hard.
4. Accidental contact — bump the table/shelf, tug the USB cable.
5. Vibration source — nearby appliance / bass / fan on the same surface.
6. Handling — pick the node up and set it back down.
7. Cable/fault — wiggle the I2C/USB cable, replug USB.

Stress bounds (NOT realistic false triggers — they establish the ceiling; run
last, gently):
8. Controlled tap on the mount surface, escalating force.
9. Kick / table slam — controlled, lowest force first.

Deferred to the very end: reboot the node (power-cycle) a few times — the reboot
transient class.

### STOP / record rules
- Record **every** disturbance, including ones that trigger nothing.
- FINAL ≥ **70 gal** on any disturbance = a would-be false alarm at a 70 floor.
  Flag it prominently. You may continue (it does NOT alarm at the live 140 floor).
- FINAL ≥ **100 gal** = STOP that class, do NOT repeat it harder (doc 11 STOP
  rule; already well past 70 and nearing the 140 alarm floor).
- Anything approaching ~**130 gal** = STOP the whole battery (margin to the live
  140 alarm floor is gone). Re-assess before continuing.

## What to read on serial (WiFi ON)

Per disturbance, watch for, in order:
- `[STA/LTA] Trigger! ratio=… STA=… gal LTA=… gal` — onset detected.
- `[STA/LTA] Trigger cancelled (transient). ratio=…` — did NOT confirm (stayed
  below the confirm hold). Record as "no event" — a useful sub-threshold datapoint.
- `[STA/LTA] Event confirmed - PGA tracking started.` — event confirmed.
- `Trigger published (PRELIM … pga=X.XXXX …)` — PRELIM peak-so-far.
- `Trigger published (FINAL … pga=Y.YYYY …)` — **FINAL peak. This Y is the
  metric-D number.**
- `Trigger publish failed!` — should NOT appear with WiFi on; if it does, the node
  lost network and the FINAL number for that disturbance is lost — re-run it.

## Recording template

| wall-clock (announced) | class | repeat | force | outcome (trigger/cancelled/confirmed) | PRELIM pga | FINAL pga | notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |  |  |

## When done — send me

1. The capture log (`battery_2026-09-26.log`) — I read the raw PRELIM/FINAL numbers.
2. The window: the `epoch=` start and end lines from the log (→ FROM_TS / TO_TS ms).

I then run the read-only sweep over that window:
```
FROM_TS=<start_ms> TO_TS=<end_ms> ADMIN_NODE_ID=NODE-52960B47 ADMIN_NODE_VERIFIED=true \
DATABASE_URL="postgres://<readonly>@host/quakealert?sslmode=disable" \
go run scripts/admin_floor_sweep.go
```
Each disturbance that confirmed an event becomes an UNCONFIRMED single-node frame;
the harness tabulates whether it would be ELIGIBLE at 70 / 80 / … That table IS
metric D at 70 — the empty 100–140 row in `08-safety §2`, filled with real data.
