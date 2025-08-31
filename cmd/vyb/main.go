package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// メインコマンド：vyb単体で実行される処理
var rootCmd = &cobra.Command{
	Use:   "vyb",
	Short: "Local AI coding assistant",
	Long:  `vyb - Feel the rhythm of perfect code. A local LLM-based coding assistant that prioritizes privacy and developer experience.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// 引数なし：対話モード開始
			fmt.Println("🎵 vyb - Feel the rhythm of perfect code")
			fmt.Println("Starting interactive mode...")
			// TODO: 対話チャットループの実装
		} else {
			// 引数あり：単発コマンド処理
			query := args[0]
			fmt.Printf("Processing: %s\n", query)
			// TODO: 単発クエリの処理
		}
	},
}

// チャットコマンド：明示的に対話モードを開始
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive chat mode",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🎵 vyb - Feel the rhythm of perfect code")
		fmt.Println("Starting interactive chat mode...")
		// TODO: 対話チャットループの実装
	},
}

// 設定管理のメインコマンド
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage vyb configuration",
}

// モデル設定コマンド：vyb config set-model qwen2.5-coder:14b
var setModelCmd = &cobra.Command{
	Use:   "set-model [model]",
	Short: "Set the LLM model to use",
	Args:  cobra.ExactArgs(1), // 引数を1つだけ受け取る
	Run: func(cmd *cobra.Command, args []string) {
		model := args[0] // 最初の引数をモデル名として取得
		fmt.Printf("Setting model to: %s\n", model)
		// TODO: モデル設定の実装
	},
}

// プロバイダー設定コマンド：vyb config set-provider ollama
var setProviderCmd = &cobra.Command{
	Use:   "set-provider [provider]",
	Short: "Set the LLM provider (ollama, lmstudio, vllm)",
	Args:  cobra.ExactArgs(1), // 引数を1つだけ受け取る
	Run: func(cmd *cobra.Command, args []string) {
		provider := args[0]
		fmt.Printf("Setting provider to: %s\n", provider)
		// TODO: プロバイダー設定の実装
	},
}

var listConfigCmd = &cobra.Command{
	Use:   "list",
	Short: "List current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Current configuration:")
		// TODO: 設定内容の表示実装
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(setModelCmd)
	configCmd.AddCommand(setProviderCmd)
	configCmd.AddCommand(listConfigCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
