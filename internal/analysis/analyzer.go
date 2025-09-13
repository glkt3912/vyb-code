package analysis

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// プロジェクト分析器の実装
type projectAnalyzer struct {
	config *AnalysisConfig
	cache  map[string]*cachedAnalysis
	mutex  sync.RWMutex
}

// キャッシュされた分析結果
type cachedAnalysis struct {
	Analysis  *ProjectAnalysis
	CachedAt  time.Time
	ExpiresAt time.Time
}

// 新しいプロジェクト分析器を作成
func NewProjectAnalyzer(config *AnalysisConfig) ProjectAnalyzer {
	if config == nil {
		config = DefaultAnalysisConfig()
	}

	return &projectAnalyzer{
		config: config,
		cache:  make(map[string]*cachedAnalysis),
	}
}

// プロジェクト全体の分析を実行
func (pa *projectAnalyzer) AnalyzeProject(projectPath string) (*ProjectAnalysis, error) {
	// キャッシュをチェック
	if pa.config.EnableCaching {
		if cached, err := pa.GetCachedAnalysis(projectPath); err == nil {
			return cached, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), pa.config.Timeout)
	defer cancel()

	analysis := &ProjectAnalysis{
		ProjectPath:     projectPath,
		ProjectName:     filepath.Base(projectPath),
		AnalyzedAt:      time.Now(),
		AnalysisVersion: "1.0.0",
		Metadata:        make(map[string]interface{}),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	addError := func(err error) {
		if err != nil {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
		}
	}

	// 並行して各種分析を実行
	analysisTasks := []func() error{
		func() error { return pa.analyzeLanguageAndFramework(analysis) },
		func() error { return pa.analyzeFileStructure(analysis) },
		func() error { return pa.analyzeGitInfo(analysis) },
		func() error { return pa.analyzeBuildSystem(analysis) },
		func() error { return pa.analyzeTestingFramework(analysis) },
	}

	if pa.config.IncludeDependencies {
		analysisTasks = append(analysisTasks, func() error {
			deps, err := pa.AnalyzeDependencies(projectPath)
			if err == nil {
				mu.Lock()
				analysis.Dependencies = deps
				mu.Unlock()
			}
			return err
		})
	}

	if pa.config.QualityMetrics {
		analysisTasks = append(analysisTasks, func() error {
			metrics, err := pa.AnalyzeQuality(projectPath)
			if err == nil {
				mu.Lock()
				analysis.QualityMetrics = metrics
				mu.Unlock()
			}
			return err
		})
	}

	if pa.config.SecurityScan {
		analysisTasks = append(analysisTasks, func() error {
			issues, err := pa.AnalyzeSecurity(projectPath)
			if err == nil {
				mu.Lock()
				analysis.SecurityIssues = issues
				mu.Unlock()
			}
			return err
		})
	}

	// 並行実行
	for _, task := range analysisTasks {
		wg.Add(1)
		go func(t func() error) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				addError(ctx.Err())
			default:
				addError(t())
			}
		}(task)
	}

	wg.Wait()

	// 技術スタック分析（他の分析結果に依存）
	pa.analyzeTechStack(analysis)

	// 推奨事項生成
	if recommendations, err := pa.GenerateRecommendations(analysis); err == nil {
		analysis.Recommendations = recommendations
	}

	// エラーハンドリング
	if len(errors) > 0 && ctx.Err() != nil {
		return nil, fmt.Errorf("プロジェクト分析がタイムアウトしました: %v", errors[0])
	}

	// キャッシュに保存
	if pa.config.EnableCaching {
		pa.CacheAnalysis(projectPath, analysis)
	}

	return analysis, nil
}

