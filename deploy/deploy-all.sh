#!/usr/bin/env bash
# One-shot: build + deploy aggregator to Eden + deploy probe to all 6 hosts (including Eden itself).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

: "${INGEST_SECRET:?set an INGEST_SECRET (any long random string, same value everywhere)}"

./deploy/build.sh
./deploy/deploy-aggregator.sh

# Give Caddy a moment to get certs. Tolerate cold-start errors and retry.
echo ">> waiting for https://bgm-status.ry.mk/api/health to answer"
for i in 1 2 3 4 5 6 7 8; do
  if curl -sf -m 8 https://bgm-status.ry.mk/api/health >/dev/null; then
    echo "OK"
    break
  fi
  sleep 8
done

# probe-id, region, arch per host
./deploy/deploy-probe.sh Eden        hong-kong-1 hk amd64
./deploy/deploy-probe.sh Tokyo-1     tokyo-1     jp amd64
./deploy/deploy-probe.sh Osaka-1     osaka-1     jp amd64
./deploy/deploy-probe.sh Osaka-2     osaka-2     jp amd64
./deploy/deploy-probe.sh Osaka-3     osaka-3     jp arm64
./deploy/deploy-probe.sh Shenzhen-1  shenzhen-1  cn amd64

echo ">> all probes deployed"
echo ">> visit https://bgm-status.ry.mk — first probe data should land within 60s"
