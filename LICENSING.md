# QuakeAlert licensing map

This file defines which license applies to which part of this repository.
It is the authoritative scope reference. In case of conflict between this
map and a per-directory `LICENSE` file, the per-directory `LICENSE` file
controls for files inside that directory.

Copyright in QuakeAlert-owned files is held by the QuakeAlert contributors
unless a file states otherwise.

## 1. Firmware — GNU GPLv3 (`firmware/LICENSE`)

Everything under `firmware/` is licensed under the GNU General Public
License v3.0 or later (SPDX: `GPL-3.0-or-later`):

- `firmware/src/` — ESP32 firmware sources (`.ino`, `.cpp`, `.h`)
- `firmware/test/` — host-side tests
- `firmware/scripts/` — firmware tooling (shell scripts)
- `firmware/platformio.ini`, `firmware/AUDIT.md`, and other files under
  `firmware/` — covered by the same license via this directory scope

Full text: `firmware/LICENSE`.

## 2. Server — GNU AGPLv3 (`server/LICENSE`)

Everything under `server/` is licensed under the GNU Affero General Public
License v3.0 or later (SPDX: `AGPL-3.0-or-later`):

- `server/cmd/`, `server/internal/` — server application sources and tests
- `server/scripts/` — server tools and simulators (`.go`, `.sh`, `.sql`)
- `server/migrations/` — schema migration placeholders
- `server/Dockerfile`, `server/docker-compose.yml`, `server/.dockerignore`

`deploy/` (production deployment configs and operator scripts for the
server stack) is likewise covered by AGPL-3.0-or-later as part of the
server distribution. Secrets, `.env` files, and provisioned certificates
are operational data, never part of the distribution.

Full text: `server/LICENSE`.

AGPLv3 §13 applies: anyone running a modified server version for users
interacting remotely over a network must offer those users the
Corresponding Source.

## 3. Android — license pending (none assigned)

`android/` is **not** covered by either license above. No Android license
has been established yet. All rights reserved pending the owner's
decision. Do not assume GPLv3/AGPLv3 applies to `android/`.

## 4. Documentation and interface contracts — license pending

`docs/`, `contracts/`, the root `*.md` files, `.github/`, and other
repository meta/tooling (including `run_e2e_test.sh` and session
transcripts) are **not** covered by the firmware/server licenses. Their
license is to be established separately. All rights reserved pending
that decision.

## 5. Trademark — separate policy, not a copyright license

The QuakeAlert name, logo, and "Official QuakeAlert Node" designation are
governed by `TRADEMARK.md`, which is independent of the copyright
licenses above. Trademark permission is never implied by copyright
licensing.

## 6. Third-party software — not ours, not relicensed

Dependencies listed in `THIRD_PARTY_NOTICES.md` remain under their own
licenses and are **not** QuakeAlert-owned. Nothing in this repository
relicenses third-party code. See that file for attribution and for the
LGPL-2.1 obligations attached to the ESP32 Arduino Core.
