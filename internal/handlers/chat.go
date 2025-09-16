package handlers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/ai"
	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/contextmanager"
	"github.com/glkt/vyb-code/internal/input"
	"github.com/glkt/vyb-code/internal/interactive"
	"github.com/glkt/vyb-code/internal/llm"
	"github.com/glkt/vyb-code/internal/logger"
	"github.com/glkt/vyb-code/internal/performance"
	"github.com/glkt/vyb-code/internal/security"
	"github.com/glkt/vyb-code/internal/streaming"
	"github.com/glkt/vyb-code/internal/tools"
)

// ChatHandler はチャット機能のハンドラー（統合システム）
type ChatHandler struct {
	log                logger.Logger
	interactiveManager interactive.SessionManager
	responseHistory    []string                     // 全ての完全な応答履歴を保持
	streamingManager   *streaming.Manager           // ストリーミング表示管理
	completer          *input.AdvancedCompleter     // 高度な補完機能
	perfMonitor        *performance.RealtimeMonitor // パフォーマンス監視
}

// NewChatHandler はチャットハンドラーを作成
func NewChatHandler(log logger.Logger, cfg *config.Config) *ChatHandler {
	// ストリーミング設定を作成（より目立つ設定）
	streamConfig := streaming.DefaultStreamConfig()
	streamConfig.TokenDelay = 25 * time.Millisecond     // 少し遅めで読みやすく
	streamConfig.SentenceDelay = 150 * time.Millisecond // 文末でより長い間隔
	streamConfig.EnableStreaming = true                 // 必ず有効

	// 作業ディレクトリを取得
	workDir, _ := os.Getwd()

	// パフォーマンス監視を作成
	var perfMonitor *performance.RealtimeMonitor
	if cfg != nil {
		perfMonitor = performance.NewRealtimeMonitor(cfg)
	}

	return &ChatHandler{
		log: log,
		// interactiveManagerは実行時に初期化
		streamingManager: streaming.NewManager(streamConfig),
		completer:        input.NewAdvancedCompleter(workDir),
		perfMonitor:      perfMonitor,
	}
}

// 旧関数名の互換性を維持
func NewChatHandlerWithMigration(log logger.Logger, migrationConfig *config.GradualMigrationConfig) *ChatHandler {
	return NewChatHandler(log, nil)
}

// initializeInteractiveManager はInteractiveSessionManagerを初期化
func (h *ChatHandler) initializeInteractiveManager(cfg *config.Config) error {
	if h.interactiveManager != nil {
		return nil // 既に初期化済み
	}

	// LLMプロバイダーを作成
	llmProvider := llm.NewOllamaClient(cfg.BaseURL)

	// ContextManagerを作成
	contextManager := contextmanager.NewSmartContextManager()

	// AIServiceを作成 (インターフェース互換性のため簡単なアダプター作成)
	// 現在はInteractiveSessionManagerでLLMプロバイダーを直接使用するため、nilでも動作する
	var aiService *ai.AIService = nil

	// EditToolを作成
	editTool := tools.NewEditTool(
		security.NewDefaultConstraints("."),
		".",
		10*1024*1024,
	)

	// VibeConfigを作成
	vibeConfig := interactive.DefaultVibeConfig()

	// InteractiveSessionManagerを作成
	h.interactiveManager = interactive.NewInteractiveSessionManager(
		contextManager,
		llmProvider,
		aiService,
		editTool,
		vibeConfig,
		cfg.Model,
		cfg,
	)

	h.log.Info("Interactive session manager initialized", nil)
	return nil
}

