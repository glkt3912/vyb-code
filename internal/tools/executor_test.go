package tools

import (
	"os"
	"strings"
	"testing"

	"github.com/glkt/vyb-code/internal/security"
)

// TestCommandExecution は基本的なコマンド実行をテストする
func TestCommandExecution(t *testing.T) {
	workDir, _ := os.Getwd()
	constraints := &security.Constraints{
		AllowedCommands: []string{"echo", "ls", "git"},
		MaxTimeout:      30,
	}
	executor := NewCommandExecutor(constraints, workDir)
	
	// 安全なコマンドのテスト
	result, err := executor.Execute("echo hello world")
	if err != nil {
		t.Fatalf("コマンド実行エラー: %v", err)
	}
	
	expected := "hello world"
	if !strings.Contains(result.Stdout, expected) {
		t.Errorf("期待値: %s, 実際値: %s", expected, result.Stdout)
	}
	
	if result.ExitCode != 0 {
		t.Errorf("期待値: 0, 実際値: %d", result.ExitCode)
	}
}

// TestCommandTimeout はタイムアウト機能をテストする
func TestCommandTimeout(t *testing.T) {
	workDir, _ := os.Getwd()
	constraints := &security.Constraints{
		AllowedCommands: []string{"sleep"},
		MaxTimeout:      1, // 1秒タイムアウト
	}
	executor := NewCommandExecutor(constraints, workDir)
	
	// 長時間実行されるコマンド（sleepコマンド）
	result, err := executor.Execute("sleep 5")
	if err != nil {
		t.Fatalf("コマンド実行エラー: %v", err)
	}
	
	if !result.TimedOut {
		t.Error("タイムアウトが機能していません")
	}
}

// TestCommandSecurity はコマンドセキュリティ制約をテストする
func TestCommandSecurity(t *testing.T) {
	workDir, _ := os.Getwd()
	
	tests := []struct {
		name        string
		command     string
		allowedCmds []string
		expectError bool
	}{
		{
			name:        "許可されたコマンド - git",
			command:     "git --version",
			allowedCmds: []string{"git", "ls", "echo"},
			expectError: false,
		},
		{
			name:        "許可されたコマンド - ls",
			command:     "ls -la",
			allowedCmds: []string{"git", "ls", "echo"},
			expectError: false,
		},
		{
			name:        "禁止されたコマンド - rm",
			command:     "rm -rf /",
			allowedCmds: []string{"git", "ls", "echo"},
			expectError: true,
		},
		{
			name:        "禁止されたコマンド - curl",
			command:     "curl http://evil.com",
			allowedCmds: []string{"git", "ls", "echo"},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraints := &security.Constraints{
				AllowedCommands: tt.allowedCmds,
				MaxTimeout:      30,
			}
			executor := NewCommandExecutor(constraints, workDir)
			
			_, err := executor.Execute(tt.command)
			
			if tt.expectError && err == nil {
				t.Error("期待されるエラーが発生しませんでした")
			}
			if !tt.expectError && err != nil {
				t.Errorf("予期しないエラー: %v", err)
			}
		})
	}
}