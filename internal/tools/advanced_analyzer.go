package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

// 高度なプロジェクト解析結果を格納する構造体
type AdvancedProjectAnalysis struct {
	BasicInfo         *ProjectAnalysis      `json:"basic_info"`
	BuildSystems      []BuildSystemInfo     `json:"build_systems"`
	Architecture      *ProjectArchitecture  `json:"architecture"`
	DetailedDeps      *DetailedDependencies `json:"detailed_dependencies"`
	HealthScore       *ProjectHealthScore   `json:"health_score"`
	SecurityAnalysis  *SecurityAnalysis     `json:"security_analysis"`
	AnalysisTimestamp time.Time             `json:"analysis_timestamp"`
	AnalysisVersion   string                `json:"analysis_version"`
}

// ビルドシステム情報
type BuildSystemInfo struct {
	Type         string            `json:"type"`         // "makefile", "docker", "ci", "language_native"
	ConfigFile   string            `json:"config_file"`  // 設定ファイルパス
	Commands     []string          `json:"commands"`     // 利用可能なコマンド
	Dependencies []string          `json:"dependencies"` // ビルド依存関係
	Targets      []string          `json:"targets"`      // ビルドターゲット
	Metadata     map[string]string `json:"metadata"`     // 追加情報
}

// プロジェクトアーキテクチャ
type ProjectArchitecture struct {
	Layers          []ArchitectureLayer `json:"layers"`           // アーキテクチャ層
	Modules         []ModuleInfo        `json:"modules"`          // モジュール情報
	Dependencies    []ModuleDependency  `json:"dependencies"`     // モジュール間依存関係
	EntryPoints     []string            `json:"entry_points"`     // エントリポイント
	DatabaseSchemas []string            `json:"database_schemas"` // データベーススキーマ
	APIEndpoints    []APIEndpoint       `json:"api_endpoints"`    // API エンドポイント
}

// アーキテクチャ層
type ArchitectureLayer struct {
	Name        string   `json:"name"`        // レイヤー名
	Description string   `json:"description"` // 説明
	Directories []string `json:"directories"` // 対応ディレクトリ
	Files       []string `json:"files"`       // 重要ファイル
}

// モジュール情報
type ModuleInfo struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Type        string   `json:"type"` // "package", "service", "library", etc.
	Language    string   `json:"language"`
	Files       []string `json:"files"`
	ExportedAPI []string `json:"exported_api"`
}

// モジュール間依存関係
type ModuleDependency struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Type   string `json:"type"`   // "import", "api_call", "database", etc.
	Weight int    `json:"weight"` // 依存度の重み
}

// API エンドポイント
type APIEndpoint struct {
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	Handler     string   `json:"handler"`
	Middlewares []string `json:"middlewares"`
}

// 詳細依存関係分析
type DetailedDependencies struct {
	DirectDeps      []DependencyInfo    `json:"direct_dependencies"`
	TransitiveDeps  []DependencyInfo    `json:"transitive_dependencies"`
	DevDeps         []DependencyInfo    `json:"dev_dependencies"`
	Conflicts       []DependencyInfo    `json:"conflicts"`
	Outdated        []DependencyInfo    `json:"outdated"`
	Vulnerabilities []VulnerabilityInfo `json:"vulnerabilities"`
}

// 依存関係情報
type DependencyInfo struct {
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	LatestVersion string    `json:"latest_version,omitempty"`
	Language      string    `json:"language"`
	Size          int64     `json:"size,omitempty"`
	License       string    `json:"license,omitempty"`
	LastUpdate    time.Time `json:"last_update,omitempty"`
	Description   string    `json:"description,omitempty"`
}

// 脆弱性情報
type VulnerabilityInfo struct {
	DependencyName string `json:"dependency_name"`
	Severity       string `json:"severity"` // "low", "medium", "high", "critical"
	CVE            string `json:"cve,omitempty"`
	Description    string `json:"description"`
	FixedVersion   string `json:"fixed_version,omitempty"`
}

