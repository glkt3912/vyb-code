package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run shared_types.go <project_root>")
		os.Exit(1)
	}

	rootDir := os.Args[1]
	internalDir := filepath.Join(rootDir, "internal")

	fmt.Printf("Analyzing dependencies in: %s\n", internalDir)

	graph, err := AnalyzeDependencies(internalDir)
	if err != nil {
		fmt.Printf("Error analyzing dependencies: %v\n", err)
		os.Exit(1)
	}

	// 依存関係グラフを表示
	graph.PrintGraph()

	// 循環依存を検出
	cycles := graph.DetectCycles()
	if len(cycles) == 0 {
		fmt.Println("\n✅ No circular dependencies found!")
	} else {
		fmt.Printf("\n❌ Found %d circular dependencies:\n", len(cycles))
		for i, cycle := range cycles {
			fmt.Printf("  Cycle %d: %s\n", i+1, strings.Join(cycle, " -> "))
		}

		fmt.Println("\n💡 Suggestions to resolve cycles:")
		suggestSolutions(cycles)
	}

	// メトリクス分析も実行
	fmt.Println("\n" + strings.Repeat("=", 60))
	runOptimizationAnalysis(graph)
}

// suggestSolutions は循環依存解決の提案を表示
func suggestSolutions(cycles [][]string) {
	suggestions := []string{
		"1. Extract common interfaces to a shared package",
		"2. Use dependency injection instead of direct imports",
		"3. Apply the Dependency Inversion Principle",
		"4. Move shared types to a separate types package",
		"5. Use event-driven communication instead of direct calls",
		"6. Introduce a mediator pattern",
		"7. Split large packages into smaller, focused ones",
	}

	for _, suggestion := range suggestions {
		fmt.Printf("   %s\n", suggestion)
	}
}

// runOptimizationAnalysis は最適化分析を実行
func runOptimizationAnalysis(graph *DependencyGraph) {
	metrics := AnalyzeDependencyComplexity(graph)

	// 現在の状態を表示
	fmt.Println("\n=== Current Dependency Metrics ===")
	fmt.Printf("%-20s %s %s %s %s\n", "Package", "In", "Out", "Coupling", "Impact")
	fmt.Println(strings.Repeat("-", 65))

	for _, m := range metrics {
		fmt.Printf("%-20s %2d %3d %8.1f %6.1f\n",
			m.PackageName, m.IncomingCount, m.OutgoingCount, m.CouplingScore, m.ImpactScore)
	}

	// 最適化提案を生成
	GenerateOptimizationSuggestions(metrics, graph)

	// 具体的なリファクタリング計画
	GenerateRefactoringPlan(metrics, graph)
}

// DependencyGraph は依存関係グラフを表す
type DependencyGraph struct {
	nodes map[string]*Node
}

// Node はグラフのノード（パッケージ）
type Node struct {
	name         string
	dependencies []string
	dependents   []string
}

// NewDependencyGraph は新しい依存関係グラフを作成
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*Node),
	}
}

// AddNode はノードを追加
func (g *DependencyGraph) AddNode(name string) {
	if _, exists := g.nodes[name]; !exists {
		g.nodes[name] = &Node{
			name:         name,
			dependencies: []string{},
			dependents:   []string{},
		}
	}
}

// AddDependency は依存関係を追加
func (g *DependencyGraph) AddDependency(from, to string) {
	g.AddNode(from)
	g.AddNode(to)

	fromNode := g.nodes[from]
	toNode := g.nodes[to]

	// 重複チェック
	for _, dep := range fromNode.dependencies {
		if dep == to {
			return
		}
	}

	fromNode.dependencies = append(fromNode.dependencies, to)
	toNode.dependents = append(toNode.dependents, from)
}

// DetectCycles は循環依存を検出
func (g *DependencyGraph) DetectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for name := range g.nodes {
		if !visited[name] {
			if cycle := g.dfs(name, visited, recStack, []string{}); len(cycle) > 0 {
				cycles = append(cycles, cycle)
			}
		}
	}

	return cycles
}

