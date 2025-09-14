package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

// UnifiedToolInterface - 統一ツールインターフェース
type UnifiedToolInterface interface {
	// 基本操作
	GetName() string
	GetDescription() string
	GetSchema() ToolSchema
	
	// 実行
	Execute(ctx context.Context, request *ToolRequest) (*ToolResponse, error)
	
	// 設定
	Configure(config map[string]interface{}) error
	ValidateRequest(request *ToolRequest) error
	
	// メタデータ
	GetCapabilities() []ToolCapability
	GetVersion() string
	IsEnabled() bool
}

// ToolSchema - ツールスキーマ統一定義
type ToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Parameters  map[string]Parameter   `json:"parameters"`
	Required    []string               `json:"required"`
	Examples    []ToolExample          `json:"examples,omitempty"`
}

// Parameter - パラメータ定義
type Parameter struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Format      string      `json:"format,omitempty"`
	Pattern     string      `json:"pattern,omitempty"`
	MinLength   *int        `json:"minLength,omitempty"`
	MaxLength   *int        `json:"maxLength,omitempty"`
	Minimum     *float64    `json:"minimum,omitempty"`
	Maximum     *float64    `json:"maximum,omitempty"`
}

// ToolExample - ツール使用例
type ToolExample struct {
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Expected    string                 `json:"expected,omitempty"`
}

// ToolRequest - 統一ツールリクエスト
type ToolRequest struct {
	ID          string                 `json:"id"`
	ToolName    string                 `json:"tool_name"`
	Parameters  map[string]interface{} `json:"parameters"`
	Context     *RequestContext        `json:"context,omitempty"`
	Options     *RequestOptions        `json:"options,omitempty"`
}

// ToolResponse - 統一ツールレスポンス
type ToolResponse struct {
	ID          string                 `json:"id"`
	ToolName    string                 `json:"tool_name"`
	Success     bool                   `json:"success"`
	Content     string                 `json:"content,omitempty"`
	Data        interface{}            `json:"data,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    *ResponseMetadata      `json:"metadata,omitempty"`
	Duration    time.Duration          `json:"duration"`
}

// RequestContext - リクエストコンテキスト
type RequestContext struct {
	WorkingDir  string            `json:"working_dir,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	UserID      string            `json:"user_id,omitempty"`
}

// RequestOptions - リクエストオプション
type RequestOptions struct {
	Timeout    time.Duration `json:"timeout,omitempty"`
	Async      bool          `json:"async,omitempty"`
	Streaming  bool          `json:"streaming,omitempty"`
	Debug      bool          `json:"debug,omitempty"`
	DryRun     bool          `json:"dry_run,omitempty"`
}

