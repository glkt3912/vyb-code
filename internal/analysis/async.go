package analysis

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// 非同期分析エンジン
type AsyncAnalyzer struct {
	workerPool  *WorkerPool
	resultCache *AnalysisCache
	timeout     time.Duration
	config      *AnalysisConfig
}

// ワーカープール
type WorkerPool struct {
	workers     int
	taskQueue   chan *AnalysisTask
	resultQueue chan *AnalysisResult
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// 分析タスク
type AnalysisTask struct {
	ID          string
	ProjectPath string
	Type        AnalysisType
	Priority    TaskPriority
	Timeout     time.Duration
	StartedAt   time.Time
}

// 分析結果
type AnalysisResult struct {
	TaskID      string
	Analysis    *ProjectAnalysis
	Error       error
	Duration    time.Duration
	CompletedAt time.Time
	Cached      bool
}

// タスク優先度
type TaskPriority int

const (
	TaskPriorityLow TaskPriority = iota
	TaskPriorityNormal
	TaskPriorityHigh
	TaskPriorityUrgent
)

// 新しい非同期分析器を作成
func NewAsyncAnalyzer(config *AnalysisConfig) *AsyncAnalyzer {
	if config == nil {
		config = DefaultAnalysisConfig()
	}

	// CPUコア数に基づいてワーカー数を決定（最大4、最小2）
	workerCount := runtime.NumCPU()
	if workerCount > 4 {
		workerCount = 4
	} else if workerCount < 2 {
		workerCount = 2
	}

	ctx, cancel := context.WithCancel(context.Background())

	workerPool := &WorkerPool{
		workers:     workerCount,
		taskQueue:   make(chan *AnalysisTask, 10),
		resultQueue: make(chan *AnalysisResult, 10),
		ctx:         ctx,
		cancel:      cancel,
	}

	analyzer := &AsyncAnalyzer{
		workerPool:  workerPool,
		resultCache: NewAnalysisCache(),
		timeout:     5 * time.Second, // デフォルト5秒タイムアウト
		config:      config,
	}

	// ワーカープールを開始
	analyzer.startWorkerPool()

	return analyzer
}

// ワーカープールを開始
func (aa *AsyncAnalyzer) startWorkerPool() {
	for i := 0; i < aa.workerPool.workers; i++ {
		aa.workerPool.wg.Add(1)
		go aa.worker(i)
	}

	// 結果処理ゴルーチン
	go aa.resultProcessor()
}

// 非同期で分析を実行
func (aa *AsyncAnalyzer) AnalyzeAsync(projectPath string, analysisType AnalysisType) <-chan *AnalysisResult {
	resultChan := make(chan *AnalysisResult, 1)

	task := &AnalysisTask{
		ID:          generateTaskID(),
		ProjectPath: projectPath,
		Type:        analysisType,
		Priority:    TaskPriorityNormal,
		Timeout:     aa.timeout,
		StartedAt:   time.Now(),
	}

	go func() {
		defer close(resultChan)

		// キャッシュチェック
		if aa.config.EnableCaching {
			if cachedResult := aa.resultCache.Get(projectPath, analysisType); cachedResult != nil {
				resultChan <- &AnalysisResult{
					TaskID:      task.ID,
					Analysis:    cachedResult,
					Duration:    0,
					CompletedAt: time.Now(),
					Cached:      true,
				}
				return
			}
		}

		// タスクをキューに送信
		select {
		case aa.workerPool.taskQueue <- task:
			// タスクが送信された
		case <-time.After(1 * time.Second):
			resultChan <- &AnalysisResult{
				TaskID: task.ID,
				Error:  fmt.Errorf("タスクキューがフル: 分析要求がタイムアウトしました"),
			}
			return
		}

		// 結果を待機（タイムアウト付き）
		select {
		case result := <-aa.workerPool.resultQueue:
			if result.TaskID == task.ID {
				resultChan <- result
			}
		case <-time.After(task.Timeout):
			resultChan <- &AnalysisResult{
				TaskID: task.ID,
				Error:  fmt.Errorf("分析タイムアウト: %v", task.Timeout),
			}
		}
	}()

	return resultChan
}

// 軽量分析（基本情報のみ）
func (aa *AsyncAnalyzer) AnalyzeLightweight(projectPath string) <-chan *AnalysisResult {
	return aa.AnalyzeAsync(projectPath, AnalysisTypeBasic)
}

// 標準分析（セキュリティ含む）
func (aa *AsyncAnalyzer) AnalyzeStandard(projectPath string) <-chan *AnalysisResult {
	return aa.AnalyzeAsync(projectPath, AnalysisTypeSecurity)
}

// フル分析（全機能）
func (aa *AsyncAnalyzer) AnalyzeFull(projectPath string) <-chan *AnalysisResult {
	return aa.AnalyzeAsync(projectPath, AnalysisTypeFull)
}

// ワーカー
func (aa *AsyncAnalyzer) worker(id int) {
	defer aa.workerPool.wg.Done()

	for {
		select {
		case <-aa.workerPool.ctx.Done():
			return
		case task := <-aa.workerPool.taskQueue:
			if task == nil {
				return
			}

			result := aa.executeTask(task)

			// 結果をキューに送信
			select {
			case aa.workerPool.resultQueue <- result:
			case <-time.After(1 * time.Second):
				// 結果キューがフルの場合は破棄
			}
		}
	}
}

// タスクを実行
func (aa *AsyncAnalyzer) executeTask(task *AnalysisTask) *AnalysisResult {
	startTime := time.Now()

	// タスクタイプに応じた分析を実行
	var analysis *ProjectAnalysis
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
	defer cancel()

	switch task.Type {
	case AnalysisTypeBasic:
		analysis, err = aa.executeBasicAnalysis(ctx, task.ProjectPath)
	case AnalysisTypeSecurity:
		analysis, err = aa.executeSecurityAnalysis(ctx, task.ProjectPath)
	case AnalysisTypeQuality:
		analysis, err = aa.executeQualityAnalysis(ctx, task.ProjectPath)
	case AnalysisTypeFull:
		analysis, err = aa.executeFullAnalysis(ctx, task.ProjectPath)
	default:
		err = fmt.Errorf("未知の分析タイプ: %v", task.Type)
	}

	duration := time.Since(startTime)

	// 成功した場合はキャッシュに保存
	if err == nil && analysis != nil && aa.config.EnableCaching {
		aa.resultCache.Set(task.ProjectPath, task.Type, analysis)
	}

	return &AnalysisResult{
		TaskID:      task.ID,
		Analysis:    analysis,
		Error:       err,
		Duration:    duration,
		CompletedAt: time.Now(),
		Cached:      false,
	}
}

// 基本分析（軽量）
func (aa *AsyncAnalyzer) executeBasicAnalysis(ctx context.Context, projectPath string) (*ProjectAnalysis, error) {
	analyzer := NewProjectAnalyzer(aa.config).(*projectAnalyzer)

	analysis := &ProjectAnalysis{
		ProjectPath:     projectPath,
		ProjectName:     filepath.Base(projectPath),
		AnalyzedAt:      time.Now(),
		AnalysisVersion: "1.0.0",
		Metadata:        make(map[string]interface{}),
	}

	// 基本分析を実行（軽量版）
	if err := analyzer.analyzeLanguageAndFramework(analysis); err != nil {
		return nil, err
	}

	// 軽量ファイル構造分析（最小限）
	if err := aa.analyzeBasicFileStructure(analysis); err != nil {
		return nil, err
	}

	return analysis, nil
}

// セキュリティ分析
func (aa *AsyncAnalyzer) executeSecurityAnalysis(ctx context.Context, projectPath string) (*ProjectAnalysis, error) {
	// 基本分析をベースに開始
	analysis, err := aa.executeBasicAnalysis(ctx, projectPath)
	if err != nil {
		return nil, err
	}

	analyzer := NewProjectAnalyzer(aa.config).(*projectAnalyzer)

	// セキュリティスキャン（タイムアウト付き）
	securityChan := make(chan []SecurityIssue, 1)
	go func() {
		if issues, err := analyzer.AnalyzeSecurity(projectPath); err == nil {
			securityChan <- issues
		} else {
			securityChan <- []SecurityIssue{}
		}
	}()

	select {
	case issues := <-securityChan:
		analysis.SecurityIssues = issues
	case <-ctx.Done():
		return analysis, fmt.Errorf("セキュリティ分析タイムアウト")
	}

	return analysis, nil
}

// 品質分析
func (aa *AsyncAnalyzer) executeQualityAnalysis(ctx context.Context, projectPath string) (*ProjectAnalysis, error) {
	analysis, err := aa.executeSecurityAnalysis(ctx, projectPath)
	if err != nil {
		return nil, err
	}

	analyzer := NewProjectAnalyzer(aa.config).(*projectAnalyzer)

	// 品質メトリクス（タイムアウト付き）
	qualityChan := make(chan *QualityMetrics, 1)
	go func() {
		if metrics, err := analyzer.AnalyzeQuality(projectPath); err == nil {
			qualityChan <- metrics
		} else {
			qualityChan <- nil
		}
	}()

	select {
	case metrics := <-qualityChan:
		analysis.QualityMetrics = metrics
	case <-ctx.Done():
		// タイムアウトしても部分的な結果を返す
	}

	return analysis, nil
}

// フル分析
func (aa *AsyncAnalyzer) executeFullAnalysis(ctx context.Context, projectPath string) (*ProjectAnalysis, error) {
	analysis, err := aa.executeQualityAnalysis(ctx, projectPath)
	if err != nil {
		return nil, err
	}

	analyzer := NewProjectAnalyzer(aa.config).(*projectAnalyzer)

	// 依存関係分析（タイムアウト付き）
	depsChan := make(chan []Dependency, 1)
	go func() {
		if deps, err := analyzer.AnalyzeDependencies(projectPath); err == nil {
			depsChan <- deps
		} else {
			depsChan <- []Dependency{}
		}
	}()

	select {
	case deps := <-depsChan:
		analysis.Dependencies = deps
	case <-ctx.Done():
		// タイムアウトしても部分的な結果を返す
	}

	// 推奨事項生成
	if recommendations, err := analyzer.GenerateRecommendations(analysis); err == nil {
		analysis.Recommendations = recommendations
	}

	return analysis, nil
}

// 結果プロセッサー（将来の拡張用）
func (aa *AsyncAnalyzer) resultProcessor() {
	// 現在は何もしないが、将来的にはロギングや統計収集に使用
	for {
		select {
		case <-aa.workerPool.ctx.Done():
			return
		case <-time.After(1 * time.Second):
			// 定期的な処理（統計更新等）
		}
	}
}

// リソースをクリーンアップ
func (aa *AsyncAnalyzer) Close() error {
	aa.workerPool.cancel()

	// タスクキューを閉じる
	close(aa.workerPool.taskQueue)

	// ワーカーの終了を待機
	aa.workerPool.wg.Wait()

	// 結果キューを閉じる
	close(aa.workerPool.resultQueue)

	return nil
}

// 軽量ファイル構造分析
func (aa *AsyncAnalyzer) analyzeBasicFileStructure(analysis *ProjectAnalysis) error {
	projectPath := analysis.ProjectPath
	structure := &FileStructure{
		Languages:   make(map[string]int),
		Directories: make([]DirectoryInfo, 0),
	}

	// ファイル数と行数の基本カウント
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)

		// 除外パターンをチェック
		for _, pattern := range aa.config.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				return filepath.SkipDir
			}
		}

		if !info.IsDir() {
			structure.TotalFiles++
			// 拡張子カウント
			ext := strings.ToLower(filepath.Ext(path))
			if ext != "" {
				structure.Languages[ext]++
			}
		}

		return nil
	})

	analysis.FileStructure = structure
	return err
}

// タスクIDを生成
func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

// タイムアウトを設定
func (aa *AsyncAnalyzer) SetTimeout(timeout time.Duration) {
	aa.timeout = timeout
}

// 統計情報を取得
func (aa *AsyncAnalyzer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"workers":      aa.workerPool.workers,
		"cache_size":   aa.resultCache.Size(),
		"timeout":      aa.timeout,
		"queue_length": len(aa.workerPool.taskQueue),
	}
}
