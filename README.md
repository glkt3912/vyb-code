# 🎵 vyb-code

> Feel the rhythm of perfect code

ローカルLLMベースのコーディングアシスタント。プライバシーを重視しながら、Claude Code相当の機能＋**バイブコーディングモード**をローカル環境で実現します。**インタラクティブなバイブコーディングがデフォルト体験**として利用できます。

## 特徴

### 🧠 科学的認知分析システム（新実装）

**ハードコーディング問題の根本的解決**

- **セマンティックエントロピー信頼度測定**: Farquhar et al. (2024) 手法による動的信頼度計算
- **論理構造推論深度分析**: LogiGLUE framework と Toulmin 論証モデルを統合
- **Guilford創造性理論測定**: 4要素（流暢性・柔軟性・独創性・精密性）による科学的評価
- **動的パラメータ生成**: 固定値(0.8, 0.9, 4)から科学的測定への完全置換
- **2024年最新研究**: 7ファイル4,100+行の科学的実装

### 🎵 バイブコーディング体験（デフォルト）

- **インタラクティブモード**: Claude Code相当＋科学的認知分析
- **コンテキスト圧縮**: 70-95%効率でメモリ使用量を最適化
- **インテリジェント差分分析**: リスク評価、影響分析、推奨アクション
- **リアルタイム提案**: コード変更に対する即座の改善提案
- **カラー対応UI**: 緑色プロンプト、青色ロゴ、コードハイライト
- **Markdown対応**: コードブロック枠線、シンタックスハイライト
- **日本語IME完全対応**: 文字消失問題を解決済み

### 🛠️ Claude Code完全互換ツール

**基本ツール（10個）**

- **Bash**: セキュアなコマンド実行（タイムアウト・制約付き）
- **File Operations**: Read, Write, Edit, MultiEdit（構造化編集）
- **Search Tools**: Glob（パターン検索）, Grep（高度検索）, LS（リスト）
- **Web Integration**: WebFetch（内容取得）, WebSearch（検索）

**高度な開発支援ツール（4個）**

- **Project Analyze**: プロジェクト構造・依存関係・セキュリティ分析
- **Build**: 自動ビルドシステム検出・パイプライン実行・最適化提案
- **Architecture Map**: コードアーキテクチャ可視化・モジュール依存関係分析
- **Dependency Scan**: 依存関係脆弱性・ライセンス・更新可能性チェック

**全14個のツール**がClaude Code同等＋バイブモード高度機能で利用可能

### 🎵 バイブコーディングモード（Phase 7完成）

**インタラクティブセッション管理（3500+ lines）**

- **文脈理解型会話**: プロジェクト全体を理解した継続的コーディング支援
- **インテリジェント差分分析**: Git差分の詳細分析とリスク評価
- **リアルタイム提案**: コード変更に応じた即座の改善案提示

**コンテキスト圧縮システム（70-95%効率）**

- **スマートメモリ管理**: 長時間セッションでも高速応答維持
- **重要度ベース圧縮**: 重要なコンテキストを優先的に保持
- **動的最適化**: セッション進行に合わせたメモリ使用量調整

**メモリ効率会話管理**

- **会話履歴最適化**: 大規模プロジェクトでの応答速度向上
- **セッション持続性**: 長時間開発作業での安定性確保
- **インテリジェント要約**: 冗長な情報を自動的に圧縮

### 🔒 高度な入力システム

- **セキュリティ強化**: 入力サニタイゼーション、レート制限、バッファ保護
- **インテリジェント補完**: Git認識、コンテキスト補完、ファジーマッチング
- **パフォーマンス最適化**: 非同期処理、キャッシュ、デバウンス制御

### ⚡ スマート機能

- **自動プロジェクト認識**: 言語・依存関係・Git情報の自動コンテキスト追加
- **便利ショートカット**: `vyb s`（git status）、`vyb build`、`vyb test`等
- **リアルタイムメタ情報**: 応答時間、トークン数、モデル名表示

### 🛡️ セキュリティ・プライバシー

- **プライバシー重視**: 全ての処理がローカルで実行 - 外部送信なし
- **セキュアなコマンド実行**: ホワイトリスト制御と30秒タイムアウト
- **包括的Git統合**: ブランチ管理、コミット作成、状態確認

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

