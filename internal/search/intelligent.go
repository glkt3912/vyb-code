package search

import (
	"container/list"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// LRUキャッシュエントリ
type astCacheEntry struct {
	key       string
	value     *ASTInfo
	timestamp time.Time
}

// インテリジェント検索エンジン
type IntelligentSearch struct {
	engine      *Engine
	astCache    map[string]*list.Element // LRUキャッシュ
	astCacheMu  sync.RWMutex
	lruList     *list.List               // LRU順序管理
	maxASTFiles int
}

// AST情報構造
type ASTInfo struct {
	FilePath   string         `json:"filePath"`
	Package    string         `json:"package"`
	Imports    []ImportInfo   `json:"imports"`
	Functions  []FunctionInfo `json:"functions"`
	Types      []TypeInfo     `json:"types"`
	Variables  []VariableInfo `json:"variables"`
	Comments   []CommentInfo  `json:"comments"`
	Complexity int            `json:"complexity"`
}

// インポート情報
type ImportInfo struct {
	Path  string `json:"path"`
	Alias string `json:"alias,omitempty"`
	Line  int    `json:"line"`
}

// 関数情報
type FunctionInfo struct {
	Name       string   `json:"name"`
	Receiver   string   `json:"receiver,omitempty"`
	Parameters []string `json:"parameters"`
	Returns    []string `json:"returns"`
	StartLine  int      `json:"startLine"`
	EndLine    int      `json:"endLine"`
	Complexity int      `json:"complexity"`
	Calls      []string `json:"calls"` // 呼び出している関数
}

// 型情報
type TypeInfo struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"` // struct, interface, etc.
	Fields    []string `json:"fields,omitempty"`
	Methods   []string `json:"methods,omitempty"`
	StartLine int      `json:"startLine"`
	EndLine   int      `json:"endLine"`
}

// 変数情報
type VariableInfo struct {
	Name  string `json:"name"`
	Type  string `json:"type,omitempty"`
	Line  int    `json:"line"`
	Scope string `json:"scope"` // package, function, block
}

// コメント情報
type CommentInfo struct {
	Text string `json:"text"`
	Line int    `json:"line"`
	Type string `json:"type"` // line, block, doc
}

// 拡張検索結果
type IntelligentResult struct {
	SearchResult                 // 基本検索結果を継承
	StructuralRelevance float64  `json:"structuralRelevance"`
	ContextRelevance    float64  `json:"contextRelevance"`
	CodeRelevance       float64  `json:"codeRelevance"`
	FinalScore          float64  `json:"finalScore"`
	ASTInfo             *ASTInfo `json:"astInfo,omitempty"`
	RelatedSymbols      []string `json:"relatedSymbols,omitempty"`
}

// スマート検索オプション
type SmartSearchOptions struct {
	SearchOptions                 // 基本オプションを継承
	UseStructuralAnalysis bool    `json:"useStructuralAnalysis"`
	UseContextRanking     bool    `json:"useContextRanking"`
	IncludeASTInfo        bool    `json:"includeASTInfo"`
	MinRelevanceScore     float64 `json:"minRelevanceScore"`
}

// 新しいインテリジェント検索エンジンを作成
func NewIntelligentSearch(engine *Engine) *IntelligentSearch {
	return &IntelligentSearch{
		engine:      engine,
		astCache:    make(map[string]*list.Element),
		lruList:     list.New(),
		maxASTFiles: 1000, // メモリ使用量制限
	}
}

// スマート検索を実行
func (is *IntelligentSearch) SmartSearch(options SmartSearchOptions) ([]IntelligentResult, error) {
	// 基本検索を実行
	basicResults, err := is.engine.SearchInFiles(options.SearchOptions)
	if err != nil {
		return nil, err
	}

	// 結果を拡張形式に変換
	intelligentResults := make([]IntelligentResult, 0, len(basicResults))

	for _, result := range basicResults {
		intelligentResult := IntelligentResult{
			SearchResult: result,
		}

		// 構造解析が有効な場合
		if options.UseStructuralAnalysis {
			if err := is.analyzeStructure(&intelligentResult, options); err == nil {
				// エラーの場合はスキップして続行
			}
		}

		// コンテキストランキングが有効な場合
		if options.UseContextRanking {
			is.calculateRelevanceScores(&intelligentResult, options.Pattern)
		}

		// 最終スコアを計算
		is.calculateFinalScore(&intelligentResult)

		// 最小関連度フィルター
		if intelligentResult.FinalScore >= options.MinRelevanceScore {
			intelligentResults = append(intelligentResults, intelligentResult)
		}
	}

	// 最終スコア順でソート
	sort.Slice(intelligentResults, func(i, j int) bool {
		return intelligentResults[i].FinalScore > intelligentResults[j].FinalScore
	})

	return intelligentResults, nil
}

