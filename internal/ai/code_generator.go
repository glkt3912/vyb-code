package ai

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

// コード生成結果
type CodeGenerationResult struct {
	GeneratedCode     []GeneratedFile        `json:"generated_code"`
	ModifiedFiles     []ModifiedFile         `json:"modified_files"`
	CreatedTests      []GeneratedTest        `json:"created_tests"`
	Documentation     []GeneratedDoc         `json:"documentation"`
	Summary           string                 `json:"summary"`
	GenerationTime    time.Duration          `json:"generation_time"`
	Suggestions       []GenerationSuggestion `json:"suggestions"`
	Warnings          []string               `json:"warnings"`
	GeneratedTimestamp time.Time             `json:"generated_timestamp"`
}

// 生成されたファイル
type GeneratedFile struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	Language    string `json:"language"`
	Type        string `json:"type"`        // "function", "class", "module", "config"
	Description string `json:"description"`
	Dependencies []string `json:"dependencies"`
}

// 変更されたファイル
type ModifiedFile struct {
	Path         string   `json:"path"`
	OriginalCode string   `json:"original_code"`
	ModifiedCode string   `json:"modified_code"`
	Changes      []Change `json:"changes"`
	Description  string   `json:"description"`
}

// コード変更
type Change struct {
	Type        string `json:"type"`        // "addition", "modification", "deletion"
	LineStart   int    `json:"line_start"`
	LineEnd     int    `json:"line_end"`
	OldContent  string `json:"old_content"`
	NewContent  string `json:"new_content"`
	Reason      string `json:"reason"`
}

// 生成されたテスト
type GeneratedTest struct {
	Path            string   `json:"path"`
	TestContent     string   `json:"test_content"`
	TestedFunction  string   `json:"tested_function"`
	TestCases       []string `json:"test_cases"`
	CoverageAreas   []string `json:"coverage_areas"`
	Dependencies    []string `json:"dependencies"`
}

// 生成されたドキュメント
type GeneratedDoc struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	DocType     string `json:"doc_type"`    // "README", "API", "user_guide", "comment"
	Target      string `json:"target"`      // what is being documented
	Format      string `json:"format"`      // "markdown", "rst", "plain"
}

