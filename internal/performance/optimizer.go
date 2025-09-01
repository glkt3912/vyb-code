package performance

import (
	"runtime"
	"sync"
	"time"
)

// パフォーマンス最適化を管理する構造体
type Optimizer struct {
	mu sync.RWMutex

	// キャッシュ設定
	cacheEnabled    bool
	cacheMaxSize    int
	cacheExpiration time.Duration

	// 並行処理設定
	maxConcurrency int
	workerPoolSize int

	// メモリ管理
	gcThreshold int64 // メモリ使用量がこの値を超えたらGCを実行
	gcInterval  time.Duration
	lastGC      time.Time
}

// 最適化設定のコンストラクタ
func NewOptimizer() *Optimizer {
	return &Optimizer{
		cacheEnabled:    true,
		cacheMaxSize:    1000,
		cacheExpiration: 30 * time.Minute,
		maxConcurrency:  runtime.NumCPU(),
		workerPoolSize:  4,
		gcThreshold:     100 * 1024 * 1024, // 100MB
		gcInterval:      5 * time.Minute,
		lastGC:          time.Now(),
	}
}

// キャッシュ設定の更新
func (o *Optimizer) SetCacheConfig(enabled bool, maxSize int, expiration time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.cacheEnabled = enabled
	o.cacheMaxSize = maxSize
	o.cacheExpiration = expiration
}

// 並行処理設定の更新
func (o *Optimizer) SetConcurrencyConfig(maxConcurrency, workerPoolSize int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if maxConcurrency <= 0 {
		maxConcurrency = runtime.NumCPU()
	}
	if workerPoolSize <= 0 {
		workerPoolSize = 4
	}

	o.maxConcurrency = maxConcurrency
	o.workerPoolSize = workerPoolSize
}

// メモリ管理設定の更新
func (o *Optimizer) SetMemoryConfig(gcThreshold int64, gcInterval time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.gcThreshold = gcThreshold
	o.gcInterval = gcInterval
}

// 自動ガベージコレクション
func (o *Optimizer) AutoGC() {
	o.mu.Lock()
	defer o.mu.Unlock()

	now := time.Now()
	if now.Sub(o.lastGC) < o.gcInterval {
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// メモリ使用量がしきい値を超えた場合、または定期実行時間が経過した場合
	if int64(m.Alloc) > o.gcThreshold || now.Sub(o.lastGC) > o.gcInterval {
		runtime.GC()
		o.lastGC = now

		// メトリクスに記録
		GetMetrics().UpdateMemoryUsage(int64(m.Alloc))
	}
}

// 並行処理用のワーカープール管理
type WorkerPool struct {
	jobs    chan func()
	workers int
	wg      sync.WaitGroup
}

// ワーカープールの作成
func (o *Optimizer) NewWorkerPool() *WorkerPool {
	o.mu.RLock()
	workerCount := o.workerPoolSize
	o.mu.RUnlock()

	wp := &WorkerPool{
		jobs:    make(chan func(), workerCount*2),
		workers: workerCount,
	}

	// ワーカーゴルーチンを開始
	for i := 0; i < workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

// ワーカーゴルーチン
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for job := range wp.jobs {
		job()
	}
}

// ジョブの追加
func (wp *WorkerPool) Submit(job func()) {
	wp.jobs <- job
}

// ワーカープールの停止
func (wp *WorkerPool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
}

// システム情報の取得
func GetSystemInfo() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"go_version":     runtime.Version(),
		"num_cpu":        runtime.NumCPU(),
		"num_goroutine":  runtime.NumGoroutine(),
		"memory_alloc":   m.Alloc,
		"memory_sys":     m.Sys,
		"gc_runs":        m.NumGC,
		"gc_pause_total": m.PauseTotalNs,
	}
}
