#!/bin/bash

# Git Workflow Assistant - Run Script
# This script runs the Git Workflow Assistant agent

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
CONFIG_FILE="$SCRIPT_DIR/git_workflow_assistant.json"

cd "$PROJECT_DIR"
exec ./bin/agent-template process "$CONFIG_FILE" "$@"