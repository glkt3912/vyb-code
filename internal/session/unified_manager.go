package session

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/glkt/vyb-code/internal/contextmanager"
	"github.com/glkt/vyb-code/internal/llm"
	"github.com/glkt/vyb-code/internal/streaming"
)

// UnifiedSessionManager - 統合セッション管理インターフェース
type UnifiedSessionManager interface {
	// セッション基本操作
	CreateSession(sessionType UnifiedSessionType, config *SessionConfig) (*UnifiedSession, error)
	GetSession(sessionID string) (*UnifiedSession, error)
	UpdateSession(session *UnifiedSession) error
	DeleteSession(sessionID string) error

	// セッション検索・一覧
	ListSessions(filter *SessionFilter, sortBy SessionSortBy, sortOrder SessionSortOrder) ([]*UnifiedSession, error)
	FindSessions(query string) ([]*UnifiedSession, error)
	GetActiveSessions() ([]*UnifiedSession, error)

	// メッセージ操作
	AddMessage(sessionID string, message *Message) error
	GetMessages(sessionID string, limit int, offset int) ([]Message, error)
	UpdateMessage(sessionID string, message *Message) error
	DeleteMessage(sessionID string, messageID string) error

	// セッション状態管理
	StartSession(sessionID string) error
	PauseSession(sessionID string) error
	ResumeSession(sessionID string) error
	CompleteSession(sessionID string) error
	ArchiveSession(sessionID string) error

	// コンテキスト管理
	UpdateContext(sessionID string, context *ContextState) error
	CompressContext(sessionID string) error
	RestoreContext(sessionID string) (*ContextState, error)

	// 永続化・インポート・エクスポート
	SaveSession(sessionID string) error
	LoadSession(sessionID string) (*UnifiedSession, error)
	ExportSession(sessionID string, format string) ([]byte, error)
	ImportSession(data []byte, format string) (*UnifiedSession, error)

	// イベント管理
	RegisterEventHandler(eventType SessionEventType, handler SessionEventHandler)
	UnregisterEventHandler(eventType SessionEventType)
	EmitEvent(event SessionEvent)

	// 統計・メトリクス
	GetSessionStats(sessionID string) (*UnifiedSessionStats, error)
	GetGlobalStats() (*GlobalSessionStats, error)

	// クリーンアップ・メンテナンス
	CleanupExpiredSessions() (int, error)
	ArchiveOldSessions(olderThan time.Duration) (int, error)
	CompactSessionStorage() error

	// 設定管理
	UpdateConfig(config *ManagerConfig) error
	GetConfig() *ManagerConfig

	// 終了処理
	Shutdown() error
}

// unifiedSessionManager - 統合セッション管理実装
type unifiedSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*UnifiedSession
	config   *ManagerConfig

	// 依存関係
	streamManager  *streaming.Manager
	contextManager contextmanager.ContextManager
	llmProvider    llm.Provider

	// 永続化
	storageDir string

	// イベント処理
	eventHandlers map[SessionEventType][]SessionEventHandler
	eventChan     chan SessionEvent

	// バックグラウンド処理
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 統計
	globalStats *GlobalSessionStats
}

// ManagerConfig - マネージャー設定
type ManagerConfig struct {
	StorageDir         string        `json:"storage_dir"`
	AutoSave           bool          `json:"auto_save"`
	SaveInterval       time.Duration `json:"save_interval"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`
	MaxSessions        int           `json:"max_sessions"`
	DefaultExpiry      time.Duration `json:"default_expiry"`
	CompressionEnabled bool          `json:"compression_enabled"`
	EventQueueSize     int           `json:"event_queue_size"`
}

// GlobalSessionStats - グローバル統計
type GlobalSessionStats struct {
	TotalSessions          int                         `json:"total_sessions"`
	ActiveSessions         int                         `json:"active_sessions"`
	ArchivedSessions       int                         `json:"archived_sessions"`
	SessionsByType         map[UnifiedSessionType]int  `json:"sessions_by_type"`
	SessionsByState        map[UnifiedSessionState]int `json:"sessions_by_state"`
	TotalMessages          int64                       `json:"total_messages"`
	TotalTokens            int64                       `json:"total_tokens"`
	AverageSessionDuration time.Duration               `json:"average_session_duration"`
	LastUpdateTime         time.Time                   `json:"last_update_time"`
}

