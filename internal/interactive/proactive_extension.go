package interactive

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/analysis"
	"github.com/glkt/vyb-code/internal/conversation"
)

// ProactiveExtension はインタラクティブセッションマネージャーのプロアクティブ拡張
type ProactiveExtension struct {
	sessionManager   *interactiveSessionManager
	proactiveManager *conversation.ProactiveManager
	projectAnalyzer  analysis.ProjectAnalyzer
	analysisCache    *analysis.ProjectAnalysis
	lastAnalysisTime time.Time
	projectPath      string
}

// NewProactiveExtension は新しいプロアクティブ拡張を作成
func NewProactiveExtension(sessionManager *interactiveSessionManager) *ProactiveExtension {
	// プロジェクトパスを取得
	projectPath, _ := os.Getwd()

	// プロジェクト分析器を作成
	analysisConfig := analysis.DefaultAnalysisConfig()
	projectAnalyzer := analysis.NewProjectAnalyzer(analysisConfig)

	// プロアクティブマネージャーを作成
	proactiveManager := conversation.NewProactiveManager(projectAnalyzer)

	return &ProactiveExtension{
		sessionManager:   sessionManager,
		proactiveManager: proactiveManager,
		projectAnalyzer:  projectAnalyzer,
		projectPath:      projectPath,
	}
}

// EnhanceProcessUserInput は既存のProcessUserInputをプロアクティブ機能で拡張
func (pe *ProactiveExtension) EnhanceProcessUserInput(
	ctx context.Context,
	sessionID string,
	input string,
) (*InteractionResponse, error) {
	// ユーザーアクションを記録
	action := conversation.UserAction{
		Type:      "question_ask",
		Target:    input,
		Timestamp: time.Now(),
		Context:   make(map[string]string),
		Success:   true,
	}
	pe.proactiveManager.RecordUserAction(action)

	// タイムアウト付き軽量分析：安全にプロアクティブ機能を実行
	var suggestions []conversation.ProactiveSuggestion

	// 科学的認知分析との統合によるプロアクティブ機能強化
	analysisCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if pe.shouldPerformAnalysis() {
		// 軽量版分析 + 科学的認知分析を実行
		if err := pe.performEnhancedAnalysis(analysisCtx, input); err != nil {
			// 分析が失敗してもメイン処理は継続
			fmt.Printf("Warning: 強化分析に失敗しました: %v\n", err)
		}
	}

	// 無限再帰を避けるため、processUserInputFallbackを直接呼び出し
	originalResponse, err := pe.sessionManager.processUserInputFallback(ctx, sessionID, input)
	if err != nil {
		return nil, err
	}

	// プロアクティブな要素で応答を拡張
	enhancedResponse := pe.enhanceResponse(originalResponse, suggestions, input)

	return enhancedResponse, nil
}

// performEnhancedAnalysis は科学的認知分析統合版プロジェクト分析を実行
func (pe *ProactiveExtension) performEnhancedAnalysis(ctx context.Context, input string) error {
	// 1. 基本的なプロジェクト情報を収集（重い処理を回避）
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// プロアクティブ分析を実行
	}

	// 2. 科学的認知分析システムが利用可能な場合は活用
	if pe.sessionManager.cognitiveAnalyzer != nil {
		// 軽量な認知分析を実行
		err := pe.performCognitiveAnalysis(ctx, input)
		if err != nil {
			// 認知分析が失敗してもプロアクティブ機能は継続
			fmt.Printf("Debug: 認知分析をスキップしました: %v\n", err)
		}
	}

	// 3. プロアクティブ提案の更新
	pe.updateProactiveSuggestions(input)

	pe.lastAnalysisTime = time.Now()
	return nil
}

