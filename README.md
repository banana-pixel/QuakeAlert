# QuakeAlert

QuakeAlert is a single-node earthquake early-warning (EEW) system: an ESP32
seismic sensor detects ground shaking, a Go server confirms and tracks the
event, and an Android app raises the warning. It is built to buy seconds of
warning before strong shaking arrives at a phone.

> **Safety status.** This is an experimental, single-node system, not a
> certified public warning service. A single sensor cannot geolocate an
> earthquake or guarantee delivery, and the project treats real-earthquake
> validation as a post-release activity. Do not rely on it as your only source
> of earthquake warning. See [Scope and limits](#scope-and-limits).

> **License status.** No license has been chosen yet. Until one is added, the
> code is "all rights reserved" by default and is **not** yet open source in the
> legal sense. Choosing the license is an owner decision; see
> [License](#license).

## Repository layout

| Path | Component |
|---|---|
| `firmware/` | ESP32 sensor firmware (C++ / Arduino, PlatformIO). Comments are in Indonesian. |
| `server/` | Detection, confirmation, event lifecycle, REST + WebSocket, MQTT ingest (Go, module targets Go 1.24). |
| `android/` | Android client (Kotlin, minSdk 28). UI is Indonesian-first with English fallback. |
| `contracts/` | Source-of-truth contracts: MQTT payload schemas, OpenAPI, DB migrations, FCM payload. |
| `deploy/` | Production Docker Compose (Caddy, Mosquitto, PostGIS, Redis), env template, operator scripts. |
| `docs/` | Specifications, decisions log (`DECISIONS.md`), current state, and planning notes. |

## How it works

1. The ESP32 runs an STA/LTA detector on an MPU6050 accelerometer. On a
   confirmed onset it publishes to MQTT over TLS.
2. Each event produces **exactly two** signed publishes that share one
   observation sequence: `PRELIM` (the peak over the first ~1.0 s after
   confirmation) and `FINAL` (the full-event peak at detrigger). See
   `contracts/mqtt/trigger.schema.json`.
3. The server verifies the HMAC signature, tracks the event through its
   lifecycle (detected, unconfirmed, confirmed, resolved), and pushes alerts to
   clients over WebSocket and Firebase Cloud Messaging.
4. The Android app shows a full-screen alert and, where enabled, a trusted-local
   warning path for the node's own location.

Canonical units across every component: PGA in gal (cm/s^2), all timestamps in
milliseconds epoch UTC, distances in km.

## Build and run

Each component builds independently. The commands below are entry points; the
authoritative details live in `deploy/` and `docs/`.

### Server (Docker Compose, recommended)

```bash
cd deploy
cp .env.prod.example .env.prod   # then fill in every required secret
chmod 600 .env.prod
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

The stack terminates TLS at Caddy, runs schema migrations once via
`migrate/migrate`, and exposes only Caddy (80/443) and Mosquitto (8883). A local
development stack lives in `server/docker-compose.yml`.

Run the Go tests:

```bash
cd server
go test ./...
```

### Firmware (PlatformIO)

```bash
cd firmware
cp src/secrets.h.example src/secrets.h   # fill in Wi-Fi, MQTT, HMAC key
./scripts/check-secrets.sh                # preflight before flashing
pio run                                    # build
# pio run -t upload                        # flash (requires connected hardware)
```

### Android (Gradle)

```bash
cd android
./gradlew assembleDebug
```

A Firebase `google-services.json` is required for FCM and is not committed.

## Configuration and domain

Production runs at `quakealert.web.id`: `api.quakealert.web.id` for the REST API
and `broker.quakealert.web.id` for MQTT. Server, firmware, and client defaults
all point at this domain. Every secret (database password, master key, JWT
secret, admin key, MQTT passwords) is supplied through the environment and is
never committed; see `deploy/.env.prod.example` for the full list.

## Scope and limits

- **Single node.** One sensor cannot triangulate an epicenter. Confirmation is
  designed around a single trusted location, not a network.
- **Best-effort delivery.** MQTT publishes are QoS 0; resilience comes from
  firmware retries and server-side deduplication, not from broker guarantees.
- **Not validated against a real earthquake in this repo's history.** Field
  validation is planned for after release. "Waiting for a real earthquake" is
  not treated as a release blocker, but it does bound what can be claimed.
- **No safety certification.** Treat warnings as supplementary.

## Contributing, security, and conduct

- Contribution workflow: [`CONTRIBUTING.md`](CONTRIBUTING.md)
- Reporting a vulnerability: [`SECURITY.md`](SECURITY.md)
- Community expectations: [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md)

## License

No `LICENSE` file exists yet, so the default of "all rights reserved" applies
and the project is not yet legally open source. The maintainer needs to choose a
license before public release. A recommendation and its rationale are recorded
in `docs/DECISIONS.md` (see the D-042 entry). Once the maintainer ratifies a
choice, add the corresponding `LICENSE` file and update this section.

## Project

- Repository: https://github.com/banana-pixel/QuakeAlert
- Maintainer: [banana-pixel](https://github.com/banana-pixel)
- Support the project: https://saweria.co/bananapixel
