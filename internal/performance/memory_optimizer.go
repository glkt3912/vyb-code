package performance

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"
)

// メモリ最適化設定
type MemoryConfig struct {
	MaxSessionSize       int64         // セッション最大サイズ（バイト）
	MaxTurnsPerSession   int           // セッション当たり最大ターン数
	CompressionThreshold int64         // 圧縮閾値（バイト）
	CompressionRatio     float64       // 目標圧縮率
	CleanupInterval      time.Duration // クリーンアップ間隔
	RetentionPeriod      time.Duration // データ保持期間
	MemoryCheckInterval  time.Duration // メモリチェック間隔
	MaxMemoryUsage       int64         // 最大メモリ使用量（バイト）
}

// デフォルトメモリ設定
func DefaultMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		MaxSessionSize:       5 * 1024 * 1024,    // 5MB
		MaxTurnsPerSession:   1000,               // 1000ターン
		CompressionThreshold: 1024 * 1024,        // 1MB
		CompressionRatio:     0.3,                // 70%圧縮
		CleanupInterval:      30 * time.Minute,   // 30分
		RetentionPeriod:      7 * 24 * time.Hour, // 7日
		MemoryCheckInterval:  5 * time.Minute,    // 5分
		MaxMemoryUsage:       100 * 1024 * 1024,  // 100MB
	}
}

// メモリ最適化マネージャー
type MemoryOptimizer struct {
	config            *MemoryConfig
	mu                sync.RWMutex
	compressionCache  map[string][]byte    // 圧縮データキャッシュ
	accessTracker     map[string]time.Time // アクセス時刻追跡
	sizeTracker       map[string]int64     // サイズ追跡
	memoryStats       *MemoryStats         // メモリ統計
	cleanupTicker     *time.Ticker         // クリーンアップタイマー
	memoryCheckTicker *time.Ticker         // メモリチェックタイマー
	stopChan          chan bool            // 停止チャンネル
}

// メモリ統計情報
type MemoryStats struct {
	TotalAllocated     int64     `json:"total_allocated"`
	TotalCompressed    int64     `json:"total_compressed"`
	CompressionRatio   float64   `json:"compression_ratio"`
	ActiveSessions     int       `json:"active_sessions"`
	CompressedSessions int       `json:"compressed_sessions"`
	LastCleanup        time.Time `json:"last_cleanup"`
	LastGC             time.Time `json:"last_gc"`
	GCCount            int       `json:"gc_count"`
	MemoryUsage        int64     `json:"memory_usage"`
	SystemMemory       int64     `json:"system_memory"`
}

// 圧縮可能データインターフェース
type Compressible interface {
	Compress() ([]byte, error)
	Decompress(data []byte) error
	GetSize() int64
	GetID() string
}

// 新しいメモリ最適化マネージャーを作成
func NewMemoryOptimizer(config *MemoryConfig) *MemoryOptimizer {
	if config == nil {
		config = DefaultMemoryConfig()
	}

	mo := &MemoryOptimizer{
		config:           config,
		compressionCache: make(map[string][]byte),
		accessTracker:    make(map[string]time.Time),
		sizeTracker:      make(map[string]int64),
		memoryStats:      &MemoryStats{},
		stopChan:         make(chan bool),
	}

	// 定期的なクリーンアップを開始
	mo.startPeriodicTasks()

	return mo
}

// 定期タスクを開始
func (mo *MemoryOptimizer) startPeriodicTasks() {
	mo.cleanupTicker = time.NewTicker(mo.config.CleanupInterval)
	mo.memoryCheckTicker = time.NewTicker(mo.config.MemoryCheckInterval)

	go func() {
		for {
			select {
			case <-mo.cleanupTicker.C:
				mo.performCleanup()
			case <-mo.memoryCheckTicker.C:
				mo.checkMemoryUsage()
			case <-mo.stopChan:
				return
			}
		}
	}()
}

// データを圧縮
func (mo *MemoryOptimizer) CompressData(obj Compressible) error {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	id := obj.GetID()
	originalSize := obj.GetSize()

	// 圧縮閾値チェック
	if originalSize < mo.config.CompressionThreshold {
		return nil
	}

	// データを圧縮
	compressed, err := mo.compressObject(obj)
	if err != nil {
		return fmt.Errorf("圧縮エラー: %w", err)
	}

	// 圧縮効果をチェック
	compressionRatio := float64(len(compressed)) / float64(originalSize)
	if compressionRatio > mo.config.CompressionRatio {
		return nil // 圧縮効果が低い場合はスキップ
	}

	// 圧縮データを保存
	mo.compressionCache[id] = compressed
	mo.sizeTracker[id] = int64(len(compressed))
	mo.accessTracker[id] = time.Now()

	// 統計更新
	mo.memoryStats.TotalCompressed += int64(len(compressed))
	mo.memoryStats.CompressedSessions++
	mo.memoryStats.CompressionRatio = float64(mo.memoryStats.TotalCompressed) / float64(mo.memoryStats.TotalAllocated)

	return nil
}

