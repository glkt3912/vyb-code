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

	"github.com/glkt/vyb-code/internal/ai"
	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/contextmanager"
	"github.com/glkt/vyb-code/internal/conversation"
	"github.com/glkt/vyb-code/internal/input"
	"github.com/glkt/vyb-code/internal/interactive"
	"github.com/glkt/vyb-code/internal/interrupt"
	"github.com/glkt/vyb-code/internal/llm"
	"github.com/glkt/vyb-code/internal/markdown"
	"github.com/glkt/vyb-code/internal/mcp"
	"github.com/glkt/vyb-code/internal/performance"
	"github.com/glkt/vyb-code/internal/security"
	"github.com/glkt/vyb-code/internal/streaming"
	"github.com/glkt/vyb-code/internal/tools"
	"github.com/glkt/vyb-code/internal/ui"
)

// llm.ProviderをAI.LLMClientに適応するアダプター
type llmProviderAdapter struct {
	provider llm.Provider
}

func (l *llmProviderAdapter) GenerateResponse(ctx context.Context, request *ai.GenerateRequest) (*ai.GenerateResponse, error) {
	// ai.GenerateRequestをllm.ChatRequestに変換
	messages := make([]llm.ChatMessage, len(request.Messages))
	for i, msg := range request.Messages {
		messages[i] = llm.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	chatReq := llm.ChatRequest{
		Model:    "qwen2.5-coder:14b", // TODO: 設定から取得
		Messages: messages,
		Stream:   false,
	}

	// LLMプロバイダーを呼び出し
	resp, err := l.provider.Chat(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	// レスポンスを変換
	return &ai.GenerateResponse{
		Content: resp.Message.Content,
	}, nil
}

// 会話セッションを管理する構造体
type Session struct {
	// 既存フィールド（後方互換性のため維持）
	provider        llm.Provider         // LLMプロバイダー
	messages        []llm.ChatMessage    // 会話履歴（段階的移行用）
	model           string               // 使用するモデル名
	mcpManager      *mcp.Manager         // MCPマネージャー
	workDir         string               // 作業ディレクトリ
	contextFiles    []string             // コンテキストファイル一覧
	projectInfo     *ProjectContext      // プロジェクト情報
	inputHistory    *InputHistory        // 入力履歴管理（後方互換性のため保持）
	markdownRender  *markdown.Renderer   // Markdownレンダラー
	inputReader     *input.Reader        // 拡張入力リーダー
	streamProcessor *streaming.Processor // ストリーミングプロセッサー
	gitOps          *tools.GitOperations // Git操作

	// Phase 7 統合フィールド
	vibeMode            bool                                   // バイブコーディングモード有効/無効
	contextManager      contextmanager.ContextManager          // コンテキスト圧縮管理
	interactiveSession  interactive.SessionManager             // インタラクティブセッション管理
	conversationManager conversation.ConversationManager       // メモリ効率会話管理
	currentSessionID    string                                 // 現在のインタラクティブセッションID
	cognitiveEngine     *conversation.CognitiveExecutionEngine // 認知推論エンジン
	currentThreadID     string                                 // 現在の会話スレッドID

	// Phase 2 軽量プロアクティブ機能
	lightProactive *conversation.LightweightProactiveManager // 軽量プロアクティブマネージャー
	contextEngine  *conversation.ContextSuggestionEngine     // コンテキスト提案エンジン
	lightMonitor   *conversation.LightweightMonitor          // 軽量プロジェクト監視

	// Phase 3 高度なインテリジェント機能
	advancedIntelligence *conversation.AdvancedIntelligenceEngine // 高度なインテリジェンスエンジン
	performanceMonitor   *performance.RealtimeMonitor             // リアルタイムパフォーマンス監視
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
		inputHistory:    NewInputHistory(100),     // 後方互換性のため保持
		markdownRender:  markdown.NewRenderer(),   // Markdownレンダラーを初期化
		inputReader:     input.NewReader(),        // 拡張入力リーダーを初期化
		streamProcessor: streaming.NewProcessor(), // ストリーミングプロセッサーを初期化
		gitOps:          gitOps,                   // Git操作を初期化
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

		// Phase 7: バイブコーディングモード設定（緊急修正：一時無効化）
		// バイブモード設定（認知推論エンジン統合済み）
		session.vibeMode = cfg.Features != nil && cfg.Features.VibeMode

		// Phase 2 & 3: プロアクティブ機能初期化
		if cfg.IsProactiveEnabled() {
			session.lightProactive = conversation.NewLightweightProactiveManager(cfg)
			session.contextEngine = conversation.NewContextSuggestionEngine(cfg)
			session.lightMonitor = conversation.NewLightweightMonitor(cfg)

			// Phase 3: 高度なインテリジェント機能を初期化
			if cfg.Proactive.Level >= config.ProactiveLevelStandard {
				fmt.Printf("[DEBUG] Initializing AdvancedIntelligenceEngine (Level: %v >= %v)\n",
					cfg.Proactive.Level, config.ProactiveLevelStandard)
				session.advancedIntelligence = conversation.NewAdvancedIntelligenceEngine(cfg, session.workDir)
				fmt.Printf("[DEBUG] AdvancedIntelligenceEngine initialized: %v\n", session.advancedIntelligence != nil)
			} else {
				fmt.Printf("[DEBUG] AdvancedIntelligenceEngine NOT initialized (Level: %v < %v)\n",
					cfg.Proactive.Level, config.ProactiveLevelStandard)
			}

			// Phase 3: パフォーマンス監視を初期化
			session.performanceMonitor = performance.NewRealtimeMonitor(cfg)
			if session.performanceMonitor != nil {
				session.performanceMonitor.Start()
			}
		}

		// バイブモード有効時のみPhase 7コンポーネントを初期化
		if session.vibeMode {
			session.initializeVibeModeComponents(provider)
		}
	}

	return session
}

// バイブモードコンポーネントを初期化
func (s *Session) initializeVibeModeComponents(provider llm.Provider) error {
	// コンテキスト管理を初期化
	s.contextManager = contextmanager.NewSmartContextManager()

	// AI服務を初期化（簡易版）
	workspaceDir, _ := os.Getwd() // 現在の作業ディレクトリを取得
	constraints := security.NewDefaultConstraints(workspaceDir)
	llmClient := &llmProviderAdapter{provider: provider} // プロバイダーをLLMClientに適応
	aiService := ai.NewAIService(llmClient, constraints)

	// ファイル編集ツールを初期化
	editTool := tools.NewEditTool(constraints, workspaceDir, 10*1024*1024) // 10MB制限

	// インタラクティブセッション管理を初期化
	vibeConfig := interactive.DefaultVibeConfig()
	s.interactiveSession = interactive.NewInteractiveSessionManager(
		s.contextManager,
		provider,
		aiService,
		editTool,
		vibeConfig,
	)

	// 認知推論エンジンを初期化
	cfg, _ := config.Load()
	s.cognitiveEngine = conversation.NewCognitiveExecutionEngine(cfg, workspaceDir, llmClient)

	// TODO: 会話管理の初期化（実装待ち）
	// s.conversationManager = conversation.NewConversationManager(...)

	return nil
}

// バイブコーディングモードでセッションを作成
func NewVibeSession(provider llm.Provider, model string, cfg *config.Config) *Session {
	if cfg == nil {
		cfg = &config.Config{}
	}

	// バイブモードを強制有効化
	if cfg.Features == nil {
		cfg.Features = &config.Features{}
	}
	cfg.Features.VibeMode = true

	return NewSessionWithConfig(provider, model, cfg)
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
	// 質問ヘッダーと内容を同じ行に表示
	s.printUserMessageWithContent(query)

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
	// Phase 3: パフォーマンス監視開始
	startTime := time.Now()
	defer func() {
		if s.performanceMonitor != nil {
			totalDuration := time.Since(startTime)
			s.performanceMonitor.RecordResponseTime(totalDuration)
		}
	}()

	// チャットリクエストを作成
	req := llm.ChatRequest{
		Model:    s.model,
		Messages: s.messages,
		Stream:   false, // 現在はストリーミング無効
	}

	// LLMプロバイダーにリクエスト送信
	llmStart := time.Now()
	ctx := context.Background()
	resp, err := s.provider.Chat(ctx, req)
	if err != nil {
		return fmt.Errorf("LLM request failed: %w", err)
	}

	// Phase 3: LLMレイテンシを記録
	if s.performanceMonitor != nil {
		llmDuration := time.Since(llmStart)
		s.performanceMonitor.RecordLLMLatency(llmDuration)
	}

	// Phase 2 & 3: プロアクティブ応答拡張
	enhancedResponse := resp.Message.Content
	fmt.Printf("[DEBUG] Messages length: %d\n", len(s.messages))
	if len(s.messages) > 0 {
		lastUserMessage := ""
		for i := len(s.messages) - 1; i >= 0; i-- {
			if s.messages[i].Role == "user" {
				lastUserMessage = s.messages[i].Content
				break
			}
		}

		// Phase 3: 高度なインテリジェンスで応答を拡張
		enhancementStart := time.Now()
		if s.advancedIntelligence != nil {
			fmt.Printf("[DEBUG] Using AdvancedIntelligenceEngine for user input: %s\n", lastUserMessage)
			if enhanced, err := s.advancedIntelligence.GenerateEnhancedResponse(resp.Message.Content, lastUserMessage, s.workDir); err == nil {
				enhancedResponse = enhanced
				fmt.Printf("[DEBUG] AdvancedIntelligence returned enhanced response\n")
				// パフォーマンス記録
				if s.performanceMonitor != nil {
					s.performanceMonitor.RecordProactiveUsage("intelligence_enhancement")
				}
			} else {
				fmt.Printf("[DEBUG] AdvancedIntelligence failed: %v\n", err)
			}
		} else if s.lightProactive != nil {
			fmt.Printf("[DEBUG] Using LightweightProactive fallback\n")
			// Phase 2: フォールバックとして軽量拡張を使用
			enhancedResponse = s.lightProactive.EnhanceResponse(resp.Message.Content, lastUserMessage, s.workDir)
			// パフォーマンス記録
			if s.performanceMonitor != nil {
				s.performanceMonitor.RecordProactiveUsage("proactive_response")
			}
		}

		// 拡張処理時間を記録
		if s.performanceMonitor != nil {
			enhancementDuration := time.Since(enhancementStart)
			s.performanceMonitor.RecordAnalysisTime(enhancementDuration)
		}
	}

	// レスポンスメッセージを履歴に追加（元のレスポンスを保存）
	s.messages = append(s.messages, resp.Message)

	// 拡張されたレスポンスを表示
	fmt.Printf("🎵 %s\n", enhancedResponse)

	return nil
}

// 会話履歴をクリアする
func (s *Session) ClearHistory() {
	s.messages = make([]llm.ChatMessage, 0)
}

// メモリ効率的な履歴管理（長い会話を圧縮）
func (s *Session) optimizeHistory() {
	const maxMessages = 20      // 最大保持メッセージ数
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

		// 質問ヘッダーと内容を同じ行に表示
		s.printUserMessageWithContent(input)

		// Phase 7: バイブモード対応処理
		if s.vibeMode {
			err = s.processVibeInput(trimmed)
		} else {
			// 従来の処理
			err = s.processTraditionalInput(trimmed)
		}

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

		// 回答ヘッダーを表示
		s.printAssistantMessageHeader()

		// Phase 4: プロアクティブ応答拡張処理をStreaming版にも適用
		content := resp.Message.Content
		enhancedContent := content

		// Phase 2 & 3: プロアクティブ応答拡張
		if len(s.messages) > 0 {
			lastUserMessage := ""
			for i := len(s.messages) - 1; i >= 0; i-- {
				if s.messages[i].Role == "user" {
					lastUserMessage = s.messages[i].Content
					break
				}
			}

			fmt.Printf("[DEBUG STREAMING] Using AdvancedIntelligenceEngine for user input: %s\n", lastUserMessage)

			// Phase 3: 高度なインテリジェンスで応答を拡張
			if s.advancedIntelligence != nil {
				if enhanced, err := s.advancedIntelligence.GenerateEnhancedResponse(content, lastUserMessage, s.workDir); err == nil {
					enhancedContent = enhanced
					fmt.Printf("[DEBUG STREAMING] AdvancedIntelligence returned enhanced response\n")
				} else {
					fmt.Printf("[DEBUG STREAMING] AdvancedIntelligence failed: %v\n", err)
				}
			} else if s.lightProactive != nil {
				fmt.Printf("[DEBUG STREAMING] Using LightweightProactive fallback\n")
				enhancedContent = s.lightProactive.EnhanceResponse(content, lastUserMessage, s.workDir)
			}
		}

		// 中断可能なレスポンス表示（拡張されたコンテンツを使用）
		if err := s.displayFormattedResponseInterruptible(enhancedContent, ctx); err != nil {
			// 中断された場合、部分的な応答も保存
			fmt.Printf("\n\033[33m⚠️  表示が中断されましたが、応答は保存されました\033[0m\n")
		}

		// メタ情報表示
		s.displayMetaInfo(duration, len(enhancedContent))

		// レスポンスメッセージを履歴に追加（拡張されたコンテンツで更新）
		enhancedMessage := resp.Message
		enhancedMessage.Content = enhancedContent
		s.messages = append(s.messages, enhancedMessage)

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
	// 言語マネージャーを使用して動的に検出
	languageManager := tools.NewLanguageManager()
	languages, err := languageManager.DetectProjectLanguages(s.workDir)
	if err != nil {
		return "Unknown"
	}

	// 最も使用されている言語を返す
	if len(languages) > 0 {
		return languages[0].GetName()
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
	cyan := "\033[36m"
	gray := "\033[90m"
	magenta := "\033[35m"
	reset := "\033[0m"

	// エレガントなロゴ表示
	fmt.Printf("\n%s⚡ %svyb%s %s- AI coding assistant%s\n",
		cyan, magenta, reset, gray, reset)

	// プロジェクト情報をClaude Code風に表示
	if s.projectInfo != nil {
		workDirName := filepath.Base(s.workDir)
		fmt.Printf("\n%s📁 %s%s%s", "\033[34m", "\033[1m", workDirName, reset)

		// 言語情報
		if s.projectInfo.Language != "Unknown" && s.projectInfo.Language != "" {
			fmt.Printf(" %s•%s %s🔧 %s%s", gray, reset, "\033[32m", s.projectInfo.Language, reset)
		}

		// Git情報
		gitInfo := s.getGitPromptInfo()
		if gitInfo.branch != "" {
			fmt.Printf(" %s•%s %s🌿 %s%s", gray, reset, cyan, gitInfo.branch, reset)
		}

		fmt.Printf("\n")

		// Phase 2: プロアクティブプロジェクト情報表示
		if s.lightProactive != nil {
			summary := s.lightProactive.GenerateProjectSummary(s.workDir)
			if summary != "" {
				fmt.Printf("%s\n", summary)
			}
		}

		// Phase 2: コンテキスト提案表示
		if s.contextEngine != nil {
			suggestions := s.contextEngine.GenerateStartupSuggestions(s.workDir)
			if len(suggestions) > 0 {
				fmt.Printf("\n%s\n", s.contextEngine.FormatSuggestions(suggestions))
			}
		}
	}
}

// メッセージ間の視覚的区切りを表示
func (s *Session) printMessageSeparator() {
	gray := "\033[90m"
	reset := "\033[0m"

	// さりげない区切り線
	fmt.Printf("\n%s────────────────────────────────────────%s\n\n", gray, reset)
}

// === Phase 1: 非同期プロジェクト分析統合 ===

// バックグラウンド分析マネージャー
type BackgroundAnalysisManager struct {
	session *Session
	config  *config.Config
	enabled bool
	cancel  context.CancelFunc
}

// バックグラウンド分析を初期化
func (s *Session) InitializeBackgroundAnalysis(cfg *config.Config) {
	if !cfg.IsProactiveEnabled() {
		return // プロアクティブ機能が無効の場合はスキップ
	}

	bam := &BackgroundAnalysisManager{
		session: s,
		config:  cfg,
		enabled: cfg.Proactive.BackgroundAnalysis,
	}

	if bam.enabled {
		ctx, cancel := context.WithCancel(context.Background())
		bam.cancel = cancel
		go bam.runBackgroundAnalysis(ctx)
	}
}

// バックグラウンド分析を実行
func (bam *BackgroundAnalysisManager) runBackgroundAnalysis(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // 5分間隔で軽量分析
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 軽量分析を実行
			bam.performLightweightAnalysis()
		}
	}
}

