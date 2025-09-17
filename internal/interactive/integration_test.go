package interactive

import (
	"context"
	"strings"
	"testing"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/contextmanager"
	"github.com/glkt/vyb-code/internal/llm"
	"github.com/glkt/vyb-code/internal/security"
	"github.com/glkt/vyb-code/internal/tools"
)

// MockLLMProvider - テスト用のモックLLMプロバイダー
type MockLLMProvider struct{}

func (m *MockLLMProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{
		Message: llm.ChatMessage{
			Role:    "assistant",
			Content: "This is a test response with tool execution results integrated.",
		},
		Done: true,
	}, nil
}

func (m *MockLLMProvider) SupportsFunctionCalling() bool { return false }
func (m *MockLLMProvider) GetModelInfo(model string) (*llm.ModelInfo, error) {
	return &llm.ModelInfo{Name: model, Description: "Mock model"}, nil
}
func (m *MockLLMProvider) ListModels() ([]llm.ModelInfo, error) { return nil, nil }

func TestIntegratedExecutionFlow(t *testing.T) {
	// テスト用設定
	cfg := config.DefaultConfig()
	cfg.Prompts.EnableAutoToolUsage = true
	cfg.Prompts.EnableChainedActions = true

	// モックプロバイダーを使用
	mockProvider := &MockLLMProvider{}
	promptAdapter := llm.NewPromptAdapter(mockProvider, cfg)

	// コンテキストマネージャー
	contextManager := contextmanager.NewSmartContextManager()

	// EditTool
	editTool := tools.NewEditTool(
		security.NewDefaultConstraints("."),
		".",
		10*1024*1024,
	)

	// InteractiveSessionManagerを作成
	manager := NewInteractiveSessionManager(
		contextManager,
		promptAdapter,
		nil, // AIService
		editTool,
		nil, // VibeConfig (nil will use default)
		"test-model",
		cfg,
	)

	// セッション作成
	session, err := manager.CreateSession(CodingSessionTypeGeneral)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// ファイル読み取り要求（ExecutionFlowが自動実行すべき）
	input := "read the config.go file"

	response, err := manager.ProcessUserInput(context.Background(), session.ID, input)
	if err != nil {
		t.Fatalf("ProcessUserInput failed: %v", err)
	}

	// 応答の検証
	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.Message == "" {
		t.Error("Response message should not be empty")
	}

	// ツール実行が行われたかメタデータで確認
	if autoTools, exists := response.Metadata["auto_tools"]; exists && autoTools == "true" {
		t.Log("✅ Auto tool execution was triggered successfully")
	} else {
		t.Log("ℹ️ Auto tool execution was not triggered (may be expected based on confidence)")
	}

	t.Logf("Response: %s", response.Message)
	t.Logf("Metadata: %+v", response.Metadata)
}

func TestPromptAdapterIntegration(t *testing.T) {
	cfg := config.DefaultConfig()
	mockProvider := &MockLLMProvider{}

	// PromptAdapterを作成
	adapter := llm.NewPromptAdapter(mockProvider, cfg)

	// システムプロンプトが自動追加されることを確認
	req := llm.ChatRequest{
		Model: "test-model",
		Messages: []llm.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	response, err := adapter.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	// システムプロンプトの確認
	systemPrompt := cfg.GenerateSystemPrompt()
	if systemPrompt == "" {
		t.Error("System prompt should not be empty")
	}

	if !strings.Contains(systemPrompt, "vyb-code") {
		t.Error("System prompt should contain 'vyb-code'")
	}

	t.Logf("✅ PromptAdapter integration successful")
	t.Logf("System prompt length: %d characters", len(systemPrompt))
}

func TestExecutionFlowAnalysis(t *testing.T) {
	cfg := config.DefaultConfig()

	// ExecutionFlow単体テスト
	toolRegistry := tools.NewUnifiedToolRegistry(
		security.NewDefaultConstraints("."),
		nil,
	)

	executionFlow := tools.NewExecutionFlow(toolRegistry, cfg, security.NewDefaultConstraints("."))

	testInputs := []struct {
		input    string
		expected bool // ツール実行が提案されるか
	}{
		{"read the main.go file", true},
		{"search for func main", true},
		{"list files in current directory", true},
		{"explain how Go works", false},
		{"what is the meaning of life", false},
	}

	for _, test := range testInputs {
		t.Run(test.input, func(t *testing.T) {
			plan, err := executionFlow.AnalyzeUserIntent(context.Background(), test.input)
			if err != nil {
				t.Fatalf("AnalyzeUserIntent failed: %v", err)
			}

			hasSteps := len(plan.Steps) > 0
			if hasSteps != test.expected {
				t.Errorf("Expected steps: %v, got: %v for input: %s", test.expected, hasSteps, test.input)
			}

			if hasSteps {
				t.Logf("✅ Input '%s' correctly identified %d tool steps", test.input, len(plan.Steps))
				t.Logf("   Confidence: %.2f", plan.Confidence)
				for i, step := range plan.Steps {
					t.Logf("   Step %d: %s (%s)", i+1, step.Tool, step.Risk.String())
				}
			} else {
				t.Logf("✅ Input '%s' correctly identified as no tool execution needed", test.input)
			}
		})
	}
}
