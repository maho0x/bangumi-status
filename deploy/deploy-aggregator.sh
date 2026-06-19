#!/usr/bin/env bash
# Deploy aggregator to the production aggregator host. TLS/routing is handled
# by the server's reverse proxy.
# NOTE: /etc/bangumi-status/aggregator.env is managed manually on the server;
# this script only updates the binary and the systemd unit.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

AGGREGATOR_HOST="${AGGREGATOR_HOST:-root@52.76.255.181}"
AGGREGATOR_SSH_PORT="${AGGREGATOR_SSH_PORT:-15000}"
AGGREGATOR_SSH_KEY="${AGGREGATOR_SSH_KEY:-}"

SSH=(ssh -p "$AGGREGATOR_SSH_PORT")
SCP=(scp -P "$AGGREGATOR_SSH_PORT" -q)
if [[ -n "$AGGREGATOR_SSH_KEY" ]]; then
  SSH+=(-i "$AGGREGATOR_SSH_KEY")
  SCP+=(-i "$AGGREGATOR_SSH_KEY")
fi

BIN="$ROOT/dist/aggregator-linux-amd64"
[[ -f "$BIN" ]] || { echo "missing $BIN — run deploy/build.sh first"; exit 1; }

echo ">> copying aggregator to $AGGREGATOR_HOST"
"${SSH[@]}" "$AGGREGATOR_HOST" 'mkdir -p /opt/bangumi-status /etc/bangumi-status'
"${SCP[@]}" "$BIN" "$AGGREGATOR_HOST":/opt/bangumi-status/aggregator.new
"${SCP[@]}" "$ROOT/deploy/aggregator.service" "$AGGREGATOR_HOST":/etc/systemd/system/bangumi-aggregator.service

"${SSH[@]}" "$AGGREGATOR_HOST" bash <<'REMOTE'
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
echo ">> aggregator deployed on $AGGREGATOR_HOST"
