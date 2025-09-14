package streaming

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LLMProcessor - LLM応答専用ストリーミングプロセッサー
type LLMProcessor struct {
	mu            sync.RWMutex
	config        *UnifiedStreamConfig
	handlers      map[EventType]EventHandler
	currentStream *LLMStream
	metrics       *StreamMetrics

	// パフォーマンス最適化
	bufferPool sync.Pool
	workerPool chan struct{}
	eventQueue chan StreamEvent
}

// LLMStream - LLM応答ストリーム情報
type LLMStream struct {
	ID         string                 `json:"id"`
	Model      string                 `json:"model"`
	StartTime  time.Time              `json:"start_time"`
	TokenCount int                    `json:"token_count"`
	ChunkCount int                    `json:"chunk_count"`
	Status     StreamStatus           `json:"status"`
	Metadata   map[string]interface{} `json:"metadata"`
	Buffer     strings.Builder        `json:"-"`
}

// NewLLMProcessor - 新しいLLMプロセッサーを作成
func NewLLMProcessor(config *UnifiedStreamConfig) *LLMProcessor {
	if config == nil {
		config = DefaultStreamConfig()
	}

	maxWorkers := runtime.NumCPU()
	if config.MaxWorkers > 0 {
		maxWorkers = config.MaxWorkers
	}

	queueSize := maxWorkers * 10
	if config.QueueSize > 0 {
		queueSize = config.QueueSize
	}

	processor := &LLMProcessor{
		config:     config,
		handlers:   make(map[EventType]EventHandler),
		metrics:    &StreamMetrics{},
		workerPool: make(chan struct{}, maxWorkers),
		eventQueue: make(chan StreamEvent, queueSize),
	}

	// バッファプールを初期化
	processor.bufferPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	// イベント処理ワーカーを開始
	go processor.startEventWorkers()

	return processor
}

// Process - LLM応答をストリーミング処理
func (p *LLMProcessor) Process(ctx context.Context, input io.Reader, output io.Writer, options *StreamOptions) error {
	if options == nil {
		options = &StreamOptions{Type: StreamTypeLLMResponse}
	}

	// ストリーム開始
	stream := p.startStream(options)
	if stream == nil {
		return fmt.Errorf("ストリーム開始に失敗しました")
	}

	defer p.completeStream()

	scanner := bufio.NewScanner(input)
	buf := make([]byte, p.config.BufferSize)
	scanner.Buffer(buf, p.config.BufferSize)

	ticker := time.NewTicker(p.config.FlushInterval)
	defer ticker.Stop()

	// バッファプールから取得
	pendingBuffer := p.bufferPool.Get().(*strings.Builder)
	defer func() {
		pendingBuffer.Reset()
		p.bufferPool.Put(pendingBuffer)
	}()

	lastFlush := time.Now()

	for {
		select {
		case <-ctx.Done():
			p.handleStreamCancel()
			return ctx.Err()
		case <-ticker.C:
			if pendingBuffer.Len() > 0 {
				if err := p.flushContent(output, pendingBuffer.String()); err != nil {
					return err
				}
				pendingBuffer.Reset()
				lastFlush = time.Now()
			}
		default:
			if scanner.Scan() {
				chunk := scanner.Text()

				if err := p.processChunk(chunk, pendingBuffer, options); err != nil {
					p.handleStreamError(err)
					return err
				}

				// 即座にフラッシュするか判定
				if time.Since(lastFlush) > p.config.FlushInterval ||
					pendingBuffer.Len() > p.config.BufferSize/2 {
					if err := p.flushContent(output, pendingBuffer.String()); err != nil {
						return err
					}
					pendingBuffer.Reset()
					lastFlush = time.Now()
				}
			} else {
				// スキャン完了
				if pendingBuffer.Len() > 0 {
					p.flushContent(output, pendingBuffer.String())
				}

				if err := scanner.Err(); err != nil {
					p.handleStreamError(err)
					return err
				}

				return nil
			}
		}
	}
}

// ProcessString - 文字列をストリーミング処理
func (p *LLMProcessor) ProcessString(ctx context.Context, content string, output io.Writer, options *StreamOptions) error {
	reader := strings.NewReader(content)
	return p.Process(ctx, reader, output, options)
}

// SetConfig - 設定を更新
func (p *LLMProcessor) SetConfig(config *UnifiedStreamConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = config
}

// GetMetrics - メトリクス取得
func (p *LLMProcessor) GetMetrics() *StreamMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	metricsCopy := *p.metrics
	return &metricsCopy
}

// startStream - ストリーム開始
func (p *LLMProcessor) startStream(options *StreamOptions) *LLMStream {
	p.mu.Lock()
	defer p.mu.Unlock()

	stream := &LLMStream{
		ID:        options.StreamID,
		Model:     options.Model,
		StartTime: time.Now(),
		Status:    StreamStatusStarting,
		Metadata:  options.Metadata,
	}

	if stream.ID == "" {
		stream.ID = fmt.Sprintf("llm-stream-%d", time.Now().UnixNano())
	}

	p.currentStream = stream
	p.metrics.TotalStreams++
	p.metrics.ActiveStreams++

	// 開始イベントを発行
	go p.emitEvent(StreamEvent{
		Type:      EventStreamStart,
		StreamID:  stream.ID,
		Timestamp: time.Now(),
		Data:      stream,
		Metadata:  options.Metadata,
	})

	return stream
}

