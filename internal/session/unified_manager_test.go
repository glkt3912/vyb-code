package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glkt/vyb-code/internal/streaming"
)

func TestUnifiedSessionManager_CreateSession(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "vyb-session-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := &ManagerConfig{
		StorageDir:     tempDir,
		AutoSave:       false, // テスト中は自動保存を無効化
		MaxSessions:    10,
		DefaultExpiry:  time.Hour,
		EventQueueSize: 10,
	}

	streamManager := streaming.NewManager(streaming.DefaultStreamConfig())
	manager, err := NewUnifiedSessionManager(config, streamManager, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Shutdown()

	tests := []struct {
		name        string
		sessionType UnifiedSessionType
		config      *SessionConfig
		wantErr     bool
	}{
		{
			name:        "Create persistent session",
			sessionType: SessionTypePersistent,
			config:      nil,
			wantErr:     false,
		},
		{
			name:        "Create chat session",
			sessionType: SessionTypeChat,
			config:      nil,
			wantErr:     false,
		},
		{
			name:        "Create interactive session",
			sessionType: SessionTypeInteractive,
			config:      nil,
			wantErr:     false,
		},
		{
			name:        "Create vibe coding session",
			sessionType: SessionTypeVibeCoding,
			config:      nil,
			wantErr:     false,
		},
		{
			name:        "Create temporary session",
			sessionType: SessionTypeTemporary,
			config:      nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := manager.CreateSession(tt.sessionType, tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if session == nil {
					t.Error("CreateSession() returned nil session")
					return
				}

				if session.Type != tt.sessionType {
					t.Errorf("Session type = %v, want %v", session.Type, tt.sessionType)
				}

				if session.State != SessionStateIdle {
					t.Errorf("Initial session state = %v, want %v", session.State, SessionStateIdle)
				}

				if session.Config == nil {
					t.Error("Session config is nil")
				}

				if session.Stats == nil {
					t.Error("Session stats is nil")
				}
			}
		})
	}
}

func TestUnifiedSessionManager_MessageOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vyb-session-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := &ManagerConfig{
		StorageDir:     tempDir,
		AutoSave:       false,
		MaxSessions:    10,
		EventQueueSize: 10,
	}

	streamManager := streaming.NewManager(streaming.DefaultStreamConfig())
	manager, err := NewUnifiedSessionManager(config, streamManager, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Shutdown()

	// テスト用セッションを作成
	session, err := manager.CreateSession(SessionTypeChat, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Add message", func(t *testing.T) {
		message := &Message{
			Role:    MessageRoleUser,
			Content: "Hello, world!",
		}

		err := manager.AddMessage(session.ID, message)
		if err != nil {
			t.Errorf("AddMessage() error = %v", err)
		}

		// メッセージが追加されたか確認
		messages, err := manager.GetMessages(session.ID, 0, 0)
		if err != nil {
			t.Errorf("GetMessages() error = %v", err)
		}

		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
		}

		if messages[0].Content != "Hello, world!" {
			t.Errorf("Message content = %s, want 'Hello, world!'", messages[0].Content)
		}

		if messages[0].Role != MessageRoleUser {
			t.Errorf("Message role = %s, want %s", messages[0].Role, MessageRoleUser)
		}
	})

	t.Run("Update message", func(t *testing.T) {
		messages, _ := manager.GetMessages(session.ID, 0, 0)
		if len(messages) == 0 {
			t.Skip("No messages to update")
		}

		updatedMessage := messages[0]
		updatedMessage.Content = "Updated content"

		err := manager.UpdateMessage(session.ID, &updatedMessage)
		if err != nil {
			t.Errorf("UpdateMessage() error = %v", err)
		}

		// 更新されたか確認
		updatedMessages, err := manager.GetMessages(session.ID, 0, 0)
		if err != nil {
			t.Errorf("GetMessages() error = %v", err)
		}

		if updatedMessages[0].Content != "Updated content" {
			t.Errorf("Message not updated properly")
		}
	})

	t.Run("Delete message", func(t *testing.T) {
		messages, _ := manager.GetMessages(session.ID, 0, 0)
		if len(messages) == 0 {
			t.Skip("No messages to delete")
		}

		messageID := messages[0].ID

		err := manager.DeleteMessage(session.ID, messageID)
		if err != nil {
			t.Errorf("DeleteMessage() error = %v", err)
		}

		// 削除されたか確認
		remainingMessages, err := manager.GetMessages(session.ID, 0, 0)
		if err != nil {
			t.Errorf("GetMessages() error = %v", err)
		}

		if len(remainingMessages) != 0 {
			t.Errorf("Expected 0 messages after deletion, got %d", len(remainingMessages))
		}
	})
}

