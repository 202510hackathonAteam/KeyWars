# ベースイメージ：軽量な Alpine 上の Go 環境を使用
FROM golang:1.25.2-alpine3.22

# 作業ディレクトリを設定
WORKDIR /app

# モジュール依存関係を先にダウンロード（キャッシュを有効活用）
COPY backend/go.mod backend/go.sum* ./
RUN go mod download

# ホットリロードツール air をインストール
RUN go install github.com/air-verse/air@latest

# ソースコードをコンテナ内にコピー
COPY backend/ .

# DB 起動待機用スクリプトをコピーして実行権限を付与
COPY docker/backend/wait-for.sh /wait-for.sh
RUN sed -i 's/\r$//' /wait-for.sh
RUN chmod +x /wait-for.sh

# アプリ起動コマンド
CMD ["/wait-for.sh", "db", "3306", "air", "-c", "/app/.air.toml"]