package performance

import (
	"sync"
	"time"
)

// キャッシュエントリ
type CacheEntry struct {
	Value     interface{}
	ExpiresAt time.Time
}

// LRUキャッシュの実装
type Cache struct {
	mu      sync.RWMutex
	items   map[string]*CacheEntry
	maxSize int
	ttl     time.Duration

	// LRU管理用
	accessOrder []string
}

// キャッシュのコンストラクタ
func NewCache(maxSize int, ttl time.Duration) *Cache {
	return &Cache{
		items:       make(map[string]*CacheEntry),
		maxSize:     maxSize,
		ttl:         ttl,
		accessOrder: make([]string, 0, maxSize),
	}
}

// キャッシュから値を取得
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// 期限切れチェック
	if time.Now().After(entry.ExpiresAt) {
		delete(c.items, key)
		c.removeFromAccessOrder(key)
		return nil, false
	}

	// アクセス順序を更新
	c.updateAccessOrder(key)

	return entry.Value, true
}

// キャッシュに値を設定
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 新しいエントリを作成
	entry := &CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}

	// 既存のエントリがある場合は更新
	if _, exists := c.items[key]; exists {
		c.items[key] = entry
		c.updateAccessOrder(key)
		return
	}

	// 容量チェック - LRUアルゴリズムで古いエントリを削除
	for len(c.items) >= c.maxSize {
		c.evictLRU()
	}

	// 新しいエントリを追加
	c.items[key] = entry
	c.accessOrder = append(c.accessOrder, key)
}

// キャッシュから削除
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	c.removeFromAccessOrder(key)
}

// キャッシュをクリア
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheEntry)
	c.accessOrder = c.accessOrder[:0]
}

// 期限切れエントリの削除
func (c *Cache) CleanExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	toDelete := make([]string, 0)

	for key, entry := range c.items {
		if now.After(entry.ExpiresAt) {
			toDelete = append(toDelete, key)
		}
	}

	for _, key := range toDelete {
		delete(c.items, key)
		c.removeFromAccessOrder(key)
	}
}

// キャッシュ統計情報の取得
func (c *Cache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"size":     len(c.items),
		"max_size": c.maxSize,
		"ttl":      c.ttl.String(),
	}
}

// LRUエントリの削除（内部関数）
func (c *Cache) evictLRU() {
	if len(c.accessOrder) > 0 {
		oldestKey := c.accessOrder[0]
		delete(c.items, oldestKey)
		c.accessOrder = c.accessOrder[1:]
	}
}

// アクセス順序の更新（内部関数）
func (c *Cache) updateAccessOrder(key string) {
	// 既存のキーを削除
	c.removeFromAccessOrder(key)
	// 最後に追加
	c.accessOrder = append(c.accessOrder, key)
}

// アクセス順序からキーを削除（内部関数）
func (c *Cache) removeFromAccessOrder(key string) {
	for i, k := range c.accessOrder {
		if k == key {
			c.accessOrder = append(c.accessOrder[:i], c.accessOrder[i+1:]...)
			break
		}
	}
}

// グローバルキャッシュインスタンス
var (
	LLMResponseCache = NewCache(100, 10*time.Minute) // LLMレスポンス用
	FileContentCache = NewCache(200, 5*time.Minute)  // ファイル内容用
	CommandCache     = NewCache(50, 2*time.Minute)   // コマンド結果用
)

// 定期的なキャッシュクリーンアップ
func StartCacheCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			LLMResponseCache.CleanExpired()
			FileContentCache.CleanExpired()
			CommandCache.CleanExpired()
		}
	}()
}
