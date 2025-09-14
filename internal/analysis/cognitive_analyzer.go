package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/glkt/vyb-code/internal/ai"
	"github.com/glkt/vyb-code/internal/config"
)

// CognitiveAnalyzer は科学的手法に基づく総合認知分析システム
// Farquhar et al. (2024) の研究に基づくセマンティックエントロピーによる信頼度測定
// 2024年最新NLP研究の統合による創造性・推論深度の動的分析
type CognitiveAnalyzer struct {
	config    *config.Config
	llmClient ai.LLMClient

	// 科学的分析コンポーネント
	entropyCalculator *SemanticEntropyCalculator
	logicalAnalyzer   *LogicalStructureAnalyzer
	creativityScorer  *CreativityScorer

	// 分析履歴とキャッシュ
	analysisHistory  []*CognitiveAnalysisResult
	analysisCache    map[string]*CognitiveAnalysisResult
	lastOptimization time.Time
}

// CognitiveAnalysisResult は認知分析の包括的結果
type CognitiveAnalysisResult struct {
	// 基本情報
	ID             string        `json:"id"`
	Timestamp      time.Time     `json:"timestamp"`
	UserInput      string        `json:"user_input"`
	Response       string        `json:"response"`
	ProcessingTime time.Duration `json:"processing_time"`

	// 科学的測定結果
	Confidence     *ConfidenceResult     `json:"confidence"`
	ReasoningDepth *ReasoningDepthResult `json:"reasoning_depth"`
	Creativity     *CreativityResult     `json:"creativity"`

	// 統合評価
	OverallQuality float64 `json:"overall_quality"`
	TrustScore     float64 `json:"trust_score"`
	InsightLevel   float64 `json:"insight_level"`

	// 動的分析
	ProcessingStrategy string                 `json:"processing_strategy"`
	AnalysisMetadata   map[string]interface{} `json:"analysis_metadata"`
	RecommendedActions []string               `json:"recommended_actions"`
}

// AnalysisRequest は分析リクエスト
type AnalysisRequest struct {
	UserInput       string                 `json:"user_input"`
	Response        string                 `json:"response"`
	Context         map[string]interface{} `json:"context"`
	AnalysisDepth   string                 `json:"analysis_depth"` // "quick", "standard", "deep"
	RequiredMetrics []string               `json:"required_metrics"`
}

// NewCognitiveAnalyzer は新しい認知分析器を作成
func NewCognitiveAnalyzer(cfg *config.Config, llmClient ai.LLMClient) *CognitiveAnalyzer {
	return &CognitiveAnalyzer{
		config:            cfg,
		llmClient:         llmClient,
		entropyCalculator: NewSemanticEntropyCalculator(llmClient),
		logicalAnalyzer:   NewLogicalStructureAnalyzer(llmClient),
		creativityScorer:  NewCreativityScorer(llmClient),
		analysisHistory:   make([]*CognitiveAnalysisResult, 0, 100),
		analysisCache:     make(map[string]*CognitiveAnalysisResult),
		lastOptimization:  time.Now(),
	}
}

// AnalyzeCognitive は包括的認知分析を実行
func (ca *CognitiveAnalyzer) AnalyzeCognitive(
	ctx context.Context,
	request *AnalysisRequest,
) (*CognitiveAnalysisResult, error) {
	startTime := time.Now()

	// キャッシュチェック
	if cached := ca.checkCache(request); cached != nil {
		return cached, nil
	}

	result := &CognitiveAnalysisResult{
		ID:        ca.generateAnalysisID(),
		Timestamp: startTime,
		UserInput: request.UserInput,
		Response:  request.Response,
	}

	// Step 1: セマンティックエントロピーによる信頼度分析
	confidence, err := ca.analyzeConfidence(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("信頼度分析エラー: %w", err)
	}
	result.Confidence = confidence

	// Step 2: 論理構造による推論深度分析
	reasoningDepth, err := ca.analyzeReasoningDepth(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("推論深度分析エラー: %w", err)
	}
	result.ReasoningDepth = reasoningDepth

	// Step 3: 創造性分析
	creativity, err := ca.analyzeCreativity(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("創造性分析エラー: %w", err)
	}
	result.Creativity = creativity

	// Step 4: 統合評価の計算
	ca.calculateIntegratedMetrics(result)

	// Step 5: 動的レコメンデーション
	result.RecommendedActions = ca.generateRecommendations(result)

	// 処理時間の記録
	result.ProcessingTime = time.Since(startTime)

	// キャッシュに保存
	ca.cacheResult(request, result)

	// 履歴に追加
	ca.addToHistory(result)

	return result, nil
}

