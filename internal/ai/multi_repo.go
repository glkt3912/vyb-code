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

// マルチリポジトリ管理システム
type MultiRepoManager struct {
	constraints  *security.Constraints
	workspaceDir string
	repositories map[string]*Repository
	config       *MultiRepoConfig
}

// リポジトリ情報
type Repository struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Path         string                 `json:"path"`
	RemoteURL    string                 `json:"remote_url,omitempty"`
	Branch       string                 `json:"branch"`
	Language     string                 `json:"primary_language"`
	Type         string                 `json:"type"`   // "service", "library", "tool", "config"
	Status       string                 `json:"status"` // "active", "archived", "deprecated"
	LastAnalysis time.Time              `json:"last_analysis"`
	Dependencies []RepoDependency       `json:"dependencies"`
	Metadata     map[string]interface{} `json:"metadata"`
	Analysis     *CodeAnalysisResult    `json:"analysis,omitempty"`
}

// リポジトリ間依存関係
type RepoDependency struct {
	TargetRepo     string  `json:"target_repo"`
	DependencyType string  `json:"dependency_type"` // "api_call", "library", "config", "data"
	Strength       string  `json:"strength"`        // "weak", "medium", "strong"
	Description    string  `json:"description"`
	Weight         float64 `json:"weight"`
}

// マルチリポジトリ設定
type MultiRepoConfig struct {
	WorkspaceDir    string   `json:"workspace_dir"`
	AutoDiscovery   bool     `json:"auto_discovery"`
	IncludePatterns []string `json:"include_patterns"`
	ExcludePatterns []string `json:"exclude_patterns"`
	MaxRepositories int      `json:"max_repositories"`
	AnalysisDepth   string   `json:"analysis_depth"` // "basic", "detailed", "comprehensive"
	SyncInterval    string   `json:"sync_interval"`  // "hourly", "daily", "weekly"
	EnableCaching   bool     `json:"enable_caching"`
	CacheExpiration string   `json:"cache_expiration"`
}

// ワークスペース分析結果
type WorkspaceAnalysis struct {
	Overview           WorkspaceOverview         `json:"overview"`
	Dependencies       []CrossRepoDependency     `json:"cross_repo_dependencies"`
	Architecture       WorkspaceArchitecture     `json:"architecture"`
	QualityMetrics     WorkspaceQuality          `json:"quality_metrics"`
	SecurityAssessment WorkspaceSecurity         `json:"security_assessment"`
	Recommendations    []WorkspaceRecommendation `json:"recommendations"`
	AnalyzedAt         time.Time                 `json:"analyzed_at"`
	ProcessingTime     time.Duration             `json:"processing_time"`
}

// ワークスペース概要
type WorkspaceOverview struct {
	TotalRepositories    int                 `json:"total_repositories"`
	ActiveRepositories   int                 `json:"active_repositories"`
	LanguageDistribution map[string]int      `json:"language_distribution"`
	TypeDistribution     map[string]int      `json:"type_distribution"`
	TotalLinesOfCode     int64               `json:"total_lines_of_code"`
	TotalFiles           int                 `json:"total_files"`
	LastUpdated          time.Time           `json:"last_updated"`
	RepositorySummary    []RepositorySummary `json:"repository_summary"`
}

// リポジトリサマリー
type RepositorySummary struct {
	Name         string    `json:"name"`
	Language     string    `json:"language"`
	Type         string    `json:"type"`
	LinesOfCode  int64     `json:"lines_of_code"`
	Files        int       `json:"files"`
	LastCommit   time.Time `json:"last_commit"`
	HealthScore  int       `json:"health_score"`
	IssueCount   int       `json:"issue_count"`
	TestCoverage float64   `json:"test_coverage"`
}

// クロスリポジトリ依存関係
type CrossRepoDependency struct {
	SourceRepo     string                `json:"source_repo"`
	TargetRepo     string                `json:"target_repo"`
	DependencyType string                `json:"dependency_type"`
	Interfaces     []InterfaceDefinition `json:"interfaces"`
	APIEndpoints   []APIEndpoint         `json:"api_endpoints"`
	DataFlows      []DataFlow            `json:"data_flows"`
	Strength       float64               `json:"strength"`
	RiskLevel      string                `json:"risk_level"`
	LastVerified   time.Time             `json:"last_verified"`
}

// インターフェース定義
type InterfaceDefinition struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"` // "API", "Library", "Protocol"
	Version       string            `json:"version"`
	Methods       []MethodSignature `json:"methods"`
	Schema        string            `json:"schema,omitempty"`
	Documentation string            `json:"documentation"`
}

// メソッドシグネチャ
type MethodSignature struct {
	Name        string      `json:"name"`
	Parameters  []Parameter `json:"parameters"`
	Returns     []Parameter `json:"returns"`
	Description string      `json:"description"`
}

