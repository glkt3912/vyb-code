package markdown

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRenderer_ProcessInlineFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasANSI  bool
	}{
		{
			name:     "Bold text",
			input:    "This is **bold** text",
			expected: "bold",
			hasANSI:  true,
		},
		{
			name:     "Italic text",
			input:    "This is *italic* text",
			expected: "italic",
			hasANSI:  true,
		},
		{
			name:     "Inline code",
			input:    "Use `console.log()` for debugging",
			expected: "console.log()",
			hasANSI:  true,
		},
		{
			name:     "Strikethrough text",
			input:    "This is ~~deleted~~ text",
			expected: "deleted",
			hasANSI:  true,
		},
		{
			name:     "Mixed formatting",
			input:    "**Bold** and *italic* with `code`",
			expected: "Bold",
			hasANSI:  true,
		},
		{
			name:     "No formatting",
			input:    "Plain text without any formatting",
			expected: "Plain text without any formatting",
			hasANSI:  false,
		},
	}

	renderer := NewRenderer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.processInlineFormatting(tt.input)

			// ANSIコードが含まれているかチェック
			hasANSI := strings.Contains(result, "\033[")
			if hasANSI != tt.hasANSI {
				t.Errorf("Expected ANSI codes: %v, got: %v", tt.hasANSI, hasANSI)
			}

			// 期待するテキストが含まれているかチェック
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected result to contain %q, got: %q", tt.expected, result)
			}
		})
	}
}

func TestRenderer_ProcessHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		level    int
	}{
		{
			name:     "H1 header",
			input:    "# Main Title",
			expected: "Main Title",
			level:    1,
		},
		{
			name:     "H2 header",
			input:    "## Subtitle",
			expected: "Subtitle",
			level:    2,
		},
		{
			name:     "H3 header",
			input:    "### Section",
			expected: "Section",
			level:    3,
		},
		{
			name:     "Not a header",
			input:    "Regular text with # symbol",
			expected: "Regular text with # symbol",
			level:    0,
		},
		{
			name:     "Empty header",
			input:    "###",
			expected: "###",
			level:    0,
		},
	}

	renderer := NewRenderer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.processHeaders(tt.input)

			// ヘッダーテキストが含まれているかチェック
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected result to contain %q, got: %q", tt.expected, result)
			}

			// ヘッダーレベルに応じたUnicode記号のチェック
			if tt.level > 0 {
				hasUnicodeSymbol := strings.Contains(result, "━") || strings.Contains(result, "─") ||
					strings.Contains(result, "▶") || strings.Contains(result, "•")
				if !hasUnicodeSymbol {
					t.Errorf("Expected header to contain Unicode symbol")
				}
			}
		})
	}
}

func TestRenderer_ProcessLists(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		isListItem bool
		listType   string
	}{
		{
			name:       "Numbered list",
			input:      "1. First item",
			isListItem: true,
			listType:   "numbered",
		},
		{
			name:       "Bullet list",
			input:      "- Bullet item",
			isListItem: true,
			listType:   "bullet",
		},
		{
			name:       "Checked checkbox",
			input:      "- [x] Completed task",
			isListItem: true,
			listType:   "checkbox_checked",
		},
		{
			name:       "Unchecked checkbox",
			input:      "- [ ] Todo item",
			isListItem: true,
			listType:   "checkbox_unchecked",
		},
		{
			name:       "Not a list",
			input:      "Regular text",
			isListItem: false,
			listType:   "",
		},
	}

	renderer := NewRenderer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.processLists(tt.input)

			if tt.isListItem {
				// リストアイテムは元の行と異なるはず
				if result == tt.input {
					t.Errorf("Expected list processing to modify the line")
				}

				// Unicode記号が含まれているかチェック
				hasUnicodeSymbol := strings.Contains(result, "▶") || strings.Contains(result, "•") ||
					strings.Contains(result, "✓") || strings.Contains(result, "☐")
				if !hasUnicodeSymbol {
					t.Errorf("Expected list item to contain Unicode symbol")
				}
			} else {
				// リストでない場合は変更されないはず
				if result != tt.input {
					t.Errorf("Expected non-list item to remain unchanged, got: %q", result)
				}
			}
		})
	}
}

