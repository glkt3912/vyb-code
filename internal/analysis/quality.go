package analysis

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 品質メトリクスの分析実装

// プロジェクトの品質メトリクスを分析
func (pa *projectAnalyzer) AnalyzeQuality(projectPath string) (*QualityMetrics, error) {
	metrics := &QualityMetrics{
		Details: make(map[string]float64),
	}

	// 並行してメトリクスを計算
	var err error

	// テストカバレッジ
	metrics.TestCoverage, _ = pa.calculateTestCoverage(projectPath)

	// コード複雑度
	metrics.CodeComplexity, _ = pa.calculateCodeComplexity(projectPath)

	// 保守性スコア
	metrics.Maintainability, _ = pa.calculateMaintainability(projectPath)

	// 重複度
	metrics.Duplication, _ = pa.calculateDuplication(projectPath)

	// 技術的負債
	metrics.TechnicalDebt, _ = pa.calculateTechnicalDebt(projectPath)

	// 問題数（lint警告等）
	metrics.IssueCount, metrics.LintWarnings, _ = pa.calculateIssues(projectPath)

	// セキュリティスコア
	metrics.SecurityScore, _ = pa.calculateSecurityScore(projectPath)

	// パフォーマンススコア
	metrics.PerformanceScore, _ = pa.calculatePerformanceScore(projectPath)

	// 詳細メトリクス
	pa.calculateDetailedMetrics(projectPath, metrics)

	return metrics, err
}

// テストカバレッジを計算
func (pa *projectAnalyzer) calculateTestCoverage(projectPath string) (float64, error) {
	// 言語別にテストカバレッジを計算
	if pa.hasGoMod(projectPath) {
		return pa.calculateGoCoverage(projectPath)
	}

	if pa.hasPackageJson(projectPath) {
		return pa.calculateJSCoverage(projectPath)
	}

	if pa.hasPythonProject(projectPath) {
		return pa.calculatePythonCoverage(projectPath)
	}

	// デフォルトはファイルベースの推定
	return pa.estimateCoverageFromFiles(projectPath)
}

// Go プロジェクトのテストカバレッジ
func (pa *projectAnalyzer) calculateGoCoverage(projectPath string) (float64, error) {
	// go test -coverprofile を実行してカバレッジを取得
	cmd := exec.Command("go", "test", "-coverprofile=coverage.out", "./...")
	cmd.Dir = projectPath

	if err := cmd.Run(); err != nil {
		return 0.0, err
	}

	// カバレッジファイルを読み込み
	coverageFile := filepath.Join(projectPath, "coverage.out")
	defer os.Remove(coverageFile) // 一時ファイルを削除

	if _, err := os.Stat(coverageFile); os.IsNotExist(err) {
		return 0.0, nil
	}

	// go tool cover で総カバレッジを計算
	cmd = exec.Command("go", "tool", "cover", "-func=coverage.out")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return 0.0, err
	}

	// 出力から総カバレッジを抽出
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "total:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				coverageStr := strings.TrimSuffix(parts[2], "%")
				if coverage, err := strconv.ParseFloat(coverageStr, 64); err == nil {
					return coverage, nil
				}
			}
		}
	}

	return 0.0, nil
}

// JavaScript プロジェクトのテストカバレッジ
func (pa *projectAnalyzer) calculateJSCoverage(projectPath string) (float64, error) {
	// package.json から coverage スクリプトを確認
	packageJsonPath := filepath.Join(projectPath, "package.json")
	if content, err := os.ReadFile(packageJsonPath); err == nil {
		if strings.Contains(string(content), "jest") || strings.Contains(string(content), "coverage") {
			// Jest でカバレッジを実行
			cmd := exec.Command("npm", "run", "test", "--", "--coverage", "--silent")
			cmd.Dir = projectPath

			output, err := cmd.Output()
			if err != nil {
				return 0.0, nil // エラーでも0を返す
			}

			// カバレッジレポートから数値を抽出
			return pa.extractCoverageFromJestOutput(string(output))
		}
	}

	return 0.0, nil
}

