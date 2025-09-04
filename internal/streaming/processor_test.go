package streaming

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestProcessor_Creation(t *testing.T) {
	t.Run("Default processor", func(t *testing.T) {
		processor := NewProcessor()

		if processor == nil {
			t.Error("Expected non-nil processor")
		}

		// デフォルト設定の確認
		if processor.config.TokenDelay != 15*time.Millisecond {
			t.Errorf("Expected default TokenDelay 15ms, got %v", processor.config.TokenDelay)
		}

		if !processor.config.EnableStreaming {
			t.Error("Expected streaming to be enabled by default")
		}

		if processor.config.MaxLineLength != 100 {
			t.Errorf("Expected default MaxLineLength 100, got %d", processor.config.MaxLineLength)
		}
	})

	t.Run("Custom config processor", func(t *testing.T) {
		config := StreamConfig{
			TokenDelay:      5 * time.Millisecond,
			SentenceDelay:   50 * time.Millisecond,
			EnableStreaming: false,
			MaxLineLength:   80,
		}

		processor := NewProcessorWithConfig(config)

		if processor.config.TokenDelay != 5*time.Millisecond {
			t.Errorf("Expected custom TokenDelay 5ms, got %v", processor.config.TokenDelay)
		}

		if processor.config.EnableStreaming {
			t.Error("Expected streaming to be disabled in custom config")
		}
	})
}

func TestProcessor_TokenizeContent(t *testing.T) {
	processor := NewProcessor()

	tests := []struct {
		name          string
		input         string
		expectedTypes []TokenType
		minTokens     int
		maxTokens     int
	}{
		{
			name:          "Simple text",
			input:         "Hello world",
			expectedTypes: []TokenType{TokenText},
			minTokens:     1,
			maxTokens:     4, // "Hello", " ", "world"
		},
		{
			name:          "Bold markdown",
			input:         "This is **bold** text",
			expectedTypes: []TokenType{TokenText, TokenMarkdown, TokenText},
			minTokens:     3,
			maxTokens:     7,
		},
		{
			name:          "Code block",
			input:         "```go\nfunc main() {}\n```",
			expectedTypes: []TokenType{TokenMarkdown, TokenCode, TokenMarkdown},
			minTokens:     3,
			maxTokens:     3,
		},
		{
			name:          "Mixed content",
			input:         "Here's `inline code` and **bold** text",
			expectedTypes: []TokenType{TokenText, TokenMarkdown, TokenText, TokenMarkdown, TokenText},
			minTokens:     5,
			maxTokens:     10,
		},
		{
			name:          "Multiple lines",
			input:         "Line 1\n\nLine 3",
			expectedTypes: []TokenType{TokenText, TokenText, TokenText}, // 各行 + 空行
			minTokens:     5,
			maxTokens:     8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := processor.tokenizeContent(tt.input)

			if len(tokens) < tt.minTokens || len(tokens) > tt.maxTokens {
				t.Errorf("Expected %d-%d tokens, got %d", tt.minTokens, tt.maxTokens, len(tokens))
			}

			// トークンが空でないことを確認
			for i, token := range tokens {
				if token.Content == "" && !token.NewLine {
					t.Errorf("Token[%d] has empty content", i)
				}
			}
		})
	}
}

func TestProcessor_IdentifyTokenType(t *testing.T) {
	processor := NewProcessor()

	tests := []struct {
		input    string
		expected TokenType
	}{
		// Markdown要素
		{"**bold**", TokenMarkdown},
		{"*italic*", TokenMarkdown},
		{"`" + "code" + "`", TokenMarkdown},
		{"~~strikethrough~~", TokenMarkdown},

		// キーワード
		{"func", TokenKeyword},
		{"package", TokenKeyword},
		{"import", TokenKeyword},
		{"var", TokenKeyword},
		{"if", TokenKeyword},

		// 数値
		{"42", TokenNumber},
		{"3.14", TokenNumber},
		{"0", TokenNumber},

		// 句読点
		{".", TokenPunctuation},
		{",", TokenPunctuation},
		{"();", TokenPunctuation},

		// 通常テキスト
		{"hello", TokenText},
		{"variable_name", TokenText},
		{"function123", TokenText},
	}

	for _, tt := range tests {
		t.Run("Identify "+tt.input, func(t *testing.T) {
			result := processor.identifyTokenType(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v for %q, got %v", tt.expected, tt.input, result)
			}
		})
	}
}

