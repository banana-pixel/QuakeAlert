# 03 — Hypotheses (falsifiable, preserved when wrong)

- Purpose: state what "100 gal is better / 140 is better" would HAVE to mean, so evidence can kill either.
- Status: proposed; none accepted.
- Last updated: 2026-09-11 UTC.
- Rule: never delete a hypothesis; mark SUPERSEDED with pointer to the ledger entry that killed it.

## H-1 — Conservative-baseline (null)

140 gal has produced no observed environmental false warnings at the current installation (observational, small-n). KEEP 140 unless positive evidence for a lower value meets RQ-CRIT. Falsified by: measured genuine-event miss attributable SOLELY to the 140 floor (not to detection/ingest/heartbeat), OR controlled evidence that 100 gal keeps false-warning risk within the accepted bound with meaningful latency gain.

## H-2 — Earlier-warning

Lowering 140→100 buys ≥ `T_min` seconds (PROPOSED, owner to set — e.g., 0.5–1 s floor for "meaningful") on the class of events that cross both, because PGA growth 100→140 takes measurable shake time. Falsified by: replay showing median gain < T_min, or gain consumed downstream (FCM/Doze/audible path dominates).

## H-3 — Safety-margin

Disturbances at this installation (footsteps, doors, furniture, cable, appliances, wind, handling, I2C/reboot) stay below 100 gal with margin, EXCEPT deliberate impacts/handling which operators can exclude procedurally. Falsified by: ANY logged disturbance ≥ 100 gal in environmental tests (§5/§9 files), or an observed production trigger 100–140 gal of non-seismic origin.

## H-4 — Phase-asymmetry

PRELIM PGA systematically understates FINAL peak (peak-so-far vs whole-event max), so the admin FINAL-only gate already captures the earliest moment the floor CAN be judged — lowering the floor cannot make PRELIM warn (gated by `ADMIN_NOT_FINAL`), only makes eligible FINALs earlier/lower. Falsified by: code showing any PRELIM→local path (none found 2026-09-11), or a future decision moving the gate to PRELIM (out of scope here).

## H-5 — No-architecture-cost

A floor-only change (140→X) needs no contract/Android/dispatch/lifecycle change — only a const + `algo_ver` decision + tests + replay profile. Falsified by: finding that any client/server contract, radius, severity override, or D-003/D-037 text assumes "140" numerically (none found 2026-09-11 beyond display copy — verify before recommendation).

## H-6 — Timing-assumption risk

STA/LTA time constants as commented (~0.5 s / ~20 s @100 Hz) are correct. Falsified by: measured INT/DMP rate ≠ 100 Hz (UNKNOWN today — §11 file), which rescales every constant and the PRELIM→FINAL latency model.
