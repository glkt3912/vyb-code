package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

// 依存関係可視化結果
type DependencyVisualization struct {
	Nodes           []DependencyNode     `json:"nodes"`
	Edges           []DependencyEdge     `json:"edges"`
	Clusters        []DependencyCluster  `json:"clusters"`
	Metrics         VisualizationMetrics `json:"metrics"`
	Layout          LayoutConfig         `json:"layout"`
	GeneratedAt     time.Time            `json:"generated_at"`
	ProjectName     string               `json:"project_name"`
	TotalFiles      int                  `json:"total_files"`
	TotalDependencies int                `json:"total_dependencies"`
}

// 依存関係ノード
type DependencyNode struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	Type         string            `json:"type"`         // "file", "package", "module", "external"
	Category     string            `json:"category"`     // "core", "util", "test", "external"
	Size         int               `json:"size"`         // ファイルサイズ or 重要度
	Complexity   int               `json:"complexity"`   // 循環複雑度
	Position     Position          `json:"position"`
	Color        string            `json:"color"`
	Metadata     map[string]interface{} `json:"metadata"`
	Dependencies []string          `json:"dependencies"` // 直接依存関係のID
	Dependents   []string          `json:"dependents"`   // このノードに依存するもののID
}

// 依存関係エッジ
type DependencyEdge struct {
	ID         string            `json:"id"`
	Source     string            `json:"source"`     // ソースノードID
	Target     string            `json:"target"`     // ターゲットノードID
	Type       string            `json:"type"`       // "import", "call", "inheritance", "composition"
	Weight     float64           `json:"weight"`     // 依存度の重み
	Strength   string            `json:"strength"`   // "weak", "medium", "strong"
	Color      string            `json:"color"`
	Style      string            `json:"style"`      // "solid", "dashed", "dotted"
	Label      string            `json:"label,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// 依存関係クラスター
type DependencyCluster struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	NodeIDs     []string `json:"node_ids"`     // クラスター内のノードID
	Type        string   `json:"type"`         // "module", "layer", "feature", "external"
	Color       string   `json:"color"`
	Boundary    Boundary `json:"boundary"`     // クラスターの境界
}

// 境界定義
type Boundary struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// 位置情報
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// 可視化メトリクス
type VisualizationMetrics struct {
	Coupling              float64            `json:"coupling"`               // 結合度
	Cohesion              float64            `json:"cohesion"`               // 凝集度
	CyclomaticComplexity  int                `json:"cyclomatic_complexity"`  // 循環複雑度
	CircularDependencies  int                `json:"circular_dependencies"`  // 循環依存数
	MaxDepth              int                `json:"max_depth"`              // 最大依存深度
	CriticalPath          []string           `json:"critical_path"`          // 重要パス
	Hotspots              []string           `json:"hotspots"`               // ホットスポット
	IsolatedNodes         []string           `json:"isolated_nodes"`         // 孤立ノード
	CentralityScores      map[string]float64 `json:"centrality_scores"`      // 中心性スコア
	ModularityScore       float64            `json:"modularity_score"`       // モジュール性スコア
}

// レイアウト設定
type LayoutConfig struct {
	Algorithm   string            `json:"algorithm"`    // "force", "hierarchical", "circular", "grid"
	Direction   string            `json:"direction"`    // "top-to-bottom", "left-to-right"
	NodeSpacing float64           `json:"node_spacing"`
	EdgeLength  float64           `json:"edge_length"`
	Iterations  int               `json:"iterations"`
	Options     map[string]interface{} `json:"options"`
}

// プロジェクト洞察
type ProjectInsights struct {
	ArchitecturePatterns []ArchitecturePattern `json:"architecture_patterns"`
	DesignIssues         []DesignIssue         `json:"design_issues"`
	RefactoringOpportunities []RefactoringOpportunity `json:"refactoring_opportunities"`
	QualityAssessment    QualityAssessment     `json:"quality_assessment"`
	Recommendations      []Recommendation      `json:"recommendations"`
}

// アーキテクチャパターン
type ArchitecturePattern struct {
	Name        string  `json:"name"`
	Confidence  float64 `json:"confidence"`   // 0-1
	Description string  `json:"description"`
	Evidence    []string `json:"evidence"`
	Benefits    string  `json:"benefits"`
	Drawbacks   string  `json:"drawbacks"`
}

// 設計上の問題
type DesignIssue struct {
	Type        string   `json:"type"`        // "god_class", "circular_dependency", "tight_coupling"
	Severity    string   `json:"severity"`    // "low", "medium", "high", "critical"
	Nodes       []string `json:"nodes"`       // 関連ノード
	Description string   `json:"description"`
	Impact      string   `json:"impact"`
	Solution    string   `json:"solution"`
}

// リファクタリング機会
type RefactoringOpportunity struct {
	Type         string   `json:"type"`         // "extract_module", "merge_modules", "split_class"
	Priority     string   `json:"priority"`     // "low", "medium", "high"
	Nodes        []string `json:"nodes"`        // 対象ノード
	Description  string   `json:"description"`
	Benefits     string   `json:"benefits"`
	Effort       string   `json:"effort"`       // "small", "medium", "large"
	Prerequisites []string `json:"prerequisites"`
}

// 品質評価
type QualityAssessment struct {
	OverallScore      int                    `json:"overall_score"`       // 0-100
	ArchitectureScore int                    `json:"architecture_score"`  // 0-100
	MaintainabilityScore int                 `json:"maintainability_score"` // 0-100
	ModularityScore   int                    `json:"modularity_score"`    // 0-100
	TestabilityScore  int                    `json:"testability_score"`   // 0-100
	DetailedMetrics   map[string]interface{} `json:"detailed_metrics"`
}

// 推奨事項
type Recommendation struct {
	Category    string `json:"category"`    // "architecture", "performance", "maintainability"
	Priority    string `json:"priority"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ActionItems []string `json:"action_items"`
	Benefits    string `json:"benefits"`
	Resources   []string `json:"resources"`
}

