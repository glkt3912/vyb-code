package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/security"
	"github.com/glkt/vyb-code/internal/tools"
)

// AI統合サービス - 各AIコンポーネントを統合して使いやすいインターフェースを提供
type AIService struct {
	codeAnalyzer     *CodeAnalyzer
	codeGenerator    *CodeGenerator
	visualizer       *DependencyVisualizer
	multiRepoManager *MultiRepoManager
	llmClient        LLMClient
	constraints      *security.Constraints
}

// 統合分析結果
type IntegratedAnalysisResult struct {
	CodeAnalysis      *CodeAnalysisResult        `json:"code_analysis"`
	Visualization     *DependencyVisualization   `json:"dependency_visualization"`
	WorkspaceAnalysis *WorkspaceAnalysis         `json:"workspace_analysis,omitempty"`
	GeneratedCode     *CodeGenerationResult      `json:"generated_code,omitempty"`
	Recommendations   []IntegratedRecommendation `json:"integrated_recommendations"`
	Summary           string                     `json:"summary"`
	ProcessingInfo    ProcessingInfo             `json:"processing_info"`
}

// 統合推奨事項
type IntegratedRecommendation struct {
	Category    string              `json:"category"` // "code_quality", "architecture", "security", "performance"
	Priority    string              `json:"priority"` // "low", "medium", "high", "critical"
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Sources     []string            `json:"sources"` // どのAIコンポーネントからの推奨か
	Actions     []RecommendedAction `json:"actions"`
	Benefits    string              `json:"benefits"`
	Effort      string              `json:"effort"`
}

// 推奨アクション
type RecommendedAction struct {
	Type        string `json:"type"` // "code_change", "refactor", "add_test", "add_doc"
	Description string `json:"description"`
	Priority    int    `json:"priority"`  // 実行順序
	Automated   bool   `json:"automated"` // 自動化可能か
}

// 処理情報
type ProcessingInfo struct {
	TotalProcessingTime  string            `json:"total_processing_time"`
	ComponentTimes       map[string]string `json:"component_times"`
	FilesAnalyzed        int               `json:"files_analyzed"`
	IssuesFound          int               `json:"issues_found"`
	SuggestionsGenerated int               `json:"suggestions_generated"`
}

// AIサービスリクエスト
type AIServiceRequest struct {
	ProjectPath          string                  `json:"project_path"`
	AnalysisType         string                  `json:"analysis_type"` // "basic", "detailed", "comprehensive"
	IncludeCodeGen       bool                    `json:"include_code_gen"`
	IncludeVisualization bool                    `json:"include_visualization"`
	IncludeMultiRepo     bool                    `json:"include_multi_repo"`
	CodeGenRequests      []CodeGenerationRequest `json:"code_gen_requests,omitempty"`
	CustomPrompts        map[string]string       `json:"custom_prompts,omitempty"`
}

// AIサービスを作成
func NewAIService(llmClient LLMClient, constraints *security.Constraints) *AIService {
	return &AIService{
		llmClient:   llmClient,
		constraints: constraints,
	}
}