// 言語とフレームワークの分析
func (pa *projectAnalyzer) analyzeLanguageAndFramework(analysis *ProjectAnalysis) error {
	projectPath := analysis.ProjectPath

	// 設定ファイルの検出
	configFiles := map[string]string{
		"package.json":     "JavaScript/Node.js",
		"go.mod":           "Go",
		"requirements.txt": "Python",
		"Cargo.toml":       "Rust",
		"pom.xml":          "Java/Maven",
		"build.gradle":     "Java/Gradle",
		"composer.json":    "PHP",
		"Gemfile":          "Ruby",
		"pubspec.yaml":     "Dart/Flutter",
	}

	// フレームワーク検出パターン
	frameworkPatterns := map[string][]string{
		"React":   {"react", "\"react\":", "from 'react'"},
		"Vue":     {"vue", "\"vue\":", "from 'vue'"},
		"Angular": {"@angular", "angular", "from '@angular'"},
		"Express": {"express", "\"express\":", "require('express')"},
		"Django":  {"django", "Django", "from django"},
		"Flask":   {"flask", "Flask", "from flask"},
		"Spring":  {"springframework", "spring-boot", "@SpringBootApplication"},
		"Gin":     {"github.com/gin-gonic/gin", "gin.Default"},
		"Echo":    {"github.com/labstack/echo", "echo.New"},
		"Actix":   {"actix-web", "use actix_web"},
	}

	// 設定ファイルをチェック
	for configFile, language := range configFiles {
		configPath := filepath.Join(projectPath, configFile)
		if _, err := os.Stat(configPath); err == nil {
			analysis.Language = language

			// 設定ファイルの内容を読んでフレームワークを検出
			if content, err := os.ReadFile(configPath); err == nil {
				contentStr := string(content)
				for framework, patterns := range frameworkPatterns {
					for _, pattern := range patterns {
						if strings.Contains(contentStr, pattern) {
							analysis.Framework = framework
							break
						}
					}
					if analysis.Framework != "" {
						break
					}
				}
			}
			break
		}
	}

	// 設定ファイルが見つからない場合、ソースファイルから推測
	if analysis.Language == "" {
		analysis.Language = pa.detectLanguageFromFiles(projectPath)
	}

	return nil
}

// ファイルから言語を検出
func (pa *projectAnalyzer) detectLanguageFromFiles(projectPath string) string {
	extensionCounts := make(map[string]int)

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// 除外パターンをチェック
		relPath, _ := filepath.Rel(projectPath, path)
		for _, pattern := range pa.config.ExcludePatterns {
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
		return "Unknown"
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

	// 拡張子から言語をマッピング
	extToLanguage := map[string]string{
		".go":   "Go",
		".js":   "JavaScript",
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
		return language
	}

	return "Unknown"
}

// ファイル構造の分析
func (pa *projectAnalyzer) analyzeFileStructure(analysis *ProjectAnalysis) error {
	projectPath := analysis.ProjectPath
	structure := &FileStructure{
		Languages:       make(map[string]int),
		Directories:     make([]DirectoryInfo, 0),
		LargestFiles:    make([]FileInfo, 0),
		RecentlyChanged: make([]FileInfo, 0),
		Patterns:        make([]ArchitecturePattern, 0),
	}

	allFiles := make([]FileInfo, 0)

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)

		// 除外パターンをチェック
		for _, pattern := range pa.config.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				return filepath.SkipDir
			}
		}

		if info.IsDir() {
			dirInfo := pa.analyzeDirectoryInfo(path, projectPath)
			structure.Directories = append(structure.Directories, *dirInfo)
			return nil
		}

		// ファイルサイズ制限をチェック
		if info.Size() > pa.config.MaxFileSize {
			return nil
		}

		fileInfo := pa.createFileInfo(path, info, projectPath)
		allFiles = append(allFiles, *fileInfo)

		// 統計を更新
		structure.TotalFiles++
		if lines, err := pa.countLines(path); err == nil {
			fileInfo.Lines = lines
			structure.TotalLines += lines
		}

		// 言語別ファイル数
		ext := strings.ToLower(filepath.Ext(path))
		if ext != "" {
			structure.Languages[ext]++
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 最大ファイルを取得（サイズ順）
	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].Size > allFiles[j].Size
	})
	if len(allFiles) > 10 {
		structure.LargestFiles = allFiles[:10]
	} else {
		structure.LargestFiles = allFiles
	}

	// 最近変更されたファイルを取得
	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].LastModified.After(allFiles[j].LastModified)
	})
	if len(allFiles) > 10 {
		structure.RecentlyChanged = allFiles[:10]
	} else {
		structure.RecentlyChanged = allFiles
	}

	// アーキテクチャパターンを検出
	structure.Patterns = pa.detectArchitecturePatterns(projectPath, structure.Directories)

	analysis.FileStructure = structure
	return nil
}

