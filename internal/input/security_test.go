package input

import (
	"strings"
	"testing"
	"time"
)

func TestSecurityValidator_SanitizeInput(t *testing.T) {
	validator := NewSecurityValidator()

	tests := []struct {
		name        string
		input       string
		expected    string
		shouldError bool
	}{
		{
			name:        "Normal text",
			input:       "hello world",
			expected:    "hello world",
			shouldError: false,
		},
		{
			name:        "Japanese text",
			input:       "こんにちは世界",
			expected:    "こんにちは世界",
			shouldError: false,
		},
		{
			name:        "With allowed control chars",
			input:       "line1\nline2\ttab",
			expected:    "line1\nline2\ttab",
			shouldError: false,
		},
		{
			name:        "With dangerous control chars",
			input:       "hello\x00world\x01test",
			expected:    "helloworldtest", // NULL文字削除、SOH削除
			shouldError: false,
		},
		{
			name:        "Allowed ANSI escape sequences",
			input:       "\033[32mgreen\033[0m",
			expected:    "\033[32mgreen\033[0m",
			shouldError: false,
		},
		{
			name:        "Dangerous escape sequences",
			input:       "\033]0;title\007",
			expected:    "]0;title\a", // 不正なエスケープシーケンスの一部のみ削除
			shouldError: false,
		},
		{
			name:        "Too long input",
			input:       strings.Repeat("a", MaxInputLength+1),
			expected:    "",
			shouldError: true,
		},
		{
			name:        "Mixed valid and invalid",
			input:       "valid\x05invalid\x06text",
			expected:    "validinvalidtext", // ENQ, ACK削除
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.SanitizeInput(tt.input)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSecurityValidator_CheckRateLimit(t *testing.T) {
	validator := NewSecurityValidator()
	clientID := "test-client"

	// 最初のリクエスト群は成功するはず
	for i := 0; i < MaxRequestsPerSec; i++ {
		err := validator.CheckRateLimit(clientID)
		if err != nil {
			t.Errorf("Request %d should succeed, got error: %v", i+1, err)
		}
	}

	// 制限を超えるリクエストは失敗するはず
	err := validator.CheckRateLimit(clientID)
	if err == nil {
		t.Errorf("Request over limit should fail")
	}

	// 時間経過後は再びリクエストできるはず
	validator.lastReset = time.Now().Add(-2 * RateLimitPeriod) // 期間経過をシミュレート
	err = validator.CheckRateLimit(clientID)
	if err != nil {
		t.Errorf("Request after period reset should succeed, got error: %v", err)
	}
}

func TestSecurityValidator_ValidateBufferSize(t *testing.T) {
	validator := NewSecurityValidator()

	tests := []struct {
		name        string
		input       string
		maxSize     int
		shouldError bool
	}{
		{
			name:        "Normal size",
			input:       "hello",
			maxSize:     100,
			shouldError: false,
		},
		{
			name:        "At limit",
			input:       strings.Repeat("a", 100),
			maxSize:     100,
			shouldError: false,
		},
		{
			name:        "Over limit",
			input:       strings.Repeat("a", 101),
			maxSize:     100,
			shouldError: true,
		},
		{
			name:        "Japanese characters",
			input:       strings.Repeat("あ", 50), // 各文字3バイト
			maxSize:     200,
			shouldError: false,
		},
		{
			name:        "Japanese over limit",
			input:       strings.Repeat("あ", 100), // 300バイト
			maxSize:     200,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateBufferSize(tt.input, tt.maxSize)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSecurityValidator_ValidateLineLength(t *testing.T) {
	validator := NewSecurityValidator()

	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "Single short line",
			input:       "hello",
			shouldError: false,
		},
		{
			name:        "Multiple short lines",
			input:       "line1\nline2\nline3",
			shouldError: false,
		},
		{
			name:        "Line at limit",
			input:       strings.Repeat("a", MaxLineLength),
			shouldError: false,
		},
		{
			name:        "Line over limit",
			input:       strings.Repeat("a", MaxLineLength+1),
			shouldError: true,
		},
		{
			name:        "One line over limit in multiple",
			input:       "short\n" + strings.Repeat("a", MaxLineLength+1) + "\nshort",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateLineLength(tt.input)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSecurityValidator_ValidateCommand(t *testing.T) {
	validator := NewSecurityValidator()

	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "Safe command",
			input:       "ls -la",
			shouldError: false,
		},
		{
			name:        "Safe vyb command",
			input:       "vyb build",
			shouldError: false,
		},
		{
			name:        "Command injection attempt",
			input:       "ls && rm -rf /",
			shouldError: true,
		},
		{
			name:        "Pipe command",
			input:       "cat file | grep pattern",
			shouldError: true,
		},
		{
			name:        "Output redirection",
			input:       "echo data > file",
			shouldError: true,
		},
		{
			name:        "Dangerous rm command",
			input:       "rm important_file",
			shouldError: true,
		},
		{
			name:        "Sudo attempt",
			input:       "sudo rm file",
			shouldError: true,
		},
		{
			name:        "Python execution",
			input:       "python -c 'import os; os.system(\"ls\")'",
			shouldError: true,
		},
		{
			name:        "Network command",
			input:       "curl http://malicious.com",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCommand(tt.input)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error but got none for input: %s", tt.input)
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error for safe input %q: %v", tt.input, err)
			}
		})
	}
}

func TestSecurityValidator_ValidatePath(t *testing.T) {
	validator := NewSecurityValidator()

	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "Safe relative path",
			input:       "src/main.go",
			shouldError: false,
		},
		{
			name:        "Safe absolute path in project",
			input:       "/home/user/project/file.txt",
			shouldError: false,
		},
		{
			name:        "Directory traversal attempt",
			input:       "../../../etc/passwd",
			shouldError: true,
		},
		{
			name:        "Windows directory traversal",
			input:       "..\\..\\windows\\system32",
			shouldError: true,
		},
		{
			name:        "System directory access",
			input:       "/etc/shadow",
			shouldError: true,
		},
		{
			name:        "Proc filesystem access",
			input:       "/proc/1/mem",
			shouldError: true,
		},
		{
			name:        "Windows system directory",
			input:       "C:\\Windows\\System32\\cmd.exe",
			shouldError: true,
		},
		{
			name:        "SSH directory access",
			input:       "~/.ssh/id_rsa",
			shouldError: true,
		},
		{
			name:        "AWS credentials",
			input:       "~/.aws/credentials",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePath(tt.input)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error but got none for path: %s", tt.input)
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error for safe path %q: %v", tt.input, err)
			}
		})
	}
}

