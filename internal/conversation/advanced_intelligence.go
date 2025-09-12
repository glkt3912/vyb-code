package conversation

import (
	"fmt"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/config"
)

// 高度なインテリジェント応答システム
type AdvancedIntelligenceEngine struct {
	lightProactive      *LightweightProactiveManager
	contextEngine       *ContextSuggestionEngine
	executionEngine     *ExecutionEngine // 実行型応答エンジン
	config              *config.Config
	enabled             bool
	conversationHistory []ConversationTurn
	userPatterns        *UserPatterns
	domainKnowledge     *DomainKnowledge
}

// 会話ターン
type ConversationTurn struct {
	UserInput    string                 `json:"user_input"`
	AIResponse   string                 `json:"ai_response"`
	Timestamp    time.Time              `json:"timestamp"`
	Context      *ConversationContext   `json:"context"`
	Satisfaction float64                `json:"satisfaction"` // 0.0-1.0
	ResponseTime time.Duration          `json:"response_time"`
	Enhanced     bool                   `json:"enhanced"` // プロアクティブ拡張されたか
	Metadata     map[string]interface{} `json:"metadata"`
}

// 会話コンテキスト
type ConversationContext struct {
	Intent         string           `json:"intent"`     // "question", "request", "problem_solving", "exploration"
	Domain         string           `json:"domain"`     // "code", "architecture", "debugging", "optimization"
	Complexity     string           `json:"complexity"` // "simple", "medium", "complex", "expert"
	ProjectInfo    *ProjectSnapshot `json:"project_info"`
	RelevantFiles  []string         `json:"relevant_files"`
	Keywords       []string         `json:"keywords"`
	Sentiment      string           `json:"sentiment"` // "positive", "neutral", "frustrated", "urgent"
	PreviousTopics []string         `json:"previous_topics"`
}

// プロジェクトスナップショット
type ProjectSnapshot struct {
	Language      string    `json:"language"`
	TechStack     []string  `json:"tech_stack"`
	FileCount     int       `json:"file_count"`
	RecentChanges int       `json:"recent_changes"`
	HealthScore   float64   `json:"health_score"`
	LastAnalysis  time.Time `json:"last_analysis"`
	ActiveBranch  string    `json:"active_branch"`
	Issues        []string  `json:"issues"`
	Opportunities []string  `json:"opportunities"`
}

// ユーザーパターン学習
type UserPatterns struct {
	PreferredResponseStyle string         `json:"preferred_style"` // "concise", "detailed", "step_by_step"
	CommonTopics           []string       `json:"common_topics"`
	QuestionPatterns       []string       `json:"question_patterns"`
	TechnicalLevel         string         `json:"technical_level"` // "beginner", "intermediate", "advanced", "expert"
	PreferredLanguages     []string       `json:"preferred_languages"`
	InteractionFrequency   map[string]int `json:"interaction_frequency"`
	AverageSessionLength   time.Duration  `json:"average_session_length"`
	PreferredFeatures      []string       `json:"preferred_features"`
	Learning               bool           `json:"learning"` // 学習モード有効/無効
}

// ドメイン知識ベース
type DomainKnowledge struct {
	CodePatterns        map[string][]CodePattern        `json:"code_patterns"`
	BestPractices       map[string][]BestPractice       `json:"best_practices"`
	CommonProblems      map[string][]CommonProblem      `json:"common_problems"`
	ArchitectureGuides  map[string][]ArchitectureGuide  `json:"architecture_guides"`
	ToolRecommendations map[string][]ToolRecommendation `json:"tool_recommendations"`
}

// コードパターン
type CodePattern struct {
	Name         string   `json:"name"`
	Language     string   `json:"language"`
	Pattern      string   `json:"pattern"`
	Description  string   `json:"description"`
	UseCase      string   `json:"use_case"`
	Examples     []string `json:"examples"`
	Alternatives []string `json:"alternatives"`
	Complexity   string   `json:"complexity"`
}

