# Changelog

All notable changes to vyb-code will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v2.0.0] - 2025-09-10

### Added

- **🎯 Vibe Coding Mode Default**: Interactive coding experience now default interface replacing traditional terminal mode
- **🤖 AI-Powered Code Intelligence**: Complete AI service integration layer with multi-repository workspace management
- **📊 Advanced Project Analyzer**: Comprehensive project analysis with architecture mapping, dependency analysis, and security scanning
- **🔧 Intelligent Build System**: Auto-detection and management of build systems (Makefile, Docker, GitHub Actions, Go native)
- **🌐 Multi-Language Support Extended**: Full support for Rust, Java, C++, C with intelligent dependency parsing
- **🔍 Claude Code Tools Complete**: All 10 core tools implemented (Bash, File Operations, Search, Web Integration)
- **📝 File Editing Tools**: Advanced Edit, MultiEdit, Read, Write tools with workspace security
- **🌐 Web Integration**: WebFetch content retrieval and WebSearch with tech site integration
- **🏗️ Version Management**: Automated version management system with centralized control
- **📦 Dependency Injection**: Modern architecture with dependency injection container
- **📋 Enhanced Configuration**: Comprehensive LLM and system settings management

### Enhanced

- **🔒 Security Framework**: Enhanced constraints validation and command execution security
- **📚 Logging System**: Modernized logger interface with structured logging
- **🏛️ Architecture Modernization**: Cleaner separation of concerns with command handlers
- **🧪 Test Coverage**: Comprehensive test suites for Phase 5 input system and AI package
- **🔧 Development Environment**: Complete VSCode development environment configuration
- **⚙️ CI/CD Improvements**: Resolved test failures, formatting issues, and workflow permissions

### Fixed

- **🔧 Release Workflow**: Fixed GitHub Actions duplicate release failures and permission issues  
- **🧪 Test Stability**: Resolved data race conditions, CI test failures, and mock component issues
- **📝 Code Quality**: Fixed formatting issues, missing newlines, and gofmt compliance
- **🔌 MCP Integration**: Updated logger calls to match new interface signature
- **🎨 Terminal Mode**: Restored Enhanced Terminal Mode functionality

### Architecture

- **🏗️ Modern CLI Design**: Dependency injection architecture in main.go
- **📦 Package Separation**: Command handlers extracted for better separation of concerns  
- **🔧 Tool Integration**: Claude Code tools integrated into unified ToolRegistry
- **🤖 AI Services**: Multi-repository workspace management and code generation engines
- **📊 Visualization**: Dependency visualization engine for project understanding

### Technical Improvements

- **📊 Code Analysis Engine**: AI-powered code analysis with comprehensive insights
- **🔄 Code Generation**: AI-powered code generation engine for development acceleration
- **🔍 Search Enhancement**: Advanced file search and grep engine improvements
- **⚡ Performance**: Input system optimizations with security, completion, and performance enhancements
- **🏛️ Centralized Management**: Version management centralization and build process improvements

## [v1.0.1] - 2025-09-05

### Fixed

- **🛠️ Windows Compatibility**: Resolved syscall compatibility issues preventing Windows builds
- **🔧 Platform Separation**: Split terminal size detection into platform-specific files
- **✅ Complete Multi-platform Support**: All 5 binaries (Linux/macOS/Windows × amd64/arm64) now build successfully

### Technical Details

- Added `reader_unix.go` with Unix syscall implementation (`TIOCGWINSZ`)
- Added `reader_windows.go` with Windows API implementation (`GetConsoleScreenBufferInfo`)
- Removed platform-specific imports from common `reader.go`
- Fixed GitHub Actions release workflow Windows build failures

## [v1.0.0] - 2025-09-05

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

[v2.0.0]: https://github.com/glkt3912/vyb-code/releases/tag/v2.0.0
[v1.0.1]: https://github.com/glkt3912/vyb-code/releases/tag/v1.0.1
[v1.0.0]: https://github.com/glkt3912/vyb-code/releases/tag/v1.0.0
