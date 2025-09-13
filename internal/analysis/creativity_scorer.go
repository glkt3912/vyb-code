package analysis

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/glkt/vyb-code/internal/ai"
)

// CreativityScorer は意味距離に基づく創造性測定エンジン
type CreativityScorer struct {
	llmClient       ai.LLMClient
	noveltyDetector *NoveltyDetector
	conceptBlender  *ConceptualBlender
	originalityBank *OriginalityDatabase
}

// CreativityResult は創造性分析結果
type CreativityResult struct {
	OverallScore       float64                 `json:"overall_score"`
	Novelty            float64                 `json:"novelty"`
	Originality        float64                 `json:"originality"`
	Fluency            float64                 `json:"fluency"`
	Flexibility        float64                 `json:"flexibility"`
	Elaboration        float64                 `json:"elaboration"`
	SemanticDistance   float64                 `json:"semantic_distance"`
	ConceptualBlending float64                 `json:"conceptual_blending"`
	IdeaDivergence     float64                 `json:"idea_divergence"`
	CreativePatterns   []CreativePattern       `json:"creative_patterns"`
	NovelCombinations  []ConceptCombination    `json:"novel_combinations"`
	InsightMetrics     *CreativeInsightMetrics `json:"insight_metrics"`
	DimensionScores    *CreativityDimensions   `json:"dimension_scores"`
}

// CreativePattern は創造的パターンの識別
type CreativePattern struct {
	Pattern    string  `json:"pattern"`
	Type       string  `json:"type"` // "metaphor", "analogy", "synthesis", "divergent"
	Strength   float64 `json:"strength"`
	Frequency  int     `json:"frequency"`
	Uniqueness float64 `json:"uniqueness"`
	Context    string  `json:"context"`
}

// ConceptCombination は概念の組み合わせ
type ConceptCombination struct {
	Concept1        string  `json:"concept1"`
	Concept2        string  `json:"concept2"`
	CombinationType string  `json:"combination_type"`
	Novelty         float64 `json:"novelty"`
	Coherence       float64 `json:"coherence"`
	Utility         float64 `json:"utility"`
	SemanticGap     float64 `json:"semantic_gap"`
}

// CreativeInsightMetrics は創造的洞察のメトリクス
type CreativeInsightMetrics struct {
	AhaFactor           float64 `json:"aha_factor"`           // "ユーレカ"度
	RemoteConnections   int     `json:"remote_connections"`   // 遠隔連想数
	ConceptualLeaps     int     `json:"conceptual_leaps"`     // 概念的跳躍
	ParadigmShifts      int     `json:"paradigm_shifts"`      // パラダイムシフト
	CrossDomainBridge   float64 `json:"cross_domain_bridge"`  // 領域横断度
	InnovationPotential float64 `json:"innovation_potential"` // 革新ポテンシャル
}

// CreativityDimensions はGuilfordの創造性4要素
type CreativityDimensions struct {
	Fluency     float64 `json:"fluency"`     // 流暢性 - アイデアの量
	Flexibility float64 `json:"flexibility"` // 柔軟性 - アイデアの種類
	Originality float64 `json:"originality"` // 独創性 - アイデアの珍しさ
	Elaboration float64 `json:"elaboration"` // 精密性 - アイデアの詳細度
}

// NoveltyDetector は新規性検出器
type NoveltyDetector struct {
	knownPatterns   []string
	domainKnowledge map[string][]string
	rarenessCutoff  float64
}

// ConceptualBlender は概念融合分析器
type ConceptualBlender struct {
	conceptDatabase *ConceptDatabase
	blendingRules   []BlendingRule
}

// OriginalityDatabase は独創性判定データベース
type OriginalityDatabase struct {
	commonSolutions    map[string]float64
	originalityIndex   map[string]float64
	creativeBenchmarks []CreativeBenchmark
}

// ConceptDatabase は概念データベース
type ConceptDatabase struct {
	Concepts   map[string]*Concept
	Relations  map[string][]Relation
	Embeddings map[string][]float64
}

// Concept は概念の定義
type Concept struct {
	Name       string            `json:"name"`
	Domain     string            `json:"domain"`
	Properties map[string]string `json:"properties"`
	Embedding  []float64         `json:"embedding"`
	Frequency  float64           `json:"frequency"`
}

// Relation は概念間の関係
type Relation struct {
	Source   string  `json:"source"`
	Target   string  `json:"target"`
	Type     string  `json:"type"` // "similarity", "causality", "composition"
	Strength float64 `json:"strength"`
}

