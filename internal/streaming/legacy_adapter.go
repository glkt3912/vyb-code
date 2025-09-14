package streaming

import (
	"context"
	"io"
	"time"
)

// LegacyStreamAdapter - 既存のストリーミング実装との互換アダプター
type LegacyStreamAdapter struct {
	manager *Manager
}

// NewLegacyStreamAdapter - 新しい互換アダプターを作成
func NewLegacyStreamAdapter(manager *Manager) *LegacyStreamAdapter {
	return &LegacyStreamAdapter{
		manager: manager,
	}
}

// ProcessLLMStream - 既存のLLMストリーミング処理をラップ
// internal/stream/processor.go から移行
func (a *LegacyStreamAdapter) ProcessLLMStream(ctx context.Context, reader io.Reader, output io.Writer) error {
	options := &StreamOptions{
		Type:            StreamTypeLLMResponse,
		EnableInterrupt: true,
		ShowThinking:    false,
		Metadata: map[string]interface{}{
			"legacy_migration": true,
			"source":           "internal/stream/processor.go",
		},
	}

	return a.manager.Process(ctx, reader, output, options)
}

// StreamContent - 既存のUI表示ストリーミング処理をラップ
// internal/streaming/processor.go から移行
func (a *LegacyStreamAdapter) StreamContent(content string) error {
	options := &StreamOptions{
		Type:            StreamTypeUIDisplay,
		EnableInterrupt: false,
		Metadata: map[string]interface{}{
			"legacy_migration": true,
			"source":           "internal/streaming/processor.go",
		},
	}

	ctx := context.Background()
	return a.manager.ProcessString(ctx, content, NewConsoleWriter(), options)
}

// StreamContentInterruptible - 中断可能ストリーミング処理をラップ
func (a *LegacyStreamAdapter) StreamContentInterruptible(content string, interrupt <-chan struct{}) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 中断チャネルを監視
	go func() {
		<-interrupt
		cancel()
	}()

	options := &StreamOptions{
		Type:            StreamTypeInterrupted,
		EnableInterrupt: true,
		Metadata: map[string]interface{}{
			"legacy_migration": true,
			"interruptible":    true,
		},
	}

	return a.manager.ProcessString(ctx, content, NewConsoleWriter(), options)
}

// SetSpeedPreset - 既存の速度プリセット設定をラップ
func (a *LegacyStreamAdapter) SetSpeedPreset(preset string) {
	config := a.manager.config

	switch preset {
	case "instant":
		config.TokenDelay = 0
		config.SentenceDelay = 0
		config.ParagraphDelay = 0
	case "fast":
		config.TokenDelay = 5 * time.Millisecond
		config.SentenceDelay = 50 * time.Millisecond
		config.ParagraphDelay = 100 * time.Millisecond
	case "normal":
		config.TokenDelay = 15 * time.Millisecond
		config.SentenceDelay = 100 * time.Millisecond
		config.ParagraphDelay = 200 * time.Millisecond
	case "slow":
		config.TokenDelay = 50 * time.Millisecond
		config.SentenceDelay = 300 * time.Millisecond
		config.ParagraphDelay = 500 * time.Millisecond
	case "typewriter":
		config.TokenDelay = 100 * time.Millisecond
		config.SentenceDelay = 500 * time.Millisecond
		config.ParagraphDelay = 1000 * time.Millisecond
	}

	a.manager.UpdateConfig(config)
}

// SetStreamingEnabled - ストリーミング有効/無効設定
func (a *LegacyStreamAdapter) SetStreamingEnabled(enabled bool) {
	config := a.manager.config
	config.EnableStreaming = enabled
	a.manager.UpdateConfig(config)
}

// UpdateConfig - 設定更新（既存のStreamConfigから変換）
func (a *LegacyStreamAdapter) UpdateLegacyConfig(legacyConfig map[string]interface{}) {
	config := a.manager.config

	if tokenDelay, ok := legacyConfig["token_delay"].(time.Duration); ok {
		config.TokenDelay = tokenDelay
	}
	if sentenceDelay, ok := legacyConfig["sentence_delay"].(time.Duration); ok {
		config.SentenceDelay = sentenceDelay
	}
	if paragraphDelay, ok := legacyConfig["paragraph_delay"].(time.Duration); ok {
		config.ParagraphDelay = paragraphDelay
	}
	if enabled, ok := legacyConfig["enable_streaming"].(bool); ok {
		config.EnableStreaming = enabled
	}
	if maxLineLength, ok := legacyConfig["max_line_length"].(int); ok {
		config.MaxLineLength = maxLineLength
	}

	a.manager.UpdateConfig(config)
}

// GetState - 現在の状態を取得（互換性用）
func (a *LegacyStreamAdapter) GetState() map[string]interface{} {
	metrics := a.manager.GetGlobalMetrics()

	return map[string]interface{}{
		"active_streams":    metrics.ActiveRequests,
		"total_requests":    metrics.TotalRequests,
		"performance_score": metrics.PerformanceScore,
		"error_count":       metrics.ErrorCount,
		"last_request":      metrics.LastRequestTime,
	}
}

// Reset - 状態をリセット（互換性用）
func (a *LegacyStreamAdapter) Reset() {
	// 新しいマネージャーインスタンスで状態をリセット
	config := a.manager.config
	a.manager = NewManager(config)
}

// ConsoleWriter - コンソール出力用ライター
type ConsoleWriter struct{}

// NewConsoleWriter - 新しいコンソールライターを作成
func NewConsoleWriter() *ConsoleWriter {
	return &ConsoleWriter{}
}

// Write - 標準出力に書き込み
func (w *ConsoleWriter) Write(p []byte) (int, error) {
	content := string(p)
	// ANSIエスケープシーケンス対応
	print(content)
	return len(p), nil
}

// CreateMigrationPlan - 既存コードの移行計画を作成
func CreateMigrationPlan() map[string]string {
	return map[string]string{
		"internal/stream/processor.go": `
// 既存の *stream.Processor を *streaming.Manager + LegacyStreamAdapter で置き換え
// 
// Before:
//   processor := stream.NewProcessor()
//   processor.ProcessLLMStream(ctx, reader, output)
//
// After:
//   manager := streaming.NewManager(streaming.DefaultStreamConfig())
//   adapter := streaming.NewLegacyStreamAdapter(manager)
//   adapter.ProcessLLMStream(ctx, reader, output)
`,
		"internal/streaming/processor.go": `
// 既存の *streaming.Processor を *streaming.Manager + LegacyStreamAdapter で置き換え
//
// Before:
//   processor := streaming.NewProcessor()
//   processor.StreamContent(content)
//
// After:
//   manager := streaming.NewManager(streaming.DefaultStreamConfig())
//   adapter := streaming.NewLegacyStreamAdapter(manager)
//   adapter.StreamContent(content)
`,
		"chat/session.go streamProcessor": `
// chat.Session の streamProcessor フィールドを更新
//
// Before:
//   streamProcessor *streaming.Processor
//
// After:
//   streamingManager *streaming.Manager
//   streamingAdapter *streaming.LegacyStreamAdapter
`,
	}
}
