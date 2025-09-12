package conversation

import (
	"fmt"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/analysis"
	"github.com/glkt/vyb-code/internal/config"
)

// 軽量プロアクティブマネージャー - Phase 2実装
type LightweightProactiveManager struct {
	lightAnalyzer    *analysis.LightweightAnalyzer
	asyncAnalyzer    *analysis.AsyncAnalyzer
	lastAnalysis     *analysis.ProjectAnalysis
	lastAnalysisTime time.Time
	config           *config.Config
	enabled          bool
}

// 新しい軽量プロアクティブマネージャーを作成
func NewLightweightProactiveManager(cfg *config.Config) *LightweightProactiveManager {
	if !cfg.IsProactiveEnabled() {
		return &LightweightProactiveManager{enabled: false}
	}

	analysisConfig := &analysis.AnalysisConfig{
		EnableCaching:  true,
		CacheExpiry:    15 * time.Minute,
		AnalysisDepth:  "quick",
		IncludeTests:   false,
		SecurityScan:   false,
		QualityMetrics: false,
		ExcludePatterns: []string{
			"node_modules/**", "vendor/**", ".git/**",
			"dist/**", "build/**", "target/**",
		},
		MaxFileSize: 256 * 1024, // 256KB制限（軽量化）
		Timeout:     time.Duration(cfg.Proactive.AnalysisTimeout) * time.Second,
	}

	return &LightweightProactiveManager{
		lightAnalyzer: analysis.NewLightweightAnalyzer(analysisConfig),
		asyncAnalyzer: analysis.NewAsyncAnalyzer(analysisConfig),
		config:        cfg,
		enabled:       true,
	}
}

// プロジェクト状態の軽量分析
func (lpm *LightweightProactiveManager) AnalyzeProjectLightly(projectPath string) (*analysis.ProjectAnalysis, error) {
	if !lpm.enabled || !lpm.config.IsProactiveEnabled() {
		return nil, fmt.Errorf("プロアクティブ機能が無効です")
	}

	// 最近分析した場合はキャッシュを使用
	if time.Since(lpm.lastAnalysisTime) < 5*time.Minute && lpm.lastAnalysis != nil {
		return lpm.lastAnalysis, nil
	}

	// 非同期で軽量分析を実行
	resultChan := lpm.asyncAnalyzer.AnalyzeLightweight(projectPath)

	// タイムアウト付きで結果を待機
	timeout := time.Duration(lpm.config.Proactive.AnalysisTimeout) * time.Second
	select {
	case result := <-resultChan:
		if result.Error != nil {
			return nil, result.Error
		}

		lpm.lastAnalysis = result.Analysis
		lpm.lastAnalysisTime = time.Now()
		return result.Analysis, nil

	case <-time.After(timeout):
		return nil, fmt.Errorf("分析タイムアウト: %v", timeout)
	}
}

