package stream

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

// TestNewProcessor は新しいプロセッサー作成をテストする
func TestNewProcessor(t *testing.T) {
	processor := NewProcessor()

	if processor == nil {
		t.Fatal("プロセッサーの作成に失敗")
	}

	if processor.bufferSize != 4096 {
		t.Errorf("期待バッファサイズ: 4096, 実際: %d", processor.bufferSize)
	}

	if processor.flushInterval != 50*time.Millisecond {
		t.Errorf("期待フラッシュ間隔: 50ms, 実際: %v", processor.flushInterval)
	}

	if processor.handlers == nil {
		t.Error("ハンドラーマップが初期化されていません")
	}
}

// TestRegisterHandler はイベントハンドラー登録をテストする
func TestRegisterHandler(t *testing.T) {
	processor := NewProcessor()
	called := false

	handler := func(event StreamEvent) error {
		called = true
		return nil
	}

	processor.RegisterHandler(EventStreamStart, handler)

	// ハンドラーが登録されたことを確認（直接的な確認は困難なので、呼び出しで確認）
	processor.emitEvent(StreamEvent{
		Type:      EventStreamStart,
		StreamID:  "test",
		Timestamp: time.Now(),
	})

	// 少し待ってからチェック（goroutineで実行されるため）
	time.Sleep(10 * time.Millisecond)

	if !called {
		t.Error("ハンドラーが呼び出されませんでした")
	}
}

// TestStartStream はストリーミング開始をテストする
func TestStartStream(t *testing.T) {
	processor := NewProcessor()

	streamID := "test-stream"
	model := "test-model"
	metadata := map[string]interface{}{
		"test": "value",
	}

	stream := processor.StartStream(streamID, model, metadata)

	if stream == nil {
		t.Fatal("ストリームの作成に失敗")
	}

	if stream.ID != streamID {
		t.Errorf("期待ストリームID: %s, 実際: %s", streamID, stream.ID)
	}

	if stream.Model != model {
		t.Errorf("期待モデル: %s, 実際: %s", model, stream.Model)
	}

	if stream.Status != StreamStatusStarting {
		t.Errorf("期待ステータス: %s, 実際: %s", StreamStatusStarting, stream.Status)
	}

	if stream.TokenCount != 0 {
		t.Errorf("初期トークン数が0ではありません: %d", stream.TokenCount)
	}

	if stream.ChunkCount != 0 {
		t.Errorf("初期チャンク数が0ではありません: %d", stream.ChunkCount)
	}

	// メトリクスが更新されたことを確認
	metrics := processor.GetMetrics()
	if metrics.TotalStreams != 1 {
		t.Errorf("期待総ストリーム数: 1, 実際: %d", metrics.TotalStreams)
	}

	if metrics.ActiveStreams != 1 {
		t.Errorf("期待アクティブストリーム数: 1, 実際: %d", metrics.ActiveStreams)
	}
}

// TestProcessChunk はチャンク処理をテストする
func TestProcessChunk(t *testing.T) {
	processor := NewProcessor()
	
	// ストリームを開始
	stream := processor.StartStream("test", "test-model", nil)
	if stream == nil {
		t.Fatal("ストリームの開始に失敗")
	}

	var buffer strings.Builder

	t.Run("通常のチャンク処理", func(t *testing.T) {
		chunk := "テストチャンク"
		err := processor.processChunk(chunk, &buffer)
		if err != nil {
			t.Fatalf("チャンク処理エラー: %v", err)
		}

		if buffer.String() != chunk {
			t.Errorf("期待バッファ内容: %s, 実際: %s", chunk, buffer.String())
		}

		// ストリーム状態を確認
		currentStream := processor.GetCurrentStream()
		if currentStream.ChunkCount != 1 {
			t.Errorf("期待チャンク数: 1, 実際: %d", currentStream.ChunkCount)
		}
	})

	t.Run("JSONストリーミング形式の処理", func(t *testing.T) {
		processor.currentStream.ChunkCount = 0 // リセット
		buffer.Reset()

		jsonChunk := `data: {"choices":[{"delta":{"content":"Hello"}}]}`
		err := processor.processChunk(jsonChunk, &buffer)
		if err != nil {
			t.Fatalf("JSONチャンク処理エラー: %v", err)
		}

		if buffer.String() != "Hello" {
			t.Errorf("期待抽出内容: Hello, 実際: %s", buffer.String())
		}
	})
}

