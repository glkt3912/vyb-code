package interactive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/glkt/vyb-code/internal/ai"
	"github.com/glkt/vyb-code/internal/analysis"
	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/contextmanager"
	"github.com/glkt/vyb-code/internal/llm"
	"github.com/glkt/vyb-code/internal/reasoning"
	"github.com/glkt/vyb-code/internal/security"
	"github.com/glkt/vyb-code/internal/tools"
	"github.com/glkt/vyb-code/internal/ui"
)

// インタラクティブセッション管理実装
type interactiveSessionManager struct {
	mu                sync.RWMutex
	sessions          map[string]*InteractiveSession
	contextManager    contextmanager.ContextManager
	llmProvider       llm.Provider
	aiService         *ai.AIService    // AI機能統合サービス
	editTool          *tools.EditTool  // ファイル編集ツール
	writeTool         *tools.WriteTool // ファイル書き込みツール
	bashTool          *tools.BashTool  // コマンド実行ツール
	vibeConfig        *VibeConfig
	activeSessions    map[string]time.Time // セッション活性状況追跡
	sessionMetrics    map[string]*SessionMetrics
	conversationFlows map[string]*ConversationFlow
	proactiveExt      *ProactiveExtension // プロアクティブ拡張
	modelName         string              // 設定されたモデル名

	// 科学的認知分析システム統合
	cognitiveAnalyzer *analysis.CognitiveAnalyzer
	cognitiveEngine   *reasoning.CognitiveEngine
	config            *config.Config
}

// NewInteractiveSessionManager は新しいインタラクティブセッション管理を作成
func NewInteractiveSessionManager(
	contextManager contextmanager.ContextManager,
	llmProvider llm.Provider,
	aiService *ai.AIService,
	editTool *tools.EditTool,
	vibeConfig *VibeConfig,
	modelName string,
	cfg *config.Config,
) SessionManager {
	if vibeConfig == nil {
		vibeConfig = DefaultVibeConfig()
	}

	// WriteToolを初期化（セキュリティ制約とパス設定）
	writeTool := tools.NewWriteTool(
		security.NewDefaultConstraints("."), // デフォルト制約
		".",                                 // 現在のディレクトリ
		10*1024*1024,                        // 10MB制限
	)

	// BashToolを初期化（コマンド実行用）
	bashTool := tools.NewBashTool(
		security.NewDefaultConstraints("."), // デフォルト制約
		".",                                 // 現在のディレクトリ
	)

	manager := &interactiveSessionManager{
		sessions:          make(map[string]*InteractiveSession),
		contextManager:    contextManager,
		llmProvider:       llmProvider,
		aiService:         aiService,
		editTool:          editTool,
		writeTool:         writeTool,
		bashTool:          bashTool,
		vibeConfig:        vibeConfig,
		activeSessions:    make(map[string]time.Time),
		sessionMetrics:    make(map[string]*SessionMetrics),
		conversationFlows: make(map[string]*ConversationFlow),
		modelName:         modelName,
		config:            cfg,
	}

	// 科学的認知分析システム初期化
	if cfg != nil && llmProvider != nil {
		// LLMProviderをLLMClientとして使用（型アサーション）
		if llmClient, ok := llmProvider.(ai.LLMClient); ok {
			manager.cognitiveAnalyzer = analysis.NewCognitiveAnalyzer(cfg, llmClient)
			manager.cognitiveEngine = reasoning.NewCognitiveEngine(cfg, llmClient)
		}
	}

	// プロアクティブ拡張を初期化
	manager.proactiveExt = NewProactiveExtension(manager)

	return manager
}

// DefaultVibeConfig はデフォルトのバイブ設定を作成
func DefaultVibeConfig() *VibeConfig {
	return &VibeConfig{
		ResponseStyle:        ResponseStyleInteractive,
		Personality:          "helpful",
		TechnicalLevel:       7, // 中級〜上級開発者向け
		MaxResponseTime:      10 * time.Second,
		IdealResponseTime:    3 * time.Second,
		ThinkingTimeRatio:    0.2, // 20%の時間を思考に使用
		ProactiveSuggestions: true,
		SuggestionFrequency:  30 * time.Second,
		LearnUserPreferences: true,
		AdaptToWorkingStyle:  true,
	}
}

// CreateSession は新しいインタラクティブセッションを作成
func (ism *interactiveSessionManager) CreateSession(sessionType CodingSessionType) (*InteractiveSession, error) {
	ism.mu.Lock()
	defer ism.mu.Unlock()

	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	now := time.Now()

	session := &InteractiveSession{
		ID:              sessionID,
		State:           SessionStateIdle,
		Type:            sessionType,
		StartTime:       now,
		LastActivity:    now,
		CurrentFile:     "",
		CurrentFunction: "",
		CurrentLine:     0,
		UserIntent:      "",
		WorkingContext:  make([]*contextmanager.ContextItem, 0),
		SessionMetadata: make(map[string]string),
		Metrics: &SessionMetrics{
			TotalInteractions:     0,
			CodeSuggestionsGiven:  0,
			SuggestionsAccepted:   0,
			SuggestionsRejected:   0,
			FilesModified:         0,
			LinesChanged:          0,
			AverageResponseTime:   0,
			TotalThinkingTime:     0,
			UserSatisfactionScore: 0.8, // デフォルト満足度
		},
	}

	// セッションタイプに応じた初期設定
	switch sessionType {
	case CodingSessionTypeDebugging:
		session.SessionMetadata["focus"] = "debugging"
		session.SessionMetadata["priority"] = "bug_fixing"
	case CodingSessionTypeRefactor:
		session.SessionMetadata["focus"] = "refactoring"
		session.SessionMetadata["priority"] = "code_quality"
	case CodingSessionTypeReview:
		session.SessionMetadata["focus"] = "review"
		session.SessionMetadata["priority"] = "quality_assurance"
	case CodingSessionTypeLearning:
		session.SessionMetadata["focus"] = "learning"
		session.SessionMetadata["priority"] = "education"
	default:
		session.SessionMetadata["focus"] = "general"
		session.SessionMetadata["priority"] = "development"
	}

	ism.sessions[sessionID] = session
	ism.activeSessions[sessionID] = now
	ism.sessionMetrics[sessionID] = session.Metrics

	// 会話フローの初期化
	ism.conversationFlows[sessionID] = &ConversationFlow{
		CurrentStep:    FlowStep{StepType: FlowStepTypeUnderstanding, StartTime: now},
		StepHistory:    make([]FlowStep, 0),
		Progress:       0.0,
		EstimatedSteps: 5, // デフォルト予想ステップ数
		CompletedSteps: 0,
		NextSteps:      []string{"ユーザーの目標を理解する"},
		FlowMetadata:   make(map[string]string),
	}

	return session, nil
}

// GetSession は指定されたセッションを取得
func (ism *interactiveSessionManager) GetSession(sessionID string) (*InteractiveSession, error) {
	ism.mu.RLock()
	defer ism.mu.RUnlock()

	session, exists := ism.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("セッション %s が見つかりません", sessionID)
	}

	// アクティビティ更新
	ism.activeSessions[sessionID] = time.Now()
	return session, nil
}

// UpdateSession はセッション情報を更新
func (ism *interactiveSessionManager) UpdateSession(session *InteractiveSession) error {
	ism.mu.Lock()
	defer ism.mu.Unlock()

	if _, exists := ism.sessions[session.ID]; !exists {
		return fmt.Errorf("セッション %s が見つかりません", session.ID)
	}

	session.LastActivity = time.Now()
	ism.sessions[session.ID] = session
	ism.sessionMetrics[session.ID] = session.Metrics

	return nil
}

// CloseSession はセッションを終了
func (ism *interactiveSessionManager) CloseSession(sessionID string) error {
	ism.mu.Lock()
	defer ism.mu.Unlock()

	session, exists := ism.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション %s が見つかりません", sessionID)
	}

	// セッション終了処理
	session.State = SessionStateIdle
	session.LastActivity = time.Now()

	// 会話フロー完了
	if flow, flowExists := ism.conversationFlows[sessionID]; flowExists {
		now := time.Now()
		flow.CurrentStep.EndTime = &now
		flow.Progress = 1.0
		flow.CompletedSteps = flow.EstimatedSteps
	}

	// リソースクリア（但し、メトリクスは保持）
	delete(ism.sessions, sessionID)
	delete(ism.activeSessions, sessionID)
	delete(ism.conversationFlows, sessionID)

	return nil
}

// ListActiveSessions はアクティブなセッション一覧を取得
func (ism *interactiveSessionManager) ListActiveSessions() ([]*InteractiveSession, error) {
	ism.mu.RLock()
	defer ism.mu.RUnlock()

	activeSessions := make([]*InteractiveSession, 0)
	cutoffTime := time.Now().Add(-1 * time.Hour) // 1時間以内にアクティビティがあるセッション

	for sessionID, lastActivity := range ism.activeSessions {
		if lastActivity.After(cutoffTime) {
			if session, exists := ism.sessions[sessionID]; exists {
				activeSessions = append(activeSessions, session)
			}
		}
	}

	return activeSessions, nil
}

// UpdateWorkingContext は作業コンテキストを更新
func (ism *interactiveSessionManager) UpdateWorkingContext(
	sessionID string,
	contextItems []*contextmanager.ContextItem,
) error {
	ism.mu.Lock()
	defer ism.mu.Unlock()

	session, exists := ism.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション %s が見つかりません", sessionID)
	}

	// 現在の作業コンテキストをコンテキスト管理に追加
	for _, item := range contextItems {
		if err := ism.contextManager.AddContext(item); err != nil {
			return fmt.Errorf("コンテキスト追加エラー: %w", err)
		}
	}

	session.WorkingContext = contextItems
	session.LastActivity = time.Now()

	return nil
}

// GetRelevantContext は関連コンテキストを取得
func (ism *interactiveSessionManager) GetRelevantContext(
	sessionID string,
	query string,
	maxItems int,
) ([]*contextmanager.ContextItem, error) {
	session, err := ism.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	// セッションの現在のコンテキストを考慮したクエリ拡張
	enhancedQuery := ism.enhanceQueryWithSessionContext(session, query)

	return ism.contextManager.GetRelevantContext(enhancedQuery, maxItems)
}

// enhanceQueryWithSessionContext はセッションコンテキストでクエリを拡張
func (ism *interactiveSessionManager) enhanceQueryWithSessionContext(
	session *InteractiveSession,
	query string,
) string {
	contextTerms := []string{query}

	if session.CurrentFile != "" {
		contextTerms = append(contextTerms, session.CurrentFile)
	}
	if session.CurrentFunction != "" {
		contextTerms = append(contextTerms, session.CurrentFunction)
	}
	if session.UserIntent != "" {
		contextTerms = append(contextTerms, session.UserIntent)
	}

	// セッションタイプに応じた重み付け
	switch session.Type {
	case CodingSessionTypeDebugging:
		contextTerms = append(contextTerms, "debug", "error", "bug", "fix")
	case CodingSessionTypeRefactor:
		contextTerms = append(contextTerms, "refactor", "improve", "optimize", "restructure")
	case CodingSessionTypeReview:
		contextTerms = append(contextTerms, "review", "quality", "best practices", "standards")
	}

	return fmt.Sprintf("%s %s", query, fmt.Sprintf("context:%s",
		fmt.Sprintf("%v", contextTerms)))
}

// GenerateCodeSuggestion はコード提案を生成
func (ism *interactiveSessionManager) GenerateCodeSuggestion(
	ctx context.Context,
	sessionID string,
	request *SuggestionRequest,
) (*CodeSuggestion, error) {
	session, err := ism.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	session.State = SessionStateProcessing

	// 関連コンテキストを取得
	relevantContext, err := ism.GetRelevantContext(sessionID, request.UserDescription, 10)
	if err != nil {
		return nil, fmt.Errorf("関連コンテキスト取得エラー: %w", err)
	}

	// LLMプロンプトの構築
	prompt := ism.buildSuggestionPrompt(session, request, relevantContext)

	// LLM呼び出し
	chatReq := llm.ChatRequest{
		Model: ism.getConfiguredModel(), // 設定からモデルを取得
		Messages: []llm.ChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	response, err := ism.llmProvider.Chat(ctx, chatReq)
	if err != nil {
		session.State = SessionStateError
		return nil, fmt.Errorf("LLM応答生成エラー: %w", err)
	}

	// 応答からコード提案を抽出・構造化
	suggestion, err := ism.parseSuggestionResponse(response.Message.Content, request)
	if err != nil {
		return nil, fmt.Errorf("提案解析エラー: %w", err)
	}

	// 提案の信頼度とインパクト評価
	suggestion.Confidence = ism.calculateSuggestionConfidence(session, request, relevantContext)
	suggestion.ImpactLevel = ism.evaluateImpactLevel(request)

	// セッション状態更新
	session.State = SessionStateWaitingForConfirmation
	session.PendingSuggestion = suggestion
	session.Metrics.CodeSuggestionsGiven++
	session.Metrics.TotalInteractions++

	responseTime := time.Since(startTime)
	session.Metrics.AverageResponseTime = ism.updateAverageResponseTime(
		session.Metrics.AverageResponseTime,
		responseTime,
		session.Metrics.TotalInteractions,
	)

	return suggestion, nil
}

// buildSuggestionPrompt は提案用のプロンプトを構築
func (ism *interactiveSessionManager) buildSuggestionPrompt(
	session *InteractiveSession,
	request *SuggestionRequest,
	context []*contextmanager.ContextItem,
) string {
	prompt := fmt.Sprintf(`あなたはClaude Code相当のAIコーディングアシスタントです。以下の状況でコード提案を行ってください。

## セッション情報
- セッションタイプ: %s
- 現在のファイル: %s
- 現在の関数: %s
- ユーザーの意図: %s

## 提案リクエスト
- 提案タイプ: %s
- ファイルパス: %s
- 対象コード:
`+"```"+`
%s
`+"```"+`
- ユーザー説明: %s

## 関連コンテキスト
%s

## 要求事項
1. 具体的で実装可能なコード提案を提供
2. 変更理由と期待される効果を説明
3. 潜在的なリスクや副作用があれば言及
4. 信頼度 (0.0-1.0) を自己評価して含める

応答は以下の形式で：
**提案コード:**
`+"```"+`
[改善されたコード]
`+"```"+`

**説明:**
[変更の説明と理由]

**信頼度:** [0.0-1.0の数値]`,
		ism.sessionTypeToString(session.Type),
		session.CurrentFile,
		session.CurrentFunction,
		session.UserIntent,
		ism.suggestionTypeToString(request.Type),
		request.FilePath,
		request.Code,
		request.UserDescription,
		ism.formatContextForPrompt(context),
	)

	return prompt
}

// ConfirmSuggestion は提案の確認応答を処理
func (ism *interactiveSessionManager) ConfirmSuggestion(
	sessionID string,
	suggestionID string,
	accepted bool,
) error {
	ism.mu.Lock()
	defer ism.mu.Unlock()

	session, exists := ism.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション %s が見つかりません", sessionID)
	}

	if session.PendingSuggestion == nil {
		return fmt.Errorf("保留中の提案がありません")
	}

	// 提案IDが一致しない場合は警告を出すが処理を継続
	if session.PendingSuggestion.ID != suggestionID {
		fmt.Printf("Warning: ConfirmSuggestion ID不一致 (要求: %s, 実際: %s) - 保留中の提案を確認します\n",
			suggestionID, session.PendingSuggestion.ID)
	}

	session.PendingSuggestion.UserConfirmed = accepted // acceptedの値に応じて設定

	if accepted {
		session.State = SessionStateExecuting
		session.Metrics.SuggestionsAccepted++
	} else {
		session.State = SessionStateIdle
		session.Metrics.SuggestionsRejected++
		// 拒否された提案をクリア
		session.PendingSuggestion = nil
		fmt.Printf("❌ 提案が拒否されました\n")
	}

	session.LastActivity = time.Now()

	// ユーザー満足度の学習更新
	ism.updateUserSatisfactionScore(session, accepted)

	return nil
}

