package core

import (
	"context"
)

// CoreComponent は全てのコアコンポーネントが実装する基本インターフェース
type CoreComponent interface {
	// Name はコンポーネント名を返す
	Name() string

	// Initialize はコンポーネントを初期化
	Initialize(ctx context.Context) error

	// Shutdown はコンポーネントをシャットダウン
	Shutdown(ctx context.Context) error

	// Health はコンポーネントの健全性をチェック
	Health(ctx context.Context) error
}

// Extension は拡張機能コンポーネントのインターフェース
type Extension interface {
	CoreComponent

	// Dependencies は依存するコアコンポーネント名を返す
	Dependencies() []string

	// IsEnabled は拡張機能が有効かチェック
	IsEnabled(ctx context.Context) bool

	// Priority は拡張機能の優先度を返す（低い数字が高優先）
	Priority() int
}

// Bridge は橋渡しコンポーネントのインターフェース
type Bridge interface {
	CoreComponent

	// ConnectsTo は接続する拡張機能名を返す
	ConnectsTo() []string

	// IsRequired は必須の橋渡しコンポーネントかどうか
	IsRequired() bool
}

// ComponentRegistry はコンポーネントの登録・管理を行う
type ComponentRegistry interface {
	// RegisterCore はコアコンポーネントを登録
	RegisterCore(component CoreComponent) error

	// RegisterExtension は拡張機能を登録
	RegisterExtension(extension Extension) error

	// RegisterBridge は橋渡しコンポーネントを登録
	RegisterBridge(bridge Bridge) error

	// GetComponent はコンポーネントを取得
	GetComponent(name string) (CoreComponent, error)

	// ListComponents は登録されているコンポーネント一覧を取得
	ListComponents() map[string]CoreComponent

	// InitializeAll は全コンポーネントを依存関係順に初期化
	InitializeAll(ctx context.Context) error

	// ShutdownAll は全コンポーネントをシャットダウン
	ShutdownAll(ctx context.Context) error
}

// ComponentType はコンポーネントの種別
type ComponentType int

const (
	TypeCore ComponentType = iota
	TypeExtension
	TypeBridge
)

// ComponentMetadata はコンポーネントのメタデータ
type ComponentMetadata struct {
	Name         string        `json:"name"`
	Type         ComponentType `json:"type"`
	Version      string        `json:"version"`
	Description  string        `json:"description"`
	Dependencies []string      `json:"dependencies"`
	Optional     bool          `json:"optional"`
	Enabled      bool          `json:"enabled"`
}

// LifecycleManager はコンポーネントのライフサイクル管理
type LifecycleManager interface {
	// Start は指定されたコンポーネントを開始
	Start(ctx context.Context, name string) error

	// Stop は指定されたコンポーネントを停止
	Stop(ctx context.Context, name string) error

	// Restart は指定されたコンポーネントを再起動
	Restart(ctx context.Context, name string) error

	// GetStatus はコンポーネントの状態を取得
	GetStatus(name string) ComponentStatus
}

// ComponentStatus はコンポーネントの状態
type ComponentStatus struct {
	Name      string `json:"name"`
	Running   bool   `json:"running"`
	Healthy   bool   `json:"healthy"`
	StartTime int64  `json:"start_time"`
	Error     string `json:"error,omitempty"`
}

// ModuleManager はモジュール管理の統合インターフェース
type ModuleManager interface {
	ComponentRegistry
	LifecycleManager

	// LoadModule は指定されたモジュールを読み込み
	LoadModule(ctx context.Context, name string) error

	// UnloadModule は指定されたモジュールをアンロード
	UnloadModule(ctx context.Context, name string) error

	// ReloadModule は指定されたモジュールを再読み込み
	ReloadModule(ctx context.Context, name string) error

	// ListModules は利用可能なモジュール一覧を取得
	ListModules() []ComponentMetadata
}