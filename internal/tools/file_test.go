package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFileOperations はファイル操作機能をテストする
func TestFileOperations(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!\nThis is a test file."

	fileOps := NewFileOperations(10*1024*1024, tmpDir) // 10MB制限、テンポラリディレクトリ

	// ファイル書き込みテスト
	err := fileOps.WriteFile(testFile, testContent)
	if err != nil {
		t.Fatalf("ファイル書き込みエラー: %v", err)
	}

	// ファイル読み込みテスト
	content, err := fileOps.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ファイル読み込みエラー: %v", err)
	}

	if content != testContent {
		t.Errorf("期待値: %s, 実際値: %s", testContent, content)
	}
}

// TestFileSize はファイルサイズ制限をテストする
func TestFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// 大きなファイルを作成（テスト用）
	largeContent := make([]byte, 1024*1024*11) // 11MB
	for i := range largeContent {
		largeContent[i] = 'a'
	}

	err := os.WriteFile(testFile, largeContent, 0644)
	if err != nil {
		t.Fatalf("大きなファイルの作成に失敗: %v", err)
	}

	// ファイルサイズ制限のテスト（10MB制限想定）
	fileOps := NewFileOperations(10*1024*1024, tmpDir)
	_, err = fileOps.ReadFile(testFile)
	if err == nil {
		t.Error("ファイルサイズ制限が機能していません")
	}
}

// TestSecurityConstraints はセキュリティ制約をテストする
func TestSecurityConstraints(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		allowed  bool
	}{
		{
			name:    "プロジェクト内ファイル",
			path:    "./internal/config/config.go",
			allowed: true,
		},
		{
			name:    "システムファイル",
			path:    "/etc/passwd",
			allowed: false,
		},
		{
			name:    "親ディレクトリ",
			path:    "../../../etc/passwd",
			allowed: false,
		},
	}

	// 現在のディレクトリをワークディレクトリとして使用
	workDir, _ := os.Getwd()
	fileOps := NewFileOperations(10*1024*1024, workDir)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := fileOps.isPathAllowed(tt.path)
			if allowed != tt.allowed {
				t.Errorf("期待値: %v, 実際値: %v", tt.allowed, allowed)
			}
		})
	}
}