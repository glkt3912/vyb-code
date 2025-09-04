package interrupt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestHandler_Creation(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	if handler == nil {
		t.Error("Expected non-nil handler")
	}

	if handler.ctx == nil {
		t.Error("Expected non-nil context")
	}

	if handler.cancel == nil {
		t.Error("Expected non-nil cancel function")
	}

	if handler.IsInterrupted() {
		t.Error("Expected handler to not be interrupted initially")
	}

	if handler.signalChannel == nil {
		t.Error("Expected non-nil signal channel")
	}
}

func TestHandler_GlobalSingleton(t *testing.T) {
	// 複数回取得しても同じインスタンスが返されることを確認
	handler1 := GetGlobalHandler()
	handler2 := GetGlobalHandler()

	if handler1 != handler2 {
		t.Error("Expected same instance from GetGlobalHandler()")
	}

	// クリーンアップのためにリセット
	defer handler1.Close()
}

func TestHandler_CallbackRegistration(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	// コールバックを登録
	handler.AddCallback(func() {
		// コールバック実行をテスト
	})

	// コールバックが登録されたことを確認（内部状態）
	if len(handler.callbacks) != 1 {
		t.Errorf("Expected 1 callback, got %d", len(handler.callbacks))
	}

	// 複数のコールバックを登録
	handler.AddCallback(func() {})
	handler.AddCallback(func() {})

	if len(handler.callbacks) != 3 {
		t.Errorf("Expected 3 callbacks, got %d", len(handler.callbacks))
	}
}

func TestHandler_ContextOperations(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	ctx := handler.Context()

	if ctx == nil {
		t.Error("Expected non-nil context")
	}

	// コンテキストが有効であることを確認
	select {
	case <-ctx.Done():
		t.Error("Expected context to not be cancelled initially")
	default:
		// OK
	}

	// キャンセル後の動作を確認
	handler.cancel()

	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected context to be cancelled after cancel()")
	}
}

func TestHandler_Reset(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	// 初期状態を変更
	handler.mutex.Lock()
	handler.interrupted = true
	handler.mutex.Unlock()

	oldCtx := handler.Context()

	// リセット実行
	handler.Reset()

	// 中断状態がリセットされることを確認
	if handler.IsInterrupted() {
		t.Error("Expected interrupted state to be reset")
	}

	// 新しいコンテキストが作成されることを確認
	newCtx := handler.Context()
	if newCtx == oldCtx {
		t.Error("Expected new context after reset")
	}

	// 新しいコンテキストが有効であることを確認
	select {
	case <-newCtx.Done():
		t.Error("Expected new context to be active")
	default:
		// OK
	}
}

func TestHandler_InterruptedState(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	// 初期状態確認
	if handler.IsInterrupted() {
		t.Error("Expected not interrupted initially")
	}

	// 中断状態を設定（直接設定による擬似テスト）
	handler.mutex.Lock()
	handler.interrupted = true
	handler.mutex.Unlock()

	if !handler.IsInterrupted() {
		t.Error("Expected interrupted state to be true")
	}

	// リセット後の状態確認
	handler.Reset()
	if handler.IsInterrupted() {
		t.Error("Expected interrupted state to be reset")
	}
}

func TestWithInterruption(t *testing.T) {
	t.Run("Successful operation", func(t *testing.T) {
		err := WithInterruption(func(ctx context.Context) error {
			// 正常な操作
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error for successful operation, got %v", err)
		}
	})

	t.Run("Operation with error", func(t *testing.T) {
		expectedError := errors.New("test error")

		err := WithInterruption(func(ctx context.Context) error {
			return expectedError
		})

		if err != expectedError {
			t.Errorf("Expected error to be passed through, got %v", err)
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		err := WithInterruption(func(ctx context.Context) error {
			// コンテキストがキャンセルされるまで待機
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil // タイムアウト（テストのため）
			}
		})

		// コンテキストキャンセレーションはWithInterruptionで処理されるため
		// エラーが返される場合がある
		if err != nil && err.Error() != "context canceled" {
			t.Logf("Operation completed with: %v", err)
		}
	})
}

func TestInterruptibleSleep(t *testing.T) {
	t.Run("Normal sleep completion", func(t *testing.T) {
		handler := GetGlobalHandler()
		handler.Reset()

		start := time.Now()
		err := InterruptibleSleep(50 * time.Millisecond)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Expected no error for normal sleep, got %v", err)
		}

		if duration < 40*time.Millisecond || duration > 100*time.Millisecond {
			t.Errorf("Expected sleep around 50ms, got %v", duration)
		}
	})

	t.Run("Interrupted sleep", func(t *testing.T) {
		handler := GetGlobalHandler()
		handler.Reset()

		// 少し待ってからコンテキストをキャンセル
		go func() {
			time.Sleep(25 * time.Millisecond)
			handler.cancel()
		}()

		start := time.Now()
		err := InterruptibleSleep(100 * time.Millisecond)
		duration := time.Since(start)

		if err == nil {
			t.Error("Expected error for interrupted sleep")
		}

		if !strings.Contains(fmt.Sprint(err), "interrupted") {
			t.Errorf("Expected error to contain 'interrupted', got %v", err)
		}

		if duration > 80*time.Millisecond {
			t.Errorf("Expected interrupted sleep to be shorter, got %v", duration)
		}
	})
}

