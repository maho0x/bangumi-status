#!/usr/bin/env bash
# Deploy aggregator to Tokyo-1. TLS/routing is handled by Caddy.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

: "${INGEST_SECRET:?must be set}"

BIN="$ROOT/dist/aggregator-linux-amd64"
[[ -f "$BIN" ]] || { echo "missing $BIN — run deploy/build.sh first"; exit 1; }

ENV_FILE="$ROOT/dist/aggregator.env"
{
  echo "INGEST_SECRET=$INGEST_SECRET"
  [[ -n "${TELEGRAM_BOT_TOKEN:-}" ]] && echo "TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN"
  [[ -n "${TELEGRAM_CHAT_ID:-}" ]]   && echo "TELEGRAM_CHAT_ID=$TELEGRAM_CHAT_ID"
  [[ -n "${STATUS_PAGE_URL:-}" ]]    && echo "STATUS_PAGE_URL=$STATUS_PAGE_URL"
} > "$ENV_FILE"
chmod 600 "$ENV_FILE"

echo ">> copying aggregator to Tokyo-1"
ssh Tokyo-1 'mkdir -p /opt/bangumi-status /etc/bangumi-status'
scp -q "$BIN" Tokyo-1:/opt/bangumi-status/aggregator.new
scp -q "$ENV_FILE" Tokyo-1:/etc/bangumi-status/aggregator.env
scp -q "$ROOT/deploy/aggregator.service" Tokyo-1:/etc/systemd/system/bangumi-aggregator.service

ssh Tokyo-1 bash <<'REMOTE'
set -e
chmod 755 /opt/bangumi-status/aggregator.new
mv /opt/bangumi-status/aggregator.new /opt/bangumi-status/aggregator
chmod 600 /etc/bangumi-status/aggregator.env
systemctl daemon-reload
systemctl enable bangumi-aggregator.service >/dev/null 2>&1
systemctl restart bangumi-aggregator.service
sleep 1
systemctl --no-pager status bangumi-aggregator.service | head -10
echo
echo ">> local health:"
curl -sf http://172.17.0.1:18361/api/health | head -c 300; echo
REMOTE

rm -f "$ENV_FILE"
echo
echo ">> aggregator deployed on Tokyo-1"
