package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/glkt/vyb-code/internal/chat"
	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/diagnostic"
	"github.com/glkt/vyb-code/internal/llm"
	"github.com/glkt/vyb-code/internal/mcp"
	"github.com/glkt/vyb-code/internal/search"
	"github.com/glkt/vyb-code/internal/security"
	"github.com/glkt/vyb-code/internal/tools"
	"github.com/glkt/vyb-code/internal/ui"
	"github.com/spf13/cobra"
)

// メインコマンド：vyb単体で実行される処理
var rootCmd = &cobra.Command{
	Use:     "vyb",
	Short:   "Local AI coding assistant",
	Long:    `vyb - Feel the rhythm of perfect code. A local LLM-based coding assistant that prioritizes privacy and developer experience.`,
	Version: GetVersionString(),
	Run: func(cmd *cobra.Command, args []string) {
		// --no-tuiフラグをチェック
		noTUI, _ := cmd.Flags().GetBool("no-tui")

		if len(args) == 0 {
			// 引数なし：対話モード開始
			startInteractiveMode(noTUI)
		} else {
			// 引数あり：単発コマンド処理
			query := args[0]
			processSingleQuery(query, noTUI)
		}
	},
}

// チャットコマンド：明示的に対話モードを開始
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive chat mode",
	Run: func(cmd *cobra.Command, args []string) {
		noTUI, _ := cmd.Flags().GetBool("no-tui")
		startInteractiveMode(noTUI)
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

// ログレベル設定コマンド
var setLogLevelCmd = &cobra.Command{
	Use:   "set-log-level [level]",
	Short: "Set log level (debug, info, warn, error)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		setLogLevel(args[0])
	},
}

// ログフォーマット設定コマンド
var setLogFormatCmd = &cobra.Command{
	Use:   "set-log-format [format]",
	Short: "Set log format (console, json)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		setLogFormat(args[0])
	},
}

// TUI有効/無効設定コマンド
var setTUIEnabledCmd = &cobra.Command{
	Use:   "set-tui [enabled]",
	Short: "Enable or disable TUI mode (true, false)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		enabled := args[0] == "true"
		setTUIEnabled(enabled)
	},
}

// TUIテーマ設定コマンド
var setTUIThemeCmd = &cobra.Command{
	Use:   "set-tui-theme [theme]",
	Short: "Set TUI theme (dark, light, vyb, auto)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		setTUITheme(args[0])
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

// 検索機能のメインコマンド
var searchCmd = &cobra.Command{
	Use:   "search [pattern]",
	Short: "Search across project files with intelligent ranking",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		smart, _ := cmd.Flags().GetBool("smart")
		maxResults, _ := cmd.Flags().GetInt("max-results")
		includeContext, _ := cmd.Flags().GetBool("context")
		performSearch(args[0], smart, maxResults, includeContext)
	},
}

// MCP操作のメインコマンド
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP (Model Context Protocol) server management",
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured MCP servers",
	Run: func(cmd *cobra.Command, args []string) {
		listMCPServers()
	},
}

var mcpConnectCmd = &cobra.Command{
	Use:   "connect [server-name]",
	Short: "Connect to MCP server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		connectMCPServer(args[0])
	},
}

var mcpToolsCmd = &cobra.Command{
	Use:   "tools [server-name]",
	Short: "List available tools from MCP server",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			listMCPTools(args[0])
		} else {
			listAllMCPTools()
		}
	},
}

var mcpDisconnectCmd = &cobra.Command{
	Use:   "disconnect [server-name]",
	Short: "Disconnect from MCP server",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			disconnectMCPServer(args[0])
		} else {
			disconnectAllMCPServers()
		}
	},
}

var mcpAddCmd = &cobra.Command{
	Use:   "add [server-name] [command...]",
	Short: "Add new MCP server configuration",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		addMCPServer(args[0], args[1:])
	},
}

// ヘルスチェック・診断機能のメインコマンド
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "System health check and diagnostics",
}

var healthCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Run health checks for all components",
	Run: func(cmd *cobra.Command, args []string) {
		runHealthCheck()
	},
}