// プロジェクト全体の統合分析を実行
func (ai *AIService) AnalyzeProject(ctx context.Context, request *AIServiceRequest) (*IntegratedAnalysisResult, error) {
	startTime := getCurrentTime()

	result := &IntegratedAnalysisResult{
		Recommendations: []IntegratedRecommendation{},
		ProcessingInfo: ProcessingInfo{
			ComponentTimes: make(map[string]string),
		},
	}

	// コード分析の実行
	if ai.codeAnalyzer == nil {
		ai.codeAnalyzer = NewCodeAnalyzer(ai.llmClient, ai.constraints, request.ProjectPath)
	}

	codeAnalysisStart := getCurrentTime()
	codeAnalysis, err := ai.codeAnalyzer.AnalyzeProject(ctx)
	if err != nil {
		return nil, fmt.Errorf("コード分析エラー: %w", err)
	}
	result.CodeAnalysis = codeAnalysis
	result.ProcessingInfo.ComponentTimes["code_analysis"] = formatDuration(getCurrentTime().Sub(codeAnalysisStart))
	result.ProcessingInfo.IssuesFound = len(codeAnalysis.Issues)

	// 依存関係可視化の実行
	if request.IncludeVisualization {
		if ai.visualizer == nil {
			ai.visualizer = NewDependencyVisualizer(ai.constraints, request.ProjectPath)
		}

		visualizationStart := getCurrentTime()
		visualization, err := ai.visualizer.VisualizeProject(ctx)
		if err != nil {
			fmt.Printf("依存関係可視化警告: %v\n", err)
		} else {
			result.Visualization = visualization
			result.ProcessingInfo.ComponentTimes["visualization"] = formatDuration(getCurrentTime().Sub(visualizationStart))
		}
	}

	// マルチリポジトリ分析の実行
	if request.IncludeMultiRepo {
		if ai.multiRepoManager == nil {
			ai.multiRepoManager = NewMultiRepoManager(ai.constraints, request.ProjectPath)
		}

		multiRepoStart := getCurrentTime()
		workspaceAnalysis, err := ai.multiRepoManager.AnalyzeWorkspace(ctx)
		if err != nil {
			fmt.Printf("ワークスペース分析警告: %v\n", err)
		} else {
			result.WorkspaceAnalysis = workspaceAnalysis
			result.ProcessingInfo.ComponentTimes["multi_repo"] = formatDuration(getCurrentTime().Sub(multiRepoStart))
		}
	}

	// コード生成の実行
	if request.IncludeCodeGen && len(request.CodeGenRequests) > 0 {
		if ai.codeGenerator == nil {
			ai.codeGenerator = NewCodeGenerator(ai.llmClient, ai.constraints, request.ProjectPath)
		}

		codeGenStart := getCurrentTime()
		for _, genRequest := range request.CodeGenRequests {
			genResult, err := ai.codeGenerator.GenerateCode(ctx, &genRequest)
			if err != nil {
				fmt.Printf("コード生成警告: %v\n", err)
				continue
			}
			result.GeneratedCode = genResult // 最後の結果を保存（複数対応は将来の拡張）
		}
		result.ProcessingInfo.ComponentTimes["code_generation"] = formatDuration(getCurrentTime().Sub(codeGenStart))
	}

	// 統合推奨事項の生成
	result.Recommendations = ai.generateIntegratedRecommendations(result)
	result.ProcessingInfo.SuggestionsGenerated = len(result.Recommendations)

	// 統合要約の生成
	summaryStart := getCurrentTime()
	summary, err := ai.generateIntegratedSummary(ctx, result)
	if err != nil {
		fmt.Printf("要約生成警告: %v\n", err)
		result.Summary = "要約生成に失敗しました"
	} else {
		result.Summary = summary
	}
	result.ProcessingInfo.ComponentTimes["summary"] = formatDuration(getCurrentTime().Sub(summaryStart))

	// 処理情報の完了
	result.ProcessingInfo.TotalProcessingTime = formatDuration(getCurrentTime().Sub(startTime))
	result.ProcessingInfo.FilesAnalyzed = ai.countAnalyzedFiles(result)

	return result, nil
}

