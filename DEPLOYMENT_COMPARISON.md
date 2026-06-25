# EduWallet Backend Deployment Analysis & Guide

## Executive Summary

This report compares three managed PaaS platforms — **Railway**, **Render**, and **Fly.io** — for deploying the EduWallet Go backend (excluding GCP). The analysis evaluates each platform against the project's specific technical requirements and provides a complete step-by-step deployment guide for the recommended option.

**Recommended Platform:** **Railway** (primary) — ideal for Go + Postgres + Redis stacks with minimal ops overhead.  
**Alternative:** **Render** — excellent if you need a generous free tier or predictable pricing.  
**Specialized choice:** **Fly.io** — best for global edge distribution and advanced infrastructure control.

---

## 1. Project Profile

| Attribute | Value |
|-----------|-------|
| Language | Go 1.25.0 |
| Framework | Gin |
| Runtime | Single statically-linked binary (`cmd/api/main.go`) |
| Port | 8080 |
| Database | PostgreSQL 16 (pgx/v5) |
| Cache | Redis 7 (go-redis/v9) |
| Migrations | golang-migrate (10 SQL migration files) |
| Modes | `api` server + `worker` background poller |
| Email | Resend API |
| Payments | Razorpay (India) |
| Auth | JWT, RBAC, multi-tenant |
| Container | Multi-stage Alpine Dockerfile already provided |
| Codebase | ~89 Go files, ~14,500 lines |

### Deployment Requirements
1. Managed PostgreSQL service
2. Managed Redis service
3. Docker container support
4. Environment variable / secrets management
5. Ability to run both API and worker processes
6. Database migration step on deploy
7. HTTPS / SSL termination
8. Health check support (`/api/v1/healthz`)
9. Reasonable pricing and scaling path
10. Minimal operational complexity

---

## 2. Platform Comparison

### 2.1 High-Level Overview

| Criterion | Railway | Render | Fly.io |
|-----------|---------|--------|--------|
| **Best for** | Rapid Go/Postgres/Redis deployment | Full-stack apps, free tier users | Global edge, fine-grained infra control |
| **Go / Docker support** | ✅ Native Dockerfile auto-detect | ✅ Native Dockerfile support | ✅ Native Dockerfile support |
| **Managed Postgres** | ✅ One-click, automatic backups | ✅ Managed Postgres ($7+/mo) | ⚠️ Must provision Fly Postgres cluster |
| **Managed Redis** | ✅ One-click Redis service | ✅ Managed Redis ($20+/mo) | ⚠️ Uses Upstash Redis (external) |
| **Multi-process (API + Worker)** | ✅ Separate services same project | ✅ Multiple services | ✅ Multiple machines/apps |
| **Env vars / Secrets** | ✅ Excellent UI + CLI | ✅ Good UI/CLI | ✅ CLI + dashboard |
| **Deploy from Git** | ✅ Auto-deploy on push | ✅ Auto-deploy on push | ✅ via GitHub Action / CLI |
| **Custom domains + SSL** | ✅ Automatic | ✅ Automatic | ✅ Automatic |
| **Health checks** | ✅ Built-in | ✅ Built-in | ✅ Built-in |
| **Rollback** | ✅ One-click | ✅ One-click | ✅ via releases |
| **Free tier** | Limited trial credits | Generous free tier (Web services, Postgres) | Generous free resources |
| **Pricing predictability** | Good | Good | Variable (per VM/region/resources) |
| **Learning curve** | Low | Low | Medium |
| **Operational complexity** | Very low | Low | Medium |

### 2.2 Detailed Scoring (1–10)

| Factor | Railway | Render | Fly.io | Notes |
|--------|---------|--------|--------|-------|
| Ease of setup for this stack | 9 | 8 | 6 | Railway's Postgres+Redis are first-class add-ons. |
| Managed database quality | 9 | 8 | 6 | Fly Postgres is self-managed cluster; needs care. |
| Worker process support | 9 | 8 | 8 | Railway makes it trivial to add a second service. |
| Pricing for small/medium SaaS | 8 | 8 | 7 | Fly can surprise if resources are not capped. |
| Go-specific DX | 9 | 8 | 7 | Railway auto-builds from Dockerfile cleanly. |
| Scaling path | 8 | 8 | 9 | Fly scales horizontally by design. |
| Global edge / latency | 6 | 5 | 10 | Fly is the clear winner for worldwide users. |
| Observability & logs | 8 | 7 | 8 | All provide structured logs and metrics. |
| Enterprise / compliance | 7 | 8 | 7 | Render has more mature enterprise features. |
| **Overall fit for EduWallet** | **9.0** | **8.0** | **7.2** | |

