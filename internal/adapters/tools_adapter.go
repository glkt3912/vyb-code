package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
	"github.com/glkt/vyb-code/internal/security"
	"github.com/glkt/vyb-code/internal/tools"
)

// ToolsAdapter - ツールシステムアダプター
type ToolsAdapter struct {
	*BaseAdapter

	// レガシーシステム
	legacyRegistry *tools.ToolRegistry

	// 統合システム
	unifiedRegistry *tools.UnifiedToolRegistry

	// セキュリティ制約
	constraints *security.Constraints
}

// NewToolsAdapter - 新しいツールアダプターを作成
func NewToolsAdapter(log logger.Logger, constraints *security.Constraints) *ToolsAdapter {
	return &ToolsAdapter{
		BaseAdapter: NewBaseAdapter(AdapterTypeTools, log),
		constraints: constraints,
	}
}

// Configure - ツールアダプターの設定
func (ta *ToolsAdapter) Configure(config *config.GradualMigrationConfig) error {
	if err := ta.BaseAdapter.Configure(config); err != nil {
		return err
	}

	// レガシーシステムの初期化
	if ta.legacyRegistry == nil {
		ta.legacyRegistry = tools.NewToolRegistry(ta.constraints, ".", 10*1024*1024, nil) // デフォルト値
	}

	// 統合システムの初期化
	if ta.unifiedRegistry == nil && ta.IsUnifiedEnabled() {
		// MCP manager は nil でも問題ない（統合システムは独立して動作）
		ta.unifiedRegistry = tools.NewUnifiedToolRegistry(ta.constraints, nil)
	}

	return nil
}

// ExecuteTool - ツール実行（統一インターフェース）
func (ta *ToolsAdapter) ExecuteTool(ctx context.Context, toolName string, parameters map[string]interface{}) (interface{}, error) {
	startTime := time.Now()

	useUnified := !ta.ShouldUseLegacy()

	var result interface{}
	var err error

	if useUnified && ta.unifiedRegistry != nil {
		result, err = ta.executeToolWithUnified(ctx, toolName, parameters)
	} else {
		result, err = ta.executeToolWithLegacy(ctx, toolName, parameters)
	}

	// フォールバック処理
	if err != nil && useUnified && ta.config.EnableFallback {
		ta.IncrementFallback()
		ta.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"tool":  toolName,
			"error": err.Error(),
		})
		result, err = ta.executeToolWithLegacy(ctx, toolName, parameters)
		useUnified = false
	}

	latency := time.Since(startTime)
	ta.UpdateMetrics(err == nil, useUnified, latency)
	ta.LogOperation(fmt.Sprintf("ExecuteTool_%s", toolName), useUnified, latency, err)

	return result, err
}

// ListTools - ツール一覧取得（統一インターフェース）
func (ta *ToolsAdapter) ListTools(ctx context.Context) ([]string, error) {
	startTime := time.Now()

	useUnified := !ta.ShouldUseLegacy()

	var toolList []string
	var err error

	if useUnified && ta.unifiedRegistry != nil {
		toolList, err = ta.listToolsWithUnified(ctx)
	} else {
		toolList, err = ta.listToolsWithLegacy(ctx)
	}

	// フォールバック処理
	if err != nil && useUnified && ta.config.EnableFallback {
		ta.IncrementFallback()
		ta.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"error": err.Error(),
		})
		toolList, err = ta.listToolsWithLegacy(ctx)
		useUnified = false
	}

	latency := time.Since(startTime)
	ta.UpdateMetrics(err == nil, useUnified, latency)
	ta.LogOperation("ListTools", useUnified, latency, err)

	return toolList, err
}

// GetToolSchema - ツールスキーマ取得（統一インターフェース）
func (ta *ToolsAdapter) GetToolSchema(ctx context.Context, toolName string) (interface{}, error) {
	startTime := time.Now()

	useUnified := !ta.ShouldUseLegacy()

	var schema interface{}
	var err error

	if useUnified && ta.unifiedRegistry != nil {
		schema, err = ta.getToolSchemaWithUnified(ctx, toolName)
	} else {
		schema, err = ta.getToolSchemaWithLegacy(ctx, toolName)
	}

	// フォールバック処理
	if err != nil && useUnified && ta.config.EnableFallback {
		ta.IncrementFallback()
		ta.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"tool":  toolName,
			"error": err.Error(),
		})
		schema, err = ta.getToolSchemaWithLegacy(ctx, toolName)
		useUnified = false
	}

	latency := time.Since(startTime)
	ta.UpdateMetrics(err == nil, useUnified, latency)
	ta.LogOperation(fmt.Sprintf("GetToolSchema_%s", toolName), useUnified, latency, err)

	return schema, err
}

