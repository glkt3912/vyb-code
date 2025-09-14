package streaming

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// UIProcessor - UI表示専用ストリーミングプロセッサー
type UIProcessor struct {
	mu      sync.RWMutex
	config  *UnifiedStreamConfig
	state   *UIState
	metrics *StreamMetrics
}

// UIState - UI表示状態
type UIState struct {
	CurrentLine     string
	CurrentPage     int
	TotalLines      int
	InCodeBlock     bool
	CodeLanguage    string
	PausedForPage   bool
	LineCount       int64
	DisplayStarted  time.Time
}

// UIToken - トークン情報（UI表示用）
type UIToken struct {
	Content   string
	Type      UITokenType
	Delay     time.Duration
	NewLine   bool
	PageBreak bool
}

// UITokenType - トークンタイプ
type UITokenType string

const (
	UITokenText        UITokenType = "text"
	UITokenKeyword     UITokenType = "keyword"
	UITokenString      UITokenType = "string"
	UITokenComment     UITokenType = "comment"
	UITokenNumber      UITokenType = "number"
	UITokenPunctuation UITokenType = "punctuation"
	UITokenMarkdown    UITokenType = "markdown"
	UITokenCode        UITokenType = "code"
)

// NewUIProcessor - 新しいUIプロセッサーを作成
func NewUIProcessor(config *UnifiedStreamConfig) *UIProcessor {
	if config == nil {
		config = DefaultStreamConfig()
	}
	
	return &UIProcessor{
		config:  config,
		state:   &UIState{},
		metrics: &StreamMetrics{},
	}
}

// Process - UI表示用ストリーミング処理
func (p *UIProcessor) Process(ctx context.Context, input io.Reader, output io.Writer, options *StreamOptions) error {
	content, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("入力読み取りエラー: %w", err)
	}
	
	return p.ProcessString(ctx, string(content), output, options)
}

// ProcessString - 文字列をUI表示用にストリーミング処理
func (p *UIProcessor) ProcessString(ctx context.Context, content string, output io.Writer, options *StreamOptions) error {
	p.mu.Lock()
	p.state.DisplayStarted = time.Now()
	p.state.LineCount = 0
	p.mu.Unlock()
	
	if !p.config.EnableStreaming {
		_, err := fmt.Fprint(output, content)
		return err
	}
	
	// コンテンツをトークンに分解
	tokens := p.tokenizeContent(content)
	
	// 中断チャネルの設定
	var interruptCh <-chan struct{}
	if options != nil && options.EnableInterrupt {
		if ctx != nil {
			doneCh := make(chan struct{})
			go func() {
				<-ctx.Done()
				close(doneCh)
			}()
			interruptCh = doneCh
		}
	}
	
	// トークンを順次出力
	for i, token := range tokens {
		// 中断チェック
		if interruptCh != nil {
			select {
			case <-interruptCh:
				fmt.Fprint(output, "\n\033[90m[中断されました]\033[0m\n")
				p.updateMetrics(true)
				return fmt.Errorf("interrupted")
			default:
			}
		}
		
		// トークン出力
		fmt.Fprint(output, token.Content)
		
		// 改行処理
		if token.NewLine {
			fmt.Fprintln(output)
			p.mu.Lock()
			p.state.CurrentLine = ""
			p.state.LineCount++
			p.mu.Unlock()
		} else {
			p.mu.Lock()
			p.state.CurrentLine += token.Content
			p.mu.Unlock()
		}
		
		// 遅延処理（最後のトークン以外）
		if i < len(tokens)-1 && token.Delay > 0 {
			time.Sleep(token.Delay)
		}
	}
	
	p.updateMetrics(false)
	return nil
}

// SetConfig - 設定を更新
func (p *UIProcessor) SetConfig(config *UnifiedStreamConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = config
}

// GetMetrics - メトリクス取得
func (p *UIProcessor) GetMetrics() *StreamMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	metricsCopy := *p.metrics
	return &metricsCopy
}

// tokenizeContent - コンテンツをトークンに分解
func (p *UIProcessor) tokenizeContent(content string) []UIToken {
	var tokens []UIToken
	lines := strings.Split(content, "\n")
	
	for lineIndex, line := range lines {
		// コードブロック判定
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			p.mu.Lock()
			p.state.InCodeBlock = !p.state.InCodeBlock
			if p.state.InCodeBlock {
				p.state.CodeLanguage = strings.TrimPrefix(strings.TrimSpace(line), "```")
			}
			p.mu.Unlock()
			
			tokens = append(tokens, UIToken{
				Content: line,
				Type:    UITokenMarkdown,
				Delay:   p.config.CodeBlockDelay,
				NewLine: true,
			})
			continue
		}
		
		p.mu.RLock()
		inCodeBlock := p.state.InCodeBlock
		p.mu.RUnlock()
		
		if inCodeBlock {
			// コードブロック内：行単位で処理
			tokens = append(tokens, p.processCodeLine(line))
		} else {
			// 通常テキスト：単語・文レベルで処理
			lineTokens := p.processTextLine(line, lineIndex)
			tokens = append(tokens, lineTokens...)
		}
	}
	
	return tokens
}

