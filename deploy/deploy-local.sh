#!/usr/bin/env bash
# Deploy eduwallet to jarvis (this box) as a Lima-engine docker-compose stack,
# co-tenant behind the shared mac-mini cloudflared tunnel. The repo lives locally,
# so unlike worqplace's deploy-to-macmini.sh everything runs without ssh.
# Idempotent + re-runnable.
#
#   bash deploy/deploy-local.sh
#
# What it does (never disrupts other tenants):
#   1. preflight (foreign listener on 8130 is fatal; our own api = redeploy)
#   2. rsync repo -> /Users/jarvis/eduwallet (preserves the deploy-dir .env)
#   3. .env template generated on FIRST deploy only, then ABORTS for you to fill it
#   4. docker compose build + up (one-shot migrate gate, then api + worker)
#   5. surgical, backed-up, validated cloudflared ingress for the public hostname
#   6. DNS route; prints the tunnel-reload command (needs sudo+TTY, first deploy only)
#   7. local + public health verification
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEPLOY_DIR="/Users/jarvis/eduwallet"
PUBLIC_HOST="eduwallet-api.udhaykumarbala.dev"
PORT="8130"
TUNNEL="mac-mini"
PROJECT="eduwallet"
CFD="/opt/homebrew/bin/cloudflared"
DOCKER="/opt/homebrew/bin/docker"
COMPOSE="${DOCKER} compose -p ${PROJECT} -f docker-compose.yml -f deploy/docker-compose.macmini.yml"
TS="$(date -u +%Y%m%d-%H%M%S)"

say() { printf '\n\033[1;36m== %s\033[0m\n' "$1"; }

# ---------------------------------------------------------------------------
say "0. preflight (port ${PORT}, docker, cloudflared config)"
${DOCKER} info >/dev/null 2>&1 || { echo "ABORT: docker engine (Lima) not reachable"; exit 1; }
HOLDER=$(lsof -ti tcp:${PORT} 2>/dev/null | head -1 || true)
if [ -n "${HOLDER}" ]; then
  OURS=$(${DOCKER} ps --filter "name=${PROJECT}-api" --format '{{.Names}}' || true)
  [ -n "${OURS}" ] || { echo "ABORT: port ${PORT} held by a non-eduwallet process (pid ${HOLDER})"; exit 1; }
  echo "port ${PORT} held by our own ${OURS} — redeploy"
fi
test -f "$HOME/.cloudflared/config.yml" || { echo "ABORT: ~/.cloudflared/config.yml not found"; exit 1; }

# ---------------------------------------------------------------------------
say "1. rsync ${REPO_ROOT} -> ${DEPLOY_DIR} (preserve deploy-dir .env)"
mkdir -p "${DEPLOY_DIR}"
rsync -a --delete \
  --exclude='.git' --exclude='.claude' --exclude='bin/' --exclude='tmp/' \
  --exclude='.env' --exclude='*.log' \
  "${REPO_ROOT}/" "${DEPLOY_DIR}/"
echo "synced."

# ---------------------------------------------------------------------------
say "2. ensure ${DEPLOY_DIR}/.env"
cd "${DEPLOY_DIR}"
if [ ! -f .env ]; then
  # First deploy with no .env: write a template with generated DB/JWT secrets and
  # FILL_ME markers, then stop. Copying the real Railway values instead of the
  # generated JWT secrets keeps existing refresh tokens valid after a data restore.
  DBP=$(openssl rand -hex 24)
  JA=$(openssl rand -base64 48 | tr -d '\n')
  JR=$(openssl rand -base64 48 | tr -d '\n')
  cat > .env <<EOF
APP_ENV=production
APP_MODE=api
APP_PORT=8080
APP_NAME=eduwallet
APP_EXTERNAL_URL=https://${PUBLIC_HOST}
CORS_ALLOWED_ORIGINS=https://${PUBLIC_HOST},https://FILL_ME_frontend_origin
TRUSTED_PROXIES=172.16.0.0/12
WORKER_POLL_INTERVAL=5s
AUTH_PUBLIC_REGISTRATION_ENABLED=false
SUPER_ADMIN_BOOTSTRAP_ENABLED=false
SUPER_ADMIN_EMAIL=admin@eduwallet.in
SUPER_ADMIN_PASSWORD=
SUPER_ADMIN_FIRST_NAME=EduWallet
SUPER_ADMIN_LAST_NAME=Owner
DB_HOST=postgres
DB_PORT=5432
DB_USER=eduwallet
DB_PASSWORD=${DBP}
DB_NAME=eduwallet_db
DB_SSL_MODE=prefer
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_DB=0
JWT_ACCESS_SECRET=${JA}
JWT_REFRESH_SECRET=${JR}
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h
RESEND_API_KEY=FILL_ME
RESEND_FROM_EMAIL=FILL_ME
RESEND_FROM_NAME=EduWallet
PAYMENT_PROVIDER=razorpay
PAYMENT_FAKE_SIGNING_SECRET=test_payment_secret
RAZORPAY_BASE_URL=https://api.razorpay.com/v1
RAZORPAY_KEY_ID=FILL_ME
RAZORPAY_KEY_SECRET=FILL_ME
RAZORPAY_WEBHOOK_SECRET=FILL_ME
EOF
  chmod 600 .env
  echo "ABORT: fresh .env template written to ${DEPLOY_DIR}/.env — fill the FILL_ME"
  echo "       values (see deploy/DEPLOY_JARVIS.md), then re-run this script."
  exit 1