// コード構造を解析
func (is *IntelligentSearch) analyzeStructure(result *IntelligentResult, options SmartSearchOptions) error {
	// Goファイルのみ解析
	if !strings.HasSuffix(result.File.Path, ".go") {
		return nil
	}

	// ASTキャッシュをチェック
	astInfo, err := is.getOrCreateASTInfo(result.File.Path)
	if err != nil {
		return err
	}

	if options.IncludeASTInfo {
		result.ASTInfo = astInfo
	}

	// 関連シンボルを特定
	result.RelatedSymbols = is.findRelatedSymbols(astInfo, options.Pattern)

	return nil
}

// AST情報を取得または作成（LRU最適化版）
func (is *IntelligentSearch) getOrCreateASTInfo(filePath string) (*ASTInfo, error) {
	// まずキャッシュチェック
	is.astCacheMu.Lock()
	if element, exists := is.astCache[filePath]; exists {
		// LRUの先頭に移動
		is.lruList.MoveToFront(element)
		entry := element.Value.(*astCacheEntry)
		
		// TTLチェック
		if time.Since(entry.timestamp) < 30*time.Minute {
			is.astCacheMu.Unlock()
			return entry.value, nil
		} else {
			// 期限切れエントリを削除
			is.lruList.Remove(element)
			delete(is.astCache, filePath)
		}
	}
	is.astCacheMu.Unlock()

	// AST解析を実行
	astInfo, err := is.parseGoFile(filePath)
	if err != nil {
		return nil, err
	}

	// キャッシュに保存（LRU管理）
	is.astCacheMu.Lock()
	defer is.astCacheMu.Unlock()
	
	if is.lruList.Len() >= is.maxASTFiles {
		is.evictLRU()
	}
	
	entry := &astCacheEntry{
		key:       filePath,
		value:     astInfo,
		timestamp: time.Now(),
	}
	
	element := is.lruList.PushFront(entry)
	is.astCache[filePath] = element

	return astInfo, nil
}

// Goファイルを解析してAST情報を抽出
func (is *IntelligentSearch) parseGoFile(filePath string) (*ASTInfo, error) {
	fileSet := token.NewFileSet()

	// ファイルをパース
	node, err := parser.ParseFile(fileSet, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	astInfo := &ASTInfo{
		FilePath: filePath,
		Package:  node.Name.Name,
	}

	// ASTを走査して情報を抽出
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ImportSpec:
			is.extractImportInfo(x, fileSet, astInfo)
		case *ast.FuncDecl:
			is.extractFunctionInfo(x, fileSet, astInfo)
		case *ast.TypeSpec:
			is.extractTypeInfo(x, fileSet, astInfo)
		case *ast.GenDecl:
			is.extractVariableInfo(x, fileSet, astInfo)
		}
		return true
	})

	// コメントを処理
	is.extractComments(node.Comments, fileSet, astInfo)

	// 複雑度を計算
	astInfo.Complexity = is.calculateComplexity(astInfo)

	return astInfo, nil
}

// インポート情報を抽出
func (is *IntelligentSearch) extractImportInfo(importSpec *ast.ImportSpec, fileSet *token.FileSet, astInfo *ASTInfo) {
	importInfo := ImportInfo{
		Path: strings.Trim(importSpec.Path.Value, `"`),
		Line: fileSet.Position(importSpec.Pos()).Line,
	}

	if importSpec.Name != nil {
		importInfo.Alias = importSpec.Name.Name
	}

	astInfo.Imports = append(astInfo.Imports, importInfo)
}

