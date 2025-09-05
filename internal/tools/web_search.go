package tools

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebSearchTool - Web検索機能（Claude Code相当）
type WebSearchTool struct {
	client  *http.Client
	timeout time.Duration
}

func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
}

type WebSearchOptions struct {
	Query           string   `json:"query"`
	AllowedDomains  []string `json:"allowed_domains,omitempty"`
	BlockedDomains  []string `json:"blocked_domains,omitempty"`
	MaxResults      int      `json:"max_results,omitempty"`
}

// 注意: 実際のWeb検索APIは外部サービス（Google Custom Search API等）が必要
// ここではシミュレーションとして、一般的な技術サイトを対象とした検索を実装
func (ws *WebSearchTool) Search(options WebSearchOptions) (*ToolExecutionResult, error) {
	if options.Query == "" {
		return &ToolExecutionResult{
			Content: "検索クエリが指定されていません",
			IsError: true,
			Tool:    "websearch",
		}, fmt.Errorf("query required")
	}

	// 最大結果数のデフォルト設定
	maxResults := options.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}

	// 実際のWeb検索の代替として、技術文書サイトから検索
	results, err := ws.searchTechSites(options.Query, maxResults, options.AllowedDomains, options.BlockedDomains)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("検索エラー: %v", err),
			IsError: true,
			Tool:    "websearch",
		}, err
	}

	// 結果をフォーマット
	var resultLines []string
	for i, result := range results {
		resultLines = append(resultLines, fmt.Sprintf(
			"## 検索結果 %d\n**タイトル:** %s\n**URL:** %s\n**概要:** %s\n**ソース:** %s\n",
			i+1, result.Title, result.URL, result.Snippet, result.Source,
		))
	}

	content := strings.Join(resultLines, "\n")

	return &ToolExecutionResult{
		Content: content,
		IsError: false,
		Tool:    "websearch",
		Metadata: map[string]interface{}{
			"query":        options.Query,
			"result_count": len(results),
			"max_results":  maxResults,
		},
	}, nil
}

// 技術サイトからの検索シミュレーション
func (ws *WebSearchTool) searchTechSites(query string, maxResults int, allowedDomains, blockedDomains []string) ([]SearchResult, error) {
	// 主要な技術文書サイト
	techSites := []TechSite{
		{"Go Documentation", "https://pkg.go.dev", "Go言語の公式パッケージドキュメント"},
		{"MDN Web Docs", "https://developer.mozilla.org", "Web開発の包括的なリソース"},
		{"Stack Overflow", "https://stackoverflow.com", "プログラミングQ&Aコミュニティ"},
		{"GitHub", "https://github.com", "ソースコードホスティングサービス"},
		{"Docker Hub", "https://hub.docker.com", "コンテナイメージレジストリ"},
		{"AWS Documentation", "https://docs.aws.amazon.com", "Amazon Web Services ドキュメント"},
		{"Kubernetes Documentation", "https://kubernetes.io/docs", "Kubernetes 公式ドキュメント"},
		{"React Documentation", "https://react.dev", "React公式ドキュメント"},
		{"Vue.js Documentation", "https://vuejs.org", "Vue.js公式ドキュメント"},
		{"Node.js Documentation", "https://nodejs.org/docs", "Node.js公式ドキュメント"},
	}

	var results []SearchResult
	queryLower := strings.ToLower(query)

	for _, site := range techSites {
		if len(results) >= maxResults {
			break
		}

		// ドメインフィルタリング
		if !ws.isAllowedDomain(site.BaseURL, allowedDomains, blockedDomains) {
			continue
		}

		// 簡単なキーワードマッチング
		if ws.matchesQuery(site, queryLower) {
			result := SearchResult{
				Title:   fmt.Sprintf("%s - %s", site.Name, query),
				URL:     fmt.Sprintf("%s/search?q=%s", site.BaseURL, url.QueryEscape(query)),
				Snippet: fmt.Sprintf("%s に関する情報。%s", query, site.Description),
				Source:  site.Name,
			}
			results = append(results, result)
		}
	}

	// より具体的な検索結果を追加（実際のAPIを使用する場合はここで外部API呼び出し）
	if len(results) == 0 {
		// フォールバック結果
		results = append(results, SearchResult{
			Title:   fmt.Sprintf("「%s」に関する技術情報", query),
			URL:     fmt.Sprintf("https://www.google.com/search?q=%s", url.QueryEscape(query+" programming")),
			Snippet: fmt.Sprintf("「%s」に関連するプログラミングと開発の情報を検索します。", query),
			Source:  "Web検索",
		})
	}

	return results, nil
}

