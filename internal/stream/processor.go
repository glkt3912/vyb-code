package stream

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

// ストリーミング応答プロセッサー
type Processor struct {
	mu            sync.RWMutex
	bufferSize    int
	flushInterval time.Duration
	handlers      map[EventType]EventHandler
	currentStream *Stream
	metrics       StreamMetrics

	// パフォーマンス最適化
	bufferPool sync.Pool        // バッファプール
	workerPool chan struct{}    // ワーカープール
	eventQueue chan StreamEvent // イベントキュー
	maxWorkers int              // 最大ワーカー数
	queueSize  int              // キューサイズ
}

// ストリーム情報
type Stream struct {
	ID         string                 `json:"id"`
	Model      string                 `json:"model"`
	StartTime  time.Time              `json:"startTime"`
	TokenCount int                    `json:"tokenCount"`
	ChunkCount int                    `json:"chunkCount"`
	Status     StreamStatus           `json:"status"`
	Metadata   map[string]interface{} `json:"metadata"`
	Buffer     strings.Builder        `json:"-"`
}

// ストリーム状態
type StreamStatus string

const (
	StreamStatusStarting  StreamStatus = "starting"
	StreamStatusStreaming StreamStatus = "streaming"
	StreamStatusCompleted StreamStatus = "completed"
	StreamStatusError     StreamStatus = "error"
	StreamStatusCanceled  StreamStatus = "canceled"
)

// イベントタイプ
type EventType string

const (
	EventStreamStart    EventType = "stream_start"
	EventChunkReceived  EventType = "chunk_received"
	EventStreamComplete EventType = "stream_complete"
	EventStreamError    EventType = "stream_error"
	EventStreamCancel   EventType = "stream_cancel"
)

// イベントハンドラー関数型
type EventHandler func(event StreamEvent) error

// ストリーミングイベント
type StreamEvent struct {
	Type      EventType   `json:"type"`
	StreamID  string      `json:"streamId"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// チャンクデータ
type ChunkData struct {
	Content    string            `json:"content"`
	TokenCount int               `json:"tokenCount"`
	IsComplete bool              `json:"isComplete"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ストリーミングメトリクス
type StreamMetrics struct {
	TotalStreams   int64         `json:"totalStreams"`
	ActiveStreams  int64         `json:"activeStreams"`
	TotalTokens    int64         `json:"totalTokens"`
	TotalChunks    int64         `json:"totalChunks"`
	AverageLatency time.Duration `json:"averageLatency"`
	ErrorCount     int64         `json:"errorCount"`
	LastStreamTime time.Time     `json:"lastStreamTime"`
}

// 新しいプロセッサーを作成
func NewProcessor() *Processor {
	maxWorkers := runtime.NumCPU()
	queueSize := maxWorkers * 10

	p := &Processor{
		bufferSize:    4096,
		flushInterval: 50 * time.Millisecond,
		handlers:      make(map[EventType]EventHandler),
		metrics:       StreamMetrics{},
		maxWorkers:    maxWorkers,
		queueSize:     queueSize,
		workerPool:    make(chan struct{}, maxWorkers),
		eventQueue:    make(chan StreamEvent, queueSize),
	}

	// バッファプールを初期化
	p.bufferPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	// イベント処理ワーカーを開始
	go p.startEventWorkers()

	return p
}

// イベントハンドラーを登録
func (p *Processor) RegisterHandler(eventType EventType, handler EventHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[eventType] = handler
}

// ストリーミング開始
func (p *Processor) StartStream(streamID, model string, metadata map[string]interface{}) *Stream {
	p.mu.Lock()
	defer p.mu.Unlock()

	stream := &Stream{
		ID:        streamID,
		Model:     model,
		StartTime: time.Now(),
		Status:    StreamStatusStarting,
		Metadata:  metadata,
	}

	p.currentStream = stream
	p.metrics.TotalStreams++
	p.metrics.ActiveStreams++

	// 開始イベントを発行
	go p.emitEvent(StreamEvent{
		Type:      EventStreamStart,
		StreamID:  streamID,
		Timestamp: time.Now(),
		Data:      stream,
	})

	return stream
}

// LLM応答をストリーミング処理
func (p *Processor) ProcessLLMStream(ctx context.Context, reader io.Reader, output io.Writer) error {
	if p.currentStream == nil {
		return fmt.Errorf("アクティブなストリームがありません")
	}

	p.currentStream.Status = StreamStatusStreaming
	scanner := bufio.NewScanner(reader)

	// バッファサイズを設定
	buf := make([]byte, p.bufferSize)
	scanner.Buffer(buf, p.bufferSize)

	ticker := time.NewTicker(p.flushInterval)
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
			// 定期的にバッファをフラッシュ
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

				// チャンクを処理
				if err := p.processChunk(chunk, pendingBuffer); err != nil {
					p.handleStreamError(err)
					return err
				}

				// 即座にフラッシュするか判定
				if time.Since(lastFlush) > p.flushInterval || pendingBuffer.Len() > p.bufferSize/2 {
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

				p.handleStreamComplete()
				return nil
			}
		}
	}
}

