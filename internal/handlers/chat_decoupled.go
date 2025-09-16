package handlers

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
)

// DecoupledChatHandler は依存関係を分離したチャットハンドラー
type DecoupledChatHandler struct {
	log              logger.Logger
	resolver         DependencyResolver
	responseHistory  []string
	
	// 遅延初期化される依存関係
	llmProvider      LLMProvider
	streamingManager StreamingManager
	completer        InputCompleter
	perfMonitor      PerformanceMonitor
	interactiveManager InteractiveSessionManager
	initialized      bool
}

// NewDecoupledChatHandler は依存関係分離型チャットハンドラーを作成
func NewDecoupledChatHandler(log logger.Logger, resolver DependencyResolver) *DecoupledChatHandler {
	return &DecoupledChatHandler{
		log:      log,
		resolver: resolver,
	}
}

// Initialize はハンドラーを初期化（依存関係を解決）
func (h *DecoupledChatHandler) Initialize(cfg *config.Config) error {
	if h.initialized {
		return nil
	}

	var err error

	// 依存関係を遅延解決
	h.llmProvider, err = h.resolver.ResolveLLMProvider(cfg)
	if err != nil {
		h.log.Warn("LLM provider initialization failed, using null provider", map[string]interface{}{
			"error": err.Error(),
		})
		h.llmProvider = &NullLLMProvider{}
	}

	h.streamingManager, err = h.resolver.ResolveStreamingManager(cfg)
	if err != nil {
		h.log.Warn("Streaming manager initialization failed", map[string]interface{}{
			"error": err.Error(),
		})
		h.streamingManager = &NullStreamingManager{}
	}

	h.completer, err = h.resolver.ResolveInputCompleter(cfg)
	if err != nil {
		h.log.Warn("Input completer initialization failed", map[string]interface{}{
			"error": err.Error(),
		})
		h.completer = &NullInputCompleter{}
	}

	h.perfMonitor, err = h.resolver.ResolvePerformanceMonitor(cfg, h.log)
	if err != nil {
		h.log.Warn("Performance monitor initialization failed", map[string]interface{}{
			"error": err.Error(),
		})
		h.perfMonitor = &NullPerformanceMonitor{}
	}

	h.interactiveManager, err = h.resolver.ResolveInteractiveManager(cfg, h.log)
	if err != nil {
		h.log.Warn("Interactive manager initialization failed", map[string]interface{}{
			"error": err.Error(),
		})
		h.interactiveManager = &NullInteractiveManager{}
	}

	h.initialized = true
	h.log.Info("Decoupled chat handler initialized", nil)
	
	return nil
}

// GetMetadata はハンドラーのメタデータを返す
func (h *DecoupledChatHandler) GetMetadata() HandlerMetadata {
	return HandlerMetadata{
		Name:        "decoupled_chat",
		Version:     "1.0.0",
		Description: "依存関係分離型チャットハンドラー",
		Capabilities: []string{
			"interactive_chat",
			"dependency_injection",
			"lazy_initialization",
			"null_object_pattern",
		},
		Dependencies: []string{
			"config",
			"logger",
		},
		Config: map[string]string{
			"architecture": "decoupled",
			"pattern":      "dependency_injection",
		},
	}
}

// Health はハンドラーの健全性をチェック
func (h *DecoupledChatHandler) Health(ctx context.Context) error {
	if h.log == nil {
		return fmt.Errorf("logger not initialized")
	}
	if h.resolver == nil {
		return fmt.Errorf("dependency resolver not initialized")
	}
	if h.initialized && h.llmProvider == nil {
		return fmt.Errorf("LLM provider not resolved")
	}
	return nil
}

