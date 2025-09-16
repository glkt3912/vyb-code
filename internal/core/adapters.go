package core

import (
	"context"
	"fmt"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
)

// CoreComponentAdapter は既存コンポーネントをCoreComponentに適応させる
type CoreComponentAdapter struct {
	name        string
	initialized bool
	logger      logger.Logger
	config      *config.Config

	// 実際の初期化・終了処理を行う関数
	initFunc     func(ctx context.Context) error
	shutdownFunc func(ctx context.Context) error
	healthFunc   func(ctx context.Context) error
}

// NewCoreComponentAdapter は新しいアダプターを作成
func NewCoreComponentAdapter(
	name string,
	logger logger.Logger,
	config *config.Config,
	initFunc func(ctx context.Context) error,
	shutdownFunc func(ctx context.Context) error,
	healthFunc func(ctx context.Context) error,
) CoreComponent {
	return &CoreComponentAdapter{
		name:         name,
		logger:       logger,
		config:       config,
		initFunc:     initFunc,
		shutdownFunc: shutdownFunc,
		healthFunc:   healthFunc,
	}
}

// Name はコンポーネント名を返す
func (a *CoreComponentAdapter) Name() string {
	return a.name
}

// Initialize はコンポーネントを初期化
func (a *CoreComponentAdapter) Initialize(ctx context.Context) error {
	if a.initialized {
		return nil
	}

	if a.initFunc != nil {
		if err := a.initFunc(ctx); err != nil {
			return fmt.Errorf("initialization failed for '%s': %w", a.name, err)
		}
	}

	a.initialized = true
	if a.logger != nil {
		a.logger.Info(fmt.Sprintf("Core component '%s' initialized", a.name), nil)
	}

	return nil
}

// Shutdown はコンポーネントをシャットダウン
func (a *CoreComponentAdapter) Shutdown(ctx context.Context) error {
	if !a.initialized {
		return nil
	}

	if a.shutdownFunc != nil {
		if err := a.shutdownFunc(ctx); err != nil {
			return fmt.Errorf("shutdown failed for '%s': %w", a.name, err)
		}
	}

	a.initialized = false
	if a.logger != nil {
		a.logger.Info(fmt.Sprintf("Core component '%s' shutdown", a.name), nil)
	}

	return nil
}

// Health はコンポーネントの健全性をチェック
func (a *CoreComponentAdapter) Health(ctx context.Context) error {
	if !a.initialized {
		return fmt.Errorf("component '%s' not initialized", a.name)
	}

	if a.healthFunc != nil {
		return a.healthFunc(ctx)
	}

	return nil
}

// ExtensionAdapter は既存コンポーネントをExtensionに適応させる
type ExtensionAdapter struct {
	*CoreComponentAdapter
	dependencies []string
	priority     int
	enabledFunc  func(ctx context.Context) bool
}

// NewExtensionAdapter は新しい拡張アダプターを作成
func NewExtensionAdapter(
	name string,
	logger logger.Logger,
	config *config.Config,
	dependencies []string,
	priority int,
	enabledFunc func(ctx context.Context) bool,
	initFunc func(ctx context.Context) error,
	shutdownFunc func(ctx context.Context) error,
	healthFunc func(ctx context.Context) error,
) Extension {
	return &ExtensionAdapter{
		CoreComponentAdapter: &CoreComponentAdapter{
			name:         name,
			logger:       logger,
			config:       config,
			initFunc:     initFunc,
			shutdownFunc: shutdownFunc,
			healthFunc:   healthFunc,
		},
		dependencies: dependencies,
		priority:     priority,
		enabledFunc:  enabledFunc,
	}
}

// Dependencies は依存するコアコンポーネント名を返す
func (e *ExtensionAdapter) Dependencies() []string {
	return e.dependencies
}

// IsEnabled は拡張機能が有効かチェック
func (e *ExtensionAdapter) IsEnabled(ctx context.Context) bool {
	if e.enabledFunc != nil {
		return e.enabledFunc(ctx)
	}

	// デフォルトでは有効
	return true
}

// Priority は拡張機能の優先度を返す
func (e *ExtensionAdapter) Priority() int {
	return e.priority
}

// BridgeAdapter は既存コンポーネントをBridgeに適応させる
type BridgeAdapter struct {
	*CoreComponentAdapter
	connectsTo []string
	required   bool
}

