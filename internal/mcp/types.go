package mcp

import (
	"time"
)

// MCPメッセージの基本構造
type Message struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPエラー構造
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ツール定義
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// ツール呼び出し要求
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ツール実行結果
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// コンテンツ定義
type Content struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
	URI  string `json:"uri,omitempty"`
}

// リソース定義
type Resource struct {
	URI         string            `json:"uri"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	MIMEType    string            `json:"mimeType,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// MCPサーバー情報
type ServerInfo struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	ProtocolVersion  string            `json:"protocolVersion"`
	Capabilities     ServerCapability  `json:"capabilities"`
	Instructions     string            `json:"instructions,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// サーバー機能定義
type ServerCapability struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Logging   *LoggingCapability   `json:"logging,omitempty"`
}

// ツール機能定義
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// リソース機能定義
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// プロンプト機能定義
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ログ機能定義
type LoggingCapability struct {
	Level string `json:"level,omitempty"` // "debug", "info", "notice", "warning", "error", "critical", "alert", "emergency"
}

// 初期化要求
type InitializeRequest struct {
	ProtocolVersion string           `json:"protocolVersion"`
	Capabilities    ClientCapability `json:"capabilities"`
	ClientInfo      ClientInfo       `json:"clientInfo"`
}

// クライアント機能定義
type ClientCapability struct {
	Sampling *SamplingCapability `json:"sampling,omitempty"`
	Roots    *RootsCapability    `json:"roots,omitempty"`
}

// サンプリング機能定義
type SamplingCapability struct {
	// 現在は空だが将来の拡張用
}

// ルート機能定義
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// クライアント情報
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ルート定義
type Root struct {
	URI  string `json:"uri"`
	Name string `json:"name,omitempty"`
}

// プロンプト定義
type Prompt struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Arguments   []PromptArg     `json:"arguments,omitempty"`
}

// プロンプト引数
type PromptArg struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// ログレベル定義
type LogLevel string

const (
	LogLevelDebug     LogLevel = "debug"
	LogLevelInfo      LogLevel = "info"
	LogLevelNotice    LogLevel = "notice"
	LogLevelWarning   LogLevel = "warning"
	LogLevelError     LogLevel = "error"
	LogLevelCritical  LogLevel = "critical"
	LogLevelAlert     LogLevel = "alert"
	LogLevelEmergency LogLevel = "emergency"
)

// ログエントリ
type LogEntry struct {
	Level   LogLevel    `json:"level"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Logger  string      `json:"logger,omitempty"`
}

// MCPセッション状態
type SessionState struct {
	ID          string            `json:"id"`
	Connected   bool              `json:"connected"`
	ServerInfo  *ServerInfo       `json:"serverInfo,omitempty"`
	Tools       []Tool            `json:"tools"`
	Resources   []Resource        `json:"resources"`
	Prompts     []Prompt          `json:"prompts"`
	LastPing    time.Time         `json:"lastPing"`
	Metadata    map[string]string `json:"metadata"`
}

// MCPイベント定義
type EventType string

const (
	EventToolsListChanged     EventType = "notifications/tools/list_changed"
	EventResourcesListChanged EventType = "notifications/resources/list_changed"
	EventPromptsListChanged   EventType = "notifications/prompts/list_changed"
	EventResourceUpdated      EventType = "notifications/resources/updated"
	EventLogMessage           EventType = "notifications/message"
	EventRootsListChanged     EventType = "notifications/roots/list_changed"
)

// 通知メッセージ
type Notification struct {
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// MCPプロトコルバージョン
const (
	MCPProtocolVersion = "2024-11-05"
	MCPMaxMessageSize  = 1024 * 1024 // 1MB
)