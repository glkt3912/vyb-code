package markdown

import (
	"fmt"
	"golang.org/x/term"
	"os"
	"regexp"
	"strings"
)

// ANSIカラーコード定数
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	// 色
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[90m"
	White   = "\033[97m"

	// 背景色
	BgBlack = "\033[40m"
	BgGray  = "\033[100m"
)

// Markdown レンダラー
type Renderer struct {
	config RenderConfig
}

// レンダリング設定
type RenderConfig struct {
	EnableColors      bool
	EnableAnimations  bool
	CodeBlockStyle    string // "bordered", "simple"
	TableStyle        string // "box", "simple"
	IndentSize        int
	MaxTableWidth     int
	UseUnicodeSymbols bool
	AutoWidth         bool // ターミナル幅に自動調整
}

// デフォルト設定でレンダラーを作成
func NewRenderer() *Renderer {
	return &Renderer{
		config: RenderConfig{
			EnableColors:      true,
			EnableAnimations:  true,
			CodeBlockStyle:    "bordered",
			TableStyle:        "box",
			IndentSize:        2,
			MaxTableWidth:     80,
			UseUnicodeSymbols: true,
			AutoWidth:         true,
		},
	}
}

// 設定付きレンダラーを作成
func NewRendererWithConfig(config RenderConfig) *Renderer {
	return &Renderer{config: config}
}

// ターミナル幅を取得（リアルタイム）
func (r *Renderer) getTerminalWidth() int {
	if !r.config.AutoWidth {
		return 80 // デフォルト幅
	}

	// 標準出力がターミナルかチェック
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return 80 // パイプや非ターミナル環境では固定幅
	}

	// 毎回リアルタイムでターミナル幅を取得
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // エラー時はデフォルト幅
	}

	// 最小・最大幅を設定
	if width < 40 {
		width = 40
	}
	if width > 120 {
		width = 120
	}

	return width
}

// メインレンダリング関数
func (r *Renderer) Render(content string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder
	inCodeBlock := false
	codeLanguage := ""
	inTable := false
	tableHeaders := []string{}
	tableRows := [][]string{}
	codeBlockLines := []string{}
	maxCodeWidth := 0

	for _, line := range lines {
		// 空行処理
		if strings.TrimSpace(line) == "" && !inCodeBlock {
			result.WriteString("\n")
			continue
		}

		// コードブロック処理
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if !inCodeBlock {
				codeLanguage = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "```"))
				inCodeBlock = true
				codeBlockLines = []string{}
				maxCodeWidth = 0
			} else {
				inCodeBlock = false
				// コードブロック全体を出力
				result.WriteString(r.renderCodeBlock(codeLanguage, codeBlockLines, maxCodeWidth))
			}
			continue
		}

		if inCodeBlock {
			codeBlockLines = append(codeBlockLines, line)
			// 最大幅を更新（表示可能文字のみ）
			lineWidth := len(strings.ReplaceAll(line, "\t", "    ")) // タブを4スペースとして計算
			if lineWidth > maxCodeWidth {
				maxCodeWidth = lineWidth
			}
			continue
		}

		// テーブル処理
		if r.isTableRow(line) {
			if !inTable {
				inTable = true
				tableHeaders = r.parseTableRow(line)
			} else {
				if r.isTableSeparator(line) {
					// ヘッダー区切り行は無視
					continue
				}
				tableRows = append(tableRows, r.parseTableRow(line))
			}
			continue
		} else if inTable {
			// テーブル終了
			result.WriteString(r.renderTable(tableHeaders, tableRows))
			result.WriteString("\n")
			inTable = false
			tableHeaders = []string{}
			tableRows = [][]string{}
		}

		// 通常の行処理
		processedLine := r.processInlineFormatting(line)

		// ヘッダー処理
		if processedLine = r.processHeaders(processedLine); processedLine != line {
			result.WriteString(processedLine + "\n")
			continue
		}

		// リスト処理
		if processedLine = r.processLists(processedLine); processedLine != line {
			result.WriteString(processedLine + "\n")
			continue
		}

		// 引用処理
		if processedLine = r.processQuotes(processedLine); processedLine != line {
			result.WriteString(processedLine + "\n")
			continue
		}

		// 通常行
		result.WriteString(processedLine + "\n")
	}

	// テーブルが未終了の場合
	if inTable {
		result.WriteString(r.renderTable(tableHeaders, tableRows))
	}

	return result.String()
}