// TestFlushContent はコンテンツフラッシュをテストする
func TestFlushContent(t *testing.T) {
	processor := NewProcessor()
	var buffer strings.Builder

	content := "テスト内容"
	err := processor.flushContent(&buffer, content)
	if err != nil {
		t.Fatalf("フラッシュエラー: %v", err)
	}

	if buffer.String() != content {
		t.Errorf("期待内容: %s, 実際: %s", content, buffer.String())
	}

	// 空内容のテスト
	buffer.Reset()
	err = processor.flushContent(&buffer, "")
	if err != nil {
		t.Fatalf("空内容フラッシュエラー: %v", err)
	}

	if buffer.String() != "" {
		t.Errorf("期待内容: 空文字, 実際: %s", buffer.String())
	}
}

// TestProcessLLMStream はLLMストリーム処理をテストする
func TestProcessLLMStream(t *testing.T) {
	processor := NewProcessor()
	
	// ストリームを開始
	stream := processor.StartStream("test", "test-model", nil)
	if stream == nil {
		t.Fatal("ストリームの開始に失敗")
	}

	t.Run("正常なストリーム処理", func(t *testing.T) {
		// テストデータを準備
		input := "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\ndata: {\"choices\":[{\"delta\":{\"content\":\" World\"}}]}\ndata: [DONE]\n"
		reader := strings.NewReader(input)
		var output strings.Builder

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := processor.ProcessLLMStream(ctx, reader, &output)
		if err != nil {
			t.Fatalf("ストリーム処理エラー: %v", err)
		}

		// 出力内容を確認
		if !strings.Contains(output.String(), "Hello World") {
			t.Errorf("期待出力に 'Hello World' が含まれていません: %s", output.String())
		}
	})

	t.Run("コンテキストキャンセル", func(t *testing.T) {
		// 新しいストリームを開始
		processor.StartStream("test-cancel", "test-model", nil)
		
		input := "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n"
		reader := strings.NewReader(input)
		var output strings.Builder

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 即座にキャンセル

		err := processor.ProcessLLMStream(ctx, reader, &output)
		if err == nil {
			t.Error("キャンセルされたコンテキストでエラーが発生しませんでした")
		}
	})
}

// TestStreamStatusTransitions はストリームステータス遷移をテストする
func TestStreamStatusTransitions(t *testing.T) {
	processor := NewProcessor()
	
	// ストリームを開始
	stream := processor.StartStream("test", "test-model", nil)
	if stream.Status != StreamStatusStarting {
		t.Errorf("期待初期ステータス: %s, 実際: %s", StreamStatusStarting, stream.Status)
	}

	// 完了処理をテスト
	processor.handleStreamComplete()
	
	// メトリクスが更新されたことを確認
	metrics := processor.GetMetrics()
	if metrics.ActiveStreams != 0 {
		t.Errorf("完了後のアクティブストリーム数: %d", metrics.ActiveStreams)
	}

	// 現在のストリームがnilになったことを確認
	if processor.GetCurrentStream() != nil {
		t.Error("完了後に現在のストリームがnilになっていません")
	}
}

// TestStreamError はストリームエラー処理をテストする
func TestStreamError(t *testing.T) {
	processor := NewProcessor()
	
	// ストリームを開始
	processor.StartStream("test", "test-model", nil)

	// エラー処理をテスト
	testError := io.EOF
	processor.handleStreamError(testError)

	// メトリクスが更新されたことを確認
	metrics := processor.GetMetrics()
	if metrics.ErrorCount != 1 {
		t.Errorf("期待エラー数: 1, 実際: %d", metrics.ErrorCount)
	}

	if metrics.ActiveStreams != 0 {
		t.Errorf("エラー後のアクティブストリーム数: %d", metrics.ActiveStreams)
	}
}

