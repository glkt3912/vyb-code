# Changelog

All notable changes to vyb-code will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.0.0] - 2025-01-05

### Added

- **🎯 Claude Code-style Terminal Mode**: Default experience with Claude Code equivalent interface
- **🌈 Colorful UI**: ANSI color-coded green prompts, blue logos, and vibrant syntax highlighting
- **📝 Markdown Support**: Complete support for code block borders, syntax highlighting, and bold text
- **🏗️ Auto Project Context**: Automatic detection and inclusion of language, dependencies, and Git info
- **📊 Real-time Metadata**: Live display of response time, token count, and model name
- **🇯🇵 Japanese IME Support**: Complete resolution of character disappearing issues during Japanese input
- **🎨 Modern TUI**: Beautiful terminal UI experience powered by Bubble Tea framework
- **⚡ Convenient Shortcuts**: Fast commands like `vyb s` (git status), `vyb build`, `vyb test`
- **🔧 MCP Protocol**: Model Context Protocol implementation for external tool integration
- **🔍 Advanced File Search**: Project-wide indexing and intelligent search capabilities
- **💾 Persistent Sessions**: Conversation history saving, restoration, and export functionality
- **⚡ Streaming Responses**: Real-time LLM output processing
- **🛡️ Comprehensive Security**: Protection against malicious LLM responses, command execution constraints, and sensitive data protection

### Security

- Command execution whitelist control (30-second timeout)
- LLM response validation and filtering system
- Automatic detection and removal of sensitive information (passwords, API keys)
- File access restrictions (project directory only)
- Security assurance for MCP external tool integration

### Features

- **Privacy-focused**: All processing runs locally with no external data transmission
- **Interactive CLI**: Natural conversation-style coding assistance
- **Comprehensive Git Integration**: Branch management, commit creation, and status monitoring
- **Project Analysis**: Automatic analysis of file structure, language distribution, and dependencies
- **Multi-language Support**: Foundation for Go, JavaScript/Node.js, and Python
- **Configuration Management**: Persistent settings for model and provider management
- **Ollama Integration**: Local LLM connectivity via HTTP API

### Supported Platforms

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

### Recommended Models

- Qwen2.5-Coder 14B/32B (primary)
- DeepSeek-Coder-V2 16B (balanced)
- CodeLlama 34B (stability)

[v1.0.0]: https://github.com/glkt/vyb-code/releases/tag/v1.0.0