// ディレクトリ情報の分析
func (pa *projectAnalyzer) analyzeDirectoryInfo(dirPath, projectPath string) *DirectoryInfo {
	relPath, _ := filepath.Rel(projectPath, dirPath)
	dirInfo := &DirectoryInfo{
		Path:       relPath,
		FileCount:  0,
		LineCount:  0,
		Purpose:    pa.detectDirectoryPurpose(relPath),
		Importance: pa.calculateDirectoryImportance(relPath),
	}

	// ディレクトリ内のファイルを数える
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || path == dirPath {
			return nil
		}

		dirInfo.FileCount++
		if lines, err := pa.countLines(path); err == nil {
			dirInfo.LineCount += lines
		}
		return nil
	})

	return dirInfo
}

// ファイル情報を作成
func (pa *projectAnalyzer) createFileInfo(filePath string, info os.FileInfo, projectPath string) *FileInfo {
	relPath, _ := filepath.Rel(projectPath, filePath)

	fileInfo := &FileInfo{
		Path:         relPath,
		Size:         info.Size(),
		Language:     pa.detectFileLanguage(filePath),
		LastModified: info.ModTime(),
		Purpose:      pa.detectFilePurpose(relPath),
		Dependencies: make([]string, 0),
	}

	// 複雑度を計算（簡易版）
	if complexity, err := pa.calculateFileComplexity(filePath); err == nil {
		fileInfo.Complexity = complexity
	}

	return fileInfo
}

// 行数をカウント
func (pa *projectAnalyzer) countLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines, scanner.Err()
}

// ディレクトリの目的を検出
func (pa *projectAnalyzer) detectDirectoryPurpose(dirPath string) string {
	purpose := map[string]string{
		"src":          "source",
		"source":       "source",
		"lib":          "source",
		"app":          "source",
		"test":         "test",
		"tests":        "test",
		"__tests__":    "test",
		"spec":         "test",
		"config":       "config",
		"configs":      "config",
		"docs":         "docs",
		"doc":          "docs",
		"assets":       "assets",
		"static":       "assets",
		"public":       "assets",
		"build":        "build",
		"dist":         "build",
		"target":       "build",
		"vendor":       "dependencies",
		"node_modules": "dependencies",
	}

	pathParts := strings.Split(dirPath, "/")
	for _, part := range pathParts {
		if p, exists := purpose[strings.ToLower(part)]; exists {
			return p
		}
	}

	return "other"
}

// ディレクトリの重要度を計算
func (pa *projectAnalyzer) calculateDirectoryImportance(dirPath string) float64 {
	importance := map[string]float64{
		"src":          0.9,
		"source":       0.9,
		"lib":          0.8,
		"app":          0.9,
		"test":         0.6,
		"tests":        0.6,
		"config":       0.7,
		"docs":         0.4,
		"assets":       0.3,
		"build":        0.2,
		"vendor":       0.1,
		"node_modules": 0.1,
	}

	pathParts := strings.Split(dirPath, "/")
	maxImportance := 0.5 // デフォルト値

	for _, part := range pathParts {
		if imp, exists := importance[strings.ToLower(part)]; exists {
			if imp > maxImportance {
				maxImportance = imp
			}
		}
	}

	// パスの深さによる重要度の調整
	depth := len(pathParts)
	if depth > 3 {
		maxImportance *= 0.8 // 深いパスは重要度を下げる
	}

	return maxImportance
}

// ファイルの言語を検出
func (pa *projectAnalyzer) detectFileLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	extToLanguage := map[string]string{
		".go":         "Go",
		".js":         "JavaScript",
		".mjs":        "JavaScript",
		".jsx":        "JavaScript",
		".ts":         "TypeScript",
		".tsx":        "TypeScript",
		".py":         "Python",
		".java":       "Java",
		".rs":         "Rust",
		".php":        "PHP",
		".rb":         "Ruby",
		".c":          "C",
		".cpp":        "C++",
		".cc":         "C++",
		".cxx":        "C++",
		".cs":         "C#",
		".dart":       "Dart",
		".html":       "HTML",
		".css":        "CSS",
		".scss":       "SCSS",
		".sass":       "SASS",
		".json":       "JSON",
		".yaml":       "YAML",
		".yml":        "YAML",
		".xml":        "XML",
		".md":         "Markdown",
		".sh":         "Shell",
		".bash":       "Shell",
		".dockerfile": "Docker",
	}

	if language, exists := extToLanguage[ext]; exists {
		return language
	}

	// 特殊ファイルの処理
	filename := strings.ToLower(filepath.Base(filePath))
	if filename == "dockerfile" || strings.HasPrefix(filename, "dockerfile.") {
		return "Docker"
	}
	if filename == "makefile" || strings.HasPrefix(filename, "makefile.") {
		return "Make"
	}

	return "Unknown"
}

