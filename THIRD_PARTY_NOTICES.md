# Third-party notices

QuakeAlert incorporates and depends on third-party software. That software
remains the property of its respective owners and is **not** covered by the
QuakeAlert firmware (GPLv3) or server (AGPLv3) licenses. Nothing here
relicenses any third-party work. No third-party source code is vendored in
this repository for attribution purposes; the notices below identify each
dependency, its license, and where the full text lives upstream.

Coverage corresponds to the verified set in
`docs/planning/licensing/final-dependency-license-compatibility-check.md`.

## Firmware dependencies (GPLv3-compatible)

- PubSubClient 2.8 (knolleary) — MQTT client library for Arduino.
  License: MIT. Full text: upstream `LICENSE.txt`
  (https://github.com/knolleary/pubsubclient).
- ArduinoJson 6.21.5 (Benoit Blanchon) — JSON serialization for embedded C++.
  License: MIT (per upstream project licensing).
  (https://github.com/bblanchon/ArduinoJson, https://arduinojson.org/)
- MPU6050 1.4.3 (ElectronicCats) — IMU driver library.
  License: MIT — Copyright (c) 2019 ElectronicCats; full text upstream.
  (https://github.com/ElectronicCats/mpu6050)
- ESP32 Arduino Core (bench-proven 3.20017.241212 / platform 7.0.1;
  `platformio.ini` pinning still owed) — MCU support package.
  License: LGPL-2.1. Obligations on distributing Node firmware built on it:
  (a) LGPL §3 / GPLv3 compatibility holds for the combination;
  (b) every commercial/official Node shipment must include the §6 relink
  kit — QuakeAlert object files, build scripts, and a pinned Core source
  reference — plus GPLv3 Installation Information where the Node is a User
  Product; (c) no hardware-design disclosure is required.

## Server dependencies (AGPLv3-compatible)

- github.com/256dpi/gomqtt v0.14.4 — MQTT 3.1.1 client.
  License: Apache-2.0 (https://github.com/256dpi/gomqtt).
- github.com/256dpi/mercury v0.2.0 — support library used by gomqtt.
  License: MIT — Copyright (c) 2018 Joël Gähwiler.
- gopkg.in/tomb.v2 — goroutine lifecycle used by gomqtt.
  License: BSD-style (permissive).
- github.com/jpillora/backoff — backoff helper (gomqtt broker/service path).
  License: MIT.
- github.com/gorilla/websocket v1.5.3 — WebSocket transport.
  License: BSD-style — Copyright (c) 2013 The Gorilla WebSocket Authors.
- github.com/go-chi/chi/v5 v5.0.12 — HTTP router.
  License: MIT.
- github.com/jackc/pgx/v5 v5.5.5 (+ pgpassfile, pgservicefile, puddle) —
  PostgreSQL driver. License: MIT — Copyright (c) 2013-2021 Jack Christensen
  (family packages carry the corresponding Jack Christensen MIT notices).
- github.com/redis/go-redis/v9 v9.5.1 — Redis client.
  License: BSD-style — Copyright (c) 2013 The go-redis Authors.
- github.com/cespare/xxhash/v2 v2.2.0 — hashing (via go-redis).
  License: MIT — Copyright (c) 2016 Caleb Spare.
- github.com/dgryski/go-rendezvous — rendezvous hashing (via go-redis).
  License: MIT.
- golang.org/x/oauth2 v0.18.0; golang.org/x/crypto, /net, /sync, /text —
  Go extended libraries. License: BSD-style — Copyright The Go Authors.
- google.golang.org/protobuf v1.31.0; github.com/golang/protobuf v1.5.3 —
  Protocol Buffers runtime. License: BSD-style — Copyright The Go Authors.
- google.golang.org/appengine v1.6.7; cloud.google.com/go/compute/metadata —
  Google Cloud libraries. License: Apache-2.0.

Mosquitto, Caddy, PostgreSQL, and Redis are deploy-time infrastructure,
not distributed code; their own licenses apply to those programs.

## Android dependencies (app licensed GPLv3; see qualification below)

- AndroidX (core-ktx, lifecycle, activity, datastore), Jetpack Compose
  (BoM, ui, graphics, material3, tooling, google-fonts) — Apache-2.0.
- Kotlin stdlib, kotlinx-coroutines (+play-services bindings, test),
  kotlinx-serialization-json — Apache-2.0.
- OkHttp 4.12.0 (Square) — Apache-2.0.
- MapLibre Native Android SDK 13.5.0 — BSD-3-Clause (per upstream
  licensing); no access token or commercial key is embedded.
- JUnit 4.13.2 — EPL-1.0, test scope only (not linked into the APK).
  Espresso / androidx-test — Apache-2.0, test scope only.
- Android Gradle Plugin, Kotlin Gradle plugins, google-services plugin —
  build tools only (Apache-2.0), not distributed in the APK.
- Google Play Services linkage (owner-granted §7 exception, see
  `android/LICENSE-EXCEPTION`):
  `play-services-location:21.3.0`, `play-services-base:18.5.0`,
  `play-services-basement:18.4.0`, `play-services-tasks:18.2.0`,
  `play-services-cloud-messaging:17.2.0`, `play-services-stats:17.0.2` are
  distributed by Google under the proprietary Android SDK License (per
  their own POM declarations; the Firebase artifacts above are Apache-2.0
  and need no exception — the last two Play libraries arrive solely as
  Firebase transitives).
  `android/LICENSE-EXCEPTION` grants a GPLv3 §7 additional permission
  covering only the combination of the QuakeAlert code with exactly these
  six artifacts; it grants nothing in Google's code, waives none of
  Google's terms, excludes commingled files, and does not extend to any
  future proprietary dependency. Mitigating design (already in the code, no change
  made): location falls back to AOSP LocationManager without Play
  Services, and every Firebase entry point degrades to WebSocket-only
  when `google-services.json` is absent, so a fully-open build variant
  remains possible.
