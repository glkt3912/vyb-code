package conversation

import (
	"fmt"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/analysis"
	"github.com/glkt/vyb-code/internal/config"
)

// コンテキスト提案システム - Phase 2実装
type ContextSuggestionEngine struct {
	lightProactive *LightweightProactiveManager
	config         *config.Config
	enabled        bool
}

// 基本コンテキスト提案
type ContextSuggestion struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "project", "workflow", "best_practice", "quick_action"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Action      string    `json:"action,omitempty"` // 実行可能なアクション
	Relevance   float64   `json:"relevance"`        // 0.0-1.0
	CreatedAt   time.Time `json:"created_at"`
}

// 新しいコンテキスト提案エンジンを作成
func NewContextSuggestionEngine(cfg *config.Config) *ContextSuggestionEngine {
	if !cfg.IsProactiveEnabled() {
		return &ContextSuggestionEngine{enabled: false}
	}

	return &ContextSuggestionEngine{
		lightProactive: NewLightweightProactiveManager(cfg),
		config:         cfg,
		enabled:        true,
	}
}

// プロジェクト開始時のコンテキスト提案を生成
func (cse *ContextSuggestionEngine) GenerateStartupSuggestions(projectPath string) []ContextSuggestion {
	if !cse.enabled {
		return []ContextSuggestion{}
	}

	suggestions := make([]ContextSuggestion, 0)

	// プロジェクト分析を実行
	analysis, err := cse.lightProactive.AnalyzeProjectLightly(projectPath)
	if err != nil {
		// エラー時は汎用提案のみ
		return cse.generateGenericSuggestions()
	}

	// 言語固有の提案
	if analysis.Language != "" && analysis.Language != "Unknown" {
		suggestions = append(suggestions, cse.generateLanguageSpecificSuggestions(analysis.Language)...)
	}

	// プロジェクト構造に基づく提案
	if analysis.FileStructure != nil {
		suggestions = append(suggestions, cse.generateStructureSuggestions(analysis.FileStructure)...)
	}

	// 技術スタックに基づく提案
	if len(analysis.TechStack) > 0 {
		suggestions = append(suggestions, cse.generateTechStackSuggestions(analysis.TechStack)...)
	}

	// Git関連の提案
	if analysis.GitInfo != nil {
		suggestions = append(suggestions, cse.generateGitSuggestions(analysis.GitInfo)...)
	}

	// 提案を関連度でソートし、上位3つに制限
	if len(suggestions) > 3 {
		suggestions = cse.sortByRelevance(suggestions)[:3]
	}

	return suggestions
}

// 言語固有の提案生成
func (cse *ContextSuggestionEngine) generateLanguageSpecificSuggestions(language string) []ContextSuggestion {
	suggestions := make([]ContextSuggestion, 0)
	timestamp := time.Now()

	switch language {
	case "Go":
		// 実際のプロジェクトの状態に基づく動的提案
		suggestions = append(suggestions, ContextSuggestion{
			ID:          "go-project-status",
			Type:        "quick_action",
			Title:       "プロジェクト状態確認",
			Description: "変更ファイル分析とテスト実行",
			Action:      "git status && go test ./...",
			Relevance:   0.9,
			CreatedAt:   timestamp,
		})

		// より具体的で実行可能な提案
		suggestions = append(suggestions, ContextSuggestion{
			ID:          "go-quality-check",
			Type:        "workflow",
			Title:       "コード品質チェック",
			Description: "フォーマット・lint・vet の実行",
			Action:      "go fmt ./... && go vet ./...",
			Relevance:   0.8,
			CreatedAt:   timestamp,
		})

	case "JavaScript", "TypeScript":
		suggestions = append(suggestions, ContextSuggestion{
			ID:          "js-modern-features",
			Type:        "best_practice",
			Title:       "モダンJavaScript/TypeScript",
			Description: "ES6+の機能活用とTypeScriptでの型安全性の向上",
			Relevance:   0.9,
			CreatedAt:   timestamp,
		})
		suggestions = append(suggestions, ContextSuggestion{
			ID:          "js-performance",
			Type:        "quick_action",
			Title:       "パフォーマンス最適化",
			Description: "バンドルサイズの削減とレンダリング最適化のテクニック",
			Relevance:   0.7,
			CreatedAt:   timestamp,
		})

	case "Python":
		suggestions = append(suggestions, ContextSuggestion{
			ID:          "python-env",
			Type:        "workflow",
			Title:       "Python環境管理",
			Description: "仮想環境、依存関係管理、パッケージングのベストプラクティス",
			Action:      "python -m venv venv && source venv/bin/activate",
			Relevance:   0.8,
			CreatedAt:   timestamp,
		})

	case "Rust":
		suggestions = append(suggestions, ContextSuggestion{
			ID:          "rust-ownership",
			Type:        "best_practice",
			Title:       "Rust所有権システム",
			Description: "所有権、借用、ライフタイムの理解とメモリ安全なコードの書き方",
			Relevance:   0.9,
			CreatedAt:   timestamp,
		})
	}

	return suggestions
}

