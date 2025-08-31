# Git Workflow Assistant

An intelligent git workflow that validates staged files, performs thorough code reviews with a focus on LLM-generated code issues, and helps generate quality commit messages.

## 🔄 Workflow Overview

```
1. Validation Script      → Checks file sizes, types, staging health
2. User Decision          → Continue or cancel based on validation
3. Get Staged Changes     → Retrieve full git diff for analysis  
4. Infer Change Goals     → Understand the purpose/objective of changes
5. Thorough Code Review   → File-by-file analysis with LLM issue focus
6. Review User Decision   → Continue, save feedback, or cancel
7. Generate Commit Msg    → Create conventional commit message
8. Message User Decision  → Accept, revise, or cancel
9. Execute Commit         → Final commit with approved message
```

## 🎯 Key Features

### 📋 **Staging Validation** 
- File size analysis (flags >1MB, blocks >10MB)
- Binary file detection
- Generated/build file warnings  
- Total diff size checking (warns >2K lines, blocks >10K)

### 🧠 **Smart Goal Inference**
- Analyzes full changeset to understand objectives
- Identifies whether it's a feature, fix, refactor, etc.
- Provides context for detailed review

### 🔍 **LLM-Focused Code Review**
Specifically looks for issues common in AI-generated code:
- Over-complicated implementations
- Generic variable/function names
- Missing error handling
- Incomplete implementations
- Unnecessary abstractions
- Hardcoded values
- Missing input validation

### 📝 **Interactive Decision Points**
- **After validation**: Continue or cancel
- **After review**: Continue to commit, save feedback, or cancel
- **After commit message**: Accept, request revision, or cancel

### 💾 **Review Feedback Saving**
Option to save detailed review to timestamped markdown file for later reference.

## 🚀 Usage

```bash
# Ensure you have staged changes
git add <files>

# Run the workflow
./examples/git_workflow/run.sh
```

## 📁 Files

- **`git_workflow_assistant.json`** - Agent configuration
- **`run.sh`** - Execution script with git repo checks
- **`validate_staging.sh`** - Staging validation script 
- **`README.md`** - This documentation

## ⚙️ Configuration

### LLM Settings
- **Model**: Claude 3 Sonnet (optimized for code analysis)
- **Temperature**: 0.3 (focused, consistent responses)
- **Max Tokens**: 4,000 per request
- **Budget**: $10.00 max, 50K tokens total

### Security
- Restricted to git directory and current directory
- 1MB max file size for analysis
- Script execution with security validation

## 🎭 Interactive Decision Points

The workflow includes clear prompts at each decision point:

### **1. After Staging Validation**
```
## Staging Validation Complete
Please review the validation findings:
- File count and sizes
- Binary/generated file warnings  
- Total diff size analysis
- Overall validation status

Options:
- c (continue): Proceed with comprehensive code review
- n (cancel): Stop the workflow to address issues first
```

### **2. After Code Review**  
```
## Code Review Complete
I've completed thorough analysis including:
- Overall goal and purpose assessment
- File-by-file code quality review
- LLM-generated code issue detection
- Security and performance considerations

Options:
- c (continue): Proceed to commit message generation
- s (save): Save review feedback to markdown file
- n (cancel): Cancel the commit process
```

### **3. After Commit Message Generation**
```
## Proposed Commit Message
Review the generated conventional commit message.

Options:
- a (accept): Accept the message and execute git commit
- r (revise): Explain what should be different
- n (cancel): Cancel the commit process
```

## 🔧 Customization

Edit `git_workflow_assistant.json` to:
- Adjust validation thresholds
- Modify review focus areas
- Change commit message format
- Update timeout values
- Add custom validation rules

## 📊 Expected Output

The workflow generates structured output including:
- Validation results with specific warnings/errors
- Change goal analysis
- File-by-file review findings
- Generated commit message
- Final commit hash (if successful)

This workflow is ideal for teams using AI-assisted development who want to ensure code quality before commits.