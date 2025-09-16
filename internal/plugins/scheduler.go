package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/glkt/vyb-code/internal/logger"
)

// PluginScheduler はプラグインの実行スケジュールを管理
type PluginScheduler struct {
	logger    logger.Logger
	tasks     map[string]*ScheduledTask
	tasksMu   sync.RWMutex
	ticker    *time.Ticker
	stopChan  chan struct{}
	running   bool
	runningMu sync.RWMutex
}

// ScheduledTask はスケジュールされたタスクの情報
type ScheduledTask struct {
	Name       string                          `json:"name"`
	Interval   time.Duration                   `json:"interval"`
	NextRun    time.Time                       `json:"next_run"`
	LastRun    time.Time                       `json:"last_run"`
	Function   func(ctx context.Context) error `json:"-"`
	RunCount   int                             `json:"run_count"`
	ErrorCount int                             `json:"error_count"`
	LastError  string                          `json:"last_error,omitempty"`
	Enabled    bool                            `json:"enabled"`
	OneTime    bool                            `json:"one_time"`
}

// NewPluginScheduler は新しいスケジューラーを作成
func NewPluginScheduler(logger logger.Logger) *PluginScheduler {
	return &PluginScheduler{
		logger:   logger,
		tasks:    make(map[string]*ScheduledTask),
		stopChan: make(chan struct{}),
	}
}

// Start はスケジューラーを開始
func (s *PluginScheduler) Start(ctx context.Context) {
	s.runningMu.Lock()
	defer s.runningMu.Unlock()

	if s.running {
		return
	}

	s.running = true
	s.ticker = time.NewTicker(1 * time.Second) // 1秒間隔でチェック

	go s.run(ctx)

	s.logger.Info("プラグインスケジューラー開始", nil)
}

// Stop はスケジューラーを停止
func (s *PluginScheduler) Stop(ctx context.Context) {
	s.runningMu.Lock()
	defer s.runningMu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.stopChan)

	if s.ticker != nil {
		s.ticker.Stop()
	}

	s.logger.Info("プラグインスケジューラー停止", nil)
}

// run はスケジューラーのメインループ
func (s *PluginScheduler) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-s.ticker.C:
			s.processTasks(ctx)
		}
	}
}

// processTasks は実行すべきタスクを処理
func (s *PluginScheduler) processTasks(ctx context.Context) {
	now := time.Now()

	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()

	for _, task := range s.tasks {
		if !task.Enabled {
			continue
		}

		if now.Before(task.NextRun) {
			continue
		}

		// タスクを実行
		go s.executeTask(ctx, task)
	}
}

// executeTask は個別のタスクを実行
func (s *PluginScheduler) executeTask(ctx context.Context, task *ScheduledTask) {
	defer func() {
		if r := recover(); r != nil {
			task.ErrorCount++
			task.LastError = fmt.Sprintf("panic: %v", r)
			s.logger.Error("スケジュールタスクパニック", map[string]interface{}{
				"task":  task.Name,
				"error": task.LastError,
			})
		}
	}()

	s.logger.Debug("スケジュールタスク実行開始", map[string]interface{}{
		"task": task.Name,
	})

	startTime := time.Now()
	err := task.Function(ctx)
	duration := time.Since(startTime)

	// タスク実行結果を更新
	s.tasksMu.Lock()
	task.LastRun = startTime
	task.RunCount++

	if err != nil {
		task.ErrorCount++
		task.LastError = err.Error()
		s.logger.Warn("スケジュールタスクエラー", map[string]interface{}{
			"task":     task.Name,
			"error":    err.Error(),
			"duration": duration,
		})
	} else {
		task.LastError = ""
		s.logger.Debug("スケジュールタスク完了", map[string]interface{}{
			"task":     task.Name,
			"duration": duration,
		})
	}

	// 次回実行時刻を設定
	if task.OneTime {
		task.Enabled = false
	} else {
		task.NextRun = time.Now().Add(task.Interval)
	}

	s.tasksMu.Unlock()
}

