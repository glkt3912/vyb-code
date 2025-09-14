package streaming

import (
	"context"
	"io"
	"time"
)

// StreamingProcessor - 統一ストリーミング処理インターフェース
type StreamingProcessor interface {
	// Process は入力を受け取り、出力にストリーミング処理して書き込む
	Process(ctx context.Context, input io.Reader, output io.Writer, options *StreamOptions) error
	
	// ProcessString は文字列をストリーミング処理して出力に書き込む
	ProcessString(ctx context.Context, content string, output io.Writer, options *StreamOptions) error
	
	// SetConfig はストリーミング設定を更新する
	SetConfig(config *UnifiedStreamConfig)
	
	// GetMetrics はストリーミングメトリクスを取得する
	GetMetrics() *StreamMetrics
}

// StreamType - ストリーミングタイプ
type StreamType string

const (
	StreamTypeLLMResponse StreamType = "llm_response" // LLM応答処理
	StreamTypeUIDisplay   StreamType = "ui_display"   // UI表示処理
	StreamTypeInterrupted StreamType = "interrupted"  // 中断可能処理
)

// StreamOptions - ストリーミングオプション
type StreamOptions struct {
	Type              StreamType        `json:"type"`
	StreamID          string           `json:"stream_id"`
	Model             string           `json:"model,omitempty"`
	EnableInterrupt   bool             `json:"enable_interrupt"`
	ShowThinking      bool             `json:"show_thinking"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	ContextBefore     int              `json:"context_before,omitempty"`
	ContextAfter      int              `json:"context_after,omitempty"`
}

// UnifiedStreamConfig - 統一ストリーミング設定
type UnifiedStreamConfig struct {
	// 基本設定
	BufferSize    int           `json:"buffer_size"`
	FlushInterval time.Duration `json:"flush_interval"`
	
	// UI表示設定
	TokenDelay      time.Duration `json:"token_delay"`
	SentenceDelay   time.Duration `json:"sentence_delay"`
	ParagraphDelay  time.Duration `json:"paragraph_delay"`
	CodeBlockDelay  time.Duration `json:"code_block_delay"`
	EnableStreaming bool          `json:"enable_streaming"`
	MaxLineLength   int           `json:"max_line_length"`
	
	// パフォーマンス設定
	MaxWorkers  int `json:"max_workers"`
	QueueSize   int `json:"queue_size"`
	Timeout     time.Duration `json:"timeout"`
}

// StreamMetrics - 統合ストリーミングメトリクス
type StreamMetrics struct {
	// 基本統計
	TotalStreams   int64         `json:"total_streams"`
	ActiveStreams  int64         `json:"active_streams"`
	TotalTokens    int64         `json:"total_tokens"`
	TotalChunks    int64         `json:"total_chunks"`
	ErrorCount     int64         `json:"error_count"`
	
	// パフォーマンス統計
	AverageLatency    time.Duration `json:"average_latency"`
	LastStreamTime    time.Time     `json:"last_stream_time"`
	ProcessingTime    time.Duration `json:"processing_time"`
	
	// UI表示統計
	TotalLines        int64         `json:"total_lines"`
	DisplayDuration   time.Duration `json:"display_duration"`
	InterruptCount    int64         `json:"interrupt_count"`
}

// StreamEvent - ストリーミングイベント
type StreamEvent struct {
	Type      EventType            `json:"type"`
	StreamID  string               `json:"stream_id"`
	Timestamp time.Time            `json:"timestamp"`
	Data      interface{}          `json:"data,omitempty"`
	Error     string               `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EventType - イベントタイプ
type EventType string

const (
	EventStreamStart     EventType = "stream_start"
	EventChunkReceived   EventType = "chunk_received"
	EventStreamComplete  EventType = "stream_complete"
	EventStreamError     EventType = "stream_error"
	EventStreamCancel    EventType = "stream_cancel"
	EventDisplayStart    EventType = "display_start"
	EventDisplayComplete EventType = "display_complete"
	EventInterrupt       EventType = "interrupt"
)

// EventHandler - イベントハンドラー関数型
type EventHandler func(event StreamEvent) error

// StreamStatus - ストリーム状態
type StreamStatus string

const (
	StreamStatusIdle      StreamStatus = "idle"
	StreamStatusStarting  StreamStatus = "starting"
	StreamStatusStreaming StreamStatus = "streaming"
	StreamStatusCompleted StreamStatus = "completed"
	StreamStatusError     StreamStatus = "error"
	StreamStatusCanceled  StreamStatus = "canceled"
	StreamStatusPaused    StreamStatus = "paused"
)

// DefaultStreamConfig - デフォルト設定
func DefaultStreamConfig() *UnifiedStreamConfig {
	return &UnifiedStreamConfig{
		BufferSize:      4096,
		FlushInterval:   50 * time.Millisecond,
		TokenDelay:      15 * time.Millisecond,
		SentenceDelay:   100 * time.Millisecond,
		ParagraphDelay:  200 * time.Millisecond,
		CodeBlockDelay:  5 * time.Millisecond,
		EnableStreaming: true,
		MaxLineLength:   100,
		MaxWorkers:      4,
		QueueSize:       40,
		Timeout:         30 * time.Second,
	}
}