// プロジェクト健康度スコア
type ProjectHealthScore struct {
	OverallScore    int                    `json:"overall_score"`   // 0-100
	CodeQuality     int                    `json:"code_quality"`    // 0-100
	TestCoverage    int                    `json:"test_coverage"`   // 0-100
	Documentation   int                    `json:"documentation"`   // 0-100
	Dependencies    int                    `json:"dependencies"`    // 0-100
	Security        int                    `json:"security"`        // 0-100
	Maintainability int                    `json:"maintainability"` // 0-100
	Recommendations []string               `json:"recommendations"` // 改善提案
	Metrics         map[string]interface{} `json:"metrics"`         // 詳細メトリクス
}

// セキュリティ分析
type SecurityAnalysis struct {
	VulnerabilityCount int                      `json:"vulnerability_count"`
	SecurityScore      int                      `json:"security_score"` // 0-100
	SecretLeaks        []SecretLeak             `json:"secret_leaks"`
	InsecurePatterns   []InsecurePattern        `json:"insecure_patterns"`
	Recommendations    []SecurityRecommendation `json:"recommendations"`
}

// 機密情報漏洩
type SecretLeak struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Type    string `json:"type"` // "api_key", "password", "token", etc.
	Pattern string `json:"pattern"`
}

// 不安全なパターン
type InsecurePattern struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// セキュリティ推奨事項
type SecurityRecommendation struct {
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

// 高度プロジェクト解析ツール
type AdvancedProjectAnalyzer struct {
	basicAnalyzer   *ProjectAnalyzer
	languageManager *LanguageManager
	constraints     *security.Constraints
	projectDir      string
}

// 高度プロジェクト解析ツールを作成
func NewAdvancedProjectAnalyzer(constraints *security.Constraints, projectDir string) *AdvancedProjectAnalyzer {
	return &AdvancedProjectAnalyzer{
		basicAnalyzer:   NewProjectAnalyzer(constraints, projectDir),
		languageManager: NewLanguageManager(),
		constraints:     constraints,
		projectDir:      projectDir,
	}
}

// 高度なプロジェクト解析を実行
func (a *AdvancedProjectAnalyzer) AnalyzeAdvanced() (*AdvancedProjectAnalysis, error) {
	analysis := &AdvancedProjectAnalysis{
		AnalysisTimestamp: time.Now(),
		AnalysisVersion:   "2.0.0",
	}

	// 基本分析を実行
	basicAnalysis, err := a.basicAnalyzer.AnalyzeProject()
	if err != nil {
		return nil, fmt.Errorf("基本解析エラー: %w", err)
	}
	analysis.BasicInfo = basicAnalysis

	// ビルドシステム解析
	buildSystems, err := a.analyzeBuildSystems()
	if err != nil {
		fmt.Printf("ビルドシステム解析警告: %v\n", err)
	} else {
		analysis.BuildSystems = buildSystems
	}

	// アーキテクチャ解析
	architecture, err := a.analyzeArchitecture()
	if err != nil {
		fmt.Printf("アーキテクチャ解析警告: %v\n", err)
	} else {
		analysis.Architecture = architecture
	}

	// 詳細依存関係解析
	detailedDeps, err := a.analyzeDetailedDependencies()
	if err != nil {
		fmt.Printf("詳細依存関係解析警告: %v\n", err)
	} else {
		analysis.DetailedDeps = detailedDeps
	}

	// セキュリティ解析
	securityAnalysis, err := a.analyzeProjectSecurity()
	if err != nil {
		fmt.Printf("セキュリティ解析警告: %v\n", err)
	} else {
		analysis.SecurityAnalysis = securityAnalysis
	}

	// 健康度スコア計算
	healthScore := a.calculateHealthScore(analysis)
	analysis.HealthScore = healthScore

	return analysis, nil
}

// ビルドシステムを解析
func (a *AdvancedProjectAnalyzer) analyzeBuildSystems() ([]BuildSystemInfo, error) {
	var buildSystems []BuildSystemInfo

	// Makefile検出
	makefilePath := filepath.Join(a.projectDir, "Makefile")
	if _, err := os.Stat(makefilePath); err == nil {
		makefile, err := a.analyzeMakefile(makefilePath)
		if err == nil {
			buildSystems = append(buildSystems, makefile)
		}
	}

	// Docker検出
	dockerfilePath := filepath.Join(a.projectDir, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err == nil {
		dockerfile, err := a.analyzeDockerfile(dockerfilePath)
		if err == nil {
			buildSystems = append(buildSystems, dockerfile)
		}
	}

	// CI設定検出
	ciSystems, err := a.analyzeCISystems()
	if err == nil {
		buildSystems = append(buildSystems, ciSystems...)
	}

	// 言語固有ビルドシステム検出
	langBuilds, err := a.analyzeLanguageBuilds()
	if err == nil {
		buildSystems = append(buildSystems, langBuilds...)
	}

	return buildSystems, nil
}

// Makefileを解析
func (a *AdvancedProjectAnalyzer) analyzeMakefile(path string) (BuildSystemInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return BuildSystemInfo{}, err
	}

	buildInfo := BuildSystemInfo{
		Type:       "makefile",
		ConfigFile: path,
		Commands:   []string{},
		Targets:    []string{},
		Metadata:   make(map[string]string),
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// ターゲット検出（行の開始がタブでなく、コロンを含む）
		if !strings.HasPrefix(line, "\t") && strings.Contains(line, ":") && !strings.HasPrefix(line, "#") {
			target := strings.Split(line, ":")[0]
			target = strings.TrimSpace(target)
			if target != "" && !strings.Contains(target, "=") {
				buildInfo.Targets = append(buildInfo.Targets, target)
			}
		}
	}

	// 一般的なコマンドを推定
	buildInfo.Commands = []string{"make", "make build", "make test", "make clean"}

	return buildInfo, nil
}

// Dockerfileを解析
func (a *AdvancedProjectAnalyzer) analyzeDockerfile(path string) (BuildSystemInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return BuildSystemInfo{}, err
	}

	buildInfo := BuildSystemInfo{
		Type:       "docker",
		ConfigFile: path,
		Commands:   []string{"docker build", "docker run"},
		Metadata:   make(map[string]string),
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "FROM ") {
			baseImage := strings.TrimPrefix(line, "FROM ")
			buildInfo.Metadata["base_image"] = baseImage
		}

		if strings.HasPrefix(line, "EXPOSE ") {
			port := strings.TrimPrefix(line, "EXPOSE ")
			buildInfo.Metadata["exposed_port"] = port
		}
	}

	return buildInfo, nil
}

