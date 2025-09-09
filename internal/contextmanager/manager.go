package contextmanager

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// SmartContextManagerの実装
type smartContextManager struct {
	mu sync.RWMutex

	// コンテキスト項目の階層管理
	immediateContext  []*ContextItem
	shortTermContext  []*ContextItem
	mediumTermContext []*ContextItem
	longTermContext   []*ContextItem

	// 圧縮履歴
	compressionHistory []*CompressedContext

	// 設定
	maxImmediateItems  int
	maxShortTermItems  int
	compressionRatio   float64
	relevanceThreshold float64

	// メトリクス
	totalCompressed   int64
	totalMemorySaved  int64
	lastCompressionAt time.Time
}

// NewSmartContextManager は新しいスマートコンテキストマネージャーを作成する
func NewSmartContextManager() ContextManager {
	return &smartContextManager{
		immediateContext:   make([]*ContextItem, 0),
		shortTermContext:   make([]*ContextItem, 0),
		mediumTermContext:  make([]*ContextItem, 0),
		longTermContext:    make([]*ContextItem, 0),
		compressionHistory: make([]*CompressedContext, 0),

		// デフォルト設定（Claude Code相当の効率性を目指す）
		maxImmediateItems:  50,  // 即座のコンテキスト最大50項目
		maxShortTermItems:  200, // 短期コンテキスト最大200項目
		compressionRatio:   0.3, // 30%に圧縮
		relevanceThreshold: 0.1, // 関連度10%以下は除外

		totalCompressed:   0,
		totalMemorySaved:  0,
		lastCompressionAt: time.Now(),
	}
}

// AddContext はコンテキスト項目を追加する
func (scm *smartContextManager) AddContext(item *ContextItem) error {
	scm.mu.Lock()
	defer scm.mu.Unlock()

	// IDが空の場合は生成
	if item.ID == "" {
		item.ID = fmt.Sprintf("ctx_%d", time.Now().UnixNano())
	}

	// タイムスタンプ設定
	item.Timestamp = time.Now()
	item.LastAccess = time.Now()

	// 重要度の自動計算（コンテンツの特徴に基づく）
	if item.Importance == 0 {
		item.Importance = scm.calculateImportance(item)
	}

	// タイプに応じて適切なコンテキストに追加
	switch item.Type {
	case ContextTypeImmediate:
		scm.immediateContext = append(scm.immediateContext, item)
		// 最大数を超えた場合は古いものを短期コンテキストに移動
		if len(scm.immediateContext) > scm.maxImmediateItems {
			oldest := scm.immediateContext[0]
			oldest.Type = ContextTypeShortTerm
			scm.shortTermContext = append(scm.shortTermContext, oldest)
			scm.immediateContext = scm.immediateContext[1:]
		}

	case ContextTypeShortTerm:
		scm.shortTermContext = append(scm.shortTermContext, item)
		// 最大数を超えた場合は圧縮を検討
		if len(scm.shortTermContext) > scm.maxShortTermItems {
			_, err := scm.compressContextInternal(false)
			if err != nil {
				return fmt.Errorf("短期コンテキスト圧縮エラー: %w", err)
			}
		}

	case ContextTypeMediumTerm:
		scm.mediumTermContext = append(scm.mediumTermContext, item)

	case ContextTypeLongTerm:
		scm.longTermContext = append(scm.longTermContext, item)
	}

	return nil
}

// GetRelevantContext は関連度の高いコンテキストを取得する
func (scm *smartContextManager) GetRelevantContext(query string, maxItems int) ([]*ContextItem, error) {
	scm.mu.RLock()
	defer scm.mu.RUnlock()

	// 全コンテキストを結合
	allItems := make([]*ContextItem, 0)
	allItems = append(allItems, scm.immediateContext...)
	allItems = append(allItems, scm.shortTermContext...)
	allItems = append(allItems, scm.mediumTermContext...)
	allItems = append(allItems, scm.longTermContext...)

	// 関連度を計算
	for _, item := range allItems {
		item.Relevance = scm.calculateRelevance(item, query)
		// アクセス回数を増やす
		item.AccessCount++
		item.LastAccess = time.Now()
	}

	// 関連度でソート（高い順）
	sort.Slice(allItems, func(i, j int) bool {
		// 関連度が同じ場合は重要度で比較
		if allItems[i].Relevance == allItems[j].Relevance {
			return allItems[i].Importance > allItems[j].Importance
		}
		return allItems[i].Relevance > allItems[j].Relevance
	})

	// 閾値以上の関連度を持つ項目のみ返す
	relevantItems := make([]*ContextItem, 0)
	for _, item := range allItems {
		if len(relevantItems) >= maxItems {
			break
		}
		if item.Relevance >= scm.relevanceThreshold {
			relevantItems = append(relevantItems, item)
		}
	}

	return relevantItems, nil
}