// runInteractiveLoop はインタラクティブな対話ループを実行
func (h *ChatHandler) runInteractiveLoop(sessionID string, cfg *config.Config) error {
	// 高度な入力システムを使用（Backspace対応）
	reader := h.createAdvancedInputReader()

	// ClaudeCode風のウェルカムメッセージ
	h.showWelcomeMessage()

	for {
		// ClaudeCode風のプロンプト表示（高度な入力システムが処理）
		input, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				fmt.Printf("\n👋 Goodbye!\n")
				break
			}
			// Ctrl+C (interrupted) の場合も正常終了として扱う
			if strings.Contains(err.Error(), "interrupted") {
				fmt.Printf("\n👋 Goodbye!\n")
				break
			}
			fmt.Printf("入力エラー: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Printf("\n👋 Goodbye!\n")
			break
		}

		// 展開コマンドの処理（ストリーミング対応）
		if input == "show" || input == "more" || input == "full" {
			if len(h.responseHistory) > 0 {
				// 最新の応答を展開
				latestResponse := h.responseHistory[len(h.responseHistory)-1]
				fmt.Printf("\n\033[38;5;27m🤖 Assistant (Full Content)\033[0m\n")

				// 完全なコンテンツをストリーミング表示
				streamOptions := &streaming.StreamOptions{
					Type:            streaming.StreamTypeUIDisplay,
					EnableInterrupt: false, // 展開時は中断無効
				}

				err := h.streamingManager.ProcessString(context.Background(), latestResponse, os.Stdout, streamOptions)
				if err != nil {
					fmt.Printf("%s", latestResponse)
				}
				fmt.Println()
				continue
			} else {
				fmt.Printf("\n\033[38;5;196m✗ Error\033[0m\nNo previous response to expand.\n\n")
				continue
			}
		}

		// ユーザー入力を表示（ClaudeCode風）
		fmt.Printf("\n\033[38;5;34m▶ You\033[0m\n%s\n\n", h.formatForDisplay(input))

		// パフォーマンス測定開始
		startTime := time.Now()
		if h.perfMonitor != nil {
			h.perfMonitor.RecordProactiveUsage("chat_request")
		}

		// インタラクティブセッションで処理（独自のプログレス表示を使用）
		response, err := h.interactiveManager.ProcessUserInput(context.Background(), sessionID, input)

		// パフォーマンス測定記録
		duration := time.Since(startTime)
		if h.perfMonitor != nil {
			h.perfMonitor.RecordResponseTime(duration)
			h.perfMonitor.RecordLLMLatency(duration) // 簡略化
		}

		if err != nil {
			fmt.Printf("\033[38;5;196m✗ Error\033[0m\n%s\n\n", err.Error())
			continue
		}

		// 完全な応答を履歴に追加
		h.responseHistory = append(h.responseHistory, response.Message)

		// 履歴を最大10件に制限
		if len(h.responseHistory) > 10 {
			h.responseHistory = h.responseHistory[1:]
		}

		// AI応答をストリーミング表示（ClaudeCode風）
		fmt.Printf("\033[38;5;27m🤖 Assistant\033[0m\n")

		// 折り畳み処理のため、まず表示用コンテンツを取得
		displayContent := h.formatForDisplay(response.Message)

		// Claude Codeライクなストリーミング表示（より積極的に）
		if len(displayContent) > 30 {
			// ほとんどのコンテンツでストリーミング表示を使用
			streamOptions := &streaming.StreamOptions{
				Type:            streaming.StreamTypeUIDisplay,
				EnableInterrupt: false, // 通常応答では中断無効
			}

			err = h.streamingManager.ProcessString(context.Background(), displayContent, os.Stdout, streamOptions)
			if err != nil {
				// ストリーミングエラー時は通常表示にフォールバック
				fmt.Printf("%s", displayContent)
			}
		} else {
			// 非常に短いコンテンツは直接表示
			fmt.Printf("%s", displayContent)
		}

		// Claude Code風メタデータ表示
		h.showResponseMetadata(duration, len(response.Message))

		// プロアクティブな機能提案
		h.showProactiveSuggestions(input, response.Message)

		fmt.Println() // 改行
	}

	// 高度な入力システムでは scanner.Err() は不要

	return nil
}

// showWelcomeMessage はClaudeCode風のウェルカムメッセージを表示
func (h *ChatHandler) showWelcomeMessage() {
	// 画面クリア（一度だけ）
	fmt.Print("\033[2J\033[H")

	// Claude Code風のウェルカムメッセージ
	fmt.Println("\033[1m🤖 vyb-code · AI Coding Assistant\033[0m")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()
	fmt.Println("🎯 \033[32mWelcome to intelligent coding!\033[0m")
	fmt.Println("💡 \033[90mIntelligent suggestions, streaming responses, and smart completion\033[0m")
	fmt.Println()
	fmt.Println("🔧 \033[90mCommands: '\033[36mhelp\033[90m' for help\033[0m")
	fmt.Println("🚪 \033[90mExit: '\033[36mexit\033[90m' or '\033[36mquit\033[90m' or \033[36mCtrl+C\033[90m\033[0m")

	// プロジェクト情報を表示
	workDir, _ := os.Getwd()
	fmt.Printf("📂 \033[90mProject: \033[36m%s\033[0m\n", filepath.Base(workDir))

	// パフォーマンス情報があれば表示
	if h.perfMonitor != nil {
		fmt.Printf("⚡ \033[90mPerformance monitoring: \033[32menabled\033[0m\n")
	}

	fmt.Println()
}

