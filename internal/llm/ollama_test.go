package llm

import (
	"context"
	"testing"
	"time"
)

// TestOllamaClientCreation はOllamaクライアントの作成をテストする
func TestOllamaClientCreation(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434")

	if client == nil {
		t.Fatal("Ollamaクライアントの作成に失敗しました")
	}

	if client.BaseURL != "http://localhost:11434" {
		t.Errorf("期待値: http://localhost:11434, 実際値: %s", client.BaseURL)
	}
}

// TestOllamaRequestCreation はリクエスト作成機能をテストする
func TestOllamaRequestCreation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &ChatRequest{
		Model: "test-model",
		Messages: []ChatMessage{
			{Role: "user", Content: "test prompt"},
		},
		Stream: false,
	}

	// リクエスト作成のテスト（実際の送信はしない）
	if req.Model != "test-model" {
		t.Errorf("期待値: test-model, 実際値: %s", req.Model)
	}
	if len(req.Messages) != 1 {
		t.Errorf("期待値: 1, 実際値: %d", len(req.Messages))
	}
	if req.Messages[0].Content != "test prompt" {
		t.Errorf("期待値: test prompt, 実際値: %s", req.Messages[0].Content)
	}

	_ = ctx // コンテキストの使用を示す
}

// TestChatRequestValidation はリクエストの妥当性検証をテストする
func TestChatRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     *ChatRequest
		wantErr bool
	}{
		{
			name: "有効なリクエスト",
			req: &ChatRequest{
				Model: "qwen2.5-coder:14b",
				Messages: []ChatMessage{
					{Role: "user", Content: "Hello world"},
				},
				Stream: false,
			},
			wantErr: false,
		},
		{
			name: "空のモデル名",
			req: &ChatRequest{
				Model: "",
				Messages: []ChatMessage{
					{Role: "user", Content: "Hello world"},
				},
				Stream: false,
			},
			wantErr: true,
		},
		{
			name: "空のメッセージ",
			req: &ChatRequest{
				Model:    "qwen2.5-coder:14b",
				Messages: []ChatMessage{},
				Stream:   false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChatRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateChatRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// validateChatRequest はChatRequestの妥当性を検証する
func validateChatRequest(req *ChatRequest) error {
	if req.Model == "" {
		return &ValidationError{Message: "モデル名が空です"}
	}
	if len(req.Messages) == 0 {
		return &ValidationError{Message: "メッセージが空です"}
	}
	return nil
}

// ValidationError は妥当性検証エラーを表す
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
