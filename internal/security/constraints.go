package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// セキュリティ制約を管理する構造体
type Constraints struct {
	AllowedCommands []string // 実行許可されたコマンド
	BlockedCommands []string // 実行禁止されたコマンド
	MaxTimeout      int      // コマンド実行の最大タイムアウト（秒）
	WorkspaceDir    string   // 作業ディレクトリ
}

// デフォルトのセキュリティ制約を作成
func NewDefaultConstraints(workspaceDir string) *Constraints {
	return &Constraints{
		AllowedCommands: []string{
			"ls", "cat", "grep", "find", "head", "tail", "wc", "sort", "uniq",
			"git", "go", "npm", "node", "python", "python3", "pip", "pip3",
			"make", "cmake", "rustc", "cargo", "javac", "java", "mvn",
			"docker", "kubectl", "helm", "terraform",
		},
		BlockedCommands: []string{
			"rm", "rmdir", "mv", "cp", "chmod", "chown", "sudo", "su",
			"curl", "wget", "ssh", "scp", "ftp", "telnet", "nc", "netcat",
			"dd", "mkfs", "fdisk", "mount", "umount", "systemctl", "service",
		},
		MaxTimeout:   30, // 30秒のタイムアウト
		WorkspaceDir: workspaceDir,
	}
}

// コマンドが実行許可されているかチェック
func (c *Constraints) IsCommandAllowed(command string) error {
	baseCommand := strings.Split(command, " ")[0]
	baseCommand = filepath.Base(baseCommand)

	// 明示的に禁止されているコマンドをチェック
	for _, blocked := range c.BlockedCommands {
		if baseCommand == blocked {
			return fmt.Errorf("command '%s' is blocked for security reasons", baseCommand)
		}
	}

	// 許可されたコマンドリストをチェック
	for _, allowed := range c.AllowedCommands {
		if baseCommand == allowed {
			return nil
		}
	}

	return fmt.Errorf("command '%s' is not in the allowed list", baseCommand)
}

// ValidateCommand はコマンドのバリデーション（IsCommandAllowedのエイリアス）
func (c *Constraints) ValidateCommand(command string) error {
	return c.IsCommandAllowed(command)
}

// パスがワークスペース内かチェック
func (c *Constraints) IsPathAllowed(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	absWorkspace, err := filepath.Abs(c.WorkspaceDir)
	if err != nil {
		return false
	}

	return strings.HasPrefix(absPath, absWorkspace)
}

// 環境変数のフィルタリング（機密情報の除外）
func (c *Constraints) FilterEnvironment() []string {
	env := os.Environ()
	filtered := make([]string, 0, len(env))

	// 機密情報を含む可能性のある環境変数を除外
	sensitiveKeys := []string{
		"PASSWORD", "SECRET", "KEY", "TOKEN", "API_KEY", "AUTH",
		"PRIVATE", "CREDENTIAL", "CERT", "SSH", "AWS", "GOOGLE",
	}

	for _, envVar := range env {
		key := strings.Split(envVar, "=")[0]
		keyUpper := strings.ToUpper(key)

		isSensitive := false
		for _, sensitive := range sensitiveKeys {
			if strings.Contains(keyUpper, sensitive) {
				isSensitive = true
				break
			}
		}

		if !isSensitive {
			filtered = append(filtered, envVar)
		}
	}

	return filtered
}
