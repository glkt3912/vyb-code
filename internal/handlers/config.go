package handlers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
	"github.com/spf13/cobra"
)

// ConfigHandler は設定管理のハンドラー
type ConfigHandler struct {
	log logger.Logger
}

// NewConfigHandler は設定ハンドラーの新しいインスタンスを作成
func NewConfigHandler(log logger.Logger) *ConfigHandler {
	return &ConfigHandler{log: log}
}

// SetModel はLLMモデルを設定
func (h *ConfigHandler) SetModel(model string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// モデル名の検証
	if model == "" {
		return fmt.Errorf("モデル名が空です")
	}

	cfg.ModelName = model

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	h.log.Info("LLMモデルを更新しました", map[string]interface{}{
		"model": model,
	})
	return nil
}

// SetProvider はLLMプロバイダーを設定
func (h *ConfigHandler) SetProvider(provider string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// プロバイダーの検証
	validProviders := []string{"ollama", "lmstudio", "vllm"}
	isValid := false
	for _, valid := range validProviders {
		if provider == valid {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("無効なプロバイダーです。有効な値: %v", validProviders)
	}

	cfg.Provider = provider

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	h.log.Info("LLMプロバイダーを更新しました", map[string]interface{}{
		"provider": provider,
	})
	return nil
}

// ListConfig は現在の設定を表示
func (h *ConfigHandler) ListConfig() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// コンソールに直接出力（ログシステムを使わない）
	fmt.Println("現在の設定:")
	fmt.Printf("  Provider: %s\n", cfg.Provider)
	fmt.Printf("  Model: %s\n", cfg.ModelName)
	fmt.Printf("  Base URL: %s\n", cfg.BaseURL)
	fmt.Printf("  Max Tokens: %d\n", cfg.MaxTokens)
	fmt.Printf("  Temperature: %g\n", cfg.Temperature)
	fmt.Printf("  Stream: %t\n", cfg.Stream)
	fmt.Printf("  Log Level: %s\n", cfg.Log.Level)
	fmt.Printf("  Log Format: %s\n", cfg.Log.Format)
	fmt.Printf("  TUI Enabled: %t (deprecated - Claude Code風インターフェースが標準)\n", cfg.TUI.Enabled)
	fmt.Printf("  TUI Theme: %s (deprecated - Claude Code風インターフェースが標準)\n", cfg.TUI.Theme)
	fmt.Printf("  File Max Size (MB): %d\n", cfg.FileMaxSizeMB)
	fmt.Printf("  Command Timeout: %d\n", cfg.CommandTimeout)

	// 段階的移行設定
	fmt.Println("  Migration Settings:")
	fmt.Printf("    Migration Mode: %s\n", cfg.Migration.MigrationMode)
	fmt.Printf("    Unified Streaming: %t\n", cfg.Migration.UseUnifiedStreaming)
	fmt.Printf("    Unified Session: %t\n", cfg.Migration.UseUnifiedSession)
	fmt.Printf("    Unified Tools: %t\n", cfg.Migration.UseUnifiedTools)
	fmt.Printf("    Unified Analysis: %t\n", cfg.Migration.UseUnifiedAnalysis)
	fmt.Printf("    Validation Enabled: %t\n", cfg.Migration.EnableValidation)
	fmt.Printf("    Fallback Enabled: %t\n", cfg.Migration.EnableFallback)
	fmt.Printf("    Metrics Enabled: %t\n", cfg.Migration.EnableMetrics)

	return nil
}

