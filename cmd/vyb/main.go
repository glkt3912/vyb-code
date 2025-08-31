package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/glkt/vyb-code/internal/chat"
	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/llm"
)

// メインコマンド：vyb単体で実行される処理
var rootCmd = &cobra.Command{
	Use:   "vyb",
	Short: "Local AI coding assistant",
	Long:  `vyb - Feel the rhythm of perfect code. A local LLM-based coding assistant that prioritizes privacy and developer experience.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// 引数なし：対話モード開始
			startInteractiveMode()
		} else {
			// 引数あり：単発コマンド処理
			query := args[0]
			processSingleQuery(query)
		}
	},
}

// チャットコマンド：明示的に対話モードを開始
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive chat mode",
	Run: func(cmd *cobra.Command, args []string) {
		startInteractiveMode()
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
		setModel(model)
	},
}

// プロバイダー設定コマンド：vyb config set-provider ollama
var setProviderCmd = &cobra.Command{
	Use:   "set-provider [provider]",
	Short: "Set the LLM provider (ollama, lmstudio, vllm)",
	Args:  cobra.ExactArgs(1), // 引数を1つだけ受け取る
	Run: func(cmd *cobra.Command, args []string) {
		provider := args[0] // 最初の引数をプロバイダー名として取得
		setProvider(provider)
	},
}

var listConfigCmd = &cobra.Command{
	Use:   "list",
	Short: "List current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		listConfig()
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

// 対話モードを開始する実装関数
func startInteractiveMode() {
	fmt.Println("🎵 vyb - Feel the rhythm of perfect code")
	
	// 設定を読み込み
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}
	
	// LLMクライアントを作成
	provider := llm.NewOllamaClient(cfg.BaseURL)
	
	// チャットセッションを開始
	session := chat.NewSession(provider, cfg.Model)
	if err := session.StartInteractive(); err != nil {
		fmt.Printf("対話セッションエラー: %v\n", err)
	}
}

// 単発クエリを処理する実装関数
func processSingleQuery(query string) {
	fmt.Printf("Processing: %s\n", query)
	
	// 設定を読み込み
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}
	
	// LLMクライアントを作成
	provider := llm.NewOllamaClient(cfg.BaseURL)
	
	// チャットセッションで単発処理
	session := chat.NewSession(provider, cfg.Model)
	if err := session.ProcessQuery(query); err != nil {
		fmt.Printf("クエリ処理エラー: %v\n", err)
	}
}

// モデルを設定する実装関数
func setModel(model string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}
	
	if err := cfg.SetModel(model); err != nil {
		fmt.Printf("モデル設定エラー: %v\n", err)
		return
	}
	
	fmt.Printf("モデルを %s に設定しました\n", model)
}

// プロバイダーを設定する実装関数
func setProvider(provider string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}
	
	if err := cfg.SetProvider(provider); err != nil {
		fmt.Printf("プロバイダー設定エラー: %v\n", err)
		return
	}
	
	fmt.Printf("プロバイダーを %s に設定しました\n", provider)
}

// 現在の設定を表示する実装関数
func listConfig() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("設定読み込みエラー: %v\n", err)
		return
	}
	
	fmt.Println("現在の設定:")
	fmt.Printf("  プロバイダー: %s\n", cfg.Provider)
	fmt.Printf("  モデル: %s\n", cfg.Model)
	fmt.Printf("  ベースURL: %s\n", cfg.BaseURL)
	fmt.Printf("  タイムアウト: %d秒\n", cfg.Timeout)
	fmt.Printf("  最大ファイルサイズ: %d MB\n", cfg.MaxFileSize/(1024*1024))
}
