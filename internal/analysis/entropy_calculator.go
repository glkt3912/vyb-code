package analysis

import (
	"math"
)

// EntropyCalculator はvon Neumannエントロピーとセマンティックエントロピーを計算
type EntropyCalculator struct{}

// EntropyResult はエントロピー計算結果
type EntropyResult struct {
	VonNeumannEntropy  float64             `json:"von_neumann_entropy"`
	SemanticEntropy    float64             `json:"semantic_entropy"`
	NormalizedEntropy  float64             `json:"normalized_entropy"`
	EntropyBreakdown   *EntropyBreakdown   `json:"entropy_breakdown"`
	UncertaintyMetrics *UncertaintyMetrics `json:"uncertainty_metrics"`
}

// EntropyBreakdown はエントロピーの詳細分解
type EntropyBreakdown struct {
	ClusterContributions []ClusterEntropy `json:"cluster_contributions"`
	DominantCluster      string           `json:"dominant_cluster"`
	EntropyDistribution  []float64        `json:"entropy_distribution"`
	InformationContent   float64          `json:"information_content"`
}

// ClusterEntropy はクラスタごとのエントロピー寄与
type ClusterEntropy struct {
	ClusterID    string  `json:"cluster_id"`
	Weight       float64 `json:"weight"`
	Contribution float64 `json:"contribution"`
	LocalEntropy float64 `json:"local_entropy"`
}

// UncertaintyMetrics は不確実性評価メトリクス
type UncertaintyMetrics struct {
	EpistemicUncertainty float64    `json:"epistemic_uncertainty"` // 知識不足による不確実性
	AleatoricUncertainty float64    `json:"aleatoric_uncertainty"` // 本質的ランダム性
	TotalUncertainty     float64    `json:"total_uncertainty"`     // 総不確実性
	ConfidenceInterval   [2]float64 `json:"confidence_interval"`   // 信頼区間
	InformationGain      float64    `json:"information_gain"`      // 情報取得量
}

// NewEntropyCalculator は新しいエントロピー計算器を作成
func NewEntropyCalculator() *EntropyCalculator {
	return &EntropyCalculator{}
}

// CalculateVonNeumannEntropy はvon Neumannエントロピーを計算
// 量子情報理論に基づくエントロピー測定により、応答の分散と不確実性を定量化
func (ec *EntropyCalculator) CalculateVonNeumannEntropy(clusters []*SemanticCluster) float64 {
	if len(clusters) == 0 {
		return 0.0
	}

	if len(clusters) == 1 {
		return 0.0 // 単一クラスタはエントロピー0
	}

	// Step 1: 確率分布の正規化
	weights := make([]float64, len(clusters))
	totalWeight := 0.0

	for i, cluster := range clusters {
		weights[i] = cluster.Weight
		totalWeight += cluster.Weight
	}

	// 正規化
	if totalWeight > 0 {
		for i := range weights {
			weights[i] /= totalWeight
		}
	}

	// Step 2: von Neumannエントロピーの計算
	// H(ρ) = -Tr(ρ log ρ) ここでρは密度行列
	entropy := 0.0
	for _, weight := range weights {
		if weight > 0 {
			entropy -= weight * math.Log(weight)
		}
	}

	return entropy
}

// CalculateSemanticEntropy はセマンティックエントロピーを計算
// Farquhar et al. (2024) の手法に基づく意味的不確実性の測定
func (ec *EntropyCalculator) CalculateSemanticEntropy(clusters []*SemanticCluster) float64 {
	if len(clusters) == 0 {
		return 0.0
	}

	// Step 1: セマンティック距離に基づく重み調整
	adjustedWeights := ec.calculateSemanticWeights(clusters)

	// Step 2: セマンティック類似性を考慮したエントロピー計算
	entropy := 0.0
	for i, cluster := range clusters {
		if adjustedWeights[i] > 0 {
			// 類似性による割引を適用
			semanticPenalty := 1.0 - cluster.SimilarityScore
			adjustedWeight := adjustedWeights[i] * (1.0 + semanticPenalty)

			entropy -= adjustedWeight * math.Log(adjustedWeight+1e-10) // 数値安定性のため小さな値を追加
		}
	}

	// Step 3: クラスタ間距離による補正
	distancePenalty := ec.calculateDistancePenalty(clusters)
	entropy *= (1.0 + distancePenalty)

	return entropy
}

