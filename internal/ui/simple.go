package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/glkt/vyb-code/internal/config"
)

// シンプルなTUIアプリ
type SimpleApp struct {
	config config.TUIConfig
	theme  *ThemeManager
}

// シンプルアプリを作成
func NewSimpleApp(cfg config.TUIConfig) *SimpleApp {
	return &SimpleApp{
		config: cfg,
		theme:  NewThemeManager(),
	}
}

// Init implements tea.Model
func (s *SimpleApp) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (s *SimpleApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return s, tea.Quit
		}
	}
	return s, nil
}

// View implements tea.Model
func (s *SimpleApp) View() string {
	header := s.theme.HeaderStyle().Render("🎵 vyb - Feel the rhythm of perfect code")
	content := s.theme.InfoStyle().Render("TUIモードが有効です。'q'で終了します。")
	footer := s.theme.InfoStyle().Render("キーボード: q=終了, Ctrl+C=強制終了")

	return header + "\n\n" + content + "\n\n" + footer
}