// ファイルの目的を検出
func (pa *projectAnalyzer) detectFilePurpose(filePath string) string {
	filename := strings.ToLower(filepath.Base(filePath))
	dirPath := strings.ToLower(filepath.Dir(filePath))

	// テストファイルの検出
	if strings.Contains(filename, "test") || strings.Contains(filename, "spec") ||
		strings.Contains(dirPath, "test") || strings.Contains(dirPath, "spec") {
		return "test"
	}

	// 設定ファイルの検出
	configFiles := []string{
		"config", "configuration", "settings", "options",
		".env", "dockerfile", "makefile",
	}
	for _, cf := range configFiles {
		if strings.Contains(filename, cf) {
			return "config"
		}
	}

	// ドキュメントファイルの検出
	if strings.HasSuffix(filename, ".md") ||
		strings.Contains(filename, "readme") ||
		strings.Contains(filename, "changelog") ||
		strings.Contains(filename, "license") {
		return "docs"
	}

	// ビルド・デプロイ関連
	if strings.Contains(filename, "build") ||
		strings.Contains(filename, "deploy") ||
		strings.Contains(filename, "ci") ||
		strings.Contains(filename, "cd") {
		return "build"
	}

	return "source"
}

// ファイルの複雑度を計算（簡易版）
func (pa *projectAnalyzer) calculateFileComplexity(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	complexity := 1 // 基本複雑度

	// 複雑度を上げるキーワード
	complexityKeywords := []string{
		"if", "else", "switch", "case", "for", "while",
		"try", "catch", "&&", "||", "?", ":",
	}

	keywordRegex := make([]*regexp.Regexp, len(complexityKeywords))
	for i, keyword := range complexityKeywords {
		keywordRegex[i] = regexp.MustCompile(`\b` + regexp.QuoteMeta(keyword) + `\b`)
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}

		for _, regex := range keywordRegex {
			matches := regex.FindAllString(line, -1)
			complexity += len(matches)
		}
	}

	return complexity, scanner.Err()
}

// アーキテクチャパターンを検出
func (pa *projectAnalyzer) detectArchitecturePatterns(projectPath string, directories []DirectoryInfo) []ArchitecturePattern {
	patterns := make([]ArchitecturePattern, 0)

	// MVCパターンの検出
	if pa.hasMVCStructure(directories) {
		patterns = append(patterns, ArchitecturePattern{
			Name:        "MVC (Model-View-Controller)",
			Confidence:  0.8,
			Description: "プロジェクトはMVCアーキテクチャパターンを使用しています",
			Files:       pa.findMVCFiles(projectPath),
		})
	}

	// マイクロサービスパターンの検出
	if pa.hasMicroservicesStructure(directories) {
		patterns = append(patterns, ArchitecturePattern{
			Name:        "Microservices",
			Confidence:  0.7,
			Description: "プロジェクトはマイクロサービスアーキテクチャを使用しています",
			Files:       pa.findMicroservicesFiles(projectPath),
		})
	}

	// レイヤードアーキテクチャの検出
	if pa.hasLayeredStructure(directories) {
		patterns = append(patterns, ArchitecturePattern{
			Name:        "Layered Architecture",
			Confidence:  0.6,
			Description: "プロジェクトはレイヤードアーキテクチャを使用しています",
			Files:       pa.findLayeredFiles(projectPath),
		})
	}

	return patterns
}

// MVCパターンの構造をチェック
func (pa *projectAnalyzer) hasMVCStructure(directories []DirectoryInfo) bool {
	hasModel, hasView, hasController := false, false, false

	for _, dir := range directories {
		path := strings.ToLower(dir.Path)
		if strings.Contains(path, "model") {
			hasModel = true
		}
		if strings.Contains(path, "view") || strings.Contains(path, "template") {
			hasView = true
		}
		if strings.Contains(path, "controller") || strings.Contains(path, "handler") {
			hasController = true
		}
	}

	return hasModel && hasView && hasController
}

// マイクロサービス構造をチェック
func (pa *projectAnalyzer) hasMicroservicesStructure(directories []DirectoryInfo) bool {
	serviceCount := 0
	for _, dir := range directories {
		path := strings.ToLower(dir.Path)
		if strings.Contains(path, "service") || strings.Contains(path, "api") {
			serviceCount++
		}
	}
	return serviceCount >= 2
}

