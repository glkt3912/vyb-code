package main

import (
	"fmt"

	"github.com/glkt/vyb-code/internal/version"
)

// GetVersionInfo はバージョン情報を取得
func GetVersionInfo() *version.Info {
	return version.Get()
}

// GetVersionString はバージョン情報を文字列として返す
func GetVersionString() string {
	info := version.Get()
	return fmt.Sprintf("vyb-code %s (%s %s, built %s)",
		info.Version, info.GoVersion, info.Platform, info.BuildTime)
}

// GetDetailedVersionString は詳細なバージョン情報を返す
func GetDetailedVersionString() string {
	info := version.Get()
	return fmt.Sprintf(`vyb-code %s

Build Information:
  Version:     %s
  Build Time:  %s
  Git Commit:  %s
  Git Branch:  %s
  Go Version:  %s
  Platform:    %s`,
		info.Version,
		info.Version,
		info.BuildTime,
		info.GitCommit,
		info.GitBranch,
		info.GoVersion,
		info.Platform,
	)
}
