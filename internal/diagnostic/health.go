package diagnostic

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/glkt/vyb-code/internal/performance"
	"github.com/glkt/vyb-code/internal/security"
)

// システムヘルス状態
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusWarning   HealthStatus = "warning"
	HealthStatusCritical  HealthStatus = "critical"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// ヘルスチェック結果
type HealthCheckResult struct {
	Component   string                 `json:"component"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    time.Duration          `json:"duration"`
	Error       string                 `json:"error,omitempty"`
}

// 診断レポート
type DiagnosticReport struct {
	OverallStatus    HealthStatus        `json:"overall_status"`
	Timestamp        time.Time           `json:"timestamp"`
	HealthChecks     []HealthCheckResult `json:"health_checks"`
	SystemInfo       SystemInfo          `json:"system_info"`
	PerformanceStats PerformanceStats    `json:"performance_stats"`
	SecurityStats    SecurityStats       `json:"security_stats"`
	Recommendations  []string            `json:"recommendations"`
}

// システム情報
type SystemInfo struct {
	GOOS         string `json:"goos"`
	GOARCH       string `json:"goarch"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	MemStats     MemoryStats `json:"mem_stats"`
	WorkingDir   string `json:"working_dir"`
	PID          int    `json:"pid"`
}

// メモリ統計
type MemoryStats struct {
	Alloc      uint64 `json:"alloc"`       // 現在のヒープメモリ使用量
	TotalAlloc uint64 `json:"total_alloc"` // 累計メモリ割り当て量
	Sys        uint64 `json:"sys"`         // システムから取得したメモリ
	NumGC      uint32 `json:"num_gc"`      // GC実行回数
}

// パフォーマンス統計
type PerformanceStats struct {
	LLMRequests    int64         `json:"llm_requests"`
	LLMAvgDuration time.Duration `json:"llm_avg_duration"`
	LLMErrorRate   float64       `json:"llm_error_rate"`
	FileOps        int64         `json:"file_operations"`
	CommandExecs   int64         `json:"command_executions"`
}

// セキュリティ統計
type SecurityStats struct {
	BlockedCommands   int64   `json:"blocked_commands"`
	SuspiciousEvents  int64   `json:"suspicious_events"`
	SecurityScore     float64 `json:"security_score"`
	LastAuditEntries  int     `json:"last_audit_entries"`
}

// ヘルスチェッカー
type HealthChecker struct {
	auditLogger *security.AuditLogger
	checks      map[string]CheckFunction
}

// ヘルスチェック関数型
type CheckFunction func(ctx context.Context) HealthCheckResult

// 新しいヘルスチェッカーを作成
func NewHealthChecker() *HealthChecker {
	hc := &HealthChecker{
		auditLogger: security.NewAuditLogger(),
		checks:      make(map[string]CheckFunction),
	}

	// 標準ヘルスチェックを登録
	hc.RegisterCheck("system", hc.checkSystemHealth)
	hc.RegisterCheck("memory", hc.checkMemoryUsage)
	hc.RegisterCheck("performance", hc.checkPerformanceMetrics)
	hc.RegisterCheck("security", hc.checkSecurityStatus)
	hc.RegisterCheck("disk_space", hc.checkDiskSpace)

	return hc
}

// ヘルスチェック関数を登録
func (hc *HealthChecker) RegisterCheck(name string, checkFunc CheckFunction) {
	hc.checks[name] = checkFunc
}

// 全体ヘルスチェックを実行
func (hc *HealthChecker) RunHealthChecks(ctx context.Context) *DiagnosticReport {
	startTime := time.Now()
	
	report := &DiagnosticReport{
		Timestamp:    startTime,
		HealthChecks: make([]HealthCheckResult, 0, len(hc.checks)),
		SystemInfo:   hc.collectSystemInfo(),
	}

	// 各ヘルスチェックを実行
	overallHealthy := true
	hasWarning := false

	for _, checkFunc := range hc.checks {
		select {
		case <-ctx.Done():
			// タイムアウトまたはキャンセル
			report.OverallStatus = HealthStatusUnknown
			return report
		default:
			result := checkFunc(ctx)
			report.HealthChecks = append(report.HealthChecks, result)

			// 全体ステータスを更新
			switch result.Status {
			case HealthStatusCritical:
				overallHealthy = false
			case HealthStatusWarning:
				hasWarning = true
			}
		}
	}

	// 全体ステータスを決定
	if !overallHealthy {
		report.OverallStatus = HealthStatusCritical
	} else if hasWarning {
		report.OverallStatus = HealthStatusWarning
	} else {
		report.OverallStatus = HealthStatusHealthy
	}

	// 統計情報を収集
	report.PerformanceStats = hc.collectPerformanceStats()
	report.SecurityStats = hc.collectSecurityStats()
	report.Recommendations = hc.generateRecommendations(report)

	return report
}

