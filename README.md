# 🎵 vyb-code

> Feel the rhythm of perfect code

ローカルLLMベースのコーディングアシスタント。プライバシーを重視しながら、ClaudeCode相当の機能をローカル環境で実現します。

## 特徴

- **プライバシー重視**: 全ての処理がローカルで実行 - 外部にデータを送信しません
- **対話型CLI**: 自然な会話形式でのコーディング支援
- **ファイル操作**: コードベースの読み取り、書き込み、検索
- **Git統合**: インテリジェントなブランチ操作とコミット生成
- **マルチLLM対応**: Ollama、LM Studio、vLLMに対応
- **セキュリティ重視**: ホワイトリスト方式による制限付きコマンド実行

## インストール

```bash
# ソースからビルド
git clone https://github.com/glkt/vyb-code
cd vyb-code
go build -o vyb ./cmd/vyb

# グローバルインストール
go install ./cmd/vyb
```

## クイックスタート

```bash
# 対話モード開始
vyb

# 質問を投げる
vyb "GoでWebサーバーを作成して"

# LLMの設定
vyb config set-provider ollama
vyb config set-model qwen2.5-coder:14b
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

🚧 **開発初期段階** - MVPフェーズ進行中

このプロジェクトは活発に開発されています。CLAUDE.mdのロードマップに従ってコア機能を実装中です。

## 貢献

このプロジェクトは標準的なGo言語の規約に従っています。詳細な開発ガイドラインはCLAUDE.mdを参照してください。

## ライセンス

BSD 3-Clause License - 詳細はLICENSEファイルを参照してください。