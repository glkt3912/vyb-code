package main

import (
	"fmt"
	"runtime"
)

// ビルド時に設定される変数（-ldflagsで注入）
var (
	Version   = "dev"     // バージョン情報
	BuildTime = "unknown" // ビルド時刻
	GitCommit = "unknown" // Gitコミットハッシュ
	GitBranch = "unknown" // Gitブランチ名
)

// バージョン情報を表示する構造体
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitCommit string `json:"git_commit"`
	GitBranch string `json:"git_branch"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// バージョン情報を取得
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// バージョン情報を文字列として返す
func GetVersionString() string {
	info := GetVersionInfo()
	return fmt.Sprintf("vyb-code %s (%s %s, built %s)",
		info.Version, info.GoVersion, info.Platform, info.BuildTime)
}

// 詳細なバージョン情報を返す
func GetDetailedVersionString() string {
	info := GetVersionInfo()
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
