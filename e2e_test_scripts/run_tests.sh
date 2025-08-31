#!/bin/bash

# Generic Agent Framework E2E Test Runner
echo "🚀 Starting Generic Agent Framework E2E Tests"
echo "=============================================="

# Default model for testing (can be overridden)
DEFAULT_MODEL="gpt-4"
MODEL="${1:-$DEFAULT_MODEL}"

echo "Using model: $MODEL"
echo "Test timestamp: $(date)"
echo

# Build the project first
echo "🔨 Building project..."
cd /Users/alanp/dev/personal/agent-template
if go build -o agent-template .; then
    echo "✅ Build successful"
    # Also create a symlink for backward compatibility with tests
    ln -sf agent-template generic-agent
    echo "✅ Created generic-agent symlink for test compatibility"
else
    echo "❌ Build failed - aborting tests"
    exit 1
fi

echo

# Define all test scripts
TEST_SCRIPTS=(
    "test_generic_agent_framework.sh"
    "test_llm_provider_integration.sh"
    "test_workflow_execution.sh"
    "test_error_handling.sh"
    "test_security_validation.sh"
)

# Run all tests
echo "🧪 Running Generic Agent Framework E2E Test Suite..."
echo "Test suite includes ${#TEST_SCRIPTS[@]} comprehensive tests"
echo

cd e2e_test_scripts

PASSED_TESTS=0
FAILED_TESTS=0
TOTAL_TESTS=${#TEST_SCRIPTS[@]}

for test_script in "${TEST_SCRIPTS[@]}"; do
    if [ -f "$test_script" ]; then
        echo "▶️  Running $test_script..."
        echo "----------------------------------------"
        
        if ./"$test_script" "$MODEL"; then
            echo "✅ $test_script PASSED"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo "❌ $test_script FAILED"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
        
        echo "----------------------------------------"
        echo
    else
        echo "⚠️  Test script $test_script not found"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
done

# Final results
echo "🏁 TEST SUITE SUMMARY"
echo "=============================================="
echo "📊 Total tests: $TOTAL_TESTS"
echo "✅ Passed: $PASSED_TESTS"
echo "❌ Failed: $FAILED_TESTS"
echo "📈 Success rate: $((PASSED_TESTS * 100 / TOTAL_TESTS))%"
echo

if [ $FAILED_TESTS -eq 0 ]; then
    echo "🎉 ALL TESTS PASSED!"
    echo "✅ Generic agent framework is working correctly"
    echo "🚀 Framework ready for deployment"
    exit 0
else
    echo "⚠️  SOME TESTS FAILED"
    echo "🔧 Framework needs attention in failing areas"
    echo "📝 Review failed test output above for details"
    exit 1
fi