// 依存関係可視化器
type DependencyVisualizer struct {
	constraints *security.Constraints
	projectDir  string
	config      *VisualizationConfig
}

// 可視化設定
type VisualizationConfig struct {
	MaxNodes        int      `json:"max_nodes"`
	MaxDepth        int      `json:"max_depth"`
	IncludeExternal bool     `json:"include_external"`
	ExcludePatterns []string `json:"exclude_patterns"`
	LayoutAlgorithm string   `json:"layout_algorithm"`
	ColorScheme     string   `json:"color_scheme"`
	GroupingStrategy string  `json:"grouping_strategy"` // "by_directory", "by_type", "by_layer"
}

// 依存関係可視化器を作成
func NewDependencyVisualizer(constraints *security.Constraints, projectDir string) *DependencyVisualizer {
	return &DependencyVisualizer{
		constraints: constraints,
		projectDir:  projectDir,
		config: &VisualizationConfig{
			MaxNodes:        200,
			MaxDepth:        10,
			IncludeExternal: false,
			ExcludePatterns: []string{"**/node_modules/**", "**/vendor/**", "**/.git/**", "**/test/**"},
			LayoutAlgorithm: "force",
			ColorScheme:     "category10",
			GroupingStrategy: "by_directory",
		},
	}
}

// 設定を更新
func (dv *DependencyVisualizer) UpdateConfig(config *VisualizationConfig) {
	if config != nil {
		dv.config = config
	}
}

// プロジェクトの依存関係を可視化
func (dv *DependencyVisualizer) VisualizeProject(ctx context.Context) (*DependencyVisualization, error) {
	startTime := time.Now()

	// プロジェクトディレクトリの存在確認
	if _, err := os.Stat(dv.projectDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("プロジェクトディレクトリが存在しません: %s", dv.projectDir)
	}

	result := &DependencyVisualization{
		Nodes:         []DependencyNode{},
		Edges:         []DependencyEdge{},
		Clusters:      []DependencyCluster{},
		GeneratedAt:   startTime,
		ProjectName:   filepath.Base(dv.projectDir),
		Layout: LayoutConfig{
			Algorithm:   dv.config.LayoutAlgorithm,
			Direction:   "top-to-bottom",
			NodeSpacing: 100.0,
			EdgeLength:  150.0,
			Iterations:  1000,
			Options:     make(map[string]interface{}),
		},
	}

	// ファイルを収集
	files, err := dv.collectAnalysisFiles()
	if err != nil {
		return nil, fmt.Errorf("ファイル収集エラー: %w", err)
	}

	result.TotalFiles = len(files)

	// ノードを作成
	err = dv.createNodes(files, result)
	if err != nil {
		return nil, fmt.Errorf("ノード作成エラー: %w", err)
	}

	// エッジを作成（依存関係を解析）
	err = dv.createEdges(files, result)
	if err != nil {
		return nil, fmt.Errorf("エッジ作成エラー: %w", err)
	}

	result.TotalDependencies = len(result.Edges)

	// クラスターを作成
	err = dv.createClusters(result)
	if err != nil {
		return nil, fmt.Errorf("クラスター作成エラー: %w", err)
	}

	// レイアウトを計算
	err = dv.calculateLayout(result)
	if err != nil {
		return nil, fmt.Errorf("レイアウト計算エラー: %w", err)
	}

	// メトリクスを計算
	dv.calculateMetrics(result)

	return result, nil
}

// 分析ファイルを収集
func (dv *DependencyVisualizer) collectAnalysisFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(dv.projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // スキップ
		}

		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(dv.projectDir, path)

		// 除外パターンをチェック
		if dv.shouldExcludeFile(relPath) {
			return nil
		}

		// 分析対象ファイルかチェック
		if dv.isAnalyzableFile(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// ファイルを除外するかチェック
func (dv *DependencyVisualizer) shouldExcludeFile(relPath string) bool {
	for _, pattern := range dv.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
		// 簡易的なグロブマッチング
		if strings.Contains(pattern, "**") {
			parts := strings.Split(pattern, "**")
			if len(parts) == 2 {
				prefix := parts[0]
				suffix := parts[1]
				if (prefix == "" || strings.HasPrefix(relPath, prefix)) &&
				   (suffix == "" || strings.HasSuffix(relPath, suffix)) {
					return true
				}
			}
		}
	}
	return false
}

// 分析可能ファイルかチェック
func (dv *DependencyVisualizer) isAnalyzableFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	analyzableExtensions := []string{
		".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".java", 
		".cpp", ".c", ".h", ".hpp", ".rs", ".rb", ".php", ".cs",
	}

	for _, analyzableExt := range analyzableExtensions {
		if ext == analyzableExt {
			return true
		}
	}
	return false
}

