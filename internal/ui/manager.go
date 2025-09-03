package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/glkt/vyb-code/internal/config"
)

// UIマネージャー - TUIとレガシーモードを統一管理
type Manager struct {
	config    config.TUIConfig
	app       *App
	spinner   *Spinner
	progress  *Progress
	theme     *ThemeManager
	isRunning bool
}

// 新しいUIマネージャーを作成
func NewManager(cfg config.TUIConfig) *Manager {
	return &Manager{
		config:   cfg,
		spinner:  NewSpinner(),
		progress: NewProgress(),
		theme:    NewThemeManager(),
	}
}

// TUIモードでアプリケーションを開始
func (m *Manager) StartTUIMode() error {
	m.app = NewApp(m.config)
	program := tea.NewProgram(m.app, tea.WithAltScreen())

	m.isRunning = true
	_, err := program.Run()
	m.isRunning = false

	return err
}

// スピナーを表示（レガシーモード）
func (m *Manager) ShowSpinner(message string) {
	if !m.config.ShowSpinner {
		return
	}

	if m.isRunning && m.app != nil {
		// TUIモード
		m.app.StartSpinner(message)
	} else {
		// レガシーモード - シンプルな点線表示
		print("⠋ " + message)
	}
}

// プログレスバーを表示（レガシーモード）
func (m *Manager) ShowProgress(message string, percent float64) {
	if !m.config.ShowProgress {
		return
	}

	if m.isRunning && m.app != nil {
		// TUIモード
		m.app.StartProgress(message)
		m.app.UpdateProgress(percent)
	} else {
		// レガシーモード - シンプルなパーセント表示
		fmt.Printf("\r%s: %.1f%%", message, percent*100)
	}
}

// スピナーを停止
func (m *Manager) StopSpinner() {
	if m.spinner != nil {
		m.spinner.Stop()
	}
}

// プログレスバーを停止
func (m *Manager) StopProgress() {
	if m.progress != nil {
		m.progress.Stop()
	}
}

// 成功メッセージを表示
func (m *Manager) ShowSuccess(message string) {
	if m.isRunning && m.app != nil {
		// TUIモード用は後で実装
		println("✅ " + message)
	} else {
		// レガシーモード
		println("✅ " + message)
	}
}

// エラーメッセージを表示
func (m *Manager) ShowError(message string) {
	if m.isRunning && m.app != nil {
		// TUIモード用は後で実装
		println("❌ " + message)
	} else {
		// レガシーモード
		println("❌ " + message)
	}
}

// 警告メッセージを表示
func (m *Manager) ShowWarning(message string) {
	if m.isRunning && m.app != nil {
		// TUIモード用は後で実装
		println("⚠️ " + message)
	} else {
		// レガシーモード
		println("⚠️ " + message)
	}
}

// 情報メッセージを表示
func (m *Manager) ShowInfo(message string) {
	if m.isRunning && m.app != nil {
		// TUIモード用は後で実装
		println("ℹ️ " + message)
	} else {
		// レガシーモード
		println("ℹ️ " + message)
	}
}
