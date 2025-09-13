package reasoning

import (
	"context"
	"testing"
	"time"

	"github.com/glkt/vyb-code/internal/ai"
	"github.com/glkt/vyb-code/internal/config"
)

// MockLLMClient はテスト用のモックLLMクライアント
type MockLLMClient struct {
	responses map[string]string
}

func (m *MockLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if response, exists := m.responses[prompt]; exists {
		return response, nil
	}
	return "Mock AI response for: " + prompt[:min(50, len(prompt))], nil
}

func (m *MockLLMClient) GenerateWithOptions(ctx context.Context, prompt string, options ai.GenerationOptions) (string, error) {
	return m.Generate(ctx, prompt)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestCognitiveEngine_BasicReasoning tests basic cognitive reasoning functionality
func TestCognitiveEngine_BasicReasoning(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	mockLLM := &MockLLMClient{
		responses: map[string]string{
			"analyze_intent": "ユーザーはファイル操作について質問している",
			"reasoning":      "論理的推論を実行中",
		},
	}

	engine := NewCognitiveEngine(cfg, mockLLM)

	// Test case 1: Simple user input processing
	ctx := context.Background()
	input := "プロジェクトのファイル一覧を表示して"

	result, err := engine.ProcessUserInput(ctx, input)

	// Assertions
	if err != nil {
		t.Fatalf("ProcessUserInput failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Session == nil {
		t.Fatal("Session should not be nil")
	}

	if result.Session.UserInput != input {
		t.Errorf("Expected input '%s', got '%s'", input, result.Session.UserInput)
	}

	if result.Confidence <= 0 {
		t.Error("Confidence should be greater than 0")
	}

	if result.ProcessingTime <= 0 {
		t.Error("ProcessingTime should be greater than 0")
	}
}

// TestSemanticIntentAnalysis tests semantic intent understanding
func TestSemanticIntentAnalysis(t *testing.T) {
	cfg := &config.Config{}
	mockLLM := &MockLLMClient{}
	engine := NewCognitiveEngine(cfg, mockLLM)

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "File operation request",
			input:    "ファイルを作成したい",
			expected: "file_creation",
		},
		{
			name:     "Git operation request",
			input:    "コミットを作成して",
			expected: "git_commit",
		},
		{
			name:     "Information request",
			input:    "プロジェクトの構造を教えて",
			expected: "information_request",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			intent := engine.analyzeSemanticIntent(tc.input)

			if intent == nil {
				t.Fatal("Intent should not be nil")
			}

			if intent.PrimaryGoal == "" {
				t.Error("PrimaryGoal should not be empty")
			}

			if intent.Confidence <= 0 {
				t.Error("Confidence should be greater than 0")
			}
		})
	}
}

// TestInferenceEngine_LogicalReasoning tests logical reasoning capabilities
func TestInferenceEngine_LogicalReasoning(t *testing.T) {
	cfg := &config.Config{}
	mockLLM := &MockLLMClient{}
	engine := NewInferenceEngine(cfg, mockLLM)

	// Create test intent and context
	intent := &SemanticIntent{
		PrimaryGoal: "file_analysis",
		Confidence:  0.8,
	}

	context := &ReasoningContext{
		CurrentTask: "analyze_project_files",
	}

	// Test inference chain building
	ctx := context.Background()
	chain, err := engine.BuildInferenceChain(ctx, intent, context)

	if err != nil {
		t.Fatalf("BuildInferenceChain failed: %v", err)
	}

	if chain == nil {
		t.Fatal("Chain should not be nil")
	}

	if chain.Approach == "" {
		t.Error("Approach should not be empty")
	}

	if chain.Confidence <= 0 {
		t.Error("Confidence should be greater than 0")
	}
}

