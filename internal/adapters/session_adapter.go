package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
	"github.com/glkt/vyb-code/internal/session"
)

// SessionAdapter - セッション管理アダプター
type SessionAdapter struct {
	*BaseAdapter

	// レガシーシステム
	legacyManager *session.Manager

	// 統合システム
	unifiedManager session.UnifiedSessionManager
}

// NewSessionAdapter - 新しいセッションアダプターを作成
func NewSessionAdapter(log logger.Logger) *SessionAdapter {
	return &SessionAdapter{
		BaseAdapter: NewBaseAdapter(AdapterTypeSession, log),
	}
}

// Configure - セッションアダプターの設定
func (sa *SessionAdapter) Configure(config *config.GradualMigrationConfig) error {
	if err := sa.BaseAdapter.Configure(config); err != nil {
		return err
	}

	// レガシーシステムの初期化
	if sa.legacyManager == nil {
		sa.legacyManager = session.NewManager()
		if err := sa.legacyManager.Initialize(); err != nil {
			sa.log.Error("レガシーセッションマネージャー初期化失敗", map[string]interface{}{"error": err})
		}
	}

	// 統合システムの初期化（暫定的に無効化）
	// if sa.unifiedManager == nil && sa.IsUnifiedEnabled() {
	//     // 統合セッション管理システムは後で実装
	// }

	sa.log.Info("セッションアダプター設定完了", map[string]interface{}{
		"unified_enabled": sa.IsUnifiedEnabled(),
		"legacy_ready":    sa.legacyManager != nil,
	})

	return nil
}

// CreateSession - セッション作成（統一インターフェース）
func (sa *SessionAdapter) CreateSession(ctx context.Context, sessionType string) (string, error) {
	startTime := time.Now()

	useUnified := !sa.ShouldUseLegacy()

	var sessionID string
	var err error

	if useUnified && sa.unifiedManager != nil {
		sessionID, err = sa.createSessionWithUnified(ctx, sessionType)
	} else {
		sessionID, err = sa.createSessionWithLegacy(ctx)
	}

	// フォールバック処理
	if err != nil && useUnified && sa.config.EnableFallback {
		sa.IncrementFallback()
		sa.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"error": err.Error(),
		})
		sessionID, err = sa.createSessionWithLegacy(ctx)
		useUnified = false
	}

	latency := time.Since(startTime)
	sa.UpdateMetrics(err == nil, useUnified, latency)
	sa.LogOperation("CreateSession", useUnified, latency, err)

	return sessionID, err
}

// GetSession - セッション取得（統一インターフェース）
func (sa *SessionAdapter) GetSession(ctx context.Context, sessionID string) (interface{}, error) {
	startTime := time.Now()

	useUnified := !sa.ShouldUseLegacy()

	var sessionData interface{}
	var err error

	if useUnified && sa.unifiedManager != nil {
		sessionData, err = sa.getSessionWithUnified(ctx, sessionID)
	} else {
		sessionData, err = sa.getSessionWithLegacy(ctx, sessionID)
	}

	// フォールバック処理
	if err != nil && useUnified && sa.config.EnableFallback {
		sa.IncrementFallback()
		sa.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"error": err.Error(),
		})
		sessionData, err = sa.getSessionWithLegacy(ctx, sessionID)
		useUnified = false
	}

	latency := time.Since(startTime)
	sa.UpdateMetrics(err == nil, useUnified, latency)
	sa.LogOperation("GetSession", useUnified, latency, err)

	return sessionData, err
}

// UpdateSession - セッション更新（統一インターフェース）
func (sa *SessionAdapter) UpdateSession(ctx context.Context, sessionID string, data interface{}) error {
	startTime := time.Now()

	useUnified := !sa.ShouldUseLegacy()

	var err error

	if useUnified && sa.unifiedManager != nil {
		err = sa.updateSessionWithUnified(ctx, sessionID, data)
	} else {
		err = sa.updateSessionWithLegacy(ctx, sessionID, data)
	}

	// フォールバック処理
	if err != nil && useUnified && sa.config.EnableFallback {
		sa.IncrementFallback()
		sa.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"error": err.Error(),
		})
		err = sa.updateSessionWithLegacy(ctx, sessionID, data)
		useUnified = false
	}

	latency := time.Since(startTime)
	sa.UpdateMetrics(err == nil, useUnified, latency)
	sa.LogOperation("UpdateSession", useUnified, latency, err)

	return err
}

