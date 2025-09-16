package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/core"
)

func TestPluginIntegration_FullWorkflow(t *testing.T) {
	// セットアップ
	ctx := context.Background()
	mockLogger := &MockLogger{}
	mockConfig := &config.Config{}
	mockComponentReg := &MockComponentRegistry{}
	mockLifecycleManager := &MockLifecycleManager{}

	// プラグイン統合システムを作成
	integration := NewPluginIntegration(mockComponentReg, mockLifecycleManager, mockLogger, mockConfig)

	// 初期化
	err := integration.Initialize(ctx)
	if err != nil {
		t.Fatalf("統合システム初期化失敗: %v", err)
	}

	// プラグイン一覧を取得
	plugins := integration.ListPlugins()
	if len(plugins) < 3 {
		t.Errorf("組み込みプラグインが正しく登録されていません。期待: 3以上, 実際: %d", len(plugins))
	}

	// 各プラグインタイプが存在することを確認
	hasCore := false
	hasExtension := false
	hasBridge := false

	for _, plugin := range plugins {
		switch plugin.Metadata.Type {
		case core.TypeCore:
			hasCore = true
		case core.TypeExtension:
			hasExtension = true
		case core.TypeBridge:
			hasBridge = true
		}
	}

	if !hasCore {
		t.Error("コアプラグインが見つかりません")
	}
	if !hasExtension {
		t.Error("拡張プラグインが見つかりません")
	}
	if !hasBridge {
		t.Error("ブリッジプラグインが見つかりません")
	}

	// 統計情報を取得
	stats := integration.GetStats()
	if !stats.Enabled {
		t.Error("プラグインシステムが有効になっていません")
	}

	if stats.BuiltinCount == 0 {
		t.Error("組み込みプラグインが登録されていません")
	}

	t.Logf("プラグイン統合テスト成功 - 組み込み: %d, 外部: %d, 総数: %d",
		stats.BuiltinCount, stats.ExternalCount, stats.ManagerStats.PluginStats.TotalPlugins)

	// 終了処理
	err = integration.Shutdown(ctx)
	if err != nil {
		t.Errorf("統合システム終了エラー: %v", err)
	}
}

func TestExamplePlugin_Lifecycle(t *testing.T) {
	ctx := context.Background()
	mockLogger := &MockLogger{}
	mockConfig := &config.Config{}

	// サンプルプラグインを作成
	component, err := NewExamplePlugin(mockLogger, mockConfig)
	if err != nil {
		t.Fatalf("サンプルプラグイン作成失敗: %v", err)
	}

	// 名前を確認
	if component.Name() != "example_plugin" {
		t.Errorf("プラグイン名が正しくありません。期待: example_plugin, 実際: %s", component.Name())
	}

	// 初期化
	err = component.Initialize(ctx)
	if err != nil {
		t.Errorf("プラグイン初期化失敗: %v", err)
	}

	// 健全性チェック
	err = component.Health(ctx)
	if err != nil {
		t.Errorf("プラグイン健全性チェック失敗: %v", err)
	}

	// シャットダウン
	err = component.Shutdown(ctx)
	if err != nil {
		t.Errorf("プラグインシャットダウン失敗: %v", err)
	}

	// シャットダウン後の健全性チェック（失敗するはず）
	err = component.Health(ctx)
	if err == nil {
		t.Error("シャットダウン後の健全性チェックが成功してしまいました")
	}
}

func TestExampleExtensionPlugin_Interface(t *testing.T) {
	ctx := context.Background()
	mockLogger := &MockLogger{}
	mockConfig := &config.Config{}

	// 拡張機能プラグインを作成
	extension, err := NewExampleExtensionPlugin(mockLogger, mockConfig)
	if err != nil {
		t.Fatalf("拡張機能プラグイン作成失敗: %v", err)
	}

	// Extension インターフェースの確認
	if extension.Name() != "example_extension" {
		t.Errorf("拡張機能名が正しくありません。期待: example_extension, 実際: %s", extension.Name())
	}

	// 依存関係の確認
	deps := extension.Dependencies()
	if len(deps) != 2 {
		t.Errorf("依存関係の数が正しくありません。期待: 2, 実際: %d", len(deps))
	}

	// 有効性の確認
	if !extension.IsEnabled(ctx) {
		t.Error("拡張機能が無効になっています")
	}

	// 優先度の確認
	if extension.Priority() != 10 {
		t.Errorf("優先度が正しくありません。期待: 10, 実際: %d", extension.Priority())
	}
}

func TestExampleBridgePlugin_Interface(t *testing.T) {
	ctx := context.Background()
	mockLogger := &MockLogger{}
	mockConfig := &config.Config{}

	// ブリッジプラグインを作成
	bridge, err := NewExampleBridgePlugin(mockLogger, mockConfig)
	if err != nil {
		t.Fatalf("ブリッジプラグイン作成失敗: %v", err)
	}
	_ = ctx // prevent unused variable error

	// Bridge インターフェースの確認
	if bridge.Name() != "example_bridge" {
		t.Errorf("ブリッジ名が正しくありません。期待: example_bridge, 実際: %s", bridge.Name())
	}

	// 接続先の確認
	connects := bridge.ConnectsTo()
	if len(connects) != 1 || connects[0] != "example_extension" {
		t.Errorf("接続先が正しくありません。期待: [example_extension], 実際: %v", connects)
	}

	// 必須性の確認
	if bridge.IsRequired() {
		t.Error("ブリッジが必須になっています（テストでは非必須のはず）")
	}
}

