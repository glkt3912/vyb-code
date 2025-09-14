package streaming

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// Manager - 統合ストリーミング管理
type Manager struct {
	mu            sync.RWMutex
	config        *UnifiedStreamConfig
	llmProcessor  *LLMProcessor
	uiProcessor   *UIProcessor
	activeStreams map[string]*ActiveStream
	globalMetrics *GlobalMetrics
}

// ActiveStream - アクティブストリーム情報
type ActiveStream struct {
	ID        string                 `json:"id"`
	Type      StreamType             `json:"type"`
	Status    StreamStatus           `json:"status"`
	StartTime time.Time              `json:"start_time"`
	Options   *StreamOptions         `json:"options"`
	Metadata  map[string]interface{} `json:"metadata"`
	processor StreamingProcessor     `json:"-"`
}

// GlobalMetrics - グローバルストリーミングメトリクス
type GlobalMetrics struct {
	TotalRequests       int64         `json:"total_requests"`
	ActiveRequests      int64         `json:"active_requests"`
	TotalLLMStreams     int64         `json:"total_llm_streams"`
	TotalUIStreams      int64         `json:"total_ui_streams"`
	AverageProcessTime  time.Duration `json:"average_process_time"`
	TotalProcessTime    time.Duration `json:"total_process_time"`
	ErrorCount          int64         `json:"error_count"`
	InterruptCount      int64         `json:"interrupt_count"`
	LastRequestTime     time.Time     `json:"last_request_time"`
	PerformanceScore    float64       `json:"performance_score"`
}

// NewManager - 新しい統合ストリーミング管理を作成
func NewManager(config *UnifiedStreamConfig) *Manager {
	if config == nil {
		config = DefaultStreamConfig()
	}
	
	manager := &Manager{
		config:        config,
		llmProcessor:  NewLLMProcessor(config),
		uiProcessor:   NewUIProcessor(config),
		activeStreams: make(map[string]*ActiveStream),
		globalMetrics: &GlobalMetrics{},
	}
	
	// パフォーマンス監視を開始
	go manager.startPerformanceMonitoring()
	
	return manager
}

// Process - 統合ストリーミング処理
func (m *Manager) Process(ctx context.Context, input io.Reader, output io.Writer, options *StreamOptions) error {
	if options == nil {
		options = &StreamOptions{Type: StreamTypeUIDisplay}
	}
	
	// ストリームIDを生成
	if options.StreamID == "" {
		options.StreamID = m.generateStreamID(options.Type)
	}
	
	// アクティブストリームを登録
	_ = m.registerActiveStream(options)
	defer m.unregisterActiveStream(options.StreamID)
	
	startTime := time.Now()
	var err error
	
	// プロセッサー選択と実行
	switch options.Type {
	case StreamTypeLLMResponse:
		err = m.llmProcessor.Process(ctx, input, output, options)
		
	case StreamTypeUIDisplay, StreamTypeInterrupted:
		err = m.uiProcessor.Process(ctx, input, output, options)
		
	default:
		err = fmt.Errorf("不明なストリームタイプ: %s", options.Type)
	}
	
	// 処理時間の記録（ここで統計も更新）
	processingTime := time.Since(startTime)
	m.updateProcessingMetrics(processingTime, err != nil, options.Type)
	
	return err
}

// ProcessString - 文字列を統合ストリーミング処理
func (m *Manager) ProcessString(ctx context.Context, content string, output io.Writer, options *StreamOptions) error {
	if options == nil {
		options = &StreamOptions{Type: StreamTypeUIDisplay}
	}
	
	// ストリームIDを生成
	if options.StreamID == "" {
		options.StreamID = m.generateStreamID(options.Type)
	}
	
	// アクティブストリームを登録
	_ = m.registerActiveStream(options)
	defer m.unregisterActiveStream(options.StreamID)
	
	startTime := time.Now()
	var err error
	
	// プロセッサー選択と実行
	switch options.Type {
	case StreamTypeLLMResponse:
		err = m.llmProcessor.ProcessString(ctx, content, output, options)
	case StreamTypeUIDisplay, StreamTypeInterrupted:
		err = m.uiProcessor.ProcessString(ctx, content, output, options)
	default:
		err = fmt.Errorf("不明なストリームタイプ: %s", options.Type)
	}
	
	// 処理時間の記録（ここで統計も更新）
	processingTime := time.Since(startTime)
	m.updateProcessingMetrics(processingTime, err != nil, options.Type)
	
	return err
}

// UpdateConfig - 設定を更新
func (m *Manager) UpdateConfig(config *UnifiedStreamConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.config = config
	m.llmProcessor.SetConfig(config)
	m.uiProcessor.SetConfig(config)
}

// GetGlobalMetrics - グローバルメトリクスを取得
func (m *Manager) GetGlobalMetrics() *GlobalMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metricsCopy := *m.globalMetrics
	return &metricsCopy
}

