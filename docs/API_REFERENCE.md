# API リファレンス

## 概要

vyb-codeプロジェクトの主要APIについて詳細に説明します。このドキュメントでは、開発者がvyb-codeを拡張・カスタマイズするために必要な情報を提供します。

## セキュリティAPI

### LLM応答検証

#### `LLMResponseValidator`

**パッケージ**: `internal/security`

LLMの応答を検証し、悪意あるコンテンツをフィルタリングします。

```go
// インスタンス作成
validator := security.NewLLMResponseValidator()

// レスポンス検証
result, err := validator.ValidateResponse(content)
if err != nil {
    log.Fatal(err)
}

// 結果の確認
if !result.IsValid {
    log.Printf("危険なコンテンツを検出: %v", result.DetectedThreats)
}
```

#### `LLMValidationResult`

```go
type LLMValidationResult struct {
    IsValid         bool     `json:"is_valid"`
    RiskLevel       string   `json:"risk_level"`       // "safe", "warning", "dangerous"
    DetectedThreats []string `json:"detected_threats"`
    FilteredContent string   `json:"filtered_content"`
    RequiresReview  bool     `json:"requires_review"`
    SecurityScore   float64  `json:"security_score"`
}
```

#### メソッド

| メソッド | 説明 | 戻り値 |
|---------|------|--------|
| `ValidateResponse(content string)` | コンテンツを検証 | `*LLMValidationResult, error` |
| `FilterResponse(content string)` | 危険部分をマスク | `string` |
| `SetSecurityLevel(level string)` | セキュリティレベル設定 | `error` |
| `SetAllowCodeGeneration(allow bool)` | コード生成許可設定 | - |

### ファイルアクセス制御

#### `Constraints`

**パッケージ**: `internal/security`

ファイルアクセスとコマンド実行を制御します。

```go
// デフォルト制約を作成
constraints := security.NewDefaultConstraints("/workspace")

// ファイルアクセス検証
err := constraints.IsFileAccessAllowed("/path/to/file", "read")
if err != nil {
    log.Printf("ファイルアクセス拒否: %v", err)
}

// コマンド実行検証
err = constraints.ValidateCommand("ls -la")
if err != nil {
    log.Printf("コマンド実行拒否: %v", err)
}
```

#### 主要フィールド

```go
type Constraints struct {
    AllowedCommands   []string
    BlockedCommands   []string
    MaxTimeout        int
    WorkspaceDir      string
    AllowedExtensions []string
    BlockedPaths      []string
    MaxFileSize       int64
    ReadOnlyMode      bool
}
```

#### メソッド

| メソッド | 説明 | 戻り値 |
|---------|------|--------|
| `IsFileAccessAllowed(path, operation)` | ファイルアクセス検証 | `error` |
| `ValidateCommand(command)` | コマンド検証 | `error` |
| `IsPathAllowed(path)` | パス検証 | `bool` |
| `ValidateFileContent(content, path)` | ファイル内容検証 | `error` |
| `SetReadOnlyMode(readOnly)` | 読み取り専用設定 | - |

### エラーハンドリング

#### `ErrorHandler`

**パッケージ**: `internal/security`

高度なエラー処理と復旧機能を提供します。

```go
// エラーハンドラー作成
logger := // your logger implementation
handler := security.NewErrorHandler(logger)

// エラー処理
enhancedErr := handler.HandleError(
    err, 
    security.ErrorCategorySecurity, 
    security.ErrorSeverityHigh
)

// 復旧手順の表示
fmt.Println(enhancedErr.String())
```

#### `EnhancedError`

```go
type EnhancedError struct {
    OriginalError    error
    Category         ErrorCategory
    Severity         ErrorSeverity
    Code             string
    Message          string
    UserMessage      string
    TechnicalDetails string
    Context          map[string]string
    RecoveryStrategy RecoveryStrategy
    RecoverySteps    []string
    RetryCount       int
    MaxRetries       int
}
```

## パフォーマンスAPI

### メモリ最適化

#### `MemoryOptimizer`

**パッケージ**: `internal/performance`

メモリ使用量を最適化し、データを自動圧縮します。