// レイヤード構造をチェック
func (pa *projectAnalyzer) hasLayeredStructure(directories []DirectoryInfo) bool {
	layers := []string{"presentation", "business", "data", "domain", "infrastructure"}
	foundLayers := 0

	for _, dir := range directories {
		path := strings.ToLower(dir.Path)
		for _, layer := range layers {
			if strings.Contains(path, layer) {
				foundLayers++
				break
			}
		}
	}

	return foundLayers >= 3
}

// MVC関連ファイルを検索
func (pa *projectAnalyzer) findMVCFiles(projectPath string) []string {
	files := make([]string, 0)

	filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)
		lowerPath := strings.ToLower(relPath)

		if strings.Contains(lowerPath, "model") ||
			strings.Contains(lowerPath, "view") ||
			strings.Contains(lowerPath, "controller") {
			files = append(files, relPath)
		}

		return nil
	})

	return files
}

// マイクロサービス関連ファイルを検索
func (pa *projectAnalyzer) findMicroservicesFiles(projectPath string) []string {
	files := make([]string, 0)

	filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)
		lowerPath := strings.ToLower(relPath)

		if strings.Contains(lowerPath, "service") ||
			strings.Contains(lowerPath, "api") ||
			strings.Contains(lowerPath, "microservice") {
			files = append(files, relPath)
		}

		return nil
	})

	return files
}

// レイヤード関連ファイルを検索
func (pa *projectAnalyzer) findLayeredFiles(projectPath string) []string {
	files := make([]string, 0)
	layers := []string{"presentation", "business", "data", "domain", "infrastructure"}

	filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)
		lowerPath := strings.ToLower(relPath)

		for _, layer := range layers {
			if strings.Contains(lowerPath, layer) {
				files = append(files, relPath)
				break
			}
		}

		return nil
	})

	return files
}

// Git情報の分析
func (pa *projectAnalyzer) analyzeGitInfo(analysis *ProjectAnalysis) error {
	projectPath := analysis.ProjectPath

	// .gitディレクトリの存在確認
	gitPath := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		return nil // Gitリポジトリではない
	}

	gitInfo := &GitInfo{}

	// 現在のブランチを取得
	if branch, err := pa.execGitCommand(projectPath, "branch", "--show-current"); err == nil {
		gitInfo.CurrentBranch = strings.TrimSpace(branch)
	}

	// 最後のコミット情報を取得
	if commit, err := pa.execGitCommand(projectPath, "rev-parse", "HEAD"); err == nil {
		gitInfo.LastCommit = strings.TrimSpace(commit)[:8] // 短縮形
	}

	// コミット数を取得
	if count, err := pa.execGitCommand(projectPath, "rev-list", "--count", "HEAD"); err == nil {
		if countInt, err := strconv.Atoi(strings.TrimSpace(count)); err == nil {
			gitInfo.CommitCount = countInt
		}
	}

	// リモートURLを取得
	if remote, err := pa.execGitCommand(projectPath, "remote", "get-url", "origin"); err == nil {
		gitInfo.RemoteURL = strings.TrimSpace(remote)
	}

	// 変更があるかチェック
	if status, err := pa.execGitCommand(projectPath, "status", "--porcelain"); err == nil {
		gitInfo.HasChanges = len(strings.TrimSpace(status)) > 0
	}

	// アクティブなブランチを取得
	if branches, err := pa.execGitCommand(projectPath, "branch", "-a"); err == nil {
		branchLines := strings.Split(branches, "\n")
		for _, line := range branchLines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "remotes/") {
				line = strings.TrimPrefix(line, "* ")
				gitInfo.ActiveBranches = append(gitInfo.ActiveBranches, line)
			}
		}
	}

	// 最後のアクティビティ時間を取得
	if lastCommit, err := pa.execGitCommand(projectPath, "log", "-1", "--format=%ct"); err == nil {
		if timestamp, err := strconv.ParseInt(strings.TrimSpace(lastCommit), 10, 64); err == nil {
			gitInfo.LastActivity = time.Unix(timestamp, 0)
		}
	}

	analysis.GitInfo = gitInfo
	return nil
}

