package search

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ファイル検索エンジン
type Engine struct {
	mu              sync.RWMutex
	workspaceDir    string
	excludePatterns []*regexp.Regexp
	indexedFiles    map[string]FileInfo
	lastIndexTime   time.Time
	maxFileSize     int64
}

// ファイル情報
type FileInfo struct {
	Path         string    `json:"path"`
	RelativePath string    `json:"relativePath"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"modTime"`
	Language     string    `json:"language"`
	Indexed      bool      `json:"indexed"`
	LineCount    int       `json:"lineCount"`
}

// 検索結果
type SearchResult struct {
	File       FileInfo `json:"file"`
	LineNumber int      `json:"lineNumber"`
	Line       string   `json:"line"`
	Context    []string `json:"context,omitempty"` // 前後の行
	MatchStart int      `json:"matchStart"`
	MatchEnd   int      `json:"matchEnd"`
}

// 検索オプション
type SearchOptions struct {
	Pattern         string   `json:"pattern"`
	CaseSensitive   bool     `json:"caseSensitive"`
	WholeWord       bool     `json:"wholeWord"`
	Regex           bool     `json:"regex"`
	IncludePatterns []string `json:"includePatterns"`
	ExcludePatterns []string `json:"excludePatterns"`
	MaxResults      int      `json:"maxResults"`
	ContextLines    int      `json:"contextLines"`
}

// 新しい検索エンジンを作成
func NewEngine(workspaceDir string) *Engine {
	return &Engine{
		workspaceDir:    workspaceDir,
		excludePatterns: compileDefaultExcludes(),
		indexedFiles:    make(map[string]FileInfo),
		maxFileSize:     10 * 1024 * 1024, // 10MB
	}
}

// デフォルトの除外パターンをコンパイル
func compileDefaultExcludes() []*regexp.Regexp {
	patterns := []string{
		`\.git/`,
		`\.vscode/`,
		`\.idea/`,
		`node_modules/`,
		`\.DS_Store`,
		`\.log$`,
		`\.tmp$`,
		`\.cache/`,
		`vendor/`,
		`target/`,
		`build/`,
		`dist/`,
		`\.pyc$`,
		`__pycache__/`,
		`\.class$`,
		`\.o$`,
		`\.so$`,
		`\.dll$`,
		`\.exe$`,
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			compiled = append(compiled, regex)
		}
	}

	return compiled
}

// プロジェクトファイルのインデックスを作成
func (e *Engine) IndexProject() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.indexedFiles = make(map[string]FileInfo)

	err := filepath.Walk(e.workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // エラーが発生したファイルはスキップ
		}

		// ディレクトリをスキップ
		if info.IsDir() {
			return nil
		}

		// 除外パターンをチェック
		relPath, _ := filepath.Rel(e.workspaceDir, path)
		if e.shouldExclude(relPath) {
			return nil
		}

		// ファイルサイズチェック
		if info.Size() > e.maxFileSize {
			return nil
		}

		// ファイル情報を作成
		fileInfo := FileInfo{
			Path:         path,
			RelativePath: relPath,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			Language:     detectLanguage(path),
			Indexed:      false,
		}

		// テキストファイルの場合は行数をカウント
		if isTextFile(path) {
			if lineCount, err := countLines(path); err == nil {
				fileInfo.LineCount = lineCount
				fileInfo.Indexed = true
			}
		}

		e.indexedFiles[path] = fileInfo
		return nil
	})

	e.lastIndexTime = time.Now()
	return err
}