// CompressContext はコンテキストを動的に圧縮する
func (scm *smartContextManager) CompressContext(forceCompress bool) (*CompressedContext, error) {
	scm.mu.Lock()
	defer scm.mu.Unlock()

	return scm.compressContextInternal(forceCompress)
}

// compressContextInternal は内部的な圧縮処理
func (scm *smartContextManager) compressContextInternal(forceCompress bool) (*CompressedContext, error) {
	now := time.Now()

	// 圧縮が必要かチェック
	if !forceCompress {
		// 最後の圧縮から1時間以内の場合はスキップ
		if now.Sub(scm.lastCompressionAt) < time.Hour {
			return nil, nil
		}
		// 短期コンテキストが閾値以下の場合はスキップ
		if len(scm.shortTermContext) < scm.maxShortTermItems {
			return nil, nil
		}
	}

	// 圧縮対象の特定（古い短期コンテキスト）
	compressTargets := make([]*ContextItem, 0)
	keepItems := make([]*ContextItem, 0)

	cutoffTime := now.Add(-2 * time.Hour) // 2時間より古いものを圧縮対象

	for _, item := range scm.shortTermContext {
		if item.Timestamp.Before(cutoffTime) {
			compressTargets = append(compressTargets, item)
		} else if forceCompress && len(compressTargets) < len(scm.shortTermContext)/2 {
			// forceCompress時でも、少なくとも半分は短期コンテキストに残す
			compressTargets = append(compressTargets, item)
		} else {
			keepItems = append(keepItems, item)
		}
	}

	if len(compressTargets) == 0 {
		return nil, nil // 圧縮対象なし
	}

	// 圧縮処理実行
	compressed, err := scm.performCompression(compressTargets)
	if err != nil {
		return nil, fmt.Errorf("圧縮実行エラー: %w", err)
	}

	// 圧縮結果を中期コンテキストに保存
	compressedItem := &ContextItem{
		ID:          fmt.Sprintf("compressed_%d", now.UnixNano()),
		Type:        ContextTypeMediumTerm,
		Content:     compressed.Summary,
		Metadata:    map[string]string{"type": "compressed_context", "original_items": fmt.Sprintf("%d", len(compressTargets))},
		Timestamp:   now,
		Relevance:   0.0, // 後で計算
		Importance:  0.8, // 圧縮されたコンテキストは重要度高
		AccessCount: 0,
		LastAccess:  now,
	}

	scm.mediumTermContext = append(scm.mediumTermContext, compressedItem)

	// 短期コンテキストを更新
	scm.shortTermContext = keepItems

	// 統計更新
	scm.totalCompressed += int64(len(compressTargets))
	scm.totalMemorySaved += int64(compressed.OriginalSize - compressed.CompressedSize)
	scm.lastCompressionAt = now

	// 圧縮履歴に記録
	scm.compressionHistory = append(scm.compressionHistory, compressed)

	// 履歴は最新100件のみ保持
	if len(scm.compressionHistory) > 100 {
		scm.compressionHistory = scm.compressionHistory[len(scm.compressionHistory)-100:]
	}

	return compressed, nil
}

