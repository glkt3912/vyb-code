package security

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// エラーカテゴリ
type ErrorCategory string

const (
	ErrorCategoryUnknown       ErrorCategory = "unknown"
	ErrorCategoryValidation    ErrorCategory = "validation"
	ErrorCategorySecurity      ErrorCategory = "security"
	ErrorCategoryPermission    ErrorCategory = "permission"
	ErrorCategoryTimeout       ErrorCategory = "timeout"
	ErrorCategoryNetwork       ErrorCategory = "network"
	ErrorCategoryFileSystem    ErrorCategory = "filesystem"
	ErrorCategoryLLM           ErrorCategory = "llm"
	ErrorCategoryConfiguration ErrorCategory = "configuration"
	ErrorCategorySystem        ErrorCategory = "system"
)

// エラー重要度
type ErrorSeverity string

const (
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// 復旧戦略
type RecoveryStrategy string

const (
	RecoveryStrategyNone       RecoveryStrategy = "none"
	RecoveryStrategyRetry      RecoveryStrategy = "retry"
	RecoveryStrategyFallback   RecoveryStrategy = "fallback"
	RecoveryStrategyUserAction RecoveryStrategy = "user_action"
	RecoveryStrategyRestart    RecoveryStrategy = "restart"
)

// 拡張エラー情報
type EnhancedError struct {
	OriginalError    error             `json:"original_error"`
	Category         ErrorCategory     `json:"category"`
	Severity         ErrorSeverity     `json:"severity"`
	Code             string            `json:"code"`
	Message          string            `json:"message"`
	UserMessage      string            `json:"user_message"`      // ユーザー向けメッセージ
	TechnicalDetails string            `json:"technical_details"` // 技術的詳細
	Context          map[string]string `json:"context"`           // エラーコンテキスト
	Timestamp        time.Time         `json:"timestamp"`
	StackTrace       string            `json:"stack_trace"`
	RecoveryStrategy RecoveryStrategy  `json:"recovery_strategy"`
	RecoverySteps    []string          `json:"recovery_steps"`
	RelatedErrors    []string          `json:"related_errors"` // 関連するエラーのID
	RetryCount       int               `json:"retry_count"`
	MaxRetries       int               `json:"max_retries"`
	LastRetry        time.Time         `json:"last_retry"`
}

// エラーハンドラー
type ErrorHandler struct {
	errorRegistry map[string]*EnhancedError
	logger        Logger // ログ機能
}

// ログインターフェース
type Logger interface {
	Error(message string, context map[string]interface{})
	Warn(message string, context map[string]interface{})
	Info(message string, context map[string]interface{})
	Debug(message string, context map[string]interface{})
}

// 新しいエラーハンドラーを作成
func NewErrorHandler(logger Logger) *ErrorHandler {
	return &ErrorHandler{
		errorRegistry: make(map[string]*EnhancedError),
		logger:        logger,
	}
}

// エラーを拡張情報付きで処理
func (eh *ErrorHandler) HandleError(err error, category ErrorCategory, severity ErrorSeverity) *EnhancedError {
	enhancedErr := &EnhancedError{
		OriginalError:    err,
		Category:         category,
		Severity:         severity,
		Code:             eh.generateErrorCode(category, severity),
		Message:          err.Error(),
		UserMessage:      eh.generateUserMessage(err, category),
		TechnicalDetails: eh.generateTechnicalDetails(err),
		Context:          eh.captureContext(),
		Timestamp:        time.Now(),
		StackTrace:       eh.captureStackTrace(),
		RecoveryStrategy: eh.determineRecoveryStrategy(category, severity),
		RecoverySteps:    eh.generateRecoverySteps(err, category),
		RelatedErrors:    []string{},
		RetryCount:       0,
		MaxRetries:       eh.getMaxRetries(category, severity),
	}

	// エラーレジストリに登録
	eh.errorRegistry[enhancedErr.Code] = enhancedErr

	// ログ出力
	eh.logError(enhancedErr)

	return enhancedErr
}

// エラーコードを生成
func (eh *ErrorHandler) generateErrorCode(category ErrorCategory, severity ErrorSeverity) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s_%s_%d", strings.ToUpper(string(category)), strings.ToUpper(string(severity)), timestamp)
}

// ユーザー向けメッセージを生成
func (eh *ErrorHandler) generateUserMessage(err error, category ErrorCategory) string {
	switch category {
	case ErrorCategorySecurity:
		return "セキュリティ上の理由により、この操作は実行できませんでした。"
	case ErrorCategoryPermission:
		return "この操作を実行する権限がありません。ファイルやディレクトリのアクセス権限を確認してください。"
	case ErrorCategoryTimeout:
		return "操作がタイムアウトしました。ネットワーク接続やシステムの負荷を確認してください。"
	case ErrorCategoryNetwork:
		return "ネットワークエラーが発生しました。インターネット接続を確認してください。"
	case ErrorCategoryFileSystem:
		return "ファイルシステムエラーが発生しました。ファイルパスやディスク容量を確認してください。"
	case ErrorCategoryLLM:
		return "AI言語モデルとの通信でエラーが発生しました。設定を確認するか、しばらく時間をおいて再試行してください。"
	case ErrorCategoryConfiguration:
		return "設定に問題があります。設定ファイルを確認してください。"
	case ErrorCategoryValidation:
		return "入力データに問題があります。入力内容を確認してください。"
	default:
		return "予期しないエラーが発生しました。サポートにお問い合わせください。"
	}
}

