package tools

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

// UnifiedReadTool - 統一読み取りツール
type UnifiedReadTool struct {
	*BaseTool
}

// NewUnifiedReadTool - 新しい統一読み取りツールを作成
func NewUnifiedReadTool(constraints *security.Constraints) *UnifiedReadTool {
	base := NewBaseTool("read", "ファイル内容を読み取ります", "1.0.0", CategoryFile)
	base.AddCapability(CapabilityFileRead)
	base.SetConstraints(constraints)
	
	// スキーマ設定
	schema := ToolSchema{
		Name:        "read",
		Description: "ファイル内容を読み取り、指定された範囲の行を返します",
		Version:     "1.0.0",
		Parameters: map[string]Parameter{
			"file_path": {
				Type:        "string",
				Description: "読み取るファイルのパス",
			},
			"offset": {
				Type:        "integer",
				Description: "読み取り開始行（省略可）",
				Minimum:     floatPtr(0),
			},
			"limit": {
				Type:        "integer", 
				Description: "読み取る行数（省略可）",
				Minimum:     floatPtr(1),
			},
		},
		Required: []string{"file_path"},
		Examples: []ToolExample{
			{
				Description: "ファイル全体を読み取り",
				Parameters: map[string]interface{}{
					"file_path": "./example.go",
				},
			},
			{
				Description: "特定の行範囲を読み取り",
				Parameters: map[string]interface{}{
					"file_path": "./example.go",
					"offset":    10,
					"limit":     20,
				},
			},
		},
	}
	base.SetSchema(schema)
	
	return &UnifiedReadTool{BaseTool: base}
}

// Execute - 読み取り実行
func (t *UnifiedReadTool) Execute(ctx context.Context, request *ToolRequest) (*ToolResponse, error) {
	filePath := request.Parameters["file_path"].(string)
	
	// セキュリティチェック（簡易実装）
	if t.constraints != nil {
		// 基本的なパス検証
		if strings.Contains(filePath, "..") || !filepath.IsAbs(filePath) && !strings.HasPrefix(filePath, "./") {
			return nil, NewExecutionError("Invalid file path: "+filePath, -1)
		}
	}
	
	// ファイル読み取り
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, NewExecutionError("Failed to read file: "+err.Error(), -1)
	}
	
	lines := strings.Split(string(content), "\n")
	
	// オフセットと制限の適用
	offset := 0
	limit := len(lines)
	
	if offsetVal, ok := request.Parameters["offset"]; ok {
		if o, ok := offsetVal.(float64); ok {
			offset = int(o)
		}
	}
	
	if limitVal, ok := request.Parameters["limit"]; ok {
		if l, ok := limitVal.(float64); ok {
			limit = int(l)
		}
	}
	
	// 範囲調整
	if offset >= len(lines) {
		lines = []string{}
	} else {
		end := offset + limit
		if end > len(lines) {
			end = len(lines)
		}
		lines = lines[offset:end]
	}
	
	// 行番号付きで結果作成
	var result strings.Builder
	for i, line := range lines {
		result.WriteString(fmt.Sprintf("%5d→%s\n", offset+i+1, line))
	}
	
	return &ToolResponse{
		ID:       request.ID,
		ToolName: t.GetName(),
		Success:  true,
		Content:  result.String(),
		Metadata: &ResponseMetadata{
			Debug: map[string]interface{}{
				"file_path":   filePath,
				"total_lines": len(strings.Split(string(content), "\n")),
				"offset":      offset,
				"limit":       limit,
				"returned":    len(lines),
			},
		},
	}, nil
}

// UnifiedWriteTool - 統一書き込みツール
type UnifiedWriteTool struct {
	*BaseTool
}

// NewUnifiedWriteTool - 新しい統一書き込みツールを作成
func NewUnifiedWriteTool(constraints *security.Constraints) *UnifiedWriteTool {
	base := NewBaseTool("write", "ファイルに内容を書き込みます", "1.0.0", CategoryFile)
	base.AddCapability(CapabilityFileWrite)
	base.SetConstraints(constraints)
	
	schema := ToolSchema{
		Name:        "write",
		Description: "指定されたファイルに内容を書き込みます",
		Version:     "1.0.0",
		Parameters: map[string]Parameter{
			"file_path": {
				Type:        "string",
				Description: "書き込むファイルのパス",
			},
			"content": {
				Type:        "string",
				Description: "書き込む内容",
			},
		},
		Required: []string{"file_path", "content"},
		Examples: []ToolExample{
			{
				Description: "新しいファイルに内容を書き込み",
				Parameters: map[string]interface{}{
					"file_path": "./new_file.txt",
					"content":   "Hello, World!",
				},
			},
		},
	}
	base.SetSchema(schema)
	
	return &UnifiedWriteTool{BaseTool: base}
}

