package streaming

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// ストリーミングプロセッサー
type Processor struct {
	config StreamConfig
	state  StreamState
}

// ストリーミング設定
type StreamConfig struct {
	TokenDelay      time.Duration // トークン間の遅延
	SentenceDelay   time.Duration // 文末での遅延
	ParagraphDelay  time.Duration // 段落間の遅延
	CodeBlockDelay  time.Duration // コードブロック内の遅延
	EnableStreaming bool          // ストリーミング有効/無効
	MaxLineLength   int           // 最大行長（改行挿入）
	EnablePaging    bool          // ページング有効
	PageSize        int           // ページサイズ（行数）
}

// ストリーミング状態
type StreamState struct {
	CurrentLine   string
	CurrentPage   int
	TotalLines    int
	InCodeBlock   bool
	CodeLanguage  string
	PausedForPage bool
}

// トークン情報
type Token struct {
	Content   string
	Type      TokenType
	Delay     time.Duration
	NewLine   bool
	PageBreak bool
}

// トークンタイプ
type TokenType string

const (
	TokenText      TokenType = "text"
	TokenKeyword   TokenType = "keyword"
	TokenString    TokenType = "string"
	TokenComment   TokenType = "comment"
	TokenNumber    TokenType = "number"
	TokenPunctuation TokenType = "punctuation"
	TokenMarkdown  TokenType = "markdown"
	TokenCode      TokenType = "code"
)

// デフォルト設定でプロセッサーを作成
func NewProcessor() *Processor {
	return &Processor{
		config: StreamConfig{
			TokenDelay:      15 * time.Millisecond,
			SentenceDelay:   100 * time.Millisecond,
			ParagraphDelay:  200 * time.Millisecond,
			CodeBlockDelay:  5 * time.Millisecond,
			EnableStreaming: true,
			MaxLineLength:   100,
			EnablePaging:    false,
			PageSize:        25,
		},
		state: StreamState{},
	}
}

// 設定付きプロセッサーを作成  
func NewProcessorWithConfig(cfg StreamConfig) *Processor {
	return &Processor{
		config: cfg,
		state:  StreamState{},
	}
}

// メインストリーミング処理
func (p *Processor) StreamContent(content string) error {
	if !p.config.EnableStreaming {
		fmt.Print(content)
		return nil
	}

	// コンテンツをトークンに分解
	tokens := p.tokenizeContent(content)
	
	// ページング準備
	if p.config.EnablePaging {
		p.state.TotalLines = strings.Count(content, "\n")
	}

	// トークンを順次出力
	for i, token := range tokens {
		// ページング処理
		if p.config.EnablePaging && token.PageBreak {
			if err := p.handlePaging(); err != nil {
				return err
			}
		}

		// トークン出力
		fmt.Print(token.Content)

		// 遅延処理（最後のトークンは遅延なし）
		if i < len(tokens)-1 && token.Delay > 0 {
			time.Sleep(token.Delay)
		}

		// 改行処理
		if token.NewLine {
			fmt.Println()
			p.state.CurrentLine = ""
		} else {
			p.state.CurrentLine += token.Content
		}
	}

	return nil
}

// コンテンツをトークンに分解
func (p *Processor) tokenizeContent(content string) []Token {
	var tokens []Token
	lines := strings.Split(content, "\n")
	
	for lineIndex, line := range lines {
		// コードブロック判定
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			p.state.InCodeBlock = !p.state.InCodeBlock
			if p.state.InCodeBlock {
				p.state.CodeLanguage = strings.TrimPrefix(strings.TrimSpace(line), "```")
			}
			
			// コードブロック境界をそのまま出力
			tokens = append(tokens, Token{
				Content:   line,
				Type:      TokenMarkdown,
				Delay:     p.config.CodeBlockDelay,
				NewLine:   true,
				PageBreak: p.shouldPageBreak(lineIndex),
			})
			continue
		}

		if p.state.InCodeBlock {
			// コードブロック内：行単位で処理
			tokens = append(tokens, p.processCodeLine(line, lineIndex))
		} else {
			// 通常テキスト：単語・文レベルで処理
			lineTokens := p.processTextLine(line, lineIndex)
			tokens = append(tokens, lineTokens...)
		}
	}

	return tokens
}

