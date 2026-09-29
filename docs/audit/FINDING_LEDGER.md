# QuakeAlert26 — Persistent Finding Ledger

Authoritative, append-only record of best-practice audit findings, decisions,
and remediations. Created 2026-09-29 (see D-AUDIT-001).

## Ledger rules (PROCESS)

- **Append-only.** New entries are appended. Historical entries are never
  edited, deleted, or overwritten. Status changes are recorded as new dated
  entries referencing the finding ID.
- **ID namespaces.** `AND` Android code · `ASE` Android security · `APE`
  Android performance · `AUI` Android UI (audit-only) · `SRV` Go server ·
  `DB` database · `MQ` MQTT · `FW` firmware · `DEP` deploy/docker ·
  `API` contracts · `CHT` chat · `CIT` CI/testing · `OSS` open-source repo ·
  `DOC` docs/hardware · `D-AUDIT` audit decisions.
- **Status lifecycle.** `OPEN` → `IN PROGRESS` → `RESOLVED` (only with
  Finding ID → commit → test → verification). `DEFERRED` / `WONTFIX`
  require a recorded reason. `INTENTIONAL` = accepted tradeoff, kept visible.
- **Decision classes.** REQUIRED / STRONGLY RECOMMENDED / OPTIONAL /
  NOT APPLICABLE / INTENTIONAL TRADE-OFF (per audit brief).
- **Evidence standard.** Every finding carries exact `path:line`. Severity
  claims without a line reference are not findings.

## Prior audit history (preserved, not moved)

- `docs/planning/admin-node-threshold/09-findings.md` — threshold research.
- `docs/planning/security/B1-secret-rotation-inventory.md` — B1 rotation +
  Phase-B completion record (HUP lesson, §7).
- `docs/planning/licensing/final-paho-v151-licensing-decision-audit.md`,
  `docs/planning/licensing/final-dependency-license-compatibility-check.md`
  — licensing audits.
- `docs/planning/release-readiness-master-audit.md` — release readiness.
- `firmware/AUDIT.md` — firmware/bench audit notes.

---

## Audit record A-2026-09-29-BP — best-practice compliance audit

- **Scope.** Full monorepo at `main` (`5b63648` post-fix): `android/`,
  `server/`, `firmware/`, `contracts/`, `deploy/`, `docs/`, `.github/`.
- **Constraints.** UI/UX audit-only (owner implements manually; zero UI
  files touched). Non-UI code changes only if low-risk, localized,
  test-covered, unrelated to production credentials/architecture.
  No production access, no push, no rewrite.
- **Methods.** Five read-only area audits + independent firsthand
  verification of every REQUIRED-class claim (guard_test.go, dispatcher.go,
  waitFor helper, atomic Drops, init-mqtt-users.sh vs B1 record,
  network.cpp refreshLocation path, router.go vs openapi.yaml,
  CONTRIBUTING.md). Tool runs: `gofmt -l` (1 hit, scripts-only),
  `go vet` clean, `go build` clean, `go test ./...` all-ok post-fix,
  `go test -race` clean on dispatch/event/ingest/ledger.

---

## Findings — Android code (AND)

- **AND-001 OPEN — STRONGLY RECOMMENDED (real defect, minor).**
  `android/app/src/main/java/id/web/quakealert/ui/app/AppRoot.kt:82` —
  `showAddSensor` is `remember`, not saveable/VM-owned; rotation hides the
  in-progress provisioning wizard while `AddSensorViewModel` keeps server
  side-effects. Fix: `rememberSaveable` or hoist to VM (cf.
  `QuakeFilterViewModel.isSheetOpen` pattern).
- **AND-002 OPEN — STRONGLY RECOMMENDED (real defect, minor).**
  `ui/warning/WarningViewModel.kt:110-114` — `protectionFacts` reads the OS
  notification grant once; goes stale after revoke-in-background (Settings
  re-reads on resume; this does not). Fix: resume re-read /
  `refreshProtectionFacts()`.
- **AND-003 OPEN — OPTIONAL (robustness).**
  `service/QuakeMessagingService.kt:74-76` — `runBlocking` DataStore read on
  the FCM thread. Fix: move into the already-async path or cache language.
- **AND-004 OPEN — OPTIONAL (maintainability).**
  `data/AppSettingsRepository.kt:203-205` — fire-and-forget DataStore writes
  swallow failures. Fix: log on failure.
- **AND-005 OPEN — OPTIONAL (style/tiny race).**
  `ui/sensors/SensorsViewModel.kt:227` — reads `_uiState.value.filter`
  inside `update`; use the lambda `state` instead.
- **AND-006 OPEN — OPTIONAL (style).**
  `domain/ActiveAlertBoard.kt:126`, `ui/common/QuakeMap.kt:185` — two safe
  but avoidable `!!`. Fix: `?: return` / `let`.
- **AND-007 INTENTIONAL — documented tradeoff.**
  `ui/main/MainScreen.kt:180` — updates overlay `showUpdates`
  `remember`-only; rotation closes it, reopen is a cheap GET. No action.

## Findings — Android security (ASE)

- **ASE-001 NOT APPLICABLE — verified clean.** PendingIntents all
  `FLAG_UPDATE_CURRENT|FLAG_IMMUTABLE`, explicit intents, distinct codes
  (`service/WarningNotifier.kt:138`, `StatusNotifier.kt:115,175`,
  `UpdatesNotifier.kt:104`); `WarningActivity exported=false`; extras
  degrade safely (`ui/warning/WarningActivity.kt:248-274`). Recorded to
  close the check.
- **ASE-002 NOT APPLICABLE — verified clean.** Cleartext denied app-wide
  except node SoftAP `192.168.4.1` (`network_security_config.xml`); release
  base is `https://` (`app/build.gradle.kts:98-128`); no pinning is the
  correct single-APK default. No change.
- **ASE-003 NOT APPLICABLE — verified clean.** JWT in internal DataStore,
  `allowBackup=false`, raise-path logs redact coordinates
  (`data/users/UserLocationRepository.kt:243-249`,
  `service/AlertRaiser.kt:102-107`); FCM data-only with two-sided drill
  fence (`data/push/PushRegistrar.kt:128-133`); `is_test`/`trusted_local`
  fail toward real-alert. No change.
- **ASE-004 OPEN — OPTIONAL (release-process note, not a code defect).**
  `AndroidManifest.xml:10,27` — `REQUEST_IGNORE_BATTERY_OPTIMIZATIONS` +
  `USE_FULL_SCREEN_INTENT` need Play-policy declarations before any Play
  release. Do not remove; code use is correct
  (`service/WarningNotifier.kt:176-180`).

