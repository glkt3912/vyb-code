package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// ビルド時に注入される変数（ldflags経由）
var (
	// Version はアプリケーションのバージョン
	Version = "dev"

	// BuildTime はビルド時刻
	BuildTime = "unknown"

	// GitCommit はGitコミットハッシュ
	GitCommit = "unknown"

	// GitBranch はGitブランチ名
	GitBranch = "unknown"
)

// Info はバージョン情報を保持する構造体
type Info struct {
	Version     string `json:"version"`
	BuildTime   string `json:"build_time"`
	GitCommit   string `json:"git_commit"`
	GitBranch   string `json:"git_branch"`
	GoVersion   string `json:"go_version"`
	Compiler    string `json:"compiler"`
	Platform    string `json:"platform"`
	VCSRevision string `json:"vcs_revision,omitempty"`
	VCSTime     string `json:"vcs_time,omitempty"`
	VCSModified bool   `json:"vcs_modified,omitempty"`
}

// Get はアプリケーションのバージョン情報を取得
func Get() *Info {
	info := &Info{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
	}

	// ランタイム情報を取得
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		info.GoVersion = buildInfo.GoVersion

		// ビルド設定から詳細情報を取得
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "-compiler":
				info.Compiler = setting.Value
			case "GOOS":
				if info.Platform == "" {
					info.Platform = setting.Value
				} else {
					info.Platform = fmt.Sprintf("%s/%s", info.Platform, setting.Value)
				}
			case "GOARCH":
				if strings.Contains(info.Platform, "/") {
					info.Platform = fmt.Sprintf("%s/%s", strings.Split(info.Platform, "/")[0], setting.Value)
				} else {
					info.Platform = fmt.Sprintf("%s/%s", info.Platform, setting.Value)
				}
			case "vcs.revision":
				if info.GitCommit == "unknown" {
					info.VCSRevision = setting.Value
				}
			case "vcs.time":
				info.VCSTime = setting.Value
			case "vcs.modified":
				info.VCSModified = setting.Value == "true"
			}
		}
	}

	return info
}

// String はバージョン情報の文字列表現を返す
func (i *Info) String() string {
	return fmt.Sprintf("vyb version %s", i.Version)
}

// Detailed は詳細なバージョン情報の文字列表現を返す
func (i *Info) Detailed() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Version: %s", i.Version))

	if i.BuildTime != "unknown" {
		parts = append(parts, fmt.Sprintf("Build Time: %s", i.BuildTime))
	}

	if i.GitCommit != "unknown" {
		parts = append(parts, fmt.Sprintf("Git Commit: %s", i.GitCommit))
	} else if i.VCSRevision != "" {
		parts = append(parts, fmt.Sprintf("VCS Revision: %s", i.VCSRevision))
	}

	if i.GitBranch != "unknown" {
		parts = append(parts, fmt.Sprintf("Git Branch: %s", i.GitBranch))
	}

	if i.VCSTime != "" {
		parts = append(parts, fmt.Sprintf("VCS Time: %s", i.VCSTime))
	}

	if i.VCSModified {
		parts = append(parts, "VCS Modified: true")
	}

	parts = append(parts, fmt.Sprintf("Go Version: %s", i.GoVersion))

	if i.Compiler != "" {
		parts = append(parts, fmt.Sprintf("Compiler: %s", i.Compiler))
	}

	if i.Platform != "" && i.Platform != "/" {
		parts = append(parts, fmt.Sprintf("Platform: %s", i.Platform))
	}

	return strings.Join(parts, "\n")
}

// GetVersion はシンプルなバージョン文字列を返す
func GetVersion() string {
	return Version
}

// GetVersionWithPrefix はv接頭辞付きのバージョン文字列を返す
func GetVersionWithPrefix() string {
	if strings.HasPrefix(Version, "v") {
		return Version
	}
	return "v" + Version
}

// GetMCPVersion はMCPクライアント用のバージョン（v接頭辞なし）を返す
func GetMCPVersion() string {
	return strings.TrimPrefix(Version, "v")
}
