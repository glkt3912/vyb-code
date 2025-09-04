package input

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/term"
)

// 拡張入力リーダー
type Reader struct {
	history     *History
	completer   *Completer
	isRawMode   bool
	oldState    *term.State
	currentLine string
	cursorPos   int
	prompt      string
}

// 入力履歴管理（既存のInputHistoryを拡張）
type History struct {
	entries  []string
	index    int
	maxSize  int
	tempLine string // 現在の未確定入力を一時保存
}

// オートコンプリート機能
type Completer struct {
	commands    []string
	filePaths   []string
	currentDir  string
	suggestions []string
}

// 特殊キーコード
const (
	KeyUp    = 65  // ↑
	KeyDown  = 66  // ↓
	KeyRight = 67  // →
	KeyLeft  = 68  // ←
	KeyEnter = 13  // Enter
	KeyTab   = 9   // Tab
	KeyESC   = 27  // ESC
	KeyBS    = 127 // Backspace
	KeyDel   = 126 // Delete
	CtrlC    = 3   // Ctrl+C
	CtrlD    = 4   // Ctrl+D
	CtrlL    = 12  // Ctrl+L
)

// 新しいリーダーを作成
func NewReader() *Reader {
	currentDir, _ := os.Getwd()

	return &Reader{
		history:   NewHistory(100),
		completer: NewCompleter(currentDir),
		isRawMode: false,
	}
}

// 履歴を作成
func NewHistory(maxSize int) *History {
	return &History{
		entries: make([]string, 0),
		index:   -1,
		maxSize: maxSize,
	}
}

// オートコンプリートを作成
func NewCompleter(workDir string) *Completer {
	return &Completer{
		commands: []string{
			"/help", "/clear", "/history", "/status", "/info", "/save", "/retry", "/edit",
			"exit", "quit",
		},
		currentDir: workDir,
	}
}

// 履歴に追加
func (h *History) Add(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	// 重複を避ける
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == line {
		h.index = len(h.entries)
		return
	}

	h.entries = append(h.entries, line)
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[1:]
	}
	h.index = len(h.entries)
	h.tempLine = ""
}

// 前の履歴を取得
func (h *History) Previous() string {
	if len(h.entries) == 0 {
		return h.tempLine
	}

	if h.index == len(h.entries) {
		// 現在の入力を一時保存
		h.tempLine = ""
	}

	if h.index > 0 {
		h.index--
	}

	return h.entries[h.index]
}

// 次の履歴を取得
func (h *History) Next() string {
	if len(h.entries) == 0 {
		return h.tempLine
	}

	if h.index < len(h.entries)-1 {
		h.index++
		return h.entries[h.index]
	} else {
		h.index = len(h.entries)
		return h.tempLine
	}
}

// プロンプト設定
func (r *Reader) SetPrompt(prompt string) {
	r.prompt = prompt
}

// Rawモードを有効化
func (r *Reader) enableRawMode() error {
	if r.isRawMode {
		return nil
	}

	// 現在の端末状態を保存
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to enable raw mode: %w", err)
	}

	r.oldState = oldState
	r.isRawMode = true
	return nil
}

// Rawモードを無効化
func (r *Reader) disableRawMode() error {
	if !r.isRawMode {
		return nil
	}

	if r.oldState != nil {
		err := term.Restore(int(os.Stdin.Fd()), r.oldState)
		if err != nil {
			return fmt.Errorf("failed to restore terminal: %w", err)
		}
	}

	r.isRawMode = false
	return nil
}

// 拡張入力読み込み（矢印キー・補完対応）
func (r *Reader) ReadLine() (string, error) {
	// プロンプト表示
	fmt.Print(r.prompt)

	// Raw mode が利用可能かチェック
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return r.readLineRaw()
	}

	// フォールバック：通常の入力
	return r.readLineFallback()
}

