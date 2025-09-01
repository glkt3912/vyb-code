package tools

import (
	"os"
	"path/filepath"
	"strings"
)

// 言語固有の設定と操作を定義するインターフェース
type LanguageSupport interface {
	GetName() string
	GetExtensions() []string
	GetBuildCommand() string
	GetTestCommand() string
	GetLintCommand() string
	GetDependencyFile() string
	ParseDependencies(content string) []string
}

// Go言語サポート
type GoLanguageSupport struct{}

func (g *GoLanguageSupport) GetName() string           { return "Go" }
func (g *GoLanguageSupport) GetExtensions() []string   { return []string{".go"} }
func (g *GoLanguageSupport) GetBuildCommand() string   { return "go build" }
func (g *GoLanguageSupport) GetTestCommand() string    { return "go test ./..." }
func (g *GoLanguageSupport) GetLintCommand() string    { return "golangci-lint run" }
func (g *GoLanguageSupport) GetDependencyFile() string { return "go.mod" }

func (g *GoLanguageSupport) ParseDependencies(content string) []string {
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
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					deps = append(deps, parts[1])
				}
			}
		} else if inRequireBlock {
			if line == ")" {
				inRequireBlock = false
			} else if line != "" && !strings.HasPrefix(line, "//") {
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					deps = append(deps, parts[0])
				}
			}
		}
	}

	return deps
}

// JavaScript/Node.js言語サポート
type JavaScriptLanguageSupport struct{}

func (js *JavaScriptLanguageSupport) GetName() string { return "JavaScript/Node.js" }
func (js *JavaScriptLanguageSupport) GetExtensions() []string {
	return []string{".js", ".ts", ".jsx", ".tsx"}
}
func (js *JavaScriptLanguageSupport) GetBuildCommand() string   { return "npm run build" }
func (js *JavaScriptLanguageSupport) GetTestCommand() string    { return "npm test" }
func (js *JavaScriptLanguageSupport) GetLintCommand() string    { return "npm run lint" }
func (js *JavaScriptLanguageSupport) GetDependencyFile() string { return "package.json" }

func (js *JavaScriptLanguageSupport) ParseDependencies(content string) []string {
	var deps []string
	// 簡易的なpackage.json解析（実際のJSONパーサーを使用することを推奨）
	lines := strings.Split(content, "\n")
	inDeps := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "\"dependencies\"") || strings.Contains(line, "\"devDependencies\"") {
			inDeps = true
			continue
		}

		if inDeps && strings.Contains(line, "}") {
			inDeps = false
			continue
		}

		if inDeps && strings.Contains(line, "\"") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				deps = append(deps, parts[1])
			}
		}
	}

	return deps
}

// Python言語サポート
type PythonLanguageSupport struct{}

func (py *PythonLanguageSupport) GetName() string           { return "Python" }
func (py *PythonLanguageSupport) GetExtensions() []string   { return []string{".py"} }
func (py *PythonLanguageSupport) GetBuildCommand() string   { return "python -m py_compile" }
func (py *PythonLanguageSupport) GetTestCommand() string    { return "python -m pytest" }
func (py *PythonLanguageSupport) GetLintCommand() string    { return "flake8" }
func (py *PythonLanguageSupport) GetDependencyFile() string { return "requirements.txt" }

func (py *PythonLanguageSupport) ParseDependencies(content string) []string {
	var deps []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// バージョン指定を除去
			depName := strings.Split(line, "==")[0]
			depName = strings.Split(depName, ">=")[0]
			depName = strings.Split(depName, "<=")[0]
			deps = append(deps, strings.TrimSpace(depName))
		}
	}

	return deps
}

// 多言語サポートマネージャー
type LanguageManager struct {
	languages map[string]LanguageSupport
}

// 多言語サポートマネージャーを作成
func NewLanguageManager() *LanguageManager {
	manager := &LanguageManager{
		languages: make(map[string]LanguageSupport),
	}

	// サポート言語を登録
	manager.RegisterLanguage(&GoLanguageSupport{})
	manager.RegisterLanguage(&JavaScriptLanguageSupport{})
	manager.RegisterLanguage(&PythonLanguageSupport{})

	return manager
}

// 言語サポートを登録
func (lm *LanguageManager) RegisterLanguage(lang LanguageSupport) {
	lm.languages[lang.GetName()] = lang
}

// ファイル拡張子から言語を検出
func (lm *LanguageManager) DetectLanguage(filePath string) LanguageSupport {
	ext := strings.ToLower(filepath.Ext(filePath))

	for _, lang := range lm.languages {
		for _, supportedExt := range lang.GetExtensions() {
			if ext == supportedExt {
				return lang
			}
		}
	}

	return nil
}

// プロジェクトの主要言語を検出
func (lm *LanguageManager) DetectProjectLanguages(projectDir string) ([]LanguageSupport, error) {
	var detectedLangs []LanguageSupport
	langCounts := make(map[string]int)

	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		lang := lm.DetectLanguage(path)
		if lang != nil {
			langCounts[lang.GetName()]++
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// ファイル数が多い言語を主要言語として選択
	for langName, count := range langCounts {
		if count > 0 { // 少なくとも1つのファイルがある言語
			if lang, exists := lm.languages[langName]; exists {
				detectedLangs = append(detectedLangs, lang)
			}
		}
	}

	return detectedLangs, nil
}

// 指定した言語のビルドコマンドを取得
func (lm *LanguageManager) GetBuildCommands(languages []LanguageSupport) []string {
	var commands []string
	for _, lang := range languages {
		if cmd := lang.GetBuildCommand(); cmd != "" {
			commands = append(commands, cmd)
		}
	}
	return commands
}

// 指定した言語のテストコマンドを取得
func (lm *LanguageManager) GetTestCommands(languages []LanguageSupport) []string {
	var commands []string
	for _, lang := range languages {
		if cmd := lang.GetTestCommand(); cmd != "" {
			commands = append(commands, cmd)
		}
	}
	return commands
}