// パラメータ
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// データフロー
type DataFlow struct {
	Source      string           `json:"source"`
	Target      string           `json:"target"`
	DataType    string           `json:"data_type"`
	Format      string           `json:"format"`      // "JSON", "XML", "Binary", etc.
	Frequency   string           `json:"frequency"`   // "real-time", "batch", "on-demand"
	Volume      string           `json:"volume"`      // "low", "medium", "high"
	Sensitivity string           `json:"sensitivity"` // "public", "internal", "confidential"
	Validation  []ValidationRule `json:"validation"`
}

// バリデーションルール
type ValidationRule struct {
	Field       string `json:"field"`
	Rule        string `json:"rule"`
	Description string `json:"description"`
}

// ワークスペースアーキテクチャ
type WorkspaceArchitecture struct {
	Patterns        []ArchitecturalPattern  `json:"patterns"`
	Layers          []ArchitecturalLayer    `json:"layers"`
	ServiceMap      []ServiceDefinition     `json:"service_map"`
	DeploymentUnits []DeploymentUnit        `json:"deployment_units"`
	Integration     IntegrationArchitecture `json:"integration"`
}

// サービス定義
type ServiceDefinition struct {
	Name         string   `json:"name"`
	Repository   string   `json:"repository"`
	Type         string   `json:"type"` // "microservice", "monolith", "library"
	Endpoints    []string `json:"endpoints"`
	Dependencies []string `json:"dependencies"`
	Consumers    []string `json:"consumers"`
	Database     string   `json:"database,omitempty"`
	Queue        string   `json:"queue,omitempty"`
	Cache        string   `json:"cache,omitempty"`
}

// デプロイメントユニット
type DeploymentUnit struct {
	Name          string            `json:"name"`
	Repositories  []string          `json:"repositories"`
	Type          string            `json:"type"`        // "container", "lambda", "vm"
	Environment   string            `json:"environment"` // "development", "staging", "production"
	Dependencies  []string          `json:"dependencies"`
	ScalingPolicy string            `json:"scaling_policy"`
	Resources     map[string]string `json:"resources"`
}

// 統合アーキテクチャ
type IntegrationArchitecture struct {
	Patterns       []string              `json:"patterns"`  // "API Gateway", "Event Bus", "Message Queue"
	Protocols      []string              `json:"protocols"` // "HTTP", "gRPC", "AMQP", "WebSocket"
	MessageBrokers []MessageBrokerConfig `json:"message_brokers"`
	APIGateways    []APIGatewayConfig    `json:"api_gateways"`
	Databases      []DatabaseConfig      `json:"databases"`
}

// メッセージブローカー設定
type MessageBrokerConfig struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"` // "Kafka", "RabbitMQ", "Redis"
	Topics  []string `json:"topics"`
	Used_by []string `json:"used_by"`
}

// APIゲートウェイ設定
type APIGatewayConfig struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"` // "Kong", "Nginx", "AWS API Gateway"
	Routes   []string `json:"routes"`
	Services []string `json:"services"`
}

// データベース設定
type DatabaseConfig struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"` // "PostgreSQL", "MongoDB", "Redis"
	Used_by []string `json:"used_by"`
	Schema  string   `json:"schema,omitempty"`
}

// ワークスペース品質
type WorkspaceQuality struct {
	OverallScore         int                  `json:"overall_score"`
	ConsistencyScore     int                  `json:"consistency_score"`
	IntegrationScore     int                  `json:"integration_score"`
	MaintainabilityScore int                  `json:"maintainability_score"`
	TestCoverageAverage  float64              `json:"test_coverage_average"`
	CodeDuplication      float64              `json:"code_duplication"`
	TechnicalDebt        float64              `json:"technical_debt"` // in hours
	QualityTrends        map[string][]float64 `json:"quality_trends"` // time series data
	ProblemRepositories  []string             `json:"problem_repositories"`
}

// ワークスペースセキュリティ
type WorkspaceSecurity struct {
	OverallRisk             string                   `json:"overall_risk"` // "low", "medium", "high", "critical"
	VulnerabilityCount      int                      `json:"vulnerability_count"`
	SecurityScore           int                      `json:"security_score"`
	CrossRepoRisks          []CrossRepoSecurityRisk  `json:"cross_repo_risks"`
	ComplianceStatus        map[string]string        `json:"compliance_status"` // e.g., "GDPR": "compliant"
	SecurityRecommendations []SecurityRecommendation `json:"security_recommendations"`
}

// クロスリポジトリセキュリティリスク
type CrossRepoSecurityRisk struct {
	Type         string   `json:"type"` // "data_exposure", "privilege_escalation", "secret_sharing"
	Severity     string   `json:"severity"`
	Repositories []string `json:"repositories"`
	Description  string   `json:"description"`
	Mitigation   string   `json:"mitigation"`
}

// ワークスペース推奨事項
type WorkspaceRecommendation struct {
	Category    string   `json:"category"` // "architecture", "security", "quality", "process"
	Priority    string   `json:"priority"` // "low", "medium", "high", "critical"
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Affected    []string `json:"affected_repositories"`
	ActionItems []string `json:"action_items"`
	Benefits    string   `json:"benefits"`
	Effort      string   `json:"effort"`   // "small", "medium", "large"
	Timeline    string   `json:"timeline"` // suggested completion time
}