// ノードを作成
func (dv *DependencyVisualizer) createNodes(files []string, result *DependencyVisualization) error {
	for _, filePath := range files {
		relPath, _ := filepath.Rel(dv.projectDir, filePath)
		
		// ファイル情報を取得
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		// ノードを作成
		node := DependencyNode{
			ID:           dv.generateNodeID(relPath),
			Label:        filepath.Base(relPath),
			Type:         "file",
			Category:     dv.categorizeFile(relPath),
			Size:         int(info.Size()),
			Complexity:   dv.calculateFileComplexity(filePath),
			Position:     Position{X: 0, Y: 0}, // レイアウト計算で設定
			Color:        dv.getNodeColor(dv.categorizeFile(relPath)),
			Metadata:     make(map[string]interface{}),
			Dependencies: []string{},
			Dependents:   []string{},
		}

		// メタデータを追加
		node.Metadata["full_path"] = relPath
		node.Metadata["file_extension"] = filepath.Ext(relPath)
		node.Metadata["directory"] = filepath.Dir(relPath)
		node.Metadata["language"] = dv.detectLanguage(filePath)

		result.Nodes = append(result.Nodes, node)
	}

	// パッケージやモジュールレベルのノードも作成
	if dv.config.IncludeExternal {
		dv.createExternalNodes(files, result)
	}

	return nil
}

// ノードIDを生成
func (dv *DependencyVisualizer) generateNodeID(path string) string {
	// パスを正規化してIDとして使用
	normalized := strings.ReplaceAll(path, "/", "_")
	normalized = strings.ReplaceAll(normalized, "\\", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return normalized
}

// ファイルを分類
func (dv *DependencyVisualizer) categorizeFile(path string) string {
	path = strings.ToLower(path)
	
	if strings.Contains(path, "test") || strings.Contains(path, "spec") {
		return "test"
	}
	if strings.Contains(path, "config") || strings.Contains(path, "setting") {
		return "config"
	}
	if strings.Contains(path, "util") || strings.Contains(path, "helper") || strings.Contains(path, "common") {
		return "util"
	}
	if strings.Contains(path, "main") || strings.Contains(path, "cmd") || strings.Contains(path, "entry") {
		return "core"
	}
	if strings.Contains(path, "internal") || strings.Contains(path, "lib") {
		return "core"
	}
	
	return "application"
}

// ファイル複雑度を計算
func (dv *DependencyVisualizer) calculateFileComplexity(filePath string) int {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}

	contentStr := string(content)
	
	// 簡易的な複雑度計算
	complexity := 1 // 基本複雑度

	// 制御構造をカウント
	keywords := []string{"if", "else", "for", "while", "switch", "case", "try", "catch"}
	for _, keyword := range keywords {
		complexity += strings.Count(strings.ToLower(contentStr), keyword+" ")
	}

	// 関数定義をカウント
	funcKeywords := []string{"func ", "function ", "def ", "public ", "private "}
	for _, keyword := range funcKeywords {
		complexity += strings.Count(strings.ToLower(contentStr), keyword)
	}

	return complexity
}

// ノードの色を取得
func (dv *DependencyVisualizer) getNodeColor(category string) string {
	colors := map[string]string{
		"core":        "#ff6b6b",
		"application": "#4ecdc4", 
		"util":        "#45b7d1",
		"test":        "#96ceb4",
		"config":      "#feca57",
		"external":    "#ff9ff3",
	}

	if color, exists := colors[category]; exists {
		return color
	}
	return "#c7c7c7"
}

// 言語を検出
func (dv *DependencyVisualizer) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	languages := map[string]string{
		".go":   "Go",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".jsx":  "JavaScript",
		".tsx":  "TypeScript",
		".py":   "Python",
		".java": "Java",
		".cpp":  "C++",
		".c":    "C",
		".rs":   "Rust",
		".rb":   "Ruby",
		".php":  "PHP",
		".cs":   "C#",
	}

	if lang, exists := languages[ext]; exists {
		return lang
	}
	return "Unknown"
}

// 外部ノードを作成
func (dv *DependencyVisualizer) createExternalNodes(files []string, result *DependencyVisualization) error {
	externalDeps := make(map[string]bool)

	// 各ファイルから外部依存関係を抽出
	for _, filePath := range files {
		deps, err := dv.extractExternalDependencies(filePath)
		if err != nil {
			continue
		}

		for _, dep := range deps {
			externalDeps[dep] = true
		}
	}

	// 外部依存関係のノードを作成
	for dep := range externalDeps {
		node := DependencyNode{
			ID:       "ext_" + dv.generateNodeID(dep),
			Label:    dep,
			Type:     "external",
			Category: "external",
			Size:     50, // 外部依存は固定サイズ
			Color:    dv.getNodeColor("external"),
			Metadata: map[string]interface{}{
				"package_name": dep,
				"is_external":  true,
			},
		}
		result.Nodes = append(result.Nodes, node)
	}

	return nil
}

// 外部依存関係を抽出
func (dv *DependencyVisualizer) extractExternalDependencies(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var deps []string
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	ext := strings.ToLower(filepath.Ext(filePath))
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		switch ext {
		case ".go":
			if strings.HasPrefix(line, "import ") {
				dep := dv.extractGoImport(line)
				if dep != "" && !dv.isInternalPackage(dep) {
					deps = append(deps, dep)
				}
			}
		case ".js", ".ts", ".jsx", ".tsx":
			if strings.HasPrefix(line, "import ") || (strings.HasPrefix(line, "const ") && strings.Contains(line, "require(")) {
				dep := dv.extractJSImport(line)
				if dep != "" && !dv.isInternalPath(dep) {
					deps = append(deps, dep)
				}
			}
		case ".py":
			if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "from ") {
				dep := dv.extractPythonImport(line)
				if dep != "" && !dv.isInternalPythonModule(dep) {
					deps = append(deps, dep)
				}
			}
		}
	}

	return deps, nil
}

