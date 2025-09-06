package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glkt/vyb-code/internal/security"
)

// ビルド実行結果
type BuildResult struct {
	Success           bool                   `json:"success"`
	Command           string                 `json:"command"`
	Output            string                 `json:"output"`
	ErrorOutput       string                 `json:"error_output"`
	ExitCode          int                    `json:"exit_code"`
	Duration          time.Duration          `json:"duration"`
	BuildSystem       string                 `json:"build_system"`
	Target            string                 `json:"target,omitempty"`
	Artifacts         []string               `json:"artifacts,omitempty"`
	Warnings          []string               `json:"warnings,omitempty"`
	Recommendations   []string               `json:"recommendations,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// ビルドパイプライン設定
type BuildPipeline struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Steps       []BuildStep     `json:"steps"`
	Environment map[string]string `json:"environment,omitempty"`
	Parallel    bool            `json:"parallel"`
	OnFailure   string          `json:"on_failure"` // "stop", "continue", "retry"
}

// ビルドステップ
type BuildStep struct {
	Name         string            `json:"name"`
	Command      string            `json:"command"`
	Args         []string          `json:"args,omitempty"`
	WorkingDir   string            `json:"working_dir,omitempty"`
	Environment  map[string]string `json:"environment,omitempty"`
	Timeout      time.Duration     `json:"timeout,omitempty"`
	RetryCount   int               `json:"retry_count,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	Condition    string            `json:"condition,omitempty"` // 実行条件
}

// ビルドパフォーマンス統計
type BuildPerformance struct {
	TotalDuration     time.Duration     `json:"total_duration"`
	StepDurations     map[string]time.Duration `json:"step_durations"`
	CacheHits         int               `json:"cache_hits"`
	CacheMisses       int               `json:"cache_misses"`
	ParallelEfficiency float64          `json:"parallel_efficiency"`
	Suggestions       []string          `json:"suggestions"`
}

// ビルドキャッシュ情報
type BuildCache struct {
	Enabled         bool              `json:"enabled"`
	CacheDirectory  string            `json:"cache_directory"`
	CacheSize       int64             `json:"cache_size_bytes"`
	LastCleanup     time.Time         `json:"last_cleanup"`
	HitRate         float64           `json:"hit_rate"`
	CachedArtifacts map[string]CachedArtifact `json:"cached_artifacts"`
}

// キャッシュされたアーティファクト
type CachedArtifact struct {
	Path         string    `json:"path"`
	Hash         string    `json:"hash"`
	Size         int64     `json:"size"`
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
	AccessCount  int       `json:"access_count"`
}

// ビルドマネージャー
type BuildManager struct {
	projectDir      string
	constraints     *security.Constraints
	executor        *CommandExecutor
	languageManager *LanguageManager
	analyzer        *AdvancedProjectAnalyzer
	cache           *BuildCache
	performance     *BuildPerformance
}

// ビルドマネージャーを作成
func NewBuildManager(constraints *security.Constraints, projectDir string) *BuildManager {
	manager := &BuildManager{
		projectDir:      projectDir,
		constraints:     constraints,
		executor:        NewCommandExecutor(constraints, projectDir),
		languageManager: NewLanguageManager(),
		analyzer:        NewAdvancedProjectAnalyzer(constraints, projectDir),
		cache: &BuildCache{
			Enabled:         true,
			CacheDirectory:  filepath.Join(projectDir, ".vyb-cache"),
			CachedArtifacts: make(map[string]CachedArtifact),
		},
		performance: &BuildPerformance{
			StepDurations: make(map[string]time.Duration),
			Suggestions:   []string{},
		},
	}

	// キャッシュディレクトリを作成
	os.MkdirAll(manager.cache.CacheDirectory, 0755)

	return manager
}

// 自動ビルド実行
func (bm *BuildManager) AutoBuild() (*BuildResult, error) {
	// プロジェクト解析
	analysis, err := bm.analyzer.AnalyzeAdvanced()
	if err != nil {
		return nil, fmt.Errorf("プロジェクト解析エラー: %w", err)
	}

	// 最適なビルドシステムを選択
	buildSystem := bm.selectOptimalBuildSystem(analysis.BuildSystems)
	if buildSystem == nil {
		return nil, fmt.Errorf("利用可能なビルドシステムが見つかりません")
	}

	// ビルド実行
	return bm.executeBuildSystem(buildSystem)
}

