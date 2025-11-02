#!/bin/sh
set -euo pipefail

# 必須設定がないと即終了
: "${MYSQL_HOST:?required}"
: "${MYSQL_PORT:?required}"
: "${MYSQL_USER:?required}"
: "${MYSQL_PASSWORD:?required}"
: "${MYSQL_DATABASE:?required}"

# DBへの接続文字列を組み立て
DSN="mysql://${MYSQL_USER}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DATABASE}?charset=utf8mb4&parseTime=true&loc=Local"

# DB起動待ち（最大60秒）
for i in $(seq 1 30); do
  nc -z "${MYSQL_HOST}" "${MYSQL_PORT}" && break
  echo "[migrate] waiting for DB..."
  sleep 2
done

# マイグレーションを実行して終了
echo "[migrate] DB ready. running migrations..."
exec migrate -path /app/migrations -database "${DSN}" -lock-timeout 15 up