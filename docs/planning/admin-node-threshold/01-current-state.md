# 01 — Current state (from source, 2026-09-11)

- Purpose: record what the code DOES today so threshold research never confuses policy (the number) with mechanism (the gates). All claims carry `file:line`.
- Status: drafted from source read 2026-09-11; needs owner ack that it matches deployed `8e93c4d` + schema 10.
- Last updated: 2026-09-11 UTC.
- Sources: files listed in §12; governance D-003/D-006/D-007/D-009/D-012/D-013/D-018/D-019/D-036 + D-037 edge (code; no separate DECISIONS entry found — GAP noted §11).

## 1. Where the Admin threshold is defined

- `server/internal/event/admin_node.go:30`: `AdminNodeMinPGAGal = 140.0` (gal, compile-time const, D-007 rationale in comments `admin_node.go:18-24`).
- `server/internal/event/admin_node.go:36`: `AdminNodeHeartbeatMaxAge = 5 * time.Minute`.
- `server/internal/dispatch/event_frame.go:65`: `TrustedLocalRadiusKm = 20` (separate const, intentionally ≠ `AlertRadiusKm = 200`, `dispatcher.go:74`).
- Production value is 140.0. The "100 gal" candidate exists NOWHERE in production code (verified by grep 2026-09-11).

## 2. Admin eligibility evaluation

- Pure function `EvaluateAdminNodeEligibility(state, snapshot)` (`admin_node.go:96-119`). Gate order (log-stable): designated → verified → `To == UNCONFIRMED` → admin contribution present → contribution `Phase == FINAL` → contribution `PeakPGA >= 140.0` (inclusive; `<` rejects) → heartbeat age ≤ 5 min. Else `Eligible:true / ADMIN_ELIGIBLE`.
- Closed reason vocab `admin_node.go:42-50`: `ADMIN_ELIGIBLE, NO_ADMIN_DESIGNATED, ADMIN_UNVERIFIED, NOT_UNCONFIRMED, NO_ADMIN_CONTRIBUTION, ADMIN_NOT_FINAL, ADMIN_PGA_BELOW_FLOOR, ADMIN_HEARTBEAT_STALE`.
- Contribution lookup `adminContribution` (`admin_node.go:124-131`): first `Evidence.Contributors[]` with `NodeID == state.StationID`. One node = one contributor (PRELIM+FINAL same entry). The PGA compared is the ADMIN's own contribution peak, NOT event peak (`admin_node.go:26-30` comment).
- Frame constructor `TrustedLocalFrameFor` (`admin_node.go:140-162`): ONLY choke point; returns `EARTHQUAKE_ALERT + TrustedLocal:true`, same event_id/revision/timestamp/state=UNCONFIRMED as the snapshot — no new state, no revision bump, no new FCM enum.
- Async emission `emitTrustedLocal` (`admin_node.go:176-197`): only for `To==UNCONFIRMED`; nil hook = inert; store read in own goroutine with 2 s timeout (`adminNodeLoadTimeout`); fail-closed (no status → no frame); slow DB delays ONLY the local frame, never normal emission (S1).
- Hook wiring `emit.go:20-55,62-75`: `Bridge.SetAdminNodeHook(src, sink)`; `EmitTransition` emits normal frame first, then admin hook additively.

## 3. PRELIM semantics

- Firmware: STA/LTA trigger (`sensor.cpp:190-201`: ratio ≥ 4.0 AND STA ≥ 1.5 gal) → hold 300 ms (`CONFIRMATION_DURATION_MS`, `sensor.cpp:203-247`) → `eventInProgress=true`, `pga=correctedMagnitude` (instantaneous peak-so-far), PRELIM slot filled (`pendingPrelim.maxPga=pga`, duration=elapsed/1000, onset, obs_seq=`bootCount<<16|inBootSeq`, detrigger=0). Published via `servePendingSlot(pendingPrelim,false)` in `firmware.ino:367-383` (PRELIM first — most warning value), QoS 0 + retry (5 s backoff, 36 attempts, 300 s max age).
- Server: `upsertContributorLocked` (`tracker.go:699-740`): new contributor `Phase=PRELIM, PeakPGA=in.PGA`; existing: PGA only rises (`in.PGA > PeakPGA`), `Phase=FINAL` sticks once set, onset NEVER moves (first-bound-wins). `classify` (`classify.go:10-21`): `peakPGA < 16.6 → DETECTED` else UNCONFIRMED/CONFIRMED by quorum. So a PRELIM ≥ 16.6 creates `DETECTED→UNCONFIRMED FLOOR_MET` + advisory WS frame — but admin gate returns `ADMIN_NOT_FINAL`. PRELIM with high PGA does NOT warn locally by design.
- PRELIM PGA is peak-SO-FAR at confirmation (≈ hundreds of ms after onset), systematically ≤ FINAL peak. 100 gal is NOT comparable between phases (§4 file).

