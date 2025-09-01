#!/bin/bash

# Enhanced LLM Code Editor - Run Script
# This script runs the enhanced LLM Code Editor agent with comprehensive validation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
CONFIG_FILE="$SCRIPT_DIR/optimized_code_editor.json"

# Check if DEEPINFRA_API_KEY is set
if [ -z "$DEEPINFRA_API_KEY" ]; then
    echo "❌ Error: DEEPINFRA_API_KEY environment variable is not set."
    echo ""
    echo "The enhanced code editor requires DeepInfra API access for:"
    echo "  • LLM processing (DeepSeek V3.1 model)"
    echo "  • Semantic embeddings (sentence-transformers model)"
    echo ""
    echo "To set your API key:"
    echo "  export DEEPINFRA_API_KEY=\"your-api-key-here\""
    echo ""
    echo "Get your API key at: https://deepinfra.com/dash/api_keys"
    exit 1
fi

# Validate we're in a code repository
if [ ! -d ".git" ] && [ ! -f "go.mod" ] && [ ! -f "package.json" ] && [ ! -f "requirements.txt" ] && [ ! -f "Cargo.toml" ]; then
    echo "⚠️  Warning: This doesn't appear to be a code repository."
    echo "   The Enhanced Code Editor works best in the root of a code project."
    echo ""
    echo "   Current directory: $(pwd)"
    echo "   Looking for: .git/, go.mod, package.json, requirements.txt, or Cargo.toml"
    echo ""
    read -p "   Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check available tools
echo "🔧 Checking available development tools..."
AVAILABLE_TOOLS=""

if command -v go >/dev/null 2>&1; then
    AVAILABLE_TOOLS+="Go "
    echo "  ✅ Go $(go version | cut -d' ' -f3)"
fi

if command -v node >/dev/null 2>&1 && command -v npm >/dev/null 2>&1; then
    AVAILABLE_TOOLS+="Node.js "  
    echo "  ✅ Node.js $(node --version) + npm $(npm --version)"
fi

if command -v python3 >/dev/null 2>&1; then
    AVAILABLE_TOOLS+="Python "
    echo "  ✅ Python $(python3 --version)"
fi

if command -v python >/dev/null 2>&1 && ! command -v python3 >/dev/null 2>&1; then
    AVAILABLE_TOOLS+="Python "
    echo "  ✅ Python $(python --version)"
fi

if [ -z "$AVAILABLE_TOOLS" ]; then
    echo "  ⚠️  No development tools detected"
    echo "     Validation features will be limited"
else
    echo "  🎯 Detected: $AVAILABLE_TOOLS"
fi

echo ""
echo "🚀 Starting Enhanced LLM Code Editor..."
echo "   Project directory: $PROJECT_DIR" 
echo "   Working directory: $(pwd)"
echo "   Config file: $CONFIG_FILE"
echo "   API Key: ${DEEPINFRA_API_KEY:0:8}..."
echo ""

# Change to project root and run the agent
cd "$PROJECT_DIR"
exec ./agent process "$CONFIG_FILE" "$@"