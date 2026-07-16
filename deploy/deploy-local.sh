#!/usr/bin/env bash
# Deploy eduwallet to jarvis (this box) as a Lima-engine docker-compose stack,
# co-tenant behind the shared mac-mini cloudflared tunnel. The repo lives locally,
# so unlike worqplace's deploy-to-macmini.sh everything runs without ssh.
# Idempotent + re-runnable.
#
#   bash deploy/deploy-local.sh                # manual deploy (trigger=manual)
#   bash deploy/deploy-local.sh --trigger=auto # used by auto-deploy-poll.sh
#
# Self-protecting phased pipeline (each phase recorded to deploy-state/status.json,
# served by the API at /api/v1/docs/deploy-status + the /api/v1/docs/deployments page):
#   prechecks -> build -> backup -> migrate+swap -> postchecks -> success|revert
# On any pre-swap failure the attempt is recorded `failed` and the script exits 1
# without disturbing the running stack. On a POSTCHECK failure the image is reverted
# to :good (last known-healthy) and the attempt is recorded `rolled_back`. The DB
# schema is NEVER auto-reverted — pre-migrate backups (deploy-state/backups) are for
# manual recovery only.
#
# Around the pipeline (unchanged, never disrupts other tenants):
#   - rsync repo -> /Users/jarvis/eduwallet (preserves the deploy-dir .env + deploy-state)
#   - .env template generated on FIRST deploy only, then ABORTS for you to fill it
#   - surgical, backed-up, validated cloudflared ingress for the public hostname
#   - DNS route; prints the tunnel-reload command (needs sudo+TTY, first deploy only)
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEPLOY_DIR="/Users/jarvis/eduwallet"
PUBLIC_HOST="eduwallet-api.udhaykumarbala.dev"
PORT="8130"
TUNNEL="mac-mini"
PROJECT="eduwallet"
IMAGE="eduwallet-backend"
CFD="/opt/homebrew/bin/cloudflared"
DOCKER="/opt/homebrew/bin/docker"
COMPOSE="${DOCKER} compose -p ${PROJECT} -f docker-compose.yml -f deploy/docker-compose.macmini.yml"

STATE_DIR="${DEPLOY_DIR}/deploy-state"
STATUS_FILE="${STATE_DIR}/status.json"
BACKUP_DIR="${STATE_DIR}/backups"
RUN_LOG="$(mktemp -t eduwallet-deploy.XXXXXX)"

# --- trigger ---------------------------------------------------------------
TRIGGER="manual"
for arg in "$@"; do
  case "$arg" in
    --trigger=auto)   TRIGGER="auto" ;;
    --trigger=manual) TRIGGER="manual" ;;
    --trigger=*)      echo "unknown flag: $arg" >&2; exit 2 ;;
    *)                echo "unknown arg: $arg" >&2; exit 2 ;;
  esac
done

# --- small helpers ---------------------------------------------------------
say()          { printf '\n\033[1;36m== %s\033[0m\n' "$1"; }
now_rfc3339()  { date -u +%Y-%m-%dT%H:%M:%SZ; }
epoch()        { date +%s; }
img_exists()   { ${DOCKER} image inspect "$1" >/dev/null 2>&1; }
pg_running()   { [ -n "$(${DOCKER} ps --filter "name=${PROJECT}-postgres-1" --filter status=running -q 2>/dev/null)" ]; }
# run a command, echoing its combined output to the console AND (synchronously) the
# run log so a failure's tail is captured for error_tail. pipefail => the command's
# exit status propagates through the tee.
run_logged()   { "$@" 2>&1 | tee -a "$RUN_LOG"; }

# --- git / build identity (from the SOURCE repo, before rsync) -------------
GIT_SHA="$(git -C "$REPO_ROOT" rev-parse HEAD 2>/dev/null || echo "")"
[ -n "$GIT_SHA" ] || GIT_SHA="0000000000000000000000000000000000000000"
SHA_SHORT="${GIT_SHA:0:12}"
COMMIT_SUBJECT="$(git -C "$REPO_ROOT" log -1 --format=%s 2>/dev/null || echo "")"
COMMIT_AUTHOR="$(git -C "$REPO_ROOT" log -1 --format=%an 2>/dev/null || echo "")"
ATTEMPT_ID="$(date -u +%Y%m%d-%H%M%S)"
BUILD_TIME="$(now_rfc3339)"