// ベストプラクティス
type BestPractice struct {
	Title       string   `json:"title"`
	Domain      string   `json:"domain"` // "code", "testing", "deployment", "security"
	Language    string   `json:"language"`
	Description string   `json:"description"`
	DoList      []string `json:"do_list"`
	DontList    []string `json:"dont_list"`
	Examples    []string `json:"examples"`
	Priority    string   `json:"priority"` // "critical", "important", "recommended", "optional"
}

// 一般的な問題
type CommonProblem struct {
	Title      string   `json:"title"`
	Category   string   `json:"category"`
	Symptoms   []string `json:"symptoms"`
	Causes     []string `json:"causes"`
	Solutions  []string `json:"solutions"`
	Prevention []string `json:"prevention"`
	Difficulty string   `json:"difficulty"`
	Frequency  float64  `json:"frequency"` // 問題の発生頻度
}

// アーキテクチャガイド
type ArchitectureGuide struct {
	Pattern     string   `json:"pattern"`
	Scale       string   `json:"scale"` // "small", "medium", "large", "enterprise"
	Description string   `json:"description"`
	Pros        []string `json:"pros"`
	Cons        []string `json:"cons"`
	WhenToUse   []string `json:"when_to_use"`
	Examples    []string `json:"examples"`
	Complexity  int      `json:"complexity"` // 1-10
}

// ツール推奨
type ToolRecommendation struct {
	Tool         string   `json:"tool"`
	Category     string   `json:"category"` // "testing", "build", "deployment", "monitoring"
	Language     string   `json:"language"`
	Purpose      string   `json:"purpose"`
	Pros         []string `json:"pros"`
	Cons         []string `json:"cons"`
	Alternatives []string `json:"alternatives"`
	Setup        []string `json:"setup"`
	Rating       float64  `json:"rating"` // 0.0-5.0
}

// 新しい高度なインテリジェンスエンジンを作成
func NewAdvancedIntelligenceEngine(cfg *config.Config, projectPath string) *AdvancedIntelligenceEngine {
	if !cfg.IsProactiveEnabled() {
		return &AdvancedIntelligenceEngine{enabled: false}
	}

	return &AdvancedIntelligenceEngine{
		lightProactive:      NewLightweightProactiveManager(cfg),
		contextEngine:       NewContextSuggestionEngine(cfg),
		executionEngine:     NewExecutionEngine(cfg, projectPath), // 実行エンジン追加
		config:              cfg,
		enabled:             true,
		conversationHistory: make([]ConversationTurn, 0),
		userPatterns:        NewUserPatterns(),
		domainKnowledge:     NewDomainKnowledge(),
	}
}

// 新しいユーザーパターンを作成
func NewUserPatterns() *UserPatterns {
	return &UserPatterns{
		PreferredResponseStyle: "balanced",
		CommonTopics:           make([]string, 0),
		QuestionPatterns:       make([]string, 0),
		TechnicalLevel:         "intermediate",
		PreferredLanguages:     make([]string, 0),
		InteractionFrequency:   make(map[string]int),
		AverageSessionLength:   15 * time.Minute,
		PreferredFeatures:      make([]string, 0),
		Learning:               true,
	}
}

// 新しいドメイン知識を作成
func NewDomainKnowledge() *DomainKnowledge {
	dk := &DomainKnowledge{
		CodePatterns:        make(map[string][]CodePattern),
		BestPractices:       make(map[string][]BestPractice),
		CommonProblems:      make(map[string][]CommonProblem),
		ArchitectureGuides:  make(map[string][]ArchitectureGuide),
		ToolRecommendations: make(map[string][]ToolRecommendation),
	}

	// 初期知識ベースを構築
	dk.initializeKnowledgeBase()
	return dk
}