// 軽量コンテキスト提案の生成
func (lpm *LightweightProactiveManager) GenerateLightContextSuggestions(userInput, projectPath string) []string {
	if !lpm.enabled || !lpm.config.IsProactiveEnabled() {
		return []string{}
	}

	suggestions := make([]string, 0)

	// プロジェクト分析（タイムアウト付き）
	analysis, err := lpm.AnalyzeProjectLightly(projectPath)
	if err != nil {
		// エラー時は基本的な提案のみ
		return lpm.generateBasicSuggestions(userInput)
	}

	// 入力内容に基づく軽量提案
	inputLower := strings.ToLower(userInput)

	// 言語固有の提案
	if analysis.Language != "" {
		if strings.Contains(inputLower, "error") || strings.Contains(inputLower, "エラー") {
			suggestions = append(suggestions,
				fmt.Sprintf("%s言語でのエラーハンドリングのベストプラクティス", analysis.Language))
		}

		if strings.Contains(inputLower, "test") || strings.Contains(inputLower, "テスト") {
			suggestions = append(suggestions,
				fmt.Sprintf("%sプロジェクトでのテスト戦略", analysis.Language))
		}

		if strings.Contains(inputLower, "optimize") || strings.Contains(inputLower, "最適化") {
			suggestions = append(suggestions,
				fmt.Sprintf("%s言語でのパフォーマンス最適化のポイント", analysis.Language))
		}
	}

	// ファイル構造に基づく提案
	if analysis.FileStructure != nil && analysis.FileStructure.TotalFiles > 0 {
		if analysis.FileStructure.TotalFiles > 50 {
			suggestions = append(suggestions, "大規模プロジェクトでのコード整理のテクニック")
		}

		// よく使われる拡張子に基づく提案
		for ext, count := range analysis.FileStructure.Languages {
			if count > 10 {
				switch ext {
				case ".go":
					suggestions = append(suggestions, "Go言語でのディレクトリ構造のベストプラクティス")
				case ".js", ".ts":
					suggestions = append(suggestions, "JavaScript/TypeScriptプロジェクトの設定最適化")
				case ".py":
					suggestions = append(suggestions, "Pythonプロジェクトでの依存関係管理")
				}
			}
		}
	}

	// 技術スタックに基づる提案
	for _, tech := range analysis.TechStack {
		if tech.Usage == "primary" {
			switch tech.Name {
			case "Docker":
				suggestions = append(suggestions, "Dockerコンテナの最適化とセキュリティ対策")
			case "Go Modules":
				suggestions = append(suggestions, "Go Modulesを使った依存関係の効率的管理")
			case "Node.js":
				suggestions = append(suggestions, "Node.jsアプリケーションのパフォーマンスチューニング")
			}
		}
	}

	// 提案が多すぎる場合は制限
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

// 基本的な提案生成（プロジェクト分析失敗時のフォールバック）
func (lpm *LightweightProactiveManager) generateBasicSuggestions(userInput string) []string {
	inputLower := strings.ToLower(userInput)
	suggestions := make([]string, 0)

	// 汎用的なキーワードベース提案
	if strings.Contains(inputLower, "git") {
		suggestions = append(suggestions, "Gitワークフローの改善提案")
	}

	if strings.Contains(inputLower, "debug") || strings.Contains(inputLower, "デバッグ") {
		suggestions = append(suggestions, "効果的なデバッグテクニック")
	}

	if strings.Contains(inputLower, "deploy") || strings.Contains(inputLower, "デプロイ") {
		suggestions = append(suggestions, "デプロイメント戦略の最適化")
	}

	return suggestions
}

// プロアクティブ応答拡張
func (lpm *LightweightProactiveManager) EnhanceResponse(originalResponse, userInput, projectPath string) string {
	if !lpm.enabled || !lpm.config.IsProactiveEnabled() || !lpm.config.Proactive.SmartSuggestions {
		return originalResponse
	}

	// 軽量分析を実行
	analysis, err := lpm.AnalyzeProjectLightly(projectPath)
	if err != nil {
		// エラー時は元の応答をそのまま返す
		return originalResponse
	}

	// 応答を拡張
	enhanced := originalResponse

	// プロジェクト情報を追加（簡潔に）
	if analysis.Language != "" && !strings.Contains(enhanced, analysis.Language) {
		enhanced += fmt.Sprintf("\n\n💡 このプロジェクトは%sを使用しています。", analysis.Language)
	}

	// 関連する技術スタック情報を追加
	if len(analysis.TechStack) > 0 {
		primaryTech := make([]string, 0)
		for _, tech := range analysis.TechStack {
			if tech.Usage == "primary" && tech.Name != analysis.Language {
				primaryTech = append(primaryTech, tech.Name)
			}
		}

		if len(primaryTech) > 0 && len(primaryTech) <= 2 {
			enhanced += fmt.Sprintf(" %sも活用されています。", strings.Join(primaryTech, "と"))
		}
	}

	return enhanced
}

// プロジェクト状態の簡易サマリー生成
func (lpm *LightweightProactiveManager) GenerateProjectSummary(projectPath string) string {
	if !lpm.enabled || !lpm.config.IsProactiveEnabled() {
		return ""
	}

	analysis, err := lpm.AnalyzeProjectLightly(projectPath)
	if err != nil {
		return ""
	}

	var parts []string

	// 基本情報
	if analysis.Language != "" {
		parts = append(parts, fmt.Sprintf("言語: %s", analysis.Language))
	}

	if analysis.FileStructure != nil && analysis.FileStructure.TotalFiles > 0 {
		parts = append(parts, fmt.Sprintf("ファイル数: %d", analysis.FileStructure.TotalFiles))
	}

	// Git情報
	if analysis.GitInfo != nil && analysis.GitInfo.Repository != "" {
		parts = append(parts, "Gitリポジトリ")
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("📊 プロジェクト: %s", strings.Join(parts, " • "))
}

// パフォーマンス統計の取得
func (lpm *LightweightProactiveManager) GetPerformanceStats() map[string]interface{} {
	stats := map[string]interface{}{
		"enabled":             lpm.enabled,
		"last_analysis_time":  lpm.lastAnalysisTime,
		"has_cached_analysis": lpm.lastAnalysis != nil,
	}

	if lpm.asyncAnalyzer != nil {
		asyncStats := lpm.asyncAnalyzer.GetStats()
		for k, v := range asyncStats {
			stats["async_"+k] = v
		}
	}

	return stats
}

// リソースクリーンアップ
func (lpm *LightweightProactiveManager) Close() error {
	if lpm.asyncAnalyzer != nil {
		return lpm.asyncAnalyzer.Close()
	}
	return nil
}

// プロアクティブ機能の動的切り替え
func (lpm *LightweightProactiveManager) UpdateConfig(cfg *config.Config) {
	lpm.config = cfg
	lpm.enabled = cfg.IsProactiveEnabled()
}
