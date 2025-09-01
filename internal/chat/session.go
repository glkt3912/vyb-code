package chat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/glkt/vyb-code/internal/llm"
)

// 会話セッションを管理する構造体
type Session struct {
	provider llm.Provider      // LLMプロバイダー
	messages []llm.ChatMessage // 会話履歴
	model    string            // 使用するモデル名
}

// 新しい会話セッションを作成
func NewSession(provider llm.Provider, model string) *Session {
	return &Session{
		provider: provider,
		messages: make([]llm.ChatMessage, 0),
		model:    model,
	}
}

// 対話ループを開始する
func (s *Session) StartInteractive() error {
	fmt.Println("対話モードを開始しました。'exit'または'quit'で終了できます。")

	// 標準入力からの読み込み用スキャナー
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")

		// ユーザー入力を読み込み
		if !scanner.Scan() {
			break // EOF または Ctrl+C
		}

		input := strings.TrimSpace(scanner.Text())

		// 終了コマンドチェック
		if input == "exit" || input == "quit" {
			fmt.Println("対話を終了します。")
			break
		}

		// 空入力はスキップ
		if input == "" {
			continue
		}

		// ユーザーメッセージを履歴に追加
		s.messages = append(s.messages, llm.ChatMessage{
			Role:    "user",
			Content: input,
		})

		// LLMに送信してレスポンス取得
		if err := s.sendToLLM(); err != nil {
			fmt.Printf("エラー: %v\n", err)
			continue
		}
	}

	// スキャナーのエラーチェック
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("input reading error: %w", err)
	}

	return nil
}

// 単発クエリを処理する
func (s *Session) ProcessQuery(query string) error {
	// クエリをメッセージ履歴に追加
	s.messages = append(s.messages, llm.ChatMessage{
		Role:    "user",
		Content: query,
	})

	// LLMに送信してレスポンス取得
	return s.sendToLLM()
}

// LLMにメッセージを送信してレスポンスを処理
func (s *Session) sendToLLM() error {
	// チャットリクエストを作成
	req := llm.ChatRequest{
		Model:    s.model,
		Messages: s.messages,
		Stream:   false, // 現在はストリーミング無効
	}

	// LLMプロバイダーにリクエスト送信
	ctx := context.Background()
	resp, err := s.provider.Chat(ctx, req)
	if err != nil {
		return fmt.Errorf("LLM request failed: %w", err)
	}

	// レスポンスメッセージを履歴に追加
	s.messages = append(s.messages, resp.Message)

	// レスポンスを表示
	fmt.Printf("🎵 %s\n", resp.Message.Content)

	return nil
}

// 会話履歴をクリアする
func (s *Session) ClearHistory() {
	s.messages = make([]llm.ChatMessage, 0)
}

// 会話履歴の件数を取得
func (s *Session) GetMessageCount() int {
	return len(s.messages)
}