// ApplySuggestion は提案を実際に適用
func (ism *interactiveSessionManager) ApplySuggestion(
	ctx context.Context,
	sessionID string,
	suggestionID string,
) error {
	session, err := ism.GetSession(sessionID)
	if err != nil {
		return err
	}

	if session.PendingSuggestion == nil {
		return fmt.Errorf("保留中の提案がありません")
	}

	// 提案IDが一致しない場合は警告を出すが処理を継続
	if session.PendingSuggestion.ID != suggestionID {
		fmt.Printf("Warning: 提案ID不一致 (要求: %s, 実際: %s) - 保留中の提案を適用します\n",
			suggestionID, session.PendingSuggestion.ID)
	}

	if !session.PendingSuggestion.UserConfirmed {
		return fmt.Errorf("提案が確認されていません: %s", suggestionID)
	}

	// 提案内容に基づいて適切な処理を実行
	suggestedCode := session.PendingSuggestion.SuggestedCode

	// コマンド実行の判定（```bash や git コマンドを含む場合）
	if ism.isCommandSuggestion(suggestedCode) {
		fmt.Printf("Debug: コマンド実行開始\n")

		// コマンドを抽出してBashToolで実行
		command := ism.extractCommandFromSuggestion(suggestedCode)
		if command != "" {
			fmt.Printf("Debug: 実行コマンド: %s\n", command)

			// BashToolでコマンド実行
			if ism.bashTool != nil {
				result, err := ism.bashTool.Execute(command, "ユーザー要求によるコマンド実行", 30000) // 30秒タイムアウト
				if err != nil {
					session.State = SessionStateError
					return fmt.Errorf("コマンド実行エラー: %v", err)
				}

				fmt.Printf("Debug: コマンド実行結果:\n%s\n", result.Content)
				session.LastCommandOutput = result.Content
			} else {
				return fmt.Errorf("BashToolが利用できません")
			}
		} else {
			return fmt.Errorf("実行可能なコマンドが見つかりません")
		}
	} else if ism.editTool != nil {
		// ファイル操作の場合（従来の処理）
		filePath := session.PendingSuggestion.FilePath
		if filePath == "" {
			// PendingSuggestionのメタデータから元の入力を取得
			originalInput := session.PendingSuggestion.Metadata["original_input"]
			if originalInput == "" {
				originalInput = "main.go" // デフォルト
			}
			filePath = ism.extractFilePathFromInput(originalInput)
		}

		if filePath != "" {
			if session.PendingSuggestion.OriginalCode == "" {
				// 新規ファイル作成
				fmt.Printf("Debug: ファイル作成中: %s\n", filePath)
				writeRequest := tools.WriteRequest{
					FilePath: filePath,
					Content:  suggestedCode,
				}

				result, err := ism.writeTool.Write(writeRequest)
				if err != nil || result.IsError {
					session.State = SessionStateError
					return fmt.Errorf("ファイル作成エラー: %v", err)
				}

				// 詳細な成功メッセージを表示
				absPath, err := filepath.Abs(filePath)
				if err != nil {
					absPath = filePath
				}
				fmt.Printf("✅ ファイルを作成しました: %s\n", absPath)
				fmt.Printf("📁 作成場所: %s\n", filepath.Dir(absPath))
				fmt.Printf("📄 内容: %d bytes\n", len(suggestedCode))
				session.LastCommandOutput = fmt.Sprintf("ファイル作成完了: %s (%d bytes)", absPath, len(suggestedCode))
			} else {
				// 既存ファイル編集
				editRequest := tools.EditRequest{
					FilePath:  filePath,
					OldString: session.PendingSuggestion.OriginalCode,
					NewString: suggestedCode,
				}

				result, err := ism.editTool.Edit(editRequest)
				if err != nil || result.IsError {
					session.State = SessionStateError
					return fmt.Errorf("ファイル編集エラー: %v", err)
				}
			}
		} else {
			return fmt.Errorf("ファイルパスが特定できません")
		}
	}

	session.PendingSuggestion.Applied = true
	session.State = SessionStateIdle
	session.Metrics.FilesModified++

	// 変更行数の概算更新
	originalLines := len(strings.Split(session.PendingSuggestion.OriginalCode, "\n"))
	suggestedLines := len(strings.Split(session.PendingSuggestion.SuggestedCode, "\n"))
	session.Metrics.LinesChanged += abs(suggestedLines - originalLines)

	session.PendingSuggestion = nil
	session.LastActivity = time.Now()

	return nil
}

// ProcessUserInput はユーザー入力を処理してインタラクティブな応答を生成
func (ism *interactiveSessionManager) ProcessUserInput(
	ctx context.Context,
	sessionID string,
	input string,
) (*InteractionResponse, error) {
	// プロアクティブ拡張が利用可能で、分析能力が必要な場合は使用
	if ism.proactiveExt != nil && ism.shouldUseProactiveExtension(input) {
		return ism.proactiveExt.EnhanceProcessUserInput(ctx, sessionID, input)
	}

	// 通常の処理を実行
	return ism.processUserInputFallback(ctx, sessionID, input)
}

// processUserInputFallback はフォールバック用のユーザー入力処理
func (ism *interactiveSessionManager) processUserInputFallback(
	ctx context.Context,
	sessionID string,
	input string,
) (*InteractionResponse, error) {
	session, err := ism.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	session.State = SessionStateProcessing
	startTime := time.Now()

	// 入力の意図解析
	intent, err := ism.analyzeUserIntent(ctx, session, input)
	if err != nil {
		return nil, fmt.Errorf("意図解析エラー: %w", err)
	}

	session.UserIntent = intent

	// SmartContextManagerを活用してユーザー入力をコンテキストに追加
	ism.addToSmartContext(sessionID, input, "user_input")

	// CognitiveEngineを活用した高度な推論処理
	reasoningInsights := ism.performCognitiveReasoning(ctx, input, intent)

	// 推論結果をセッションメタデータに追加
	if session.SessionMetadata == nil {
		session.SessionMetadata = make(map[string]string)
	}
	if reasoningInsights["status"] == "reasoning_completed" {
		session.SessionMetadata["reasoning_confidence"] = fmt.Sprintf("%.2f", reasoningInsights["confidence"])
		session.SessionMetadata["reasoning_processing_time"] = fmt.Sprintf("%v", reasoningInsights["processing_time"])
		if reasoningInsights["solution_approach"] != nil {
			session.SessionMetadata["reasoning_approach"] = reasoningInsights["solution_approach"].(string)
		}
	}

	// 確認応答の処理チェック
	trimmedInput := strings.TrimSpace(strings.ToLower(input))
	if (trimmedInput == "y" || trimmedInput == "yes" || trimmedInput == "はい" || trimmedInput == "ok") && session.PendingSuggestion != nil {
		// 提案確認処理
		fmt.Printf("Debug: 提案確認受理 - ID: %s\n", session.PendingSuggestion.ID)

		err = ism.ConfirmSuggestion(sessionID, session.PendingSuggestion.ID, true)
		if err != nil {
			session.State = SessionStateError
			return nil, fmt.Errorf("提案確認エラー: %w", err)
		}

		err = ism.ApplySuggestion(ctx, sessionID, session.PendingSuggestion.ID)
		if err != nil {
			session.State = SessionStateError
			return nil, fmt.Errorf("提案適用エラー: %w", err)
		}

		// 確認完了応答を生成
		response := &InteractionResponse{
			SessionID:            sessionID,
			ResponseType:         ResponseTypeCompletion,
			Message:              "✅ 提案を適用しました！",
			RequiresConfirmation: false,
			Metadata: map[string]string{
				"action":        "suggestion_applied",
				"suggestion_id": session.PendingSuggestion.ID,
				"file_path":     session.PendingSuggestion.FilePath,
			},
			GeneratedAt: time.Now(),
		}

		// 提案をクリア
		session.PendingSuggestion = nil
		session.State = SessionStateWaitingForInput
		session.LastActivity = time.Now()

		return response, nil
	}

	// 会話フローの進行
	err = ism.advanceConversationFlow(session, intent)
	if err != nil {
		return nil, fmt.Errorf("会話フロー進行エラー: %w", err)
	}

	// コンテキストに入力を追加
	contextItem := &contextmanager.ContextItem{
		Type:       contextmanager.ContextTypeImmediate,
		Content:    fmt.Sprintf("ユーザー入力: %s\n意図: %s", input, intent),
		Metadata:   map[string]string{"type": "user_input", "session_id": sessionID},
		Importance: 0.7,
	}

	err = ism.contextManager.AddContext(contextItem)
	if err != nil {
		return nil, fmt.Errorf("コンテキスト追加エラー: %w", err)
	}

	// 応答生成
	response, err := ism.generateInteractiveResponse(ctx, session, input, intent)
	if err != nil {
		session.State = SessionStateError
		return nil, fmt.Errorf("応答生成エラー: %w", err)
	}

	// メトリクス更新
	responseTime := time.Since(startTime)
	session.Metrics.TotalInteractions++
	session.Metrics.AverageResponseTime = ism.updateAverageResponseTime(
		session.Metrics.AverageResponseTime,
		responseTime,
		session.Metrics.TotalInteractions,
	)

	session.State = SessionStateWaitingForInput
	session.LastActivity = time.Now()

	return response, nil
}

// GetSessionState はセッション状態を取得
func (ism *interactiveSessionManager) GetSessionState(sessionID string) (SessionState, error) {
	session, err := ism.GetSession(sessionID)
	if err != nil {
		return SessionStateError, err
	}
	return session.State, nil
}

// UpdateSessionState はセッション状態を更新
func (ism *interactiveSessionManager) UpdateSessionState(sessionID string, state SessionState) error {
	ism.mu.Lock()
	defer ism.mu.Unlock()

	session, exists := ism.sessions[sessionID]
	if !exists {
		return fmt.Errorf("セッション %s が見つかりません", sessionID)
	}

	session.State = state
	session.LastActivity = time.Now()

	return nil
}

// GetSessionMetrics はセッションメトリクスを取得
func (ism *interactiveSessionManager) GetSessionMetrics(sessionID string) (*SessionMetrics, error) {
	ism.mu.RLock()
	defer ism.mu.RUnlock()

	metrics, exists := ism.sessionMetrics[sessionID]
	if !exists {
		return nil, fmt.Errorf("セッション %s のメトリクスが見つかりません", sessionID)
	}

	return metrics, nil
}

// UpdateSessionMetrics はセッションメトリクスを更新
func (ism *interactiveSessionManager) UpdateSessionMetrics(sessionID string, metrics *SessionMetrics) error {
	ism.mu.Lock()
	defer ism.mu.Unlock()

	if _, exists := ism.sessions[sessionID]; !exists {
		return fmt.Errorf("セッション %s が見つかりません", sessionID)
	}

	ism.sessionMetrics[sessionID] = metrics
	ism.sessions[sessionID].Metrics = metrics

	return nil
}

// GetSuggestionHistory は提案履歴を取得
func (ism *interactiveSessionManager) GetSuggestionHistory(sessionID string) ([]*CodeSuggestion, error) {
	// 提案履歴の永続化は将来のバージョンで実装予定
	// 現在は簡単な実装
	session, err := ism.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	history := make([]*CodeSuggestion, 0)
	if session.PendingSuggestion != nil {
		history = append(history, session.PendingSuggestion)
	}

	return history, nil
}

// buildInteractivePrompt はClaude Code式統一プロンプトを構築
func (ism *interactiveSessionManager) buildInteractivePrompt(session *InteractiveSession, input string, intent string) string {
	// SmartContextManagerから最適化されたコンテキストを取得（70-95%圧縮効率）
	optimizedContext := ism.getOptimizedContext(session.ID, input, 50)

	// セッション履歴を取得して文脈を構築
	contextHistory := ism.buildSessionContext(session)

	// ベースプロンプトを構築 - 構造化応答を強制
	basePrompt := fmt.Sprintf(`あなたは vyb AIコーディングアシスタントです。Claude Code のような連続的なコーディング体験を提供してください。

## 🚨 CRITICAL: 構造化応答の必須使用
**あなたは必ず以下の構造化タグを使用してください。これは絶対の要求です:**

### 必須パターン判定:
- 分析・状況確認 → <ANALYSIS>詳細な分析クエリ</ANALYSIS> を最優先で使用
- コマンド実行 → <COMMAND>command_here</COMMAND>
- ファイル作成 → <FILECREATE>path/file.ext|content</FILECREATE>
- ファイル読み取り → <FILEREAD>filename.ext</FILEREAD>
- 次の提案 → <SUGGESTION>具体的な次のアクション</SUGGESTION>

## 🛠 Available Tools (構造化タグ必須)
1. <ANALYSIS>query</ANALYSIS> - プロジェクト/コード分析 (分析系質問では絶対必須)
2. <COMMAND>command</COMMAND> - Bashコマンド実行
3. <FILECREATE>path|content</FILECREATE> - ファイル作成
4. <FILEREAD>filename</FILEREAD> - ファイル読み取り
5. <SUGGESTION>action</SUGGESTION> - 次の作業提案

## 🔄 Session Context & History
- Project: %s
- Current File: %s
- User Intent: %s
- Last Command Output: %s

### Optimized Context (SmartContextManager - 70-95%% compression):
%s

### Recent Session History:
%s

## 📝 User Request
%s

## 📋 Action Plan:
1. 適切な構造化タグで実行
2. 結果を分析
3. 次のステップを提案

**必須実行例:**

ユーザー要求: "git statusを実行"
→ 必須応答: <COMMAND>git status</COMMAND>

ユーザー要求: "現状を分析"
→ 必須応答: <ANALYSIS>プロジェクト状況の詳細分析</ANALYSIS>

ユーザー要求: "ファイルを作成"
→ 必須応答: <FILECREATE>filename.ext|content here</FILECREATE>

🚨 **CRITICAL**: あなたの応答は必ずこれらのタグを含む必要があります。タグなしの応答は許可されません。`,
		ism.sessionTypeToString(session.Type),
		session.CurrentFile,
		intent,
		session.LastCommandOutput,
		optimizedContext,
		contextHistory,
		input,
	)

	// プロアクティブ拡張が利用可能な場合、プロンプトを拡張
	if ism.proactiveExt != nil {
		enhancedPrompt := ism.proactiveExt.EnhancePrompt(basePrompt, input)
		return enhancedPrompt
	}

	return basePrompt
}

// buildSessionContext はセッション履歴から文脈を構築
func (ism *interactiveSessionManager) buildSessionContext(session *InteractiveSession) string {
	if session.Metrics.TotalInteractions == 0 {
		return "新しいセッションです。"
	}

	context := fmt.Sprintf(`
- 総インタラクション数: %d
- 受け入れられた提案: %d / %d
- 変更されたファイル: %d
- 変更された行数: %d`,
		session.Metrics.TotalInteractions,
		session.Metrics.SuggestionsAccepted,
		session.Metrics.CodeSuggestionsGiven,
		session.Metrics.FilesModified,
		session.Metrics.LinesChanged,
	)

	// 最後の作業内容があれば追加
	if session.LastCommandOutput != "" {
		context += "\n- 最後の実行結果: " + session.LastCommandOutput[:min(200, len(session.LastCommandOutput))] + "..."
	}

	return context
}

// min は二つの整数の最小値を返す
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// normalizeLanguage はLLM応答の言語を日本語に統一
func (ism *interactiveSessionManager) normalizeLanguage(content string) string {
	// 繁体字・簡体字の一般的なパターンを日本語に変換
	replacements := map[string]string{
		"创建文件": "ファイル作成",
		"創建文件": "ファイル作成",
		"创建":   "作成",
		"創建":   "作成",
		"文件":   "ファイル",
		"執行":   "実行",
		"执行":   "実行",
		"運行":   "実行",
		"运行":   "実行",
		"開始":   "開始",
		"开始":   "開始",
		"完成":   "完了",
		"成功":   "成功",
		"失敗":   "失敗",
		"失败":   "失敗",
		"錯誤":   "エラー",
		"错误":   "エラー",
		"檔案":   "ファイル",
		"目錄":   "ディレクトリ",
		"目录":   "ディレクトリ",
		"資料夾":  "フォルダ",
		"资料夹":  "フォルダ",
	}

	result := content
	for chinese, japanese := range replacements {
		result = strings.ReplaceAll(result, chinese, japanese)
	}

	return result
}

