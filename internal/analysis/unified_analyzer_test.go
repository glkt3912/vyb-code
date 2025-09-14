package analysis

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glkt/vyb-code/internal/ai"
	"github.com/glkt/vyb-code/internal/config"
)

// MockLLMClient - テスト用のモックLLMクライアント
type MockLLMClient struct{}

func (m *MockLLMClient) GenerateResponse(ctx context.Context, request *ai.GenerateRequest) (*ai.GenerateResponse, error) {
	return &ai.GenerateResponse{
		Content: "Mock response for testing",
	}, nil
}

func TestNewUnifiedAnalyzer(t *testing.T) {
	cfg := &config.Config{}
	llmClient := &MockLLMClient{}

	analyzer := NewUnifiedAnalyzer(cfg, llmClient)

	if analyzer == nil {
		t.Error("Expected analyzer to be created")
	}

	if analyzer.config == nil {
		t.Error("Expected config to be initialized")
	}

	if analyzer.llmClient == nil {
		t.Error("Expected LLM client to be set")
	}

	if analyzer.cache == nil {
		t.Error("Expected cache to be initialized")
	}

	if analyzer.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}
}

func TestUnifiedAnalyzer_ConfigManagement(t *testing.T) {
	analyzer := NewUnifiedAnalyzer(&config.Config{}, &MockLLMClient{})

	t.Run("Get default config", func(t *testing.T) {
		config := analyzer.GetConfig()
		if config == nil {
			t.Error("Expected config to be returned")
		}

		if config.EnableCaching != true {
			t.Error("Expected caching to be enabled by default")
		}

		if config.CognitiveDepth != "standard" {
			t.Error("Expected default cognitive depth to be 'standard'")
		}
	})

	t.Run("Update valid config", func(t *testing.T) {
		newConfig := &UnifiedAnalysisConfig{
			EnableCaching:      false,
			CacheExpiry:        time.Hour,
			Timeout:            5 * time.Minute,
			CognitiveDepth:     "deep",
			EnableConfidence:   true,
			EnableReasoning:    true,
			EnableCreativity:   false,
			MaxCacheSize:       500,
			AnalysisWorkers:    2,
			ConcurrentAnalyses: 1,
			ProjectConfig:      DefaultAnalysisConfig(),
		}

		err := analyzer.UpdateConfig(newConfig)
		if err != nil {
			t.Errorf("Expected config update to succeed: %v", err)
		}

		updatedConfig := analyzer.GetConfig()
		if updatedConfig.EnableCaching != false {
			t.Error("Expected caching to be disabled")
		}

		if updatedConfig.CognitiveDepth != "deep" {
			t.Error("Expected cognitive depth to be updated")
		}
	})

	t.Run("Update invalid config", func(t *testing.T) {
		invalidConfig := &UnifiedAnalysisConfig{
			EnableCaching:      true,
			CacheExpiry:        -time.Hour, // Invalid negative value
			MaxCacheSize:       100,
			AnalysisWorkers:    1,
			ConcurrentAnalyses: 1,
		}

		err := analyzer.UpdateConfig(invalidConfig)
		if err == nil {
			t.Error("Expected config validation to fail for negative cache expiry")
		}
	})
}