type TechSite struct {
	Name        string
	BaseURL     string
	Description string
}

func (ws *WebSearchTool) matchesQuery(site TechSite, queryLower string) bool {
	// 基本的なキーワードマッチング
	siteName := strings.ToLower(site.Name)
	siteDesc := strings.ToLower(site.Description)
	
	// Go関連のクエリ
	if strings.Contains(queryLower, "go") || strings.Contains(queryLower, "golang") {
		return strings.Contains(siteName, "go") || strings.Contains(siteDesc, "go")
	}
	
	// JavaScript関連
	if strings.Contains(queryLower, "javascript") || strings.Contains(queryLower, "js") || 
	   strings.Contains(queryLower, "react") || strings.Contains(queryLower, "vue") || strings.Contains(queryLower, "node") {
		return strings.Contains(siteName, "javascript") || strings.Contains(siteName, "react") || 
		       strings.Contains(siteName, "vue") || strings.Contains(siteName, "node") ||
		       strings.Contains(siteName, "mdn")
	}
	
	// Docker関連
	if strings.Contains(queryLower, "docker") || strings.Contains(queryLower, "container") {
		return strings.Contains(siteName, "docker") || strings.Contains(siteDesc, "container")
	}
	
	// AWS関連
	if strings.Contains(queryLower, "aws") || strings.Contains(queryLower, "amazon") || strings.Contains(queryLower, "cloud") {
		return strings.Contains(siteName, "aws") || strings.Contains(siteDesc, "amazon")
	}
	
	// Kubernetes関連
	if strings.Contains(queryLower, "kubernetes") || strings.Contains(queryLower, "k8s") {
		return strings.Contains(siteName, "kubernetes")
	}
	
	// 一般的な開発関連クエリ
	if strings.Contains(queryLower, "api") || strings.Contains(queryLower, "documentation") || 
	   strings.Contains(queryLower, "tutorial") || strings.Contains(queryLower, "example") {
		return true // ほとんどの技術サイトが該当
	}
	
	return false
}

func (ws *WebSearchTool) isAllowedDomain(siteURL string, allowedDomains, blockedDomains []string) bool {
	u, err := url.Parse(siteURL)
	if err != nil {
		return false
	}
	
	domain := u.Hostname()
	
	// 明示的に禁止されたドメインをチェック
	for _, blocked := range blockedDomains {
		if strings.Contains(domain, blocked) {
			return false
		}
	}
	
	// 許可されたドメインが指定されている場合はそれのみを許可
	if len(allowedDomains) > 0 {
		for _, allowed := range allowedDomains {
			if strings.Contains(domain, allowed) {
				return true
			}
		}
		return false
	}
	
	return true
}

// 実際の検索API統合用のスタブ（将来の拡張用）
func (ws *WebSearchTool) searchWithAPI(query string, maxResults int) ([]SearchResult, error) {
	// 実際のGoogle Custom Search API, Bing Search API等との統合はここで実装
	// API キーやエンドポイントの設定が必要
	
	return nil, fmt.Errorf("外部検索API統合は未実装")
}

// リアルタイム検索結果の取得（特定サイトから）
func (ws *WebSearchTool) fetchFromSite(siteURL, query string) (*SearchResult, error) {
	// 特定サイトから実際にコンテンツを取得
	searchURL := fmt.Sprintf("%s/search?q=%s", siteURL, url.QueryEscape(query))
	
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("User-Agent", "vyb-code/1.0 (Local AI Assistant)")
	
	resp, err := ws.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	
	// レスポンスの解析（実装はサイトごとに異なる）
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// 基本的なHTMLからのテキスト抽出（実際にはより高度な解析が必要）
	content := string(body)
	
	return &SearchResult{
		Title:   fmt.Sprintf("検索結果: %s", query),
		URL:     searchURL,
		Snippet: ws.extractSnippet(content, query),
		Source:  siteURL,
	}, nil
}

func (ws *WebSearchTool) extractSnippet(content, query string) string {
	// 簡単なスニペット抽出
	contentLower := strings.ToLower(content)
	queryLower := strings.ToLower(query)
	
	index := strings.Index(contentLower, queryLower)
	if index == -1 {
		// クエリが見つからない場合は最初の200文字を返す
		if len(content) > 200 {
			return content[:200] + "..."
		}
		return content
	}
	
	// クエリ周辺のテキストを抽出
	start := index - 50
	if start < 0 {
		start = 0
	}
	
	end := index + len(query) + 150
	if end > len(content) {
		end = len(content)
	}
	
	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}
	
	return snippet
}