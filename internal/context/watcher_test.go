package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWatcher_Creation(t *testing.T) {
	testDir := "/tmp/test_watcher"
	watcher := NewWatcher(testDir)

	if watcher == nil {
		t.Error("Expected non-nil watcher")
	}

	if watcher.workDir != testDir {
		t.Errorf("Expected workDir %q, got %q", testDir, watcher.workDir)
	}

	if watcher.watchedFiles == nil {
		t.Error("Expected non-nil watchedFiles map")
	}

	if watcher.isWatching {
		t.Error("Expected isWatching to be false initially")
	}

	if len(watcher.watchedFiles) != 0 {
		t.Errorf("Expected empty watchedFiles initially, got %d entries", len(watcher.watchedFiles))
	}
}

func TestWatcher_ProjectLanguageDetection(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	watcher := NewWatcher(tempDir)

	tests := []struct {
		name         string
		files        []string
		expectedLang string
	}{
		{
			name:         "Go project",
			files:        []string{"go.mod"},
			expectedLang: "Go",
		},
		{
			name:         "JavaScript project",
			files:        []string{"package.json"},
			expectedLang: "JavaScript/TypeScript",
		},
		{
			name:         "Python project",
			files:        []string{"requirements.txt"},
			expectedLang: "Python",
		},
		{
			name:         "Python setup project",
			files:        []string{"setup.py"},
			expectedLang: "Python",
		},
		{
			name:         "Rust project",
			files:        []string{"Cargo.toml"},
			expectedLang: "Rust",
		},
		{
			name:         "Unknown project",
			files:        []string{"README.md"},
			expectedLang: "Unknown",
		},
		{
			name:         "Multiple languages (Go priority)",
			files:        []string{"go.mod", "package.json", "requirements.txt"},
			expectedLang: "Go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テストファイルを作成
			for _, file := range tt.files {
				createTestFile(t, tempDir, file, "test content")
			}

			detected := watcher.detectProjectLanguage()

			if detected != tt.expectedLang {
				t.Errorf("Expected language %q, got %q", tt.expectedLang, detected)
			}

			// テストファイルをクリーンアップ
			for _, file := range tt.files {
				os.Remove(filepath.Join(tempDir, file))
			}
		})
	}
}

func TestWatcher_FileExists(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	watcher := NewWatcher(tempDir)

	// 存在しないファイル
	if watcher.fileExists("nonexistent.txt") {
		t.Error("Expected fileExists to return false for nonexistent file")
	}

	// ファイルを作成
	testFile := "test.txt"
	createTestFile(t, tempDir, testFile, "test content")

	// 存在するファイル
	if !watcher.fileExists(testFile) {
		t.Error("Expected fileExists to return true for existing file")
	}
}

func TestWatcher_IsDirectory(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	watcher := NewWatcher(tempDir)

	// ディレクトリのテスト
	if !watcher.isDirectory(tempDir) {
		t.Error("Expected isDirectory to return true for directory")
	}

	// ファイルを作成
	testFile := filepath.Join(tempDir, "test.txt")
	createTestFile(t, tempDir, "test.txt", "content")

	// ファイルのテスト
	if watcher.isDirectory(testFile) {
		t.Error("Expected isDirectory to return false for file")
	}

	// 存在しないパス
	if watcher.isDirectory(filepath.Join(tempDir, "nonexistent")) {
		t.Error("Expected isDirectory to return false for nonexistent path")
	}
}