fi
if grep -q 'FILL_ME' .env; then
  echo "ABORT: ${DEPLOY_DIR}/.env still contains FILL_ME values — fill them, then re-run."
  exit 1
fi
chmod 600 .env

# ---------------------------------------------------------------------------
say "3. compose validate + build + up"
${COMPOSE} config -q || { echo "ABORT: compose config invalid"; exit 1; }
${COMPOSE} up -d --build
echo "--- migrate result (want exit 0) ---"
${DOCKER} inspect -f '{{.State.Status}} exit={{.State.ExitCode}}' ${PROJECT}-migrate-1 2>/dev/null \
  || echo "(migrate container not found)"
${COMPOSE} ps --format '{{.Name}} | {{.Service}} | {{.Status}}'

# ---------------------------------------------------------------------------
say "4. local health (readyz pings postgres + redis)"
READY=""
for i in $(seq 1 20); do
  if curl -fsS "http://localhost:${PORT}/api/v1/readyz" >/dev/null 2>&1; then
    READY="yes"; echo "readyz OK (attempt ${i})"; break
  fi
  sleep 3
done
curl -s "http://localhost:${PORT}/api/v1/readyz" || true; echo
[ -n "${READY}" ] || { echo "ABORT: api not ready — check: ${COMPOSE} logs api migrate"; exit 1; }

# ---------------------------------------------------------------------------
say "5. cloudflared ingress (surgical, backed up, validated)"
CFG="$HOME/.cloudflared/config.yml"
cp "${CFG}" "${CFG}.bak-${TS}"
PUBLIC_HOST="${PUBLIC_HOST}" PORT="${PORT}" CFG="${CFG}" python3 - <<'PY'
import os, re, sys
p = os.environ["CFG"]
host = os.environ["PUBLIC_HOST"]
port = os.environ["PORT"]
s = open(p).read()
if host in s:
    print("ingress already present — no change")
    sys.exit(0)
lines = s.splitlines()
idx = None
for i, ln in enumerate(lines):
    # LAST match = the global catch-all list item; per-hostname 404s have no leading "- "
    if re.match(r'^\s*-\s+service:\s*http_status:\s*404\s*$', ln):
        idx = i
if idx is None:
    sys.stderr.write("ABORT: no global 404 catch-all found\n")
    sys.exit(2)
ind = lines[idx][: len(lines[idx]) - len(lines[idx].lstrip())]
lines[idx:idx] = [f"{ind}- hostname: {host}", f"{ind}  service: http://localhost:{port}"]
open(p, "w").write("\n".join(lines) + "\n")
print(f"inserted ingress for {host} -> http://localhost:{port}")
PY
${CFD} tunnel ingress validate \
  || { echo "VALIDATION FAILED — restoring backup"; cp "${CFG}.bak-${TS}" "${CFG}"; exit 1; }

# ---------------------------------------------------------------------------
say "6. DNS route (CNAME on the shared tunnel)"
${CFD} tunnel route dns ${TUNNEL} ${PUBLIC_HOST} 2>&1 | tail -1 || echo "(route may already exist — non-fatal)"

# ---------------------------------------------------------------------------
say "7. tunnel reload (manual — needs sudo+TTY; FIRST deploy only)"
echo ">>> sudo kill -HUP \$(pgrep -f 'cloudflared tunnel.*run')"
echo ">>> Zero-downtime reload of the ROOT cloudflared daemon. Only needed when the"
echo ">>> hostname was just added; code-only redeploys (step 3) never need it."

# ---------------------------------------------------------------------------
say "8. public check"
sleep 4
if curl -fsS "https://${PUBLIC_HOST}/api/v1/healthz" >/dev/null 2>&1; then
  echo "PUBLIC OK: https://${PUBLIC_HOST}/api/v1/healthz"
else
  echo "(public URL not live yet — run the reload in step 7, then:"
  echo "  curl https://${PUBLIC_HOST}/api/v1/healthz )"
fi
echo
echo "Done. public https://${PUBLIC_HOST}/api/v1 | local http://localhost:${PORT}/api/v1"
