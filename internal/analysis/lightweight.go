package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 軽量分析器 - 基本的な分析のみを高速で実行
type LightweightAnalyzer struct {
	config *AnalysisConfig
}

// 新しい軽量分析器を作成
func NewLightweightAnalyzer(config *AnalysisConfig) *LightweightAnalyzer {
	if config == nil {
		config = &AnalysisConfig{
			EnableCaching:  true,
			CacheExpiry:    10 * time.Minute, // 短めのキャッシュ
			AnalysisDepth:  "quick",
			IncludeTests:   false, // テスト分析をスキップ
			SecurityScan:   false, // セキュリティスキャンをスキップ
			QualityMetrics: false, // 品質メトリクスをスキップ
			ExcludePatterns: []string{
				"node_modules/**",
				"vendor/**",
				".git/**",
				"dist/**",
				"build/**",
				"target/**",
				"*.log",
				"*.tmp",
			},
			MaxFileSize: 512 * 1024,       // 512KB制限（軽量化）
			Timeout:     10 * time.Second, // 短いタイムアウト
		}
	}

	return &LightweightAnalyzer{
		config: config,
	}
}

// 軽量プロジェクト分析
func (la *LightweightAnalyzer) AnalyzeProject(projectPath string) (*ProjectAnalysis, error) {
	startTime := time.Now()

	analysis := &ProjectAnalysis{
		ProjectPath:     projectPath,
		ProjectName:     filepath.Base(projectPath),
		AnalyzedAt:      startTime,
		AnalysisVersion: "1.0.0-lightweight",
		Metadata:        make(map[string]interface{}),
	}

	// 基本言語検出（最優先）
	if err := la.detectLanguage(analysis); err != nil {
		// エラーがあっても続行
		analysis.Language = "Unknown"
	}

	// 基本ファイル構造（軽量版）
	if err := la.analyzeBasicStructure(analysis); err != nil {
		// エラーがあっても続行
	}

	// Git情報（軽量版）
	la.analyzeGitInfoLightweight(analysis)

	// 軽量技術スタック検出
	la.detectTechStackLightweight(analysis)

	// 実行時間を記録
	duration := time.Since(startTime)
	analysis.Metadata["analysis_duration"] = duration.String()
	analysis.Metadata["analysis_mode"] = "lightweight"

	return analysis, nil
}

// 言語検出（高速版）
func (la *LightweightAnalyzer) detectLanguage(analysis *ProjectAnalysis) error {
	projectPath := analysis.ProjectPath

	// 設定ファイルの検出（最も信頼性が高い）
	configFiles := map[string]string{
		"go.mod":           "Go",
		"package.json":     "JavaScript",
		"requirements.txt": "Python",
		"Cargo.toml":       "Rust",
		"pom.xml":          "Java",
		"build.gradle":     "Java",
		"composer.json":    "PHP",
		"Gemfile":          "Ruby",
	}

	// 設定ファイルをチェック
	for configFile, language := range configFiles {
		configPath := filepath.Join(projectPath, configFile)
		if _, err := os.Stat(configPath); err == nil {
			analysis.Language = language
			return nil
		}
	}

	// 設定ファイルが見つからない場合は拡張子から推測
	return la.detectLanguageFromExtensions(analysis)
}

