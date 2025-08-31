package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OllamaのHTTP APIに接続するためのクライアント構造体
type OllamaClient struct {
	BaseURL    string       // Ollama server URL (e.g., "http://localhost:11434")
	HTTPClient *http.Client // HTTP通信用のクライアント
}

// デフォルト設定でOllamaクライアントを作成するコンストラクタ関数
func NewOllamaClient(baseURL string) *OllamaClient {
	// URLが指定されていない場合はデフォルトのOllama URLを使用
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &OllamaClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second, // LLM応答用の2分タイムアウト
		},
	}
}

// Ollamaにチャットリクエストを送信し、レスポンスを返すメソッド
func (c *OllamaClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// リクエスト構造体をJSON形式に変換（API呼び出し用）
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// キャンセル可能なHTTP POSTリクエストを作成
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/chat", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// JSONコンテンツタイプのヘッダーを設定
	httpReq.Header.Set("Content-Type", "application/json")

	// OllamaにHTTPリクエストを送信
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() // レスポンスボディを確実にクローズ

	// HTTPステータスが成功かチェック
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API returned status %d", resp.StatusCode)
	}

	// JSONレスポンスを構造体に変換
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// OllamaがFunction Callingに対応しているかを返す（現在は未対応）
func (c *OllamaClient) SupportsFunctionCalling() bool {
	return false // Ollamaは現在Function Calling未対応
}

// 指定されたモデルの情報を取得する（簡易実装）
func (c *OllamaClient) GetModelInfo(model string) (*ModelInfo, error) {
	// 基本的なモデル情報を返す（拡張版ではOllama APIを呼び出す）
	return &ModelInfo{
		Name:        model,
		Size:        "unknown", // 実際のサイズ取得に拡張可能
		Description: "Ollama model",
	}, nil
}

// Ollamaから利用可能なモデル一覧を取得する
func (c *OllamaClient) ListModels() ([]ModelInfo, error) {
	// OllamaのタグエンドポイントへのGETリクエストを作成
	httpReq, err := http.NewRequest("GET", c.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// モデル一覧取得のリクエストを送信
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() // レスポンスボディを確実にクローズ

	// Ollamaのレスポンス形式を解析するための匿名構造体
	var result struct {
		Models []struct {
			Name string `json:"name"` // モデル名
			Size int64  `json:"size"` // モデルサイズ（バイト）
		} `json:"models"`
	}

	// JSONレスポンスを解析
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 独自のModelInfo形式に変換
	var models []ModelInfo
	for _, model := range result.Models {
		models = append(models, ModelInfo{
			Name:        model.Name,
			Size:        fmt.Sprintf("%d bytes", model.Size), // バイトを文字列に変換
			Description: "Ollama model",
		})
	}

	return models, nil
}