// CIシステムを解析
func (a *AdvancedProjectAnalyzer) analyzeCISystems() ([]BuildSystemInfo, error) {
	var ciSystems []BuildSystemInfo

	// GitHub Actions
	githubDir := filepath.Join(a.projectDir, ".github", "workflows")
	if _, err := os.Stat(githubDir); err == nil {
		files, err := os.ReadDir(githubDir)
		if err == nil {
			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".yml") || strings.HasSuffix(file.Name(), ".yaml") {
					ciSystems = append(ciSystems, BuildSystemInfo{
						Type:       "github_actions",
						ConfigFile: filepath.Join(githubDir, file.Name()),
						Commands:   []string{"gh workflow run"},
						Metadata:   map[string]string{"platform": "github"},
					})
				}
			}
		}
	}

	// GitLab CI
	gitlabCIPath := filepath.Join(a.projectDir, ".gitlab-ci.yml")
	if _, err := os.Stat(gitlabCIPath); err == nil {
		ciSystems = append(ciSystems, BuildSystemInfo{
			Type:       "gitlab_ci",
			ConfigFile: gitlabCIPath,
			Commands:   []string{"gitlab-runner exec"},
			Metadata:   map[string]string{"platform": "gitlab"},
		})
	}

	return ciSystems, nil
}

