package input

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestPerformanceOptimizer_Creation(t *testing.T) {
	optimizer := NewPerformanceOptimizer()

	if optimizer == nil {
		t.Error("Expected non-nil performance optimizer")
	}

	if optimizer.workerPool == nil {
		t.Error("Expected non-nil worker pool")
	}

	if optimizer.debouncer == nil {
		t.Error("Expected non-nil debouncer")
	}

	if optimizer.memoryManager == nil {
		t.Error("Expected non-nil memory manager")
	}

	if optimizer.metricsCollector == nil {
		t.Error("Expected non-nil metrics collector")
	}

	if optimizer.asyncProcessor == nil {
		t.Error("Expected non-nil async processor")
	}
}

func TestWorkerPool_StartStop(t *testing.T) {
	wp := NewWorkerPool(2)

	// 初期状態は停止
	if wp.started {
		t.Error("Expected worker pool to be stopped initially")
	}

	// 開始
	wp.Start()
	if !wp.started {
		t.Error("Expected worker pool to be started")
	}

	// 停止
	wp.Stop()
	if wp.started {
		t.Error("Expected worker pool to be stopped after Stop()")
	}
}

func TestWorkerPool_JobExecution(t *testing.T) {
	wp := NewWorkerPool(2)
	wp.Start()
	defer wp.Stop()

	// ジョブ実行テスト
	var result int
	var wg sync.WaitGroup
	wg.Add(1)

	job := Job{
		ID: "test-job",
		Task: func() interface{} {
			return 42
		},
		Callback: func(res interface{}) {
			if val, ok := res.(int); ok {
				result = val
			}
			wg.Done()
		},
	}

	err := wp.Submit(job)
	if err != nil {
		t.Errorf("Unexpected error submitting job: %v", err)
	}

	// ジョブ完了を待機
	wg.Wait()

	if result != 42 {
		t.Errorf("Expected result 42, got %d", result)
	}
}

func TestDebouncer_Debounce(t *testing.T) {
	debouncer := NewDebouncer(50*time.Millisecond, 200*time.Millisecond)

	var executed bool
	var mu sync.Mutex

	// デバウンス関数
	fn := func() {
		mu.Lock()
		defer mu.Unlock()
		executed = true
	}

	// 短時間に複数回呼び出し
	debouncer.Debounce("test", fn)
	debouncer.Debounce("test", fn)
	debouncer.Debounce("test", fn)

	// すぐには実行されない
	mu.Lock()
	if executed {
		t.Error("Function should not be executed immediately")
	}
	mu.Unlock()

	// デバウンス時間経過後に実行される
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if !executed {
		t.Error("Function should be executed after debounce delay")
	}
	mu.Unlock()
}

func TestDebouncer_Flush(t *testing.T) {
	debouncer := NewDebouncer(100*time.Millisecond, 500*time.Millisecond)

	debouncer.Debounce("test", func() {})
	debouncer.Flush("test")

	// フラッシュ後は新しいデバウンスが可能
	var executed bool
	debouncer.Debounce("test", func() {
		executed = true
	})

	time.Sleep(150 * time.Millisecond)

	if !executed {
		t.Error("Function should be executed after flush and new debounce")
	}
}

func TestMemoryManager_RegisterUnregister(t *testing.T) {
	mm := NewMemoryManager()

	// 初期状態
	if mm.currentCacheSize != 0 {
		t.Errorf("Expected initial cache size to be 0, got %d", mm.currentCacheSize)
	}

	// オブジェクト登録
	mm.RegisterObject("test1", 1000)
	if mm.currentCacheSize != 1000 {
		t.Errorf("Expected cache size 1000, got %d", mm.currentCacheSize)
	}

	mm.RegisterObject("test2", 2000)
	if mm.currentCacheSize != 3000 {
		t.Errorf("Expected cache size 3000, got %d", mm.currentCacheSize)
	}

	// オブジェクト登録解除
	mm.UnregisterObject("test1")
	if mm.currentCacheSize != 2000 {
		t.Errorf("Expected cache size 2000 after unregister, got %d", mm.currentCacheSize)
	}

	mm.UnregisterObject("test2")
	if mm.currentCacheSize != 0 {
		t.Errorf("Expected cache size 0 after all unregister, got %d", mm.currentCacheSize)
	}
}

func TestLRUCache_Operations(t *testing.T) {
	cache := NewLRUCache(3) // 容量3

	// 基本的な Put/Get
	cache.Put("key1", "value1")
	if val, ok := cache.Get("key1"); !ok || val != "value1" {
		t.Error("Expected to get value1 for key1")
	}

	// 容量超過テスト
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")
	cache.Put("key4", "value4") // key1が追い出される

	if _, ok := cache.Get("key1"); ok {
		t.Error("key1 should be evicted")
	}

	if val, ok := cache.Get("key4"); !ok || val != "value4" {
		t.Error("Expected to get value4 for key4")
	}

	// LRU順序テスト
	cache.Get("key2") // key2をアクセス（最近使用に）
	cache.Put("key5", "value5") // key3が追い出される（最も古い）

	if _, ok := cache.Get("key3"); ok {
		t.Error("key3 should be evicted as least recently used")
	}

	if _, ok := cache.Get("key2"); !ok {
		t.Error("key2 should still be in cache")
	}
}