// 軽量分析を実行
func (bam *BackgroundAnalysisManager) performLightweightAnalysis() {
	if !bam.enabled || !bam.config.IsProactiveEnabled() {
		return
	}

	// 軽量分析を別ゴルーチンで実行（ノンブロッキング）
	go func() {
		// 実際の分析実装は後のフェーズで追加
		// 現在は設定の検証とログ出力のみ
		if bam.config.Proactive.Level != config.ProactiveLevelOff {
			// 軽量分析処理のプレースホルダー
			_ = bam.session.workDir
		}
	}()
}

// セッション終了時のクリーンアップ
func (s *Session) CleanupBackgroundProcesses() {
	// Phase 2 & 3: プロアクティブコンポーネントのクリーンアップ
	if s.lightProactive != nil {
		s.lightProactive.Close()
	}
	if s.contextEngine != nil {
		s.contextEngine.Close()
	}
	if s.lightMonitor != nil {
		s.lightMonitor.Close()
	}
	if s.advancedIntelligence != nil {
		s.advancedIntelligence.Close()
	}
	if s.performanceMonitor != nil {
		s.performanceMonitor.Close()
	}

	// バックグラウンド処理をキャンセル（実装は後のフェーズで完成）
	if s.vibeMode {
		// 現在は何もしない（Phase 1では軽量実装）
	}
}

