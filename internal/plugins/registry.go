package plugins

import (
	"context"
	"fmt"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/core"
	"github.com/glkt/vyb-code/internal/logger"
)

// PluginRegistry はプラグインの動的登録・管理を行う
type PluginRegistry struct {
	mu               sync.RWMutex
	plugins          map[string]*PluginInfo
	loadedModules    map[string]*plugin.Plugin
	componentReg     core.ComponentRegistry
	lifecycleManager core.LifecycleManager
	logger           logger.Logger
	config           *config.Config
	discoveryPaths   []string
}

// PluginInfo はプラグインの情報を保持
type PluginInfo struct {
	Metadata   core.ComponentMetadata `json:"metadata"`
	FilePath   string                 `json:"file_path"`
	LoadTime   time.Time              `json:"load_time"`
	LastUsed   time.Time              `json:"last_used"`
	UsageCount int                    `json:"usage_count"`
	Status     PluginStatus           `json:"status"`
	Component  core.CoreComponent     `json:"-"`
	Module     *plugin.Plugin         `json:"-"`
}

// PluginStatus はプラグインの状態
type PluginStatus int

const (
	StatusUnloaded PluginStatus = iota
	StatusLoading
	StatusLoaded
	StatusActive
	StatusError
	StatusDisabled
)

