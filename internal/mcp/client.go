package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// MCPクライアント実装
type Client struct {
	mu         sync.RWMutex
	conn       io.ReadWriteCloser
	serverProc *exec.Cmd
	session    *SessionState
	messageID  int64
	handlers   map[string]MessageHandler
	tools      map[string]Tool
	resources  map[string]Resource
	prompts    map[string]Prompt
	ctx        context.Context
	cancel     context.CancelFunc
	logger     Logger
}

// メッセージハンドラー関数型
type MessageHandler func(message Message) error

// ログインターフェース
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// MCPクライアントの設定
type ClientConfig struct {
	ServerCommand []string          `json:"serverCommand"`
	ServerArgs    []string          `json:"serverArgs"`
	Environment   map[string]string `json:"environment"`
	WorkingDir    string            `json:"workingDir"`
	Timeout       time.Duration     `json:"timeout"`
	Logger        Logger            `json:"-"`
}

// 新しいMCPクライアントを作成
func NewClient(config ClientConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		session:   &SessionState{},
		handlers:  make(map[string]MessageHandler),
		tools:     make(map[string]Tool),
		resources: make(map[string]Resource),
		prompts:   make(map[string]Prompt),
		ctx:       ctx,
		cancel:    cancel,
		logger:    config.Logger,
	}
}

// MCPサーバーに接続
func (c *Client) Connect(config ClientConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// サーバープロセスを起動
	if err := c.startServer(config); err != nil {
		return fmt.Errorf("サーバー起動失敗: %w", err)
	}

	// 初期化シーケンスを実行
	if err := c.initialize(); err != nil {
		c.stopServer()
		return fmt.Errorf("初期化失敗: %w", err)
	}

	// メッセージ受信を開始
	go c.messageLoop()

	c.session.Connected = true
	c.session.LastPing = time.Now()

	if c.logger != nil {
		c.logger.Info("MCPサーバーに接続しました", "server", c.session.ServerInfo.Name)
	}

	return nil
}

// サーバープロセスを起動
func (c *Client) startServer(config ClientConfig) error {
	if len(config.ServerCommand) == 0 {
		return fmt.Errorf("サーバーコマンドが指定されていません")
	}

	cmd := exec.CommandContext(c.ctx, config.ServerCommand[0], config.ServerCommand[1:]...)
	cmd.Dir = config.WorkingDir

	// 環境変数を設定
	for key, value := range config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// stdin/stdoutパイプを設定
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return err
	}

	// サーバープロセスを開始
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return err
	}

	c.serverProc = cmd
	c.conn = &pipeConn{stdin: stdin, stdout: stdout}

	return nil
}

// パイプ接続の実装
type pipeConn struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (p *pipeConn) Read(data []byte) (int, error) {
	return p.stdout.Read(data)
}

func (p *pipeConn) Write(data []byte) (int, error) {
	return p.stdin.Write(data)
}

func (p *pipeConn) Close() error {
	p.stdin.Close()
	return p.stdout.Close()
}

// 初期化シーケンス
func (c *Client) initialize() error {
	initReq := InitializeRequest{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities: ClientCapability{
			Sampling: &SamplingCapability{},
			Roots:    &RootsCapability{ListChanged: true},
		},
		ClientInfo: ClientInfo{
			Name:    "vyb-code",
			Version: "1.0.0",
		},
	}

	response, err := c.sendRequest("initialize", initReq)
	if err != nil {
		return err
	}

	// サーバー情報を解析
	var serverInfo ServerInfo
	if err := json.Unmarshal(response.Result.([]byte), &serverInfo); err != nil {
		return fmt.Errorf("サーバー情報解析失敗: %w", err)
	}

	c.session.ServerInfo = &serverInfo

	// ツール一覧を取得
	if err := c.refreshTools(); err != nil {
		return fmt.Errorf("ツール一覧取得失敗: %w", err)
	}

	// リソース一覧を取得
	if err := c.refreshResources(); err != nil {
		return fmt.Errorf("リソース一覧取得失敗: %w", err)
	}

	return nil
}

// ツール一覧を更新
func (c *Client) refreshTools() error {
	response, err := c.sendRequest("tools/list", nil)
	if err != nil {
		return err
	}

	var toolsResp struct {
		Tools []Tool `json:"tools"`
	}

	if err := json.Unmarshal(response.Result.([]byte), &toolsResp); err != nil {
		return err
	}

	// ツールマップを更新
	c.tools = make(map[string]Tool)
	for _, tool := range toolsResp.Tools {
		c.tools[tool.Name] = tool
	}

	c.session.Tools = toolsResp.Tools
	return nil
}

// リソース一覧を更新
func (c *Client) refreshResources() error {
	response, err := c.sendRequest("resources/list", nil)
	if err != nil {
		return err
	}

	var resourcesResp struct {
		Resources []Resource `json:"resources"`
	}

	if err := json.Unmarshal(response.Result.([]byte), &resourcesResp); err != nil {
		return err
	}

	// リソースマップを更新
	c.resources = make(map[string]Resource)
	for _, resource := range resourcesResp.Resources {
		c.resources[resource.URI] = resource
	}

	c.session.Resources = resourcesResp.Resources
	return nil
}

