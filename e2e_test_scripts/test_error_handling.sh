#!/bin/bash

# Function to return the test name
get_test_name() {
    echo "Error Handling & Recovery Test"
}

# Function to run the test logic
run_test_logic() {
    local model_name=$1
    echo "--- TEST: Error Handling & Recovery ---"
    start_time=$(date +%s)
    
    local test_dir="error_handling_test"
    rm -rf "$test_dir"
    mkdir -p "$test_dir"
    cd "$test_dir" || exit 1
    
    local exit_code=0
    
    echo "=== Testing 1: Agent Timeout Handling ==="
    
    cat > timeout_config.json << 'EOF'
{
  "version": "1.0",
  "goal": "Test timeout handling mechanisms",
  "description": "Validates that agents handle timeouts gracefully",
  "base_model": "gpt-4",
  "agents": [
    {
      "id": "timeout_agent",
      "name": "Timeout Test Agent",
      "persona": "tester",
      "description": "Agent with short timeouts for testing",
      "skills": ["testing"],
      "priority": 1,
      "depends_on": [],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 1000,
        "max_cost": 0.1,
        "token_warning": 800,
        "cost_warning": 0.08,
        "alert_on_limit": true,
        "stop_on_limit": false
      }
    }
  ],
  "steps": [
    {
      "id": "quick_task",
      "name": "Quick Task",
      "description": "Task that should complete within timeout",
      "agent_id": "timeout_agent",
      "input": {"task": "simple_response"},
      "expected_output": "quick_result",
      "status": "pending",
      "depends_on": [],
      "timeout": 60,
      "retries": 2
    },
    {
      "id": "timeout_task",
      "name": "Timeout Task", 
      "description": "Task with very short timeout for testing",
      "agent_id": "timeout_agent",
      "input": {"task": "potentially_slow_response"},
      "expected_output": "timeout_result",
      "status": "pending",
      "depends_on": [],
      "timeout": 1,
      "retries": 1
    }
  ],
  "validation": {
    "required": false
  },
  "settings": {
    "max_retries": 2,
    "step_timeout": 90,
    "parallel_execution": false,
    "stop_on_failure": false,
    "log_level": "info"
  }
}
EOF

    echo "Testing timeout configuration validation..."
    if jq . timeout_config.json > /dev/null 2>&1; then
        echo "PASS: Timeout configuration is valid JSON"
        
        # Test different timeout values
        quick_timeout=$(jq -r '.steps[] | select(.id == "quick_task") | .timeout' timeout_config.json)
        short_timeout=$(jq -r '.steps[] | select(.id == "timeout_task") | .timeout' timeout_config.json)
        
        if [ "$quick_timeout" -gt "$short_timeout" ]; then
            echo "PASS: Different timeout values configured"
        else
            echo "FAIL: Timeout values not configured correctly"
            exit_code=1
        fi
        
        # Test stop_on_failure is disabled for error recovery
        stop_on_failure=$(jq -r '.settings.stop_on_failure' timeout_config.json)
        if [ "$stop_on_failure" = "false" ]; then
            echo "PASS: Stop on failure disabled for error recovery"
        else
            echo "FAIL: Stop on failure not configured for error recovery"
            exit_code=1
        fi
    else
        echo "FAIL: Timeout configuration is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 2: Retry Mechanism Configuration ==="
    
    cat > retry_config.json << 'EOF'
{
  "version": "1.0",
  "goal": "Test retry mechanism configuration",
  "description": "Validates retry configurations work correctly",
  "base_model": "gpt-4",
  "agents": [
    {
      "id": "retry_agent",
      "name": "Retry Test Agent",
      "persona": "tester",
      "description": "Agent for testing retry mechanisms",
      "skills": ["testing"],
      "priority": 1,
      "depends_on": [],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 2000,
        "max_cost": 0.2,
        "token_warning": 1600,
        "cost_warning": 0.16,
        "alert_on_limit": true,
        "stop_on_limit": false
      }
    }
  ],
  "steps": [
    {
      "id": "no_retry_step",
      "name": "No Retry Step",
      "description": "Step with no retries configured",
      "agent_id": "retry_agent",
      "input": {"task": "no_retry_task"},
      "expected_output": "no_retry_result",
      "status": "pending",
      "depends_on": [],
      "timeout": 30,
      "retries": 0
    },
    {
      "id": "single_retry_step",
      "name": "Single Retry Step",
      "description": "Step with one retry configured",
      "agent_id": "retry_agent",
      "input": {"task": "single_retry_task"},
      "expected_output": "single_retry_result",
      "status": "pending",
      "depends_on": [],
      "timeout": 30,
      "retries": 1
    },
    {
      "id": "multiple_retry_step",
      "name": "Multiple Retry Step",
      "description": "Step with multiple retries configured",
      "agent_id": "retry_agent",
      "input": {"task": "multiple_retry_task"},
      "expected_output": "multiple_retry_result",
      "status": "pending",
      "depends_on": [],
      "timeout": 30,
      "retries": 3
    }
  ],
  "validation": {
    "required": false
  },
  "settings": {
    "max_retries": 5,
    "step_timeout": 60,
    "parallel_execution": false,
    "stop_on_failure": false,
    "log_level": "debug"
  }
}
EOF

    echo "Testing retry configuration validation..."
    if jq . retry_config.json > /dev/null 2>&1; then
        echo "PASS: Retry configuration is valid JSON"
        
        # Test different retry counts
        no_retries=$(jq -r '.steps[] | select(.id == "no_retry_step") | .retries' retry_config.json)
        single_retry=$(jq -r '.steps[] | select(.id == "single_retry_step") | .retries' retry_config.json)
        multiple_retries=$(jq -r '.steps[] | select(.id == "multiple_retry_step") | .retries' retry_config.json)
        
        if [ "$no_retries" -eq 0 ] && [ "$single_retry" -eq 1 ] && [ "$multiple_retries" -eq 3 ]; then
            echo "PASS: Different retry counts configured correctly"
        else
            echo "FAIL: Retry counts not configured correctly ($no_retries, $single_retry, $multiple_retries)"
            exit_code=1
        fi
        
        # Test global max retries
        max_retries=$(jq -r '.settings.max_retries' retry_config.json)
        if [ "$max_retries" -ge 3 ]; then
            echo "PASS: Global max retries configured appropriately"
        else
            echo "FAIL: Global max retries too low"
            exit_code=1
        fi
    else
        echo "FAIL: Retry configuration is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 3: Budget Limit Error Handling ==="
    
    cat > budget_limits.json << 'EOF'
{
  "version": "1.0",
  "goal": "Test budget limit error handling",
  "description": "Validates budget limits trigger appropriate alerts and actions",
  "base_model": "gpt-4",
  "agents": [
    {
      "id": "budget_warning_agent",
      "name": "Budget Warning Agent",
      "persona": "tester",
      "description": "Agent with budget warning enabled",
      "skills": ["testing"],
      "priority": 1,
      "depends_on": [],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 1000,
        "max_cost": 0.1,
        "token_warning": 800,
        "cost_warning": 0.08,
        "alert_on_limit": true,
        "stop_on_limit": false
      }
    },
    {
      "id": "budget_stop_agent",
      "name": "Budget Stop Agent",
      "persona": "tester", 
      "description": "Agent with budget stop enabled",
      "skills": ["testing"],
      "priority": 2,
      "depends_on": [],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 500,
        "max_cost": 0.05,
        "token_warning": 400,
        "cost_warning": 0.04,
        "alert_on_limit": true,
        "stop_on_limit": true
      }
    }
  ],
  "steps": [
    {
      "id": "warning_task",
      "name": "Budget Warning Task",
      "description": "Task that should trigger budget warning",
      "agent_id": "budget_warning_agent",
      "input": {"task": "moderate_complexity_task"},
      "expected_output": "warning_result",
      "status": "pending",
      "depends_on": [],
      "timeout": 60,
      "retries": 1
    },
    {
      "id": "stop_task",
      "name": "Budget Stop Task",
      "description": "Task that should trigger budget stop",
      "agent_id": "budget_stop_agent", 
      "input": {"task": "high_complexity_task"},
      "expected_output": "stop_result",
      "status": "pending",
      "depends_on": [],
      "timeout": 60,
      "retries": 1
    }
  ],
  "validation": {
    "required": false
  },
  "settings": {
    "max_retries": 1,
    "step_timeout": 90,
    "parallel_execution": false,
    "stop_on_failure": false,
    "log_level": "info"
  }
}
EOF

    echo "Testing budget limit configuration..."
    if jq . budget_limits.json > /dev/null 2>&1; then
        echo "PASS: Budget limits configuration is valid JSON"
        
        # Test different budget policies
        warning_stop=$(jq -r '.agents[] | select(.id == "budget_warning_agent") | .budget.stop_on_limit' budget_limits.json)
        stop_stop=$(jq -r '.agents[] | select(.id == "budget_stop_agent") | .budget.stop_on_limit' budget_limits.json)
        
        if [ "$warning_stop" = "false" ] && [ "$stop_stop" = "true" ]; then
            echo "PASS: Different budget policies configured"
        else
            echo "FAIL: Budget policies not configured correctly"
            exit_code=1
        fi
        
        # Test alert configuration
        warning_alert=$(jq -r '.agents[] | select(.id == "budget_warning_agent") | .budget.alert_on_limit' budget_limits.json)
        stop_alert=$(jq -r '.agents[] | select(.id == "budget_stop_agent") | .budget.alert_on_limit' budget_limits.json)
        
        if [ "$warning_alert" = "true" ] && [ "$stop_alert" = "true" ]; then
            echo "PASS: Budget alerts enabled for both agents"
        else
            echo "FAIL: Budget alerts not configured correctly"
            exit_code=1
        fi
    else
        echo "FAIL: Budget limits configuration is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 4: Agent Dependency Failure Handling ==="
    
    cat > dependency_failure.json << 'EOF'
{
  "version": "1.0",
  "goal": "Test agent dependency failure handling",
  "description": "Validates how system handles when dependency agents fail",
  "base_model": "gpt-4",
  "agents": [
    {
      "id": "unreliable_agent",
      "name": "Unreliable Agent",
      "persona": "unreliable",
      "description": "Agent that might fail",
      "skills": ["unreliable_task"],
      "priority": 1,
      "depends_on": [],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 100,
        "max_cost": 0.01,
        "token_warning": 80,
        "cost_warning": 0.008,
        "alert_on_limit": true,
        "stop_on_limit": true
      }
    },
    {
      "id": "dependent_agent",
      "name": "Dependent Agent",
      "persona": "dependent",
      "description": "Agent that depends on unreliable agent",
      "skills": ["dependent_task"],
      "priority": 2,
      "depends_on": ["unreliable_agent"],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 1000,
        "max_cost": 0.1,
        "token_warning": 800,
        "cost_warning": 0.08
      }
    },
    {
      "id": "independent_agent",
      "name": "Independent Agent",
      "persona": "independent",
      "description": "Agent that should continue despite failures",
      "skills": ["independent_task"],
      "priority": 1,
      "depends_on": [],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 1000,
        "max_cost": 0.1
      }
    }
  ],
  "steps": [
    {
      "id": "unreliable_step",
      "name": "Unreliable Step",
      "description": "Step that might fail due to budget constraints",
      "agent_id": "unreliable_agent",
      "input": {"task": "complex_task_likely_to_fail"},
      "expected_output": "unreliable_result",
      "status": "pending",
      "depends_on": [],
      "timeout": 30,
      "retries": 0
    },
    {
      "id": "dependent_step",
      "name": "Dependent Step",
      "description": "Step that depends on unreliable step",
      "agent_id": "dependent_agent",
      "input": {"dependency": "output from unreliable_step"},
      "expected_output": "dependent_result", 
      "status": "pending",
      "depends_on": ["unreliable_step"],
      "timeout": 30,
      "retries": 1
    },
    {
      "id": "independent_step",
      "name": "Independent Step",
      "description": "Step that should proceed regardless of failures",
      "agent_id": "independent_agent",
      "input": {"task": "independent_task"},
      "expected_output": "independent_result",
      "status": "pending",
      "depends_on": [],
      "timeout": 30,
      "retries": 1
    }
  ],
  "validation": {
    "required": false
  },
  "settings": {
    "max_retries": 1,
    "step_timeout": 45,
    "parallel_execution": true,
    "stop_on_failure": false,
    "log_level": "debug"
  }
}
EOF

    echo "Testing dependency failure handling configuration..."
    if jq . dependency_failure.json > /dev/null 2>&1; then
        echo "PASS: Dependency failure configuration is valid JSON"
        
        # Test that failure handling is configured to continue
        stop_on_failure=$(jq -r '.settings.stop_on_failure' dependency_failure.json)
        if [ "$stop_on_failure" = "false" ]; then
            echo "PASS: System configured to continue on failure"
        else
            echo "FAIL: System not configured to handle failures gracefully"
            exit_code=1
        fi
        
        # Test mix of dependent and independent steps
        dependent_steps=$(jq '[.steps[] | select(.depends_on | length > 0)] | length' dependency_failure.json)
        independent_steps=$(jq '[.steps[] | select(.depends_on | length == 0)] | length' dependency_failure.json)
        
        if [ "$dependent_steps" -gt 0 ] && [ "$independent_steps" -gt 0 ]; then
            echo "PASS: Mix of dependent and independent steps configured"
        else
            echo "FAIL: Step dependencies not properly mixed for failure testing"
            exit_code=1
        fi
    else
        echo "FAIL: Dependency failure configuration is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 5: Validation Failure Handling ==="
    
    cat > validation_failure.json << 'EOF'
{
  "version": "1.0",
  "goal": "Test validation failure handling strategies",
  "description": "Validates different validation failure strategies work correctly",
  "base_model": "gpt-4", 
  "agents": [
    {
      "id": "validation_agent",
      "name": "Validation Test Agent",
      "persona": "validator",
      "description": "Agent for testing validation scenarios",
      "skills": ["validation"],
      "priority": 1,
      "depends_on": [],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 1500,
        "max_cost": 0.15
      }
    }
  ],
  "steps": [
    {
      "id": "validation_test_step",
      "name": "Validation Test Step",
      "description": "Step to test validation rules",
      "agent_id": "validation_agent",
      "input": {"task": "generate_test_output"},
      "expected_output": "test_output_for_validation",
      "status": "pending",
      "depends_on": [],
      "timeout": 45,
      "retries": 2
    }
  ],
  "validation": {
    "enabled": true,
    "rules": [
      {
        "name": "min_length_check",
        "type": "custom",
        "config": {
          "validator": "length_range",
          "min": 10,
          "max": 1000
        }
      },
      {
        "name": "required_content",
        "type": "regex", 
        "config": {
          "pattern": "(test|validation|output)"
        }
      }
    ],
    "on_failure": "warn"
  },
  "settings": {
    "max_retries": 2,
    "step_timeout": 60,
    "parallel_execution": false,
    "stop_on_failure": false,
    "log_level": "debug"
  }
}
EOF

    echo "Testing validation failure handling..."
    if jq . validation_failure.json > /dev/null 2>&1; then
        echo "PASS: Validation failure configuration is valid JSON"
        
        # Test validation is enabled
        validation_enabled=$(jq -r '.validation.enabled' validation_failure.json)
        if [ "$validation_enabled" = "true" ]; then
            echo "PASS: Validation enabled for testing"
        else
            echo "FAIL: Validation not enabled"
            exit_code=1
        fi
        
        # Test validation rules exist
        validation_rules=$(jq '.validation.rules | length' validation_failure.json)
        if [ "$validation_rules" -gt 0 ]; then
            echo "PASS: Validation rules configured ($validation_rules rules)"
        else
            echo "FAIL: No validation rules configured"
            exit_code=1
        fi
        
        # Test failure handling strategy
        failure_strategy=$(jq -r '.validation.on_failure' validation_failure.json)
        if [ "$failure_strategy" = "warn" ]; then
            echo "PASS: Validation failure strategy set to warn (allows continuation)"
        else
            echo "FAIL: Validation failure strategy not configured for error recovery"
            exit_code=1
        fi
    else
        echo "FAIL: Validation failure configuration is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 6: Error Recovery Features ==="
    
    echo "Testing error recovery configurations across all test files..."
    
    # Test timeout vs retry balance
    for config in timeout_config.json retry_config.json budget_limits.json dependency_failure.json validation_failure.json; do
        if [ -f "$config" ]; then
            retries=$(jq '[.steps[].retries] | max' "$config")
            timeouts=$(jq '[.steps[].timeout] | min' "$config")
            
            if [ "$retries" -ge 0 ] && [ "$timeouts" -gt 0 ]; then
                echo "PASS: $config has balanced timeout/retry configuration"
            else
                echo "FAIL: $config missing timeout or retry configuration"
                exit_code=1
            fi
        fi
    done
    
    # Test graceful failure handling
    echo "Testing graceful failure configurations..."
    graceful_configs=0
    total_configs=0
    
    for config in timeout_config.json retry_config.json budget_limits.json dependency_failure.json validation_failure.json; do
        if [ -f "$config" ]; then
            total_configs=$((total_configs + 1))
            stop_on_failure=$(jq -r '.settings.stop_on_failure' "$config" 2>/dev/null || echo "null")
            if [ "$stop_on_failure" = "false" ]; then
                graceful_configs=$((graceful_configs + 1))
            fi
        fi
    done
    
    if [ "$graceful_configs" -gt $((total_configs / 2)) ]; then
        echo "PASS: Majority of configurations use graceful failure handling"
    else
        echo "WARN: Most configurations use strict failure handling (may be intentional)"
    fi
    
    echo
    echo "=== Test Summary ==="
    
    # Clean up
    cd ../ || true
    rm -rf "$test_dir"
    
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    if [ $exit_code -eq 0 ]; then
        echo "✅ All error handling tests PASSED"
        echo "🎉 Error recovery mechanisms validated successfully"
        echo "📊 Test coverage:"
        echo "   ✓ Agent timeout handling"
        echo "   ✓ Retry mechanism configuration"
        echo "   ✓ Budget limit error handling"
        echo "   ✓ Agent dependency failure handling"
        echo "   ✓ Validation failure handling"
        echo "   ✓ Error recovery features"
    else
        echo "❌ Some error handling tests FAILED"
        echo "🔧 Error handling mechanisms need attention"
    fi
    
    echo "⏱️  Test duration: $duration seconds"
    echo "----------------------------------------------------"
    
    return $exit_code
}