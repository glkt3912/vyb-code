package ui

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glkt/vyb-code/internal/chat"
	"github.com/glkt/vyb-code/internal/config"
)

// チャット用TUIアプリ
type ChatApp struct {
	config       config.TUIConfig
	theme        *ThemeManager
	session      *chat.Session
	viewport     viewport.Model
	textarea     textarea.Model
	messages     []ChatMessage
	loading      bool
	err          error
	ready        bool
	statusMsg    string // 一時的なステータスメッセージ
	statusTimer  *time.Timer // ステータスメッセージタイマー
}

// チャットメッセージ
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// チャット応答メッセージ
type ChatResponseMsg struct {
	Content string
	Error   error
}

// コピー完了メッセージ
type CopyCompleteMsg struct {
	Success bool
	Message string
}

// ステータス消去メッセージ
type ClearStatusMsg struct{}


// 新しいチャットアプリを作成
func NewChatApp(cfg config.TUIConfig, session *chat.Session) *ChatApp {
	// ビューポート設定
	vp := viewport.New(80, 20)
	vp.YPosition = 0

	// テキストエリア設定
	ta := textarea.New()
	ta.Placeholder = "メッセージを入力... (Tab/Ctrl+S/F1: 送信 / Enter: 改行)"
	ta.Focus()
	ta.Prompt = "> "
	ta.CharLimit = 2000
	ta.SetWidth(80)
	ta.SetHeight(2)
	ta.ShowLineNumbers = false
	// シンプルな設定：Enterで改行を許可
	ta.KeyMap.InsertNewline.SetEnabled(true)

	return &ChatApp{
		config:   cfg,
		theme:    NewThemeManager(),
		session:  session,
		viewport: vp,
		textarea: ta,
		messages: []ChatMessage{
			{
				Role:    "assistant",
				Content: "🎵 vyb へようこそ！何かお手伝いできることはありますか？",
			},
		},
		loading: false,
		ready:   false,
	}
}

// Init implements tea.Model
func (c *ChatApp) Init() tea.Cmd {
	// 手動で画面をクリア（Alternative Screen無しでも美しい表示）
	fmt.Print("\033[H\033[2J")
	
	return tea.Batch(
		textarea.Blink,
		c.updateViewport(),
	)
}

