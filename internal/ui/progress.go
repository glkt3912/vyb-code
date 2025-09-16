package ui

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ========== ClaudeCode風 ProgressIndicator ==========

// ProgressIndicator はClaudeCode風の進捗表示
type ProgressIndicator struct {
	startTime   time.Time
	message     string
	tokensSent  int
	tokensRecv  int
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	isActive    bool
	interrupted bool
	animation   []rune
	animIndex   int
}

// NewProgressIndicator は新しい進捗インジケーターを作成
func NewProgressIndicator(message string, tokensSent int) *ProgressIndicator {
	ctx, cancel := context.WithCancel(context.Background())

	return &ProgressIndicator{
		startTime:   time.Now(),
		message:     message,
		tokensSent:  tokensSent,
		tokensRecv:  0,
		ctx:         ctx,
		cancel:      cancel,
		isActive:    false,
		interrupted: false,
		animation:   []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}, // スピナー文字
		animIndex:   0,
	}
}

// Start は進捗表示を開始
func (p *ProgressIndicator) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isActive {
		return
	}

	p.isActive = true

	// 新しい行を開始して進捗表示専用の行を作成
	fmt.Fprint(os.Stderr, "\n")

	// Escキーでの中断監視を開始
	go p.watchForInterrupt()

	// 進捗表示ループを開始
	go p.displayLoop()
}

// Stop は進捗表示を停止
func (p *ProgressIndicator) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isActive {
		return
	}

	p.isActive = false
	p.cancel()

	// 進捗行を完全にクリア
	fmt.Fprint(os.Stderr, "\r\033[2K")
	os.Stderr.Sync()
}

// UpdateTokens は受信トークン数を更新
func (p *ProgressIndicator) UpdateTokens(received int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tokensRecv = received
}

// IsInterrupted は中断されたかを返す
func (p *ProgressIndicator) IsInterrupted() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.interrupted
}

// GetContext は中断可能なコンテキストを返す
func (p *ProgressIndicator) GetContext() context.Context {
	return p.ctx
}

// displayLoop は進捗表示のメインループ
func (p *ProgressIndicator) displayLoop() {
	ticker := time.NewTicker(200 * time.Millisecond) // 200msごとに更新（より安定）
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.mu.RLock()
			if !p.isActive {
				p.mu.RUnlock()
				return
			}

			// 進捗情報を取得
			elapsed := time.Since(p.startTime)
			spinner := p.animation[p.animIndex%len(p.animation)]

			// 表示文字列を構築
			progressStr := p.buildProgressString(spinner, elapsed)
			p.mu.RUnlock()

			// 進捗を表示（stderrに出力して分離）
			fmt.Fprintf(os.Stderr, "\r\033[2K%s", progressStr)
			// フラッシュを強制
			os.Stderr.Sync()

			p.mu.Lock()
			p.animIndex++
			p.mu.Unlock()
		}
	}
}

// buildProgressString は進捗表示文字列を構築
func (p *ProgressIndicator) buildProgressString(spinner rune, elapsed time.Duration) string {
	// ClaudeCode風のフォーマット: ✢ Creating… (31s · ↑ 673 tokens · esc to interrupt)

	// 経過時間をフォーマット
	seconds := int(elapsed.Seconds())
	timeStr := fmt.Sprintf("%ds", seconds)

	// トークン情報
	var tokenStr string
	if p.tokensRecv > 0 {
		tokenStr = fmt.Sprintf("↑ %d ↓ %d tokens", p.tokensSent, p.tokensRecv)
	} else {
		tokenStr = fmt.Sprintf("↑ %d tokens", p.tokensSent)
	}

	// メッセージの省略処理
	message := p.message
	if len(message) > 20 {
		message = message[:17] + "…"
	}

	// 全体の文字列を構築
	progressStr := fmt.Sprintf("\033[90m%c %s (%s · %s · ctrl+c to interrupt)\033[0m",
		spinner, message, timeStr, tokenStr)

	return progressStr
}

// watchForInterrupt はCtrl+Cでの中断を監視（より安全な実装）
func (p *ProgressIndicator) watchForInterrupt() {
	// シグナル監視のみに変更（入力監視は他のシステムと競合するため削除）
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-p.ctx.Done():
		signal.Stop(sigCh)
		return
	case <-sigCh:
		// Ctrl+Cでの中断
		p.mu.Lock()
		p.interrupted = true
		p.mu.Unlock()
		p.cancel()
		signal.Stop(sigCh)
		return
	}
}

// CompleteWithResult は完了時の結果表示
func (p *ProgressIndicator) CompleteWithResult(success bool, finalMessage string) {
	p.Stop()

	elapsed := time.Since(p.startTime)
	seconds := elapsed.Round(time.Millisecond)

	var icon string
	var color string
	if success {
		icon = "✓"
		color = "\033[32m" // 緑
	} else {
		icon = "✗"
		color = "\033[31m" // 赤
	}

	// 完了メッセージを進捗行に上書きして表示
	fmt.Fprintf(os.Stderr, "\r\033[2K%s%s %s\033[0m (%v)\n",
		color, icon, finalMessage, seconds)
	os.Stderr.Sync()
}

// ShowStreamingProgress はストリーミング応答用の進捗表示
func ShowStreamingProgress(message string, tokensSent int, onInterrupt func()) *ProgressIndicator {
	indicator := NewProgressIndicator(message, tokensSent)

	// 中断時のコールバック設定
	go func() {
		<-indicator.GetContext().Done()
		if indicator.IsInterrupted() && onInterrupt != nil {
			onInterrupt()
		}
	}()

	indicator.Start()
	return indicator
}