func TestProcessor_SpeedPresets(t *testing.T) {
	processor := NewProcessor()

	tests := []struct {
		preset        string
		tokenDelay    time.Duration
		sentenceDelay time.Duration
	}{
		{"instant", 0, 0},
		{"fast", 5 * time.Millisecond, 50 * time.Millisecond},
		{"normal", 15 * time.Millisecond, 100 * time.Millisecond},
		{"slow", 50 * time.Millisecond, 300 * time.Millisecond},
		{"typewriter", 100 * time.Millisecond, 500 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run("Preset "+tt.preset, func(t *testing.T) {
			processor.SetSpeedPreset(tt.preset)

			if processor.config.TokenDelay != tt.tokenDelay {
				t.Errorf("Expected TokenDelay %v for %s preset, got %v",
					tt.tokenDelay, tt.preset, processor.config.TokenDelay)
			}

			if processor.config.SentenceDelay != tt.sentenceDelay {
				t.Errorf("Expected SentenceDelay %v for %s preset, got %v",
					tt.sentenceDelay, tt.preset, processor.config.SentenceDelay)
			}
		})
	}
}

func TestProcessor_SmartWordSplit(t *testing.T) {
	processor := NewProcessor()

	tests := []struct {
		name     string
		input    string
		contains []string // 含まれるべき単語
		excludes []string // 含まれてはいけない要素
	}{
		{
			name:     "Simple sentence",
			input:    "Hello world test",
			contains: []string{"Hello", "world", "test"},
			excludes: []string{},
		},
		{
			name:     "Markdown formatting",
			input:    "This is **bold** and *italic*",
			contains: []string{"This", "is", "**bold**", "and", "*italic*"},
			excludes: []string{"**bold", "bold**", "*italic", "italic*"},
		},
		{
			name:     "Inline code",
			input:    "Use " + "`" + "console.log()" + "`" + " for debugging",
			contains: []string{"Use", "`" + "console.log()" + "`", "for", "debugging"},
			excludes: []string{"console.log()", "`console.log()", "console.log()`"},
		},
		{
			name:     "Mixed formatting",
			input:    "**Bold** " + "`" + "code" + "`" + " and normal text",
			contains: []string{"**Bold**", "`" + "code" + "`", "and", "normal", "text"},
			excludes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.smartWordSplit(tt.input)

			// 含まれるべき要素をチェック
			for _, expected := range tt.contains {
				found := false
				for _, word := range result {
					if word == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find %q in result %v", expected, result)
				}
			}

			// 含まれてはいけない要素をチェック
			for _, excluded := range tt.excludes {
				for _, word := range result {
					if word == excluded {
						t.Errorf("Did not expect to find %q in result %v", excluded, result)
					}
				}
			}
		})
	}
}

func TestProcessor_CalculateDelay(t *testing.T) {
	processor := NewProcessor()

	tests := []struct {
		word        string
		tokenType   TokenType
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			word:        "func",
			tokenType:   TokenKeyword,
			expectedMin: 30 * time.Millisecond, // TokenDelay * 2
			expectedMax: 30 * time.Millisecond,
		},
		{
			word:        ".",
			tokenType:   TokenPunctuation,
			expectedMin: 7 * time.Millisecond, // TokenDelay / 2
			expectedMax: 8 * time.Millisecond,
		},
		{
			word:        "。",
			tokenType:   TokenPunctuation,
			expectedMin: 100 * time.Millisecond, // SentenceDelay
			expectedMax: 100 * time.Millisecond,
		},
		{
			word:        "verylongvariablename",
			tokenType:   TokenText,
			expectedMin: 30 * time.Millisecond, // TokenDelay * 2 for long words
			expectedMax: 30 * time.Millisecond,
		},
		{
			word:        "short",
			tokenType:   TokenText,
			expectedMin: 15 * time.Millisecond, // Normal TokenDelay
			expectedMax: 15 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.word, tt.tokenType), func(t *testing.T) {
			result := processor.calculateDelay(tt.word, tt.tokenType)

			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("Expected delay between %v and %v for %q (%v), got %v",
					tt.expectedMin, tt.expectedMax, tt.word, tt.tokenType, result)
			}
		})
	}
}

func TestProcessor_StateManagement(t *testing.T) {
	processor := NewProcessor()

	t.Run("Initial state", func(t *testing.T) {
		state := processor.GetState()

		if state.CurrentLine != "" {
			t.Errorf("Expected empty CurrentLine initially, got %q", state.CurrentLine)
		}

		if state.InCodeBlock {
			t.Error("Expected InCodeBlock to be false initially")
		}

		if state.CurrentPage != 0 {
			t.Errorf("Expected CurrentPage to be 0 initially, got %d", state.CurrentPage)
		}
	})

	t.Run("State reset", func(t *testing.T) {
		// 状態を変更
		processor.state.CurrentLine = "test line"
		processor.state.InCodeBlock = true
		processor.state.CurrentPage = 5

		// リセット
		processor.Reset()

		state := processor.GetState()
		if state.CurrentLine != "" {
			t.Errorf("Expected CurrentLine to be reset, got %q", state.CurrentLine)
		}

		if state.InCodeBlock {
			t.Error("Expected InCodeBlock to be reset to false")
		}

		if state.CurrentPage != 0 {
			t.Errorf("Expected CurrentPage to be reset to 0, got %d", state.CurrentPage)
		}
	})
}

