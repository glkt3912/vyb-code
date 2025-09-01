# MCP Server Configuration Examples

This document provides examples of how to configure MCP (Model Context Protocol) servers with vyb-code.

## Overview

vyb-code supports MCP for integrating external tools and resources. You can configure multiple MCP servers in your `~/.vyb/config.json` file.

## Configuration Format

```json
{
  "provider": "ollama",
  "model": "qwen2.5-coder:14b",
  "base_url": "http://localhost:11434",
  "timeout": 30,
  "max_file_size": 10485760,
  "workspace_mode": "project_only",
  "mcp_servers": {
    "filesystem": {
      "name": "filesystem",
      "command": ["npx", "@modelcontextprotocol/server-filesystem"],
      "args": ["/home/user/projects"],
      "environment": {},
      "workingDir": "",
      "enabled": true,
      "autoConnect": true
    },
    "git": {
      "name": "git",
      "command": ["npx", "@modelcontextprotocol/server-git"],
      "args": [],
      "environment": {},
      "workingDir": "",
      "enabled": true,
      "autoConnect": false
    }
  }
}
```

## Popular MCP Servers

### Filesystem Server
```bash
vyb mcp add filesystem npx @modelcontextprotocol/server-filesystem /path/to/directory
```

### Git Server
```bash
vyb mcp add git npx @modelcontextprotocol/server-git
```

### SQLite Server
```bash
vyb mcp add sqlite npx @modelcontextprotocol/server-sqlite path/to/database.db
```

### GitHub Server
```bash
vyb mcp add github npx @modelcontextprotocol/server-github
```

## CLI Commands

### List configured servers
```bash
vyb mcp list
```

### Connect to a specific server
```bash
vyb mcp connect filesystem
```

### List available tools
```bash
vyb mcp tools filesystem
vyb mcp tools  # All servers
```

### Disconnect from servers
```bash
vyb mcp disconnect filesystem
vyb mcp disconnect  # All servers
```

## Security Considerations

vyb-code applies strict security constraints to MCP tools:

- **Path restrictions**: MCP tools can only access files within the workspace directory
- **Command validation**: Dangerous commands are blocked through security constraints
- **Network restrictions**: URL-based operations are restricted by default
- **Tool whitelisting**: Only approved tool patterns are allowed

## Integration with Chat Sessions

When using interactive mode (`vyb` or `vyb chat`), MCP tools are automatically discovered and made available to the LLM. The conversation history tracks which MCP tools were used for debugging and auditing purposes.

## Troubleshooting

### Server connection issues
1. Verify the server command is correctly installed
2. Check that the server supports the MCP protocol version (2024-11-05)
3. Ensure working directory permissions are correct

### Tool execution failures
1. Check tool arguments match the expected schema
2. Verify security constraints allow the operation
3. Review server logs for detailed error messages

## Example Configurations

### Development Environment
```json
{
  "mcp_servers": {
    "filesystem": {
      "name": "filesystem",
      "command": ["npx", "@modelcontextprotocol/server-filesystem"],
      "args": ["/home/dev/projects"],
      "enabled": true,
      "autoConnect": true
    },
    "git": {
      "name": "git", 
      "command": ["npx", "@modelcontextprotocol/server-git"],
      "enabled": true,
      "autoConnect": true
    }
  }
}
```

### Data Analysis Environment
```json
{
  "mcp_servers": {
    "sqlite": {
      "name": "sqlite",
      "command": ["npx", "@modelcontextprotocol/server-sqlite"],
      "args": ["data/analytics.db"],
      "enabled": true,
      "autoConnect": false
    }
  }
}
```