// showInputPrompt はClaudeCode風の入力プロンプトを表示
func (h *ChatHandler) showInputPrompt() {
	fmt.Print("💬 You: ")
}

// readInputWithCompletion は補完機能付きの入力読み取り
func (h *ChatHandler) readInputWithCompletion() (string, error) {
	// 基本的な実装: 将来的にはreadlineライブラリやタブ補完を統合
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("input scanning failed")
	}

	input := strings.TrimSpace(scanner.Text())

	// より積極的な補完候補表示
	if len(input) > 1 && len(input) < 25 {
		suggestions := h.completer.GetAdvancedSuggestions(input)
		if len(suggestions) > 0 && len(suggestions) <= 4 {
			fmt.Printf("\033[90m💡 Smart suggestions: ")
			for i, suggestion := range suggestions {
				if i > 0 {
					fmt.Printf(" • ")
				}
				// タイプと説明を含む表示
				fmt.Printf("\033[36m%s\033[90m", suggestion.Text)
				if suggestion.Description != "" && suggestion.Description != "ファジーマッチング" {
					fmt.Printf(" (%s)", suggestion.Description)
				}
			}
			fmt.Printf("\033[0m\n")
		}
	}

	return input, scanner.Err()
}

// createAdvancedInputReader は高度な入力リーダーを作成（Backspace対応）
func (h *ChatHandler) createAdvancedInputReader() *input.Reader {
	// 高度な入力リーダーを作成
	reader := input.NewReader()

	// ClaudeCode風のプロンプトを設定
	reader.SetPrompt("💬 You: ")

	// セキュリティとパフォーマンス最適化を有効化
	reader.EnableSecurity()
	reader.EnableOptimization()

	return reader
}

// showResponseMetadata はClaudeCode風のメタデータを表示
func (h *ChatHandler) showResponseMetadata(duration time.Duration, responseLength int) {
	// 応答時間をフォーマット
	durationStr := h.formatDuration(duration)

	// トークン数を推定（簡易計算: 文字数/4）
	estimatedTokens := responseLength / 4

	// パフォーマンス情報があれば表示
	var perfInfo string
	if h.perfMonitor != nil {
		metrics := h.perfMonitor.GetMetrics()
		memUsage := metrics.MemoryUsage.Current
		perfInfo = fmt.Sprintf("· %.1fMB", memUsage)
	}

	// Claude Code風フォーマット: (2.3s · ↓ 456 tokens · 45.2MB)
	metadataStr := fmt.Sprintf("\033[90m(%s · ↓ %d tokens%s)\033[0m",
		durationStr, estimatedTokens, perfInfo)

	fmt.Printf("%s", metadataStr)
}

// formatDuration は時間を読みやすい形式にフォーマット
func (h *ChatHandler) formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d)/float64(time.Millisecond))
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// showPerformanceSummary はパフォーマンスサマリーを表示
func (h *ChatHandler) showPerformanceSummary() {
	if h.perfMonitor == nil {
		return
	}

	summary := h.perfMonitor.GeneratePerformanceSummary()
	fmt.Printf("\n%s\n", summary)
}

// showProactiveSuggestions はClaudeCode風のプロアクティブな提案を表示
func (h *ChatHandler) showProactiveSuggestions(userInput, response string) {
	suggestions := h.generateContextualSuggestions(userInput, response)

	if len(suggestions) > 0 {
		fmt.Printf("\n\033[90m💡 Quick actions: ")
		for i, suggestion := range suggestions {
			if i > 0 {
				fmt.Printf(" • ")
			}
			fmt.Printf("%s", suggestion)
		}
		fmt.Printf("\033[0m")
	}
}

