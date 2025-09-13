package conversation

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/config"
)

// コマンド実行結果のキャッシュエントリ
type CacheEntry struct {
	Result    *ExecutionResult `json:"result"`
	Timestamp time.Time        `json:"timestamp"`
	TTL       time.Duration    `json:"ttl"`
}

// 実行型応答エンジン - コマンド実行とマルチツール連携
type ExecutionEngine struct {
	config          *config.Config
	projectPath     string
	enabled         bool
	allowedCommands map[string]bool
	safetyLimits    *SafetyLimits
	cache           map[string]*CacheEntry // パフォーマンス最適化用キャッシュ
	lastUserInput   string                 // マルチツールワークフロー用ユーザー入力保持
}

// 安全性制限
type SafetyLimits struct {
	MaxExecutionTime time.Duration `json:"max_execution_time"`
	AllowedPaths     []string      `json:"allowed_paths"`
	ForbiddenPaths   []string      `json:"forbidden_paths"`
	MaxOutputSize    int           `json:"max_output_size"`
	ReadOnlyMode     bool          `json:"read_only_mode"`
}

// 実行結果
type ExecutionResult struct {
	Command   string        `json:"command"`
	Output    string        `json:"output"`
	Error     string        `json:"error,omitempty"`
	ExitCode  int           `json:"exit_code"`
	Duration  time.Duration `json:"duration"`
	Truncated bool          `json:"truncated"`
	Timestamp time.Time     `json:"timestamp"`
}

// コンテキスト分析結果
type ContextAnalysis struct {
	Intent           string            `json:"intent"`
	RequiredAction   string            `json:"required_action"`
	ProjectContext   map[string]string `json:"project_context"`
	RelevantFiles    []string          `json:"relevant_files"`
	SuggestedCommand string            `json:"suggested_command"`
	SafeToExecute    bool              `json:"safe_to_execute"`
	Reasoning        string            `json:"reasoning"`
}

// 新しい実行エンジンを作成
func NewExecutionEngine(cfg *config.Config, projectPath string) *ExecutionEngine {
	return &ExecutionEngine{
		config:      cfg,
		projectPath: projectPath,
		enabled:     cfg.IsProactiveEnabled(),
		allowedCommands: map[string]bool{
			"git":    true,
			"ls":     true,
			"pwd":    true,
			"find":   true,
			"grep":   true,
			"cat":    true,
			"head":   true,
			"tail":   true,
			"wc":     true,
			"stat":   true,
			"file":   true,
			"go":     true,
			"npm":    true,
			"python": true,
			"node":   true,
		},
		safetyLimits: &SafetyLimits{
			MaxExecutionTime: 30 * time.Second,
			AllowedPaths:     []string{projectPath},
			ForbiddenPaths:   []string{"/etc", "/usr", "/var", "/sys"},
			MaxOutputSize:    10 * 1024, // 10KB
			ReadOnlyMode:     true,      // デフォルトは読み取り専用
		},
		cache: make(map[string]*CacheEntry), // キャッシュ初期化
	}
}

// ユーザー入力を分析して実行可能なアクションを特定
func (ee *ExecutionEngine) AnalyzeUserIntent(userInput string) *ContextAnalysis {
	// マルチツールワークフロー用にユーザー入力を保存
	ee.lastUserInput = userInput

	analysis := &ContextAnalysis{
		ProjectContext: make(map[string]string),
		RelevantFiles:  make([]string, 0),
		SafeToExecute:  false,
	}

	inputLower := strings.ToLower(userInput)
	originalInput := strings.TrimSpace(userInput)

	// 明示的コマンドの検出（最優先）
	if explicitCommand, isExplicit := ee.detectExplicitCommand(originalInput); isExplicit {
		analysis.Intent = "explicit_command"
		analysis.RequiredAction = "execute_explicit_command"
		analysis.SuggestedCommand = explicitCommand
		analysis.SafeToExecute = true
		analysis.Reasoning = "明示的コマンドを直接実行"
		return analysis
	}

	// スマート意図解釈（複数ツール連携）
	smartIntent := ee.detectSmartIntent(inputLower, originalInput)
	if smartIntent != "" {
		analysis.Intent = smartIntent
		analysis.RequiredAction = "execute_multi_tool_workflow"
		analysis.SuggestedCommand = fmt.Sprintf("multi-tool:%s", smartIntent)
		analysis.SafeToExecute = true
		analysis.Reasoning = fmt.Sprintf("複数ツール連携ワークフロー（%s）を実行", smartIntent)
		return analysis
	}

	// 包括的意図解釈システム
	intentType := ee.detectComprehensiveIntent(inputLower, originalInput)

	switch intentType {
	case "file_read":
		analysis.Intent = "file_read"
		analysis.RequiredAction = "read_file"
		analysis.SuggestedCommand = ee.inferFileReadCommand(inputLower, originalInput)
		analysis.SafeToExecute = true
		analysis.Reasoning = "ファイル読み取り要求を検出"
		return analysis

	case "file_search":
		analysis.Intent = "file_search"
		analysis.RequiredAction = "search_files"
		analysis.SuggestedCommand = ee.inferSearchCommand(inputLower, originalInput)
		analysis.SafeToExecute = true
		analysis.Reasoning = "ファイル検索要求を検出"
		return analysis

	case "file_write":
		analysis.Intent = "file_write"
		analysis.RequiredAction = "write_file"
		analysis.SuggestedCommand = ee.inferFileWriteCommand(inputLower, originalInput)
		analysis.SafeToExecute = true
		analysis.Reasoning = "ファイル作成要求を検出"
		return analysis

	case "git_analysis":
		analysis.Intent = "git_analysis"
		analysis.RequiredAction = "execute_git_command"
		analysis.SuggestedCommand = ee.inferGitCommand(inputLower)
		analysis.SafeToExecute = true
		analysis.Reasoning = "Git コマンドは安全に実行可能"
		return analysis

	case "file_exploration":
		analysis.Intent = "file_exploration"
		analysis.RequiredAction = "explore_files"
		analysis.SuggestedCommand = ee.inferFileCommand(inputLower)
		analysis.SafeToExecute = true
		analysis.Reasoning = "ファイル探索要求を検出"
		return analysis

	case "project_analysis":
		analysis.Intent = "project_analysis"
		analysis.RequiredAction = "analyze_project"
		analysis.SuggestedCommand = ee.inferAnalysisCommand(inputLower)
		analysis.SafeToExecute = true
		analysis.Reasoning = "プロジェクト分析は安全に実行可能"
		return analysis
	}

	// デフォルト動作 - 明確なアクションが特定されない場合
	analysis.Intent = "explanation_request"
	analysis.RequiredAction = "provide_explanation"
	analysis.Reasoning = "明確な実行可能アクションが特定されませんでした"

	return analysis
}

// 明示的コマンドの検出
func (ee *ExecutionEngine) detectExplicitCommand(input string) (string, bool) {
	trimmedInput := strings.TrimSpace(input)

	// 明示的なコマンドパターンを検出
	explicitCommands := []string{
		"cat ", "head ", "tail ", "less ", "more ",
		"grep ", "find ", "rg ", "ag ",
		"ls ", "ll ", "dir ", "tree ",
		"git ", "go ", "npm ", "yarn ", "make ",
		"python ", "node ", "java ", "rustc ",
		"touch ", "mkdir ", "cp ", "mv ", "rm ",
		"pwd", "whoami", "date", "uname",
	}

	for _, cmd := range explicitCommands {
		if strings.HasPrefix(trimmedInput, cmd) || trimmedInput == strings.TrimSpace(cmd) {
			return trimmedInput, true
		}
	}

	return "", false
}

// 包括的意図解釈システム
func (ee *ExecutionEngine) detectComprehensiveIntent(inputLower, originalInput string) string {
	// ファイル読み取り意図の検出
	fileReadPatterns := []string{
		"cat ", "read ", "show ", "view ", "display ", "print ",
		"内容", "中身", "読む", "表示", "見せて", "確認して", "開いて",
		"readmeファイル", "ファイルの内容", "ファイル内容", "の内容",
		"what's in", "show me", "let me see", "content of",
	}

	for _, pattern := range fileReadPatterns {
		if strings.Contains(inputLower, pattern) {
			return "file_read"
		}
	}

	// ファイル検索意図の検出
	searchPatterns := []string{
		"grep ", "find ", "search ", "locate ", "rg ", "ag ",
		"検索", "探す", "見つけて", "探して", "どこに", "含む",
		"bashtoolを検索", "を検索", "で検索", "から検索",
		"containing", "includes", "has", "with",
		"プロジェクト内で", "ファイル内で", "コード内で",
	}

	for _, pattern := range searchPatterns {
		if strings.Contains(inputLower, pattern) {
			return "file_search"
		}
	}

	// ファイル作成意図の検出
	writePatterns := []string{
		"create ", "write ", "make ", "generate ", "touch ",
		"作成", "書く", "作って", "書いて", "生成", "新しい",
		"テストファイル", "ファイルを作", "新しいファイル",
		"hello world", "example", "sample",
	}

	for _, pattern := range writePatterns {
		if strings.Contains(inputLower, pattern) {
			return "file_write"
		}
	}

	// 既存の意図検出を統合
	if ee.containsGitIntent(inputLower) {
		return "git_analysis"
	}

	if ee.containsFileIntent(inputLower) {
		return "file_exploration"
	}

	if ee.containsAnalysisIntent(inputLower) {
		return "project_analysis"
	}

	return ""
}

