package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// 確認ダイアログモデル（Claude Code風シンプル版）
type ConfirmationModel struct {
	Title     string
	Message   string
	Options   []string
	Selected  int
	Confirmed bool
	Cancelled bool
}

// 新しい確認ダイアログを作成
func NewConfirmationDialog(title, message string, options []string) ConfirmationModel {
	if len(options) == 0 {
		options = []string{"はい", "いいえ"}
	}

	return ConfirmationModel{
		Title:    title,
		Message:  message,
		Options:  options,
		Selected: 1, // デフォルトは「いいえ」
	}
}

// 標準的な実行確認ダイアログ
func NewExecutionConfirmDialog(command string) ConfirmationModel {
	title := "⚠️  コマンド実行確認"
	message := fmt.Sprintf("以下のコマンドを実行しますか？\n\n%s", command)
	options := []string{"✅ 実行", "❌ キャンセル"}

	model := NewConfirmationDialog(title, message, options)
	model.Selected = 1 // デフォルトはキャンセル（安全性優先）
	return model
}

// 確認ダイアログを実行して結果を取得（Claude Code風シンプル版）
func RunConfirmationDialog(model ConfirmationModel) (bool, error) {
	// タイトルとメッセージ表示
	fmt.Printf("\n\033[33m%s\033[0m\n", model.Title)
	fmt.Printf("%s\n\n", model.Message)

	// オプション表示
	for i, option := range model.Options {
		if i == 0 {
			fmt.Printf("  \033[32m[y]\033[0m %s\n", option)
		} else {
			fmt.Printf("  \033[31m[n]\033[0m %s\n", option)
		}
	}

	fmt.Printf("\n入力してください (y/n): ")

	// 入力受付
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false, fmt.Errorf("入力読み取りエラー")
	}

	input := strings.ToLower(strings.TrimSpace(scanner.Text()))

	switch input {
	case "y", "yes", "はい", "実行":
		return true, nil
	case "n", "no", "いいえ", "キャンセル", "":
		return false, nil
	default:
		fmt.Printf("無効な入力です。デフォルトで「いいえ」を選択します。\n")
		return false, nil
	}
}
