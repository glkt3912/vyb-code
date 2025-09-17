# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

vyb-code is a local AI coding assistant that provides Claude Code-equivalent functionality using local LLMs. The project aims to offer privacy-focused AI coding assistance with a natural "vibe" development experience.

**Core Concept**: "Feel the rhythm of perfect code" - Local LLM-based coding assistant prioritizing privacy and developer experience.

**Current Status**: Phase 8 completed - Post-Migration Cleanup Complete with Full Functionality Restored. Features include unified system architecture, hardcoding problem resolution with scientific measurement systems (Semantic Entropy, Logical Structure Analysis, Guilford Creativity Theory), interactive coding assistance with context compression (70-95% efficiency), intelligent diff analysis with risk assessment, real-time code suggestions, complete Claude Code-style interactive mode, all 10 core tools (Bash, File Operations, Search, Web Integration), advanced project analysis, comprehensive functionality validation, and fully operational chat/vibe coding functionality.

## Architecture

This is a Go-based CLI application with the following planned structure:

```
vyb-code/
├── cmd/vyb/              # Main CLI entry point
├── internal/
│   ├── llm/             # LLM integration (Ollama, LM Studio, vLLM)
│   ├── tools/           # File operations, command execution, Git
│   ├── handlers/        # Chat and other request handlers
│   ├── config/          # Configuration management
│   ├── adapters/        # Gradual migration system adapters
│   ├── security/        # Security constraints & LLM response protection
│   ├── input/           # Enhanced input system (security, completion, performance)
│   ├── mcp/             # Model Context Protocol implementation
│   ├── search/          # Advanced file search and grep engine
│   ├── session/         # Persistent conversation management
│   ├── streaming/       # Real-time response streaming
│   ├── performance/     # Metrics and optimization
│   ├── interactive/     # Vibe coding mode session management
│   ├── contextmanager/  # Intelligent context compression (70-95% efficiency)
│   ├── conversation/    # Memory-efficient dialogue management
│   ├── analysis/        # Scientific cognitive analysis system
│   ├── migration/       # Gradual migration system monitoring
│   └── ui/              # Interactive UI components (confirmations, dialogs)
└── pkg/types/           # Public type definitions
```

## Development Commands

```bash
# Quick development build
/usr/bin/make build
# or
./scripts/build.sh dev

# Run all tests
/usr/bin/make test
# or
go test ./...

# Run tests with coverage
/usr/bin/make coverage

# Run specific package tests
go test ./internal/config -v
go test ./internal/llm -v
go test ./internal/tools -v
go test ./internal/input -v
go test ./internal/performance -v
go test ./internal/security -v
go test ./internal/mcp -v
go test ./internal/search -v
go test ./internal/session -v
go test ./internal/streaming -v
go test ./internal/interactive -v
go test ./internal/contextmanager -v
go test ./internal/conversation -v
go test ./internal/ui -v

# Test input system components
go test ./internal/input -v -run TestSecurity
go test ./internal/input -v -run TestPerformance
go test ./internal/input -v -run TestCompletion

# Test MCP functionality
go test ./internal/mcp -v -run TestMCP
go test ./internal/config -v -run TestMCPServerConfig

# Test vibe coding mode functionality
go test ./internal/interactive -v -run TestInteractiveSession
go test ./internal/contextmanager -v -run TestContextManager
go test ./internal/ui -v -run TestConfirmationDialog

# Test scientific cognitive analysis system
go test ./internal/analysis -v -run TestSemanticEntropy
go test ./internal/analysis -v -run TestLogicalAnalyzer
go test ./internal/analysis -v -run TestCreativityScorer
go test ./internal/analysis -v -run TestCognitiveAnalyzer

# Format and lint
/usr/bin/make fmt
/usr/bin/make lint

# Full CI check
/usr/bin/make check

# Multi-platform build
./scripts/build.sh

# Release preparation
./scripts/release.sh v1.0.0
```

## Technical Stack

- **Language**: Go (chosen for performance, single binary distribution, concurrency)
- **LLM Integration**: HTTP API via Ollama (primary), LM Studio, vLLM
- **Recommended Models**:
  - Qwen2.5-Coder 32B/14B (primary)
  - DeepSeek-Coder-V2 16B (balanced)
  - CodeLlama 34B (stability)

## Key Requirements

### Core MVP Features

