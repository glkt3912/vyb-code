package mcp

import (
	"testing"
	"time"
)

// MCPクライアントのテスト
func TestNewClient(t *testing.T) {
	config := ClientConfig{
		ServerCommand: []string{"echo", "test"},
		Timeout:       30 * time.Second,
	}

	client := NewClient(config)

	if client == nil {
		t.Fatal("クライアントの作成に失敗しました")
	}

	if client.IsConnected() {
		t.Error("新しいクライアントは接続されていないはずです")
	}
}

// MCPサーバー設定のテスト
func TestClientConfig(t *testing.T) {
	config := ClientConfig{
		ServerCommand: []string{"test-server"},
		ServerArgs:    []string{"--port", "8080"},
		Environment:   map[string]string{"TEST": "value"},
		WorkingDir:    "/tmp",
		Timeout:       60 * time.Second,
	}

	if len(config.ServerCommand) != 1 {
		t.Errorf("期待されるコマンド数: 1, 実際: %d", len(config.ServerCommand))
	}

	if config.ServerCommand[0] != "test-server" {
		t.Errorf("期待されるコマンド: test-server, 実際: %s", config.ServerCommand[0])
	}

	if config.Environment["TEST"] != "value" {
		t.Error("環境変数が正しく設定されていません")
	}
}

// セッション状態のテスト
func TestSessionState(t *testing.T) {
	state := &SessionState{
		ID:        "test-session",
		Connected: false,
		Tools:     make([]Tool, 0),
		Resources: make([]Resource, 0),
		Prompts:   make([]Prompt, 0),
		LastPing:  time.Now(),
		Metadata:  make(map[string]string),
	}

	if state.ID != "test-session" {
		t.Errorf("期待されるID: test-session, 実際: %s", state.ID)
	}

	if state.Connected {
		t.Error("新しいセッション状態は非接続であるべきです")
	}

	if len(state.Tools) != 0 {
		t.Error("新しいセッション状態にはツールがないはずです")
	}
}

// ツール定義のテスト
func TestToolDefinition(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "テスト用ツール",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "入力文字列",
				},
			},
		},
	}

	if tool.Name != "test_tool" {
		t.Errorf("期待されるツール名: test_tool, 実際: %s", tool.Name)
	}

	if tool.Description != "テスト用ツール" {
		t.Errorf("期待される説明: テスト用ツール, 実際: %s", tool.Description)
	}

	schema, ok := tool.InputSchema.(map[string]interface{})
	if !ok {
		t.Error("InputSchemaはmap[string]interface{}である必要があります")
	}

	if schema["type"] != "object" {
		t.Error("スキーマタイプがobjectではありません")
	}
}

// ツール呼び出しのテスト
func TestToolCall(t *testing.T) {
	call := ToolCall{
		Name: "test_tool",
		Arguments: map[string]interface{}{
			"input": "test value",
			"count": 42,
		},
	}

	if call.Name != "test_tool" {
		t.Errorf("期待されるツール名: test_tool, 実際: %s", call.Name)
	}

	if call.Arguments["input"] != "test value" {
		t.Error("引数が正しく設定されていません")
	}

	if call.Arguments["count"] != 42 {
		t.Error("数値引数が正しく設定されていません")
	}
}

// ツール実行結果のテスト
func TestToolResult(t *testing.T) {
	result := ToolResult{
		Content: []Content{
			{
				Type: "text",
				Text: "ツール実行結果",
			},
		},
		IsError: false,
	}

	if len(result.Content) != 1 {
		t.Errorf("期待されるコンテンツ数: 1, 実際: %d", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Error("コンテンツタイプがtextではありません")
	}

	if result.Content[0].Text != "ツール実行結果" {
		t.Error("コンテンツテキストが正しくありません")
	}

	if result.IsError {
		t.Error("成功結果でIsErrorがtrueです")
	}
}

// エラー処理のテスト
func TestMCPError(t *testing.T) {
	err := &MCPError{
		Code:    -1001,
		Message: "テストエラー",
		Data:    map[string]interface{}{"detail": "詳細情報"},
	}

	if err.Code != -1001 {
		t.Errorf("期待されるエラーコード: -1001, 実際: %d", err.Code)
	}

	if err.Message != "テストエラー" {
		t.Errorf("期待されるエラーメッセージ: テストエラー, 実際: %s", err.Message)
	}

	data, ok := err.Data.(map[string]interface{})
	if !ok {
		t.Error("エラーデータがmap[string]interface{}ではありません")
	}

	if data["detail"] != "詳細情報" {
		t.Error("エラーデータの詳細が正しくありません")
	}
}