// スマート意図解釈 - コンテキストを考慮した高度な判断
func (ee *ExecutionEngine) detectSmartIntent(inputLower, originalInput string) string {
	// より高度な意図解釈：コンテキストを考慮した判断

	// プロジェクト理解が必要な場合
	if ee.requiresProjectUnderstanding(inputLower) {
		return "project_understanding"
	}

	// 複数ファイルの調査が必要な場合
	if ee.requiresMultiFileInvestigation(inputLower) {
		return "multi_file_investigation"
	}

	// 問題解決型のリクエスト
	if ee.isProblemSolving(inputLower) {
		return "problem_solving"
	}

	// 学習・理解支援型のリクエスト
	if ee.isLearningAssistance(inputLower) {
		return "learning_assistance"
	}

	return ""
}

// プロジェクト理解が必要かの判定
func (ee *ExecutionEngine) requiresProjectUnderstanding(input string) bool {
	patterns := []string{
		"プロジェクトの構造", "アーキテクチャ", "全体像", "概要", "構成",
		"project structure", "architecture", "overview", "how it works",
		"依存関係", "関連", "つながり", "関係", "影響",
		"dependencies", "relationships", "connections", "impact",
	}

	for _, pattern := range patterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}
	return false
}

// 複数ファイル調査が必要かの判定
func (ee *ExecutionEngine) requiresMultiFileInvestigation(input string) bool {
	patterns := []string{
		"どこで使われている", "どこで定義", "どこで実装", "使用箇所",
		"where is used", "where defined", "usage", "references",
		"すべての", "全て", "全部", "一覧", "リスト",
		"all", "every", "list", "show all", "find all",
	}

	for _, pattern := range patterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}
	return false
}

// 問題解決型かの判定
func (ee *ExecutionEngine) isProblemSolving(input string) bool {
	patterns := []string{
		"エラー", "問題", "バグ", "解決", "修正", "直す",
		"error", "issue", "bug", "problem", "fix", "solve",
		"動かない", "失敗", "うまくいかない", "おかしい",
		"doesn't work", "fails", "broken", "wrong",
	}

	for _, pattern := range patterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}
	return false
}

// 学習支援型かの判定
func (ee *ExecutionEngine) isLearningAssistance(input string) bool {
	patterns := []string{
		"教えて", "説明して", "わからない", "理解したい", "学びたい",
		"explain", "teach", "help understand", "how to", "what is",
		"なぜ", "どうして", "仕組み", "原理", "動作",
		"why", "how", "mechanism", "principle", "works",
	}

	for _, pattern := range patterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}
	return false
}

// 複数ツール連携実行システム
func (ee *ExecutionEngine) executeMultiToolWorkflow(intent string, inputLower, originalInput string) (*MultiToolResult, error) {
	switch intent {
	case "project_understanding":
		return ee.projectUnderstandingWorkflow(inputLower, originalInput)
	case "multi_file_investigation":
		return ee.multiFileInvestigationWorkflow(inputLower, originalInput)
	case "problem_solving":
		return ee.problemSolvingWorkflow(inputLower, originalInput)
	case "learning_assistance":
		return ee.learningAssistanceWorkflow(inputLower, originalInput)
	}

	return nil, fmt.Errorf("未知のワークフロー: %s", intent)
}

// 複数ツール実行結果
type MultiToolResult struct {
	Steps    []ToolStep    `json:"steps"`
	Summary  string        `json:"summary"`
	Duration time.Duration `json:"duration"`
}

type ToolStep struct {
	Tool     string        `json:"tool"`
	Command  string        `json:"command"`
	Output   string        `json:"output"`
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
}

// プロジェクト理解ワークフロー
func (ee *ExecutionEngine) projectUnderstandingWorkflow(inputLower, originalInput string) (*MultiToolResult, error) {
	start := time.Now()
	result := &MultiToolResult{
		Steps: make([]ToolStep, 0),
	}

	// Step 1: プロジェクト構造の把握
	step1 := ee.executeToolStep("ls", "ls -la")
	result.Steps = append(result.Steps, step1)

	// Step 2: 重要ファイルの確認
	step2 := ee.executeToolStep("cat", "cat README.md")
	result.Steps = append(result.Steps, step2)

	// Step 3: 依存関係の確認 (Go プロジェクトの場合)
	step3 := ee.executeToolStep("cat", "cat go.mod")
	result.Steps = append(result.Steps, step3)

	// Step 4: コードファイルの概要
	step4 := ee.executeToolStep("find", "find . -name '*.go' | head -10")
	result.Steps = append(result.Steps, step4)

	result.Duration = time.Since(start)
	result.Summary = ee.generateProjectSummary(result.Steps)

	return result, nil
}

// 複数ファイル調査ワークフロー
func (ee *ExecutionEngine) multiFileInvestigationWorkflow(inputLower, originalInput string) (*MultiToolResult, error) {
	start := time.Now()
	result := &MultiToolResult{
		Steps: make([]ToolStep, 0),
	}

	// 検索対象を特定
	searchTarget := ee.extractSearchTarget(originalInput)

	if searchTarget != "" {
		// Step 1: 検索実行
		step1 := ee.executeToolStep("grep", fmt.Sprintf("grep -r %s . --exclude-dir=.git", searchTarget))
		result.Steps = append(result.Steps, step1)

		// Step 2: ファイル一覧取得
		step2 := ee.executeToolStep("grep", fmt.Sprintf("grep -l %s $(find . -name '*.go' -not -path './.git/*')", searchTarget))
		result.Steps = append(result.Steps, step2)
	}

	result.Duration = time.Since(start)
	result.Summary = ee.generateInvestigationSummary(result.Steps, searchTarget)

	return result, nil
}

// 問題解決ワークフロー
func (ee *ExecutionEngine) problemSolvingWorkflow(inputLower, originalInput string) (*MultiToolResult, error) {
	start := time.Now()
	result := &MultiToolResult{
		Steps: make([]ToolStep, 0),
	}

	// Step 1: 現在の状況確認
	step1 := ee.executeToolStep("git", "git status")
	result.Steps = append(result.Steps, step1)

	// Step 2: 最近の変更確認
	step2 := ee.executeToolStep("git", "git log --oneline -5")
	result.Steps = append(result.Steps, step2)

	// Step 3: ビルド状況確認（Go プロジェクトの場合）
	step3 := ee.executeToolStep("go", "go build -n ./...")
	result.Steps = append(result.Steps, step3)

	result.Duration = time.Since(start)
	result.Summary = ee.generateProblemAnalysis(result.Steps)

	return result, nil
}

