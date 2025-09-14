package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/glkt/vyb-code/internal/security"
)

// UnifiedLSTool - 統一LSツール（ディレクトリ一覧）
type UnifiedLSTool struct {
	*BaseTool
}

// LSEntry - ディレクトリエントリ
type LSEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"is_dir"`
	Size     int64  `json:"size"`
	ModTime  string `json:"mod_time"`
	Mode     string `json:"mode"`
	IsHidden bool   `json:"is_hidden"`
}

// LSResult - LS実行結果
type LSResult struct {
	Path        string    `json:"path"`
	Entries     []LSEntry `json:"entries"`
	TotalCount  int       `json:"total_count"`
	DirCount    int       `json:"dir_count"`
	FileCount   int       `json:"file_count"`
	HiddenCount int       `json:"hidden_count"`
}

// NewUnifiedLSTool - 新しい統一LSツールを作成
func NewUnifiedLSTool(constraints *security.Constraints) *UnifiedLSTool {
	base := NewBaseTool("ls", "Lists files and directories in a given path with optional filtering", "1.0.0", CategoryFile)
	base.AddCapability(CapabilityFileRead)
	base.SetConstraints(constraints)

	// スキーマ設定
	schema := ToolSchema{
		Name:        "ls",
		Description: "Lists files and directories in a given path with optional filtering",
		Version:     "1.0.0",
		Parameters: map[string]Parameter{
			"path": {
				Type:        "string",
				Description: "The absolute path to the directory to list (must be absolute, not relative)",
			},
			"ignore": {
				Type:        "array",
				Description: "List of glob patterns to ignore",
			},
			"show_hidden": {
				Type:        "boolean",
				Description: "Show hidden files and directories (starting with .)",
			},
			"recursive": {
				Type:        "boolean",
				Description: "List directories recursively",
			},
		},
		Required: []string{"path"},
		Examples: []ToolExample{
			{
				Description: "List current directory contents",
				Parameters: map[string]interface{}{
					"path": ".",
				},
			},
			{
				Description: "List with ignore patterns",
				Parameters: map[string]interface{}{
					"path":   "./src",
					"ignore": []string{"*.log", "node_modules"},
				},
			},
		},
	}
	base.SetSchema(schema)

	return &UnifiedLSTool{BaseTool: base}
}

// Execute - LSツールを実行
func (t *UnifiedLSTool) Execute(ctx context.Context, request *ToolRequest) (*ToolResponse, error) {
	if err := t.ValidateRequest(request); err != nil {
		return nil, err
	}

	// パラメータ取得
	path, ok := request.Parameters["path"].(string)
	if !ok || path == "" {
		return nil, NewToolError("invalid_parameter", "Path parameter is required")
	}

	// オプション取得
	showHidden := false
	if sh, ok := request.Parameters["show_hidden"].(bool); ok {
		showHidden = sh
	}

	recursive := false
	if rec, ok := request.Parameters["recursive"].(bool); ok {
		recursive = rec
	}

	var ignorePatterns []string
	if ignore, ok := request.Parameters["ignore"]; ok {
		if patterns, ok := ignore.([]interface{}); ok {
			for _, p := range patterns {
				if str, ok := p.(string); ok {
					ignorePatterns = append(ignorePatterns, str)
				}
			}
		}
	}

	// セキュリティ制約チェック（簡易実装）
	if strings.Contains(path, "..") {
		return nil, NewToolError("security_violation", "Path contains invalid characters: "+path)
	}

	// ディレクトリリストを実行
	result, err := t.performLS(path, showHidden, recursive, ignorePatterns)
	if err != nil {
		return nil, NewToolError("execution_failed", fmt.Sprintf("Directory listing failed: %v", err))
	}

	response := &ToolResponse{
		ID:       request.ID,
		ToolName: t.name,
		Success:  true,
		Data: map[string]interface{}{
			"result":          result,
			"path":            path,
			"show_hidden":     showHidden,
			"recursive":       recursive,
			"ignore_patterns": ignorePatterns,
		},
	}

	return response, nil
}

// GetSchema - ツールスキーマを取得
func (t *UnifiedLSTool) GetSchema() ToolSchema {
	return t.schema
}

// ValidateRequest - リクエストを検証
func (t *UnifiedLSTool) ValidateRequest(request *ToolRequest) error {
	if err := t.BaseTool.ValidateRequest(request); err != nil {
		return err
	}

	path, ok := request.Parameters["path"].(string)
	if !ok {
		return NewToolError("invalid_parameter", "Path must be a string")
	}

	if strings.TrimSpace(path) == "" {
		return NewToolError("invalid_parameter", "Path cannot be empty")
	}

	return nil
}

// Internal methods

// performLS - ディレクトリリストを実行
func (t *UnifiedLSTool) performLS(path string, showHidden, recursive bool, ignorePatterns []string) (*LSResult, error) {
	// パスの正規化
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// ディレクトリの存在確認
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access path: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", absPath)
	}

	result := &LSResult{
		Path:    absPath,
		Entries: []LSEntry{},
	}

	if recursive {
		err = t.walkRecursive(absPath, showHidden, ignorePatterns, result)
	} else {
		err = t.listSingle(absPath, showHidden, ignorePatterns, result)
	}

	if err != nil {
		return nil, err
	}

	// 統計を計算
	t.calculateStats(result)

	// エントリをソート（ディレクトリ優先、名前順）
	sort.Slice(result.Entries, func(i, j int) bool {
		if result.Entries[i].IsDir != result.Entries[j].IsDir {
			return result.Entries[i].IsDir
		}
		return result.Entries[i].Name < result.Entries[j].Name
	})

	return result, nil
}

// listSingle - 単一ディレクトリをリスト
func (t *UnifiedLSTool) listSingle(dirPath string, showHidden bool, ignorePatterns []string, result *LSResult) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(dirPath, name)

		// 隠しファイル/ディレクトリのフィルタリング
		if !showHidden && t.isHidden(name) {
			continue
		}

		// 無視パターンチェック
		if t.shouldIgnore(name, ignorePatterns) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue // エラーのあるエントリはスキップ
		}

		lsEntry := LSEntry{
			Name:     name,
			Path:     fullPath,
			IsDir:    entry.IsDir(),
			Size:     info.Size(),
			ModTime:  info.ModTime().Format("2006-01-02 15:04:05"),
			Mode:     info.Mode().String(),
			IsHidden: t.isHidden(name),
		}

		result.Entries = append(result.Entries, lsEntry)
	}

	return nil
}

// walkRecursive - 再帰的にディレクトリをウォーク
func (t *UnifiedLSTool) walkRecursive(rootPath string, showHidden bool, ignorePatterns []string, result *LSResult) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // エラーのあるパスはスキップ
		}

		// ルートディレクトリ自体はスキップ
		if path == rootPath {
			return nil
		}

		name := filepath.Base(path)

		// 隠しファイル/ディレクトリのフィルタリング
		if !showHidden && t.isHidden(name) {
			if info.IsDir() {
				return filepath.SkipDir // 隠しディレクトリは中身もスキップ
			}
			return nil
		}

		// 無視パターンチェック
		if t.shouldIgnore(name, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		lsEntry := LSEntry{
			Name:     name,
			Path:     path,
			IsDir:    info.IsDir(),
			Size:     info.Size(),
			ModTime:  info.ModTime().Format("2006-01-02 15:04:05"),
			Mode:     info.Mode().String(),
			IsHidden: t.isHidden(name),
		}

		result.Entries = append(result.Entries, lsEntry)
		return nil
	})
}

// isHidden - 隠しファイル/ディレクトリ判定
func (t *UnifiedLSTool) isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

// shouldIgnore - 無視パターンマッチング
func (t *UnifiedLSTool) shouldIgnore(name string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// calculateStats - 統計を計算
func (t *UnifiedLSTool) calculateStats(result *LSResult) {
	result.TotalCount = len(result.Entries)
	result.DirCount = 0
	result.FileCount = 0
	result.HiddenCount = 0

	for _, entry := range result.Entries {
		if entry.IsDir {
			result.DirCount++
		} else {
			result.FileCount++
		}

		if entry.IsHidden {
			result.HiddenCount++
		}
	}
}
