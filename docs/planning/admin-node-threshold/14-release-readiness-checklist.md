# 14 — Release-readiness checklist (GUIDELINE ONLY — no cleanup performed)

- Purpose: future pre-launch procedure for cutting a clean production dataset from the current PRIVATE TESTING system. Guideline only; executing any step needs a separate explicit owner order at release time.
- Status: PROPOSED guideline; NOT executed; production stays as-is (private testing).
- Last updated: 2026-09-13 UTC.
- Authority note: RESEARCH planning area; does not override `/contracts`, `PROJECT_RULES.md`, `docs/DECISIONS.md`, `docs/CURRENT_STATE.md` (PROJECT_RULES §5).
- Context (owner-declared 2026-09-13): system is PRIVATE TESTING, not publicly released. Battery observations from NODE-52960B47 (incl. CHECKPOINT 4) are PRIVATE TEST DATA. A clean/reset production dataset ships only at public launch.

## Binding rules for the future cleanup (when ordered)

- Full logical + filesystem backup BEFORE any delete/reset (dump + verify restore on an isolated host first).
- Every destructive step prints affected row counts BEFORE and AFTER; archive the counts + the exact statements in a dated `docs/evidence/release-<date>/` bundle with `SHA256SUMS.txt` (M1′/M6′ precedent).
- No step runs against production while the live server writes: stop the server container first (approved downtime), re-enable after verification.
- `algo_ver` history (V4): never rewrite surviving rows to new semantics; deleted test rows are gone, kept rows keep their labels.

## Checklist (all PENDING until release stage)

### 1. Identify/remove test observations
- [ ] Define the test windows (battery windows per ledger R-017+, drill windows, bench-identity periods) as explicit `[start_ms, end_ms]` ranges + node list.
- [ ] `SELECT` (read-only first): count `sensor_observations` rows inside each window; reconcile against the ledger-recorded battery classes.
- [ ] Delete ONLY rows in those windows (and any rows from bench/test identities); record counts. Note FK/order: `event_state_log` references `earthquake_events`; emissions reference neither (nullable `event_id`) — delete emissions for removed events in the same transaction.
- [ ] Residual risk: QoS-0/ledger-drop rows already missing are UNKNOWN by design — state the residual explicitly, never claim a provably clean cut.

### 2. Clear/reset test event history
- [ ] Delete `event_state_log` rows + `earthquake_events` rows for events whose evidence falls entirely inside test windows (membership check via `evidence_summary.contributors[].node_id` + `decided_at` window, D-015 non-causal rule).
- [ ] Delete `event_near_confirmed` rows for the same events (follower table, migration 000009).
- [ ] Keep genuine felt-event rows (if any) untouched; list kept vs removed event_ids in the evidence bundle.

### 3. Event IDs/sequences
- [ ] `event_id` values are random UUIDs — no sequence to reset; verify no application code assumes contiguity.
- [ ] `BIGSERIAL` sequences (`observation_id`, `emission_id`, log `id`): do NOT reset (gaps are normal after deletes; resetting risks collisions). Record current `last_value` before/after for the bundle.
- [ ] `obs_seq` / boot counters live on the device (NVS), not the DB — nothing to reset server-side.

### 4. Sensor/node identity and credentials
- [ ] Resolve the R-016 identity question: confirm which physical unit is the production unit; exactly one unit holds the production `station_id` + HMAC secret.
- [ ] Any second unit gets a DISTINCT identity via the supported wizard flow (never shared keys — dedup/`obs_seq` collision risk).
- [ ] Verify `iot_nodes` row: correct coordinates (surveyed, not IP-derived), `verified=true`, `is_active=true`.

### 5. Admin Node designation
- [ ] After cleanup, designate the production Admin Node fresh (`admin-designate`), verify exactly-one-holder (partial-unique index), verified + heartbeat-fresh; record designation timestamp (closes the R-009 tension with a clean activation record).
- [ ] Repair the D-036/D-037 governance record (activation verification + D-037 DECISIONS entry) before launch.

### 6. Test notification/audit artifacts
- [ ] `alert_emissions` rows for removed events: delete in the same transaction (they are the audit trail of test alarms).
- [ ] Server log lines are ephemeral (no action); client-side test notifications live on test devices only — confirm test-device set, no public audience ever pushed (verify via emission audiences: only `NONE`/`TOKENS_RADIUS_20KM` to known test tokens).
- [ ] FCM test tokens: prune/rotate any token that should not exist at launch.

### 7. Database integrity after cleanup
- [ ] FK check: no orphan `event_state_log` / emission rows referencing deleted `event_id`s.
- [ ] Constraint/index check: partial-unique admin index intact; `event_id,revision` uniqueness intact.
- [ ] `VACUUM ANALYZE` on touched tables; row counts + `pg_stat_user_tables` before/after in the bundle.
- [ ] Spot replay: run the read-only replay window over the kept dataset; expect same decisions on kept events (D-013).

### 8. Production configuration review
- [ ] Broker ACL/topic scope, TLS, HMAC secrets rotated if bench exposure is a concern; `.env.prod` values verified present-but-unprinted (FIREBASE_SETUP §5 inventory pattern).
- [ ] `EVENT_TRACKER_ENABLED`, correlation/attach/independence/resolve/sweep values recorded; single-node guard states as intended for launch fleet size.
- [ ] Firmware on the production unit: release build, correct broker/identity/NVS, warmup elapsed, heartbeat flowing.

### 9. Final end-to-end smoke test (AFTER cleanup, BEFORE opening)
- [ ] Controlled sub-alarm excitation (below any warning floor): PRELIM→FINAL ingested → `UNCONFIRMED` advisory only → RESOLVED; assert NO push, NO local frame, ledger rows present, then DELETE these smoke rows too (or pre-declare the smoke window as test data and remove per §1).
- [ ] One authorized ≥-floor FINAL check ONLY if owner explicitly orders a live-fire test with test devices in the 20 km set; otherwise skip (never alarm an unconsenting audience).

### 10. Release freeze and post-release monitoring
- [ ] Freeze: code + thresholds + contracts + `algo_ver` pinned; tag release commit; record `algo_ver` + binary hash + schema version in the launch bundle.
- [ ] Monitor week 1: trigger rate, verification-failure rate, `ledger_drops_total`, near-confirmed list (expect honest-empty one-node), heartbeat continuity, FCM outcomes; alert the owner on ANY ≥-floor non-seismic eligible (H-3 watch continues in production).
- [ ] Publish the launch evidence bundle; update `CURRENT_STATE.md` (RELEASED column) only with owner sign-off per PROJECT_RULES §8.

## Explicit non-goals (never part of cleanup)

- Rewriting kept rows to new semantics (V4 violation); resetting BIGSERIAL sequences; deleting evidence bundles or ledger history; resolving U-001…U-009 by implementation.
