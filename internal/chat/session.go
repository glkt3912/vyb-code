package chat

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/llm"
	"github.com/glkt/vyb-code/internal/mcp"
)

// 会話セッションを管理する構造体
type Session struct {
	provider      llm.Provider      // LLMプロバイダー
	messages      []llm.ChatMessage // 会話履歴
	model         string            // 使用するモデル名
	mcpManager    *mcp.Manager      // MCPマネージャー
	workDir       string            // 作業ディレクトリ
	contextFiles  []string          // コンテキストファイル一覧
	projectInfo   *ProjectContext   // プロジェクト情報
}

// プロジェクトコンテキスト情報
type ProjectContext struct {
	Language     string            `json:"language"`
	Framework    string            `json:"framework"`
	Dependencies []string          `json:"dependencies"`
	Structure    map[string]string `json:"structure"`
	GitBranch    string            `json:"git_branch"`
	GitStatus    string            `json:"git_status"`
}

// 新しい会話セッションを作成
func NewSession(provider llm.Provider, model string) *Session {
	workDir, _ := os.Getwd()
	
	session := &Session{
		provider:     provider,
		messages:     make([]llm.ChatMessage, 0),
		model:        model,
		mcpManager:   mcp.NewManager(),
		workDir:      workDir,
		contextFiles: make([]string, 0),
	}
	
	// プロジェクト情報を初期化
	session.initializeProjectContext()
	
	return session
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

// MCPサーバーに接続
func (s *Session) ConnectMCPServer(name string, config mcp.ClientConfig) error {
	return s.mcpManager.ConnectServer(name, config)
}

// MCPサーバーから切断
func (s *Session) DisconnectMCPServer(name string) error {
	return s.mcpManager.DisconnectServer(name)
}

// 利用可能なMCPツールを取得
func (s *Session) GetMCPTools() map[string][]mcp.Tool {
	return s.mcpManager.GetAllTools()
}

// MCPツールを実行
func (s *Session) CallMCPTool(serverName, toolName string, arguments map[string]interface{}) (*mcp.ToolResult, error) {
	return s.mcpManager.CallTool(serverName, toolName, arguments)
}

// セッション終了時にMCP接続をクリーンアップ
func (s *Session) Close() error {
	return s.mcpManager.DisconnectAll()
}

// Claude Code風拡張ターミナルモードを開始
func (s *Session) StartEnhancedTerminal() error {
	// カラフルな起動メッセージ
	s.printWelcomeMessage()

	// 高度な入力読み込み用
	reader := bufio.NewReader(os.Stdin)

	for {
		// Claude風カラープロンプト
		s.printColoredPrompt()

		// マルチライン入力サポート
		input, err := s.readMultilineInput(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("入力エラー: %v\n", err)
			continue
		}

		// 終了コマンドチェック
		trimmed := strings.TrimSpace(input)
		if trimmed == "exit" || trimmed == "quit" {
			fmt.Println("goodbye")
			break
		}

		// 空入力はスキップ
		if trimmed == "" {
			continue
		}

		// コンテキスト情報付きでユーザーメッセージを履歴に追加
		contextualInput := s.buildContextualPrompt(input)
		s.messages = append(s.messages, llm.ChatMessage{
			Role:    "user",
			Content: contextualInput,
		})

		// thinking状態表示を開始
		stopThinking := s.startThinkingAnimation()

		// Claude Code風ストリーミング応答で送信
		err = s.sendToLLMStreamingWithThinking(stopThinking)
		
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// レスポンス後に区切り線
		fmt.Println()
	}

	return nil
}

// Claude Code風インタラクティブ入力（日本語対応）
func (s *Session) readMultilineInput(reader *bufio.Reader) (string, error) {
	// 日本語入力（IME）対応のため、行ベース入力を使用
	var lines []string
	
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		// 改行を除去
		line = strings.TrimSuffix(line, "\n")

		// 最初の行の場合
		if len(lines) == 0 {
			// 空でない最初の行は即座に送信（Claude風）
			if line != "" {
				return line, nil
			}
			// 空行の場合は継続
			continue
		}

		// 複数行モードでの処理
		// 空行で送信
		if line == "" {
			return strings.Join(lines, "\n"), nil
		}
		
		lines = append(lines, line)
	}
}

// シンプル入力（Rawモード失敗時のフォールバック）
func (s *Session) readSimpleInput(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(line, "\n"), nil
}