// 学習支援ワークフロー
func (ee *ExecutionEngine) learningAssistanceWorkflow(inputLower, originalInput string) (*MultiToolResult, error) {
	start := time.Now()
	result := &MultiToolResult{
		Steps: make([]ToolStep, 0),
	}

	// 学習対象を特定
	topic := ee.extractLearningTopic(originalInput)

	if topic == "" {
		topic = "general"
	}

	// 学習タイプの判定
	learningType := ee.determineLearningType(inputLower, topic)

	// 動的ツール選択による学習支援
	switch learningType {
	case "file_specific":
		// 特定ファイルについての説明
		if strings.HasSuffix(topic, ".md") || strings.HasSuffix(topic, ".go") {
			step1 := ee.executeToolStep("cat", fmt.Sprintf("cat %s", topic))
			result.Steps = append(result.Steps, step1)
		}

	case "concept_explanation":
		// 概念説明：関連ファイル検索 + 定義検索
		step1 := ee.executeToolStep("grep", ee.buildSafeGrepCommand(topic, "definition"))
		result.Steps = append(result.Steps, step1)

		step2 := ee.executeToolStep("find", fmt.Sprintf("find . -name '*.go' -type f | head -10"))
		result.Steps = append(result.Steps, step2)

		step3 := ee.executeToolStep("grep", ee.buildSafeGrepCommand(topic, "usage"))
		result.Steps = append(result.Steps, step3)

	case "code_analysis":
		// コード分析：構造体/関数/型の検索
		step1 := ee.executeToolStep("grep", ee.buildSafeGrepCommand(topic, "struct_func"))
		result.Steps = append(result.Steps, step1)

		step2 := ee.executeToolStep("grep", ee.buildSafeGrepCommand(topic, "type_def"))
		result.Steps = append(result.Steps, step2)

		step3 := ee.executeToolStep("find", "find . -name '*.go' -exec grep -l 'func.*' {} \\; | head -5")
		result.Steps = append(result.Steps, step3)

	case "architecture_understanding":
		// アーキテクチャ理解：プロジェクト構造分析
		step1 := ee.executeToolStep("find", "find . -type d -name 'internal' -o -name 'cmd' -o -name 'pkg'")
		result.Steps = append(result.Steps, step1)

		step2 := ee.executeToolStep("cat", "cat go.mod")
		result.Steps = append(result.Steps, step2)

		step3 := ee.executeToolStep("find", "find . -name '*.go' | head -10")
		result.Steps = append(result.Steps, step3)

		step4 := ee.executeToolStep("cat", "cat CLAUDE.md")
		result.Steps = append(result.Steps, step4)

	default:
		// 汎用学習支援
		step1 := ee.executeToolStep("find", "find . -name 'README.md' -o -name 'CLAUDE.md'")
		result.Steps = append(result.Steps, step1)

		if topic != "general" {
			step2 := ee.executeToolStep("grep", ee.buildSafeGrepCommand(topic, "general"))
			result.Steps = append(result.Steps, step2)
		}

		step3 := ee.executeToolStep("ls", "ls -la")
		result.Steps = append(result.Steps, step3)
	}

	result.Duration = time.Since(start)
	result.Summary = ee.generateLearningGuidance(result.Steps, topic)

	return result, nil
}

// ツール実行ステップ
func (ee *ExecutionEngine) executeToolStep(tool, command string) ToolStep {
	start := time.Now()
	result, err := ee.ExecuteCommand(command)

	step := ToolStep{
		Tool:     tool,
		Command:  command,
		Duration: time.Since(start),
		Success:  err == nil && result != nil,
	}

	if step.Success {
		step.Output = result.Output
	} else if err != nil {
		step.Output = err.Error()
	}

	return step
}

// ヘルパー関数群
func (ee *ExecutionEngine) extractSearchTarget(input string) string {
	// まず全文から技術的なキーワードを直接検索
	techPatterns := []string{
		"ExecutionEngine", "ContextManager", "AdvancedIntelligence", "LightweightProactive",
		"BashTool", "ReadTool", "WriteTool", "GrepTool", "MCPServer", "SessionManager",
	}

	for _, pattern := range techPatterns {
		if strings.Contains(input, pattern) {
			return pattern
		}
	}

	// 技術的なパターンを含む単語を検索
	words := strings.Fields(input)
	excludePatterns := []string{"が", "で", "を", "に", "は", "から", "まで", "と", "や", "どこ", "調べて", "使われて", "について"}

	for _, word := range words {
		if len(word) >= 4 {
			// 助詞を取り除いて検査
			cleanWord := word
			for _, exclude := range excludePatterns {
				cleanWord = strings.ReplaceAll(cleanWord, exclude, "")
			}

			if len(cleanWord) >= 4 {
				upperWord := strings.ToUpper(cleanWord)
				if strings.Contains(upperWord, "ENGINE") ||
					strings.Contains(upperWord, "MANAGER") ||
					strings.Contains(upperWord, "SERVICE") ||
					strings.Contains(upperWord, "TOOL") ||
					strings.Contains(upperWord, "HANDLER") ||
					strings.Contains(upperWord, "CONTROLLER") ||
					len(cleanWord) >= 8 {
					return cleanWord
				}
			}
		}
	}
	return ""
}

func (ee *ExecutionEngine) extractLearningTopic(input string) string {
	inputLower := strings.ToLower(input)

	// 技術キーワード辞書
	technicalTerms := map[string]string{
		"execution":    "ExecutionEngine",
		"エンジン":         "Engine",
		"エンジンの":        "Engine",
		"実行":           "execution",
		"バイブ":          "vibe",
		"vibe":         "vibe",
		"バイブモード":       "vibe_mode",
		"コンテキスト":       "context",
		"context":      "context",
		"プロアクティブ":      "proactive",
		"proactive":    "proactive",
		"マネージャー":       "Manager",
		"manager":      "Manager",
		"ワークフロー":       "workflow",
		"workflow":     "workflow",
		"セッション":        "session",
		"session":      "session",
		"goroutine":    "goroutine",
		"go":           "go",
		"golang":       "go",
		"docker":       "docker",
		"コンテナ":         "docker",
		"git":          "git",
		"github":       "github",
		"api":          "api",
		"rest":         "rest",
		"json":         "json",
		"yaml":         "yaml",
		"config":       "config",
		"設定":           "config",
		"構成":           "config",
		"ファイル":         "file",
		"file":         "file",
		"directory":    "directory",
		"ディレクトリ":       "directory",
		"プロジェクト":       "project",
		"project":      "project",
		"アーキテクチャ":      "architecture",
		"architecture": "architecture",
		"設計":           "design",
		"design":       "design",
	}

	// 特定のファイルや関数への言及
	if strings.Contains(inputLower, "claude.md") {
		return "CLAUDE.md"
	}
	if strings.Contains(inputLower, "readme") {
		return "README.md"
	}
	if strings.Contains(inputLower, "makefile") {
		return "Makefile"
	}

	// 複合語の検出（例：「バイブモードについて」）
	for japanese, english := range technicalTerms {
		if strings.Contains(inputLower, japanese) {
			return english
		}
	}

	// Goの構造体/型/関数の検出
	words := strings.Fields(input)
	for _, word := range words {
		// 大文字で始まる単語（Go の公開型/関数）
		if len(word) > 2 && word[0] >= 'A' && word[0] <= 'Z' {
			return word
		}

		// キャメルケースの検出
		if len(word) > 4 && strings.ContainsAny(word, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			return word
		}
	}

	// 意味のある単語の抽出（フィルタリング強化）
	stopWords := map[string]bool{
		"について": true, "って": true, "とは": true, "です": true, "ます": true,
		"する": true, "した": true, "される": true, "されている": true,
		"what": true, "how": true, "why": true, "when": true, "where": true,
		"explain": true, "teach": true, "help": true, "understand": true,
		"わからない": true, "教えて": true, "説明して": true, "理解": true,
		"that": true, "this": true, "the": true, "and": true, "or": true,
	}

	for _, word := range words {
		cleanWord := strings.Trim(word, ",.?!()[]{}\"'")
		if len(cleanWord) > 3 && !stopWords[strings.ToLower(cleanWord)] {
			return cleanWord
		}
	}

	return ""
}

func (ee *ExecutionEngine) generateProjectSummary(steps []ToolStep) string {
	return "プロジェクト構造とファイル構成を分析しました。"
}

func (ee *ExecutionEngine) generateInvestigationSummary(steps []ToolStep, target string) string {
	return fmt.Sprintf("%s に関する複数ファイル調査を完了しました。", target)
}

func (ee *ExecutionEngine) generateProblemAnalysis(steps []ToolStep) string {
	return "現在の状況とプロジェクトの健全性を分析しました。"
}

func (ee *ExecutionEngine) generateLearningGuidance(steps []ToolStep, topic string) string {
	if len(steps) == 0 {
		return "学習支援の実行中にエラーが発生しました。"
	}

	successfulSteps := 0
	var findings []string

	for _, step := range steps {
		if step.Success && step.Output != "" {
			successfulSteps++

			// 出力の要約生成
			if len(step.Output) > 100 {
				findings = append(findings, fmt.Sprintf("- %s コマンドで有用な情報を発見", step.Tool))
			}
		}
	}

	if successfulSteps == 0 {
		return fmt.Sprintf("%s についての詳細情報は見つかりませんでした。プロジェクト内で関連するファイルや実装を探してみることをお勧めします。", topic)
	}

	summary := fmt.Sprintf("🎓 %s について %d個のツールで情報を収集しました。", topic, successfulSteps)

	if len(findings) > 0 {
		summary += "\n\n発見した情報:\n" + strings.Join(findings, "\n")
	}

	// トピック固有のガイダンス追加
	switch strings.ToLower(topic) {
	case "vibe", "vibe_mode":
		summary += "\n\n💡 バイブモードは vyb-code の対話型コーディング機能です。"
	case "executionengine", "execution":
		summary += "\n\n⚡ ExecutionEngine はコマンド実行とマルチツール連携を担当するコンポーネントです。"
	case "workflow":
		summary += "\n\n🔄 ワークフローは複数のツールを組み合わせた自動実行システムです。"
	}

	return summary
}

