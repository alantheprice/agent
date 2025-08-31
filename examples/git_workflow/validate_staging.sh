#!/bin/bash

# Staging Validation Script
echo '=== STAGING VALIDATION REPORT ==='
echo

# Get staged files
STAGED_FILES=$(git diff --staged --name-only)
FILE_COUNT=$(echo "$STAGED_FILES" | grep -v '^$' | wc -l | tr -d ' ')

if [ -z "$STAGED_FILES" ] || [ "$FILE_COUNT" -eq 0 ]; then
  echo '❌ No staged changes found'
  echo 'RESULT: NO_CHANGES'
  exit 0
fi

echo "📁 Staged files: $FILE_COUNT"
echo

# Initialize status tracking
WARNINGS=''
ERRORS=''
STATUS='OK'

# Check for large number of files
if [ "$FILE_COUNT" -gt 50 ]; then
  ERRORS="$ERRORS\n🚨 CRITICAL: $FILE_COUNT files staged (>50 is unusual for code review)"
  STATUS='ERROR'
elif [ "$FILE_COUNT" -gt 20 ]; then
  WARNINGS="$WARNINGS\n⚠️  WARNING: $FILE_COUNT files staged (>20 files - consider smaller commits)"
  [ "$STATUS" != 'ERROR' ] && STATUS='WARNING'
fi

# Check individual file sizes and types
echo '📄 File Analysis:'
LARGE_FILES=''
BINARY_FILES=''
GENERATED_FILES=''
TOTAL_SIZE=0

