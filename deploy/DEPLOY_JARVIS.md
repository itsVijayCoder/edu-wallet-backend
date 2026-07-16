# EduWallet — jarvis (Mac mini) Deployment Plan & Runbook

**Status: DEPLOYED 2026-07-17 — fresh DB (no Railway data), api + worker running, locally
verified (readyz, swagger, api-test, seeded admin login). Public URL live after the
one-time tunnel reload. Railway instance still up in parallel until decommissioned.**

Migration of the EduWallet backend from Railway to the jarvis Mac mini, following the
worqplace reference pattern (co-tenant docker-compose stack under the Lima engine,
published through the shared `mac-mini` cloudflared tunnel).

## Allocation

| Item | Value |
|---|---|
| Public URL | `https://eduwallet-api.udhaykumarbala.dev` (API base `/api/v1`) |
| Host port | `127.0.0.1:8130` → container `:8080` (loopback-only; cloudflared is the only way in) |
| Deploy dir | `/Users/jarvis/eduwallet` (`.env` lives ONLY here, never in git) |
| Repo | `/Users/jarvis/susanoox/edu-wallet-backend` (local on this box) |
| Compose project | `eduwallet` (containers `eduwallet-{postgres,redis,migrate,api,worker}-1`) |
| Compose files | `docker-compose.yml` (base) + `deploy/docker-compose.macmini.yml` (overlay) |
| Deploy script | `deploy/deploy-local.sh` (local run, no ssh — repo is on the box) |
| Tunnel | shared `mac-mini` (ID f206d4ea…); ingress inserted above the 404 catch-all |

## Topology

```
internet → Cloudflare edge → cloudflared (root daemon, this host)
        → http://localhost:8130 → Lima port-forward → api container :8080
api ─┬─ postgres:16-alpine   (compose-internal only, volume pg_data)
     └─ redis:7-alpine       (compose-internal only, volume redis_data)
worker (APP_MODE=worker, no HTTP port) — drains reminder jobs from Postgres
migrate (one-shot, gates api+worker) — ./eduwallet-migrate up
```

- `restart: unless-stopped` everywhere + Lima auto-start (`com.udhay.lima`) ⇒ reboot recovery is automatic.
- Worker is required for reminder **delivery** — the API only enqueues reminder jobs; without the worker they sit in Postgres forever.
- Both api and worker need Postgres **and** Redis reachable at boot (10s startup context) or they exit.

## Key decisions (and why)

1. **`APP_ENV=production` with the Railway values carried over verbatim.** The production
   config gate (internal/config/config.go:129-183) requires Resend + Razorpay values, but
   the Railway placeholders (`rzp_test_xxx_or_rzp_live_xxx`, `re_xxxxxxxx`,
   `noreply@yourdomain.com`) all **pass** validation — Railway itself ran this way. So the
   move preserves the status quo: app boots, payments/email fail at runtime until real keys
   are supplied. Do **not** switch APP_ENV away from production to "relax" the gate — any
   non-production env force-enables open public registration (cmd/api/main.go:88).
2. **JWT secrets reused from Railway** (in the deploy-dir `.env`). Note this does NOT
   preserve sessions: refresh tokens are validated against **Redis** (auth.go:307,
   `refreshKey(userID)`), and the new Redis starts empty — every user re-logs-in once
   after cutover regardless. Reuse is still the right call (in-flight access tokens stay
   verifiable during the overlap; nothing to gain by rotating).
3. **`TRUSTED_PROXIES=172.16.0.0/12`.** Without it Gin trusts no proxy, every request
   resolves to the docker bridge gateway IP, and all clients share ONE rate-limit bucket
   per route (e.g. a global 5/min on login — one abuser locks out everyone). The CIDR
   covers compose bridge subnets; safe because the port is loopback-bound and only
   cloudflared (which sets X-Forwarded-For from the verified client IP) can reach it.
