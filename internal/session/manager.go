package session

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// 永続的会話セッション管理
type Manager struct {
	mu          sync.RWMutex
	sessionsDir string
	currentID   string
	sessions    map[string]*Session
	maxSessions int
	maxTurnAge  time.Duration
}

// セッション構造
type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Model     string    `json:"model"`
	Provider  string    `json:"provider"`
	TurnCount int       `json:"turnCount"`
	Turns     []Turn    `json:"turns"`
	Context   Context   `json:"context"`
	Metadata  Metadata  `json:"metadata"`
}

// 会話ターン
type Turn struct {
	ID        string        `json:"id"`
	Type      TurnType      `json:"type"` // "user", "assistant", "system"
	Content   string        `json:"content"`
	Timestamp time.Time     `json:"timestamp"`
	Metadata  Metadata      `json:"metadata,omitempty"`
	Files     []string      `json:"files,omitempty"`    // 関連ファイル
	Commands  []string      `json:"commands,omitempty"` // 実行したコマンド
	Tools     []string      `json:"tools,omitempty"`    // 使用したツール
	MCPTools  []MCPToolCall `json:"mcpTools,omitempty"` // 使用したMCPツール
}

// ターンタイプ
type TurnType string

const (
	TurnTypeUser      TurnType = "user"
	TurnTypeAssistant TurnType = "assistant"
	TurnTypeSystem    TurnType = "system"
	TurnTypeTool      TurnType = "tool"
)

