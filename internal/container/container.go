package container

import (
	"context"
	"fmt"
	"sync"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/core"
	"github.com/glkt/vyb-code/internal/handlers"
	"github.com/glkt/vyb-code/internal/logger"
)

// Container は依存性注入コンテナー
type Container struct {
	mu            sync.RWMutex
	services      map[string]interface{}
	config        *config.Config
	logger        logger.Logger
	factory       *handlers.HandlerFactory // ハンドラーファクトリー
	moduleManager core.ModuleManager       // モジュールマネージャー
}

// NewContainer は新しいコンテナーを作成
func NewContainer() *Container {
	return &Container{
		services: make(map[string]interface{}),
	}
}

// Initialize はコンテナーを初期化
func (c *Container) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Config を初期化
	cfg, err := config.Load()
	if err != nil {
		// デフォルト設定を作成
		cfg = config.DefaultConfig()
		err = nil
	}
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}
	c.config = cfg

	// Logger を初期化
	loggerConfig := logger.Config{
		Level:     logger.ParseLevel(cfg.Log.Level),
		Component: "container",
	}
	loggerInstance, err := logger.NewLogger(loggerConfig)
	if err != nil {
		return fmt.Errorf("ロガー初期化エラー: %w", err)
	}
	c.logger = loggerInstance
	c.services["logger"] = loggerInstance

	c.logger.Info("Container 初期化開始", map[string]interface{}{
		"log_level":  cfg.Log.Level,
		"log_format": cfg.Log.Format,
	})

	// ハンドラーファクトリーを初期化
	c.factory = handlers.NewHandlerFactory(c.logger, c.config)

	// ハンドラーファクトリーにハンドラー作成関数を登録
	c.factory.RegisterHandler("chat", func(log logger.Logger, cfg *config.Config) handlers.Handler {
		return handlers.NewChatHandler(log, cfg)
	})
	c.factory.RegisterHandler("tools", func(log logger.Logger, cfg *config.Config) handlers.Handler {
		return handlers.NewToolsHandler(log)
	})
	c.factory.RegisterHandler("config", func(log logger.Logger, cfg *config.Config) handlers.Handler {
		return handlers.NewConfigHandler(log)
	})
	c.factory.RegisterHandler("git", func(log logger.Logger, cfg *config.Config) handlers.Handler {
		return handlers.NewGitHandler(log)
	})

	// モジュールマネージャーを初期化
	if cfg.IsFeatureEnabled("modular_architecture") {
		c.initializeModuleManager(cfg)
	}

	// 従来の方法でもハンドラーを作成（後方互換性のため）
	c.logger.Info("統合チャットハンドラーを初期化", map[string]interface{}{
		"system_type": "unified",
	})
	chatHandler := handlers.NewChatHandler(c.logger, c.config)
	c.services["chat_handler"] = chatHandler

	// ツールハンドラー
	toolsHandler := handlers.NewToolsHandler(c.logger)
	c.services["tools_handler"] = toolsHandler

	// Gitハンドラー
	gitHandler := handlers.NewGitHandler(c.logger)
	c.services["git_handler"] = gitHandler

	// 設定ハンドラー
	configHandler := handlers.NewConfigHandler(c.logger)
	c.services["config_handler"] = configHandler

	c.logger.Info("Container 初期化完了", map[string]interface{}{
		"services_count": len(c.services),
	})

	return nil
}

// GetService はサービスを取得
func (c *Container) GetService(name string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	service, exists := c.services[name]
	if !exists {
		return nil, fmt.Errorf("サービス '%s' が見つかりません", name)
	}
	return service, nil
}

// GetConfig は設定を取得
func (c *Container) GetConfig() *config.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// GetLogger はロガーを取得
func (c *Container) GetLogger() logger.Logger {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.logger
}

// GetChatHandler はチャットハンドラーを取得
func (c *Container) GetChatHandler() (*handlers.ChatHandler, error) {
	service, err := c.GetService("chat_handler")
	if err != nil {
		return nil, err
	}
	handler, ok := service.(*handlers.ChatHandler)
	if !ok {
		return nil, fmt.Errorf("チャットハンドラーの型変換に失敗")
	}
	return handler, nil
}