// 知識ベースの初期化
func (dk *DomainKnowledge) initializeKnowledgeBase() {
	// Go言語のベストプラクティス
	dk.BestPractices["go"] = []BestPractice{
		{
			Title:       "エラーハンドリング",
			Domain:      "code",
			Language:    "Go",
			Description: "Goにおける適切なエラーハンドリングの実践",
			DoList: []string{
				"エラーを明示的にチェック",
				"エラーメッセージを詳細に",
				"カスタムエラー型を活用",
			},
			DontList: []string{
				"エラーを無視しない",
				"panic()を多用しない",
			},
			Priority: "critical",
		},
		{
			Title:       "並行処理",
			Domain:      "code",
			Language:    "Go",
			Description: "Goのgoroutineとchannelを使った効率的な並行処理",
			DoList: []string{
				"channelを使った通信",
				"context.Contextでのタイムアウト",
				"sync.WaitGroupでの同期",
			},
			Priority: "important",
		},
	}

	// JavaScript/TypeScript のベストプラクティス
	dk.BestPractices["javascript"] = []BestPractice{
		{
			Title:       "非同期処理",
			Domain:      "code",
			Language:    "JavaScript",
			Description: "Promise、async/awaitを使った適切な非同期処理",
			DoList: []string{
				"async/awaitの使用",
				"エラーハンドリング（try-catch）",
				"Promise.allで並行処理",
			},
			Priority: "critical",
		},
	}

	// 一般的な問題
	dk.CommonProblems["performance"] = []CommonProblem{
		{
			Title:      "メモリリーク",
			Category:   "performance",
			Symptoms:   []string{"メモリ使用量の継続的増加", "アプリケーションの動作の遅延"},
			Causes:     []string{"未解放のリソース", "循環参照", "大量のデータのキャッシュ"},
			Solutions:  []string{"プロファイリングツールの使用", "ガベージコレクション監視", "リソースの適切な解放"},
			Difficulty: "medium",
			Frequency:  0.3,
		},
	}

	// アーキテクチャガイド
	dk.ArchitectureGuides["microservices"] = []ArchitectureGuide{
		{
			Pattern:     "Microservices",
			Scale:       "large",
			Description: "独立してデプロイ可能な小さなサービスの集合",
			Pros:        []string{"スケーラビリティ", "技術多様性", "独立したデプロイ"},
			Cons:        []string{"複雑性の増加", "ネットワーク遅延", "デバッグの困難さ"},
			WhenToUse:   []string{"大規模チーム", "高トラフィック", "多様な技術要件"},
			Complexity:  8,
		},
	}
}

// 高度なインテリジェント応答の生成
func (aie *AdvancedIntelligenceEngine) GenerateEnhancedResponse(originalResponse, userInput, projectPath string) (string, error) {
	if !aie.enabled {
		return originalResponse, nil
	}

	// 実行型応答の優先処理
	if aie.executionEngine != nil {
		// ユーザーの実際の入力を抽出（プロジェクトコンテキストから）
		actualUserInput := aie.extractActualUserInput(userInput)

		analysis := aie.executionEngine.AnalyzeUserIntent(actualUserInput)

		// 実行可能なアクションが特定された場合の実行処理
		if analysis.SafeToExecute && (analysis.RequiredAction == "execute_git_command" ||
			analysis.RequiredAction == "explore_files" ||
			analysis.RequiredAction == "analyze_project" ||
			analysis.RequiredAction == "execute_explicit_command" ||
			analysis.RequiredAction == "read_file" ||
			analysis.RequiredAction == "search_files" ||
			analysis.RequiredAction == "write_file" ||
			analysis.RequiredAction == "execute_multi_tool_workflow") {

			result, err := aie.executionEngine.ExecuteCommand(analysis.SuggestedCommand)
			if err == nil && result != nil {
				// 実行結果をフォーマット
				executionResponse := aie.executionEngine.FormatExecutionResult(result, analysis)

				// 実行に成功した場合は実行結果を返す（説明モード回避）
				aie.recordExecutionTurn(actualUserInput, executionResponse, analysis)
				return executionResponse, nil
			}
		}
	}

	// 会話コンテキストを分析
	context, err := aie.analyzeConversationContext(userInput, projectPath)
	if err != nil {
		// エラー時は元の応答を返す
		return originalResponse, nil
	}

	// プロジェクトスナップショットを作成
	snapshot, err := aie.createProjectSnapshot(projectPath)
	if err != nil {
		snapshot = &ProjectSnapshot{} // 空のスナップショット
	}
	context.ProjectInfo = snapshot

	// 応答を拡張
	enhancedResponse := aie.enhanceWithIntelligence(originalResponse, context)

	// 会話履歴に記録
	turn := ConversationTurn{
		UserInput:  userInput,
		AIResponse: originalResponse,
		Timestamp:  time.Now(),
		Context:    context,
		Enhanced:   true,
		Metadata:   make(map[string]interface{}),
	}
	aie.addToHistory(turn)

	// ユーザーパターンを学習
	if aie.userPatterns.Learning {
		aie.learnUserPatterns(userInput, context)
	}

	return enhancedResponse, nil
}