// 特定のビルドシステムでビルド実行
func (bm *BuildManager) BuildWithSystem(systemType string, target string) (*BuildResult, error) {
	analysis, err := bm.analyzer.AnalyzeAdvanced()
	if err != nil {
		return nil, fmt.Errorf("プロジェクト解析エラー: %w", err)
	}

	// 指定されたビルドシステムを検索
	var targetSystem *BuildSystemInfo
	for _, system := range analysis.BuildSystems {
		if system.Type == systemType {
			targetSystem = &system
			break
		}
	}

	if targetSystem == nil {
		return nil, fmt.Errorf("ビルドシステム '%s' が見つかりません", systemType)
	}

	return bm.executeBuildSystemWithTarget(targetSystem, target)
}

// ビルドパイプライン実行
func (bm *BuildManager) ExecutePipeline(pipeline *BuildPipeline) ([]*BuildResult, error) {
	var results []*BuildResult
	startTime := time.Now()

	fmt.Printf("ビルドパイプライン '%s' を開始します\n", pipeline.Name)

	if pipeline.Parallel && len(pipeline.Steps) > 1 {
		// 並列実行
		results, err := bm.executeStepsParallel(pipeline.Steps, pipeline.Environment)
		if err != nil {
			return results, err
		}
	} else {
		// 順次実行
		for _, step := range pipeline.Steps {
			result, err := bm.executeStep(&step, pipeline.Environment)
			results = append(results, result)

			if err != nil || !result.Success {
				if pipeline.OnFailure == "stop" {
					fmt.Printf("ステップ '%s' が失敗したため、パイプラインを停止します\n", step.Name)
					break
				} else if pipeline.OnFailure == "retry" && step.RetryCount > 0 {
					fmt.Printf("ステップ '%s' をリトライします\n", step.Name)
					// リトライ実装
					for i := 0; i < step.RetryCount; i++ {
						retryResult, retryErr := bm.executeStep(&step, pipeline.Environment)
						if retryErr == nil && retryResult.Success {
							results[len(results)-1] = retryResult
							break
						}
						if i == step.RetryCount-1 {
							return results, fmt.Errorf("ステップ '%s' が %d 回リトライ後も失敗しました", step.Name, step.RetryCount)
						}
						time.Sleep(time.Second * time.Duration(i+1)) // exponential backoff
					}
				}
			}
		}
	}

	// パフォーマンス統計を更新
	bm.performance.TotalDuration = time.Since(startTime)
	bm.calculateParallelEfficiency(results, pipeline.Parallel)

	return results, nil
}

// 最適なビルドシステムを選択
func (bm *BuildManager) selectOptimalBuildSystem(systems []BuildSystemInfo) *BuildSystemInfo {
	if len(systems) == 0 {
		return nil
	}

	// 優先順位：Makefile > 言語固有 > Docker > CI
	priorities := map[string]int{
		"makefile":      100,
		"go_native":     90,
		"javascript_native": 85,
		"python_native": 85,
		"docker":        70,
		"github_actions": 50,
		"gitlab_ci":     50,
	}

	var bestSystem *BuildSystemInfo
	bestPriority := -1

	for _, system := range systems {
		if priority, exists := priorities[system.Type]; exists && priority > bestPriority {
			bestPriority = priority
			bestSystem = &system
		}
	}

	return bestSystem
}

// ビルドシステム実行
func (bm *BuildManager) executeBuildSystem(system *BuildSystemInfo) (*BuildResult, error) {
	return bm.executeBuildSystemWithTarget(system, "")
}