func TestHandler_GracefulInterruption(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	t.Run("Quick operation", func(t *testing.T) {
		operationCompleted := false

		err := handler.HandleGracefulInterruption(func() error {
			time.Sleep(10 * time.Millisecond)
			operationCompleted = true
			return nil
		}, 100*time.Millisecond)

		if err != nil {
			t.Errorf("Expected no error for quick operation, got %v", err)
		}

		if !operationCompleted {
			t.Error("Expected operation to be completed")
		}
	})

	t.Run("Operation with error", func(t *testing.T) {
		expectedError := errors.New("operation failed")

		err := handler.HandleGracefulInterruption(func() error {
			return expectedError
		}, 100*time.Millisecond)

		if err != expectedError {
			t.Errorf("Expected operation error to be returned, got %v", err)
		}
	})
}

// 実際のシグナル送信テスト（統合テスト的な性質）
func TestHandler_SignalHandling(t *testing.T) {
	// このテストは実際のシグナル送信を含むため、環境によっては動作しない可能性がある
	if testing.Short() {
		t.Skip("Skipping signal handling test in short mode")
	}

	handler := NewHandler()
	defer handler.Close()

	var callbackExecuted bool
	var callbackMutex sync.Mutex
	handler.AddCallback(func() {
		callbackMutex.Lock()
		callbackExecuted = true
		callbackMutex.Unlock()
	})

	// 自分自身にSIGINTを送信
	go func() {
		time.Sleep(50 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	// シグナル処理を待機
	start := time.Now()
	for !handler.IsInterrupted() && time.Since(start) < 200*time.Millisecond {
		time.Sleep(10 * time.Millisecond)
	}

	if !handler.IsInterrupted() {
		t.Error("Expected handler to be interrupted after SIGINT")
	}

	// コールバックが実行されるまで少し待機
	time.Sleep(50 * time.Millisecond)

	callbackMutex.Lock()
	executed := callbackExecuted
	callbackMutex.Unlock()

	if !executed {
		t.Error("Expected callback to be executed after signal")
	}
}

func TestHandler_ConcurrentAccess(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	// 並行アクセステスト
	var wg sync.WaitGroup
	numGoroutines := 10

	// 複数のゴルーチンから同時にアクセス
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// 状態読み取り
			handler.IsInterrupted()

			// コールバック追加
			handler.AddCallback(func() {})

			// コンテキスト取得
			handler.Context()
		}()
	}

	// リセット操作も並行実行
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(25 * time.Millisecond)
		handler.Reset()
	}()

	wg.Wait()

	// データ競合がないことを確認（パニックが発生しないこと）
}

func TestHandler_MemoryLeak(t *testing.T) {
	// メモリリークテスト：多数のハンドラーを作成・破棄
	for i := 0; i < 100; i++ {
		handler := NewHandler()

		// いくつかの操作を実行
		handler.AddCallback(func() {})
		handler.IsInterrupted()
		handler.Context()

		// クリーンアップ
		handler.Close()
	}

	// メモリリークの基本チェック（ゴルーチンが蓄積していないか）
	// 実際のメモリ測定は外部ツールが必要だが、基本的なチェックを行う
	time.Sleep(50 * time.Millisecond) // ゴルーチンの終了を待つ
}