// StartVibeChat はバイブコーディングモードを開始
func (h *DecoupledChatHandler) StartVibeChat(cfg *config.Config) error {
	if !h.initialized {
		if err := h.Initialize(cfg); err != nil {
			return fmt.Errorf("initialization failed: %w", err)
		}
	}

	fmt.Printf("🚀 Starting decoupled vibe coding mode...\n")

	// セッションを作成
	session, err := h.interactiveManager.CreateSession(1) // GeneralSessionType
	if err != nil {
		return fmt.Errorf("session creation failed: %w", err)
	}

	h.log.Info("Decoupled vibe coding session started", map[string]interface{}{
		"session_id": session.ID,
	})

	// パフォーマンス監視を開始
	if err := h.perfMonitor.Start(); err != nil {
		h.log.Warn("Performance monitoring failed to start", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// インタラクティブループを開始
	return h.runDecoupledInteractiveLoop(session.ID, cfg)
}

// runDecoupledInteractiveLoop は依存関係分離型の対話ループを実行
func (h *DecoupledChatHandler) runDecoupledInteractiveLoop(sessionID string, cfg *config.Config) error {
	scanner := bufio.NewScanner(os.Stdin)

	// ウェルカムメッセージ
	h.showDecoupledWelcomeMessage()

	for {
		fmt.Print("💬 You: ")
		
		if !scanner.Scan() {
			if scanner.Err() != nil {
				return fmt.Errorf("input error: %w", scanner.Err())
			}
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Printf("\n👋 Goodbye!\n")
			break
		}

		// 展開コマンド処理
		if input == "show" || input == "more" || input == "full" {
			h.handleExpandCommand()
			continue
		}

		// ユーザー入力を表示
		fmt.Printf("\n🎯 Processing: %s\n", input)

		// パフォーマンス測定開始
		startTime := time.Now()
		h.perfMonitor.RecordProactiveUsage("chat_request")

		// 実際の処理（インターフェース経由）
		response, err := h.processUserInputDecoupled(context.Background(), sessionID, input)
		
		// パフォーマンス測定
		duration := time.Since(startTime)
		h.perfMonitor.RecordResponseTime(duration)

		if err != nil {
			fmt.Printf("❌ Error: %s\n", err.Error())
			continue
		}

		// 応答を履歴に追加
		h.responseHistory = append(h.responseHistory, response.Message)
		if len(h.responseHistory) > 10 {
			h.responseHistory = h.responseHistory[1:]
		}

		// 応答表示
		h.displayDecoupledResponse(response, duration)

		fmt.Println()
	}

	return scanner.Err()
}

// processUserInputDecoupled は依存関係分離型の入力処理
func (h *DecoupledChatHandler) processUserInputDecoupled(ctx context.Context, sessionID, input string) (ResponseInfo, error) {
	// インタラクティブマネージャー経由で処理
	response, err := h.interactiveManager.ProcessUserInput(ctx, sessionID, input)
	if err != nil {
		// フォールバック: 直接LLMを使用
		h.log.Warn("Interactive manager failed, using LLM fallback", map[string]interface{}{
			"error": err.Error(),
		})

		llmResponse, llmErr := h.llmProvider.Generate(ctx, input, map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  1000,
		})
		if llmErr != nil {
			return ResponseInfo{}, fmt.Errorf("both interactive manager and LLM failed: %v, %v", err, llmErr)
		}

		return ResponseInfo{
			Message: llmResponse,
			Type:    "llm_fallback",
		}, nil
	}

	return response, nil
}

// showDecoupledWelcomeMessage はウェルカムメッセージを表示
func (h *DecoupledChatHandler) showDecoupledWelcomeMessage() {
	fmt.Print("\033[2J\033[H") // 画面クリア
	
	fmt.Println("🤖 vyb-code · Decoupled AI Coding Assistant")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()
	fmt.Println("🎯 Welcome to dependency-decoupled coding!")
	fmt.Println("💡 Low-coupling, high-cohesion architecture")
	fmt.Println()
	fmt.Println("🔧 Commands: 'help' for help")
	fmt.Println("🚪 Exit: 'exit' or 'quit'")
	
	// プロジェクト情報
	workDir, _ := os.Getwd()
	fmt.Printf("📂 Project: %s\n", filepath.Base(workDir))
	fmt.Printf("🏗️  Architecture: Decoupled\n")
	
	fmt.Println()
}

// displayDecoupledResponse は応答を表示
func (h *DecoupledChatHandler) displayDecoupledResponse(response ResponseInfo, duration time.Duration) {
	fmt.Printf("🤖 Assistant (%s)\n", response.Type)
	
	// ストリーミング表示を試行
	if h.streamingManager.IsEnabled() && len(response.Message) > 50 {
		err := h.streamingManager.ProcessString(context.Background(), response.Message, os.Stdout, nil)
		if err != nil {
			// フォールバック: 直接表示
			fmt.Print(response.Message)
		}
	} else {
		fmt.Print(response.Message)
	}

	// メタデータ表示
	fmt.Printf("\n\n💬 (%s · %s)",
		h.formatDuration(duration),
		response.Type)
}

// handleExpandCommand は展開コマンドを処理
func (h *DecoupledChatHandler) handleExpandCommand() {
	if len(h.responseHistory) > 0 {
		latest := h.responseHistory[len(h.responseHistory)-1]
		fmt.Printf("\n🔍 Full Content:\n%s\n\n", latest)
	} else {
		fmt.Printf("\n❌ No previous response to expand.\n\n")
	}
}

// formatDuration は時間をフォーマット
func (h *DecoupledChatHandler) formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d)/float64(time.Millisecond))
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}