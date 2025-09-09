package interactive

import (
	"context"
	"time"

	"github.com/glkt/vyb-code/internal/contextmanager"
)

// インタラクティブセッションの状態
type SessionState int

const (
	SessionStateIdle SessionState = iota
	SessionStateWaitingForInput
	SessionStateProcessing
	SessionStateWaitingForConfirmation
	SessionStateExecuting
	SessionStateError
)

// コーディングセッションの種類
type CodingSessionType int

const (
	CodingSessionTypeGeneral   CodingSessionType = iota // 一般的なコーディング
	CodingSessionTypeDebugging                          // デバッグ作業
	CodingSessionTypeRefactor                           // リファクタリング
	CodingSessionTypeReview                             // コードレビュー
	CodingSessionTypeLearning                           // 学習・説明
)

// インタラクティブセッション
type InteractiveSession struct {
	ID                string                        `json:"id"`
	State             SessionState                  `json:"state"`
	Type              CodingSessionType             `json:"type"`
	StartTime         time.Time                     `json:"start_time"`
	LastActivity      time.Time                     `json:"last_activity"`
	CurrentFile       string                        `json:"current_file"`
	CurrentFunction   string                        `json:"current_function"`
	CurrentLine       int                           `json:"current_line"`
	UserIntent        string                        `json:"user_intent"`
	WorkingContext    []*contextmanager.ContextItem `json:"working_context"`
	PendingSuggestion *CodeSuggestion               `json:"pending_suggestion,omitempty"`
	SessionMetadata   map[string]string             `json:"session_metadata"`
	Metrics           *SessionMetrics               `json:"metrics"`
	LastCommandOutput string                        `json:"last_command_output,omitempty"` // 最後のコマンド実行結果
}

// コード提案
type CodeSuggestion struct {
	ID            string            `json:"id"`
	Type          SuggestionType    `json:"type"`
	OriginalCode  string            `json:"original_code"`
	SuggestedCode string            `json:"suggested_code"`
	Explanation   string            `json:"explanation"`
	Confidence    float64           `json:"confidence"` // 0.0-1.0
	ImpactLevel   ImpactLevel       `json:"impact_level"`
	FilePath      string            `json:"file_path"`
	LineRange     [2]int            `json:"line_range"` // [start, end]
	Metadata      map[string]string `json:"metadata"`
	CreatedAt     time.Time         `json:"created_at"`
	UserConfirmed bool              `json:"user_confirmed"`
	Applied       bool              `json:"applied"`
}

// 提案の種類
type SuggestionType int

const (
	SuggestionTypeImprovement    SuggestionType = iota // 改善提案
	SuggestionTypeBugFix                               // バグ修正
	SuggestionTypeOptimization                         // パフォーマンス最適化
	SuggestionTypeRefactoring                          // リファクタリング
	SuggestionTypeDocumentation                        // ドキュメント追加
	SuggestionTypeSecurity                             // セキュリティ修正
	SuggestionTypeTestGeneration                       // テスト生成
)

// 影響レベル
type ImpactLevel int

const (
	ImpactLevelLow      ImpactLevel = iota // 低影響（コメント、フォーマット等）
	ImpactLevelMedium                      // 中影響（ロジック改善等）
	ImpactLevelHigh                        // 高影響（アーキテクチャ変更等）
	ImpactLevelCritical                    // 重大（セキュリティ等）
)

// セッションメトリクス
type SessionMetrics struct {
	TotalInteractions     int           `json:"total_interactions"`
	CodeSuggestionsGiven  int           `json:"code_suggestions_given"`
	SuggestionsAccepted   int           `json:"suggestions_accepted"`
	SuggestionsRejected   int           `json:"suggestions_rejected"`
	FilesModified         int           `json:"files_modified"`
	LinesChanged          int           `json:"lines_changed"`
	AverageResponseTime   time.Duration `json:"average_response_time"`
	TotalThinkingTime     time.Duration `json:"total_thinking_time"`
	UserSatisfactionScore float64       `json:"user_satisfaction_score"` // 0.0-1.0
}

