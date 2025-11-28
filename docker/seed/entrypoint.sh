#!/bin/sh
set -euo pipefail

# 必須環境変数チェック
: "${MYSQL_HOST:?required}"
: "${MYSQL_PORT:?required}"
: "${MYSQL_USER:?required}"
: "${MYSQL_PASSWORD:?required}"
: "${MYSQL_DATABASE:?required}"

# DB起動待機（最大60秒）
for i in $(seq 1 30); do
  nc -z "${MYSQL_HOST}" "${MYSQL_PORT}" && break
  echo "[seed] waiting for DB..."
  sleep 2
done

echo "[seed] running initial data loader..."
exec /usr/local/bin/seed