func TestRenderer_TableProcessing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isTable bool
	}{
		{
			name:    "Table row",
			input:   "| Column 1 | Column 2 | Column 3 |",
			isTable: true,
		},
		{
			name:    "Table separator",
			input:   "|----------|----------|----------|",
			isTable: true,
		},
		{
			name:    "Not a table",
			input:   "Regular text with | pipe symbol",
			isTable: false,
		},
		{
			name:    "Single pipe",
			input:   "Just one | pipe",
			isTable: false,
		},
	}

	renderer := NewRenderer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isTableRow := renderer.isTableRow(tt.input)
			if isTableRow != tt.isTable {
				t.Errorf("Expected isTableRow: %v, got: %v", tt.isTable, isTableRow)
			}

			if tt.isTable {
				// テーブル行の解析テスト
				parsed := renderer.parseTableRow(tt.input)
				if len(parsed) < 2 {
					t.Errorf("Expected table row to have at least 2 columns, got: %d", len(parsed))
				}
			}
		})
	}
}

func TestRenderer_SyntaxHighlighting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // 期待される色付きキーワード
	}{
		{
			name:     "Go function",
			input:    "func main() { return }",
			expected: []string{"func", "return"},
		},
		{
			name:     "JavaScript function",
			input:    "function test() { const x = 42; }",
			expected: []string{"function", "const"},
		},
		{
			name:     "String literals",
			input:    `fmt.Println("Hello world")`,
			expected: []string{"\"Hello world\""},
		},
		{
			name:     "Comments",
			input:    "// This is a comment",
			expected: []string{"// This is a comment"},
		},
		{
			name:     "Numbers",
			input:    "x := 42.5",
			expected: []string{"42.5"},
		},
	}

	renderer := NewRenderer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.applySyntaxHighlighting(tt.input)

			for _, expectedKeyword := range tt.expected {
				if !strings.Contains(result, expectedKeyword) {
					t.Errorf("Expected result to contain highlighted %q, got: %q", expectedKeyword, result)
				}
			}

			// ANSIカラーコードが含まれているかチェック
			if !strings.Contains(result, "\033[") {
				t.Errorf("Expected syntax highlighting to add ANSI color codes")
			}
		})
	}
}

func TestRenderer_FullMarkdownDocument(t *testing.T) {
	markdownContent := `# Main Title

This is a paragraph with **bold** text and *italic* text.

## Code Example

Here's some code:

` + "```go" + `
package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
` + "```" + `

## Features List

1. First feature
2. Second feature with ` + "`" + `inline code` + "`" + `
- Bullet point
- [x] Completed task  
- [ ] Todo item

> This is a blockquote
> with multiple lines

| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Data 1   | Data 2   | Data 3   |
| More     | Data     | Here     |
`

	renderer := NewRenderer()
	result := renderer.Render(markdownContent)

	t.Run("Contains headers", func(t *testing.T) {
		if !strings.Contains(result, "Main Title") {
			t.Errorf("Expected rendered output to contain header")
		}
	})

	t.Run("Contains formatted text", func(t *testing.T) {
		// Bold and italic should be processed
		if !strings.Contains(result, "\033[1m") { // Bold ANSI code
			t.Errorf("Expected bold formatting")
		}
	})

	t.Run("Contains code block", func(t *testing.T) {
		// コードブロック境界文字をチェック（実装に応じて調整）
		hasCodeBlock := strings.Contains(result, "╭─") || strings.Contains(result, "┌─") ||
			strings.Contains(result, "```")
		if !hasCodeBlock {
			t.Errorf("Expected code block formatting")
		}
	})

	t.Run("Contains syntax highlighting", func(t *testing.T) {
		// Check for Go keyword highlighting
		if !strings.Contains(result, "package") || !strings.Contains(result, "func") {
			t.Errorf("Expected Go syntax highlighting")
		}
	})

	t.Run("Contains list formatting", func(t *testing.T) {
		// Check for Unicode list symbols
		hasListSymbols := strings.Contains(result, "▶") || strings.Contains(result, "•") ||
			strings.Contains(result, "✓") || strings.Contains(result, "☐")
		if !hasListSymbols {
			t.Errorf("Expected list formatting with Unicode symbols")
		}
	})

	t.Run("Contains table", func(t *testing.T) {
		if !strings.Contains(result, "┌") || !strings.Contains(result, "┐") {
			t.Errorf("Expected table formatting with box-drawing characters")
		}
	})
}

func TestRenderer_ConfigurableOptions(t *testing.T) {
	tests := []struct {
		name   string
		config RenderConfig
		input  string
		check  func(string) bool
	}{
		{
			name: "Colors disabled",
			config: RenderConfig{
				EnableColors: false,
			},
			input: "**bold text**",
			check: func(result string) bool {
				return !strings.Contains(result, "\033[")
			},
		},
		{
			name: "Simple code blocks",
			config: RenderConfig{
				EnableColors:   true,
				CodeBlockStyle: "simple",
			},
			input: "```\ncode\n```",
			check: func(result string) bool {
				return strings.Contains(result, "┌─") && !strings.Contains(result, "╭─")
			},
		},
		{
			name: "Unicode symbols disabled",
			config: RenderConfig{
				EnableColors:      true,
				UseUnicodeSymbols: false,
			},
			input: "# Header",
			check: func(result string) bool {
				return !strings.Contains(result, "━━━") && !strings.Contains(result, "──")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewRendererWithConfig(tt.config)
			result := renderer.Render(tt.input)

			if !tt.check(result) {
				t.Errorf("Configuration check failed for input: %q, result: %q", tt.input, result)
			}
		})
	}
}