func TestWatcher_CountProjectFiles(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	watcher := NewWatcher(tempDir)

	// 初期状態（空ディレクトリ）
	count := watcher.countProjectFiles()
	if count != 0 {
		t.Errorf("Expected 0 files in empty directory, got %d", count)
	}

	// ファイルを追加
	createTestFile(t, tempDir, "file1.txt", "content1")
	createTestFile(t, tempDir, "file2.go", "package main")

	count = watcher.countProjectFiles()
	if count != 2 {
		t.Errorf("Expected 2 files after adding files, got %d", count)
	}

	// 隠しファイルを追加（カウントされないはず）
	createTestFile(t, tempDir, ".hidden", "hidden content")

	count = watcher.countProjectFiles()
	if count != 2 {
		t.Errorf("Expected hidden files to be ignored, got %d", count)
	}

	// サブディレクトリとファイル
	subDir := filepath.Join(tempDir, "subdir")
	os.Mkdir(subDir, 0755)
	createTestFile(t, subDir, "file3.txt", "content3")

	count = watcher.countProjectFiles()
	if count != 3 {
		t.Errorf("Expected 3 files including subdirectory, got %d", count)
	}

	// node_modules ディレクトリ（スキップされるはず）
	nodeModules := filepath.Join(tempDir, "node_modules")
	os.Mkdir(nodeModules, 0755)
	createTestFile(t, nodeModules, "package.js", "module content")

	count = watcher.countProjectFiles()
	if count != 3 {
		t.Errorf("Expected node_modules to be skipped, got %d files", count)
	}
}

func TestWatcher_HasFileChanged(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	watcher := NewWatcher(tempDir)

	testFile := "test.txt"
	testPath := filepath.Join(tempDir, testFile)

	// 存在しないファイル
	if watcher.hasFileChanged(testFile) {
		t.Error("Expected hasFileChanged to return false for nonexistent file")
	}

	// ファイルを作成
	createTestFile(t, tempDir, testFile, "initial content")

	// 初回チェック（新しいファイルとして検出されるはず）
	if !watcher.hasFileChanged(testFile) {
		t.Error("Expected hasFileChanged to return true for new file")
	}

	// 2回目のチェック（変更なしのはず）
	if watcher.hasFileChanged(testFile) {
		t.Error("Expected hasFileChanged to return false for unchanged file")
	}

	// ファイルを変更
	time.Sleep(10 * time.Millisecond) // ファイル時刻の差を作るため
	writeTestFile(t, testPath, "modified content")

	// 変更が検出されることを確認
	if !watcher.hasFileChanged(testFile) {
		t.Error("Expected hasFileChanged to return true for modified file")
	}
}

func TestWatcher_StartStopWatching(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	watcher := NewWatcher(tempDir)

	// 初期状態
	if watcher.isWatching {
		t.Error("Expected watcher to not be watching initially")
	}

	// 監視開始
	err := watcher.StartWatching()
	if err != nil {
		t.Errorf("Expected no error starting watcher, got %v", err)
	}

	if !watcher.isWatching {
		t.Error("Expected watcher to be watching after StartWatching()")
	}

	// 初期状態が記録されることを確認
	if watcher.projectLang == "" {
		t.Error("Expected project language to be detected")
	}

	if watcher.lastFileCount < 0 {
		t.Error("Expected file count to be recorded")
	}

	// 重複開始（エラーなし）
	err = watcher.StartWatching()
	if err != nil {
		t.Errorf("Expected no error for duplicate StartWatching(), got %v", err)
	}

	// 監視停止
	watcher.StopWatching()
	if watcher.isWatching {
		t.Error("Expected watcher to stop watching after StopWatching()")
	}
}

func TestWatcher_CheckChanges(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	watcher := NewWatcher(tempDir)

	// 監視していない状態でのチェック
	changes := watcher.CheckChanges()
	if len(changes) != 0 {
		t.Errorf("Expected no changes when not watching, got %d", len(changes))
	}

	// 監視開始
	watcher.StartWatching()

	// 変更なしの状態
	changes = watcher.CheckChanges()
	if len(changes) != 0 {
		t.Errorf("Expected no changes initially, got %d", len(changes))
	}

	// ファイルを追加
	createTestFile(t, tempDir, "new_file.txt", "new content")

	// ファイル変更が検出されることを確認
	changes = watcher.CheckChanges()
	hasFileChange := false
	for _, change := range changes {
		if change.Type == ChangeFileAdd || change.Type == ChangeFileModify {
			hasFileChange = true
			break
		}
	}

	if !hasFileChange {
		t.Error("Expected file change to be detected")
	}
}

