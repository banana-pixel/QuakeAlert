# Final Paho v1.5.1 licensing decision audit (READ-ONLY record — nothing changed)

- Date: 2026-09-27 UTC. Status: RESEARCH RECORD, decides nothing in the repository.
- Question: can `github.com/eclipse-paho/paho.mqtt.golang v1.5.1` remain in QuakeAlert Server if the server is distributed under AGPLv3?
- Prior findings reused (not repeated): MPU6050 MIT CLEAR; ESP32 Arduino Core LGPL-2.1 CLEAR-with-artifacts; Android original code CLEAR; no ntfy/GPL/copyleft code in-tree.
- Authority note: planning-area record only; does not override `/contracts`, `PROJECT_RULES.md`, `docs/DECISIONS.md` (PROJECT_RULES §5).

## 1. Clearly documented AGPLv3 path? NO

No path in the v1.5.1 distribution documents allows an AGPLv3 work to convey statically-linked Paho code without further review:

- The EPL-2.0 Secondary License mechanism (§3.2(a)) requires the Initial Contributor to have **attached** the Exhibit A notice. A 101-file tree grep of the exact v1.5.1 distribution finds "Exhibit A" **only** in `LICENSE` and `epl-v20` (the license template itself); **zero `.go` files** carry the notice. The license text itself warns that including the template is not sufficient. Condition §3.2(a)(ii) therefore fails on the evidence.
- EPL-2.0 defines "Secondary License" as "the GNU General Public License, Version 2.0, or any later versions of that license." AGPLv3 ("GNU Affero General Public License") is a distinct license, not textually "a later version of the GPL" — and the text does not resolve whether it qualifies. Unresolved on its face, and moot given the missing Exhibit A.
- The EDL-1.0 dual-license alternative (permissive, BSD-style) is real text in every file header, but whether a downstream combiner may validly elect EDL-1.0 alone for a binary conveyed inside an AGPLv3 program is a legal determination, not an engineering reading. Not assumed either way.

## 2. Relied-upon path (if YES)

Not applicable — no YES path established.

## 3. Unresolved point + replacement assessment

The single unresolved point: **whether the EDL-1.0 election is valid for conveying Paho v1.5.1 inside the AGPLv3 server binary.** Everything else is settled against AGPL compatibility: direct static import (`server/internal/ingest/subscriber.go:9`, `server/cmd/quakealert/main.go:18`) makes this a combined-work (not mere MQTT network communication with Mosquitto, which is unproblematic), EPL-2.0 §3.1(b)(iv) conflicts with AGPLv3 §5 whole-work terms on conveyance, and the secondary-license escape is doubly closed (no Exhibit A; AGPL scope unresolved). If counsel rules the EDL-1.0 election valid, Paho stays with no code change. If not, replacing Paho with a permissively-licensed MQTT client (criterion: OSI-approved permissive license + MQTT 3.1.1 parity; swap localized behind the existing `TriggerVerifier`/`TriggerHandler` interfaces) is the clean engineering path. **Paho NOT replaced by this audit.**

## 4. Final status

**NEEDS LEGAL REVIEW**

- Exact license: EPL-2.0, dual EPL-2.0/EDL-1.0 (v1.5.1; no NOTICE file).
- Exhibit A present: NO (template only; 101-file tree grep).
- Secondary License applies: NO.
- AGPLv3 compatibility: NOT ESTABLISHED for the as-linked conveyed binary.
- Repository change required: none unless counsel rules adversely (then: client swap).
- Commercial hardware impact: none (server-side only).
- Hosted-service impact: the AGPL server decision stays gated on this ruling.

## Evidence pointers (all observed, none created here)

- v1.5.1 `client.go` header (EPL/EDL dual grant, no Exhibit A); upstream `NOTICE` 404; tree grep outputs; EPL-2.0 §3.2(a)/§3.1(b)(iv)/Exhibit A/self-sufficiency warning texts; QuakeAlert import sites above.
- Related records: licensing strategy audit + blocker verification (prior turns); `docs/planning/admin-node-threshold/` R-series and `docs/DECISIONS.md` untouched by this file.

## Implementation: Paho replaced (owner-ordered, this session)

- Paho replacement IMPLEMENTED; paho EPL-2.0 fully removed from the server module.
- Selected replacement: `github.com/256dpi/gomqtt` **v0.14.4** (pinned), **Apache-2.0** (repo label + full LICENSE text verified upstream; MQTT 3.1.1 core purpose; versions v0.1.0→v0.14.4; CI present). Transitives compiled in: `gopkg.in/tomb.v2` (BSD-style, license text verified from module zip), `github.com/gorilla/websocket` (BSD, pre-existing dep); `jpillora/backoff` (MIT) only in the unused broker/service path.
- Implementation commit: `Replace Paho MQTT client with permissive Go client` (this session; no push).
- Files: NEW `server/internal/ingest/mqttclient.go` (local Message/Handler/Client abstraction + gomqtt adapter + reconnect supervisor) and `mqttclient_test.go` (7 focused tests); EDITED `subscriber.go` (type swap only), `subscriber_test.go` (fake assertion), `cmd/quakealert/main.go` (Dial wiring, identical option values), 4× `server/scripts/sim_*.go` (mechanical connect/publish conversion), `go.mod`/`go.sum` (paho out, gomqtt+tomb in).
- Behavior preserved: topics, QoS 1, CleanSession=false, KeepAlive 30s, ConnectTimeout=IOTimeout, TLS system-CA TLS1.2+, auto-reconnect with resubscribe, connection-lost log line, Connect-wait-error, IsConnected health, Disconnect(250) quiesce. Documented deltas: bounded subscribe/connect waits (server IO discipline), IsConnected false during outage (honest health), serial callback dispatch (handler is fast/sync).
- Tests passed: `gofmt` clean; `go build ./...` OK; `go vet ./...` clean; sim files vetted individually; full `go test ./...` green except pre-existing flaky `TestBlockedLedgerDoesNotDelayDispatch` on 1-vCPU (proven unrelated: fails intermittently with and without this change, passes at GOMAXPROCS=8, touches untouched packages); `event`/`config`/`ingest` `-race` green.
- Production deployment NOT performed. No known functional regression.
