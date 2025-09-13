package analysis

import (
	"fmt"
	"sort"
)

// 推奨事項の生成実装

// 推奨事項を生成
func (pa *projectAnalyzer) GenerateRecommendations(analysis *ProjectAnalysis) ([]Recommendation, error) {
	recommendations := make([]Recommendation, 0)

	// 各カテゴリの推奨事項を生成
	generators := []func(*ProjectAnalysis) []Recommendation{
		pa.generateSecurityRecommendations,
		pa.generatePerformanceRecommendations,
		pa.generateMaintainabilityRecommendations,
		pa.generateBestPracticeRecommendations,
		pa.generateDependencyRecommendations,
		pa.generateTestingRecommendations,
		pa.generateDocumentationRecommendations,
	}

	for _, generator := range generators {
		generated := generator(analysis)
		recommendations = append(recommendations, generated...)
	}

	// 優先度順にソート
	pa.sortRecommendationsByPriority(recommendations)

	return recommendations, nil
}

// セキュリティ推奨事項
func (pa *projectAnalyzer) generateSecurityRecommendations(analysis *ProjectAnalysis) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// セキュリティスコアが低い場合
	if analysis.QualityMetrics != nil && analysis.QualityMetrics.SecurityScore < 70 {
		recommendations = append(recommendations, Recommendation{
			Type:        "security",
			Priority:    "high",
			Title:       "セキュリティスコアの改善",
			Description: fmt.Sprintf("プロジェクトのセキュリティスコア（%.1f点）が低いため、セキュリティ強化が必要です", analysis.QualityMetrics.SecurityScore),
			Action:      "セキュリティ問題を修正し、セキュリティテストを追加してください",
			Files:       pa.getSecurityIssueFiles(analysis),
			Effort:      "high",
			Impact:      "high",
		})
	}

	// クリティカルなセキュリティ問題がある場合
	criticalIssues := pa.getCriticalSecurityIssues(analysis.SecurityIssues)
	if len(criticalIssues) > 0 {
		recommendations = append(recommendations, Recommendation{
			Type:        "security",
			Priority:    "critical",
			Title:       "クリティカルなセキュリティ問題の修正",
			Description: fmt.Sprintf("%d個のクリティカルなセキュリティ問題が検出されました", len(criticalIssues)),
			Action:      "シークレットの漏洩、SQLインジェクション、XSS脆弱性などを即座に修正してください",
			Files:       pa.getIssueFiles(criticalIssues),
			Effort:      "high",
			Impact:      "critical",
		})
	}

	// .envファイルがgitignoreにない場合
	if pa.hasEnvFileNotInGitignore(analysis) {
		recommendations = append(recommendations, Recommendation{
			Type:        "security",
			Priority:    "high",
			Title:       "環境変数ファイルの保護",
			Description: ".envファイルがGitリポジトリに含まれる可能性があります",
			Action:      ".envファイルを.gitignoreに追加してください",
			Files:       []string{".env", ".gitignore"},
			Effort:      "low",
			Impact:      "high",
		})
	}

	return recommendations
}

// パフォーマンス推奨事項
func (pa *projectAnalyzer) generatePerformanceRecommendations(analysis *ProjectAnalysis) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// 大きなファイルがある場合
	if analysis.FileStructure != nil && len(analysis.FileStructure.LargestFiles) > 0 {
		largeFiles := make([]string, 0)
		for _, file := range analysis.FileStructure.LargestFiles {
			if file.Size > 500*1024 { // 500KB以上
				largeFiles = append(largeFiles, file.Path)
			}
		}

		if len(largeFiles) > 0 {
			recommendations = append(recommendations, Recommendation{
				Type:        "performance",
				Priority:    "medium",
				Title:       "大きなファイルの最適化",
				Description: fmt.Sprintf("%d個の大きなファイル（500KB以上）が検出されました", len(largeFiles)),
				Action:      "大きなファイルを分割するか、最適化を検討してください",
				Files:       largeFiles,
				Effort:      "medium",
				Impact:      "medium",
			})
		}
	}

	// 複雑度が高い場合
	if analysis.QualityMetrics != nil && analysis.QualityMetrics.CodeComplexity > 15 {
		recommendations = append(recommendations, Recommendation{
			Type:        "performance",
			Priority:    "medium",
			Title:       "コード複雑度の改善",
			Description: fmt.Sprintf("平均複雑度（%.1f）が高いため、パフォーマンスに影響する可能性があります", analysis.QualityMetrics.CodeComplexity),
			Action:      "複雑な関数を分割し、アルゴリズムを最適化してください",
			Files:       []string{},
			Effort:      "high",
			Impact:      "medium",
		})
	}

	// 古い依存関係がある場合
	outdatedDeps := pa.getOutdatedDependencies(analysis.Dependencies)
	if len(outdatedDeps) > 0 {
		recommendations = append(recommendations, Recommendation{
			Type:        "performance",
			Priority:    "low",
			Title:       "依存関係の更新",
			Description: fmt.Sprintf("%d個の古い依存関係が検出されました", len(outdatedDeps)),
			Action:      "依存関係を最新バージョンに更新して、パフォーマンス向上とセキュリティ強化を図ってください",
			Files:       pa.getDependencyFiles(analysis),
			Effort:      "medium",
			Impact:      "low",
		})
	}

	return recommendations
}