// performCompression は実際の圧縮処理を行う
func (scm *smartContextManager) performCompression(items []*ContextItem) (*CompressedContext, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("圧縮対象がありません")
	}

	// 全コンテンツを結合
	var allContent strings.Builder
	var keyPoints []string
	var importantFiles []string
	var recentDecisions []string

	originalSize := 0

	for _, item := range items {
		allContent.WriteString(item.Content)
		allContent.WriteString("\n")
		originalSize += len(item.Content)

		// メタデータから重要な情報を抽出
		if fileType, exists := item.Metadata["file"]; exists {
			if !contains(importantFiles, fileType) {
				importantFiles = append(importantFiles, fileType)
			}
		}

		if decision, exists := item.Metadata["decision"]; exists {
			if !contains(recentDecisions, decision) {
				recentDecisions = append(recentDecisions, decision)
			}
		}

		// 重要度の高い項目をキーポイントとして抽出
		if item.Importance > 0.7 {
			summary := scm.extractSummary(item.Content)
			if summary != "" && !contains(keyPoints, summary) {
				keyPoints = append(keyPoints, summary)
			}
		}
	}

	// 要約生成（簡単な実装 - 実際は LLM を使用することも可能）
	summary := scm.generateSummary(allContent.String())

	compressed := &CompressedContext{
		Summary:         summary,
		KeyPoints:       keyPoints,
		ImportantFiles:  importantFiles,
		RecentDecisions: recentDecisions,
		Metadata: map[string]string{
			"compressed_items": fmt.Sprintf("%d", len(items)),
			"compression_type": "automatic",
		},
		CompressedAt:   time.Now(),
		OriginalSize:   originalSize,
		CompressedSize: len(summary) + len(strings.Join(keyPoints, "")),
	}

	return compressed, nil
}

// CalculateRelevance は関連度を計算する
func (scm *smartContextManager) CalculateRelevance(item *ContextItem, query string) float64 {
	return scm.calculateRelevance(item, query)
}

// calculateRelevance は内部的な関連度計算
func (scm *smartContextManager) calculateRelevance(item *ContextItem, query string) float64 {
	if query == "" {
		return 0.5 // デフォルト関連度
	}

	queryWords := strings.Fields(strings.ToLower(query))
	contentWords := strings.Fields(strings.ToLower(item.Content))

	if len(queryWords) == 0 || len(contentWords) == 0 {
		return 0.0
	}

	// シンプルなTF-IDF風の関連度計算
	matches := 0
	for _, qWord := range queryWords {
		for _, cWord := range contentWords {
			if strings.Contains(cWord, qWord) || strings.Contains(qWord, cWord) {
				matches++
			}
		}
	}

	baseRelevance := float64(matches) / float64(len(queryWords))

	// 重要度による重み付け
	weightedRelevance := baseRelevance * (0.5 + 0.5*item.Importance)

	// アクセス頻度による重み付け
	accessWeight := math.Min(1.0, float64(item.AccessCount)/10.0)
	finalRelevance := weightedRelevance * (0.8 + 0.2*accessWeight)

	// 時間による減衰
	timeSinceCreation := time.Since(item.Timestamp)
	timeDecay := math.Exp(-float64(timeSinceCreation.Hours()) / 24.0) // 24時間で約37%に減衰

	return math.Min(1.0, finalRelevance*timeDecay)
}

// calculateImportance は重要度を計算する
func (scm *smartContextManager) calculateImportance(item *ContextItem) float64 {
	content := strings.ToLower(item.Content)

	// 重要キーワードによるスコアリング
	importance := 0.3 // ベーススコア

	importantKeywords := []string{
		"error", "エラー", "bug", "バグ", "fix", "修正",
		"security", "セキュリティ", "vulnerability", "脆弱性",
		"performance", "パフォーマンス", "optimization", "最適化",
		"todo", "やること", "important", "重要",
		"function", "関数", "class", "クラス", "interface", "インターフェース",
	}

	for _, keyword := range importantKeywords {
		if strings.Contains(content, keyword) {
			importance += 0.1
		}
	}

	// ファイル種別による重み
	if fileType, exists := item.Metadata["file_type"]; exists {
		switch fileType {
		case "go", "py", "js", "ts":
			importance += 0.2 // ソースコードファイル
		case "md", "txt":
			importance += 0.1 // ドキュメントファイル
		}
	}

	return math.Min(1.0, importance)
}