1. **Interactive CLI**: Real-time conversation with multi-turn context
2. **File Operations**: Read/write files, project search
3. **Command Execution**: Secure shell command execution (30s timeout)
4. **Git Integration**: Branch operations, commit generation, diff analysis

### Security Constraints

- **Local-only execution** (no external data transmission)
- **Command whitelist**: Restricted command execution
- **File access limits**: Project directory only
- **Input validation**: Injection attack prevention

### Performance Targets

- LLM response: <10 seconds
- File operations: <1 second
- Memory usage: <100MB (excluding LLM)

## Implementation Phases

### Phase 1: MVP (✅ Completed)

- ✅ Basic CLI structure with Cobra
- ✅ Ollama integration with HTTP API client
- ✅ File read/write operations with security constraints
- ✅ Interactive chat mode with conversation history
- ✅ Configuration management with JSON persistence

### Phase 2: Feature Expansion (✅ Completed)

- ✅ Command execution with security constraints (whitelist, timeout)
- ✅ Git integration (branch management, commits, status)
- ✅ Project analysis (language detection, dependencies, structure)
- ✅ Multi-language support foundation (Go, JS, Python)

### Phase 3: Quality & Distribution (✅ Completed)

- ✅ Testing infrastructure (unit tests for all core packages)
- ✅ Performance optimization (metrics collection, caching, worker pools)
- ✅ Package distribution (GitHub Actions, GoReleaser, multi-platform)
- ✅ Build system (Makefile, scripts, CI/CD pipeline)

### Phase 4: Claude Code Feature Parity (✅ Completed)

- ✅ MCP protocol implementation for external tool integration
- ✅ Advanced file search and grep system with project-wide indexing
- ✅ Persistent conversation sessions with export/import capabilities
- ✅ Streaming response processing for real-time user experience
- ✅ Comprehensive security enhancements against malicious LLM responses
- ✅ Enhanced CLI integration with backward compatibility
- ✅ Intelligent search with AST-based code structure analysis

### Phase 4+ Enhanced Terminal Mode (✅ Completed)

- ✅ **Claude Code-style terminal interface** (now **default mode**)
- ✅ **Japanese IME support** (resolved character disappearing issues)
- ✅ **Colored UI components** (ANSI color codes for prompts, logos, metadata)
- ✅ **Markdown formatting** (code blocks with syntax highlighting, bold text)
- ✅ **Automatic project context** (language detection, dependencies, Git info)
- ✅ **Real-time metadata display** (response time, token count, model name)
- ✅ **Convenient shortcuts** (`vyb s`, `vyb build`, `vyb test`)
- ✅ **Auto-detection systems** (Makefile/Go/Node.js build and test commands)
- ✅ **Claude Code-style interface** (streaming responses, intelligent completion, progress indicators)

### Phase 5: Enhanced Input System (✅ Completed)

- ✅ **Security enhancements** (input sanitization, buffer overflow protection, rate limiting)
- ✅ **Advanced autocompletion** (context-aware completion, Git integration, fuzzy matching)
- ✅ **Performance optimization** (worker pools, LRU caching, async processing, debouncing)
- ✅ **UTF-8 complete support** (Japanese IME, multibyte character processing, encoding validation)
- ✅ **Intelligent input processing** (project analysis, command prediction, history optimization)
- ✅ **Integrated architecture** (`internal/input/` package with comprehensive functionality)

### Phase 5+ Claude Code Tool Parity (✅ Completed)

- ✅ **Complete Claude Code Tool Suite** (all 10 core tools implemented)
- ✅ **Bash Tool** (secure command execution with timeout and validation)
- ✅ **File Operations** (Read, Write, Edit, MultiEdit with workspace security)
- ✅ **Search Tools** (Glob pattern matching, advanced Grep with regex/filters, LS directory listing)
- ✅ **Web Integration** (WebFetch content retrieval, WebSearch with domain filtering)
- ✅ **Security Framework** (comprehensive constraints, input validation, error handling)
- ✅ **Tool Registry Integration** (unified interface with native and MCP tools)
- ✅ **Functionality Validation** (comprehensive testing suite confirming Claude Code equivalence)

### Phase 7: Scientific Cognitive Analysis System (✅ Completed)

### Phase 8: Gradual Migration System (✅ Completed)

**完全な段階的移行アーキテクチャ**

