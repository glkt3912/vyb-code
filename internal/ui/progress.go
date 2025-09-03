package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// プログレスバーコンポーネント
type Progress struct {
	progress progress.Model
	message  string
	active   bool
	current  float64
	style    lipgloss.Style
}

// 新しいプログレスバーを作成
func NewProgress() *Progress {
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 40

	return &Progress{
		progress: prog,
		active:   false,
		current:  0.0,
		style: lipgloss.NewStyle().
			MarginLeft(1).
			MarginRight(1),
	}
}

// プログレスバーを開始
func (p *Progress) Start(message string) tea.Cmd {
	p.message = message
	p.active = true
	p.current = 0.0
	return nil
}

// プログレスバーを停止
func (p *Progress) Stop() {
	p.active = false
	p.message = ""
	p.current = 0.0
}

// 進捗を更新
func (p *Progress) SetProgress(percent float64) tea.Cmd {
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}

	p.current = percent
	return p.progress.SetPercent(percent)
}

// メッセージを更新
func (p *Progress) SetMessage(message string) {
	p.message = message
}

// アクティブ状態を確認
func (p *Progress) IsActive() bool {
	return p.active
}

// 現在の進捗を取得
func (p *Progress) GetProgress() float64 {
	return p.current
}

// Init implements tea.Model
func (p *Progress) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (p *Progress) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !p.active {
		return p, nil
	}

	switch msg := msg.(type) {
	case ProgressMsg:
		return p, p.SetProgress(float64(msg))
	}

	var cmd tea.Cmd
	newModel, cmd := p.progress.Update(msg)
	if prog, ok := newModel.(progress.Model); ok {
		p.progress = prog
	}
	return p, cmd
}

// View implements tea.Model
func (p *Progress) View() string {
	if !p.active {
		return ""
	}

	progressBar := p.progress.View()
	percentage := fmt.Sprintf("%.1f%%", p.current*100)

	// メッセージと進捗バーを組み合わせ
	if p.message != "" {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			p.style.Render(p.message),
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				progressBar,
				lipgloss.NewStyle().MarginLeft(2).Render(percentage),
			),
		)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		progressBar,
		lipgloss.NewStyle().MarginLeft(2).Render(percentage),
	)
}

// 幅を設定
func (p *Progress) SetWidth(width int) {
	p.progress.Width = width
}

// カラーグラデーションを設定
func (p *Progress) SetColors(colors []string) {
	var lipglossColors []lipgloss.Color
	for _, color := range colors {
		lipglossColors = append(lipglossColors, lipgloss.Color(color))
	}

	if len(lipglossColors) >= 2 {
		p.progress = progress.New(
			progress.WithGradient(string(lipglossColors[0]), string(lipglossColors[1])),
		)
		p.progress.Width = p.progress.Width
	}
}

// ファイル検索用プログレスバー
func NewSearchProgress() *Progress {
	p := NewProgress()
	p.SetColors([]string{"#00FFFF", "#0080FF"}) // シアンから青のグラデーション
	p.SetWidth(50)
	return p
}

// プロジェクト分析用プログレスバー
func NewAnalysisProgress() *Progress {
	p := NewProgress()
	p.SetColors([]string{"#FFFF00", "#FF8000"}) // 黄色からオレンジのグラデーション
	p.SetWidth(50)
	return p
}

// LLM処理用プログレスバー
func NewLLMProgress() *Progress {
	p := NewProgress()
	p.SetColors([]string{"#00FF00", "#80FF00"}) // 緑から明るい緑のグラデーション
	p.SetWidth(50)
	return p
}

// ヘルスチェック用プログレスバー
func NewHealthProgress() *Progress {
	p := NewProgress()
	p.SetColors([]string{"#FF69B4", "#FF1493"}) // ピンクから濃いピンクのグラデーション
	p.SetWidth(50)
	return p
}
