package interrupt

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// 中断ハンドラー
type Handler struct {
	ctx           context.Context
	cancel        context.CancelFunc
	interrupted   bool
	mutex         sync.Mutex
	callbacks     []func()
	signalChannel chan os.Signal
}

// グローバルハンドラー（プロセス全体で共有）
var globalHandler *Handler
var globalMutex sync.Mutex

// 新しい中断ハンドラーを作成
func NewHandler() *Handler {
	ctx, cancel := context.WithCancel(context.Background())

	handler := &Handler{
		ctx:           ctx,
		cancel:        cancel,
		interrupted:   false,
		callbacks:     make([]func(), 0),
		signalChannel: make(chan os.Signal, 1),
	}

	// シグナル受信を設定
	signal.Notify(handler.signalChannel, syscall.SIGINT, syscall.SIGTERM)

	// シグナル処理用ゴルーチンを開始
	go handler.handleSignals()

	return handler
}

// グローバルハンドラーを取得（シングルトンパターン）
func GetGlobalHandler() *Handler {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if globalHandler == nil {
		globalHandler = NewHandler()
	}

	return globalHandler
}

// シグナル処理
func (h *Handler) handleSignals() {
	// signalChannelの安全な読み取り
	h.mutex.Lock()
	sigChan := h.signalChannel
	h.mutex.Unlock()

	if sigChan == nil {
		return
	}

	select {
	case sig := <-sigChan:
		h.mutex.Lock()
		h.interrupted = true
		h.mutex.Unlock()

		fmt.Printf("\n\033[90m[中断シグナルを受信: %v]\033[0m\n", sig)

		// 登録されたコールバックを実行
		h.mutex.Lock()
		callbacks := make([]func(), len(h.callbacks))
		copy(callbacks, h.callbacks)
		h.mutex.Unlock()

		for _, callback := range callbacks {
			callback()
		}

		// コンテキストをキャンセル
		if h.cancel != nil {
			h.cancel()
		}

	case <-h.ctx.Done():
		// コンテキストがキャンセルされた場合
		return
	}
}

// 中断状態をチェック
func (h *Handler) IsInterrupted() bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.interrupted
}

// コンテキストを取得
func (h *Handler) Context() context.Context {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.ctx
}

// 中断コールバックを追加
func (h *Handler) AddCallback(callback func()) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.callbacks = append(h.callbacks, callback)
}

// 中断状態をリセット
func (h *Handler) Reset() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.interrupted = false

	// 新しいコンテキストを作成
	if h.cancel != nil {
		h.cancel()
	}
	h.ctx, h.cancel = context.WithCancel(context.Background())

	// シグナルチャネルをクリア
	select {
	case <-h.signalChannel:
		// バッファリングされたシグナルを削除
	default:
		// バッファが空の場合は何もしない
	}

	// 新しいシグナル処理を開始
	go h.handleSignals()
}

// クリーンアップ
func (h *Handler) Close() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.cancel != nil {
		h.cancel()
		h.cancel = nil
	}

	if h.signalChannel != nil {
		signal.Stop(h.signalChannel)
		// チャネルがまだ開いている場合のみ閉じる
		select {
		case <-h.signalChannel:
			// すでに閉じられている
		default:
			close(h.signalChannel)
		}
		h.signalChannel = nil
	}
}

// 中断可能な操作のヘルパー関数
func WithInterruption(operation func(context.Context) error) error {
	handler := GetGlobalHandler()

	// 操作前に中断状態をリセット
	handler.Reset()

	return operation(handler.Context())
}

// 中断可能なスリープ
func InterruptibleSleep(duration time.Duration) error {
	handler := GetGlobalHandler()

	select {
	case <-time.After(duration):
		return nil
	case <-handler.Context().Done():
		return fmt.Errorf("sleep interrupted")
	}
}

// 段階的中断（警告 → 強制終了）
func (h *Handler) HandleGracefulInterruption(operation func() error, timeout time.Duration) error {
	done := make(chan error, 1)

	// 操作を別ゴルーチンで実行
	go func() {
		done <- operation()
	}()

	select {
	case err := <-done:
		return err
	case <-h.Context().Done():
		fmt.Printf("\n\033[33m⚠️  中断中... (操作完了まで最大 %v 待機)\033[0m\n", timeout)

		// タイムアウト付きで操作完了を待機
		select {
		case err := <-done:
			fmt.Printf("\033[32m✓ 操作が正常に完了しました\033[0m\n")
			return err
		case <-time.After(timeout):
			fmt.Printf("\033[31m✗ タイムアウトしました。操作を強制終了します\033[0m\n")
			return fmt.Errorf("operation timed out after interruption")
		}
	}
}