// parseAndExecuteStructuredResponse はLLM応答を解析して実際のツール実行を行う
func (ism *interactiveSessionManager) parseAndExecuteStructuredResponse(
	ctx context.Context,
	session *InteractiveSession,
	llmResponse string,
	originalInput string,
) (*InteractionResponse, error) {
	var allResults []string
	var executedActions []string

	// 1. コマンド実行パターンをチェック
	commandRegex := regexp.MustCompile(`<COMMAND>(.*?)</COMMAND>`)
	commandMatches := commandRegex.FindAllStringSubmatch(llmResponse, -1)

	if len(commandMatches) > 0 {
		for _, match := range commandMatches {
			if len(match) > 1 {
				command := strings.TrimSpace(match[1])
				result, err := ism.executeBashCommand(ctx, session, command)
				if err != nil {
					allResults = append(allResults, fmt.Sprintf("⚠️ コマンドエラー: %v", err))
				} else {
					// git diff の場合は要約版を使用
					if strings.Contains(command, "git diff") {
						summarizedResult := ism.summarizeGitDiff(result)
						allResults = append(allResults, fmt.Sprintf("✅ `%s`:\n%s", command, summarizedResult))
					} else {
						allResults = append(allResults, fmt.Sprintf("✅ `%s`:\n%s", command, result))
					}
				}
				executedActions = append(executedActions, fmt.Sprintf("コマンド実行: %s", command))
			}
		}
	}

	// 2. ファイル作成パターンをチェック
	fileCreateRegex := regexp.MustCompile(`<FILECREATE>(.*?)\|(.*?)</FILECREATE>`)
	fileMatches := fileCreateRegex.FindAllStringSubmatch(llmResponse, -1)

	if len(fileMatches) > 0 {
		for _, match := range fileMatches {
			if len(match) > 2 {
				filePath := strings.TrimSpace(match[1])
				content := strings.TrimSpace(match[2])
				err := ism.createFile(ctx, session, filePath, content)
				if err != nil {
					allResults = append(allResults, fmt.Sprintf("⚠️ ファイル作成エラー (%s): %v", filePath, err))
				} else {
					allResults = append(allResults, fmt.Sprintf("✅ ファイル作成成功: %s", filePath))
				}
				executedActions = append(executedActions, fmt.Sprintf("ファイル作成: %s", filePath))
			}
		}
	}

	// 3. ファイル読み取りパターンをチェック
	fileReadRegex := regexp.MustCompile(`<FILEREAD>(.*?)</FILEREAD>`)
	readMatches := fileReadRegex.FindAllStringSubmatch(llmResponse, -1)

	if len(readMatches) > 0 {
		for _, match := range readMatches {
			if len(match) > 1 {
				filePath := strings.TrimSpace(match[1])
				content, err := ism.readFile(ctx, session, filePath)
				if err != nil {
					allResults = append(allResults, fmt.Sprintf("⚠️ ファイル読み取りエラー (%s): %v", filePath, err))
				} else {
					// 内容が長すぎる場合は省略
					displayContent := content
					if len(content) > 500 {
						displayContent = content[:500] + "...(省略)"
					}
					allResults = append(allResults, fmt.Sprintf("📄 %s:\n%s", filePath, displayContent))
				}
				executedActions = append(executedActions, fmt.Sprintf("ファイル読み込み: %s", filePath))
			}
		}
	}

	// 4. 分析パターンをチェック
	analysisRegex := regexp.MustCompile(`<ANALYSIS>(.*?)</ANALYSIS>`)
	analysisMatches := analysisRegex.FindAllStringSubmatch(llmResponse, -1)

	if len(analysisMatches) > 0 {
		for _, match := range analysisMatches {
			if len(match) > 1 {
				query := strings.TrimSpace(match[1])
				result := ism.performAnalysis(session, query)
				allResults = append(allResults, fmt.Sprintf("🔍 分析結果:\n%s", result))
				executedActions = append(executedActions, fmt.Sprintf("分析実行: %s", query))
			}
		}
	}

	// 5. 提案パターンをチェック
	suggestionRegex := regexp.MustCompile(`<SUGGESTION>(.*?)</SUGGESTION>`)
	suggestionMatches := suggestionRegex.FindAllStringSubmatch(llmResponse, -1)

	var suggestions []string
	if len(suggestionMatches) > 0 {
		for _, match := range suggestionMatches {
			if len(match) > 1 {
				suggestion := strings.TrimSpace(match[1])
				suggestions = append(suggestions, suggestion)
				executedActions = append(executedActions, fmt.Sprintf("提案: %s", suggestion))
			}
		}
	}

	// 何らかのアクションが実行された場合、統合された応答を生成
	if len(executedActions) > 0 {
		cleanMessage := ism.extractCleanMessage(llmResponse)

		// 実行結果をまとめる
		var responseMessage strings.Builder
		responseMessage.WriteString(cleanMessage)

		if len(allResults) > 0 {
			responseMessage.WriteString("\n\n🔄 **実行結果:**\n")
			responseMessage.WriteString(strings.Join(allResults, "\n\n"))
		}

		// 提案があれば追加
		if len(suggestions) > 0 {
			responseMessage.WriteString("\n\n💡 **次のステップの提案:**\n• ")
			responseMessage.WriteString(strings.Join(suggestions, "\n• "))
		}

		// 連続体験のための次のステップ提案を追加
		nextStepPrompt := ism.generateNextStepSuggestion(executedActions, allResults)
		if nextStepPrompt != "" {
			responseMessage.WriteString("\n\n")
			responseMessage.WriteString(nextStepPrompt)
		}

		response := &InteractionResponse{
			SessionID:            session.ID,
			ResponseType:         ResponseTypeMessage,
			Message:              responseMessage.String(),
			RequiresConfirmation: false,
			GeneratedAt:          time.Now(),
			Metadata: map[string]string{
				"actions_count":    fmt.Sprintf("%d", len(executedActions)),
				"has_executions":   "true",
				"executed_actions": strings.Join(executedActions, ", "),
			},
		}
		return response, nil
	}

	// 構造化されたパターンが見つからない場合はnilを返す（通常処理に戻る）
	return nil, nil
}

// extractCleanMessage は構造化タグを除去したメッセージを抽出
func (ism *interactiveSessionManager) extractCleanMessage(content string) string {
	// 構造化タグを除去
	content = regexp.MustCompile(`<COMMAND>.*?</COMMAND>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<FILECREATE>.*?</FILECREATE>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<FILEREAD>.*?</FILEREAD>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<ANALYSIS>.*?</ANALYSIS>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<SUGGESTION>.*?</SUGGESTION>`).ReplaceAllString(content, "")

	// 改行を整理
	content = strings.TrimSpace(content)
	if content == "" {
		return "実行しました。"
	}

	return content
}

// executeBashCommand は実際にBashコマンドを実行
func (ism *interactiveSessionManager) executeBashCommand(ctx context.Context, session *InteractiveSession, command string) (string, error) {
	if ism.bashTool == nil {
		return "", fmt.Errorf("BashToolが初期化されていません")
	}

	result, err := ism.bashTool.Execute(command, "Interactive command execution", 30000) // 30秒タイムアウト
	if err != nil {
		return "", fmt.Errorf("コマンド実行エラー: %w", err)
	}

	session.LastCommandOutput = result.Content
	return result.Content, nil
}

