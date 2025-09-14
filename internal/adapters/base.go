package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
)

// AdapterType - アダプター種別
type AdapterType string

const (
	AdapterTypeStreaming AdapterType = "streaming"
	AdapterTypeSession   AdapterType = "session"
	AdapterTypeTools     AdapterType = "tools"
	AdapterTypeAnalysis  AdapterType = "analysis"
)

// AdapterInterface - 統一アダプターインターフェース
type AdapterInterface interface {
	// 基本操作
	GetType() AdapterType
	IsUnifiedEnabled() bool

	// 設定
	Configure(config *config.GradualMigrationConfig) error

	// ヘルスチェック
	HealthCheck(ctx context.Context) error

	// メトリクス
	GetMetrics() *AdapterMetrics
}

// AdapterMetrics - アダプターメトリクス
type AdapterMetrics struct {
	AdapterType     AdapterType   `json:"adapter_type"`
	UnifiedEnabled  bool          `json:"unified_enabled"`
	TotalRequests   int64         `json:"total_requests"`
	UnifiedRequests int64         `json:"unified_requests"`
	LegacyRequests  int64         `json:"legacy_requests"`
	SuccessfulRuns  int64         `json:"successful_runs"`
	FailedRuns      int64         `json:"failed_runs"`
	FallbackCount   int64         `json:"fallback_count"`
	AverageLatency  time.Duration `json:"average_latency"`
	LastUsed        time.Time     `json:"last_used"`
}

// BaseAdapter - 基底アダプター実装
type BaseAdapter struct {
	adapterType    AdapterType
	unifiedEnabled bool
	config         *config.GradualMigrationConfig
	log            logger.Logger
	metrics        *AdapterMetrics
}

// NewBaseAdapter - 新しい基底アダプターを作成
func NewBaseAdapter(adapterType AdapterType, log logger.Logger) *BaseAdapter {
	return &BaseAdapter{
		adapterType:    adapterType,
		unifiedEnabled: false,
		log:            log,
		metrics: &AdapterMetrics{
			AdapterType:    adapterType,
			UnifiedEnabled: false,
			LastUsed:       time.Now(),
		},
	}
}

// GetType - アダプター種別を取得
func (ba *BaseAdapter) GetType() AdapterType {
	return ba.adapterType
}

// IsUnifiedEnabled - 統合システム使用状況を取得
func (ba *BaseAdapter) IsUnifiedEnabled() bool {
	return ba.unifiedEnabled
}

// Configure - 設定を適用
func (ba *BaseAdapter) Configure(config *config.GradualMigrationConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	ba.config = config

	// アダプタータイプに応じて統合システム使用状況を設定
	switch ba.adapterType {
	case AdapterTypeStreaming:
		ba.unifiedEnabled = config.UseUnifiedStreaming
	case AdapterTypeSession:
		ba.unifiedEnabled = config.UseUnifiedSession
	case AdapterTypeTools:
		ba.unifiedEnabled = config.UseUnifiedTools
	case AdapterTypeAnalysis:
		ba.unifiedEnabled = config.UseUnifiedAnalysis
	default:
		return fmt.Errorf("unknown adapter type: %s", ba.adapterType)
	}

	ba.metrics.UnifiedEnabled = ba.unifiedEnabled

	if config.LogMigrationInfo {
		ba.log.Info("アダプター設定更新", map[string]interface{}{
			"type":            string(ba.adapterType),
			"unified_enabled": ba.unifiedEnabled,
			"migration_mode":  config.MigrationMode,
		})
	}

	return nil
}

// HealthCheck - 基本ヘルスチェック
func (ba *BaseAdapter) HealthCheck(ctx context.Context) error {
	// 基本的な設定チェック
	if ba.config == nil {
		return fmt.Errorf("adapter not configured")
	}

	return nil
}

// GetMetrics - メトリクスを取得
func (ba *BaseAdapter) GetMetrics() *AdapterMetrics {
	// メトリクスのコピーを返す
	metricsCopy := *ba.metrics
	return &metricsCopy
}

// UpdateMetrics - メトリクスを更新
func (ba *BaseAdapter) UpdateMetrics(success bool, useUnified bool, latency time.Duration) {
	ba.metrics.TotalRequests++
	ba.metrics.LastUsed = time.Now()

	if useUnified {
		ba.metrics.UnifiedRequests++
	} else {
		ba.metrics.LegacyRequests++
	}

	if success {
		ba.metrics.SuccessfulRuns++
	} else {
		ba.metrics.FailedRuns++
	}

	// 平均レイテンシの更新（移動平均）
	if ba.metrics.TotalRequests == 1 {
		ba.metrics.AverageLatency = latency
	} else {
		// 移動平均計算
		alpha := 0.1 // 平滑化係数
		ba.metrics.AverageLatency = time.Duration(
			float64(ba.metrics.AverageLatency)*(1-alpha) + float64(latency)*alpha,
		)
	}
}

// IncrementFallback - フォールバック回数を増加
func (ba *BaseAdapter) IncrementFallback() {
	ba.metrics.FallbackCount++
}

// ShouldUseLegacy - レガシーシステムを使用すべきかを判定
func (ba *BaseAdapter) ShouldUseLegacy() bool {
	if ba.config == nil {
		return true // 設定なしの場合はレガシーを使用
	}

	switch ba.config.MigrationMode {
	case "legacy":
		return true
	case "unified":
		return false
	case "gradual":
		return !ba.unifiedEnabled
	default:
		return true // 不明なモードの場合はレガシーを使用
	}
}

// LogOperation - 操作をログ記録
func (ba *BaseAdapter) LogOperation(operation string, useUnified bool, latency time.Duration, err error) {
	if ba.config == nil || !ba.config.LogMigrationInfo {
		return
	}

	logFields := map[string]interface{}{
		"adapter_type": string(ba.adapterType),
		"operation":    operation,
		"unified":      useUnified,
		"latency_ms":   latency.Milliseconds(),
	}

	if err != nil {
		logFields["error"] = err.Error()
		ba.log.Error("アダプター操作失敗", logFields)
	} else {
		ba.log.Info("アダプター操作成功", logFields)
	}
}