// Update implements tea.Model
func (c *ChatApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// ターミナルサイズに基づいてコンポーネントサイズを調整
		headerHeight := 2
		helpHeight := 1
		inputHeight := 3
		
		// ビューポートサイズを計算（余白を考慮）
		c.viewport.Width = msg.Width - 4
		c.viewport.Height = msg.Height - headerHeight - helpHeight - inputHeight - 2
		
		// テキストエリアの幅を調整
		c.textarea.SetWidth(msg.Width - 4)
		
		c.ready = true
		return c, c.updateViewport()

	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlC:
			// 終了時にカーソルを復元
			fmt.Print("\033[?25h") // カーソル表示
			return c, tea.Quit
		case msg.Type == tea.KeyTab:
			// Tab でメッセージ送信（確実に動作）
			if !c.loading && c.textarea.Value() != "" {
				return c, c.sendMessage()
			}
			return c, nil
		case msg.Type == tea.KeyCtrlS:
			// Ctrl+S でメッセージ送信（安全）
			if !c.loading && c.textarea.Value() != "" {
				return c, c.sendMessage()
			}
			return c, nil
		case msg.Type == tea.KeyF1:
			// F1 でメッセージ送信（確実）
			if !c.loading && c.textarea.Value() != "" {
				return c, c.sendMessage()
			}
			return c, nil
		case msg.String() == "ctrl+e":
			// Ctrl+E で会話履歴をプレーンテキスト出力
			return c, c.exportChat()
		case msg.String() == "ctrl+p":
			// Ctrl+P で会話履歴を標準出力に出力
			return c, c.printChat()
		case msg.Type == tea.KeyF2:
			// F2 で全履歴をクリップボードにコピー
			return c, c.copyAllToClipboard()
		case msg.String() == "ctrl+y":
			// Ctrl+Y で最後のレスポンスをクリップボードにコピー
			return c, c.copyLastResponseToClipboard()
		case msg.Type == tea.KeyEnter:
			// 通常のEnterは改行（textareaのデフォルト動作）
			if !c.loading {
				var cmd tea.Cmd
				c.textarea, cmd = c.textarea.Update(msg)
				return c, cmd
			}
		case msg.String() == "shift+up":
			// Shift+上矢印で3行上にスクロール
			c.viewport.LineUp(3)
			return c, nil
		case msg.String() == "shift+down":
			// Shift+下矢印で3行下にスクロール
			c.viewport.LineDown(3)
			return c, nil
		case msg.Type == tea.KeyPgUp:
			// Page Upでフルページ上にスクロール
			c.viewport.ViewUp()
			return c, nil
		case msg.Type == tea.KeyPgDown:
			// Page Downでフルページ下にスクロール
			c.viewport.ViewDown()
			return c, nil
		case msg.String() == "ctrl+u":
			// Ctrl+U で大きく上にスクロール（画面の3/4）
			scrollLines := (c.viewport.Height * 3) / 4
			if scrollLines < 5 {
				scrollLines = 5
			}
			c.viewport.LineUp(scrollLines)
			return c, nil
		case msg.String() == "ctrl+d":
			// Ctrl+D で大きく下にスクロール（画面の3/4）
			scrollLines := (c.viewport.Height * 3) / 4
			if scrollLines < 5 {
				scrollLines = 5
			}
			c.viewport.LineDown(scrollLines)
			return c, nil
		case msg.String() == "ctrl+home":
			// Ctrl+Home で一番上にスクロール
			c.viewport.GotoTop()
			return c, nil
		case msg.String() == "ctrl+end":
			// Ctrl+End で一番下にスクロール
			c.viewport.GotoBottom()
			return c, nil
		case msg.Type == tea.KeyEsc:
			if c.loading {
				// 処理をキャンセル（今回は簡略化）
				c.loading = false
			}
		default:
			// その他のキー入力はテキストエリアに渡す
			if !c.loading {
				var cmd tea.Cmd
				c.textarea, cmd = c.textarea.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp:
			// マウスホイール上で5行スクロール
			c.viewport.LineUp(5)
			return c, nil
		case tea.MouseWheelDown:
			// マウスホイール下で5行スクロール
			c.viewport.LineDown(5)
			return c, nil
		case tea.MouseRight:
			// 右クリックで最後のレスポンスをクリップボードにコピー
			return c, c.copyLastResponseToClipboard()
		}

	case ChatResponseMsg:
		c.loading = false
		if msg.Error != nil {
			c.err = msg.Error
			c.messages = append(c.messages, ChatMessage{
				Role:    "assistant",
				Content: "エラー: " + msg.Error.Error(),
			})
		} else {
			c.messages = append(c.messages, ChatMessage{
				Role:    "assistant",
				Content: msg.Content,
			})
		}
		cmds = append(cmds, c.updateViewport())

	case CopyCompleteMsg:
		// コピー完了メッセージを表示
		c.statusMsg = msg.Message
		// 3秒後にステータスを消去
		return c, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return ClearStatusMsg{}
		})

	case ClearStatusMsg:
		// ステータスメッセージをクリア
		c.statusMsg = ""

	default:
		// その他のメッセージはviewportに渡す
		var cmd tea.Cmd
		c.viewport, cmd = c.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return c, tea.Batch(cmds...)
}

