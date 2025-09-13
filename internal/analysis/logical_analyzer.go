package analysis

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/glkt/vyb-code/internal/ai"
)

// LogicalStructureAnalyzer は論理構造分析による推論深度測定
type LogicalStructureAnalyzer struct {
	llmClient ai.LLMClient
	patterns  *LogicalPatternDatabase
}

// ReasoningDepthResult は推論深度分析結果
type ReasoningDepthResult struct {
	OverallDepth      int                      `json:"overall_depth"`
	LogicalConnectors int                      `json:"logical_connectors"`
	ChainLength       int                      `json:"chain_length"`
	AbstractionLevels int                      `json:"abstraction_levels"`
	CausalRelations   int                      `json:"causal_relations"`
	InferenceSteps    []*InferenceStepAnalysis `json:"inference_steps"`
	ComplexityScore   float64                  `json:"complexity_score"`
	LogicalCoherence  float64                  `json:"logical_coherence"`
	ArgumentStructure *ArgumentStructure       `json:"argument_structure"`
	ReasoningPatterns []string                 `json:"reasoning_patterns"`
}

// InferenceStepAnalysis は推論ステップの詳細分析
type InferenceStepAnalysis struct {
	StepNumber       int      `json:"step_number"`
	Type             string   `json:"type"` // "premise", "inference", "conclusion"
	Content          string   `json:"content"`
	LogicalStrength  float64  `json:"logical_strength"`
	Dependencies     []int    `json:"dependencies"`
	Connectors       []string `json:"connectors"`
	AbstractionLevel int      `json:"abstraction_level"`
}

// ArgumentStructure は論証構造の分析
type ArgumentStructure struct {
	Premises         []string `json:"premises"`
	Conclusions      []string `json:"conclusions"`
	WarrantLinks     []string `json:"warrant_links"`
	CounterArguments []string `json:"counter_arguments"`
	Rebuttals        []string `json:"rebuttals"`
	Qualifiers       []string `json:"qualifiers"`
	StructuralDepth  int      `json:"structural_depth"`
	ToulminScore     float64  `json:"toulmin_score"`
}

// LogicalPatternDatabase は論理パターンのデータベース
type LogicalPatternDatabase struct {
	ConnectorPatterns  []LogicalPattern `json:"connector_patterns"`
	InferencePatterns  []LogicalPattern `json:"inference_patterns"`
	CausalPatterns     []LogicalPattern `json:"causal_patterns"`
	AbstractionMarkers []string         `json:"abstraction_markers"`
}

// LogicalPattern は論理パターンの定義
type LogicalPattern struct {
	Name       string   `json:"name"`
	Patterns   []string `json:"patterns"`
	Weight     float64  `json:"weight"`
	Category   string   `json:"category"`
	Complexity int      `json:"complexity"`
}

// NewLogicalStructureAnalyzer は新しい論理構造分析器を作成
func NewLogicalStructureAnalyzer(llmClient ai.LLMClient) *LogicalStructureAnalyzer {
	return &LogicalStructureAnalyzer{
		llmClient: llmClient,
		patterns:  initializeLogicalPatterns(),
	}
}

// AnalyzeReasoningDepth は推論深度を分析
func (lsa *LogicalStructureAnalyzer) AnalyzeReasoningDepth(
	ctx context.Context,
	text string,
	originalQuery string,
) (*ReasoningDepthResult, error) {

	// Step 1: 論理構造の基本分析
	connectors := lsa.extractLogicalConnectors(text)
	chainLength := lsa.calculateChainLength(text)
	abstractionLevels := lsa.analyzeAbstractionLevels(text)

	// Step 2: 因果関係の分析
	causalRelations := lsa.analyzeCausalRelations(text)

	// Step 3: 推論ステップの分解
	inferenceSteps, err := lsa.decomposeInferenceSteps(ctx, text)
	if err != nil {
		// エラー時は簡易分析を使用
		inferenceSteps = lsa.simpleInferenceDecomposition(text)
	}

	// Step 4: 論証構造の分析
	argumentStructure, err := lsa.analyzeArgumentStructure(ctx, text, originalQuery)
	if err != nil {
		// エラー時は基本構造を生成
		argumentStructure = lsa.basicArgumentStructure(text)
	}

	// Step 5: 複雑性スコアの計算
	complexityScore := lsa.calculateComplexityScore(connectors, chainLength, abstractionLevels, causalRelations)

	// Step 6: 論理的一貫性の評価
	logicalCoherence := lsa.evaluateLogicalCoherence(inferenceSteps, argumentStructure)

	// Step 7: 推論パターンの特定
	reasoningPatterns := lsa.identifyReasoningPatterns(text, inferenceSteps)

	// Step 8: 総合的推論深度の計算
	overallDepth := lsa.calculateOverallDepth(
		len(connectors), chainLength, abstractionLevels, causalRelations,
		len(inferenceSteps), argumentStructure.StructuralDepth,
	)

	return &ReasoningDepthResult{
		OverallDepth:      overallDepth,
		LogicalConnectors: len(connectors),
		ChainLength:       chainLength,
		AbstractionLevels: abstractionLevels,
		CausalRelations:   causalRelations,
		InferenceSteps:    inferenceSteps,
		ComplexityScore:   complexityScore,
		LogicalCoherence:  logicalCoherence,
		ArgumentStructure: argumentStructure,
		ReasoningPatterns: reasoningPatterns,
	}, nil
}