// 言語固有ビルドシステムを解析
func (a *AdvancedProjectAnalyzer) analyzeLanguageBuilds() ([]BuildSystemInfo, error) {
	var langBuilds []BuildSystemInfo

	languages, err := a.languageManager.DetectProjectLanguages(a.projectDir)
	if err != nil {
		return langBuilds, err
	}

	for _, lang := range languages {
		buildInfo := BuildSystemInfo{
			Type:     fmt.Sprintf("%s_native", strings.ToLower(lang.GetName())),
			Commands: []string{},
			Metadata: make(map[string]string),
		}

		// ビルドコマンドを追加
		if buildCmd := lang.GetBuildCommand(); buildCmd != "" {
			buildInfo.Commands = append(buildInfo.Commands, buildCmd)
		}
		if testCmd := lang.GetTestCommand(); testCmd != "" {
			buildInfo.Commands = append(buildInfo.Commands, testCmd)
		}
		if lintCmd := lang.GetLintCommand(); lintCmd != "" {
			buildInfo.Commands = append(buildInfo.Commands, lintCmd)
		}

		// 依存関係ファイルを確認
		if depFile := lang.GetDependencyFile(); depFile != "" {
			depPath := filepath.Join(a.projectDir, depFile)
			if _, err := os.Stat(depPath); err == nil {
				buildInfo.ConfigFile = depPath
			}
		}

		buildInfo.Metadata["language"] = lang.GetName()
		langBuilds = append(langBuilds, buildInfo)
	}

	return langBuilds, nil
}

// プロジェクトアーキテクチャを解析
func (a *AdvancedProjectAnalyzer) analyzeArchitecture() (*ProjectArchitecture, error) {
	arch := &ProjectArchitecture{
		Layers:       []ArchitectureLayer{},
		Modules:      []ModuleInfo{},
		Dependencies: []ModuleDependency{},
		EntryPoints:  []string{},
	}

	// 一般的なアーキテクチャパターンを検出
	err := a.detectArchitecturalLayers(arch)
	if err != nil {
		return nil, err
	}

	// モジュール検出
	err = a.detectModules(arch)
	if err != nil {
		return nil, err
	}

	// エントリポイント検出
	err = a.detectEntryPoints(arch)
	if err != nil {
		return nil, err
	}

	return arch, nil
}

// アーキテクチャ層を検出
func (a *AdvancedProjectAnalyzer) detectArchitecturalLayers(arch *ProjectArchitecture) error {
	// 一般的な層構造パターン
	commonLayers := map[string][]string{
		"Frontend/UI":   {"ui", "frontend", "web", "public", "static", "assets"},
		"API/Handler":   {"api", "handler", "controller", "route", "endpoint"},
		"Service/Logic": {"service", "business", "logic", "core", "domain"},
		"Data/Storage":  {"data", "storage", "repository", "dao", "model", "entity"},
		"Config":        {"config", "configuration", "env", "settings"},
		"Tools/Utils":   {"util", "utils", "tool", "tools", "helper", "common"},
		"Test":          {"test", "tests", "spec", "specs", "__tests__"},
		"Documentation": {"doc", "docs", "documentation", "readme"},
	}

	// プロジェクト内のディレクトリを走査
	entries, err := os.ReadDir(a.projectDir)
	if err != nil {
		return err
	}

	for layerName, keywords := range commonLayers {
		layer := ArchitectureLayer{
			Name:        layerName,
			Description: fmt.Sprintf("%s related components", layerName),
			Directories: []string{},
			Files:       []string{},
		}

		for _, entry := range entries {
			if entry.IsDir() {
				dirName := strings.ToLower(entry.Name())
				for _, keyword := range keywords {
					if strings.Contains(dirName, keyword) {
						layer.Directories = append(layer.Directories, entry.Name())
						break
					}
				}
			}
		}

		if len(layer.Directories) > 0 {
			arch.Layers = append(arch.Layers, layer)
		}
	}

	return nil
}