// Python プロジェクトのテストカバレッジ
func (pa *projectAnalyzer) calculatePythonCoverage(projectPath string) (float64, error) {
	// pytest-cov または coverage.py を使用してカバレッジを計算
	cmd := exec.Command("python", "-m", "pytest", "--cov=.", "--cov-report=term-missing", "--quiet")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return 0.0, nil
	}

	return pa.extractCoverageFromPytestOutput(string(output))
}

// ファイルからカバレッジを推定
func (pa *projectAnalyzer) estimateCoverageFromFiles(projectPath string) (float64, error) {
	sourceFiles := 0
	testFiles := 0

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

		fileName := strings.ToLower(info.Name())
		if pa.isSourceFile(fileName) {
			sourceFiles++
			if pa.isTestFile(fileName) {
				testFiles++
			}
		}

		return nil
	})

	if err != nil || sourceFiles == 0 {
		return 0.0, err
	}

	// 簡易推定: テストファイルの割合に基づく
	estimatedCoverage := float64(testFiles) / float64(sourceFiles) * 60.0 // 最大60%の推定
	if estimatedCoverage > 100.0 {
		estimatedCoverage = 85.0 // 上限設定
	}

	return estimatedCoverage, nil
}

// コード複雑度を計算
func (pa *projectAnalyzer) calculateCodeComplexity(projectPath string) (float64, error) {
	totalComplexity := 0
	fileCount := 0

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

		if pa.isSourceFile(strings.ToLower(info.Name())) {
			if complexity, err := pa.calculateFileComplexity(path); err == nil {
				totalComplexity += complexity
				fileCount++
			}
		}

		return nil
	})

	if err != nil || fileCount == 0 {
		return 0.0, err
	}

	averageComplexity := float64(totalComplexity) / float64(fileCount)
	return averageComplexity, nil
}

// 保守性スコアを計算
func (pa *projectAnalyzer) calculateMaintainability(projectPath string) (float64, error) {
	// 複数の要因を組み合わせて保守性スコアを計算
	var factors []float64

	// ファイルサイズの適正性
	if sizeScore, err := pa.calculateFileSizeScore(projectPath); err == nil {
		factors = append(factors, sizeScore)
	}

	// コメント率
	if commentScore, err := pa.calculateCommentRatio(projectPath); err == nil {
		factors = append(factors, commentScore)
	}

	// ディレクトリ構造の整理度
	if structureScore, err := pa.calculateStructureScore(projectPath); err == nil {
		factors = append(factors, structureScore)
	}

	// 命名規則の一貫性
	if namingScore, err := pa.calculateNamingConsistency(projectPath); err == nil {
		factors = append(factors, namingScore)
	}

	if len(factors) == 0 {
		return 50.0, nil // デフォルト値
	}

	// 重み付き平均を計算
	totalScore := 0.0
	for _, score := range factors {
		totalScore += score
	}

	return totalScore / float64(len(factors)), nil
}

// 重複度を計算
func (pa *projectAnalyzer) calculateDuplication(projectPath string) (float64, error) {
	duplicateLines := 0
	totalLines := 0

	// ファイル内容をハッシュ化して重複を検出
	lineHashes := make(map[string]int)

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

		if pa.isSourceFile(strings.ToLower(info.Name())) {
			if lines, duplicates, err := pa.analyzeDuplicatesInFile(path, lineHashes); err == nil {
				totalLines += lines
				duplicateLines += duplicates
			}
		}

		return nil
	})

	if err != nil || totalLines == 0 {
		return 0.0, err
	}

	duplicationPercent := float64(duplicateLines) / float64(totalLines) * 100.0
	return duplicationPercent, nil
}