4. **DB via `DB_*` parts, `DB_SSL_MODE=prefer`, no `DATABASE_URL`.** The same `DB_USER`/
   `DB_PASSWORD`/`DB_NAME` vars drive both the app DSN and the postgres container init
   (base compose interpolates them from the same `.env`) — one source of truth, no
   password duplication. `prefer` passes the production gate (only `disable` is rejected)
   and pgx falls back to plaintext against the TLS-less compose Postgres. The Railway
   `DB_SSL_MODE=require` would fail here; `sslmode=disable` in a DATABASE_URL fails the gate.
   **Exception — the migrate binary:** golang-migrate connects via **lib/pq**, which
   rejects `sslmode=prefer` outright (`unsupported sslmode`), while the production gate
   rejects `disable`. The overlay therefore overrides the migrate service alone with
   `APP_ENV=development` + `DB_SSL_MODE=disable` (verified: migrate would otherwise exit 1
   and the `depends_on` gate would block api+worker forever).
5. **One-shot `migrate` service owns migrations** (api/worker override `command: ["./api"]`
   to skip render-start.sh). Runs once per `up`, gates api+worker via
   `service_completed_successfully`, avoids re-running migrate on every crash-restart, and
   after a Railway restore it just logs "no new migrations to apply" (schema_migrations is
   already at v12 in the dump). Note: the migrate binary calls `config.Load()`, so it needs
   the full validated `.env` too.
6. **Full-host tunnel exposure** (no path gating). Everything lives under `/api/v1/`; the
   frontend needs the whole surface, and `POST /api/v1/webhooks/razorpay` (HMAC-verified,
   unauthenticated) must stay reachable by Razorpay's servers.

## Cutover procedure

### Step 0 — pre-flight (done 2026-07-17)
- [x] Port 8130 free, hostname unclaimed in `~/.cloudflared/config.yml`
- [x] `deploy/docker-compose.macmini.yml` + `deploy/deploy-local.sh` authored
- [x] `/Users/jarvis/eduwallet/.env` created (mode 600): Railway values carried over,
      fresh local DB password generated, `TRUSTED_PROXIES` + new external URL set

### Step 1 — data migration from Railway (recommended; skip for a fresh start)

Needs the Railway **public** DB URL:
`postgresql://postgres:<POSTGRES_PASSWORD>@<RAILWAY_TCP_PROXY_DOMAIN>:<RAILWAY_TCP_PROXY_PORT>/railway`
(from the Railway Postgres service → `DATABASE_PUBLIC_URL`).

```bash
cd /Users/jarvis/eduwallet
export RAILWAY_DB_URL='postgresql://postgres:...@....railway.app:PORT/railway'

# 1a. Railway server major decides the pg_dump client image — pg_dump refuses
#     if the server is NEWER than the client:
/opt/homebrew/bin/docker run --rm postgres:16-alpine psql "$RAILWAY_DB_URL" -tAc "show server_version;"
#     If it reports 17.x: use postgres:17-alpine below AND bump the postgres image
#     in docker-compose.yml to 17-alpine so dump/restore majors match end-to-end.

# 1b. Dump (schema + data + schema_migrations):
/opt/homebrew/bin/docker run --rm postgres:16-alpine \
  pg_dump "$RAILWAY_DB_URL" --no-owner --no-privileges --clean --if-exists \
  > eduwallet_railway.sql

# 1c. Start ONLY postgres (fresh volume, creds initialized from .env):
/opt/homebrew/bin/docker compose -p eduwallet -f docker-compose.yml \
  -f deploy/docker-compose.macmini.yml up -d postgres

# 1d. Restore:
cat eduwallet_railway.sql | /opt/homebrew/bin/docker exec -i eduwallet-postgres-1 \
  psql -U eduwallet -d eduwallet_db
```

Restore **before** the full stack comes up; the migrate service then no-ops at v12.
Redis is not migrated. It holds only rate-limit counters (reset harmlessly) and the
refresh-token store — meaning **every user logs in once after cutover**; the Postgres
`refresh_tokens` table is legacy (only ever deleted from) and does not restore sessions.