### 2.3 Pricing Snapshot (Approximate, as of 2026)

> Prices change; verify on each platform's website before provisioning.

| Service | Railway | Render | Fly.io |
|---------|---------|--------|--------|
| Web service (small) | ~$5–$15/mo | **Free tier** then ~$7–$25/mo | **Free allowance**, then ~$1.94–$5/mo per small VM |
| Managed Postgres | ~$15–$30/mo starter | $7–$15/mo starter | Self-managed cluster ~$5–$15/mo |
| Managed Redis | ~$5–$15/mo | $20+/mo | Upstash Redis ~$10+/mo |
| Estimated minimum prod stack | **$25–$60/mo** | **$27–$60/mo** | **$15–$40/mo** but variable |
| Estimated comfortable prod stack | $50–$120/mo | $60–$150/mo | $40–$100/mo |

**Cost note:** Fly.io can appear cheapest but requires manual provisioning of Postgres and Redis clusters. Operational time and risk should be factored in.

---

## 3. Platform-Specific Analysis

### 3.1 Railway ✅ Recommended

**Why it fits EduWallet best:**
- **Zero-config Dockerfile detection.** Railway reads your existing multi-stage `Dockerfile` and exposes port 8080 automatically.
- **First-class managed Postgres and Redis.** Add both as services in the same project; networking and connection strings are injected as environment variables.
- **Simple multi-service model.** Deploy the API as one service, duplicate it with `APP_MODE=worker` for the background worker — all inside the same project with shared env vars.
- **Automatic SSL, custom domains, and deployments on git push.**
- **Great CLI and dashboard** for logs, metrics, and environment variables.

**Considerations:**
- Pricing is resource-based; watch usage if you scale up.
- Less global edge presence than Fly.io.

**Verdict:** Best balance of simplicity, managed services, and Go support.

### 3.2 Render (Strong Alternative)

**Why consider Render:**
- **Generous free tier** for web services and Postgres — great for staging/MVP.
- Native Docker support and managed Postgres.
- Straightforward `render.yaml` Blueprint for infrastructure-as-code.
- Good enterprise/compliance story.

**Considerations:**
- Managed Redis starts at $20+/mo, which is higher than Railway.
- Worker services require an explicit background worker configuration.
- Docker build times can be slower than Railway.

**Verdict:** Excellent if cost-conscious and using the free tier, or if you need Render's enterprise features.

### 3.3 Fly.io (Specialized / Advanced)

**Why consider Fly.io:**
- **Best-in-class global edge deployment.** Run VMs close to your users.
- Very fast cold starts and excellent performance for Go binaries.
- Fine-grained control over VM sizes, regions, and scaling.
- Good free resource allowance.

**Considerations:**
- **No native managed Postgres/Redis.** You must run `fly postgres create` (self-managed cluster) and use Upstash or self-hosted Redis.
- Higher operational complexity: you manage database backups, scaling, failovers.
- Pricing is granular and can spike with traffic if not capped.

**Verdict:** Choose if your users are global and you have ops bandwidth to manage the data layer.

---

## 4. Final Recommendation

| Scenario | Recommended Platform |
|----------|---------------------|
| Fastest path to production for EduWallet | **Railway** |
| Lowest cost MVP / free-tier staging | **Render** |
| Global user base, edge latency critical | **Fly.io** |
| Managed Postgres + Redis with minimal ops | **Railway** |
| Team with strong DevOps wanting full control | **Fly.io** |

**Primary recommendation: Railway.** It aligns perfectly with EduWallet's stack (Go binary + Postgres + Redis), minimizes operational overhead, and gets you from git push to live URL in minutes.

---

## 5. A-to-Z Deployment Guide: Railway

This section provides a complete, production-ready deployment of EduWallet on Railway.

### 5.1 Prerequisites

