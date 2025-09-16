package core

import (
	"context"
	"testing"
)

// MockComponent はテスト用のモックコンポーネント
type MockComponent struct {
	name        string
	initialized bool
	initError   error
	healthError error
}

func (m *MockComponent) Name() string {
	return m.name
}

func (m *MockComponent) Initialize(ctx context.Context) error {
	if m.initError != nil {
		return m.initError
	}
	m.initialized = true
	return nil
}

func (m *MockComponent) Shutdown(ctx context.Context) error {
	m.initialized = false
	return nil
}

func (m *MockComponent) Health(ctx context.Context) error {
	if !m.initialized {
		return m.healthError
	}
	return nil
}

func TestComponentRegistry_RegisterCore(t *testing.T) {
	registry := NewComponentRegistry()

	component := &MockComponent{name: "test-core"}

	err := registry.RegisterCore(component)
	if err != nil {
		t.Fatalf("RegisterCore failed: %v", err)
	}

	// 同じ名前での重複登録はエラーになる
	err = registry.RegisterCore(component)
	if err == nil {
		t.Fatal("Expected error for duplicate registration")
	}
}

func TestComponentRegistry_GetComponent(t *testing.T) {
	registry := NewComponentRegistry()

	component := &MockComponent{name: "test-core"}
	registry.RegisterCore(component)

	retrieved, err := registry.GetComponent("test-core")
	if err != nil {
		t.Fatalf("GetComponent failed: %v", err)
	}

	if retrieved.Name() != "test-core" {
		t.Fatalf("Expected component name 'test-core', got '%s'", retrieved.Name())
	}

	// 存在しないコンポーネントの取得はエラーになる
	_, err = registry.GetComponent("non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent component")
	}
}

func TestComponentRegistry_InitializeAll(t *testing.T) {
	registry := NewComponentRegistry()

	component1 := &MockComponent{name: "core1"}
	component2 := &MockComponent{name: "core2"}

	registry.RegisterCore(component1)
	registry.RegisterCore(component2)

	err := registry.InitializeAll(context.Background())
	if err != nil {
		t.Fatalf("InitializeAll failed: %v", err)
	}

	if !component1.initialized {
		t.Fatal("Component1 not initialized")
	}
	if !component2.initialized {
		t.Fatal("Component2 not initialized")
	}
}

func TestModuleManager(t *testing.T) {
	manager := NewModuleManager()

	component := &MockComponent{name: "test-module"}

	err := manager.RegisterCore(component)
	if err != nil {
		t.Fatalf("RegisterCore failed: %v", err)
	}

	modules := manager.ListModules()
	if len(modules) != 1 {
		t.Fatalf("Expected 1 module, got %d", len(modules))
	}

	if modules[0].Name != "test-module" {
		t.Fatalf("Expected module name 'test-module', got '%s'", modules[0].Name)
	}

	if modules[0].Type != TypeCore {
		t.Fatalf("Expected module type TypeCore, got %v", modules[0].Type)
	}
}
