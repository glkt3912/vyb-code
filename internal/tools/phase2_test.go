package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glkt/vyb-code/internal/security"
)

// Phase2機能の統合テスト

func TestAdvancedProjectAnalyzer(t *testing.T) {
	// テスト用プロジェクトディレクトリ作成
	tempDir := setupTestProject(t)
	defer os.RemoveAll(tempDir)

	constraints := &security.Constraints{
		MaxTimeout: 30,
	}

	analyzer := NewAdvancedProjectAnalyzer(constraints, tempDir)

	// 高度解析実行
	analysis, err := analyzer.AnalyzeAdvanced()
	if err != nil {
		t.Fatalf("高度解析エラー: %v", err)
	}

	// 基本情報の確認
	if analysis.BasicInfo == nil {
		t.Fatal("基本情報が取得されていません")
	}

	// アーキテクチャ情報の確認
	if analysis.Architecture == nil {
		t.Fatal("アーキテクチャ情報が取得されていません")
	}

	// セキュリティ分析の確認
	if analysis.SecurityAnalysis == nil {
		t.Fatal("セキュリティ分析が実行されていません")
	}

	// 健康度スコアの確認
	if analysis.HealthScore == nil {
		t.Fatal("健康度スコアが計算されていません")
	}

	if analysis.HealthScore.OverallScore < 0 || analysis.HealthScore.OverallScore > 100 {
		t.Errorf("健康度スコアが範囲外: %d", analysis.HealthScore.OverallScore)
	}

	t.Logf("高度解析完了: 健康度スコア=%d, セキュリティスコア=%d",
		analysis.HealthScore.OverallScore, analysis.SecurityAnalysis.SecurityScore)
}

func TestBuildManager(t *testing.T) {
	// テスト用プロジェクトディレクトリ作成
	tempDir := setupTestProject(t)
	defer os.RemoveAll(tempDir)

	// Makefile作成
	makefileContent := `
all: build test

build:
	echo "Building project"

test:
	echo "Running tests"

clean:
	echo "Cleaning up"
`
	makefilePath := filepath.Join(tempDir, "Makefile")
	if err := os.WriteFile(makefilePath, []byte(makefileContent), 0644); err != nil {
		t.Fatalf("Makefile作成エラー: %v", err)
	}

	constraints := &security.Constraints{
		MaxTimeout:      30,
		AllowedCommands: []string{"echo", "make", "go"},
	}

	buildManager := NewBuildManager(constraints, tempDir)

	// 自動ビルド実行
	result, err := buildManager.AutoBuild()
	if err != nil {
		t.Fatalf("自動ビルドエラー: %v", err)
	}

	if !result.Success {
		t.Errorf("ビルドが失敗しました: %s", result.ErrorOutput)
	}

	// ビルドシステムは makefile または go_native のいずれかが選択される
	if result.BuildSystem != "makefile" && result.BuildSystem != "go_native" {
		t.Errorf("期待されるビルドシステム: makefile または go_native, 実際: %s", result.BuildSystem)
	}

	t.Logf("ビルド成功: %s, 実行時間: %v", result.Command, result.Duration)

	// パイプライン実行テスト
	pipeline, err := buildManager.CreatePresetPipeline("go_standard")
	if err != nil {
		t.Fatalf("パイプライン作成エラー: %v", err)
	}

	if pipeline.Name != "Go Standard Pipeline" {
		t.Errorf("期待されるパイプライン名: Go Standard Pipeline, 実際: %s", pipeline.Name)
	}

	if len(pipeline.Steps) == 0 {
		t.Error("パイプラインにステップがありません")
	}
}