// NewUnifiedSessionManager - 新しい統合セッション管理を作成
func NewUnifiedSessionManager(
	config *ManagerConfig,
	streamManager *streaming.Manager,
	contextManager contextmanager.ContextManager,
	llmProvider llm.Provider,
) (UnifiedSessionManager, error) {
	config = ValidateManagerConfig(config)

	// ストレージディレクトリを作成
	if err := os.MkdirAll(config.StorageDir, 0755); err != nil {
		return nil, fmt.Errorf("ストレージディレクトリ作成エラー: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &unifiedSessionManager{
		sessions:       make(map[string]*UnifiedSession),
		config:         config,
		streamManager:  streamManager,
		contextManager: contextManager,
		llmProvider:    llmProvider,
		storageDir:     config.StorageDir,
		eventHandlers:  make(map[SessionEventType][]SessionEventHandler),
		eventChan:      make(chan SessionEvent, config.EventQueueSize),
		ctx:            ctx,
		cancel:         cancel,
		globalStats: &GlobalSessionStats{
			SessionsByType:  make(map[UnifiedSessionType]int),
			SessionsByState: make(map[UnifiedSessionState]int),
			LastUpdateTime:  time.Now(),
		},
	}

	// 既存セッションを読み込み
	if err := manager.loadExistingSessions(); err != nil {
		return nil, fmt.Errorf("既存セッション読み込みエラー: %w", err)
	}

	// バックグラウンド処理を開始
	manager.startBackgroundTasks()

	return manager, nil
}

// CreateSession - 新しいセッションを作成
func (m *unifiedSessionManager) CreateSession(sessionType UnifiedSessionType, config *SessionConfig) (*UnifiedSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config == nil {
		config = CreateSessionConfig(sessionType)
	}

	// セッション制限チェック
	if len(m.sessions) >= m.config.MaxSessions {
		return nil, fmt.Errorf("最大セッション数(%d)に達しています", m.config.MaxSessions)
	}

	sessionID := m.generateSessionID(sessionType)
	now := time.Now()

	session := &UnifiedSession{
		ID:             sessionID,
		Type:           sessionType,
		State:          SessionStateIdle,
		CreatedAt:      now,
		LastAccessedAt: now,
		UpdatedAt:      now,
		Config:         config,
		Metadata:       make(map[string]interface{}),
		Messages:       make([]Message, 0),
		Stats:          &UnifiedSessionStats{LastActivityTime: now},
		manager:        m,
		streamManager:  m.streamManager,
		contextManager: m.contextManager,
		llmProvider:    m.llmProvider,
		eventHandlers:  make(map[SessionEventType]SessionEventHandler),
	}

	// 有効期限設定
	if m.config.DefaultExpiry > 0 {
		expiryTime := now.Add(m.config.DefaultExpiry)
		session.ExpiresAt = &expiryTime
	}

	// コンテキスト状態を初期化
	if sessionType == SessionTypeInteractive || sessionType == SessionTypeVibeCoding {
		session.Context = &ContextState{
			TokenCount:  0,
			OpenFiles:   make([]string, 0),
			RecentFiles: make([]string, 0),
		}
	}

	// 履歴状態を初期化
	session.History = &HistoryState{
		CompressionEnabled: config.ContextCompression,
	}

	m.sessions[sessionID] = session

	// 統計更新
	m.updateGlobalStats()

	// イベント発行
	m.emitEvent(SessionEvent{
		Type:      EventSessionCreated,
		SessionID: sessionID,
		Timestamp: now,
		Data:      session,
	})

	// 自動保存
	if m.config.AutoSave && config.PersistToDisk {
		go m.SaveSession(sessionID)
	}

	return session, nil
}

// GetSession - セッションを取得
func (m *unifiedSessionManager) GetSession(sessionID string) (*UnifiedSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		// ディスクから読み込み試行
		if loadedSession, err := m.loadSessionFromDisk(sessionID); err == nil {
			m.mu.RUnlock()
			m.mu.Lock()
			m.sessions[sessionID] = loadedSession
			m.mu.Unlock()
			m.mu.RLock()
			return loadedSession, nil
		}
		return nil, fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	// 最終アクセス時刻を更新
	session.LastAccessedAt = time.Now()

	return session, nil
}

// UpdateSession - セッションを更新
func (m *unifiedSessionManager) UpdateSession(session *UnifiedSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[session.ID]; !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", session.ID)
	}

	session.UpdatedAt = time.Now()
	m.sessions[session.ID] = session

	// 統計更新
	m.updateGlobalStats()

	// イベント発行
	m.emitEvent(SessionEvent{
		Type:      EventSessionStarted,
		SessionID: session.ID,
		Timestamp: time.Now(),
	})

	// 自動保存
	if m.config.AutoSave && session.Config.PersistToDisk {
		go m.SaveSession(session.ID)
	}

	return nil
}