// createFile は実際にファイルを作成
func (ism *interactiveSessionManager) createFile(ctx context.Context, session *InteractiveSession, filePath string, content string) error {
	if ism.writeTool == nil {
		return fmt.Errorf("WriteToolが初期化されていません")
	}

	// WriteRequestを作成
	writeReq := tools.WriteRequest{
		FilePath: filePath,
		Content:  content,
	}

	result, err := ism.writeTool.Write(writeReq)
	if err != nil {
		return fmt.Errorf("ファイル作成エラー: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("ファイル作成失敗: %s", result.Content)
	}

	return nil
}

// readFile は実際にファイルを読み取り
func (ism *interactiveSessionManager) readFile(ctx context.Context, session *InteractiveSession, filePath string) (string, error) {
	// ReadToolがない場合は、BashToolでcatコマンドを使用
	if ism.bashTool == nil {
		return "", fmt.Errorf("ファイル読み取りツールが初期化されていません")
	}

	result, err := ism.bashTool.Execute(fmt.Sprintf("cat %s", filePath), "Read file content", 10000)
	if err != nil {
		return "", fmt.Errorf("ファイル読み取りエラー: %w", err)
	}

	if result.IsError {
		return "", fmt.Errorf("ファイル読み取り失敗: %s", result.Content)
	}

	return result.Content, nil
}

// isDangerousFileOperation は危険なファイル操作かどうかを判定
func (ism *interactiveSessionManager) isDangerousFileOperation(suggestion *CodeSuggestion) bool {
	filePath := suggestion.FilePath
	content := suggestion.SuggestedCode

	// 危険なファイルパス
	dangerousPaths := []string{
		"/etc/",
		"/usr/",
		"/var/",
		"/root/",
		"/home/",
		"../",      // ディレクトリトラバーサル
		"./../../", // ディレクトリトラバーサル
	}

	for _, dangerous := range dangerousPaths {
		if strings.Contains(filePath, dangerous) {
			return true
		}
	}

	// 危険なファイル拡張子
	dangerousExtensions := []string{
		".sh",
		".bash",
		".exe",
		".bat",
		".cmd",
		".com",
		".scr",
		".vbs",
		".ps1",
	}

	for _, ext := range dangerousExtensions {
		if strings.HasSuffix(filePath, ext) {
			return true
		}
	}

	// 危険な内容
	dangerousContent := []string{
		"rm -rf",
		"sudo ",
		"chmod 777",
		"del /f",
		"format ",
		"dd if=",
		"DROP TABLE",
		"DELETE FROM",
	}

	lowerContent := strings.ToLower(content)
	for _, dangerous := range dangerousContent {
		if strings.Contains(lowerContent, strings.ToLower(dangerous)) {
			return true
		}
	}

	// 基本的なファイル作成は安全
	return false
}

// generateFallbackResponse はLLM失敗時のフォールバック応答を生成
func (ism *interactiveSessionManager) generateFallbackResponse(session *InteractiveSession, input string, intent string, err error) (*InteractionResponse, error) {
	fallbackMessage := fmt.Sprintf("申し訳ございませんが、AI応答の生成に失敗しました。\n\n要求内容: %s\n意図: %s\n\n基本的な支援は可能ですので、より具体的な質問をお試しください。\n\nエラー詳細: %v", input, intent, err)

	return &InteractionResponse{
		SessionID:            session.ID,
		ResponseType:         ResponseTypeMessage,
		Message:              fallbackMessage,
		RequiresConfirmation: false,
		Metadata: map[string]string{
			"fallback": "true",
			"error":    err.Error(),
			"intent":   intent,
		},
		GeneratedAt: time.Now(),
	}, nil
}

// performAnalysis は科学的認知分析システムを使用した高度な分析処理を実行
func (ism *interactiveSessionManager) performAnalysis(_ *InteractiveSession, query string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var analysisComponents []string

	// 1. 既存の統合分析システムを活用
	unifiedAnalysisResult := ism.performUnifiedAnalysis(query)
	if unifiedAnalysisResult != "" {
		analysisComponents = append(analysisComponents, unifiedAnalysisResult)
	}

	// 2. 高度な認知分析の実行
	if ism.cognitiveAnalyzer != nil {
		cognitiveResult := ism.performDetailedCognitiveAnalysis(ctx, query)
		if cognitiveResult != "" {
			analysisComponents = append(analysisComponents, cognitiveResult)
		}
	}

	// 3. プロジェクト構造の詳細分析
	if projectPath, err := os.Getwd(); err == nil {
		structureAnalysis := ism.performProjectStructureAnalysisImpl(projectPath)
		if structureAnalysis != "" {
			analysisComponents = append(analysisComponents, structureAnalysis)
		}
	}

	// 4. Git分析の詳細化
	gitAnalysis := ism.performDetailedGitAnalysis()
	if gitAnalysis != "" {
		analysisComponents = append(analysisComponents, gitAnalysis)
	}

	// 5. 依存関係とセキュリティ分析
	securityAnalysis := ism.performSecurityAnalysisImpl()
	if securityAnalysis != "" {
		analysisComponents = append(analysisComponents, securityAnalysis)
	}

	// 6. フォールバック
	if len(analysisComponents) == 0 {
		analysisComponents = append(analysisComponents, ism.performBasicAnalysis(query))
	}

	// 分析結果の統合フォーマット
	return ism.formatComprehensiveAnalysisResponse(query, analysisComponents)
}

// performScientificCognitiveAnalysis は科学的認知分析を実行
func (ism *interactiveSessionManager) performScientificCognitiveAnalysis(query string) string {
	if ism.cognitiveAnalyzer == nil {
		return ""
	}

	ctx := context.Background()

	// 分析リクエストを作成
	analysisRequest := &analysis.AnalysisRequest{
		UserInput: query,
		Response:  "", // 初期分析時は空
		Context: map[string]interface{}{
			"project_type":  "go",
			"analysis_type": "project_analysis",
		},
		AnalysisDepth:   "standard",
		RequiredMetrics: []string{"confidence", "reasoning_depth", "creativity"},
	}

	// 認知分析を実行
	result, err := ism.cognitiveAnalyzer.AnalyzeCognitive(ctx, analysisRequest)
	if err != nil {
		return fmt.Sprintf("🧠 科学的認知分析: エラー (%v)", err)
	}

	// 結果をフォーマット
	return ism.formatCognitiveAnalysisResult(result)
}

// formatCognitiveAnalysisResult は認知分析結果をフォーマット
func (ism *interactiveSessionManager) formatCognitiveAnalysisResult(result *analysis.CognitiveAnalysisResult) string {
	var parts []string

	parts = append(parts, "🧠 **科学的認知分析結果**")

	// セマンティックエントロピーによる信頼度
	if result.Confidence != nil {
		parts = append(parts, fmt.Sprintf("📊 信頼度: %.2f (セマンティックエントロピー: %.3f)",
			result.Confidence.OverallConfidence, result.Confidence.SemanticEntropy))
	}

	// 推論深度分析
	if result.ReasoningDepth != nil {
		parts = append(parts, fmt.Sprintf("🔬 推論深度: %d (論理構造スコア: %.3f)",
			result.ReasoningDepth.OverallDepth, result.ReasoningDepth.ComplexityScore))
	}

	// 創造性測定
	if result.Creativity != nil {
		parts = append(parts, fmt.Sprintf("🎨 創造性: %.2f (流暢性: %.2f, 柔軟性: %.2f, 独創性: %.2f)",
			result.Creativity.OverallScore,
			result.Creativity.Fluency,
			result.Creativity.Flexibility,
			result.Creativity.Originality))
	}

	// 統合評価
	parts = append(parts, fmt.Sprintf("⚡ 統合品質スコア: %.2f", result.OverallQuality))
	parts = append(parts, fmt.Sprintf("🔒 信頼スコア: %.2f", result.TrustScore))

	// 推奨アクション
	if len(result.RecommendedActions) > 0 {
		parts = append(parts, "📋 推奨アクション:")
		for _, action := range result.RecommendedActions {
			parts = append(parts, fmt.Sprintf("  • %s", action))
		}
	}

	return strings.Join(parts, "\n")
}

// collectBasicProjectInfo は基本的なプロジェクト情報を収集
func (ism *interactiveSessionManager) collectBasicProjectInfo() string {
	var info []string

	if ism.bashTool != nil {
		// Git状態
		if result, err := ism.bashTool.Execute("git status --porcelain", "Git status", 3000); err == nil && !result.IsError {
			fileCount := len(strings.Split(strings.TrimSpace(result.Content), "\n"))
			if strings.TrimSpace(result.Content) != "" {
				info = append(info, fmt.Sprintf("📊 Git状態: %d個のファイルに変更", fileCount))
			} else {
				info = append(info, "📊 Git状態: クリーン")
			}
		}

		// プロジェクト規模
		if result, err := ism.bashTool.Execute("find . -name '*.go' -type f | wc -l", "Go files count", 3000); err == nil && !result.IsError {
			info = append(info, fmt.Sprintf("🏗️ プロジェクト規模: %s個のGoファイル", strings.TrimSpace(result.Content)))
		}
	}

	if len(info) > 0 {
		return strings.Join(info, "\n")
	}

	return ""
}

// performBasicAnalysis はフォールバック用の基本分析
func (ism *interactiveSessionManager) performBasicAnalysis(query string) string {
	return fmt.Sprintf("🔍 基本分析: %s\n（科学的認知分析システムが利用できません）\n💡 基本的なプロジェクト状態を確認しました", query)
}

// performUnifiedAnalysis は既存のUnifiedAnalyzerを活用した統合分析を実行
func (ism *interactiveSessionManager) performUnifiedAnalysis(query string) string {
	if ism.config == nil {
		return ""
	}

	// UnifiedAnalyzerを初期化
	if llmClient, ok := ism.llmProvider.(ai.LLMClient); ok {
		unifiedAnalyzer := analysis.NewUnifiedAnalyzer(ism.config, llmClient)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// プロジェクト分析を実行
		projectAnalysis, err := unifiedAnalyzer.AnalyzeProject(ctx, ".")
		if err != nil {
			return fmt.Sprintf("🔬 統合分析エラー: %v", err)
		}

		// 分析結果をフォーマット
		return ism.formatProjectAnalysis(projectAnalysis)
	}

	return ""
}

// formatProjectAnalysis はプロジェクト分析結果をフォーマット
func (ism *interactiveSessionManager) formatProjectAnalysis(analysis *analysis.ProjectAnalysis) string {
	if analysis == nil {
		return ""
	}

	var result []string

	// プロジェクト基本情報
	result = append(result, fmt.Sprintf("📋 **プロジェクト概要**"))
	result = append(result, fmt.Sprintf("  • 名前: %s", analysis.ProjectName))
	result = append(result, fmt.Sprintf("  • 言語: %s", analysis.Language))
	result = append(result, fmt.Sprintf("  • フレームワーク: %s", analysis.Framework))

	// ファイル構造
	if analysis.FileStructure != nil {
		result = append(result, fmt.Sprintf("🏗️ **プロジェクト構造**"))
		result = append(result, fmt.Sprintf("  • 総ファイル数: %d", analysis.FileStructure.TotalFiles))
		result = append(result, fmt.Sprintf("  • 総行数: %s", ism.formatNumber(analysis.FileStructure.TotalLines)))

		if len(analysis.FileStructure.Languages) > 0 {
			result = append(result, "  • 言語別ファイル数:")
			for lang, count := range analysis.FileStructure.Languages {
				result = append(result, fmt.Sprintf("    - %s: %d ファイル", lang, count))
			}
		}
	}

	// 品質メトリクス
	if analysis.QualityMetrics != nil {
		result = append(result, fmt.Sprintf("📊 **コード品質**"))
		result = append(result, fmt.Sprintf("  • テストカバレッジ: %.1f%%", analysis.QualityMetrics.TestCoverage*100))
		result = append(result, fmt.Sprintf("  • 保守性: %.1f/10", analysis.QualityMetrics.Maintainability*10))
		result = append(result, fmt.Sprintf("  • 複雑度: %.1f", analysis.QualityMetrics.CodeComplexity))
		if analysis.QualityMetrics.IssueCount > 0 {
			result = append(result, fmt.Sprintf("  • ⚠️ 検出された問題: %d件", analysis.QualityMetrics.IssueCount))
		}
	}

	// 依存関係
	if len(analysis.Dependencies) > 0 {
		result = append(result, fmt.Sprintf("📦 **依存関係 (%d件)**", len(analysis.Dependencies)))
		outdatedCount := 0
		vulnerableCount := 0
		for _, dep := range analysis.Dependencies {
			if dep.Outdated {
				outdatedCount++
			}
			if len(dep.Vulnerabilities) > 0 {
				vulnerableCount++
			}
		}
		if outdatedCount > 0 {
			result = append(result, fmt.Sprintf("  • ⚠️ 古いバージョン: %d件", outdatedCount))
		}
		if vulnerableCount > 0 {
			result = append(result, fmt.Sprintf("  • 🔒 セキュリティ問題: %d件", vulnerableCount))
		}
	}

	// セキュリティ問題
	if len(analysis.SecurityIssues) > 0 {
		result = append(result, fmt.Sprintf("🔒 **セキュリティ問題 (%d件)**", len(analysis.SecurityIssues)))
		for _, issue := range analysis.SecurityIssues {
			result = append(result, fmt.Sprintf("  • %s: %s", issue.Type, issue.Description))
		}
	}

	// 改善提案
	if len(analysis.Recommendations) > 0 {
		result = append(result, fmt.Sprintf("💡 **改善提案 (%d件)**", len(analysis.Recommendations)))
		for i, rec := range analysis.Recommendations {
			if i < 5 { // 最初の5件のみ表示
				result = append(result, fmt.Sprintf("  • %s: %s", rec.Type, rec.Description))
			}
		}
		if len(analysis.Recommendations) > 5 {
			result = append(result, fmt.Sprintf("  • ... および%d件の追加提案", len(analysis.Recommendations)-5))
		}
	}

	return strings.Join(result, "\n")
}

// formatNumber は数値を読みやすい形式でフォーマット
func (ism *interactiveSessionManager) formatNumber(num int) string {
	if num < 1000 {
		return fmt.Sprintf("%d", num)
	} else if num < 1000000 {
		return fmt.Sprintf("%.1fk", float64(num)/1000)
	} else {
		return fmt.Sprintf("%.1fM", float64(num)/1000000)
	}
}

// performDetailedGitAnalysis は詳細なGit分析を実行
func (ism *interactiveSessionManager) performDetailedGitAnalysis() string {
	if ism.bashTool == nil {
		return ""
	}

	var gitResults []string

	// Git状態の詳細分析
	if result, err := ism.bashTool.Execute("git status --porcelain -b", "Git detailed status", 5000); err == nil && !result.IsError {
		lines := strings.Split(strings.TrimSpace(result.Content), "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[0], "##") {
			branchInfo := strings.TrimPrefix(lines[0], "## ")
			gitResults = append(gitResults, fmt.Sprintf("🌿 **ブランチ**: %s", branchInfo))
		}

		modifiedCount := 0
		addedCount := 0
		deletedCount := 0
		for _, line := range lines[1:] {
			if len(line) >= 2 {
				status := line[:2]
				if strings.Contains(status, "M") {
					modifiedCount++
				}
				if strings.Contains(status, "A") {
					addedCount++
				}
				if strings.Contains(status, "D") {
					deletedCount++
				}
			}
		}

		if modifiedCount+addedCount+deletedCount > 0 {
			gitResults = append(gitResults, fmt.Sprintf("📝 **変更統計**: 変更 %d, 追加 %d, 削除 %d", modifiedCount, addedCount, deletedCount))
		}
	}

	// コミット履歴分析
	if result, err := ism.bashTool.Execute("git log --oneline -10 --no-merges", "Recent commits", 5000); err == nil && !result.IsError {
		commitLines := strings.Split(strings.TrimSpace(result.Content), "\n")
		if len(commitLines) > 0 {
			gitResults = append(gitResults, fmt.Sprintf("📋 **最近のコミット** (%d件):", len(commitLines)))
			for i, commit := range commitLines {
				if i < 3 { // 最新3件のみ表示
					gitResults = append(gitResults, fmt.Sprintf("  • %s", commit))
				}
			}
		}
	}

	// ブランチ分析
	if result, err := ism.bashTool.Execute("git branch -a", "Git branches", 3000); err == nil && !result.IsError {
		branches := strings.Split(strings.TrimSpace(result.Content), "\n")
		localBranches := 0
		remoteBranches := 0
		for _, branch := range branches {
			branch = strings.TrimSpace(branch)
			if strings.HasPrefix(branch, "remotes/") {
				remoteBranches++
			} else if branch != "" {
				localBranches++
			}
		}
		gitResults = append(gitResults, fmt.Sprintf("🌳 **ブランチ**: ローカル %d, リモート %d", localBranches, remoteBranches))
	}

	if len(gitResults) > 0 {
		return strings.Join(gitResults, "\n")
	}

	return ""
}

// addToSmartContext はSmartContextManagerを活用してコンテキストを追加
func (ism *interactiveSessionManager) addToSmartContext(sessionID, content, contentType string) {
	if ism.contextManager == nil {
		return
	}

	// コンテキスト項目を作成
	contextItem := &contextmanager.ContextItem{
		Content:    content,
		Type:       contextmanager.ContextTypeImmediate, // 最新の情報として即座コンテキストに追加
		Importance: 0.8,                                 // デフォルト重要度
		Timestamp:  time.Now(),
		LastAccess: time.Now(),
		Metadata: map[string]string{
			"session_id":   sessionID,
			"content_type": contentType,
		},
	}

	// SmartContextManagerに追加
	err := ism.contextManager.AddContext(contextItem)
	if err != nil {
		// エラーログ（実運用では適切なロガーを使用）
		fmt.Printf("コンテキスト追加エラー: %v\n", err)
	}
}

// getOptimizedContext はSmartContextManagerから最適化されたコンテキストを取得
func (ism *interactiveSessionManager) getOptimizedContext(sessionID, query string, maxItems int) string {
	if ism.contextManager == nil {
		return "（コンテキスト管理利用不可）"
	}

	// 関連コンテキストを取得
	contextItems, err := ism.contextManager.GetRelevantContext(query, maxItems)
	if err != nil {
		return "（コンテキスト取得エラー）"
	}

	// コンテキストアイテムがない場合
	if len(contextItems) == 0 {
		return "（コンテキストなし）"
	}

	// コンテキスト圧縮（70-95%効率）
	compressedContext, err := ism.contextManager.CompressContext(true) // forceCompress = true
	if err != nil {
		// 圧縮失敗時はそのまま使用
		var contents []string
		for _, item := range contextItems {
			if item != nil && item.Content != "" {
				contents = append(contents, item.Content)
			}
		}
		if len(contents) == 0 {
			return "（有効なコンテキストなし）"
		}
		return strings.Join(contents, "\n")
	}

	// 圧縮結果がnilでないことを確認
	if compressedContext == nil {
		return "（コンテキスト圧縮結果なし）"
	}

	// サマリーとキーポイントを組み合わせて返す
	var result []string
	if compressedContext.Summary != "" {
		result = append(result, compressedContext.Summary)
	}
	if len(compressedContext.KeyPoints) > 0 {
		result = append(result, "主要ポイント:")
		for _, point := range compressedContext.KeyPoints {
			if point != "" {
				result = append(result, "• "+point)
			}
		}
	}

	if len(result) == 0 {
		return "（コンテキスト処理結果なし）"
	}

	return strings.Join(result, "\n")
}

// performCognitiveReasoning はCognitiveEngineを活用した高度な推論を実行
func (ism *interactiveSessionManager) performCognitiveReasoning(ctx context.Context, input, intent string) map[string]interface{} {
	if ism.cognitiveEngine == nil {
		return map[string]interface{}{
			"status": "cognitive_engine_unavailable",
		}
	}

	// タイムアウト付きで推論を実行
	reasoningCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// 推論を実行
	reasoningResult, err := ism.cognitiveEngine.ProcessUserInput(reasoningCtx, input)
	if err != nil {
		return map[string]interface{}{
			"status": "reasoning_failed",
			"error":  err.Error(),
		}
	}

	// 推論結果を分析
	insights := map[string]interface{}{
		"status":          "reasoning_completed",
		"confidence":      reasoningResult.Confidence,
		"processing_time": reasoningResult.ProcessingTime,
	}

	// 推論チェーンの情報を追加
	if len(reasoningResult.InferenceChains) > 0 {
		insights["inference_chains_count"] = len(reasoningResult.InferenceChains)
		// 推論チェーンの詳細は必要に応じて追加
	}

	// 学習的洞察があれば追加
	if len(reasoningResult.Insights) > 0 {
		insights["learning_insights_count"] = len(reasoningResult.Insights)
	}

	// 選択された解決策の情報
	if reasoningResult.SelectedSolution != nil {
		insights["solution_approach"] = reasoningResult.SelectedSolution.Approach
		insights["solution_confidence"] = reasoningResult.SelectedSolution.Confidence
	}

	return insights
}

// generateNextStepSuggestion は実行結果を分析して具体的な次のステップ提案を生成
func (ism *interactiveSessionManager) generateNextStepSuggestion(executedActions []string, results []string) string {
	if len(executedActions) == 0 {
		return ""
	}

	var suggestions []string

	// 実行結果を分析して具体的な提案を生成
	for i, action := range executedActions {
		if i < len(results) {
			result := results[i]

			// git status 分析
			if strings.Contains(action, "git status") {
				gitSuggestions := ism.analyzeGitStatus(result)
				suggestions = append(suggestions, gitSuggestions...)
			} else if strings.Contains(action, "git diff") {
				// git diff 分析
				diffSuggestions := ism.analyzeGitDiff(result)
				suggestions = append(suggestions, diffSuggestions...)
			} else if strings.Contains(action, "ファイル作成") || strings.Contains(action, "ファイル読み込み") {
				// ファイル作成/読み込み分析
				fileSuggestions := ism.analyzeFileOperations(action, result)
				suggestions = append(suggestions, fileSuggestions...)
			} else if strings.Contains(result, "エラー") || strings.Contains(result, "失敗") {
				// コマンド実行エラー分析
				errorSuggestions := ism.analyzeErrors(result)
				suggestions = append(suggestions, errorSuggestions...)
			} else if strings.Contains(action, "go build") || strings.Contains(action, "make") {
				// ビルド結果分析
				buildSuggestions := ism.analyzeBuildResults(result)
				suggestions = append(suggestions, buildSuggestions...)
			}
		}
	}

	// 重複除去
	uniqueSuggestions := ism.removeDuplicateSuggestions(suggestions)

	if len(uniqueSuggestions) > 0 {
		return fmt.Sprintf("💡 **具体的な次のステップ:**\n• %s", strings.Join(uniqueSuggestions, "\n• "))
	}

	return "💡 **次のステップ:** 他にご質問や作業があればお聞かせください。"
}

// analyzeGitStatus はgit statusの結果を分析して具体的提案を生成
func (ism *interactiveSessionManager) analyzeGitStatus(gitOutput string) []string {
	var suggestions []string

	// 変更されたファイル数をカウント
	modifiedFiles := strings.Count(gitOutput, "modified:")
	untrackedDirs := strings.Count(gitOutput, "/")

	if strings.Contains(gitOutput, "Changes not staged for commit") {
		if modifiedFiles > 10 {
			suggestions = append(suggestions, fmt.Sprintf("多数の変更ファイル(%d個)があります。`git add .` で一括ステージング", modifiedFiles))
		} else if modifiedFiles > 0 {
			suggestions = append(suggestions, "変更されたファイルを確認し、必要に応じて `git add <ファイル名>` でステージング")
		}

		// 変更内容の詳細分析を提案
		suggestions = append(suggestions, "実際の変更内容を確認: `git diff` で詳細な差分を表示")
		suggestions = append(suggestions, "特定ファイルの変更内容を確認: `git diff <ファイル名>`")
	}

	if strings.Contains(gitOutput, "Untracked files") {
		if untrackedDirs > 0 {
			suggestions = append(suggestions, "新しいディレクトリが追加されています。内容を確認して `git add` でトラッキング")
		}
		suggestions = append(suggestions, "未追跡ファイルの内容を確認し、必要に応じてGitに追加")
	}

	if strings.Contains(gitOutput, "no changes added to commit") {
		suggestions = append(suggestions, "変更をステージング後、`git commit -m \"説明\"` でコミット作成")
	}

	if strings.Contains(gitOutput, "feature/") {
		suggestions = append(suggestions, "機能ブランチでの作業中です。完了後はメインブランチへのマージを検討")
	}

	return suggestions
}

// analyzeGitDiff はgit diffの結果を分析してコード変更の意味を理解
func (ism *interactiveSessionManager) analyzeGitDiff(diffOutput string) []string {
	var suggestions []string

	if strings.TrimSpace(diffOutput) == "" {
		suggestions = append(suggestions, "変更はありません。新しい機能の実装を検討しましょう")
		return suggestions
	}

	// 変更の意味を分析
	changeAnalysis := ism.analyzeChangeSemantics(diffOutput)

	// 分析結果に基づく智能的な提案を生成
	suggestions = ism.generateSemanticSuggestions(changeAnalysis)

	return suggestions
}

// ChangeSemantics は変更の意味を表現する構造体
type ChangeSemantics struct {
	ChangeType         string                 // 変更の種類（feature, fix, refactor, docs等）
	AffectedAreas      []string               // 影響を受ける領域
	RiskLevel          string                 // リスクレベル（low, medium, high, critical）
	TestRequirements   []string               // 必要なテスト
	ReviewPoints       []string               // レビューすべき点
	Dependencies       []string               // 影響する依存関係
	ArchitectureImpact string                 // アーキテクチャへの影響
	QualityMetrics     map[string]interface{} // 品質メトリクス
}

// analyzeChangeSemantics は変更の意味を分析
func (ism *interactiveSessionManager) analyzeChangeSemantics(diffOutput string) *ChangeSemantics {
	analysis := &ChangeSemantics{
		AffectedAreas:    []string{},
		TestRequirements: []string{},
		ReviewPoints:     []string{},
		Dependencies:     []string{},
		QualityMetrics:   make(map[string]interface{}),
	}

	// 1. 変更の種類を判定
	analysis.ChangeType = ism.detectChangeType(diffOutput)

	// 2. 影響領域の分析
	analysis.AffectedAreas = ism.identifyAffectedAreas(diffOutput)

	// 3. リスクレベルの評価
	analysis.RiskLevel = ism.evaluateRiskLevel(diffOutput, analysis.AffectedAreas)

	// 4. テスト要件の特定
	analysis.TestRequirements = ism.identifyTestRequirements(diffOutput, analysis.ChangeType)

	// 5. レビューポイントの抽出
	analysis.ReviewPoints = ism.extractReviewPoints(diffOutput, analysis.ChangeType, analysis.RiskLevel)

	// 6. 依存関係への影響
	analysis.Dependencies = ism.analyzeDependencyImpact(diffOutput)

	// 7. アーキテクチャ影響の評価
	analysis.ArchitectureImpact = ism.evaluateArchitectureImpact(diffOutput, analysis.AffectedAreas)

	return analysis
}

// detectChangeType は変更の種類を検出
func (ism *interactiveSessionManager) detectChangeType(diffOutput string) string {
	// 新機能追加の検出
	if strings.Contains(diffOutput, "+func New") || strings.Contains(diffOutput, "+type ") ||
		strings.Contains(diffOutput, "+// API:") {
		return "feature"
	}

	// バグ修正の検出
	if strings.Contains(diffOutput, "fix") || strings.Contains(diffOutput, "bug") ||
		strings.Contains(diffOutput, "-\t\treturn err") || strings.Contains(diffOutput, "+\t\treturn fmt.Errorf") {
		return "fix"
	}

	// リファクタリングの検出
	if strings.Contains(diffOutput, "-func ") && strings.Contains(diffOutput, "+func ") {
		return "refactor"
	}

	// ドキュメント更新の検出
	if strings.Contains(diffOutput, "README.md") || strings.Contains(diffOutput, "CLAUDE.md") ||
		strings.Contains(diffOutput, "+//") {
		return "docs"
	}

	// テスト追加の検出
	if strings.Contains(diffOutput, "_test.go") || strings.Contains(diffOutput, "+\tfunc Test") {
		return "test"
	}

	// 設定変更の検出
	if strings.Contains(diffOutput, "config") || strings.Contains(diffOutput, ".json") ||
		strings.Contains(diffOutput, ".yaml") {
		return "config"
	}

	return "general"
}

// identifyAffectedAreas は影響を受ける領域を特定
func (ism *interactiveSessionManager) identifyAffectedAreas(diffOutput string) []string {
	areas := []string{}

	if strings.Contains(diffOutput, "internal/llm/") {
		areas = append(areas, "LLM統合レイヤー")
	}
	if strings.Contains(diffOutput, "internal/handlers/chat.go") {
		areas = append(areas, "チャット機能")
	}
	if strings.Contains(diffOutput, "internal/interactive/") {
		areas = append(areas, "インタラクティブ機能")
	}
	if strings.Contains(diffOutput, "internal/tools/") {
		areas = append(areas, "ツール機能")
	}
	if strings.Contains(diffOutput, "internal/config/") {
		areas = append(areas, "設定システム")
	}
	if strings.Contains(diffOutput, "internal/security/") {
		areas = append(areas, "セキュリティ層")
	}
	if strings.Contains(diffOutput, "cmd/vyb/") {
		areas = append(areas, "CLI インターフェース")
	}

	return areas
}

// evaluateRiskLevel はリスクレベルを評価
func (ism *interactiveSessionManager) evaluateRiskLevel(diffOutput string, affectedAreas []string) string {
	// 重要なファイルの変更
	if strings.Contains(diffOutput, "internal/security/") ||
		strings.Contains(diffOutput, "password") || strings.Contains(diffOutput, "auth") {
		return "critical"
	}

	// 複数の重要領域に影響
	if len(affectedAreas) >= 3 {
		return "high"
	}

	// コア機能への影響
	if strings.Contains(diffOutput, "internal/llm/") ||
		strings.Contains(diffOutput, "internal/handlers/chat.go") {
		return "medium"
	}

	// ドキュメントやテストのみ
	if strings.Contains(diffOutput, "README.md") || strings.Contains(diffOutput, "_test.go") {
		return "low"
	}

	return "medium"
}

// identifyTestRequirements はテスト要件を特定
func (ism *interactiveSessionManager) identifyTestRequirements(diffOutput string, changeType string) []string {
	requirements := []string{}

	switch changeType {
	case "feature":
		requirements = append(requirements, "新機能の単体テスト作成")
		requirements = append(requirements, "統合テストシナリオの追加")
		requirements = append(requirements, "エッジケースのテスト")
	case "fix":
		requirements = append(requirements, "バグ再現テストの作成")
		requirements = append(requirements, "リグレッションテストの実行")
	case "refactor":
		requirements = append(requirements, "既存テストの動作確認")
		requirements = append(requirements, "パフォーマンステストの実行")
	case "config":
		requirements = append(requirements, "設定値のバリデーションテスト")
		requirements = append(requirements, "異常設定時の動作確認")
	}

	// コード内容に基づく追加テスト
	if strings.Contains(diffOutput, "error") {
		requirements = append(requirements, "エラーハンドリングのテスト")
	}
	if strings.Contains(diffOutput, "concurrent") || strings.Contains(diffOutput, "goroutine") {
		requirements = append(requirements, "並行処理の競合状態テスト")
	}

	return requirements
}

// extractReviewPoints はレビューポイントを抽出
func (ism *interactiveSessionManager) extractReviewPoints(diffOutput string, changeType string, riskLevel string) []string {
	points := []string{}

	// リスクレベル別のレビューポイント
	switch riskLevel {
	case "critical":
		points = append(points, "セキュリティ影響の詳細確認")
		points = append(points, "権限昇格の可能性チェック")
		points = append(points, "データ漏洩リスクの評価")
	case "high":
		points = append(points, "アーキテクチャ設計との整合性確認")
		points = append(points, "パフォーマンス影響の測定")
		points = append(points, "後方互換性の保証")
	case "medium":
		points = append(points, "コード品質と可読性の確認")
		points = append(points, "エラーハンドリングの適切性")
	case "low":
		points = append(points, "ドキュメントの正確性確認")
		points = append(points, "コードスタイルの統一")
	}

	// 特定パターンのレビューポイント
	if strings.Contains(diffOutput, "+import") {
		points = append(points, "新しい依存関係の必要性と安全性")
	}
	if strings.Contains(diffOutput, "TODO") || strings.Contains(diffOutput, "FIXME") {
		points = append(points, "残存する技術的負債の対応計画")
	}

	return points
}

// analyzeDependencyImpact は依存関係への影響を分析
func (ism *interactiveSessionManager) analyzeDependencyImpact(diffOutput string) []string {
	dependencies := []string{}

	if strings.Contains(diffOutput, "go.mod") {
		dependencies = append(dependencies, "Go モジュール依存関係の更新")
	}
	if strings.Contains(diffOutput, "+import") {
		dependencies = append(dependencies, "新規パッケージ依存の追加")
	}
	if strings.Contains(diffOutput, "internal/llm/") {
		dependencies = append(dependencies, "LLM プロバイダー統合への影響")
	}
	if strings.Contains(diffOutput, "internal/tools/") {
		dependencies = append(dependencies, "ツールチェーン統合への影響")
	}

	return dependencies
}

// evaluateArchitectureImpact はアーキテクチャ影響を評価
func (ism *interactiveSessionManager) evaluateArchitectureImpact(diffOutput string, affectedAreas []string) string {
	if len(affectedAreas) >= 4 {
		return "システム全体アーキテクチャに重大な変更。設計レビュー必須"
	}

	if strings.Contains(diffOutput, "interface") && strings.Contains(diffOutput, "+") {
		return "新しいインターフェース追加。契約設計の確認が必要"
	}

	if strings.Contains(diffOutput, "internal/") && len(affectedAreas) >= 2 {
		return "内部アーキテクチャの結合度に影響。モジュール境界の再検討推奨"
	}

	if strings.Contains(diffOutput, "config") {
		return "設定アーキテクチャの変更。運用環境への影響確認必要"
	}

	return "局所的変更。アーキテクチャ影響は限定的"
}

// generateSemanticSuggestions は意味解析に基づく提案を生成
func (ism *interactiveSessionManager) generateSemanticSuggestions(analysis *ChangeSemantics) []string {
	suggestions := []string{}

	// 変更タイプ別の主要メッセージ
	switch analysis.ChangeType {
	case "feature":
		suggestions = append(suggestions, "🚀 新機能開発: "+strings.Join(analysis.AffectedAreas, ", ")+"への機能追加を検出")
	case "fix":
		suggestions = append(suggestions, "🔧 バグ修正: 品質向上のための修正を実施")
	case "refactor":
		suggestions = append(suggestions, "♻️ リファクタリング: コード品質改善を検出")
	case "docs":
		suggestions = append(suggestions, "📚 ドキュメント更新: プロジェクト情報の最新化")
	case "config":
		suggestions = append(suggestions, "⚙️ 設定変更: システム動作に影響する設定の変更")
	default:
		suggestions = append(suggestions, "🔄 一般的な変更: "+strings.Join(analysis.AffectedAreas, ", ")+"の更新")
	}

	// リスクレベル別の警告
	switch analysis.RiskLevel {
	case "critical":
		suggestions = append(suggestions, "⚠️ 【重要】クリティカル変更: 慎重なレビューとテストが必須")
	case "high":
		suggestions = append(suggestions, "⚡ 高影響変更: 十分なテストとレビューを実施")
	case "medium":
		suggestions = append(suggestions, "📋 中程度の影響: 標準的なレビュープロセスを実施")
	}

	// アーキテクチャ影響
	if analysis.ArchitectureImpact != "局所的変更。アーキテクチャ影響は限定的" {
		suggestions = append(suggestions, "🏗️ アーキテクチャ影響: "+analysis.ArchitectureImpact)
	}

	// テスト要件
	if len(analysis.TestRequirements) > 0 {
		suggestions = append(suggestions, "🧪 推奨テスト: "+strings.Join(analysis.TestRequirements, ", "))
	}

	// レビューポイント
	if len(analysis.ReviewPoints) > 0 {
		suggestions = append(suggestions, "👁️ レビューポイント: "+strings.Join(analysis.ReviewPoints, ", "))
	}

	// 依存関係への影響
	if len(analysis.Dependencies) > 0 {
		suggestions = append(suggestions, "🔗 依存関係: "+strings.Join(analysis.Dependencies, ", "))
	}

	// 次ステップの提案
	suggestions = append(suggestions, "✅ 推奨次ステップ: 変更内容確認後、適切なテスト実行とコードレビュー実施")

	return suggestions
}

// summarizeGitDiff はgit diffの出力を詳細分析して要約する
func (ism *interactiveSessionManager) summarizeGitDiff(diffOutput string) string {
	if strings.TrimSpace(diffOutput) == "" {
		return "変更はありません。"
	}

	// 詳細分析を実行
	analysis := ism.performDetailedDiffAnalysis(diffOutput)

	// 結果をフォーマット
	summary := fmt.Sprintf("📊 **変更サマリー**\n")
	summary += fmt.Sprintf("• ファイル数: %d個  ", len(analysis.ChangedFiles))
	summary += fmt.Sprintf("• 変更規模: +%d行, -%d行  ", analysis.AddedLines, analysis.DeletedLines)
	summary += fmt.Sprintf("• リスクレベル: %s\n", ism.formatRiskLevel(analysis.RiskLevel))

	// ファイル別詳細情報
	if len(analysis.FileSummaries) > 0 {
		summary += "\n📝 **変更ファイル詳細:**\n"
		for i, fileSummary := range analysis.FileSummaries {
			if i >= 6 { // 最大6個まで詳細表示
				summary += fmt.Sprintf("• ... その他 %d個のファイル\n", len(analysis.FileSummaries)-6)
				break
			}

			icon := ism.getFileTypeIcon(fileSummary.Path)
			summary += fmt.Sprintf("• %s **%s** (+%d/-%d行) %s\n",
				icon, fileSummary.Path, fileSummary.AddedLines, fileSummary.DeletedLines, fileSummary.ChangeType)

			// 重要な変更内容を表示
			if len(fileSummary.KeyChanges) > 0 {
				for _, change := range fileSummary.KeyChanges[:min(2, len(fileSummary.KeyChanges))] {
					summary += fmt.Sprintf("  └ %s\n", change)
				}
			}
		}
	}

	// 影響度分析
	if len(analysis.ImpactAreas) > 0 {
		summary += "\n🎯 **影響領域:**\n"
		for _, area := range analysis.ImpactAreas {
			summary += fmt.Sprintf("• %s %s\n", area.Icon, area.Description)
		}
	}

	// 具体的な技術的変更
	if len(analysis.TechnicalChanges) > 0 {
		summary += "\n🔧 **技術的変更:**\n"
		for _, change := range analysis.TechnicalChanges {
			summary += fmt.Sprintf("• %s\n", change)
		}
	}

	// セキュリティ・品質の注意点
	if len(analysis.SecurityConcerns) > 0 || len(analysis.QualityIssues) > 0 {
		summary += "\n⚠️ **要注意:**\n"
		for _, concern := range analysis.SecurityConcerns {
			summary += fmt.Sprintf("• 🔐 %s\n", concern)
		}
		for _, issue := range analysis.QualityIssues {
			summary += fmt.Sprintf("• 📊 %s\n", issue)
		}
	}

	// パフォーマンス影響
	if analysis.PerformanceImpact != "" {
		summary += fmt.Sprintf("\n⚡ **パフォーマンス影響:** %s\n", analysis.PerformanceImpact)
	}

	summary += "\n💡 個別ファイルの詳細: `git diff <ファイル名>` | 全diff確認: `git diff --no-pager`"

	return summary
}

// DetailedDiffAnalysis は詳細なdiff分析結果
type DetailedDiffAnalysis struct {
	ChangedFiles      []string
	AddedLines        int
	DeletedLines      int
	RiskLevel         string
	FileSummaries     []FileSummary
	ImpactAreas       []ImpactArea
	TechnicalChanges  []string
	SecurityConcerns  []string
	QualityIssues     []string
	PerformanceImpact string
}

// FileSummary はファイル別の変更サマリー
type FileSummary struct {
	Path         string
	AddedLines   int
	DeletedLines int
	ChangeType   string
	KeyChanges   []string
}

// ImpactArea は影響領域
type ImpactArea struct {
	Icon        string
	Description string
}

// performDetailedDiffAnalysis は詳細なdiff分析を実行
func (ism *interactiveSessionManager) performDetailedDiffAnalysis(diffOutput string) *DetailedDiffAnalysis {
	analysis := &DetailedDiffAnalysis{
		ChangedFiles:     ism.extractChangedFilesFromDiff(diffOutput),
		AddedLines:       strings.Count(diffOutput, "\n+") - strings.Count(diffOutput, "\n+++"),
		DeletedLines:     strings.Count(diffOutput, "\n-") - strings.Count(diffOutput, "\n---"),
		FileSummaries:    []FileSummary{},
		ImpactAreas:      []ImpactArea{},
		TechnicalChanges: []string{},
		SecurityConcerns: []string{},
		QualityIssues:    []string{},
	}

	// リスクレベル評価
	analysis.RiskLevel = ism.calculateRiskLevel(analysis.AddedLines, analysis.DeletedLines, analysis.ChangedFiles)

	// ファイル別分析
	analysis.FileSummaries = ism.analyzeIndividualFiles(diffOutput, analysis.ChangedFiles)

	// 影響領域の特定
	analysis.ImpactAreas = ism.identifyImpactAreas(analysis.ChangedFiles, diffOutput)

	// 技術的変更の抽出
	analysis.TechnicalChanges = ism.extractTechnicalChanges(diffOutput)

	// セキュリティ・品質チェック
	analysis.SecurityConcerns = ism.identifySecurityConcerns(diffOutput)
	analysis.QualityIssues = ism.identifyQualityIssues(diffOutput, analysis)

	// パフォーマンス影響評価
	analysis.PerformanceImpact = ism.evaluatePerformanceImpact(diffOutput, analysis.ChangedFiles)

	return analysis
}

// calculateRiskLevel はリスクレベルを計算
func (ism *interactiveSessionManager) calculateRiskLevel(addedLines, deletedLines int, changedFiles []string) string {
	totalChange := addedLines + deletedLines

	// 重要なファイルのチェック
	hasSecurityFile := false
	hasCoreFile := false
	for _, file := range changedFiles {
		if strings.Contains(file, "security") || strings.Contains(file, "auth") {
			hasSecurityFile = true
		}
		if strings.Contains(file, "main.go") || strings.Contains(file, "session.go") {
			hasCoreFile = true
		}
	}

	if hasSecurityFile || totalChange > 500 {
		return "🔴 HIGH"
	} else if hasCoreFile || totalChange > 200 || len(changedFiles) > 8 {
		return "🟡 MEDIUM"
	}
	return "🟢 LOW"
}

// analyzeIndividualFiles はファイル別の詳細分析
func (ism *interactiveSessionManager) analyzeIndividualFiles(diffOutput string, changedFiles []string) []FileSummary {
	summaries := []FileSummary{}

	for _, file := range changedFiles {
		// ファイル別の変更行数を計算
		fileSection := ism.extractFileSection(diffOutput, file)
		addedLines := strings.Count(fileSection, "\n+") - strings.Count(fileSection, "\n+++")
		deletedLines := strings.Count(fileSection, "\n-") - strings.Count(fileSection, "\n---")

		// 変更タイプを判定
		changeType := ism.determineChangeType(fileSection, file)

		// 主要な変更を抽出
		keyChanges := ism.extractKeyChanges(fileSection, file)

		summaries = append(summaries, FileSummary{
			Path:         file,
			AddedLines:   addedLines,
			DeletedLines: deletedLines,
			ChangeType:   changeType,
			KeyChanges:   keyChanges,
		})
	}

	return summaries
}

// extractFileSection はファイル別のdiff部分を抽出
func (ism *interactiveSessionManager) extractFileSection(diffOutput, fileName string) string {
	lines := strings.Split(diffOutput, "\n")
	var fileLines []string
	inFile := false

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") && strings.Contains(line, fileName) {
			inFile = true
			fileLines = []string{line}
		} else if strings.HasPrefix(line, "diff --git") && inFile {
			break
		} else if inFile {
			fileLines = append(fileLines, line)
		}
	}

	return strings.Join(fileLines, "\n")
}