// String はPluginStatusの文字列表現を返す
func (s PluginStatus) String() string {
	switch s {
	case StatusUnloaded:
		return "unloaded"
	case StatusLoading:
		return "loading"
	case StatusLoaded:
		return "loaded"
	case StatusActive:
		return "active"
	case StatusError:
		return "error"
	case StatusDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

// PluginManifest はプラグインのマニフェストファイル構造
type PluginManifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	License      string            `json:"license"`
	EntryPoint   string            `json:"entry_point"`
	Type         string            `json:"type"`
	Dependencies []string          `json:"dependencies"`
	Config       map[string]string `json:"config"`
	Permissions  []string          `json:"permissions"`
	MinVersion   string            `json:"min_version"`
	MaxVersion   string            `json:"max_version"`
}

// PluginFactory はプラグインの作成関数の型
type PluginFactory func(logger.Logger, *config.Config) (core.CoreComponent, error)

// NewPluginRegistry は新しいプラグインレジストリを作成
func NewPluginRegistry(
	componentReg core.ComponentRegistry,
	lifecycleManager core.LifecycleManager,
	logger logger.Logger,
	config *config.Config,
) *PluginRegistry {
	return &PluginRegistry{
		plugins:          make(map[string]*PluginInfo),
		loadedModules:    make(map[string]*plugin.Plugin),
		componentReg:     componentReg,
		lifecycleManager: lifecycleManager,
		logger:           logger,
		config:           config,
		discoveryPaths:   []string{"./plugins", "./extensions", "/usr/local/lib/vyb-plugins"},
	}
}

// DiscoverPlugins は指定されたパスからプラグインを発見
func (r *PluginRegistry) DiscoverPlugins(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("プラグイン発見を開始", map[string]interface{}{
		"discovery_paths": r.discoveryPaths,
	})

	discovered := 0
	for _, path := range r.discoveryPaths {
		count, err := r.discoverInPath(ctx, path)
		if err != nil {
			r.logger.Warn("プラグイン発見エラー", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
			continue
		}
		discovered += count
	}

	r.logger.Info("プラグイン発見完了", map[string]interface{}{
		"discovered_count": discovered,
		"total_plugins":    len(r.plugins),
	})

	return nil
}

// discoverInPath は指定されたパス内のプラグインを発見
func (r *PluginRegistry) discoverInPath(ctx context.Context, path string) (int, error) {
	// プラグインファイルのパターン（.so, .dll, .dylib）
	patterns := []string{"*.so", "*.dll", "*.dylib"}
	discovered := 0

	for _, pattern := range patterns {
		fullPattern := filepath.Join(path, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			if r.shouldSkipFile(match) {
				continue
			}

			manifest, err := r.loadManifest(match)
			if err != nil {
				r.logger.Warn("マニフェスト読み込みエラー", map[string]interface{}{
					"file":  match,
					"error": err.Error(),
				})
				continue
			}

			if err := r.registerPlugin(manifest, match); err != nil {
				r.logger.Warn("プラグイン登録エラー", map[string]interface{}{
					"plugin": manifest.Name,
					"file":   match,
					"error":  err.Error(),
				})
				continue
			}

			discovered++
		}
	}

	return discovered, nil
}

// shouldSkipFile はファイルをスキップすべきかチェック
func (r *PluginRegistry) shouldSkipFile(filename string) bool {
	base := filepath.Base(filename)

	// 隠しファイルやバックアップファイルをスキップ
	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, "~") {
		return true
	}

	// テストファイルをスキップ
	if strings.Contains(base, "_test") || strings.Contains(base, "test_") {
		return true
	}

	return false
}

// loadManifest はプラグインのマニフェストを読み込み
func (r *PluginRegistry) loadManifest(pluginPath string) (*PluginManifest, error) {
	// マニフェストファイルのパスを生成
	dir := filepath.Dir(pluginPath)
	baseName := strings.TrimSuffix(filepath.Base(pluginPath), filepath.Ext(pluginPath))
	manifestPath := filepath.Join(dir, baseName+".json")

	// デフォルトマニフェストを作成（マニフェストファイルが存在しない場合）
	manifest := &PluginManifest{
		Name:        baseName,
		Version:     "1.0.0",
		Description: fmt.Sprintf("プラグイン %s", baseName),
		EntryPoint:  "NewPlugin",
		Type:        "extension",
	}

	r.logger.Debug("マニフェスト情報", map[string]interface{}{
		"plugin":        baseName,
		"manifest_path": manifestPath,
		"default":       true,
	})

	return manifest, nil
}

// registerPlugin はプラグインを登録
func (r *PluginRegistry) registerPlugin(manifest *PluginManifest, filePath string) error {
	if _, exists := r.plugins[manifest.Name]; exists {
		return fmt.Errorf("プラグイン %s は既に登録済み", manifest.Name)
	}

	pluginInfo := &PluginInfo{
		Metadata: core.ComponentMetadata{
			Name:         manifest.Name,
			Type:         r.parseComponentType(manifest.Type),
			Version:      manifest.Version,
			Description:  manifest.Description,
			Dependencies: manifest.Dependencies,
			Optional:     true,
			Enabled:      true,
		},
		FilePath:   filePath,
		LoadTime:   time.Now(),
		Status:     StatusUnloaded,
		UsageCount: 0,
	}

	r.plugins[manifest.Name] = pluginInfo

	r.logger.Info("プラグイン登録完了", map[string]interface{}{
		"name":    manifest.Name,
		"version": manifest.Version,
		"path":    filePath,
	})

	return nil
}

// parseComponentType は文字列からComponentTypeに変換
func (r *PluginRegistry) parseComponentType(typeStr string) core.ComponentType {
	switch strings.ToLower(typeStr) {
	case "core":
		return core.TypeCore
	case "extension":
		return core.TypeExtension
	case "bridge":
		return core.TypeBridge
	default:
		return core.TypeExtension
	}
}

// LoadPlugin は指定されたプラグインを動的に読み込み
func (r *PluginRegistry) LoadPlugin(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pluginInfo, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("プラグイン %s が見つかりません", name)
	}

	if pluginInfo.Status == StatusLoaded || pluginInfo.Status == StatusActive {
		r.logger.Debug("プラグイン既に読み込み済み", map[string]interface{}{
			"name":   name,
			"status": pluginInfo.Status.String(),
		})
		return nil
	}

	pluginInfo.Status = StatusLoading

	r.logger.Info("プラグイン読み込み開始", map[string]interface{}{
		"name": name,
		"path": pluginInfo.FilePath,
	})

	// プラグインファイルを読み込み
	p, err := plugin.Open(pluginInfo.FilePath)
	if err != nil {
		pluginInfo.Status = StatusError
		return fmt.Errorf("プラグイン読み込みエラー %s: %w", name, err)
	}

	// エントリーポイントを取得
	factorySymbol, err := p.Lookup("NewPlugin")
	if err != nil {
		pluginInfo.Status = StatusError
		return fmt.Errorf("エントリーポイント 'NewPlugin' が見つかりません %s: %w", name, err)
	}

	// ファクトリー関数にキャスト
	factory, ok := factorySymbol.(func(logger.Logger, *config.Config) (core.CoreComponent, error))
	if !ok {
		pluginInfo.Status = StatusError
		return fmt.Errorf("無効なエントリーポイントシグネチャ %s", name)
	}

	// コンポーネントを作成
	component, err := factory(r.logger, r.config)
	if err != nil {
		pluginInfo.Status = StatusError
		return fmt.Errorf("コンポーネント作成エラー %s: %w", name, err)
	}

	// プラグイン情報を更新
	pluginInfo.Module = p
	pluginInfo.Component = component
	pluginInfo.Status = StatusLoaded
	pluginInfo.LoadTime = time.Now()
	r.loadedModules[name] = p

	// コンポーネントレジストリに登録
	if err := r.registerToComponentRegistry(pluginInfo); err != nil {
		pluginInfo.Status = StatusError
		return fmt.Errorf("コンポーネントレジストリ登録エラー %s: %w", name, err)
	}

	r.logger.Info("プラグイン読み込み完了", map[string]interface{}{
		"name":      name,
		"load_time": time.Since(pluginInfo.LoadTime),
	})

	return nil
}