// TestDynamicProblemSolver_CreativeSolutions tests creative problem solving
func TestDynamicProblemSolver_CreativeSolutions(t *testing.T) {
	cfg := &config.Config{}
	mockLLM := &MockLLMClient{}
	solver := NewDynamicProblemSolver(cfg, mockLLM)

	// Define test problem
	problem := &ProblemDefinition{
		Description: "ユーザーが求めているファイル操作を効率的に実行する方法",
		Constraints: []string{"安全性を確保", "パフォーマンスを重視"},
		Goals:       []string{"迅速な実行", "エラー処理"},
	}

	context := &ReasoningContext{
		CurrentTask: "file_operation_optimization",
	}

	// Test solution generation
	ctx := context.Background()
	result, err := solver.GenerateSolution(ctx, problem, context)

	if err != nil {
		t.Fatalf("GenerateSolution failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.SelectedSolution == nil {
		t.Fatal("SelectedSolution should not be nil")
	}

	if result.SelectedSolution.Confidence <= 0 {
		t.Error("Solution confidence should be greater than 0")
	}

	if len(result.AlternativeSolutions) == 0 {
		t.Error("Should generate alternative solutions")
	}
}

// TestAdaptiveLearner_LearningFromInteraction tests learning capabilities
func TestAdaptiveLearner_LearningFromInteraction(t *testing.T) {
	cfg := &config.Config{}
	learner := NewAdaptiveLearner(cfg)

	// Create test interaction
	interaction := &ConversationTurn{
		ID:              "test_turn_1",
		Content:         "ファイル一覧を表示して",
		CognitiveLoad:   0.6,
		ResponseQuality: 0.8,
		Timestamp:       time.Now(),
	}

	outcome := &InteractionOutcome{
		Success: true,
	}

	context := &ReasoningContext{
		CurrentTask: "file_listing",
	}

	// Test learning
	learningResult, err := learner.LearnFromInteraction(interaction, outcome, context)

	if err != nil {
		t.Fatalf("LearnFromInteraction failed: %v", err)
	}

	if learningResult == nil {
		t.Fatal("Learning result should not be nil")
	}

	if learningResult.Type == "" {
		t.Error("Learning type should not be empty")
	}

	if learningResult.ConfidenceLevel <= 0 {
		t.Error("Learning confidence should be greater than 0")
	}
}

// TestContextualMemory_MemoryOperations tests memory management
func TestContextualMemory_MemoryOperations(t *testing.T) {
	cfg := &config.Config{}
	memory := NewContextualMemory(cfg)

	// Test storing interaction
	turn := &ConversationTurn{
		ID:              "test_turn_2",
		Content:         "プロジェクト分析を実行",
		CognitiveLoad:   0.7,
		ResponseQuality: 0.9,
		Timestamp:       time.Now(),
	}

	err := memory.StoreInteraction(turn)
	if err != nil {
		t.Fatalf("StoreInteraction failed: %v", err)
	}

	// Test retrieving context
	intent := &SemanticIntent{
		PrimaryGoal: "project_analysis",
		Confidence:  0.8,
	}

	relevantContext, err := memory.RetrieveRelevantContext(intent)
	if err != nil {
		t.Fatalf("RetrieveRelevantContext failed: %v", err)
	}

	if relevantContext == nil {
		t.Fatal("Relevant context should not be nil")
	}

	if relevantContext.Intent != intent {
		t.Error("Context intent should match input intent")
	}
}

// TestCognitiveIntegration tests full cognitive system integration
func TestCognitiveIntegration(t *testing.T) {
	cfg := &config.Config{}
	mockLLM := &MockLLMClient{
		responses: map[string]string{
			"complex_analysis": "複合的な分析を実行しています",
		},
	}

	// Initialize all components
	cognitiveEngine := NewCognitiveEngine(cfg, mockLLM)

	// Test complex reasoning scenario
	ctx := context.Background()
	complexInput := "プロジェクトの依存関係を分析して、潜在的な問題を特定し、最適化案を提示して"

	result, err := cognitiveEngine.ProcessUserInput(ctx, complexInput)

	if err != nil {
		t.Fatalf("Complex reasoning failed: %v", err)
	}

	// Verify comprehensive reasoning
	if result.Session.InferenceChains == nil || len(result.Session.InferenceChains) == 0 {
		t.Error("Should generate inference chains for complex reasoning")
	}

	if result.Session.Solutions == nil || len(result.Session.Solutions) == 0 {
		t.Error("Should generate multiple solutions for complex problems")
	}

	if result.Session.SelectedSolution == nil {
		t.Error("Should select optimal solution")
	}

	if result.Confidence <= 0.5 {
		t.Error("Complex reasoning should have reasonable confidence")
	}
}

// TestPerformanceMetrics tests cognitive performance measurement
func TestPerformanceMetrics(t *testing.T) {
	cfg := &config.Config{}
	mockLLM := &MockLLMClient{}
	engine := NewCognitiveEngine(cfg, mockLLM)

	// Measure processing time
	start := time.Now()

	ctx := context.Background()
	input := "簡単なタスク処理"

	result, err := engine.ProcessUserInput(ctx, input)

	processingDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Performance test failed: %v", err)
	}

	// Performance assertions
	if processingDuration > 5*time.Second {
		t.Errorf("Processing took too long: %v", processingDuration)
	}

	if result.ProcessingTime <= 0 {
		t.Error("ProcessingTime should be recorded")
	}

	// Verify efficiency metrics
	if result.Session.SelectedSolution != nil {
		if result.Session.SelectedSolution.Efficiency <= 0 {
			t.Error("Solution efficiency should be measured")
		}
	}
}

// Benchmark tests for performance evaluation
func BenchmarkCognitiveEngine_SimpleReasoning(b *testing.B) {
	cfg := &config.Config{}
	mockLLM := &MockLLMClient{}
	engine := NewCognitiveEngine(cfg, mockLLM)

	ctx := context.Background()
	input := "ファイル操作"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.ProcessUserInput(ctx, input)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func BenchmarkCognitiveEngine_ComplexReasoning(b *testing.B) {
	cfg := &config.Config{}
	mockLLM := &MockLLMClient{}
	engine := NewCognitiveEngine(cfg, mockLLM)

	ctx := context.Background()
	complexInput := "プロジェクト全体の分析を行い、依存関係を調査し、最適化提案を生成して実行計画を策定する"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.ProcessUserInput(ctx, complexInput)
		if err != nil {
			b.Fatalf("Complex benchmark failed: %v", err)
		}
	}
}