// Execute - 書き込み実行
func (t *UnifiedWriteTool) Execute(ctx context.Context, request *ToolRequest) (*ToolResponse, error) {
	filePath := request.Parameters["file_path"].(string)
	content := request.Parameters["content"].(string)
	
	// セキュリティチェック（簡易実装）
	if t.constraints != nil {
		// 基本的なパス検証
		if strings.Contains(filePath, "..") || !filepath.IsAbs(filePath) && !strings.HasPrefix(filePath, "./") {
			return nil, NewExecutionError("Invalid file path: "+filePath, -1)
		}
	}
	
	// ディレクトリ作成
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, NewExecutionError("Failed to create directory: "+err.Error(), -1)
	}
	
	// ファイル書き込み
	err := ioutil.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return nil, NewExecutionError("Failed to write file: "+err.Error(), -1)
	}
	
	return &ToolResponse{
		ID:       request.ID,
		ToolName: t.GetName(),
		Success:  true,
		Content:  fmt.Sprintf("File written successfully: %s", filePath),
		Metadata: &ResponseMetadata{
			Debug: map[string]interface{}{
				"file_path":    filePath,
				"content_size": len(content),
			},
		},
	}, nil
}

// UnifiedEditTool - 統一編集ツール
type UnifiedEditTool struct {
	*BaseTool
}

// NewUnifiedEditTool - 新しい統一編集ツールを作成
func NewUnifiedEditTool(constraints *security.Constraints) *UnifiedEditTool {
	base := NewBaseTool("edit", "ファイルの部分編集を行います", "1.0.0", CategoryFile)
	base.AddCapability(CapabilityFileRead)
	base.AddCapability(CapabilityFileWrite)
	base.AddCapability(CapabilityFileEdit)
	base.SetConstraints(constraints)
	
	schema := ToolSchema{
		Name:        "edit",
		Description: "ファイル内の文字列を検索・置換します",
		Version:     "1.0.0",
		Parameters: map[string]Parameter{
			"file_path": {
				Type:        "string",
				Description: "編集するファイルのパス",
			},
			"old_string": {
				Type:        "string",
				Description: "置換対象の文字列",
			},
			"new_string": {
				Type:        "string",
				Description: "置換後の文字列",
			},
			"replace_all": {
				Type:        "boolean",
				Description: "全ての出現箇所を置換するか（デフォルト：false）",
				Default:     false,
			},
		},
		Required: []string{"file_path", "old_string", "new_string"},
		Examples: []ToolExample{
			{
				Description: "最初の出現箇所のみ置換",
				Parameters: map[string]interface{}{
					"file_path":   "./example.go",
					"old_string":  "oldFunction",
					"new_string":  "newFunction",
					"replace_all": false,
				},
			},
			{
				Description: "全ての出現箇所を置換",
				Parameters: map[string]interface{}{
					"file_path":   "./example.go",
					"old_string":  "oldVariable",
					"new_string":  "newVariable",
					"replace_all": true,
				},
			},
		},
	}
	base.SetSchema(schema)
	
	return &UnifiedEditTool{BaseTool: base}
}

// Execute - 編集実行
func (t *UnifiedEditTool) Execute(ctx context.Context, request *ToolRequest) (*ToolResponse, error) {
	filePath := request.Parameters["file_path"].(string)
	oldString := request.Parameters["old_string"].(string)
	newString := request.Parameters["new_string"].(string)
	
	replaceAll := false
	if replaceAllVal, ok := request.Parameters["replace_all"]; ok {
		if ra, ok := replaceAllVal.(bool); ok {
			replaceAll = ra
		}
	}
	
	// セキュリティチェック（簡易実装）
	if t.constraints != nil {
		// 基本的なパス検証
		if strings.Contains(filePath, "..") || !filepath.IsAbs(filePath) && !strings.HasPrefix(filePath, "./") {
			return nil, NewExecutionError("Invalid file path: "+filePath, -1)
		}
	}
	
	// ファイル読み取り
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, NewExecutionError("Failed to read file: "+err.Error(), -1)
	}
	
	originalContent := string(content)
	var newContent string
	var replacements int
	
	if replaceAll {
		newContent = strings.ReplaceAll(originalContent, oldString, newString)
		replacements = strings.Count(originalContent, oldString)
	} else {
		// 最初の出現のみ置換
		if strings.Contains(originalContent, oldString) {
			newContent = strings.Replace(originalContent, oldString, newString, 1)
			replacements = 1
		} else {
			newContent = originalContent
			replacements = 0
		}
	}
	
	if replacements == 0 {
		return nil, NewExecutionError("Target string not found in file", -1)
	}
	
	// ファイル書き込み
	err = ioutil.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		return nil, NewExecutionError("Failed to write file: "+err.Error(), -1)
	}
	
	return &ToolResponse{
		ID:       request.ID,
		ToolName: t.GetName(),
		Success:  true,
		Content:  fmt.Sprintf("Successfully replaced %d occurrence(s) in %s", replacements, filePath),
		Metadata: &ResponseMetadata{
			Debug: map[string]interface{}{
				"file_path":     filePath,
				"old_string":    oldString,
				"new_string":    newString,
				"replace_all":   replaceAll,
				"replacements":  replacements,
				"original_size": len(originalContent),
				"new_size":      len(newContent),
			},
		},
	}, nil
}