// DeleteSession - セッションを削除
func (m *unifiedSessionManager) DeleteSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	// ディスクから削除
	sessionFile := filepath.Join(m.storageDir, sessionID+".json")
	os.Remove(sessionFile)

	// メモリから削除
	delete(m.sessions, sessionID)

	// 統計更新
	m.updateGlobalStats()

	// イベント発行
	m.emitEvent(SessionEvent{
		Type:      EventSessionCompleted,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data:      session,
	})

	return nil
}

// ListSessions - セッション一覧を取得
func (m *unifiedSessionManager) ListSessions(filter *SessionFilter, sortBy SessionSortBy, sortOrder SessionSortOrder) ([]*UnifiedSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []*UnifiedSession

	// フィルタリング
	for _, session := range m.sessions {
		if m.matchesFilter(session, filter) {
			sessions = append(sessions, session)
		}
	}

	// ソート
	m.sortSessions(sessions, sortBy, sortOrder)

	// 制限適用
	if filter != nil {
		if filter.Offset > 0 && filter.Offset < len(sessions) {
			sessions = sessions[filter.Offset:]
		}
		if filter.Limit > 0 && filter.Limit < len(sessions) {
			sessions = sessions[:filter.Limit]
		}
	}

	return sessions, nil
}

// その他の主要メソッドの実装...

// generateSessionID - セッションIDを生成
func (m *unifiedSessionManager) generateSessionID(sessionType UnifiedSessionType) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s-%d", sessionType, timestamp)
}

// loadExistingSessions - 既存セッションを読み込み
func (m *unifiedSessionManager) loadExistingSessions() error {
	files, err := ioutil.ReadDir(m.storageDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // ディレクトリが存在しない場合はエラーなし
		}
		return err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		sessionID := strings.TrimSuffix(file.Name(), ".json")
		if session, err := m.loadSessionFromDisk(sessionID); err == nil {
			m.sessions[sessionID] = session
		}
	}

	m.updateGlobalStats()
	return nil
}

// loadSessionFromDisk - ディスクからセッション読み込み
func (m *unifiedSessionManager) loadSessionFromDisk(sessionID string) (*UnifiedSession, error) {
	sessionFile := filepath.Join(m.storageDir, sessionID+".json")
	data, err := ioutil.ReadFile(sessionFile)
	if err != nil {
		return nil, err
	}

	var session UnifiedSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	// 内部参照を復元
	session.manager = m
	session.streamManager = m.streamManager
	session.contextManager = m.contextManager
	session.llmProvider = m.llmProvider
	session.eventHandlers = make(map[SessionEventType]SessionEventHandler)

	return &session, nil
}

// updateGlobalStats - グローバル統計を更新
func (m *unifiedSessionManager) updateGlobalStats() {
	m.globalStats.TotalSessions = len(m.sessions)
	m.globalStats.ActiveSessions = 0
	m.globalStats.ArchivedSessions = 0
	m.globalStats.TotalMessages = 0
	m.globalStats.TotalTokens = 0

	// 型・状態別統計をリセット
	m.globalStats.SessionsByType = make(map[UnifiedSessionType]int)
	m.globalStats.SessionsByState = make(map[UnifiedSessionState]int)

	var totalDuration time.Duration
	sessionCount := 0

	for _, session := range m.sessions {
		// 型別統計
		m.globalStats.SessionsByType[session.Type]++

		// 状態別統計
		m.globalStats.SessionsByState[session.State]++
		if session.State == SessionStateActive {
			m.globalStats.ActiveSessions++
		}
		if session.State == SessionStateArchived {
			m.globalStats.ArchivedSessions++
		}

		// メッセージ・トークン統計
		m.globalStats.TotalMessages += int64(len(session.Messages))
		if session.Stats != nil {
			m.globalStats.TotalTokens += session.Stats.TotalTokens
			totalDuration += session.Stats.TotalDuration
			sessionCount++
		}
	}

	// 平均セッション時間を計算
	if sessionCount > 0 {
		m.globalStats.AverageSessionDuration = totalDuration / time.Duration(sessionCount)
	}

	m.globalStats.LastUpdateTime = time.Now()
}