// SetLogLevel はログレベルを設定
func (h *ConfigHandler) SetLogLevel(level string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// ログレベルの検証
	validLevels := []string{"debug", "info", "warn", "error"}
	isValid := false
	for _, valid := range validLevels {
		if level == valid {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("無効なログレベルです。有効な値: %v", validLevels)
	}

	cfg.Log.Level = level

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	h.log.Info("ログレベルを更新しました", map[string]interface{}{
		"level": level,
	})
	return nil
}

// SetLogFormat はログフォーマットを設定
func (h *ConfigHandler) SetLogFormat(format string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// ログフォーマットの検証
	validFormats := []string{"console", "json"}
	isValid := false
	for _, valid := range validFormats {
		if format == valid {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("無効なログフォーマットです。有効な値: %v", validFormats)
	}

	cfg.Log.Format = format

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	h.log.Info("ログフォーマットを更新しました", map[string]interface{}{
		"format": format,
	})
	return nil
}

// SetTUIEnabled はTUIの有効/無効を設定（非推奨 - Claude Code風インターフェースに移行済み）
func (h *ConfigHandler) SetTUIEnabled(enabled bool) error {
	h.log.Warn("TUI設定は非推奨です。Claude Code風インターフェースが常に使用されます。", nil)
	fmt.Println("⚠️  TUI設定は非推奨です。Claude Code風インターフェースが標準となりました。")
	return nil
}

// SetTUITheme はTUIテーマを設定（非推奨 - Claude Code風インターフェースに移行済み）
func (h *ConfigHandler) SetTUITheme(theme string) error {
	h.log.Warn("TUIテーマ設定は非推奨です。Claude Code風インターフェースが常に使用されます。", nil)
	fmt.Println("⚠️  TUIテーマ設定は非推奨です。Claude Code風インターフェースが標準となりました。")
	return nil
}

// 段階的移行設定のメソッド

// SetMigrationMode は移行モードを設定
func (h *ConfigHandler) SetMigrationMode(mode string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	validModes := []string{"gradual", "compatibility", "unified"}
	isValid := false
	for _, valid := range validModes {
		if mode == valid {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("無効な移行モードです。有効な値: %v", validModes)
	}

	cfg.Migration.MigrationMode = mode

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	h.log.Info("移行モードを更新しました", map[string]interface{}{
		"mode": mode,
	})
	return nil
}

// EnableUnifiedStreaming は統合ストリーミングシステムを有効化
func (h *ConfigHandler) EnableUnifiedStreaming(enable bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	cfg.Migration.UseUnifiedStreaming = enable

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	status := "無効化"
	if enable {
		status = "有効化"
	}

	h.log.Info("統合ストリーミングシステムを"+status+"しました", map[string]interface{}{
		"enabled": enable,
	})
	return nil
}

// EnableUnifiedSession は統合セッション管理を有効化
func (h *ConfigHandler) EnableUnifiedSession(enable bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	cfg.Migration.UseUnifiedSession = enable

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	status := "無効化"
	if enable {
		status = "有効化"
	}

	h.log.Info("統合セッション管理を"+status+"しました", map[string]interface{}{
		"enabled": enable,
	})
	return nil
}

// EnableUnifiedTools は統合ツールシステムを有効化
func (h *ConfigHandler) EnableUnifiedTools(enable bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	cfg.Migration.UseUnifiedTools = enable

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	status := "無効化"
	if enable {
		status = "有効化"
	}

	h.log.Info("統合ツールシステムを"+status+"しました", map[string]interface{}{
		"enabled": enable,
	})
	return nil
}

// EnableUnifiedAnalysis は統合分析システムを有効化
func (h *ConfigHandler) EnableUnifiedAnalysis(enable bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	cfg.Migration.UseUnifiedAnalysis = enable

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	status := "無効化"
	if enable {
		status = "有効化"
	}

	h.log.Info("統合分析システムを"+status+"しました", map[string]interface{}{
		"enabled": enable,
	})
	return nil
}

// EnableValidation は検証機能を有効化
func (h *ConfigHandler) EnableValidation(enable bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	cfg.Migration.EnableValidation = enable

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	status := "無効化"
	if enable {
		status = "有効化"
	}

	h.log.Info("検証機能を"+status+"しました", map[string]interface{}{
		"enabled": enable,
	})
	return nil
}

// CreateConfigCommands は設定関連のcobraコマンドを作成
func (h *ConfigHandler) CreateConfigCommands() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage vyb configuration",
	}

	// set-model コマンド
	setModelCmd := &cobra.Command{
		Use:   "set-model [model]",
		Short: "Set the LLM model to use",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.SetModel(args[0])
		},
	}

	// set-provider コマンド
	setProviderCmd := &cobra.Command{
		Use:   "set-provider [provider]",
		Short: "Set the LLM provider (ollama, lmstudio, vllm)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.SetProvider(args[0])
		},
	}

	// list コマンド
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.ListConfig()
		},
	}

	// set-log-level コマンド
	setLogLevelCmd := &cobra.Command{
		Use:   "set-log-level [level]",
		Short: "Set log level (debug, info, warn, error)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.SetLogLevel(args[0])
		},
	}

	// set-log-format コマンド
	setLogFormatCmd := &cobra.Command{
		Use:   "set-log-format [format]",
		Short: "Set log format (console, json)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.SetLogFormat(args[0])
		},
	}

	// set-tui コマンド（非推奨）
	setTUICmd := &cobra.Command{
		Use:   "set-tui [true|false]",
		Short: "Enable or disable TUI mode (deprecated - Claude Code風インターフェースが標準)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, err := strconv.ParseBool(args[0])
			if err != nil {
				return fmt.Errorf("invalid boolean value: %s", args[0])
			}
			return h.SetTUIEnabled(enabled)
		},
	}

	// set-tui-theme コマンド（非推奨）
	setTUIThemeCmd := &cobra.Command{
		Use:   "set-tui-theme [theme]",
		Short: "Set TUI theme (deprecated - Claude Code風インターフェースが標準)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.SetTUITheme(args[0])
		},
	}

	// 段階的移行設定コマンド
	setMigrationModeCmd := &cobra.Command{
		Use:   "set-migration-mode [mode]",
		Short: "Set the gradual migration mode (gradual, compatibility, unified)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.SetMigrationMode(args[0])
		},
	}

	enableUnifiedStreamingCmd := &cobra.Command{
		Use:   "enable-unified-streaming [true|false]",
		Short: "Enable or disable unified streaming system",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			enable, err := strconv.ParseBool(args[0])
			if err != nil {
				return fmt.Errorf("無効な値です。true または false を指定してください")
			}
			return h.EnableUnifiedStreaming(enable)
		},
	}

	enableUnifiedSessionCmd := &cobra.Command{
		Use:   "enable-unified-session [true|false]",
		Short: "Enable or disable unified session management",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			enable, err := strconv.ParseBool(args[0])
			if err != nil {
				return fmt.Errorf("無効な値です。true または false を指定してください")
			}
			return h.EnableUnifiedSession(enable)
		},
	}

	enableUnifiedToolsCmd := &cobra.Command{
		Use:   "enable-unified-tools [true|false]",
		Short: "Enable or disable unified tools system",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			enable, err := strconv.ParseBool(args[0])
			if err != nil {
				return fmt.Errorf("無効な値です。true または false を指定してください")
			}
			return h.EnableUnifiedTools(enable)
		},
	}

	enableUnifiedAnalysisCmd := &cobra.Command{
		Use:   "enable-unified-analysis [true|false]",
		Short: "Enable or disable unified analysis system",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			enable, err := strconv.ParseBool(args[0])
			if err != nil {
				return fmt.Errorf("無効な値です。true または false を指定してください")
			}
			return h.EnableUnifiedAnalysis(enable)
		},
	}

	enableValidationCmd := &cobra.Command{
		Use:   "enable-validation [true|false]",
		Short: "Enable or disable migration validation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			enable, err := strconv.ParseBool(args[0])
			if err != nil {
				return fmt.Errorf("無効な値です。true または false を指定してください")
			}
			return h.EnableValidation(enable)
		},
	}

	// サブコマンドを追加
	configCmd.AddCommand(setModelCmd, setProviderCmd, listCmd)
	configCmd.AddCommand(setLogLevelCmd, setLogFormatCmd)
	configCmd.AddCommand(setTUICmd, setTUIThemeCmd)

	// 段階的移行コマンドを追加
	configCmd.AddCommand(setMigrationModeCmd)
	configCmd.AddCommand(enableUnifiedStreamingCmd, enableUnifiedSessionCmd)
	configCmd.AddCommand(enableUnifiedToolsCmd, enableUnifiedAnalysisCmd)
	configCmd.AddCommand(enableValidationCmd)

	return configCmd
}

// Handler インターフェース実装

// Initialize はハンドラーを初期化
func (h *ConfigHandler) Initialize(cfg *config.Config) error {
	// ConfigHandlerは特別な初期化を必要としない
	return nil
}

// GetMetadata はハンドラーのメタデータを返す
func (h *ConfigHandler) GetMetadata() HandlerMetadata {
	return HandlerMetadata{
		Name:        "config",
		Version:     "1.0.0",
		Description: "設定管理ハンドラー",
		Capabilities: []string{
			"config_management",
			"model_selection",
			"provider_settings",
			"logging_config",
			"proactive_settings",
		},
		Dependencies: []string{
			"config",
		},
		Config: map[string]string{
			"storage_type": "json_file",
			"auto_save":    "true",
		},
	}
}

// Health はハンドラーの健全性をチェック
func (h *ConfigHandler) Health(ctx context.Context) error {
	if h.log == nil {
		return fmt.Errorf("logger not initialized")
	}

	// 設定ファイルへのアクセステスト
	_, err := config.Load()
	if err != nil {
		return fmt.Errorf("config file access failed: %w", err)
	}

	return nil
}
