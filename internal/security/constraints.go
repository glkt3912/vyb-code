package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// セキュリティ制約を管理する構造体
type Constraints struct {
	AllowedCommands   []string // 実行許可されたコマンド
	BlockedCommands   []string // 実行禁止されたコマンド
	MaxTimeout        int      // コマンド実行の最大タイムアウト（秒）
	WorkspaceDir      string   // 作業ディレクトリ
	AllowedExtensions []string // 読み書き許可されたファイル拡張子
	BlockedPaths      []string // アクセス禁止パス
	MaxFileSize       int64    // 最大ファイルサイズ（バイト）
	ReadOnlyMode      bool     // 読み取り専用モード
}

// デフォルトのセキュリティ制約を作成
func NewDefaultConstraints(workspaceDir string) *Constraints {
	return &Constraints{
		AllowedCommands: []string{
			"ls", "cat", "grep", "find", "head", "tail", "wc", "sort", "uniq", "echo",
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
		AllowedExtensions: []string{
			".go", ".js", ".ts", ".py", ".java", ".c", ".cpp", ".h", ".hpp",
			".rs", ".rb", ".php", ".cs", ".kt", ".swift", ".dart", ".scala",
			".html", ".css", ".scss", ".sass", ".less", ".vue", ".jsx", ".tsx",
			".json", ".xml", ".yaml", ".yml", ".toml", ".ini", ".conf",
			".md", ".txt", ".log", ".csv", ".sql", ".sh", ".bash", ".zsh",
			".dockerfile", ".gitignore", ".gitattributes", ".editorconfig",
			".env.example", ".env.template", ".sample",
		},
		BlockedPaths: []string{
			"/etc", "/usr", "/bin", "/sbin", "/root", "/var/log",
			"/proc", "/sys", "/dev", "/tmp", "/var/tmp",
			"~/.ssh", "~/.aws", "~/.gcp", "~/.azure",
			".env", ".env.local", ".env.production",
		},
		MaxFileSize:  10 * 1024 * 1024, // 10MB
		ReadOnlyMode: false,
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

// ファイルアクセスが許可されているかチェック
func (c *Constraints) IsFileAccessAllowed(filePath string, operation string) error {
	// パスの正規化
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("パス正規化失敗: %w", err)
	}

	// ワークスペース外へのアクセスチェック
	if !c.IsPathAllowed(absPath) {
		return fmt.Errorf("ワークスペース外のファイルアクセスは禁止されています: %s", filePath)
	}

	// 禁止パスチェック
	for _, blockedPath := range c.BlockedPaths {
		if strings.Contains(absPath, blockedPath) || strings.HasPrefix(filepath.Base(filePath), blockedPath) {
			return fmt.Errorf("アクセス禁止パスです: %s", filePath)
		}
	}

	// 読み取り専用モードでの書き込みチェック
	if c.ReadOnlyMode && (operation == "write" || operation == "create" || operation == "delete") {
		return fmt.Errorf("読み取り専用モードのため書き込み操作は禁止されています")
	}

	// ファイル拡張子チェック
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != "" {
		allowed := false
		for _, allowedExt := range c.AllowedExtensions {
			if ext == allowedExt {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("許可されていないファイル拡張子です: %s", ext)
		}
	}

	// ファイルサイズチェック（書き込み前は困難なので、既存ファイルのみ）
	if operation == "read" {
		if info, err := os.Stat(absPath); err == nil {
			if info.Size() > c.MaxFileSize {
				return fmt.Errorf("ファイルサイズが制限を超えています: %d bytes (最大: %d bytes)", info.Size(), c.MaxFileSize)
			}
		}
	}

	return nil
}

// ディレクトリ作成が許可されているかチェック
func (c *Constraints) IsDirectoryCreationAllowed(dirPath string) error {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("パス正規化失敗: %w", err)
	}

	if !c.IsPathAllowed(absPath) {
		return fmt.Errorf("ワークスペース外のディレクトリ作成は禁止されています: %s", dirPath)
	}

	if c.ReadOnlyMode {
		return fmt.Errorf("読み取り専用モードのためディレクトリ作成は禁止されています")
	}

	// 禁止パスチェック
	for _, blockedPath := range c.BlockedPaths {
		if strings.Contains(absPath, blockedPath) {
			return fmt.Errorf("アクセス禁止パスです: %s", dirPath)
		}
	}

	return nil
}

// セキュリティ制約を読み取り専用モードに設定
func (c *Constraints) SetReadOnlyMode(readOnly bool) {
	c.ReadOnlyMode = readOnly
}

// 最大ファイルサイズを設定
func (c *Constraints) SetMaxFileSize(maxSize int64) {
	c.MaxFileSize = maxSize
}

// 許可された拡張子を追加
func (c *Constraints) AddAllowedExtension(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ext = strings.ToLower(ext)
	for _, allowed := range c.AllowedExtensions {
		if allowed == ext {
			return // 既に存在する
		}
	}
	c.AllowedExtensions = append(c.AllowedExtensions, ext)
}

// 禁止パスを追加
func (c *Constraints) AddBlockedPath(path string) {
	for _, blocked := range c.BlockedPaths {
		if blocked == path {
			return // 既に存在する
		}
	}
	c.BlockedPaths = append(c.BlockedPaths, path)
}

// ファイルの内容が安全かチェック
func (c *Constraints) ValidateFileContent(content []byte, filePath string) error {
	// ファイルサイズチェック
	if int64(len(content)) > c.MaxFileSize {
		return fmt.Errorf("ファイル内容が最大サイズを超えています: %d bytes (最大: %d bytes)", len(content), c.MaxFileSize)
	}

	// バイナリファイルの検出（実行可能ファイルなど）
	if c.isBinaryContent(content) {
		ext := strings.ToLower(filepath.Ext(filePath))
		// 実行可能ファイル拡張子をチェック
		executableExts := []string{".exe", ".bat", ".cmd", ".com", ".scr", ".bin", ".app", ".dmg", ".pkg"}
		for _, execExt := range executableExts {
			if ext == execExt {
				return fmt.Errorf("実行可能ファイルの書き込みは禁止されています: %s", filePath)
			}
		}
	}

	return nil
}

// バイナリコンテンツかどうか判定
func (c *Constraints) isBinaryContent(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// ELF、PE、Mach-Oなどの実行可能ファイルヘッダーをチェック
	binarySignatures := [][]byte{
		{0x7F, 0x45, 0x4C, 0x46}, // ELF
		{0x4D, 0x5A},             // PE (MZ)
		{0xFE, 0xED, 0xFA, 0xCE}, // Mach-O (32-bit)
		{0xFE, 0xED, 0xFA, 0xCF}, // Mach-O (64-bit)
		{0xCA, 0xFE, 0xBA, 0xBE}, // Mach-O Universal
	}

	for _, sig := range binarySignatures {
		if len(content) >= len(sig) && string(content[:len(sig)]) == string(sig) {
			return true
		}
	}

	// NULL文字の存在をチェック（簡易的なバイナリ検出）
	for i, b := range content {
		if b == 0 && i < 1024 { // 最初の1KB内にNULL文字
			return true
		}
	}

	return false
}
