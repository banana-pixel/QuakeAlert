# Contributing to QuakeAlert

Thank you for your interest. QuakeAlert is a life-safety earthquake
early-warning project, so correctness, honesty about what is verified, and
careful change control matter more than speed.

Before you start, note the per-directory license map in `LICENSING.md`
(firmware: GPL-3.0-or-later; server and deployment: AGPL-3.0-or-later;
Android: GPL-3.0-or-later with a linking exception; docs, contracts, and
repository meta: license pending, all rights reserved for now). Please
coordinate with the maintainer before reusing or redistributing code outside
those terms.

## Ground rules

- **Decisions are owner-governed.** Architectural and behavioural decisions are
  recorded in `docs/DECISIONS.md` and governed by `PROJECT_RULES.md`. Do not
  resolve an open question by implementation; propose it and let the maintainer
  record the decision first.
- **Do not commit secrets.** Credentials come from the environment only. Never
  commit `.env`, `.env.prod`, `secrets.h`, keystores, service-account JSON, or
  MQTT password files. If you touch `.gitignore`, keep those exclusions.
- **Report faithfully.** If tests fail, say so with the output. Do not claim a
  step is done without evidence, and do not invent evidence.
- **Match the surrounding code.** Follow each component's existing style, naming,
  and comment language (firmware comments are Indonesian; the Android UI is
  Indonesian-first).

## Repository conventions

- Canonical units everywhere: PGA in gal (cm/s^2), timestamps in milliseconds
  epoch UTC, distances in km.
- The MQTT trigger protocol publishes exactly two signed messages per event
  (`PRELIM` and `FINAL`) sharing one observation sequence. Changes here must
  keep `contracts/mqtt/trigger.schema.json` and the firmware in agreement.
- Contracts in `contracts/` are the source of truth. Update the contract and the
  implementation together.

## Building and testing

### Server (Go)

```bash
cd server
go build ./...
go test ./...
```

Note: `TestBlockedLedgerDoesNotDelayDispatch` depends on scheduling and can fail
on a single-core host (`GOMAXPROCS=1`); it passes under multiple CPUs. Run it
with `GOMAXPROCS>=2` when reproducing.

### Firmware (PlatformIO)

```bash
cd firmware
cp src/secrets.h.example src/secrets.h
./scripts/check-secrets.sh
pio run
```

Do not commit `src/secrets.h`.

### Android (Gradle)

```bash
cd android
./gradlew testDebugUnitTest
./gradlew assembleDebug
```

## Pull requests

1. Branch from `development`.
2. Keep the change focused; explain what you changed, why, and what you tested.
3. Include or update tests for behavioural changes.
4. Do not force-push shared branches, and do not merge your own PR without
   maintainer review.
5. If your change affects a recorded decision or contract, note it in the PR and
   flag it for the maintainer to update `docs/DECISIONS.md`.

## Reporting bugs and vulnerabilities

- Functional bugs: open a GitHub issue with reproduction steps and environment.
- Security issues: follow [`SECURITY.md`](SECURITY.md) and report privately.

## Style

Use plain hyphens, not em dashes, in prose and comments (a project convention).
