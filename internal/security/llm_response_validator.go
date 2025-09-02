package security

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// LLMレスポンス検証器
type LLMResponseValidator struct {
	maliciousPatterns   []*regexp.Regexp // 悪意のあるコードパターン
	privateInfoPatterns []*regexp.Regexp // プライベート情報パターン
	harmfulPatterns     []*regexp.Regexp // 有害コンテンツパターン
	maxResponseLength   int              // 最大レスポンス長
	allowCodeGeneration bool             // コード生成を許可するか
}

// LLMレスポンス検証結果
type LLMValidationResult struct {
	IsValid         bool     `json:"is_valid"`
	RiskLevel       string   `json:"risk_level"` // "safe", "warning", "dangerous"
	DetectedThreats []string `json:"detected_threats"`
	FilteredContent string   `json:"filtered_content"` // フィルタリング後のコンテンツ
	RequiresReview  bool     `json:"requires_review"`  // 人的レビューが必要か
	TruncatedReason string   `json:"truncated_reason"` // 切り詰めの理由
	SecurityScore   float64  `json:"security_score"`   // セキュリティスコア (0-10)
}

// 新しいLLMレスポンス検証器を作成
func NewLLMResponseValidator() *LLMResponseValidator {
	return &LLMResponseValidator{
		maliciousPatterns:   compileMaliciousPatterns(),
		privateInfoPatterns: compilePrivateInfoPatterns(),
		harmfulPatterns:     compileHarmfulPatterns(),
		maxResponseLength:   50000, // 50KB制限
		allowCodeGeneration: true,  // デフォルトでコード生成を許可
	}
}

// LLMレスポンスを検証
func (v *LLMResponseValidator) ValidateResponse(content string) (*LLMValidationResult, error) {
	result := &LLMValidationResult{
		IsValid:         true,
		RiskLevel:       "safe",
		DetectedThreats: make([]string, 0),
		FilteredContent: content,
		RequiresReview:  false,
		SecurityScore:   0.0,
	}

	// サイズチェック
	if utf8.RuneCountInString(content) > v.maxResponseLength {
		result.FilteredContent = v.truncateResponse(content)
		result.TruncatedReason = "最大長制限により切り詰め"
	}

	// 悪意のあるコード検出
	threats := v.detectMaliciousContent(content)
	if len(threats) > 0 {
		result.DetectedThreats = append(result.DetectedThreats, threats...)
		result.SecurityScore += float64(len(threats)) * 2.0
		result.RiskLevel = "dangerous"
	}

	// プライベート情報漏洩チェック
	privateThreats := v.detectPrivateInfo(content)
	if len(privateThreats) > 0 {
		result.DetectedThreats = append(result.DetectedThreats, privateThreats...)
		result.SecurityScore += float64(len(privateThreats)) * 1.5
		if result.RiskLevel == "safe" {
			result.RiskLevel = "warning"
		}
	}

	// 有害コンテンツチェック
	harmfulThreats := v.detectHarmfulContent(content)
	if len(harmfulThreats) > 0 {
		result.DetectedThreats = append(result.DetectedThreats, harmfulThreats...)
		result.SecurityScore += float64(len(harmfulThreats)) * 3.0
		result.RiskLevel = "dangerous"
	}

	// コード生成検証
	if v.allowCodeGeneration {
		codeThreats := v.validateGeneratedCode(content)
		if len(codeThreats) > 0 {
			result.DetectedThreats = append(result.DetectedThreats, codeThreats...)
			result.SecurityScore += float64(len(codeThreats)) * 1.0
			if result.RiskLevel == "safe" {
				result.RiskLevel = "warning"
			}
		}
	}

	// 最終判定
	if result.SecurityScore > 5.0 {
		result.IsValid = false
		result.RequiresReview = true
	} else if result.SecurityScore > 2.0 {
		result.RequiresReview = true
	}

	return result, nil
}

// 悪意のあるコンテンツを検出
func (v *LLMResponseValidator) detectMaliciousContent(content string) []string {
	var threats []string

	for _, pattern := range v.maliciousPatterns {
		if pattern.MatchString(content) {
			threats = append(threats, "悪意のあるコードパターンを検出: "+pattern.String())
		}
	}

	return threats
}

// プライベート情報を検出
func (v *LLMResponseValidator) detectPrivateInfo(content string) []string {
	var threats []string

	for _, pattern := range v.privateInfoPatterns {
		if pattern.MatchString(content) {
			threats = append(threats, "プライベート情報パターンを検出: "+pattern.String())
		}
	}

	return threats
}

