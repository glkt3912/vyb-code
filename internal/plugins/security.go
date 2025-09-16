package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/glkt/vyb-code/internal/logger"
)

// PluginSecurity はプラグインのセキュリティ管理を行う
type PluginSecurity struct {
	logger          logger.Logger
	policy          SecurityPolicy
	trustedHashes   map[string]string // プラグインの信頼できるハッシュ
	blacklistedNames []string          // ブラックリストされたプラグイン名
	whitelistedPaths []string          // 許可されたパス
}

// SecurityPolicy はセキュリティポリシーの設定
type SecurityPolicy struct {
	Level               SecurityLevel `json:"level"`
	RequireSignature    bool          `json:"require_signature"`
	RequireHashCheck    bool          `json:"require_hash_check"`
	AllowUnsignedLocal  bool          `json:"allow_unsigned_local"`
	MaxPluginSize       int64         `json:"max_plugin_size"`
	AllowedExtensions   []string      `json:"allowed_extensions"`
	RestrictedPaths     []string      `json:"restricted_paths"`
}

// SecurityLevel はセキュリティレベル
type SecurityLevel int

const (
	SecurityLevelLow SecurityLevel = iota
	SecurityLevelModerate
	SecurityLevelHigh
	SecurityLevelStrict
)

// String はSecurityLevelの文字列表現を返す
func (s SecurityLevel) String() string {
	switch s {
	case SecurityLevelLow:
		return "low"
	case SecurityLevelModerate:
		return "moderate"
	case SecurityLevelHigh:
		return "high"
	case SecurityLevelStrict:
		return "strict"
	default:
		return "unknown"
	}
}

// NewPluginSecurity は新しいセキュリティマネージャーを作成
func NewPluginSecurity(logger logger.Logger) *PluginSecurity {
	return &PluginSecurity{
		logger:        logger,
		trustedHashes: make(map[string]string),
		policy: SecurityPolicy{
			Level:             SecurityLevelModerate,
			RequireSignature:  false,
			RequireHashCheck:  true,
			AllowUnsignedLocal: true,
			MaxPluginSize:     50 * 1024 * 1024, // 50MB
			AllowedExtensions: []string{".so", ".dll", ".dylib"},
			RestrictedPaths:   []string{"/system", "/etc", "/var"},
		},
	}
}

// Initialize はセキュリティマネージャーを初期化
func (s *PluginSecurity) Initialize(ctx context.Context) error {
	s.logger.Info("プラグインセキュリティ初期化開始", map[string]interface{}{
		"policy_level": s.policy.Level.String(),
	})

	// デフォルトの信頼できるハッシュを読み込み
	if err := s.loadTrustedHashes(); err != nil {
		s.logger.Warn("信頼ハッシュ読み込み警告", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// ブラックリストを読み込み
	if err := s.loadBlacklist(); err != nil {
		s.logger.Warn("ブラックリスト読み込み警告", map[string]interface{}{
			"error": err.Error(),
		})
	}

	s.logger.Info("プラグインセキュリティ初期化完了", nil)
	return nil
}

// loadTrustedHashes は信頼できるハッシュを読み込み
func (s *PluginSecurity) loadTrustedHashes() error {
	// 実装では設定ファイルから読み込み
	// 現在はデフォルト値を設定
	s.trustedHashes["example_plugin"] = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	return nil
}

// loadBlacklist はブラックリストを読み込み
func (s *PluginSecurity) loadBlacklist() error {
	// 実装では設定ファイルから読み込み
	// 現在はデフォルト値を設定
	s.blacklistedNames = []string{"malware_plugin", "dangerous_tool"}
	return nil
}

// ValidatePlugin はプラグインのセキュリティを検証
func (s *PluginSecurity) ValidatePlugin(pluginName string) error {
	s.logger.Debug("プラグインセキュリティ検証開始", map[string]interface{}{
		"plugin": pluginName,
	})

	// ブラックリストチェック
	if err := s.checkBlacklist(pluginName); err != nil {
		return fmt.Errorf("ブラックリストチェック失敗: %w", err)
	}

	s.logger.Debug("プラグインセキュリティ検証完了", map[string]interface{}{
		"plugin": pluginName,
	})

	return nil
}

// ValidatePluginFile はプラグインファイルのセキュリティを検証
func (s *PluginSecurity) ValidatePluginFile(filePath string) error {
	s.logger.Debug("プラグインファイル検証開始", map[string]interface{}{
		"file": filePath,
	})

	// ファイル存在チェック
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("ファイルアクセスエラー: %w", err)
	}

	// パス検証
	if err := s.validatePath(filePath); err != nil {
		return fmt.Errorf("パス検証失敗: %w", err)
	}

	// 拡張子チェック
	if err := s.validateExtension(filePath); err != nil {
		return fmt.Errorf("拡張子検証失敗: %w", err)
	}

	// ファイルサイズチェック
	if err := s.validateFileSize(filePath); err != nil {
		return fmt.Errorf("ファイルサイズ検証失敗: %w", err)
	}

	// ハッシュチェック（必要な場合）
	if s.policy.RequireHashCheck {
		if err := s.validateHash(filePath); err != nil {
			return fmt.Errorf("ハッシュ検証失敗: %w", err)
		}
	}

	s.logger.Debug("プラグインファイル検証完了", map[string]interface{}{
		"file": filePath,
	})

	return nil
}

// checkBlacklist はブラックリストをチェック
func (s *PluginSecurity) checkBlacklist(pluginName string) error {
	for _, blocked := range s.blacklistedNames {
		if strings.EqualFold(pluginName, blocked) {
			return fmt.Errorf("プラグイン %s はブラックリストに登録されています", pluginName)
		}
	}
	return nil
}

// validatePath はファイルパスを検証
func (s *PluginSecurity) validatePath(filePath string) error {
	// 絶対パスに変換
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("絶対パス変換エラー: %w", err)
	}

	// 制限されたパスをチェック
	for _, restricted := range s.policy.RestrictedPaths {
		if strings.HasPrefix(absPath, restricted) {
			return fmt.Errorf("制限されたパス %s にアクセスしようとしています", restricted)
		}
	}

	// パストラバーサル攻撃をチェック
	if strings.Contains(absPath, "..") {
		return fmt.Errorf("パストラバーサル攻撃の可能性があります")
	}

	return nil
}