var diagnosticsCmd = &cobra.Command{
	Use:   "diagnostics",
	Short: "Run comprehensive system diagnostics",
	Run: func(cmd *cobra.Command, args []string) {
		runDiagnostics()
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(gitCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(healthCmd)

	// メインコマンドのフラグ
	rootCmd.Flags().Bool("no-tui", false, "Disable TUI mode (use plain text output)")

	// チャットコマンドのフラグ
	chatCmd.Flags().Bool("no-tui", false, "Disable TUI mode (use plain text output)")

	// 検索コマンドのフラグ
	searchCmd.Flags().Bool("smart", false, "Use intelligent search with AST analysis")
	searchCmd.Flags().Int("max-results", 50, "Maximum number of results to return")
	searchCmd.Flags().Bool("context", true, "Include context lines in results")

	configCmd.AddCommand(setModelCmd)
	configCmd.AddCommand(setProviderCmd)
	configCmd.AddCommand(listConfigCmd)
	configCmd.AddCommand(setLogLevelCmd)
	configCmd.AddCommand(setLogFormatCmd)
	configCmd.AddCommand(setTUIEnabledCmd)
	configCmd.AddCommand(setTUIThemeCmd)

	gitCmd.AddCommand(gitStatusCmd)
	gitCmd.AddCommand(gitBranchCmd)
	gitCmd.AddCommand(gitCommitCmd)

	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpConnectCmd)
	mcpCmd.AddCommand(mcpToolsCmd)
	mcpCmd.AddCommand(mcpDisconnectCmd)
	mcpCmd.AddCommand(mcpAddCmd)

	healthCmd.AddCommand(healthCheckCmd)
	healthCmd.AddCommand(diagnosticsCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// 対話モードを開始する実装関数
func startInteractiveMode(noTUI bool) {
	// 設定を読み込み
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}

	// TUIモードの判定
	useTUI := cfg.TUI.Enabled && !noTUI

	if useTUI {
		// TUIモードで開始
		// LLMクライアントとセッションを作成
		provider := llm.NewOllamaClient(cfg.BaseURL)
		session := chat.NewSession(provider, cfg.Model)
		
		// チャット機能付きTUIアプリを作成
		app := ui.NewChatApp(cfg.TUI, session)
		program := tea.NewProgram(app, 
			// tea.WithAltScreen() を削除 - 通常のターミナル選択を有効化
			tea.WithMouseCellMotion(), // マウスサポート
			tea.WithMouseAllMotion(),   // すべてのマウス動作
		)

		if _, err := program.Run(); err != nil {
			// TTYがない環境（CI、SSH等）では自動的にレガシーモードに切り替え
			if strings.Contains(err.Error(), "/dev/tty") || strings.Contains(err.Error(), "inappropriate ioctl") {
				startLegacyInteractiveMode(cfg)
			} else {
				fmt.Printf("TUIエラー: %v\n", err)
				startLegacyInteractiveMode(cfg)
			}
		}
	} else {
		// 従来の対話モード
		startLegacyInteractiveMode(cfg)
	}
}

// 従来の対話モード（TUI無効時）
func startLegacyInteractiveMode(cfg *config.Config) {
	fmt.Println("🎵 vyb - Feel the rhythm of perfect code")

	// LLMクライアントを作成
	provider := llm.NewOllamaClient(cfg.BaseURL)

	// チャットセッションを開始
	session := chat.NewSession(provider, cfg.Model)
	if err := session.StartInteractive(); err != nil {
		fmt.Printf("対話セッションエラー: %v\n", err)
	}
}

// 単発クエリを処理する実装関数
func processSingleQuery(query string, noTUI bool) {
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
	fmt.Printf("  ログレベル: %s\n", cfg.Logging.Level)
	fmt.Printf("  ログフォーマット: %s\n", cfg.Logging.Format)
	fmt.Printf("  ログ出力先: %v\n", cfg.Logging.Output)

	fmt.Println("\nTUI設定:")
	tuiStatus := "無効"
	if cfg.TUI.Enabled {
		tuiStatus = "有効"
	}
	fmt.Printf("  TUIモード: %s\n", tuiStatus)
	fmt.Printf("  テーマ: %s\n", cfg.TUI.Theme)
	fmt.Printf("  スピナー表示: %t\n", cfg.TUI.ShowSpinner)
	fmt.Printf("  プログレスバー表示: %t\n", cfg.TUI.ShowProgress)
	fmt.Printf("  アニメーション: %t\n", cfg.TUI.Animation)
}

// ログレベルを設定する実装関数
func setLogLevel(level string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}

	if err := cfg.SetLogLevel(level); err != nil {
		fmt.Printf("ログレベル設定エラー: %v\n", err)
		return
	}

	fmt.Printf("ログレベルを %s に設定しました\n", level)
}

// ログフォーマットを設定する実装関数
func setLogFormat(format string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}

	if err := cfg.SetLogFormat(format); err != nil {
		fmt.Printf("ログフォーマット設定エラー: %v\n", err)
		return
	}

	fmt.Printf("ログフォーマットを %s に設定しました\n", format)
}

