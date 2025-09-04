package recovery

import (
	"fmt"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/config"
)

// 回復システム
type System struct {
	config        *config.Config
	fallbackModel string
	retryAttempts int
	maxRetries    int
	retryDelay    time.Duration
}

// エラータイプ
type ErrorType string

const (
	ErrorConnection     ErrorType = "connection"
	ErrorModel          ErrorType = "model"
	ErrorTimeout        ErrorType = "timeout"
	ErrorRateLimit      ErrorType = "rate_limit"
	ErrorAuthentication ErrorType = "auth"
	ErrorUnknown        ErrorType = "unknown"
)

// エラー情報
type ErrorInfo struct {
	Type       ErrorType
	Original   error
	Severity   int
	Suggestion string
	CanRecover bool
	RetryAfter time.Duration
}

// 回復システムを作成
func NewSystem(cfg *config.Config) *System {
	return &System{
		config:        cfg,
		fallbackModel: "qwen2.5-coder:7b", // より軽量なフォールバックモデル
		retryAttempts: 0,
		maxRetries:    3,
		retryDelay:    2 * time.Second,
	}
}

// エラー分析と回復提案
func (r *System) AnalyzeError(err error) *ErrorInfo {
	if err == nil {
		return nil
	}

	errStr := strings.ToLower(err.Error())

	// 接続エラー（connection/dial/refusedが含まれる場合は優先）
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "dial") ||
		strings.Contains(errStr, "refused") {
		return &ErrorInfo{
			Type:       ErrorConnection,
			Original:   err,
			Severity:   8,
			Suggestion: "Ollamaサーバーが起動しているか確認してください。 `ollama serve` でサーバーを起動できます。",
			CanRecover: true,
			RetryAfter: 5 * time.Second,
		}
	}

	// タイムアウトエラー（connection関連でない場合）
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return &ErrorInfo{
			Type:       ErrorTimeout,
			Original:   err,
			Severity:   5,
			Suggestion: "タイムアウトが発生しました。ネットワークやサーバーの状態を確認してください。",
			CanRecover: true,
			RetryAfter: 3 * time.Second,
		}
	}

	// モデルエラー
	if strings.Contains(errStr, "model") && (strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "not available")) {
		return &ErrorInfo{
			Type:       ErrorModel,
			Original:   err,
			Severity:   7,
			Suggestion: fmt.Sprintf("モデル '%s' が見つかりません。`ollama pull %s` でインストールするか、別のモデルを試してください。", r.config.Model, r.config.Model),
			CanRecover: true,
			RetryAfter: 1 * time.Second,
		}
	}

	// レート制限エラー
	if strings.Contains(errStr, "rate") || strings.Contains(errStr, "too many") {
		return &ErrorInfo{
			Type:       ErrorRateLimit,
			Original:   err,
			Severity:   4,
			Suggestion: "レート制限に達しました。しばらく待ってから再試行してください。",
			CanRecover: true,
			RetryAfter: 10 * time.Second,
		}
	}

	// 認証エラー
	if strings.Contains(errStr, "auth") || strings.Contains(errStr, "unauthorized") {
		return &ErrorInfo{
			Type:       ErrorAuthentication,
			Original:   err,
			Severity:   9,
			Suggestion: "認証に失敗しました。APIキーや設定を確認してください。",
			CanRecover: false,
		}
	}

	// 未知のエラー
	return &ErrorInfo{
		Type:       ErrorUnknown,
		Original:   err,
		Severity:   6,
		Suggestion: "予期しないエラーが発生しました。ログを確認してください。",
		CanRecover: true,
		RetryAfter: 2 * time.Second,
	}
}

