package input

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	// セキュリティ制限の定数
	MaxInputLength    = 8192        // 最大入力文字数
	MaxLineLength     = 1024        // 1行あたりの最大文字数
	RateLimitPeriod   = time.Second // レート制限期間
	MaxRequestsPerSec = 100         // 秒あたりの最大リクエスト数
)

// 入力セキュリティバリデーター
type SecurityValidator struct {
	requestCounts map[string]int // IPごとのリクエスト数
	lastReset     time.Time      // 最後のリセット時間
	bufferLimit   int            // バッファサイズ制限
}

// 新しいセキュリティバリデーターを作成
func NewSecurityValidator() *SecurityValidator {
	return &SecurityValidator{
		requestCounts: make(map[string]int),
		lastReset:     time.Now(),
		bufferLimit:   MaxInputLength,
	}
}

// 入力サニタイゼーション - 危険な制御文字を除去
func (s *SecurityValidator) SanitizeInput(input string) (string, error) {
	if len(input) > s.bufferLimit {
		return "", fmt.Errorf("入力が最大長 %d を超えています", s.bufferLimit)
	}

	var result strings.Builder
	result.Grow(len(input)) // 事前に容量を確保

	for i, r := range input {
		// UTF-8の妥当性チェック
		if r == utf8.RuneError {
			if _, size := utf8.DecodeRuneInString(input[i:]); size == 1 {
				// 不正なUTF-8シーケンス
				continue
			}
		}

		// 危険な制御文字をフィルタリング
		if s.isDangerousControlChar(r) {
			continue
		}

		// ANSI エスケープシーケンスの制限付き許可
		if r == '\033' && i+1 < len(input) {
			if !s.isAllowedEscapeSequence(input[i:]) {
				continue
			}
		}

		result.WriteRune(r)
	}

	return result.String(), nil
}

// 危険な制御文字かどうかを判定
func (s *SecurityValidator) isDangerousControlChar(r rune) bool {
	// 許可する制御文字
	allowedControls := map[rune]bool{
		'\t':   true, // タブ
		'\n':   true, // 改行
		'\r':   true, // 復帰
		'\b':   true, // バックスペース
		'\f':   true, // フォームフィード
		'\v':   true, // 垂直タブ
		'\a':   true, // ベル
		'\033': true, // エスケープ（後で詳細チェック）
	}

	// 制御文字であり、許可されていない場合は危険とみなす
	if unicode.IsControl(r) && !allowedControls[r] {
		return true
	}

	// NULL文字や特定の危険な文字を明示的に禁止
	dangerousChars := []rune{
		0x00, // NULL
		0x01, // SOH (Start of Heading)
		0x02, // STX (Start of Text)
		0x03, // ETX (End of Text) - ただしCtrl+Cは別途処理
		0x04, // EOT (End of Transmission) - ただしCtrl+Dは別途処理
		0x05, // ENQ (Enquiry)
		0x06, // ACK (Acknowledge)
		0x0E, // SO (Shift Out)
		0x0F, // SI (Shift In)
		0x10, // DLE (Data Link Escape)
		0x11, // DC1 (Device Control 1)
		0x12, // DC2 (Device Control 2)
		0x13, // DC3 (Device Control 3)
		0x14, // DC4 (Device Control 4)
		0x15, // NAK (Negative Acknowledge)
		0x16, // SYN (Synchronous Idle)
		0x17, // ETB (End of Transmission Block)
		0x18, // CAN (Cancel)
		0x19, // EM (End of Medium)
		0x1A, // SUB (Substitute)
		0x1C, // FS (File Separator)
		0x1D, // GS (Group Separator)
		0x1E, // RS (Record Separator)
		0x1F, // US (Unit Separator)
		0x7F, // DEL (Delete) - ただしBackspaceは別途処理
	}

	for _, dangerous := range dangerousChars {
		if r == dangerous {
			return true
		}
	}

	return false
}

// 許可されるANSIエスケープシーケンスかどうかを判定
func (s *SecurityValidator) isAllowedEscapeSequence(input string) bool {
	if len(input) < 2 {
		return false
	}

	// 基本的なANSIエスケープシーケンスのみ許可
	allowedSequences := []string{
		"\033[A",  // 上矢印
		"\033[B",  // 下矢印
		"\033[C",  // 右矢印
		"\033[D",  // 左矢印
		"\033[H",  // Home
		"\033[F",  // End
		"\033[2J", // 画面クリア
		"\033[K",  // 行末まで削除
		"\033[G",  // カーソルを行頭に移動
		"\033[s",  // カーソル位置保存
		"\033[u",  // カーソル位置復元
	}

	for _, allowed := range allowedSequences {
		if strings.HasPrefix(input, allowed) {
			return true
		}
	}

	// カラーコード（基本的な形式のみ）
	if len(input) >= 4 && strings.HasPrefix(input, "\033[") {
		// \033[XXm の形式（XXは数字）
		for i := 2; i < len(input); i++ {
			char := input[i]
			if char == 'm' {
				// 有効なカラーコード
				return true
			}
			if char < '0' || char > '9' {
				if char != ';' { // セミコロンは複数値の区切り文字として許可
					break
				}
			}
		}
	}

	return false
}