// Gitコマンドを実行
func (pa *projectAnalyzer) execGitCommand(projectPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// ビルドシステムの分析
func (pa *projectAnalyzer) analyzeBuildSystem(analysis *ProjectAnalysis) error {
	projectPath := analysis.ProjectPath

	buildSystems := map[string]*BuildSystem{
		"Makefile": {
			Name:          "Make",
			ConfigFile:    "Makefile",
			BuildCommands: []string{"make", "make build"},
			TestCommands:  []string{"make test"},
			LintCommands:  []string{"make lint"},
		},
		"package.json": {
			Name:          "npm",
			ConfigFile:    "package.json",
			BuildCommands: []string{"npm run build", "npm install"},
			TestCommands:  []string{"npm test", "npm run test"},
			LintCommands:  []string{"npm run lint"},
		},
		"go.mod": {
			Name:          "Go Modules",
			ConfigFile:    "go.mod",
			BuildCommands: []string{"go build", "go install"},
			TestCommands:  []string{"go test ./..."},
			LintCommands:  []string{"go vet ./..."},
		},
		"pom.xml": {
			Name:          "Maven",
			ConfigFile:    "pom.xml",
			BuildCommands: []string{"mvn compile", "mvn package"},
			TestCommands:  []string{"mvn test"},
			LintCommands:  []string{"mvn checkstyle:check"},
		},
		"build.gradle": {
			Name:          "Gradle",
			ConfigFile:    "build.gradle",
			BuildCommands: []string{"gradle build", "./gradlew build"},
			TestCommands:  []string{"gradle test", "./gradlew test"},
			LintCommands:  []string{"gradle check", "./gradlew check"},
		},
		"Cargo.toml": {
			Name:          "Cargo",
			ConfigFile:    "Cargo.toml",
			BuildCommands: []string{"cargo build", "cargo build --release"},
			TestCommands:  []string{"cargo test"},
			LintCommands:  []string{"cargo clippy"},
		},
	}

	// ビルドシステムを検出
	for configFile, buildSystem := range buildSystems {
		configPath := filepath.Join(projectPath, configFile)
		if _, err := os.Stat(configPath); err == nil {
			analysis.BuildSystem = buildSystem

			// 設定ファイルからスクリプトを抽出
			if buildSystem.Name == "npm" {
				scripts, err := pa.extractNpmScripts(configPath)
				if err == nil {
					buildSystem.Scripts = scripts
				}
			}

			break
		}
	}

	return nil
}

// npmスクリプトを抽出
func (pa *projectAnalyzer) extractNpmScripts(packageJsonPath string) (map[string]string, error) {
	content, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return nil, err
	}

	var packageData map[string]interface{}
	if err := json.Unmarshal(content, &packageData); err != nil {
		return nil, err
	}

	scripts := make(map[string]string)
	if scriptsData, exists := packageData["scripts"].(map[string]interface{}); exists {
		for name, command := range scriptsData {
			if cmdStr, ok := command.(string); ok {
				scripts[name] = cmdStr
			}
		}
	}

	return scripts, nil
}

// テストフレームワークの分析
func (pa *projectAnalyzer) analyzeTestingFramework(analysis *ProjectAnalysis) error {
	projectPath := analysis.ProjectPath

	testingFrameworks := map[string]*TestingFramework{
		"Jest": {
			Name:            "Jest",
			ConfigFiles:     []string{"jest.config.js", "jest.config.json", "package.json"},
			TestDirectories: []string{"__tests__", "test", "tests"},
			TestCommands:    []string{"npm test", "jest"},
		},
		"Go Test": {
			Name:            "Go Test",
			ConfigFiles:     []string{"go.mod"},
			TestDirectories: []string{},
			TestCommands:    []string{"go test ./..."},
		},
		"pytest": {
			Name:            "pytest",
			ConfigFiles:     []string{"pytest.ini", "setup.cfg", "pyproject.toml"},
			TestDirectories: []string{"tests", "test"},
			TestCommands:    []string{"pytest", "python -m pytest"},
		},
		"JUnit": {
			Name:            "JUnit",
			ConfigFiles:     []string{"pom.xml", "build.gradle"},
			TestDirectories: []string{"src/test", "test"},
			TestCommands:    []string{"mvn test", "gradle test"},
		},
	}

	// テストフレームワークを検出
	for _, framework := range testingFrameworks {
		found := false

		// 設定ファイルの存在確認
		for _, configFile := range framework.ConfigFiles {
			configPath := filepath.Join(projectPath, configFile)
			if _, err := os.Stat(configPath); err == nil {
				found = true
				break
			}
		}

		// テストディレクトリの存在確認
		if !found {
			for _, testDir := range framework.TestDirectories {
				testPath := filepath.Join(projectPath, testDir)
				if stat, err := os.Stat(testPath); err == nil && stat.IsDir() {
					found = true
					break
				}
			}
		}

		if found {
			analysis.TestingFramework = framework

			// テストファイルを検索
			testFiles, err := pa.findTestFiles(projectPath, framework)
			if err == nil {
				framework.TestFiles = testFiles
			}

			break
		}
	}

	return nil
}

