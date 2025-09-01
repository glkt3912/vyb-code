package security

import (
	"regexp"
	"strings"
)

// 危険なコマンドパターンを検出するバリデーター
type CommandValidator struct {
	dangerousPatterns  []*regexp.Regexp // 危険なパターンの正規表現
	suspiciousPatterns []*regexp.Regexp // 疑わしいパターンの正規表現
	bannedCombinations []string         // 禁止された文字列組み合わせ
}

// コマンド検証結果
type ValidationResult struct {
	IsAllowed        bool     `json:"is_allowed"`
	RiskLevel        string   `json:"risk_level"` // "safe", "suspicious", "dangerous"
	DetectedPatterns []string `json:"detected_patterns"`
	Reason           string   `json:"reason"`
	RequiresConfirm  bool     `json:"requires_confirm"` // ユーザー確認が必要か
}

// コマンドバリデーターのコンストラクタ
func NewCommandValidator() *CommandValidator {
	return &CommandValidator{
		dangerousPatterns:  compileDangerousPatterns(),
		suspiciousPatterns: compileSuspiciousPatterns(),
		bannedCombinations: []string{
			"rm -rf", "sudo rm", "chmod 777", "curl.*|.*sh", "wget.*|.*sh",
			"echo.*>.*passwd", "cat.*>.*shadow", "nc -l", "netcat -l",
			"dd if=", "mkfs", "fdisk", "mount /dev", "umount -f",
		},
	}
}

// 危険なパターンをコンパイル
func compileDangerousPatterns() []*regexp.Regexp {
	patterns := []string{
		// コマンドインジェクション
		`[;&|]\s*rm\s+-rf`,      // ; rm -rf, && rm -rf
		`[;&|]\s*sudo`,          // ; sudo, && sudo
		`[;&|]\s*curl.*\|\s*sh`, // ; curl ... | sh
		`[;&|]\s*wget.*\|\s*sh`, // ; wget ... | sh

		// データ窃取
		`cat\s+.*passwd.*\|\s*curl`, // cat /etc/passwd | curl
		`grep\s+.*shadow.*\|\s*nc`,  // grep shadow | nc
		`find.*-exec.*rm`,           // find ... -exec rm

		// システム破壊
		`rm\s+-rf\s+/`,    // rm -rf /
		`dd\s+if=.*of=.*`, // dd if=/dev/zero of=/dev/sda
		`:(){ :|:& };:`,   // fork bomb

		// ネットワーク通信
		`curl\s+.*\|\s*bash`, // curl ... | bash
		`wget.*&&.*sh`,       // wget && sh
		`nc.*-e.*sh`,         // netcat reverse shell

		// 権限昇格
		`sudo\s+chmod.*777`,  // sudo chmod 777
		`sudo\s+chown.*root`, // sudo chown root
		`su\s+-.*`,           // su - root
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			compiled = append(compiled, regex)
		}
	}

	return compiled
}

// 疑わしいパターンをコンパイル
func compileSuspiciousPatterns() []*regexp.Regexp {
	patterns := []string{
		// 複数コマンド結合
		`.*[;&|]{2}.*`,    // &&, ||, ;
		`.*\|\s*sh\s*$`,   // ... | sh
		`.*\|\s*bash\s*$`, // ... | bash

		// 外部通信の可能性
		`curl\s+http`, // curl http://
		`wget\s+http`, // wget http://
		`nc\s+.*\d+`,  // nc host port

		// ファイルシステム操作
		`mv\s+.*\s+/`,      // mv file /system/path
		`cp\s+.*\s+/`,      // cp file /system/path
		`chmod\s+[0-9]{3}`, // chmod 755

		// 環境変数操作
		`export\s+.*=`, // export VAR=value
		`unset\s+.*`,   // unset VAR

		// プロセス制御
		`kill\s+-9`, // kill -9
		`pkill.*`,   // pkill process
		`killall.*`, // killall process
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			compiled = append(compiled, regex)
		}
	}

	return compiled
}