// 技術的詳細を生成
func (eh *ErrorHandler) generateTechnicalDetails(err error) string {
	return fmt.Sprintf("Error Type: %T, Error Message: %s", err, err.Error())
}

// コンテキスト情報を取得
func (eh *ErrorHandler) captureContext() map[string]string {
	context := make(map[string]string)

	// ランタイム情報
	context["go_version"] = runtime.Version()
	context["goos"] = runtime.GOOS
	context["goarch"] = runtime.GOARCH
	context["num_goroutines"] = fmt.Sprintf("%d", runtime.NumGoroutine())

	// メモリ情報
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	context["memory_alloc"] = fmt.Sprintf("%d", m.Alloc)
	context["memory_total_alloc"] = fmt.Sprintf("%d", m.TotalAlloc)
	context["memory_sys"] = fmt.Sprintf("%d", m.Sys)
	context["memory_num_gc"] = fmt.Sprintf("%d", m.NumGC)

	return context
}

// スタックトレースを取得
func (eh *ErrorHandler) captureStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// 復旧戦略を決定
func (eh *ErrorHandler) determineRecoveryStrategy(category ErrorCategory, severity ErrorSeverity) RecoveryStrategy {
	switch category {
	case ErrorCategoryTimeout, ErrorCategoryNetwork:
		return RecoveryStrategyRetry
	case ErrorCategoryLLM:
		if severity == ErrorSeverityLow || severity == ErrorSeverityMedium {
			return RecoveryStrategyRetry
		}
		return RecoveryStrategyFallback
	case ErrorCategorySecurity, ErrorCategoryPermission:
		return RecoveryStrategyUserAction
	case ErrorCategoryConfiguration:
		return RecoveryStrategyUserAction
	case ErrorCategorySystem:
		if severity == ErrorSeverityCritical {
			return RecoveryStrategyRestart
		}
		return RecoveryStrategyFallback
	default:
		return RecoveryStrategyNone
	}
}

// 復旧手順を生成
func (eh *ErrorHandler) generateRecoverySteps(err error, category ErrorCategory) []string {
	switch category {
	case ErrorCategorySecurity:
		return []string{
			"セキュリティ設定を確認してください",
			"実行しようとしたコマンドが許可されているか確認してください",
			"ワークスペースディレクトリ内で作業していることを確認してください",
		}
	case ErrorCategoryPermission:
		return []string{
			"ファイルやディレクトリのアクセス権限を確認してください",
			"現在のユーザーに適切な権限があることを確認してください",
			"sudo権限が必要な場合は管理者に相談してください",
		}
	case ErrorCategoryTimeout:
		return []string{
			"ネットワーク接続を確認してください",
			"システムの負荷を確認してください",
			"タイムアウト設定を見直してください",
			"しばらく時間をおいて再試行してください",
		}
	case ErrorCategoryNetwork:
		return []string{
			"インターネット接続を確認してください",
			"プロキシ設定を確認してください",
			"ファイアウォール設定を確認してください",
			"DNSの設定を確認してください",
		}
	case ErrorCategoryFileSystem:
		return []string{
			"ファイルパスが正しいことを確認してください",
			"ディスク容量を確認してください",
			"ファイルが他のプロセスによって使用されていないか確認してください",
			"権限問題がないか確認してください",
		}
	case ErrorCategoryLLM:
		return []string{
			"LLMサーバー（Ollama等）が起動していることを確認してください",
			"モデルが正しくダウンロードされていることを確認してください",
			"設定ファイルのLLM設定を確認してください",
			"利用可能なメモリ容量を確認してください",
		}
	case ErrorCategoryConfiguration:
		return []string{
			"設定ファイル（~/.vyb/config.json）を確認してください",
			"設定値が正しい形式であることを確認してください",
			"設定ファイルの構文エラーがないか確認してください",
			"デフォルト設定にリセットを検討してください",
		}
	case ErrorCategoryValidation:
		return []string{
			"入力データの形式を確認してください",
			"必須フィールドが全て入力されていることを確認してください",
			"データの値の範囲を確認してください",
		}
	default:
		return []string{
			"エラーメッセージの詳細を確認してください",
			"ログファイルを確認してください",
			"問題が解決しない場合はサポートにお問い合わせください",
		}
	}
}

