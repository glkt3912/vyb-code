package recovery

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/glkt/vyb-code/internal/config"
)

func TestSystem_Creation(t *testing.T) {
	cfg := &config.Config{
		Model:    "qwen2.5-coder:14b",
		Provider: "ollama",
	}
	
	system := NewSystem(cfg)
	
	if system == nil {
		t.Error("Expected non-nil recovery system")
	}
	
	if system.config != cfg {
		t.Error("Expected config to be set correctly")
	}
	
	if system.fallbackModel != "qwen2.5-coder:7b" {
		t.Errorf("Expected fallback model 'qwen2.5-coder:7b', got %q", system.fallbackModel)
	}
	
	if system.retryAttempts != 0 {
		t.Errorf("Expected retryAttempts to be 0 initially, got %d", system.retryAttempts)
	}
	
	if system.maxRetries != 3 {
		t.Errorf("Expected maxRetries to be 3, got %d", system.maxRetries)
	}
	
	if system.retryDelay != 2*time.Second {
		t.Errorf("Expected retryDelay to be 2s, got %v", system.retryDelay)
	}
}

func TestSystem_AnalyzeError(t *testing.T) {
	cfg := &config.Config{Model: "test-model"}
	system := NewSystem(cfg)
	
	tests := []struct {
		name           string
		error          error
		expectedType   ErrorType
		expectedCanRecover bool
		expectedSeverity   int
	}{
		{
			name:           "Connection error",
			error:          errors.New("connection refused"),
			expectedType:   ErrorConnection,
			expectedCanRecover: true,
			expectedSeverity:   8,
		},
		{
			name:           "Dial error",
			error:          errors.New("dial tcp: connection timeout"),
			expectedType:   ErrorConnection,
			expectedCanRecover: true,
			expectedSeverity:   8,
		},
		{
			name:           "Model not found",
			error:          errors.New("model not found"),
			expectedType:   ErrorModel,
			expectedCanRecover: true,
			expectedSeverity:   7,
		},
		{
			name:           "Model not available",
			error:          errors.New("model not available"),
			expectedType:   ErrorModel,
			expectedCanRecover: true,
			expectedSeverity:   7,
		},
		{
			name:           "Timeout error",
			error:          errors.New("request timeout"),
			expectedType:   ErrorTimeout,
			expectedCanRecover: true,
			expectedSeverity:   5,
		},
		{
			name:           "Deadline exceeded",
			error:          errors.New("context deadline exceeded"),
			expectedType:   ErrorTimeout,
			expectedCanRecover: true,
			expectedSeverity:   5,
		},
		{
			name:           "Rate limit",
			error:          errors.New("rate limit exceeded"),
			expectedType:   ErrorRateLimit,
			expectedCanRecover: true,
			expectedSeverity:   4,
		},
		{
			name:           "Too many requests",
			error:          errors.New("too many requests"),
			expectedType:   ErrorRateLimit,
			expectedCanRecover: true,
			expectedSeverity:   4,
		},
		{
			name:           "Authentication error",
			error:          errors.New("unauthorized access"),
			expectedType:   ErrorAuthentication,
			expectedCanRecover: false,
			expectedSeverity:   9,
		},
		{
			name:           "Auth error",
			error:          errors.New("auth failed"),
			expectedType:   ErrorAuthentication,
			expectedCanRecover: false,
			expectedSeverity:   9,
		},
		{
			name:           "Unknown error",
			error:          errors.New("unexpected error occurred"),
			expectedType:   ErrorUnknown,
			expectedCanRecover: true,
			expectedSeverity:   6,
		},
		{
			name:           "Nil error",
			error:          nil,
			expectedType:   "",
			expectedCanRecover: false,
			expectedSeverity:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorInfo := system.AnalyzeError(tt.error)
			
			if tt.error == nil {
				if errorInfo != nil {
					t.Error("Expected nil errorInfo for nil error")
				}
				return
			}
			
			if errorInfo == nil {
				t.Error("Expected non-nil errorInfo for non-nil error")
				return
			}
			
			if errorInfo.Type != tt.expectedType {
				t.Errorf("Expected error type %v, got %v", tt.expectedType, errorInfo.Type)
			}
			
			if errorInfo.CanRecover != tt.expectedCanRecover {
				t.Errorf("Expected CanRecover %v, got %v", tt.expectedCanRecover, errorInfo.CanRecover)
			}
			
			if errorInfo.Severity != tt.expectedSeverity {
				t.Errorf("Expected severity %d, got %d", tt.expectedSeverity, errorInfo.Severity)
			}
			
			if errorInfo.Original != tt.error {
				t.Errorf("Expected original error to be preserved")
			}
			
			if errorInfo.Suggestion == "" {
				t.Error("Expected non-empty suggestion")
			}
			
			if errorInfo.RetryAfter < 0 {
				t.Error("Expected non-negative RetryAfter")
			}
		})
	}
}