func TestPluginCommand_ListCommand(t *testing.T) {
	// セットアップ
	ctx := context.Background()
	mockLogger := &MockLogger{}
	mockConfig := &config.Config{}
	mockComponentReg := &MockComponentRegistry{}
	mockLifecycleManager := &MockLifecycleManager{}

	integration := NewPluginIntegration(mockComponentReg, mockLifecycleManager, mockLogger, mockConfig)
	err := integration.Initialize(ctx)
	if err != nil {
		t.Fatalf("統合システム初期化失敗: %v", err)
	}

	// プラグインコマンドを作成
	cmd := integration.CreatePluginCommand()

	// リストコマンドを実行（出力は確認しないがエラーが発生しないことを確認）
	err = cmd.ExecuteListCommand(ctx)
	if err != nil {
		t.Errorf("リストコマンド実行失敗: %v", err)
	}

	// 統計コマンドを実行
	err = cmd.ExecuteStatsCommand()
	if err != nil {
		t.Errorf("統計コマンド実行失敗: %v", err)
	}

	// 終了処理
	integration.Shutdown(ctx)
}

func TestPluginManager_LoadUnloadCycle(t *testing.T) {
	ctx := context.Background()
	mockLogger := &MockLogger{}
	mockConfig := &config.Config{}
	mockComponentReg := &MockComponentRegistry{}
	mockLifecycleManager := &MockLifecycleManager{}

	integration := NewPluginIntegration(mockComponentReg, mockLifecycleManager, mockLogger, mockConfig)
	err := integration.Initialize(ctx)
	if err != nil {
		t.Fatalf("統合システム初期化失敗: %v", err)
	}

	pluginName := "example_core"

	// 初期状態でプラグインがロードされていることを確認
	info, err := integration.GetPluginInfo(pluginName)
	if err != nil {
		t.Fatalf("プラグイン情報取得失敗: %v", err)
	}

	if info.Status != StatusLoaded {
		t.Errorf("プラグインの初期状態が正しくありません。期待: %s, 実際: %s",
			StatusLoaded.String(), info.Status.String())
	}

	// プラグインをアンロード
	err = integration.UnloadPlugin(ctx, pluginName)
	if err != nil {
		t.Errorf("プラグインアンロード失敗: %v", err)
	}

	// アンロード後の状態を確認
	info, err = integration.GetPluginInfo(pluginName)
	if err != nil {
		t.Fatalf("アンロード後のプラグイン情報取得失敗: %v", err)
	}

	if info.Status != StatusUnloaded {
		t.Errorf("アンロード後の状態が正しくありません。期待: %s, 実際: %s",
			StatusUnloaded.String(), info.Status.String())
	}

	t.Log("プラグインの読み込み・アンロードサイクルテスト完了")

	// 終了処理
	integration.Shutdown(ctx)
}

func TestPluginScheduler_Integration(t *testing.T) {
	ctx := context.Background()
	mockLogger := &MockLogger{}
	scheduler := NewPluginScheduler(mockLogger)

	// スケジューラーを開始
	scheduler.Start(ctx)
	defer scheduler.Stop(ctx)

	// テスト用タスクをスケジュール
	executed := false
	testTask := func(ctx context.Context) error {
		executed = true
		return nil
	}
	_ = executed // prevent unused variable error

	err := scheduler.ScheduleOnce("test_task", 100*time.Millisecond, testTask)
	if err != nil {
		t.Fatalf("タスクスケジュール失敗: %v", err)
	}

	// タスクが実行されるまで少し待機
	time.Sleep(200 * time.Millisecond)

	// 統計を確認
	stats := scheduler.GetStats()
	if !stats.Running {
		t.Error("スケジューラーが実行中になっていません")
	}

	if stats.TotalTasks == 0 {
		t.Error("タスクが登録されていません")
	}

	t.Logf("スケジューラー統計 - 総タスク: %d, 有効: %d, 実行回数: %d",
		stats.TotalTasks, stats.EnabledTasks, stats.TotalRuns)
}

func TestPluginSecurity_Integration(t *testing.T) {
	ctx := context.Background()
	mockLogger := &MockLogger{}
	security := NewPluginSecurity(mockLogger)

	// 初期化
	err := security.Initialize(ctx)
	if err != nil {
		t.Fatalf("セキュリティ初期化失敗: %v", err)
	}

	// セキュリティレベルを変更
	security.SetSecurityLevel(SecurityLevelHigh)

	// セキュリティ情報を取得
	info := security.GetSecurityInfo()
	if info.Policy.Level != SecurityLevelHigh {
		t.Errorf("セキュリティレベルが正しく設定されていません。期待: %s, 実際: %s",
			SecurityLevelHigh.String(), info.Policy.Level.String())
	}

	// ブラックリストテスト
	security.AddToBlacklist("malicious_plugin")
	err = security.ValidatePlugin("malicious_plugin")
	if err == nil {
		t.Error("ブラックリストされたプラグインが検証を通過しました")
	}

	// 正常プラグインのテスト
	err = security.ValidatePlugin("safe_plugin")
	if err != nil {
		t.Errorf("安全なプラグインの検証に失敗: %v", err)
	}

	t.Log("プラグインセキュリティ統合テスト完了")
}