// 技術的負債を計算
func (pa *projectAnalyzer) calculateTechnicalDebt(projectPath string) (time.Duration, error) {
	debtMinutes := 0

	// TODO、FIXME、HACK などのコメントを検索
	techDebtPatterns := []string{"TODO", "FIXME", "HACK", "XXX", "BUG", "DEBT"}
	techDebtRegex := make([]*regexp.Regexp, len(techDebtPatterns))

	for i, pattern := range techDebtPatterns {
		techDebtRegex[i] = regexp.MustCompile(`(?i)(//|#|<!--).*` + pattern)
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

		if pa.isSourceFile(strings.ToLower(info.Name())) {
			if debt, err := pa.calculateFileDebt(path, techDebtRegex); err == nil {
				debtMinutes += debt
			}
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return time.Duration(debtMinutes) * time.Minute, nil
}

// 問題数を計算
func (pa *projectAnalyzer) calculateIssues(projectPath string) (int, int, error) {
	issues := 0
	lintWarnings := 0

	// 言語別にlintツールを実行
	if pa.hasGoMod(projectPath) {
		if goIssues, goWarnings, err := pa.runGoLint(projectPath); err == nil {
			issues += goIssues
			lintWarnings += goWarnings
		}
	}

	if pa.hasPackageJson(projectPath) {
		if jsIssues, jsWarnings, err := pa.runJSLint(projectPath); err == nil {
			issues += jsIssues
			lintWarnings += jsWarnings
		}
	}

	// TODO、FIXME等のコメントも問題としてカウント
	if commentIssues, err := pa.countCommentIssues(projectPath); err == nil {
		issues += commentIssues
	}

	return issues, lintWarnings, nil
}

// セキュリティスコアを計算
func (pa *projectAnalyzer) calculateSecurityScore(projectPath string) (float64, error) {
	securityScore := 100.0 // 最大スコア

	// セキュリティ問題の検出
	securityIssues, err := pa.AnalyzeSecurity(projectPath)
	if err != nil {
		return 50.0, err // エラー時は中間値
	}

	// 重要度別にスコアを減点
	for _, issue := range securityIssues {
		switch issue.Severity {
		case "critical":
			securityScore -= 20.0
		case "high":
			securityScore -= 10.0
		case "medium":
			securityScore -= 5.0
		case "low":
			securityScore -= 2.0
		}
	}

	if securityScore < 0 {
		securityScore = 0
	}

	return securityScore, nil
}

// パフォーマンススコアを計算
func (pa *projectAnalyzer) calculatePerformanceScore(projectPath string) (float64, error) {
	score := 100.0

	// ファイルサイズの問題
	if largFiles, err := pa.findLargeFiles(projectPath); err == nil {
		score -= float64(len(largFiles)) * 5.0 // 大きなファイル1つあたり5点減点
	}

	// 深いネスト構造
	if deepNesting, err := pa.countDeepNesting(projectPath); err == nil {
		score -= float64(deepNesting) * 2.0 // 深いネスト1つあたり2点減点
	}

	// パフォーマンス関連の問題パターン
	if perfIssues, err := pa.countPerformanceIssues(projectPath); err == nil {
		score -= float64(perfIssues) * 3.0
	}

	if score < 0 {
		score = 0
	}

	return score, nil
}

// 詳細メトリクスを計算
func (pa *projectAnalyzer) calculateDetailedMetrics(projectPath string, metrics *QualityMetrics) {
	// ファイル数
	if fileCount, err := pa.countFiles(projectPath); err == nil {
		metrics.Details["file_count"] = float64(fileCount)
	}

	// 総行数
	if lineCount, err := pa.countTotalLines(projectPath); err == nil {
		metrics.Details["total_lines"] = float64(lineCount)
	}

	// 関数数
	if functionCount, err := pa.countFunctions(projectPath); err == nil {
		metrics.Details["function_count"] = float64(functionCount)
	}

	// クラス数
	if classCount, err := pa.countClasses(projectPath); err == nil {
		metrics.Details["class_count"] = float64(classCount)
	}

	// コメント率
	if commentRatio, err := pa.calculateCommentRatio(projectPath); err == nil {
		metrics.Details["comment_ratio"] = commentRatio
	}
}

// ヘルパー関数群

func (pa *projectAnalyzer) hasGoMod(projectPath string) bool {
	_, err := os.Stat(filepath.Join(projectPath, "go.mod"))
	return err == nil
}

func (pa *projectAnalyzer) hasPackageJson(projectPath string) bool {
	_, err := os.Stat(filepath.Join(projectPath, "package.json"))
	return err == nil
}

func (pa *projectAnalyzer) hasPythonProject(projectPath string) bool {
	files := []string{"requirements.txt", "setup.py", "pyproject.toml", "Pipfile"}
	for _, file := range files {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
			return true
		}
	}
	return false
}

func (pa *projectAnalyzer) isSourceFile(fileName string) bool {
	sourceExts := []string{".go", ".js", ".ts", ".py", ".java", ".rs", ".php", ".rb", ".c", ".cpp", ".cs", ".dart"}
	for _, ext := range sourceExts {
		if strings.HasSuffix(fileName, ext) {
			return true
		}
	}
	return false
}

func (pa *projectAnalyzer) isTestFile(fileName string) bool {
	return strings.Contains(fileName, "test") || strings.Contains(fileName, "spec")
}

// Jest出力からカバレッジを抽出
func (pa *projectAnalyzer) extractCoverageFromJestOutput(output string) (float64, error) {
	// Jest のカバレッジ出力から数値を抽出
	coverageRegex := regexp.MustCompile(`All files\s+\|\s+([0-9.]+)`)
	matches := coverageRegex.FindStringSubmatch(output)

	if len(matches) > 1 {
		if coverage, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return coverage, nil
		}
	}

	return 0.0, nil
}