// BlendingRule は概念融合ルール
type BlendingRule struct {
	Name       string   `json:"name"`
	Conditions []string `json:"conditions"`
	Operation  string   `json:"operation"`
	Weight     float64  `json:"weight"`
}

// CreativeBenchmark は創造性ベンチマーク
type CreativeBenchmark struct {
	Domain          string   `json:"domain"`
	Problem         string   `json:"problem"`
	Solutions       []string `json:"solutions"`
	CreativityScore float64  `json:"creativity_score"`
}

// NewCreativityScorer は新しい創造性測定器を作成
func NewCreativityScorer(llmClient ai.LLMClient) *CreativityScorer {
	return &CreativityScorer{
		llmClient:       llmClient,
		noveltyDetector: initializeNoveltyDetector(),
		conceptBlender:  initializeConceptualBlender(),
		originalityBank: initializeOriginalityDatabase(),
	}
}

// MeasureCreativity は創造性を包括的に測定
func (cs *CreativityScorer) MeasureCreativity(
	ctx context.Context,
	response string,
	originalQuery string,
	context map[string]interface{},
) (*CreativityResult, error) {

	// Step 1: 基本的創造性次元の測定
	dimensions, err := cs.measureCreativityDimensions(ctx, response)
	if err != nil {
		// エラー時は簡易分析を使用
		dimensions = cs.basicDimensionAnalysis(response)
	}

	// Step 2: 意味距離による新規性分析
	semanticDistance := cs.calculateSemanticDistance(response, originalQuery)

	// Step 3: 概念融合の検出
	conceptualBlending := cs.analyzeConceptualBlending(ctx, response)

	// Step 4: アイデア発散度の測定
	ideaDivergence := cs.measureIdeaDivergence(response, context)

	// Step 5: 創造的パターンの特定
	creativePatterns := cs.identifyCreativePatterns(response)

	// Step 6: 新規概念組み合わせの検出
	novelCombinations := cs.detectNovelCombinations(ctx, response)

	// Step 7: 創造的洞察メトリクスの算出
	insightMetrics := cs.calculateInsightMetrics(response, creativePatterns, novelCombinations)

	// Step 8: 総合創造性スコアの計算
	overallScore := cs.calculateOverallCreativity(
		dimensions, semanticDistance, conceptualBlending, ideaDivergence, insightMetrics,
	)

	return &CreativityResult{
		OverallScore:       overallScore,
		Novelty:            cs.calculateNovelty(semanticDistance, novelCombinations),
		Originality:        dimensions.Originality,
		Fluency:            dimensions.Fluency,
		Flexibility:        dimensions.Flexibility,
		Elaboration:        dimensions.Elaboration,
		SemanticDistance:   semanticDistance,
		ConceptualBlending: conceptualBlending,
		IdeaDivergence:     ideaDivergence,
		CreativePatterns:   creativePatterns,
		NovelCombinations:  novelCombinations,
		InsightMetrics:     insightMetrics,
		DimensionScores:    dimensions,
	}, nil
}