// レート制限チェック
func (s *SecurityValidator) CheckRateLimit(clientID string) error {
	now := time.Now()

	// 期間リセットのチェック
	if now.Sub(s.lastReset) >= RateLimitPeriod {
		s.requestCounts = make(map[string]int)
		s.lastReset = now
	}

	// 現在のリクエスト数を確認
	if s.requestCounts[clientID] >= MaxRequestsPerSec {
		return fmt.Errorf("レート制限に達しました: %d requests/sec", MaxRequestsPerSec)
	}

	// リクエスト数を増加
	s.requestCounts[clientID]++
	return nil
}

// バッファオーバーフロー保護
func (s *SecurityValidator) ValidateBufferSize(input string, maxSize int) error {
	if len(input) > maxSize {
		return fmt.Errorf("入力サイズが制限を超えています: %d > %d bytes", len(input), maxSize)
	}

	// UTF-8の文字数でも制限チェック（バイト数制限が優先、文字数は参考チェック）
	runeCount := utf8.RuneCountInString(input)
	// ASCII文字の場合は文字数がバイト数とほぼ同じなのでmaxSizeを使用
	// マルチバイト文字の場合はバイト数制限で既に制御されているのでゆるい制限
	maxRunes := maxSize // ASCII相当の文字数制限
	if runeCount > maxRunes {
		return fmt.Errorf("入力文字数が制限を超えています: %d > %d characters", runeCount, maxRunes)
	}

	return nil
}

// 行長制限チェック
func (s *SecurityValidator) ValidateLineLength(input string) error {
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		if len(line) > MaxLineLength {
			return fmt.Errorf("行 %d が最大長 %d を超えています: %d bytes", i+1, MaxLineLength, len(line))
		}
	}
	return nil
}

// コマンドインジェクション保護
func (s *SecurityValidator) ValidateCommand(input string) error {
	// 危険なコマンド文字列パターンをチェック
	dangerousPatterns := []string{
		"&&", "||", "|", ">", "<", ">>", "<<",
		";", "`", "$(",
		"rm ", "del ", "format", "mkfs",
		"sudo ", "su ", "chmod +x",
		"/bin/", "/sbin/", "/usr/bin/",
		"python -c", "perl -e", "ruby -e",
		"curl ", "wget ", "nc ", "netcat",
		"ssh ", "scp ", "rsync",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerInput, pattern) {
			return fmt.Errorf("危険なコマンドパターンが検出されました: %s", pattern)
		}
	}

	return nil
}

// パスインジェクション保護
func (s *SecurityValidator) ValidatePath(input string) error {
	// 危険なパスパターンをチェック
	dangerousPathPatterns := []string{
		"../", "..\\",
		"/etc/", "/proc/", "/sys/", "/dev/",
		"C:\\Windows\\", "C:\\Program Files\\",
		"~/.ssh/", "~/.aws/", "~/.config/",
	}

	for _, pattern := range dangerousPathPatterns {
		if strings.Contains(input, pattern) {
			return fmt.Errorf("危険なパスパターンが検出されました: %s", pattern)
		}
	}

	return nil
}

// 統合セキュリティ検証
func (s *SecurityValidator) ValidateInput(input string, clientID string) (string, error) {
	// レート制限チェック
	if err := s.CheckRateLimit(clientID); err != nil {
		return "", err
	}

	// バッファサイズ検証
	if err := s.ValidateBufferSize(input, s.bufferLimit); err != nil {
		return "", err
	}

	// 行長制限チェック
	if err := s.ValidateLineLength(input); err != nil {
		return "", err
	}

	// コマンドインジェクション検証
	if err := s.ValidateCommand(input); err != nil {
		return "", err
	}

	// パスインジェクション検証（パスらしき文字列が含まれる場合）
	if strings.Contains(input, "/") || strings.Contains(input, "\\") {
		if err := s.ValidatePath(input); err != nil {
			return "", err
		}
	}

	// 入力サニタイゼーション
	sanitized, err := s.SanitizeInput(input)
	if err != nil {
		return "", err
	}

	return sanitized, nil
}

// セキュリティ設定更新
func (s *SecurityValidator) UpdateSecuritySettings(bufferLimit int, requestsPerSec int) {
	if bufferLimit > 0 && bufferLimit <= MaxInputLength {
		s.bufferLimit = bufferLimit
	}
	// 他の設定も必要に応じて更新可能
}
