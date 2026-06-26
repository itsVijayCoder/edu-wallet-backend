# EduWallet — Render.com Deployment Guide

## Overview

This guide documents the end-to-end process for deploying the **EduWallet backend** (Go) to **Render.com** using:

| Service | Purpose |
|---|---|
| **Render Web Service** | Containerized API server (Docker) |
| **Render PostgreSQL** | Primary database |
| **Render Redis** | Caching, sessions, rate limiting |
| **Render Background Worker** (optional) | Background reminder-email loop |

The app ships with a multi-stage `Dockerfile` that builds two binaries:
`./api` (HTTP server) and `./eduwallet-migrate` (migrations), plus the
`migrations/` folder. Both are copied into the runtime image.

---

## How your app is wired (key facts from the code)

- **Entry point**: `cmd/api/main.go` → binary `./api`, listens on `APP_PORT` (default `8080`)
- **Dockerfile**: multi-stage build, produces `./api` + `./eduwallet-migrate`, copies `migrations/`, `EXPOSE 8080`, `CMD ["./api"]`
- **DB config** (`internal/database/postgres.go:25`): `DATABASE_URL` takes precedence over `DB_*` vars
- **Redis config** (`internal/database/redis.go:27`): `REDIS_URL` takes precedence over `REDIS_*` vars
- **Health checks**: `GET /api/v1/healthz` (liveness) and `GET /api/v1/readyz` (pings Postgres + Redis)
- **Two modes**: `APP_MODE=api` (HTTP server) and `APP_MODE=worker` (background loop)
- **Migrations**: `./eduwallet-migrate up` — reuses `config.Load()`, reads the same env vars
- **Production gate** (`internal/config/config.go:113`): if `APP_ENV=production`, the app **refuses to start** unless ALL of these are met:
  - `JWT_ACCESS_SECRET` ≠ `JWT_REFRESH_SECRET`, both ≥ 32 chars
  - `DB_SSL_MODE` ≠ `disable` (or use `DATABASE_URL` without `sslmode=disable`)
  - `APP_EXTERNAL_URL` is a real `https://` URL (not localhost)
  - `CORS_ALLOWED_ORIGINS` has ≥ 1 `https://` origin, no `*`
  - `RESEND_API_KEY` is set
  - `RESEND_FROM_EMAIL` is not `example.com`
  - `PAYMENT_PROVIDER=razorpay` + `RAZORPAY_KEY_ID` / `RAZORPAY_KEY_SECRET` / `RAZORPAY_WEBHOOK_SECRET`

> **Two paths below.** Path A gets you live fast (staging, fake payments, no email). Path B flips on full production validation.

---

## Prerequisites

1. Code pushed to a GitHub/GitLab repo (Render deploys from your git repo)
2. A Render account at https://render.com
3. Locally, generate two **different** JWT secrets now:
   ```bash
   openssl rand -base64 48   # → paste as JWT_ACCESS_SECRET
   openssl rand -base64 48   # → paste as JWT_REFRESH_SECRET (must be different!)
   ```

---

## Step 1 — Create the PostgreSQL database on Render

1. Render Dashboard → **New +** → **PostgreSQL**
2. Fill in:
   - **Name**: `eduwallet-db`
   - **Database**: `eduwallet_db`
   - **User**: `eduwallet`
   - **Region**: pick the same region you'll use for the web service
   - **PostgreSQL Version**: 16
   - **Instance Type**: Free (expires after 90 days) or Starter (~$7/mo, persistent)
3. Click **Create Database**
4. Once provisioned, scroll to **Connections**. You'll see:
   - **Internal Database URL** — use this for the web service (same Render region, low latency)
   - **External Database URL** — use this only from your local machine

Copy the **Internal Database URL**. It looks like:
```
postgresql://eduwallet:PASSWORD@dpg-xxxxx-a:5432/eduwallet_db
```

> This string does **not** contain `sslmode=disable`, so it passes your app's production validation.

---

## Step 2 — Create the Redis instance on Render

1. Render Dashboard → **New +** → **Redis**
2. Fill in:
   - **Name**: `eduwallet-redis`
   - **Region**: same region as Postgres
   - **Instance Type**: Free (ephemeral, data not persisted) or Starter (~$7/mo, persistent)
3. Click **Create Redis**
4. Once provisioned, copy the **Internal Redis URL**. It looks like:
   ```
   redis://red-xxxxx:6379
   ```