// GetMemoryUsage はメモリ使用量を取得する
func (scm *smartContextManager) GetMemoryUsage() (int64, error) {
	scm.mu.RLock()
	defer scm.mu.RUnlock()

	var totalSize int64

	// 各コンテキストのサイズを計算
	contexts := [][]*ContextItem{
		scm.immediateContext,
		scm.shortTermContext,
		scm.mediumTermContext,
		scm.longTermContext,
	}

	for _, contextItems := range contexts {
		for _, item := range contextItems {
			totalSize += int64(len(item.Content))
			// メタデータのサイズも考慮
			for k, v := range item.Metadata {
				totalSize += int64(len(k) + len(v))
			}
		}
	}

	return totalSize, nil
}

// GetStats は統計情報を取得する
func (scm *smartContextManager) GetStats() (*ContextStats, error) {
	scm.mu.RLock()
	defer scm.mu.RUnlock()

	memoryUsage, _ := scm.GetMemoryUsage()

	// 平均関連度計算
	totalItems := len(scm.immediateContext) + len(scm.shortTermContext) + len(scm.mediumTermContext) + len(scm.longTermContext)
	var totalRelevance float64
	allContexts := [][]*ContextItem{scm.immediateContext, scm.shortTermContext, scm.mediumTermContext, scm.longTermContext}

	for _, context := range allContexts {
		for _, item := range context {
			totalRelevance += item.Relevance
		}
	}

	averageRelevance := 0.0
	if totalItems > 0 {
		averageRelevance = totalRelevance / float64(totalItems)
	}

	return &ContextStats{
		TotalItems:         totalItems,
		ImmediateItems:     len(scm.immediateContext),
		ShortTermItems:     len(scm.shortTermContext),
		MediumTermItems:    len(scm.mediumTermContext),
		LongTermItems:      len(scm.longTermContext),
		TotalMemoryUsage:   memoryUsage,
		CompressionRatio:   scm.compressionRatio,
		LastCompressionAt:  scm.lastCompressionAt,
		AverageRelevance:   averageRelevance,
		CompressionHistory: len(scm.compressionHistory),
	}, nil
}

// ClearContext は指定したタイプのコンテキストをクリアする
func (scm *smartContextManager) ClearContext(contextType ContextType) error {
	scm.mu.Lock()
	defer scm.mu.Unlock()

	switch contextType {
	case ContextTypeImmediate:
		scm.immediateContext = scm.immediateContext[:0]
	case ContextTypeShortTerm:
		scm.shortTermContext = scm.shortTermContext[:0]
	case ContextTypeMediumTerm:
		scm.mediumTermContext = scm.mediumTermContext[:0]
	case ContextTypeLongTerm:
		scm.longTermContext = scm.longTermContext[:0]
	default:
		return fmt.Errorf("無効なコンテキストタイプ: %d", contextType)
	}

	return nil
}

// ヘルパー関数

// contains はスライスに指定の文字列が含まれているかチェックする
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// extractSummary はコンテンツから要約を抽出する
func (scm *smartContextManager) extractSummary(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return ""
	}

	// 最初の行を要約として使用（簡単な実装）
	summary := strings.TrimSpace(lines[0])
	if len(summary) > 100 {
		summary = summary[:100] + "..."
	}

	return summary
}

// generateSummary は全体的な要約を生成する
func (scm *smartContextManager) generateSummary(content string) string {
	if len(content) == 0 {
		return ""
	}

	lines := strings.Split(content, "\n")
	if len(lines) <= 3 {
		return content
	}

	// 重要そうな行を抽出（キーワード含む行）
	importantLines := make([]string, 0)
	keywords := []string{"function", "class", "error", "todo", "important", "fix", "bug"}

	for _, line := range lines {
		lowerLine := strings.ToLower(line)
		for _, keyword := range keywords {
			if strings.Contains(lowerLine, keyword) {
				importantLines = append(importantLines, strings.TrimSpace(line))
				break
			}
		}

		// 最大10行まで
		if len(importantLines) >= 10 {
			break
		}
	}

	if len(importantLines) > 0 {
		return strings.Join(importantLines, "\n")
	}

	// 重要な行が見つからない場合は最初の数行を返す
	maxLines := 5
	if len(lines) < maxLines {
		maxLines = len(lines)
	}

	return strings.Join(lines[:maxLines], "\n")
}
