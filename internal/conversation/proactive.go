package conversation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/analysis"
)

// プロアクティブ会話システムの実装

// ProactiveManager はプロアクティブな提案と対話を管理
type ProactiveManager struct {
	projectAnalyzer     analysis.ProjectAnalyzer
	lastProjectAnalysis *analysis.ProjectAnalysis
	lastAnalysisTime    time.Time
	userContext         *UserContext
	suggestionHistory   []ProactiveSuggestion
}

// UserContext はユーザーの行動パターンと好みを記録
type UserContext struct {
	CurrentTask        string                 `json:"current_task"`
	RecentActions      []UserAction           `json:"recent_actions"`
	PreferredLanguages []string               `json:"preferred_languages"`
	WorkingStyle       string                 `json:"working_style"` // "detail", "concise", "guided"
	FocusAreas         []string               `json:"focus_areas"`   // "security", "performance", "maintainability"
	LastActiveTime     time.Time              `json:"last_active_time"`
	SessionDuration    time.Duration          `json:"session_duration"`
	InteractionPattern InteractionPattern     `json:"interaction_pattern"`
	ProjectContext     *ProjectContextInfo    `json:"project_context"`
	Preferences        map[string]interface{} `json:"preferences"`
}

// UserAction はユーザーの行動を記録
type UserAction struct {
	Type      string            `json:"type"`   // "file_edit", "command_run", "question_ask"
	Target    string            `json:"target"` // ファイルパス、コマンド、質問内容
	Timestamp time.Time         `json:"timestamp"`
	Context   map[string]string `json:"context"`
	Success   bool              `json:"success"`
	Duration  time.Duration     `json:"duration"`
	Result    string            `json:"result"`
}

// InteractionPattern はユーザーの対話パターン
type InteractionPattern struct {
	QuestionsPerSession    int            `json:"questions_per_session"`
	AverageResponseTime    time.Duration  `json:"average_response_time"`
	PreferredResponseStyle string         `json:"preferred_response_style"` // "technical", "explanatory", "actionable"
	ConfirmationFrequency  string         `json:"confirmation_frequency"`   // "always", "important", "never"
	ToolUsageFrequency     map[string]int `json:"tool_usage_frequency"`
}

// ProjectContextInfo はプロジェクト固有のコンテキスト
type ProjectContextInfo struct {
	CurrentBranch     string        `json:"current_branch"`
	RecentCommits     []string      `json:"recent_commits"`
	ModifiedFiles     []string      `json:"modified_files"`
	ActiveFeature     string        `json:"active_feature"`
	DeploymentStage   string        `json:"deployment_stage"` // "development", "staging", "production"
	TechnicalDebt     time.Duration `json:"technical_debt"`
	TeamMembers       []string      `json:"team_members"`
	ProjectPriorities []string      `json:"project_priorities"`
}

// ProactiveSuggestion はプロアクティブな提案
type ProactiveSuggestion struct {
	ID          string                     `json:"id"`
	Type        string                     `json:"type"`     // "security", "performance", "refactor", "test", "documentation"
	Priority    string                     `json:"priority"` // "critical", "high", "medium", "low"
	Title       string                     `json:"title"`
	Description string                     `json:"description"`
	Action      string                     `json:"action"`
	Files       []string                   `json:"files"`
	Commands    []string                   `json:"commands"`
	Reasoning   string                     `json:"reasoning"`
	Context     ProactiveSuggestionContext `json:"context"`
	CreatedAt   time.Time                  `json:"created_at"`
	ExpiresAt   time.Time                  `json:"expires_at"`
	Accepted    bool                       `json:"accepted"`
	AcceptedAt  *time.Time                 `json:"accepted_at,omitempty"`
}

// ProactiveSuggestionContext は提案のコンテキスト
type ProactiveSuggestionContext struct {
	TriggerEvent string            `json:"trigger_event"` // "file_change", "time_based", "quality_drop"
	RelatedFiles []string          `json:"related_files"`
	Dependencies []string          `json:"dependencies"`
	Impact       string            `json:"impact"` // "high", "medium", "low"
	Effort       string            `json:"effort"` // "high", "medium", "low"
	Benefits     []string          `json:"benefits"`
	Risks        []string          `json:"risks"`
	Alternatives []string          `json:"alternatives"`
	Metadata     map[string]string `json:"metadata"`
}

