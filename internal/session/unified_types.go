package session

import (
	"sync"
	"time"

	"github.com/glkt/vyb-code/internal/contextmanager"
	"github.com/glkt/vyb-code/internal/llm"
	// "github.com/glkt/vyb-code/internal/streaming" // 削除されたパッケージ
)

// UnifiedSessionType - 統合セッションタイプ
type UnifiedSessionType string

const (
	SessionTypePersistent  UnifiedSessionType = "persistent"  // 永続セッション
	SessionTypeChat        UnifiedSessionType = "chat"        // チャット会話
	SessionTypeInteractive UnifiedSessionType = "interactive" // インタラクティブ
	SessionTypeVibeCoding  UnifiedSessionType = "vibe_coding" // バイブコーディング
	SessionTypeTemporary   UnifiedSessionType = "temporary"   // 一時セッション
)

// UnifiedSessionState - 統合セッション状態
type UnifiedSessionState string

const (
	SessionStateIdle      UnifiedSessionState = "idle"
	SessionStateActive    UnifiedSessionState = "active"
	SessionStatePaused    UnifiedSessionState = "paused"
	SessionStateCompleted UnifiedSessionState = "completed"
	SessionStateError     UnifiedSessionState = "error"
	SessionStateArchived  UnifiedSessionState = "archived"
)

// UnifiedSession - 統合セッション構造体
type UnifiedSession struct {
	// 基本情報
	ID             string              `json:"id"`
	Type           UnifiedSessionType  `json:"type"`
	State          UnifiedSessionState `json:"state"`
	CreatedAt      time.Time           `json:"created_at"`
	LastAccessedAt time.Time           `json:"last_accessed_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	ExpiresAt      *time.Time          `json:"expires_at,omitempty"`

	// セッション設定
	Config   *SessionConfig         `json:"config"`
	Metadata map[string]interface{} `json:"metadata"`
	Tags     []string               `json:"tags,omitempty"`

	// コンテンツ管理
	Messages []Message     `json:"messages"`
	Context  *ContextState `json:"context,omitempty"`
	History  *HistoryState `json:"history,omitempty"`

	// 統計・パフォーマンス
	Stats *UnifiedSessionStats `json:"stats"`

	// 内部状態（JSONエクスポート対象外）
	mu      sync.RWMutex          `json:"-"`
	manager UnifiedSessionManager `json:"-"`
	// streamManager  *streaming.Manager            `json:"-"` // 使用停止
	contextManager contextmanager.ContextManager `json:"-"`
	llmProvider    llm.Provider                  `json:"-"`

	// イベントハンドラー
	eventHandlers map[SessionEventType]SessionEventHandler `json:"-"`
}

// SessionConfig - セッション設定
type SessionConfig struct {
	// 基本設定
	MaxMessages   int  `json:"max_messages"`
	MaxTokens     int  `json:"max_tokens"`
	AutoSave      bool `json:"auto_save"`
	PersistToDisk bool `json:"persist_to_disk"`

	// タイムアウト設定
	IdleTimeout   time.Duration `json:"idle_timeout"`
	ActiveTimeout time.Duration `json:"active_timeout"`

	// LLM設定
	Model            string  `json:"model,omitempty"`
	Temperature      float64 `json:"temperature,omitempty"`
	StreamingEnabled bool    `json:"streaming_enabled"`

	// コンテキスト管理設定
	ContextCompression bool `json:"context_compression"`
	MaxContextSize     int  `json:"max_context_size"`

	// インタラクティブ機能設定
	InteractiveMode bool `json:"interactive_mode"`
	VibeMode        bool `json:"vibe_mode"`
	ProactiveMode   bool `json:"proactive_mode"`

	// カスタム設定
	CustomSettings map[string]interface{} `json:"custom_settings,omitempty"`
}

// Message - 統合メッセージ構造体
type Message struct {
	ID         string                 `json:"id"`
	Role       MessageRole            `json:"role"`
	Content    string                 `json:"content"`
	Timestamp  time.Time              `json:"timestamp"`
	TokenCount int                    `json:"token_count,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`

	// 関連情報
	ParentID string `json:"parent_id,omitempty"`
	ThreadID string `json:"thread_id,omitempty"`

	// 実行情報
	ToolCalls   []ToolCall   `json:"tool_calls,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// MessageRole - メッセージ役割
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
	MessageRoleTool      MessageRole = "tool"
)

// ToolCall - ツール呼び出し情報
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    interface{}            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
}

// Attachment - メッセージ添付ファイル
type Attachment struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "file", "image", "code", etc.
	Name     string `json:"name"`
	Content  string `json:"content"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

// ContextState - コンテキスト状態
type ContextState struct {
	// 現在のコンテキスト
	CurrentContext    string `json:"current_context"`
	CompressedContext string `json:"compressed_context,omitempty"`

	// コンテキスト統計
	TokenCount       int     `json:"token_count"`
	CompressionRatio float64 `json:"compression_ratio,omitempty"`

	// プロジェクト情報
	WorkingDirectory string                 `json:"working_directory,omitempty"`
	ProjectInfo      map[string]interface{} `json:"project_info,omitempty"`

	// ファイルコンテキスト
	OpenFiles   []string `json:"open_files,omitempty"`
	RecentFiles []string `json:"recent_files,omitempty"`
}

// HistoryState - 履歴状態
type HistoryState struct {
	// 履歴統計
	TotalMessages int   `json:"total_messages"`
	TotalTokens   int64 `json:"total_tokens"`

	// アーカイブされたメッセージ
	ArchivedMessages int       `json:"archived_messages"`
	LastArchiveTime  time.Time `json:"last_archive_time,omitempty"`

	// 履歴圧縮
	CompressionEnabled bool    `json:"compression_enabled"`
	CompressionRatio   float64 `json:"compression_ratio,omitempty"`
}

// UnifiedSessionStats - 統合セッション統計
type UnifiedSessionStats struct {
	// 基本統計
	MessageCount      int `json:"message_count"`
	UserMessages      int `json:"user_messages"`
	AssistantMessages int `json:"assistant_messages"`
	SystemMessages    int `json:"system_messages"`
	ToolMessages      int `json:"tool_messages"`

	// トークン統計
	TotalTokens  int64 `json:"total_tokens"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`

	// 時間統計
	TotalDuration       time.Duration `json:"total_duration"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastActivityTime    time.Time     `json:"last_activity_time"`

	// パフォーマンス統計
	ToolCallCount   int `json:"tool_call_count"`
	SuccessfulCalls int `json:"successful_calls"`
	FailedCalls     int `json:"failed_calls"`

	// コンテキスト統計
	ContextSwitches   int `json:"context_switches"`
	CompressionEvents int `json:"compression_events"`

	// インタラクティブ統計（バイブモード用）
	InteractionCount    int `json:"interaction_count"`
	SuggestionCount     int `json:"suggestion_count"`
	AcceptedSuggestions int `json:"accepted_suggestions"`
}

// SessionEvent - セッションイベント
type SessionEvent struct {
	Type      SessionEventType       `json:"type"`
	SessionID string                 `json:"session_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      interface{}            `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SessionEventType - セッションイベントタイプ
type SessionEventType string

const (
	EventSessionCreated   SessionEventType = "session_created"
	EventSessionStarted   SessionEventType = "session_started"
	EventSessionPaused    SessionEventType = "session_paused"
	EventSessionResumed   SessionEventType = "session_resumed"
	EventSessionCompleted SessionEventType = "session_completed"
	EventSessionArchived  SessionEventType = "session_archived"
	EventSessionError     SessionEventType = "session_error"
	EventMessageAdded     SessionEventType = "message_added"
	EventMessageUpdated   SessionEventType = "message_updated"
	EventContextUpdated   SessionEventType = "context_updated"
	EventToolCalled       SessionEventType = "tool_called"
	EventStreamingStart   SessionEventType = "streaming_start"
	EventStreamingEnd     SessionEventType = "streaming_end"
)

// SessionEventHandler - セッションイベントハンドラー
type SessionEventHandler func(event SessionEvent) error

// SessionFilter - セッション検索フィルター
type SessionFilter struct {
	Types         []UnifiedSessionType   `json:"types,omitempty"`
	States        []UnifiedSessionState  `json:"states,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	CreatedAfter  *time.Time             `json:"created_after,omitempty"`
	CreatedBefore *time.Time             `json:"created_before,omitempty"`
	AccessedAfter *time.Time             `json:"accessed_after,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Limit         int                    `json:"limit,omitempty"`
	Offset        int                    `json:"offset,omitempty"`
}

// SessionSortBy - セッションソート基準
type SessionSortBy string

const (
	SortByCreatedAt    SessionSortBy = "created_at"
	SortByLastAccessed SessionSortBy = "last_accessed"
	SortByUpdatedAt    SessionSortBy = "updated_at"
	SortByMessageCount SessionSortBy = "message_count"
	SortByTotalTokens  SessionSortBy = "total_tokens"
)

// SessionSortOrder - セッションソート順序
type SessionSortOrder string

const (
	SortOrderAsc  SessionSortOrder = "asc"
	SortOrderDesc SessionSortOrder = "desc"
)

// DefaultSessionConfig - デフォルトセッション設定
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		MaxMessages:        1000,
		MaxTokens:          32000,
		AutoSave:           true,
		PersistToDisk:      true,
		IdleTimeout:        30 * time.Minute,
		ActiveTimeout:      24 * time.Hour,
		StreamingEnabled:   true,
		ContextCompression: true,
		MaxContextSize:     16000,
		InteractiveMode:    false,
		VibeMode:           false,
		ProactiveMode:      false,
		CustomSettings:     make(map[string]interface{}),
	}
}

