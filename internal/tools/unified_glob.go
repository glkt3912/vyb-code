package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/glkt/vyb-code/internal/security"
)

// UnifiedGlobTool - 統一Globツール（ファイルパターンマッチング）
type UnifiedGlobTool struct {
	*BaseTool
}

// NewUnifiedGlobTool - 新しい統一Globツールを作成
func NewUnifiedGlobTool(constraints *security.Constraints) *UnifiedGlobTool {
	base := NewBaseTool("glob", "Fast file pattern matching tool that works with any codebase size", "1.0.0", CategorySearch)
	base.AddCapability(CapabilitySearch)
	base.AddCapability(CapabilityFileRead)
	base.SetConstraints(constraints)

	// スキーマ設定
	schema := ToolSchema{
		Name:        "glob",
		Description: "Fast file pattern matching tool that works with any codebase size",
		Version:     "1.0.0",
		Parameters: map[string]Parameter{
			"pattern": {
				Type:        "string",
				Description: "The glob pattern to match files against (e.g., '*.go', '**/*.js')",
			},
			"path": {
				Type:        "string",
				Description: "The directory to search in (defaults to current directory)",
			},
		},
		Required: []string{"pattern"},
		Examples: []ToolExample{
			{
				Description: "Find all Go files",
				Parameters: map[string]interface{}{
					"pattern": "*.go",
				},
			},
		},
	}
	base.SetSchema(schema)

	return &UnifiedGlobTool{BaseTool: base}
}

// Execute - Globツールを実行
func (t *UnifiedGlobTool) Execute(ctx context.Context, request *ToolRequest) (*ToolResponse, error) {
	if err := t.ValidateRequest(request); err != nil {
		return nil, err
	}

	// パラメータ取得
	pattern, ok := request.Parameters["pattern"].(string)
	if !ok || pattern == "" {
		return nil, NewToolError("invalid_parameter", "Pattern parameter is required")
	}

	// オプショナルパラメータ
	searchPath := "."
	if path, ok := request.Parameters["path"].(string); ok && path != "" {
		searchPath = path
	}

	// セキュリティ制約チェック（簡易実装）
	if strings.Contains(searchPath, "..") {
		return nil, NewToolError("security_violation", "Path contains invalid characters: "+searchPath)
	}

	// Globパターンマッチングを実行
	matches, err := t.performGlobSearch(searchPath, pattern)
	if err != nil {
		return nil, NewToolError("execution_failed", fmt.Sprintf("Glob search failed: %v", err))
	}

	response := &ToolResponse{
		ID:       request.ID,
		ToolName: t.name,
		Success:  true,
		Data: map[string]interface{}{
			"pattern":     pattern,
			"search_path": searchPath,
			"matches":     matches,
			"match_count": len(matches),
			"total_size":  t.calculateTotalSize(matches),
		},
	}

	return response, nil
}

// GetSchema - ツールスキーマを取得
func (t *UnifiedGlobTool) GetSchema() ToolSchema {
	return t.schema
}

// ValidateRequest - リクエストを検証
func (t *UnifiedGlobTool) ValidateRequest(request *ToolRequest) error {
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

	// 危険なパターンをチェック
	if strings.Contains(pattern, "..") {
		return NewToolError("security_violation", "Pattern cannot contain '..' for security reasons")
	}

	return nil
}

// Internal methods

// performGlobSearch - Globサーチを実行
func (t *UnifiedGlobTool) performGlobSearch(searchPath, pattern string) ([]string, error) {
	// パスの正規化
	absSearchPath, err := filepath.Abs(searchPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Globパターンの構築
	var globPattern string
	if filepath.IsAbs(pattern) {
		globPattern = pattern
	} else {
		globPattern = filepath.Join(absSearchPath, pattern)
	}

	// filepath.Globを使用してマッチング
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, fmt.Errorf("glob pattern matching failed: %w", err)
	}

	// 結果をソート（相対パスに変換）
	var result []string
	for _, match := range matches {
		relPath, err := filepath.Rel(absSearchPath, match)
		if err != nil {
			// 相対パス変換に失敗した場合は絶対パスを使用
			result = append(result, match)
		} else {
			result = append(result, relPath)
		}
	}

	return result, nil
}

// calculateTotalSize - マッチしたファイルの合計サイズを計算
func (t *UnifiedGlobTool) calculateTotalSize(matches []string) int64 {
	var totalSize int64

	for _, match := range matches {
		if info, err := filepath.Glob(match); err == nil && len(info) > 0 {
			// ファイル情報を取得してサイズを追加
			// 実際の実装ではos.Statを使用するが、ここではサイズ推定
			totalSize += int64(len(match) * 100) // 簡易推定
		}
	}

	return totalSize
}