// 会話コンテキストの分析
func (aie *AdvancedIntelligenceEngine) analyzeConversationContext(userInput, projectPath string) (*ConversationContext, error) {
	context := &ConversationContext{
		Keywords:       make([]string, 0),
		PreviousTopics: make([]string, 0),
		RelevantFiles:  make([]string, 0),
	}

	// 意図の分析
	context.Intent = aie.analyzeIntent(userInput)

	// ドメインの特定
	context.Domain = aie.identifyDomain(userInput)

	// 複雑さの評価
	context.Complexity = aie.assessComplexity(userInput)

	// 感情の分析
	context.Sentiment = aie.analyzeSentiment(userInput)

	// キーワード抽出
	context.Keywords = aie.extractKeywords(userInput)

	// 過去の会話トピックを参照
	if len(aie.conversationHistory) > 0 {
		recentTopics := aie.extractRecentTopics(5) // 直近5回の会話
		context.PreviousTopics = recentTopics
	}

	return context, nil
}

// 意図の分析
func (aie *AdvancedIntelligenceEngine) analyzeIntent(userInput string) string {
	inputLower := strings.ToLower(userInput)

	// 質問パターン
	if strings.Contains(inputLower, "?") ||
		strings.HasPrefix(inputLower, "what") ||
		strings.HasPrefix(inputLower, "how") ||
		strings.HasPrefix(inputLower, "why") ||
		strings.HasPrefix(inputLower, "when") ||
		strings.HasPrefix(inputLower, "where") {
		return "question"
	}

	// 問題解決パターン
	if strings.Contains(inputLower, "error") ||
		strings.Contains(inputLower, "issue") ||
		strings.Contains(inputLower, "problem") ||
		strings.Contains(inputLower, "fix") ||
		strings.Contains(inputLower, "debug") {
		return "problem_solving"
	}

	// リクエストパターン
	if strings.Contains(inputLower, "help") ||
		strings.Contains(inputLower, "create") ||
		strings.Contains(inputLower, "generate") ||
		strings.Contains(inputLower, "implement") {
		return "request"
	}

	// 探索パターン
	if strings.Contains(inputLower, "explore") ||
		strings.Contains(inputLower, "understand") ||
		strings.Contains(inputLower, "learn") ||
		strings.Contains(inputLower, "explain") {
		return "exploration"
	}

	return "general"
}

// ドメインの特定
func (aie *AdvancedIntelligenceEngine) identifyDomain(userInput string) string {
	inputLower := strings.ToLower(userInput)

	// コード関連
	if strings.Contains(inputLower, "code") ||
		strings.Contains(inputLower, "function") ||
		strings.Contains(inputLower, "variable") ||
		strings.Contains(inputLower, "class") ||
		strings.Contains(inputLower, "method") {
		return "code"
	}

	// アーキテクチャ関連
	if strings.Contains(inputLower, "architecture") ||
		strings.Contains(inputLower, "design") ||
		strings.Contains(inputLower, "pattern") ||
		strings.Contains(inputLower, "structure") {
		return "architecture"
	}

	// デバッグ関連
	if strings.Contains(inputLower, "debug") ||
		strings.Contains(inputLower, "error") ||
		strings.Contains(inputLower, "bug") ||
		strings.Contains(inputLower, "trace") {
		return "debugging"
	}

	// 最適化関連
	if strings.Contains(inputLower, "optimize") ||
		strings.Contains(inputLower, "performance") ||
		strings.Contains(inputLower, "speed") ||
		strings.Contains(inputLower, "efficiency") {
		return "optimization"
	}

	return "general"
}