# --- state recorder (python3 heredocs, atomic tmp+mv, no jq) ---------------
CUR_PHASE=""
PHASE_EPOCH=0
ATTEMPT_EPOCH=0
ATTEMPT_STARTED=0
FINALIZED=0

state_init() {
  mkdir -p "${STATE_DIR}" "${BACKUP_DIR}"
  STATUS_FILE="$STATUS_FILE" python3 - <<'PY'
import os, json
p = os.environ["STATUS_FILE"]
if not os.path.exists(p):
    tmp = p + ".tmp"
    with open(tmp, "w") as f:
        json.dump({"latest_main": None, "held": None, "attempts": []}, f, indent=2)
    os.replace(tmp, p)
PY
}

# start a new (pessimistically `failed`) attempt at attempts[0]; cap the list at 20.
attempt_start() {
  ATTEMPT_EPOCH="$(epoch)"
  STATUS_FILE="$STATUS_FILE" A_ID="$ATTEMPT_ID" A_SHA="$GIT_SHA" A_SHORT="$SHA_SHORT" \
  A_SUBJECT="$COMMIT_SUBJECT" A_AUTHOR="$COMMIT_AUTHOR" A_TRIGGER="$TRIGGER" \
  A_STARTED="$(now_rfc3339)" python3 - <<'PY'
import os, json
p = os.environ["STATUS_FILE"]
try:
    data = json.load(open(p))
except Exception:
    data = {"latest_main": None, "held": None, "attempts": []}
att = {
    "id": os.environ["A_ID"],
    "sha": os.environ["A_SHA"], "sha_short": os.environ["A_SHORT"],
    "commit_subject": os.environ.get("A_SUBJECT", ""),
    "commit_author": os.environ.get("A_AUTHOR", ""),
    "trigger": os.environ["A_TRIGGER"],
    "started_at": os.environ["A_STARTED"], "finished_at": None, "duration_s": 0,
    "outcome": "failed",
    "phases": [], "error_tail": None,
}
data.setdefault("attempts", [])
data["attempts"].insert(0, att)
data["attempts"] = data["attempts"][:20]
tmp = p + ".tmp"
with open(tmp, "w") as f:
    json.dump(data, f, indent=2)
os.replace(tmp, p)
PY
  ATTEMPT_STARTED=1
}

# record_phase NAME STATUS DURATION_S DETAIL  -> append to attempts[0].phases
record_phase() {
  STATUS_FILE="$STATUS_FILE" PH_NAME="$1" PH_STATUS="$2" PH_DUR="$3" PH_DETAIL="$4" python3 - <<'PY'
import os, json
p = os.environ["STATUS_FILE"]
data = json.load(open(p))
ph = {
    "name": os.environ["PH_NAME"],
    "status": os.environ["PH_STATUS"],
    "duration_s": int(os.environ.get("PH_DUR", "0") or 0),
    "detail": os.environ.get("PH_DETAIL", ""),
}
if data.get("attempts"):
    data["attempts"][0].setdefault("phases", []).append(ph)
tmp = p + ".tmp"
with open(tmp, "w") as f:
    json.dump(data, f, indent=2)
os.replace(tmp, p)
PY
}

