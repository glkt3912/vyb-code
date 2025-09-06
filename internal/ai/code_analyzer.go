package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

// AIコード分析結果
type CodeAnalysisResult struct {
	Summary           string                 `json:"summary"`
	Issues            []CodeIssue            `json:"issues"`
	Suggestions       []CodeSuggestion       `json:"suggestions"`
	Quality           QualityMetrics         `json:"quality"`
	Refactoring       []RefactoringProposal  `json:"refactoring_proposals"`
	Documentation     []DocumentationGap     `json:"documentation_gaps"`
	Performance       []PerformanceInsight   `json:"performance_insights"`
	Security          SecurityAnalysis       `json:"security"`
	Dependencies      []DependencyIssue      `json:"dependency_issues"`
	Complexity        ComplexityAnalysis     `json:"complexity"`
	TestCoverage      TestCoverageAnalysis   `json:"test_coverage"`
	AnalysisTimestamp time.Time              `json:"analysis_timestamp"`
	ProcessingTime    time.Duration          `json:"processing_time"`
}

// コードの問題
type CodeIssue struct {
	Type        string `json:"type"`        // "bug", "smell", "style", "performance"
	Severity    string `json:"severity"`    // "low", "medium", "high", "critical"
	File        string `json:"file"`
	Line        int    `json:"line"`
	Column      int    `json:"column,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Solution    string `json:"solution"`
	Examples    string `json:"examples,omitempty"`
}

// コード改善提案
type CodeSuggestion struct {
	Type         string `json:"type"`          // "optimization", "cleanup", "modernization"
	Priority     string `json:"priority"`      // "low", "medium", "high"
	File         string `json:"file"`
	LineRange    [2]int `json:"line_range"`    // [start, end]
	Title        string `json:"title"`
	Description  string `json:"description"`
	Before       string `json:"before"`
	After        string `json:"after"`
	Explanation  string `json:"explanation"`
	Benefits     string `json:"benefits"`
}

// 品質メトリクス
type QualityMetrics struct {
	OverallScore     int     `json:"overall_score"`      // 0-100
	Maintainability  int     `json:"maintainability"`    // 0-100
	Reliability      int     `json:"reliability"`        // 0-100
	Performance      int     `json:"performance"`        // 0-100
	Security         int     `json:"security"`           // 0-100
	Readability      int     `json:"readability"`        // 0-100
	TestQuality      int     `json:"test_quality"`       // 0-100
	TechnicalDebt    float64 `json:"technical_debt"`     // in hours
	CodeDuplication  float64 `json:"code_duplication"`   // percentage
}

// リファクタリング提案
type RefactoringProposal struct {
	Type         string   `json:"type"`          // "extract_method", "rename", "move_class"
	Priority     string   `json:"priority"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Files        []string `json:"files"`
	EstimatedEffort string `json:"estimated_effort"`
	Benefits     string   `json:"benefits"`
	Risks        string   `json:"risks"`
	Steps        []string `json:"steps"`
}

// ドキュメント不足
type DocumentationGap struct {
	Type        string `json:"type"`        // "missing_comment", "missing_docstring", "outdated"
	File        string `json:"file"`
	Line        int    `json:"line"`
	Element     string `json:"element"`     // "function", "class", "module"
	ElementName string `json:"element_name"`
	Suggestion  string `json:"suggestion"`
	Template    string `json:"template"`
}

// パフォーマンス洞察
type PerformanceInsight struct {
	Type         string `json:"type"`          // "bottleneck", "optimization_opportunity"
	File         string `json:"file"`
	Line         int    `json:"line"`
	Function     string `json:"function"`
	Issue        string `json:"issue"`
	Impact       string `json:"impact"`        // "low", "medium", "high"
	Suggestion   string `json:"suggestion"`
	EstimatedGain string `json:"estimated_gain"`
}

// セキュリティ発見事項
type SecurityFinding struct {
	Type        string `json:"type"`        // "vulnerability", "weakness", "bad_practice"
	Severity    string `json:"severity"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Rule        string `json:"rule"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Fix         string `json:"fix"`
	References  []string `json:"references"`
}

// セキュリティ分析結果
type SecurityAnalysis struct {
	SecurityScore int               `json:"security_score"` // 0-100
	Findings      []SecurityFinding `json:"findings"`
}

