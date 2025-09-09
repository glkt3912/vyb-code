package tools

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FileOperations - 廃止予定: UnifiedFileOperationsを使用してください
// Deprecated: Use UnifiedFileOperations in edit_tools.go instead
type FileOperations struct {
	MaxFileSize int64  // 読み込み可能な最大ファイルサイズ
	WorkDir     string // 作業ディレクトリ（セキュリティ制約用）
}

// NewFileOperations - 廃止予定: NewUnifiedFileOperationsを使用してください
// Deprecated: Use NewUnifiedFileOperations in edit_tools.go instead
func NewFileOperations(maxFileSize int64, workDir string) *FileOperations {
	return &FileOperations{
		MaxFileSize: maxFileSize,
		WorkDir:     workDir,
	}
}

// ファイルを読み込んで内容を返す
func (f *FileOperations) ReadFile(filePath string) (string, error) {
	// セキュリティチェック：作業ディレクトリ内かどうか確認
	if !f.isPathAllowed(filePath) {
		return "", fmt.Errorf("access denied: path outside workspace")
	}

	// ファイル情報を取得してサイズをチェック
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	// ファイルサイズ制限チェック
	if info.Size() > f.MaxFileSize {
		return "", fmt.Errorf("file too large: %d bytes (max: %d)", info.Size(), f.MaxFileSize)
	}

	// ファイル内容を読み込み
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// ファイルに内容を書き込む
func (f *FileOperations) WriteFile(filePath, content string) error {
	// セキュリティチェック：作業ディレクトリ内かどうか確認
	if !f.isPathAllowed(filePath) {
		return fmt.Errorf("access denied: path outside workspace")
	}

	// ディレクトリが存在しない場合は作成
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// ファイルに書き込み（権限644）
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// 指定されたパターンを含むファイルを検索する
func (f *FileOperations) SearchFiles(pattern string) ([]string, error) {
	var matches []string

	// 作業ディレクトリ以下を再帰的に走査
	err := filepath.WalkDir(f.WorkDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // エラーのあるファイルはスキップ
		}

		// ディレクトリはスキップ
		if d.IsDir() {
			return nil
		}

		// バイナリファイルや大きなファイルはスキップ
		if !f.isTextFile(path) {
			return nil
		}

		// ファイル内容を読み込んで検索
		content, err := f.ReadFile(path)
		if err != nil {
			return nil // 読み込めないファイルはスキップ
		}

		// パターンマッチング（大文字小文字区別なし）
		if strings.Contains(strings.ToLower(content), strings.ToLower(pattern)) {
			matches = append(matches, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search files: %w", err)
	}

	return matches, nil
}

// パスが許可されたワークスペース内かどうかをチェック
func (f *FileOperations) isPathAllowed(path string) bool {
	// 絶対パスに変換
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// 作業ディレクトリの絶対パス
	absWorkDir, err := filepath.Abs(f.WorkDir)
	if err != nil {
		return false
	}

	// パスが作業ディレクトリ以下にあるかチェック
	return strings.HasPrefix(absPath, absWorkDir)
}

// ファイルがテキストファイルかどうかを判定
func (f *FileOperations) isTextFile(path string) bool {
	// 拡張子ベースの簡易判定
	ext := strings.ToLower(filepath.Ext(path))
	textExts := []string{
		".go", ".js", ".ts", ".py", ".java", ".c", ".cpp", ".h", ".hpp",
		".md", ".txt", ".json", ".yaml", ".yml", ".xml", ".html", ".css",
		".sh", ".bash", ".sql", ".rs", ".php", ".rb", ".kt", ".swift",
	}

	for _, textExt := range textExts {
		if ext == textExt {
			return true
		}
	}

	return false
}
