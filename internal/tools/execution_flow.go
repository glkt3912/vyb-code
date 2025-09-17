package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/security"
)

// ExecutionFlow - Claude Code風ツール実行フロー管理
type ExecutionFlow struct {
	registry    *UnifiedToolRegistry
	config      *config.Config
	constraints *security.Constraints

	// 実行統計
	executionHistory []ExecutionStep
	autoExecution    bool
	chainedExecution bool
}

// ExecutionStep - 実行ステップ記録
type ExecutionStep struct {
	StepID      string                 `json:"step_id"`
	Tool        string                 `json:"tool"`
	Parameters  map[string]interface{} `json:"parameters"`
	Result      *ToolResponse          `json:"result"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Success     bool                   `json:"success"`
	AutoTrigger bool                   `json:"auto_trigger"`
	Reasoning   string                 `json:"reasoning"`
}

// ExecutionPlan - 実行計画
type ExecutionPlan struct {
	Steps                []PlannedStep `json:"steps"`
	Reasoning            string        `json:"reasoning"`
	Confidence           float64       `json:"confidence"` // 0.0-1.0
	RequiresConfirmation bool          `json:"requires_confirmation"`
}

// PlannedStep - 計画されたステップ
type PlannedStep struct {
	Tool        string                 `json:"tool"`
	Parameters  map[string]interface{} `json:"parameters"`
	Description string                 `json:"description"`
	Rationale   string                 `json:"rationale"`
	Risk        RiskLevel              `json:"risk"`
}

// RiskLevel - リスクレベル
type RiskLevel int

const (
	RiskLevelSafe     RiskLevel = iota // 安全（読み取り専用等）
	RiskLevelLow                       // 低リスク（軽微な変更）
	RiskLevelMedium                    // 中リスク（重要な変更）
	RiskLevelHigh                      // 高リスク（破壊的変更）
	RiskLevelCritical                  // 重大（システム変更等）
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLevelSafe:
		return "safe"
	case RiskLevelLow:
		return "low"
	case RiskLevelMedium:
		return "medium"
	case RiskLevelHigh:
		return "high"
	case RiskLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// NewExecutionFlow - 新しい実行フローを作成
func NewExecutionFlow(registry *UnifiedToolRegistry, cfg *config.Config, constraints *security.Constraints) *ExecutionFlow {
	autoExecution := false
	chainedExecution := false

	if cfg.Prompts != nil {
		autoExecution = cfg.Prompts.EnableAutoToolUsage
		chainedExecution = cfg.Prompts.EnableChainedActions
	}

	return &ExecutionFlow{
		registry:         registry,
		config:           cfg,
		constraints:      constraints,
		executionHistory: make([]ExecutionStep, 0),
		autoExecution:    autoExecution,
		chainedExecution: chainedExecution,
	}
}

// AnalyzeUserIntent - ユーザー意図からツール実行計画を分析
func (ef *ExecutionFlow) AnalyzeUserIntent(ctx context.Context, userInput string) (*ExecutionPlan, error) {
	plan := &ExecutionPlan{
		Steps:      make([]PlannedStep, 0),
		Confidence: 0.0,
	}

	// Claude Code的なパターン解析
	steps := ef.extractToolSteps(userInput)

	for _, step := range steps {
		plannedStep := PlannedStep{
			Tool:        step.tool,
			Parameters:  step.parameters,
			Description: step.description,
			Rationale:   step.rationale,
			Risk:        ef.assessRisk(step.tool, step.parameters),
		}
		plan.Steps = append(plan.Steps, plannedStep)
	}

	// 実行計画の信頼度と必要性を判断
	plan.Confidence = ef.calculateConfidence(plan.Steps)
	plan.RequiresConfirmation = ef.requiresConfirmation(plan.Steps)
	plan.Reasoning = ef.generatePlanReasoning(userInput, plan.Steps)

	return plan, nil
}

// extractToolSteps - ユーザー入力からツール実行ステップを抽出
func (ef *ExecutionFlow) extractToolSteps(userInput string) []toolStepCandidate {
	var steps []toolStepCandidate

	inputLower := strings.ToLower(userInput)

	// ファイル読み取りパターン（元の文字列から抽出してケース保持）
	filePattern := regexp.MustCompile(`(?:read|show|display|view|check|see|look\s+at|examine)\s+(?:me\s+)?(?:the\s+)?(?:content\s+of\s+)?(?:file\s+)?([^\s]+\.[a-zA-Z]+)`)
	if matches := filePattern.FindAllStringSubmatch(userInput, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 1 {
				filePath := match[1]
				// 相対パスを正規化（セキュリティ要件を満たすため）
				if !filepath.IsAbs(filePath) && !strings.HasPrefix(filePath, "./") && !strings.HasPrefix(filePath, "/") {
					filePath = "./" + filePath
				}
				steps = append(steps, toolStepCandidate{
					tool: "read",
					parameters: map[string]interface{}{
						"file_path": filePath,
					},
					description: fmt.Sprintf("Read file %s", match[1]),
					rationale:   "User wants to examine file content",
				})
			}
		}
	}

	// ファイル検索パターン
	if matches := regexp.MustCompile(`(?:find|search|locate|grep)\s+(?:for\s+)?["\']?([^"'\s]+)["\']?`).FindAllStringSubmatch(inputLower, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 1 {
				steps = append(steps, toolStepCandidate{
					tool: "grep",
					parameters: map[string]interface{}{
						"pattern":     match[1],
						"output_mode": "files_with_matches",
					},
					description: fmt.Sprintf("Search for pattern '%s'", match[1]),
					rationale:   "User wants to find specific content",
				})
			}
		}
	}

	// ディレクトリリストパターン
	if regexp.MustCompile(`(?:list|show|ls)(?:\s+(?:me\s+)?(?:the\s+)?)?(?:files|dir|directory|folder)`).MatchString(inputLower) {
		steps = append(steps, toolStepCandidate{
			tool: "ls",
			parameters: map[string]interface{}{
				"path": ".",
			},
			description: "List current directory contents",
			rationale:   "User wants to see directory structure",
		})
	}

	// Git操作パターン
	if regexp.MustCompile(`(?:git\s+status|check\s+git|git\s+changes)`).MatchString(inputLower) {
		steps = append(steps, toolStepCandidate{
			tool: "bash",
			parameters: map[string]interface{}{
				"command":     "git status",
				"description": "Check git repository status",
			},
			description: "Check git status",
			rationale:   "User wants to see current git state",
		})
	}

	// コマンド実行パターン
	if matches := regexp.MustCompile(`(?:run|execute|exec)\s+["\']?([^"'\n]+)["\']?`).FindAllStringSubmatch(inputLower, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 1 {
				steps = append(steps, toolStepCandidate{
					tool: "bash",
					parameters: map[string]interface{}{
						"command":     match[1],
						"description": fmt.Sprintf("Execute command: %s", match[1]),
					},
					description: fmt.Sprintf("Execute: %s", match[1]),
					rationale:   "User explicitly requested command execution",
				})
			}
		}
	}

	return steps
}