// ValidateToolRequest - ツールリクエスト検証（統一インターフェース）
func (ta *ToolsAdapter) ValidateToolRequest(ctx context.Context, toolName string, parameters map[string]interface{}) error {
	startTime := time.Now()

	useUnified := !ta.ShouldUseLegacy()

	var err error

	if useUnified && ta.unifiedRegistry != nil {
		err = ta.validateToolRequestWithUnified(ctx, toolName, parameters)
	} else {
		err = ta.validateToolRequestWithLegacy(ctx, toolName, parameters)
	}

	// フォールバック処理
	if err != nil && useUnified && ta.config.EnableFallback {
		ta.IncrementFallback()
		ta.log.Warn("統合システムでエラー発生、レガシーシステムにフォールバック", map[string]interface{}{
			"tool":  toolName,
			"error": err.Error(),
		})
		err = ta.validateToolRequestWithLegacy(ctx, toolName, parameters)
		useUnified = false
	}

	latency := time.Since(startTime)
	ta.UpdateMetrics(err == nil, useUnified, latency)
	ta.LogOperation(fmt.Sprintf("ValidateToolRequest_%s", toolName), useUnified, latency, err)

	return err
}

// HealthCheck - ツールアダプターのヘルスチェック
func (ta *ToolsAdapter) HealthCheck(ctx context.Context) error {
	if err := ta.BaseAdapter.HealthCheck(ctx); err != nil {
		return err
	}

	// レガシーシステムのヘルスチェック
	if ta.legacyRegistry == nil {
		return fmt.Errorf("legacy tools registry not initialized")
	}

	// 統合システムのヘルスチェック（有効な場合）
	if ta.IsUnifiedEnabled() {
		if ta.unifiedRegistry == nil {
			return fmt.Errorf("unified tools registry not initialized")
		}

		// 統合システムの基本機能チェック（利用可能ツール数の確認）
		tools := ta.unifiedRegistry.ListTools()
		if len(tools) == 0 {
			return fmt.Errorf("no tools available in unified registry")
		}
	}

	return nil
}

// Internal methods

// executeToolWithUnified - 統合システムでのツール実行
func (ta *ToolsAdapter) executeToolWithUnified(ctx context.Context, toolName string, parameters map[string]interface{}) (interface{}, error) {
	if ta.unifiedRegistry == nil {
		return nil, fmt.Errorf("unified tools registry not initialized")
	}

	// ツールリクエスト構築
	request := &tools.ToolRequest{
		ID:         fmt.Sprintf("req_%d", time.Now().UnixNano()),
		ToolName:   toolName,
		Parameters: parameters,
	}

	// ツール実行
	response, err := ta.unifiedRegistry.ExecuteTool(ctx, request)
	if err != nil {
		return nil, err
	}

	if !response.Success {
		return nil, fmt.Errorf("tool execution failed: %s", response.Error)
	}

	return response.Data, nil
}

// executeToolWithLegacy - レガシーシステムでのツール実行
func (ta *ToolsAdapter) executeToolWithLegacy(ctx context.Context, toolName string, parameters map[string]interface{}) (interface{}, error) {
	if ta.legacyRegistry == nil {
		return nil, fmt.Errorf("legacy tools registry not initialized")
	}

	result, err := ta.legacyRegistry.ExecuteTool(toolName, parameters)
	if err != nil {
		return nil, err
	}
	return result.Content, nil
}

// listToolsWithUnified - 統合システムでのツール一覧取得
func (ta *ToolsAdapter) listToolsWithUnified(ctx context.Context) ([]string, error) {
	if ta.unifiedRegistry == nil {
		return nil, fmt.Errorf("unified tools registry not initialized")
	}

	tools := ta.unifiedRegistry.ListTools()
	var toolNames []string

	for _, tool := range tools {
		toolNames = append(toolNames, tool.GetName())
	}

	return toolNames, nil
}

