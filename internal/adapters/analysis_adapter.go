package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/glkt/vyb-code/internal/analysis"
	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
)

// AnalysisAdapter - 分析システムアダプター
type AnalysisAdapter struct {
	*BaseAdapter

	// レガシーシステム
	legacyAnalyzer    analysis.ProjectAnalyzer
	cognitiveAnalyzer *analysis.CognitiveAnalyzer

	// 統合システム
	unifiedAnalyzer *analysis.UnifiedAnalyzer
}

// NewAnalysisAdapter - 新しい分析アダプターを作成
func NewAnalysisAdapter(log logger.Logger) *AnalysisAdapter {
	return &AnalysisAdapter{
		BaseAdapter: NewBaseAdapter(AdapterTypeAnalysis, log),
	}
}

// Configure - 分析アダプターの設定
func (aa *AnalysisAdapter) Configure(config *config.GradualMigrationConfig) error {
	if err := aa.BaseAdapter.Configure(config); err != nil {
		return err
	}

	// レガシーシステムの初期化
	if aa.legacyAnalyzer == nil {
		// デフォルト設定を使用
		analysisConfig := analysis.DefaultAnalysisConfig()
		aa.legacyAnalyzer = analysis.NewProjectAnalyzer(analysisConfig)
	}

	// 認知分析は暫定的に無効化（LLMクライアントが必要なため）
	// if aa.cognitiveAnalyzer == nil {
	//     aa.cognitiveAnalyzer = analysis.NewCognitiveAnalyzer(config, llmClient)
	// }

	// 統合システムの初期化（暫定的に無効化）
	// if aa.unifiedAnalyzer == nil && aa.IsUnifiedEnabled() {
	//     // 統合分析システムは後で実装
	// }

	aa.log.Info("分析アダプター設定完了", map[string]interface{}{
		"unified_enabled": aa.IsUnifiedEnabled(),
		"legacy_ready":    aa.legacyAnalyzer != nil,
	})

	return nil
}

// AnalyzeProject - プロジェクト分析（統一インターフェース）
func (aa *AnalysisAdapter) AnalyzeProject(ctx context.Context, projectPath string) (interface{}, error) {
	startTime := time.Now()

	useUnified := !aa.ShouldUseLegacy()

	var result interface{}
	var err error

	if useUnified && aa.unifiedAnalyzer != nil {
		result, err = aa.analyzeProjectWithUnified(ctx, projectPath)
	} else {
		result, err = aa.analyzeProjectWithLegacy(ctx, projectPath)
	}

	// フォールバック処理
	if err != nil && useUnified && aa.config.EnableFallback {
		aa.IncrementFallback()
		aa.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"project": projectPath,
			"error":   err.Error(),
		})
		result, err = aa.analyzeProjectWithLegacy(ctx, projectPath)
		useUnified = false
	}

	latency := time.Since(startTime)
	aa.UpdateMetrics(err == nil, useUnified, latency)
	aa.LogOperation("AnalyzeProject", useUnified, latency, err)

	return result, err
}

// AnalyzeCognitive - 認知分析（統一インターフェース）
func (aa *AnalysisAdapter) AnalyzeCognitive(ctx context.Context, request interface{}) (interface{}, error) {
	startTime := time.Now()

	useUnified := !aa.ShouldUseLegacy()

	var result interface{}
	var err error

	if useUnified && aa.unifiedAnalyzer != nil {
		result, err = aa.analyzeCognitiveWithUnified(ctx, request)
	} else {
		result, err = aa.analyzeCognitiveWithLegacy(ctx, request)
	}

	// フォールバック処理
	if err != nil && useUnified && aa.config.EnableFallback {
		aa.IncrementFallback()
		aa.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"error": err.Error(),
		})
		result, err = aa.analyzeCognitiveWithLegacy(ctx, request)
		useUnified = false
	}

	latency := time.Since(startTime)
	aa.UpdateMetrics(err == nil, useUnified, latency)
	aa.LogOperation("AnalyzeCognitive", useUnified, latency, err)

	return result, err
}

