package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glkt/vyb-code/internal/security"
)

// EditTool - 構造化コード編集ツール（Claude Code相当）
type EditTool struct {
	constraints *security.Constraints
	workDir     string
	maxFileSize int64
}

func NewEditTool(constraints *security.Constraints, workDir string, maxFileSize int64) *EditTool {
	return &EditTool{
		constraints: constraints,
		workDir:     workDir,
		maxFileSize: maxFileSize,
	}
}

type EditRequest struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

func (e *EditTool) Edit(req EditRequest) (*ToolExecutionResult, error) {
	// ファイルパスの検証
	absPath, err := filepath.Abs(req.FilePath)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("パス解決エラー: %v", err),
			IsError: true,
			Tool:    "edit",
		}, err
	}

	// セキュリティ制約チェック（ワークスペース内かどうか）
	if !e.constraints.IsPathAllowed(absPath) {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("パスがワークスペース外です: %s", req.FilePath),
			IsError: true,
			Tool:    "edit",
		}, fmt.Errorf("path outside workspace: %s", req.FilePath)
	}

	// ファイルの存在確認
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("ファイルが存在しません: %s", req.FilePath),
			IsError: true,
			Tool:    "edit",
		}, err
	}

	// ファイル内容読み込み
	content, err := os.ReadFile(absPath)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("ファイル読み込みエラー: %v", err),
			IsError: true,
			Tool:    "edit",
		}, err
	}

	// サイズ制限チェック
	if int64(len(content)) > e.maxFileSize {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("ファイルサイズが制限を超えています: %d bytes", len(content)),
			IsError: true,
			Tool:    "edit",
		}, fmt.Errorf("file too large")
	}

	originalContent := string(content)

	// 文字列置換の実行
	var modifiedContent string
	var replacements int

	if req.ReplaceAll {
		modifiedContent = strings.ReplaceAll(originalContent, req.OldString, req.NewString)
		replacements = strings.Count(originalContent, req.OldString)
	} else {
		// 単一置換（一意性チェック付き）
		occurrences := strings.Count(originalContent, req.OldString)
		if occurrences == 0 {
			return &ToolExecutionResult{
				Content: fmt.Sprintf("指定された文字列が見つかりません: %s", req.OldString),
				IsError: true,
				Tool:    "edit",
			}, fmt.Errorf("string not found")
		}
		if occurrences > 1 {
			return &ToolExecutionResult{
				Content: fmt.Sprintf("指定された文字列が複数存在します（%d箇所）。replace_all=trueを使用するか、より具体的な文字列を指定してください", occurrences),
				IsError: true,
				Tool:    "edit",
			}, fmt.Errorf("ambiguous match")
		}

		modifiedContent = strings.Replace(originalContent, req.OldString, req.NewString, 1)
		replacements = 1
	}

	// 変更があるかチェック
	if modifiedContent == originalContent {
		return &ToolExecutionResult{
			Content: "ファイルに変更はありませんでした",
			IsError: false,
			Tool:    "edit",
			Metadata: map[string]interface{}{
				"file_path":    req.FilePath,
				"replacements": 0,
				"changed":      false,
			},
		}, nil
	}

	// ファイルに書き込み
	err = os.WriteFile(absPath, []byte(modifiedContent), 0644)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("ファイル書き込みエラー: %v", err),
			IsError: true,
			Tool:    "edit",
		}, err
	}

	return &ToolExecutionResult{
		Content: fmt.Sprintf("ファイルを正常に編集しました: %d箇所を置換", replacements),
		IsError: false,
		Tool:    "edit",
		Metadata: map[string]interface{}{
			"file_path":     req.FilePath,
			"replacements":  replacements,
			"changed":       true,
			"original_size": len(originalContent),
			"new_size":      len(modifiedContent),
		},
	}, nil
}

// MultiEditTool - 複数の編集操作を一つのファイルに対して実行
type MultiEditTool struct {
	editTool *EditTool
}

func NewMultiEditTool(constraints *security.Constraints, workDir string, maxFileSize int64) *MultiEditTool {
	return &MultiEditTool{
		editTool: NewEditTool(constraints, workDir, maxFileSize),
	}
}

type MultiEditRequest struct {
	FilePath string        `json:"file_path"`
	Edits    []EditRequest `json:"edits"`
}