// determineChangeType は変更タイプを判定
func (ism *interactiveSessionManager) determineChangeType(fileSection, fileName string) string {
	if strings.Contains(fileSection, "+func New") {
		return "新機能追加"
	} else if strings.Contains(fileSection, "+type ") && strings.Contains(fileSection, "struct") {
		return "構造拡張"
	} else if strings.Contains(fileSection, "test") {
		return "テスト更新"
	} else if strings.Contains(fileName, "config") {
		return "設定変更"
	} else if strings.Contains(fileName, ".md") {
		return "ドキュメント"
	} else if strings.Contains(fileSection, "-") && strings.Contains(fileSection, "+") {
		return "リファクタ"
	} else if strings.Count(fileSection, "+") > strings.Count(fileSection, "-") {
		return "機能拡張"
	}
	return "修正・改善"
}

// extractKeyChanges は主要な変更を抽出
func (ism *interactiveSessionManager) extractKeyChanges(fileSection, fileName string) []string {
	changes := []string{}

	// 新しい関数
	if funcMatches := regexp.MustCompile(`\+func\s+(\w+)`).FindAllStringSubmatch(fileSection, -1); len(funcMatches) > 0 {
		if len(funcMatches) <= 3 {
			for _, match := range funcMatches {
				changes = append(changes, fmt.Sprintf("新関数: %s()", match[1]))
			}
		} else {
			changes = append(changes, fmt.Sprintf("%d個の新しい関数を追加", len(funcMatches)))
		}
	}

	// 新しい構造体
	if structMatches := regexp.MustCompile(`\+type\s+(\w+)\s+struct`).FindAllStringSubmatch(fileSection, -1); len(structMatches) > 0 {
		for _, match := range structMatches {
			changes = append(changes, fmt.Sprintf("新構造体: %s", match[1]))
		}
	}

	// インポート変更
	if strings.Contains(fileSection, "+\t\"") {
		importCount := strings.Count(fileSection, "+\t\"")
		changes = append(changes, fmt.Sprintf("%d個のパッケージを新規導入", importCount))
	}

	// エラーハンドリング改善
	if strings.Contains(fileSection, "fmt.Errorf") || strings.Contains(fileSection, "errors.New") {
		changes = append(changes, "エラーハンドリング強化")
	}

	return changes
}

// formatRiskLevel はリスクレベルをフォーマット
func (ism *interactiveSessionManager) formatRiskLevel(riskLevel string) string {
	switch riskLevel {
	case "🔴 HIGH":
		return "🔴 HIGH (要慎重レビュー)"
	case "🟡 MEDIUM":
		return "🟡 MEDIUM (標準レビュー)"
	default:
		return "🟢 LOW (軽微な変更)"
	}
}

// identifyImpactAreas は影響領域を特定
func (ism *interactiveSessionManager) identifyImpactAreas(changedFiles []string, diffOutput string) []ImpactArea {
	areas := []ImpactArea{}
	areaMap := make(map[string]bool)

	for _, file := range changedFiles {
		if strings.Contains(file, "internal/handlers/chat.go") && !areaMap["chat"] {
			areas = append(areas, ImpactArea{Icon: "💬", Description: "チャット・会話システム"})
			areaMap["chat"] = true
		}
		if strings.Contains(file, "internal/interactive/") && !areaMap["interactive"] {
			areas = append(areas, ImpactArea{Icon: "🎯", Description: "インタラクティブ機能"})
			areaMap["interactive"] = true
		}
		if strings.Contains(file, "internal/config/") && !areaMap["config"] {
			areas = append(areas, ImpactArea{Icon: "⚙️", Description: "設定・構成管理"})
			areaMap["config"] = true
		}
		if strings.Contains(file, "cmd/") && !areaMap["cli"] {
			areas = append(areas, ImpactArea{Icon: "🖥️", Description: "CLI インターフェース"})
			areaMap["cli"] = true
		}
		if strings.Contains(file, "internal/tools/") && !areaMap["tools"] {
			areas = append(areas, ImpactArea{Icon: "🔧", Description: "ツール・ユーティリティ"})
			areaMap["tools"] = true
		}
		if strings.Contains(file, "internal/handlers/") && !areaMap["handlers"] {
			areas = append(areas, ImpactArea{Icon: "🎛️", Description: "ハンドラー・処理制御"})
			areaMap["handlers"] = true
		}
	}

	return areas
}

