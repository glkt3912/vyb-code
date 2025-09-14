package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

func TestUnifiedToolRegistry_RegisterAndExecute(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "unified-tools-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// テスト用の制約を作成
	constraints := security.NewDefaultConstraints(tempDir)
	registry := NewUnifiedToolRegistry(constraints, nil)

	t.Run("List registered tools", func(t *testing.T) {
		tools := registry.ListTools()
		if len(tools) < 3 {
			t.Errorf("Expected at least 3 tools, got %d", len(tools))
		}

		// 基本ツールが登録されているか確認
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.GetName()] = true
		}

		expectedTools := []string{"read", "write", "edit", "bash"}
		for _, expected := range expectedTools {
			if !toolNames[expected] {
				t.Errorf("Expected tool '%s' not found", expected)
			}
		}
	})

	t.Run("Get tool by name", func(t *testing.T) {
		readTool, err := registry.GetTool("read")
		if err != nil {
			t.Errorf("Failed to get read tool: %v", err)
		}

		if readTool.GetName() != "read" {
			t.Errorf("Tool name mismatch: got %s, want read", readTool.GetName())
		}

		if readTool.GetDescription() == "" {
			t.Error("Tool description is empty")
		}
	})

	t.Run("Get non-existent tool", func(t *testing.T) {
		_, err := registry.GetTool("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent tool")
		}
	})
}

func TestUnifiedWriteTool_Execute(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "unified-write-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	constraints := security.NewDefaultConstraints(tempDir)
	tool := NewUnifiedWriteTool(constraints)

	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"

	request := &ToolRequest{
		ID:       "test-1",
		ToolName: "write",
		Parameters: map[string]interface{}{
			"file_path": testFile,
			"content":   testContent,
		},
	}

	response, err := tool.Execute(context.Background(), request)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if !response.Success {
		t.Errorf("Execute was not successful: %s", response.Error)
	}

	// ファイルが作成されたか確認
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Failed to read written file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("File content mismatch: got %s, want %s", string(content), testContent)
	}
}

func TestUnifiedReadTool_Execute(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "unified-read-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// テストファイルを作成
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	constraints := security.NewDefaultConstraints(tempDir)
	tool := NewUnifiedReadTool(constraints)

	t.Run("Read entire file", func(t *testing.T) {
		request := &ToolRequest{
			ID:       "test-1",
			ToolName: "read",
			Parameters: map[string]interface{}{
				"file_path": testFile,
			},
		}

		response, err := tool.Execute(context.Background(), request)
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}

		if !response.Success {
			t.Errorf("Execute was not successful: %s", response.Error)
		}

		if response.Content == "" {
			t.Error("Response content is empty")
		}

		// 行番号付きの形式であることを確認
		if !contains(response.Content, "1→Line 1") {
			t.Error("Expected line numbers in output")
		}
	})

	t.Run("Read with offset and limit", func(t *testing.T) {
		request := &ToolRequest{
			ID:       "test-2",
			ToolName: "read",
			Parameters: map[string]interface{}{
				"file_path": testFile,
				"offset":    float64(1), // 2行目から
				"limit":     float64(2), // 2行分
			},
		}

		response, err := tool.Execute(context.Background(), request)
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}

		if !response.Success {
			t.Errorf("Execute was not successful: %s", response.Error)
		}

		// 2行目と3行目が含まれているか確認
		if !contains(response.Content, "2→Line 2") || !contains(response.Content, "3→Line 3") {
			t.Errorf("Expected lines 2-3, got: %s", response.Content)
		}

		// 1行目と4行目は含まれていないか確認
		if contains(response.Content, "1→Line 1") || contains(response.Content, "4→Line 4") {
			t.Errorf("Unexpected lines in output: %s", response.Content)
		}
	})
}

func TestUnifiedEditTool_Execute(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "unified-edit-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// テストファイルを作成
	testFile := filepath.Join(tempDir, "test.go")
	originalContent := `package main

import "fmt"

func oldFunction() {
	fmt.Println("old message")
}

func anotherOldFunction() {
	fmt.Println("another old message")
}

func main() {
	oldFunction()
}`

	err = os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	constraints := security.NewDefaultConstraints(tempDir)
	tool := NewUnifiedEditTool(constraints)

	t.Run("Replace first occurrence", func(t *testing.T) {
		request := &ToolRequest{
			ID:       "test-1",
			ToolName: "edit",
			Parameters: map[string]interface{}{
				"file_path":   testFile,
				"old_string":  "oldFunction",
				"new_string":  "newFunction",
				"replace_all": false,
			},
		}

		response, err := tool.Execute(context.Background(), request)
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}

		if !response.Success {
			t.Errorf("Execute was not successful: %s", response.Error)
		}

		// ファイル内容を確認
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		contentStr := string(content)

		// 最初の出現のみ置換されていることを確認
		if !contains(contentStr, "func newFunction()") {
			t.Error("First occurrence was not replaced")
		}

		if !contains(contentStr, "func anotherOldFunction()") {
			t.Error("Second occurrence should not be replaced")
		}
	})

	t.Run("Replace all occurrences", func(t *testing.T) {
		// ファイルをリセット
		err = os.WriteFile(testFile, []byte(originalContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		request := &ToolRequest{
			ID:       "test-2",
			ToolName: "edit",
			Parameters: map[string]interface{}{
				"file_path":   testFile,
				"old_string":  "old",
				"new_string":  "new",
				"replace_all": true,
			},
		}

		response, err := tool.Execute(context.Background(), request)
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}

		if !response.Success {
			t.Errorf("Execute was not successful: %s", response.Error)
		}

		// ファイル内容を確認
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		contentStr := string(content)

		// 全ての "old" が "new" に置換されていることを確認（小文字の"old"のみ）
		if contains(contentStr, " old ") || contains(contentStr, "\"old ") {
			t.Error("Some occurrences of 'old' were not replaced")
		}

		// 正確な置換確認：小文字の"old"のみが置換される
		if !contains(contentStr, "func newFunction()") {
			t.Error("First oldFunction was not replaced correctly")
		}

		// "anotherOldFunction"は"Old"（大文字）なので変更されない
		if !contains(contentStr, "func anotherOldFunction()") {
			t.Error("anotherOldFunction should remain unchanged (capital 'Old')")
		}

		// ただし、メッセージ内の"old"は置換される
		if !contains(contentStr, "another new message") {
			t.Error("Message 'old' should be replaced with 'new'")
		}
	})
}

