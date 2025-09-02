package diagnostic

import (
	"context"
	"testing"
	"time"
)

// TestNewHealthChecker は新しいヘルスチェッカー作成をテストする
func TestNewHealthChecker(t *testing.T) {
	checker := NewHealthChecker()

	if checker == nil {
		t.Fatal("ヘルスチェッカーの作成に失敗")
	}

	if checker.auditLogger == nil {
		t.Error("監査ロガーが初期化されていません")
	}

	if len(checker.checks) == 0 {
		t.Error("ヘルスチェック関数が登録されていません")
	}

	// 標準チェックが登録されているか確認
	expectedChecks := []string{"system", "memory", "performance", "security", "disk_space"}
	for _, checkName := range expectedChecks {
		if _, exists := checker.checks[checkName]; !exists {
			t.Errorf("標準チェック '%s' が登録されていません", checkName)
		}
	}
}

// TestRegisterCheck はヘルスチェック関数登録をテストする
func TestRegisterCheck(t *testing.T) {
	checker := NewHealthChecker()

	testCheckCalled := false
	testCheck := func(ctx context.Context) HealthCheckResult {
		testCheckCalled = true
		return HealthCheckResult{
			Component: "test",
			Status:    HealthStatusHealthy,
			Message:   "テストチェック",
			Timestamp: time.Now(),
		}
	}

	checker.RegisterCheck("test_check", testCheck)

	if _, exists := checker.checks["test_check"]; !exists {
		t.Error("カスタムチェックが登録されていません")
	}

	// チェックが実行されることを確認
	ctx := context.Background()
	report := checker.RunHealthChecks(ctx)

	if !testCheckCalled {
		t.Error("登録されたチェック関数が呼び出されませんでした")
	}

	// テストチェックの結果が含まれているか確認
	found := false
	for _, check := range report.HealthChecks {
		if check.Component == "test" {
			found = true
			break
		}
	}

	if !found {
		t.Error("テストチェックの結果がレポートに含まれていません")
	}
}

// TestRunHealthChecks はヘルスチェック実行をテストする
func TestRunHealthChecks(t *testing.T) {
	checker := NewHealthChecker()

	ctx := context.Background()
	report := checker.RunHealthChecks(ctx)

	if report == nil {
		t.Fatal("ヘルスチェックレポートがnilです")
	}

	if report.OverallStatus == "" {
		t.Error("全体ステータスが設定されていません")
	}

	if len(report.HealthChecks) == 0 {
		t.Error("ヘルスチェック結果がありません")
	}

	if report.Timestamp.IsZero() {
		t.Error("タイムスタンプが設定されていません")
	}

	// 必須フィールドをチェック
	if report.SystemInfo.GOOS == "" {
		t.Error("システム情報が収集されていません")
	}

	if len(report.Recommendations) == 0 {
		t.Error("推奨事項が生成されていません")
	}
}

// TestSystemHealthCheck はシステムヘルスチェックをテストする
func TestSystemHealthCheck(t *testing.T) {
	checker := NewHealthChecker()

	ctx := context.Background()
	result := checker.checkSystemHealth(ctx)

	if result.Component != "system" {
		t.Errorf("期待コンポーネント: system, 実際: %s", result.Component)
	}

	if result.Status == "" {
		t.Error("ヘルス状態が設定されていません")
	}

	if result.Message == "" {
		t.Error("ヘルスメッセージが設定されていません")
	}

	if result.Duration == 0 {
		t.Error("実行時間が記録されていません")
	}

	// メトリクスをチェック
	if result.Metrics == nil {
		t.Error("メトリクスが収集されていません")
	}

	if goroutines, exists := result.Metrics["goroutines"]; !exists {
		t.Error("Goroutine数メトリクスがありません")
	} else if goroutines.(int) <= 0 {
		t.Error("Goroutine数が無効です")
	}
}

// TestMemoryHealthCheck はメモリヘルスチェックをテストする
func TestMemoryHealthCheck(t *testing.T) {
	checker := NewHealthChecker()

	ctx := context.Background()
	result := checker.checkMemoryUsage(ctx)

	if result.Component != "memory" {
		t.Errorf("期待コンポーネント: memory, 実際: %s", result.Component)
	}

	if result.Status == "" {
		t.Error("メモリヘルス状態が設定されていません")
	}

	// メトリクスをチェック
	if result.Metrics == nil {
		t.Error("メモリメトリクスが収集されていません")
	}

	requiredMetrics := []string{"alloc_mb", "sys_mb", "num_gc"}
	for _, metric := range requiredMetrics {
		if _, exists := result.Metrics[metric]; !exists {
			t.Errorf("メトリクス '%s' がありません", metric)
		}
	}
}

