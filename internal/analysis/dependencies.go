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

// 依存関係の分析実装

// 依存関係を分析
func (pa *projectAnalyzer) AnalyzeDependencies(projectPath string) ([]Dependency, error) {
	dependencies := make([]Dependency, 0)

	// 各言語/エコシステムの依存関係ファイルをチェック
	analyzers := []func(string) ([]Dependency, error){
		pa.analyzePackageJson,     // Node.js/npm
		pa.analyzeGoMod,           // Go modules
		pa.analyzeRequirementsTxt, // Python
		pa.analyzeCargoToml,       // Rust
		pa.analyzePomXml,          // Java/Maven
		pa.analyzeBuildGradle,     // Java/Gradle
		pa.analyzeComposerJson,    // PHP
		pa.analyzeGemfile,         // Ruby
		pa.analyzePubspecYaml,     // Dart/Flutter
	}

	for _, analyzer := range analyzers {
		deps, err := analyzer(projectPath)
		if err == nil && len(deps) > 0 {
			dependencies = append(dependencies, deps...)
		}
	}

	// 重複を除去
	dependencies = pa.deduplicateDependencies(dependencies)

	// 脆弱性情報を追加（簡易版）
	for i := range dependencies {
		dependencies[i].Vulnerabilities = pa.checkVulnerabilities(dependencies[i])
		dependencies[i].Outdated = pa.checkOutdated(dependencies[i])
	}

	return dependencies, nil
}

// package.json の分析
func (pa *projectAnalyzer) analyzePackageJson(projectPath string) ([]Dependency, error) {
	packageJsonPath := filepath.Join(projectPath, "package.json")

	if _, err := os.Stat(packageJsonPath); os.IsNotExist(err) {
		return nil, err
	}

	content, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return nil, err
	}

	var packageData map[string]interface{}
	if err := json.Unmarshal(content, &packageData); err != nil {
		return nil, err
	}

	dependencies := make([]Dependency, 0)

	// dependencies
	if deps, exists := packageData["dependencies"].(map[string]interface{}); exists {
		for name, version := range deps {
			if versionStr, ok := version.(string); ok {
				dependencies = append(dependencies, Dependency{
					Name:    name,
					Version: versionStr,
					Type:    "direct",
					Source:  "package.json",
				})
			}
		}
	}

	// devDependencies
	if devDeps, exists := packageData["devDependencies"].(map[string]interface{}); exists {
		for name, version := range devDeps {
			if versionStr, ok := version.(string); ok {
				dependencies = append(dependencies, Dependency{
					Name:    name,
					Version: versionStr,
					Type:    "dev",
					Source:  "package.json",
				})
			}
		}
	}

	// peerDependencies
	if peerDeps, exists := packageData["peerDependencies"].(map[string]interface{}); exists {
		for name, version := range peerDeps {
			if versionStr, ok := version.(string); ok {
				dependencies = append(dependencies, Dependency{
					Name:    name,
					Version: versionStr,
					Type:    "peer",
					Source:  "package.json",
				})
			}
		}
	}

	// optionalDependencies
	if optionalDeps, exists := packageData["optionalDependencies"].(map[string]interface{}); exists {
		for name, version := range optionalDeps {
			if versionStr, ok := version.(string); ok {
				dependencies = append(dependencies, Dependency{
					Name:    name,
					Version: versionStr,
					Type:    "optional",
					Source:  "package.json",
				})
			}
		}
	}

	return dependencies, nil
}