// 生成提案
type GenerationSuggestion struct {
	Type         string `json:"type"`         // "improvement", "alternative", "extension"
	Priority     string `json:"priority"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Implementation string `json:"implementation"`
	Benefits     string `json:"benefits"`
}

// コード生成リクエスト
type CodeGenerationRequest struct {
	Type            string            `json:"type"`             // "function", "class", "module", "test", "refactor"
	Language        string            `json:"language"`         // target programming language
	Description     string            `json:"description"`      // what to generate
	Context         *GenerationContext `json:"context"`         // surrounding code context
	Requirements    []string          `json:"requirements"`     // specific requirements
	Style           *StylePreferences `json:"style"`           // coding style preferences
	TargetFile      string            `json:"target_file"`      // where to place the code
	ExistingCode    string            `json:"existing_code"`    // existing code to modify
	Dependencies    []string          `json:"dependencies"`     // required dependencies
	TestRequired    bool              `json:"test_required"`    // whether to generate tests
	DocRequired     bool              `json:"doc_required"`     // whether to generate docs
}

// 生成コンテキスト
type GenerationContext struct {
	ProjectType     string            `json:"project_type"`     // "web", "cli", "library", etc.
	Framework       string            `json:"framework"`        // "react", "gin", "django", etc.
	Architecture    string            `json:"architecture"`     // "mvc", "clean", "hexagonal"
	ExistingFiles   []string          `json:"existing_files"`   // related existing files
	ImportStatements []string         `json:"import_statements"` // existing imports
	GlobalVariables []string          `json:"global_variables"` // available global vars
	CustomTypes     map[string]string `json:"custom_types"`     // project-specific types
}

// スタイル設定
type StylePreferences struct {
	IndentationType string            `json:"indentation_type"` // "spaces", "tabs"
	IndentationSize int               `json:"indentation_size"`
	LineLength      int               `json:"line_length"`
	NamingConvention string           `json:"naming_convention"` // "camelCase", "snake_case", etc.
	CommentStyle    string            `json:"comment_style"`     // preferred comment format
	ErrorHandling   string            `json:"error_handling"`    // error handling strategy
	CustomRules     map[string]string `json:"custom_rules"`      // project-specific style rules
}

// AIコード生成器
type CodeGenerator struct {
	llmClient     LLMClient
	constraints   *security.Constraints
	projectDir    string
	config        *GenerationConfig
}

// 生成設定
type GenerationConfig struct {
	MaxFileSize       int64             `json:"max_file_size"`
	AllowedLanguages  []string          `json:"allowed_languages"`
	DefaultStyle      *StylePreferences `json:"default_style"`
	SafetyMode        bool              `json:"safety_mode"`        // extra validation
	BackupOriginals   bool              `json:"backup_originals"`   // backup before modification
	CustomPrompts     map[string]string `json:"custom_prompts"`     // custom generation prompts
	ValidationRules   []string          `json:"validation_rules"`   // code validation rules
}

// AIコード生成器を作成
func NewCodeGenerator(llmClient LLMClient, constraints *security.Constraints, projectDir string) *CodeGenerator {
	return &CodeGenerator{
		llmClient:   llmClient,
		constraints: constraints,
		projectDir:  projectDir,
		config: &GenerationConfig{
			MaxFileSize:      1024 * 1024, // 1MB
			AllowedLanguages: []string{"go", "javascript", "typescript", "python", "java", "rust"},
			DefaultStyle: &StylePreferences{
				IndentationType:  "tabs",
				IndentationSize:  4,
				LineLength:       100,
				NamingConvention: "camelCase",
				CommentStyle:     "standard",
				ErrorHandling:    "return_error",
				CustomRules:      make(map[string]string),
			},
			SafetyMode:      true,
			BackupOriginals: true,
			CustomPrompts:   make(map[string]string),
			ValidationRules: []string{
				"no_hardcoded_secrets",
				"proper_error_handling",
				"input_validation",
				"secure_by_default",
			},
		},
	}
}

// 設定を更新
func (cg *CodeGenerator) UpdateConfig(config *GenerationConfig) {
	if config != nil {
		cg.config = config
	}
}

// コードを生成
func (cg *CodeGenerator) GenerateCode(ctx context.Context, request *CodeGenerationRequest) (*CodeGenerationResult, error) {
	startTime := time.Now()

	result := &CodeGenerationResult{
		GeneratedCode:      []GeneratedFile{},
		ModifiedFiles:      []ModifiedFile{},
		CreatedTests:       []GeneratedTest{},
		Documentation:      []GeneratedDoc{},
		Suggestions:        []GenerationSuggestion{},
		Warnings:           []string{},
		GeneratedTimestamp: startTime,
	}

	// リクエスト検証
	if err := cg.validateRequest(request); err != nil {
		return nil, fmt.Errorf("リクエスト検証エラー: %w", err)
	}

	// プロジェクトコンテキストを分析
	context, err := cg.analyzeProjectContext(request)
	if err != nil {
		fmt.Printf("コンテキスト分析警告: %v\n", err)
	}

	// メインコード生成
	switch request.Type {
	case "function":
		err = cg.generateFunction(ctx, request, context, result)
	case "class":
		err = cg.generateClass(ctx, request, context, result)
	case "module":
		err = cg.generateModule(ctx, request, context, result)
	case "test":
		err = cg.generateTest(ctx, request, context, result)
	case "refactor":
		err = cg.performRefactoring(ctx, request, context, result)
	default:
		return nil, fmt.Errorf("サポートされていない生成タイプ: %s", request.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("コード生成エラー: %w", err)
	}

	// 追加のテスト生成
	if request.TestRequired && request.Type != "test" {
		testErr := cg.generateAdditionalTests(ctx, request, result)
		if testErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("テスト生成警告: %v", testErr))
		}
	}

	// ドキュメント生成
	if request.DocRequired {
		docErr := cg.generateDocumentation(ctx, request, result)
		if docErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("ドキュメント生成警告: %v", docErr))
		}
	}

	// 生成されたコードの検証
	err = cg.validateGeneratedCode(result)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("コード検証警告: %v", err))
	}

	// 改善提案を生成
	cg.generateImprovementSuggestions(ctx, request, result)

	// 要約を生成
	err = cg.generateSummary(ctx, result)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("要約生成警告: %v", err))
	}

	result.GenerationTime = time.Since(startTime)
	
	return result, nil
}

// リクエスト検証
func (cg *CodeGenerator) validateRequest(request *CodeGenerationRequest) error {
	if request.Description == "" {
		return fmt.Errorf("説明が必要です")
	}

	if request.Language == "" {
		return fmt.Errorf("言語の指定が必要です")
	}

	// 許可された言語かチェック
	allowed := false
	for _, lang := range cg.config.AllowedLanguages {
		if strings.EqualFold(lang, request.Language) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("サポートされていない言語: %s", request.Language)
	}

	// セキュリティチェック
	if cg.config.SafetyMode {
		dangerousKeywords := []string{
			"rm -rf", "delete", "drop database", "exec", "eval", "system",
			"shell_exec", "passthru", "__import__", "subprocess",
		}
		
		descLower := strings.ToLower(request.Description)
		for _, keyword := range dangerousKeywords {
			if strings.Contains(descLower, keyword) {
				return fmt.Errorf("安全でない操作が検出されました: %s", keyword)
			}
		}
	}

	return nil
}

// プロジェクトコンテキストを分析
func (cg *CodeGenerator) analyzeProjectContext(request *CodeGenerationRequest) (*GenerationContext, error) {
	context := &GenerationContext{
		ExistingFiles:   []string{},
		ImportStatements: []string{},
		GlobalVariables: []string{},
		CustomTypes:     make(map[string]string),
	}

	// プロジェクトタイプを推測
	context.ProjectType = cg.detectProjectType()

	// フレームワークを検出
	context.Framework = cg.detectFramework(request.Language)

	// 既存ファイルを分析
	if request.TargetFile != "" {
		targetPath := filepath.Join(cg.projectDir, request.TargetFile)
		if _, err := os.Stat(targetPath); err == nil {
			content, err := os.ReadFile(targetPath)
			if err == nil {
				context.ImportStatements = cg.extractImports(string(content), request.Language)
				context.CustomTypes = cg.extractCustomTypes(string(content), request.Language)
			}
		}
	}

	return context, nil
}

// プロジェクトタイプを検出
func (cg *CodeGenerator) detectProjectType() string {
	// 各種ファイルの存在をチェックしてプロジェクトタイプを推測
	files := map[string]string{
		"main.go":       "cli",
		"server.go":     "web",
		"package.json":  "web",
		"Dockerfile":    "containerized",
		"go.mod":        "library",
		"setup.py":      "python_package",
	}

	for filename, projectType := range files {
		if _, err := os.Stat(filepath.Join(cg.projectDir, filename)); err == nil {
			return projectType
		}
	}

	return "unknown"
}

// フレームワークを検出
func (cg *CodeGenerator) detectFramework(language string) string {
	frameworkFiles := map[string]map[string]string{
		"go": {
			"gin":     "github.com/gin-gonic/gin",
			"echo":    "github.com/labstack/echo",
			"fiber":   "github.com/gofiber/fiber",
			"cobra":   "github.com/spf13/cobra",
		},
		"javascript": {
			"react":   "react",
			"vue":     "vue",
			"angular": "@angular/core",
			"express": "express",
		},
		"python": {
			"django":  "django",
			"flask":   "flask",
			"fastapi": "fastapi",
		},
	}

	// go.mod や package.json から依存関係をチェック
	if frameworks, exists := frameworkFiles[strings.ToLower(language)]; exists {
		for framework, importPath := range frameworks {
			if cg.hasFrameworkDependency(language, importPath) {
				return framework
			}
		}
	}

	return "none"
}

// フレームワーク依存関係をチェック
func (cg *CodeGenerator) hasFrameworkDependency(language, importPath string) bool {
	switch strings.ToLower(language) {
	case "go":
		goModPath := filepath.Join(cg.projectDir, "go.mod")
		if content, err := os.ReadFile(goModPath); err == nil {
			return strings.Contains(string(content), importPath)
		}
	case "javascript", "typescript":
		packageJsonPath := filepath.Join(cg.projectDir, "package.json")
		if content, err := os.ReadFile(packageJsonPath); err == nil {
			return strings.Contains(string(content), importPath)
		}
	}
	return false
}

// import文を抽出
func (cg *CodeGenerator) extractImports(content, language string) []string {
	var imports []string
	
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		switch strings.ToLower(language) {
		case "go":
			if strings.HasPrefix(line, "import ") {
				imports = append(imports, line)
			}
		case "javascript", "typescript":
			if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "const ") && strings.Contains(line, "require(") {
				imports = append(imports, line)
			}
		case "python":
			if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "from ") {
				imports = append(imports, line)
			}
		}
	}
	
	return imports
}

// カスタムタイプを抽出
func (cg *CodeGenerator) extractCustomTypes(content, language string) map[string]string {
	types := make(map[string]string)
	
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		switch strings.ToLower(language) {
		case "go":
			if strings.HasPrefix(line, "type ") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					types[parts[1]] = strings.Join(parts[2:], " ")
				}
			}
		case "typescript":
			if strings.HasPrefix(line, "interface ") || strings.HasPrefix(line, "type ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					types[parts[1]] = "custom"
				}
			}
		}
	}
	
	return types
}

// 関数を生成
func (cg *CodeGenerator) generateFunction(ctx context.Context, request *CodeGenerationRequest, context *GenerationContext, result *CodeGenerationResult) error {
	prompt := cg.buildFunctionPrompt(request, context)
	
	response, err := cg.llmClient.GenerateResponse(ctx, &GenerateRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.2,
	})

	if err != nil {
		return err
	}

	// コードブロックを抽出
	code := cg.extractCodeBlock(response.Content)
	if code == "" {
		return fmt.Errorf("有効なコードブロックが見つかりません")
	}

	// 生成されたファイルを追加
	generatedFile := GeneratedFile{
		Path:        cg.generateFilePath(request),
		Content:     code,
		Language:    request.Language,
		Type:        "function",
		Description: request.Description,
		Dependencies: cg.extractDependencies(code, request.Language),
	}

	result.GeneratedCode = append(result.GeneratedCode, generatedFile)

	return nil
}

// 関数生成プロンプトを構築
func (cg *CodeGenerator) buildFunctionPrompt(request *CodeGenerationRequest, context *GenerationContext) string {
	prompt := fmt.Sprintf(`あなたは経験豊富な%sプログラマーです。以下の要求に基づいて高品質な関数を作成してください：

要求: %s
言語: %s
プロジェクトタイプ: %s
フレームワーク: %s

`, request.Language, request.Description, request.Language, context.ProjectType, context.Framework)

	// 既存のimport文を追加
	if len(context.ImportStatements) > 0 {
		prompt += "既存のimport文:\n"
		for _, imp := range context.ImportStatements {
			prompt += fmt.Sprintf("- %s\n", imp)
		}
		prompt += "\n"
	}

	// カスタムタイプを追加
	if len(context.CustomTypes) > 0 {
		prompt += "既存のカスタムタイプ:\n"
		for name, definition := range context.CustomTypes {
			prompt += fmt.Sprintf("- %s: %s\n", name, definition)
		}
		prompt += "\n"
	}

	// 要件を追加
	if len(request.Requirements) > 0 {
		prompt += "追加要件:\n"
		for _, req := range request.Requirements {
			prompt += fmt.Sprintf("- %s\n", req)
		}
		prompt += "\n"
	}

	prompt += `コード生成時の注意事項:
1. 適切なエラーハンドリングを含める
2. 明確で理解しやすいコメントを追加
3. セキュアなコード実装を心がける
4. 既存のプロジェクト構造に適合させる
5. 適切なテスタビリティを確保

必要に応じて適切な import 文も含めて、完全に動作するコードを提供してください。コードブロックは ` + "```" + request.Language + ` で囲んでください。`

	return prompt
}

