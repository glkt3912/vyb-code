package analysis

import (
	"context"
	"fmt"
	"time"
)

// AnalyzeFile - ファイル分析を実行
func (ua *UnifiedAnalyzer) AnalyzeFile(ctx context.Context, filePath string) (*FileInfo, error) {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	return ua.projectAnalyzer.AnalyzeFile(filePath)
}

// AnalyzeDirectory - ディレクトリ分析を実行
func (ua *UnifiedAnalyzer) AnalyzeDirectory(ctx context.Context, dirPath string) (*DirectoryInfo, error) {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	return ua.projectAnalyzer.AnalyzeDirectory(dirPath)
}

// GetCachedAnalysis - キャッシュされた分析結果を取得
func (ua *UnifiedAnalyzer) GetCachedAnalysis(key string, analysisType AnalysisType) (interface{}, error) {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	result := ua.getCachedResult(key, analysisType)
	if result == nil {
		return nil, fmt.Errorf("キャッシュされた分析結果が見つかりません: %s", key)
	}

	return result, nil
}

// CacheAnalysis - 分析結果をキャッシュに保存
func (ua *UnifiedAnalyzer) CacheAnalysis(key string, analysisType AnalysisType, result interface{}) error {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	ua.cacheResult(key, analysisType, result)
	return nil
}

// InvalidateCache - キャッシュを無効化
func (ua *UnifiedAnalyzer) InvalidateCache(key string, analysisType AnalysisType) error {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	cacheKey := fmt.Sprintf("%s_%d", key, analysisType)
	delete(ua.cache, cacheKey)

	return nil
}

// GetAnalysisMetrics - 分析メトリクスを取得
func (ua *UnifiedAnalyzer) GetAnalysisMetrics() *UnifiedAnalysisMetrics {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	// メトリクスのコピーを返す
	metrics := *ua.metrics
	return &metrics
}

// UpdateConfig - 設定を更新
func (ua *UnifiedAnalyzer) UpdateConfig(config *UnifiedAnalysisConfig) error {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	if config == nil {
		return fmt.Errorf("設定がnilです")
	}

	// 設定検証
	if err := ua.validateConfig(config); err != nil {
		return fmt.Errorf("設定検証エラー: %w", err)
	}

	ua.config = config

	// サブ分析器の設定も更新
	if config.ProjectConfig != nil {
		ua.projectAnalyzer = NewProjectAnalyzer(config.ProjectConfig)
	}

	return nil
}

// GetConfig - 設定を取得
func (ua *UnifiedAnalyzer) GetConfig() *UnifiedAnalysisConfig {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	// 設定のコピーを返す
	config := *ua.config
	return &config
}

// ValidateConfig - 設定を検証
func (ua *UnifiedAnalyzer) validateConfig(config *UnifiedAnalysisConfig) error {
	if config.CacheExpiry <= 0 {
		return fmt.Errorf("キャッシュ有効期限は正の値である必要があります")
	}

	if config.Timeout <= 0 {
		return fmt.Errorf("タイムアウトは正の値である必要があります")
	}

	if config.MaxCacheSize <= 0 {
		return fmt.Errorf("最大キャッシュサイズは正の値である必要があります")
	}

	if config.AnalysisWorkers <= 0 {
		return fmt.Errorf("分析ワーカー数は正の値である必要があります")
	}

	if config.ConcurrentAnalyses <= 0 {
		return fmt.Errorf("並行分析数は正の値である必要があります")
	}

	validDepths := map[string]bool{"quick": true, "standard": true, "deep": true}
	if !validDepths[config.CognitiveDepth] {
		return fmt.Errorf("不正な認知分析深度: %s", config.CognitiveDepth)
	}

	return nil
}

// CleanupCache - 期限切れキャッシュをクリーンアップ
func (ua *UnifiedAnalyzer) CleanupCache() int {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	now := time.Now()
	cleanedCount := 0

	for key, entry := range ua.cache {
		if now.After(entry.ExpiresAt) {
			delete(ua.cache, key)
			cleanedCount++
		}
	}

	// メトリクス更新
	ua.metrics.CacheSize = len(ua.cache)
	ua.metrics.CacheUtilization = float64(len(ua.cache)) / float64(ua.config.MaxCacheSize)
	ua.metrics.LastUpdated = time.Now()

	return cleanedCount
}

// GetCacheStatistics - キャッシュ統計を取得
func (ua *UnifiedAnalyzer) GetCacheStatistics() map[string]interface{} {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	// タイプ別統計
	typeStats := make(map[AnalysisType]int)
	totalAccess := 0
	oldestEntry := time.Now()
	newestEntry := time.Time{}

	for _, entry := range ua.cache {
		typeStats[entry.AnalysisType]++
		totalAccess += entry.AccessCount

		if entry.CachedAt.Before(oldestEntry) {
			oldestEntry = entry.CachedAt
		}
		if entry.CachedAt.After(newestEntry) {
			newestEntry = entry.CachedAt
		}
	}

	averageAccess := 0.0
	if len(ua.cache) > 0 {
		averageAccess = float64(totalAccess) / float64(len(ua.cache))
	}

	return map[string]interface{}{
		"cache_size":        len(ua.cache),
		"max_cache_size":    ua.config.MaxCacheSize,
		"utilization_rate":  ua.metrics.CacheUtilization,
		"hit_rate":          ua.metrics.CacheHitRate,
		"type_distribution": typeStats,
		"total_accesses":    totalAccess,
		"average_accesses":  averageAccess,
		"oldest_entry":      oldestEntry,
		"newest_entry":      newestEntry,
		"cache_expiry":      ua.config.CacheExpiry,
	}
}

