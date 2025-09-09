package contextmanager

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestSmartContextManager は基本的なコンテキスト管理機能をテスト
func TestSmartContextManager(t *testing.T) {
	manager := NewSmartContextManager()

	// 基本的なコンテキスト追加
	item := &ContextItem{
		Type:       ContextTypeImmediate,
		Content:    "テスト用のコンテキスト内容",
		Metadata:   map[string]string{"test": "value"},
		Importance: 0.8,
	}

	err := manager.AddContext(item)
	if err != nil {
		t.Fatalf("コンテキスト追加エラー: %v", err)
	}

	// 統計情報確認
	stats, err := manager.GetStats()
	if err != nil {
		t.Fatalf("統計取得エラー: %v", err)
	}

	if stats.TotalItems != 1 {
		t.Errorf("期待される総項目数: 1, 実際: %d", stats.TotalItems)
	}

	if stats.ImmediateItems != 1 {
		t.Errorf("期待される即座項目数: 1, 実際: %d", stats.ImmediateItems)
	}
}

// TestContextCompression は圧縮機能をテスト
func TestContextCompression(t *testing.T) {
	manager := NewSmartContextManager()

	// 大量のコンテキストを追加して圧縮をトリガー
	testItems := createTestContextItems(300) // 最大短期項目数(200)を超える

	for _, item := range testItems {
		err := manager.AddContext(item)
		if err != nil {
			t.Fatalf("コンテキスト追加エラー: %v", err)
		}
	}

	// 強制圧縮実行
	compressed, err := manager.CompressContext(true)
	if err != nil {
		t.Fatalf("圧縮実行エラー: %v", err)
	}

	if compressed == nil {
		t.Fatalf("圧縮結果がnilです")
	}

	// 圧縮効果の確認
	if compressed.CompressedSize >= compressed.OriginalSize {
		t.Errorf("圧縮効果が見られません。オリジナル: %d, 圧縮後: %d",
			compressed.OriginalSize, compressed.CompressedSize)
	}

	t.Logf("圧縮結果 - オリジナル: %d bytes, 圧縮後: %d bytes, 圧縮率: %.2f%%",
		compressed.OriginalSize, compressed.CompressedSize,
		(1.0-float64(compressed.CompressedSize)/float64(compressed.OriginalSize))*100)

	// 圧縮後の統計確認
	stats, err := manager.GetStats()
	if err != nil {
		t.Fatalf("統計取得エラー: %v", err)
	}

	if stats.MediumTermItems == 0 {
		t.Errorf("中期コンテキストに圧縮済みアイテムがありません")
	}

	t.Logf("圧縮後統計 - 総項目: %d, 即座: %d, 短期: %d, 中期: %d, 長期: %d",
		stats.TotalItems, stats.ImmediateItems, stats.ShortTermItems,
		stats.MediumTermItems, stats.LongTermItems)
}

// TestRelevanceCalculation は関連度計算をテスト
func TestRelevanceCalculation(t *testing.T) {
	manager := NewSmartContextManager()

	// 関連度テスト用のアイテム
	items := []*ContextItem{
		{
			Type:       ContextTypeImmediate,
			Content:    "Go言語でのエラーハンドリング関数の実装",
			Importance: 0.9,
		},
		{
			Type:       ContextTypeShortTerm,
			Content:    "JavaScript配列操作のパフォーマンス最適化",
			Importance: 0.7,
		},
		{
			Type:       ContextTypeShortTerm,
			Content:    "Goでのgoroutineとチャネルの使用例",
			Importance: 0.8,
		},
	}

	for _, item := range items {
		err := manager.AddContext(item)
		if err != nil {
			t.Fatalf("コンテキスト追加エラー: %v", err)
		}
	}

	// Go言語に関する問い合わせ
	relevantItems, err := manager.GetRelevantContext("Go エラー処理", 10)
	if err != nil {
		t.Fatalf("関連コンテキスト取得エラー: %v", err)
	}

	if len(relevantItems) == 0 {
		t.Fatalf("関連アイテムが見つかりません")
	}

	// 最も関連度が高いアイテムがGo言語のエラーハンドリングであることを確認
	topItem := relevantItems[0]
	if !strings.Contains(topItem.Content, "Go") || !strings.Contains(topItem.Content, "エラー") {
		t.Errorf("最も関連度が高いアイテムが期待されるものではありません: %s", topItem.Content)
	}

	t.Logf("関連度が最も高いアイテム (関連度: %.3f): %s",
		topItem.Relevance, topItem.Content)

	// 関連度の降順確認
	for i := 1; i < len(relevantItems); i++ {
		if relevantItems[i-1].Relevance < relevantItems[i].Relevance {
			t.Errorf("関連度がソートされていません。位置 %d: %.3f, 位置 %d: %.3f",
				i-1, relevantItems[i-1].Relevance, i, relevantItems[i].Relevance)
		}
	}
}