// TestPerformanceHealthCheck はパフォーマンスヘルスチェックをテストする
func TestPerformanceHealthCheck(t *testing.T) {
	checker := NewHealthChecker()

	ctx := context.Background()
	result := checker.checkPerformanceMetrics(ctx)

	if result.Component != "performance" {
		t.Errorf("期待コンポーネント: performance, 実際: %s", result.Component)
	}

	if result.Status == "" {
		t.Error("パフォーマンスヘルス状態が設定されていません")
	}

	// メトリクスをチェック
	expectedMetrics := []string{"llm_requests", "llm_avg_duration", "llm_error_rate"}
	for _, metric := range expectedMetrics {
		if _, exists := result.Metrics[metric]; !exists {
			t.Errorf("パフォーマンスメトリクス '%s' がありません", metric)
		}
	}
}

// TestContextCancellation はコンテキストキャンセルをテストする
func TestContextCancellation(t *testing.T) {
	checker := NewHealthChecker()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 即座にキャンセル

	report := checker.RunHealthChecks(ctx)

	if report.OverallStatus != HealthStatusUnknown {
		t.Errorf("キャンセル時の期待ステータス: %s, 実際: %s",
			HealthStatusUnknown, report.OverallStatus)
	}
}

// TestHealthStatusEmoji はヘルス状態絵文字をテストする
func TestHealthStatusEmoji(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthStatusHealthy, "✅"},
		{HealthStatusWarning, "⚠️"},
		{HealthStatusCritical, "❌"},
		{HealthStatusUnknown, "❓"},
	}

	for _, test := range tests {
		emoji := getHealthEmoji(test.status)
		if emoji != test.expected {
			t.Errorf("ステータス %s の期待絵文字: %s, 実際: %s",
				test.status, test.expected, emoji)
		}
	}
}

// TestCollectSystemInfo はシステム情報収集をテストする
func TestCollectSystemInfo(t *testing.T) {
	checker := NewHealthChecker()

	sysInfo := checker.collectSystemInfo()

	if sysInfo.GOOS == "" {
		t.Error("OSが設定されていません")
	}

	if sysInfo.GOARCH == "" {
		t.Error("アーキテクチャが設定されていません")
	}

	if sysInfo.NumCPU <= 0 {
		t.Error("CPU数が無効です")
	}

	if sysInfo.PID <= 0 {
		t.Error("PIDが無効です")
	}

	if sysInfo.WorkingDir == "" {
		t.Error("作業ディレクトリが設定されていません")
	}
}

// TestGenerateRecommendations は推奨事項生成をテストする
func TestGenerateRecommendations(t *testing.T) {
	checker := NewHealthChecker()

	// 正常な状態のレポート
	healthyReport := &DiagnosticReport{
		OverallStatus: HealthStatusHealthy,
		HealthChecks: []HealthCheckResult{
			{Status: HealthStatusHealthy, Component: "test"},
		},
		SystemInfo: SystemInfo{
			NumGoroutine: 10,
			MemStats:     MemoryStats{Alloc: 50 * 1024 * 1024}, // 50MB
		},
		PerformanceStats: PerformanceStats{
			LLMErrorRate:   2.0,
			LLMAvgDuration: 5 * time.Second,
		},
	}

	recommendations := checker.generateRecommendations(healthyReport)

	if len(recommendations) == 0 {
		t.Error("推奨事項が生成されていません")
	}

	// 問題のある状態のレポート
	problematicReport := &DiagnosticReport{
		SystemInfo: SystemInfo{
			NumGoroutine: 150,                                   // 多すぎる
			MemStats:     MemoryStats{Alloc: 300 * 1024 * 1024}, // 300MB
		},
		PerformanceStats: PerformanceStats{
			LLMErrorRate:   15.0,             // 高いエラー率
			LLMAvgDuration: 20 * time.Second, // 遅い応答
		},
	}

	problemRecommendations := checker.generateRecommendations(problematicReport)

	if len(problemRecommendations) <= len(recommendations) {
		t.Error("問題のある状態で推奨事項が増えていません")
	}
}
