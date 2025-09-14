package analysis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/glkt/vyb-code/internal/ai"
	"github.com/glkt/vyb-code/internal/config"
)

// UnifiedAnalysisInterface - 統一分析インターフェース
// プロジェクト分析と認知分析を一元化
type UnifiedAnalysisInterface interface {
	// プロジェクト分析
	AnalyzeProject(ctx context.Context, projectPath string) (*ProjectAnalysis, error)
	AnalyzeFile(ctx context.Context, filePath string) (*FileInfo, error)
	AnalyzeDirectory(ctx context.Context, dirPath string) (*DirectoryInfo, error)

	// 認知分析
	AnalyzeCognitive(ctx context.Context, request *AnalysisRequest) (*CognitiveAnalysisResult, error)

	// 統合分析（プロジェクト + 認知）
	AnalyzeComprehensive(ctx context.Context, projectPath string, cognitiveRequests []*AnalysisRequest) (*ComprehensiveAnalysisResult, error)

	// キャッシュ管理
	GetCachedAnalysis(key string, analysisType AnalysisType) (interface{}, error)
	CacheAnalysis(key string, analysisType AnalysisType, result interface{}) error
	InvalidateCache(key string, analysisType AnalysisType) error

	// メトリクス
	GetAnalysisMetrics() *UnifiedAnalysisMetrics

	// 設定管理
	UpdateConfig(config *UnifiedAnalysisConfig) error
	GetConfig() *UnifiedAnalysisConfig
}

// UnifiedAnalyzer - 統一分析器実装
type UnifiedAnalyzer struct {
	mu sync.RWMutex

	// 設定
	config *UnifiedAnalysisConfig

	// 依存関係
	llmClient ai.LLMClient

	// サブ分析器
	projectAnalyzer   ProjectAnalyzer
	cognitiveAnalyzer *CognitiveAnalyzer

	// 統一キャッシュシステム
	cache map[string]*UnifiedCacheEntry

	// メトリクス
	metrics *UnifiedAnalysisMetrics
}

// UnifiedAnalysisConfig - 統一分析設定
type UnifiedAnalysisConfig struct {
	// 共通設定
	EnableCaching bool          `json:"enable_caching"`
	CacheExpiry   time.Duration `json:"cache_expiry"`
	Timeout       time.Duration `json:"timeout"`

	// プロジェクト分析設定
	ProjectConfig *AnalysisConfig `json:"project_config"`

	// 認知分析設定
	CognitiveDepth   string `json:"cognitive_depth"` // "quick", "standard", "deep"
	EnableConfidence bool   `json:"enable_confidence"`
	EnableReasoning  bool   `json:"enable_reasoning"`
	EnableCreativity bool   `json:"enable_creativity"`

	// パフォーマンス設定
	MaxCacheSize       int `json:"max_cache_size"`
	AnalysisWorkers    int `json:"analysis_workers"`
	ConcurrentAnalyses int `json:"concurrent_analyses"`
}

// UnifiedCacheEntry - 統一キャッシュエントリ
type UnifiedCacheEntry struct {
	Key          string       `json:"key"`
	AnalysisType AnalysisType `json:"analysis_type"`
	Result       interface{}  `json:"result"`
	CachedAt     time.Time    `json:"cached_at"`
	ExpiresAt    time.Time    `json:"expires_at"`
	AccessCount  int          `json:"access_count"`
	LastAccessed time.Time    `json:"last_accessed"`
}

// ComprehensiveAnalysisResult - 包括的分析結果
type ComprehensiveAnalysisResult struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`

	// プロジェクト分析結果
	ProjectAnalysis *ProjectAnalysis `json:"project_analysis"`

	// 認知分析結果
	CognitiveResults []*CognitiveAnalysisResult `json:"cognitive_results"`

	// 統合メトリクス
	OverallScore   float64 `json:"overall_score"`
	QualityIndex   float64 `json:"quality_index"`
	CognitiveIndex float64 `json:"cognitive_index"`

	// 推奨事項
	UnifiedRecommendations []UnifiedRecommendation `json:"unified_recommendations"`

	// メタデータ
	ProcessingTime time.Duration          `json:"processing_time"`
	AnalysisPath   string                 `json:"analysis_path"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// UnifiedRecommendation - 統一推奨事項