func TestUnifiedAnalyzer_CacheManagement(t *testing.T) {
	analyzer := NewUnifiedAnalyzer(&config.Config{}, &MockLLMClient{})

	t.Run("Cache and retrieve analysis", func(t *testing.T) {
		testKey := "test-key"
		testResult := &ProjectAnalysis{
			ProjectName: "Test Project",
			Language:    "Go",
		}

		err := analyzer.CacheAnalysis(testKey, AnalysisTypeBasic, testResult)
		if err != nil {
			t.Errorf("Expected caching to succeed: %v", err)
		}

		cached, err := analyzer.GetCachedAnalysis(testKey, AnalysisTypeBasic)
		if err != nil {
			t.Errorf("Expected cached result retrieval to succeed: %v", err)
		}

		cachedProject, ok := cached.(*ProjectAnalysis)
		if !ok {
			t.Error("Expected cached result to be ProjectAnalysis type")
		}

		if cachedProject.ProjectName != "Test Project" {
			t.Error("Expected cached project name to match")
		}
	})

	t.Run("Cache invalidation", func(t *testing.T) {
		testKey := "invalidation-test"
		testResult := &ProjectAnalysis{ProjectName: "Test"}

		analyzer.CacheAnalysis(testKey, AnalysisTypeBasic, testResult)

		err := analyzer.InvalidateCache(testKey, AnalysisTypeBasic)
		if err != nil {
			t.Errorf("Expected cache invalidation to succeed: %v", err)
		}

		_, err = analyzer.GetCachedAnalysis(testKey, AnalysisTypeBasic)
		if err == nil {
			t.Error("Expected cached result to be invalidated")
		}
	})

	t.Run("Cache cleanup", func(t *testing.T) {
		// 期限切れエントリを作成
		analyzer.config.CacheExpiry = time.Millisecond
		analyzer.CacheAnalysis("expired-key", AnalysisTypeBasic, &ProjectAnalysis{})

		time.Sleep(time.Millisecond * 10)

		cleanedCount := analyzer.CleanupCache()
		if cleanedCount == 0 {
			t.Error("Expected at least one entry to be cleaned up")
		}
	})
}

func TestUnifiedAnalyzer_MetricsTracking(t *testing.T) {
	analyzer := NewUnifiedAnalyzer(&config.Config{}, &MockLLMClient{})

	t.Run("Initial metrics", func(t *testing.T) {
		metrics := analyzer.GetAnalysisMetrics()
		if metrics == nil {
			t.Error("Expected metrics to be returned")
		}

		if metrics.TotalAnalyses != 0 {
			t.Error("Expected initial total analyses to be 0")
		}

		if metrics.CacheHitRate != 0 {
			t.Error("Expected initial cache hit rate to be 0")
		}
	})

	t.Run("Metrics after analysis", func(t *testing.T) {
		// シミュレートされた分析を実行
		analyzer.updateAnalysisMetrics(time.Second, "project")
		analyzer.updateAnalysisMetrics(time.Second*2, "cognitive")

		metrics := analyzer.GetAnalysisMetrics()

		if metrics.TotalAnalyses != 2 {
			t.Errorf("Expected total analyses to be 2, got %d", metrics.TotalAnalyses)
		}

		if metrics.ProjectAnalyses != 1 {
			t.Errorf("Expected project analyses to be 1, got %d", metrics.ProjectAnalyses)
		}

		if metrics.CognitiveAnalyses != 1 {
			t.Errorf("Expected cognitive analyses to be 1, got %d", metrics.CognitiveAnalyses)
		}

		expectedAvg := time.Duration(1.5 * float64(time.Second))
		if metrics.AverageProcessingTime != expectedAvg {
			t.Errorf("Expected average processing time to be %v, got %v",
				expectedAvg, metrics.AverageProcessingTime)
		}
	})

	t.Run("Reset metrics", func(t *testing.T) {
		analyzer.ResetMetrics()

		metrics := analyzer.GetAnalysisMetrics()
		if metrics.TotalAnalyses != 0 {
			t.Error("Expected metrics to be reset")
		}
	})
}

func TestUnifiedAnalyzer_PerformanceOptimization(t *testing.T) {
	analyzer := NewUnifiedAnalyzer(&config.Config{}, &MockLLMClient{})

	t.Run("Performance optimization", func(t *testing.T) {
		// テストデータをセットアップ
		analyzer.CacheAnalysis("test1", AnalysisTypeBasic, &ProjectAnalysis{})
		analyzer.CacheAnalysis("test2", AnalysisTypeFull, &ProjectAnalysis{})

		result := analyzer.PerformanceOptimization()

		if result == nil {
			t.Error("Expected optimization result to be returned")
		}

		if result.ProcessingTime <= 0 {
			t.Error("Expected processing time to be recorded")
		}

		if result.OptimizationScore < 0 || result.OptimizationScore > 1 {
			t.Errorf("Expected optimization score to be between 0 and 1, got %f",
				result.OptimizationScore)
		}
	})
}

