package version

import (
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()
	
	if info == nil {
		t.Fatal("Get() should not return nil")
	}
	
	if info.Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestGetVersion(t *testing.T) {
	version := GetVersion()
	
	if version == "" {
		t.Error("GetVersion() should not return empty string")
	}
}

func TestGetVersionWithPrefix(t *testing.T) {
	version := GetVersionWithPrefix()
	
	if version == "" {
		t.Error("GetVersionWithPrefix() should not return empty string")
	}
	
	if !strings.HasPrefix(version, "v") {
		t.Errorf("GetVersionWithPrefix() should start with 'v', got: %s", version)
	}
}

func TestGetMCPVersion(t *testing.T) {
	mcpVersion := GetMCPVersion()
	
	if mcpVersion == "" {
		t.Error("GetMCPVersion() should not return empty string")
	}
	
	if strings.HasPrefix(mcpVersion, "v") {
		t.Errorf("GetMCPVersion() should not start with 'v', got: %s", mcpVersion)
	}
}

func TestInfoString(t *testing.T) {
	info := Get()
	str := info.String()
	
	if str == "" {
		t.Error("Info.String() should not be empty")
	}
	
	if !strings.Contains(str, "vyb version") {
		t.Errorf("Info.String() should contain 'vyb version', got: %s", str)
	}
}

func TestInfoDetailed(t *testing.T) {
	info := Get()
	detailed := info.Detailed()
	
	if detailed == "" {
		t.Error("Info.Detailed() should not be empty")
	}
	
	if !strings.Contains(detailed, "Version:") {
		t.Errorf("Info.Detailed() should contain 'Version:', got: %s", detailed)
	}
	
	if !strings.Contains(detailed, "Go Version:") {
		t.Errorf("Info.Detailed() should contain 'Go Version:', got: %s", detailed)
	}
}

func TestVersionConsistency(t *testing.T) {
	// バージョンの一貫性チェック
	info := Get()
	version := GetVersion()
	versionWithPrefix := GetVersionWithPrefix()
	mcpVersion := GetMCPVersion()
	
	if info.Version != version {
		t.Errorf("Info.Version (%s) should match GetVersion() (%s)", info.Version, version)
	}
	
	// v接頭辞の処理確認
	expectedWithPrefix := version
	if !strings.HasPrefix(version, "v") {
		expectedWithPrefix = "v" + version
	}
	
	if versionWithPrefix != expectedWithPrefix {
		t.Errorf("GetVersionWithPrefix() (%s) should be %s", versionWithPrefix, expectedWithPrefix)
	}
	
	expectedMCP := strings.TrimPrefix(version, "v")
	if mcpVersion != expectedMCP {
		t.Errorf("GetMCPVersion() (%s) should be %s", mcpVersion, expectedMCP)
	}
}