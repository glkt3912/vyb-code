package llm

import (
	"context"
	"fmt"

	"github.com/glkt/vyb-code/internal/config"
)

// PromptAdapter - プロンプト統合機能付きLLMプロバイダーアダプター
type PromptAdapter struct {
	provider Provider
	config   *config.Config
}

// NewPromptAdapter は新しいプロンプトアダプターを作成
func NewPromptAdapter(provider Provider, cfg *config.Config) *PromptAdapter {
	return &PromptAdapter{
		provider: provider,
		config:   cfg,
	}
}

// Chat はシステムプロンプトを自動追加してチャットリクエストを送信
func (pa *PromptAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// システムプロンプトを生成
	systemPrompt := pa.config.GenerateSystemPrompt()

	// メッセージの先頭にシステムプロンプトを追加（存在しない場合）
	messages := pa.ensureSystemMessage(req.Messages, systemPrompt)

	// モデル特有の設定を適用
	enhancedReq := pa.enhanceRequest(req, messages)

	// 元のプロバイダーに委譲
	return pa.provider.Chat(ctx, enhancedReq)
}

// ensureSystemMessage はシステムメッセージの存在を保証
func (pa *PromptAdapter) ensureSystemMessage(messages []ChatMessage, systemPrompt string) []ChatMessage {
	// システムメッセージが既に存在するかチェック
	hasSystemMessage := false
	for _, msg := range messages {
		if msg.Role == "system" {
			hasSystemMessage = true
			break
		}
	}

	// システムメッセージが存在しない場合は追加
	if !hasSystemMessage {
		systemMsg := ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		}
		// システムメッセージを先頭に追加
		return append([]ChatMessage{systemMsg}, messages...)
	}

	return messages
}

// enhanceRequest はモデル特有の設定でリクエストを強化
func (pa *PromptAdapter) enhanceRequest(originalReq ChatRequest, messages []ChatMessage) ChatRequest {
	enhancedReq := originalReq
	enhancedReq.Messages = messages

	// モデル特有の設定を取得
	if modelConfig, exists := pa.config.GetModelPromptConfig(originalReq.Model); exists {
		// Temperature設定
		if enhancedReq.Temperature == nil && modelConfig.Temperature > 0 {
			enhancedReq.Temperature = &modelConfig.Temperature
		}

		// TopP設定
		if enhancedReq.TopP == nil && modelConfig.TopP > 0 {
			enhancedReq.TopP = &modelConfig.TopP
		}
	}

	// 基本設定からフォールバック
	if enhancedReq.Temperature == nil {
		temp := pa.config.Temperature
		enhancedReq.Temperature = &temp
	}

	if enhancedReq.MaxTokens == nil && pa.config.MaxTokens > 0 {
		enhancedReq.MaxTokens = &pa.config.MaxTokens
	}

	return enhancedReq
}

// SupportsFunctionCalling は元のプロバイダーに委譲
func (pa *PromptAdapter) SupportsFunctionCalling() bool {
	return pa.provider.SupportsFunctionCalling()
}

// GetModelInfo は元のプロバイダーに委譲
func (pa *PromptAdapter) GetModelInfo(model string) (*ModelInfo, error) {
	return pa.provider.GetModelInfo(model)
}

// ListModels は元のプロバイダーに委譲
func (pa *PromptAdapter) ListModels() ([]ModelInfo, error) {
	return pa.provider.ListModels()
}

// UpdateConfig は設定を更新
func (pa *PromptAdapter) UpdateConfig(cfg *config.Config) {
	pa.config = cfg
}

// GetCurrentSystemPrompt は現在のシステムプロンプトを取得
func (pa *PromptAdapter) GetCurrentSystemPrompt() string {
	return pa.config.GenerateSystemPrompt()
}

// ValidatePromptConfig はプロンプト設定を検証
func (pa *PromptAdapter) ValidatePromptConfig() error {
	if pa.config.Prompts == nil {
		return fmt.Errorf("prompt config is nil")
	}
	return pa.config.Prompts.Validate()
}

// GetEffectiveConfig は実際に使用される設定を取得（デバッグ用）
func (pa *PromptAdapter) GetEffectiveConfig(modelName string) map[string]interface{} {
	effectiveConfig := make(map[string]interface{})

	// 基本設定
	effectiveConfig["base_temperature"] = pa.config.Temperature
	effectiveConfig["base_max_tokens"] = pa.config.MaxTokens
	effectiveConfig["base_model"] = pa.config.Model

	// プロンプト設定
	if pa.config.Prompts != nil {
		effectiveConfig["tool_usage_style"] = pa.config.Prompts.ToolUsageStyle.String()
		effectiveConfig["response_pattern"] = pa.config.Prompts.ResponsePattern
		effectiveConfig["technical_level"] = pa.config.Prompts.TechnicalLevel
		effectiveConfig["concise_tendency"] = pa.config.Prompts.ConciseTendency
	}

	// モデル特有の設定
	if modelConfig, exists := pa.config.GetModelPromptConfig(modelName); exists {
		effectiveConfig["model_temperature"] = modelConfig.Temperature
		effectiveConfig["model_top_p"] = modelConfig.TopP
		effectiveConfig["special_instructions"] = modelConfig.SpecialInstructions
	}

	return effectiveConfig
}