type UnifiedRecommendation struct {
	Type        string   `json:"type"`     // "project", "cognitive", "integrated"
	Category    string   `json:"category"` // "security", "quality", "performance", "reasoning"
	Priority    string   `json:"priority"` // "critical", "high", "medium", "low"
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
	Files       []string `json:"files,omitempty"`

	// 影響評価
	Impact     string  `json:"impact"`     // "high", "medium", "low"
	Effort     string  `json:"effort"`     // "high", "medium", "low"
	Confidence float64 `json:"confidence"` // 0.0-1.0
}

// UnifiedAnalysisMetrics - 統一分析メトリクス
type UnifiedAnalysisMetrics struct {
	// 実行統計
	TotalAnalyses         int `json:"total_analyses"`
	ProjectAnalyses       int `json:"project_analyses"`
	CognitiveAnalyses     int `json:"cognitive_analyses"`
	ComprehensiveAnalyses int `json:"comprehensive_analyses"`

	// パフォーマンス統計
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	CacheHitRate          float64       `json:"cache_hit_rate"`
	ErrorRate             float64       `json:"error_rate"`

	// 品質統計
	AverageQualityScore   float64 `json:"average_quality_score"`
	AverageCognitiveScore float64 `json:"average_cognitive_score"`
	AverageConfidence     float64 `json:"average_confidence"`

	// キャッシュ統計
	CacheSize        int     `json:"cache_size"`
	CacheUtilization float64 `json:"cache_utilization"`

	// タイムスタンプ
	LastUpdated time.Time `json:"last_updated"`
}

// NewUnifiedAnalyzer - 新しい統一分析器を作成
func NewUnifiedAnalyzer(cfg *config.Config, llmClient ai.LLMClient) *UnifiedAnalyzer {
	unifiedConfig := DefaultUnifiedAnalysisConfig()

	analyzer := &UnifiedAnalyzer{
		config:    unifiedConfig,
		llmClient: llmClient,
		cache:     make(map[string]*UnifiedCacheEntry),
		metrics:   &UnifiedAnalysisMetrics{LastUpdated: time.Now()},
	}

	// サブ分析器を初期化
	analyzer.projectAnalyzer = NewProjectAnalyzer(unifiedConfig.ProjectConfig)
	analyzer.cognitiveAnalyzer = NewCognitiveAnalyzer(cfg, llmClient)

	return analyzer
}

// AnalyzeProject - プロジェクト分析を実行
func (ua *UnifiedAnalyzer) AnalyzeProject(ctx context.Context, projectPath string) (*ProjectAnalysis, error) {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	startTime := time.Now()

	// キャッシュチェック
	if ua.config.EnableCaching {
		if cached := ua.getCachedResult(projectPath, AnalysisTypeBasic); cached != nil {
			if result, ok := cached.(*ProjectAnalysis); ok {
				ua.updateCacheMetrics(true)
				return result, nil
			}
		}
	}

	// プロジェクト分析を実行
	result, err := ua.projectAnalyzer.AnalyzeProject(projectPath)
	if err != nil {
		ua.updateErrorMetrics()
		return nil, err
	}

	// キャッシュに保存
	if ua.config.EnableCaching {
		ua.cacheResult(projectPath, AnalysisTypeBasic, result)
	}

	// メトリクス更新
	ua.updateAnalysisMetrics(time.Since(startTime), "project")
	ua.updateCacheMetrics(false)

	return result, nil
}

// AnalyzeCognitive - 認知分析を実行
func (ua *UnifiedAnalyzer) AnalyzeCognitive(ctx context.Context, request *AnalysisRequest) (*CognitiveAnalysisResult, error) {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	startTime := time.Now()

	// キャッシュキーを生成
	cacheKey := ua.generateCognitiveKey(request)

	// キャッシュチェック
	if ua.config.EnableCaching {
		if cached := ua.getCachedResult(cacheKey, AnalysisTypeFull); cached != nil {
			if result, ok := cached.(*CognitiveAnalysisResult); ok {
				ua.updateCacheMetrics(true)
				return result, nil
			}
		}
	}

	// 認知分析を実行
	result, err := ua.cognitiveAnalyzer.AnalyzeCognitive(ctx, request)
	if err != nil {
		ua.updateErrorMetrics()
		return nil, err
	}

	// キャッシュに保存
	if ua.config.EnableCaching {
		ua.cacheResult(cacheKey, AnalysisTypeFull, result)
	}

	// メトリクス更新
	ua.updateAnalysisMetrics(time.Since(startTime), "cognitive")
	ua.updateCacheMetrics(false)

	return result, nil
}