func TestRenderer_EdgeCases(t *testing.T) {
	renderer := NewRenderer()

	t.Run("Empty content", func(t *testing.T) {
		result := renderer.Render("")
		// 空入力は空文字または改行のみを返す可能性がある
		if result != "" && result != "\n" {
			t.Errorf("Expected empty or newline result for empty input, got: %q", result)
		}
	})

	t.Run("Only whitespace", func(t *testing.T) {
		result := renderer.Render("   \n\n   \n")
		lines := strings.Split(result, "\n")
		if len(lines) < 3 {
			t.Errorf("Expected whitespace to be preserved")
		}
	})

	t.Run("Malformed markdown", func(t *testing.T) {
		malformed := "**unclosed bold and `unclosed code"
		result := renderer.Render(malformed)
		// 壊れたMarkdownでもクラッシュしないことを確認
		if result == "" {
			t.Errorf("Expected non-empty result for malformed markdown")
		}
	})

	t.Run("Very long table", func(t *testing.T) {
		longTable := "| " + strings.Repeat("Very Long Column Header ", 10) + " |"
		result := renderer.Render(longTable)
		// テーブルが切り詰められることを確認
		if len(result) > 1000 {
			t.Errorf("Expected long table to be truncated")
		}
	})
}

func TestRenderer_PerformanceInlineFormatting(t *testing.T) {
	// パフォーマンステスト：大きなMarkdownドキュメントの処理
	largeContent := strings.Repeat("This is **bold** text with *italic* and `code` formatting.\n", 1000)

	renderer := NewRenderer()

	// 処理時間を測定
	start := time.Now()
	result := renderer.Render(largeContent)
	duration := time.Since(start)

	if duration > 5*time.Second {
		t.Errorf("Rendering took too long: %v", duration)
	}

	if len(result) == 0 {
		t.Errorf("Expected non-empty result for large content")
	}

	// メモリリークがないことを確認（基本的なチェック）
	if len(result) > len(largeContent)*10 {
		t.Errorf("Result size suggests possible memory leak: input=%d, output=%d", len(largeContent), len(result))
	}
}

// テーブルレンダリングの詳細テスト
func TestRenderer_TableRendering(t *testing.T) {
	tableContent := `| Name | Age | City |
|------|-----|------|
| Alice | 30 | Tokyo |
| Bob | 25 | Osaka |`

	renderer := NewRenderer()
	result := renderer.Render(tableContent)

	t.Run("Table structure", func(t *testing.T) {
		// Box-drawing charactersが含まれているかチェック
		boxChars := []string{"┌", "┐", "└", "┘", "├", "┤", "┬", "┴", "┼", "│", "─"}
		hasBoxChars := false
		for _, char := range boxChars {
			if strings.Contains(result, char) {
				hasBoxChars = true
				break
			}
		}
		if !hasBoxChars {
			t.Errorf("Expected table to contain box-drawing characters")
		}
	})

	t.Run("Table data", func(t *testing.T) {
		expectedData := []string{"Alice", "30", "Tokyo", "Bob", "25", "Osaka"}
		for _, data := range expectedData {
			if !strings.Contains(result, data) {
				t.Errorf("Expected table to contain data: %q", data)
			}
		}
	})
}

// Benchmark tests
func BenchmarkRenderer_SimpleText(b *testing.B) {
	renderer := NewRenderer()
	content := "Simple text without any formatting."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Render(content)
	}
}

func BenchmarkRenderer_ComplexMarkdown(b *testing.B) {
	renderer := NewRenderer()
	content := "# Header\n\n**Bold** and *italic* text with " + "`" + "code" + "`" + ".\n\n" + "```" + "go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n" + "```" + "\n\n| Col1 | Col2 |\n|------|------|\n| Data | More |\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Render(content)
	}
}

func BenchmarkRenderer_LargeDocument(b *testing.B) {
	renderer := NewRenderer()
	// 大きなドキュメントを生成
	var builder strings.Builder
	for i := 0; i < 100; i++ {
		builder.WriteString(fmt.Sprintf("## Section %d\n\n", i))
		builder.WriteString("This is **bold** text with *italic* formatting.\n\n")
		builder.WriteString("```go\nfunc example() { return nil }\n```\n\n")
	}
	content := builder.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Render(content)
	}
}