// registerToComponentRegistry はコンポーネントレジストリに登録
func (r *PluginRegistry) registerToComponentRegistry(pluginInfo *PluginInfo) error {
	switch pluginInfo.Metadata.Type {
	case core.TypeCore:
		return r.componentReg.RegisterCore(pluginInfo.Component)
	case core.TypeExtension:
		if ext, ok := pluginInfo.Component.(core.Extension); ok {
			return r.componentReg.RegisterExtension(ext)
		}
		return fmt.Errorf("拡張機能インターフェース未実装 %s", pluginInfo.Metadata.Name)
	case core.TypeBridge:
		if bridge, ok := pluginInfo.Component.(core.Bridge); ok {
			return r.componentReg.RegisterBridge(bridge)
		}
		return fmt.Errorf("ブリッジインターフェース未実装 %s", pluginInfo.Metadata.Name)
	default:
		return fmt.Errorf("未知のコンポーネント型 %s", pluginInfo.Metadata.Name)
	}
}

// UnloadPlugin は指定されたプラグインをアンロード
func (r *PluginRegistry) UnloadPlugin(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pluginInfo, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("プラグイン %s が見つかりません", name)
	}

	if pluginInfo.Status == StatusUnloaded {
		return nil
	}

	r.logger.Info("プラグインアンロード開始", map[string]interface{}{
		"name": name,
	})

	// コンポーネントをシャットダウン
	if pluginInfo.Component != nil {
		if err := pluginInfo.Component.Shutdown(ctx); err != nil {
			r.logger.Warn("プラグインシャットダウンエラー", map[string]interface{}{
				"name":  name,
				"error": err.Error(),
			})
		}
	}

	// メモリから削除
	pluginInfo.Status = StatusUnloaded
	pluginInfo.Component = nil
	delete(r.loadedModules, name)

	r.logger.Info("プラグインアンロード完了", map[string]interface{}{
		"name": name,
	})

	return nil
}

// GetPlugin は指定されたプラグイン情報を取得
func (r *PluginRegistry) GetPlugin(name string) (*PluginInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pluginInfo, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("プラグイン %s が見つかりません", name)
	}

	// 使用統計を更新
	pluginInfo.LastUsed = time.Now()
	pluginInfo.UsageCount++

	return pluginInfo, nil
}

// ListPlugins は登録されているプラグイン一覧を取得
func (r *PluginRegistry) ListPlugins() map[string]*PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*PluginInfo)
	for name, info := range r.plugins {
		result[name] = info
	}
	return result
}

// GetPluginStats はプラグインの統計情報を取得
func (r *PluginRegistry) GetPluginStats() PluginStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := PluginStats{
		TotalPlugins: len(r.plugins),
		LoadedCount:  0,
		ActiveCount:  0,
		ErrorCount:   0,
	}

	for _, info := range r.plugins {
		switch info.Status {
		case StatusLoaded:
			stats.LoadedCount++
		case StatusActive:
			stats.ActiveCount++
		case StatusError:
			stats.ErrorCount++
		}
	}

	return stats
}

// PluginStats はプラグインの統計情報
type PluginStats struct {
	TotalPlugins int `json:"total_plugins"`
	LoadedCount  int `json:"loaded_count"`
	ActiveCount  int `json:"active_count"`
	ErrorCount   int `json:"error_count"`
}

// AddDiscoveryPath は発見パスを追加
func (r *PluginRegistry) AddDiscoveryPath(path string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.discoveryPaths {
		if existing == path {
			return
		}
	}

	r.discoveryPaths = append(r.discoveryPaths, path)
	r.logger.Info("発見パス追加", map[string]interface{}{
		"path": path,
	})
}

// RemoveDiscoveryPath は発見パスを削除
func (r *PluginRegistry) RemoveDiscoveryPath(path string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, existing := range r.discoveryPaths {
		if existing == path {
			r.discoveryPaths = append(r.discoveryPaths[:i], r.discoveryPaths[i+1:]...)
			r.logger.Info("発見パス削除", map[string]interface{}{
				"path": path,
			})
			return
		}
	}
}