// 最大再試行回数を取得
func (eh *ErrorHandler) getMaxRetries(category ErrorCategory, severity ErrorSeverity) int {
	switch category {
	case ErrorCategoryTimeout, ErrorCategoryNetwork:
		return 3
	case ErrorCategoryLLM:
		if severity == ErrorSeverityHigh || severity == ErrorSeverityCritical {
			return 1
		}
		return 2
	case ErrorCategorySecurity, ErrorCategoryPermission:
		return 0 // セキュリティエラーは再試行しない
	default:
		return 1
	}
}

// エラーをログに出力
func (eh *ErrorHandler) logError(err *EnhancedError) {
	logContext := map[string]interface{}{
		"error_code":        err.Code,
		"category":          err.Category,
		"severity":          err.Severity,
		"message":           err.Message,
		"recovery_strategy": err.RecoveryStrategy,
		"retry_count":       err.RetryCount,
		"max_retries":       err.MaxRetries,
	}

	switch err.Severity {
	case ErrorSeverityCritical, ErrorSeverityHigh:
		eh.logger.Error(err.UserMessage, logContext)
	case ErrorSeverityMedium:
		eh.logger.Warn(err.UserMessage, logContext)
	default:
		eh.logger.Info(err.UserMessage, logContext)
	}
}

// エラーの再試行
func (eh *ErrorHandler) RetryError(errorCode string) (*EnhancedError, bool) {
	err, exists := eh.errorRegistry[errorCode]
	if !exists {
		return nil, false
	}

	if err.RetryCount >= err.MaxRetries {
		return err, false
	}

	if err.RecoveryStrategy != RecoveryStrategyRetry {
		return err, false
	}

	// 再試行間隔の制御（指数バックオフ）
	retryInterval := time.Duration(1<<uint(err.RetryCount)) * time.Second
	if time.Since(err.LastRetry) < retryInterval {
		return err, false
	}

	err.RetryCount++
	err.LastRetry = time.Now()

	eh.logger.Info("エラーを再試行します", map[string]interface{}{
		"error_code":  errorCode,
		"retry_count": err.RetryCount,
		"max_retries": err.MaxRetries,
	})

	return err, true
}

// エラー統計情報を取得
func (eh *ErrorHandler) GetErrorStats() map[string]interface{} {
	stats := make(map[string]interface{})

	categoryCounts := make(map[ErrorCategory]int)
	severityCounts := make(map[ErrorSeverity]int)
	totalErrors := len(eh.errorRegistry)

	for _, err := range eh.errorRegistry {
		categoryCounts[err.Category]++
		severityCounts[err.Severity]++
	}

	stats["total_errors"] = totalErrors
	stats["category_breakdown"] = categoryCounts
	stats["severity_breakdown"] = severityCounts

	return stats
}

// エラー履歴をクリア
func (eh *ErrorHandler) ClearErrorHistory() {
	eh.errorRegistry = make(map[string]*EnhancedError)
	eh.logger.Info("エラー履歴をクリアしました", nil)
}

// 特定カテゴリのエラーを取得
func (eh *ErrorHandler) GetErrorsByCategory(category ErrorCategory) []*EnhancedError {
	var errors []*EnhancedError
	for _, err := range eh.errorRegistry {
		if err.Category == category {
			errors = append(errors, err)
		}
	}
	return errors
}

// 最近のエラーを取得
func (eh *ErrorHandler) GetRecentErrors(limit int) []*EnhancedError {
	var errors []*EnhancedError
	for _, err := range eh.errorRegistry {
		errors = append(errors, err)
	}

	// タイムスタンプでソート（最新順）
	for i := 0; i < len(errors)-1; i++ {
		for j := i + 1; j < len(errors); j++ {
			if errors[i].Timestamp.Before(errors[j].Timestamp) {
				errors[i], errors[j] = errors[j], errors[i]
			}
		}
	}

	if limit > 0 && len(errors) > limit {
		errors = errors[:limit]
	}

	return errors
}

// エラーの詳細表示用文字列を生成
func (err *EnhancedError) String() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("エラーコード: %s\n", err.Code))
	builder.WriteString(fmt.Sprintf("カテゴリ: %s\n", err.Category))
	builder.WriteString(fmt.Sprintf("重要度: %s\n", err.Severity))
	builder.WriteString(fmt.Sprintf("メッセージ: %s\n", err.UserMessage))
	builder.WriteString(fmt.Sprintf("発生時刻: %s\n", err.Timestamp.Format("2006-01-02 15:04:05")))

	if len(err.RecoverySteps) > 0 {
		builder.WriteString("\n復旧手順:\n")
		for i, step := range err.RecoverySteps {
			builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
	}

	if err.RetryCount > 0 {
		builder.WriteString(fmt.Sprintf("\n再試行回数: %d/%d\n", err.RetryCount, err.MaxRetries))
	}

	return builder.String()
}
