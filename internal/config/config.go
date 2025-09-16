package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MCPサーバー設定
type MCPServerConfig struct {
	Name        string            `json:"name"`        // サーバー名
	Command     []string          `json:"command"`     // 起動コマンド
	Args        []string          `json:"args"`        // コマンド引数
	Environment map[string]string `json:"environment"` // 環境変数
	WorkingDir  string            `json:"workingDir"`  // 作業ディレクトリ
	Enabled     bool              `json:"enabled"`     // 有効/無効
	AutoConnect bool              `json:"autoConnect"` // 自動接続
}

// ログ設定
type LogConfig struct {
	Level         string            `json:"level"`          // ログレベル（debug, info, warn, error）
	Format        string            `json:"format"`         // 出力フォーマット（console, json）
	Output        []string          `json:"output"`         // 出力先（stdout, stderr, file:/path）
	ShowCaller    bool              `json:"show_caller"`    // 呼び出し元表示
	ShowTimestamp bool              `json:"show_timestamp"` // タイムスタンプ表示
	ColorEnabled  bool              `json:"color_enabled"`  // 色付き出力
	FileRotation  bool              `json:"file_rotation"`  // ファイルローテーション
	MaxFileSize   int64             `json:"max_file_size"`  // ログファイル最大サイズ
	Context       map[string]string `json:"context"`        // デフォルトコンテキスト
}

// TUI設定（非推奨 - Claude Code風インターフェースに移行済み）
type TUIConfig struct {
	Enabled      bool   `json:"enabled"`       // TUI有効/無効（非推奨）
	Theme        string `json:"theme"`         // カラーテーマ（非推奨）
	ShowSpinner  bool   `json:"show_spinner"`  // スピナー表示（非推奨）
	ShowProgress bool   `json:"show_progress"` // プログレスバー表示（非推奨）
	Animation    bool   `json:"animation"`     // アニメーション有効（非推奨）
}

// ターミナルモード専用設定
type TerminalModeConfig struct {
	TypingSpeed     int  `json:"typing_speed"`      // タイピング速度（ミリ秒）
	ShowGitInPrompt bool `json:"show_git_prompt"`   // プロンプトにGit情報表示
	ShowProjectInfo bool `json:"show_project_info"` // プロジェクト情報表示
	HistorySize     int  `json:"history_size"`      // 入力履歴サイズ
	EnableSlashCmd  bool `json:"enable_slash_cmd"`  // スラッシュコマンド有効
	AutoSaveSession bool `json:"auto_save_session"` // セッション自動保存
}

// vybの設定情報を管理する構造体
type Config struct {
	// LLM設定
	Provider    string  `json:"provider"`    // LLMプロバイダー（ollama、lmstudio等）
	Model       string  `json:"model"`       // 使用するモデル名
	ModelName   string  `json:"model_name"`  // モデル名（互換性）
	BaseURL     string  `json:"base_url"`    // LLMサーバーのURL
	Timeout     int     `json:"timeout"`     // リクエストタイムアウト（秒）
	Temperature float64 `json:"temperature"` // 生成時の温度パラメータ
	MaxTokens   int     `json:"max_tokens"`  // 最大トークン数
	Stream      bool    `json:"stream"`      // ストリーミング応答

	// システム設定
	MaxFileSize    int64  `json:"max_file_size"`    // 読み込み可能な最大ファイルサイズ
	FileMaxSizeMB  int    `json:"file_max_size_mb"` // ファイル最大サイズ（MB）
	WorkspaceMode  string `json:"workspace_mode"`   // ワークスペースモード（project_only等）
	WorkspacePath  string `json:"workspace_path"`   // 作業ディレクトリパス
	CommandTimeout int    `json:"command_timeout"`  // コマンド実行タイムアウト（秒）
	MaxHistory     int    `json:"max_history"`      // 履歴保持数

	// サブ設定
	MCPServers   map[string]MCPServerConfig `json:"mcp_servers"`   // MCPサーバー設定
	Log          LogConfig                  `json:"log"`           // ログ設定
	Logging      LogConfig                  `json:"logging"`       // ログ設定（互換性）
	TUI          TUIConfig                  `json:"tui"`           // TUI設定
	TerminalMode TerminalModeConfig         `json:"terminal_mode"` // ターミナルモード設定
	Markdown     MarkdownConfig             `json:"markdown"`      // Markdown設定
	Features     *Features                  `json:"features"`      // 機能設定
	Proactive    ProactiveConfig            `json:"proactive"`     // プロアクティブ設定
	Migration    GradualMigrationConfig     `json:"migration"`     // 段階的移行設定
}

