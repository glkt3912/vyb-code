package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

// UnifiedWebSearchTool - 統一WebSearchツール（Web検索）
type UnifiedWebSearchTool struct {
	*BaseTool
	httpClient *http.Client
}

// WebSearchResult - Web検索結果
type WebSearchResult struct {
	Query          string             `json:"query"`
	Results        []SearchResultItem `json:"results"`
	TotalResults   int                `json:"total_results"`
	SearchTime     time.Duration      `json:"search_time"`
	AllowedDomains []string           `json:"allowed_domains,omitempty"`
	BlockedDomains []string           `json:"blocked_domains,omitempty"`
	SafeSearch     bool               `json:"safe_search"`
}

// SearchResultItem - 検索結果アイテム
type SearchResultItem struct {
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Snippet     string  `json:"snippet"`
	Domain      string  `json:"domain"`
	PublishedAt string  `json:"published_at,omitempty"`
	Language    string  `json:"language,omitempty"`
	Relevance   float64 `json:"relevance"`
}

// NewUnifiedWebSearchTool - 新しい統一WebSearchツールを作成
func NewUnifiedWebSearchTool(constraints *security.Constraints) *UnifiedWebSearchTool {
	base := NewBaseTool("websearch", "Allows Claude to search the web and use the results to inform responses", "1.0.0", CategoryWeb)
	base.AddCapability(CapabilityNetwork)
	base.SetConstraints(constraints)

	// HTTPクライアント設定
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// スキーマ設定
	schema := ToolSchema{
		Name:        "websearch",
		Description: "Allows Claude to search the web and use the results to inform responses",
		Version:     "1.0.0",
		Parameters: map[string]Parameter{
			"query": {
				Type:        "string",
				Description: "The search query to use",
				MinLength:   intPtr(2),
			},
			"allowed_domains": {
				Type:        "array",
				Description: "Only include search results from these domains",
			},
			"blocked_domains": {
				Type:        "array",
				Description: "Never include search results from these domains",
			},
			"max_results": {
				Type:        "integer",
				Description: "Maximum number of search results to return (default: 10)",
				Minimum:     floatPtr(1),
				Maximum:     floatPtr(50),
			},
			"safe_search": {
				Type:        "boolean",
				Description: "Enable safe search filtering (default: true)",
			},
			"language": {
				Type:        "string",
				Description: "Search language preference (ISO 639-1 code, e.g., 'en', 'ja')",
				Pattern:     "^[a-z]{2}$",
			},
		},
		Required: []string{"query"},
		Examples: []ToolExample{
			{
				Description: "Search for recent Go programming tutorials",
				Parameters: map[string]interface{}{
					"query":       "Go programming tutorial 2024",
					"max_results": 5,
					"language":    "en",
				},
			},
			{
				Description: "Search within specific domains",
				Parameters: map[string]interface{}{
					"query":           "artificial intelligence",
					"allowed_domains": []string{"arxiv.org", "research.google.com"},
				},
			},
		},
	}
	base.SetSchema(schema)

	tool := &UnifiedWebSearchTool{
		BaseTool:   base,
		httpClient: client,
	}

	return tool
}

// Execute - WebSearchツールを実行
func (t *UnifiedWebSearchTool) Execute(ctx context.Context, request *ToolRequest) (*ToolResponse, error) {
	if err := t.ValidateRequest(request); err != nil {
		return nil, err
	}

	// パラメータ取得
	query, _ := request.Parameters["query"].(string)

	// オプション取得
	maxResults := 10
	if max, ok := request.Parameters["max_results"].(float64); ok {
		maxResults = int(max)
	}

	safeSearch := true
	if safe, ok := request.Parameters["safe_search"].(bool); ok {
		safeSearch = safe
	}

	language := "en"
	if lang, ok := request.Parameters["language"].(string); ok && lang != "" {
		language = lang
	}

	var allowedDomains []string
	if allowed, ok := request.Parameters["allowed_domains"]; ok {
		if domains, ok := allowed.([]interface{}); ok {
			for _, d := range domains {
				if str, ok := d.(string); ok {
					allowedDomains = append(allowedDomains, str)
				}
			}
		}
	}

	var blockedDomains []string
	if blocked, ok := request.Parameters["blocked_domains"]; ok {
		if domains, ok := blocked.([]interface{}); ok {
			for _, d := range domains {
				if str, ok := d.(string); ok {
					blockedDomains = append(blockedDomains, str)
				}
			}
		}
	}

	// Web検索を実行
	result, err := t.performWebSearch(ctx, query, maxResults, safeSearch, language, allowedDomains, blockedDomains)
	if err != nil {
		return nil, NewToolError("execution_failed", fmt.Sprintf("Web search failed: %v", err))
	}

	response := &ToolResponse{
		ID:       request.ID,
		ToolName: t.name,
		Success:  true,
		Data: map[string]interface{}{
			"result":          result,
			"query":           query,
			"max_results":     maxResults,
			"safe_search":     safeSearch,
			"language":        language,
			"allowed_domains": allowedDomains,
			"blocked_domains": blockedDomains,
		},
	}

	return response, nil
}

