package input

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/term"
)

// 拡張入力リーダー
type Reader struct {
	history            *History
	completer          *Completer
	securityValidator  *SecurityValidator    // セキュリティ機能
	perfOptimizer      *PerformanceOptimizer // パフォーマンス最適化
	isRawMode          bool
	oldState           *term.State
	currentLine        string
	cursorPos          int
	prompt             string
	clientID           string // セキュリティ用のクライアントID
	enableOptimization bool   // パフォーマンス最適化の有効/無効
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
	commands          []string
	filePaths         []string
	currentDir        string
	suggestions       []string
	advancedCompleter *AdvancedCompleter // 高度な補完機能
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

	perfOptimizer := NewPerformanceOptimizer()
	perfOptimizer.Start() // パフォーマンス最適化を開始

	return &Reader{
		history:            NewHistory(100),
		completer:          NewCompleter(currentDir),
		securityValidator:  NewSecurityValidator(),
		perfOptimizer:      perfOptimizer,
		isRawMode:          false,
		clientID:           "local", // ローカル実行用のデフォルトID
		enableOptimization: true,    // デフォルトで有効
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
		currentDir:        workDir,
		advancedCompleter: NewAdvancedCompleter(workDir),
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

			// セキュリティ検証を実行
			sanitizedLine, err := r.securityValidator.ValidateInput(line, r.clientID)
			if err != nil {
				// セキュリティエラーの場合は警告表示して再入力
				fmt.Printf("\033[31m警告: %s\033[0m\n", err.Error())
				r.currentLine = ""
				r.cursorPos = 0
				r.redrawLine()
				continue
			}

			r.history.Add(sanitizedLine)
			return sanitizedLine, nil

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
			// Tab: 高度なオートコンプリート（パフォーマンス最適化付き）
			var candidates []CompletionCandidate

			if r.enableOptimization && r.perfOptimizer != nil {
				// パフォーマンス最適化版を使用
				candidates = r.perfOptimizer.OptimizedCompletion(r.currentLine, r.completer.advancedCompleter)
			} else {
				// 通常版を使用
				candidates = r.completer.advancedCompleter.GetAdvancedSuggestions(r.currentLine)
			}

			if len(candidates) == 1 {
				r.currentLine = candidates[0].Text
				r.cursorPos = len(r.currentLine)
				r.redrawLine()
			} else if len(candidates) > 1 {
				r.showAdvancedSuggestions(candidates)
				r.redrawLine()
			}

		case KeyBS, 8: // Backspace (127 or 8)
			// Backspace: カーソルの左の文字を削除（UTF-8対応）
			if r.cursorPos > 0 {
				runes := []rune(r.currentLine)
				if r.cursorPos <= len(runes) {
					// ルーン（文字）単位で削除
					newRunes := append(runes[:r.cursorPos-1], runes[r.cursorPos:]...)
					r.currentLine = string(newRunes)
					r.cursorPos--
					r.redrawLine()
				}
			}

		case KeyESC:
			// エスケープシーケンス処理
			if err := r.handleEscapeSequence(); err != nil {
				continue
			}

		default:
			// 通常文字
			if b >= 32 && b <= 126 {
				// ASCII印字可能文字 - ルーン単位で挿入
				runes := []rune(r.currentLine)
				newRunes := append(runes[:r.cursorPos], append([]rune{rune(b)}, runes[r.cursorPos:]...)...)
				newLine := string(newRunes)
				if len(newLine) > MaxLineLength {
					// 長すぎる場合は無視
					continue
				}
				r.currentLine = newLine
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
				r.cursorPos = len([]rune(r.currentLine)) // ルーン単位でカーソル位置設定
				r.redrawLine()
			}

		case KeyDown:
			// 下矢印: 履歴を進む
			if next := r.history.Next(); next != r.currentLine {
				r.currentLine = next
				r.cursorPos = len([]rune(r.currentLine)) // ルーン単位でカーソル位置設定
				r.redrawLine()
			}

		case KeyLeft:
			// 左矢印: カーソルを左に移動（UTF-8対応）
			if r.cursorPos > 0 {
				r.cursorPos--
				r.redrawLine() // 完全な再描画でカーソル位置を同期
			}

		case KeyRight:
			// 右矢印: カーソルを右に移動（UTF-8対応）
			runes := []rune(r.currentLine)
			if r.cursorPos < len(runes) {
				r.cursorPos++
				r.redrawLine() // 完全な再描画でカーソル位置を同期
			}

		case '3':
			// Delete key: [3~ シーケンス - より安全な処理
			buffer2 := make([]byte, 1)
			n2, err2 := os.Stdin.Read(buffer2)
			if err2 != nil {
				// 読み取りエラーの場合は静かに無視
				return nil
			}
			if n2 == 1 && buffer2[0] == '~' {
				// Delete: カーソル位置の文字を削除（UTF-8対応）
				r.performDelete()
			}
			// それ以外の場合は無視（Deleteキーでない）
		}
	}

	return nil
}