// 統合推奨事項を生成
func (ai *AIService) generateIntegratedRecommendations(result *IntegratedAnalysisResult) []IntegratedRecommendation {
	var recommendations []IntegratedRecommendation

	// コード分析からの推奨事項
	if result.CodeAnalysis != nil {
		// 重要な問題を推奨事項に変換
		criticalIssues := ai.filterCriticalIssues(result.CodeAnalysis.Issues)
		for _, issue := range criticalIssues {
			recommendation := IntegratedRecommendation{
				Category:    "code_quality",
				Priority:    issue.Severity,
				Title:       issue.Title,
				Description: issue.Description,
				Sources:     []string{"code_analysis"},
				Actions: []RecommendedAction{
					{
						Type:        "code_change",
						Description: issue.Solution,
						Priority:    1,
						Automated:   false,
					},
				},
				Benefits: "コード品質と信頼性の向上",
				Effort:   ai.estimateEffort(issue.Severity),
			}
			recommendations = append(recommendations, recommendation)
		}

		// パフォーマンス改善提案
		for _, insight := range result.CodeAnalysis.Performance {
			if insight.Impact == "high" {
				recommendation := IntegratedRecommendation{
					Category:    "performance",
					Priority:    "high",
					Title:       "パフォーマンス最適化",
					Description: insight.Suggestion,
					Sources:     []string{"code_analysis"},
					Actions: []RecommendedAction{
						{
							Type:        "refactor",
							Description: insight.Suggestion,
							Priority:    2,
							Automated:   false,
						},
					},
					Benefits: insight.EstimatedGain,
					Effort:   "medium",
				}
				recommendations = append(recommendations, recommendation)
			}
		}
	}

	// 可視化からの推奨事項
	if result.Visualization != nil {
		// 循環依存の解決
		if result.Visualization.Metrics.CircularDependencies > 0 {
			recommendation := IntegratedRecommendation{
				Category:    "architecture",
				Priority:    "high",
				Title:       "循環依存の解決",
				Description: fmt.Sprintf("%d個の循環依存が検出されました。これらを解決してアーキテクチャを改善してください。", result.Visualization.Metrics.CircularDependencies),
				Sources:     []string{"dependency_visualization"},
				Actions: []RecommendedAction{
					{
						Type:        "refactor",
						Description: "インターフェースやDIパターンを使用して循環依存を解消",
						Priority:    1,
						Automated:   false,
					},
				},
				Benefits: "アーキテクチャの改善、保守性の向上",
				Effort:   "large",
			}
			recommendations = append(recommendations, recommendation)
		}

		// ホットスポットの改善
		if len(result.Visualization.Metrics.Hotspots) > 0 {
			recommendation := IntegratedRecommendation{
				Category:    "architecture",
				Priority:    "medium",
				Title:       "ホットスポットの分散",
				Description: "高い結合度を持つコンポーネントが検出されました。責任を分散することを検討してください。",
				Sources:     []string{"dependency_visualization"},
				Actions: []RecommendedAction{
					{
						Type:        "refactor",
						Description: "大きなコンポーネントを小さな単位に分割",
						Priority:    2,
						Automated:   false,
					},
				},
				Benefits: "保守性とテスタビリティの向上",
				Effort:   "medium",
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	// ワークスペース分析からの推奨事項
	if result.WorkspaceAnalysis != nil {
		for _, wsRec := range result.WorkspaceAnalysis.Recommendations {
			recommendation := IntegratedRecommendation{
				Category:    wsRec.Category,
				Priority:    wsRec.Priority,
				Title:       wsRec.Title,
				Description: wsRec.Description,
				Sources:     []string{"workspace_analysis"},
				Actions: []RecommendedAction{
					{
						Type:        "process_improvement",
						Description: strings.Join(wsRec.ActionItems, "; "),
						Priority:    1,
						Automated:   false,
					},
				},
				Benefits: wsRec.Benefits,
				Effort:   wsRec.Effort,
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	// 優先度でソート
	ai.sortRecommendationsByPriority(recommendations)

	return recommendations
}

// 重要な問題をフィルタ
func (ai *AIService) filterCriticalIssues(issues []CodeIssue) []CodeIssue {
	var criticalIssues []CodeIssue

	for _, issue := range issues {
		if issue.Severity == "critical" || issue.Severity == "high" {
			criticalIssues = append(criticalIssues, issue)
		}
	}

	return criticalIssues
}

// 工数を見積もり
func (ai *AIService) estimateEffort(severity string) string {
	switch severity {
	case "critical":
		return "large"
	case "high":
		return "medium"
	case "medium":
		return "small"
	default:
		return "small"
	}
}

// 推奨事項を優先度でソート
func (ai *AIService) sortRecommendationsByPriority(recommendations []IntegratedRecommendation) {
	priorityOrder := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
	}

	for i := 0; i < len(recommendations)-1; i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if priorityOrder[recommendations[i].Priority] < priorityOrder[recommendations[j].Priority] {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}
}

// 統合要約を生成
func (ai *AIService) generateIntegratedSummary(ctx context.Context, result *IntegratedAnalysisResult) (string, error) {
	summaryPrompt := ai.buildSummaryPrompt(result)

	response, err := ai.llmClient.GenerateResponse(ctx, &GenerateRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: summaryPrompt,
			},
		},
		Temperature: 0.3,
	})

	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// 要約プロンプトを構築
func (ai *AIService) buildSummaryPrompt(result *IntegratedAnalysisResult) string {
	prompt := `あなたはソフトウェア開発の専門家です。以下の包括的なプロジェクト分析結果に基づいて、統合的な要約レポートを日本語で作成してください：

## 分析結果

### コード分析結果:`

	if result.CodeAnalysis != nil {
		prompt += fmt.Sprintf(`
- 検出された問題: %d件
- 改善提案: %d件  
- 品質スコア: %d/100
- セキュリティスコア: %d/100`,
			len(result.CodeAnalysis.Issues),
			len(result.CodeAnalysis.Suggestions),
			result.CodeAnalysis.Quality.OverallScore,
			result.CodeAnalysis.Security.SecurityScore)
	}

	if result.Visualization != nil {
		prompt += fmt.Sprintf(`

### 依存関係分析結果:
- 総ノード数: %d個
- 総エッジ数: %d個
- 循環依存: %d個
- 最大深度: %d
- ホットスポット: %d個`,
			len(result.Visualization.Nodes),
			len(result.Visualization.Edges),
			result.Visualization.Metrics.CircularDependencies,
			result.Visualization.Metrics.MaxDepth,
			len(result.Visualization.Metrics.Hotspots))
	}

	if result.WorkspaceAnalysis != nil {
		prompt += fmt.Sprintf(`

### ワークスペース分析結果:
- 総リポジトリ数: %d個
- アクティブリポジトリ: %d個
- 品質スコア: %d/100
- セキュリティスコア: %d/100`,
			result.WorkspaceAnalysis.Overview.TotalRepositories,
			result.WorkspaceAnalysis.Overview.ActiveRepositories,
			result.WorkspaceAnalysis.QualityMetrics.OverallScore,
			result.WorkspaceAnalysis.SecurityAssessment.SecurityScore)
	}

	prompt += fmt.Sprintf(`

### 統合推奨事項:
- 総推奨事項数: %d件
- 重要度別内訳: %s

## 要求事項

以下の形式で包括的な要約を作成してください：

1. **全体的な評価** - プロジェクトの健康状態の総合判断
2. **主要な発見事項** - 最も重要な問題や特徴
3. **優先対応事項** - 緊急性の高い改善ポイント（TOP3）
4. **長期的改善目標** - 持続的な品質向上のための戦略
5. **次のステップ** - 具体的な行動計画

レポートは開発チームが理解しやすく、実行可能な内容にしてください。`,
		len(result.Recommendations),
		ai.categorizeRecommendations(result.Recommendations))

	return prompt
}

// 推奨事項を分類
func (ai *AIService) categorizeRecommendations(recommendations []IntegratedRecommendation) string {
	counts := make(map[string]int)

	for _, rec := range recommendations {
		counts[rec.Priority]++
	}

	return fmt.Sprintf("Critical: %d, High: %d, Medium: %d, Low: %d",
		counts["critical"], counts["high"], counts["medium"], counts["low"])
}

// 分析されたファイル数をカウント
func (ai *AIService) countAnalyzedFiles(result *IntegratedAnalysisResult) int {
	if result.CodeAnalysis != nil {
		// 問題が見つかったファイル数を基準にする
		fileSet := make(map[string]bool)
		for _, issue := range result.CodeAnalysis.Issues {
			if issue.File != "" {
				fileSet[issue.File] = true
			}
		}
		for _, suggestion := range result.CodeAnalysis.Suggestions {
			if suggestion.File != "" {
				fileSet[suggestion.File] = true
			}
		}
		return len(fileSet)
	}

	if result.Visualization != nil {
		return result.Visualization.TotalFiles
	}

	return 0
}

// インタラクティブな改善提案
func (ai *AIService) GetInteractiveRecommendations(ctx context.Context, projectPath string, userQuery string) (*InteractiveResponse, error) {
	// プロジェクト情報を取得
	analyzer := NewCodeAnalyzer(ai.llmClient, ai.constraints, projectPath)
	analysis, err := analyzer.AnalyzeProject(ctx)
	if err != nil {
		return nil, fmt.Errorf("プロジェクト分析エラー: %w", err)
	}

	// ユーザーの質問に基づいてカスタマイズされた回答を生成
	prompt := fmt.Sprintf(`あなたはAIコーディングアシスタントです。以下のプロジェクト分析結果を基に、ユーザーの質問に答えてください：

プロジェクト分析結果:
- 検出された問題: %d件
- 改善提案: %d件
- 品質スコア: %d/100

ユーザーの質問: %s

具体的で実用的な回答を日本語で提供してください。必要に応じてコード例や具体的な手順を含めてください。`,
		len(analysis.Issues),
		len(analysis.Suggestions),
		analysis.Quality.OverallScore,
		userQuery)

	response, err := ai.llmClient.GenerateResponse(ctx, &GenerateRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.4,
	})

	if err != nil {
		return nil, err
	}

	return &InteractiveResponse{
		Response:    response.Content,
		ProjectInfo: ai.summarizeProjectInfo(analysis),
		Suggestions: ai.extractRelevantSuggestions(analysis, userQuery),
	}, nil
}

// インタラクティブレスポンス
type InteractiveResponse struct {
	Response    string             `json:"response"`
	ProjectInfo ProjectInfoSummary `json:"project_info"`
	Suggestions []CodeSuggestion   `json:"relevant_suggestions"`
}

// プロジェクト情報サマリー
type ProjectInfoSummary struct {
	QualityScore  int `json:"quality_score"`
	IssueCount    int `json:"issue_count"`
	SecurityScore int `json:"security_score"`
}

// プロジェクト情報を要約
func (ai *AIService) summarizeProjectInfo(analysis *CodeAnalysisResult) ProjectInfoSummary {
	return ProjectInfoSummary{
		QualityScore:  analysis.Quality.OverallScore,
		IssueCount:    len(analysis.Issues),
		SecurityScore: analysis.Security.SecurityScore,
	}
}

// 関連する提案を抽出
func (ai *AIService) extractRelevantSuggestions(analysis *CodeAnalysisResult, query string) []CodeSuggestion {
	var relevant []CodeSuggestion
	queryLower := strings.ToLower(query)

	for _, suggestion := range analysis.Suggestions {
		if strings.Contains(strings.ToLower(suggestion.Title), queryLower) ||
			strings.Contains(strings.ToLower(suggestion.Description), queryLower) {
			relevant = append(relevant, suggestion)
		}
	}

	// 最大5件まで
	if len(relevant) > 5 {
		relevant = relevant[:5]
	}

	return relevant
}

// Vyb-codeツールとの統合
func (ai *AIService) IntegrateWithTools(toolRegistry *tools.ToolRegistry) error {
	// TODO: Implement tool registration when ToolRegistry has RegisterTool method
	// For now, this is a placeholder for future integration

	// カスタムツールとしてAI機能を登録する予定
	// プロジェクト分析ツール: "ai_analyze"
	// コード生成ツール: "ai_generate"
	// 対話的推奨ツール: "ai_ask"

	return nil
}

// 分析ツールハンドラー
func (ai *AIService) handleAnalysisTool(params map[string]interface{}) (*ToolResult, error) {
	// パラメータを解析
	request := &AIServiceRequest{
		ProjectPath:          ".", // 現在のディレクトリ
		AnalysisType:         ai.getStringParam(params, "analysis_type", "detailed"),
		IncludeVisualization: ai.getBoolParam(params, "include_visualization", true),
		IncludeMultiRepo:     ai.getBoolParam(params, "include_multi_repo", false),
	}

	// 分析を実行
	ctx := context.Background()
	result, err := ai.AnalyzeProject(ctx, request)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: fmt.Sprintf("分析エラー: %v", err),
		}, nil
	}

	// 結果をJSON形式で返却
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: fmt.Sprintf("結果シリアライズエラー: %v", err),
		}, nil
	}

	return &ToolResult{
		IsError: false,
		Content: string(resultJSON),
		Metadata: map[string]interface{}{
			"processing_time": result.ProcessingInfo.TotalProcessingTime,
			"files_analyzed":  result.ProcessingInfo.FilesAnalyzed,
			"issues_found":    result.ProcessingInfo.IssuesFound,
		},
	}, nil
}

