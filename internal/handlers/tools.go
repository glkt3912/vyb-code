package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
	"github.com/glkt/vyb-code/internal/security"
	"github.com/glkt/vyb-code/internal/tools"
	"github.com/spf13/cobra"
)

// ToolsHandler はツール関連機能のハンドラー
type ToolsHandler struct {
	log logger.Logger
}

// NewToolsHandler はツールハンドラーの新しいインスタンスを作成
func NewToolsHandler(log logger.Logger) *ToolsHandler {
	return &ToolsHandler{log: log}
}

// ExecuteCommand はセキュアなコマンド実行
func (h *ToolsHandler) ExecuteCommand(command string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// セキュリティ制約
	constraints := &security.Constraints{
		AllowedCommands: []string{"ls", "cat", "pwd", "git", "npm", "go", "make"},
		MaxTimeout:      cfg.CommandTimeout,
	}

	// コマンド実行器を作成
	executor := tools.NewCommandExecutor(constraints, cfg.WorkspacePath)

	result, err := executor.Execute(command)
	if err != nil {
		return fmt.Errorf("コマンド実行エラー: %w", err)
	}

	h.log.Info("コマンド実行結果", map[string]interface{}{
		"command":   result.Command,
		"exit_code": result.ExitCode,
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
		"duration":  result.Duration,
		"timed_out": result.TimedOut,
	})

	return nil
}

// SearchFiles はファイル検索
func (h *ToolsHandler) SearchFiles(pattern string, smartMode bool, maxResults int, showContext bool) error {
	h.log.Info("検索機能実行", map[string]interface{}{
		"pattern":      pattern,
		"smart_mode":   smartMode,
		"max_results":  maxResults,
		"show_context": showContext,
	})

	// 設定読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// Grepツールを作成して検索実行
	grepTool := tools.NewGrepTool(cfg.WorkspacePath)

	grepOptions := tools.GrepOptions{
		Pattern:         pattern,
		Path:            cfg.WorkspacePath,
		CaseInsensitive: true,
		LineNumbers:     true,
		HeadLimit:       maxResults,
		OutputMode:      "content",
	}

	if showContext {
		grepOptions.ContextBefore = 3
		grepOptions.ContextAfter = 3
	}

	result, err := grepTool.Search(grepOptions)
	if err != nil {
		return fmt.Errorf("検索エラー: %w", err)
	}

	// 結果を表示
	if result.IsError {
		fmt.Printf("検索エラー: %s\n", result.Content)
	} else {
		fmt.Printf("🔍 検索結果:\n%s\n", result.Content)
	}

	return nil
}

// FindFiles はファイル名パターンで検索
func (h *ToolsHandler) FindFiles(pattern string) error {
	h.log.Info("ファイル検索機能実行", map[string]interface{}{
		"pattern": pattern,
	})

	// 設定読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// Globツールを使用してファイル検索
	globTool := tools.NewGlobTool(cfg.WorkspacePath)
	result, err := globTool.Find(pattern, cfg.WorkspacePath)
	if err != nil {
		return fmt.Errorf("ファイル検索エラー: %w", err)
	}

	// 結果を表示
	if result.IsError {
		fmt.Printf("ファイル検索エラー: %s\n", result.Content)
	} else {
		fmt.Printf("📁 見つかったファイル:\n%s\n", result.Content)
	}

	return nil
}

// GrepFiles はgrep検索
func (h *ToolsHandler) GrepFiles(pattern string, filePattern string) error {
	h.log.Info("grep検索機能実行", map[string]interface{}{
		"pattern":      pattern,
		"file_pattern": filePattern,
	})

	// 設定読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// Grepツールを作成してgrep検索実行
	grepTool := tools.NewGrepTool(cfg.WorkspacePath)

	grepOptions := tools.GrepOptions{
		Pattern:         pattern,
		Path:            cfg.WorkspacePath,
		Glob:            filePattern,
		CaseInsensitive: true,
		LineNumbers:     true,
		HeadLimit:       100,
		OutputMode:      "content",
		ContextBefore:   2,
		ContextAfter:    2,
	}

	result, err := grepTool.Search(grepOptions)
	if err != nil {
		return fmt.Errorf("grep検索エラー: %w", err)
	}

	// 結果を表示
	if result.IsError {
		fmt.Printf("grep検索エラー: %s\n", result.Content)
	} else {
		fmt.Printf("🔍 grep検索結果:\n%s\n", result.Content)
	}

	return nil
}

