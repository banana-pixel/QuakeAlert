# LIVE-EQ-2026-09-12-001 — Read-only production audit

BMKG M5.9 deep event vs QuakeAlert production (VPS 100.91.154.117).

- STOP CONDITION observed: forensic investigation only. No production
  modification, no restart, no deploy, no migration, no threshold change,
  no ESP32 contact. All VPS access was read-only:
  `docker ps / inspect / images / logs`, `git rev-parse / status / log`,
  `SELECT`-only `psql` via `docker exec`, public `GET /healthz`,
  `date / timedatectl` (read-only).
- One inadvertent read exposed secret values in a `docker inspect` Env dump
  on the local terminal. They are NOT recorded here and must not be copied
  from shell history into any file.

## 1. Audit identity

- Audit ID: LIVE-EQ-2026-09-12-001
- Audit date/time (actual reads): 2026-09-11 21:42–21:46 UTC
  = 2026-09-12 04:42–04:46 WIB
- Auditor method: SSH as `opc` over Tailscale to 100.91.154.117 + public
  `https://api.quakealert.web.id/healthz`. No writes issued.

## 2. BMKG earthquake metadata (as reported in task)

- Time: 2026-09-12 04:23:56 WIB = 2026-09-11 21:23:56 UTC
  = 1789161836000 ms epoch UTC
- Magnitude: M5.9
- Location: 6 km southeast of Kepulauan Seribu, DKI Jakarta
- Depth: 376 km
- BMKG event ID: 20260912042631

## 3. Audit window

- Required: 2026-09-12 04:00:00 WIB (= 2026-09-11 21:00:00 UTC,
  1789160400000) to current time (~21:42 UTC, 1789162978000).
- Searched: 21:00–21:45 UTC via `docker logs --since/--until` and
  `SELECT ... WHERE received_ts/decided_at BETWEEN 1789160400000 AND 1789164000000`.
- Widened: 20:00–22:00 UTC (03:00–05:00 WIB, 1789156800000–1789164000000)
  across `sensor_observations`, `event_state_log`, `alert_emissions`.
  Result in all three tables: **0 rows**.
- Delayed-ingest check: no observation with `received_ts >= 1789160400000`
  exists at all (`COUNT(*)=0`); last stored observation is id 91 at
  `received_ts` 1789130407473 (12:40:07 UTC). No late-arriving row with
  `onset_ts` near 21:23:56 UTC exists.

## 4. Production state (STEP 1, read-only)

- Running commit (`git -C /opt/quakealert rev-parse HEAD`): `8e93c4d96b24ef9ec4c39197a781db665894b95f`
  = local HEAD. Message: `feat(d-037): trusted-local FINAL-edge exception`.
  `git status --short`: only untracked `deploy/.env.prod.bak-*`,
  `server/cmd/canary_main.go`, `server/internal/decrypttmp.go` (not modified
  tracked files for deploy state). `git log --oneline -5` head matches local.
- Container image (`docker inspect` / `docker images`):
  `quakealert-server:prod`, image ID `d73cc4758dbf`
  (full `sha256:d73cc47...`), built 2026-09-10 00:45:54 UTC,
  `State.StartedAt` 2026-09-10T00:46:07Z. `docker ps`: Up 45 hours.
- Deployment timestamp: ~2026-09-10 00:46 UTC (= 07:46 WIB).
- Database schema (`SELECT version, dirty FROM schema_migrations`): `10|f`.
  Migration 000010 (Admin Node) applied.
- `algo_ver` (`SELECT algo_ver, COUNT(*) FROM earthquake_events GROUP BY`):
  `phase3-1.1/ic=5`: 44 rows; NULL (pre-Phase-3): 6 rows.
- Active Admin Node (`SELECT station_id, verified, is_admin_node,
  last_heartbeat, NOW()-last_heartbeat`): sole holder
  `NODE-52960B47`, `verified=t`, `is_admin_node=t`,
  `last_heartbeat` 2026-09-11 21:42:37 UTC, age 21.6 s at read 21:42:58 UTC.
  `SELECT COUNT(*) WHERE is_admin_node` = 1. Other 5 nodes verified but
  `is_admin_node=f`, heartbeats 16–17 days stale.