// 生成ツールハンドラー
func (ai *AIService) handleGenerationTool(params map[string]interface{}) (*ToolResult, error) {
	// パラメータを解析
	genType := ai.getStringParam(params, "type", "function")
	language := ai.getStringParam(params, "language", "go")
	description := ai.getStringParam(params, "description", "")
	requirements := ai.getStringArrayParam(params, "requirements", []string{})

	if description == "" {
		return &ToolResult{
			IsError: true,
			Content: "説明パラメータが必要です",
		}, nil
	}

	request := &CodeGenerationRequest{
		Type:         genType,
		Language:     language,
		Description:  description,
		Requirements: requirements,
		TestRequired: true,
		DocRequired:  true,
	}

	if ai.codeGenerator == nil {
		ai.codeGenerator = NewCodeGenerator(ai.llmClient, ai.constraints, ".")
	}

	ctx := context.Background()
	result, err := ai.codeGenerator.GenerateCode(ctx, request)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: fmt.Sprintf("コード生成エラー: %v", err),
		}, nil
	}

	// 結果をJSON形式で返却
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: fmt.Sprintf("結果シリアライズエラー: %v", err),
		}, nil
	}

	return &ToolResult{
		IsError: false,
		Content: string(resultJSON),
		Metadata: map[string]interface{}{
			"generation_time": result.GenerationTime.String(),
			"generated_files": len(result.GeneratedCode),
			"created_tests":   len(result.CreatedTests),
			"generated_docs":  len(result.Documentation),
		},
	}, nil
}