// データを解凍
func (mo *MemoryOptimizer) DecompressData(id string, obj Compressible) error {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	compressed, exists := mo.compressionCache[id]
	if !exists {
		return fmt.Errorf("圧縮データが見つかりません: %s", id)
	}

	// アクセス時刻を更新
	mo.accessTracker[id] = time.Now()

	// データを解凍
	return mo.decompressObject(compressed, obj)
}

// オブジェクトを圧縮
func (mo *MemoryOptimizer) compressObject(obj Compressible) ([]byte, error) {
	// オブジェクトを圧縮可能形式に変換
	data, err := obj.Compress()
	if err != nil {
		return nil, err
	}

	// gzip圧縮
	return mo.gzipCompress(data)
}

// オブジェクトを解凍
func (mo *MemoryOptimizer) decompressObject(compressed []byte, obj Compressible) error {
	// gzip解凍
	data, err := mo.gzipDecompress(compressed)
	if err != nil {
		return err
	}

	// オブジェクトにデータを復元
	return obj.Decompress(data)
}

// gzip圧縮
func (mo *MemoryOptimizer) gzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// gzip解凍
func (mo *MemoryOptimizer) gzipDecompress(compressed []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var result bytes.Buffer
	if _, err := io.Copy(&result, reader); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}

// クリーンアップを実行
func (mo *MemoryOptimizer) performCleanup() {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	now := time.Now()
	deletedCount := 0

	// 古いデータを削除
	for id, lastAccess := range mo.accessTracker {
		if now.Sub(lastAccess) > mo.config.RetentionPeriod {
			delete(mo.compressionCache, id)
			delete(mo.accessTracker, id)
			delete(mo.sizeTracker, id)
			deletedCount++
		}
	}

	mo.memoryStats.LastCleanup = now
	mo.memoryStats.CompressedSessions -= deletedCount

	// ガベージコレクションを実行
	runtime.GC()
	mo.memoryStats.LastGC = now
	mo.memoryStats.GCCount++
}

// メモリ使用量をチェック
func (mo *MemoryOptimizer) checkMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mo.mu.Lock()
	mo.memoryStats.MemoryUsage = int64(m.Alloc)
	mo.memoryStats.SystemMemory = int64(m.Sys)
	mo.mu.Unlock()

	// メモリ使用量が制限を超えた場合の対応
	if int64(m.Alloc) > mo.config.MaxMemoryUsage {
		mo.emergencyCleanup()
	}
}

// 緊急クリーンアップ
func (mo *MemoryOptimizer) emergencyCleanup() {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	// 使用頻度の低いデータを優先的に削除
	accessTimes := make([]time.Time, 0, len(mo.accessTracker))
	for _, lastAccess := range mo.accessTracker {
		accessTimes = append(accessTimes, lastAccess)
	}

	// アクセス時刻でソート（古い順）
	for i := 0; i < len(accessTimes)-1; i++ {
		for j := i + 1; j < len(accessTimes); j++ {
			if accessTimes[i].After(accessTimes[j]) {
				accessTimes[i], accessTimes[j] = accessTimes[j], accessTimes[i]
			}
		}
	}

	// 古いデータを削除（最大50%まで）
	deleteCount := len(mo.compressionCache) / 2
	deletedCount := 0

	for id, lastAccess := range mo.accessTracker {
		if deletedCount >= deleteCount {
			break
		}

		if lastAccess.Equal(accessTimes[deletedCount]) {
			delete(mo.compressionCache, id)
			delete(mo.accessTracker, id)
			delete(mo.sizeTracker, id)
			deletedCount++
		}
	}

	mo.memoryStats.CompressedSessions -= deletedCount

	// 強制ガベージコレクション
	runtime.GC()
	runtime.GC() // 2回実行でより効果的
}

// メモリ統計を取得
func (mo *MemoryOptimizer) GetMemoryStats() *MemoryStats {
	mo.mu.RLock()
	defer mo.mu.RUnlock()

	// 現在のランタイム統計を取得
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := *mo.memoryStats
	stats.MemoryUsage = int64(m.Alloc)
	stats.SystemMemory = int64(m.Sys)
	stats.ActiveSessions = len(mo.accessTracker)

	return &stats
}

