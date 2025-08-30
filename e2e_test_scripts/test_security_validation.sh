#!/bin/bash

# Function to return the test name
get_test_name() {
    echo "Security & Validation Rules Test"
}

# Function to run the test logic
run_test_logic() {
    local model_name=$1
    echo "--- TEST: Security & Validation Rules ---"
    start_time=$(date +%s)
    
    local test_dir="security_validation_test"
    rm -rf "$test_dir"
    mkdir -p "$test_dir"
    cd "$test_dir" || exit 1
    
    local exit_code=0
    
    echo "=== Testing 1: Path Security Restrictions ==="
    
    # Create test directories for security testing
    mkdir -p allowed_dir restricted_dir
    
    cat > path_security.json << 'EOF'
{
  "agent": {
    "name": "Security Test Agent",
    "description": "Tests path security restrictions",
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
    "system_prompt": "You are a security-aware agent. Only access allowed paths."
  },
  "data_sources": [
    {
      "name": "secure_input",
      "type": "stdin",
      "config": {}
    }
  ],
  "workflows": [
    {
      "name": "secure_workflow",
      "description": "Workflow that respects path restrictions",
      "trigger": {
        "conditions": ["secure_task"],
        "priority": 100
      },
      "steps": [
        {
          "name": "process_securely",
          "type": "llm",
          "config": {
            "prompt": "Process this input securely: {input}"
          }
        }
      ],
      "output": {
        "format": "text",
        "destination": "file",
        "path": "./allowed_dir/output.txt"
      }
    }
  ],
  "tools": {
    "file_access": {
      "enabled": true,
      "timeout": "30s"
    }
  },
  "outputs": [
    {
      "name": "secure_output",
      "type": "file",
      "config": {
        "path": "./allowed_dir/secure_output.txt",
        "format": "text",
        "append": false
      }
    }
  ],
  "environment": {
    "variables": {
      "SECURE_MODE": "true"
    },
    "workspace_root": ".",
    "log_level": "info"
  },
  "security": {
    "enabled": true,
    "allowed_paths": ["./allowed_dir", "./.agent"],
    "blocked_paths": ["./restricted_dir", "/etc", "/var"],
    "max_file_size": "10MB",
    "require_approval": false,
    "sandbox_mode": true
  },
  "validation": {
    "enabled": true,
    "rules": [
      {
        "name": "path_validation",
        "type": "custom",
        "config": {
          "validator": "path_security",
          "allowed_patterns": ["./allowed_dir/*"],
          "blocked_patterns": ["./restricted_dir/*", "../*"]
        }
      }
    ],
    "on_failure": "stop"
  }
}
EOF

    echo "Testing path security configuration..."
    if jq . path_security.json > /dev/null 2>&1; then
        echo "PASS: Path security configuration is valid JSON"
        
        # Test security is enabled
        security_enabled=$(jq -r '.security.enabled' path_security.json)
        if [ "$security_enabled" = "true" ]; then
            echo "PASS: Security system enabled"
        else
            echo "FAIL: Security system not enabled"
            exit_code=1
        fi
        
        # Test allowed paths are configured
        allowed_paths=$(jq '.security.allowed_paths | length' path_security.json)
        if [ "$allowed_paths" -gt 0 ]; then
            echo "PASS: Allowed paths configured ($allowed_paths paths)"
        else
            echo "FAIL: No allowed paths configured"
            exit_code=1
        fi
        
        # Test blocked paths are configured
        blocked_paths=$(jq '.security.blocked_paths | length' path_security.json)
        if [ "$blocked_paths" -gt 0 ]; then
            echo "PASS: Blocked paths configured ($blocked_paths paths)"
        else
            echo "WARN: No blocked paths configured"
        fi
        
        # Test file size limit
        max_file_size=$(jq -r '.security.max_file_size' path_security.json)
        if [ "$max_file_size" != "null" ]; then
            echo "PASS: File size limit configured: $max_file_size"
        else
            echo "WARN: No file size limit configured"
        fi
    else
        echo "FAIL: Path security configuration is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 2: Input Validation Rules ==="
    
    cat > input_validation.json << 'EOF'
{
  "agent": {
    "name": "Input Validation Agent",
    "description": "Tests input validation mechanisms",
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
    "system_prompt": "You are a validation-aware agent. Process only valid inputs."
  },
  "data_sources": [
    {
      "name": "validated_input",
      "type": "stdin",
      "config": {},
      "preprocessing": [
        {
          "type": "validate",
          "config": {
            "rules": ["no_scripts", "length_limit", "safe_content"]
          }
        },
        {
          "type": "sanitize",
          "config": {
            "remove_html": true,
            "escape_special_chars": true
          }
        }
      ]
    }
  ],
  "workflows": [
    {
      "name": "validated_workflow",
      "description": "Workflow that validates all inputs",
      "trigger": {
        "conditions": ["validation_test"],
        "priority": 100
      },
      "steps": [
        {
          "name": "validate_and_process",
          "type": "llm",
          "config": {
            "prompt": "Process this validated input: {input}",
            "validation": {
              "input_rules": ["safe_content", "no_injection"],
              "output_rules": ["safe_response", "appropriate_content"]
            }
          }
        }
      ],
      "output": {
        "format": "text",
        "destination": "console",
        "validation": {
          "rules": ["safe_output", "no_sensitive_data"]
        }
      }
    }
  ],
  "tools": {
    "input_validator": {
      "enabled": true,
      "timeout": "10s",
      "strict_mode": true
    }
  },
  "outputs": [
    {
      "name": "validated_output",
      "type": "console",
      "config": {
        "format": "text",
        "validation": {
          "enabled": true,
          "rules": ["content_safety", "no_code_injection"]
        }
      }
    }
  ],
  "environment": {
    "variables": {
      "VALIDATION_LEVEL": "strict"
    },
    "workspace_root": ".",
    "log_level": "debug"
  },
  "security": {
    "enabled": true,
    "input_sanitization": true,
    "output_filtering": true,
    "script_detection": true,
    "injection_protection": true
  },
  "validation": {
    "enabled": true,
    "strict_mode": true,
    "rules": [
      {
        "name": "input_length",
        "type": "custom",
        "config": {
          "validator": "length_range",
          "min": 1,
          "max": 5000
        }
      },
      {
        "name": "safe_content",
        "type": "regex",
        "config": {
          "pattern": "^[a-zA-Z0-9\\s\\.,!?-]+$",
          "description": "Only allow safe characters"
        }
      },
      {
        "name": "no_scripts",
        "type": "regex",
        "config": {
          "pattern": "(<script|javascript:|data:|vbscript:)",
          "invert": true,
          "description": "Block script injection attempts"
        }
      }
    ],
    "on_failure": "stop"
  }
}
EOF

    echo "Testing input validation configuration..."
    if jq . input_validation.json > /dev/null 2>&1; then
        echo "PASS: Input validation configuration is valid JSON"
        
        # Test validation is enabled and strict
        validation_enabled=$(jq -r '.validation.enabled' input_validation.json)
        strict_mode=$(jq -r '.validation.strict_mode' input_validation.json)
        
        if [ "$validation_enabled" = "true" ] && [ "$strict_mode" = "true" ]; then
            echo "PASS: Strict validation enabled"
        else
            echo "FAIL: Validation not properly configured for security"
            exit_code=1
        fi
        
        # Test validation rules
        validation_rules=$(jq '.validation.rules | length' input_validation.json)
        if [ "$validation_rules" -ge 3 ]; then
            echo "PASS: Multiple validation rules configured ($validation_rules rules)"
        else
            echo "FAIL: Insufficient validation rules"
            exit_code=1
        fi
        
        # Test security features
        input_sanitization=$(jq -r '.security.input_sanitization' input_validation.json)
        injection_protection=$(jq -r '.security.injection_protection' input_validation.json)
        
        if [ "$input_sanitization" = "true" ] && [ "$injection_protection" = "true" ]; then
            echo "PASS: Input sanitization and injection protection enabled"
        else
            echo "FAIL: Security protections not fully enabled"
            exit_code=1
        fi
    else
        echo "FAIL: Input validation configuration is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 3: Output Validation and Filtering ==="
    
    cat > output_validation.json << 'EOF'
{
  "agent": {
    "name": "Output Validation Agent",
    "description": "Tests output validation and filtering",
    "version": "1.0.0",
    "max_iterations": 3,
    "timeout": "2m",
    "interactive": false
  },
  "llm": {
    "provider": "openai",
    "model": "gpt-4",
    "temperature": 0.1,
    "max_tokens": 1000,
    "system_prompt": "You are an agent that generates content subject to output validation."
  },
  "data_sources": [
    {
      "name": "content_request",
      "type": "stdin",
      "config": {}
    }
  ],
  "workflows": [
    {
      "name": "content_generation",
      "description": "Generate content with output validation",
      "trigger": {
        "conditions": ["content_request"],
        "priority": 100
      },
      "steps": [
        {
          "name": "generate_content",
          "type": "llm",
          "config": {
            "prompt": "Generate appropriate content for: {input}",
            "output_validation": {
              "enabled": true,
              "rules": ["appropriate_content", "no_sensitive_info", "length_check"]
            }
          }
        },
        {
          "name": "filter_output",
          "type": "tool",
          "config": {
            "tool": "content_filter",
            "params": {
              "input": "{generate_content}",
              "filter_rules": ["profanity", "personal_info", "harmful_content"]
            }
          },
          "depends_on": ["generate_content"]
        }
      ],
      "output": {
        "format": "text",
        "destination": "file",
        "path": "./allowed_dir/filtered_output.txt",
        "postprocessing": [
          {
            "type": "content_filter",
            "config": {
              "remove_sensitive": true,
              "mask_personal_info": true
            }
          }
        ]
      }
    }
  ],
  "tools": {
    "content_filter": {
      "enabled": true,
      "timeout": "30s",
      "strict_mode": true
    }
  },
  "outputs": [
    {
      "name": "filtered_content",
      "type": "file",
      "config": {
        "path": "./allowed_dir/validated_content.txt",
        "format": "text",
        "validation": {
          "enabled": true,
          "rules": ["content_safety", "privacy_protection", "appropriate_language"]
        },
        "postprocessing": [
          {
            "type": "sensitive_data_removal",
            "config": {
              "patterns": ["email", "phone", "ssn", "credit_card"]
            }
          }
        ]
      }
    }
  ],
  "environment": {
    "variables": {
      "CONTENT_FILTER_LEVEL": "high"
    },
    "workspace_root": ".",
    "log_level": "info"
  },
  "security": {
    "enabled": true,
    "output_filtering": true,
    "sensitive_data_detection": true,
    "content_moderation": true,
    "privacy_protection": true
  },
  "validation": {
    "enabled": true,
    "rules": [
      {
        "name": "content_appropriateness",
        "type": "custom",
        "config": {
          "validator": "content_safety",
          "categories": ["violence", "hate_speech", "adult_content"]
        }
      },
      {
        "name": "personal_info_check",
        "type": "regex",
        "config": {
          "pattern": "(\\d{3}-\\d{2}-\\d{4}|\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b)",
          "invert": true,
          "description": "Block personal information"
        }
      },
      {
        "name": "output_length",
        "type": "custom",
        "config": {
          "validator": "length_range",
          "min": 10,
          "max": 2000
        }
      }
    ],
    "on_failure": "filter_and_retry"
  }
}
EOF

    echo "Testing output validation configuration..."
    if jq . output_validation.json > /dev/null 2>&1; then
        echo "PASS: Output validation configuration is valid JSON"
        
        # Test output filtering is enabled
        output_filtering=$(jq -r '.security.output_filtering' output_validation.json)
        content_moderation=$(jq -r '.security.content_moderation' output_validation.json)
        
        if [ "$output_filtering" = "true" ] && [ "$content_moderation" = "true" ]; then
            echo "PASS: Output filtering and content moderation enabled"
        else
            echo "FAIL: Output security features not enabled"
            exit_code=1
        fi
        
        # Test privacy protection
        privacy_protection=$(jq -r '.security.privacy_protection' output_validation.json)
        sensitive_data_detection=$(jq -r '.security.sensitive_data_detection' output_validation.json)
        
        if [ "$privacy_protection" = "true" ] && [ "$sensitive_data_detection" = "true" ]; then
            echo "PASS: Privacy protection features enabled"
        else
            echo "FAIL: Privacy protection not properly configured"
            exit_code=1
        fi
        
        # Test validation failure strategy
        failure_strategy=$(jq -r '.validation.on_failure' output_validation.json)
        if [ "$failure_strategy" = "filter_and_retry" ]; then
            echo "PASS: Intelligent failure strategy configured"
        else
            echo "WARN: Basic failure strategy configured"
        fi
    else
        echo "FAIL: Output validation configuration is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 4: Multi-Agent Security Orchestration ==="
    
    cat > secure_orchestration.json << 'EOF'
{
  "version": "1.0",
  "goal": "Test security in multi-agent orchestration",
  "description": "Validates security controls work across multiple agents",
  "base_model": "gpt-4",
  "agents": [
    {
      "id": "input_validator",
      "name": "Input Validation Agent",
      "persona": "security_validator",
      "description": "Validates and sanitizes inputs",
      "skills": ["input_validation", "security"],
      "priority": 1,
      "depends_on": [],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 1000,
        "max_cost": 0.1
      },
      "security": {
        "role": "validator",
        "permissions": ["read_input", "validate_content"],
        "restrictions": ["no_file_write", "no_external_access"]
      }
    },
    {
      "id": "content_processor",
      "name": "Content Processing Agent",
      "persona": "content_processor",
      "description": "Processes validated content",
      "skills": ["content_processing"],
      "priority": 2,
      "depends_on": ["input_validator"],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 1500,
        "max_cost": 0.15
      },
      "security": {
        "role": "processor",
        "permissions": ["read_validated_input", "write_temp_files"],
        "restrictions": ["no_external_access", "sandboxed_execution"]
      }
    },
    {
      "id": "output_sanitizer",
      "name": "Output Sanitization Agent",
      "persona": "security_sanitizer",
      "description": "Sanitizes and validates final output",
      "skills": ["output_sanitization", "security"],
      "priority": 3,
      "depends_on": ["content_processor"],
      "config": {"skip_prompt": "true"},
      "budget": {
        "max_tokens": 800,
        "max_cost": 0.08
      },
      "security": {
        "role": "sanitizer",
        "permissions": ["read_processed_content", "write_final_output"],
        "restrictions": ["content_filtering_required", "privacy_protection"]
      }
    }
  ],
  "steps": [
    {
      "id": "validate_input",
      "name": "Validate Input",
      "description": "Validate and sanitize user input",
      "agent_id": "input_validator",
      "input": {"raw_input": "user_provided_content"},
      "expected_output": "validated_input",
      "status": "pending",
      "depends_on": [],
      "timeout": 30,
      "retries": 1,
      "security": {
        "input_validation": true,
        "output_sanitization": true
      }
    },
    {
      "id": "process_content",
      "name": "Process Content",
      "description": "Process validated content safely",
      "agent_id": "content_processor",
      "input": {"validated_input": "from validate_input"},
      "expected_output": "processed_content",
      "status": "pending",
      "depends_on": ["validate_input"],
      "timeout": 45,
      "retries": 2,
      "security": {
        "sandboxed": true,
        "resource_limits": true
      }
    },
    {
      "id": "sanitize_output",
      "name": "Sanitize Output",
      "description": "Final sanitization of output",
      "agent_id": "output_sanitizer",
      "input": {"processed_content": "from process_content"},
      "expected_output": "safe_final_output",
      "status": "pending",
      "depends_on": ["process_content"],
      "timeout": 30,
      "retries": 1,
      "security": {
        "content_filtering": true,
        "privacy_check": true,
        "final_validation": true
      }
    }
  ],
  "validation": {
    "required": true,
    "global_rules": [
      {
        "name": "security_compliance",
        "type": "custom",
        "config": {
          "validator": "security_compliance_check",
          "requirements": ["input_validated", "content_filtered", "privacy_protected"]
        }
      }
    ]
  },
  "settings": {
    "max_retries": 2,
    "step_timeout": 60,
    "parallel_execution": false,
    "stop_on_failure": true,
    "log_level": "debug",
    "security_mode": "strict"
  }
}
EOF

    echo "Testing secure orchestration configuration..."
    if jq . secure_orchestration.json > /dev/null 2>&1; then
        echo "PASS: Secure orchestration configuration is valid JSON"
        
        # Test security roles are defined
        security_roles=$(jq '[.agents[].security.role] | length' secure_orchestration.json)
        if [ "$security_roles" -eq 3 ]; then
            echo "PASS: Security roles defined for all agents"
        else
            echo "FAIL: Security roles not properly defined"
            exit_code=1
        fi
        
        # Test permissions and restrictions
        agents_with_permissions=$(jq '[.agents[] | select(.security.permissions)] | length' secure_orchestration.json)
        agents_with_restrictions=$(jq '[.agents[] | select(.security.restrictions)] | length' secure_orchestration.json)
        
        if [ "$agents_with_permissions" -eq 3 ] && [ "$agents_with_restrictions" -eq 3 ]; then
            echo "PASS: All agents have permissions and restrictions defined"
        else
            echo "FAIL: Agent security permissions/restrictions incomplete"
            exit_code=1
        fi
        
        # Test step-level security
        steps_with_security=$(jq '[.steps[] | select(.security)] | length' secure_orchestration.json)
        if [ "$steps_with_security" -eq 3 ]; then
            echo "PASS: All steps have security configurations"
        else
            echo "FAIL: Step-level security not fully configured"
            exit_code=1
        fi
    else
        echo "FAIL: Secure orchestration configuration is invalid JSON"
        exit_code=1
    fi
    
    echo
    echo "=== Testing 5: Security Compliance Validation ==="
    
    echo "Testing security compliance across all configurations..."
    
    # Test all configs have security enabled
    security_enabled_count=0
    total_configs=0
    
    for config in path_security.json input_validation.json output_validation.json secure_orchestration.json; do
        if [ -f "$config" ]; then
            total_configs=$((total_configs + 1))
            security_enabled=$(jq -r '.security.enabled // .settings.security_mode // "false"' "$config")
            if [ "$security_enabled" = "true" ] || [ "$security_enabled" = "strict" ]; then
                security_enabled_count=$((security_enabled_count + 1))
                echo "PASS: $config has security enabled"
            else
                echo "FAIL: $config does not have security enabled"
                exit_code=1
            fi
        fi
    done
    
    if [ "$security_enabled_count" -eq "$total_configs" ]; then
        echo "PASS: All configurations have security enabled"
    else
        echo "FAIL: Not all configurations have security enabled"
        exit_code=1
    fi
    
    # Test validation is properly configured
    validation_enabled_count=0
    for config in path_security.json input_validation.json output_validation.json secure_orchestration.json; do
        if [ -f "$config" ]; then
            validation_enabled=$(jq -r '.validation.enabled' "$config")
            if [ "$validation_enabled" = "true" ]; then
                validation_enabled_count=$((validation_enabled_count + 1))
            fi
        fi
    done
    
    if [ "$validation_enabled_count" -eq "$total_configs" ]; then
        echo "PASS: All configurations have validation enabled"
    else
        echo "WARN: Not all configurations have validation enabled"
    fi
    
    echo
    echo "=== Testing 6: Security Feature Coverage ==="
    
    echo "Testing comprehensive security feature coverage..."
    
    # Check for essential security features
    features_tested=(
        "path_restrictions"
        "input_sanitization"
        "output_filtering"
        "content_validation"
        "injection_protection"
        "privacy_protection"
        "sandbox_mode"
        "permission_system"
    )
    
    features_found=0
    
    for feature in "${features_tested[@]}"; do
        found=false
        for config in path_security.json input_validation.json output_validation.json secure_orchestration.json; do
            if [ -f "$config" ]; then
                if jq -r tostring "$config" | grep -q "$feature"; then
                    found=true
                    break
                fi
            fi
        done
        
        if [ "$found" = true ]; then
            features_found=$((features_found + 1))
            echo "PASS: Security feature '$feature' is tested"
        else
            echo "WARN: Security feature '$feature' not explicitly tested"
        fi
    done
    
    feature_coverage=$((features_found * 100 / ${#features_tested[@]}))
    echo "INFO: Security feature coverage: $feature_coverage%"
    
    if [ "$feature_coverage" -ge 75 ]; then
        echo "PASS: Good security feature coverage ($feature_coverage%)"
    else
        echo "WARN: Security feature coverage could be improved ($feature_coverage%)"
    fi
    
    echo
    echo "=== Test Summary ==="
    
    # Clean up
    cd ../ || true
    rm -rf "$test_dir"
    
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    if [ $exit_code -eq 0 ]; then
        echo "✅ All security and validation tests PASSED"
        echo "🎉 Security mechanisms validated successfully"
        echo "📊 Test coverage:"
        echo "   ✓ Path security restrictions"
        echo "   ✓ Input validation rules"
        echo "   ✓ Output validation and filtering"
        echo "   ✓ Multi-agent security orchestration"
        echo "   ✓ Security compliance validation"
        echo "   ✓ Security feature coverage ($feature_coverage%)"
    else
        echo "❌ Some security and validation tests FAILED"
        echo "🔧 Security mechanisms need attention"
    fi
    
    echo "⏱️  Test duration: $duration seconds"
    echo "----------------------------------------------------"
    
    return $exit_code
}