package ai

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glkt/vyb-code/internal/security"
)

// モックLLMクライアント
type MockLLMClient struct {
	responses map[string]string
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: map[string]string{
			"code_analysis": `{
				"issues": [
					{
						"type": "bug",
						"severity": "high",
						"line": 10,
						"title": "Null pointer exception",
						"description": "Variable may be null",
						"solution": "Add null check"
					}
				],
				"suggestions": [
					{
						"type": "optimization",
						"priority": "medium",
						"line_range": [5, 8],
						"title": "Use more efficient algorithm",
						"description": "Current O(n²) can be optimized to O(n log n)",
						"benefits": "Improved performance"
					}
				],
				"documentation_gaps": [],
				"performance_insights": []
			}`,
			"code_generation": "以下は生成されたGoのコードです：\n\n```go\npackage main\n\nfunc TestFunction() {\n\t// Generated test code\n\treturn\n}\n```",
			"test_generation": "以下はテストコードです：\n\n```go\npackage main\n\nimport \"testing\"\n\nfunc TestTestFunction(t *testing.T) {\n\t// Generated test\n\tTestFunction()\n}\n```",
			"summary":         "This is a test summary of the analysis results.",
		},
	}
}

func (m *MockLLMClient) GenerateResponse(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error) {
	content := request.Messages[0].Content

	// プロンプトの種類に応じてレスポンスを選択
	if strings.Contains(content, "分析") || strings.Contains(content, "analysis") {
		return &GenerateResponse{
			Content: m.responses["code_analysis"],
		}, nil
	} else if strings.Contains(content, "テスト") || strings.Contains(content, "test") {
		return &GenerateResponse{
			Content: m.responses["test_generation"],
		}, nil
	} else if strings.Contains(content, "生成") || strings.Contains(content, "generate") {
		return &GenerateResponse{
			Content: m.responses["code_generation"],
		}, nil
	} else if strings.Contains(content, "要約") || strings.Contains(content, "summary") {
		return &GenerateResponse{
			Content: m.responses["summary"],
		}, nil
	}

	return &GenerateResponse{
		Content: "Mock response",
	}, nil
}

// テストプロジェクトのセットアップ
func setupTestProject(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "ai-test-*")
	if err != nil {
		t.Fatalf("テンポラリディレクトリ作成エラー: %v", err)
	}

	// テスト用ファイル構造を作成
	dirs := []string{
		"internal/service",
		"internal/repository",
		"cmd/server",
		"pkg/utils",
		"test",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("ディレクトリ作成エラー: %v", err)
		}
	}

	// テスト用ソースファイルを作成
	files := map[string]string{
		"internal/service/user.go": `package service

import "errors"

type UserService struct {
	repo UserRepository
}

func (s *UserService) GetUser(id string) (*User, error) {
	if id == "" {
		return nil, errors.New("invalid id")
	}
	
	user := s.repo.FindByID(id)
	if user == nil {
		return nil, errors.New("user not found")
	}
	
	return user, nil
}`,

		"internal/repository/user.go": `package repository

type User struct {
	ID   string
	Name string
}

type UserRepository interface {
	FindByID(id string) *User
}

type userRepo struct{}

func (r *userRepo) FindByID(id string) *User {
	// Mock implementation
	return &User{ID: id, Name: "Test User"}
}`,

		"cmd/server/main.go": `package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", handler)
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}`,

		"pkg/utils/helper.go": `package utils

import "strings"

func ToUpperCase(s string) string {
	return strings.ToUpper(s)
}

func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}`,

		"test/user_test.go": `package test

import (
	"testing"
)

func TestUserService(t *testing.T) {
	// Test implementation
}`,

		"go.mod": `module github.com/test/ai-project

go 1.20

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/stretchr/testify v1.8.4
)`,

		"README.md": `# AI Test Project

This is a test project for AI code analysis functionality.

## Features

- User management service
- REST API endpoints  
- Database repository pattern

## Usage

` + "```bash" + `
go run cmd/server/main.go
` + "```",
	}

	for filename, content := range files {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("ファイル作成エラー (%s): %v", filename, err)
		}
	}

	return tempDir
}

