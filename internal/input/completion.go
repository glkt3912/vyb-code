package input

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// 高度なオートコンプリート機能
type AdvancedCompleter struct {
	workDir         string
	cache           *CompletionCache
	gitCompleter    *GitCompleter
	projectAnalyzer *ProjectAnalyzer
	fuzzyMatcher    *FuzzyMatcher
}

// 補完キャッシュシステム
type CompletionCache struct {
	fileCache    map[string]*CacheEntry
	commandCache map[string]*CacheEntry
	gitCache     *CacheEntry
	maxAge       time.Duration
	maxSize      int
}

// キャッシュエントリ
type CacheEntry struct {
	data      []string
	timestamp time.Time
}

// Git関連補完機能
type GitCompleter struct {
	workDir string
}

// プロジェクト構造解析
type ProjectAnalyzer struct {
	workDir       string
	projectType   string
	dependencies  []string
	buildCommands []string
	testCommands  []string
	mainFiles     []string
}

// ファジーマッチング機能
type FuzzyMatcher struct {
	threshold float64 // マッチング閾値（0.0-1.0）
}

// 補完候補の詳細情報
type CompletionCandidate struct {
	Text        string
	Description string
	Type        CompletionType
	Score       float64
	Context     map[string]interface{}
}

// 補完タイプ
type CompletionType int

const (
	CompletionFile CompletionType = iota
	CompletionCommand
	CompletionGitBranch
	CompletionGitFile
	CompletionProjectCommand
	CompletionDependency
)

// 高度なコンプリーターを作成
func NewAdvancedCompleter(workDir string) *AdvancedCompleter {
	return &AdvancedCompleter{
		workDir:         workDir,
		cache:           NewCompletionCache(),
		gitCompleter:    NewGitCompleter(workDir),
		projectAnalyzer: NewProjectAnalyzer(workDir),
		fuzzyMatcher:    NewFuzzyMatcher(0.6),
	}
}

// 補完キャッシュを作成
func NewCompletionCache() *CompletionCache {
	return &CompletionCache{
		fileCache:    make(map[string]*CacheEntry),
		commandCache: make(map[string]*CacheEntry),
		maxAge:       5 * time.Minute,
		maxSize:      1000,
	}
}

// Git補完機能を作成
func NewGitCompleter(workDir string) *GitCompleter {
	return &GitCompleter{
		workDir: workDir,
	}
}

// プロジェクト解析機能を作成
func NewProjectAnalyzer(workDir string) *ProjectAnalyzer {
	analyzer := &ProjectAnalyzer{
		workDir: workDir,
	}
	analyzer.analyze()
	return analyzer
}

// ファジーマッチャーを作成
func NewFuzzyMatcher(threshold float64) *FuzzyMatcher {
	return &FuzzyMatcher{
		threshold: threshold,
	}
}

// 高度な補完候補を取得
func (ac *AdvancedCompleter) GetAdvancedSuggestions(input string) []CompletionCandidate {
	input = strings.TrimSpace(input)
	if input == "" {
		return ac.getDefaultSuggestions()
	}

	var candidates []CompletionCandidate

	// コンテキスト依存の補完
	if strings.HasPrefix(input, "/") {
		candidates = append(candidates, ac.getSlashCommandCompletions(input)...)
	} else if strings.HasPrefix(input, "git ") {
		candidates = append(candidates, ac.getGitCompletions(input)...)
	} else if ac.isPathLike(input) {
		candidates = append(candidates, ac.getFileCompletions(input)...)
	} else {
		// 一般的なコマンド補完
		candidates = append(candidates, ac.getCommandCompletions(input)...)
		candidates = append(candidates, ac.getProjectCommandCompletions(input)...)

		// ファジーマッチングによる候補拡張
		fuzzyCandidates := ac.getFuzzyCompletions(input)
		candidates = append(candidates, fuzzyCandidates...)
	}

	// スコアでソート
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// 重複除去
	candidates = ac.removeDuplicates(candidates)

	// 上位候補のみ返す（最大20個）
	if len(candidates) > 20 {
		candidates = candidates[:20]
	}

	return candidates
}