// インライン書式を処理（**bold**, *italic*, `code`）
func (r *Renderer) processInlineFormatting(line string) string {
	if !r.config.EnableColors {
		return line
	}

	// **太字** 処理 (より robust)
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	line = boldRegex.ReplaceAllString(line, Bold+"$1"+Reset)

	// *斜体* 処理
	italicRegex := regexp.MustCompile(`\*([^*\s][^*]*[^*\s])\*`)
	line = italicRegex.ReplaceAllString(line, Italic+"$1"+Reset)

	// `インラインコード` 処理
	codeRegex := regexp.MustCompile("`([^`]+)`")
	line = codeRegex.ReplaceAllString(line, BgGray+Yellow+"$1"+Reset)

	// ~~取り消し線~~ 処理
	strikeRegex := regexp.MustCompile(`~~([^~]+)~~`)
	line = strikeRegex.ReplaceAllString(line, "\033[9m$1"+Reset)

	return line
}

// ヘッダーを処理
func (r *Renderer) processHeaders(line string) string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "#") {
		return line
	}

	if !r.config.EnableColors {
		return line
	}

	// ヘッダーレベルを取得
	level := 0
	for _, char := range line {
		if char == '#' {
			level++
		} else {
			break
		}
	}

	if level > 6 {
		return line
	}

	// ヘッダーテキストを抽出
	headerText := strings.TrimSpace(line[level:])
	if headerText == "" {
		return line
	}

	// レベルに応じたスタイル適用
	var style string
	switch level {
	case 1:
		style = Bold + Blue
	case 2:
		style = Bold + Cyan
	case 3:
		style = Bold + Green
	case 4:
		style = Bold + Yellow
	case 5:
		style = Bold + Magenta
	case 6:
		style = Bold + Gray
	}

	// Unicode記号を使用
	prefix := ""
	if r.config.UseUnicodeSymbols {
		switch level {
		case 1:
			prefix = "━━━ "
		case 2:
			prefix = "── "
		case 3:
			prefix = "▶ "
		default:
			prefix = "• "
		}
	}

	return fmt.Sprintf("%s%s%s%s", style, prefix, headerText, Reset)
}

// リストを処理
func (r *Renderer) processLists(line string) string {
	trimmed := strings.TrimSpace(line)

	// 番号付きリスト
	numberedRegex := regexp.MustCompile(`^(\d+)\.\s+(.*)$`)
	if matches := numberedRegex.FindStringSubmatch(trimmed); len(matches) == 3 {
		number := matches[1]
		text := matches[2]
		if r.config.EnableColors {
			return fmt.Sprintf("  %s%s.%s %s", Cyan, number, Reset, r.processInlineFormatting(text))
		}
		return fmt.Sprintf("  %s. %s", number, text)
	}

	// 箇条書きリスト
	bulletRegex := regexp.MustCompile(`^[-*+]\s+(.*)$`)
	if matches := bulletRegex.FindStringSubmatch(trimmed); len(matches) == 2 {
		text := matches[1]
		bullet := "•"
		if r.config.UseUnicodeSymbols {
			bullet = "▶"
		}
		if r.config.EnableColors {
			return fmt.Sprintf("  %s%s%s %s", Green, bullet, Reset, r.processInlineFormatting(text))
		}
		return fmt.Sprintf("  %s %s", bullet, text)
	}

	// チェックボックス
	checkboxRegex := regexp.MustCompile(`^[-*+]\s+\[([ x])\]\s+(.*)$`)
	if matches := checkboxRegex.FindStringSubmatch(trimmed); len(matches) == 3 {
		checked := matches[1] == "x"
		text := matches[2]

		var checkbox string
		if r.config.EnableColors {
			if checked {
				checkbox = fmt.Sprintf("%s✓%s", Green, Reset)
			} else {
				checkbox = fmt.Sprintf("%s☐%s", Gray, Reset)
			}
		} else {
			if checked {
				checkbox = "[x]"
			} else {
				checkbox = "[ ]"
			}
		}
		return fmt.Sprintf("  %s %s", checkbox, r.processInlineFormatting(text))
	}

	return line
}

// 引用を処理
func (r *Renderer) processQuotes(line string) string {
	if !strings.HasPrefix(strings.TrimSpace(line), ">") {
		return line
	}

	// 引用レベルを取得
	level := 0
	trimmed := strings.TrimSpace(line)
	for _, char := range trimmed {
		if char == '>' {
			level++
		} else {
			break
		}
	}

	// 引用テキストを抽出
	quoteText := strings.TrimSpace(trimmed[level:])

	if !r.config.EnableColors {
		return fmt.Sprintf("%s %s", strings.Repeat(">", level), quoteText)
	}

	// インデントと境界線
	indent := strings.Repeat("  ", level-1)
	border := fmt.Sprintf("%s▌%s", Blue, Reset)

	return fmt.Sprintf("%s%s %s%s", indent, border, Gray, r.processInlineFormatting(quoteText)) + Reset
}

