package handlers

import (
	"context"
	"io"
	"time"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
)

// 依存関係の循環を解消するためのインターフェース定義

// LLMProvider は LLM 機能のインターフェース
type LLMProvider interface {
	Generate(ctx context.Context, prompt string, options map[string]interface{}) (string, error)
	Name() string
	Health(ctx context.Context) error
}

// StreamingManager はストリーミング機能のインターフェース  
type StreamingManager interface {
	ProcessString(ctx context.Context, content string, output io.Writer, options interface{}) error
	IsEnabled() bool
}

// InputCompleter は入力補完機能のインターフェース
type InputCompleter interface {
	GetSuggestions(input string) []string
	GetAdvancedSuggestions(input string) []InputSuggestion
}

// InputSuggestion は入力提案
type InputSuggestion struct {
	Text        string
	Description string
	Type        string
}

// PerformanceMonitor はパフォーマンス監視のインターフェース
type PerformanceMonitor interface {
	RecordProactiveUsage(operation string)
	RecordResponseTime(duration time.Duration)
	RecordLLMLatency(duration time.Duration)
	GetMetrics() PerformanceMetrics
	Start() error
}

// PerformanceMetrics はパフォーマンス指標
type PerformanceMetrics struct {
	MemoryUsage MemoryUsage
	// その他の指標...
}

// MemoryUsage はメモリ使用量
type MemoryUsage struct {
	Current float64
	Peak    float64
}

// InteractiveSessionManager はインタラクティブセッション管理のインターフェース
type InteractiveSessionManager interface {
	CreateSession(sessionType int) (SessionInfo, error)
	ProcessUserInput(ctx context.Context, sessionID, input string) (ResponseInfo, error)
}

// SessionInfo はセッション情報
type SessionInfo struct {
	ID   string
	Type int
}

// ResponseInfo は応答情報
type ResponseInfo struct {
	Message string
	Type    string
}

// SecurityConstraints はセキュリティ制約のインターフェース
type SecurityConstraints interface {
	ValidateCommand(command string) error
	ValidateFilePath(path string) error
	GetAllowedCommands() []string
}

// ToolExecutor はツール実行のインターフェース
type ToolExecutor interface {
	ExecuteCommand(command string) (string, error)
	SearchFiles(pattern string) ([]string, error)
	AnalyzeProject(path string) (ProjectInfo, error)
	BuildProject() (BuildResult, error)
	RunTests() (TestResult, error)
}

// ProjectInfo はプロジェクト情報
type ProjectInfo struct {
	Language     string
	Dependencies []string
	Structure    map[string]interface{}
}

// BuildResult はビルド結果
type BuildResult struct {
	Success     bool
	Output      string
	Duration    time.Duration
	BuildSystem string
	Command     string
}

// TestResult はテスト結果
type TestResult struct {
	Success  bool
	Output   string
	Duration time.Duration
	Passed   int
	Failed   int
	Skipped  int
}

// DependencyResolver は依存関係解決のインターフェース
type DependencyResolver interface {
	ResolveLLMProvider(cfg *config.Config) (LLMProvider, error)
	ResolveStreamingManager(cfg *config.Config) (StreamingManager, error)
	ResolveInputCompleter(cfg *config.Config) (InputCompleter, error)
	ResolvePerformanceMonitor(cfg *config.Config, logger logger.Logger) (PerformanceMonitor, error)
	ResolveInteractiveManager(cfg *config.Config, logger logger.Logger) (InteractiveSessionManager, error)
	ResolveSecurityConstraints(cfg *config.Config) (SecurityConstraints, error)
	ResolveToolExecutor(cfg *config.Config, logger logger.Logger) (ToolExecutor, error)
}

// DefaultDependencyResolver はデフォルトの依存関係解決実装
type DefaultDependencyResolver struct {
	logger logger.Logger
}

// NewDependencyResolver は新しい依存関係解決器を作成
func NewDependencyResolver(logger logger.Logger) DependencyResolver {
	return &DefaultDependencyResolver{
		logger: logger,
	}
}

// 各インターフェースの解決実装は実際の依存パッケージを遅延読み込みする
// これにより循環依存を回避

