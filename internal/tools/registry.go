package tools

import (
	"fmt"
	"strings"
	"sync"

	"github.com/glkt/vyb-code/internal/mcp"
	"github.com/glkt/vyb-code/internal/security"
)

// 統一ツールインターフェース
type UnifiedTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`       // "native" or "mcp"
	ServerName  string                 `json:"serverName"` // MCPツールの場合のサーバー名
	Schema      map[string]interface{} `json:"schema"`     // ツールスキーマ
	Handler     ToolHandler            `json:"-"`          // ネイティブツールハンドラー
}

// ツールハンドラー関数型
type ToolHandler func(arguments map[string]interface{}) (interface{}, error)

// ツール実行結果
type ToolExecutionResult struct {
	Content  string                 `json:"content"`
	IsError  bool                   `json:"isError"`
	Metadata map[string]interface{} `json:"metadata"`
	Tool     string                 `json:"tool"`
	Server   string                 `json:"server,omitempty"`
	Duration string                 `json:"duration"`
	ExitCode int                    `json:"exitCode,omitempty"`
}

// ツールレジストリ：ネイティブとMCPツールを統合管理
type ToolRegistry struct {
	mu          sync.RWMutex
	nativeTools map[string]UnifiedTool
	mcpManager  *mcp.Manager
	constraints *security.Constraints
	fileOps     *FileOperations
	executor    *CommandExecutor
	gitOps      *GitOperations
}

// 新しいツールレジストリを作成
func NewToolRegistry(constraints *security.Constraints, workDir string, maxFileSize int64, mcpManager *mcp.Manager) *ToolRegistry {
	registry := &ToolRegistry{
		nativeTools: make(map[string]UnifiedTool),
		mcpManager:  mcpManager,
		constraints: constraints,
		fileOps:     NewFileOperations(maxFileSize, workDir),
		executor:    NewCommandExecutor(constraints, workDir),
		gitOps:      NewGitOperations(constraints, workDir),
	}

	// ネイティブツールを登録
	registry.registerNativeTools()
	return registry
}

// ネイティブツールを登録
func (r *ToolRegistry) registerNativeTools() {
	// ファイル読み取りツール
	r.nativeTools["read_file"] = UnifiedTool{
		Name:        "read_file",
		Description: "ファイルの内容を読み取ります",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "読み取るファイルのパス",
				},
			},
			"required": []string{"file_path"},
		},
		Handler: r.handleReadFile,
	}

	// ファイル書き込みツール
	r.nativeTools["write_file"] = UnifiedTool{
		Name:        "write_file",
		Description: "ファイルに内容を書き込みます",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "書き込むファイルのパス",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "書き込む内容",
				},
			},
			"required": []string{"file_path", "content"},
		},
		Handler: r.handleWriteFile,
	}

	// コマンド実行ツール
	r.nativeTools["execute_command"] = UnifiedTool{
		Name:        "execute_command",
		Description: "シェルコマンドを安全に実行します",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "実行するコマンド",
				},
			},
			"required": []string{"command"},
		},
		Handler: r.handleExecuteCommand,
	}

	// Git操作ツール
	r.nativeTools["git_status"] = UnifiedTool{
		Name:        "git_status",
		Description: "Git状態を取得します",
		Type:        "native",
		Schema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: r.handleGitStatus,
	}
}

// ネイティブツールハンドラー実装
func (r *ToolRegistry) handleReadFile(arguments map[string]interface{}) (interface{}, error) {
	filePath, ok := arguments["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path引数が必要です")
	}

	content, err := r.fileOps.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content":   content,
		"file_path": filePath,
	}, nil
}

func (r *ToolRegistry) handleWriteFile(arguments map[string]interface{}) (interface{}, error) {
	filePath, ok := arguments["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path引数が必要です")
	}

	content, ok := arguments["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content引数が必要です")
	}

	if err := r.fileOps.WriteFile(filePath, content); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":    "success",
		"file_path": filePath,
		"size":      len(content),
	}, nil
}

func (r *ToolRegistry) handleExecuteCommand(arguments map[string]interface{}) (interface{}, error) {
	command, ok := arguments["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command引数が必要です")
	}

	result, err := r.executor.Execute(command)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
		"exit_code": result.ExitCode,
		"duration":  result.Duration,
		"timed_out": result.TimedOut,
	}, nil
}

func (r *ToolRegistry) handleGitStatus(arguments map[string]interface{}) (interface{}, error) {
	result, err := r.gitOps.GetStatus()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
		"exit_code": result.ExitCode,
		"clean":     result.Stdout == "",
	}, nil
}

// 利用可能な全ツールを取得
func (r *ToolRegistry) GetAllTools() map[string]UnifiedTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]UnifiedTool)

	// ネイティブツールを追加
	for name, tool := range r.nativeTools {
		result[name] = tool
	}

	// MCPツールを追加
	if r.mcpManager != nil {
		mcpTools := r.mcpManager.GetAllTools()
		for serverName, tools := range mcpTools {
			for _, tool := range tools {
				unifiedTool := UnifiedTool{
					Name:        fmt.Sprintf("%s.%s", serverName, tool.Name),
					Description: tool.Description,
					Type:        "mcp",
					ServerName:  serverName,
					Schema:      tool.InputSchema.(map[string]interface{}),
				}
				result[unifiedTool.Name] = unifiedTool
			}
		}
	}

	return result
}

// ツールを実行
func (r *ToolRegistry) ExecuteTool(toolName string, arguments map[string]interface{}) (*ToolExecutionResult, error) {
	r.mu.RLock()
	tool, isNative := r.nativeTools[toolName]
	r.mu.RUnlock()

	// ネイティブツールの場合
	if isNative {
		result, err := tool.Handler(arguments)
		if err != nil {
			return &ToolExecutionResult{
				Content: fmt.Sprintf("ツール実行エラー: %v", err),
				IsError: true,
				Tool:    toolName,
			}, err
		}

		return &ToolExecutionResult{
			Content:  fmt.Sprintf("ツール実行成功: %v", result),
			IsError:  false,
			Tool:     toolName,
			Metadata: result.(map[string]interface{}),
		}, nil
	}

	// MCPツールの場合
	parts := strings.Split(toolName, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("MCPツール名は 'server.tool' 形式である必要があります")
	}

	serverName, actualToolName := parts[0], parts[1]
	mcpResult, err := r.mcpManager.CallTool(serverName, actualToolName, arguments)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("MCPツール実行エラー: %v", err),
			IsError: true,
			Tool:    toolName,
			Server:  serverName,
		}, err
	}

	content := ""
	if len(mcpResult.Content) > 0 {
		content = mcpResult.Content[0].Text
	}

	return &ToolExecutionResult{
		Content: content,
		IsError: mcpResult.IsError,
		Tool:    toolName,
		Server:  serverName,
	}, nil
}