// Go import を抽出
func (dv *DependencyVisualizer) extractGoImport(line string) string {
	// import "package" または import ("package1", "package2") 形式を処理
	if strings.Contains(line, "\"") {
		start := strings.Index(line, "\"")
		end := strings.LastIndex(line, "\"")
		if start != -1 && end != -1 && start < end {
			return line[start+1 : end]
		}
	}
	return ""
}

// JavaScript/TypeScript import を抽出
func (dv *DependencyVisualizer) extractJSImport(line string) string {
	// import ... from 'package' または require('package') 形式を処理
	if strings.Contains(line, "from") {
		parts := strings.Split(line, "from")
		if len(parts) > 1 {
			importPart := strings.TrimSpace(parts[1])
			importPart = strings.Trim(importPart, "\"';")
			return importPart
		}
	}
	if strings.Contains(line, "require(") {
		start := strings.Index(line, "require(") + 8
		end := strings.Index(line[start:], ")") + start
		if start < len(line) && end > start {
			importPart := line[start:end]
			importPart = strings.Trim(importPart, "\"'")
			return importPart
		}
	}
	return ""
}

// Python import を抽出
func (dv *DependencyVisualizer) extractPythonImport(line string) string {
	if strings.HasPrefix(line, "import ") {
		parts := strings.Fields(line)
		if len(parts) > 1 {
			return parts[1]
		}
	}
	if strings.HasPrefix(line, "from ") {
		parts := strings.Fields(line)
		if len(parts) > 1 {
			return parts[1]
		}
	}
	return ""
}

// 内部パッケージかチェック（Go）
func (dv *DependencyVisualizer) isInternalPackage(pkg string) bool {
	// プロジェクトのモジュール名で始まるかチェック
	goModPath := filepath.Join(dv.projectDir, "go.mod")
	if content, err := os.ReadFile(goModPath); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "module ") {
				module := strings.TrimPrefix(line, "module ")
				module = strings.TrimSpace(module)
				if strings.HasPrefix(pkg, module) {
					return true
				}
			}
		}
	}
	
	// 相対パスまたは標準ライブラリ
	return strings.HasPrefix(pkg, "./") || strings.HasPrefix(pkg, "../") || 
		   !strings.Contains(pkg, ".") || strings.Contains(pkg, "/")
}

// 内部パス かチェック（JS/TS）
func (dv *DependencyVisualizer) isInternalPath(path string) bool {
	return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") || strings.HasPrefix(path, "/")
}

// 内部Pythonモジュールかチェック
func (dv *DependencyVisualizer) isInternalPythonModule(module string) bool {
	// 標準ライブラリの簡易チェック
	stdLibs := []string{"os", "sys", "json", "time", "datetime", "re", "collections", "itertools"}
	for _, std := range stdLibs {
		if module == std || strings.HasPrefix(module, std+".") {
			return true
		}
	}
	
	// 相対import
	return strings.HasPrefix(module, ".")
}

// エッジを作成
func (dv *DependencyVisualizer) createEdges(files []string, result *DependencyVisualization) error {
	nodeMap := make(map[string]*DependencyNode)
	for i := range result.Nodes {
		nodeMap[result.Nodes[i].ID] = &result.Nodes[i]
	}

	edgeID := 0
	for _, filePath := range files {
		relPath, _ := filepath.Rel(dv.projectDir, filePath)
		sourceID := dv.generateNodeID(relPath)
		
		// ファイルの依存関係を分析
		deps, err := dv.analyzeDependencies(filePath, files)
		if err != nil {
			continue
		}

		for _, dep := range deps {
			targetID := dv.generateNodeID(dep.Path)
			
			// ノードが存在するかチェック
			if _, exists := nodeMap[targetID]; !exists {
				continue
			}

			// エッジを作成
			edge := DependencyEdge{
				ID:       fmt.Sprintf("edge_%d", edgeID),
				Source:   sourceID,
				Target:   targetID,
				Type:     dep.Type,
				Weight:   dep.Weight,
				Strength: dv.calculateDependencyStrength(dep.Weight),
				Color:    dv.getEdgeColor(dep.Type),
				Style:    dv.getEdgeStyle(dep.Type),
				Metadata: map[string]interface{}{
					"relationship": dep.Type,
					"confidence":   dep.Weight,
				},
			}

			result.Edges = append(result.Edges, edge)
			
			// ノードに依存関係情報を追加
			nodeMap[sourceID].Dependencies = append(nodeMap[sourceID].Dependencies, targetID)
			nodeMap[targetID].Dependents = append(nodeMap[targetID].Dependents, sourceID)
			
			edgeID++
		}
	}

	return nil
}

// 依存関係情報
type FileDependency struct {
	Path   string
	Type   string  // "import", "call", "reference"
	Weight float64 // 依存の強さ
}

