FROM golang:1.25.2-alpine3.22

# コンテナ内の作業ディレクトリを/appに設定
WORKDIR /app

# airインストール
RUN go install github.com/air-verse/air@latest

# COPY go.mod go.sum ./
# RUN go mod download

# アプリケーションコードをコピー
COPY ./app /app

# Goアプリケーションを実行
CMD ["air"]