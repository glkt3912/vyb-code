package input

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// パフォーマンス最適化システム
type PerformanceOptimizer struct {
	workerPool      *WorkerPool
	debouncer       *Debouncer
	memoryManager   *MemoryManager
	metricsCollector *MetricsCollector
	asyncProcessor  *AsyncProcessor
}

// ワーカープール（並列処理用）
type WorkerPool struct {
	workers    int
	jobChannel chan Job
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	started    bool
	mu         sync.RWMutex
}

// ジョブ定義
type Job struct {
	ID       string
	Task     func() interface{}
	Callback func(interface{})
	Priority int
	Created  time.Time
}

// デバウンサー（連続入力の制御）
type Debouncer struct {
	mu       sync.Mutex
	timers   map[string]*time.Timer
	delay    time.Duration
	maxDelay time.Duration
}

// メモリ管理システム
type MemoryManager struct {
	mu                 sync.RWMutex
	maxCacheSize       int64
	currentCacheSize   int64
	gcThreshold        float64
	lastCleanup        time.Time
	cleanupInterval    time.Duration
	objectSizes        map[string]int64
	lruCache          *LRUCache
}

// LRUキャッシュ実装
type LRUCache struct {
	capacity int
	cache    map[string]*Node
	head     *Node
	tail     *Node
	mu       sync.RWMutex
}

// LRUキャッシュノード
type Node struct {
	key   string
	value interface{}
	prev  *Node
	next  *Node
}

// メトリクス収集システム
type MetricsCollector struct {
	mu                sync.RWMutex
	requestCount      int64
	averageLatency    time.Duration
	peakLatency       time.Duration
	errorCount        int64
	cacheHitRate      float64
	totalRequests     int64
	cacheHits         int64
	memoryUsage       int64
	gcCount           int64
	startTime         time.Time
}

// 非同期処理システム
type AsyncProcessor struct {
	completionJobs chan Job
	resultCache    map[string]interface{}
	cacheMu        sync.RWMutex
	maxCacheAge    time.Duration
	cacheTimestamps map[string]time.Time
}

// パフォーマンスオプティマイザーを作成
func NewPerformanceOptimizer() *PerformanceOptimizer {
	return &PerformanceOptimizer{
		workerPool:       NewWorkerPool(runtime.NumCPU()),
		debouncer:        NewDebouncer(100*time.Millisecond, 500*time.Millisecond),
		memoryManager:    NewMemoryManager(),
		metricsCollector: NewMetricsCollector(),
		asyncProcessor:   NewAsyncProcessor(),
	}
}

// ワーカープールを作成
func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkerPool{
		workers:    workers,
		jobChannel: make(chan Job, workers*2), // バッファ付き
		ctx:        ctx,
		cancel:     cancel,
	}
}

// ワーカープールを開始
func (wp *WorkerPool) Start() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if wp.started {
		return
	}

	wp.started = true
	
	// ワーカーゴルーチンを起動
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// ワーカープールを停止
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if !wp.started {
		return
	}

	wp.cancel()
	close(wp.jobChannel)
	wp.wg.Wait()
	wp.started = false
}

// ジョブを投入
func (wp *WorkerPool) Submit(job Job) error {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	
	if !wp.started {
		return fmt.Errorf("worker pool not started")
	}

	select {
	case wp.jobChannel <- job:
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
		return fmt.Errorf("job queue full")
	}
}

// ワーカーのメイン処理
func (wp *WorkerPool) worker(_ int) {
	defer wp.wg.Done()
	
	for {
		select {
		case job, ok := <-wp.jobChannel:
			if !ok {
				return
			}
			
			// ジョブを実行
			start := time.Now()
			result := job.Task()
			elapsed := time.Since(start)
			
			// メトリクス記録（簡易実装）
			_ = elapsed
			
			// コールバック実行
			if job.Callback != nil {
				job.Callback(result)
			}
			
		case <-wp.ctx.Done():
			return
		}
	}
}

// デバウンサーを作成
func NewDebouncer(delay, maxDelay time.Duration) *Debouncer {
	return &Debouncer{
		timers:   make(map[string]*time.Timer),
		delay:    delay,
		maxDelay: maxDelay,
	}
}

// デバウンス処理
func (d *Debouncer) Debounce(key string, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// 既存のタイマーがあればキャンセル
	if timer, exists := d.timers[key]; exists {
		timer.Stop()
		delete(d.timers, key)
	}
	
	// 新しいタイマーを設定
	d.timers[key] = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		delete(d.timers, key)
		d.mu.Unlock()
		fn()
	})
}

// 即座に実行（デバウンス解除）
func (d *Debouncer) Flush(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if timer, exists := d.timers[key]; exists {
		timer.Stop()
		delete(d.timers, key)
	}
}

