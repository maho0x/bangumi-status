#!/usr/bin/env bash
# Deploy aggregator to the aggregator host. TLS/routing is handled by the
# server's reverse proxy.
# NOTE: /etc/bangumi-status/aggregator.env is managed manually on the host;
# this script only updates the binary and the systemd unit.
#
# By default the aggregator runs on THIS machine, so deployment is local:
# swap the binary + unit file and restart the systemd service in place.
# To target a remote host instead, set AGGREGATOR_HOST=user@host (SSH/SCP).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

AGGREGATOR_HOST="${AGGREGATOR_HOST:-local}"
AGGREGATOR_SSH_PORT="${AGGREGATOR_SSH_PORT:-15000}"
AGGREGATOR_SSH_KEY="${AGGREGATOR_SSH_KEY:-}"

BIN="$ROOT/dist/aggregator-linux-amd64"
[[ -f "$BIN" ]] || { echo "missing $BIN — run deploy/build.sh first"; exit 1; }
UNIT="$ROOT/deploy/aggregator.service"

# The post-copy steps are identical local vs remote: install the staged binary,
# reload systemd, (re)start the service, then hit the local health endpoint.
read -r -d '' RESTART <<'REMOTE' || true
set -e
chmod 755 /opt/bangumi-status/aggregator.new
mv /opt/bangumi-status/aggregator.new /opt/bangumi-status/aggregator
systemctl daemon-reload
systemctl enable bangumi-aggregator.service >/dev/null 2>&1
systemctl restart bangumi-aggregator.service
sleep 2
systemctl --no-pager status bangumi-aggregator.service | head -10
echo
echo ">> local health:"
curl -sf http://172.17.0.1:18361/api/health | head -c 300; echo
REMOTE

if [[ "$AGGREGATOR_HOST" == "local" || "$AGGREGATOR_HOST" == "localhost" ]]; then
  echo ">> deploying aggregator locally"
  mkdir -p /opt/bangumi-status /etc/bangumi-status
  install -m 755 "$BIN" /opt/bangumi-status/aggregator.new
  install -m 644 "$UNIT" /etc/systemd/system/bangumi-aggregator.service
  bash -c "$RESTART"
  echo
  echo ">> aggregator deployed locally"
  exit 0
fi

SSH=(ssh -p "$AGGREGATOR_SSH_PORT")
SCP=(scp -P "$AGGREGATOR_SSH_PORT" -q)
if [[ -n "$AGGREGATOR_SSH_KEY" ]]; then
  SSH+=(-i "$AGGREGATOR_SSH_KEY")
  SCP+=(-i "$AGGREGATOR_SSH_KEY")
fi

echo ">> copying aggregator to $AGGREGATOR_HOST"
"${SSH[@]}" "$AGGREGATOR_HOST" 'mkdir -p /opt/bangumi-status /etc/bangumi-status'
"${SCP[@]}" "$BIN" "$AGGREGATOR_HOST":/opt/bangumi-status/aggregator.new
"${SCP[@]}" "$UNIT" "$AGGREGATOR_HOST":/etc/systemd/system/bangumi-aggregator.service

"${SSH[@]}" "$AGGREGATOR_HOST" bash <<REMOTE
$RESTART
REMOTE

echo
echo ">> aggregator deployed on $AGGREGATOR_HOST"
