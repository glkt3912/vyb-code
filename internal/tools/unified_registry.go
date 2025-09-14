package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/glkt/vyb-code/internal/mcp"
	"github.com/glkt/vyb-code/internal/security"
)

// UnifiedToolRegistry - 統一ツールレジストリ
type UnifiedToolRegistry struct {
	mu          sync.RWMutex
	tools       map[string]UnifiedToolInterface
	categories  map[ToolCategory][]string
	constraints *security.Constraints
	mcpManager  *mcp.Manager
	
	// 実行統計
	execStats   map[string]*ToolExecutionStats
	globalStats *GlobalToolStats
}

// ToolExecutionStats - ツール実行統計
type ToolExecutionStats struct {
	TotalExecutions   int64         `json:"total_executions"`
	SuccessfulRuns    int64         `json:"successful_runs"`
	FailedRuns        int64         `json:"failed_runs"`
	AverageTime       time.Duration `json:"average_time"`
	TotalTime         time.Duration `json:"total_time"`
	LastExecuted      time.Time     `json:"last_executed"`
	ErrorRate         float64       `json:"error_rate"`
}

// GlobalToolStats - グローバルツール統計
type GlobalToolStats struct {
	TotalTools        int                            `json:"total_tools"`
	ActiveTools       int                            `json:"active_tools"`
	TotalExecutions   int64                          `json:"total_executions"`
	ExecutionsByTool  map[string]int64               `json:"executions_by_tool"`
	ExecutionsByCategory map[ToolCategory]int64      `json:"executions_by_category"`
	AverageResponseTime time.Duration               `json:"average_response_time"`
	LastUpdate        time.Time                      `json:"last_update"`
}

// NewUnifiedToolRegistry - 新しい統一ツールレジストリを作成
func NewUnifiedToolRegistry(constraints *security.Constraints, mcpManager *mcp.Manager) *UnifiedToolRegistry {
	registry := &UnifiedToolRegistry{
		tools:       make(map[string]UnifiedToolInterface),
		categories:  make(map[ToolCategory][]string),
		constraints: constraints,
		mcpManager:  mcpManager,
		execStats:   make(map[string]*ToolExecutionStats),
		globalStats: &GlobalToolStats{
			ExecutionsByTool:     make(map[string]int64),
			ExecutionsByCategory: make(map[ToolCategory]int64),
			LastUpdate:          time.Now(),
		},
	}
	
	// デフォルトツールを登録
	registry.registerDefaultTools()
	
	return registry
}

// RegisterTool - ツールを登録
func (r *UnifiedToolRegistry) RegisterTool(tool UnifiedToolInterface) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	name := tool.GetName()
	if name == "" {
		return fmt.Errorf("ツール名は必須です")
	}
	
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("ツール '%s' は既に登録されています", name)
	}
	
	r.tools[name] = tool
	
	// カテゴリ分類（BaseToolの場合）
	if baseTool, ok := tool.(*BaseTool); ok {
		r.categories[baseTool.category] = append(r.categories[baseTool.category], name)
	}
	
	// 統計初期化
	r.execStats[name] = &ToolExecutionStats{
		LastExecuted: time.Now(),
	}
	
	r.updateGlobalStats()
	return nil
}

// UnregisterTool - ツールの登録を解除
func (r *UnifiedToolRegistry) UnregisterTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("ツール '%s' が見つかりません", name)
	}
	
	delete(r.tools, name)
	delete(r.execStats, name)
	
	// カテゴリから削除
	for category, tools := range r.categories {
		for i, toolName := range tools {
			if toolName == name {
				r.categories[category] = append(tools[:i], tools[i+1:]...)
				break
			}
		}
	}
	
	r.updateGlobalStats()
	return nil
}

// GetTool - ツールを取得
func (r *UnifiedToolRegistry) GetTool(name string) (UnifiedToolInterface, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("ツール '%s' が見つかりません", name)
	}
	
	if !tool.IsEnabled() {
		return nil, ErrToolNotEnabled
	}
	
	return tool, nil
}

// ListTools - ツール一覧を取得
func (r *UnifiedToolRegistry) ListTools() []UnifiedToolInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tools := make([]UnifiedToolInterface, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	
	return tools
}

