#!/bin/bash
set -e

# 設定
REGISTRY="asia-northeast1-docker.pkg.dev/keywars-477702/cloudrun-repo"
PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)" # ルートディレクトリのパスを取得

# 確認プロンプト
confirm_step() {
  local step_name=$1 # 1つ目の引数
  local default=${2:-yes}  # 2つ目の引数
  
  if [ "$default" = "yes" ]; then
    read -p "${step_name} (Y/n): " response
    response=${response:-yes}
  else
    read -p "${step_name} (y/N): " response
    response=${response:-no}
  fi
  
  [ "$response" = "yes" ] || [ "$response" = "y" ] # Yesかyならtrue
}

# フロントエンドビルドファイルとFireBase用ファイルを同期
echo "Frontend Build file Phase"
if confirm_step "Copy frontend build files to firebase public?"; then
    # クリーンアップ
    rm -rf "$PROJECT_ROOT/infra/firebase/public"/*

    # index.htmlとassetsをコピー
    rsync -av "$PROJECT_ROOT/frontend/dist/index.html" "$PROJECT_ROOT/infra/firebase/public/"
    rsync -av "$PROJECT_ROOT/frontend/dist/assets/" "$PROJECT_ROOT/infra/firebase/public/assets/"
    
    # imgをpublic/imgにコピー
    mkdir -p "$PROJECT_ROOT/infra/firebase/public/public"
    rsync -av "$PROJECT_ROOT/frontend/dist/img/" "$PROJECT_ROOT/infra/firebase/public/public/img/"
    echo "Firebase hosting files are now update"
else
    echo "Skipped"
fi
echo ""

# バックエンドイメージプッシュ
echo "Backend images push Phase"
if confirm_step "Push backend docker images to artifact registry?"; then
    docker push "${REGISTRY}/api-image:latest"
    docker push "${REGISTRY}/ws-image:latest"
    docker push "${REGISTRY}/migration-image:latest"
    docker push "${REGISTRY}/seed-image:latest"
    echo "Images push to artifact registry"
else
    echo "Skipped"
fi
echo ""

echo "Push completed!"