// NewProactiveManager は新しいプロアクティブマネージャーを作成
func NewProactiveManager(projectAnalyzer analysis.ProjectAnalyzer) *ProactiveManager {
	return &ProactiveManager{
		projectAnalyzer:   projectAnalyzer,
		userContext:       NewUserContext(),
		suggestionHistory: make([]ProactiveSuggestion, 0),
	}
}

// NewUserContext は新しいユーザーコンテキストを作成
func NewUserContext() *UserContext {
	return &UserContext{
		RecentActions:      make([]UserAction, 0, 50), // 最大50の行動を記録
		PreferredLanguages: make([]string, 0),
		FocusAreas:         make([]string, 0),
		LastActiveTime:     time.Now(),
		WorkingStyle:       "guided", // デフォルトはガイド付き
		InteractionPattern: InteractionPattern{
			ToolUsageFrequency:     make(map[string]int),
			PreferredResponseStyle: "explanatory",
			ConfirmationFrequency:  "important",
		},
		ProjectContext: &ProjectContextInfo{
			ProjectPriorities: []string{"quality", "security", "performance"},
		},
		Preferences: make(map[string]interface{}),
	}
}

// AnalyzeAndSuggest はプロジェクトを分析してプロアクティブな提案を生成
func (pm *ProactiveManager) AnalyzeAndSuggest(ctx context.Context, projectPath string) ([]ProactiveSuggestion, error) {
	// プロジェクト分析を実行（キャッシュを考慮）
	shouldAnalyze := pm.shouldPerformAnalysis()
	var projectAnalysis *analysis.ProjectAnalysis
	var err error

	if shouldAnalyze {
		projectAnalysis, err = pm.projectAnalyzer.AnalyzeProject(projectPath)
		if err != nil {
			return nil, fmt.Errorf("プロジェクト分析エラー: %w", err)
		}
		pm.lastProjectAnalysis = projectAnalysis
		pm.lastAnalysisTime = time.Now()
	} else {
		projectAnalysis = pm.lastProjectAnalysis
	}

	if projectAnalysis == nil {
		return []ProactiveSuggestion{}, nil
	}

	// プロアクティブな提案を生成
	suggestions := make([]ProactiveSuggestion, 0)

	// 各種提案ジェネレーターを実行
	generators := []func(*analysis.ProjectAnalysis) ([]ProactiveSuggestion, error){
		pm.generateSecuritySuggestions,
		pm.generatePerformanceSuggestions,
		pm.generateMaintainabilitySuggestions,
		pm.generateTestingSuggestions,
		pm.generateRefactoringSuggestions,
		pm.generateDocumentationSuggestions,
	}

	for _, generator := range generators {
		if generated, err := generator(projectAnalysis); err == nil {
			suggestions = append(suggestions, generated...)
		}
	}

	// ユーザーコンテキストに基づいてフィルタリング
	filteredSuggestions := pm.filterSuggestionsByUserContext(suggestions)

	// 優先度順にソート
	pm.sortSuggestionsByPriority(filteredSuggestions)

	// 提案履歴に追加
	pm.suggestionHistory = append(pm.suggestionHistory, filteredSuggestions...)

	return filteredSuggestions, nil
}

// RecordUserAction はユーザーの行動を記録
func (pm *ProactiveManager) RecordUserAction(action UserAction) {
	action.Timestamp = time.Now()
	pm.userContext.RecentActions = append(pm.userContext.RecentActions, action)

	// 最大50の行動のみを保持
	if len(pm.userContext.RecentActions) > 50 {
		pm.userContext.RecentActions = pm.userContext.RecentActions[1:]
	}

	// パターンを更新
	pm.updateInteractionPattern(action)

	// プロジェクトコンテキストを更新
	pm.updateProjectContext(action)
}