## Findings — Android performance (APE)

- **APE-001 OPEN — STRONGLY RECOMMENDED (credible perf risk).**
  `device/AlertSiren.kt:74-86` — blocking `MediaPlayer.prepare()` on the
  main thread from `WarningActivity.onCreate`
  (`ui/warning/WarningActivity.kt:85`); can jank the first alert frame.
  Fix: `prepareAsync`/`onPrepared` or build on `Dispatchers.IO`. (Alert-path
  code: owner decision required before implementation — NOT implemented by
  this audit.)
- **APE-002 NOT APPLICABLE — verified bounded.** Health probe 30s,
  `WhileSubscribed(5s)`, WS ping 20s, location `syncIfStale` 6h+1km
  (`data/network/ServerHealthMonitor.kt`,
  `data/network/QuakeNetwork.kt:67-71`,
  `data/users/UserLocationRepository.kt`). No change.
- **APE-003 NOT APPLICABLE — verified fine.** All lists keyed + paged;
  no bitmap handling; single MapLibre GeoJSON source
  (`ui/common/QuakeMap.kt:217-241`). No change.

## Findings — Android UI, audit-only (AUI) — NO UI FILES MODIFIED

- **AUI-001 informational.** Loading/error/empty/offline/no-position states
  consistently covered across History/Sensors/Chat/Updates/Warning. No action.
- **AUI-002 informational.** Accessibility semantics present on interactive
  chrome (Role.Tab/RadioButton/Button/Switch, 55dp targets,
  `ui/main/MainScreen.kt:369-385`). Siren/mute controls not fully traced;
  not claimed as defect. Manual follow-up may confirm.
- **AUI-003 informational.** Fixed dark theme only; light-mode toggle
  disabled; window background matches app black (no flash on recreate).
  Intentional single-theme constraint. Owner-manual if revisited.
- **AUI-004 informational.** Portrait-only activities; non-interactive maps
  except wizard picker. Intentional. Owner-manual if revisited.
- **AUI-005 informational.** Life-safety copy discipline observed
  ("Distance unknown", drill-vs-real separation, resolved-vs-cancelled).
  No action.

## Findings — Go server (SRV)

- **SRV-001 RESOLVED → commit `5b63648`, test:
  `go test -run TestBlockedLedgerDoesNotDelayDispatch -count=6
  ./internal/dispatch/` 6/6 PASS (pre-fix ~2/5), full `go test ./...`
  all-ok, `go vet`/`gofmt` clean. Verification: firsthand
  (`guard_test.go:380-383` vs `dispatcher.go:239` goroutine;
  `waitFor` at `dispatcher_test.go:142`; `Drops()` atomic at
  `ledger/writer.go:195,610`).** Was REQUIRED (flaky S1 guard):
  `server/internal/dispatch/guard_test.go:380-383` read `w.Drops()`
  synchronously while `dispatchFCM` enqueues via goroutine. Now polls with
  the file's existing `waitFor`.
- **SRV-002 OPEN — STRONGLY RECOMMENDED (S1 latency, documented tradeoff).**
  `server/internal/dispatch/dispatcher.go:174-192` — Fase-2 `Dispatch`
  does synchronous `SaveEvent` before WS/FCM broadcast on the default path
  (`EVENT_TRACKER_ENABLED=false`, `config.go:199`). Fase-3
  (`event_frame.go:27-61`) already emits first. Do NOT reorder alone
  (stable `event_id` is load-bearing for dedup/all-clear); document the S1
  exception or prefer the Fase-3 path. Owner decision required.
- **SRV-003 OPEN — STRONGLY RECOMMENDED (real defect).**
  `dispatcher.go:186-192` — `SaveEvent` failure broadcasts ID-less
  CONFIRMED alert (fail-open correct, but dedup/all-clear break during DB
  incidents). Define the ID-less CONFIRMED contract or emit a marked
  ephemeral ID. Owner decision required.
- **SRV-004 OPEN — STRONGLY RECOMMENDED (DoS/availability).**
  `server/internal/api/api.go:445,546,585` et al. (~20 sites) — HTTP
  handlers pass `r.Context()` to Postgres with no per-query timeout,
  contrary to ADR-0002 §3 (≤2s); only MQTT ingest + healthz bound IO.
  Fix: wrap with `cfg.IOTimeout` or pool statement timeout. Non-warning
  path; routine change, left for owner scheduling.
- **SRV-005 INTENTIONAL — recorded tradeoff, no action.**
  `ingest/verifier.go:192,241`, `consensus/engine.go:113-118`,
  `event/tracker.go:191-195` — DB reads on the decision path (auth +
  geometry) fail closed on DB outage: correct vs spoofing (S4), in tension
  with literal S1. Any secret/location cache is a PROPOSAL with
  staleness/revocation hazards, not a recommendation.
- **SRV-006 OPEN — OPTIONAL.** `config/config.go:388,171,390-393` —
  package-level `parseWarnings` without mutex (boot-only in prod; tests
  only risk). Fix: local accumulator.
- **SRV-007 OPEN — OPTIONAL (style).** `server/scripts/admin_floor_sweep.go`
  sole `gofmt -l` hit (build-ignored research script). Fix: `gofmt -w`.
- **SRV-008 OPEN — OPTIONAL.** `ingest/subscriber.go:61-63` — "QoS 1
  (life-safety, at-least-once)" comment contradicts D-008 (effective QoS 0).
  One-line comment correction per PROJECT_RULES §8. Left for owner batch.
- **SRV-009 INTENTIONAL — noted tradeoff, not a vuln.**
  `api/jwt.go:44-92`, `config.go:197` — 30-day anonymous JWTs, no
  revocation; implementation sound (`alg` pinned, `hmac.Equal`, mandatory
  `exp`/`sub`); no JWT path reaches dispatch. No action.
- **SRV-010 OPEN — OPTIONAL (conditional hardening).**
  `api/router.go:31`, `api/api.go:298-304` — rate-limit IP derives from
  `RealIP` (trusts `X-Forwarded-For`); safe only behind the assumed proxy
  (`cmd/quakealert/main.go:335`). Fix: document edge-stripping / never
  expose `:8080` directly.