// 関数情報を抽出
func (is *IntelligentSearch) extractFunctionInfo(funcDecl *ast.FuncDecl, fileSet *token.FileSet, astInfo *ASTInfo) {
	funcInfo := FunctionInfo{
		Name:      funcDecl.Name.Name,
		StartLine: fileSet.Position(funcDecl.Pos()).Line,
		EndLine:   fileSet.Position(funcDecl.End()).Line,
	}

	// レシーバーを抽出
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		if field := funcDecl.Recv.List[0]; field.Type != nil {
			funcInfo.Receiver = is.typeToString(field.Type)
		}
	}

	// パラメータを抽出
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			paramType := is.typeToString(param.Type)
			for _, name := range param.Names {
				funcInfo.Parameters = append(funcInfo.Parameters, name.Name+":"+paramType)
			}
		}
	}

	// 戻り値を抽出
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			funcInfo.Returns = append(funcInfo.Returns, is.typeToString(result.Type))
		}
	}

	// 関数呼び出しを抽出
	funcInfo.Calls = is.extractFunctionCalls(funcDecl)
	funcInfo.Complexity = is.calculateFunctionComplexity(funcDecl)

	astInfo.Functions = append(astInfo.Functions, funcInfo)
}

// 型情報を抽出
func (is *IntelligentSearch) extractTypeInfo(typeSpec *ast.TypeSpec, fileSet *token.FileSet, astInfo *ASTInfo) {
	typeInfo := TypeInfo{
		Name:      typeSpec.Name.Name,
		StartLine: fileSet.Position(typeSpec.Pos()).Line,
		EndLine:   fileSet.Position(typeSpec.End()).Line,
	}

	switch t := typeSpec.Type.(type) {
	case *ast.StructType:
		typeInfo.Kind = "struct"
		if t.Fields != nil {
			for _, field := range t.Fields.List {
				fieldType := is.typeToString(field.Type)
				for _, name := range field.Names {
					typeInfo.Fields = append(typeInfo.Fields, name.Name+":"+fieldType)
				}
			}
		}
	case *ast.InterfaceType:
		typeInfo.Kind = "interface"
		if t.Methods != nil {
			for _, method := range t.Methods.List {
				for _, name := range method.Names {
					typeInfo.Methods = append(typeInfo.Methods, name.Name)
				}
			}
		}
	default:
		typeInfo.Kind = "alias"
	}

	astInfo.Types = append(astInfo.Types, typeInfo)
}

// 変数情報を抽出
func (is *IntelligentSearch) extractVariableInfo(genDecl *ast.GenDecl, fileSet *token.FileSet, astInfo *ASTInfo) {
	if genDecl.Tok != token.VAR && genDecl.Tok != token.CONST {
		return
	}

	for _, spec := range genDecl.Specs {
		if valueSpec, ok := spec.(*ast.ValueSpec); ok {
			varType := ""
			if valueSpec.Type != nil {
				varType = is.typeToString(valueSpec.Type)
			}

			for _, name := range valueSpec.Names {
				varInfo := VariableInfo{
					Name:  name.Name,
					Type:  varType,
					Line:  fileSet.Position(name.Pos()).Line,
					Scope: "package",
				}
				astInfo.Variables = append(astInfo.Variables, varInfo)
			}
		}
	}
}

// コメント情報を抽出
func (is *IntelligentSearch) extractComments(comments []*ast.CommentGroup, fileSet *token.FileSet, astInfo *ASTInfo) {
	for _, commentGroup := range comments {
		for _, comment := range commentGroup.List {
			commentInfo := CommentInfo{
				Text: strings.TrimPrefix(strings.TrimPrefix(comment.Text, "//"), "/*"),
				Line: fileSet.Position(comment.Pos()).Line,
				Type: "line",
			}
			if strings.HasPrefix(comment.Text, "/*") {
				commentInfo.Type = "block"
			}
			astInfo.Comments = append(astInfo.Comments, commentInfo)
		}
	}
}

// 関数内の関数呼び出しを抽出
func (is *IntelligentSearch) extractFunctionCalls(funcDecl *ast.FuncDecl) []string {
	var calls []string
	callMap := make(map[string]bool)

	ast.Inspect(funcDecl, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if callName := is.extractCallName(callExpr.Fun); callName != "" {
				if !callMap[callName] {
					calls = append(calls, callName)
					callMap[callName] = true
				}
			}
		}
		return true
	})

	return calls
}

