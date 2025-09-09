package interactive

import (
	"testing"
	"time"

	"github.com/glkt/vyb-code/internal/contextmanager"
)

// TestInteractiveSession は基本的なインタラクティブセッションをテストする
func TestInteractiveSession(t *testing.T) {
	session := &InteractiveSession{
		ID:              "test-session-123",
		State:           SessionStateIdle,
		Type:            CodingSessionTypeGeneral,
		StartTime:       time.Now(),
		LastActivity:    time.Now(),
		CurrentFile:     "main.go",
		CurrentFunction: "main",
		CurrentLine:     10,
		UserIntent:      "ファイル作成",
		WorkingContext:  []*contextmanager.ContextItem{},
		SessionMetadata: make(map[string]string),
		Metrics:         &SessionMetrics{},
	}

	if session.ID != "test-session-123" {
		t.Errorf("期待値: test-session-123, 実際値: %s", session.ID)
	}

	if session.State != SessionStateIdle {
		t.Errorf("期待値: %d, 実際値: %d", SessionStateIdle, session.State)
	}

	if session.Type != CodingSessionTypeGeneral {
		t.Errorf("期待値: %d, 実際値: %d", CodingSessionTypeGeneral, session.Type)
	}

	if session.CurrentFile != "main.go" {
		t.Errorf("期待値: main.go, 実際値: %s", session.CurrentFile)
	}
}

// TestCodeSuggestion はコード提案をテストする
func TestCodeSuggestion(t *testing.T) {
	suggestion := &CodeSuggestion{
		ID:            "suggestion-123",
		Type:          SuggestionTypeImprovement,
		OriginalCode:  "func old() {}",
		SuggestedCode: "func improved() {}",
		Explanation:   "関数名を改善しました",
		Confidence:    0.9,
		ImpactLevel:   ImpactLevelMedium,
		FilePath:      "main.go",
		LineRange:     [2]int{10, 12},
		Metadata:      make(map[string]string),
		CreatedAt:     time.Now(),
		UserConfirmed: false,
		Applied:       false,
	}

	if suggestion.ID != "suggestion-123" {
		t.Errorf("期待値: suggestion-123, 実際値: %s", suggestion.ID)
	}

	if suggestion.Type != SuggestionTypeImprovement {
		t.Errorf("期待値: %d, 実際値: %d", SuggestionTypeImprovement, suggestion.Type)
	}

	if suggestion.Confidence != 0.9 {
		t.Errorf("期待値: 0.9, 実際値: %f", suggestion.Confidence)
	}

	if suggestion.ImpactLevel != ImpactLevelMedium {
		t.Errorf("期待値: %d, 実際値: %d", ImpactLevelMedium, suggestion.ImpactLevel)
	}

	if suggestion.UserConfirmed {
		t.Error("UserConfirmed が true になっています")
	}
}

// TestSessionStates はセッション状態をテストする
func TestSessionStates(t *testing.T) {
	states := []SessionState{
		SessionStateIdle,
		SessionStateWaitingForInput,
		SessionStateProcessing,
		SessionStateWaitingForConfirmation,
		SessionStateExecuting,
		SessionStateError,
	}

	for i, state := range states {
		if int(state) != i {
			t.Errorf("セッション状態 %d の値が期待値と異なります: %d", i, int(state))
		}
	}
}

// TestCodingSessionTypes はコーディングセッション種類をテストする
func TestCodingSessionTypes(t *testing.T) {
	types := []CodingSessionType{
		CodingSessionTypeGeneral,
		CodingSessionTypeDebugging,
		CodingSessionTypeRefactor,
		CodingSessionTypeReview,
		CodingSessionTypeLearning,
	}

	for i, sType := range types {
		if int(sType) != i {
			t.Errorf("セッション種類 %d の値が期待値と異なります: %d", i, int(sType))
		}
	}
}

// TestSuggestionTypes は提案タイプをテストする
func TestSuggestionTypes(t *testing.T) {
	types := []SuggestionType{
		SuggestionTypeImprovement,
		SuggestionTypeBugFix,
		SuggestionTypeOptimization,
		SuggestionTypeRefactoring,
		SuggestionTypeDocumentation,
		SuggestionTypeSecurity,
		SuggestionTypeTestGeneration,
	}

	for i, sType := range types {
		if int(sType) != i {
			t.Errorf("提案タイプ %d の値が期待値と異なります: %d", i, int(sType))
		}
	}
}

// TestImpactLevels は影響レベルをテストする
func TestImpactLevels(t *testing.T) {
	levels := []ImpactLevel{
		ImpactLevelLow,
		ImpactLevelMedium,
		ImpactLevelHigh,
		ImpactLevelCritical,
	}

	for i, level := range levels {
		if int(level) != i {
			t.Errorf("影響レベル %d の値が期待値と異なります: %d", i, int(level))
		}
	}
}

// TestInteractionResponse はインタラクション応答をテストする
func TestInteractionResponse(t *testing.T) {
	response := &InteractionResponse{
		SessionID:            "test-session",
		ResponseType:         ResponseTypeMessage,
		Message:              "テスト応答",
		Suggestions:          []*CodeSuggestion{},
		ContextUpdate:        []*contextmanager.ContextItem{},
		RequiresConfirmation: false,
		Metadata:             make(map[string]string),
		GeneratedAt:          time.Now(),
	}

	if response.SessionID != "test-session" {
		t.Errorf("期待値: test-session, 実際値: %s", response.SessionID)
	}

	if response.ResponseType != ResponseTypeMessage {
		t.Errorf("期待値: %d, 実際値: %d", ResponseTypeMessage, response.ResponseType)
	}

	if response.Message != "テスト応答" {
		t.Errorf("期待値: テスト応答, 実際値: %s", response.Message)
	}

	if response.RequiresConfirmation {
		t.Error("RequiresConfirmation が true になっています")
	}
}

// TestSuggestionRequest は提案リクエストをテストする
func TestSuggestionRequest(t *testing.T) {
	request := &SuggestionRequest{
		Type:            SuggestionTypeImprovement,
		FilePath:        "main.go",
		Code:            "func test() {}",
		LineRange:       [2]int{1, 3},
		UserDescription: "関数を改善してください",
		Context:         make(map[string]string),
		Priority:        5,
	}

	if request.Type != SuggestionTypeImprovement {
		t.Errorf("期待値: %d, 実際値: %d", SuggestionTypeImprovement, request.Type)
	}

	if request.FilePath != "main.go" {
		t.Errorf("期待値: main.go, 実際値: %s", request.FilePath)
	}

	if request.Priority != 5 {
		t.Errorf("期待値: 5, 実際値: %d", request.Priority)
	}
}

// TestSessionMetrics はセッションメトリクスをテストする
func TestSessionMetrics(t *testing.T) {
	metrics := &SessionMetrics{
		TotalInteractions:     10,
		CodeSuggestionsGiven:  5,
		SuggestionsAccepted:   3,
		SuggestionsRejected:   2,
		FilesModified:         2,
		LinesChanged:          50,
		AverageResponseTime:   2 * time.Second,
		TotalThinkingTime:     5 * time.Second,
		UserSatisfactionScore: 0.8,
	}

	if metrics.TotalInteractions != 10 {
		t.Errorf("期待値: 10, 実際値: %d", metrics.TotalInteractions)
	}

	if metrics.SuggestionsAccepted != 3 {
		t.Errorf("期待値: 3, 実際値: %d", metrics.SuggestionsAccepted)
	}

	if metrics.UserSatisfactionScore != 0.8 {
		t.Errorf("期待値: 0.8, 実際値: %f", metrics.UserSatisfactionScore)
	}
}