// クラスを生成
func (cg *CodeGenerator) generateClass(ctx context.Context, request *CodeGenerationRequest, context *GenerationContext, result *CodeGenerationResult) error {
	// クラス生成の実装（関数生成と似たパターン）
	prompt := cg.buildClassPrompt(request, context)
	
	response, err := cg.llmClient.GenerateResponse(ctx, &GenerateRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.2,
	})

	if err != nil {
		return err
	}

	code := cg.extractCodeBlock(response.Content)
	if code == "" {
		return fmt.Errorf("有効なコードブロックが見つかりません")
	}

	generatedFile := GeneratedFile{
		Path:        cg.generateFilePath(request),
		Content:     code,
		Language:    request.Language,
		Type:        "class",
		Description: request.Description,
		Dependencies: cg.extractDependencies(code, request.Language),
	}

	result.GeneratedCode = append(result.GeneratedCode, generatedFile)

	return nil
}

// クラス生成プロンプトを構築
func (cg *CodeGenerator) buildClassPrompt(request *CodeGenerationRequest, context *GenerationContext) string {
	// クラス生成用のプロンプト（関数用と似た構造）
	return fmt.Sprintf(`あなたは経験豊富な%sプログラマーです。以下の要求に基づいて高品質なクラスを作成してください：

要求: %s
言語: %s

適切なコンストラクタ、メソッド、プロパティを含む完全なクラス定義を提供してください。
コードブロックは `+"`"+`%s で囲んでください。`,
		request.Language, request.Description, request.Language, request.Language)
}