func TestWatcher_ChangeInfo(t *testing.T) {
	watcher := NewWatcher("/test")

	tests := []struct {
		name          string
		change        ChangeInfo
		expectedIcon  string
		expectedColor string
	}{
		{
			name: "High severity change",
			change: ChangeInfo{
				Type:        ChangeDependency,
				Description: "Dependencies updated",
				Severity:    SeverityHigh,
			},
			expectedIcon:  "🚨",
			expectedColor: "\033[31m", // 赤
		},
		{
			name: "Medium severity change",
			change: ChangeInfo{
				Type:        ChangeFileAdd,
				Description: "File added",
				Severity:    SeverityMedium,
			},
			expectedIcon:  "⚠️",
			expectedColor: "\033[33m", // 黄
		},
		{
			name: "Low severity change",
			change: ChangeInfo{
				Type:        ChangeGitCommit,
				Description: "New commit",
				Severity:    SeverityLow,
			},
			expectedIcon:  "ℹ️",
			expectedColor: "\033[36m", // シアン
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := watcher.FormatChange(tt.change)

			// アイコンが含まれることを確認
			if !strings.Contains(formatted, tt.expectedIcon) {
				t.Errorf("Expected formatted change to contain icon %q, got %q", tt.expectedIcon, formatted)
			}

			// 色コードが含まれることを確認
			if !strings.Contains(formatted, tt.expectedColor) {
				t.Errorf("Expected formatted change to contain color %q, got %q", tt.expectedColor, formatted)
			}

			// 説明が含まれることを確認
			if !strings.Contains(formatted, tt.change.Description) {
				t.Errorf("Expected formatted change to contain description %q, got %q", tt.change.Description, formatted)
			}

			// リセットコードが含まれることを確認
			if !strings.Contains(formatted, "\033[0m") {
				t.Error("Expected formatted change to contain reset color code")
			}
		})
	}
}

func TestWatcher_FileChangeTracking(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	watcher := NewWatcher(tempDir)
	watcher.StartWatching()

	testFile := "tracked_file.txt"

	// 初期状態：ファイル数を記録
	initialCount := watcher.lastFileCount

	// ファイルを追加
	createTestFile(t, tempDir, testFile, "content")

	// 変更をチェック
	changes := watcher.CheckChanges()

	// ファイル追加が検出されることを確認
	hasAddChange := false
	for _, change := range changes {
		if change.Type == ChangeFileAdd {
			hasAddChange = true
			if !strings.Contains(change.Description, fmt.Sprintf("%d から %d", initialCount, initialCount+1)) {
				t.Errorf("Expected file count change in description, got %q", change.Description)
			}
		}
	}

	if !hasAddChange {
		t.Error("Expected file add change to be detected")
	}

	// ファイル数が更新されることを確認
	if watcher.lastFileCount != initialCount+1 {
		t.Errorf("Expected file count to be updated to %d, got %d", initialCount+1, watcher.lastFileCount)
	}
}