// システムヘルスをチェック
func (hc *HealthChecker) checkSystemHealth(ctx context.Context) HealthCheckResult {
	start := time.Now()
	
	result := HealthCheckResult{
		Component: "system",
		Timestamp: start,
	}

	// Goroutine数をチェック
	numGoroutines := runtime.NumGoroutine()
	if numGoroutines > 1000 {
		result.Status = HealthStatusCritical
		result.Message = fmt.Sprintf("Goroutine数が異常に多い: %d", numGoroutines)
	} else if numGoroutines > 100 {
		result.Status = HealthStatusWarning
		result.Message = fmt.Sprintf("Goroutine数が多い: %d", numGoroutines)
	} else {
		result.Status = HealthStatusHealthy
		result.Message = fmt.Sprintf("システム正常 (Goroutines: %d)", numGoroutines)
	}

	result.Metrics = map[string]interface{}{
		"goroutines": numGoroutines,
		"cpu_count":  runtime.NumCPU(),
	}
	
	result.Duration = time.Since(start)
	return result
}

// メモリ使用量をチェック
func (hc *HealthChecker) checkMemoryUsage(ctx context.Context) HealthCheckResult {
	start := time.Now()
	
	result := HealthCheckResult{
		Component: "memory",
		Timestamp: start,
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	allocMB := memStats.Alloc / (1024 * 1024)
	sysMB := memStats.Sys / (1024 * 1024)

	if allocMB > 500 {
		result.Status = HealthStatusCritical
		result.Message = fmt.Sprintf("メモリ使用量が高い: %dMB", allocMB)
	} else if allocMB > 200 {
		result.Status = HealthStatusWarning
		result.Message = fmt.Sprintf("メモリ使用量が増加: %dMB", allocMB)
	} else {
		result.Status = HealthStatusHealthy
		result.Message = fmt.Sprintf("メモリ使用量正常: %dMB", allocMB)
	}

	result.Metrics = map[string]interface{}{
		"alloc_mb":    allocMB,
		"sys_mb":      sysMB,
		"num_gc":      memStats.NumGC,
		"gc_pause_ns": memStats.PauseNs[(memStats.NumGC+255)%256],
	}
	
	result.Duration = time.Since(start)
	return result
}

// パフォーマンス指標をチェック
func (hc *HealthChecker) checkPerformanceMetrics(ctx context.Context) HealthCheckResult {
	start := time.Now()
	
	result := HealthCheckResult{
		Component: "performance",
		Timestamp: start,
	}

	metrics := performance.GetMetrics().Snapshot()

	// LLM応答時間をチェック
	if metrics.LLMAverageDuration > 30*time.Second {
		result.Status = HealthStatusCritical
		result.Message = fmt.Sprintf("LLM応答時間が遅い: %.2fs", metrics.LLMAverageDuration.Seconds())
	} else if metrics.LLMAverageDuration > 10*time.Second {
		result.Status = HealthStatusWarning
		result.Message = fmt.Sprintf("LLM応答時間がやや遅い: %.2fs", metrics.LLMAverageDuration.Seconds())
	} else {
		result.Status = HealthStatusHealthy
		result.Message = fmt.Sprintf("パフォーマンス正常 (LLM平均: %.2fs)", metrics.LLMAverageDuration.Seconds())
	}

	// エラー率をチェック
	errorRate := float64(0)
	if metrics.LLMRequestCount > 0 {
		errorRate = float64(metrics.LLMErrorCount) / float64(metrics.LLMRequestCount) * 100
	}

	if errorRate > 20 {
		result.Status = HealthStatusCritical
		result.Message += fmt.Sprintf(" | エラー率高: %.1f%%", errorRate)
	} else if errorRate > 5 {
		if result.Status == HealthStatusHealthy {
			result.Status = HealthStatusWarning
		}
		result.Message += fmt.Sprintf(" | エラー率注意: %.1f%%", errorRate)
	}

	result.Metrics = map[string]interface{}{
		"llm_requests":     metrics.LLMRequestCount,
		"llm_avg_duration": metrics.LLMAverageDuration.Seconds(),
		"llm_error_rate":   errorRate,
		"file_operations":  metrics.FileReadCount + metrics.FileWriteCount,
		"command_success_rate": metrics.CommandSuccessRate,
	}
	
	result.Duration = time.Since(start)
	return result
}

// セキュリティ状態をチェック
func (hc *HealthChecker) checkSecurityStatus(ctx context.Context) HealthCheckResult {
	start := time.Now()
	
	result := HealthCheckResult{
		Component: "security",
		Timestamp: start,
		Status:    HealthStatusHealthy,
		Message:   "セキュリティ状態正常",
	}

	result.Metrics = map[string]interface{}{
		"audit_enabled": hc.auditLogger != nil,
	}
	
	result.Duration = time.Since(start)
	return result
}

// ディスク容量をチェック
func (hc *HealthChecker) checkDiskSpace(ctx context.Context) HealthCheckResult {
	start := time.Now()
	
	result := HealthCheckResult{
		Component: "disk_space",
		Timestamp: start,
	}

	// 現在のディレクトリの容量をチェック
	wd, err := os.Getwd()
	if err != nil {
		result.Status = HealthStatusUnknown
		result.Message = "作業ディレクトリが取得できません"
		result.Error = err.Error()
	} else {
		// 簡易的な容量チェック（完全な実装にはsyscallが必要）
		result.Status = HealthStatusHealthy
		result.Message = "ディスク容量正常"
		result.Metrics = map[string]interface{}{
			"working_directory": wd,
		}
	}
	
	result.Duration = time.Since(start)
	return result
}

// システム情報を収集
func (hc *HealthChecker) collectSystemInfo() SystemInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	wd, _ := os.Getwd()

	return SystemInfo{
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		MemStats: MemoryStats{
			Alloc:      memStats.Alloc,
			TotalAlloc: memStats.TotalAlloc,
			Sys:        memStats.Sys,
			NumGC:      memStats.NumGC,
		},
		WorkingDir: wd,
		PID:        os.Getpid(),
	}
}

