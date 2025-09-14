package analysis

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/glkt/vyb-code/internal/ai"
)

// セマンティックエントロピーによる信頼度測定 (Farquhar et al. 2024)
// 複数の応答をセマンティック類似性でクラスタリングし、
// von Neumann entropyを計算して信頼度を定量化

// SemanticEntropyCalculator はセマンティックエントロピーに基づく信頼度計算を実行
type SemanticEntropyCalculator struct {
	llmClient          ai.LLMClient
	nliAnalyzer        *NLIAnalyzer
	semanticClustering *SemanticClustering
	entropyCalculator  *EntropyCalculator
}

// SemanticCluster は意味的に類似した応答のクラスタ
type SemanticCluster struct {
	ID              string    `json:"id"`
	Responses       []string  `json:"responses"`
	Prototype       string    `json:"prototype"`        // クラスタの代表応答
	SimilarityScore float64   `json:"similarity_score"` // クラスタ内類似度
	Weight          float64   `json:"weight"`           // クラスタの重み (応答数/総数)
	SemanticVector  []float64 `json:"semantic_vector"`  // セマンティック埋め込み
}

// ConfidenceResult はセマンティックエントロピー分析結果
type ConfidenceResult struct {
	OverallConfidence  float64             `json:"overall_confidence"`
	SemanticEntropy    float64             `json:"semantic_entropy"`
	VonNeumannEntropy  float64             `json:"von_neumann_entropy"`
	Clusters           []*SemanticCluster  `json:"clusters"`
	ConsistencyScore   float64             `json:"consistency_score"`
	AgreementLevel     string              `json:"agreement_level"`
	UncertaintyFactors []string            `json:"uncertainty_factors"`
	ReliabilityMetrics *ReliabilityMetrics `json:"reliability_metrics"`
}

// ReliabilityMetrics は信頼性評価の詳細メトリクス
type ReliabilityMetrics struct {
	InterClusterDistance float64 `json:"inter_cluster_distance"` // クラスタ間距離
	IntraClusterCohesion float64 `json:"intra_cluster_cohesion"` // クラスタ内凝集度
	DistributionEntropy  float64 `json:"distribution_entropy"`   // 分布エントロピー
	CalibrationScore     float64 `json:"calibration_score"`      // キャリブレーション精度
}

// NLIAnalyzer は自然言語推論による含意関係分析
type NLIAnalyzer struct {
	llmClient ai.LLMClient
}

// NewSemanticEntropyCalculator は新しいセマンティックエントロピー計算器を作成
func NewSemanticEntropyCalculator(llmClient ai.LLMClient) *SemanticEntropyCalculator {
	return &SemanticEntropyCalculator{
		llmClient:          llmClient,
		nliAnalyzer:        NewNLIAnalyzer(llmClient),
		semanticClustering: NewSemanticClustering(llmClient),
		entropyCalculator:  NewEntropyCalculator(),
	}
}

// CalculateConfidence はセマンティックエントロピーに基づく信頼度を計算
func (sec *SemanticEntropyCalculator) CalculateConfidence(
	ctx context.Context,
	responses []string,
	originalQuery string,
) (*ConfidenceResult, error) {

	if len(responses) < 2 {
		// 単一応答の場合は自己整合性チェックで代替
		return sec.calculateSingleResponseConfidence(ctx, responses[0], originalQuery)
	}

	// Step 1: NLI分析による含意関係の特定
	entailmentMatrix, err := sec.nliAnalyzer.AnalyzeEntailments(ctx, responses)
	if err != nil {
		return nil, fmt.Errorf("NLI分析エラー: %w", err)
	}

	// Step 2: セマンティッククラスタリング
	clusters, err := sec.semanticClustering.ClusterByMeaning(ctx, responses, entailmentMatrix)
	if err != nil {
		return nil, fmt.Errorf("セマンティッククラスタリングエラー: %w", err)
	}

	// Step 3: von Neumann entropy計算
	entropy := sec.entropyCalculator.CalculateVonNeumannEntropy(clusters)
	semanticEntropy := sec.entropyCalculator.CalculateSemanticEntropy(clusters)

	// Step 4: 信頼度計算 (高エントロピー = 低信頼度)
	confidence := sec.calculateConfidenceFromEntropy(entropy, semanticEntropy)

	// Step 5: 詳細メトリクス計算
	reliabilityMetrics := sec.calculateReliabilityMetrics(clusters, entailmentMatrix)
	consistencyScore := sec.calculateConsistencyScore(clusters)
	uncertaintyFactors := sec.identifyUncertaintyFactors(clusters, entailmentMatrix)

	return &ConfidenceResult{
		OverallConfidence:  confidence,
		SemanticEntropy:    semanticEntropy,
		VonNeumannEntropy:  entropy,
		Clusters:           clusters,
		ConsistencyScore:   consistencyScore,
		AgreementLevel:     sec.categorizeAgreementLevel(confidence, len(clusters)),
		UncertaintyFactors: uncertaintyFactors,
		ReliabilityMetrics: reliabilityMetrics,
	}, nil
}

