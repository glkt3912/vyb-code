package ai

import (
	"context"
)

// LLMClient represents an interface for LLM interactions
// AIパッケージで使用するLLMクライアントのインターフェース
type LLMClient interface {
	GenerateResponse(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error)
}

// GenerateRequest represents a request to generate content
// コンテンツ生成リクエスト
type GenerateRequest struct {
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// GenerateResponse represents a response from content generation
// コンテンツ生成レスポンス
type GenerateResponse struct {
	Content string `json:"content"`
}

// Message represents a single message in conversation
// 会話における1つのメッセージ
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"`
}

// APIEndpoint - API エンドポイント情報
type APIEndpoint struct {
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	Handler     string   `json:"handler"`
	Middlewares []string `json:"middlewares"`
}

// ArchitecturalPattern - アーキテクチャパターン
type ArchitecturalPattern struct {
	Name        string  `json:"name"`
	Confidence  float64 `json:"confidence"`   // 0-1
	Description string  `json:"description"`
	Evidence    []string `json:"evidence"`
	Benefits    string  `json:"benefits"`
	Drawbacks   string  `json:"drawbacks"`
}

// ArchitecturalLayer - アーキテクチャ層
type ArchitecturalLayer struct {
	Name        string   `json:"name"`        // レイヤー名
	Description string   `json:"description"` // 説明
	Directories []string `json:"directories"` // 対応ディレクトリ
	Files       []string `json:"files"`       // 重要ファイル
}

// SecurityRecommendation - セキュリティ推奨事項
type SecurityRecommendation struct {
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

// ToolDefinition - ツール定義（ツール登録で使用）
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolResult - ツール実行結果
type ToolResult struct {
	IsError  bool                   `json:"is_error"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}