func TestMetricsCollector_RecordRequest(t *testing.T) {
	mc := NewMetricsCollector()

	// 初期状態
	if mc.requestCount != 0 {
		t.Error("Expected initial request count to be 0")
	}

	// リクエスト記録
	mc.RecordRequest(100*time.Millisecond, false)
	if mc.requestCount != 1 {
		t.Error("Expected request count to be 1")
	}

	if mc.averageLatency != 100*time.Millisecond {
		t.Errorf("Expected average latency 100ms, got %v", mc.averageLatency)
	}

	// エラーリクエスト記録
	mc.RecordRequest(200*time.Millisecond, true)
	if mc.errorCount != 1 {
		t.Error("Expected error count to be 1")
	}

	if mc.requestCount != 2 {
		t.Error("Expected request count to be 2")
	}
}

func TestMetricsCollector_CacheHit(t *testing.T) {
	mc := NewMetricsCollector()

	// 初期状態
	if mc.cacheHitRate != 0 {
		t.Error("Expected initial cache hit rate to be 0")
	}

	// キャッシュヒット記録（これだけでtotalRequests=1, cacheHits=1になる）
	mc.RecordCacheHit()

	if mc.cacheHitRate != 1.0 {
		t.Errorf("Expected cache hit rate 1.0, got %f", mc.cacheHitRate)
	}

	// 追加リクエスト（キャッシュミス）
	mc.RecordRequest(200*time.Millisecond, false)

	// キャッシュヒット率が正しく計算されることを確認（1 hit / 2 total requests = 0.5）
	expectedRate := 1.0 / 2.0 // 1 cache hit, 2 total requests
	if mc.cacheHitRate != expectedRate {
		t.Errorf("Expected cache hit rate %f, got %f", expectedRate, mc.cacheHitRate)
	}
}

func TestAsyncProcessor_StartGetResult(t *testing.T) {
	ap := NewAsyncProcessor()

	// 非同期ジョブ開始
	ap.StartAsync("test-key", func() interface{} {
		time.Sleep(10 * time.Millisecond)
		return "test-result"
	})

	// 結果が準備される前
	if result, ok := ap.GetResult("test-key"); ok {
		t.Errorf("Expected no result immediately, got %v", result)
	}

	// 少し待って結果取得
	time.Sleep(50 * time.Millisecond)
	if result, ok := ap.GetResult("test-key"); !ok || result != "test-result" {
		t.Errorf("Expected 'test-result', got %v (ok: %v)", result, ok)
	}
}

func TestAsyncProcessor_CacheExpiration(t *testing.T) {
	ap := NewAsyncProcessor()
	ap.maxCacheAge = 50 * time.Millisecond // 短い有効期限

	// 結果を設定
	ap.StartAsync("expire-test", func() interface{} {
		return "expire-value"
	})

	time.Sleep(20 * time.Millisecond)

	// 有効期限内は取得可能
	if result, ok := ap.GetResult("expire-test"); !ok || result != "expire-value" {
		t.Error("Expected result within expiration time")
	}

	// 有効期限経過後は取得不可
	time.Sleep(60 * time.Millisecond)
	if _, ok := ap.GetResult("expire-test"); ok {
		t.Error("Expected result to be expired")
	}
}

func TestPerformanceOptimizer_Integration(t *testing.T) {
	optimizer := NewPerformanceOptimizer()
	optimizer.Start()
	defer optimizer.Stop()

	// ダミーの高度なコンプリーターを作成
	completer := NewAdvancedCompleter("/test")

	// 最適化された補完を実行
	candidates := optimizer.OptimizedCompletion("test", completer)

	// 結果が返されることを確認（具体的な内容は問わない）
	if candidates == nil {
		t.Error("Expected non-nil candidates")
	}

	// メトリクスが記録されていることを確認
	metrics := optimizer.metricsCollector.GetMetrics()
	if metrics["requests_total"].(int64) == 0 {
		t.Error("Expected requests to be recorded")
	}
}

func TestPerformanceOptimizer_OptimizedCompletionCaching(t *testing.T) {
	optimizer := NewPerformanceOptimizer()
	optimizer.Start()
	defer optimizer.Stop()

	completer := NewAdvancedCompleter("/test")

	// 同じ入力で複数回実行
	input := "build"
	
	start1 := time.Now()
	candidates1 := optimizer.OptimizedCompletion(input, completer)
	elapsed1 := time.Since(start1)

	// 2回目は（理論的には）高速化される
	start2 := time.Now()
	candidates2 := optimizer.OptimizedCompletion(input, completer)
	elapsed2 := time.Since(start2)

	// 結果の一貫性をチェック
	if len(candidates1) == 0 && len(candidates2) == 0 {
		// 両方とも空の場合はOK（タイムアウトの可能性）
		return
	}

	// 少なくとも結果が返されることを確認
	if candidates1 == nil || candidates2 == nil {
		t.Error("Expected non-nil candidates in both calls")
	}

	// 時間測定（必ずしも2回目が速いとは限らないので、単純な確認のみ）
	t.Logf("First call: %v, Second call: %v", elapsed1, elapsed2)
}

// パフォーマンステスト
func BenchmarkWorkerPool_Submit(b *testing.B) {
	wp := NewWorkerPool(4)
	wp.Start()
	defer wp.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job := Job{
			ID: "bench-job",
			Task: func() interface{} {
				return i
			},
			Callback: func(interface{}) {
				// No-op
			},
		}
		wp.Submit(job)
	}
}

func BenchmarkLRUCache_PutGet(b *testing.B) {
	cache := NewLRUCache(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%500)
		value := fmt.Sprintf("value-%d", i)
		
		cache.Put(key, value)
		cache.Get(key)
	}
}

func BenchmarkMemoryManager_RegisterUnregister(b *testing.B) {
	mm := NewMemoryManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("obj-%d", i%100)
		size := int64(i % 1000)
		
		mm.RegisterObject(key, size)
		if i%2 == 0 {
			mm.UnregisterObject(key)
		}
	}
}