// GetSchema - ツールスキーマを取得
func (t *UnifiedWebSearchTool) GetSchema() ToolSchema {
	return t.schema
}

// ValidateRequest - リクエストを検証
func (t *UnifiedWebSearchTool) ValidateRequest(request *ToolRequest) error {
	if err := t.BaseTool.ValidateRequest(request); err != nil {
		return err
	}

	// クエリ検証
	query, ok := request.Parameters["query"].(string)
	if !ok {
		return NewToolError("invalid_parameter", "Query must be a string")
	}

	query = strings.TrimSpace(query)
	if len(query) < 2 {
		return NewToolError("invalid_parameter", "Query must be at least 2 characters long")
	}

	// 言語検証（オプション）
	if lang, ok := request.Parameters["language"]; ok {
		if langStr, ok := lang.(string); ok {
			if len(langStr) != 2 {
				return NewToolError("invalid_parameter", "Language must be a 2-character ISO 639-1 code")
			}
		}
	}

	return nil
}

// Internal methods

// performWebSearch - Web検索を実行
func (t *UnifiedWebSearchTool) performWebSearch(
	ctx context.Context,
	query string,
	maxResults int,
	safeSearch bool,
	language string,
	allowedDomains,
	blockedDomains []string,
) (*WebSearchResult, error) {

	startTime := time.Now()

	// 実際のWeb検索API呼び出し（ここではモック実装）
	// 本格的な実装では Google Search API, Bing Search API, DuckDuckGo API などを使用
	results, err := t.performMockSearch(query, maxResults, allowedDomains, blockedDomains)
	if err != nil {
		return nil, fmt.Errorf("search API error: %w", err)
	}

	searchResult := &WebSearchResult{
		Query:          query,
		Results:        results,
		TotalResults:   len(results),
		SearchTime:     time.Since(startTime),
		AllowedDomains: allowedDomains,
		BlockedDomains: blockedDomains,
		SafeSearch:     safeSearch,
	}

	return searchResult, nil
}

// performMockSearch - モック検索実行（実際の実装では外部API使用）
func (t *UnifiedWebSearchTool) performMockSearch(
	query string,
	maxResults int,
	allowedDomains,
	blockedDomains []string,
) ([]SearchResultItem, error) {

	// モックデータ生成
	mockResults := []SearchResultItem{
		{
			Title:     fmt.Sprintf("Results for '%s' - Example Site", query),
			URL:       "https://example.com/search?q=" + url.QueryEscape(query),
			Snippet:   fmt.Sprintf("Comprehensive information about %s with detailed explanations and examples.", query),
			Domain:    "example.com",
			Language:  "en",
			Relevance: 0.95,
		},
		{
			Title:     fmt.Sprintf("%s Tutorial and Guide", query),
			URL:       "https://tutorials.org/" + url.QueryEscape(strings.ToLower(query)),
			Snippet:   fmt.Sprintf("Learn about %s with step-by-step tutorials and practical examples.", query),
			Domain:    "tutorials.org",
			Language:  "en",
			Relevance: 0.87,
		},
		{
			Title:     fmt.Sprintf("Advanced %s Techniques", query),
			URL:       "https://advanced.tech/" + url.QueryEscape(query),
			Snippet:   fmt.Sprintf("Professional techniques and best practices for %s implementation.", query),
			Domain:    "advanced.tech",
			Language:  "en",
			Relevance: 0.82,
		},
	}

	// ドメインフィルタリングを適用
	var filteredResults []SearchResultItem
	for _, result := range mockResults {
		// 禁止ドメインチェック
		if t.isDomainBlocked(result.Domain, blockedDomains) {
			continue
		}

		// 許可ドメインチェック（許可リストが設定されている場合）
		if len(allowedDomains) > 0 && !t.isDomainAllowed(result.Domain, allowedDomains) {
			continue
		}

		filteredResults = append(filteredResults, result)
	}

	// 結果数制限
	if len(filteredResults) > maxResults {
		filteredResults = filteredResults[:maxResults]
	}

	return filteredResults, nil
}

// isDomainBlocked - ドメインが禁止リストに含まれるかチェック
func (t *UnifiedWebSearchTool) isDomainBlocked(domain string, blockedDomains []string) bool {
	for _, blocked := range blockedDomains {
		if strings.Contains(domain, blocked) {
			return true
		}
	}
	return false
}

// isDomainAllowed - ドメインが許可リストに含まれるかチェック
func (t *UnifiedWebSearchTool) isDomainAllowed(domain string, allowedDomains []string) bool {
	for _, allowed := range allowedDomains {
		if strings.Contains(domain, allowed) {
			return true
		}
	}
	return false
}

// Helper functions
func intPtr(i int) *int {
	return &i
}
