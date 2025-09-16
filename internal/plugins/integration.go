package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/core"
	"github.com/glkt/vyb-code/internal/logger"
)

// PluginIntegration はプラグインシステムと既存システムの統合を行う
type PluginIntegration struct {
	manager          *PluginManager
	componentReg     core.ComponentRegistry
	lifecycleManager core.LifecycleManager
	logger           logger.Logger
	config           *config.Config
	enabled          bool
}

// NewPluginIntegration は新しいプラグイン統合システムを作成
func NewPluginIntegration(
	componentReg core.ComponentRegistry,
	lifecycleManager core.LifecycleManager,
	logger logger.Logger,
	config *config.Config,
) *PluginIntegration {
	manager := NewPluginManager(componentReg, lifecycleManager, logger, config)

	return &PluginIntegration{
		manager:          manager,
		componentReg:     componentReg,
		lifecycleManager: lifecycleManager,
		logger:           logger,
		config:           config,
		enabled:          true,
	}
}

// Initialize はプラグイン統合システムを初期化
func (i *PluginIntegration) Initialize(ctx context.Context) error {
	i.logger.Info("プラグイン統合システム初期化開始", nil)

	// プラグインマネージャーを初期化
	if err := i.manager.Initialize(ctx); err != nil {
		return fmt.Errorf("プラグインマネージャー初期化エラー: %w", err)
	}

	// 組み込みプラグインを登録
	if err := i.registerBuiltinPlugins(ctx); err != nil {
		return fmt.Errorf("組み込みプラグイン登録エラー: %w", err)
	}

	// プラグイン発見パスを設定
	i.setupDiscoveryPaths()

	// プラグイン発見を実行
	if err := i.manager.registry.DiscoverPlugins(ctx); err != nil {
		i.logger.Warn("プラグイン発見警告", map[string]interface{}{
			"error": err.Error(),
		})
	}

	i.logger.Info("プラグイン統合システム初期化完了", nil)
	return nil
}

// registerBuiltinPlugins は組み込みプラグインを登録
func (i *PluginIntegration) registerBuiltinPlugins(ctx context.Context) error {
	builtinPlugins := []struct {
		name    string
		factory PluginFactory
		type_   core.ComponentType
	}{
		{"example_core", NewExamplePlugin, core.TypeCore},
		{"example_extension", func(l logger.Logger, c *config.Config) (core.CoreComponent, error) {
			return NewExampleExtensionPlugin(l, c)
		}, core.TypeExtension},
		{"example_bridge", func(l logger.Logger, c *config.Config) (core.CoreComponent, error) {
			return NewExampleBridgePlugin(l, c)
		}, core.TypeBridge},
	}

	for _, plugin := range builtinPlugins {
		if err := i.registerBuiltinPlugin(ctx, plugin.name, plugin.factory, plugin.type_); err != nil {
			i.logger.Warn("組み込みプラグイン登録警告", map[string]interface{}{
				"plugin": plugin.name,
				"error":  err.Error(),
			})
		}
	}

	return nil
}

// registerBuiltinPlugin は個別の組み込みプラグインを登録
func (i *PluginIntegration) registerBuiltinPlugin(
	ctx context.Context,
	name string,
	factory PluginFactory,
	componentType core.ComponentType,
) error {
	// プラグイン情報を作成
	pluginInfo := &PluginInfo{
		Metadata: core.ComponentMetadata{
			Name:         name,
			Type:         componentType,
			Version:      "1.0.0",
			Description:  fmt.Sprintf("組み込み%sプラグイン", name),
			Dependencies: []string{},
			Optional:     true,
			Enabled:      true,
		},
		FilePath:   "builtin",
		Status:     StatusLoaded,
		UsageCount: 0,
	}

	// コンポーネントを作成
	component, err := factory(i.logger, i.config)
	if err != nil {
		return fmt.Errorf("組み込みプラグイン作成エラー %s: %w", name, err)
	}

	pluginInfo.Component = component

	// レジストリに追加
	i.manager.registry.plugins[name] = pluginInfo

	// コンポーネントレジストリに登録
	switch componentType {
	case core.TypeCore:
		if err := i.componentReg.RegisterCore(component); err != nil {
			return fmt.Errorf("コアコンポーネント登録エラー %s: %w", name, err)
		}
	case core.TypeExtension:
		if ext, ok := component.(core.Extension); ok {
			if err := i.componentReg.RegisterExtension(ext); err != nil {
				return fmt.Errorf("拡張コンポーネント登録エラー %s: %w", name, err)
			}
		}
	case core.TypeBridge:
		if bridge, ok := component.(core.Bridge); ok {
			if err := i.componentReg.RegisterBridge(bridge); err != nil {
				return fmt.Errorf("ブリッジコンポーネント登録エラー %s: %w", name, err)
			}
		}
	}

	i.logger.Info("組み込みプラグイン登録完了", map[string]interface{}{
		"name": name,
		"type": componentType,
	})

	return nil
}

// setupDiscoveryPaths はプラグイン発見パスを設定
func (i *PluginIntegration) setupDiscoveryPaths() {
	// ユーザーディレクトリのプラグインパス
	homeDir, _ := os.UserHomeDir()
	userPluginDir := filepath.Join(homeDir, ".vyb", "plugins")
	i.manager.registry.AddDiscoveryPath(userPluginDir)

	// システムワイドなプラグインパス
	systemPaths := []string{
		"/usr/local/lib/vyb-plugins",
		"/opt/vyb/plugins",
		"./plugins",
		"./extensions",
	}

	for _, path := range systemPaths {
		i.manager.registry.AddDiscoveryPath(path)
	}

	i.logger.Info("プラグイン発見パス設定完了", map[string]interface{}{
		"user_dir":     userPluginDir,
		"system_paths": systemPaths,
	})
}

