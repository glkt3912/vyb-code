package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glkt/vyb-code/internal/config"
)

// メインアプリケーションモデル
type App struct {
	model       AppModel
	theme       *ThemeManager
	initialized bool
}

// 新しいアプリケーションを作成
func NewApp(config config.TUIConfig) *App {
	return &App{
		model: AppModel{
			State:       UIStateIdle,
			CurrentView: ViewMain,
			Config:      config,
			Spinner:     SpinnerModel{spinner: NewSpinner()},
			ProgressBar: ProgressModel{progress: NewProgress()},
		},
		theme:       NewThemeManager(),
		initialized: false,
	}
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	a.initialized = true
	return tea.Batch(
		tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return TickMsg(t)
		}),
		a.model.Spinner.spinner.Init(),
		a.model.ProgressBar.progress.Init(),
	)
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "h":
			// ヘルプ表示
			return a, nil
		case "1":
			a.model.CurrentView = ViewMain
		case "2":
			a.model.CurrentView = ViewConfig
		case "3":
			a.model.CurrentView = ViewHealth
		case "4":
			a.model.CurrentView = ViewSearch
		case "5":
			a.model.CurrentView = ViewAnalyze
		}

	case TickMsg:
		// 定期更新
		cmds = append(cmds, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return TickMsg(t)
		}))

	case ProgressMsg:
		a.model.ProgressVal = float64(msg)
		if a.model.ProgressBar.progress != nil {
			var cmd tea.Cmd
			newModel, cmd := a.model.ProgressBar.progress.Update(msg)
			if progModel, ok := newModel.(*Progress); ok {
				a.model.ProgressBar.progress = progModel
			}
			cmds = append(cmds, cmd)
		}

	case CompleteMsg:
		a.model.State = UIStateCompleted
		a.model.Spinner.Active = false
		a.model.ProgressBar.Active = false

	case ErrorMsg:
		a.model.State = UIStateError
		a.model.Error = msg
		a.model.Spinner.Active = false
		a.model.ProgressBar.Active = false

	case ViewChangeMsg:
		a.model.CurrentView = ViewType(msg)
	}

	// サブコンポーネントの更新
	if a.model.Spinner.Active && a.model.Spinner.spinner != nil {
		var cmd tea.Cmd
		newModel, cmd := a.model.Spinner.spinner.Update(msg)
		if spinnerModel, ok := newModel.(*Spinner); ok {
			a.model.Spinner.spinner = spinnerModel
		}
		cmds = append(cmds, cmd)
	}

	if a.model.ProgressBar.Active && a.model.ProgressBar.progress != nil {
		var cmd tea.Cmd
		newModel, cmd := a.model.ProgressBar.progress.Update(msg)
		if progModel, ok := newModel.(*Progress); ok {
			a.model.ProgressBar.progress = progModel
		}
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

// View implements tea.Model
func (a *App) View() string {
	if !a.initialized {
		return "初期化中..."
	}

	// ヘッダー
	header := a.renderHeader()

	// メインコンテンツ
	content := a.renderContent()

	// フッター
	footer := a.renderFooter()

	// 全体レイアウト
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

// ヘッダーをレンダリング
func (a *App) renderHeader() string {
	theme := a.theme.GetTheme()

	title := a.theme.HeaderStyle().Render("🎵 vyb - Feel the rhythm of perfect code")

	status := ""
	switch a.model.State {
	case UIStateLoading:
		status = a.theme.InfoStyle().Render("読み込み中...")
	case UIStateProcessing:
		status = a.theme.InfoStyle().Render("処理中...")
	case UIStateCompleted:
		status = a.theme.SuccessStyle().Render("完了")
	case UIStateError:
		status = a.theme.ErrorStyle().Render("エラー")
	default:
		status = a.theme.InfoStyle().Render("準備完了")
	}

	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Padding(0, 2).
		MarginBottom(1)

	return headerBox.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			title,
			lipgloss.NewStyle().MarginLeft(4).Render(status),
		),
	)
}

// メインコンテンツをレンダリング
func (a *App) renderContent() string {
	var content string

	switch a.model.CurrentView {
	case ViewMain:
		content = a.renderMainView()
	case ViewConfig:
		content = a.renderConfigView()
	case ViewHealth:
		content = a.renderHealthView()
	case ViewSearch:
		content = a.renderSearchView()
	case ViewAnalyze:
		content = a.renderAnalyzeView()
	default:
		content = a.renderMainView()
	}

	// プログレスバーまたはスピナーを表示
	if a.model.Spinner.Active {
		content += "\n" + a.model.Spinner.spinner.View()
	}

	if a.model.ProgressBar.Active {
		content += "\n" + a.model.ProgressBar.progress.View()
	}

	return content
}

// メインビューをレンダリング
func (a *App) renderMainView() string {
	theme := a.theme.GetTheme()

	menuItems := []string{
		"1. メイン (現在のビュー)",
		"2. 設定",
		"3. ヘルスチェック",
		"4. 検索",
		"5. プロジェクト分析",
	}

	menu := ""
	for _, item := range menuItems {
		menu += a.theme.InfoStyle().Render("  "+item) + "\n"
	}

	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Padding(1, 2).
		Height(10)

	return contentBox.Render(
		a.theme.SubheaderStyle().Render("メニュー") + "\n\n" + menu,
	)
}

// 設定ビューをレンダリング
func (a *App) renderConfigView() string {
	return a.theme.BorderStyle().Render("設定画面（実装予定）")
}

// ヘルスビューをレンダリング
func (a *App) renderHealthView() string {
	return a.theme.BorderStyle().Render("ヘルスチェック画面（実装予定）")
}

// 検索ビューをレンダリング
func (a *App) renderSearchView() string {
	return a.theme.BorderStyle().Render("検索画面（実装予定）")
}

// 分析ビューをレンダリング
func (a *App) renderAnalyzeView() string {
	return a.theme.BorderStyle().Render("プロジェクト分析画面（実装予定）")
}

// フッターをレンダリング
func (a *App) renderFooter() string {
	helpText := a.theme.InfoStyle().Render("q: 終了 | h: ヘルプ | 1-5: ビュー切り替え")

	footerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.GetTheme().Border).
		Padding(0, 2).
		MarginTop(1)

	return footerBox.Render(helpText)
}

// スピナーを開始
func (a *App) StartSpinner(message string) tea.Cmd {
	a.model.State = UIStateLoading
	a.model.Spinner.Active = true
	a.model.Spinner.Message = message
	return a.model.Spinner.spinner.Start(message)
}

// プログレスバーを開始
func (a *App) StartProgress(message string) tea.Cmd {
	a.model.State = UIStateProcessing
	a.model.ProgressBar.Active = true
	a.model.ProgressBar.Message = message
	return a.model.ProgressBar.progress.Start(message)
}

// 進捗を更新
func (a *App) UpdateProgress(percent float64) tea.Cmd {
	a.model.ProgressVal = percent
	return a.model.ProgressBar.progress.SetProgress(percent)
}

// 処理完了
func (a *App) Complete() tea.Cmd {
	return func() tea.Msg { return CompleteMsg{} }
}

// エラー処理
func (a *App) SetError(err error) tea.Cmd {
	return func() tea.Msg { return ErrorMsg(err) }
}
