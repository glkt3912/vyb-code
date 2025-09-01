package search

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// テスト用のGoコードサンプル
const testGoCode = `package main

import (
	"fmt"
	"net/http"
	"context"
)

// ユーザー情報を表す構造体
type User struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
	Email string ` + "`json:\"email\"`" + `
}

// HTTPサーバーを開始する関数
func startServer(port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", handleUsers)
	mux.HandleFunc("/health", handleHealth)
	
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	
	fmt.Printf("サーバーをポート %s で開始します\n", port)
	return server.ListenAndServe()
}

// ユーザー一覧を処理するハンドラー
func handleUsers(w http.ResponseWriter, r *http.Request) {
	users := []User{
		{ID: 1, Name: "田中", Email: "tanaka@example.com"},
		{ID: 2, Name: "佐藤", Email: "sato@example.com"},
	}
	
	switch r.Method {
	case http.MethodGet:
		getUserList(w, users)
	case http.MethodPost:
		createUser(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ヘルスチェックハンドラー
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// ユーザー一覧を取得
func getUserList(w http.ResponseWriter, users []User) {
	// JSON エンコード処理（簡略化）
	fmt.Fprintf(w, "User list: %+v", users)
}

// 新しいユーザーを作成
func createUser(w http.ResponseWriter, r *http.Request) {
	// ユーザー作成処理（簡略化）
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User created"))
}

func main() {
	if err := startServer("8080"); err != nil {
		fmt.Printf("サーバーエラー: %v\n", err)
	}
}`

// TestIntelligentSearchAST はAST解析機能をテストする
func TestIntelligentSearchAST(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir := t.TempDir()

	// テスト用Goファイルを作成
	testFile := filepath.Join(tempDir, "test.go")
	if err := os.WriteFile(testFile, []byte(testGoCode), 0644); err != nil {
		t.Fatalf("テストファイル作成失敗: %v", err)
	}

	// 検索エンジンを作成
	engine := NewEngine(tempDir)
	intelligentSearch := NewIntelligentSearch(engine)

	// AST解析をテスト
	astInfo, err := intelligentSearch.parseGoFile(testFile)
	if err != nil {
		t.Fatalf("AST解析失敗: %v", err)
	}

	// パッケージ名をチェック
	if astInfo.Package != "main" {
		t.Errorf("期待されるパッケージ名: main, 実際: %s", astInfo.Package)
	}

	// インポート数をチェック
	expectedImports := 3 // fmt, net/http, context
	if len(astInfo.Imports) != expectedImports {
		t.Errorf("期待されるインポート数: %d, 実際: %d", expectedImports, len(astInfo.Imports))
	}

	// 関数数をチェック
	expectedFunctions := 6 // startServer, handleUsers, handleHealth, getUserList, createUser, main
	if len(astInfo.Functions) != expectedFunctions {
		t.Errorf("期待される関数数: %d, 実際: %d", expectedFunctions, len(astInfo.Functions))
	}

	// 型数をチェック
	expectedTypes := 1 // User struct
	if len(astInfo.Types) != expectedTypes {
		t.Errorf("期待される型数: %d, 実際: %d", expectedTypes, len(astInfo.Types))
	}

	// 特定の関数が検出されることを確認
	foundStartServer := false
	for _, function := range astInfo.Functions {
		if function.Name == "startServer" {
			foundStartServer = true
			// パラメータをチェック
			if len(function.Parameters) != 1 {
				t.Errorf("startServer関数のパラメータ数: 期待1, 実際%d", len(function.Parameters))
			}
			// 戻り値をチェック
			if len(function.Returns) != 1 || function.Returns[0] != "error" {
				t.Errorf("startServer関数の戻り値が正しくありません")
			}
			break
		}
	}
	if !foundStartServer {
		t.Error("startServer関数が検出されませんでした")
	}

	// User構造体が検出されることを確認
	foundUserType := false
	for _, typeInfo := range astInfo.Types {
		if typeInfo.Name == "User" && typeInfo.Kind == "struct" {
			foundUserType = true
			// フィールド数をチェック
			if len(typeInfo.Fields) != 3 {
				t.Errorf("User構造体のフィールド数: 期待3, 実際%d", len(typeInfo.Fields))
			}
			break
		}
	}
	if !foundUserType {
		t.Error("User構造体が検出されませんでした")
	}
}