func TestSystem_ErrorTypeDescription(t *testing.T) {
	system := NewSystem(&config.Config{})
	
	tests := []struct {
		errorType   ErrorType
		expected    string
	}{
		{ErrorConnection, "接続エラー"},
		{ErrorModel, "モデルエラー"},
		{ErrorTimeout, "タイムアウトエラー"},
		{ErrorRateLimit, "レート制限エラー"},
		{ErrorAuthentication, "認証エラー"},
		{ErrorUnknown, "不明なエラー"},
		{ErrorType("invalid"), "不明なエラー"}, // デフォルトケース
	}

	for _, tt := range tests {
		result := system.getErrorTypeDescription(tt.errorType)
		if result != tt.expected {
			t.Errorf("Expected description %q for %v, got %q", tt.expected, tt.errorType, result)
		}
	}
}

func TestSystem_Reset(t *testing.T) {
	system := NewSystem(&config.Config{})
	
	// リトライカウンターを設定
	system.retryAttempts = 5
	
	// リセット実行
	system.Reset()
	
	if system.retryAttempts != 0 {
		t.Errorf("Expected retryAttempts to be reset to 0, got %d", system.retryAttempts)
	}
}

func TestSystem_AttemptRecoveryCannotRecover(t *testing.T) {
	system := NewSystem(&config.Config{})
	
	// 回復不可能なエラー
	errorInfo := &ErrorInfo{
		Type:       ErrorAuthentication,
		Original:   errors.New("auth failed"),
		CanRecover: false,
		Suggestion: "認証設定を確認してください",
	}
	
	operationCalled := false
	operation := func() error {
		operationCalled = true
		return nil
	}
	
	err := system.AttemptRecovery(errorInfo, operation)
	
	// 操作が実行されないことを確認
	if operationCalled {
		t.Error("Expected operation not to be called for unrecoverable error")
	}
	
	// 元のエラーが返されることを確認
	if err != errorInfo.Original {
		t.Errorf("Expected original error to be returned, got %v", err)
	}
}

func TestSystem_AttemptRecoveryMaxRetriesExceeded(t *testing.T) {
	system := NewSystem(&config.Config{})
	
	// 最大リトライ回数を超えるように設定
	system.retryAttempts = system.maxRetries
	
	errorInfo := &ErrorInfo{
		Type:       ErrorConnection,
		Original:   errors.New("connection failed"),
		CanRecover: true,
		Suggestion: "Ollamaを起動してください",
	}
	
	operationCalled := false
	operation := func() error {
		operationCalled = true
		return nil
	}
	
	err := system.AttemptRecovery(errorInfo, operation)
	
	// 操作が実行されないことを確認
	if operationCalled {
		t.Error("Expected operation not to be called when max retries exceeded")
	}
	
	// エラーが返されることを確認
	if err == nil {
		t.Error("Expected error when max retries exceeded")
	}
}

func TestSystem_AttemptRecoverySuccessfulRetry(t *testing.T) {
	system := NewSystem(&config.Config{})
	
	errorInfo := &ErrorInfo{
		Type:       ErrorConnection,
		Original:   errors.New("connection failed"),
		CanRecover: true,
		Suggestion: "再試行中...",
		RetryAfter: 10 * time.Millisecond, // テスト用に短い時間
	}
	
	operationCallCount := 0
	operation := func() error {
		operationCallCount++
		return nil // 成功
	}
	
	start := time.Now()
	err := system.AttemptRecovery(errorInfo, operation)
	duration := time.Since(start)
	
	if err != nil {
		t.Errorf("Expected no error for successful retry, got %v", err)
	}
	
	if operationCallCount != 1 {
		t.Errorf("Expected operation to be called once, got %d", operationCallCount)
	}
	
	// リトライカウンターがリセットされることを確認
	if system.retryAttempts != 0 {
		t.Errorf("Expected retryAttempts to be reset after success, got %d", system.retryAttempts)
	}
	
	// 遅延が適用されることを確認
	if duration < errorInfo.RetryAfter {
		t.Errorf("Expected delay of at least %v, got %v", errorInfo.RetryAfter, duration)
	}
}

