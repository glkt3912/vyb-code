package performance

import (
	"sync"
	"time"
)

// パフォーマンス指標を収集する構造体
type Metrics struct {
	mu sync.RWMutex

	// LLM関連の指標
	LLMRequestCount    int64         `json:"llm_request_count"`
	LLMTotalDuration   time.Duration `json:"llm_total_duration"`
	LLMAverageDuration time.Duration `json:"llm_average_duration"`
	LLMErrorCount      int64         `json:"llm_error_count"`

	// ファイル操作の指標
	FileReadCount     int64         `json:"file_read_count"`
	FileWriteCount    int64         `json:"file_write_count"`
	FileTotalSize     int64         `json:"file_total_size"`
	FileOperationTime time.Duration `json:"file_operation_time"`

	// コマンド実行の指標
	CommandCount       int64         `json:"command_count"`
	CommandSuccessRate float64       `json:"command_success_rate"`
	CommandTotalTime   time.Duration `json:"command_total_time"`

	// メモリ使用量
	MemoryUsage     int64 `json:"memory_usage"`
	PeakMemoryUsage int64 `json:"peak_memory_usage"`
}

// グローバルメトリクスインスタンス
var globalMetrics = &Metrics{}

// グローバルメトリクスを取得
func GetMetrics() *Metrics {
	return globalMetrics
}

// LLMリクエストの記録
func (m *Metrics) RecordLLMRequest(duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.LLMRequestCount++
	m.LLMTotalDuration += duration

	if m.LLMRequestCount > 0 {
		m.LLMAverageDuration = m.LLMTotalDuration / time.Duration(m.LLMRequestCount)
	}

	if !success {
		m.LLMErrorCount++
	}
}

// ファイル操作の記録
func (m *Metrics) RecordFileOperation(operation string, size int64, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch operation {
	case "read":
		m.FileReadCount++
	case "write":
		m.FileWriteCount++
	}

	m.FileTotalSize += size
	m.FileOperationTime += duration
}

// コマンド実行の記録
func (m *Metrics) RecordCommandExecution(duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CommandCount++
	m.CommandTotalTime += duration

	// 成功率の計算
	successCount := m.CommandCount
	if !success {
		successCount--
	}
	m.CommandSuccessRate = float64(successCount) / float64(m.CommandCount) * 100.0
}

// メモリ使用量の更新
func (m *Metrics) UpdateMemoryUsage(current int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.MemoryUsage = current
	if current > m.PeakMemoryUsage {
		m.PeakMemoryUsage = current
	}
}

// メトリクスのリセット
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	*m = Metrics{}
}

// メトリクスのコピーを安全に取得
func (m *Metrics) Snapshot() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return *m
}