// TestSmartSearch はスマート検索機能をテストする
func TestSmartSearch(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir := t.TempDir()

	// テスト用Goファイルを作成
	testFile := filepath.Join(tempDir, "server.go")
	if err := os.WriteFile(testFile, []byte(testGoCode), 0644); err != nil {
		t.Fatalf("テストファイル作成失敗: %v", err)
	}

	// 検索エンジンを作成してインデックス
	engine := NewEngine(tempDir)
	if err := engine.IndexProject(); err != nil {
		t.Fatalf("インデックス作成失敗: %v", err)
	}

	// スマート検索オプションを設定
	options := SmartSearchOptions{
		SearchOptions: SearchOptions{
			Pattern:      "server",
			MaxResults:   10,
			ContextLines: 1,
		},
		UseStructuralAnalysis: true,
		UseContextRanking:     true,
		IncludeASTInfo:        true,
		MinRelevanceScore:     0.0,
	}

	// スマート検索を実行
	results, err := engine.SmartSearch(options)
	if err != nil {
		t.Fatalf("スマート検索失敗: %v", err)
	}

	// 結果が存在することを確認
	if len(results) == 0 {
		t.Error("検索結果が見つかりませんでした")
		return
	}

	// スコアが計算されていることを確認
	for i, result := range results {
		if result.FinalScore < 0 || result.FinalScore > 1 {
			t.Errorf("結果%d: 不正なスコア値: %f", i, result.FinalScore)
		}

		// スマート検索の場合、構造的関連度が計算されているはず
		if result.StructuralRelevance < 0 || result.StructuralRelevance > 1 {
			t.Errorf("結果%d: 不正な構造的関連度: %f", i, result.StructuralRelevance)
		}
	}

	// 結果がスコア順でソートされていることを確認
	for i := 1; i < len(results); i++ {
		if results[i-1].FinalScore < results[i].FinalScore {
			t.Errorf("結果がスコア順でソートされていません: %f < %f",
				results[i-1].FinalScore, results[i].FinalScore)
		}
	}
}

// TestRelevanceScoring は関連度スコアリングをテストする
func TestRelevanceScoring(t *testing.T) {
	intelligentSearch := &IntelligentSearch{}

	// テストケース: 構造的関連度
	testASTInfo := &ASTInfo{
		Functions: []FunctionInfo{
			{Name: "startServer", Calls: []string{"http.ListenAndServe", "fmt.Printf"}},
			{Name: "handleUsers", Calls: []string{"getUserList", "createUser"}},
		},
		Types: []TypeInfo{
			{Name: "User", Kind: "struct"},
		},
	}

	// "server"パターンでの構造的関連度をテスト
	structuralScore := intelligentSearch.calculateStructuralRelevance(testASTInfo, "server")
	if structuralScore <= 0 {
		t.Errorf("構造的関連度が正しく計算されていません: %f", structuralScore)
	}

	// "user"パターンでの構造的関連度をテスト
	userScore := intelligentSearch.calculateStructuralRelevance(testASTInfo, "user")
	if userScore <= 0 {
		t.Errorf("ユーザー関連の構造的関連度が正しく計算されていません: %f", userScore)
	}

	// コード関連度のテスト
	testLine := "func startServer(port string) error {"
	codeScore := intelligentSearch.calculateCodeRelevance(testLine, "server")
	if codeScore <= 0 {
		t.Errorf("コード関連度が正しく計算されていません: %f", codeScore)
	}

	// 完全一致のテスト
	exactMatchScore := intelligentSearch.calculateCodeRelevance("server", "server")
	if exactMatchScore != 0.8 { // isCodeElement が false の場合
		t.Errorf("完全一致のコード関連度が期待値と異なります: %f", exactMatchScore)
	}
}