// dfs は深度優先探索で循環を検出
func (g *DependencyGraph) dfs(node string, visited, recStack map[string]bool, path []string) []string {
	visited[node] = true
	recStack[node] = true
	path = append(path, node)

	for _, dep := range g.nodes[node].dependencies {
		if !visited[dep] {
			if cycle := g.dfs(dep, visited, recStack, path); len(cycle) > 0 {
				return cycle
			}
		} else if recStack[dep] {
			// 循環を発見
			cycleStart := -1
			for i, p := range path {
				if p == dep {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := make([]string, len(path)-cycleStart)
				copy(cycle, path[cycleStart:])
				cycle = append(cycle, dep) // 循環を完成
				return cycle
			}
		}
	}

	recStack[node] = false
	return []string{}
}

// PrintGraph はグラフを表示
func (g *DependencyGraph) PrintGraph() {
	fmt.Println("\n=== Dependency Graph ===")

	var nodes []string
	for name := range g.nodes {
		nodes = append(nodes, name)
	}
	sort.Strings(nodes)

	for _, name := range nodes {
		node := g.nodes[name]
		if len(node.dependencies) > 0 {
			fmt.Printf("%s -> %s\n", name, strings.Join(node.dependencies, ", "))
		}
	}
}

// AnalyzeDependencies はプロジェクトの依存関係を分析
func AnalyzeDependencies(rootDir string) (*DependencyGraph, error) {
	graph := NewDependencyGraph()

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		if strings.Contains(path, "vendor/") {
			return nil
		}

		return analyzeFile(path, rootDir, graph)
	})

	return graph, err
}

// analyzeFile は個別ファイルの依存関係を分析
func analyzeFile(filePath, rootDir string, graph *DependencyGraph) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %v", filePath, err)
	}

	packageName := getPackageName(filePath, rootDir)
	graph.AddNode(packageName)

	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		if strings.HasPrefix(importPath, "github.com/glkt/vyb-code/internal/") {
			depPackage := strings.TrimPrefix(importPath, "github.com/glkt/vyb-code/internal/")
			graph.AddDependency(packageName, depPackage)
		}
	}

	return nil
}

// getPackageName はファイルパスからパッケージ名を取得
func getPackageName(filePath, rootDir string) string {
	rel, err := filepath.Rel(rootDir, filePath)
	if err != nil {
		return filepath.Dir(filePath)
	}

	dir := filepath.Dir(rel)
	if strings.HasPrefix(dir, "internal/") {
		return strings.TrimPrefix(dir, "internal/")
	}

	return dir
}

// DependencyMetrics は依存関係メトリクスを計算
type DependencyMetrics struct {
	PackageName     string
	IncomingCount   int // このパッケージに依存するパッケージ数
	OutgoingCount   int // このパッケージが依存するパッケージ数
	CouplingScore   float64
	ImpactScore     float64
	ModularityScore float64
}

// AnalyzeDependencyComplexity は依存関係の複雑さを分析
func AnalyzeDependencyComplexity(graph *DependencyGraph) []DependencyMetrics {
	var metrics []DependencyMetrics

	for name, node := range graph.nodes {
		metric := DependencyMetrics{
			PackageName:   name,
			IncomingCount: len(node.dependents),
			OutgoingCount: len(node.dependencies),
		}

		// 結合度スコア（依存関係の多さ）
		metric.CouplingScore = float64(metric.OutgoingCount)

		// 影響度スコア（他から依存される多さ）
		metric.ImpactScore = float64(metric.IncomingCount)

		// モジュール性スコア（理想的な依存関係との差）
		idealOutgoing := 3.0 // 理想的な依存数
		metric.ModularityScore = 10.0 - (abs(float64(metric.OutgoingCount)-idealOutgoing) + float64(metric.IncomingCount)*0.5)

		metrics = append(metrics, metric)
	}

	// インパクトスコアでソート
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].ImpactScore > metrics[j].ImpactScore
	})

	return metrics
}

