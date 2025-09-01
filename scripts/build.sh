#!/bin/bash

# vyb-code ビルドスクリプト
# 複数プラットフォーム向けのバイナリ生成

set -e

# ビルド設定
PROJECT_NAME="vyb"
VERSION=${VERSION:-"dev"}
BUILD_DIR="dist"
CMD_PATH="./cmd/vyb"

# ビルド情報
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# LDFLAGSの設定（バイナリにビルド情報を埋め込み）
LDFLAGS="-X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}' -X 'main.GitCommit=${GIT_COMMIT}' -X 'main.GitBranch=${GIT_BRANCH}'"

echo "=== vyb-code Build Script ==="
echo "Version: ${VERSION}"
echo "Build Time: ${BUILD_TIME}"
echo "Git Commit: ${GIT_COMMIT}"
echo "Git Branch: ${GIT_BRANCH}"
echo

# ビルドディレクトリの作成
mkdir -p ${BUILD_DIR}

# 単一プラットフォーム向けビルド（開発用）
if [ "${1}" = "dev" ]; then
    echo "Building for development (current platform)..."
    go build -ldflags="${LDFLAGS}" -o "${BUILD_DIR}/${PROJECT_NAME}" ${CMD_PATH}
    echo "Build complete: ${BUILD_DIR}/${PROJECT_NAME}"
    exit 0
fi

# 複数プラットフォーム向けビルド
platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

echo "Building for multiple platforms..."

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    
    output_name="${PROJECT_NAME}-${VERSION}-${GOOS}-${GOARCH}"
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi
    
    echo "Building ${output_name}..."
    
    env GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="${LDFLAGS}" \
        -o "${BUILD_DIR}/${output_name}" \
        ${CMD_PATH}
    
    if [ $? -ne 0 ]; then
        echo "Build failed for ${platform}"
        exit 1
    fi
done

echo
echo "=== Build Summary ==="
echo "Built binaries:"
ls -la ${BUILD_DIR}/
echo
echo "All builds completed successfully!"