// extractLogicalConnectors は論理的接続詞を抽出
func (lsa *LogicalStructureAnalyzer) extractLogicalConnectors(text string) []string {
	var connectors []string
	lowerText := strings.ToLower(text)

	// 日本語と英語の論理接続詞パターン
	connectorPatterns := []string{
		// 日本語
		"なぜなら", "したがって", "そのため", "つまり", "すなわち", "ただし", "しかし",
		"また", "さらに", "一方", "対照的に", "同様に", "例えば", "具体的には",
		"結論として", "要するに", "むしろ", "それゆえ", "このように", "このため",
		// 英語
		"because", "therefore", "thus", "hence", "consequently", "however", "moreover",
		"furthermore", "additionally", "in contrast", "similarly", "for example", "specifically",
		"in conclusion", "in summary", "rather", "as a result", "in this way",
	}

	for _, pattern := range connectorPatterns {
		if strings.Contains(lowerText, pattern) {
			connectors = append(connectors, pattern)
		}
	}

	return connectors
}

// calculateChainLength は推論チェーンの長さを計算
func (lsa *LogicalStructureAnalyzer) calculateChainLength(text string) int {
	// 文を分割
	sentences := lsa.splitIntoSentences(text)

	// 推論を示すパターンの検出
	reasoningIndicators := []string{
		"なぜなら", "したがって", "そのため", "つまり", "結論として",
		"because", "therefore", "thus", "consequently", "in conclusion",
	}

	chainLength := 0
	for _, sentence := range sentences {
		lowerSentence := strings.ToLower(sentence)
		for _, indicator := range reasoningIndicators {
			if strings.Contains(lowerSentence, indicator) {
				chainLength++
				break
			}
		}
	}

	// 最小チェーン長は1
	return maxInt(1, chainLength)
}

// analyzeAbstractionLevels は抽象化レベルを分析
func (lsa *LogicalStructureAnalyzer) analyzeAbstractionLevels(text string) int {
	lowerText := strings.ToLower(text)
	abstractionLevel := 1 // 基本レベル

	// 具体的表現（レベルを下げる）
	concreteMarkers := []string{
		"例えば", "具体的には", "実際に", "for example", "specifically", "in practice",
		"実装", "コード", "ファイル", "関数", "implementation", "code", "function",
	}

	// 抽象的表現（レベルを上げる）
	abstractMarkers := []string{
		"一般的に", "概念的に", "理論的に", "原則として", "generally", "conceptually", "theoretically",
		"パターン", "アーキテクチャ", "設計", "原理", "pattern", "architecture", "design", "principle",
		"メタ", "抽象", "概念", "理念", "meta", "abstract", "concept", "philosophy",
	}

	concreteCount := 0
	abstractCount := 0

	for _, marker := range concreteMarkers {
		if strings.Contains(lowerText, marker) {
			concreteCount++
		}
	}

	for _, marker := range abstractMarkers {
		if strings.Contains(lowerText, marker) {
			abstractCount++
		}
	}

	// 抽象度の計算
	if abstractCount > concreteCount {
		abstractionLevel = 2 + abstractCount - concreteCount
	} else if concreteCount > abstractCount {
		abstractionLevel = maxInt(1, 2-(concreteCount-abstractCount))
	} else {
		abstractionLevel = 2 // 中間レベル
	}

	return minInt(5, abstractionLevel) // 最大5レベル
}

