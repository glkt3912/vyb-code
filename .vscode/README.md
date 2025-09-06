# VSCode設定について

このディレクトリには、vyb-codeプロジェクト開発用のVSCode設定が含まれています。

## 📁 ファイル構成

### `settings.json`

- **ファイル保存時の自動設定**
  - `files.insertFinalNewline: true` - ファイル末尾に改行を自動追加
  - `files.trimFinalNewlines: true` - 余分な改行を削除
  - `files.trimTrailingWhitespace: true` - 行末の空白を削除

- **Go言語設定**
  - `go.formatTool: "gofmt"` - 標準のgofmtを使用
  - `editor.formatOnSave: true` - 保存時に自動フォーマット
  - `source.organizeImports` - インポートの自動整理

### `extensions.json`

推奨プラグイン：

- `golang.go` - Go言語サポート（必須）
- `eamodio.gitlens` - Git履歴とブレーム表示
- `yzhang.markdown-all-in-one` - Markdownサポート
- `ms-vscode.makefile-tools` - Makefile実行

### `launch.json`

デバッグ設定：

- **Debug vyb-code** - メインプログラムのデバッグ
- **Debug vyb-code with args** - 引数付き実行
- **Debug vyb-code terminal mode** - ターミナルモードのデバッグ
- **Debug Tests** - テスト用デバッグ

### `tasks.json`

よく使うタスク：

- **Go: Build** - プロジェクトビルド（Ctrl+Shift+P）
- **Go: Test** - 全テスト実行
- **Go: Test with Coverage** - カバレッジ付きテスト
- **Make: Full CI Check** - CI相当の全チェック

### `snippets.json`

コードスニペット：

- `gotest` - テスト関数テンプレート
- `iferr` - エラーチェックテンプレート
- `claudetool` - Claude Codeツールテンプレート

## 🚀 使用方法

1. **推奨プラグインのインストール**

   ```
   Ctrl+Shift+P → "Extensions: Show Recommended Extensions"
   ```

2. **自動フォーマット確認**
   - Goファイルを編集して保存
   - 自動的にgofmtが実行される
   - ファイル末尾に改行が追加される

3. **タスク実行**

   ```
   Ctrl+Shift+P → "Tasks: Run Task"
   - Go: Build - ビルド実行
   - Go: Test - テスト実行
   - Make: Full CI Check - CI相当チェック
   ```

4. **デバッグ実行**

   ```
   F5 - "Debug vyb-code"設定でデバッグ開始
   ```

## ⚙️ カスタマイズ

### 個人設定の追加

プロジェクト固有でない設定は `~/.vscode/settings.json` に追加してください。

### チーム共有

このディレクトリの設定はGitで管理され、チーム全体で共有されます。

## 🔧 トラブルシューティング

### Goプラグインが動作しない

```bash
# Go拡張機能の再インストール
Ctrl+Shift+P → "Go: Install/Update Tools"
```

### フォーマットが適用されない

```bash
# gofmtが正しくインストールされているか確認
which gofmt
go version
```

### デバッグが動作しない

```bash
# Delveデバッガーのインストール
go install github.com/go-delve/delve/cmd/dlv@latest
```

このVSCode設定により、開発効率とコード品質の両方が向上し、CI/CD パイプラインとの整合性も保たれます。