### Step 2 — deploy

```bash
cd /Users/jarvis/susanoox/edu-wallet-backend
bash deploy/deploy-local.sh              # manual (trigger=manual)
```

The script runs a **self-protecting phased pipeline** — every phase is recorded to
`deploy-state/status.json` and served live at `/api/v1/docs/deploy-status` +
the `/api/v1/docs/deployments` page (see "Self-protecting pipeline" below). Around
the pipeline it still does: rsync → first-run `.env` bootstrap → cloudflared ingress
insert (backup + validate + auto-rollback) → DNS route → prints the reload command.
Phases: **prechecks → build → backup → migrate+swap → postchecks → success | revert**.

### Step 3 — tunnel reload (first deploy only, needs sudo+TTY)

```bash
sudo kill -HUP $(pgrep -f 'cloudflared tunnel.*run')
```

### Step 4 — verify

```bash
curl -s http://localhost:8130/api/v1/readyz          # {"status":"ok",...} pg+redis up
curl -s https://eduwallet-api.udhaykumarbala.dev/api/v1/healthz
open https://eduwallet-api.udhaykumarbala.dev/api/v1/docs   # Swagger (server URL auto-set from APP_EXTERNAL_URL)
```

### Step 5 — post-deploy hardening (IMPORTANT)

1. **Rotate the seeded super admin.** Migration 000010 plants
   `admin@eduwallet.in` / `password` (super_admin role) on ANY database that ran the
   migrations — including the restored Railway data. Before/immediately after the public
   URL goes live: set in `.env` `SUPER_ADMIN_BOOTSTRAP_ENABLED=true` +
   `SUPER_ADMIN_PASSWORD=<strong, non-"password">`, `docker compose ... up -d api` once,
   confirm login, then set it back to `false` and restart. (The weak-password guard
   hard-fails startup if you enable bootstrap while the password is still `password`.)
2. Supply **real Razorpay keys** (`RAZORPAY_KEY_ID/KEY_SECRET/WEBHOOK_SECRET`) when
   payments are needed, and point the Razorpay dashboard webhook at
   `https://eduwallet-api.udhaykumarbala.dev/api/v1/webhooks/razorpay`.
3. Supply a **real Resend key + verified sender** when email (OTP, reminders) is needed —
   placeholders boot fine but every send fails at runtime.