// ファイルを除外すべきかチェック
func (e *Engine) shouldExclude(path string) bool {
	for _, pattern := range e.excludePatterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

// テキストファイルの検索
func (e *Engine) SearchInFiles(options SearchOptions) ([]SearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []SearchResult
	resultCount := 0

	// 検索パターンを準備
	var searchRegex *regexp.Regexp
	if options.Regex {
		regex, err := regexp.Compile(options.Pattern)
		if err != nil {
			return nil, fmt.Errorf("正規表現エラー: %w", err)
		}
		searchRegex = regex
	} else {
		pattern := regexp.QuoteMeta(options.Pattern)
		if options.WholeWord {
			pattern = `\b` + pattern + `\b`
		}
		if !options.CaseSensitive {
			pattern = "(?i)" + pattern
		}
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		searchRegex = regex
	}

	// ファイルフィルターを準備
	includeFilter := compilePatterns(options.IncludePatterns)
	excludeFilter := compilePatterns(options.ExcludePatterns)

	// 各ファイルを検索
	for _, fileInfo := range e.indexedFiles {
		if !fileInfo.Indexed {
			continue
		}

		// パターンフィルターをチェック
		if !matchesPatterns(fileInfo.RelativePath, includeFilter) {
			continue
		}
		if matchesPatterns(fileInfo.RelativePath, excludeFilter) {
			continue
		}

		// ファイル内容を検索
		fileResults, err := e.searchInFile(fileInfo, searchRegex, options)
		if err != nil {
			continue // エラーファイルはスキップ
		}

		results = append(results, fileResults...)
		resultCount += len(fileResults)

		// 最大結果数チェック
		if options.MaxResults > 0 && resultCount >= options.MaxResults {
			break
		}
	}

	// 結果をソート（ファイル名順）
	sort.Slice(results, func(i, j int) bool {
		if results[i].File.RelativePath == results[j].File.RelativePath {
			return results[i].LineNumber < results[j].LineNumber
		}
		return results[i].File.RelativePath < results[j].File.RelativePath
	})

	// 最大結果数で切り詰め
	if options.MaxResults > 0 && len(results) > options.MaxResults {
		results = results[:options.MaxResults]
	}

	return results, nil
}

// 単一ファイル内を検索
func (e *Engine) searchInFile(fileInfo FileInfo, regex *regexp.Regexp, options SearchOptions) ([]SearchResult, error) {
	file, err := os.Open(fileInfo.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []SearchResult
	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// ファイル全体を読み込み（コンテキスト表示のため）
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lines = append(lines, line)

		// パターンマッチング
		matches := regex.FindAllStringIndex(line, -1)
		for _, match := range matches {
			result := SearchResult{
				File:       fileInfo,
				LineNumber: lineNum,
				Line:       line,
				MatchStart: match[0],
				MatchEnd:   match[1],
			}

			// コンテキスト行を追加
			if options.ContextLines > 0 {
				result.Context = e.getContext(lines, lineNum-1, options.ContextLines)
			}

			results = append(results, result)
		}
	}

	return results, scanner.Err()
}

// コンテキスト行を取得
func (e *Engine) getContext(lines []string, targetLine, contextLines int) []string {
	start := targetLine - contextLines
	end := targetLine + contextLines + 1

	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}

	return lines[start:end]
}

// ファイル名でフィルター
func (e *Engine) FindFiles(pattern string) ([]FileInfo, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var regex *regexp.Regexp
	var err error

	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		// Globパターンを正規表現に変換
		regexPattern := globToRegex(pattern)
		regex, err = regexp.Compile(regexPattern)
	} else {
		// 部分一致検索
		regexPattern := "(?i)" + regexp.QuoteMeta(pattern)
		regex, err = regexp.Compile(regexPattern)
	}

	if err != nil {
		return nil, fmt.Errorf("パターンエラー: %w", err)
	}

	var results []FileInfo
	for _, fileInfo := range e.indexedFiles {
		if regex.MatchString(fileInfo.RelativePath) {
			results = append(results, fileInfo)
		}
	}

	// ファイル名順でソート
	sort.Slice(results, func(i, j int) bool {
		return results[i].RelativePath < results[j].RelativePath
	})

	return results, nil
}

// インデックス統計を取得
func (e *Engine) GetIndexStats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := map[string]interface{}{
		"total_files":     len(e.indexedFiles),
		"indexed_files":   0,
		"last_index_time": e.lastIndexTime,
	}

	languageCount := make(map[string]int)
	indexedCount := 0

	for _, fileInfo := range e.indexedFiles {
		if fileInfo.Indexed {
			indexedCount++
		}
		languageCount[fileInfo.Language]++
	}

	stats["indexed_files"] = indexedCount
	stats["languages"] = languageCount

	return stats
}

// ヘルパー関数群