// モジュールを生成
func (cg *CodeGenerator) generateModule(ctx context.Context, request *CodeGenerationRequest, context *GenerationContext, result *CodeGenerationResult) error {
	// モジュール生成の実装
	return fmt.Errorf("モジュール生成は未実装")
}

// テストを生成
func (cg *CodeGenerator) generateTest(ctx context.Context, request *CodeGenerationRequest, context *GenerationContext, result *CodeGenerationResult) error {
	prompt := cg.buildTestPrompt(request, context)
	
	response, err := cg.llmClient.GenerateResponse(ctx, &GenerateRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1, // テストは一貫性を重視
	})

	if err != nil {
		return err
	}

	code := cg.extractCodeBlock(response.Content)
	if code == "" {
		return fmt.Errorf("有効なテストコードが見つかりません")
	}

	testFile := GeneratedTest{
		Path:           cg.generateTestFilePath(request),
		TestContent:    code,
		TestedFunction: request.Description,
		TestCases:      cg.extractTestCases(code),
		CoverageAreas:  []string{"basic_functionality", "error_handling", "edge_cases"},
		Dependencies:   cg.extractDependencies(code, request.Language),
	}

	result.CreatedTests = append(result.CreatedTests, testFile)

	return nil
}

// テスト生成プロンプトを構築
func (cg *CodeGenerator) buildTestPrompt(request *CodeGenerationRequest, context *GenerationContext) string {
	return fmt.Sprintf(`以下のコードに対する包括的なテストを%sで作成してください：

テスト対象: %s
既存コード:
%s

以下の観点でテストを作成してください:
1. 正常系のテストケース
2. 異常系のテストケース  
3. 境界値テスト
4. エラーハンドリングのテスト

テストコードは`+"`"+`%s で囲んでください。`,
		request.Language, request.Description, request.ExistingCode, request.Language)
}