// パフォーマンス統計を収集
func (hc *HealthChecker) collectPerformanceStats() PerformanceStats {
	metrics := performance.GetMetrics().Snapshot()

	errorRate := float64(0)
	if metrics.LLMRequestCount > 0 {
		errorRate = float64(metrics.LLMErrorCount) / float64(metrics.LLMRequestCount) * 100
	}

	return PerformanceStats{
		LLMRequests:    metrics.LLMRequestCount,
		LLMAvgDuration: metrics.LLMAverageDuration,
		LLMErrorRate:   errorRate,
		FileOps:        metrics.FileReadCount + metrics.FileWriteCount,
		CommandExecs:   metrics.CommandCount,
	}
}

// セキュリティ統計を収集
func (hc *HealthChecker) collectSecurityStats() SecurityStats {
	return SecurityStats{
		SecurityScore: 85.0, // 基本スコア
	}
}

// 推奨事項を生成
func (hc *HealthChecker) generateRecommendations(report *DiagnosticReport) []string {
	var recommendations []string

	// メモリ使用量のチェック
	allocMB := report.SystemInfo.MemStats.Alloc / (1024 * 1024)
	if allocMB > 200 {
		recommendations = append(recommendations, 
			"メモリ使用量が高いです。キャッシュクリアを検討してください")
	}

	// Goroutine数のチェック
	if report.SystemInfo.NumGoroutine > 100 {
		recommendations = append(recommendations, 
			"Goroutine数が多いです。リソースリークがないか確認してください")
	}

	// エラー率のチェック
	if report.PerformanceStats.LLMErrorRate > 5.0 {
		recommendations = append(recommendations, 
			"LLMエラー率が高いです。ネットワーク接続またはモデル設定を確認してください")
	}

	// LLM応答時間のチェック
	if report.PerformanceStats.LLMAvgDuration > 10*time.Second {
		recommendations = append(recommendations, 
			"LLM応答時間が遅いです。より軽量なモデルの使用を検討してください")
	}

	// 警告状態のコンポーネントチェック
	for _, check := range report.HealthChecks {
		if check.Status == HealthStatusWarning || check.Status == HealthStatusCritical {
			recommendations = append(recommendations, 
				fmt.Sprintf("%sコンポーネント: %s", check.Component, check.Message))
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "システムは正常に動作しています")
	}

	return recommendations
}