// マルチリポジトリマネージャーを作成
func NewMultiRepoManager(constraints *security.Constraints, workspaceDir string) *MultiRepoManager {
	return &MultiRepoManager{
		constraints:  constraints,
		workspaceDir: workspaceDir,
		repositories: make(map[string]*Repository),
		config: &MultiRepoConfig{
			WorkspaceDir:    workspaceDir,
			AutoDiscovery:   true,
			IncludePatterns: []string{"**/.git"},
			ExcludePatterns: []string{"**/node_modules/**", "**/vendor/**", "**/.*"},
			MaxRepositories: 50,
			AnalysisDepth:   "detailed",
			SyncInterval:    "daily",
			EnableCaching:   true,
			CacheExpiration: "24h",
		},
	}
}

// 設定を更新
func (mrm *MultiRepoManager) UpdateConfig(config *MultiRepoConfig) {
	if config != nil {
		mrm.config = config
	}
}

// リポジトリを自動発見
func (mrm *MultiRepoManager) DiscoverRepositories(ctx context.Context) error {
	if !mrm.config.AutoDiscovery {
		return nil
	}

	// ワークスペースディレクトリを走査
	err := filepath.Walk(mrm.workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // スキップ
		}

		// .gitディレクトリを見つけた場合
		if info.IsDir() && info.Name() == ".git" {
			repoPath := filepath.Dir(path)
			return mrm.registerRepository(repoPath)
		}

		return nil
	})

	return err
}

// リポジトリを登録
func (mrm *MultiRepoManager) registerRepository(repoPath string) error {
	// 除外パターンをチェック
	relPath, _ := filepath.Rel(mrm.workspaceDir, repoPath)
	if mrm.shouldExcludeRepository(relPath) {
		return nil
	}

	// 既に登録済みかチェック
	repoID := mrm.generateRepoID(repoPath)
	if _, exists := mrm.repositories[repoID]; exists {
		return nil
	}

	// リポジトリ情報を分析
	repo := &Repository{
		ID:           repoID,
		Name:         filepath.Base(repoPath),
		Path:         repoPath,
		Status:       "active",
		LastAnalysis: time.Time{},
		Dependencies: []RepoDependency{},
		Metadata:     make(map[string]interface{}),
	}

	// 基本情報を収集
	err := mrm.analyzeRepository(repo)
	if err != nil {
		fmt.Printf("リポジトリ分析警告 %s: %v\n", repoPath, err)
		return nil
	}

	mrm.repositories[repoID] = repo
	return nil
}

// リポジトリを除外するかチェック
func (mrm *MultiRepoManager) shouldExcludeRepository(relPath string) bool {
	for _, pattern := range mrm.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
		// 簡易的なグロブマッチング
		if strings.Contains(pattern, "**") {
			parts := strings.Split(pattern, "**")
			if len(parts) == 2 {
				prefix := parts[0]
				suffix := parts[1]
				if (prefix == "" || strings.HasPrefix(relPath, prefix)) &&
					(suffix == "" || strings.HasSuffix(relPath, suffix)) {
					return true
				}
			}
		}
	}
	return false
}

// リポジトリIDを生成
func (mrm *MultiRepoManager) generateRepoID(repoPath string) string {
	relPath, _ := filepath.Rel(mrm.workspaceDir, repoPath)
	return strings.ReplaceAll(strings.ReplaceAll(relPath, "/", "_"), "\\", "_")
}

// リポジトリを分析
func (mrm *MultiRepoManager) analyzeRepository(repo *Repository) error {
	// Git情報を取得
	err := mrm.extractGitInfo(repo)
	if err != nil {
		fmt.Printf("Git情報取得警告 %s: %v\n", repo.Path, err)
	}

	// プログラミング言語を検出
	repo.Language = mrm.detectPrimaryLanguage(repo.Path)

	// リポジトリタイプを推測
	repo.Type = mrm.inferRepositoryType(repo.Path)

	// 基本統計を収集
	err = mrm.collectRepositoryStats(repo)
	if err != nil {
		fmt.Printf("統計収集警告 %s: %v\n", repo.Path, err)
	}

	return nil
}

// Git情報を抽出
func (mrm *MultiRepoManager) extractGitInfo(repo *Repository) error {
	// 現在のブランチを取得（簡略化）
	gitHeadPath := filepath.Join(repo.Path, ".git", "HEAD")
	if content, err := os.ReadFile(gitHeadPath); err == nil {
		head := strings.TrimSpace(string(content))
		if strings.HasPrefix(head, "ref: refs/heads/") {
			repo.Branch = strings.TrimPrefix(head, "ref: refs/heads/")
		}
	}

	// リモートURLを取得（簡略化）
	gitConfigPath := filepath.Join(repo.Path, ".git", "config")
	if content, err := os.ReadFile(gitConfigPath); err == nil {
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if strings.Contains(line, `[remote "origin"]`) && i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(nextLine, "url = ") {
					repo.RemoteURL = strings.TrimPrefix(nextLine, "url = ")
					break
				}
			}
		}
	}

	return nil
}