// 複雑さの評価
func (aie *AdvancedIntelligenceEngine) assessComplexity(userInput string) string {
	inputLower := strings.ToLower(userInput)

	// 高複雑度キーワード
	expertKeywords := []string{"architecture", "scalability", "distributed", "microservices", "optimization", "algorithm", "concurrent"}
	complexKeywords := []string{"implement", "design", "integrate", "refactor", "deploy", "security"}

	expertCount := 0
	complexCount := 0

	for _, keyword := range expertKeywords {
		if strings.Contains(inputLower, keyword) {
			expertCount++
		}
	}

	for _, keyword := range complexKeywords {
		if strings.Contains(inputLower, keyword) {
			complexCount++
		}
	}

	if expertCount >= 2 || strings.Contains(inputLower, "enterprise") {
		return "expert"
	}
	if expertCount >= 1 || complexCount >= 2 {
		return "complex"
	}
	if complexCount >= 1 || len(strings.Fields(userInput)) > 15 {
		return "medium"
	}

	return "simple"
}

// 感情の分析
func (aie *AdvancedIntelligenceEngine) analyzeSentiment(userInput string) string {
	inputLower := strings.ToLower(userInput)

	// 緊急/フラストレーション
	if strings.Contains(inputLower, "urgent") ||
		strings.Contains(inputLower, "asap") ||
		strings.Contains(inputLower, "stuck") ||
		strings.Contains(inputLower, "frustrated") ||
		strings.Contains(inputLower, "broken") {
		if strings.Contains(inputLower, "urgent") || strings.Contains(inputLower, "asap") {
			return "urgent"
		}
		return "frustrated"
	}

	// ポジティブ
	if strings.Contains(inputLower, "great") ||
		strings.Contains(inputLower, "awesome") ||
		strings.Contains(inputLower, "thanks") ||
		strings.Contains(inputLower, "perfect") {
		return "positive"
	}

	return "neutral"
}

// キーワード抽出
func (aie *AdvancedIntelligenceEngine) extractKeywords(userInput string) []string {
	// 技術キーワードを抽出
	techKeywords := []string{
		"go", "golang", "javascript", "typescript", "python", "rust", "java",
		"docker", "kubernetes", "api", "database", "redis", "postgres", "mysql",
		"test", "testing", "ci", "cd", "git", "github", "deploy", "deployment",
		"performance", "security", "architecture", "microservices", "monolith",
	}

	inputLower := strings.ToLower(userInput)
	keywords := make([]string, 0)

	for _, keyword := range techKeywords {
		if strings.Contains(inputLower, keyword) {
			keywords = append(keywords, keyword)
		}
	}

	return keywords
}

// プロジェクトスナップショットの作成
func (aie *AdvancedIntelligenceEngine) createProjectSnapshot(projectPath string) (*ProjectSnapshot, error) {
	snapshot := &ProjectSnapshot{
		LastAnalysis:  time.Now(),
		Issues:        make([]string, 0),
		Opportunities: make([]string, 0),
	}

	// 軽量分析を実行
	if aie.lightProactive != nil {
		analysis, err := aie.lightProactive.AnalyzeProjectLightly(projectPath)
		if err == nil && analysis != nil {
			snapshot.Language = analysis.Language
			snapshot.TechStack = make([]string, 0)

			for _, tech := range analysis.TechStack {
				if tech.Usage == "primary" {
					snapshot.TechStack = append(snapshot.TechStack, tech.Name)
				}
			}

			if analysis.FileStructure != nil {
				snapshot.FileCount = analysis.FileStructure.TotalFiles
			}
		}
	}

	return snapshot, nil
}

