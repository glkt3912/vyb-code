package migration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
)

// ValidationResult - 検証結果
type ValidationResult struct {
	SystemName       string                 `json:"system_name"`
	Timestamp        time.Time              `json:"timestamp"`
	Success          bool                   `json:"success"`
	LegacyResult     interface{}            `json:"legacy_result"`
	UnifiedResult    interface{}            `json:"unified_result"`
	ComparisonResult *ComparisonResult      `json:"comparison_result"`
	Error            error                  `json:"error,omitempty"`
	Details          map[string]interface{} `json:"details,omitempty"`
}

// ComparisonResult - 結果比較の詳細
type ComparisonResult struct {
	Match       bool                   `json:"match"`
	Differences []string               `json:"differences,omitempty"`
	Similarity  float64                `json:"similarity"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Validator - 移行検証システム
type Validator struct {
	config  *config.GradualMigrationConfig
	log     logger.Logger
	results map[string]*ValidationResult
	mu      sync.RWMutex

	// 検証統計
	totalValidations   int64
	successValidations int64
	failureValidations int64
}

// NewValidator - 新しい検証システムを作成
func NewValidator(cfg *config.GradualMigrationConfig, log logger.Logger) *Validator {
	return &Validator{
		config:  cfg,
		log:     log,
		results: make(map[string]*ValidationResult),
	}
}

// ValidateSystemMigration - システム移行の検証
func (v *Validator) ValidateSystemMigration(ctx context.Context, systemName string, legacyFunc, unifiedFunc func(ctx context.Context) (interface{}, error)) *ValidationResult {
	v.mu.Lock()
	defer v.mu.Unlock()

	startTime := time.Now()
	result := &ValidationResult{
		SystemName: systemName,
		Timestamp:  startTime,
		Details:    make(map[string]interface{}),
	}

	v.totalValidations++

	// レガシーシステムでの実行
	legacyResult, legacyErr := legacyFunc(ctx)
	if legacyErr != nil {
		result.Error = fmt.Errorf("legacy system error: %w", legacyErr)
		result.Success = false
		v.failureValidations++
		v.results[systemName] = result
		v.logValidationResult(result)
		return result
	}
	result.LegacyResult = legacyResult

	// 統合システムでの実行
	unifiedResult, unifiedErr := unifiedFunc(ctx)
	if unifiedErr != nil {
		result.Error = fmt.Errorf("unified system error: %w", unifiedErr)
		result.Success = false
		v.failureValidations++
		v.results[systemName] = result
		v.logValidationResult(result)
		return result
	}
	result.UnifiedResult = unifiedResult

	// 結果比較
	comparison := v.compareResults(legacyResult, unifiedResult)
	result.ComparisonResult = comparison
	result.Success = comparison.Match

	// 実行時間記録
	result.Details["execution_time"] = time.Since(startTime)

	if result.Success {
		v.successValidations++
	} else {
		v.failureValidations++
	}

	v.results[systemName] = result
	v.logValidationResult(result)

	return result
}

// ValidateStreamingMigration - ストリーミング移行の検証
func (v *Validator) ValidateStreamingMigration(ctx context.Context, testData string) *ValidationResult {
	return v.ValidateSystemMigration(ctx, "streaming",
		func(ctx context.Context) (interface{}, error) {
			// レガシーストリーミング処理のシミュレーション
			return map[string]interface{}{
				"data":    testData,
				"chunks":  len(testData) / 10,
				"method":  "legacy_streaming",
				"latency": time.Millisecond * 50,
			}, nil
		},
		func(ctx context.Context) (interface{}, error) {
			// 統合ストリーミング処理のシミュレーション
			return map[string]interface{}{
				"data":    testData,
				"chunks":  len(testData) / 10,
				"method":  "unified_streaming",
				"latency": time.Millisecond * 30,
			}, nil
		},
	)
}

// ValidateSessionMigration - セッション移行の検証
func (v *Validator) ValidateSessionMigration(ctx context.Context, sessionData interface{}) *ValidationResult {
	return v.ValidateSystemMigration(ctx, "session",
		func(ctx context.Context) (interface{}, error) {
			// レガシーセッション処理のシミュレーション
			return map[string]interface{}{
				"session_id":  "legacy_session_123",
				"data":        sessionData,
				"storage":     "file_based",
				"compression": false,
			}, nil
		},
		func(ctx context.Context) (interface{}, error) {
			// 統合セッション処理のシミュレーション
			return map[string]interface{}{
				"session_id":  "unified_session_123",
				"data":        sessionData,
				"storage":     "unified_storage",
				"compression": true,
			}, nil
		},
	)
}

// ValidateToolsMigration - ツール移行の検証
func (v *Validator) ValidateToolsMigration(ctx context.Context, toolName string, parameters map[string]interface{}) *ValidationResult {
	return v.ValidateSystemMigration(ctx, "tools",
		func(ctx context.Context) (interface{}, error) {
			// レガシーツール処理のシミュレーション
			return map[string]interface{}{
				"tool":       toolName,
				"parameters": parameters,
				"registry":   "legacy_registry",
				"execution":  "sequential",
			}, nil
		},
		func(ctx context.Context) (interface{}, error) {
			// 統合ツール処理のシミュレーション
			return map[string]interface{}{
				"tool":       toolName,
				"parameters": parameters,
				"registry":   "unified_registry",
				"execution":  "optimized",
			}, nil
		},
	)
}

// ValidateAnalysisMigration - 分析移行の検証
func (v *Validator) ValidateAnalysisMigration(ctx context.Context, analysisRequest interface{}) *ValidationResult {
	return v.ValidateSystemMigration(ctx, "analysis",
		func(ctx context.Context) (interface{}, error) {
			// レガシー分析処理のシミュレーション
			return map[string]interface{}{
				"request":    analysisRequest,
				"analyzer":   "legacy_analyzer",
				"cognitive":  false,
				"confidence": 0.8,
			}, nil
		},
		func(ctx context.Context) (interface{}, error) {
			// 統合分析処理のシミュレーション
			return map[string]interface{}{
				"request":    analysisRequest,
				"analyzer":   "unified_analyzer",
				"cognitive":  true,
				"confidence": 0.9,
			}, nil
		},
	)
}

// compareResults - 結果の比較
func (v *Validator) compareResults(legacy, unified interface{}) *ComparisonResult {
	comparison := &ComparisonResult{
		Metadata: make(map[string]interface{}),
	}

	// 簡易的な比較（実際の実装では、より詳細な比較ロジックが必要）
	legacyMap, legacyOk := legacy.(map[string]interface{})
	unifiedMap, unifiedOk := unified.(map[string]interface{})

	if !legacyOk || !unifiedOk {
		comparison.Match = false
		comparison.Differences = append(comparison.Differences, "type mismatch")
		comparison.Similarity = 0.0
		return comparison
	}

	// 重要なフィールドの比較
	var matches, total int
	var differences []string

	// データの一致チェック
	if legacyData, exists := legacyMap["data"]; exists {
		if unifiedData, exists := unifiedMap["data"]; exists {
			total++
			if fmt.Sprintf("%v", legacyData) == fmt.Sprintf("%v", unifiedData) {
				matches++
			} else {
				differences = append(differences, "data content mismatch")
			}
		}
	}

	// メソッドの違いは許容（これが移行の目的）
	if legacyMethod, exists := legacyMap["method"]; exists {
		if unifiedMethod, exists := unifiedMap["method"]; exists {
			total++
			if legacyMethod != unifiedMethod {
				// これは期待される違い
				comparison.Metadata["method_changed"] = true
			} else {
				matches++
			}
		}
	}

	// パフォーマンスの比較
	if legacyLatency, exists := legacyMap["latency"]; exists {
		if unifiedLatency, exists := unifiedMap["latency"]; exists {
			total++
			comparison.Metadata["performance_improvement"] =
				legacyLatency.(time.Duration) > unifiedLatency.(time.Duration)
		}
	}

	// 類似度計算
	if total > 0 {
		comparison.Similarity = float64(matches) / float64(total)
	}

	// 結果判定（70%以上の類似度で成功とする）
	comparison.Match = comparison.Similarity >= 0.7
	comparison.Differences = differences

	return comparison
}

// GetValidationResult - 検証結果を取得
func (v *Validator) GetValidationResult(systemName string) *ValidationResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.results[systemName]
}

// GetAllValidationResults - 全ての検証結果を取得
func (v *Validator) GetAllValidationResults() map[string]*ValidationResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	results := make(map[string]*ValidationResult)
	for k, v := range v.results {
		results[k] = v
	}

	return results
}

// GetValidationStats - 検証統計を取得
func (v *Validator) GetValidationStats() *ValidationStats {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return &ValidationStats{
		TotalValidations:   v.totalValidations,
		SuccessValidations: v.successValidations,
		FailureValidations: v.failureValidations,
		SuccessRate:        float64(v.successValidations) / float64(v.totalValidations),
		LastUpdate:         time.Now(),
	}
}

// logValidationResult - 検証結果をログに記録
func (v *Validator) logValidationResult(result *ValidationResult) {
	logData := map[string]interface{}{
		"system":    result.SystemName,
		"success":   result.Success,
		"timestamp": result.Timestamp,
	}

	if result.ComparisonResult != nil {
		logData["similarity"] = result.ComparisonResult.Similarity
		logData["match"] = result.ComparisonResult.Match
	}

	if result.Error != nil {
		logData["error"] = result.Error.Error()
		v.log.Error("移行検証失敗", logData)
	} else if result.Success {
		v.log.Info("移行検証成功", logData)
	} else {
		v.log.Warn("移行検証不一致", logData)
	}
}

// ValidationStats - 検証統計情報
type ValidationStats struct {
	TotalValidations   int64     `json:"total_validations"`
	SuccessValidations int64     `json:"success_validations"`
	FailureValidations int64     `json:"failure_validations"`
	SuccessRate        float64   `json:"success_rate"`
	LastUpdate         time.Time `json:"last_update"`
}