// 主要プログラミング言語を検出
func (mrm *MultiRepoManager) detectPrimaryLanguage(repoPath string) string {
	languageCounts := make(map[string]int)

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// .gitディレクトリをスキップ
		if strings.Contains(path, "/.git/") || strings.Contains(path, "\\.git\\") {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if lang := mrm.extensionToLanguage(ext); lang != "" {
			languageCounts[lang]++
		}

		return nil
	})

	if err != nil {
		return "Unknown"
	}

	// 最も多いファイル数の言語を返す
	maxCount := 0
	primaryLanguage := "Unknown"
	for lang, count := range languageCounts {
		if count > maxCount {
			maxCount = count
			primaryLanguage = lang
		}
	}

	return primaryLanguage
}

// 拡張子から言語をマッピング
func (mrm *MultiRepoManager) extensionToLanguage(ext string) string {
	mapping := map[string]string{
		".go":    "Go",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".py":    "Python",
		".java":  "Java",
		".cpp":   "C++",
		".c":     "C",
		".rs":    "Rust",
		".rb":    "Ruby",
		".php":   "PHP",
		".cs":    "C#",
		".kt":    "Kotlin",
		".swift": "Swift",
	}

	return mapping[ext]
}

// リポジトリタイプを推測
func (mrm *MultiRepoManager) inferRepositoryType(repoPath string) string {
	// 特定のファイルの存在でタイプを推測
	files := map[string]string{
		"main.go":            "service",
		"cmd/main.go":        "service",
		"server.go":          "service",
		"package.json":       "service",
		"Dockerfile":         "service",
		"go.mod":             "library",
		"setup.py":           "library",
		"pom.xml":            "library",
		"Makefile":           "tool",
		"docker-compose.yml": "config",
		"terraform":          "config",
		"ansible":            "config",
	}

	for filename, repoType := range files {
		if _, err := os.Stat(filepath.Join(repoPath, filename)); err == nil {
			return repoType
		}
	}

	return "unknown"
}