// 学習タイプの判定
func (ee *ExecutionEngine) determineLearningType(inputLower, topic string) string {
	// ファイル固有の質問
	if strings.HasSuffix(topic, ".md") || strings.HasSuffix(topic, ".go") ||
		strings.HasSuffix(topic, ".json") || strings.HasSuffix(topic, ".yml") {
		return "file_specific"
	}

	// アーキテクチャ理解
	if strings.Contains(inputLower, "アーキテクチャ") || strings.Contains(inputLower, "architecture") ||
		strings.Contains(inputLower, "構造") || strings.Contains(inputLower, "structure") ||
		strings.Contains(inputLower, "設計") || strings.Contains(inputLower, "design") ||
		strings.Contains(inputLower, "プロジェクト") || strings.Contains(inputLower, "project") {
		return "architecture_understanding"
	}

	// コード分析
	if strings.Contains(inputLower, "関数") || strings.Contains(inputLower, "function") ||
		strings.Contains(inputLower, "func") || strings.Contains(inputLower, "method") ||
		strings.Contains(inputLower, "struct") || strings.Contains(inputLower, "type") ||
		strings.Contains(inputLower, "実装") || strings.Contains(inputLower, "implement") ||
		strings.ContainsAny(topic, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return "code_analysis"
	}

	// 概念説明
	if strings.Contains(inputLower, "とは") || strings.Contains(inputLower, "what is") ||
		strings.Contains(inputLower, "説明") || strings.Contains(inputLower, "explain") ||
		strings.Contains(inputLower, "について") || strings.Contains(inputLower, "理解") {
		return "concept_explanation"
	}

	return "general"
}

// 安全なgrepコマンドの構築
func (ee *ExecutionEngine) buildSafeGrepCommand(topic, searchType string) string {
	// 特殊文字のエスケープ
	escapedTopic := strings.ReplaceAll(topic, "'", "\\'")
	escapedTopic = strings.ReplaceAll(escapedTopic, "\"", "\\\"")
	escapedTopic = strings.ReplaceAll(escapedTopic, "`", "\\`")

	switch searchType {
	case "definition":
		return fmt.Sprintf("grep -r \"type %s\" . --include='*.go' | head -5", escapedTopic)
	case "usage":
		return fmt.Sprintf("grep -r \"%s\" . --include='*.go' | head -10", escapedTopic)
	case "struct_func":
		return fmt.Sprintf("grep -r \"func.*%s\\|struct.*%s\" . --include='*.go' | head -8", escapedTopic, escapedTopic)
	case "type_def":
		return fmt.Sprintf("grep -r \"type.*%s\" . --include='*.go' | head -5", escapedTopic)
	case "general":
		return fmt.Sprintf("grep -ri \"%s\" . --exclude-dir=.git --exclude-dir=vendor | head -10", escapedTopic)
	default:
		return fmt.Sprintf("grep -r \"%s\" . --include='*.go' | head -5", escapedTopic)
	}
}

// ファイル読み取りコマンド推論
func (ee *ExecutionEngine) inferFileReadCommand(inputLower, originalInput string) string {
	// ファイル名の推測
	if strings.Contains(inputLower, "readme") {
		return "cat README.md"
	}
	if strings.Contains(inputLower, "package.json") {
		return "cat package.json"
	}
	if strings.Contains(inputLower, "go.mod") {
		return "cat go.mod"
	}
	if strings.Contains(inputLower, "makefile") {
		return "cat Makefile"
	}

	// Claude.md の特別処理
	if strings.Contains(inputLower, "claude.md") || strings.Contains(inputLower, "claude") {
		return "cat CLAUDE.md"
	}

	// ファイル名が明示的に含まれている場合
	words := strings.Fields(originalInput)
	for _, word := range words {
		if strings.Contains(word, ".") && (strings.HasSuffix(word, ".go") ||
			strings.HasSuffix(word, ".md") || strings.HasSuffix(word, ".json") ||
			strings.HasSuffix(word, ".yml") || strings.HasSuffix(word, ".yaml")) {
			return "cat " + word
		}
	}

	// デフォルト: README.mdを表示
	return "cat README.md"
}

// ファイル検索コマンド推論
func (ee *ExecutionEngine) inferSearchCommand(inputLower, originalInput string) string {
	// 検索対象の抽出
	searchTarget := ""

	// "BashTool" のような具体的な検索語を抽出
	words := strings.Fields(originalInput)
	for _, word := range words {
		if len(word) > 3 && !strings.Contains(word, "検索") && !strings.Contains(word, "search") {
			// 単語が意味のある検索語と思われる場合
			if strings.Contains(strings.ToUpper(word), "TOOL") ||
				strings.Contains(word, "func") || strings.Contains(word, "type") ||
				len(word) > 5 {
				searchTarget = word
				break
			}
		}
	}

	if searchTarget == "" {
		// パターンマッチでの検索語抽出
		if strings.Contains(inputLower, "bashtool") {
			searchTarget = "BashTool"
		} else if strings.Contains(inputLower, "function") {
			searchTarget = "func"
		} else if strings.Contains(inputLower, "struct") {
			searchTarget = "struct"
		}
	}

	if searchTarget != "" {
		return fmt.Sprintf("grep -r \"%s\" . --exclude-dir=.git", searchTarget)
	}

	// デフォルト: Go ファイルの検索
	return "find . -name '*.go' | head -20"
}

// ファイル作成コマンド推論
func (ee *ExecutionEngine) inferFileWriteCommand(inputLower, originalInput string) string {
	filename := "example.go"

	// ファイル名の推測
	words := strings.Fields(originalInput)
	for _, word := range words {
		if strings.Contains(word, ".go") || strings.Contains(word, ".md") ||
			strings.Contains(word, ".txt") || strings.Contains(word, ".json") {
			filename = word
			break
		}
	}

	// テストファイルの場合
	if strings.Contains(inputLower, "test") {
		filename = "test_example.go"
	}

	// Hello World の場合
	if strings.Contains(inputLower, "hello") {
		content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
		return fmt.Sprintf("cat > %s << 'EOF'\n%sEOF", filename, content)
	}

	// 基本的なGo ファイル作成
	content := `package main

import "fmt"

// HelloWorld は挨拶を表示する関数
func HelloWorld() {
	fmt.Println("Hello from vyb!")
}

func main() {
	HelloWorld()
}
`
	return fmt.Sprintf("cat > %s << 'EOF'\n%sEOF", filename, content)
}

// Git関連の意図を検出
func (ee *ExecutionEngine) containsGitIntent(input string) bool {
	gitPatterns := []string{
		// 明確なgitコマンド
		"git status", "git diff", "git log", "git branch", "git show",
		"git remote", "git tag", "git stash", "git reflog", "git add", "git commit",
		// Git固有の日本語パターン
		"変更を確認", "コミット履歴", "git状態", "変更ファイル",
		"追跡状況", "リポジトリ状態", "コミット状況",
		// ブランチ比較パターン
		"ブランチ", "main", "比較", "変更を行った", "どういう変更",
		"mainと比較", "ブランチ間", "差分", "違い",
		// Git固有の自然言語パターン
		"何が変更された", "変更されたファイル", "uncommitted",
		"staged", "unstaged", "tracking", "untracked",
		// 未ステージング関連（優先パターン）
		"未ステージング", "未ステージングファイル", "ステージングされていない",
		"現状の変更", "現状.*変更", "変更.*確認",
	}

	for _, pattern := range gitPatterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}
	return false
}

// ファイル探索の意図を検出
func (ee *ExecutionEngine) containsFileIntent(input string) bool {
	// 先にGit関連の除外をチェック
	if strings.Contains(input, "git") || strings.Contains(input, "未ステージング") ||
		strings.Contains(input, "現状") && strings.Contains(input, "変更") {
		return false
	}

	filePatterns := []string{
		// 基本コマンド
		"ls", "ll", "find", "tree", "dir", "list",
		// 日本語パターン（Git固有のものは除外）
		"ファイル一覧", "ディレクトリ", "ファイルを探", "構造を確認",
		"一覧表示", "ファイル表示", "フォルダ一覧", "ディレクトリ構造",
		"ファイル構造", "プロジェクト構造", "ファイルリスト",
		// 自然言語パターン
		"何があるか", "どんなファイル", "ファイルの中身", "含まれているファイル",
		"プロジェクトの中身", "ディレクトリの中", "フォルダの中",
		"show files", "list files", "show directory", "file structure",
		"project structure", "what files", "contents of", "files in",
		// 探索系（Gitではない文脈のもの）
		"探す", "検索", "見つける", "調べる",
		"search", "explore", "browse", "navigate",
	}

	for _, pattern := range filePatterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}
	return false
}

