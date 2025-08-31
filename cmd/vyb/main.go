package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vyb",
	Short: "Local AI coding assistant",
	Long:  `vyb - Feel the rhythm of perfect code. A local LLM-based coding assistant that prioritizes privacy and developer experience.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// Start interactive mode
			fmt.Println("🎵 vyb - Feel the rhythm of perfect code")
			fmt.Println("Starting interactive mode...")
			// TODO: Implement interactive chat loop
		} else {
			// Handle single command
			query := args[0]
			fmt.Printf("Processing: %s\n", query)
			// TODO: Process single query
		}
	},
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive chat mode",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🎵 vyb - Feel the rhythm of perfect code")
		fmt.Println("Starting interactive chat mode...")
		// TODO: Implement interactive chat loop
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage vyb configuration",
}

var setModelCmd = &cobra.Command{
	Use:   "set-model [model]",
	Short: "Set the LLM model to use",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		model := args[0]
		fmt.Printf("Setting model to: %s\n", model)
		// TODO: Implement model configuration
	},
}

var setProviderCmd = &cobra.Command{
	Use:   "set-provider [provider]",
	Short: "Set the LLM provider (ollama, lmstudio, vllm)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := args[0]
		fmt.Printf("Setting provider to: %s\n", provider)
		// TODO: Implement provider configuration
	},
}

var listConfigCmd = &cobra.Command{
	Use:   "list",
	Short: "List current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Current configuration:")
		// TODO: Implement configuration listing
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