// モジュールを検出
func (a *AdvancedProjectAnalyzer) detectModules(arch *ProjectArchitecture) error {
	// Go モジュール検出
	return filepath.Walk(a.projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}

		// internal や cmd などの重要ディレクトリ
		dirName := filepath.Base(path)
		if dirName == "internal" || dirName == "cmd" || dirName == "pkg" {
			subDirs, err := os.ReadDir(path)
			if err != nil {
				return nil
			}

			for _, subDir := range subDirs {
				if subDir.IsDir() {
					module := ModuleInfo{
						Name:     subDir.Name(),
						Path:     filepath.Join(path, subDir.Name()),
						Type:     "package",
						Language: "Go",
					}
					arch.Modules = append(arch.Modules, module)
				}
			}
		}

		return nil
	})
}

// エントリポイントを検出
func (a *AdvancedProjectAnalyzer) detectEntryPoints(arch *ProjectArchitecture) error {
	commonEntryPoints := []string{
		"main.go", "cmd/*/main.go",
		"index.js", "app.js", "server.js",
		"index.html", "main.html",
		"main.py", "__init__.py",
		"Main.java",
	}

	for _, pattern := range commonEntryPoints {
		matches, err := filepath.Glob(filepath.Join(a.projectDir, pattern))
		if err != nil {
			continue
		}

		for _, match := range matches {
			relPath, _ := filepath.Rel(a.projectDir, match)
			arch.EntryPoints = append(arch.EntryPoints, relPath)
		}
	}

	return nil
}

// 詳細依存関係を解析
func (a *AdvancedProjectAnalyzer) analyzeDetailedDependencies() (*DetailedDependencies, error) {
	deps := &DetailedDependencies{
		DirectDeps:      []DependencyInfo{},
		TransitiveDeps:  []DependencyInfo{},
		DevDeps:         []DependencyInfo{},
		Vulnerabilities: []VulnerabilityInfo{},
	}

	// Go mod 依存関係の詳細解析
	err := a.analyzeGoModDetailed(deps)
	if err != nil {
		fmt.Printf("Go mod詳細解析警告: %v\n", err)
	}

	// npm/package.json 依存関係の詳細解析
	err = a.analyzeNpmDetailed(deps)
	if err != nil {
		fmt.Printf("npm詳細解析警告: %v\n", err)
	}

	return deps, nil
}

// Go mod の詳細解析
func (a *AdvancedProjectAnalyzer) analyzeGoModDetailed(deps *DetailedDependencies) error {
	goModPath := filepath.Join(a.projectDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return err // go.mod が存在しない場合
	}

	lines := strings.Split(string(content), "\n")
	inRequireBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "require ") {
			if strings.Contains(line, "(") {
				inRequireBlock = true
				continue
			} else {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					depInfo := DependencyInfo{
						Name:     parts[1],
						Version:  parts[2],
						Language: "Go",
					}
					deps.DirectDeps = append(deps.DirectDeps, depInfo)
				}
			}
		} else if inRequireBlock {
			if line == ")" {
				inRequireBlock = false
			} else if line != "" && !strings.HasPrefix(line, "//") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					depInfo := DependencyInfo{
						Name:     parts[0],
						Version:  parts[1],
						Language: "Go",
					}
					deps.DirectDeps = append(deps.DirectDeps, depInfo)
				}
			}
		}
	}

	return nil
}

// npm の詳細解析
func (a *AdvancedProjectAnalyzer) analyzeNpmDetailed(deps *DetailedDependencies) error {
	packageJsonPath := filepath.Join(a.projectDir, "package.json")
	content, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return err // package.json が存在しない場合
	}

	// 簡易的なJSON解析（実際の実装では proper JSON parser を使用）
	var packageData map[string]interface{}
	if err := json.Unmarshal(content, &packageData); err != nil {
		return err
	}

	// dependencies 解析
	if dependencies, ok := packageData["dependencies"].(map[string]interface{}); ok {
		for name, version := range dependencies {
			if versionStr, ok := version.(string); ok {
				depInfo := DependencyInfo{
					Name:     name,
					Version:  versionStr,
					Language: "JavaScript",
				}
				deps.DirectDeps = append(deps.DirectDeps, depInfo)
			}
		}
	}

	// devDependencies 解析
	if devDependencies, ok := packageData["devDependencies"].(map[string]interface{}); ok {
		for name, version := range devDependencies {
			if versionStr, ok := version.(string); ok {
				depInfo := DependencyInfo{
					Name:     name,
					Version:  versionStr,
					Language: "JavaScript",
				}
				deps.DevDeps = append(deps.DevDeps, depInfo)
			}
		}
	}

	return nil
}