func TestSystem_AttemptRecoveryFailedRetry(t *testing.T) {
	system := NewSystem(&config.Config{})
	
	originalError := errors.New("connection failed")
	secondError := errors.New("still failing")
	
	errorInfo := &ErrorInfo{
		Type:       ErrorConnection,
		Original:   originalError,
		CanRecover: true,
		Suggestion: "再試行中...",
		RetryAfter: 1 * time.Millisecond,
	}
	
	operationCallCount := 0
	operation := func() error {
		operationCallCount++
		if operationCallCount == 1 {
			return secondError // 再度失敗
		}
		return nil // 2回目は成功
	}
	
	// 最大リトライ回数を1に設定してテスト簡略化
	system.maxRetries = 1
	
	_ = system.AttemptRecovery(errorInfo, operation)
	
	// リトライが実行されることを確認
	if operationCallCount == 0 {
		t.Error("Expected operation to be called at least once")
	}
	
	// リトライカウンターが増加することを確認
	if system.retryAttempts == 0 {
		t.Error("Expected retryAttempts to be incremented")
	}
}

func TestSystem_ModelFallback(t *testing.T) {
	originalModel := "original-model"
	cfg := &config.Config{
		Model: originalModel,
	}
	system := NewSystem(cfg)
	
	errorInfo := &ErrorInfo{
		Type:       ErrorModel,
		Original:   errors.New("model not found"),
		CanRecover: true,
		Suggestion: "フォールバックモデルを試行",
	}
	
	operationCallCount := 0
	var usedModel string
	
	operation := func() error {
		operationCallCount++
		usedModel = cfg.Model
		return nil // 成功
	}
	
	err := system.AttemptRecovery(errorInfo, operation)
	
	if err != nil {
		t.Errorf("Expected no error for successful fallback, got %v", err)
	}
	
	if operationCallCount != 1 {
		t.Errorf("Expected operation to be called once, got %d", operationCallCount)
	}
	
	// フォールバックモデルが使用されることを確認
	if usedModel != system.fallbackModel {
		t.Errorf("Expected fallback model %q to be used, got %q", system.fallbackModel, usedModel)
	}
	
	// 元のモデル設定が復元されることを確認
	if cfg.Model != originalModel {
		t.Errorf("Expected original model %q to be restored, got %q", originalModel, cfg.Model)
	}
}

func TestSystem_ModelFallbackFailure(t *testing.T) {
	cfg := &config.Config{Model: "original-model"}
	system := NewSystem(cfg)
	
	errorInfo := &ErrorInfo{
		Type:       ErrorModel,
		Original:   errors.New("model not found"),
		CanRecover: true,
	}
	
	fallbackError := errors.New("fallback also failed")
	operation := func() error {
		return fallbackError
	}
	
	err := system.AttemptRecovery(errorInfo, operation)
	
	// フォールバックも失敗した場合のエラーハンドリング
	if err == nil {
		t.Error("Expected error when fallback fails")
	}
	
	if err != fallbackError {
		t.Errorf("Expected fallback error to be returned, got %v", err)
	}
}

func TestSystem_MaxRetriesHandling(t *testing.T) {
	system := NewSystem(&config.Config{})
	system.maxRetries = 2
	
	errorInfo := &ErrorInfo{
		Type:       ErrorConnection,
		Original:   errors.New("persistent connection error"),
		CanRecover: true,
		RetryAfter: 1 * time.Millisecond,
	}
	
	operationCallCount := 0
	operation := func() error {
		operationCallCount++
		// 常に失敗
		return errors.New("still failing")
	}
	
	err := system.AttemptRecovery(errorInfo, operation)
	
	// 最大リトライ回数に達することを確認
	if system.retryAttempts < system.maxRetries {
		t.Errorf("Expected to reach max retries %d, got %d", system.maxRetries, system.retryAttempts)
	}
	
	// エラーが返されることを確認
	if err == nil {
		t.Error("Expected error when max retries exceeded")
	}
}