// calculateSingleResponseConfidence は単一応答の信頼度を自己整合性で評価
func (sec *SemanticEntropyCalculator) calculateSingleResponseConfidence(
	ctx context.Context,
	response string,
	originalQuery string,
) (*ConfidenceResult, error) {

	// 内部一貫性チェック
	consistencyScore, err := sec.analyzeInternalConsistency(ctx, response)
	if err != nil {
		return nil, fmt.Errorf("内部一貫性分析エラー: %w", err)
	}

	// 論理的妥当性チェック
	logicalValidityScore, err := sec.analyzeLogicalValidity(ctx, response, originalQuery)
	if err != nil {
		return nil, fmt.Errorf("論理的妥当性分析エラー: %w", err)
	}

	// 複合信頼度計算
	confidence := (consistencyScore + logicalValidityScore) / 2.0

	return &ConfidenceResult{
		OverallConfidence:  confidence,
		SemanticEntropy:    0.0, // 単一応答のためエントロピーなし
		VonNeumannEntropy:  0.0,
		Clusters:           []*SemanticCluster{}, // 単一応答のためクラスタなし
		ConsistencyScore:   consistencyScore,
		AgreementLevel:     sec.categorizeAgreementLevel(confidence, 1),
		UncertaintyFactors: sec.identifySingleResponseUncertaintyFactors(response),
		ReliabilityMetrics: &ReliabilityMetrics{
			CalibrationScore: logicalValidityScore,
		},
	}, nil
}

// calculateConfidenceFromEntropy はエントロピーから信頼度を計算
func (sec *SemanticEntropyCalculator) calculateConfidenceFromEntropy(
	vonNeumannEntropy float64,
	semanticEntropy float64,
) float64 {
	// 正規化されたエントロピー (0-1範囲)
	normalizedVN := math.Min(vonNeumannEntropy/math.Log(10), 1.0) // log(10)で正規化
	normalizedSE := math.Min(semanticEntropy/math.Log(10), 1.0)

	// 複合エントロピー (重み付き平均)
	compositeEntropy := 0.6*normalizedVN + 0.4*normalizedSE

	// 信頼度 = 1 - エントロピー (高エントロピー = 低信頼度)
	confidence := 1.0 - compositeEntropy

	// 0.1-0.99の範囲に制限 (極値を回避)
	return math.Max(0.1, math.Min(0.99, confidence))
}

// calculateConsistencyScore はクラスタの一貫性スコアを計算
func (sec *SemanticEntropyCalculator) calculateConsistencyScore(clusters []*SemanticCluster) float64 {
	if len(clusters) == 0 {
		return 0.0
	}

	// 最大クラスタの重みを一貫性スコアとして使用
	maxWeight := 0.0
	for _, cluster := range clusters {
		if cluster.Weight > maxWeight {
			maxWeight = cluster.Weight
		}
	}

	return maxWeight
}