// リポジトリ統計を収集
func (mrm *MultiRepoManager) collectRepositoryStats(repo *Repository) error {
	var totalFiles int
	var totalLines int64

	err := filepath.Walk(repo.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// .gitディレクトリをスキップ
		if strings.Contains(path, "/.git/") || strings.Contains(path, "\\.git\\") {
			return nil
		}

		// ソースファイルのみカウント
		if mrm.isSourceFile(path) {
			totalFiles++

			// 行数をカウント
			if content, err := os.ReadFile(path); err == nil {
				lines := strings.Split(string(content), "\n")
				totalLines += int64(len(lines))
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// メタデータに統計を保存
	repo.Metadata["total_files"] = totalFiles
	repo.Metadata["total_lines"] = totalLines

	return nil
}

// ソースファイルかチェック
func (mrm *MultiRepoManager) isSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	sourceExtensions := []string{
		".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".java",
		".cpp", ".c", ".h", ".hpp", ".rs", ".rb", ".php", ".cs",
	}

	for _, sourceExt := range sourceExtensions {
		if ext == sourceExt {
			return true
		}
	}
	return false
}

// ワークスペースを分析
func (mrm *MultiRepoManager) AnalyzeWorkspace(ctx context.Context) (*WorkspaceAnalysis, error) {
	startTime := time.Now()

	// リポジトリを発見
	err := mrm.DiscoverRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("リポジトリ発見エラー: %w", err)
	}

	analysis := &WorkspaceAnalysis{
		AnalyzedAt: startTime,
	}

	// 概要を生成
	analysis.Overview = mrm.generateOverview()

	// クロスリポジトリ依存関係を分析
	analysis.Dependencies, err = mrm.analyzeCrossRepoDependencies(ctx)
	if err != nil {
		fmt.Printf("クロスリポジトリ依存関係分析警告: %v\n", err)
	}

	// アーキテクチャを分析
	analysis.Architecture = mrm.analyzeWorkspaceArchitecture()

	// 品質メトリクスを計算
	analysis.QualityMetrics = mrm.calculateWorkspaceQuality()

	// セキュリティ評価
	analysis.SecurityAssessment = mrm.assessWorkspaceSecurity()

	// 推奨事項を生成
	analysis.Recommendations = mrm.generateWorkspaceRecommendations()

	analysis.ProcessingTime = time.Since(startTime)

	return analysis, nil
}

// 概要を生成
func (mrm *MultiRepoManager) generateOverview() WorkspaceOverview {
	overview := WorkspaceOverview{
		LanguageDistribution: make(map[string]int),
		TypeDistribution:     make(map[string]int),
		RepositorySummary:    []RepositorySummary{},
		LastUpdated:          time.Now(),
	}

	var totalLines int64
	var totalFiles int
	activeRepos := 0

	for _, repo := range mrm.repositories {
		overview.TotalRepositories++

		if repo.Status == "active" {
			activeRepos++
		}

		// 言語分布
		overview.LanguageDistribution[repo.Language]++

		// タイプ分布
		overview.TypeDistribution[repo.Type]++

		// 統計集計
		if files, ok := repo.Metadata["total_files"].(int); ok {
			totalFiles += files
		}
		if lines, ok := repo.Metadata["total_lines"].(int64); ok {
			totalLines += lines
		}

		// リポジトリサマリー
		summary := RepositorySummary{
			Name:        repo.Name,
			Language:    repo.Language,
			Type:        repo.Type,
			HealthScore: 80, // デフォルト値
		}

		if files, ok := repo.Metadata["total_files"].(int); ok {
			summary.Files = files
		}
		if lines, ok := repo.Metadata["total_lines"].(int64); ok {
			summary.LinesOfCode = lines
		}

		overview.RepositorySummary = append(overview.RepositorySummary, summary)
	}

	overview.ActiveRepositories = activeRepos
	overview.TotalFiles = totalFiles
	overview.TotalLinesOfCode = totalLines

	return overview
}

// クロスリポジトリ依存関係を分析
func (mrm *MultiRepoManager) analyzeCrossRepoDependencies(ctx context.Context) ([]CrossRepoDependency, error) {
	var dependencies []CrossRepoDependency

	// 各リポジトリペアを分析
	for sourceID, sourceRepo := range mrm.repositories {
		for targetID, targetRepo := range mrm.repositories {
			if sourceID == targetID {
				continue
			}

			// 依存関係を検出
			deps := mrm.detectRepoDependencies(sourceRepo, targetRepo)
			for _, dep := range deps {
				crossDep := CrossRepoDependency{
					SourceRepo:     sourceRepo.Name,
					TargetRepo:     targetRepo.Name,
					DependencyType: dep.DependencyType,
					Strength:       dep.Weight,
					RiskLevel:      mrm.calculateRiskLevel(dep.Weight),
					LastVerified:   time.Now(),
				}

				dependencies = append(dependencies, crossDep)
			}
		}
	}

	return dependencies, nil
}

// リポジトリ間依存関係を検出
func (mrm *MultiRepoManager) detectRepoDependencies(source, target *Repository) []RepoDependency {
	var dependencies []RepoDependency

	// 名前参照を検索
	if mrm.hasNameReference(source.Path, target.Name) {
		dependencies = append(dependencies, RepoDependency{
			TargetRepo:     target.Name,
			DependencyType: "name_reference",
			Strength:       "weak",
			Weight:         0.3,
			Description:    fmt.Sprintf("%s references %s by name", source.Name, target.Name),
		})
	}

	// API呼び出しを検索
	if mrm.hasAPIReference(source.Path, target.Name) {
		dependencies = append(dependencies, RepoDependency{
			TargetRepo:     target.Name,
			DependencyType: "api_call",
			Strength:       "strong",
			Weight:         0.8,
			Description:    fmt.Sprintf("%s makes API calls to %s", source.Name, target.Name),
		})
	}

	// ライブラリ依存を検索
	if mrm.hasLibraryDependency(source.Path, target.Name) {
		dependencies = append(dependencies, RepoDependency{
			TargetRepo:     target.Name,
			DependencyType: "library",
			Strength:       "strong",
			Weight:         1.0,
			Description:    fmt.Sprintf("%s depends on %s as library", source.Name, target.Name),
		})
	}

	return dependencies
}

// 名前参照をチェック
func (mrm *MultiRepoManager) hasNameReference(sourcePath, targetName string) bool {
	// 簡略化された実装
	found := false

	filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !mrm.isSourceFile(path) {
			return nil
		}

		if content, err := os.ReadFile(path); err == nil {
			if strings.Contains(string(content), targetName) {
				found = true
			}
		}

		return nil
	})

	return found
}

// API参照をチェック
func (mrm *MultiRepoManager) hasAPIReference(sourcePath, targetName string) bool {
	// HTTP呼び出しのパターンを検索
	apiPatterns := []string{
		"http://",
		"https://",
		"fetch(",
		"axios.",
		"http.Get",
		"http.Post",
	}

	found := false

	filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !mrm.isSourceFile(path) {
			return nil
		}

		if content, err := os.ReadFile(path); err == nil {
			contentStr := string(content)
			for _, pattern := range apiPatterns {
				if strings.Contains(contentStr, pattern) && strings.Contains(contentStr, targetName) {
					found = true
					break
				}
			}
		}

		return nil
	})

	return found
}

// ライブラリ依存をチェック
func (mrm *MultiRepoManager) hasLibraryDependency(sourcePath, targetName string) bool {
	// go.mod, package.json, requirements.txt などをチェック
	depFiles := []string{"go.mod", "package.json", "requirements.txt", "pom.xml"}

	for _, depFile := range depFiles {
		depPath := filepath.Join(sourcePath, depFile)
		if content, err := os.ReadFile(depPath); err == nil {
			if strings.Contains(string(content), targetName) {
				return true
			}
		}
	}

	return false
}