// listToolsWithLegacy - レガシーシステムでのツール一覧取得
func (ta *ToolsAdapter) listToolsWithLegacy(ctx context.Context) ([]string, error) {
	if ta.legacyRegistry == nil {
		return nil, fmt.Errorf("legacy tools registry not initialized")
	}

	// TODO: ToolRegistryにListToolsメソッドがないため、GetAllToolsを使用
	allTools := ta.legacyRegistry.GetAllTools()
	var tools []string
	for name := range allTools {
		tools = append(tools, name)
	}
	return tools, nil
}

// getToolSchemaWithUnified - 統合システムでのツールスキーマ取得
func (ta *ToolsAdapter) getToolSchemaWithUnified(ctx context.Context, toolName string) (interface{}, error) {
	if ta.unifiedRegistry == nil {
		return nil, fmt.Errorf("unified tools registry not initialized")
	}

	tool, err := ta.unifiedRegistry.GetTool(toolName)
	if err != nil {
		return nil, err
	}

	return tool.GetSchema(), nil
}

// getToolSchemaWithLegacy - レガシーシステムでのツールスキーマ取得
func (ta *ToolsAdapter) getToolSchemaWithLegacy(ctx context.Context, toolName string) (interface{}, error) {
	if ta.legacyRegistry == nil {
		return nil, fmt.Errorf("legacy tools registry not initialized")
	}

	// TODO: GetToolSchemaメソッドが存在しないため、GetAllToolsでスキーマを取得
	allTools := ta.legacyRegistry.GetAllTools()
	tool, exists := allTools[toolName]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", toolName)
	}
	return tool.Schema, nil
}

// validateToolRequestWithUnified - 統合システムでのツールリクエスト検証
func (ta *ToolsAdapter) validateToolRequestWithUnified(ctx context.Context, toolName string, parameters map[string]interface{}) error {
	if ta.unifiedRegistry == nil {
		return fmt.Errorf("unified tools registry not initialized")
	}

	tool, err := ta.unifiedRegistry.GetTool(toolName)
	if err != nil {
		return err
	}

	request := &tools.ToolRequest{
		ID:         fmt.Sprintf("validate_%d", time.Now().UnixNano()),
		ToolName:   toolName,
		Parameters: parameters,
	}

	return tool.ValidateRequest(request)
}

// validateToolRequestWithLegacy - レガシーシステムでのツールリクエスト検証
func (ta *ToolsAdapter) validateToolRequestWithLegacy(ctx context.Context, toolName string, parameters map[string]interface{}) error {
	if ta.legacyRegistry == nil {
		return fmt.Errorf("legacy tools registry not initialized")
	}

	// TODO: ValidateToolRequestメソッドが存在しないため、簡易チェック
	allTools := ta.legacyRegistry.GetAllTools()
	_, exists := allTools[toolName]
	if !exists {
		return fmt.Errorf("tool '%s' not found", toolName)
	}
	if parameters == nil {
		return fmt.Errorf("parameters cannot be nil")
	}
	return nil
}

// GetToolsMetrics - ツール固有のメトリクスを取得
func (ta *ToolsAdapter) GetToolsMetrics() *ToolsMetrics {
	baseMetrics := ta.GetMetrics()

	toolsMetrics := &ToolsMetrics{
		AdapterMetrics:  *baseMetrics,
		UnifiedRegistry: ta.unifiedRegistry != nil,
		LegacyRegistry:  ta.legacyRegistry != nil,
	}

	// 統合システムのメトリクス取得
	if ta.unifiedRegistry != nil {
		unifiedStats := ta.unifiedRegistry.GetGlobalStats()
		toolsMetrics.UnifiedToolsMetrics = unifiedStats
	}

	// レガシーシステムのメトリクス取得
	if ta.legacyRegistry != nil {
		// TODO: GetStatisticsメソッドが存在しないため、簡易統計を作成
		allTools := ta.legacyRegistry.GetAllTools()
		legacyStats := map[string]interface{}{
			"total_tools":  len(allTools),
			"native_tools": len(allTools), // 暂定的に全てnativeとしてカウント
			"mcp_tools":    0,
		}
		toolsMetrics.LegacyToolsMetrics = legacyStats
	}

	return toolsMetrics
}

// ToolsMetrics - ツールアダプター固有のメトリクス
type ToolsMetrics struct {
	AdapterMetrics
	UnifiedRegistry     bool                   `json:"unified_registry"`
	LegacyRegistry      bool                   `json:"legacy_registry"`
	UnifiedToolsMetrics *tools.GlobalToolStats `json:"unified_metrics,omitempty"`
	LegacyToolsMetrics  interface{}            `json:"legacy_metrics,omitempty"`
}
