package handlers

import (
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

	h.log.Info("現在の設定", map[string]interface{}{
		"provider":         cfg.Provider,
		"model_name":       cfg.ModelName,
		"base_url":         cfg.BaseURL,
		"max_tokens":       cfg.MaxTokens,
		"temperature":      cfg.Temperature,
		"stream":           cfg.Stream,
		"log_level":        cfg.Log.Level,
		"log_format":       cfg.Log.Format,
		"tui_enabled":      cfg.TUI.Enabled,
		"tui_theme":        cfg.TUI.Theme,
		"terminal_mode":    cfg.TerminalMode,
		"file_max_size_mb": cfg.FileMaxSizeMB,
		"command_timeout":  cfg.CommandTimeout,
	})
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

// SetTUIEnabled はTUIの有効/無効を設定
func (h *ConfigHandler) SetTUIEnabled(enabled bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	cfg.TUI.Enabled = enabled

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	h.log.Info("TUI設定を更新しました", map[string]interface{}{
		"enabled": enabled,
	})
	return nil
}

// SetTUITheme はTUIテーマを設定
func (h *ConfigHandler) SetTUITheme(theme string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定読み込みエラー: %w", err)
	}

	// テーマの検証
	validThemes := []string{"dark", "light", "auto", "vyb"}
	isValid := false
	for _, valid := range validThemes {
		if theme == valid {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("無効なテーマです。有効な値: %v", validThemes)
	}

	cfg.TUI.Theme = theme

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("設定保存エラー: %w", err)
	}

	h.log.Info("TUIテーマを更新しました", map[string]interface{}{
		"theme": theme,
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

	// set-tui コマンド
	setTUICmd := &cobra.Command{
		Use:   "set-tui [true|false]",
		Short: "Enable or disable TUI mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, err := strconv.ParseBool(args[0])
			if err != nil {
				return fmt.Errorf("invalid boolean value: %s", args[0])
			}
			return h.SetTUIEnabled(enabled)
		},
	}

	// set-tui-theme コマンド
	setTUIThemeCmd := &cobra.Command{
		Use:   "set-tui-theme [theme]",
		Short: "Set TUI theme (dark, light, auto, vyb)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.SetTUITheme(args[0])
		},
	}

	// サブコマンドを追加
	configCmd.AddCommand(setModelCmd, setProviderCmd, listCmd)
	configCmd.AddCommand(setLogLevelCmd, setLogFormatCmd)
	configCmd.AddCommand(setTUICmd, setTUIThemeCmd)

	return configCmd
}
