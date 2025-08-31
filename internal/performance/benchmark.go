package performance

import (
	"context"
	"runtime"
	"time"
)

// ベンチマーク結果を格納する構造体
type BenchmarkResult struct {
	Name           string        `json:"name"`
	Duration       time.Duration `json:"duration"`
	OperationsPerSec float64     `json:"operations_per_sec"`
	MemoryAllocated int64        `json:"memory_allocated"`
	Success        bool          `json:"success"`
	Error          string        `json:"error,omitempty"`
}

// ベンチマーク実行器
type Benchmarker struct {
	results []BenchmarkResult
}

// ベンチマーカーのコンストラクタ
func NewBenchmarker() *Benchmarker {
	return &Benchmarker{
		results: make([]BenchmarkResult, 0),
	}
}

// 関数のベンチマーク実行
func (b *Benchmarker) BenchmarkFunction(name string, iterations int, fn func() error) BenchmarkResult {
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	startTime := time.Now()
	var lastError error
	successCount := 0
	
	for i := 0; i < iterations; i++ {
		if err := fn(); err != nil {
			lastError = err
		} else {
			successCount++
		}
	}
	
	duration := time.Since(startTime)
	
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	result := BenchmarkResult{
		Name:             name,
		Duration:         duration,
		OperationsPerSec: float64(iterations) / duration.Seconds(),
		MemoryAllocated:  int64(memAfter.TotalAlloc - memBefore.TotalAlloc),
		Success:          successCount == iterations,
	}
	
	if lastError != nil {
		result.Error = lastError.Error()
	}
	
	b.results = append(b.results, result)
	return result
}

// LLMレスポンス時間のベンチマーク
func (b *Benchmarker) BenchmarkLLMResponse(ctx context.Context, llmFunc func(context.Context) error) BenchmarkResult {
	return b.BenchmarkFunction("LLM Response", 1, func() error {
		return llmFunc(ctx)
	})
}

// ファイル操作のベンチマーク
func (b *Benchmarker) BenchmarkFileOperations(readFunc, writeFunc func() error) []BenchmarkResult {
	results := make([]BenchmarkResult, 0, 2)
	
	// ファイル読み込みベンチマーク
	readResult := b.BenchmarkFunction("File Read", 10, readFunc)
	results = append(results, readResult)
	
	// ファイル書き込みベンチマーク
	writeResult := b.BenchmarkFunction("File Write", 10, writeFunc)
	results = append(results, writeResult)
	
	return results
}

// すべてのベンチマーク結果を取得
func (b *Benchmarker) GetResults() []BenchmarkResult {
	return b.results
}

// ベンチマーク結果をクリア
func (b *Benchmarker) ClearResults() {
	b.results = b.results[:0]
}

// パフォーマンス分析レポートの生成
func (b *Benchmarker) GenerateReport() map[string]interface{} {
	if len(b.results) == 0 {
		return map[string]interface{}{
			"message": "ベンチマーク結果がありません",
		}
	}
	
	// 統計情報の計算
	totalDuration := time.Duration(0)
	totalMemory := int64(0)
	successCount := 0
	
	for _, result := range b.results {
		totalDuration += result.Duration
		totalMemory += result.MemoryAllocated
		if result.Success {
			successCount++
		}
	}
	
	avgDuration := totalDuration / time.Duration(len(b.results))
	avgMemory := totalMemory / int64(len(b.results))
	successRate := float64(successCount) / float64(len(b.results)) * 100.0
	
	return map[string]interface{}{
		"total_benchmarks":   len(b.results),
		"average_duration":   avgDuration.String(),
		"total_memory":       totalMemory,
		"average_memory":     avgMemory,
		"success_rate":       successRate,
		"detailed_results":   b.results,
		"system_info":        GetSystemInfo(),
	}
}

// 軽量なパフォーマンス計測
func MeasureExecution(name string, fn func()) time.Duration {
	start := time.Now()
	fn()
	duration := time.Since(start)
	
	// グローバルメトリクスに記録
	GetMetrics().RecordCommandExecution(duration, true)
	
	return duration
}