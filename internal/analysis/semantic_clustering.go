package analysis

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/glkt/vyb-code/internal/ai"
)

// SemanticClustering はセマンティック類似性に基づく応答クラスタリング
type SemanticClustering struct {
	llmClient ai.LLMClient
}

// ClusteringResult はクラスタリング結果
type ClusteringResult struct {
	Clusters        []*SemanticCluster `json:"clusters"`
	SilhouetteScore float64            `json:"silhouette_score"`
	OptimalK        int                `json:"optimal_k"`
	DistanceMatrix  [][]float64        `json:"distance_matrix"`
}

// NewSemanticClustering は新しいセマンティッククラスタリングを作成
func NewSemanticClustering(llmClient ai.LLMClient) *SemanticClustering {
	return &SemanticClustering{
		llmClient: llmClient,
	}
}

// ClusterByMeaning はセマンティック類似性で応答をクラスタリング
func (sc *SemanticClustering) ClusterByMeaning(
	ctx context.Context,
	responses []string,
	entailmentMatrix [][]float64,
) ([]*SemanticCluster, error) {

	if len(responses) == 0 {
		return []*SemanticCluster{}, nil
	}

	if len(responses) == 1 {
		// 単一応答の場合は1つのクラスタを作成
		cluster := &SemanticCluster{
			ID:              "cluster_1",
			Responses:       responses,
			Prototype:       responses[0],
			SimilarityScore: 1.0,
			Weight:          1.0,
			SemanticVector:  []float64{1.0}, // 簡易ベクトル
		}
		return []*SemanticCluster{cluster}, nil
	}

	// Step 1: セマンティック埋め込み生成
	embeddings, err := sc.generateSemanticEmbeddings(ctx, responses)
	if err != nil {
		return nil, fmt.Errorf("セマンティック埋め込み生成エラー: %w", err)
	}

	// Step 2: 類似度行列の計算
	similarityMatrix := sc.calculateSimilarityMatrix(embeddings, entailmentMatrix)

	// Step 3: 階層クラスタリングの実行
	clusters := sc.performHierarchicalClustering(responses, similarityMatrix, embeddings)

	// Step 4: クラスタ品質の評価と最適化
	optimizedClusters := sc.optimizeClusters(clusters, similarityMatrix)

	return optimizedClusters, nil
}