// プロジェクト分析の意図を検出
func (ee *ExecutionEngine) containsAnalysisIntent(input string) bool {
	analysisPatterns := []string{
		// Go専用
		"go mod", "go list", "go version", "go env", "go vet", "go fmt",
		"go test", "go build", "go run", "go clean", "go get",
		// 一般的な分析
		"プロジェクト分析", "コード分析", "依存関係", "構造を分析",
		"品質", "メトリクス", "テスト実行", "ビルド",
		// 自然言語
		"依存関係を確認", "モジュールを確認", "テストを実行",
		"プロジェクトをビルド", "コードを検証", "フォーマット",
		"analyze project", "check dependencies", "run tests", "build project",
		"code quality", "project health", "module info", "package info",
		// ツール系
		"npm", "yarn", "pip", "cargo", "maven", "gradle", "make",
		"package.json", "go.mod", "Cargo.toml", "requirements.txt",
	}

	for _, pattern := range analysisPatterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}
	return false
}

// Git コマンドを推測
func (ee *ExecutionEngine) inferGitCommand(input string) string {
	// ブランチ比較パターンの処理
	if (strings.Contains(input, "main") && strings.Contains(input, "比較")) ||
		strings.Contains(input, "mainと比較") ||
		(strings.Contains(input, "ブランチ") && strings.Contains(input, "変更")) ||
		strings.Contains(input, "どういう変更") {
		return "git diff main --name-status"
	}

	// 未ステージングファイルの言及がある場合は必ずgit status
	if strings.Contains(input, "未ステージング") || strings.Contains(input, "unstaged") {
		return "git status"
	}

	// 現状の変更確認
	if strings.Contains(input, "現状") && (strings.Contains(input, "変更") || strings.Contains(input, "ファイル")) {
		return "git status"
	}

	// 従来のパターン
	if strings.Contains(input, "status") || strings.Contains(input, "ステータス") ||
		strings.Contains(input, "ファイルを分析") {
		return "git status"
	}
	if strings.Contains(input, "diff") && !strings.Contains(input, "未ステージング") {
		return "git diff"
	}
	if strings.Contains(input, "log") || strings.Contains(input, "履歴") {
		return "git log --oneline -10"
	}
	if strings.Contains(input, "branch") || strings.Contains(input, "ブランチ") {
		return "git branch -v"
	}
	return "git status" // デフォルト
}

// ファイルコマンドを推測
func (ee *ExecutionEngine) inferFileCommand(input string) string {
	// 明確なlsコマンド系
	if strings.Contains(input, "ls") || strings.Contains(input, "一覧") ||
		strings.Contains(input, "list") {
		return "ls -la"
	}

	// 詳細一覧
	if strings.Contains(input, "詳細") || strings.Contains(input, "ll") ||
		strings.Contains(input, "long") {
		return "ls -la"
	}

	// 構造表示
	if strings.Contains(input, "構造") || strings.Contains(input, "tree") ||
		strings.Contains(input, "structure") {
		return "find . -type d -name '.git' -prune -o -type d -print | head -15"
	}

	// ファイル検索
	if strings.Contains(input, "find") || strings.Contains(input, "探") ||
		strings.Contains(input, "search") {
		return "find . -type f -not -path './.git/*' | head -20"
	}

	// Go ファイル検索
	if strings.Contains(input, "go") || strings.Contains(input, "*.go") {
		return "find . -name '*.go' -not -path './.git/*' | head -20"
	}

	// デフォルト: シンプルな一覧
	return "ls -la"
}

// 分析コマンドを推測
func (ee *ExecutionEngine) inferAnalysisCommand(input string) string {
	// Go関連
	if strings.Contains(input, "go mod") {
		return "go mod tidy && go list -m all"
	}
	if strings.Contains(input, "go test") {
		return "go test -v ./..."
	}
	if strings.Contains(input, "go build") {
		return "go build -v ./..."
	}
	if strings.Contains(input, "go") || strings.Contains(input, "依存関係") {
		return "go list -m all"
	}

	// Node.js関連
	if strings.Contains(input, "npm") {
		return "npm list --depth=0"
	}
	if strings.Contains(input, "yarn") {
		return "yarn list --depth=0"
	}
	if strings.Contains(input, "node") {
		return "node --version && npm --version"
	}

	// Python関連
	if strings.Contains(input, "python") || strings.Contains(input, "pip") {
		return "python --version && pip list"
	}

	// 一般的なプロジェクト分析
	if strings.Contains(input, "分析") || strings.Contains(input, "analyze") {
		return "find . -type f -name '*.go' | wc -l && echo 'Go files found'"
	}

	return "ls -la && echo 'Project analysis'"
}

// コマンドを安全に実行
func (ee *ExecutionEngine) ExecuteCommand(command string) (*ExecutionResult, error) {
	if !ee.enabled {
		return nil, fmt.Errorf("実行エンジンが無効です")
	}

	// マルチツールワークフロー処理
	if strings.HasPrefix(command, "multi-tool:") {
		workflowType := strings.TrimPrefix(command, "multi-tool:")
		return ee.executeMultiToolWorkflowAndFormat(workflowType, command)
	}

	// キャッシュチェック（パフォーマンス最適化）
	if cached := ee.getCachedResult(command); cached != nil {
		return cached, nil
	}

	// コマンドの安全性をチェック
	if !ee.isCommandSafe(command) {
		return &ExecutionResult{
			Command:   command,
			Error:     "コマンドが安全性チェックを通過しませんでした",
			ExitCode:  -1,
			Timestamp: time.Now(),
		}, fmt.Errorf("unsafe command: %s", command)
	}

	result, err := ee.runCommand(command)

	// 成功した場合はキャッシュに保存
	if err == nil && result != nil {
		ee.cacheResult(command, result)
	}

	return result, err
}

// コマンドの安全性をチェック
func (ee *ExecutionEngine) isCommandSafe(command string) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	baseCmd := parts[0]

	// 許可されたコマンドかチェック
	if !ee.allowedCommands[baseCmd] {
		return false
	}

	// 危険なオプションをチェック
	dangerousPatterns := []string{
		"rm ", "mv ", "cp /", "> /", ">> /",
		"sudo", "su ", "chmod", "chown",
		"--delete", "--force", "-f",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return false
		}
	}

	return true
}

// 実際にコマンドを実行
func (ee *ExecutionEngine) runCommand(command string) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Command:   command,
		Timestamp: time.Now(),
	}

	start := time.Now()

	// コマンドを分解
	parts := strings.Fields(command)
	if len(parts) == 0 {
		result.Error = "空のコマンド"
		result.ExitCode = -1
		return result, fmt.Errorf("empty command")
	}

	// 実行ディレクトリを設定
	var cmd *exec.Cmd

	// パイプが含まれている場合はシェルを経由して実行
	if strings.Contains(command, "|") || strings.Contains(command, ">") || strings.Contains(command, "<") {
		cmd = exec.Command("sh", "-c", command)
	} else {
		cmd = exec.Command(parts[0], parts[1:]...)
	}
	cmd.Dir = ee.projectPath

	// タイムアウト処理の改善
	done := make(chan struct{})
	var output []byte
	var err error

	go func() {
		defer close(done)
		output, err = cmd.CombinedOutput()
	}()

	// タイムアウトチェック
	select {
	case <-done:
		// 正常終了
	case <-time.After(ee.safetyLimits.MaxExecutionTime):
		// タイムアウト時はプロセスを強制終了
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		result.Error = fmt.Sprintf("コマンドタイムアウト (%v)", ee.safetyLimits.MaxExecutionTime)
		result.ExitCode = -1
		result.Duration = time.Since(start)
		return result, fmt.Errorf("command timeout after %v", ee.safetyLimits.MaxExecutionTime)
	}

	result.Duration = time.Since(start)

	// エラーハンドリングの改善
	if err != nil {
		result.Error = ee.formatError(err, command)
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}

		// 特定のエラーの場合は有用な情報を提供
		if strings.Contains(err.Error(), "executable file not found") {
			result.Error += " (コマンドが見つかりません。パスを確認してください)"
		} else if strings.Contains(err.Error(), "permission denied") {
			result.Error += " (権限不足です。実行権限を確認してください)"
		}
	} else {
		result.ExitCode = 0
	}

	// 出力サイズ制限とフォーマット改善
	outputStr := string(output)
	if len(outputStr) > ee.safetyLimits.MaxOutputSize {
		// 先頭と末尾の重要部分を保持
		halfSize := ee.safetyLimits.MaxOutputSize / 2
		outputStr = outputStr[:halfSize] +
			"\n... [中間部分省略] ...\n" +
			outputStr[len(outputStr)-halfSize:]
		result.Truncated = true
	}

	result.Output = outputStr
	return result, nil
}

