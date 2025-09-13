package conversation

import (
	"strings"
	"testing"

	"github.com/glkt/vyb-code/internal/config"
)

// ExecutionEngine の基本機能テスト
func TestExecutionEngine(t *testing.T) {
	cfg := &config.Config{
		Features: &config.Features{
			ProactiveMode: true,
			VibeMode:      true,
		},
		Proactive: config.ProactiveConfig{
			Enabled:          true,
			Level:            3,
			AnalysisTimeout:  60,
			SmartSuggestions: true,
		},
	}

	ee := NewExecutionEngine(cfg, "/tmp")
	if ee == nil {
		t.Fatal("ExecutionEngine の作成に失敗")
	}
}

// 学習対象抽出のテスト
func TestExtractLearningTopic(t *testing.T) {
	cfg := &config.Config{
		Features: &config.Features{
			ProactiveMode: true,
		},
		Proactive: config.ProactiveConfig{
			Enabled: true,
			Level:   3,
		},
	}

	ee := NewExecutionEngine(cfg, "/tmp")

	tests := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			input:    "ExecutionEngine の仕組みを説明して",
			expected: "ExecutionEngine",
			desc:     "CamelCase detection",
		},
		{
			input:    "CLAUDE.md ファイルの内容は？",
			expected: "CLAUDE.md",
			desc:     "File name detection",
		},
		{
			input:    "README.md について教えて",
			expected: "README.md",
			desc:     "File name detection with Japanese",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := ee.extractLearningTopic(test.input)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s' for input: %s", test.expected, result, test.input)
			}
		})
	}
}

// 学習タイプ判定のテスト
func TestDetermineLearningType(t *testing.T) {
	cfg := &config.Config{
		Features:  &config.Features{ProactiveMode: true},
		Proactive: config.ProactiveConfig{Enabled: true, Level: 3},
	}

	ee := NewExecutionEngine(cfg, "/tmp")

	tests := []struct {
		input    string
		topic    string
		expected string
		desc     string
	}{
		{
			input:    "claude.mdファイルの説明をして",
			topic:    "CLAUDE.md",
			expected: "file_specific",
			desc:     "File-specific query",
		},
		{
			input:    "ExecutionEngineとは何ですか",
			topic:    "ExecutionEngine",
			expected: "code_analysis",
			desc:     "Code analysis query (camelCase detection)",
		},
		{
			input:    "プロジェクトのアーキテクチャを理解したい",
			topic:    "architecture",
			expected: "architecture_understanding",
			desc:     "Architecture understanding query",
		},
		{
			input:    "ExecutionEngine関数の実装を見たい",
			topic:    "ExecutionEngine",
			expected: "code_analysis",
			desc:     "Code analysis query",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			inputLower := strings.ToLower(test.input)
			result := ee.determineLearningType(inputLower, test.topic)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s' for input: %s", test.expected, result, test.input)
			}
		})
	}
}

// 安全なgrepコマンド構築のテスト
func TestBuildSafeGrepCommand(t *testing.T) {
	cfg := &config.Config{
		Features:  &config.Features{ProactiveMode: true},
		Proactive: config.ProactiveConfig{Enabled: true, Level: 3},
	}

	ee := NewExecutionEngine(cfg, "/tmp")

	tests := []struct {
		topic      string
		searchType string
		expected   string
		desc       string
	}{
		{
			topic:      "ExecutionEngine",
			searchType: "definition",
			expected:   "grep -r \"type ExecutionEngine\" . --include='*.go' | head -5",
			desc:       "Type definition search",
		},
		{
			topic:      "vibe",
			searchType: "usage",
			expected:   "grep -r \"vibe\" . --include='*.go' | head -10",
			desc:       "Usage search",
		},
		{
			topic:      "test'input",
			searchType: "general",
			expected:   "grep -ri \"test\\'input\" . --exclude-dir=.git --exclude-dir=vendor | head -10",
			desc:       "Special character escaping",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := ee.buildSafeGrepCommand(test.topic, test.searchType)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

// ユーザー意図分析のテスト
func TestAnalyzeUserIntent(t *testing.T) {
	cfg := &config.Config{
		Features: &config.Features{ProactiveMode: true},
		Proactive: config.ProactiveConfig{
			Enabled:          true,
			Level:            3,
			AnalysisTimeout:  60,
			SmartSuggestions: true,
		},
	}

	ee := NewExecutionEngine(cfg, "/tmp")

	tests := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			input:    "バイブモードについて教えて",
			expected: "execute_multi_tool_workflow",
			desc:     "Learning assistance trigger",
		},
		{
			input:    "git status を確認して",
			expected: "execute_explicit_command",
			desc:     "Git command execution",
		},
		{
			input:    "README.md の内容を読んで",
			expected: "read_file",
			desc:     "File reading request",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			analysis := ee.AnalyzeUserIntent(test.input)
			if analysis.RequiredAction != test.expected {
				t.Errorf("Expected action '%s', got '%s' for input: %s", test.expected, analysis.RequiredAction, test.input)
			}
		})
	}
}