# finalize_attempt OUTCOME  -> set outcome/finished_at/duration + sanitized error_tail
finalize_attempt() {
  local outcome="$1" finished dur tailsrc=""
  finished="$(now_rfc3339)"
  dur=$(( $(epoch) - ATTEMPT_EPOCH ))
  [ "$outcome" = "success" ] || tailsrc="$RUN_LOG"
  STATUS_FILE="$STATUS_FILE" F_OUTCOME="$outcome" F_FINISHED="$finished" F_DUR="$dur" \
  F_TAILSRC="$tailsrc" python3 - <<'PY'
import os, json, re
p = os.environ["STATUS_FILE"]
data = json.load(open(p))
def sanitize(path):
    if not path or not os.path.exists(path):
        return None
    bad = re.compile(r'secret|password|token|_key|apikey|authorization', re.I)
    lines = []
    with open(path, errors='replace') as f:
        for ln in f:
            ln = ln.rstrip('\n')
            if bad.search(ln):
                continue
            lines.append(ln)
    lines = lines[-15:]
    s = "\n".join(lines)
    if len(s) > 2000:
        s = s[-2000:]
    return s or None
if data.get("attempts"):
    a = data["attempts"][0]
    a["outcome"] = os.environ["F_OUTCOME"]
    a["finished_at"] = os.environ["F_FINISHED"]
    a["duration_s"] = int(os.environ.get("F_DUR", "0") or 0)
    a["error_tail"] = None if os.environ["F_OUTCOME"] == "success" else sanitize(os.environ.get("F_TAILSRC", ""))
tmp = p + ".tmp"
with open(tmp, "w") as f:
    json.dump(data, f, indent=2)
os.replace(tmp, p)
PY
  FINALIZED=1
}

# --- phase flow helpers ----------------------------------------------------
phase_begin() { CUR_PHASE="$1"; PHASE_EPOCH="$(epoch)"; say "$1"; }
phase_dur()   { echo $(( $(epoch) - PHASE_EPOCH )); }
phase_ok()    { record_phase "$CUR_PHASE" ok      "$(phase_dur)" "$1"; }
phase_skip()  { record_phase "$CUR_PHASE" skipped "$(phase_dur)" "$1"; }
# fail_phase DETAIL  -> record current phase failed, finalize attempt failed, exit 1
fail_phase()  {
  echo "ABORT [$CUR_PHASE]: $1" | tee -a "$RUN_LOG" >&2
  record_phase "$CUR_PHASE" failed "$(phase_dur)" "$1"
  finalize_attempt failed
  exit 1
}

# do_revert REASON  -> record postchecks failed, restore :good, finalize rolled_back
do_revert() {
  local reason="$1" detail
  echo "postcheck failure: $reason" >> "$RUN_LOG"
  record_phase postchecks failed "$(phase_dur)" "$reason"
  phase_begin revert
  if img_exists "${IMAGE}:good"; then
    ${DOCKER} tag "${IMAGE}:good" "${IMAGE}:latest" >>"$RUN_LOG" 2>&1 || true
    run_logged ${COMPOSE} up -d --no-build api worker || true
    if wait_http "http://localhost:${PORT}/api/v1/readyz" 90; then
      detail="reverted to :good — healthy (${reason})"
      record_phase revert ok "$(phase_dur)" "$detail"
    else
      detail="REVERT UNHEALTHY — manual intervention (${reason})"
      record_phase revert failed "$(phase_dur)" "$detail"
    fi
  else
    detail="no :good baseline image — cannot revert (${reason})"
    record_phase revert failed "$(phase_dur)" "$detail"
  fi
  finalize_attempt rolled_back
  echo "ROLLED BACK: ${detail}" | tee -a "$RUN_LOG" >&2
  exit 1
}

# wait_http URL TIMEOUT_S -> 0 when it answers 2xx/3xx within the window
wait_http() {
  local url="$1" timeout="$2" waited=0
  while [ "$waited" -lt "$timeout" ]; do
    if curl -fsS "$url" >/dev/null 2>&1; then return 0; fi
    sleep 3; waited=$(( waited + 3 ))
  done
  return 1
}

# crash safety: any unexpected exit mid-attempt leaves a coherent `failed` record.
on_exit() {
  local rc=$?
  if [ "$ATTEMPT_STARTED" = 1 ] && [ "$FINALIZED" != 1 ]; then
    echo "unexpected exit ${rc} during phase ${CUR_PHASE}" >> "$RUN_LOG" 2>/dev/null || true
    finalize_attempt failed || true
  fi
}
trap on_exit EXIT

