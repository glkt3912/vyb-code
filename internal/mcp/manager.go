package mcp

import (
	"fmt"
	"sync"

	"github.com/glkt/vyb-code/internal/logger"
)

// MCPマネージャー：複数のMCPサーバーを管理
type Manager struct {
	mu      sync.RWMutex
	clients map[string]*Client
	logger  Logger
}

// 新しいMCPマネージャーを作成
func NewManager() *Manager {
	// 構造化ロガーを使用
	vybLogger := logger.WithComponent("mcp")

	return &Manager{
		clients: make(map[string]*Client),
		logger:  &StructuredLoggerAdapter{vybLogger: vybLogger},
	}
}

// VybLoggerをMCP Logger interfaceに適合させるアダプター
type StructuredLoggerAdapter struct {
	vybLogger *logger.VybLogger
}

func (s *StructuredLoggerAdapter) Debug(msg string, args ...interface{}) {
	s.vybLogger.Debug(msg, args...)
}

func (s *StructuredLoggerAdapter) Info(msg string, args ...interface{}) {
	s.vybLogger.Info(msg, args...)
}

func (s *StructuredLoggerAdapter) Warn(msg string, args ...interface{}) {
	s.vybLogger.Warn(msg, args...)
}

func (s *StructuredLoggerAdapter) Error(msg string, args ...interface{}) {
	s.vybLogger.Error(msg, args...)
}

// サーバーに接続
func (m *Manager) ConnectServer(name string, config ClientConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 既に接続済みの場合はスキップ
	if client, exists := m.clients[name]; exists && client.IsConnected() {
		return fmt.Errorf("MCPサーバー '%s' は既に接続済みです", name)
	}

	// ロガーを設定
	config.Logger = m.logger

	// 新しいクライアントを作成
	client := NewClient(config)

	// 接続を試行
	if err := client.Connect(config); err != nil {
		return fmt.Errorf("MCPサーバー '%s' への接続失敗: %w", name, err)
	}

	m.clients[name] = client
	return nil
}

// サーバーから切断
func (m *Manager) DisconnectServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[name]
	if !exists {
		return fmt.Errorf("MCPサーバー '%s' は接続されていません", name)
	}

	if err := client.Close(); err != nil {
		return fmt.Errorf("MCPサーバー '%s' の切断失敗: %w", name, err)
	}

	delete(m.clients, name)
	return nil
}

// 全サーバーから切断
func (m *Manager) DisconnectAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastError error
	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			lastError = err
			m.logger.Error("MCPサーバー切断失敗", "server", name, "error", err)
		}
	}

	// 全てクリア
	m.clients = make(map[string]*Client)
	return lastError
}

// 接続済みサーバー一覧を取得
func (m *Manager) GetConnectedServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]string, 0, len(m.clients))
	for name, client := range m.clients {
		if client.IsConnected() {
			servers = append(servers, name)
		}
	}

	return servers
}

// 全ツールを取得
func (m *Manager) GetAllTools() map[string][]Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]Tool)
	for name, client := range m.clients {
		if client.IsConnected() {
			result[name] = client.GetTools()
		}
	}

	return result
}

// 特定サーバーのツールを取得
func (m *Manager) GetServerTools(serverName string) ([]Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[serverName]
	if !exists {
		return nil, fmt.Errorf("MCPサーバー '%s' は接続されていません", serverName)
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("MCPサーバー '%s' は切断されています", serverName)
	}

	return client.GetTools(), nil
}

// ツールを実行
func (m *Manager) CallTool(serverName, toolName string, arguments map[string]interface{}) (*ToolResult, error) {
	m.mu.RLock()
	client, exists := m.clients[serverName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("MCPサーバー '%s' は接続されていません", serverName)
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("MCPサーバー '%s' は切断されています", serverName)
	}

	return client.CallTool(toolName, arguments)
}

// 全サーバーの接続状態をチェック
func (m *Manager) HealthCheck() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]bool)
	for name, client := range m.clients {
		status[name] = client.IsConnected() && client.Ping() == nil
	}

	return status
}

// ログレベルを設定
func (m *Manager) SetLogger(logger Logger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logger = logger
}