// エラーメッセージをユーザーフレンドリーにフォーマット
func (ee *ExecutionEngine) formatError(err error, command string) string {
	baseErr := err.Error()

	// 一般的なエラーパターンを翻訳
	errorMappings := map[string]string{
		"no such file or directory": "ファイルまたはディレクトリが見つかりません",
		"permission denied":         "権限が拒否されました",
		"command not found":         "コマンドが見つかりません",
		"connection refused":        "接続が拒否されました",
		"network is unreachable":    "ネットワークに到達できません",
	}

	for pattern, translation := range errorMappings {
		if strings.Contains(strings.ToLower(baseErr), pattern) {
			return fmt.Sprintf("%s (%s)", translation, baseErr)
		}
	}

	return baseErr
}

// エラー解決の提案を生成
func (ee *ExecutionEngine) suggestErrorResolution(errorMsg, command string) string {
	var suggestions []string

	// コマンド固有の提案
	parts := strings.Fields(command)
	if len(parts) > 0 {
		baseCmd := parts[0]

		switch baseCmd {
		case "git":
			if strings.Contains(errorMsg, "not a git repository") {
				suggestions = append(suggestions, "`git init` でリポジトリを初期化")
			} else if strings.Contains(errorMsg, "nothing to commit") {
				suggestions = append(suggestions, "`git add .` でファイルをステージング")
			}

		case "go":
			if strings.Contains(errorMsg, "no Go files") {
				suggestions = append(suggestions, "Goファイル（*.go）が存在するディレクトリで実行")
			} else if strings.Contains(errorMsg, "cannot find module") {
				suggestions = append(suggestions, "`go mod init <module-name>` でモジュールを初期化")
			}

		case "npm", "yarn":
			if strings.Contains(errorMsg, "package.json") {
				suggestions = append(suggestions, "`npm init` でpackage.jsonを作成")
			}

		case "python":
			if strings.Contains(errorMsg, "command not found") {
				suggestions = append(suggestions, "`python3` または `py` コマンドを試す")
			}
		}
	}

	// 一般的なエラー対応
	if strings.Contains(errorMsg, "permission denied") {
		suggestions = append(suggestions, "ファイルの権限を確認（`chmod +x filename`）")
	} else if strings.Contains(errorMsg, "command not found") {
		suggestions = append(suggestions, "必要なソフトウェアがインストールされているか確認")
	}

	if len(suggestions) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("💡 **解決提案**:\n")
	for _, suggestion := range suggestions {
		builder.WriteString(fmt.Sprintf("• %s\n", suggestion))
	}

	return builder.String()
}

// 実行結果をユーザーフレンドリーにフォーマット
func (ee *ExecutionEngine) FormatExecutionResult(result *ExecutionResult, analysis *ContextAnalysis) string {
	var builder strings.Builder

	// 実行情報のヘッダーを追加
	builder.WriteString(fmt.Sprintf("⚡ **実行完了** (`%s`) - %v\n\n",
		result.Command, result.Duration.Round(time.Millisecond)))

	if result.ExitCode != 0 {
		builder.WriteString(fmt.Sprintf("❌ **エラー**: %s\n\n", result.Error))

		// エラーの場合も可能な限り出力を表示
		if result.Output != "" {
			builder.WriteString("📝 **出力**:\n```\n")
			builder.WriteString(result.Output)
			builder.WriteString("\n```\n")
		}

		// エラー解決の提案
		builder.WriteString(ee.suggestErrorResolution(result.Error, result.Command))
		return builder.String()
	}

	// コンテキストに応じた結果の解釈
	switch analysis.Intent {
	case "git_analysis":
		builder.WriteString(ee.interpretGitOutput(result.Output, result.Command))
	case "file_exploration":
		builder.WriteString(ee.interpretFileOutput(result.Output, result.Command))
	case "project_analysis":
		builder.WriteString(ee.interpretAnalysisOutput(result.Output, result.Command))
	default:
		builder.WriteString("📝 **結果**:\n```\n")
		builder.WriteString(result.Output)
		builder.WriteString("\n```\n")
	}

	// 追加情報
	if result.Truncated {
		builder.WriteString("\n⚠️  出力が長すぎるため一部省略されました\n")
	}

	// パフォーマンス情報
	if result.Duration > time.Second {
		builder.WriteString(fmt.Sprintf("\n⏱️  実行時間: %v (通常より時間がかかりました)\n", result.Duration))
	}

	return builder.String()
}

// Git出力の解釈
func (ee *ExecutionEngine) interpretGitOutput(output, command string) string {
	var builder strings.Builder

	if strings.Contains(command, "diff") {
		return ee.interpretGitDiff(output)
	} else if strings.Contains(command, "status") {
		return ee.interpretGitStatus(output)
	} else {
		builder.WriteString(fmt.Sprintf("```\n%s\n```\n", output))
	}

	return builder.String()
}

// Git status の詳細解釈
func (ee *ExecutionEngine) interpretGitStatus(output string) string {
	var builder strings.Builder
	builder.WriteString("📊 **Gitステータス詳細分析**:\n\n")

	lines := strings.Split(output, "\n")
	var modifiedFiles []string
	var untrackedFiles []string
	var branch string

	// ファイル情報を解析
	inUntracked := false
	inChanges := false

	for _, line := range lines {
		origLine := line // タブ含む元の行を保持
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "On branch") {
			branch = strings.TrimPrefix(line, "On branch ")
		} else if strings.Contains(line, "Changes not staged for commit:") {
			inChanges = true
			inUntracked = false
		} else if strings.Contains(line, "Changes to be committed:") {
			inChanges = true
			inUntracked = false
		} else if strings.Contains(line, "Untracked files:") {
			inUntracked = true
			inChanges = false
		} else if inChanges && strings.HasPrefix(origLine, "\t") && strings.Contains(line, "modified:") {
			filename := strings.TrimSpace(strings.TrimPrefix(line, "modified:"))
			if filename != "" {
				modifiedFiles = append(modifiedFiles, filename)
			}
		} else if inChanges && strings.HasPrefix(origLine, "\t") && strings.Contains(line, "new file:") {
			filename := strings.TrimSpace(strings.TrimPrefix(line, "new file:"))
			if filename != "" {
				modifiedFiles = append(modifiedFiles, filename)
			}
		} else if inChanges && strings.HasPrefix(origLine, "\t") && strings.Contains(line, "deleted:") {
			filename := strings.TrimSpace(strings.TrimPrefix(line, "deleted:"))
			if filename != "" {
				modifiedFiles = append(modifiedFiles, filename)
			}
		} else if inUntracked && strings.HasPrefix(origLine, "\t") && !strings.Contains(line, "(use") && !strings.HasPrefix(line, "(") {
			filename := strings.TrimSpace(line)
			if filename != "" && !strings.Contains(filename, "no changes") {
				untrackedFiles = append(untrackedFiles, filename)
			}
		} else if strings.Contains(line, "no changes added to commit") {
			// git status の最後の行に達したら状態をリセット
			inUntracked = false
			inChanges = false
		}
	}

	// ブランチとサマリー
	if branch != "" {
		builder.WriteString(fmt.Sprintf("🌿 **現在のブランチ**: `%s`\n", branch))
	}

	builder.WriteString(fmt.Sprintf("📊 **サマリー**: 変更済み %d個、未追跡 %d個\n\n", len(modifiedFiles), len(untrackedFiles)))

	// 変更されたファイルの詳細分析
	if len(modifiedFiles) > 0 {
		builder.WriteString("🔄 **変更されたファイル**:\n")
		for _, file := range modifiedFiles {
			analysis := ee.analyzeFileChange(file)
			builder.WriteString(fmt.Sprintf("• **%s**\n", file))
			builder.WriteString(fmt.Sprintf("  - 種類: %s\n", analysis.FileType))
			builder.WriteString(fmt.Sprintf("  - 役割: %s\n", analysis.Purpose))
			if analysis.PotentialImpact != "" {
				builder.WriteString(fmt.Sprintf("  - 影響: %s\n", analysis.PotentialImpact))
			}
		}
		builder.WriteString("\n")
	}

	// 未追跡ファイルの分析
	if len(untrackedFiles) > 0 {
		builder.WriteString("📁 **未追跡ファイル**:\n")
		for _, file := range untrackedFiles {
			analysis := ee.analyzeNewFile(file)
			builder.WriteString(fmt.Sprintf("• **%s**\n", file))
			builder.WriteString(fmt.Sprintf("  - 種類: %s\n", analysis.FileType))
			if analysis.Purpose != "" {
				builder.WriteString(fmt.Sprintf("  - 目的: %s\n", analysis.Purpose))
			}
		}
		builder.WriteString("\n")
	}

	// 原始出力
	builder.WriteString("📋 **生出力**:\n```\n")
	builder.WriteString(output)
	builder.WriteString("\n```\n\n")

	// インテリジェントな推奨アクション
	builder.WriteString("💡 **推奨アクション**:\n")
	if len(modifiedFiles) > 0 {
		builder.WriteString("• `git diff <file>` で具体的な変更内容を確認\n")
		for _, file := range modifiedFiles {
			if strings.Contains(file, "config") {
				builder.WriteString("• 設定変更後はテストして動作確認\n")
				break
			}
		}
	}
	if len(untrackedFiles) > 0 {
		builder.WriteString("• `git add .` で全ファイルをステージング\n")
		// ドキュメントファイルがある場合の特別な推奨
		hasDocFiles := false
		for _, file := range untrackedFiles {
			if strings.HasSuffix(file, ".md") {
				hasDocFiles = true
				break
			}
		}
		if hasDocFiles {
			builder.WriteString("• ドキュメントファイルは最新の実装と整合性を確認\n")
		}
	}
	if len(modifiedFiles) > 0 || len(untrackedFiles) > 0 {
		builder.WriteString("• `git commit -m \"meaningful message\"` で変更をコミット\n")
		builder.WriteString("• コミットメッセージは変更の目的と影響を明記\n")
	}

	return builder.String()
}