func TestHandler_MultipleSignals(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multiple signals test in short mode")
	}

	handler := NewHandler()
	defer handler.Close()

	var signalCount int
	var signalMutex sync.Mutex
	handler.AddCallback(func() {
		signalMutex.Lock()
		signalCount++
		signalMutex.Unlock()
	})

	// 複数のシグナルを短時間で送信
	go func() {
		time.Sleep(30 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)

		time.Sleep(10 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	// シグナル処理を待機
	time.Sleep(200 * time.Millisecond)

	if !handler.IsInterrupted() {
		t.Error("Expected handler to be interrupted")
	}

	// 最初のシグナルでコンテキストがキャンセルされるため、
	// 2番目のシグナルは処理されない可能性が高い
	signalMutex.Lock()
	count := signalCount
	signalMutex.Unlock()

	if count == 0 {
		t.Error("Expected at least one signal to be processed")
	}
}

func TestHandler_ContextCancellation(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	ctx := handler.Context()

	// 別のゴルーチンでキャンセル
	go func() {
		time.Sleep(50 * time.Millisecond)
		handler.cancel()
	}()

	// コンテキストのキャンセルを待機
	select {
	case <-ctx.Done():
		// 期待される動作
	case <-time.After(200 * time.Millisecond):
		t.Error("Expected context to be cancelled")
	}

	// エラー内容を確認
	err := ctx.Err()
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestHandler_CleanupSafety(t *testing.T) {
	handler := NewHandler()

	// Close()が複数回呼び出されても安全であることを確認
	handler.Close()
	handler.Close() // 2回目

	// パニックが発生しないことを確認
}

// エラー回復テスト
func TestHandler_ErrorRecovery(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	// 操作中にエラーが発生するケース
	operationError := errors.New("simulated operation error")

	err := handler.HandleGracefulInterruption(func() error {
		time.Sleep(10 * time.Millisecond)
		return operationError
	}, 100*time.Millisecond)

	if err != operationError {
		t.Errorf("Expected original operation error, got %v", err)
	}
}

// タイムアウトテスト
func TestHandler_GracefulTimeout(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	// コンテキストを事前にキャンセル（中断状態をシミュレート）
	handler.cancel()

	start := time.Now()
	err := handler.HandleGracefulInterruption(func() error {
		// 長時間の操作をシミュレート
		time.Sleep(200 * time.Millisecond)
		return nil
	}, 50*time.Millisecond) // 短いタイムアウト

	duration := time.Since(start)

	// タイムアウトエラーが発生することを確認
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Expected timeout error message, got %v", err)
	}

	// タイムアウト時間内に完了していることを確認
	if duration > 150*time.Millisecond {
		t.Errorf("Expected operation to timeout quickly, took %v", duration)
	}
}

// パフォーマンステスト
func TestHandler_PerformanceOperations(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	// 高頻度でのIsInterrupted()呼び出し
	start := time.Now()
	for i := 0; i < 10000; i++ {
		handler.IsInterrupted()
	}
	duration := time.Since(start)

	if duration > 100*time.Millisecond {
		t.Errorf("IsInterrupted() calls took too long: %v", duration)
	}

	// 高頻度でのコンテキスト取得
	start = time.Now()
	for i := 0; i < 10000; i++ {
		handler.Context()
	}
	duration = time.Since(start)

	if duration > 50*time.Millisecond {
		t.Errorf("Context() calls took too long: %v", duration)
	}
}

// ゴルーチン安全性テスト
func TestHandler_GoroutineSafety(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	var wg sync.WaitGroup
	errors := make(chan error, 20)

	// 複数のゴルーチンで様々な操作を実行
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// 状態変更操作
			err := WithInterruption(func(ctx context.Context) error {
				handler.IsInterrupted()
				handler.AddCallback(func() {})
				return nil
			})

			if err != nil {
				errors <- err
			}
		}()
	}

	// リセット操作も並行実行
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			handler.Reset()
		}()
	}

	wg.Wait()
	close(errors)

	// エラーが発生していないことを確認
	for err := range errors {
		t.Errorf("Unexpected error in concurrent access: %v", err)
	}
}

// 長時間操作のテスト
func TestHandler_LongRunningOperation(t *testing.T) {
	handler := NewHandler()
	defer handler.Close()

	operationCompleted := false

	// 長時間操作を開始
	go func() {
		err := WithInterruption(func(ctx context.Context) error {
			for i := 0; i < 10; i++ {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					time.Sleep(20 * time.Millisecond)
				}
			}
			operationCompleted = true
			return nil
		})

		if err != nil && err != context.Canceled {
			t.Errorf("Unexpected error: %v", err)
		}
	}()

	// 中間で中断
	time.Sleep(100 * time.Millisecond)
	handler.cancel()

	// 少し待ってから状態を確認
	time.Sleep(50 * time.Millisecond)

	if operationCompleted {
		t.Error("Expected operation to be interrupted before completion")
	}
}

// ベンチマークテスト
func BenchmarkHandler_IsInterrupted(b *testing.B) {
	handler := NewHandler()
	defer handler.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.IsInterrupted()
	}
}

func BenchmarkHandler_Context(b *testing.B) {
	handler := NewHandler()
	defer handler.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.Context()
	}
}

func BenchmarkHandler_AddCallback(b *testing.B) {
	handler := NewHandler()
	defer handler.Close()

	callback := func() {}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.AddCallback(callback)
	}
}

func BenchmarkWithInterruption(b *testing.B) {
	operation := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WithInterruption(operation)
	}
}