// go.mod の分析
func (pa *projectAnalyzer) analyzeGoMod(projectPath string) ([]Dependency, error) {
	goModPath := filepath.Join(projectPath, "go.mod")

	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return nil, err
	}

	file, err := os.Open(goModPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	dependencies := make([]Dependency, 0)
	scanner := bufio.NewScanner(file)

	// go.modの構文解析
	inRequireBlock := false
	requireRegex := regexp.MustCompile(`^\s*require\s+(.*)`)
	blockStartRegex := regexp.MustCompile(`^\s*require\s*\(\s*$`)
	blockEndRegex := regexp.MustCompile(`^\s*\)\s*$`)
	dependencyRegex := regexp.MustCompile(`^\s*([^\s]+)\s+([^\s]+)(?:\s+//\s*(.*))?`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// コメント行をスキップ
		if strings.HasPrefix(line, "//") || line == "" {
			continue
		}

		// require ブロックの開始
		if blockStartRegex.MatchString(line) {
			inRequireBlock = true
			continue
		}

		// require ブロックの終了
		if inRequireBlock && blockEndRegex.MatchString(line) {
			inRequireBlock = false
			continue
		}

		// 単一行のrequire
		if matches := requireRegex.FindStringSubmatch(line); matches != nil {
			depLine := strings.TrimSpace(matches[1])
			if depMatches := dependencyRegex.FindStringSubmatch(depLine); depMatches != nil {
				dependencies = append(dependencies, Dependency{
					Name:    depMatches[1],
					Version: depMatches[2],
					Type:    "direct",
					Source:  "go.mod",
				})
			}
		}

		// require ブロック内の依存関係
		if inRequireBlock {
			if matches := dependencyRegex.FindStringSubmatch(line); matches != nil {
				depType := "direct"
				if len(matches) > 3 && strings.Contains(matches[3], "indirect") {
					depType = "indirect"
				}

				dependencies = append(dependencies, Dependency{
					Name:    matches[1],
					Version: matches[2],
					Type:    depType,
					Source:  "go.mod",
				})
			}
		}
	}

	return dependencies, scanner.Err()
}

// requirements.txt の分析
func (pa *projectAnalyzer) analyzeRequirementsTxt(projectPath string) ([]Dependency, error) {
	requirementsPath := filepath.Join(projectPath, "requirements.txt")

	if _, err := os.Stat(requirementsPath); os.IsNotExist(err) {
		return nil, err
	}

	file, err := os.Open(requirementsPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	dependencies := make([]Dependency, 0)
	scanner := bufio.NewScanner(file)

	// requirements.txt の構文解析
	dependencyRegex := regexp.MustCompile(`^([a-zA-Z0-9._-]+)([>=<~!]+)([a-zA-Z0-9._-]+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// コメント行や空行をスキップ
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		if matches := dependencyRegex.FindStringSubmatch(line); matches != nil {
			dependencies = append(dependencies, Dependency{
				Name:    matches[1],
				Version: matches[3],
				Type:    "direct",
				Source:  "requirements.txt",
			})
		} else {
			// バージョン指定なしの場合
			if name := strings.TrimSpace(line); name != "" {
				dependencies = append(dependencies, Dependency{
					Name:    name,
					Version: "latest",
					Type:    "direct",
					Source:  "requirements.txt",
				})
			}
		}
	}

	return dependencies, scanner.Err()
}

// Cargo.toml の分析
func (pa *projectAnalyzer) analyzeCargoToml(projectPath string) ([]Dependency, error) {
	cargoTomlPath := filepath.Join(projectPath, "Cargo.toml")

	if _, err := os.Stat(cargoTomlPath); os.IsNotExist(err) {
		return nil, err
	}

	content, err := os.ReadFile(cargoTomlPath)
	if err != nil {
		return nil, err
	}

	dependencies := make([]Dependency, 0)
	lines := strings.Split(string(content), "\n")

	currentSection := ""
	dependencyRegex := regexp.MustCompile(`^([a-zA-Z0-9._-]+)\s*=\s*"([^"]+)"`)
	sectionRegex := regexp.MustCompile(`^\[([^\]]+)\]`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// コメント行をスキップ
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// セクションの検出
		if matches := sectionRegex.FindStringSubmatch(line); matches != nil {
			currentSection = matches[1]
			continue
		}

		// 依存関係の検出
		if strings.Contains(currentSection, "dependencies") {
			if matches := dependencyRegex.FindStringSubmatch(line); matches != nil {
				depType := "direct"
				if strings.Contains(currentSection, "dev-dependencies") {
					depType = "dev"
				} else if strings.Contains(currentSection, "build-dependencies") {
					depType = "build"
				}

				dependencies = append(dependencies, Dependency{
					Name:    matches[1],
					Version: matches[2],
					Type:    depType,
					Source:  "Cargo.toml",
				})
			}
		}
	}

	return dependencies, nil
}