func TestLanguageManagerExtended(t *testing.T) {
	manager := NewLanguageManager()

	// 拡張言語サポートテスト
	testCases := []struct {
		filename     string
		expectedLang string
	}{
		{"main.go", "Go"},
		{"app.js", "JavaScript/Node.js"},
		{"component.tsx", "JavaScript/Node.js"},
		{"main.py", "Python"},
		{"main.rs", "Rust"},
		{"Main.java", "Java"},
		{"main.cpp", "C++"},
		{"header.hpp", "C++"},
		{"program.c", "C"},
	}

	for _, tc := range testCases {
		lang := manager.DetectLanguage(tc.filename)
		if lang == nil {
			t.Errorf("言語が検出されませんでした: %s", tc.filename)
			continue
		}

		if lang.GetName() != tc.expectedLang {
			t.Errorf("期待される言語: %s, 実際: %s (ファイル: %s)",
				tc.expectedLang, lang.GetName(), tc.filename)
		}
	}

	// Rustの依存関係解析テスト
	rustLang := &RustLanguageSupport{}
	cargoToml := `
[package]
name = "test-project"
version = "0.1.0"

[dependencies]
serde = "1.0"
tokio = { version = "1.0", features = ["full"] }
clap = "4.0"

[dev-dependencies]
tempfile = "3.0"
`
	deps := rustLang.ParseDependencies(cargoToml)
	expectedDeps := []string{"serde", "tokio", "clap", "tempfile"}

	if len(deps) != len(expectedDeps) {
		t.Errorf("期待される依存関係数: %d, 実際: %d", len(expectedDeps), len(deps))
	}

	for _, expectedDep := range expectedDeps {
		found := false
		for _, dep := range deps {
			if dep == expectedDep {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("依存関係が見つかりません: %s", expectedDep)
		}
	}
}

func TestToolRegistryPhase2Integration(t *testing.T) {
	// テスト用プロジェクトディレクトリ作成
	tempDir := setupTestProject(t)
	defer os.RemoveAll(tempDir)

	constraints := &security.Constraints{
		MaxTimeout:      30,
		AllowedCommands: []string{"echo", "make", "go"},
	}

	registry := NewToolRegistry(constraints, tempDir, 1024*1024, nil)

	// Phase2ツールが登録されているか確認
	allTools := registry.GetAllTools()

	phase2Tools := []string{
		"project_analyze",
		"build",
		"architecture_map",
		"dependency_scan",
	}

	for _, toolName := range phase2Tools {
		if tool, exists := allTools[toolName]; !exists {
			t.Errorf("Phase2ツール '%s' が登録されていません", toolName)
		} else {
			if tool.Type != "native" {
				t.Errorf("ツール '%s' のタイプが不正: %s", toolName, tool.Type)
			}
		}
	}

	// プロジェクト解析ツールの実行テスト
	result, err := registry.ExecuteTool("project_analyze", map[string]interface{}{
		"analysis_type": "basic",
	})

	if err != nil {
		t.Fatalf("project_analyzeツール実行エラー: %v", err)
	}

	if result.IsError {
		t.Errorf("project_analyzeツールがエラーを返しました: %s", result.Content)
	}

	contentPreview := result.Content
	if len(contentPreview) > 100 {
		contentPreview = contentPreview[:100] + "..."
	}
	t.Logf("プロジェクト解析結果: %s", contentPreview)

	// ビルドツールの実行テスト
	buildResult, err := registry.ExecuteTool("build", map[string]interface{}{
		"system": "auto",
		"target": "build",
	})

	if err != nil {
		// ビルドツールが見つからない場合はスキップ
		t.Logf("ビルドツール実行をスキップ: %v", err)
	} else if !buildResult.IsError {
		t.Logf("ビルドツール実行成功")
	}
}

func TestSecurityAnalysis(t *testing.T) {
	// テスト用プロジェクトディレクトリ作成
	tempDir := setupTestProject(t)
	defer os.RemoveAll(tempDir)

	// 機密情報を含むファイル作成
	secretFile := filepath.Join(tempDir, "config.go")
	secretContent := `
package main

var apiKey = "sk-abc123def456ghi789"
var password = "super_secret_password"
var token = "ghp_1234567890abcdefghijk"

func main() {
    // Some code here
}
`
	if err := os.WriteFile(secretFile, []byte(secretContent), 0644); err != nil {
		t.Fatalf("シークレットファイル作成エラー: %v", err)
	}

	// 不安全なパターンを含むファイル作成
	insecureFile := filepath.Join(tempDir, "database.go")
	insecureContent := `
package main

import "database/sql"

func getUserData(userID string) {
    query := "SELECT * FROM users WHERE id = " + userID
    db.Query(query) // SQL injection vulnerability
}

func hashPassword(password string) string {
    return fmt.Sprintf("%x", md5.Sum([]byte(password))) // Weak hash
}
`
	if err := os.WriteFile(insecureFile, []byte(insecureContent), 0644); err != nil {
		t.Fatalf("不安全ファイル作成エラー: %v", err)
	}

	constraints := &security.Constraints{
		MaxTimeout: 30,
	}

	analyzer := NewAdvancedProjectAnalyzer(constraints, tempDir)
	analysis, err := analyzer.AnalyzeAdvanced()
	if err != nil {
		t.Fatalf("セキュリティ分析エラー: %v", err)
	}

	if analysis.SecurityAnalysis == nil {
		t.Fatal("セキュリティ分析結果がありません")
	}

	// 機密情報漏洩の検出確認
	if len(analysis.SecurityAnalysis.SecretLeaks) == 0 {
		t.Error("機密情報漏洩が検出されませんでした")
	} else {
		t.Logf("検出された機密情報漏洩: %d件", len(analysis.SecurityAnalysis.SecretLeaks))
		for _, leak := range analysis.SecurityAnalysis.SecretLeaks {
			t.Logf("  - %s:%d (%s)", leak.File, leak.Line, leak.Type)
		}
	}

	// 不安全パターンの検出確認
	if len(analysis.SecurityAnalysis.InsecurePatterns) == 0 {
		t.Error("不安全パターンが検出されませんでした")
	} else {
		t.Logf("検出された不安全パターン: %d件", len(analysis.SecurityAnalysis.InsecurePatterns))
		for _, pattern := range analysis.SecurityAnalysis.InsecurePatterns {
			t.Logf("  - %s:%d (%s)", pattern.File, pattern.Line, pattern.Pattern)
		}
	}

	// セキュリティスコアの確認
	if analysis.SecurityAnalysis.SecurityScore == 100 {
		t.Error("セキュリティ問題があるのにスコアが100です")
	}

	t.Logf("セキュリティスコア: %d", analysis.SecurityAnalysis.SecurityScore)
}

func TestArchitectureMapping(t *testing.T) {
	// テスト用プロジェクトディレクトリ作成
	tempDir := setupTestProject(t)
	defer os.RemoveAll(tempDir)

	// アーキテクチャディレクトリ作成
	dirs := []string{
		"api/handlers",
		"internal/service",
		"internal/repository",
		"ui/components",
		"config",
		"tests",
		"docs",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("ディレクトリ作成エラー: %v", err)
		}

		// 各ディレクトリにダミーファイルを作成
		dummyFile := filepath.Join(tempDir, dir, "dummy.go")
		if err := os.WriteFile(dummyFile, []byte("package main\n"), 0644); err != nil {
			t.Fatalf("ダミーファイル作成エラー: %v", err)
		}
	}

	constraints := &security.Constraints{
		MaxTimeout: 30,
	}

	analyzer := NewAdvancedProjectAnalyzer(constraints, tempDir)
	analysis, err := analyzer.AnalyzeAdvanced()
	if err != nil {
		t.Fatalf("アーキテクチャ分析エラー: %v", err)
	}

	if analysis.Architecture == nil {
		t.Fatal("アーキテクチャ分析結果がありません")
	}

	// レイヤーの検出確認
	if len(analysis.Architecture.Layers) == 0 {
		t.Error("アーキテクチャレイヤーが検出されませんでした")
	} else {
		t.Logf("検出されたアーキテクチャレイヤー: %d個", len(analysis.Architecture.Layers))
		for _, layer := range analysis.Architecture.Layers {
			t.Logf("  - %s: %v", layer.Name, layer.Directories)
		}
	}

	// モジュールの検出確認
	t.Logf("検出されたモジュール: %d個", len(analysis.Architecture.Modules))
}