1. **Railway account:** [https://railway.app](https://railway.app)
2. **GitHub repository** with your EduWallet backend code pushed.
3. **Railway CLI** installed (optional but recommended):
   ```bash
   npm install -g @railway/cli
   railway login
   ```
4. **Dockerfile present** (already exists in the repo root).
5. **Domain ready** (optional): custom domain for production.

### 5.2 Step 1 — Create a Railway Project

**Via Dashboard:**
1. Go to [https://railway.app/dashboard](https://railway.app/dashboard).
2. Click **New Project**.
3. Choose **Deploy from GitHub repo**.
4. Select your `eduwallet-backend` repository.
5. Railway will auto-detect the Dockerfile and start building.

**Via CLI:**
```bash
railway login
railway init
# Select "Deploy from GitHub repo" and choose your repository
```

### 5.3 Step 2 — Add PostgreSQL Service

1. In your Railway project, click **New** → **Database** → **Add PostgreSQL**.
2. Wait for the database to provision.
3. Railway automatically injects a `DATABASE_URL` environment variable.

**The simplest and most reliable approach is to use Railway's injected `DATABASE_URL` connection string.** EduWallet now supports `DATABASE_URL` directly. In the **Variables** tab of your API service, add:

```env
DATABASE_URL=${{Postgres.DATABASE_URL}}
DB_SSL_MODE=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
```

> Replace `Postgres` with the actual service name Railway assigned (e.g., `Postgres`, `postgres`, or `PostgreSQL`).
>
> **Why this is recommended:** `DATABASE_URL` is a single connection string that already includes host, port, user, password, database name, and SSL mode. It avoids the "DB_PASSWORD empty" errors that can happen when individual variable references fail to resolve.
>
> **Alternative (individual variables):** If you prefer to map individual fields, use:
> ```env
> DB_HOST=${{Postgres.PGHOST}}
> DB_PORT=${{Postgres.PGPORT}}
> DB_USER=${{Postgres.PGUSER}}
> DB_PASSWORD=${{Postgres.PGPASSWORD}}
> DB_NAME=${{Postgres.PGDATABASE}}
> DB_SSL_MODE=require
> ```
> Always use the **private/internal** `DATABASE_URL` or `PG*` variables, not `DATABASE_PUBLIC_URL`, for the app service.

### 5.4 Step 3 — Add Redis Service

1. Click **New** → **Database** → **Add Redis**.
2. Wait for Redis to provision.
3. Railway injects a `REDIS_URL` variable.

**The simplest approach is to use Railway's injected `REDIS_URL` connection string.** EduWallet now supports `REDIS_URL` directly. In the **Variables** tab of your API service, add:

```env
REDIS_URL=${{Redis.REDIS_URL}}
```

> Replace `Redis` with the actual service name Railway assigned.
>
> **Why this is recommended:** `REDIS_URL` is a single connection string (e.g., `redis://default:password@host:6379/0`) that avoids password parsing issues.
>
> **Alternative (individual variables):**
> ```env
> REDIS_HOST=${{Redis.REDISHOST}}
> REDIS_PORT=${{Redis.REDISPORT}}
> REDIS_PASSWORD=${{Redis.REDISPASSWORD}}
> REDIS_DB=0
> ```
>
> **⚠️ Critical:** Do **not** set a custom `REDIS_PASSWORD` that contains quotes, spaces, or command-line flags such as `--save`. The Redis `requirepass` directive accepts only a single token. If your Redis service is crashing with `requirepass "--save" "60" "1"`, regenerate the Redis service or set a plain alphanumeric password. See the troubleshooting section below for details.

### 5.5 Step 4 — Configure Application Environment Variables

In your main API service, set the following variables:

```env
APP_ENV=production
APP_MODE=api
APP_PORT=8080
APP_NAME=eduwallet
APP_EXTERNAL_URL=https://your-domain.railway.app
CORS_ALLOWED_ORIGINS=https://your-frontend-domain.com
WORKER_POLL_INTERVAL=5s

AUTH_PUBLIC_REGISTRATION_ENABLED=false

JWT_ACCESS_SECRET=your-access-secret-min-32-chars
JWT_REFRESH_SECRET=your-refresh-secret-different-from-access
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

RESEND_API_KEY=re_xxxxxxxx
RESEND_FROM_EMAIL=noreply@yourdomain.com
RESEND_FROM_NAME=EduWallet

PAYMENT_PROVIDER=razorpay
RAZORPAY_KEY_ID=rzp_test_xxx_or_rzp_live_xxx
RAZORPAY_KEY_SECRET=your_razorpay_secret
RAZORPAY_WEBHOOK_SECRET=your_webhook_secret
RAZORPAY_BASE_URL=https://api.razorpay.com/v1
```

**Generate strong JWT secrets locally:**
```bash
openssl rand -base64 48
```
Run it twice and use different values for access and refresh secrets.

### 5.6 Step 5 — Run Database Migrations

Railway does not automatically run migrations. You have three options:

#### Option A: One-Off Migration Job (Recommended first deploy)

1. In your Railway project, open the **API service**.
2. Click **Deploy** (or use the existing deployment).
3. Open a **shell** into the running container.
4. Run migrations:
   ```bash
   ./api migrate
   ```
   If your binary does not have a migrate command, install and run `golang-migrate` using Railway's injected `DATABASE_URL`:
   ```bash
   apk add --no-cache curl
   curl -L https://github.com/golang-migrate/migrate/releases/download/v4.19.1/migrate.linux-amd64.tar.gz | tar xvz
   ./migrate -path ./migrations -database "${DATABASE_URL}" up
   ```
   Railway's Postgres service injects `DATABASE_URL` automatically (e.g., `postgresql://user:pass@host:5432/db?sslmode=require`).

#### Option B: Add a Migrate Command to Your Go Binary

Create a small CLI entry in `cmd/migrate/main.go` to run migrations using the same config package. Then add a Railway one-off command.

#### Option C: Start Command with Migration (Simpler but riskier)

Modify the Dockerfile `CMD` to run migrations before starting the server:

```dockerfile
# Add migrate binary to runtime stage
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate

CMD ["sh", "-c", "migrate -path ./migrations -database \"${DATABASE_URL}\" up && ./api"]
```

> **Warning:** This can cause issues with zero-downtime deploys if migrations lock tables. Use only for simple deployments.

**Recommended long-term approach:** Use a CI/CD GitHub Action that runs migrations before deploying (see Section 5.10).

### 5.7 Step 6 — Deploy the API Service

1. Railway auto-deploys when you push to the default branch.
2. Go to your service **Settings** → verify the **Start Command** is empty (let Dockerfile `CMD` run).
3. Verify the **Healthcheck Path** is `/api/v1/healthz`.
4. Check **Deploy Logs** for errors.
5. Once healthy, Railway provides a public URL like `https://eduwallet-backend-production.up.railway.app`.

### 5.8 Step 7 — Deploy the Worker Service

EduWallet has a `worker` mode for background operations (reminders). Deploy it as a separate service.

1. In Railway, click **New** → **Empty Service** (or duplicate the existing service).
2. Source it from the same GitHub repo.
3. In service **Settings**, set the start command to:
   ```bash
   ./api
   ```
   (The binary will read `APP_MODE=worker` from env vars.)
4. In the worker service **Variables**, add:
   ```env
   APP_MODE=worker
   APP_ENV=production
   WORKER_POLL_INTERVAL=5s
   DATABASE_URL=${{Postgres.DATABASE_URL}}
   REDIS_URL=${{Redis.REDIS_URL}}
   ```
   Share the same JWT, Resend, and Razorpay variables from the API service (Railway supports shared variable groups).
5. **Do not expose a public domain** for the worker service.
6. Deploy.

### 5.9 Step 9 — Add a Custom Domain

1. In your API service, go to **Settings** → **Domains**.
2. Click **Generate Domain** for a free Railway subdomain, or **Custom Domain** to use your own.
3. If custom, add the CNAME record Railway provides to your DNS.
4. Update `APP_EXTERNAL_URL` and `CORS_ALLOWED_ORIGINS` to use the new domain.
5. Redeploy.

### 5.10 Step 10 — GitHub Actions CI/CD (Optional but Recommended)

Add `.github/workflows/deploy.yml` to run tests and migrations before Railway deploys:

```yaml
name: CI/CD

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: go test -race -count=1 ./...

  deploy:
    needs: test
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install Railway CLI
        run: npm install -g @railway/cli
      - name: Run migrations
        run: railway run --service eduwallet-api migrate
        env:
          RAILWAY_TOKEN: ${{ secrets.RAILWAY_TOKEN }}
      - name: Deploy to Railway
        run: railway up --service eduwallet-api
        env:
          RAILWAY_TOKEN: ${{ secrets.RAILWAY_TOKEN }}
```

> Note: This assumes you add a `migrate` command to your binary. Adjust accordingly.

Set the `RAILWAY_TOKEN` secret in your GitHub repository settings.

### 5.11 Step 11 — Monitoring & Operations

- **Logs:** Railway dashboard → Service → Deploy Logs / Runtime Logs.
- **Metrics:** Railway provides CPU, memory, and disk usage per service.
- **Alerts:** Configure usage alerts in Railway billing settings.
- **Backups:** PostgreSQL backups are automatic daily; download from the Postgres service settings.
- **Uptime:** Use an external monitor like UptimeRobot or Pingdom on `/api/v1/healthz`.

---

## 6. Troubleshooting

### Redis crash: `requirepass "--save" "60" "1"`

If your Redis service logs show:

```
>>> 'requirepass "--save" "60" "1"'
wrong number of arguments
```

**Root cause:** Redis's `requirepass` directive in `redis.conf` accepts exactly one token. The password value contains quotes, spaces, or command-line flags, so Redis interprets it as multiple arguments.

**Fix:**

1. **If using Railway managed Redis:**
   - Do **not** manually override the password in the Redis service settings.
   - Delete the Redis service and recreate it via **New → Database → Add Redis**.
   - Use the injected `REDIS_URL`, `REDISHOST`, `REDISPORT`, and `REDISPASSWORD` variables in your app service.

2. **If using a custom Redis container or image:**
   - Set `REDIS_PASSWORD` to a plain alphanumeric string with **no spaces, quotes, or dashes**.
   - Good: `aB3dE9fG0hJ1kL2mN`
   - Bad: `--save "60" "1"`, `my secret!`, `pass-with-dashes`

3. **Verify the config:** Open a shell in the Redis container and check `/etc/redis/redis.conf` (or the file path from the error). The line should look like:
   ```
   requirepass aB3dE9fG0hJ1kL2mN
   ```
   not:
   ```
   requirepass "--save" "60" "1"
   ```

4. **Redeploy** the Redis service and then restart your API/worker services so they pick up the new credentials.

### `DB_PASSWORD` should not be empty

If you see:

```
error: load config: parse config: env: environment variable "DB_PASSWORD" should not be empty
```

**Root cause:** The `DB_PASSWORD` variable reference (e.g., `${{Postgres.PGPASSWORD}}`) is resolving to empty, usually because the Postgres service name in the reference does not match exactly.

**Fix (recommended):** Use `DATABASE_URL` instead of individual variables. EduWallet now supports `DATABASE_URL` directly:

```env
DATABASE_URL=${{Postgres.DATABASE_URL}}
```

**Alternative fixes:**

1. Open your API service → **Variables**.
2. Delete the empty `DB_PASSWORD` variable.
3. Use Railway's **Add Reference** button (key icon) to select the Postgres service and `PGPASSWORD` variable. This ensures the service name and variable name are correct.
4. Redeploy.

### API cannot connect to Postgres

If the API crashes with connection errors:

1. Confirm you are using the **private/internal** `DATABASE_URL` or the `PGHOST` / `PGPORT` / `PGUSER` / `PGPASSWORD` / `PGDATABASE` variables.
2. Ensure `DB_SSL_MODE=require` (or that `DATABASE_URL` includes `sslmode=require`).
3. Check that the API service and Postgres service are in the same Railway project and network.
4. Verify the migration has been run and the database exists.

### Worker not processing jobs

1. Verify the worker service has `APP_MODE=worker`.
2. Ensure the worker service has the same database and Redis variables as the API service.
3. Check worker logs for polling interval and errors.

---

## 7. Production Checklist

- [ ] `APP_ENV=production` set
- [ ] `DATABASE_URL` is set (or `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME`) with SSL enabled
- [ ] `DB_SSL_MODE=require` (or `DATABASE_URL` includes `sslmode=require`)
- [ ] `APP_EXTERNAL_URL` uses HTTPS
- [ ] `CORS_ALLOWED_ORIGINS` does not include `*`
- [ ] JWT access and refresh secrets are different and ≥32 chars
- [ ] `RESEND_API_KEY` and sender email are real
- [ ] `PAYMENT_PROVIDER=razorpay` with all Razorpay keys set
- [ ] `AUTH_PUBLIC_REGISTRATION_ENABLED=false` unless self-signup is intended
- [ ] Database migrations applied successfully
- [ ] Worker service running with `APP_MODE=worker`
- [ ] Health check endpoint `/api/v1/healthz` returning 200
- [ ] Custom domain configured with SSL
- [ ] CI/CD pipeline running tests before deploy
- [ ] Backups verified (download and test restore)

---

## 8. Quick Reference: Platform Decision Matrix

| Your Priority | Choose |
|--------------|--------|
| Fastest, simplest production deploy | **Railway** |
| Best managed Postgres + Redis experience | **Railway** |
| Free tier / low-cost MVP | **Render** |
| Enterprise features / compliance | **Render** |
| Global edge / low latency worldwide | **Fly.io** |
| Maximum infrastructure control | **Fly.io** |
| Lowest operational overhead | **Railway** |

---

## 9. Conclusion

For the EduWallet backend — a Go/Gin service requiring PostgreSQL, Redis, an API server, and a background worker — **Railway is the best deployment platform** among Railway, Render, and Fly.io. It offers the most streamlined setup, excellent managed database integration, and the lowest operational complexity, allowing the team to focus on product development rather than infrastructure management.

Render is a strong alternative, especially for cost-sensitive MVPs, while Fly.io suits teams needing global edge distribution and willing to manage more infrastructure complexity.

Follow the Railway A-to-Z guide above to deploy EduWallet to production with confidence.
