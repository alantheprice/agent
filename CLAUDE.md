# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Building and Running
```bash
# Build the main executable
go build -o agent-template .

# Run an agent configuration
./agent-template process examples/configs/content_creator.json

# Validate a configuration file
./agent-template validate examples/configs/web_scraper.json
```

### Testing
```bash
# Run unit tests with coverage
go test ./pkg/... -v -coverprofile=coverage.out

# Run the comprehensive test suite
./scripts/run-tests.sh

# Run E2E tests
./e2e_test_scripts/run_tests.sh

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Run tests for specific packages
go test ./pkg/generic/... -v
go test ./pkg/embedding/... -v
```

## Architecture Overview

This is a **JSON-driven AI agent framework** that enables creating specialized AI workflows without code changes. The system follows a modular, plugin-based architecture with sophisticated context management.

### Core Components

- **pkg/generic/**: Core framework components
  - `agent.go` - Main agent orchestration logic
  - `workflow_engine.go` - Step execution and dependency management  
  - `config.go` - Configuration loading and validation
  - `tool_registry.go` - Tool registration and execution
  - `template_engine.go` - Context substitution and templating

- **pkg/providers/**: LLM provider implementations
  - Support for OpenAI, Anthropic, Google Gemini, Ollama, DeepInfra, Groq
  - `llm/factory.go` - Provider factory pattern
  - `cache/response_cache.go` - Response caching system

- **pkg/orchestration/**: Multi-agent coordination
  - `multi_agent_orchestrator.go` - Orchestrates multiple agents
  - `dependencies.go` - Dependency resolution
  - `state.go` - Shared state management

- **pkg/embedding/**: Semantic search capabilities
  - `vector_db.go` - Vector storage and retrieval
  - `generator.go` - Text embedding generation
  - `datasource.go` - File indexing and search

- **cmd/**: CLI interface built with Cobra
  - `process.go` - Main command for running agents
  - `root.go` - CLI configuration and help

### Configuration System

Agents are defined entirely through JSON configuration files with these key sections:

- **agent**: Basic metadata, timeouts, iteration limits
- **llm**: LLM provider, model, temperature, system prompts
- **workflows**: Multi-step workflows with dependency chains
- **tools**: Available tools (shell_command, file_reader, web_fetch, git_diff, ask_user)
- **data_sources**: Input sources (files, directories, APIs, git repositories)
- **security**: Path restrictions, command filtering, resource limits

### Context Flow System

The framework features sophisticated context management:
- **Simple substitution**: `{step_name}` replaces with step output
- **Dependency-based execution**: Steps wait for required dependencies  
- **Global context storage**: All step results available to subsequent steps
- **Parameterized variables**: `{{variable}}` for configuration templating

Future enhancements planned:
- Dot notation access: `{step.field.subfield}`
- Context transformers and filters
- Multi-stage analysis chains
- Context scoping and inheritance

## Agent Examples

### Pre-configured Agents
- `examples/configs/content_creator.json` - Content generation and copywriting
- `examples/configs/web_scraper.json` - Web scraping and data extraction
- `examples/git_workflow/git_workflow_assistant.json` - Git analysis and commit generation
- `examples/code_editor/llm_code_editor.json` - AI-powered code editing with semantic search
- `examples/multi_agent_orchestration/` - Multi-agent workflows

### Running Examples
```bash
# Content creation
./agent-template process examples/configs/content_creator.json

# Code editing (requires DEEPINFRA_API_KEY)
export DEEPINFRA_API_KEY="your-api-key"
./agent-template process examples/code_editor/llm_code_editor.json

# Git workflow assistant
./agent-template process examples/git_workflow/git_workflow_assistant.json
```

## Key Implementation Patterns

### Workflow Step Dependencies
Steps can depend on previous steps using `depends_on` arrays. The workflow engine resolves dependencies and executes steps in correct order.

### Tool System
Tools are registered in `pkg/generic/tool_registry.go` and can be:
- Shell commands with security validation
- File operations with path restrictions  
- Web requests with timeout controls
- Git operations for repository analysis
- User interaction for approval workflows

### Error Handling
- Configurable retry logic with exponential backoff
- `continue_on_error` flag for non-critical steps
- Comprehensive error logging with context

### Security Controls
- Path allowlisting for file operations
- Command filtering for shell execution
- API key management through environment variables
- Resource limits (timeouts, file sizes, iterations)

## Development Guidelines

### Adding New Tools
1. Implement tool interface in `pkg/generic/tool_registry.go`
2. Add security validation in tool execution
3. Update configuration schema if needed
4. Add tests in `pkg/generic/tool_registry_test.go`

### Adding LLM Providers
1. Create provider in `pkg/providers/llm/[provider]/`
2. Implement the `LLMProvider` interface
3. Register in `pkg/providers/llm/factory.go`
4. Add integration tests

### Configuration Changes
- Configuration schemas are in `schemas/` directory
- Validate changes don't break existing agent configs
- Update example configurations in `examples/configs/`