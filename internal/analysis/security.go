package analysis

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// セキュリティ分析の実装

// セキュリティ問題を分析
func (pa *projectAnalyzer) AnalyzeSecurity(projectPath string) ([]SecurityIssue, error) {
	issues := make([]SecurityIssue, 0)

	// 各種セキュリティスキャンを実行
	scanners := []func(string) ([]SecurityIssue, error){
		pa.scanSecrets,              // シークレット漏洩
		pa.scanVulnerablePatterns,   // 脆弱なコードパターン
		pa.scanDependencyVulns,      // 依存関係の脆弱性
		pa.scanFilePermissions,      // ファイル権限の問題
		pa.scanConfigIssues,         // 設定の問題
		pa.scanSQLInjection,         // SQLインジェクション
		pa.scanXSS,                  // XSS脆弱性
		pa.scanPathTraversal,        // パストラバーサル
		pa.scanHardcodedCredentials, // ハードコードされた認証情報
	}

	for _, scanner := range scanners {
		if foundIssues, err := scanner(projectPath); err == nil {
			issues = append(issues, foundIssues...)
		}
	}

	// 重要度順にソート
	pa.sortSecurityIssuesBySeverity(issues)

	return issues, nil
}

// シークレット漏洩をスキャン
func (pa *projectAnalyzer) scanSecrets(projectPath string) ([]SecurityIssue, error) {
	issues := make([]SecurityIssue, 0)

	// シークレットパターン
	secretPatterns := map[string]*regexp.Regexp{
		"API Key":           regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*["']?([a-zA-Z0-9]{20,})`),
		"AWS Access Key":    regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		"AWS Secret Key":    regexp.MustCompile(`[0-9a-zA-Z/+=]{40}`),
		"GitHub Token":      regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
		"Private Key":       regexp.MustCompile(`-----BEGIN (RSA |EC |)PRIVATE KEY-----`),
		"Database Password": regexp.MustCompile(`(?i)(password|pwd|pass)\s*[:=]\s*["']([^"'\s]{8,})`),
		"JWT Token":         regexp.MustCompile(`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`),
		"OAuth Token":       regexp.MustCompile(`(?i)(access[_-]?token|oauth[_-]?token)\s*[:=]\s*["']?([a-zA-Z0-9_-]{20,})`),
	}

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)

		// 除外パターンをチェック
		for _, pattern := range pa.config.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				return nil
			}
		}

		// バイナリファイルをスキップ
		if pa.isBinaryFile(path) {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			for secretType, pattern := range secretPatterns {
				if matches := pattern.FindStringSubmatch(line); matches != nil {
					severity := "high"
					if secretType == "Private Key" || secretType == "AWS Secret Key" {
						severity = "critical"
					}

					issues = append(issues, SecurityIssue{
						Type:        "secret_exposure",
						Severity:    severity,
						Description: fmt.Sprintf("Potential %s found in source code", secretType),
						File:        relPath,
						Line:        lineNum,
						Suggestion:  "Remove hardcoded secrets and use environment variables or secure vaults",
						CWE:         "CWE-798",
					})
				}
			}
		}

		return scanner.Err()
	})

	return issues, err
}

// 脆弱なコードパターンをスキャン
func (pa *projectAnalyzer) scanVulnerablePatterns(projectPath string) ([]SecurityIssue, error) {
	issues := make([]SecurityIssue, 0)

	// 脆弱なパターン
	vulnPatterns := map[string]struct {
		pattern     *regexp.Regexp
		severity    string
		description string
		cwe         string
		suggestion  string
	}{
		"eval": {
			pattern:     regexp.MustCompile(`\beval\s*\(`),
			severity:    "critical",
			description: "Use of eval() can lead to code injection vulnerabilities",
			cwe:         "CWE-95",
			suggestion:  "Avoid using eval() and use safer alternatives for dynamic code execution",
		},
		"innerHTML": {
			pattern:     regexp.MustCompile(`\.innerHTML\s*=`),
			severity:    "medium",
			description: "Direct use of innerHTML can lead to XSS vulnerabilities",
			cwe:         "CWE-79",
			suggestion:  "Use textContent or properly sanitize HTML content",
		},
		"document.write": {
			pattern:     regexp.MustCompile(`document\.write\s*\(`),
			severity:    "medium",
			description: "document.write() can be exploited for XSS attacks",
			cwe:         "CWE-79",
			suggestion:  "Use modern DOM manipulation methods instead of document.write()",
		},
		"MD5": {
			pattern:     regexp.MustCompile(`\bmd5\s*\(|hashlib\.md5|crypto\.md5`),
			severity:    "medium",
			description: "MD5 is cryptographically broken and should not be used",
			cwe:         "CWE-327",
			suggestion:  "Use SHA-256 or other secure hashing algorithms",
		},
		"SHA1": {
			pattern:     regexp.MustCompile(`\bsha1\s*\(|hashlib\.sha1|crypto\.sha1`),
			severity:    "medium",
			description: "SHA-1 is cryptographically weak and should be avoided",
			cwe:         "CWE-327",
			suggestion:  "Use SHA-256 or other secure hashing algorithms",
		},
		"insecure_random": {
			pattern:     regexp.MustCompile(`Math\.random\(\)|random\.random\(\)|rand\(\)`),
			severity:    "low",
			description: "Insecure random number generation for security purposes",
			cwe:         "CWE-338",
			suggestion:  "Use cryptographically secure random number generators",
		},
	}

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)

		// 除外パターンをチェック
		for _, pattern := range pa.config.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				return nil
			}
		}

		if pa.isBinaryFile(path) {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			for _, vulnPattern := range vulnPatterns {
				if vulnPattern.pattern.MatchString(line) {
					issues = append(issues, SecurityIssue{
						Type:        "vulnerable_pattern",
						Severity:    vulnPattern.severity,
						Description: vulnPattern.description,
						File:        relPath,
						Line:        lineNum,
						Suggestion:  vulnPattern.suggestion,
						CWE:         vulnPattern.cwe,
					})
				}
			}
		}

		return scanner.Err()
	})

	return issues, err
}

