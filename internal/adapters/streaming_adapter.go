package adapters

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
	"github.com/glkt/vyb-code/internal/stream"
	"github.com/glkt/vyb-code/internal/streaming"
)

// StreamingAdapter - ストリーミングシステムアダプター
type StreamingAdapter struct {
	*BaseAdapter

	// レガシーシステム
	legacyProcessor *stream.Processor

	// 統合システム
	unifiedManager *streaming.Manager
}

// NewStreamingAdapter - 新しいストリーミングアダプターを作成
func NewStreamingAdapter(log logger.Logger) *StreamingAdapter {
	return &StreamingAdapter{
		BaseAdapter: NewBaseAdapter(AdapterTypeStreaming, log),
	}
}

// Configure - ストリーミングアダプターの設定
func (sa *StreamingAdapter) Configure(config *config.GradualMigrationConfig) error {
	if err := sa.BaseAdapter.Configure(config); err != nil {
		return err
	}

	// レガシーシステムの初期化
	if sa.legacyProcessor == nil {
		sa.legacyProcessor = stream.NewProcessor()
	}

	// 統合システムの初期化（暫定的に無効化）
	// if sa.unifiedManager == nil && sa.IsUnifiedEnabled() {
	//     // 統合システムは後で実装
	// }

	sa.log.Info("ストリーミングアダプター設定完了", map[string]interface{}{
		"unified_enabled": sa.IsUnifiedEnabled(),
		"legacy_ready":    sa.legacyProcessor != nil,
	})

	return nil
}

// ProcessStream - ストリーム処理（統一インターフェース）
func (sa *StreamingAdapter) ProcessStream(ctx context.Context, input io.Reader, output io.Writer) error {
	startTime := time.Now()

	useUnified := !sa.ShouldUseLegacy()

	var err error
	if useUnified && sa.unifiedManager != nil {
		err = sa.processWithUnified(ctx, input, output)
	} else {
		err = sa.processWithLegacy(ctx, input, output)
	}

	// フォールバック処理
	if err != nil && useUnified && sa.config.EnableFallback {
		sa.IncrementFallback()
		sa.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"error": err.Error(),
		})
		err = sa.processWithLegacy(ctx, input, output)
		useUnified = false
	}

	latency := time.Since(startTime)
	sa.UpdateMetrics(err == nil, useUnified, latency)
	sa.LogOperation("ProcessStream", useUnified, latency, err)

	return err
}

// ProcessString - 文字列処理（統一インターフェース）
func (sa *StreamingAdapter) ProcessString(ctx context.Context, content string, output io.Writer) error {
	startTime := time.Now()

	useUnified := !sa.ShouldUseLegacy()

	var err error
	if useUnified && sa.unifiedManager != nil {
		err = sa.processStringWithUnified(ctx, content, output)
	} else {
		err = sa.processStringWithLegacy(ctx, content, output)
	}

	// フォールバック処理
	if err != nil && useUnified && sa.config.EnableFallback {
		sa.IncrementFallback()
		sa.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"error": err.Error(),
		})
		err = sa.processStringWithLegacy(ctx, content, output)
		useUnified = false
	}

	latency := time.Since(startTime)
	sa.UpdateMetrics(err == nil, useUnified, latency)
	sa.LogOperation("ProcessString", useUnified, latency, err)

	return err
}

// HealthCheck - ストリーミングアダプターのヘルスチェック
func (sa *StreamingAdapter) HealthCheck(ctx context.Context) error {
	if err := sa.BaseAdapter.HealthCheck(ctx); err != nil {
		return err
	}

	// レガシーシステムのヘルスチェック
	if sa.legacyProcessor == nil {
		return fmt.Errorf("legacy processor not initialized")
	}

	// 統合システムのヘルスチェック（有効な場合）
	if sa.IsUnifiedEnabled() {
		if sa.unifiedManager == nil {
			return fmt.Errorf("unified manager not initialized")
		}

		// 統合システムの基本機能チェック（HealthCheckメソッドが存在しない場合は基本チェックのみ）
		// TODO: 実際のヘルスチェックメソッドが実装されたら有効化
		// if err := sa.unifiedManager.HealthCheck(ctx); err != nil {
		//     return fmt.Errorf("unified streaming health check failed: %w", err)
		// }
	}

	return nil
}

