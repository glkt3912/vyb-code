package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/glkt/vyb-code/internal/ai"
)

// NLI (Natural Language Inference) 分析器
// 2つのテキスト間の含意関係 (entailment, contradiction, neutral) を分析

// EntailmentRelation は含意関係の種類
type EntailmentRelation string

const (
	Entailment    EntailmentRelation = "entailment"    // 含意 (A → B)
	Contradiction EntailmentRelation = "contradiction" // 矛盾 (A ↔ ¬B)
	Neutral       EntailmentRelation = "neutral"       // 中立 (A ⊥ B)
)

// EntailmentResult はNLI分析の結果
type EntailmentResult struct {
	PremiseText    string             `json:"premise_text"`
	HypothesisText string             `json:"hypothesis_text"`
	Relation       EntailmentRelation `json:"relation"`
	Confidence     float64            `json:"confidence"`
	Explanation    string             `json:"explanation"`
	LogicalBasis   string             `json:"logical_basis"`
}

// AnalyzeEntailments は複数応答間の含意関係行列を作成
func (nli *NLIAnalyzer) AnalyzeEntailments(ctx context.Context, responses []string) ([][]float64, error) {
	n := len(responses)
	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
	}

	// 各ペアの含意関係を分析
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				matrix[i][j] = 1.0 // 自己含意は1.0
				continue
			}

			result, err := nli.AnalyzeEntailment(ctx, responses[i], responses[j])
			if err != nil {
				// エラー時はニュートラルとして扱う
				matrix[i][j] = 0.5
				continue
			}

			// 含意関係を数値に変換
			switch result.Relation {
			case Entailment:
				matrix[i][j] = result.Confidence
			case Contradiction:
				matrix[i][j] = 1.0 - result.Confidence // 矛盾は負の含意
			case Neutral:
				matrix[i][j] = 0.5 // 中立は0.5
			}
		}
	}

	return matrix, nil
}