// pytest出力からカバレッジを抽出
func (pa *projectAnalyzer) extractCoverageFromPytestOutput(output string) (float64, error) {
	// pytest-cov の出力から数値を抽出
	coverageRegex := regexp.MustCompile(`TOTAL\s+\d+\s+\d+\s+(\d+)%`)
	matches := coverageRegex.FindStringSubmatch(output)

	if len(matches) > 1 {
		if coverage, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return coverage, nil
		}
	}

	return 0.0, nil
}

// 残りのヘルパー関数は実装の詳細により省略...
// 実際のプロダクトでは各メトリクスの詳細な計算ロジックを実装

// 簡易実装例
func (pa *projectAnalyzer) calculateFileSizeScore(projectPath string) (float64, error) {
	// ファイルサイズの分布から適正性を評価
	return 80.0, nil // 仮の値
}

func (pa *projectAnalyzer) calculateCommentRatio(projectPath string) (float64, error) {
	totalLines := 0
	commentLines := 0

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if pa.isSourceFile(strings.ToLower(info.Name())) {
			if lines, comments, err := pa.countCommentsInFile(path); err == nil {
				totalLines += lines
				commentLines += comments
			}
		}
		return nil
	})

	if err != nil || totalLines == 0 {
		return 0.0, err
	}

	return float64(commentLines) / float64(totalLines) * 100.0, nil
}

func (pa *projectAnalyzer) countCommentsInFile(filePath string) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	totalLines := 0
	commentLines := 0
	scanner := bufio.NewScanner(file)

	// 簡易的なコメント検出
	commentPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\s*//`),  // JavaScript, Go, etc.
		regexp.MustCompile(`^\s*#`),   // Python, Shell, etc.
		regexp.MustCompile(`^\s*/\*`), // Multi-line comments start
		regexp.MustCompile(`^\s*\*`),  // Multi-line comments middle
	}

	for scanner.Scan() {
		line := scanner.Text()
		totalLines++

		for _, pattern := range commentPatterns {
			if pattern.MatchString(line) {
				commentLines++
				break
			}
		}
	}

	return totalLines, commentLines, scanner.Err()
}

func (pa *projectAnalyzer) calculateStructureScore(projectPath string) (float64, error) {
	// ディレクトリ構造の整理度を評価
	return 75.0, nil // 仮の値
}

func (pa *projectAnalyzer) calculateNamingConsistency(projectPath string) (float64, error) {
	// 命名規則の一貫性を評価
	return 70.0, nil // 仮の値
}