// TestMemoryEfficiency はメモリ効率をテスト
func TestMemoryEfficiency(t *testing.T) {
	manager := NewSmartContextManager()

	// 大量のデータを追加
	largeItems := createLargeTestItems(100)

	initialMemory, err := manager.GetMemoryUsage()
	if err != nil {
		t.Fatalf("初期メモリ使用量取得エラー: %v", err)
	}

	for _, item := range largeItems {
		err := manager.AddContext(item)
		if err != nil {
			t.Fatalf("大容量コンテキスト追加エラー: %v", err)
		}
	}

	beforeCompressionMemory, err := manager.GetMemoryUsage()
	if err != nil {
		t.Fatalf("圧縮前メモリ使用量取得エラー: %v", err)
	}

	// 圧縮実行
	compressed, err := manager.CompressContext(true)
	if err != nil {
		t.Fatalf("圧縮実行エラー: %v", err)
	}

	afterCompressionMemory, err := manager.GetMemoryUsage()
	if err != nil {
		t.Fatalf("圧縮後メモリ使用量取得エラー: %v", err)
	}

	memorySaved := beforeCompressionMemory - afterCompressionMemory
	compressionRatio := 1.0 - float64(afterCompressionMemory)/float64(beforeCompressionMemory)

	t.Logf("メモリ使用量 - 初期: %d bytes, 圧縮前: %d bytes, 圧縮後: %d bytes",
		initialMemory, beforeCompressionMemory, afterCompressionMemory)
	t.Logf("メモリ節約: %d bytes (%.2f%% 削減)", memorySaved, compressionRatio*100)

	if memorySaved <= 0 {
		t.Errorf("メモリ節約効果が見られません")
	}

	if compressed != nil {
		t.Logf("圧縮統計 - キーポイント: %d個, 重要ファイル: %d個",
			len(compressed.KeyPoints), len(compressed.ImportantFiles))
	}
}

// TestTimeBasedCompression は時間ベースの圧縮をテスト
func TestTimeBasedCompression(t *testing.T) {
	manager := NewSmartContextManager()

	// 古い項目を追加（3時間前）
	oldItem := &ContextItem{
		Type:       ContextTypeShortTerm,
		Content:    "古いコンテキスト内容 - 3時間前のセッション",
		Timestamp:  time.Now().Add(-3 * time.Hour),
		Importance: 0.6,
	}

	// 新しい項目を追加（30分前）
	newItem := &ContextItem{
		Type:       ContextTypeShortTerm,
		Content:    "新しいコンテキスト内容 - 30分前のセッション",
		Timestamp:  time.Now().Add(-30 * time.Minute),
		Importance: 0.7,
	}

	err := manager.AddContext(oldItem)
	if err != nil {
		t.Fatalf("古いアイテム追加エラー: %v", err)
	}

	err = manager.AddContext(newItem)
	if err != nil {
		t.Fatalf("新しいアイテム追加エラー: %v", err)
	}

	statsBeforeCompression, _ := manager.GetStats()

	// 時間ベース圧縮実行
	compressed, err := manager.CompressContext(true)
	if err != nil {
		t.Fatalf("時間ベース圧縮エラー: %v", err)
	}

	statsAfterCompression, _ := manager.GetStats()

	t.Logf("圧縮前 - 短期項目: %d, 中期項目: %d",
		statsBeforeCompression.ShortTermItems, statsBeforeCompression.MediumTermItems)
	t.Logf("圧縮後 - 短期項目: %d, 中期項目: %d",
		statsAfterCompression.ShortTermItems, statsAfterCompression.MediumTermItems)

	if compressed != nil && len(compressed.KeyPoints) > 0 {
		t.Logf("圧縮された内容のキーポイント: %v", compressed.KeyPoints)
	}

	// 新しいアイテムは短期コンテキストに残っている、古いアイテムは圧縮されているべき
	if statsAfterCompression.ShortTermItems == 0 {
		t.Errorf("新しいアイテムが短期コンテキストから消えました")
	}
}