// CodeAnalyzer のテスト
func TestCodeAnalyzer_AnalyzeProject(t *testing.T) {
	// テストプロジェクトをセットアップ
	tempDir := setupTestProject(t)
	defer os.RemoveAll(tempDir)

	mockClient := NewMockLLMClient()
	constraints := &security.Constraints{
		MaxTimeout: 30,
	}

	analyzer := NewCodeAnalyzer(mockClient, constraints, tempDir)

	// プロジェクト分析を実行
	ctx := context.Background()
	result, err := analyzer.AnalyzeProject(ctx)

	if err != nil {
		t.Fatalf("プロジェクト分析エラー: %v", err)
	}

	// 結果を検証
	if result == nil {
		t.Fatal("分析結果がnilです")
	}

	if len(result.Issues) == 0 {
		t.Error("問題が検出されませんでした")
	}

	if result.Summary == "" {
		t.Error("要約が生成されませんでした")
	}

	if result.ProcessingTime == 0 {
		t.Error("処理時間が記録されていません")
	}

	t.Logf("分析完了: 問題数=%d, 提案数=%d, 処理時間=%v",
		len(result.Issues), len(result.Suggestions), result.ProcessingTime)
}

func TestCodeAnalyzer_Config(t *testing.T) {
	mockClient := NewMockLLMClient()
	constraints := &security.Constraints{MaxTimeout: 30}
	analyzer := NewCodeAnalyzer(mockClient, constraints, "/tmp")

	// デフォルト設定をテスト
	if analyzer.config.MaxFileSize != 1024*1024 {
		t.Error("デフォルトファイルサイズが正しくありません")
	}

	if analyzer.config.AnalysisDepth != "detailed" {
		t.Error("デフォルト分析深度が正しくありません")
	}

	// 設定を更新
	newConfig := &AnalysisConfig{
		MaxFileSize:   2048 * 1024,
		AnalysisDepth: "comprehensive",
	}
	analyzer.UpdateConfig(newConfig)

	if analyzer.config.MaxFileSize != 2048*1024 {
		t.Error("設定更新が反映されていません")
	}
}

// CodeGenerator のテスト
func TestCodeGenerator_GenerateCode(t *testing.T) {
	tempDir := setupTestProject(t)
	defer os.RemoveAll(tempDir)

	mockClient := NewMockLLMClient()
	constraints := &security.Constraints{MaxTimeout: 30}

	generator := NewCodeGenerator(mockClient, constraints, tempDir)

	// 関数生成リクエスト
	request := &CodeGenerationRequest{
		Type:        "function",
		Language:    "go",
		Description: "Calculate sum of integers",
		Requirements: []string{
			"Handle empty slice",
			"Return error for invalid input",
		},
		TestRequired: true,
		DocRequired:  true,
	}

	ctx := context.Background()
	result, err := generator.GenerateCode(ctx, request)

	if err != nil {
		t.Fatalf("コード生成エラー: %v", err)
	}

	// 結果を検証
	if len(result.GeneratedCode) == 0 {
		t.Error("コードが生成されませんでした")
	}

	if len(result.CreatedTests) == 0 {
		t.Error("テストが生成されませんでした")
	}

	if len(result.Documentation) == 0 {
		t.Error("ドキュメントが生成されませんでした")
	}

	if result.Summary == "" {
		t.Error("要約が生成されませんでした")
	}

	t.Logf("生成完了: コード=%d個, テスト=%d個, ドキュメント=%d個",
		len(result.GeneratedCode), len(result.CreatedTests), len(result.Documentation))
}

func TestCodeGenerator_ValidateRequest(t *testing.T) {
	mockClient := NewMockLLMClient()
	constraints := &security.Constraints{MaxTimeout: 30}
	generator := NewCodeGenerator(mockClient, constraints, "/tmp")

	// 有効なリクエスト
	validRequest := &CodeGenerationRequest{
		Type:        "function",
		Language:    "go",
		Description: "Test function",
	}

	if err := generator.validateRequest(validRequest); err != nil {
		t.Errorf("有効なリクエストが拒否されました: %v", err)
	}

	// 無効なリクエスト - 説明なし
	invalidRequest1 := &CodeGenerationRequest{
		Type:     "function",
		Language: "go",
	}

	if err := generator.validateRequest(invalidRequest1); err == nil {
		t.Error("無効なリクエスト（説明なし）が受け入れられました")
	}

	// 無効なリクエスト - サポートされていない言語
	invalidRequest2 := &CodeGenerationRequest{
		Type:        "function",
		Language:    "cobol",
		Description: "Test function",
	}

	if err := generator.validateRequest(invalidRequest2); err == nil {
		t.Error("無効なリクエスト（サポートされていない言語）が受け入れられました")
	}

	// セーフティモードでの危険なリクエスト
	dangerousRequest := &CodeGenerationRequest{
		Type:        "function",
		Language:    "go",
		Description: "Execute rm -rf command",
	}

	if err := generator.validateRequest(dangerousRequest); err == nil {
		t.Error("危険なリクエストが受け入れられました")
	}
}

