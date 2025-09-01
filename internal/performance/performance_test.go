package performance

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// TestMetricsRecording はメトリクス記録機能をテストする
func TestMetricsRecording(t *testing.T) {
	metrics := &Metrics{}

	// LLMリクエストの記録
	metrics.RecordLLMRequest(100*time.Millisecond, true)
	metrics.RecordLLMRequest(200*time.Millisecond, false)

	if metrics.LLMRequestCount != 2 {
		t.Errorf("期待値: 2, 実際値: %d", metrics.LLMRequestCount)
	}
	if metrics.LLMErrorCount != 1 {
		t.Errorf("期待値: 1, 実際値: %d", metrics.LLMErrorCount)
	}
	if metrics.LLMAverageDuration != 150*time.Millisecond {
		t.Errorf("期待値: 150ms, 実際値: %v", metrics.LLMAverageDuration)
	}
}

// TestCacheOperations はキャッシュ操作をテストする
func TestCacheOperations(t *testing.T) {
	cache := NewCache(3, 1*time.Second)

	// 値の設定
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// 値の取得
	value, exists := cache.Get("key1")
	if !exists {
		t.Error("キーが見つかりません")
	}
	if value != "value1" {
		t.Errorf("期待値: value1, 実際値: %v", value)
	}

	// 容量制限のテスト
	cache.Set("key3", "value3")
	cache.Set("key4", "value4") // これによりkey2が削除されるはず（key1はアクセスしたため）

	// key1は最近アクセスされたため残っているはず
	_, exists = cache.Get("key1")
	if !exists {
		t.Error("最近アクセスされたキーが削除されました")
	}

	// key2は最古のため削除されているはず
	_, exists = cache.Get("key2")
	if exists {
		t.Error("LRU削除が機能していません")
	}
}

// TestCacheExpiration はキャッシュの期限切れをテストする
func TestCacheExpiration(t *testing.T) {
	cache := NewCache(10, 50*time.Millisecond)

	cache.Set("test", "value")

	// 期限切れ前の取得
	value, exists := cache.Get("test")
	if !exists || value != "value" {
		t.Error("期限切れ前に値が取得できません")
	}

	// 期限切れまで待機
	time.Sleep(100 * time.Millisecond)

	// 期限切れ後の取得
	_, exists = cache.Get("test")
	if exists {
		t.Error("期限切れが機能していません")
	}
}

// TestBenchmarker はベンチマーク機能をテストする
func TestBenchmarker(t *testing.T) {
	benchmarker := NewBenchmarker()

	// 成功するテスト関数
	successFunc := func() error {
		time.Sleep(1 * time.Millisecond)
		return nil
	}

	// 失敗するテスト関数
	errorFunc := func() error {
		return errors.New("test error")
	}

	// 成功ケースのベンチマーク
	result1 := benchmarker.BenchmarkFunction("Success Test", 5, successFunc)
	if !result1.Success {
		t.Error("成功ケースのベンチマークが失敗しました")
	}
	if result1.OperationsPerSec <= 0 {
		t.Error("操作/秒が正しく計算されていません")
	}

	// 失敗ケースのベンチマーク
	result2 := benchmarker.BenchmarkFunction("Error Test", 3, errorFunc)
	if result2.Success {
		t.Error("失敗ケースが成功として記録されました")
	}
	if result2.Error == "" {
		t.Error("エラーメッセージが記録されていません")
	}

	// 結果数の確認
	results := benchmarker.GetResults()
	if len(results) != 2 {
		t.Errorf("期待値: 2, 実際値: %d", len(results))
	}
}

// TestOptimizer は最適化設定をテストする
func TestOptimizer(t *testing.T) {
	optimizer := NewOptimizer()

	// キャッシュ設定のテスト
	optimizer.SetCacheConfig(false, 500, 10*time.Minute)

	// 並行処理設定のテスト
	optimizer.SetConcurrencyConfig(8, 2)

	// メモリ設定のテスト
	optimizer.SetMemoryConfig(50*1024*1024, 2*time.Minute)

	// 設定が正しく反映されているかは内部実装のため、エラーが発生しないことを確認
}

// TestWorkerPool はワーカープールをテストする
func TestWorkerPool(t *testing.T) {
	optimizer := NewOptimizer()
	pool := optimizer.NewWorkerPool()

	// テスト用のカウンタ（アトミック操作で競合状態を回避）
	var counter int64
	done := make(chan bool, 5)

	// 5つのジョブを投入
	for i := 0; i < 5; i++ {
		pool.Submit(func() {
			atomic.AddInt64(&counter, 1)
			done <- true
		})
	}

	// すべてのジョブが完了するまで待機
	for i := 0; i < 5; i++ {
		select {
		case <-done:
			// ジョブ完了
		case <-time.After(1 * time.Second):
			t.Fatal("ワーカープールがタイムアウトしました")
		}
	}

	pool.Stop()

	finalCounter := atomic.LoadInt64(&counter)
	if finalCounter != 5 {
		t.Errorf("期待値: 5, 実際値: %d", finalCounter)
	}
}

// TestMeasureExecution は実行時間計測をテストする
func TestMeasureExecution(t *testing.T) {
	duration := MeasureExecution("test", func() {
		time.Sleep(10 * time.Millisecond)
	})

	if duration < 10*time.Millisecond {
		t.Errorf("実行時間が短すぎます: %v", duration)
	}
	if duration > 50*time.Millisecond {
		t.Errorf("実行時間が長すぎます: %v", duration)
	}
}
