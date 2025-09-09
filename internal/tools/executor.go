package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

// コマンド実行結果を格納する構造体
type ExecutionResult struct {
	Command  string `json:"command"`   // 実行されたコマンド
	ExitCode int    `json:"exit_code"` // 終了コード
	Stdout   string `json:"stdout"`    // 標準出力
	Stderr   string `json:"stderr"`    // 標準エラー
	Duration string `json:"duration"`  // 実行時間
	TimedOut bool   `json:"timed_out"` // タイムアウトフラグ
}

// CommandExecutor - 廃止予定: BashToolベースの統一実装
// Deprecated: Use BashTool from claude_tools.go instead
type CommandExecutor struct {
	constraints *security.Constraints // セキュリティ制約
	workDir     string                // 作業ディレクトリ
	bashTool    *BashTool             // 内部でBashToolを使用
}

// コマンド実行ハンドラーを作成するコンストラクタ
// Deprecated: Use NewBashTool from claude_tools.go instead
func NewCommandExecutor(constraints *security.Constraints, workDir string) *CommandExecutor {
	return &CommandExecutor{
		constraints: constraints,
		workDir:     workDir,
		bashTool:    NewBashTool(constraints, workDir), // 内部でBashToolを使用
	}
}

// セキュアなコマンド実行 - BashToolベースの統一実装
// Deprecated: Use BashTool.Execute from claude_tools.go instead
func (e *CommandExecutor) Execute(command string) (*ExecutionResult, error) {
	startTime := time.Now()

	// BashToolを使用してコマンド実行
	toolResult, err := e.bashTool.Execute(command, "", int(e.constraints.MaxTimeout*1000)) // ミリ秒に変換

	duration := time.Since(startTime)

	// ToolExecutionResultをExecutionResultに変換
	result := &ExecutionResult{
		Command:  command,
		ExitCode: toolResult.ExitCode,
		Stdout:   toolResult.Content,
		Stderr:   "", // BashToolではstderrはContentに統合される
		Duration: duration.String(),
		TimedOut: toolResult.TimedOut, // BashToolのTimedOutフラグを使用
	}

	// エラーがある場合はstderrに設定
	if toolResult.IsError {
		result.Stderr = toolResult.Content
		result.Stdout = ""
	}

	return result, err
}

// 作業ディレクトリを変更
func (e *CommandExecutor) ChangeWorkingDirectory(newDir string) error {
	// セキュリティチェック：指定されたパスが許可されているか
	if !e.constraints.IsPathAllowed(newDir) {
		return fmt.Errorf("access denied: path outside workspace")
	}

	// ディレクトリが存在するかチェック
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", newDir)
	}

	e.workDir = newDir
	return nil
}

// 現在の作業ディレクトリを取得
func (e *CommandExecutor) GetWorkingDirectory() string {
	return e.workDir
}

// インタラクティブコマンドの実行（標準入力が必要なコマンド用）
func (e *CommandExecutor) ExecuteInteractive(command string, input string) (*ExecutionResult, error) {
	startTime := time.Now()

	// セキュリティチェック
	if err := e.constraints.IsCommandAllowed(command); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.constraints.MaxTimeout)*time.Second)
	defer cancel()

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = e.workDir
	cmd.Env = e.constraints.FilterEnvironment()

	// 標準入力を設定
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	result := &ExecutionResult{
		Command:  command,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration.String(),
		TimedOut: ctx.Err() == context.DeadlineExceeded,
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = -1
		}
	} else {
		result.ExitCode = 0
	}

	return result, nil
}
