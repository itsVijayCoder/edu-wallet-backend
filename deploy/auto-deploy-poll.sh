#!/usr/bin/env bash
# Poll GitHub main and redeploy eduwallet when new commits land.
# Team workflow: push/merge to main on github.com/susanoox/edu-wallet-backend →
# this poller (LaunchAgent com.udhay.eduwallet-autodeploy, every 5 min) picks it
# up and runs deploy/deploy-local.sh. No jarvis access needed for the team.
#
# Uses a dedicated clean clone (/Users/jarvis/eduwallet-src) so the developer
# working repo on this box is never touched. State file records the last
# successfully deployed SHA; failures are retried on the next poll (see the log:
# ~/Library/Logs/eduwallet-autodeploy.out.log).
set -euo pipefail
export PATH="/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin"
export GIT_TERMINAL_PROMPT=0

SRC="/Users/jarvis/eduwallet-src"
REPO_URL="https://github.com/susanoox/edu-wallet-backend"
BRANCH="main"
STATE="/Users/jarvis/eduwallet/.deployed-sha"
LOCK="/tmp/eduwallet-autodeploy.lock"

mkdir "$LOCK" 2>/dev/null || { echo "another run in progress — skipping"; exit 0; }
trap 'rmdir "$LOCK"' EXIT

if [ ! -d "$SRC/.git" ]; then
  git clone --branch "$BRANCH" "$REPO_URL" "$SRC"
fi
git -C "$SRC" fetch --quiet origin "$BRANCH"
REMOTE_SHA=$(git -C "$SRC" rev-parse "origin/$BRANCH")
CURRENT=$(cat "$STATE" 2>/dev/null || echo none)
[ "$REMOTE_SHA" = "$CURRENT" ] && exit 0

echo "[$(date -u +%FT%TZ)] new commit ${REMOTE_SHA} (deployed: ${CURRENT}) — deploying"
git -C "$SRC" reset --hard --quiet "$REMOTE_SHA"
bash "$SRC/deploy/deploy-local.sh"
echo "$REMOTE_SHA" > "$STATE"
echo "[$(date -u +%FT%TZ)] deployed ${REMOTE_SHA} OK"