// 有害コンテンツを検出
func (v *LLMResponseValidator) detectHarmfulContent(content string) []string {
	var threats []string

	for _, pattern := range v.harmfulPatterns {
		if pattern.MatchString(content) {
			threats = append(threats, "有害コンテンツパターンを検出: "+pattern.String())
		}
	}

	return threats
}

// 生成されたコードを検証
func (v *LLMResponseValidator) validateGeneratedCode(content string) []string {
	var threats []string

	// コードブロックを抽出
	codeBlocks := v.extractCodeBlocks(content)

	for _, code := range codeBlocks {
		// 危険な関数呼び出しをチェック
		if v.containsDangerousFunctions(code) {
			threats = append(threats, "危険な関数呼び出しを含むコードを検出")
		}

		// システムコマンド実行をチェック
		if v.containsSystemCommands(code) {
			threats = append(threats, "システムコマンド実行を含むコードを検出")
		}

		// ネットワーク通信をチェック
		if v.containsNetworkCalls(code) {
			threats = append(threats, "ネットワーク通信を含むコードを検出")
		}
	}

	return threats
}

// レスポンスを切り詰め
func (v *LLMResponseValidator) truncateResponse(content string) string {
	runes := []rune(content)
	if len(runes) > v.maxResponseLength {
		return string(runes[:v.maxResponseLength]) + "\n\n[内容が長すぎるため切り詰められました]"
	}
	return content
}

// コードブロックを抽出
func (v *LLMResponseValidator) extractCodeBlocks(content string) []string {
	// マークダウンコードブロックを抽出
	codeBlockRegex := regexp.MustCompile("```[\\s\\S]*?```")
	matches := codeBlockRegex.FindAllString(content, -1)

	var codeBlocks []string
	for _, match := range matches {
		// 先頭と末尾の```を除去
		code := strings.TrimPrefix(match, "```")
		code = strings.TrimSuffix(code, "```")
		code = strings.TrimSpace(code)

		// 言語指定がある場合は除去
		lines := strings.Split(code, "\n")
		if len(lines) > 0 && !strings.Contains(lines[0], " ") {
			code = strings.Join(lines[1:], "\n")
		}

		codeBlocks = append(codeBlocks, code)
	}

	return codeBlocks
}

// 危険な関数呼び出しを含むかチェック
func (v *LLMResponseValidator) containsDangerousFunctions(code string) bool {
	dangerousFunctions := []string{
		"exec", "system", "eval", "os.remove", "os.rmdir", "shutil.rmtree",
		"subprocess.call", "subprocess.run", "os.system", "shell_exec",
		"passthru", "__import__", "compile", "exec(", "eval(",
	}

	codeLower := strings.ToLower(code)
	for _, function := range dangerousFunctions {
		if strings.Contains(codeLower, function) {
			return true
		}
	}

	return false
}

// システムコマンドを含むかチェック
func (v *LLMResponseValidator) containsSystemCommands(code string) bool {
	systemPatterns := []string{
		"rm -", "sudo ", "chmod ", "chown ", "dd if=", "mkfs",
		"mount ", "umount ", "kill ", "killall ", "pkill ",
		"nc -l", "netcat ", "/bin/sh", "/bin/bash", "sh -c",
	}

	codeLower := strings.ToLower(code)
	for _, pattern := range systemPatterns {
		if strings.Contains(codeLower, pattern) {
			return true
		}
	}

	return false
}

// ネットワーク通信を含むかチェック
func (v *LLMResponseValidator) containsNetworkCalls(code string) bool {
	networkPatterns := []string{
		"http.get", "http.post", "requests.", "urllib", "fetch(",
		"axios.", "curl ", "wget ", "socket.", "connect(",
		"bind(", "listen(", "accept(", "send(", "recv(",
	}

	codeLower := strings.ToLower(code)
	for _, pattern := range networkPatterns {
		if strings.Contains(codeLower, pattern) {
			return true
		}
	}

	return false
}

// 悪意のあるパターンをコンパイル
func compileMaliciousPatterns() []*regexp.Regexp {
	patterns := []string{
		// コマンドインジェクション（より広範囲にマッチ）
		`rm\s+-rf\s*[\/\*]`,
		`sudo\s+rm`,
		`chmod\s+777`,
		`wget.*\|\s*sh`,
		`curl.*\|\s*bash`,
		`rm\s+-rf\s*/`,

		// リバースシェル
		`nc\s+-[el].*\d+`,
		`bash\s+-i\s+>&`,
		`python.*socket.*connect`,
		`perl.*socket.*connect`,

		// データ破壊
		`dd\s+if=.*of=/dev`,
		`mkfs\.\w+\s+/dev`,
		`formatc\:`,

		// 権限昇格
		`sudo\s+su\s*-`,
		`chmod\s+\+s\s+`,
		`setuid\s*\(\s*0\s*\)`,

		// 基本的な危険コマンド
		`rm\s+-rf`,
		`format\s+c:`,
		`del\s+/[sf]`,
	}

	var compiledPatterns []*regexp.Regexp
	for _, pattern := range patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			compiledPatterns = append(compiledPatterns, compiled)
		}
	}

	return compiledPatterns
}

