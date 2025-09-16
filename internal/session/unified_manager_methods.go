package session

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AddMessage - メッセージを追加
func (m *unifiedSessionManager) AddMessage(sessionID string, message *Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	// メッセージIDを生成（未設定の場合）
	if message.ID == "" {
		message.ID = fmt.Sprintf("msg-%d-%d", time.Now().UnixNano(), len(session.Messages))
	}

	// タイムスタンプを設定（未設定の場合）
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	// メッセージを追加
	session.Messages = append(session.Messages, *message)
	session.UpdatedAt = time.Now()
	session.LastAccessedAt = time.Now()

	// 統計を更新
	m.updateMessageStats(session, message)

	// メッセージ数制限チェック
	if len(session.Messages) > session.Config.MaxMessages {
		// 古いメッセージをアーカイブ
		archiveCount := len(session.Messages) - session.Config.MaxMessages
		session.History.ArchivedMessages += archiveCount
		session.Messages = session.Messages[archiveCount:]
		session.History.LastArchiveTime = time.Now()
	}

	// イベント発行
	m.emitEvent(SessionEvent{
		Type:      EventMessageAdded,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data:      message,
	})

	// 自動保存
	if m.config.AutoSave && session.Config.PersistToDisk {
		go m.SaveSession(sessionID)
	}

	return nil
}

// GetMessages - メッセージを取得
func (m *unifiedSessionManager) GetMessages(sessionID string, limit int, offset int) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	messages := session.Messages

	// オフセット適用
	if offset > 0 {
		if offset >= len(messages) {
			return []Message{}, nil
		}
		messages = messages[offset:]
	}

	// 制限適用
	if limit > 0 && limit < len(messages) {
		messages = messages[:limit]
	}

	return messages, nil
}

// UpdateMessage - メッセージを更新
func (m *unifiedSessionManager) UpdateMessage(sessionID string, message *Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	// メッセージを検索して更新
	for i, msg := range session.Messages {
		if msg.ID == message.ID {
			session.Messages[i] = *message
			session.UpdatedAt = time.Now()

			// イベント発行
			m.emitEvent(SessionEvent{
				Type:      EventMessageUpdated,
				SessionID: sessionID,
				Timestamp: time.Now(),
				Data:      message,
			})

			return nil
		}
	}

	return fmt.Errorf("メッセージ '%s' が見つかりません", message.ID)
}

// DeleteMessage - メッセージを削除
func (m *unifiedSessionManager) DeleteMessage(sessionID string, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	// メッセージを検索して削除
	for i, msg := range session.Messages {
		if msg.ID == messageID {
			// スライスから削除
			session.Messages = append(session.Messages[:i], session.Messages[i+1:]...)
			session.UpdatedAt = time.Now()

			// 統計を更新
			m.updateStatsAfterMessageDeletion(session, &msg)

			return nil
		}
	}

	return fmt.Errorf("メッセージ '%s' が見つかりません", messageID)
}

// StartSession - セッションを開始
func (m *unifiedSessionManager) StartSession(sessionID string) error {
	return m.updateSessionState(sessionID, SessionStateActive, EventSessionStarted)
}

// PauseSession - セッションを一時停止
func (m *unifiedSessionManager) PauseSession(sessionID string) error {
	return m.updateSessionState(sessionID, SessionStatePaused, EventSessionPaused)
}

// ResumeSession - セッションを再開
func (m *unifiedSessionManager) ResumeSession(sessionID string) error {
	return m.updateSessionState(sessionID, SessionStateActive, EventSessionResumed)
}

// CompleteSession - セッションを完了
func (m *unifiedSessionManager) CompleteSession(sessionID string) error {
	return m.updateSessionState(sessionID, SessionStateCompleted, EventSessionCompleted)
}

// ArchiveSession - セッションをアーカイブ
func (m *unifiedSessionManager) ArchiveSession(sessionID string) error {
	return m.updateSessionState(sessionID, SessionStateArchived, EventSessionArchived)
}

// UpdateContext - コンテキストを更新
func (m *unifiedSessionManager) UpdateContext(sessionID string, context *ContextState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	session.Context = context
	session.UpdatedAt = time.Now()

	// イベント発行
	m.emitEvent(SessionEvent{
		Type:      EventContextUpdated,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data:      context,
	})

	return nil
}

// CompressContext - コンテキストを圧縮
func (m *unifiedSessionManager) CompressContext(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	if session.Context == nil || !session.Config.ContextCompression {
		return fmt.Errorf("コンテキスト圧縮が無効です")
	}

	// コンテキストマネージャーを使用して圧縮
	if m.contextManager != nil {
		// 簡易圧縮実装（実際のcompressContextManagerのシグネチャに合わせて調整が必要）
		originalSize := len(session.Context.CurrentContext)
		// ここでは簡略化した圧縮処理
		compressed := session.Context.CurrentContext // 実際の実装では圧縮処理を行う
		compressedSize := len(compressed)
		ratio := float64(compressedSize) / float64(originalSize)

		session.Context.CompressedContext = compressed
		session.Context.CompressionRatio = ratio
		session.UpdatedAt = time.Now()
	}

	return nil
}

