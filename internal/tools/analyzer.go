package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glkt/vyb-code/internal/security"
)

// プロジェクト分析結果を格納する構造体
type ProjectAnalysis struct {
	TotalFiles       int                 `json:"total_files"`
	FilesByLanguage  map[string]int      `json:"files_by_language"`
	ProjectStructure map[string][]string `json:"project_structure"`
	Dependencies     []string            `json:"dependencies"`
	GitInfo          *GitProjectInfo     `json:"git_info"`
}

// Git情報を格納する構造体
type GitProjectInfo struct {
	CurrentBranch string   `json:"current_branch"`
	Branches      []string `json:"branches"`
	RecentCommits []string `json:"recent_commits"`
	Status        string   `json:"status"`
}

// ProjectAnalyzer - 廃止予定: AI分析機能を使用してください
// Deprecated: Use AI-powered code analysis from internal/ai package instead
type ProjectAnalyzer struct {
	fileOps     *UnifiedFileOperations // ファイル操作（統一版）
	gitOps      *GitOperations         // Git操作
	constraints *security.Constraints  // セキュリティ制約
	projectDir  string                 // プロジェクトディレクトリ
}

// プロジェクト解析ハンドラーを作成するコンストラクタ
// Deprecated: Use AI-powered code analysis from internal/ai package instead
func NewProjectAnalyzer(constraints *security.Constraints, projectDir string) *ProjectAnalyzer {
	return &ProjectAnalyzer{
		fileOps:     NewUnifiedFileOperations(int64(constraints.MaxTimeout)*1024*1024, projectDir), // 統一版を利用
		gitOps:      NewGitOperations(constraints, projectDir),
		constraints: constraints,
		projectDir:  projectDir,
	}
}

// プロジェクト全体を分析
func (p *ProjectAnalyzer) AnalyzeProject() (*ProjectAnalysis, error) {
	analysis := &ProjectAnalysis{
		FilesByLanguage:  make(map[string]int),
		ProjectStructure: make(map[string][]string),
		Dependencies:     []string{},
	}

	// ファイル分析
	if err := p.analyzeFiles(analysis); err != nil {
		return nil, fmt.Errorf("ファイル分析エラー: %w", err)
	}

	// Git情報分析
	gitInfo, err := p.analyzeGitInfo()
	if err != nil {
		// Gitリポジトリでない場合はスキップ
		fmt.Printf("Git情報取得をスキップ: %v\n", err)
	} else {
		analysis.GitInfo = gitInfo
	}

	// 依存関係分析
	if err := p.analyzeDependencies(analysis); err != nil {
		fmt.Printf("依存関係分析エラー: %v\n", err)
	}

	return analysis, nil
}

// ファイル構造とプログラミング言語を分析
func (p *ProjectAnalyzer) analyzeFiles(analysis *ProjectAnalysis) error {
	return filepath.Walk(p.projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // エラーファイルはスキップ
		}

		// ディレクトリ処理
		if info.IsDir() {
			// 隠しディレクトリや一般的な無視ディレクトリをスキップ
			dirname := filepath.Base(path)
			if strings.HasPrefix(dirname, ".") ||
				dirname == "node_modules" ||
				dirname == "vendor" ||
				dirname == "target" ||
				dirname == "__pycache__" {
				return filepath.SkipDir
			}

			// ディレクトリ構造を記録
			relPath, _ := filepath.Rel(p.projectDir, path)
			if relPath != "." {
				parentDir := filepath.Dir(relPath)
				if parentDir == "." {
					parentDir = "root"
				}
				analysis.ProjectStructure[parentDir] = append(analysis.ProjectStructure[parentDir], dirname)
			}
			return nil
		}

		// ファイル処理
		analysis.TotalFiles++

		// プログラミング言語を判定
		ext := strings.ToLower(filepath.Ext(path))
		language := p.getLanguageFromExtension(ext)
		if language != "" {
			analysis.FilesByLanguage[language]++
		}

		return nil
	})
}