// リファクタリングを実行
func (cg *CodeGenerator) performRefactoring(ctx context.Context, request *CodeGenerationRequest, context *GenerationContext, result *CodeGenerationResult) error {
	// リファクタリングの実装
	return fmt.Errorf("リファクタリングは未実装")
}

// 追加テストを生成
func (cg *CodeGenerator) generateAdditionalTests(ctx context.Context, request *CodeGenerationRequest, result *CodeGenerationResult) error {
	// 生成されたコードに対するテストを作成
	for _, generatedFile := range result.GeneratedCode {
		testRequest := &CodeGenerationRequest{
			Type:         "test",
			Language:     generatedFile.Language,
			Description:  generatedFile.Description,
			ExistingCode: generatedFile.Content,
		}
		
		err := cg.generateTest(ctx, testRequest, &GenerationContext{}, result)
		if err != nil {
			return err
		}
	}
	return nil
}

// ドキュメントを生成
func (cg *CodeGenerator) generateDocumentation(ctx context.Context, request *CodeGenerationRequest, result *CodeGenerationResult) error {
	// ドキュメント生成の実装
	for _, generatedFile := range result.GeneratedCode {
		docContent := cg.generateDocContent(generatedFile)
		
		doc := GeneratedDoc{
			Path:    strings.TrimSuffix(generatedFile.Path, filepath.Ext(generatedFile.Path)) + "_doc.md",
			Content: docContent,
			DocType: "API",
			Target:  generatedFile.Description,
			Format:  "markdown",
		}
		
		result.Documentation = append(result.Documentation, doc)
	}
	return nil
}

