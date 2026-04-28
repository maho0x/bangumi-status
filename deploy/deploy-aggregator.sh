#!/usr/bin/env bash
# Deploy aggregator to Tokyo-1. TLS/routing is handled by Caddy.
# NOTE: /etc/bangumi-status/aggregator.env is managed manually on the server;
# this script only updates the binary and the systemd unit.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

BIN="$ROOT/dist/aggregator-linux-amd64"
[[ -f "$BIN" ]] || { echo "missing $BIN — run deploy/build.sh first"; exit 1; }

echo ">> copying aggregator to Tokyo-1"
ssh Tokyo-1 'mkdir -p /opt/bangumi-status /etc/bangumi-status'
scp -q "$BIN" Tokyo-1:/opt/bangumi-status/aggregator.new
scp -q "$ROOT/deploy/aggregator.service" Tokyo-1:/etc/systemd/system/bangumi-aggregator.service

ssh Tokyo-1 bash <<'REMOTE'
set -e
chmod 755 /opt/bangumi-status/aggregator.new
mv /opt/bangumi-status/aggregator.new /opt/bangumi-status/aggregator
systemctl daemon-reload
systemctl enable bangumi-aggregator.service >/dev/null 2>&1
systemctl restart bangumi-aggregator.service
sleep 1
systemctl --no-pager status bangumi-aggregator.service | head -10
echo
echo ">> local health:"
curl -sf http://172.17.0.1:18361/api/health | head -c 300; echo
REMOTE

echo
echo ">> aggregator deployed on Tokyo-1"