- ✅ **適応アダプター設計** - レガシーと統合システム間のシームレス切り替え
- ✅ **設定駆動移行** - `config.json` で移行モードを制御
- ✅ **フォールバック機構** - エラー発生時の自動復旧システム
- ✅ **メトリクス収集** - 移行プロセスの監視と検証
- ✅ **CLI管理インターフェース** - `vyb config` コマンドでの移行制御
- ✅ **安全な統合移行** - レガシーシステムから統合システムへの完全移行
- ✅ **クリーンアップ完了** - レガシーファイル削除とコンパイル修正

**チャット機能復旧準備**

- ✅ **移行システム統合** - 段階的アダプターシステムの完全実装
- ✅ **レガシー参照削除** - `internal/chat`, `internal/stream` パッケージ参照の全面除去
- ✅ **ビルド安定化** - 全コンパイルエラー解消とテスト環境構築

**Hardcoding Problem Resolution - Fundamental Solution**

- ✅ **Semantic Entropy Confidence Measurement** (354 lines) - Dynamic confidence calculation using Farquhar et al. (2024) methodology
- ✅ **Logical Structure Reasoning Depth Analysis** (649 lines) - LogiGLUE framework + Toulmin argumentation model integration
- ✅ **Guilford Creativity Theory Measurement** (650 lines) - 4-element (fluency, flexibility, originality, elaboration) scientific evaluation
- ✅ **Natural Language Inference Analyzer** (495 lines) - Entailment relationship analysis for semantic clustering
- ✅ **Semantic Clustering Engine** (756 lines) - Meaning-based response grouping with similarity analysis
- ✅ **Entropy Calculation Engine** (649 lines) - von Neumann entropy for uncertainty quantification
- ✅ **Integrated Cognitive Analysis Framework** (550 lines) - Comprehensive system coordinating all analysis components
- ✅ **Dynamic Parameter Generation** - Complete replacement of fixed values (0.8, 0.9, 4) with scientific measurement
- ✅ **2024 Research Integration** - Implementation based on latest NLP and cognitive science research

**Vibe Coding Mode Continued Development**

- ✅ **Interactive Session Management** (3500+ lines) - Scientific analysis integrated vibe coding
- ✅ **Context Compression System** (70-95% efficiency) - Smart context management for enhanced performance
- ✅ **Intelligent Diff Analysis** - Risk assessment, file-specific change detection, security concern identification
- ✅ **Real-time Code Suggestions** - Context-aware suggestions with confirmation dialogs
- ✅ **Memory-efficient Dialogue Management** - Optimized conversation handling for long sessions
- ✅ **Default Experience Integration** - Vibe mode as primary interface replacing traditional terminal mode
- ✅ **Enhanced UI Components** - Modern confirmation dialogs and interactive elements

### Phase 2: Advanced Development Features (✅ Completed)

- ✅ **Advanced Project Analysis** (architecture mapping, dependency analysis, security scanning)
- ✅ **Intelligent Build System Integration** (auto-detection, pipeline management, multi-platform support)
- ✅ **Extended Language Support** (Rust, Java, C++, C with intelligent dependency parsing)
- ✅ **Security Analysis Engine** (secret leak detection, vulnerability scanning, code pattern analysis)
- ✅ **Project Health Scoring** (code quality metrics, test coverage analysis, maintainability assessment)
- ✅ **Build Pipeline Management** (preset pipelines, parallel execution, performance optimization)
- ✅ **Comprehensive Testing Framework** (integration tests, benchmark tests, error handling validation)

## Development Priorities

1. **Privacy First**: All processing must remain local
2. **Security**: Comprehensive protection against malicious LLM responses
3. **Performance**: Real-time streaming and concurrent processing
4. **Claude Code Parity**: Feature equivalence with enterprise capabilities
5. **Extensibility**: MCP protocol and plugin architecture
6. **Developer Experience**: Intuitive workflows and intelligent assistance
7. **Modern UI/UX**: Terminal user interface with brand identity and accessibility

## Configuration

**Comprehensive configuration system** using `~/.vyb/config.json`:

