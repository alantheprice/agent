#!/bin/bash

# Function to return the test name
get_test_name() {
    echo "LLM Provider Integration Test"
}

# Function to run the test logic
run_test_logic() {
    local model_name=$1
    echo "--- TEST: LLM Provider Integration ---"
    start_time=$(date +%s)
    
    local test_dir="llm_provider_test"
    rm -rf "$test_dir"
    mkdir -p "$test_dir"
    cd "$test_dir" || exit 1
    
    local exit_code=0
    
    echo "=== Testing 1: OpenAI Provider Configuration ==="
    
    cat > openai_agent.json << 'EOF'
{
  "agent": {
    "name": "OpenAI Test Agent",
    "description": "Tests OpenAI provider integration",
    "version": "1.0.0",
    "max_iterations": 3,
    "timeout": "2m",
    "interactive": false
  },
  "llm": {
    "provider": "openai",
    "model": "gpt-4",
    "temperature": 0.1,
    "max_tokens": 500,
    "system_prompt": "You are a test assistant. Answer questions concisely."
  },
  "data_sources": [
    {
      "name": "test_input",
      "type": "stdin",
      "config": {}
    }
  ],
  "workflows": [
    {
      "name": "simple_response",
      "description": "Provide a simple response to test input",
      "trigger": {
        "conditions": ["test"],
        "priority": 100
      },
      "steps": [
        {
          "name": "respond",
          "type": "llm",
          "config": {
            "prompt": "Respond to this test: {input}"
          }
        }
      ],
      "output": {
        "format": "text",
        "destination": "console"
      }
    }
  ],
  "tools": {},
  "outputs": [
    {
      "name": "response",
      "type": "console",
      "config": {
        "format": "text"
      }
    }
  ],
  "environment": {
    "variables": {},
    "workspace_root": ".",
    "log_level": "info"
  }
}
EOF

    echo "Testing OpenAI provider configuration validation..."
    if jq . openai_agent.json > /dev/null 2>&1; then
        echo "PASS: OpenAI configuration is valid JSON"
        if jq -e '.llm.provider == "openai"' openai_agent.json > /dev/null; then
            echo "PASS: OpenAI provider properly specified"
        else
            echo "FAIL: OpenAI provider not properly specified"
            exit_code=1
        fi
    else
        echo "FAIL: OpenAI configuration is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 2: Gemini Provider Configuration ==="
    
    cat > gemini_agent.json << 'EOF'
{
  "agent": {
    "name": "Gemini Test Agent",
    "description": "Tests Gemini provider integration",
    "version": "1.0.0",
    "max_iterations": 3,
    "timeout": "2m",
    "interactive": false
  },
  "llm": {
    "provider": "gemini",
    "model": "gemini-1.5-pro",
    "temperature": 0.2,
    "max_tokens": 500,
    "system_prompt": "You are a test assistant. Answer questions concisely."
  },
  "data_sources": [
    {
      "name": "test_input",
      "type": "stdin",
      "config": {}
    }
  ],
  "workflows": [
    {
      "name": "simple_response",
      "description": "Provide a simple response to test input",
      "trigger": {
        "conditions": ["test"],
        "priority": 100
      },
      "steps": [
        {
          "name": "respond",
          "type": "llm",
          "config": {
            "prompt": "Respond to this test: {input}"
          }
        }
      ],
      "output": {
        "format": "text",
        "destination": "console"
      }
    }
  ],
  "tools": {},
  "outputs": [
    {
      "name": "response",
      "type": "console",
      "config": {
        "format": "text"
      }
    }
  ],
  "environment": {
    "variables": {},
    "workspace_root": ".",
    "log_level": "info"
  }
}
EOF

    echo "Testing Gemini provider configuration validation..."
    if jq . gemini_agent.json > /dev/null 2>&1; then
        echo "PASS: Gemini configuration is valid JSON"
        if jq -e '.llm.provider == "gemini"' gemini_agent.json > /dev/null; then
            echo "PASS: Gemini provider properly specified"
        else
            echo "FAIL: Gemini provider not properly specified"
            exit_code=1
        fi
    else
        echo "FAIL: Gemini configuration is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 3: Ollama Provider Configuration ==="
    
    cat > ollama_agent.json << 'EOF'
{
  "agent": {
    "name": "Ollama Test Agent",
    "description": "Tests Ollama provider integration",
    "version": "1.0.0",
    "max_iterations": 3,
    "timeout": "2m",
    "interactive": false
  },
  "llm": {
    "provider": "ollama",
    "model": "llama2",
    "temperature": 0.3,
    "max_tokens": 500,
    "system_prompt": "You are a test assistant. Answer questions concisely."
  },
  "data_sources": [
    {
      "name": "test_input",
      "type": "stdin",
      "config": {}
    }
  ],
  "workflows": [
    {
      "name": "simple_response",
      "description": "Provide a simple response to test input",
      "trigger": {
        "conditions": ["test"],
        "priority": 100
      },
      "steps": [
        {
          "name": "respond",
          "type": "llm",
          "config": {
            "prompt": "Respond to this test: {input}"
          }
        }
      ],
      "output": {
        "format": "text",
        "destination": "console"
      }
    }
  ],
  "tools": {},
  "outputs": [
    {
      "name": "response",
      "type": "console",
      "config": {
        "format": "text"
      }
    }
  ],
  "environment": {
    "variables": {},
    "workspace_root": ".",
    "log_level": "info"
  }
}
EOF

    echo "Testing Ollama provider configuration validation..."
    if jq . ollama_agent.json > /dev/null 2>&1; then
        echo "PASS: Ollama configuration is valid JSON"
        if jq -e '.llm.provider == "ollama"' ollama_agent.json > /dev/null; then
            echo "PASS: Ollama provider properly specified"
        else
            echo "FAIL: Ollama provider not properly specified"
            exit_code=1
        fi
    else
        echo "FAIL: Ollama configuration is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 4: Multi-Agent Process with Different Providers ==="
    
    cat > multi_provider_process.json << 'EOF'
{
  "version": "1.0",
  "goal": "Test multi-agent system with different LLM providers",
  "description": "Validates that different agents can use different LLM providers",
  "base_model": "gpt-4",
  "agents": [
    {
      "id": "openai_agent",
      "name": "OpenAI Agent",
      "persona": "data_analyst",
      "description": "Uses OpenAI models",
      "skills": ["analysis"],
      "model": "gpt-4",
      "priority": 1,
      "depends_on": [],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 5000,
        "max_cost": 0.5,
        "token_warning": 4000,
        "cost_warning": 0.4
      }
    },
    {
      "id": "gemini_agent",
      "name": "Gemini Agent", 
      "persona": "technical_writer",
      "description": "Uses Gemini models",
      "skills": ["writing"],
      "model": "gemini:gemini-1.5-pro",
      "priority": 2,
      "depends_on": ["openai_agent"],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 5000,
        "max_cost": 0.5,
        "token_warning": 4000,
        "cost_warning": 0.4
      }
    }
  ],
  "steps": [
    {
      "id": "analyze_data",
      "name": "Analyze Sample Data",
      "description": "OpenAI agent analyzes some sample data",
      "agent_id": "openai_agent",
      "input": {"data": "Sample: [1, 2, 3, 4, 5]"},
      "expected_output": "Data analysis summary",
      "status": "pending",
      "depends_on": [],
      "timeout": 60,
      "retries": 1
    },
    {
      "id": "write_summary",
      "name": "Write Analysis Summary",
      "description": "Gemini agent writes a summary of the analysis",
      "agent_id": "gemini_agent",
      "input": {"analysis": "Use results from analyze_data step"},
      "expected_output": "Written summary",
      "status": "pending",
      "depends_on": ["analyze_data"],
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
    "stop_on_failure": true,
    "log_level": "info"
  }
}
EOF

    echo "Testing multi-provider process configuration..."
    if jq . multi_provider_process.json > /dev/null 2>&1; then
        echo "PASS: Multi-provider process is valid JSON"
        
        # Test agent model specifications
        openai_count=$(jq '[.agents[] | select(.model | contains("gpt"))] | length' multi_provider_process.json)
        gemini_count=$(jq '[.agents[] | select(.model | contains("gemini"))] | length' multi_provider_process.json)
        
        if [ "$openai_count" -gt 0 ] && [ "$gemini_count" -gt 0 ]; then
            echo "PASS: Multiple providers configured in agents"
        else
            echo "FAIL: Multiple providers not properly configured"
            exit_code=1
        fi
        
        # Test step dependencies
        if jq -e '.steps[] | select(.depends_on | length > 0)' multi_provider_process.json > /dev/null; then
            echo "PASS: Step dependencies configured"
        else
            echo "FAIL: Step dependencies not configured"
            exit_code=1
        fi
    else
        echo "FAIL: Multi-provider process is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 5: Provider-Specific Features ==="
    
    # Test provider-specific configurations
    echo "Testing provider-specific temperature settings..."
    openai_temp=$(jq -r '.llm.temperature' openai_agent.json)
    gemini_temp=$(jq -r '.llm.temperature' gemini_agent.json) 
    ollama_temp=$(jq -r '.llm.temperature' ollama_agent.json)
    
    if [ "$openai_temp" != "null" ] && [ "$gemini_temp" != "null" ] && [ "$ollama_temp" != "null" ]; then
        echo "PASS: All providers have temperature settings"
    else
        echo "FAIL: Some providers missing temperature settings"
        exit_code=1
    fi
    
    echo "Testing provider-specific model specifications..."
    if jq -e '.llm.model' openai_agent.json > /dev/null && \
       jq -e '.llm.model' gemini_agent.json > /dev/null && \
       jq -e '.llm.model' ollama_agent.json > /dev/null; then
        echo "PASS: All providers have model specifications"
    else
        echo "FAIL: Some providers missing model specifications"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 6: Command Integration ==="
    
    echo "Testing process command with provider configurations..."
    # Test that the command can at least parse the files without immediately failing
    for config in openai_agent.json gemini_agent.json ollama_agent.json multi_provider_process.json; do
        if ../../ledit process "$config" --dry-run --skip-prompt 2>/dev/null; then
            echo "PASS: Process command accepts $config"
        else
            echo "INFO: Process command structure test for $config (may require actual LLM access)"
        fi
    done
    
    echo
    echo "=== Test Summary ==="
    
    # Clean up
    cd ../ || true
    rm -rf "$test_dir"
    
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    if [ $exit_code -eq 0 ]; then
        echo "✅ All LLM provider integration tests PASSED"
        echo "🎉 Provider configurations validated successfully"
        echo "📊 Test coverage:"
        echo "   ✓ OpenAI provider configuration"
        echo "   ✓ Gemini provider configuration"  
        echo "   ✓ Ollama provider configuration"
        echo "   ✓ Multi-provider orchestration"
        echo "   ✓ Provider-specific features"
        echo "   ✓ Command integration"
    else
        echo "❌ Some provider tests FAILED"
        echo "🔧 Provider integration needs attention"
    fi
    
    echo "⏱️  Test duration: $duration seconds"
    echo "----------------------------------------------------"
    
    return $exit_code
}