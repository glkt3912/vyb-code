package input

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestHistory_Add(t *testing.T) {
	tests := []struct {
		name     string
		inputs   []string
		expected []string
	}{
		{
			name:     "Basic addition",
			inputs:   []string{"command1", "command2", "command3"},
			expected: []string{"command1", "command2", "command3"},
		},
		{
			name:     "Skip empty lines",
			inputs:   []string{"command1", "", "   ", "command2"},
			expected: []string{"command1", "command2"},
		},
		{
			name:     "Skip duplicates",
			inputs:   []string{"command1", "command1", "command2"},
			expected: []string{"command1", "command2"},
		},
		{
			name:     "Trim whitespace",
			inputs:   []string{"  command1  ", "command2\n", "\tcommand3\t"},
			expected: []string{"command1", "command2", "command3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := NewHistory(10)

			for _, input := range tt.inputs {
				history.Add(input)
			}

			if len(history.entries) != len(tt.expected) {
				t.Errorf("Expected %d entries, got %d", len(tt.expected), len(history.entries))
			}

			for i, expected := range tt.expected {
				if i >= len(history.entries) || history.entries[i] != expected {
					t.Errorf("Expected entry[%d] = %q, got %q", i, expected, history.entries[i])
				}
			}
		})
	}
}

func TestHistory_MaxSize(t *testing.T) {
	maxSize := 3
	history := NewHistory(maxSize)

	// 最大サイズを超える履歴を追加
	inputs := []string{"cmd1", "cmd2", "cmd3", "cmd4", "cmd5"}
	for _, input := range inputs {
		history.Add(input)
	}

	// 最大サイズを超えないことを確認
	if len(history.entries) != maxSize {
		t.Errorf("Expected history size to be %d, got %d", maxSize, len(history.entries))
	}

	// 最新の項目が保持されていることを確認
	expected := []string{"cmd3", "cmd4", "cmd5"}
	for i, exp := range expected {
		if history.entries[i] != exp {
			t.Errorf("Expected entry[%d] = %q, got %q", i, exp, history.entries[i])
		}
	}
}

func TestHistory_Navigation(t *testing.T) {
	history := NewHistory(10)
	entries := []string{"first", "second", "third"}

	for _, entry := range entries {
		history.Add(entry)
	}

	t.Run("Previous navigation", func(t *testing.T) {
		// 最初のPrevious()は最後の項目を返す
		result := history.Previous()
		if result != "third" {
			t.Errorf("Expected 'third', got %q", result)
		}

		// 次のPrevious()は2番目の項目を返す
		result = history.Previous()
		if result != "second" {
			t.Errorf("Expected 'second', got %q", result)
		}

		// さらにPrevious()は最初の項目を返す
		result = history.Previous()
		if result != "first" {
			t.Errorf("Expected 'first', got %q", result)
		}

		// 境界チェック: これ以上戻れない
		result = history.Previous()
		if result != "first" {
			t.Errorf("Expected 'first' (boundary), got %q", result)
		}
	})

	t.Run("Next navigation", func(t *testing.T) {
		// Previous()で戻った状態からNext()をテスト
		result := history.Next()
		if result != "second" {
			t.Errorf("Expected 'second', got %q", result)
		}

		result = history.Next()
		if result != "third" {
			t.Errorf("Expected 'third', got %q", result)
		}

		// 境界チェック: 最後まで進んだ場合は tempLine を返す
		result = history.Next()
		if result != "" { // tempLine is initially empty
			t.Errorf("Expected empty tempLine, got %q", result)
		}
	})
}

func TestCompleter_GetSuggestions(t *testing.T) {
	completer := NewCompleter("/test/dir")

	tests := []struct {
		name     string
		input    string
		expected []string
		contains bool // 完全一致ではなく含まれているかチェック
	}{
		{
			name:     "Slash commands",
			input:    "/h",
			expected: []string{"/help", "/history"},
			contains: true,
		},
		{
			name:     "Single slash command",
			input:    "/help",
			expected: []string{"/help"},
			contains: true,
		},
		{
			name:     "Common commands",
			input:    "h",
			expected: []string{"help"},
			contains: true,
		},
		{
			name:     "Build command",
			input:    "b",
			expected: []string{"build"},
			contains: true,
		},
		{
			name:     "Empty input",
			input:    "",
			expected: []string{},
			contains: false,
		},
		{
			name:     "No matches",
			input:    "xyz123",
			expected: []string{},
			contains: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := completer.GetSuggestions(tt.input)

			if len(tt.expected) == 0 {
				if len(result) != 0 {
					t.Errorf("Expected no suggestions, got %v", result)
				}
				return
			}

			if tt.contains {
				// 期待される項目がすべて含まれているかチェック
				for _, expected := range tt.expected {
					found := false
					for _, suggestion := range result {
						if strings.Contains(suggestion, expected) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find suggestion containing %q, got %v", expected, result)
					}
				}
			} else {
				// 完全一致チェック
				if len(result) != len(tt.expected) {
					t.Errorf("Expected %d suggestions, got %d: %v", len(tt.expected), len(result), result)
					return
				}

				for i, expected := range tt.expected {
					if result[i] != expected {
						t.Errorf("Expected suggestion[%d] = %q, got %q", i, expected, result[i])
					}
				}
			}
		})
	}
}