// デフォルト候補を取得
func (ac *AdvancedCompleter) getDefaultSuggestions() []CompletionCandidate {
	var candidates []CompletionCandidate

	// よく使われるコマンド
	commonCommands := []string{
		"help", "build", "test", "analyze", "status", "search", "find",
	}

	for _, cmd := range commonCommands {
		candidates = append(candidates, CompletionCandidate{
			Text:        cmd,
			Description: fmt.Sprintf("vyb %s コマンド", cmd),
			Type:        CompletionCommand,
			Score:       0.8,
		})
	}

	// プロジェクト固有のコマンド
	projectCommands := ac.projectAnalyzer.buildCommands
	for _, cmd := range projectCommands {
		candidates = append(candidates, CompletionCandidate{
			Text:        cmd,
			Description: "プロジェクトビルドコマンド",
			Type:        CompletionProjectCommand,
			Score:       0.9,
		})
	}

	return candidates
}

// スラッシュコマンド補完
func (ac *AdvancedCompleter) getSlashCommandCompletions(input string) []CompletionCandidate {
	var candidates []CompletionCandidate

	slashCommands := map[string]string{
		"/help":    "ヘルプ表示",
		"/clear":   "画面クリア",
		"/history": "履歴表示",
		"/status":  "ステータス表示",
		"/info":    "情報表示",
		"/save":    "セッション保存",
		"/retry":   "再実行",
		"/edit":    "編集モード",
		"/exit":    "終了",
		"/quit":    "終了",
	}

	for cmd, desc := range slashCommands {
		if strings.HasPrefix(cmd, input) {
			score := ac.calculateScore(input, cmd)
			candidates = append(candidates, CompletionCandidate{
				Text:        cmd,
				Description: desc,
				Type:        CompletionCommand,
				Score:       score,
			})
		}
	}

	return candidates
}

// Git関連補完
func (ac *AdvancedCompleter) getGitCompletions(input string) []CompletionCandidate {
	var candidates []CompletionCandidate

	// キャッシュチェック
	if cached := ac.cache.getGitCache(); cached != nil {
		return ac.filterGitCandidates(cached, input)
	}

	parts := strings.Fields(input)
	if len(parts) < 2 {
		return candidates
	}

	gitSubcommand := parts[1]

	switch gitSubcommand {
	case "checkout", "switch", "merge":
		// ブランチ補完
		branches := ac.gitCompleter.getBranches()
		for _, branch := range branches {
			candidates = append(candidates, CompletionCandidate{
				Text:        fmt.Sprintf("git %s %s", gitSubcommand, branch),
				Description: fmt.Sprintf("ブランチ %s に%s", branch, getGitActionDescription(gitSubcommand)),
				Type:        CompletionGitBranch,
				Score:       0.9,
			})
		}

	case "add", "rm", "restore":
		// 変更されたファイル補完
		files := ac.gitCompleter.getModifiedFiles()
		for _, file := range files {
			candidates = append(candidates, CompletionCandidate{
				Text:        fmt.Sprintf("git %s %s", gitSubcommand, file),
				Description: fmt.Sprintf("ファイル %s を%s", file, getGitActionDescription(gitSubcommand)),
				Type:        CompletionGitFile,
				Score:       0.85,
			})
		}

	case "diff", "show", "log":
		// ファイルとブランチ両方
		branches := ac.gitCompleter.getBranches()
		files := ac.gitCompleter.getTrackedFiles()

		for _, branch := range branches {
			candidates = append(candidates, CompletionCandidate{
				Text:        fmt.Sprintf("git %s %s", gitSubcommand, branch),
				Description: fmt.Sprintf("ブランチ %s の%s", branch, getGitActionDescription(gitSubcommand)),
				Type:        CompletionGitBranch,
				Score:       0.8,
			})
		}

		for _, file := range files {
			candidates = append(candidates, CompletionCandidate{
				Text:        fmt.Sprintf("git %s %s", gitSubcommand, file),
				Description: fmt.Sprintf("ファイル %s の%s", file, getGitActionDescription(gitSubcommand)),
				Type:        CompletionGitFile,
				Score:       0.75,
			})
		}
	}

	// キャッシュに保存
	ac.cache.setGitCache(candidates)

	return candidates
}