// 関数呼び出し名を抽出
func (is *IntelligentSearch) extractCallName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		if ident, ok := x.X.(*ast.Ident); ok {
			return ident.Name + "." + x.Sel.Name
		}
		return x.Sel.Name
	}
	return ""
}

// 型表現を文字列に変換
func (is *IntelligentSearch) typeToString(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return is.typeToString(x.X) + "." + x.Sel.Name
	case *ast.StarExpr:
		return "*" + is.typeToString(x.X)
	case *ast.ArrayType:
		return "[]" + is.typeToString(x.Elt)
	case *ast.MapType:
		return "map[" + is.typeToString(x.Key) + "]" + is.typeToString(x.Value)
	case *ast.ChanType:
		return "chan " + is.typeToString(x.Value)
	}
	return "unknown"
}

// 関連シンボルを検索
func (is *IntelligentSearch) findRelatedSymbols(astInfo *ASTInfo, pattern string) []string {
	var symbols []string
	symbolMap := make(map[string]bool)

	pattern = strings.ToLower(pattern)

	// 関数名から関連シンボルを検索
	for _, function := range astInfo.Functions {
		if strings.Contains(strings.ToLower(function.Name), pattern) {
			// 関数の呼び出し先を関連シンボルとして追加
			for _, call := range function.Calls {
				if !symbolMap[call] {
					symbols = append(symbols, call)
					symbolMap[call] = true
				}
			}
		}
	}

	// 型から関連シンボルを検索
	for _, typeInfo := range astInfo.Types {
		if strings.Contains(strings.ToLower(typeInfo.Name), pattern) {
			// 型のフィールドやメソッドを関連シンボルとして追加
			for _, field := range typeInfo.Fields {
				if !symbolMap[field] {
					symbols = append(symbols, field)
					symbolMap[field] = true
				}
			}
			for _, method := range typeInfo.Methods {
				if !symbolMap[method] {
					symbols = append(symbols, method)
					symbolMap[method] = true
				}
			}
		}
	}

	return symbols
}

// 関連度スコアを計算
func (is *IntelligentSearch) calculateRelevanceScores(result *IntelligentResult, pattern string) {
	// 構造的関連度（関数、型、変数の一致度）
	result.StructuralRelevance = is.calculateStructuralRelevance(result.ASTInfo, pattern)

	// コンテキスト関連度（周辺コードとの関連性）
	result.ContextRelevance = is.calculateContextRelevance(result, pattern)

	// コード関連度（実際のコード内容との関連性）
	result.CodeRelevance = is.calculateCodeRelevance(result.Line, pattern)
}

// 構造的関連度を計算
func (is *IntelligentSearch) calculateStructuralRelevance(astInfo *ASTInfo, pattern string) float64 {
	if astInfo == nil {
		return 0.5 // デフォルト値
	}

	score := 0.0
	pattern = strings.ToLower(pattern)

	// 関数名マッチング
	for _, function := range astInfo.Functions {
		if strings.Contains(strings.ToLower(function.Name), pattern) {
			score += 0.8
		}
		// 関数呼び出しマッチング
		for _, call := range function.Calls {
			if strings.Contains(strings.ToLower(call), pattern) {
				score += 0.4
			}
		}
	}

	// 型名マッチング
	for _, typeInfo := range astInfo.Types {
		if strings.Contains(strings.ToLower(typeInfo.Name), pattern) {
			score += 0.7
		}
	}

	// 変数名マッチング
	for _, variable := range astInfo.Variables {
		if strings.Contains(strings.ToLower(variable.Name), pattern) {
			score += 0.3
		}
	}

	// 正規化（0-1の範囲）
	return normalizeScore(score)
}

// コンテキスト関連度を計算
func (is *IntelligentSearch) calculateContextRelevance(result *IntelligentResult, pattern string) float64 {
	score := 0.0

	// コンテキスト行での一致
	for _, contextLine := range result.Context {
		if strings.Contains(strings.ToLower(contextLine), strings.ToLower(pattern)) {
			score += 0.2
		}
	}

	// ファイル名での一致
	fileName := filepath.Base(result.File.Path)
	if strings.Contains(strings.ToLower(fileName), strings.ToLower(pattern)) {
		score += 0.5
	}

	return normalizeScore(score)
}