```go
// 設定作成
config := performance.DefaultMemoryConfig()
config.MaxMemoryUsage = 100 * 1024 * 1024 // 100MB

// オプティマイザー作成
optimizer := performance.NewMemoryOptimizer(config)

// データ圧縮
compressibleData := &performance.CompressibleSession{
    ID:   "session-123",
    Data: sessionData,
}
err := optimizer.CompressData(compressibleData)
```

#### `MemoryConfig`

```go
type MemoryConfig struct {
    MaxSessionSize       int64
    MaxTurnsPerSession   int
    CompressionThreshold int64
    CompressionRatio     float64
    CleanupInterval      time.Duration
    RetentionPeriod      time.Duration
    MaxMemoryUsage       int64
}
```

#### メソッド

| メソッド | 説明 | 戻り値 |
|---------|------|--------|
| `CompressData(obj Compressible)` | データ圧縮 | `error` |
| `DecompressData(id, obj)` | データ解凍 | `error` |
| `GetMemoryStats()` | メモリ統計取得 | `*MemoryStats` |
| `GenerateMemoryReport()` | レポート生成 | `map[string]interface{}` |

### 並行処理管理

#### `ConcurrencyManager`

**パッケージ**: `internal/performance`

並行処理を効率的に管理します。

```go
// 設定作成
config := performance.DefaultConcurrencyConfig()
config.MaxWorkers = 8

// マネージャー作成・開始
manager := performance.NewConcurrencyManager(config)
err := manager.Start()
if err != nil {
    log.Fatal(err)
}

// タスク実行
task := &performance.Task{
    ID:       "task-1",
    Priority: performance.PriorityHigh,
    Function: func(ctx context.Context) (interface{}, error) {
        // タスクの処理
        return "result", nil
    },
}

err = manager.SubmitTask(task)
```

#### `Task`

```go
type Task struct {
    ID          string
    Priority    TaskPriority
    Function    func(ctx context.Context) (interface{}, error)
    Context     context.Context
    Retry       int
    CreatedAt   time.Time
    Result      interface{}
    Error       error
    Metadata    map[string]interface{}
}
```

#### メソッド

| メソッド | 説明 | 戻り値 |
|---------|------|--------|
| `Start()` | 並行処理開始 | `error` |
| `Stop()` | 並行処理停止 | `error` |
| `SubmitTask(task)` | タスク提出 | `error` |
| `ExecuteParallel(tasks)` | 並列実行 | `([]*Task, error)` |
| `GetStats()` | 統計取得 | `*ConcurrencyStats` |
| `HealthCheck()` | ヘルスチェック | `map[string]interface{}` |

## ツールAPI

### Claude Code互換ツール

#### `BashTool`

**パッケージ**: `internal/tools`

セキュアなコマンド実行を提供します。

```go
// Bashツール作成
constraints := security.NewDefaultConstraints("/workspace")
bashTool := tools.NewBashTool(constraints, "/workspace")

// コマンド実行
result, err := bashTool.Execute("ls -la", "ディレクトリ一覧表示", 30000)
if err != nil {
    log.Printf("コマンド実行エラー: %v", err)
}

fmt.Printf("出力: %s\n", result.Content)
fmt.Printf("終了コード: %d\n", result.ExitCode)
```

#### `GrepTool`

```go
// Grepツール作成
grepTool := tools.NewGrepTool("/workspace")

// 検索実行
options := tools.GrepOptions{
    Pattern:      "func.*Error",
    Glob:         "*.go",
    OutputMode:   "content",
    LineNumbers:  true,
    HeadLimit:    10,
}

result, err := grepTool.Search(options)
```

#### `ToolExecutionResult`

```go
type ToolExecutionResult struct {
    Content  string
    IsError  bool
    Tool     string
    ExitCode int
    Duration string
    TimedOut bool
    Metadata map[string]interface{}
}
```

## 設定API

### 設定管理

#### `Config`

**パッケージ**: `internal/config`

アプリケーション設定を管理します。

```go
// 設定読み込み
config, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// プロアクティブレベル設定
err = config.SetProactiveLevel(config.ProactiveLevelStandard)
if err != nil {
    log.Printf("設定エラー: %v", err)
}

// セキュリティレベル設定
config.Features.ProactiveMode = true
config.Proactive.Enabled = true
```

#### 主要設定項目

```go
type Config struct {
    // LLM設定
    Provider    string
    Model       string
    BaseURL     string
    Timeout     int
    
    // セキュリティ設定
    MaxFileSize   int64
    WorkspaceMode string
    
    // 機能設定
    Features   *Features
    Proactive  ProactiveConfig
    TUI        TUIConfig
}
```