// analyzeConfidence は信頼度分析を実行
func (ca *CognitiveAnalyzer) analyzeConfidence(
	ctx context.Context,
	request *AnalysisRequest,
) (*ConfidenceResult, error) {

	// 複数応答生成による分析が理想的だが、単一応答でも分析可能
	responses := []string{request.Response}

	// 追加応答生成（深度分析の場合）
	if request.AnalysisDepth == "deep" {
		additionalResponses, err := ca.generateAdditionalResponses(ctx, request.UserInput, 3)
		if err == nil {
			responses = append(responses, additionalResponses...)
		}
	}

	return ca.entropyCalculator.CalculateConfidence(ctx, responses, request.UserInput)
}

// analyzeReasoningDepth は推論深度分析を実行
func (ca *CognitiveAnalyzer) analyzeReasoningDepth(
	ctx context.Context,
	request *AnalysisRequest,
) (*ReasoningDepthResult, error) {

	return ca.logicalAnalyzer.AnalyzeReasoningDepth(
		ctx,
		request.Response,
		request.UserInput,
	)
}

// analyzeCreativity は創造性分析を実行
func (ca *CognitiveAnalyzer) analyzeCreativity(
	ctx context.Context,
	request *AnalysisRequest,
) (*CreativityResult, error) {

	return ca.creativityScorer.MeasureCreativity(
		ctx,
		request.Response,
		request.UserInput,
		request.Context,
	)
}

// calculateIntegratedMetrics は統合メトリクスを計算
func (ca *CognitiveAnalyzer) calculateIntegratedMetrics(result *CognitiveAnalysisResult) {
	// 総合品質スコア（重み付き平均）
	confidenceWeight := 0.4
	reasoningWeight := 0.35
	creativityWeight := 0.25

	confidenceScore := result.Confidence.OverallConfidence
	reasoningScore := float64(result.ReasoningDepth.OverallDepth) / 10.0 // 0-1範囲に正規化
	creativityScore := result.Creativity.OverallScore

	result.OverallQuality = confidenceScore*confidenceWeight +
		reasoningScore*reasoningWeight +
		creativityScore*creativityWeight

	// 信頼スコア（信頼度と論理的一貫性の統合）
	result.TrustScore = (confidenceScore + result.ReasoningDepth.LogicalCoherence) / 2.0

	// 洞察レベル（推論深度と創造性の統合）
	result.InsightLevel = (reasoningScore + creativityScore) / 2.0

	// 処理戦略の決定
	result.ProcessingStrategy = ca.determineProcessingStrategy(result)

	// メタデータの設定
	result.AnalysisMetadata = map[string]interface{}{
		"entropy_type":          "semantic_von_neumann",
		"reasoning_approach":    result.ReasoningDepth.ReasoningPatterns,
		"creativity_dimensions": result.Creativity.DimensionScores,
		"analysis_timestamp":    result.Timestamp,
		"processing_quality":    result.OverallQuality,
	}
}

// generateAdditionalResponses は追加応答を生成（深度分析用）
func (ca *CognitiveAnalyzer) generateAdditionalResponses(
	ctx context.Context,
	query string,
	count int,
) ([]string, error) {

	responses := make([]string, 0, count)

	for i := 0; i < count; i++ {
		prompt := fmt.Sprintf(`
質問: %s

この質問に対して、異なるアプローチで回答してください。
回答バリエーション #%d として、別の視点や方法論で答えてください。
`, query, i+1)

		request := &ai.GenerateRequest{
			Messages: []ai.Message{
				{Role: "user", Content: prompt},
			},
			Temperature: 0.7 + float64(i)*0.1, // 温度を変化させて多様性を確保
		}

		response, err := ca.llmClient.GenerateResponse(ctx, request)
		if err != nil {
			continue // エラー時はスキップ
		}

		responses = append(responses, response.Content)
	}

	return responses, nil
}

// determineProcessingStrategy は処理戦略を決定
func (ca *CognitiveAnalyzer) determineProcessingStrategy(result *CognitiveAnalysisResult) string {
	confidence := result.Confidence.OverallConfidence
	complexity := float64(result.ReasoningDepth.OverallDepth) / 10.0
	creativity := result.Creativity.OverallScore

	if confidence > 0.8 && complexity > 0.7 {
		return "high_confidence_complex"
	} else if creativity > 0.8 && complexity > 0.6 {
		return "creative_exploration"
	} else if confidence < 0.5 || complexity < 0.3 {
		return "need_clarification"
	} else {
		return "balanced_analysis"
	}
}

