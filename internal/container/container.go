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

	// 設定を読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}
	c.config = cfg

	// ロガーを初期化（logger.Config型を作成）
	loggerConfig := logger.Config{
		Level:         logger.ParseLevel(cfg.Log.Level),
		Format:        cfg.Log.Format,
		Output:        cfg.Log.Output,
		ShowCaller:    cfg.Log.ShowCaller,
		ShowTimestamp: cfg.Log.ShowTimestamp,
		ColorEnabled:  cfg.Log.ColorEnabled,
	}

	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		return fmt.Errorf("ロガー初期化エラー: %w", err)
	}
	c.logger = log
	c.services["logger"] = log

	// ハンドラーを初期化
	if err := c.initializeHandlers(); err != nil {
		return fmt.Errorf("ハンドラー初期化エラー: %w", err)
	}

	c.logger.Info("コンテナー初期化完了", map[string]interface{}{
		"services_count": len(c.services),
	})

	return nil
}

// initializeHandlers はハンドラーを初期化
func (c *Container) initializeHandlers() error {
	// 設定ハンドラー
	configHandler := handlers.NewConfigHandler(c.logger)
	c.services["config_handler"] = configHandler

	// チャットハンドラー
	chatHandler := handlers.NewChatHandler(c.logger)
	c.services["chat_handler"] = chatHandler

	// ツールハンドラー
	toolsHandler := handlers.NewToolsHandler(c.logger)
	c.services["tools_handler"] = toolsHandler

	// Gitハンドラー
	gitHandler := handlers.NewGitHandler(c.logger)
	c.services["git_handler"] = gitHandler

	return nil
}

// GetService は指定されたサービスを取得
func (c *Container) GetService(name string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	service, exists := c.services[name]
	if !exists {
		return nil, fmt.Errorf("サービス '%s' が見つかりません", name)
	}

	return service, nil
}

// GetLogger はロガーを取得
func (c *Container) GetLogger() logger.Logger {
	return c.logger
}

// GetConfig は設定を取得
func (c *Container) GetConfig() *config.Config {
	return c.config
}

// GetConfigHandler は設定ハンドラーを取得
func (c *Container) GetConfigHandler() (*handlers.ConfigHandler, error) {
	service, err := c.GetService("config_handler")
	if err != nil {
		return nil, err
	}

	handler, ok := service.(*handlers.ConfigHandler)
	if !ok {
		return nil, fmt.Errorf("設定ハンドラーの型変換に失敗しました")
	}

	return handler, nil
}

// GetChatHandler はチャットハンドラーを取得
func (c *Container) GetChatHandler() (*handlers.ChatHandler, error) {
	service, err := c.GetService("chat_handler")
	if err != nil {
		return nil, err
	}

	handler, ok := service.(*handlers.ChatHandler)
	if !ok {
		return nil, fmt.Errorf("チャットハンドラーの型変換に失敗しました")
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
		return nil, fmt.Errorf("ツールハンドラーの型変換に失敗しました")
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
		return nil, fmt.Errorf("Gitハンドラーの型変換に失敗しました")
	}

	return handler, nil
}

// RegisterService はサービスを登録
func (c *Container) RegisterService(name string, service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services[name] = service

	if c.logger != nil {
		c.logger.Debug("サービス登録", map[string]interface{}{
			"name": name,
			"type": fmt.Sprintf("%T", service),
		})
	}
}

// Shutdown はコンテナーをシャットダウン
func (c *Container) Shutdown() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.logger != nil {
		c.logger.Info("コンテナーシャットダウン開始", nil)
	}

	// サービスのクリーンアップ（必要に応じて）
	for name, service := range c.services {
		// Shutdownメソッドを持つサービスがあればクリーンアップ
		if shutdownable, ok := service.(interface{ Shutdown() error }); ok {
			if err := shutdownable.Shutdown(); err != nil && c.logger != nil {
				c.logger.Warn("サービスシャットダウンエラー", map[string]interface{}{
					"service": name,
					"error":   err,
				})
			}
		}
	}

	c.services = make(map[string]interface{})

	if c.logger != nil {
		c.logger.Info("コンテナーシャットダウン完了", nil)
	}

	return nil
}

// Health はコンテナーのヘルスチェック
func (c *Container) Health() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	health := map[string]interface{}{
		"services_count": len(c.services),
		"services":       make([]string, 0, len(c.services)),
		"status":         "healthy",
	}

	for name := range c.services {
		health["services"] = append(health["services"].([]string), name)
	}

	return health
}