// ScheduleRepeating は繰り返し実行タスクをスケジュール
func (s *PluginScheduler) ScheduleRepeating(name string, interval time.Duration, fn func(ctx context.Context) error) error {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()

	if _, exists := s.tasks[name]; exists {
		return fmt.Errorf("タスク %s は既に存在します", name)
	}

	task := &ScheduledTask{
		Name:     name,
		Interval: interval,
		NextRun:  time.Now().Add(interval),
		Function: fn,
		Enabled:  true,
		OneTime:  false,
	}

	s.tasks[name] = task

	s.logger.Info("繰り返しタスクスケジュール", map[string]interface{}{
		"task":     name,
		"interval": interval,
		"next_run": task.NextRun,
	})

	return nil
}

// ScheduleOnce は一回限りのタスクをスケジュール
func (s *PluginScheduler) ScheduleOnce(name string, delay time.Duration, fn func(ctx context.Context) error) error {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()

	if _, exists := s.tasks[name]; exists {
		return fmt.Errorf("タスク %s は既に存在します", name)
	}

	task := &ScheduledTask{
		Name:     name,
		NextRun:  time.Now().Add(delay),
		Function: fn,
		Enabled:  true,
		OneTime:  true,
	}

	s.tasks[name] = task

	s.logger.Info("一回限りタスクスケジュール", map[string]interface{}{
		"task":     name,
		"delay":    delay,
		"run_time": task.NextRun,
	})

	return nil
}

// RemoveTask はタスクを削除
func (s *PluginScheduler) RemoveTask(name string) error {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()

	if _, exists := s.tasks[name]; !exists {
		return fmt.Errorf("タスク %s が見つかりません", name)
	}

	delete(s.tasks, name)

	s.logger.Info("タスク削除", map[string]interface{}{
		"task": name,
	})

	return nil
}

// EnableTask はタスクを有効化
func (s *PluginScheduler) EnableTask(name string) error {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()

	task, exists := s.tasks[name]
	if !exists {
		return fmt.Errorf("タスク %s が見つかりません", name)
	}

	task.Enabled = true

	// 次回実行時刻を再設定（一回限りでない場合）
	if !task.OneTime {
		task.NextRun = time.Now().Add(task.Interval)
	}

	s.logger.Info("タスク有効化", map[string]interface{}{
		"task": name,
	})

	return nil
}

// DisableTask はタスクを無効化
func (s *PluginScheduler) DisableTask(name string) error {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()

	task, exists := s.tasks[name]
	if !exists {
		return fmt.Errorf("タスク %s が見つかりません", name)
	}

	task.Enabled = false

	s.logger.Info("タスク無効化", map[string]interface{}{
		"task": name,
	})

	return nil
}

// GetTask はタスク情報を取得
func (s *PluginScheduler) GetTask(name string) (*ScheduledTask, error) {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()

	task, exists := s.tasks[name]
	if !exists {
		return nil, fmt.Errorf("タスク %s が見つかりません", name)
	}

	// コピーを返す（Function は除く）
	taskCopy := *task
	taskCopy.Function = nil
	return &taskCopy, nil
}

// ListTasks は全タスクの一覧を取得
func (s *PluginScheduler) ListTasks() map[string]*ScheduledTask {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()

	result := make(map[string]*ScheduledTask)
	for name, task := range s.tasks {
		taskCopy := *task
		taskCopy.Function = nil
		result[name] = &taskCopy
	}

	return result
}

// GetStats はスケジューラーの統計情報を取得
func (s *PluginScheduler) GetStats() SchedulerStats {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()

	stats := SchedulerStats{
		TotalTasks:   len(s.tasks),
		EnabledTasks: 0,
		TotalRuns:    0,
		TotalErrors:  0,
	}

	for _, task := range s.tasks {
		if task.Enabled {
			stats.EnabledTasks++
		}
		stats.TotalRuns += task.RunCount
		stats.TotalErrors += task.ErrorCount
	}

	s.runningMu.RLock()
	stats.Running = s.running
	s.runningMu.RUnlock()

	return stats
}

// SchedulerStats はスケジューラーの統計情報
type SchedulerStats struct {
	Running      bool `json:"running"`
	TotalTasks   int  `json:"total_tasks"`
	EnabledTasks int  `json:"enabled_tasks"`
	TotalRuns    int  `json:"total_runs"`
	TotalErrors  int  `json:"total_errors"`
}

// IsRunning はスケジューラーが実行中かチェック
func (s *PluginScheduler) IsRunning() bool {
	s.runningMu.RLock()
	defer s.runningMu.RUnlock()
	return s.running
}
