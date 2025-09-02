package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
		{
			name: "空のコンテンツ",
			req: &ChatRequest{
				Model: "qwen2.5-coder:14b",
				Messages: []ChatMessage{
					{Role: "user", Content: ""},
				},
				Stream: false,
			},
			wantErr: true,
		},
		{
			name: "無効なロール",
			req: &ChatRequest{
				Model: "qwen2.5-coder:14b",
				Messages: []ChatMessage{
					{Role: "invalid", Content: "test"},
				},
				Stream: false,
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
	// メッセージの内容と役割をチェック
	for i, msg := range req.Messages {
		if msg.Content == "" {
			return &ValidationError{Message: fmt.Sprintf("メッセージ[%d]の内容が空です", i)}
		}
		if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" {
			return &ValidationError{Message: fmt.Sprintf("メッセージ[%d]の役割が無効です: %s", i, msg.Role)}
		}
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

// TestOllamaChat はChat機能の正常系をテストする
func TestOllamaChat(t *testing.T) {
	// モックサーバーを作成
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("期待値: POST, 実際値: %s", r.Method)
		}
		if r.URL.Path != "/api/chat" {
			t.Errorf("期待値: /api/chat, 実際値: %s", r.URL.Path)
		}

		// 正常なレスポンスを返す
		resp := ChatResponse{
			Message: ChatMessage{
				Role:    "assistant",
				Content: "テストレスポンス",
			},
			Done: true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)
	req := ChatRequest{
		Model: "test-model",
		Messages: []ChatMessage{
			{Role: "user", Content: "テストメッセージ"},
		},
		Stream: false,
	}

	ctx := context.Background()
	resp, err := client.Chat(ctx, req)

	if err != nil {
		t.Fatalf("Chat呼び出しエラー: %v", err)
	}
	if resp == nil {
		t.Fatal("レスポンスがnilです")
	}
	if resp.Message.Content != "テストレスポンス" {
		t.Errorf("期待値: テストレスポンス, 実際値: %s", resp.Message.Content)
	}
}

// TestOllamaChatNetworkError はネットワークエラーのテストする
func TestOllamaChatNetworkError(t *testing.T) {
	client := NewOllamaClient("http://invalid-host:99999")
	req := ChatRequest{
		Model: "test-model",
		Messages: []ChatMessage{
			{Role: "user", Content: "テストメッセージ"},
		},
		Stream: false,
	}

	ctx := context.Background()
	_, err := client.Chat(ctx, req)

	if err == nil {
		t.Fatal("ネットワークエラーが期待されましたが、エラーが発生しませんでした")
	}
	if !strings.Contains(err.Error(), "failed to send request") {
		t.Errorf("期待されるエラーメッセージが含まれていません: %v", err)
	}
}

// TestOllamaChatHTTPError はHTTPエラーステータスのテスト
func TestOllamaChatHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)
	req := ChatRequest{
		Model: "test-model",
		Messages: []ChatMessage{
			{Role: "user", Content: "テストメッセージ"},
		},
		Stream: false,
	}

	ctx := context.Background()
	_, err := client.Chat(ctx, req)

	if err == nil {
		t.Fatal("HTTPエラーが期待されましたが、エラーが発生しませんでした")
	}
	if !strings.Contains(err.Error(), "ollama API returned status 500") {
		t.Errorf("期待されるエラーメッセージが含まれていません: %v", err)
	}
}

// TestOllamaChatJSONDecodeError はJSONデコードエラーのテスト
func TestOllamaChatJSONDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)
	req := ChatRequest{
		Model: "test-model",
		Messages: []ChatMessage{
			{Role: "user", Content: "テストメッセージ"},
		},
		Stream: false,
	}

	ctx := context.Background()
	_, err := client.Chat(ctx, req)

	if err == nil {
		t.Fatal("JSONデコードエラーが期待されましたが、エラーが発生しませんでした")
	}
	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Errorf("期待されるエラーメッセージが含まれていません: %v", err)
	}
}

// TestOllamaChatTimeout はタイムアウト処理のテスト
func TestOllamaChatTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 意図的に遅延を発生させる
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatResponse{
			Message: ChatMessage{Role: "assistant", Content: "遅延レスポンス"},
			Done:    true,
		})
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)
	req := ChatRequest{
		Model: "test-model",
		Messages: []ChatMessage{
			{Role: "user", Content: "テストメッセージ"},
		},
		Stream: false,
	}

	// 短いタイムアウトでテスト
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Chat(ctx, req)

	if err == nil {
		t.Fatal("タイムアウトエラーが期待されましたが、エラーが発生しませんでした")
	}
}