// AnalyzeEntailment は2つのテキスト間の含意関係を分析
func (nli *NLIAnalyzer) AnalyzeEntailment(
	ctx context.Context,
	premise string,
	hypothesis string,
) (*EntailmentResult, error) {

	prompt := nli.buildNLIPrompt(premise, hypothesis)

	request := &ai.GenerateRequest{
		Messages: []ai.Message{
			{Role: "system", Content: nli.buildNLISystemPrompt()},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1, // 低温度で一貫性を重視
	}

	response, err := nli.llmClient.GenerateResponse(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("NLI分析リクエストエラー: %w", err)
	}

	// 応答をパース
	result, err := nli.parseNLIResponse(response.Content, premise, hypothesis)
	if err != nil {
		return nil, fmt.Errorf("NLI応答パースエラー: %w", err)
	}

	return result, nil
}

// buildNLIPrompt はNLI分析用のプロンプトを構築
func (nli *NLIAnalyzer) buildNLIPrompt(premise, hypothesis string) string {
	return fmt.Sprintf(`
以下の2つのテキストの論理的関係を分析してください：

【前提】: %s

【仮説】: %s

これらの関係を以下の3つから選択し、理由とともに説明してください：

1. **含意 (entailment)**: 前提が真であれば、仮説も必然的に真である
2. **矛盾 (contradiction)**: 前提が真であれば、仮説は必然的に偽である
3. **中立 (neutral)**: 前提の真偽は仮説の真偽に影響しない

回答は以下のJSON形式で提供してください：
{
  "relation": "entailment|contradiction|neutral",
  "confidence": 0.0-1.0の数値,
  "explanation": "判定理由の詳細説明",
  "logical_basis": "論理的根拠"
}
`, premise, hypothesis)
}

// buildNLISystemPrompt はNLI分析用のシステムプロンプトを構築
func (nli *NLIAnalyzer) buildNLISystemPrompt() string {
	return `あなたは自然言語推論 (Natural Language Inference) の専門家です。

以下の原則に従って分析を行ってください：

1. **厳密な論理分析**: 感情や推測ではなく、論理的構造に基づいて判定
2. **文脈の考慮**: 暗黙の前提や常識的推論も考慮
3. **証拠の重み**: 明示的な証拠と暗示的な証拠を適切に評価
4. **不確実性の認識**: 曖昧な場合は confidence を下げて報告

特に以下の点に注意：
- 部分的含意と完全含意を区別
- 語彙的含意と論理的含意を区別
- 前提の一部否定と全否定を区別
- 条件文や量化文の論理構造を正確に解析`
}

// parseNLIResponse はNLI分析の応答をパース
func (nli *NLIAnalyzer) parseNLIResponse(
	response string,
	premise string,
	hypothesis string,
) (*EntailmentResult, error) {

	// JSON部分を抽出
	jsonStr := nli.extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("JSON形式の応答が見つかりません")
	}

	// JSON構造体
	var parsed struct {
		Relation     string  `json:"relation"`
		Confidence   float64 `json:"confidence"`
		Explanation  string  `json:"explanation"`
		LogicalBasis string  `json:"logical_basis"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("JSON パースエラー: %w", err)
	}

	// 関係の検証
	relation, err := nli.parseRelation(parsed.Relation)
	if err != nil {
		return nil, err
	}

	// 信頼度の検証
	confidence := nli.validateConfidence(parsed.Confidence)

	return &EntailmentResult{
		PremiseText:    premise,
		HypothesisText: hypothesis,
		Relation:       relation,
		Confidence:     confidence,
		Explanation:    parsed.Explanation,
		LogicalBasis:   parsed.LogicalBasis,
	}, nil
}

// extractJSON は応答からJSON部分を抽出
func (nli *NLIAnalyzer) extractJSON(response string) string {
	// JSON開始位置を探索
	start := strings.Index(response, "{")
	if start == -1 {
		return ""
	}

	// JSON終了位置を探索
	depth := 0
	for i := start; i < len(response); i++ {
		char := response[i]
		if char == '{' {
			depth++
		} else if char == '}' {
			depth--
			if depth == 0 {
				return response[start : i+1]
			}
		}
	}

	return ""
}

// parseRelation は関係文字列をEntailmentRelationに変換
func (nli *NLIAnalyzer) parseRelation(relationStr string) (EntailmentRelation, error) {
	normalized := strings.ToLower(strings.TrimSpace(relationStr))

	switch normalized {
	case "entailment", "含意":
		return Entailment, nil
	case "contradiction", "矛盾":
		return Contradiction, nil
	case "neutral", "中立":
		return Neutral, nil
	default:
		return Neutral, fmt.Errorf("不明な関係: %s", relationStr)
	}
}

// validateConfidence は信頼度値を検証・正規化
func (nli *NLIAnalyzer) validateConfidence(confidence float64) float64 {
	// 0.0-1.0の範囲に制限
	if confidence < 0.0 {
		return 0.0
	}
	if confidence > 1.0 {
		return 1.0
	}
	return confidence
}

// analyzeInternalConsistency は内部一貫性を分析
func (sec *SemanticEntropyCalculator) analyzeInternalConsistency(
	ctx context.Context,
	response string,
) (float64, error) {

	prompt := fmt.Sprintf(`
以下のテキストの内部一貫性を分析してください：

【テキスト】: %s

以下の観点から一貫性を評価し、0.0-1.0のスコアで採点してください：

1. **論理的一貫性**: 矛盾する記述がないか
2. **主張の整合性**: 主要な主張が一貫しているか  
3. **証拠の整合性**: 提示された証拠が主張を支持しているか
4. **論調の一貫性**: 文章全体の論調が統一されているか

回答形式：
{
  "consistency_score": 0.0-1.0,
  "logical_issues": ["特定した論理的問題"],
  "contradictions": ["発見した矛盾"],
  "overall_assessment": "総合評価"
}
`, response)

	request := &ai.GenerateRequest{
		Messages: []ai.Message{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
	}

	aiResponse, err := sec.llmClient.GenerateResponse(ctx, request)
	if err != nil {
		return 0.5, err // デフォルト値
	}

	// 簡易パース (完全な実装では詳細なJSONパースが必要)
	score := sec.extractConsistencyScore(aiResponse.Content)
	return score, nil
}

// analyzeLogicalValidity は論理的妥当性を分析
func (sec *SemanticEntropyCalculator) analyzeLogicalValidity(
	ctx context.Context,
	response string,
	originalQuery string,
) (float64, error) {

	prompt := fmt.Sprintf(`
質問に対する回答の論理的妥当性を評価してください：

【質問】: %s
【回答】: %s

以下の基準で評価し、0.0-1.0のスコアで採点してください：

1. **関連性**: 質問に対する直接的な回答になっているか
2. **完全性**: 質問の全側面に対応しているか
3. **論理構造**: 論理的な推論過程が明確か
4. **根拠**: 主張に十分な根拠があるか

回答形式：
{
  "validity_score": 0.0-1.0,
  "relevance": 0.0-1.0,
  "completeness": 0.0-1.0,
  "reasoning": 0.0-1.0,
  "evidence": 0.0-1.0
}
`, originalQuery, response)

	request := &ai.GenerateRequest{
		Messages: []ai.Message{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
	}

	aiResponse, err := sec.llmClient.GenerateResponse(ctx, request)
	if err != nil {
		return 0.5, err // デフォルト値
	}

	// 簡易パース
	score := sec.extractValidityScore(aiResponse.Content)
	return score, nil
}

// Helper methods for parsing scores
func (sec *SemanticEntropyCalculator) extractConsistencyScore(response string) float64 {
	return sec.extractScoreFromJSON(response, "consistency_score")
}

func (sec *SemanticEntropyCalculator) extractValidityScore(response string) float64 {
	return sec.extractScoreFromJSON(response, "validity_score")
}

func (sec *SemanticEntropyCalculator) extractScoreFromJSON(response, fieldName string) float64 {
	// JSON抽出
	jsonStr := strings.TrimSpace(response)
	if !strings.Contains(jsonStr, fieldName) {
		return 0.5 // デフォルト値
	}

	// 簡易的な値抽出 (完全な実装ではJSON parserを使用)
	searchStr := fmt.Sprintf(`"%s":`, fieldName)
	startIdx := strings.Index(jsonStr, searchStr)
	if startIdx == -1 {
		return 0.5
	}

	// コロンの後の数値を探索
	startIdx += len(searchStr)
	for i := startIdx; i < len(jsonStr); i++ {
		char := jsonStr[i]
		if char >= '0' && char <= '9' || char == '.' {
			// 数値の開始位置を特定
			endIdx := i + 1
			for endIdx < len(jsonStr) && (jsonStr[endIdx] >= '0' && jsonStr[endIdx] <= '9' || jsonStr[endIdx] == '.') {
				endIdx++
			}

			if scoreStr := jsonStr[i:endIdx]; scoreStr != "" {
				if score, err := strconv.ParseFloat(scoreStr, 64); err == nil {
					return math.Max(0.0, math.Min(1.0, score)) // 0-1範囲に制限
				}
			}
			break
		}
	}

	return 0.5 // パース失敗時のデフォルト値
}

// calculateReliabilityMetrics は信頼性メトリクスを計算
func (sec *SemanticEntropyCalculator) calculateReliabilityMetrics(
	clusters []*SemanticCluster,
	entailmentMatrix [][]float64,
) *ReliabilityMetrics {

	metrics := &ReliabilityMetrics{}

	if len(clusters) == 0 {
		return metrics
	}

	// クラスタ間距離の計算
	metrics.InterClusterDistance = sec.calculateInterClusterDistance(clusters)

	// クラスタ内凝集度の計算
	metrics.IntraClusterCohesion = sec.calculateIntraClusterCohesion(clusters)

	// 分布エントロピーの計算
	metrics.DistributionEntropy = sec.calculateDistributionEntropy(clusters)

	// キャリブレーション精度の計算
	metrics.CalibrationScore = sec.calculateCalibrationScore(entailmentMatrix)

	return metrics
}

func (sec *SemanticEntropyCalculator) calculateInterClusterDistance(clusters []*SemanticCluster) float64 {
	if len(clusters) < 2 {
		return 0.0
	}

	totalDistance := 0.0
	pairCount := 0

	for i := 0; i < len(clusters); i++ {
		for j := i + 1; j < len(clusters); j++ {
			distance := sec.calculateClusterDistance(clusters[i], clusters[j])
			totalDistance += distance
			pairCount++
		}
	}

	if pairCount == 0 {
		return 0.0
	}

	return totalDistance / float64(pairCount)
}

func (sec *SemanticEntropyCalculator) calculateIntraClusterCohesion(clusters []*SemanticCluster) float64 {
	if len(clusters) == 0 {
		return 0.0
	}

	totalCohesion := 0.0
	for _, cluster := range clusters {
		totalCohesion += cluster.SimilarityScore
	}

	return totalCohesion / float64(len(clusters))
}

func (sec *SemanticEntropyCalculator) calculateDistributionEntropy(clusters []*SemanticCluster) float64 {
	entropy := 0.0
	for _, cluster := range clusters {
		if cluster.Weight > 0 {
			entropy -= cluster.Weight * math.Log(cluster.Weight)
		}
	}
	return entropy
}

func (sec *SemanticEntropyCalculator) calculateCalibrationScore(entailmentMatrix [][]float64) float64 {
	if len(entailmentMatrix) == 0 {
		return 0.0
	}

	// 含意関係の対称性チェック
	asymmetryPenalty := 0.0
	totalPairs := 0

	for i := 0; i < len(entailmentMatrix); i++ {
		for j := 0; j < len(entailmentMatrix[i]); j++ {
			if i != j && j < len(entailmentMatrix) && i < len(entailmentMatrix[j]) {
				diff := math.Abs(entailmentMatrix[i][j] - entailmentMatrix[j][i])
				asymmetryPenalty += diff
				totalPairs++
			}
		}
	}

	if totalPairs == 0 {
		return 1.0
	}

	// 非対称性が少ないほど高いキャリブレーションスコア
	avgAsymmetry := asymmetryPenalty / float64(totalPairs)
	return math.Max(0.0, 1.0-avgAsymmetry)
}

func (sec *SemanticEntropyCalculator) calculateClusterDistance(cluster1, cluster2 *SemanticCluster) float64 {
	// セマンティックベクトル間の距離計算 (簡易版)
	if len(cluster1.SemanticVector) != len(cluster2.SemanticVector) {
		return 1.0 // 異なる次元の場合は最大距離
	}

	sumSquares := 0.0
	for i := 0; i < len(cluster1.SemanticVector); i++ {
		diff := cluster1.SemanticVector[i] - cluster2.SemanticVector[i]
		sumSquares += diff * diff
	}

	return math.Sqrt(sumSquares)
}