// メモリ管理システムを作成
func NewMemoryManager() *MemoryManager {
	return &MemoryManager{
		maxCacheSize:    100 * 1024 * 1024, // 100MB
		gcThreshold:     0.8,               // 80%でGC実行
		cleanupInterval: 5 * time.Minute,
		objectSizes:     make(map[string]int64),
		lruCache:        NewLRUCache(1000), // 1000エントリ
		lastCleanup:     time.Now(),
	}
}

// メモリ使用量チェック
func (mm *MemoryManager) CheckMemoryUsage() {
	mm.mu.RLock()
	usage := float64(mm.currentCacheSize) / float64(mm.maxCacheSize)
	mm.mu.RUnlock()
	
	if usage > mm.gcThreshold {
		mm.performCleanup()
	}
}

// クリーンアップ実行
func (mm *MemoryManager) performCleanup() {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	// LRUキャッシュから古いエントリを削除
	removed := mm.lruCache.removeOldest(int(float64(mm.lruCache.capacity) * 0.3))
	
	// 削除されたエントリのサイズを減算
	for _, key := range removed {
		if size, exists := mm.objectSizes[key]; exists {
			mm.currentCacheSize -= size
			delete(mm.objectSizes, key)
		}
	}
	
	mm.lastCleanup = time.Now()
	runtime.GC() // ガベージコレクション実行
}

// オブジェクトサイズを登録
func (mm *MemoryManager) RegisterObject(key string, size int64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	mm.objectSizes[key] = size
	mm.currentCacheSize += size
}

// オブジェクトサイズを削除
func (mm *MemoryManager) UnregisterObject(key string) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	if size, exists := mm.objectSizes[key]; exists {
		mm.currentCacheSize -= size
		delete(mm.objectSizes, key)
	}
}

// LRUキャッシュを作成
func NewLRUCache(capacity int) *LRUCache {
	cache := &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*Node),
	}
	
	// ダミーのヘッドとテール
	cache.head = &Node{}
	cache.tail = &Node{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head
	
	return cache
}

// LRUキャッシュから取得
func (lru *LRUCache) Get(key string) (interface{}, bool) {
	lru.mu.RLock()
	node, exists := lru.cache[key]
	lru.mu.RUnlock()
	
	if !exists {
		return nil, false
	}
	
	// アクセスされたノードを先頭に移動
	lru.mu.Lock()
	lru.moveToHead(node)
	lru.mu.Unlock()
	
	return node.value, true
}

// LRUキャッシュに設定
func (lru *LRUCache) Put(key string, value interface{}) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	
	if node, exists := lru.cache[key]; exists {
		// 既存のエントリを更新
		node.value = value
		lru.moveToHead(node)
		return
	}
	
	// 新しいエントリを追加
	newNode := &Node{
		key:   key,
		value: value,
	}
	
	lru.cache[key] = newNode
	lru.addToHead(newNode)
	
	// 容量超過チェック
	if len(lru.cache) > lru.capacity {
		tail := lru.removeTail()
		delete(lru.cache, tail.key)
	}
}

// ノードを先頭に移動
func (lru *LRUCache) moveToHead(node *Node) {
	lru.removeNode(node)
	lru.addToHead(node)
}

// ノードを先頭に追加
func (lru *LRUCache) addToHead(node *Node) {
	node.prev = lru.head
	node.next = lru.head.next
	lru.head.next.prev = node
	lru.head.next = node
}

// ノードを削除
func (lru *LRUCache) removeNode(node *Node) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

// 末尾ノードを削除
func (lru *LRUCache) removeTail() *Node {
	lastNode := lru.tail.prev
	lru.removeNode(lastNode)
	return lastNode
}

// 古いエントリを削除（指定数）
func (lru *LRUCache) removeOldest(count int) []string {
	var removed []string
	
	for i := 0; i < count && len(lru.cache) > 0; i++ {
		tail := lru.removeTail()
		delete(lru.cache, tail.key)
		removed = append(removed, tail.key)
	}
	
	return removed
}

// メトリクス収集システムを作成
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
	}
}

// リクエスト記録
func (mc *MetricsCollector) RecordRequest(latency time.Duration, isError bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.requestCount++
	mc.totalRequests++
	
	if isError {
		mc.errorCount++
	}
	
	// 平均レイテンシ更新（移動平均）
	if mc.averageLatency == 0 {
		mc.averageLatency = latency
	} else {
		mc.averageLatency = (mc.averageLatency*9 + latency) / 10
	}
	
	// ピークレイテンシ更新
	if latency > mc.peakLatency {
		mc.peakLatency = latency
	}
	
	// キャッシュヒット率を再計算
	if mc.totalRequests > 0 {
		mc.cacheHitRate = float64(mc.cacheHits) / float64(mc.totalRequests)
	}
}

