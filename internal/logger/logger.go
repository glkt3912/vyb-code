package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ログレベル定義
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	OffLevel
)

// ログレベル文字列表現
var levelNames = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
	FatalLevel: "FATAL",
	OffLevel:   "OFF",
}

// ログレベルをパース
func ParseLevel(level string) Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DebugLevel
	case "INFO":
		return InfoLevel
	case "WARN", "WARNING":
		return WarnLevel
	case "ERROR":
		return ErrorLevel
	case "FATAL":
		return FatalLevel
	case "OFF":
		return OffLevel
	default:
		return InfoLevel
	}
}

// ログエントリ構造
type Entry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// ログ出力フォーマッター
type Formatter interface {
	Format(entry Entry) ([]byte, error)
}

// コンソールフォーマッター（人間が読みやすい形式）
type ConsoleFormatter struct {
	ShowCaller    bool
	ShowTimestamp bool
	ColorEnabled  bool
}

// JSON形式で出力
func (f *ConsoleFormatter) Format(entry Entry) ([]byte, error) {
	var parts []string

	// タイムスタンプ
	if f.ShowTimestamp {
		timestamp := entry.Timestamp.Format("15:04:05")
		parts = append(parts, timestamp)
	}

	// ログレベル（色付き）
	level := entry.Level
	if f.ColorEnabled {
		switch entry.Level {
		case "DEBUG":
			level = "\033[36m" + level + "\033[0m" // シアン
		case "INFO":
			level = "\033[32m" + level + "\033[0m" // 緑
		case "WARN":
			level = "\033[33m" + level + "\033[0m" // 黄
		case "ERROR":
			level = "\033[31m" + level + "\033[0m" // 赤
		case "FATAL":
			level = "\033[35m" + level + "\033[0m" // マゼンタ
		}
	}
	parts = append(parts, fmt.Sprintf("[%s]", level))

	// コンポーネント
	if entry.Component != "" {
		parts = append(parts, fmt.Sprintf("(%s)", entry.Component))
	}

	// メッセージ
	parts = append(parts, entry.Message)

	// コンテキスト情報
	if len(entry.Context) > 0 {
		contextStr := ""
		for k, v := range entry.Context {
			if contextStr != "" {
				contextStr += " "
			}
			contextStr += fmt.Sprintf("%s=%v", k, v)
		}
		parts = append(parts, fmt.Sprintf("{%s}", contextStr))
	}

	// エラー情報
	if entry.Error != "" {
		parts = append(parts, fmt.Sprintf("error=%s", entry.Error))
	}

	// 呼び出し元情報
	if f.ShowCaller && entry.Caller != "" {
		parts = append(parts, fmt.Sprintf("at=%s", entry.Caller))
	}

	output := strings.Join(parts, " ") + "\n"
	return []byte(output), nil
}

// JSONフォーマッター（構造化ログ）
type JSONFormatter struct{}

// JSON形式で出力
func (f *JSONFormatter) Format(entry Entry) ([]byte, error) {
	return json.Marshal(entry)
}

// vyb統合ロガー
type VybLogger struct {
	mu            sync.RWMutex
	level         Level
	component     string
	formatter     Formatter
	outputs       []io.Writer
	context       map[string]interface{}
	testMode      bool      // テストモード（os.Exitを無効化）
	exitHandler   func(int) // 終了処理のカスタムハンドラー
	fatalCallback func()    // Fatalログ時のコールバック
}

// ログ設定
type Config struct {
	Level         Level             `json:"level"`
	Format        string            `json:"format"` // "console", "json"
	Output        []string          `json:"output"` // "stdout", "stderr", "file:/path/to/log"
	ShowCaller    bool              `json:"show_caller"`
	ShowTimestamp bool              `json:"show_timestamp"`
	ColorEnabled  bool              `json:"color_enabled"`
	Component     string            `json:"component"`
	Context       map[string]string `json:"context"`
}

// デフォルト設定
func DefaultConfig() Config {
	return Config{
		Level:         InfoLevel,
		Format:        "console",
		Output:        []string{"stdout"},
		ShowCaller:    false,
		ShowTimestamp: true,
		ColorEnabled:  true,
		Component:     "",
		Context:       make(map[string]string),
	}
}

// 新しいロガーを作成
func NewLogger(config Config) (*VybLogger, error) {
	logger := &VybLogger{
		level:     config.Level,
		component: config.Component,
		context:   make(map[string]interface{}),
	}

	// フォーマッターを設定
	switch config.Format {
	case "json":
		logger.formatter = &JSONFormatter{}
	default:
		logger.formatter = &ConsoleFormatter{
			ShowCaller:    config.ShowCaller,
			ShowTimestamp: config.ShowTimestamp,
			ColorEnabled:  config.ColorEnabled,
		}
	}

	// 出力先を設定
	outputs := make([]io.Writer, 0, len(config.Output))
	for _, output := range config.Output {
		writer, err := parseOutput(output)
		if err != nil {
			return nil, fmt.Errorf("出力先解析エラー '%s': %w", output, err)
		}
		outputs = append(outputs, writer)
	}
	logger.outputs = outputs

	// 初期コンテキストを設定
	for k, v := range config.Context {
		logger.context[k] = v
	}

	return logger, nil
}

