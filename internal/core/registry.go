package core

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// DefaultComponentRegistry はComponentRegistryのデフォルト実装
type DefaultComponentRegistry struct {
	mu         sync.RWMutex
	components map[string]CoreComponent
	extensions map[string]Extension
	bridges    map[string]Bridge
	status     map[string]*ComponentStatus
}

// NewComponentRegistry は新しいComponentRegistryを作成
func NewComponentRegistry() ComponentRegistry {
	return &DefaultComponentRegistry{
		components: make(map[string]CoreComponent),
		extensions: make(map[string]Extension),
		bridges:    make(map[string]Bridge),
		status:     make(map[string]*ComponentStatus),
	}
}

// RegisterCore はコアコンポーネントを登録
func (r *DefaultComponentRegistry) RegisterCore(component CoreComponent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := component.Name()
	if _, exists := r.components[name]; exists {
		return fmt.Errorf("core component '%s' already registered", name)
	}

	r.components[name] = component
	r.status[name] = &ComponentStatus{
		Name:    name,
		Running: false,
		Healthy: false,
	}

	return nil
}

// RegisterExtension は拡張機能を登録
func (r *DefaultComponentRegistry) RegisterExtension(extension Extension) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := extension.Name()
	if _, exists := r.extensions[name]; exists {
		return fmt.Errorf("extension '%s' already registered", name)
	}

	r.extensions[name] = extension
	r.status[name] = &ComponentStatus{
		Name:    name,
		Running: false,
		Healthy: false,
	}

	return nil
}

// RegisterBridge は橋渡しコンポーネントを登録
func (r *DefaultComponentRegistry) RegisterBridge(bridge Bridge) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := bridge.Name()
	if _, exists := r.bridges[name]; exists {
		return fmt.Errorf("bridge component '%s' already registered", name)
	}

	r.bridges[name] = bridge
	r.status[name] = &ComponentStatus{
		Name:    name,
		Running: false,
		Healthy: false,
	}

	return nil
}

// GetComponent はコンポーネントを取得
func (r *DefaultComponentRegistry) GetComponent(name string) (CoreComponent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// コアコンポーネントをチェック
	if comp, exists := r.components[name]; exists {
		return comp, nil
	}

	// 拡張機能をチェック
	if ext, exists := r.extensions[name]; exists {
		return ext, nil
	}

	// 橋渡しコンポーネントをチェック
	if bridge, exists := r.bridges[name]; exists {
		return bridge, nil
	}

	return nil, fmt.Errorf("component '%s' not found", name)
}

// ListComponents は登録されているコンポーネント一覧を取得
func (r *DefaultComponentRegistry) ListComponents() map[string]CoreComponent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]CoreComponent)

	// コアコンポーネントを追加
	for name, comp := range r.components {
		result[name] = comp
	}

	// 拡張機能を追加
	for name, ext := range r.extensions {
		result[name] = ext
	}

	// 橋渡しコンポーネントを追加
	for name, bridge := range r.bridges {
		result[name] = bridge
	}

	return result
}

// InitializeAll は全コンポーネントを依存関係順に初期化
func (r *DefaultComponentRegistry) InitializeAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. コアコンポーネントを初期化
	for name, comp := range r.components {
		if err := r.initializeComponent(ctx, name, comp); err != nil {
			return fmt.Errorf("failed to initialize core component '%s': %w", name, err)
		}
	}

	// 2. 橋渡しコンポーネントを初期化
	for name, bridge := range r.bridges {
		if err := r.initializeComponent(ctx, name, bridge); err != nil {
			return fmt.Errorf("failed to initialize bridge component '%s': %w", name, err)
		}
	}

	// 3. 拡張機能を優先度順に初期化
	extensions := r.getSortedExtensions()
	for _, ext := range extensions {
		name := ext.Name()

		// 依存関係チェック
		if err := r.checkDependencies(ext.Dependencies()); err != nil {
			return fmt.Errorf("extension '%s' dependency check failed: %w", name, err)
		}

		// 有効性チェック
		if !ext.IsEnabled(ctx) {
			continue
		}

		if err := r.initializeComponent(ctx, name, ext); err != nil {
			return fmt.Errorf("failed to initialize extension '%s': %w", name, err)
		}
	}

	return nil
}