// リスクレベルを計算
func (mrm *MultiRepoManager) calculateRiskLevel(weight float64) string {
	if weight >= 0.8 {
		return "high"
	} else if weight >= 0.5 {
		return "medium"
	} else {
		return "low"
	}
}

// ワークスペースアーキテクチャを分析
func (mrm *MultiRepoManager) analyzeWorkspaceArchitecture() WorkspaceArchitecture {
	architecture := WorkspaceArchitecture{
		Patterns:        []ArchitecturalPattern{},
		Layers:          []ArchitecturalLayer{},
		ServiceMap:      []ServiceDefinition{},
		DeploymentUnits: []DeploymentUnit{},
		Integration: IntegrationArchitecture{
			Patterns:       []string{},
			Protocols:      []string{},
			MessageBrokers: []MessageBrokerConfig{},
			APIGateways:    []APIGatewayConfig{},
			Databases:      []DatabaseConfig{},
		},
	}

	// アーキテクチャパターンを検出
	architecture.Patterns = mrm.detectArchitecturalPatterns()

	// サービスマップを生成
	architecture.ServiceMap = mrm.generateServiceMap()

	// 統合パターンを分析
	architecture.Integration = mrm.analyzeIntegrationPatterns()

	return architecture
}

// アーキテクチャパターンを検出
func (mrm *MultiRepoManager) detectArchitecturalPatterns() []ArchitecturalPattern {
	var patterns []ArchitecturalPattern

	// マイクロサービス検出
	serviceCount := 0
	for _, repo := range mrm.repositories {
		if repo.Type == "service" {
			serviceCount++
		}
	}

	if serviceCount > 3 {
		patterns = append(patterns, ArchitecturalPattern{
			Name:        "Microservices",
			Confidence:  0.8,
			Description: fmt.Sprintf("Detected %d services suggesting microservices architecture", serviceCount),
			Evidence:    []string{fmt.Sprintf("%d service repositories", serviceCount)},
			Benefits:    "Scalability, independent deployment, technology diversity",
			Drawbacks:   "Increased complexity, distributed system challenges",
		})
	}

	// モノリス検出
	for _, repo := range mrm.repositories {
		if files, ok := repo.Metadata["total_files"].(int); ok && files > 100 {
			if repo.Type == "service" {
				patterns = append(patterns, ArchitecturalPattern{
					Name:        "Monolith",
					Confidence:  0.6,
					Description: "Large service repository suggesting monolithic architecture",
					Evidence:    []string{fmt.Sprintf("Large repository with %d files", files)},
					Benefits:    "Simplicity, easier testing, single deployment",
					Drawbacks:   "Scaling challenges, technology lock-in",
				})
				break
			}
		}
	}

	return patterns
}

// サービスマップを生成
func (mrm *MultiRepoManager) generateServiceMap() []ServiceDefinition {
	var services []ServiceDefinition

	for _, repo := range mrm.repositories {
		if repo.Type == "service" {
			service := ServiceDefinition{
				Name:         repo.Name,
				Repository:   repo.Name,
				Type:         "microservice",
				Endpoints:    mrm.extractEndpoints(repo.Path),
				Dependencies: mrm.extractServiceDependencies(repo),
				Consumers:    []string{}, // 実装時に計算
			}

			// データベース検出
			if mrm.hasDatabase(repo.Path) {
				service.Database = "detected"
			}

			// キュー検出
			if mrm.hasMessageQueue(repo.Path) {
				service.Queue = "detected"
			}

			services = append(services, service)
		}
	}

	return services
}

// エンドポイントを抽出
func (mrm *MultiRepoManager) extractEndpoints(repoPath string) []string {
	var endpoints []string

	// HTTPハンドラーパターンを検索
	patterns := []string{
		"@RequestMapping",
		"@GetMapping",
		"@PostMapping",
		"http.HandleFunc",
		"router.",
		"app.get(",
		"app.post(",
	}

	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !mrm.isSourceFile(path) {
			return nil
		}

		if content, err := os.ReadFile(path); err == nil {
			contentStr := string(content)
			for _, pattern := range patterns {
				if strings.Contains(contentStr, pattern) {
					// 簡略化：パターンが見つかった場合は存在するものとする
					endpoints = append(endpoints, "detected")
					break
				}
			}
		}

		return nil
	})

	if len(endpoints) == 0 {
		endpoints = []string{"none"}
	}

	return endpoints
}

// サービス依存関係を抽出
func (mrm *MultiRepoManager) extractServiceDependencies(repo *Repository) []string {
	var dependencies []string

	for _, dep := range repo.Dependencies {
		dependencies = append(dependencies, dep.TargetRepo)
	}

	return dependencies
}

// データベースをチェック
func (mrm *MultiRepoManager) hasDatabase(repoPath string) bool {
	dbPatterns := []string{
		"database/sql",
		"gorm.io",
		"mongoose",
		"sequelize",
		"sqlalchemy",
		"jdbc:",
	}

	return mrm.hasAnyPattern(repoPath, dbPatterns)
}

