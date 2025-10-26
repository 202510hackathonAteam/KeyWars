# ベースイメージ：軽量な Alpine 上の Go 環境を使用
FROM golang:1.25.2-alpine3.22

# 作業ディレクトリを設定
WORKDIR /app

# モジュール依存関係を先にダウンロード（キャッシュを有効活用）
COPY go.mod go.sum ./
RUN go mod download

# ホットリロードツール air をインストール
RUN go install github.com/air-verse/air@latest

# ソースコードをコンテナ内にコピー
COPY . .

# DB 起動待機用スクリプトをコピーして実行権限を付与
COPY wait-for.sh /wait-for.sh
RUN chmod +x /wait-for.sh

# 環境変数設定とアプリ起動コマンド
ENV GO111MODULE=on
CMD ["/wait-for.sh", "db", "3306", "air", "-c", ".air.toml"]