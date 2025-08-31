# Git Workflow Assistant Agent

An AI-powered agent that provides comprehensive code review, generates commit messages, and guides users through the git commit process with interactive approvals.

## 🚀 Features

### Core Capabilities
- **Comprehensive Code Review**: Analyzes staged changes for quality issues, security concerns, and best practices
- **Interactive Workflow**: Guides users through approval steps with y/n/r options
- **Commit Message Generation**: Creates conventional commit messages following best practices
- **Multi-Step Validation**: Ensures code quality before allowing commits
- **Expert Analysis**: Provides specific, actionable feedback on code improvements

### Workflow Steps

#### 1. **Staged Changes Analysis**
```
get_staged_changes → analyze_staged_changes
```
- Retrieves current staged git changes
- Performs comprehensive code quality analysis
- Identifies potential bugs, security issues, performance problems
- Provides line-by-line feedback where applicable

#### 2. **User Review & Approval**
```
user_review_approval → handle_revision_request (optional)
```
- Presents analysis findings to user
- Allows approval (y), cancellation (n), or revision request (r)
- Provides additional detailed suggestions if revision requested

#### 3. **Commit Message Generation**
```
generate_commit_message → commit_message_approval
```
- Creates conventional commit messages (type: description format)
- Follows 50-character subject line limit
- Includes detailed body with reasoning and impact
- Requests user approval for generated message

#### 4. **Commit Execution**
```
execute_commit
```
- Executes git commit with approved message
- Provides confirmation with commit hash
- Ensures successful completion

## 🛠️ Usage

### Basic Usage
```bash
# Stage your changes
git add <files>

# Run the git workflow agent
./agent-template run examples/configs/git_workflow_assistant.json
```

### Demo Script
```bash
# Run the demo to see capabilities
./examples/demo_git_workflow.sh
```

### Workflow Selection

The agent includes two workflows:

1. **Comprehensive Git Workflow** (`comprehensive_git_workflow`)
   - Full code review with detailed analysis
   - Multiple user interaction points
   - Revision handling
   - Complete commit message generation

2. **Quick Commit Review** (`quick_commit_review`)
   - Simplified workflow for rapid commits
   - Basic analysis and commit message generation
   - Single approval step

## ⚙️ Configuration

### LLM Settings
- **Provider**: Anthropic Claude 3 Sonnet
- **Temperature**: 0.3 (focused, consistent responses)
- **Max Tokens**: 4,000 per request
- **System Prompt**: Expert software engineer and code reviewer

### Data Sources
- **Git Repository Info**: Reads `.git/HEAD` for current branch information
- **Project Config**: Accesses `.gitignore` for project context

### Tools Available
- `git_diff`: Get git diff for staged changes
- `git_status`: Check repository status  
- `git_commit`: Execute git commit with message
- `ask_user`: Interactive user input and approval
- `validate_input`: Validate user response format

### Budget Controls
- **Max Cost**: $10.00 per execution
- **Max Tokens**: 50,000 tokens total
- **Warning Threshold**: 80% of budget

### Security
- **Allowed Paths**: `.git/`, `./` (current directory)
- **Max File Size**: 1MB for analysis

## 📋 Example Analysis Output

The agent provides detailed analysis including:

### Code Quality Issues
- Unused variables or imports
- Code complexity warnings
- Naming convention violations
- Documentation gaps

### Security Concerns
- Potential vulnerabilities
- Input validation issues
- Authentication/authorization problems
- Sensitive data exposure risks

### Best Practice Violations
- Framework-specific anti-patterns
- Performance optimization opportunities
- Error handling improvements
- Testing coverage suggestions

### Performance Implications
- Algorithm efficiency concerns
- Memory usage optimization
- Database query performance
- Network request optimization

## 🔧 Customization

### Modifying Prompts
Edit the workflow step prompts in `git_workflow_assistant.json` to:
- Change analysis focus areas
- Adjust commit message format
- Modify user interaction text
- Add project-specific requirements

### Adjusting Analysis Depth
Configure the agent behavior by modifying:
- `temperature`: Lower for more consistent, higher for creative analysis
- `max_tokens`: Adjust for longer/shorter analysis
- `timeout`: Modify step time limits

### Adding Custom Validation
Extend the security section to add:
- File type restrictions
- Directory access limits
- Command execution constraints
- Custom validation rules

## 🎯 Best Practices

### For Effective Code Review
1. **Stage Meaningful Changes**: Group related changes together
2. **Provide Context**: Ensure commit represents logical unit of work
3. **Review Feedback**: Take time to read and consider agent suggestions
4. **Iterate on Quality**: Use revision requests for complex changes

### For Commit Messages
- The agent generates conventional commit format: `type(scope): description`
- Common types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
- Descriptions are kept under 50 characters
- Body provides detailed explanation when needed

### For Workflow Integration
- Use as part of regular development process
- Combine with existing CI/CD pipelines
- Establish team conventions for agent feedback handling
- Document project-specific customizations

## 🚨 Limitations

### Current Framework Limitations
- Command-based data sources not yet supported (planned feature)
- Advanced conditional logic simplified for compatibility
- User input validation is basic (y/n/r patterns)
- Git command execution through tools (not direct shell access)

### Recommendations for Production Use
- Test thoroughly in development environment
- Establish clear team guidelines for agent feedback
- Consider integration with existing code review processes
- Monitor token usage and costs

## 📈 Future Enhancements

### Planned Features
- Direct git command execution
- Advanced conditional workflow logic
- Integration with GitHub/GitLab APIs
- Custom validation rule engine
- Team-specific configuration templates

### Extension Opportunities
- Integration with linting tools
- Automated test execution
- Code coverage analysis
- Security vulnerability scanning
- Performance benchmark integration

---

This Git Workflow Assistant demonstrates the power of the generic agent template system for creating sophisticated, interactive development workflows.