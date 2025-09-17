package config

import (
	"fmt"
	"strings"
)

// 応答スタイル（interactive/types.goから移動）
type ResponseStyle int

const (
	ResponseStyleConcise     ResponseStyle = iota // 簡潔
	ResponseStyleDetailed                         // 詳細
	ResponseStyleInteractive                      // インタラクティブ
	ResponseStyleEducational                      // 教育的
)

// Claude Code体験再現のためのプロンプト設定
type PromptConfig struct {
	// コア動作プロンプト
	SystemPrompt     string `json:"system_prompt"`     // ベースシステムプロンプト
	PersonalityStyle string `json:"personality_style"` // 応答スタイル
	TechnicalLevel   int    `json:"technical_level"`   // 技術レベル(1-10)

	// 動作パターン設定
	ToolUsageStyle      ToolUsageStyle `json:"tool_usage_style"`     // ツール使用パターン
	ResponsePattern     ResponseStyle  `json:"response_pattern"`     // 応答パターン
	ProactiveLevel      string         `json:"proactive_level"`      // プロアクティブ度
	ConciseTendency     float64        `json:"concise_tendency"`     // 簡潔性(0.0-1.0)
	ExplanationTendency float64        `json:"explanation_tendency"` // 説明度(0.0-1.0)

	// Claude Code特有の行動パターン
	EnableAutoToolUsage  bool `json:"enable_auto_tool_usage"` // 自動ツール使用
	EnableChainedActions bool `json:"enable_chained_actions"` // 連続アクション
	EnableContextAware   bool `json:"enable_context_aware"`   // コンテキスト認識
	EnableProgressReport bool `json:"enable_progress_report"` // 進捗報告

	// モデル別最適化
	ModelSpecific map[string]ModelPromptConfig `json:"model_specific"` // モデル別設定
}

// ツール使用スタイル
type ToolUsageStyle int

const (
	ToolUsageConservative ToolUsageStyle = iota // 保守的（明示的指示のみ）
	ToolUsageBalanced                           // バランス型（適度な自動実行）
	ToolUsageProactive                          // プロアクティブ（積極的実行）
	ToolUsageAggressive                         // アグレッシブ（最大限自動化）
)

func (t ToolUsageStyle) String() string {
	switch t {
	case ToolUsageConservative:
		return "conservative"
	case ToolUsageBalanced:
		return "balanced"
	case ToolUsageProactive:
		return "proactive"
	case ToolUsageAggressive:
		return "aggressive"
	default:
		return "balanced"
	}
}

// モデル別プロンプト設定
type ModelPromptConfig struct {
	SystemPromptSuffix  string   `json:"system_prompt_suffix"` // モデル特有の追加指示
	Temperature         float64  `json:"temperature"`          // 温度設定
	TopP                float64  `json:"top_p"`                // TopP設定
	SpecialInstructions []string `json:"special_instructions"` // 特別指示
}

// デフォルトプロンプト設定
func DefaultPromptConfig() *PromptConfig {
	return &PromptConfig{
		SystemPrompt:     generateClaudeCodeSystemPrompt(),
		PersonalityStyle: "helpful_engineer",
		TechnicalLevel:   7, // 中級〜上級エンジニア向け

		ToolUsageStyle:      ToolUsageBalanced,
		ResponsePattern:     ResponseStyleInteractive,
		ProactiveLevel:      "standard",
		ConciseTendency:     0.7, // やや簡潔
		ExplanationTendency: 0.6, // 適度な説明

		EnableAutoToolUsage:  true,
		EnableChainedActions: true,
		EnableContextAware:   true,
		EnableProgressReport: true,

		ModelSpecific: map[string]ModelPromptConfig{
			"qwen2.5-coder": {
				SystemPromptSuffix: "\n\n特に、Go言語とPython開発での実用的なアドバイスを重視してください。",
				Temperature:        0.7,
				TopP:               0.9,
				SpecialInstructions: []string{
					"コード例は必ず実行可能で完全なものを提供",
					"セキュリティベストプラクティスを常に考慮",
				},
			},
			"deepseek-coder": {
				SystemPromptSuffix: "\n\n詳細な技術的説明と最適化された実装を提供してください。",
				Temperature:        0.6,
				TopP:               0.85,
				SpecialInstructions: []string{
					"パフォーマンス最適化の観点を含める",
					"アーキテクチャの設計パターンを説明",
				},
			},
			"codellama": {
				SystemPromptSuffix: "\n\nステップバイステップの解説と実用的なコード例を重視してください。",
				Temperature:        0.8,
				TopP:               0.95,
				SpecialInstructions: []string{
					"初心者にも理解しやすい説明を含める",
					"コードの動作原理を丁寧に説明",
				},
			},
		},
	}
}

