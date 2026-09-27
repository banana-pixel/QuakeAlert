# QuakeAlert production deployment (`deploy/`)

This directory is the production deployment: compose stack, reverse proxy,
broker config, operator scripts, and secret placeholders. Read this before
touching production.

## Which host is production

- Service: `api.quakealert.web.id`
- Public IP: `168.110.217.4`
- Tailscale IP: `100.91.154.117` — **the same host, reached over Tailscale,
  not over public SSH.**

`100.91.154.117` is the production VPS, not a development host. It looks like
one (repo checkout, android-sdk, adb, no app ports on some scans), because the
application stack runs in Docker and the database lives in the `postgis`
container — a bare `ps`/`ss` on the host shows almost nothing. Do not conclude
"nothing runs here" from host-level process listings; check
`docker ps` (containers `quakealert-server`, `quakealert-caddy`,
`quakealert-mosquitto`, `quakealert-postgis`, `quakealert-redis`).

Public SSH to `168.110.217.4` is not used and must stay that way; firewall and
Tailscale configuration are out of scope for deployments. Administrative
access is SSH as `opc` over the Tailscale path using the environment's
provisioned key. No credential is stored here: `.env.prod` (600,
operator-owned) holds secrets and is never printed, committed, or invented —
only referenced by filename.

To verify you are on the right host, match public egress (read-only, no
changes):

```
curl -4 -s -m 15 ifconfig.me   # must print 168.110.217.4
```

## Deploy convention

From `/opt/quakealert` (or the synced checkout acting as deploy source):

1. Record rollback state first: current commit, server image ID
   (`docker images quakealert-server:prod`), and `schema_migrations` max.
2. Check out the exact release commit (detached, by SHA — never "latest").
3. `docker compose --env-file /opt/quakealert/deploy/.env.prod -f \
   /opt/quakealert/deploy/docker-compose.prod.yml build server`
   (build-only; running containers are untouched).
4. `... up -d` — the `migrate` service (golang-migrate, read-only mount of
   `../contracts/db/migrations`) runs to completion before the server
   recreates, so schema and binary move together.
5. Verify: `docker logs quakealert-server` (no errors), public `/healthz`,
   `schema_migrations`, endpoint presence, `/sensors` + `/events` smoke.

Rollback is checkout of the previous commit + rebuild + `up -d`; a schema
downgrade additionally needs its `down` migration and is only safe while no
row depends on the new schema (e.g. no designated Admin Node).