> Your app reads `REDIS_URL` via `redis.ParseURL` (`internal/database/redis.go:28`), so this works directly.

---

## Step 3 — Create the Web Service (the API) on Render

1. Render Dashboard → **New +** → **Web Service**
2. Connect your GitHub/GitLab account and select the `eduwallet-backend` repo
3. Fill in:
   - **Name**: `eduwallet-api`
   - **Region**: same region as Postgres + Redis
   - **Runtime**: **Docker** (Render reads your `Dockerfile`)
   - **Instance Type**: Free (spins down after 15 min idle) or Starter (~$7/mo, always-on)
   - **Docker Command**: see **Step 5** below (this is how we run migrations on the free plan)
   - **Port**: `8080` (must match `APP_PORT` below and the Dockerfile's `EXPOSE 8080`)
4. **Do not click "Create Web Service" yet** — configure env vars first (Step 4).

---

## Step 4 — Configure environment variables on the web service

In the service's **Environment** tab, add these. I've split them into "always required" and "production-only".

### Always required (both staging and production)

| Key | Value |
|---|---|
| `APP_ENV` | `staging` *(Path A)* or `production` *(Path B)* |
| `APP_MODE` | `api` |
| `APP_PORT` | `8080` |
| `APP_NAME` | `eduwallet` |
| `DATABASE_URL` | *(paste Render Postgres **Internal Database URL** from Step 1)* |
| `REDIS_URL` | *(paste Render Redis **Internal Redis URL** from Step 2)* |
| `JWT_ACCESS_SECRET` | *(paste 1st `openssl rand` output)* |
| `JWT_REFRESH_SECRET` | *(paste 2nd `openssl rand` output — must differ!)* |
| `JWT_ACCESS_EXPIRY` | `15m` |
| `JWT_REFRESH_EXPIRY` | `168h` |
| `AUTH_PUBLIC_REGISTRATION_ENABLED` | `false` |

### Path A — Staging (skip strict validation, get live fast)

Add these to use fake payments and skip the production gate:

| Key | Value |
|---|---|
| `APP_EXTERNAL_URL` | `https://eduwallet-api.onrender.com` *(your Render URL)* |
| `CORS_ALLOWED_ORIGINS` | `https://your-frontend-app.onrender.com` |
| `PAYMENT_PROVIDER` | `fake` |
| `PAYMENT_FAKE_SIGNING_SECRET` | `test_payment_secret` |
| `RESEND_API_KEY` | *(leave empty — email disabled)* |
| `RESEND_FROM_EMAIL` | `noreply@example.com` |
| `RESEND_FROM_NAME` | `eduwallet` |

> **Note on `APP_EXTERNAL_URL`:** Your Render URL is `https://<service-name>.onrender.com`. You can see it at the top of the service page after creation. Set `APP_EXTERNAL_URL` to that. Even in staging, set it to the `https://...onrender.com` URL so password-reset emails (if you later enable Resend) point to the right place.

With `APP_ENV=staging`, the `validateProduction()` block is **skipped entirely** (`internal/config/config.go:104`), so missing Razorpay/Resend won't block startup.

### Path B — Full production (all validations enforced)

Only switch `APP_ENV` to `production` once you have **all** of these:

| Key | Value |
|---|---|
| `APP_EXTERNAL_URL` | `https://api.yourdomain.com` (your custom domain, must be `https`) |
| `CORS_ALLOWED_ORIGINS` | `https://yourdomain.com,https://www.yourdomain.com` (comma-separated, no `*`) |
| `PAYMENT_PROVIDER` | `razorpay` |
| `RAZORPAY_KEY_ID` | `rzp_live_xxx` or `rzp_test_xxx` |
| `RAZORPAY_KEY_SECRET` | *(from Razorpay dashboard)* |
| `RAZORPAY_WEBHOOK_SECRET` | *(from Razorpay webhook settings)* |
| `RAZORPAY_BASE_URL` | `https://api.razorpay.com/v1` |
| `RESEND_API_KEY` | `re_xxx` (from Resend) |
| `RESEND_FROM_EMAIL` | `noreply@yourdomain.com` (must NOT be `example.com`) |
| `RESEND_FROM_NAME` | `EduWallet` |

> If `APP_ENV=production` and any of these are missing/placeholder, the container **exits immediately** with a clear error in the logs.

---

## Step 5 — Run database migrations (Free Plan workaround)

> **The problem:** On Render's **Free** plan, the **Pre-Deploy Command** and **Shell** features are **not available**. Both are paid-tier features. So you cannot run `./eduwallet-migrate up` via Pre-Deploy, and you cannot SSH into the container via Shell.

The Dockerfile already bundles `./eduwallet-migrate` and the `migrations/` folder inside the image. Below are **two free-plan-safe methods** to run migrations.

### Method A — Start Script (Recommended, automatic)

Instead of using Pre-Deploy, use a start script that runs migrations **then** starts the API on every boot. This works on the free plan because the "Docker Command" field is available on all tiers.

The repo includes `render-start.sh` at the root, and the `Dockerfile` copies it into the image. The script contents:

```sh
#!/bin/sh
set -e
./eduwallet-migrate up
exec ./api
```

1. On the web service → **Settings** → find **Docker Command**
2. Set it to:
   ```
   ./render-start.sh
   ```
3. Save.

**How this works:**
- `./eduwallet-migrate up` — applies all pending migrations; if there are none, it logs `no new migrations to apply` and exits 0 (the code treats `migrate.ErrNoChange` as success — see `cmd/migrate/main.go:139`)
- `set -e` — if migrations fail, the script exits immediately and the API does **not** start (fail-fast — don't serve traffic against a half-migrated schema)
- `exec ./api` — replaces the shell process with the API so `SIGTERM` reaches Go directly for graceful shutdown (`cmd/api/main.go:179`)

> **Why a script and not an inline command?** Render's Docker Command field does not reliably handle shell quoting (e.g., `sh -c '...'` gets mangled, producing `sh: ...: not found`). A script file sidesteps all quoting issues.

**Why this is safe:**
- `migrate up` is **idempotent** — running it when all migrations are already applied is a no-op that completes in milliseconds.
- On the free plan, the service spins down after 15 min idle and wakes on the next request. Each wake runs the start command, so migrations are checked on every cold start. The added latency is negligible (~100–500ms).
- If a migration fails, the API won't start — this is **fail-fast** behavior, which is what you want (don't serve traffic against a half-migrated schema).
- The migrate binary calls `config.Load()` (`cmd/migrate/main.go:93`), so it picks up `DATABASE_URL` from the service env vars — same connection as the API.

> **Concurrency note:** If you have multiple instances (Starter+ plan with scaling), multiple instances could run `migrate up` simultaneously. `golang-migrate` uses an advisory lock (`schema_migrations` table) to prevent concurrent migration runs, so this is safe. On the free plan you only have one instance, so this is not a concern.

### Method B — Run migrations locally against the External Database URL (Manual)

If you prefer to run migrations manually (or Method A fails and you need to debug), run the migrate binary from your local machine against Render's **External** Database URL.

1. Go to Render Dashboard → your PostgreSQL instance → **Connections** tab
2. Copy the **External Database URL** (only works from outside Render's network)
3. Build the migrate binary locally:
   ```bash
   make migrate
   ```
   This produces `bin/eduwallet-migrate`.

4. Run migrations with the external URL. Because the migrate binary reuses `config.Load()`, you need to provide the minimum env vars it validates. The easiest way is to override `DATABASE_URL` in the shell while keeping the rest from your local `.env`:

   ```bash
   APP_ENV=development \
   DATABASE_URL="postgresql://eduwallet:PASSWORD@host.render.com:5432/eduwallet_db?sslmode=require" \
   ./bin/eduwallet-migrate up
   ```

   **Important details:**
   - `APP_ENV=development` — skips the production validation gate so you don't need Razorpay/Resend keys just to migrate. (If you set `APP_ENV=production`, `config.Load()` will require ALL production env vars — see `internal/config/config.go:113`.)
   - `DATABASE_URL` — paste the **External** URL. Append `?sslmode=require` if it's not already present (Render's external endpoint requires SSL).
   - `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` — these are `notEmpty` required fields. Your local `.env` already has 64-char values, and `make migrate` / the binary loads `.env` via `godotenv.Load()`. If you're not using `.env`, set them to any 32+ char strings.

5. You should see:
   ```
   migrations applied successfully
   current schema version=NN dirty=false
   ```

6. After migrations succeed, the API service (started via Method A or a normal deploy) will connect to a fully-migrated database.

> **When to use Method B:**
> - Initial setup (create all tables before the first deploy)
> - Debugging a failed migration (you get the full error locally)
> - Rolling back a migration: `./bin/eduwallet-migrate down 1` (with the same env vars)
> - Checking the current version: `./bin/eduwallet-migrate version`

### Method C — Upgrade to a paid plan for Pre-Deploy + Shell (Optional)

If you upgrade to the **Starter** plan (~$7/mo) or higher, you get access to:
- **Pre-Deploy Command** — runs `./eduwallet-migrate up` before every deploy, separate from the start command
- **Shell** — interactive shell into the running container for debugging

With a paid plan, set:
- **Pre-Deploy Command**: `./eduwallet-migrate up`
- **Docker Command**: leave blank (uses the Dockerfile's `CMD ["./api"]`)

This is the cleanest setup but costs money. Methods A and B cover everything you need on the free plan.

---

## Step 6 — Deploy and verify

1. Click **Create Web Service** (or **Manual Deploy** → **Deploy latest commit** if already created)
2. Watch the build logs. You should see:
   - Docker build completes
   - Start command runs: `migrations applied successfully` (or `no new migrations to apply`)
   - `starting eduwallet env=staging port=8080`
   - `connected to PostgreSQL` / `connected to Redis`
   - `listening :8080`
3. Once live, verify the health endpoint:
   ```bash
   curl https://eduwallet-api.onrender.com/api/v1/readyz
   ```
   Expected:
   ```json
   {"status":"ok","postgres":"ok","redis":"ok"}
   ```

If `readyz` returns `"ok"` for both Postgres and Redis, you're live.

> **Free plan cold starts:** The free tier spins down after 15 min of inactivity. The first request after spin-down takes ~30–60s to wake up. During this wake-up, the start command runs migrations (no-op) then starts the API. Subsequent requests are fast until the next 15-min idle period.

---

## Step 7 — (Optional) Deploy the background Worker

The worker (`APP_MODE=worker`) runs the reminder-email loop (`cmd/api/main.go:217`). It needs the same DB/Redis/JWT env vars but no HTTP port.

1. Render Dashboard → **New +** → **Background Worker**
2. Select the same repo, **Runtime: Docker**
3. **Docker Command**: `./render-start.sh`
   (Same script as the API — runs migrations then starts the binary. `APP_MODE=worker` makes it run the worker loop instead of the HTTP server.)
4. **Environment** — copy the **exact same** env vars as the API service, but change:
   - `APP_MODE` = `worker`
   - Remove `APP_PORT` / `APP_EXTERNAL_URL` / `CORS_ALLOWED_ORIGINS` (not needed)
   - Keep `DATABASE_URL`, `REDIS_URL`, `JWT_*`, `RESEND_*`, `PAYMENT_*`
5. Create. The worker runs continuously and polls every `WORKER_POLL_INTERVAL` (default `5s`).

> **Note:** Background Workers are not available on the free plan — the minimum is Starter (~$7/mo). If you're on the free plan, skip the worker for now. The API service works fine without it; you just won't have automated reminder emails.

---

## Step 8 — (Optional) Custom domain

1. Web service → **Settings** → **Custom Domains** → **Add Custom Domain**
2. Enter `api.yourdomain.com`
3. Render gives you a CNAME target (e.g., `eduwallet-api.onrender.com`)
4. Add a CNAME record at your DNS provider: `api` → Render's target
5. Once DNS resolves, Render issues a TLS certificate automatically
6. Update `APP_EXTERNAL_URL` to `https://api.yourdomain.com` and redeploy

> Custom domains are available on the free plan.

---

## Free Plan Limitations

| Feature | Free Plan | Starter (~$7/mo) |
|---|---|---|
| Web Service | Spins down after 15 min idle (cold starts) | Always on |
| PostgreSQL | Expires after 90 days (data deleted) | Persistent |
| Redis | Ephemeral (data lost on restart) | Persistent |
| Pre-Deploy Command | Not available | Available |
| Shell | Not available | Available |
| Background Worker | Not available | Available |
| Custom Domains | Available | Available |
| Docker Command override | Available | Available |

**What this means for your deployment:**
- Use **Method A** (Docker Command override) for migrations — works on free plan
- Use **Method B** (local migrate against external URL) as a fallback
- Free Redis loses data on restart — sessions/rate-limit counters reset (users get logged out). For production, upgrade Redis to Starter.
- Free Postgres expires after 90 days — set a calendar reminder to upgrade before then, or start with Starter from day one.

---

## Troubleshooting

### Container exits immediately / "deploy failed"

Check the logs. The most common causes are the production validations in `internal/config/config.go:113`:

- `JWT_ACCESS_SECRET and JWT_REFRESH_SECRET must be different in production` → regenerate one of them
- `DB_SSL_MODE=disable is not allowed in production` → use `DATABASE_URL` instead of `DB_*` vars (Render's URL has no `sslmode=disable`)
- `APP_EXTERNAL_URL is required in production` → set it to your `https://...onrender.com` URL
- `CORS_ALLOWED_ORIGINS is required in production` → set at least one `https://` origin
- `RESEND_API_KEY is required in production` → add your Resend key
- `PAYMENT_PROVIDER must be razorpay in production` → set `PAYMENT_PROVIDER=razorpay` + all Razorpay keys
- **Quick fix to get live:** set `APP_ENV=staging` to skip all of these while you gather the real keys.

### `readyz` returns `postgres: "error"` or `redis: "error"`

The app started but can't reach the DB/Redis. Verify:
- `DATABASE_URL` / `REDIS_URL` are the **Internal** URLs (not External)
- Postgres, Redis, and the web service are in the **same Render region**
- The DB/Redis instances are not still provisioning

### Migrations didn't run / tables missing

- If using **Method A**: confirm the **Docker Command** is `./render-start.sh`. Check the deploy logs for `migrations applied successfully` right after the start command runs.
- If using **Method B**: run `./bin/eduwallet-migrate version` locally (with the external URL) to check the current schema version.
- Common migration error: `dirty=true` — a previous migration failed partway. Run `./bin/eduwallet-migrate force <version>` (with external URL) to clear the dirty flag, then `./bin/eduwallet-migrate up`.

### Port mismatch / 502 Bad Gateway

Render service **Port** setting must equal `APP_PORT` (both `8080`). The Dockerfile `EXPOSE 8080` and `APP_PORT=8080` must all agree.

### Can't connect from local machine to Render DB

Use the **External Database URL** + Render's external DB host/port from the Postgres **Connections** tab. The Internal URL only works between Render services. Append `?sslmode=require` if SSL errors occur.

### `migrate` binary not found in container

The Dockerfile builds `eduwallet-migrate` and copies it to `/app/eduwallet-migrate`. The start command runs from WORKDIR `/app`, so `./eduwallet-migrate` resolves correctly. If you see "file not found", verify the Dockerfile wasn't modified and the build stage completed successfully.

---

## Quick reference — minimal env vars to go live (Path A / staging)

```
APP_ENV=staging
APP_MODE=api
APP_PORT=8080
APP_NAME=eduwallet
APP_EXTERNAL_URL=https://eduwallet-api.onrender.com
CORS_ALLOWED_ORIGINS=https://your-frontend.onrender.com
DATABASE_URL=<Render Postgres Internal URL>
REDIS_URL=<Render Redis Internal URL>
JWT_ACCESS_SECRET=<openssl rand -base64 48>
JWT_REFRESH_SECRET=<different openssl rand -base64 48>
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h
AUTH_PUBLIC_REGISTRATION_ENABLED=false
PAYMENT_PROVIDER=fake
PAYMENT_FAKE_SIGNING_SECRET=test_payment_secret
```

**Docker Command**: `sh -c './eduwallet-migrate up && exec ./api'`
**Port**: `8080`
**Runtime**: Docker
**Health Check Path**: `/api/v1/readyz`

---

## Quick reference — local migration command (Method B)

```bash
# Build the migrate binary
make migrate

# Run against Render's External Database URL
APP_ENV=development \
DATABASE_URL="postgresql://eduwallet:PASSWORD@host.render.com:5432/eduwallet_db?sslmode=require" \
./bin/eduwallet-migrate up

# Check current version
APP_ENV=development \
DATABASE_URL="postgresql://eduwallet:PASSWORD@host.render.com:5432/eduwallet_db?sslmode=require" \
./bin/eduwallet-migrate version

# Roll back one migration
APP_ENV=development \
DATABASE_URL="postgresql://eduwallet:PASSWORD@host.render.com:5432/eduwallet_db?sslmode=require" \
./bin/eduwallet-migrate down 1
```