# ===========================================================================
# PREP (unrecorded): rsync + first-run .env bootstrap
# ===========================================================================
say "prep: rsync ${REPO_ROOT} -> ${DEPLOY_DIR} (preserve .env + deploy-state)"
mkdir -p "${DEPLOY_DIR}"
rsync -a --delete \
  --exclude='.git' --exclude='.claude' --exclude='bin/' --exclude='tmp/' \
  --exclude='.env' --exclude='*.log' --exclude='.deployed-sha' --exclude='*.sql' \
  --exclude='deploy-state/' \
  "${REPO_ROOT}/" "${DEPLOY_DIR}/"
echo "synced (trigger=${TRIGGER}, sha=${SHA_SHORT})."

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
chmod 600 .env

# ===========================================================================
# begin recorded attempt
# ===========================================================================
state_init
attempt_start
echo "attempt ${ATTEMPT_ID} started (sha ${SHA_SHORT}, trigger ${TRIGGER})"

# --- PHASE 1: prechecks ----------------------------------------------------
phase_begin "prechecks"
${DOCKER} info >/dev/null 2>&1 || fail_phase "docker engine (Lima) not reachable"
test -f "$HOME/.cloudflared/config.yml" || fail_phase "~/.cloudflared/config.yml not found"
${COMPOSE} config -q >>"$RUN_LOG" 2>&1 || fail_phase "compose config invalid"
if grep -q 'FILL_ME' .env; then fail_phase ".env still contains FILL_ME values"; fi

# disk: HARD-fail if < 5GB free on the /Users volume; docker df informational only
AVAIL_KB="$(df -Pk /Users 2>/dev/null | awk 'NR==2 {print $4}')"
if [ -n "$AVAIL_KB" ] && [ "$AVAIL_KB" -lt 5242880 ]; then
  fail_phase "low disk: $(( AVAIL_KB / 1024 ))MB free on /Users (< 5GB)"
fi
echo "--- docker system df (informational) ---"
${DOCKER} system df 2>/dev/null || echo "(docker system df unavailable)"

# port sanity: a foreign listener on our port is fatal; our own api = redeploy
HOLDER="$(lsof -ti tcp:${PORT} 2>/dev/null | head -1 || true)"
if [ -n "$HOLDER" ]; then
  OURS="$(${DOCKER} ps --filter "name=${PROJECT}-api" --format '{{.Names}}' 2>/dev/null || true)"
  if [ -z "$OURS" ]; then fail_phase "port ${PORT} held by foreign process (pid ${HOLDER})"; fi
  echo "port ${PORT} held by our own ${OURS} — redeploy"
fi

# dirty-schema guard: refuse to deploy on top of a half-applied migration
if pg_running; then
  DIRTY="$(${DOCKER} exec ${PROJECT}-postgres-1 psql -U eduwallet -d eduwallet_db -tAc \
    "SELECT dirty FROM schema_migrations LIMIT 1" 2>/dev/null | tr -d '[:space:]' || true)"
  if [ "$DIRTY" = "t" ]; then
    fail_phase "schema_migrations is dirty — resolve with: ${COMPOSE} run --rm migrate ./eduwallet-migrate force <version>"
  fi
  echo "schema_migrations clean (dirty=${DIRTY:-none})"
else
  echo "postgres not running — dirty-schema check skipped"
fi
phase_ok "config valid, disk ok, port ok, schema clean"

# --- PHASE 2: build --------------------------------------------------------
phase_begin "build"
export GIT_SHA BUILD_TIME
echo "GIT_SHA=${SHA_SHORT} BUILD_TIME=${BUILD_TIME}"
# snapshot the currently-running image as :good the FIRST time (so a bad new build
# can be reverted). Normally :good is re-tagged on success (phase 6).
if img_exists "${IMAGE}:latest" && ! img_exists "${IMAGE}:good"; then
  if ${DOCKER} tag "${IMAGE}:latest" "${IMAGE}:good" >>"$RUN_LOG" 2>&1; then
    echo "baseline: tagged existing ${IMAGE}:latest -> ${IMAGE}:good"
  fi
fi
run_logged ${COMPOSE} build || fail_phase "compose build failed"
phase_ok "image ${IMAGE}:latest built (sha ${SHA_SHORT})"