// extractTechnicalChanges は技術的変更を抽出
func (ism *interactiveSessionManager) extractTechnicalChanges(diffOutput string) []string {
	changes := []string{}

	// 同期・並行処理の追加
	if strings.Contains(diffOutput, "+sync.") || strings.Contains(diffOutput, "+go func") {
		changes = append(changes, "並行処理・同期機能の追加")
	}

	// エラーハンドリング強化
	if strings.Contains(diffOutput, "+\t\treturn fmt.Errorf") {
		errorCount := strings.Count(diffOutput, "+\t\treturn fmt.Errorf")
		changes = append(changes, fmt.Sprintf("エラーハンドリング改善 (%d箇所)", errorCount))
	}

	// 新しいインターフェース追加
	if strings.Contains(diffOutput, "+type ") && strings.Contains(diffOutput, "interface") {
		changes = append(changes, "新インターフェース定義の追加")
	}

	// コンテキスト処理
	if strings.Contains(diffOutput, "context.Context") || strings.Contains(diffOutput, "ctx context.Context") {
		changes = append(changes, "コンテキスト管理の統合")
	}

	// メモリ管理改善
	if strings.Contains(diffOutput, "sync.Pool") || strings.Contains(diffOutput, "make([]") {
		changes = append(changes, "メモリ使用効率の最適化")
	}

	// ログ機能追加
	if strings.Contains(diffOutput, "log.") || strings.Contains(diffOutput, "logger.") {
		changes = append(changes, "ログ機能の強化")
	}

	return changes
}

// identifySecurityConcerns はセキュリティ懸念を特定
func (ism *interactiveSessionManager) identifySecurityConcerns(diffOutput string) []string {
	concerns := []string{}

	// 認証関連
	if strings.Contains(diffOutput, "auth") || strings.Contains(diffOutput, "token") {
		concerns = append(concerns, "認証・認可システムの変更")
	}

	// パスワード・秘密情報
	if strings.Contains(diffOutput, "password") || strings.Contains(diffOutput, "secret") || strings.Contains(diffOutput, "key") {
		concerns = append(concerns, "機密情報の取り扱い変更")
	}

	// ファイルアクセス権限
	if strings.Contains(diffOutput, "os.OpenFile") || strings.Contains(diffOutput, "0644") || strings.Contains(diffOutput, "0755") {
		concerns = append(concerns, "ファイル権限・アクセス制御の変更")
	}

	// 外部コマンド実行
	if strings.Contains(diffOutput, "exec.Command") || strings.Contains(diffOutput, "exec.CommandContext") {
		concerns = append(concerns, "外部コマンド実行によるセキュリティ影響")
	}

	// 入力検証
	if strings.Contains(diffOutput, "strings.Contains") && strings.Contains(diffOutput, "user") {
		concerns = append(concerns, "ユーザー入力処理の変更 - 検証強化を確認")
	}

	return concerns
}

// identifyQualityIssues は品質問題を特定
func (ism *interactiveSessionManager) identifyQualityIssues(diffOutput string, analysis *DetailedDiffAnalysis) []string {
	issues := []string{}

	// 大規模な関数追加
	funcCount := strings.Count(diffOutput, "+func ")
	if funcCount > 10 {
		issues = append(issues, fmt.Sprintf("大量の関数追加 (%d個) - 複雑度増加に注意", funcCount))
	}

	// テストの不足
	hasTest := false
	for _, file := range analysis.ChangedFiles {
		if strings.Contains(file, "_test.go") {
			hasTest = true
			break
		}
	}
	if analysis.AddedLines > 200 && !hasTest {
		issues = append(issues, "大きな変更に対するテストコードの追加が推奨")
	}

	// エラーハンドリングの不足
	addedFuncs := strings.Count(diffOutput, "+func ")
	errorHandling := strings.Count(diffOutput, "return") + strings.Count(diffOutput, "err")
	if addedFuncs > 3 && errorHandling < addedFuncs {
		issues = append(issues, "エラーハンドリングの不足が疑われます")
	}

	// コメント不足
	commentCount := strings.Count(diffOutput, "+//")
	if analysis.AddedLines > 300 && commentCount < 10 {
		issues = append(issues, "コードコメント・ドキュメントの追加を検討")
	}

	return issues
}

// evaluatePerformanceImpact はパフォーマンス影響を評価
func (ism *interactiveSessionManager) evaluatePerformanceImpact(diffOutput string, changedFiles []string) string {
	impacts := []string{}

	// 並行処理の追加
	if strings.Contains(diffOutput, "+go func") || strings.Contains(diffOutput, "+sync.") {
		impacts = append(impacts, "並行処理による高速化期待")
	}

	// データベース・I/O操作
	if strings.Contains(diffOutput, "os.ReadFile") || strings.Contains(diffOutput, "os.WriteFile") {
		impacts = append(impacts, "ファイルI/O処理の追加")
	}

	// ネットワーク処理
	if strings.Contains(diffOutput, "http.") || strings.Contains(diffOutput, "net/") {
		impacts = append(impacts, "ネットワーク通信処理の追加")
	}

	// メモリ使用量の変化
	if strings.Contains(diffOutput, "make([]") || strings.Contains(diffOutput, "make(map") {
		impacts = append(impacts, "メモリ使用量への影響")
	}

	// 大量のループ処理
	if strings.Count(diffOutput, "+\tfor ") > 5 {
		impacts = append(impacts, "複数ループ処理による計算負荷増加")
	}

	if len(impacts) == 0 {
		return "軽微 - 大きなパフォーマンス影響なし"
	}

	return strings.Join(impacts, "、")
}

// getFileTypeIcon はファイルタイプに応じたアイコンを返す
func (ism *interactiveSessionManager) getFileTypeIcon(filename string) string {
	ext := filepath.Ext(filename)
	basename := filepath.Base(filename)

	switch {
	case ext == ".go":
		return "🐹"
	case ext == ".js" || ext == ".ts" || ext == ".jsx" || ext == ".tsx":
		return "📜"
	case ext == ".py":
		return "🐍"
	case ext == ".md":
		return "📚"
	case ext == ".json" || ext == ".yaml" || ext == ".yml":
		return "⚙️"
	case strings.Contains(basename, "test"):
		return "🧪"
	case ext == ".dockerfile" || basename == "Dockerfile":
		return "🐳"
	case ext == ".sh" || ext == ".bash":
		return "⚡"
	case strings.Contains(filename, "config"):
		return "🔧"
	default:
		return "📄"
	}
}

// extractChangePatterns は変更パターンを抽出
func (ism *interactiveSessionManager) extractChangePatterns(diffOutput string) []string {
	patterns := []string{}

	// 新機能追加
	if strings.Contains(diffOutput, "+func New") {
		patterns = append(patterns, "🚀 新しいコンストラクタ関数の追加")
	}
	if strings.Contains(diffOutput, "+type ") && strings.Contains(diffOutput, "struct") {
		patterns = append(patterns, "🏗️ 新しい構造体定義の追加")
	}
	if strings.Contains(diffOutput, "+func ") {
		funcCount := strings.Count(diffOutput, "+func ")
		if funcCount > 1 {
			patterns = append(patterns, fmt.Sprintf("⚡ %d個の新しい関数の追加", funcCount))
		} else {
			patterns = append(patterns, "⚡ 新しい関数の追加")
		}
	}

	// インポート変更
	if strings.Contains(diffOutput, "+import") {
		patterns = append(patterns, "📦 新しいパッケージの導入")
	}

	// 設定変更
	if strings.Contains(diffOutput, "config") || strings.Contains(diffOutput, "Config") {
		patterns = append(patterns, "⚙️ 設定システムの変更")
	}

	// テスト追加
	if strings.Contains(diffOutput, "_test.go") {
		patterns = append(patterns, "🧪 テストコードの追加・変更")
	}

	// ドキュメント更新
	if strings.Contains(diffOutput, "README.md") || strings.Contains(diffOutput, "CLAUDE.md") {
		patterns = append(patterns, "📚 プロジェクトドキュメントの更新")
	}

	// エラーハンドリング改善
	if strings.Contains(diffOutput, "fmt.Errorf") || strings.Contains(diffOutput, "errors.New") {
		patterns = append(patterns, "🔧 エラーハンドリングの改善")
	}

	// 依存関係
	if strings.Contains(diffOutput, "go.mod") {
		patterns = append(patterns, "🔗 Go モジュール依存関係の更新")
	}

	// パフォーマンス改善
	if strings.Contains(diffOutput, "goroutine") || strings.Contains(diffOutput, "sync.") {
		patterns = append(patterns, "⚡ 並行処理・パフォーマンス改善")
	}

	return patterns
}

// extractChangedFilesFromDiff はdiff出力から変更されたファイル一覧を抽出
func (ism *interactiveSessionManager) extractChangedFilesFromDiff(diffOutput string) []string {
	var files []string
	lines := strings.Split(diffOutput, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			// "diff --git a/file.go b/file.go" の形式
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				filename := strings.TrimPrefix(parts[2], "a/")
				files = append(files, filename)
			}
		}
	}

	return files
}

// analyzeGoCodeChanges はGoコードの変更を詳細分析
func (ism *interactiveSessionManager) analyzeGoCodeChanges(filename, diffOutput string) []string {
	var suggestions []string

	// 構造体の変更
	if strings.Contains(diffOutput, "type ") && strings.Contains(diffOutput, "struct") {
		suggestions = append(suggestions, fmt.Sprintf("Go構造体変更(%s): APIの後方互換性を確認", filename))
	}

	// インターフェース の変更
	if strings.Contains(diffOutput, "interface") {
		suggestions = append(suggestions, fmt.Sprintf("Go インターフェース変更(%s): 実装クラスへの影響を確認", filename))
	}

	// メソッドの変更
	if strings.Contains(diffOutput, "func (") {
		suggestions = append(suggestions, fmt.Sprintf("Goメソッド変更(%s): 関連する単体テストの更新", filename))
	}

	// パッケージ名の変更
	if strings.Contains(diffOutput, "package ") {
		suggestions = append(suggestions, fmt.Sprintf("Goパッケージ変更(%s): インポート文の全体的な更新が必要", filename))
	}

	// コンストラクタ関数
	if strings.Contains(diffOutput, "+func New") {
		suggestions = append(suggestions, fmt.Sprintf("新しいコンストラクタ(%s): 初期化ロジックの検証", filename))
	}

	// エラー定義
	if strings.Contains(diffOutput, "errors.New") || strings.Contains(diffOutput, "fmt.Errorf") {
		suggestions = append(suggestions, fmt.Sprintf("エラーメッセージ変更(%s): エラーハンドリングテストの確認", filename))
	}

	// 実行後の提案
	suggestions = append(suggestions, fmt.Sprintf("`go fmt %s` でフォーマット、`go vet %s` で静的解析", filename, filename))

	return suggestions
}

// analyzeFileOperations はファイル操作の結果を分析
func (ism *interactiveSessionManager) analyzeFileOperations(action, result string) []string {
	var suggestions []string

	if strings.Contains(action, "ファイル作成") && strings.Contains(result, "成功") {
		suggestions = append(suggestions, "作成されたファイルの内容を確認し、必要に応じて編集")
		suggestions = append(suggestions, "関連するテストファイルの作成を検討")
	}

	if strings.Contains(action, "ファイル読み込み") {
		if strings.Contains(result, "go") {
			suggestions = append(suggestions, "Goコードの構文チェック: `go fmt` と `go vet` を実行")
		}
		suggestions = append(suggestions, "ファイル内容に基づいて必要な修正や改善を実施")
	}

	return suggestions
}

// analyzeErrors はエラー内容を分析して解決策を提案
func (ism *interactiveSessionManager) analyzeErrors(errorOutput string) []string {
	var suggestions []string

	if strings.Contains(errorOutput, "permission denied") {
		suggestions = append(suggestions, "権限エラーです。`chmod +x` または管理者権限で再実行")
	}

	if strings.Contains(errorOutput, "command not found") {
		suggestions = append(suggestions, "コマンドが見つかりません。インストール状況を確認")
	}

	if strings.Contains(errorOutput, "go: cannot find module") {
		suggestions = append(suggestions, "`go mod tidy` でモジュール依存関係を解決")
	}

	if strings.Contains(errorOutput, "syntax error") {
		suggestions = append(suggestions, "構文エラーがあります。該当ファイルを確認して修正")
	}

	return suggestions
}

// analyzeBuildResults はビルド結果を分析
func (ism *interactiveSessionManager) analyzeBuildResults(buildOutput string) []string {
	var suggestions []string

	if strings.Contains(buildOutput, "Build succeeded") || len(strings.TrimSpace(buildOutput)) == 0 {
		suggestions = append(suggestions, "ビルド成功！テストの実行を検討: `go test ./...`")
		suggestions = append(suggestions, "実行ファイルの動作確認を実施")
	}

	if strings.Contains(buildOutput, "error:") || strings.Contains(buildOutput, "failed") {
		suggestions = append(suggestions, "ビルドエラーを修正後、再度ビルドを実行")
		suggestions = append(suggestions, "依存関係の確認: `go mod download`")
	}

	return suggestions
}

// removeDuplicateSuggestions は重複する提案を除去
func (ism *interactiveSessionManager) removeDuplicateSuggestions(suggestions []string) []string {
	seen := make(map[string]bool)
	var unique []string

	for _, suggestion := range suggestions {
		if !seen[suggestion] {
			seen[suggestion] = true
			unique = append(unique, suggestion)
		}
	}

	return unique
}

// determineResponseType はLLM応答から応答タイプを判定（Claude Code式簡素化）
func (ism *interactiveSessionManager) determineResponseType(llmResponse string, intent string) ResponseType {
	// コードブロック（```）を含む場合はコード提案として扱う
	if strings.Contains(llmResponse, "```") {
		return ResponseTypeCodeSuggestion
	}

	// 作成要求の場合でコードが含まれる場合はコード提案
	if intent == "creation_request" && (strings.Contains(llmResponse, "package") ||
		strings.Contains(llmResponse, "func") || strings.Contains(llmResponse, "def") ||
		strings.Contains(llmResponse, "class")) {
		return ResponseTypeCodeSuggestion
	}

	// 確認プロンプトが含まれる場合
	if strings.Contains(llmResponse, "適用しますか") || strings.Contains(llmResponse, "実行しますか") {
		return ResponseTypeConfirmation
	}

	// デフォルトはメッセージ（モデルの自然な判断を信頼）
	return ResponseTypeMessage
}

// requiresConfirmation は確認が必要かどうかを判定
func (ism *interactiveSessionManager) requiresConfirmation(responseType ResponseType, intent string) bool {
	// コード提案は基本的に確認が必要
	if responseType == ResponseTypeCodeSuggestion {
		return true
	}

	// ファイル修正系の意図は確認が必要
	if strings.Contains(intent, "修正") || strings.Contains(intent, "最適化") || strings.Contains(intent, "リファクタリング") {
		return true
	}

	return false
}

