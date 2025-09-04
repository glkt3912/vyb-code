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

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/input"
	"github.com/glkt/vyb-code/internal/interrupt"
	"github.com/glkt/vyb-code/internal/llm"
	"github.com/glkt/vyb-code/internal/markdown"
	"github.com/glkt/vyb-code/internal/mcp"
	"github.com/glkt/vyb-code/internal/streaming"
	"github.com/glkt/vyb-code/internal/tools"
	"github.com/glkt/vyb-code/internal/security"
)

// 会話セッションを管理する構造体
type Session struct {
	provider         llm.Provider         // LLMプロバイダー
	messages         []llm.ChatMessage    // 会話履歴
	model            string               // 使用するモデル名
	mcpManager       *mcp.Manager         // MCPマネージャー
	workDir          string               // 作業ディレクトリ
	contextFiles     []string             // コンテキストファイル一覧
	projectInfo      *ProjectContext      // プロジェクト情報
	inputHistory     *InputHistory        // 入力履歴管理（後方互換性のため保持）
	markdownRender   *markdown.Renderer   // Markdownレンダラー
	inputReader      *input.Reader        // 拡張入力リーダー
	streamProcessor  *streaming.Processor // ストリーミングプロセッサー
	gitOps           *tools.GitOperations // Git操作
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

// 入力履歴管理
type InputHistory struct {
	history []string
	index   int
	maxSize int
}

// 新しい入力履歴マネージャーを作成
func NewInputHistory(maxSize int) *InputHistory {
	return &InputHistory{
		history: make([]string, 0),
		index:   -1,
		maxSize: maxSize,
	}
}

// 履歴にコマンドを追加
func (h *InputHistory) Add(command string) {
	command = strings.TrimSpace(command)
	if command == "" || (len(h.history) > 0 && h.history[len(h.history)-1] == command) {
		return
	}

	h.history = append(h.history, command)
	if len(h.history) > h.maxSize {
		h.history = h.history[1:]
	}
	h.index = len(h.history)
}

// 前の履歴を取得（上矢印）
func (h *InputHistory) Previous() string {
	if len(h.history) == 0 {
		return ""
	}
	if h.index > 0 {
		h.index--
	}
	return h.history[h.index]
}

// 次の履歴を取得（下矢印）
func (h *InputHistory) Next() string {
	if len(h.history) == 0 {
		return ""
	}
	if h.index < len(h.history)-1 {
		h.index++
		return h.history[h.index]
	} else {
		h.index = len(h.history)
		return ""
	}
}

// 履歴をリセット
func (h *InputHistory) Reset() {
	h.index = len(h.history)
}

// 新しい会話セッションを作成
func NewSession(provider llm.Provider, model string) *Session {
	workDir, _ := os.Getwd()

	// Git操作を初期化
	constraints := security.NewDefaultConstraints(workDir)
	gitOps := tools.NewGitOperations(constraints, workDir)
	
	session := &Session{
		provider:        provider,
		messages:        make([]llm.ChatMessage, 0),
		model:           model,
		mcpManager:      mcp.NewManager(),
		workDir:         workDir,
		contextFiles:    make([]string, 0),
		inputHistory:    NewInputHistory(100),  // 後方互換性のため保持
		markdownRender:  markdown.NewRenderer(), // Markdownレンダラーを初期化
		inputReader:     input.NewReader(),     // 拡張入力リーダーを初期化
		streamProcessor: streaming.NewProcessor(), // ストリーミングプロセッサーを初期化
		gitOps:          gitOps,                // Git操作を初期化
	}

	// プロジェクト情報を初期化
	session.initializeProjectContext()

	return session
}

// 設定に基づいてセッションを作成
func NewSessionWithConfig(provider llm.Provider, model string, cfg *config.Config) *Session {
	session := NewSession(provider, model)
	
	// 設定に基づいてストリーミングプロセッサーを調整
	if cfg != nil {
		streamConfig := streaming.StreamConfig{
			TokenDelay:      time.Duration(cfg.TerminalMode.TypingSpeed) * time.Millisecond,
			SentenceDelay:   time.Duration(cfg.TerminalMode.TypingSpeed*6) * time.Millisecond,
			ParagraphDelay:  time.Duration(cfg.TerminalMode.TypingSpeed*12) * time.Millisecond,
			CodeBlockDelay:  time.Duration(cfg.TerminalMode.TypingSpeed/3) * time.Millisecond,
			EnableStreaming: cfg.TerminalMode.TypingSpeed > 0,
			MaxLineLength:   100,
			EnablePaging:    false,
			PageSize:        25,
		}
		session.streamProcessor.UpdateConfig(streamConfig)
		
		// 入力履歴サイズを設定
		if cfg.TerminalMode.HistorySize > 0 {
			session.inputHistory = NewInputHistory(cfg.TerminalMode.HistorySize)
		}
	}

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

// メモリ効率的な履歴管理（長い会話を圧縮）
func (s *Session) optimizeHistory() {
	const maxMessages = 20 // 最大保持メッセージ数
	const summaryThreshold = 30 // 要約開始の閾値

	if len(s.messages) <= maxMessages {
		return
	}

	// 古いメッセージを要約
	if len(s.messages) > summaryThreshold {
		oldMessages := s.messages[:len(s.messages)-maxMessages]
		recentMessages := s.messages[len(s.messages)-maxMessages:]

		// 要約を作成（簡易実装）
		summary := s.createConversationSummary(oldMessages)
		
		// 要約メッセージで置換
		summaryMessage := llm.ChatMessage{
			Role:    "user",
			Content: fmt.Sprintf("# 前回までの会話要約\n%s\n\n--- 以下、最近の会話 ---", summary),
		}

		s.messages = append([]llm.ChatMessage{summaryMessage}, recentMessages...)
	}
}

// 会話要約を作成
func (s *Session) createConversationSummary(messages []llm.ChatMessage) string {
	if len(messages) == 0 {
		return "（前回の会話なし）"
	}

	var topics []string
	var codeFiles []string
	
	for _, msg := range messages {
		if msg.Role == "user" {
			// ユーザーの質問から主要トピックを抽出
			content := strings.ToLower(msg.Content)
			
			// ファイル名の抽出
			if strings.Contains(content, ".go") || strings.Contains(content, ".js") || 
			   strings.Contains(content, ".py") || strings.Contains(content, ".ts") {
				// 簡易ファイル名抽出
				words := strings.Fields(msg.Content)
				for _, word := range words {
					if strings.Contains(word, ".") && len(word) < 50 {
						codeFiles = append(codeFiles, word)
					}
				}
			}

			// 主要アクション動詞を抽出
			actions := []string{"作成", "修正", "追加", "削除", "実装", "改善", "分析", "説明"}
			for _, action := range actions {
				if strings.Contains(content, action) {
					topics = append(topics, action)
					break
				}
			}
		}
	}

	summary := fmt.Sprintf("過去 %d 件のメッセージ", len(messages))
	if len(topics) > 0 {
		summary += fmt.Sprintf("（主な作業: %s）", strings.Join(topics[:min(3, len(topics))], "、"))
	}
	if len(codeFiles) > 0 {
		summary += fmt.Sprintf("（関連ファイル: %s）", strings.Join(codeFiles[:min(3, len(codeFiles))], "、"))
	}

	return summary
}

// minヘルパー関数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

	// クリーンアップを保証
	defer func() {
		if s.inputReader != nil {
			s.inputReader.Close()
		}
	}()

	for {
		// Claude風カラープロンプトを設定
		s.inputReader.SetPrompt(s.buildColoredPrompt())

		// 拡張入力システムで読み込み
		input, err := s.inputReader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			if strings.Contains(err.Error(), "interrupted") {
				fmt.Printf("\n^C\n")
				continue
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

		// スラッシュコマンド処理
		if strings.HasPrefix(trimmed, "/") {
			if s.handleSlashCommand(trimmed) {
				continue
			}
			// スラッシュコマンドが無効の場合、通常処理に進む
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

		// レスポンス後に視覚的区切り
		s.printMessageSeparator()
	}

	return nil
}

// Claude Code風高度入力システム（履歴・マルチライン・編集対応）
func (s *Session) readMultilineInput(reader *bufio.Reader) (string, error) {
	var currentInput strings.Builder
	isMultilineMode := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		// 改行を除去
		line = strings.TrimSuffix(line, "\n")

		// 特殊キー処理をシミュレート（実際のキーコード処理は別途必要）
		if strings.HasSuffix(line, "\\") && !isMultilineMode {
			// マルチラインモード開始（\ + Enter）
			isMultilineMode = true
			currentInput.WriteString(strings.TrimSuffix(line, "\\"))
			currentInput.WriteString("\n")
			fmt.Print("... ")
			continue
		}

		if isMultilineMode {
			// マルチラインモード中
			if line == "" {
				// 空行で送信完了
				result := strings.TrimSpace(currentInput.String())
				if result != "" {
					s.inputHistory.Add(result)
				}
				return result, nil
			}
			currentInput.WriteString(line)
			currentInput.WriteString("\n")
			fmt.Print("... ")
			continue
		}

		// 単一行入力処理
		if line != "" {
			s.inputHistory.Add(line)
			return line, nil
		}

		// 空行の場合は継続
		continue
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

// Claude Code風ストリーミング応答でLLMに送信（thinking制御付き・中断対応）
func (s *Session) sendToLLMStreamingWithThinking(stopThinking func()) error {
	// 中断可能な操作として実行
	return interrupt.WithInterruption(func(ctx context.Context) error {
		// リクエスト開始時間を記録
		startTime := time.Now()

		// チャットリクエストを作成
		req := llm.ChatRequest{
			Model:    s.model,
			Messages: s.messages,
			Stream:   false, // 既存API構造を活用
		}

		// 中断可能なLLMリクエスト
		respCh := make(chan *llm.ChatResponse, 1)
		errCh := make(chan error, 1)

		go func() {
			resp, err := s.provider.Chat(context.Background(), req)
			if err != nil {
				errCh <- err
				return
			}
			respCh <- resp
		}()

		// レスポンス待機（中断可能）
		var resp *llm.ChatResponse
		var err error

		select {
		case resp = <-respCh:
			// 正常レスポンス受信
		case err = <-errCh:
			// エラー受信
		case <-ctx.Done():
			// 中断された
			stopThinking()
			fmt.Printf("\n\033[33m⚠️  リクエストが中断されました\033[0m\n")
			return fmt.Errorf("request interrupted")
		}

		// thinking状態を停止
		stopThinking()

		if err != nil {
			return fmt.Errorf("LLM request failed: %w", err)
		}

		// レスポンス時間を計算
		duration := time.Since(startTime)

		// Claude Code風のクリーンなレスポンス表示（中断可能）
		content := resp.Message.Content
		
		// 中断可能なレスポンス表示
		if err := s.displayFormattedResponseInterruptible(content, ctx); err != nil {
			// 中断された場合、部分的な応答も保存
			fmt.Printf("\n\033[33m⚠️  表示が中断されましたが、応答は保存されました\033[0m\n")
		}

		// メタ情報表示
		s.displayMetaInfo(duration, len(content))

		// レスポンスメッセージを履歴に追加
		s.messages = append(s.messages, resp.Message)
		
		// 長い会話の場合、メモリ効率化を実行
		s.optimizeHistory()

		return nil
	})
}

// Claude Code風thinking状態アニメーション
func (s *Session) startThinkingAnimation() func() {
	// Claude Code風のよりエレガントなアニメーション
	frames := []string{
		"thinking",
		"thinking .",
		"thinking . .",
		"thinking . . .",
		"thinking . .",
		"thinking .",
	}
	frameIndex := 0

	// カラーコード
	gray := "\033[90m"
	reset := "\033[0m"

	// 停止チャネル
	stopCh := make(chan struct{})

	// アニメーションゴルーチンを開始
	go func() {
		ticker := time.NewTicker(400 * time.Millisecond) // より滑らかなアニメーション
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				// thinkingテキストを完全にクリアして改行位置に戻る
				fmt.Print("\r" + strings.Repeat(" ", 60) + "\r")
				return
			case <-ticker.C:
				// Claude Code風のグレー色でアニメーション表示
				fmt.Printf("\r%s%s%s", gray, frames[frameIndex], reset)
				frameIndex = (frameIndex + 1) % len(frames)
			}
		}
	}()

	// 停止関数を返す
	return func() {
		close(stopCh)
		time.Sleep(200 * time.Millisecond) // クリア処理の完了を待つ
		// 確実にクリア
		fmt.Print("\r" + strings.Repeat(" ", 60) + "\r")
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
	// 実際のGitブランチ名を取得
	if s.gitOps != nil {
		if branch, err := s.gitOps.GetCurrentBranch(); err == nil {
			return branch
		}
	}
	// フォールバック
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

// Claude Code風起動メッセージを表示
func (s *Session) printWelcomeMessage() {
	// ANSIカラーコード
	bold := "\033[1m"
	blue := "\033[34m"
	cyan := "\033[36m"
	gray := "\033[90m"
	green := "\033[32m"
	magenta := "\033[35m"
	reset := "\033[0m"

	// 美しい境界線
	fmt.Printf("\n%s╭──────────────────────────────────────────╮%s\n", gray, reset)
	
	// メインタイトル
	fmt.Printf("%s│%s  %s%s🎵 vyb%s %s- Feel the rhythm of perfect code%s %s│%s\n", 
		gray, reset, bold, magenta, reset, cyan, reset, gray, reset)
	
	fmt.Printf("%s╰──────────────────────────────────────────╯%s\n", gray, reset)

	// プロジェクト情報をClaude Code風に表示
	if s.projectInfo != nil {
		workDirName := filepath.Base(s.workDir)
		fmt.Printf("\n%s📁 %s%s%s", blue, bold, workDirName, reset)

		// 言語情報
		if s.projectInfo.Language != "Unknown" && s.projectInfo.Language != "" {
			fmt.Printf(" %s•%s %s🔧 %s%s", gray, reset, green, s.projectInfo.Language, reset)
		}

		// Git情報
		gitInfo := s.getGitPromptInfo()
		if gitInfo.branch != "" {
			fmt.Printf(" %s•%s %s🌿 %s%s", gray, reset, cyan, gitInfo.branch, reset)
		}

		fmt.Printf("\n")
	}

	// 拡張ヘルプヒント
	fmt.Printf("\n%s✨ 拡張機能:%s\n", cyan, reset)
	fmt.Printf("  %s↑↓%s 履歴ナビゲーション  %sTab%s オートコンプリート  %s/help%s コマンド一覧\n", 
		green, gray, green, gray, green, reset)
	fmt.Printf("  %s/edit%s マルチライン入力  %s/retry%s レスポンス再生成  %sexit%s 終了\n\n",
		green, gray, green, gray, green, reset)
}

// メッセージ間の視覚的区切りを表示
func (s *Session) printMessageSeparator() {
	gray := "\033[90m"
	reset := "\033[0m"
	
	// さりげない区切り線
	fmt.Printf("\n%s────────────────────────────────────────%s\n\n", gray, reset)
}

// Claude Code風プロンプト文字列を構築
func (s *Session) buildColoredPrompt() string {
	// ANSIカラーコード
	green := "\033[32m"
	blue := "\033[34m"
	yellow := "\033[33m"
	gray := "\033[90m"
	reset := "\033[0m"

	// Git情報を取得
	gitInfo := s.getGitPromptInfo()
	projectName := filepath.Base(s.workDir)

	// ベースプロンプト
	prompt := fmt.Sprintf("%s%s%s", blue, projectName, reset)

	// Git情報を追加
	if gitInfo.branch != "" {
		prompt += fmt.Sprintf("%s[%s%s%s]%s", gray, green, gitInfo.branch, gray, reset)
	}

	// 変更ファイル数を表示
	if gitInfo.changes > 0 {
		prompt += fmt.Sprintf("%s(%s%d%s)%s", gray, yellow, gitInfo.changes, gray, reset)
	}

	// 最終プロンプト記号
	prompt += fmt.Sprintf(" %s>%s ", green, reset)

	return prompt
}

// Claude Code風動的プロンプトを表示（廃止予定）
func (s *Session) printColoredPrompt() {
	// ANSIカラーコード
	green := "\033[32m"
	blue := "\033[34m"
	yellow := "\033[33m"
	gray := "\033[90m"
	reset := "\033[0m"

	// Git情報を取得
	gitInfo := s.getGitPromptInfo()
	projectName := filepath.Base(s.workDir)

	// ベースプロンプト
	prompt := fmt.Sprintf("%s%s%s", blue, projectName, reset)

	// Git情報を追加
	if gitInfo.branch != "" {
		prompt += fmt.Sprintf("%s[%s%s%s]%s", gray, green, gitInfo.branch, gray, reset)
	}

	// 変更ファイル数を表示
	if gitInfo.changes > 0 {
		prompt += fmt.Sprintf("%s(%s%d%s)%s", gray, yellow, gitInfo.changes, gray, reset)
	}

	// 最終プロンプト記号
	prompt += fmt.Sprintf(" %s>%s ", green, reset)

	fmt.Print(prompt)
}

// Gitプロンプト情報
type GitPromptInfo struct {
	branch  string
	changes int
	status  string
}

// Git情報をプロンプト用に取得
func (s *Session) getGitPromptInfo() GitPromptInfo {
	info := GitPromptInfo{}

	// 実際のGitブランチ名を取得
	if s.gitOps != nil {
		if currentBranch, err := s.gitOps.GetCurrentBranch(); err == nil {
			info.branch = currentBranch
		} else if s.projectInfo != nil && s.projectInfo.GitBranch != "" {
			info.branch = s.projectInfo.GitBranch
		} else {
			info.branch = "main" // フォールバック
		}
	} else {
		info.branch = "main" // gitOpsが利用できない場合
	}

	// TODO: 実際のgit statusコマンドを実行して変更ファイル数を取得
	// 現在は固定値
	info.changes = 0

	return info
}

// Claude Code風メタ情報表示
func (s *Session) displayMetaInfo(duration time.Duration, contentLength int) {
	// ANSIカラーコード
	gray := "\033[90m"
	reset := "\033[0m"

	// より正確なトークン数推定（日本語考慮）
	estimatedTokens := s.estimateTokenCount(contentLength)

	// レスポンス速度評価
	speedEmoji := s.getSpeedEmoji(duration)

	// Claude Code風のよりリッチなメタ情報表示
	fmt.Printf("\n%s%s %dms • 📝 ~%d tokens • 🤖 %s%s\n\n",
		gray,
		speedEmoji,
		duration.Milliseconds(),
		estimatedTokens,
		s.model,
		reset)
}

// トークン数をより正確に推定
func (s *Session) estimateTokenCount(contentLength int) int {
	// 日本語と英語の混在を考慮した推定
	// 日本語文字は約1.5トークン、英語は約0.25トークン

	// 簡易推定：文字数 ÷ 3.5
	return contentLength * 10 / 35
}

// レスポンス速度に応じた絵文字を取得
func (s *Session) getSpeedEmoji(duration time.Duration) string {
	ms := duration.Milliseconds()

	if ms < 1000 {
		return "⚡" // 非常に高速
	} else if ms < 3000 {
		return "🕒" // 高速
	} else if ms < 10000 {
		return "⏳" // 普通
	} else {
		return "🐌" // 低速
	}
}

// Claude Code風ストリーミング表示（拡張Markdown対応）
func (s *Session) displayFormattedResponse(content string) {
	// Markdownレンダリングされたコンテンツを取得
	rendered := s.markdownRender.Render(content)

	// 高度なストリーミング処理で表示
	if err := s.streamProcessor.StreamContent(rendered); err != nil {
		// ストリーミングエラー時はフォールバック
		fmt.Print(rendered)
	}
}

// 中断可能なレスポンス表示
func (s *Session) displayFormattedResponseInterruptible(content string, ctx context.Context) error {
	// Markdownレンダリングされたコンテンツを取得
	rendered := s.markdownRender.Render(content)

	// 中断チャネルを作成
	interruptCh := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(interruptCh)
	}()

	// 中断可能なストリーミング処理で表示
	return s.streamProcessor.StreamContentInterruptible(rendered, interruptCh)
}

// 文字ごとのタイピング効果で行を表示（既にレンダリング済みの行に対して）
func (s *Session) printTypingLine(line string) {
	// 既にMarkdownレンダリング済みなので、タイピング効果のみ適用
	runes := []rune(line)
	for i, r := range runes {
		fmt.Print(string(r))

		// ANSIエスケープシーケンスはスキップ
		if r == '\033' {
			// エスケープシーケンス中は高速表示
			continue
		}

		// タイピング速度調整（句読点後は少し長めの停止）
		delay := time.Millisecond * 15
		if strings.ContainsRune("。、！？", r) {
			delay = time.Millisecond * 100
		} else if strings.ContainsRune(" \t", r) {
			delay = time.Millisecond * 30
		}

		// 最後の文字でない場合のみ待機
		if i < len(runes)-1 {
			time.Sleep(delay)
		}
	}
	fmt.Println() // 行末の改行
}


// スラッシュコマンドを処理
func (s *Session) handleSlashCommand(command string) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	cmd := parts[0]
	args := parts[1:]

	green := "\033[32m"
	yellow := "\033[33m"
	cyan := "\033[36m"
	reset := "\033[0m"

	switch cmd {
	case "/help", "/h":
		fmt.Printf("%s--- Claude Code風コマンド ---%s\n", cyan, reset)
		fmt.Printf("%s/help, /h%s      - このヘルプを表示\n", green, reset)
		fmt.Printf("%s/clear, /c%s     - 会話履歴をクリア\n", green, reset)
		fmt.Printf("%s/retry%s         - 最後のレスポンスを再生成\n", green, reset)
		fmt.Printf("%s/edit%s          - マルチライン入力モード\n", green, reset)
		fmt.Printf("%s/history%s       - 入力履歴を表示\n", green, reset)
		fmt.Printf("%s/status%s        - プロジェクト状態を表示\n", green, reset)
		fmt.Printf("%s/info%s          - システム情報を表示\n", green, reset)
		fmt.Printf("%s/save <file>%s   - 会話を保存\n", green, reset)
		fmt.Printf("\n%s矢印キー%s: ↑↓で履歴 • %sTab%s: 補完 • %sCtrl+C%s: キャンセル\n", cyan, reset, cyan, reset, cyan, reset)
		fmt.Printf("%sexit, quit%s     - セッション終了\n", yellow, reset)
		return true

	case "/clear", "/c":
		s.ClearHistory()
		fmt.Printf("%s会話履歴をクリアしました%s\n", green, reset)
		return true

	case "/history":
		if len(s.inputHistory.history) == 0 {
			fmt.Printf("%s入力履歴はありません%s\n", yellow, reset)
		} else {
			fmt.Printf("%s--- 入力履歴 ---%s\n", cyan, reset)
			for i, cmd := range s.inputHistory.history {
				fmt.Printf("%s%3d%s: %s\n", green, i+1, reset, cmd)
			}
		}
		return true

	case "/status":
		s.displayProjectStatus()
		return true

	case "/info":
		s.displaySystemInfo()
		return true

	case "/retry":
		// 最後のユーザーメッセージを再送信
		if len(s.messages) >= 2 && s.messages[len(s.messages)-1].Role == "assistant" {
			// 最後のアシスタント応答を削除
			s.messages = s.messages[:len(s.messages)-1]
			
			fmt.Printf("%s最後のレスポンスを再生成中...%s\n", green, reset)
			
			// thinking状態表示を開始
			stopThinking := s.startThinkingAnimation()
			
			// 再送信
			if err := s.sendToLLMStreamingWithThinking(stopThinking); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		} else {
			fmt.Printf("%s再生成する前のメッセージがありません%s\n", yellow, reset)
		}
		return true

	case "/edit":
		// マルチライン入力モード
		fmt.Printf("%sマルチライン入力モード (空行で送信):%s\n", green, reset)
		multilineInput, err := s.inputReader.ReadMultiLine()
		if err != nil {
			if strings.Contains(err.Error(), "interrupted") {
				fmt.Printf("%sマルチライン入力がキャンセルされました%s\n", yellow, reset)
			} else {
				fmt.Printf("%sマルチライン入力エラー: %v%s\n", yellow, err, reset)
			}
			return true
		}

		if strings.TrimSpace(multilineInput) != "" {
			// マルチライン入力を処理
			contextualInput := s.buildContextualPrompt(multilineInput)
			s.messages = append(s.messages, llm.ChatMessage{
				Role:    "user",
				Content: contextualInput,
			})

			// thinking状態表示を開始
			stopThinking := s.startThinkingAnimation()

			// 送信
			if err := s.sendToLLMStreamingWithThinking(stopThinking); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

			// レスポンス後に区切り線
			fmt.Println()
		}
		return true

	case "/save":
		if len(args) > 0 {
			s.saveConversation(args[0])
		} else {
			fmt.Printf("%sファイル名を指定してください: /save <filename>%s\n", yellow, reset)
		}
		return true

	default:
		// 未知のスラッシュコマンド
		fmt.Printf("%s未知のコマンド: %s%s\n", yellow, cmd, reset)
		fmt.Printf("利用可能なコマンドは %s/help%s で確認できます\n", green, reset)
		return true
	}
}

// プロジェクト状態を表示
func (s *Session) displayProjectStatus() {
	cyan := "\033[36m"
	green := "\033[32m"
	reset := "\033[0m"

	fmt.Printf("%s--- プロジェクト状態 ---%s\n", cyan, reset)

	if s.projectInfo != nil {
		fmt.Printf("%s言語:%s %s\n", green, reset, s.projectInfo.Language)
		fmt.Printf("%s作業ディレクトリ:%s %s\n", green, reset, s.workDir)
		if s.projectInfo.GitBranch != "" {
			fmt.Printf("%sGitブランチ:%s %s\n", green, reset, s.projectInfo.GitBranch)
		}
		if len(s.projectInfo.Dependencies) > 0 {
			fmt.Printf("%s依存関係:%s %s\n", green, reset, strings.Join(s.projectInfo.Dependencies, ", "))
		}
	}

	fmt.Printf("%s会話履歴:%s %d件のメッセージ\n", green, reset, s.GetMessageCount())
	fmt.Printf("%s入力履歴:%s %d件のコマンド\n", green, reset, len(s.inputHistory.history))
}

// システム情報を表示
func (s *Session) displaySystemInfo() {
	cyan := "\033[36m"
	green := "\033[32m"
	reset := "\033[0m"

	fmt.Printf("%s--- システム情報 ---%s\n", cyan, reset)
	fmt.Printf("%sモデル:%s %s\n", green, reset, s.model)
	fmt.Printf("%s作業ディレクトリ:%s %s\n", green, reset, s.workDir)
	fmt.Printf("%sMCP接続:%s %d台のサーバー\n", green, reset, len(s.GetMCPTools()))
}

// 会話を保存
func (s *Session) saveConversation(filename string) {
	red := "\033[31m"
	reset := "\033[0m"

	// TODO: 実装予定 - 会話履歴をJSONまたはMarkdown形式で保存
	fmt.Printf("%s会話保存機能は開発中です: %s%s\n", red, filename, reset)
}
