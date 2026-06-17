#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC_DIR="$ROOT_DIR/src"
DIST_DIR="$ROOT_DIR/dist/epusdt-pure-gateway"
BIN_PATH="$DIST_DIR/epusdt"
ENV_PATH="$DIST_DIR/.env"
GATEWAY_PATH="$DIST_DIR/gateway.yaml"
SERVICE_PATH="$DIST_DIR/epusdt.service"
ARCHIVE_PATH="$ROOT_DIR/dist/epusdt-pure-gateway.tar.gz"
GO_CACHE_DIR="$ROOT_DIR/.gocache"
GO_TMP_DIR="$ROOT_DIR/.gotmp"

APP_URI="${APP_URI:-https://pay.example.com}"
HTTP_LISTEN="${HTTP_LISTEN:-0.0.0.0:8000}"
PAYMENT_URL_TEMPLATE="${PAYMENT_URL_TEMPLATE:-https://a.example.com/pay/{trade_id}}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/epusdt}"
RUN_USER="${RUN_USER:-root}"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR/data" "$DIST_DIR/runtime" "$DIST_DIR/logs"
mkdir -p "$GO_CACHE_DIR" "$GO_TMP_DIR"

cp "$SRC_DIR/gateway.yaml" "$GATEWAY_PATH"

cat >"$ENV_PATH" <<EOF
app_name=epusdt
app_uri=$APP_URI
log_level=info
http_access_log=true
sql_debug=false
http_listen=$HTTP_LISTEN

static_path=/static
runtime_root_path=./runtime

log_save_path=./logs
log_max_size=32
log_max_age=7
max_backups=3

db_type=sqlite
sqlite_database_filename=./data/epusdt.db
sqlite_table_prefix=

runtime_sqlite_filename=epusdt-runtime.db

queue_concurrency=10
queue_poll_interval_ms=1000
callback_retry_base_seconds=5

order_expiration_time=10
order_notice_max_retry=3

api_rate_url=

gateway_pure_mode=true
gateway_config=gateway.yaml
gateway_payment_url_template=$PAYMENT_URL_TEMPLATE

install=false
EOF

cat >"$SERVICE_PATH" <<EOF
[Unit]
Description=Epusdt Pure Gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$DEPLOY_DIR
ExecStart=$DEPLOY_DIR/epusdt --config $DEPLOY_DIR http start
Restart=always
RestartSec=3
User=$RUN_USER

[Install]
WantedBy=multi-user.target
EOF

pushd "$SRC_DIR" >/dev/null
env GOCACHE="$GO_CACHE_DIR" GOTMPDIR="$GO_TMP_DIR" go build -o "$BIN_PATH" .
popd >/dev/null

chmod +x "$BIN_PATH"

mkdir -p "$ROOT_DIR/dist"
tar -C "$ROOT_DIR/dist" -czf "$ARCHIVE_PATH" "$(basename "$DIST_DIR")"

echo "release dir: $DIST_DIR"
echo "archive:     $ARCHIVE_PATH"
