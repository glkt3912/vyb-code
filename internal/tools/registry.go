package tools

import (
	"fmt"
	"strings"
	"sync"

	"github.com/glkt/vyb-code/internal/mcp"
	"github.com/glkt/vyb-code/internal/security"
)

// 統一ツールインターフェース
type UnifiedTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`       // "native" or "mcp"
	ServerName  string                 `json:"serverName"` // MCPツールの場合のサーバー名
	Schema      map[string]interface{} `json:"schema"`     // ツールスキーマ
	Handler     ToolHandler            `json:"-"`          // ネイティブツールハンドラー
}

// ツールハンドラー関数型
type ToolHandler func(arguments map[string]interface{}) (interface{}, error)

// ツール実行結果
type ToolExecutionResult struct {
	Content  string                 `json:"content"`
	IsError  bool                   `json:"isError"`
	Metadata map[string]interface{} `json:"metadata"`
	Tool     string                 `json:"tool"`
	Server   string                 `json:"server,omitempty"`
	Duration string                 `json:"duration"`
	ExitCode int                    `json:"exitCode,omitempty"`
}

// ツールレジストリ：ネイティブとMCPツールを統合管理
type ToolRegistry struct {
	mu          sync.RWMutex
	nativeTools map[string]UnifiedTool
	mcpManager  *mcp.Manager
	constraints *security.Constraints
	fileOps     *FileOperations
	executor    *CommandExecutor
	gitOps      *GitOperations
	// Claude Codeツール
	bashTool      *BashTool
	globTool      *GlobTool
	grepTool      *GrepTool
	lsTool        *LSTool
	webFetchTool  *WebFetchTool
	webSearchTool *WebSearchTool
	editTool      *EditTool
	multiEditTool *MultiEditTool
	readTool      *ReadTool
	writeTool     *WriteTool
	// Phase 2 高度機能ツール
	advancedAnalyzer *AdvancedProjectAnalyzer
	buildManager     *BuildManager
}

// 新しいツールレジストリを作成
func NewToolRegistry(constraints *security.Constraints, workDir string, maxFileSize int64, mcpManager *mcp.Manager) *ToolRegistry {
	registry := &ToolRegistry{
		nativeTools: make(map[string]UnifiedTool),
		mcpManager:  mcpManager,
		constraints: constraints,
		fileOps:     NewFileOperations(maxFileSize, workDir),
		executor:    NewCommandExecutor(constraints, workDir),
		gitOps:      NewGitOperations(constraints, workDir),
		// Claude Codeツールを初期化
		bashTool:      NewBashTool(constraints, workDir),
		globTool:      NewGlobTool(workDir),
		grepTool:      NewGrepTool(workDir),
		lsTool:        NewLSTool(workDir),
		webFetchTool:  NewWebFetchTool(),
		webSearchTool: NewWebSearchTool(),
		editTool:      NewEditTool(constraints, workDir, maxFileSize),
		multiEditTool: NewMultiEditTool(constraints, workDir, maxFileSize),
		readTool:      NewReadTool(constraints, workDir, maxFileSize),
		writeTool:     NewWriteTool(constraints, workDir, maxFileSize),
		// Phase 2 高度機能ツールを初期化
		advancedAnalyzer: NewAdvancedProjectAnalyzer(constraints, workDir),
		buildManager:     NewBuildManager(constraints, workDir),
	}

	// ネイティブツールを登録
	registry.registerNativeTools()
	// Claude Codeツールを登録
	registry.registerClaudeTools()
	return registry
}

// ネイティブツールを登録
func (r *ToolRegistry) registerNativeTools() {
	// ファイル読み取りツール
	r.nativeTools["read_file"] = UnifiedTool{
		Name:        "read_file",
		Description: "ファイルの内容を読み取ります",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "読み取るファイルのパス",
				},
			},
			"required": []string{"file_path"},
		},
		Handler: r.handleReadFile,
	}

	// ファイル書き込みツール
	r.nativeTools["write_file"] = UnifiedTool{
		Name:        "write_file",
		Description: "ファイルに内容を書き込みます",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "書き込むファイルのパス",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "書き込む内容",
				},
			},
			"required": []string{"file_path", "content"},
		},
		Handler: r.handleWriteFile,
	}

	// コマンド実行ツール
	r.nativeTools["execute_command"] = UnifiedTool{
		Name:        "execute_command",
		Description: "シェルコマンドを安全に実行します",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "実行するコマンド",
				},
			},
			"required": []string{"command"},
		},
		Handler: r.handleExecuteCommand,
	}

	// Git操作ツール
	r.nativeTools["git_status"] = UnifiedTool{
		Name:        "git_status",
		Description: "Git状態を取得します",
		Type:        "native",
		Schema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: r.handleGitStatus,
	}
}

// Claude Codeツールを登録
func (r *ToolRegistry) registerClaudeTools() {
	// Bashツール
	r.nativeTools["bash"] = UnifiedTool{
		Name:        "bash",
		Description: "セキュアなシェルコマンド実行",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "実行するコマンド",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "コマンドの説明",
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"description": "タイムアウト（ミリ秒）",
				},
			},
			"required": []string{"command"},
		},
		Handler: r.handleBashTool,
	}

	// Globツール
	r.nativeTools["glob"] = UnifiedTool{
		Name:        "glob",
		Description: "ファイルパターンマッチング",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "検索パターン",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "検索パス（省略可）",
				},
			},
			"required": []string{"pattern"},
		},
		Handler: r.handleGlobTool,
	}

	// Grepツール
	r.nativeTools["grep"] = UnifiedTool{
		Name:        "grep",
		Description: "高度なファイル検索",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "検索パターン（正規表現）",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "検索パス",
				},
				"glob": map[string]interface{}{
					"type":        "string",
					"description": "ファイルフィルター",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "ファイル種別",
				},
				"output_mode": map[string]interface{}{
					"type":        "string",
					"description": "出力モード: content, files_with_matches, count",
				},
				"case_insensitive": map[string]interface{}{
					"type":        "boolean",
					"description": "大文字小文字を無視",
				},
			},
			"required": []string{"pattern"},
		},
		Handler: r.handleGrepTool,
	}

	// LSツール
	r.nativeTools["ls"] = UnifiedTool{
		Name:        "ls",
		Description: "ディレクトリリスト表示",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "リストするパス",
				},
				"ignore": map[string]interface{}{
					"type":        "array",
					"description": "無視するパターン",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
		Handler: r.handleLSTool,
	}

	// WebFetchツール
	r.nativeTools["webfetch"] = UnifiedTool{
		Name:        "webfetch",
		Description: "Web内容の取得と処理",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "取得するURL",
				},
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "処理プロンプト",
				},
			},
			"required": []string{"url", "prompt"},
		},
		Handler: r.handleWebFetchTool,
	}

	// WebSearchツール
	r.nativeTools["websearch"] = UnifiedTool{
		Name:        "websearch",
		Description: "Web検索機能",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "検索クエリ",
				},
				"allowed_domains": map[string]interface{}{
					"type":        "array",
					"description": "許可するドメイン",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"blocked_domains": map[string]interface{}{
					"type":        "array",
					"description": "ブロックするドメイン",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"max_results": map[string]interface{}{
					"type":        "number",
					"description": "最大結果数",
				},
			},
			"required": []string{"query"},
		},
		Handler: r.handleWebSearchTool,
	}

	// Editツール
	r.nativeTools["edit"] = UnifiedTool{
		Name:        "edit",
		Description: "ファイルの構造化編集",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "編集するファイルパス",
				},
				"old_string": map[string]interface{}{
					"type":        "string",
					"description": "置換対象の文字列",
				},
				"new_string": map[string]interface{}{
					"type":        "string",
					"description": "置換後の文字列",
				},
				"replace_all": map[string]interface{}{
					"type":        "boolean",
					"description": "全て置換するか",
				},
			},
			"required": []string{"file_path", "old_string", "new_string"},
		},
		Handler: r.handleEditTool,
	}

	// MultiEditツール
	r.nativeTools["multiedit"] = UnifiedTool{
		Name:        "multiedit",
		Description: "ファイルの複数箇所編集",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "編集するファイルパス",
				},
				"edits": map[string]interface{}{
					"type":        "array",
					"description": "編集操作の配列",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"old_string": map[string]interface{}{
								"type":        "string",
								"description": "置換対象の文字列",
							},
							"new_string": map[string]interface{}{
								"type":        "string",
								"description": "置換後の文字列",
							},
							"replace_all": map[string]interface{}{
								"type":        "boolean",
								"description": "全て置換するか",
							},
						},
						"required": []string{"old_string", "new_string"},
					},
				},
			},
			"required": []string{"file_path", "edits"},
		},
		Handler: r.handleMultiEditTool,
	}

	// Readツール（拡張版）
	r.nativeTools["read"] = UnifiedTool{
		Name:        "read",
		Description: "ファイル内容の読み取り",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "読み取るファイルパス",
				},
				"offset": map[string]interface{}{
					"type":        "number",
					"description": "読み取り開始行",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "読み取る行数",
				},
			},
			"required": []string{"file_path"},
		},
		Handler: r.handleReadTool,
	}

	// Writeツール
	r.nativeTools["write"] = UnifiedTool{
		Name:        "write",
		Description: "ファイルへの書き込み",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "書き込むファイルパス",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "書き込む内容",
				},
			},
			"required": []string{"file_path", "content"},
		},
		Handler: r.handleWriteTool,
	}

	// Phase 2: 高度プロジェクト解析ツール
	r.nativeTools["project_analyze"] = UnifiedTool{
		Name:        "project_analyze",
		Description: "プロジェクトの高度な解析（アーキテクチャ、依存関係、セキュリティ）",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"analysis_type": map[string]interface{}{
					"type":        "string",
					"description": "解析タイプ（basic, advanced, security, architecture）",
					"default":     "advanced",
				},
			},
		},
		Handler: r.handleProjectAnalyzeTool,
	}

	// Phase 2: ビルドツール
	r.nativeTools["build"] = UnifiedTool{
		Name:        "build",
		Description: "プロジェクトの自動ビルドとパイプライン実行",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"system": map[string]interface{}{
					"type":        "string",
					"description": "ビルドシステム（auto, makefile, docker, go, javascript）",
					"default":     "auto",
				},
				"target": map[string]interface{}{
					"type":        "string",
					"description": "ビルドターゲット（build, test, clean）",
					"default":     "build",
				},
				"pipeline": map[string]interface{}{
					"type":        "string",
					"description": "パイプライン名（go_standard, javascript_standard, full_ci）",
				},
			},
		},
		Handler: r.handleBuildTool,
	}

	// Phase 2: アーキテクチャマッピングツール
	r.nativeTools["architecture_map"] = UnifiedTool{
		Name:        "architecture_map",
		Description: "プロジェクト構造とモジュール依存関係の可視化",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"format": map[string]interface{}{
					"type":        "string",
					"description": "出力形式（json, summary, detailed）",
					"default":     "summary",
				},
				"include_dependencies": map[string]interface{}{
					"type":        "boolean",
					"description": "モジュール間依存関係を含める",
					"default":     true,
				},
			},
		},
		Handler: r.handleArchitectureMapTool,
	}

	// Phase 2: 依存関係分析ツール
	r.nativeTools["dependency_scan"] = UnifiedTool{
		Name:        "dependency_scan",
		Description: "依存関係の詳細分析（脆弱性、ライセンス、更新可能性）",
		Type:        "native",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"scan_type": map[string]interface{}{
					"type":        "string",
					"description": "スキャンタイプ（all, security, outdated, conflicts）",
					"default":     "all",
				},
				"include_transitive": map[string]interface{}{
					"type":        "boolean",
					"description": "推移的依存関係を含める",
					"default":     true,
				},
			},
		},
		Handler: r.handleDependencyScanTool,
	}
}

// Claude Codeツールハンドラー実装
func (r *ToolRegistry) handleBashTool(arguments map[string]interface{}) (interface{}, error) {
	command, ok := arguments["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command引数が必要です")
	}

	description := ""
	if desc, ok := arguments["description"].(string); ok {
		description = desc
	}

	timeout := 0
	if timeoutVal, ok := arguments["timeout"].(float64); ok {
		timeout = int(timeoutVal)
	}

	result, err := r.bashTool.Execute(command, description, timeout)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content":   result.Content,
		"exit_code": result.ExitCode,
		"duration":  result.Duration,
		"metadata":  result.Metadata,
	}, nil
}

func (r *ToolRegistry) handleGlobTool(arguments map[string]interface{}) (interface{}, error) {
	pattern, ok := arguments["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern引数が必要です")
	}

	path := ""
	if pathVal, ok := arguments["path"].(string); ok {
		path = pathVal
	}

	result, err := r.globTool.Find(pattern, path)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content":  result.Content,
		"metadata": result.Metadata,
	}, nil
}

func (r *ToolRegistry) handleGrepTool(arguments map[string]interface{}) (interface{}, error) {
	options := GrepOptions{}

	if pattern, ok := arguments["pattern"].(string); ok {
		options.Pattern = pattern
	} else {
		return nil, fmt.Errorf("pattern引数が必要です")
	}

	if path, ok := arguments["path"].(string); ok {
		options.Path = path
	}
	if glob, ok := arguments["glob"].(string); ok {
		options.Glob = glob
	}
	if fileType, ok := arguments["type"].(string); ok {
		options.Type = fileType
	}
	if outputMode, ok := arguments["output_mode"].(string); ok {
		options.OutputMode = outputMode
	}
	if caseInsensitive, ok := arguments["case_insensitive"].(bool); ok {
		options.CaseInsensitive = caseInsensitive
	}

	result, err := r.grepTool.Search(options)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content":  result.Content,
		"metadata": result.Metadata,
	}, nil
}

func (r *ToolRegistry) handleLSTool(arguments map[string]interface{}) (interface{}, error) {
	path := ""
	if pathVal, ok := arguments["path"].(string); ok {
		path = pathVal
	}

	var ignore []string
	if ignoreVal, ok := arguments["ignore"].([]interface{}); ok {
		for _, item := range ignoreVal {
			if str, ok := item.(string); ok {
				ignore = append(ignore, str)
			}
		}
	}

	result, err := r.lsTool.List(path, ignore)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content":  result.Content,
		"metadata": result.Metadata,
	}, nil
}

func (r *ToolRegistry) handleWebFetchTool(arguments map[string]interface{}) (interface{}, error) {
	url, ok := arguments["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url引数が必要です")
	}

	prompt, ok := arguments["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt引数が必要です")
	}

	result, err := r.webFetchTool.Fetch(url, prompt)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content":  result.Content,
		"metadata": result.Metadata,
	}, nil
}

func (r *ToolRegistry) handleWebSearchTool(arguments map[string]interface{}) (interface{}, error) {
	options := WebSearchOptions{}

	if query, ok := arguments["query"].(string); ok {
		options.Query = query
	} else {
		return nil, fmt.Errorf("query引数が必要です")
	}

	if allowedInterface, ok := arguments["allowed_domains"].([]interface{}); ok {
		for _, domain := range allowedInterface {
			if domainStr, ok := domain.(string); ok {
				options.AllowedDomains = append(options.AllowedDomains, domainStr)
			}
		}
	}

	if blockedInterface, ok := arguments["blocked_domains"].([]interface{}); ok {
		for _, domain := range blockedInterface {
			if domainStr, ok := domain.(string); ok {
				options.BlockedDomains = append(options.BlockedDomains, domainStr)
			}
		}
	}

	if maxResults, ok := arguments["max_results"].(float64); ok {
		options.MaxResults = int(maxResults)
	}

	result, err := r.webSearchTool.Search(options)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content":  result.Content,
		"metadata": result.Metadata,
	}, nil
}

func (r *ToolRegistry) handleEditTool(arguments map[string]interface{}) (interface{}, error) {
	req := EditRequest{}

	if filePath, ok := arguments["file_path"].(string); ok {
		req.FilePath = filePath
	} else {
		return nil, fmt.Errorf("file_path引数が必要です")
	}

	if oldString, ok := arguments["old_string"].(string); ok {
		req.OldString = oldString
	} else {
		return nil, fmt.Errorf("old_string引数が必要です")
	}

	if newString, ok := arguments["new_string"].(string); ok {
		req.NewString = newString
	} else {
		return nil, fmt.Errorf("new_string引数が必要です")
	}

	if replaceAll, ok := arguments["replace_all"].(bool); ok {
		req.ReplaceAll = replaceAll
	}

	result, err := r.editTool.Edit(req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content":  result.Content,
		"metadata": result.Metadata,
	}, nil
}

func (r *ToolRegistry) handleMultiEditTool(arguments map[string]interface{}) (interface{}, error) {
	req := MultiEditRequest{}

	if filePath, ok := arguments["file_path"].(string); ok {
		req.FilePath = filePath
	} else {
		return nil, fmt.Errorf("file_path引数が必要です")
	}

	if editsInterface, ok := arguments["edits"].([]interface{}); ok {
		for _, editInterface := range editsInterface {
			if editMap, ok := editInterface.(map[string]interface{}); ok {
				edit := EditRequest{}

				if oldString, ok := editMap["old_string"].(string); ok {
					edit.OldString = oldString
				} else {
					return nil, fmt.Errorf("各編集にold_string引数が必要です")
				}

				if newString, ok := editMap["new_string"].(string); ok {
					edit.NewString = newString
				} else {
					return nil, fmt.Errorf("各編集にnew_string引数が必要です")
				}

				if replaceAll, ok := editMap["replace_all"].(bool); ok {
					edit.ReplaceAll = replaceAll
				}

				req.Edits = append(req.Edits, edit)
			}
		}
	} else {
		return nil, fmt.Errorf("edits引数が必要です")
	}

	result, err := r.multiEditTool.MultiEdit(req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content":  result.Content,
		"metadata": result.Metadata,
	}, nil
}

func (r *ToolRegistry) handleReadTool(arguments map[string]interface{}) (interface{}, error) {
	req := ReadRequest{}

	if filePath, ok := arguments["file_path"].(string); ok {
		req.FilePath = filePath
	} else {
		return nil, fmt.Errorf("file_path引数が必要です")
	}

	if offset, ok := arguments["offset"].(float64); ok {
		req.Offset = int(offset)
	}

	if limit, ok := arguments["limit"].(float64); ok {
		req.Limit = int(limit)
	}

	result, err := r.readTool.Read(req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content":  result.Content,
		"metadata": result.Metadata,
	}, nil
}

func (r *ToolRegistry) handleWriteTool(arguments map[string]interface{}) (interface{}, error) {
	req := WriteRequest{}

	if filePath, ok := arguments["file_path"].(string); ok {
		req.FilePath = filePath
	} else {
		return nil, fmt.Errorf("file_path引数が必要です")
	}

	if content, ok := arguments["content"].(string); ok {
		req.Content = content
	} else {
		return nil, fmt.Errorf("content引数が必要です")
	}

	result, err := r.writeTool.Write(req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content":  result.Content,
		"metadata": result.Metadata,
	}, nil
}

// ネイティブツールハンドラー実装
func (r *ToolRegistry) handleReadFile(arguments map[string]interface{}) (interface{}, error) {
	filePath, ok := arguments["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path引数が必要です")
	}

	content, err := r.fileOps.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content":   content,
		"file_path": filePath,
	}, nil
}

func (r *ToolRegistry) handleWriteFile(arguments map[string]interface{}) (interface{}, error) {
	filePath, ok := arguments["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path引数が必要です")
	}

	content, ok := arguments["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content引数が必要です")
	}

	if err := r.fileOps.WriteFile(filePath, content); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":    "success",
		"file_path": filePath,
		"size":      len(content),
	}, nil
}

func (r *ToolRegistry) handleExecuteCommand(arguments map[string]interface{}) (interface{}, error) {
	command, ok := arguments["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command引数が必要です")
	}

	result, err := r.executor.Execute(command)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
		"exit_code": result.ExitCode,
		"duration":  result.Duration,
		"timed_out": result.TimedOut,
	}, nil
}

func (r *ToolRegistry) handleGitStatus(arguments map[string]interface{}) (interface{}, error) {
	result, err := r.gitOps.GetStatus()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
		"exit_code": result.ExitCode,
		"clean":     result.Stdout == "",
	}, nil
}

// 利用可能な全ツールを取得
func (r *ToolRegistry) GetAllTools() map[string]UnifiedTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]UnifiedTool)

	// ネイティブツールを追加
	for name, tool := range r.nativeTools {
		result[name] = tool
	}

	// MCPツールを追加
	if r.mcpManager != nil {
		mcpTools := r.mcpManager.GetAllTools()
		for serverName, tools := range mcpTools {
			for _, tool := range tools {
				unifiedTool := UnifiedTool{
					Name:        fmt.Sprintf("%s.%s", serverName, tool.Name),
					Description: tool.Description,
					Type:        "mcp",
					ServerName:  serverName,
					Schema:      tool.InputSchema.(map[string]interface{}),
				}
				result[unifiedTool.Name] = unifiedTool
			}
		}
	}

	return result
}

// ツールを実行
func (r *ToolRegistry) ExecuteTool(toolName string, arguments map[string]interface{}) (*ToolExecutionResult, error) {
	r.mu.RLock()
	tool, isNative := r.nativeTools[toolName]
	r.mu.RUnlock()

	// ネイティブツールの場合
	if isNative {
		result, err := tool.Handler(arguments)
		if err != nil {
			return &ToolExecutionResult{
				Content: fmt.Sprintf("ツール実行エラー: %v", err),
				IsError: true,
				Tool:    toolName,
			}, err
		}

		return &ToolExecutionResult{
			Content:  fmt.Sprintf("ツール実行成功: %v", result),
			IsError:  false,
			Tool:     toolName,
			Metadata: result.(map[string]interface{}),
		}, nil
	}

	// MCPツールの場合
	parts := strings.Split(toolName, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("MCPツール名は 'server.tool' 形式である必要があります")
	}

	serverName, actualToolName := parts[0], parts[1]
	mcpResult, err := r.mcpManager.CallTool(serverName, actualToolName, arguments)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("MCPツール実行エラー: %v", err),
			IsError: true,
			Tool:    toolName,
			Server:  serverName,
		}, err
	}

	content := ""
	if len(mcpResult.Content) > 0 {
		content = mcpResult.Content[0].Text
	}

	return &ToolExecutionResult{
		Content: content,
		IsError: mcpResult.IsError,
		Tool:    toolName,
		Server:  serverName,
	}, nil
}

// Phase 2 ツールハンドラー実装

// プロジェクト解析ツールハンドラー
func (r *ToolRegistry) handleProjectAnalyzeTool(arguments map[string]interface{}) (interface{}, error) {
	analysisType := "advanced"
	if aType, ok := arguments["analysis_type"].(string); ok {
		analysisType = aType
	}

	switch analysisType {
	case "basic":
		basicAnalyzer := NewProjectAnalyzer(r.constraints, r.fileOps.WorkDir)
		result, err := basicAnalyzer.AnalyzeProject()
		if err != nil {
			return nil, fmt.Errorf("基本分析エラー: %w", err)
		}
		return map[string]interface{}{
			"analysis_type": "basic",
			"result":        result,
		}, nil

	case "advanced", "security", "architecture":
		result, err := r.advancedAnalyzer.AnalyzeAdvanced()
		if err != nil {
			return nil, fmt.Errorf("高度分析エラー: %w", err)
		}

		// 特定の分析タイプの場合は結果をフィルタリング
		switch analysisType {
		case "security":
			return map[string]interface{}{
				"analysis_type": "security",
				"result": map[string]interface{}{
					"security_analysis": result.SecurityAnalysis,
					"health_score":      result.HealthScore,
				},
			}, nil
		case "architecture":
			return map[string]interface{}{
				"analysis_type": "architecture",
				"result": map[string]interface{}{
					"architecture":     result.Architecture,
					"build_systems":    result.BuildSystems,
					"basic_info":       result.BasicInfo,
				},
			}, nil
		default:
			return map[string]interface{}{
				"analysis_type": "advanced",
				"result":        result,
			}, nil
		}

	default:
		return nil, fmt.Errorf("不明な分析タイプ: %s", analysisType)
	}
}

// ビルドツールハンドラー
func (r *ToolRegistry) handleBuildTool(arguments map[string]interface{}) (interface{}, error) {
	system := "auto"
	if s, ok := arguments["system"].(string); ok {
		system = s
	}

	target := "build"
	if t, ok := arguments["target"].(string); ok {
		target = t
	}

	// パイプライン実行の場合
	if pipelineName, ok := arguments["pipeline"].(string); ok {
		pipeline, err := r.buildManager.CreatePresetPipeline(pipelineName)
		if err != nil {
			return nil, fmt.Errorf("パイプライン作成エラー: %w", err)
		}

		results, err := r.buildManager.ExecutePipeline(pipeline)
		if err != nil {
			return nil, fmt.Errorf("パイプライン実行エラー: %w", err)
		}

		return map[string]interface{}{
			"pipeline":      pipelineName,
			"results":       results,
			"performance":   r.buildManager.performance,
		}, nil
	}

	// 通常のビルド実行
	var result *BuildResult
	var err error

	if system == "auto" {
		result, err = r.buildManager.AutoBuild()
	} else {
		result, err = r.buildManager.BuildWithSystem(system, target)
	}

	if err != nil {
		return nil, fmt.Errorf("ビルドエラー: %w", err)
	}

	return map[string]interface{}{
		"build_result": result,
		"cache_stats":  r.buildManager.cache,
	}, nil
}

// アーキテクチャマッピングツールハンドラー
func (r *ToolRegistry) handleArchitectureMapTool(arguments map[string]interface{}) (interface{}, error) {
	format := "summary"
	if f, ok := arguments["format"].(string); ok {
		format = f
	}

	includeDependencies := true
	if inc, ok := arguments["include_dependencies"].(bool); ok {
		includeDependencies = inc
	}

	analysis, err := r.advancedAnalyzer.AnalyzeAdvanced()
	if err != nil {
		return nil, fmt.Errorf("アーキテクチャ分析エラー: %w", err)
	}

	if analysis.Architecture == nil {
		return nil, fmt.Errorf("アーキテクチャ情報が見つかりません")
	}

	switch format {
	case "json":
		return map[string]interface{}{
			"architecture": analysis.Architecture,
			"build_systems": analysis.BuildSystems,
		}, nil

	case "summary":
		summary := map[string]interface{}{
			"layers_count":        len(analysis.Architecture.Layers),
			"modules_count":       len(analysis.Architecture.Modules),
			"entry_points_count":  len(analysis.Architecture.EntryPoints),
			"build_systems_count": len(analysis.BuildSystems),
		}

		if len(analysis.Architecture.Layers) > 0 {
			layerNames := make([]string, len(analysis.Architecture.Layers))
			for i, layer := range analysis.Architecture.Layers {
				layerNames[i] = layer.Name
			}
			summary["layers"] = layerNames
		}

		if len(analysis.Architecture.EntryPoints) > 0 {
			summary["entry_points"] = analysis.Architecture.EntryPoints
		}

		if includeDependencies && len(analysis.Architecture.Dependencies) > 0 {
			summary["dependency_count"] = len(analysis.Architecture.Dependencies)
		}

		return summary, nil

	case "detailed":
		result := map[string]interface{}{
			"architecture": analysis.Architecture,
			"basic_info":   analysis.BasicInfo,
		}

		if includeDependencies {
			result["detailed_dependencies"] = analysis.DetailedDeps
		}

		return result, nil

	default:
		return nil, fmt.Errorf("不明な出力形式: %s", format)
	}
}

// 依存関係分析ツールハンドラー
func (r *ToolRegistry) handleDependencyScanTool(arguments map[string]interface{}) (interface{}, error) {
	scanType := "all"
	if st, ok := arguments["scan_type"].(string); ok {
		scanType = st
	}

	includeTransitive := true
	if inc, ok := arguments["include_transitive"].(bool); ok {
		includeTransitive = inc
	}

	analysis, err := r.advancedAnalyzer.AnalyzeAdvanced()
	if err != nil {
		return nil, fmt.Errorf("依存関係分析エラー: %w", err)
	}

	if analysis.DetailedDeps == nil {
		return nil, fmt.Errorf("詳細依存関係情報が見つかりません")
	}

	result := make(map[string]interface{})

	switch scanType {
	case "all":
		result["direct_dependencies"] = analysis.DetailedDeps.DirectDeps
		result["dev_dependencies"] = analysis.DetailedDeps.DevDeps
		if includeTransitive {
			result["transitive_dependencies"] = analysis.DetailedDeps.TransitiveDeps
		}
		result["vulnerabilities"] = analysis.DetailedDeps.Vulnerabilities
		result["conflicts"] = analysis.DetailedDeps.Conflicts
		result["outdated"] = analysis.DetailedDeps.Outdated

	case "security":
		result["vulnerabilities"] = analysis.DetailedDeps.Vulnerabilities
		if analysis.SecurityAnalysis != nil {
			result["security_analysis"] = analysis.SecurityAnalysis
		}

	case "outdated":
		result["outdated"] = analysis.DetailedDeps.Outdated
		result["direct_dependencies"] = analysis.DetailedDeps.DirectDeps

	case "conflicts":
		result["conflicts"] = analysis.DetailedDeps.Conflicts

	default:
		return nil, fmt.Errorf("不明なスキャンタイプ: %s", scanType)
	}

	// 統計情報を追加
	result["summary"] = map[string]interface{}{
		"direct_deps_count":     len(analysis.DetailedDeps.DirectDeps),
		"dev_deps_count":        len(analysis.DetailedDeps.DevDeps),
		"transitive_deps_count": len(analysis.DetailedDeps.TransitiveDeps),
		"vulnerabilities_count": len(analysis.DetailedDeps.Vulnerabilities),
		"conflicts_count":       len(analysis.DetailedDeps.Conflicts),
		"outdated_count":        len(analysis.DetailedDeps.Outdated),
	}

	return result, nil
}
