#!/bin/bash

# Git Workflow Assistant - Run Script
# This script runs the Git Workflow Assistant agent

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
CONFIG_FILE="$SCRIPT_DIR/git_workflow_assistant.json"

echo "🚀 Starting Git Workflow Assistant..."
echo "============================================="
echo "Configuration: $CONFIG_FILE"
echo

# Check if we're in a git repository
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "❌ Error: Not in a git repository"
    echo "   This agent needs to be run from within a git repository."
    exit 1
fi

# Check for staged changes
if ! git diff --staged --quiet; then
    echo "✅ Found staged changes - ready for review!"
    echo
    echo "Staged files:"
    git diff --staged --name-only | sed 's/^/   • /'
    echo
else
    echo "⚠️  No staged changes found"
    echo "   Please stage some changes first:"
    echo "   git add <files>"
    echo
    exit 1
fi

# Run the agent
echo "🔧 Running Git Workflow Assistant..."
echo

cd "$PROJECT_DIR"
exec ./bin/agent-template run --config "$CONFIG_FILE" "$@"