// メッセージキューをチェック
func (mrm *MultiRepoManager) hasMessageQueue(repoPath string) bool {
	queuePatterns := []string{
		"kafka",
		"rabbitmq",
		"redis",
		"amqp",
		"pubsub",
	}

	return mrm.hasAnyPattern(repoPath, queuePatterns)
}

// パターンの存在をチェック
func (mrm *MultiRepoManager) hasAnyPattern(repoPath string, patterns []string) bool {
	found := false

	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !mrm.isSourceFile(path) {
			return nil
		}

		if content, err := os.ReadFile(path); err == nil {
			contentStr := strings.ToLower(string(content))
			for _, pattern := range patterns {
				if strings.Contains(contentStr, strings.ToLower(pattern)) {
					found = true
					return filepath.SkipDir // 見つかったら終了
				}
			}
		}

		return nil
	})

	return found
}

// 統合パターンを分析
func (mrm *MultiRepoManager) analyzeIntegrationPatterns() IntegrationArchitecture {
	integration := IntegrationArchitecture{
		Patterns:       []string{},
		Protocols:      []string{},
		MessageBrokers: []MessageBrokerConfig{},
		APIGateways:    []APIGatewayConfig{},
		Databases:      []DatabaseConfig{},
	}

	// プロトコルを検出
	if mrm.hasWorkspacePattern([]string{"http", "https"}) {
		integration.Protocols = append(integration.Protocols, "HTTP")
	}
	if mrm.hasWorkspacePattern([]string{"grpc"}) {
		integration.Protocols = append(integration.Protocols, "gRPC")
	}
	if mrm.hasWorkspacePattern([]string{"websocket"}) {
		integration.Protocols = append(integration.Protocols, "WebSocket")
	}

	// パターンを検出
	if len(mrm.repositories) > 5 {
		integration.Patterns = append(integration.Patterns, "Microservices")
	}
	if mrm.hasWorkspacePattern([]string{"gateway", "proxy"}) {
		integration.Patterns = append(integration.Patterns, "API Gateway")
	}

	return integration
}

// ワークスペース全体でパターンをチェック
func (mrm *MultiRepoManager) hasWorkspacePattern(patterns []string) bool {
	for _, repo := range mrm.repositories {
		if mrm.hasAnyPattern(repo.Path, patterns) {
			return true
		}
	}
	return false
}

// ワークスペース品質を計算
func (mrm *MultiRepoManager) calculateWorkspaceQuality() WorkspaceQuality {
	quality := WorkspaceQuality{
		QualityTrends:       make(map[string][]float64),
		ProblemRepositories: []string{},
	}

	var totalScore int
	var repoCount int
	var totalCoverage float64
	var coverageCount int

	for _, repo := range mrm.repositories {
		if repo.Status == "active" {
			repoCount++

			// 仮の健康スコア（実際の実装では詳細分析）
			healthScore := 80
			if files, ok := repo.Metadata["total_files"].(int); ok && files > 200 {
				healthScore -= 10 // 大きすぎるリポジトリはペナルティ
			}

			totalScore += healthScore

			if healthScore < 60 {
				quality.ProblemRepositories = append(quality.ProblemRepositories, repo.Name)
			}

			// テストカバレッジ（仮の値）
			coverage := 75.0
			totalCoverage += coverage
			coverageCount++
		}
	}

	if repoCount > 0 {
		quality.OverallScore = totalScore / repoCount
		quality.ConsistencyScore = mrm.calculateConsistencyScore()
		quality.IntegrationScore = mrm.calculateIntegrationScore()
		quality.MaintainabilityScore = quality.OverallScore // 簡略化
	}

	if coverageCount > 0 {
		quality.TestCoverageAverage = totalCoverage / float64(coverageCount)
	}

	quality.CodeDuplication = 5.0 // 仮の値
	quality.TechnicalDebt = 120.0 // 仮の値（時間）

	return quality
}

// 一貫性スコアを計算
func (mrm *MultiRepoManager) calculateConsistencyScore() int {
	// 言語、構造、パターンの一貫性を評価
	languageCount := len(mrm.getUniqueLanguages())

	// 言語が少ないほど一貫性が高い
	consistencyScore := 100
	if languageCount > 3 {
		consistencyScore -= (languageCount - 3) * 10
	}

	if consistencyScore < 0 {
		consistencyScore = 0
	}

	return consistencyScore
}

// 統合スコアを計算
func (mrm *MultiRepoManager) calculateIntegrationScore() int {
	// 依存関係の健全性を評価
	totalDependencies := 0
	circularDependencies := 0

	for _, repo := range mrm.repositories {
		totalDependencies += len(repo.Dependencies)
		// 循環依存の検出（簡略化）
	}

	integrationScore := 80
	if circularDependencies > 0 {
		integrationScore -= circularDependencies * 20
	}

	return integrationScore
}