// Markdown設定
type MarkdownConfig struct {
	Enabled         bool `json:"enabled"`          // Markdown有効/無効
	SyntaxHighlight bool `json:"syntax_highlight"` // シンタックスハイライト
}

// 機能設定
type Features struct {
	VibeMode      bool `json:"vibe_mode"`      // バイブコーディングモード
	ProactiveMode bool `json:"proactive_mode"` // プロアクティブモード
}

// 段階的移行設定
type GradualMigrationConfig struct {
	// システム選択
	UseUnifiedStreaming bool `json:"use_unified_streaming"` // 統合ストリーミングを使用
	UseUnifiedSession   bool `json:"use_unified_session"`   // 統合セッション管理を使用
	UseUnifiedTools     bool `json:"use_unified_tools"`     // 統合ツールシステムを使用
	UseUnifiedAnalysis  bool `json:"use_unified_analysis"`  // 統合分析システムを使用

	// 移行モード
	MigrationMode string `json:"migration_mode"` // gradual, compatibility, unified

	// 検証・監視設定
	EnableValidation  bool `json:"enable_validation"`  // 新旧システム比較検証を有効化
	ValidationTimeout int  `json:"validation_timeout"` // 検証タイムアウト（秒）
	EnableFallback    bool `json:"enable_fallback"`    // エラー時にレガシーシステムにフォールバック

	// モニタリング設定
	EnableMetrics    bool `json:"enable_metrics"`     // メトリクス収集を有効化
	MetricsInterval  int  `json:"metrics_interval"`   // メトリクス収集間隔（秒）
	LogMigrationInfo bool `json:"log_migration_info"` // 移行情報をログに記録
}

// プロアクティブ設定
type ProactiveConfig struct {
	Enabled            bool           `json:"enabled"`             // プロアクティブ機能有効/無効
	Level              ProactiveLevel `json:"level"`               // プロアクティブレベル
	AnalysisTimeout    int            `json:"analysis_timeout"`    // 分析タイムアウト（秒）
	BackgroundAnalysis bool           `json:"background_analysis"` // バックグラウンド分析
	ContextCompression bool           `json:"context_compression"` // コンテキスト圧縮
	SmartSuggestions   bool           `json:"smart_suggestions"`   // スマート提案
	ProjectMonitoring  bool           `json:"project_monitoring"`  // プロジェクト監視
}

// プロアクティブレベル
type ProactiveLevel int

const (
	ProactiveLevelOff      ProactiveLevel = iota // 無効
	ProactiveLevelMinimal                        // 最小限（8-12秒）
	ProactiveLevelBasic                          // 基本（15-25秒）
	ProactiveLevelStandard                       // 標準（30-45秒）
	ProactiveLevelAdvanced                       // 高度（60-90秒）
	ProactiveLevelFull                           // 完全（120秒+）
)

// プロアクティブレベルの文字列表現
func (level ProactiveLevel) String() string {
	switch level {
	case ProactiveLevelOff:
		return "off"
	case ProactiveLevelMinimal:
		return "minimal"
	case ProactiveLevelBasic:
		return "basic"
	case ProactiveLevelStandard:
		return "standard"
	case ProactiveLevelAdvanced:
		return "advanced"
	case ProactiveLevelFull:
		return "full"
	default:
		return "off"
	}
}