// テストファイルを検索
func (pa *projectAnalyzer) findTestFiles(projectPath string, framework *TestingFramework) ([]string, error) {
	testFiles := make([]string, 0)

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)
		filename := strings.ToLower(info.Name())

		// 除外パターンをチェック
		for _, pattern := range pa.config.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				return nil
			}
		}

		// テストファイルの検出
		isTestFile := false

		// ファイル名パターン
		if strings.Contains(filename, "test") || strings.Contains(filename, "spec") {
			isTestFile = true
		}

		// ディレクトリパターン
		dirPath := strings.ToLower(filepath.Dir(relPath))
		for _, testDir := range framework.TestDirectories {
			if strings.Contains(dirPath, strings.ToLower(testDir)) {
				isTestFile = true
				break
			}
		}

		if isTestFile {
			testFiles = append(testFiles, relPath)
		}

		return nil
	})

	return testFiles, err
}

// 技術スタック分析
func (pa *projectAnalyzer) analyzeTechStack(analysis *ProjectAnalysis) {
	techStack := make([]Technology, 0)

	// 言語を追加
	if analysis.Language != "" {
		techStack = append(techStack, Technology{
			Name:         analysis.Language,
			Type:         "language",
			Confidence:   0.9,
			Usage:        "primary",
			DetectedFrom: []string{"file_analysis"},
		})
	}

	// フレームワークを追加
	if analysis.Framework != "" {
		techStack = append(techStack, Technology{
			Name:         analysis.Framework,
			Type:         "framework",
			Confidence:   0.8,
			Usage:        "primary",
			DetectedFrom: []string{"dependency_analysis"},
		})
	}

	// ビルドシステムを追加
	if analysis.BuildSystem != nil {
		techStack = append(techStack, Technology{
			Name:         analysis.BuildSystem.Name,
			Type:         "tool",
			Confidence:   0.9,
			Usage:        "development",
			DetectedFrom: []string{"config_file"},
		})
	}

	// テストフレームワークを追加
	if analysis.TestingFramework != nil {
		techStack = append(techStack, Technology{
			Name:         analysis.TestingFramework.Name,
			Type:         "framework",
			Confidence:   0.8,
			Usage:        "testing",
			DetectedFrom: []string{"config_file", "test_directory"},
		})
	}

	// 依存関係から技術を抽出
	for _, dep := range analysis.Dependencies {
		confidence := 0.7
		if dep.Type == "direct" {
			confidence = 0.8
		}

		techStack = append(techStack, Technology{
			Name:         dep.Name,
			Version:      dep.Version,
			Type:         "library",
			Confidence:   confidence,
			Usage:        pa.determineDependencyUsage(dep),
			DetectedFrom: []string{"dependency_file"},
		})
	}

	analysis.TechStack = techStack
}

// 依存関係の使用用途を判定
func (pa *projectAnalyzer) determineDependencyUsage(dep Dependency) string {
	if dep.Type == "dev" {
		return "development"
	}
	if dep.Type == "test" {
		return "testing"
	}
	if dep.Type == "peer" || dep.Type == "optional" {
		return "secondary"
	}
	return "primary"
}

// その他の必要なメソッドの実装（続きは次のファイルで）
func (pa *projectAnalyzer) AnalyzeFile(filePath string) (*FileInfo, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	projectPath := filepath.Dir(filePath)
	return pa.createFileInfo(filePath, info, projectPath), nil
}

func (pa *projectAnalyzer) AnalyzeDirectory(dirPath string) (*DirectoryInfo, error) {
	projectPath := filepath.Dir(dirPath)
	return pa.analyzeDirectoryInfo(dirPath, projectPath), nil
}

// キャッシュ関連の実装は次のファイルで続きます...
