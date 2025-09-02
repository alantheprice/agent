# Agent Template System

A flexible, JSON-driven framework for creating AI-powered agents that can execute complex workflows with sophisticated context management and data flow capabilities.

## Overview

The Agent Template System allows you to define AI agents through declarative JSON configurations, enabling rapid development of specialized AI workflows for tasks like code review, web scraping, content generation, data analysis, and more.

## Quick Start

```bash
# Build the agent
go build -o agent .

# Set up credentials (interactive)
./agent setup-provider deepinfra

# Run an agent (will auto-prompt for credentials if missing)
./agent process examples/configs/web_scraper_simple.json

# List available providers and models
./agent list-providers
./agent list-models deepinfra
```

## Credential Management

The system provides intelligent, automatic credential management with multiple layers of security and convenience.

### 🔐 Automatic Credential Detection

Credentials are detected in this priority order:

1. **Environment Variables** (highest priority)
   ```bash
   export DEEPINFRA_API_KEY="your-key-here"
   export OPENAI_API_KEY="your-openai-key"
   ```

2. **Credentials File** (`~/.agents/credentials.json`)
   - Automatically created on first use
   - Secure permissions (600)
   - JSON format for easy management

3. **Interactive Prompt** (when agent runs)
   - Auto-prompts when credentials missing
   - Provides helpful links to get API keys
   - Secure password-style input
   - Automatically saves for future use

### 📋 Credential Commands

```bash
# Interactive setup for specific provider
./agent setup-provider deepinfra

# Import from legacy location
./agent seed-keys

# Test credential system
./agent test-credentials

# List provider status
./agent list-providers

# Test specific provider
./agent test-provider deepinfra
```

### 🔑 Provider API Keys

Get your API keys from these providers:

| Provider | Environment Variable | Get API Key |
|----------|---------------------|-------------|
| **DeepInfra** | `DEEPINFRA_API_KEY` | https://deepinfra.com/dash/api_keys |
| **DeepSeek** | `DEEPSEEK_API_KEY` | https://platform.deepseek.com/api_keys |
| **Cerebras** | `CEREBRAS_API_KEY` | https://cloud.cerebras.ai/platform |
| **Gemini** | `GEMINI_API_KEY` | https://makersuite.google.com/app/apikey |
| **Groq** | `GROQ_API_KEY` | https://console.groq.com/keys |
| **GitHub** | `GITHUB_TOKEN` | https://github.com/settings/tokens |
| **Lambda AI** | `LAMBDA_API_KEY` | https://lambdalabs.com/service/gpu-cloud |
| **OpenAI** | `OPENAI_API_KEY` | https://platform.openai.com/api-keys |
| **Jina AI** | `JINA_API_KEY` | https://jina.ai/ |

### 💡 Usage Examples

```bash
# Use environment variable (temporary)
DEEPINFRA_API_KEY="your-key" ./agent process config.json

# Set up persistent credentials
./agent setup-provider deepinfra  # Interactive setup

# Check what's configured
./agent test-credentials

# Run agent (auto-prompts if needed)
./agent process examples/configs/content_creator.json
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
- **DeepInfra** (DeepSeek-V3.1, Llama, Mixtral) - *Default*
- **DeepSeek** (DeepSeek-Chat, DeepSeek-Coder)
- **Cerebras** (Llama 3.1 70B/8B)
- **Google** (Gemini Pro, Gemini 1.5)
- **Groq** (Llama 3.1, Mixtral)
- **GitHub Models** (GPT-4o, Llama 3.1)
- **Lambda AI** (Hermes 3 Llama)
- **Ollama** (Local models)
- **OpenAI** (GPT-4, GPT-3.5)
- **Jina AI** (Embeddings)

### 🔐 Intelligent Credential Management
Automatic credential detection and setup:
- **Environment Variables**: Priority detection (e.g., `DEEPINFRA_API_KEY`)
- **Secure Storage**: Encrypted storage in `~/.agents/credentials.json`
- **Interactive Setup**: Auto-prompts for missing keys when needed
- **Legacy Migration**: Import from `~/.ledit/api_keys.json`

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
./agent process examples/git_workflow/git_workflow_assistant.json
```

### Web Scraper Agent
Extracts structured data from websites according to JSON schemas:
```bash
./agent process examples/configs/web_scraper.json
```

### Content Creator Agent
Generates articles, blog posts, and marketing content:
```bash
./agent process examples/configs/content_creator.json
```

### Multi-Agent Orchestration
Coordinate multiple agents for complex workflows:
```bash
./agent process examples/multi_agent_orchestration/content_pipeline.json
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