// TestRelatedSymbols は関連シンボル検索をテストする
func TestRelatedSymbols(t *testing.T) {
	intelligentSearch := &IntelligentSearch{}

	testASTInfo := &ASTInfo{
		Functions: []FunctionInfo{
			{Name: "startServer", Calls: []string{"http.ListenAndServe", "fmt.Printf"}},
			{Name: "handleUsers", Calls: []string{"getUserList", "createUser"}},
		},
		Types: []TypeInfo{
			{Name: "User", Kind: "struct", Fields: []string{"ID:int", "Name:string"}},
		},
	}

	// "server"に関連するシンボルを検索
	symbols := intelligentSearch.findRelatedSymbols(testASTInfo, "server")

	// 関連シンボルが見つかることを確認
	expectedSymbols := []string{"http.ListenAndServe", "fmt.Printf"}
	for _, expected := range expectedSymbols {
		found := false
		for _, symbol := range symbols {
			if symbol == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("期待される関連シンボル '%s' が見つかりませんでした", expected)
		}
	}
}

// TestASTCache はASTキャッシュ機能をテストする
func TestASTCache(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir := t.TempDir()

	// テスト用ファイルを作成
	testFile := filepath.Join(tempDir, "cache_test.go")
	if err := os.WriteFile(testFile, []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatalf("テストファイル作成失敗: %v", err)
	}

	engine := NewEngine(tempDir)
	intelligentSearch := NewIntelligentSearch(engine)

	// 最初の呼び出し（キャッシュなし）
	astInfo1, err := intelligentSearch.getOrCreateASTInfo(testFile)
	if err != nil {
		t.Fatalf("AST解析失敗: %v", err)
	}

	// 二回目の呼び出し（キャッシュあり）
	astInfo2, err := intelligentSearch.getOrCreateASTInfo(testFile)
	if err != nil {
		t.Fatalf("AST解析失敗: %v", err)
	}

	// 同じオブジェクトが返されることを確認（キャッシュが機能）
	if astInfo1 != astInfo2 {
		t.Error("ASTキャッシュが機能していません")
	}

	// キャッシュ統計を確認
	stats := intelligentSearch.GetASTStats()
	if stats["cached_files"].(int) != 1 {
		t.Errorf("キャッシュファイル数が正しくありません: %v", stats["cached_files"])
	}

	// キャッシュクリア機能をテスト
	intelligentSearch.ClearASTCache()
	clearedStats := intelligentSearch.GetASTStats()
	if clearedStats["cached_files"].(int) != 0 {
		t.Error("ASTキャッシュがクリアされていません")
	}
}

// TestCodeElementDetection はコード要素検出をテストする
func TestCodeElementDetection(t *testing.T) {
	intelligentSearch := &IntelligentSearch{}

	testCases := []struct {
		line     string
		pattern  string
		expected bool
		desc     string
	}{
		{"func startServer(port string) error {", "startServer", true, "関数定義"},
		{"result := calculateTotal(items)", "calculateTotal", true, "関数呼び出し"},
		{"type User struct {", "User", true, "型定義"},
		{"var serverPort string", "serverPort", true, "変数宣言"},
		{"server := &http.Server{", "server", true, "変数代入"},
		{"// this is a comment about server", "server", false, "コメント内"},
		{"fmt.Println(\"server started\")", "server", false, "文字列内"},
	}

	for _, tc := range testCases {
		result := intelligentSearch.isCodeElement(tc.line, tc.pattern)
		if result != tc.expected {
			t.Errorf("%s: 期待値 %t, 実際 %t (line: %s, pattern: %s)",
				tc.desc, tc.expected, result, tc.line, tc.pattern)
		}
	}
}

// TestComplexityCalculation は複雑度計算をテストする
func TestComplexityCalculation(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir := t.TempDir()

	complexCode := `package main

func simpleFunction() {
	// 複雑度 1（基本パス）
}

func complexFunction(x int) int {
	if x > 0 {          // +1
		for i := 0; i < x; i++ {  // +1
			if i%2 == 0 {     // +1
				continue
			}
			switch i {        // +1
			case 1:           // +1
				return 1
			case 2:           // +1
				return 2
			default:          // +1
				return -1
			}
		}
	}
	return 0
}`

	testFile := filepath.Join(tempDir, "complex.go")
	if err := os.WriteFile(testFile, []byte(complexCode), 0644); err != nil {
		t.Fatalf("テストファイル作成失敗: %v", err)
	}

	engine := NewEngine(tempDir)
	intelligentSearch := NewIntelligentSearch(engine)

	astInfo, err := intelligentSearch.parseGoFile(testFile)
	if err != nil {
		t.Fatalf("AST解析失敗: %v", err)
	}

	// 関数が2つ検出されることを確認
	if len(astInfo.Functions) != 2 {
		t.Errorf("期待される関数数: 2, 実際: %d", len(astInfo.Functions))
	}

	// 複雑な関数の複雑度をチェック
	for _, function := range astInfo.Functions {
		if function.Name == "simpleFunction" {
			if function.Complexity != 1 {
				t.Errorf("simpleFunction の複雑度: 期待1, 実際%d", function.Complexity)
			}
		} else if function.Name == "complexFunction" {
			if function.Complexity < 5 { // 少なくとも5以上のはず
				t.Errorf("complexFunction の複雑度が低すぎます: %d", function.Complexity)
			}
		}
	}
}