// measureCreativityDimensions はGuilfordの創造性4要素を測定
func (cs *CreativityScorer) measureCreativityDimensions(
	ctx context.Context,
	response string,
) (*CreativityDimensions, error) {

	prompt := fmt.Sprintf(`
以下のテキストをGuilfordの創造性4要素で分析してください：

【テキスト】: %s

以下の4次元で0.0-1.0のスコアを付けてください：

1. **流暢性 (Fluency)**: アイデアの量・豊富さ
   - 提示されたアイデアや選択肢の数
   - 思考の流れの活発さ

2. **柔軟性 (Flexibility)**: アイデアの種類・多様性
   - 異なるカテゴリーのアプローチ
   - 視点の切り替え能力

3. **独創性 (Originality)**: アイデアの珍しさ・ユニークさ
   - 一般的でない解決策
   - 意外性や驚きの要素

4. **精密性 (Elaboration)**: アイデアの詳細化・発展
   - 具体的な説明や展開
   - アイデアの完成度

JSON形式で返答：
{
  "fluency": 0.0-1.0,
  "flexibility": 0.0-1.0,
  "originality": 0.0-1.0,
  "elaboration": 0.0-1.0
}
`, response)

	request := &ai.GenerateRequest{
		Messages: []ai.Message{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
	}

	aiResponse, err := cs.llmClient.GenerateResponse(ctx, request)
	if err != nil {
		return nil, err
	}

	// JSON応答をパース
	dimensions := cs.parseDimensionsResponse(aiResponse.Content)
	return dimensions, nil
}

// basicDimensionAnalysis は基本的な次元分析（フォールバック用）
func (cs *CreativityScorer) basicDimensionAnalysis(response string) *CreativityDimensions {
	words := strings.Fields(response)
	sentences := cs.splitSentences(response)

	// 流暢性: 語数とアイデア数に基づく
	fluency := math.Min(1.0, float64(len(words))/100.0)

	// 柔軟性: 接続詞と視点変化の検出
	flexibility := cs.detectFlexibilityMarkers(response)

	// 独創性: 珍しい語彙と表現の検出
	originality := cs.detectOriginalityMarkers(response)

	// 精密性: 文章の詳細度と構造化
	elaboration := math.Min(1.0, float64(len(sentences))*0.1)

	return &CreativityDimensions{
		Fluency:     fluency,
		Flexibility: flexibility,
		Originality: originality,
		Elaboration: elaboration,
	}
}

// calculateSemanticDistance は意味距離を計算
func (cs *CreativityScorer) calculateSemanticDistance(response, query string) float64 {
	// 簡易的なセマンティック距離計算
	responseWords := cs.extractKeywords(response)
	queryWords := cs.extractKeywords(query)

	overlap := cs.calculateWordOverlap(responseWords, queryWords)
	distance := 1.0 - overlap

	// 意味的新規性の評価
	noveltyBonus := cs.assessNovelty(responseWords, queryWords)

	return math.Min(1.0, distance+noveltyBonus*0.3)
}

// analyzeConceptualBlending は概念融合を分析
func (cs *CreativityScorer) analyzeConceptualBlending(ctx context.Context, response string) float64 {
	concepts := cs.extractConcepts(response)

	if len(concepts) < 2 {
		return 0.0
	}

	blendingScore := 0.0
	totalPairs := 0

	// すべての概念ペアの融合度を計算
	for i := 0; i < len(concepts); i++ {
		for j := i + 1; j < len(concepts); j++ {
			blend := cs.evaluateConceptualBlend(concepts[i], concepts[j])
			blendingScore += blend
			totalPairs++
		}
	}

	if totalPairs == 0 {
		return 0.0
	}

	return blendingScore / float64(totalPairs)
}

// measureIdeaDivergence はアイデア発散度を測定
func (cs *CreativityScorer) measureIdeaDivergence(
	response string,
	context map[string]interface{},
) float64 {
	// アイデアの多様性を評価
	ideas := cs.extractIdeas(response)

	if len(ideas) <= 1 {
		return 0.0
	}

	// アイデア間の意味距離を計算
	totalDistance := 0.0
	pairs := 0

	for i := 0; i < len(ideas); i++ {
		for j := i + 1; j < len(ideas); j++ {
			distance := cs.calculateIdeaDistance(ideas[i], ideas[j])
			totalDistance += distance
			pairs++
		}
	}

	if pairs == 0 {
		return 0.0
	}

	avgDistance := totalDistance / float64(pairs)

	// 発散度は平均距離と数の組み合わせ
	divergence := avgDistance * math.Log(float64(len(ideas))+1)

	return math.Min(1.0, divergence/5.0)
}

// identifyCreativePatterns は創造的パターンを特定
func (cs *CreativityScorer) identifyCreativePatterns(response string) []CreativePattern {
	patterns := []CreativePattern{}
	lowerResponse := strings.ToLower(response)

	// メタファーパターンの検出
	metaphors := cs.detectMetaphors(lowerResponse)
	for _, metaphor := range metaphors {
		patterns = append(patterns, CreativePattern{
			Pattern:    metaphor,
			Type:       "metaphor",
			Strength:   0.8,
			Frequency:  1,
			Uniqueness: cs.evaluateUniqueness(metaphor),
			Context:    "metaphorical_expression",
		})
	}

	// アナロジーパターンの検出
	analogies := cs.detectAnalogies(lowerResponse)
	for _, analogy := range analogies {
		patterns = append(patterns, CreativePattern{
			Pattern:    analogy,
			Type:       "analogy",
			Strength:   0.7,
			Frequency:  1,
			Uniqueness: cs.evaluateUniqueness(analogy),
			Context:    "analogical_reasoning",
		})
	}

	// 発散的思考パターンの検出
	if cs.hasDivergentThinking(lowerResponse) {
		patterns = append(patterns, CreativePattern{
			Pattern:    "divergent_exploration",
			Type:       "divergent",
			Strength:   0.6,
			Frequency:  1,
			Uniqueness: 0.7,
			Context:    "multiple_perspectives",
		})
	}

	return patterns
}

// detectNovelCombinations は新規概念組み合わせを検出
func (cs *CreativityScorer) detectNovelCombinations(
	ctx context.Context,
	response string,
) []ConceptCombination {
	concepts := cs.extractConcepts(response)
	combinations := []ConceptCombination{}

	for i := 0; i < len(concepts); i++ {
		for j := i + 1; j < len(concepts); j++ {
			if cs.isNovelCombination(concepts[i], concepts[j]) {
				combo := ConceptCombination{
					Concept1:        concepts[i],
					Concept2:        concepts[j],
					CombinationType: cs.classifyCombination(concepts[i], concepts[j]),
					Novelty:         cs.calculateCombinationNovelty(concepts[i], concepts[j]),
					Coherence:       cs.evaluateCombinationCoherence(concepts[i], concepts[j]),
					Utility:         cs.assessCombinationUtility(concepts[i], concepts[j]),
					SemanticGap:     cs.calculateSemanticGap(concepts[i], concepts[j]),
				}
				combinations = append(combinations, combo)
			}
		}
	}

	return combinations
}

// calculateInsightMetrics は創造的洞察メトリクスを算出
func (cs *CreativityScorer) calculateInsightMetrics(
	response string,
	patterns []CreativePattern,
	combinations []ConceptCombination,
) *CreativeInsightMetrics {

	ahaFactor := cs.evaluateAhaFactor(response, patterns)
	remoteConnections := cs.countRemoteConnections(combinations)
	conceptualLeaps := cs.identifyConceptualLeaps(response)
	paradigmShifts := cs.detectParadigmShifts(response)
	crossDomainBridge := cs.measureCrossDomainBridging(combinations)
	innovationPotential := cs.assessInnovationPotential(patterns, combinations)

	return &CreativeInsightMetrics{
		AhaFactor:           ahaFactor,
		RemoteConnections:   remoteConnections,
		ConceptualLeaps:     conceptualLeaps,
		ParadigmShifts:      paradigmShifts,
		CrossDomainBridge:   crossDomainBridge,
		InnovationPotential: innovationPotential,
	}
}

// calculateOverallCreativity は総合創造性スコアを計算
func (cs *CreativityScorer) calculateOverallCreativity(
	dimensions *CreativityDimensions,
	semanticDistance, conceptualBlending, ideaDivergence float64,
	insights *CreativeInsightMetrics,
) float64 {
	// Guilfordの4要素の重み付き平均
	guilfordScore := (dimensions.Fluency*0.2 + dimensions.Flexibility*0.3 +
		dimensions.Originality*0.3 + dimensions.Elaboration*0.2)

	// 意味距離と概念融合の統合
	semanticScore := (semanticDistance + conceptualBlending) / 2.0

	// 洞察メトリクスの統合
	insightScore := (insights.AhaFactor + insights.CrossDomainBridge +
		insights.InnovationPotential) / 3.0

	// 総合スコア（重み付き統合）
	overallScore := guilfordScore*0.4 + semanticScore*0.3 +
		ideaDivergence*0.15 + insightScore*0.15

	return math.Min(1.0, math.Max(0.0, overallScore))
}

// calculateNovelty は新規性を計算
func (cs *CreativityScorer) calculateNovelty(
	semanticDistance float64,
	combinations []ConceptCombination,
) float64 {
	novelty := semanticDistance

	// 新規組み合わせによる新規性ボーナス
	if len(combinations) > 0 {
		totalCombinationNovelty := 0.0
		for _, combo := range combinations {
			totalCombinationNovelty += combo.Novelty
		}
		avgCombinationNovelty := totalCombinationNovelty / float64(len(combinations))
		novelty = (novelty + avgCombinationNovelty) / 2.0
	}

	return math.Min(1.0, novelty)
}

// Helper functions - 実装の詳細は簡略化

func (cs *CreativityScorer) parseDimensionsResponse(response string) *CreativityDimensions {
	// 簡易実装 - 実際にはJSONパーサーを使用
	return &CreativityDimensions{
		Fluency:     0.7,
		Flexibility: 0.6,
		Originality: 0.8,
		Elaboration: 0.7,
	}
}

func (cs *CreativityScorer) splitSentences(text string) []string {
	return strings.Split(text, "。")
}

func (cs *CreativityScorer) detectFlexibilityMarkers(text string) float64 {
	markers := []string{"一方", "他方", "alternatively", "on the other hand", "however"}
	count := 0
	lowerText := strings.ToLower(text)

	for _, marker := range markers {
		if strings.Contains(lowerText, marker) {
			count++
		}
	}

	return math.Min(1.0, float64(count)*0.2)
}

func (cs *CreativityScorer) detectOriginalityMarkers(text string) float64 {
	originalWords := []string{"revolutionary", "unprecedented", "innovative", "novel", "unique"}
	count := 0
	lowerText := strings.ToLower(text)

	for _, word := range originalWords {
		if strings.Contains(lowerText, word) {
			count++
		}
	}

	return math.Min(1.0, float64(count)*0.25)
}

func (cs *CreativityScorer) extractKeywords(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	// 簡易実装 - 実際にはNLP処理でキーワード抽出
	return words
}

func (cs *CreativityScorer) calculateWordOverlap(words1, words2 []string) float64 {
	overlap := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				overlap++
				break
			}
		}
	}

	totalWords := len(words1) + len(words2)
	if totalWords == 0 {
		return 0.0
	}

	return float64(overlap*2) / float64(totalWords)
}