// プロジェクト構造に基づく提案生成
func (cse *ContextSuggestionEngine) generateStructureSuggestions(structure *analysis.FileStructure) []ContextSuggestion {
	suggestions := make([]ContextSuggestion, 0)
	timestamp := time.Now()

	// 大規模プロジェクトの場合
	if structure.TotalFiles > 100 {
		suggestions = append(suggestions, ContextSuggestion{
			ID:          "large-project-organization",
			Type:        "project",
			Title:       "大規模プロジェクト管理",
			Description: "ファイル整理、モジュール分割、依存関係の管理方法",
			Relevance:   0.8,
			CreatedAt:   timestamp,
		})
	}

	// 多言語プロジェクトの場合
	if len(structure.Languages) > 3 {
		suggestions = append(suggestions, ContextSuggestion{
			ID:          "multi-lang-project",
			Type:        "project",
			Title:       "多言語プロジェクト",
			Description: "複数の言語を使ったプロジェクトでの統一的な開発環境の構築",
			Relevance:   0.7,
			CreatedAt:   timestamp,
		})
	}

	// テストファイルが少ない場合
	testFileCount := 0
	for ext := range structure.Languages {
		if strings.Contains(ext, "test") || strings.Contains(ext, "spec") {
			testFileCount++
		}
	}
	if testFileCount < structure.TotalFiles/10 { // 10%未満の場合
		suggestions = append(suggestions, ContextSuggestion{
			ID:          "improve-test-coverage",
			Type:        "workflow",
			Title:       "テストカバレッジ向上",
			Description: "プロジェクトの品質向上のためのテスト戦略と自動化",
			Relevance:   0.6,
			CreatedAt:   timestamp,
		})
	}

	return suggestions
}

// 技術スタックに基づく提案生成
func (cse *ContextSuggestionEngine) generateTechStackSuggestions(techStack []analysis.Technology) []ContextSuggestion {
	suggestions := make([]ContextSuggestion, 0)
	timestamp := time.Now()

	for _, tech := range techStack {
		if tech.Usage != "primary" {
			continue
		}

		switch tech.Name {
		case "Docker":
			suggestions = append(suggestions, ContextSuggestion{
				ID:          "docker-optimization",
				Type:        "quick_action",
				Title:       "Dockerコンテナ最適化",
				Description: "イメージサイズ削減、マルチステージビルド、セキュリティ強化",
				Action:      "docker system prune -af",
				Relevance:   0.7,
				CreatedAt:   timestamp,
			})

		case "Node.js":
			suggestions = append(suggestions, ContextSuggestion{
				ID:          "nodejs-performance",
				Type:        "best_practice",
				Title:       "Node.js パフォーマンス",
				Description: "メモリ使用量最適化、非同期処理のベストプラクティス",
				Relevance:   0.8,
				CreatedAt:   timestamp,
			})

		case "Go Modules":
			suggestions = append(suggestions, ContextSuggestion{
				ID:          "go-modules-management",
				Type:        "workflow",
				Title:       "Go Modules管理",
				Description: "依存関係の整理、バージョン管理、セキュリティアップデート",
				Action:      "go mod tidy && go mod verify",
				Relevance:   0.8,
				CreatedAt:   timestamp,
			})
		}
	}

	return suggestions
}