- ✅ LLM provider and model selection (`vyb config set-model`, `vyb config set-provider`)
- ✅ Timeout and file size limits
- ✅ Workspace mode restrictions
- ✅ Security settings and command restrictions
- ✅ Performance optimization settings
- ✅ **Migration system configuration** (completed unified mode after PR#32, PR#33)
- ✅ **Legacy TUI configuration** (deprecated - Claude Code風インターフェースが標準)

**Current config commands:**

```bash
vyb config list                    # Show current settings
vyb config set-model <model>       # Set LLM model
vyb config set-provider <provider> # Set LLM provider
vyb config set-migration-mode <mode> # Set migration mode (unified mode is default)

# Legacy TUI configuration commands (deprecated)
vyb config set-tui <true|false>      # TUI mode setting (deprecated)
vyb config set-tui-theme <theme>     # TUI theme setting (deprecated)

# Note: TUI configuration commands are deprecated after migration completion (PR#32, PR#33)
# Claude Code-style interface is now the default and only supported interface
```

**All implemented commands:**

```bash
# Interactive sessions (Terminal mode is now default!)
vyb                                # Start Claude Code-style interactive mode (DEFAULT)
vyb chat                           # Start interactive chat session (same as default)
vyb vibe                           # Start vibe coding mode explicitly (same as default)

# Search and discovery
vyb search <pattern>               # Search across project files
vyb search <pattern> --smart       # Intelligent search with AST analysis and relevance scoring
vyb search <pattern> --max-results N # Limit number of results
vyb search <pattern> --context      # Include/exclude context lines
vyb find <pattern>                 # Find files by name pattern
vyb grep <pattern>                 # Advanced grep with context

# Session management
vyb sessions list                  # List all conversation sessions
vyb sessions switch <id>           # Switch active session
vyb sessions export <id>           # Export session data
vyb sessions delete <id>           # Delete session

# Command execution (with security validation)
vyb exec <command>                 # Execute shell command securely

# Git operations
vyb git status                     # Show git status
vyb git branch [name]              # Create/list branches
vyb git commit <message>           # Create commit

# Project analysis
vyb analyze                        # Analyze project structure
vyb analyze --path <dir>           # Analyze specific directory

# Configuration
vyb config list                    # Show current settings
vyb config set-model <model>       # Set LLM model
vyb config set-provider <provider> # Set LLM provider

# Convenient shortcuts (NEW)
vyb s                              # Quick git status
vyb build                          # Auto-detect and build project
vyb test                           # Auto-detect and run tests
vyb quick explain <file>           # Explain file (development)
vyb quick gen <description>        # Generate code (development)
vyb quick summarize                # Summarize conversation (development)

# MCP (Model Context Protocol) operations
vyb mcp list                       # List configured MCP servers
vyb mcp add <name> <command>       # Add new MCP server
vyb mcp connect <server>           # Connect to MCP server
vyb mcp tools [server]             # List available tools
vyb mcp disconnect [server]        # Disconnect from MCP server
```

## Development Guidelines

### Code Comments

- **All comments must be in Japanese** for better readability by Japanese developers
- Include both technical explanations and purpose of functions/types
- Use format: `// 日本語での説明 (English technical terms if needed)`

### Git Workflow

- **Exclude AI attribution in commits**: Do not include "Generated with Claude Code" or "Co-Authored-By: Claude"
- **Clean commit messages**: Focus on clear, concise descriptions of changes
- **PR process**: Use feature branches, create descriptive PRs without AI-generated footers
- Follow conventional commit format: `feat:`, `fix:`, `docs:`, etc.

## Memories

- 🤖 Added a new memory to track project insights and development context
- 🚀 Phase 4+ completed: Enhanced Claude Code-style terminal mode with Japanese IME support, colored UI, Markdown formatting, automatic project context, and convenient shortcuts
- ✨ Terminal-mode is now the **default experience** - no flags needed for Claude Code-style interface
- 🔒 Phase 5 completed: Enhanced input system with comprehensive security, performance optimization, and intelligent completion features
- 🛠️ Phase 5+ completed: Full Claude Code tool parity achieved - all 10 core tools (Bash, File Operations, Search, Web Integration) implemented and validated with comprehensive security framework
- 🧠 Phase 7 completed: Scientific Cognitive Analysis System Implementation Complete - Hardcoding problem fundamentally resolved with 2024 research-based dynamic measurement systems (7 files, 4,100+ lines) replacing fixed values with Semantic Entropy, Logical Structure Analysis, and Guilford Creativity Theory
- 🔄 Phase 8 completed: Gradual Migration System Implementation Complete - Full system migration from compatibility to unified architecture with adapter-based transition, comprehensive cleanup, and chat functionality restoration preparation
