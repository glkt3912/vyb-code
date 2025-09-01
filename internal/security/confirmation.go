package security

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ユーザー確認インターフェース
type UserConfirmation interface {
	ConfirmCommand(command string, reason string) bool
	ConfirmRiskyOperation(operation string, details string) bool
}

// コンソールベースのユーザー確認実装
type ConsoleConfirmation struct {
	scanner *bufio.Scanner
}

// コンソール確認のコンストラクタ
func NewConsoleConfirmation() *ConsoleConfirmation {
	return &ConsoleConfirmation{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// コマンド実行の確認
func (c *ConsoleConfirmation) ConfirmCommand(command string, reason string) bool {
	fmt.Printf("\n⚠️  セキュリティ警告\n")
	fmt.Printf("検出された問題: %s\n", reason)
	fmt.Printf("実行予定のコマンド: %s\n", command)
	fmt.Printf("\nこのコマンドを実行しますか？ (y/N): ")

	return c.getUserConfirmation()
}

// リスクの高い操作の確認
func (c *ConsoleConfirmation) ConfirmRiskyOperation(operation string, details string) bool {
	fmt.Printf("\n🚨 高リスク操作の検出\n")
	fmt.Printf("操作: %s\n", operation)
	fmt.Printf("詳細: %s\n", details)
	fmt.Printf("\n本当に続行しますか？ (yes/NO): ")

	if !c.scanner.Scan() {
		return false
	}

	response := strings.TrimSpace(c.scanner.Text())
	return strings.ToLower(response) == "yes"
}

// ユーザーからの確認入力を取得
func (c *ConsoleConfirmation) getUserConfirmation() bool {
	if !c.scanner.Scan() {
		return false
	}

	response := strings.TrimSpace(c.scanner.Text())
	response = strings.ToLower(response)

	return response == "y" || response == "yes"
}

// セキュアなコマンド実行管理
type SecureExecutor struct {
	validator    *CommandValidator
	confirmation UserConfirmation
	constraints  *Constraints
	auditLog     *AuditLogger
	strictMode   bool // 厳格モード（疑わしいコマンドも拒否）
}

// セキュアな実行器のコンストラクタ
func NewSecureExecutor(constraints *Constraints, strictMode bool) *SecureExecutor {
	return &SecureExecutor{
		validator:    NewCommandValidator(),
		confirmation: NewConsoleConfirmation(),
		constraints:  constraints,
		auditLog:     NewAuditLogger(),
		strictMode:   strictMode,
	}
}

// コマンド実行前の包括的検証
func (s *SecureExecutor) ValidateAndConfirm(command string) (bool, error) {
	// 1. 基本的なセキュリティ制約チェック
	if err := s.constraints.IsCommandAllowed(command); err != nil {
		s.auditLog.LogBlocked(command, "基本制約違反", err.Error())
		return false, err
	}

	// 2. 高度なパターン検証
	result := s.validator.ValidateCommand(command)

	// 3. 監査ログに記録
	s.auditLog.LogValidation(command, result)

	// 4. 危険なコマンドは即座に拒否
	if result.RiskLevel == "dangerous" {
		s.auditLog.LogBlocked(command, "危険パターン", result.Reason)
		fmt.Printf("🚫 危険なコマンドが検出されました: %s\n", result.Reason)
		return false, fmt.Errorf("dangerous command detected: %s", result.Reason)
	}

	// 5. 疑わしいコマンドの処理
	if result.RiskLevel == "suspicious" {
		if s.strictMode {
			// 厳格モードでは疑わしいコマンドも拒否
			s.auditLog.LogBlocked(command, "疑わしいパターン（厳格モード）", result.Reason)
			fmt.Printf("🚫 厳格モードにより疑わしいコマンドが拒否されました\n")
			return false, fmt.Errorf("suspicious command blocked in strict mode")
		}

		// 通常モードではユーザー確認
		if !s.confirmation.ConfirmCommand(command, result.Reason) {
			s.auditLog.LogBlocked(command, "ユーザー拒否", "ユーザーによる実行拒否")
			return false, fmt.Errorf("command execution declined by user")
		}

		s.auditLog.LogApproved(command, "ユーザー承認", "疑わしいが承認済み")
	}

	// 6. 安全なコマンドは承認
	s.auditLog.LogApproved(command, "安全", "パターン検証通過")
	return true, nil
}

// LLM応答の分析と警告
func (s *SecureExecutor) AnalyzeLLMResponse(response string) []string {
	suspiciousCommands := s.validator.ValidateLLMResponse(response)

	if len(suspiciousCommands) > 0 {
		fmt.Printf("\n⚠️  LLM応答に疑わしいコマンドが含まれています:\n")
		for i, cmd := range suspiciousCommands {
			fmt.Printf("  %d. %s\n", i+1, cmd)
		}
		fmt.Printf("\n実行前に十分に内容を確認してください。\n")

		// 監査ログに記録
		for _, cmd := range suspiciousCommands {
			s.auditLog.LogSuspiciousLLM(response, cmd)
		}
	}

	return suspiciousCommands
}

// 厳格モードの設定
func (s *SecureExecutor) SetStrictMode(enabled bool) {
	s.strictMode = enabled
	s.auditLog.LogConfigChange("strict_mode", fmt.Sprintf("%v", enabled))
}
