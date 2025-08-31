# 🎵 vyb-code

> Feel the rhythm of perfect code

ローカルLLMベースのコーディングアシスタント。プライバシーを重視しながら、ClaudeCode相当の機能をローカル環境で実現します。

## 特徴

- **プライバシー重視**: 全ての処理がローカルで実行 - 外部にデータを送信しません
- **対話型CLI**: 自然な会話形式でのコーディング支援
- **ファイル操作**: セキュアなファイル読み取り、書き込み、検索機能
- **設定管理**: 永続化された設定でモデルとプロバイダーを管理
- **Ollama統合**: HTTP API経由でのローカルLLM連携（実装済み）
- **セキュリティ重視**: ワークスペース制限とファイルサイズ制限

## インストール

```bash
# リポジトリをクローン
git clone https://github.com/glkt3912/vyb-code
cd vyb-code

# 依存関係を取得
go mod tidy

# ビルド
go build -o vyb ./cmd/vyb

# 動作確認
./vyb config list
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

# 対話モード開始
vyb
> exit  # 終了

# 単発質問
vyb "GoでWebサーバーを作成して"
```

## 推奨モデル

1. **Qwen2.5-Coder 14B/32B** - コーディングタスクで最高性能
2. **DeepSeek-Coder-V2 16B** - 性能とリソース使用量のバランス型
3. **CodeLlama 34B** - 安定性重視

## 動作要件

- Go 1.20以上
- ローカルLLMプロバイダ（Ollama推奨）
- 8GB+のRAM（小さなモデル用）、16GB+（大きなモデル用）

## プロジェクトステータス

✅ **MVP完成** - 基本機能実装済み

### 実装済み機能
- ✅ CLI基盤（cobra）
- ✅ 設定管理（JSON永続化）
- ✅ Ollama HTTP API統合
- ✅ 対話型チャットセッション
- ✅ セキュアなファイル操作
- ✅ 基本コマンド体系

### 次期開発予定
- 🔄 コマンド実行機能
- 🔄 Git統合
- 🔄 プロジェクト解析
- 🔄 テストインフラ

## 貢献

このプロジェクトは標準的なGo言語の規約に従っています。詳細な開発ガイドラインはCLAUDE.mdを参照してください。

## ライセンス

BSD 3-Clause License - 詳細はLICENSEファイルを参照してください。