// ターゲット指定でビルドシステム実行
func (bm *BuildManager) executeBuildSystemWithTarget(system *BuildSystemInfo, target string) (*BuildResult, error) {
	startTime := time.Now()

	var command string
	var args []string

	switch system.Type {
	case "makefile":
		command = "make"
		if target != "" {
			args = []string{target}
		} else if len(system.Targets) > 0 {
			// デフォルトターゲットまたは最初のターゲットを使用
			for _, t := range system.Targets {
				if t == "build" || t == "all" || t == "default" {
					args = []string{t}
					break
				}
			}
			if len(args) == 0 {
				args = []string{system.Targets[0]}
			}
		}

	case "docker":
		command = "docker"
		args = []string{"build", ".", "-t", filepath.Base(bm.projectDir)}

	case "go_native":
		command = "go"
		if target == "test" {
			args = []string{"test", "./..."}
		} else {
			args = []string{"build", "./cmd/..."}
		}

	case "javascript_native":
		command = "npm"
		if target == "test" {
			args = []string{"test"}
		} else {
			args = []string{"run", "build"}
		}

	case "python_native":
		command = "python"
		if target == "test" {
			args = []string{"-m", "pytest"}
		} else {
			args = []string{"-m", "py_compile", "."}
		}

	default:
		return nil, fmt.Errorf("サポートされていないビルドシステム: %s", system.Type)
	}

	// コマンド実行
	fullCommand := command
	if len(args) > 0 {
		fullCommand = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}
	
	result, err := bm.executor.Execute(fullCommand)
	if err != nil {
		return &BuildResult{
			Success:      false,
			Command:      fullCommand,
			ErrorOutput:  err.Error(),
			ExitCode:     -1,
			Duration:     time.Since(startTime),
			BuildSystem:  system.Type,
			Target:       target,
		}, err
	}

	// 結果を構築
	buildResult := &BuildResult{
		Success:     result.ExitCode == 0,
		Command:     fmt.Sprintf("%s %s", command, strings.Join(args, " ")),
		Output:      result.Stdout,
		ErrorOutput: result.Stderr,
		ExitCode:    result.ExitCode,
		Duration:    time.Since(startTime),
		BuildSystem: system.Type,
		Target:      target,
		Metadata:    make(map[string]interface{}),
	}

	// アーティファクトを検出
	buildResult.Artifacts = bm.detectBuildArtifacts(system.Type)

	// 警告を解析
	buildResult.Warnings = bm.parseWarnings(result.Stderr, system.Type)

	// 推奨事項を生成
	buildResult.Recommendations = bm.generateBuildRecommendations(buildResult, system)

	// パフォーマンス統計を更新
	bm.performance.StepDurations[system.Type] = buildResult.Duration

	return buildResult, nil
}

// ビルドステップ実行
func (bm *BuildManager) executeStep(step *BuildStep, globalEnv map[string]string) (*BuildResult, error) {
	startTime := time.Now()

	// 環境変数をマージ
	env := make(map[string]string)
	for k, v := range globalEnv {
		env[k] = v
	}
	for k, v := range step.Environment {
		env[k] = v
	}

	// 作業ディレクトリを設定（将来の拡張用）
	// workingDir := bm.projectDir
	// if step.WorkingDir != "" {
	//   workingDir = filepath.Join(bm.projectDir, step.WorkingDir)
	// }

	// タイムアウト設定（将来の拡張用）
	// timeout := time.Duration(bm.constraints.MaxTimeout) * time.Second
	// if step.Timeout > 0 {
	//   timeout = step.Timeout
	// }

	// コマンド実行
	fullCommand := step.Command
	if len(step.Args) > 0 {
		fullCommand = fmt.Sprintf("%s %s", step.Command, strings.Join(step.Args, " "))
	}
	
	// 環境変数を設定（現在のExecutorは環境変数設定をサポートしていないため、将来の拡張ポイント）
	// TODO: CommandExecutorに環境変数とワーキングディレクトリサポートを追加
	
	result, err := bm.executor.Execute(fullCommand)

	buildResult := &BuildResult{
		Success:     result.ExitCode == 0,
		Command:     fmt.Sprintf("%s %s", step.Command, strings.Join(step.Args, " ")),
		Output:      result.Stdout,
		ErrorOutput: result.Stderr,
		ExitCode:    result.ExitCode,
		Duration:    time.Since(startTime),
		BuildSystem: "custom_step",
		Target:      step.Name,
		Metadata:    make(map[string]interface{}),
	}

	if err != nil {
		buildResult.ErrorOutput = err.Error()
		return buildResult, err
	}

	// パフォーマンス統計を更新
	bm.performance.StepDurations[step.Name] = buildResult.Duration

	return buildResult, nil
}

// 並列ステップ実行
func (bm *BuildManager) executeStepsParallel(steps []BuildStep, globalEnv map[string]string) ([]*BuildResult, error) {
	resultChan := make(chan *BuildResult, len(steps))
	errorChan := make(chan error, len(steps))

	// 並列実行
	for _, step := range steps {
		go func(s BuildStep) {
			result, err := bm.executeStep(&s, globalEnv)
			if err != nil {
				errorChan <- err
			} else {
				resultChan <- result
			}
		}(step)
	}

	// 結果を収集
	var results []*BuildResult
	var errors []error

	for i := 0; i < len(steps); i++ {
		select {
		case result := <-resultChan:
			results = append(results, result)
		case err := <-errorChan:
			errors = append(errors, err)
		case <-time.After(time.Duration(bm.constraints.MaxTimeout) * time.Second):
			return results, fmt.Errorf("並列実行タイムアウト")
		}
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("並列実行中にエラー: %v", errors[0])
	}

	return results, nil
}