// ファイル補完
func (ac *AdvancedCompleter) getFileCompletions(input string) []CompletionCandidate {
	var candidates []CompletionCandidate

	// キャッシュチェック
	dir := filepath.Dir(input)
	if cached := ac.cache.getFileCache(dir); cached != nil {
		return ac.filterFileCandidates(cached, input)
	}

	// ディレクトリ内容を取得
	entries, err := os.ReadDir(dir)
	if err != nil {
		// 現在ディレクトリを試行
		entries, err = os.ReadDir(".")
		if err != nil {
			return candidates
		}
		dir = "."
	}

	prefix := filepath.Base(input)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			fullPath := filepath.Join(dir, entry.Name())

			var description string
			var score float64 = 0.7

			if entry.IsDir() {
				description = "ディレクトリ"
				fullPath += "/"
				score = 0.75
			} else {
				// ファイル拡張子による説明
				ext := filepath.Ext(entry.Name())
				description = getFileTypeDescription(ext)

				// プロジェクト関連ファイルは優先度高
				if ac.isProjectRelatedFile(entry.Name()) {
					score = 0.9
				}
			}

			candidates = append(candidates, CompletionCandidate{
				Text:        fullPath,
				Description: description,
				Type:        CompletionFile,
				Score:       score,
			})
		}
	}

	// キャッシュに保存
	filenames := make([]string, len(candidates))
	for i, candidate := range candidates {
		filenames[i] = candidate.Text
	}
	ac.cache.setFileCache(dir, filenames)

	return candidates
}

// コマンド補完
func (ac *AdvancedCompleter) getCommandCompletions(input string) []CompletionCandidate {
	var candidates []CompletionCandidate

	commands := map[string]string{
		"help":     "ヘルプ表示",
		"analyze":  "プロジェクト解析",
		"build":    "プロジェクトビルド",
		"test":     "テスト実行",
		"status":   "ステータス表示",
		"search":   "ファイル検索",
		"find":     "ファイル検索",
		"grep":     "文字列検索",
		"git":      "Git操作",
		"config":   "設定管理",
		"sessions": "セッション管理",
		"exec":     "コマンド実行",
	}

	for cmd, desc := range commands {
		if strings.HasPrefix(cmd, input) {
			score := ac.calculateScore(input, cmd)
			candidates = append(candidates, CompletionCandidate{
				Text:        cmd,
				Description: desc,
				Type:        CompletionCommand,
				Score:       score,
			})
		}
	}

	return candidates
}

// プロジェクト固有コマンド補完
func (ac *AdvancedCompleter) getProjectCommandCompletions(input string) []CompletionCandidate {
	var candidates []CompletionCandidate

	// ビルドコマンド
	for _, cmd := range ac.projectAnalyzer.buildCommands {
		if strings.Contains(strings.ToLower(cmd), strings.ToLower(input)) {
			candidates = append(candidates, CompletionCandidate{
				Text:        cmd,
				Description: "プロジェクトビルドコマンド",
				Type:        CompletionProjectCommand,
				Score:       0.85,
			})
		}
	}

	// テストコマンド
	for _, cmd := range ac.projectAnalyzer.testCommands {
		if strings.Contains(strings.ToLower(cmd), strings.ToLower(input)) {
			candidates = append(candidates, CompletionCandidate{
				Text:        cmd,
				Description: "プロジェクトテストコマンド",
				Type:        CompletionProjectCommand,
				Score:       0.85,
			})
		}
	}

	return candidates
}

// ファジー補完
func (ac *AdvancedCompleter) getFuzzyCompletions(input string) []CompletionCandidate {
	var candidates []CompletionCandidate

	// 既存の全候補を取得してファジーマッチング
	allCandidates := []string{
		"help", "build", "test", "analyze", "status", "search", "find", "grep",
		"git", "config", "sessions", "exec", "clear", "history", "info",
	}

	// プロジェクト固有も追加
	allCandidates = append(allCandidates, ac.projectAnalyzer.buildCommands...)
	allCandidates = append(allCandidates, ac.projectAnalyzer.testCommands...)

	for _, candidate := range allCandidates {
		score := ac.fuzzyMatcher.match(input, candidate)
		if score >= ac.fuzzyMatcher.threshold {
			candidates = append(candidates, CompletionCandidate{
				Text:        candidate,
				Description: "ファジーマッチング",
				Type:        CompletionCommand,
				Score:       score,
			})
		}
	}

	return candidates
}