// AnalyzeProject はプロジェクト分析
func (h *ToolsHandler) AnalyzeProject(path string) error {
	h.log.Info("プロジェクト分析機能実行", map[string]interface{}{
		"path": path,
	})

	// 設定読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// 分析対象パスの決定
	analyzePath := cfg.WorkspacePath
	if path != "" {
		if filepath.IsAbs(path) {
			analyzePath = path
		} else {
			analyzePath = filepath.Join(cfg.WorkspacePath, path)
		}
	}

	// セキュリティ制約設定
	constraints := &security.Constraints{
		AllowedCommands: []string{"git", "ls", "find"},
		MaxTimeout:      cfg.CommandTimeout,
	}

	// プロジェクト分析器を作成
	analyzer := tools.NewProjectAnalyzer(constraints, analyzePath)

	// プロジェクト分析実行
	analysis, err := analyzer.AnalyzeProject()
	if err != nil {
		return fmt.Errorf("プロジェクト分析エラー: %w", err)
	}

	// 結果を表示
	fmt.Printf("📊 プロジェクト分析結果:\n")
	fmt.Printf("  総ファイル数: %d\n", analysis.TotalFiles)
	fmt.Printf("  言語別ファイル数:\n")
	for lang, count := range analysis.FilesByLanguage {
		fmt.Printf("    %s: %d\n", lang, count)
	}

	if analysis.GitInfo != nil {
		fmt.Printf("  Git情報:\n")
		fmt.Printf("    現在のブランチ: %s\n", analysis.GitInfo.CurrentBranch)
		fmt.Printf("    ブランチ数: %d\n", len(analysis.GitInfo.Branches))
		fmt.Printf("    ステータス: %s\n", analysis.GitInfo.Status)
	}

	fmt.Printf("  依存関係: %d個\n", len(analysis.Dependencies))

	return nil
}

// QuickGitStatus はGitステータスの簡易表示
func (h *ToolsHandler) QuickGitStatus() error {
	h.log.Info("Gitステータス機能実行", nil)

	// 設定読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// セキュリティ制約設定
	constraints := &security.Constraints{
		AllowedCommands: []string{"git"},
		MaxTimeout:      cfg.CommandTimeout,
	}

	// Git操作器を作成
	gitOps := tools.NewGitOperations(constraints, cfg.WorkspacePath)

	// Gitステータス取得
	status, err := gitOps.GetStatus()
	if err != nil {
		return fmt.Errorf("gitステータス取得エラー: %w", err)
	}

	// ブランチ情報取得
	branchesResult, err := gitOps.GetBranches()
	if err != nil {
		h.log.Warn("ブランチ情報取得に失敗", map[string]interface{}{"error": err.Error()})
	}

	// 現在のブランチ取得
	currentBranch, err := gitOps.GetCurrentBranch()
	if err != nil {
		h.log.Warn("現在のブランチ取得に失敗", map[string]interface{}{"error": err.Error()})
	}

	// 結果を表示
	fmt.Printf("🌿 Git ステータス:\n")
	if status.Stdout != "" {
		fmt.Printf("%s\n", status.Stdout)
	} else {
		fmt.Printf("  作業ディレクトリはクリーンです\n")
	}

	if currentBranch != "" {
		fmt.Printf("📍 現在のブランチ: %s\n", currentBranch)
	}

	if branchesResult != nil && branchesResult.Stdout != "" {
		fmt.Printf("\n📝 ブランチ一覧:\n")
		branches := strings.Split(strings.TrimSpace(branchesResult.Stdout), "\n")
		for _, branch := range branches {
			branch = strings.TrimSpace(branch)
			if branch != "" {
				fmt.Printf("  %s\n", branch)
			}
		}
	}

	return nil
}

// AutoBuild は自動ビルド実行
func (h *ToolsHandler) AutoBuild() error {
	h.log.Info("自動ビルド機能実行", nil)

	// 設定読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// セキュリティ制約設定
	constraints := &security.Constraints{
		AllowedCommands: []string{"make", "go", "npm", "yarn", "cargo", "mvn", "gradle", "python", "pip"},
		MaxTimeout:      cfg.CommandTimeout * 3, // ビルドは長時間かかる可能性があるので3倍に設定
	}

	// プロジェクトパスを決定
	workspacePath := cfg.WorkspacePath
	if workspacePath == "" {
		if cwd, err := os.Getwd(); err == nil {
			workspacePath = cwd
		} else {
			return fmt.Errorf("現在のディレクトリ取得エラー: %w", err)
		}
	}

	// ビルドマネージャーを作成
	buildManager := tools.NewBuildManager(constraints, workspacePath)

	// 自動ビルド実行
	result, err := buildManager.AutoBuild()
	if err != nil {
		return fmt.Errorf("自動ビルドエラー: %w", err)
	}

	// 結果を表示
	fmt.Printf("🔨 ビルド結果:\n")
	fmt.Printf("  ビルドシステム: %s\n", result.BuildSystem)
	fmt.Printf("  コマンド: %s\n", result.Command)
	fmt.Printf("  実行時間: %v\n", result.Duration)

	if result.Success {
		fmt.Printf("  ✅ ビルド成功\n")
		if result.Output != "" {
			fmt.Printf("出力:\n%s\n", result.Output)
		}
	} else {
		fmt.Printf("  ❌ ビルド失敗 (終了コード: %d)\n", result.ExitCode)
		if result.ErrorOutput != "" {
			fmt.Printf("エラー:\n%s\n", result.ErrorOutput)
		}
	}

	return nil
}