func (pa *projectAnalyzer) analyzeDuplicatesInFile(filePath string, lineHashes map[string]int) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	lines := 0
	duplicates := 0
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "//") && !strings.HasPrefix(line, "#") {
			lines++
			lineHashes[line]++
			if lineHashes[line] > 1 {
				duplicates++
			}
		}
	}

	return lines, duplicates, scanner.Err()
}

func (pa *projectAnalyzer) calculateFileDebt(filePath string, patterns []*regexp.Regexp) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	debtMinutes := 0
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				// TODOの種類に応じて負債時間を設定
				if strings.Contains(strings.ToUpper(line), "FIXME") {
					debtMinutes += 30 // 修正に30分
				} else if strings.Contains(strings.ToUpper(line), "TODO") {
					debtMinutes += 15 // 実装に15分
				} else {
					debtMinutes += 10 // その他は10分
				}
				break
			}
		}
	}

	return debtMinutes, scanner.Err()
}

func (pa *projectAnalyzer) runGoLint(projectPath string) (int, int, error) {
	// go vet を実行してlint結果を取得
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// go vet はwarningがあるとエラーを返すが、出力は有効
		lines := strings.Split(string(output), "\n")
		warnings := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				warnings++
			}
		}
		return warnings, warnings, nil
	}

	return 0, 0, nil
}

func (pa *projectAnalyzer) runJSLint(projectPath string) (int, int, error) {
	// eslint を実行
	cmd := exec.Command("npx", "eslint", ".", "--format", "json")
	cmd.Dir = projectPath

	_, err := cmd.Output()
	if err != nil {
		return 0, 0, nil // eslintが無い場合は無視
	}

	// JSON出力を解析して警告数をカウント
	// 実装の詳細は省略
	return 0, 0, nil
}

func (pa *projectAnalyzer) countCommentIssues(projectPath string) (int, error) {
	issues := 0
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)TODO`),
		regexp.MustCompile(`(?i)FIXME`),
		regexp.MustCompile(`(?i)HACK`),
		regexp.MustCompile(`(?i)XXX`),
	}

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if pa.isSourceFile(strings.ToLower(info.Name())) {
			if fileIssues, err := pa.countFileIssues(path, patterns); err == nil {
				issues += fileIssues
			}
		}
		return nil
	})

	return issues, err
}

func (pa *projectAnalyzer) countFileIssues(filePath string, patterns []*regexp.Regexp) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	issues := 0
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				issues++
				break
			}
		}
	}

	return issues, scanner.Err()
}

func (pa *projectAnalyzer) findLargeFiles(projectPath string) ([]string, error) {
	const maxSize = 10 * 1024 * 1024 // 10MB
	largeFiles := make([]string, 0)

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if info.Size() > maxSize {
			relPath, _ := filepath.Rel(projectPath, path)
			largeFiles = append(largeFiles, relPath)
		}
		return nil
	})

	return largeFiles, err
}

func (pa *projectAnalyzer) countDeepNesting(projectPath string) (int, error) {
	// 深いネスト構造をカウント（簡易実装）
	return 0, nil
}

func (pa *projectAnalyzer) countPerformanceIssues(projectPath string) (int, error) {
	// パフォーマンス問題をカウント（簡易実装）
	return 0, nil
}

func (pa *projectAnalyzer) countFiles(projectPath string) (int, error) {
	count := 0
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		count++
		return nil
	})
	return count, err
}

func (pa *projectAnalyzer) countTotalLines(projectPath string) (int, error) {
	totalLines := 0
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if pa.isSourceFile(strings.ToLower(info.Name())) {
			if lines, err := pa.countLines(path); err == nil {
				totalLines += lines
			}
		}
		return nil
	})
	return totalLines, err
}

func (pa *projectAnalyzer) countFunctions(projectPath string) (int, error) {
	// 関数数をカウント（簡易実装）
	return 0, nil
}

func (pa *projectAnalyzer) countClasses(projectPath string) (int, error) {
	// クラス数をカウント（簡易実装）
	return 0, nil
}