// スコア計算
func (ac *AdvancedCompleter) calculateScore(input, candidate string) float64 {
	if input == candidate {
		return 1.0
	}

	if strings.HasPrefix(candidate, input) {
		return 0.9 + 0.1*(float64(len(input))/float64(len(candidate)))
	}

	if strings.Contains(candidate, input) {
		return 0.7
	}

	return 0.0
}

// パスライクな文字列かチェック
func (ac *AdvancedCompleter) isPathLike(input string) bool {
	return strings.Contains(input, "/") || strings.Contains(input, "\\") || strings.HasPrefix(input, ".")
}

// プロジェクト関連ファイルかチェック
func (ac *AdvancedCompleter) isProjectRelatedFile(filename string) bool {
	projectFiles := []string{
		"go.mod", "go.sum", "Cargo.toml", "package.json", "pom.xml",
		"Makefile", "Dockerfile", "README.md", "CHANGELOG.md",
		".gitignore", ".env", ".env.example",
	}

	for _, pf := range projectFiles {
		if filename == pf {
			return true
		}
	}

	// 拡張子チェック
	ext := filepath.Ext(filename)
	importantExts := []string{".go", ".rs", ".js", ".ts", ".py", ".java", ".c", ".cpp", ".h"}
	for _, ie := range importantExts {
		if ext == ie {
			return true
		}
	}

	return false
}

// 重複除去
func (ac *AdvancedCompleter) removeDuplicates(candidates []CompletionCandidate) []CompletionCandidate {
	seen := make(map[string]bool)
	var result []CompletionCandidate

	for _, candidate := range candidates {
		if !seen[candidate.Text] {
			seen[candidate.Text] = true
			result = append(result, candidate)
		}
	}

	return result
}

// ファジーマッチング実装
func (fm *FuzzyMatcher) match(input, target string) float64 {
	if input == "" {
		return 0.0
	}

	input = strings.ToLower(input)
	target = strings.ToLower(target)

	if input == target {
		return 1.0
	}

	if strings.HasPrefix(target, input) {
		return 0.9
	}

	if strings.Contains(target, input) {
		return 0.7
	}

	// レーベンシュタイン距離による類似度計算
	return fm.levenshteinSimilarity(input, target)
}

// レーベンシュタイン距離による類似度
func (fm *FuzzyMatcher) levenshteinSimilarity(s1, s2 string) float64 {
	len1, len2 := len(s1), len(s2)
	if len1 == 0 {
		return 0.0
	}
	if len2 == 0 {
		return 0.0
	}

	// 動的プログラミングによる距離計算
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // 削除
				matrix[i][j-1]+1,      // 挿入
				matrix[i-1][j-1]+cost, // 置換
			)
		}
	}

	distance := matrix[len1][len2]
	maxLen := max(len1, len2)

	similarity := 1.0 - float64(distance)/float64(maxLen)
	return similarity
}

// ヘルパー関数
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func getGitActionDescription(action string) string {
	descriptions := map[string]string{
		"checkout": "チェックアウト",
		"switch":   "切り替え",
		"merge":    "マージ",
		"add":      "ステージング",
		"rm":       "削除",
		"restore":  "復元",
		"diff":     "差分表示",
		"show":     "表示",
		"log":      "ログ表示",
	}

	if desc, ok := descriptions[action]; ok {
		return desc
	}
	return "操作"
}

func getFileTypeDescription(ext string) string {
	descriptions := map[string]string{
		".go":   "Goソースファイル",
		".rs":   "Rustソースファイル",
		".js":   "JavaScriptファイル",
		".ts":   "TypeScriptファイル",
		".py":   "Pythonファイル",
		".java": "Javaファイル",
		".c":    "Cソースファイル",
		".cpp":  "C++ソースファイル",
		".h":    "ヘッダーファイル",
		".md":   "Markdownファイル",
		".json": "JSONファイル",
		".yaml": "YAMLファイル",
		".yml":  "YAMLファイル",
		".toml": "TOMLファイル",
		".txt":  "テキストファイル",
		".log":  "ログファイル",
	}

	if desc, ok := descriptions[ext]; ok {
		return desc
	}
	return "ファイル"
}

