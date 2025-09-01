#!/bin/bash

# vyb-code インストールスクリプト
# GitHub Releasesから最新版をダウンロードしてインストール

set -e

# 設定
GITHUB_REPO="glkt/vyb-code"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="vyb"

# プラットフォーム検出
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case $arch in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) echo "Unsupported architecture: $arch"; exit 1 ;;
    esac
    
    case $os in
        linux) platform="linux-${arch}" ;;
        darwin) platform="darwin-${arch}" ;;
        *) echo "Unsupported OS: $os"; exit 1 ;;
    esac
    
    echo $platform
}

# 最新リリースの取得
get_latest_release() {
    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    
    if command -v curl > /dev/null; then
        curl -s "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget > /dev/null; then
        wget -qO- "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        echo "Error: curl or wget is required"
        exit 1
    fi
}

# バージョン指定の処理
VERSION=${1:-$(get_latest_release)}

if [ -z "$VERSION" ]; then
    echo "Error: Could not determine version to install"
    exit 1
fi

PLATFORM=$(detect_platform)

echo "=== vyb-code Installer ==="
echo "Version: ${VERSION}"
echo "Platform: ${PLATFORM}"
echo "Install directory: ${INSTALL_DIR}"
echo

# ダウンロードURL
ARCHIVE_NAME="${BINARY_NAME}-${VERSION}-${PLATFORM}.tar.gz"
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"

echo "Downloading ${ARCHIVE_NAME}..."

# 一時ディレクトリ
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# ダウンロード
if command -v curl > /dev/null; then
    curl -L -o "${TMP_DIR}/${ARCHIVE_NAME}" "$DOWNLOAD_URL"
elif command -v wget > /dev/null; then
    wget -O "${TMP_DIR}/${ARCHIVE_NAME}" "$DOWNLOAD_URL"
else
    echo "Error: curl or wget is required"
    exit 1
fi

# 展開
echo "Extracting archive..."
cd "$TMP_DIR"
tar xzf "$ARCHIVE_NAME"

# バイナリファイルの確認
BINARY_FILE="${BINARY_NAME}-${VERSION}-${PLATFORM}"
if [ ! -f "$BINARY_FILE" ]; then
    echo "Error: Binary file not found in archive"
    exit 1
fi

# 実行権限の付与
chmod +x "$BINARY_FILE"

# インストール（sudo権限が必要な場合）
echo "Installing to ${INSTALL_DIR}..."

if [ -w "$INSTALL_DIR" ]; then
    # 書き込み権限がある場合
    cp "$BINARY_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
else
    # sudo権限が必要な場合
    echo "Administrator privileges required for installation to ${INSTALL_DIR}"
    sudo cp "$BINARY_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
fi

# インストール確認
echo "Verifying installation..."
if command -v "$BINARY_NAME" > /dev/null; then
    echo "✅ Installation successful!"
    echo
    "${BINARY_NAME}" --version
    echo
    echo "Try running: ${BINARY_NAME} --help"
else
    echo "❌ Installation failed or ${INSTALL_DIR} is not in PATH"
    exit 1
fi