// generateContextualSuggestions は文脈に応じた提案を生成
func (h *ChatHandler) generateContextualSuggestions(userInput, response string) []string {
	var suggestions []string

	inputLower := strings.ToLower(userInput)
	responseLower := strings.ToLower(response)

	// Git関連の提案
	if strings.Contains(inputLower, "git") || strings.Contains(responseLower, "git") {
		suggestions = append(suggestions, "git status")
	}

	// ファイル編集の提案
	if strings.Contains(responseLower, "edit") || strings.Contains(responseLower, "modify") || strings.Contains(responseLower, "change") {
		suggestions = append(suggestions, "show files")
	}

	// テスト実行の提案
	if strings.Contains(responseLower, "test") || strings.Contains(responseLower, "bug") {
		suggestions = append(suggestions, "run tests")
	}

	// ビルドの提案
	if strings.Contains(responseLower, "build") || strings.Contains(responseLower, "compile") {
		suggestions = append(suggestions, "build project")
	}

	// プロジェクト分析の提案
	if strings.Contains(inputLower, "analyze") || strings.Contains(inputLower, "structure") {
		suggestions = append(suggestions, "analyze project")
	}

	// 最大3つまでに制限
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

// isContentFolded は表示コンテンツが折り畳まれているかチェック
func (h *ChatHandler) isContentFolded(original, display string) bool {
	return len(original) != len(display) || strings.Contains(display, "more lines hidden")
}

// formatForDisplay はClaudeCode風に非常に長いコンテンツのみを折りたたみ表示
func (h *ChatHandler) formatForDisplay(content string) string {
	lines := strings.Split(content, "\n")

	// 非常に長い場合のみ折りたたむ（かなり寛容に）
	if len(lines) > 50 || len(content) > 3000 {
		// 最初の35行を表示
		displayLines := lines
		if len(lines) > 35 {
			displayLines = lines[:35]
		}

		preview := strings.Join(displayLines, "\n")

		hiddenLines := len(lines) - len(displayLines)
		if hiddenLines > 0 {
			preview += fmt.Sprintf("\n\n\033[38;5;242m⋮ %d more lines hidden\033[0m", hiddenLines)
			preview += "\n\033[38;5;244m💡 Type 'show' to expand full content\033[0m"
		}

		return preview
	}

	// 長い場合は最初の25行まで表示
	if len(lines) > 30 || len(content) > 2000 {
		displayLines := lines
		if len(lines) > 25 {
			displayLines = lines[:25]
		}

		preview := strings.Join(displayLines, "\n")

		hiddenLines := len(lines) - len(displayLines)
		if hiddenLines > 0 {
			preview += fmt.Sprintf("\n\n\033[38;5;242m⋮ %d more lines\033[0m", hiddenLines)
		}

		return preview
	}

	// それ以外はそのまま表示（実用的に）
	return content
}

// 統合されたバイブコーディング機能
func (h *ChatHandler) StartVibeChat(cfg *config.Config) error {
	fmt.Printf("🚀 Starting vibe coding mode...\n")

	// InteractiveSessionManagerを初期化
	if err := h.initializeInteractiveManager(cfg); err != nil {
		return fmt.Errorf("interactive manager initialization failed: %w", err)
	}

	// 新しいインタラクティブセッションを開始
	session, err := h.interactiveManager.CreateSession(interactive.CodingSessionTypeGeneral)
	if err != nil {
		return fmt.Errorf("vibe coding session creation failed: %w", err)
	}
	sessionID := session.ID
	if err != nil {
		return fmt.Errorf("vibe coding session start failed: %w", err)
	}

	fmt.Printf("🎵 Vibe coding session started: %s\n", sessionID)

	// パフォーマンス監視を開始
	if h.perfMonitor != nil {
		err := h.perfMonitor.Start()
		if err != nil {
			fmt.Printf("⚠️ Performance monitoring failed to start: %v\n", err)
		}
	}

	// インタラクティブループを開始
	return h.runInteractiveLoop(sessionID, cfg)
}

func (h *ChatHandler) StartChatSession(cfg *config.Config) error {
	fmt.Printf("💬 Starting chat session...\n")

	// InteractiveSessionManagerを初期化
	if err := h.initializeInteractiveManager(cfg); err != nil {
		return fmt.Errorf("interactive manager initialization failed: %w", err)
	}

	// 新しいチャットセッションを開始
	session, err := h.interactiveManager.CreateSession(interactive.CodingSessionTypeGeneral)
	if err != nil {
		return fmt.Errorf("chat session creation failed: %w", err)
	}
	sessionID := session.ID
	if err != nil {
		return fmt.Errorf("chat session start failed: %w", err)
	}

	fmt.Printf("💬 Chat session started: %s\n", sessionID)

	// パフォーマンス監視を開始
	if h.perfMonitor != nil {
		err := h.perfMonitor.Start()
		if err != nil {
			fmt.Printf("⚠️ Performance monitoring failed to start: %v\n", err)
		}
	}

	// インタラクティブループを開始
	return h.runInteractiveLoop(sessionID, cfg)
}

func (h *ChatHandler) ContinueSession(resumeID string, cfg *config.Config, terminalMode bool, planMode bool) error {
	fmt.Printf("🔄 Continuing session: %s\n", resumeID)

	// InteractiveSessionManagerを初期化
	if err := h.initializeInteractiveManager(cfg); err != nil {
		return fmt.Errorf("interactive manager initialization failed: %w", err)
	}

	// セッションを再開
	fmt.Printf("🎯 Session resumed: %s\n", resumeID)

	// インタラクティブループを開始
	return h.runInteractiveLoop(resumeID, cfg)
}

func (h *ChatHandler) RunSingleQuery(query string, resumeID string, cfg *config.Config) error {
	// InteractiveSessionManagerを初期化
	if err := h.initializeInteractiveManager(cfg); err != nil {
		return err
	}

	var sessionID string
	var err error

	if resumeID != "" {
		// 既存セッションを使用
		sessionID = resumeID
		fmt.Printf("🔄 Using existing session: %s\n", sessionID)
	} else {
		// 新しい一時セッションを作成
		session, createErr := h.interactiveManager.CreateSession(interactive.CodingSessionTypeGeneral)
		if createErr != nil {
			return fmt.Errorf("temporary session creation failed: %w", createErr)
		}
		sessionID = session.ID
		fmt.Printf("📝 Created temporary session: %s\n", sessionID)
	}

	// クエリを処理
	response, err := h.interactiveManager.ProcessUserInput(context.Background(), sessionID, query)
	if err != nil {
		return fmt.Errorf("query processing failed: %w", err)
	}

	fmt.Printf("🤖 Response: %s\n", response.Message)
	return nil
}

// handleSessionContinuation - 旧互換性関数
func (h *ChatHandler) handleSessionContinuation(resumeID string, terminalMode bool, planMode bool) error {
	return fmt.Errorf("use ContinueSession method instead")
}

// GetMigrationStatus は移行状況を返す
func (h *ChatHandler) GetMigrationStatus() map[string]interface{} {
	return map[string]interface{}{
		"migration_status":    "completed",
		"system_type":         "unified",
		"interactive_manager": h.interactiveManager != nil,
		"status":              "fully integrated system using InteractiveSessionManager",
	}
}

// Handler インターフェース実装

// Initialize はハンドラーを初期化（既存の初期化処理を分離）
func (h *ChatHandler) Initialize(cfg *config.Config) error {
	if h.interactiveManager != nil {
		return nil // 既に初期化済み
	}
	return h.initializeInteractiveManager(cfg)
}

// GetMetadata はハンドラーのメタデータを返す
func (h *ChatHandler) GetMetadata() HandlerMetadata {
	return HandlerMetadata{
		Name:        "chat",
		Version:     "1.0.0",
		Description: "Claude Code風対話インターフェース",
		Capabilities: []string{
			"interactive_chat",
			"streaming_response",
			"context_management",
			"vibe_coding",
			"session_management",
		},
		Dependencies: []string{
			"interactive",
			"streaming",
			"input",
			"performance",
		},
		Config: map[string]string{
			"default_mode": "vibe_coding",
			"ui_style":     "claude_code",
		},
	}
}

// Health はハンドラーの健全性をチェック
func (h *ChatHandler) Health(ctx context.Context) error {
	// 基本的な健全性チェック
	if h.log == nil {
		return fmt.Errorf("logger not initialized")
	}
	if h.streamingManager == nil {
		return fmt.Errorf("streaming manager not initialized")
	}
	if h.completer == nil {
		return fmt.Errorf("completer not initialized")
	}
	// インタラクティブマネージャーは遅延初期化されるためオプショナル
	return nil
}
