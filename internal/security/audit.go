package security

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// 監査ログエントリ
type AuditEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"event_type"` // "command", "file", "llm_response"
	Action      string    `json:"action"`     // "allowed", "blocked", "suspicious"
	Command     string    `json:"command,omitempty"`
	FilePath    string    `json:"file_path,omitempty"`
	LLMResponse string    `json:"llm_response,omitempty"`
	Reason      string    `json:"reason"`
	RiskLevel   string    `json:"risk_level"` // "safe", "suspicious", "dangerous"
	UserID      string    `json:"user_id"`    // システムユーザー名
}

// 監査ログ管理
type AuditLogger struct {
	logFile    string
	maxEntries int
	enabled    bool
}

// 監査ログのコンストラクタ
func NewAuditLogger() *AuditLogger {
	homeDir, _ := os.UserHomeDir()
	logFile := filepath.Join(homeDir, ".vyb", "audit.log")

	return &AuditLogger{
		logFile:    logFile,
		maxEntries: 1000, // 最大1000エントリ
		enabled:    true,
	}
}

// 監査ログの有効/無効設定
func (a *AuditLogger) SetEnabled(enabled bool) {
	a.enabled = enabled
}

// コマンドブロックの記録
func (a *AuditLogger) LogBlocked(command, reason, details string) {
	if !a.enabled {
		return
	}

	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: "command",
		Action:    "blocked",
		Command:   command,
		Reason:    reason,
		RiskLevel: "dangerous",
		UserID:    getUserID(),
	}

	a.writeEntry(entry)
}

// コマンド承認の記録
func (a *AuditLogger) LogApproved(command, reason, details string) {
	if !a.enabled {
		return
	}

	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: "command",
		Action:    "allowed",
		Command:   command,
		Reason:    reason,
		RiskLevel: "safe",
		UserID:    getUserID(),
	}

	a.writeEntry(entry)
}

// コマンド検証結果の記録
func (a *AuditLogger) LogValidation(command string, result *ValidationResult) {
	if !a.enabled {
		return
	}

	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: "command",
		Action:    "validation",
		Command:   command,
		Reason:    result.Reason,
		RiskLevel: result.RiskLevel,
		UserID:    getUserID(),
	}

	a.writeEntry(entry)
}

// 疑わしいLLM応答の記録
func (a *AuditLogger) LogSuspiciousLLM(response, suspiciousCommand string) {
	if !a.enabled {
		return
	}

	entry := AuditEntry{
		Timestamp:   time.Now(),
		EventType:   "llm_response",
		Action:      "suspicious_detected",
		Command:     suspiciousCommand,
		LLMResponse: truncateString(response, 500), // 長すぎる場合は切り詰め
		Reason:      "LLM応答に疑わしいコマンドが含まれています",
		RiskLevel:   "suspicious",
		UserID:      getUserID(),
	}

	a.writeEntry(entry)
}

// 設定変更の記録
func (a *AuditLogger) LogConfigChange(setting, newValue string) {
	if !a.enabled {
		return
	}

	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: "config",
		Action:    "changed",
		Reason:    fmt.Sprintf("設定変更: %s = %s", setting, newValue),
		RiskLevel: "safe",
		UserID:    getUserID(),
	}

	a.writeEntry(entry)
}

// 監査ログエントリをファイルに書き込み
func (a *AuditLogger) writeEntry(entry AuditEntry) {
	// ログディレクトリを作成
	logDir := filepath.Dir(a.logFile)
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return // 失敗しても処理は続行
	}

	// JSON形式でエントリを追記
	file, err := os.OpenFile(a.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer file.Close()

	// JSONエンコード
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	// ファイルに書き込み
	file.Write(data)
	file.WriteString("\n")

	// ローテーション（エントリ数が上限を超えた場合）
	a.rotateIfNeeded()
}

// ログファイルのローテーション
func (a *AuditLogger) rotateIfNeeded() {
	// ファイルサイズをチェック（簡易実装）
	info, err := os.Stat(a.logFile)
	if err != nil {
		return
	}

	// 1MB を超えた場合はローテーション
	if info.Size() > 1024*1024 {
		backupFile := a.logFile + ".old"
		os.Rename(a.logFile, backupFile)
	}
}

// 監査ログの表示
func (a *AuditLogger) ShowRecentLogs(limit int) error {
	file, err := os.Open(a.logFile)
	if err != nil {
		return fmt.Errorf("監査ログファイルを開けません: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	entries := make([]AuditEntry, 0)

	// 全エントリを読み込み
	for scanner.Scan() {
		var entry AuditEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
			entries = append(entries, entry)
		}
	}

	// 最新のエントリから表示
	start := len(entries) - limit
	if start < 0 {
		start = 0
	}

	fmt.Printf("\n📋 監査ログ（最新%d件）:\n", limit)
	for i := start; i < len(entries); i++ {
		entry := entries[i]
		status := getStatusEmoji(entry.Action, entry.RiskLevel)
		fmt.Printf("%s [%s] %s: %s\n",
			status,
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.EventType,
			entry.Reason,
		)

		if entry.Command != "" {
			fmt.Printf("   コマンド: %s\n", entry.Command)
		}
	}

	return nil
}

// ステータス絵文字を取得
func getStatusEmoji(action, riskLevel string) string {
	switch action {
	case "blocked":
		return "🚫"
	case "allowed":
		if riskLevel == "suspicious" {
			return "⚠️"
		}
		return "✅"
	case "suspicious_detected":
		return "🔍"
	default:
		return "ℹ️"
	}
}

// 現在のユーザーIDを取得
func getUserID() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}

// 文字列を指定長で切り詰め
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}