- **SRV-011 OPEN — OPTIONAL.** `api/admin_nodes.go:85` — empty-verify rule
  depends on `ContentLength` (chunked empty body → 400 vs `Content-Length: 0`
  → approve default). Operator ergonomics only.
- **SRV-012 NOT APPLICABLE as defect / OPTIONAL hardening.**
  `cmd/quakealert/main.go:515-527` — `WS_ALLOWED_ORIGINS="*"` documented;
  impact bounded (Bearer-JWT WS, TLS at proxy). Optional: warn-log on `*`.

## Findings — database (DB)

- **DB-001 OPEN — STRONGLY RECOMMENDED.**
  `server/internal/store/store.go:32-53` — pool has no statement timeout /
  `MaxConnLifetime`. One hung backend holds 1/8 of the pool indefinitely.
  Fix: pool-level timeout + staggered lifetime. (Pairs with SRV-004.)
- **DB-002 OPEN — STRONGLY RECOMMENDED (forensic consistency).**
  `store.go:263-294` — `ResolveEvent`/`ResolveStaleEvents` write terminal
  `RESOLVED` without an `event_state_log` row, breaking the D-012 log
  contract. Fix: route through `EventUnit` or document the exception.
- **DB-003 OPEN — OPTIONAL (perf).**
  `store/event_lifecycle.go:201`, `store.go:283-288`, `chat.go:238-244` —
  no index on `earthquake_events(status)` / `chat_messages(sender_id)`.
  Fix: partial index + `(sender_id, created_at)` index.
- **DB-004 OPEN — OPTIONAL.** `000001_init_schema.up.sql:64`,
  `event_lifecycle.go:87`, `event_timeline.go:54` — dual NULL/zero for
  unknown times. Convention works; three states for two meanings.
- **DB-005 OPEN — OPTIONAL (style).**
  `contracts/db/migrations/000010_admin_node.up.sql:2` — header says
  D-036 PROPOSED; `docs/DECISIONS.md:1042` says ACCEPTED. Comment-only;
  left untouched (applied migration; owner batch).
- **DB-006 INTENTIONAL — no action.** `event_lifecycle.go:91-152`,
  `ledger/writer.go:456-498` — non-transactional upsert+log pair is
  designed-for (S1: persistence follows emission). Do not re-flag.
- **DB-007 INTENTIONAL — no action.** `ingest/obsseq.go:20-39` — dedup in
  memory by design (DB cannot replace it; async drop-oldest ledger).
  Bounded restart window accepted. Do not re-flag.

## Findings — MQTT (MQ)

- **MQ-001 INTENTIONAL — D-008, no action.** `firmware/src/mqtt.cpp:38-52`,
  `contracts/mqtt/trigger.schema.json:5` — device QoS 0 with retry+dedup
  model. Honestly named per PROJECT_RULES §8.
- **MQ-002 OPEN — OPTIONAL.** `deploy/mosquitto/mosquitto.prod.conf:54-56`
  — stale "QoS 1 dipakai untuk trigger" comment contradicts D-008.
  Comment-only; deploy file left untouched by this audit (owner batch).
- **MQ-003 OPEN — OPTIONAL.** `server/internal/ingest/subscriber.go:61-62,74-77`
  — subscribe-side "at-least-once" comment overclaims end-to-end (device
  leg is QoS 0). Scope the comment to the subscription leg. Same as
  SRV-008; owner batch.
- **MQ-004 OPEN — OPTIONAL.** `ingest/verifier.go:225-248`,
  `ingest/obsseq.go:61-71`, `firmware/src/firmware.ino:300-311` — QoS 1
  duplicate handling correct except bounded post-restart window
  (firmware re-stamps `ts`). Document the window; no protocol change.
- **MQ-005 OPEN — OPTIONAL (hardening).** Broker does not set
  `retain_available false` (`mosquitto.prod.conf`); publishers are clean
  (`mqtt.cpp:51,432` non-retained). A retained poison frame would replay
  on resubscribe (bounded by freshness+HMAC). Fix or document acceptance.
- **MQ-006 OPEN — OPTIONAL.** `cmd/quakealert/main.go:481`,
  `docker-compose.prod.yml:85`, `mqttclient.go:156-174` — fixed ClientID +
  persistent session assumes single instance (ADR-0001); concurrent
  instances would flap. Document the assumption or unique-ify the ID.
- **MQ-007 OPEN — STRONGLY RECOMMENDED (operational defect).**
  `deploy/scripts/init-mqtt-users.sh:77` instructs `kill -s HUP mosquitto`
  to reload credentials; B1 record proves HUP does NOT reload passwd
  (`docs/planning/security/B1-secret-rotation-inventory.md:117-119`).
  Following the script silently fails to apply rotations. Firsthand
  verified both sides. NOT changed (deploy script + production-credential
  procedure → owner must approve; see D-AUDIT-005).
- **MQ-008 OPEN — STRONGLY RECOMMENDED (stale design doc).**
  `firmware/src/mqtt.cpp:408-433` LWT/`status` frames are dead letters
  (server subscribes only trigger+heartbeat,
  `ingest/subscriber.go:13-17,63-72`); `docs/ESP32_SYSTEM_DESIGN.md:36`
  describes a nonexistent Python/`seismo/*` system. Subscribe-and-handle
  or document unconsumed-by-design; banner/fix the doc (see DOC-002).
- **MQ-009 INTENTIONAL — no action.** `mosquitto/aclfile:24-34`,
  `ingest/verifier.go:192-223` — fleet-shared node credential contained by
  per-node HMAC + verified gate. Documented tradeoff.
- **MQ-010 OPEN — OPTIONAL (defensive).** `cmd/quakealert/main.go:467-474`
  — Go-side TLS hostname verification depends on `gomqtt/transport`
  propagating `ServerName`. Set it explicitly from the broker host.
- **MQ-011 INTENTIONAL — no action.** `firmware.ino:313-367`,
  `config.h:131-133` — RAM-only offline queue with loud drops + 5-min cap
  matching server freshness window. Correct device tradeoff.

## Findings — firmware (FW)

- **FW-001 OPEN — STRONGLY RECOMMENDED (reliability).**
  `firmware/src/sensor.cpp:63-104,351-368` — MPU init failure bricks
  sensing (task parks on `portMAX_DELAY`); only the 30s task-WDT
  reboot-loop exits, each reboot writing NVS boot counter
  (`firmware.ino:191-194`); node looks alive via heartbeat but can never
  trigger. Needs dead-sensor safe state + backoff. Hardware-dependent;
  NOT implemented by this audit.