// テキスト行を処理
func (p *Processor) processTextLine(line string, lineIndex int) []Token {
	var tokens []Token
	
	if strings.TrimSpace(line) == "" {
		// 空行：段落区切りとして処理
		return []Token{{
			Content:   "",
			Type:      TokenText,
			Delay:     p.config.ParagraphDelay,
			NewLine:   true,
			PageBreak: p.shouldPageBreak(lineIndex),
		}}
	}

	// Markdownフォーマットを考慮した単語分割
	words := p.smartWordSplit(line)
	
	for wordIndex, word := range words {
		tokenType := p.identifyTokenType(word)
		delay := p.calculateDelay(word, tokenType)
		
		isLastWord := wordIndex == len(words)-1
		
		tokens = append(tokens, Token{
			Content:   word,
			Type:      tokenType,
			Delay:     delay,
			NewLine:   isLastWord, // 行末で改行
			PageBreak: p.shouldPageBreak(lineIndex) && wordIndex == 0,
		})

		// 単語間にスペースを追加（最後の単語以外）
		if !isLastWord && !strings.HasSuffix(word, " ") {
			tokens = append(tokens, Token{
				Content: " ",
				Type:    TokenText,
				Delay:   p.config.TokenDelay / 2,
			})
		}
	}

	return tokens
}

// コード行を処理
func (p *Processor) processCodeLine(line string, lineIndex int) Token {
	return Token{
		Content:   line,
		Type:      TokenCode,
		Delay:     p.config.CodeBlockDelay,
		NewLine:   true,
		PageBreak: p.shouldPageBreak(lineIndex),
	}
}

// スマート単語分割（Markdown考慮）
func (p *Processor) smartWordSplit(line string) []string {
	// Markdown要素を保持しながら分割
	var words []string
	
	// 正規表現でMarkdown要素と通常テキストを分離
	markdownRegex := regexp.MustCompile(`(\*\*[^*]+\*\*|\*[^*]+\*|`+"`"+`[^`+"`"+`]+`+"`"+`|~~[^~]+~~)`)
	
	parts := markdownRegex.Split(line, -1)
	matches := markdownRegex.FindAllString(line, -1)
	
	matchIndex := 0
	for i, part := range parts {
		// 通常テキスト部分を単語分割
		if part != "" {
			normalWords := strings.Fields(part)
			words = append(words, normalWords...)
		}
		
		// Markdown要素を追加
		if matchIndex < len(matches) && i < len(parts)-1 {
			words = append(words, matches[matchIndex])
			matchIndex++
		}
	}
	
	return words
}

// トークンタイプを識別
func (p *Processor) identifyTokenType(word string) TokenType {
	// Markdown要素の判定
	if strings.HasPrefix(word, "**") && strings.HasSuffix(word, "**") {
		return TokenMarkdown
	}
	if strings.HasPrefix(word, "*") && strings.HasSuffix(word, "*") && !strings.HasPrefix(word, "**") {
		return TokenMarkdown
	}
	if strings.HasPrefix(word, "`") && strings.HasSuffix(word, "`") {
		return TokenMarkdown
	}
	if strings.HasPrefix(word, "~~") && strings.HasSuffix(word, "~~") {
		return TokenMarkdown
	}

	// プログラミングキーワード
	keywords := []string{"func", "package", "import", "var", "const", "if", "else", "for", "return"}
	for _, keyword := range keywords {
		if word == keyword {
			return TokenKeyword
		}
	}

	// 数値判定
	if matched, _ := regexp.MatchString(`^\d+(\.\d+)?$`, word); matched {
		return TokenNumber
	}

	// 句読点判定
	if matched, _ := regexp.MatchString(`^[.,!?;:()[\]{}]+$`, word); matched {
		return TokenPunctuation
	}

	return TokenText
}

// 遅延時間を計算
func (p *Processor) calculateDelay(word string, tokenType TokenType) time.Duration {
	baseDelay := p.config.TokenDelay

	// トークンタイプに応じた調整
	switch tokenType {
	case TokenKeyword:
		return baseDelay * 2 // キーワードは少し長めに
	case TokenCode:
		return p.config.CodeBlockDelay
	case TokenPunctuation:
		// 文末記号は長めの遅延
		if strings.ContainsAny(word, "。！？") {
			return p.config.SentenceDelay
		}
		return baseDelay / 2
	}

	// 文字数に応じた調整
	runeCount := utf8.RuneCountInString(word)
	if runeCount > 10 {
		return baseDelay * 2
	}

	return baseDelay
}