// ビルドアーティファクトを検出
func (bm *BuildManager) detectBuildArtifacts(buildSystemType string) []string {
	var artifacts []string
	
	commonArtifacts := map[string][]string{
		"go_native":     {"vyb", "vyb.exe", "cmd/*/main", "*.exe"},
		"javascript_native": {"dist/", "build/", "lib/", "*.bundle.js"},
		"python_native": {"dist/", "build/", "*.wheel", "*.egg"},
		"docker":       {"Dockerfile", "*.tar", "docker-compose.yml"},
		"makefile":     {"*.o", "*.a", "*.so", "bin/", "build/"},
	}

	patterns, exists := commonArtifacts[buildSystemType]
	if !exists {
		return artifacts
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(bm.projectDir, pattern))
		if err != nil {
			continue
		}

		for _, match := range matches {
			relPath, _ := filepath.Rel(bm.projectDir, match)
			artifacts = append(artifacts, relPath)
		}
	}

	return artifacts
}

// 警告を解析
func (bm *BuildManager) parseWarnings(stderr string, buildSystemType string) []string {
	var warnings []string

	warningPatterns := map[string][]string{
		"go_native": {
			`warning: .*`,
			`deprecated: .*`,
		},
		"javascript_native": {
			`WARNING: .*`,
			`warning .*`,
			`DEPRECATION: .*`,
		},
		"python_native": {
			`Warning: .*`,
			`DeprecationWarning: .*`,
		},
	}

	patterns, exists := warningPatterns[buildSystemType]
	if !exists {
		return warnings
	}

	lines := strings.Split(stderr, "\n")
	for _, line := range lines {
		for _, pattern := range patterns {
			// 簡易的なパターンマッチ（実際の実装では正規表現を使用）
			if strings.Contains(strings.ToLower(line), strings.ToLower(strings.Split(pattern, " ")[0])) {
				warnings = append(warnings, strings.TrimSpace(line))
				break
			}
		}
	}

	return warnings
}

// ビルド推奨事項を生成
func (bm *BuildManager) generateBuildRecommendations(result *BuildResult, system *BuildSystemInfo) []string {
	var recommendations []string

	// エラー分析による推奨
	if !result.Success {
		if strings.Contains(result.ErrorOutput, "permission denied") {
			recommendations = append(recommendations, "実行権限の問題があります。chmod +x で実行権限を付与してください")
		}
		if strings.Contains(result.ErrorOutput, "not found") {
			recommendations = append(recommendations, "必要なツールがインストールされていない可能性があります")
		}
		if strings.Contains(result.ErrorOutput, "out of memory") {
			recommendations = append(recommendations, "メモリ不足です。ビルドプロセスを最適化するか、メモリを増やしてください")
		}
	}

	// パフォーマンス分析による推奨
	if result.Duration > time.Minute*5 {
		recommendations = append(recommendations, "ビルド時間が長いです。キャッシュやインクリメンタルビルドの使用を検討してください")
	}

	// 警告分析による推奨
	if len(result.Warnings) > 10 {
		recommendations = append(recommendations, "警告が多数あります。コード品質向上のため対処することを推奨します")
	}

	// ビルドシステム固有の推奨
	switch system.Type {
	case "makefile":
		if len(system.Targets) < 3 {
			recommendations = append(recommendations, "Makefileに clean, test, install ターゲットの追加を検討してください")
		}
	case "docker":
		if !strings.Contains(result.Output, "COPY") {
			recommendations = append(recommendations, "Dockerマルチステージビルドを使用してイメージサイズを最適化できます")
		}
	}

	return recommendations
}

// 並列効率を計算
func (bm *BuildManager) calculateParallelEfficiency(results []*BuildResult, isParallel bool) {
	if !isParallel || len(results) < 2 {
		bm.performance.ParallelEfficiency = 0.0
		return
	}

	// 理論的な並列時間と実際の並列時間を比較
	totalSequentialTime := time.Duration(0)
	maxParallelTime := time.Duration(0)

	for _, result := range results {
		totalSequentialTime += result.Duration
		if result.Duration > maxParallelTime {
			maxParallelTime = result.Duration
		}
	}

	if maxParallelTime > 0 {
		bm.performance.ParallelEfficiency = float64(totalSequentialTime) / float64(maxParallelTime) / float64(len(results))
	}

	// パフォーマンス推奨事項
	if bm.performance.ParallelEfficiency < 0.5 {
		bm.performance.Suggestions = append(bm.performance.Suggestions,
			"並列効率が低いです。ステップ間の依存関係を見直してください")
	}
}

