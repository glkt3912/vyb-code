package chat

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/glkt/vyb-code/internal/llm"
)

// MockProvider は LLM プロバイダーのモック実装
type MockProvider struct {
	responses   []llm.ChatResponse
	callCount   int
	shouldError bool
	errorMsg    string
}

// NewMockProvider は新しいモックプロバイダーを作成
func NewMockProvider(responses []llm.ChatResponse) *MockProvider {
	return &MockProvider{
		responses:   responses,
		callCount:   0,
		shouldError: false,
		errorMsg:    "",
	}
}

// Chat はモックのチャット応答を返す
func (m *MockProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	if m.shouldError {
		return nil, fmt.Errorf(m.errorMsg)
	}

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

// TestProcessQueryWithLLMError はLLMエラー時のクエリ処理をテストする
func TestProcessQueryWithLLMError(t *testing.T) {
	mockProvider := &MockProvider{
		responses:   []llm.ChatResponse{},
		callCount:   0,
		shouldError: true,
		errorMsg:    "LLM connection failed",
	}
	session := NewSession(mockProvider, "test-model")

	err := session.ProcessQuery("テストクエリ")
	if err == nil {
		t.Fatal("LLMエラーが期待されましたが、エラーが発生しませんでした")
	}

	// エラーメッセージを確認
	if !contains(err.Error(), "LLM request failed") {
		t.Errorf("期待されるエラーメッセージが含まれていません: %v", err)
	}

	// ユーザーメッセージは履歴に追加されているはず
	if len(session.messages) != 1 {
		t.Errorf("期待メッセージ数: 1, 実際: %d", len(session.messages))
	}
	if session.messages[0].Role != "user" {
		t.Errorf("ユーザーメッセージのロールが期待値と異なります: %s", session.messages[0].Role)
	}
}

// TestProcessEmptyQuery は空クエリの処理をテストする
func TestProcessEmptyQuery(t *testing.T) {
	mockProvider := NewMockProvider([]llm.ChatResponse{
		{
			Message: llm.ChatMessage{
				Role:    "assistant",
				Content: "空のクエリです",
			},
		},
	})
	session := NewSession(mockProvider, "test-model")

	err := session.ProcessQuery("")
	if err != nil {
		t.Fatalf("空クエリ処理エラー: %v", err)
	}

	// 空クエリでもメッセージとして追加されるはず
	if len(session.messages) != 2 {
		t.Errorf("期待メッセージ数: 2, 実際: %d", len(session.messages))
	}
}

// TestLongConversationHistory は長い会話履歴での動作をテストする
func TestLongConversationHistory(t *testing.T) {
	mockProvider := NewMockProvider([]llm.ChatResponse{})
	session := NewSession(mockProvider, "test-model")

	// 多数のメッセージを追加（fmtを使わずに文字列結合）
	for i := 0; i < 100; i++ {
		session.messages = append(session.messages,
			llm.ChatMessage{Role: "user", Content: "ユーザーメッセージ"},
			llm.ChatMessage{Role: "assistant", Content: "アシスタント応答"},
		)
	}

	// メッセージ数確認
	if count := session.GetMessageCount(); count != 200 {
		t.Errorf("期待メッセージ数: 200, 実際: %d", count)
	}

	// 履歴クリアの動作確認
	session.ClearHistory()
	if count := session.GetMessageCount(); count != 0 {
		t.Errorf("クリア後の期待メッセージ数: 0, 実際: %d", count)
	}
}

// TestMultipleQueryProcessing は複数クエリの連続処理をテストする
func TestMultipleQueryProcessing(t *testing.T) {
	responses := []llm.ChatResponse{
		{Message: llm.ChatMessage{Role: "assistant", Content: "最初の応答"}},
		{Message: llm.ChatMessage{Role: "assistant", Content: "2番目の応答"}},
		{Message: llm.ChatMessage{Role: "assistant", Content: "3番目の応答"}},
	}
	mockProvider := NewMockProvider(responses)
	session := NewSession(mockProvider, "test-model")

	// 複数のクエリを順次処理
	queries := []string{"クエリ1", "クエリ2", "クエリ3"}
	for _, query := range queries {
		err := session.ProcessQuery(query)
		if err != nil {
			t.Fatalf("クエリ処理エラー: %v", err)
		}
	}

	// 全メッセージが履歴に追加されていることを確認（各クエリ + 各応答 = 6メッセージ）
	if count := session.GetMessageCount(); count != 6 {
		t.Errorf("期待メッセージ数: 6, 実際: %d", count)
	}

	// 最後の応答内容を確認
	lastMessage := session.messages[len(session.messages)-1]
	if lastMessage.Content != "3番目の応答" {
		t.Errorf("最後の応答内容が期待値と異なります: %s", lastMessage.Content)
	}
}

// contains はstrings.Containsのヘルパー関数
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