// ツールを実行
func (c *Client) CallTool(name string, arguments map[string]interface{}) (*ToolResult, error) {
	c.mu.RLock()
	_, exists := c.tools[name]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("ツール '%s' が見つかりません", name)
	}

	// ツール呼び出し要求を作成
	callReq := struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}{
		Name:      name,
		Arguments: arguments,
	}

	response, err := c.sendRequest("tools/call", callReq)
	if err != nil {
		return nil, fmt.Errorf("ツール実行失敗: %w", err)
	}

	var result ToolResult
	if err := json.Unmarshal(response.Result.([]byte), &result); err != nil {
		return nil, fmt.Errorf("結果解析失敗: %w", err)
	}

	if c.logger != nil {
		c.logger.Info("ツール実行完了", "tool", name, "result_size", len(result.Content))
	}

	return &result, nil
}

// リソースを読み取り
func (c *Client) ReadResource(uri string) (*Content, error) {
	c.mu.RLock()
	_, exists := c.resources[uri]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("リソース '%s' が見つかりません", uri)
	}

	readReq := struct {
		URI string `json:"uri"`
	}{
		URI: uri,
	}

	response, err := c.sendRequest("resources/read", readReq)
	if err != nil {
		return nil, fmt.Errorf("リソース読み取り失敗: %w", err)
	}

	var readResp struct {
		Contents []Content `json:"contents"`
	}

	if err := json.Unmarshal(response.Result.([]byte), &readResp); err != nil {
		return nil, fmt.Errorf("レスポンス解析失敗: %w", err)
	}

	if len(readResp.Contents) == 0 {
		return nil, fmt.Errorf("リソースが空です")
	}

	if c.logger != nil {
		c.logger.Info("リソース読み取り完了", "uri", uri, "type", readResp.Contents[0].Type)
	}

	return &readResp.Contents[0], nil
}

// 要求メッセージを送信
func (c *Client) sendRequest(method string, params interface{}) (*Message, error) {
	c.messageID++

	msg := Message{
		JSONRPC: "2.0",
		ID:      c.messageID,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// メッセージを送信
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return nil, err
	}

	// 応答を待機（タイムアウト付き）
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	return c.waitForResponse(ctx, c.messageID)
}

// 応答メッセージを待機
func (c *Client) waitForResponse(ctx context.Context, id int64) (*Message, error) {
	// 簡易実装：実際の本格的な実装では非同期メッセージキューが必要
	decoder := json.NewDecoder(c.conn)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("応答タイムアウト")
		default:
			var msg Message
			if err := decoder.Decode(&msg); err != nil {
				return nil, err
			}

			if msg.ID == id {
				if msg.Error != nil {
					return nil, fmt.Errorf("MCPエラー: %s", msg.Error.Message)
				}
				return &msg, nil
			}

			// 他のメッセージ（通知など）を処理
			go c.handleMessage(msg)
		}
	}
}

// メッセージ受信ループ
func (c *Client) messageLoop() {
	decoder := json.NewDecoder(c.conn)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			var msg Message
			if err := decoder.Decode(&msg); err != nil {
				if c.logger != nil {
					c.logger.Error("メッセージ受信エラー", "error", err)
				}
				return
			}

			go c.handleMessage(msg)
		}
	}
}

// メッセージを処理
func (c *Client) handleMessage(msg Message) {
	if handler, exists := c.handlers[msg.Method]; exists {
		if err := handler(msg); err != nil && c.logger != nil {
			c.logger.Error("メッセージ処理エラー", "method", msg.Method, "error", err)
		}
	}
}

// メッセージハンドラーを登録
func (c *Client) RegisterHandler(method string, handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[method] = handler
}

// 利用可能なツール一覧を取得
func (c *Client) GetTools() []Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tools := make([]Tool, 0, len(c.tools))
	for _, tool := range c.tools {
		tools = append(tools, tool)
	}

	return tools
}

// 利用可能なリソース一覧を取得
func (c *Client) GetResources() []Resource {
	c.mu.RLock()
	defer c.mu.RUnlock()

	resources := make([]Resource, 0, len(c.resources))
	for _, resource := range c.resources {
		resources = append(resources, resource)
	}

	return resources
}

// セッション状態を取得
func (c *Client) GetSessionState() *SessionState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// セッション状態のコピーを返す
	state := *c.session
	return &state
}

// 接続を閉じる
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cancel()

	if c.conn != nil {
		c.conn.Close()
	}

	if c.serverProc != nil {
		c.stopServer()
	}

	c.session.Connected = false

	if c.logger != nil {
		c.logger.Info("MCPクライアントを閉じました")
	}

	return nil
}

// サーバープロセスを停止
func (c *Client) stopServer() {
	if c.serverProc == nil {
		return
	}

	// 優雅にシャットダウンを試行
	if c.serverProc.Process != nil {
		c.serverProc.Process.Signal(os.Interrupt)

		// 5秒待機
		done := make(chan error, 1)
		go func() {
			done <- c.serverProc.Wait()
		}()

		select {
		case <-time.After(5 * time.Second):
			// タイムアウト：強制終了
			c.serverProc.Process.Kill()
			c.serverProc.Wait()
		case <-done:
			// 正常終了
		}
	}

	c.serverProc = nil
}

// 接続状態をチェック
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.session.Connected
}

// ヘルスチェック
func (c *Client) Ping() error {
	if !c.IsConnected() {
		return fmt.Errorf("MCPサーバーに接続されていません")
	}

	_, err := c.sendRequest("ping", nil)
	if err != nil {
		c.session.Connected = false
		return err
	}

	c.session.LastPing = time.Now()
	return nil
}