// Git関連の提案生成
func (cse *ContextSuggestionEngine) generateGitSuggestions(gitInfo *analysis.GitInfo) []ContextSuggestion {
	suggestions := make([]ContextSuggestion, 0)
	timestamp := time.Now()

	// Git リポジトリがある場合の一般的な提案
	if gitInfo.Repository != "" {
		suggestions = append(suggestions, ContextSuggestion{
			ID:          "git-workflow",
			Type:        "workflow",
			Title:       "Git ワークフロー改善",
			Description: "ブランチ戦略、コミットメッセージ、コードレビューのベストプラクティス",
			Action:      "git status",
			Relevance:   0.6,
			CreatedAt:   timestamp,
		})
	}

	return suggestions
}

// 汎用提案生成（分析失敗時のフォールバック）
func (cse *ContextSuggestionEngine) generateGenericSuggestions() []ContextSuggestion {
	timestamp := time.Now()
	return []ContextSuggestion{
		{
			ID:          "development-setup",
			Type:        "workflow",
			Title:       "開発環境セットアップ",
			Description: "効率的な開発のための環境構築とツール設定",
			Relevance:   0.5,
			CreatedAt:   timestamp,
		},
		{
			ID:          "code-quality",
			Type:        "best_practice",
			Title:       "コード品質向上",
			Description: "リファクタリング、コードレビュー、静的解析の活用方法",
			Relevance:   0.6,
			CreatedAt:   timestamp,
		},
		{
			ID:          "documentation",
			Type:        "project",
			Title:       "ドキュメント作成",
			Description: "保守しやすいドキュメントの書き方と自動生成の活用",
			Relevance:   0.4,
			CreatedAt:   timestamp,
		},
	}
}

// 関連度でソート
func (cse *ContextSuggestionEngine) sortByRelevance(suggestions []ContextSuggestion) []ContextSuggestion {
	// 簡単なバブルソート（提案数が少ないため）
	n := len(suggestions)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if suggestions[j].Relevance < suggestions[j+1].Relevance {
				suggestions[j], suggestions[j+1] = suggestions[j+1], suggestions[j]
			}
		}
	}
	return suggestions
}

// 提案を表示用文字列にフォーマット
func (cse *ContextSuggestionEngine) FormatSuggestions(suggestions []ContextSuggestion) string {
	if len(suggestions) == 0 {
		return ""
	}

	// デバッグ/開発用：この機能を一時的に無効化
	// TODO: 将来的により有用な動的コンテキスト分析を実装する予定
	return "" // 現在は表示を抑制

	// 表示頻度を制限（実際に役立つ時だけ表示）
	relevantSuggestions := make([]ContextSuggestion, 0)
	for _, suggestion := range suggestions {
		if suggestion.Relevance > 0.7 && suggestion.Type == "quick_action" {
			relevantSuggestions = append(relevantSuggestions, suggestion)
		}
	}

	if len(relevantSuggestions) == 0 {
		return "" // 関連性が低い場合は表示しない
	}

	var builder strings.Builder
	builder.WriteString("🚀 推奨アクション:\n")

	for i, suggestion := range relevantSuggestions {
		icon := "📋"
		switch suggestion.Type {
		case "workflow":
			icon = "⚡"
		case "best_practice":
			icon = "⭐"
		case "quick_action":
			icon = "🚀"
		case "project":
			icon = "📁"
		}

		builder.WriteString(fmt.Sprintf("   %s %s\n", icon, suggestion.Title))
		if i < len(relevantSuggestions)-1 {
			builder.WriteString("     " + strings.ReplaceAll(suggestion.Description, "\n", "\n     ") + "\n")
		}
	}

	return builder.String()
}

// リソースクリーンアップ
func (cse *ContextSuggestionEngine) Close() error {
	if cse.lightProactive != nil {
		return cse.lightProactive.Close()
	}
	return nil
}