// Raw mode での高度な入力処理
func (r *Reader) readLineRaw() (string, error) {
	if err := r.enableRawMode(); err != nil {
		return r.readLineFallback()
	}
	defer r.disableRawMode()

	r.currentLine = ""
	r.cursorPos = 0

	buffer := make([]byte, 1)

	for {
		n, err := os.Stdin.Read(buffer)
		if err == io.EOF {
			return r.currentLine, err
		}
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}

		b := buffer[0]

		switch b {
		case KeyEnter:
			// 入力確定
			r.clearCurrentLine()
			fmt.Printf("%s\n", r.currentLine)
			line := r.currentLine
			r.history.Add(line)
			return line, nil

		case CtrlC:
			// Ctrl+C: キャンセル
			r.clearCurrentLine()
			fmt.Println("^C")
			return "", fmt.Errorf("interrupted")

		case CtrlD:
			// Ctrl+D: EOF
			if r.currentLine == "" {
				r.clearCurrentLine()
				return "", io.EOF
			}

		case CtrlL:
			// Ctrl+L: 画面クリア
			fmt.Print("\033[2J\033[H")
			r.redrawLine()

		case KeyTab:
			// Tab: オートコンプリート
			suggestions := r.completer.GetSuggestions(r.currentLine)
			if len(suggestions) == 1 {
				r.currentLine = suggestions[0]
				r.cursorPos = len(r.currentLine)
				r.redrawLine()
			} else if len(suggestions) > 1 {
				r.showSuggestions(suggestions)
				r.redrawLine()
			}

		case KeyBS:
			// Backspace
			if r.cursorPos > 0 {
				r.currentLine = r.currentLine[:r.cursorPos-1] + r.currentLine[r.cursorPos:]
				r.cursorPos--
				r.redrawLine()
			}

		case KeyESC:
			// エスケープシーケンス処理
			if err := r.handleEscapeSequence(); err != nil {
				continue
			}

		default:
			// 通常文字
			if b >= 32 && b <= 126 {
				// ASCII印字可能文字
				r.currentLine = r.currentLine[:r.cursorPos] + string(b) + r.currentLine[r.cursorPos:]
				r.cursorPos++
				r.redrawLine()
			} else if b >= 128 {
				// UTF-8マルチバイト文字の開始
				if err := r.handleUTF8Input(b); err != nil {
					continue
				}
			}
		}
	}
}

// エスケープシーケンスを処理（矢印キーなど）
func (r *Reader) handleEscapeSequence() error {
	// エスケープシーケンスの続きを読み取り
	buffer := make([]byte, 2)
	n, err := os.Stdin.Read(buffer)
	if err != nil || n < 2 {
		return err
	}

	// [ + キーコード のパターン
	if buffer[0] == '[' {
		switch buffer[1] {
		case KeyUp:
			// 上矢印: 履歴を戻る
			if prev := r.history.Previous(); prev != r.currentLine {
				r.currentLine = prev
				r.cursorPos = len(r.currentLine)
				r.redrawLine()
			}

		case KeyDown:
			// 下矢印: 履歴を進む
			if next := r.history.Next(); next != r.currentLine {
				r.currentLine = next
				r.cursorPos = len(r.currentLine)
				r.redrawLine()
			}

		case KeyLeft:
			// 左矢印: カーソルを左に移動
			if r.cursorPos > 0 {
				r.cursorPos--
				fmt.Print("\033[D") // カーソルを1文字左に移動
			}

		case KeyRight:
			// 右矢印: カーソルを右に移動
			if r.cursorPos < len(r.currentLine) {
				r.cursorPos++
				fmt.Print("\033[C") // カーソルを1文字右に移動
			}
		}
	}

	return nil
}

// UTF-8入力を処理
func (r *Reader) handleUTF8Input(firstByte byte) error {
	// UTF-8文字の残りバイトを読み取り
	var utf8Bytes []byte
	utf8Bytes = append(utf8Bytes, firstByte)

	// UTF-8シーケンス長を判定
	var expectedLen int
	if firstByte&0x80 == 0 {
		expectedLen = 1
	} else if firstByte&0xE0 == 0xC0 {
		expectedLen = 2
	} else if firstByte&0xF0 == 0xE0 {
		expectedLen = 3
	} else if firstByte&0xF8 == 0xF0 {
		expectedLen = 4
	} else {
		return fmt.Errorf("invalid UTF-8 start byte")
	}

	// 残りのバイトを読み取り
	if expectedLen > 1 {
		buffer := make([]byte, expectedLen-1)
		n, err := os.Stdin.Read(buffer)
		if err != nil || n != expectedLen-1 {
			return fmt.Errorf("incomplete UTF-8 sequence")
		}
		utf8Bytes = append(utf8Bytes, buffer...)
	}

	// UTF-8文字列に変換
	char := string(utf8Bytes)

	// 現在の行に文字を挿入
	r.currentLine = r.currentLine[:r.cursorPos] + char + r.currentLine[r.cursorPos:]
	r.cursorPos += len(char)
	r.redrawLine()

	return nil
}