func TestProcessor_ConfigUpdate(t *testing.T) {
	processor := NewProcessor()

	// 新しい設定
	newConfig := StreamConfig{
		TokenDelay:      25 * time.Millisecond,
		SentenceDelay:   150 * time.Millisecond,
		EnableStreaming: false,
		MaxLineLength:   120,
	}

	processor.UpdateConfig(newConfig)

	if processor.config.TokenDelay != 25*time.Millisecond {
		t.Errorf("Expected TokenDelay to be updated to 25ms, got %v", processor.config.TokenDelay)
	}

	if processor.config.EnableStreaming {
		t.Error("Expected EnableStreaming to be updated to false")
	}

	if processor.config.MaxLineLength != 120 {
		t.Errorf("Expected MaxLineLength to be updated to 120, got %d", processor.config.MaxLineLength)
	}
}

func TestProcessor_StreamingToggle(t *testing.T) {
	processor := NewProcessor()

	// 初期状態（有効）を確認
	if !processor.config.EnableStreaming {
		t.Error("Expected streaming to be enabled initially")
	}

	// ストリーミングを無効化
	processor.SetStreamingEnabled(false)
	if processor.config.EnableStreaming {
		t.Error("Expected streaming to be disabled after SetStreamingEnabled(false)")
	}

	// ストリーミングを再有効化
	processor.SetStreamingEnabled(true)
	if !processor.config.EnableStreaming {
		t.Error("Expected streaming to be enabled after SetStreamingEnabled(true)")
	}
}

func TestProcessor_CodeBlockDetection(t *testing.T) {
	processor := NewProcessor()

	codeBlockContent := "```go\npackage main\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"
	tokens := processor.tokenizeContent(codeBlockContent)

	// コードブロックの開始と終了が検出されることを確認
	hasMarkdownToken := false
	hasCodeToken := false

	for _, token := range tokens {
		switch token.Type {
		case TokenMarkdown:
			hasMarkdownToken = true
		case TokenCode:
			hasCodeToken = true
		}
	}

	if !hasMarkdownToken {
		t.Error("Expected to find TokenMarkdown for code block markers")
	}

	if !hasCodeToken {
		t.Error("Expected to find TokenCode for code content")
	}
}

func TestProcessor_PagingCalculation(t *testing.T) {
	config := StreamConfig{
		EnablePaging: true,
		PageSize:     10,
		TokenDelay:   1 * time.Millisecond,
	}
	processor := NewProcessorWithConfig(config)

	tests := []struct {
		lineIndex   int
		shouldBreak bool
	}{
		{0, false},  // 最初の行
		{5, false},  // ページ内
		{9, false},  // ページ末
		{10, true},  // 新ページ
		{20, true},  // 次のページ
		{25, false}, // ページ内
	}

	for _, tt := range tests {
		result := processor.shouldPageBreak(tt.lineIndex)
		if result != tt.shouldBreak {
			t.Errorf("Expected shouldPageBreak(%d) = %v, got %v",
				tt.lineIndex, tt.shouldBreak, result)
		}
	}
}

func TestProcessor_DelayCalculation(t *testing.T) {
	processor := NewProcessor()

	t.Run("Keyword delay", func(t *testing.T) {
		delay := processor.calculateDelay("func", TokenKeyword)
		expected := processor.config.TokenDelay * 2
		if delay != expected {
			t.Errorf("Expected keyword delay %v, got %v", expected, delay)
		}
	})

	t.Run("Sentence ending delay", func(t *testing.T) {
		delay := processor.calculateDelay("。", TokenPunctuation)
		expected := processor.config.SentenceDelay
		if delay != expected {
			t.Errorf("Expected sentence delay %v, got %v", expected, delay)
		}
	})

	t.Run("Long word delay", func(t *testing.T) {
		longWord := "verylongwordwithmorethan10characters"
		delay := processor.calculateDelay(longWord, TokenText)
		expected := processor.config.TokenDelay * 2
		if delay != expected {
			t.Errorf("Expected long word delay %v, got %v", expected, delay)
		}
	})

	t.Run("Normal punctuation delay", func(t *testing.T) {
		delay := processor.calculateDelay(",", TokenPunctuation)
		expected := processor.config.TokenDelay / 2
		if delay != expected {
			t.Errorf("Expected punctuation delay %v, got %v", expected, delay)
		}
	})
}