// AnalyzeComprehensive - 包括的分析を実行
func (ua *UnifiedAnalyzer) AnalyzeComprehensive(
	ctx context.Context,
	projectPath string,
	cognitiveRequests []*AnalysisRequest,
) (*ComprehensiveAnalysisResult, error) {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	startTime := time.Now()
	result := &ComprehensiveAnalysisResult{
		ID:           ua.generateAnalysisID(),
		Timestamp:    startTime,
		AnalysisPath: projectPath,
	}

	// プロジェクト分析を実行
	projectResult, err := ua.AnalyzeProject(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("プロジェクト分析エラー: %w", err)
	}
	result.ProjectAnalysis = projectResult

	// 認知分析を並列実行
	cognitiveResults := make([]*CognitiveAnalysisResult, len(cognitiveRequests))
	errChan := make(chan error, len(cognitiveRequests))

	for i, request := range cognitiveRequests {
		go func(idx int, req *AnalysisRequest) {
			cogResult, err := ua.AnalyzeCognitive(ctx, req)
			if err != nil {
				errChan <- err
				return
			}
			cognitiveResults[idx] = cogResult
			errChan <- nil
		}(i, request)
	}

	// 結果を収集
	for i := 0; i < len(cognitiveRequests); i++ {
		if err := <-errChan; err != nil {
			return nil, fmt.Errorf("認知分析エラー: %w", err)
		}
	}

	result.CognitiveResults = cognitiveResults

	// 統合メトリクスを計算
	ua.calculateIntegratedMetrics(result)

	// 統合推奨事項を生成
	result.UnifiedRecommendations = ua.generateUnifiedRecommendations(result)

	// メタデータを設定
	result.ProcessingTime = time.Since(startTime)
	result.Metadata = map[string]interface{}{
		"analysis_timestamp": startTime,
		"cognitive_count":    len(cognitiveRequests),
		"cache_utilized":     ua.config.EnableCaching,
	}

	// メトリクス更新
	ua.updateAnalysisMetrics(result.ProcessingTime, "comprehensive")

	return result, nil
}

// DefaultUnifiedAnalysisConfig - デフォルト統一分析設定
func DefaultUnifiedAnalysisConfig() *UnifiedAnalysisConfig {
	return &UnifiedAnalysisConfig{
		EnableCaching:      true,
		CacheExpiry:        30 * time.Minute,
		Timeout:            10 * time.Minute,
		ProjectConfig:      DefaultAnalysisConfig(),
		CognitiveDepth:     "standard",
		EnableConfidence:   true,
		EnableReasoning:    true,
		EnableCreativity:   true,
		MaxCacheSize:       1000,
		AnalysisWorkers:    4,
		ConcurrentAnalyses: 2,
	}
}

// Helper methods

func (ua *UnifiedAnalyzer) generateAnalysisID() string {
	return fmt.Sprintf("unified_%d", time.Now().UnixNano())
}

func (ua *UnifiedAnalyzer) generateCognitiveKey(request *AnalysisRequest) string {
	return fmt.Sprintf("cognitive_%s_%s_%s",
		hashString(request.UserInput),
		hashString(request.Response),
		request.AnalysisDepth)
}

func (ua *UnifiedAnalyzer) getCachedResult(key string, analysisType AnalysisType) interface{} {
	cacheKey := fmt.Sprintf("%s_%d", key, analysisType)
	entry, exists := ua.cache[cacheKey]
	if !exists {
		return nil
	}

	// 期限チェック
	if time.Now().After(entry.ExpiresAt) {
		delete(ua.cache, cacheKey)
		return nil
	}

	// アクセス統計更新
	entry.AccessCount++
	entry.LastAccessed = time.Now()

	return entry.Result
}

func (ua *UnifiedAnalyzer) cacheResult(key string, analysisType AnalysisType, result interface{}) {
	cacheKey := fmt.Sprintf("%s_%d", key, analysisType)

	entry := &UnifiedCacheEntry{
		Key:          cacheKey,
		AnalysisType: analysisType,
		Result:       result,
		CachedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(ua.config.CacheExpiry),
		AccessCount:  0,
		LastAccessed: time.Now(),
	}

	ua.cache[cacheKey] = entry

	// キャッシュサイズ制限
	if len(ua.cache) > ua.config.MaxCacheSize {
		ua.evictOldestCacheEntry()
	}
}