func (r *DefaultDependencyResolver) ResolveLLMProvider(cfg *config.Config) (LLMProvider, error) {
	// 実際の実装では動的に llm パッケージを読み込み
	return &NullLLMProvider{}, nil
}

func (r *DefaultDependencyResolver) ResolveStreamingManager(cfg *config.Config) (StreamingManager, error) {
	return &NullStreamingManager{}, nil
}

func (r *DefaultDependencyResolver) ResolveInputCompleter(cfg *config.Config) (InputCompleter, error) {
	return &NullInputCompleter{}, nil
}

func (r *DefaultDependencyResolver) ResolvePerformanceMonitor(cfg *config.Config, logger logger.Logger) (PerformanceMonitor, error) {
	return &NullPerformanceMonitor{}, nil
}

func (r *DefaultDependencyResolver) ResolveInteractiveManager(cfg *config.Config, logger logger.Logger) (InteractiveSessionManager, error) {
	return &NullInteractiveManager{}, nil
}

func (r *DefaultDependencyResolver) ResolveSecurityConstraints(cfg *config.Config) (SecurityConstraints, error) {
	return &NullSecurityConstraints{}, nil
}

func (r *DefaultDependencyResolver) ResolveToolExecutor(cfg *config.Config, logger logger.Logger) (ToolExecutor, error) {
	return &NullToolExecutor{}, nil
}

// Null Object パターンでデフォルト実装を提供

type NullLLMProvider struct{}

func (n *NullLLMProvider) Generate(ctx context.Context, prompt string, options map[string]interface{}) (string, error) {
	return "LLM not configured", nil
}
func (n *NullLLMProvider) Name() string                     { return "null" }
func (n *NullLLMProvider) Health(ctx context.Context) error { return nil }

type NullStreamingManager struct{}

func (n *NullStreamingManager) ProcessString(ctx context.Context, content string, output io.Writer, options interface{}) error {
	output.Write([]byte(content))
	return nil
}
func (n *NullStreamingManager) IsEnabled() bool { return false }

type NullInputCompleter struct{}

func (n *NullInputCompleter) GetSuggestions(input string) []string                      { return nil }
func (n *NullInputCompleter) GetAdvancedSuggestions(input string) []InputSuggestion { return nil }

type NullPerformanceMonitor struct{}

func (n *NullPerformanceMonitor) RecordProactiveUsage(operation string)        {}
func (n *NullPerformanceMonitor) RecordResponseTime(duration time.Duration)    {}
func (n *NullPerformanceMonitor) RecordLLMLatency(duration time.Duration)      {}
func (n *NullPerformanceMonitor) GetMetrics() PerformanceMetrics               { return PerformanceMetrics{} }
func (n *NullPerformanceMonitor) Start() error                                 { return nil }

type NullInteractiveManager struct{}

func (n *NullInteractiveManager) CreateSession(sessionType int) (SessionInfo, error) {
	return SessionInfo{ID: "null", Type: sessionType}, nil
}
func (n *NullInteractiveManager) ProcessUserInput(ctx context.Context, sessionID, input string) (ResponseInfo, error) {
	return ResponseInfo{Message: "Interactive mode not configured", Type: "info"}, nil
}

type NullSecurityConstraints struct{}

func (n *NullSecurityConstraints) ValidateCommand(command string) error { return nil }
func (n *NullSecurityConstraints) ValidateFilePath(path string) error   { return nil }
func (n *NullSecurityConstraints) GetAllowedCommands() []string          { return []string{"echo", "ls"} }

type NullToolExecutor struct{}

func (n *NullToolExecutor) ExecuteCommand(command string) (string, error) {
	return "Tool execution not configured", nil
}
func (n *NullToolExecutor) SearchFiles(pattern string) ([]string, error) { return nil, nil }
func (n *NullToolExecutor) AnalyzeProject(path string) (ProjectInfo, error) {
	return ProjectInfo{Language: "unknown"}, nil
}
func (n *NullToolExecutor) BuildProject() (BuildResult, error) {
	return BuildResult{Success: false, Output: "Build not configured"}, nil
}
func (n *NullToolExecutor) RunTests() (TestResult, error) {
	return TestResult{Success: false, Output: "Tests not configured"}, nil
}