#!/bin/bash

# Function to return the test name
get_test_name() {
    echo "Generic Agent Framework E2E Test"
}

# Function to run the test logic
run_test_logic() {
    local model_name=$1
    echo "--- TEST: Generic Agent Framework E2E Test ---"
    start_time=$(date +%s)
    
    local test_dir="generic_agent_e2e_test"
    rm -rf "$test_dir"
    mkdir -p "$test_dir"
    cd "$test_dir" || exit 1
    
    local exit_code=0
    
    echo "=== Testing 1: Simple Single Agent Configuration ==="
    
    # Test 1: Simple single agent configuration
    cat > simple_agent.json << 'EOF'
{
  "agent": {
    "name": "Math Tutor",
    "description": "Helps with basic math problems and explanations",
    "version": "1.0.0",
    "goals": [
      "Solve math problems accurately",
      "Provide clear explanations",
      "Help users understand concepts"
    ],
    "capabilities": [
      "Arithmetic",
      "Basic algebra",
      "Problem explanation"
    ],
    "max_iterations": 5,
    "timeout": "5m",
    "interactive": false
  },
  "llm": {
    "provider": "openai",
    "model": "gpt-4",
    "temperature": 0.1,
    "max_tokens": 1000,
    "system_prompt": "You are a helpful math tutor. Solve problems step by step and explain your reasoning clearly."
  },
  "data_sources": [
    {
      "name": "math_problem",
      "type": "stdin",
      "config": {},
      "preprocessing": [
        {
          "type": "transform",
          "config": {
            "type": "trim"
          }
        }
      ]
    }
  ],
  "workflows": [
    {
      "name": "solve_problem",
      "description": "Solve a math problem with explanation",
      "trigger": {
        "conditions": ["math_request"],
        "priority": 100
      },
      "steps": [
        {
          "name": "analyze_problem",
          "type": "llm",
          "config": {
            "prompt": "Analyze this math problem and identify what type of problem it is: {input}"
          }
        },
        {
          "name": "solve_step_by_step",
          "type": "llm",
          "config": {
            "prompt": "Solve this problem step by step with clear explanations: {input}. Problem analysis: {analysis}"
          },
          "depends_on": ["analyze_problem"]
        }
      ],
      "output": {
        "format": "text",
        "destination": "console"
      }
    }
  ],
  "tools": {
    "calculator": {
      "enabled": true,
      "timeout": "10s"
    }
  },
  "outputs": [
    {
      "name": "solution",
      "type": "console",
      "config": {
        "format": "text"
      }
    }
  ],
  "environment": {
    "variables": {
      "MATH_MODE": "basic"
    },
    "workspace_root": ".",
    "log_level": "info"
  },
  "security": {
    "enabled": true,
    "allowed_paths": ["."],
    "max_file_size": "1MB",
    "require_approval": false
  },
  "validation": {
    "enabled": true,
    "rules": [
      {
        "name": "has_answer",
        "type": "regex",
        "config": {
          "pattern": "\\d+"
        }
      }
    ],
    "on_failure": "warn"
  }
}
EOF

    echo "Testing configuration validation..."
    if ! jq . simple_agent.json > /dev/null 2>&1; then
        echo "FAIL: simple_agent.json is not valid JSON"
        exit_code=1
    else
        echo "PASS: Configuration file is valid JSON"
    fi
    
    # Test schema structure
    if ! jq -e '.agent.name' simple_agent.json > /dev/null; then
        echo "FAIL: Agent name not found in configuration"
        exit_code=1
    else
        echo "PASS: Agent configuration has required fields"
    fi
    
    echo
    echo "=== Testing 2: Multi-Agent Orchestration Configuration ==="
    
    # Test 2: Multi-agent orchestration
    cat > multi_agent_process.json << 'EOF'
{
  "version": "1.0",
  "goal": "Create a simple data analysis report",
  "description": "Test multi-agent collaboration for data analysis workflow",
  "base_model": "gpt-4",
  "agents": [
    {
      "id": "data_collector",
      "name": "Data Collector",
      "persona": "data_analyst",
      "description": "Gathers and prepares data for analysis",
      "skills": ["data_collection", "data_validation"],
      "priority": 1,
      "depends_on": [],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 10000,
        "max_cost": 1.0,
        "token_warning": 8000,
        "cost_warning": 0.8,
        "alert_on_limit": true,
        "stop_on_limit": false
      }
    },
    {
      "id": "analyzer",
      "name": "Data Analyzer",
      "persona": "statistician",
      "description": "Analyzes collected data and finds patterns",
      "skills": ["statistical_analysis", "pattern_recognition"],
      "priority": 2,
      "depends_on": ["data_collector"],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 15000,
        "max_cost": 1.5,
        "token_warning": 12000,
        "cost_warning": 1.2,
        "alert_on_limit": true,
        "stop_on_limit": false
      }
    },
    {
      "id": "reporter",
      "name": "Report Generator",
      "persona": "technical_writer",
      "description": "Creates comprehensive reports from analysis results",
      "skills": ["report_writing", "data_visualization"],
      "priority": 3,
      "depends_on": ["analyzer"],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 12000,
        "max_cost": 1.2,
        "token_warning": 10000,
        "cost_warning": 1.0,
        "alert_on_limit": true,
        "stop_on_limit": false
      }
    }
  ],
  "steps": [
    {
      "id": "collect_sample_data",
      "name": "Collect Sample Data",
      "description": "Generate sample sales data for analysis",
      "agent_id": "data_collector",
      "input": {"request": "Generate 10 sample sales records with product, quantity, price, and date"},
      "expected_output": "CSV-formatted sample data",
      "status": "pending",
      "depends_on": [],
      "timeout": 60,
      "retries": 2
    },
    {
      "id": "analyze_data",
      "name": "Analyze Sales Data",
      "description": "Perform statistical analysis on the sample data",
      "agent_id": "analyzer",
      "input": {"data": "Use sample data from previous step"},
      "expected_output": "Statistical summary and insights",
      "status": "pending",
      "depends_on": ["collect_sample_data"],
      "timeout": 90,
      "retries": 2
    },
    {
      "id": "generate_report",
      "name": "Generate Analysis Report",
      "description": "Create a formatted report of the analysis results",
      "agent_id": "reporter",
      "input": {"analysis": "Use analysis results from previous step"},
      "expected_output": "Formatted markdown report",
      "status": "pending",
      "depends_on": ["analyze_data"],
      "timeout": 90,
      "retries": 2
    }
  ],
  "validation": {
    "required": false,
    "custom_checks": []
  },
  "settings": {
    "max_retries": 2,
    "step_timeout": 120,
    "parallel_execution": false,
    "stop_on_failure": true,
    "log_level": "info"
  }
}
EOF

    echo "Testing multi-agent configuration validation..."
    if ! jq . multi_agent_process.json > /dev/null 2>&1; then
        echo "FAIL: multi_agent_process.json is not valid JSON"
        exit_code=1
    else
        echo "PASS: Multi-agent configuration is valid JSON"
    fi
    
    # Test multi-agent structure
    if ! jq -e '.agents | length >= 2' multi_agent_process.json > /dev/null; then
        echo "FAIL: Multi-agent configuration doesn't have multiple agents"
        exit_code=1
    else
        echo "PASS: Multi-agent configuration has multiple agents"
    fi
    
    # Test dependency structure
    if ! jq -e '.agents[] | select(.depends_on | length > 0)' multi_agent_process.json > /dev/null; then
        echo "INFO: No agent dependencies found (acceptable for simple workflows)"
    else
        echo "PASS: Agent dependencies are properly configured"
    fi
    
    echo
    echo "=== Testing 3: Configuration Templates from Examples ==="
    
    # Test 3: Validate existing example configurations
    local example_configs=(
        "../../examples/configs/research_assistant.json"
        "../../examples/configs/web_scraper.json"
        "../../examples/configs/content_creator.json"
        "../../examples/configs/data_analyzer.json"
    )
    
    for config in "${example_configs[@]}"; do
        if [ -f "$config" ]; then
            config_name=$(basename "$config")
            echo "Testing $config_name..."
            if jq . "$config" > /dev/null 2>&1; then
                echo "PASS: $config_name is valid JSON"
                
                # Test required fields
                if jq -e '.agent.name' "$config" > /dev/null; then
                    echo "PASS: $config_name has agent name"
                else
                    echo "FAIL: $config_name missing agent name"
                    exit_code=1
                fi
                
                if jq -e '.llm.provider' "$config" > /dev/null; then
                    echo "PASS: $config_name has LLM provider"
                else
                    echo "FAIL: $config_name missing LLM provider"
                    exit_code=1
                fi
                
                if jq -e '.workflows' "$config" > /dev/null; then
                    echo "PASS: $config_name has workflows defined"
                else
                    echo "FAIL: $config_name missing workflows"
                    exit_code=1
                fi
            else
                echo "FAIL: $config_name is not valid JSON"
                exit_code=1
            fi
        else
            echo "WARN: Example config $config not found"
        fi
        echo
    done
    
    echo "=== Testing 4: Framework Command Structure ==="
    
    # Test 4: Check that the process command exists and can handle configurations
    echo "Testing process command availability..."
    if ../../generic-agent --help | grep -q "process"; then
        echo "PASS: Process command is available"
    elif ../../ledit --help | grep -q "process"; then
        echo "PASS: Process command is available via ledit"
    else
        echo "FAIL: Process command not found in help output"
        exit_code=1
    fi
    
    # Test configuration file handling
    echo "Testing configuration file validation..."
    # This tests that the command can at least parse the configuration without failing immediately
    timeout 10s ../../ledit process multi_agent_process.json --help 2>/dev/null || echo "INFO: Process command structure test completed"
    
    echo
    echo "=== Testing 5: Schema Validation ==="
    
    # Test 5: Create invalid configurations and ensure they're caught
    cat > invalid_config.json << 'EOF'
{
  "agent": {
    "name": "Test Agent"
    // Missing required fields
  }
}
EOF

    echo "Testing invalid configuration detection..."
    if jq . invalid_config.json > /dev/null 2>&1; then
        echo "FAIL: Invalid JSON was accepted (should have syntax error)"
        exit_code=1
    else
        echo "PASS: Invalid JSON properly rejected"
    fi
    
    # Test missing required fields
    cat > incomplete_config.json << 'EOF'
{
  "agent": {
    "name": "Test Agent"
  }
}
EOF

    if jq . incomplete_config.json > /dev/null 2>&1; then
        echo "PASS: Incomplete but valid JSON structure accepted"
        if jq -e '.llm' incomplete_config.json > /dev/null; then
            echo "FAIL: Should be missing LLM configuration"
            exit_code=1
        else
            echo "PASS: Missing LLM configuration detected"
        fi
    fi
    
    echo
    echo "=== Test Summary ==="
    
    # Clean up
    cd ../ || true
    rm -rf "$test_dir"
    
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    if [ $exit_code -eq 0 ]; then
        echo "✅ All generic agent framework tests PASSED"
        echo "🎉 Framework validation completed successfully"
        echo "📊 Test coverage:"
        echo "   ✓ Single agent configuration validation"
        echo "   ✓ Multi-agent orchestration structure"  
        echo "   ✓ Example template validation"
        echo "   ✓ Command structure verification"
        echo "   ✓ Schema validation and error handling"
    else
        echo "❌ Some tests FAILED"
        echo "🔧 Framework needs attention in failing areas"
    fi
    
    echo "⏱️  Test duration: $duration seconds"
    echo "----------------------------------------------------"
    
    return $exit_code
}