package conversation

import (
	"time"

	"github.com/glkt/vyb-code/internal/contextmanager"
)

// メモリ効率的な会話管理システム

// 会話の種類
type ConversationType int

const (
	ConversationTypeInteractive ConversationType = iota // インタラクティブな対話
	ConversationTypeCodeReview                          // コードレビュー会話
	ConversationTypeDebugging                           // デバッグ支援会話
	ConversationTypeLearning                            // 学習支援会話
	ConversationTypePlanning                            // 計画・設計会話
)

// 会話メッセージの圧縮状態
type CompressionState int

const (
	CompressionStateUncompressed CompressionState = iota // 未圧縮
	CompressionStatePartial                              // 部分圧縮
	CompressionStateCompressed                           // 完全圧縮
	CompressionStateArchived                             // アーカイブ済み
)

// 効率的会話メッセージ
type EfficientChatMessage struct {
	// 基本情報
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`

	// コンテキスト管理
	ContextItems    []*contextmanager.ContextItem `json:"context_items,omitempty"`
	RelevanceScore  float64                       `json:"relevance_score"`
	ImportanceScore float64                       `json:"importance_score"`

	// 圧縮管理
	CompressionState CompressionState `json:"compression_state"`
	OriginalLength   int              `json:"original_length"`
	CompressedLength int              `json:"compressed_length,omitempty"`
	Summary          string           `json:"summary,omitempty"`

	// メタデータ
	Metadata     map[string]string `json:"metadata"`
	AccessCount  int               `json:"access_count"`
	LastAccessed time.Time         `json:"last_accessed"`

	// 関連性追跡
	RelatedMessages []string `json:"related_messages,omitempty"`
	ThreadID        string   `json:"thread_id,omitempty"`
}

// 会話スレッド
type ConversationThread struct {
	ID           string                  `json:"id"`
	Type         ConversationType        `json:"type"`
	Title        string                  `json:"title"`
	Messages     []*EfficientChatMessage `json:"messages"`
	StartTime    time.Time               `json:"start_time"`
	LastActivity time.Time               `json:"last_activity"`

	// 効率化設定
	MaxMessages      int     `json:"max_messages"`      // 最大保持メッセージ数
	CompressionRatio float64 `json:"compression_ratio"` // 圧縮比率

	// メタデータとメトリクス
	Metadata         map[string]string `json:"metadata"`
	TotalTokens      int               `json:"total_tokens"`
	CompressedTokens int               `json:"compressed_tokens"`

	// 学習データ
	UserPreferences  map[string]interface{} `json:"user_preferences,omitempty"`
	ConversationFlow []string               `json:"conversation_flow"`
}

// 会話圧縮結果
type CompressionResult struct {
	ThreadID           string    `json:"thread_id"`
	OriginalMessages   int       `json:"original_messages"`
	CompressedMessages int       `json:"compressed_messages"`
	OriginalTokens     int       `json:"original_tokens"`
	CompressedTokens   int       `json:"compressed_tokens"`
	CompressionRatio   float64   `json:"compression_ratio"`
	Summary            string    `json:"summary"`
	KeyTopics          []string  `json:"key_topics"`
	Timestamp          time.Time `json:"timestamp"`
}

// メモリ効率的会話管理インターフェース
type ConversationManager interface {
	// スレッド管理
	CreateThread(conversationType ConversationType, title string) (*ConversationThread, error)
	GetThread(threadID string) (*ConversationThread, error)
	UpdateThread(thread *ConversationThread) error
	DeleteThread(threadID string) error
	ListThreads(limit int) ([]*ConversationThread, error)

	// メッセージ管理
	AddMessage(threadID string, message *EfficientChatMessage) error
	GetMessages(threadID string, limit int, includeCompressed bool) ([]*EfficientChatMessage, error)
	GetRelevantMessages(threadID string, query string, limit int) ([]*EfficientChatMessage, error)

	// 圧縮管理
	CompressThread(threadID string, aggressive bool) (*CompressionResult, error)
	AutoCompress(threadID string) (*CompressionResult, error)

	// コンテキスト統合
	UpdateMessageContext(threadID string, messageID string, contextItems []*contextmanager.ContextItem) error
	GetConversationContext(threadID string, maxItems int) ([]*contextmanager.ContextItem, error)

	// 効率性最適化
	OptimizeMemoryUsage(threadID string) error
	GetMemoryStats(threadID string) (*MemoryStats, error)

	// 学習と適応
	LearnFromInteraction(threadID string, messageID string, userFeedback UserFeedback) error
	AdaptToUserPreferences(threadID string) error
}

// メモリ統計
type MemoryStats struct {
	ThreadID             string    `json:"thread_id"`
	TotalMessages        int       `json:"total_messages"`
	UncompressedMessages int       `json:"uncompressed_messages"`
	CompressedMessages   int       `json:"compressed_messages"`
	ArchivedMessages     int       `json:"archived_messages"`
	TotalMemoryUsage     int64     `json:"total_memory_usage"`     // bytes
	OptimizedMemoryUsage int64     `json:"optimized_memory_usage"` // bytes
	MemorySavings        int64     `json:"memory_savings"`         // bytes
	CompressionRatio     float64   `json:"compression_ratio"`
	LastOptimized        time.Time `json:"last_optimized"`
}

// ユーザーフィードバック
type UserFeedback struct {
	Type      FeedbackType      `json:"type"`
	Rating    int               `json:"rating"` // 1-5
	Helpful   bool              `json:"helpful"`
	Relevant  bool              `json:"relevant"`
	Clear     bool              `json:"clear"`
	Comments  string            `json:"comments,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// フィードバックタイプ
type FeedbackType int

const (
	FeedbackTypeResponse   FeedbackType = iota // 応答に対するフィードバック
	FeedbackTypeSuggestion                     // 提案に対するフィードバック
	FeedbackTypeGeneral                        // 一般的なフィードバック
)

// 会話最適化設定
type OptimizationConfig struct {
	// 圧縮設定
	AutoCompressionEnabled     bool          `json:"auto_compression_enabled"`
	CompressionThreshold       int           `json:"compression_threshold"` // メッセージ数
	CompressionInterval        time.Duration `json:"compression_interval"`
	AggressiveCompressionRatio float64       `json:"aggressive_compression_ratio"` // 0.0-1.0

	// メモリ管理
	MaxMemoryUsage            int64         `json:"max_memory_usage"` // bytes
	MemoryCheckInterval       time.Duration `json:"memory_check_interval"`
	LowMemoryCompressionRatio float64       `json:"low_memory_compression_ratio"`

	// アクセスパターン学習
	AccessPatternLearning bool    `json:"access_pattern_learning"`
	RelevanceDecayFactor  float64 `json:"relevance_decay_factor"`  // 時間による関連性減衰
	ImportanceBoostFactor float64 `json:"importance_boost_factor"` // 重要度ブースト

	// ユーザー適応
	UserAdaptationEnabled bool    `json:"user_adaptation_enabled"`
	PreferenceWeight      float64 `json:"preference_weight"`      // ユーザー好み重み
	FeedbackLearningRate  float64 `json:"feedback_learning_rate"` // フィードバック学習率
}

// 会話分析結果
type ConversationAnalysis struct {
	ThreadID          string    `json:"thread_id"`
	AnalysisTimestamp time.Time `json:"analysis_timestamp"`

	// パターン分析
	ConversationPatterns []string `json:"conversation_patterns"`
	UserBehaviorPatterns []string `json:"user_behavior_patterns"`
	TopicTransitions     []string `json:"topic_transitions"`

	// 効率性メトリクス
	AverageResponseTime    time.Duration `json:"average_response_time"`
	ContextSwitchFrequency float64       `json:"context_switch_frequency"`
	RedundancyScore        float64       `json:"redundancy_score"` // 0.0-1.0
	ClarityScore           float64       `json:"clarity_score"`    // 0.0-1.0

	// 推奨事項
	OptimizationRecommendations []string `json:"optimization_recommendations"`
	CompressionRecommendations  []string `json:"compression_recommendations"`

	// 学習洞察
	LearnedPreferences     map[string]interface{} `json:"learned_preferences"`
	ImprovementSuggestions []string               `json:"improvement_suggestions"`
}

// インテリジェントメッセージフィルター
type MessageFilter struct {
	// フィルター条件
	MinRelevanceScore  float64       `json:"min_relevance_score"`
	MinImportanceScore float64       `json:"min_importance_score"`
	MaxAge             time.Duration `json:"max_age"`
	RequiredRoles      []string      `json:"required_roles,omitempty"`
	ExcludedRoles      []string      `json:"excluded_roles,omitempty"`

	// コンテキストフィルター
	RequireContext bool     `json:"require_context"`
	ContextTypes   []string `json:"context_types,omitempty"`

	// 圧縮状態フィルター
	AllowedCompressionStates []CompressionState `json:"allowed_compression_states"`

	// メタデータフィルター
	MetadataFilters map[string]string `json:"metadata_filters,omitempty"`

	// 並び順
	SortBy    SortCriteria `json:"sort_by"`
	SortOrder SortOrder    `json:"sort_order"`
}

// ソート条件
type SortCriteria int

const (
	SortByTimestamp SortCriteria = iota
	SortByRelevance
	SortByImportance
	SortByAccessCount
)

// ソート順
type SortOrder int

const (
	SortOrderAsc SortOrder = iota
	SortOrderDesc
)