// 保守性推奨事項
func (pa *projectAnalyzer) generateMaintainabilityRecommendations(analysis *ProjectAnalysis) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// 保守性スコアが低い場合
	if analysis.QualityMetrics != nil && analysis.QualityMetrics.Maintainability < 60 {
		recommendations = append(recommendations, Recommendation{
			Type:        "maintainability",
			Priority:    "medium",
			Title:       "保守性の向上",
			Description: fmt.Sprintf("保守性スコア（%.1f点）が低いため、メンテナンスが困難になる可能性があります", analysis.QualityMetrics.Maintainability),
			Action:      "コードの構造化、コメントの追加、命名規則の統一を行ってください",
			Files:       []string{},
			Effort:      "high",
			Impact:      "high",
		})
	}

	// 重複コードが多い場合
	if analysis.QualityMetrics != nil && analysis.QualityMetrics.Duplication > 10 {
		recommendations = append(recommendations, Recommendation{
			Type:        "maintainability",
			Priority:    "medium",
			Title:       "コード重複の削減",
			Description: fmt.Sprintf("コード重複率（%.1f%%）が高いため、保守性に影響しています", analysis.QualityMetrics.Duplication),
			Action:      "共通のコードを関数やモジュールに抽出して重複を減らしてください",
			Files:       []string{},
			Effort:      "medium",
			Impact:      "medium",
		})
	}

	// 技術的負債が多い場合
	if analysis.QualityMetrics != nil && analysis.QualityMetrics.TechnicalDebt.Hours() > 8 {
		recommendations = append(recommendations, Recommendation{
			Type:        "maintainability",
			Priority:    "medium",
			Title:       "技術的負債の解消",
			Description: fmt.Sprintf("技術的負債（%s）が蓄積されています", analysis.QualityMetrics.TechnicalDebt.String()),
			Action:      "TODO、FIXME、HACKコメントを解決し、技術的負債を減らしてください",
			Files:       []string{},
			Effort:      "high",
			Impact:      "medium",
		})
	}

	return recommendations
}

// ベストプラクティス推奨事項
func (pa *projectAnalyzer) generateBestPracticeRecommendations(analysis *ProjectAnalysis) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// README.mdがない場合
	if !pa.hasReadmeFile(analysis) {
		recommendations = append(recommendations, Recommendation{
			Type:        "best_practice",
			Priority:    "low",
			Title:       "READMEファイルの追加",
			Description: "プロジェクトにREADME.mdファイルがありません",
			Action:      "プロジェクトの概要、インストール方法、使用方法を記述したREADME.mdを作成してください",
			Files:       []string{"README.md"},
			Effort:      "low",
			Impact:      "low",
		})
	}

	// ライセンスファイルがない場合
	if !pa.hasLicenseFile(analysis) {
		recommendations = append(recommendations, Recommendation{
			Type:        "best_practice",
			Priority:    "low",
			Title:       "ライセンスファイルの追加",
			Description: "プロジェクトにライセンスファイルがありません",
			Action:      "適切なライセンス（MIT、Apache、GPLなど）を選択してLICENSEファイルを追加してください",
			Files:       []string{"LICENSE"},
			Effort:      "low",
			Impact:      "low",
		})
	}

	// .gitignoreが不十分な場合
	if pa.hasInsufficientGitignore(analysis) {
		recommendations = append(recommendations, Recommendation{
			Type:        "best_practice",
			Priority:    "low",
			Title:       ".gitignoreの改善",
			Description: ".gitignoreファイルが不十分で、不要なファイルがコミットされる可能性があります",
			Action:      "言語固有の.gitignoreテンプレートを使用して、適切なファイルを除外してください",
			Files:       []string{".gitignore"},
			Effort:      "low",
			Impact:      "low",
		})
	}

	// アーキテクチャパターンが検出されない場合
	if analysis.FileStructure != nil && len(analysis.FileStructure.Patterns) == 0 {
		recommendations = append(recommendations, Recommendation{
			Type:        "best_practice",
			Priority:    "low",
			Title:       "アーキテクチャパターンの適用",
			Description: "明確なアーキテクチャパターンが検出されませんでした",
			Action:      "MVC、MVP、Clean Architectureなどの適切なアーキテクチャパターンを導入してください",
			Files:       []string{},
			Effort:      "high",
			Impact:      "medium",
		})
	}

	return recommendations
}

