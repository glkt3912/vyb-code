package session

import (
	"os"
	"testing"
	"time"
)

// TestNewManager は新しいセッションマネージャー作成をテストする
func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("セッションマネージャーの作成に失敗")
	}

	if manager.sessions == nil {
		t.Error("セッションマップが初期化されていません")
	}

	if manager.maxSessions != 100 {
		t.Errorf("期待値: 100, 実際値: %d", manager.maxSessions)
	}

	if manager.maxTurnAge != 30*24*time.Hour {
		t.Errorf("期待値: %v, 実際値: %v", 30*24*time.Hour, manager.maxTurnAge)
	}
}

// TestInitialize はセッションマネージャーの初期化をテストする
func TestInitialize(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	
	manager := &Manager{
		sessionsDir: tempDir,
		sessions:    make(map[string]*Session),
		maxSessions: 100,
		maxTurnAge:  30 * 24 * time.Hour,
	}

	err := manager.Initialize()
	if err != nil {
		t.Fatalf("セッションマネージャーの初期化エラー: %v", err)
	}

	// ディレクトリが作成されたことを確認
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("セッションディレクトリが作成されていません")
	}
}

// TestCreateSession はセッション作成をテストする
func TestCreateSession(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	
	manager := &Manager{
		sessionsDir: tempDir,
		sessions:    make(map[string]*Session),
		maxSessions: 100,
		maxTurnAge:  30 * 24 * time.Hour,
	}

	err := manager.Initialize()
	if err != nil {
		t.Fatalf("初期化エラー: %v", err)
	}

	// セッションを作成
	title := "テストセッション"
	model := "test-model"
	provider := "test-provider"

	session, err := manager.CreateSession(title, model, provider)
	if err != nil {
		t.Fatalf("セッション作成エラー: %v", err)
	}

	// セッションの内容を確認
	if session.Title != title {
		t.Errorf("期待タイトル: %s, 実際: %s", title, session.Title)
	}

	if session.Model != model {
		t.Errorf("期待モデル: %s, 実際: %s", model, session.Model)
	}

	if session.Provider != provider {
		t.Errorf("期待プロバイダー: %s, 実際: %s", provider, session.Provider)
	}

	if session.TurnCount != 0 {
		t.Errorf("初期ターン数が0ではありません: %d", session.TurnCount)
	}

	// マネージャーの状態確認
	if len(manager.sessions) != 1 {
		t.Errorf("期待セッション数: 1, 実際: %d", len(manager.sessions))
	}

	if manager.currentID != session.ID {
		t.Error("現在のセッションIDが正しく設定されていません")
	}
}

// TestAddTurn はターン追加をテストする
func TestAddTurn(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	
	manager := &Manager{
		sessionsDir: tempDir,
		sessions:    make(map[string]*Session),
		maxSessions: 100,
		maxTurnAge:  30 * 24 * time.Hour,
	}

	manager.Initialize()

	// セッションを作成
	session, err := manager.CreateSession("テスト", "model", "provider")
	if err != nil {
		t.Fatalf("セッション作成エラー: %v", err)
	}

	// ターンを追加
	content := "テスト内容"
	metadata := make(Metadata)
	metadata["test"] = "value"

	err = manager.AddTurn(session.ID, TurnTypeUser, content, metadata)
	if err != nil {
		t.Fatalf("ターン追加エラー: %v", err)
	}

	// セッションを取得して確認
	updatedSession, err := manager.GetSession(session.ID)
	if err != nil {
		t.Fatalf("セッション取得エラー: %v", err)
	}

	if updatedSession.TurnCount != 1 {
		t.Errorf("期待ターン数: 1, 実際: %d", updatedSession.TurnCount)
	}

	if len(updatedSession.Turns) != 1 {
		t.Errorf("期待ターン配列長: 1, 実際: %d", len(updatedSession.Turns))
	}

	turn := updatedSession.Turns[0]
	if turn.Type != TurnTypeUser {
		t.Errorf("期待ターンタイプ: %s, 実際: %s", TurnTypeUser, turn.Type)
	}

	if turn.Content != content {
		t.Errorf("期待内容: %s, 実際: %s", content, turn.Content)
	}
}