// 依存関係の脆弱性をスキャン
func (pa *projectAnalyzer) scanDependencyVulns(projectPath string) ([]SecurityIssue, error) {
	issues := make([]SecurityIssue, 0)

	// 依存関係を分析
	dependencies, err := pa.AnalyzeDependencies(projectPath)
	if err != nil {
		return issues, err
	}

	// 既知の脆弱な依存関係
	vulnDependencies := map[string]struct {
		versions    []string
		severity    string
		description string
		cve         string
		suggestion  string
	}{
		"lodash": {
			versions:    []string{"4.17.20", "4.17.19", "4.17.18"},
			severity:    "high",
			description: "Prototype pollution vulnerability in lodash",
			cve:         "CVE-2020-8203",
			suggestion:  "Update to lodash version 4.17.21 or later",
		},
		"moment": {
			versions:    []string{"2.29.3", "2.29.2", "2.29.1"},
			severity:    "medium",
			description: "ReDoS vulnerability in moment.js",
			cve:         "CVE-2022-31129",
			suggestion:  "Update to moment version 2.29.4 or consider using day.js",
		},
		"jquery": {
			versions:    []string{"3.5.1", "3.5.0", "3.4.1"},
			severity:    "medium",
			description: "XSS vulnerability in jQuery",
			cve:         "CVE-2020-11022",
			suggestion:  "Update to jQuery version 3.6.0 or later",
		},
	}

	for _, dep := range dependencies {
		if vuln, exists := vulnDependencies[dep.Name]; exists {
			for _, vulnVersion := range vuln.versions {
				if dep.Version == vulnVersion || strings.HasPrefix(dep.Version, vulnVersion) {
					issues = append(issues, SecurityIssue{
						Type:        "vulnerable_dependency",
						Severity:    vuln.severity,
						Description: fmt.Sprintf("%s (version %s): %s", dep.Name, dep.Version, vuln.description),
						File:        dep.Source,
						Line:        0,
						Suggestion:  vuln.suggestion,
						CWE:         vuln.cve,
					})
				}
			}
		}
	}

	return issues, nil
}

// ファイル権限の問題をスキャン
func (pa *projectAnalyzer) scanFilePermissions(projectPath string) ([]SecurityIssue, error) {
	issues := make([]SecurityIssue, 0)

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)

		// 実行ファイルの権限チェック
		if !info.IsDir() && (info.Mode()&0111) != 0 {
			// 実行権限がある場合の追加チェック
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".sh" || ext == ".bash" || ext == ".py" || ext == ".rb" {
				if (info.Mode() & 0002) != 0 { // world-writable
					issues = append(issues, SecurityIssue{
						Type:        "insecure_permissions",
						Severity:    "medium",
						Description: "Script file is world-writable and executable",
						File:        relPath,
						Line:        0,
						Suggestion:  "Remove world-write permission: chmod o-w " + relPath,
						CWE:         "CWE-732",
					})
				}
			}
		}

		// 設定ファイルの権限チェック
		if pa.isConfigFile(path) && (info.Mode()&0044) != 0 { // world-readable
			issues = append(issues, SecurityIssue{
				Type:        "insecure_permissions",
				Severity:    "low",
				Description: "Configuration file is world-readable",
				File:        relPath,
				Line:        0,
				Suggestion:  "Restrict read permissions: chmod o-r " + relPath,
				CWE:         "CWE-732",
			})
		}

		return nil
	})

	return issues, err
}

