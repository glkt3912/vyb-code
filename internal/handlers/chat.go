package handlers

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/glkt/vyb-code/internal/chat"
	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/llm"
	"github.com/glkt/vyb-code/internal/logger"
	"github.com/glkt/vyb-code/internal/ui"
)

// ChatHandler はチャット機能のハンドラー
type ChatHandler struct {
	log logger.Logger
}

// NewChatHandler はチャットハンドラーの新しいインスタンスを作成
func NewChatHandler(log logger.Logger) *ChatHandler {
	return &ChatHandler{log: log}
}

// StartInteractiveModeWithOptions は拡張オプションで対話モードを開始
func (h *ChatHandler) StartInteractiveModeWithOptions(noTUI bool, terminalMode bool, planMode bool, continueSession bool, resumeID string) error {
	// 既存の関数を呼び出し（将来的にセッション復元機能を追加）
	return h.StartInteractiveMode(noTUI, terminalMode, planMode)
	// TODO: continueSession と resumeID の処理を実装
}

// ProcessSingleQueryWithOptions は拡張オプションで単発クエリを処理
func (h *ChatHandler) ProcessSingleQueryWithOptions(query string, noTUI bool, continueSession bool, resumeID string) error {
	// 既存の関数を呼び出し（将来的にセッション復元機能を追加）
	return h.ProcessSingleQuery(query, noTUI)
	// TODO: continueSession と resumeID の処理を実装
}

// StartInteractiveMode は対話モードを開始
func (h *ChatHandler) StartInteractiveMode(noTUI bool, terminalMode bool, planMode bool) error {
	// 設定を読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// モードの判定
	if terminalMode {
		// Claude Code風ターミナルモード
		return h.startEnhancedTerminalMode(cfg, planMode)
	} else {
		// TUIモードの判定
		useTUI := cfg.TUI.Enabled && !noTUI

		if useTUI {
			// TUIモードで開始
			app := ui.NewSimpleApp(cfg.TUI)
			program := tea.NewProgram(app, tea.WithAltScreen())

			if _, err := program.Run(); err != nil {
				h.log.Error("TUIエラー", map[string]interface{}{"error": err})
				// フォールバックで通常モード
				return h.startLegacyInteractiveMode(cfg)
			}
		} else {
			// 通常の対話モード
			return h.startLegacyInteractiveMode(cfg)
		}
	}
	return nil
}

// startEnhancedTerminalMode はClaude Code風ターミナルモードを開始
func (h *ChatHandler) startEnhancedTerminalMode(cfg *config.Config, planMode bool) error {
	h.log.Info("Enhanced Terminal Mode開始", map[string]interface{}{
		"plan_mode": planMode,
	})

	// プロンプト表示とメイン処理
	return h.runEnhancedTerminalLoop(cfg, planMode)
}

// startLegacyInteractiveMode は従来の対話モードを開始
func (h *ChatHandler) startLegacyInteractiveMode(cfg *config.Config) error {
	h.log.Info("Legacy Interactive Mode開始", nil)

	// TODO: LLMプロバイダー実装
	h.log.Info("LLMプロバイダー初期化（開発中）", map[string]interface{}{
		"base_url": cfg.BaseURL,
		"model":    cfg.ModelName,
	})

	// とりあえず開発中メッセージで終了
	return fmt.Errorf("Legacy Interactive Mode機能は開発中です")
}

// ProcessSingleQuery は単発クエリを処理
func (h *ChatHandler) ProcessSingleQuery(query string, noTUI bool) error {
	// 設定を読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// TODO: 単発クエリ処理実装
	h.log.Info("単発クエリ処理（開発中）", map[string]interface{}{
		"query":  query,
		"no_tui": noTUI,
		"model":  cfg.ModelName,
	})

	return fmt.Errorf("単発クエリ機能は開発中です")
}

// StartVibeCodingMode はバイブコーディングモードを開始
func (h *ChatHandler) StartVibeCodingMode() error {
	h.log.Info("Vibe Coding Mode開始", nil)

	// 設定を読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// LLMプロバイダーを初期化
	provider := llm.NewOllamaClient(cfg.BaseURL)

	// バイブモード専用セッションを作成
	session := chat.NewVibeSession(provider, cfg.ModelName, cfg)

	// バイブ起動メッセージを表示
	h.printVibeWelcomeMessage(session)

	// Enhanced Terminal Modeを開始
	return session.StartEnhancedTerminal()
}

// printVibeWelcomeMessage はバイブモード専用の起動メッセージを表示
func (h *ChatHandler) printVibeWelcomeMessage(session *chat.Session) {
	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	magenta := "\033[35m"
	reset := "\033[0m"
	bold := "\033[1m"

	fmt.Printf("\n")
	fmt.Printf("%s%s🎵 vyb - Feel the rhythm of perfect code%s%s\n", cyan, bold, reset, reset)
	fmt.Printf("%s╭─ Mode: %sVibe Coding %s(AI-powered interactive experience)%s\n",
		green, magenta, green, reset)
	fmt.Printf("%s├─ Context: %sCompression enabled %s(70-95%% efficiency)%s\n",
		green, yellow, green, reset)
	fmt.Printf("%s├─ Features: %sCode suggestions, real-time analysis, smart completion%s\n",
		green, cyan, reset)
	fmt.Printf("%s╰─ Type 'exit' to quit, '/help' for commands%s\n", green, reset)
	fmt.Printf("\n")
}

// runEnhancedTerminalLoop はEnhanced Terminal Modeのメインループ
func (h *ChatHandler) runEnhancedTerminalLoop(cfg *config.Config, planMode bool) error {
	h.log.Info("Enhanced Terminal Mode開始", map[string]interface{}{
		"plan_mode": planMode,
		"model":     cfg.ModelName,
	})

	// LLMプロバイダーを初期化
	provider := llm.NewOllamaClient(cfg.BaseURL)

	// セッションを作成（設定付き）
	session := chat.NewSessionWithConfig(provider, cfg.ModelName, cfg)

	// 実際のEnhanced Terminal Modeを開始
	return session.StartEnhancedTerminal()
}