func (ua *UnifiedAnalyzer) evictOldestCacheEntry() {
	oldestTime := time.Now()
	oldestKey := ""

	for key, entry := range ua.cache {
		if entry.LastAccessed.Before(oldestTime) {
			oldestTime = entry.LastAccessed
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(ua.cache, oldestKey)
	}
}

func (ua *UnifiedAnalyzer) updateAnalysisMetrics(processingTime time.Duration, analysisType string) {
	ua.metrics.TotalAnalyses++

	switch analysisType {
	case "project":
		ua.metrics.ProjectAnalyses++
	case "cognitive":
		ua.metrics.CognitiveAnalyses++
	case "comprehensive":
		ua.metrics.ComprehensiveAnalyses++
	}

	// 平均処理時間を更新
	totalTime := ua.metrics.AverageProcessingTime * time.Duration(ua.metrics.TotalAnalyses-1)
	ua.metrics.AverageProcessingTime = (totalTime + processingTime) / time.Duration(ua.metrics.TotalAnalyses)

	ua.metrics.LastUpdated = time.Now()
}

func (ua *UnifiedAnalyzer) updateCacheMetrics(cacheHit bool) {
	if cacheHit {
		// キャッシュヒット率を更新
		totalAccess := float64(ua.metrics.TotalAnalyses)
		currentHits := ua.metrics.CacheHitRate * (totalAccess - 1)
		ua.metrics.CacheHitRate = (currentHits + 1) / totalAccess
	} else {
		totalAccess := float64(ua.metrics.TotalAnalyses)
		currentHits := ua.metrics.CacheHitRate * (totalAccess - 1)
		ua.metrics.CacheHitRate = currentHits / totalAccess
	}

	ua.metrics.CacheSize = len(ua.cache)
	ua.metrics.CacheUtilization = float64(len(ua.cache)) / float64(ua.config.MaxCacheSize)
}

func (ua *UnifiedAnalyzer) updateErrorMetrics() {
	ua.metrics.ErrorRate = (ua.metrics.ErrorRate*float64(ua.metrics.TotalAnalyses) + 1) / float64(ua.metrics.TotalAnalyses+1)
}

func (ua *UnifiedAnalyzer) calculateIntegratedMetrics(result *ComprehensiveAnalysisResult) {
	// プロジェクト品質スコア
	projectScore := 0.0
	if result.ProjectAnalysis != nil && result.ProjectAnalysis.QualityMetrics != nil {
		projectScore = result.ProjectAnalysis.QualityMetrics.Maintainability
	}

	// 認知スコアの平均
	cognitiveScore := 0.0
	if len(result.CognitiveResults) > 0 {
		totalScore := 0.0
		for _, cogResult := range result.CognitiveResults {
			totalScore += cogResult.OverallQuality
		}
		cognitiveScore = totalScore / float64(len(result.CognitiveResults))
	}

	// 統合スコアを計算
	result.QualityIndex = projectScore
	result.CognitiveIndex = cognitiveScore
	result.OverallScore = (projectScore*0.6 + cognitiveScore*0.4) // プロジェクト重視
}

func (ua *UnifiedAnalyzer) generateUnifiedRecommendations(result *ComprehensiveAnalysisResult) []UnifiedRecommendation {
	recommendations := []UnifiedRecommendation{}

	// プロジェクト推奨事項を統合
	if result.ProjectAnalysis != nil {
		for _, rec := range result.ProjectAnalysis.Recommendations {
			unified := UnifiedRecommendation{
				Type:        "project",
				Category:    rec.Type,
				Priority:    rec.Priority,
				Title:       rec.Title,
				Description: rec.Description,
				Actions:     []string{rec.Action},
				Files:       rec.Files,
				Impact:      rec.Impact,
				Effort:      rec.Effort,
				Confidence:  0.8, // デフォルト信頼度
			}
			recommendations = append(recommendations, unified)
		}
	}

	// 認知推奨事項を統合
	for _, cogResult := range result.CognitiveResults {
		for _, action := range cogResult.RecommendedActions {
			unified := UnifiedRecommendation{
				Type:        "cognitive",
				Category:    "reasoning",
				Priority:    "medium",
				Title:       "認知分析推奨",
				Description: action,
				Actions:     []string{action},
				Impact:      "medium",
				Effort:      "low",
				Confidence:  cogResult.TrustScore,
			}
			recommendations = append(recommendations, unified)
		}
	}

	return recommendations
}