// DeleteSession - セッション削除（統一インターフェース）
func (sa *SessionAdapter) DeleteSession(ctx context.Context, sessionID string) error {
	startTime := time.Now()

	useUnified := !sa.ShouldUseLegacy()

	var err error

	if useUnified && sa.unifiedManager != nil {
		err = sa.deleteSessionWithUnified(ctx, sessionID)
	} else {
		err = sa.deleteSessionWithLegacy(ctx, sessionID)
	}

	// フォールバック処理
	if err != nil && useUnified && sa.config.EnableFallback {
		sa.IncrementFallback()
		sa.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"error": err.Error(),
		})
		err = sa.deleteSessionWithLegacy(ctx, sessionID)
		useUnified = false
	}

	latency := time.Since(startTime)
	sa.UpdateMetrics(err == nil, useUnified, latency)
	sa.LogOperation("DeleteSession", useUnified, latency, err)

	return err
}

// ListSessions - セッション一覧取得（統一インターフェース）
func (sa *SessionAdapter) ListSessions(ctx context.Context) ([]string, error) {
	startTime := time.Now()

	useUnified := !sa.ShouldUseLegacy()

	var sessions []string
	var err error

	if useUnified && sa.unifiedManager != nil {
		sessions, err = sa.listSessionsWithUnified(ctx)
	} else {
		sessions, err = sa.listSessionsWithLegacy(ctx)
	}

	// フォールバック処理
	if err != nil && useUnified && sa.config.EnableFallback {
		sa.IncrementFallback()
		sa.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"error": err.Error(),
		})
		sessions, err = sa.listSessionsWithLegacy(ctx)
		useUnified = false
	}

	latency := time.Since(startTime)
	sa.UpdateMetrics(err == nil, useUnified, latency)
	sa.LogOperation("ListSessions", useUnified, latency, err)

	return sessions, err
}

// HealthCheck - セッションアダプターのヘルスチェック
func (sa *SessionAdapter) HealthCheck(ctx context.Context) error {
	if err := sa.BaseAdapter.HealthCheck(ctx); err != nil {
		return err
	}

	// レガシーシステムのヘルスチェック
	if sa.legacyManager == nil {
		return fmt.Errorf("legacy session manager not initialized")
	}

	// 統合システムのヘルスチェック（有効な場合）
	if sa.IsUnifiedEnabled() {
		if sa.unifiedManager == nil {
			return fmt.Errorf("unified session manager not initialized")
		}

		// 統合システムの基本機能チェック（HealthCheckメソッドが存在しない場合は基本チェックのみ）
		// TODO: 実際のヘルスチェックメソッドが実装されたら有効化
		// if err := sa.unifiedManager.HealthCheck(ctx); err != nil {
		//     return fmt.Errorf("unified session health check failed: %w", err)
		// }
	}

	return nil
}

// Internal methods

// createSessionWithUnified - 統合システムでのセッション作成
func (sa *SessionAdapter) createSessionWithUnified(ctx context.Context, sessionType string) (string, error) {
	if sa.unifiedManager == nil {
		return "", fmt.Errorf("unified session manager not initialized")
	}

	// 統合セッション作成
	sessionConfig := &session.SessionConfig{
		AutoSave:      true,
		PersistToDisk: true,
		MaxMessages:   1000,
		MaxTokens:     4096,
	}

	// セッション型を解析
	var unifiedSessionType session.UnifiedSessionType
	switch sessionType {
	case "vibe":
		unifiedSessionType = session.SessionTypeVibeCoding
	case "interactive":
		unifiedSessionType = session.SessionTypeInteractive
	default:
		unifiedSessionType = session.SessionTypeChat
	}

	unifiedSession, err := sa.unifiedManager.CreateSession(unifiedSessionType, sessionConfig)
	if err != nil {
		return "", err
	}

	return unifiedSession.ID, nil
}

// createSessionWithLegacy - レガシーシステムでのセッション作成
func (sa *SessionAdapter) createSessionWithLegacy(ctx context.Context) (string, error) {
	if sa.legacyManager == nil {
		return "", fmt.Errorf("legacy session manager not initialized")
	}

	session, err := sa.legacyManager.CreateSession("vibe", "Default Session", "interactive")
	if err != nil {
		return "", err
	}

	return session.ID, nil
}

// getSessionWithUnified - 統合システムでのセッション取得
func (sa *SessionAdapter) getSessionWithUnified(ctx context.Context, sessionID string) (interface{}, error) {
	if sa.unifiedManager == nil {
		return nil, fmt.Errorf("unified session manager not initialized")
	}

	return sa.unifiedManager.GetSession(sessionID)
}

