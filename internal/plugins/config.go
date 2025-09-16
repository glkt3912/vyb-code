package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/core"
	"github.com/glkt/vyb-code/internal/logger"
)

// PluginConfigManager はプラグインの設定管理を行う
type PluginConfigManager struct {
	config        *config.Config
	logger        logger.Logger
	configDir     string
	pluginConfigs map[string]*PluginConfig
}

// PluginConfig は個別プラグインの設定
type PluginConfig struct {
	Metadata core.ComponentMetadata `json:"metadata"`
	Settings map[string]interface{} `json:"settings"`
	Advanced AdvancedPluginConfig   `json:"advanced"`
}

// AdvancedPluginConfig は高度な設定項目
type AdvancedPluginConfig struct {
	LoadOrder        int               `json:"load_order"`
	MemoryLimit      int64             `json:"memory_limit"` // バイト単位
	CPULimit         float64           `json:"cpu_limit"`    // CPU使用率上限（0-1）
	Timeout          int               `json:"timeout"`      // 秒単位
	Retry            RetryConfig       `json:"retry"`
	ResourceLimits   ResourceLimits    `json:"resource_limits"`
	Environment      map[string]string `json:"environment"`
	NetworkAccess    NetworkConfig     `json:"network_access"`
	FileSystemAccess FileSystemConfig  `json:"filesystem_access"`
}

// RetryConfig は再試行設定
type RetryConfig struct {
	Enabled     bool `json:"enabled"`
	MaxAttempts int  `json:"max_attempts"`
	Interval    int  `json:"interval"` // 秒単位
	Backoff     bool `json:"backoff"`  // 指数バックオフ
}

// ResourceLimits はリソース制限
type ResourceLimits struct {
	MaxGoroutines int   `json:"max_goroutines"`
	MaxFiles      int   `json:"max_files"`
	MaxSockets    int   `json:"max_sockets"`
	DiskQuota     int64 `json:"disk_quota"` // バイト単位
}

// NetworkConfig はネットワークアクセス設定
type NetworkConfig struct {
	Allowed      bool     `json:"allowed"`
	AllowedHosts []string `json:"allowed_hosts"`
	BlockedHosts []string `json:"blocked_hosts"`
	AllowedPorts []int    `json:"allowed_ports"`
	RequireHTTPS bool     `json:"require_https"`
}

// FileSystemConfig はファイルシステムアクセス設定
type FileSystemConfig struct {
	ReadOnlyPaths  []string `json:"readonly_paths"`
	WritablePaths  []string `json:"writable_paths"`
	ForbiddenPaths []string `json:"forbidden_paths"`
	TempDirAccess  bool     `json:"temp_dir_access"`
}

// NewPluginConfigManager は新しい設定マネージャーを作成
func NewPluginConfigManager(config *config.Config, logger logger.Logger) *PluginConfigManager {
	// Get user's home directory for config
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".vyb", "plugins")

	return &PluginConfigManager{
		config:        config,
		logger:        logger,
		configDir:     configDir,
		pluginConfigs: make(map[string]*PluginConfig),
	}
}

// Initialize は設定マネージャーを初期化
func (m *PluginConfigManager) Initialize() error {
	// 設定ディレクトリを作成
	if err := os.MkdirAll(m.configDir, 0755); err != nil {
		return fmt.Errorf("設定ディレクトリ作成エラー: %w", err)
	}

	// 既存の設定ファイルを読み込み
	if err := m.loadExistingConfigs(); err != nil {
		m.logger.Warn("既存設定読み込み警告", map[string]interface{}{
			"error": err.Error(),
		})
	}

	m.logger.Info("プラグイン設定マネージャー初期化完了", map[string]interface{}{
		"config_dir":     m.configDir,
		"loaded_configs": len(m.pluginConfigs),
	})

	return nil
}

// loadExistingConfigs は既存の設定ファイルを読み込み
func (m *PluginConfigManager) loadExistingConfigs() error {
	pattern := filepath.Join(m.configDir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("設定ファイル検索エラー: %w", err)
	}

	for _, match := range matches {
		pluginName := strings.TrimSuffix(filepath.Base(match), ".json")
		if err := m.loadPluginConfig(pluginName); err != nil {
			m.logger.Warn("プラグイン設定読み込み警告", map[string]interface{}{
				"plugin": pluginName,
				"error":  err.Error(),
			})
		}
	}

	return nil
}

