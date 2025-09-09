package ui

import (
	"testing"
	"time"
)

// TestDefaultTUIConfig はデフォルトTUI設定をテストする
func TestDefaultTUIConfig(t *testing.T) {
	config := DefaultTUIConfig()

	if !config.Enabled {
		t.Error("Enabled が false になっています")
	}

	if config.Theme != "auto" {
		t.Errorf("期待値: auto, 実際値: %s", config.Theme)
	}

	if !config.ShowSpinner {
		t.Error("ShowSpinner が false になっています")
	}

	if !config.ShowProgress {
		t.Error("ShowProgress が false になっています")
	}

	if !config.Animation {
		t.Error("Animation が false になっています")
	}
}

// TestUIState はUI状態をテストする
func TestUIState(t *testing.T) {
	states := []UIState{
		UIStateIdle,
		UIStateLoading,
		UIStateProcessing,
		UIStateCompleted,
		UIStateError,
	}

	expectedStrings := []string{
		"idle",
		"loading",
		"processing",
		"completed",
		"error",
	}

	for i, state := range states {
		if string(state) != expectedStrings[i] {
			t.Errorf("UI状態 %d: 期待値 %s, 実際値 %s", i, expectedStrings[i], string(state))
		}
	}
}

// TestAppModel はアプリケーションモデルをテストする
func TestAppModel(t *testing.T) {
	model := AppModel{
		State:       UIStateIdle,
		CurrentView: ViewMain,
		Config:      DefaultTUIConfig(),
		Message:     "テストメッセージ",
		Error:       nil,
		StartTime:   time.Now(),
		ProgressVal: 0.5,
		Spinner:     SpinnerModel{Active: false, Message: ""},
		ProgressBar: ProgressModel{Active: false, Value: 0.0, Message: ""},
		Menu:        MenuModel{Active: false, Selected: 0, Items: []MenuItem{}},
	}

	if model.State != UIStateIdle {
		t.Errorf("期待値: %s, 実際値: %s", UIStateIdle, model.State)
	}

	if model.CurrentView != ViewMain {
		t.Errorf("期待値: %s, 実際値: %s", ViewMain, model.CurrentView)
	}

	if model.Message != "テストメッセージ" {
		t.Errorf("期待値: テストメッセージ, 実際値: %s", model.Message)
	}

	if model.ProgressVal != 0.5 {
		t.Errorf("期待値: 0.5, 実際値: %f", model.ProgressVal)
	}
}

// TestViewType はビュータイプをテストする
func TestViewType(t *testing.T) {
	views := []ViewType{
		ViewMain,
		ViewConfig,
		ViewHealth,
		ViewSearch,
		ViewAnalyze,
	}

	expectedStrings := []string{
		"main",
		"config",
		"health",
		"search",
		"analyze",
	}

	for i, view := range views {
		if string(view) != expectedStrings[i] {
			t.Errorf("ビュータイプ %d: 期待値 %s, 実際値 %s", i, expectedStrings[i], string(view))
		}
	}
}

// TestSpinnerModel はスピナーモデルをテストする
func TestSpinnerModel(t *testing.T) {
	spinner := SpinnerModel{
		Active:  true,
		Message: "読み込み中...",
		spinner: nil, // プライベートフィールド
	}

	if !spinner.Active {
		t.Error("Active が false になっています")
	}

	if spinner.Message != "読み込み中..." {
		t.Errorf("期待値: 読み込み中..., 実際値: %s", spinner.Message)
	}
}

// TestProgressModel はプログレスモデルをテストする
func TestProgressModel(t *testing.T) {
	progress := ProgressModel{
		Active:   true,
		Value:    0.75,
		Message:  "処理中...",
		progress: nil, // プライベートフィールド
	}

	if !progress.Active {
		t.Error("Active が false になっています")
	}

	if progress.Value != 0.75 {
		t.Errorf("期待値: 0.75, 実際値: %f", progress.Value)
	}

	if progress.Message != "処理中..." {
		t.Errorf("期待値: 処理中..., 実際値: %s", progress.Message)
	}
}

// TestMenuModel はメニューモデルをテストする
func TestMenuModel(t *testing.T) {
	items := []MenuItem{
		{
			Label:       "項目1",
			Description: "最初の項目",
			Action:      nil,
			Enabled:     true,
		},
		{
			Label:       "項目2",
			Description: "2番目の項目",
			Action:      nil,
			Enabled:     false,
		},
	}

	menu := MenuModel{
		Active:   true,
		Selected: 0,
		Items:    items,
	}

	if !menu.Active {
		t.Error("Active が false になっています")
	}

	if menu.Selected != 0 {
		t.Errorf("期待値: 0, 実際値: %d", menu.Selected)
	}

	if len(menu.Items) != 2 {
		t.Errorf("期待値: 2, 実際値: %d", len(menu.Items))
	}

	if menu.Items[0].Label != "項目1" {
		t.Errorf("期待値: 項目1, 実際値: %s", menu.Items[0].Label)
	}

	if menu.Items[1].Enabled {
		t.Error("項目2の Enabled が true になっています")
	}
}

// TestMenuItem はメニュー項目をテストする
func TestMenuItem(t *testing.T) {
	item := MenuItem{
		Label:       "テスト項目",
		Description: "テスト用の項目です",
		Action:      nil,
		Enabled:     true,
	}

	if item.Label != "テスト項目" {
		t.Errorf("期待値: テスト項目, 実際値: %s", item.Label)
	}

	if item.Description != "テスト用の項目です" {
		t.Errorf("期待値: テスト用の項目です, 実際値: %s", item.Description)
	}

	if !item.Enabled {
		t.Error("Enabled が false になっています")
	}
}