// Internal methods

// processWithUnified - 統合システムでの処理
func (sa *StreamingAdapter) processWithUnified(ctx context.Context, input io.Reader, output io.Writer) error {
	if sa.unifiedManager == nil {
		return fmt.Errorf("unified streaming manager not initialized")
	}

	// TODO: 実際のstreaming.Managerのメソッドに合わせて実装
	// 暫定的にエラーを返す
	return fmt.Errorf("unified streaming processing not yet implemented")
}

// processWithLegacy - レガシーシステムでの処理
func (sa *StreamingAdapter) processWithLegacy(ctx context.Context, input io.Reader, output io.Writer) error {
	if sa.legacyProcessor == nil {
		return fmt.Errorf("legacy processor not initialized")
	}

	// TODO: stream.Processorの実際のProcessメソッドに合わせて実装
	// return sa.legacyProcessor.Process(ctx, input, output)
	return fmt.Errorf("legacy stream processing not yet implemented")
}

// processStringWithUnified - 統合システムでの文字列処理
func (sa *StreamingAdapter) processStringWithUnified(ctx context.Context, content string, output io.Writer) error {
	if sa.unifiedManager == nil {
		return fmt.Errorf("unified streaming manager not initialized")
	}

	// TODO: 実際のstreaming.StreamOptionsとメソッドに合わせて実装
	// options := &streaming.StreamOptions{
	//     BufferSize:    4096,
	//     EnableMetrics: sa.config.EnableMetrics,
	//     Timeout:       time.Duration(sa.config.ValidationTimeout) * time.Second,
	// }
	//
	// processor := sa.unifiedManager.GetLLMProcessor()
	// if processor == nil {
	//     return fmt.Errorf("LLM processor not available")
	// }
	//
	// return processor.ProcessString(ctx, content, output, options)
	return fmt.Errorf("unified string processing not yet implemented")
}

// processStringWithLegacy - レガシーシステムでの文字列処理
func (sa *StreamingAdapter) processStringWithLegacy(ctx context.Context, content string, output io.Writer) error {
	if sa.legacyProcessor == nil {
		return fmt.Errorf("legacy processor not initialized")
	}

	// TODO: stream.Processorの実際のProcessStringメソッドに合わせて実装
	// return sa.legacyProcessor.ProcessString(ctx, content, output)
	return fmt.Errorf("legacy string processing not yet implemented")
}

// GetStreamingMetrics - ストリーミング固有のメトリクスを取得
func (sa *StreamingAdapter) GetStreamingMetrics() *StreamingMetrics {
	baseMetrics := sa.GetMetrics()

	streamingMetrics := &StreamingMetrics{
		AdapterMetrics:  *baseMetrics,
		UnifiedManager:  sa.unifiedManager != nil,
		LegacyProcessor: sa.legacyProcessor != nil,
	}

	// 統合システムのメトリクス取得（実際のAPIに合わせて修正が必要）
	if sa.unifiedManager != nil {
		// TODO: 実際のstreaming.ManagerのGetMetricsメソッドに合わせて実装
		// unifiedMetrics := sa.unifiedManager.GetMetrics()
		// streamingMetrics.UnifiedStreamMetrics = unifiedMetrics
	}

	return streamingMetrics
}

// StreamingMetrics - ストリーミングアダプター固有のメトリクス
type StreamingMetrics struct {
	AdapterMetrics
	UnifiedManager       bool                     `json:"unified_manager"`
	LegacyProcessor      bool                     `json:"legacy_processor"`
	UnifiedStreamMetrics *streaming.StreamMetrics `json:"unified_metrics,omitempty"`
}