// toolStepCandidate - ツール実行候補
type toolStepCandidate struct {
	tool        string
	parameters  map[string]interface{}
	description string
	rationale   string
}

// assessRisk - ツール実行のリスクレベルを評価
func (ef *ExecutionFlow) assessRisk(toolName string, parameters map[string]interface{}) RiskLevel {
	switch toolName {
	case "read", "ls", "grep":
		return RiskLevelSafe // 読み取り専用
	case "bash":
		if cmd, ok := parameters["command"].(string); ok {
			cmdLower := strings.ToLower(cmd)
			// 危険なコマンドパターン
			dangerousPatterns := []string{
				"rm ", "del ", "delete", "format", "mkfs",
				"dd ", "fdisk", "parted", "shutdown", "reboot",
				"chmod 777", "chown", "usermod", "passwd",
			}
			for _, pattern := range dangerousPatterns {
				if strings.Contains(cmdLower, pattern) {
					return RiskLevelCritical
				}
			}
			// 変更系コマンド
			changePatterns := []string{
				"git commit", "git push", "git merge", "git rebase",
				"npm install", "pip install", "cargo install",
				"make install", "sudo ",
			}
			for _, pattern := range changePatterns {
				if strings.Contains(cmdLower, pattern) {
					return RiskLevelMedium
				}
			}
			return RiskLevelLow // その他のコマンド
		}
		return RiskLevelLow
	case "edit", "write":
		return RiskLevelMedium // ファイル変更
	case "multiedit":
		return RiskLevelHigh // 複数ファイル変更
	default:
		return RiskLevelLow
	}
}