- Server timezone: host `timedatectl`: `Time zone: GMT`, `date`/`date -u`
  identical; container Env `TZ=UTC`; container logs in `Z`. Effective: UTC.
- Current server time: DB `NOW()` 2026-09-11 21:42:58 UTC;
  host `date` 21:46:45 UTC at later read. Consistent.
- Container status (`docker ps`): `quakealert-server` Up 45h;
  `quakealert-caddy` Up 2 weeks; `quakealert-mosquitto` Up 2 weeks (healthy);
  `quakealert-postgis` Up 2 weeks (healthy); `quakealert-redis` Up 2 weeks
  (healthy).
- Service health (`GET https://api.quakealert.web.id/healthz`):
  `{"status":"ok","database":"ok","mqtt":"ok"}`.
- Tracker config observed (read-only Env keys, values only for non-secrets):
  `EVENT_TRACKER_ENABLED=true`, `CORRELATION_WINDOW_MS=20000`,
  `ATTACH_RADIUS_KM=50`, `INDEPENDENCE_CELL_KM=5`,
  `MIN_INDEPENDENT_CELLS=2`, `TZ=UTC`. Secret values seen in the same dump
  are deliberately omitted here.
- Version correspondence: running commit == expected D-037 head; schema 10;
  image built 2026-09-10. Production matches expected deployed version.

## 5. Earthquake event evidence (STEP 2)

Searched production logs + ledger for PRELIM / FINAL / EARTHQUAKE_ALERT /
EARTHQUAKE_ADVISORY / EVENT_RESOLVED / event_id / obs_seq / PGA / state
transitions / dispatch / FCM / WebSocket / audience / ledger entries, in
21:00–21:45 UTC and widened 20:00–22:00 UTC:

- `sensor_observations` in window: 0 rows (any node, any PGA — table stores
  below-floor triggers too, so 0 means no trigger reached the server).
- `event_state_log` in window: 0 rows.
- `alert_emissions` in window: 0 rows.
- `docker logs quakealert-server --since 21:00 --until 21:45`: only periodic
  `event: counters` + `ledger: counters` every 5 min, byte-stable:
  `event_created_total=15`, `UNCONFIRMED=12`, `RESOLVED=12`,
  `ledger_rows_written_total=108`, all drop/failure counters 0.
  `grep -c "trigger diterima"` 20:00–21:45Z = 0;
  `grep -c "trusted-local"` = 0.
- Control: same log vocabulary DOES appear for the prior event —
  12:29:35Z `trigger diterima ... pga 19.55`, 12:40:02Z PRELIM 1.13 gal,
  12:40:07Z FINAL 288.36 gal + `trusted-local emitted ... ADMIN_ELIGIBLE`.
  Absence in the audit window is therefore meaningful, not a logging gap.
- Latest durable rows (all pre-window, for reference):
  - observations 90 (PRELIM, obs_seq 2293766, 1.1329 gal, 12:40:02Z) + 91
    (FINAL, same obs_seq, 288.3640 gal, 12:40:07Z, SENSOR onset, attempt 1).
  - event `004c5f65-1f09-421d-93fc-b3afd3b9b2ee`, RESOLVED rev 2,
    max 288.3640 gal, 1 node / 1 cell, `phase3-1.1/ic=5`.
  - state log ids 87–88 (FLOOR_MET then NO_NEW_EVIDENCE).
  - emissions 94 (ADVISORY), 95 (ALERT, audience TOKENS_RADIUS_20KM,
    fcm_attempted 24 / succeeded 2), 96 (RESOLVED).

Conclusion STEP 2: **no corresponding QuakeAlert event in the audit window,
narrow or widened. No delayed processing observed up to 04:46 WIB.**

## 6. NODE-52960B47 observation (STEP 3)

- Identity: `NODE-52960B47`, Ngamprah, Jawa Barat
  (`lat -6.8562093, lon 107.5289622`), verified, sole Admin Node.
- Online during earthquake? `last_heartbeat` fresh at audit end (21 s age).
  Heartbeat period is 60 s (`HEARTBEAT_INTERVAL_MS 60000`, unauthenticated
  MQTT `sensor/<id>/heartbeat` → `UpdateHeartbeat`, `last_heartbeat=NOW()`).
  Fresh-now proves MQTT liveness at 21:42Z. Continuity THROUGH 21:23:56Z is
  UNKNOWN: server stores only the last heartbeat, no history table; no
  heartbeat log line at INFO; mosquitto logs show only `healthcheck` TLS
  lines at this verbosity. No disconnect/reconnect evidence found, but none
  could be produced from available server records either.
