#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC_DIR="$ROOT_DIR/src"
RUN_DIR="$ROOT_DIR/output-pure-gateway"
BIN_PATH="$RUN_DIR/epusdt"
ENV_PATH="$RUN_DIR/.env"
GATEWAY_PATH="$RUN_DIR/gateway.yaml"
DATA_DIR="$RUN_DIR/data"
RUNTIME_DIR="$RUN_DIR/runtime"
GO_CACHE_DIR="$ROOT_DIR/.gocache"
GO_TMP_DIR="$ROOT_DIR/.gotmp"

APP_URI="${APP_URI:-http://127.0.0.1:8000}"
HTTP_LISTEN="${HTTP_LISTEN:-127.0.0.1:8000}"
PAYMENT_URL_TEMPLATE="${PAYMENT_URL_TEMPLATE:-http://127.0.0.1:3000/pay/{trade_id}}"

mkdir -p "$RUN_DIR" "$DATA_DIR" "$RUNTIME_DIR"
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

pushd "$SRC_DIR" >/dev/null
env GOCACHE="$GO_CACHE_DIR" GOTMPDIR="$GO_TMP_DIR" go build -o "$BIN_PATH" .
popd >/dev/null

chmod +x "$BIN_PATH"

echo "binary: $BIN_PATH"
echo "env:    $ENV_PATH"
echo "yaml:   $GATEWAY_PATH"
echo "listen: $HTTP_LISTEN"

exec "$BIN_PATH" --config "$RUN_DIR" http start