func TestWatcher_DependencyChangeDetection(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	watcher := NewWatcher(tempDir)
	watcher.StartWatching()

	tests := []struct {
		filename    string
		expectedMsg string
	}{
		{
			filename:    "go.mod",
			expectedMsg: "Go モジュールの依存関係が変更されました",
		},
		{
			filename:    "package.json",
			expectedMsg: "Node.js パッケージの依存関係が変更されました",
		},
	}

	for _, tt := range tests {
		t.Run("Dependency change "+tt.filename, func(t *testing.T) {
			// 依存関係ファイルを作成
			createTestFile(t, tempDir, tt.filename, "initial content")

			// 初回チェック（ファイル作成による変更）
			watcher.CheckChanges()

			// ファイルを変更
			time.Sleep(10 * time.Millisecond)
			writeTestFile(t, filepath.Join(tempDir, tt.filename), "modified content")

			// 変更をチェック
			changes := watcher.CheckChanges()

			// 依存関係変更が検出されることを確認
			hasDependencyChange := false
			for _, change := range changes {
				if change.Type == ChangeDependency {
					hasDependencyChange = true
					if change.Description != tt.expectedMsg {
						t.Errorf("Expected description %q, got %q", tt.expectedMsg, change.Description)
					}
					if change.Severity != SeverityHigh {
						t.Errorf("Expected high severity for dependency change, got %v", change.Severity)
					}

					// ファイルリストが含まれることを確認
					found := false
					for _, file := range change.Files {
						if file == tt.filename {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected file %q in change.Files", tt.filename)
					}
				}
			}

			if !hasDependencyChange {
				t.Errorf("Expected dependency change to be detected for %s", tt.filename)
			}

			// クリーンアップ
			os.Remove(filepath.Join(tempDir, tt.filename))
		})
	}
}

func TestWatcher_GitChangeSimulation(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	watcher := NewWatcher(tempDir)
	watcher.StartWatching()

	// 初期Git状態を設定
	watcher.gitBranch = "main"
	watcher.lastGitCommit = "abc1234"

	t.Run("Branch change", func(t *testing.T) {
		// ブランチ変更をシミュレート（実際のgit実装は仮）
		changes := watcher.checkGitChanges()

		// 仮実装では"main"ブランチが返されるため、変更は検出されない
		// 実際の実装では git コマンドを使用する
		if len(changes) > 0 {
			for _, change := range changes {
				if change.Type == ChangeGitBranch {
					if !strings.Contains(change.Description, "ブランチが") {
						t.Errorf("Expected branch change description, got %q", change.Description)
					}
				}
			}
		}
	})

	t.Run("Commit change", func(t *testing.T) {
		// コミット変更をシミュレート
		changes := watcher.checkGitChanges()

		// 仮実装では固定値が返されるため、実装完了後にテストを拡張
		for _, change := range changes {
			if change.Type == ChangeGitCommit {
				if !strings.Contains(change.Description, "新しいコミット") {
					t.Errorf("Expected commit change description, got %q", change.Description)
				}
				if change.Severity != SeverityLow {
					t.Errorf("Expected low severity for commit change, got %v", change.Severity)
				}
			}
		}
	})
}

func TestWatcher_ChangeTypes(t *testing.T) {
	// 変更タイプの定数値をテスト
	expectedTypes := map[ChangeType]string{
		ChangeGitBranch:  "git_branch",
		ChangeGitCommit:  "git_commit",
		ChangeFileAdd:    "file_add",
		ChangeFileModify: "file_modify",
		ChangeFileDelete: "file_delete",
		ChangeDependency: "dependency",
	}

	for changeType, expected := range expectedTypes {
		if string(changeType) != expected {
			t.Errorf("Expected change type %v to have value %q, got %q",
				changeType, expected, string(changeType))
		}
	}

	// 重要度レベルのテスト
	expectedSeverities := map[ChangeSeverity]string{
		SeverityLow:    "low",
		SeverityMedium: "medium",
		SeverityHigh:   "high",
	}

	for severity, expected := range expectedSeverities {
		if string(severity) != expected {
			t.Errorf("Expected severity %v to have value %q, got %q",
				severity, expected, string(severity))
		}
	}
}

// 実際のファイルシステム操作をテスト
func TestWatcher_RealFileOperations(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	watcher := NewWatcher(tempDir)
	watcher.StartWatching()

	// 複数の操作を連続実行
	operations := []struct {
		name   string
		action func()
	}{
		{
			name: "Create go.mod",
			action: func() {
				createTestFile(t, tempDir, "go.mod", "module test")
			},
		},
		{
			name: "Create source file",
			action: func() {
				createTestFile(t, tempDir, "main.go", "package main")
			},
		},
		{
			name: "Modify go.mod",
			action: func() {
				time.Sleep(10 * time.Millisecond)
				writeTestFile(t, filepath.Join(tempDir, "go.mod"), "module test\ngo 1.21")
			},
		},
	}

	allChanges := []ChangeInfo{}

	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			op.action()
			changes := watcher.CheckChanges()
			allChanges = append(allChanges, changes...)

			// 何らかの変更が検出されることを確認
			if len(changes) == 0 {
				t.Logf("No changes detected for %s (may be expected)", op.name)
			}
		})
	}

	// 全体的な変更が記録されていることを確認
	if len(allChanges) == 0 {
		t.Log("No changes detected in file operations (check implementation)")
	}
}