// 設定の問題をスキャン
func (pa *projectAnalyzer) scanConfigIssues(projectPath string) ([]SecurityIssue, error) {
	issues := make([]SecurityIssue, 0)

	// .env ファイルのチェック
	envFiles := []string{".env", ".env.local", ".env.production"}
	for _, envFile := range envFiles {
		envPath := filepath.Join(projectPath, envFile)
		if _, err := os.Stat(envPath); err == nil {
			// .envファイルが.gitignoreに含まれているかチェック
			gitignorePath := filepath.Join(projectPath, ".gitignore")
			if gitignoreContent, err := os.ReadFile(gitignorePath); err == nil {
				if !strings.Contains(string(gitignoreContent), envFile) {
					issues = append(issues, SecurityIssue{
						Type:        "config_exposure",
						Severity:    "high",
						Description: fmt.Sprintf("Environment file %s is not in .gitignore", envFile),
						File:        envFile,
						Line:        0,
						Suggestion:  "Add " + envFile + " to .gitignore to prevent secret exposure",
						CWE:         "CWE-200",
					})
				}
			}
		}
	}

	// package.json のセキュリティ設定チェック
	packageJsonPath := filepath.Join(projectPath, "package.json")
	if content, err := os.ReadFile(packageJsonPath); err == nil {
		var packageData map[string]interface{}
		if err := json.Unmarshal(content, &packageData); err == nil {
			// npm audit の設定をチェック
			if scripts, exists := packageData["scripts"].(map[string]interface{}); exists {
				if _, hasAudit := scripts["audit"]; !hasAudit {
					issues = append(issues, SecurityIssue{
						Type:        "missing_security_config",
						Severity:    "low",
						Description: "No npm audit script configured",
						File:        "package.json",
						Line:        0,
						Suggestion:  "Add 'audit': 'npm audit' to scripts section",
						CWE:         "CWE-1059",
					})
				}
			}
		}
	}

	return issues, nil
}

// SQLインジェクションをスキャン
func (pa *projectAnalyzer) scanSQLInjection(projectPath string) ([]SecurityIssue, error) {
	issues := make([]SecurityIssue, 0)

	// SQLインジェクションのパターン
	sqlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`"SELECT\s+.*"\s*\+`),
		regexp.MustCompile(`"INSERT\s+.*"\s*\+`),
		regexp.MustCompile(`"UPDATE\s+.*"\s*\+`),
		regexp.MustCompile(`"DELETE\s+.*"\s*\+`),
		regexp.MustCompile(`query\s*=\s*.*\+.*`),
		regexp.MustCompile(`sql\s*=\s*.*\+.*`),
	}

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)

		// 除外パターンをチェック
		for _, pattern := range pa.config.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				return nil
			}
		}

		if pa.isBinaryFile(path) {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			for _, pattern := range sqlPatterns {
				if pattern.MatchString(line) {
					issues = append(issues, SecurityIssue{
						Type:        "sql_injection",
						Severity:    "high",
						Description: "Potential SQL injection vulnerability",
						File:        relPath,
						Line:        lineNum,
						Suggestion:  "Use parameterized queries or prepared statements",
						CWE:         "CWE-89",
					})
					break
				}
			}
		}

		return scanner.Err()
	})

	return issues, err
}

// XSS脆弱性をスキャン
func (pa *projectAnalyzer) scanXSS(projectPath string) ([]SecurityIssue, error) {
	issues := make([]SecurityIssue, 0)

	// XSSのパターン
	xssPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\.innerHTML\s*=\s*.*\+`),
		regexp.MustCompile(`document\.write\s*\(\s*.*\+`),
		regexp.MustCompile(`\$\(\s*.*\+.*\s*\)`),
		regexp.MustCompile(`dangerouslySetInnerHTML`),
	}

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)

		// JavaScriptファイルのみをチェック
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".js" && ext != ".jsx" && ext != ".ts" && ext != ".tsx" {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			for _, pattern := range xssPatterns {
				if pattern.MatchString(line) {
					issues = append(issues, SecurityIssue{
						Type:        "xss",
						Severity:    "medium",
						Description: "Potential XSS vulnerability",
						File:        relPath,
						Line:        lineNum,
						Suggestion:  "Sanitize user input before rendering or use safe DOM methods",
						CWE:         "CWE-79",
					})
					break
				}
			}
		}

		return scanner.Err()
	})

	return issues, err
}

// パストラバーサルをスキャン
func (pa *projectAnalyzer) scanPathTraversal(projectPath string) ([]SecurityIssue, error) {
	issues := make([]SecurityIssue, 0)

	// パストラバーサルのパターン
	pathTraversalPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\.\./`),
		regexp.MustCompile(`path\.join\s*\(\s*.*\+`),
		regexp.MustCompile(`os\.path\.join\s*\(\s*.*\+`),
		regexp.MustCompile(`filepath\.Join\s*\(\s*.*\+`),
	}

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)

		if pa.isBinaryFile(path) {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			for _, pattern := range pathTraversalPatterns {
				if pattern.MatchString(line) {
					issues = append(issues, SecurityIssue{
						Type:        "path_traversal",
						Severity:    "medium",
						Description: "Potential path traversal vulnerability",
						File:        relPath,
						Line:        lineNum,
						Suggestion:  "Validate and sanitize file paths, use allowlists for permitted files",
						CWE:         "CWE-22",
					})
					break
				}
			}
		}

		return scanner.Err()
	})

	return issues, err
}