// 文字列からプロアクティブレベルを解析
func ParseProactiveLevel(s string) ProactiveLevel {
	switch strings.ToLower(s) {
	case "off", "disabled":
		return ProactiveLevelOff
	case "minimal", "min":
		return ProactiveLevelMinimal
	case "basic":
		return ProactiveLevelBasic
	case "standard", "std":
		return ProactiveLevelStandard
	case "advanced", "adv":
		return ProactiveLevelAdvanced
	case "full", "max":
		return ProactiveLevelFull
	default:
		return ProactiveLevelMinimal
	}
}

// デフォルト設定を返すコンストラクタ関数
func DefaultConfig() *Config {
	return &Config{
		// LLM設定
		Provider:    "ollama",
		Model:       "qwen2.5-coder:14b",
		ModelName:   "qwen2.5-coder:14b",
		BaseURL:     "http://localhost:11434",
		Timeout:     120, // 2分に延長
		Temperature: 0.7,
		MaxTokens:   4096,
		Stream:      true,

		// システム設定
		MaxFileSize:    10 * 1024 * 1024, // 10MB
		FileMaxSizeMB:  10,
		WorkspaceMode:  "project_only",
		WorkspacePath:  ".",
		CommandTimeout: 60, // 1分に延長
		MaxHistory:     100,

		// サブ設定
		MCPServers: make(map[string]MCPServerConfig),
		Log: LogConfig{
			Level:         "info",
			Format:        "console",
			Output:        []string{"stdout"},
			ShowCaller:    false,
			ShowTimestamp: true,
			ColorEnabled:  true,
			FileRotation:  false,
			MaxFileSize:   10 * 1024 * 1024, // 10MB
			Context:       make(map[string]string),
		},
		Logging: LogConfig{
			Level:         "info",
			Format:        "console",
			Output:        []string{"stdout"},
			ShowCaller:    false,
			ShowTimestamp: true,
			ColorEnabled:  true,
			FileRotation:  false,
			MaxFileSize:   10 * 1024 * 1024, // 10MB
			Context:       make(map[string]string),
		},
		TUI: TUIConfig{
			Enabled:      true,
			Theme:        "vyb",
			ShowSpinner:  true,
			ShowProgress: true,
			Animation:    true,
		},
		TerminalMode: TerminalModeConfig{
			TypingSpeed:     15, // 15ms per character
			ShowGitInPrompt: true,
			ShowProjectInfo: true,
			HistorySize:     100,
			EnableSlashCmd:  true,
			AutoSaveSession: false,
		},
		Markdown: MarkdownConfig{
			Enabled:         true,
			SyntaxHighlight: true,
		},
		Features: &Features{
			VibeMode:      true, // バイブコーディングモード有効
			ProactiveMode: true, // Phase 2: プロアクティブモード有効化
		},
		Proactive: ProactiveConfig{
			Enabled:            true,                   // Phase 2: 軽量プロアクティブ機能有効化
			Level:              ProactiveLevelStandard, // Phase 4: 実行エンジン有効化のためStandardレベル
			AnalysisTimeout:    60,                     // 1分タイムアウト（十分な時間を確保）
			BackgroundAnalysis: true,
			ContextCompression: true,
			SmartSuggestions:   true,
			ProjectMonitoring:  false, // 重い処理は引き続き無効
		},
		Migration: GradualMigrationConfig{
			// デフォルトは段階的移行モード（安全な設定）
			UseUnifiedStreaming: false, // 段階的に有効化
			UseUnifiedSession:   false, // 段階的に有効化
			UseUnifiedTools:     false, // 段階的に有効化
			UseUnifiedAnalysis:  false, // 段階的に有効化

			MigrationMode: "gradual", // gradual, compatibility, unified

			// 検証・監視はデフォルトで有効
			EnableValidation:  true,
			ValidationTimeout: 30, // 30秒
			EnableFallback:    true,

			// モニタリングは有効
			EnableMetrics:    true,
			MetricsInterval:  60, // 1分間隔
			LogMigrationInfo: true,
		},
	}
}