# --- PHASE 3: backup -------------------------------------------------------
phase_begin "backup"
if pg_running; then
  mkdir -p "${BACKUP_DIR}"
  BFILE="${BACKUP_DIR}/pre-${ATTEMPT_ID}-${SHA_SHORT}.sql.gz"
  if ${DOCKER} exec ${PROJECT}-postgres-1 pg_dump -U eduwallet eduwallet_db 2>>"$RUN_LOG" | gzip > "$BFILE"; then
    # keep only the newest 7 pre-deploy dumps
    ls -1t "${BACKUP_DIR}"/pre-*.sql.gz 2>/dev/null | tail -n +8 | while IFS= read -r old; do rm -f "$old"; done
    phase_ok "pg_dump -> ${BFILE##*/} ($(du -h "$BFILE" 2>/dev/null | awk '{print $1}'))"
  else
    rm -f "$BFILE"
    fail_phase "pg_dump failed"
  fi
else
  phase_skip "postgres not running — no pre-deploy backup"
fi

# --- PHASE 4: migrate + swap ----------------------------------------------
phase_begin "migrate"
if ! run_logged ${COMPOSE} up -d; then
  ${DOCKER} logs --tail 60 "${PROJECT}-migrate-1" >>"$RUN_LOG" 2>&1 || true
  fail_phase "compose up failed (migrate/api/worker did not come up)"
fi
MSTATE="$(${DOCKER} inspect -f '{{.State.Status}} {{.State.ExitCode}}' "${PROJECT}-migrate-1" 2>/dev/null || echo "missing 1")"
MEXIT="$(echo "$MSTATE" | awk '{print $2}')"
if [ -n "$MEXIT" ] && [ "$MEXIT" != "0" ]; then
  ${DOCKER} logs --tail 60 "${PROJECT}-migrate-1" >>"$RUN_LOG" 2>&1 || true
  fail_phase "migrate container exited ${MEXIT}"
fi
${COMPOSE} ps --format '{{.Name}} | {{.Service}} | {{.Status}}' || true
phase_ok "stack up; migrate exit ${MEXIT:-?}"

# --- PHASE 5: postchecks (must all pass, else revert) ----------------------
phase_begin "postchecks"
FAILREASON=""

# 5a. readyz 200 within 90s
if ! wait_http "http://localhost:${PORT}/api/v1/readyz" 90; then
  FAILREASON="readyz not 200 within 90s"
fi

# 5b. the NEW binary is serving: deploy-status build.sha == GIT_SHA
if [ -z "$FAILREASON" ]; then
  SERVED_SHA="$(curl -fsS "http://localhost:${PORT}/api/v1/docs/deploy-status" 2>/dev/null \
    | python3 -c 'import sys,json; print(json.load(sys.stdin).get("build",{}).get("sha",""))' 2>/dev/null || echo "")"
  if [ "$SERVED_SHA" != "$GIT_SHA" ]; then
    FAILREASON="deploy-status sha mismatch (serving='${SERVED_SHA:0:12}' want='${SHA_SHORT}')"
  fi
fi

# 5c. swagger docs == 200
if [ -z "$FAILREASON" ]; then
  CODE="$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:${PORT}/api/v1/docs" 2>/dev/null || echo 000)"
  if [ "$CODE" != "200" ]; then FAILREASON="/api/v1/docs returned ${CODE}"; fi
fi

# 5d. login probe returns 4xx (routing/handler alive; NOT 5xx, NOT conn-refused)
if [ -z "$FAILREASON" ]; then
  CODE="$(curl -s -o /dev/null -w '%{http_code}' -X POST "http://localhost:${PORT}/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d '{"email":"probe@invalid.example","password":"wrong"}' 2>/dev/null || echo 000)"
  case "$CODE" in
    4[0-9][0-9]) echo "login probe -> ${CODE} (ok)" ;;
    *)           FAILREASON="login probe returned ${CODE} (want 4xx)" ;;
  esac
fi