- Observations around 04:23:56 WIB: none. Last pair 90/91 at 19:40 WIB
  (8 h 43 min before BMKG time). Zero rows 20:00–22:00 UTC.
- PRELIM? None in window. FINAL? None in window. PGA? None (no trigger).
  obs_seq? None. event IDs? None. Trigger? None (`trigger diterima` count 0).
- MQTT ingestion succeeded? For the window there was nothing to ingest.
  Ingest path proven healthy by: prior pair ingested with 7–14 ms
  publish→received latency, `verify_result=OK`, `ledger_drops_total=0`,
  `ledger_unknown_node_rejections_total=0`, `ledger_write_failures_total=0`
  cumulative since 2026-09-10 boot and unchanged through the window.
- Connection gaps / I2C 263 / MPU6050 / GPIO15 / watchdog / reboot /
  firmware restart around event: UNKNOWN from server side. No `ERROR`/`WARN`
  in 20:00–21:45Z server logs besides zero-valued failure counters; no
  node-side serial access performed (forbidden). Firmware constants for
  context only: STA/LTA trigger `STA_LTA_TRIGGER_RATIO 4.0` +
  `MIN_STA_THRESHOLD_GAL 1.5`, `HEARTBEAT_INTERVAL_MS 60000`, QoS 0 with
  firmware retry (`TRIGGER_MAX_ATTEMPTS 36`, backoff 5 s, max age 300 s).

Record: **NO EVIDENCE OF SENSOR-LEVEL DETECTION** in server records.
Absence of a server event must not be read as proof the ground did not move
— only that no trigger from this node reached the server in the window.

## 7. End-to-end pipeline (STEP 4)

No candidate event exists, so every stage after ingest is empty:

- ESP32 trigger: UNKNOWN (no on-device log; no server row).
- MQTT publish: no evidence (no row, no `trigger diterima` line).
- Observation ingest (`sensor_observations`): 0 rows — earliest confirmed stop.
- Event tracker create: 0 (`event_created_total` stable at 15).
- PRELIM: none. State UNCONFIRMED: none. FINAL: none.
- Admin eligibility / consensus: not reached (nothing to evaluate).
- Dispatch (FCM/WebSocket): none in window (`alert_emissions` 0).
- Android warning: none.

Latencies: all UNKNOWN (no timestamps exist). Not estimated.

## 8. Admin Node gates (STEP 5, production policy 140 gal)

Evaluated against `AdminNodeMinPGAGal = 140.0`
(`server/internal/event/admin_node.go:30`), `AdminNodeHeartbeatMaxAge = 5 min`:

- Designated Admin Node (exactly one holder)? PASS (sole `NODE-52960B47`).
- Verified? PASS (`verified=t`).
- Heartbeat fresh ≤5 min at audit end? PASS (21 s). At 21:23:56Z? UNKNOWN
  (no history).
- Event UNCONFIRMED? FAIL (no event held UNCONFIRMED in window).
- Admin Node contributor? FAIL (no contributors).
- FINAL phase from Admin? FAIL (no FINAL).
- PGA ≥ 140 gal on Admin contribution? FAIL (no PGA; nothing to compare).
  Recorded fact: no PGA exists to compare — not "below threshold".
- Local audience available (tokens ≤20 km)? UNKNOWN (no dispatch evaluated).
- Token-targeted dispatch (`TOKENS_RADIUS_20KM`, no GeoTopic fallback)?
  UNKNOWN (no emission).
- `trusted_local=true` emitted? FAIL (0 `trusted-local emitted` lines).
- D-037 FINAL-edge (non-transition FINAL while UNCONFIRMED, exactly-once)?
  NOT APPLICABLE (no UNCONFIRMED event, no FINAL edge, `adminEdgeFired`
  never set for this window).

Threshold note: 140 gal was not the limiting factor — the chain stopped
before any PGA existed. No conclusion on threshold correctness follows.

## 9. Normal consensus (STEP 6)

