#!/bin/bash

# LLM Code Editor - Run Script
# This script runs the LLM Code Editor agent

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
CONFIG_FILE="$SCRIPT_DIR/llm_code_editor.json"

# Check if DEEPINFRA_API_KEY is set
if [ -z "$DEEPINFRA_API_KEY" ]; then
    echo "Warning: DEEPINFRA_API_KEY environment variable is not set."
    echo "The embedding functionality will use placeholder implementations."
    echo ""
    echo "To use real embeddings, set your DeepInfra API key:"
    echo "export DEEPINFRA_API_KEY=\"your-api-key-here\""
    echo ""
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Ensure we're in a directory that looks like a code project
if [ ! -d ".git" ] && [ ! -f "go.mod" ] && [ ! -f "package.json" ] && [ ! -f "requirements.txt" ]; then
    echo "Warning: This doesn't appear to be a code repository."
    echo "The LLM Code Editor works best when run in the root of a code project."
    echo ""
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Change to project root and run the agent
cd "$PROJECT_DIR"

echo "Starting LLM Code Editor..."
echo "Project directory: $PROJECT_DIR"
echo "Working directory: $(pwd)"
echo "Config file: $CONFIG_FILE"
echo ""

exec ./agent-template process "$CONFIG_FILE" "$@"