// ドキュメント内容を生成
func (cg *CodeGenerator) generateDocContent(file GeneratedFile) string {
	return fmt.Sprintf(`# %s

## Description
%s

## Language
%s

## Dependencies
%s

## Usage
See the generated code in %s

## Generated Code
`+"```"+`%s
%s
`+"```"+`
`, file.Description, file.Description, file.Language, 
		strings.Join(file.Dependencies, ", "), file.Path, 
		file.Language, file.Content)
}

// 生成されたコードを検証
func (cg *CodeGenerator) validateGeneratedCode(result *CodeGenerationResult) error {
	for _, file := range result.GeneratedCode {
		// 基本的な構文チェック
		if err := cg.performBasicSyntaxCheck(file); err != nil {
			return fmt.Errorf("構文チェックエラー in %s: %w", file.Path, err)
		}
		
		// セキュリティチェック
		if err := cg.performSecurityCheck(file); err != nil {
			return fmt.Errorf("セキュリティチェックエラー in %s: %w", file.Path, err)
		}
	}
	return nil
}

// 基本構文チェック
func (cg *CodeGenerator) performBasicSyntaxCheck(file GeneratedFile) error {
	// 言語別の基本的な構文チェック（簡略版）
	switch strings.ToLower(file.Language) {
	case "go":
		return cg.checkGoSyntax(file.Content)
	case "javascript", "typescript":
		return cg.checkJSSyntax(file.Content)
	case "python":
		return cg.checkPythonSyntax(file.Content)
	}
	return nil
}

// Go構文チェック
func (cg *CodeGenerator) checkGoSyntax(content string) error {
	// 基本的なチェック
	if !strings.Contains(content, "package ") {
		return fmt.Errorf("package宣言がありません")
	}
	
	// 括弧の対応チェック
	if strings.Count(content, "{") != strings.Count(content, "}") {
		return fmt.Errorf("括弧の対応が不正です")
	}
	
	return nil
}

// JavaScript構文チェック
func (cg *CodeGenerator) checkJSSyntax(content string) error {
	// 基本的なチェック
	if strings.Count(content, "{") != strings.Count(content, "}") {
		return fmt.Errorf("括弧の対応が不正です")
	}
	return nil
}

// Python構文チェック
func (cg *CodeGenerator) checkPythonSyntax(content string) error {
	// 基本的なチェック - インデントの確認など
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		if strings.HasSuffix(strings.TrimSpace(line), ":") {
			// 次の行のインデントをチェック
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != "" {
				if !strings.HasPrefix(lines[i+1], "    ") && !strings.HasPrefix(lines[i+1], "\t") {
					return fmt.Errorf("インデントが不正です（行 %d）", i+2)
				}
			}
		}
	}
	return nil
}

// セキュリティチェック
func (cg *CodeGenerator) performSecurityCheck(file GeneratedFile) error {
	content := strings.ToLower(file.Content)
	
	// 危険なパターンをチェック
	dangerousPatterns := []string{
		"eval(", "exec(", "system(", "shell_exec(", 
		"rm -rf", "delete from", "drop table",
		"password", "secret", "api_key", "token",
	}
	
	for _, pattern := range dangerousPatterns {
		if strings.Contains(content, pattern) {
			return fmt.Errorf("潜在的に危険なパターンを検出: %s", pattern)
		}
	}
	
	return nil
}

// 改善提案を生成
func (cg *CodeGenerator) generateImprovementSuggestions(ctx context.Context, request *CodeGenerationRequest, result *CodeGenerationResult) {
	// パフォーマンス改善提案
	result.Suggestions = append(result.Suggestions, GenerationSuggestion{
		Type:        "improvement",
		Priority:    "medium",
		Title:       "パフォーマンス最適化",
		Description: "生成されたコードのパフォーマンスを向上させる機会があります",
		Implementation: "アルゴリズムの最適化とメモリ使用量の削減を検討してください",
		Benefits:    "実行時間の短縮とリソース使用量の削減",
	})

	// テスト拡張提案
	if !request.TestRequired {
		result.Suggestions = append(result.Suggestions, GenerationSuggestion{
			Type:        "extension",
			Priority:    "high", 
			Title:       "テストケースの追加",
			Description: "生成されたコードに対する包括的なテストの作成を推奨します",
			Implementation: "単体テスト、統合テスト、エッジケースのテストを追加",
			Benefits:    "コードの信頼性と保守性の向上",
		})
	}
}