// GetProcessorMetrics - プロセッサー別メトリクスを取得
func (m *Manager) GetProcessorMetrics() map[string]*StreamMetrics {
	return map[string]*StreamMetrics{
		"llm_processor": m.llmProcessor.GetMetrics(),
		"ui_processor":  m.uiProcessor.GetMetrics(),
	}
}

// GetActiveStreams - アクティブストリーム一覧を取得
func (m *Manager) GetActiveStreams() map[string]*ActiveStream {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]*ActiveStream)
	for id, stream := range m.activeStreams {
		streamCopy := *stream
		result[id] = &streamCopy
	}
	
	return result
}

// CancelStream - ストリームをキャンセル
func (m *Manager) CancelStream(streamID string) error {
	m.mu.RLock()
	stream, exists := m.activeStreams[streamID]
	m.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("ストリーム '%s' が見つかりません", streamID)
	}
	
	stream.Status = StreamStatusCanceled
	return nil
}

// generateStreamID - ストリームIDを生成
func (m *Manager) generateStreamID(streamType StreamType) string {
	return fmt.Sprintf("%s-%d", streamType, time.Now().UnixNano())
}

// registerActiveStream - アクティブストリームを登録
func (m *Manager) registerActiveStream(options *StreamOptions) *ActiveStream {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var processor StreamingProcessor
	switch options.Type {
	case StreamTypeLLMResponse:
		processor = m.llmProcessor
	case StreamTypeUIDisplay, StreamTypeInterrupted:
		processor = m.uiProcessor
	}
	
	stream := &ActiveStream{
		ID:        options.StreamID,
		Type:      options.Type,
		Status:    StreamStatusStarting,
		StartTime: time.Now(),
		Options:   options,
		Metadata:  options.Metadata,
		processor: processor,
	}
	
	m.activeStreams[options.StreamID] = stream
	return stream
}

// unregisterActiveStream - アクティブストリームの登録を解除
func (m *Manager) unregisterActiveStream(streamID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if stream, exists := m.activeStreams[streamID]; exists {
		stream.Status = StreamStatusCompleted
		delete(m.activeStreams, streamID)
		m.globalMetrics.ActiveRequests--
	}
}

// updateProcessingMetrics - 処理メトリクスを更新
func (m *Manager) updateProcessingMetrics(processingTime time.Duration, hasError bool, streamType StreamType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// リクエスト統計を更新
	m.globalMetrics.TotalRequests++
	m.globalMetrics.ActiveRequests++
	m.globalMetrics.LastRequestTime = time.Now()
	m.globalMetrics.TotalProcessTime += processingTime
	
	// ストリームタイプ別統計を更新
	switch streamType {
	case StreamTypeLLMResponse:
		m.globalMetrics.TotalLLMStreams++
	case StreamTypeUIDisplay, StreamTypeInterrupted:
		m.globalMetrics.TotalUIStreams++
	}
	
	// 平均処理時間を計算
	if m.globalMetrics.TotalRequests > 0 {
		m.globalMetrics.AverageProcessTime = m.globalMetrics.TotalProcessTime / time.Duration(m.globalMetrics.TotalRequests)
	}
	
	if hasError {
		m.globalMetrics.ErrorCount++
	}
	
	// パフォーマンススコアを計算（0.0-1.0）
	m.calculatePerformanceScore()
}

// calculatePerformanceScore - パフォーマンススコアを計算
func (m *Manager) calculatePerformanceScore() {
	if m.globalMetrics.TotalRequests == 0 {
		m.globalMetrics.PerformanceScore = 1.0
		return
	}
	
	// エラー率を考慮
	errorRate := float64(m.globalMetrics.ErrorCount) / float64(m.globalMetrics.TotalRequests)
	
	// 処理速度を考慮（理想的な処理時間を1秒とする）
	idealProcessTime := 1 * time.Second
	speedRatio := float64(idealProcessTime) / float64(m.globalMetrics.AverageProcessTime)
	if speedRatio > 1.0 {
		speedRatio = 1.0
	}
	
	// 総合スコア計算（エラー率: 70%, 速度: 30%）
	m.globalMetrics.PerformanceScore = (1.0-errorRate)*0.7 + speedRatio*0.3
}

// startPerformanceMonitoring - パフォーマンス監視を開始
func (m *Manager) startPerformanceMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.performanceCheck()
		}
	}
}

// performanceCheck - パフォーマンスチェック
func (m *Manager) performanceCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 長時間実行中のストリームをチェック
	now := time.Now()
	for id, stream := range m.activeStreams {
		if now.Sub(stream.StartTime) > m.config.Timeout {
			stream.Status = StreamStatusError
			delete(m.activeStreams, id)
			m.globalMetrics.ActiveRequests--
			m.globalMetrics.ErrorCount++
		}
	}
}