// performCognitiveAnalysis は軽量な科学的認知分析を実行
func (pe *ProactiveExtension) performCognitiveAnalysis(ctx context.Context, input string) error {
	// タイムアウト短縮のため、クイック分析のみ実行
	request := &analysis.AnalysisRequest{
		UserInput: input,
		Response:  "",
		Context: map[string]interface{}{
			"analysis_type": "proactive_quick",
			"timeout":       "3s",
		},
		AnalysisDepth:   "quick",                // 軽量分析
		RequiredMetrics: []string{"confidence"}, // 最小限のメトリクス
	}

	// 認知分析を実行（エラーが発生してもプロアクティブ機能に影響しない）
	_, err := pe.sessionManager.cognitiveAnalyzer.AnalyzeCognitive(ctx, request)
	return err
}

// updateProactiveSuggestions はプロアクティブ提案を更新
func (pe *ProactiveExtension) updateProactiveSuggestions(input string) {
	// 入力に基づいて適切な提案を生成
	// 実装は将来的に拡張可能

	// 基本的な提案パターンのマッチング
	suggestions := pe.generateBasicSuggestions(input)

	// プロアクティブマネージャーに提案を登録（直接的な方法）
	if pe.proactiveManager != nil && len(suggestions) > 0 {
		// suggestionHistoryに直接追加（プライベートフィールドのため、将来的にPublicメソッドが追加されることを想定）
		// 現在は提案の生成のみ行い、実際の統合は将来の拡張で対応
		fmt.Printf("Debug: 生成された提案数: %d\n", len(suggestions))
	}
}

// generateBasicSuggestions は基本的な提案を生成
func (pe *ProactiveExtension) generateBasicSuggestions(input string) []conversation.ProactiveSuggestion {
	var suggestions []conversation.ProactiveSuggestion

	// 入力パターンに基づく提案生成
	lowerInput := strings.ToLower(input)

	// エラー関連の質問
	if strings.Contains(lowerInput, "エラー") || strings.Contains(lowerInput, "error") {
		suggestions = append(suggestions, conversation.ProactiveSuggestion{
			ID:          fmt.Sprintf("debug_%d", time.Now().UnixNano()),
			Type:        "debugging_help",
			Priority:    "high",
			Title:       "デバッグ支援機能",
			Description: "デバッグ支援機能を提案",
			Action:      "error_analysis",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		})
	}

	// ファイル操作関連
	if strings.Contains(lowerInput, "ファイル") || strings.Contains(lowerInput, "file") {
		suggestions = append(suggestions, conversation.ProactiveSuggestion{
			ID:          fmt.Sprintf("file_%d", time.Now().UnixNano()),
			Type:        "file_operations",
			Priority:    "medium",
			Title:       "ファイル操作最適化",
			Description: "ファイル操作の最適化を提案",
			Action:      "file_management",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		})
	}

	// テスト関連
	if strings.Contains(lowerInput, "テスト") || strings.Contains(lowerInput, "test") {
		suggestions = append(suggestions, conversation.ProactiveSuggestion{
			ID:          fmt.Sprintf("test_%d", time.Now().UnixNano()),
			Type:        "testing_support",
			Priority:    "high",
			Title:       "テスト実行改善",
			Description: "テスト実行・改善を提案",
			Action:      "test_enhancement",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		})
	}

	return suggestions
}

// EnhancePrompt はLLMへのプロンプトをプロジェクトコンテキストで強化
func (pe *ProactiveExtension) EnhancePrompt(originalPrompt, userInput string) string {
	// プロジェクト情報を追加
	projectInfo := pe.buildProjectContextPrompt()

	// ユーザーコンテキストを追加
	userContext := pe.buildUserContextPrompt()

	// プロアクティブなガイダンスを追加
	proactiveGuidance := pe.buildProactiveGuidancePrompt(userInput)

	// 強化されたプロンプトを構築
	enhancedPrompt := fmt.Sprintf(`%s

🔍 **プロジェクトコンテキスト**:
%s

👤 **ユーザーコンテキスト**:
%s

💡 **プロアクティブガイダンス**:
%s

上記の情報を考慮して、具体的で実用的な回答を提供してください。
必要に応じてファイルの読み取り、コマンドの実行、コード分析を積極的に行ってください。

Claude Codeのような詳細で親しみやすいトーンで、次のアクションを提案してください。`,
		originalPrompt, projectInfo, userContext, proactiveGuidance)

	return enhancedPrompt
}

