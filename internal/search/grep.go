package search

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// 高性能グレップ実装
type Grep struct {
	engine     *Engine
	maxWorkers int
	maxResults int
	timeout    time.Duration
}

// グレップ結果の詳細
type GrepMatch struct {
	File       FileInfo `json:"file"`
	LineNumber int      `json:"lineNumber"`
	Line       string   `json:"line"`
	Before     []string `json:"before,omitempty"`
	After      []string `json:"after,omitempty"`
	Column     int      `json:"column"`
	MatchText  string   `json:"matchText"`
}

// グレップオプション
type GrepOptions struct {
	Pattern       string   `json:"pattern"`
	Regex         bool     `json:"regex"`
	IgnoreCase    bool     `json:"ignoreCase"`
	WholeWord     bool     `json:"wholeWord"`
	Invert        bool     `json:"invert"`        // マッチしない行を表示
	LineNumbers   bool     `json:"lineNumbers"`   // 行番号表示
	Count         bool     `json:"count"`         // マッチ数のみ表示
	FilesOnly     bool     `json:"filesOnly"`     // ファイル名のみ表示
	ContextBefore int      `json:"contextBefore"` // 前の行数
	ContextAfter  int      `json:"contextAfter"`  // 後の行数
	MaxMatches    int      `json:"maxMatches"`    // ファイル毎の最大マッチ数
	Include       []string `json:"include"`       // 含めるファイルパターン
	Exclude       []string `json:"exclude"`       // 除外するファイルパターン
}

// 新しいグレップインスタンスを作成
func NewGrep(engine *Engine) *Grep {
	return &Grep{
		engine:     engine,
		maxWorkers: 8,
		maxResults: 1000,
		timeout:    30 * time.Second,
	}
}

// グレップ実行のメイン関数
func (g *Grep) Search(options GrepOptions) ([]GrepMatch, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	// 検索パターンを準備
	regex, err := g.compilePattern(options)
	if err != nil {
		return nil, fmt.Errorf("パターンコンパイルエラー: %w", err)
	}

	// 対象ファイルを選択
	files := g.selectFiles(options)
	if len(files) == 0 {
		return []GrepMatch{}, nil
	}

	// 並行検索を実行
	return g.searchConcurrent(ctx, files, regex, options)
}

// 検索パターンをコンパイル
func (g *Grep) compilePattern(options GrepOptions) (*regexp.Regexp, error) {
	pattern := options.Pattern

	if !options.Regex {
		pattern = regexp.QuoteMeta(pattern)
	}

	if options.WholeWord {
		pattern = `\b` + pattern + `\b`
	}

	if options.IgnoreCase {
		pattern = "(?i)" + pattern
	}

	return regexp.Compile(pattern)
}

// 対象ファイルを選択
func (g *Grep) selectFiles(options GrepOptions) []FileInfo {
	g.engine.mu.RLock()
	defer g.engine.mu.RUnlock()

	var files []FileInfo
	includeFilter := compilePatterns(options.Include)
	excludeFilter := compilePatterns(options.Exclude)

	for _, fileInfo := range g.engine.indexedFiles {
		if !fileInfo.Indexed {
			continue
		}

		// パターンフィルターをチェック
		if len(includeFilter) > 0 && !matchesPatterns(fileInfo.RelativePath, includeFilter) {
			continue
		}
		if matchesPatterns(fileInfo.RelativePath, excludeFilter) {
			continue
		}

		files = append(files, fileInfo)
	}

	return files
}

// 並行検索を実行
func (g *Grep) searchConcurrent(ctx context.Context, files []FileInfo, regex *regexp.Regexp, options GrepOptions) ([]GrepMatch, error) {
	resultChan := make(chan []GrepMatch, len(files))
	semaphore := make(chan struct{}, g.maxWorkers)
	var wg sync.WaitGroup

	// 各ファイルを並行処理
	for _, file := range files {
		wg.Add(1)
		go func(f FileInfo) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()

				matches, err := g.searchFile(f, regex, options)
				if err == nil {
					resultChan <- matches
				} else {
					resultChan <- []GrepMatch{}
				}
			case <-ctx.Done():
				resultChan <- []GrepMatch{}
			}
		}(file)
	}

	// 結果収集を開始
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 結果をマージ
	var allMatches []GrepMatch
	for matches := range resultChan {
		allMatches = append(allMatches, matches...)

		// 最大結果数チェック
		if len(allMatches) >= g.maxResults {
			break
		}
	}

	// 結果数制限
	if len(allMatches) > g.maxResults {
		allMatches = allMatches[:g.maxResults]
	}

	return allMatches, nil
}