// 出力先を解析
func parseOutput(output string) (io.Writer, error) {
	switch output {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		if strings.HasPrefix(output, "file:") {
			filePath := strings.TrimPrefix(output, "file:")

			// ディレクトリが存在しない場合は作成
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("ログディレクトリ作成失敗: %w", err)
			}

			file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return nil, fmt.Errorf("ログファイル開封失敗: %w", err)
			}
			return file, nil
		}
		return nil, fmt.Errorf("不明な出力先: %s", output)
	}
}

// ログレベルを設定
func (l *VybLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// コンポーネント名を設定
func (l *VybLogger) SetComponent(component string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.component = component
}

// コンテキストを追加
func (l *VybLogger) WithContext(key string, value interface{}) *VybLogger {
	l.mu.Lock()
	defer l.mu.Unlock()

	// コンテキストのコピーを作成
	newContext := make(map[string]interface{})
	for k, v := range l.context {
		newContext[k] = v
	}
	newContext[key] = value

	// 新しいロガーを作成
	return &VybLogger{
		level:     l.level,
		component: l.component,
		formatter: l.formatter,
		outputs:   l.outputs,
		context:   newContext,
	}
}

// 複数のコンテキストを追加
func (l *VybLogger) WithFields(fields map[string]interface{}) *VybLogger {
	l.mu.Lock()
	defer l.mu.Unlock()

	// コンテキストのコピーを作成
	newContext := make(map[string]interface{})
	for k, v := range l.context {
		newContext[k] = v
	}
	for k, v := range fields {
		newContext[k] = v
	}

	// 新しいロガーを作成
	return &VybLogger{
		level:     l.level,
		component: l.component,
		formatter: l.formatter,
		outputs:   l.outputs,
		context:   newContext,
	}
}

// ログエントリを書き込み
func (l *VybLogger) log(level Level, msg string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// ログレベルチェック
	if level < l.level {
		return
	}

	// メッセージをフォーマット
	message := fmt.Sprintf(msg, args...)

	// 呼び出し元情報を取得
	caller := ""
	if l.formatter != nil {
		if formatter, ok := l.formatter.(*ConsoleFormatter); ok && formatter.ShowCaller {
			_, file, line, ok := runtime.Caller(2)
			if ok {
				caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
			}
		}
	}

	// ログエントリを作成
	entry := Entry{
		Timestamp: time.Now(),
		Level:     levelNames[level],
		Message:   message,
		Component: l.component,
		Context:   l.context,
		Caller:    caller,
	}

	// フォーマットして出力
	formatted, err := l.formatter.Format(entry)
	if err != nil {
		// フォーマットエラーの場合はfallbackでfmt.Printf
		fmt.Printf("[LOG-ERROR] Failed to format log: %v\n", err)
		return
	}

	// 全ての出力先に書き込み
	for _, output := range l.outputs {
		output.Write(formatted)
	}
}

// ログメソッド実装
func (l *VybLogger) Debug(msg string, args ...interface{}) {
	l.log(DebugLevel, msg, args...)
}

func (l *VybLogger) Info(msg string, args ...interface{}) {
	l.log(InfoLevel, msg, args...)
}

func (l *VybLogger) Warn(msg string, args ...interface{}) {
	l.log(WarnLevel, msg, args...)
}

func (l *VybLogger) Error(msg string, args ...interface{}) {
	l.log(ErrorLevel, msg, args...)
}

func (l *VybLogger) Fatal(msg string, args ...interface{}) {
	l.log(FatalLevel, msg, args...)

	// コールバックが設定されている場合は実行
	if l.fatalCallback != nil {
		l.fatalCallback()
	}

	// テストモードでない場合のみ終了
	if !l.testMode {
		if l.exitHandler != nil {
			l.exitHandler(1)
		} else {
			os.Exit(1)
		}
	}
}

// テストモードを設定（os.Exitを無効化）
func (l *VybLogger) SetTestMode(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.testMode = enabled
}

// カスタム終了ハンドラーを設定
func (l *VybLogger) SetExitHandler(handler func(int)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.exitHandler = handler
}

// Fatalログ時のコールバックを設定
func (l *VybLogger) SetFatalCallback(callback func()) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fatalCallback = callback
}

// グローバルロガー
var globalLogger *VybLogger
var once sync.Once

// グローバルロガーを初期化
func InitGlobalLogger(config Config) error {
	var err error
	once.Do(func() {
		globalLogger, err = NewLogger(config)
	})
	return err
}

// グローバルロガーを取得
func GetLogger() *VybLogger {
	if globalLogger == nil {
		// デフォルト設定で初期化
		config := DefaultConfig()
		globalLogger, _ = NewLogger(config)
	}
	return globalLogger
}

// 便利関数（グローバルロガー使用）
func Debug(msg string, args ...interface{}) {
	GetLogger().Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	GetLogger().Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	GetLogger().Warn(msg, args...)
}

func Error(msg string, args ...interface{}) {
	GetLogger().Error(msg, args...)
}

func Fatal(msg string, args ...interface{}) {
	GetLogger().Fatal(msg, args...)
}

// コンテキスト付きロガー作成
func WithContext(key string, value interface{}) *VybLogger {
	return GetLogger().WithContext(key, value)
}

func WithFields(fields map[string]interface{}) *VybLogger {
	return GetLogger().WithFields(fields)
}

func WithComponent(component string) *VybLogger {
	logger := GetLogger()
	logger.SetComponent(component)
	return logger
}