- **FW-002 OPEN — STRONGLY RECOMMENDED (concurrency, low severity).**
  `sensor.cpp:143,149,165,334` vs `mqtt.cpp:336,340,343-344,363-364` vs
  `network.cpp:575` — diagnostic counters/status (`mpuErrorCounter`,
  `DMPReady`, `lastPgaStr`, …) shared across tasks without a common lock
  (event slots themselves are correctly guarded). Guard or snapshot under
  lock. Hardware-dependent; NOT implemented.
- **FW-003 OPEN — REQUIRED (performance/correctness).**
  `firmware/src/network.cpp:575-578`, `config.h:126`, `state.h:163`,
  `firmware.ino:121` — `LOCATION_RETRY_INTERVAL_MS` defined but never
  referenced; `lastLocRetry` write-only; while unresolved, every 250ms
  iteration calls full `refreshLocation()` (`network.cpp:472-549`:
  `WiFi.scanNetworks` + BeaconDB HTTPS POST + ipinfo fallback). Firsthand
  verified (the "cheap NVS read" comment at :570 holds only when the cache
  exists). Spams APIs at ~4Hz, stalls WiFi/NTP maintenance. Gate retries
  on the defined interval + `http.setTimeout(HTTP_TIMEOUT_MS)`.
  Hardware-dependent; NOT implemented by this audit (owner + bench).
- **FW-004 OPEN — STRONGLY RECOMMENDED.**
  `network.cpp:421-461,487-550`, `config.h:119` — blocking HTTPS with no
  timeout (`HTTP_TIMEOUT_MS` defined, unused); 2–4KB JSON docs + `String`
  bodies in the 12KB maintenance task. Apply the defined timeout; bound
  allocations. NOT implemented (owner + bench).
- **FW-005 OPEN — STRONGLY RECOMMENDED (security).**
  `network.cpp:185-189` — `/config` persists any non-empty `hmac_key`
  without validation (unlike `station_id`/port/broker). Weak key ⇒ all
  triggers HMAC-rejected server-side with no local signal. Validate
  length/hex before NVS write. NOT implemented (owner + bench).
- **FW-006 OPEN — OPTIONAL.** `mqtt.cpp:128-131`, `utils.cpp:224-253` —
  NVS reopened per publish retry; stack key copy never wiped. Cache key
  in RAM at connect; `memset` after use.
- **FW-007 OPEN — OPTIONAL.** `mqtt.cpp:404-406` — `random()` unseeded
  (no `randomSeed()` in tree); clientId repeats across boots. Seed from
  `esp_random()` (already used at `firmware.ino:149`).
- **FW-008 OPEN — STRONGLY RECOMMENDED (perf).**
  `mqtt.cpp:282-327`, `firmware.ino:482-483` — I2C temperature/connection
  probe inside the MQTT callback (up to 100ms mutex + 2 transactions)
  stalls loop-time alert/heartbeat work. Defer to sensor task or cache.
  NOT implemented (owner + bench).
- **FW-009 OPEN — OPTIONAL.** `utils.cpp` (14 sites), `network.cpp:78,96`
  — `stateMutex` always `portMAX_DELAY`. Bounded takes where feasible;
  WDT-covered today.
- **FW-010 OPEN — OPTIONAL.** `config.h:26,31`, `firmware.ino:214-246` —
  hand-tuned task stacks with no `uxTaskGetStackHighWaterMark` check.
  Log watermarks at boot.
- **FW-011 OPEN — STRONGLY RECOMMENDED.**
  `firmware/platformio.ini:5-17` — `espressif32`/framework float, `^`
  lib ranges; `AUDIT.md:105-107` "ter-pin" claim overstates. Pin platform
  + Core (also CIT-004 / LGPL relink note).
- **FW-012 OPEN — OPTIONAL (style).**
  `firmware/src/2026-09-27-045555-this-session-is-being.txt` — stray
  session scratch in source dir (gitignored, not compiled, no secrets).
  Delete at owner convenience.

## Findings — deploy/docker (DEP)

- **DEP-001 OPEN — STRONGLY RECOMMENDED.**
  `server/Dockerfile:17,41`, `deploy/docker-compose.prod.yml:32,150,175,193,214`,
  `server/docker-compose.yml` — only prod PostGIS is digest-pinned; six
  other images float. Pin digests for the life-safety stack. (Deploy
  surface: owner approves; this audit changed nothing.)
- **DEP-002 OPEN — OPTIONAL.** No `user:`/`cap_drop` in either compose;
  non-root enforced only via server image (`Dockerfile:61`). Assert
  least-privilege per service.
- **DEP-003 INTENTIONAL — no action.** `docker-compose.prod.yml:74-93,229-238`,
  `init-mqtt-users.sh:55-68` — secrets visible in `docker inspect`/`ps`
  is the standard compose pattern (nothing baked into layers;
  `.dockerignore` excludes secret material). Scope = host compromise.
- **DEP-004 OPEN — OPTIONAL.** No `stop_grace_period`/`stop_signal`
  anywhere; 10s default SIGTERM→SIGKILL for PostGIS/Caddy/Mosquitto.
  Tune grace explicitly.
- **DEP-005 OPEN — OPTIONAL (style).** `server/Dockerfile:45,50-53` —
  duplicate `ARG VERSION` + LABEL block (merge artifact, harmless).
- **DEP-006 INTENTIONAL — no action.** `docker-compose.prod.yml:53-59`,
  `Caddyfile:41-58` — Caddy `service_started` on server is by-design
  (distroless ⇒ no exec healthcheck); covered by `health_uri` + 30s retry.

## Findings — API/contracts (API)

- **API-001 OPEN — REQUIRED.** `contracts/openapi/openapi.yaml:1558-1577`
  (`RerollResponse` promises nullable `region_code`) vs
  `server/internal/api/api.go:757-760` (never sent) vs
  `UpdateLocationDto.kt:90-93` (not modeled). Emit it or remove from
  contract. (Public contract → owner decision; NOT changed.)
- **API-002 OPEN — REQUIRED.** `api.go:854-864,913-927` sends
  `region_code`; `openapi.yaml:1848-1872` omits it. Add to contract.
  Mirror of API-001.
- **API-003 OPEN — REQUIRED.** `api.go:607-626,681-689` sends
  `status=Pending` + `verified`; `openapi.yaml:1515-1556` allows only
  Online/Offline, no `verified`. Strict validators reject the most
  important sensor state. Fix contract.