- Contributing nodes: 0. Independent contributors/cells: 0.
- PRELIM received? No. UNCONFIRMED reached? No. CONFIRMED criteria
  (≥16.6 gal + ≥3 nodes + ≥2 cells/5 km): not approached (0/3 nodes).
- Advisory emitted? No. Alert emitted? No. EVENT_RESOLVED? No.
- One-node fleet CONFIRMED stays unreachable by density (S2) regardless;
  irrelevant here since no trigger existed at all.

## 10. Absence classification (STEP 7)

**A. SENSOR DID NOT OBSERVE** — earliest confirmed failure: no trigger row,
no `trigger diterima` line, counters unchanged, drops 0.

- A vs B boundary: server records cannot separate "ground motion never
  exceeded firmware STA 1.5 gal / ratio 4.0" (A) from "motion present but
  STA/LTA did not confirm within 300 ms" (B) without on-device raw accel /
  STA/LTA logs, which were not accessed. Working classification is A
  (no sensor-level detection evidence), with B explicitly not excludable.
- Consequences downstream (C–I) did not occur because there was nothing to
  ingest, track, confirm, or dispatch. No evidence for C (ingest failed),
  D, E, F, G, H, I. Not J: evidence of absence in server records is strong
  (triple-zero across tables + logs + stable counters + healthy ingest path
  demonstrated 8 h earlier).

## 11. Physical interpretation (STEP 8)

- M5.9, 376 km deep, ~6 km SE Kepulauan Seribu. Node at Ngamprah
  (-6.856, 107.529): epicentral distance order 100–160 km depending on
  island reference; hypocentral distance ~398 km for 130 km epi + 376 km
  depth. Deep in-slab event at ~400 km hypocentral range is expected to
  produce very weak surface PGA at the node (order <1 gal), plausibly below
  the firmware 1.5-gal STA floor and far below the 16.6-gal server floor.
- Seismological inference only (INFERRED, not measured): consistent with A.
- Separate questions:
  - Earthquake occurred per BMKG? Taken as reported (not verified here).
  - Detectable motion at sensor? UNKNOWN (no on-device measurement).
  - Firmware detected? No evidence (no trigger).
  - Server ingested? No (0 rows).
  - Policy accepted? N/A. Notification delivered? No.
- This is NOT a software detection failure on available evidence.

## 12. PRELIM / FINAL / PGA (STEP 9)

- PRELIM PGA: none. FINAL PGA: none. Production threshold: 140 gal.
  Research candidate: 100 gal.
- Hypotheticals are vacuous: with no trigger, neither 140 nor 100 (nor 60 /
  80 / 120 / 160) would have fired, changed timing, or mattered. FINAL was
  not the limiter; sensor detection itself was. Analysis only; no behaviour
  changed; single event never justifies threshold change (PROJECT_RULES §6).

## 13. D-036 / D-037 (STEP 10)

- D-036 local-warning path is DEPLOYED and live (schema 10, binary 8e93c4d,
  sole designated verified fresh Admin; prior event 004c5f65 shows
  `trusted-local emitted ADMIN_ELIGIBLE` + `TOKENS_RADIUS_20KM` FCM attempt).
- For THIS window D-036/D-037 were not relevant:
  - No PRELIM made any event UNCONFIRMED before FINAL (no event).
  - No Admin FINAL arrived (no FINAL).
  - 140 gal never crossed (no PGA).
  - D-037 non-transition FINAL edge did not occur (no edge, no emission).
  - No trusted-local emission (count 0); exactly-once invariant untested here.
  - D-019 privacy/audit: no raise-path log expected; none exists — satisfied
    vacuously (no coordinates logged because nothing raised).

## 14. Android delivery (STEP 11)

No warning dispatched, so:

- FCM dispatch / success / token audience: none in window.
- WebSocket dispatch / `trusted_local` / notification event: none.
- Only claim supported: SERVER DISPATCH did not occur. FCM ACCEPTED,
  DEVICE RECEIVED, NOTIFICATION POSTED, USER-AUDIBLE ALARM are all N/A.
  Prior-event note (context, not this window): emission 95 attempted 24
  tokens, 2 succeeded, 22 `NotRegistered` 404 + 1 deadline — delivery
  hygiene issue on stale tokens, unrelated to this audit.

## 15. Sensor health / firmware (STEP 12)

Production-server audit only; no firmware contact, no reset, no upload.