// プライベート情報パターンをコンパイル
func compilePrivateInfoPatterns() []*regexp.Regexp {
	patterns := []string{
		// 認証情報
		`(?i)password\s*[:=]\s*["\']?\w+`,
		`(?i)api[_-]?key\s*[:=]\s*["\']?\w+`,
		`(?i)secret\s*[:=]\s*["\']?\w+`,
		`(?i)token\s*[:=]\s*["\']?\w+`,

		// 個人情報
		`\b\d{3}-\d{2}-\d{4}\b`,                          // SSN形式
		`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`,     // クレジットカード
		`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`, // メールアドレス

		// システム情報
		`/etc/passwd`,
		`/etc/shadow`,
		`id_rsa`,
		`private.*key`,
	}

	var compiledPatterns []*regexp.Regexp
	for _, pattern := range patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			compiledPatterns = append(compiledPatterns, compiled)
		}
	}

	return compiledPatterns
}

// 有害コンテンツパターンをコンパイル
func compileHarmfulPatterns() []*regexp.Regexp {
	patterns := []string{
		// 差別的表現
		`(?i)(hate|discrimination|racist|sexist).*content`,

		// 暴力的内容
		`(?i)(violence|harmful|destructive).*code`,

		// 不正アクセス
		`(?i)(hack|crack|exploit|backdoor)`,
		`(?i)(penetration|intrusion|bypass).*security`,

		// マルウェア関連
		`(?i)(malware|virus|trojan|rootkit)`,
		`(?i)(keylogger|spyware|ransomware)`,
	}

	var compiledPatterns []*regexp.Regexp
	for _, pattern := range patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			compiledPatterns = append(compiledPatterns, compiled)
		}
	}

	return compiledPatterns
}

// レスポンスにフィルターを適用
func (v *LLMResponseValidator) FilterResponse(content string) string {
	// 悪意のあるパターンをマスク
	filtered := content

	for _, pattern := range v.maliciousPatterns {
		filtered = pattern.ReplaceAllString(filtered, "[悪意のあるコードがフィルタリングされました]")
	}

	for _, pattern := range v.privateInfoPatterns {
		filtered = pattern.ReplaceAllString(filtered, "[プライベート情報がフィルタリングされました]")
	}

	for _, pattern := range v.harmfulPatterns {
		filtered = pattern.ReplaceAllString(filtered, "[有害コンテンツがフィルタリングされました]")
	}

	return filtered
}

// コード生成許可設定
func (v *LLMResponseValidator) SetAllowCodeGeneration(allow bool) {
	v.allowCodeGeneration = allow
}

// 最大レスポンス長設定
func (v *LLMResponseValidator) SetMaxResponseLength(length int) {
	if length > 0 {
		v.maxResponseLength = length
	}
}

// セキュリティレベルを設定（strict, moderate, permissive）
func (v *LLMResponseValidator) SetSecurityLevel(level string) error {
	switch level {
	case "strict":
		v.allowCodeGeneration = false
		v.maxResponseLength = 20000
	case "moderate":
		v.allowCodeGeneration = true
		v.maxResponseLength = 50000
	case "permissive":
		v.allowCodeGeneration = true
		v.maxResponseLength = 100000
	default:
		return fmt.Errorf("無効なセキュリティレベル: %s", level)
	}

	return nil
}

// レスポンスの統計情報を取得
func (v *LLMResponseValidator) GetValidationStats(results []*LLMValidationResult) map[string]interface{} {
	stats := make(map[string]interface{})

	totalResponses := len(results)
	safeCount := 0
	warningCount := 0
	dangerousCount := 0
	avgSecurityScore := 0.0

	for _, result := range results {
		switch result.RiskLevel {
		case "safe":
			safeCount++
		case "warning":
			warningCount++
		case "dangerous":
			dangerousCount++
		}
		avgSecurityScore += result.SecurityScore
	}

	if totalResponses > 0 {
		avgSecurityScore /= float64(totalResponses)
	}

	stats["total_responses"] = totalResponses
	stats["safe_responses"] = safeCount
	stats["warning_responses"] = warningCount
	stats["dangerous_responses"] = dangerousCount
	stats["average_security_score"] = avgSecurityScore
	stats["safety_rate"] = float64(safeCount) / float64(totalResponses) * 100

	return stats
}