// LoadPlugin はプラグインを読み込み
func (i *PluginIntegration) LoadPlugin(ctx context.Context, name string) error {
	if !i.enabled {
		return fmt.Errorf("プラグインシステムが無効です")
	}

	return i.manager.LoadPluginSafe(ctx, name)
}

// UnloadPlugin はプラグインをアンロード
func (i *PluginIntegration) UnloadPlugin(ctx context.Context, name string) error {
	if !i.enabled {
		return fmt.Errorf("プラグインシステムが無効です")
	}

	return i.manager.UnloadPluginSafe(ctx, name)
}

// ListPlugins はプラグイン一覧を取得
func (i *PluginIntegration) ListPlugins() map[string]*PluginInfo {
	return i.manager.registry.ListPlugins()
}

// GetPluginInfo はプラグイン情報を取得
func (i *PluginIntegration) GetPluginInfo(name string) (*EnhancedPluginInfo, error) {
	return i.manager.GetPluginInfo(name)
}

// EnablePlugin はプラグインを有効化
func (i *PluginIntegration) EnablePlugin(ctx context.Context, name string) error {
	return i.manager.EnablePlugin(ctx, name)
}

// DisablePlugin はプラグインを無効化
func (i *PluginIntegration) DisablePlugin(ctx context.Context, name string) error {
	return i.manager.DisablePlugin(ctx, name)
}

// RestartPlugin はプラグインを再起動
func (i *PluginIntegration) RestartPlugin(ctx context.Context, name string) error {
	return i.manager.RestartPlugin(ctx, name)
}

// GetStats は統計情報を取得
func (i *PluginIntegration) GetStats() PluginSystemStats {
	managerStats := i.manager.GetStats()

	return PluginSystemStats{
		Enabled:       i.enabled,
		ManagerStats:  managerStats,
		BuiltinCount:  i.countBuiltinPlugins(),
		ExternalCount: i.countExternalPlugins(),
	}
}

// countBuiltinPlugins は組み込みプラグインの数を数える
func (i *PluginIntegration) countBuiltinPlugins() int {
	count := 0
	for _, plugin := range i.manager.registry.plugins {
		if plugin.FilePath == "builtin" {
			count++
		}
	}
	return count
}

// countExternalPlugins は外部プラグインの数を数える
func (i *PluginIntegration) countExternalPlugins() int {
	count := 0
	for _, plugin := range i.manager.registry.plugins {
		if plugin.FilePath != "builtin" {
			count++
		}
	}
	return count
}

// DiscoverPlugins はプラグインの発見を実行
func (i *PluginIntegration) DiscoverPlugins(ctx context.Context) error {
	return i.manager.registry.DiscoverPlugins(ctx)
}

// CreatePluginCommand はプラグイン用のCLIコマンドを作成
func (i *PluginIntegration) CreatePluginCommand() *PluginCommand {
	return &PluginCommand{
		integration: i,
	}
}

// Shutdown はプラグイン統合システムを終了
func (i *PluginIntegration) Shutdown(ctx context.Context) error {
	i.logger.Info("プラグイン統合システム終了開始", nil)

	if err := i.manager.Shutdown(ctx); err != nil {
		return fmt.Errorf("プラグインマネージャー終了エラー: %w", err)
	}

	i.enabled = false

	i.logger.Info("プラグイン統合システム終了完了", nil)
	return nil
}

// PluginSystemStats はプラグインシステムの統計情報
type PluginSystemStats struct {
	Enabled       bool         `json:"enabled"`
	ManagerStats  ManagerStats `json:"manager_stats"`
	BuiltinCount  int          `json:"builtin_count"`
	ExternalCount int          `json:"external_count"`
}

// PluginCommand はプラグイン管理用のCLIコマンド
type PluginCommand struct {
	integration *PluginIntegration
}

// ExecuteListCommand はlist コマンドを実行
func (c *PluginCommand) ExecuteListCommand(ctx context.Context) error {
	plugins := c.integration.ListPlugins()

	fmt.Println("登録済みプラグイン一覧:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-20s %-10s %-10s %-15s %s\n", "名前", "タイプ", "状態", "バージョン", "説明")
	fmt.Println(strings.Repeat("-", 80))

	for _, info := range plugins {
		typeStr := ""
		switch info.Metadata.Type {
		case core.TypeCore:
			typeStr = "Core"
		case core.TypeExtension:
			typeStr = "Extension"
		case core.TypeBridge:
			typeStr = "Bridge"
		}

		fmt.Printf("%-20s %-10s %-10s %-15s %s\n",
			info.Metadata.Name,
			typeStr,
			info.Status.String(),
			info.Metadata.Version,
			info.Metadata.Description,
		)
	}

	return nil
}

// ExecuteStatsCommand はstats コマンドを実行
func (c *PluginCommand) ExecuteStatsCommand() error {
	stats := c.integration.GetStats()

	fmt.Println("プラグインシステム統計情報:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("システム有効: %t\n", stats.Enabled)
	fmt.Printf("組み込みプラグイン: %d\n", stats.BuiltinCount)
	fmt.Printf("外部プラグイン: %d\n", stats.ExternalCount)
	fmt.Printf("総プラグイン数: %d\n", stats.ManagerStats.PluginStats.TotalPlugins)
	fmt.Printf("読み込み済み: %d\n", stats.ManagerStats.PluginStats.LoadedCount)
	fmt.Printf("アクティブ: %d\n", stats.ManagerStats.PluginStats.ActiveCount)
	fmt.Printf("エラー: %d\n", stats.ManagerStats.PluginStats.ErrorCount)

	return nil
}
