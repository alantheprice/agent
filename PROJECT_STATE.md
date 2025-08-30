# Agent Template System - Project State & Goals

## Project Overview

**Current Status**: In transition from coding-specific agent to configuration-driven generic agent framework  
**Last Updated**: 2025-01-30  
**Target Architecture**: Generic agent templating system supporting any LLM-powered agentic workflow

## Vision & Goals

### Primary Goal
Transform from a monolithic coding agent (`ledit`) into a flexible, configuration-driven generic agent template system that can be used to create any type of agentic workflow.

### Key Objectives
1. **Generic Agent Framework**: Create a system that supports any agent type through JSON configuration
2. **Multi-Agent Orchestration**: Enable coordination of multiple specialized agents 
3. **Workflow-Driven Architecture**: Support complex, multi-step workflows with dependencies
4. **LLM Provider Agnostic**: Support multiple LLM providers (OpenAI, Anthropic, Gemini, Ollama, etc.)
5. **Template Library**: Build a collection of pre-configured agent templates for common use cases

## Current Architecture Analysis

### Legacy Components (ledit-specific)
- **Command Structure**: CLI commands like `code`, `fix`, `commit`, `review`, `question`
- **Workspace Management**: File indexing and context management for code editing
- **Git Integration**: Automated commit message generation and change tracking
- **Code-Specific Features**: Language-aware editing, build validation, linting

### Emerging Generic Architecture

#### Configuration System
- **Agent Definitions**: JSON-based agent configuration with personas, skills, and models
- **Workflow Specifications**: Multi-step processes with dependencies and validation
- **Data Sources**: Flexible input handling (stdin, files, directories, web)
- **Output Formats**: Multiple output destinations and formats

#### Core Components (Generic)
- **Agent Runtime**: Executes configured agents with specified workflows
- **LLM Integration**: Provider-agnostic model management
- **Tool System**: Extensible tool framework for agent capabilities
- **Validation Framework**: Configurable validation rules and checks

## Transition State

### What's Working (Generic)
✅ **Multi-Agent Orchestration**: `ledit process` command with JSON configuration  
✅ **Agent Configuration**: JSON-based agent definitions with personas and skills  
✅ **Workflow System**: Steps with dependencies, timeouts, and retries  
✅ **Budget Controls**: Token and cost management per agent  
✅ **LLM Provider Support**: Multiple model providers with override capabilities  

### What's Legacy (Coding-Specific)
❌ **CLI Commands**: `code`, `fix`, `commit`, `review` are coding-specific  
❌ **Workspace Context**: File indexing assumes code editing use case  
❌ **Git Integration**: Hardcoded for software development workflows  
❌ **Validation Commands**: Build/test/lint assume development environment  

### Configuration Examples Available
- **Research Assistant**: Information gathering and synthesis
- **Web Scraper**: Data extraction and processing
- **Content Creator**: Content generation workflows
- **Data Analyzer**: Data processing and analysis
- **Multi-Agent Development**: Software development with specialized agents

## Testing & Validation Status

### E2E Test Categories
1. **Legacy Tests** (Need updating/removing):
   - `test_agent_*_workflow.sh` - Coding-specific workflows
   - `test_fix_command.sh` - Code fixing functionality
   - `test_multi_file_edit.sh` - Code editing operations
   - Build/test/lint validation tests

2. **Generic Tests** (Need creating):
   - Configuration validation tests
   - Multi-agent orchestration tests
   - Workflow execution tests
   - Provider integration tests
   - Template validation tests

### Current Test Status
⚠️ **Most E2E tests are outdated** - designed for the legacy coding agent functionality

## Next Steps & Priorities

### Phase 1: Core Framework Completion
1. **Template Validation System**: Ensure configuration schemas are robust
2. **Generic Tool Framework**: Abstract tool system from coding-specific tools
3. **Provider Integration**: Test all LLM providers with generic configurations
4. **Error Handling**: Improve error messages and recovery mechanisms

### Phase 2: Testing Framework
1. **Update E2E Tests**: Replace coding-specific tests with generic workflow tests
2. **Configuration Tests**: Validate all example configurations work correctly
3. **Integration Tests**: Test multi-agent orchestration scenarios
4. **Performance Tests**: Validate system performance with complex workflows

### Phase 3: Documentation & Examples
1. **User Guides**: Create comprehensive documentation for creating custom agents
2. **Template Library**: Expand collection of pre-built agent configurations
3. **Migration Guide**: Document transition from ledit to generic agent system
4. **Best Practices**: Guidelines for effective agent configuration

### Phase 4: Legacy Cleanup
1. **Command Deprecation**: Phase out coding-specific commands or make them optional
2. **Code Refactoring**: Remove hardcoded assumptions about coding workflows
3. **Configuration Migration**: Tools to convert legacy configurations

## Technical Debt & Blockers

### Current Issues
- **Mixed Architecture**: Legacy code-editing features mixed with generic framework
- **Test Coverage**: E2E tests don't cover new generic functionality
- **Documentation Gap**: Limited documentation for creating custom agent configurations
- **Schema Validation**: Need stronger validation for configuration files

### Dependencies
- Go ecosystem and CLI framework (Cobra)
- Multiple LLM provider SDKs
- JSON schema validation
- File system operations and workspace management

## Success Metrics

### Framework Adoption
- Number of different agent types successfully configured
- Variety of use cases beyond software development
- Community contributions of new templates

### Technical Quality
- All E2E tests passing for generic functionality
- Configuration validation catches common errors
- Multi-agent workflows execute reliably
- Performance benchmarks for complex orchestrations

### User Experience
- Clear documentation for creating custom agents
- Easy configuration and deployment
- Helpful error messages and debugging tools
- Smooth migration path from legacy functionality

## Configuration Schema Evolution

### Current Schema Elements
- Agent definitions with personas and skills
- Workflow steps with dependencies
- Data sources and preprocessing
- Output configurations
- Security and validation rules
- Budget controls and timeouts

### Planned Improvements
- Schema versioning and migration support
- Enhanced validation rules
- Template inheritance and composition
- Dynamic configuration generation
- Runtime configuration updates

## Notes & Observations

- The multi-agent orchestration system is the strongest part of the new architecture
- Configuration examples show good variety of use cases beyond coding
- Legacy CLI commands still work but don't align with generic vision
- Need clearer separation between framework core and domain-specific tools
- Testing strategy needs complete overhaul to match new architecture

---

**Next Session TODO**:
1. Review and update E2E tests to focus on generic functionality
2. Test example configurations to ensure they work correctly  
3. Identify and document remaining legacy dependencies
4. Plan migration strategy for existing users