// 圧縮状況を取得
func (mo *MemoryOptimizer) GetCompressionInfo() map[string]interface{} {
	mo.mu.RLock()
	defer mo.mu.RUnlock()

	info := make(map[string]interface{})

	totalOriginalSize := int64(0)
	totalCompressedSize := int64(0)

	for id, compressedSize := range mo.sizeTracker {
		totalCompressedSize += compressedSize

		// 元のサイズは推定（圧縮前サイズが記録されていない場合）
		if originalSize, exists := mo.getOriginalSize(id); exists {
			totalOriginalSize += originalSize
		}
	}

	compressionRatio := 0.0
	if totalOriginalSize > 0 {
		compressionRatio = float64(totalCompressedSize) / float64(totalOriginalSize)
	}

	info["total_items"] = len(mo.compressionCache)
	info["total_original_size"] = totalOriginalSize
	info["total_compressed_size"] = totalCompressedSize
	info["compression_ratio"] = compressionRatio
	info["space_saved"] = totalOriginalSize - totalCompressedSize

	return info
}

// 元のサイズを取得（推定）
func (mo *MemoryOptimizer) getOriginalSize(id string) (int64, bool) {
	// ここでは圧縮されたサイズから元のサイズを推定
	// 実際の実装では、圧縮前のサイズを記録しておく方が正確
	if compressedSize, exists := mo.sizeTracker[id]; exists {
		estimatedOriginal := int64(float64(compressedSize) / mo.config.CompressionRatio)
		return estimatedOriginal, true
	}
	return 0, false
}

// メモリ最適化を停止
func (mo *MemoryOptimizer) Stop() {
	if mo.cleanupTicker != nil {
		mo.cleanupTicker.Stop()
	}
	if mo.memoryCheckTicker != nil {
		mo.memoryCheckTicker.Stop()
	}

	close(mo.stopChan)
}

// セッション用圧縮可能実装
type CompressibleSession struct {
	ID       string
	Data     interface{}
	Size     int64
	original []byte
}

func (cs *CompressibleSession) Compress() ([]byte, error) {
	data, err := json.Marshal(cs.Data)
	if err != nil {
		return nil, err
	}
	cs.original = data
	return data, nil
}

func (cs *CompressibleSession) Decompress(data []byte) error {
	return json.Unmarshal(data, &cs.Data)
}

func (cs *CompressibleSession) GetSize() int64 {
	if cs.Size > 0 {
		return cs.Size
	}
	if len(cs.original) > 0 {
		return int64(len(cs.original))
	}

	// サイズを推定
	data, err := json.Marshal(cs.Data)
	if err == nil {
		cs.Size = int64(len(data))
	}
	return cs.Size
}

func (cs *CompressibleSession) GetID() string {
	return cs.ID
}

// メモリ使用量レポートを生成
func (mo *MemoryOptimizer) GenerateMemoryReport() map[string]interface{} {
	stats := mo.GetMemoryStats()
	compressionInfo := mo.GetCompressionInfo()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	report := map[string]interface{}{
		"timestamp":        time.Now(),
		"memory_stats":     stats,
		"compression_info": compressionInfo,
		"runtime_stats": map[string]interface{}{
			"alloc":          m.Alloc,
			"total_alloc":    m.TotalAlloc,
			"sys":            m.Sys,
			"num_gc":         m.NumGC,
			"gc_cpu_percent": m.GCCPUFraction,
			"heap_alloc":     m.HeapAlloc,
			"heap_sys":       m.HeapSys,
			"heap_idle":      m.HeapIdle,
			"heap_inuse":     m.HeapInuse,
		},
		"optimization_config": map[string]interface{}{
			"max_session_size":      mo.config.MaxSessionSize,
			"compression_threshold": mo.config.CompressionThreshold,
			"compression_ratio":     mo.config.CompressionRatio,
			"cleanup_interval":      mo.config.CleanupInterval.String(),
			"retention_period":      mo.config.RetentionPeriod.String(),
			"max_memory_usage":      mo.config.MaxMemoryUsage,
		},
	}

	return report
}

// メモリ使用量アラートをチェック
func (mo *MemoryOptimizer) CheckMemoryAlerts() []string {
	var alerts []string

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// メモリ使用量が90%を超えた場合
	if int64(m.Alloc) > mo.config.MaxMemoryUsage*9/10 {
		alerts = append(alerts, "メモリ使用量が制限の90%を超えています")
	}

	// 圧縮効果が低い場合
	if mo.memoryStats.CompressionRatio > 0.7 {
		alerts = append(alerts, "データ圧縮の効果が低くなっています")
	}

	// セッション数が多すぎる場合
	if len(mo.compressionCache) > 100 {
		alerts = append(alerts, "圧縮セッション数が多すぎます")
	}

	return alerts
}