// getSessionWithLegacy - レガシーシステムでのセッション取得
func (sa *SessionAdapter) getSessionWithLegacy(ctx context.Context, sessionID string) (interface{}, error) {
	if sa.legacyManager == nil {
		return nil, fmt.Errorf("legacy session manager not initialized")
	}

	return sa.legacyManager.GetSession(sessionID)
}

// updateSessionWithUnified - 統合システムでのセッション更新
func (sa *SessionAdapter) updateSessionWithUnified(ctx context.Context, sessionID string, data interface{}) error {
	if sa.unifiedManager == nil {
		return fmt.Errorf("unified session manager not initialized")
	}

	// dataを*UnifiedSessionに変換（暫定的にスキップ）
	// TODO: 実際のデータ変換ロジックを実装
	return fmt.Errorf("session update not implemented for unified system")
}

// updateSessionWithLegacy - レガシーシステムでのセッション更新
func (sa *SessionAdapter) updateSessionWithLegacy(ctx context.Context, sessionID string, data interface{}) error {
	if sa.legacyManager == nil {
		return fmt.Errorf("legacy session manager not initialized")
	}

	// レガシーセッションマネージャーには UpdateSession メソッドがないため、暫定的にスキップ
	// TODO: 必要に応じて適切な更新ロジックを実装
	return nil
}

// deleteSessionWithUnified - 統合システムでのセッション削除
func (sa *SessionAdapter) deleteSessionWithUnified(ctx context.Context, sessionID string) error {
	if sa.unifiedManager == nil {
		return fmt.Errorf("unified session manager not initialized")
	}

	return sa.unifiedManager.DeleteSession(sessionID)
}

// deleteSessionWithLegacy - レガシーシステムでのセッション削除
func (sa *SessionAdapter) deleteSessionWithLegacy(ctx context.Context, sessionID string) error {
	if sa.legacyManager == nil {
		return fmt.Errorf("legacy session manager not initialized")
	}

	return sa.legacyManager.DeleteSession(sessionID)
}

// listSessionsWithUnified - 統合システムでのセッション一覧取得
func (sa *SessionAdapter) listSessionsWithUnified(ctx context.Context) ([]string, error) {
	if sa.unifiedManager == nil {
		return nil, fmt.Errorf("unified session manager not initialized")
	}

	sessions, err := sa.unifiedManager.ListSessions(nil, session.SortByCreatedAt, session.SortOrderDesc)
	if err != nil {
		return nil, err
	}

	var sessionIDs []string
	for _, sess := range sessions {
		sessionIDs = append(sessionIDs, sess.ID)
	}

	return sessionIDs, nil
}

// listSessionsWithLegacy - レガシーシステムでのセッション一覧取得
func (sa *SessionAdapter) listSessionsWithLegacy(ctx context.Context) ([]string, error) {
	if sa.legacyManager == nil {
		return nil, fmt.Errorf("legacy session manager not initialized")
	}

	sessions := sa.legacyManager.ListSessions()
	sessionIDs := make([]string, len(sessions))
	for i, sess := range sessions {
		sessionIDs[i] = sess.ID
	}

	return sessionIDs, nil
}

// GetSessionMetrics - セッション固有のメトリクスを取得
func (sa *SessionAdapter) GetSessionMetrics() *SessionMetrics {
	baseMetrics := sa.GetMetrics()

	sessionMetrics := &SessionMetrics{
		AdapterMetrics: *baseMetrics,
		UnifiedManager: sa.unifiedManager != nil,
		LegacyManager:  sa.legacyManager != nil,
	}

	// 統合システムのメトリクス取得
	if sa.unifiedManager != nil {
		// TODO: 実際のメトリクス取得メソッドに合わせて実装
		// unifiedMetrics, _ := sa.unifiedManager.GetSessionStats("all")
		// sessionMetrics.UnifiedSessionMetrics = unifiedMetrics
	}

	// レガシーシステムのメトリクス取得
	if sa.legacyManager != nil {
		legacyMetrics := sa.legacyManager.GetStats()
		sessionMetrics.LegacySessionMetrics = legacyMetrics
	}

	return sessionMetrics
}

// SessionMetrics - セッションアダプター固有のメトリクス
type SessionMetrics struct {
	AdapterMetrics
	UnifiedManager        bool                         `json:"unified_manager"`
	LegacyManager         bool                         `json:"legacy_manager"`
	UnifiedSessionMetrics *session.UnifiedSessionStats `json:"unified_metrics,omitempty"`
	LegacySessionMetrics  interface{}                  `json:"legacy_metrics,omitempty"`
}
