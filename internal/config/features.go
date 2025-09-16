package config

import (
	"fmt"
	"strings"
	"sync"
)

// FeatureFlag は機能フラグの定義
type FeatureFlag struct {
	Name        string            `json:"name"`        // フラグ名
	Enabled     bool              `json:"enabled"`     // 有効/無効
	Description string            `json:"description"` // 説明
	Version     string            `json:"version"`     // 対象バージョン
	Conditions  map[string]string `json:"conditions"`  // 有効化条件
}

// FeatureManager は機能フラグの管理者
type FeatureManager struct {
	mu    sync.RWMutex
	flags map[string]*FeatureFlag
}

// NewFeatureManager は新しいFeatureManagerを作成
func NewFeatureManager() *FeatureManager {
	return &FeatureManager{
		flags: make(map[string]*FeatureFlag),
	}
}

// RegisterFlag は機能フラグを登録
func (fm *FeatureManager) RegisterFlag(flag *FeatureFlag) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.flags[flag.Name] = flag
}

// IsEnabled は機能フラグが有効かチェック
func (fm *FeatureManager) IsEnabled(name string) bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	flag, exists := fm.flags[name]
	if !exists {
		return false
	}

	return flag.Enabled
}

// EnableFlag は機能フラグを有効化
func (fm *FeatureManager) EnableFlag(name string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	flag, exists := fm.flags[name]
	if !exists {
		return fmt.Errorf("feature flag '%s' not found", name)
	}

	flag.Enabled = true
	return nil
}

// DisableFlag は機能フラグを無効化
func (fm *FeatureManager) DisableFlag(name string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	flag, exists := fm.flags[name]
	if !exists {
		return fmt.Errorf("feature flag '%s' not found", name)
	}

	flag.Enabled = false
	return nil
}

// ListFlags は全ての機能フラグを取得
func (fm *FeatureManager) ListFlags() map[string]*FeatureFlag {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	result := make(map[string]*FeatureFlag)
	for name, flag := range fm.flags {
		// ディープコピーを作成
		flagCopy := *flag
		result[name] = &flagCopy
	}

	return result
}

// GetFlag は指定された機能フラグを取得
func (fm *FeatureManager) GetFlag(name string) (*FeatureFlag, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	flag, exists := fm.flags[name]
	if !exists {
		return nil, fmt.Errorf("feature flag '%s' not found", name)
	}

	// ディープコピーを返す
	flagCopy := *flag
	return &flagCopy, nil
}

// 定義済み機能フラグ定数
const (
	FeatureLegacyTUI           = "legacy_tui"           // レガシーTUI (非推奨)
	FeatureClaudeCodeStyle     = "claude_code_style"    // Claude Code風UI
	FeatureStreamingUI         = "streaming_ui"         // ストリーミングUI
	FeatureAdvancedInput       = "advanced_input"       // 高度な入力システム
	FeatureProactiveMode       = "proactive_mode"       // プロアクティブモード
	FeaturePluginArchitecture  = "plugin_architecture"  // プラグインアーキテクチャ
	FeatureFactoryPattern      = "factory_pattern"      // ファクトリーパターン
	FeatureModularArchitecture = "modular_architecture" // モジュラーアーキテクチャ
)

// DefaultFeatureFlags はデフォルトの機能フラグ定義
func DefaultFeatureFlags() map[string]*FeatureFlag {
	return map[string]*FeatureFlag{
		FeatureLegacyTUI: {
			Name:        FeatureLegacyTUI,
			Enabled:     false, // レガシーTUIは無効
			Description: "レガシーTUIモード（非推奨）",
			Version:     "1.0.0",
			Conditions:  map[string]string{"deprecation": "true"},
		},
		FeatureClaudeCodeStyle: {
			Name:        FeatureClaudeCodeStyle,
			Enabled:     true, // Claude Code風UIは有効
			Description: "Claude Code風インターフェース",
			Version:     "2.0.0",
			Conditions:  map[string]string{"ui_style": "claude_code"},
		},
		FeatureStreamingUI: {
			Name:        FeatureStreamingUI,
			Enabled:     true, // ストリーミングUIは有効
			Description: "ストリーミング応答表示",
			Version:     "2.0.0",
			Conditions:  map[string]string{"response_mode": "streaming"},
		},
		FeatureAdvancedInput: {
			Name:        FeatureAdvancedInput,
			Enabled:     true, // 高度な入力システムは有効
			Description: "高度な入力システム（補完・セキュリティ）",
			Version:     "2.1.0",
			Conditions:  map[string]string{"input_mode": "advanced"},
		},
		FeatureProactiveMode: {
			Name:        FeatureProactiveMode,
			Enabled:     true, // プロアクティブモードは有効
			Description: "プロアクティブAI機能",
			Version:     "2.2.0",
			Conditions:  map[string]string{"ai_mode": "proactive"},
		},
		FeaturePluginArchitecture: {
			Name:        FeaturePluginArchitecture,
			Enabled:     false, // プラグインアーキテクチャは開発中
			Description: "プラグインアーキテクチャ（実験的）",
			Version:     "3.0.0",
			Conditions:  map[string]string{"experimental": "true"},
		},
		FeatureFactoryPattern: {
			Name:        FeatureFactoryPattern,
			Enabled:     true, // ファクトリーパターンは有効
			Description: "ファクトリーパターンによるハンドラー管理",
			Version:     "2.3.0",
			Conditions:  map[string]string{"architecture": "modern"},
		},
		FeatureModularArchitecture: {
			Name:        FeatureModularArchitecture,
			Enabled:     false, // モジュラーアーキテクチャは実験的
			Description: "Core/Extensions分離型モジュラーアーキテクチャ",
			Version:     "3.0.0",
			Conditions:  map[string]string{"experimental": "true", "architecture": "modular"},
		},
	}
}

// InitializeFeatureManager はグローバル機能フラグマネージャーを初期化
func InitializeFeatureManager() *FeatureManager {
	fm := NewFeatureManager()

	// デフォルト機能フラグを登録
	for _, flag := range DefaultFeatureFlags() {
		fm.RegisterFlag(flag)
	}

	return fm
}

// 設定に機能フラグサポートを追加
func (c *Config) GetFeatureManager() *FeatureManager {
	if c.featureManager == nil {
		c.featureManager = InitializeFeatureManager()
	}
	return c.featureManager
}

// IsFeatureEnabled は機能フラグが有効かチェック
func (c *Config) IsFeatureEnabled(flagName string) bool {
	return c.GetFeatureManager().IsEnabled(flagName)
}

// EnableFeature は機能フラグを有効化
func (c *Config) EnableFeature(flagName string) error {
	return c.GetFeatureManager().EnableFlag(flagName)
}

// DisableFeature は機能フラグを無効化
func (c *Config) DisableFeature(flagName string) error {
	return c.GetFeatureManager().DisableFlag(flagName)
}

// ListFeatures は全ての機能フラグを取得
func (c *Config) ListFeatures() map[string]*FeatureFlag {
	return c.GetFeatureManager().ListFlags()
}

// GetFeatureStatus は機能フラグの状態を文字列で取得
func (c *Config) GetFeatureStatus() string {
	flags := c.ListFeatures()
	var enabled, disabled []string

	for name, flag := range flags {
		if flag.Enabled {
			enabled = append(enabled, name)
		} else {
			disabled = append(disabled, name)
		}
	}

	result := fmt.Sprintf("Enabled (%d): %s\n", len(enabled), strings.Join(enabled, ", "))
	result += fmt.Sprintf("Disabled (%d): %s", len(disabled), strings.Join(disabled, ", "))

	return result
}
