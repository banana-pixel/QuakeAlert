# 05 — Experiment plan: threshold benchmark matrix

- Purpose: how to compare candidates fairly — detection AND false-warning AND latency AND robustness, never detection alone.
- Status: plan; no benchmark executed.
- Last updated: 2026-09-11 UTC.
- Objective function (binding): MINIMUM useful warning latency SUBJECT TO acceptable false-warning risk AND system simplicity.

## 1. Candidates

60, 80, 100, 120, 140 (baseline/control), 160 gal. Add more ONLY if evidence suggests (e.g., 100–120 fine steps if the knee lands there; 200+ only to map the severe-override interaction — trusted-local has NO severe override, so high floors test nothing new; record why if added).

## 2. Per-candidate metrics (all six, every candidate)

- A. detection rate: fraction of eligible-class seismic events (FINAL, admin-contributed, UNCONFIRMED, fresh heartbeat) with admin PGA ≥ floor. Denominator = that class, NOT all ledger rows (S5 honesty).
- B. missed-event rate: 1 − A on the same denominator + list of still-missed events with reason (below floor vs no FINAL vs stale heartbeat vs no event).
- C. warning latency: PRELIM→FINAL gap, onset→eligibility, eligibility→dispatch, dispatch→audible (locked Doze vs in-use heads-up). Report distributions (n, p50/p95 with n stated; n<20 → report max + n, never a "p95 claim").
- D. false-warning rate: environmental/anomaly triggers ≥ floor per unit time + per disturbance class; requires the §5/§9 disturbance battery — without it D is UNKNOWN, not zero.
- E. mounting/environment sensitivity: same excitation across mounts/sites; pass = floor decision stable, fail = flips.
- F. sensor-anomaly robustness: I2C faults, FIFO overflow + STA/LTA reset, reboot/warmup/cooldown, QoS0 loss (PRELIM-lost), retry/duplicate, clock-skew/negative age — does the floor invite a new spurious eligible?

## 3. Method (offline only)

- Copy `EvaluateAdminNodeEligibility` into a research-only harness with the floor as a parameter (production const untouched). Feed: (i) historical snapshots at transition + edge time, (ii) synthetic PGA sweeps, (iii) hardware-test logs. Vary ONLY the float; gate order/vocab/phase/heartbeat/radius fixed.
- Read-only production extraction under server-enforced `default_transaction_read_only=on` + before/after `pg_stat_user_tables` + metadata (M1′/M6′ precedent). Never write.
- No population RATE claim until n supports it; until then report counts + exact cases (S9, §8 honesty).

## 4. Anti-gaming rules

- Never tune the floor to make a phase exit pass (§6). Never average SENSOR with PUBLISH_BOUND onsets. Never merge drill with real. Never cite simulation as field. Never rewrite history rows to the new floor (V4) — replay RE-EVALUATES, never rewrites.
