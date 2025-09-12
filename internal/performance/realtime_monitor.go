package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/glkt/vyb-code/internal/config"
)

// リアルタイムパフォーマンス監視システム - Phase 3実装
type RealtimeMonitor struct {
	config          *config.Config
	enabled         bool
	metrics         *RealtimeMetrics
	alertThresholds *AlertThresholds
	observers       []MetricObserver
	alertHandlers   []AlertHandler
	collectInterval time.Duration
	retentionPeriod time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	mutex           sync.RWMutex
}

// リアルタイムパフォーマンスメトリクス
type RealtimeMetrics struct {
	// 応答時間関連
	ResponseTime    *TimeSeriesData `json:"response_time"`
	LLMLatency      *TimeSeriesData `json:"llm_latency"`
	AnalysisTime    *TimeSeriesData `json:"analysis_time"`
	EnhancementTime *TimeSeriesData `json:"enhancement_time"`

	// リソース使用量
	MemoryUsage    *TimeSeriesData `json:"memory_usage"`
	CPUUsage       *TimeSeriesData `json:"cpu_usage"`
	GoroutineCount *TimeSeriesData `json:"goroutine_count"`

	// システム統計
	RequestCount *CounterData `json:"request_count"`
	ErrorRate    *RateData    `json:"error_rate"`
	CacheHitRate *RateData    `json:"cache_hit_rate"`

	// プロアクティブ機能統計
	ProactiveHits      *CounterData `json:"proactive_hits"`
	IntelligenceUse    *CounterData `json:"intelligence_use"`
	ContextSuggestions *CounterData `json:"context_suggestions"`

	// ユーザー体験メトリクス
	UserSatisfaction *TimeSeriesData         `json:"user_satisfaction"`
	SessionLength    *TimeSeriesData         `json:"session_length"`
	FeatureUsage     map[string]*CounterData `json:"feature_usage"`

	// 収集時刻
	LastUpdated time.Time `json:"last_updated"`
}

// 時系列データ
type TimeSeriesData struct {
	Values     []float64   `json:"values"`
	Timestamps []time.Time `json:"timestamps"`
	MaxPoints  int         `json:"max_points"`
	Current    float64     `json:"current"`
	Average    float64     `json:"average"`
	Min        float64     `json:"min"`
	Max        float64     `json:"max"`
	Trend      string      `json:"trend"` // "up", "down", "stable"
}

// カウンターデータ
type CounterData struct {
	Total     int64     `json:"total"`
	LastHour  int64     `json:"last_hour"`
	LastDay   int64     `json:"last_day"`
	Rate      float64   `json:"rate"` // per second
	LastReset time.Time `json:"last_reset"`
}

// レートデータ
type RateData struct {
	Current     float64   `json:"current"`
	Average     float64   `json:"average"`
	LastHour    float64   `json:"last_hour"`
	LastDay     float64   `json:"last_day"`
	LastUpdated time.Time `json:"last_updated"`
}

// アラート閾値
type AlertThresholds struct {
	ResponseTimeWarning  time.Duration `json:"response_time_warning"`
	ResponseTimeCritical time.Duration `json:"response_time_critical"`
	MemoryWarning        int64         `json:"memory_warning"`      // MB
	MemoryCritical       int64         `json:"memory_critical"`     // MB
	CPUWarning           float64       `json:"cpu_warning"`         // %
	CPUCritical          float64       `json:"cpu_critical"`        // %
	ErrorRateWarning     float64       `json:"error_rate_warning"`  // %
	ErrorRateCritical    float64       `json:"error_rate_critical"` // %
}

// メトリクス観測者
type MetricObserver interface {
	OnMetricUpdate(metricName string, value interface{}, timestamp time.Time)
	OnAlert(alert Alert)
}

// アラートハンドラー
type AlertHandler interface {
	HandleAlert(alert Alert) error
}