// キャッシュヒット記録
func (mc *MetricsCollector) RecordCacheHit() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.cacheHits++
	mc.totalRequests++ // キャッシュヒットもリクエストとしてカウント
	mc.cacheHitRate = float64(mc.cacheHits) / float64(mc.totalRequests)
}

// メモリ使用量記録
func (mc *MetricsCollector) RecordMemoryUsage(usage int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.memoryUsage = usage
}

// メトリクス取得
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	uptime := time.Since(mc.startTime)
	
	return map[string]interface{}{
		"requests_total":      mc.requestCount,
		"errors_total":        mc.errorCount,
		"average_latency_ms":  mc.averageLatency.Milliseconds(),
		"peak_latency_ms":     mc.peakLatency.Milliseconds(),
		"cache_hit_rate":      mc.cacheHitRate,
		"memory_usage_bytes":  mc.memoryUsage,
		"uptime_seconds":      uptime.Seconds(),
		"requests_per_second": float64(mc.requestCount) / uptime.Seconds(),
	}
}

// 非同期処理システムを作成
func NewAsyncProcessor() *AsyncProcessor {
	return &AsyncProcessor{
		completionJobs:  make(chan Job, 100),
		resultCache:     make(map[string]interface{}),
		maxCacheAge:     10 * time.Minute,
		cacheTimestamps: make(map[string]time.Time),
	}
}

// 非同期ジョブを開始
func (ap *AsyncProcessor) StartAsync(key string, task func() interface{}) {
	go func() {
		result := task()
		
		ap.cacheMu.Lock()
		ap.resultCache[key] = result
		ap.cacheTimestamps[key] = time.Now()
		ap.cacheMu.Unlock()
		
		// 古いキャッシュエントリを削除
		ap.cleanupCache()
	}()
}

// 結果を取得（非同期）
func (ap *AsyncProcessor) GetResult(key string) (interface{}, bool) {
	ap.cacheMu.RLock()
	defer ap.cacheMu.RUnlock()
	
	result, exists := ap.resultCache[key]
	if !exists {
		return nil, false
	}
	
	// 有効期限チェック
	if timestamp, ok := ap.cacheTimestamps[key]; ok {
		if time.Since(timestamp) > ap.maxCacheAge {
			return nil, false
		}
	}
	
	return result, true
}

// キャッシュクリーンアップ
func (ap *AsyncProcessor) cleanupCache() {
	ap.cacheMu.Lock()
	defer ap.cacheMu.Unlock()
	
	now := time.Now()
	for key, timestamp := range ap.cacheTimestamps {
		if now.Sub(timestamp) > ap.maxCacheAge {
			delete(ap.resultCache, key)
			delete(ap.cacheTimestamps, key)
		}
	}
}

// パフォーマンス最適化されたオートコンプリート
func (po *PerformanceOptimizer) OptimizedCompletion(input string, completer *AdvancedCompleter) []CompletionCandidate {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		po.metricsCollector.RecordRequest(elapsed, false)
	}()

	// 非同期で結果があるかチェック
	if result, exists := po.asyncProcessor.GetResult("completion:" + input); exists {
		po.metricsCollector.RecordCacheHit()
		if candidates, ok := result.([]CompletionCandidate); ok {
			return candidates
		}
	}

	// デバウンス処理で連続入力を制御
	resultChan := make(chan []CompletionCandidate, 1)
	
	po.debouncer.Debounce("completion", func() {
		// ワーカープールで並列実行
		job := Job{
			ID: "completion:" + input,
			Task: func() interface{} {
				return completer.GetAdvancedSuggestions(input)
			},
			Callback: func(result interface{}) {
				if candidates, ok := result.([]CompletionCandidate); ok {
					// 非同期キャッシュに保存
					po.asyncProcessor.StartAsync("completion:"+input, func() interface{} {
						return candidates
					})
					
					select {
					case resultChan <- candidates:
					default:
					}
				}
			},
		}
		
		// ワーカープールが開始されていない場合は直接実行
		if err := po.workerPool.Submit(job); err != nil {
			candidates := completer.GetAdvancedSuggestions(input)
			select {
			case resultChan <- candidates:
			default:
			}
		}
	})

	// タイムアウト付きで結果を待機
	select {
	case candidates := <-resultChan:
		return candidates
	case <-time.After(500 * time.Millisecond):
		// タイムアウトの場合は空の結果を返す
		return []CompletionCandidate{}
	}
}

// パフォーマンスオプティマイザーを開始
func (po *PerformanceOptimizer) Start() {
	po.workerPool.Start()
	
	// 定期的なメモリクリーンアップ
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			po.memoryManager.CheckMemoryUsage()
		}
	}()
}

// パフォーマンスオプティマイザーを停止
func (po *PerformanceOptimizer) Stop() {
	po.workerPool.Stop()
}