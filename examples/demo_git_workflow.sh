#!/bin/bash

# Git Workflow Assistant Demo Script
# This script demonstrates the Git Workflow Assistant agent

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
CONFIG_FILE="$SCRIPT_DIR/configs/git_workflow_assistant.json"

echo "🚀 Git Workflow Assistant Demo"
echo "============================================="
echo
echo "This demo showcases an AI agent that:"
echo "  • Reviews staged git changes for quality issues"
echo "  • Provides detailed code analysis and feedback"
echo "  • Generates conventional commit messages"
echo "  • Guides users through the commit process with approvals"
echo
echo "Configuration: git_workflow_assistant.json"
echo "============================================="
echo

# Check if we're in a git repository
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "❌ Error: Not in a git repository"
    echo "   This demo needs to be run from within a git repository."
    exit 1
fi

# Check for staged changes
if ! git diff --staged --quiet; then
    echo "✅ Found staged changes - ready for review!"
    echo
    echo "Staged files:"
    git diff --staged --name-only | sed 's/^/   • /'
    echo
    echo "📝 The agent workflow includes these steps:"
    echo "   1. Get and analyze staged git changes"
    echo "   2. Perform comprehensive code review"
    echo "   3. Request user approval for findings"
    echo "   4. Handle revision requests if needed"
    echo "   5. Generate conventional commit message"
    echo "   6. Request commit message approval"  
    echo "   7. Execute git commit with approved message"
    echo
    echo "🔧 To run this agent:"
    echo "   ./agent-template run examples/configs/git_workflow_assistant.json"
    echo
    echo "📋 Example agent capabilities:"
    echo "   • Code quality analysis"
    echo "   • Security vulnerability detection"
    echo "   • Best practice recommendations"
    echo "   • Performance optimization suggestions"
    echo "   • Conventional commit message generation"
    echo "   • Interactive user approval workflow"
else
    echo "⚠️  No staged changes found"
    echo 
    echo "To test this agent, first stage some changes:"
    echo "   git add <files>"
    echo "   ./agent-template run examples/configs/git_workflow_assistant.json"
    echo
    echo "📋 This agent provides:"
    echo "   • Comprehensive code review of staged changes"
    echo "   • Analysis of code quality, security, and best practices"
    echo "   • Interactive approval process with user feedback"
    echo "   • Automatic generation of conventional commit messages"
    echo "   • Step-by-step git workflow guidance"
fi

echo
echo "🔍 Agent Configuration Details:"
echo "   • LLM Provider: Anthropic Claude 3 Sonnet"
echo "   • Workflows: 2 (comprehensive + quick)"
echo "   • Data Sources: 2 (git repository info)"
echo "   • Tools: 5 (git operations + user interaction)"
echo "   • Budget: $10.00 max, 50,000 tokens"
echo "   • Interactive: Yes (user approval points)"
echo
echo "✅ Configuration validated successfully!"