package conversation

import (
	"testing"
	"time"
)

// TestEfficientChatMessage は効率的会話メッセージをテストする
func TestEfficientChatMessage(t *testing.T) {
	msg := &EfficientChatMessage{
		ID:               "test-msg-123",
		Role:             "user",
		Content:          "テストメッセージ",
		Timestamp:        time.Now(),
		RelevanceScore:   0.8,
		ImportanceScore:  0.9,
		CompressionState: CompressionStateUncompressed,
		OriginalLength:   100,
		AccessCount:      1,
		LastAccessed:     time.Now(),
		Metadata:         make(map[string]string),
	}

	if msg.ID != "test-msg-123" {
		t.Errorf("期待値: test-msg-123, 実際値: %s", msg.ID)
	}

	if msg.Role != "user" {
		t.Errorf("期待値: user, 実際値: %s", msg.Role)
	}

	if msg.Content != "テストメッセージ" {
		t.Errorf("期待値: テストメッセージ, 実際値: %s", msg.Content)
	}

	if msg.RelevanceScore != 0.8 {
		t.Errorf("期待値: 0.8, 実際値: %f", msg.RelevanceScore)
	}

	if msg.CompressionState != CompressionStateUncompressed {
		t.Errorf("期待値: %d, 実際値: %d", CompressionStateUncompressed, msg.CompressionState)
	}
}

// TestConversationThread は会話スレッドをテストする
func TestConversationThread(t *testing.T) {
	thread := &ConversationThread{
		ID:               "thread-123",
		Type:             ConversationTypeInteractive,
		Title:            "テスト会話",
		Messages:         []*EfficientChatMessage{},
		StartTime:        time.Now(),
		LastActivity:     time.Now(),
		MaxMessages:      100,
		CompressionRatio: 0.5,
		TotalTokens:      500,
		CompressedTokens: 250,
		Metadata:         make(map[string]string),
	}

	if thread.ID != "thread-123" {
		t.Errorf("期待値: thread-123, 実際値: %s", thread.ID)
	}

	if thread.Type != ConversationTypeInteractive {
		t.Errorf("期待値: %d, 実際値: %d", ConversationTypeInteractive, thread.Type)
	}

	if thread.Title != "テスト会話" {
		t.Errorf("期待値: テスト会話, 実際値: %s", thread.Title)
	}

	if len(thread.Messages) != 0 {
		t.Errorf("期待値: 0, 実際値: %d", len(thread.Messages))
	}

	if thread.MaxMessages != 100 {
		t.Errorf("期待値: 100, 実際値: %d", thread.MaxMessages)
	}
}

// TestCompressionStates は圧縮状態をテストする
func TestCompressionStates(t *testing.T) {
	states := []CompressionState{
		CompressionStateUncompressed,
		CompressionStatePartial,
		CompressionStateCompressed,
		CompressionStateArchived,
	}

	for i, state := range states {
		if int(state) != i {
			t.Errorf("圧縮状態 %d の値が期待値と異なります: %d", i, int(state))
		}
	}
}

// TestConversationTypes は会話タイプをテストする
func TestConversationTypes(t *testing.T) {
	types := []ConversationType{
		ConversationTypeInteractive,
		ConversationTypeCodeReview,
		ConversationTypeDebugging,
		ConversationTypeLearning,
		ConversationTypePlanning,
	}

	for i, cType := range types {
		if int(cType) != i {
			t.Errorf("会話タイプ %d の値が期待値と異なります: %d", i, int(cType))
		}
	}
}

// TestUserFeedback はユーザーフィードバックをテストする
func TestUserFeedback(t *testing.T) {
	feedback := UserFeedback{
		Type:      FeedbackTypeResponse,
		Rating:    4,
		Helpful:   true,
		Relevant:  true,
		Clear:     false,
		Comments:  "良い応答でした",
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}

	if feedback.Type != FeedbackTypeResponse {
		t.Errorf("期待値: %d, 実際値: %d", FeedbackTypeResponse, feedback.Type)
	}

	if feedback.Rating != 4 {
		t.Errorf("期待値: 4, 実際値: %d", feedback.Rating)
	}

	if !feedback.Helpful {
		t.Error("Helpful が false になっています")
	}

	if feedback.Clear {
		t.Error("Clear が true になっています")
	}
}
