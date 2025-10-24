#!/bin/sh
set -eu

host="$1"
port="$2"

echo "Waiting for $host:$port to be ready..."

while ! nc -z "$host" "$port" >/dev/null 2>&1; do
  sleep 1
done

echo "Ready: $host:$port"
shift 2
exec "$@"