// extractCodeSuggestionsFromLLM はLLM応答からコード提案を抽出
func (ism *interactiveSessionManager) extractCodeSuggestionsFromLLM(llmResponse string, originalInput string) ([]*CodeSuggestion, error) {
	var suggestions []*CodeSuggestion

	// 元の入力からファイルパスを抽出
	suggestedFilePath := ism.extractFilePathFromInput(originalInput)

	// マークダウンコードブロックを抽出
	codeBlockPattern := regexp.MustCompile("```(?:go|javascript|python|rust|java|c|cpp|csharp)?\\s*\\n([\\s\\S]*?)\\n```")
	matches := codeBlockPattern.FindAllStringSubmatch(llmResponse, -1)

	for i, match := range matches {
		if len(match) > 1 {
			suggestion := &CodeSuggestion{
				ID:            fmt.Sprintf("llm_suggestion_%d_%d", time.Now().UnixNano(), i),
				Type:          SuggestionTypeImprovement,
				OriginalCode:  "", // 元コードは別途特定が必要
				SuggestedCode: strings.TrimSpace(match[1]),
				Explanation:   ism.extractExplanationFromLLM(llmResponse, i),
				Confidence:    0.85, // LLM生成なので高い信頼度
				ImpactLevel:   ImpactLevelMedium,
				FilePath:      suggestedFilePath, // 抽出されたファイルパスを設定
				LineRange:     [2]int{0, 0},
				Metadata: map[string]string{
					"generated_by":   "llm",
					"model":          "qwen2.5-coder:14b",
					"benefits":       "AI生成による実装, ベストプラクティスに基づく",
					"risks":          "実際の動作確認が必要",
					"estimated_time": "5-10分",
					"original_input": originalInput,
				},
				CreatedAt: time.Now(),
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	// コードブロックがない場合は一般的な提案として扱う
	if len(suggestions) == 0 {
		suggestion := &CodeSuggestion{
			ID:            fmt.Sprintf("llm_suggestion_%d", time.Now().UnixNano()),
			Type:          SuggestionTypeImprovement,
			OriginalCode:  "",
			SuggestedCode: llmResponse, // 全体を提案として扱う
			Explanation:   "AIによる提案",
			Confidence:    0.7,
			ImpactLevel:   ImpactLevelLow,
			FilePath:      suggestedFilePath, // 抽出されたファイルパスを設定
			LineRange:     [2]int{0, 0},
			Metadata: map[string]string{
				"generated_by":   "llm",
				"model":          "qwen2.5-coder:14b",
				"estimated_time": "確認が必要",
				"original_input": originalInput,
			},
			CreatedAt: time.Now(),
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// extractExplanationFromLLM はLLM応答から説明部分を抽出
func (ism *interactiveSessionManager) extractExplanationFromLLM(llmResponse string, codeIndex int) string {
	// コードブロック前後のテキストから説明を抽出（簡易実装）
	lines := strings.Split(llmResponse, "\n")
	var explanation strings.Builder

	inCodeBlock := false
	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if !inCodeBlock && strings.TrimSpace(line) != "" {
			explanation.WriteString(line)
			explanation.WriteString(" ")
		}
	}

	result := strings.TrimSpace(explanation.String())
	if result == "" {
		return "AIによるコード提案です。"
	}

	// 長すぎる場合は短縮
	if len(result) > 200 {
		return result[:200] + "..."
	}

	return result
}

// ヘルパーメソッド

// sessionTypeToString はセッションタイプを文字列に変換
func (ism *interactiveSessionManager) sessionTypeToString(sessionType CodingSessionType) string {
	switch sessionType {
	case CodingSessionTypeDebugging:
		return "デバッグ作業"
	case CodingSessionTypeRefactor:
		return "リファクタリング"
	case CodingSessionTypeReview:
		return "コードレビュー"
	case CodingSessionTypeLearning:
		return "学習・説明"
	default:
		return "一般的なコーディング"
	}
}

// suggestionTypeToString は提案タイプを文字列に変換
func (ism *interactiveSessionManager) suggestionTypeToString(suggestionType SuggestionType) string {
	switch suggestionType {
	case SuggestionTypeBugFix:
		return "バグ修正"
	case SuggestionTypeOptimization:
		return "パフォーマンス最適化"
	case SuggestionTypeRefactoring:
		return "リファクタリング"
	case SuggestionTypeDocumentation:
		return "ドキュメント追加"
	case SuggestionTypeSecurity:
		return "セキュリティ修正"
	case SuggestionTypeTestGeneration:
		return "テスト生成"
	default:
		return "改善提案"
	}
}

// formatContextForPrompt はコンテキストをプロンプト用に整形
func (ism *interactiveSessionManager) formatContextForPrompt(context []*contextmanager.ContextItem) string {
	if len(context) == 0 {
		return "関連コンテキストなし"
	}

	var formatted strings.Builder
	for i, item := range context {
		formatted.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Content))
	}

	return formatted.String()
}

// updateAverageResponseTime は平均応答時間を更新
func (ism *interactiveSessionManager) updateAverageResponseTime(
	currentAverage time.Duration,
	newTime time.Duration,
	totalCount int,
) time.Duration {
	if totalCount <= 1 {
		return newTime
	}

	total := currentAverage*time.Duration(totalCount-1) + newTime
	return total / time.Duration(totalCount)
}

// abs は絶対値を返す
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// calculateSuggestionConfidence は提案の信頼度を計算
func (ism *interactiveSessionManager) calculateSuggestionConfidence(
	session *InteractiveSession,
	request *SuggestionRequest,
	context []*contextmanager.ContextItem,
) float64 {
	confidence := 0.5 // ベース信頼度

	// コンテキストの豊富さによる調整
	if len(context) > 5 {
		confidence += 0.2
	}

	// 提案タイプによる調整
	switch request.Type {
	case SuggestionTypeBugFix:
		confidence += 0.1 // バグ修正は比較的信頼度高
	case SuggestionTypeOptimization:
		confidence -= 0.1 // パフォーマンス最適化は慎重に
	case SuggestionTypeSecurity:
		confidence += 0.15 // セキュリティは重要
	}

	// セッション履歴による調整
	if session.Metrics.SuggestionsAccepted > session.Metrics.SuggestionsRejected {
		confidence += 0.1
	}

	// 0.0-1.0の範囲に正規化
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// evaluateImpactLevel は影響レベルを評価
func (ism *interactiveSessionManager) evaluateImpactLevel(request *SuggestionRequest) ImpactLevel {
	switch request.Type {
	case SuggestionTypeSecurity:
		return ImpactLevelCritical
	case SuggestionTypeBugFix:
		return ImpactLevelHigh
	case SuggestionTypeOptimization, SuggestionTypeRefactoring:
		return ImpactLevelMedium
	case SuggestionTypeDocumentation:
		return ImpactLevelLow
	default:
		return ImpactLevelMedium
	}
}

// updateUserSatisfactionScore はユーザー満足度スコアを更新
func (ism *interactiveSessionManager) updateUserSatisfactionScore(session *InteractiveSession, accepted bool) {
	currentScore := session.Metrics.UserSatisfactionScore
	weight := 0.1 // 学習レート

	if accepted {
		session.Metrics.UserSatisfactionScore = currentScore + weight*(1.0-currentScore)
	} else {
		session.Metrics.UserSatisfactionScore = currentScore + weight*(0.0-currentScore)
	}
}

// analyzeUserIntent はClaude Code式簡素化された意図解析（安全性チェック中心）
func (ism *interactiveSessionManager) analyzeUserIntent(
	ctx context.Context,
	session *InteractiveSession,
	input string,
) (string, error) {
	// Claude Code式: モデル自体の判断を優先、最低限の分類のみ
	lowerInput := strings.ToLower(input)

	// 危険なコマンドの基本チェック（安全性のため）
	if strings.Contains(lowerInput, "rm -rf") || strings.Contains(lowerInput, "sudo") ||
		strings.Contains(lowerInput, "delete") && strings.Contains(lowerInput, "all") {
		return "potentially_dangerous", nil
	}

	// 基本的な要求タイプの大まかな分類（モデル判断の補助程度）
	if strings.Contains(lowerInput, "作って") || strings.Contains(lowerInput, "作成") ||
		strings.Contains(lowerInput, "実装") || strings.Contains(lowerInput, "create") {
		return "creation_request", nil
	}

	// その他はすべて一般的な要求として扱う（モデルが詳細判断）
	return "general_request", nil
}

// advanceConversationFlow は会話フローを進行
func (ism *interactiveSessionManager) advanceConversationFlow(
	session *InteractiveSession,
	intent string,
) error {
	flow, exists := ism.conversationFlows[session.ID]
	if !exists {
		return fmt.Errorf("会話フローが見つかりません")
	}

	// 現在のステップを完了
	now := time.Now()
	flow.CurrentStep.EndTime = &now
	flow.CurrentStep.Success = true
	flow.StepHistory = append(flow.StepHistory, flow.CurrentStep)
	flow.CompletedSteps++

	// 次のステップを決定
	nextStepType := ism.determineNextFlowStep(flow.CurrentStep.StepType, intent)
	flow.CurrentStep = FlowStep{
		StepID:      fmt.Sprintf("step_%d", time.Now().UnixNano()),
		StepType:    nextStepType,
		Description: ism.getStepDescription(nextStepType),
		StartTime:   now,
	}

	// 進捗更新
	flow.Progress = float64(flow.CompletedSteps) / float64(flow.EstimatedSteps)
	if flow.Progress > 1.0 {
		flow.Progress = 1.0
	}

	return nil
}

// determineNextFlowStep は次のフローステップを決定
func (ism *interactiveSessionManager) determineNextFlowStep(
	currentStep FlowStepType,
	intent string,
) FlowStepType {
	switch currentStep {
	case FlowStepTypeUnderstanding:
		return FlowStepTypeAnalysis
	case FlowStepTypeAnalysis:
		return FlowStepTypePlanning
	case FlowStepTypePlanning:
		return FlowStepTypeImplementation
	case FlowStepTypeImplementation:
		return FlowStepTypeTesting
	case FlowStepTypeTesting:
		return FlowStepTypeVerification
	default:
		return FlowStepTypeCompletion
	}
}

// getStepDescription はステップの説明を取得
func (ism *interactiveSessionManager) getStepDescription(stepType FlowStepType) string {
	switch stepType {
	case FlowStepTypeUnderstanding:
		return "要求の理解"
	case FlowStepTypeAnalysis:
		return "コード分析"
	case FlowStepTypePlanning:
		return "実装計画"
	case FlowStepTypeImplementation:
		return "実装"
	case FlowStepTypeTesting:
		return "テスト"
	case FlowStepTypeVerification:
		return "検証"
	case FlowStepTypeCompletion:
		return "完了"
	default:
		return "不明"
	}
}

// generateInteractiveResponse はインタラクティブな応答を生成
func (ism *interactiveSessionManager) generateInteractiveResponse(
	ctx context.Context,
	session *InteractiveSession,
	input string,
	intent string,
) (*InteractionResponse, error) {
	// 応答時間計測開始
	startTime := time.Now()

	// LLM統合による実際の応答生成
	prompt := ism.buildInteractivePrompt(session, input, intent)

	// トークン数を推定
	estimatedTokens := len(prompt) / 4

	// ClaudeCode風進捗表示を開始
	progressIndicator := ui.NewProgressIndicator("Generating response…", estimatedTokens)
	progressIndicator.Start()

	defer func() {
		progressIndicator.Stop()
	}()

	// LLM呼び出し
	chatReq := llm.ChatRequest{
		Model: ism.getConfiguredModel(), // 設定からモデルを取得
		Messages: []llm.ChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	// 中断可能なコンテキストを作成
	llmCtx := progressIndicator.GetContext()
	if llmCtx.Err() != nil {
		progressIndicator.CompleteWithResult(false, "Request interrupted")
		return nil, fmt.Errorf("request interrupted by user")
	}

	llmResponse, err := ism.llmProvider.Chat(llmCtx, chatReq)
	if err != nil {
		// LLM失敗時の進捗表示完了
		progressIndicator.CompleteWithResult(false, "LLM request failed")
		return ism.generateFallbackResponse(session, input, intent, err)
	}

	// 応答受信トークン数を更新
	receivedTokens := len(llmResponse.Message.Content) / 4
	progressIndicator.UpdateTokens(receivedTokens)

	// LLM応答の言語統一処理（繁体字等を日本語に修正）
	cleanedResponse := ism.normalizeLanguage(llmResponse.Message.Content)
	llmResponse.Message.Content = cleanedResponse

	// SmartContextManagerにLLM応答を追加
	ism.addToSmartContext(session.ID, cleanedResponse, "llm_response")

	// 構造化された応答を解析して実際のツール実行を行う
	finalResponse, err := ism.parseAndExecuteStructuredResponse(ctx, session, llmResponse.Message.Content, input)
	if err != nil {
		return ism.generateFallbackResponse(session, input, intent, err)
	}

	if finalResponse != nil {
		// 構造化応答にもメタ情報を追加
		ism.addMetaInfoToResponse(finalResponse, startTime, chatReq.Model, len(prompt))

		// 構造化応答完了の進捗メッセージ
		progressIndicator.CompleteWithResult(true, "Structured response executed successfully")

		return finalResponse, nil
	}

	// 構造化応答がない場合：分析系の質問は強制的にANALYSISを実行
	if ism.shouldForceAnalysis(input, intent) {
		analysisResult := ism.performAnalysis(session, input)

		response := &InteractionResponse{
			SessionID:            session.ID,
			ResponseType:         ResponseTypeAnalysis,
			Message:              fmt.Sprintf("🔍 **プロジェクト分析結果**\n\n%s\n\n**AI応答:**\n%s", analysisResult, llmResponse.Message.Content),
			RequiresConfirmation: false,
			Metadata: map[string]string{
				"forced_analysis": "true",
				"analysis_result": "included",
				"ai_response":     "enhanced",
			},
			GeneratedAt: time.Now(),
		}

		// メタ情報を追加
		ism.addMetaInfoToResponse(response, startTime, chatReq.Model, len(prompt))

		// 分析完了の進捗メッセージ
		progressIndicator.CompleteWithResult(true, "Analysis completed successfully")

		return response, nil
	}

	// 通常のLLM応答を返す
	responseType := ism.determineResponseType(llmResponse.Message.Content, intent)

	response := &InteractionResponse{
		SessionID:            session.ID,
		ResponseType:         responseType,
		Message:              llmResponse.Message.Content,
		RequiresConfirmation: ism.requiresConfirmation(responseType, intent),
		Metadata:             make(map[string]string),
		GeneratedAt:          time.Now(),
	}

	// コード提案の場合、提案を解析
	if responseType == ResponseTypeCodeSuggestion {
		suggestions, err := ism.extractCodeSuggestionsFromLLM(llmResponse.Message.Content, input)
		if err == nil && len(suggestions) > 0 {
			response.Suggestions = suggestions

			// Claude Code式: コマンド実行の場合は即座に実行
			if ism.isCommandSuggestion(suggestions[0].SuggestedCode) {
				extractedCmd := ism.extractCommandFromSuggestion(suggestions[0].SuggestedCode)
				if ism.isSafeCommand(extractedCmd) {
					// 安全なコマンドは即座に実行
					err = ism.executeCommandDirectly(ctx, session, suggestions[0])
					if err != nil {
						return nil, fmt.Errorf("コマンド実行エラー: %w", err)
					}
					// 実行結果を応答に含める
					response.ResponseType = ResponseTypeMessage
					response.Message = fmt.Sprintf("コマンド実行結果:\n%s", session.LastCommandOutput)
					response.RequiresConfirmation = false
					session.State = SessionStateIdle
				} else {
					// 危険なコマンドは確認を求める
					session.PendingSuggestion = suggestions[0]
					session.State = SessionStateWaitingForConfirmation
				}
			} else {
				// ファイル操作の場合は危険性を判定
				if ism.isDangerousFileOperation(suggestions[0]) {
					// 危険なファイル操作は確認を求める
					session.PendingSuggestion = suggestions[0]
					session.State = SessionStateWaitingForConfirmation
				} else {
					// 安全なファイル操作は確認を求める（基本動作）
					session.PendingSuggestion = suggestions[0]
					session.State = SessionStateWaitingForConfirmation
				}
			}
		}
	}

	response.Metadata["intent"] = intent
	response.Metadata["session_type"] = ism.sessionTypeToString(session.Type)
	response.Metadata["llm_model"] = chatReq.Model

	// メタ情報を追加
	ism.addMetaInfoToResponse(response, startTime, chatReq.Model, len(prompt))

	// 成功時の進捗完了メッセージ
	progressIndicator.CompleteWithResult(true, "Response generated successfully")

	return response, nil
}

// isSafeCommand は安全なコマンドかどうかを判定
func (ism *interactiveSessionManager) isSafeCommand(command string) bool {
	command = strings.TrimSpace(command)

	// 安全なコマンドのリスト（読み取り専用操作）
	safeCommands := []string{
		"git status",
		"git log",
		"git branch",
		"git diff",
		"git show",
		"ls",
		"pwd",
		"cat",
		"head",
		"tail",
		"grep",
		"find",
		"which",
		"echo",
		"date",
		"whoami",
		"id",
	}

	// コマンドの先頭部分を抽出
	commandParts := strings.Fields(command)
	if len(commandParts) == 0 {
		return false
	}

	baseCommand := commandParts[0]

	// 安全なコマンドかチェック
	for _, safe := range safeCommands {
		safeParts := strings.Fields(safe)
		if len(safeParts) > 0 && safeParts[0] == baseCommand {
			// git statusのような複合コマンドの場合、完全一致をチェック
			if len(safeParts) > 1 {
				if len(commandParts) >= len(safeParts) {
					match := true
					for i, part := range safeParts {
						if commandParts[i] != part {
							match = false
							break
						}
					}
					if match {
						return true
					}
				}
			} else {
				return true
			}
		}
	}

	return false
}

// executeCommandDirectly は安全なコマンドを直接実行
func (ism *interactiveSessionManager) executeCommandDirectly(ctx context.Context, session *InteractiveSession, suggestion *CodeSuggestion) error {
	command := ism.extractCommandFromSuggestion(suggestion.SuggestedCode)
	if command == "" {
		return fmt.Errorf("コマンドを抽出できませんでした")
	}

	// BashToolを使ってコマンド実行
	result, err := ism.bashTool.Execute(command, "安全なコマンドの直接実行", 30000)
	if err != nil {
		return fmt.Errorf("コマンド実行失敗: %w", err)
	}

	// セッションに実行結果を保存
	session.LastCommandOutput = result.Content
	session.LastActivity = time.Now()

	// 提案を適用済みにマーク
	suggestion.Applied = true
	suggestion.UserConfirmed = true

	return nil
}

// parseSuggestionResponse は提案応答を解析
func (ism *interactiveSessionManager) parseSuggestionResponse(
	response string,
	request *SuggestionRequest,
) (*CodeSuggestion, error) {
	// 簡易的な解析実装
	suggestion := &CodeSuggestion{
		ID:            fmt.Sprintf("suggestion_%d", time.Now().UnixNano()),
		Type:          request.Type,
		OriginalCode:  request.Code,
		SuggestedCode: "// 改善されたコード\n" + request.Code,
		Explanation:   "コードの改善を提案しました。",
		Confidence:    0.8,
		FilePath:      request.FilePath,
		LineRange:     request.LineRange,
		Metadata:      make(map[string]string),
		CreatedAt:     time.Now(),
		UserConfirmed: false,
		Applied:       false,
	}

	// 実際は応答テキストからコードブロックや説明を抽出
	// 高度な解析は将来のバージョンで実装予定

	return suggestion, nil
}

// extractFilePathFromInput はユーザー入力からファイルパスを抽出
func (ism *interactiveSessionManager) extractFilePathFromInput(input string) string {
	// ファイル名のパターンを抽出（例: "test.goを作成", "hello.jsファイル"等）
	patterns := []string{
		`(\w+\.(?:go|js|py|java|rs|cpp|c|ts|jsx|tsx|vue|html|css|json|yaml|yml|xml|md|txt))`,
		`(\w+ファイル)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(input)
		if len(matches) > 1 {
			filename := matches[1]
			// "ファイル"という文字が含まれている場合は、適切な拡張子を推定
			if strings.Contains(filename, "ファイル") {
				baseFilename := strings.ReplaceAll(filename, "ファイル", "")
				// コンテキストから言語を推定
				if strings.Contains(input, "Go") || strings.Contains(input, "go") {
					return baseFilename + ".go"
				} else if strings.Contains(input, "JavaScript") || strings.Contains(input, "JS") {
					return baseFilename + ".js"
				} else if strings.Contains(input, "Python") {
					return baseFilename + ".py"
				}
				// デフォルトは.txtファイル
				return baseFilename + ".txt"
			}
			return filename
		}
	}

	// 特定できない場合はデフォルトのファイル名を生成
	if strings.Contains(input, "Go") || strings.Contains(input, "go") {
		return "main.go"
	} else if strings.Contains(input, "JavaScript") || strings.Contains(input, "JS") {
		return "index.js"
	} else if strings.Contains(input, "Python") {
		return "main.py"
	}

	return "output.txt" // 最終フォールバック
}

// isCommandSuggestion はコマンド実行の提案かどうかを判定
func (ism *interactiveSessionManager) isCommandSuggestion(suggestedCode string) bool {
	// 1. BashToolパターン（最も一般的）
	if strings.Contains(suggestedCode, "BashTool") {
		return true
	}

	// 2. bashコードブロック
	if strings.Contains(suggestedCode, "```bash") || strings.Contains(suggestedCode, "```sh") {
		return true
	}

	// 3. $ プレフィックス付きコマンド
	if strings.Contains(suggestedCode, "$ ") {
		return true
	}

	// 4. 一般的なコマンドの直接記述
	lines := strings.Split(suggestedCode, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "git ") ||
			strings.HasPrefix(line, "ls") ||
			strings.HasPrefix(line, "pwd") ||
			strings.HasPrefix(line, "cat ") ||
			strings.HasPrefix(line, "head ") ||
			strings.HasPrefix(line, "tail ") {
			return true
		}
	}

	// 5. FileOperations等のファイル操作は除外（コマンド実行ではない）
	if strings.Contains(suggestedCode, "FileOperations") {
		return false // ファイル操作として扱う
	}

	// 6. git関連のキーワードがある場合
	if strings.Contains(suggestedCode, "git status") ||
		strings.Contains(suggestedCode, "git branch") ||
		strings.Contains(suggestedCode, "git log") {
		return true
	}

	// 7. 単体で短いコマンドの場合
	trimmed := strings.TrimSpace(suggestedCode)
	if trimmed == "git status" || trimmed == "ls" || trimmed == "pwd" {
		return true
	}

	return false
}

// extractCommandFromSuggestion は提案からコマンドを抽出
func (ism *interactiveSessionManager) extractCommandFromSuggestion(suggestedCode string) string {
	// 1. BashToolパターンの抽出（最も一般的）
	if strings.Contains(suggestedCode, "BashTool") {
		// BashTool.run_command("git status") パターン
		re := regexp.MustCompile(`BashTool\.run_command\("([^"]+)"\)`)
		matches := re.FindStringSubmatch(suggestedCode)
		if len(matches) > 1 {
			return matches[1]
		}

		// BashTool("git status") パターン
		re = regexp.MustCompile(`BashTool\("([^"]+)"\)`)
		matches = re.FindStringSubmatch(suggestedCode)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	// 2. コードブロック内のコマンド抽出
	lines := strings.Split(suggestedCode, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// bash/sh コードブロック内のコマンド
		if strings.HasPrefix(line, "git ") ||
			strings.HasPrefix(line, "ls") ||
			strings.HasPrefix(line, "pwd") ||
			strings.HasPrefix(line, "cat ") ||
			strings.HasPrefix(line, "head ") ||
			strings.HasPrefix(line, "tail ") {
			return line
		}

		// $ プレフィックス付きコマンド
		if strings.HasPrefix(line, "$ ") {
			return strings.TrimPrefix(line, "$ ")
		}
	}

	// 3. 直接コマンド形式
	trimmed := strings.TrimSpace(suggestedCode)
	if strings.HasPrefix(trimmed, "git ") ||
		strings.HasPrefix(trimmed, "ls") ||
		strings.HasPrefix(trimmed, "pwd") ||
		strings.HasPrefix(trimmed, "cat ") {
		return trimmed
	}

	// 4. ツール形式の一般的パターン
	if strings.Contains(suggestedCode, "git status") {
		return "git status"
	}
	if strings.Contains(suggestedCode, "git branch") {
		return "git branch"
	}
	if strings.Contains(suggestedCode, `"ls"`) || strings.Contains(suggestedCode, "ls") {
		return "ls -la"
	}

	// 5. 実際のgit status結果が表示されている場合、コマンドを実行すべき
	if strings.Contains(suggestedCode, "On branch") ||
		strings.Contains(suggestedCode, "nothing to commit") ||
		strings.Contains(suggestedCode, "working tree clean") ||
		strings.Contains(suggestedCode, "Changes not staged") ||
		strings.Contains(suggestedCode, "Untracked files") {
		return "git status"
	}

	return ""
}

// GetProactiveExtension はプロアクティブ拡張を取得
// getConfiguredModel は設定からモデル名を取得
func (ism *interactiveSessionManager) getConfiguredModel() string {
	if ism.modelName != "" {
		return ism.modelName
	}
	return "qwen2.5-coder:14b" // デフォルトモデル
}

func (ism *interactiveSessionManager) GetProactiveExtension() *ProactiveExtension {
	return ism.proactiveExt
}

// shouldUseProactiveExtension はプロアクティブ拡張を使用するかどうかを判定
func (ism *interactiveSessionManager) shouldUseProactiveExtension(input string) bool {
	lowerInput := strings.ToLower(input)

	// 分析が必要なキーワードパターン
	analysisKeywords := []string{
		"分析", "analyze", "問題", "problem", "状況", "状態", "エラー", "error",
		"リポジトリ", "repository", "プロジェクト", "project", "コード", "code",
		"品質", "quality", "セキュリティ", "security", "パフォーマンス", "performance",
		"構造", "structure", "依存関係", "dependency", "テスト", "test", "カバレッジ", "coverage",
	}

	for _, keyword := range analysisKeywords {
		if strings.Contains(lowerInput, keyword) {
			return true
		}
	}

	// 質問文の判定
	questionPatterns := []string{
		"どう", "なぜ", "なに", "いつ", "どこ", "どの", "what", "why", "how", "when", "where", "which",
		"？", "?", "教えて", "説明", "確認", "調べ", "check", "explain", "tell me", "show me",
	}

	for _, pattern := range questionPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}

	return false
}

// shouldForceAnalysis は分析を強制実行すべきかどうかを判定
func (ism *interactiveSessionManager) shouldForceAnalysis(input, intent string) bool {
	lowerInput := strings.ToLower(input)
	lowerIntent := strings.ToLower(intent)

	// 明確な分析要求キーワード
	forceAnalysisKeywords := []string{
		"分析", "analyze", "問題点", "問題", "状況", "状態",
		"リポジトリ", "repository", "プロジェクト", "project",
		"現状", "current", "詳細", "detail", "調査", "investigate",
		"確認", "check", "診断", "diagnose", "評価", "evaluate",
	}

	for _, keyword := range forceAnalysisKeywords {
		if strings.Contains(lowerInput, keyword) || strings.Contains(lowerIntent, keyword) {
			return true
		}
	}

	return false
}

// performDetailedCognitiveAnalysis は詳細な認知分析を実行
func (ism *interactiveSessionManager) performDetailedCognitiveAnalysis(ctx context.Context, query string) string {
	if ism.cognitiveAnalyzer == nil {
		return ""
	}

	// 深度の高い分析リクエストを作成
	request := &analysis.AnalysisRequest{
		UserInput:       query,
		Response:        "",
		AnalysisDepth:   "deep",
		RequiredMetrics: []string{"semantic_entropy", "confidence", "reasoning", "creativity"},
		Context: map[string]interface{}{
			"analysis_type": "detailed_cognitive",
			"user_query":    query,
			"timestamp":     time.Now().Format(time.RFC3339),
		},
	}

	result, err := ism.cognitiveAnalyzer.AnalyzeCognitive(ctx, request)
	if err != nil {
		return fmt.Sprintf("⚠️ 認知分析エラー: %v", err)
	}

	// 詳細な結果をフォーマット
	var details []string
	details = append(details, "🧠 **詳細認知分析結果**")
	details = append(details, fmt.Sprintf("  • **信頼度スコア**: %.2f/1.0 %s",
		result.TrustScore, ism.getConfidenceEmoji(result.TrustScore)))
	details = append(details, fmt.Sprintf("  • **全体品質**: %.2f/1.0", result.OverallQuality))

	if result.Confidence != nil {
		details = append(details, fmt.Sprintf("  • **信頼度分析**: %.3f (セマンティックエントロピー: %.3f)",
			result.Confidence.OverallConfidence, result.Confidence.SemanticEntropy))
	}

	if result.ReasoningDepth != nil {
		details = append(details, fmt.Sprintf("  • **論理的推論**: 深度%d (論理構造評価: %.1f)",
			result.ReasoningDepth.OverallDepth, result.ReasoningDepth.LogicalCoherence))
	}

	if result.Creativity != nil {
		details = append(details, fmt.Sprintf("  • **創造性評価**: %.2f (流暢性: %.1f, 独創性: %.1f)",
			result.Creativity.OverallScore, result.Creativity.Fluency, result.Creativity.Originality))
	}

	if len(result.RecommendedActions) > 0 {
		details = append(details, "  • **推奨アクション**:")
		for i, action := range result.RecommendedActions {
			if i < 3 { // 最大3つまで表示
				details = append(details, fmt.Sprintf("    - %s", action))
			}
		}
	}

	return strings.Join(details, "\n")
}

// performProjectStructureAnalysisImpl はプロジェクト構造の詳細分析を実行
func (ism *interactiveSessionManager) performProjectStructureAnalysisImpl(projectPath string) string {
	var details []string
	details = append(details, "🏗️ **プロジェクト構造分析**")

	// ファイル数とディレクトリ構造の分析
	fileCount := 0
	dirCount := 0
	var largeFiles []string
	var languageStats = make(map[string]int)

	filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// 隠しディレクトリ(.git等)をスキップ
		if strings.Contains(path, "/.") {
			return nil
		}

		if info.IsDir() {
			dirCount++
		} else {
			fileCount++

			// 大きなファイルをチェック
			if info.Size() > 1024*1024 { // 1MB以上
				largeFiles = append(largeFiles, fmt.Sprintf("%s (%s)",
					path, ism.formatFileSize(info.Size())))
			}

			// 言語別統計
			ext := filepath.Ext(path)
			if lang := ism.getLanguageFromExtension(ext); lang != "" {
				languageStats[lang]++
			}
		}

		return nil
	})

	details = append(details, fmt.Sprintf("  • **規模**: %d ファイル, %d ディレクトリ", fileCount, dirCount))

	// 言語別統計
	if len(languageStats) > 0 {
		var langDetails []string
		for lang, count := range languageStats {
			langDetails = append(langDetails, fmt.Sprintf("%s(%d)", lang, count))
		}
		if len(langDetails) <= 5 {
			details = append(details, fmt.Sprintf("  • **言語分布**: %s", strings.Join(langDetails, ", ")))
		}
	}

	// 大きなファイルの警告
	if len(largeFiles) > 0 {
		details = append(details, "  • **大きなファイル**:")
		for i, file := range largeFiles {
			if i < 3 { // 最大3つまで表示
				details = append(details, fmt.Sprintf("    - ⚠️ %s", file))
			}
		}
	}

	return strings.Join(details, "\n")
}

// performSecurityAnalysisImpl はセキュリティ分析を実行
func (ism *interactiveSessionManager) performSecurityAnalysisImpl() string {
	var details []string
	details = append(details, "🔒 **セキュリティ分析**")

	projectPath, err := os.Getwd()
	if err != nil {
		return ""
	}

	var sensitiveFiles []string

	// セキュリティ関連のファイルパターンをチェック
	securityPatterns := map[string]string{
		"password": "パスワード関連",
		"secret":   "シークレット情報",
		"api_key":  "APIキー",
		"token":    "トークン",
		"private":  "プライベート情報",
	}

	filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// 隠しファイル・ディレクトリをスキップ
		if strings.Contains(path, "/.") {
			return nil
		}

		filename := strings.ToLower(filepath.Base(path))
		for pattern, description := range securityPatterns {
			if strings.Contains(filename, pattern) {
				sensitiveFiles = append(sensitiveFiles, fmt.Sprintf("%s (%s)", path, description))
			}
		}

		// 設定ファイルで機密情報の可能性があるものをチェック
		if strings.HasSuffix(filename, ".env") ||
			strings.HasSuffix(filename, ".config") ||
			strings.HasSuffix(filename, ".yaml") ||
			strings.HasSuffix(filename, ".yml") {
			sensitiveFiles = append(sensitiveFiles, fmt.Sprintf("%s (設定ファイル)", path))
		}

		return nil
	})

	// 結果の構築
	if len(sensitiveFiles) > 0 {
		details = append(details, fmt.Sprintf("  • **注意を要するファイル**: %d件", len(sensitiveFiles)))
		for i, file := range sensitiveFiles {
			if i < 5 { // 最大5つまで表示
				details = append(details, fmt.Sprintf("    - ⚠️ %s", file))
			}
		}
	} else {
		details = append(details, "  • ✅ **機密性の懸念**: 明らかなセキュリティリスクは検出されませんでした")
	}

	// Git関連のセキュリティチェック
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); err == nil {
		gitignoreExists := false
		if _, err := os.Stat(filepath.Join(projectPath, ".gitignore")); err == nil {
			gitignoreExists = true
		}

		if gitignoreExists {
			details = append(details, "  • ✅ **.gitignore**: 存在します")
		} else {
			details = append(details, "  • ⚠️ **.gitignore**: 存在しません（推奨）")
		}
	}

	return strings.Join(details, "\n")
}