func TestSystem_ErrorInfoValidation(t *testing.T) {
	cfg := &config.Config{Model: "test-model"}
	system := NewSystem(cfg)
	
	testError := errors.New("test error")
	errorInfo := system.AnalyzeError(testError)
	
	// ErrorInfo の基本フィールドが設定されることを確認
	if errorInfo.Original != testError {
		t.Error("Expected original error to be preserved")
	}
	
	if errorInfo.Suggestion == "" {
		t.Error("Expected non-empty suggestion")
	}
	
	if errorInfo.Type == "" {
		t.Error("Expected non-empty error type")
	}
	
	if errorInfo.Severity < 1 || errorInfo.Severity > 10 {
		t.Errorf("Expected severity between 1-10, got %d", errorInfo.Severity)
	}
	
	if errorInfo.RetryAfter < 0 {
		t.Error("Expected non-negative RetryAfter duration")
	}
}

func TestSystem_SuggestionContent(t *testing.T) {
	cfg := &config.Config{Model: "test-model"}
	system := NewSystem(cfg)
	
	tests := []struct {
		errorType ErrorType
		keywords  []string
	}{
		{
			ErrorConnection, 
			[]string{"Ollama", "サーバー", "起動"},
		},
		{
			ErrorModel, 
			[]string{"モデル", "見つかりません", "インストール", cfg.Model},
		},
		{
			ErrorTimeout, 
			[]string{"タイムアウト", "ネットワーク", "状態"},
		},
		{
			ErrorRateLimit, 
			[]string{"レート制限", "待って", "再試行"},
		},
		{
			ErrorAuthentication, 
			[]string{"認証", "失敗", "APIキー", "設定"},
		},
		{
			ErrorUnknown, 
			[]string{"予期しない", "エラー", "ログ"},
		},
	}

	for _, tt := range tests {
		// 該当するエラーを作成
		var testError error
		switch tt.errorType {
		case ErrorConnection:
			testError = errors.New("connection refused")
		case ErrorModel:
			testError = errors.New("model not found")
		case ErrorTimeout:
			testError = errors.New("request timeout")
		case ErrorRateLimit:
			testError = errors.New("rate limit exceeded")
		case ErrorAuthentication:
			testError = errors.New("auth failed")
		default:
			testError = errors.New("unknown error")
		}
		
		errorInfo := system.AnalyzeError(testError)
		
		// 期待されるキーワードが含まれているかチェック
		for _, keyword := range tt.keywords {
			if !strings.Contains(errorInfo.Suggestion, keyword) {
				t.Errorf("Expected suggestion for %v to contain %q, got: %q", 
					tt.errorType, keyword, errorInfo.Suggestion)
			}
		}
	}
}

func TestSystem_RecursiveRetry(t *testing.T) {
	system := NewSystem(&config.Config{})
	system.maxRetries = 3
	
	errorInfo := &ErrorInfo{
		Type:       ErrorConnection,
		Original:   errors.New("connection failed"),
		CanRecover: true,
		RetryAfter: 1 * time.Millisecond,
	}
	
	operationCallCount := 0
	operation := func() error {
		operationCallCount++
		if operationCallCount < 3 {
			return errors.New("still failing")
		}
		return nil // 3回目で成功
	}
	
	err := system.AttemptRecovery(errorInfo, operation)
	
	if err != nil {
		t.Errorf("Expected eventual success, got %v", err)
	}
	
	// 3回呼び出されることを確認（初回 + 2回のリトライ）
	if operationCallCount != 3 {
		t.Errorf("Expected 3 operation calls, got %d", operationCallCount)
	}
	
	// リトライカウンターがリセットされることを確認
	if system.retryAttempts != 0 {
		t.Errorf("Expected retryAttempts to be reset after success, got %d", system.retryAttempts)
	}
}

func TestSystem_CaseSensitiveErrorDetection(t *testing.T) {
	system := NewSystem(&config.Config{})
	
	// 大文字小文字を含むエラーメッセージのテスト
	tests := []struct {
		errorMsg     string
		expectedType ErrorType
	}{
		{"Connection REFUSED", ErrorConnection},
		{"CONNECTION timeout", ErrorConnection},
		{"Model NOT FOUND", ErrorModel},
		{"MODEL not available", ErrorModel},
		{"TIMEOUT exceeded", ErrorTimeout},
		{"Request TIMEOUT", ErrorTimeout},
		{"RATE limit exceeded", ErrorRateLimit},
		{"Too Many REQUESTS", ErrorRateLimit},
		{"AUTH failed", ErrorAuthentication},
		{"UNAUTHORIZED access", ErrorAuthentication},
	}

	for _, tt := range tests {
		t.Run(tt.errorMsg, func(t *testing.T) {
			err := errors.New(tt.errorMsg)
			errorInfo := system.AnalyzeError(err)
			
			if errorInfo.Type != tt.expectedType {
				t.Errorf("Expected error type %v for %q, got %v", 
					tt.expectedType, tt.errorMsg, errorInfo.Type)
			}
		})
	}
}