// validateExtension は拡張子を検証
func (s *PluginSecurity) validateExtension(filePath string) error {
	ext := filepath.Ext(filePath)
	
	for _, allowed := range s.policy.AllowedExtensions {
		if strings.EqualFold(ext, allowed) {
			return nil
		}
	}

	return fmt.Errorf("許可されていない拡張子 %s です", ext)
}

// validateFileSize はファイルサイズを検証
func (s *PluginSecurity) validateFileSize(filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("ファイル情報取得エラー: %w", err)
	}

	if fileInfo.Size() > s.policy.MaxPluginSize {
		return fmt.Errorf("ファイルサイズ %d は制限 %d を超えています", 
			fileInfo.Size(), s.policy.MaxPluginSize)
	}

	return nil
}

// validateHash はファイルハッシュを検証
func (s *PluginSecurity) validateHash(filePath string) error {
	// ファイルハッシュを計算
	hash, err := s.calculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("ハッシュ計算エラー: %w", err)
	}

	// プラグイン名を取得
	pluginName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	// 信頼できるハッシュと比較
	if trustedHash, exists := s.trustedHashes[pluginName]; exists {
		if hash != trustedHash {
			return fmt.Errorf("ハッシュが一致しません。期待: %s, 実際: %s", trustedHash, hash)
		}
	} else {
		// 信頼できるハッシュが存在しない場合の処理
		if s.policy.Level >= SecurityLevelHigh {
			return fmt.Errorf("プラグイン %s の信頼できるハッシュが登録されていません", pluginName)
		}
		
		s.logger.Warn("信頼ハッシュ未登録", map[string]interface{}{
			"plugin": pluginName,
			"hash":   hash,
		})
	}

	return nil
}

// calculateFileHash はファイルのSHA256ハッシュを計算
func (s *PluginSecurity) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("ファイルオープンエラー: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("ハッシュ計算エラー: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// AddTrustedHash は信頼できるハッシュを追加
func (s *PluginSecurity) AddTrustedHash(pluginName, hash string) {
	s.trustedHashes[pluginName] = hash
	s.logger.Info("信頼ハッシュ追加", map[string]interface{}{
		"plugin": pluginName,
		"hash":   hash[:16] + "...", // 先頭16文字のみログ出力
	})
}

// RemoveTrustedHash は信頼できるハッシュを削除
func (s *PluginSecurity) RemoveTrustedHash(pluginName string) {
	delete(s.trustedHashes, pluginName)
	s.logger.Info("信頼ハッシュ削除", map[string]interface{}{
		"plugin": pluginName,
	})
}

// AddToBlacklist はブラックリストに追加
func (s *PluginSecurity) AddToBlacklist(pluginName string) {
	for _, existing := range s.blacklistedNames {
		if strings.EqualFold(existing, pluginName) {
			return // 既に存在
		}
	}

	s.blacklistedNames = append(s.blacklistedNames, pluginName)
	s.logger.Info("ブラックリスト追加", map[string]interface{}{
		"plugin": pluginName,
	})
}

// RemoveFromBlacklist はブラックリストから削除
func (s *PluginSecurity) RemoveFromBlacklist(pluginName string) {
	for i, existing := range s.blacklistedNames {
		if strings.EqualFold(existing, pluginName) {
			s.blacklistedNames = append(s.blacklistedNames[:i], s.blacklistedNames[i+1:]...)
			s.logger.Info("ブラックリスト削除", map[string]interface{}{
				"plugin": pluginName,
			})
			return
		}
	}
}

// SetSecurityLevel はセキュリティレベルを設定
func (s *PluginSecurity) SetSecurityLevel(level SecurityLevel) {
	oldLevel := s.policy.Level
	s.policy.Level = level

	s.logger.Info("セキュリティレベル変更", map[string]interface{}{
		"old_level": oldLevel.String(),
		"new_level": level.String(),
	})
}

// GetSecurityInfo はセキュリティ情報を取得
func (s *PluginSecurity) GetSecurityInfo() SecurityInfo {
	return SecurityInfo{
		Policy:           s.policy,
		TrustedCount:     len(s.trustedHashes),
		BlacklistedCount: len(s.blacklistedNames),
	}
}

// SecurityInfo はセキュリティ情報
type SecurityInfo struct {
	Policy           SecurityPolicy `json:"policy"`
	TrustedCount     int            `json:"trusted_count"`
	BlacklistedCount int            `json:"blacklisted_count"`
}