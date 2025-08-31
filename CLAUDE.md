# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

vyb-code is a local AI coding assistant that provides Claude Code-equivalent functionality using local LLMs. The project aims to offer privacy-focused AI coding assistance with a natural "vibe" development experience.

**Core Concept**: "Feel the rhythm of perfect code" - Local LLM-based coding assistant prioritizing privacy and developer experience.

## Architecture

This is a Go-based CLI application with the following planned structure:

```
vyb-code/
├── cmd/vyb/              # Main CLI entry point
├── internal/
│   ├── llm/             # LLM integration (Ollama, LM Studio, vLLM)
│   ├── tools/           # File operations, command execution, Git
│   ├── chat/            # Conversation session management
│   ├── config/          # Configuration management
│   ├── cache/           # Response caching
│   └── security/        # Security constraints
└── pkg/types/           # Public type definitions
```

## Development Commands

**Note**: This is a new project. Once implemented, common commands will likely include:

```bash
# Build the project
go build -o vyb ./cmd/vyb

# Run tests
go test ./...

# Run specific tests
go test ./internal/llm -v

# Lint (when configured)
golangci-lint run

# Install locally
go install ./cmd/vyb
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

### Phase 1: MVP (3-4 weeks)
- Basic CLI structure with Cobra
- Ollama integration
- File read/write operations
- Interactive mode
- Configuration management

### Phase 2: Feature Expansion (3 weeks)
- Command execution
- Git integration
- Project analysis
- Multi-language support foundation

### Phase 3: Quality & Distribution (2 weeks)
- Testing infrastructure
- Performance optimization
- Package distribution
- Documentation

## Development Priorities

1. **Privacy First**: All processing must remain local
2. **Security**: Implement comprehensive command/file access restrictions
3. **Performance**: Optimize for responsive user experience
4. **Extensibility**: Design for future plugin/template systems

## Configuration

Planned configuration system using `~/.vyb/config.yaml`:
- LLM provider and model selection
- Security settings and command restrictions
- Workspace and file access policies
- Timeout and performance tuning