1. **[Ollama](https://ollama.ai/)**をインストールして起動
   - ローカルLLM実行プラットフォーム
   - [インストールガイド](https://ollama.ai/download) | [GitHub](https://github.com/ollama/ollama)

2. **コーディング用モデル**をダウンロード
   - [Qwen2.5-Coder](https://ollama.ai/library/qwen2.5-coder) - 推奨モデル
   - [CodeLlama](https://ollama.ai/library/codellama) - 安定性重視
   - [DeepSeek-Coder](https://ollama.ai/library/deepseek-coder) - バランス型

```bash
# Ollamaでモデルをダウンロード（例）
ollama pull qwen2.5-coder:14b

# その他の推奨モデル
ollama pull codellama:34b
ollama pull deepseek-coder:16b
```

### 基本的な使用方法

```bash
# 設定確認
vyb config list

# モデル設定
vyb config set-model qwen2.5-coder:14b

# 🎯 Claude Code風ターミナルモード（デフォルト）- Claude Code相当の体験
vyb                               # ターミナルモードで開始（推奨）
vyb chat                          # チャットコマンドでも同じ

# 🎨 TUIモード - モダンなターミナルUI体験
vyb --no-terminal-mode            # TUIモード（レガシー）
vyb chat --no-terminal-mode

# 📟 プレーンテキストモード - 従来のシンプル表示
vyb --no-tui                      # 完全テキストモード
vyb --no-terminal-mode --no-tui   # レガシー + テキスト

# 🎵 テーマ設定
vyb config set-tui-theme vyb       # vybブランドテーマ
vyb config set-tui-theme dark      # ダークテーマ
vyb config set-tui-theme light     # ライトテーマ

# ⚙️ TUI設定
vyb config set-tui true            # TUIモード有効化
vyb config set-tui false           # TUIモード無効化

# 単発質問
vyb "GoでWebサーバーを作成して"

# ⚡ 便利ショートカット
vyb s                              # git status の短縮形
vyb build                          # プロジェクト自動ビルド（Makefile/Go/Node.js対応）
vyb test                           # プロジェクト自動テスト

# コマンド実行（プログレスバー表示）
vyb exec "ls -la"
vyb exec "go version"

# Git操作（スピナー表示）
vyb git status
vyb git branch new-feature
vyb git commit "feat: add new functionality"

# プロジェクト分析（進捗表示）
vyb analyze

# 🚀 便利機能（開発中）
vyb quick explain main.go          # ファイル説明
vyb quick gen "HTTPクライアント"    # コード生成
vyb quick summarize                # 会話要約

# MCP（Model Context Protocol）操作
vyb mcp add filesystem npx @modelcontextprotocol/server-filesystem
vyb mcp list
vyb mcp connect filesystem
vyb mcp tools filesystem
vyb mcp disconnect filesystem
```

## 推奨モデル

| モデル | サイズ | 特徴 | リンク |
|--------|--------|------|--------|
| **Qwen2.5-Coder** | 14B/32B | コーディングタスクで最高性能 | [Ollama](https://ollama.ai/library/qwen2.5-coder) \| [HuggingFace](https://huggingface.co/Qwen/Qwen2.5-Coder-14B-Instruct) |
| **DeepSeek-Coder-V2** | 16B | 性能とリソース使用量のバランス型 | [Ollama](https://ollama.ai/library/deepseek-coder) \| [GitHub](https://github.com/deepseek-ai/DeepSeek-Coder) |
| **CodeLlama** | 34B | 安定性重視、Meta開発 | [Ollama](https://ollama.ai/library/codellama) \| [Meta AI](https://ai.meta.com/blog/code-llama-large-language-model-coding/) |

## 動作要件

- Go 1.20以上
- ローカルLLMプロバイダ（Ollama推奨）
- 8GB+のRAM（小さなモデル用）、16GB+（大きなモデル用）
- **GPU加速（推奨）**: NVIDIA GPU 8GB+ VRAM、CUDA 11.8+

### 🚀 GPU加速化セットアップ

GPU加速を使用することでLLM推論の大幅な高速化が期待できます。

```bash
# GPU対応のOllama実行（NVIDIA GPU + CUDA環境）
docker run -d --name ollama-gpu --gpus all -p 11434:11434 -v ollama-data:/root/.ollama ollama/ollama

# モデル設定
vyb config set-model qwen2.5-coder:14b
```

**GPU要件**: NVIDIA GPU 8GB+ VRAM、CUDA 11.8+
**詳細**: [Ollama GPU Guide](https://github.com/ollama/ollama?tab=readme-ov-file#nvidia-gpu)

## プロジェクトステータス

✅ **Phase 7完成** - 科学的認知分析システム実装完了

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
- ✅ **Claude Code風ターミナルモード**（カラー対応、Markdown表示、自動コンテキスト）

### Phase 5: 高度な入力システム（完成）

- ✅ **セキュリティ強化**（入力サニタイゼーション、バッファオーバーフロー保護、レート制限）
- ✅ **高度なオートコンプリート**（コンテキスト認識、Git統合、ファジーマッチング）
- ✅ **パフォーマンス最適化**（ワーカープール、LRUキャッシュ、非同期処理、デバウンス処理）
- ✅ **UTF-8完全対応**（日本語IME、マルチバイト文字処理、文字エンコーディング検証）
- ✅ **インテリジェント入力処理**（プロジェクト解析、コマンド予測、履歴最適化）

### Phase 7: 科学的認知分析システム（完成）

**ハードコーディング問題の根本的解決**

- ✅ **セマンティックエントロピー信頼度測定**（354行）- Farquhar et al. (2024) 手法による動的信頼度計算
- ✅ **論理構造推論深度分析**（649行）- LogiGLUE framework + Toulmin 論証モデル統合
- ✅ **Guilford創造性理論測定**（650行）- 4要素による科学的創造性評価エンジン
- ✅ **統合認知分析フレームワーク**（550行）- 全要素を統合する包括的認知システム
- ✅ **NLI分析エンジン**（495行）- 自然言語推論による含意関係分析
- ✅ **セマンティッククラスタリング**（756行）- 意味的類似性による応答グループ化
- ✅ **エントロピー計算エンジン**（649行）- von Neumann エントロピーによる不確実性定量化
- ✅ **動的パラメータ生成** - 固定値(0.8, 0.9, 4) → 科学的測定への完全置換

**バイブコーディングモード継続発展**

- ✅ **インタラクティブセッション管理**（3500+ lines）- 科学的分析統合バイブコーディング
- ✅ **コンテキスト圧縮システム**（70-95%効率）- メモリ使用量最適化
- ✅ **インテリジェント差分分析** - Git差分のリスク評価、影響分析
- ✅ **リアルタイム提案システム** - コード変更への即座の改善提案
- ✅ **デフォルト体験統合** - `vyb`コマンド実行時の標準モード化

## 新機能詳細（Phase 4-7）

### 🔒 高度な入力システム (`internal/input/`)

Phase 5で実装された包括的な入力処理システム：

- **セキュリティ強化**: 入力サニタイゼーション、バッファオーバーフロー保護、レート制限
- **パフォーマンス最適化**: ワーカープール、LRUキャッシュ、非同期処理、デバウンス制御
- **インテリジェント補完**: Git認識、プロジェクト解析、ファジーマッチング
- **UTF-8完全対応**: 日本語IME、マルチバイト文字処理、文字エンコーディング検証

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

### ⚡ ストリーミング応答 (`internal/streaming/`)

LLM応答のリアルタイム表示でClaude Code同等のユーザー体験を実現。

### 🎵 バイブコーディングモード (`internal/interactive/`, `internal/contextmanager/`)

Phase 7で実装されたインタラクティブコーディング体験：

- **インタラクティブセッション管理**: 3500+行の包括的バイブコーディングシステム（Git差分分析、リアルタイム提案、確認ダイアログ）
- **コンテキスト圧縮**: 70-95%効率のスマートメモリ管理（重要度ベース保持、動的最適化）
- **メモリ効率会話管理**: 大規模プロジェクトでの長時間セッション対応
- **Enhanced UIコンポーネント**: Bubble Tea統合による洗練されたユーザーインターフェース
- **デフォルト体験**: `vyb`コマンドでの標準バイブコーディング体験提供

## ドキュメント

詳細な技術情報については`docs/`ディレクトリを参照してください：

- **[docs/TECHNICAL_METHODS_EXPLAINED.md](docs/TECHNICAL_METHODS_EXPLAINED.md)** - 科学的認知分析システムの詳細解説
- **[docs/VERIFICATION_REPORT.md](docs/VERIFICATION_REPORT.md)** - ハードコーディング問題解決の検証結果
- **[docs/architecture.md](docs/architecture.md)** - システムアーキテクチャとMCP統合
- **[docs/performance-benchmarks.md](docs/performance-benchmarks.md)** - GPU加速化パフォーマンス結果
- **[docs/gpu-setup.md](docs/gpu-setup.md)** - GPU環境構築ガイド

## 貢献

このプロジェクトは標準的なGo言語の規約に従っています。詳細な開発ガイドラインは[CLAUDE.md](CLAUDE.md)を参照してください。

## ライセンス

BSD 3-Clause License - 詳細はLICENSEファイルを参照してください。