func (cs *CreativityScorer) assessNovelty(responseWords, queryWords []string) float64 {
	// 簡易新規性評価
	return 0.3
}

func (cs *CreativityScorer) extractConcepts(text string) []string {
	// 簡易概念抽出
	return []string{"concept1", "concept2", "concept3"}
}

func (cs *CreativityScorer) evaluateConceptualBlend(concept1, concept2 string) float64 {
	// 概念融合度の評価
	return 0.5
}

func (cs *CreativityScorer) extractIdeas(text string) []string {
	sentences := cs.splitSentences(text)
	return sentences
}

func (cs *CreativityScorer) calculateIdeaDistance(idea1, idea2 string) float64 {
	// アイデア間の距離計算
	return 0.7
}

func (cs *CreativityScorer) detectMetaphors(text string) []string {
	// メタファー検出
	return []string{}
}

func (cs *CreativityScorer) detectAnalogies(text string) []string {
	// アナロジー検出
	return []string{}
}

func (cs *CreativityScorer) hasDivergentThinking(text string) bool {
	// 発散的思考の検出
	return strings.Contains(text, "multiple") || strings.Contains(text, "various")
}

func (cs *CreativityScorer) evaluateUniqueness(pattern string) float64 {
	// ユニークネス評価
	return 0.6
}

