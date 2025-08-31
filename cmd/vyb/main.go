package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/glkt/vyb-code/internal/chat"
	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/llm"
	"github.com/glkt/vyb-code/internal/security"
	"github.com/glkt/vyb-code/internal/tools"
)

// メインコマンド：vyb単体で実行される処理
var rootCmd = &cobra.Command{
	Use:   "vyb",
	Short: "Local AI coding assistant",
	Long:  `vyb - Feel the rhythm of perfect code. A local LLM-based coding assistant that prioritizes privacy and developer experience.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// 引数なし：対話モード開始
			startInteractiveMode()
		} else {
			// 引数あり：単発コマンド処理
			query := args[0]
			processSingleQuery(query)
		}
	},
}

// チャットコマンド：明示的に対話モードを開始
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive chat mode",
	Run: func(cmd *cobra.Command, args []string) {
		startInteractiveMode()
	},
}

// 設定管理のメインコマンド
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage vyb configuration",
}

// モデル設定コマンド：vyb config set-model qwen2.5-coder:14b
var setModelCmd = &cobra.Command{
	Use:   "set-model [model]",
	Short: "Set the LLM model to use",
	Args:  cobra.ExactArgs(1), // 引数を1つだけ受け取る
	Run: func(cmd *cobra.Command, args []string) {
		model := args[0] // 最初の引数をモデル名として取得
		setModel(model)
	},
}

// プロバイダー設定コマンド：vyb config set-provider ollama
var setProviderCmd = &cobra.Command{
	Use:   "set-provider [provider]",
	Short: "Set the LLM provider (ollama, lmstudio, vllm)",
	Args:  cobra.ExactArgs(1), // 引数を1つだけ受け取る
	Run: func(cmd *cobra.Command, args []string) {
		provider := args[0] // 最初の引数をプロバイダー名として取得
		setProvider(provider)
	},
}

var listConfigCmd = &cobra.Command{
	Use:   "list",
	Short: "List current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		listConfig()
	},
}

// コマンド実行機能
var execCmd = &cobra.Command{
	Use:   "exec [command]",
	Short: "Execute shell command with security constraints",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		executeCommand(args)
	},
}

// Git操作のメインコマンド
var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git operations with enhanced functionality",
}

var gitStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show git status",
	Run: func(cmd *cobra.Command, args []string) {
		gitStatus()
	},
}

var gitBranchCmd = &cobra.Command{
	Use:   "branch [branch-name]",
	Short: "Create and checkout new branch",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			createBranch(args[0])
		} else {
			listBranches()
		}
	},
}

var gitCommitCmd = &cobra.Command{
	Use:   "commit [message]",
	Short: "Create commit with message",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commitChanges(args[0])
	},
}

// プロジェクト分析コマンド
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze project structure and dependencies",
	Run: func(cmd *cobra.Command, args []string) {
		analyzeProject()
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(gitCmd)
	rootCmd.AddCommand(analyzeCmd)

	configCmd.AddCommand(setModelCmd)
	configCmd.AddCommand(setProviderCmd)
	configCmd.AddCommand(listConfigCmd)

	gitCmd.AddCommand(gitStatusCmd)
	gitCmd.AddCommand(gitBranchCmd)
	gitCmd.AddCommand(gitCommitCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// 対話モードを開始する実装関数
func startInteractiveMode() {
	fmt.Println("🎵 vyb - Feel the rhythm of perfect code")
	
	// 設定を読み込み
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}
	
	// LLMクライアントを作成
	provider := llm.NewOllamaClient(cfg.BaseURL)
	
	// チャットセッションを開始
	session := chat.NewSession(provider, cfg.Model)
	if err := session.StartInteractive(); err != nil {
		fmt.Printf("対話セッションエラー: %v\n", err)
	}
}

// 単発クエリを処理する実装関数
func processSingleQuery(query string) {
	fmt.Printf("Processing: %s\n", query)
	
	// 設定を読み込み
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}
	
	// LLMクライアントを作成
	provider := llm.NewOllamaClient(cfg.BaseURL)
	
	// チャットセッションで単発処理
	session := chat.NewSession(provider, cfg.Model)
	if err := session.ProcessQuery(query); err != nil {
		fmt.Printf("クエリ処理エラー: %v\n", err)
	}
}

// モデルを設定する実装関数
func setModel(model string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}
	
	if err := cfg.SetModel(model); err != nil {
		fmt.Printf("モデル設定エラー: %v\n", err)
		return
	}
	
	fmt.Printf("モデルを %s に設定しました\n", model)
}

// プロバイダーを設定する実装関数
func setProvider(provider string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}
	
	if err := cfg.SetProvider(provider); err != nil {
		fmt.Printf("プロバイダー設定エラー: %v\n", err)
		return
	}
	
	fmt.Printf("プロバイダーを %s に設定しました\n", provider)
}

// 現在の設定を表示する実装関数
func listConfig() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}
	
	fmt.Println("現在の設定:")
	fmt.Printf("  プロバイダー: %s\n", cfg.Provider)
	fmt.Printf("  モデル: %s\n", cfg.Model)
	fmt.Printf("  ベースURL: %s\n", cfg.BaseURL)
	fmt.Printf("  タイムアウト: %d秒\n", cfg.Timeout)
	fmt.Printf("  最大ファイルサイズ: %d MB\n", cfg.MaxFileSize/(1024*1024))
}

// コマンド実行の実装関数
func executeCommand(args []string) {
	// 現在のディレクトリを取得
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("作業ディレクトリ取得エラー: %v\n", err)
		return
	}

	// セキュリティ制約を設定
	constraints := security.NewDefaultConstraints(workDir)
	executor := tools.NewCommandExecutor(constraints, workDir)

	// コマンドを結合
	command := strings.Join(args, " ")

	fmt.Printf("実行中: %s\n", command)

	// コマンド実行
	result, err := executor.Execute(command)
	if err != nil {
		fmt.Printf("実行エラー: %v\n", err)
		return
	}

	// 結果を表示
	fmt.Printf("終了コード: %d\n", result.ExitCode)
	fmt.Printf("実行時間: %s\n", result.Duration)

	if result.Stdout != "" {
		fmt.Printf("標準出力:\n%s", result.Stdout)
	}

	if result.Stderr != "" {
		fmt.Printf("標準エラー:\n%s", result.Stderr)
	}

	if result.TimedOut {
		fmt.Println("⚠️  コマンドがタイムアウトしました")
	}
}

// Git状態確認の実装関数
func gitStatus() {
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("作業ディレクトリ取得エラー: %v\n", err)
		return
	}

	constraints := security.NewDefaultConstraints(workDir)
	gitOps := tools.NewGitOperations(constraints, workDir)

	result, err := gitOps.GetStatus()
	if err != nil {
		fmt.Printf("Git状態取得エラー: %v\n", err)
		return
	}

	if result.ExitCode != 0 {
		fmt.Printf("Gitエラー: %s\n", result.Stderr)
		return
	}

	fmt.Println("Git状態:")
	if result.Stdout == "" {
		fmt.Println("  変更なし（クリーン）")
	} else {
		fmt.Print(result.Stdout)
	}
}

// ブランチ作成の実装関数
func createBranch(branchName string) {
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("作業ディレクトリ取得エラー: %v\n", err)
		return
	}

	constraints := security.NewDefaultConstraints(workDir)
	gitOps := tools.NewGitOperations(constraints, workDir)

	result, err := gitOps.CreateAndCheckoutBranch(branchName)
	if err != nil {
		fmt.Printf("ブランチ作成エラー: %v\n", err)
		return
	}

	if result.ExitCode != 0 {
		fmt.Printf("Gitエラー: %s\n", result.Stderr)
		return
	}

	fmt.Printf("ブランチ '%s' を作成してチェックアウトしました\n", branchName)
}

// ブランチ一覧の実装関数
func listBranches() {
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("作業ディレクトリ取得エラー: %v\n", err)
		return
	}

	constraints := security.NewDefaultConstraints(workDir)
	gitOps := tools.NewGitOperations(constraints, workDir)

	result, err := gitOps.GetBranches()
	if err != nil {
		fmt.Printf("ブランチ取得エラー: %v\n", err)
		return
	}

	if result.ExitCode != 0 {
		fmt.Printf("Gitエラー: %s\n", result.Stderr)
		return
	}

	fmt.Println("ブランチ一覧:")
	fmt.Print(result.Stdout)
}

// コミット作成の実装関数
func commitChanges(message string) {
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("作業ディレクトリ取得エラー: %v\n", err)
		return
	}

	constraints := security.NewDefaultConstraints(workDir)
	gitOps := tools.NewGitOperations(constraints, workDir)

	// ファイルをステージング
	_, err = gitOps.AddFiles(nil) // nilで全ファイルを追加
	if err != nil {
		fmt.Printf("ファイルステージングエラー: %v\n", err)
		return
	}

	// コミット作成
	result, err := gitOps.Commit(message)
	if err != nil {
		fmt.Printf("コミット作成エラー: %v\n", err)
		return
	}

	if result.ExitCode != 0 {
		fmt.Printf("Gitエラー: %s\n", result.Stderr)
		return
	}

	fmt.Printf("コミットを作成しました: %s\n", message)
}

// プロジェクト分析の実装関数
func analyzeProject() {
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("作業ディレクトリ取得エラー: %v\n", err)
		return
	}

	constraints := security.NewDefaultConstraints(workDir)
	analyzer := tools.NewProjectAnalyzer(constraints, workDir)

	fmt.Println("プロジェクト分析中...")
	analysis, err := analyzer.AnalyzeProject()
	if err != nil {
		fmt.Printf("プロジェクト分析エラー: %v\n", err)
		return
	}

	// 分析結果を表示
	fmt.Println("\n📊 プロジェクト分析結果:")
	fmt.Printf("  総ファイル数: %d\n", analysis.TotalFiles)

	fmt.Println("\n📝 プログラミング言語:")
	for lang, count := range analysis.FilesByLanguage {
		fmt.Printf("  %s: %d ファイル\n", lang, count)
	}

	fmt.Println("\n📁 プロジェクト構造:")
	for dir, contents := range analysis.ProjectStructure {
		fmt.Printf("  %s/\n", dir)
		for _, item := range contents {
			fmt.Printf("    └── %s\n", item)
		}
	}

	if len(analysis.Dependencies) > 0 {
		fmt.Println("\n📦 依存関係:")
		for _, dep := range analysis.Dependencies {
			fmt.Printf("  - %s\n", dep)
		}
	}

	if analysis.GitInfo != nil {
		fmt.Println("\n🔀 Git情報:")
		fmt.Printf("  現在のブランチ: %s\n", analysis.GitInfo.CurrentBranch)
		fmt.Printf("  状態: %s\n", analysis.GitInfo.Status)
		
		if len(analysis.GitInfo.Branches) > 0 {
			fmt.Printf("  ブランチ数: %d\n", len(analysis.GitInfo.Branches))
		}
		
		if len(analysis.GitInfo.RecentCommits) > 0 {
			fmt.Println("  最近のコミット:")
			for i, commit := range analysis.GitInfo.RecentCommits {
				if i < 3 { // 最新3件のみ表示
					fmt.Printf("    - %s\n", commit)
				}
			}
		}
	}
}