func TestSecurityValidator_ValidateInput(t *testing.T) {
	validator := NewSecurityValidator()
	clientID := "test-client"

	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "Normal safe input",
			input:       "vyb build",
			shouldError: false,
		},
		{
			name:        "Japanese input",
			input:       "こんにちは",
			shouldError: false,
		},
		{
			name:        "Input with safe path",
			input:       "read src/main.go",
			shouldError: false,
		},
		{
			name:        "Too long input",
			input:       strings.Repeat("a", MaxInputLength+1),
			shouldError: true,
		},
		{
			name:        "Input with dangerous path",
			input:       "cat ../../../etc/passwd",
			shouldError: true,
		},
		{
			name:        "Command injection attempt",
			input:       "ls && rm file",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateInput(tt.input, clientID)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error but got none for input: %s", tt.input)
			}

			if !tt.shouldError {
				if err != nil {
					t.Errorf("Unexpected error for safe input %q: %v", tt.input, err)
				} else if result == "" {
					t.Errorf("Expected non-empty result for valid input")
				}
			}
		})
	}
}

func TestSecurityValidator_isDangerousControlChar(t *testing.T) {
	validator := NewSecurityValidator()

	tests := []struct {
		name     string
		char     rune
		expected bool
	}{
		{name: "Tab (allowed)", char: '\t', expected: false},
		{name: "Newline (allowed)", char: '\n', expected: false},
		{name: "Carriage return (allowed)", char: '\r', expected: false},
		{name: "Backspace (allowed)", char: '\b', expected: false},
		{name: "Escape (allowed)", char: '\033', expected: false},
		{name: "NULL (dangerous)", char: 0x00, expected: true},
		{name: "SOH (dangerous)", char: 0x01, expected: true},
		{name: "STX (dangerous)", char: 0x02, expected: true},
		{name: "Normal ASCII letter", char: 'A', expected: false},
		{name: "Normal ASCII digit", char: '5', expected: false},
		{name: "Japanese character", char: 'あ', expected: false},
		{name: "DEL (dangerous)", char: 0x7F, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isDangerousControlChar(tt.char)
			if result != tt.expected {
				t.Errorf("Expected %v for char %U, got %v", tt.expected, tt.char, result)
			}
		})
	}
}

func TestSecurityValidator_isAllowedEscapeSequence(t *testing.T) {
	validator := NewSecurityValidator()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "Up arrow", input: "\033[A", expected: true},
		{name: "Down arrow", input: "\033[B", expected: true},
		{name: "Right arrow", input: "\033[C", expected: true},
		{name: "Left arrow", input: "\033[D", expected: true},
		{name: "Clear screen", input: "\033[2J", expected: true},
		{name: "Color code red", input: "\033[31m", expected: true},
		{name: "Color code with multiple values", input: "\033[1;32m", expected: true},
		{name: "Reset color", input: "\033[0m", expected: true},
		{name: "Dangerous title sequence", input: "\033]0;title\007", expected: false},
		{name: "Invalid sequence", input: "\033[Z", expected: false},
		{name: "Short sequence", input: "\033", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isAllowedEscapeSequence(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v for sequence %q, got %v", tt.expected, tt.input, result)
			}
		})
	}
}

func TestSecurityValidator_UpdateSecuritySettings(t *testing.T) {
	validator := NewSecurityValidator()

	// 初期設定確認
	originalLimit := validator.bufferLimit

	// 設定更新
	newLimit := 4096
	validator.UpdateSecuritySettings(newLimit, 50)

	// 設定が更新されたか確認
	if validator.bufferLimit != newLimit {
		t.Errorf("Expected buffer limit to be %d, got %d", newLimit, validator.bufferLimit)
	}

	// 不正な値の場合は更新されないはず
	validator.UpdateSecuritySettings(-1, -1)
	if validator.bufferLimit != newLimit {
		t.Errorf("Buffer limit should not change with invalid value")
	}

	// 最大値を超える場合は更新されないはず
	validator.UpdateSecuritySettings(MaxInputLength+1, 50)
	if validator.bufferLimit != newLimit {
		t.Errorf("Buffer limit should not exceed maximum")
	}

	// 元の設定を復元（他のテストに影響しないように）
	validator.bufferLimit = originalLimit
}

// パフォーマンステスト
func BenchmarkSecurityValidator_SanitizeInput(b *testing.B) {
	validator := NewSecurityValidator()
	input := strings.Repeat("Hello World! ", 100) // 約1.2KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.SanitizeInput(input)
	}
}

func BenchmarkSecurityValidator_ValidateInput(b *testing.B) {
	validator := NewSecurityValidator()
	input := "vyb build test command"
	clientID := "benchmark-client"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// レート制限をリセットして測定に影響しないようにする
		if i%MaxRequestsPerSec == 0 {
			validator.requestCounts = make(map[string]int)
			validator.lastReset = time.Now()
		}
		validator.ValidateInput(input, clientID)
	}
}