package container

import (
	"fmt"
	"sync"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/handlers"
	"github.com/glkt/vyb-code/internal/logger"
)

// Container は依存性注入コンテナー
type Container struct {
	mu       sync.RWMutex
	services map[string]interface{}
	config   *config.Config
	logger   logger.Logger
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

	// 統合チャットハンドラー
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