func TestUnifiedAnalyzer_CacheStatistics(t *testing.T) {
	analyzer := NewUnifiedAnalyzer(&config.Config{}, &MockLLMClient{})

	t.Run("Cache statistics", func(t *testing.T) {
		// テストデータを追加
		analyzer.CacheAnalysis("stats1", AnalysisTypeBasic, &ProjectAnalysis{})
		analyzer.CacheAnalysis("stats2", AnalysisTypeFull, &CognitiveAnalysisResult{})

		stats := analyzer.GetCacheStatistics()

		if stats == nil {
			t.Error("Expected cache statistics to be returned")
		}

		cacheSize, ok := stats["cache_size"].(int)
		if !ok || cacheSize != 2 {
			t.Errorf("Expected cache size to be 2, got %v", stats["cache_size"])
		}

		maxSize, ok := stats["max_cache_size"].(int)
		if !ok || maxSize <= 0 {
			t.Error("Expected max cache size to be positive")
		}

		utilizationRate, ok := stats["utilization_rate"].(float64)
		if !ok || utilizationRate < 0 || utilizationRate > 1 {
			t.Errorf("Expected utilization rate to be between 0 and 1, got %v",
				stats["utilization_rate"])
		}
	})
}

func TestDefaultUnifiedAnalysisConfig(t *testing.T) {
	config := DefaultUnifiedAnalysisConfig()

	if config == nil {
		t.Error("Expected default config to be created")
	}

	if !config.EnableCaching {
		t.Error("Expected caching to be enabled by default")
	}

	if config.CacheExpiry <= 0 {
		t.Error("Expected positive cache expiry")
	}

	if config.Timeout <= 0 {
		t.Error("Expected positive timeout")
	}

	if config.CognitiveDepth != "standard" {
		t.Error("Expected default cognitive depth to be 'standard'")
	}

	if config.MaxCacheSize <= 0 {
		t.Error("Expected positive max cache size")
	}

	if config.AnalysisWorkers <= 0 {
		t.Error("Expected positive analysis workers count")
	}

	if config.ConcurrentAnalyses <= 0 {
		t.Error("Expected positive concurrent analyses count")
	}
}

func TestConfigValidation(t *testing.T) {
	analyzer := NewUnifiedAnalyzer(&config.Config{}, &MockLLMClient{})

	testCases := []struct {
		name        string
		config      *UnifiedAnalysisConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid config",
			config: &UnifiedAnalysisConfig{
				EnableCaching:      true,
				CacheExpiry:        time.Hour,
				Timeout:            time.Minute,
				CognitiveDepth:     "standard",
				MaxCacheSize:       100,
				AnalysisWorkers:    2,
				ConcurrentAnalyses: 1,
			},
			expectError: false,
		},
		{
			name: "Invalid cache expiry",
			config: &UnifiedAnalysisConfig{
				CacheExpiry:        -time.Hour,
				Timeout:            time.Minute,
				MaxCacheSize:       100,
				AnalysisWorkers:    2,
				ConcurrentAnalyses: 1,
			},
			expectError: true,
			errorMsg:    "キャッシュ有効期限",
		},
		{
			name: "Invalid timeout",
			config: &UnifiedAnalysisConfig{
				CacheExpiry:        time.Hour,
				Timeout:            -time.Minute,
				MaxCacheSize:       100,
				AnalysisWorkers:    2,
				ConcurrentAnalyses: 1,
			},
			expectError: true,
			errorMsg:    "タイムアウト",
		},
		{
			name: "Invalid cognitive depth",
			config: &UnifiedAnalysisConfig{
				CacheExpiry:        time.Hour,
				Timeout:            time.Minute,
				CognitiveDepth:     "invalid",
				MaxCacheSize:       100,
				AnalysisWorkers:    2,
				ConcurrentAnalyses: 1,
			},
			expectError: true,
			errorMsg:    "不正な認知分析深度",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := analyzer.validateConfig(tc.config)

			if tc.expectError {
				if err == nil {
					t.Error("Expected validation error")
				} else if tc.errorMsg != "" && !contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to succeed: %v", err)
				}
			}
		})
	}
}

// Helper function for string containment check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 ||
		strings.Contains(strings.ToLower(s), strings.ToLower(substr)))
}