// コード関連度を計算
func (is *IntelligentSearch) calculateCodeRelevance(line, pattern string) float64 {
	line = strings.ToLower(line)
	pattern = strings.ToLower(pattern)

	// 完全一致
	if strings.Contains(line, pattern) {
		// パターンがコード要素（関数、変数等）の一部かチェック
		if is.isCodeElement(line, pattern) {
			return 1.0
		}
		return 0.8
	}

	// 部分一致
	words := strings.Fields(pattern)
	matchCount := 0
	for _, word := range words {
		if strings.Contains(line, word) {
			matchCount++
		}
	}

	if len(words) > 0 {
		return float64(matchCount) / float64(len(words)) * 0.6
	}

	return 0.0
}

// 最終スコアを計算
func (is *IntelligentSearch) calculateFinalScore(result *IntelligentResult) {
	// 重み付き平均
	weights := map[string]float64{
		"structural": 0.4,
		"context":    0.3,
		"code":       0.3,
	}

	result.FinalScore =
		result.StructuralRelevance*weights["structural"] +
			result.ContextRelevance*weights["context"] +
			result.CodeRelevance*weights["code"]
}

// コード要素かどうかを判定
func (is *IntelligentSearch) isCodeElement(line, pattern string) bool {
	// 関数定義・呼び出しパターン
	codePatterns := []string{
		"func " + pattern,
		pattern + "(",
		"type " + pattern,
		"var " + pattern,
		"const " + pattern,
		"." + pattern + "(",
		pattern + " :=",
		pattern + " =",
	}

	for _, codePattern := range codePatterns {
		if strings.Contains(line, codePattern) {
			return true
		}
	}

	return false
}

// 関数の複雑度を計算（サイクロマティック複雑度の簡易版）
func (is *IntelligentSearch) calculateFunctionComplexity(funcDecl *ast.FuncDecl) int {
	complexity := 1 // 基本パス

	ast.Inspect(funcDecl, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		}
		return true
	})

	return complexity
}

// AST情報全体の複雑度を計算
func (is *IntelligentSearch) calculateComplexity(astInfo *ASTInfo) int {
	complexity := 0
	for _, function := range astInfo.Functions {
		complexity += function.Complexity
	}
	return complexity
}

// LRU方式でASTエントリを削除
func (is *IntelligentSearch) evictLRU() {
	if is.lruList.Len() == 0 {
		return
	}
	
	// 最も古いエントリを削除
	oldest := is.lruList.Back()
	if oldest != nil {
		entry := oldest.Value.(*astCacheEntry)
		delete(is.astCache, entry.key)
		is.lruList.Remove(oldest)
	}
}

// スコアを正規化（0-1の範囲）
func normalizeScore(score float64) float64 {
	if score > 1.0 {
		return 1.0
	}
	if score < 0.0 {
		return 0.0
	}
	return score
}

// ASTキャッシュクリア
func (is *IntelligentSearch) ClearASTCache() {
	is.astCacheMu.Lock()
	defer is.astCacheMu.Unlock()

	is.astCache = make(map[string]*list.Element)
	is.lruList = list.New()
}

// AST統計情報取得
func (is *IntelligentSearch) GetASTStats() map[string]interface{} {
	is.astCacheMu.RLock()
	defer is.astCacheMu.RUnlock()

	stats := map[string]interface{}{
		"cached_files":       len(is.astCache),
		"max_files":          is.maxASTFiles,
		"lru_list_length":    is.lruList.Len(),
		"cache_utilization":  float64(len(is.astCache)) / float64(is.maxASTFiles) * 100,
	}
	
	// キャッシュ効率性統計
	expiredCount := 0
	if is.lruList.Len() > 0 {
		for element := is.lruList.Front(); element != nil; element = element.Next() {
			entry := element.Value.(*astCacheEntry)
			if time.Since(entry.timestamp) > 30*time.Minute {
				expiredCount++
			}
		}
	}
	
	stats["expired_entries"] = expiredCount
	
	return stats
}