// GetProactiveSuggestions は現在のプロアクティブ提案を取得
func (pe *ProactiveExtension) GetProactiveSuggestions(ctx context.Context) ([]conversation.ProactiveSuggestion, error) {
	return pe.proactiveManager.AnalyzeAndSuggest(ctx, pe.projectPath)
}

// AcceptSuggestion はプロアクティブ提案を受け入れ
func (pe *ProactiveExtension) AcceptSuggestion(suggestionID string) error {
	return pe.proactiveManager.AcceptSuggestion(suggestionID)
}

// UpdateUserPreferences はユーザーの好みを更新
func (pe *ProactiveExtension) UpdateUserPreferences(preferences map[string]interface{}) {
	pe.proactiveManager.UpdateUserPreferences(preferences)
}

// 内部メソッド

func (pe *ProactiveExtension) shouldPerformAnalysis() bool {
	// 初回分析
	if pe.analysisCache == nil {
		return true
	}

	// 5分以上経過
	if time.Since(pe.lastAnalysisTime) > 5*time.Minute {
		return true
	}

	// ユーザーの活動が活発
	userContext := pe.proactiveManager.GetUserContext()
	recentActions := 0
	cutoff := time.Now().Add(-2 * time.Minute)

	for _, action := range userContext.RecentActions {
		if action.Timestamp.After(cutoff) {
			recentActions++
		}
	}

	return recentActions > 3
}

func (pe *ProactiveExtension) performProjectAnalysis(ctx context.Context) error {
	analysis, err := pe.projectAnalyzer.AnalyzeProject(pe.projectPath)
	if err != nil {
		return err
	}

	pe.analysisCache = analysis
	pe.lastAnalysisTime = time.Now()
	return nil
}

func (pe *ProactiveExtension) enhanceResponse(
	originalResponse *InteractionResponse,
	suggestions []conversation.ProactiveSuggestion,
	userInput string,
) *InteractionResponse {
	// 元の応答をコピー
	enhanced := *originalResponse

	// プロアクティブな要素を追加
	if len(suggestions) > 0 {
		enhanced.Message += "\n\n" + pe.formatSuggestions(suggestions)
	}

	// 関連するプロジェクト情報を追加
	if pe.analysisCache != nil {
		projectInsights := pe.getRelevantProjectInsights(userInput)
		if projectInsights != "" {
			enhanced.Message += "\n\n" + projectInsights
		}
	}

	// メタデータにプロアクティブ情報を追加
	if enhanced.Metadata == nil {
		enhanced.Metadata = make(map[string]string)
	}
	enhanced.Metadata["proactive_suggestions_count"] = fmt.Sprintf("%d", len(suggestions))
	enhanced.Metadata["project_analyzed"] = fmt.Sprintf("%v", pe.analysisCache != nil)

	return &enhanced
}

func (pe *ProactiveExtension) buildProjectContextPrompt() string {
	if pe.analysisCache == nil {
		return "プロジェクト分析を実行中..."
	}

	context := make([]string, 0)

	// 基本情報
	context = append(context, fmt.Sprintf("**言語**: %s", pe.analysisCache.Language))
	if pe.analysisCache.Framework != "" {
		context[len(context)-1] += fmt.Sprintf(" (%s)", pe.analysisCache.Framework)
	}

	// ファイル統計
	if pe.analysisCache.FileStructure != nil {
		context = append(context, fmt.Sprintf("**ファイル数**: %d個, **総行数**: %d行",
			pe.analysisCache.FileStructure.TotalFiles,
			pe.analysisCache.FileStructure.TotalLines))
	}

	// 品質メトリクス
	if pe.analysisCache.QualityMetrics != nil {
		metrics := pe.analysisCache.QualityMetrics
		context = append(context, fmt.Sprintf("**品質スコア**: 保守性 %.0f点, セキュリティ %.0f点, カバレッジ %.1f%%",
			metrics.Maintainability, metrics.SecurityScore, metrics.TestCoverage))
	}

	// 最近の変更（Git情報）
	if pe.analysisCache.GitInfo != nil {
		git := pe.analysisCache.GitInfo
		if git.HasChanges {
			context = append(context, "**状態**: 未コミットの変更あり")
		}
		context = append(context, fmt.Sprintf("**ブランチ**: %s", git.CurrentBranch))
	}

	// 主な技術スタック
	if len(pe.analysisCache.TechStack) > 0 {
		primaryTech := make([]string, 0)
		for _, tech := range pe.analysisCache.TechStack {
			if tech.Usage == "primary" && len(primaryTech) < 3 {
				primaryTech = append(primaryTech, tech.Name)
			}
		}
		if len(primaryTech) > 0 {
			context = append(context, fmt.Sprintf("**主要技術**: %s", strings.Join(primaryTech, ", ")))
		}
	}

	return strings.Join(context, "\n")
}

