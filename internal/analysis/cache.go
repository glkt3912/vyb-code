package analysis

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// キャッシュ管理の実装

// キャッシュされた分析結果を取得
func (pa *projectAnalyzer) GetCachedAnalysis(projectPath string) (*ProjectAnalysis, error) {
	pa.mutex.RLock()
	defer pa.mutex.RUnlock()

	cached, exists := pa.cache[projectPath]
	if !exists {
		return nil, fmt.Errorf("キャッシュが見つかりません")
	}

	if time.Now().After(cached.ExpiresAt) {
		// 期限切れキャッシュを削除
		delete(pa.cache, projectPath)
		return nil, fmt.Errorf("キャッシュが期限切れです")
	}

	return cached.Analysis, nil
}

// 分析結果をキャッシュに保存
func (pa *projectAnalyzer) CacheAnalysis(projectPath string, analysis *ProjectAnalysis) error {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()

	cached := &cachedAnalysis{
		Analysis:  analysis,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(pa.config.CacheExpiry),
	}

	pa.cache[projectPath] = cached

	// ディスクキャッシュも保存
	return pa.saveCacheToDisk(projectPath, cached)
}

// キャッシュを無効化
func (pa *projectAnalyzer) InvalidateCache(projectPath string) error {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()

	delete(pa.cache, projectPath)

	// ディスクキャッシュも削除
	cacheFile := pa.getCacheFilePath(projectPath)
	if _, err := os.Stat(cacheFile); err == nil {
		return os.Remove(cacheFile)
	}

	return nil
}

// ディスクキャッシュファイルのパスを取得
func (pa *projectAnalyzer) getCacheFilePath(projectPath string) string {
	// プロジェクトパスをハッシュ化してファイル名に使用
	hash := pa.hashProjectPath(projectPath)
	cacheDir := filepath.Join(os.TempDir(), "vyb-analysis-cache")
	return filepath.Join(cacheDir, fmt.Sprintf("%s.json", hash))
}

// プロジェクトパスをハッシュ化
func (pa *projectAnalyzer) hashProjectPath(projectPath string) string {
	// 簡易ハッシュ化（実際の実装ではより堅牢なハッシュ関数を使用）
	absPath, _ := filepath.Abs(projectPath)
	hash := 0
	for _, char := range absPath {
		hash = hash*31 + int(char)
	}
	return fmt.Sprintf("%x", hash)
}

// キャッシュをディスクに保存
func (pa *projectAnalyzer) saveCacheToDisk(projectPath string, cached *cachedAnalysis) error {
	cacheFile := pa.getCacheFilePath(projectPath)
	cacheDir := filepath.Dir(cacheFile)

	// キャッシュディレクトリを作成
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	// JSON形式で保存
	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// ディスクからキャッシュを読み込み
func (pa *projectAnalyzer) loadCacheFromDisk(projectPath string) (*cachedAnalysis, error) {
	cacheFile := pa.getCacheFilePath(projectPath)

	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("ディスクキャッシュが見つかりません")
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var cached cachedAnalysis
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}

	// 期限切れチェック
	if time.Now().After(cached.ExpiresAt) {
		os.Remove(cacheFile) // 期限切れファイルを削除
		return nil, fmt.Errorf("ディスクキャッシュが期限切れです")
	}

	return &cached, nil
}

// 初期化時にディスクキャッシュを読み込み
func (pa *projectAnalyzer) InitializeCache() {
	// 実際の実装では、起動時に既存のキャッシュファイルを読み込む処理を追加
	// 今回は簡略化
}

// キャッシュクリーンアップ（期限切れエントリを削除）
func (pa *projectAnalyzer) CleanupExpiredCache() {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()

	now := time.Now()
	for projectPath, cached := range pa.cache {
		if now.After(cached.ExpiresAt) {
			delete(pa.cache, projectPath)

			// ディスクキャッシュも削除
			cacheFile := pa.getCacheFilePath(projectPath)
			if _, err := os.Stat(cacheFile); err == nil {
				os.Remove(cacheFile)
			}
		}
	}
}

// 拡張キャッシュシステム（非同期処理用）

// 拡張分析キャッシュ
type AnalysisCache struct {
	memoryCache map[string]*CachedEntry
	diskCache   *DiskCache
	mutex       sync.RWMutex
	maxEntries  int
	defaultTTL  time.Duration
}

// キャッシュエントリ
type CachedEntry struct {
	Key          string            `json:"key"`
	ProjectPath  string            `json:"project_path"`
	Analysis     *ProjectAnalysis  `json:"analysis"`
	AnalysisType AnalysisType      `json:"analysis_type"`
	CreatedAt    time.Time         `json:"created_at"`
	ExpiresAt    time.Time         `json:"expires_at"`
	LastAccess   time.Time         `json:"last_access"`
	FileHashes   map[string]string `json:"file_hashes"`
	HitCount     int               `json:"hit_count"`
}

// ディスクキャッシュ
type DiskCache struct {
	cacheDir    string
	enabled     bool
	maxSize     int64 // バイト数
	currentSize int64
}

// 新しい拡張分析キャッシュを作成
func NewAnalysisCache() *AnalysisCache {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".vyb", "cache", "analysis")

	// キャッシュディレクトリを作成
	os.MkdirAll(cacheDir, 0755)

	diskCache := &DiskCache{
		cacheDir: cacheDir,
		enabled:  true,
		maxSize:  100 * 1024 * 1024, // 100MB
	}

	cache := &AnalysisCache{
		memoryCache: make(map[string]*CachedEntry),
		diskCache:   diskCache,
		maxEntries:  50,               // メモリ内最大50エントリ
		defaultTTL:  30 * time.Minute, // デフォルト30分
	}

	// 定期クリーンアップを開始
	go cache.startCleanupRoutine()

	return cache
}