// ファイル分析結果
type FileAnalysis struct {
	FileType        string
	Purpose         string
	PotentialImpact string
}

// 変更されたファイルを分析
func (ee *ExecutionEngine) analyzeFileChange(filePath string) *FileAnalysis {
	analysis := &FileAnalysis{}

	// ファイル種別の判定
	if strings.HasSuffix(filePath, ".go") {
		analysis.FileType = "Go source file"

		// ファイルパスから役割を推定
		if strings.Contains(filePath, "/config/") {
			analysis.Purpose = "アプリケーション設定管理"
			analysis.PotentialImpact = "設定変更により動作が変化する可能性"
		} else if strings.Contains(filePath, "/chat/") {
			analysis.Purpose = "チャット・セッション管理"
			analysis.PotentialImpact = "ユーザーインタラクション動作に影響"
		} else if strings.Contains(filePath, "/interactive/") {
			analysis.Purpose = "インタラクティブモード制御"
			analysis.PotentialImpact = "UI/UX体験に直接影響"
		} else if strings.Contains(filePath, "/llm/") {
			analysis.Purpose = "LLM統合・通信制御"
			analysis.PotentialImpact = "AI応答品質・性能に影響"
		} else if strings.Contains(filePath, "/tools/") {
			analysis.Purpose = "ツール・コマンド実行"
			analysis.PotentialImpact = "コマンド実行機能に影響"
		} else if strings.Contains(filePath, "/conversation/") {
			analysis.Purpose = "会話処理・AI機能"
			analysis.PotentialImpact = "AI応答の質と機能に影響"
		} else {
			analysis.Purpose = "コア機能実装"
			analysis.PotentialImpact = "基本動作に影響する可能性"
		}
	} else if strings.HasSuffix(filePath, ".md") {
		analysis.FileType = "Markdown documentation"
		analysis.Purpose = "プロジェクトドキュメント"
		analysis.PotentialImpact = "ドキュメント更新（機能への直接影響なし）"
	} else if strings.HasSuffix(filePath, ".json") {
		analysis.FileType = "JSON configuration"
		analysis.Purpose = "設定・メタデータ"
		analysis.PotentialImpact = "設定変更により動作が変化"
	} else {
		analysis.FileType = "Other file"
		analysis.Purpose = "不明"
	}

	return analysis
}

// 新しいファイルを分析
func (ee *ExecutionEngine) analyzeNewFile(filePath string) *FileAnalysis {
	analysis := &FileAnalysis{}

	if strings.HasSuffix(filePath, ".go") {
		analysis.FileType = "Go source file (新規)"

		// Phase判定
		if strings.Contains(filePath, "conversation") {
			analysis.Purpose = "新しい会話処理機能"
		} else if strings.Contains(filePath, "analysis") {
			analysis.Purpose = "プロジェクト分析機能"
		} else if strings.Contains(filePath, "performance") {
			analysis.Purpose = "パフォーマンス監視機能"
		} else if strings.Contains(filePath, "interactive") {
			analysis.Purpose = "インタラクティブ機能拡張"
		} else {
			analysis.Purpose = "新機能実装"
		}
	} else if strings.HasSuffix(filePath, ".md") {
		analysis.FileType = "Markdown document (新規)"
		if strings.Contains(strings.ToUpper(filePath), "PHASE") {
			analysis.Purpose = "開発フェーズドキュメント"
		} else if strings.Contains(strings.ToUpper(filePath), "SUMMARY") {
			analysis.Purpose = "実装サマリー文書"
		} else {
			analysis.Purpose = "プロジェクトドキュメント"
		}
	} else if strings.HasSuffix(filePath, "/") {
		analysis.FileType = "Directory (新規)"
		analysis.Purpose = "新機能用ディレクトリ"
	} else {
		analysis.FileType = "Other file (新規)"
		analysis.Purpose = "新規追加ファイル"
	}

	return analysis
}

// Git diff の解釈
func (ee *ExecutionEngine) interpretGitDiff(output string) string {
	var builder strings.Builder
	builder.WriteString("📝 **ブランチ間差分分析**:\n\n")

	if strings.TrimSpace(output) == "" {
		builder.WriteString("✅ **結果**: mainブランチとの差分はありません（同期済み）\n")
		return builder.String()
	}

	lines := strings.Split(output, "\n")
	var addedFiles []string
	var modifiedFiles []string
	var deletedFiles []string

	// --name-status フォーマットを解析
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			status := parts[0]
			filename := parts[1]

			switch status {
			case "A":
				addedFiles = append(addedFiles, filename)
			case "M":
				modifiedFiles = append(modifiedFiles, filename)
			case "D":
				deletedFiles = append(deletedFiles, filename)
			}
		}
	}

	// サマリーを表示
	totalChanges := len(addedFiles) + len(modifiedFiles) + len(deletedFiles)
	builder.WriteString(fmt.Sprintf("🔢 **変更サマリー**: 全 %d ファイル (", totalChanges))

	summary := []string{}
	if len(addedFiles) > 0 {
		summary = append(summary, fmt.Sprintf("新規 %d", len(addedFiles)))
	}
	if len(modifiedFiles) > 0 {
		summary = append(summary, fmt.Sprintf("変更 %d", len(modifiedFiles)))
	}
	if len(deletedFiles) > 0 {
		summary = append(summary, fmt.Sprintf("削除 %d", len(deletedFiles)))
	}
	builder.WriteString(strings.Join(summary, "、"))
	builder.WriteString(")\n\n")

	// 詳細リストを表示
	if len(addedFiles) > 0 {
		builder.WriteString("🆕 **新規追加ファイル**:\n")
		for _, file := range addedFiles {
			builder.WriteString(fmt.Sprintf("• %s\n", file))
		}
		builder.WriteString("\n")
	}

	if len(modifiedFiles) > 0 {
		builder.WriteString("📝 **変更ファイル**:\n")
		for _, file := range modifiedFiles {
			builder.WriteString(fmt.Sprintf("• %s\n", file))
		}
		builder.WriteString("\n")
	}

	if len(deletedFiles) > 0 {
		builder.WriteString("🗑️ **削除ファイル**:\n")
		for _, file := range deletedFiles {
			builder.WriteString(fmt.Sprintf("• %s\n", file))
		}
		builder.WriteString("\n")
	}

	// コミット履歴も表示
	builder.WriteString("💡 **推奨アクション**:\n")
	builder.WriteString("• `git log main..HEAD --oneline` でコミット履歴を確認\n")
	builder.WriteString("• `git diff main <file>` で具体的な変更内容を確認\n")

	return builder.String()
}