// インテリジェンス拡張
func (aie *AdvancedIntelligenceEngine) enhanceWithIntelligence(originalResponse string, context *ConversationContext) string {
	enhanced := originalResponse

	// コンテキストに応じた情報追加
	if context.Domain != "general" {
		enhanced += aie.addDomainSpecificInsights(context)
	}

	// 複雑度に応じた詳細レベル調整
	enhanced += aie.adjustDetailLevel(context)

	// 感情に応じた応答調整
	enhanced += aie.adjustForSentiment(context)

	// プロジェクト固有の提案追加
	if context.ProjectInfo != nil && context.ProjectInfo.Language != "" {
		enhanced += aie.addProjectSpecificSuggestions(context.ProjectInfo)
	}

	return enhanced
}

// ドメイン固有の洞察追加
func (aie *AdvancedIntelligenceEngine) addDomainSpecificInsights(context *ConversationContext) string {
	var insights strings.Builder

	switch context.Domain {
	case "code":
		if bestPractices, exists := aie.domainKnowledge.BestPractices[context.ProjectInfo.Language]; exists {
			for _, bp := range bestPractices {
				if bp.Priority == "critical" || bp.Priority == "important" {
					insights.WriteString(fmt.Sprintf("\n\n💡 **%s**: %s", bp.Title, bp.Description))
					break // 最初の重要なベストプラクティスのみ
				}
			}
		}

	case "debugging":
		insights.WriteString("\n\n🔍 **デバッグのヒント**: 問題を特定するため、まず再現手順を明確にし、ログを確認してください。")

	case "optimization":
		insights.WriteString("\n\n⚡ **最適化のアプローチ**: まずボトルネックを特定し、プロファイリングツールで測定してから改善を実施することをお勧めします。")
	}

	return insights.String()
}

// 詳細レベル調整
func (aie *AdvancedIntelligenceEngine) adjustDetailLevel(context *ConversationContext) string {
	switch context.Complexity {
	case "expert":
		return "\n\n🎓 **上級者向け**: より高度な実装パターンやパフォーマンス考慮事項については、お気軽にお聞きください。"
	case "simple":
		return "\n\n📚 **基本ガイド**: ステップバイステップの説明が必要でしたら、詳細に解説いたします。"
	}
	return ""
}

// 感情に応じた調整
func (aie *AdvancedIntelligenceEngine) adjustForSentiment(context *ConversationContext) string {
	switch context.Sentiment {
	case "urgent":
		return "\n\n⚡ **緊急対応**: 迅速な解決のため、最も効果的なアプローチを優先してご提案します。"
	case "frustrated":
		return "\n\n🤝 **サポート**: 問題解決をお手伝いします。一歩ずつ確実に進めていきましょう。"
	case "positive":
		return "\n\n🎉 **素晴らしい**: さらなる改善や新機能の実装についてもご相談ください！"
	}
	return ""
}

// プロジェクト固有提案
func (aie *AdvancedIntelligenceEngine) addProjectSpecificSuggestions(projectInfo *ProjectSnapshot) string {
	var suggestions strings.Builder

	if projectInfo.Language != "" {
		suggestions.WriteString(fmt.Sprintf("\n\n🔧 **%s プロジェクト**: ", projectInfo.Language))

		switch projectInfo.Language {
		case "Go":
			suggestions.WriteString("Go Modulesの最適化やgoroutineの効率的な利用についてもサポートできます。")
		case "JavaScript", "TypeScript":
			suggestions.WriteString("パッケージの最適化やバンドルサイズの削減についてもお手伝いできます。")
		case "Python":
			suggestions.WriteString("仮想環境の管理や依存関係の最適化についてもご相談ください。")
		}
	}

	return suggestions.String()
}

