package chat

import (
	"context"
	"testing"

	"github.com/glkt/vyb-code/internal/llm"
)

// MockProvider は LLM プロバイダーのモック実装
type MockProvider struct {
	responses []llm.ChatResponse
	callCount int
}

// NewMockProvider は新しいモックプロバイダーを作成
func NewMockProvider(responses []llm.ChatResponse) *MockProvider {
	return &MockProvider{
		responses: responses,
		callCount: 0,
	}
}

// Chat はモックのチャット応答を返す
func (m *MockProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	if m.callCount >= len(m.responses) {
		return &llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    "assistant",
				Content: "デフォルト応答",
			},
		}, nil
	}

	response := m.responses[m.callCount]
	m.callCount++
	return &response, nil
}

// SupportsFunctionCalling はfunction calling対応状況を返す
func (m *MockProvider) SupportsFunctionCalling() bool {
	return false
}

// GetModelInfo はモデル情報を返す
func (m *MockProvider) GetModelInfo(model string) (*llm.ModelInfo, error) {
	return &llm.ModelInfo{
		Name:        model,
		Description: "Test Model",
		Size:        "14B",
	}, nil
}

// ListModels は利用可能なモデル一覧を返す
func (m *MockProvider) ListModels() ([]llm.ModelInfo, error) {
	return []llm.ModelInfo{
		{
			Name:        "test-model",
			Description: "Test Model", 
			Size:        "14B",
		},
	}, nil
}

// TestNewSession は新しいセッション作成をテストする
func TestNewSession(t *testing.T) {
	mockProvider := NewMockProvider([]llm.ChatResponse{})
	model := "test-model"

	session := NewSession(mockProvider, model)

	if session == nil {
		t.Fatal("セッションの作成に失敗")
	}

	if session.model != model {
		t.Errorf("期待値: %s, 実際値: %s", model, session.model)
	}

	// プロバイダー設定の確認（型検証）
	if session.provider == nil {
		t.Error("プロバイダーが設定されていません")
	}

	if len(session.messages) != 0 {
		t.Errorf("初期メッセージ数が0ではありません: %d", len(session.messages))
	}

	if session.mcpManager == nil {
		t.Error("MCPマネージャーが初期化されていません")
	}
}

// TestProcessQuery は単発クエリ処理をテストする
func TestProcessQuery(t *testing.T) {
	// モック応答を設定
	mockResponse := llm.ChatResponse{
		Message: llm.ChatMessage{
			Role:    "assistant",
			Content: "テスト応答です",
		},
	}
	mockProvider := NewMockProvider([]llm.ChatResponse{mockResponse})

	session := NewSession(mockProvider, "test-model")
	
	// クエリを処理
	query := "テストクエリ"
	err := session.ProcessQuery(query)
	if err != nil {
		t.Fatalf("クエリ処理エラー: %v", err)
	}

	// メッセージ履歴を確認
	if len(session.messages) != 2 {
		t.Errorf("期待メッセージ数: 2, 実際: %d", len(session.messages))
	}

	// ユーザーメッセージを確認
	if session.messages[0].Role != "user" {
		t.Errorf("最初のメッセージのロールが期待値と異なります: %s", session.messages[0].Role)
	}

	if session.messages[0].Content != query {
		t.Errorf("ユーザーメッセージの内容が期待値と異なります: %s", session.messages[0].Content)
	}

	// アシスタント応答を確認
	if session.messages[1].Role != "assistant" {
		t.Errorf("応答メッセージのロールが期待値と異なります: %s", session.messages[1].Role)
	}

	if session.messages[1].Content != mockResponse.Message.Content {
		t.Errorf("応答メッセージの内容が期待値と異なります: %s", session.messages[1].Content)
	}
}

// TestClearHistory は履歴クリア機能をテストする
func TestClearHistory(t *testing.T) {
	session := NewSession(NewMockProvider([]llm.ChatResponse{}), "test-model")

	// いくつかのメッセージを追加
	session.messages = append(session.messages, llm.ChatMessage{
		Role:    "user",
		Content: "テストメッセージ1",
	})
	session.messages = append(session.messages, llm.ChatMessage{
		Role:    "assistant", 
		Content: "テスト応答1",
	})

	// 履歴をクリア
	session.ClearHistory()

	// メッセージが空になったことを確認
	if len(session.messages) != 0 {
		t.Errorf("履歴クリア後のメッセージ数が0ではありません: %d", len(session.messages))
	}
}

// TestGetMessageCount はメッセージ数取得をテストする
func TestGetMessageCount(t *testing.T) {
	session := NewSession(NewMockProvider([]llm.ChatResponse{}), "test-model")

	// 初期状態でのメッセージ数確認
	if count := session.GetMessageCount(); count != 0 {
		t.Errorf("初期メッセージ数が0ではありません: %d", count)
	}

	// メッセージを追加
	session.messages = append(session.messages, llm.ChatMessage{
		Role:    "user",
		Content: "テスト",
	})

	// メッセージ数確認
	if count := session.GetMessageCount(); count != 1 {
		t.Errorf("期待メッセージ数: 1, 実際: %d", count)
	}
}

// TestMCPServerOperations はMCPサーバー操作をテストする
func TestMCPServerOperations(t *testing.T) {
	session := NewSession(NewMockProvider([]llm.ChatResponse{}), "test-model")

	// MCPマネージャーが正しく初期化されていることを確認
	if session.mcpManager == nil {
		t.Fatal("MCPマネージャーが初期化されていません")
	}

	// MCP操作は実際の接続を必要とするため、基本的な関数呼び出しのみテスト
	t.Run("GetMCPTools", func(t *testing.T) {
		tools := session.GetMCPTools()
		if tools == nil {
			t.Error("MCPツール取得でnilが返されました")
		}
	})
}

// TestSessionClose はセッション終了処理をテストする
func TestSessionClose(t *testing.T) {
	session := NewSession(NewMockProvider([]llm.ChatResponse{}), "test-model")

	// セッション終了処理
	err := session.Close()
	if err != nil {
		t.Errorf("セッション終了エラー: %v", err)
	}
}