// abs は絶対値を返す
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// GenerateOptimizationSuggestions は最適化提案を生成
func GenerateOptimizationSuggestions(metrics []DependencyMetrics, graph *DependencyGraph) {
	fmt.Println("\n=== Dependency Optimization Suggestions ===")

	// 1. 高結合度パッケージを特定
	fmt.Println("\n🔗 High Coupling Packages (Consider Splitting):")
	for _, m := range metrics {
		if m.CouplingScore > 7 {
			fmt.Printf("  - %s (dependencies: %d)\n", m.PackageName, m.OutgoingCount)
			printDependencies(m.PackageName, graph)
		}
	}

	// 2. 高影響度パッケージを特定
	fmt.Println("\n🎯 High Impact Packages (Core Components):")
	for _, m := range metrics {
		if m.ImpactScore > 3 {
			fmt.Printf("  - %s (dependents: %d)\n", m.PackageName, m.IncomingCount)
		}
	}

	// 3. 改善提案
	fmt.Println("\n💡 Specific Optimization Recommendations:")

	for _, m := range metrics {
		if m.CouplingScore > 5 && m.ImpactScore > 2 {
			fmt.Printf("  📦 %s: Consider creating interfaces to reduce coupling\n", m.PackageName)
		}
		if m.OutgoingCount > 8 {
			fmt.Printf("  ✂️  %s: Consider splitting into smaller modules\n", m.PackageName)
		}
		if m.IncomingCount == 0 && m.OutgoingCount > 0 {
			fmt.Printf("  🗑️  %s: Potential dead code or entry point\n", m.PackageName)
		}
	}

	// 4. アーキテクチャ層の提案
	suggestArchitectureLayers(metrics)
}

// printDependencies は依存関係を表示
func printDependencies(packageName string, graph *DependencyGraph) {
	node := graph.nodes[packageName]
	if len(node.dependencies) > 0 {
		fmt.Printf("    Dependencies: %s\n", strings.Join(node.dependencies, ", "))
	}
}

// suggestArchitectureLayers はアーキテクチャ層を提案
func suggestArchitectureLayers(metrics []DependencyMetrics) {
	fmt.Println("\n🏗️  Suggested Architecture Layers:")

	// 各パッケージを層に分類
	layers := map[string][]string{
		"Infrastructure": {},
		"Core":           {},
		"Service":        {},
		"Handler":        {},
		"Extension":      {},
	}

	for _, m := range metrics {
		switch {
		case strings.Contains(m.PackageName, "config") || strings.Contains(m.PackageName, "logger"):
			layers["Infrastructure"] = append(layers["Infrastructure"], m.PackageName)
		case m.ImpactScore > 4:
			layers["Core"] = append(layers["Core"], m.PackageName)
		case m.CouplingScore > 6:
			layers["Service"] = append(layers["Service"], m.PackageName)
		case strings.Contains(m.PackageName, "handlers"):
			layers["Handler"] = append(layers["Handler"], m.PackageName)
		default:
			layers["Extension"] = append(layers["Extension"], m.PackageName)
		}
	}

	for layer, packages := range layers {
		if len(packages) > 0 {
			fmt.Printf("  %s Layer: %s\n", layer, strings.Join(packages, ", "))
		}
	}
}

// GenerateRefactoringPlan は具体的なリファクタリング計画を生成
func GenerateRefactoringPlan(metrics []DependencyMetrics, graph *DependencyGraph) {
	fmt.Println("\n=== Refactoring Action Plan ===")

	// Priority 1: 最も問題のあるパッケージ
	fmt.Println("\n🚨 Priority 1 (Critical):")
	for _, m := range metrics {
		if m.CouplingScore > 8 || (m.ImpactScore > 5 && m.CouplingScore > 5) {
			fmt.Printf("  1. Refactor %s:\n", m.PackageName)
			fmt.Printf("     - Extract interfaces for high-level abstractions\n")
			fmt.Printf("     - Use dependency injection instead of direct imports\n")
			fmt.Printf("     - Consider splitting into 2-3 smaller packages\n")
			fmt.Printf("     - Current coupling: %.1f, Impact: %.1f\n\n", m.CouplingScore, m.ImpactScore)
		}
	}

	// Priority 2: 改善が推奨される
	fmt.Println("⚠️  Priority 2 (Recommended):")
	for _, m := range metrics {
		if m.CouplingScore > 5 && m.CouplingScore <= 8 {
			fmt.Printf("  2. Improve %s:\n", m.PackageName)
			fmt.Printf("     - Review dependencies and remove unnecessary ones\n")
			fmt.Printf("     - Create facade patterns for complex interactions\n")
			fmt.Printf("     - Current coupling: %.1f\n\n", m.CouplingScore)
		}
	}

	// Priority 3: 監視対象
	fmt.Println("👀 Priority 3 (Monitor):")
	for _, m := range metrics {
		if m.ModularityScore < 7 && m.CouplingScore <= 5 {
			fmt.Printf("  3. Monitor %s (Modularity: %.1f)\n", m.PackageName, m.ModularityScore)
		}
	}
}