// View implements tea.Model
func (c *ChatApp) View() string {
	if !c.ready {
		return "初期化中..."
	}

	// 計算されたサイズ
	headerHeight := 2
	helpHeight := 1
	inputHeight := 3
	
	// 利用可能な高さを計算
	availableHeight := c.viewport.Height + headerHeight + helpHeight + inputHeight
	contentHeight := availableHeight - headerHeight - helpHeight - inputHeight

	// ヘッダー（固定）
	header := lipgloss.NewStyle().
		Height(headerHeight).
		Render(c.theme.HeaderStyle().Render("🎵 vyb - AI コーディングアシスタント"))

	// メッセージビュー（スクロール可能）
	messagesView := lipgloss.NewStyle().
		Height(contentHeight).
		Render(c.viewport.View())

	// 入力エリア（固定）
	var inputArea string
	if c.loading {
		inputArea = lipgloss.NewStyle().
			Height(inputHeight).
			Render(c.theme.InfoStyle().Render("🤖 考え中..."))
	} else {
		inputArea = lipgloss.NewStyle().
			Height(inputHeight).
			Render(c.textarea.View())
	}

	// ステータスメッセージまたはヘルプテキスト（固定）
	var helpText string
	if c.statusMsg != "" {
		helpText = c.theme.SuccessStyle().Render(c.statusMsg)
	} else {
		helpText = c.theme.InfoStyle().Render("Tab/F1: 送信 | Enter: 改行 | F2: 全履歴コピー | 右クリック/Ctrl+Y: 最新回答コピー | Ctrl+C: 終了")
	}
	
	help := lipgloss.NewStyle().
		Height(helpHeight).
		Render(helpText)

	// フレックスレイアウト
	return lipgloss.NewStyle().
		Width(c.viewport.Width).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				header,
				messagesView,
				inputArea,
				help,
			),
		)
}

// メッセージを送信
func (c *ChatApp) sendMessage() tea.Cmd {
	userMessage := strings.TrimSpace(c.textarea.Value())
	if userMessage == "" {
		return nil
	}

	// ユーザーメッセージを追加
	c.messages = append(c.messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	// 入力をクリア
	c.textarea.Reset()
	c.loading = true

	// LLM応答を取得（非同期）
	return tea.Cmd(func() tea.Msg {
		response, err := c.session.SendMessage(userMessage)
		return ChatResponseMsg{
			Content: response,
			Error:   err,
		}
	})
}

// テキストを指定幅で折り返し（UTF-8対応）
func (c *ChatApp) wrapText(text string, width int) string {
	if width <= 10 {
		return text
	}

	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if utf8.RuneCountInString(line) <= width {
			lines = append(lines, line)
			continue
		}

		// 長い行を文字単位で分割
		runes := []rune(line)
		for len(runes) > width {
			lines = append(lines, string(runes[:width]))
			runes = runes[width:]
		}
		if len(runes) > 0 {
			lines = append(lines, string(runes))
		}
	}
	return strings.Join(lines, "\n")
}

// ビューポートを更新
func (c *ChatApp) updateViewport() tea.Cmd {
	var content strings.Builder
	
	// メッセージエリアの幅（余白を考慮）
	messageWidth := c.viewport.Width - 4
	if messageWidth < 40 {
		messageWidth = 40 // 最小幅を確保
	}

	for i, msg := range c.messages {
		var style lipgloss.Style
		var prefix string

		if msg.Role == "user" {
			style = c.theme.InfoStyle()
			prefix = "👤 あなた: "
		} else {
			style = c.theme.AccentStyle()
			prefix = "🤖 vyb: "
		}

		// プレフィックス部分
		content.WriteString(style.Render(prefix) + "\n")
		
		// メッセージ内容を折り返し処理（プレフィックス分をインデント）
		wrappedContent := c.wrapText(msg.Content, messageWidth-4)
		
		// 各行にインデントを追加
		indentedLines := []string{}
		for _, line := range strings.Split(wrappedContent, "\n") {
			indentedLines = append(indentedLines, "  "+line)
		}
		
		content.WriteString(strings.Join(indentedLines, "\n"))
		
		// 最後のメッセージ以外は改行を2つ追加
		if i < len(c.messages)-1 {
			content.WriteString("\n\n")
		} else {
			// 最後のメッセージは1つの改行のみ（途切れ防止）
			content.WriteString("\n")
		}
	}

	// 新しいメッセージが追加された時だけ一番下に移動
	currentContent := c.viewport.View()
	newContent := content.String()
	
	c.viewport.SetContent(newContent)
	
	// コンテンツが変更された場合のみ最下部に移動
	if currentContent != newContent {
		c.viewport.GotoBottom()
	}
	
	return nil
}