// NewBridgeAdapter は新しいブリッジアダプターを作成
func NewBridgeAdapter(
	name string,
	logger logger.Logger,
	config *config.Config,
	connectsTo []string,
	required bool,
	initFunc func(ctx context.Context) error,
	shutdownFunc func(ctx context.Context) error,
	healthFunc func(ctx context.Context) error,
) Bridge {
	return &BridgeAdapter{
		CoreComponentAdapter: &CoreComponentAdapter{
			name:         name,
			logger:       logger,
			config:       config,
			initFunc:     initFunc,
			shutdownFunc: shutdownFunc,
			healthFunc:   healthFunc,
		},
		connectsTo: connectsTo,
		required:   required,
	}
}

// ConnectsTo は接続する拡張機能名を返す
func (b *BridgeAdapter) ConnectsTo() []string {
	return b.connectsTo
}

// IsRequired は必須の橋渡しコンポーネントかどうか
func (b *BridgeAdapter) IsRequired() bool {
	return b.required
}

// ComponentFactory はコンポーネント作成のファクトリー
type ComponentFactory struct {
	logger logger.Logger
	config *config.Config
}

// NewComponentFactory は新しいコンポーネントファクトリーを作成
func NewComponentFactory(logger logger.Logger, config *config.Config) *ComponentFactory {
	return &ComponentFactory{
		logger: logger,
		config: config,
	}
}

// CreateCoreComponents は既存のパッケージからコアコンポーネントを作成
func (cf *ComponentFactory) CreateCoreComponents() []CoreComponent {
	var components []CoreComponent

	// Logger コンポーネント
	components = append(components, NewCoreComponentAdapter(
		"logger",
		cf.logger,
		cf.config,
		func(ctx context.Context) error {
			// ロガーは既に初期化済みなので何もしない
			return nil
		},
		func(ctx context.Context) error {
			// ロガーのクリーンアップ（必要に応じて）
			return nil
		},
		func(ctx context.Context) error {
			if cf.logger == nil {
				return fmt.Errorf("logger is nil")
			}
			return nil
		},
	))

	// Config コンポーネント
	components = append(components, NewCoreComponentAdapter(
		"config",
		cf.logger,
		cf.config,
		func(ctx context.Context) error {
			// 設定は既に読み込み済みなので何もしない
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			if cf.config == nil {
				return fmt.Errorf("config is nil")
			}
			return nil
		},
	))

	// Security コンポーネント
	components = append(components, NewCoreComponentAdapter(
		"security",
		cf.logger,
		cf.config,
		func(ctx context.Context) error {
			// セキュリティ制約の初期化
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			// セキュリティコンポーネントの健全性チェック
			return nil
		},
	))

	// Version コンポーネント
	components = append(components, NewCoreComponentAdapter(
		"version",
		cf.logger,
		cf.config,
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
	))

	return components
}

// CreateExtensions は既存のパッケージから拡張機能を作成
func (cf *ComponentFactory) CreateExtensions() []Extension {
	var extensions []Extension

	// AI Extension
	extensions = append(extensions, NewExtensionAdapter(
		"ai",
		cf.logger,
		cf.config,
		[]string{"config", "logger"}, // 依存関係
		10,                           // 優先度
		func(ctx context.Context) bool {
			// Feature flagで制御
			return cf.config.IsFeatureEnabled("ai_features")
		},
		func(ctx context.Context) error {
			// AI機能の初期化
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
	))

	// Analysis Extension
	extensions = append(extensions, NewExtensionAdapter(
		"analysis",
		cf.logger,
		cf.config,
		[]string{"config", "logger"},
		20,
		func(ctx context.Context) bool {
			return cf.config.IsFeatureEnabled("analysis_features")
		},
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
	))

	// Performance Extension
	extensions = append(extensions, NewExtensionAdapter(
		"performance",
		cf.logger,
		cf.config,
		[]string{"config", "logger"},
		30,
		func(ctx context.Context) bool {
			return cf.config.IsFeatureEnabled("performance_monitoring")
		},
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
	))

	return extensions
}

// CreateBridges は既存のパッケージから橋渡しコンポーネントを作成
func (cf *ComponentFactory) CreateBridges() []Bridge {
	var bridges []Bridge

	// LLM Bridge
	bridges = append(bridges, NewBridgeAdapter(
		"llm",
		cf.logger,
		cf.config,
		[]string{"ai", "analysis"}, // 接続先
		true,                       // 必須
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
	))

	// Tools Bridge
	bridges = append(bridges, NewBridgeAdapter(
		"tools",
		cf.logger,
		cf.config,
		[]string{"analysis"},
		true,
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
	))

	return bridges
}