func (me *MultiEditTool) MultiEdit(req MultiEditRequest) (*ToolExecutionResult, error) {
	// パス検証
	absPath, err := filepath.Abs(req.FilePath)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("パス解決エラー: %v", err),
			IsError: true,
			Tool:    "multiedit",
		}, err
	}

	// セキュリティ制約チェック
	if !me.editTool.constraints.IsPathAllowed(absPath) {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("パスがワークスペース外です: %s", req.FilePath),
			IsError: true,
			Tool:    "multiedit",
		}, fmt.Errorf("path outside workspace")
	}

	// ファイルの存在確認
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// ファイルが存在しない場合は新規作成
		if len(req.Edits) == 0 {
			return &ToolExecutionResult{
				Content: "編集操作が指定されていません",
				IsError: true,
				Tool:    "multiedit",
			}, fmt.Errorf("no edits specified")
		}

		// 最初の編集で空文字列から開始する場合のみ新規ファイル作成を許可
		firstEdit := req.Edits[0]
		if firstEdit.OldString != "" {
			return &ToolExecutionResult{
				Content: fmt.Sprintf("ファイルが存在しません: %s", req.FilePath),
				IsError: true,
				Tool:    "multiedit",
			}, fmt.Errorf("file not found")
		}

		// 新規ファイルを作成
		err = os.WriteFile(absPath, []byte(firstEdit.NewString), 0644)
		if err != nil {
			return &ToolExecutionResult{
				Content: fmt.Sprintf("新規ファイル作成エラー: %v", err),
				IsError: true,
				Tool:    "multiedit",
			}, err
		}

		// 残りの編集を実行
		req.Edits = req.Edits[1:]
	}

	// 現在のファイル内容を読み込み
	content, err := os.ReadFile(absPath)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("ファイル読み込みエラー: %v", err),
			IsError: true,
			Tool:    "multiedit",
		}, err
	}

	currentContent := string(content)
	totalReplacements := 0
	successfulEdits := 0

	// 各編集を順次実行
	for i, editReq := range req.Edits {
		editReq.FilePath = req.FilePath // ファイルパスを設定

		// 文字列置換
		var modifiedContent string
		var replacements int

		if editReq.ReplaceAll {
			modifiedContent = strings.ReplaceAll(currentContent, editReq.OldString, editReq.NewString)
			replacements = strings.Count(currentContent, editReq.OldString)
		} else {
			occurrences := strings.Count(currentContent, editReq.OldString)
			if occurrences == 0 {
				return &ToolExecutionResult{
					Content: fmt.Sprintf("編集 %d: 指定された文字列が見つかりません: %s", i+1, editReq.OldString),
					IsError: true,
					Tool:    "multiedit",
				}, fmt.Errorf("string not found in edit %d", i+1)
			}
			if occurrences > 1 {
				return &ToolExecutionResult{
					Content: fmt.Sprintf("編集 %d: 指定された文字列が複数存在します（%d箇所）", i+1, occurrences),
					IsError: true,
					Tool:    "multiedit",
				}, fmt.Errorf("ambiguous match in edit %d", i+1)
			}

			modifiedContent = strings.Replace(currentContent, editReq.OldString, editReq.NewString, 1)
			replacements = 1
		}

		// 現在のコンテンツを更新
		if modifiedContent != currentContent {
			currentContent = modifiedContent
			totalReplacements += replacements
			successfulEdits++
		}
	}

	// 変更があれば書き込み
	if totalReplacements > 0 {
		err = os.WriteFile(absPath, []byte(currentContent), 0644)
		if err != nil {
			return &ToolExecutionResult{
				Content: fmt.Sprintf("ファイル書き込みエラー: %v", err),
				IsError: true,
				Tool:    "multiedit",
			}, err
		}
	}

	return &ToolExecutionResult{
		Content: fmt.Sprintf("マルチ編集完了: %d個の編集を実行、%d箇所を置換", successfulEdits, totalReplacements),
		IsError: false,
		Tool:    "multiedit",
		Metadata: map[string]interface{}{
			"file_path":          req.FilePath,
			"total_edits":        len(req.Edits),
			"successful_edits":   successfulEdits,
			"total_replacements": totalReplacements,
			"changed":            totalReplacements > 0,
		},
	}, nil
}

// ReadTool - 拡張ファイル読み取りツール
type ReadTool struct {
	constraints *security.Constraints
	workDir     string
	maxFileSize int64
}

func NewReadTool(constraints *security.Constraints, workDir string, maxFileSize int64) *ReadTool {
	return &ReadTool{
		constraints: constraints,
		workDir:     workDir,
		maxFileSize: maxFileSize,
	}
}

type ReadRequest struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset,omitempty"` // 読み取り開始行（1から開始）
	Limit    int    `json:"limit,omitempty"`  // 読み取る行数
}