// TestConcurrentAccess は並行アクセスをテスト
func TestConcurrentAccess(t *testing.T) {
	manager := NewSmartContextManager()

	// 並行してコンテキストを追加
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			item := &ContextItem{
				Type:       ContextTypeShortTerm,
				Content:    fmt.Sprintf("並行テスト項目 %d - データ内容", id),
				Importance: 0.5 + float64(id)/20.0,
			}

			err := manager.AddContext(item)
			if err != nil {
				t.Errorf("並行アクセス時のコンテキスト追加エラー (ID: %d): %v", id, err)
			}
		}(i)
	}

	// すべてのgoroutineの完了を待機
	for i := 0; i < 10; i++ {
		<-done
	}

	stats, err := manager.GetStats()
	if err != nil {
		t.Fatalf("並行アクセス後の統計取得エラー: %v", err)
	}

	if stats.TotalItems != 10 {
		t.Errorf("期待される総項目数: 10, 実際: %d", stats.TotalItems)
	}

	t.Logf("並行アクセステスト完了 - 総項目数: %d", stats.TotalItems)
}

// ヘルパー関数

// createTestContextItems はテスト用のコンテキスト項目を作成
func createTestContextItems(count int) []*ContextItem {
	items := make([]*ContextItem, count)

	contentTemplates := []string{
		"Go言語での%d番目の関数実装について",
		"JavaScript%d個目の配列操作最適化",
		"Python%d番目のデータ処理アルゴリズム",
		"Java%d個目のオブジェクト指向設計パターン",
		"TypeScript%d番目の型安全性の確保方法",
	}

	for i := 0; i < count; i++ {
		template := contentTemplates[i%len(contentTemplates)]
		items[i] = &ContextItem{
			Type:       ContextTypeShortTerm, // 短期コンテキストとして追加
			Content:    fmt.Sprintf(template, i+1),
			Timestamp:  time.Now().Add(time.Duration(-i) * time.Minute),
			Importance: 0.3 + float64(i%7)/10.0, // 0.3-0.9の範囲でバリエーション
			Metadata: map[string]string{
				"test_id":  fmt.Sprintf("item_%d", i),
				"language": []string{"Go", "JavaScript", "Python", "Java", "TypeScript"}[i%5],
				"category": "test_data",
			},
		}
	}

	return items
}

// createLargeTestItems は大容量テスト項目を作成
func createLargeTestItems(count int) []*ContextItem {
	items := make([]*ContextItem, count)

	// 大きなコンテンツを生成（各アイテム約1KB）
	largeContent := strings.Repeat("大容量テストデータ内容。", 100)

	for i := 0; i < count; i++ {
		items[i] = &ContextItem{
			Type:       ContextTypeShortTerm,
			Content:    fmt.Sprintf("項目%d: %s", i, largeContent),
			Timestamp:  time.Now().Add(time.Duration(-i) * time.Hour),
			Importance: 0.5 + float64(i%10)/20.0,
			Metadata: map[string]string{
				"size":     "large",
				"test_id":  fmt.Sprintf("large_item_%d", i),
				"category": "memory_test",
			},
		}
	}

	return items
}

// BenchmarkContextCompression は圧縮パフォーマンスのベンチマーク
func BenchmarkContextCompression(b *testing.B) {
	manager := NewSmartContextManager()

	// テストデータ準備
	testItems := createTestContextItems(1000)
	for _, item := range testItems {
		manager.AddContext(item)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := manager.CompressContext(true)
		if err != nil {
			b.Fatalf("圧縮ベンチマークエラー: %v", err)
		}
	}
}

// BenchmarkRelevanceSearch は関連度検索パフォーマンスのベンチマーク
func BenchmarkRelevanceSearch(b *testing.B) {
	manager := NewSmartContextManager()

	// 大量のテストデータ準備
	testItems := createTestContextItems(5000)
	for _, item := range testItems {
		manager.AddContext(item)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := manager.GetRelevantContext("Go 関数 最適化", 20)
		if err != nil {
			b.Fatalf("関連度検索ベンチマークエラー: %v", err)
		}
	}
}