// calculateConfidence - 実行計画の信頼度を計算
func (ef *ExecutionFlow) calculateConfidence(steps []PlannedStep) float64 {
	if len(steps) == 0 {
		return 0.0
	}

	totalConfidence := 0.0
	for _, step := range steps {
		stepConfidence := 0.8 // 基本信頼度

		// ツール別調整
		switch step.Tool {
		case "read", "ls", "grep":
			stepConfidence = 0.9 // 安全で確実
		case "bash":
			stepConfidence = 0.7 // コマンド依存
		case "edit", "write":
			stepConfidence = 0.6 // 変更系は慎重
		}

		// リスク別調整
		switch step.Risk {
		case RiskLevelSafe:
			stepConfidence += 0.1
		case RiskLevelHigh, RiskLevelCritical:
			stepConfidence -= 0.2
		}

		totalConfidence += stepConfidence
	}

	return totalConfidence / float64(len(steps))
}

// requiresConfirmation - 確認が必要かどうか判断
func (ef *ExecutionFlow) requiresConfirmation(steps []PlannedStep) bool {
	for _, step := range steps {
		if step.Risk >= RiskLevelMedium {
			return true
		}
	}
	return false
}

// generatePlanReasoning - 実行計画の理由を生成
func (ef *ExecutionFlow) generatePlanReasoning(userInput string, steps []PlannedStep) string {
	if len(steps) == 0 {
		return "No automatic tool usage suggested for this request."
	}

	reasoning := fmt.Sprintf("Based on your request, I suggest %d tool operations:\n", len(steps))

	for i, step := range steps {
		reasoning += fmt.Sprintf("%d. %s (%s risk): %s\n",
			i+1, step.Description, step.Risk.String(), step.Rationale)
	}

	return reasoning
}

// ExecutePlan - 実行計画を実行
func (ef *ExecutionFlow) ExecutePlan(ctx context.Context, plan *ExecutionPlan) ([]ExecutionStep, error) {
	if !ef.autoExecution && plan.RequiresConfirmation {
		return nil, fmt.Errorf("execution requires user confirmation")
	}

	results := make([]ExecutionStep, 0, len(plan.Steps))

	for i, plannedStep := range plan.Steps {
		stepID := fmt.Sprintf("step_%d_%d", time.Now().Unix(), i)

		step := ExecutionStep{
			StepID:      stepID,
			Tool:        plannedStep.Tool,
			Parameters:  plannedStep.Parameters,
			StartTime:   time.Now(),
			AutoTrigger: true,
			Reasoning:   plannedStep.Rationale,
		}

		// ツール実行
		request := &ToolRequest{
			ToolName:   plannedStep.Tool,
			Parameters: plannedStep.Parameters,
		}

		response, err := ef.registry.ExecuteTool(ctx, request)
		step.EndTime = time.Now()

		if err != nil {
			step.Success = false
			step.Result = &ToolResponse{
				Success: false,
				Content: fmt.Sprintf("Error: %v", err),
				Error:   err.Error(),
			}
		} else {
			step.Success = true
			step.Result = response
		}

		results = append(results, step)
		ef.executionHistory = append(ef.executionHistory, step)

		// チェーン実行が無効化されている場合、最初のステップのみ実行
		if !ef.chainedExecution && i == 0 {
			break
		}

		// エラー時は以降のステップを中断
		if !step.Success {
			break
		}
	}

	return results, nil
}

// GetExecutionHistory - 実行履歴を取得
func (ef *ExecutionFlow) GetExecutionHistory() []ExecutionStep {
	return ef.executionHistory
}

// UpdateConfig - 設定を更新
func (ef *ExecutionFlow) UpdateConfig(cfg *config.Config) {
	ef.config = cfg
	if cfg.Prompts != nil {
		ef.autoExecution = cfg.Prompts.EnableAutoToolUsage
		ef.chainedExecution = cfg.Prompts.EnableChainedActions
	}
}