// プロアクティブ機能の状態チェック
func (s *Session) CheckProactiveStatus(cfg *config.Config) {
	if cfg.IsProactiveEnabled() {
		// プロアクティブ機能が有効な場合の処理
		timeout, backgroundAnalysis, monitoring := cfg.GetProactiveLevelConfig()

		if backgroundAnalysis {
			// バックグラウンド分析が有効
			_ = timeout
			_ = monitoring
		}
	}
}

// コンテキスト圧縮の実行（軽量版）
func (s *Session) CompressContextIfNeeded() {
	// Phase 1: 基本的な圧縮ロジックのプレースホルダー
	if len(s.messages) > 30 {
		// 既存の optimizeHistory を呼び出し
		s.optimizeHistory()
	}
}

// プロアクティブ設定の動的更新
func (s *Session) UpdateProactiveSettings(cfg *config.Config) {
	// 設定変更時の動的更新処理
	if cfg.IsProactiveEnabled() {
		// プロアクティブ機能を有効化
		s.InitializeBackgroundAnalysis(cfg)
	} else {
		// プロアクティブ機能を無効化
		s.CleanupBackgroundProcesses()
	}
}

// パフォーマンス統計の取得
func (s *Session) GetPerformanceStats() map[string]interface{} {
	stats := map[string]interface{}{
		"message_count": len(s.messages),
		"vibe_mode":     s.vibeMode,
		"session_id":    s.currentSessionID,
		"thread_id":     s.currentThreadID,
	}

	// Phase 2 & 3: プロアクティブコンポーネントの統計を追加
	if s.lightProactive != nil {
		proactiveStats := s.lightProactive.GetPerformanceStats()
		for k, v := range proactiveStats {
			stats["proactive_"+k] = v
		}
	}

	if s.lightMonitor != nil {
		monitorStats := s.lightMonitor.GetStats()
		for k, v := range monitorStats {
			stats["monitor_"+k] = v
		}
	}

	if s.advancedIntelligence != nil {
		intelligenceStats := s.advancedIntelligence.GetStats()
		for k, v := range intelligenceStats {
			stats["intelligence_"+k] = v
		}
	}

	if s.performanceMonitor != nil {
		perfMetrics := s.performanceMonitor.GetMetrics()
		stats["performance_response_time"] = perfMetrics.ResponseTime.Current
		stats["performance_memory_usage"] = perfMetrics.MemoryUsage.Current
		stats["performance_request_count"] = perfMetrics.RequestCount.Total
	}

	return stats
}