# 5e. stability window: 30s, healthz every 5s all 200 AND api RestartCount == 0
if [ -z "$FAILREASON" ]; then
  for i in $(seq 1 7); do
    CODE="$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:${PORT}/api/v1/healthz" 2>/dev/null || echo 000)"
    if [ "$CODE" != "200" ]; then FAILREASON="healthz ${CODE} during stability window"; break; fi
    RC="$(${DOCKER} inspect -f '{{.RestartCount}}' "${PROJECT}-api-1" 2>/dev/null || echo '?')"
    if [ "$RC" != "0" ]; then FAILREASON="api RestartCount=${RC} during stability window"; break; fi
    if [ "$i" -lt 7 ]; then sleep 5; fi
  done
fi

# 5f. public URL healthz — WARN ONLY (never fails the deploy)
if curl -fsS "https://${PUBLIC_HOST}/api/v1/healthz" >/dev/null 2>&1; then
  echo "public healthz OK: https://${PUBLIC_HOST}/api/v1/healthz"
else
  echo "WARN: public healthz not reachable (expected on first deploy before the tunnel reload)"
fi

if [ -n "$FAILREASON" ]; then do_revert "$FAILREASON"; fi
phase_ok "healthy; new binary serving (sha ${SHA_SHORT})"

# --- PHASE 6: success ------------------------------------------------------
if ${DOCKER} tag "${IMAGE}:latest" "${IMAGE}:good" >>"$RUN_LOG" 2>&1; then
  echo "tagged ${IMAGE}:latest -> ${IMAGE}:good (new known-healthy baseline)"
else
  echo "WARN: could not re-tag ${IMAGE}:good (deploy stays healthy)"
fi
finalize_attempt success
echo "attempt ${ATTEMPT_ID} — SUCCESS (sha ${SHA_SHORT})"

# ===========================================================================
# post-success infra (idempotent; failures here WARN only — the deploy is healthy)
# ===========================================================================
say "cloudflared ingress (surgical, backed up, validated)"
CFG="$HOME/.cloudflared/config.yml"
# NOTE: the deploy is already recorded SUCCESS above. Everything here is idempotent
# infra whose failure must NOT flip the exit code (else the poller skips .deployed-sha
# for a healthy build) — so every step below is guarded and warns instead of aborting.
if [ -f "$CFG" ]; then
  cp "${CFG}" "${CFG}.bak-${ATTEMPT_ID}" || true
  if PUBLIC_HOST="${PUBLIC_HOST}" PORT="${PORT}" CFG="${CFG}" python3 - <<'PY'
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
    sys.stderr.write("no global 404 catch-all found\n")
    sys.exit(2)
ind = lines[idx][: len(lines[idx]) - len(lines[idx].lstrip())]
lines[idx:idx] = [f"{ind}- hostname: {host}", f"{ind}  service: http://localhost:{port}"]
open(p, "w").write("\n".join(lines) + "\n")
print(f"inserted ingress for {host} -> http://localhost:{port}")
PY
  then
    if ${CFD} tunnel ingress validate; then
      say "DNS route (CNAME on the shared tunnel)"
      ${CFD} tunnel route dns ${TUNNEL} ${PUBLIC_HOST} 2>&1 | tail -1 || echo "(route may already exist — non-fatal)"
    else
      echo "WARN: ingress validation failed — restoring backup (deploy stays healthy)"
      cp "${CFG}.bak-${ATTEMPT_ID}" "${CFG}" || true
    fi
  else
    echo "WARN: ingress edit failed (no catch-all?) — restoring backup (deploy stays healthy)"
    cp "${CFG}.bak-${ATTEMPT_ID}" "${CFG}" || true
  fi
else
  echo "WARN: ~/.cloudflared/config.yml not found — skipping ingress/DNS"
fi

say "tunnel reload (manual — needs sudo+TTY; FIRST deploy only)"
echo ">>> sudo kill -HUP \$(pgrep -f 'cloudflared tunnel.*run')"
echo ">>> Zero-downtime reload of the ROOT cloudflared daemon. Only needed when the"
echo ">>> hostname was just added; code-only redeploys never need it."

echo
echo "Done. public https://${PUBLIC_HOST}/api/v1 | local http://localhost:${PORT}/api/v1"
echo "deployments page: https://${PUBLIC_HOST}/api/v1/docs/deployments (local: http://localhost:${PORT}/api/v1/docs/deployments)"