// 要約を生成
func (cg *CodeGenerator) generateSummary(ctx context.Context, result *CodeGenerationResult) error {
	summaryPrompt := fmt.Sprintf(`以下のコード生成結果の要約を日本語で作成してください：

生成されたファイル数: %d
変更されたファイル数: %d  
作成されたテスト数: %d
生成されたドキュメント数: %d
処理時間: %v

簡潔で実用的な要約を提供してください。`,
		len(result.GeneratedCode),
		len(result.ModifiedFiles),
		len(result.CreatedTests),
		len(result.Documentation),
		result.GenerationTime)

	response, err := cg.llmClient.GenerateResponse(ctx, &GenerateRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: summaryPrompt,
			},
		},
		Temperature: 0.3,
	})

	if err != nil {
		return err
	}

	result.Summary = response.Content
	return nil
}

// ユーティリティ関数群

// コードブロックを抽出
func (cg *CodeGenerator) extractCodeBlock(content string) string {
	// ```language から ``` までの部分を抽出
	start := strings.Index(content, "```")
	if start == -1 {
		return ""
	}
	
	// 最初の改行までスキップ
	start = strings.Index(content[start:], "\n") + start + 1
	if start <= 0 {
		return ""
	}
	
	end := strings.Index(content[start:], "```")
	if end == -1 {
		return content[start:]
	}
	
	return strings.TrimSpace(content[start : start+end])
}

// 依存関係を抽出
func (cg *CodeGenerator) extractDependencies(code, language string) []string {
	var deps []string
	
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		switch strings.ToLower(language) {
		case "go":
			if strings.HasPrefix(line, "import ") {
				deps = append(deps, line)
			}
		case "javascript", "typescript":
			if strings.HasPrefix(line, "import ") || (strings.HasPrefix(line, "const ") && strings.Contains(line, "require(")) {
				deps = append(deps, line)
			}
		case "python":
			if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "from ") {
				deps = append(deps, line)
			}
		}
	}
	
	return deps
}

// ファイルパスを生成
func (cg *CodeGenerator) generateFilePath(request *CodeGenerationRequest) string {
	if request.TargetFile != "" {
		return request.TargetFile
	}
	
	// デフォルトのファイル名を生成
	extension := cg.getFileExtension(request.Language)
	filename := strings.ReplaceAll(strings.ToLower(request.Description), " ", "_")
	filename = strings.ReplaceAll(filename, "-", "_")
	
	// 安全なファイル名に変換
	validChars := "abcdefghijklmnopqrstuvwxyz0123456789_"
	var safeName strings.Builder
	for _, char := range filename {
		if strings.ContainsRune(validChars, char) {
			safeName.WriteRune(char)
		}
	}
	
	return safeName.String() + extension
}

// テストファイルパスを生成
func (cg *CodeGenerator) generateTestFilePath(request *CodeGenerationRequest) string {
	basePath := cg.generateFilePath(request)
	extension := filepath.Ext(basePath)
	name := strings.TrimSuffix(basePath, extension)
	
	switch strings.ToLower(request.Language) {
	case "go":
		return name + "_test" + extension
	case "javascript", "typescript":
		return name + ".test" + extension
	case "python":
		return "test_" + name + extension
	default:
		return name + "_test" + extension
	}
}

// ファイル拡張子を取得
func (cg *CodeGenerator) getFileExtension(language string) string {
	extensions := map[string]string{
		"go":         ".go",
		"javascript": ".js",
		"typescript": ".ts",
		"python":     ".py",
		"java":       ".java",
		"rust":       ".rs",
		"c":          ".c",
		"cpp":        ".cpp",
		"c++":        ".cpp",
	}
	
	if ext, exists := extensions[strings.ToLower(language)]; exists {
		return ext
	}
	return ".txt"
}

// テストケースを抽出
func (cg *CodeGenerator) extractTestCases(code string) []string {
	var testCases []string
	
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Goのテスト関数
		if strings.HasPrefix(line, "func Test") && strings.Contains(line, "(") {
			testCases = append(testCases, line)
		}
		
		// JavaScriptのテスト
		if (strings.HasPrefix(line, "it(") || strings.HasPrefix(line, "test(")) && strings.Contains(line, "\"") {
			testCases = append(testCases, line)
		}
		
		// Pythonのテスト
		if strings.HasPrefix(line, "def test_") && strings.Contains(line, "(") {
			testCases = append(testCases, line)
		}
	}
	
	return testCases
}
