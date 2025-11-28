# ─────────────────────────────
# 初期データ投入専用のコンテナ
# ─────────────────────────────
FROM golang:1.25.2-alpine3.22 AS builder

# ソースコードをビルドする作業ディレクトリ
WORKDIR /src

# 依存関係を取得（キャッシュを効かせるため最初に分離）
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# アプリケーションコードをコピー
COPY backend/ .

# 初期データ投入処理（cmd/seed/main.go）をビルド
RUN go build -o /out/seed ./cmd/seed

# ─────────────────────────────
# 実行用ステージ（軽量イメージ）
# ─────────────────────────────
FROM alpine:3.20

# TLS検証とDB待機に必要な最低限のユーティリティのみ導入
RUN apk add --no-cache ca-certificates netcat-openbsd

# 実行時の作業ディレクトリ
WORKDIR /app

# ビルド済みのバイナリだけをコピー（最小構成でデプロイ）
COPY --from=builder /out/seed /usr/local/bin/seed

# 起動時スクリプトを配置
COPY docker/seed/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh
RUN sed -i 's/\r$//' /usr/local/bin/entrypoint.sh

# コンテナ起動時は entrypoint.sh を実行
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]