func TestUnifiedSessionManager_SessionStateManagement(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vyb-session-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := &ManagerConfig{
		StorageDir:     tempDir,
		AutoSave:       false,
		MaxSessions:    10,
		EventQueueSize: 10,
	}

	streamManager := streaming.NewManager(streaming.DefaultStreamConfig())
	manager, err := NewUnifiedSessionManager(config, streamManager, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Shutdown()

	session, err := manager.CreateSession(SessionTypeInteractive, nil)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		action    func() error
		wantState UnifiedSessionState
	}{
		{
			name:      "Start session",
			action:    func() error { return manager.StartSession(session.ID) },
			wantState: SessionStateActive,
		},
		{
			name:      "Pause session",
			action:    func() error { return manager.PauseSession(session.ID) },
			wantState: SessionStatePaused,
		},
		{
			name:      "Resume session",
			action:    func() error { return manager.ResumeSession(session.ID) },
			wantState: SessionStateActive,
		},
		{
			name:      "Complete session",
			action:    func() error { return manager.CompleteSession(session.ID) },
			wantState: SessionStateCompleted,
		},
		{
			name:      "Archive session",
			action:    func() error { return manager.ArchiveSession(session.ID) },
			wantState: SessionStateArchived,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action()
			if err != nil {
				t.Errorf("Action error = %v", err)
				return
			}

			updatedSession, err := manager.GetSession(session.ID)
			if err != nil {
				t.Errorf("GetSession() error = %v", err)
				return
			}

			if updatedSession.State != tt.wantState {
				t.Errorf("Session state = %v, want %v", updatedSession.State, tt.wantState)
			}
		})
	}
}

func TestUnifiedSessionManager_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vyb-session-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := &ManagerConfig{
		StorageDir:     tempDir,
		AutoSave:       false,
		MaxSessions:    10,
		EventQueueSize: 10,
	}

	streamManager := streaming.NewManager(streaming.DefaultStreamConfig())
	manager, err := NewUnifiedSessionManager(config, streamManager, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// セッションを作成
	session, err := manager.CreateSession(SessionTypePersistent, nil)
	if err != nil {
		t.Fatal(err)
	}

	// メッセージを追加
	message := &Message{
		Role:    MessageRoleUser,
		Content: "Test persistence",
	}
	err = manager.AddMessage(session.ID, message)
	if err != nil {
		t.Fatal(err)
	}

	// セッションを保存
	err = manager.SaveSession(session.ID)
	if err != nil {
		t.Errorf("SaveSession() error = %v", err)
	}

	// ファイルが作成されたか確認
	sessionFile := filepath.Join(tempDir, session.ID+".json")
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Error("Session file was not created")
	}

	// 新しいマネージャーで読み込み
	manager.Shutdown()

	manager2, err := NewUnifiedSessionManager(config, streamManager, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer manager2.Shutdown()

	// セッションが復元されたか確認
	loadedSession, err := manager2.GetSession(session.ID)
	if err != nil {
		t.Errorf("Failed to load session: %v", err)
	}

	if loadedSession.Type != SessionTypePersistent {
		t.Errorf("Loaded session type = %v, want %v", loadedSession.Type, SessionTypePersistent)
	}

	messages, err := manager2.GetMessages(session.ID, 0, 0)
	if err != nil {
		t.Errorf("GetMessages() error = %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message in loaded session, got %d", len(messages))
	}

	if len(messages) > 0 && messages[0].Content != "Test persistence" {
		t.Errorf("Loaded message content = %s, want 'Test persistence'", messages[0].Content)
	}
}