func TestBuildSystemDetection(t *testing.T) {
	// テスト用プロジェクトディレクトリ作成
	tempDir := setupTestProject(t)
	defer os.RemoveAll(tempDir)

	// 複数のビルドシステムファイルを作成
	buildFiles := map[string]string{
		"Makefile": `
all: build test
build:
	go build ./...
test:
	go test ./...
`,
		"Dockerfile": `
FROM golang:1.20
WORKDIR /app
COPY . .
RUN go build -o main ./cmd/vyb
EXPOSE 8080
CMD ["./main"]
`,
		"docker-compose.yml": `
version: '3'
services:
  app:
    build: .
    ports:
      - "8080:8080"
`,
		".github/workflows/ci.yml": `
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
    - run: go test ./...
`,
	}

	for filename, content := range buildFiles {
		filePath := filepath.Join(tempDir, filename)
		if strings.Contains(filename, "/") {
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				t.Fatalf("ディレクトリ作成エラー: %v", err)
			}
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("ファイル作成エラー (%s): %v", filename, err)
		}
	}

	constraints := &security.Constraints{
		MaxTimeout: 30,
	}

	analyzer := NewAdvancedProjectAnalyzer(constraints, tempDir)
	analysis, err := analyzer.AnalyzeAdvanced()
	if err != nil {
		t.Fatalf("ビルドシステム分析エラー: %v", err)
	}

	if len(analysis.BuildSystems) == 0 {
		t.Error("ビルドシステムが検出されませんでした")
	} else {
		t.Logf("検出されたビルドシステム: %d個", len(analysis.BuildSystems))
		for _, system := range analysis.BuildSystems {
			t.Logf("  - %s: %s", system.Type, system.ConfigFile)
		}
	}

	// 期待されるビルドシステムが検出されているか確認
	expectedSystems := []string{"makefile", "docker", "github_actions", "go_native"}
	detectedSystems := make(map[string]bool)
	for _, system := range analysis.BuildSystems {
		detectedSystems[system.Type] = true
	}

	for _, expected := range expectedSystems {
		if !detectedSystems[expected] {
			t.Errorf("期待されるビルドシステム '%s' が検出されませんでした", expected)
		}
	}
}