- **API-004 OPEN — REQUIRED.** `router.go:83` `POST /api/v1/admin/test-alert`
  (202, `testalert.go:66-87,185`) has no OpenAPI path (verified: zero
  `test-alert` hits in `openapi.yaml`). Add path + 202 + error codes.
- **API-005 OPEN — STRONGLY RECOMMENDED.** `dispatch/fcm.go:43-54`
  (no `node_count`) vs `dispatch/ws.go:53` (always sent) vs
  `contracts/mqtt/alert.schema.json:62-66` (required) vs
  `contracts/fcm/alert_payload.json:32-42` (absent) vs
  `FcmAlertMapper.kt:52` (`?:0`). Background (life-safety) path shows 0
  reporters. Add optional `node_count` to FCM contract + emit.
- **API-006 OPEN — REQUIRED.** `api/chat.go:205-212` (no UUID validation)
  + `store/chat.go:291` (`::uuid` cast) + `chat.go:297-300` (→ 500):
  malformed `client_message_id` (contract says `format:uuid`,
  `openapi.yaml:1792-1799`) yields 500 not 400. Validate pre-DB.
- **API-007 OPEN — STRONGLY RECOMMENDED.** `api.go:127-146`
  (`nodeLocationName` rejects empty/`NODE-`/char-runs) vs
  `openapi.yaml:1414-1446` (only `maxLength:150`). Document rules in
  contract `description`.
- **API-008 OPEN — OPTIONAL.** `api/healthz.go:40,53-59` sends
  `"mqtt":"unknown"` when no probe; contract (`openapi.yaml:84-90`) says
  omit. Omit on nil probe or amend contract.
- **API-009 OPEN — STRONGLY RECOMMENDED.** `api/chat.go:241-246,282-291`,
  `chatfilter.go` — blocked-word 400 and duplicate-content 429 invisible
  in contract (`openapi.yaml:593-608`). Document codes/messages; no
  behavior change.
- **API-010 OPEN — STRONGLY RECOMMENDED.** `000001_init_schema:73-82`
  (`message TEXT`, no CHECK) vs contract `maxLength:500`
  (`openapi.yaml:1782-1785`); `store/chat.go:32-35` comment promises a
  store-level check that `InsertChatMessage` lacks. Add check or DB
  CHECK + fix the comment.

## Findings — chat (CHT) — chat EXISTS, no CHAT-XXX no-chat record

Evidence: `docs/CHAT_DESIGN.md`, `server/internal/api/chat.go`,
`chatfilter.go`, `server/internal/store/chat.go`, migrations `000001:73-84`
+ `000003`, `dispatch/ws.go:96-106,244-275`, `android/.../ui/chat/*`.

- **CHT-001 OPEN — STRONGLY RECOMMENDED.** `api/chatfilter.go:24-52` —
  ~26-word closed-world whole-word filter; no leetspeak/normalization/
  homoglyph/zero-width handling; bypass trivial. Document limits; expand
  only via reviewed policy. No implementation by this audit.
- **CHT-002 OPEN — STRONGLY RECOMMENDED before promoting chat beyond
  community channel; OPTIONAL if it stays explicitly non-instructional.**
  `CHAT_DESIGN.md:169-182`, `openapi.yaml:1738-1743`,
  `store/chat.go:275-320`, `ws.go:96-106` — no report/block/mute/
  admin-delete; `is_admin` always false. Current tradeoff (chat
  non-authoritative; alerts unwritable) is disclosed; recorded either way.
- **CHT-003 OPEN — STRONGLY RECOMMENDED.** `api/chat.go:25,268-277`
  (2s window), `chatfilter.go:17` + `chat.go:282-291` (5-min duplicate),
  `router.go:62-64` (auth), `chat.go:329-348` (server-side membership) —
  present per-user controls; gap: no per-IP/global brake, store does not
  re-enforce length (see API-010).
- **CHT-004 OPEN — OPTIONAL.** `dispatch/ws.go:182-196`,
  `QuakeWebSocketClient.kt:238-242`, `CLIENT_SPEC.md:233-234,248-252` —
  WS channel snapshot requires client `reconnect()` after region change.
  Verify all region-change call sites reconnect.

## Findings — CI/testing (CIT)

- **CIT-001 OPEN — STRONGLY RECOMMENDED (security).**
  `.github/workflows/ci.yml:1-453` — no secret scanning
  (gitleaks/trufflehog); `check_contracts.sh:80-85` only greps private
  keys. Add gitleaks job; keep key check.
- **CIT-002 OPEN — STRONGLY RECOMMENDED.**
  `server/scripts/check_contracts.sh:45-78` (run at `ci.yml:167-170`) —
  checks codes/presence only; API-001..004 drift passes CI. Extend to diff
  routes vs `paths` + required fields/enums; fail loud (protects ADR-0004).
- **CIT-003 OPEN — OPTIONAL.** CI stops at `gofmt`/`vet`/`lint`/`shellcheck`
  (`ci.yml:54-64,110-114,190-199`); add `govulncheck` + `staticcheck`.
- **CIT-004 OPEN — STRONGLY RECOMMENDED (repro + legal).**
  `firmware/platformio.ini:5-7` unversioned platform/framework;
  `THIRD_PARTY_NOTICES.md:24-31` already owes pinning + LGPL relink ref.
  Pin platform + Core version. (Same root cause as FW-011.)
- **CIT-005 NOT APPLICABLE — baseline recorded.** `ci.yml:69-74`
  (`-race -count=1`), serial sims, `CONTRIBUTING.md:46-48` documents the
  (now-fixed) S1 flake's `GOMAXPROCS>=2` need. SRV-001 fix supersedes the
  flake note; the `GOMAXPROCS` guidance stays valid. No action.
- **CIT-006 OPEN — STRONGLY RECOMMENDED (targeted tests, not %).**
  Unasserted: malformed `client_message_id` 500-path (API-006), `mqtt`
  unknown-vs-absent (API-008), provision `NODE-`/char-run rejections,
  FCM `node_count` absence, firmware on-device faults (bench-only),
  Android instrumented tests not in CI (only `testDebugUnitTest`).

## Findings — open-source repo (OSS)

- **OSS-001 RESOLVED → commit `5b63648`. Verification: firsthand read of
  `CONTRIBUTING.md:7-9` pre-fix; post-fix text points at `LICENSING.md`
  scope map (firmware GPL / server+deploy AGPL / Android GPL+exception /
  docs+contracts pending).** Was REQUIRED: stale "no license chosen"
  notice contradicted the licensed tree.