// 依存関係を分析
func (dv *DependencyVisualizer) analyzeDependencies(filePath string, allFiles []string) ([]FileDependency, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var deps []FileDependency
	contentStr := string(content)
	
	// 他のファイルに対する参照を検索
	for _, otherFile := range allFiles {
		if otherFile == filePath {
			continue
		}

		otherRel, _ := filepath.Rel(dv.projectDir, otherFile)
		baseName := strings.TrimSuffix(filepath.Base(otherFile), filepath.Ext(otherFile))
		
		// ファイル名や関数名の参照を検索
		if strings.Contains(contentStr, baseName) {
			weight := dv.calculateReferenceWeight(contentStr, baseName)
			if weight > 0 {
				deps = append(deps, FileDependency{
					Path:   otherRel,
					Type:   "reference",
					Weight: weight,
				})
			}
		}

		// import文での直接的な依存関係
		if dv.hasDirectImport(contentStr, otherRel, filePath) {
			deps = append(deps, FileDependency{
				Path:   otherRel,
				Type:   "import",
				Weight: 1.0,
			})
		}
	}

	return deps, nil
}

// 参照の重みを計算
func (dv *DependencyVisualizer) calculateReferenceWeight(content, target string) float64 {
	count := float64(strings.Count(content, target))
	if count == 0 {
		return 0
	}
	
	// 出現回数に基づいて重みを計算（最大1.0）
	weight := count / 10.0
	if weight > 1.0 {
		weight = 1.0
	}
	
	return weight
}

// 直接importがあるかチェック
func (dv *DependencyVisualizer) hasDirectImport(content, targetPath, sourcePath string) bool {
	ext := strings.ToLower(filepath.Ext(sourcePath))
	
	switch ext {
	case ".go":
		// Go の場合、相対パスでのimportをチェック
		targetDir := filepath.Dir(targetPath)
		sourceDir := filepath.Dir(sourcePath)
		
		if targetDir != sourceDir {
			relativePath, _ := filepath.Rel(sourceDir, targetDir)
			return strings.Contains(content, fmt.Sprintf("\"%s\"", relativePath))
		}
		return false
		
	case ".js", ".ts", ".jsx", ".tsx":
		// JavaScript/TypeScript の場合
		relativePath, _ := filepath.Rel(filepath.Dir(sourcePath), targetPath)
		relativePath = strings.TrimSuffix(relativePath, filepath.Ext(relativePath))
		
		if !strings.HasPrefix(relativePath, ".") {
			relativePath = "./" + relativePath
		}
		
		return strings.Contains(content, fmt.Sprintf("'%s'", relativePath)) ||
			   strings.Contains(content, fmt.Sprintf("\"%s\"", relativePath))
			   
	case ".py":
		// Python の場合
		targetModule := strings.ReplaceAll(strings.TrimSuffix(targetPath, ".py"), "/", ".")
		return strings.Contains(content, fmt.Sprintf("import %s", targetModule)) ||
			   strings.Contains(content, fmt.Sprintf("from %s", targetModule))
	}
	
	return false
}

// 依存関係の強度を計算
func (dv *DependencyVisualizer) calculateDependencyStrength(weight float64) string {
	if weight >= 0.8 {
		return "strong"
	} else if weight >= 0.4 {
		return "medium"
	} else {
		return "weak"
	}
}

// エッジの色を取得
func (dv *DependencyVisualizer) getEdgeColor(depType string) string {
	colors := map[string]string{
		"import":    "#2d3436",
		"call":      "#0984e3",
		"reference": "#6c5ce7",
		"inheritance": "#e84393",
	}

	if color, exists := colors[depType]; exists {
		return color
	}
	return "#636e72"
}

// エッジのスタイルを取得
func (dv *DependencyVisualizer) getEdgeStyle(depType string) string {
	styles := map[string]string{
		"import":    "solid",
		"call":      "solid",
		"reference": "dashed",
		"inheritance": "dotted",
	}

	if style, exists := styles[depType]; exists {
		return style
	}
	return "solid"
}

// クラスターを作成
func (dv *DependencyVisualizer) createClusters(result *DependencyVisualization) error {
	switch dv.config.GroupingStrategy {
	case "by_directory":
		return dv.createDirectoryClusters(result)
	case "by_type":
		return dv.createTypeClusters(result) 
	case "by_layer":
		return dv.createLayerClusters(result)
	default:
		return dv.createDirectoryClusters(result)
	}
}

// ディレクトリベースのクラスターを作成
func (dv *DependencyVisualizer) createDirectoryClusters(result *DependencyVisualization) error {
	dirGroups := make(map[string][]string)
	
	for _, node := range result.Nodes {
		if node.Type == "file" {
			if fullPath, ok := node.Metadata["full_path"].(string); ok {
				dir := filepath.Dir(fullPath)
				if dir == "." {
					dir = "root"
				}
				dirGroups[dir] = append(dirGroups[dir], node.ID)
			}
		}
	}
	
	clusterID := 0
	for dir, nodeIDs := range dirGroups {
		if len(nodeIDs) > 1 { // 複数ファイルがある場合のみクラスター化
			cluster := DependencyCluster{
				ID:          fmt.Sprintf("cluster_%d", clusterID),
				Label:       dir,
				Description: fmt.Sprintf("Directory: %s", dir),
				NodeIDs:     nodeIDs,
				Type:        "module",
				Color:       dv.getClusterColor("module"),
			}
			result.Clusters = append(result.Clusters, cluster)
			clusterID++
		}
	}
	
	return nil
}