// AnalyzeComprehensive - 包括的分析（統一インターフェース）
func (aa *AnalysisAdapter) AnalyzeComprehensive(ctx context.Context, projectPath string, cognitiveRequests []interface{}) (interface{}, error) {
	startTime := time.Now()

	useUnified := !aa.ShouldUseLegacy()

	var result interface{}
	var err error

	if useUnified && aa.unifiedAnalyzer != nil {
		result, err = aa.analyzeComprehensiveWithUnified(ctx, projectPath, cognitiveRequests)
	} else {
		result, err = aa.analyzeComprehensiveWithLegacy(ctx, projectPath, cognitiveRequests)
	}

	// フォールバック処理
	if err != nil && useUnified && aa.config.EnableFallback {
		aa.IncrementFallback()
		aa.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"project": projectPath,
			"error":   err.Error(),
		})
		result, err = aa.analyzeComprehensiveWithLegacy(ctx, projectPath, cognitiveRequests)
		useUnified = false
	}

	latency := time.Since(startTime)
	aa.UpdateMetrics(err == nil, useUnified, latency)
	aa.LogOperation("AnalyzeComprehensive", useUnified, latency, err)

	return result, err
}

// HealthCheck - 分析アダプターのヘルスチェック
func (aa *AnalysisAdapter) HealthCheck(ctx context.Context) error {
	if err := aa.BaseAdapter.HealthCheck(ctx); err != nil {
		return err
	}

	// レガシーシステムのヘルスチェック
	if aa.legacyAnalyzer == nil {
		return fmt.Errorf("legacy project analyzer not initialized")
	}

	// 認知分析は暫定的に無効化されているため、スキップ
	// if aa.cognitiveAnalyzer == nil {
	//     return fmt.Errorf("legacy cognitive analyzer not initialized")
	// }

	// 統合システムのヘルスチェック（有効な場合）
	if aa.IsUnifiedEnabled() {
		if aa.unifiedAnalyzer == nil {
			return fmt.Errorf("unified analyzer not initialized")
		}

		// 統合システムの基本機能チェック
		metrics := aa.unifiedAnalyzer.GetAnalysisMetrics()
		if metrics == nil {
			return fmt.Errorf("unified analyzer metrics not available")
		}
	}

	return nil
}

// Internal methods

// analyzeProjectWithUnified - 統合システムでのプロジェクト分析
func (aa *AnalysisAdapter) analyzeProjectWithUnified(ctx context.Context, projectPath string) (interface{}, error) {
	if aa.unifiedAnalyzer == nil {
		return nil, fmt.Errorf("unified analyzer not initialized")
	}

	return aa.unifiedAnalyzer.AnalyzeProject(ctx, projectPath)
}

// analyzeProjectWithLegacy - レガシーシステムでのプロジェクト分析
func (aa *AnalysisAdapter) analyzeProjectWithLegacy(ctx context.Context, projectPath string) (interface{}, error) {
	if aa.legacyAnalyzer == nil {
		return nil, fmt.Errorf("legacy project analyzer not initialized")
	}

	return aa.legacyAnalyzer.AnalyzeProject(projectPath)
}

// analyzeCognitiveWithUnified - 統合システムでの認知分析
func (aa *AnalysisAdapter) analyzeCognitiveWithUnified(ctx context.Context, request interface{}) (interface{}, error) {
	if aa.unifiedAnalyzer == nil {
		return nil, fmt.Errorf("unified analyzer not initialized")
	}

	// requestの型チェックと変換が必要だが、ここでは簡略化
	analysisRequest, ok := request.(*analysis.AnalysisRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for unified analyzer")
	}

	return aa.unifiedAnalyzer.AnalyzeCognitive(ctx, analysisRequest)
}

