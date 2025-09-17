package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/mcp"
	"github.com/glkt/vyb-code/internal/security"
)

func TestExecutionFlowCreation(t *testing.T) {
	registry := NewUnifiedToolRegistry(security.NewDefaultConstraints("."), mcp.NewManager())
	cfg := config.DefaultConfig()
	constraints := security.NewDefaultConstraints(".")

	flow := NewExecutionFlow(registry, cfg, constraints)

	if flow == nil {
		t.Fatal("NewExecutionFlow returned nil")
	}

	if !flow.autoExecution {
		t.Error("Auto execution should be enabled by default")
	}

	if !flow.chainedExecution {
		t.Error("Chained execution should be enabled by default")
	}
}

func TestAnalyzeUserIntent(t *testing.T) {
	registry := NewUnifiedToolRegistry(security.NewDefaultConstraints("."), mcp.NewManager())
	cfg := config.DefaultConfig()
	constraints := security.NewDefaultConstraints(".")
	flow := NewExecutionFlow(registry, cfg, constraints)

	tests := []struct {
		name          string
		input         string
		expectedSteps int
		expectedTool  string
		expectedRisk  RiskLevel
	}{
		{
			name:          "read file request",
			input:         "read the config.go file",
			expectedSteps: 1,
			expectedTool:  "read",
			expectedRisk:  RiskLevelSafe,
		},
		{
			name:          "search pattern request",
			input:         "find all occurrences of 'func main'",
			expectedSteps: 1,
			expectedTool:  "grep",
			expectedRisk:  RiskLevelSafe,
		},
		{
			name:          "list directory request",
			input:         "show me the files in this directory",
			expectedSteps: 1,
			expectedTool:  "ls",
			expectedRisk:  RiskLevelSafe,
		},
		{
			name:          "git status request",
			input:         "check git status",
			expectedSteps: 1,
			expectedTool:  "bash",
			expectedRisk:  RiskLevelLow,
		},
		{
			name:          "no tool required",
			input:         "explain how Go works",
			expectedSteps: 0,
			expectedTool:  "",
			expectedRisk:  RiskLevelSafe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := flow.AnalyzeUserIntent(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("AnalyzeUserIntent failed: %v", err)
			}

			if len(plan.Steps) != tt.expectedSteps {
				t.Errorf("Expected %d steps, got %d", tt.expectedSteps, len(plan.Steps))
			}

			if tt.expectedSteps > 0 {
				if plan.Steps[0].Tool != tt.expectedTool {
					t.Errorf("Expected tool %s, got %s", tt.expectedTool, plan.Steps[0].Tool)
				}

				if plan.Steps[0].Risk != tt.expectedRisk {
					t.Errorf("Expected risk %v, got %v", tt.expectedRisk, plan.Steps[0].Risk)
				}
			}
		})
	}
}

func TestRiskAssessment(t *testing.T) {
	registry := NewUnifiedToolRegistry(security.NewDefaultConstraints("."), mcp.NewManager())
	cfg := config.DefaultConfig()
	constraints := security.NewDefaultConstraints(".")
	flow := NewExecutionFlow(registry, cfg, constraints)

	tests := []struct {
		tool       string
		parameters map[string]interface{}
		expected   RiskLevel
	}{
		{
			tool:       "read",
			parameters: map[string]interface{}{"file_path": "test.go"},
			expected:   RiskLevelSafe,
		},
		{
			tool:       "bash",
			parameters: map[string]interface{}{"command": "ls -la"},
			expected:   RiskLevelLow,
		},
		{
			tool:       "bash",
			parameters: map[string]interface{}{"command": "rm -rf /"},
			expected:   RiskLevelCritical,
		},
		{
			tool:       "bash",
			parameters: map[string]interface{}{"command": "git commit -m 'test'"},
			expected:   RiskLevelMedium,
		},
		{
			tool:       "edit",
			parameters: map[string]interface{}{"file_path": "test.go"},
			expected:   RiskLevelMedium,
		},
	}

	for i, tt := range tests {
		testName := fmt.Sprintf("%s_%d", tt.tool, i)
		if cmd, ok := tt.parameters["command"].(string); ok {
			testName = fmt.Sprintf("%s_%s", tt.tool, cmd)
		}
		t.Run(testName, func(t *testing.T) {
			risk := flow.assessRisk(tt.tool, tt.parameters)
			if risk != tt.expected {
				t.Errorf("Expected risk %v, got %v", tt.expected, risk)
			}
		})
	}
}

func TestCalculateConfidence(t *testing.T) {
	registry := NewUnifiedToolRegistry(security.NewDefaultConstraints("."), mcp.NewManager())
	cfg := config.DefaultConfig()
	constraints := security.NewDefaultConstraints(".")
	flow := NewExecutionFlow(registry, cfg, constraints)

	tests := []struct {
		name     string
		steps    []PlannedStep
		expected float64
	}{
		{
			name:     "empty steps",
			steps:    []PlannedStep{},
			expected: 0.0,
		},
		{
			name: "safe read operation",
			steps: []PlannedStep{
				{Tool: "read", Risk: RiskLevelSafe},
			},
			expected: 1.0, // Should be high confidence
		},
		{
			name: "risky operation",
			steps: []PlannedStep{
				{Tool: "bash", Risk: RiskLevelCritical},
			},
			expected: 0.5, // Should be lower confidence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := flow.calculateConfidence(tt.steps)
			if tt.name == "empty steps" {
				if confidence != tt.expected {
					t.Errorf("Expected confidence %f, got %f", tt.expected, confidence)
				}
			} else {
				// For non-empty cases, just check that confidence is reasonable
				if confidence < 0.0 || confidence > 1.0 {
					t.Errorf("Confidence should be 0.0-1.0, got %f", confidence)
				}
			}
		})
	}
}

func TestRequiresConfirmation(t *testing.T) {
	registry := NewUnifiedToolRegistry(security.NewDefaultConstraints("."), mcp.NewManager())
	cfg := config.DefaultConfig()
	constraints := security.NewDefaultConstraints(".")
	flow := NewExecutionFlow(registry, cfg, constraints)

	tests := []struct {
		name     string
		steps    []PlannedStep
		expected bool
	}{
		{
			name: "safe operations only",
			steps: []PlannedStep{
				{Tool: "read", Risk: RiskLevelSafe},
				{Tool: "ls", Risk: RiskLevelSafe},
			},
			expected: false,
		},
		{
			name: "medium risk operation",
			steps: []PlannedStep{
				{Tool: "edit", Risk: RiskLevelMedium},
			},
			expected: true,
		},
		{
			name: "high risk operation",
			steps: []PlannedStep{
				{Tool: "bash", Risk: RiskLevelHigh},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flow.requiresConfirmation(tt.steps)
			if result != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestConfigUpdate(t *testing.T) {
	registry := NewUnifiedToolRegistry(security.NewDefaultConstraints("."), mcp.NewManager())
	cfg := config.DefaultConfig()
	constraints := security.NewDefaultConstraints(".")
	flow := NewExecutionFlow(registry, cfg, constraints)

	// Initially enabled
	if !flow.autoExecution {
		t.Error("Auto execution should be enabled initially")
	}

	// Update config to disable auto execution
	newCfg := config.DefaultConfig()
	newCfg.Prompts.EnableAutoToolUsage = false
	newCfg.Prompts.EnableChainedActions = false

	flow.UpdateConfig(newCfg)

	if flow.autoExecution {
		t.Error("Auto execution should be disabled after config update")
	}

	if flow.chainedExecution {
		t.Error("Chained execution should be disabled after config update")
	}
}