// ストリーミング中断のテスト
func TestProcessor_InterruptibleStreaming(t *testing.T) {
	processor := NewProcessor()
	processor.config.TokenDelay = 10 * time.Millisecond

	t.Run("Normal completion", func(t *testing.T) {
		interrupt := make(chan struct{})
		content := "Short test content"

		// 中断せずに完了
		err := processor.StreamContentInterruptible(content, interrupt)
		if err != nil {
			t.Errorf("Expected no error for normal completion, got %v", err)
		}
	})

	t.Run("Immediate interruption", func(t *testing.T) {
		interrupt := make(chan struct{})
		content := "Test content that will be interrupted"

		// すぐに中断
		close(interrupt)

		err := processor.StreamContentInterruptible(content, interrupt)
		if err == nil {
			t.Error("Expected error for interrupted streaming")
		}

		if !strings.Contains(err.Error(), "interrupted") {
			t.Errorf("Expected error to contain 'interrupted', got %v", err)
		}
	})

	t.Run("Delayed interruption", func(t *testing.T) {
		interrupt := make(chan struct{})
		content := strings.Repeat("word ", 100) // 長いコンテンツ

		// 少し待ってから中断
		go func() {
			time.Sleep(50 * time.Millisecond)
			close(interrupt)
		}()

		err := processor.StreamContentInterruptible(content, interrupt)
		if err == nil {
			t.Error("Expected error for delayed interruption")
		}
	})
}

// 非ストリーミングモードのテスト
func TestProcessor_NonStreamingMode(t *testing.T) {
	config := StreamConfig{
		EnableStreaming: false,
		TokenDelay:      100 * time.Millisecond, // 遅延は無視されるはず
	}
	processor := NewProcessorWithConfig(config)

	content := "Test content for non-streaming mode"

	start := time.Now()
	err := processor.StreamContent(content)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error in non-streaming mode, got %v", err)
	}

	// 遅延がないことを確認（即座に出力）
	if duration > 50*time.Millisecond {
		t.Errorf("Non-streaming mode took too long: %v", duration)
	}
}

// エッジケースのテスト
func TestProcessor_EdgeCases(t *testing.T) {
	processor := NewProcessor()

	t.Run("Empty content", func(t *testing.T) {
		tokens := processor.tokenizeContent("")
		// 空のコンテンツは1つの空トークンを生成する可能性がある
		if len(tokens) > 1 {
			t.Errorf("Expected 0-1 tokens for empty content, got %d", len(tokens))
		}
	})

	t.Run("Only whitespace", func(t *testing.T) {
		tokens := processor.tokenizeContent("   \n\n   \n")
		// 空行トークンが生成されることを確認
		if len(tokens) == 0 {
			t.Error("Expected tokens for whitespace-only content")
		}
	})

	t.Run("Only code block markers", func(t *testing.T) {
		tokens := processor.tokenizeContent("```\n```")
		if len(tokens) != 2 {
			t.Errorf("Expected 2 tokens for empty code block, got %d", len(tokens))
		}
	})

	t.Run("Malformed markdown", func(t *testing.T) {
		malformed := "**unclosed bold and `unclosed code"
		tokens := processor.tokenizeContent(malformed)
		// エラーが発生せず、何らかのトークンが生成されることを確認
		if len(tokens) == 0 {
			t.Error("Expected tokens even for malformed markdown")
		}
	})
}

// パフォーマンステスト
func BenchmarkProcessor_TokenizeContent(b *testing.B) {
	processor := NewProcessor()
	content := "# Large Document Test\n\nThis is a **large** document with *italic* text and " + "`" + "inline code" + "`" + ".\n\n" + "```" + "go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, world!\")\n    for i := 0; i < 100; i++ {\n        fmt.Printf(\"Number: %d\\n\", i)\n    }\n}\n" + "```" + "\n\n## Another Section\n\nMore **bold** text and additional content to test performance.\n"
	content = strings.Repeat(content, 10) // 10倍に拡大

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.tokenizeContent(content)
	}
}

func BenchmarkProcessor_SmartWordSplit(b *testing.B) {
	processor := NewProcessor()
	line := "This is a **complex** line with *italic* text and `inline code` formatting"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.smartWordSplit(line)
	}
}

func BenchmarkProcessor_IdentifyTokenType(b *testing.B) {
	processor := NewProcessor()
	words := []string{"func", "**bold**", "variable", "`code`", "123", ".", "hello"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		word := words[i%len(words)]
		processor.identifyTokenType(word)
	}
}