// UpdateUserPreferences はユーザーの好みを更新
func (pm *ProactiveManager) UpdateUserPreferences(preferences map[string]interface{}) {
	for key, value := range preferences {
		pm.userContext.Preferences[key] = value
	}

	// 特定の設定を構造化データに反映
	if workingStyle, ok := preferences["working_style"].(string); ok {
		pm.userContext.WorkingStyle = workingStyle
	}

	if focusAreas, ok := preferences["focus_areas"].([]string); ok {
		pm.userContext.FocusAreas = focusAreas
	}
}

// GetContextualResponse はコンテキストに基づいた応答を生成
func (pm *ProactiveManager) GetContextualResponse(input string, projectAnalysis *analysis.ProjectAnalysis) string {
	// ユーザーの意図を分析
	intent := pm.analyzeUserIntent(input)

	// コンテキストに基づいた応答を構築
	response := pm.buildContextualResponse(input, intent, projectAnalysis)

	return response
}

// 内部メソッド

func (pm *ProactiveManager) shouldPerformAnalysis() bool {
	// 初回分析の場合
	if pm.lastProjectAnalysis == nil {
		return true
	}

	// 前回の分析から5分以上経過している場合
	if time.Since(pm.lastAnalysisTime) > 5*time.Minute {
		return true
	}

	// ユーザーが重要な変更を行った場合
	if pm.hasSignificantChanges() {
		return true
	}

	return false
}

func (pm *ProactiveManager) hasSignificantChanges() bool {
	// 最近のアクションをチェック
	recentActions := pm.getRecentActions(time.Minute * 2)

	significantActions := 0
	for _, action := range recentActions {
		if action.Type == "file_edit" || action.Type == "command_run" {
			significantActions++
		}
	}

	return significantActions > 3 // 2分間で3つ以上の重要なアクション
}

func (pm *ProactiveManager) getRecentActions(duration time.Duration) []UserAction {
	cutoff := time.Now().Add(-duration)
	recent := make([]UserAction, 0)

	for _, action := range pm.userContext.RecentActions {
		if action.Timestamp.After(cutoff) {
			recent = append(recent, action)
		}
	}

	return recent
}