// analyzeCognitiveWithLegacy - レガシーシステムでの認知分析
func (aa *AnalysisAdapter) analyzeCognitiveWithLegacy(ctx context.Context, request interface{}) (interface{}, error) {
	if aa.cognitiveAnalyzer == nil {
		return nil, fmt.Errorf("legacy cognitive analyzer not initialized")
	}

	// requestの型チェックと変換が必要だが、暫定的にスタブを返す
	// TODO: 実際の認知分析リクエスト型に合わせて実装
	return map[string]interface{}{
		"analysis_type": "cognitive",
		"status":        "legacy_stub",
		"timestamp":     time.Now(),
	}, nil
}

// analyzeComprehensiveWithUnified - 統合システムでの包括的分析
func (aa *AnalysisAdapter) analyzeComprehensiveWithUnified(ctx context.Context, projectPath string, cognitiveRequests []interface{}) (interface{}, error) {
	if aa.unifiedAnalyzer == nil {
		return nil, fmt.Errorf("unified analyzer not initialized")
	}

	// cognitiveRequestsの型変換
	var analysisRequests []*analysis.AnalysisRequest
	for _, req := range cognitiveRequests {
		if analysisReq, ok := req.(*analysis.AnalysisRequest); ok {
			analysisRequests = append(analysisRequests, analysisReq)
		}
	}

	return aa.unifiedAnalyzer.AnalyzeComprehensive(ctx, projectPath, analysisRequests)
}

// analyzeComprehensiveWithLegacy - レガシーシステムでの包括的分析
func (aa *AnalysisAdapter) analyzeComprehensiveWithLegacy(ctx context.Context, projectPath string, cognitiveRequests []interface{}) (interface{}, error) {
	// レガシーシステムでは包括的分析を個別に実行して統合
	projectResult, err := aa.analyzeProjectWithLegacy(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("project analysis failed: %w", err)
	}

	var cognitiveResults []interface{}
	for _, req := range cognitiveRequests {
		cognitiveResult, err := aa.analyzeCognitiveWithLegacy(ctx, req)
		if err != nil {
			aa.log.Warn("認知分析失敗", map[string]interface{}{"error": err})
			continue
		}
		cognitiveResults = append(cognitiveResults, cognitiveResult)
	}

	// 簡易的な包括的結果構造
	comprehensiveResult := map[string]interface{}{
		"project_analysis":   projectResult,
		"cognitive_results":  cognitiveResults,
		"analysis_timestamp": time.Now(),
		"legacy_mode":        true,
	}

	return comprehensiveResult, nil
}

// GetAnalysisMetrics - 分析固有のメトリクスを取得
func (aa *AnalysisAdapter) GetAnalysisMetrics() *AnalysisMetrics {
	baseMetrics := aa.GetMetrics()

	analysisMetrics := &AnalysisMetrics{
		AdapterMetrics:    *baseMetrics,
		UnifiedAnalyzer:   aa.unifiedAnalyzer != nil,
		LegacyAnalyzer:    aa.legacyAnalyzer != nil,
		CognitiveAnalyzer: aa.cognitiveAnalyzer != nil,
	}

	// 統合システムのメトリクス取得
	if aa.unifiedAnalyzer != nil {
		unifiedMetrics := aa.unifiedAnalyzer.GetAnalysisMetrics()
		analysisMetrics.UnifiedAnalysisMetrics = unifiedMetrics
	}

	return analysisMetrics
}

// AnalysisMetrics - 分析アダプター固有のメトリクス
type AnalysisMetrics struct {
	AdapterMetrics
	UnifiedAnalyzer        bool                             `json:"unified_analyzer"`
	LegacyAnalyzer         bool                             `json:"legacy_analyzer"`
	CognitiveAnalyzer      bool                             `json:"cognitive_analyzer"`
	UnifiedAnalysisMetrics *analysis.UnifiedAnalysisMetrics `json:"unified_metrics,omitempty"`
}
