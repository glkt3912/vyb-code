package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/core"
	"github.com/glkt/vyb-code/internal/logger"
)

func TestPluginRegistry_Creation(t *testing.T) {
	mockLogger := &MockLogger{}
	mockConfig := &config.Config{}
	mockComponentReg := &MockComponentRegistry{}
	mockLifecycleManager := &MockLifecycleManager{}

	registry := NewPluginRegistry(mockComponentReg, mockLifecycleManager, mockLogger, mockConfig)
	
	if registry == nil {
		t.Fatal("プラグインレジストリの作成に失敗")
	}

	if len(registry.plugins) != 0 {
		t.Error("初期状態でプラグインが存在しています")
	}
}

func TestPluginManager_Creation(t *testing.T) {
	mockLogger := &MockLogger{}
	mockConfig := &config.Config{}
	mockComponentReg := &MockComponentRegistry{}
	mockLifecycleManager := &MockLifecycleManager{}

	manager := NewPluginManager(mockComponentReg, mockLifecycleManager, mockLogger, mockConfig)
	
	if manager == nil {
		t.Fatal("プラグインマネージャーの作成に失敗")
	}

	if manager.autoDiscovery != true {
		t.Error("デフォルトの自動発見設定が間違っています")
	}
}

func TestPluginScheduler_Creation(t *testing.T) {
	mockLogger := &MockLogger{}
	
	scheduler := NewPluginScheduler(mockLogger)
	
	if scheduler == nil {
		t.Fatal("プラグインスケジューラーの作成に失敗")
	}

	if len(scheduler.tasks) != 0 {
		t.Error("初期状態でタスクが存在しています")
	}
}

func TestPluginScheduler_TaskScheduling(t *testing.T) {
	mockLogger := &MockLogger{}
	scheduler := NewPluginScheduler(mockLogger)
	
	// テスト用のタスク関数
	executed := false
	testTask := func(ctx context.Context) error {
		executed = true
		return nil
	}
	_ = executed // prevent unused variable error
	
	// 繰り返しタスクをスケジュール
	err := scheduler.ScheduleRepeating("test_task", 1*time.Second, testTask)
	if err != nil {
		t.Fatalf("タスクスケジュール失敗: %v", err)
	}
	
	// タスクが登録されたかチェック
	if len(scheduler.tasks) != 1 {
		t.Error("タスクが正しく登録されていません")
	}
	
	// タスク情報を取得
	task, err := scheduler.GetTask("test_task")
	if err != nil {
		t.Fatalf("タスク取得失敗: %v", err)
	}
	
	if task.Name != "test_task" {
		t.Error("タスク名が正しくありません")
	}
	
	if task.Interval != 1*time.Second {
		t.Error("タスク間隔が正しくありません")
	}
}

func TestPluginSecurity_Creation(t *testing.T) {
	mockLogger := &MockLogger{}
	
	security := NewPluginSecurity(mockLogger)
	
	if security == nil {
		t.Fatal("プラグインセキュリティの作成に失敗")
	}

	if security.policy.Level != SecurityLevelModerate {
		t.Error("デフォルトセキュリティレベルが正しくありません")
	}
}

func TestPluginSecurity_ValidationBasic(t *testing.T) {
	mockLogger := &MockLogger{}
	security := NewPluginSecurity(mockLogger)
	
	ctx := context.Background()
	if err := security.Initialize(ctx); err != nil {
		t.Fatalf("セキュリティ初期化失敗: %v", err)
	}
	
	// 正常なプラグイン名のテスト
	err := security.ValidatePlugin("valid_plugin")
	if err != nil {
		t.Errorf("正常なプラグイン名の検証に失敗: %v", err)
	}
	
	// ブラックリストされたプラグインのテスト
	security.AddToBlacklist("malicious_plugin")
	err = security.ValidatePlugin("malicious_plugin")
	if err == nil {
		t.Error("ブラックリストされたプラグインが検証を通過しました")
	}
}

