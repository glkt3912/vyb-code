package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestParseLevel はログレベル解析をテストする
func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"DEBUG", DebugLevel},
		{"debug", DebugLevel},
		{"INFO", InfoLevel},
		{"info", InfoLevel},
		{"WARN", WarnLevel},
		{"WARNING", WarnLevel},
		{"ERROR", ErrorLevel},
		{"FATAL", FatalLevel},
		{"OFF", OffLevel},
		{"invalid", InfoLevel}, // デフォルト
	}

	for _, test := range tests {
		result := ParseLevel(test.input)
		if result != test.expected {
			t.Errorf("ParseLevel(%s) = %v; 期待値 %v", test.input, result, test.expected)
		}
	}
}

// TestConsoleFormatter はコンソールフォーマッターをテストする
func TestConsoleFormatter(t *testing.T) {
	formatter := &ConsoleFormatter{
		ShowCaller:    false,
		ShowTimestamp: true,
		ColorEnabled:  false,
	}

	entry := Entry{
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Level:     "INFO",
		Message:   "テストメッセージ",
		Component: "test",
		Context: map[string]interface{}{
			"key": "value",
		},
	}

	output, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("フォーマットエラー: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "12:00:00") {
		t.Error("タイムスタンプが含まれていません")
	}

	if !strings.Contains(outputStr, "[INFO]") {
		t.Error("ログレベルが含まれていません")
	}

	if !strings.Contains(outputStr, "テストメッセージ") {
		t.Error("メッセージが含まれていません")
	}

	if !strings.Contains(outputStr, "(test)") {
		t.Error("コンポーネント名が含まれていません")
	}

	if !strings.Contains(outputStr, "key=value") {
		t.Error("コンテキストが含まれていません")
	}
}

// TestJSONFormatter はJSONフォーマッターをテストする
func TestJSONFormatter(t *testing.T) {
	formatter := &JSONFormatter{}

	entry := Entry{
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Level:     "INFO",
		Message:   "テストメッセージ",
		Component: "test",
		Context: map[string]interface{}{
			"key": "value",
		},
	}

	output, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("フォーマットエラー: %v", err)
	}

	// JSONとしてパースできることを確認
	var parsed Entry
	if err := json.Unmarshal(output, &parsed); err != nil {
		t.Fatalf("JSON解析エラー: %v", err)
	}

	if parsed.Level != entry.Level {
		t.Errorf("期待レベル: %s, 実際: %s", entry.Level, parsed.Level)
	}

	if parsed.Message != entry.Message {
		t.Errorf("期待メッセージ: %s, 実際: %s", entry.Message, parsed.Message)
	}
}

// TestNewLogger はロガー作成をテストする
func TestNewLogger(t *testing.T) {
	config := DefaultConfig()
	config.Component = "test-component"

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("ロガー作成エラー: %v", err)
	}

	if logger.level != InfoLevel {
		t.Errorf("期待レベル: %v, 実際: %v", InfoLevel, logger.level)
	}

	if logger.component != "test-component" {
		t.Errorf("期待コンポーネント: %s, 実際: %s", "test-component", logger.component)
	}

	if len(logger.outputs) != 1 {
		t.Errorf("期待出力数: 1, 実際: %d", len(logger.outputs))
	}
}

// TestLoggerLevels はログレベル制御をテストする
func TestLoggerLevels(t *testing.T) {
	var buffer bytes.Buffer

	config := DefaultConfig()
	config.Level = WarnLevel
	config.Output = []string{"stdout"}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("ロガー作成エラー: %v", err)
	}

	// 出力先をバッファに変更
	logger.outputs = []io.Writer{&buffer}

	// デバッグとインフォは出力されない
	logger.Debug("デバッグメッセージ")
	logger.Info("インフォメッセージ")

	if buffer.Len() > 0 {
		t.Error("WARNレベル以下のメッセージが出力されました")
	}

	// ワーニングとエラーは出力される
	logger.Warn("ワーニングメッセージ")
	if buffer.Len() == 0 {
		t.Error("WARNメッセージが出力されませんでした")
	}

	buffer.Reset()
	logger.Error("エラーメッセージ")
	if buffer.Len() == 0 {
		t.Error("ERRORメッセージが出力されませんでした")
	}
}

// TestWithContext はコンテキスト付きロガーをテストする
func TestWithContext(t *testing.T) {
	var buffer bytes.Buffer

	config := DefaultConfig()
	logger, _ := NewLogger(config)
	logger.outputs = []io.Writer{&buffer}

	// コンテキスト付きロガーを作成
	contextLogger := logger.WithContext("request_id", "12345")

	contextLogger.Info("テストメッセージ")

	output := buffer.String()
	if !strings.Contains(output, "request_id=12345") {
		t.Error("コンテキストが含まれていません")
	}

	// 元のロガーにはコンテキストが残らない
	buffer.Reset()
	logger.Info("元のロガー")

	output = buffer.String()
	if strings.Contains(output, "request_id=12345") {
		t.Error("元のロガーにコンテキストが残っています")
	}
}