// タイプベースのクラスターを作成
func (dv *DependencyVisualizer) createTypeClusters(result *DependencyVisualization) error {
	typeGroups := make(map[string][]string)
	
	for _, node := range result.Nodes {
		typeGroups[node.Category] = append(typeGroups[node.Category], node.ID)
	}
	
	clusterID := 0
	for category, nodeIDs := range typeGroups {
		if len(nodeIDs) > 1 {
			cluster := DependencyCluster{
				ID:          fmt.Sprintf("cluster_%d", clusterID),
				Label:       category,
				Description: fmt.Sprintf("Category: %s", category),
				NodeIDs:     nodeIDs,
				Type:        "layer",
				Color:       dv.getClusterColor(category),
			}
			result.Clusters = append(result.Clusters, cluster)
			clusterID++
		}
	}
	
	return nil
}

// レイヤーベースのクラスターを作成
func (dv *DependencyVisualizer) createLayerClusters(result *DependencyVisualization) error {
	// アーキテクチャレイヤーに基づいたクラスタリング
	layerGroups := map[string][]string{
		"presentation": {},
		"application":  {},
		"domain":       {},
		"infrastructure": {},
	}
	
	for _, node := range result.Nodes {
		if fullPath, ok := node.Metadata["full_path"].(string); ok {
			layer := dv.determineArchitecturalLayer(fullPath)
			layerGroups[layer] = append(layerGroups[layer], node.ID)
		}
	}
	
	clusterID := 0
	for layer, nodeIDs := range layerGroups {
		if len(nodeIDs) > 0 {
			cluster := DependencyCluster{
				ID:          fmt.Sprintf("layer_cluster_%d", clusterID),
				Label:       layer,
				Description: fmt.Sprintf("Architectural Layer: %s", layer),
				NodeIDs:     nodeIDs,
				Type:        "layer",
				Color:       dv.getClusterColor(layer),
			}
			result.Clusters = append(result.Clusters, cluster)
			clusterID++
		}
	}
	
	return nil
}

// アーキテクチャレイヤーを決定
func (dv *DependencyVisualizer) determineArchitecturalLayer(path string) string {
	path = strings.ToLower(path)
	
	if strings.Contains(path, "ui") || strings.Contains(path, "view") || strings.Contains(path, "controller") {
		return "presentation"
	}
	if strings.Contains(path, "service") || strings.Contains(path, "usecase") || strings.Contains(path, "application") {
		return "application"
	}
	if strings.Contains(path, "domain") || strings.Contains(path, "entity") || strings.Contains(path, "model") {
		return "domain"
	}
	if strings.Contains(path, "repository") || strings.Contains(path, "dao") || strings.Contains(path, "infrastructure") {
		return "infrastructure"
	}
	
	return "application" // デフォルト
}

// クラスターの色を取得
func (dv *DependencyVisualizer) getClusterColor(clusterType string) string {
	colors := map[string]string{
		"module":         "#e9ecef",
		"layer":          "#f8f9fa",
		"core":           "#ffebee",
		"application":    "#e3f2fd",
		"util":           "#f3e5f5",
		"test":           "#e8f5e8",
		"presentation":   "#fff3e0",
		"domain":         "#fce4ec",
		"infrastructure": "#e0f2f1",
	}

	if color, exists := colors[clusterType]; exists {
		return color
	}
	return "#f5f5f5"
}

// レイアウトを計算
func (dv *DependencyVisualizer) calculateLayout(result *DependencyVisualization) error {
	switch result.Layout.Algorithm {
	case "force":
		return dv.calculateForceLayout(result)
	case "hierarchical":
		return dv.calculateHierarchicalLayout(result)
	case "circular":
		return dv.calculateCircularLayout(result)
	default:
		return dv.calculateForceLayout(result)
	}
}

// Force-directed レイアウト計算
func (dv *DependencyVisualizer) calculateForceLayout(result *DependencyVisualization) error {
	// 簡易的なforce-directedアルゴリズム
	nodeCount := len(result.Nodes)
	if nodeCount == 0 {
		return nil
	}

	// 初期位置をランダムに配置
	for i := range result.Nodes {
		angle := float64(i) * 2 * 3.14159 / float64(nodeCount)
		radius := 200.0
		result.Nodes[i].Position.X = radius * (1 + 0.5*float64(i)/float64(nodeCount)) * math.Cos(angle)
		result.Nodes[i].Position.Y = radius * (1 + 0.5*float64(i)/float64(nodeCount)) * math.Sin(angle)
	}

	// エッジマップを作成
	edgeMap := make(map[string][]string)
	for _, edge := range result.Edges {
		edgeMap[edge.Source] = append(edgeMap[edge.Source], edge.Target)
		edgeMap[edge.Target] = append(edgeMap[edge.Target], edge.Source)
	}

	// 簡易的な力による位置調整（実際の実装ではより複雑なアルゴリズムを使用）
	for iteration := 0; iteration < 100; iteration++ {
		for i := range result.Nodes {
			// 隣接ノードに基づいて位置を調整
			if neighbors, exists := edgeMap[result.Nodes[i].ID]; exists {
				avgX, avgY := 0.0, 0.0
				count := 0
				
				for j := range result.Nodes {
					for _, neighborID := range neighbors {
						if result.Nodes[j].ID == neighborID {
							avgX += result.Nodes[j].Position.X
							avgY += result.Nodes[j].Position.Y
							count++
						}
					}
				}
				
				if count > 0 {
					avgX /= float64(count)
					avgY /= float64(count)
					
					// 隣接ノードの重心に向かって移動（減衰係数付き）
					alpha := 0.1
					result.Nodes[i].Position.X += alpha * (avgX - result.Nodes[i].Position.X)
					result.Nodes[i].Position.Y += alpha * (avgY - result.Nodes[i].Position.Y)
				}
			}
		}
	}

	return nil
}