// 会話履歴をファイルにエクスポート
func (c *ChatApp) exportChat() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filename := fmt.Sprintf("vyb-chat_%s.txt", timestamp)
		
		var content strings.Builder
		content.WriteString(fmt.Sprintf("vyb Chat Export - %s\n", time.Now().Format("2006-01-02 15:04:05")))
		content.WriteString(strings.Repeat("=", 50) + "\n\n")
		
		for _, msg := range c.messages {
			if msg.Role == "user" {
				content.WriteString("👤 あなた:\n")
			} else {
				content.WriteString("🤖 vyb:\n")
			}
			content.WriteString(msg.Content + "\n\n")
		}
		
		if err := os.WriteFile(filename, []byte(content.String()), 0644); err == nil {
			c.messages = append(c.messages, ChatMessage{
				Role:    "system",
				Content: fmt.Sprintf("会話履歴を %s にエクスポートしました", filename),
			})
		}
		
		return CompleteMsg{}
	})
}

// 会話履歴を標準出力に出力
func (c *ChatApp) printChat() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("vyb Chat History")
		fmt.Println(strings.Repeat("=", 50))
		
		for _, msg := range c.messages {
			if msg.Role == "user" {
				fmt.Println("👤 あなた:")
			} else {
				fmt.Println("🤖 vyb:")
			}
			fmt.Println(msg.Content)
			fmt.Println()
		}
		
		fmt.Println(strings.Repeat("=", 50))
		
		return CompleteMsg{}
	})
}

// 全履歴をクリップボードにコピー
func (c *ChatApp) copyAllToClipboard() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		var content strings.Builder
		
		for _, msg := range c.messages {
			if msg.Role == "user" {
				content.WriteString("あなた: " + msg.Content + "\n\n")
			} else {
				content.WriteString("vyb: " + msg.Content + "\n\n")
			}
		}
		
		err := clipboard.WriteAll(content.String())
		if err != nil {
			return CopyCompleteMsg{
				Success: false,
				Message: "❌ コピー失敗: " + err.Error(),
			}
		}
		
		return CopyCompleteMsg{
			Success: true,
			Message: "📋 全履歴をクリップボードにコピーしました",
		}
	})
}

// 最後のレスポンスをクリップボードにコピー
func (c *ChatApp) copyLastResponseToClipboard() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		// 最後のアシスタントメッセージを探す
		for i := len(c.messages) - 1; i >= 0; i-- {
			if c.messages[i].Role == "assistant" {
				err := clipboard.WriteAll(c.messages[i].Content)
				if err != nil {
					return CopyCompleteMsg{
						Success: false,
						Message: "❌ コピー失敗: " + err.Error(),
					}
				}
				
				return CopyCompleteMsg{
					Success: true,
					Message: "📋 最新回答をクリップボードにコピーしました",
				}
			}
		}
		
		return CopyCompleteMsg{
			Success: false,
			Message: "❌ コピー対象が見つかりません",
		}
	})
}

// コピー用に履歴を出力（選択しやすい形式）- 後方互換性のため保持
func (c *ChatApp) printChatForCopy() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		fmt.Println() // 空行
		
		for _, msg := range c.messages {
			if msg.Role == "user" {
				fmt.Printf("あなた: %s\n", msg.Content)
			} else {
				fmt.Printf("vyb: %s\n", msg.Content)
			}
			fmt.Println()
		}
		
		return CompleteMsg{}
	})
}

// 最後のレスポンスのみをコピー用に出力 - 後方互換性のため保持
func (c *ChatApp) printLastResponse() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		// 最後のアシスタントメッセージを探す
		for i := len(c.messages) - 1; i >= 0; i-- {
			if c.messages[i].Role == "assistant" {
				fmt.Println() // 空行
				fmt.Println(c.messages[i].Content)
				fmt.Println() // 空行
				break
			}
		}
		
		return CompleteMsg{}
	})
}