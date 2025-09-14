package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/glkt/vyb-code/internal/security"
)

// UnifiedGrepTool - 統一Grepツール（高度なファイル検索）
type UnifiedGrepTool struct {
	*BaseTool
}

// GrepResult - Grep検索結果
type GrepResult struct {
	File        string `json:"file"`
	LineNumber  int    `json:"line_number"`
	Line        string `json:"line"`
	Match       string `json:"match"`
	ColumnStart int    `json:"column_start"`
	ColumnEnd   int    `json:"column_end"`
}

// GrepSummary - Grep検索サマリー
type GrepSummary struct {
	Pattern          string       `json:"pattern"`
	SearchPath       string       `json:"search_path"`
	Results          []GrepResult `json:"results"`
	TotalFiles       int          `json:"total_files"`
	MatchCount       int          `json:"match_count"`
	FilesWithMatches int          `json:"files_with_matches"`
}

// NewUnifiedGrepTool - 新しい統一Grepツールを作成
func NewUnifiedGrepTool(constraints *security.Constraints) *UnifiedGrepTool {
	base := NewBaseTool("grep", "A powerful search tool built on regex patterns with file filtering", "1.0.0", CategorySearch)
	base.AddCapability(CapabilitySearch)
	base.AddCapability(CapabilityFileRead)
	base.SetConstraints(constraints)

	// スキーマ設定
	schema := ToolSchema{
		Name:        "grep",
		Description: "A powerful search tool built on regex patterns with file filtering",
		Version:     "1.0.0",
		Parameters: map[string]Parameter{
			"pattern": {
				Type:        "string",
				Description: "The regular expression pattern to search for",
			},
			"path": {
				Type:        "string",
				Description: "File or directory to search in (defaults to current directory)",
			},
			"output_mode": {
				Type:        "string",
				Description: "Output mode: 'content', 'files_with_matches', 'count' (default: files_with_matches)",
				Enum:        []string{"content", "files_with_matches", "count"},
			},
			"-i": {
				Type:        "boolean",
				Description: "Case insensitive search",
			},
			"-n": {
				Type:        "boolean",
				Description: "Show line numbers in output",
			},
			"glob": {
				Type:        "string",
				Description: "Glob pattern to filter files (e.g., '*.js', '*.{ts,tsx}')",
			},
			"type": {
				Type:        "string",
				Description: "File type to search (js, py, go, etc.)",
			},
			"head_limit": {
				Type:        "number",
				Description: "Limit output to first N results",
				Minimum:     floatPtr(1),
			},
		},
		Required: []string{"pattern"},
		Examples: []ToolExample{
			{
				Description: "Search for function definitions in Go files",
				Parameters: map[string]interface{}{
					"pattern": "func\\s+\\w+",
					"type":    "go",
				},
			},
		},
	}
	base.SetSchema(schema)

	return &UnifiedGrepTool{BaseTool: base}
}

// Execute - Grepツールを実行
func (t *UnifiedGrepTool) Execute(ctx context.Context, request *ToolRequest) (*ToolResponse, error) {
	if err := t.ValidateRequest(request); err != nil {
		return nil, err
	}

	// パラメータ取得
	pattern, _ := request.Parameters["pattern"].(string)

	// オプション取得
	searchPath := "."
	if path, ok := request.Parameters["path"].(string); ok && path != "" {
		searchPath = path
	}

	outputMode := "files_with_matches"
	if mode, ok := request.Parameters["output_mode"].(string); ok {
		outputMode = mode
	}

	caseInsensitive := false
	if ci, ok := request.Parameters["-i"].(bool); ok {
		caseInsensitive = ci
	}

	showLineNumbers := false
	if ln, ok := request.Parameters["-n"].(bool); ok {
		showLineNumbers = ln
	}

	globPattern := ""
	if glob, ok := request.Parameters["glob"].(string); ok {
		globPattern = glob
	}

	fileType := ""
	if ft, ok := request.Parameters["type"].(string); ok {
		fileType = ft
	}

	headLimit := 0
	if hl, ok := request.Parameters["head_limit"].(float64); ok {
		headLimit = int(hl)
	}

	// セキュリティ制約チェック（簡易実装）
	if strings.Contains(searchPath, "..") {
		return nil, NewToolError("security_violation", "Path contains invalid characters: "+searchPath)
	}

	// Grep検索を実行
	summary, err := t.performGrep(ctx, pattern, searchPath, &UnifiedGrepOptions{
		OutputMode:      outputMode,
		CaseInsensitive: caseInsensitive,
		ShowLineNumbers: showLineNumbers,
		GlobPattern:     globPattern,
		FileType:        fileType,
		HeadLimit:       headLimit,
	})
	if err != nil {
		return nil, NewToolError("execution_failed", fmt.Sprintf("Grep search failed: %v", err))
	}

	response := &ToolResponse{
		ID:       request.ID,
		ToolName: t.name,
		Success:  true,
		Data: map[string]interface{}{
			"summary":          summary,
			"pattern":          pattern,
			"search_path":      searchPath,
			"output_mode":      outputMode,
			"case_insensitive": caseInsensitive,
		},
	}

	return response, nil
}

// UnifiedGrepOptions - 統一Grep検索オプション
type UnifiedGrepOptions struct {
	OutputMode      string
	CaseInsensitive bool
	ShowLineNumbers bool
	GlobPattern     string
	FileType        string
	HeadLimit       int
}

