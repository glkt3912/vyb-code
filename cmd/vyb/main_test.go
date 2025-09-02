package main

import (
	"os"
	"testing"
)

// TestSetModel はモデル設定機能をテストする
func TestSetModel(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// モデル設定をテスト
	testModel := "test-model:latest"

	// 関数を直接呼び出してテスト（出力は無視）
	setModel(testModel)

	// 実際の設定ファイルから確認は統合テストで行う
}

// TestSetProvider はプロバイダー設定機能をテストする
func TestSetProvider(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// プロバイダー設定をテスト
	testProvider := "test-provider"

	// 関数を直接呼び出してテスト（出力は無視）
	setProvider(testProvider)
}

// TestListConfig は設定表示機能をテストする
func TestListConfig(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// 設定表示機能をテスト（出力は無視）
	listConfig()
}

// TestMinFunction はmin関数をテストする
func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{10, 10, 10},
		{0, -1, -1},
	}

	for _, test := range tests {
		result := min(test.a, test.b)
		if result != test.expected {
			t.Errorf("min(%d, %d) = %d; 期待値 %d", test.a, test.b, result, test.expected)
		}
	}
}

// TestGitOperationsFunctions はGit操作関数の基本テストを行う
func TestGitOperationsFunctions(t *testing.T) {
	// テスト用ディレクトリの設定
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	// Gitリポジトリを初期化
	if err := os.Mkdir(".git", 0755); err != nil {
		t.Skip("Gitテスト環境の準備に失敗")
	}

	// Git操作関数のテスト（エラーハンドリングのみ）
	// 実際のGit操作は統合テストで検証
	t.Run("gitStatus", func(t *testing.T) {
		gitStatus() // パニックしないことを確認
	})

	t.Run("listBranches", func(t *testing.T) {
		listBranches() // パニックしないことを確認
	})
}

// TestMCPFunctions はMCP操作関数の基本テストを行う
func TestMCPFunctions(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	t.Run("listMCPServers", func(t *testing.T) {
		listMCPServers() // パニックしないことを確認
	})

	t.Run("listMCPTools", func(t *testing.T) {
		listMCPTools("test-server") // パニックしないことを確認
	})

	t.Run("listAllMCPTools", func(t *testing.T) {
		listAllMCPTools() // パニックしないことを確認
	})
}

// TestAnalyzeProject はプロジェクト分析機能をテストする
func TestAnalyzeProject(t *testing.T) {
	// テスト用ディレクトリの設定
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	// テスト用ファイルを作成
	os.WriteFile("test.go", []byte("package main"), 0644)

	// プロジェクト分析をテスト（パニックしないことを確認）
	analyzeProject()
}