// コマンドの検証を実行
func (v *CommandValidator) ValidateCommand(command string) *ValidationResult {
	result := &ValidationResult{
		IsAllowed:        true,
		RiskLevel:        "safe",
		DetectedPatterns: make([]string, 0),
		RequiresConfirm:  false,
	}

	// 禁止された組み合わせをチェック
	commandLower := strings.ToLower(command)
	for _, banned := range v.bannedCombinations {
		if matched, _ := regexp.MatchString(banned, commandLower); matched {
			result.IsAllowed = false
			result.RiskLevel = "dangerous"
			result.DetectedPatterns = append(result.DetectedPatterns, banned)
			result.Reason = "禁止されたコマンド組み合わせが検出されました"
			return result
		}
	}

	// 危険なパターンをチェック
	for _, pattern := range v.dangerousPatterns {
		if pattern.MatchString(command) {
			result.IsAllowed = false
			result.RiskLevel = "dangerous"
			result.DetectedPatterns = append(result.DetectedPatterns, pattern.String())
			result.Reason = "危険なコマンドパターンが検出されました"
			return result
		}
	}

	// 疑わしいパターンをチェック
	for _, pattern := range v.suspiciousPatterns {
		if pattern.MatchString(command) {
			result.RiskLevel = "suspicious"
			result.DetectedPatterns = append(result.DetectedPatterns, pattern.String())
			result.RequiresConfirm = true
			result.Reason = "疑わしいコマンドパターンが検出されました。実行前に確認してください"
		}
	}

	return result
}

// 単純なコマンドかどうかをチェック（複合コマンドでない）
func (v *CommandValidator) IsSimpleCommand(command string) bool {
	// 危険な演算子をチェック
	dangerousOperators := []string{"&&", "||", ";", "|", ">", ">>", "<", "$(", "`"}

	for _, op := range dangerousOperators {
		if strings.Contains(command, op) {
			return false
		}
	}

	return true
}

// LLM応答の検証（コードブロックの抽出と検証）
func (v *CommandValidator) ValidateLLMResponse(response string) []string {
	var suspiciousCommands []string
	seen := make(map[string]bool) // 重複を避ける

	// コードブロックを抽出（```bash ... ```）
	codeBlockRegex := regexp.MustCompile("(?s)```(?:bash|shell|sh)?\n(.*?)\n```")
	matches := codeBlockRegex.FindAllStringSubmatch(response, -1)

	for _, match := range matches {
		if len(match) > 1 {
			commands := strings.Split(match[1], "\n")
			for _, cmd := range commands {
				cmd = strings.TrimSpace(cmd)
				if cmd == "" || strings.HasPrefix(cmd, "#") {
					continue // 空行やコメントをスキップ
				}

				// 各コマンドを検証
				result := v.ValidateCommand(cmd)
				if result.RiskLevel == "dangerous" || result.RiskLevel == "suspicious" {
					if !seen[cmd] {
						suspiciousCommands = append(suspiciousCommands, cmd)
						seen[cmd] = true
					}
				}
			}
		}
	}

	// インラインコマンドも検索（`command`形式）
	inlineRegex := regexp.MustCompile("`([^`]+)`")
	inlineMatches := inlineRegex.FindAllStringSubmatch(response, -1)

	for _, match := range inlineMatches {
		if len(match) > 1 {
			cmd := strings.TrimSpace(match[1])
			// インラインコマンドは実際のコマンドのみ検証（説明文は除外）
			if strings.Contains(cmd, " ") && len(strings.Fields(cmd)) <= 10 {
				result := v.ValidateCommand(cmd)
				if result.RiskLevel == "dangerous" || result.RiskLevel == "suspicious" {
					if !seen[cmd] {
						suspiciousCommands = append(suspiciousCommands, cmd)
						seen[cmd] = true
					}
				}
			}
		}
	}

	return suspiciousCommands
}