// Claude Code風ストリーミング応答でLLMに送信（thinking制御付き）
func (s *Session) sendToLLMStreamingWithThinking(stopThinking func()) error {
	// リクエスト開始時間を記録
	startTime := time.Now()
	
	// チャットリクエストを作成
	req := llm.ChatRequest{
		Model:    s.model,
		Messages: s.messages,
		Stream:   false, // 既存API構造を活用
	}

	// LLMプロバイダーにリクエスト送信
	ctx := context.Background()
	resp, err := s.provider.Chat(ctx, req)
	
	// レスポンス受信後、thinking状態を停止
	stopThinking()
	
	if err != nil {
		return fmt.Errorf("LLM request failed: %w", err)
	}
	
	// レスポンス時間を計算
	duration := time.Since(startTime)
	
	// Claude Code風のクリーンなレスポンス表示
	content := resp.Message.Content
	
	// Markdown対応のレスポンス表示
	s.displayFormattedResponse(content)

	// 最終改行
	fmt.Println()
	
	// メタ情報表示（Claude Code風）
	s.displayMetaInfo(duration, len(content))

	// レスポンスメッセージを履歴に追加
	s.messages = append(s.messages, resp.Message)

	return nil
}

// thinking状態のアニメーション表示を開始
func (s *Session) startThinkingAnimation() func() {
	// アニメーション用の文字列
	frames := []string{"thinking", "thinking.", "thinking..", "thinking..."}
	frameIndex := 0
	
	// 停止チャネル
	stopCh := make(chan struct{})
	
	// アニメーションゴルーチンを開始
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-stopCh:
				// thinkingテキストを完全にクリアして改行位置に戻る
				fmt.Print("\r" + strings.Repeat(" ", 50) + "\r")
				return
			case <-ticker.C:
				// カーソル位置に戻してアニメーション表示
				fmt.Printf("\r%s", frames[frameIndex])
				frameIndex = (frameIndex + 1) % len(frames)
			}
		}
	}()
	
	// 停止関数を返す
	return func() {
		close(stopCh)
		time.Sleep(300 * time.Millisecond) // クリア処理の完了をしっかり待つ
		// 追加でクリア処理を確実に実行
		fmt.Print("\r" + strings.Repeat(" ", 50) + "\r")
	}
}

// プロジェクトコンテキストを初期化
func (s *Session) initializeProjectContext() {
	s.projectInfo = &ProjectContext{}
	
	// プロジェクト言語を検出
	s.projectInfo.Language = s.detectProjectLanguage()
	
	// Gitブランチを取得
	s.projectInfo.GitBranch = s.getCurrentGitBranch()
	
	// 依存関係を取得
	s.projectInfo.Dependencies = s.getProjectDependencies()
}

// プロジェクト言語を検出
func (s *Session) detectProjectLanguage() string {
	// go.modの存在確認
	if _, err := os.Stat("go.mod"); err == nil {
		return "Go"
	}
	// package.jsonの存在確認
	if _, err := os.Stat("package.json"); err == nil {
		return "JavaScript/TypeScript"
	}
	// requirements.txtやsetup.pyの確認
	if _, err := os.Stat("requirements.txt"); err == nil {
		return "Python"
	}
	// Cargo.tomlの確認
	if _, err := os.Stat("Cargo.toml"); err == nil {
		return "Rust"
	}
	return "Unknown"
}

// 現在のGitブランチを取得
func (s *Session) getCurrentGitBranch() string {
	// 簡易Git情報取得（実装簡素化）
	return "main"
}

// プロジェクト依存関係を取得
func (s *Session) getProjectDependencies() []string {
	deps := make([]string, 0)
	
	// Go依存関係
	if _, err := os.Stat("go.mod"); err == nil {
		deps = append(deps, "cobra", "bubbletea")
	}
	
	return deps
}

// LLMリクエストにコンテキスト情報を追加
func (s *Session) buildContextualPrompt(userInput string) string {
	var contextBuilder strings.Builder
	
	// プロジェクト情報をコンテキストに追加
	if s.projectInfo != nil {
		contextBuilder.WriteString(fmt.Sprintf("# Project Context\n"))
		contextBuilder.WriteString(fmt.Sprintf("Language: %s\n", s.projectInfo.Language))
		contextBuilder.WriteString(fmt.Sprintf("Working Directory: %s\n", s.workDir))
		if s.projectInfo.GitBranch != "" {
			contextBuilder.WriteString(fmt.Sprintf("Git Branch: %s\n", s.projectInfo.GitBranch))
		}
		if len(s.projectInfo.Dependencies) > 0 {
			contextBuilder.WriteString(fmt.Sprintf("Dependencies: %s\n", strings.Join(s.projectInfo.Dependencies, ", ")))
		}
		contextBuilder.WriteString("\n---\n\n")
	}
	
	// ユーザー入力を追加
	contextBuilder.WriteString(userInput)
	
	return contextBuilder.String()
}