func TestCompleter_SlashCommands(t *testing.T) {
	completer := NewCompleter("/test")

	// すべてのスラッシュコマンドが含まれているかテスト
	expectedCommands := []string{"/help", "/clear", "/history", "/status", "/info", "/save", "/retry", "/edit"}

	for _, cmd := range expectedCommands {
		t.Run("Command "+cmd, func(t *testing.T) {
			found := false
			for _, available := range completer.commands {
				if available == cmd {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected command %q to be available", cmd)
			}
		})
	}

	// 部分一致テスト
	suggestions := completer.GetSuggestions("/")
	for _, cmd := range expectedCommands {
		found := false
		for _, suggestion := range suggestions {
			if suggestion == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected %q in suggestions for '/', got %v", cmd, suggestions)
		}
	}
}

func TestReader_Creation(t *testing.T) {
	reader := NewReader()

	if reader == nil {
		t.Error("Expected non-nil reader")
	}

	if reader.history == nil {
		t.Error("Expected non-nil history")
	}

	if reader.completer == nil {
		t.Error("Expected non-nil completer")
	}

	if reader.securityValidator == nil {
		t.Error("Expected non-nil security validator")
	}

	if reader.clientID == "" {
		t.Error("Expected non-empty client ID")
	}

	if reader.isRawMode {
		t.Error("Expected raw mode to be disabled initially")
	}

	// 履歴の初期状態をチェック
	if len(reader.history.entries) != 0 {
		t.Errorf("Expected empty history initially, got %d entries", len(reader.history.entries))
	}

	// インデックスの初期値をチェック
	if reader.history.index != -1 {
		t.Errorf("Expected history index to be -1 initially, got %d", reader.history.index)
	}
}

func TestReader_PromptHandling(t *testing.T) {
	reader := NewReader()

	// 初期プロンプトテスト
	if reader.prompt != "" {
		t.Errorf("Expected empty initial prompt, got %q", reader.prompt)
	}

	// プロンプト設定テスト
	testPrompt := "vyb> "
	reader.SetPrompt(testPrompt)

	if reader.prompt != testPrompt {
		t.Errorf("Expected prompt %q, got %q", testPrompt, reader.prompt)
	}

	// Unicode文字を含むプロンプトのテスト
	unicodePrompt := "🤖 vyb> "
	reader.SetPrompt(unicodePrompt)

	if reader.prompt != unicodePrompt {
		t.Errorf("Expected Unicode prompt %q, got %q", unicodePrompt, reader.prompt)
	}
}

func TestHistory_EmptyHistoryNavigation(t *testing.T) {
	history := NewHistory(10)

	// 空の履歴でのナビゲーションテスト
	prev := history.Previous()
	if prev != "" {
		t.Errorf("Expected empty string for empty history Previous(), got %q", prev)
	}

	next := history.Next()
	if next != "" {
		t.Errorf("Expected empty string for empty history Next(), got %q", next)
	}
}

func TestCompleter_FilePathEdgeCases(t *testing.T) {
	completer := NewCompleter("/nonexistent/dir")

	tests := []struct {
		name  string
		input string
		desc  string
	}{
		{
			name:  "Invalid directory path",
			input: "/nonexistent/file.txt",
			desc:  "Should handle non-existent directories gracefully",
		},
		{
			name:  "Complex path with dots",
			input: "../some/path",
			desc:  "Should handle relative paths with parent directory references",
		},
		{
			name:  "Path with spaces",
			input: "path with spaces/",
			desc:  "Should handle paths containing spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// エラーが発生しないことを確認
			suggestions := completer.GetSuggestions(tt.input)
			// 空のリストが返されることを期待（エラーが発生しないこと）
			if suggestions == nil {
				// nil の場合は空スライスとして扱う（パニックが発生しなければOK）
				t.Logf("Suggestions is nil for input %q, which is acceptable", tt.input)
			}
			// パニックが発生しなければOK
		})
	}
}