// DependencyVisualizer のテスト
func TestDependencyVisualizer_VisualizeProject(t *testing.T) {
	tempDir := setupTestProject(t)
	defer os.RemoveAll(tempDir)

	constraints := &security.Constraints{MaxTimeout: 30}
	visualizer := NewDependencyVisualizer(constraints, tempDir)

	ctx := context.Background()
	result, err := visualizer.VisualizeProject(ctx)

	if err != nil {
		t.Fatalf("依存関係可視化エラー: %v", err)
	}

	// 結果を検証
	if len(result.Nodes) == 0 {
		t.Error("ノードが作成されませんでした")
	}

	if result.ProjectName == "" {
		t.Error("プロジェクト名が設定されていません")
	}

	if result.TotalFiles == 0 {
		t.Error("ファイル数が正しくありません")
	}

	// メトリクスを検証
	if result.Metrics.MaxDepth == 0 {
		t.Error("最大深度が計算されていません")
	}

	t.Logf("可視化完了: ノード=%d個, エッジ=%d個, クラスター=%d個",
		len(result.Nodes), len(result.Edges), len(result.Clusters))
}

func TestDependencyVisualizer_Config(t *testing.T) {
	constraints := &security.Constraints{MaxTimeout: 30}
	visualizer := NewDependencyVisualizer(constraints, "/tmp")

	// デフォルト設定をテスト
	if visualizer.config.MaxNodes != 200 {
		t.Error("デフォルト最大ノード数が正しくありません")
	}

	if visualizer.config.LayoutAlgorithm != "force" {
		t.Error("デフォルトレイアウトアルゴリズムが正しくありません")
	}

	// 設定を更新
	newConfig := &VisualizationConfig{
		MaxNodes:        100,
		LayoutAlgorithm: "hierarchical",
	}
	visualizer.UpdateConfig(newConfig)

	if visualizer.config.MaxNodes != 100 {
		t.Error("設定更新が反映されていません")
	}
}

// MultiRepoManager のテスト
func TestMultiRepoManager_DiscoverRepositories(t *testing.T) {
	// ワークスペースディレクトリを作成
	workspaceDir, err := os.MkdirTemp("", "multi-repo-test-*")
	if err != nil {
		t.Fatalf("ワークスペースディレクトリ作成エラー: %v", err)
	}
	defer os.RemoveAll(workspaceDir)

	// 複数のリポジトリを作成
	repos := []string{"project-a", "project-b", "shared-lib"}

	for _, repoName := range repos {
		repoDir := filepath.Join(workspaceDir, repoName)
		gitDir := filepath.Join(repoDir, ".git")

		if err := os.MkdirAll(gitDir, 0755); err != nil {
			t.Fatalf("リポジトリディレクトリ作成エラー: %v", err)
		}

		// HEAD ファイルを作成
		headContent := "ref: refs/heads/main\n"
		if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte(headContent), 0644); err != nil {
			t.Fatalf("HEADファイル作成エラー: %v", err)
		}

		// テスト用ソースファイルを作成
		mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello from ` + repoName + `")
}`
		if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte(mainGo), 0644); err != nil {
			t.Fatalf("メインファイル作成エラー: %v", err)
		}

		// go.mod を作成
		goMod := `module github.com/test/` + repoName + `

