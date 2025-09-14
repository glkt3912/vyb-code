package handlers

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/glkt/vyb-code/internal/adapters"
	"github.com/glkt/vyb-code/internal/chat"
	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/llm"
	"github.com/glkt/vyb-code/internal/logger"
	"github.com/glkt/vyb-code/internal/migration"
	"github.com/glkt/vyb-code/internal/session"
	"github.com/glkt/vyb-code/internal/ui"
)

// ChatHandler はチャット機能のハンドラー（段階的移行対応）
type ChatHandler struct {
	log            logger.Logger
	sessionManager *session.Manager

	// 段階的移行システム
	sessionAdapter   *adapters.SessionAdapter
	streamingAdapter *adapters.StreamingAdapter
	toolsAdapter     *adapters.ToolsAdapter
	analysisAdapter  *adapters.AnalysisAdapter

	// 移行監視・検証
	monitor   *migration.MigrationMonitor
	validator *migration.Validator
}

// NewChatHandler はチャットハンドラーの新しいインスタンスを作成
func NewChatHandler(log logger.Logger) *ChatHandler {
	return NewChatHandlerWithMigration(log, nil)
}

// NewChatHandlerWithMigration は段階的移行対応のチャットハンドラーを作成
func NewChatHandlerWithMigration(log logger.Logger, migrationConfig *config.GradualMigrationConfig) *ChatHandler {
	// レガシーセッションマネージャー初期化
	sessionMgr := session.NewManager()
	if err := sessionMgr.Initialize(); err != nil {
		log.Error("セッションマネージャー初期化失敗", map[string]interface{}{"error": err})
	}

	handler := &ChatHandler{
		log:            log,
		sessionManager: sessionMgr,
	}

	// 段階的移行システムの初期化
	if migrationConfig != nil && migrationConfig.MigrationMode != "legacy" {
		handler.initializeMigrationSystem(migrationConfig)
	}

	return handler
}

// initializeMigrationSystem は段階的移行システムを初期化（暫定版）
func (h *ChatHandler) initializeMigrationSystem(cfg *config.GradualMigrationConfig) {
	// 暫定的に基本ログのみでアダプター初期化を無効化
	h.log.Info("段階的移行システム初期化（暫定版）", map[string]interface{}{
		"migration_mode": cfg.MigrationMode,
		"validation":     cfg.EnableValidation,
		"monitoring":     cfg.EnableMetrics,
		"note":           "アダプター実装は段階的に有効化予定",
	})

	// アダプター初期化は後で実装
	// TODO: 各アダプターの初期化を段階的に有効化
}

// StartInteractiveModeWithOptions は拡張オプションで対話モードを開始
func (h *ChatHandler) StartInteractiveModeWithOptions(noTUI bool, terminalMode bool, planMode bool, continueSession bool, resumeID string) error {
	// セッション継続機能
	if continueSession || resumeID != "" {
		return h.handleSessionContinuation(resumeID, noTUI, terminalMode, planMode)
	}

	// 通常の対話モード開始
	return h.StartInteractiveMode(noTUI, terminalMode, planMode)
}