- I2C Error 263 / MPU6050 comms / GPIO15 / watchdog / reboot / boot seq /
  MQTT reconnect / Wi-Fi disconnect / task stalls / missing observations:
  all UNKNOWN server-side (no heartbeat history, no status-topic archive,
  no error lines in 20:00–21:45Z server logs).
- Positive health signals: `last_heartbeat` 21 s fresh at 21:42:58Z;
  broker + DB + server all `ok`; ledger drops/rejections/failures all 0;
  prior trigger pair 8 h earlier ingested cleanly (attempt 1, OK, 7–14 ms).
- Negative: 8 h 43 min trigger silence before the quake is normal for a
  quiet sensor (prior gaps similar), not evidence of outage.

## 16. Threshold research evidence (STEP 13)

For candidate Admin floors 60 / 80 / 100 / 120 / 140 / 160 gal:

- Would it have triggered? No (no observation at any PGA).
- At what stage / how much earlier / FINAL required? N/A.
- Would changing threshold have changed outcome? No.
- Status per candidate: UNKNOWN→HYPOTHETICAL-no-effect (no OBSERVED PGA to
  compare). This event provides NO calibration evidence for any candidate
  and must not be cited for/against 100 gal (PROJECT_RULES §6: population,
  not single event).

## 17. Confidence and unknowns

- Confidence HIGH that no QuakeAlert event/dispatch corresponds to BMKG
  20260912042631 in 04:00–04:46 WIB (triple-table zero, zero trigger logs,
  stable counters, healthy ingest path, live admin path demonstrated).
- Confidence MEDIUM that the stop was pre-ingest (A, sensor did not produce
  a trigger). Cannot split A vs B without firmware raw logs.
- Remaining unknowns: heartbeat continuity through 21:23:56Z; on-device
  raw motion/STA/LTA; I2C/WDT/reboot state; whether faint motion arrived
  below 1.5 gal; BMKG catalogue accuracy (out of scope; never ground truth
  per PROJECT_RULES §2).

## 18. Recommended next steps (evidence-supported only)

1. No production change, no threshold change, no deploy, no restart.
2. Optional (non-production): pull ESP32 serial/ring logs offline IF already
   buffered locally — never from production device — to split A vs B.
   If that requires a write/SSH to the node: STOP per safety rule.
3. Optional research: file this audit under threshold-research inputs as
   "no-evidence event" (counts toward denominator honesty, S5), not as a
   miss.
4. Token hygiene follow-up (separate, non-urgent): 22/24 UNREGISTERED FCM
   tokens on emission 95 — prune stale tokens out of band.

## 19. Evidence / provenance

- SSH reads (all SELECT-only, no writes):
  `docker ps`, `docker inspect quakealert-server`, `docker images`,
  `git rev-parse/status/log`, `docker logs --since/--until` (audit + control
  12:25–12:45Z), `docker logs ... | grep -c`, `psql SELECT` on
  `schema_migrations`, `iot_nodes` (+ PostGIS location), `COUNT ... WHERE
  is_admin_node`, `sensor_observations` (last-20, window, counts),
  `earthquake_events` (last-10), `event_state_log` (last-10, window count),
  `alert_emissions` (last-10, window count), `event_near_confirmed` count,
  `date/timedatectl`, public `GET /healthz`.
- Local facts: `server/internal/event/admin_node.go:30` (140.0),
  `firmware/src/config.h:103-105,130` (STA/LTA 4.0, 1.5 gal, 60 s heartbeat),
  `firmware/src/firmware.ino:479-481` (heartbeat loop).
- What was NOT done: no INSERT/UPDATE/DELETE/ALTER/DROP/TRUNCATE, no
  migrations, no `compose up/down/restart`, no file writes on VPS, no test
  alerts, no ESP32 contact, no Admin threshold/config change.

## 20. Change control

- ID: LIVE-EQ-2026-09-12-001. Timestamp: 2026-09-12 04:46 WIB.
  Topic: BMKG M5.9 deep event audit. Finding: no sensor-level detection
  evidence (A). Evidence/source: §19. Confidence: high (absence),
  medium (A vs B). Impact: none on thresholds/conclusions. Action: file only.
  Related experiment/decision: D-036/D-037 live; threshold research (no data).
  Earlier conclusions changed: none; nothing superseded.
