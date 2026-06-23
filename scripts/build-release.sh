#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC_DIR="$ROOT_DIR/src"
DIST_DIR="$ROOT_DIR/dist/epusdt-pure-gateway"
BIN_PATH="$DIST_DIR/epusdt"
ENV_PATH="$DIST_DIR/.env"
GATEWAY_PATH="$DIST_DIR/gateway.yaml"
SUPERVISOR_CONF_PATH="$DIST_DIR/epusdt.supervisor.conf"
ARCHIVE_PATH="$ROOT_DIR/dist/epusdt-pure-gateway.tar.gz"
GO_CACHE_DIR="$ROOT_DIR/.gocache"
GO_TMP_DIR="$ROOT_DIR/.gotmp"
GO_MOD_CACHE_DIR="$ROOT_DIR/.gomodcache"
GOPROXY_VALUE="${GOPROXY:-https://proxy.golang.org,direct}"
GOSUMDB_VALUE="${GOSUMDB:-sum.golang.org}"
GO_DOWNLOAD_RETRIES="${GO_DOWNLOAD_RETRIES:-3}"

APP_URI="${APP_URI:-https://pay.example.com}"
HTTP_LISTEN="${HTTP_LISTEN:-127.0.0.1:8325}"
PAYMENT_URL_TEMPLATE="${PAYMENT_URL_TEMPLATE:-https://a.example.com/pay/{trade_id}}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/epusdt}"
SERVICE_NAME="${SERVICE_NAME:-epusdt}"
RUN_USER="${RUN_USER:-root}"
SUPERVISOR_LOG_DIR="${SUPERVISOR_LOG_DIR:-$DEPLOY_DIR/logs}"
TARGET_GOOS="${TARGET_GOOS:-linux}"
TARGET_GOARCH="${TARGET_GOARCH:-amd64}"

retry() {
  local attempts="$1"
  shift
  local n=1
  until "$@"; do
    if [ "$n" -ge "$attempts" ]; then
      return 1
    fi
    echo "command failed, retrying ($n/$attempts): $*"
    n=$((n + 1))
    sleep 2
  done
}

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR/data" "$DIST_DIR/runtime" "$DIST_DIR/logs"
mkdir -p "$GO_CACHE_DIR" "$GO_TMP_DIR" "$GO_MOD_CACHE_DIR"

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

cat >"$SUPERVISOR_CONF_PATH" <<EOF
[program:$SERVICE_NAME]
command=$DEPLOY_DIR/epusdt --config $DEPLOY_DIR http start
directory=$DEPLOY_DIR
user=$RUN_USER
autostart=true
autorestart=true
startsecs=3
startretries=10
stdout_logfile=$SUPERVISOR_LOG_DIR/epusdt.log
stderr_logfile=$SUPERVISOR_LOG_DIR/epusdt.err.log
stopasgroup=true
killasgroup=true
EOF

pushd "$SRC_DIR" >/dev/null
retry "$GO_DOWNLOAD_RETRIES" env \
  GOCACHE="$GO_CACHE_DIR" \
  GOTMPDIR="$GO_TMP_DIR" \
  GOMODCACHE="$GO_MOD_CACHE_DIR" \
  GOPROXY="$GOPROXY_VALUE" \
  GOSUMDB="$GOSUMDB_VALUE" \
  go mod download

env \
  GOCACHE="$GO_CACHE_DIR" \
  GOTMPDIR="$GO_TMP_DIR" \
  GOMODCACHE="$GO_MOD_CACHE_DIR" \
  GOPROXY="$GOPROXY_VALUE" \
  GOSUMDB="$GOSUMDB_VALUE" \
  GOOS="$TARGET_GOOS" \
  GOARCH="$TARGET_GOARCH" \
  CGO_ENABLED=0 \
  go build -o "$BIN_PATH" .
popd >/dev/null

chmod +x "$BIN_PATH"

mkdir -p "$ROOT_DIR/dist"
TAR_FLAGS=(-C "$ROOT_DIR/dist" -czf "$ARCHIVE_PATH" "$(basename "$DIST_DIR")")
if tar --help 2>/dev/null | grep -q -- '--no-mac-metadata'; then
  env COPYFILE_DISABLE=1 COPY_EXTENDED_ATTRIBUTES_DISABLE=1 tar --no-mac-metadata "${TAR_FLAGS[@]}"
else
  env COPYFILE_DISABLE=1 COPY_EXTENDED_ATTRIBUTES_DISABLE=1 tar "${TAR_FLAGS[@]}"
fi

echo "release dir: $DIST_DIR"
echo "archive:     $ARCHIVE_PATH"
echo "target:      ${TARGET_GOOS}/${TARGET_GOARCH}"