// matchesFilter - フィルターマッチング
func (m *unifiedSessionManager) matchesFilter(session *UnifiedSession, filter *SessionFilter) bool {
	if filter == nil {
		return true
	}

	// タイプフィルター
	if len(filter.Types) > 0 {
		found := false
		for _, t := range filter.Types {
			if session.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 状態フィルター
	if len(filter.States) > 0 {
		found := false
		for _, s := range filter.States {
			if session.State == s {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// タグフィルター
	if len(filter.Tags) > 0 {
		for _, filterTag := range filter.Tags {
			found := false
			for _, sessionTag := range session.Tags {
				if sessionTag == filterTag {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// 時間フィルター
	if filter.CreatedAfter != nil && session.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}
	if filter.CreatedBefore != nil && session.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}
	if filter.AccessedAfter != nil && session.LastAccessedAt.Before(*filter.AccessedAfter) {
		return false
	}

	return true
}

// sortSessions - セッションソート
func (m *unifiedSessionManager) sortSessions(sessions []*UnifiedSession, sortBy SessionSortBy, sortOrder SessionSortOrder) {
	sort.Slice(sessions, func(i, j int) bool {
		var less bool

		switch sortBy {
		case SortByCreatedAt:
			less = sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
		case SortByLastAccessed:
			less = sessions[i].LastAccessedAt.Before(sessions[j].LastAccessedAt)
		case SortByUpdatedAt:
			less = sessions[i].UpdatedAt.Before(sessions[j].UpdatedAt)
		case SortByMessageCount:
			less = len(sessions[i].Messages) < len(sessions[j].Messages)
		case SortByTotalTokens:
			tokensI := int64(0)
			tokensJ := int64(0)
			if sessions[i].Stats != nil {
				tokensI = sessions[i].Stats.TotalTokens
			}
			if sessions[j].Stats != nil {
				tokensJ = sessions[j].Stats.TotalTokens
			}
			less = tokensI < tokensJ
		default:
			less = sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
		}

		if sortOrder == SortOrderDesc {
			return !less
		}
		return less
	})
}

// emitEvent - イベントを発行
func (m *unifiedSessionManager) emitEvent(event SessionEvent) {
	select {
	case m.eventChan <- event:
		// 正常にキューに追加
	default:
		// キューが満杯の場合はログ出力（実装省略）
	}
}

// startBackgroundTasks - バックグラウンドタスクを開始
func (m *unifiedSessionManager) startBackgroundTasks() {
	// イベント処理ワーカー
	m.wg.Add(1)
	go m.eventWorker()

	// 自動保存ワーカー
	if m.config.AutoSave {
		m.wg.Add(1)
		go m.autoSaveWorker()
	}

	// クリーンアップワーカー
	m.wg.Add(1)
	go m.cleanupWorker()
}

// eventWorker - イベント処理ワーカー
func (m *unifiedSessionManager) eventWorker() {
	defer m.wg.Done()

	for {
		select {
		case event := <-m.eventChan:
			m.processEvent(event)
		case <-m.ctx.Done():
			return
		}
	}
}

// processEvent - イベント処理
func (m *unifiedSessionManager) processEvent(event SessionEvent) {
	m.mu.RLock()
	handlers := m.eventHandlers[event.Type]
	m.mu.RUnlock()

	for _, handler := range handlers {
		if handler != nil {
			handler(event)
		}
	}
}

// autoSaveWorker - 自動保存ワーカー
func (m *unifiedSessionManager) autoSaveWorker() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.SaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.saveAllSessions()
		case <-m.ctx.Done():
			// 終了時に最後の保存を実行
			m.saveAllSessions()
			return
		}
	}
}

// cleanupWorker - クリーンアップワーカー
func (m *unifiedSessionManager) cleanupWorker() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.CleanupExpiredSessions()
		case <-m.ctx.Done():
			return
		}
	}
}

// saveAllSessions - 全セッション保存
func (m *unifiedSessionManager) saveAllSessions() {
	m.mu.RLock()
	sessionIDs := make([]string, 0, len(m.sessions))
	for id, session := range m.sessions {
		if session.Config.PersistToDisk {
			sessionIDs = append(sessionIDs, id)
		}
	}
	m.mu.RUnlock()

	for _, sessionID := range sessionIDs {
		m.SaveSession(sessionID)
	}
}

// DefaultManagerConfig - デフォルトマネージャー設定
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		StorageDir:         ".vyb/sessions",
		AutoSave:           true,
		SaveInterval:       5 * time.Minute,
		CleanupInterval:    time.Hour,
		MaxSessions:        1000,
		DefaultExpiry:      7 * 24 * time.Hour, // 1週間
		CompressionEnabled: true,
		EventQueueSize:     100,
	}
}

// ValidateManagerConfig - マネージャー設定を検証・修正
func ValidateManagerConfig(config *ManagerConfig) *ManagerConfig {
	if config == nil {
		return DefaultManagerConfig()
	}

	// ゼロ値や負の値を修正
	if config.SaveInterval <= 0 {
		config.SaveInterval = 5 * time.Minute
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = time.Hour
	}
	if config.MaxSessions <= 0 {
		config.MaxSessions = 1000
	}
	if config.EventQueueSize <= 0 {
		config.EventQueueSize = 100
	}

	return config
}
