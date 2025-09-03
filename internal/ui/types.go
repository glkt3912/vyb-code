package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/glkt/vyb-code/internal/config"
)

// デフォルトTUI設定
func DefaultTUIConfig() config.TUIConfig {
	return config.TUIConfig{
		Enabled:      true,
		Theme:        "auto",
		ShowSpinner:  true,
		ShowProgress: true,
		Animation:    true,
	}
}

// UI状態管理
type UIState string

const (
	UIStateIdle       UIState = "idle"
	UIStateLoading    UIState = "loading"
	UIStateProcessing UIState = "processing"
	UIStateCompleted  UIState = "completed"
	UIStateError      UIState = "error"
)

// アプリケーションモデル
type AppModel struct {
	State       UIState
	CurrentView ViewType
	Config      config.TUIConfig
	Message     string
	Error       error
	StartTime   time.Time
	ProgressVal float64 // 0.0-1.0

	// サブコンポーネント
	Spinner     SpinnerModel
	ProgressBar ProgressModel
	Menu        MenuModel
}

// ビューの種類
type ViewType string

const (
	ViewMain    ViewType = "main"
	ViewConfig  ViewType = "config"
	ViewHealth  ViewType = "health"
	ViewSearch  ViewType = "search"
	ViewAnalyze ViewType = "analyze"
)

// Tea Program メッセージ
type TickMsg time.Time
type ProgressMsg float64
type CompleteMsg struct{}
type ErrorMsg error
type ViewChangeMsg ViewType

// スピナーモデル
type SpinnerModel struct {
	Active  bool
	Message string
	spinner *Spinner
}

// プログレスバーモデル
type ProgressModel struct {
	Active   bool
	Value    float64
	Message  string
	progress *Progress
}

// メニューモデル
type MenuModel struct {
	Active   bool
	Selected int
	Items    []MenuItem
}

// メニューアイテム
type MenuItem struct {
	Label       string
	Description string
	Action      func() tea.Cmd
	Enabled     bool
}