// formatComprehensiveAnalysisResponse は包括的分析レスポンスをフォーマット
func (ism *interactiveSessionManager) formatComprehensiveAnalysisResponse(query string, components []string) string {
	if len(components) == 0 {
		return fmt.Sprintf("🔍 **分析完了**\n\n**クエリ**: %s\n\n分析を実行しましたが、詳細な結果を取得できませんでした。", query)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("🔍 **高度な分析結果**\n\n"))
	result.WriteString(fmt.Sprintf("**クエリ**: %s\n\n", query))

	// 分析コンポーネントを結合
	result.WriteString(strings.Join(components, "\n\n"))

	// 総合的なアクション提案を追加
	result.WriteString("\n\n💡 **推奨アクション**")
	result.WriteString("\n  • 分析結果を基にした具体的な改善を検討してください")
	result.WriteString("\n  • コード品質・構造の最適化を進めてください")
	result.WriteString("\n  • セキュリティ面での懸念があれば優先的に対応してください")
	result.WriteString("\n  • プロジェクトの健全性向上を継続的に行ってください")

	return result.String()
}

// 補助メソッド
func (ism *interactiveSessionManager) getConfidenceEmoji(confidence float64) string {
	if confidence >= 0.8 {
		return "🟢"
	} else if confidence >= 0.6 {
		return "🟡"
	} else {
		return "🔴"
	}
}

func (ism *interactiveSessionManager) getUncertaintyLevel(entropy float64) string {
	if entropy < 0.3 {
		return "低"
	} else if entropy < 0.7 {
		return "中"
	} else {
		return "高"
	}
}

func (ism *interactiveSessionManager) formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}

func (ism *interactiveSessionManager) getLanguageFromExtension(ext string) string {
	languages := map[string]string{
		".go":    "Go",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".py":    "Python",
		".java":  "Java",
		".cpp":   "C++",
		".c":     "C",
		".rs":    "Rust",
		".php":   "PHP",
		".rb":    "Ruby",
		".swift": "Swift",
		".kt":    "Kotlin",
		".cs":    "C#",
		".html":  "HTML",
		".css":   "CSS",
		".sql":   "SQL",
		".sh":    "Shell",
		".yml":   "YAML",
		".yaml":  "YAML",
		".json":  "JSON",
		".xml":   "XML",
		".md":    "Markdown",
	}

	return languages[strings.ToLower(ext)]
}

// addMetaInfoToResponse は応答にリアルタイムメタ情報を追加
func (ism *interactiveSessionManager) addMetaInfoToResponse(response *InteractionResponse, startTime time.Time, modelName string, promptLength int) {
	responseTime := time.Since(startTime)

	// トークン数の概算（プロンプト長 / 4）
	estimatedTokens := promptLength / 4

	// メタ情報をメッセージの末尾に追加
	metaInfo := fmt.Sprintf("\n\n---\n⏱️ **応答時間**: %v | 🤖 **モデル**: %s | 📊 **推定トークン**: %d",
		responseTime.Round(time.Millisecond),
		modelName,
		estimatedTokens)

	response.Message += metaInfo

	// メタデータにも詳細情報を追加
	if response.Metadata == nil {
		response.Metadata = make(map[string]string)
	}
	response.Metadata["response_time_ms"] = fmt.Sprintf("%d", responseTime.Milliseconds())
	response.Metadata["model_name"] = modelName
	response.Metadata["estimated_tokens"] = fmt.Sprintf("%d", estimatedTokens)
	response.Metadata["prompt_length"] = fmt.Sprintf("%d", promptLength)
}
