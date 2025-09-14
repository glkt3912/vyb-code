package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/glkt/vyb-code/internal/adapters"
	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
)

// MigrationMonitor - 移行監視システム
type MigrationMonitor struct {
	config    *config.GradualMigrationConfig
	log       logger.Logger
	validator *Validator

	// アダプター参照
	adapters map[string]adapters.AdapterInterface

	// 監視データ
	systemMetrics map[string]*SystemMetrics
	alerts        []Alert
	mu            sync.RWMutex

	// 監視制御
	running  bool
	stopCh   chan struct{}
	interval time.Duration
}

// SystemMetrics - システムメトリクス
type SystemMetrics struct {
	AdapterName     string                 `json:"adapter_name"`
	Timestamp       time.Time              `json:"timestamp"`
	IsHealthy       bool                   `json:"is_healthy"`
	UnifiedEnabled  bool                   `json:"unified_enabled"`
	LegacyFallbacks int64                  `json:"legacy_fallbacks"`
	SuccessfulOps   int64                  `json:"successful_ops"`
	FailedOps       int64                  `json:"failed_ops"`
	AverageLatency  time.Duration          `json:"average_latency"`
	DetailsMetrics  map[string]interface{} `json:"details_metrics,omitempty"`
}

// Alert - 警告情報
type Alert struct {
	ID        string                 `json:"id"`
	Level     AlertLevel             `json:"level"`
	System    string                 `json:"system"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Resolved  bool                   `json:"resolved"`
}

// AlertLevel - 警告レベル
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelError    AlertLevel = "error"
	AlertLevelCritical AlertLevel = "critical"
)

// NewMigrationMonitor - 新しい移行監視システムを作成
func NewMigrationMonitor(cfg *config.GradualMigrationConfig, log logger.Logger, validator *Validator) *MigrationMonitor {
	return &MigrationMonitor{
		config:        cfg,
		log:           log,
		validator:     validator,
		adapters:      make(map[string]adapters.AdapterInterface),
		systemMetrics: make(map[string]*SystemMetrics),
		alerts:        make([]Alert, 0),
		interval:      time.Duration(30) * time.Second, // デフォルト30秒間隔
		stopCh:        make(chan struct{}),
	}
}

// RegisterAdapter - アダプターを登録
func (mm *MigrationMonitor) RegisterAdapter(name string, adapter adapters.AdapterInterface) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.adapters[name] = adapter
	mm.log.Info("アダプター登録", map[string]interface{}{
		"adapter": name,
		"type":    adapter.GetType(),
	})
}

// StartMonitoring - 監視開始
func (mm *MigrationMonitor) StartMonitoring(ctx context.Context) error {
	mm.mu.Lock()
	if mm.running {
		mm.mu.Unlock()
		return fmt.Errorf("monitor already running")
	}
	mm.running = true
	mm.mu.Unlock()

	mm.log.Info("移行監視開始", map[string]interface{}{
		"interval": mm.interval.String(),
		"adapters": len(mm.adapters),
	})

	go mm.monitoringLoop(ctx)

	return nil
}

// StopMonitoring - 監視停止
func (mm *MigrationMonitor) StopMonitoring() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if !mm.running {
		return
	}

	close(mm.stopCh)
	mm.running = false
	mm.log.Info("移行監視停止", nil)
}

// monitoringLoop - 監視ループ
func (mm *MigrationMonitor) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(mm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			mm.log.Info("監視終了（コンテキストキャンセル）", nil)
			return
		case <-mm.stopCh:
			mm.log.Info("監視終了（手動停止）", nil)
			return
		case <-ticker.C:
			mm.collectMetrics(ctx)
			mm.analyzeMetrics()
			mm.generateAlerts()
		}
	}
}

// collectMetrics - メトリクス収集
func (mm *MigrationMonitor) collectMetrics(ctx context.Context) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for name, adapter := range mm.adapters {
		metrics := mm.collectAdapterMetrics(ctx, name, adapter)
		mm.systemMetrics[name] = metrics
	}
}

// collectAdapterMetrics - 個別アダプターのメトリクス収集
func (mm *MigrationMonitor) collectAdapterMetrics(ctx context.Context, name string, adapter adapters.AdapterInterface) *SystemMetrics {
	startTime := time.Now()

	metrics := &SystemMetrics{
		AdapterName:    name,
		Timestamp:      startTime,
		UnifiedEnabled: adapter.IsUnifiedEnabled(),
		DetailsMetrics: make(map[string]interface{}),
	}

	// ヘルスチェック
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := adapter.HealthCheck(healthCtx); err != nil {
		metrics.IsHealthy = false
		metrics.DetailsMetrics["health_error"] = err.Error()
	} else {
		metrics.IsHealthy = true
	}

	// 基本メトリクス取得
	if baseMetrics := adapter.GetMetrics(); baseMetrics != nil {
		metrics.LegacyFallbacks = baseMetrics.FallbackCount
		metrics.SuccessfulOps = baseMetrics.SuccessfulRuns
		metrics.FailedOps = baseMetrics.FailedRuns
		metrics.AverageLatency = baseMetrics.AverageLatency

		// 詳細メトリクス
		metrics.DetailsMetrics["total_requests"] = baseMetrics.TotalRequests
		metrics.DetailsMetrics["unified_requests"] = baseMetrics.UnifiedRequests
		metrics.DetailsMetrics["legacy_requests"] = baseMetrics.LegacyRequests
	}

	// アダプタータイプ別の詳細メトリクス
	switch adapter.GetType() {
	case adapters.AdapterTypeStreaming:
		if streamingAdapter, ok := adapter.(*adapters.StreamingAdapter); ok {
			streamMetrics := streamingAdapter.GetStreamingMetrics()
			metrics.DetailsMetrics["streaming_metrics"] = streamMetrics
		}
	case adapters.AdapterTypeSession:
		if sessionAdapter, ok := adapter.(*adapters.SessionAdapter); ok {
			sessionMetrics := sessionAdapter.GetSessionMetrics()
			metrics.DetailsMetrics["session_metrics"] = sessionMetrics
		}
	case adapters.AdapterTypeTools:
		if toolsAdapter, ok := adapter.(*adapters.ToolsAdapter); ok {
			toolsMetrics := toolsAdapter.GetToolsMetrics()
			metrics.DetailsMetrics["tools_metrics"] = toolsMetrics
		}
	case adapters.AdapterTypeAnalysis:
		if analysisAdapter, ok := adapter.(*adapters.AnalysisAdapter); ok {
			analysisMetrics := analysisAdapter.GetAnalysisMetrics()
			metrics.DetailsMetrics["analysis_metrics"] = analysisMetrics
		}
	}

	return metrics
}

// analyzeMetrics - メトリクス分析
func (mm *MigrationMonitor) analyzeMetrics() {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	for name, metrics := range mm.systemMetrics {
		// 健全性チェック
		if !metrics.IsHealthy {
			mm.createAlert(AlertLevelError, name, "システムが非健全状態", map[string]interface{}{
				"adapter": name,
				"details": metrics.DetailsMetrics,
			})
		}

		// フォールバック率チェック
		if metrics.SuccessfulOps > 0 {
			fallbackRate := float64(metrics.LegacyFallbacks) / float64(metrics.SuccessfulOps)
			if fallbackRate > 0.3 { // 30%を超えるフォールバック率
				mm.createAlert(AlertLevelWarning, name, "高いフォールバック率を検出", map[string]interface{}{
					"fallback_rate": fallbackRate,
					"fallbacks":     metrics.LegacyFallbacks,
					"total_ops":     metrics.SuccessfulOps,
				})
			}
		}

		// エラー率チェック
		totalOps := metrics.SuccessfulOps + metrics.FailedOps
		if totalOps > 0 {
			errorRate := float64(metrics.FailedOps) / float64(totalOps)
			if errorRate > 0.1 { // 10%を超えるエラー率
				mm.createAlert(AlertLevelError, name, "高いエラー率を検出", map[string]interface{}{
					"error_rate": errorRate,
					"failed_ops": metrics.FailedOps,
					"total_ops":  totalOps,
				})
			}
		}

		// レイテンシーチェック
		if metrics.AverageLatency > time.Second*10 { // 10秒を超える平均レイテンシー
			mm.createAlert(AlertLevelWarning, name, "高いレイテンシーを検出", map[string]interface{}{
				"average_latency": metrics.AverageLatency.String(),
			})
		}
	}
}

// generateAlerts - アラート生成
func (mm *MigrationMonitor) generateAlerts() {
	// 検証システムからの結果も確認
	if mm.validator != nil {
		validationStats := mm.validator.GetValidationStats()

		if validationStats.TotalValidations > 0 && validationStats.SuccessRate < 0.8 { // 80%未満の成功率
			mm.createAlert(AlertLevelWarning, "validation", "検証成功率が低下", map[string]interface{}{
				"success_rate":       validationStats.SuccessRate,
				"total_validations":  validationStats.TotalValidations,
				"failed_validations": validationStats.FailureValidations,
			})
		}
	}
}

// createAlert - アラート作成
func (mm *MigrationMonitor) createAlert(level AlertLevel, system, message string, data map[string]interface{}) {
	alert := Alert{
		ID:        fmt.Sprintf("%s_%s_%d", system, level, time.Now().UnixNano()),
		Level:     level,
		System:    system,
		Message:   message,
		Timestamp: time.Now(),
		Data:      data,
		Resolved:  false,
	}

	mm.alerts = append(mm.alerts, alert)

	// ログ出力
	logData := map[string]interface{}{
		"alert_id": alert.ID,
		"level":    alert.Level,
		"system":   alert.System,
		"message":  alert.Message,
	}

	if data != nil {
		logData["data"] = data
	}

	switch level {
	case AlertLevelCritical, AlertLevelError:
		mm.log.Error("移行監視アラート", logData)
	case AlertLevelWarning:
		mm.log.Warn("移行監視警告", logData)
	default:
		mm.log.Info("移行監視情報", logData)
	}
}

// GetSystemMetrics - システムメトリクス取得
func (mm *MigrationMonitor) GetSystemMetrics(systemName string) *SystemMetrics {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	return mm.systemMetrics[systemName]
}

// GetAllSystemMetrics - 全システムメトリクス取得
func (mm *MigrationMonitor) GetAllSystemMetrics() map[string]*SystemMetrics {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	metrics := make(map[string]*SystemMetrics)
	for k, v := range mm.systemMetrics {
		metrics[k] = v
	}

	return metrics
}

// GetAlerts - アラート取得
func (mm *MigrationMonitor) GetAlerts(includeResolved bool) []Alert {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var alerts []Alert
	for _, alert := range mm.alerts {
		if includeResolved || !alert.Resolved {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// ResolveAlert - アラート解決
func (mm *MigrationMonitor) ResolveAlert(alertID string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for i, alert := range mm.alerts {
		if alert.ID == alertID {
			mm.alerts[i].Resolved = true
			mm.log.Info("アラート解決", map[string]interface{}{
				"alert_id": alertID,
				"system":   alert.System,
			})
			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", alertID)
}

// GetMonitoringStatus - 監視ステータス取得
func (mm *MigrationMonitor) GetMonitoringStatus() *MonitoringStatus {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	status := &MonitoringStatus{
		Running:            mm.running,
		RegisteredAdapters: len(mm.adapters),
		MonitoringInterval: mm.interval,
		LastUpdate:         time.Now(),
	}

	// アクティブアラート数
	for _, alert := range mm.alerts {
		if !alert.Resolved {
			status.ActiveAlerts++
		}
	}

	// システム健全性サマリー
	healthy, total := 0, 0
	for _, metrics := range mm.systemMetrics {
		total++
		if metrics.IsHealthy {
			healthy++
		}
	}

	if total > 0 {
		status.SystemHealthRatio = float64(healthy) / float64(total)
	}

	return status
}

// ExportMetrics - メトリクスエクスポート
func (mm *MigrationMonitor) ExportMetrics() ([]byte, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	export := map[string]interface{}{
		"timestamp":         time.Now(),
		"system_metrics":    mm.systemMetrics,
		"alerts":            mm.alerts,
		"monitoring_status": mm.GetMonitoringStatus(),
	}

	if mm.validator != nil {
		export["validation_stats"] = mm.validator.GetValidationStats()
		export["validation_results"] = mm.validator.GetAllValidationResults()
	}

	return json.MarshalIndent(export, "", "  ")
}

// MonitoringStatus - 監視ステータス
type MonitoringStatus struct {
	Running            bool          `json:"running"`
	RegisteredAdapters int           `json:"registered_adapters"`
	MonitoringInterval time.Duration `json:"monitoring_interval"`
	ActiveAlerts       int           `json:"active_alerts"`
	SystemHealthRatio  float64       `json:"system_health_ratio"`
	LastUpdate         time.Time     `json:"last_update"`
}
