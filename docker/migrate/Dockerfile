FROM alpine:3.20

# migrate 実行時に必要な最低限のツール類（TLS検証やヘルスチェック用）
RUN apk add --no-cache ca-certificates curl netcat-openbsd

# golang-migrate バイナリを取得
ARG MIGRATE_VER=4.17.1
ARG TARGETARCH

RUN set -eux; \
  case "${TARGETARCH:-amd64}" in \
    amd64) ARCH=amd64 ;; \
    arm64) ARCH=arm64 ;; \
    armv7|arm) ARCH=armv6 ;; \
    *) echo "unsupported arch: ${TARGETARCH}"; exit 1 ;; \
  esac; \
  URL="https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VER}/migrate.linux-${ARCH}.tar.gz"; \
  echo "Downloading: $URL"; \
  curl -fsSL "$URL" -o /tmp/migrate.tgz; \
  tar -xzf /tmp/migrate.tgz -C /usr/local/bin; \
  mv /usr/local/bin/migrate.linux-* /usr/local/bin/migrate || true; \
  chmod +x /usr/local/bin/migrate; \
  rm -f /tmp/migrate.tgz; \
  /usr/local/bin/migrate -version

# マイグレーションSQLをイメージに内包(本番環境で「外部マウントなし」で動作するようにするために)
WORKDIR /app
COPY backend/migrations /app/migrations

# エントリーポイントスクリプトで migrate 実行を制御
COPY docker/migrate/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh
RUN sed -i 's/\r$//' /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]