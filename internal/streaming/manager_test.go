package streaming

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestManager_ProcessString(t *testing.T) {
	manager := NewManager(DefaultStreamConfig())

	tests := []struct {
		name    string
		content string
		options *StreamOptions
		wantErr bool
	}{
		{
			name:    "UI display streaming",
			content: "Hello, world!",
			options: &StreamOptions{
				Type:     StreamTypeUIDisplay,
				StreamID: "test-ui-1",
			},
			wantErr: false,
		},
		{
			name:    "LLM response streaming",
			content: "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\ndata: [DONE]\n",
			options: &StreamOptions{
				Type:     StreamTypeLLMResponse,
				StreamID: "test-llm-1",
				Model:    "test-model",
			},
			wantErr: false,
		},
		{
			name:    "Interruptible streaming",
			content: "This is a test message that should be interruptible.",
			options: &StreamOptions{
				Type:            StreamTypeInterrupted,
				StreamID:        "test-interrupt-1",
				EnableInterrupt: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &strings.Builder{}
			ctx := context.Background()

			err := manager.ProcessString(ctx, tt.content, output, tt.options)

			if (err != nil) != tt.wantErr {
				t.Errorf("Manager.ProcessString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && output.Len() == 0 {
				t.Errorf("Manager.ProcessString() no output produced")
			}
		})
	}
}

func TestManager_ProcessStringWithCancellation(t *testing.T) {
	manager := NewManager(DefaultStreamConfig())

	ctx, cancel := context.WithCancel(context.Background())
	options := &StreamOptions{
		Type:            StreamTypeInterrupted,
		StreamID:        "test-cancel",
		EnableInterrupt: true,
	}

	output := &strings.Builder{}

	// 少し遅延後にキャンセル
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	content := strings.Repeat("This is a long content that should be interrupted. ", 100)
	err := manager.ProcessString(ctx, content, output, options)

	if err == nil {
		t.Errorf("Manager.ProcessString() should return error when cancelled")
	}

	if !strings.Contains(err.Error(), "interrupted") && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("Manager.ProcessString() should return cancellation error, got: %v", err)
	}
}

func TestManager_GetGlobalMetrics(t *testing.T) {
	manager := NewManager(DefaultStreamConfig())

	// 初期メトリクス
	metrics := manager.GetGlobalMetrics()
	if metrics.TotalRequests != 0 {
		t.Errorf("Initial TotalRequests should be 0, got %d", metrics.TotalRequests)
	}

	// ストリーミング実行
	ctx := context.Background()
	options := &StreamOptions{Type: StreamTypeUIDisplay}
	output := &strings.Builder{}

	manager.ProcessString(ctx, "test content", output, options)

	// メトリクス確認
	metrics = manager.GetGlobalMetrics()
	if metrics.TotalRequests != 1 {
		t.Errorf("TotalRequests should be 1 after processing, got %d", metrics.TotalRequests)
	}

	if metrics.TotalUIStreams != 1 {
		t.Errorf("TotalUIStreams should be 1, got %d", metrics.TotalUIStreams)
	}
}

func TestManager_UpdateConfig(t *testing.T) {
	manager := NewManager(DefaultStreamConfig())

	newConfig := &UnifiedStreamConfig{
		BufferSize:      8192,
		FlushInterval:   100 * time.Millisecond,
		TokenDelay:      20 * time.Millisecond,
		EnableStreaming: false,
		MaxWorkers:      2,
		QueueSize:       20,
		Timeout:         60 * time.Second,
	}

	manager.UpdateConfig(newConfig)

	// 設定が正しく更新されているか確認
	if manager.config.BufferSize != 8192 {
		t.Errorf("BufferSize not updated, got %d", manager.config.BufferSize)
	}

	if manager.config.EnableStreaming != false {
		t.Errorf("EnableStreaming not updated, got %v", manager.config.EnableStreaming)
	}
}

// TestLegacyStreamAdapter removed - legacy streaming system deleted

func TestStreamOptions_Validation(t *testing.T) {
	manager := NewManager(DefaultStreamConfig())

	tests := []struct {
		name    string
		options *StreamOptions
		wantErr bool
	}{
		{
			name:    "Valid options",
			options: &StreamOptions{Type: StreamTypeUIDisplay},
			wantErr: false,
		},
		{
			name:    "Nil options (should use defaults)",
			options: nil,
			wantErr: false,
		},
		{
			name:    "Unknown stream type",
			options: &StreamOptions{Type: "unknown_type"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &strings.Builder{}
			ctx := context.Background()

			err := manager.ProcessString(ctx, "test", output, tt.options)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkManager_ProcessString(b *testing.B) {
	manager := NewManager(DefaultStreamConfig())
	options := &StreamOptions{Type: StreamTypeUIDisplay}
	content := "This is a benchmark test content for streaming performance measurement."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		output := &strings.Builder{}
		ctx := context.Background()

		err := manager.ProcessString(ctx, content, output, options)
		if err != nil {
			b.Errorf("ProcessString() error = %v", err)
		}
	}
}

// BenchmarkLegacyStreamAdapter_StreamContent removed - legacy streaming system deleted
