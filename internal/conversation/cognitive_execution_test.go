package conversation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glkt/vyb-code/internal/ai"
	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/reasoning"
)

// MockLLMClient for cognitive execution testing
type MockLLMClient struct {
	responses   map[string]string
	shouldError bool
}

func (m *MockLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.shouldError {
		return "", fmt.Errorf("mock LLM error")
	}
	if response, exists := m.responses[prompt]; exists {
		return response, nil
	}
	return "Mock cognitive response", nil
}

func (m *MockLLMClient) GenerateResponse(ctx context.Context, request *ai.GenerateRequest) (*ai.GenerateResponse, error) {
	content, err := m.Generate(ctx, "")
	if err != nil {
		return nil, err
	}
	return &ai.GenerateResponse{Content: content}, nil
}

// TestCognitiveExecutionEngine_Integration tests full cognitive execution integration
func TestCognitiveExecutionEngine_Integration(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	projectPath := "/tmp/test-project"
	mockLLM := &MockLLMClient{
		responses: map[string]string{
			"cognitive_analysis": "認知分析完了",
		},
	}

	engine := NewCognitiveExecutionEngine(cfg, projectPath, mockLLM)

	// Test cognitive processing
	ctx := context.Background()
	input := "プロジェクトのGit状態を確認して"

	result, err := engine.ProcessUserInputCognitively(ctx, input)

	// Assertions
	if err != nil {
		t.Fatalf("ProcessUserInputCognitively failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.ExecutionResult == nil {
		t.Fatal("ExecutionResult should not be nil")
	}

	if result.ReasoningResult == nil {
		t.Fatal("ReasoningResult should not be nil")
	}

	if result.ProcessingStrategy == "" {
		t.Error("ProcessingStrategy should not be empty")
	}

	if result.TotalProcessingTime <= 0 {
		t.Error("TotalProcessingTime should be greater than 0")
	}
}

// TestCognitiveFallback tests fallback to traditional execution
func TestCognitiveFallback(t *testing.T) {
	cfg := &config.Config{}
	projectPath := "/tmp/test-project"

	// Create engine with mock LLM that returns error to trigger fallback
	mockLLM := &MockLLMClient{
		shouldError: true, // エラーを強制発生させてフォールバックをトリガー
		responses: map[string]string{
			"error": "mock error",
		},
	}
	engine := NewCognitiveExecutionEngine(cfg, projectPath, mockLLM)

	ctx := context.Background()
	input := "ls -la"

	result, err := engine.ProcessUserInputCognitively(ctx, input)

	if err != nil {
		t.Fatalf("Fallback execution failed: %v", err)
	}

	if result.ProcessingStrategy != "traditional_fallback" {
		t.Errorf("Expected fallback strategy, got %s", result.ProcessingStrategy)
	}
}

// TestLearningIntegration tests learning from execution results
func TestLearningIntegration(t *testing.T) {
	cfg := &config.Config{}
	projectPath := "/tmp/test-project"
	mockLLM := &MockLLMClient{}

	engine := NewCognitiveExecutionEngine(cfg, projectPath, mockLLM)

	// Create test execution result
	executionResult := &ExecutionResult{
		Command:   "git status",
		Output:    "On branch main\nnothing to commit",
		ExitCode:  0,
		Duration:  time.Millisecond * 100,
		Timestamp: time.Now(),
	}

	// Create test reasoning result
	reasoningResult := &reasoning.ReasoningResult{
		Confidence: 0.8,
		Session: &reasoning.ReasoningSession{
			SelectedSolution: &reasoning.ReasoningSolution{
				CreativityScore: 0.6,
			},
		},
	}

	// Test learning from execution
	learningOutcomes, err := engine.learnFromExecution(executionResult, reasoningResult, "git status")

	if err != nil {
		t.Fatalf("Learning from execution failed: %v", err)
	}

	if len(learningOutcomes) == 0 {
		t.Error("Should generate learning outcomes")
	}

	if learningOutcomes[0].Type == "" {
		t.Error("Learning outcome should have type")
	}
}

// TestIntelligentRecommendations tests intelligent recommendation generation
func TestIntelligentRecommendations(t *testing.T) {
	cfg := &config.Config{}
	projectPath := "/tmp/test-project"
	mockLLM := &MockLLMClient{}

	engine := NewCognitiveExecutionEngine(cfg, projectPath, mockLLM)

	// Create test context
	reasoningContext := &reasoning.ReasoningContext{
		DomainKnowledge: map[string]interface{}{
			"task": "project_optimization",
		},
	}

	ctx := context.Background()
	recommendations, err := engine.GenerateIntelligentRecommendations(ctx, reasoningContext)

	if err != nil {
		t.Fatalf("GenerateIntelligentRecommendations failed: %v", err)
	}

	// Should generate some recommendations (even if empty for now)
	if recommendations == nil {
		t.Error("Recommendations should not be nil")
	}
}

// TestCognitivePerformanceOptimization tests performance optimization
func TestCognitivePerformanceOptimization(t *testing.T) {
	cfg := &config.Config{}
	projectPath := "/tmp/test-project"
	mockLLM := &MockLLMClient{}

	engine := NewCognitiveExecutionEngine(cfg, projectPath, mockLLM)

	// Test optimization
	err := engine.OptimizeCognitivePerformance()

	if err != nil {
		t.Fatalf("OptimizeCognitivePerformance failed: %v", err)
	}

	// Verify optimization timestamp updated
	if engine.lastCognitiveUpdate.IsZero() {
		t.Error("lastCognitiveUpdate should be set after optimization")
	}
}

// TestExecutionStrategySelection tests dynamic execution strategy selection
func TestExecutionStrategySelection(t *testing.T) {
	cfg := &config.Config{}
	projectPath := "/tmp/test-project"
	mockLLM := &MockLLMClient{}

	engine := NewCognitiveExecutionEngine(cfg, projectPath, mockLLM)

	// Test different reasoning results
	testCases := []struct {
		name       string
		confidence float64
		creativity float64
		expected   string
	}{
		{
			name:       "High confidence - direct execution",
			confidence: 0.9,
			creativity: 0.3,
			expected:   "direct_execution",
		},
		{
			name:       "High creativity - creative exploration",
			confidence: 0.6,
			creativity: 0.8,
			expected:   "creative_exploration",
		},
		{
			name:       "Medium values - multi-tool workflow",
			confidence: 0.5,
			creativity: 0.5,
			expected:   "multi_tool_workflow",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reasoningResult := &reasoning.ReasoningResult{
				Confidence: tc.confidence,
				Session: &reasoning.ReasoningSession{
					SelectedSolution: &reasoning.ReasoningSolution{
						CreativityScore: tc.creativity,
					},
				},
			}

			strategy := engine.determineExecutionStrategy(reasoningResult, "test input")

			if strategy.Approach != tc.expected {
				t.Errorf("Expected approach %s, got %s", tc.expected, strategy.Approach)
			}
		})
	}
}

