#!/bin/bash
set -e

# 設定
REGISTRY="asia-northeast1-docker.pkg.dev/keywars-477702/cloudrun-repo"
PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)" # ルートディレクトリのパスを取得

# 確認プロンプト
confirm() {
    local message=$1 # 確認時のメッセージを引数に持つ
    echo -n "${message} (y/n) [y]: "
    read -r answer
    # 空入力（Enter）またはyの場合は実行
    if [ -z "$answer" ] || [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
        return 0
    fi
    echo "Skipped"
    return 1
}

# フロントエンドビルド
build_frontend() {
    echo "[1/3] Building Frontend"
    if confirm "Build frontend?"; then
        cd "$PROJECT_ROOT"
        docker compose run --rm frontend yarn vite build
        echo "Frontend build completed"
    fi
    echo ""
}

# バックエンドビルド
build_backend() {
    echo "[2/3] Building Backend Images"
    if confirm "Build backend images?"; then
        # APIイメージ
        echo "Building api-image..."
        docker build \
            -f "$PROJECT_ROOT/docker/backend/Dockerfile.prod" \
            -t "${REGISTRY}/api-image:latest" \
            "$PROJECT_ROOT"
        
        # WebSocketイメージ
        echo "Building ws-image..."
        docker build \
            -f "$PROJECT_ROOT/docker/backend/Dockerfile.prod" \
            -t "${REGISTRY}/ws-image:latest" \
            "$PROJECT_ROOT"
        
        # Migrationイメージ
        echo "Building migration-image..."
        docker build \
            -f "$PROJECT_ROOT/docker/migrate/Dockerfile" \
            -t "${REGISTRY}/migration-image:latest" \
            "$PROJECT_ROOT"
        
        # Seedイメージ
        echo "Building seed-image..."
        docker build \
            -f "$PROJECT_ROOT/docker/seed/Dockerfile" \
            -t "${REGISTRY}/seed-image:latest" \
            "$PROJECT_ROOT"
        
        echo "Backend images build completed"
    fi
    echo ""
}

# イメージプッシュ
push_images() {
    echo "[3/3] Pushing Images"
    if confirm "Push images to registry?"; then
        docker push "${REGISTRY}/api-image:latest"
        docker push "${REGISTRY}/ws-image:latest"
        docker push "${REGISTRY}/migration-image:latest"
        docker push "${REGISTRY}/seed-image:latest"
        echo "Images pushed successfully"
    fi
    echo ""
}

# メイン処理
main() {
    echo "Build Pipeline Started"
    echo "Registry: ${REGISTRY}"
    echo "Project Root: ${PROJECT_ROOT}"
    echo ""
    
    build_frontend
    build_backend
    push_images
    
    echo "completed!"
}

main