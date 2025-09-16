package plugins

import (
	"context"
	"fmt"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/core"
	"github.com/glkt/vyb-code/internal/logger"
)

// ExamplePlugin は簡単なプラグインの実装例
type ExamplePlugin struct {
	name    string
	logger  logger.Logger
	config  *config.Config
	enabled bool
}

// NewExamplePlugin は新しいサンプルプラグインを作成
func NewExamplePlugin(logger logger.Logger, config *config.Config) (core.CoreComponent, error) {
	return &ExamplePlugin{
		name:    "example_plugin",
		logger:  logger,
		config:  config,
		enabled: true,
	}, nil
}

// Name はプラグイン名を返す
func (p *ExamplePlugin) Name() string {
	return p.name
}

// Initialize はプラグインを初期化
func (p *ExamplePlugin) Initialize(ctx context.Context) error {
	p.logger.Info("サンプルプラグイン初期化開始", map[string]interface{}{
		"name": p.name,
	})

	// 初期化ロジック
	p.enabled = true

	p.logger.Info("サンプルプラグイン初期化完了", map[string]interface{}{
		"name": p.name,
	})

	return nil
}

// Shutdown はプラグインをシャットダウン
func (p *ExamplePlugin) Shutdown(ctx context.Context) error {
	p.logger.Info("サンプルプラグインシャットダウン開始", map[string]interface{}{
		"name": p.name,
	})

	p.enabled = false

	p.logger.Info("サンプルプラグインシャットダウン完了", map[string]interface{}{
		"name": p.name,
	})

	return nil
}

// Health はプラグインの健全性をチェック
func (p *ExamplePlugin) Health(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("プラグイン %s は無効状態です", p.name)
	}

	return nil
}

// ExampleExtensionPlugin は拡張機能プラグインの実装例
type ExampleExtensionPlugin struct {
	ExamplePlugin
	dependencies []string
	priority     int
}

// NewExampleExtensionPlugin は新しい拡張機能プラグインを作成
func NewExampleExtensionPlugin(logger logger.Logger, config *config.Config) (core.Extension, error) {
	base := &ExamplePlugin{
		name:    "example_extension",
		logger:  logger,
		config:  config,
		enabled: true,
	}

	return &ExampleExtensionPlugin{
		ExamplePlugin: *base,
		dependencies:  []string{"config", "logger"},
		priority:      10,
	}, nil
}

// Dependencies は依存するコアコンポーネント名を返す
func (p *ExampleExtensionPlugin) Dependencies() []string {
	return p.dependencies
}

// IsEnabled は拡張機能が有効かチェック
func (p *ExampleExtensionPlugin) IsEnabled(ctx context.Context) bool {
	return p.enabled
}

// Priority は拡張機能の優先度を返す
func (p *ExampleExtensionPlugin) Priority() int {
	return p.priority
}

// ExampleBridgePlugin はブリッジプラグインの実装例
type ExampleBridgePlugin struct {
	ExamplePlugin
	connectsTo []string
	required   bool
}

// NewExampleBridgePlugin は新しいブリッジプラグインを作成
func NewExampleBridgePlugin(logger logger.Logger, config *config.Config) (core.Bridge, error) {
	base := &ExamplePlugin{
		name:    "example_bridge",
		logger:  logger,
		config:  config,
		enabled: true,
	}

	return &ExampleBridgePlugin{
		ExamplePlugin: *base,
		connectsTo:    []string{"example_extension"},
		required:      false,
	}, nil
}

// ConnectsTo は接続する拡張機能名を返す
func (p *ExampleBridgePlugin) ConnectsTo() []string {
	return p.connectsTo
}

// IsRequired は必須のブリッジコンポーネントかどうか
func (p *ExampleBridgePlugin) IsRequired() bool {
	return p.required
}
