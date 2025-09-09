package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// 確認ダイアログモデル
type ConfirmationModel struct {
	Title     string
	Message   string
	Options   []string
	Selected  int
	Confirmed bool
	Cancelled bool
	Width     int
	Height    int
}

// キーマップ
type confirmationKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Select key.Binding
	Cancel key.Binding
}

func (k confirmationKeyMap) ShortHelp() []key.Help {
	return []key.Help{
		{Key: "↑/↓", Desc: "選択"},
		{Key: "Enter", Desc: "決定"},
		{Key: "q/Esc", Desc: "キャンセル"},
	}
}

func (k confirmationKeyMap) FullHelp() [][]key.Help {
	return [][]key.Help{
		{
			{Key: k.Up.Keys()[0], Desc: k.Up.Help().Desc},
			{Key: k.Down.Keys()[0], Desc: k.Down.Help().Desc},
			{Key: k.Select.Keys()[0], Desc: k.Select.Help().Desc},
			{Key: k.Cancel.Keys()[0], Desc: k.Cancel.Help().Desc},
		},
	}
}

// デフォルトキーマップ
func defaultConfirmationKeys() confirmationKeyMap {
	return confirmationKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "上に移動"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "下に移動"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "左に移動"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "右に移動"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("Enter/Space", "選択"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q/Esc", "キャンセル"),
		),
	}
}

// 新しい確認ダイアログを作成
func NewConfirmationDialog(title, message string, options []string) ConfirmationModel {
	if len(options) == 0 {
		options = []string{"はい", "いいえ"}
	}

	return ConfirmationModel{
		Title:    title,
		Message:  message,
		Options:  options,
		Selected: 1,  // デフォルトは「いいえ」
		Width:    45, // 小さくした
		Height:   10, // 小さくした
	}
}

// 標準的な実行確認ダイアログ
func NewExecutionConfirmDialog(command string) ConfirmationModel {
	title := "⚠️  コマンド実行確認"
	message := fmt.Sprintf("以下のコマンドを実行しますか？\n\n%s", command)
	options := []string{"✅ 実行", "❌ キャンセル"}

	model := NewConfirmationDialog(title, message, options)
	model.Selected = 1 // デフォルトはキャンセル（安全性優先）
	return model
}

// 初期化
func (m ConfirmationModel) Init() tea.Cmd {
	return nil
}

// 更新
func (m ConfirmationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keys := defaultConfirmationKeys()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if m.Selected > 0 {
				m.Selected--
			}
		case key.Matches(msg, keys.Down):
			if m.Selected < len(m.Options)-1 {
				m.Selected++
			}
		case key.Matches(msg, keys.Left):
			if m.Selected > 0 {
				m.Selected--
			}
		case key.Matches(msg, keys.Right):
			if m.Selected < len(m.Options)-1 {
				m.Selected++
			}
		case key.Matches(msg, keys.Select):
			m.Confirmed = m.Selected == 0 // 最初のオプション（はい/実行）が選択された場合
			return m, tea.Quit
		case key.Matches(msg, keys.Cancel):
			m.Cancelled = true
			return m, tea.Quit
		}
	case tea.MouseMsg:
		// マウスクリック対応
		if msg.Type == tea.MouseLeft {
			// ボタン領域でのクリックを検出（簡易実装）
			// 実際のボタン位置計算は複雑なので、左右で判定
			buttonWidth := 12 // 各ボタンの推定幅
			totalWidth := buttonWidth * len(m.Options)
			startX := (m.Width - totalWidth) / 2

			for i := range m.Options {
				buttonStart := startX + (i * buttonWidth)
				buttonEnd := buttonStart + buttonWidth

				if msg.X >= buttonStart && msg.X < buttonEnd {
					m.Selected = i
					m.Confirmed = i == 0
					return m, tea.Quit
				}
			}
		}
	}

	return m, nil
}

// ビュー
func (m ConfirmationModel) View() string {
	// コンパクトなスタイル定義
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7"))

	selectedButtonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("12")).
		Bold(true).
		Padding(0, 1)

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Background(lipgloss.Color("8")).
		Padding(0, 1)

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(1).
		Width(m.Width)

	// コンパクトなコンテンツ構築
	content := titleStyle.Render(m.Title) + "\n"
	content += messageStyle.Render(m.Message) + "\n\n"

	// 横並びボタン
	buttons := ""
	for i, option := range m.Options {
		if i > 0 {
			buttons += "  " // ボタン間スペースを増やす
		}
		if i == m.Selected {
			buttons += selectedButtonStyle.Render(fmt.Sprintf("→ %s ←", option)) // 選択状態を明確に
		} else {
			buttons += buttonStyle.Render(fmt.Sprintf("  %s  ", option))
		}
	}

	content += lipgloss.NewStyle().Align(lipgloss.Center).Render(buttons) + "\n"

	// 簡潔なヘルプ
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Align(lipgloss.Center)

	content += helpStyle.Render("↑↓/クリック: 選択 Enter: 決定 q: キャンセル")

	return containerStyle.Render(content)
}

// 確認ダイアログを実行して結果を取得
func RunConfirmationDialog(model ConfirmationModel) (bool, error) {
	// マウスサポートを有効にする
	program := tea.NewProgram(model, tea.WithMouseCellMotion())
	finalModel, err := program.Run()
	if err != nil {
		return false, err
	}

	result := finalModel.(ConfirmationModel)
	if result.Cancelled {
		return false, nil
	}

	return result.Confirmed, nil
}