// Git情報を分析
func (p *ProjectAnalyzer) analyzeGitInfo() (*GitProjectInfo, error) {
	gitInfo := &GitProjectInfo{}

	// 現在のブランチを取得
	currentBranch, err := p.gitOps.GetCurrentBranch()
	if err != nil {
		return nil, err
	}
	gitInfo.CurrentBranch = currentBranch

	// ブランチ一覧を取得
	branchResult, err := p.gitOps.GetBranches()
	if err != nil {
		return nil, err
	}

	if branchResult.ExitCode == 0 {
		branches := strings.Split(strings.TrimSpace(branchResult.Stdout), "\n")
		for _, branch := range branches {
			branch = strings.TrimSpace(branch)
			if branch != "" && !strings.HasPrefix(branch, "remotes/") {
				branch = strings.TrimPrefix(branch, "* ")
				gitInfo.Branches = append(gitInfo.Branches, branch)
			}
		}
	}

	// 最近のコミットを取得
	logResult, err := p.gitOps.GetLog("-5")
	if err == nil && logResult.ExitCode == 0 {
		commits := strings.Split(strings.TrimSpace(logResult.Stdout), "\n")
		gitInfo.RecentCommits = commits
	}

	// Git状態を取得
	statusResult, err := p.gitOps.GetStatus()
	if err == nil && statusResult.ExitCode == 0 {
		if statusResult.Stdout == "" {
			gitInfo.Status = "clean"
		} else {
			gitInfo.Status = "dirty"
		}
	}

	return gitInfo, nil
}

// 依存関係を分析
func (p *ProjectAnalyzer) analyzeDependencies(analysis *ProjectAnalysis) error {
	// Go mod ファイルをチェック
	goModPath := filepath.Join(p.projectDir, "go.mod")
	if content, err := p.fileOps.ReadFile(goModPath); err == nil {
		deps := p.parseGoModDependencies(content)
		analysis.Dependencies = append(analysis.Dependencies, deps...)
	}

	// package.json ファイルをチェック
	packageJsonPath := filepath.Join(p.projectDir, "package.json")
	if content, err := p.fileOps.ReadFile(packageJsonPath); err == nil {
		// 簡易的なpackage.json解析（実際のJSONパーサーを使用することを推奨）
		if strings.Contains(content, "dependencies") {
			analysis.Dependencies = append(analysis.Dependencies, "npm/node dependencies found")
		}
	}

	// requirements.txt ファイルをチェック
	requirementsPath := filepath.Join(p.projectDir, "requirements.txt")
	if content, err := p.fileOps.ReadFile(requirementsPath); err == nil {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				analysis.Dependencies = append(analysis.Dependencies, "python: "+line)
			}
		}
	}

	return nil
}

// ファイル拡張子からプログラミング言語を判定
func (p *ProjectAnalyzer) getLanguageFromExtension(ext string) string {
	languageMap := map[string]string{
		".go":    "Go",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".jsx":   "React JSX",
		".tsx":   "React TSX",
		".py":    "Python",
		".java":  "Java",
		".c":     "C",
		".cpp":   "C++",
		".cc":    "C++",
		".cxx":   "C++",
		".c++":   "C++",
		".h":     "C/C++ Header",
		".hpp":   "C++ Header",
		".hh":    "C++ Header",
		".hxx":   "C++ Header",
		".h++":   "C++ Header",
		".rs":    "Rust",
		".php":   "PHP",
		".rb":    "Ruby",
		".kt":    "Kotlin",
		".swift": "Swift",
		".sh":    "Shell",
		".bash":  "Bash",
		".zsh":   "Zsh",
		".fish":  "Fish",
		".ps1":   "PowerShell",
		".md":    "Markdown",
		".yml":   "YAML",
		".yaml":  "YAML",
		".json":  "JSON",
		".xml":   "XML",
		".sql":   "SQL",
		".scala": "Scala",
		".clj":   "Clojure",
		".hs":    "Haskell",
		".ml":    "OCaml",
		".fs":    "F#",
		".dart":  "Dart",
		".r":     "R",
		".m":     "Objective-C/MATLAB",
		".cs":    "C#",
		".vb":    "Visual Basic",
		".pl":    "Perl",
		".lua":   "Lua",
		".elm":   "Elm",
		".ex":    "Elixir",
		".exs":   "Elixir Script",
		".nim":   "Nim",
		".jl":    "Julia",
		".zig":   "Zig",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}
	return ""
}

// go.modファイルから依存関係を解析
func (p *ProjectAnalyzer) parseGoModDependencies(content string) []string {
	var deps []string
	lines := strings.Split(content, "\n")

	inRequireBlock := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "require ") {
			if strings.Contains(line, "(") {
				inRequireBlock = true
				continue
			} else {
				// 単行のrequire文
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					deps = append(deps, "go: "+parts[1])
				}
			}
		} else if inRequireBlock {
			if line == ")" {
				inRequireBlock = false
			} else if line != "" && !strings.HasPrefix(line, "//") {
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					deps = append(deps, "go: "+parts[0])
				}
			}
		}
	}

	return deps
}