// PerformanceOptimization - パフォーマンス最適化を実行
func (ua *UnifiedAnalyzer) PerformanceOptimization() *OptimizationResult {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	startTime := time.Now()
	result := &OptimizationResult{
		Timestamp: startTime,
	}

	// 1. キャッシュクリーンアップ
	result.CacheCleanedEntries = ua.cleanupCacheInternal()

	// 2. メトリクス再計算
	ua.recalculateMetrics()

	// 3. 設定最適化提案
	result.ConfigSuggestions = ua.generateConfigOptimizations()

	// 4. メモリ使用量推定
	result.EstimatedMemoryUsage = ua.estimateMemoryUsage()

	result.ProcessingTime = time.Since(startTime)
	result.OptimizationScore = ua.calculateOptimizationScore()

	return result
}

// OptimizationResult - 最適化結果
type OptimizationResult struct {
	Timestamp            time.Time          `json:"timestamp"`
	ProcessingTime       time.Duration      `json:"processing_time"`
	CacheCleanedEntries  int                `json:"cache_cleaned_entries"`
	ConfigSuggestions    []ConfigSuggestion `json:"config_suggestions"`
	EstimatedMemoryUsage int64              `json:"estimated_memory_usage"`
	OptimizationScore    float64            `json:"optimization_score"`
}

// ConfigSuggestion - 設定最適化提案
type ConfigSuggestion struct {
	Parameter      string      `json:"parameter"`
	CurrentValue   interface{} `json:"current_value"`
	SuggestedValue interface{} `json:"suggested_value"`
	Reason         string      `json:"reason"`
	Impact         string      `json:"impact"` // "high", "medium", "low"
}

// Internal helper methods

func (ua *UnifiedAnalyzer) cleanupCacheInternal() int {
	now := time.Now()
	cleanedCount := 0

	for key, entry := range ua.cache {
		if now.After(entry.ExpiresAt) {
			delete(ua.cache, key)
			cleanedCount++
		}
	}

	return cleanedCount
}

func (ua *UnifiedAnalyzer) recalculateMetrics() {
	// キャッシュメトリクスの再計算
	ua.metrics.CacheSize = len(ua.cache)
	ua.metrics.CacheUtilization = float64(len(ua.cache)) / float64(ua.config.MaxCacheSize)
	ua.metrics.LastUpdated = time.Now()
}

func (ua *UnifiedAnalyzer) generateConfigOptimizations() []ConfigSuggestion {
	suggestions := []ConfigSuggestion{}

	// キャッシュヒット率に基づく提案
	if ua.metrics.CacheHitRate < 0.3 {
		suggestions = append(suggestions, ConfigSuggestion{
			Parameter:      "cache_expiry",
			CurrentValue:   ua.config.CacheExpiry,
			SuggestedValue: ua.config.CacheExpiry * 2,
			Reason:         "キャッシュヒット率が低いため、有効期限の延長を推奨",
			Impact:         "medium",
		})
	}

	// キャッシュサイズに基づく提案
	if ua.metrics.CacheUtilization > 0.9 {
		suggestions = append(suggestions, ConfigSuggestion{
			Parameter:      "max_cache_size",
			CurrentValue:   ua.config.MaxCacheSize,
			SuggestedValue: ua.config.MaxCacheSize * 2,
			Reason:         "キャッシュ使用率が高いため、サイズ拡張を推奨",
			Impact:         "high",
		})
	}

	// 処理時間に基づく提案
	if ua.metrics.AverageProcessingTime > 10*time.Second {
		suggestions = append(suggestions, ConfigSuggestion{
			Parameter:      "analysis_workers",
			CurrentValue:   ua.config.AnalysisWorkers,
			SuggestedValue: ua.config.AnalysisWorkers * 2,
			Reason:         "平均処理時間が長いため、ワーカー数の増加を推奨",
			Impact:         "high",
		})
	}

	return suggestions
}

func (ua *UnifiedAnalyzer) estimateMemoryUsage() int64 {
	// 概算メモリ使用量を計算
	cacheMemory := int64(len(ua.cache) * 1024) // 1エントリあたり約1KB
	metricsMemory := int64(1024)               // メトリクス用
	configMemory := int64(512)                 // 設定用

	return cacheMemory + metricsMemory + configMemory
}

func (ua *UnifiedAnalyzer) calculateOptimizationScore() float64 {
	score := 1.0

	// キャッシュヒット率の貢献
	score += ua.metrics.CacheHitRate * 0.3

	// キャッシュ使用率の最適性
	if ua.metrics.CacheUtilization > 0.5 && ua.metrics.CacheUtilization < 0.9 {
		score += 0.2 // 適切な使用率
	}

	// エラー率の影響（負の影響）
	score -= ua.metrics.ErrorRate * 0.5

	// 処理時間の効率性
	if ua.metrics.AverageProcessingTime < 5*time.Second {
		score += 0.2
	}

	// 0.0-1.0の範囲に正規化
	if score > 1.0 {
		score = 1.0
	} else if score < 0.0 {
		score = 0.0
	}

	return score
}

// ResetMetrics - メトリクスをリセット
func (ua *UnifiedAnalyzer) ResetMetrics() {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	ua.metrics = &UnifiedAnalysisMetrics{
		LastUpdated: time.Now(),
	}
}

// GetAnalysisHistory - 分析履歴を取得（認知分析器から）
func (ua *UnifiedAnalyzer) GetAnalysisHistory(limit int) []*CognitiveAnalysisResult {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	if ua.cognitiveAnalyzer != nil {
		return ua.cognitiveAnalyzer.GetAnalysisHistory(limit)
	}

	return []*CognitiveAnalysisResult{}
}