// セッションコンテキスト
type Context struct {
	WorkspaceDir  string            `json:"workspaceDir"`
	CurrentBranch string            `json:"currentBranch,omitempty"`
	RecentFiles   []string          `json:"recentFiles,omitempty"`
	ProjectInfo   map[string]string `json:"projectInfo,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	ActiveTools   []string          `json:"activeTools,omitempty"`
	Memory        []MemoryItem      `json:"memory,omitempty"`
}

// メモリアイテム
type MemoryItem struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Type      string    `json:"type"` // "fact", "preference", "context"
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Relevance float64   `json:"relevance"` // 関連度スコア
}

// MCPツール呼び出し記録
type MCPToolCall struct {
	Server    string                 `json:"server"`
	Tool      string                 `json:"tool"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    string                 `json:"result"`
	Success   bool                   `json:"success"`
	Duration  string                 `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}

// メタデータ
type Metadata map[string]interface{}

// セッション統計
type SessionStats struct {
	TotalSessions  int            `json:"totalSessions"`
	ActiveSessions int            `json:"activeSessions"`
	TotalTurns     int            `json:"totalTurns"`
	AverageTurns   float64        `json:"averageTurns"`
	ModelUsage     map[string]int `json:"modelUsage"`
	LanguageStats  map[string]int `json:"languageStats"`
	RecentActivity []time.Time    `json:"recentActivity"`
}

// 新しいセッションマネージャーを作成
func NewManager() *Manager {
	homeDir, _ := os.UserHomeDir()
	sessionsDir := filepath.Join(homeDir, ".vyb", "sessions")

	return &Manager{
		sessionsDir: sessionsDir,
		sessions:    make(map[string]*Session),
		maxSessions: 100,
		maxTurnAge:  30 * 24 * time.Hour, // 30日
	}
}

// セッションディレクトリを初期化
func (m *Manager) Initialize() error {
	if err := os.MkdirAll(m.sessionsDir, 0700); err != nil {
		return fmt.Errorf("セッションディレクトリ作成失敗: %w", err)
	}

	// 既存セッションを読み込み
	return m.loadExistingSessions()
}

// 新しいセッションを作成
func (m *Manager) CreateSession(title, model, provider string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// セッションIDを生成
	id := m.generateSessionID(title)

	session := &Session{
		ID:        id,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Model:     model,
		Provider:  provider,
		TurnCount: 0,
		Turns:     make([]Turn, 0),
		Context:   Context{},
		Metadata:  make(Metadata),
	}

	// ワークスペース情報を設定
	if workDir, err := os.Getwd(); err == nil {
		session.Context.WorkspaceDir = workDir
	}

	m.sessions[id] = session
	m.currentID = id

	// ファイルに保存
	if err := m.saveSession(session); err != nil {
		delete(m.sessions, id)
		return nil, fmt.Errorf("セッション保存失敗: %w", err)
	}

	return session, nil
}

// セッションIDを生成
func (m *Manager) generateSessionID(title string) string {
	timestamp := time.Now().Format("20060102-150405")
	hash := sha256.Sum256([]byte(title + timestamp))
	return fmt.Sprintf("%s-%x", timestamp, hash[:4])
}

// セッションにターンを追加
func (m *Manager) AddTurn(sessionID string, turnType TurnType, content string, metadata Metadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	turn := Turn{
		ID:        fmt.Sprintf("turn-%d", session.TurnCount+1),
		Type:      turnType,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	session.Turns = append(session.Turns, turn)
	session.TurnCount++
	session.UpdatedAt = time.Now()

	// セッションを保存
	return m.saveSession(session)
}

// セッションを取得
func (m *Manager) GetSession(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	// セッションのコピーを返す
	sessionCopy := *session
	return &sessionCopy, nil
}

// 現在のセッションを取得
func (m *Manager) GetCurrentSession() (*Session, error) {
	if m.currentID == "" {
		return nil, fmt.Errorf("アクティブなセッションがありません")
	}
	return m.GetSession(m.currentID)
}

// セッション一覧を取得
func (m *Manager) ListSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessionCopy := *session
		sessions = append(sessions, &sessionCopy)
	}

	// 更新時間順でソート（新しい順）
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions
}

// セッションを切り替え
func (m *Manager) SwitchSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[sessionID]; !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	m.currentID = sessionID
	return nil
}

// セッションを削除
func (m *Manager) DeleteSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[sessionID]; !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	// ファイルを削除
	sessionFile := filepath.Join(m.sessionsDir, sessionID+".json")
	if err := os.Remove(sessionFile); err != nil {
		return fmt.Errorf("セッションファイル削除失敗: %w", err)
	}

	delete(m.sessions, sessionID)

	// 現在のセッションが削除された場合はクリア
	if m.currentID == sessionID {
		m.currentID = ""
	}

	return nil
}

// メモリを追加
func (m *Manager) AddMemory(sessionID, key, value, memoryType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	memory := MemoryItem{
		Key:       key,
		Value:     value,
		Type:      memoryType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Relevance: 1.0,
	}

	session.Context.Memory = append(session.Context.Memory, memory)
	session.UpdatedAt = time.Now()

	return m.saveSession(session)
}

// MCPツール呼び出しを記録
func (m *Manager) AddMCPToolCall(sessionID string, toolCall MCPToolCall) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	// 最新のターンを取得または作成
	if len(session.Turns) == 0 {
		return fmt.Errorf("セッションにターンがありません")
	}

	// 最新のターンにMCPツール呼び出しを追加
	lastTurnIndex := len(session.Turns) - 1
	session.Turns[lastTurnIndex].MCPTools = append(session.Turns[lastTurnIndex].MCPTools, toolCall)
	session.UpdatedAt = time.Now()

	return m.saveSession(session)
}

// セッション統計を取得
func (m *Manager) GetStats() SessionStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := SessionStats{
		TotalSessions:  len(m.sessions),
		ActiveSessions: 0,
		TotalTurns:     0,
		ModelUsage:     make(map[string]int),
		LanguageStats:  make(map[string]int),
		RecentActivity: make([]time.Time, 0),
	}

	recentThreshold := time.Now().Add(-24 * time.Hour)

	for _, session := range m.sessions {
		stats.TotalTurns += session.TurnCount

		if session.UpdatedAt.After(recentThreshold) {
			stats.ActiveSessions++
			stats.RecentActivity = append(stats.RecentActivity, session.UpdatedAt)
		}

		stats.ModelUsage[session.Model]++

		// ファイル言語統計
		for _, turn := range session.Turns {
			for _, file := range turn.Files {
				ext := strings.ToLower(filepath.Ext(file))
				stats.LanguageStats[ext]++
			}
		}
	}

	if stats.TotalSessions > 0 {
		stats.AverageTurns = float64(stats.TotalTurns) / float64(stats.TotalSessions)
	}

	return stats
}

// セッションをファイルに保存
func (m *Manager) saveSession(session *Session) error {
	sessionFile := filepath.Join(m.sessionsDir, session.ID+".json")

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sessionFile, data, 0600)
}

// 既存セッションを読み込み
func (m *Manager) loadExistingSessions() error {
	files, err := os.ReadDir(m.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // ディレクトリが存在しない場合は正常
		}
		return err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		sessionFile := filepath.Join(m.sessionsDir, file.Name())
		data, err := os.ReadFile(sessionFile)
		if err != nil {
			continue // エラーファイルはスキップ
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue // 破損ファイルはスキップ
		}

		m.sessions[session.ID] = &session
	}

	return nil
}

// 古いセッションをクリーンアップ
func (m *Manager) CleanupOldSessions() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-m.maxTurnAge)
	var toDelete []string

	for id, session := range m.sessions {
		if session.UpdatedAt.Before(cutoff) {
			toDelete = append(toDelete, id)
		}
	}

	// セッション数制限もチェック
	if len(m.sessions) > m.maxSessions {
		// 古い順にソート
		sessions := make([]*Session, 0, len(m.sessions))
		for _, session := range m.sessions {
			sessions = append(sessions, session)
		}

		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].UpdatedAt.Before(sessions[j].UpdatedAt)
		})

		// 超過分を削除対象に追加
		excess := len(sessions) - m.maxSessions
		for i := 0; i < excess; i++ {
			toDelete = append(toDelete, sessions[i].ID)
		}
	}

	// 削除実行
	for _, id := range toDelete {
		if err := m.DeleteSession(id); err != nil {
			continue // エラーは無視して続行
		}
	}

	return nil
}

// セッション検索
func (m *Manager) SearchSessions(query string) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*Session

	for _, session := range m.sessions {
		// タイトルで検索
		if strings.Contains(strings.ToLower(session.Title), query) {
			sessionCopy := *session
			results = append(results, &sessionCopy)
			continue
		}

		// ターン内容で検索
		for _, turn := range session.Turns {
			if strings.Contains(strings.ToLower(turn.Content), query) {
				sessionCopy := *session
				results = append(results, &sessionCopy)
				break
			}
		}
	}

	// 関連度順でソート（更新時間順）
	sort.Slice(results, func(i, j int) bool {
		return results[i].UpdatedAt.After(results[j].UpdatedAt)
	})

	return results
}

// セッションエクスポート
func (m *Manager) ExportSession(sessionID, format string) ([]byte, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return json.MarshalIndent(session, "", "  ")
	case "markdown":
		return m.exportToMarkdown(session), nil
	case "text":
		return m.exportToText(session), nil
	default:
		return nil, fmt.Errorf("未対応のフォーマット: %s", format)
	}
}

// Markdown形式でエクスポート
func (m *Manager) exportToMarkdown(session *Session) []byte {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# %s\n\n", session.Title))
	builder.WriteString(fmt.Sprintf("**作成日時:** %s\n", session.CreatedAt.Format("2006-01-02 15:04:05")))
	builder.WriteString(fmt.Sprintf("**更新日時:** %s\n", session.UpdatedAt.Format("2006-01-02 15:04:05")))
	builder.WriteString(fmt.Sprintf("**モデル:** %s (%s)\n", session.Model, session.Provider))
	builder.WriteString(fmt.Sprintf("**ターン数:** %d\n\n", session.TurnCount))

	for i, turn := range session.Turns {
		builder.WriteString(fmt.Sprintf("## Turn %d - %s\n\n", i+1, turn.Type))
		builder.WriteString(fmt.Sprintf("**時刻:** %s\n\n", turn.Timestamp.Format("15:04:05")))

		if turn.Type == TurnTypeUser {
			builder.WriteString("### ユーザー\n\n")
		} else if turn.Type == TurnTypeAssistant {
			builder.WriteString("### アシスタント\n\n")
		}

		builder.WriteString(turn.Content)
		builder.WriteString("\n\n")

		if len(turn.Files) > 0 {
			builder.WriteString("**関連ファイル:**\n")
			for _, file := range turn.Files {
				builder.WriteString(fmt.Sprintf("- `%s`\n", file))
			}
			builder.WriteString("\n")
		}

		if len(turn.Commands) > 0 {
			builder.WriteString("**実行コマンド:**\n")
			for _, cmd := range turn.Commands {
				builder.WriteString(fmt.Sprintf("```bash\n%s\n```\n", cmd))
			}
			builder.WriteString("\n")
		}
	}

	return []byte(builder.String())
}

// テキスト形式でエクスポート
func (m *Manager) exportToText(session *Session) []byte {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("セッション: %s\n", session.Title))
	builder.WriteString(fmt.Sprintf("作成: %s | 更新: %s\n",
		session.CreatedAt.Format("2006-01-02 15:04:05"),
		session.UpdatedAt.Format("2006-01-02 15:04:05")))
	builder.WriteString(strings.Repeat("=", 60) + "\n\n")

	for i, turn := range session.Turns {
		timestamp := turn.Timestamp.Format("15:04:05")
		builder.WriteString(fmt.Sprintf("[%s] %s #%d:\n", timestamp, turn.Type, i+1))
		builder.WriteString(turn.Content)
		builder.WriteString("\n\n")
	}

	return []byte(builder.String())
}

// セッションをインポート
func (m *Manager) ImportSession(data []byte, format string) (*Session, error) {
	switch format {
	case "json":
		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			return nil, fmt.Errorf("JSONパース失敗: %w", err)
		}

		// 新しいIDを生成
		session.ID = m.generateSessionID(session.Title)
		session.CreatedAt = time.Now()
		session.UpdatedAt = time.Now()

		m.mu.Lock()
		m.sessions[session.ID] = &session
		m.mu.Unlock()

		if err := m.saveSession(&session); err != nil {
			return nil, err
		}

		return &session, nil
	default:
		return nil, fmt.Errorf("未対応のフォーマット: %s", format)
	}
}

// コンテキスト圧縮（古いターンを要約）
func (m *Manager) CompressContext(sessionID string, maxTurns int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionID)
	}

	if len(session.Turns) <= maxTurns {
		return nil // 圧縮不要
	}

	// 古いターンを要約に変換
	turnsToCompress := session.Turns[:len(session.Turns)-maxTurns]
	remainingTurns := session.Turns[len(session.Turns)-maxTurns:]

	// 簡易要約作成
	summary := m.createTurnSummary(turnsToCompress)

	// 要約ターンを作成
	summaryTurn := Turn{
		ID:        "summary-compressed",
		Type:      TurnTypeSystem,
		Content:   summary,
		Timestamp: time.Now(),
		Metadata:  Metadata{"compressed": true, "original_turns": len(turnsToCompress)},
	}

	// セッションを更新
	session.Turns = append([]Turn{summaryTurn}, remainingTurns...)
	session.UpdatedAt = time.Now()

	return m.saveSession(session)
}

// ターン要約を作成
func (m *Manager) createTurnSummary(turns []Turn) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("以前の会話要約（%dターン）:\n", len(turns)))

	for _, turn := range turns {
		content := turn.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		builder.WriteString(fmt.Sprintf("- %s: %s\n", turn.Type, content))
	}

	return builder.String()
}