func TestPluginConfigManager_Creation(t *testing.T) {
	mockLogger := &MockLogger{}
	mockConfig := &config.Config{}
	
	configManager := NewPluginConfigManager(mockConfig, mockLogger)
	
	if configManager == nil {
		t.Fatal("プラグイン設定マネージャーの作成に失敗")
	}

	if len(configManager.pluginConfigs) != 0 {
		t.Error("初期状態で設定が存在しています")
	}
}

func TestPluginConfigManager_DefaultConfig(t *testing.T) {
	mockLogger := &MockLogger{}
	mockConfig := &config.Config{}
	
	configManager := NewPluginConfigManager(mockConfig, mockLogger)
	pluginName := "test_plugin"
	
	// デフォルト設定を取得
	config, err := configManager.GetPluginConfig(pluginName)
	if err != nil {
		t.Fatalf("デフォルト設定取得失敗: %v", err)
	}
	
	if config.Metadata.Name != pluginName {
		t.Error("デフォルト設定のプラグイン名が正しくありません")
	}
	
	if !config.Metadata.Enabled {
		t.Error("デフォルト設定では有効になっているべきです")
	}
}

func TestSecurityLevel_String(t *testing.T) {
	tests := []struct {
		level    SecurityLevel
		expected string
	}{
		{SecurityLevelLow, "low"},
		{SecurityLevelModerate, "moderate"},
		{SecurityLevelHigh, "high"},
		{SecurityLevelStrict, "strict"},
	}
	
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("SecurityLevel.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestPluginStatus_String(t *testing.T) {
	tests := []struct {
		status   PluginStatus
		expected string
	}{
		{StatusUnloaded, "unloaded"},
		{StatusLoading, "loading"},
		{StatusLoaded, "loaded"},
		{StatusActive, "active"},
		{StatusError, "error"},
		{StatusDisabled, "disabled"},
	}
	
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("PluginStatus.String() = %v, want %v", got, tt.expected)
		}
	}
}

// MockComponentRegistry はテスト用のモックコンポーネントレジストリ
type MockComponentRegistry struct{}

func (m *MockComponentRegistry) RegisterCore(component core.CoreComponent) error {
	return nil
}

func (m *MockComponentRegistry) RegisterExtension(extension core.Extension) error {
	return nil
}

func (m *MockComponentRegistry) RegisterBridge(bridge core.Bridge) error {
	return nil
}

func (m *MockComponentRegistry) GetComponent(name string) (core.CoreComponent, error) {
	return nil, nil
}

func (m *MockComponentRegistry) ListComponents() map[string]core.CoreComponent {
	return make(map[string]core.CoreComponent)
}

func (m *MockComponentRegistry) InitializeAll(ctx context.Context) error {
	return nil
}

func (m *MockComponentRegistry) ShutdownAll(ctx context.Context) error {
	return nil
}

// MockLifecycleManager はテスト用のモックライフサイクルマネージャー
type MockLifecycleManager struct{}

func (m *MockLifecycleManager) Start(ctx context.Context, name string) error {
	return nil
}

func (m *MockLifecycleManager) Stop(ctx context.Context, name string) error {
	return nil
}

func (m *MockLifecycleManager) Restart(ctx context.Context, name string) error {
	return nil
}

func (m *MockLifecycleManager) GetStatus(name string) core.ComponentStatus {
	return core.ComponentStatus{
		Name:    name,
		Running: true,
		Healthy: true,
	}
}

// MockLogger はテスト用のモックロガー
type MockLogger struct{}

func (m *MockLogger) Debug(msg string, fields map[string]interface{}) {}
func (m *MockLogger) Info(msg string, fields map[string]interface{})  {}
func (m *MockLogger) Warn(msg string, fields map[string]interface{})  {}
func (m *MockLogger) Error(msg string, fields map[string]interface{}) {}
func (m *MockLogger) Fatal(msg string, fields map[string]interface{}) {}
func (m *MockLogger) SetLevel(level string) error                     { return nil }
func (m *MockLogger) GetLevel() string                                { return "info" }
func (m *MockLogger) WithFields(fields map[string]interface{}) logger.Logger {
	return m
}