// CalculateComprehensiveEntropy は包括的なエントロピー分析を実行
func (ec *EntropyCalculator) CalculateComprehensiveEntropy(clusters []*SemanticCluster) *EntropyResult {
	vonNeumannEntropy := ec.CalculateVonNeumannEntropy(clusters)
	semanticEntropy := ec.CalculateSemanticEntropy(clusters)

	// 正規化エントロピー (0-1範囲)
	maxPossibleEntropy := math.Log(float64(len(clusters)))
	normalizedEntropy := 0.0
	if maxPossibleEntropy > 0 {
		normalizedEntropy = (vonNeumannEntropy + semanticEntropy) / (2 * maxPossibleEntropy)
	}

	// 詳細分析
	breakdown := ec.calculateEntropyBreakdown(clusters, vonNeumannEntropy)
	uncertaintyMetrics := ec.calculateUncertaintyMetrics(clusters, vonNeumannEntropy, semanticEntropy)

	return &EntropyResult{
		VonNeumannEntropy:  vonNeumannEntropy,
		SemanticEntropy:    semanticEntropy,
		NormalizedEntropy:  normalizedEntropy,
		EntropyBreakdown:   breakdown,
		UncertaintyMetrics: uncertaintyMetrics,
	}
}

// calculateSemanticWeights はセマンティック類似性に基づく重み調整
func (ec *EntropyCalculator) calculateSemanticWeights(clusters []*SemanticCluster) []float64 {
	weights := make([]float64, len(clusters))
	totalWeight := 0.0

	for i, cluster := range clusters {
		// 基本重みにセマンティック多様性を考慮
		semanticDiversity := ec.calculateSemanticDiversity(cluster, clusters)
		weights[i] = cluster.Weight * (1.0 + semanticDiversity)
		totalWeight += weights[i]
	}

	// 正規化
	if totalWeight > 0 {
		for i := range weights {
			weights[i] /= totalWeight
		}
	}

	return weights
}

