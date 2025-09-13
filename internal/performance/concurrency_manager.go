package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// 並行処理設定
type ConcurrencyConfig struct {
	MaxWorkers        int           // 最大ワーカー数
	MaxQueueSize      int           // 最大キューサイズ
	WorkerTimeout     time.Duration // ワーカータイムアウト
	TaskTimeout       time.Duration // タスクタイムアウト
	RetryAttempts     int           // 再試行回数
	RetryDelay        time.Duration // 再試行間隔
	MonitoringEnabled bool          // 監視機能有効
	HealthCheckPeriod time.Duration // ヘルスチェック間隔
}

// デフォルト並行処理設定
func DefaultConcurrencyConfig() *ConcurrencyConfig {
	return &ConcurrencyConfig{
		MaxWorkers:        runtime.GOMAXPROCS(0) * 2, // CPU数の2倍
		MaxQueueSize:      1000,
		WorkerTimeout:     30 * time.Second,
		TaskTimeout:       10 * time.Second,
		RetryAttempts:     3,
		RetryDelay:        1 * time.Second,
		MonitoringEnabled: true,
		HealthCheckPeriod: 10 * time.Second,
	}
}

// タスク定義
type Task struct {
	ID          string                                         // タスクID
	Priority    TaskPriority                                   // 優先度
	Function    func(ctx context.Context) (interface{}, error) // 実行関数
	Context     context.Context                                // コンテキスト
	Retry       int                                            // 再試行回数
	CreatedAt   time.Time                                      // 作成時刻
	StartedAt   time.Time                                      // 開始時刻
	CompletedAt time.Time                                      // 完了時刻
	Result      interface{}                                    // 結果
	Error       error                                          // エラー
	Metadata    map[string]interface{}                         // メタデータ
}

// タスク優先度
type TaskPriority int

const (
	PriorityLow      TaskPriority = 1
	PriorityNormal   TaskPriority = 2
	PriorityHigh     TaskPriority = 3
	PriorityCritical TaskPriority = 4
)

// タスク状態
type TaskStatus int

const (
	StatusPending TaskStatus = iota
	StatusRunning
	StatusCompleted
	StatusFailed
	StatusCanceled
)

// ワーカー統計
type WorkerStats struct {
	WorkerID       int       `json:"worker_id"`
	TasksProcessed int64     `json:"tasks_processed"`
	TasksSucceeded int64     `json:"tasks_succeeded"`
	TasksFailed    int64     `json:"tasks_failed"`
	AvgProcessTime float64   `json:"avg_process_time"`
	LastActive     time.Time `json:"last_active"`
	Status         string    `json:"status"`
}

// 並行処理統計
type ConcurrencyStats struct {
	ActiveWorkers     int32          `json:"active_workers"`
	QueueSize         int            `json:"queue_size"`
	ProcessedTasks    int64          `json:"processed_tasks"`
	SuccessfulTasks   int64          `json:"successful_tasks"`
	FailedTasks       int64          `json:"failed_tasks"`
	AvgProcessingTime time.Duration  `json:"avg_processing_time"`
	WorkerStats       []*WorkerStats `json:"worker_stats"`
	LastUpdated       time.Time      `json:"last_updated"`
}

// 並行処理マネージャー
type ConcurrencyManager struct {
	config      *ConcurrencyConfig
	workers     []*Worker
	taskQueue   chan *Task
	stats       *ConcurrencyStats
	workerStats map[int]*WorkerStats
	mu          sync.RWMutex
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	isRunning   bool
}

// ワーカー
type Worker struct {
	ID           int
	manager      *ConcurrencyManager
	stats        *WorkerStats
	stopChan     chan bool
	taskCount    int64
	successCount int64
	failCount    int64
	totalTime    time.Duration
}

// 新しい並行処理マネージャーを作成
func NewConcurrencyManager(config *ConcurrencyConfig) *ConcurrencyManager {
	if config == nil {
		config = DefaultConcurrencyConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	cm := &ConcurrencyManager{
		config:      config,
		workers:     make([]*Worker, 0, config.MaxWorkers),
		taskQueue:   make(chan *Task, config.MaxQueueSize),
		stats:       &ConcurrencyStats{},
		workerStats: make(map[int]*WorkerStats),
		ctx:         ctx,
		cancel:      cancel,
	}

	return cm
}

// 並行処理を開始
func (cm *ConcurrencyManager) Start() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.isRunning {
		return fmt.Errorf("並行処理マネージャーは既に実行中です")
	}

	// ワーカーを起動
	for i := 0; i < cm.config.MaxWorkers; i++ {
		worker := &Worker{
			ID:       i,
			manager:  cm,
			stats:    &WorkerStats{WorkerID: i, Status: "starting"},
			stopChan: make(chan bool, 1),
		}
		cm.workers = append(cm.workers, worker)
		cm.workerStats[i] = worker.stats

		cm.wg.Add(1)
		go worker.run()
	}

	// 監視機能を開始
	if cm.config.MonitoringEnabled {
		cm.wg.Add(1)
		go cm.monitor()
	}

	cm.isRunning = true
	return nil
}