## 4. FINAL semantics

- Firmware: detrigger when `ratio < 1.5` or 60 s timeout (`sensor.cpp:249-287`); `pendingReport.maxPga = max over whole event`, duration full, detrigger TS set; same publish/retry path as PRELIM.
- Server absorb: same `upsertContributorLocked`; PGA ratchets up; Phase flips to FINAL. Two outcomes:
  - (a) Transition case: event was DETECTED (e.g., PRELIM below 16.6, or FINAL is first sighting incl. lost-PRELIM QoS0 case) → `transitionLocked` → `DETECTED→UNCONFIRMED` → normal `EmitTransition` → admin hook evaluated on that snapshot (eligible if FINAL ≥ 140).
  - (b) Edge case (D-037): event ALREADY UNCONFIRMED → `UNCONFIRMED→UNCONFIRMED` illegal → NO transition/revision/log row → `tracker.go:407-427` records `localPending` snapshot (exactly-once per event via `adminEdgeFired`, in-memory only) → `publish()` drains BEFORE the empty-transition early-return (`tracker.go:213-235`) to `Bridge.EmitAdminEdge` (`emit.go:81-86`) → same `emitTrustedLocal` evaluator. Later FINALs/duplicates/retries are not edges (`wasFinal=true`).
- `adminEdgeFired` (`event.go:198-206`): not state/revision/log, not persisted/reconciled/replay-compared; restart loses it (safe: restart triggers no new absorb; client revision dedup absorbs residual).

## 5. PGA calculation path (firmware → server)

- Sensor: MPU6050 FS_4 (±4 g), DLPF BW_5, `setRate(99)`, DMP FIFO, GPIO15 RISING ISR → `sensorTask` (Core 0) → `processSensorData` (`sensor.cpp:95-288`).
- Per sample: `vectorMagnitude = sqrt(x²+y²+z²) * DATA_RATIO` (gal; `DATA_RATIO=980.665/8192≈0.1197`, `config.h:110`); `baselineEMA α=0.005` (~200-sample gravity/DC tracker); `corrected=|vector-baseline|` (ALL downstream uses this); `STA α=1/50` (~0.5 s @100 Hz), `LTA α=1/2000` (~20 s @100 Hz, frozen during event); `LTA clamp 0.5 gal`; `ratio=STA/LTA`.
- Gates: warmup 45 s (`LTA_WARMUP_TIME_MS`), cooldown 60 s between events, `MIN_STA 1.5 gal`, trigger 4.0 / detrigger 1.5, confirm 300 ms, cap 60 s.
- Wire: PGA float32 → 4-decimal JSON (`mqtt.cpp:152-163`), canonical HMAC string, `onset_ts` from `millis()`→epoch at publish (`firmware.ino:300-308`, so late NTP still correct), `dur_ms`, `obs_seq` shared PRELIM/FINAL, `attempt_no` 1-based.
- Server: `Input.PGA` float64 (`input.go`), contributor `PeakPGA` = max absorbed; event `peakPGA()` = max over contributors (`event.go:259-270`); MMI via Wald `3.66*log10(PGA)-1.66` (`centroid.go:80-89`); labels: <16.6 light, <137.2 moderate, ≥137.2 strong (`centroid.go:97-106`). NOTE: 140 gal sits just above the "strong" boundary 137.2 gal.
- Quantization ≈ 0.12 gal/LSB + float32/4-decimal rounding (negligible at 100+ gal); clipping ±4 g ≈ ±3923 gal (far above candidate range — no clipping concern for 60–160 gal).

## 6. Event lifecycle / dispatch / Android / replay / tests