// 設定ファイルのパスを取得する（~/.vyb/config.json）
func GetConfigPath() (string, error) {
	// ユーザーのホームディレクトリを取得
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// ~/.vybディレクトリのパスを作成
	configDir := filepath.Join(homeDir, ".vyb")
	// ディレクトリが存在しなければ作成（権限755）
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	// 設定ファイルのフルパスを返す
	return filepath.Join(configDir, "config.json"), nil
}

// 設定ファイルを読み込んで設定を返す
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// 設定ファイルが存在するかチェック
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 設定ファイルが存在しない場合はデフォルト設定を返す
		return DefaultConfig(), nil
	}

	// 設定ファイルを読み込み
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// JSONをConfig構造体に変換
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 後方互換性のためのフィールド初期化
	if config.Features == nil {
		config.Features = &Features{
			VibeMode:      true, // バイブコーディングモード有効
			ProactiveMode: true, // Phase 2: プロアクティブモード有効化
		}
	}

	// プロアクティブ設定の初期化
	if config.Proactive.Level == 0 && !config.Proactive.Enabled {
		config.Proactive = ProactiveConfig{
			Enabled:            true, // Phase 2: 有効化
			Level:              ProactiveLevelMinimal,
			AnalysisTimeout:    8, // 短縮
			BackgroundAnalysis: true,
			ContextCompression: true,
			SmartSuggestions:   true,
			ProjectMonitoring:  false,
		}
	}

	return &config, nil
}