// Delete操作を実行（共通化）
func (r *Reader) performDelete() {
	runes := []rune(r.currentLine)
	if r.cursorPos < len(runes) {
		// ルーン（文字）単位で削除
		newRunes := append(runes[:r.cursorPos], runes[r.cursorPos+1:]...)
		r.currentLine = string(newRunes)
		// カーソル位置はそのまま（削除した文字の次にカーソルが移動する）
		r.redrawLine()
	}
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

	// UTF-8文字をルーン単位で挿入
	runes := []rune(r.currentLine)
	charRunes := []rune(char)
	newRunes := append(runes[:r.cursorPos], append(charRunes, runes[r.cursorPos:]...)...)
	newLine := string(newRunes)

	if len(newLine) > MaxLineLength {
		return fmt.Errorf("行長制限を超えています")
	}

	// 現在の行に文字を挿入
	r.currentLine = newLine
	r.cursorPos += len(charRunes) // ルーン数で更新
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

	// カーソルを正しい位置に移動 (UTF-8対応)
	runes := []rune(r.currentLine)
	if r.cursorPos < len(runes) {
		// UTF-8文字の実際の表示幅を計算
		displayWidth := r.calculateDisplayWidth(runes[:r.cursorPos])
		fmt.Printf("\033[%dG", len(r.prompt)+displayWidth+1)
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

// 高度な補完候補を表示（詳細情報付き）
func (r *Reader) showAdvancedSuggestions(candidates []CompletionCandidate) {
	// 現在の行を保存
	fmt.Print("\033[s")

	// 新しい行に移動
	fmt.Print("\n")

	// カラーコード
	gray := "\033[90m"
	green := "\033[32m"
	blue := "\033[34m"
	yellow := "\033[33m"
	cyan := "\033[36m"
	reset := "\033[0m"

	fmt.Printf("%s候補:%s\n", gray, reset)

	maxDisplay := 6
	if len(candidates) > maxDisplay {
		candidates = candidates[:maxDisplay]
	}

	for i, candidate := range candidates {
		// 補完タイプによる色分け
		var typeColor string
		var typeIcon string

		switch candidate.Type {
		case CompletionFile:
			typeColor = green
			typeIcon = "📄"
		case CompletionCommand:
			typeColor = blue
			typeIcon = "⚡"
		case CompletionGitBranch:
			typeColor = yellow
			typeIcon = "🌿"
		case CompletionGitFile:
			typeColor = cyan
			typeIcon = "📝"
		case CompletionProjectCommand:
			typeColor = "\033[35m" // マゼンタ
			typeIcon = "🔧"
		default:
			typeColor = gray
			typeIcon = "💡"
		}

		// スコア表示（デバッグ用、実際は非表示推奨）
		scoreStr := ""
		if candidate.Score > 0 {
			scoreStr = fmt.Sprintf(" %s(%.2f)%s", gray, candidate.Score, reset)
		}

		fmt.Printf("  %s%s %s%s%s %s%s%s%s\n",
			typeColor, typeIcon, candidate.Text, reset,
			scoreStr,
			gray, candidate.Description, reset, gray)

		if i >= maxDisplay-1 && len(candidates) > maxDisplay {
			fmt.Printf("  %s... (%d more)%s\n", gray, len(candidates)-maxDisplay, reset)
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

	// セキュリティ検証を実行
	sanitizedLine, err := r.securityValidator.ValidateInput(line, r.clientID)
	if err != nil {
		// セキュリティエラーの場合は警告表示
		fmt.Printf("\033[31m警告: %s\033[0m\n", err.Error())
		return "", err
	}

	r.history.Add(sanitizedLine)
	return sanitizedLine, nil
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

// getTerminalSize は reader_unix.go と reader_windows.go で実装

// クライアントIDを設定（セキュリティ用）
func (r *Reader) SetClientID(clientID string) {
	r.clientID = clientID
}

// セキュリティ設定を更新
func (r *Reader) UpdateSecuritySettings(bufferLimit int, requestsPerSec int) {
	if r.securityValidator != nil {
		r.securityValidator.UpdateSecuritySettings(bufferLimit, requestsPerSec)
	}
}

// セキュリティ機能を無効化（テスト用など）
func (r *Reader) DisableSecurity() {
	r.securityValidator = nil
}

// セキュリティ機能を有効化
func (r *Reader) EnableSecurity() {
	if r.securityValidator == nil {
		r.securityValidator = NewSecurityValidator()
	}
}

// パフォーマンス最適化を有効化
func (r *Reader) EnableOptimization() {
	r.enableOptimization = true
	if r.perfOptimizer == nil {
		r.perfOptimizer = NewPerformanceOptimizer()
		r.perfOptimizer.Start()
	}
}

// パフォーマンス最適化を無効化
func (r *Reader) DisableOptimization() {
	r.enableOptimization = false
}

// パフォーマンスメトリクスを取得
func (r *Reader) GetPerformanceMetrics() map[string]interface{} {
	if r.perfOptimizer != nil && r.perfOptimizer.metricsCollector != nil {
		return r.perfOptimizer.metricsCollector.GetMetrics()
	}
	return map[string]interface{}{}
}

// UTF-8文字の表示幅を計算（東アジア文字対応）
func (r *Reader) calculateDisplayWidth(runes []rune) int {
	width := 0
	for _, r := range runes {
		if r < 32 || r == 127 {
			// 制御文字は幅0
			continue
		} else if r < 127 {
			// ASCII文字は幅1
			width++
		} else if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
			// CJK文字（中国語、日本語、韓国語）は幅2
			width += 2
		} else if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || unicode.Is(unicode.Mc, r) {
			// 結合文字は幅0
			continue
		} else {
			// その他のUnicode文字は幅1
			width++
		}
	}
	return width
}

// リーダーのクリーンアップ
func (r *Reader) Close() error {
	// パフォーマンス最適化システムを停止
	if r.perfOptimizer != nil {
		r.perfOptimizer.Stop()
	}

	return r.disableRawMode()
}