// processTextLine - テキスト行を処理
func (p *UIProcessor) processTextLine(line string, lineIndex int) []UIToken {
	var tokens []UIToken
	
	if strings.TrimSpace(line) == "" {
		// 空行：段落区切りとして処理
		return []UIToken{{
			Content: "",
			Type:    UITokenText,
			Delay:   p.config.ParagraphDelay,
			NewLine: true,
		}}
	}
	
	// Markdownフォーマットを考慮した単語分割
	words := p.smartWordSplit(line)
	
	for wordIndex, word := range words {
		tokenType := p.identifyTokenType(word)
		delay := p.calculateDelay(word, tokenType)
		
		isLastWord := wordIndex == len(words)-1
		
		tokens = append(tokens, UIToken{
			Content: word,
			Type:    tokenType,
			Delay:   delay,
			NewLine: isLastWord, // 行末で改行
		})
		
		// 単語間にスペースを追加（最後の単語以外）
		if !isLastWord && !strings.HasSuffix(word, " ") {
			tokens = append(tokens, UIToken{
				Content: " ",
				Type:    UITokenText,
				Delay:   p.config.TokenDelay / 2,
			})
		}
	}
	
	return tokens
}

// processCodeLine - コード行を処理
func (p *UIProcessor) processCodeLine(line string) UIToken {
	return UIToken{
		Content: line,
		Type:    UITokenCode,
		Delay:   p.config.CodeBlockDelay,
		NewLine: true,
	}
}

// smartWordSplit - スマート単語分割（Markdown考慮）
func (p *UIProcessor) smartWordSplit(line string) []string {
	var words []string
	
	// 正規表現でMarkdown要素と通常テキストを分離
	markdownRegex := regexp.MustCompile(`(\*\*[^*]+\*\*|\*[^*]+\*|` + "`" + `[^` + "`" + `]+` + "`" + `|~~[^~]+~~)`)
	
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

// identifyTokenType - トークンタイプを識別
func (p *UIProcessor) identifyTokenType(word string) UITokenType {
	// Markdown要素の判定
	if strings.HasPrefix(word, "**") && strings.HasSuffix(word, "**") {
		return UITokenMarkdown
	}
	if strings.HasPrefix(word, "*") && strings.HasSuffix(word, "*") && !strings.HasPrefix(word, "**") {
		return UITokenMarkdown
	}
	if strings.HasPrefix(word, "`") && strings.HasSuffix(word, "`") {
		return UITokenMarkdown
	}
	if strings.HasPrefix(word, "~~") && strings.HasSuffix(word, "~~") {
		return UITokenMarkdown
	}
	
	// プログラミングキーワード
	keywords := []string{"func", "package", "import", "var", "const", "if", "else", "for", "return"}
	for _, keyword := range keywords {
		if word == keyword {
			return UITokenKeyword
		}
	}
	
	// 数値判定
	if matched, _ := regexp.MatchString(`^\d+(\.\d+)?$`, word); matched {
		return UITokenNumber
	}
	
	// 句読点判定
	if matched, _ := regexp.MatchString(`^[.,!?;:()[\]{}]+$`, word); matched {
		return UITokenPunctuation
	}
	
	return UITokenText
}

// calculateDelay - 遅延時間を計算
func (p *UIProcessor) calculateDelay(word string, tokenType UITokenType) time.Duration {
	baseDelay := p.config.TokenDelay
	
	// トークンタイプに応じた調整
	switch tokenType {
	case UITokenKeyword:
		return baseDelay * 2 // キーワードは少し長めに
	case UITokenCode:
		return p.config.CodeBlockDelay
	case UITokenPunctuation:
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

// updateMetrics - メトリクス更新
func (p *UIProcessor) updateMetrics(interrupted bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.metrics.TotalLines = p.state.LineCount
	if !p.state.DisplayStarted.IsZero() {
		p.metrics.DisplayDuration = time.Since(p.state.DisplayStarted)
	}
	
	if interrupted {
		p.metrics.InterruptCount++
	}
}