// RestoreContext - コンテキストを復元
func (m *unifiedSessionManager) RestoreContext(sessionID string) (*ContextState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	if session.Context == nil {
		return nil, fmt.Errorf("コンテキストが存在しません")
	}

	// コンテキストを返す（必要に応じて展開）
	return session.Context, nil
}

// SaveSession - セッションを保存
func (m *unifiedSessionManager) SaveSession(sessionID string) error {
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	if !session.Config.PersistToDisk {
		return nil // 永続化無効の場合は何もしない
	}

	// セッションをJSONに変換
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("セッションシリアライゼーションエラー: %w", err)
	}

	// ファイルに保存
	sessionFile := filepath.Join(m.storageDir, sessionID+".json")
	if err := ioutil.WriteFile(sessionFile, data, 0644); err != nil {
		return fmt.Errorf("セッション保存エラー: %w", err)
	}

	return nil
}

// LoadSession - セッションを読み込み
func (m *unifiedSessionManager) LoadSession(sessionID string) (*UnifiedSession, error) {
	return m.loadSessionFromDisk(sessionID)
}

// ExportSession - セッションをエクスポート
func (m *unifiedSessionManager) ExportSession(sessionID string, format string) ([]byte, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(format) {
	case "json":
		return json.MarshalIndent(session, "", "  ")
	case "text":
		return m.exportAsText(session), nil
	default:
		return nil, fmt.Errorf("未対応のフォーマット: %s", format)
	}
}

// ImportSession - セッションをインポート
func (m *unifiedSessionManager) ImportSession(data []byte, format string) (*UnifiedSession, error) {
	switch strings.ToLower(format) {
	case "json":
		var session UnifiedSession
		if err := json.Unmarshal(data, &session); err != nil {
			return nil, fmt.Errorf("JSONデシリアライゼーションエラー: %w", err)
		}

		// 内部参照を復元
		session.manager = m
		// session.streamManager = m.streamManager // 使用停止
		session.contextManager = m.contextManager
		session.llmProvider = m.llmProvider
		session.eventHandlers = make(map[SessionEventType]SessionEventHandler)

		// セッションを登録
		m.mu.Lock()
		m.sessions[session.ID] = &session
		m.mu.Unlock()

		return &session, nil

	default:
		return nil, fmt.Errorf("未対応のフォーマット: %s", format)
	}
}

// RegisterEventHandler - イベントハンドラーを登録
func (m *unifiedSessionManager) RegisterEventHandler(eventType SessionEventType, handler SessionEventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.eventHandlers[eventType] = append(m.eventHandlers[eventType], handler)
}

// UnregisterEventHandler - イベントハンドラーを削除
func (m *unifiedSessionManager) UnregisterEventHandler(eventType SessionEventType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.eventHandlers, eventType)
}

// EmitEvent - イベントを発行
func (m *unifiedSessionManager) EmitEvent(event SessionEvent) {
	m.emitEvent(event)
}

// GetSessionStats - セッション統計を取得
func (m *unifiedSessionManager) GetSessionStats(sessionID string) (*UnifiedSessionStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	if session.Stats == nil {
		return &UnifiedSessionStats{}, nil
	}

	// 統計のコピーを返す
	statsCopy := *session.Stats
	return &statsCopy, nil
}

// GetGlobalStats - グローバル統計を取得
func (m *unifiedSessionManager) GetGlobalStats() (*GlobalSessionStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 最新統計に更新
	m.updateGlobalStats()

	// コピーを返す
	statsCopy := *m.globalStats
	return &statsCopy, nil
}

// CleanupExpiredSessions - 期限切れセッションをクリーンアップ
func (m *unifiedSessionManager) CleanupExpiredSessions() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var expiredSessions []string

	for sessionID, session := range m.sessions {
		if session.ExpiresAt != nil && now.After(*session.ExpiresAt) {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}

	// 期限切れセッションを削除
	for _, sessionID := range expiredSessions {
		delete(m.sessions, sessionID)

		// ディスクからも削除
		sessionFile := filepath.Join(m.storageDir, sessionID+".json")
		os.Remove(sessionFile)
	}

	if len(expiredSessions) > 0 {
		m.updateGlobalStats()
	}

	return len(expiredSessions), nil
}

// ArchiveOldSessions - 古いセッションをアーカイブ
func (m *unifiedSessionManager) ArchiveOldSessions(olderThan time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoffTime := time.Now().Add(-olderThan)
	var archivedCount int

	for _, session := range m.sessions {
		if session.LastAccessedAt.Before(cutoffTime) && session.State != SessionStateArchived {
			session.State = SessionStateArchived
			session.UpdatedAt = time.Now()
			archivedCount++
		}
	}

	if archivedCount > 0 {
		m.updateGlobalStats()
	}

	return archivedCount, nil
}

