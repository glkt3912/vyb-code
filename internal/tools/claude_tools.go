package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

// Claude Code相当のツール実装

// BashTool - セキュアなコマンド実行
type BashTool struct {
	constraints *security.Constraints
	workDir     string
	timeout     time.Duration
}

func NewBashTool(constraints *security.Constraints, workDir string) *BashTool {
	return &BashTool{
		constraints: constraints,
		workDir:     workDir,
		timeout:     30 * time.Second, // デフォルト30秒タイムアウト
	}
}

func (b *BashTool) Execute(command string, description string, timeoutMs int) (*ToolExecutionResult, error) {
	// タイムアウト設定
	timeout := b.timeout
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
		if timeout > 10*time.Minute { // 最大10分制限
			timeout = 10 * time.Minute
		}
	}

	// セキュリティ制約チェック
	if err := b.constraints.ValidateCommand(command); err != nil {
		return &ToolExecutionResult{
			Content:  fmt.Sprintf("コマンド実行拒否: %v", err),
			IsError:  true,
			Tool:     "bash",
			ExitCode: -1,
		}, err
	}

	// コマンド実行
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = b.workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe作成エラー: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe作成エラー: %w", err)
	}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return &ToolExecutionResult{
			Content:  fmt.Sprintf("コマンド開始エラー: %v", err),
			IsError:  true,
			Tool:     "bash",
			ExitCode: -1,
			Duration: time.Since(start).String(),
		}, err
	}

	// 出力を読み取り
	var stdoutBuf, stderrBuf strings.Builder
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			stdoutBuf.WriteString(scanner.Text())
			stdoutBuf.WriteString("\n")
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrBuf.WriteString(scanner.Text())
			stderrBuf.WriteString("\n")
		}
	}()

	err = cmd.Wait()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	// 出力構築
	output := stdoutBuf.String()
	if stderrBuf.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderrBuf.String()
	}

	return &ToolExecutionResult{
		Content:  output,
		IsError:  exitCode != 0,
		Tool:     "bash",
		ExitCode: exitCode,
		Duration: duration.String(),
		Metadata: map[string]interface{}{
			"command":     command,
			"description": description,
			"timeout_ms":  timeoutMs,
		},
	}, nil
}

// GlobTool - ファイルパターンマッチング
type GlobTool struct {
	workDir string
}

func NewGlobTool(workDir string) *GlobTool {
	return &GlobTool{workDir: workDir}
}

func (g *GlobTool) Find(pattern string, path string) (*ToolExecutionResult, error) {
	searchDir := g.workDir
	if path != "" {
		if filepath.IsAbs(path) {
			searchDir = path
		} else {
			searchDir = filepath.Join(g.workDir, path)
		}
	}

	matches, err := g.globRecursive(searchDir, pattern)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("Glob検索エラー: %v", err),
			IsError: true,
			Tool:    "glob",
		}, err
	}

	// 変更時刻でソート（新しい順）
	sort.Slice(matches, func(i, j int) bool {
		info1, err1 := os.Stat(matches[i])
		info2, err2 := os.Stat(matches[j])
		if err1 != nil || err2 != nil {
			return false
		}
		return info1.ModTime().After(info2.ModTime())
	})

	result := strings.Join(matches, "\n")
	return &ToolExecutionResult{
		Content: result,
		IsError: false,
		Tool:    "glob",
		Metadata: map[string]interface{}{
			"pattern":     pattern,
			"path":        path,
			"match_count": len(matches),
		},
	}, nil
}

func (g *GlobTool) globRecursive(dir, pattern string) ([]string, error) {
	var matches []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // スキップして継続
		}

		// 隠しファイル/ディレクトリをスキップ
		if strings.HasPrefix(filepath.Base(path), ".") && path != dir {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// パターンマッチング
		matched, matchErr := filepath.Match(pattern, filepath.Base(path))
		if matchErr != nil {
			return nil
		}

		if matched {
			relPath, relErr := filepath.Rel(g.workDir, path)
			if relErr == nil {
				matches = append(matches, relPath)
			} else {
				matches = append(matches, path)
			}
		}

		return nil
	})

	return matches, err
}

// GrepTool - 高度な検索機能
type GrepTool struct {
	workDir string
}