func (pe *ProactiveExtension) buildUserContextPrompt() string {
	userContext := pe.proactiveManager.GetUserContext()
	context := make([]string, 0)

	// 作業スタイル
	context = append(context, fmt.Sprintf("**作業スタイル**: %s", userContext.WorkingStyle))

	// フォーカスエリア
	if len(userContext.FocusAreas) > 0 {
		context = append(context, fmt.Sprintf("**重点分野**: %s", strings.Join(userContext.FocusAreas, ", ")))
	}

	// 現在のタスク
	if userContext.ProjectContext != nil && userContext.ProjectContext.ActiveFeature != "" {
		context = append(context, fmt.Sprintf("**現在のタスク**: %s", userContext.ProjectContext.ActiveFeature))
	}

	// 最近の活動
	if len(userContext.RecentActions) > 0 {
		recentActions := userContext.RecentActions
		if len(recentActions) > 3 {
			recentActions = recentActions[len(recentActions)-3:]
		}
		actionTypes := make([]string, 0)
		for _, action := range recentActions {
			actionTypes = append(actionTypes, action.Type)
		}
		context = append(context, fmt.Sprintf("**最近の活動**: %s", strings.Join(actionTypes, ", ")))
	}

	return strings.Join(context, "\n")
}

func (pe *ProactiveExtension) buildProactiveGuidancePrompt(userInput string) string {
	guidance := make([]string, 0)

	// 入力の種類に基づくガイダンス
	inputLower := strings.ToLower(userInput)

	if strings.Contains(inputLower, "error") || strings.Contains(inputLower, "bug") {
		guidance = append(guidance, "🔍 エラーの詳細分析とログの確認を提案")
		guidance = append(guidance, "🛠️ 関連ファイルの検査と修正案の提示")
	}

	if strings.Contains(inputLower, "test") {
		guidance = append(guidance, "🧪 テストカバレッジの現状確認")
		guidance = append(guidance, "📝 追加すべきテストケースの提案")
	}

	if strings.Contains(inputLower, "optimize") || strings.Contains(inputLower, "improve") {
		guidance = append(guidance, "⚡ パフォーマンス分析と最適化ポイントの特定")
		guidance = append(guidance, "📊 品質メトリクスに基づく改善提案")
	}

	if strings.Contains(inputLower, "security") {
		guidance = append(guidance, "🔒 セキュリティ分析の実行")
		guidance = append(guidance, "🛡️ 検出された脆弱性の修正案")
	}

	// デフォルトガイダンス
	if len(guidance) == 0 {
		guidance = append(guidance, "🤖 プロジェクト状況を踏まえた具体的な提案")
		guidance = append(guidance, "🔧 必要に応じた実践的なアクション")
	}

	return strings.Join(guidance, "\n")
}