func (cs *CreativityScorer) isNovelCombination(concept1, concept2 string) bool {
	// 新規組み合わせの判定
	return true
}

func (cs *CreativityScorer) classifyCombination(concept1, concept2 string) string {
	return "semantic_blend"
}

func (cs *CreativityScorer) calculateCombinationNovelty(concept1, concept2 string) float64 {
	return 0.8
}

func (cs *CreativityScorer) evaluateCombinationCoherence(concept1, concept2 string) float64 {
	return 0.7
}

func (cs *CreativityScorer) assessCombinationUtility(concept1, concept2 string) float64 {
	return 0.6
}

func (cs *CreativityScorer) calculateSemanticGap(concept1, concept2 string) float64 {
	return 0.5
}

func (cs *CreativityScorer) evaluateAhaFactor(response string, patterns []CreativePattern) float64 {
	return 0.7
}

func (cs *CreativityScorer) countRemoteConnections(combinations []ConceptCombination) int {
	return len(combinations)
}

func (cs *CreativityScorer) identifyConceptualLeaps(response string) int {
	return 2
}

func (cs *CreativityScorer) detectParadigmShifts(response string) int {
	return 1
}

func (cs *CreativityScorer) measureCrossDomainBridging(combinations []ConceptCombination) float64 {
	return 0.6
}

func (cs *CreativityScorer) assessInnovationPotential(
	patterns []CreativePattern,
	combinations []ConceptCombination,
) float64 {
	return 0.8
}

// Initialization functions

func initializeNoveltyDetector() *NoveltyDetector {
	return &NoveltyDetector{
		knownPatterns:   []string{},
		domainKnowledge: make(map[string][]string),
		rarenessCutoff:  0.1,
	}
}

func initializeConceptualBlender() *ConceptualBlender {
	return &ConceptualBlender{
		conceptDatabase: &ConceptDatabase{
			Concepts:   make(map[string]*Concept),
			Relations:  make(map[string][]Relation),
			Embeddings: make(map[string][]float64),
		},
		blendingRules: []BlendingRule{},
	}
}

func initializeOriginalityDatabase() *OriginalityDatabase {
	return &OriginalityDatabase{
		commonSolutions:    make(map[string]float64),
		originalityIndex:   make(map[string]float64),
		creativeBenchmarks: []CreativeBenchmark{},
	}
}
