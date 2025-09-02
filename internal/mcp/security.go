package mcp

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

// MCPツールセキュリティバリデーター
type ToolSecurityValidator struct {
	constraints  *security.Constraints
	whitelist    map[string]bool // 許可されたツール名
	blacklist    map[string]bool // 禁止されたツール名
	rateLimiter  *RateLimiter    // レート制限機能
	auditLogger  *AuditLogger    // 監査ログ機能
	riskAnalyzer *RiskAnalyzer   // リスク分析機能
}

// レート制限機能
type RateLimiter struct {
	mu                sync.RWMutex
	toolCallCount     map[string][]time.Time // ツール毎の呼び出し時刻
	maxCallsPerMinute int
	windowDuration    time.Duration
}

// 監査ログ機能
type AuditLogger struct {
	mu      sync.RWMutex
	events  []SecurityEvent
	maxSize int
}

// セキュリティイベント
type SecurityEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"eventType"`
	ToolName    string                 `json:"toolName"`
	Arguments   map[string]interface{} `json:"arguments"`
	Action      string                 `json:"action"` // "allowed", "denied", "warning"
	Reason      string                 `json:"reason"`
	RiskScore   float64                `json:"riskScore"`
	SessionInfo string                 `json:"sessionInfo"`
}

// リスク分析機能
type RiskAnalyzer struct {
	mu                sync.RWMutex
	recentTools       []string
	toolCombinations  map[string]float64 // ツール組み合わせのリスクスコア
	highRiskThreshold float64
}

// 新しいセキュリティバリデーターを作成
func NewToolSecurityValidator(constraints *security.Constraints) *ToolSecurityValidator {
	return &ToolSecurityValidator{
		constraints:  constraints,
		whitelist:    make(map[string]bool),
		blacklist:    make(map[string]bool),
		rateLimiter:  NewRateLimiter(60, time.Minute), // 1分間に60回まで
		auditLogger:  NewAuditLogger(1000),            // 最大1000イベントを記録
		riskAnalyzer: NewRiskAnalyzer(8.0),            // 高リスク闾値: 8.0
	}
}

// 新しいレート制限器を作成
func NewRateLimiter(maxCalls int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		toolCallCount:     make(map[string][]time.Time),
		maxCallsPerMinute: maxCalls,
		windowDuration:    window,
	}
}

// 新しい監査ロガーを作成
func NewAuditLogger(maxSize int) *AuditLogger {
	return &AuditLogger{
		events:  make([]SecurityEvent, 0),
		maxSize: maxSize,
	}
}

// 新しいリスク分析器を作成
func NewRiskAnalyzer(threshold float64) *RiskAnalyzer {
	return &RiskAnalyzer{
		recentTools:       make([]string, 0),
		toolCombinations:  initializeRiskCombinations(),
		highRiskThreshold: threshold,
	}
}

// ツール実行を検証
func (v *ToolSecurityValidator) ValidateToolCall(toolName string, arguments map[string]interface{}) error {
	// レート制限チェック
	if !v.rateLimiter.AllowCall(toolName) {
		v.auditLogger.LogEvent(SecurityEvent{
			Timestamp: time.Now(),
			EventType: "rate_limit_exceeded",
			ToolName:  toolName,
			Action:    "denied",
			Reason:    "レート制限超過",
		})
		return fmt.Errorf("ツール '%s' のレート制限を超過しました", toolName)
	}

	// ブラックリストチェック
	if v.blacklist[toolName] {
		v.auditLogger.LogEvent(SecurityEvent{
			Timestamp: time.Now(),
			EventType: "blacklist_violation",
			ToolName:  toolName,
			Action:    "denied",
			Reason:    "ブラックリストに登録済み",
		})
		return fmt.Errorf("ツール '%s' は禁止されています", toolName)
	}

	// ホワイトリストが設定されている場合はチェック
	if len(v.whitelist) > 0 && !v.whitelist[toolName] {
		v.auditLogger.LogEvent(SecurityEvent{
			Timestamp: time.Now(),
			EventType: "whitelist_violation",
			ToolName:  toolName,
			Action:    "denied",
			Reason:    "ホワイトリストに未登録",
		})
		return fmt.Errorf("ツール '%s' は許可されていません", toolName)
	}

	// 危険なツール名パターンをチェック
	if v.isDangerousTool(toolName) {
		v.auditLogger.LogEvent(SecurityEvent{
			Timestamp: time.Now(),
			EventType: "dangerous_tool_attempt",
			ToolName:  toolName,
			Action:    "denied",
			Reason:    "危険なツールパターンを検出",
		})
		return fmt.Errorf("危険なツール '%s' の実行は禁止されています", toolName)
	}

	// リスク分析
	riskScore := v.riskAnalyzer.AnalyzeRisk(toolName, arguments)
	if riskScore > v.riskAnalyzer.highRiskThreshold {
		v.auditLogger.LogEvent(SecurityEvent{
			Timestamp: time.Now(),
			EventType: "high_risk_operation",
			ToolName:  toolName,
			Arguments: arguments,
			Action:    "denied",
			Reason:    "高リスク操作を検出",
			RiskScore: riskScore,
		})
		return fmt.Errorf("ツール '%s' のリスクスコアが闾値を超過しました: %.2f", toolName, riskScore)
	}

	// 引数の検証
	if err := v.validateArguments(toolName, arguments); err != nil {
		v.auditLogger.LogEvent(SecurityEvent{
			Timestamp: time.Now(),
			EventType: "argument_validation_failed",
			ToolName:  toolName,
			Arguments: arguments,
			Action:    "denied",
			Reason:    err.Error(),
		})
		return fmt.Errorf("引数検証失敗: %w", err)
	}

	// 成功ログ
	v.auditLogger.LogEvent(SecurityEvent{
		Timestamp: time.Now(),
		EventType: "tool_call_approved",
		ToolName:  toolName,
		Arguments: arguments,
		Action:    "allowed",
		Reason:    "セキュリティ検証成功",
		RiskScore: riskScore,
	})

	// リスク分析器にツール使用を記録
	v.riskAnalyzer.RecordToolUse(toolName)

	return nil
}

// 危険なツールかどうかを判定
func (v *ToolSecurityValidator) isDangerousTool(toolName string) bool {
	dangerousPatterns := []string{
		"exec", "shell", "system", "eval", "run",
		"delete", "remove", "rm", "kill", "destroy",
		"network", "http", "curl", "wget", "fetch",
		"admin", "sudo", "root", "privilege",
	}

	toolLower := strings.ToLower(toolName)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(toolLower, pattern) {
			return true
		}
	}

	return false
}

// 引数を検証
func (v *ToolSecurityValidator) validateArguments(toolName string, arguments map[string]interface{}) error {
	// コマンド系の引数チェック
	if command, exists := arguments["command"]; exists {
		if commandStr, ok := command.(string); ok {
			if err := v.constraints.IsCommandAllowed(commandStr); err != nil {
				return fmt.Errorf("コマンド '%s' は許可されていません: %w", commandStr, err)
			}
		}
	}

	// ファイルパス系の引数チェック
	for key, value := range arguments {
		if strings.Contains(strings.ToLower(key), "path") || strings.Contains(strings.ToLower(key), "file") {
			if pathStr, ok := value.(string); ok {
				if !v.constraints.IsPathAllowed(pathStr) {
					return fmt.Errorf("パス '%s' はワークスペース外です", pathStr)
				}
			}
		}
	}

	// URL系の引数チェック
	for key := range arguments {
		if strings.Contains(strings.ToLower(key), "url") || strings.Contains(strings.ToLower(key), "uri") {
			return fmt.Errorf("URLアクセスは許可されていません: %s", key)
		}
	}

	return nil
}

// ホワイトリストにツールを追加
func (v *ToolSecurityValidator) AddToWhitelist(toolName string) {
	v.whitelist[toolName] = true
}

// ブラックリストにツールを追加
func (v *ToolSecurityValidator) AddToBlacklist(toolName string) {
	v.blacklist[toolName] = true
}

// デフォルトのセキュリティ設定を適用
func (v *ToolSecurityValidator) ApplyDefaultPolicy() {
	// 安全なツールをホワイトリストに追加
	safeTools := []string{
		"read_file", "write_file", "list_files", "search_files",
		"git_status", "git_log", "git_diff", "git_branch",
		"analyze_code", "format_code", "lint_code",
	}

	for _, tool := range safeTools {
		v.AddToWhitelist(tool)
	}

	// 危険なツールをブラックリストに追加
	dangerousTools := []string{
		"system_exec", "shell_exec", "network_request",
		"file_delete", "admin_command", "privilege_escalation",
	}

	for _, tool := range dangerousTools {
		v.AddToBlacklist(tool)
	}
}

// レート制限をチェックして呼び出しを許可するかを判定
func (rl *RateLimiter) AllowCall(toolName string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// 古い呼び出し記録を削除
	if calls, exists := rl.toolCallCount[toolName]; exists {
		var validCalls []time.Time
		for _, callTime := range calls {
			if now.Sub(callTime) <= rl.windowDuration {
				validCalls = append(validCalls, callTime)
			}
		}
		rl.toolCallCount[toolName] = validCalls
	}

	// 現在の呼び出し数をチェック
	if len(rl.toolCallCount[toolName]) >= rl.maxCallsPerMinute {
		return false
	}

	// 新しい呼び出しを記録
	rl.toolCallCount[toolName] = append(rl.toolCallCount[toolName], now)
	return true
}

// セキュリティイベントをログに記録
func (al *AuditLogger) LogEvent(event SecurityEvent) {
	al.mu.Lock()
	defer al.mu.Unlock()

	// 最大サイズを超えた場合は古いイベントを削除
	if len(al.events) >= al.maxSize {
		// 最初の要素を削除してサイズを制限
		al.events = al.events[1:]
	}

	al.events = append(al.events, event)
}

// 最近のセキュリティイベントを取得
func (al *AuditLogger) GetRecentEvents(limit int) []SecurityEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	start := len(al.events) - limit
	if start < 0 {
		start = 0
	}

	return al.events[start:]
}

// リスク分析を実行
func (ra *RiskAnalyzer) AnalyzeRisk(toolName string, arguments map[string]interface{}) float64 {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	baseRisk := ra.getBaseRisk(toolName)
	combinationRisk := ra.getCombinationRisk(toolName)
	argumentRisk := ra.getArgumentRisk(arguments)

	return baseRisk + combinationRisk + argumentRisk
}

// ツール使用を記録
func (ra *RiskAnalyzer) RecordToolUse(toolName string) {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	ra.recentTools = append(ra.recentTools, toolName)

	// 最新10件のみ保持
	if len(ra.recentTools) > 10 {
		ra.recentTools = ra.recentTools[len(ra.recentTools)-10:]
	}
}

// ベースリスクスコアを取得
func (ra *RiskAnalyzer) getBaseRisk(toolName string) float64 {
	riskScores := map[string]float64{
		"read_file":    1.0,
		"write_file":   3.0,
		"delete_file":  7.0,
		"exec_command": 9.0,
		"network_call": 8.0,
		"system_info":  2.0,
	}

	if score, exists := riskScores[toolName]; exists {
		return score
	}

	// 危険なパターンを含む場合は高リスク
	if ra.containsDangerousPattern(toolName) {
		return 6.0
	}

	return 2.0 // デフォルトリスク
}

// ツール組み合わせによるリスクを評価
func (ra *RiskAnalyzer) getCombinationRisk(toolName string) float64 {
	if len(ra.recentTools) == 0 {
		return 0.0
	}

	// 最近使用されたツールとの組み合わせリスクをチェック
	for _, recentTool := range ra.recentTools {
		if combination := recentTool + "+" + toolName; ra.toolCombinations[combination] > 0 {
			return ra.toolCombinations[combination]
		}
	}

	return 0.0
}

// 引数に基づくリスクを評価
func (ra *RiskAnalyzer) getArgumentRisk(arguments map[string]interface{}) float64 {
	risk := 0.0

	for key, value := range arguments {
		keyLower := strings.ToLower(key)

		// ファイルパス関連の引数
		if strings.Contains(keyLower, "path") || strings.Contains(keyLower, "file") {
			if pathStr, ok := value.(string); ok {
				if strings.Contains(pathStr, "..") || strings.HasPrefix(pathStr, "/") {
					risk += 2.0 // パストラバーサルの可能性
				}
			}
		}

		// コマンド関連の引数
		if strings.Contains(keyLower, "command") || strings.Contains(keyLower, "cmd") {
			if cmdStr, ok := value.(string); ok {
				if strings.Contains(cmdStr, ";") || strings.Contains(cmdStr, "|") || strings.Contains(cmdStr, "&") {
					risk += 3.0 // コマンドインジェクションの可能性
				}
			}
		}

		// ネットワーク関連の引数
		if strings.Contains(keyLower, "url") || strings.Contains(keyLower, "host") {
			risk += 4.0 // 外部通信の可能性
		}
	}

	return risk
}

// 危険なパターンを含むかチェック
func (ra *RiskAnalyzer) containsDangerousPattern(toolName string) bool {
	dangerousPatterns := []string{"exec", "shell", "system", "admin", "root", "delete", "kill"}
	toolLower := strings.ToLower(toolName)

	for _, pattern := range dangerousPatterns {
		if strings.Contains(toolLower, pattern) {
			return true
		}
	}

	return false
}

// リスクの高いツール組み合わせを初期化
func initializeRiskCombinations() map[string]float64 {
	return map[string]float64{
		"read_file+write_file":     2.0,
		"write_file+exec_command":  5.0,
		"read_file+network_call":   4.0,
		"system_info+exec_command": 6.0,
		"delete_file+write_file":   4.0,
	}
}

// セキュリティ統計情報を取得
func (v *ToolSecurityValidator) GetSecurityStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// レート制限統計
	v.rateLimiter.mu.RLock()
	stats["rate_limit_tool_count"] = len(v.rateLimiter.toolCallCount)
	v.rateLimiter.mu.RUnlock()

	// 監査ログ統計
	v.auditLogger.mu.RLock()
	stats["total_events"] = len(v.auditLogger.events)

	// イベント種類別集計
	eventTypes := make(map[string]int)
	for _, event := range v.auditLogger.events {
		eventTypes[event.EventType]++
	}
	stats["event_types"] = eventTypes
	v.auditLogger.mu.RUnlock()

	// ホワイトリスト・ブラックリスト統計
	stats["whitelist_size"] = len(v.whitelist)
	stats["blacklist_size"] = len(v.blacklist)

	return stats
}