// Phase 2: プロジェクト状態レポートを取得
func (s *Session) GetProjectStatusReport() (string, error) {
	if s.lightMonitor == nil {
		return "", fmt.Errorf("監視機能が無効です")
	}

	return s.lightMonitor.GenerateStatusReport(s.workDir)
}

// Phase 3: パフォーマンスサマリーを取得
func (s *Session) GetPerformanceSummary() string {
	if s.performanceMonitor == nil {
		return "パフォーマンス監視が無効です"
	}

	return s.performanceMonitor.GeneratePerformanceSummary()
}

// 質問の開始を表示
func (s *Session) printUserMessageHeader() {
	blue := "\033[34m"
	bold := "\033[1m"
	reset := "\033[0m"

	fmt.Printf("\n%s%s💬 質問:%s\n", blue, bold, reset)
}

// 質問ヘッダーと内容を同じ行に表示
func (s *Session) printUserMessageWithContent(content string) {
	blue := "\033[34m"
	bold := "\033[1m"
	reset := "\033[0m"

	// 前の行（プロンプト+入力）を完全にクリア
	fmt.Printf("\r\033[K\033[A\r\033[K")

	// 質問ヘッダーと内容を表示
	fmt.Printf("%s%s💬 質問:%s\n%s\n", blue, bold, reset, content)
}