func (pe *ProactiveExtension) formatSuggestions(suggestions []conversation.ProactiveSuggestion) string {
	if len(suggestions) == 0 {
		return ""
	}

	result := "💡 **プロアクティブ提案**:"

	for i, suggestion := range suggestions {
		if i >= 3 { // 最大3つまで表示
			break
		}

		priority := suggestion.Priority
		priorityIcon := "🔵"
		switch priority {
		case "critical":
			priorityIcon = "🔴"
		case "high":
			priorityIcon = "🟡"
		case "medium":
			priorityIcon = "🟠"
		}

		// strings.Titleは非推奨のため手動で先頭文字を大文字化
		capitalizedPriority := priority
		if len(priority) > 0 {
			capitalizedPriority = strings.ToUpper(priority[:1]) + priority[1:]
		}

		result += fmt.Sprintf("\n%d. [%s %s] **%s**",
			i+1, priorityIcon, capitalizedPriority, suggestion.Title)

		if suggestion.Description != "" {
			result += fmt.Sprintf("\n   %s", suggestion.Description)
		}

		if suggestion.Action != "" {
			result += fmt.Sprintf("\n   💼 **提案**: %s", suggestion.Action)
		}

		if len(suggestion.Files) > 0 && len(suggestion.Files) <= 3 {
			result += fmt.Sprintf("\n   📁 **関連ファイル**: %s", strings.Join(suggestion.Files, ", "))
		}
	}

	if len(suggestions) > 3 {
		result += fmt.Sprintf("\n\n他に %d 個の提案があります。", len(suggestions)-3)
	}

	return result
}

func (pe *ProactiveExtension) getRelevantProjectInsights(userInput string) string {
	if pe.analysisCache == nil {
		return ""
	}

	insights := make([]string, 0)
	inputLower := strings.ToLower(userInput)

	// セキュリティ関連の質問
	if strings.Contains(inputLower, "security") || strings.Contains(inputLower, "secure") {
		criticalIssues := 0
		for _, issue := range pe.analysisCache.SecurityIssues {
			if issue.Severity == "critical" {
				criticalIssues++
			}
		}
		if criticalIssues > 0 {
			insights = append(insights, fmt.Sprintf("🚨 **セキュリティ警告**: %d個のクリティカルな問題が検出されています", criticalIssues))
		} else if pe.analysisCache.QualityMetrics != nil {
			insights = append(insights, fmt.Sprintf("🛡️ **セキュリティスコア**: %.0f点", pe.analysisCache.QualityMetrics.SecurityScore))
		}
	}

	// テスト関連の質問
	if strings.Contains(inputLower, "test") {
		if pe.analysisCache.QualityMetrics != nil {
			coverage := pe.analysisCache.QualityMetrics.TestCoverage
			if coverage < 50 {
				insights = append(insights, fmt.Sprintf("📊 **テストカバレッジ**: %.1f%% (改善推奨)", coverage))
			} else {
				insights = append(insights, fmt.Sprintf("✅ **テストカバレッジ**: %.1f%%", coverage))
			}
		}
	}

	// パフォーマンス関連
	if strings.Contains(inputLower, "performance") || strings.Contains(inputLower, "slow") {
		if pe.analysisCache.FileStructure != nil && len(pe.analysisCache.FileStructure.LargestFiles) > 0 {
			largeFile := pe.analysisCache.FileStructure.LargestFiles[0]
			if largeFile.Size > 1024*1024 { // 1MB以上
				insights = append(insights, fmt.Sprintf("⚠️ **パフォーマンス注意**: %s が %d KB と大きめです",
					largeFile.Path, largeFile.Size/1024))
			}
		}
	}

	// 依存関係関連
	if strings.Contains(inputLower, "dependency") || strings.Contains(inputLower, "update") {
		outdatedCount := 0
		for _, dep := range pe.analysisCache.Dependencies {
			if dep.Outdated {
				outdatedCount++
			}
		}
		if outdatedCount > 0 {
			insights = append(insights, fmt.Sprintf("📦 **依存関係**: %d個の古い依存関係があります", outdatedCount))
		}
	}

	if len(insights) > 0 {
		return "🔍 **関連するプロジェクト情報**:\n" + strings.Join(insights, "\n")
	}

	return ""
}

// GetProjectAnalysis は現在のプロジェクト分析を取得
func (pe *ProactiveExtension) GetProjectAnalysis() *analysis.ProjectAnalysis {
	return pe.analysisCache
}

// RefreshAnalysis は強制的にプロジェクト分析を更新
func (pe *ProactiveExtension) RefreshAnalysis(ctx context.Context) error {
	return pe.performProjectAnalysis(ctx)
}