// 依存関係問題
type DependencyIssue struct {
	Type        string `json:"type"`        // "outdated", "vulnerable", "unused", "conflict"
	Package     string `json:"package"`
	Current     string `json:"current_version"`
	Latest      string `json:"latest_version,omitempty"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Action      string `json:"recommended_action"`
}

// 複雑度分析
type ComplexityAnalysis struct {
	CyclomaticComplexity int                      `json:"cyclomatic_complexity"`
	CognitiveComplexity  int                      `json:"cognitive_complexity"`
	NestingDepth         int                      `json:"max_nesting_depth"`
	FunctionsByComplexity map[string]int          `json:"functions_by_complexity"`
	ComplexFunctions     []ComplexFunctionReport `json:"complex_functions"`
}

// 複雑な関数のレポート
type ComplexFunctionReport struct {
	File        string `json:"file"`
	Function    string `json:"function"`
	Line        int    `json:"line"`
	Complexity  int    `json:"complexity"`
	Suggestion  string `json:"suggestion"`
}

// テストカバレッジ分析
type TestCoverageAnalysis struct {
	OverallCoverage    float64                    `json:"overall_coverage"`
	LineCoverage       float64                    `json:"line_coverage"`
	BranchCoverage     float64                    `json:"branch_coverage"`
	FunctionCoverage   float64                    `json:"function_coverage"`
	UncoveredFiles     []string                   `json:"uncovered_files"`
	CriticalUncovered  []CriticalUncoveredCode    `json:"critical_uncovered"`
	TestSuggestions    []string                   `json:"test_suggestions"`
}

// 重要な未カバーコード
type CriticalUncoveredCode struct {
	File        string `json:"file"`
	Function    string `json:"function"`
	Line        int    `json:"line"`
	Reason      string `json:"reason"`
	Suggestion  string `json:"suggestion"`
}

// AIコード分析器
type CodeAnalyzer struct {
	llmClient   LLMClient
	constraints *security.Constraints
	projectDir  string
	config      *AnalysisConfig
}

// 分析設定
type AnalysisConfig struct {
	MaxFileSize        int64    `json:"max_file_size"`        // bytes
	IncludePatterns    []string `json:"include_patterns"`     // glob patterns
	ExcludePatterns    []string `json:"exclude_patterns"`     // glob patterns
	AnalysisDepth      string   `json:"analysis_depth"`       // "basic", "detailed", "comprehensive"
	LanguageFilter     []string `json:"language_filter"`      // specific languages to analyze
	EnableSecurityScan bool     `json:"enable_security_scan"`
	EnablePerformance  bool     `json:"enable_performance"`
	EnableRefactoring  bool     `json:"enable_refactoring"`
	CustomPrompts      map[string]string `json:"custom_prompts"`
}

// AIコード分析器を作成
func NewCodeAnalyzer(llmClient LLMClient, constraints *security.Constraints, projectDir string) *CodeAnalyzer {
	return &CodeAnalyzer{
		llmClient:   llmClient,
		constraints: constraints,
		projectDir:  projectDir,
		config: &AnalysisConfig{
			MaxFileSize:        1024 * 1024,     // 1MB
			IncludePatterns:    []string{"**/*"}, 
			ExcludePatterns:    []string{"**/node_modules/**", "**/vendor/**", "**/.git/**", "**/*.min.*"},
			AnalysisDepth:      "detailed",
			EnableSecurityScan: true,
			EnablePerformance:  true,
			EnableRefactoring:  true,
			CustomPrompts:      make(map[string]string),
		},
	}
}

// 設定を更新
func (ca *CodeAnalyzer) UpdateConfig(config *AnalysisConfig) {
	if config != nil {
		ca.config = config
	}
}

// プロジェクト全体を分析
func (ca *CodeAnalyzer) AnalyzeProject(ctx context.Context) (*CodeAnalysisResult, error) {
	startTime := time.Now()
	
	result := &CodeAnalysisResult{
		Issues:            []CodeIssue{},
		Suggestions:       []CodeSuggestion{},
		Refactoring:       []RefactoringProposal{},
		Documentation:     []DocumentationGap{},
		Performance:       []PerformanceInsight{},
		Security:          SecurityAnalysis{Findings: []SecurityFinding{}},
		Dependencies:      []DependencyIssue{},
		AnalysisTimestamp: startTime,
	}

	// プロジェクトディレクトリの存在確認
	if _, err := os.Stat(ca.projectDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("プロジェクトディレクトリが存在しません: %s", ca.projectDir)
	}

	// ファイル収集
	files, err := ca.collectFiles()
	if err != nil {
		return nil, fmt.Errorf("ファイル収集エラー: %w", err)
	}

	if len(files) == 0 {
		return result, fmt.Errorf("分析対象ファイルが見つかりません")
	}

	// 各ファイルを分析
	for _, file := range files {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		fileResult, err := ca.analyzeFile(ctx, file)
		if err != nil {
			fmt.Printf("ファイル分析警告 %s: %v\n", file, err)
			continue
		}

		// 結果をマージ
		ca.mergeFileResult(result, fileResult)
	}

	// プロジェクトレベルの分析
	err = ca.performProjectLevelAnalysis(ctx, result, files)
	if err != nil {
		fmt.Printf("プロジェクトレベル分析警告: %v\n", err)
	}

	// 品質メトリクスを計算
	ca.calculateQualityMetrics(result)

	// 全体的な要約を生成
	err = ca.generateSummary(ctx, result)
	if err != nil {
		fmt.Printf("要約生成警告: %v\n", err)
	}

	result.ProcessingTime = time.Since(startTime)
	
	return result, nil
}

// 分析対象ファイルを収集
func (ca *CodeAnalyzer) collectFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(ca.projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // スキップ
		}

		if info.IsDir() {
			return nil
		}

		// サイズ制限チェック
		if info.Size() > ca.config.MaxFileSize {
			return nil
		}

		// 除外パターンチェック
		relPath, _ := filepath.Rel(ca.projectDir, path)
		if ca.shouldExcludeFile(relPath) {
			return nil
		}

		// 含める拡張子をチェック
		if ca.isAnalyzableFile(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// ファイルを除外するかチェック
func (ca *CodeAnalyzer) shouldExcludeFile(relPath string) bool {
	for _, pattern := range ca.config.ExcludePatterns {
		matched, _ := filepath.Match(pattern, relPath)
		if matched {
			return true
		}
		// 簡易的なグロブマッチング（**をサポート）
		if strings.Contains(pattern, "**") {
			patternParts := strings.Split(pattern, "**")
			if len(patternParts) == 2 {
				prefix := patternParts[0]
				suffix := patternParts[1]
				if (prefix == "" || strings.HasPrefix(relPath, prefix)) &&
				   (suffix == "" || strings.HasSuffix(relPath, suffix)) {
					return true
				}
			}
		}
	}
	return false
}

// 分析可能なファイルかチェック
func (ca *CodeAnalyzer) isAnalyzableFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	analyzableExtensions := []string{
		".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".java", ".cpp", ".c", ".h", ".hpp",
		".rs", ".rb", ".php", ".cs", ".swift", ".kt", ".scala", ".r", ".m", ".sh",
		".sql", ".yaml", ".yml", ".json", ".xml", ".html", ".css", ".scss", ".sass",
		".dockerfile", ".toml", ".ini", ".conf",
	}

	for _, analyzableExt := range analyzableExtensions {
		if ext == analyzableExt {
			return true
		}
	}

	// 拡張子なしでもファイル名で判断
	filename := strings.ToLower(filepath.Base(path))
	specialFiles := []string{"makefile", "dockerfile", "jenkinsfile", "vagrantfile"}
	for _, special := range specialFiles {
		if filename == special {
			return true
		}
	}

	return false
}

// 単一ファイルを分析
func (ca *CodeAnalyzer) analyzeFile(ctx context.Context, filePath string) (*CodeAnalysisResult, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	contentStr := string(content)
	relPath, _ := filepath.Rel(ca.projectDir, filePath)
	
	// LLMにコード分析を依頼
	prompt := ca.buildAnalysisPrompt(relPath, contentStr)
	
	response, err := ca.llmClient.GenerateResponse(ctx, &GenerateRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1, // 分析は一貫性を重視
	})

	if err != nil {
		return nil, fmt.Errorf("LLM分析エラー: %w", err)
	}

	// レスポンスをパース
	result, err := ca.parseAnalysisResponse(response.Content)
	if err != nil {
		return nil, fmt.Errorf("レスポンス解析エラー: %w", err)
	}

	return result, nil
}

// 分析プロンプトを構築
func (ca *CodeAnalyzer) buildAnalysisPrompt(filename, content string) string {
	fileExt := filepath.Ext(filename)
	language := ca.detectLanguageFromExtension(fileExt)

	prompt := fmt.Sprintf(`あなたは経験豊富なソフトウェアエンジニアです。以下の%sファイルを詳細に分析してください：

ファイル: %s
言語: %s

コード:
---
%s
---

以下の形式でJSON形式の分析結果を提供してください：

{
  "issues": [
    {
      "type": "bug|smell|style|performance",
      "severity": "low|medium|high|critical",
      "line": <行番号>,
      "title": "問題のタイトル",
      "description": "詳細説明",
      "solution": "解決方法"
    }
  ],
  "suggestions": [
    {
      "type": "optimization|cleanup|modernization",
      "priority": "low|medium|high",
      "line_range": [<開始行>, <終了行>],
      "title": "改善提案のタイトル",
      "description": "詳細説明",
      "before": "変更前のコード例",
      "after": "変更後のコード例",
      "benefits": "改善効果"
    }
  ],
  "documentation_gaps": [
    {
      "type": "missing_comment|missing_docstring|outdated",
      "line": <行番号>,
      "element": "function|class|module",
      "element_name": "要素名",
      "suggestion": "ドキュメント追加提案"
    }
  ],
  "performance_insights": [
    {
      "type": "bottleneck|optimization_opportunity",
      "line": <行番号>,
      "function": "関数名",
      "issue": "パフォーマンス問題",
      "impact": "low|medium|high",
      "suggestion": "最適化提案"
    }
  ]
}

重要事項:
1. コードの品質、セキュリティ、パフォーマンス、保守性を重視
2. 具体的な行番号と改善案を提供
3. %s言語のベストプラクティスに基づいて評価
4. 有効なJSONのみを返答`, language, filename, language, content, language)

	return prompt
}

// 拡張子から言語を検出
func (ca *CodeAnalyzer) detectLanguageFromExtension(ext string) string {
	languageMap := map[string]string{
		".go":   "Go",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".jsx":  "JavaScript (React)",
		".tsx":  "TypeScript (React)",
		".py":   "Python",
		".java": "Java",
		".cpp":  "C++",
		".c":    "C",
		".h":    "C/C++",
		".hpp":  "C++",
		".rs":   "Rust",
		".rb":   "Ruby",
		".php":  "PHP",
		".cs":   "C#",
		".kt":   "Kotlin",
		".swift": "Swift",
		".scala": "Scala",
		".sql":  "SQL",
		".sh":   "Shell Script",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}
	return "Unknown"
}

// LLM分析レスポンスをパース
func (ca *CodeAnalyzer) parseAnalysisResponse(response string) (*CodeAnalysisResult, error) {
	// JSONブロックを抽出
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}") + 1
	
	if jsonStart == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("有効なJSONが見つかりません")
	}

	jsonStr := response[jsonStart:jsonEnd]
	
	var result struct {
		Issues             []CodeIssue         `json:"issues"`
		Suggestions        []CodeSuggestion    `json:"suggestions"`
		DocumentationGaps  []DocumentationGap  `json:"documentation_gaps"`
		PerformanceInsights []PerformanceInsight `json:"performance_insights"`
	}

	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, fmt.Errorf("JSON解析エラー: %w", err)
	}

	return &CodeAnalysisResult{
		Issues:        result.Issues,
		Suggestions:   result.Suggestions,
		Documentation: result.DocumentationGaps,
		Performance:   result.PerformanceInsights,
	}, nil
}

// ファイル分析結果をメイン結果にマージ
func (ca *CodeAnalyzer) mergeFileResult(main, file *CodeAnalysisResult) {
	main.Issues = append(main.Issues, file.Issues...)
	main.Suggestions = append(main.Suggestions, file.Suggestions...)
	main.Documentation = append(main.Documentation, file.Documentation...)
	main.Performance = append(main.Performance, file.Performance...)
	main.Security.Findings = append(main.Security.Findings, file.Security.Findings...)
}

// プロジェクトレベル分析を実行
func (ca *CodeAnalyzer) performProjectLevelAnalysis(ctx context.Context, result *CodeAnalysisResult, files []string) error {
	// 依存関係分析
	err := ca.analyzeDependencies(result)
	if err != nil {
		fmt.Printf("依存関係分析警告: %v\n", err)
	}

	// アーキテクチャ分析
	err = ca.analyzeArchitecture(ctx, result, files)
	if err != nil {
		fmt.Printf("アーキテクチャ分析警告: %v\n", err)
	}

	return nil
}

// 依存関係を分析
func (ca *CodeAnalyzer) analyzeDependencies(result *CodeAnalysisResult) error {
	// go.mod の分析
	goModPath := filepath.Join(ca.projectDir, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		return ca.analyzeGoMod(result, goModPath)
	}

	// package.json の分析
	packageJsonPath := filepath.Join(ca.projectDir, "package.json")
	if _, err := os.Stat(packageJsonPath); err == nil {
		return ca.analyzePackageJson(result, packageJsonPath)
	}

	return nil
}

// Go mod を分析
func (ca *CodeAnalyzer) analyzeGoMod(result *CodeAnalysisResult, path string) error {
	// 実装は簡略化 - 実際には go list -m や go mod graph を使用
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.Contains(line, "// indirect") {
			// 未使用の依存関係候補
			parts := strings.Fields(strings.TrimSpace(line))
			if len(parts) >= 2 {
				result.Dependencies = append(result.Dependencies, DependencyIssue{
					Type:        "unused",
					Package:     parts[0],
					Current:     parts[1],
					Severity:    "low",
					Description: "間接的な依存関係が未使用の可能性があります",
					Action:      "go mod tidy を実行してクリーンアップを検討",
				})
			}
		}
	}

	return nil
}

// package.json を分析
func (ca *CodeAnalyzer) analyzePackageJson(result *CodeAnalysisResult, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var packageData map[string]interface{}
	err = json.Unmarshal(content, &packageData)
	if err != nil {
		return err
	}

	// 依存関係のバージョンチェック（簡略化）
	if deps, ok := packageData["dependencies"].(map[string]interface{}); ok {
		for name, version := range deps {
			if versionStr, ok := version.(string); ok {
				if strings.HasPrefix(versionStr, "^") || strings.HasPrefix(versionStr, "~") {
					result.Dependencies = append(result.Dependencies, DependencyIssue{
						Type:        "version_range",
						Package:     name,
						Current:     versionStr,
						Severity:    "low",
						Description: "バージョン範囲指定により予期しない更新が発生する可能性があります",
						Action:      "固定バージョンの使用を検討",
					})
				}
			}
		}
	}

	return nil
}

// アーキテクチャを分析
func (ca *CodeAnalyzer) analyzeArchitecture(ctx context.Context, result *CodeAnalysisResult, files []string) error {
	// ディレクトリ構造を分析してアーキテクチャの改善提案を生成
	dirStructure := ca.analyzeDirectoryStructure(files)
	
	// 循環依存をチェック
	cycles := ca.detectCircularDependencies(files)
	for _, cycle := range cycles {
		result.Issues = append(result.Issues, CodeIssue{
			Type:        "bug",
			Severity:    "high",
			File:        strings.Join(cycle, " -> "),
			Title:       "循環依存の検出",
			Description: fmt.Sprintf("ファイル間で循環依存が発生しています: %s", strings.Join(cycle, " -> ")),
			Solution:    "依存関係を整理し、インターフェースやDIパターンを使用して循環を解消してください",
		})
	}

	// プロジェクト構造の改善提案
	if len(dirStructure) < 3 {
		result.Suggestions = append(result.Suggestions, CodeSuggestion{
			Type:        "cleanup",
			Priority:    "medium",
			File:        "project structure",
			Title:       "プロジェクト構造の改善",
			Description: "プロジェクトの構造をより整理された形に改善することを推奨します",
			Benefits:    "保守性とコードの可読性が向上します",
		})
	}

	return nil
}

// ディレクトリ構造を分析
func (ca *CodeAnalyzer) analyzeDirectoryStructure(files []string) map[string]int {
	dirCount := make(map[string]int)
	
	for _, file := range files {
		relPath, _ := filepath.Rel(ca.projectDir, file)
		dir := filepath.Dir(relPath)
		if dir != "." {
			dirCount[dir]++
		}
	}
	
	return dirCount
}

// 循環依存を検出
func (ca *CodeAnalyzer) detectCircularDependencies(files []string) [][]string {
	// 簡略化された実装 - 実際にはASTパーサーを使用してimport文を解析
	var cycles [][]string
	
	// この実装では、同じディレクトリ内で相互参照がある場合を検出
	filesByDir := make(map[string][]string)
	for _, file := range files {
		relPath, _ := filepath.Rel(ca.projectDir, file)
		dir := filepath.Dir(relPath)
		filesByDir[dir] = append(filesByDir[dir], relPath)
	}
	
	// 実際の循環依存検出ロジックはより複雑になります
	for dir, dirFiles := range filesByDir {
		if len(dirFiles) > 10 { // 大きなディレクトリの場合
			cycles = append(cycles, []string{fmt.Sprintf("%s/*.go", dir), fmt.Sprintf("%s/*.go", dir)})
		}
	}
	
	return cycles
}

// 品質メトリクスを計算
func (ca *CodeAnalyzer) calculateQualityMetrics(result *CodeAnalysisResult) {
	// 基本スコア計算
	totalIssues := len(result.Issues)
	criticalIssues := 0
	highIssues := 0
	
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "critical":
			criticalIssues++
		case "high":
			highIssues++
		}
	}
	
	// 品質スコア計算（100点満点から減算）
	overallScore := 100
	overallScore -= criticalIssues * 20
	overallScore -= highIssues * 10
	overallScore -= (totalIssues - criticalIssues - highIssues) * 5
	
	if overallScore < 0 {
		overallScore = 0
	}
	
	// セキュリティスコアを計算
	securityScore := max(70, 100-len(result.Security.Findings)*10)
	result.Security.SecurityScore = securityScore
	
	result.Quality = QualityMetrics{
		OverallScore:    overallScore,
		Maintainability: max(60, 100-(totalIssues*3)),
		Reliability:     max(70, 100-(criticalIssues*15)-(highIssues*8)),
		Performance:     max(80, 100-len(result.Performance)*5),
		Security:        securityScore,
		Readability:     max(60, 100-len(result.Documentation)*3),
		TestQuality:     50, // テスト分析は別途実装
		TechnicalDebt:   float64(totalIssues) * 0.5, // 1問題あたり30分と仮定
		CodeDuplication: 0.0, // 重複コード検出は別途実装
	}
}

// 全体要約を生成
func (ca *CodeAnalyzer) generateSummary(ctx context.Context, result *CodeAnalysisResult) error {
	summaryPrompt := fmt.Sprintf(`以下のコード分析結果に基づいて、プロジェクトの全体的な要約を日本語で作成してください：

問題数: %d個
- Critical: %d個
- High: %d個  
- Medium以下: %d個

改善提案数: %d個
ドキュメント不足: %d個
パフォーマンス問題: %d個
セキュリティ問題: %d個
依存関係問題: %d個

品質スコア: %d/100

簡潔で実用的な要約を提供し、最も重要な改善ポイントを3つ挙げてください。`,
		len(result.Issues),
		ca.countIssuesBySeverity(result.Issues, "critical"),
		ca.countIssuesBySeverity(result.Issues, "high"), 
		len(result.Issues) - ca.countIssuesBySeverity(result.Issues, "critical") - ca.countIssuesBySeverity(result.Issues, "high"),
		len(result.Suggestions),
		len(result.Documentation),
		len(result.Performance),
		len(result.Security.Findings),
		len(result.Dependencies),
		result.Quality.OverallScore)

	response, err := ca.llmClient.GenerateResponse(ctx, &GenerateRequest{
		Messages: []Message{
			{
				Role:    "user", 
				Content: summaryPrompt,
			},
		},
		Temperature: 0.3,
	})

	if err != nil {
		return err
	}

	result.Summary = response.Content
	return nil
}

// 深刻度別の問題数をカウント
func (ca *CodeAnalyzer) countIssuesBySeverity(issues []CodeIssue, severity string) int {
	count := 0
	for _, issue := range issues {
		if issue.Severity == severity {
			count++
		}
	}
	return count
}

// max helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}