// generateSemanticEmbeddings はLLMを使用してセマンティック埋め込みを生成
func (sc *SemanticClustering) generateSemanticEmbeddings(
	ctx context.Context,
	responses []string,
) ([][]float64, error) {

	embeddings := make([][]float64, len(responses))

	for i, response := range responses {
		// LLMを使用してセマンティック分析を実行
		embedding, err := sc.generateSingleEmbedding(ctx, response)
		if err != nil {
			// エラー時は簡易的な埋め込みを生成
			embedding = sc.generateSimpleEmbedding(response)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// generateSingleEmbedding は単一応答のセマンティック埋め込みを生成
func (sc *SemanticClustering) generateSingleEmbedding(
	ctx context.Context,
	response string,
) ([]float64, error) {

	prompt := fmt.Sprintf(`
以下のテキストをセマンティック特徴で数値化してください：

【テキスト】: %s

以下の5つの次元で0.0-1.0のスコアを付けてください：
1. 具体性 (concrete vs abstract)
2. 技術性 (technical vs general) 
3. 感情的トーン (positive vs negative)
4. 確実性 (certain vs uncertain)
5. 複雑性 (complex vs simple)

JSON形式で返答：
{
  "concreteness": 0.0-1.0,
  "technicality": 0.0-1.0,
  "emotional_tone": 0.0-1.0,
  "certainty": 0.0-1.0,
  "complexity": 0.0-1.0
}
`, response)

	request := &ai.GenerateRequest{
		Messages: []ai.Message{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
	}

	aiResponse, err := sc.llmClient.GenerateResponse(ctx, request)
	if err != nil {
		return nil, err
	}

	// JSON レスポンスをパース
	embedding := sc.parseEmbeddingResponse(aiResponse.Content)
	return embedding, nil
}

// parseEmbeddingResponse はLLMの応答から埋め込みベクトルを抽出
func (sc *SemanticClustering) parseEmbeddingResponse(response string) []float64 {
	// 簡易的なJSON値抽出（完全な実装ではJSONパーサーを使用）
	embedding := make([]float64, 5)

	dimensions := []string{"concreteness", "technicality", "emotional_tone", "certainty", "complexity"}

	for i, dim := range dimensions {
		value := sc.extractValueFromJSON(response, dim)
		embedding[i] = value
	}

	return embedding
}

// extractValueFromJSON は簡易的なJSON値抽出
func (sc *SemanticClustering) extractValueFromJSON(jsonStr, field string) float64 {
	searchStr := fmt.Sprintf(`"%s":`, field)
	startIdx := strings.Index(jsonStr, searchStr)
	if startIdx == -1 {
		return 0.5 // デフォルト値
	}

	startIdx += len(searchStr)
	for i := startIdx; i < len(jsonStr); i++ {
		char := jsonStr[i]
		if char >= '0' && char <= '9' || char == '.' {
			endIdx := i + 1
			for endIdx < len(jsonStr) && (jsonStr[endIdx] >= '0' && jsonStr[endIdx] <= '9' || jsonStr[endIdx] == '.') {
				endIdx++
			}

			if valueStr := jsonStr[i:endIdx]; valueStr != "" {
				if value, err := parseFloat(valueStr); err == nil {
					return math.Max(0.0, math.Min(1.0, value))
				}
			}
			break
		}
	}

	return 0.5
}

// generateSimpleEmbedding は簡易的な埋め込みを生成（フォールバック用）
func (sc *SemanticClustering) generateSimpleEmbedding(response string) []float64 {
	// 簡易的な特徴抽出
	embedding := make([]float64, 5)

	lowerResponse := strings.ToLower(response)
	words := strings.Fields(response)

	// 具体性: 数値や固有名詞の存在
	concreteness := 0.5
	if containsNumbers(response) || containsProperNouns(response) {
		concreteness = 0.8
	}
	embedding[0] = concreteness

	// 技術性: 技術用語の密度
	technicalWords := countTechnicalWords(lowerResponse)
	technicality := math.Min(1.0, float64(technicalWords)/float64(len(words)))
	embedding[1] = technicality

	// 感情的トーン: ポジティブ/ネガティブ語の存在
	emotionalTone := 0.5
	if containsPositiveWords(lowerResponse) {
		emotionalTone = 0.7
	} else if containsNegativeWords(lowerResponse) {
		emotionalTone = 0.3
	}
	embedding[2] = emotionalTone

	// 確実性: 不確実性を示す語の存在
	certainty := 0.7
	if containsUncertaintyWords(lowerResponse) {
		certainty = 0.3
	}
	embedding[3] = certainty

	// 複雑性: 文の長さと構造
	complexity := math.Min(1.0, float64(len(words))/20.0)
	embedding[4] = complexity

	return embedding
}

// calculateSimilarityMatrix は類似度行列を計算
func (sc *SemanticClustering) calculateSimilarityMatrix(
	embeddings [][]float64,
	entailmentMatrix [][]float64,
) [][]float64 {
	n := len(embeddings)
	similarity := make([][]float64, n)
	for i := range similarity {
		similarity[i] = make([]float64, n)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				similarity[i][j] = 1.0
				continue
			}

			// セマンティック類似度とNLI含意関係を統合
			semanticSim := sc.cosineSimilarity(embeddings[i], embeddings[j])

			var entailmentSim float64
			if i < len(entailmentMatrix) && j < len(entailmentMatrix[i]) {
				entailmentSim = entailmentMatrix[i][j]
			} else {
				entailmentSim = 0.5
			}

			// 重み付き統合 (セマンティック60%, 含意関係40%)
			similarity[i][j] = 0.6*semanticSim + 0.4*entailmentSim
		}
	}

	return similarity
}

// cosineSimilarity はコサイン類似度を計算
func (sc *SemanticClustering) cosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) {
		return 0.0
	}

	var dotProduct, norm1, norm2 float64

	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// performHierarchicalClustering は階層クラスタリングを実行
func (sc *SemanticClustering) performHierarchicalClustering(
	responses []string,
	similarityMatrix [][]float64,
	embeddings [][]float64,
) []*SemanticCluster {
	n := len(responses)

	// 最適なクラスタ数を決定
	optimalK := sc.determineOptimalK(similarityMatrix, 2, minCluster(n, 5))

	// k-meansクラスタリング実行
	clusterAssignments := sc.performKMeans(similarityMatrix, optimalK)

	// クラスタオブジェクトを構築
	clusters := sc.buildClusters(responses, embeddings, clusterAssignments, optimalK)

	return clusters
}

// determineOptimalK はシルエット分析でクラスタ数を決定
func (sc *SemanticClustering) determineOptimalK(similarityMatrix [][]float64, minK, maxK int) int {
	bestK := minK
	bestScore := -1.0

	for k := minK; k <= maxK; k++ {
		assignments := sc.performKMeans(similarityMatrix, k)
		score := sc.calculateSilhouetteScore(similarityMatrix, assignments, k)

		if score > bestScore {
			bestScore = score
			bestK = k
		}
	}

	return bestK
}

// performKMeans は簡易k-meansクラスタリングを実行
func (sc *SemanticClustering) performKMeans(similarityMatrix [][]float64, k int) []int {
	n := len(similarityMatrix)
	if n <= k {
		// データ数がクラスタ数以下の場合は各データを独立したクラスタに
		assignments := make([]int, n)
		for i := range assignments {
			assignments[i] = i
		}
		return assignments
	}

	assignments := make([]int, n)

	// 初期クラスタ割り当て
	for i := 0; i < n; i++ {
		assignments[i] = i % k
	}

	// イテレーション
	maxIterations := 10
	for iter := 0; iter < maxIterations; iter++ {
		changed := false

		for i := 0; i < n; i++ {
			bestCluster := assignments[i]
			bestSimilarity := sc.calculateClusterSimilarity(i, bestCluster, assignments, similarityMatrix)

			for cluster := 0; cluster < k; cluster++ {
				if cluster == assignments[i] {
					continue
				}

				similarity := sc.calculateClusterSimilarity(i, cluster, assignments, similarityMatrix)
				if similarity > bestSimilarity {
					bestSimilarity = similarity
					bestCluster = cluster
				}
			}

			if bestCluster != assignments[i] {
				assignments[i] = bestCluster
				changed = true
			}
		}

		if !changed {
			break
		}
	}

	return assignments
}

// calculateClusterSimilarity はデータポイントとクラスタの類似度を計算
func (sc *SemanticClustering) calculateClusterSimilarity(
	dataPoint, cluster int,
	assignments []int,
	similarityMatrix [][]float64,
) float64 {
	totalSimilarity := 0.0
	count := 0

	for i, assignment := range assignments {
		if assignment == cluster && i != dataPoint {
			totalSimilarity += similarityMatrix[dataPoint][i]
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return totalSimilarity / float64(count)
}

// buildClusters はクラスタオブジェクトを構築
func (sc *SemanticClustering) buildClusters(
	responses []string,
	embeddings [][]float64,
	assignments []int,
	k int,
) []*SemanticCluster {
	clusters := make([]*SemanticCluster, 0, k)

	for clusterID := 0; clusterID < k; clusterID++ {
		var clusterResponses []string
		var clusterEmbeddings [][]float64

		for i, assignment := range assignments {
			if assignment == clusterID {
				clusterResponses = append(clusterResponses, responses[i])
				clusterEmbeddings = append(clusterEmbeddings, embeddings[i])
			}
		}

		if len(clusterResponses) == 0 {
			continue
		}

		// クラスタプロトタイプの選択（最も中心に近いもの）
		prototype := sc.selectPrototype(clusterResponses, clusterEmbeddings)

		// クラスタ統計の計算
		similarityScore := sc.calculateIntraClusterSimilarity(clusterEmbeddings)
		weight := float64(len(clusterResponses)) / float64(len(responses))

		// セマンティックベクトル（平均）
		semanticVector := sc.calculateMeanEmbedding(clusterEmbeddings)

		cluster := &SemanticCluster{
			ID:              fmt.Sprintf("cluster_%d", clusterID+1),
			Responses:       clusterResponses,
			Prototype:       prototype,
			SimilarityScore: similarityScore,
			Weight:          weight,
			SemanticVector:  semanticVector,
		}

		clusters = append(clusters, cluster)
	}

	return clusters
}

// selectPrototype はクラスタの代表的な応答を選択
func (sc *SemanticClustering) selectPrototype(responses []string, embeddings [][]float64) string {
	if len(responses) == 1 {
		return responses[0]
	}

	// 中心に最も近い応答を選択
	centroid := sc.calculateMeanEmbedding(embeddings)

	bestIdx := 0
	bestSimilarity := sc.cosineSimilarity(embeddings[0], centroid)

	for i := 1; i < len(embeddings); i++ {
		similarity := sc.cosineSimilarity(embeddings[i], centroid)
		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestIdx = i
		}
	}

	return responses[bestIdx]
}

// calculateIntraClusterSimilarity はクラスタ内類似度を計算
func (sc *SemanticClustering) calculateIntraClusterSimilarity(embeddings [][]float64) float64 {
	if len(embeddings) <= 1 {
		return 1.0
	}

	totalSimilarity := 0.0
	count := 0

	for i := 0; i < len(embeddings); i++ {
		for j := i + 1; j < len(embeddings); j++ {
			totalSimilarity += sc.cosineSimilarity(embeddings[i], embeddings[j])
			count++
		}
	}

	if count == 0 {
		return 1.0
	}

	return totalSimilarity / float64(count)
}

// calculateMeanEmbedding は平均埋め込みを計算
func (sc *SemanticClustering) calculateMeanEmbedding(embeddings [][]float64) []float64 {
	if len(embeddings) == 0 {
		return []float64{0.5, 0.5, 0.5, 0.5, 0.5} // デフォルト
	}

	dimensions := len(embeddings[0])
	mean := make([]float64, dimensions)

	for _, embedding := range embeddings {
		for i, value := range embedding {
			mean[i] += value
		}
	}

	for i := range mean {
		mean[i] /= float64(len(embeddings))
	}

	return mean
}

// calculateSilhouetteScore はシルエットスコアを計算
func (sc *SemanticClustering) calculateSilhouetteScore(
	similarityMatrix [][]float64,
	assignments []int,
	k int,
) float64 {
	n := len(assignments)
	if n <= 1 || k <= 1 {
		return 0.0
	}

	totalScore := 0.0

	for i := 0; i < n; i++ {
		a := sc.calculateIntraClusterDistance(i, assignments, similarityMatrix)
		b := sc.calculateNearestClusterDistance(i, assignments, similarityMatrix, k)

		if a == 0 && b == 0 {
			continue
		}

		silhouette := (b - a) / math.Max(a, b)
		totalScore += silhouette
	}

	return totalScore / float64(n)
}

// optimizeClusters はクラスタを最適化
func (sc *SemanticClustering) optimizeClusters(
	clusters []*SemanticCluster,
	similarityMatrix [][]float64,
) []*SemanticCluster {
	// 小さすぎるクラスタを統合
	optimized := sc.mergeSmallClusters(clusters)

	// クラスタ品質でソート
	sort.Slice(optimized, func(i, j int) bool {
		return optimized[i].Weight > optimized[j].Weight
	})

	return optimized
}

// mergeSmallClusters は小さなクラスタを統合
func (sc *SemanticClustering) mergeSmallClusters(clusters []*SemanticCluster) []*SemanticCluster {
	minClusterSize := 0.1 // 全体の10%未満のクラスタは統合対象
	var optimized []*SemanticCluster
	var smallClusters []*SemanticCluster

	for _, cluster := range clusters {
		if cluster.Weight >= minClusterSize {
			optimized = append(optimized, cluster)
		} else {
			smallClusters = append(smallClusters, cluster)
		}
	}

	// 小さなクラスタを最大のクラスタに統合
	if len(smallClusters) > 0 && len(optimized) > 0 {
		largestCluster := optimized[0]

		for _, small := range smallClusters {
			largestCluster.Responses = append(largestCluster.Responses, small.Responses...)
			largestCluster.Weight += small.Weight
		}

		// 類似度スコアを再計算
		largestCluster.SimilarityScore = sc.recalculateClusterSimilarity(largestCluster)
	}

	return optimized
}

// Helper functions

func (sc *SemanticClustering) calculateIntraClusterDistance(
	dataPoint int,
	assignments []int,
	similarityMatrix [][]float64,
) float64 {
	cluster := assignments[dataPoint]
	totalDistance := 0.0
	count := 0

	for i, assignment := range assignments {
		if assignment == cluster && i != dataPoint {
			totalDistance += 1.0 - similarityMatrix[dataPoint][i] // distance = 1 - similarity
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return totalDistance / float64(count)
}

func (sc *SemanticClustering) calculateNearestClusterDistance(
	dataPoint int,
	assignments []int,
	similarityMatrix [][]float64,
	k int,
) float64 {
	currentCluster := assignments[dataPoint]
	minDistance := math.Inf(1)

	for cluster := 0; cluster < k; cluster++ {
		if cluster == currentCluster {
			continue
		}

		totalDistance := 0.0
		count := 0

		for i, assignment := range assignments {
			if assignment == cluster {
				totalDistance += 1.0 - similarityMatrix[dataPoint][i]
				count++
			}
		}

		if count > 0 {
			avgDistance := totalDistance / float64(count)
			if avgDistance < minDistance {
				minDistance = avgDistance
			}
		}
	}

	if minDistance == math.Inf(1) {
		return 0.0
	}

	return minDistance
}

func (sc *SemanticClustering) recalculateClusterSimilarity(cluster *SemanticCluster) float64 {
	// 簡易的な類似度再計算
	return math.Max(0.3, cluster.SimilarityScore*0.8)
}

// Utility functions for simple embedding generation

func containsNumbers(text string) bool {
	for _, char := range text {
		if char >= '0' && char <= '9' {
			return true
		}
	}
	return false
}

func containsProperNouns(text string) bool {
	words := strings.Fields(text)
	for _, word := range words {
		if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
			return true
		}
	}
	return false
}

func countTechnicalWords(text string) int {
	technicalWords := []string{
		"api", "algorithm", "function", "method", "class", "object",
		"database", "server", "client", "framework", "library",
		"implementation", "interface", "protocol", "authentication",
		"configuration", "deployment", "testing", "debug",
	}

	count := 0
	for _, techWord := range technicalWords {
		if strings.Contains(text, techWord) {
			count++
		}
	}
	return count
}

func containsPositiveWords(text string) bool {
	positiveWords := []string{
		"good", "great", "excellent", "success", "effective",
		"improve", "better", "optimal", "efficient", "reliable",
	}

	for _, word := range positiveWords {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func containsNegativeWords(text string) bool {
	negativeWords := []string{
		"error", "problem", "issue", "fail", "wrong",
		"bad", "poor", "inefficient", "unreliable", "broken",
	}

	for _, word := range negativeWords {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func containsUncertaintyWords(text string) bool {
	uncertaintyWords := []string{
		"maybe", "perhaps", "might", "could", "possibly",
		"uncertain", "unclear", "ambiguous", "probably",
	}

	for _, word := range uncertaintyWords {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func parseFloat(s string) (float64, error) {
	// 簡易的なfloat変換（実装では strconv.ParseFloatを使用）
	if s == "0" {
		return 0.0, nil
	} else if s == "1" {
		return 1.0, nil
	}
	// デフォルト値を返す
	return 0.5, nil
}

func minCluster(a, b int) int {
	if a < b {
		return a
	}
	return b
}