// ProcessSingleQueryWithOptions は拡張オプションで単発クエリを処理
func (h *ChatHandler) ProcessSingleQueryWithOptions(query string, noTUI bool, continueSession bool, resumeID string) error {
	// セッション継続での単発クエリ処理
	if continueSession || resumeID != "" {
		return h.handleSingleQueryWithSession(query, resumeID, noTUI)
	}

	// 通常の単発クエリ処理
	return h.ProcessSingleQuery(query, noTUI)
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

// handleSessionContinuation はセッション継続を処理
func (h *ChatHandler) handleSessionContinuation(resumeID string, noTUI bool, terminalMode bool, planMode bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	var sessionToResume *session.Session

	if resumeID != "" {
		// 特定のセッションIDで復元
		sessionToResume, err = h.getSessionWithAdapter(resumeID)
		if err != nil {
			return fmt.Errorf("セッション '%s' の取得に失敗: %w", resumeID, err)
		}
		h.log.Info("セッション復元", map[string]interface{}{"session_id": resumeID})
	} else {
		// 最新のセッションを継続
		sessionToResume, err = h.getCurrentSessionWithAdapter()
		if err != nil {
			// 現在のセッションがない場合は、最新のセッションを取得
			sessions, listErr := h.listSessionsWithAdapter()
			if listErr != nil || len(sessions) == 0 {
				return fmt.Errorf("継続可能なセッションがありません")
			}
			// 最初のセッションIDを使って取得
			sessionToResume, err = h.getSessionWithAdapter(sessions[0])
			if err != nil {
				return fmt.Errorf("セッション取得失敗: %w", err)
			}
		}
		h.log.Info("最新セッション継続", map[string]interface{}{"session_id": sessionToResume.ID})
	}

	// セッション継続メッセージを表示
	h.printSessionContinuationMessage(sessionToResume)

	// セッションを現在のアクティブセッションに設定
	if err := h.sessionManager.SwitchSession(sessionToResume.ID); err != nil {
		return fmt.Errorf("セッション切り替え失敗: %w", err)
	}

	// 継続したセッションで対話モードを開始
	return h.runContinuedSession(sessionToResume, cfg, noTUI, terminalMode, planMode)
}

// handleSingleQueryWithSession はセッション継続での単発クエリを処理
func (h *ChatHandler) handleSingleQueryWithSession(query, resumeID string, noTUI bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	var sessionToResume *session.Session

	if resumeID != "" {
		sessionToResume, err = h.sessionManager.GetSession(resumeID)
		if err != nil {
			return fmt.Errorf("セッション '%s' の取得に失敗: %w", resumeID, err)
		}
	} else {
		sessionToResume, err = h.sessionManager.GetCurrentSession()
		if err != nil {
			sessions := h.sessionManager.ListSessions()
			if len(sessions) == 0 {
				return fmt.Errorf("継続可能なセッションがありません")
			}
			sessionToResume = sessions[0]
		}
	}

	// セッションに新しいターンを追加
	metadata := session.Metadata{"query_type": "single"}
	if err := h.sessionManager.AddTurn(sessionToResume.ID, session.TurnTypeUser, query, metadata); err != nil {
		h.log.Error("ターン追加失敗", map[string]interface{}{"error": err})
	}

	// 単発クエリを実行（セッションコンテキスト付き）
	return h.processSingleQueryWithContext(query, sessionToResume, cfg, noTUI)
}

// runContinuedSession は継続されたセッションを実行
func (h *ChatHandler) runContinuedSession(sess *session.Session, cfg *config.Config, noTUI bool, terminalMode bool, planMode bool) error {
	// LLMプロバイダーを初期化
	provider := llm.NewOllamaClient(cfg.BaseURL)

	// セッションコンテキストを復元してチャットセッションを作成
	chatSession := chat.NewSessionWithConfig(provider, sess.Model, cfg)

	// セッション履歴をチャットセッションに復元
	if err := h.restoreSessionHistory(chatSession, sess); err != nil {
		h.log.Error("セッション履歴復元失敗", map[string]interface{}{"error": err})
	}

	// モードに応じて継続
	if terminalMode {
		return chatSession.StartEnhancedTerminal()
	} else {
		// TUIまたは通常モード
		if cfg.TUI.Enabled && !noTUI {
			app := ui.NewSimpleApp(cfg.TUI)
			program := tea.NewProgram(app, tea.WithAltScreen())
			if _, err := program.Run(); err != nil {
				h.log.Error("TUIエラー", map[string]interface{}{"error": err})
			}
		}
		// フォールバック: 通常の対話モード継続
		return chatSession.StartEnhancedTerminal()
	}
}

// processSingleQueryWithContext はコンテキスト付きで単発クエリを処理
func (h *ChatHandler) processSingleQueryWithContext(query string, sess *session.Session, cfg *config.Config, noTUI bool) error {
	// セッションコンテキストを含めてクエリを処理
	h.log.Info("セッションコンテキスト付き単発クエリ処理", map[string]interface{}{
		"query":      query,
		"session_id": sess.ID,
		"turn_count": sess.TurnCount,
		"provider":   cfg.Provider,
	})

	// TODO: 実際のLLM呼び出しとレスポンス処理
	// 現在は開発中のため基本情報を表示
	fmt.Printf("セッション '%s' (作成: %s) でクエリを処理中...\n",
		sess.Title, sess.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("クエリ: %s\n", query)
	fmt.Printf("過去のターン数: %d\n", sess.TurnCount)

	return nil
}

// restoreSessionHistory はセッション履歴をチャットセッションに復元
func (h *ChatHandler) restoreSessionHistory(chatSession *chat.Session, sess *session.Session) error {
	// セッションの過去のターンを復元
	for _, turn := range sess.Turns {
		// チャットセッションにターン情報を追加
		// TODO: chat.Sessionに適切な履歴復元メソッドを実装
		h.log.Debug("ターン復元", map[string]interface{}{
			"turn_id":   turn.ID,
			"turn_type": turn.Type,
			"timestamp": turn.Timestamp,
		})
	}

	return nil
}

// printSessionContinuationMessage はセッション継続メッセージを表示
func (h *ChatHandler) printSessionContinuationMessage(sess *session.Session) {
	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	reset := "\033[0m"
	bold := "\033[1m"

	fmt.Printf("\n")
	fmt.Printf("%s%s🔄 セッション継続%s%s\n", cyan, bold, reset, reset)
	fmt.Printf("%s├─ セッション: %s%s%s\n", green, yellow, sess.Title, reset)
	fmt.Printf("%s├─ 作成日時: %s%s%s\n", green, yellow, sess.CreatedAt.Format("2006-01-02 15:04:05"), reset)
	fmt.Printf("%s├─ ターン数: %s%d%s\n", green, yellow, sess.TurnCount, reset)
	fmt.Printf("%s├─ モデル: %s%s (%s)%s\n", green, yellow, sess.Model, sess.Provider, reset)
	fmt.Printf("%s╰─ 最終更新: %s%s%s\n", green, yellow, sess.UpdatedAt.Format("2006-01-02 15:04:05"), reset)
	fmt.Printf("\n")
}

// 段階的移行対応ヘルパーメソッド

// getSessionWithAdapter はアダプターを使用してセッションを取得
func (h *ChatHandler) getSessionWithAdapter(sessionID string) (*session.Session, error) {
	ctx := context.Background()

	if h.sessionAdapter != nil {
		// アダプター経由でセッション取得
		sessionData, err := h.sessionAdapter.GetSession(ctx, sessionID)
		if err != nil {
			h.log.Error("アダプター経由セッション取得失敗", map[string]interface{}{
				"session_id": sessionID,
				"error":      err,
			})
			// フォールバック: レガシー方式
			return h.sessionManager.GetSession(sessionID)
		}

		// sessionDataを*session.Sessionに変換
		if sess, ok := sessionData.(*session.Session); ok {
			return sess, nil
		}

		h.log.Warn("セッションデータ型変換失敗", map[string]interface{}{
			"session_id": sessionID,
			"type":       fmt.Sprintf("%T", sessionData),
		})
		// フォールバック
		return h.sessionManager.GetSession(sessionID)
	}

	// レガシー方式
	return h.sessionManager.GetSession(sessionID)
}

// getCurrentSessionWithAdapter はアダプターを使用して現在のセッションを取得
func (h *ChatHandler) getCurrentSessionWithAdapter() (*session.Session, error) {
	// レガシー方式（現在のセッション概念は主にレガシー側にある）
	return h.sessionManager.GetCurrentSession()
}

// listSessionsWithAdapter はアダプターを使用してセッション一覧を取得
func (h *ChatHandler) listSessionsWithAdapter() ([]string, error) {
	ctx := context.Background()

	if h.sessionAdapter != nil {
		// アダプター経由でセッション一覧取得
		sessions, err := h.sessionAdapter.ListSessions(ctx)
		if err != nil {
			h.log.Error("アダプター経由セッション一覧取得失敗", map[string]interface{}{
				"error": err,
			})
			// フォールバック: レガシー方式
			legacySessions := h.sessionManager.ListSessions()
			sessionIDs := make([]string, len(legacySessions))
			for i, sess := range legacySessions {
				sessionIDs[i] = sess.ID
			}
			return sessionIDs, nil
		}
		return sessions, nil
	}

	// レガシー方式
	legacySessions := h.sessionManager.ListSessions()
	sessionIDs := make([]string, len(legacySessions))
	for i, sess := range legacySessions {
		sessionIDs[i] = sess.ID
	}
	return sessionIDs, nil
}

// createSessionWithAdapter はアダプターを使用してセッションを作成
func (h *ChatHandler) createSessionWithAdapter(sessionType string) (string, error) {
	ctx := context.Background()

	if h.sessionAdapter != nil {
		// アダプター経由でセッション作成
		sessionID, err := h.sessionAdapter.CreateSession(ctx, sessionType)
		if err != nil {
			h.log.Error("アダプター経由セッション作成失敗", map[string]interface{}{
				"type":  sessionType,
				"error": err,
			})
			// フォールバック: レガシー方式
			return h.createLegacySession()
		}
		return sessionID, nil
	}

	// レガシー方式
	return h.createLegacySession()
}

// createLegacySession はレガシー方式でセッションを作成
func (h *ChatHandler) createLegacySession() (string, error) {
	sess, err := h.sessionManager.CreateSession("vibe", "Default Session", "interactive")
	if err != nil {
		return "", err
	}
	return sess.ID, nil
}

// updateSessionWithAdapter はアダプターを使用してセッションを更新
func (h *ChatHandler) updateSessionWithAdapter(sessionID string, data interface{}) error {
	ctx := context.Background()

	if h.sessionAdapter != nil {
		// アダプター経由でセッション更新
		err := h.sessionAdapter.UpdateSession(ctx, sessionID, data)
		if err != nil {
			h.log.Error("アダプター経由セッション更新失敗", map[string]interface{}{
				"session_id": sessionID,
				"error":      err,
			})
			// フォールバック処理は省略（更新は必須ではない）
		}
		return err
	}

	// レガシー方式では基本的な更新のみ
	return nil
}

// GetMigrationStatus は移行ステータスを取得
func (h *ChatHandler) GetMigrationStatus() map[string]interface{} {
	status := map[string]interface{}{
		"migration_enabled": h.sessionAdapter != nil,
		"adapters": map[string]interface{}{
			"session":   h.sessionAdapter != nil,
			"streaming": h.streamingAdapter != nil,
			"tools":     h.toolsAdapter != nil,
			"analysis":  h.analysisAdapter != nil,
		},
		"monitoring": h.monitor != nil,
		"validation": h.validator != nil,
	}

	if h.monitor != nil {
		status["monitoring_status"] = h.monitor.GetMonitoringStatus()
	}

	if h.validator != nil {
		status["validation_stats"] = h.validator.GetValidationStats()
	}

	return status
}
