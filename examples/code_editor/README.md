# LLM Code Editor

An intelligent AI-powered code editor that uses natural language to understand requirements, semantic search to find relevant files, and makes precise edits with full context understanding.

## 🔄 Workflow Overview

```
1. Welcome & Setup         → Introduction and requirement collection
2. Collect Requirements    → User describes desired code changes (parallel)
3. Build Embeddings        → Index workspace files for semantic search (parallel)
4. Analyze Requirements    → Extract key concepts and search terms
5. Select Relevant Files   → Use embeddings to find 3 most relevant files
6. Confirm Selection       → User approval of selected files
7. Perform Code Editing    → Agentic editing with mandatory file reading
8. Validate Changes        → Run compilation/tests if available
9. Provide Summary         → Comprehensive report of changes made
```

## 🎯 Key Features

### 🧠 **Semantic File Selection**
- Uses embeddings to understand code semantics
- Searches across multiple file types (Go, JS, TS, Python, etc.)
- Excludes build artifacts and dependencies automatically
- Finds the 3 most relevant files based on user requirements

### 🔍 **Intelligent Code Analysis**
- Analyzes user requirements to extract technical concepts
- Identifies change types (feature, bugfix, refactor, enhancement)
- Generates targeted search keywords for file selection
- Considers file patterns and naming conventions

### ✏️ **Safe Code Editing**
- **MANDATORY RULE**: Always reads complete file contents before editing
- Uses available tools (read_file, write_file, shell_command, etc.)
- Makes minimal, precise changes that preserve code quality
- Validates syntax and compilation when possible

### 🚦 **Parallel Processing**
- Collects user requirements and builds embeddings simultaneously
- Maximizes efficiency by running independent tasks in parallel
- Reduces overall workflow execution time

### ⚙️ **Tool-Based Architecture**
- Leverages built-in tools for file operations
- Supports shell commands for validation and testing
- Interactive user confirmation at key decision points
- Comprehensive error handling and logging

## 🚀 Usage

```bash
# Set your DeepInfra API key for embeddings
export DEEPINFRA_API_KEY="your-api-key-here"

# Ensure you're in a code repository you want to edit
cd /path/to/your/project

# Run the code editor
./examples/code_editor/run.sh
```

## 📁 Files

- **`llm_code_editor.json`** - Agent configuration with workflow definition
- **`run.sh`** - Execution script
- **`README.md`** - This documentation

## ⚙️ Configuration

### LLM Settings
- **Model**: DeepSeek V3.1 (optimized for code understanding)
- **Temperature**: 0.2 (precise, consistent responses)
- **Max Tokens**: 4,000 per request
- **Budget**: $5.00 max, 200K tokens total

### Embedding Settings
- **Provider**: DeepInfra with sentence-transformers/all-MiniLM-L6-v2
- **Chunk Size**: 1000 characters for large files
- **File Patterns**: Supports 18+ programming languages
- **Storage**: Local cache in `./embeddings_cache`

### Security
- Restricted to current directory and parent directories
- 1MB max file size for analysis  
- Allowed commands: go, npm, python, node, bash
- Path validation and traversal protection

## 🎭 Interactive Decision Points

The workflow includes user confirmation at key stages:

### **1. File Selection Confirmation**
```
I've selected these files for editing based on your requirements:

- ./pkg/generic/agent.go (similarity: 0.85)
- ./cmd/agent.go (similarity: 0.75) 
- ./pkg/generic/workflow_engine.go (similarity: 0.65)

Do you want to proceed with editing these files? (y/n/modify):
```

### **2. Requirements Collection**
```
What code changes would you like me to make? Please describe the 
functionality, features, or fixes you need:
```

## 🛡️ Safety Features

### **Mandatory File Reading**
- **NEVER** edits a file without first reading its complete contents
- Understands full file context before making changes
- Prevents destructive edits or context-unaware modifications

### **Validation Pipeline**
- Attempts to compile/validate changes after editing
- Supports Go, Node.js, and Python project validation
- Reports success/failure of validation attempts

### **Minimal Changes Philosophy**
- Makes precise, targeted edits that address requirements
- Preserves existing code structure and patterns
- Maintains code quality and consistency

## 🔧 Customization

Edit `llm_code_editor.json` to:
- Adjust embedding file patterns and exclusions
- Modify validation commands for your tech stack
- Change similarity thresholds for file selection
- Update timeout values for different operations
- Add custom metadata for embedding context

### Supported File Types
```
*.go, *.js, *.ts, *.tsx, *.py, *.java, *.cpp, *.c, *.h, 
*.cs, *.rb, *.php, *.swift, *.kt, *.rs, *.vue, *.jsx, *.json
```

### Excluded Patterns
```
node_modules/**, .git/**, vendor/**, dist/**, build/**, 
*.min.js, *.map
```

## 📊 Expected Output

The workflow generates structured output including:
- User requirements analysis with extracted concepts
- Selected files with similarity scores and previews
- Detailed summary of changes made to each file
- Validation results (compilation/testing status)
- Recommendations for next steps

## 🚧 Advanced Usage

### Custom File Patterns
Modify the `file_patterns` in the embedding configuration to include additional file types specific to your project.

### Integration with CI/CD
The validation step can be extended to run your project's specific test suite or linting tools by modifying the `validate_changes` step.

### Multi-Repository Support
The embedding system can be configured to index multiple source paths if your project spans multiple repositories.

This code editor is ideal for projects where you need intelligent, context-aware code modifications based on natural language requirements.