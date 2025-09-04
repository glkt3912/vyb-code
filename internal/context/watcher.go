package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// コンテキスト監視器
type Watcher struct {
	workDir        string
	lastGitCommit  string
	lastFileCount  int
	lastModTime    time.Time
	watchedFiles   map[string]time.Time
	gitBranch      string
	projectLang    string
	isWatching     bool
}

// プロジェクト状態の変更情報
type ChangeInfo struct {
	Type        ChangeType
	Description string
	Files       []string
	Severity    ChangeSeverity
}

type ChangeType string
type ChangeSeverity string

const (
	ChangeGitBranch  ChangeType = "git_branch"
	ChangeGitCommit  ChangeType = "git_commit"
	ChangeFileAdd    ChangeType = "file_add"
	ChangeFileModify ChangeType = "file_modify"
	ChangeFileDelete ChangeType = "file_delete"
	ChangeDependency ChangeType = "dependency"
	
	SeverityLow    ChangeSeverity = "low"
	SeverityMedium ChangeSeverity = "medium"
	SeverityHigh   ChangeSeverity = "high"
)

// 新しいウォッチャーを作成
func NewWatcher(workDir string) *Watcher {
	return &Watcher{
		workDir:      workDir,
		watchedFiles: make(map[string]time.Time),
		isWatching:   false,
	}
}

// 監視を開始
func (w *Watcher) StartWatching() error {
	if w.isWatching {
		return nil
	}

	// 初期状態を記録
	if err := w.captureInitialState(); err != nil {
		return fmt.Errorf("failed to capture initial state: %w", err)
	}

	w.isWatching = true
	return nil
}

// 初期状態をキャプチャ
func (w *Watcher) captureInitialState() error {
	// Git情報
	w.gitBranch = w.getCurrentGitBranch()
	w.lastGitCommit = w.getLastGitCommit()
	
	// プロジェクト言語
	w.projectLang = w.detectProjectLanguage()
	
	// ファイル状態
	w.lastFileCount = w.countProjectFiles()
	w.lastModTime = time.Now()
	
	return nil
}

// 変更をチェック
func (w *Watcher) CheckChanges() []ChangeInfo {
	if !w.isWatching {
		return []ChangeInfo{}
	}

	var changes []ChangeInfo
	
	// Git変更をチェック
	if gitChanges := w.checkGitChanges(); len(gitChanges) > 0 {
		changes = append(changes, gitChanges...)
	}
	
	// ファイル変更をチェック
	if fileChanges := w.checkFileChanges(); len(fileChanges) > 0 {
		changes = append(changes, fileChanges...)
	}
	
	// 依存関係変更をチェック
	if depChanges := w.checkDependencyChanges(); len(depChanges) > 0 {
		changes = append(changes, depChanges...)
	}

	return changes
}

// Git変更をチェック
func (w *Watcher) checkGitChanges() []ChangeInfo {
	var changes []ChangeInfo
	
	// ブランチ変更チェック
	currentBranch := w.getCurrentGitBranch()
	if currentBranch != w.gitBranch && currentBranch != "" {
		changes = append(changes, ChangeInfo{
			Type:        ChangeGitBranch,
			Description: fmt.Sprintf("ブランチが %s から %s に変更されました", w.gitBranch, currentBranch),
			Severity:    SeverityMedium,
		})
		w.gitBranch = currentBranch
	}
	
	// コミット変更チェック
	currentCommit := w.getLastGitCommit()
	if currentCommit != w.lastGitCommit && currentCommit != "" {
		changes = append(changes, ChangeInfo{
			Type:        ChangeGitCommit,
			Description: fmt.Sprintf("新しいコミットが追加されました: %s", currentCommit[:8]),
			Severity:    SeverityLow,
		})
		w.lastGitCommit = currentCommit
	}

	return changes
}