// Claude Code風システムプロンプト生成
func generateClaudeCodeSystemPrompt() string {
	return `You are vyb-code, a local AI coding assistant that provides Claude Code-equivalent functionality with privacy-focused development.

## Core Identity & Behavior
- You are an expert software engineer with deep knowledge across multiple programming languages
- Your primary goal is to help developers write better code efficiently and securely
- You work locally with complete privacy - no external data transmission
- You take direct action by reading files, making edits, running commands, and providing real-time assistance

## Interaction Style
- Be concise yet thorough - provide exactly what's needed without unnecessary verbosity
- Use tools proactively when they would help accomplish the user's goals
- Break down complex tasks into manageable steps
- Provide actionable suggestions with clear reasoning
- Ask clarifying questions only when truly necessary

## Tool Usage Philosophy
- Automatically use appropriate tools to understand context (Read, Grep, LS)
- Make file edits when requested or when they directly solve stated problems
- Run commands when they provide useful information or accomplish tasks
- Chain multiple tool uses efficiently to complete objectives
- Always validate changes with appropriate testing tools when available

## Response Patterns
- Lead with the most important information or action
- Use bullet points and structured formatting for clarity
- Include specific file paths, line numbers, and commands when relevant
- Provide both the "what" and the "why" but prioritize actionable content
- Show progress on multi-step tasks

## Code Quality Standards
- Follow established code conventions and patterns in the project
- Prioritize security, performance, and maintainability
- Use existing libraries and frameworks found in the codebase
- Write clean, well-structured, and documented code
- Never introduce vulnerabilities or anti-patterns

## Error Handling & Safety
- Validate inputs and check prerequisites before making changes
- Use appropriate error handling in all code
- Create backups or use version control for significant changes
- Test modifications when possible
- Fail gracefully with helpful error messages

## Context Awareness
- Analyze project structure and dependencies before suggesting changes
- Understand the user's current workflow and adapt accordingly
- Maintain awareness of the broader codebase architecture
- Consider the impact of changes on related components
- Remember context from previous interactions in the session

Remember: You are not just a chatbot - you are an active coding partner that reads, analyzes, edits, and executes to help developers accomplish their goals efficiently.`
}

// レスポンススタイル別のインストラクション生成
func (pc *PromptConfig) GetStyleInstructions() string {
	var instructions []string

	// 基本スタイル
	switch pc.ResponsePattern {
	case ResponseStyleConcise:
		instructions = append(instructions, "Keep responses brief and to the point.")
	case ResponseStyleDetailed:
		instructions = append(instructions, "Provide detailed explanations with examples.")
	case ResponseStyleInteractive:
		instructions = append(instructions, "Engage in interactive problem-solving.")
	case ResponseStyleEducational:
		instructions = append(instructions, "Focus on teaching concepts and best practices.")
	}

	// ツール使用スタイル
	switch pc.ToolUsageStyle {
	case ToolUsageConservative:
		instructions = append(instructions, "Use tools only when explicitly requested.")
	case ToolUsageBalanced:
		instructions = append(instructions, "Use tools when they help accomplish stated goals.")
	case ToolUsageProactive:
		instructions = append(instructions, "Proactively use tools to provide comprehensive assistance.")
	case ToolUsageAggressive:
		instructions = append(instructions, "Actively use all available tools to maximize productivity.")
	}

	// 簡潔性調整
	if pc.ConciseTendency > 0.7 {
		instructions = append(instructions, "Prioritize brevity - provide only essential information.")
	} else if pc.ConciseTendency < 0.3 {
		instructions = append(instructions, "Provide comprehensive explanations and context.")
	}

	// プロアクティブ機能
	if pc.EnableChainedActions {
		instructions = append(instructions, "Chain multiple actions to complete tasks efficiently.")
	}
	if pc.EnableProgressReport {
		instructions = append(instructions, "Report progress on multi-step tasks.")
	}

	return strings.Join(instructions, " ")
}

// モデル別システムプロンプト生成
func (pc *PromptConfig) GenerateSystemPrompt(modelName string) string {
	basePrompt := pc.SystemPrompt

	// スタイル指示を追加
	styleInstructions := pc.GetStyleInstructions()
	if styleInstructions != "" {
		basePrompt += "\n\n## Interaction Guidelines\n" + styleInstructions
	}

	// モデル特有の設定を追加
	if modelConfig, exists := pc.ModelSpecific[modelName]; exists {
		if modelConfig.SystemPromptSuffix != "" {
			basePrompt += modelConfig.SystemPromptSuffix
		}

		if len(modelConfig.SpecialInstructions) > 0 {
			basePrompt += "\n\n## Special Instructions\n"
			for _, instruction := range modelConfig.SpecialInstructions {
				basePrompt += "- " + instruction + "\n"
			}
		}
	}

	return basePrompt
}

// 技術レベルに応じた説明調整
func (pc *PromptConfig) GetExplanationLevel() string {
	switch {
	case pc.TechnicalLevel <= 3:
		return "Provide detailed explanations with examples for beginners."
	case pc.TechnicalLevel <= 6:
		return "Balance technical accuracy with clear explanations."
	case pc.TechnicalLevel <= 8:
		return "Use appropriate technical terminology with concise explanations."
	default:
		return "Assume deep technical knowledge and focus on advanced concepts."
	}
}

// プロンプト設定の検証
func (pc *PromptConfig) Validate() error {
	if pc.SystemPrompt == "" {
		return fmt.Errorf("system prompt cannot be empty")
	}
	if pc.TechnicalLevel < 1 || pc.TechnicalLevel > 10 {
		return fmt.Errorf("technical level must be between 1 and 10")
	}
	if pc.ConciseTendency < 0.0 || pc.ConciseTendency > 1.0 {
		return fmt.Errorf("concise tendency must be between 0.0 and 1.0")
	}
	if pc.ExplanationTendency < 0.0 || pc.ExplanationTendency > 1.0 {
		return fmt.Errorf("explanation tendency must be between 0.0 and 1.0")
	}
	return nil
}