// 他のヘルパーメソッド...
func (aie *AdvancedIntelligenceEngine) addToHistory(turn ConversationTurn) {
	aie.conversationHistory = append(aie.conversationHistory, turn)

	// 履歴サイズ制限
	if len(aie.conversationHistory) > 50 {
		aie.conversationHistory = aie.conversationHistory[1:]
	}
}

// 実行型応答の履歴記録
func (aie *AdvancedIntelligenceEngine) recordExecutionTurn(userInput, executionResponse string, analysis *ContextAnalysis) {
	context := &ConversationContext{
		Intent:      analysis.Intent,
		Domain:      "execution",
		Complexity:  "simple", // 実行は基本的にシンプル
		Sentiment:   "neutral",
		Keywords:    []string{analysis.SuggestedCommand},
		ProjectInfo: nil, // 実行時は詳細分析を省略
	}

	turn := ConversationTurn{
		UserInput:  userInput,
		AIResponse: executionResponse,
		Timestamp:  time.Now(),
		Context:    context,
		Enhanced:   true,
		Metadata: map[string]interface{}{
			"execution_type": analysis.RequiredAction,
			"command":        analysis.SuggestedCommand,
			"safe_execution": analysis.SafeToExecute,
		},
	}

	aie.addToHistory(turn)
}

func (aie *AdvancedIntelligenceEngine) extractRecentTopics(count int) []string {
	if len(aie.conversationHistory) == 0 {
		return []string{}
	}

	topics := make([]string, 0)
	start := len(aie.conversationHistory) - count
	if start < 0 {
		start = 0
	}

	for i := start; i < len(aie.conversationHistory); i++ {
		if aie.conversationHistory[i].Context != nil {
			topics = append(topics, aie.conversationHistory[i].Context.Domain)
		}
	}

	return topics
}

func (aie *AdvancedIntelligenceEngine) learnUserPatterns(userInput string, context *ConversationContext) {
	// 技術レベル学習
	if context.Complexity == "expert" {
		aie.userPatterns.TechnicalLevel = "expert"
	} else if context.Complexity == "complex" && aie.userPatterns.TechnicalLevel != "expert" {
		aie.userPatterns.TechnicalLevel = "advanced"
	}

	// よく使われるトピックの学習
	if context.Domain != "general" {
		found := false
		for _, topic := range aie.userPatterns.CommonTopics {
			if topic == context.Domain {
				found = true
				break
			}
		}
		if !found {
			aie.userPatterns.CommonTopics = append(aie.userPatterns.CommonTopics, context.Domain)
		}
	}
}

// パフォーマンス統計
func (aie *AdvancedIntelligenceEngine) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":            aie.enabled,
		"conversation_count": len(aie.conversationHistory),
		"technical_level":    aie.userPatterns.TechnicalLevel,
		"common_topics":      aie.userPatterns.CommonTopics,
		"learning_enabled":   aie.userPatterns.Learning,
	}
}

// リソースクリーンアップ
func (aie *AdvancedIntelligenceEngine) Close() error {
	if aie.lightProactive != nil {
		aie.lightProactive.Close()
	}
	if aie.contextEngine != nil {
		aie.contextEngine.Close()
	}
	return nil
}

// ユーザーの実際の入力を抽出（プロジェクトコンテキストから）
func (aie *AdvancedIntelligenceEngine) extractActualUserInput(fullInput string) string {
	lines := strings.Split(fullInput, "\n")

	// "---" の後の部分を探す
	foundSeparator := false
	var userInputLines []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			foundSeparator = true
			continue
		}

		if foundSeparator {
			// 空行をスキップして実際のユーザー入力のみを取得
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				userInputLines = append(userInputLines, trimmed)
			}
		}
	}

	if len(userInputLines) > 0 {
		// 最後の非空行を実際のユーザー入力として返す
		return userInputLines[len(userInputLines)-1]
	}

	// フォールバック：元の入力をそのまま返す
	return fullInput
}