// ResponseMetadata - レスポンスメタデータ
type ResponseMetadata struct {
	ExecutionTime time.Duration          `json:"execution_time"`
	ResourceUsage *ResourceUsage         `json:"resource_usage,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	Debug         map[string]interface{} `json:"debug,omitempty"`
}

// ResourceUsage - リソース使用状況
type ResourceUsage struct {
	CPUTime    time.Duration `json:"cpu_time,omitempty"`
	MemoryUsed int64         `json:"memory_used,omitempty"`
	DiskRead   int64         `json:"disk_read,omitempty"`
	DiskWrite  int64         `json:"disk_write,omitempty"`
	NetworkIO  int64         `json:"network_io,omitempty"`
}

// ToolCapability - ツール機能
type ToolCapability string

const (
	CapabilityFileRead    ToolCapability = "file_read"
	CapabilityFileWrite   ToolCapability = "file_write"
	CapabilityFileEdit    ToolCapability = "file_edit"
	CapabilityCommand     ToolCapability = "command_execution"
	CapabilityNetwork     ToolCapability = "network_access"
	CapabilitySearch      ToolCapability = "search"
	CapabilityGit         ToolCapability = "git_operations"
	CapabilityInteractive ToolCapability = "interactive"
	CapabilityStreaming   ToolCapability = "streaming"
	CapabilityAsync       ToolCapability = "async_execution"
)

// ToolCategory - ツールカテゴリ
type ToolCategory string

const (
	CategoryFile    ToolCategory = "file_operations"
	CategoryCommand ToolCategory = "command_execution" 
	CategorySearch  ToolCategory = "search_operations"
	CategoryWeb     ToolCategory = "web_operations"
	CategoryGit     ToolCategory = "git_operations"
	CategoryAnalysis ToolCategory = "code_analysis"
	CategoryUtility ToolCategory = "utility"
)

// BaseTool - 基本ツール実装
type BaseTool struct {
	name         string
	description  string
	version      string
	category     ToolCategory
	capabilities []ToolCapability
	schema       ToolSchema
	constraints  *security.Constraints
	enabled      bool
	config       map[string]interface{}
}

// NewBaseTool - 新しい基本ツールを作成
func NewBaseTool(name, description, version string, category ToolCategory) *BaseTool {
	return &BaseTool{
		name:         name,
		description:  description,
		version:      version,
		category:     category,
		capabilities: make([]ToolCapability, 0),
		enabled:      true,
		config:       make(map[string]interface{}),
	}
}

// GetName - ツール名を取得
func (bt *BaseTool) GetName() string {
	return bt.name
}

// GetDescription - ツール説明を取得
func (bt *BaseTool) GetDescription() string {
	return bt.description
}

// GetSchema - ツールスキーマを取得
func (bt *BaseTool) GetSchema() ToolSchema {
	return bt.schema
}

// GetCapabilities - ツール機能一覧を取得
func (bt *BaseTool) GetCapabilities() []ToolCapability {
	return bt.capabilities
}

// GetVersion - バージョンを取得
func (bt *BaseTool) GetVersion() string {
	return bt.version
}

// IsEnabled - 有効状態を取得
func (bt *BaseTool) IsEnabled() bool {
	return bt.enabled
}

// Configure - ツール設定
func (bt *BaseTool) Configure(config map[string]interface{}) error {
	bt.config = config
	return nil
}

// Execute - デフォルト実行（サブクラスでオーバーライドする）
func (bt *BaseTool) Execute(ctx context.Context, request *ToolRequest) (*ToolResponse, error) {
	return nil, fmt.Errorf("Execute method not implemented for tool: %s", bt.name)
}

// ValidateRequest - リクエスト検証
func (bt *BaseTool) ValidateRequest(request *ToolRequest) error {
	if request == nil {
		return ErrInvalidRequest
	}
	
	if request.ToolName != bt.name {
		return ErrToolNameMismatch
	}
	
	// 必須パラメータチェック
	for _, required := range bt.schema.Required {
		if _, exists := request.Parameters[required]; !exists {
			return NewValidationError("required parameter missing: " + required)
		}
	}
	
	return nil
}

// AddCapability - 機能を追加
func (bt *BaseTool) AddCapability(cap ToolCapability) {
	for _, existing := range bt.capabilities {
		if existing == cap {
			return // 重複を避ける
		}
	}
	bt.capabilities = append(bt.capabilities, cap)
}

// SetSchema - スキーマを設定
func (bt *BaseTool) SetSchema(schema ToolSchema) {
	bt.schema = schema
}

// SetConstraints - セキュリティ制約を設定
func (bt *BaseTool) SetConstraints(constraints *security.Constraints) {
	bt.constraints = constraints
}

// 共通エラー定義
var (
	ErrInvalidRequest    = NewToolError("invalid_request", "Invalid tool request")
	ErrToolNameMismatch  = NewToolError("tool_name_mismatch", "Tool name mismatch")
	ErrToolNotEnabled    = NewToolError("tool_not_enabled", "Tool is not enabled")
	ErrExecutionTimeout  = NewToolError("execution_timeout", "Tool execution timeout")
	ErrInvalidParameters = NewToolError("invalid_parameters", "Invalid parameters provided")
	ErrSecurityViolation = NewToolError("security_violation", "Security constraint violation")
)

// ToolError - ツールエラー
type ToolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewToolError - 新しいツールエラーを作成
func NewToolError(code, message string) *ToolError {
	return &ToolError{
		Code:    code,
		Message: message,
	}
}

// Error - エラーメッセージを返す
func (e *ToolError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// ValidationError - バリデーションエラー
type ValidationError struct {
	*ToolError
}

// NewValidationError - 新しいバリデーションエラーを作成
func NewValidationError(details string) *ValidationError {
	return &ValidationError{
		ToolError: &ToolError{
			Code:    "validation_error",
			Message: "Validation failed",
			Details: details,
		},
	}
}

// ExecutionError - 実行エラー
type ExecutionError struct {
	*ToolError
	ExitCode int `json:"exit_code,omitempty"`
}

// NewExecutionError - 新しい実行エラーを作成
func NewExecutionError(message string, exitCode int) *ExecutionError {
	return &ExecutionError{
		ToolError: &ToolError{
			Code:    "execution_error", 
			Message: message,
		},
		ExitCode: exitCode,
	}
}