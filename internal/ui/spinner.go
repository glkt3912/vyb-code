package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// スピナーコンポーネント
type Spinner struct {
	spinner spinner.Model
	message string
	active  bool
	style   lipgloss.Style
}

// 新しいスピナーを作成
func NewSpinner() *Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &Spinner{
		spinner: s,
		active:  false,
		style: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginLeft(1),
	}
}

// スピナーを開始
func (s *Spinner) Start(message string) tea.Cmd {
	s.message = message
	s.active = true
	return s.spinner.Tick
}

// スピナーを停止
func (s *Spinner) Stop() {
	s.active = false
	s.message = ""
}

// メッセージを更新
func (s *Spinner) SetMessage(message string) {
	s.message = message
}

// アクティブ状態を確認
func (s *Spinner) IsActive() bool {
	return s.active
}

// Init implements tea.Model
func (s *Spinner) Init() tea.Cmd {
	return s.spinner.Tick
}

// Update implements tea.Model
func (s *Spinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !s.active {
		return s, nil
	}

	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// View implements tea.Model
func (s *Spinner) View() string {
	if !s.active {
		return ""
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		s.spinner.View(),
		s.style.Render(" "+s.message),
	)
}

// プリセットスピナースタイル
func (s *Spinner) SetStyle(style SpinnerStyle) {
	switch style {
	case SpinnerStyleDot:
		s.spinner.Spinner = spinner.Dot
	case SpinnerStyleLine:
		s.spinner.Spinner = spinner.Line
	case SpinnerStyleMiniDot:
		s.spinner.Spinner = spinner.MiniDot
	case SpinnerStylePoints:
		s.spinner.Spinner = spinner.Points
	case SpinnerStylePulse:
		s.spinner.Spinner = spinner.Pulse
	default:
		s.spinner.Spinner = spinner.Dot
	}
}

// スピナースタイル定義
type SpinnerStyle int

const (
	SpinnerStyleDot SpinnerStyle = iota
	SpinnerStyleLine
	SpinnerStyleMiniDot
	SpinnerStylePoints
	SpinnerStylePulse
)

// LLM応答待機用の特別なスピナー
func NewLLMSpinner() *Spinner {
	s := NewSpinner()
	s.SetStyle(SpinnerStylePulse)
	s.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86")) // 明るい緑
	return s
}

// 検索処理用の特別なスピナー
func NewSearchSpinner() *Spinner {
	s := NewSpinner()
	s.SetStyle(SpinnerStyleDot)
	s.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("33")) // 青
	return s
}

// ヘルスチェック用の特別なスピナー
func NewHealthSpinner() *Spinner {
	s := NewSpinner()
	s.SetStyle(SpinnerStylePoints)
	s.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // オレンジ
	return s
}