// 回答の開始を表示
func (s *Session) printAssistantMessageHeader() {
	green := "\033[32m"
	bold := "\033[1m"
	reset := "\033[0m"

	fmt.Printf("\n%s%s🤖 回答:%s\n", green, bold, reset)
}

// Claude Code風プロンプト文字列を構築
func (s *Session) buildColoredPrompt() string {
	// ANSIカラーコード
	green := "\033[32m"
	blue := "\033[34m"
	bold := "\033[1m"
	reset := "\033[0m"

	// シンプルなプロンプト
	prompt := fmt.Sprintf("%s%svyb%s %s>%s ", blue, bold, reset, green, reset)

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
			// 質問ヘッダーと内容を表示
			s.printUserMessageWithContent(multilineInput)

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

// Phase 7: バイブコーディング用入力処理（プロアクティブAI機能統合）
func (s *Session) processVibeInput(input string) error {
	// 認知推論エンジンを使用した高度処理
	return s.processCognitiveInput(input)
}

// 認知推論エンジンを使用した高度入力処理
func (s *Session) processCognitiveInput(input string) error {
	if s.cognitiveEngine == nil {
		fmt.Printf("⚠️ 認知推論エンジンが初期化されていません。従来処理にフォールバック\n")
		return s.processTraditionalInput(input)
	}

	// thinking状態表示を開始
	stopThinking := s.startThinkingAnimation()
	defer func() {
		if stopThinking != nil {
			stopThinking()
		}
	}()

	// 認知推論処理を実行
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := s.cognitiveEngine.ProcessUserInputCognitively(ctx, input)
	if err != nil {
		fmt.Printf("❌ 認知処理エラー: %v\n", err)
		// エラー時は従来処理にフォールバック
		return s.processTraditionalInput(input)
	}

	// 認知処理結果を表示
	s.displayCognitiveResult(result)

	// 認知処理結果をLLMプロンプトに統合して最終応答を生成
	return s.generateEnhancedResponse(input, result)
}

// 認知処理結果を視覚的に表示
func (s *Session) displayCognitiveResult(result *conversation.CognitiveExecutionResult) {
	if result == nil {
		return
	}

	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	reset := "\033[0m"
	bold := "\033[1m"

	// 認知処理メタデータを表示
	fmt.Printf("%s%s🧠 認知処理完了%s%s\n", cyan, bold, reset, reset)
	fmt.Printf("   戦略: %s%s%s\n", yellow, result.ProcessingStrategy, reset)
	fmt.Printf("   信頼度: %s%.2f%s\n", green, result.ConfidenceLevel, reset)
	fmt.Printf("   創造性: %s%.2f%s\n", green, result.CreativityScore, reset)
	fmt.Printf("   推論深度: %s%d%s\n", green, result.ReasoningDepth, reset)
	fmt.Printf("   処理時間: %s%v%s\n", cyan, result.TotalProcessingTime, reset)

	// 学習成果がある場合表示
	if len(result.LearningOutcomes) > 0 {
		fmt.Printf("   学習成果: %s%d件獲得%s\n", yellow, len(result.LearningOutcomes), reset)
	}

	// 認知洞察がある場合表示
	if len(result.CognitiveInsights) > 0 {
		fmt.Printf("   認知洞察: %s%d件生成%s\n", green, len(result.CognitiveInsights), reset)
	}
}

// 認知処理結果をLLMプロンプトに統合して拡張応答を生成
func (s *Session) generateEnhancedResponse(input string, result *conversation.CognitiveExecutionResult) error {
	// 認知処理結果をプロンプトに統合
	enhancedPrompt := s.buildCognitiveEnhancedPrompt(input, result)

	// 拡張プロンプトでメッセージ履歴を更新
	s.messages = append(s.messages, llm.ChatMessage{
		Role:    "user",
		Content: enhancedPrompt,
	})

	// 認知分析情報を含むシステムメッセージを追加
	systemPrompt := s.buildCognitiveSystemPrompt(result)
	s.messages = append(s.messages, llm.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// ストリーミング応答で送信
	stopThinking := s.startThinkingAnimation()
	return s.sendToLLMStreamingWithThinking(stopThinking)
}

// 認知処理結果を統合したプロンプトを構築
func (s *Session) buildCognitiveEnhancedPrompt(input string, result *conversation.CognitiveExecutionResult) string {
	var prompt strings.Builder

	// プロジェクトコンテキスト
	prompt.WriteString(s.buildContextualPrompt(input))
	prompt.WriteString("\n\n---\n\n")

	// 認知分析結果を追加
	prompt.WriteString("🧠 **認知分析結果:**\n")
	prompt.WriteString(fmt.Sprintf("- 処理戦略: %s\n", result.ProcessingStrategy))
	prompt.WriteString(fmt.Sprintf("- 信頼度: %.2f\n", result.ConfidenceLevel))
	prompt.WriteString(fmt.Sprintf("- 創造性: %.2f\n", result.CreativityScore))
	prompt.WriteString(fmt.Sprintf("- 推論深度: %d\n", result.ReasoningDepth))

	// 認知洞察を追加
	if len(result.CognitiveInsights) > 0 {
		prompt.WriteString("\n**認知洞察:**\n")
		for i, insight := range result.CognitiveInsights {
			prompt.WriteString(fmt.Sprintf("%d. %s (信頼度: %.2f)\n",
				i+1, insight.Description, insight.Confidence))
		}
	}

	// 推奨アクションを追加
	if len(result.NextStepSuggestions) > 0 {
		prompt.WriteString("\n**推奨アクション:**\n")
		for i, suggestion := range result.NextStepSuggestions {
			prompt.WriteString(fmt.Sprintf("%d. %s (優先度: %s)\n",
				i+1, suggestion.Description, suggestion.Priority))
		}
	}

	return prompt.String()
}

// 認知処理結果に基づくシステムプロンプトを構築
func (s *Session) buildCognitiveSystemPrompt(result *conversation.CognitiveExecutionResult) string {
	var prompt strings.Builder

	prompt.WriteString("あなたは高度な認知推論システムと統合されたAIアシスタントです。")
	prompt.WriteString("上記の認知分析結果を活用し、以下の点を重視して応答してください：\n\n")

	if result.ConfidenceLevel >= 0.8 {
		prompt.WriteString("- 高い信頼度（%.2f）の分析結果に基づき、確信を持った推論を提供\n")
	} else if result.ConfidenceLevel >= 0.6 {
		prompt.WriteString("- 中程度の信頼度（%.2f）を考慮し、複数の可能性を検討\n")
	} else {
		prompt.WriteString("- 低い信頼度（%.2f）のため、慎重で探究的なアプローチを採用\n")
	}

	if result.CreativityScore >= 0.7 {
		prompt.WriteString("- 高い創造性（%.2f）を活用し、革新的な解決策を提示\n")
	}

	if result.ReasoningDepth >= 3 {
		prompt.WriteString("- 深い推論（レベル%d）に基づき、多面的な分析を提供\n")
	}

	prompt.WriteString("- 認知洞察と推奨アクションを具体的で実行可能な形で展開\n")
	prompt.WriteString("- 単なる情報提示ではなく、真の理解と分析的思考を示す\n")

	return fmt.Sprintf(prompt.String(), result.ConfidenceLevel, result.CreativityScore, result.ReasoningDepth)
}

// 従来の入力処理（後方互換性）
func (s *Session) processTraditionalInput(input string) error {
	// コンテキスト情報付きでユーザーメッセージを履歴に追加
	contextualInput := s.buildContextualPrompt(input)
	s.messages = append(s.messages, llm.ChatMessage{
		Role:    "user",
		Content: contextualInput,
	})

	// thinking状態表示を開始
	stopThinking := s.startThinkingAnimation()

	// Claude Code風ストリーミング応答で送信
	return s.sendToLLMStreamingWithThinking(stopThinking)
}

// コード提案応答の処理
func (s *Session) handleCodeSuggestionResponse(response *interactive.InteractionResponse) error {
	if len(response.Suggestions) == 0 {
		return s.handleMessageResponse(response)
	}

	suggestion := response.Suggestions[0]

	// 提案表示
	s.displayCodeSuggestion(suggestion)

	if response.RequiresConfirmation {
		// ユーザー確認を求める
		confirmed, err := s.getUserConfirmation("この提案を適用しますか？")
		if err != nil {
			return err
		}

		// 確認応答を送信
		err = s.interactiveSession.ConfirmSuggestion(s.currentSessionID, suggestion.ID, confirmed)
		if err != nil {
			return err
		}

		// 確認された場合は提案を適用
		if confirmed {
			ctx := context.Background()
			err = s.interactiveSession.ApplySuggestion(ctx, s.currentSessionID, suggestion.ID)
			if err != nil {
				return fmt.Errorf("提案適用エラー: %w", err)
			}
			fmt.Printf("✅ 提案を適用しました！\n")
		}

		return nil
	}

	return nil
}

// 確認応答の処理
func (s *Session) handleConfirmationResponse(response *interactive.InteractionResponse) error {
	// 確認が必要な場合の処理
	if response.RequiresConfirmation {
		confirmed, err := s.getUserConfirmation(response.Message)
		if err != nil {
			return err
		}

		if confirmed && len(response.Suggestions) > 0 {
			// 提案を適用
			ctx := context.Background()
			return s.interactiveSession.ApplySuggestion(ctx, s.currentSessionID, response.Suggestions[0].ID)
		}
	}

	return nil
}

// メッセージ応答の処理
func (s *Session) handleMessageResponse(response *interactive.InteractionResponse) error {
	// ストリーミング風に応答を表示
	err := s.streamProcessor.StreamContent(response.Message)
	if err != nil {
		// ストリーミングに失敗した場合は通常表示
		fmt.Print(response.Message)
	}

	// コンテキストマネージャーに応答を追加
	if s.contextManager != nil {
		contextItem := &contextmanager.ContextItem{
			Type:       contextmanager.ContextTypeImmediate,
			Content:    fmt.Sprintf("アシスタント応答: %s", response.Message),
			Metadata:   map[string]string{"type": "assistant_response", "session_id": s.currentSessionID},
			Importance: 0.6,
		}
		s.contextManager.AddContext(contextItem)
	}

	return nil
}

// コード提案の表示
func (s *Session) displayCodeSuggestion(suggestion *interactive.CodeSuggestion) {
	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	reset := "\033[0m"

	fmt.Printf("\n%s💡 コード提案%s\n", cyan, reset)
	fmt.Printf("%s信頼度:%s %.1f%% ", green, reset, suggestion.Confidence*100)

	impactText := map[interactive.ImpactLevel]string{
		interactive.ImpactLevelLow:      "低影響",
		interactive.ImpactLevelMedium:   "中影響",
		interactive.ImpactLevelHigh:     "高影響",
		interactive.ImpactLevelCritical: "重大影響",
	}
	fmt.Printf("%s影響:%s %s\n", yellow, reset, impactText[suggestion.ImpactLevel])

	if suggestion.Explanation != "" {
		fmt.Printf("\n%s説明:%s %s\n", cyan, reset, suggestion.Explanation)
	}

	if suggestion.SuggestedCode != suggestion.OriginalCode {
		fmt.Printf("\n%s提案コード:%s\n", green, reset)
		fmt.Printf("```\n%s\n```\n", suggestion.SuggestedCode)
	}
}

// ユーザー確認を取得
func (s *Session) getUserConfirmation(message string) (bool, error) {
	// TTY利用可能性をチェック
	if s.isTTYAvailable() {
		// Bubble Teaベースの確認ダイアログを使用
		dialog := ui.NewConfirmationDialog("📝 確認", message, []string{"✅ はい", "❌ いいえ"})

		confirmed, err := ui.RunConfirmationDialog(dialog)
		if err == nil {
			return confirmed, nil
		}
		// Bubble Teaでエラーが発生した場合はフォールバック
	}

	// 従来の方式（TTY利用不可またはBubble Teaエラー時）
	fmt.Printf("\n%s [y/N]: ", message)
	response, readErr := s.inputReader.ReadLine()
	if readErr != nil {
		return false, readErr
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// TTYが利用可能かどうかを確認
func (s *Session) isTTYAvailable() bool {
	// 標準入力がTTYかどうかを確認
	file, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	file.Close()

	// また、標準入出力がパイプでないことも確認
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	// パイプ経由の場合はTTYではない
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// プロアクティブAI機能統合メソッド群

// プロアクティブなプロジェクト分析を表示
func (s *Session) displayProactiveAnalysis() {
	// 簡略化された実装：バイブモードが有効な場合に分析情報を表示
	if s.vibeMode {
		s.displayCompactProjectInsights(nil)
	}
}

// コンパクトなプロジェクト洞察を表示
func (s *Session) displayCompactProjectInsights(analysis interface{}) {
	gray := "\033[90m"
	reset := "\033[0m"

	// 簡潔なプロジェクト状況表示
	fmt.Printf("%s🔍 ", gray)

	// 実際の分析データがあるかどうかに関係なく、プロアクティブな雰囲気を演出
	insights := []string{
		"プロジェクト状況を分析中...",
		"コンテキストを理解中...",
		"最適な提案を準備中...",
	}

	insight := insights[time.Now().Second()%len(insights)]
	fmt.Printf("%s%s\n", insight, reset)
}

// 強化されたthinkingアニメーション
func (s *Session) startEnhancedThinkingAnimation() func() {
	// より詳細で情報豊富なアニメーション
	frames := []string{
		"プロジェクトを分析",
		"プロジェクトを分析 .",
		"プロジェクトを分析 . .",
		"プロジェクトを分析 . . .",
		"コンテキスト理解中",
		"コンテキスト理解中 .",
		"コンテキスト理解中 . .",
		"最適な回答を生成",
		"最適な回答を生成 .",
		"最適な回答を生成 . .",
	}
	frameIndex := 0

	// カラーコード
	cyan := "\033[36m"
	gray := "\033[90m"
	reset := "\033[0m"

	// 停止チャネル
	stopCh := make(chan struct{})

	// アニメーションゴルーチンを開始
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond) // より詳細なアニメーション
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				// アニメーションを完全にクリア
				fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")
				return
			case <-ticker.C:
				// プロアクティブ風のアニメーション表示
				fmt.Printf("\r%s💭 %s%s%s%s", cyan, gray, frames[frameIndex], strings.Repeat(" ", 30), reset)
				frameIndex = (frameIndex + 1) % len(frames)
			}
		}
	}()

	// 停止関数を返す
	return func() {
		close(stopCh)
		time.Sleep(300 * time.Millisecond) // クリア処理の完了を待つ
		fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")
	}
}

// 強化されたインタラクティブ入力処理
func (s *Session) processEnhancedInteractiveInput(ctx context.Context, input string) (*interactive.InteractionResponse, error) {
	// 現在の実装では直接インタラクティブセッションを使用
	return s.interactiveSession.ProcessUserInput(ctx, s.currentSessionID, input)
}

// 強化された応答処理
func (s *Session) handleEnhancedResponse(response *interactive.InteractionResponse) error {
	// まず回答ヘッダーを表示
	s.printAssistantMessageHeader()

	// 応答タイプに応じて処理
	switch response.ResponseType {
	case interactive.ResponseTypeCodeSuggestion:
		return s.handleEnhancedCodeSuggestionResponse(response)
	case interactive.ResponseTypeConfirmation:
		return s.handleEnhancedConfirmationResponse(response)
	case interactive.ResponseTypeMessage:
		return s.handleEnhancedMessageResponse(response)
	default:
		return s.handleEnhancedMessageResponse(response)
	}
}

// 強化されたコード提案応答処理
func (s *Session) handleEnhancedCodeSuggestionResponse(response *interactive.InteractionResponse) error {
	if len(response.Suggestions) == 0 {
		return s.handleEnhancedMessageResponse(response)
	}

	suggestion := response.Suggestions[0]

	// メッセージがある場合は最初に表示
	if response.Message != "" {
		err := s.streamProcessor.StreamContent(response.Message)
		if err != nil {
			fmt.Print(response.Message)
		}
		fmt.Println()
	}

	// 強化された提案表示
	s.displayEnhancedCodeSuggestion(suggestion)

	// プロアクティブな関連情報を表示
	s.displayProactiveSuggestionContext(response)

	if response.RequiresConfirmation {
		confirmed, err := s.getUserConfirmation("この提案を適用しますか？")
		if err != nil {
			return err
		}

		err = s.interactiveSession.ConfirmSuggestion(s.currentSessionID, suggestion.ID, confirmed)
		if err != nil {
			return err
		}

		if confirmed {
			ctx := context.Background()
			err = s.interactiveSession.ApplySuggestion(ctx, s.currentSessionID, suggestion.ID)
			if err != nil {
				return fmt.Errorf("提案適用エラー: %w", err)
			}
			fmt.Printf("✅ 提案を適用しました！\n")
		}
	}

	return nil
}

// 強化された確認応答処理
func (s *Session) handleEnhancedConfirmationResponse(response *interactive.InteractionResponse) error {
	// メッセージがある場合は表示
	if response.Message != "" {
		err := s.streamProcessor.StreamContent(response.Message)
		if err != nil {
			fmt.Print(response.Message)
		}
		fmt.Println()
	}

	if response.RequiresConfirmation {
		confirmed, err := s.getUserConfirmation(response.Message)
		if err != nil {
			return err
		}

		if confirmed && len(response.Suggestions) > 0 {
			ctx := context.Background()
			return s.interactiveSession.ApplySuggestion(ctx, s.currentSessionID, response.Suggestions[0].ID)
		}
	}

	return nil
}

// 強化されたメッセージ応答処理
func (s *Session) handleEnhancedMessageResponse(response *interactive.InteractionResponse) error {
	// プロアクティブな前置き情報
	s.displayProactivePreContext(response)

	// メイン応答をストリーミング表示
	err := s.streamProcessor.StreamContent(response.Message)
	if err != nil {
		fmt.Print(response.Message)
	}

	// プロアクティブな後続き情報
	s.displayProactivePostContext(response)

	// コンテキストマネージャーに応答を追加
	if s.contextManager != nil {
		contextItem := &contextmanager.ContextItem{
			Type:       contextmanager.ContextTypeImmediate,
			Content:    fmt.Sprintf("アシスタント応答: %s", response.Message),
			Metadata:   map[string]string{"type": "assistant_response", "session_id": s.currentSessionID},
			Importance: 0.6,
		}
		s.contextManager.AddContext(contextItem)
	}

	return nil
}

// 強化されたコード提案表示
func (s *Session) displayEnhancedCodeSuggestion(suggestion *interactive.CodeSuggestion) {
	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	magenta := "\033[35m"
	reset := "\033[0m"

	fmt.Printf("\n%s💡 プロアクティブコード提案%s\n", cyan, reset)
	fmt.Printf("%s信頼度:%s %.1f%% ", green, reset, suggestion.Confidence*100)

	impactText := map[interactive.ImpactLevel]string{
		interactive.ImpactLevelLow:      "🟢 低影響",
		interactive.ImpactLevelMedium:   "🟡 中影響",
		interactive.ImpactLevelHigh:     "🟠 高影響",
		interactive.ImpactLevelCritical: "🔴 重大影響",
	}
	fmt.Printf("%s影響:%s %s\n", yellow, reset, impactText[suggestion.ImpactLevel])

	if suggestion.Explanation != "" {
		fmt.Printf("\n%s🔍 分析結果:%s %s\n", magenta, reset, suggestion.Explanation)
	}

	if suggestion.SuggestedCode != suggestion.OriginalCode {
		fmt.Printf("\n%s📝 提案コード:%s\n", green, reset)
		fmt.Printf("```\n%s\n```\n", suggestion.SuggestedCode)
	}
}

// プロアクティブ提案のコンテキストを表示
func (s *Session) displayProactiveSuggestionContext(response *interactive.InteractionResponse) {
	gray := "\033[90m"
	reset := "\033[0m"

	// メタデータからプロアクティブな情報を表示
	if response.Metadata != nil {
		if count, exists := response.Metadata["proactive_suggestions_count"]; exists {
			fmt.Printf("%s💭 他に%s個の提案があります%s\n", gray, count, reset)
		}

		if analyzed, exists := response.Metadata["project_analyzed"]; exists && analyzed == "true" {
			fmt.Printf("%s🔬 プロジェクト分析に基づく提案です%s\n", gray, reset)
		}
	}
}

// プロアクティブな前置きコンテキストを表示
func (s *Session) displayProactivePreContext(response *interactive.InteractionResponse) {
	// レスポンスにプロアクティブな前置き情報があれば表示
	if response.Metadata != nil {
		gray := "\033[90m"
		reset := "\033[0m"

		if projectInfo, exists := response.Metadata["project_context"]; exists {
			fmt.Printf("%s%s%s\n", gray, projectInfo, reset)
		}
	}
}

// プロアクティブな後続きコンテキストを表示
func (s *Session) displayProactivePostContext(response *interactive.InteractionResponse) {
	// レスポンスにプロアクティブな追加情報があれば表示
	if response.Metadata != nil {
		gray := "\033[90m"
		cyan := "\033[36m"
		reset := "\033[0m"

		if suggestions, exists := response.Metadata["related_suggestions"]; exists {
			fmt.Printf("\n%s💡 関連する提案:%s %s%s\n", cyan, reset, gray, suggestions)
		}

		if nextSteps, exists := response.Metadata["suggested_next_steps"]; exists {
			fmt.Printf("%s🎯 次のステップ:%s %s%s\n", cyan, reset, gray, nextSteps)
		}
	}
}
