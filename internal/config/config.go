package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// TUI設定
type TUIConfig struct {
	Enabled      bool   `json:"enabled"`       // TUI有効/無効
	Theme        string `json:"theme"`         // カラーテーマ (dark, light, auto, vyb)
	ShowSpinner  bool   `json:"show_spinner"`  // スピナー表示
	ShowProgress bool   `json:"show_progress"` // プログレスバー表示
	Animation    bool   `json:"animation"`     // アニメーション有効
}

// vybの設定情報を管理する構造体
type Config struct {
	Provider      string                     `json:"provider"`       // LLMプロバイダー（ollama、lmstudio等）
	Model         string                     `json:"model"`          // 使用するモデル名
	BaseURL       string                     `json:"base_url"`       // LLMサーバーのURL
	Timeout       int                        `json:"timeout"`        // リクエストタイムアウト（秒）
	MaxFileSize   int64                      `json:"max_file_size"`  // 読み込み可能な最大ファイルサイズ
	WorkspaceMode string                     `json:"workspace_mode"` // ワークスペースモード（project_only等）
	MCPServers    map[string]MCPServerConfig `json:"mcp_servers"`    // MCPサーバー設定
	Logging       LogConfig                  `json:"logging"`        // ログ設定
	TUI           TUIConfig                  `json:"tui"`            // TUI設定
}

// デフォルト設定を返すコンストラクタ関数
func DefaultConfig() *Config {
	return &Config{
		Provider:      "ollama",
		Model:         "qwen2.5-coder:14b",
		BaseURL:       "http://localhost:11434",
		Timeout:       30,
		MaxFileSize:   10 * 1024 * 1024, // 10MB
		WorkspaceMode: "project_only",
		MCPServers:    make(map[string]MCPServerConfig),
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

	return &config, nil
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
	c.TUI.Enabled = enabled
	// デフォルト値を設定
	if c.TUI.Theme == "" {
		c.TUI.Theme = "vyb"
	}
	if !c.TUI.ShowSpinner && !c.TUI.ShowProgress {
		c.TUI.ShowSpinner = true
		c.TUI.ShowProgress = true
		c.TUI.Animation = true
	}
	return c.Save()
}

// TUIテーマを設定して保存する
func (c *Config) SetTUITheme(theme string) error {
	c.TUI.Theme = theme
	return c.Save()
}

// TUIアニメーションを設定して保存する
func (c *Config) SetTUIAnimation(enabled bool) error {
	c.TUI.Animation = enabled
	return c.Save()
}