// パフォーマンステスト
func TestHistory_LargeHistory(t *testing.T) {
	maxSize := 1000
	history := NewHistory(maxSize)

	start := time.Now()

	// 大量の履歴を追加
	for i := 0; i < 2000; i++ {
		history.Add(fmt.Sprintf("command%d", i))
	}

	duration := time.Since(start)

	// パフォーマンスチェック（1秒以内）
	if duration > time.Second {
		t.Errorf("Adding 2000 entries took too long: %v", duration)
	}

	// サイズ制限が機能していることを確認
	if len(history.entries) != maxSize {
		t.Errorf("Expected history size %d, got %d", maxSize, len(history.entries))
	}

	// 最新のエントリが保持されていることを確認
	lastEntry := history.entries[len(history.entries)-1]
	if lastEntry != "command1999" {
		t.Errorf("Expected last entry to be 'command1999', got %q", lastEntry)
	}
}

func BenchmarkHistory_Add(b *testing.B) {
	history := NewHistory(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		history.Add(fmt.Sprintf("command%d", i))
	}
}

func BenchmarkCompleter_GetSuggestions(b *testing.B) {
	completer := NewCompleter("/test")

	testInputs := []string{
		"/h",
		"help",
		"build",
		"test",
		"./src/",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := testInputs[i%len(testInputs)]
		completer.GetSuggestions(input)
	}
}

// UTF-8処理のテスト（実際のキーボード入力は困難なため基本的な検証のみ）
func TestReader_UTF8Support(t *testing.T) {
	reader := NewReader()

	// UTF-8文字を含む履歴の追加テスト
	utf8Commands := []string{
		"echo 'こんにちは'",
		"grep 'テスト' file.txt",
		"find . -name '*.日本語'",
	}

	for _, cmd := range utf8Commands {
		reader.history.Add(cmd)
	}

	// 履歴にUTF-8文字が正しく保存されているかチェック
	for i, expected := range utf8Commands {
		if reader.history.entries[i] != expected {
			t.Errorf("Expected UTF-8 command %q, got %q", expected, reader.history.entries[i])
		}
	}
}

func TestReader_ErrorHandling(t *testing.T) {
	reader := NewReader()

	// rawModeが無効な状態でのクリーンアップテスト
	err := reader.Close()
	if err != nil {
		t.Errorf("Expected no error when closing inactive reader, got %v", err)
	}

	// 複数回のClose()呼び出しテスト
	err = reader.Close()
	if err != nil {
		t.Errorf("Expected no error on multiple Close() calls, got %v", err)
	}
}

// 実際の端末入力のシミュレーション（ユニットテストレベル）
func TestReader_InputProcessing(t *testing.T) {
	reader := NewReader()
	reader.SetPrompt("test> ")

	tests := []struct {
		name        string
		input       string
		expectEmpty bool
	}{
		{
			name:        "Regular command",
			input:       "help",
			expectEmpty: false,
		},
		{
			name:        "Empty input",
			input:       "",
			expectEmpty: true,
		},
		{
			name:        "Whitespace only",
			input:       "   ",
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 履歴に追加して処理をテスト
			reader.history.Add(tt.input)

			if tt.expectEmpty {
				// 空の入力は履歴に追加されないはず
				lastIndex := len(reader.history.entries) - 1
				if lastIndex >= 0 && reader.history.entries[lastIndex] == strings.TrimSpace(tt.input) {
					t.Errorf("Expected empty input to not be added to history")
				}
			} else {
				// 通常の入力は履歴に追加されるはず
				lastIndex := len(reader.history.entries) - 1
				if lastIndex < 0 || reader.history.entries[lastIndex] != strings.TrimSpace(tt.input) {
					t.Errorf("Expected input %q to be added to history", strings.TrimSpace(tt.input))
				}
			}
		})
	}
}

func TestCompleter_CommandCompletion(t *testing.T) {
	completer := NewCompleter("/test")

	// 一般的なコマンドの補完テスト
	tests := []struct {
		input      string
		shouldFind string
	}{
		{"h", "help"},
		{"bu", "build"},
		{"t", "test"},
		{"st", "status"},
		{"a", "analyze"},
	}

	for _, tt := range tests {
		t.Run("Complete "+tt.input, func(t *testing.T) {
			suggestions := completer.GetSuggestions(tt.input)
			found := false
			for _, suggestion := range suggestions {
				if strings.Contains(suggestion, tt.shouldFind) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected to find %q in suggestions for %q, got %v", tt.shouldFind, tt.input, suggestions)
			}
		})
	}
}

