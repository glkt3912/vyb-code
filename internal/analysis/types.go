package analysis

import (
	"time"
)

// 分析タイプ
type AnalysisType int

const (
	AnalysisTypeBasic AnalysisType = iota
	AnalysisTypeSecurity
	AnalysisTypeQuality
	AnalysisTypeFull
)

// プロジェクト分析結果の構造体
type ProjectAnalysis struct {
	ProjectPath      string                 `json:"project_path"`
	ProjectName      string                 `json:"project_name"`
	Language         string                 `json:"language"`
	Framework        string                 `json:"framework"`
	Dependencies     []Dependency           `json:"dependencies"`
	FileStructure    *FileStructure         `json:"file_structure"`
	QualityMetrics   *QualityMetrics        `json:"quality_metrics"`
	TechStack        []Technology           `json:"tech_stack"`
	BuildSystem      *BuildSystem           `json:"build_system"`
	TestingFramework *TestingFramework      `json:"testing_framework"`
	GitInfo          *GitInfo               `json:"git_info"`
	SecurityIssues   []SecurityIssue        `json:"security_issues"`
	Recommendations  []Recommendation       `json:"recommendations"`
	AnalyzedAt       time.Time              `json:"analyzed_at"`
	AnalysisVersion  string                 `json:"analysis_version"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// 依存関係情報
type Dependency struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Type            string   `json:"type"`   // "direct", "dev", "peer", "optional"
	Source          string   `json:"source"` // "npm", "go.mod", "requirements.txt", etc.
	Outdated        bool     `json:"outdated"`
	Vulnerabilities []string `json:"vulnerabilities"`
}

// ファイル構造情報
type FileStructure struct {
	TotalFiles      int                   `json:"total_files"`
	TotalLines      int                   `json:"total_lines"`
	Languages       map[string]int        `json:"languages"` // 言語別ファイル数
	Directories     []DirectoryInfo       `json:"directories"`
	LargestFiles    []FileInfo            `json:"largest_files"`
	RecentlyChanged []FileInfo            `json:"recently_changed"`
	Patterns        []ArchitecturePattern `json:"patterns"`
}

// ディレクトリ情報
type DirectoryInfo struct {
	Path       string  `json:"path"`
	FileCount  int     `json:"file_count"`
	LineCount  int     `json:"line_count"`
	Purpose    string  `json:"purpose"`    // "source", "test", "config", "docs", "assets"
	Importance float64 `json:"importance"` // 0.0-1.0
}

// ファイル情報
type FileInfo struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	Lines        int       `json:"lines"`
	Language     string    `json:"language"`
	LastModified time.Time `json:"last_modified"`
	Complexity   int       `json:"complexity"`
	Purpose      string    `json:"purpose"`
	Dependencies []string  `json:"dependencies"`
}

// アーキテクチャパターン
type ArchitecturePattern struct {
	Name        string   `json:"name"`
	Confidence  float64  `json:"confidence"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
}

// 品質メトリクス
type QualityMetrics struct {
	TestCoverage     float64            `json:"test_coverage"`
	CodeComplexity   float64            `json:"code_complexity"`
	Maintainability  float64            `json:"maintainability"`
	Duplication      float64            `json:"duplication"`
	TechnicalDebt    time.Duration      `json:"technical_debt"`
	IssueCount       int                `json:"issue_count"`
	LintWarnings     int                `json:"lint_warnings"`
	SecurityScore    float64            `json:"security_score"`
	PerformanceScore float64            `json:"performance_score"`
	Details          map[string]float64 `json:"details"`
}

// 技術スタック
type Technology struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Type         string   `json:"type"` // "language", "framework", "library", "tool"
	Confidence   float64  `json:"confidence"`
	Usage        string   `json:"usage"` // "primary", "secondary", "testing", "development"
	DetectedFrom []string `json:"detected_from"`
}

// ビルドシステム
type BuildSystem struct {
	Name          string            `json:"name"`
	ConfigFile    string            `json:"config_file"`
	BuildCommands []string          `json:"build_commands"`
	TestCommands  []string          `json:"test_commands"`
	LintCommands  []string          `json:"lint_commands"`
	Scripts       map[string]string `json:"scripts"`
	Targets       []string          `json:"targets"`
}

// テストフレームワーク
type TestingFramework struct {
	Name            string   `json:"name"`
	ConfigFiles     []string `json:"config_files"`
	TestDirectories []string `json:"test_directories"`
	TestFiles       []string `json:"test_files"`
	TestCommands    []string `json:"test_commands"`
	Coverage        bool     `json:"coverage"`
}

// Git情報
type GitInfo struct {
	Repository     string    `json:"repository"`
	CurrentBranch  string    `json:"current_branch"`
	LastCommit     string    `json:"last_commit"`
	CommitCount    int       `json:"commit_count"`
	Contributors   []string  `json:"contributors"`
	LastActivity   time.Time `json:"last_activity"`
	RemoteURL      string    `json:"remote_url"`
	HasChanges     bool      `json:"has_changes"`
	ActiveBranches []string  `json:"active_branches"`
}