- **OSS-002 OPEN — STRONGLY RECOMMENDED (legal).**
  `LICENSING.md:71-77` — `docs/` + `contracts/` + root meta all-rights-
  reserved pending. The integration surface (OpenAPI/MQTT/FCM schemas)
  cannot be safely reused by forks/integrators. Choose CC-BY-4.0 /
  Apache-2.0 or state intent. Owner + counsel decision.
- **OSS-003 OPEN — OPTIONAL.** No `CODEOWNERS` (`.github/` holds only
  `workflows/ci.yml`). Add minimal owners for `contracts/` +
  ingest/consensus/dispatch + firmware crypto.
- **OSS-004 OPEN — OPTIONAL.** No issue/PR templates; contract/decision
  checklist (`CONTRIBUTING.md:70-77`) unenforced. Add templates (relates
  to API-001..004 drift).
- **OSS-005 OPEN — OPTIONAL.** No root changelog/tag/release workflow
  (only `docs/planning/admin-node-threshold/CHANGELOG.md`;
  `deploy/README.md:40-53` deploys by SHA). Add `CHANGELOG.md` + tag
  convention at first published release.
- **OSS-006 OPEN — OPTIONAL.** `CODE_OF_CONDUCT.md:1-46` truncated
  (`APPEND-MARKER`, no contact); committed root
  `google-services-{debug,release}.json` vs README "(uncommitted)"
  wording confuses secret hygiene. Finish CoC; clarify the Firebase files
  carry no secrets.

## Findings — docs/hardware (DOC)

- **DOC-001 OPEN — STRONGLY RECOMMENDED.** No BOM/wiring/assembly/
  enclosure/power spec (no `firmware/BOM*`, no `docs/HARDWARE*`; only
  `config.h:152-158` pins + `platformio.ini:14-17` libs). Publish a
  minimal facts-only guide (tested board + IMU breakout, pin table,
  flashing/provisioning links, reset + LED facts); mark enclosure/power/
  mounting TBD. No assumed instructions.
- **DOC-002 OPEN — REQUIRED (docs defect).**
  `docs/ESP32_SYSTEM_DESIGN.md:5-6,13,32-36` describes a superseded
  Python/`seismo/*`/single-sensor-broadcast system, contradicting
  `contracts/mqtt/*.schema.json`, Go+PostGIS consensus, and FCM+WS
  dispatch. Banner as superseded pointing at SYSTEM_SPEC/CLIENT_SPEC/
  `contracts/`; do not delete history. (Overlaps MQ-008.)
- **DOC-003 OPEN — STRONGLY RECOMMENDED.** No operator node-lifecycle page:
  compose solely from cited behaviors (`api.go:495-603` revoke rules,
  `firmware.ino:395-421` factory reset, `AUDIT.md:66-99` rotation order,
  serial failure signatures). No new procedures invented.

---

## Remediation log

- **2026-09-29 — SRV-001 RESOLVED.** Finding ID → commit `5b63648`
  (`server/internal/dispatch/guard_test.go`: poll `w.Drops()>0` via
  existing `waitFor`) → test `TestBlockedLedgerDoesNotDelayDispatch
  -count=6` 6/6 PASS + full `go test ./...` all-ok + `go vet`/`gofmt`
  clean → RESOLVED. Note: `CONTRIBUTING.md:46-48` `GOMAXPROCS` guidance
  remains valid; the flake it described is fixed.
- **2026-09-29 — OSS-001 RESOLVED.** Finding ID → commit `5b63648`
  (`CONTRIBUTING.md:7-9` rewritten to the `LICENSING.md` scope map) →
  test: textual verification (no code path) → RESOLVED.
- **2026-09-29 — API-001 RESOLVED.** Finding ID → commit `4430926`
  (`contracts/openapi/openapi.yaml`: `region_code` removed from
  `RerollResponse`) → tests: `check_api_drift.py` DTO parity
  RerollResponse↔rerollResponse 2/2 fields + full `go test ./...` all-ok +
  `check_contracts.sh` 32/32 → verification: handler (`api.go:715-718`)
  never sent it, Android `RerollPseudonymResponseDto` never modeled it,
  reroll performs no region computation (`UpdatePseudonym` only), and the
  removed description referenced an "update" reroll does not perform —
  stale copy-paste, option B → RESOLVED.
- **2026-09-29 — API-002 RESOLVED.** Finding ID → commit `4430926`
  (`openapi.yaml`: nullable `region_code`, maxLength 50, added to
  `UpdateLocationResponse`) → tests: DTO parity 6/6 fields + suite +
  contracts 32/32 → verification: handler computes/returns it
  (`api.go:913-927`), Android consumes it (`UpdateLocationDto.kt:66`,
  `QuakeApiClient.kt:243`), CLIENT_SPEC documents `{…, region_code|null}`
  — implementation is the source of truth, contract was stale → RESOLVED.
- **2026-09-29 — API-003 RESOLVED.** Finding ID → commit `4430926`
  (`openapi.yaml` Station: enum `[Online, Offline, Pending]` + `verified`
  boolean with trust-before-health description) → tests: DTO parity 11/11
  + Station.status enum gate + suite + contracts 32/32 → verification:
  semantics unchanged (code already sent both; `api.go:681-689`,
  migration 000005, Android `SensorDto`/`SensorMappers` already handle
  both) — contract now represents reality, nothing simplified → RESOLVED.
- **2026-09-29 — API-004 RESOLVED.** Finding ID → commit `4430926`
  (`openapi.yaml`: `POST /api/v1/admin/test-alert` + `CreateTestAlertRequest`
  + `TestAlertResponse`, adminKey security, 202/400/401/503/500) →
  tests: route↔path parity 22/22 + 202-code check + suite + contracts
  32/32 → verification: official operator endpoint (CLIENT_SPEC §5.8,
  `deploy/scripts/test-alert.sh` expects 202, sibling admin routes all
  documented, Error.code enum already anticipated drill UNAVAILABLE) —
  documenting changes no exposure; handler untouched → RESOLVED.
