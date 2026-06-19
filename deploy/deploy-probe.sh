#!/usr/bin/env bash
# Deploy probe binary + systemd unit to a remote host.
# Usage: deploy-probe.sh <ssh-host> <probe-id> <region> [arch]
#   arch defaults to amd64. Use arm64 for Osaka-3.
# Optional env: PROBE_SSH_PORT=15000 PROBE_SSH_KEY=/path/to/key
set -euo pipefail

HOST="${1:?ssh host required, e.g. Tokyo-1}"
PID="${2:?probe id required, e.g. tokyo-1}"
REGION="${3:?region required: ISO 3166-1 alpha-2, e.g. jp|cn|sg}"
ARCH="${4:-amd64}"
PROBE_SSH_PORT="${PROBE_SSH_PORT:-22}"
PROBE_SSH_KEY="${PROBE_SSH_KEY:-}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/dist/probe-linux-$ARCH"
[[ -f "$BIN" ]] || { echo "missing $BIN — run deploy/build.sh first"; exit 1; }

INFO="$ROOT/info"
[[ -f "$INFO" ]] || { echo "missing $INFO (Bangumi credentials)"; exit 1; }

ENV_FILE="$ROOT/dist/probe.env.$PID"

AGG_URL="${AGG_URL:-https://bgm-status.ry.mk/api/ingest}"
if [[ -z "${INGEST_SECRET:-}" ]]; then
  echo "INGEST_SECRET must be set in environment"; exit 1
fi

SSH=(ssh -p "$PROBE_SSH_PORT")
SCP=(scp -P "$PROBE_SSH_PORT" -q)
if [[ -n "$PROBE_SSH_KEY" ]]; then
  SSH+=(-i "$PROBE_SSH_KEY")
  SCP+=(-i "$PROBE_SSH_KEY")
fi

# Extract values from info (careful: cookie JSON has = signs)
BGM_USER_AGENT=$(grep '^BGM_USER_AGENT=' "$INFO" | cut -d= -f2-)
BGM_COOKIE_JSON=$(grep '^BGM_COOKIE_JSON=' "$INFO" | cut -d= -f2-)
BGM_API_TOKEN=$(grep '^BGM_API_TOKEN=' "$INFO" | cut -d= -f2-)

# Convert cookie JSON to Cookie header string via python (pre-computed locally).
BGM_COOKIE=$(python3 -c "
import json,sys
cs=json.loads(sys.argv[1])
print('; '.join(f\"{c['name']}={c['value']}\" for c in cs))
" "$BGM_COOKIE_JSON")

printf 'PROBE_ID=%s\nREGION=%s\nAGG_URL=%s\nINGEST_SECRET=%s\nBGM_USER_AGENT=%s\nBGM_COOKIE=%s\nBGM_API_TOKEN=%s\n' \
  "$PID" "$REGION" "$AGG_URL" "$INGEST_SECRET" "$BGM_USER_AGENT" "$BGM_COOKIE" "$BGM_API_TOKEN" \
  > "$ENV_FILE"
chmod 600 "$ENV_FILE"

echo ">> copying probe binary + env to $HOST"
"${SSH[@]}" "$HOST" 'mkdir -p /opt/bangumi-status /etc/bangumi-status'
"${SCP[@]}" "$BIN" "$HOST":/opt/bangumi-status/probe.new
"${SCP[@]}" "$ENV_FILE" "$HOST":/etc/bangumi-status/probe.env
"${SCP[@]}" "$ROOT/deploy/probe.service" "$HOST":/etc/systemd/system/bangumi-probe.service

"${SSH[@]}" "$HOST" bash <<'REMOTE'
set -e
chmod 755 /opt/bangumi-status/probe.new
mv /opt/bangumi-status/probe.new /opt/bangumi-status/probe
chmod 600 /etc/bangumi-status/probe.env
systemctl daemon-reload
systemctl enable bangumi-probe.service
systemctl restart bangumi-probe.service
sleep 2
systemctl --no-pager status bangumi-probe.service | head -12
REMOTE

rm -f "$ENV_FILE"
echo ">> $HOST OK"