// プロジェクトセキュリティを解析
func (a *AdvancedProjectAnalyzer) analyzeProjectSecurity() (*SecurityAnalysis, error) {
	security := &SecurityAnalysis{
		VulnerabilityCount: 0,
		SecurityScore:      100,
		SecretLeaks:        []SecretLeak{},
		InsecurePatterns:   []InsecurePattern{},
		Recommendations:    []SecurityRecommendation{},
	}

	// 機密情報漏洩をスキャン
	err := a.scanForSecretLeaks(security)
	if err != nil {
		fmt.Printf("機密情報スキャン警告: %v\n", err)
	}

	// 不安全なパターンをスキャン
	err = a.scanInsecurePatterns(security)
	if err != nil {
		fmt.Printf("不安全パターンスキャン警告: %v\n", err)
	}

	// セキュリティスコア計算
	a.calculateSecurityScore(security)

	return security, nil
}

// 機密情報漏洩をスキャン
func (a *AdvancedProjectAnalyzer) scanForSecretLeaks(security *SecurityAnalysis) error {
	// 危険なパターン
	secretPatterns := map[string]string{
		"api_key":     `(?i)(api[_-]?key|apikey)\s*[:=]\s*["\']?([a-zA-Z0-9_\-]{20,})["\']?`,
		"password":    `(?i)(password|pwd)\s*[:=]\s*["\']?([^\s"']{8,})["\']?`,
		"token":       `(?i)(token|auth[_-]?token)\s*[:=]\s*["\']?([a-zA-Z0-9_\-\.]{20,})["\']?`,
		"secret":      `(?i)(secret|secret[_-]?key)\s*[:=]\s*["\']?([a-zA-Z0-9_\-]{16,})["\']?`,
		"private_key": `-----BEGIN (RSA |)PRIVATE KEY-----`,
	}

	return filepath.Walk(a.projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// バイナリファイルや無視すべきファイルをスキップ
		if a.shouldSkipFile(path) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		contentStr := string(content)
		lines := strings.Split(contentStr, "\n")

		for secretType, pattern := range secretPatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				continue
			}

			for lineNum, line := range lines {
				if re.MatchString(line) {
					relPath, _ := filepath.Rel(a.projectDir, path)
					leak := SecretLeak{
						File:    relPath,
						Line:    lineNum + 1,
						Type:    secretType,
						Pattern: pattern,
					}
					security.SecretLeaks = append(security.SecretLeaks, leak)
				}
			}
		}

		return nil
	})
}

// 不安全なパターンをスキャン
func (a *AdvancedProjectAnalyzer) scanInsecurePatterns(security *SecurityAnalysis) error {
	insecurePatterns := map[string]map[string]string{
		"sql_injection": {
			"pattern":     `(?i)(query|execute)\s*\(\s*["\']?.*\+.*["\']?\s*\)`,
			"description": "Potential SQL injection vulnerability",
			"severity":    "high",
		},
		"command_injection": {
			"pattern":     `(?i)(exec|system|shell_exec|passthru)\s*\(\s*.*\$`,
			"description": "Potential command injection vulnerability",
			"severity":    "high",
		},
		"weak_hash": {
			"pattern":     `(?i)(md5|sha1)\s*\(`,
			"description": "Use of weak hashing algorithm",
			"severity":    "medium",
		},
	}

	return filepath.Walk(a.projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if a.shouldSkipFile(path) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		contentStr := string(content)
		lines := strings.Split(contentStr, "\n")

		for patternName, patternInfo := range insecurePatterns {
			re, err := regexp.Compile(patternInfo["pattern"])
			if err != nil {
				continue
			}

			for lineNum, line := range lines {
				if re.MatchString(line) {
					relPath, _ := filepath.Rel(a.projectDir, path)
					insecure := InsecurePattern{
						File:        relPath,
						Line:        lineNum + 1,
						Pattern:     patternName,
						Description: patternInfo["description"],
						Severity:    patternInfo["severity"],
					}
					security.InsecurePatterns = append(security.InsecurePatterns, insecure)
				}
			}
		}

		return nil
	})
}

