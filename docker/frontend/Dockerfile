FROM node:22-slim

# 作業ディレクトリ
WORKDIR /app

COPY frontend/ .

# ビルドに必要なツールを追加
RUN apt-get update && apt-get install -y \
    build-essential python3 g++ make \
    && rm -rf /var/lib/apt/lists/* 

RUN corepack enable

# node_modules が残っている場合は削除してインストール
RUN yarn install --frozen-lockfile


# 開発用コマンド
CMD ["yarn", "run", "dev"]