// ListToolsByCategory - カテゴリ別ツール一覧を取得
func (r *UnifiedToolRegistry) ListToolsByCategory(category ToolCategory) []UnifiedToolInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var tools []UnifiedToolInterface
	if toolNames, exists := r.categories[category]; exists {
		for _, name := range toolNames {
			if tool, exists := r.tools[name]; exists && tool.IsEnabled() {
				tools = append(tools, tool)
			}
		}
	}
	
	return tools
}

// ExecuteTool - ツールを実行
func (r *UnifiedToolRegistry) ExecuteTool(ctx context.Context, request *ToolRequest) (*ToolResponse, error) {
	if request == nil {
		return nil, ErrInvalidRequest
	}
	
	tool, err := r.GetTool(request.ToolName)
	if err != nil {
		return r.createErrorResponse(request, err), err
	}
	
	// リクエスト検証
	if err := tool.ValidateRequest(request); err != nil {
		return r.createErrorResponse(request, err), err
	}
	
	// セキュリティ制約チェック
	if r.constraints != nil {
		if err := r.validateSecurity(tool, request); err != nil {
			return r.createErrorResponse(request, err), err
		}
	}
	
	// 実行統計記録開始
	startTime := time.Now()
	
	// ツール実行
	response, err := tool.Execute(ctx, request)
	
	// 実行統計更新
	r.updateExecutionStats(request.ToolName, startTime, err == nil)
	
	if err != nil {
		return r.createErrorResponse(request, err), err
	}
	
	if response == nil {
		response = &ToolResponse{
			ID:       request.ID,
			ToolName: request.ToolName,
			Success:  true,
			Duration: time.Since(startTime),
		}
	} else {
		response.Duration = time.Since(startTime)
	}
	
	return response, nil
}

// GetToolSchemas - 全ツールのスキーマを取得
func (r *UnifiedToolRegistry) GetToolSchemas() map[string]ToolSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	schemas := make(map[string]ToolSchema)
	for name, tool := range r.tools {
		schemas[name] = tool.GetSchema()
	}
	
	return schemas
}

// GetExecutionStats - ツール実行統計を取得
func (r *UnifiedToolRegistry) GetExecutionStats(toolName string) (*ToolExecutionStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	stats, exists := r.execStats[toolName]
	if !exists {
		return nil, fmt.Errorf("ツール '%s' の統計が見つかりません", toolName)
	}
	
	// コピーを返す
	statsCopy := *stats
	return &statsCopy, nil
}

// GetGlobalStats - グローバル統計を取得
func (r *UnifiedToolRegistry) GetGlobalStats() *GlobalToolStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	r.updateGlobalStats()
	statsCopy := *r.globalStats
	return &statsCopy
}

// SearchTools - ツールを検索
func (r *UnifiedToolRegistry) SearchTools(query string, category ToolCategory, capabilities []ToolCapability) []UnifiedToolInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var results []UnifiedToolInterface
	
	for _, tool := range r.tools {
		if !tool.IsEnabled() {
			continue
		}
		
		// カテゴリフィルター
		if category != "" {
			if baseTool, ok := tool.(*BaseTool); ok {
				if baseTool.category != category {
					continue
				}
			}
		}
		
		// 機能フィルター
		if len(capabilities) > 0 {
			toolCaps := tool.GetCapabilities()
			hasAllCaps := true
			for _, reqCap := range capabilities {
				found := false
				for _, toolCap := range toolCaps {
					if toolCap == reqCap {
						found = true
						break
					}
				}
				if !found {
					hasAllCaps = false
					break
				}
			}
			if !hasAllCaps {
				continue
			}
		}
		
		// テキスト検索
		if query != "" {
			name := tool.GetName()
			desc := tool.GetDescription()
			if !containsIgnoreCase(name, query) && !containsIgnoreCase(desc, query) {
				continue
			}
		}
		
		results = append(results, tool)
	}
	
	// 結果をソート（名前順）
	sort.Slice(results, func(i, j int) bool {
		return results[i].GetName() < results[j].GetName()
	})
	
	return results
}

// 内部メソッド

