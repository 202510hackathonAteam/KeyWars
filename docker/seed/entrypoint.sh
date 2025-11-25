#!/bin/sh
set -euo pipefail

# 必須環境変数チェック
: "${MYSQL_HOST:?required}"
: "${MYSQL_PORT:?required}"
: "${MYSQL_USER:?required}"
: "${MYSQL_PASSWORD:?required}"
: "${MYSQL_DATABASE:?required}"

# DB起動待機
for i in $(seq 1 120); do
  mysqladmin ping -h "${MYSQL_HOST}" -u "${MYSQL_USER}" -p"${MYSQL_PASSWORD}" --silent && break
  echo "[seed] waiting for MySQL to be ready..."
  sleep 2
done

echo "[seed] running initial data loader..."
exec /usr/local/bin/seed