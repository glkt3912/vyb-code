# vyb-code Architecture Overview

## Table of Contents

- [Project Structure](#project-structure)
- [MCP Integration Architecture](#mcp-integration-architecture)
- [Security Model](#security-model)
- [Tool System](#tool-system)
- [Session Management](#session-management)
- [Performance Considerations](#performance-considerations)

## Project Structure

vyb-code follows a clean architecture pattern with clear separation of concerns:

```
vyb-code/
├── cmd/vyb/                     # CLI entry point and command handlers
├── internal/
│   ├── chat/                    # Interactive conversation sessions
│   ├── config/                  # Configuration management
│   ├── llm/                     # LLM provider integrations
│   ├── mcp/                     # Model Context Protocol implementation
│   │   ├── client.go           # MCP client with JSON-RPC 2.0
│   │   ├── manager.go          # Multi-server connection management
│   │   ├── security.go         # MCP-specific security validation
│   │   └── types.go            # MCP protocol type definitions
│   ├── security/               # Core security constraints
│   ├── session/                # Persistent conversation management
│   ├── tools/                  # Native tool implementations
│   │   └── registry.go         # Unified tool interface
│   ├── search/                 # Advanced search and grep
│   ├── stream/                 # Real-time response processing
│   └── performance/            # Metrics and optimization
└── docs/                       # Documentation and examples
```

## MCP Integration Architecture

### Protocol Implementation

vyb-code implements the Model Context Protocol (MCP) specification 2024-11-05:

```go
// Core message structure following JSON-RPC 2.0
type Message struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id,omitempty"`
    Method  string      `json:"method,omitempty"`
    Params  interface{} `json:"params,omitempty"`
    Result  interface{} `json:"result,omitempty"`
    Error   *MCPError   `json:"error,omitempty"`
}
```

### Multi-Server Management

The MCP Manager (`internal/mcp/manager.go`) provides:

- **Concurrent connections**: Multiple MCP servers simultaneously
- **Connection lifecycle**: Automatic reconnection and health monitoring
- **Tool aggregation**: Unified view of tools across all servers
- **Security enforcement**: Per-server security validation

### Server Configuration

MCP servers are configured in `~/.vyb/config.json`:

```json
{
  "mcp_servers": {
    "filesystem": {
      "name": "filesystem",
      "command": ["npx", "@modelcontextprotocol/server-filesystem"],
      "args": ["/workspace"],
      "enabled": true,
      "autoConnect": true
    }
  }
}
```

## Security Model

### Layered Security Approach

1. **Configuration Level**: Server enable/disable controls
2. **Connection Level**: Process isolation and communication sandboxing
3. **Tool Level**: Name-based whitelisting/blacklisting
4. **Argument Level**: Parameter validation and workspace restrictions

### MCP-Specific Security

```go
type ToolSecurityValidator struct {
    constraints *security.Constraints
    whitelist   map[string]bool // Allowed tool names
    blacklist   map[string]bool // Forbidden tool names
}
```

**Security Checks:**
- Path restriction to workspace directory
- Command validation through existing security constraints
- URL access prevention
- Dangerous tool pattern detection

### Safe Tool Patterns

- File operations: `read_file`, `write_file`, `list_files`
- Git operations: `git_status`, `git_log`, `git_diff`
- Code analysis: `analyze_code`, `format_code`, `lint_code`

### Blocked Tool Patterns

- System execution: `*exec*`, `*shell*`, `*system*`
- Network access: `*http*`, `*curl*`, `*wget*`
- File destruction: `*delete*`, `*remove*`, `*destroy*`
- Privilege escalation: `*admin*`, `*sudo*`, `*root*`

## Tool System

### Unified Tool Interface

The tool registry (`internal/tools/registry.go`) provides a unified interface:

```go
type UnifiedTool struct {
    Name        string                 // Unique tool identifier
    Description string                 // Human-readable description
    Type        string                 // "native" or "mcp"
    ServerName  string                 // MCP server name (if applicable)
    Schema      map[string]interface{} // JSON schema for validation
    Handler     ToolHandler            // Native tool handler function
}
```

### Native vs MCP Tools

**Native Tools** (`Type: "native"`):
- Built into vyb-code
- Direct Go function execution
- Maximum performance and security
- Examples: file operations, git commands, project analysis

**MCP Tools** (`Type: "mcp"`):
- External server communication
- JSON-RPC protocol overhead
- Extensible ecosystem
- Examples: database access, web APIs, specialized analysis tools

### Tool Execution Flow

1. **Tool Discovery**: Registry aggregates native + MCP tools
2. **Security Validation**: Multi-layer security checks
3. **Execution**: Native handler or MCP client call
4. **Result Processing**: Unified response format
5. **Session Tracking**: Usage logging in conversation history

## Session Management

### Conversation Persistence

Sessions are stored in `~/.vyb/sessions/`:

```go
type Session struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
    Model     string    `json:"model"`
    Provider  string    `json:"provider"`
    Turns     []Turn    `json:"turns"`
    Context   Context   `json:"context"`
}
```

### MCP Tool Tracking

Each conversation turn tracks MCP tool usage:

```go
type Turn struct {
    Tools    []string      `json:"tools,omitempty"`
    MCPTools []MCPToolCall `json:"mcpTools,omitempty"`
}

type MCPToolCall struct {
    Server    string                 `json:"server"`
    Tool      string                 `json:"tool"`
    Arguments map[string]interface{} `json:"arguments"`
    Result    string                 `json:"result"`
    Success   bool                   `json:"success"`
    Duration  string                 `json:"duration"`
    Timestamp time.Time              `json:"timestamp"`
}
```

### Session Features

- **Export/Import**: JSON, Markdown, and plain text formats
- **Context Compression**: Automatic summarization of old turns
- **Search**: Full-text search across conversation history
- **Statistics**: Usage metrics and model performance tracking

## Performance Considerations

### Connection Management

- **Lazy Loading**: MCP servers started only when needed
- **Connection Pooling**: Reuse existing connections when possible
- **Timeout Handling**: 30-second timeouts for all MCP operations
- **Error Recovery**: Automatic reconnection for transient failures

### Memory Management

- **Session Compression**: Old conversations automatically compressed
- **Tool Result Caching**: Frequent tool results cached locally
- **Resource Cleanup**: Proper cleanup of MCP connections on exit

### Concurrency

- **Thread Safety**: All MCP operations are mutex-protected
- **Async Processing**: Non-blocking tool execution where possible
- **Worker Pools**: Efficient resource utilization for parallel operations

## Integration Points

### Chat Session Integration

```go
type Session struct {
    provider   llm.Provider
    messages   []llm.ChatMessage
    model      string
    mcpManager *mcp.Manager  // Automatic MCP integration
}
```

### LLM Provider Integration

MCP tools are automatically discovered and included in LLM context:

1. **Tool Discovery**: Registry lists all available tools
2. **Schema Injection**: Tool schemas provided to LLM for function calling
3. **Execution Routing**: LLM tool calls routed to appropriate handlers
4. **Result Integration**: Tool results seamlessly integrated into conversation

### CLI Integration

MCP functionality exposed through intuitive CLI commands:

- `vyb mcp list` - Server management
- `vyb mcp connect <server>` - Interactive connection
- `vyb mcp tools` - Tool discovery
- Automatic integration in `vyb` and `vyb chat` modes

## Design Principles

1. **Security First**: Every MCP operation validated through security constraints
2. **Privacy Preservation**: No external data transmission, local-only processing
3. **Developer Experience**: Intuitive commands and seamless integration
4. **Performance**: Efficient resource usage and responsive interactions
5. **Extensibility**: Plugin architecture through MCP protocol
6. **Reliability**: Robust error handling and recovery mechanisms

This architecture enables vyb-code to provide Claude Code-equivalent functionality while maintaining strict local privacy and security guarantees.