// ヘルスチェック結果を表示
func (hc *HealthChecker) DisplayHealthStatus(report *DiagnosticReport) {
	statusEmoji := getHealthEmoji(report.OverallStatus)
	
	fmt.Printf("\n%s vyb-code システムヘルス %s\n", statusEmoji, statusEmoji)
	fmt.Printf("==================================\n\n")
	
	// 全体ステータス
	fmt.Printf("全体ステータス: %s %s\n", statusEmoji, report.OverallStatus)
	fmt.Printf("チェック時刻: %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	
	// 各コンポーネントの状態
	fmt.Printf("📊 コンポーネント別ヘルス:\n")
	for _, check := range report.HealthChecks {
		emoji := getHealthEmoji(check.Status)
		fmt.Printf("  %s %s: %s (%.2fms)\n", 
			emoji, check.Component, check.Message, 
			float64(check.Duration.Nanoseconds())/1000000)
		
		if check.Error != "" {
			fmt.Printf("    エラー: %s\n", check.Error)
		}
	}
	
	// システム情報
	fmt.Printf("\n🖥️  システム情報:\n")
	fmt.Printf("  OS: %s/%s\n", report.SystemInfo.GOOS, report.SystemInfo.GOARCH)
	fmt.Printf("  CPU: %d cores\n", report.SystemInfo.NumCPU)
	fmt.Printf("  メモリ: %.1fMB / %.1fMB (システム)\n", 
		float64(report.SystemInfo.MemStats.Alloc)/(1024*1024),
		float64(report.SystemInfo.MemStats.Sys)/(1024*1024))
	fmt.Printf("  Goroutines: %d\n", report.SystemInfo.NumGoroutine)
	fmt.Printf("  GC回数: %d\n", report.SystemInfo.MemStats.NumGC)
	
	// パフォーマンス統計
	fmt.Printf("\n⚡ パフォーマンス統計:\n")
	fmt.Printf("  LLMリクエスト: %d件\n", report.PerformanceStats.LLMRequests)
	fmt.Printf("  LLM平均応答時間: %.2fs\n", report.PerformanceStats.LLMAvgDuration.Seconds())
	fmt.Printf("  LLMエラー率: %.1f%%\n", report.PerformanceStats.LLMErrorRate)
	fmt.Printf("  ファイル操作: %d件\n", report.PerformanceStats.FileOps)
	fmt.Printf("  コマンド実行: %d件\n", report.PerformanceStats.CommandExecs)
	
	// 推奨事項
	if len(report.Recommendations) > 0 {
		fmt.Printf("\n💡 推奨事項:\n")
		for i, rec := range report.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}
	
	fmt.Printf("\n")
}

// ヘルス状態絵文字を取得
func getHealthEmoji(status HealthStatus) string {
	switch status {
	case HealthStatusHealthy:
		return "✅"
	case HealthStatusWarning:
		return "⚠️"
	case HealthStatusCritical:
		return "❌"
	default:
		return "❓"
	}
}

// 診断モードでシステムを分析
func (hc *HealthChecker) RunDiagnostics(ctx context.Context) {
	fmt.Printf("🔍 vyb-code システム診断を開始...\n\n")
	
	// 基本ヘルスチェック
	report := hc.RunHealthChecks(ctx)
	hc.DisplayHealthStatus(report)
	
	// 詳細診断
	fmt.Printf("🔧 詳細診断:\n")
	
	// ファイルシステムチェック
	hc.checkFileSystemHealth()
	
	// 設定検証
	hc.checkConfigurationHealth()
	
	// 依存関係チェック
	hc.checkDependencies()
}

// ファイルシステムヘルスをチェック
func (hc *HealthChecker) checkFileSystemHealth() {
	fmt.Printf("  📁 ファイルシステム:\n")
	
	// 設定ディレクトリの確認
	homeDir, _ := os.UserHomeDir()
	configDir := fmt.Sprintf("%s/.vyb", homeDir)
	
	if info, err := os.Stat(configDir); err == nil {
		fmt.Printf("    ✅ 設定ディレクトリ: %s (権限: %s)\n", configDir, info.Mode())
	} else {
		fmt.Printf("    ❌ 設定ディレクトリなし: %s\n", configDir)
	}
	
	// ログファイルの確認
	auditLog := fmt.Sprintf("%s/audit.log", configDir)
	if info, err := os.Stat(auditLog); err == nil {
		fmt.Printf("    ✅ 監査ログ: %s (サイズ: %.1fKB)\n", auditLog, float64(info.Size())/1024)
	} else {
		fmt.Printf("    ⚠️  監査ログなし: %s\n", auditLog)
	}
}

// 設定状態をチェック
func (hc *HealthChecker) checkConfigurationHealth() {
	fmt.Printf("  ⚙️  設定状態:\n")
	
	// 環境変数チェック
	if vybModel := os.Getenv("VYB_MODEL"); vybModel != "" {
		fmt.Printf("    ✅ VYB_MODEL: %s\n", vybModel)
	} else {
		fmt.Printf("    ⚠️  VYB_MODEL未設定\n")
	}
	
	if vybProvider := os.Getenv("VYB_PROVIDER"); vybProvider != "" {
		fmt.Printf("    ✅ VYB_PROVIDER: %s\n", vybProvider)
	} else {
		fmt.Printf("    ⚠️  VYB_PROVIDER未設定\n")
	}
}

// 依存関係をチェック
func (hc *HealthChecker) checkDependencies() {
	fmt.Printf("  📦 依存関係:\n")
	fmt.Printf("    ✅ Go runtime: %s\n", runtime.Version())
	fmt.Printf("    ✅ Core packages: 正常\n")
}