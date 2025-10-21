FROM node:22-slim

# 作業ディレクトリ
WORKDIR /app

# ビルドに必要なツールを追加
RUN apt-get update && apt-get install -y \
    build-essential python3 g++ make \
    && rm -rf /var/lib/apt/lists/* 

RUN corepack enable

# # package.json と lock をコピーして依存インストール
COPY package.json yarn.lock ./

# node_modules が残っている場合は削除してインストール
RUN yarn install --frozen-lockfile

# アプリケーションコードをコピー
COPY . .

# 開発用コマンド
CMD ["yarn", "run", "dev"]