// 並行処理を停止
func (cm *ConcurrencyManager) Stop() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.isRunning {
		return fmt.Errorf("並行処理マネージャーは実行されていません")
	}

	// コンテキストをキャンセル
	cm.cancel()

	// 全ワーカーに停止信号を送信
	for _, worker := range cm.workers {
		select {
		case worker.stopChan <- true:
		default:
		}
	}

	// ワーカーの停止を待機
	cm.wg.Wait()

	// リソースをクリーンアップ
	close(cm.taskQueue)
	cm.workers = nil
	cm.isRunning = false

	return nil
}

// タスクを提出
func (cm *ConcurrencyManager) SubmitTask(task *Task) error {
	if !cm.isRunning {
		return fmt.Errorf("並行処理マネージャーが実行されていません")
	}

	if task == nil {
		return fmt.Errorf("タスクがnilです")
	}

	if task.Context == nil {
		task.Context = context.Background()
	}

	task.CreatedAt = time.Now()

	select {
	case cm.taskQueue <- task:
		return nil
	case <-cm.ctx.Done():
		return fmt.Errorf("並行処理マネージャーが停止されました")
	default:
		return fmt.Errorf("タスクキューが満杯です")
	}
}

// 複数のタスクを並列実行
func (cm *ConcurrencyManager) ExecuteParallel(tasks []*Task) ([]*Task, error) {
	if len(tasks) == 0 {
		return tasks, nil
	}

	results := make([]*Task, len(tasks))
	resultChan := make(chan struct {
		index int
		task  *Task
	}, len(tasks))

	// 全タスクを提出
	for i, task := range tasks {
		if err := cm.SubmitTask(task); err != nil {
			return nil, fmt.Errorf("タスク%dの提出に失敗: %w", i, err)
		}

		// 結果を待機するgoroutineを起動
		go func(index int, t *Task) {
			// タスクの完了を待機
			for {
				if t.CompletedAt.IsZero() && t.Error == nil {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				resultChan <- struct {
					index int
					task  *Task
				}{index, t}
				break
			}
		}(i, task)
	}

	// 全結果の収集
	completed := 0
	timeout := time.After(cm.config.TaskTimeout * time.Duration(len(tasks)))

	for completed < len(tasks) {
		select {
		case result := <-resultChan:
			results[result.index] = result.task
			completed++
		case <-timeout:
			return results, fmt.Errorf("タスク実行がタイムアウトしました")
		case <-cm.ctx.Done():
			return results, fmt.Errorf("並行処理マネージャーが停止されました")
		}
	}

	return results, nil
}

// ワーカーの実行ループ
func (w *Worker) run() {
	defer w.manager.wg.Done()

	w.stats.Status = "running"
	w.stats.LastActive = time.Now()

	for {
		select {
		case task := <-w.manager.taskQueue:
			w.processTask(task)
		case <-w.stopChan:
			w.stats.Status = "stopped"
			return
		case <-w.manager.ctx.Done():
			w.stats.Status = "stopped"
			return
		}
	}
}

// タスクを処理
func (w *Worker) processTask(task *Task) {
	startTime := time.Now()
	task.StartedAt = startTime

	atomic.AddInt32(&w.manager.stats.ActiveWorkers, 1)
	defer atomic.AddInt32(&w.manager.stats.ActiveWorkers, -1)

	// タスクタイムアウト設定
	ctx, cancel := context.WithTimeout(task.Context, w.manager.config.TaskTimeout)
	defer cancel()

	// タスク実行
	result, err := task.Function(ctx)

	endTime := time.Now()
	processingTime := endTime.Sub(startTime)

	// 結果を設定
	task.Result = result
	task.Error = err
	task.CompletedAt = endTime

	// 統計を更新
	w.updateStats(processingTime, err == nil)

	// 失敗時の再試行
	if err != nil && task.Retry < w.manager.config.RetryAttempts {
		task.Retry++
		time.Sleep(w.manager.config.RetryDelay)

		// 再試行としてキューに戻す
		select {
		case w.manager.taskQueue <- task:
		default:
			// キューが満杯の場合は諦める
		}
		return
	}

	// 全体統計を更新
	atomic.AddInt64(&w.manager.stats.ProcessedTasks, 1)
	if err == nil {
		atomic.AddInt64(&w.manager.stats.SuccessfulTasks, 1)
	} else {
		atomic.AddInt64(&w.manager.stats.FailedTasks, 1)
	}
}

// ワーカー統計を更新
func (w *Worker) updateStats(processingTime time.Duration, success bool) {
	w.manager.mu.Lock()
	defer w.manager.mu.Unlock()

	atomic.AddInt64(&w.taskCount, 1)
	w.totalTime += processingTime

	if success {
		atomic.AddInt64(&w.successCount, 1)
	} else {
		atomic.AddInt64(&w.failCount, 1)
	}

	w.stats.TasksProcessed = atomic.LoadInt64(&w.taskCount)
	w.stats.TasksSucceeded = atomic.LoadInt64(&w.successCount)
	w.stats.TasksFailed = atomic.LoadInt64(&w.failCount)
	w.stats.LastActive = time.Now()

	if w.stats.TasksProcessed > 0 {
		w.stats.AvgProcessTime = float64(w.totalTime.Nanoseconds()) / float64(w.stats.TasksProcessed) / 1e6 // ミリ秒
	}
}

// 監視ループ
func (cm *ConcurrencyManager) monitor() {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.config.HealthCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.updateStats()
		case <-cm.ctx.Done():
			return
		}
	}
}