- **2026-09-29 — API-006 RESOLVED.** Finding ID → commit `4430926`
  (`server/internal/api/chat.go`: `clientMessageIDPattern` + 400
  INVALID_ARGUMENT for non-empty non-UUID, before membership/rate-limit/DB;
  empty stays allowed as "no key"; `chat_test.go`: corrected
  `PersistsThenBroadcasts` fixture `"c-1"` → contract-example UUID per
  authority hierarchy, added
  `TestCreateChatMessage_MalformedClientMessageIDIsBadRequest` — 4 malformed
  shapes → 400 + envelope code + nothing stored + quota unspent, valid UUID
  → 201 passthrough) → tests: new test PASS, full `go test ./...` all-ok,
  `go test -race ./internal/api/` clean, `go vet`/`gofmt` clean →
  verification: malformed input can no longer reach the `::uuid` cast
  (`store/chat.go:291`); error envelope and quota-ordering conventions
  preserved → RESOLVED.
- **2026-09-29 — CIT-002 RESOLVED.** Finding ID → commit `4afd603`
  (`server/scripts/check_api_drift.py` new + `check_contracts.sh` §7 runs
  gate and `--selftest`) → tests: gate 8/8 PASS on reconciled tree;
  selftest replays all six API-001..004 drift classes on generated mutated
  copies (each caught) + clean tree passes; `check_contracts.sh` 32/32;
  `shellcheck --severity=warning` clean → verification: gate is
  deterministic/offline/secret-free/prod-free, uses the already-required
  PyYAML, fails loudly on empty extraction (no silent pass), fixtures
  generated from real files (cannot rot) → RESOLVED.
- **2026-09-29 — FW-003 IN PROGRESS (fix staged, bench verification
  pending owner).** Finding ID → commit `5790753`
  (`firmware/src/network.cpp` only, +28/−6).
  - Reproduction/root cause (firsthand, static — deterministic path):
    `networkMaintenanceTask` (`network.cpp:562-583`, every 250 ms) called
    `refreshLocation()` on every iteration while `locationResolved` false;
    with no NVS cache that runs `WiFi.scanNetworks()` + BeaconDB HTTPS
    POST + ipinfo fallback (`network.cpp:472-549`) at ~4 Hz.
    `LOCATION_RETRY_INTERVAL_MS` (`config.h:126`) was defined but
    unreferenced; `lastLocRetry` (`firmware.ino:121`) was write-only
    (`network.cpp:576`, since removed); `HTTP_TIMEOUT_MS` (`config.h:119`)
    was defined but unused. Trigger condition: WiFi connected + no
    provisioned/NVS location + geolocation failing (fresh nodes, cleared
    NVS, no BeaconDB coverage + ipinfo failing). Consequences confirmed by
    reading: API spam, maintenance-task stall (Wi-Fi/NTP recovery held),
    power waste; sensor task unaffected (separate task/core) but shared
    I2C/bus contention possible during scans.
  - Remediation: in-function gate — NVS cache path stays ungated
    (identical boot/reconnect timing with cache); network stages gated to
    60 s via the project's own fixed-interval-throttle convention
    (cf. MQTT 5 s, Wi-Fi 30 s); first network attempt per boot immediate
    (`lastLocRetry==0` sentinel); attempts unbounded (recovery preserved);
    `http.setTimeout(HTTP_TIMEOUT_MS)` on both location HTTP calls;
    stale "cheap NVS read" comment corrected; no new cross-task sharing
    (gate variable lives in the one maintenance task); `prefs.end()`
    before the early return (no Preferences leak).
  - Before/after: before — up to 1 scan + 2 HTTPS per 250 ms while
    unresolved; after — same immediate first attempt, then ≤1 network
    attempt per 60 s, each bounded to 3 s; cache/provisioned behavior
    byte-identical timing.
  - Tests: baseline `pio run` SUCCESS (pio 6.2.0, esp32dev) at
    `7556570`-tree; post-fix `pio run` SUCCESS (22 s); `canonical-host-test.sh`
    canonical + onset suites pass before and after; gate-arithmetic model
    (32-bit `millis()` wrap incl. boundary/exact/wrap cases) 7/7
    verified in Python; `secrets.h` generated from `.example` (gitignored,
    placeholder values, never printed), `check-secrets.sh` preflight pass.
  - Regression mapping: sensor.cpp/onset/mqtt.cpp/portal/NVS-write paths
    untouched; WDT still reset every 250 ms iteration; strictly less
    blocking in the maintenance task. On-device sensor-rate, heartbeat,
    MQTT, Wi-Fi, watchdog behavior NOT observable here (no bench hardware
    in this environment — no /dev/ttyUSB*, no ESP32 on lsusb).
  - Status: IN PROGRESS, not RESOLVED — per the ledger rule, FW-003
    requires actual bench evidence (serial-observed retry cadence +
    heartbeat/MQTT/watchdog sanity on device). Owner bench procedure:
    build at `5790753`, flash dev node, observe `No location in NVS`
    cadence go 250 ms → 60 s, confirm heartbeat/MQTT/watchdog nominal.

## Audit decisions (D-AUDIT-xxx)

- **D-AUDIT-001 — Ledger location.** No repository finding ledger existed
  (only track-scoped `09-findings.md` / research ledgers). Created this
  file at `docs/audit/FINDING_LEDGER.md` per the audit record requirement.
  Creating it is the mandated exception to "do not automatically add files".
- **D-AUDIT-002 — UI boundary.** Zero UI files inspected-for-change; all
  AUI findings are report-only for manual owner implementation. Confirmed
  by `git status` (no `android/**/ui/**`, layout, res, or theme file
  touched).
- **D-AUDIT-003 — Change bar.** Only SRV-001 (test-only, REQUIRED) and
  OSS-001 (docs-only, REQUIRED) met ALL of: firsthand-verified, REQUIRED,
  localized, reversible, test-covered, non-UI, unrelated to prod
  credentials/architecture. Everything else documented, not changed.
- **D-AUDIT-004 — Contract freeze.** API-001..010 touch public contracts
  (`contracts/` is top of the authority hierarchy; PROJECT_RULES §9
  requires a decision before behavior/contract changes). Reported with
  evidence; CIT-002's route-vs-path diff gate is the structural fix.
- **D-AUDIT-005 — Deploy/prod-credential freeze.** MQ-007, DEP-001/002/004
  and all `deploy/` + rotation-procedure changes deferred to explicit
  owner approval. No production system accessed or modified.
- **D-AUDIT-006 — Firmware/hardware freeze.** FW-001..011 require bench
  hardware to verify; alert-path-adjacent FW items additionally require
  owner safety review. Recorded with file:line for bench scheduling.