// 単一ファイルを検索
func (g *Grep) searchFile(fileInfo FileInfo, regex *regexp.Regexp, options GrepOptions) ([]GrepMatch, error) {
	file, err := os.Open(fileInfo.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var matches []GrepMatch
	var lines []string
	lineNum := 0

	// コンテキスト表示のため全行を保持
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lines = append(lines, line)

		// パターンマッチング
		matched := regex.MatchString(line)
		if options.Invert {
			matched = !matched
		}

		if matched {
			match := GrepMatch{
				File:       fileInfo,
				LineNumber: lineNum,
				Line:       line,
			}

			// マッチ位置を特定
			if !options.Invert && !options.Count {
				if loc := regex.FindStringIndex(line); loc != nil {
					match.Column = loc[0] + 1
					match.MatchText = line[loc[0]:loc[1]]
				}
			}

			matches = append(matches, match)

			// ファイル毎の最大マッチ数チェック
			if options.MaxMatches > 0 && len(matches) >= options.MaxMatches {
				break
			}
		}
	}

	// コンテキスト行を追加
	if (options.ContextBefore > 0 || options.ContextAfter > 0) && !options.Count && !options.FilesOnly {
		for i := range matches {
			g.addContext(&matches[i], lines, options)
		}
	}

	return matches, scanner.Err()
}

// コンテキスト行を追加
func (g *Grep) addContext(match *GrepMatch, lines []string, options GrepOptions) {
	lineIdx := match.LineNumber - 1 // 0ベースのインデックス

	// 前のコンテキスト
	if options.ContextBefore > 0 {
		start := lineIdx - options.ContextBefore
		if start < 0 {
			start = 0
		}
		match.Before = lines[start:lineIdx]
	}

	// 後のコンテキスト
	if options.ContextAfter > 0 {
		end := lineIdx + 1 + options.ContextAfter
		if end > len(lines) {
			end = len(lines)
		}
		match.After = lines[lineIdx+1 : end]
	}
}

// マッチ数のみを取得
func (g *Grep) Count(options GrepOptions) (map[string]int, error) {
	options.Count = true
	matches, err := g.Search(options)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, match := range matches {
		counts[match.File.RelativePath]++
	}

	return counts, nil
}

// マッチするファイル名のみを取得
func (g *Grep) ListFiles(options GrepOptions) ([]string, error) {
	options.FilesOnly = true
	matches, err := g.Search(options)
	if err != nil {
		return nil, err
	}

	fileSet := make(map[string]bool)
	var files []string

	for _, match := range matches {
		if !fileSet[match.File.RelativePath] {
			files = append(files, match.File.RelativePath)
			fileSet[match.File.RelativePath] = true
		}
	}

	return files, nil
}

// 結果をコンソール向けにフォーマット
func (g *Grep) FormatResults(matches []GrepMatch, options GrepOptions) string {
	var builder strings.Builder

	if options.Count {
		// カウント表示
		counts := make(map[string]int)
		for _, match := range matches {
			counts[match.File.RelativePath]++
		}

		for file, count := range counts {
			builder.WriteString(fmt.Sprintf("%s:%d\n", file, count))
		}
		return builder.String()
	}

	if options.FilesOnly {
		// ファイル名のみ表示
		fileSet := make(map[string]bool)
		for _, match := range matches {
			if !fileSet[match.File.RelativePath] {
				builder.WriteString(match.File.RelativePath + "\n")
				fileSet[match.File.RelativePath] = true
			}
		}
		return builder.String()
	}

	// 通常の検索結果表示
	currentFile := ""
	for _, match := range matches {
		// ファイル名が変わったら表示
		if match.File.RelativePath != currentFile {
			if currentFile != "" {
				builder.WriteString("\n")
			}
			builder.WriteString(fmt.Sprintf("📁 %s\n", match.File.RelativePath))
			currentFile = match.File.RelativePath
		}

		// コンテキスト前を表示
		for i, line := range match.Before {
			lineNum := match.LineNumber - len(match.Before) + i
			builder.WriteString(fmt.Sprintf("  %d- %s\n", lineNum, line))
		}

		// マッチ行を表示（ハイライト付き）
		if options.LineNumbers {
			builder.WriteString(fmt.Sprintf("  %d: ", match.LineNumber))
		} else {
			builder.WriteString("  ")
		}

		// マッチ部分をハイライト
		if match.Column > 0 && match.MatchText != "" {
			before := match.Line[:match.Column-1]
			after := match.Line[match.Column-1+len(match.MatchText):]
			builder.WriteString(fmt.Sprintf("%s\033[1;31m%s\033[0m%s\n", before, match.MatchText, after))
		} else {
			builder.WriteString(match.Line + "\n")
		}

		// コンテキスト後を表示
		for i, line := range match.After {
			lineNum := match.LineNumber + 1 + i
			builder.WriteString(fmt.Sprintf("  %d+ %s\n", lineNum, line))
		}
	}

	return builder.String()
}