## 使用例

### 基本的な使用パターン

#### 1. セキュアなファイル操作

```go
// セキュリティ制約を設定
constraints := security.NewDefaultConstraints("/workspace")
constraints.SetMaxFileSize(5 * 1024 * 1024) // 5MB制限

// ファイル読み取り前の検証
err := constraints.IsFileAccessAllowed("/workspace/file.go", "read")
if err != nil {
    return fmt.Errorf("ファイルアクセス拒否: %w", err)
}

// ファイル内容の検証
content, err := os.ReadFile("/workspace/file.go")
if err == nil {
    err = constraints.ValidateFileContent(content, "/workspace/file.go")
    if err != nil {
        return fmt.Errorf("ファイル内容検証エラー: %w", err)
    }
}
```

#### 2. LLM応答の安全な処理

```go
// 検証器を設定
validator := security.NewLLMResponseValidator()
validator.SetSecurityLevel("moderate")

// LLM応答を検証
response := "# LLMからの応答内容"
result, err := validator.ValidateResponse(response)
if err != nil {
    return err
}

if !result.IsValid {
    log.Printf("危険なコンテンツを検出: %v", result.DetectedThreats)
    response = result.FilteredContent // フィルタリング済みコンテンツを使用
}
```

#### 3. 並行処理の実装

```go
// 並行処理マネージャーを起動
config := performance.DefaultConcurrencyConfig()
manager := performance.NewConcurrencyManager(config)
manager.Start()
defer manager.Stop()

// 複数タスクの並列実行
tasks := []*performance.Task{
    {
        ID:       "task1",
        Priority: performance.PriorityHigh,
        Function: func(ctx context.Context) (interface{}, error) {
            return processFile("file1.go"), nil
        },
    },
    {
        ID:       "task2", 
        Priority: performance.PriorityNormal,
        Function: func(ctx context.Context) (interface{}, error) {
            return processFile("file2.go"), nil
        },
    },
}

results, err := manager.ExecuteParallel(tasks)
if err != nil {
    log.Printf("並列実行エラー: %v", err)
}
```

## エラー処理ベストプラクティス

### 1. 適切なエラーカテゴリの選択

```go
// セキュリティエラー
if isSecurityViolation(err) {
    handler.HandleError(err, security.ErrorCategorySecurity, security.ErrorSeverityHigh)
}

// ネットワークエラー
if isNetworkError(err) {
    handler.HandleError(err, security.ErrorCategoryNetwork, security.ErrorSeverityMedium)
}
```

### 2. 復旧手順の実装

```go
enhancedErr := handler.HandleError(err, category, severity)

if enhancedErr.RecoveryStrategy == security.RecoveryStrategyRetry {
    // 再試行ロジック
    if enhancedErr.RetryCount < enhancedErr.MaxRetries {
        time.Sleep(time.Duration(enhancedErr.RetryCount) * time.Second)
        // 再試行実行
    }
}
```

## 監視とログ

### パフォーマンス監視

```go
// メモリ統計の取得
memStats := optimizer.GetMemoryStats()
log.Printf("メモリ使用量: %d bytes", memStats.MemoryUsage)
log.Printf("圧縮率: %.2f%%", memStats.CompressionRatio*100)

// 並行処理統計の取得
concStats := manager.GetStats()
log.Printf("アクティブワーカー: %d", concStats.ActiveWorkers)
log.Printf("成功率: %.2f%%", float64(concStats.SuccessfulTasks)/float64(concStats.ProcessedTasks)*100)
```

### ヘルスチェック

```go
// システム全体の健全性チェック
healthData := manager.HealthCheck()
if healthData["status"] != "healthy" {
    log.Printf("システム警告: %v", healthData["warning"])
}

// メモリアラートチェック
alerts := optimizer.CheckMemoryAlerts()
for _, alert := range alerts {
    log.Printf("メモリアラート: %s", alert)
}
```

## まとめ

これらのAPIを使用することで、vyb-codeにClaude Code相当のセキュリティ、パフォーマンス、機能性を実装できます。各APIは独立して使用でき、必要に応じて組み合わせることが可能です。

詳細な実装例については、各パッケージのテストファイルも参照してください。