func (pm *ProactiveManager) generateSecuritySuggestions(projectAnalysis *analysis.ProjectAnalysis) ([]ProactiveSuggestion, error) {
	suggestions := make([]ProactiveSuggestion, 0)

	// クリティカルなセキュリティ問題
	for _, issue := range projectAnalysis.SecurityIssues {
		if issue.Severity == "critical" {
			suggestion := ProactiveSuggestion{
				ID:          fmt.Sprintf("security_%d", time.Now().UnixNano()),
				Type:        "security",
				Priority:    "critical",
				Title:       "緊急: クリティカルなセキュリティ問題",
				Description: fmt.Sprintf("ファイル %s でクリティカルなセキュリティ問題が検出されました: %s", issue.File, issue.Description),
				Action:      issue.Suggestion,
				Files:       []string{issue.File},
				Reasoning:   "クリティカルなセキュリティ脆弱性は即座に修正する必要があります。",
				Context: ProactiveSuggestionContext{
					TriggerEvent: "security_scan",
					Impact:       "high",
					Effort:       "medium",
					Benefits:     []string{"セキュリティリスクの除去", "コンプライアンス向上"},
					Risks:        []string{"修正しない場合の深刻な脆弱性リスク"},
				},
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(time.Hour), // 1時間で期限切れ
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions, nil
}

func (pm *ProactiveManager) generatePerformanceSuggestions(projectAnalysis *analysis.ProjectAnalysis) ([]ProactiveSuggestion, error) {
	suggestions := make([]ProactiveSuggestion, 0)

	// 大きなファイルの最適化提案
	if projectAnalysis.FileStructure != nil {
		for _, file := range projectAnalysis.FileStructure.LargestFiles {
			if file.Size > 1024*1024 { // 1MB以上
				suggestion := ProactiveSuggestion{
					ID:          fmt.Sprintf("performance_%d", time.Now().UnixNano()),
					Type:        "performance",
					Priority:    "medium",
					Title:       "大きなファイルの最適化",
					Description: fmt.Sprintf("ファイル %s (%d KB) が大きすぎます。パフォーマンスに影響する可能性があります。", file.Path, file.Size/1024),
					Action:      "ファイルを分割するか、不要な部分を削除することを検討してください",
					Files:       []string{file.Path},
					Reasoning:   "大きなファイルは読み込み時間とメモリ使用量に影響します。",
					Context: ProactiveSuggestionContext{
						TriggerEvent: "file_size_analysis",
						Impact:       "medium",
						Effort:       "medium",
						Benefits:     []string{"読み込み速度向上", "メモリ効率改善"},
					},
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	return suggestions, nil
}

func (pm *ProactiveManager) generateMaintainabilitySuggestions(projectAnalysis *analysis.ProjectAnalysis) ([]ProactiveSuggestion, error) {
	suggestions := make([]ProactiveSuggestion, 0)

	// 技術的負債の提案
	if projectAnalysis.QualityMetrics != nil && projectAnalysis.QualityMetrics.TechnicalDebt > time.Hour {
		suggestion := ProactiveSuggestion{
			ID:          fmt.Sprintf("maintainability_%d", time.Now().UnixNano()),
			Type:        "refactor",
			Priority:    "medium",
			Title:       "技術的負債の解消",
			Description: fmt.Sprintf("プロジェクトに%sの技術的負債が蓄積されています", projectAnalysis.QualityMetrics.TechnicalDebt.String()),
			Action:      "TODO、FIXME、HACKコメントを確認し、優先順位をつけて解決していきましょう",
			Reasoning:   "技術的負債は時間とともに保守性を低下させます。",
			Context: ProactiveSuggestionContext{
				TriggerEvent: "debt_analysis",
				Impact:       "medium",
				Effort:       "high",
				Benefits:     []string{"保守性向上", "開発効率改善", "バグ削減"},
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 1週間
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

func (pm *ProactiveManager) generateTestingSuggestions(projectAnalysis *analysis.ProjectAnalysis) ([]ProactiveSuggestion, error) {
	suggestions := make([]ProactiveSuggestion, 0)

	// テストカバレッジの改善提案
	if projectAnalysis.QualityMetrics != nil && projectAnalysis.QualityMetrics.TestCoverage < 50 {
		suggestion := ProactiveSuggestion{
			ID:          fmt.Sprintf("testing_%d", time.Now().UnixNano()),
			Type:        "test",
			Priority:    "medium",
			Title:       "テストカバレッジの向上",
			Description: fmt.Sprintf("現在のテストカバレッジは%.1f%%です。品質向上のためにテストを追加しませんか？", projectAnalysis.QualityMetrics.TestCoverage),
			Action:      "重要な機能から順にユニットテストを追加していきましょう",
			Reasoning:   "適切なテストカバレッジは品質とリファクタリングの安全性を向上させます。",
			Context: ProactiveSuggestionContext{
				TriggerEvent: "coverage_analysis",
				Impact:       "high",
				Effort:       "high",
				Benefits:     []string{"品質向上", "リグレッション防止", "リファクタリング安全性"},
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(3 * 24 * time.Hour),
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

func (pm *ProactiveManager) generateRefactoringSuggestions(projectAnalysis *analysis.ProjectAnalysis) ([]ProactiveSuggestion, error) {
	suggestions := make([]ProactiveSuggestion, 0)

	// コード重複の提案
	if projectAnalysis.QualityMetrics != nil && projectAnalysis.QualityMetrics.Duplication > 15 {
		suggestion := ProactiveSuggestion{
			ID:          fmt.Sprintf("refactor_%d", time.Now().UnixNano()),
			Type:        "refactor",
			Priority:    "low",
			Title:       "コード重複の削減",
			Description: fmt.Sprintf("%.1f%%のコード重複が検出されました。リファクタリングを検討しませんか？", projectAnalysis.QualityMetrics.Duplication),
			Action:      "重複したコードを共通の関数やモジュールに抽出しましょう",
			Reasoning:   "コード重複はメンテナンスコストを増大させ、バグの原因となります。",
			Context: ProactiveSuggestionContext{
				TriggerEvent: "duplication_analysis",
				Impact:       "medium",
				Effort:       "medium",
				Benefits:     []string{"保守性向上", "バグ削減", "コード品質改善"},
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

func (pm *ProactiveManager) generateDocumentationSuggestions(projectAnalysis *analysis.ProjectAnalysis) ([]ProactiveSuggestion, error) {
	suggestions := make([]ProactiveSuggestion, 0)

	// READMEファイルの提案
	if !pm.hasReadmeFile(projectAnalysis) {
		suggestion := ProactiveSuggestion{
			ID:          fmt.Sprintf("docs_%d", time.Now().UnixNano()),
			Type:        "documentation",
			Priority:    "low",
			Title:       "READMEファイルの作成",
			Description: "プロジェクトにREADME.mdがありません。プロジェクトの概要を説明するファイルを作成しませんか？",
			Action:      "README.mdファイルを作成し、プロジェクトの目的、インストール方法、使用方法を記述しましょう",
			Files:       []string{"README.md"},
			Reasoning:   "READMEファイルは新しい開発者やユーザーがプロジェクトを理解するために重要です。",
			Context: ProactiveSuggestionContext{
				TriggerEvent: "documentation_analysis",
				Impact:       "low",
				Effort:       "low",
				Benefits:     []string{"プロジェクト理解向上", "新規開発者のオンボーディング改善"},
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

func (pm *ProactiveManager) filterSuggestionsByUserContext(suggestions []ProactiveSuggestion) []ProactiveSuggestion {
	filtered := make([]ProactiveSuggestion, 0)

	for _, suggestion := range suggestions {
		// ユーザーの重点分野に基づくフィルタリング
		if pm.isRelevantToUser(suggestion) {
			// 同じタイプの提案が最近受け入れられている場合はスキップ
			if !pm.wasRecentlyAccepted(suggestion.Type) {
				filtered = append(filtered, suggestion)
			}
		}
	}

	// 上位5つまでに制限
	if len(filtered) > 5 {
		filtered = filtered[:5]
	}

	return filtered
}

func (pm *ProactiveManager) isRelevantToUser(suggestion ProactiveSuggestion) bool {
	// フォーカスエリアが設定されている場合はそれに基づく
	if len(pm.userContext.FocusAreas) > 0 {
		for _, area := range pm.userContext.FocusAreas {
			if area == suggestion.Type {
				return true
			}
		}
		return false
	}

	// 優先度が高い提案は常に関連性が高い
	if suggestion.Priority == "critical" || suggestion.Priority == "high" {
		return true
	}

	return true // デフォルトでは全て関連性があるとみなす
}

func (pm *ProactiveManager) wasRecentlyAccepted(suggestionType string) bool {
	cutoff := time.Now().Add(-24 * time.Hour) // 24時間以内

	for _, suggestion := range pm.suggestionHistory {
		if suggestion.Type == suggestionType &&
			suggestion.Accepted &&
			suggestion.AcceptedAt != nil &&
			suggestion.AcceptedAt.After(cutoff) {
			return true
		}
	}

	return false
}

func (pm *ProactiveManager) sortSuggestionsByPriority(suggestions []ProactiveSuggestion) {
	priorityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	// バブルソート（簡易実装）
	for i := 0; i < len(suggestions)-1; i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if priorityOrder[suggestions[i].Priority] > priorityOrder[suggestions[j].Priority] {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}
}

func (pm *ProactiveManager) updateInteractionPattern(action UserAction) {
	pattern := &pm.userContext.InteractionPattern

	// ツール使用頻度を更新
	if action.Type == "tool_use" {
		if tool, exists := action.Context["tool"]; exists {
			pattern.ToolUsageFrequency[tool]++
		}
	}

	// 応答時間を更新（質問の場合）
	if action.Type == "question_ask" && action.Duration > 0 {
		if pattern.AverageResponseTime == 0 {
			pattern.AverageResponseTime = action.Duration
		} else {
			// 移動平均を計算
			pattern.AverageResponseTime = (pattern.AverageResponseTime + action.Duration) / 2
		}
	}
}

func (pm *ProactiveManager) updateProjectContext(action UserAction) {
	context := pm.userContext.ProjectContext

	// ファイル編集の場合
	if action.Type == "file_edit" {
		// 最近編集されたファイルリストを更新
		context.ModifiedFiles = append(context.ModifiedFiles, action.Target)

		// 重複を除去し、最新の10ファイルのみ保持
		seen := make(map[string]bool)
		unique := make([]string, 0)
		for i := len(context.ModifiedFiles) - 1; i >= 0 && len(unique) < 10; i-- {
			file := context.ModifiedFiles[i]
			if !seen[file] {
				seen[file] = true
				unique = append([]string{file}, unique...)
			}
		}
		context.ModifiedFiles = unique
	}

	// 現在のタスクを推測
	if action.Type == "question_ask" {
		context.ActiveFeature = pm.inferCurrentTask(action.Target)
	}
}

func (pm *ProactiveManager) inferCurrentTask(question string) string {
	question = strings.ToLower(question)

	taskKeywords := map[string]string{
		"test":     "testing",
		"bug":      "debugging",
		"fix":      "debugging",
		"deploy":   "deployment",
		"secure":   "security_review",
		"optim":    "optimization",
		"refact":   "refactoring",
		"document": "documentation",
		"review":   "code_review",
	}

	for keyword, task := range taskKeywords {
		if strings.Contains(question, keyword) {
			return task
		}
	}

	return "general_development"
}

func (pm *ProactiveManager) analyzeUserIntent(input string) string {
	input = strings.ToLower(input)

	intents := map[string][]string{
		"help_request":        {"help", "how", "what", "why", "explain", "show me"},
		"action_request":      {"run", "execute", "create", "delete", "modify", "update"},
		"information_request": {"status", "check", "list", "show", "display"},
		"problem_solving":     {"error", "bug", "issue", "problem", "not working", "failed"},
		"optimization":        {"optimize", "improve", "faster", "better", "efficient"},
		"learning":            {"learn", "understand", "tutorial", "guide", "example"},
	}

	for intent, keywords := range intents {
		for _, keyword := range keywords {
			if strings.Contains(input, keyword) {
				return intent
			}
		}
	}

	return "general_query"
}

func (pm *ProactiveManager) buildContextualResponse(input, intent string, projectAnalysis *analysis.ProjectAnalysis) string {
	// ベースレスポンスを構築
	response := pm.buildBaseResponse(input, intent)

	// プロジェクト情報を追加
	if projectAnalysis != nil {
		contextInfo := pm.buildProjectContext(projectAnalysis)
		if contextInfo != "" {
			response += "\n\n" + contextInfo
		}
	}

	// 関連する提案を追加
	suggestions := pm.getRelevantSuggestions(intent)
	if len(suggestions) > 0 {
		response += "\n\n" + pm.formatSuggestions(suggestions)
	}

	return response
}

func (pm *ProactiveManager) buildBaseResponse(input, intent string) string {
	// 意図に基づいた基本応答パターン
	switch intent {
	case "help_request":
		return "お手伝いします。プロジェクトの状況を分析しながら最適な解決策を提案させていただきます。"
	case "action_request":
		return "承知しました。安全に実行できるよう、必要に応じて事前チェックを行います。"
	case "problem_solving":
		return "問題の解決をお手伝いします。まず状況を確認させていただきます。"
	case "optimization":
		return "最適化について一緒に検討しましょう。現在のプロジェクト状況に基づいて提案します。"
	default:
		return "ご質問を承りました。プロジェクトのコンテキストを考慮してお答えします。"
	}
}

func (pm *ProactiveManager) buildProjectContext(projectAnalysis *analysis.ProjectAnalysis) string {
	context := make([]string, 0)

	// プロジェクト基本情報
	if projectAnalysis.Language != "" {
		context = append(context, fmt.Sprintf("📊 **プロジェクト情報**: %s", projectAnalysis.Language))
		if projectAnalysis.Framework != "" {
			context[len(context)-1] += fmt.Sprintf(" (%s)", projectAnalysis.Framework)
		}
	}

	// 品質メトリクス
	if projectAnalysis.QualityMetrics != nil {
		metrics := projectAnalysis.QualityMetrics
		context = append(context, fmt.Sprintf("🎯 **品質状況**: 保守性 %.0f点, セキュリティ %.0f点, テストカバレッジ %.1f%%",
			metrics.Maintainability, metrics.SecurityScore, metrics.TestCoverage))
	}

	// 最近の変更
	if len(pm.userContext.ProjectContext.ModifiedFiles) > 0 {
		files := pm.userContext.ProjectContext.ModifiedFiles
		if len(files) > 3 {
			files = files[:3]
		}
		context = append(context, fmt.Sprintf("📝 **最近の変更**: %s", strings.Join(files, ", ")))
	}

	if len(context) > 0 {
		return strings.Join(context, "\n")
	}

	return ""
}

func (pm *ProactiveManager) getRelevantSuggestions(intent string) []ProactiveSuggestion {
	relevant := make([]ProactiveSuggestion, 0)

	// 最近の提案から関連するものを抽出
	for _, suggestion := range pm.suggestionHistory {
		if pm.isSuggestionRelevantToIntent(suggestion, intent) && !suggestion.Accepted {
			relevant = append(relevant, suggestion)
		}
	}

	// 最大3つまでに制限
	if len(relevant) > 3 {
		relevant = relevant[:3]
	}

	return relevant
}

func (pm *ProactiveManager) isSuggestionRelevantToIntent(suggestion ProactiveSuggestion, intent string) bool {
	relevanceMap := map[string][]string{
		"problem_solving": {"security", "performance", "refactor"},
		"optimization":    {"performance", "refactor"},
		"help_request":    {"documentation", "test"},
		"action_request":  {"security", "refactor", "test"},
	}

	if relevantTypes, exists := relevanceMap[intent]; exists {
		for _, t := range relevantTypes {
			if suggestion.Type == t {
				return true
			}
		}
	}

	return false
}

func (pm *ProactiveManager) formatSuggestions(suggestions []ProactiveSuggestion) string {
	if len(suggestions) == 0 {
		return ""
	}

	result := "💡 **関連する提案**:"
	for i, suggestion := range suggestions {
		priority := suggestion.Priority
		if priority == "critical" {
			priority = "🔴 緊急"
		} else if priority == "high" {
			priority = "🟡 重要"
		} else {
			priority = "🔵 推奨"
		}

		result += fmt.Sprintf("\n%d. [%s] %s", i+1, priority, suggestion.Title)
		if suggestion.Description != "" {
			result += fmt.Sprintf("\n   %s", suggestion.Description)
		}
	}

	return result
}

func (pm *ProactiveManager) hasReadmeFile(projectAnalysis *analysis.ProjectAnalysis) bool {
	// 簡易実装：実際はファイル構造から README.md を検索
	return false
}

// AcceptSuggestion は提案を受け入れる
func (pm *ProactiveManager) AcceptSuggestion(suggestionID string) error {
	for i := range pm.suggestionHistory {
		if pm.suggestionHistory[i].ID == suggestionID {
			pm.suggestionHistory[i].Accepted = true
			now := time.Now()
			pm.suggestionHistory[i].AcceptedAt = &now
			return nil
		}
	}
	return fmt.Errorf("提案ID %s が見つかりません", suggestionID)
}

// GetUserContext はユーザーコンテキストを取得
func (pm *ProactiveManager) GetUserContext() *UserContext {
	return pm.userContext
}

// GetSuggestionHistory は提案履歴を取得
func (pm *ProactiveManager) GetSuggestionHistory() []ProactiveSuggestion {
	return pm.suggestionHistory
}