// generateRecommendations は動的レコメンデーションを生成
func (ca *CognitiveAnalyzer) generateRecommendations(result *CognitiveAnalysisResult) []string {
	recommendations := []string{}

	// 信頼度に基づくレコメンデーション
	if result.Confidence.OverallConfidence < 0.6 {
		recommendations = append(recommendations,
			"信頼度が低いため追加情報や文脈の提供を検討",
			"複数の応答を生成して比較検討することを推奨")
	}

	// 推論深度に基づくレコメンデーション
	if result.ReasoningDepth.OverallDepth < 4 {
		recommendations = append(recommendations,
			"より詳細な分析や段階的な推論を追加",
			"論理的接続詞を使用した構造化された説明を検討")
	}

	// 創造性に基づくレコメンデーション
	if result.Creativity.OverallScore > 0.7 && result.Creativity.Originality > 0.8 {
		recommendations = append(recommendations,
			"創造的な解決策が提示されているため実装可能性の検討を推奨",
			"アイデアの具体化と詳細な計画立案を検討")
	}

	// 総合品質に基づくレコメンデーション
	switch result.ProcessingStrategy {
	case "need_clarification":
		recommendations = append(recommendations,
			"質問の明確化や追加説明が必要",
			"より具体的な要求事項の提供を検討")
	case "creative_exploration":
		recommendations = append(recommendations,
			"創造的アプローチの活用を推奨",
			"複数の選択肢や代替案の検討を推奨")
	}

	return recommendations
}

// Helper functions

func (ca *CognitiveAnalyzer) generateAnalysisID() string {
	return fmt.Sprintf("analysis_%d", time.Now().UnixNano())
}

func (ca *CognitiveAnalyzer) checkCache(request *AnalysisRequest) *CognitiveAnalysisResult {
	cacheKey := ca.generateCacheKey(request)
	if result, exists := ca.analysisCache[cacheKey]; exists {
		// キャッシュの有効性チェック（5分間有効）
		if time.Since(result.Timestamp) < 5*time.Minute {
			return result
		}
		// 期限切れのキャッシュを削除
		delete(ca.analysisCache, cacheKey)
	}
	return nil
}

func (ca *CognitiveAnalyzer) generateCacheKey(request *AnalysisRequest) string {
	return fmt.Sprintf("%s_%s_%s",
		hashString(request.UserInput),
		hashString(request.Response),
		request.AnalysisDepth)
}

func (ca *CognitiveAnalyzer) cacheResult(request *AnalysisRequest, result *CognitiveAnalysisResult) {
	cacheKey := ca.generateCacheKey(request)
	ca.analysisCache[cacheKey] = result

	// キャッシュサイズ制限（最大100エントリ）
	if len(ca.analysisCache) > 100 {
		// 古いエントリを削除
		oldestTime := time.Now()
		oldestKey := ""

		for key, cached := range ca.analysisCache {
			if cached.Timestamp.Before(oldestTime) {
				oldestTime = cached.Timestamp
				oldestKey = key
			}
		}

		if oldestKey != "" {
			delete(ca.analysisCache, oldestKey)
		}
	}
}

func (ca *CognitiveAnalyzer) addToHistory(result *CognitiveAnalysisResult) {
	ca.analysisHistory = append(ca.analysisHistory, result)

	// 履歴サイズ制限（最大100エントリ）
	if len(ca.analysisHistory) > 100 {
		ca.analysisHistory = ca.analysisHistory[1:]
	}
}

// GetAnalysisHistory は分析履歴を取得
func (ca *CognitiveAnalyzer) GetAnalysisHistory(limit int) []*CognitiveAnalysisResult {
	if limit <= 0 || limit > len(ca.analysisHistory) {
		return ca.analysisHistory
	}

	start := len(ca.analysisHistory) - limit
	return ca.analysisHistory[start:]
}

// GetAnalysisMetrics は分析メトリクスを取得
func (ca *CognitiveAnalyzer) GetAnalysisMetrics() map[string]interface{} {
	if len(ca.analysisHistory) == 0 {
		return map[string]interface{}{}
	}

	totalQuality := 0.0
	totalTrust := 0.0
	totalInsight := 0.0
	totalConfidence := 0.0
	totalCreativity := 0.0

	for _, result := range ca.analysisHistory {
		totalQuality += result.OverallQuality
		totalTrust += result.TrustScore
		totalInsight += result.InsightLevel
		totalConfidence += result.Confidence.OverallConfidence
		totalCreativity += result.Creativity.OverallScore
	}

	count := float64(len(ca.analysisHistory))

	return map[string]interface{}{
		"total_analyses":     len(ca.analysisHistory),
		"average_quality":    totalQuality / count,
		"average_trust":      totalTrust / count,
		"average_insight":    totalInsight / count,
		"average_confidence": totalConfidence / count,
		"average_creativity": totalCreativity / count,
		"cache_hit_rate":     float64(len(ca.analysisCache)) / count,
		"last_analysis":      ca.analysisHistory[len(ca.analysisHistory)-1].Timestamp,
	}
}

// 簡易ハッシュ関数
func hashString(s string) string {
	hash := 0
	for _, char := range s {
		hash = hash*31 + int(char)
	}
	return fmt.Sprintf("%x", hash)
}