// パターンリストをコンパイル
func compilePatterns(patterns []string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			compiled = append(compiled, regex)
		}
	}
	return compiled
}

// パターンにマッチするかチェック
func matchesPatterns(path string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

// Globパターンを正規表現に変換
func globToRegex(glob string) string {
	regex := regexp.QuoteMeta(glob)
	regex = strings.ReplaceAll(regex, "\\*", ".*")
	regex = strings.ReplaceAll(regex, "\\?", ".")
	return "^" + regex + "$"
}

// ファイルの言語を検出
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	langMap := map[string]string{
		".go":   "Go",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".py":   "Python",
		".java": "Java",
		".cpp":  "C++",
		".c":    "C",
		".rs":   "Rust",
		".php":  "PHP",
		".rb":   "Ruby",
		".sh":   "Shell",
		".bash": "Shell",
		".zsh":  "Shell",
		".fish": "Shell",
		".ps1":  "PowerShell",
		".sql":  "SQL",
		".html": "HTML",
		".css":  "CSS",
		".scss": "SCSS",
		".less": "LESS",
		".json": "JSON",
		".yaml": "YAML",
		".yml":  "YAML",
		".xml":  "XML",
		".md":   "Markdown",
		".txt":  "Text",
	}

	if lang, exists := langMap[ext]; exists {
		return lang
	}

	return "Unknown"
}

// テキストファイルかどうか判定
func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	textExts := []string{
		".go", ".js", ".ts", ".py", ".java", ".cpp", ".c", ".rs",
		".php", ".rb", ".sh", ".bash", ".zsh", ".fish", ".ps1",
		".sql", ".html", ".css", ".scss", ".less", ".json",
		".yaml", ".yml", ".xml", ".md", ".txt", ".toml",
		".ini", ".conf", ".config", ".env", ".gitignore",
		".dockerfile", ".makefile", ".cmake",
	}

	for _, textExt := range textExts {
		if ext == textExt {
			return true
		}
	}

	// ファイル名での判定
	basename := strings.ToLower(filepath.Base(path))
	specialFiles := []string{
		"makefile", "dockerfile", "rakefile", "gemfile",
		"readme", "license", "changelog", "authors",
	}

	for _, special := range specialFiles {
		if basename == special {
			return true
		}
	}

	return false
}

// ファイルの行数をカウント
func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}

	return count, scanner.Err()
}

// インデックスの再構築が必要かチェック
func (e *Engine) NeedsReindex() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 1時間以上経過した場合は再インデックス
	return time.Since(e.lastIndexTime) > time.Hour
}

// 特定のファイルパターンで検索
func (e *Engine) SearchByPattern(pattern string) ([]FileInfo, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	regex, err := regexp.Compile("(?i)" + regexp.QuoteMeta(pattern))
	if err != nil {
		return nil, err
	}

	var results []FileInfo
	for _, fileInfo := range e.indexedFiles {
		if regex.MatchString(fileInfo.RelativePath) {
			results = append(results, fileInfo)
		}
	}

	return results, nil
}

// 言語別ファイル検索
func (e *Engine) SearchByLanguage(language string) []FileInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []FileInfo
	for _, fileInfo := range e.indexedFiles {
		if strings.EqualFold(fileInfo.Language, language) {
			results = append(results, fileInfo)
		}
	}

	return results
}

// 最近変更されたファイルを取得
func (e *Engine) GetRecentFiles(limit int, since time.Duration) []FileInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()

	cutoff := time.Now().Add(-since)
	var results []FileInfo

	for _, fileInfo := range e.indexedFiles {
		if fileInfo.ModTime.After(cutoff) {
			results = append(results, fileInfo)
		}
	}

	// 変更時間順でソート（新しい順）
	sort.Slice(results, func(i, j int) bool {
		return results[i].ModTime.After(results[j].ModTime)
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// ファイル内容のプレビューを取得
func (e *Engine) GetFilePreview(path string, lines int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var preview []string
	lineCount := 0

	for scanner.Scan() && lineCount < lines {
		preview = append(preview, scanner.Text())
		lineCount++
	}

	return preview, scanner.Err()
}
