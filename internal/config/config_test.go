package config

import (
	"os"
	"testing"
)

// TestConfigLoadDefault はデフォルト設定の読み込みをテストする
func TestConfigLoadDefault(t *testing.T) {
	// 存在しない設定ファイルからロードする場合、デフォルト設定が返されることを確認
	config, err := Load()
	if err != nil {
		t.Fatalf("設定読み込みエラー: %v", err)
	}

	// デフォルト値の検証
	if config.Provider != "ollama" {
		t.Errorf("期待値: ollama, 実際値: %s", config.Provider)
	}
	if config.Model != "qwen2.5-coder:14b" {
		t.Errorf("期待値: qwen2.5-coder:14b, 実際値: %s", config.Model)
	}
	if config.BaseURL != "http://localhost:11434" {
		t.Errorf("期待値: http://localhost:11434, 実際値: %s", config.BaseURL)
	}
}

// TestConfigSave は設定ファイルの保存機能をテストする
func TestConfigSave(t *testing.T) {
	_ = t.TempDir() // テスト用ディレクトリ

	config := &Config{
		Provider:      "ollama",
		Model:         "test-model",
		BaseURL:       "http://localhost:11434",
		Timeout:       15,
		MaxFileSize:   524288,
		WorkspaceMode: "project_only",
	}

	// 設定を保存
	err := config.Save()
	if err != nil {
		t.Fatalf("設定保存エラー: %v", err)
	}

	// ファイルが存在することを確認
	savedConfigPath, _ := GetConfigPath()
	if _, err := os.Stat(savedConfigPath); os.IsNotExist(err) {
		t.Fatal("設定ファイルが作成されていません")
	}

	// 保存された設定を読み込み直して検証
	loadedConfig, err := Load()
	if err != nil {
		t.Fatalf("保存された設定の読み込みエラー: %v", err)
	}

	if loadedConfig.Model != "test-model" {
		t.Errorf("期待値: test-model, 実際値: %s", loadedConfig.Model)
	}
}

// TestConfigDefault はデフォルト設定の生成をテストする
func TestConfigDefault(t *testing.T) {
	config := DefaultConfig()

	if config.Provider != "ollama" {
		t.Errorf("デフォルトプロバイダーが正しくありません: %s", config.Provider)
	}
	if config.BaseURL != "http://localhost:11434" {
		t.Errorf("デフォルトエンドポイントが正しくありません: %s", config.BaseURL)
	}
	if config.Timeout <= 0 {
		t.Error("Timeoutは正の値でなければなりません")
	}
}