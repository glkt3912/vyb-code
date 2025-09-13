package conversation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/analysis"
	"github.com/glkt/vyb-code/internal/config"
)

// 軽量プロジェクト状態監視システム - Phase 2実装
type LightweightMonitor struct {
	config           *config.Config
	lightAnalyzer    *analysis.LightweightAnalyzer
	enabled          bool
	lastCheckTime    time.Time
	lastProjectState *ProjectState
}

// プロジェクト状態
type ProjectState struct {
	ProjectPath   string                 `json:"project_path"`
	LastModified  time.Time              `json:"last_modified"`
	FileCount     int                    `json:"file_count"`
	Language      string                 `json:"language"`
	GitBranch     string                 `json:"git_branch"`
	RecentChanges []FileChange           `json:"recent_changes"`
	HealthScore   float64                `json:"health_score"` // 0.0-1.0
	IssueCount    int                    `json:"issue_count"`
	Notifications []StateNotification    `json:"notifications"`
	TechStack     []string               `json:"tech_stack"`
	LastAnalyzed  time.Time              `json:"last_analyzed"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ファイル変更情報
type FileChange struct {
	FilePath   string    `json:"file_path"`
	ChangeType string    `json:"change_type"` // "modified", "added", "deleted"
	Timestamp  time.Time `json:"timestamp"`
	Size       int64     `json:"size"`
	Language   string    `json:"language"`
}

// 状態通知
type StateNotification struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "info", "warning", "error", "success"
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	Severity    string    `json:"severity"` // "low", "medium", "high", "critical"
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Dismissed   bool      `json:"dismissed"`
	ActionItems []string  `json:"action_items"`
}

// 新しい軽量監視システムを作成
func NewLightweightMonitor(cfg *config.Config) *LightweightMonitor {
	if !cfg.IsProactiveEnabled() || !cfg.Proactive.ProjectMonitoring {
		return &LightweightMonitor{enabled: false}
	}

	analysisConfig := &analysis.AnalysisConfig{
		EnableCaching:  true,
		CacheExpiry:    10 * time.Minute,
		AnalysisDepth:  "quick",
		IncludeTests:   false,
		SecurityScan:   false,
		QualityMetrics: false,
		ExcludePatterns: []string{
			"node_modules/**", "vendor/**", ".git/**",
			"dist/**", "build/**", "target/**",
			"*.log", "*.tmp", ".cache/**",
		},
		MaxFileSize: 128 * 1024,      // 128KB制限（監視用により軽量化）
		Timeout:     3 * time.Second, // 短いタイムアウト
	}

	return &LightweightMonitor{
		config:        cfg,
		lightAnalyzer: analysis.NewLightweightAnalyzer(analysisConfig),
		enabled:       true,
		lastCheckTime: time.Now(),
	}
}

// プロジェクト状態をチェック
func (lm *LightweightMonitor) CheckProjectState(projectPath string) (*ProjectState, error) {
	if !lm.enabled {
		return nil, fmt.Errorf("監視機能が無効です")
	}

	// 頻繁なチェックを避けるため、最小間隔を設定
	if time.Since(lm.lastCheckTime) < 30*time.Second {
		if lm.lastProjectState != nil {
			return lm.lastProjectState, nil
		}
	}

	state := &ProjectState{
		ProjectPath:   projectPath,
		LastAnalyzed:  time.Now(),
		Notifications: make([]StateNotification, 0),
		Metadata:      make(map[string]interface{}),
	}

	// 基本プロジェクト情報を収集
	if err := lm.collectBasicInfo(state); err != nil {
		return nil, err
	}

	// 軽量分析を実行
	if err := lm.performLightweightAnalysis(state); err != nil {
		// 分析エラーは警告として扱い、続行
		state.Notifications = append(state.Notifications, StateNotification{
			ID:        "analysis_warning",
			Type:      "warning",
			Title:     "分析制限",
			Message:   "プロジェクト分析の一部がスキップされました",
			Severity:  "low",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		})
	}

	// 最近の変更を検出
	lm.detectRecentChanges(state)

	// ヘルススコアを計算
	lm.calculateHealthScore(state)

	// 通知を生成
	lm.generateNotifications(state)

	lm.lastProjectState = state
	lm.lastCheckTime = time.Now()

	return state, nil
}

// 基本情報を収集
func (lm *LightweightMonitor) collectBasicInfo(state *ProjectState) error {
	// プロジェクトディレクトリの最終更新時刻を取得
	if info, err := os.Stat(state.ProjectPath); err == nil {
		state.LastModified = info.ModTime()
	}

	// ファイル数をカウント（軽量版）
	fileCount := 0
	err := filepath.Walk(state.ProjectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// 除外パターンをチェック
		relPath, _ := filepath.Rel(state.ProjectPath, path)
		for _, pattern := range []string{"node_modules", "vendor", ".git", "dist", "build"} {
			if strings.Contains(relPath, pattern) {
				return nil
			}
		}

		fileCount++

		// 軽量化のため最大100ファイルまで
		if fileCount > 100 {
			return filepath.SkipDir
		}

		return nil
	})

	state.FileCount = fileCount
	return err
}

// 軽量分析を実行
func (lm *LightweightMonitor) performLightweightAnalysis(state *ProjectState) error {
	// 3秒のタイムアウト付き分析
	analysis, err := lm.lightAnalyzer.AnalyzeWithLevel(state.ProjectPath, analysis.LevelMinimal)
	if err != nil {
		return err
	}

	// 基本情報を状態に反映
	state.Language = analysis.Language

	// 技術スタックを抽出
	techStack := make([]string, 0)
	for _, tech := range analysis.TechStack {
		if tech.Usage == "primary" || tech.Usage == "runtime" {
			techStack = append(techStack, tech.Name)
		}
	}
	state.TechStack = techStack

	return nil
}

// 最近の変更を検出（簡易版）
func (lm *LightweightMonitor) detectRecentChanges(state *ProjectState) {
	recentChanges := make([]FileChange, 0)
	cutoffTime := time.Now().Add(-24 * time.Hour) // 過去24時間

	// 最大5つのファイル変更のみチェック（軽量化）
	count := 0
	err := filepath.Walk(state.ProjectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || count >= 5 {
			return nil
		}

		// 除外パターン
		relPath, _ := filepath.Rel(state.ProjectPath, path)
		for _, pattern := range []string{"node_modules", "vendor", ".git"} {
			if strings.Contains(relPath, pattern) {
				return nil
			}
		}

		if info.ModTime().After(cutoffTime) {
			ext := strings.ToLower(filepath.Ext(path))
			language := "Unknown"

			// 簡易言語検出
			switch ext {
			case ".go":
				language = "Go"
			case ".js", ".mjs":
				language = "JavaScript"
			case ".ts":
				language = "TypeScript"
			case ".py":
				language = "Python"
			case ".java":
				language = "Java"
			case ".rs":
				language = "Rust"
			}

			recentChanges = append(recentChanges, FileChange{
				FilePath:   relPath,
				ChangeType: "modified",
				Timestamp:  info.ModTime(),
				Size:       info.Size(),
				Language:   language,
			})
			count++
		}

		return nil
	})

	if err == nil {
		state.RecentChanges = recentChanges
	}
}

// ヘルススコアを計算
func (lm *LightweightMonitor) calculateHealthScore(state *ProjectState) {
	score := 1.0 // 初期スコア

	// ファイル数に基づく調整
	if state.FileCount == 0 {
		score *= 0.5 // 空プロジェクト
	} else if state.FileCount > 1000 {
		score *= 0.9 // 大規模プロジェクト
	}

	// 最近の活動に基づく調整
	if len(state.RecentChanges) > 0 {
		score *= 1.1 // 積極的な開発
		if score > 1.0 {
			score = 1.0
		}
	} else {
		// 24時間以上更新がない場合
		if time.Since(state.LastModified) > 24*time.Hour {
			score *= 0.8
		}
	}

	// 技術スタックの多様性に基づく調整
	if len(state.TechStack) > 3 {
		score *= 0.95 // 複雑性ペナルティ
	}

	state.HealthScore = score
}

// 通知を生成
func (lm *LightweightMonitor) generateNotifications(state *ProjectState) {
	notifications := make([]StateNotification, 0)

	// ヘルススコアに基づく通知
	if state.HealthScore < 0.7 {
		severity := "medium"
		if state.HealthScore < 0.5 {
			severity = "high"
		}

		notifications = append(notifications, StateNotification{
			ID:        "health_score_low",
			Type:      "warning",
			Title:     "プロジェクト状態注意",
			Message:   fmt.Sprintf("プロジェクトのヘルススコア: %.1f%%", state.HealthScore*100),
			Severity:  severity,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(2 * time.Hour),
			ActionItems: []string{
				"最近の変更を確認",
				"テストの実行を検討",
				"依存関係の更新を確認",
			},
		})
	}

	// 最近の活動に基づく通知
	if len(state.RecentChanges) > 5 {
		notifications = append(notifications, StateNotification{
			ID:        "high_activity",
			Type:      "info",
			Title:     "活発な開発",
			Message:   fmt.Sprintf("過去24時間で%dファイルが更新されました", len(state.RecentChanges)),
			Severity:  "low",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		})
	} else if len(state.RecentChanges) == 0 && time.Since(state.LastModified) > 48*time.Hour {
		notifications = append(notifications, StateNotification{
			ID:        "low_activity",
			Type:      "info",
			Title:     "開発活動低下",
			Message:   "48時間以上更新がありません",
			Severity:  "low",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(4 * time.Hour),
			ActionItems: []string{
				"プロジェクトステータスの確認",
				"pending タスクの整理",
			},
		})
	}

	state.Notifications = notifications
}

// 監視レポートを生成
func (lm *LightweightMonitor) GenerateStatusReport(projectPath string) (string, error) {
	state, err := lm.CheckProjectState(projectPath)
	if err != nil {
		return "", err
	}

	var builder strings.Builder

	// ヘッダー
	builder.WriteString("📊 プロジェクト状態レポート\n")
	builder.WriteString("────────────────────────\n")

	// 基本情報
	builder.WriteString(fmt.Sprintf("🏗️  ファイル数: %d\n", state.FileCount))
	if state.Language != "" {
		builder.WriteString(fmt.Sprintf("💻 主言語: %s\n", state.Language))
	}

	// ヘルススコア
	healthIcon := "✅"
	if state.HealthScore < 0.7 {
		healthIcon = "⚠️"
	}
	if state.HealthScore < 0.5 {
		healthIcon = "❌"
	}
	builder.WriteString(fmt.Sprintf("%s ヘルススコア: %.1f%%\n", healthIcon, state.HealthScore*100))

	// 最近の活動
	if len(state.RecentChanges) > 0 {
		builder.WriteString(fmt.Sprintf("📝 最近の更新: %d件 (24時間以内)\n", len(state.RecentChanges)))
	}

	// 技術スタック
	if len(state.TechStack) > 0 {
		builder.WriteString(fmt.Sprintf("🛠️  技術: %s\n", strings.Join(state.TechStack, ", ")))
	}

	// アクティブな通知
	activeNotifications := 0
	for _, notif := range state.Notifications {
		if !notif.Dismissed && time.Now().Before(notif.ExpiresAt) {
			activeNotifications++
		}
	}

	if activeNotifications > 0 {
		builder.WriteString(fmt.Sprintf("🔔 通知: %d件\n", activeNotifications))
	}

	return builder.String(), nil
}

// 軽量監視の有効/無効を切り替え
func (lm *LightweightMonitor) SetEnabled(enabled bool) {
	lm.enabled = enabled
}

// 監視が有効かどうか確認
func (lm *LightweightMonitor) IsEnabled() bool {
	return lm.enabled
}

// 統計情報を取得
func (lm *LightweightMonitor) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"enabled":          lm.enabled,
		"last_check_time":  lm.lastCheckTime,
		"has_cached_state": lm.lastProjectState != nil,
	}

	if lm.lastProjectState != nil {
		stats["cached_file_count"] = lm.lastProjectState.FileCount
		stats["cached_language"] = lm.lastProjectState.Language
		stats["cached_health_score"] = lm.lastProjectState.HealthScore
		stats["active_notifications"] = len(lm.lastProjectState.Notifications)
	}

	return stats
}

// リソースクリーンアップ
func (lm *LightweightMonitor) Close() error {
	lm.enabled = false
	lm.lastProjectState = nil
	return nil
}