func (r *ReadTool) Read(req ReadRequest) (*ToolExecutionResult, error) {
	// パス検証
	absPath, err := filepath.Abs(req.FilePath)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("パス解決エラー: %v", err),
			IsError: true,
			Tool:    "read",
		}, err
	}

	// セキュリティ制約チェック
	if !r.constraints.IsPathAllowed(absPath) {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("パスがワークスペース外です: %s", req.FilePath),
			IsError: true,
			Tool:    "read",
		}, fmt.Errorf("path outside workspace")
	}

	// ファイル存在確認
	fileInfo, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("ファイルが存在しません: %s", req.FilePath),
			IsError: true,
			Tool:    "read",
		}, err
	}

	// サイズ制限チェック
	if fileInfo.Size() > r.maxFileSize {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("ファイルサイズが制限を超えています: %d bytes", fileInfo.Size()),
			IsError: true,
			Tool:    "read",
		}, fmt.Errorf("file too large")
	}

	// ファイル読み取り
	file, err := os.Open(absPath)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("ファイル読み取りエラー: %v", err),
			IsError: true,
			Tool:    "read",
		}, err
	}
	defer file.Close()

	var content strings.Builder
	scanner := bufio.NewScanner(file)
	lineNum := 1
	totalLines := 0

	// オフセット処理
	for scanner.Scan() {
		totalLines++

		// オフセット以前の行をスキップ
		if req.Offset > 0 && lineNum < req.Offset {
			lineNum++
			continue
		}

		// 制限行数チェック
		if req.Limit > 0 && lineNum >= req.Offset+req.Limit {
			break
		}

		// 行番号付きで追加（Claude Codeスタイル）
		content.WriteString(fmt.Sprintf("%5d→%s\n", lineNum, scanner.Text()))
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("ファイル読み取りエラー: %v", err),
			IsError: true,
			Tool:    "read",
		}, err
	}

	return &ToolExecutionResult{
		Content: content.String(),
		IsError: false,
		Tool:    "read",
		Metadata: map[string]interface{}{
			"file_path":   req.FilePath,
			"total_lines": totalLines,
			"lines_read":  lineNum - max(1, req.Offset),
			"file_size":   fileInfo.Size(),
			"offset":      req.Offset,
			"limit":       req.Limit,
		},
	}, nil
}

// WriteTool - 構造化ファイル書き込みツール
type WriteTool struct {
	constraints *security.Constraints
	workDir     string
	maxFileSize int64
}

func NewWriteTool(constraints *security.Constraints, workDir string, maxFileSize int64) *WriteTool {
	return &WriteTool{
		constraints: constraints,
		workDir:     workDir,
		maxFileSize: maxFileSize,
	}
}

type WriteRequest struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func (w *WriteTool) Write(req WriteRequest) (*ToolExecutionResult, error) {
	// パス検証
	absPath, err := filepath.Abs(req.FilePath)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("パス解決エラー: %v", err),
			IsError: true,
			Tool:    "write",
		}, err
	}

	// セキュリティ制約チェック
	if !w.constraints.IsPathAllowed(absPath) {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("パスがワークスペース外です: %s", req.FilePath),
			IsError: true,
			Tool:    "write",
		}, fmt.Errorf("path outside workspace")
	}

	// サイズ制限チェック
	if int64(len(req.Content)) > w.maxFileSize {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("コンテンツサイズが制限を超えています: %d bytes", len(req.Content)),
			IsError: true,
			Tool:    "write",
		}, fmt.Errorf("content too large")
	}

	// 既存ファイルの存在チェック（上書き警告のため）
	overwriting := false
	if _, err := os.Stat(absPath); err == nil {
		overwriting = true
	}

	// ディレクトリ作成（必要に応じて）
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("ディレクトリ作成エラー: %v", err),
			IsError: true,
			Tool:    "write",
		}, err
	}

	// ファイル書き込み
	err = os.WriteFile(absPath, []byte(req.Content), 0644)
	if err != nil {
		return &ToolExecutionResult{
			Content: fmt.Sprintf("ファイル書き込みエラー: %v", err),
			IsError: true,
			Tool:    "write",
		}, err
	}

	action := "作成"
	if overwriting {
		action = "上書き"
	}

	return &ToolExecutionResult{
		Content: fmt.Sprintf("ファイルを正常に%sしました: %s (%d bytes)", action, req.FilePath, len(req.Content)),
		IsError: false,
		Tool:    "write",
		Metadata: map[string]interface{}{
			"file_path":   req.FilePath,
			"size":        len(req.Content),
			"overwriting": overwriting,
			"action":      action,
		},
	}, nil
}