func TestSystem_RetryDelayAccumulation(t *testing.T) {
	system := NewSystem(&config.Config{})
	
	errorInfo := &ErrorInfo{
		Type:       ErrorConnection,
		Original:   errors.New("connection failed"),
		CanRecover: true,
		RetryAfter: 25 * time.Millisecond,
	}
	
	failureCount := 0
	operation := func() error {
		failureCount++
		if failureCount < 3 {
			return errors.New("still failing")
		}
		return nil
	}
	
	start := time.Now()
	err := system.AttemptRecovery(errorInfo, operation)
	totalDuration := time.Since(start)
	
	if err != nil {
		t.Errorf("Expected eventual success, got %v", err)
	}
	
	// 複数回のリトライ遅延が蓄積されることを確認
	expectedMinDuration := errorInfo.RetryAfter * 2 // 2回のリトライ
	if totalDuration < expectedMinDuration {
		t.Errorf("Expected total duration at least %v, got %v", expectedMinDuration, totalDuration)
	}
}

// パフォーマンステスト
func TestSystem_PerformanceErrorAnalysis(t *testing.T) {
	system := NewSystem(&config.Config{Model: "test-model"})
	
	testErrors := []error{
		errors.New("connection refused"),
		errors.New("model not found"),
		errors.New("timeout occurred"),
		errors.New("rate limit exceeded"),
		errors.New("auth failed"),
		errors.New("unknown error"),
	}
	
	start := time.Now()
	
	for i := 0; i < 1000; i++ {
		err := testErrors[i%len(testErrors)]
		system.AnalyzeError(err)
	}
	
	duration := time.Since(start)
	
	// パフォーマンスチェック（1000回の分析が100ms以内）
	if duration > 100*time.Millisecond {
		t.Errorf("Error analysis took too long: %v", duration)
	}
}

// エッジケースのテスト
func TestSystem_EdgeCases(t *testing.T) {
	system := NewSystem(&config.Config{})
	
	t.Run("Empty error message", func(t *testing.T) {
		err := errors.New("")
		errorInfo := system.AnalyzeError(err)
		
		if errorInfo == nil {
			t.Error("Expected errorInfo for empty error message")
		}
		
		// 空のメッセージでも適切に分類されることを確認
		if errorInfo.Type == "" {
			t.Error("Expected error type to be determined for empty message")
		}
	})
	
	t.Run("Very long error message", func(t *testing.T) {
		longMessage := strings.Repeat("error ", 1000) + "connection refused"
		err := errors.New(longMessage)
		errorInfo := system.AnalyzeError(err)
		
		if errorInfo.Type != ErrorConnection {
			t.Errorf("Expected ErrorConnection for long message, got %v", errorInfo.Type)
		}
	})
	
	t.Run("Multiple error keywords", func(t *testing.T) {
		// 複数のエラータイプキーワードを含むメッセージ
		err := errors.New("connection timeout and model not found")
		errorInfo := system.AnalyzeError(err)
		
		// 最初にマッチしたタイプが返されることを確認
		if errorInfo.Type != ErrorConnection {
			t.Errorf("Expected first matched type (ErrorConnection), got %v", errorInfo.Type)
		}
	})
}

// ベンチマークテスト
func BenchmarkSystem_AnalyzeError(b *testing.B) {
	system := NewSystem(&config.Config{Model: "test-model"})
	err := errors.New("connection refused by server")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		system.AnalyzeError(err)
	}
}

func BenchmarkSystem_GetErrorTypeDescription(b *testing.B) {
	system := NewSystem(&config.Config{})
	errorTypes := []ErrorType{
		ErrorConnection, ErrorModel, ErrorTimeout, 
		ErrorRateLimit, ErrorAuthentication, ErrorUnknown,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		errorType := errorTypes[i%len(errorTypes)]
		system.getErrorTypeDescription(errorType)
	}
}