package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

// UnifiedWebFetchTool - 統一WebFetchツール（Webコンテンツ取得）
type UnifiedWebFetchTool struct {
	*BaseTool
	httpClient *http.Client
}

// WebFetchResult - WebFetch結果
type WebFetchResult struct {
	URL         string            `json:"url"`
	StatusCode  int               `json:"status_code"`
	ContentType string            `json:"content_type"`
	Content     string            `json:"content"`
	Size        int64             `json:"size"`
	Headers     map[string]string `json:"headers"`
	RedirectURL string            `json:"redirect_url,omitempty"`
}

// NewUnifiedWebFetchTool - 新しい統一WebFetchツールを作成
func NewUnifiedWebFetchTool(constraints *security.Constraints) *UnifiedWebFetchTool {
	base := NewBaseTool("webfetch", "Fetches content from a specified URL and processes it", "1.0.0", CategoryWeb)
	base.AddCapability(CapabilityNetwork)
	base.SetConstraints(constraints)

	// HTTPクライアント設定
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// リダイレクト回数制限（最大10回）
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	// スキーマ設定
	schema := ToolSchema{
		Name:        "webfetch",
		Description: "Fetches content from a specified URL and processes it using an AI model",
		Version:     "1.0.0",
		Parameters: map[string]Parameter{
			"url": {
				Type:        "string",
				Description: "The URL to fetch content from",
				Format:      "uri",
			},
			"prompt": {
				Type:        "string",
				Description: "The prompt to run on the fetched content",
			},
			"timeout": {
				Type:        "integer",
				Description: "Request timeout in seconds (default: 30)",
				Minimum:     floatPtr(1),
				Maximum:     floatPtr(300),
			},
			"follow_redirects": {
				Type:        "boolean",
				Description: "Whether to follow HTTP redirects (default: true)",
			},
		},
		Required: []string{"url", "prompt"},
		Examples: []ToolExample{
			{
				Description: "Fetch and summarize a web page",
				Parameters: map[string]interface{}{
					"url":    "https://example.com",
					"prompt": "Summarize the main content of this page",
				},
			},
		},
	}
	base.SetSchema(schema)

	tool := &UnifiedWebFetchTool{
		BaseTool:   base,
		httpClient: client,
	}

	return tool
}

// Execute - WebFetchツールを実行
func (t *UnifiedWebFetchTool) Execute(ctx context.Context, request *ToolRequest) (*ToolResponse, error) {
	if err := t.ValidateRequest(request); err != nil {
		return nil, err
	}

	// パラメータ取得
	urlStr, _ := request.Parameters["url"].(string)
	prompt, _ := request.Parameters["prompt"].(string)

	// オプション取得
	timeoutSec := 30
	if timeout, ok := request.Parameters["timeout"].(float64); ok {
		timeoutSec = int(timeout)
	}

	followRedirects := true
	if follow, ok := request.Parameters["follow_redirects"].(bool); ok {
		followRedirects = follow
	}

	// HTTPクライアントのタイムアウト設定
	t.httpClient.Timeout = time.Duration(timeoutSec) * time.Second

	// リダイレクト設定
	if !followRedirects {
		t.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Web取得を実行
	result, err := t.performWebFetch(ctx, urlStr, prompt)
	if err != nil {
		return nil, NewToolError("execution_failed", fmt.Sprintf("Web fetch failed: %v", err))
	}

	response := &ToolResponse{
		ID:       request.ID,
		ToolName: t.name,
		Success:  true,
		Data: map[string]interface{}{
			"result":           result,
			"url":              urlStr,
			"prompt":           prompt,
			"timeout":          timeoutSec,
			"follow_redirects": followRedirects,
		},
	}

	return response, nil
}

// GetSchema - ツールスキーマを取得
func (t *UnifiedWebFetchTool) GetSchema() ToolSchema {
	return t.schema
}

// ValidateRequest - リクエストを検証
func (t *UnifiedWebFetchTool) ValidateRequest(request *ToolRequest) error {
	if err := t.BaseTool.ValidateRequest(request); err != nil {
		return err
	}

	// URL検証
	urlStr, ok := request.Parameters["url"].(string)
	if !ok {
		return NewToolError("invalid_parameter", "URL must be a string")
	}

	if strings.TrimSpace(urlStr) == "" {
		return NewToolError("invalid_parameter", "URL cannot be empty")
	}

	// URL形式検証
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return NewToolError("invalid_parameter", fmt.Sprintf("Invalid URL format: %v", err))
	}

	// HTTP/HTTPSスキームのみ許可
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return NewToolError("invalid_parameter", "Only HTTP and HTTPS URLs are allowed")
	}

	// プロンプト検証
	prompt, ok := request.Parameters["prompt"].(string)
	if !ok {
		return NewToolError("invalid_parameter", "Prompt must be a string")
	}

	if strings.TrimSpace(prompt) == "" {
		return NewToolError("invalid_parameter", "Prompt cannot be empty")
	}

	return nil
}

// Internal methods

// performWebFetch - Web取得を実行
func (t *UnifiedWebFetchTool) performWebFetch(ctx context.Context, urlStr, prompt string) (*WebFetchResult, error) {
	// HTTPリクエストを作成
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// User-Agentを設定
	req.Header.Set("User-Agent", "VybCode/1.0 (WebFetch Tool)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	// HTTPリクエストを実行
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// レスポンス読み取り
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// ヘッダー情報を取得
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// リダイレクトURL検出
	var redirectURL string
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		redirectURL = resp.Header.Get("Location")
	}

	result := &WebFetchResult{
		URL:         urlStr,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Content:     string(content),
		Size:        int64(len(content)),
		Headers:     headers,
		RedirectURL: redirectURL,
	}

	// プロンプト処理（簡易実装 - 実際の実装ではLLMを使用）
	processedContent := t.processContentWithPrompt(result.Content, prompt)
	result.Content = processedContent

	return result, nil
}

// processContentWithPrompt - プロンプトでコンテンツを処理（簡易実装）
func (t *UnifiedWebFetchTool) processContentWithPrompt(content, prompt string) string {
	// 実際の実装ではLLMを使用するが、ここでは簡易処理
	maxLength := 5000
	if len(content) > maxLength {
		content = content[:maxLength] + "... (truncated)"
	}

	// HTMLタグの簡易除去
	content = t.stripHTMLTags(content)

	return fmt.Sprintf("Prompt: %s\n\nContent:\n%s", prompt, content)
}

// stripHTMLTags - HTMLタグを簡易除去
func (t *UnifiedWebFetchTool) stripHTMLTags(content string) string {
	// 簡易的なHTMLタグ除去（本格的な実装では正規表現やHTMLパーサーを使用）
	result := content

	// よく使われるHTMLタグを除去
	tags := []string{"<script>", "</script>", "<style>", "</style>", "<head>", "</head>"}
	for _, tag := range tags {
		result = strings.ReplaceAll(result, tag, "")
	}

	// 連続する空白を単一の空白に変換
	result = strings.ReplaceAll(result, "\n", " ")
	result = strings.ReplaceAll(result, "\t", " ")

	// 複数の空白を単一に
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	return strings.TrimSpace(result)
}