// チャンクを処理
func (p *Processor) processChunk(chunk string, buffer *strings.Builder) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentStream == nil {
		return fmt.Errorf("アクティブなストリームがありません")
	}

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
		Data: ChunkData{
			Content:    chunk,
			TokenCount: p.currentStream.TokenCount,
			IsComplete: false,
		},
	})

	return nil
}

// コンテンツをフラッシュ
func (p *Processor) flushContent(output io.Writer, content string) error {
	if content == "" {
		return nil
	}

	// 出力にリアルタイムで書き込み
	if _, err := output.Write([]byte(content)); err != nil {
		return err
	}

	// 出力を強制フラッシュ
	if flusher, ok := output.(interface{ Flush() error }); ok {
		flusher.Flush()
	}

	return nil
}

// ストリーム完了を処理
func (p *Processor) handleStreamComplete() {
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

// ストリームエラーを処理
func (p *Processor) handleStreamError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentStream == nil {
		return
	}

	p.currentStream.Status = StreamStatusError
	p.metrics.ActiveStreams--
	p.metrics.ErrorCount++

	// エラーイベントを発行
	go p.emitEvent(StreamEvent{
		Type:      EventStreamError,
		StreamID:  p.currentStream.ID,
		Timestamp: time.Now(),
		Error:     err.Error(),
	})

	p.currentStream = nil
}

// ストリームキャンセルを処理
func (p *Processor) handleStreamCancel() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentStream == nil {
		return
	}

	p.currentStream.Status = StreamStatusCanceled
	p.metrics.ActiveStreams--

	// キャンセルイベントを発行
	go p.emitEvent(StreamEvent{
		Type:      EventStreamCancel,
		StreamID:  p.currentStream.ID,
		Timestamp: time.Now(),
	})

	p.currentStream = nil
}

// イベント処理ワーカーを開始
func (p *Processor) startEventWorkers() {
	for {
		select {
		case event := <-p.eventQueue:
			p.processEvent(event)
		}
	}
}

// イベントを処理
func (p *Processor) processEvent(event StreamEvent) {
	// ワーカープールを使用
	p.workerPool <- struct{}{}
	go func() {
		defer func() { <-p.workerPool }()

		p.mu.RLock()
		handler, exists := p.handlers[event.Type]
		p.mu.RUnlock()

		if exists {
			if err := handler(event); err != nil {
				// エラーハンドリングは各実装に委譲
			}
		}
	}()
}

// イベントを発行（最適化版）
func (p *Processor) emitEvent(event StreamEvent) {
	select {
	case p.eventQueue <- event:
		// キューに正常に追加
	default:
		// キューが満杯の場合は直接処理
		p.processEvent(event)
	}
}

// 現在のストリーム情報を取得
func (p *Processor) GetCurrentStream() *Stream {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.currentStream == nil {
		return nil
	}

	// ストリーム情報のコピーを返す
	streamCopy := *p.currentStream
	return &streamCopy
}

// ストリーミングメトリクスを取得
func (p *Processor) GetMetrics() StreamMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.metrics
}

// 設定を更新
func (p *Processor) UpdateConfig(bufferSize int, flushInterval time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if bufferSize > 0 {
		p.bufferSize = bufferSize
	}
	if flushInterval > 0 {
		p.flushInterval = flushInterval
	}
}

// ストリームをキャンセル
func (p *Processor) CancelStream() {
	p.handleStreamCancel()
}

// メトリクスをリセット
func (p *Processor) ResetMetrics() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.metrics = StreamMetrics{}
}

// プロセッサーを安全にシャットダウン
func (p *Processor) Shutdown(ctx context.Context) error {
	// 現在のストリームを完了
	if p.currentStream != nil {
		p.handleStreamComplete()
	}

	// イベントキューを処理完了まで待機
	done := make(chan struct{})
	go func() {
		defer close(done)
		for len(p.eventQueue) > 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
