package contextmanager

import (
	"time"

	"github.com/glkt/vyb-code/internal/llm"
)

// コンテキスト管理の種類
type ContextType int

const (
	// 即座のコンテキスト - 常時保持（現在編集中のファイル・関数）
	ContextTypeImmediate ContextType = iota
	// 短期コンテキスト - 1-2時間保持（最近の対話・変更履歴）
	ContextTypeShortTerm
	// 中期コンテキスト - 要約保持（プロジェクト構造・主要決定事項）
	ContextTypeMediumTerm
	// 長期コンテキスト - メタデータ（開発パターン・設定）
	ContextTypeLongTerm
)

// コンテキスト項目
type ContextItem struct {
	ID          string            `json:"id"`
	Type        ContextType       `json:"type"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata"`
	Timestamp   time.Time         `json:"timestamp"`
	Relevance   float64           `json:"relevance"`    // 0.0-1.0の関連度スコア
	Importance  float64           `json:"importance"`   // 0.0-1.0の重要度スコア
	AccessCount int               `json:"access_count"` // アクセス回数
	LastAccess  time.Time         `json:"last_access"`
}

// 圧縮されたコンテキスト
type CompressedContext struct {
	Summary         string            `json:"summary"`
	KeyPoints       []string          `json:"key_points"`
	ImportantFiles  []string          `json:"important_files"`
	RecentDecisions []string          `json:"recent_decisions"`
	Metadata        map[string]string `json:"metadata"`
	CompressedAt    time.Time         `json:"compressed_at"`
	OriginalSize    int               `json:"original_size"`
	CompressedSize  int               `json:"compressed_size"`
}

// スマートコンテキストマネージャー
type SmartContextManager struct {
	// コンテキスト項目の階層管理
	immediateContext  []*ContextItem // 即座のコンテキスト
	shortTermContext  []*ContextItem // 短期コンテキスト
	mediumTermContext []*ContextItem // 中期コンテキスト（圧縮済み）
	longTermContext   []*ContextItem // 長期コンテキスト（メタデータ）

	// 圧縮履歴
	compressionHistory []*CompressedContext

	// 設定
	maxImmediateItems  int     // 即座のコンテキストの最大項目数
	maxShortTermItems  int     // 短期コンテキストの最大項目数
	compressionRatio   float64 // 圧縮比率（0.0-1.0）
	relevanceThreshold float64 // 関連度閾値

	// メトリクス
	totalCompressed   int64     // 圧縮された項目数
	totalMemorySaved  int64     // 節約されたメモリ量（バイト）
	lastCompressionAt time.Time // 最後の圧縮実行時刻
}

// コンテキスト管理インターフェース
type ContextManager interface {
	// コンテキスト項目の追加
	AddContext(item *ContextItem) error

	// コンテキストの取得（重要度順）
	GetRelevantContext(query string, maxItems int) ([]*ContextItem, error)

	// コンテキストの動的圧縮
	CompressContext(forceCompress bool) (*CompressedContext, error)

	// 関連度の計算
	CalculateRelevance(item *ContextItem, query string) float64

	// メモリ使用量の取得
	GetMemoryUsage() (int64, error)

	// 統計情報の取得
	GetStats() (*ContextStats, error)

	// コンテキストのクリア
	ClearContext(contextType ContextType) error
}

// コンテキスト統計
type ContextStats struct {
	TotalItems         int       `json:"total_items"`
	ImmediateItems     int       `json:"immediate_items"`
	ShortTermItems     int       `json:"short_term_items"`
	MediumTermItems    int       `json:"medium_term_items"`
	LongTermItems      int       `json:"long_term_items"`
	TotalMemoryUsage   int64     `json:"total_memory_usage"`
	CompressionRatio   float64   `json:"compression_ratio"`
	LastCompressionAt  time.Time `json:"last_compression_at"`
	AverageRelevance   float64   `json:"average_relevance"`
	CompressionHistory int       `json:"compression_history"`
}

// 会話メッセージの拡張（コンテキスト対応）
type EnhancedChatMessage struct {
	llm.ChatMessage
	ContextItems []*ContextItem `json:"context_items"`
	Compressed   bool           `json:"compressed"`
	Summary      string         `json:"summary,omitempty"`
}

// インタラクティブセッション状態
type InteractiveSessionState struct {
	CurrentFile     string            `json:"current_file"`
	CurrentFunction string            `json:"current_function"`
	WorkingContext  []*ContextItem    `json:"working_context"`
	RecentChanges   []string          `json:"recent_changes"`
	UserIntent      string            `json:"user_intent"`
	SessionMetadata map[string]string `json:"session_metadata"`
	StartTime       time.Time         `json:"start_time"`
	LastActivity    time.Time         `json:"last_activity"`
}
