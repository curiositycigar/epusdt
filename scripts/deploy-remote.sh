#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARCHIVE_PATH="$ROOT_DIR/dist/epusdt-pure-gateway.tar.gz"

: "${DEPLOY_HOST:?DEPLOY_HOST is required, e.g. 1.2.3.4}"
: "${DEPLOY_USER:?DEPLOY_USER is required, e.g. root}"

DEPLOY_PORT="${DEPLOY_PORT:-22}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/epusdt}"
SERVICE_NAME="${SERVICE_NAME:-epusdt}"
REMOTE_TMP="${REMOTE_TMP:-/tmp/epusdt-pure-gateway.tar.gz}"
SUPERVISOR_CONF_DIR="${SUPERVISOR_CONF_DIR:-/etc/supervisord.d}"
SUPERVISOR_CONF_EXT="${SUPERVISOR_CONF_EXT:-.conf}"

"$ROOT_DIR/scripts/build-release.sh"

cat "$ARCHIVE_PATH" | ssh -p "$DEPLOY_PORT" "${DEPLOY_USER}@${DEPLOY_HOST}" "cat > \"$REMOTE_TMP\""

ssh -p "$DEPLOY_PORT" "${DEPLOY_USER}@${DEPLOY_HOST}" bash -s -- \
  "$DEPLOY_DIR" \
  "$SERVICE_NAME" \
  "$REMOTE_TMP" \
  "$SUPERVISOR_CONF_DIR" \
  "$SUPERVISOR_CONF_EXT" <<'EOF'
set -euo pipefail

DEPLOY_DIR="$1"
SERVICE_NAME="$2"
REMOTE_TMP="$3"
SUPERVISOR_CONF_DIR="$4"
SUPERVISOR_CONF_EXT="$5"

mkdir -p "$DEPLOY_DIR"
rm -rf "$DEPLOY_DIR.new"
mkdir -p "$DEPLOY_DIR.new"
tar -xzf "$REMOTE_TMP" -C "$DEPLOY_DIR.new"

INNER_DIR="$(find "$DEPLOY_DIR.new" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
if [ -z "$INNER_DIR" ]; then
  echo "release archive missing inner directory" >&2
  exit 1
fi

mkdir -p "$DEPLOY_DIR"
if [ -f "$DEPLOY_DIR/.env" ]; then cp "$DEPLOY_DIR/.env" "$DEPLOY_DIR/.env.bak"; fi
if [ -f "$DEPLOY_DIR/gateway.yaml" ]; then cp "$DEPLOY_DIR/gateway.yaml" "$DEPLOY_DIR/gateway.yaml.bak"; fi
if [ -d "$DEPLOY_DIR/data" ]; then mkdir -p "$INNER_DIR/data"; cp -R "$DEPLOY_DIR/data/." "$INNER_DIR/data/" || true; fi
if [ -d "$DEPLOY_DIR/runtime" ]; then mkdir -p "$INNER_DIR/runtime"; cp -R "$DEPLOY_DIR/runtime/." "$INNER_DIR/runtime/" || true; fi
if [ -d "$DEPLOY_DIR/logs" ]; then mkdir -p "$INNER_DIR/logs"; cp -R "$DEPLOY_DIR/logs/." "$INNER_DIR/logs/" || true; fi

rm -rf "$DEPLOY_DIR"/*
cp -R "$INNER_DIR/." "$DEPLOY_DIR/"
chmod +x "$DEPLOY_DIR/epusdt"
mkdir -p "$DEPLOY_DIR/logs"

mkdir -p "$SUPERVISOR_CONF_DIR"
CONF_PATH="$SUPERVISOR_CONF_DIR/$SERVICE_NAME$SUPERVISOR_CONF_EXT"
rm -f "$SUPERVISOR_CONF_DIR/$SERVICE_NAME.conf" "$SUPERVISOR_CONF_DIR/$SERVICE_NAME.ini"
cp "$DEPLOY_DIR/epusdt.supervisor.conf" "$CONF_PATH"

echo "supervisor conf: $CONF_PATH"
supervisorctl reread
supervisorctl update
supervisorctl restart "$SERVICE_NAME" || supervisorctl start "$SERVICE_NAME"
STATUS="$(supervisorctl status "$SERVICE_NAME" || true)"
echo "$STATUS"
if echo "$STATUS" | grep -qE 'BACKOFF|FATAL|EXITED'; then
  echo "==== epusdt stderr ===="
  tail -n 100 "$DEPLOY_DIR/logs/epusdt.err.log" || true
  echo "==== epusdt stdout ===="
  tail -n 100 "$DEPLOY_DIR/logs/epusdt.log" || true
  exit 1
fi

rm -rf "$DEPLOY_DIR.new"
rm -f "$REMOTE_TMP"
EOF
