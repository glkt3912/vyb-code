package handlers

import (
	"context"
	"fmt"

	"github.com/glkt/vyb-code/internal/config"
	"github.com/glkt/vyb-code/internal/logger"
	"github.com/spf13/cobra"
)

// Handler はすべてのハンドラーが実装すべき基本インターフェース
// 変更容易性向上のため、各ハンドラーの共通契約を定義
type Handler interface {
	// Initialize はハンドラーを初期化
	Initialize(cfg *config.Config) error

	// GetMetadata はハンドラーのメタデータを返す
	GetMetadata() HandlerMetadata

	// Health はハンドラーの健全性をチェック
	Health(ctx context.Context) error
}

// CommandProvider はCobraコマンドを提供するハンドラー
type CommandProvider interface {
	Handler
	// CreateCommands はCobraコマンドを作成
	CreateCommands() []*cobra.Command
}

// SessionHandler はセッション管理機能を持つハンドラー
type SessionHandler interface {
	Handler
	// StartSession はセッションを開始
	StartSession(cfg *config.Config) error

	// StopSession はセッションを停止
	StopSession() error

	// GetSessionStatus はセッション状態を取得
	GetSessionStatus() SessionStatus
}

// HandlerMetadata はハンドラーのメタデータ
type HandlerMetadata struct {
	Name         string            `json:"name"`         // ハンドラー名
	Version      string            `json:"version"`      // バージョン
	Description  string            `json:"description"`  // 説明
	Capabilities []string          `json:"capabilities"` // 機能一覧
	Dependencies []string          `json:"dependencies"` // 依存関係
	Config       map[string]string `json:"config"`       // 設定情報
}

// SessionStatus はセッション状態
type SessionStatus struct {
	ID        string `json:"id"`         // セッションID
	Active    bool   `json:"active"`     // アクティブ状態
	StartTime int64  `json:"start_time"` // 開始時刻
	Requests  int    `json:"requests"`   // リクエスト数
}

// HandlerFactory はハンドラーの作成を管理
type HandlerFactory struct {
	log      logger.Logger
	config   *config.Config
	registry map[string]HandlerCreator // ハンドラー作成関数のレジストリ
}

// HandlerCreator はハンドラー作成関数の型
type HandlerCreator func(log logger.Logger, cfg *config.Config) Handler

// NewHandlerFactory はファクトリーの新しいインスタンスを作成
func NewHandlerFactory(log logger.Logger, cfg *config.Config) *HandlerFactory {
	return &HandlerFactory{
		log:      log,
		config:   cfg,
		registry: make(map[string]HandlerCreator),
	}
}

// RegisterHandler はハンドラー作成関数を登録
func (f *HandlerFactory) RegisterHandler(name string, creator HandlerCreator) {
	f.registry[name] = creator
}

// CreateHandler は指定されたハンドラーを作成
func (f *HandlerFactory) CreateHandler(name string) (Handler, error) {
	creator, exists := f.registry[name]
	if !exists {
		return nil, fmt.Errorf("handler '%s' not found", name)
	}

	handler := creator(f.log, f.config)
	if err := handler.Initialize(f.config); err != nil {
		return nil, fmt.Errorf("handler initialization failed: %w", err)
	}

	return handler, nil
}

// ListAvailableHandlers は利用可能なハンドラー一覧を返す
func (f *HandlerFactory) ListAvailableHandlers() []string {
	handlers := make([]string, 0, len(f.registry))
	for name := range f.registry {
		handlers = append(handlers, name)
	}
	return handlers
}

// GetHandlerMetadata は指定されたハンドラーのメタデータを取得
func (f *HandlerFactory) GetHandlerMetadata(name string) (HandlerMetadata, error) {
	handler, err := f.CreateHandler(name)
	if err != nil {
		return HandlerMetadata{}, err
	}
	return handler.GetMetadata(), nil
}
