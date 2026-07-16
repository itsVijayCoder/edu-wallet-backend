# Production Deployment

This Compose file intentionally runs only the API and worker. Use managed PostgreSQL and Redis (or separately hardened private services), and terminate TLS at a reverse proxy. Do not expose PostgreSQL or Redis to the Internet.

## Before deployment

1. Create a secret environment file outside the repository. At a minimum set production database/Redis endpoints, unique 32+ character JWT secrets, HTTPS external URL and CORS origin, verified Resend sender, and Razorpay credentials.
2. Set `TRUSTED_PROXIES` to only the IPs/CIDRs of the reverse proxy that terminates traffic. Leave it blank only for direct deployments. This prevents forged `X-Forwarded-For` values from evading IP-based limits.
3. Back up the database and confirm the migration version. The API startup migrates before serving traffic; run one API replica during migrations to avoid concurrent deployment races.
4. Set `SUPER_ADMIN_BOOTSTRAP_ENABLED=false` after the initial credential bootstrap.

## Start

```bash
ENV_FILE=/secure/eduwallet.production.env \
  docker compose -f docker-compose.production.yml up -d --build
```

The API binds to loopback only. Configure the reverse proxy to send HTTPS traffic to `127.0.0.1:8080`; keep `APP_EXTERNAL_URL` and CORS origins on the public HTTPS domain. Run at least one worker replica for reminder/export jobs.

## Verify and operate

```bash
docker compose -f docker-compose.production.yml ps
curl -fsS http://127.0.0.1:8080/api/v1/healthz
docker compose -f docker-compose.production.yml logs -f api worker
```

Monitor the health endpoint, application error rate, Redis availability, failed jobs, payment-webhook failures, database storage, and backup restoration tests. The Compose services run non-root with a read-only filesystem, dropped Linux capabilities, resource limits, health checks, and bounded local logs.
