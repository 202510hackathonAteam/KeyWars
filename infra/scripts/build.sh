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
    read -p "Run ${step_name}? (Y/n): " response
    response=${response:-yes}
  else
    read -p "Run ${step_name}? (y/N): " response
    response=${response:-no}
  fi
  
  [ "$response" = "yes" ] || [ "$response" = "y" ] # Yesかyならtrue
}

# フロントエンドビルド
echo "Frontend Build Phase"
if confirm_step "Frontend Build"; then
    cd "$PROJECT_ROOT"
    docker compose run --rm frontend yarn vite build
    echo "Frontend build completed"
else
    echo "Skipped"
fi
echo ""


# バックエンドビルド
echo "Backend Build Phase"
if confirm_step "Backend Build"; then
    # イメージ名とDockerfileのパスを配列で定義
    local images=(
        "api-image:$PROJECT_ROOT/docker/backend/Dockerfile.prod"
        "ws-image:$PROJECT_ROOT/docker/backend/Dockerfile.prod"
        "migration-image:$PROJECT_ROOT/docker/migrate/Dockerfile"
        "seed-image:$PROJECT_ROOT/docker/seed/Dockerfile"
    )
        
    for image in "${images[@]}"; do
        local image_name="${image%%:*}" # keyを参照
        local dockerfile="${image#*:}" # valueを参照
        
        echo "Building ${image_name}..."
        docker build \
            -f "$dockerfile" \
            -t "${REGISTRY}/${image_name}:latest" \
            "$PROJECT_ROOT"
    done
else
    echo "Skipped"
fi
echo ""

echo "Build completed!"
