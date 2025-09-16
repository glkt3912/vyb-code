package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/core"
	"github.com/glkt/vyb-code/internal/logger"
)

// PluginManager は高レベルなプラグイン管理機能を提供
type PluginManager struct {
	registry         *PluginRegistry
	scheduler        *PluginScheduler
	security         *PluginSecurity
	configManager    *PluginConfigManager
	logger           logger.Logger
	config           *config.Config
	
	// プラグイン実行コンテキスト
	contexts         map[string]context.CancelFunc
	contextsMu       sync.RWMutex
	
	// 自動管理設定
	autoDiscovery    bool
	autoLoad         bool
	discoveryInterval time.Duration
}

// PluginManagerConfig はプラグインマネージャーの設定
type PluginManagerConfig struct {
	AutoDiscovery     bool          `json:"auto_discovery"`
	AutoLoad          bool          `json:"auto_load"`
	DiscoveryInterval time.Duration `json:"discovery_interval"`
	SecurityPolicy    string        `json:"security_policy"`
	MaxConcurrent     int           `json:"max_concurrent"`
	Timeout           time.Duration `json:"timeout"`
}

// NewPluginManager は新しいプラグインマネージャーを作成
func NewPluginManager(
	componentReg core.ComponentRegistry,
	lifecycleManager core.LifecycleManager,
	logger logger.Logger,
	config *config.Config,
) *PluginManager {
	registry := NewPluginRegistry(componentReg, lifecycleManager, logger, config)
	
	return &PluginManager{
		registry:         registry,
		scheduler:        NewPluginScheduler(logger),
		security:         NewPluginSecurity(logger),
		configManager:    NewPluginConfigManager(config, logger),
		logger:           logger,
		config:           config,
		contexts:         make(map[string]context.CancelFunc),
		autoDiscovery:    true,
		autoLoad:         false,
		discoveryInterval: 30 * time.Second,
	}
}

