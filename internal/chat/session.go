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
	provider     llm.Provider      // LLMプロバイダー
	messages     []llm.ChatMessage // 会話履歴
	model        string            // 使用するモデル名
	mcpManager   *mcp.Manager      // MCPマネージャー
	workDir      string            // 作業ディレクトリ
	contextFiles []string          // コンテキストファイル一覧
	projectInfo  *ProjectContext   // プロジェクト情報
	inputHistory *InputHistory     // 入力履歴管理
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

	session := &Session{
		provider:     provider,
		messages:     make([]llm.ChatMessage, 0),
		model:        model,
		mcpManager:   mcp.NewManager(),
		workDir:      workDir,
		contextFiles: make([]string, 0),
		inputHistory: NewInputHistory(100), // 最大100個の履歴を保持
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

		// レスポンス後に区切り線
		fmt.Println()
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

// Claude Code風起動メッセージを表示
func (s *Session) printWelcomeMessage() {
	// ANSIカラーコード
	bold := "\033[1m"
	blue := "\033[34m"
	cyan := "\033[36m"
	gray := "\033[90m"
	green := "\033[32m"
	reset := "\033[0m"

	// メインタイトル
	fmt.Printf("\n%s%svyb%s %s- Feel the rhythm of perfect code%s\n", bold, blue, reset, cyan, reset)

	// プロジェクト情報をClaude Code風に表示
	if s.projectInfo != nil {
		workDirName := filepath.Base(s.workDir)
		fmt.Printf("%s%s%s", gray, workDirName, reset)

		// 言語情報
		if s.projectInfo.Language != "Unknown" && s.projectInfo.Language != "" {
			fmt.Printf(" %s•%s %s%s%s", gray, reset, green, s.projectInfo.Language, reset)
		}

		// Git情報
		gitInfo := s.getGitPromptInfo()
		if gitInfo.branch != "" {
			fmt.Printf(" %s•%s %s%s%s", gray, reset, cyan, gitInfo.branch, reset)
		}

		fmt.Printf("\n")
	}

	// ヘルプヒント
	fmt.Printf("%sType your message and press Enter. Use %s/help%s for commands, or %sexit%s to quit.%s\n\n",
		gray, green, gray, green, gray, reset)
}

// Claude Code風動的プロンプトを表示
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

	// ブランチ名を取得（簡易実装）
	if s.projectInfo != nil && s.projectInfo.GitBranch != "" {
		info.branch = s.projectInfo.GitBranch
	} else {
		info.branch = "main" // デフォルト
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

// Claude Code風ストリーミング表示（文字ごとのタイピング効果）
func (s *Session) displayFormattedResponse(content string) {
	lines := strings.Split(content, "\n")
	inCodeBlock := false
	codeLanguage := ""

	for lineIndex, line := range lines {
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
			// コードブロック内（タイピング効果なし）
			s.printCodeLine(line)
		} else {
			// 通常テキスト：文字ごとのタイピング効果
			s.printTypingLine(line)
		}

		// 行間の自然な間隔
		if lineIndex < len(lines)-1 {
			time.Sleep(time.Millisecond * 50)
		}
	}
}

// 文字ごとのタイピング効果で行を表示
func (s *Session) printTypingLine(line string) {
	// Markdown **太字** の前処理
	processedLine := s.processMarkdownFormatting(line)

	// 文字ごとに表示（日本語対応）
	runes := []rune(processedLine)
	for i, r := range runes {
		fmt.Print(string(r))

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

// Markdown書式を処理
func (s *Session) processMarkdownFormatting(line string) string {
	// **太字** 対応
	if strings.Contains(line, "**") {
		bold := "\033[1m"
		reset := "\033[0m"

		parts := strings.Split(line, "**")
		result := parts[0]

		for i := 1; i < len(parts); i++ {
			if i%2 == 1 {
				// 奇数番目：太字開始
				result += bold + parts[i]
			} else {
				// 偶数番目：太字終了
				result += reset + parts[i]
			}
		}
		line = result
	}

	return line
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
		fmt.Printf("%s/history%s       - 入力履歴を表示\n", green, reset)
		fmt.Printf("%s/status%s        - プロジェクト状態を表示\n", green, reset)
		fmt.Printf("%s/info%s          - システム情報を表示\n", green, reset)
		fmt.Printf("%s/save <file>%s   - 会話を保存\n", green, reset)
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