// TUI有効/無効を設定する実装関数
func setTUIEnabled(enabled bool) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}

	if err := cfg.SetTUIEnabled(enabled); err != nil {
		fmt.Printf("TUI設定エラー: %v\n", err)
		return
	}

	status := "無効"
	if enabled {
		status = "有効"
	}
	fmt.Printf("TUIモードを %s に設定しました\n", status)
}

// TUIテーマを設定する実装関数
func setTUITheme(theme string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}

	if err := cfg.SetTUITheme(theme); err != nil {
		fmt.Printf("TUIテーマ設定エラー: %v\n", err)
		return
	}

	fmt.Printf("TUIテーマを %s に設定しました\n", theme)
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

	// セキュア実行器を作成（厳格モード無効）
	secureExecutor := security.NewSecureExecutor(constraints, false)

	// 従来のコマンド実行器も作成
	executor := tools.NewCommandExecutor(constraints, workDir)

	// コマンドを結合
	command := strings.Join(args, " ")

	// セキュリティ検証と確認
	allowed, err := secureExecutor.ValidateAndConfirm(command)
	if err != nil || !allowed {
		fmt.Printf("セキュリティチェックによりコマンド実行が拒否されました: %v\n", err)
		return
	}

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

// MCPサーバー一覧表示の実装関数
func listMCPServers() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}

	servers := cfg.GetMCPServers()
	if len(servers) == 0 {
		fmt.Println("設定されたMCPサーバーはありません")
		return
	}

	fmt.Println("設定済みMCPサーバー:")
	for name, server := range servers {
		status := "無効"
		if server.Enabled {
			status = "有効"
		}
		fmt.Printf("  %s (%s)\n", name, status)
		fmt.Printf("    コマンド: %s\n", strings.Join(server.Command, " "))
		if server.AutoConnect {
			fmt.Printf("    自動接続: はい\n")
		}
	}
}

// MCPサーバー接続の実装関数
func connectMCPServer(serverName string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}

	serverConfig, err := cfg.GetMCPServer(serverName)
	if err != nil {
		fmt.Printf("MCPサーバー取得エラー: %v\n", err)
		return
	}

	if !serverConfig.Enabled {
		fmt.Printf("MCPサーバー '%s' は無効化されています\n", serverName)
		return
	}

	// MCPクライアントを作成
	clientConfig := mcp.ClientConfig{
		ServerCommand: serverConfig.Command,
		ServerArgs:    serverConfig.Args,
		Environment:   serverConfig.Environment,
		WorkingDir:    serverConfig.WorkingDir,
	}

	client := mcp.NewClient(clientConfig)

	// サーバーに接続
	fmt.Printf("MCPサーバー '%s' に接続中...\n", serverName)
	if err := client.Connect(clientConfig); err != nil {
		fmt.Printf("接続エラー: %v\n", err)
		return
	}

	fmt.Printf("✅ MCPサーバー '%s' に接続しました\n", serverName)

	// 利用可能なツールを表示
	tools := client.GetTools()
	if len(tools) > 0 {
		fmt.Printf("利用可能なツール (%d個):\n", len(tools))
		for _, tool := range tools {
			fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
		}
	}

	// リソースを表示
	resources := client.GetResources()
	if len(resources) > 0 {
		fmt.Printf("利用可能なリソース (%d個):\n", len(resources))
		for _, resource := range resources {
			fmt.Printf("  - %s: %s\n", resource.Name, resource.Description)
		}
	}
}

// MCPツール一覧表示の実装関数（特定サーバー）
func listMCPTools(serverName string) {
	fmt.Printf("MCPサーバー '%s' のツール一覧機能は開発中です\n", serverName)
}

// MCPツール一覧表示の実装関数（全サーバー）
func listAllMCPTools() {
	fmt.Println("全MCPサーバーのツール一覧機能は開発中です")
}

// MCPサーバー切断の実装関数（特定サーバー）
func disconnectMCPServer(serverName string) {
	fmt.Printf("MCPサーバー '%s' の切断機能は開発中です\n", serverName)
}

// MCPサーバー切断の実装関数（全サーバー）
func disconnectAllMCPServers() {
	fmt.Println("全MCPサーバーの切断機能は開発中です")
}

// MCPサーバー追加の実装関数
func addMCPServer(name string, command []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}

	serverConfig := config.MCPServerConfig{
		Name:        name,
		Command:     command,
		Args:        []string{},
		Environment: make(map[string]string),
		WorkingDir:  "",
		Enabled:     true,
		AutoConnect: false,
	}

	if err := cfg.AddMCPServer(name, serverConfig); err != nil {
		fmt.Printf("MCPサーバー追加エラー: %v\n", err)
		return
	}

	fmt.Printf("MCPサーバー '%s' を追加しました\n", name)
}