// ユニークな言語を取得
func (mrm *MultiRepoManager) getUniqueLanguages() []string {
	languages := make(map[string]bool)

	for _, repo := range mrm.repositories {
		if repo.Language != "Unknown" && repo.Language != "" {
			languages[repo.Language] = true
		}
	}

	var uniqueLanguages []string
	for lang := range languages {
		uniqueLanguages = append(uniqueLanguages, lang)
	}

	return uniqueLanguages
}

// ワークスペースセキュリティを評価
func (mrm *MultiRepoManager) assessWorkspaceSecurity() WorkspaceSecurity {
	security := WorkspaceSecurity{
		CrossRepoRisks:          []CrossRepoSecurityRisk{},
		ComplianceStatus:        make(map[string]string),
		SecurityRecommendations: []SecurityRecommendation{},
	}

	var totalVulnerabilities int
	var totalSecurityScore int
	var repoCount int

	for _, repo := range mrm.repositories {
		if repo.Status == "active" {
			repoCount++

			// 仮のセキュリティ評価
			vulnerabilities := mrm.countVulnerabilities(repo.Path)
			totalVulnerabilities += vulnerabilities

			securityScore := 90 - (vulnerabilities * 10)
			if securityScore < 0 {
				securityScore = 0
			}
			totalSecurityScore += securityScore
		}
	}

	security.VulnerabilityCount = totalVulnerabilities

	if repoCount > 0 {
		security.SecurityScore = totalSecurityScore / repoCount
	}

	// リスクレベルを決定
	if security.VulnerabilityCount == 0 {
		security.OverallRisk = "low"
	} else if security.VulnerabilityCount < 5 {
		security.OverallRisk = "medium"
	} else {
		security.OverallRisk = "high"
	}

	// コンプライアンス状態（簡略化）
	security.ComplianceStatus["GDPR"] = "unknown"
	security.ComplianceStatus["SOC2"] = "unknown"

	return security
}

// 脆弱性数をカウント（簡略化）
func (mrm *MultiRepoManager) countVulnerabilities(repoPath string) int {
	vulnerabilityPatterns := []string{
		"password",
		"secret",
		"api_key",
		"token",
		"eval(",
		"exec(",
	}

	count := 0

	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !mrm.isSourceFile(path) {
			return nil
		}

		if content, err := os.ReadFile(path); err == nil {
			contentStr := strings.ToLower(string(content))
			for _, pattern := range vulnerabilityPatterns {
				count += strings.Count(contentStr, pattern)
			}
		}

		return nil
	})

	return count
}

// ワークスペース推奨事項を生成
func (mrm *MultiRepoManager) generateWorkspaceRecommendations() []WorkspaceRecommendation {
	var recommendations []WorkspaceRecommendation

	// アーキテクチャ推奨事項
	if len(mrm.repositories) > 10 {
		recommendations = append(recommendations, WorkspaceRecommendation{
			Category:    "architecture",
			Priority:    "medium",
			Title:       "リポジトリ統合の検討",
			Description: "多数の小さなリポジトリがありますが、関連するものの統合を検討してください",
			Affected:    []string{}, // 具体的なリポジトリリスト
			ActionItems: []string{"関連リポジトリの特定", "統合計画の策定", "段階的な統合実行"},
			Benefits:    "管理の簡素化、依存関係の明確化",
			Effort:      "large",
			Timeline:    "3-6 months",
		})
	}

	// セキュリティ推奨事項
	recommendations = append(recommendations, WorkspaceRecommendation{
		Category:    "security",
		Priority:    "high",
		Title:       "セキュリティスキャンの導入",
		Description: "全リポジトリに対する定期的なセキュリティスキャンを導入してください",
		ActionItems: []string{"スキャンツールの選定", "CI/CDパイプラインへの統合", "アラート体制の構築"},
		Benefits:    "早期脆弱性発見、セキュリティ向上",
		Effort:      "medium",
		Timeline:    "1-2 months",
	})

	// 品質推奨事項
	if len(mrm.getUniqueLanguages()) > 5 {
		recommendations = append(recommendations, WorkspaceRecommendation{
			Category:    "quality",
			Priority:    "medium",
			Title:       "言語標準化の検討",
			Description: "多くの異なる言語が使用されています。標準化を検討してください",
			ActionItems: []string{"主要言語の選定", "移行計画の策定", "チーム教育"},
			Benefits:    "保守性向上、チーム効率性向上",
			Effort:      "large",
			Timeline:    "6-12 months",
		})
	}

	return recommendations
}

// JSON出力
func (mrm *MultiRepoManager) ExportToJSON(analysis *WorkspaceAnalysis) ([]byte, error) {
	return json.MarshalIndent(analysis, "", "  ")
}

// リポジトリ情報を取得
func (mrm *MultiRepoManager) GetRepository(repoID string) (*Repository, bool) {
	repo, exists := mrm.repositories[repoID]
	return repo, exists
}

// 全リポジトリを取得
func (mrm *MultiRepoManager) GetAllRepositories() map[string]*Repository {
	return mrm.repositories
}