// コードブロック全体を描画（幅調整付き）
func (r *Renderer) renderCodeBlock(language string, lines []string, maxWidth int) string {
	if !r.config.EnableColors {
		result := fmt.Sprintf("```%s\n", language)
		for _, line := range lines {
			result += line + "\n"
		}
		result += "```\n"
		return result
	}

	var result strings.Builder

	// 最小幅を確保（言語名 + 装飾を考慮）
	minWidth := 20
	if language != "" {
		minWidth = len(language) + 8 // "╭─  ─╮" の分
	}

	// ターミナル幅を取得して調整
	terminalWidth := r.getTerminalWidth()

	// 実際のコンテンツ幅を決定（ボーダー + マージンを考慮）
	maxAllowedWidth := terminalWidth - 6 // 両側の余白とボーダーを考慮
	contentWidth := maxWidth
	if contentWidth < minWidth {
		contentWidth = minWidth
	}
	if contentWidth > maxAllowedWidth {
		contentWidth = maxAllowedWidth
	}

	// 上部境界線（言語名を含む）
	switch r.config.CodeBlockStyle {
	case "bordered":
		if language != "" {
			// 色コードを除いた実際の表示長を計算
			headerDisplayLen := len(fmt.Sprintf("─ %s ", language))

			// コンテンツ幅に合わせて調整
			totalBorderLen := contentWidth + 2                       // "│ " の分
			remainingDashes := totalBorderLen - headerDisplayLen - 2 // ╭╮の分
			if remainingDashes < 1 {
				remainingDashes = 1
			}

			result.WriteString(fmt.Sprintf("\n%s╭─ %s%s%s %s─╮%s\n",
				Gray, Blue, language, Gray, strings.Repeat("─", remainingDashes), Reset))
		} else {
			result.WriteString(fmt.Sprintf("\n%s╭%s╮%s\n",
				Gray, strings.Repeat("─", contentWidth+2), Reset))
		}
	default:
		if language != "" {
			headerDisplayLen := len(fmt.Sprintf("─ %s ", language))
			totalBorderLen := contentWidth + 2
			remainingDashes := totalBorderLen - headerDisplayLen - 2
			if remainingDashes < 1 {
				remainingDashes = 1
			}
			result.WriteString(fmt.Sprintf("\n%s┌─ %s %s─┐%s\n",
				Gray, language, strings.Repeat("─", remainingDashes), Reset))
		} else {
			result.WriteString(fmt.Sprintf("\n%s┌%s┐%s\n",
				Gray, strings.Repeat("─", contentWidth+2), Reset))
		}
	}

	// コード行
	for _, line := range lines {
		result.WriteString(r.renderCodeLine(line))
	}

	// 下部境界線
	switch r.config.CodeBlockStyle {
	case "bordered":
		result.WriteString(fmt.Sprintf("%s╰%s╯%s\n\n",
			Gray, strings.Repeat("─", contentWidth+2), Reset))
	default:
		result.WriteString(fmt.Sprintf("%s└%s┘%s\n\n",
			Gray, strings.Repeat("─", contentWidth+2), Reset))
	}

	return result.String()
}

// コード行を描画（拡張シンタックスハイライト）
func (r *Renderer) renderCodeLine(line string) string {
	if !r.config.EnableColors {
		return fmt.Sprintf("│ %s\n", line)
	}

	// シンタックスハイライト
	highlighted := r.applySyntaxHighlighting(line)

	switch r.config.CodeBlockStyle {
	case "bordered":
		return fmt.Sprintf("%s│%s %s\n", Gray, Reset, highlighted)
	default:
		return fmt.Sprintf("%s│%s %s\n", Gray, Reset, highlighted)
	}
}

// 拡張シンタックスハイライト
func (r *Renderer) applySyntaxHighlighting(line string) string {
	// Go のキーワード
	goKeywords := []string{"package", "import", "func", "var", "const", "type", "struct", "interface", "if", "else", "for", "range", "return", "defer", "go", "select", "case", "default", "switch"}

	// JavaScript/TypeScript のキーワード
	jsKeywords := []string{"function", "const", "let", "var", "class", "interface", "type", "import", "export", "default", "if", "else", "for", "while", "return", "async", "await"}

	// Python のキーワード
	pyKeywords := []string{"def", "class", "import", "from", "if", "elif", "else", "for", "while", "return", "try", "except", "finally", "with", "as"}

	// 全キーワードをまとめる
	allKeywords := append(goKeywords, jsKeywords...)
	allKeywords = append(allKeywords, pyKeywords...)

	result := line

	// キーワードハイライト
	for _, keyword := range allKeywords {
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(keyword))
		regex := regexp.MustCompile(pattern)
		result = regex.ReplaceAllString(result, Blue+keyword+Reset)
	}

	// 文字列リテラル
	stringRegex := regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)"`)
	result = stringRegex.ReplaceAllString(result, Green+"\"$1\""+Reset)

	// 単一引用符文字列
	singleStringRegex := regexp.MustCompile(`'([^'\\]*(\\.[^'\\]*)*)'`)
	result = singleStringRegex.ReplaceAllString(result, Green+"'$1'"+Reset)

	// コメント
	commentRegex := regexp.MustCompile(`//.*$`)
	result = commentRegex.ReplaceAllString(result, Gray+"$0"+Reset)

	// 数値
	numberRegex := regexp.MustCompile(`\b\d+(\.\d+)?\b`)
	result = numberRegex.ReplaceAllString(result, Cyan+"$0"+Reset)

	return result
}

