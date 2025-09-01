package mcp

import (
	"fmt"
	"strings"

	"github.com/glkt/vyb-code/internal/security"
)

// MCPツールセキュリティバリデーター
type ToolSecurityValidator struct {
	constraints *security.Constraints
	whitelist   map[string]bool // 許可されたツール名
	blacklist   map[string]bool // 禁止されたツール名
}

// 新しいセキュリティバリデーターを作成
func NewToolSecurityValidator(constraints *security.Constraints) *ToolSecurityValidator {
	return &ToolSecurityValidator{
		constraints: constraints,
		whitelist:   make(map[string]bool),
		blacklist:   make(map[string]bool),
	}
}

// ツール実行を検証
func (v *ToolSecurityValidator) ValidateToolCall(toolName string, arguments map[string]interface{}) error {
	// ブラックリストチェック
	if v.blacklist[toolName] {
		return fmt.Errorf("ツール '%s' は禁止されています", toolName)
	}

	// ホワイトリストが設定されている場合はチェック
	if len(v.whitelist) > 0 && !v.whitelist[toolName] {
		return fmt.Errorf("ツール '%s' は許可されていません", toolName)
	}

	// 危険なツール名パターンをチェック
	if v.isDangerousTool(toolName) {
		return fmt.Errorf("危険なツール '%s' の実行は禁止されています", toolName)
	}

	// 引数の検証
	if err := v.validateArguments(toolName, arguments); err != nil {
		return fmt.Errorf("引数検証失敗: %w", err)
	}

	return nil
}

// 危険なツールかどうかを判定
func (v *ToolSecurityValidator) isDangerousTool(toolName string) bool {
	dangerousPatterns := []string{
		"exec", "shell", "system", "eval", "run",
		"delete", "remove", "rm", "kill", "destroy",
		"network", "http", "curl", "wget", "fetch",
		"admin", "sudo", "root", "privilege",
	}

	toolLower := strings.ToLower(toolName)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(toolLower, pattern) {
			return true
		}
	}

	return false
}

// 引数を検証
func (v *ToolSecurityValidator) validateArguments(toolName string, arguments map[string]interface{}) error {
	// コマンド系の引数チェック
	if command, exists := arguments["command"]; exists {
		if commandStr, ok := command.(string); ok {
			if err := v.constraints.IsCommandAllowed(commandStr); err != nil {
				return fmt.Errorf("コマンド '%s' は許可されていません: %w", commandStr, err)
			}
		}
	}

	// ファイルパス系の引数チェック
	for key, value := range arguments {
		if strings.Contains(strings.ToLower(key), "path") || strings.Contains(strings.ToLower(key), "file") {
			if pathStr, ok := value.(string); ok {
				if !v.constraints.IsPathAllowed(pathStr) {
					return fmt.Errorf("パス '%s' はワークスペース外です", pathStr)
				}
			}
		}
	}

	// URL系の引数チェック
	for key := range arguments {
		if strings.Contains(strings.ToLower(key), "url") || strings.Contains(strings.ToLower(key), "uri") {
			return fmt.Errorf("URLアクセスは許可されていません: %s", key)
		}
	}

	return nil
}

// ホワイトリストにツールを追加
func (v *ToolSecurityValidator) AddToWhitelist(toolName string) {
	v.whitelist[toolName] = true
}

// ブラックリストにツールを追加
func (v *ToolSecurityValidator) AddToBlacklist(toolName string) {
	v.blacklist[toolName] = true
}

// デフォルトのセキュリティ設定を適用
func (v *ToolSecurityValidator) ApplyDefaultPolicy() {
	// 安全なツールをホワイトリストに追加
	safeTools := []string{
		"read_file", "write_file", "list_files", "search_files",
		"git_status", "git_log", "git_diff", "git_branch",
		"analyze_code", "format_code", "lint_code",
	}

	for _, tool := range safeTools {
		v.AddToWhitelist(tool)
	}

	// 危険なツールをブラックリストに追加
	dangerousTools := []string{
		"system_exec", "shell_exec", "network_request",
		"file_delete", "admin_command", "privilege_escalation",
	}

	for _, tool := range dangerousTools {
		v.AddToBlacklist(tool)
	}
}