func TestUnifiedSessionManager_SessionFiltering(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vyb-session-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := &ManagerConfig{
		StorageDir:     tempDir,
		AutoSave:       false,
		MaxSessions:    20,
		EventQueueSize: 10,
	}

	streamManager := streaming.NewManager(streaming.DefaultStreamConfig())
	manager, err := NewUnifiedSessionManager(config, streamManager, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Shutdown()

	// 複数のセッションを作成
	chatSession, _ := manager.CreateSession(SessionTypeChat, nil)
	interactiveSession, _ := manager.CreateSession(SessionTypeInteractive, nil)
	vibeSession, _ := manager.CreateSession(SessionTypeVibeCoding, nil)

	// 状態を変更
	manager.StartSession(chatSession.ID)
	manager.StartSession(interactiveSession.ID)
	manager.PauseSession(vibeSession.ID)

	t.Run("Filter by type", func(t *testing.T) {
		filter := &SessionFilter{
			Types: []UnifiedSessionType{SessionTypeChat, SessionTypeInteractive},
		}

		sessions, err := manager.ListSessions(filter, SortByCreatedAt, SortOrderAsc)
		if err != nil {
			t.Errorf("ListSessions() error = %v", err)
		}

		if len(sessions) != 2 {
			t.Errorf("Expected 2 sessions, got %d", len(sessions))
		}

		for _, session := range sessions {
			if session.Type != SessionTypeChat && session.Type != SessionTypeInteractive {
				t.Errorf("Unexpected session type: %v", session.Type)
			}
		}
	})

	t.Run("Filter by state", func(t *testing.T) {
		filter := &SessionFilter{
			States: []UnifiedSessionState{SessionStateActive},
		}

		sessions, err := manager.ListSessions(filter, SortByCreatedAt, SortOrderAsc)
		if err != nil {
			t.Errorf("ListSessions() error = %v", err)
		}

		if len(sessions) != 2 {
			t.Errorf("Expected 2 active sessions, got %d", len(sessions))
		}

		for _, session := range sessions {
			if session.State != SessionStateActive {
				t.Errorf("Expected active session, got state: %v", session.State)
			}
		}
	})

	t.Run("Get active sessions", func(t *testing.T) {
		activeSessions, err := manager.GetActiveSessions()
		if err != nil {
			t.Errorf("GetActiveSessions() error = %v", err)
		}

		if len(activeSessions) != 2 {
			t.Errorf("Expected 2 active sessions, got %d", len(activeSessions))
		}
	})
}

func TestUnifiedSessionManager_Statistics(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vyb-session-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := &ManagerConfig{
		StorageDir:     tempDir,
		AutoSave:       false,
		MaxSessions:    10,
		EventQueueSize: 10,
	}

	streamManager := streaming.NewManager(streaming.DefaultStreamConfig())
	manager, err := NewUnifiedSessionManager(config, streamManager, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Shutdown()

	// セッションを作成
	session, err := manager.CreateSession(SessionTypeChat, nil)
	if err != nil {
		t.Fatal(err)
	}

	// メッセージを追加
	userMessage := &Message{
		Role:       MessageRoleUser,
		Content:    "Hello",
		TokenCount: 5,
	}
	assistantMessage := &Message{
		Role:       MessageRoleAssistant,
		Content:    "Hi there!",
		TokenCount: 3,
	}

	manager.AddMessage(session.ID, userMessage)
	manager.AddMessage(session.ID, assistantMessage)

	t.Run("Session stats", func(t *testing.T) {
		stats, err := manager.GetSessionStats(session.ID)
		if err != nil {
			t.Errorf("GetSessionStats() error = %v", err)
		}

		if stats.MessageCount != 2 {
			t.Errorf("Message count = %d, want 2", stats.MessageCount)
		}

		if stats.UserMessages != 1 {
			t.Errorf("User messages = %d, want 1", stats.UserMessages)
		}

		if stats.AssistantMessages != 1 {
			t.Errorf("Assistant messages = %d, want 1", stats.AssistantMessages)
		}

		if stats.TotalTokens != 8 {
			t.Errorf("Total tokens = %d, want 8", stats.TotalTokens)
		}
	})

	t.Run("Global stats", func(t *testing.T) {
		globalStats, err := manager.GetGlobalStats()
		if err != nil {
			t.Errorf("GetGlobalStats() error = %v", err)
		}

		if globalStats.TotalSessions != 1 {
			t.Errorf("Total sessions = %d, want 1", globalStats.TotalSessions)
		}

		if globalStats.TotalMessages != 2 {
			t.Errorf("Total messages = %d, want 2", globalStats.TotalMessages)
		}

		if globalStats.TotalTokens != 8 {
			t.Errorf("Total tokens = %d, want 8", globalStats.TotalTokens)
		}
	})
}

func BenchmarkUnifiedSessionManager_CreateSession(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "vyb-session-bench-*")
	defer os.RemoveAll(tempDir)

	config := &ManagerConfig{
		StorageDir:     tempDir,
		AutoSave:       false,
		MaxSessions:    1000,
		EventQueueSize: 100,
	}

	streamManager := streaming.NewManager(streaming.DefaultStreamConfig())
	manager, _ := NewUnifiedSessionManager(config, streamManager, nil, nil)
	defer manager.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.CreateSession(SessionTypeTemporary, nil)
		if err != nil {
			b.Errorf("CreateSession() error = %v", err)
		}
	}
}

func BenchmarkUnifiedSessionManager_AddMessage(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "vyb-session-bench-*")
	defer os.RemoveAll(tempDir)

	config := &ManagerConfig{
		StorageDir:     tempDir,
		AutoSave:       false,
		MaxSessions:    10,
		EventQueueSize: 100,
	}

	streamManager := streaming.NewManager(streaming.DefaultStreamConfig())
	manager, _ := NewUnifiedSessionManager(config, streamManager, nil, nil)
	defer manager.Shutdown()

	session, _ := manager.CreateSession(SessionTypeTemporary, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		message := &Message{
			Role:    MessageRoleUser,
			Content: "Benchmark message",
		}
		err := manager.AddMessage(session.ID, message)
		if err != nil {
			b.Errorf("AddMessage() error = %v", err)
		}
	}
}