// pom.xml の分析（簡易版）
func (pa *projectAnalyzer) analyzePomXml(projectPath string) ([]Dependency, error) {
	pomXmlPath := filepath.Join(projectPath, "pom.xml")

	if _, err := os.Stat(pomXmlPath); os.IsNotExist(err) {
		return nil, err
	}

	content, err := os.ReadFile(pomXmlPath)
	if err != nil {
		return nil, err
	}

	dependencies := make([]Dependency, 0)
	contentStr := string(content)

	// 簡易XML解析（実際の実装ではより堅牢なXMLパーサーを使用）
	dependencyRegex := regexp.MustCompile(`<dependency>[\s\S]*?<groupId>([^<]+)</groupId>[\s\S]*?<artifactId>([^<]+)</artifactId>[\s\S]*?<version>([^<]+)</version>[\s\S]*?</dependency>`)
	matches := dependencyRegex.FindAllStringSubmatch(contentStr, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			name := fmt.Sprintf("%s:%s", match[1], match[2])
			dependencies = append(dependencies, Dependency{
				Name:    name,
				Version: match[3],
				Type:    "direct",
				Source:  "pom.xml",
			})
		}
	}

	return dependencies, nil
}

// build.gradle の分析（簡易版）
func (pa *projectAnalyzer) analyzeBuildGradle(projectPath string) ([]Dependency, error) {
	buildGradlePath := filepath.Join(projectPath, "build.gradle")

	if _, err := os.Stat(buildGradlePath); os.IsNotExist(err) {
		return nil, err
	}

	content, err := os.ReadFile(buildGradlePath)
	if err != nil {
		return nil, err
	}

	dependencies := make([]Dependency, 0)
	lines := strings.Split(string(content), "\n")

	// Gradleの依存関係構文を解析
	dependencyRegex := regexp.MustCompile(`(implementation|compile|testImplementation|testCompile|api)\s+['"]([^:]+):([^:]+):([^'"]+)['"]`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if matches := dependencyRegex.FindStringSubmatch(line); matches != nil {
			depType := "direct"
			if strings.Contains(matches[1], "test") {
				depType = "test"
			}

			name := fmt.Sprintf("%s:%s", matches[2], matches[3])
			dependencies = append(dependencies, Dependency{
				Name:    name,
				Version: matches[4],
				Type:    depType,
				Source:  "build.gradle",
			})
		}
	}

	return dependencies, nil
}

// composer.json の分析
func (pa *projectAnalyzer) analyzeComposerJson(projectPath string) ([]Dependency, error) {
	composerJsonPath := filepath.Join(projectPath, "composer.json")

	if _, err := os.Stat(composerJsonPath); os.IsNotExist(err) {
		return nil, err
	}

	content, err := os.ReadFile(composerJsonPath)
	if err != nil {
		return nil, err
	}

	var composerData map[string]interface{}
	if err := json.Unmarshal(content, &composerData); err != nil {
		return nil, err
	}

	dependencies := make([]Dependency, 0)

	// require
	if require, exists := composerData["require"].(map[string]interface{}); exists {
		for name, version := range require {
			if versionStr, ok := version.(string); ok {
				dependencies = append(dependencies, Dependency{
					Name:    name,
					Version: versionStr,
					Type:    "direct",
					Source:  "composer.json",
				})
			}
		}
	}

	// require-dev
	if requireDev, exists := composerData["require-dev"].(map[string]interface{}); exists {
		for name, version := range requireDev {
			if versionStr, ok := version.(string); ok {
				dependencies = append(dependencies, Dependency{
					Name:    name,
					Version: versionStr,
					Type:    "dev",
					Source:  "composer.json",
				})
			}
		}
	}

	return dependencies, nil
}