// ページブレークが必要かを判定
func (p *Processor) shouldPageBreak(lineIndex int) bool {
	if !p.config.EnablePaging {
		return false
	}
	
	return lineIndex > 0 && lineIndex%p.config.PageSize == 0
}

// ページング処理
func (p *Processor) handlePaging() error {
	if p.state.PausedForPage {
		return nil
	}

	p.state.CurrentPage++
	
	// ページ継続の確認
	fmt.Printf("\n\033[90m--- ページ %d/%d ---\033[0m", 
		p.state.CurrentPage, 
		(p.state.TotalLines/p.config.PageSize)+1)
	fmt.Printf("\033[90m (Enter: 継続, q: 終了)\033[0m ")

	// ユーザー入力を待機
	var response string
	fmt.Scanln(&response)
	
	if response == "q" || response == "quit" {
		return fmt.Errorf("ユーザーによって中断されました")
	}

	// 画面をクリアしてページ表示を削除
	fmt.Print("\033[G\033[K")
	
	return nil
}

// ストリーミング設定を更新
func (p *Processor) UpdateConfig(config StreamConfig) {
	p.config = config
}

// ストリーミング有効/無効切り替え
func (p *Processor) SetStreamingEnabled(enabled bool) {
	p.config.EnableStreaming = enabled
}

// 速度プリセット設定
func (p *Processor) SetSpeedPreset(preset string) {
	switch preset {
	case "instant":
		p.config.TokenDelay = 0
		p.config.SentenceDelay = 0
		p.config.ParagraphDelay = 0
	case "fast":
		p.config.TokenDelay = 5 * time.Millisecond
		p.config.SentenceDelay = 50 * time.Millisecond
		p.config.ParagraphDelay = 100 * time.Millisecond
	case "normal":
		p.config.TokenDelay = 15 * time.Millisecond
		p.config.SentenceDelay = 100 * time.Millisecond
		p.config.ParagraphDelay = 200 * time.Millisecond
	case "slow":
		p.config.TokenDelay = 50 * time.Millisecond
		p.config.SentenceDelay = 300 * time.Millisecond
		p.config.ParagraphDelay = 500 * time.Millisecond
	case "typewriter":
		p.config.TokenDelay = 100 * time.Millisecond
		p.config.SentenceDelay = 500 * time.Millisecond
		p.config.ParagraphDelay = 1000 * time.Millisecond
	}
}

// 段落分割ストリーミング
func (p *Processor) StreamParagraphs(content string) error {
	if !p.config.EnableStreaming {
		fmt.Print(content)
		return nil
	}

	paragraphs := strings.Split(content, "\n\n")
	
	for i, paragraph := range paragraphs {
		if err := p.StreamContent(paragraph); err != nil {
			return err
		}
		
		// 段落間の遅延
		if i < len(paragraphs)-1 {
			time.Sleep(p.config.ParagraphDelay)
			fmt.Print("\n\n")
		}
	}
	
	return nil
}

// 現在の状態を取得
func (p *Processor) GetState() StreamState {
	return p.state
}

// 状態をリセット
func (p *Processor) Reset() {
	p.state = StreamState{}
}

// 中断可能なストリーミング
func (p *Processor) StreamContentInterruptible(content string, interrupt <-chan struct{}) error {
	if !p.config.EnableStreaming {
		select {
		case <-interrupt:
			return fmt.Errorf("interrupted")
		default:
			fmt.Print(content)
			return nil
		}
	}

	tokens := p.tokenizeContent(content)
	
	for i, token := range tokens {
		// 中断チェック
		select {
		case <-interrupt:
			fmt.Print("\n\033[90m[中断されました]\033[0m\n")
			return fmt.Errorf("interrupted")
		default:
		}

		// トークン出力
		fmt.Print(token.Content)

		// 改行処理
		if token.NewLine {
			fmt.Println()
		}

		// 遅延処理（最後のトークン以外）
		if i < len(tokens)-1 && token.Delay > 0 {
			time.Sleep(token.Delay)
		}
	}

	return nil
}