// TestGetSession はセッション取得をテストする
func TestGetSession(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	
	manager := &Manager{
		sessionsDir: tempDir,
		sessions:    make(map[string]*Session),
		maxSessions: 100,
		maxTurnAge:  30 * 24 * time.Hour,
	}

	manager.Initialize()

	// セッションを作成
	session, err := manager.CreateSession("テスト", "model", "provider")
	if err != nil {
		t.Fatalf("セッション作成エラー: %v", err)
	}

	// セッションを取得
	retrieved, err := manager.GetSession(session.ID)
	if err != nil {
		t.Fatalf("セッション取得エラー: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("期待ID: %s, 実際: %s", session.ID, retrieved.ID)
	}

	if retrieved.Title != session.Title {
		t.Errorf("期待タイトル: %s, 実際: %s", session.Title, retrieved.Title)
	}

	// 存在しないセッションを取得
	_, err = manager.GetSession("nonexistent")
	if err == nil {
		t.Error("存在しないセッションの取得でエラーが発生しませんでした")
	}
}

// TestListSessions はセッション一覧をテストする
func TestListSessions(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	
	manager := &Manager{
		sessionsDir: tempDir,
		sessions:    make(map[string]*Session),
		maxSessions: 100,
		maxTurnAge:  30 * 24 * time.Hour,
	}

	manager.Initialize()

	// 複数のセッションを作成
	session1, _ := manager.CreateSession("セッション1", "model1", "provider1")
	time.Sleep(time.Millisecond) // 異なる更新時刻を保証
	session2, _ := manager.CreateSession("セッション2", "model2", "provider2")

	// セッション一覧を取得
	sessions := manager.ListSessions()

	if len(sessions) != 2 {
		t.Errorf("期待セッション数: 2, 実際: %d", len(sessions))
	}

	// 新しい順でソートされていることを確認
	if sessions[0].ID != session2.ID {
		t.Error("セッションが更新時間順でソートされていません")
	}

	if sessions[1].ID != session1.ID {
		t.Error("セッションが更新時間順でソートされていません")
	}
}

// TestDeleteSession はセッション削除をテストする
func TestDeleteSession(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	
	manager := &Manager{
		sessionsDir: tempDir,
		sessions:    make(map[string]*Session),
		maxSessions: 100,
		maxTurnAge:  30 * 24 * time.Hour,
	}

	manager.Initialize()

	// セッションを作成
	session, err := manager.CreateSession("テスト", "model", "provider")
	if err != nil {
		t.Fatalf("セッション作成エラー: %v", err)
	}

	// セッションを削除
	err = manager.DeleteSession(session.ID)
	if err != nil {
		t.Fatalf("セッション削除エラー: %v", err)
	}

	// セッションが削除されたことを確認
	if len(manager.sessions) != 0 {
		t.Errorf("セッションが削除されていません: %d", len(manager.sessions))
	}

	if manager.currentID != "" {
		t.Error("現在のセッションIDがクリアされていません")
	}

	// 存在しないセッションの削除
	err = manager.DeleteSession("nonexistent")
	if err == nil {
		t.Error("存在しないセッションの削除でエラーが発生しませんでした")
	}
}

// TestSwitchSession はセッション切り替えをテストする
func TestSwitchSession(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	
	manager := &Manager{
		sessionsDir: tempDir,
		sessions:    make(map[string]*Session),
		maxSessions: 100,
		maxTurnAge:  30 * 24 * time.Hour,
	}

	manager.Initialize()

	// 複数のセッションを作成
	session1, _ := manager.CreateSession("セッション1", "model1", "provider1")
	_, _ = manager.CreateSession("セッション2", "model2", "provider2")

	// セッション2に切り替え
	err := manager.SwitchSession(session1.ID)
	if err != nil {
		t.Fatalf("セッション切り替えエラー: %v", err)
	}

	if manager.currentID != session1.ID {
		t.Errorf("期待現在セッション: %s, 実際: %s", session1.ID, manager.currentID)
	}

	// 存在しないセッションへの切り替え
	err = manager.SwitchSession("nonexistent")
	if err == nil {
		t.Error("存在しないセッションへの切り替えでエラーが発生しませんでした")
	}
}

// TestGetStats はセッション統計をテストする
func TestGetStats(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	
	manager := &Manager{
		sessionsDir: tempDir,
		sessions:    make(map[string]*Session),
		maxSessions: 100,
		maxTurnAge:  30 * 24 * time.Hour,
	}

	manager.Initialize()

	// セッションを作成
	session, _ := manager.CreateSession("テスト", "test-model", "test-provider")
	manager.AddTurn(session.ID, TurnTypeUser, "テスト内容", make(Metadata))

	// 統計を取得
	stats := manager.GetStats()

	if stats.TotalSessions != 1 {
		t.Errorf("期待総セッション数: 1, 実際: %d", stats.TotalSessions)
	}

	if stats.TotalTurns != 1 {
		t.Errorf("期待総ターン数: 1, 実際: %d", stats.TotalTurns)
	}

	if stats.AverageTurns != 1.0 {
		t.Errorf("期待平均ターン数: 1.0, 実際: %f", stats.AverageTurns)
	}

	if stats.ModelUsage["test-model"] != 1 {
		t.Errorf("期待モデル使用数: 1, 実際: %d", stats.ModelUsage["test-model"])
	}
}

// TestGenerateSessionID はセッションID生成をテストする
func TestGenerateSessionID(t *testing.T) {
	manager := NewManager()

	title1 := "テストセッション1"
	title2 := "テストセッション2"
	
	id1 := manager.generateSessionID(title1)
	id2 := manager.generateSessionID(title2)

	// 異なるタイトルで異なるIDが生成されることを確認
	if id1 == id2 {
		t.Error("異なるタイトルで異なるIDが生成されませんでした")
	}

	// IDの形式確認（タイムスタンプ-ハッシュ）
	if len(id1) < 15 { // 最小長チェック
		t.Errorf("生成されたIDが短すぎます: %s", id1)
	}

	// 同じタイトルでも時間が異なれば違うIDになることを確認
	time.Sleep(1 * time.Second) // 秒単位の精度のため1秒待機
	id3 := manager.generateSessionID(title1)
	if id1 == id3 {
		t.Error("同じタイトルでも異なる時刻で異なるIDが生成されませんでした")
	}
}