// ShutdownAll は全コンポーネントをシャットダウン
func (r *DefaultComponentRegistry) ShutdownAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errors []string

	// 1. 拡張機能を逆順でシャットダウン
	extensions := r.getSortedExtensions()
	for i := len(extensions) - 1; i >= 0; i-- {
		ext := extensions[i]
		name := ext.Name()
		if err := r.shutdownComponent(ctx, name, ext); err != nil {
			errors = append(errors, fmt.Sprintf("extension '%s': %v", name, err))
		}
	}

	// 2. 橋渡しコンポーネントをシャットダウン
	for name, bridge := range r.bridges {
		if err := r.shutdownComponent(ctx, name, bridge); err != nil {
			errors = append(errors, fmt.Sprintf("bridge '%s': %v", name, err))
		}
	}

	// 3. コアコンポーネントをシャットダウン
	for name, comp := range r.components {
		if err := r.shutdownComponent(ctx, name, comp); err != nil {
			errors = append(errors, fmt.Sprintf("core '%s': %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}

// initializeComponent は個別コンポーネントを初期化
func (r *DefaultComponentRegistry) initializeComponent(ctx context.Context, name string, comp CoreComponent) error {
	status := r.status[name]
	status.StartTime = time.Now().Unix()

	if err := comp.Initialize(ctx); err != nil {
		status.Error = err.Error()
		return err
	}

	status.Running = true
	status.Healthy = true
	status.Error = ""

	return nil
}

// shutdownComponent は個別コンポーネントをシャットダウン
func (r *DefaultComponentRegistry) shutdownComponent(ctx context.Context, name string, comp CoreComponent) error {
	status := r.status[name]

	if err := comp.Shutdown(ctx); err != nil {
		status.Error = err.Error()
		status.Healthy = false
		return err
	}

	status.Running = false
	status.Healthy = false
	status.Error = ""

	return nil
}

// getSortedExtensions は拡張機能を優先度順にソート
func (r *DefaultComponentRegistry) getSortedExtensions() []Extension {
	extensions := make([]Extension, 0, len(r.extensions))
	for _, ext := range r.extensions {
		extensions = append(extensions, ext)
	}

	sort.Slice(extensions, func(i, j int) bool {
		return extensions[i].Priority() < extensions[j].Priority()
	})

	return extensions
}

// checkDependencies は依存関係をチェック
func (r *DefaultComponentRegistry) checkDependencies(dependencies []string) error {
	for _, dep := range dependencies {
		status, exists := r.status[dep]
		if !exists {
			return fmt.Errorf("dependency '%s' not found", dep)
		}

		if !status.Running || !status.Healthy {
			return fmt.Errorf("dependency '%s' is not running or unhealthy", dep)
		}
	}

	return nil
}

// DefaultModuleManager はModuleManagerのデフォルト実装
type DefaultModuleManager struct {
	*DefaultComponentRegistry
}

// NewModuleManager は新しいModuleManagerを作成
func NewModuleManager() ModuleManager {
	registry := &DefaultComponentRegistry{
		components: make(map[string]CoreComponent),
		extensions: make(map[string]Extension),
		bridges:    make(map[string]Bridge),
		status:     make(map[string]*ComponentStatus),
	}

	return &DefaultModuleManager{
		DefaultComponentRegistry: registry,
	}
}

// LifecycleManager interface implementation

// Start は指定されたコンポーネントを開始
func (m *DefaultModuleManager) Start(ctx context.Context, name string) error {
	component, err := m.GetComponent(name)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.initializeComponent(ctx, name, component)
}

// Stop は指定されたコンポーネントを停止
func (m *DefaultModuleManager) Stop(ctx context.Context, name string) error {
	component, err := m.GetComponent(name)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.shutdownComponent(ctx, name, component)
}

// Restart は指定されたコンポーネントを再起動
func (m *DefaultModuleManager) Restart(ctx context.Context, name string) error {
	if err := m.Stop(ctx, name); err != nil {
		return fmt.Errorf("failed to stop component '%s': %w", name, err)
	}

	if err := m.Start(ctx, name); err != nil {
		return fmt.Errorf("failed to start component '%s': %w", name, err)
	}

	return nil
}

// GetStatus はコンポーネントの状態を取得
func (m *DefaultModuleManager) GetStatus(name string) ComponentStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if status, exists := m.status[name]; exists {
		return *status
	}

	return ComponentStatus{
		Name:    name,
		Running: false,
		Healthy: false,
		Error:   "component not found",
	}
}

// LoadModule は指定されたモジュールを読み込み
func (m *DefaultModuleManager) LoadModule(ctx context.Context, name string) error {
	// 実装は具体的なモジュール読み込み機構に依存
	// 現在はプレースホルダー実装
	return fmt.Errorf("module loading not implemented yet")
}

// UnloadModule は指定されたモジュールをアンロード
func (m *DefaultModuleManager) UnloadModule(ctx context.Context, name string) error {
	return m.Stop(ctx, name)
}

// ReloadModule は指定されたモジュールを再読み込み
func (m *DefaultModuleManager) ReloadModule(ctx context.Context, name string) error {
	return m.Restart(ctx, name)
}

// ListModules は利用可能なモジュール一覧を取得
func (m *DefaultModuleManager) ListModules() []ComponentMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var modules []ComponentMetadata

	// コアコンポーネント
	for name := range m.components {
		status := m.status[name]
		modules = append(modules, ComponentMetadata{
			Name:    name,
			Type:    TypeCore,
			Version: "1.0.0", // 実際の実装では動的に取得
			Enabled: status.Running,
		})
	}

	// 拡張機能
	for name, ext := range m.extensions {
		status := m.status[name]
		modules = append(modules, ComponentMetadata{
			Name:         name,
			Type:         TypeExtension,
			Version:      "1.0.0",
			Dependencies: ext.Dependencies(),
			Optional:     true,
			Enabled:      status.Running,
		})
	}

	// 橋渡しコンポーネント
	for name, bridge := range m.bridges {
		status := m.status[name]
		modules = append(modules, ComponentMetadata{
			Name:     name,
			Type:     TypeBridge,
			Version:  "1.0.0",
			Optional: !bridge.IsRequired(),
			Enabled:  status.Running,
		})
	}

	return modules
}