// GetPluginConfig はプラグイン設定を取得
func (m *PluginConfigManager) GetPluginConfig(pluginName string) (*PluginConfig, error) {
	// メモリから取得を試行
	if config, exists := m.pluginConfigs[pluginName]; exists {
		return config, nil
	}

	// ファイルから読み込みを試行
	if err := m.loadPluginConfig(pluginName); err != nil {
		// 設定が存在しない場合はデフォルト設定を作成
		return m.createDefaultConfig(pluginName), nil
	}

	return m.pluginConfigs[pluginName], nil
}

// loadPluginConfig はファイルからプラグイン設定を読み込み
func (m *PluginConfigManager) loadPluginConfig(pluginName string) error {
	configPath := filepath.Join(m.configDir, pluginName+".json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("設定ファイル読み込みエラー: %w", err)
	}

	var pluginConfig PluginConfig
	if err := json.Unmarshal(data, &pluginConfig); err != nil {
		return fmt.Errorf("設定ファイル解析エラー: %w", err)
	}

	m.pluginConfigs[pluginName] = &pluginConfig

	m.logger.Debug("プラグイン設定読み込み完了", map[string]interface{}{
		"plugin": pluginName,
	})

	return nil
}

// createDefaultConfig はデフォルト設定を作成
func (m *PluginConfigManager) createDefaultConfig(pluginName string) *PluginConfig {
	return &PluginConfig{
		Metadata: core.ComponentMetadata{
			Name:         pluginName,
			Type:         core.TypeExtension,
			Version:      "1.0.0",
			Description:  fmt.Sprintf("プラグイン %s のデフォルト設定", pluginName),
			Dependencies: []string{},
			Optional:     true,
			Enabled:      true,
		},
		Settings: make(map[string]interface{}),
		Advanced: AdvancedPluginConfig{
			LoadOrder:   1000,
			MemoryLimit: 100 * 1024 * 1024, // 100MB
			CPULimit:    0.8,               // 80%
			Timeout:     30,                // 30秒
			Retry: RetryConfig{
				Enabled:     true,
				MaxAttempts: 3,
				Interval:    5,
				Backoff:     true,
			},
			ResourceLimits: ResourceLimits{
				MaxGoroutines: 100,
				MaxFiles:      50,
				MaxSockets:    10,
				DiskQuota:     1024 * 1024 * 1024, // 1GB
			},
			Environment: make(map[string]string),
			NetworkAccess: NetworkConfig{
				Allowed:      false,
				RequireHTTPS: true,
			},
			FileSystemAccess: FileSystemConfig{
				ReadOnlyPaths:  []string{"/etc", "/usr"},
				WritablePaths:  []string{"/tmp"},
				ForbiddenPaths: []string{"/system", "/var"},
				TempDirAccess:  true,
			},
		},
	}
}

// SavePluginConfig はプラグイン設定を保存
func (m *PluginConfigManager) SavePluginConfig(pluginName string, metadata *core.ComponentMetadata) error {
	// 既存設定を取得または作成
	pluginConfig, err := m.GetPluginConfig(pluginName)
	if err != nil {
		return fmt.Errorf("プラグイン設定取得エラー: %w", err)
	}

	// メタデータを更新
	if metadata != nil {
		pluginConfig.Metadata = *metadata
	}

	// メモリに保存
	m.pluginConfigs[pluginName] = pluginConfig

	// ファイルに保存
	return m.saveConfigToFile(pluginName, pluginConfig)
}