// CreateSessionConfig - セッションタイプ別設定作成
func CreateSessionConfig(sessionType UnifiedSessionType) *SessionConfig {
	config := DefaultSessionConfig()

	switch sessionType {
	case SessionTypePersistent:
		config.PersistToDisk = true
		config.AutoSave = true
		config.MaxMessages = 5000
		config.IdleTimeout = time.Hour
		config.ActiveTimeout = 7 * 24 * time.Hour // 1週間

	case SessionTypeChat:
		config.PersistToDisk = true
		config.AutoSave = true
		config.StreamingEnabled = true
		config.MaxMessages = 1000
		config.IdleTimeout = 30 * time.Minute
		config.ActiveTimeout = 24 * time.Hour

	case SessionTypeInteractive:
		config.InteractiveMode = true
		config.StreamingEnabled = true
		config.ContextCompression = true
		config.MaxMessages = 500
		config.IdleTimeout = 15 * time.Minute
		config.ActiveTimeout = 2 * time.Hour

	case SessionTypeVibeCoding:
		config.VibeMode = true
		config.InteractiveMode = true
		config.ProactiveMode = true
		config.ContextCompression = true
		config.StreamingEnabled = true
		config.MaxMessages = 2000
		config.IdleTimeout = time.Hour
		config.ActiveTimeout = 8 * time.Hour

	case SessionTypeTemporary:
		config.PersistToDisk = false
		config.AutoSave = false
		config.MaxMessages = 100
		config.IdleTimeout = 10 * time.Minute
		config.ActiveTimeout = time.Hour
	}

	return config
}
