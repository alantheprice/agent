# Agent Template System

A flexible, JSON-driven framework for creating AI-powered agents that can execute complex workflows with sophisticated context management and data flow capabilities.

## Overview

The Agent Template System allows you to define AI agents through declarative JSON configurations, enabling rapid development of specialized AI workflows for tasks like code review, web scraping, content generation, data analysis, and more.

## Quick Start

```bash
# Build the agent
go build -o agent-template .

# Run an agent
./agent-template process examples/configs/web_scraper_simple.json

# Validate configuration
./agent-template validate examples/configs/content_creator.json
```

## Core Features

### 🔧 JSON-Driven Configuration
Define agents entirely through JSON files with no code changes required:
```json
{
  "agent": {
    "name": "My Agent",
    "description": "Custom AI workflow"
  },
  "workflows": [...],
  "tools": {...}
}
```

### 🧠 Multi-LLM Support
Built-in support for multiple LLM providers:
- OpenAI (GPT-3.5, GPT-4)
- Anthropic (Claude)
- Google (Gemini)
- Ollama (Local models)
- DeepInfra
- Groq

### 🔄 Workflow Orchestration
Define complex multi-step workflows with dependencies:
```json
{
  "steps": [
    {"name": "data_fetch", "type": "tool"},
    {"name": "analysis", "type": "llm", "depends_on": ["data_fetch"]},
    {"name": "output", "type": "tool", "depends_on": ["analysis"]}
  ]
}
```

## Enhanced Context Flow System

The Agent Template System features a sophisticated context management system that enables complex data flows between workflow steps.

### Context Reference Patterns

#### Basic Template Substitution
```json
{
  "prompt": "Analyze this data: {previous_step}"
}
```

#### Structured Data Access (Planned)
```json
{
  "prompt": "Review file: {git_diff.files[0].name} with {git_diff.added_lines} additions"
}
```

#### Context Transformers (Planned)
```json
{
  "context_transforms": [
    {
      "source": "git_diff.output",
      "transform": "extract_lines", 
      "params": {"pattern": "^\\+"},
      "store_as": "added_lines"
    }
  ]
}
```

#### Analysis Chains (Planned)
```json
{
  "analysis_chain": {
    "input": "code_changes",
    "stages": [
      {"type": "security_scan", "store_as": "security_issues"},
      {"type": "quality_analysis", "store_as": "quality_metrics"},
      {"type": "synthesis", "inputs": ["security_issues", "quality_metrics"]}
    ]
  }
}
```

#### Advanced Context Management (Planned)
```json
{
  "context_scope": {
    "inherit": ["git_diff", "user_preferences"],
    "private": ["api_keys", "temp_data"],
    "export": ["final_analysis"]
  }
}
```

### Data Flow Capabilities

#### Current Capabilities
- **Simple String Substitution**: `{step_name}` replaces with step output
- **Dependency-Based Execution**: Steps wait for required dependencies
- **Global Context Storage**: All step results available to subsequent steps
- **Parameterized Variables**: `{{variable}}` for configuration templating

#### Planned Enhancements

**Phase 1: Enhanced Template Engine**
- Dot notation access: `{step.field.subfield}`
- Array indexing: `{step.items[0]}`
- Built-in functions: `{len(step.items)}`, `{join(step.lines, '\n')}`

**Phase 2: Context Transformers**
- Pre-processing filters and transforms
- Post-processing data manipulation
- Built-in extractors and aggregators

**Phase 3: Analysis Chain Primitives**
- Multi-stage reasoning patterns
- Context accumulation strategies
- Synthesis and merge operations

**Phase 4: Advanced Context Management**
- Context scoping and inheritance
- Conditional context flow
- Type validation and constraints

## Agent Types & Examples

### Code Review Agent
Analyzes git changes, provides detailed feedback, and generates commit messages:
```bash
./agent-template process examples/git_workflow/git_workflow_assistant.json
```

### Web Scraper Agent
Extracts structured data from websites according to JSON schemas:
```bash
./agent-template process examples/configs/web_scraper.json
```

### Content Creator Agent
Generates articles, blog posts, and marketing content:
```bash
./agent-template process examples/configs/content_creator.json
```

### Multi-Agent Orchestration
Coordinate multiple agents for complex workflows:
```bash
./agent-template process examples/multi_agent_orchestration/content_pipeline.json
```

## Configuration Schema

### Agent Definition
```json
{
  "agent": {
    "name": "Agent Name",
    "description": "What this agent does",
    "version": "1.0.0",
    "max_iterations": 10,
    "timeout": "5m",
    "interactive": false
  }
}
```

### LLM Configuration
```json
{
  "llm": {
    "provider": "openai|anthropic|gemini|ollama|deepinfra|groq",
    "model": "gpt-4|claude-3-sonnet|gemini-pro",
    "temperature": 0.7,
    "max_tokens": 4000,
    "system_prompt": "You are an expert assistant..."
  }
}
```

### Workflow Steps
```json
{
  "steps": [
    {
      "name": "step_name",
      "type": "tool|llm|llm_display|condition|loop|parallel",
      "config": {...},
      "depends_on": ["previous_step"],
      "retry": {
        "max_attempts": 3,
        "backoff": "exponential"
      },
      "continue_on_error": false
    }
  ]
}
```

### Tools Configuration
```json
{
  "tools": {
    "shell_command": {"enabled": true},
    "file_reader": {"enabled": true},
    "web_fetch": {"enabled": true},
    "git_diff": {"enabled": true},
    "ask_user": {"enabled": true}
  }
}
```

## Advanced Features

### Data Sources
Ingest data from multiple sources:
- Files (JSON, text, CSV)
- Web APIs
- Databases
- Git repositories
- User input

### Security Controls
- Path restrictions
- Command filtering
- API key management
- Timeout enforcement
- Resource limits

### Output Processing
Multiple output formats and destinations:
- Console output
- File writing
- JSON structured data
- Markdown reports
- Database storage

### Monitoring & Metrics
- Execution timing
- Token usage tracking
- Cost monitoring
- Error logging
- Performance analytics

## Development Roadmap

### Phase 1: Enhanced Template Engine ⏳
- [ ] Dot notation access (`{step.field}`)
- [ ] Array indexing (`{step[0]}`)
- [ ] Built-in functions (`{len()}`, `{join()}`)
- [ ] Type-safe template rendering

### Phase 2: Context Transformers ⏳
- [ ] Pre/post-processing transforms
- [ ] Built-in extractors and filters
- [ ] Custom transformer plugins
- [ ] Data validation pipeline

### Phase 3: Analysis Chain Primitives ⏳
- [ ] Multi-stage reasoning patterns
- [ ] Context accumulation strategies
- [ ] Synthesis operations
- [ ] Chain validation

### Phase 4: Advanced Context Management ⏳
- [ ] Context scoping and inheritance
- [ ] Conditional context flow
- [ ] Type system and validation
- [ ] Context debugging tools

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.