// TestStreamCancel はストリームキャンセル処理をテストする
func TestStreamCancel(t *testing.T) {
	processor := NewProcessor()
	
	// ストリームを開始
	processor.StartStream("test", "test-model", nil)

	// キャンセル処理をテスト
	processor.CancelStream()

	// メトリクスが更新されたことを確認
	metrics := processor.GetMetrics()
	if metrics.ActiveStreams != 0 {
		t.Errorf("キャンセル後のアクティブストリーム数: %d", metrics.ActiveStreams)
	}

	// 現在のストリームがnilになったことを確認
	if processor.GetCurrentStream() != nil {
		t.Error("キャンセル後に現在のストリームがnilになっていません")
	}
}

// TestUpdateConfig は設定更新をテストする
func TestUpdateConfig(t *testing.T) {
	processor := NewProcessor()

	newBufferSize := 8192
	newFlushInterval := 100 * time.Millisecond

	processor.UpdateConfig(newBufferSize, newFlushInterval)

	if processor.bufferSize != newBufferSize {
		t.Errorf("期待バッファサイズ: %d, 実際: %d", newBufferSize, processor.bufferSize)
	}

	if processor.flushInterval != newFlushInterval {
		t.Errorf("期待フラッシュ間隔: %v, 実際: %v", newFlushInterval, processor.flushInterval)
	}

	// 無効な値でのテスト
	processor.UpdateConfig(0, 0)
	
	// 値が変更されないことを確認
	if processor.bufferSize != newBufferSize {
		t.Error("無効な値でバッファサイズが変更されました")
	}

	if processor.flushInterval != newFlushInterval {
		t.Error("無効な値でフラッシュ間隔が変更されました")
	}
}

// TestResetMetrics はメトリクスリセットをテストする
func TestResetMetrics(t *testing.T) {
	processor := NewProcessor()

	// メトリクスを設定
	processor.StartStream("test", "test-model", nil)
	processor.handleStreamComplete()

	// リセット前の確認
	metrics := processor.GetMetrics()
	if metrics.TotalStreams == 0 {
		t.Error("リセット前のメトリクスが0です")
	}

	// メトリクスをリセット
	processor.ResetMetrics()

	// リセット後の確認
	resetMetrics := processor.GetMetrics()
	if resetMetrics.TotalStreams != 0 {
		t.Errorf("リセット後の総ストリーム数が0ではありません: %d", resetMetrics.TotalStreams)
	}

	if resetMetrics.ActiveStreams != 0 {
		t.Errorf("リセット後のアクティブストリーム数が0ではありません: %d", resetMetrics.ActiveStreams)
	}
}

// TestGetCurrentStream は現在ストリーム取得をテストする
func TestGetCurrentStream(t *testing.T) {
	processor := NewProcessor()

	// ストリームがない状態
	current := processor.GetCurrentStream()
	if current != nil {
		t.Error("ストリームがない状態でnilが返されませんでした")
	}

	// ストリームを開始
	stream := processor.StartStream("test", "test-model", nil)
	
	// 現在のストリームを取得
	current = processor.GetCurrentStream()
	if current == nil {
		t.Fatal("現在のストリームがnilです")
	}

	if current.ID != stream.ID {
		t.Errorf("期待ストリームID: %s, 実際: %s", stream.ID, current.ID)
	}

	// コピーが返されることを確認（元のストリームを変更しても影響しない）
	current.TokenCount = 999
	actualCurrent := processor.GetCurrentStream()
	if actualCurrent.TokenCount == 999 {
		t.Error("ストリームのコピーではなく参照が返されています")
	}
}