// AutoTest は自動テスト実行
func (h *ToolsHandler) AutoTest() error {
	h.log.Info("自動テスト機能実行", nil)

	// 設定読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// セキュリティ制約設定
	constraints := &security.Constraints{
		AllowedCommands: []string{"ls", "make", "go", "npm", "yarn", "cargo", "mvn", "gradle", "python", "pytest", "jest"},
		MaxTimeout:      cfg.CommandTimeout * 5, // テストはさらに長時間かかる可能性があるので5倍に設定
	}

	// プロジェクトパスを決定
	workspacePath := cfg.WorkspacePath
	if workspacePath == "" {
		if cwd, err := os.Getwd(); err == nil {
			workspacePath = cwd
		} else {
			return fmt.Errorf("現在のディレクトリ取得エラー: %w", err)
		}
	}

	// コマンド実行器を作成
	executor := tools.NewCommandExecutor(constraints, workspacePath)

	// テストコマンドを自動検出して実行
	var testCommand string
	var buildSystem string

	// Go プロジェクト
	if _, err := executor.Execute("ls go.mod"); err == nil {
		testCommand = "go test ./..."
		buildSystem = "Go"
	} else if _, err := executor.Execute("ls package.json"); err == nil {
		// Node.js プロジェクト
		testCommand = "npm test"
		buildSystem = "Node.js"
	} else if _, err := executor.Execute("ls Makefile"); err == nil {
		// Makefile プロジェクト
		testCommand = "make test"
		buildSystem = "Make"
	} else {
		return fmt.Errorf("テスト可能なプロジェクトが見つかりません")
	}

	// テスト実行
	result, err := executor.Execute(testCommand)
	if err != nil {
		return fmt.Errorf("テスト実行エラー: %w", err)
	}

	// 結果を表示
	fmt.Printf("🧪 テスト結果:\n")
	fmt.Printf("  テストシステム: %s\n", buildSystem)
	fmt.Printf("  コマンド: %s\n", testCommand)
	fmt.Printf("  実行時間: %v\n", result.Duration)

	if result.ExitCode == 0 {
		fmt.Printf("  ✅ テスト成功\n")
		if result.Stdout != "" {
			fmt.Printf("出力:\n%s\n", result.Stdout)
		}
	} else {
		fmt.Printf("  ❌ テスト失敗 (終了コード: %d)\n", result.ExitCode)
		if result.Stderr != "" {
			fmt.Printf("エラー:\n%s\n", result.Stderr)
		}
	}

	return nil
}

// CreateToolCommands はツール関連のcobraコマンドを作成
func (h *ToolsHandler) CreateToolCommands() []*cobra.Command {
	var commands []*cobra.Command

	// exec コマンド
	execCmd := &cobra.Command{
		Use:   "exec [command]",
		Short: "Execute shell command securely",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			command := strings.Join(args, " ")
			return h.ExecuteCommand(command)
		},
	}

	// search コマンド
	searchCmd := &cobra.Command{
		Use:   "search [pattern]",
		Short: "Search across project files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			smart, _ := cmd.Flags().GetBool("smart")
			maxResults, _ := cmd.Flags().GetInt("max-results")
			context, _ := cmd.Flags().GetBool("context")
			return h.SearchFiles(args[0], smart, maxResults, context)
		},
	}
	searchCmd.Flags().Bool("smart", false, "Enable intelligent search")
	searchCmd.Flags().Int("max-results", 50, "Maximum number of results")
	searchCmd.Flags().Bool("context", false, "Show context lines")

	// find コマンド
	findCmd := &cobra.Command{
		Use:   "find [pattern]",
		Short: "Find files by name pattern",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.FindFiles(args[0])
		},
	}

	// grep コマンド
	grepCmd := &cobra.Command{
		Use:   "grep [pattern]",
		Short: "Advanced grep with context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePattern, _ := cmd.Flags().GetString("files")
			return h.GrepFiles(args[0], filePattern)
		},
	}
	grepCmd.Flags().String("files", "*", "File pattern to search in")

	// analyze コマンド
	analyzeCmd := &cobra.Command{
		Use:   "analyze [path]",
		Short: "Analyze project structure",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) > 0 {
				path = args[0]
			}
			return h.AnalyzeProject(path)
		},
	}

	// s コマンド (git status shortcut)
	statusCmd := &cobra.Command{
		Use:   "s",
		Short: "Quick git status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.QuickGitStatus()
		},
	}

	// build コマンド
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Auto-detect and build project",
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.AutoBuild()
		},
	}

	// test コマンド
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Auto-detect and run tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.AutoTest()
		},
	}

	commands = append(commands, execCmd, searchCmd, findCmd, grepCmd)
	commands = append(commands, analyzeCmd, statusCmd, buildCmd, testCmd)

	return commands
}