4. Update frontend(s) to the new API base URL and add their real origins to
   `CORS_ALLOWED_ORIGINS` (only frontend origins matter; the API's own host is inert there).
5. Decommission Railway once traffic is confirmed on the new URL (rollback = just keep
   Railway until then; nothing on jarvis touches it).

## Team deploys (pull-based — no jarvis access needed)

**Push or merge to `main` on `github.com/susanoox/edu-wallet-backend`; jarvis deploys it
within ~5 minutes.** The LaunchAgent `com.udhay.eduwallet-autodeploy` runs
`deploy/auto-deploy-poll.sh` (from `/Users/jarvis/eduwallet/deploy/`, refreshed by every
deploy) every 300s: it fetches `origin/main` into the dedicated clean clone
`/Users/jarvis/eduwallet-src` and, on a new commit, resets to it and runs
`deploy/deploy-local.sh --trigger=auto` (the full phased pipeline below).

- **Live status:** every deploy is recorded and shown at
  `https://eduwallet-api.udhaykumarbala.dev/api/v1/docs/deployments` (auto-refreshes;
  running version, "Up to date / Behind main / HELD" badge, per-attempt phases + error
  tails). Raw JSON: `/api/v1/docs/deploy-status`.
- **Every poll refreshes `latest_main`** in `deploy-state/status.json` (fuels the
  "Behind main" badge) even when nothing is deployed.
- Last deployed SHA: `/Users/jarvis/eduwallet/.deployed-sha` — written only on a
  successful deploy; a failed deploy is retried on the next poll — **until it is held**.
- **Hold (auto-stop retrying a bad commit):** after **2 consecutive
  failed/rolled_back** attempts for the *same* `origin/main` SHA, the poller stops
  retrying it (records `held` in status.json, logs `HELD`, the page shows a HELD badge)
  and waits. **Clearing a hold, two ways:**
  1. **Push a new commit to `main`** (the normal path — a fix or a revert). The poller
     sees the new `origin/main` SHA and clears the hold automatically on the next poll.
  2. **Manually delete the `held` key** on the box (e.g. to retry the same SHA after
     fixing the *environment* rather than the code), then force a poll:
     ```bash
     python3 - <<'PY'
     import json, os
     p = "/Users/jarvis/eduwallet/deploy-state/status.json"
     d = json.load(open(p)); d["held"] = None
     json.dump(d, open(p + ".tmp", "w"), indent=2); os.replace(p + ".tmp", p)
     PY
     launchctl kickstart -k gui/501/com.udhay.eduwallet-autodeploy
     ```
     (This only un-pauses the poller; if the commit is still broken it re-holds after 2
     more failed attempts.)
- Logs: `~/Library/Logs/eduwallet-autodeploy.{out,err}.log`
- Schema migrations in pushed commits apply automatically (one-shot migrate service).
- Rollback: revert the bad commit on `main` — the poller deploys the revert. (Manual
  alternative on the box: reset `eduwallet-src` to a good SHA, run
  `deploy/deploy-local.sh`, then write that SHA to `.deployed-sha`.)
- The developer working repo (`/Users/jarvis/susanoox/edu-wallet-backend`) is never
  touched by the poller.
- Agent management: `launchctl kickstart -k gui/501/com.udhay.eduwallet-autodeploy`
  (deploy now), `launchctl bootout gui/501/com.udhay.eduwallet-autodeploy` (disable).

## Self-protecting pipeline (deploy-local.sh)

Both manual and auto deploys run the same phased pipeline. Each phase is timed and
recorded (with a one-line `detail`) to `deploy-state/status.json`; the attempt is
written **pessimistically as `failed` up front and flushed at every phase boundary**, so
a killed/crashed run still leaves a coherent record (an `EXIT` trap finalizes any
unfinished attempt). All timestamps are UTC RFC3339. status.json keeps the **newest 20**
attempts.

| Phase | What it does | Failure handling |
|---|---|---|
| **prechecks** | docker reachable, `compose config -q`, `.env` has no `FILL_ME`, ≥5GB free on `/Users` (docker `system df` informational), port 8130 not held by a foreign process, `schema_migrations.dirty` not true (skipped if pg/table absent) | record `failed`, exit 1 — running stack untouched |
| **build** | export `GIT_SHA` (deployed 40-hex) + `BUILD_TIME`, `compose build` (separately, not `up --build`). First run tags the existing `:latest`→`:good` as a revert baseline | record `failed`, exit 1 |
| **backup** | if postgres up: `pg_dump \| gzip` → `deploy-state/backups/pre-<id>-<sha_short>.sql.gz`, keep newest **7** | record `failed`, exit 1 (skipped if pg down) |
| **migrate+swap** | `compose up -d` (one-shot `migrate` gates api+worker), verify migrate exit 0 | record `failed` (with migrate logs), exit 1 |
| **postchecks** | local `readyz` 200 ≤90s; `deploy-status` reports `build.sha == GIT_SHA` (proves the NEW binary is serving); `/api/v1/docs` 200; login probe returns 4xx (not 5xx/refused); 30s stability window (healthz every 5s + api `RestartCount == 0`); public healthz **warn-only** | on any failure → **revert** |
| **success** | `docker tag :latest :good` (new known-healthy baseline), record `success` | — |
| **revert** | `docker tag :good :latest` → `compose up -d --no-build api worker` → wait `readyz` ≤90s; record **`rolled_back`**. If revert itself is unhealthy: detail `REVERT UNHEALTHY — manual intervention` | exit 1 |

**Revert semantics — read this.** Revert swaps the **container image** back to the last
known-healthy `:good` build. It does **NOT** revert the database schema: the
`migrate+swap` phase has already applied any new migrations, and rolling the image back
runs the *old* binary against the *new* schema. That is safe for additive migrations but
**a destructive/renaming migration paired with a bad build can leave the old binary
unable to serve** — in that case restore from the pre-deploy dump:

```bash
gunzip -c /Users/jarvis/eduwallet/deploy-state/backups/pre-<id>-<sha_short>.sql.gz \
  | /opt/homebrew/bin/docker exec -i eduwallet-postgres-1 psql -U eduwallet -d eduwallet_db
```

Backups live in `deploy-state/backups/` (newest 7 kept). The `deploy-state/` dir is
created by the pipeline, bind-mounted **read-only** into the api container at
`/app/deploy-state`, and is never rsync'd (excluded alongside `.env`).

## Manual redeploy (from the box)

```bash
cd /Users/jarvis/susanoox/edu-wallet-backend && bash deploy/deploy-local.sh
```
No tunnel reload needed. Migrations run automatically via the migrate service.

## Operations

```bash
D=/opt/homebrew/bin/docker; C="$D compose -p eduwallet -f docker-compose.yml -f deploy/docker-compose.macmini.yml"
cd /Users/jarvis/eduwallet
$C ps                    # status
$C logs -f api           # api logs (worker/migrate/postgres likewise)
$C restart api           # restart one service
$C down                  # stop stack (volumes survive)
$D exec -it eduwallet-postgres-1 psql -U eduwallet -d eduwallet_db   # SQL shell
```

- Postgres creds initialize from `.env` on FIRST volume creation only — changing
  `DB_PASSWORD` later requires an `ALTER USER` inside Postgres, not just an .env edit.
- If migrate ever reports a dirty schema: `$C run --rm migrate ./eduwallet-migrate force <version>`.
- Rate limiter fails CLOSED: while Redis is down, login/register/webhook/payment routes
  return 503 (and `readyz` reports it). Compose health-gating + restart policy cover this.

## Gotchas quick-reference

| Trap | Consequence | Handled by |
|---|---|---|
| APP_ENV≠production | public registration force-enabled | keep `production` |
| `sslmode=disable` anywhere / `DB_SSL_MODE=disable` | startup refusal in production | `DB_SSL_MODE=prefer`, no DATABASE_URL |
| `sslmode=prefer` in the **migrate** container | lib/pq rejects it → migrate exits 1 → api/worker blocked | overlay overrides migrate: `APP_ENV=development` + `DB_SSL_MODE=disable` |
| `TRUSTED_PROXIES` unset | one global rate-limit bucket for all users | `172.16.0.0/12` in .env |
| `APP_PORT=8130` in .env | container binds 8130 internally, healthcheck (hardcoded :8080) breaks | keep `APP_PORT=8080`; 8130 is only the host side |
| seeded `admin@eduwallet.in`/`password` | well-known super-admin creds live on the public URL | Step 5.1 rotation |
| worker omitted | reminder emails silently never send | worker service in overlay |
| `RESEND_FROM_EMAIL` unset | inherits `noreply@example.com` default → startup refusal | explicit value in .env |
| pg_dump client < Railway server major | dump aborts with version mismatch | Step 1a check |

## Open items (user input needed)

- **Real frontend origin(s)** for `CORS_ALLOWED_ORIGINS` — current list (API's own host +
  `eduwallet-api.asthrix.live`) passes validation but no browser frontend is listed. What
  domain does the SPA run on?
- **Razorpay + Resend real credentials** — placeholders carried from Railway; payments and
  email are non-functional until replaced.
- **Railway `DATABASE_PUBLIC_URL`** (TCP proxy host:port) — needed for the data dump in
  Step 1; not derivable from the private URL.
- Whether `eduwallet-api.asthrix.live` should eventually CNAME to this deployment or be
  retired (DNS for asthrix.live is outside this box).