// UnifiedBashTool - 統一Bashツール
type UnifiedBashTool struct {
	*BaseTool
	timeout time.Duration
}

// NewUnifiedBashTool - 新しい統一Bashツールを作成
func NewUnifiedBashTool(constraints *security.Constraints) *UnifiedBashTool {
	base := NewBaseTool("bash", "シェルコマンドを安全に実行します", "1.0.0", CategoryCommand)
	base.AddCapability(CapabilityCommand)
	base.SetConstraints(constraints)
	
	schema := ToolSchema{
		Name:        "bash",
		Description: "シェルコマンドを実行し、結果を返します",
		Version:     "1.0.0",
		Parameters: map[string]Parameter{
			"command": {
				Type:        "string",
				Description: "実行するコマンド",
			},
			"description": {
				Type:        "string",
				Description: "コマンドの説明（省略可）",
			},
			"timeout": {
				Type:        "integer",
				Description: "タイムアウト（ミリ秒、省略可、デフォルト：30000）",
				Default:     30000,
				Minimum:     floatPtr(1000),
				Maximum:     floatPtr(600000), // 最大10分
			},
		},
		Required: []string{"command"},
		Examples: []ToolExample{
			{
				Description: "ディレクトリ内容を一覧表示",
				Parameters: map[string]interface{}{
					"command":     "ls -la",
					"description": "List directory contents",
				},
			},
			{
				Description: "Git状態を確認",
				Parameters: map[string]interface{}{
					"command":     "git status",
					"description": "Check git status",
					"timeout":     10000,
				},
			},
		},
	}
	base.SetSchema(schema)
	
	return &UnifiedBashTool{
		BaseTool: base,
		timeout:  30 * time.Second,
	}
}

// Execute - コマンド実行
func (t *UnifiedBashTool) Execute(ctx context.Context, request *ToolRequest) (*ToolResponse, error) {
	command := request.Parameters["command"].(string)
	
	description := ""
	if desc, ok := request.Parameters["description"].(string); ok {
		description = desc
	}
	
	timeout := t.timeout
	if timeoutVal, ok := request.Parameters["timeout"]; ok {
		if timeoutMs, ok := timeoutVal.(float64); ok {
			timeout = time.Duration(timeoutMs) * time.Millisecond
		}
	}
	
	// セキュリティチェック
	if t.constraints != nil {
		if err := t.constraints.ValidateCommand(command); err != nil {
			return nil, NewExecutionError("Command validation failed: "+err.Error(), -1)
		}
	}
	
	// コンテキストでタイムアウト設定
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// コマンド実行
	cmd := exec.CommandContext(cmdCtx, "bash", "-c", command)
	
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)
	
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}
	
	// タイムアウト判定
	timedOut := cmdCtx.Err() == context.DeadlineExceeded
	
	success := exitCode == 0 && !timedOut
	errorMsg := ""
	if err != nil && !timedOut {
		errorMsg = err.Error()
	} else if timedOut {
		errorMsg = "Command execution timed out"
	}
	
	return &ToolResponse{
		ID:       request.ID,
		ToolName: t.GetName(),
		Success:  success,
		Content:  string(output),
		Error:    errorMsg,
		Metadata: &ResponseMetadata{
			ExecutionTime: duration,
			Debug: map[string]interface{}{
				"command":     command,
				"description": description,
				"exit_code":   exitCode,
				"timed_out":   timedOut,
				"timeout_ms":  timeout.Milliseconds(),
			},
		},
	}, nil
}

// 他のツール実装の宣言（実装は省略）

func NewUnifiedGlobTool() *BaseTool {
	base := NewBaseTool("glob", "ファイルパターンマッチング", "1.0.0", CategorySearch)
	base.AddCapability(CapabilitySearch)
	// TODO: 実装
	return base
}

func NewUnifiedGrepTool() *BaseTool {
	base := NewBaseTool("grep", "高度なファイル検索", "1.0.0", CategorySearch)
	base.AddCapability(CapabilitySearch)
	// TODO: 実装
	return base
}

func NewUnifiedLSTool() *BaseTool {
	base := NewBaseTool("ls", "ディレクトリリスト表示", "1.0.0", CategoryFile)
	base.AddCapability(CapabilityFileRead)
	// TODO: 実装
	return base
}

func NewUnifiedWebFetchTool() *BaseTool {
	base := NewBaseTool("webfetch", "Web内容取得", "1.0.0", CategoryWeb)
	base.AddCapability(CapabilityNetwork)
	// TODO: 実装
	return base
}

func NewUnifiedWebSearchTool() *BaseTool {
	base := NewBaseTool("websearch", "Web検索", "1.0.0", CategoryWeb)
	base.AddCapability(CapabilityNetwork)
	base.AddCapability(CapabilitySearch)
	// TODO: 実装
	return base
}

// ヘルパー関数
func floatPtr(f float64) *float64 {
	return &f
}