package llm

import "context"

// ChatMessage represents a single message in the conversation
// LLMとの会話における1つのメッセージ（ユーザーまたはAIからの発言）
type ChatMessage struct {
	Role    string `json:"role"`    // "user" or "assistant" - 発言者の役割
	Content string `json:"content"` // Message content - メッセージの内容
}

// ChatRequest represents a request to the LLM API
// LLM APIへのリクエスト（会話履歴とモデル指定を含む）
type ChatRequest struct {
	Model    string        `json:"model"`    // Model name (e.g., "qwen2.5-coder:14b") - 使用するモデル名
	Messages []ChatMessage `json:"messages"` // Conversation history - 会話履歴の配列
	Stream   bool          `json:"stream"`   // Whether to stream response - ストリーミング応答の有無
}

// ChatResponse represents a response from the LLM API
// LLM APIからのレスポンス
type ChatResponse struct {
	Message ChatMessage `json:"message"` // AI's response message - AIからの返答メッセージ
	Done    bool        `json:"done"`    // Whether response is complete - 応答完了フラグ
}

// ModelInfo contains information about an available LLM model
// 利用可能なLLMモデルの情報
type ModelInfo struct {
	Name        string // Model name - モデル名
	Size        string // Model size (e.g., "7B", "13B") - モデルサイズ
	Description string // Model description - モデルの説明
}

// Provider defines the interface for LLM providers (Ollama, LM Studio, etc.)
// LLMプロバイダー（Ollama、LM Studio等）の共通インターフェース
type Provider interface {
	// Chat sends a chat request and returns the response
	// チャットリクエストを送信し、レスポンスを返す
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	
	// SupportsFunctionCalling returns whether this provider supports function calling
	// このプロバイダーがFunction Callingに対応しているかを返す
	SupportsFunctionCalling() bool
	
	// GetModelInfo retrieves information about a specific model
	// 指定されたモデルの情報を取得する
	GetModelInfo(model string) (*ModelInfo, error)
	
	// ListModels returns all available models
	// 利用可能なモデル一覧を返す
	ListModels() ([]ModelInfo, error)
}