// ファイルをスキップするかを判断
func (a *AdvancedProjectAnalyzer) shouldSkipFile(path string) bool {
	skipPatterns := []string{
		".git/", "node_modules/", "vendor/", "target/", "__pycache__/",
		".exe", ".bin", ".so", ".dll", ".dylib",
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico",
		".zip", ".tar", ".gz", ".rar", ".7z",
		".min.js", ".min.css",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

// セキュリティスコアを計算
func (a *AdvancedProjectAnalyzer) calculateSecurityScore(security *SecurityAnalysis) {
	score := 100

	// 機密情報漏洩のペナルティ
	score -= len(security.SecretLeaks) * 15

	// 不安全パターンのペナルティ
	for _, pattern := range security.InsecurePatterns {
		switch pattern.Severity {
		case "critical":
			score -= 25
		case "high":
			score -= 15
		case "medium":
			score -= 10
		case "low":
			score -= 5
		}
	}

	if score < 0 {
		score = 0
	}

	security.SecurityScore = score
	security.VulnerabilityCount = len(security.SecretLeaks) + len(security.InsecurePatterns)
}

// プロジェクト健康度スコアを計算
func (a *AdvancedProjectAnalyzer) calculateHealthScore(analysis *AdvancedProjectAnalysis) *ProjectHealthScore {
	health := &ProjectHealthScore{
		Recommendations: []string{},
		Metrics:         make(map[string]interface{}),
	}

	// 各指標を計算
	health.CodeQuality = a.calculateCodeQualityScore(analysis)
	health.TestCoverage = a.calculateTestCoverageScore(analysis)
	health.Documentation = a.calculateDocumentationScore(analysis)
	health.Dependencies = a.calculateDependencyScore(analysis)
	health.Maintainability = a.calculateMaintainabilityScore(analysis)

	if analysis.SecurityAnalysis != nil {
		health.Security = analysis.SecurityAnalysis.SecurityScore
	} else {
		health.Security = 50 // デフォルト値
	}

	// 総合スコア計算
	health.OverallScore = (health.CodeQuality + health.TestCoverage + health.Documentation +
		health.Dependencies + health.Security + health.Maintainability) / 6

	// 推奨事項を生成
	a.generateRecommendations(health, analysis)

	return health
}

// コード品質スコアを計算
func (a *AdvancedProjectAnalyzer) calculateCodeQualityScore(analysis *AdvancedProjectAnalysis) int {
	_ = analysis // 将来の拡張用パラメータ
	score := 70  // 基本スコア

	// ビルドシステムの存在確認
	if len(analysis.BuildSystems) > 0 {
		score += 15
	}

	// アーキテクチャの組織化度
	if analysis.Architecture != nil && len(analysis.Architecture.Layers) > 2 {
		score += 10
	}

	// TODO: 実際のコード品質メトリクス（静的解析結果など）を統合

	if score > 100 {
		score = 100
	}

	return score
}

// テストカバレッジスコアを計算
func (a *AdvancedProjectAnalyzer) calculateTestCoverageScore(analysis *AdvancedProjectAnalysis) int {
	score := 30 // 基本スコア（テストファイルの存在を確認していない場合）

	if analysis.BasicInfo != nil {
		// テストファイルの存在確認
		for lang, count := range analysis.BasicInfo.FilesByLanguage {
			if strings.Contains(strings.ToLower(lang), "test") && count > 0 {
				score += 40
				break
			}
		}
	}

	// テスト用ディレクトリの存在確認
	testDirs := []string{"test", "tests", "spec", "specs", "__tests__"}
	for _, testDir := range testDirs {
		if _, err := os.Stat(filepath.Join(a.projectDir, testDir)); err == nil {
			score += 30
			break
		}
	}

	if score > 100 {
		score = 100
	}

	return score
}

// ドキュメントスコアを計算
func (a *AdvancedProjectAnalyzer) calculateDocumentationScore(analysis *AdvancedProjectAnalysis) int {
	score := 20 // 基本スコア

	// README の存在確認
	readmeFiles := []string{"README.md", "README.txt", "README.rst", "README"}
	for _, readme := range readmeFiles {
		if _, err := os.Stat(filepath.Join(a.projectDir, readme)); err == nil {
			score += 40
			break
		}
	}

	// ドキュメントディレクトリの存在確認
	docDirs := []string{"docs", "doc", "documentation"}
	for _, docDir := range docDirs {
		if _, err := os.Stat(filepath.Join(a.projectDir, docDir)); err == nil {
			score += 25
			break
		}
	}

	// CHANGELOG の存在確認
	changelogFiles := []string{"CHANGELOG.md", "CHANGELOG.txt", "HISTORY.md"}
	for _, changelog := range changelogFiles {
		if _, err := os.Stat(filepath.Join(a.projectDir, changelog)); err == nil {
			score += 15
			break
		}
	}

	if score > 100 {
		score = 100
	}

	return score
}

// 依存関係スコアを計算
func (a *AdvancedProjectAnalyzer) calculateDependencyScore(analysis *AdvancedProjectAnalysis) int {
	score := 80 // 基本スコア

	if analysis.DetailedDeps != nil {
		// 脆弱性のペナルティ
		score -= len(analysis.DetailedDeps.Vulnerabilities) * 10

		// 古い依存関係のペナルティ
		score -= len(analysis.DetailedDeps.Outdated) * 5

		// 依存関係の競合のペナルティ
		score -= len(analysis.DetailedDeps.Conflicts) * 15
	}

	if score < 0 {
		score = 0
	}

	return score
}

// 保守性スコアを計算
func (a *AdvancedProjectAnalyzer) calculateMaintainabilityScore(analysis *AdvancedProjectAnalysis) int {
	score := 60 // 基本スコア

	// プロジェクト構造の組織化度
	if analysis.Architecture != nil {
		score += len(analysis.Architecture.Layers) * 5
		score += len(analysis.Architecture.Modules) * 3
	}

	// ビルドシステムの多様性（適度な複雑さ）
	buildSystemCount := len(analysis.BuildSystems)
	if buildSystemCount >= 1 && buildSystemCount <= 3 {
		score += 15
	} else if buildSystemCount > 3 {
		score -= 5 // 複雑すぎる場合はペナルティ
	}

	if score > 100 {
		score = 100
	}

	return score
}

// 推奨事項を生成
func (a *AdvancedProjectAnalyzer) generateRecommendations(health *ProjectHealthScore, analysis *AdvancedProjectAnalysis) {
	// コード品質関連の推奨
	if health.CodeQuality < 70 {
		health.Recommendations = append(health.Recommendations, "コード品質向上のため、リンターやフォーマッターの導入を検討してください")
	}

	// テストカバレッジ関連の推奨
	if health.TestCoverage < 50 {
		health.Recommendations = append(health.Recommendations, "テストカバレッジを向上させるため、単体テストの追加を検討してください")
	}

	// ドキュメント関連の推奨
	if health.Documentation < 60 {
		health.Recommendations = append(health.Recommendations, "プロジェクトの理解を助けるため、README.mdやドキュメントの充実を検討してください")
	}

	// セキュリティ関連の推奨
	if health.Security < 80 {
		health.Recommendations = append(health.Recommendations, "セキュリティ向上のため、機密情報の適切な管理と脆弱性対策を検討してください")
	}

	// 依存関係関連の推奨
	if health.Dependencies < 70 {
		health.Recommendations = append(health.Recommendations, "依存関係の健全性向上のため、古いパッケージの更新を検討してください")
	}

	// ビルドシステム関連の推奨
	if len(analysis.BuildSystems) == 0 {
		health.Recommendations = append(health.Recommendations, "ビルド自動化のため、MakefileやCI/CDパイプラインの導入を検討してください")
	}
}
