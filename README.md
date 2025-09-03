# 🎵 vyb-code

> Feel the rhythm of perfect code

ローカルLLMベースのコーディングアシスタント。プライバシーを重視しながら、ClaudeCode相当の機能をローカル環境で実現します。

## 特徴

- **🎨 モダンTUI**: Bubble Teaフレームワークによる美しいターミナルUI体験
- **🔄 後方互換性**: `--no-tui`フラグで従来のテキストモードも利用可能
- **🎵 ブランドテーマ**: vybオリジナルカラーパレットによる視覚的アイデンティティ
- **⚡ リアルタイム表示**: プログレスバーとスピナーによる処理状況の可視化
- **プライバシー重視**: 全ての処理がローカルで実行 - 外部にデータを送信しません
- **対話型CLI**: 自然な会話形式でのコーディング支援
- **セキュアなコマンド実行**: ホワイトリスト制御と30秒タイムアウト
- **包括的Git統合**: ブランチ管理、コミット作成、状態確認
- **プロジェクト分析**: ファイル構造、言語分布、依存関係の自動解析
- **ファイル操作**: セキュアなファイル読み取り、書き込み、検索機能
- **多言語サポート**: Go、JavaScript/Node.js、Python対応基盤
- **設定管理**: 永続化された設定でモデルとプロバイダーを管理
- **Ollama統合**: HTTP API経由でのローカルLLM連携
- **セキュリティ重視**: ワークスペース制限と環境変数フィルタリング

## インストール

### 自動インストール（推奨）

```bash
# ワンライナーインストール
curl -sf https://raw.githubusercontent.com/glkt/vyb-code/main/scripts/install.sh | bash

# または手動ダウンロード
wget https://github.com/glkt/vyb-code/releases/latest/download/vyb-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m).tar.gz
tar xzf vyb-*.tar.gz
sudo mv vyb /usr/local/bin/
```

### ソースからビルド

```bash
# リポジトリをクローン
git clone https://github.com/glkt/vyb-code
cd vyb-code

# 開発ビルド
make build
# または
./scripts/build.sh dev

# 動作確認
./vyb --version
```

## クイックスタート

### 前提条件

1. Ollamaをインストールして起動
2. コーディング用モデルをダウンロード

```bash
# Ollamaでモデルをダウンロード（例）
ollama pull qwen2.5-coder:14b
```

### 基本的な使用方法

```bash
# 設定確認
vyb config list

# モデル設定
vyb config set-model qwen2.5-coder:14b

# 🎨 TUIモード（デフォルト）- モダンなターミナルUI体験
vyb chat
vyb

# 📟 レガシーモード - 従来のテキストベース
vyb chat --no-tui
vyb --no-tui

# 🎵 テーマ設定
vyb config set-tui-theme vyb       # vybブランドテーマ
vyb config set-tui-theme dark      # ダークテーマ  
vyb config set-tui-theme light     # ライトテーマ

# ⚙️ TUI設定
vyb config set-tui true            # TUIモード有効化
vyb config set-tui false           # TUIモード無効化

# 単発質問
vyb "GoでWebサーバーを作成して"

# コマンド実行（プログレスバー表示）
vyb exec "ls -la"
vyb exec "go version"

# Git操作（スピナー表示）
vyb git status
vyb git branch new-feature
vyb git commit "feat: add new functionality"

# プロジェクト分析（進捗表示）
vyb analyze

# MCP（Model Context Protocol）操作
vyb mcp add filesystem npx @modelcontextprotocol/server-filesystem
vyb mcp list
vyb mcp connect filesystem
vyb mcp tools filesystem
vyb mcp disconnect filesystem
```

## 推奨モデル

1. **Qwen2.5-Coder 14B/32B** - コーディングタスクで最高性能
2. **DeepSeek-Coder-V2 16B** - 性能とリソース使用量のバランス型
3. **CodeLlama 34B** - 安定性重視

## 動作要件