// セキュリティ問題
type SecurityIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"` // "critical", "high", "medium", "low"
	Description string `json:"description"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Suggestion  string `json:"suggestion"`
	CWE         string `json:"cwe,omitempty"`
}

// 推奨事項
type Recommendation struct {
	Type        string   `json:"type"`     // "performance", "security", "maintainability", "best_practice"
	Priority    string   `json:"priority"` // "critical", "high", "medium", "low"
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Action      string   `json:"action"`
	Files       []string `json:"files"`
	Effort      string   `json:"effort"` // "low", "medium", "high"
	Impact      string   `json:"impact"` // "low", "medium", "high"
}

// プロジェクト分析インターフェース
type ProjectAnalyzer interface {
	// プロジェクト全体の分析
	AnalyzeProject(projectPath string) (*ProjectAnalysis, error)

	// 部分的な分析
	AnalyzeFile(filePath string) (*FileInfo, error)
	AnalyzeDirectory(dirPath string) (*DirectoryInfo, error)

	// 依存関係分析
	AnalyzeDependencies(projectPath string) ([]Dependency, error)

	// 品質メトリクス分析
	AnalyzeQuality(projectPath string) (*QualityMetrics, error)

	// セキュリティ分析
	AnalyzeSecurity(projectPath string) ([]SecurityIssue, error)

	// 推奨事項生成
	GenerateRecommendations(analysis *ProjectAnalysis) ([]Recommendation, error)

	// キャッシュ管理
	GetCachedAnalysis(projectPath string) (*ProjectAnalysis, error)
	CacheAnalysis(projectPath string, analysis *ProjectAnalysis) error
	InvalidateCache(projectPath string) error
}

// 分析設定
type AnalysisConfig struct {
	EnableCaching       bool                   `json:"enable_caching"`
	CacheExpiry         time.Duration          `json:"cache_expiry"`
	AnalysisDepth       string                 `json:"analysis_depth"` // "quick", "standard", "deep"
	IncludeTests        bool                   `json:"include_tests"`
	IncludeDependencies bool                   `json:"include_dependencies"`
	SecurityScan        bool                   `json:"security_scan"`
	QualityMetrics      bool                   `json:"quality_metrics"`
	ExcludePatterns     []string               `json:"exclude_patterns"`
	LanguageConfig      map[string]interface{} `json:"language_config"`
	MaxFileSize         int64                  `json:"max_file_size"`
	Timeout             time.Duration          `json:"timeout"`
}

// デフォルト設定を取得
func DefaultAnalysisConfig() *AnalysisConfig {
	return &AnalysisConfig{
		EnableCaching:       true,
		CacheExpiry:         30 * time.Minute,
		AnalysisDepth:       "standard",
		IncludeTests:        true,
		IncludeDependencies: true,
		SecurityScan:        true,
		QualityMetrics:      true,
		ExcludePatterns: []string{
			"node_modules/**",
			"vendor/**",
			".git/**",
			"dist/**",
			"build/**",
			"target/**",
			"*.log",
			"*.tmp",
			"*.cache",
		},
		MaxFileSize: 1024 * 1024, // 1MB
		Timeout:     5 * time.Minute,
	}
}

// 分析結果のサマリー
type AnalysisSummary struct {
	ProjectName      string    `json:"project_name"`
	Language         string    `json:"language"`
	FileCount        int       `json:"file_count"`
	LineCount        int       `json:"line_count"`
	QualityScore     float64   `json:"quality_score"`
	SecurityScore    float64   `json:"security_score"`
	TechDebt         string    `json:"tech_debt"`
	LastAnalyzed     time.Time `json:"last_analyzed"`
	CriticalIssues   int       `json:"critical_issues"`
	Recommendations  int       `json:"recommendations"`
	MainTechnologies []string  `json:"main_technologies"`
}

// サマリー生成
func (pa *ProjectAnalysis) GenerateSummary() *AnalysisSummary {
	summary := &AnalysisSummary{
		ProjectName:     pa.ProjectName,
		Language:        pa.Language,
		LastAnalyzed:    pa.AnalyzedAt,
		CriticalIssues:  0,
		Recommendations: len(pa.Recommendations),
	}

	if pa.FileStructure != nil {
		summary.FileCount = pa.FileStructure.TotalFiles
		summary.LineCount = pa.FileStructure.TotalLines
	}

	if pa.QualityMetrics != nil {
		summary.QualityScore = pa.QualityMetrics.Maintainability
		summary.SecurityScore = pa.QualityMetrics.SecurityScore
		summary.TechDebt = pa.QualityMetrics.TechnicalDebt.String()
	}

	// クリティカル問題をカウント
	for _, issue := range pa.SecurityIssues {
		if issue.Severity == "critical" {
			summary.CriticalIssues++
		}
	}

	// 主要技術を抽出
	for _, tech := range pa.TechStack {
		if tech.Usage == "primary" {
			summary.MainTechnologies = append(summary.MainTechnologies, tech.Name)
		}
	}

	return summary
}