func TestReader_SecurityIntegration(t *testing.T) {
	reader := NewReader()

	t.Run("SetClientID", func(t *testing.T) {
		testClientID := "test-integration-client"
		reader.SetClientID(testClientID)

		if reader.clientID != testClientID {
			t.Errorf("Expected client ID to be %q, got %q", testClientID, reader.clientID)
		}
	})

	t.Run("DisableSecurity", func(t *testing.T) {
		reader.DisableSecurity()

		if reader.securityValidator != nil {
			t.Error("Expected security validator to be nil after disable")
		}
	})

	t.Run("EnableSecurity", func(t *testing.T) {
		reader.EnableSecurity()

		if reader.securityValidator == nil {
			t.Error("Expected security validator to be non-nil after enable")
		}
	})

	t.Run("UpdateSecuritySettings", func(t *testing.T) {
		reader.EnableSecurity() // セキュリティを有効化

		// 設定更新
		reader.UpdateSecuritySettings(4096, 50)

		// 設定が反映されているかは内部的な確認なので、
		// エラーが発生しないことを確認
		if reader.securityValidator == nil {
			t.Error("Security validator should not be nil")
		}
	})
}

func TestReader_SecurityValidation(t *testing.T) {
	reader := NewReader()
	reader.SetClientID("test-validation-client")

	// セキュリティが無効の場合はバリデーションをスキップ
	reader.DisableSecurity()

	// 通常は危険とされる入力もセキュリティ無効時は通る
	dangerousInput := "test\x00dangerous"
	reader.history.Add(dangerousInput)

	// 履歴に追加されることを確認
	if len(reader.history.entries) == 0 {
		t.Error("Expected input to be added to history when security is disabled")
	}

	// セキュリティ有効化後のテスト
	reader.EnableSecurity()
	reader.history = NewHistory(100) // 履歴リセット

	// 安全な入力は通る
	safeInput := "vyb build"
	reader.history.Add(safeInput)

	if len(reader.history.entries) != 1 {
		t.Error("Expected safe input to be added to history")
	}

	if reader.history.entries[0] != safeInput {
		t.Errorf("Expected %q in history, got %q", safeInput, reader.history.entries[0])
	}
}

func TestReader_PerformanceOptimization(t *testing.T) {
	reader := NewReader()

	t.Run("EnableOptimization", func(t *testing.T) {
		reader.EnableOptimization()

		if !reader.enableOptimization {
			t.Error("Expected optimization to be enabled")
		}

		if reader.perfOptimizer == nil {
			t.Error("Expected non-nil performance optimizer")
		}
	})

	t.Run("DisableOptimization", func(t *testing.T) {
		reader.DisableOptimization()

		if reader.enableOptimization {
			t.Error("Expected optimization to be disabled")
		}
	})

	t.Run("GetPerformanceMetrics", func(t *testing.T) {
		reader.EnableOptimization()

		metrics := reader.GetPerformanceMetrics()
		if metrics == nil {
			t.Error("Expected non-nil metrics")
		}

		// 基本的なメトリクスが存在することを確認
		if _, exists := metrics["requests_total"]; !exists {
			t.Error("Expected requests_total metric")
		}
	})
}

func TestReader_AdvancedCompletion(t *testing.T) {
	reader := NewReader()

	// 高度な補完機能が組み込まれていることを確認
	if reader.completer.advancedCompleter == nil {
		t.Error("Expected non-nil advanced completer")
	}

	// 補完機能のプロジェクト解析が動作することを確認
	analyzer := reader.completer.advancedCompleter.projectAnalyzer
	if analyzer == nil {
		t.Error("Expected non-nil project analyzer")
	}

	if analyzer.projectType == "" {
		t.Error("Expected project type to be detected")
	}
}

func TestReader_Close(t *testing.T) {
	reader := NewReader()

	// クローズ処理でパフォーマンス最適化システムが停止されることを確認
	err := reader.Close()
	if err != nil {
		t.Errorf("Unexpected error during close: %v", err)
	}

	// 複数回のClose()呼び出しが安全であることを確認
	err = reader.Close()
	if err != nil {
		t.Errorf("Expected no error on multiple Close() calls, got %v", err)
	}
}