// カラフルな起動メッセージを表示
func (s *Session) printWelcomeMessage() {
	// ANSIカラーコード
	bold := "\033[1m"
	blue := "\033[34m"
	cyan := "\033[36m"
	reset := "\033[0m"
	
	fmt.Printf("%s%svyb%s %s- Feel the rhythm of perfect code%s\n\n", bold, blue, reset, cyan, reset)
	
	// プロジェクト情報を簡潔に表示
	if s.projectInfo != nil {
		workDirName := filepath.Base(s.workDir)
		fmt.Printf("%s📁 %s%s %s(%s)%s\n\n", cyan, workDirName, reset, "\033[90m", s.projectInfo.Language, reset)
	}
}

// カラー対応プロンプトを表示
func (s *Session) printColoredPrompt() {
	// ANSIカラーコード
	green := "\033[32m"
	reset := "\033[0m"
	
	fmt.Printf("%s>%s ", green, reset)
}

// メタ情報を表示（Claude Code風）
func (s *Session) displayMetaInfo(duration time.Duration, contentLength int) {
	// ANSIカラーコード
	gray := "\033[90m"
	reset := "\033[0m"
	
	// 簡易的なトークン数推定（文字数÷4）
	estimatedTokens := contentLength / 4
	
	// Claude Code風のメタ情報表示（グレー色）
	fmt.Printf("\n%s🕒 %dms • 📝 ~%d tokens • 🤖 %s%s\n\n", 
		gray,
		duration.Milliseconds(), 
		estimatedTokens, 
		s.model,
		reset)
}

// フォーマット済みレスポンスを表示（Markdown対応）
func (s *Session) displayFormattedResponse(content string) {
	lines := strings.Split(content, "\n")
	inCodeBlock := false
	codeLanguage := ""
	
	for _, line := range lines {
		// コードブロック開始の検出
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				// コードブロック開始
				inCodeBlock = true
				codeLanguage = strings.TrimPrefix(line, "```")
				s.printCodeBlockHeader(codeLanguage)
			} else {
				// コードブロック終了
				inCodeBlock = false
				s.printCodeBlockFooter()
			}
			continue
		}
		
		if inCodeBlock {
			// コードブロック内
			s.printCodeLine(line)
		} else {
			// 通常テキスト（Markdown強調対応）
			s.printFormattedLine(line)
		}
		
		// Claude風タイピング効果
		time.Sleep(time.Millisecond * 2)
	}
}

// コードブロックヘッダーを表示
func (s *Session) printCodeBlockHeader(language string) {
	gray := "\033[90m"
	reset := "\033[0m"
	
	if language != "" {
		fmt.Printf("\n%s┌─ %s ─%s\n", gray, language, reset)
	} else {
		fmt.Printf("\n%s┌─ code ─%s\n", gray, reset)
	}
}

// コードブロックフッターを表示
func (s *Session) printCodeBlockFooter() {
	gray := "\033[90m"
	reset := "\033[0m"
	
	fmt.Printf("%s└────────%s\n\n", gray, reset)
}

// コード行を表示（シンタックスハイライト風）
func (s *Session) printCodeLine(line string) {
	blue := "\033[94m"
	yellow := "\033[93m"
	green := "\033[92m"
	reset := "\033[0m"
	
	// 簡易シンタックスハイライト
	if strings.Contains(line, "func ") {
		line = strings.ReplaceAll(line, "func ", blue+"func "+reset)
	}
	if strings.Contains(line, "import ") {
		line = strings.ReplaceAll(line, "import ", yellow+"import "+reset)
	}
	if strings.Contains(line, "package ") {
		line = strings.ReplaceAll(line, "package ", green+"package "+reset)
	}
	
	fmt.Printf("│ %s\n", line)
}

// フォーマット済みテキスト行を表示
func (s *Session) printFormattedLine(line string) {
	// **太字** 対応
	if strings.Contains(line, "**") {
		bold := "\033[1m"
		reset := "\033[0m"
		line = strings.ReplaceAll(line, "**", bold)
		// 奇数回目のreplace後にresetを追加
		parts := strings.Split(line, bold)
		result := parts[0]
		for i := 1; i < len(parts); i++ {
			if i%2 == 1 {
				result += bold + parts[i]
			} else {
				result += reset + parts[i]
			}
		}
		line = result
	}
	
	fmt.Println(line)
}
