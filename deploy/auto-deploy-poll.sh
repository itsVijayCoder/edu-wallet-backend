#!/usr/bin/env bash
# Poll GitHub main and redeploy eduwallet when new commits land.
# Team workflow: push/merge to main on github.com/susanoox/edu-wallet-backend →
# this poller (LaunchAgent com.udhay.eduwallet-autodeploy, every 5 min) picks it
# up and runs deploy/deploy-local.sh --trigger=auto. No jarvis access needed.
#
# Uses a dedicated clean clone (/Users/jarvis/eduwallet-src) so the developer
# working repo on this box is never touched. Every poll refreshes `latest_main`
# in deploy-state/status.json (fuels the "Behind main" badge on the deployments
# page). State file (.deployed-sha) records the last SUCCESSFULLY deployed SHA.
#
# Hold: after 2 consecutive failed/rolled_back attempts for the SAME origin/main
# SHA, the poller stops retrying that SHA (records `held`) until a NEW commit
# lands (which clears the hold). Watch ~/Library/Logs/eduwallet-autodeploy.out.log.
set -euo pipefail
# /usr/local/bin carries docker-credential-osxkeychain (docker's credsStore helper —
# buildkit hard-fails image pulls without it under launchd's minimal PATH)
export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
export GIT_TERMINAL_PROMPT=0

SRC="/Users/jarvis/eduwallet-src"
REPO_URL="https://github.com/susanoox/edu-wallet-backend"
BRANCH="main"
DEPLOY_DIR="/Users/jarvis/eduwallet"
STATE="${DEPLOY_DIR}/.deployed-sha"
STATE_DIR="${DEPLOY_DIR}/deploy-state"
STATUS_FILE="${STATE_DIR}/status.json"
LOCK="/tmp/eduwallet-autodeploy.lock"

mkdir "$LOCK" 2>/dev/null || { echo "another run in progress — skipping"; exit 0; }
trap 'rmdir "$LOCK"' EXIT

if [ ! -d "$SRC/.git" ]; then
  git clone --branch "$BRANCH" "$REPO_URL" "$SRC"
fi
git -C "$SRC" fetch --quiet origin "$BRANCH"
REMOTE_SHA=$(git -C "$SRC" rev-parse "origin/$BRANCH")
CURRENT=$(cat "$STATE" 2>/dev/null || echo none)

# Every poll: refresh latest_main + (re)compute the hold flag atomically. Prints
# HELD when the last 2 attempts for REMOTE_SHA both failed and it is still HEAD.
mkdir -p "$STATE_DIR"
HOLD=$(STATUS_FILE="$STATUS_FILE" REMOTE_SHA="$REMOTE_SHA" CHECKED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)" python3 - <<'PY'
import os, json
p = os.environ["STATUS_FILE"]
sha = os.environ["REMOTE_SHA"]
now = os.environ["CHECKED_AT"]
try:
    data = json.load(open(p))
except Exception:
    data = {"latest_main": None, "held": None, "attempts": []}

data["latest_main"] = {"sha": sha, "checked_at": now}

# a NEW origin/main SHA clears any existing hold
held = data.get("held")
if held and held.get("sha") != sha:
    held = None

# count consecutive most-recent failed/rolled_back attempts for THIS sha
fails = 0
for a in data.get("attempts", []):
    if a.get("sha") != sha:
        continue
    if a.get("outcome") in ("failed", "rolled_back"):
        fails += 1
    else:
        break
hold = fails >= 2
if hold:
    since = held.get("since") if (held and held.get("sha") == sha) else now
    held = {"sha": sha, "reason": "2+ consecutive failed deploys", "since": since, "failures": fails}
data["held"] = held

tmp = p + ".tmp"
with open(tmp, "w") as f:
    json.dump(data, f, indent=2)
os.replace(tmp, p)
print("HELD" if hold else "OK")
PY
)

# already at HEAD — nothing to do (latest_main was still refreshed above)
if [ "$REMOTE_SHA" = "$CURRENT" ]; then
  exit 0
fi

if [ "$HOLD" = "HELD" ]; then
  echo "[$(date -u +%FT%TZ)] HELD: ${REMOTE_SHA} failed 2x — not retrying until a new commit lands"
  exit 0
fi

echo "[$(date -u +%FT%TZ)] new commit ${REMOTE_SHA} (deployed: ${CURRENT}) — deploying"
git -C "$SRC" reset --hard --quiet "$REMOTE_SHA"
if bash "$SRC/deploy/deploy-local.sh" --trigger=auto; then
  echo "$REMOTE_SHA" > "$STATE"
  echo "[$(date -u +%FT%TZ)] deployed ${REMOTE_SHA} OK"
else
  echo "[$(date -u +%FT%TZ)] deploy of ${REMOTE_SHA} FAILED — see the deployments page / next poll re-evaluates hold"
fi
