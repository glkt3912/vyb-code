package tools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/glkt/vyb-code/internal/security"
)

// Git操作を管理する構造体
type GitOperations struct {
	executor *CommandExecutor // コマンド実行ハンドラー
	repoPath string           // リポジトリのパス
}

// Git操作ハンドラーを作成するコンストラクタ
func NewGitOperations(constraints *security.Constraints, repoPath string) *GitOperations {
	return &GitOperations{
		executor: NewCommandExecutor(constraints, repoPath),
		repoPath: repoPath,
	}
}

// リポジトリの状態を取得
func (g *GitOperations) GetStatus() (*ExecutionResult, error) {
	return g.executor.Execute("git status --porcelain")
}

// ブランチ一覧を取得
func (g *GitOperations) GetBranches() (*ExecutionResult, error) {
	return g.executor.Execute("git branch -a")
}

// 現在のブランチ名を取得
func (g *GitOperations) GetCurrentBranch() (string, error) {
	result, err := g.executor.Execute("git rev-parse --abbrev-ref HEAD")
	if err != nil {
		return "", err
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("git command failed: %s", result.Stderr)
	}

	return strings.TrimSpace(result.Stdout), nil
}

// 新しいブランチを作成してチェックアウト
func (g *GitOperations) CreateAndCheckoutBranch(branchName string) (*ExecutionResult, error) {
	// ブランチ名のバリデーション
	if err := g.validateBranchName(branchName); err != nil {
		return nil, err
	}

	return g.executor.Execute(fmt.Sprintf("git checkout -b %s", branchName))
}

// 既存のブランチにチェックアウト
func (g *GitOperations) CheckoutBranch(branchName string) (*ExecutionResult, error) {
	if err := g.validateBranchName(branchName); err != nil {
		return nil, err
	}

	return g.executor.Execute(fmt.Sprintf("git checkout %s", branchName))
}

// ファイルをステージング
func (g *GitOperations) AddFiles(files []string) (*ExecutionResult, error) {
	// ファイルパスの検証
	for _, file := range files {
		if !g.isPathSafe(file) {
			return nil, fmt.Errorf("unsafe file path: %s", file)
		}
	}

	if len(files) == 0 {
		return g.executor.Execute("git add .")
	}

	command := fmt.Sprintf("git add %s", strings.Join(files, " "))
	return g.executor.Execute(command)
}

// コミットを作成
func (g *GitOperations) Commit(message string) (*ExecutionResult, error) {
	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("commit message cannot be empty")
	}

	// コミットメッセージをエスケープ
	escapedMessage := strings.ReplaceAll(message, "\"", "\\\"")
	command := fmt.Sprintf("git commit -m \"%s\"", escapedMessage)

	return g.executor.Execute(command)
}

// 差分を取得
func (g *GitOperations) GetDiff(options string) (*ExecutionResult, error) {
	command := "git diff"
	if options != "" {
		command += " " + options
	}

	return g.executor.Execute(command)
}

// ログを取得
func (g *GitOperations) GetLog(options string) (*ExecutionResult, error) {
	command := "git log --oneline"
	if options != "" {
		command += " " + options
	}

	return g.executor.Execute(command)
}

// リモートからプル
func (g *GitOperations) Pull(remote, branch string) (*ExecutionResult, error) {
	if remote == "" {
		remote = "origin"
	}
	if branch == "" {
		branch = "main"
	}

	command := fmt.Sprintf("git pull %s %s", remote, branch)
	return g.executor.Execute(command)
}

// リモートにプッシュ
func (g *GitOperations) Push(remote, branch string) (*ExecutionResult, error) {
	if remote == "" {
		remote = "origin"
	}

	var command string
	if branch == "" {
		command = fmt.Sprintf("git push %s", remote)
	} else {
		command = fmt.Sprintf("git push %s %s", remote, branch)
	}

	return g.executor.Execute(command)
}

// ブランチ名のバリデーション
func (g *GitOperations) validateBranchName(branchName string) error {
	if strings.TrimSpace(branchName) == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// 危険な文字をチェック
	invalidChars := []string{"..", "~", "^", ":", "?", "*", "[", "\\", " "}
	for _, char := range invalidChars {
		if strings.Contains(branchName, char) {
			return fmt.Errorf("branch name contains invalid character: %s", char)
		}
	}

	return nil
}

// パスが安全かチェック
func (g *GitOperations) isPathSafe(path string) bool {
	// 相対パスの危険なパターンをチェック
	if strings.Contains(path, "..") {
		return false
	}

	// 絶対パスの場合、リポジトリ内かチェック
	if filepath.IsAbs(path) {
		absRepoPath, err := filepath.Abs(g.repoPath)
		if err != nil {
			return false
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return false
		}

		return strings.HasPrefix(absPath, absRepoPath)
	}

	return true
}