// TestWithFields はフィールド付きロガーをテストする
func TestWithFields(t *testing.T) {
	var buffer bytes.Buffer

	config := DefaultConfig()
	logger, _ := NewLogger(config)
	logger.outputs = []io.Writer{&buffer}

	// フィールド付きロガーを作成
	fields := map[string]interface{}{
		"user_id":    "user123",
		"session_id": "sess456",
	}
	fieldLogger := logger.WithFields(fields)

	fieldLogger.Info("テストメッセージ")

	output := buffer.String()
	if !strings.Contains(output, "user_id=user123") {
		t.Error("user_idフィールドが含まれていません")
	}

	if !strings.Contains(output, "session_id=sess456") {
		t.Error("session_idフィールドが含まれていません")
	}
}

// TestGlobalLogger はグローバルロガーをテストする
func TestGlobalLogger(t *testing.T) {
	// グローバルロガーをリセット
	globalLogger = nil
	once = sync.Once{}

	// デフォルト設定で初期化
	config := DefaultConfig()
	err := InitGlobalLogger(config)
	if err != nil {
		t.Fatalf("グローバルロガー初期化エラー: %v", err)
	}

	// ロガーが取得できることを確認
	logger := GetLogger()
	if logger == nil {
		t.Fatal("グローバルロガーが取得できません")
	}

	// 便利関数が動作することを確認
	Info("テストメッセージ")
	Debug("デバッグメッセージ")
	Warn("ワーニングメッセージ")
	Error("エラーメッセージ")
}

// TestFileOutput はファイル出力をテストする
func TestFileOutput(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := DefaultConfig()
	config.Output = []string{"file:" + logFile}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("ロガー作成エラー: %v", err)
	}

	// ログを出力
	logger.Info("ファイルテストメッセージ")

	// ファイルが作成されたことを確認
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("ログファイルが作成されていません")
	}

	// ファイル内容を確認
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ログファイル読み込みエラー: %v", err)
	}

	if !strings.Contains(string(content), "ファイルテストメッセージ") {
		t.Error("ログメッセージがファイルに書き込まれていません")
	}
}

// TestColorOutput は色付き出力をテストする
func TestColorOutput(t *testing.T) {
	formatter := &ConsoleFormatter{
		ColorEnabled: true,
	}

	entry := Entry{
		Level:   "ERROR",
		Message: "エラーメッセージ",
	}

	output, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("フォーマットエラー: %v", err)
	}

	outputStr := string(output)
	// ANSI色コードが含まれていることを確認
	if !strings.Contains(outputStr, "\033[31m") { // 赤色
		t.Error("色付きフォーマットが適用されていません")
	}
}

// TestVybLoggerTestMode はテストモード機能をテストする
func TestVybLoggerTestMode(t *testing.T) {
	var buffer bytes.Buffer

	config := DefaultConfig()
	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("ロガー作成エラー: %v", err)
	}

	logger.outputs = []io.Writer{&buffer}

	// テストモードを有効化
	logger.SetTestMode(true)

	// Fatalログを出力（os.Exitが呼ばれないことを確認）
	logger.Fatal("テストFatalメッセージ")

	// バッファにFatalメッセージが出力されていることを確認
	output := buffer.String()
	if !strings.Contains(output, "FATAL") {
		t.Error("Fatalメッセージが出力されていません")
	}

	if !strings.Contains(output, "テストFatalメッセージ") {
		t.Error("Fatalメッセージの内容が正しくありません")
	}
}

// TestVybLoggerCustomExitHandler はカスタム終了ハンドラーをテストする
func TestVybLoggerCustomExitHandler(t *testing.T) {
	var buffer bytes.Buffer
	var exitCode int
	exitCalled := false

	config := DefaultConfig()
	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("ロガー作成エラー: %v", err)
	}

	logger.outputs = []io.Writer{&buffer}

	// カスタム終了ハンドラーを設定
	logger.SetExitHandler(func(code int) {
		exitCode = code
		exitCalled = true
	})

	// Fatalログを出力
	logger.Fatal("カスタムハンドラーテスト")

	// カスタムハンドラーが呼ばれたことを確認
	if !exitCalled {
		t.Error("カスタム終了ハンドラーが呼ばれませんでした")
	}

	if exitCode != 1 {
		t.Errorf("期待終了コード: 1, 実際: %d", exitCode)
	}
}

// TestVybLoggerFatalCallback はFatalコールバック機能をテストする
func TestVybLoggerFatalCallback(t *testing.T) {
	var buffer bytes.Buffer
	callbackCalled := false

	config := DefaultConfig()
	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("ロガー作成エラー: %v", err)
	}

	logger.outputs = []io.Writer{&buffer}
	logger.SetTestMode(true) // os.Exitを無効化

	// Fatalコールバックを設定
	logger.SetFatalCallback(func() {
		callbackCalled = true
	})

	// Fatalログを出力
	logger.Fatal("コールバックテスト")

	// コールバックが呼ばれたことを確認
	if !callbackCalled {
		t.Error("Fatalコールバックが呼ばれませんでした")
	}
}
