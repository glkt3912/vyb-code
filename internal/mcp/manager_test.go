package mcp

import (
	"testing"
)

// MCPマネージャーのテスト
func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("マネージャーの作成に失敗しました")
	}

	if manager.clients == nil {
		t.Error("クライアントマップが初期化されていません")
	}

	if len(manager.clients) != 0 {
		t.Error("新しいマネージャーにクライアントが存在します")
	}
}

// 接続済みサーバー一覧のテスト
func TestGetConnectedServers(t *testing.T) {
	manager := NewManager()

	servers := manager.GetConnectedServers()
	if len(servers) != 0 {
		t.Error("新しいマネージャーに接続済みサーバーが存在します")
	}
}

// 全ツール取得のテスト
func TestGetAllTools(t *testing.T) {
	manager := NewManager()

	tools := manager.GetAllTools()
	if len(tools) != 0 {
		t.Error("新しいマネージャーにツールが存在します")
	}
}

// ヘルスチェックのテスト
func TestHealthCheck(t *testing.T) {
	manager := NewManager()

	status := manager.HealthCheck()
	if len(status) != 0 {
		t.Error("新しいマネージャーにサーバー状態が存在します")
	}
}

// SimpleLoggerのテスト
func TestSimpleLogger(t *testing.T) {
	logger := &SimpleLogger{}

	// ログメソッドが正常に実行されることを確認
	logger.Debug("テストデバッグメッセージ")
	logger.Info("テスト情報メッセージ")
	logger.Warn("テスト警告メッセージ")
	logger.Error("テストエラーメッセージ")

	// エラーが発生しないことを確認
	logger.Debug("フォーマット付きメッセージ: %s", "テスト")
	logger.Info("フォーマット付きメッセージ: %d", 42)
}