// キャッシュ操作
func (cc *CompletionCache) getFileCache(dir string) []CompletionCandidate {
	if entry, exists := cc.fileCache[dir]; exists {
		if time.Since(entry.timestamp) < cc.maxAge {
			var candidates []CompletionCandidate
			for _, text := range entry.data {
				candidates = append(candidates, CompletionCandidate{
					Text:  text,
					Type:  CompletionFile,
					Score: 0.7,
				})
			}
			return candidates
		}
		// 期限切れエントリを削除
		delete(cc.fileCache, dir)
	}
	return nil
}

func (cc *CompletionCache) setFileCache(dir string, files []string) {
	cc.fileCache[dir] = &CacheEntry{
		data:      files,
		timestamp: time.Now(),
	}

	// キャッシュサイズ制限
	if len(cc.fileCache) > cc.maxSize {
		// 古いエントリから削除（簡易実装）
		for k := range cc.fileCache {
			delete(cc.fileCache, k)
			if len(cc.fileCache) <= cc.maxSize/2 {
				break
			}
		}
	}
}

func (cc *CompletionCache) getGitCache() []CompletionCandidate {
	if cc.gitCache != nil && time.Since(cc.gitCache.timestamp) < cc.maxAge {
		var candidates []CompletionCandidate
		for _, text := range cc.gitCache.data {
			candidates = append(candidates, CompletionCandidate{
				Text:  text,
				Type:  CompletionGitBranch,
				Score: 0.8,
			})
		}
		return candidates
	}
	return nil
}

func (cc *CompletionCache) setGitCache(candidates []CompletionCandidate) {
	var data []string
	for _, candidate := range candidates {
		data = append(data, candidate.Text)
	}

	cc.gitCache = &CacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
}

// Git補完の具体的実装
func (gc *GitCompleter) getBranches() []string {
	// 簡易実装：実際のgitコマンドは呼ばずサンプルデータを返す
	// 本番環境では exec.Command("git", "branch") を使用
	return []string{"main", "develop", "feature/new-feature", "bugfix/issue-123"}
}

func (gc *GitCompleter) getModifiedFiles() []string {
	// 簡易実装：実際のgitコマンドは呼ばずサンプルデータを返す
	// 本番環境では exec.Command("git", "status", "--porcelain") を使用
	return []string{"internal/input/reader.go", "internal/input/security.go", "README.md"}
}

func (gc *GitCompleter) getTrackedFiles() []string {
	// 簡易実装：実際のgitコマンドは呼ばずサンプルデータを返す
	// 本番環境では exec.Command("git", "ls-files") を使用
	return []string{
		"go.mod", "go.sum", "main.go", "README.md",
		"internal/input/reader.go", "internal/input/security.go",
		"cmd/vyb/main.go", "pkg/types/types.go",
	}
}

// プロジェクト解析の具体的実装
func (pa *ProjectAnalyzer) analyze() {
	pa.detectProjectType()
	pa.findBuildCommands()
	pa.findTestCommands()
	pa.findMainFiles()
	pa.analyzeDependencies()
}

func (pa *ProjectAnalyzer) detectProjectType() {
	// Go プロジェクトチェック
	if _, err := os.Stat(filepath.Join(pa.workDir, "go.mod")); err == nil {
		pa.projectType = "go"
		return
	}

	// Node.js プロジェクトチェック
	if _, err := os.Stat(filepath.Join(pa.workDir, "package.json")); err == nil {
		pa.projectType = "nodejs"
		return
	}

	// Rust プロジェクトチェック
	if _, err := os.Stat(filepath.Join(pa.workDir, "Cargo.toml")); err == nil {
		pa.projectType = "rust"
		return
	}

	// Python プロジェクトチェック
	if _, err := os.Stat(filepath.Join(pa.workDir, "requirements.txt")); err == nil {
		pa.projectType = "python"
		return
	}
	if _, err := os.Stat(filepath.Join(pa.workDir, "pyproject.toml")); err == nil {
		pa.projectType = "python"
		return
	}

	// デフォルト
	pa.projectType = "generic"
}