// 依存関係推奨事項
func (pa *projectAnalyzer) generateDependencyRecommendations(analysis *ProjectAnalysis) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// 脆弱な依存関係がある場合
	vulnerableDeps := pa.getVulnerableDependencies(analysis.Dependencies)
	if len(vulnerableDeps) > 0 {
		recommendations = append(recommendations, Recommendation{
			Type:        "security",
			Priority:    "high",
			Title:       "脆弱な依存関係の更新",
			Description: fmt.Sprintf("%d個の脆弱性のある依存関係が検出されました", len(vulnerableDeps)),
			Action:      "セキュリティ脆弱性のある依存関係を安全なバージョンに更新してください",
			Files:       pa.getDependencyFiles(analysis),
			Effort:      "medium",
			Impact:      "high",
		})
	}

	// 使用されていない依存関係がある場合（簡易チェック）
	if analysis.Language == "JavaScript" && pa.hasUnusedDependencies(analysis) {
		recommendations = append(recommendations, Recommendation{
			Type:        "maintainability",
			Priority:    "low",
			Title:       "未使用依存関係の削除",
			Description: "使用されていない可能性のある依存関係が検出されました",
			Action:      "npm depcheck や類似ツールを使用して未使用の依存関係を特定・削除してください",
			Files:       []string{"package.json"},
			Effort:      "medium",
			Impact:      "low",
		})
	}

	return recommendations
}

// テスト推奨事項
func (pa *projectAnalyzer) generateTestingRecommendations(analysis *ProjectAnalysis) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// テストカバレッジが低い場合
	if analysis.QualityMetrics != nil && analysis.QualityMetrics.TestCoverage < 50 {
		recommendations = append(recommendations, Recommendation{
			Type:        "testing",
			Priority:    "medium",
			Title:       "テストカバレッジの向上",
			Description: fmt.Sprintf("テストカバレッジ（%.1f%%）が低いため、品質リスクが高まっています", analysis.QualityMetrics.TestCoverage),
			Action:      "ユニットテスト、統合テストを追加してカバレッジを70%以上に向上させてください",
			Files:       []string{},
			Effort:      "high",
			Impact:      "high",
		})
	}

	// テストフレームワークが設定されていない場合
	if analysis.TestingFramework == nil {
		recommendations = append(recommendations, Recommendation{
			Type:        "testing",
			Priority:    "medium",
			Title:       "テストフレームワークの導入",
			Description: "テストフレームワークが設定されていません",
			Action:      fmt.Sprintf("%s向けのテストフレームワークを導入してください", analysis.Language),
			Files:       []string{},
			Effort:      "medium",
			Impact:      "high",
		})
	}

	return recommendations
}