- Go 1.20以上
- ローカルLLMプロバイダ（Ollama推奨）
- 8GB+のRAM（小さなモデル用）、16GB+（大きなモデル用）
- **GPU加速（推奨）**: NVIDIA GPU 8GB+ VRAM、CUDA 11.8+

### 🚀 GPU加速化セットアップ

GPU加速により **78%のパフォーマンス向上** を実現できます（qwen2.5-coder:14b で 13.4秒 → 3.2秒）。

```bash
# クイックセットアップ
docker run -d --name ollama-vyb-gpu --gpus all -p 11434:11434 -v ollama-vyb:/root/.ollama ollama/ollama
./vyb config set-model qwen2.5-coder:14b
```

詳細な設定手順: [GPU Setup Guide](docs/gpu-setup.md)  
パフォーマンス詳細: [Performance Benchmarks](docs/performance-benchmarks.md)

## プロジェクトステータス

✅ **Phase 4完成** - Claude Code機能パリティ達成

### Phase 1: MVP機能（完成）

- ✅ CLI基盤（cobra）
- ✅ 設定管理（JSON永続化）
- ✅ Ollama HTTP API統合
- ✅ 対話型チャットセッション
- ✅ セキュアなファイル操作

### Phase 2: 機能拡張（完成）

- ✅ セキュアなコマンド実行（ホワイトリスト・タイムアウト制御）
- ✅ 包括的Git統合（ブランチ管理、コミット、状態確認）
- ✅ プロジェクト分析機能（言語検出、依存関係解析）
- ✅ 多言語サポート基盤（Go、JS、Python）

### Phase 3: 品質・配布（完成）

- ✅ テストインフラストラクチャ（全モジュール対応）
- ✅ パフォーマンス最適化（メトリクス、キャッシュ、並行処理）
- ✅ パッケージ配布対応（GitHub Actions、GoReleaser）
- ✅ ビルドシステム（マルチプラットフォーム対応）

### Phase 4: Claude Code機能パリティ（完成）

- ✅ MCPプロトコル実装（外部ツール連携基盤）
- ✅ 高度ファイル検索・グレップシステム（プロジェクト全体インデックス）
- ✅ 永続的会話セッション管理（履歴保存・復元・エクスポート）
- ✅ ストリーミング応答処理（リアルタイムLLM出力）
- ✅ 包括的セキュリティ強化（悪意あるLLM応答対策）
- ✅ 高度CLI統合（後方互換性維持）
- ✅ **モダンTUI統合**（Bubble Teaフレームワーク、テーマシステム）

## 新機能詳細（Phase 4）

### 🎨 モダンTUI (`internal/ui/`)

Bubble Teaフレームワークによる本格的なターミナルUI体験を提供：

- **テーマシステム**: vybブランド、ダーク、ライトテーマの自動切り替え
- **インタラクティブコンポーネント**: プログレスバー、スピナー、メニューシステム
- **リアルタイム表示**: 処理状況の視覚的フィードバック
- **完全後方互換**: `--no-tui`フラグで従来モード継続利用可能

### 🔧 MCP統合 (`internal/mcp/`)

外部ツールとの標準プロトコル通信を実現。Claude Codeと同様の拡張可能なツールエコシステムを提供。

### 🔍 インテリジェント検索 (`internal/search/`)

プロジェクト全体の高速インデックス化と正規表現対応検索。巨大なコードベースでも瞬時にファイル・コード片を発見。

### 💾 永続セッション (`internal/session/`)

会話履歴の永続化でプロジェクト状況を継続的に記憶。セッション検索・エクスポート・インポート機能付き。

### ⚡ ストリーミング応答 (`internal/stream/`)

LLM応答のリアルタイム表示でClaude Code同等のユーザー体験を実現。

## 貢献

このプロジェクトは標準的なGo言語の規約に従っています。詳細な開発ガイドラインはCLAUDE.mdを参照してください。

## ライセンス

BSD 3-Clause License - 詳細はLICENSEファイルを参照してください。