// calculateSemanticDiversity はクラスタのセマンティック多様性を計算
func (ec *EntropyCalculator) calculateSemanticDiversity(
	targetCluster *SemanticCluster,
	allClusters []*SemanticCluster,
) float64 {
	if len(allClusters) <= 1 {
		return 0.0
	}

	totalDistance := 0.0
	count := 0

	for _, otherCluster := range allClusters {
		if otherCluster.ID != targetCluster.ID {
			distance := ec.calculateSemanticDistance(
				targetCluster.SemanticVector,
				otherCluster.SemanticVector,
			)
			totalDistance += distance
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return totalDistance / float64(count)
}

// calculateSemanticDistance はセマンティックベクトル間の距離を計算
func (ec *EntropyCalculator) calculateSemanticDistance(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) {
		return 1.0 // 異なる次元の場合は最大距離
	}

	sumSquares := 0.0
	for i := 0; i < len(vec1); i++ {
		diff := vec1[i] - vec2[i]
		sumSquares += diff * diff
	}

	return math.Sqrt(sumSquares)
}

// calculateDistancePenalty はクラスタ間距離に基づくペナルティを計算
func (ec *EntropyCalculator) calculateDistancePenalty(clusters []*SemanticCluster) float64 {
	if len(clusters) < 2 {
		return 0.0
	}

	totalDistance := 0.0
	pairCount := 0

	for i := 0; i < len(clusters); i++ {
		for j := i + 1; j < len(clusters); j++ {
			distance := ec.calculateSemanticDistance(
				clusters[i].SemanticVector,
				clusters[j].SemanticVector,
			)
			totalDistance += distance
			pairCount++
		}
	}

	if pairCount == 0 {
		return 0.0
	}

	avgDistance := totalDistance / float64(pairCount)

	// 距離が小さいほどペナルティが大きくなる（類似したクラスタが多い場合）
	return math.Max(0.0, 1.0-avgDistance)
}

// calculateEntropyBreakdown はエントロピーの詳細分解を計算
func (ec *EntropyCalculator) calculateEntropyBreakdown(
	clusters []*SemanticCluster,
	totalEntropy float64,
) *EntropyBreakdown {
	clusterContributions := make([]ClusterEntropy, len(clusters))
	var dominantCluster string
	maxWeight := 0.0

	entropyDistribution := make([]float64, len(clusters))

	for i, cluster := range clusters {
		// 各クラスタのエントロピー寄与を計算
		localEntropy := 0.0
		if cluster.Weight > 0 {
			localEntropy = -cluster.Weight * math.Log(cluster.Weight)
		}

		contribution := localEntropy / math.Max(totalEntropy, 1e-10)

		clusterContributions[i] = ClusterEntropy{
			ClusterID:    cluster.ID,
			Weight:       cluster.Weight,
			Contribution: contribution,
			LocalEntropy: localEntropy,
		}

		entropyDistribution[i] = contribution

		if cluster.Weight > maxWeight {
			maxWeight = cluster.Weight
			dominantCluster = cluster.ID
		}
	}

	// 情報内容の計算
	informationContent := ec.calculateInformationContent(clusters)

	return &EntropyBreakdown{
		ClusterContributions: clusterContributions,
		DominantCluster:      dominantCluster,
		EntropyDistribution:  entropyDistribution,
		InformationContent:   informationContent,
	}
}

// calculateInformationContent は情報内容を計算
func (ec *EntropyCalculator) calculateInformationContent(clusters []*SemanticCluster) float64 {
	if len(clusters) == 0 {
		return 0.0
	}

	// Shannon情報量の計算
	totalInformation := 0.0

	for _, cluster := range clusters {
		if cluster.Weight > 0 {
			information := -math.Log(cluster.Weight) * cluster.Weight
			totalInformation += information
		}
	}

	return totalInformation
}

// calculateUncertaintyMetrics は不確実性メトリクスを計算
func (ec *EntropyCalculator) calculateUncertaintyMetrics(
	clusters []*SemanticCluster,
	vonNeumannEntropy, semanticEntropy float64,
) *UncertaintyMetrics {
	// 認識的不確実性 (知識不足による)
	epistemicUncertainty := ec.calculateEpistemicUncertainty(clusters)

	// 偶然的不確実性 (本質的ランダム性)
	aleatoricUncertainty := ec.calculateAleatoricUncertainty(clusters, vonNeumannEntropy)

	// 総不確実性
	totalUncertainty := epistemicUncertainty + aleatoricUncertainty

	// 信頼区間の計算
	confidenceInterval := ec.calculateConfidenceInterval(
		totalUncertainty,
		len(clusters),
	)

	// 情報取得量
	informationGain := ec.calculateInformationGain(vonNeumannEntropy, semanticEntropy)

	return &UncertaintyMetrics{
		EpistemicUncertainty: epistemicUncertainty,
		AleatoricUncertainty: aleatoricUncertainty,
		TotalUncertainty:     totalUncertainty,
		ConfidenceInterval:   confidenceInterval,
		InformationGain:      informationGain,
	}
}

// calculateEpistemicUncertainty は認識的不確実性を計算
func (ec *EntropyCalculator) calculateEpistemicUncertainty(clusters []*SemanticCluster) float64 {
	if len(clusters) == 0 {
		return 0.0
	}

	// クラスタ内類似度の分散に基づく不確実性
	similarities := make([]float64, len(clusters))
	for i, cluster := range clusters {
		similarities[i] = cluster.SimilarityScore
	}

	variance := ec.calculateVariance(similarities)

	// 分散が大きいほど認識的不確実性が高い
	return math.Min(1.0, variance*2.0)
}

// calculateAleatoricUncertainty は偶然的不確実性を計算
func (ec *EntropyCalculator) calculateAleatoricUncertainty(
	clusters []*SemanticCluster,
	entropy float64,
) float64 {
	if len(clusters) <= 1 {
		return 0.0
	}

	// エントロピーに基づく本質的不確実性
	maxEntropy := math.Log(float64(len(clusters)))

	if maxEntropy > 0 {
		return entropy / maxEntropy
	}

	return 0.0
}

// calculateVariance は分散を計算
func (ec *EntropyCalculator) calculateVariance(values []float64) float64 {
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

// calculateConfidenceInterval は信頼区間を計算
func (ec *EntropyCalculator) calculateConfidenceInterval(
	uncertainty float64,
	sampleSize int,
) [2]float64 {
	// 簡易的な信頼区間計算 (95%信頼区間)
	standardError := uncertainty / math.Sqrt(float64(sampleSize))
	margin := 1.96 * standardError // 95%信頼区間

	lowerBound := math.Max(0.0, uncertainty-margin)
	upperBound := math.Min(1.0, uncertainty+margin)

	return [2]float64{lowerBound, upperBound}
}

// calculateInformationGain は情報取得量を計算
func (ec *EntropyCalculator) calculateInformationGain(
	vonNeumannEntropy, semanticEntropy float64,
) float64 {
	// 基準エントロピー（最大エントロピー）からの減少量
	baselineEntropy := math.Log(2.0) // バイナリの最大エントロピー

	totalEntropy := (vonNeumannEntropy + semanticEntropy) / 2.0
	informationGain := math.Max(0.0, baselineEntropy-totalEntropy)

	return informationGain
}

// QuantifyUncertainty はファーカー論文の手法による不確実性定量化
func (ec *EntropyCalculator) QuantifyUncertainty(
	clusters []*SemanticCluster,
	entailmentMatrix [][]float64,
) float64 {
	// Farquhar et al. (2024) "Detecting Hallucinations in Large Language Models Using Semantic Entropy"
	// の手法に基づく不確実性定量化

	if len(clusters) == 0 {
		return 1.0 // 最大不確実性
	}

	if len(clusters) == 1 {
		return 0.0 // 完全確実性
	}

	// Step 1: セマンティッククラスタの確率分布
	probabilities := make([]float64, len(clusters))
	for i, cluster := range clusters {
		probabilities[i] = cluster.Weight
	}

	// Step 2: セマンティックエントロピーの計算
	semanticEntropy := 0.0
	for _, prob := range probabilities {
		if prob > 0 {
			semanticEntropy -= prob * math.Log(prob)
		}
	}

	// Step 3: 含意関係による補正
	entailmentPenalty := ec.calculateEntailmentPenalty(entailmentMatrix)

	// Step 4: 正規化された不確実性スコア
	maxEntropy := math.Log(float64(len(clusters)))
	normalizedUncertainty := semanticEntropy / math.Max(maxEntropy, 1e-10)

	// 含意関係による調整
	finalUncertainty := normalizedUncertainty * (1.0 + entailmentPenalty)

	return math.Max(0.0, math.Min(1.0, finalUncertainty))
}

// calculateEntailmentPenalty は含意関係による不確実性ペナルティを計算
func (ec *EntropyCalculator) calculateEntailmentPenalty(entailmentMatrix [][]float64) float64 {
	if len(entailmentMatrix) == 0 {
		return 0.0
	}

	// 含意関係の非対称性と矛盾を検出
	asymmetryPenalty := 0.0
	contradictionPenalty := 0.0
	totalPairs := 0

	for i := 0; i < len(entailmentMatrix); i++ {
		for j := 0; j < len(entailmentMatrix[i]); j++ {
			if i != j && j < len(entailmentMatrix) && i < len(entailmentMatrix[j]) {
				// 非対称性ペナルティ
				asymmetry := math.Abs(entailmentMatrix[i][j] - entailmentMatrix[j][i])
				asymmetryPenalty += asymmetry

				// 矛盾ペナルティ (両方向で低い含意 = 矛盾の可能性)
				if entailmentMatrix[i][j] < 0.3 && entailmentMatrix[j][i] < 0.3 {
					contradictionPenalty += 0.5
				}

				totalPairs++
			}
		}
	}

	if totalPairs == 0 {
		return 0.0
	}

	avgAsymmetry := asymmetryPenalty / float64(totalPairs)
	avgContradiction := contradictionPenalty / float64(totalPairs)

	return (avgAsymmetry + avgContradiction) / 2.0
}