// saveConfigToFile は設定をファイルに保存
func (m *PluginConfigManager) saveConfigToFile(pluginName string, pluginConfig *PluginConfig) error {
	configPath := filepath.Join(m.configDir, pluginName+".json")

	data, err := json.MarshalIndent(pluginConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("設定シリアライズエラー: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("設定ファイル書き込みエラー: %w", err)
	}

	m.logger.Debug("プラグイン設定保存完了", map[string]interface{}{
		"plugin": pluginName,
		"path":   configPath,
	})

	return nil
}

// UpdatePluginSettings はプラグインの設定を更新
func (m *PluginConfigManager) UpdatePluginSettings(pluginName string, settings map[string]interface{}) error {
	pluginConfig, err := m.GetPluginConfig(pluginName)
	if err != nil {
		return fmt.Errorf("プラグイン設定取得エラー: %w", err)
	}

	// 設定を更新
	for key, value := range settings {
		pluginConfig.Settings[key] = value
	}

	// 保存
	return m.saveConfigToFile(pluginName, pluginConfig)
}

// GetPluginSetting は特定の設定値を取得
func (m *PluginConfigManager) GetPluginSetting(pluginName, key string) (interface{}, error) {
	pluginConfig, err := m.GetPluginConfig(pluginName)
	if err != nil {
		return nil, err
	}

	value, exists := pluginConfig.Settings[key]
	if !exists {
		return nil, fmt.Errorf("設定 %s が見つかりません", key)
	}

	return value, nil
}

// SetPluginSetting は特定の設定値を設定
func (m *PluginConfigManager) SetPluginSetting(pluginName, key string, value interface{}) error {
	pluginConfig, err := m.GetPluginConfig(pluginName)
	if err != nil {
		return err
	}

	pluginConfig.Settings[key] = value
	return m.saveConfigToFile(pluginName, pluginConfig)
}

// DeletePluginConfig はプラグイン設定を削除
func (m *PluginConfigManager) DeletePluginConfig(pluginName string) error {
	// メモリから削除
	delete(m.pluginConfigs, pluginName)

	// ファイルを削除
	configPath := filepath.Join(m.configDir, pluginName+".json")
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("設定ファイル削除エラー: %w", err)
	}

	m.logger.Info("プラグイン設定削除完了", map[string]interface{}{
		"plugin": pluginName,
	})

	return nil
}

// ListPluginConfigs は全プラグイン設定を一覧取得
func (m *PluginConfigManager) ListPluginConfigs() map[string]*PluginConfig {
	result := make(map[string]*PluginConfig)
	for name, config := range m.pluginConfigs {
		result[name] = config
	}
	return result
}

// ValidatePluginConfig はプラグイン設定を検証
func (m *PluginConfigManager) ValidatePluginConfig(pluginName string) error {
	pluginConfig, err := m.GetPluginConfig(pluginName)
	if err != nil {
		return err
	}

	// 基本検証
	if pluginConfig.Metadata.Name == "" {
		return fmt.Errorf("プラグイン名が空です")
	}

	// リソース制限の検証
	if pluginConfig.Advanced.MemoryLimit <= 0 {
		return fmt.Errorf("無効なメモリ制限: %d", pluginConfig.Advanced.MemoryLimit)
	}

	if pluginConfig.Advanced.CPULimit < 0 || pluginConfig.Advanced.CPULimit > 1 {
		return fmt.Errorf("無効なCPU制限: %f", pluginConfig.Advanced.CPULimit)
	}

	if pluginConfig.Advanced.Timeout <= 0 {
		return fmt.Errorf("無効なタイムアウト: %d", pluginConfig.Advanced.Timeout)
	}

	m.logger.Debug("プラグイン設定検証完了", map[string]interface{}{
		"plugin": pluginName,
	})

	return nil
}

// GetConfigStats は設定統計を取得
func (m *PluginConfigManager) GetConfigStats() ConfigStats {
	stats := ConfigStats{
		TotalConfigs:   len(m.pluginConfigs),
		EnabledConfigs: 0,
		ConfigDir:      m.configDir,
	}

	for _, config := range m.pluginConfigs {
		if config.Metadata.Enabled {
			stats.EnabledConfigs++
		}
	}

	return stats
}

// ConfigStats は設定統計情報
type ConfigStats struct {
	TotalConfigs   int    `json:"total_configs"`
	EnabledConfigs int    `json:"enabled_configs"`
	ConfigDir      string `json:"config_dir"`
}
