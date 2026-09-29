#!/usr/bin/env bash
# Build the frontend, then cross-compile aggregator (amd64) and probe
# (amd64 + arm64) into ./dist/. The aggregator embeds web/dist.
set -euo pipefail
cd "$(dirname "$0")/.."

echo ">> web"
(cd web && npm ci --no-fund --no-audit && npm run build)

mkdir -p dist
export CGO_ENABLED=0
build() { GOOS=linux GOARCH="$2" go build -trimpath -ldflags "-s -w" -o "dist/$1-linux-$2" "./cmd/$1"; echo ">> $1 linux/$2"; }
build aggregator amd64
build probe amd64
build probe arm64

ls -la dist/
