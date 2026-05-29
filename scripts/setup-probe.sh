#!/usr/bin/env bash
# setup-probe.sh — install the bangumi-status probe on the LOCAL machine.
# Intended for third-party operators who want to contribute a probe node.
#
# Run as root on the host that will be the probe (NOT the aggregator host).
#
# Required env:
#   PROBE_ID         e.g. paris-1   (assigned by the maintainer)
#   REGION           ISO 3166-1 alpha-2 country code (lowercase), e.g. jp / cn / us / de / fr
#   AGG_URL          ingest URL, e.g. https://bgm-status.ry.mk/api/ingest
#   INGEST_SECRET    per-probe token issued by the maintainer
#
# Optional env (omit both to run a guest-only probe — no Auth checks):
#   BGM_COOKIE       full Cookie header value for bangumi.tv (signed-in)
#   BGM_API_TOKEN    bearer token from https://next.bgm.tv/demo/access-token
#   BGM_USER_AGENT   custom UA string (default: a generic browser UA)
#   PROBE_BIN        path to a pre-built probe binary (default: ./probe)
#   INTERVAL         check interval, e.g. 60s (default: 60s)
set -euo pipefail

: "${PROBE_ID:?PROBE_ID is required}"
: "${REGION:?REGION is required}"
: "${AGG_URL:?AGG_URL is required}"
: "${INGEST_SECRET:?INGEST_SECRET is required}"

BGM_USER_AGENT="${BGM_USER_AGENT:-Mozilla/5.0 (X11; Linux x86_64) bangumi-status-probe}"
INTERVAL="${INTERVAL:-60s}"
PROBE_BIN="${PROBE_BIN:-./probe}"

if [[ $EUID -ne 0 ]]; then
  echo "must run as root (writes /etc/bangumi-status and /etc/systemd/system)"
  exit 1
fi

if [[ ! -x "$PROBE_BIN" ]]; then
  echo "probe binary not found at $PROBE_BIN"
  echo "build it with: GOOS=linux GOARCH=\$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') \\"
  echo "              go build -trimpath -ldflags '-s -w' -o probe ./cmd/probe"
  exit 1
fi

install -d -m 0755 /opt/bangumi-status
install -d -m 0750 /etc/bangumi-status
install -m 0755 "$PROBE_BIN" /opt/bangumi-status/probe

ENV_FILE=/etc/bangumi-status/probe.env
umask 077
{
  printf 'PROBE_ID=%s\n'      "$PROBE_ID"
  printf 'REGION=%s\n'        "$REGION"
  printf 'AGG_URL=%s\n'       "$AGG_URL"
  printf 'INGEST_SECRET=%s\n' "$INGEST_SECRET"
  printf 'BGM_USER_AGENT=%s\n' "$BGM_USER_AGENT"
  if [[ -n "${BGM_COOKIE:-}" ]]; then
    printf 'BGM_COOKIE=%s\n' "$BGM_COOKIE"
  fi
  if [[ -n "${BGM_API_TOKEN:-}" ]]; then
    printf 'BGM_API_TOKEN=%s\n' "$BGM_API_TOKEN"
  fi
} > "$ENV_FILE"
chmod 600 "$ENV_FILE"

cat > /etc/systemd/system/bangumi-probe.service <<EOF
[Unit]
Description=Bangumi Status Probe (third-party node)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
DynamicUser=yes
EnvironmentFile=/etc/bangumi-status/probe.env
ExecStart=/opt/bangumi-status/probe -interval=${INTERVAL}
Restart=on-failure
RestartSec=10s
StandardOutput=journal
StandardError=journal

NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
PrivateDevices=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectKernelLogs=yes
ProtectControlGroups=yes
ProtectClock=yes
ProtectHostname=yes
ProtectProc=invisible
RestrictAddressFamilies=AF_INET AF_INET6
RestrictNamespaces=yes
RestrictRealtime=yes
RestrictSUIDSGID=yes
LockPersonality=yes
MemoryDenyWriteExecute=yes
SystemCallArchitectures=native
SystemCallFilter=@system-service
SystemCallFilter=~@privileged @resources
CapabilityBoundingSet=
AmbientCapabilities=
UMask=0077

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable bangumi-probe.service
systemctl restart bangumi-probe.service
sleep 2
systemctl --no-pager status bangumi-probe.service | head -15

echo
echo ">> probe installed. Logs: journalctl -u bangumi-probe -f"