// CompactSessionStorage - セッションストレージを最適化
func (m *unifiedSessionManager) CompactSessionStorage() error {
	// アーカイブされたセッションを圧縮保存する実装
	// 実装は省略（必要に応じて後で追加）
	return nil
}

// UpdateConfig - 設定を更新
func (m *unifiedSessionManager) UpdateConfig(config *ManagerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = config
	return nil
}

// GetConfig - 設定を取得
func (m *unifiedSessionManager) GetConfig() *ManagerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	configCopy := *m.config
	return &configCopy
}

// FindSessions - セッションを検索
func (m *unifiedSessionManager) FindSessions(query string) ([]*UnifiedSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*UnifiedSession
	query = strings.ToLower(query)

	for _, session := range m.sessions {
		// ID検索
		if strings.Contains(strings.ToLower(session.ID), query) {
			results = append(results, session)
			continue
		}

		// タグ検索
		for _, tag := range session.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, session)
				break
			}
		}

		// メッセージ内容検索
		for _, message := range session.Messages {
			if strings.Contains(strings.ToLower(message.Content), query) {
				results = append(results, session)
				break
			}
		}
	}

	return results, nil
}

// GetActiveSessions - アクティブセッション一覧を取得
func (m *unifiedSessionManager) GetActiveSessions() ([]*UnifiedSession, error) {
	filter := &SessionFilter{
		States: []UnifiedSessionState{SessionStateActive},
	}

	return m.ListSessions(filter, SortByLastAccessed, SortOrderDesc)
}

// Shutdown - シャットダウン処理
func (m *unifiedSessionManager) Shutdown() error {
	// コンテキストをキャンセル
	m.cancel()

	// バックグラウンドタスクの完了を待機
	m.wg.Wait()

	// 最終保存
	m.saveAllSessions()

	return nil
}

// ヘルパーメソッド

// updateSessionState - セッション状態を更新
func (m *unifiedSessionManager) updateSessionState(sessionID string, newState UnifiedSessionState, eventType SessionEventType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	oldState := session.State
	session.State = newState
	session.UpdatedAt = time.Now()

	// 統計更新
	m.updateGlobalStats()

	// イベント発行
	m.emitEvent(SessionEvent{
		Type:      eventType,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"old_state": oldState,
			"new_state": newState,
		},
	})

	return nil
}

// updateMessageStats - メッセージ統計を更新
func (m *unifiedSessionManager) updateMessageStats(session *UnifiedSession, message *Message) {
	if session.Stats == nil {
		session.Stats = &UnifiedSessionStats{
			LastActivityTime: time.Now(),
		}
	}

	session.Stats.MessageCount++
	session.Stats.LastActivityTime = time.Now()

	switch message.Role {
	case MessageRoleUser:
		session.Stats.UserMessages++
	case MessageRoleAssistant:
		session.Stats.AssistantMessages++
	case MessageRoleSystem:
		session.Stats.SystemMessages++
	case MessageRoleTool:
		session.Stats.ToolMessages++
	}

	if message.TokenCount > 0 {
		session.Stats.TotalTokens += int64(message.TokenCount)
	}

	if len(message.ToolCalls) > 0 {
		session.Stats.ToolCallCount += len(message.ToolCalls)
		for _, toolCall := range message.ToolCalls {
			if toolCall.Error == "" {
				session.Stats.SuccessfulCalls++
			} else {
				session.Stats.FailedCalls++
			}
		}
	}
}

// updateStatsAfterMessageDeletion - メッセージ削除後の統計更新
func (m *unifiedSessionManager) updateStatsAfterMessageDeletion(session *UnifiedSession, message *Message) {
	if session.Stats == nil {
		return
	}

	session.Stats.MessageCount--

	switch message.Role {
	case MessageRoleUser:
		session.Stats.UserMessages--
	case MessageRoleAssistant:
		session.Stats.AssistantMessages--
	case MessageRoleSystem:
		session.Stats.SystemMessages--
	case MessageRoleTool:
		session.Stats.ToolMessages--
	}

	if message.TokenCount > 0 {
		session.Stats.TotalTokens -= int64(message.TokenCount)
	}
}

// exportAsText - セッションをテキスト形式でエクスポート
func (m *unifiedSessionManager) exportAsText(session *UnifiedSession) []byte {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("Session: %s\n", session.ID))
	result.WriteString(fmt.Sprintf("Type: %s\n", session.Type))
	result.WriteString(fmt.Sprintf("Created: %s\n", session.CreatedAt.Format(time.RFC3339)))
	result.WriteString(fmt.Sprintf("Messages: %d\n\n", len(session.Messages)))

	for _, message := range session.Messages {
		result.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			message.Timestamp.Format("2006-01-02 15:04:05"),
			message.Role,
			message.Content))
		result.WriteString("\n")
	}

	return []byte(result.String())
}