// processChunk - チャンクを処理
func (p *LLMProcessor) processChunk(chunk string, buffer *strings.Builder, options *StreamOptions) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentStream == nil {
		return fmt.Errorf("アクティブなストリームがありません")
	}

	p.currentStream.Status = StreamStatusStreaming

	// JSONストリーミング形式の解析
	if strings.HasPrefix(chunk, "data: ") {
		jsonData := strings.TrimPrefix(chunk, "data: ")
		if jsonData == "[DONE]" {
			return nil
		}

		var response struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
			// JSON解析に失敗した場合はそのまま追加
			buffer.WriteString(chunk)
			return nil
		}

		// コンテンツを抽出
		if len(response.Choices) > 0 {
			content := response.Choices[0].Delta.Content
			if content != "" {
				buffer.WriteString(content)
				p.currentStream.Buffer.WriteString(content)
				p.currentStream.TokenCount += len(strings.Fields(content))
			}
		}
	} else {
		// 非JSON形式の場合はそのまま追加
		buffer.WriteString(chunk)
		p.currentStream.Buffer.WriteString(chunk)
	}

	p.currentStream.ChunkCount++
	p.metrics.TotalChunks++

	// チャンク受信イベントを発行
	go p.emitEvent(StreamEvent{
		Type:      EventChunkReceived,
		StreamID:  p.currentStream.ID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"content":     chunk,
			"token_count": p.currentStream.TokenCount,
			"is_complete": false,
		},
	})

	return nil
}

// flushContent - コンテンツをフラッシュ
func (p *LLMProcessor) flushContent(output io.Writer, content string) error {
	if content == "" {
		return nil
	}

	if _, err := output.Write([]byte(content)); err != nil {
		return err
	}

	// 出力を強制フラッシュ
	if flusher, ok := output.(interface{ Flush() error }); ok {
		flusher.Flush()
	}

	return nil
}

// completeStream - ストリーム完了処理
func (p *LLMProcessor) completeStream() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentStream == nil {
		return
	}

	p.currentStream.Status = StreamStatusCompleted
	duration := time.Since(p.currentStream.StartTime)

	p.metrics.ActiveStreams--
	p.metrics.TotalTokens += int64(p.currentStream.TokenCount)
	p.metrics.LastStreamTime = time.Now()
	p.metrics.ProcessingTime += duration

	// 平均レイテンシーを更新
	if p.metrics.TotalStreams > 0 {
		p.metrics.AverageLatency = time.Duration(
			(int64(p.metrics.AverageLatency)*p.metrics.TotalStreams + int64(duration)) /
				(p.metrics.TotalStreams + 1))
	} else {
		p.metrics.AverageLatency = duration
	}

	// 完了イベントを発行
	go p.emitEvent(StreamEvent{
		Type:      EventStreamComplete,
		StreamID:  p.currentStream.ID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"duration":     duration,
			"total_tokens": p.currentStream.TokenCount,
			"total_chunks": p.currentStream.ChunkCount,
		},
	})

	p.currentStream = nil
}

// handleStreamError - ストリームエラー処理
func (p *LLMProcessor) handleStreamError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentStream == nil {
		return
	}

	p.currentStream.Status = StreamStatusError
	p.metrics.ActiveStreams--
	p.metrics.ErrorCount++

	go p.emitEvent(StreamEvent{
		Type:      EventStreamError,
		StreamID:  p.currentStream.ID,
		Timestamp: time.Now(),
		Error:     err.Error(),
	})

	p.currentStream = nil
}

// handleStreamCancel - ストリームキャンセル処理
func (p *LLMProcessor) handleStreamCancel() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentStream == nil {
		return
	}

	p.currentStream.Status = StreamStatusCanceled
	p.metrics.ActiveStreams--

	go p.emitEvent(StreamEvent{
		Type:      EventStreamCancel,
		StreamID:  p.currentStream.ID,
		Timestamp: time.Now(),
	})

	p.currentStream = nil
}

// RegisterHandler - イベントハンドラーを登録
func (p *LLMProcessor) RegisterHandler(eventType EventType, handler EventHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[eventType] = handler
}

// startEventWorkers - イベント処理ワーカー開始
func (p *LLMProcessor) startEventWorkers() {
	for {
		select {
		case event := <-p.eventQueue:
			p.processEvent(event)
		}
	}
}

// processEvent - イベントを処理
func (p *LLMProcessor) processEvent(event StreamEvent) {
	p.workerPool <- struct{}{}
	go func() {
		defer func() { <-p.workerPool }()

		p.mu.RLock()
		handler, exists := p.handlers[event.Type]
		p.mu.RUnlock()

		if exists {
			handler(event)
		}
	}()
}

// emitEvent - イベントを発行
func (p *LLMProcessor) emitEvent(event StreamEvent) {
	select {
	case p.eventQueue <- event:
		// キューに正常に追加
	default:
		// キューが満杯の場合は直接処理
		p.processEvent(event)
	}
}