// 対話ツールハンドラー
func (ai *AIService) handleInteractiveTool(params map[string]interface{}) (*ToolResult, error) {
	query := ai.getStringParam(params, "query", "")
	if query == "" {
		return &ToolResult{
			IsError: true,
			Content: "質問パラメータが必要です",
		}, nil
	}

	ctx := context.Background()
	response, err := ai.GetInteractiveRecommendations(ctx, ".", query)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: fmt.Sprintf("対話エラー: %v", err),
		}, nil
	}

	// 結果をJSON形式で返却
	resultJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: fmt.Sprintf("結果シリアライズエラー: %v", err),
		}, nil
	}

	return &ToolResult{
		IsError: false,
		Content: string(resultJSON),
	}, nil
}

// パラメータヘルパー関数
func (ai *AIService) getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if val, ok := params[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func (ai *AIService) getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := params[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func (ai *AIService) getStringArrayParam(params map[string]interface{}, key string, defaultValue []string) []string {
	if val, ok := params[key]; ok {
		if arr, ok := val.([]interface{}); ok {
			var result []string
			for _, item := range arr {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return defaultValue
}

// ユーティリティ関数
func getCurrentTime() time.Time {
	return time.Now()
}

func formatDuration(d time.Duration) string {
	return d.String()
}
