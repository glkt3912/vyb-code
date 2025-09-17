package config

import (
	"testing"
)

func TestDefaultPromptConfig(t *testing.T) {
	cfg := DefaultPromptConfig()

	if cfg == nil {
		t.Fatal("DefaultPromptConfig returned nil")
	}

	if cfg.SystemPrompt == "" {
		t.Error("SystemPrompt should not be empty")
	}

	if cfg.TechnicalLevel < 1 || cfg.TechnicalLevel > 10 {
		t.Errorf("TechnicalLevel should be 1-10, got %d", cfg.TechnicalLevel)
	}

	if cfg.ConciseTendency < 0.0 || cfg.ConciseTendency > 1.0 {
		t.Errorf("ConciseTendency should be 0.0-1.0, got %f", cfg.ConciseTendency)
	}

	if cfg.ExplanationTendency < 0.0 || cfg.ExplanationTendency > 1.0 {
		t.Errorf("ExplanationTendency should be 0.0-1.0, got %f", cfg.ExplanationTendency)
	}
}

func TestPromptConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *PromptConfig
		expectError bool
	}{
		{
			name:        "valid config",
			config:      DefaultPromptConfig(),
			expectError: false,
		},
		{
			name: "empty system prompt",
			config: &PromptConfig{
				SystemPrompt:        "",
				TechnicalLevel:      5,
				ConciseTendency:     0.5,
				ExplanationTendency: 0.5,
			},
			expectError: true,
		},
		{
			name: "invalid technical level",
			config: &PromptConfig{
				SystemPrompt:        "test",
				TechnicalLevel:      15,
				ConciseTendency:     0.5,
				ExplanationTendency: 0.5,
			},
			expectError: true,
		},
		{
			name: "invalid concise tendency",
			config: &PromptConfig{
				SystemPrompt:        "test",
				TechnicalLevel:      5,
				ConciseTendency:     1.5,
				ExplanationTendency: 0.5,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestToolUsageStyle(t *testing.T) {
	tests := []struct {
		style    ToolUsageStyle
		expected string
	}{
		{ToolUsageConservative, "conservative"},
		{ToolUsageBalanced, "balanced"},
		{ToolUsageProactive, "proactive"},
		{ToolUsageAggressive, "aggressive"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.style.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.style.String())
			}
		})
	}
}

func TestGenerateSystemPrompt(t *testing.T) {
	cfg := DefaultPromptConfig()

	// テスト: デフォルトプロンプト生成
	prompt := cfg.GenerateSystemPrompt("qwen2.5-coder")
	if prompt == "" {
		t.Error("Generated system prompt should not be empty")
	}

	// テスト: モデル特有の指示が含まれているか
	if !contains(prompt, "Go言語とPython開発") {
		t.Error("Model-specific instructions should be included")
	}

	// テスト: 基本システムプロンプトが含まれているか
	if !contains(prompt, "vyb-code") {
		t.Error("Base system prompt should be included")
	}
}

func TestGetStyleInstructions(t *testing.T) {
	cfg := DefaultPromptConfig()
	cfg.ResponsePattern = ResponseStyleConcise
	cfg.ToolUsageStyle = ToolUsageProactive

	instructions := cfg.GetStyleInstructions()
	if instructions == "" {
		t.Error("Style instructions should not be empty")
	}

	// 簡潔モードの指示が含まれているかチェック
	if !contains(instructions, "brief") {
		t.Error("Concise style instructions should be included")
	}
}

func TestGetExplanationLevel(t *testing.T) {
	cfg := DefaultPromptConfig()

	tests := []struct {
		level    int
		contains string
	}{
		{2, "detailed explanations"},
		{5, "Balance technical"},
		{8, "appropriate technical"},
		{10, "deep technical"},
	}

	for _, tt := range tests {
		t.Run("level_"+string(rune(tt.level)), func(t *testing.T) {
			cfg.TechnicalLevel = tt.level
			explanation := cfg.GetExplanationLevel()
			if !contains(explanation, tt.contains) {
				t.Errorf("Level %d should contain '%s', got '%s'", tt.level, tt.contains, explanation)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