// 自動回復を試行
func (r *System) AttemptRecovery(errorInfo *ErrorInfo, operation func() error) error {
	if !errorInfo.CanRecover {
		return r.displayErrorGuidance(errorInfo)
	}

	// リトライ制限チェック
	if r.retryAttempts >= r.maxRetries {
		fmt.Printf("\033[31m❌ 最大再試行回数 (%d回) に達しました\033[0m\n", r.maxRetries)
		return r.displayErrorGuidance(errorInfo)
	}

	r.retryAttempts++

	// 回復メッセージ表示
	fmt.Printf("\033[33m🔄 自動回復試行中 (%d/%d)...\033[0m\n", r.retryAttempts, r.maxRetries)
	fmt.Printf("\033[90m   %s\033[0m\n", errorInfo.Suggestion)

	// 指定された時間だけ待機
	if errorInfo.RetryAfter > 0 {
		fmt.Printf("\033[90m   %v 待機中...\033[0m\n", errorInfo.RetryAfter)
		time.Sleep(errorInfo.RetryAfter)
	}

	// モデルフォールバック（モデルエラーの場合）
	if errorInfo.Type == ErrorModel {
		return r.tryFallbackModel(operation)
	}

	// 通常の再試行
	if err := operation(); err != nil {
		// 再帰的に回復を試行
		newErrorInfo := r.AnalyzeError(err)
		return r.AttemptRecovery(newErrorInfo, operation)
	}

	// 成功時
	fmt.Printf("\033[32m✅ 回復に成功しました\033[0m\n")
	r.retryAttempts = 0
	return nil
}

// フォールバックモデルを試行
func (r *System) tryFallbackModel(operation func() error) error {
	originalModel := r.config.Model

	fmt.Printf("\033[33m🔄 フォールバックモデルに切り替え中: %s\033[0m\n", r.fallbackModel)

	// 一時的にモデルを変更
	r.config.Model = r.fallbackModel

	err := operation()

	// 元のモデル設定を復元
	r.config.Model = originalModel

	if err != nil {
		fmt.Printf("\033[31m❌ フォールバックモデルでも失敗しました\033[0m\n")
		return err
	}

	fmt.Printf("\033[32m✅ フォールバックモデルで成功しました\033[0m\n")
	fmt.Printf("\033[33m💡 元のモデル '%s' は利用できません。設定を確認してください。\033[0m\n", originalModel)

	return nil
}

// エラーガイダンスを表示
func (r *System) displayErrorGuidance(errorInfo *ErrorInfo) error {
	fmt.Printf("\n\033[31m━━━ エラー診断 ━━━\033[0m\n")
	fmt.Printf("\033[31m種類:\033[0m %s\n", r.getErrorTypeDescription(errorInfo.Type))
	fmt.Printf("\033[31mメッセージ:\033[0m %s\n", errorInfo.Original.Error())
	fmt.Printf("\033[33m提案:\033[0m %s\n", errorInfo.Suggestion)

	// タイプ別の詳細ガイダンス
	switch errorInfo.Type {
	case ErrorConnection:
		fmt.Printf("\n\033[36m🔧 接続問題の解決方法:\033[0m\n")
		fmt.Printf("  1. \033[32mollama serve\033[0m でOllamaサーバーを起動\n")
		fmt.Printf("  2. \033[32mvyb config set-provider lmstudio\033[0m で別プロバイダーを試行\n")
		fmt.Printf("  3. \033[32mvyb config list\033[0m で設定を確認\n")

	case ErrorModel:
		fmt.Printf("\n\033[36m🤖 モデル問題の解決方法:\033[0m\n")
		fmt.Printf("  1. \033[32mollama list\033[0m でインストール済みモデルを確認\n")
		fmt.Printf("  2. \033[32mollama pull %s\033[0m でモデルをインストール\n", r.config.Model)
		fmt.Printf("  3. \033[32mvyb config set-model qwen2.5-coder:7b\033[0m で軽量モデルに変更\n")

	case ErrorTimeout:
		fmt.Printf("\n\033[36m⏱️ タイムアウト問題の解決方法:\033[0m\n")
		fmt.Printf("  1. より軽量なモデルを使用\n")
		fmt.Printf("  2. リクエスト内容を短縮\n")
		fmt.Printf("  3. ネットワーク環境を確認\n")
	}

	return errorInfo.Original
}

// エラータイプの説明を取得
func (r *System) getErrorTypeDescription(errorType ErrorType) string {
	switch errorType {
	case ErrorConnection:
		return "接続エラー"
	case ErrorModel:
		return "モデルエラー"
	case ErrorTimeout:
		return "タイムアウトエラー"
	case ErrorRateLimit:
		return "レート制限エラー"
	case ErrorAuthentication:
		return "認証エラー"
	default:
		return "不明なエラー"
	}
}

// 回復統計をリセット
func (r *System) Reset() {
	r.retryAttempts = 0
}