func NewGrepTool(workDir string) *GrepTool {
	return &GrepTool{workDir: workDir}
}

type GrepOptions struct {
	Pattern         string `json:"pattern"`
	Path            string `json:"path,omitempty"`
	Glob            string `json:"glob,omitempty"`
	Type            string `json:"type,omitempty"`
	OutputMode      string `json:"output_mode,omitempty"` // content, files_with_matches, count
	CaseInsensitive bool   `json:"case_insensitive,omitempty"`
	ContextBefore   int    `json:"context_before,omitempty"`
	ContextAfter    int    `json:"context_after,omitempty"`
	LineNumbers     bool   `json:"line_numbers,omitempty"`
	HeadLimit       int    `json:"head_limit,omitempty"`
	Multiline       bool   `json:"multiline,omitempty"`
}

func (g *GrepTool) Search(options GrepOptions) (*ToolExecutionResult, error) {
	searchDir := g.workDir
	if options.Path != "" {
		if filepath.IsAbs(options.Path) {
			searchDir = options.Path
		} else {
			searchDir = filepath.Join(g.workDir, options.Path)
		}
	}

	// 正規表現コンパイル
	pattern := options.Pattern
	if options.CaseInsensitive {
		pattern = "(?i)" + pattern
	}
	if options.Multiline {
		pattern = "(?s)" + pattern
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("正規表現エラー: %v", err),
			IsError: true,
			Tool:    "grep",
		}, err
	}

	matches, err := g.searchFiles(searchDir, regex, options)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("検索エラー: %v", err),
			IsError: true,
			Tool:    "grep",
		}, err
	}

	// 出力モードに応じて結果をフォーマット
	result := g.formatResults(matches, options)

	return &ToolExecutionResult{
		Content: result,
		IsError: false,
		Tool:    "grep",
		Metadata: map[string]interface{}{
			"pattern":     options.Pattern,
			"match_count": len(matches),
			"output_mode": options.OutputMode,
		},
	}, nil
}

type GrepMatch struct {
	File       string
	LineNumber int
	Line       string
	Context    []string
}

func (g *GrepTool) searchFiles(dir string, regex *regexp.Regexp, options GrepOptions) ([]GrepMatch, error) {
	var matches []GrepMatch

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// 隠しファイルをスキップ
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}

		// ファイル型フィルタリング
		if options.Type != "" && !g.matchFileType(path, options.Type) {
			return nil
		}

		// Globパターンフィルタリング
		if options.Glob != "" {
			matched, _ := filepath.Match(options.Glob, filepath.Base(path))
			if !matched {
				return nil
			}
		}

		fileMatches, err := g.searchInFile(path, regex, options)
		if err == nil {
			matches = append(matches, fileMatches...)
		}

		return nil
	})

	return matches, err
}

func (g *GrepTool) searchInFile(filePath string, regex *regexp.Regexp, options GrepOptions) ([]GrepMatch, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []GrepMatch
	scanner := bufio.NewScanner(file)
	lineNum := 1
	var lines []string

	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)

		if regex.MatchString(line) {
			relPath, _ := filepath.Rel(g.workDir, filePath)

			var context []string
			if options.ContextBefore > 0 || options.ContextAfter > 0 {
				start := max(0, lineNum-1-options.ContextBefore)
				end := min(len(lines), lineNum+options.ContextAfter)
				context = lines[start:end]
			}

			matches = append(matches, GrepMatch{
				File:       relPath,
				LineNumber: lineNum,
				Line:       line,
				Context:    context,
			})
		}
		lineNum++
	}

	return matches, scanner.Err()
}

func (g *GrepTool) matchFileType(path, fileType string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch fileType {
	case "go":
		return ext == ".go"
	case "js":
		return ext == ".js" || ext == ".jsx"
	case "ts":
		return ext == ".ts" || ext == ".tsx"
	case "py":
		return ext == ".py"
	case "java":
		return ext == ".java"
	case "c":
		return ext == ".c" || ext == ".h"
	case "cpp":
		return ext == ".cpp" || ext == ".cxx" || ext == ".cc" || ext == ".hpp"
	case "rust":
		return ext == ".rs"
	default:
		return true
	}
}