// Gemfile の分析
func (pa *projectAnalyzer) analyzeGemfile(projectPath string) ([]Dependency, error) {
	gemfilePath := filepath.Join(projectPath, "Gemfile")

	if _, err := os.Stat(gemfilePath); os.IsNotExist(err) {
		return nil, err
	}

	file, err := os.Open(gemfilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	dependencies := make([]Dependency, 0)
	scanner := bufio.NewScanner(file)

	// Gemfileの構文解析
	gemRegex := regexp.MustCompile(`gem\s+['"]([^'"]+)['"](?:\s*,\s*['"]([^'"]+)['"])?`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// コメント行をスキップ
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		if matches := gemRegex.FindStringSubmatch(line); matches != nil {
			version := "latest"
			if len(matches) > 2 && matches[2] != "" {
				version = matches[2]
			}

			dependencies = append(dependencies, Dependency{
				Name:    matches[1],
				Version: version,
				Type:    "direct",
				Source:  "Gemfile",
			})
		}
	}

	return dependencies, scanner.Err()
}

// pubspec.yaml の分析
func (pa *projectAnalyzer) analyzePubspecYaml(projectPath string) ([]Dependency, error) {
	pubspecPath := filepath.Join(projectPath, "pubspec.yaml")

	if _, err := os.Stat(pubspecPath); os.IsNotExist(err) {
		return nil, err
	}

	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		return nil, err
	}

	dependencies := make([]Dependency, 0)
	lines := strings.Split(string(content), "\n")

	currentSection := ""
	dependencyRegex := regexp.MustCompile(`^\s+([a-zA-Z0-9._-]+):\s*(.*)`)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// コメント行をスキップ
		if strings.HasPrefix(trimmedLine, "#") || trimmedLine == "" {
			continue
		}

		// セクションの検出
		if strings.HasSuffix(trimmedLine, ":") && !strings.HasPrefix(line, " ") {
			currentSection = strings.TrimSuffix(trimmedLine, ":")
			continue
		}

		// 依存関係の検出
		if currentSection == "dependencies" || currentSection == "dev_dependencies" {
			if matches := dependencyRegex.FindStringSubmatch(line); matches != nil {
				depType := "direct"
				if currentSection == "dev_dependencies" {
					depType = "dev"
				}

				version := strings.TrimSpace(matches[2])
				if version == "" {
					version = "latest"
				}

				dependencies = append(dependencies, Dependency{
					Name:    matches[1],
					Version: version,
					Type:    depType,
					Source:  "pubspec.yaml",
				})
			}
		}
	}

	return dependencies, nil
}

// 依存関係の重複を除去
func (pa *projectAnalyzer) deduplicateDependencies(dependencies []Dependency) []Dependency {
	seen := make(map[string]bool)
	result := make([]Dependency, 0)

	for _, dep := range dependencies {
		key := fmt.Sprintf("%s:%s:%s", dep.Name, dep.Version, dep.Source)
		if !seen[key] {
			seen[key] = true
			result = append(result, dep)
		}
	}

	return result
}

// 脆弱性チェック（簡易版）
func (pa *projectAnalyzer) checkVulnerabilities(dep Dependency) []string {
	// 実際の実装では、セキュリティデータベースとの連携を行う
	// 今回は既知の脆弱性のあるパッケージの例を返す
	vulnerablePackages := map[string][]string{
		"lodash": {"CVE-2019-10744", "CVE-2020-8203"},
		"jquery": {"CVE-2020-11022", "CVE-2020-11023"},
		"moment": {"CVE-2022-31129"},
	}

	if vulns, exists := vulnerablePackages[dep.Name]; exists {
		return vulns
	}

	return []string{}
}

// 古いバージョンチェック（簡易版）
func (pa *projectAnalyzer) checkOutdated(dep Dependency) bool {
	// 実際の実装では、パッケージレジストリから最新バージョンを取得して比較
	// 今回は簡易的な判定を行う

	// セマンティックバージョンの古いパターンを検出
	if strings.HasPrefix(dep.Version, "^") || strings.HasPrefix(dep.Version, "~") {
		return false // 自動更新される範囲内
	}

	// 固定バージョンで古そうなパターン
	oldPatterns := []string{
		"1.0", "1.1", "1.2", "1.3", "1.4", // 非常に古いバージョン
		"0.1", "0.2", "0.3", "0.4", "0.5", // 初期バージョン
	}

	for _, pattern := range oldPatterns {
		if strings.HasPrefix(dep.Version, pattern) {
			return true
		}
	}

	return false
}