// プリセットパイプラインを作成
func (bm *BuildManager) CreatePresetPipeline(pipelineType string) (*BuildPipeline, error) {
	switch pipelineType {
	case "go_standard":
		return &BuildPipeline{
			Name:        "Go Standard Pipeline",
			Description: "Go プロジェクトの標準的なビルドパイプライン",
			Steps: []BuildStep{
				{Name: "dependencies", Command: "go", Args: []string{"mod", "download"}},
				{Name: "format_check", Command: "go", Args: []string{"fmt", "./..."}},
				{Name: "vet", Command: "go", Args: []string{"vet", "./..."}},
				{Name: "test", Command: "go", Args: []string{"test", "-v", "./..."}},
				{Name: "build", Command: "go", Args: []string{"build", "-o", "vyb", "./cmd/vyb"}},
			},
			Parallel:  false,
			OnFailure: "stop",
		}, nil

	case "javascript_standard":
		return &BuildPipeline{
			Name:        "JavaScript Standard Pipeline",  
			Description: "JavaScript/Node.js プロジェクトの標準的なビルドパイプライン",
			Steps: []BuildStep{
				{Name: "dependencies", Command: "npm", Args: []string{"install"}},
				{Name: "lint", Command: "npm", Args: []string{"run", "lint"}},
				{Name: "test", Command: "npm", Args: []string{"test"}},
				{Name: "build", Command: "npm", Args: []string{"run", "build"}},
			},
			Parallel:  false,
			OnFailure: "stop",
		}, nil

	case "full_ci":
		return &BuildPipeline{
			Name:        "Full CI Pipeline",
			Description: "完全なCI/CDパイプライン（複数言語対応）",
			Steps: []BuildStep{
				{Name: "prepare", Command: "echo", Args: []string{"Starting full CI pipeline"}},
				{Name: "security_scan", Command: "echo", Args: []string{"Security scan placeholder"}},
				{Name: "dependency_check", Command: "echo", Args: []string{"Dependency check placeholder"}},
				{Name: "build_all", Command: "make", Args: []string{"all"}},
				{Name: "integration_test", Command: "echo", Args: []string{"Integration test placeholder"}},
			},
			Parallel:  false,
			OnFailure: "continue",
		}, nil

	default:
		return nil, fmt.Errorf("不明なパイプラインタイプ: %s", pipelineType)
	}
}

// ビルドキャッシュを管理
func (bm *BuildManager) ManageCache() *BuildCache {
	// キャッシュサイズを計算
	bm.calculateCacheSize()

	// キャッシュヒット率を計算
	bm.calculateCacheHitRate()

	// 古いキャッシュをクリーンアップ
	bm.cleanupOldCache()

	return bm.cache
}

// キャッシュサイズを計算
func (bm *BuildManager) calculateCacheSize() {
	var totalSize int64

	filepath.Walk(bm.cache.CacheDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		totalSize += info.Size()
		return nil
	})

	bm.cache.CacheSize = totalSize
}

// キャッシュヒット率を計算
func (bm *BuildManager) calculateCacheHitRate() {
	total := bm.performance.CacheHits + bm.performance.CacheMisses
	if total > 0 {
		bm.cache.HitRate = float64(bm.performance.CacheHits) / float64(total)
	}
}

// 古いキャッシュをクリーンアップ
func (bm *BuildManager) cleanupOldCache() {
	cutoff := time.Now().AddDate(0, 0, -7) // 7日前

	for path, artifact := range bm.cache.CachedArtifacts {
		if artifact.LastAccessed.Before(cutoff) {
			os.Remove(filepath.Join(bm.cache.CacheDirectory, path))
			delete(bm.cache.CachedArtifacts, path)
		}
	}

	bm.cache.LastCleanup = time.Now()
}

// パフォーマンス統計をJSON形式で取得
func (bm *BuildManager) GetPerformanceStats() ([]byte, error) {
	return json.MarshalIndent(bm.performance, "", "  ")
}

// キャッシュ統計をJSON形式で取得
func (bm *BuildManager) GetCacheStats() ([]byte, error) {
	return json.MarshalIndent(bm.cache, "", "  ")
}