// キャッシュからエントリを取得
func (ac *AnalysisCache) Get(projectPath string, analysisType AnalysisType) *ProjectAnalysis {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	key := ac.generateKey(projectPath, analysisType)

	// メモリキャッシュを確認
	if entry, exists := ac.memoryCache[key]; exists {
		if ac.isValidEntry(entry, projectPath) {
			entry.LastAccess = time.Now()
			entry.HitCount++
			return entry.Analysis
		} else {
			// 無効なエントリを削除
			delete(ac.memoryCache, key)
		}
	}

	// ディスクキャッシュを確認
	if ac.diskCache.enabled {
		if entry := ac.loadFromDisk(key); entry != nil {
			if ac.isValidEntry(entry, projectPath) {
				// メモリキャッシュに復元
				ac.memoryCache[key] = entry
				entry.LastAccess = time.Now()
				entry.HitCount++
				return entry.Analysis
			}
		}
	}

	return nil
}

// キャッシュにエントリを保存
func (ac *AnalysisCache) Set(projectPath string, analysisType AnalysisType, analysis *ProjectAnalysis) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	key := ac.generateKey(projectPath, analysisType)

	// ファイルハッシュを計算
	fileHashes := ac.calculateFileHashes(projectPath)

	entry := &CachedEntry{
		Key:          key,
		ProjectPath:  projectPath,
		Analysis:     analysis,
		AnalysisType: analysisType,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(ac.defaultTTL),
		LastAccess:   time.Now(),
		FileHashes:   fileHashes,
		HitCount:     0,
	}

	// メモリキャッシュに保存
	ac.memoryCache[key] = entry

	// メモリキャッシュサイズ制限チェック
	if len(ac.memoryCache) > ac.maxEntries {
		ac.evictLRU()
	}

	// ディスクキャッシュに保存
	if ac.diskCache.enabled {
		go ac.saveToDisk(entry)
	}
}

// キャッシュサイズを取得
func (ac *AnalysisCache) Size() int {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()
	return len(ac.memoryCache)
}

// 内部メソッド

// キーを生成
func (ac *AnalysisCache) generateKey(projectPath string, analysisType AnalysisType) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s_%d", projectPath, analysisType)))
	return fmt.Sprintf("%x", hash)
}

// エントリの有効性をチェック
func (ac *AnalysisCache) isValidEntry(entry *CachedEntry, projectPath string) bool {
	// 期限切れチェック
	if time.Now().After(entry.ExpiresAt) {
		return false
	}

	// ファイル変更チェック
	currentHashes := ac.calculateFileHashes(projectPath)
	return ac.hashesMatch(entry.FileHashes, currentHashes)
}

// ファイルハッシュを計算
func (ac *AnalysisCache) calculateFileHashes(projectPath string) map[string]string {
	hashes := make(map[string]string)

	// 重要なファイルのみハッシュ化（パフォーマンス考慮）
	importantFiles := []string{
		"go.mod", "go.sum",
		"package.json", "package-lock.json",
		"requirements.txt", "setup.py",
		"Dockerfile", ".gitignore",
		"README.md", "Makefile",
	}

	for _, filename := range importantFiles {
		filePath := filepath.Join(projectPath, filename)
		if info, err := os.Stat(filePath); err == nil {
			// ファイルサイズと更新時刻でハッシュ代用（高速）
			hash := fmt.Sprintf("%d_%d", info.Size(), info.ModTime().Unix())
			hashes[filename] = hash
		}
	}

	return hashes
}

// ハッシュの一致チェック
func (ac *AnalysisCache) hashesMatch(oldHashes, newHashes map[string]string) bool {
	// 重要ファイルに変更があるかチェック
	for filename, oldHash := range oldHashes {
		if newHash, exists := newHashes[filename]; !exists || oldHash != newHash {
			return false
		}
	}

	// 新しいファイルが追加されているかチェック
	for filename := range newHashes {
		if _, exists := oldHashes[filename]; !exists {
			return false
		}
	}

	return true
}

// LRU方式で最も古いエントリを削除
func (ac *AnalysisCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, entry := range ac.memoryCache {
		if entry.LastAccess.Before(oldestTime) {
			oldestTime = entry.LastAccess
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(ac.memoryCache, oldestKey)
	}
}

// ディスクに保存
func (ac *AnalysisCache) saveToDisk(entry *CachedEntry) {
	filename := filepath.Join(ac.diskCache.cacheDir, entry.Key+".json")

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	if err := os.WriteFile(filename, data, 0644); err == nil {
		ac.diskCache.currentSize += int64(len(data))
	}
}

// ディスクから読み込み
func (ac *AnalysisCache) loadFromDisk(key string) *CachedEntry {
	filename := filepath.Join(ac.diskCache.cacheDir, key+".json")

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}

	var entry CachedEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}

	return &entry
}

// 定期クリーンアップルーチン
func (ac *AnalysisCache) startCleanupRoutine() {
	ticker := time.NewTicker(10 * time.Minute) // 10分間隔
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ac.performMaintenance()
		}
	}
}

// メンテナンス処理
func (ac *AnalysisCache) performMaintenance() {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	now := time.Now()

	// 期限切れエントリを削除
	for key, entry := range ac.memoryCache {
		if now.After(entry.ExpiresAt) {
			delete(ac.memoryCache, key)
		}
	}
}
