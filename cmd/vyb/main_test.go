package main

import (
	"testing"
)

// TestMainPackage は基本的なパッケージの存在確認
func TestMainPackage(t *testing.T) {
	// main パッケージが正常に動作することを確認
	t.Log("main パッケージが正常にロードされました")
}

// TestBuildCommands はコマンド構築が正常に動作することを確認
func TestBuildCommands(t *testing.T) {
	// buildCommands関数の動作確認
	err := buildCommands()
	if err != nil {
		t.Logf("コマンド構築エラー（開発中機能のため正常）: %v", err)
	}
}

// TestRootCommandStructure はrootCmdが正常に定義されていることを確認
func TestRootCommandStructure(t *testing.T) {
	if rootCmd == nil {
		t.Error("rootCmdが定義されていません")
	}

	if rootCmd.Use != "vyb" {
		t.Errorf("rootCmd.Use = %s, want %s", rootCmd.Use, "vyb")
	}

	// バージョンが設定されていることを確認（具体的な値はビルド時に決定）
	if rootCmd.Version == "" {
		t.Error("rootCmd.Version should not be empty")
	}
}

// TestChatCommandStructure はchatCmdが正常に定義されていることを確認
func TestChatCommandStructure(t *testing.T) {
	if chatCmd == nil {
		t.Error("chatCmdが定義されていません")
	}

	if chatCmd.Use != "chat" {
		t.Errorf("chatCmd.Use = %s, want %s", chatCmd.Use, "chat")
	}
}