- **D-AUDIT-007 — Chat verdict.** Chat EXISTS (design doc, API, store,
  migrations, WS fanout, Android UI); therefore no CHAT-XXX no-chat
  record. Moderation posture recorded as CHT-001..004; current
  non-authoritative positioning is disclosed and intentional.
- **D-AUDIT-008 — No-push rule.** Audit commits stay local until the owner
  reviews; no push, no force-push, no history rewrite (verified:
  fast-forward-only local commits on `main`).
- **D-AUDIT-009 — Priority mapping.** P0 = release-blocking open defect
  (none: no evidence of wrong/missed alerts from code as written).
  P1 = open REQUIRED. P2 = STRONGLY RECOMMENDED. P3 = OPTIONAL.
  INTENTIONAL / NOT APPLICABLE stay visible to prevent re-flagging.
- **D-AUDIT-010 — Reconciliation rationale (2026-09-29 batch).** For each
  API finding the direction was chosen from evidence, not by default:
  API-001 option B (contract stale: handler+Android+behavior agree);
  API-002 implementation-is-truth (handler+Android+CLIENT_SPEC agree);
  API-003 contract-gain expressiveness (semantics untouched);
  API-004 document (official operator endpoint: CLIENT_SPEC §5.8,
  `deploy/scripts/test-alert.sh`, sibling admin paths documented —
  documenting changes no exposure); API-006 validate-at-edge (contract
  already `format:uuid`; real store already cast). D-AUDIT-004's freeze is
  superseded for exactly these six findings by this batch's explicit scope.
- **D-AUDIT-011 — Gate design (2026-09-29 batch).** Python (not a Go test)
  because the contract is YAML and PyYAML is already a gate prerequisite
  (`check_contracts.sh` §2); a Go gate would add a YAML dependency to
  `go.mod` for tooling. Structured extraction (method+path pairs,
  brace-balanced structs, must-find assertions) instead of bare grep.
  Selftest fixtures are generated from the real files per run, so they
  cannot rot independently. Extend via `DTO_TABLE`, not new logic.
- **D-AUDIT-012 — Existing-test conflict (2026-09-29 batch).**
  `TestCreateChatMessage_PersistsThenBroadcasts` used `"c-1"` as
  `client_message_id`; the contract (`format:uuid`), the real store
  (`NULLIF($6,'')::uuid`), and Android (always UUID) all disagree with it.
  Per PROJECT_RULES §5 (contracts > tests) the fixture was the defect;
  corrected to the contract's example UUID with an explanatory comment —
  a test correction to the authoritative contract, not a weakening.
- **D-AUDIT-013 — Bench limitation (2026-09-29 firmware batch).** This
  environment is a cloud VM with no ESP32 attached (no /dev/ttyUSB*,
  lsusb shows QEMU devices only) and no preinstalled PlatformIO
  (6.2.0 installed into a venv for build checks; prior bench used
  6.1.19 — recorded version skew). Therefore: static-firsthand
  confirmation + `pio run` build + host tests + arithmetic model are the
  available verification; flash/serial/observation are impossible here.
  FW-003 stays IN PROGRESS until owner bench evidence lands; no firmware
  finding is marked RESOLVED without device observation.
- **D-AUDIT-014 — Related-firmware adjudication (2026-09-29 batch).**
  FW-001 → B (dead-sensor safe state changes reliability behavior; needs
  MPU-fail bench observation). FW-002 → B (locking the mqtt/status paths
  risks deadlock; needs bench; FW-003 added no new cross-task sharing).
  FW-004 → partially absorbed (timeout half applied under FW-003's
  recorded remediation; allocation bounding + on-device validation remain
  → still OPEN, remainder is B). FW-005 → B (provisioning-acceptance
  change needs bench provisioning test). FW-008 → B (loop-timing change
  needs heartbeat/alert-latency observation). All remain OPEN; none
  implemented merely for proximity.

## Open-item index (as of 2026-09-29)

- **P1 (open REQUIRED):** FW-003, DOC-002, API-001, API-002, API-003,
  API-004, API-006.
- **P2 (STRONGLY RECOMMENDED):** AND-001, AND-002, APE-001, SRV-002,
  SRV-003, SRV-004, FW-001, FW-002, FW-004, FW-005, FW-008, FW-011,
  DB-001, DB-002, MQ-007, MQ-008, API-005, API-007, API-009, API-010,
  CHT-001, CHT-002, CHT-003, CIT-001, CIT-002, CIT-004, CIT-006,
  OSS-002, DOC-001, DOC-003, DEP-001.

### Amendment 2026-09-29 — remediation batch (API-001/002/003/004/006, CIT-002)

- **P1 now RESOLVED:** API-001, API-002, API-003, API-004, API-006
  (remediation log above). **P1 remaining OPEN:** FW-003, DOC-002.
- **P2 now RESOLVED:** CIT-002. **P2 remaining OPEN:** all others listed
  above except CIT-002.
- Original entries above are preserved verbatim per the append-only rule;
  this amendment is the status transition record. No new findings were
  created during remediation (one validator typo caught its own test
  fixture mid-work; no ledger impact).

### Amendment 2026-09-29 — firmware batch (FW-003 + related)

- **FW-003: OPEN → IN PROGRESS** (fix `5790753` staged; build + host-test
  verified; bench evidence pending owner — see remediation log).
- **FW-001, FW-002, FW-005, FW-008: remain OPEN** (all → B: separate bench
  experiment required; reasons in D-AUDIT-014).
- **FW-004: remains OPEN** (timeout half applied under FW-003; allocation
  bounding + on-device validation remain).
- **P1 remaining OPEN:** FW-003 (IN PROGRESS), DOC-002.
- **P3 (OPTIONAL):** AND-003, AND-004, AND-005, AND-006, ASE-004, SRV-006,
  SRV-007, SRV-008, SRV-010, SRV-011, SRV-012, DB-003, DB-004, DB-005,
  MQ-002, MQ-003, MQ-004, MQ-005, MQ-006, MQ-010, FW-006, FW-007, FW-009,
  FW-010, FW-012, DEP-002, DEP-004, DEP-005, API-008, CHT-004, CIT-003,
  OSS-003, OSS-004, OSS-005, OSS-006.
- **Intentional / N/A (visible, no action):** AND-007, ASE-001, ASE-002,
  ASE-003, APE-002, APE-003, AUI-001..005 (informational), SRV-005,
  SRV-009, DB-006, DB-007, MQ-001, MQ-009, MQ-011, DEP-003, DEP-006,
  CIT-005.