func (pa *ProjectAnalyzer) findBuildCommands() {
	switch pa.projectType {
	case "go":
		pa.buildCommands = []string{"go build", "go install", "make build", "make"}

		// Makefileが存在する場合は追加
		if _, err := os.Stat(filepath.Join(pa.workDir, "Makefile")); err == nil {
			pa.buildCommands = append(pa.buildCommands, "make all", "make install")
		}

	case "nodejs":
		pa.buildCommands = []string{"npm run build", "npm install", "yarn build", "yarn install"}

		// package.jsonからスクリプトを読み取る（簡易実装）
		pa.buildCommands = append(pa.buildCommands, "npm start", "npm run dev")

	case "rust":
		pa.buildCommands = []string{"cargo build", "cargo build --release", "cargo install"}

	case "python":
		pa.buildCommands = []string{"python setup.py build", "pip install -e .", "python -m build"}

	default:
		pa.buildCommands = []string{"make", "make build", "./build.sh"}
	}
}

func (pa *ProjectAnalyzer) findTestCommands() {
	switch pa.projectType {
	case "go":
		pa.testCommands = []string{"go test", "go test ./...", "make test", "go test -v"}

	case "nodejs":
		pa.testCommands = []string{"npm test", "npm run test", "yarn test", "jest"}

	case "rust":
		pa.testCommands = []string{"cargo test", "cargo test --release"}

	case "python":
		pa.testCommands = []string{"python -m pytest", "pytest", "python -m unittest"}

	default:
		pa.testCommands = []string{"make test", "./test.sh", "test"}
	}
}

func (pa *ProjectAnalyzer) findMainFiles() {
	switch pa.projectType {
	case "go":
		pa.mainFiles = []string{"main.go", "cmd/main.go", "app.go"}

		// cmd/ ディレクトリ内を検索
		cmdDir := filepath.Join(pa.workDir, "cmd")
		if entries, err := os.ReadDir(cmdDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					mainFile := filepath.Join("cmd", entry.Name(), "main.go")
					pa.mainFiles = append(pa.mainFiles, mainFile)
				}
			}
		}

	case "nodejs":
		pa.mainFiles = []string{"index.js", "app.js", "server.js", "src/index.js"}

	case "rust":
		pa.mainFiles = []string{"src/main.rs", "src/lib.rs"}

	case "python":
		pa.mainFiles = []string{"main.py", "app.py", "__main__.py", "src/main.py"}

	default:
		pa.mainFiles = []string{"main", "app", "index"}
	}
}

func (pa *ProjectAnalyzer) analyzeDependencies() {
	switch pa.projectType {
	case "go":
		// go.mod から依存関係を解析（簡易実装）
		pa.dependencies = []string{
			"golang.org/x/term", "github.com/spf13/cobra",
			"github.com/spf13/viper", "gopkg.in/yaml.v3",
		}

	case "nodejs":
		// package.json から依存関係を解析（簡易実装）
		pa.dependencies = []string{"react", "express", "lodash", "axios"}

	case "rust":
		// Cargo.toml から依存関係を解析（簡易実装）
		pa.dependencies = []string{"serde", "tokio", "clap", "anyhow"}

	case "python":
		// requirements.txt から依存関係を解析（簡易実装）
		pa.dependencies = []string{"requests", "numpy", "pandas", "flask"}
	}
}

// キャッシュフィルタリング関数
func (ac *AdvancedCompleter) filterGitCandidates(cached []CompletionCandidate, input string) []CompletionCandidate {
	var filtered []CompletionCandidate

	for _, candidate := range cached {
		if strings.Contains(strings.ToLower(candidate.Text), strings.ToLower(input)) {
			candidate.Score = ac.calculateScore(input, candidate.Text)
			filtered = append(filtered, candidate)
		}
	}

	return filtered
}

func (ac *AdvancedCompleter) filterFileCandidates(cached []CompletionCandidate, input string) []CompletionCandidate {
	var filtered []CompletionCandidate

	prefix := filepath.Base(input)
	for _, candidate := range cached {
		filename := filepath.Base(candidate.Text)
		if strings.HasPrefix(filename, prefix) {
			candidate.Score = ac.calculateScore(prefix, filename)
			filtered = append(filtered, candidate)
		}
	}

	return filtered
}