// Save は設定をファイルに保存
func Save(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// 設定をJSONにマーシャル
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// ファイルに書き込み
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// 現在の設定をファイルに保存する
func (c *Config) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Config構造体を整形されたJSONに変換
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// JSONデータをファイルに書き込み（権限644）
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// モデルを設定して保存する
func (c *Config) SetModel(model string) error {
	c.Model = model // モデル名を更新
	return c.Save() // ファイルに保存
}

// プロバイダーを設定して保存する
func (c *Config) SetProvider(provider string) error {
	c.Provider = provider // プロバイダー名を更新
	return c.Save()       // ファイルに保存
}

// MCPサーバーを追加する
func (c *Config) AddMCPServer(name string, server MCPServerConfig) error {
	if c.MCPServers == nil {
		c.MCPServers = make(map[string]MCPServerConfig)
	}
	c.MCPServers[name] = server
	return c.Save()
}

// MCPサーバーを削除する
func (c *Config) RemoveMCPServer(name string) error {
	if c.MCPServers == nil {
		return fmt.Errorf("MCPサーバー '%s' が見つかりません", name)
	}
	if _, exists := c.MCPServers[name]; !exists {
		return fmt.Errorf("MCPサーバー '%s' が見つかりません", name)
	}
	delete(c.MCPServers, name)
	return c.Save()
}

// MCPサーバー一覧を取得する
func (c *Config) GetMCPServers() map[string]MCPServerConfig {
	if c.MCPServers == nil {
		return make(map[string]MCPServerConfig)
	}
	return c.MCPServers
}

// MCPサーバーを取得する
func (c *Config) GetMCPServer(name string) (MCPServerConfig, error) {
	if c.MCPServers == nil {
		return MCPServerConfig{}, fmt.Errorf("MCPサーバー '%s' が見つかりません", name)
	}
	server, exists := c.MCPServers[name]
	if !exists {
		return MCPServerConfig{}, fmt.Errorf("MCPサーバー '%s' が見つかりません", name)
	}
	return server, nil
}

// ログレベルを設定して保存する
func (c *Config) SetLogLevel(level string) error {
	c.Logging.Level = level
	return c.Save()
}

// ログフォーマットを設定して保存する
func (c *Config) SetLogFormat(format string) error {
	c.Logging.Format = format
	return c.Save()
}

// ログ出力先を設定して保存する
func (c *Config) SetLogOutput(outputs []string) error {
	c.Logging.Output = outputs
	return c.Save()
}

// TUI有効/無効を設定して保存する
func (c *Config) SetTUIEnabled(enabled bool) error {
	// TUI設定は非推奨 - Claude Code風インターフェースが標準
	return nil
}

// TUIテーマを設定して保存する
func (c *Config) SetTUITheme(theme string) error {
	// TUI設定は非推奨 - Claude Code風インターフェースが標準
	return nil
}

// TUIアニメーションを設定して保存する
func (c *Config) SetTUIAnimation(enabled bool) error {
	// TUI設定は非推奨 - Claude Code風インターフェースが標準
	return nil
}

// プロアクティブモードを設定して保存する
func (c *Config) SetProactiveEnabled(enabled bool) error {
	c.Proactive.Enabled = enabled
	c.Features.ProactiveMode = enabled
	return c.Save()
}

// プロアクティブレベルを設定して保存する
func (c *Config) SetProactiveLevel(level ProactiveLevel) error {
	c.Proactive.Level = level
	return c.Save()
}

// プロアクティブレベルを文字列で設定して保存する
func (c *Config) SetProactiveLevelString(levelStr string) error {
	level := ParseProactiveLevel(levelStr)
	return c.SetProactiveLevel(level)
}

// プロアクティブ分析タイムアウトを設定して保存する
func (c *Config) SetProactiveTimeout(timeoutSeconds int) error {
	c.Proactive.AnalysisTimeout = timeoutSeconds
	return c.Save()
}

// バックグラウンド分析を設定して保存する
func (c *Config) SetBackgroundAnalysis(enabled bool) error {
	c.Proactive.BackgroundAnalysis = enabled
	return c.Save()
}

// コンテキスト圧縮を設定して保存する
func (c *Config) SetContextCompression(enabled bool) error {
	c.Proactive.ContextCompression = enabled
	return c.Save()
}

// スマート提案を設定して保存する
func (c *Config) SetSmartSuggestions(enabled bool) error {
	c.Proactive.SmartSuggestions = enabled
	return c.Save()
}

// プロジェクト監視を設定して保存する
func (c *Config) SetProjectMonitoring(enabled bool) error {
	c.Proactive.ProjectMonitoring = enabled
	return c.Save()
}

// プロアクティブ設定を一括更新して保存する
func (c *Config) UpdateProactiveConfig(config ProactiveConfig) error {
	c.Proactive = config
	c.Features.ProactiveMode = config.Enabled
	return c.Save()
}

// 現在のプロアクティブ設定を取得する
func (c *Config) GetProactiveConfig() ProactiveConfig {
	return c.Proactive
}

// プロアクティブ機能が有効かどうか確認
func (c *Config) IsProactiveEnabled() bool {
	return c.Proactive.Enabled && c.Features.ProactiveMode
}

// プロアクティブレベルに応じた設定を取得
func (c *Config) GetProactiveLevelConfig() (timeout int, backgroundAnalysis bool, monitoring bool) {
	switch c.Proactive.Level {
	case ProactiveLevelOff:
		return 0, false, false
	case ProactiveLevelMinimal:
		return 30, true, false // 30秒に延長
	case ProactiveLevelBasic:
		return 60, true, false // 1分に延長
	case ProactiveLevelStandard:
		return 90, true, false // 1.5分に延長
	case ProactiveLevelAdvanced:
		return 120, true, true // 2分に延長
	case ProactiveLevelFull:
		return 180, true, true // 3分に延長
	default:
		return c.Proactive.AnalysisTimeout, c.Proactive.BackgroundAnalysis, c.Proactive.ProjectMonitoring
	}
}
