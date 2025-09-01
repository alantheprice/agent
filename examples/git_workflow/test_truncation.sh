#!/bin/bash

# Test script to demonstrate the git diff truncation functionality
# This script creates a large file change to test the truncation logic

echo "Creating a test scenario with large file changes..."

# Create a test file with many lines
TEST_FILE="large_test_file.txt"
echo "Creating $TEST_FILE with 100 lines..."

for i in {1..100}; do
    echo "This is line $i of the test file" >> "$TEST_FILE"
done

# Stage the file
git add "$TEST_FILE"

echo ""
echo "Now running the truncation logic..."
echo "----------------------------------------"

# Run the same truncation logic from the workflow
MAX_LINES=50
TEMP_DIFF=$(mktemp)
git diff --staged > "$TEMP_DIFF"

# Get list of modified files
FILES=$(git diff --staged --name-only)

echo "# Git Diff with Large File Truncation"
echo "# Files with >$MAX_LINES changes are truncated"
echo ""

for file in $FILES; do
  CHANGE_COUNT=$(git diff --staged "$file" | grep -E "^[+-]" | grep -v "^[+-]{3}" | wc -l)
  
  if [ "$CHANGE_COUNT" -gt "$MAX_LINES" ]; then
    echo "diff --git a/$file b/$file"
    echo "# TRUNCATED: File has $CHANGE_COUNT changes (showing summary only)"
    echo "# File type: $(file -b "$file" 2>/dev/null || echo "unknown")"
    
    # Show file header info
    git diff --staged "$file" | head -5
    
    echo "..."
    echo "# [TRUNCATED: Showing first 10 and last 10 changes out of $CHANGE_COUNT total]"
    echo "..."
    
    # Show first 10 changes
    git diff --staged "$file" | grep -E "^[+-]" | grep -v "^[+-]{3}" | head -10
    
    echo "..."
    echo "# [... $(($CHANGE_COUNT - 20)) lines omitted ...]"
    echo "..."
    
    # Show last 10 changes  
    git diff --staged "$file" | grep -E "^[+-]" | grep -v "^[+-]{3}" | tail -10
    
    echo ""
  else
    # Show full diff for files with reasonable change count
    git diff --staged "$file"
    echo ""
  fi
done

rm "$TEMP_DIFF"

echo "----------------------------------------"
echo "Test completed. Cleaning up..."

# Clean up
git reset HEAD "$TEST_FILE"
rm -f "$TEST_FILE"

echo "Removed $TEST_FILE and unstaged changes."