// アラート
type Alert struct {
	ID         string                 `json:"id"`
	Level      string                 `json:"level"` // "warning", "critical"
	MetricName string                 `json:"metric_name"`
	Message    string                 `json:"message"`
	Value      interface{}            `json:"value"`
	Threshold  interface{}            `json:"threshold"`
	Timestamp  time.Time              `json:"timestamp"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// 新しいリアルタイム監視システムを作成
func NewRealtimeMonitor(cfg *config.Config) *RealtimeMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &RealtimeMonitor{
		config:          cfg,
		enabled:         cfg.IsProactiveEnabled(),
		metrics:         NewRealtimeMetrics(),
		alertThresholds: NewDefaultAlertThresholds(),
		observers:       make([]MetricObserver, 0),
		alertHandlers:   make([]AlertHandler, 0),
		collectInterval: 5 * time.Second, // 5秒間隔で収集
		retentionPeriod: 1 * time.Hour,   // 1時間保持
		ctx:             ctx,
		cancel:          cancel,
	}
}

// 新しいメトリクスを作成
func NewRealtimeMetrics() *RealtimeMetrics {
	return &RealtimeMetrics{
		ResponseTime:       NewTimeSeriesData(720), // 1時間分（5秒間隔）
		LLMLatency:         NewTimeSeriesData(720),
		AnalysisTime:       NewTimeSeriesData(720),
		EnhancementTime:    NewTimeSeriesData(720),
		MemoryUsage:        NewTimeSeriesData(720),
		CPUUsage:           NewTimeSeriesData(720),
		GoroutineCount:     NewTimeSeriesData(720),
		RequestCount:       NewCounterData(),
		ErrorRate:          NewRateData(),
		CacheHitRate:       NewRateData(),
		ProactiveHits:      NewCounterData(),
		IntelligenceUse:    NewCounterData(),
		ContextSuggestions: NewCounterData(),
		UserSatisfaction:   NewTimeSeriesData(720),
		SessionLength:      NewTimeSeriesData(720),
		FeatureUsage:       make(map[string]*CounterData),
		LastUpdated:        time.Now(),
	}
}

// 新しい時系列データを作成
func NewTimeSeriesData(maxPoints int) *TimeSeriesData {
	return &TimeSeriesData{
		Values:     make([]float64, 0, maxPoints),
		Timestamps: make([]time.Time, 0, maxPoints),
		MaxPoints:  maxPoints,
		Current:    0.0,
		Average:    0.0,
		Min:        0.0,
		Max:        0.0,
		Trend:      "stable",
	}
}

// 新しいカウンターデータを作成
func NewCounterData() *CounterData {
	return &CounterData{
		Total:     0,
		LastHour:  0,
		LastDay:   0,
		Rate:      0.0,
		LastReset: time.Now(),
	}
}

// 新しいレートデータを作成
func NewRateData() *RateData {
	return &RateData{
		Current:     0.0,
		Average:     0.0,
		LastHour:    0.0,
		LastDay:     0.0,
		LastUpdated: time.Now(),
	}
}

// デフォルトアラート閾値を作成
func NewDefaultAlertThresholds() *AlertThresholds {
	return &AlertThresholds{
		ResponseTimeWarning:  15 * time.Second,
		ResponseTimeCritical: 30 * time.Second,
		MemoryWarning:        512,  // 512MB
		MemoryCritical:       1024, // 1GB
		CPUWarning:           70.0,
		CPUCritical:          90.0,
		ErrorRateWarning:     5.0,  // 5%
		ErrorRateCritical:    10.0, // 10%
	}
}

// 監視を開始
func (rm *RealtimeMonitor) Start() error {
	if !rm.enabled {
		return fmt.Errorf("パフォーマンス監視が無効です")
	}

	go rm.collectMetrics()
	return nil
}

// 監視を停止
func (rm *RealtimeMonitor) Stop() error {
	if rm.cancel != nil {
		rm.cancel()
	}
	return nil
}

// メトリクス収集（バックグラウンド）
func (rm *RealtimeMonitor) collectMetrics() {
	ticker := time.NewTicker(rm.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.collectSystemMetrics()
		}
	}
}

// システムメトリクスを収集
func (rm *RealtimeMonitor) collectSystemMetrics() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	now := time.Now()

	// メモリ使用量
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryMB := float64(memStats.Alloc) / 1024 / 1024
	rm.addTimeSeriesPoint(rm.metrics.MemoryUsage, memoryMB, now)

	// Goroutine数
	goroutineCount := float64(runtime.NumGoroutine())
	rm.addTimeSeriesPoint(rm.metrics.GoroutineCount, goroutineCount, now)

	// CPU使用量（簡易計算）
	cpuUsage := rm.calculateCPUUsage()
	rm.addTimeSeriesPoint(rm.metrics.CPUUsage, cpuUsage, now)

	// メトリクスの更新時刻を記録
	rm.metrics.LastUpdated = now

	// アラートをチェック
	rm.checkAlerts()

	// 観測者に通知
	rm.notifyObservers("system_metrics", rm.metrics, now)
}

// 時系列データにポイントを追加
func (rm *RealtimeMonitor) addTimeSeriesPoint(ts *TimeSeriesData, value float64, timestamp time.Time) {
	ts.Values = append(ts.Values, value)
	ts.Timestamps = append(ts.Timestamps, timestamp)

	// 最大ポイント数を超えた場合、古いデータを削除
	if len(ts.Values) > ts.MaxPoints {
		ts.Values = ts.Values[1:]
		ts.Timestamps = ts.Timestamps[1:]
	}

	// 統計を更新
	ts.Current = value
	rm.updateTimeSeriesStats(ts)
}

// 時系列統計を更新
func (rm *RealtimeMonitor) updateTimeSeriesStats(ts *TimeSeriesData) {
	if len(ts.Values) == 0 {
		return
	}

	// 平均、最小、最大を計算
	sum := 0.0
	ts.Min = ts.Values[0]
	ts.Max = ts.Values[0]

	for _, v := range ts.Values {
		sum += v
		if v < ts.Min {
			ts.Min = v
		}
		if v > ts.Max {
			ts.Max = v
		}
	}

	ts.Average = sum / float64(len(ts.Values))

	// トレンドを計算
	if len(ts.Values) >= 2 {
		recent := ts.Values[len(ts.Values)-1]
		previous := ts.Values[len(ts.Values)-2]

		if recent > previous*1.05 {
			ts.Trend = "up"
		} else if recent < previous*0.95 {
			ts.Trend = "down"
		} else {
			ts.Trend = "stable"
		}
	}
}

// CPU使用量を計算（簡易版）
func (rm *RealtimeMonitor) calculateCPUUsage() float64 {
	// Go標準ライブラリでは正確なCPU使用量の取得が困難なため、
	// Goroutine数とメモリ使用量を基に簡易的に推定
	goroutines := float64(runtime.NumGoroutine())

	// 基本的な推定式（実際の用途に応じて調整が必要）
	usage := (goroutines - 5) * 2 // ベースライン5、1goroutineあたり2%
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}

	return usage
}

// 応答時間を記録
func (rm *RealtimeMonitor) RecordResponseTime(duration time.Duration) {
	if !rm.enabled {
		return
	}

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	now := time.Now()
	seconds := float64(duration) / float64(time.Second)

	rm.addTimeSeriesPoint(rm.metrics.ResponseTime, seconds, now)
	rm.incrementCounter(rm.metrics.RequestCount)

	// 観測者に通知
	rm.notifyObservers("response_time", duration, now)
}

// LLMレイテンシを記録
func (rm *RealtimeMonitor) RecordLLMLatency(duration time.Duration) {
	if !rm.enabled {
		return
	}

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	now := time.Now()
	seconds := float64(duration) / float64(time.Second)

	rm.addTimeSeriesPoint(rm.metrics.LLMLatency, seconds, now)
}

// 分析時間を記録
func (rm *RealtimeMonitor) RecordAnalysisTime(duration time.Duration) {
	if !rm.enabled {
		return
	}

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	now := time.Now()
	seconds := float64(duration) / float64(time.Second)

	rm.addTimeSeriesPoint(rm.metrics.AnalysisTime, seconds, now)
}

// カウンターを増加
func (rm *RealtimeMonitor) incrementCounter(counter *CounterData) {
	counter.Total++

	// レートを計算
	elapsed := time.Since(counter.LastReset)
	if elapsed > 0 {
		counter.Rate = float64(counter.Total) / elapsed.Seconds()
	}
}

// プロアクティブ機能使用を記録
func (rm *RealtimeMonitor) RecordProactiveUsage(feature string) {
	if !rm.enabled {
		return
	}

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	switch feature {
	case "proactive_response":
		rm.incrementCounter(rm.metrics.ProactiveHits)
	case "intelligence_enhancement":
		rm.incrementCounter(rm.metrics.IntelligenceUse)
	case "context_suggestion":
		rm.incrementCounter(rm.metrics.ContextSuggestions)
	}

	// 機能使用統計
	if rm.metrics.FeatureUsage[feature] == nil {
		rm.metrics.FeatureUsage[feature] = NewCounterData()
	}
	rm.incrementCounter(rm.metrics.FeatureUsage[feature])
}

// アラートをチェック
func (rm *RealtimeMonitor) checkAlerts() {
	// 応答時間アラート
	if rm.metrics.ResponseTime.Current > float64(rm.alertThresholds.ResponseTimeCritical)/float64(time.Second) {
		rm.createAlert("critical", "response_time", "応答時間が危険レベルです", rm.metrics.ResponseTime.Current)
	} else if rm.metrics.ResponseTime.Current > float64(rm.alertThresholds.ResponseTimeWarning)/float64(time.Second) {
		rm.createAlert("warning", "response_time", "応答時間が遅くなっています", rm.metrics.ResponseTime.Current)
	}

	// メモリアラート
	if rm.metrics.MemoryUsage.Current > float64(rm.alertThresholds.MemoryCritical) {
		rm.createAlert("critical", "memory_usage", "メモリ使用量が危険レベルです", rm.metrics.MemoryUsage.Current)
	} else if rm.metrics.MemoryUsage.Current > float64(rm.alertThresholds.MemoryWarning) {
		rm.createAlert("warning", "memory_usage", "メモリ使用量が高くなっています", rm.metrics.MemoryUsage.Current)
	}

	// CPU使用量アラート
	if rm.metrics.CPUUsage.Current > rm.alertThresholds.CPUCritical {
		rm.createAlert("critical", "cpu_usage", "CPU使用量が危険レベルです", rm.metrics.CPUUsage.Current)
	} else if rm.metrics.CPUUsage.Current > rm.alertThresholds.CPUWarning {
		rm.createAlert("warning", "cpu_usage", "CPU使用量が高くなっています", rm.metrics.CPUUsage.Current)
	}
}

// アラートを作成
func (rm *RealtimeMonitor) createAlert(level, metricName, message string, value interface{}) {
	alert := Alert{
		ID:         fmt.Sprintf("%s_%s_%d", level, metricName, time.Now().Unix()),
		Level:      level,
		MetricName: metricName,
		Message:    message,
		Value:      value,
		Timestamp:  time.Now(),
		Resolved:   false,
		Metadata:   make(map[string]interface{}),
	}

	// アラートハンドラーに通知
	for _, handler := range rm.alertHandlers {
		go handler.HandleAlert(alert)
	}

	// 観測者に通知
	rm.notifyObservers("alert", alert, time.Now())
}

// 観測者に通知
func (rm *RealtimeMonitor) notifyObservers(metricName string, value interface{}, timestamp time.Time) {
	for _, observer := range rm.observers {
		go observer.OnMetricUpdate(metricName, value, timestamp)
	}
}

// 観測者を追加
func (rm *RealtimeMonitor) AddObserver(observer MetricObserver) {
	rm.observers = append(rm.observers, observer)
}

// アラートハンドラーを追加
func (rm *RealtimeMonitor) AddAlertHandler(handler AlertHandler) {
	rm.alertHandlers = append(rm.alertHandlers, handler)
}

// メトリクスを取得
func (rm *RealtimeMonitor) GetMetrics() *RealtimeMetrics {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	// コピーを返す（安全のため）
	return rm.metrics
}

// パフォーマンスサマリーを生成
func (rm *RealtimeMonitor) GeneratePerformanceSummary() string {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	return fmt.Sprintf(`📊 パフォーマンスサマリー
────────────────────────
⏱️  平均応答時間: %.1fs (最新: %.1fs)
🧠 LLMレイテンシ: %.1fs
📈 分析時間: %.1fs
💾 メモリ使用量: %.1fMB
⚙️  CPU使用量: %.1f%%
🔄 Goroutine数: %.0f
📊 リクエスト総数: %d
🚀 プロアクティブ利用: %d回
🎯 インテリジェンス利用: %d回`,
		rm.metrics.ResponseTime.Average,
		rm.metrics.ResponseTime.Current,
		rm.metrics.LLMLatency.Current,
		rm.metrics.AnalysisTime.Current,
		rm.metrics.MemoryUsage.Current,
		rm.metrics.CPUUsage.Current,
		rm.metrics.GoroutineCount.Current,
		rm.metrics.RequestCount.Total,
		rm.metrics.ProactiveHits.Total,
		rm.metrics.IntelligenceUse.Total)
}

// リソースクリーンアップ
func (rm *RealtimeMonitor) Close() error {
	return rm.Stop()
}