// registerDefaultTools - デフォルトツールを登録
func (r *UnifiedToolRegistry) registerDefaultTools() {
	// ファイルツール
	readTool := NewUnifiedReadTool(r.constraints)
	writeTool := NewUnifiedWriteTool(r.constraints)
	editTool := NewUnifiedEditTool(r.constraints)
	
	r.RegisterTool(readTool)
	r.RegisterTool(writeTool)
	r.RegisterTool(editTool)
	
	// コマンドツール
	bashTool := NewUnifiedBashTool(r.constraints)
	r.RegisterTool(bashTool)
	
	// 検索ツール（TODO: 実装）
	// globTool := NewUnifiedGlobTool()
	// grepTool := NewUnifiedGrepTool()
	// lsTool := NewUnifiedLSTool()
	
	// r.RegisterTool(globTool)
	// r.RegisterTool(grepTool)
	// r.RegisterTool(lsTool)
	
	// Webツール（TODO: 実装）
	// webFetchTool := NewUnifiedWebFetchTool()
	// webSearchTool := NewUnifiedWebSearchTool()
	
	// r.RegisterTool(webFetchTool)
	// r.RegisterTool(webSearchTool)
}

// createErrorResponse - エラーレスポンスを作成
func (r *UnifiedToolRegistry) createErrorResponse(request *ToolRequest, err error) *ToolResponse {
	return &ToolResponse{
		ID:       request.ID,
		ToolName: request.ToolName,
		Success:  false,
		Error:    err.Error(),
		Duration: 0,
	}
}

// validateSecurity - セキュリティ検証
func (r *UnifiedToolRegistry) validateSecurity(tool UnifiedToolInterface, request *ToolRequest) error {
	// ファイル操作ツールの場合（簡易チェック）
	if hasCapability(tool.GetCapabilities(), CapabilityFileWrite) {
		if filePath, ok := request.Parameters["file_path"].(string); ok {
			if strings.Contains(filePath, "..") {
				return NewToolError("security_violation", "File path contains invalid characters: "+filePath)
			}
		}
	}
	
	// コマンド実行ツールの場合
	if hasCapability(tool.GetCapabilities(), CapabilityCommand) {
		if command, ok := request.Parameters["command"].(string); ok {
			if err := r.constraints.ValidateCommand(command); err != nil {
				return NewToolError("security_violation", "Command validation failed: "+err.Error())
			}
		}
	}
	
	return nil
}

// updateExecutionStats - 実行統計を更新
func (r *UnifiedToolRegistry) updateExecutionStats(toolName string, startTime time.Time, success bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	stats, exists := r.execStats[toolName]
	if !exists {
		stats = &ToolExecutionStats{}
		r.execStats[toolName] = stats
	}
	
	duration := time.Since(startTime)
	stats.TotalExecutions++
	stats.TotalTime += duration
	stats.LastExecuted = time.Now()
	
	if success {
		stats.SuccessfulRuns++
	} else {
		stats.FailedRuns++
	}
	
	// 平均時間を計算
	stats.AverageTime = stats.TotalTime / time.Duration(stats.TotalExecutions)
	
	// エラー率を計算
	if stats.TotalExecutions > 0 {
		stats.ErrorRate = float64(stats.FailedRuns) / float64(stats.TotalExecutions)
	}
	
	// グローバル統計更新
	r.globalStats.ExecutionsByTool[toolName]++
	r.globalStats.TotalExecutions++
}

// updateGlobalStats - グローバル統計を更新
func (r *UnifiedToolRegistry) updateGlobalStats() {
	r.globalStats.TotalTools = len(r.tools)
	r.globalStats.ActiveTools = 0
	
	var totalTime time.Duration
	var totalExecs int64
	
	for _, tool := range r.tools {
		if tool.IsEnabled() {
			r.globalStats.ActiveTools++
		}
	}
	
	for _, stats := range r.execStats {
		totalTime += stats.TotalTime
		totalExecs += stats.TotalExecutions
	}
	
	if totalExecs > 0 {
		r.globalStats.AverageResponseTime = totalTime / time.Duration(totalExecs)
	}
	
	r.globalStats.LastUpdate = time.Now()
}

// ヘルパー関数

// hasCapability - ツールが特定の機能を持つかチェック
func hasCapability(capabilities []ToolCapability, target ToolCapability) bool {
	for _, cap := range capabilities {
		if cap == target {
			return true
		}
	}
	return false
}

// containsIgnoreCase - 大文字小文字を無視した文字列検索
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (len(substr) == 0 || 
			strings.Contains(strings.ToLower(s), strings.ToLower(substr)))
}