go 1.20`
		if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte(goMod), 0644); err != nil {
			t.Fatalf("go.mod作成エラー: %v", err)
		}
	}

	constraints := &security.Constraints{MaxTimeout: 30}
	manager := NewMultiRepoManager(constraints, workspaceDir)

	ctx := context.Background()
	err = manager.DiscoverRepositories(ctx)

	if err != nil {
		t.Fatalf("リポジトリ発見エラー: %v", err)
	}

	// 結果を検証
	repositories := manager.GetAllRepositories()
	if len(repositories) != len(repos) {
		t.Errorf("期待されるリポジトリ数: %d, 実際: %d", len(repos), len(repositories))
	}

	for _, repoName := range repos {
		found := false
		for _, repo := range repositories {
			if repo.Name == repoName {
				found = true

				// 基本プロパティを検証
				if repo.Language != "Go" {
					t.Errorf("リポジトリ %s の言語が正しくありません: %s", repoName, repo.Language)
				}

				if repo.Branch != "main" {
					t.Errorf("リポジトリ %s のブランチが正しくありません: %s", repoName, repo.Branch)
				}

				break
			}
		}

		if !found {
			t.Errorf("リポジトリ %s が発見されませんでした", repoName)
		}
	}

	t.Logf("発見完了: %d個のリポジトリ", len(repositories))
}

func TestMultiRepoManager_AnalyzeWorkspace(t *testing.T) {
	// 簡単なワークスペースを作成
	workspaceDir, err := os.MkdirTemp("", "workspace-test-*")
	if err != nil {
		t.Fatalf("ワークスペースディレクトリ作成エラー: %v", err)
	}
	defer os.RemoveAll(workspaceDir)

	// 単一リポジトリを作成
	repoDir := filepath.Join(workspaceDir, "test-repo")
	gitDir := filepath.Join(repoDir, ".git")

	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("リポジトリディレクトリ作成エラー: %v", err)
	}

	// 基本ファイルを作成
	files := map[string]string{
		".git/HEAD": "ref: refs/heads/main\n",
		"main.go": `package main
import "fmt"
func main() { fmt.Println("Hello") }`,
		"go.mod": "module test-repo\ngo 1.20\n",
	}

	for filename, content := range files {
		filePath := filepath.Join(repoDir, filename)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			continue // ディレクトリが既に存在
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("ファイル作成エラー (%s): %v", filename, err)
		}
	}

	constraints := &security.Constraints{MaxTimeout: 30}
	manager := NewMultiRepoManager(constraints, workspaceDir)

	ctx := context.Background()
	analysis, err := manager.AnalyzeWorkspace(ctx)

	if err != nil {
		t.Fatalf("ワークスペース分析エラー: %v", err)
	}

	// 結果を検証
	if analysis.Overview.TotalRepositories == 0 {
		t.Error("リポジトリが検出されませんでした")
	}

	if analysis.Overview.ActiveRepositories == 0 {
		t.Error("アクティブリポジトリが検出されませんでした")
	}

	if len(analysis.Overview.LanguageDistribution) == 0 {
		t.Error("言語分布が計算されませんでした")
	}

	if analysis.QualityMetrics.OverallScore == 0 {
		t.Error("品質スコアが計算されませんでした")
	}

	if len(analysis.Recommendations) == 0 {
		t.Error("推奨事項が生成されませんでした")
	}

	t.Logf("ワークスペース分析完了: リポジトリ=%d個, 品質スコア=%d, 推奨事項=%d個",
		analysis.Overview.TotalRepositories,
		analysis.QualityMetrics.OverallScore,
		len(analysis.Recommendations))
}

// ベンチマークテスト
func BenchmarkCodeAnalyzer_AnalyzeProject(b *testing.B) {
	tempDir := setupTestProject(&testing.T{})
	defer os.RemoveAll(tempDir)

	mockClient := NewMockLLMClient()
	constraints := &security.Constraints{MaxTimeout: 30}
	analyzer := NewCodeAnalyzer(mockClient, constraints, tempDir)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeProject(ctx)
		if err != nil {
			b.Fatalf("プロジェクト分析エラー: %v", err)
		}
	}
}

func BenchmarkDependencyVisualizer_VisualizeProject(b *testing.B) {
	tempDir := setupTestProject(&testing.T{})
	defer os.RemoveAll(tempDir)

	constraints := &security.Constraints{MaxTimeout: 30}
	visualizer := NewDependencyVisualizer(constraints, tempDir)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := visualizer.VisualizeProject(ctx)
		if err != nil {
			b.Fatalf("依存関係可視化エラー: %v", err)
		}
	}
}