// 階層レイアウト計算
func (dv *DependencyVisualizer) calculateHierarchicalLayout(result *DependencyVisualization) error {
	// トポロジカルソートに基づく階層レイアウト
	levels := dv.calculateNodeLevels(result)
	
	levelGroups := make(map[int][]int)
	for nodeIndex, level := range levels {
		levelGroups[level] = append(levelGroups[level], nodeIndex)
	}

	levelHeight := 150.0
	nodeSpacing := 120.0

	for level, nodeIndices := range levelGroups {
		startX := -float64(len(nodeIndices)-1) * nodeSpacing / 2
		for i, nodeIndex := range nodeIndices {
			result.Nodes[nodeIndex].Position.X = startX + float64(i)*nodeSpacing
			result.Nodes[nodeIndex].Position.Y = float64(level) * levelHeight
		}
	}

	return nil
}

// 円形レイアウト計算
func (dv *DependencyVisualizer) calculateCircularLayout(result *DependencyVisualization) error {
	nodeCount := len(result.Nodes)
	if nodeCount == 0 {
		return nil
	}

	radius := 300.0
	for i := range result.Nodes {
		angle := float64(i) * 2 * 3.14159 / float64(nodeCount)
		result.Nodes[i].Position.X = radius * math.Cos(angle)
		result.Nodes[i].Position.Y = radius * math.Sin(angle)
	}

	return nil
}

// ノードレベルを計算（階層用）
func (dv *DependencyVisualizer) calculateNodeLevels(result *DependencyVisualization) []int {
	nodeCount := len(result.Nodes)
	levels := make([]int, nodeCount)
	
	// ノードIDからインデックスへのマップ
	nodeIndexMap := make(map[string]int)
	for i, node := range result.Nodes {
		nodeIndexMap[node.ID] = i
	}
	
	// 入次数を計算
	inDegree := make([]int, nodeCount)
	for _, edge := range result.Edges {
		if targetIndex, exists := nodeIndexMap[edge.Target]; exists {
			inDegree[targetIndex]++
		}
	}
	
	// BFSでレベルを計算
	queue := []int{}
	for i, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, i)
			levels[i] = 0
		}
	}
	
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		
		// 隣接ノードを処理
		for _, edge := range result.Edges {
			if sourceIndex, exists := nodeIndexMap[edge.Source]; exists && sourceIndex == current {
				if targetIndex, exists := nodeIndexMap[edge.Target]; exists {
					inDegree[targetIndex]--
					if inDegree[targetIndex] == 0 {
						levels[targetIndex] = levels[current] + 1
						queue = append(queue, targetIndex)
					}
				}
			}
		}
	}
	
	return levels
}

// メトリクスを計算
func (dv *DependencyVisualizer) calculateMetrics(result *DependencyVisualization) {
	metrics := &VisualizationMetrics{
		CentralityScores: make(map[string]float64),
	}

	// 基本メトリクス
	metrics.CircularDependencies = dv.detectCircularDependencies(result)
	metrics.MaxDepth = dv.calculateMaxDepth(result)
	metrics.CriticalPath = dv.findCriticalPath(result)
	metrics.Hotspots = dv.identifyHotspots(result)
	metrics.IsolatedNodes = dv.findIsolatedNodes(result)

	// 中心性スコアを計算
	dv.calculateCentralityScores(result, metrics)

	// 結合度と凝集度
	metrics.Coupling = dv.calculateCoupling(result)
	metrics.Cohesion = dv.calculateCohesion(result)
	
	// モジュール性スコア
	metrics.ModularityScore = dv.calculateModularityScore(result)

	result.Metrics = *metrics
}

// 循環依存を検出
func (dv *DependencyVisualizer) detectCircularDependencies(result *DependencyVisualization) int {
	// DFS で循環を検出
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	cycles := 0
	
	var dfs func(string) bool
	dfs = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true
		
		// 隣接ノードを探索
		for _, edge := range result.Edges {
			if edge.Source == nodeID {
				if !visited[edge.Target] {
					if dfs(edge.Target) {
						cycles++
						return true
					}
				} else if recStack[edge.Target] {
					cycles++
					return true
				}
			}
		}
		
		recStack[nodeID] = false
		return false
	}
	
	for _, node := range result.Nodes {
		if !visited[node.ID] {
			dfs(node.ID)
		}
	}
	
	return cycles
}