// インタラクティブセッション管理インターフェース
type SessionManager interface {
	// セッション管理
	CreateSession(sessionType CodingSessionType) (*InteractiveSession, error)
	GetSession(sessionID string) (*InteractiveSession, error)
	UpdateSession(session *InteractiveSession) error
	CloseSession(sessionID string) error
	ListActiveSessions() ([]*InteractiveSession, error)

	// コンテキスト統合管理
	UpdateWorkingContext(sessionID string, contextItems []*contextmanager.ContextItem) error
	GetRelevantContext(sessionID string, query string, maxItems int) ([]*contextmanager.ContextItem, error)

	// コード提案管理
	GenerateCodeSuggestion(ctx context.Context, sessionID string, request *SuggestionRequest) (*CodeSuggestion, error)
	ConfirmSuggestion(sessionID string, suggestionID string, accepted bool) error
	ApplySuggestion(ctx context.Context, sessionID string, suggestionID string) error
	GetSuggestionHistory(sessionID string) ([]*CodeSuggestion, error)

	// インタラクティブ対話
	ProcessUserInput(ctx context.Context, sessionID string, input string) (*InteractionResponse, error)
	GetSessionState(sessionID string) (SessionState, error)
	UpdateSessionState(sessionID string, state SessionState) error

	// メトリクス
	GetSessionMetrics(sessionID string) (*SessionMetrics, error)
	UpdateSessionMetrics(sessionID string, metrics *SessionMetrics) error
}

// 提案リクエスト
type SuggestionRequest struct {
	Type            SuggestionType    `json:"type"`
	FilePath        string            `json:"file_path"`
	Code            string            `json:"code"`
	LineRange       [2]int            `json:"line_range"`
	UserDescription string            `json:"user_description"`
	Context         map[string]string `json:"context"`
	Priority        int               `json:"priority"` // 1-10
}

// インタラクション応答
type InteractionResponse struct {
	SessionID            string                        `json:"session_id"`
	ResponseType         ResponseType                  `json:"response_type"`
	Message              string                        `json:"message"`
	Suggestions          []*CodeSuggestion             `json:"suggestions,omitempty"`
	ContextUpdate        []*contextmanager.ContextItem `json:"context_update,omitempty"`
	RequiresConfirmation bool                          `json:"requires_confirmation"`
	Metadata             map[string]string             `json:"metadata"`
	GeneratedAt          time.Time                     `json:"generated_at"`
}

// 応答の種類
type ResponseType int

const (
	ResponseTypeMessage        ResponseType = iota // 単純なメッセージ
	ResponseTypeCodeSuggestion                     // コード提案
	ResponseTypeQuestion                           // ユーザーへの質問
	ResponseTypeConfirmation                       // 確認要求
	ResponseTypeError                              // エラー応答
	ResponseTypeCompletion                         // 作業完了
)

// Claude Code風の自然な対話フロー管理
type ConversationFlow struct {
	CurrentStep    FlowStep          `json:"current_step"`
	StepHistory    []FlowStep        `json:"step_history"`
	UserGoal       string            `json:"user_goal"`
	Progress       float64           `json:"progress"` // 0.0-1.0
	EstimatedSteps int               `json:"estimated_steps"`
	CompletedSteps int               `json:"completed_steps"`
	NextSteps      []string          `json:"next_steps"`
	FlowMetadata   map[string]string `json:"flow_metadata"`
}

// フローステップ
type FlowStep struct {
	StepID      string            `json:"step_id"`
	StepType    FlowStepType      `json:"step_type"`
	Description string            `json:"description"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     *time.Time        `json:"end_time,omitempty"`
	Result      map[string]string `json:"result"`
	Success     bool              `json:"success"`
}

// フローステップの種類
type FlowStepType int

const (
	FlowStepTypeUnderstanding  FlowStepType = iota // 要求の理解
	FlowStepTypeAnalysis                           // コード分析
	FlowStepTypePlanning                           // 実装計画
	FlowStepTypeImplementation                     // 実装
	FlowStepTypeTesting                            // テスト
	FlowStepTypeVerification                       // 検証
	FlowStepTypeCompletion                         // 完了
)

// バイブ体験エンジンの設定
type VibeConfig struct {
	// 応答スタイル設定
	ResponseStyle  ResponseStyle `json:"response_style"`
	Personality    string        `json:"personality"`     // "helpful", "concise", "detailed"
	TechnicalLevel int           `json:"technical_level"` // 1-10 (初心者-エキスパート)

	// タイミング設定
	MaxResponseTime   time.Duration `json:"max_response_time"`
	IdealResponseTime time.Duration `json:"ideal_response_time"`
	ThinkingTimeRatio float64       `json:"thinking_time_ratio"` // 0.0-1.0

	// 提案頻度設定
	ProactiveSuggestions bool          `json:"proactive_suggestions"`
	SuggestionFrequency  time.Duration `json:"suggestion_frequency"`

	// 学習設定
	LearnUserPreferences bool `json:"learn_user_preferences"`
	AdaptToWorkingStyle  bool `json:"adapt_to_working_style"`
}

// 応答スタイル
type ResponseStyle int

const (
	ResponseStyleConcise     ResponseStyle = iota // 簡潔
	ResponseStyleDetailed                         // 詳細
	ResponseStyleInteractive                      // インタラクティブ
	ResponseStyleEducational                      // 教育的
)