// テスト用プロジェクト環境のセットアップ
func setupTestProject(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "vyb-test-*")
	if err != nil {
		t.Fatalf("テンポラリディレクトリ作成エラー: %v", err)
	}

	// 基本的なGoプロジェクト構造を作成
	dirs := []string{
		"cmd/vyb",
		"internal/tools",
		"internal/config",
		"pkg/types",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("ディレクトリ作成エラー: %v", err)
		}
	}

	// go.modファイル作成
	goMod := `module github.com/test/project

go 1.20

require (
	github.com/spf13/cobra v1.9.1
	golang.org/x/term v0.6.0
)
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("go.mod作成エラー: %v", err)
	}

	// main.goファイル作成
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "cmd/vyb/main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("main.go作成エラー: %v", err)
	}

	// README.mdファイル作成
	readme := `# Test Project

This is a test project for vyb-code functionality validation.

## Features

- Project analysis
- Build system integration
- Security scanning

## Usage

` + "```bash" + `
go build ./cmd/vyb
./vyb --help
` + "```" + `
`
	if err := os.WriteFile(filepath.Join(tempDir, "README.md"), []byte(readme), 0644); err != nil {
		t.Fatalf("README.md作成エラー: %v", err)
	}

	return tempDir
}

// ベンチマークテスト
func BenchmarkAdvancedProjectAnalysis(b *testing.B) {
	tempDir := setupTestProject(&testing.T{})
	defer os.RemoveAll(tempDir)

	constraints := &security.Constraints{
		MaxTimeout: 30,
	}

	analyzer := NewAdvancedProjectAnalyzer(constraints, tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeAdvanced()
		if err != nil {
			b.Fatalf("解析エラー: %v", err)
		}
	}
}

func BenchmarkBuildSystemDetection(b *testing.B) {
	tempDir := setupTestProject(&testing.T{})
	defer os.RemoveAll(tempDir)

	// Makefile作成
	makefileContent := `all: build test
build:
	echo "Building"
test:
	echo "Testing"`

	if err := os.WriteFile(filepath.Join(tempDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		b.Fatalf("Makefile作成エラー: %v", err)
	}

	constraints := &security.Constraints{
		MaxTimeout: 30,
	}

	analyzer := NewAdvancedProjectAnalyzer(constraints, tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analysis, err := analyzer.AnalyzeAdvanced()
		if err != nil {
			b.Fatalf("解析エラー: %v", err)
		}
		_ = analysis.BuildSystems // Use the result to prevent optimization
	}
}

// エラーケースのテスト
func TestErrorHandling(t *testing.T) {
	// 存在しないディレクトリでの分析
	constraints := &security.Constraints{
		MaxTimeout: 30,
	}

	analyzer := NewAdvancedProjectAnalyzer(constraints, "/nonexistent/directory")
	_, err := analyzer.AnalyzeAdvanced()
	if err == nil {
		t.Error("存在しないディレクトリでエラーが発生しませんでした")
	}

	// 不正なパラメータでのツール実行
	registry := NewToolRegistry(constraints, os.TempDir(), 1024*1024, nil)
	result, err := registry.ExecuteTool("project_analyze", map[string]interface{}{
		"analysis_type": "invalid_type",
	})

	if err == nil && !result.IsError {
		t.Error("不正なパラメータでエラーが発生しませんでした")
	}
}