// 拡張子から言語検出（高速版）
func (la *LightweightAnalyzer) detectLanguageFromExtensions(analysis *ProjectAnalysis) error {
	projectPath := analysis.ProjectPath
	extensionCounts := make(map[string]int)
	maxFiles := 50 // 最大50ファイルまでしか見ない（高速化）

	fileCount := 0
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// ファイル数制限
		fileCount++
		if fileCount > maxFiles {
			return filepath.SkipDir
		}

		// 除外パターンチェック
		relPath, _ := filepath.Rel(projectPath, path)
		for _, pattern := range la.config.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				return nil
			}
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != "" {
			extensionCounts[ext]++
		}
		return nil
	})

	if err != nil {
		return err
	}

	// 最も多い拡張子から言語を推測
	maxCount := 0
	dominantExt := ""
	for ext, count := range extensionCounts {
		if count > maxCount {
			maxCount = count
			dominantExt = ext
		}
	}

	// 拡張子マッピング
	extToLanguage := map[string]string{
		".go":   "Go",
		".js":   "JavaScript",
		".mjs":  "JavaScript",
		".ts":   "TypeScript",
		".py":   "Python",
		".java": "Java",
		".rs":   "Rust",
		".php":  "PHP",
		".rb":   "Ruby",
		".c":    "C",
		".cpp":  "C++",
		".cs":   "C#",
		".dart": "Dart",
	}

	if language, exists := extToLanguage[dominantExt]; exists {
		analysis.Language = language
	} else {
		analysis.Language = "Unknown"
	}

	return nil
}

// 基本構造分析（軽量版）
func (la *LightweightAnalyzer) analyzeBasicStructure(analysis *ProjectAnalysis) error {
	projectPath := analysis.ProjectPath
	structure := &FileStructure{
		Languages:   make(map[string]int),
		Directories: make([]DirectoryInfo, 0),
	}

	maxFiles := 100 // ファイル数制限
	fileCount := 0

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)

		// 除外パターンチェック
		for _, pattern := range la.config.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			// 重要なディレクトリのみ記録
			if la.isImportantDirectory(relPath) {
				structure.Directories = append(structure.Directories, DirectoryInfo{
					Path:       relPath,
					Purpose:    la.detectDirectoryPurpose(relPath),
					Importance: la.calculateDirectoryImportance(relPath),
				})
			}
			return nil
		}

		// ファイル数制限
		fileCount++
		if fileCount > maxFiles {
			return filepath.SkipDir
		}

		// ファイルサイズ制限
		if info.Size() > la.config.MaxFileSize {
			return nil
		}

		structure.TotalFiles++

		// 言語別ファイル数
		ext := strings.ToLower(filepath.Ext(path))
		if ext != "" {
			structure.Languages[ext]++
		}

		return nil
	})

	analysis.FileStructure = structure
	return err
}

// 重要なディレクトリかどうか判定
func (la *LightweightAnalyzer) isImportantDirectory(dirPath string) bool {
	importantDirs := []string{
		"src", "source", "lib", "app",
		"test", "tests", "__tests__",
		"config", "configs",
		"docs", "doc",
		"cmd", "main",
	}

	pathLower := strings.ToLower(dirPath)
	for _, important := range importantDirs {
		if strings.Contains(pathLower, important) {
			return true
		}
	}

	return false
}

// ディレクトリ目的検出（軽量版）
func (la *LightweightAnalyzer) detectDirectoryPurpose(dirPath string) string {
	purpose := map[string]string{
		"src":       "source",
		"source":    "source",
		"lib":       "source",
		"app":       "source",
		"cmd":       "source",
		"main":      "source",
		"test":      "test",
		"tests":     "test",
		"__tests__": "test",
		"spec":      "test",
		"config":    "config",
		"configs":   "config",
		"docs":      "docs",
		"doc":       "docs",
	}

	pathLower := strings.ToLower(dirPath)
	for keyword, p := range purpose {
		if strings.Contains(pathLower, keyword) {
			return p
		}
	}

	return "other"
}

// ディレクトリ重要度計算（軽量版）
func (la *LightweightAnalyzer) calculateDirectoryImportance(dirPath string) float64 {
	importance := map[string]float64{
		"src":    0.9,
		"source": 0.9,
		"lib":    0.8,
		"app":    0.9,
		"cmd":    0.8,
		"main":   0.8,
		"test":   0.6,
		"tests":  0.6,
		"config": 0.7,
		"docs":   0.4,
	}

	pathLower := strings.ToLower(dirPath)
	maxImportance := 0.3 // デフォルト値

	for keyword, imp := range importance {
		if strings.Contains(pathLower, keyword) {
			if imp > maxImportance {
				maxImportance = imp
			}
		}
	}

	return maxImportance
}