// ファイル出力の解釈
func (ee *ExecutionEngine) interpretFileOutput(output, command string) string {
	var builder strings.Builder
	builder.WriteString("📁 **ファイル構造**:\n")

	lines := strings.Split(output, "\n")
	fileCount := 0
	dirCount := 0
	var importantFiles []string
	var largeFiles []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "d") {
			dirCount++
		} else if strings.HasPrefix(line, "-") {
			fileCount++

			// 重要ファイルを特定
			if strings.Contains(line, "go.mod") ||
				strings.Contains(line, "package.json") ||
				strings.Contains(line, "Cargo.toml") ||
				strings.Contains(line, "requirements.txt") ||
				strings.Contains(line, "Makefile") ||
				strings.Contains(line, "README") {
				parts := strings.Fields(line)
				if len(parts) > 8 {
					importantFiles = append(importantFiles, parts[8])
				}
			}

			// 大きなファイルを特定（10MB以上）
			parts := strings.Fields(line)
			if len(parts) > 4 {
				sizeStr := parts[4]
				if strings.HasSuffix(sizeStr, "M") ||
					(strings.HasSuffix(sizeStr, "K") && len(sizeStr) > 5) {
					if len(parts) > 8 {
						largeFiles = append(largeFiles, parts[8])
					}
				}
			}
		}
	}

	// 統計情報
	if fileCount > 0 || dirCount > 0 {
		builder.WriteString(fmt.Sprintf("📊 **統計**: ディレクトリ %d個、ファイル %d個\n\n", dirCount, fileCount))
	}

	// 重要ファイルの表示
	if len(importantFiles) > 0 {
		builder.WriteString("🔍 **重要ファイル**:\n")
		for _, file := range importantFiles {
			builder.WriteString(fmt.Sprintf("• %s\n", file))
		}
		builder.WriteString("\n")
	}

	// 大きなファイルの警告
	if len(largeFiles) > 0 {
		builder.WriteString("⚠️  **大きなファイル**:\n")
		for _, file := range largeFiles {
			builder.WriteString(fmt.Sprintf("• %s\n", file))
		}
		builder.WriteString("\n")
	}

	// 詳細出力
	builder.WriteString("📝 **詳細**:\n```\n")
	builder.WriteString(output)
	builder.WriteString("\n```\n")

	// 推奨アクション
	if len(importantFiles) > 0 {
		builder.WriteString("💡 **推奨アクション**:\n")
		for _, file := range importantFiles {
			if strings.Contains(file, "go.mod") {
				builder.WriteString("• `go mod tidy` で依存関係を整理\n")
			} else if strings.Contains(file, "package.json") {
				builder.WriteString("• `npm install` で依存関係をインストール\n")
			} else if strings.Contains(file, "Makefile") {
				builder.WriteString("• `make` でビルドコマンドを確認\n")
			}
		}
	}

	return builder.String()
}

// 分析出力の解釈
func (ee *ExecutionEngine) interpretAnalysisOutput(output, command string) string {
	var builder strings.Builder
	builder.WriteString("🔍 **プロジェクト分析**:\n")
	builder.WriteString(fmt.Sprintf("```\n%s\n```\n", output))
	return builder.String()
}

// キャッシュからの結果取得
func (ee *ExecutionEngine) getCachedResult(command string) *ExecutionResult {
	if ee.cache == nil {
		return nil
	}

	entry, exists := ee.cache[command]
	if !exists {
		return nil
	}

	// TTLをチェック
	if time.Since(entry.Timestamp) > entry.TTL {
		delete(ee.cache, command) // 期限切れエントリを削除
		return nil
	}

	// キャッシュヒットの情報を追加
	cachedResult := *entry.Result // コピーを作成
	cachedResult.Timestamp = time.Now()
	cachedResult.Output += "\n\n💾 *キャッシュからの結果*"

	return &cachedResult
}

// 結果をキャッシュに保存
func (ee *ExecutionEngine) cacheResult(command string, result *ExecutionResult) {
	if ee.cache == nil {
		ee.cache = make(map[string]*CacheEntry)
	}

	// キャッシュ可能なコマンドかチェック
	if !ee.isCacheable(command) {
		return
	}

	// TTL設定（コマンドによって異なる）
	ttl := ee.getTTLForCommand(command)

	// キャッシュサイズ制限（最大50エントリ）
	if len(ee.cache) >= 50 {
		ee.evictOldestEntry()
	}

	ee.cache[command] = &CacheEntry{
		Result:    result,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

// コマンドがキャッシュ可能かチェック
func (ee *ExecutionEngine) isCacheable(command string) bool {
	// 以下のコマンドはキャッシュしない（状態が変化する可能性があるため）
	uncacheableCommands := []string{
		"git status", "git diff", "git log",
		"ps", "top", "df", "free", "uptime",
	}

	for _, uncacheable := range uncacheableCommands {
		if strings.Contains(command, uncacheable) {
			return false
		}
	}

	return true
}

// コマンドのTTL（Time To Live）を取得
func (ee *ExecutionEngine) getTTLForCommand(command string) time.Duration {
	// 静的なファイル情報系は長めのTTL
	if strings.Contains(command, "ls -la") ||
		strings.Contains(command, "find") {
		return 30 * time.Second
	}

	// プロジェクト分析系は中程度のTTL
	if strings.Contains(command, "go list") ||
		strings.Contains(command, "npm list") {
		return 60 * time.Second
	}

	// デフォルト
	return 15 * time.Second
}

// 最古のキャッシュエントリを削除
func (ee *ExecutionEngine) evictOldestEntry() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range ee.cache {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}

	if oldestKey != "" {
		delete(ee.cache, oldestKey)
	}
}

// キャッシュ統計情報を取得
func (ee *ExecutionEngine) GetCacheStats() map[string]interface{} {
	if ee.cache == nil {
		return map[string]interface{}{
			"enabled": false,
			"size":    0,
		}
	}

	// 有効なエントリをカウント
	validEntries := 0
	for _, entry := range ee.cache {
		if time.Since(entry.Timestamp) <= entry.TTL {
			validEntries++
		}
	}

	return map[string]interface{}{
		"enabled":       true,
		"total_size":    len(ee.cache),
		"valid_entries": validEntries,
		"hit_rate":      float64(validEntries) / float64(len(ee.cache)) * 100,
	}
}

// マルチツールワークフローの実行と結果フォーマット
func (ee *ExecutionEngine) executeMultiToolWorkflowAndFormat(workflowType, originalCommand string) (*ExecutionResult, error) {
	start := time.Now()

	// 保存されたユーザー入力を使用
	inputLower := strings.ToLower(ee.lastUserInput)
	userInput := ee.lastUserInput

	// ワークフローを実行
	multiResult, err := ee.executeMultiToolWorkflow(workflowType, inputLower, userInput)
	if err != nil {
		return &ExecutionResult{
			Command:   originalCommand,
			Error:     err.Error(),
			ExitCode:  -1,
			Timestamp: time.Now(),
		}, err
	}

	// 複数ステップの結果を単一のExecutionResultにフォーマット
	var outputBuilder strings.Builder

	outputBuilder.WriteString(fmt.Sprintf("🔧 マルチツールワークフロー実行: %s\n", workflowType))
	outputBuilder.WriteString(fmt.Sprintf("📊 実行時間: %v\n", multiResult.Duration))
	outputBuilder.WriteString(fmt.Sprintf("🛠️ 実行ステップ数: %d\n\n", len(multiResult.Steps)))

	for i, step := range multiResult.Steps {
		outputBuilder.WriteString(fmt.Sprintf("Step %d: %s\n", i+1, step.Tool))
		outputBuilder.WriteString(fmt.Sprintf("Command: %s\n", step.Command))
		if step.Success {
			outputBuilder.WriteString("✅ 成功\n")
			if step.Output != "" {
				outputBuilder.WriteString(fmt.Sprintf("Output:\n%s\n", step.Output))
			}
		} else {
			outputBuilder.WriteString("❌ 失敗\n")
		}
		outputBuilder.WriteString(fmt.Sprintf("Duration: %v\n", step.Duration))
		outputBuilder.WriteString("---\n")
	}

	if multiResult.Summary != "" {
		outputBuilder.WriteString("\n📋 サマリー:\n")
		outputBuilder.WriteString(multiResult.Summary)
	}

	return &ExecutionResult{
		Command:   originalCommand,
		Output:    outputBuilder.String(),
		ExitCode:  0,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}, nil
}

// クリーンアップ
func (ee *ExecutionEngine) Close() error {
	// キャッシュをクリア
	if ee.cache != nil {
		ee.cache = nil
	}
	return nil
}