// ドキュメント推奨事項
func (pa *projectAnalyzer) generateDocumentationRecommendations(analysis *ProjectAnalysis) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// コメント率が低い場合
	if details, exists := analysis.QualityMetrics.Details["comment_ratio"]; exists && details < 10 {
		recommendations = append(recommendations, Recommendation{
			Type:        "maintainability",
			Priority:    "low",
			Title:       "コメントの充実",
			Description: fmt.Sprintf("コメント率（%.1f%%）が低く、コードの理解が困難になる可能性があります", details),
			Action:      "複雑なロジックや重要な関数にコメントを追加してください",
			Files:       []string{},
			Effort:      "low",
			Impact:      "medium",
		})
	}

	// APIドキュメントがない場合（Go/JavaScriptプロジェクト）
	if (analysis.Language == "Go" || analysis.Language == "JavaScript") && !pa.hasAPIDocumentation(analysis) {
		recommendations = append(recommendations, Recommendation{
			Type:        "best_practice",
			Priority:    "low",
			Title:       "APIドキュメントの作成",
			Description: "APIドキュメントが不足しています",
			Action:      "公開APIやメインな機能についてドキュメントを作成してください",
			Files:       []string{"docs/"},
			Effort:      "medium",
			Impact:      "low",
		})
	}

	return recommendations
}

// ヘルパー関数

func (pa *projectAnalyzer) getCriticalSecurityIssues(issues []SecurityIssue) []SecurityIssue {
	critical := make([]SecurityIssue, 0)
	for _, issue := range issues {
		if issue.Severity == "critical" {
			critical = append(critical, issue)
		}
	}
	return critical
}

func (pa *projectAnalyzer) getSecurityIssueFiles(analysis *ProjectAnalysis) []string {
	files := make(map[string]bool)
	for _, issue := range analysis.SecurityIssues {
		if issue.File != "" {
			files[issue.File] = true
		}
	}

	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}
	return result
}

func (pa *projectAnalyzer) getIssueFiles(issues []SecurityIssue) []string {
	files := make(map[string]bool)
	for _, issue := range issues {
		if issue.File != "" {
			files[issue.File] = true
		}
	}

	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}
	return result
}

func (pa *projectAnalyzer) getOutdatedDependencies(dependencies []Dependency) []Dependency {
	outdated := make([]Dependency, 0)
	for _, dep := range dependencies {
		if dep.Outdated {
			outdated = append(outdated, dep)
		}
	}
	return outdated
}

func (pa *projectAnalyzer) getVulnerableDependencies(dependencies []Dependency) []Dependency {
	vulnerable := make([]Dependency, 0)
	for _, dep := range dependencies {
		if len(dep.Vulnerabilities) > 0 {
			vulnerable = append(vulnerable, dep)
		}
	}
	return vulnerable
}

func (pa *projectAnalyzer) getDependencyFiles(analysis *ProjectAnalysis) []string {
	files := make(map[string]bool)
	for _, dep := range analysis.Dependencies {
		if dep.Source != "" {
			files[dep.Source] = true
		}
	}

	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}
	return result
}

func (pa *projectAnalyzer) hasEnvFileNotInGitignore(analysis *ProjectAnalysis) bool {
	// 簡易実装：実際は.envファイルの存在と.gitignoreの内容をチェック
	return false
}

func (pa *projectAnalyzer) hasReadmeFile(analysis *ProjectAnalysis) bool {
	if analysis.FileStructure == nil {
		return false
	}

	for _, dir := range analysis.FileStructure.Directories {
		if dir.Path == "." || dir.Path == "" {
			// ルートディレクトリでREADMEファイルを探す
			// 実際の実装ではファイル一覧をチェック
			return false // 簡易実装
		}
	}
	return false
}

func (pa *projectAnalyzer) hasLicenseFile(analysis *ProjectAnalysis) bool {
	// 簡易実装：実際はLICENSE、LICENSE.txt、LICENSE.mdファイルの存在をチェック
	return false
}

func (pa *projectAnalyzer) hasInsufficientGitignore(analysis *ProjectAnalysis) bool {
	// 簡易実装：実際は.gitignoreの内容を分析
	return false
}

func (pa *projectAnalyzer) hasUnusedDependencies(analysis *ProjectAnalysis) bool {
	// 簡易実装：実際は依存関係の使用状況を分析
	return false
}

func (pa *projectAnalyzer) hasAPIDocumentation(analysis *ProjectAnalysis) bool {
	// 簡易実装：実際はdocs/フォルダやコメント内のAPI文書をチェック
	return false
}

func (pa *projectAnalyzer) sortRecommendationsByPriority(recommendations []Recommendation) {
	priorityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	sort.Slice(recommendations, func(i, j int) bool {
		return priorityOrder[recommendations[i].Priority] < priorityOrder[recommendations[j].Priority]
	})
}