func TestUnifiedBashTool_Execute(t *testing.T) {
	constraints := security.NewDefaultConstraints(".")
	tool := NewUnifiedBashTool(constraints)

	t.Run("Simple command", func(t *testing.T) {
		request := &ToolRequest{
			ID:       "test-1",
			ToolName: "bash",
			Parameters: map[string]interface{}{
				"command":     "echo 'Hello, World!'",
				"description": "Test echo command",
			},
		}

		response, err := tool.Execute(context.Background(), request)
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}

		if !response.Success {
			t.Errorf("Execute was not successful: %s", response.Error)
		}

		if !contains(response.Content, "Hello, World!") {
			t.Errorf("Expected output not found: %s", response.Content)
		}
	})

	t.Run("Command with timeout", func(t *testing.T) {
		request := &ToolRequest{
			ID:       "test-2",
			ToolName: "bash",
			Parameters: map[string]interface{}{
				"command": "sleep 2",
				"timeout": float64(1000), // 1秒でタイムアウト
			},
		}

		start := time.Now()
		response, _ := tool.Execute(context.Background(), request)
		duration := time.Since(start)

		// タイムアウトが発生することを確認
		if duration > 2*time.Second {
			t.Error("Command should have timed out")
		}

		if response != nil && response.Success {
			t.Error("Command should have failed due to timeout")
		}

		if response != nil && !contains(response.Error, "timeout") {
			t.Errorf("Expected timeout error, got: %s", response.Error)
		}
	})
}

func TestToolValidation(t *testing.T) {
	tool := NewUnifiedWriteTool(nil)

	t.Run("Valid request", func(t *testing.T) {
		request := &ToolRequest{
			ToolName: "write",
			Parameters: map[string]interface{}{
				"file_path": "/tmp/test.txt",
				"content":   "test content",
			},
		}

		err := tool.ValidateRequest(request)
		if err != nil {
			t.Errorf("Valid request failed validation: %v", err)
		}
	})

	t.Run("Missing required parameter", func(t *testing.T) {
		request := &ToolRequest{
			ToolName: "write",
			Parameters: map[string]interface{}{
				"file_path": "/tmp/test.txt",
				// content is missing
			},
		}

		err := tool.ValidateRequest(request)
		if err == nil {
			t.Error("Expected validation error for missing parameter")
		}
	})

	t.Run("Tool name mismatch", func(t *testing.T) {
		request := &ToolRequest{
			ToolName: "read", // wrong tool name
			Parameters: map[string]interface{}{
				"file_path": "/tmp/test.txt",
				"content":   "test content",
			},
		}

		err := tool.ValidateRequest(request)
		if err == nil {
			t.Error("Expected validation error for tool name mismatch")
		}
	})
}

func TestToolRegistry_Statistics(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "unified-stats-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	constraints := security.NewDefaultConstraints(tempDir)
	registry := NewUnifiedToolRegistry(constraints, nil)

	// テスト実行
	testFile := filepath.Join(tempDir, "stats_test.txt")
	request := &ToolRequest{
		ID:       "stats-test",
		ToolName: "write",
		Parameters: map[string]interface{}{
			"file_path": testFile,
			"content":   "stats test",
		},
	}

	// 複数回実行
	for i := 0; i < 3; i++ {
		_, err := registry.ExecuteTool(context.Background(), request)
		if err != nil {
			t.Errorf("Tool execution failed: %v", err)
		}
	}

	// 統計確認
	stats, err := registry.GetExecutionStats("write")
	if err != nil {
		t.Errorf("Failed to get execution stats: %v", err)
	}

	if stats.TotalExecutions != 3 {
		t.Errorf("Expected 3 total executions, got %d", stats.TotalExecutions)
	}

	if stats.SuccessfulRuns != 3 {
		t.Errorf("Expected 3 successful runs, got %d", stats.SuccessfulRuns)
	}

	if stats.ErrorRate != 0.0 {
		t.Errorf("Expected 0%% error rate, got %.2f%%", stats.ErrorRate*100)
	}

	// グローバル統計確認
	globalStats := registry.GetGlobalStats()
	if globalStats.TotalExecutions < 3 {
		t.Errorf("Expected at least 3 total executions, got %d", globalStats.TotalExecutions)
	}
}

// ヘルパー関数
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