// TestSearchResultRanking は検索結果ランキングをテストする
func TestSearchResultRanking(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir := t.TempDir()

	// 複数のテストファイルを作成
	files := map[string]string{
		"server.go": `package main
func startServer() { }
func stopServer() { }`,

		"client.go": `package main  
func createClient() { }
func serverConnection() { }`, // "server"を含むが関連性は低い

		"utils.go": `package main
func helper() { }
const timeout = 30`,
	}

	for fileName, content := range files {
		testFile := filepath.Join(tempDir, fileName)
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("テストファイル作成失敗: %v", err)
		}
	}

	// 検索エンジンを作成してインデックス
	engine := NewEngine(tempDir)
	if err := engine.IndexProject(); err != nil {
		t.Fatalf("インデックス作成失敗: %v", err)
	}

	// "server" パターンでスマート検索
	options := SmartSearchOptions{
		SearchOptions: SearchOptions{
			Pattern:      "server",
			MaxResults:   10,
			ContextLines: 1,
		},
		UseStructuralAnalysis: true,
		UseContextRanking:     true,
		MinRelevanceScore:     0.0,
	}

	results, err := engine.SmartSearch(options)
	if err != nil {
		t.Fatalf("スマート検索失敗: %v", err)
	}

	// 結果が存在することを確認
	if len(results) == 0 {
		t.Fatal("検索結果が見つかりませんでした")
	}

	// server.go のファイルが上位にランクされることを期待
	topResult := results[0]
	if !strings.Contains(topResult.File.RelativePath, "server.go") {
		t.Logf("注意: トップ結果がserver.goではありません: %s (スコア: %.2f)",
			topResult.File.RelativePath, topResult.FinalScore)
	}

	// 全ての結果でスコアが計算されていることを確認
	for i, result := range results {
		if result.FinalScore <= 0 {
			t.Errorf("結果%d: スコアが計算されていません: %f", i, result.FinalScore)
		}
	}
}

// BenchmarkIntelligentSearch はインテリジェント検索のベンチマーク
func BenchmarkIntelligentSearch(b *testing.B) {
	// 一時ディレクトリを作成
	tempDir := b.TempDir()

	// 複数のテストファイルを作成
	for i := 0; i < 10; i++ {
		fileName := filepath.Join(tempDir, fmt.Sprintf("file%d.go", i))
		content := fmt.Sprintf(`package main
import "fmt"
func function%d() {
	fmt.Println("Function %d")
}
type Type%d struct {
	Field string
}`, i, i, i)

		if err := os.WriteFile(fileName, []byte(content), 0644); err != nil {
			b.Fatalf("テストファイル作成失敗: %v", err)
		}
	}

	// 検索エンジンを作成
	engine := NewEngine(tempDir)
	if err := engine.IndexProject(); err != nil {
		b.Fatalf("インデックス作成失敗: %v", err)
	}

	options := SmartSearchOptions{
		SearchOptions: SearchOptions{
			Pattern:      "function",
			MaxResults:   20,
			ContextLines: 1,
		},
		UseStructuralAnalysis: true,
		UseContextRanking:     true,
		MinRelevanceScore:     0.0,
	}

	b.ResetTimer()

	// ベンチマーク実行
	for i := 0; i < b.N; i++ {
		_, err := engine.SmartSearch(options)
		if err != nil {
			b.Fatalf("スマート検索失敗: %v", err)
		}
	}
}