while IFS= read -r file; do
  [ -z "$file" ] && continue
  if [ ! -f "$file" ]; then
    echo "   ⚠️  File not found: $file (deleted)"
    continue
  fi
  
  # Get file size (cross-platform)
  if command -v stat >/dev/null 2>&1; then
    if stat -f%z "$file" >/dev/null 2>&1; then
      SIZE=$(stat -f%z "$file")  # macOS
    else
      SIZE=$(stat -c%s "$file" 2>/dev/null || echo 0)  # Linux
    fi
  else
    SIZE=0
  fi
  
  TOTAL_SIZE=$((TOTAL_SIZE + SIZE))
  
  # Format size for display
  if [ "$SIZE" -gt 1073741824 ]; then  # 1GB
    SIZE_DISPLAY="$(echo "scale=1; $SIZE/1073741824" | bc 2>/dev/null || echo "$((SIZE/1073741824))")GB"
  elif [ "$SIZE" -gt 1048576 ]; then  # 1MB
    SIZE_DISPLAY="$(echo "scale=1; $SIZE/1048576" | bc 2>/dev/null || echo "$((SIZE/1048576))")MB"
  elif [ "$SIZE" -gt 1024 ]; then  # 1KB
    SIZE_DISPLAY="$(echo "scale=1; $SIZE/1024" | bc 2>/dev/null || echo "$((SIZE/1024))")KB"
  else
    SIZE_DISPLAY="${SIZE}B"
  fi
  
  # Check if binary (simple check)
  IS_BINARY=false
  if command -v file >/dev/null 2>&1; then
    file "$file" | grep -q -E 'binary|executable|archive|image|audio|video' && IS_BINARY=true
  else
    # Fallback: check for null bytes in first 1024 bytes
    head -c 1024 "$file" | grep -q '\0' 2>/dev/null && IS_BINARY=true
  fi
  
  if [ "$IS_BINARY" = true ]; then
    BINARY_FILES="$BINARY_FILES\n   📦 $file ($SIZE_DISPLAY)"
    if [ "$SIZE" -gt 1048576 ]; then  # 1MB
      ERRORS="$ERRORS\n🚨 CRITICAL: Binary file $file is $SIZE_DISPLAY (should not be in git)"
      STATUS='ERROR'
    fi
  fi
  
  # Check for generated/build files
  case "$file" in
    *.min.js|*.min.css|*.map|*.log|*.tmp|*temp*|*cache*|node_modules/*|dist/*|build/*|target/*|*.class|*.jar|*.war|*.pyc|*.pyo|__pycache__/*|*.o|*.so|*.dylib|*.exe|*.dll|coverage/*|.coverage|*.lcov)
      GENERATED_FILES="$GENERATED_FILES\n   🔧 $file ($SIZE_DISPLAY - likely generated)"
      WARNINGS="$WARNINGS\n⚠️  WARNING: Staging generated/build file: $file"
      [ "$STATUS" = 'OK' ] && STATUS='WARNING'
      ;;
  esac
  
  # Check for large source files
  if [ "$SIZE" -gt 10485760 ]; then  # 10MB
    ERRORS="$ERRORS\n🚨 CRITICAL: File $file is $SIZE_DISPLAY (>10MB - too large for code review)"
    LARGE_FILES="$LARGE_FILES\n   📏 $file ($SIZE_DISPLAY)"
    STATUS='ERROR'
  elif [ "$SIZE" -gt 1048576 ]; then  # 1MB
    WARNINGS="$WARNINGS\n⚠️  WARNING: File $file is $SIZE_DISPLAY (>1MB - consider if necessary)"
    LARGE_FILES="$LARGE_FILES\n   📏 $file ($SIZE_DISPLAY)"
    [ "$STATUS" = 'OK' ] && STATUS='WARNING'
  else
    echo "   ✓ $file ($SIZE_DISPLAY)"
  fi
done <<< "$STAGED_FILES"

echo

# Format total size
if [ "$TOTAL_SIZE" -gt 1073741824 ]; then  # 1GB
  TOTAL_DISPLAY="$(echo "scale=1; $TOTAL_SIZE/1073741824" | bc 2>/dev/null || echo "$((TOTAL_SIZE/1073741824))")GB"
elif [ "$TOTAL_SIZE" -gt 1048576 ]; then  # 1MB
  TOTAL_DISPLAY="$(echo "scale=1; $TOTAL_SIZE/1048576" | bc 2>/dev/null || echo "$((TOTAL_SIZE/1048576))")MB"
elif [ "$TOTAL_SIZE" -gt 1024 ]; then  # 1KB
  TOTAL_DISPLAY="$(echo "scale=1; $TOTAL_SIZE/1024" | bc 2>/dev/null || echo "$((TOTAL_SIZE/1024))")KB"
else
  TOTAL_DISPLAY="${TOTAL_SIZE}B"
fi

echo "📊 Total staged size: $TOTAL_DISPLAY"

# Check total diff size
DIFF_LINES=$(git diff --staged | wc -l | tr -d ' ')
echo "📝 Total diff lines: $DIFF_LINES"

if [ "$DIFF_LINES" -gt 10000 ]; then
  ERRORS="$ERRORS\n🚨 CRITICAL: Diff is $DIFF_LINES lines (>10,000 - too large for effective review)"
  STATUS='ERROR'
elif [ "$DIFF_LINES" -gt 2000 ]; then
  WARNINGS="$WARNINGS\n⚠️  WARNING: Diff is $DIFF_LINES lines (>2,000 - consider smaller commits)"
  [ "$STATUS" = 'OK' ] && STATUS='WARNING'
fi

# Print summaries
if [ -n "$LARGE_FILES" ]; then
  echo -e "\n📏 Large Files:$LARGE_FILES"
fi

if [ -n "$BINARY_FILES" ]; then
  echo -e "\n📦 Binary Files:$BINARY_FILES"
fi

if [ -n "$GENERATED_FILES" ]; then
  echo -e "\n🔧 Generated/Build Files:$GENERATED_FILES"
fi

# Print warnings and errors
if [ -n "$WARNINGS" ]; then
  echo -e "\n⚠️  WARNINGS:$WARNINGS"
fi

if [ -n "$ERRORS" ]; then
  echo -e "\n🚨 CRITICAL ISSUES:$ERRORS"
fi

echo
echo "📋 VALIDATION STATUS: $STATUS"
echo 'RESULT: '$STATUS
echo '=== END VALIDATION REPORT ==='