// 最大深度を計算
func (dv *DependencyVisualizer) calculateMaxDepth(result *DependencyVisualization) int {
	maxDepth := 0
	
	// 各ノードから開始してDFSで最大深度を計算
	for _, startNode := range result.Nodes {
		depth := dv.calculateDepthFromNode(result, startNode.ID, make(map[string]bool))
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	
	return maxDepth
}

// 特定のノードからの深度を計算
func (dv *DependencyVisualizer) calculateDepthFromNode(result *DependencyVisualization, nodeID string, visited map[string]bool) int {
	if visited[nodeID] {
		return 0 // 循環を避ける
	}
	
	visited[nodeID] = true
	maxChildDepth := 0
	
	for _, edge := range result.Edges {
		if edge.Source == nodeID {
			childDepth := dv.calculateDepthFromNode(result, edge.Target, visited)
			if childDepth > maxChildDepth {
				maxChildDepth = childDepth
			}
		}
	}
	
	delete(visited, nodeID)
	return maxChildDepth + 1
}

// クリティカルパスを見つける
func (dv *DependencyVisualizer) findCriticalPath(result *DependencyVisualization) []string {
	// 最も長い依存チェーンを見つける
	longestPath := []string{}
	
	for _, startNode := range result.Nodes {
		path := dv.findLongestPath(result, startNode.ID, make(map[string]bool))
		if len(path) > len(longestPath) {
			longestPath = path
		}
	}
	
	return longestPath
}

// 最長パスを見つける
func (dv *DependencyVisualizer) findLongestPath(result *DependencyVisualization, nodeID string, visited map[string]bool) []string {
	if visited[nodeID] {
		return []string{}
	}
	
	visited[nodeID] = true
	longestChildPath := []string{}
	
	for _, edge := range result.Edges {
		if edge.Source == nodeID {
			childPath := dv.findLongestPath(result, edge.Target, visited)
			if len(childPath) > len(longestChildPath) {
				longestChildPath = childPath
			}
		}
	}
	
	delete(visited, nodeID)
	return append([]string{nodeID}, longestChildPath...)
}

// ホットスポットを特定
func (dv *DependencyVisualizer) identifyHotspots(result *DependencyVisualization) []string {
	degreeMap := make(map[string]int)
	
	// 各ノードの次数を計算
	for _, edge := range result.Edges {
		degreeMap[edge.Source]++
		degreeMap[edge.Target]++
	}
	
	var hotspots []string
	threshold := len(result.Edges) / len(result.Nodes) * 2 // 平均の2倍以上
	
	for nodeID, degree := range degreeMap {
		if degree >= threshold {
			hotspots = append(hotspots, nodeID)
		}
	}
	
	return hotspots
}

// 孤立ノードを見つける
func (dv *DependencyVisualizer) findIsolatedNodes(result *DependencyVisualization) []string {
	connected := make(map[string]bool)
	
	for _, edge := range result.Edges {
		connected[edge.Source] = true
		connected[edge.Target] = true
	}
	
	var isolated []string
	for _, node := range result.Nodes {
		if !connected[node.ID] {
			isolated = append(isolated, node.ID)
		}
	}
	
	return isolated
}

// 中心性スコアを計算
func (dv *DependencyVisualizer) calculateCentralityScores(result *DependencyVisualization, metrics *VisualizationMetrics) {
	// 次数中心性を計算
	for _, node := range result.Nodes {
		degree := 0
		for _, edge := range result.Edges {
			if edge.Source == node.ID || edge.Target == node.ID {
				degree++
			}
		}
		metrics.CentralityScores[node.ID] = float64(degree) / float64(len(result.Nodes)-1)
	}
}

// 結合度を計算
func (dv *DependencyVisualizer) calculateCoupling(result *DependencyVisualization) float64 {
	if len(result.Nodes) <= 1 {
		return 0.0
	}
	
	maxPossibleEdges := len(result.Nodes) * (len(result.Nodes) - 1)
	return float64(len(result.Edges)) / float64(maxPossibleEdges)
}

// 凝集度を計算
func (dv *DependencyVisualizer) calculateCohesion(result *DependencyVisualization) float64 {
	if len(result.Clusters) == 0 {
		return 0.0
	}
	
	totalInternalEdges := 0
	totalPossibleInternalEdges := 0
	
	for _, cluster := range result.Clusters {
		internalEdges := 0
		possibleInternalEdges := len(cluster.NodeIDs) * (len(cluster.NodeIDs) - 1)
		
		for _, edge := range result.Edges {
			sourceInCluster := false
			targetInCluster := false
			
			for _, nodeID := range cluster.NodeIDs {
				if edge.Source == nodeID {
					sourceInCluster = true
				}
				if edge.Target == nodeID {
					targetInCluster = true
				}
			}
			
			if sourceInCluster && targetInCluster {
				internalEdges++
			}
		}
		
		totalInternalEdges += internalEdges
		totalPossibleInternalEdges += possibleInternalEdges
	}
	
	if totalPossibleInternalEdges == 0 {
		return 0.0
	}
	
	return float64(totalInternalEdges) / float64(totalPossibleInternalEdges)
}

// モジュール性スコアを計算
func (dv *DependencyVisualizer) calculateModularityScore(result *DependencyVisualization) float64 {
	// 簡易的なモジュール性計算
	if len(result.Clusters) == 0 {
		return 0.0
	}
	
	// クラスター内のエッジ数 vs クラスター間のエッジ数
	intraClusterEdges := 0
	interClusterEdges := 0
	
	for _, edge := range result.Edges {
		sourceCluster := dv.findNodeCluster(edge.Source, result)
		targetCluster := dv.findNodeCluster(edge.Target, result)
		
		if sourceCluster == targetCluster && sourceCluster != "" {
			intraClusterEdges++
		} else {
			interClusterEdges++
		}
	}
	
	totalEdges := intraClusterEdges + interClusterEdges
	if totalEdges == 0 {
		return 0.0
	}
	
	return float64(intraClusterEdges) / float64(totalEdges)
}

// ノードのクラスターを見つける
func (dv *DependencyVisualizer) findNodeCluster(nodeID string, result *DependencyVisualization) string {
	for _, cluster := range result.Clusters {
		for _, clusterNodeID := range cluster.NodeIDs {
			if clusterNodeID == nodeID {
				return cluster.ID
			}
		}
	}
	return ""
}

// 出力形式を変換
func (dv *DependencyVisualizer) ExportToJSON(visualization *DependencyVisualization) ([]byte, error) {
	return json.MarshalIndent(visualization, "", "  ")
}