// 統計を更新
func (cm *ConcurrencyManager) updateStats() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.stats.QueueSize = len(cm.taskQueue)
	cm.stats.LastUpdated = time.Now()

	// ワーカー統計をコピー
	workerStats := make([]*WorkerStats, 0, len(cm.workerStats))
	for _, stats := range cm.workerStats {
		statsCopy := *stats
		workerStats = append(workerStats, &statsCopy)
	}
	cm.stats.WorkerStats = workerStats

	// 平均処理時間を計算
	totalTime := int64(0)
	validWorkers := 0
	for _, stats := range cm.workerStats {
		if stats.TasksProcessed > 0 {
			totalTime += int64(stats.AvgProcessTime * float64(stats.TasksProcessed))
			validWorkers++
		}
	}

	if cm.stats.ProcessedTasks > 0 {
		cm.stats.AvgProcessingTime = time.Duration(totalTime / cm.stats.ProcessedTasks * 1e6) // ナノ秒に変換
	}
}

// 現在の統計を取得
func (cm *ConcurrencyManager) GetStats() *ConcurrencyStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 統計のコピーを返す
	stats := *cm.stats
	stats.ActiveWorkers = atomic.LoadInt32(&cm.stats.ActiveWorkers)
	stats.ProcessedTasks = atomic.LoadInt64(&cm.stats.ProcessedTasks)
	stats.SuccessfulTasks = atomic.LoadInt64(&cm.stats.SuccessfulTasks)
	stats.FailedTasks = atomic.LoadInt64(&cm.stats.FailedTasks)
	stats.QueueSize = len(cm.taskQueue)

	return &stats
}

// ヘルスチェック
func (cm *ConcurrencyManager) HealthCheck() map[string]interface{} {
	stats := cm.GetStats()

	health := map[string]interface{}{
		"status":           "healthy",
		"active_workers":   stats.ActiveWorkers,
		"queue_size":       stats.QueueSize,
		"queue_capacity":   cm.config.MaxQueueSize,
		"processed_tasks":  stats.ProcessedTasks,
		"success_rate":     float64(stats.SuccessfulTasks) / float64(stats.ProcessedTasks) * 100,
		"avg_process_time": stats.AvgProcessingTime.String(),
		"is_running":       cm.isRunning,
		"last_updated":     stats.LastUpdated,
	}

	// ヘルス状態を判定
	if stats.QueueSize > cm.config.MaxQueueSize*8/10 {
		health["status"] = "degraded"
		health["warning"] = "キューサイズが容量の80%を超えています"
	}

	if stats.ActiveWorkers < int32(cm.config.MaxWorkers)/2 {
		health["status"] = "degraded"
		health["warning"] = "アクティブワーカー数が少なすぎます"
	}

	if stats.ProcessedTasks > 0 {
		successRate := float64(stats.SuccessfulTasks) / float64(stats.ProcessedTasks)
		if successRate < 0.9 {
			health["status"] = "unhealthy"
			health["error"] = "成功率が90%を下回っています"
		}
	}

	return health
}

// 設定を動的に更新
func (cm *ConcurrencyManager) UpdateConfig(config *ConcurrencyConfig) error {
	if !cm.isRunning {
		cm.config = config
		return nil
	}

	// 実行中の設定変更は一部のみ許可
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if config.TaskTimeout != cm.config.TaskTimeout {
		cm.config.TaskTimeout = config.TaskTimeout
	}

	if config.RetryAttempts != cm.config.RetryAttempts {
		cm.config.RetryAttempts = config.RetryAttempts
	}

	if config.RetryDelay != cm.config.RetryDelay {
		cm.config.RetryDelay = config.RetryDelay
	}

	return nil
}

// キューをクリア
func (cm *ConcurrencyManager) ClearQueue() int {
	clearedCount := 0

	for {
		select {
		case <-cm.taskQueue:
			clearedCount++
		default:
			return clearedCount
		}
	}
}