// TestCognitiveInsights tests cognitive insights generation
func TestCognitiveInsights(t *testing.T) {
	cfg := &config.Config{}
	projectPath := "/tmp/test-project"
	mockLLM := &MockLLMClient{}

	engine := NewCognitiveExecutionEngine(cfg, projectPath, mockLLM)

	insights, err := engine.GetCognitiveInsights()

	if err != nil {
		t.Fatalf("GetCognitiveInsights failed: %v", err)
	}

	if insights == nil {
		t.Fatal("Insights should not be nil")
	}

	if insights.Timestamp.IsZero() {
		t.Error("Insights timestamp should be set")
	}
}

// TestMultiToolWorkflow tests multi-tool workflow execution
func TestMultiToolWorkflow(t *testing.T) {
	cfg := &config.Config{}
	projectPath := "/tmp/test-project"
	mockLLM := &MockLLMClient{}

	engine := NewCognitiveExecutionEngine(cfg, projectPath, mockLLM)

	// Create strategy for multi-tool workflow
	strategy := &DynamicExecutionStrategy{
		Approach:        "multi_tool_workflow",
		Parallelization: true,
	}

	reasoningResult := &reasoning.ReasoningResult{
		Session: &reasoning.ReasoningSession{},
	}

	ctx := context.Background()
	result, err := engine.executeMultiToolWorkflow(ctx, strategy, reasoningResult)

	if err != nil {
		t.Fatalf("executeMultiToolWorkflow failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Command != "multi_tool_workflow" {
		t.Errorf("Expected command 'multi_tool_workflow', got %s", result.Command)
	}
}

// TestCreativeExploration tests creative exploration execution
func TestCreativeExploration(t *testing.T) {
	cfg := &config.Config{}
	projectPath := "/tmp/test-project"
	mockLLM := &MockLLMClient{}

	engine := NewCognitiveExecutionEngine(cfg, projectPath, mockLLM)

	strategy := &DynamicExecutionStrategy{
		Approach: "creative_exploration",
	}

	reasoningResult := &reasoning.ReasoningResult{
		Session: &reasoning.ReasoningSession{},
	}

	ctx := context.Background()
	result, err := engine.executeCreativeExploration(ctx, strategy, reasoningResult)

	if err != nil {
		t.Fatalf("executeCreativeExploration failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Command != "creative_exploration" {
		t.Errorf("Expected command 'creative_exploration', got %s", result.Command)
	}
}

// BenchmarkCognitiveExecution benchmarks cognitive execution performance
func BenchmarkCognitiveExecution(b *testing.B) {
	cfg := &config.Config{}
	projectPath := "/tmp/test-project"
	mockLLM := &MockLLMClient{}

	engine := NewCognitiveExecutionEngine(cfg, projectPath, mockLLM)

	ctx := context.Background()
	input := "git status"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.ProcessUserInputCognitively(ctx, input)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}
