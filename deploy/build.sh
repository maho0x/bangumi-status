#!/usr/bin/env bash
# Cross-compile aggregator (amd64) and probe (amd64 + arm64) into ./dist/
set -euo pipefail
cd "$(dirname "$0")/.."

mkdir -p dist
export CGO_ENABLED=0

echo ">> aggregator linux/amd64"
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/aggregator-linux-amd64 ./cmd/aggregator

echo ">> probe linux/amd64"
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/probe-linux-amd64 ./cmd/probe

echo ">> probe linux/arm64"
GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/probe-linux-arm64 ./cmd/probe

ls -la dist/