// GetToolsHandler はツールハンドラーを取得
func (c *Container) GetToolsHandler() (*handlers.ToolsHandler, error) {
	service, err := c.GetService("tools_handler")
	if err != nil {
		return nil, err
	}
	handler, ok := service.(*handlers.ToolsHandler)
	if !ok {
		return nil, fmt.Errorf("ツールハンドラーの型変換に失敗")
	}
	return handler, nil
}

// GetGitHandler はGitハンドラーを取得
func (c *Container) GetGitHandler() (*handlers.GitHandler, error) {
	service, err := c.GetService("git_handler")
	if err != nil {
		return nil, err
	}
	handler, ok := service.(*handlers.GitHandler)
	if !ok {
		return nil, fmt.Errorf("Gitハンドラーの型変換に失敗")
	}
	return handler, nil
}

// GetConfigHandler は設定ハンドラーを取得
func (c *Container) GetConfigHandler() (*handlers.ConfigHandler, error) {
	service, err := c.GetService("config_handler")
	if err != nil {
		return nil, err
	}
	handler, ok := service.(*handlers.ConfigHandler)
	if !ok {
		return nil, fmt.Errorf("設定ハンドラーの型変換に失敗")
	}
	return handler, nil
}

// Shutdown はコンテナーをシャットダウン
func (c *Container) Shutdown() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("Container シャットダウン開始", nil)

	// サービスのクリーンアップ（必要に応じて）
	c.services = make(map[string]interface{})

	c.logger.Info("Container シャットダウン完了", nil)
	return nil
}

// GetHandlerFactory はハンドラーファクトリーを取得
func (c *Container) GetHandlerFactory() *handlers.HandlerFactory {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.factory
}

// CreateHandler は新しいファクトリーパターンでハンドラーを作成
func (c *Container) CreateHandler(name string) (handlers.Handler, error) {
	if c.factory == nil {
		return nil, fmt.Errorf("handler factory not initialized")
	}
	return c.factory.CreateHandler(name)
}

// ListAvailableHandlers は利用可能なハンドラー一覧を取得
func (c *Container) ListAvailableHandlers() []string {
	if c.factory == nil {
		return []string{}
	}
	return c.factory.ListAvailableHandlers()
}

// initializeModuleManager はモジュールマネージャーを初期化
func (c *Container) initializeModuleManager(cfg *config.Config) {
	c.logger.Info("モジュラーアーキテクチャ初期化", nil)
	
	// モジュールマネージャーを作成
	c.moduleManager = core.NewModuleManager()
	
	// コンポーネントファクトリーを作成
	factory := core.NewComponentFactory(c.logger, cfg)
	
	// コアコンポーネントを登録
	for _, component := range factory.CreateCoreComponents() {
		if err := c.moduleManager.RegisterCore(component); err != nil {
			c.logger.Error("コアコンポーネント登録エラー", map[string]interface{}{
				"component": component.Name(),
				"error":     err.Error(),
			})
		}
	}
	
	// 拡張機能を登録
	for _, extension := range factory.CreateExtensions() {
		if err := c.moduleManager.RegisterExtension(extension); err != nil {
			c.logger.Error("拡張機能登録エラー", map[string]interface{}{
				"extension": extension.Name(),
				"error":     err.Error(),
			})
		}
	}
	
	// 橋渡しコンポーネントを登録
	for _, bridge := range factory.CreateBridges() {
		if err := c.moduleManager.RegisterBridge(bridge); err != nil {
			c.logger.Error("ブリッジコンポーネント登録エラー", map[string]interface{}{
				"bridge": bridge.Name(),
				"error":   err.Error(),
			})
		}
	}
	
	// 全コンポーネントを初期化
	if err := c.moduleManager.InitializeAll(context.Background()); err != nil {
		c.logger.Error("モジュール初期化エラー", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		c.logger.Info("モジュラーアーキテクチャ初期化完了", nil)
	}
}

// GetModuleManager はモジュールマネージャーを取得
func (c *Container) GetModuleManager() core.ModuleManager {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.moduleManager
}

// ListModules は利用可能なモジュール一覧を取得
func (c *Container) ListModules() []core.ComponentMetadata {
	if c.moduleManager == nil {
		return []core.ComponentMetadata{}
	}
	return c.moduleManager.ListModules()
}