// GetSchema - ツールスキーマを取得
func (t *UnifiedGrepTool) GetSchema() ToolSchema {
	return t.schema
}

// ValidateRequest - リクエストを検証
func (t *UnifiedGrepTool) ValidateRequest(request *ToolRequest) error {
	if err := t.BaseTool.ValidateRequest(request); err != nil {
		return err
	}

	pattern, ok := request.Parameters["pattern"].(string)
	if !ok {
		return NewToolError("invalid_parameter", "Pattern must be a string")
	}

	if strings.TrimSpace(pattern) == "" {
		return NewToolError("invalid_parameter", "Pattern cannot be empty")
	}

	// 正規表現パターンの検証
	if _, err := regexp.Compile(pattern); err != nil {
		return NewToolError("invalid_parameter", fmt.Sprintf("Invalid regex pattern: %v", err))
	}

	return nil
}

// Internal methods

// performGrep - Grep検索を実行
func (t *UnifiedGrepTool) performGrep(ctx context.Context, pattern, searchPath string, options *UnifiedGrepOptions) (*GrepSummary, error) {
	// 正規表現をコンパイル
	var regex *regexp.Regexp
	var err error

	if options.CaseInsensitive {
		regex, err = regexp.Compile("(?i)" + pattern)
	} else {
		regex, err = regexp.Compile(pattern)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to compile regex: %w", err)
	}

	summary := &GrepSummary{
		Pattern:    pattern,
		SearchPath: searchPath,
		Results:    []GrepResult{},
	}

	// ファイル一覧を取得
	files, err := t.getFilesToSearch(searchPath, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get files: %w", err)
	}

	summary.TotalFiles = len(files)
	filesWithMatches := make(map[string]bool)

	for _, filePath := range files {
		// コンテキストキャンセレーションをチェック
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		matches, err := t.searchInFile(filePath, regex, options)
		if err != nil {
			continue // エラーのあるファイルはスキップ
		}

		if len(matches) > 0 {
			filesWithMatches[filePath] = true
			summary.Results = append(summary.Results, matches...)
			summary.MatchCount += len(matches)

			// HeadLimitチェック
			if options.HeadLimit > 0 && len(summary.Results) >= options.HeadLimit {
				summary.Results = summary.Results[:options.HeadLimit]
				break
			}
		}
	}

	summary.FilesWithMatches = len(filesWithMatches)

	return summary, nil
}

// getFilesToSearch - 検索対象ファイル一覧を取得
func (t *UnifiedGrepTool) getFilesToSearch(searchPath string, options *UnifiedGrepOptions) ([]string, error) {
	var files []string

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // エラーは無視して続行
		}

		// ディレクトリはスキップ
		if info.IsDir() {
			return nil
		}

		// ファイルタイプフィルター
		if options.FileType != "" && !t.matchesFileType(path, options.FileType) {
			return nil
		}

		// Globパターンフィルター
		if options.GlobPattern != "" {
			matched, err := filepath.Match(options.GlobPattern, filepath.Base(path))
			if err != nil || !matched {
				return nil
			}
		}

		// バイナリファイルをスキップ
		if t.isBinaryFile(path) {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// searchInFile - ファイル内を検索
func (t *UnifiedGrepTool) searchInFile(filePath string, regex *regexp.Regexp, options *UnifiedGrepOptions) ([]GrepResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []GrepResult
	scanner := bufio.NewScanner(file)
	lineNumber := 1

	for scanner.Scan() {
		line := scanner.Text()

		// 正規表現マッチング
		matches := regex.FindAllStringIndex(line, -1)
		for _, match := range matches {
			result := GrepResult{
				File:        filePath,
				LineNumber:  lineNumber,
				Line:        line,
				Match:       line[match[0]:match[1]],
				ColumnStart: match[0],
				ColumnEnd:   match[1],
			}
			results = append(results, result)
		}

		lineNumber++
	}

	return results, scanner.Err()
}

// matchesFileType - ファイルタイプマッチング
func (t *UnifiedGrepTool) matchesFileType(filePath, fileType string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch fileType {
	case "go":
		return ext == ".go"
	case "js":
		return ext == ".js" || ext == ".mjs"
	case "ts":
		return ext == ".ts" || ext == ".tsx"
	case "py":
		return ext == ".py"
	case "java":
		return ext == ".java"
	case "cpp", "c++":
		return ext == ".cpp" || ext == ".cxx" || ext == ".cc"
	case "c":
		return ext == ".c" || ext == ".h"
	case "rust":
		return ext == ".rs"
	case "json":
		return ext == ".json"
	case "yaml", "yml":
		return ext == ".yaml" || ext == ".yml"
	case "md":
		return ext == ".md"
	default:
		return false
	}
}

// isBinaryFile - バイナリファイル判定（簡易）
func (t *UnifiedGrepTool) isBinaryFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	binaryExts := []string{
		".exe", ".bin", ".so", ".dll", ".dylib",
		".jpg", ".jpeg", ".png", ".gif", ".bmp",
		".mp3", ".mp4", ".avi", ".mkv",
		".zip", ".tar", ".gz", ".rar",
		".pdf", ".doc", ".docx", ".xls", ".xlsx",
	}

	for _, binExt := range binaryExts {
		if ext == binExt {
			return true
		}
	}

	return false
}