// TestListModels はモデル一覧取得機能のテスト
func TestListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("期待値: GET, 実際値: %s", r.Method)
		}
		if r.URL.Path != "/api/tags" {
			t.Errorf("期待値: /api/tags, 実際値: %s", r.URL.Path)
		}

		// モックレスポンス
		resp := struct {
			Models []struct {
				Name string `json:"name"`
				Size int64  `json:"size"`
			} `json:"models"`
		}{
			Models: []struct {
				Name string `json:"name"`
				Size int64  `json:"size"`
			}{
				{Name: "qwen2.5-coder:14b", Size: 14000000000},
				{Name: "codellama:7b", Size: 7000000000},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)
	models, err := client.ListModels()

	if err != nil {
		t.Fatalf("ListModelsエラー: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("期待値: 2モデル, 実際値: %d", len(models))
	}
	if models[0].Name != "qwen2.5-coder:14b" {
		t.Errorf("期待値: qwen2.5-coder:14b, 実際値: %s", models[0].Name)
	}
}

// TestListModelsError はモデル一覧取得のエラー処理をテスト
func TestListModelsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)
	_, err := client.ListModels()

	if err == nil {
		t.Fatal("エラーが期待されましたが、エラーが発生しませんでした")
	}
}

// TestGetModelInfo はモデル情報取得機能のテスト
func TestGetModelInfo(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434")
	info, err := client.GetModelInfo("test-model")

	if err != nil {
		t.Fatalf("GetModelInfoエラー: %v", err)
	}
	if info == nil {
		t.Fatal("モデル情報がnilです")
	}
	if info.Name != "test-model" {
		t.Errorf("期待値: test-model, 実際値: %s", info.Name)
	}
	if info.Description != "Ollama model" {
		t.Errorf("期待値: Ollama model, 実際値: %s", info.Description)
	}
}

// TestSupportsFunctionCalling はFunction Calling対応状況のテスト
func TestSupportsFunctionCalling(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434")
	if client.SupportsFunctionCalling() {
		t.Error("Ollamaは現在Function Calling未対応のはずです")
	}
}

// TestNewOllamaClientWithEmptyURL は空URL指定時のデフォルト設定テスト
func TestNewOllamaClientWithEmptyURL(t *testing.T) {
	client := NewOllamaClient("")

	if client.BaseURL != "http://localhost:11434" {
		t.Errorf("期待値: http://localhost:11434, 実際値: %s", client.BaseURL)
	}
	if client.HTTPClient.Timeout != 120*time.Second {
		t.Errorf("期待値: 120s, 実際値: %v", client.HTTPClient.Timeout)
	}
}

// TestChatContextCancellation はコンテキストキャンセルのテスト
func TestChatContextCancellation(t *testing.T) {
	// 遅延のあるモックサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// キャンセルされるまで待機
		select {
		case <-time.After(1 * time.Second):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ChatResponse{
				Message: ChatMessage{Role: "assistant", Content: "遅延レスポンス"},
				Done:    true,
			})
		case <-r.Context().Done():
			return
		}
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)
	req := ChatRequest{
		Model: "test-model",
		Messages: []ChatMessage{
			{Role: "user", Content: "テストメッセージ"},
		},
		Stream: false,
	}

	// コンテキストを即座にキャンセル
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Chat(ctx, req)

	if err == nil {
		t.Fatal("コンテキストキャンセルエラーが期待されましたが、エラーが発生しませんでした")
	}
}

// TestChatMalformedJSON は不正なJSONレスポンスのテスト
func TestChatMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{broken json}"))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)
	req := ChatRequest{
		Model: "test-model",
		Messages: []ChatMessage{
			{Role: "user", Content: "テストメッセージ"},
		},
		Stream: false,
	}

	ctx := context.Background()
	_, err := client.Chat(ctx, req)

	if err == nil {
		t.Fatal("JSONデコードエラーが期待されましたが、エラーが発生しませんでした")
	}
	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Errorf("期待されるエラーメッセージが含まれていません: %v", err)
	}
}

// TestListModelsJSONError はモデル一覧取得時のJSONエラーテスト
func TestListModelsJSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("malformed"))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)
	_, err := client.ListModels()

	if err == nil {
		t.Fatal("JSONデコードエラーが期待されましたが、エラーが発生しませんでした")
	}
	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Errorf("期待されるエラーメッセージが含まれていません: %v", err)
	}
}
