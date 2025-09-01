package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// vybの設定情報を管理する構造体
type Config struct {
	Provider      string `json:"provider"`       // LLMプロバイダー（ollama、lmstudio等）
	Model         string `json:"model"`          // 使用するモデル名
	BaseURL       string `json:"base_url"`       // LLMサーバーのURL
	Timeout       int    `json:"timeout"`        // リクエストタイムアウト（秒）
	MaxFileSize   int64  `json:"max_file_size"`  // 読み込み可能な最大ファイルサイズ
	WorkspaceMode string `json:"workspace_mode"` // ワークスペースモード（project_only等）
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