- Lifecycle D-003 (`event.go:75-117`): DETECTED→UNCONFIRMED→CONFIRMED→RESOLVED/CANCELLED; downgrade illegal; revision++ only on transition; D-037 is the ONLY non-transition emission and changes none of this.
- Normal dispatch: `FrameFor` (`emit.go:98-141`): UNCONFIRMED→ADVISORY WS-only no push (D-009); CONFIRMED→ALERT WS+FCM; RESOLVED/CANCELLED→all-clear only if EverConfirmed. `dispatchFCM` (`dispatcher.go:221-297`): severe (MMI≥VII or PGA≥250, `severity.go`) → GeoTopic no distance filter; else tokens within 200 km; else GeoTopic fallback — UNLESS single-node guard (`NodeCount≤1` → no GeoTopic ever, `dispatcher.go:315-317`).
- Trusted-local dispatch (`event_frame.go:82-138`): WS broadcast identical + FCM ONLY to tokens within 20 km (`trustedLocalTokens`→`FCMTokensWithin`), NO GeoTopic fallback in ANY case (severe/guard-off/no-tokens), no-token → `AudienceNone` observed-zero; validity defaulted from resolveAfter; audit line `trusted-local emitted ADMIN_ELIGIBLE` with event_id+outcome only (D-019, no coords).
- D-018 validity: `ValidityMs` duration on every frame (`ws.go:70-75`, `fcm.go:77-83`, `event_frame.go:34-36,86-88`); absent = legacy 15-min window, never expired; expired → never raised (client-enforced).
- D-019 audit: server one line per local emission; client `raiseAlert/raise/gated-out/notify` one line with event_id+outcome, never position (`DECISIONS.md` D-019).
- Android: `FcmAlertMapper.kt:73` (`trusted_local=="true"` trimmed), `WsAlertMessageDto.kt:73` (default false), `AlertGate.decideFor` (`AlertGate.kt:148-166`) routes trusted-local → `decideLocalWarning` (distance-only vs `LOCAL_WARNING_RADIUS_KM=20`, NO severe override — single-station authority must not claim network confirmation; unknown position fails open). `WarningActivity` `EXTRA_IS_LOCAL_WARNING` → "LOCAL WARNING" pill + local copy; validity/dedup/logging per event_id apply unchanged.
- Replay D-013 (`replay.go`, `replay_admin_test.go`): read-only two SELECTs, canonical order `received_ts,observation_id`, grouping bijection F2, delta F3, operator-asserted params, `INDEPENDENCE_CELL_KM` contradiction rejected; admin evaluator replayed via operator-asserted profile; historical `trusted_local` never compared; `adminEdgeFired` not compared. Group by `algo_ver` (V5).
- Tests today: `admin_node_test.go` (gate matrix incl. 140.0 eligible / 139.999 reject); `admin_edge_test.go` (exactly-one, revision-no-bump, concurrent-FINAL collapse, no-edge cases); `replay_admin_test.go` (140 + 5-min boundaries, unverify); `trusted_local_test.go` (no GeoTopic even above Severe 250 gal/MMI VIII; no-token → AudienceNone); Android `LocalWarningGateTest`, `TrustedLocalMapperTest`, `TrustedLocalRaiseTest`; store `admin_node_test.go` (000010 up/down, single-active, unverify auto-clears `store.go:429-436`).

## 7. Governance / algo_ver

- D-007: PGA floor + quorum are compile-time (admin 140 follows it). D-006/V1-V7: decision meaning travels with `algo_ver`; bump when meaning changes; never rewrite history.
- Current `algo_ver = phase3-1.1/ic=<km>` (`store.go:26,71-73`) — encodes ONLY independence radius, NOT the admin floor. Changing 140→X changes the meaning of every future `ADMIN_ELIGIBLE` decision under an identical label → V3/V6 question OPEN: new decision must state whether the admin floor joins `algo_ver` (e.g., `/ic=5/admin=140`), a separate admin version, or why comparison stays valid. No threshold/quorum/radius change is approved today (`CURRENT_STATE.md` Active gate).
- D-036 status tension RECORDED (not resolved here): `DECISIONS.md` D-036 says "adopted, NOT activated" (3 preconditions unmet as of 2026-09-09) while `CURRENT_STATE.md:48-51` + `AUDIT.md` LIVE-EQ say binary `c85ddcf`/`8e93c4d` + migration 000010 DEPLOYED, sole holder `NODE-52960B47` designated ~2026-09-09T21:15Z, local path live (emission 95 `TOKENS_RADIUS_20KM`, prior `ADMIN_ELIGIBLE` 2026-09-11 12:40 FINAL 288.36 gal). D-037 (FINAL-edge) has NO separate DECISIONS entry found — code + commit message only. Both need governance closeout outside this research.