// ハードコードされた認証情報をスキャン
func (pa *projectAnalyzer) scanHardcodedCredentials(projectPath string) ([]SecurityIssue, error) {
	issues := make([]SecurityIssue, 0)

	// 認証情報のパターン
	credentialPatterns := map[string]*regexp.Regexp{
		"Password": regexp.MustCompile(`(?i)(password|pwd)\s*[:=]\s*["']([^"'\s]{6,})["']`),
		"Username": regexp.MustCompile(`(?i)(username|user)\s*[:=]\s*["']([^"'\s]{3,})["']`),
		"Token":    regexp.MustCompile(`(?i)(token|auth)\s*[:=]\s*["']([a-zA-Z0-9_-]{10,})["']`),
		"Secret":   regexp.MustCompile(`(?i)(secret|key)\s*[:=]\s*["']([a-zA-Z0-9_-]{10,})["']`),
	}

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)

		// 除外パターンをチェック
		for _, pattern := range pa.config.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				return nil
			}
		}

		if pa.isBinaryFile(path) {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// コメント行はスキップ
			trimmedLine := strings.TrimSpace(line)
			if strings.HasPrefix(trimmedLine, "//") || strings.HasPrefix(trimmedLine, "#") {
				continue
			}

			for credType, pattern := range credentialPatterns {
				if matches := pattern.FindStringSubmatch(line); matches != nil {
					// 明らかにテスト用やプレースホルダーの値は除外
					value := matches[2]
					if pa.isPlaceholderValue(value) {
						continue
					}

					issues = append(issues, SecurityIssue{
						Type:        "hardcoded_credentials",
						Severity:    "high",
						Description: fmt.Sprintf("Hardcoded %s found in source code", strings.ToLower(credType)),
						File:        relPath,
						Line:        lineNum,
						Suggestion:  "Move credentials to environment variables or secure configuration",
						CWE:         "CWE-798",
					})
				}
			}
		}

		return scanner.Err()
	})

	return issues, err
}

// ヘルパー関数

func (pa *projectAnalyzer) isBinaryFile(filePath string) bool {
	// ファイル拡張子でバイナリファイルを判定
	binaryExts := []string{".exe", ".dll", ".so", ".dylib", ".bin", ".jpg", ".jpeg", ".png", ".gif", ".pdf", ".zip", ".tar", ".gz"}
	ext := strings.ToLower(filepath.Ext(filePath))

	for _, binaryExt := range binaryExts {
		if ext == binaryExt {
			return true
		}
	}

	return false
}

func (pa *projectAnalyzer) isConfigFile(filePath string) bool {
	fileName := strings.ToLower(filepath.Base(filePath))
	configFiles := []string{".env", "config.json", "config.yaml", "config.yml", "settings.json"}

	for _, configFile := range configFiles {
		if fileName == configFile || strings.Contains(fileName, "config") {
			return true
		}
	}

	return false
}

func (pa *projectAnalyzer) isPlaceholderValue(value string) bool {
	placeholders := []string{
		"password", "123456", "admin", "root", "test", "demo", "example",
		"changeme", "placeholder", "your_password_here", "your_api_key_here",
		"xxxxxxxx", "aaaaaaaa", "bbbbbbbb",
	}

	lowerValue := strings.ToLower(value)
	for _, placeholder := range placeholders {
		if strings.Contains(lowerValue, placeholder) {
			return true
		}
	}

	// 単純な繰り返しパターンもプレースホルダーとみなす
	if len(value) > 1 {
		firstChar := value[0]
		allSame := true
		for _, char := range value {
			if char != rune(firstChar) {
				allSame = false
				break
			}
		}
		if allSame {
			return true
		}
	}

	return false
}

func (pa *projectAnalyzer) sortSecurityIssuesBySeverity(issues []SecurityIssue) {
	severityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	// ソート処理（Go標準ライブラリのsortパッケージを使用）
	for i := 0; i < len(issues)-1; i++ {
		for j := i + 1; j < len(issues); j++ {
			if severityOrder[issues[i].Severity] > severityOrder[issues[j].Severity] {
				issues[i], issues[j] = issues[j], issues[i]
			}
		}
	}
}