func (g *GrepTool) formatResults(matches []GrepMatch, options GrepOptions) string {
	if len(matches) == 0 {
		return ""
	}

	// HeadLimitの適用
	if options.HeadLimit > 0 && len(matches) > options.HeadLimit {
		matches = matches[:options.HeadLimit]
	}

	switch options.OutputMode {
	case "files_with_matches":
		seen := make(map[string]bool)
		var files []string
		for _, match := range matches {
			if !seen[match.File] {
				files = append(files, match.File)
				seen[match.File] = true
			}
		}
		return strings.Join(files, "\n")

	case "count":
		counts := make(map[string]int)
		for _, match := range matches {
			counts[match.File]++
		}
		var result []string
		for file, count := range counts {
			result = append(result, fmt.Sprintf("%s:%d", file, count))
		}
		return strings.Join(result, "\n")

	default: // "content"
		var result []string
		for _, match := range matches {
			if options.LineNumbers {
				result = append(result, fmt.Sprintf("%s:%d:%s", match.File, match.LineNumber, match.Line))
			} else {
				result = append(result, fmt.Sprintf("%s:%s", match.File, match.Line))
			}
		}
		return strings.Join(result, "\n")
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// LSTool - 拡張ディレクトリリスト
type LSTool struct {
	workDir string
}

func NewLSTool(workDir string) *LSTool {
	return &LSTool{workDir: workDir}
}

func (l *LSTool) List(path string, ignore []string) (*ToolExecutionResult, error) {
	targetDir := l.workDir
	if path != "" {
		if filepath.IsAbs(path) {
			targetDir = path
		} else {
			targetDir = filepath.Join(l.workDir, path)
		}
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("ディレクトリ読み取りエラー: %v", err),
			IsError: true,
			Tool:    "ls",
		}, err
	}

	var result []string
	for _, entry := range entries {
		name := entry.Name()

		// 無視パターンチェック
		if l.shouldIgnore(name, ignore) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if entry.IsDir() {
			result = append(result, fmt.Sprintf("- %s/", name))
		} else {
			size := info.Size()
			modTime := info.ModTime().Format("2006-01-02 15:04:05")
			result = append(result, fmt.Sprintf("  %s (%d bytes, %s)", name, size, modTime))
		}
	}

	return &ToolExecutionResult{
		Content: strings.Join(result, "\n"),
		IsError: false,
		Tool:    "ls",
		Metadata: map[string]interface{}{
			"path":        path,
			"entry_count": len(result),
		},
	}, nil
}

func (l *LSTool) shouldIgnore(name string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// WebFetchTool - Web内容取得
type WebFetchTool struct {
	client  *http.Client
	timeout time.Duration
}

func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}
}

func (w *WebFetchTool) Fetch(url string, prompt string) (*ToolExecutionResult, error) {
	// HTTP/HTTPSチェック
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("リクエスト作成エラー: %v", err),
			IsError: true,
			Tool:    "webfetch",
		}, err
	}

	req.Header.Set("User-Agent", "vyb-code/1.0 (Local AI Assistant)")

	resp, err := w.client.Do(req)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("HTTP取得エラー: %v", err),
			IsError: true,
			Tool:    "webfetch",
		}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("HTTP エラー: %d %s", resp.StatusCode, resp.Status),
			IsError: true,
			Tool:    "webfetch",
		}, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("レスポンス読み取りエラー: %v", err),
			IsError: true,
			Tool:    "webfetch",
		}, err
	}

	content := string(body)

	// HTMLからMarkdownへの基本的な変換
	content = w.htmlToMarkdown(content)

	return &ToolExecutionResult{
		Content: content,
		IsError: false,
		Tool:    "webfetch",
		Metadata: map[string]interface{}{
			"url":            url,
			"status_code":    resp.StatusCode,
			"content_type":   resp.Header.Get("Content-Type"),
			"content_length": len(content),
			"prompt":         prompt,
		},
	}, nil
}

func (w *WebFetchTool) htmlToMarkdown(html string) string {
	// 基本的なHTML→Markdown変換
	content := html

	// HTMLタグの除去（簡易版）
	re := regexp.MustCompile(`<[^>]*>`)
	content = re.ReplaceAllString(content, "")

	// 実際の実装では、より高度なHTML→Markdown変換ライブラリを使用
	return strings.TrimSpace(content)
}