// Git情報分析（軽量版）
func (la *LightweightAnalyzer) analyzeGitInfoLightweight(analysis *ProjectAnalysis) {
	projectPath := analysis.ProjectPath

	// .gitディレクトリの存在確認のみ
	gitPath := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		return // Gitリポジトリではない
	}

	analysis.GitInfo = &GitInfo{
		Repository: "detected", // 詳細分析はスキップ
	}
}

// 技術スタック検出（軽量版）
func (la *LightweightAnalyzer) detectTechStackLightweight(analysis *ProjectAnalysis) {
	techStack := make([]Technology, 0)

	// 言語を追加
	if analysis.Language != "" && analysis.Language != "Unknown" {
		techStack = append(techStack, Technology{
			Name:         analysis.Language,
			Type:         "language",
			Confidence:   0.9,
			Usage:        "primary",
			DetectedFrom: []string{"config_file_or_extension"},
		})
	}

	// 主要な設定ファイルから技術を検出
	projectPath := analysis.ProjectPath
	configTech := map[string]Technology{
		"package.json": {
			Name:         "Node.js",
			Type:         "runtime",
			Confidence:   0.8,
			Usage:        "primary",
			DetectedFrom: []string{"package.json"},
		},
		"go.mod": {
			Name:         "Go Modules",
			Type:         "dependency_manager",
			Confidence:   0.9,
			Usage:        "primary",
			DetectedFrom: []string{"go.mod"},
		},
		"Cargo.toml": {
			Name:         "Cargo",
			Type:         "dependency_manager",
			Confidence:   0.9,
			Usage:        "primary",
			DetectedFrom: []string{"Cargo.toml"},
		},
		"requirements.txt": {
			Name:         "pip",
			Type:         "dependency_manager",
			Confidence:   0.8,
			Usage:        "primary",
			DetectedFrom: []string{"requirements.txt"},
		},
		"Dockerfile": {
			Name:         "Docker",
			Type:         "containerization",
			Confidence:   0.9,
			Usage:        "deployment",
			DetectedFrom: []string{"Dockerfile"},
		},
	}

	for configFile, tech := range configTech {
		configPath := filepath.Join(projectPath, configFile)
		if _, err := os.Stat(configPath); err == nil {
			techStack = append(techStack, tech)
		}
	}

	analysis.TechStack = techStack
}

// サマリー生成（軽量版）
func (la *LightweightAnalyzer) GenerateSummary(analysis *ProjectAnalysis) *AnalysisSummary {
	summary := &AnalysisSummary{
		ProjectName:  analysis.ProjectName,
		Language:     analysis.Language,
		LastAnalyzed: analysis.AnalyzedAt,
	}

	if analysis.FileStructure != nil {
		summary.FileCount = analysis.FileStructure.TotalFiles
	}

	// 主要技術を抽出
	for _, tech := range analysis.TechStack {
		if tech.Usage == "primary" {
			summary.MainTechnologies = append(summary.MainTechnologies, tech.Name)
		}
	}

	return summary
}

// 分析レベル設定
type LightweightLevel int

const (
	LevelMinimal  LightweightLevel = iota // 最小限（1-2秒）
	LevelBasic                            // 基本（3-5秒）
	LevelStandard                         // 標準（8-12秒）
)

// レベル別分析
func (la *LightweightAnalyzer) AnalyzeWithLevel(projectPath string, level LightweightLevel) (*ProjectAnalysis, error) {
	// レベルに応じて設定を調整
	switch level {
	case LevelMinimal:
		la.config.MaxFileSize = 256 * 1024 // 256KB
		la.config.Timeout = 5 * time.Second
	case LevelBasic:
		la.config.MaxFileSize = 512 * 1024 // 512KB
		la.config.Timeout = 8 * time.Second
	case LevelStandard:
		la.config.MaxFileSize = 1024 * 1024 // 1MB
		la.config.Timeout = 12 * time.Second
	}

	return la.AnalyzeProject(projectPath)
}
