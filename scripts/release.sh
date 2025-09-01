#!/bin/bash

# vyb-code リリーススクリプト
# GitHub Releasesへのアップロード

set -e

# 設定
PROJECT_NAME="vyb"
BUILD_DIR="dist"
CHANGELOG_FILE="CHANGELOG.md"

# 引数チェック
if [ $# -ne 1 ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v1.0.0"
    exit 1
fi

VERSION=$1

# バージョン形式のチェック
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must be in format vX.Y.Z (e.g., v1.0.0)"
    exit 1
fi

echo "=== vyb-code Release Script ==="
echo "Version: ${VERSION}"
echo

# 作業ディレクトリのクリーンアップ
echo "Cleaning workspace..."
git status --porcelain
if [ ! -z "$(git status --porcelain)" ]; then
    echo "Error: Working directory is not clean. Please commit or stash changes."
    exit 1
fi

# 最新のmainブランチを確認
echo "Checking main branch..."
git fetch origin
if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]; then
    echo "Error: Local main branch is not up to date with origin/main"
    exit 1
fi

# バージョンタグの確認
if git tag -l | grep -q "^${VERSION}$"; then
    echo "Error: Tag ${VERSION} already exists"
    exit 1
fi

# テスト実行
echo "Running tests..."
go test ./...
if [ $? -ne 0 ]; then
    echo "Error: Tests failed"
    exit 1
fi

# ビルド実行
echo "Building release binaries..."
VERSION=${VERSION} ./scripts/build.sh
if [ $? -ne 0 ]; then
    echo "Error: Build failed"
    exit 1
fi

# アーカイブの作成
echo "Creating archives..."
cd ${BUILD_DIR}

for binary in ${PROJECT_NAME}-${VERSION}-*; do
    if [ -f "$binary" ]; then
        platform=$(echo $binary | sed "s/${PROJECT_NAME}-${VERSION}-//")
        
        if [[ $platform == *"windows"* ]]; then
            # Windows用はZIPアーカイブ
            zip "${binary}.zip" "$binary"
            echo "Created: ${binary}.zip"
        else
            # Unix系はtar.gzアーカイブ
            tar czf "${binary}.tar.gz" "$binary"
            echo "Created: ${binary}.tar.gz"
        fi
    fi
done

cd ..

# タグの作成
echo "Creating git tag..."
git tag -a "${VERSION}" -m "Release ${VERSION}"

# リリースノートの生成
echo "Generating release notes..."
RELEASE_NOTES="release-notes-${VERSION}.md"

cat > ${RELEASE_NOTES} << EOF
# vyb-code ${VERSION}

## Changes
<!-- List of changes since last release -->

## Binary Downloads
- **Linux x64**: \`${PROJECT_NAME}-${VERSION}-linux-amd64.tar.gz\`
- **Linux ARM64**: \`${PROJECT_NAME}-${VERSION}-linux-arm64.tar.gz\`
- **macOS x64**: \`${PROJECT_NAME}-${VERSION}-darwin-amd64.tar.gz\`
- **macOS ARM64**: \`${PROJECT_NAME}-${VERSION}-darwin-arm64.tar.gz\`
- **Windows x64**: \`${PROJECT_NAME}-${VERSION}-windows-amd64.exe.zip\`

## Installation
\`\`\`bash
# Linux/macOS
curl -L https://github.com/glkt/vyb-code/releases/download/${VERSION}/${PROJECT_NAME}-${VERSION}-\$(uname -s | tr '[:upper:]' '[:lower:]')-\$(uname -m).tar.gz | tar xz
sudo mv ${PROJECT_NAME} /usr/local/bin/

# Or download manually from releases page
\`\`\`

## Verification
\`\`\`bash
${PROJECT_NAME} --version
\`\`\`
EOF

echo "Release preparation complete!"
echo
echo "Next steps:"
echo "1. Edit ${RELEASE_NOTES} to add changelog"
echo "2. Push tag: git push origin ${VERSION}"
echo "3. Create GitHub release with generated binaries"
echo
echo "Files ready for release:"
ls -la ${BUILD_DIR}/*.{tar.gz,zip} 2>/dev/null || true