// 検索機能の実装関数
func performSearch(pattern string, smart bool, maxResults int, includeContext bool) {
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("作業ディレクトリ取得エラー: %v\n", err)
		return
	}

	// 検索エンジンを作成
	engine := search.NewEngine(workDir)

	// プロジェクトインデックスを構築
	fmt.Println("プロジェクトインデックス構築中...")
	if err := engine.IndexProject(); err != nil {
		fmt.Printf("インデックス構築エラー: %v\n", err)
		return
	}

	if smart {
		// スマート検索を実行
		fmt.Printf("🔍 スマート検索実行中: %s\n", pattern)

		smartOptions := search.SmartSearchOptions{
			SearchOptions: search.SearchOptions{
				Pattern:      pattern,
				MaxResults:   maxResults,
				ContextLines: 2,
			},
			UseStructuralAnalysis: true,
			UseContextRanking:     true,
			IncludeASTInfo:        false, // パフォーマンス考慮
			MinRelevanceScore:     0.1,
		}

		results, err := engine.SmartSearch(smartOptions)
		if err != nil {
			fmt.Printf("スマート検索エラー: %v\n", err)
			return
		}

		displayIntelligentResults(results, includeContext)
	} else {
		// 通常検索を実行
		fmt.Printf("🔍 検索実行中: %s\n", pattern)

		searchOptions := search.SearchOptions{
			Pattern:      pattern,
			MaxResults:   maxResults,
			ContextLines: 2,
		}

		results, err := engine.SearchInFiles(searchOptions)
		if err != nil {
			fmt.Printf("検索エラー: %v\n", err)
			return
		}

		displaySearchResults(results, includeContext)
	}

	// 検索統計を表示
	stats := engine.GetIndexStats()
	fmt.Printf("\n📊 検索統計: %d件中から検索\n", stats["total_files"])

	if smart {
		intelligentStats := engine.GetIntelligentSearchStats()
		fmt.Printf("AST解析ファイル: %d件\n", intelligentStats["cached_files"])
	}
}

// インテリジェント検索結果を表示
func displayIntelligentResults(results []search.IntelligentResult, includeContext bool) {
	if len(results) == 0 {
		fmt.Println("検索結果が見つかりませんでした")
		return
	}

	fmt.Printf("\n🎯 スマート検索結果 (%d件):\n\n", len(results))

	for i, result := range results {
		// ファイル情報とスコア表示
		fmt.Printf("%d. 📁 %s:%d (スコア: %.2f)\n",
			i+1, result.File.RelativePath, result.LineNumber, result.FinalScore)

		// スコア詳細
		fmt.Printf("   構造: %.2f | コンテキスト: %.2f | コード: %.2f\n",
			result.StructuralRelevance, result.ContextRelevance, result.CodeRelevance)

		// マッチした行を表示
		fmt.Printf("   %s\n", result.Line)

		// 関連シンボル表示
		if len(result.RelatedSymbols) > 0 {
			fmt.Printf("   関連: %s\n", strings.Join(result.RelatedSymbols[:min(3, len(result.RelatedSymbols))], ", "))
		}

		// コンテキスト表示
		if includeContext && len(result.Context) > 0 {
			fmt.Println("   コンテキスト:")
			for j, contextLine := range result.Context {
				if j < 2 { // 最大2行まで表示
					fmt.Printf("     %s\n", contextLine)
				}
			}
		}

		fmt.Println()
	}
}

// 通常検索結果を表示
func displaySearchResults(results []search.SearchResult, includeContext bool) {
	if len(results) == 0 {
		fmt.Println("検索結果が見つかりませんでした")
		return
	}

	fmt.Printf("\n🔍 検索結果 (%d件):\n\n", len(results))

	for i, result := range results {
		fmt.Printf("%d. 📁 %s:%d\n", i+1, result.File.RelativePath, result.LineNumber)
		fmt.Printf("   %s\n", result.Line)

		if includeContext && len(result.Context) > 0 {
			fmt.Println("   コンテキスト:")
			for j, contextLine := range result.Context {
				if j < 2 {
					fmt.Printf("     %s\n", contextLine)
				}
			}
		}

		fmt.Println()
	}
}

// min関数（Go 1.20では標準ライブラリにない）
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ヘルスチェック実行の実装関数
func runHealthCheck() {
	fmt.Println("🏥 vyb-code ヘルスチェック開始...")

	checker := diagnostic.NewHealthChecker()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	report := checker.RunHealthChecks(ctx)
	checker.DisplayHealthStatus(report)
}

// 診断実行の実装関数
func runDiagnostics() {
	checker := diagnostic.NewHealthChecker()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	checker.RunDiagnostics(ctx)
}
