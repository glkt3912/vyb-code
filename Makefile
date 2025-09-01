# vyb-code Makefile
# vybプロジェクトのビルドとテスト自動化

.PHONY: build test clean install deps lint fmt check coverage help

# デフォルトターゲット
all: deps fmt lint test build

# 依存関係のインストール
deps:
	go mod download
	go mod tidy

# コードフォーマット
fmt:
	go fmt ./...

# リンター実行（golangci-lintが利用可能な場合）
lint:
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping lint check"; \
		go vet ./...; \
	fi

# テスト実行
test:
	go test -v ./...

# カバレッジ付きテスト
coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# ビルド
build:
	go build -o vyb ./cmd/vyb

# インストール（$GOPATH/binまたは$GOBIN）
install:
	go install ./cmd/vyb

# クリーンアップ
clean:
	rm -f vyb
	rm -f coverage.out coverage.html
	go clean

# 完全チェック（CI用）
check: deps fmt lint test build

# ヘルプ表示
help:
	@echo "vyb-code Makefile targets:"
	@echo "  build     - Build the vyb binary"
	@echo "  test      - Run all tests"
	@echo "  coverage  - Run tests with coverage report"
	@echo "  lint      - Run linter (golangci-lint if available)"
	@echo "  fmt       - Format code"
	@echo "  deps      - Download and tidy dependencies"
	@echo "  install   - Install vyb to GOPATH/bin"
	@echo "  clean     - Clean build artifacts"
	@echo "  check     - Run full CI checks"
	@echo "  help      - Show this help"