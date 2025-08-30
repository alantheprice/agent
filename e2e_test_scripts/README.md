# Generic Agent Framework E2E Tests

## Overview

This directory contains end-to-end tests for the generic agent framework, replacing the legacy coding-specific tests that were designed for the previous `ledit` functionality.

## Test Structure

### Main Test: `test_generic_agent_framework.sh`

Comprehensive e2e test that validates the entire generic agent framework:

**Test Coverage:**
1. **Single Agent Configuration**: Validates basic agent configuration files
2. **Multi-Agent Orchestration**: Tests complex multi-agent workflows  
3. **Example Template Validation**: Ensures all example configurations work
4. **Command Structure**: Verifies the process command is available
5. **Schema Validation**: Tests configuration validation and error handling

**Test Categories:**
- ✅ JSON configuration validation
- ✅ Required field verification
- ✅ Agent dependency structure
- ✅ Workflow step definitions
- ✅ Budget and timeout configurations
- ✅ Example template compatibility
- ✅ Error handling for invalid configurations

### Test Runner: `run_tests.sh`

Simple test runner that:
- Builds the project
- Executes the main e2e test
- Provides clear pass/fail reporting
- Supports model selection via command line argument

## Usage

### Run All Tests
```bash
# Run with default model (gpt-4)
./run_tests.sh

# Run with specific model
./run_tests.sh "gpt-3.5-turbo"
```

### Run Individual Test
```bash
# Run just the framework test
./test_generic_agent_framework.sh "gpt-4"
```

## Test Philosophy

### Framework-Focused Testing
Unlike the legacy tests that focused on specific code generation tasks, these tests validate:
- **Configuration Structure**: JSON schema validation
- **Framework Components**: Multi-agent orchestration, workflow execution
- **Template System**: Reusable agent configurations
- **Integration Points**: LLM providers, tool systems, validation rules

### Infrastructure over Output
The tests focus on ensuring the framework infrastructure works correctly rather than validating specific LLM outputs, making them more reliable and deterministic.

## Example Test Configurations

The test creates several example configurations to validate:

### Simple Math Tutor Agent
- Basic single-agent configuration
- Simple workflow with dependencies  
- Input/output validation
- Tool integration (calculator)

### Multi-Agent Data Analysis Process
- Multiple specialized agents (collector, analyzer, reporter)
- Complex dependency chains
- Budget controls and timeouts
- Step-by-step workflow execution

## Test Results

### Success Criteria
Tests pass when:
- All JSON configurations are valid
- Required schema fields are present
- Agent dependencies are properly structured
- Workflow steps have correct linkages
- Command structure is available
- Error handling works for invalid configurations

### Failure Scenarios
Tests fail when:
- Configuration files have syntax errors
- Required fields are missing from agent definitions
- Agent dependencies create circular references
- Workflow steps reference non-existent agents
- Framework commands are not available

## Legacy Test Cleanup

All legacy tests were removed as they were:
- Specific to code editing workflows
- Dependent on software development assumptions
- Focused on `ledit` command structure rather than generic framework
- Testing specific LLM outputs rather than framework functionality

The new tests are designed for the generic agent template system vision.

## Complete Test Suite

Current comprehensive test coverage:

### ✅ Implemented Tests
- **Generic Framework Test** (`test_generic_agent_framework.sh`): Core framework validation
- **LLM Provider Integration** (`test_llm_provider_integration.sh`): Multi-provider support  
- **Workflow Execution** (`test_workflow_execution.sh`): Complex dependency management
- **Error Handling** (`test_error_handling.sh`): Failure scenarios and recovery
- **Security & Validation** (`test_security_validation.sh`): Security controls and data validation

### 🚀 Future Test Expansion
Potential additional tests:
- **Performance Tests**: Multi-agent execution timing and resource usage
- **Real LLM Integration**: Tests with actual API calls (currently framework-focused)
- **Template Library Tests**: Validation of all pre-built agent configurations  
- **CLI Interface Tests**: Command-line argument and flag validation
- **Migration Tests**: Legacy configuration conversion utilities

## Test Philosophy Evolution

The test suite has evolved from legacy code-specific tests to framework-focused validation:

**Previous Approach** (Legacy):
- Tested specific coding outputs and file modifications
- Relied on LLM performance for test success/failure
- Focused on `ledit` command functionality

**Current Approach** (Generic Framework):
- Tests framework infrastructure and configuration validation
- Validates multi-agent orchestration and workflow execution
- Framework-agnostic testing independent of specific LLM outputs
- Comprehensive coverage of security, error handling, and provider integration