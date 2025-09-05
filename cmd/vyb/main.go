package main

import (
	"fmt"
	"os"

	"github.com/glkt/vyb-code/internal/container"
	"github.com/spf13/cobra"
)

var (
	// グローバルコンテナー
	appContainer *container.Container
)

// メインコマンド：vyb単体で実行される処理
var rootCmd = &cobra.Command{
	Use:     "vyb",
	Short:   "Local AI coding assistant",
	Long:    `vyb - Feel the rhythm of perfect code. A local LLM-based coding assistant that prioritizes privacy and developer experience.`,
	Version: "v1.0.1",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// コンテナー初期化
		appContainer = container.NewContainer()
		return appContainer.Initialize()
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// コンテナークリーンアップ
		if appContainer != nil {
			return appContainer.Shutdown()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// フラグをチェック
		noTUI, _ := cmd.Flags().GetBool("no-tui")
		terminalMode, _ := cmd.Flags().GetBool("terminal-mode")
		noTerminalMode, _ := cmd.Flags().GetBool("no-terminal-mode")
		planMode, _ := cmd.Flags().GetBool("plan-mode")
		continueSession, _ := cmd.Flags().GetBool("continue")
		resumeID, _ := cmd.Flags().GetString("resume")

		// terminal-modeのロジック調整（デフォルトtrue、no-terminal-modeでfalse）
		if noTerminalMode {
			terminalMode = false
		} else {
			terminalMode = true // デフォルトでterminal-mode
		}

		chatHandler, err := appContainer.GetChatHandler()
		if err != nil {
			return fmt.Errorf("チャットハンドラー取得エラー: %w", err)
		}

		if len(args) == 0 {
			// 引数なし：対話モード開始
			return chatHandler.StartInteractiveModeWithOptions(noTUI, terminalMode, planMode, continueSession, resumeID)
		} else {
			// 引数あり：単発コマンド処理
			query := args[0]
			return chatHandler.ProcessSingleQueryWithOptions(query, noTUI, continueSession, resumeID)
		}
	},
}

// チャットコマンド：明示的に対話モードを開始
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive chat mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		noTUI, _ := cmd.Flags().GetBool("no-tui")
		terminalMode, _ := cmd.Flags().GetBool("terminal-mode")
		noTerminalMode, _ := cmd.Flags().GetBool("no-terminal-mode")
		planMode, _ := cmd.Flags().GetBool("plan-mode")
		continueSession, _ := cmd.Flags().GetBool("continue")
		resumeID, _ := cmd.Flags().GetString("resume")

		// terminal-modeのロジック調整
		if noTerminalMode {
			terminalMode = false
		} else {
			terminalMode = true
		}

		chatHandler, err := appContainer.GetChatHandler()
		if err != nil {
			return fmt.Errorf("チャットハンドラー取得エラー: %w", err)
		}

		return chatHandler.StartInteractiveModeWithOptions(noTUI, terminalMode, planMode, continueSession, resumeID)
	},
}

func init() {
	// ルートコマンドにフラグを追加
	rootCmd.PersistentFlags().Bool("no-tui", false, "Disable TUI mode")
	rootCmd.PersistentFlags().Bool("terminal-mode", false, "Enable Claude Code-style terminal mode")
	rootCmd.PersistentFlags().Bool("no-terminal-mode", false, "Disable terminal mode")
	rootCmd.PersistentFlags().Bool("plan-mode", false, "Enable plan mode")
	rootCmd.PersistentFlags().Bool("continue", false, "Continue previous session")
	rootCmd.PersistentFlags().String("resume", "", "Resume specific session ID")

	// チャットコマンドにフラグを追加
	chatCmd.Flags().Bool("no-tui", false, "Disable TUI mode")
	chatCmd.Flags().Bool("terminal-mode", false, "Enable Claude Code-style terminal mode")
	chatCmd.Flags().Bool("no-terminal-mode", false, "Disable terminal mode")
	chatCmd.Flags().Bool("plan-mode", false, "Enable plan mode")
	chatCmd.Flags().Bool("continue", false, "Continue previous session")
	chatCmd.Flags().String("resume", "", "Resume specific session ID")

	// サブコマンドを追加（これらは初期化時に動的に追加される）
	rootCmd.AddCommand(chatCmd)
}

func main() {
	// コマンドの動的構築
	if err := buildCommands(); err != nil {
		fmt.Fprintf(os.Stderr, "コマンド構築エラー: %v\n", err)
		os.Exit(1)
	}

	// コマンド実行
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "実行エラー: %v\n", err)
		os.Exit(1)
	}
}

// buildCommands は動的にコマンドを構築
func buildCommands() error {
	// 一時的なコンテナーでコマンドを構築
	tempContainer := container.NewContainer()
	if err := tempContainer.Initialize(); err != nil {
		return fmt.Errorf("一時コンテナー初期化エラー: %w", err)
	}
	defer tempContainer.Shutdown()

	// 設定コマンド
	configHandler, err := tempContainer.GetConfigHandler()
	if err != nil {
		return fmt.Errorf("設定ハンドラー取得エラー: %w", err)
	}
	configCmd := configHandler.CreateConfigCommands()
	rootCmd.AddCommand(configCmd)

	// ツールコマンド
	toolsHandler, err := tempContainer.GetToolsHandler()
	if err != nil {
		return fmt.Errorf("ツールハンドラー取得エラー: %w", err)
	}
	toolCommands := toolsHandler.CreateToolCommands()
	for _, cmd := range toolCommands {
		rootCmd.AddCommand(cmd)
	}

	// Gitコマンド
	gitHandler, err := tempContainer.GetGitHandler()
	if err != nil {
		return fmt.Errorf("Gitハンドラー取得エラー: %w", err)
	}
	gitCmd := gitHandler.CreateGitCommands()
	rootCmd.AddCommand(gitCmd)

	return nil
}