// テーブル行かどうかを判定
func (r *Renderer) isTableRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.Contains(trimmed, "|") && strings.Count(trimmed, "|") >= 2
}

// テーブル区切り行かどうかを判定
func (r *Renderer) isTableSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, "|") {
		return false
	}

	// セパレータには主に |-: の文字が含まれる
	cleaned := strings.ReplaceAll(trimmed, "|", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")

	return cleaned == ""
}

// テーブル行をパース
func (r *Renderer) parseTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "|") {
		trimmed = trimmed[1:]
	}
	if strings.HasSuffix(trimmed, "|") {
		trimmed = trimmed[:len(trimmed)-1]
	}

	parts := strings.Split(trimmed, "|")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	return parts
}

// テーブルを描画
func (r *Renderer) renderTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	if !r.config.EnableColors {
		return r.renderSimpleTable(headers, rows)
	}

	var result strings.Builder

	// カラム幅を計算
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// 最大幅制限
	maxWidth := r.config.MaxTableWidth / len(headers)
	for i := range colWidths {
		if colWidths[i] > maxWidth {
			colWidths[i] = maxWidth
		}
	}

	// ヘッダー行
	result.WriteString(fmt.Sprintf("\n%s┌", Gray))
	for i, width := range colWidths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			result.WriteString("┬")
		}
	}
	result.WriteString(fmt.Sprintf("┐%s\n", Reset))

	// ヘッダー内容
	result.WriteString(fmt.Sprintf("%s│%s", Gray, Reset))
	for i, header := range headers {
		content := r.truncateString(header, colWidths[i])
		padding := strings.Repeat(" ", colWidths[i]-len(content))
		result.WriteString(fmt.Sprintf(" %s%s%s%s %s│%s", Bold, Cyan, content, Reset, padding, Gray))
	}
	result.WriteString(fmt.Sprintf("%s\n", Reset))

	// 区切り行
	result.WriteString(fmt.Sprintf("%s├", Gray))
	for i, width := range colWidths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			result.WriteString("┼")
		}
	}
	result.WriteString(fmt.Sprintf("┤%s\n", Reset))

	// データ行
	for _, row := range rows {
		result.WriteString(fmt.Sprintf("%s│%s", Gray, Reset))
		for i := range colWidths {
			var content string
			if i < len(row) {
				content = r.truncateString(row[i], colWidths[i])
			}
			padding := strings.Repeat(" ", colWidths[i]-len(content))
			result.WriteString(fmt.Sprintf(" %s%s %s│%s", content, padding, Gray, Reset))
		}
		result.WriteString("\n")
	}

	// 終了行
	result.WriteString(fmt.Sprintf("%s└", Gray))
	for i, width := range colWidths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			result.WriteString("┴")
		}
	}
	result.WriteString(fmt.Sprintf("┘%s\n\n", Reset))

	return result.String()
}

// シンプルテーブル描画（カラー無効時）
func (r *Renderer) renderSimpleTable(headers []string, rows [][]string) string {
	var result strings.Builder

	// ヘッダー
	result.WriteString(strings.Join(headers, " | ") + "\n")
	result.WriteString(strings.Repeat("-", len(strings.Join(headers, " | "))) + "\n")

	// データ行
	for _, row := range rows {
		result.WriteString(strings.Join(row, " | ") + "\n")
	}

	return result.String() + "\n"
}

// 文字列を指定幅で切り詰め
func (r *Renderer) truncateString(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return s[:maxWidth]
	}
	return s[:maxWidth-3] + "..."
}

// レンダリング設定を更新
func (r *Renderer) UpdateConfig(config RenderConfig) {
	r.config = config
}

// カラー有効/無効を切り替え
func (r *Renderer) SetColorsEnabled(enabled bool) {
	r.config.EnableColors = enabled
}

// アニメーション有効/無効を切り替え
func (r *Renderer) SetAnimationsEnabled(enabled bool) {
	r.config.EnableAnimations = enabled
}