// ファイル変更をチェック
func (w *Watcher) checkFileChanges() []ChangeInfo {
	var changes []ChangeInfo
	
	// プロジェクトファイル数をチェック
	currentFileCount := w.countProjectFiles()
	if currentFileCount != w.lastFileCount {
		changeType := ChangeFileAdd
		if currentFileCount < w.lastFileCount {
			changeType = ChangeFileDelete
		}
		
		changes = append(changes, ChangeInfo{
			Type:        changeType,
			Description: fmt.Sprintf("ファイル数が %d から %d に変更されました", w.lastFileCount, currentFileCount),
			Severity:    SeverityMedium,
		})
		w.lastFileCount = currentFileCount
	}

	return changes
}

// 依存関係変更をチェック
func (w *Watcher) checkDependencyChanges() []ChangeInfo {
	var changes []ChangeInfo
	
	// go.mod の変更チェック
	if w.hasFileChanged("go.mod") {
		changes = append(changes, ChangeInfo{
			Type:        ChangeDependency,
			Description: "Go モジュールの依存関係が変更されました",
			Files:       []string{"go.mod"},
			Severity:    SeverityHigh,
		})
	}
	
	// package.json の変更チェック
	if w.hasFileChanged("package.json") {
		changes = append(changes, ChangeInfo{
			Type:        ChangeDependency,
			Description: "Node.js パッケージの依存関係が変更されました",
			Files:       []string{"package.json"},
			Severity:    SeverityHigh,
		})
	}

	return changes
}

// 現在のGitブランチを取得
func (w *Watcher) getCurrentGitBranch() string {
	// 簡易実装（実際のgitコマンド実行は別途実装）
	if gitDir := filepath.Join(w.workDir, ".git"); w.isDirectory(gitDir) {
		return "main" // 仮実装
	}
	return ""
}

// 最後のGitコミットを取得
func (w *Watcher) getLastGitCommit() string {
	// 簡易実装（実際のgitコマンド実行は別途実装）
	return "abc1234" // 仮実装
}

// プロジェクト言語を検出
func (w *Watcher) detectProjectLanguage() string {
	if w.fileExists("go.mod") {
		return "Go"
	}
	if w.fileExists("package.json") {
		return "JavaScript/TypeScript"
	}
	if w.fileExists("requirements.txt") || w.fileExists("setup.py") {
		return "Python"
	}
	if w.fileExists("Cargo.toml") {
		return "Rust"
	}
	return "Unknown"
}

// プロジェクトファイル数を数える
func (w *Watcher) countProjectFiles() int {
	count := 0
	
	filepath.Walk(w.workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// 隠しディレクトリやnode_modules等をスキップ
		relativePath := strings.TrimPrefix(path, w.workDir)
		if strings.Contains(relativePath, "/.") || strings.Contains(relativePath, "node_modules") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		if !info.IsDir() {
			count++
		}
		
		return nil
	})
	
	return count
}

// ファイルが変更されたかチェック
func (w *Watcher) hasFileChanged(filename string) bool {
	fullPath := filepath.Join(w.workDir, filename)
	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}
	
	lastMod, exists := w.watchedFiles[fullPath]
	w.watchedFiles[fullPath] = info.ModTime()
	
	return !exists || info.ModTime().After(lastMod)
}

// ファイルが存在するかチェック
func (w *Watcher) fileExists(filename string) bool {
	_, err := os.Stat(filepath.Join(w.workDir, filename))
	return err == nil
}

// ディレクトリかチェック
func (w *Watcher) isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// 変更情報をフォーマットして表示
func (w *Watcher) FormatChange(change ChangeInfo) string {
	var color string
	var icon string
	
	switch change.Severity {
	case SeverityHigh:
		color = "\033[31m" // 赤
		icon = "🚨"
	case SeverityMedium:
		color = "\033[33m" // 黄
		icon = "⚠️"
	default:
		color = "\033[36m" // シアン
		icon = "ℹ️"
	}
	
	reset := "\033[0m"
	
	return fmt.Sprintf("%s%s %s%s", color, icon, change.Description, reset)
}

// 監視を停止
func (w *Watcher) StopWatching() {
	w.isWatching = false
}