// 現在の行を再描画
func (r *Reader) redrawLine() {
	// カーソルを行頭に移動
	fmt.Print("\033[G")

	// プロンプトと現在の行を表示
	fmt.Printf("%s%s", r.prompt, r.currentLine)

	// 行末まで削除
	fmt.Print("\033[K")

	// カーソルを正しい位置に移動
	if r.cursorPos < len(r.currentLine) {
		// カーソルが行末でない場合、正しい位置に移動
		fmt.Printf("\033[%dG", len(r.prompt)+r.cursorPos+1)
	}
}

// 現在の行をクリア
func (r *Reader) clearCurrentLine() {
	fmt.Print("\033[G\033[K")
}

// 補完候補を表示
func (r *Reader) showSuggestions(suggestions []string) {
	// 現在の行を保存
	fmt.Print("\033[s")

	// 新しい行に移動
	fmt.Print("\n")

	// 候補を表示
	gray := "\033[90m"
	green := "\033[32m"
	reset := "\033[0m"

	fmt.Printf("%s候補:%s", gray, reset)
	for i, suggestion := range suggestions {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%s%s%s", green, suggestion, reset)
		if i > 5 { // 最大6個まで表示
			fmt.Printf(" %s...%s", gray, reset)
			break
		}
	}
	fmt.Print("\n")

	// 元の位置に戻る
	fmt.Print("\033[u")
}

// フォールバック：通常の入力処理
func (r *Reader) readLineFallback() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	line = strings.TrimSuffix(line, "\n")
	r.history.Add(line)
	return line, nil
}

// オートコンプリート候補を取得
func (c *Completer) GetSuggestions(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return []string{}
	}

	var suggestions []string

	// スラッシュコマンド補完
	if strings.HasPrefix(input, "/") {
		for _, cmd := range c.commands {
			if strings.HasPrefix(cmd, input) {
				suggestions = append(suggestions, cmd)
			}
		}
		return suggestions
	}

	// ファイルパス補完
	if strings.Contains(input, "/") || strings.Contains(input, ".") {
		return c.getFilePathSuggestions(input)
	}

	// 一般的なコマンド補完
	commonCommands := []string{"help", "analyze", "build", "test", "status"}
	for _, cmd := range commonCommands {
		if strings.HasPrefix(cmd, input) {
			suggestions = append(suggestions, cmd)
		}
	}

	return suggestions
}

// ファイルパス補完候補を取得
func (c *Completer) getFilePathSuggestions(input string) []string {
	var suggestions []string

	// 入力されたパスの親ディレクトリを取得
	dir := filepath.Dir(input)
	if dir == "." {
		dir = c.currentDir
	}

	// ディレクトリの内容を読み取り
	entries, err := os.ReadDir(dir)
	if err != nil {
		return suggestions
	}

	prefix := filepath.Base(input)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			fullPath := filepath.Join(dir, entry.Name())
			if entry.IsDir() {
				suggestions = append(suggestions, fullPath+"/")
			} else {
				suggestions = append(suggestions, fullPath)
			}
		}
	}

	return suggestions
}

// マルチライン入力の読み取り
func (r *Reader) ReadMultiLine() (string, error) {
	var lines []string

	fmt.Printf("マルチライン入力モード (空行で送信, Ctrl+C でキャンセル):\n")

	for {
		// 継続プロンプト
		if len(lines) > 0 {
			r.SetPrompt("... ")
		}

		line, err := r.ReadLine()
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "interrupted") {
				return "", err
			}
			continue
		}

		// 空行で送信完了
		if strings.TrimSpace(line) == "" {
			if len(lines) > 0 {
				return strings.Join(lines, "\n"), nil
			}
			continue
		}

		lines = append(lines, line)
	}
}

// 端末サイズを取得
func getTerminalSize() (int, int) {
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	ws := &winsize{}
	ret, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(os.Stdout.Fd()),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(ret) == -1 {
		return 80, 24 // デフォルト値
	}

	return int(ws.Col), int(ws.Row)
}

// リーダーのクリーンアップ
func (r *Reader) Close() error {
	return r.disableRawMode()
}