// categorizeAgreementLevel は信頼度レベルを分類
func (sec *SemanticEntropyCalculator) categorizeAgreementLevel(confidence float64, clusterCount int) string {
	if confidence >= 0.9 && clusterCount <= 2 {
		return "high_agreement"
	} else if confidence >= 0.7 && clusterCount <= 3 {
		return "moderate_agreement"
	} else if confidence >= 0.5 {
		return "low_agreement"
	} else {
		return "disagreement"
	}
}

// identifyUncertaintyFactors は不確実性の要因を特定
func (sec *SemanticEntropyCalculator) identifyUncertaintyFactors(
	clusters []*SemanticCluster,
	entailmentMatrix [][]float64,
) []string {
	factors := []string{}

	// 多数のクラスタ = 意見の分散
	if len(clusters) > 4 {
		factors = append(factors, "high_response_diversity")
	}

	// 均等分布 = 明確な合意なし
	if sec.isUniformDistribution(clusters) {
		factors = append(factors, "uniform_distribution")
	}

	// 低い含意関係 = 論理的一貫性の欠如
	avgEntailment := sec.calculateAverageEntailment(entailmentMatrix)
	if avgEntailment < 0.3 {
		factors = append(factors, "low_logical_consistency")
	}

	// クラスタ内類似度が低い = あいまいなグループ化
	for _, cluster := range clusters {
		if cluster.SimilarityScore < 0.5 {
			factors = append(factors, "ambiguous_clustering")
			break
		}
	}

	return factors
}

// Helper methods

func (sec *SemanticEntropyCalculator) isUniformDistribution(clusters []*SemanticCluster) bool {
	if len(clusters) < 2 {
		return false
	}

	// 重みの分散を計算
	weights := make([]float64, len(clusters))
	for i, cluster := range clusters {
		weights[i] = cluster.Weight
	}

	variance := sec.calculateVariance(weights)
	return variance < 0.05 // 低分散 = 均等分布
}

func (sec *SemanticEntropyCalculator) calculateVariance(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	// 平均計算
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// 分散計算
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}

	return variance / float64(len(values))
}

func (sec *SemanticEntropyCalculator) calculateAverageEntailment(matrix [][]float64) float64 {
	if len(matrix) == 0 || len(matrix[0]) == 0 {
		return 0.0
	}

	total := 0.0
	count := 0

	for i := 0; i < len(matrix); i++ {
		for j := 0; j < len(matrix[i]); j++ {
			if i != j { // 自己との関係は除外
				total += matrix[i][j]
				count++
			}
		}
	}

	if count == 0 {
		return 0.0
	}

	return total / float64(count)
}

func (sec *SemanticEntropyCalculator) identifySingleResponseUncertaintyFactors(response string) []string {
	factors := []string{}

	// 不確実性を示す表現を検出
	uncertainPhrases := []string{
		"たぶん", "おそらく", "可能性がある", "かもしれない",
		"思われる", "と考えられる", "推測される", "不明",
	}

	lowerResponse := strings.ToLower(response)
	for _, phrase := range uncertainPhrases {
		if strings.Contains(lowerResponse, phrase) {
			factors = append(factors, "hedging_language")
			break
		}
	}

	// 短すぎる応答
	if len(strings.Fields(response)) < 10 {
		factors = append(factors, "insufficient_detail")
	}

	// 矛盾する記述の検出 (簡易版)
	if sec.detectContradictions(response) {
		factors = append(factors, "internal_contradictions")
	}

	return factors
}

func (sec *SemanticEntropyCalculator) detectContradictions(text string) bool {
	// 簡易的な矛盾検出 (完全な実装では高度なNLP分析が必要)
	contradictoryPairs := [][]string{
		{"可能", "不可能"},
		{"正しい", "間違い"},
		{"有効", "無効"},
		{"必要", "不要"},
	}

	lowerText := strings.ToLower(text)
	for _, pair := range contradictoryPairs {
		if strings.Contains(lowerText, pair[0]) && strings.Contains(lowerText, pair[1]) {
			return true
		}
	}

	return false
}

// NewNLIAnalyzer は新しいNLI分析器を作成
func NewNLIAnalyzer(llmClient ai.LLMClient) *NLIAnalyzer {
	return &NLIAnalyzer{
		llmClient: llmClient,
	}
}