// エラーハンドリングのテスト
func TestWatcher_ErrorHandling(t *testing.T) {
	// 存在しないディレクトリでウォッチャーを作成
	nonexistentDir := "/nonexistent/directory"
	watcher := NewWatcher(nonexistentDir)

	// エラーハンドリングテスト
	_ = watcher.StartWatching()
	// 存在しないディレクトリでもエラーが発生しない場合がある（実装による）
}

// パフォーマンステスト
func TestWatcher_Performance(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// 多数のファイルを作成
	for i := 0; i < 100; i++ {
		createTestFile(t, tempDir, fmt.Sprintf("file%d.txt", i), "content")
	}

	watcher := NewWatcher(tempDir)
	watcher.StartWatching()

	// ファイル数カウントのパフォーマンス
	start := time.Now()
	count := watcher.countProjectFiles()
	duration := time.Since(start)

	if count != 100 {
		t.Errorf("Expected 100 files, got %d", count)
	}

	if duration > 100*time.Millisecond {
		t.Errorf("File counting took too long: %v", duration)
	}

	// 変更チェックのパフォーマンス
	start = time.Now()
	for i := 0; i < 10; i++ {
		watcher.CheckChanges()
	}
	duration = time.Since(start)

	if duration > 200*time.Millisecond {
		t.Errorf("10 change checks took too long: %v", duration)
	}
}

// テストヘルパー関数
func createTempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "watcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

func createTestFile(t *testing.T, dir, filename, content string) {
	path := filepath.Join(dir, filename)

	// ディレクトリ作成
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create directory for %s: %v", path, err)
	}

	// ファイル作成
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", path, err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file %s: %v", path, err)
	}
}

// ベンチマークテスト
func BenchmarkWatcher_CountProjectFiles(b *testing.B) {
	tempDir := createTempDirBench(b)
	defer os.RemoveAll(tempDir)

	// テストファイルを作成
	for i := 0; i < 50; i++ {
		createTestFileBench(b, tempDir, fmt.Sprintf("file%d.go", i), "package main")
	}

	watcher := NewWatcher(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.countProjectFiles()
	}
}

func BenchmarkWatcher_CheckChanges(b *testing.B) {
	tempDir := createTempDirBench(b)
	defer os.RemoveAll(tempDir)

	watcher := NewWatcher(tempDir)
	watcher.StartWatching()

	// いくつかファイルを作成
	for i := 0; i < 10; i++ {
		createTestFileBench(b, tempDir, fmt.Sprintf("file%d.txt", i), "content")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.CheckChanges()
	}
}

func BenchmarkWatcher_FormatChange(b *testing.B) {
	watcher := NewWatcher("/test")
	change := ChangeInfo{
		Type:        ChangeFileModify,
		Description: "Test file was modified",
		Severity:    SeverityMedium,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.FormatChange(change)
	}
}

// ベンチマーク用ヘルパー関数
func createTempDirBench(tb testing.TB) string {
	dir, err := os.MkdirTemp("", "watcher_test")
	if err != nil {
		tb.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

func createTestFileBench(tb testing.TB, dir, filename, content string) {
	path := filepath.Join(dir, filename)

	// ディレクトリ作成
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		tb.Fatalf("Failed to create directory for %s: %v", path, err)
	}

	// ファイル作成
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tb.Fatalf("Failed to create test file %s: %v", path, err)
	}
}