// Initialize はプラグインマネージャーを初期化
func (m *PluginManager) Initialize(ctx context.Context) error {
	m.logger.Info("プラグインマネージャー初期化開始", nil)

	// 設定を読み込み
	if err := m.loadConfig(); err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// セキュリティポリシーを初期化
	if err := m.security.Initialize(ctx); err != nil {
		return fmt.Errorf("セキュリティ初期化エラー: %w", err)
	}

	// 初回プラグイン発見
	if m.autoDiscovery {
		if err := m.registry.DiscoverPlugins(ctx); err != nil {
			m.logger.Warn("初回プラグイン発見エラー", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// 自動読み込み
	if m.autoLoad {
		if err := m.loadEnabledPlugins(ctx); err != nil {
			m.logger.Warn("自動プラグイン読み込みエラー", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// スケジューラーを開始
	m.scheduler.Start(ctx)

	// 自動発見スケジュールを設定
	if m.autoDiscovery && m.discoveryInterval > 0 {
		m.scheduleAutoDiscovery(ctx)
	}

	m.logger.Info("プラグインマネージャー初期化完了", map[string]interface{}{
		"auto_discovery": m.autoDiscovery,
		"auto_load":      m.autoLoad,
	})

	return nil
}

// loadConfig は設定を読み込み
func (m *PluginManager) loadConfig() error {
	// デフォルト設定
	cfg := &PluginManagerConfig{
		AutoDiscovery:     true,
		AutoLoad:          false,
		DiscoveryInterval: 30 * time.Second,
		SecurityPolicy:    "moderate",
		MaxConcurrent:     10,
		Timeout:           30 * time.Second,
	}

	// 設定から値を取得（実装は設定システムに依存）
	m.autoDiscovery = cfg.AutoDiscovery
	m.autoLoad = cfg.AutoLoad
	m.discoveryInterval = cfg.DiscoveryInterval

	return nil
}

// loadEnabledPlugins は有効なプラグインを全て読み込み
func (m *PluginManager) loadEnabledPlugins(ctx context.Context) error {
	plugins := m.registry.ListPlugins()
	loaded := 0
	
	for name, info := range plugins {
		if !info.Metadata.Enabled {
			continue
		}

		if err := m.LoadPluginSafe(ctx, name); err != nil {
			m.logger.Warn("プラグイン読み込み失敗", map[string]interface{}{
				"name":  name,
				"error": err.Error(),
			})
			continue
		}
		loaded++
	}

	m.logger.Info("有効プラグイン読み込み完了", map[string]interface{}{
		"loaded": loaded,
		"total":  len(plugins),
	})

	return nil
}

// scheduleAutoDiscovery は自動発見をスケジュール
func (m *PluginManager) scheduleAutoDiscovery(ctx context.Context) {
	m.scheduler.ScheduleRepeating("auto_discovery", m.discoveryInterval, func(ctx context.Context) error {
		return m.registry.DiscoverPlugins(ctx)
	})
}

// LoadPluginSafe は安全にプラグインを読み込み
func (m *PluginManager) LoadPluginSafe(ctx context.Context, name string) error {
	// セキュリティチェック
	if err := m.security.ValidatePlugin(name); err != nil {
		return fmt.Errorf("セキュリティ検証失敗 %s: %w", name, err)
	}

	// タイムアウト付きコンテキスト作成
	loadCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 読み込み実行
	if err := m.registry.LoadPlugin(loadCtx, name); err != nil {
		return fmt.Errorf("プラグイン読み込みエラー %s: %w", name, err)
	}

	// 実行コンテキストを保存
	m.contextsMu.Lock()
	m.contexts[name] = cancel
	m.contextsMu.Unlock()

	// 読み込み後の初期化
	if err := m.initializeLoadedPlugin(loadCtx, name); err != nil {
		// 読み込み失敗時はアンロード
		m.registry.UnloadPlugin(loadCtx, name)
		return fmt.Errorf("プラグイン初期化エラー %s: %w", name, err)
	}

	m.logger.Info("プラグイン安全読み込み完了", map[string]interface{}{
		"name": name,
	})

	return nil
}

// initializeLoadedPlugin は読み込まれたプラグインを初期化
func (m *PluginManager) initializeLoadedPlugin(ctx context.Context, name string) error {
	pluginInfo, err := m.registry.GetPlugin(name)
	if err != nil {
		return err
	}

	if pluginInfo.Component == nil {
		return fmt.Errorf("プラグインコンポーネントが見つかりません %s", name)
	}

	// コンポーネントを初期化
	if err := pluginInfo.Component.Initialize(ctx); err != nil {
		return fmt.Errorf("コンポーネント初期化エラー %s: %w", name, err)
	}

	// 健全性チェック
	if err := pluginInfo.Component.Health(ctx); err != nil {
		return fmt.Errorf("プラグイン健全性チェック失敗 %s: %w", name, err)
	}

	pluginInfo.Status = StatusActive
	return nil
}

// UnloadPluginSafe は安全にプラグインをアンロード
func (m *PluginManager) UnloadPluginSafe(ctx context.Context, name string) error {
	// 実行コンテキストをキャンセル
	m.contextsMu.Lock()
	if cancel, exists := m.contexts[name]; exists {
		cancel()
		delete(m.contexts, name)
	}
	m.contextsMu.Unlock()

	// アンロード実行
	if err := m.registry.UnloadPlugin(ctx, name); err != nil {
		return fmt.Errorf("プラグインアンロードエラー %s: %w", name, err)
	}

	m.logger.Info("プラグイン安全アンロード完了", map[string]interface{}{
		"name": name,
	})

	return nil
}

// RestartPlugin はプラグインを再起動
func (m *PluginManager) RestartPlugin(ctx context.Context, name string) error {
	m.logger.Info("プラグイン再起動開始", map[string]interface{}{
		"name": name,
	})

	// アンロード
	if err := m.UnloadPluginSafe(ctx, name); err != nil {
		return fmt.Errorf("再起動用アンロードエラー %s: %w", name, err)
	}

	// 少し待機
	time.Sleep(1 * time.Second)

	// 再読み込み
	if err := m.LoadPluginSafe(ctx, name); err != nil {
		return fmt.Errorf("再起動用読み込みエラー %s: %w", name, err)
	}

	m.logger.Info("プラグイン再起動完了", map[string]interface{}{
		"name": name,
	})

	return nil
}

// EnablePlugin はプラグインを有効化
func (m *PluginManager) EnablePlugin(ctx context.Context, name string) error {
	pluginInfo, err := m.registry.GetPlugin(name)
	if err != nil {
		return err
	}

	if pluginInfo.Metadata.Enabled {
		return nil // 既に有効
	}

	pluginInfo.Metadata.Enabled = true
	
	// 設定を保存
	if err := m.configManager.SavePluginConfig(name, &pluginInfo.Metadata); err != nil {
		return fmt.Errorf("設定保存エラー %s: %w", name, err)
	}

	// 自動読み込みが有効なら読み込み
	if m.autoLoad {
		if err := m.LoadPluginSafe(ctx, name); err != nil {
			return fmt.Errorf("有効化後読み込みエラー %s: %w", name, err)
		}
	}

	m.logger.Info("プラグイン有効化", map[string]interface{}{
		"name": name,
	})

	return nil
}

// DisablePlugin はプラグインを無効化
func (m *PluginManager) DisablePlugin(ctx context.Context, name string) error {
	pluginInfo, err := m.registry.GetPlugin(name)
	if err != nil {
		return err
	}

	if !pluginInfo.Metadata.Enabled {
		return nil // 既に無効
	}

	// 読み込まれていればアンロード
	if pluginInfo.Status == StatusLoaded || pluginInfo.Status == StatusActive {
		if err := m.UnloadPluginSafe(ctx, name); err != nil {
			return fmt.Errorf("無効化用アンロードエラー %s: %w", name, err)
		}
	}

	pluginInfo.Metadata.Enabled = false
	pluginInfo.Status = StatusDisabled
	
	// 設定を保存
	if err := m.configManager.SavePluginConfig(name, &pluginInfo.Metadata); err != nil {
		return fmt.Errorf("設定保存エラー %s: %w", name, err)
	}

	m.logger.Info("プラグイン無効化", map[string]interface{}{
		"name": name,
	})

	return nil
}

// GetPluginInfo は詳細なプラグイン情報を取得
func (m *PluginManager) GetPluginInfo(name string) (*EnhancedPluginInfo, error) {
	pluginInfo, err := m.registry.GetPlugin(name)
	if err != nil {
		return nil, err
	}

	enhanced := &EnhancedPluginInfo{
		PluginInfo: *pluginInfo,
	}

	// 健全性チェック
	if pluginInfo.Component != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		enhanced.HealthStatus = "healthy"
		if err := pluginInfo.Component.Health(ctx); err != nil {
			enhanced.HealthStatus = "unhealthy"
			enhanced.HealthError = err.Error()
		}
	}

	// 依存関係の状態をチェック
	enhanced.DependencyStatus = m.checkDependencyStatus(pluginInfo.Metadata.Dependencies)

	return enhanced, nil
}

// checkDependencyStatus は依存関係の状態をチェック
func (m *PluginManager) checkDependencyStatus(dependencies []string) map[string]string {
	status := make(map[string]string)
	
	for _, dep := range dependencies {
		if info, err := m.registry.GetPlugin(dep); err != nil {
			status[dep] = "missing"
		} else {
			status[dep] = info.Status.String()
		}
	}
	
	return status
}

// ListPluginsDetailed は詳細なプラグイン一覧を取得
func (m *PluginManager) ListPluginsDetailed() (map[string]*EnhancedPluginInfo, error) {
	plugins := m.registry.ListPlugins()
	result := make(map[string]*EnhancedPluginInfo)
	
	for name := range plugins {
		info, err := m.GetPluginInfo(name)
		if err != nil {
			continue
		}
		result[name] = info
	}
	
	return result, nil
}

// Shutdown はプラグインマネージャーを終了
func (m *PluginManager) Shutdown(ctx context.Context) error {
	m.logger.Info("プラグインマネージャー終了開始", nil)

	// 全コンテキストをキャンセル
	m.contextsMu.Lock()
	for _, cancel := range m.contexts {
		cancel()
	}
	m.contexts = make(map[string]context.CancelFunc)
	m.contextsMu.Unlock()

	// スケジューラーを停止
	m.scheduler.Stop(ctx)

	// 全プラグインをアンロード
	plugins := m.registry.ListPlugins()
	for name := range plugins {
		if err := m.registry.UnloadPlugin(ctx, name); err != nil {
			m.logger.Warn("プラグインアンロードエラー", map[string]interface{}{
				"name":  name,
				"error": err.Error(),
			})
		}
	}

	m.logger.Info("プラグインマネージャー終了完了", nil)
	return nil
}

// EnhancedPluginInfo は拡張されたプラグイン情報
type EnhancedPluginInfo struct {
	PluginInfo       `json:"plugin_info"`
	HealthStatus     string            `json:"health_status"`
	HealthError      string            `json:"health_error,omitempty"`
	DependencyStatus map[string]string `json:"dependency_status"`
}

// GetStats は統計情報を取得
func (m *PluginManager) GetStats() ManagerStats {
	registryStats := m.registry.GetPluginStats()
	
	return ManagerStats{
		PluginStats:       registryStats,
		AutoDiscovery:     m.autoDiscovery,
		AutoLoad:          m.autoLoad,
		DiscoveryInterval: m.discoveryInterval,
		ActiveContexts:    len(m.contexts),
	}
}

// ManagerStats はマネージャーの統計情報
type ManagerStats struct {
	PluginStats       PluginStats   `json:"plugin_stats"`
	AutoDiscovery     bool          `json:"auto_discovery"`
	AutoLoad          bool          `json:"auto_load"`
	DiscoveryInterval time.Duration `json:"discovery_interval"`
	ActiveContexts    int           `json:"active_contexts"`
}