## 8. What must NOT change if we only evaluate threshold policy

Policy = the single float `140.0`. Mechanism that MUST stay byte-identical: gate ORDER + reason vocab; FINAL-only rule; admin-own-contribution rule; UNCONFIRMED-only rule; inclusive `>=` boundary semantics; 5-min heartbeat; 20 km radius + no-GeoTopic + token-only; WS+FCM shape + `trusted_local` additive/omitempty; validity/dedup/logging; revision-no-bump + exactly-once edge; single-active index + designate/revoke/unverify; CONFIRMED/advisory/normal dispatch; contracts; `algo_ver` base (unless the decision explicitly bumps it). Any benchmark MUST vary only the floor value in an offline copy of the evaluator — never the production const.

## 9. Observed production anchors (for replay §6)

- Event `004c5f65` (2026-09-11 12:40Z): PRELIM 1.13 gal (obs 90) → FINAL 288.36 gal (obs 91, same obs_seq) → `ADMIN_ELIGIBLE` + `TOKENS_RADIUS_20KM` (24 attempted / 2 succeeded, 22 NotRegistered). Proves the live path fires above 140.
- Event `4fcc3374` (P4-M6): 77.8888 gal, 1 node/1 cell, RESOLVED rev2 — below admin floor, no local frame (useful negative).
- LIVE-EQ M5.9 deep 2026-09-11 21:23Z: ZERO rows 20:00–22:00 UTC (no trigger at any PGA) — threshold-irrelevant, filed as no-evidence denominator (S5).
- 2026-09-01 SENSOR event: onset→decided 6166 ms vs `dur_ms` 6138 (server share 28 ms, 17 ms transit) — proves onset→decided structurally contains the shake when PRELIM < 16.6 floor.

## 10. Unknowns carried forward

GPIO15/DMP true rate (assumed 100 Hz, `setRate(99)` — UNMEASURED); mounting/site transfer; per-disturbance gal mapping (§5 file); heartbeat continuity through any quake window (no history table).

## 11. GAP: D-037 governance

D-037 FINAL-edge exception is implemented (`tracker.go`, `emit.go`, `event.go`) and deployed (`8e93c4d "feat(d-037)"`) but no `docs/DECISIONS.md` D-037 entry was found 2026-09-11. Recommendation (not decision): owner records D-037 (scope: explicit D-003 exception, exactly-once, no state/revision/log) or marks the commit PROPOSED. Research proceeds on code-as-deployed.

## 12. File:line index (spot-checkable)

Threshold `event/admin_node.go:30,112`; gates `admin_node.go:96-119`; frame `admin_node.go:140-162`; async `admin_node.go:176-197`; hook `event/emit.go:20-86`; edge detect `event/tracker.go:407-427`, drain `tracker.go:213-235`; edge flag `event/event.go:198-206`; classify `event/classify.go:10-21`; upsert `event/tracker.go:699-740`; lifecycle `event/event.go:75-117`; snapshot `event/snapshot.go:87-120`; algo_ver `event/store.go:26,71-73`; radius `dispatch/event_frame.go:60-65,82-138`; normal FCM `dispatch/dispatcher.go:221-317`; severe `dispatch/severity.go:13-31`; validity `dispatch/ws.go:70-81`, `dispatch/fcm.go:77-90`; designate `store/admin_node.go:49-108`, status `store/admin_node.go:146-168`, unverify-clear `store/store.go:429-436`; firmware trigger `firmware/src/sensor.cpp:141-287`, PRELIM `sensor.cpp:209-247`, FINAL `sensor.cpp:249-287`, publish `firmware/src/firmware.ino:260-383`, retry/epoch `firmware.ino:297-308`; MMI `consensus/centroid.go:80-106`; floors `consensus/engine.go:19-26`; Android gate `AlertGate.kt:138-203`, radius `SafetyPolicy.kt:37-44,60`, mapper `FcmAlertMapper.kt:73`, `WsAlertMessageDto.kt:73`.