// analyzeCausalRelations は因果関係を分析
func (lsa *LogicalStructureAnalyzer) analyzeCausalRelations(text string) int {
	lowerText := strings.ToLower(text)
	causalCount := 0

	causalPatterns := []string{
		// 直接的因果関係
		"原因", "結果", "影響", "effect", "cause", "result", "impact",
		// 因果接続詞
		"ので", "から", "ため", "よって", "because", "since", "due to", "leads to",
		// 条件関係
		"もし", "なら", "場合", "if", "when", "provided that", "given that",
	}

	for _, pattern := range causalPatterns {
		count := strings.Count(lowerText, pattern)
		causalCount += count
	}

	return causalCount
}

// decomposeInferenceSteps はLLMを使用して推論ステップを分解
func (lsa *LogicalStructureAnalyzer) decomposeInferenceSteps(
	ctx context.Context,
	text string,
) ([]*InferenceStepAnalysis, error) {

	prompt := fmt.Sprintf(`
以下のテキストの推論ステップを分析し、各ステップを特定してください：

【テキスト】: %s

各推論ステップを以下の形式でJSON配列として返してください：
[
  {
    "step_number": 1,
    "type": "premise|inference|conclusion",
    "content": "ステップの内容",
    "logical_strength": 0.0-1.0,
    "connectors": ["使用された論理接続詞"],
    "abstraction_level": 1-5
  }
]

推論の流れを段階的に追跡し、各ステップの論理的強度と抽象レベルを評価してください。
`, text)

	request := &ai.GenerateRequest{
		Messages: []ai.Message{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
	}

	response, err := lsa.llmClient.GenerateResponse(ctx, request)
	if err != nil {
		return nil, err
	}

	// JSON応答をパース（実装では適切なJSONパーシングを行う）
	steps := lsa.parseInferenceStepsResponse(response.Content)
	return steps, nil
}

// simpleInferenceDecomposition は簡易推論分解（フォールバック用）
func (lsa *LogicalStructureAnalyzer) simpleInferenceDecomposition(text string) []*InferenceStepAnalysis {
	sentences := lsa.splitIntoSentences(text)
	steps := make([]*InferenceStepAnalysis, 0)

	for i, sentence := range sentences {
		stepType := "inference"
		if i == 0 {
			stepType = "premise"
		} else if i == len(sentences)-1 {
			stepType = "conclusion"
		}

		connectors := lsa.findConnectorsInSentence(sentence)
		abstractionLevel := lsa.evaluateSentenceAbstraction(sentence)

		step := &InferenceStepAnalysis{
			StepNumber:       i + 1,
			Type:             stepType,
			Content:          sentence,
			LogicalStrength:  0.7, // デフォルト値
			Dependencies:     []int{},
			Connectors:       connectors,
			AbstractionLevel: abstractionLevel,
		}

		steps = append(steps, step)
	}

	return steps
}

// analyzeArgumentStructure はToulminモデルに基づく論証構造分析
func (lsa *LogicalStructureAnalyzer) analyzeArgumentStructure(
	ctx context.Context,
	text string,
	originalQuery string,
) (*ArgumentStructure, error) {

	prompt := fmt.Sprintf(`
以下のテキストをToulmin論証モデルに基づいて分析してください：

【質問】: %s
【テキスト】: %s

以下の構成要素を特定し、JSON形式で返してください：
{
  "premises": ["前提1", "前提2"],
  "conclusions": ["結論1", "結論2"],
  "warrant_links": ["論拠となる推論規則"],
  "counter_arguments": ["反論や代替案"],
  "rebuttals": ["反駁"],
  "qualifiers": ["限定詞や条件"],
  "structural_depth": 論証構造の深度(1-5)
}

Toulminモデルの6要素（主張・根拠・論拠・支持・反駁・限定）を基に詳細に分析してください。
`, originalQuery, text)

	request := &ai.GenerateRequest{
		Messages: []ai.Message{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
	}

	response, err := lsa.llmClient.GenerateResponse(ctx, request)
	if err != nil {
		return nil, err
	}

	// JSON応答をパース
	structure := lsa.parseArgumentStructureResponse(response.Content)
	structure.ToulminScore = lsa.calculateToulminScore(structure)

	return structure, nil
}

// basicArgumentStructure は基本的な論証構造を生成（フォールバック用）
func (lsa *LogicalStructureAnalyzer) basicArgumentStructure(text string) *ArgumentStructure {
	sentences := lsa.splitIntoSentences(text)

	premises := []string{}
	conclusions := []string{}

	// 簡易分類
	for _, sentence := range sentences {
		if lsa.isPremise(sentence) {
			premises = append(premises, sentence)
		} else if lsa.isConclusion(sentence) {
			conclusions = append(conclusions, sentence)
		}
	}

	// デフォルトで最初の文を前提、最後の文を結論とする
	if len(premises) == 0 && len(sentences) > 0 {
		premises = append(premises, sentences[0])
	}
	if len(conclusions) == 0 && len(sentences) > 1 {
		conclusions = append(conclusions, sentences[len(sentences)-1])
	}

	return &ArgumentStructure{
		Premises:         premises,
		Conclusions:      conclusions,
		WarrantLinks:     []string{},
		CounterArguments: []string{},
		Rebuttals:        []string{},
		Qualifiers:       []string{},
		StructuralDepth:  minInt(3, len(sentences)),
		ToulminScore:     0.5, // デフォルトスコア
	}
}

// calculateComplexityScore は複雑性スコアを計算
func (lsa *LogicalStructureAnalyzer) calculateComplexityScore(
	connectors []string,
	chainLength, abstractionLevels, causalRelations int,
) float64 {
	// 各要素を0-1範囲に正規化
	connectorsNorm := minFloat(1.0, float64(len(connectors))/10.0)
	chainNorm := minFloat(1.0, float64(chainLength)/10.0)
	abstractionNorm := float64(abstractionLevels) / 5.0
	causalNorm := minFloat(1.0, float64(causalRelations)/10.0)

	// 重み付き平均
	complexity := 0.3*connectorsNorm + 0.3*chainNorm + 0.2*abstractionNorm + 0.2*causalNorm

	return complexity
}

// evaluateLogicalCoherence は論理的一貫性を評価
func (lsa *LogicalStructureAnalyzer) evaluateLogicalCoherence(
	steps []*InferenceStepAnalysis,
	structure *ArgumentStructure,
) float64 {
	if len(steps) == 0 {
		return 0.5
	}

	// ステップ間の論理的強度の平均
	totalStrength := 0.0
	for _, step := range steps {
		totalStrength += step.LogicalStrength
	}
	avgStepStrength := totalStrength / float64(len(steps))

	// 論証構造のスコア
	structureScore := structure.ToulminScore

	// 統合スコア
	coherence := (avgStepStrength + structureScore) / 2.0

	return coherence
}

// identifyReasoningPatterns は推論パターンを特定
func (lsa *LogicalStructureAnalyzer) identifyReasoningPatterns(
	text string,
	steps []*InferenceStepAnalysis,
) []string {
	patterns := []string{}
	lowerText := strings.ToLower(text)

	// 演繹的パターン
	if strings.Contains(lowerText, "一般的に") && strings.Contains(lowerText, "したがって") {
		patterns = append(patterns, "deductive_reasoning")
	}

	// 帰納的パターン
	if strings.Contains(lowerText, "例えば") && strings.Contains(lowerText, "一般的に") {
		patterns = append(patterns, "inductive_reasoning")
	}

	// 類推パターン
	if strings.Contains(lowerText, "同様に") || strings.Contains(lowerText, "analogous") {
		patterns = append(patterns, "analogical_reasoning")
	}

	// 因果推論パターン
	if strings.Contains(lowerText, "原因") && strings.Contains(lowerText, "結果") {
		patterns = append(patterns, "causal_reasoning")
	}

	// 段階的推論パターン
	if len(steps) > 3 {
		patterns = append(patterns, "multi_step_reasoning")
	}

	return patterns
}

// calculateOverallDepth は総合的推論深度を計算
func (lsa *LogicalStructureAnalyzer) calculateOverallDepth(
	connectors, chainLength, abstractionLevels, causalRelations,
	inferenceSteps, structuralDepth int,
) int {
	// 各要素の重み付きスコア
	connectorsScore := minInt(3, connectors/2)
	chainScore := minInt(4, chainLength)
	abstractionScore := minInt(3, abstractionLevels-1)
	causalScore := minInt(2, causalRelations/2)
	stepsScore := minInt(3, inferenceSteps/2)
	structureScore := minInt(3, structuralDepth)

	// 総合深度（最大10）
	totalDepth := connectorsScore + chainScore + abstractionScore +
		causalScore + stepsScore + structureScore

	// 1-10の範囲に制限
	return maxInt(1, minInt(10, totalDepth))
}

// Helper functions

func (lsa *LogicalStructureAnalyzer) splitIntoSentences(text string) []string {
	// 日本語と英語の文境界を検出
	re := regexp.MustCompile(`[.!?。！？]\s+`)
	sentences := re.Split(text, -1)

	// 空文字列を除去
	var result []string
	for _, sentence := range sentences {
		trimmed := strings.TrimSpace(sentence)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func (lsa *LogicalStructureAnalyzer) findConnectorsInSentence(sentence string) []string {
	return lsa.extractLogicalConnectors(sentence)
}

func (lsa *LogicalStructureAnalyzer) evaluateSentenceAbstraction(sentence string) int {
	return lsa.analyzeAbstractionLevels(sentence)
}

func (lsa *LogicalStructureAnalyzer) isPremise(sentence string) bool {
	lowerSentence := strings.ToLower(sentence)
	premiseMarkers := []string{"なぜなら", "given that", "assuming", "前提として"}

	for _, marker := range premiseMarkers {
		if strings.Contains(lowerSentence, marker) {
			return true
		}
	}

	return false
}

func (lsa *LogicalStructureAnalyzer) isConclusion(sentence string) bool {
	lowerSentence := strings.ToLower(sentence)
	conclusionMarkers := []string{"したがって", "結論として", "therefore", "in conclusion"}

	for _, marker := range conclusionMarkers {
		if strings.Contains(lowerSentence, marker) {
			return true
		}
	}

	return false
}

func (lsa *LogicalStructureAnalyzer) calculateToulminScore(structure *ArgumentStructure) float64 {
	score := 0.0

	// 各要素の存在による加点
	if len(structure.Premises) > 0 {
		score += 0.2
	}
	if len(structure.Conclusions) > 0 {
		score += 0.2
	}
	if len(structure.WarrantLinks) > 0 {
		score += 0.15
	}
	if len(structure.CounterArguments) > 0 {
		score += 0.15
	}
	if len(structure.Rebuttals) > 0 {
		score += 0.15
	}
	if len(structure.Qualifiers) > 0 {
		score += 0.15
	}

	return score
}

// JSON parsing helpers (simplified implementations)
func (lsa *LogicalStructureAnalyzer) parseInferenceStepsResponse(response string) []*InferenceStepAnalysis {
	// 簡易実装 - 実際にはJSONパーサーを使用
	return []*InferenceStepAnalysis{}
}

func (lsa *LogicalStructureAnalyzer) parseArgumentStructureResponse(response string) *ArgumentStructure {
	// 簡易実装 - 実際にはJSONパーサーを使用
	return &ArgumentStructure{
		Premises:         []string{},
		Conclusions:      []string{},
		WarrantLinks:     []string{},
		CounterArguments: []string{},
		Rebuttals:        []string{},
		Qualifiers:       []string{},
		StructuralDepth:  1,
	}
}

// initializeLogicalPatterns は論理パターンデータベースを初期化
func initializeLogicalPatterns() *LogicalPatternDatabase {
	return &LogicalPatternDatabase{
		ConnectorPatterns: []LogicalPattern{
			{Name: "causal", Patterns: []string{"because", "なぜなら"}, Weight: 1.0, Category: "causal", Complexity: 2},
			{Name: "consequential", Patterns: []string{"therefore", "したがって"}, Weight: 1.0, Category: "logical", Complexity: 3},
		},
		InferencePatterns: []LogicalPattern{
			{Name: "deductive", Patterns: []string{"all", "すべて"}, Weight: 0.8, Category: "deductive", Complexity: 3},
			{Name: "inductive", Patterns: []string{"example", "例えば"}, Weight: 0.7, Category: "inductive", Complexity: 2},
		},
		CausalPatterns: []LogicalPattern{
			{Name: "cause_effect", Patterns: []string{"cause", "原因"}, Weight: 1.0, Category: "causal", Complexity: 2},
		},
		AbstractionMarkers: []string{"概念", "理論", "原理", "concept", "theory", "principle"},
	}
}

// Utility functions
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
