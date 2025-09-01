package generic

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestNewWorkflowEngine(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create minimal required components
	toolRegistry, _ := NewToolRegistry(map[string]Tool{}, &Security{Enabled: false}, logger)
	llmClient, _ := NewLLMClient(LLMConfig{Provider: "openai", Model: "gpt-4"}, logger)
	validator, _ := NewValidator(Validation{Enabled: false}, logger)

	workflows := []Workflow{
		{
			Name: "test-workflow",
			Steps: []Step{
				{Name: "step1", Type: "llm"},
			},
		},
	}

	engine, err := NewWorkflowEngine(workflows, toolRegistry, llmClient, validator, logger)

	if err != nil {
		t.Errorf("Unexpected error creating workflow engine: %v", err)
	}

	if engine == nil {
		t.Error("Expected workflow engine to be created")
	}

	if engine.toolRegistry != toolRegistry {
		t.Error("Expected tool registry to be set")
	}

	if engine.llmClient != llmClient {
		t.Error("Expected LLM client to be set")
	}

	if engine.validator != validator {
		t.Error("Expected validator to be set")
	}

	if engine.templateEngine == nil {
		t.Error("Expected template engine to be initialized")
	}

	if engine.transformPipeline == nil {
		t.Error("Expected transform pipeline to be initialized")
	}
}

func TestBuildDependencyGraph(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	toolRegistry, _ := NewToolRegistry(map[string]Tool{}, &Security{Enabled: false}, logger)
	llmClient, _ := NewLLMClient(LLMConfig{Provider: "openai", Model: "gpt-4"}, logger)
	validator, _ := NewValidator(Validation{Enabled: false}, logger)

	engine, _ := NewWorkflowEngine([]Workflow{}, toolRegistry, llmClient, validator, logger)

	tests := []struct {
		name           string
		steps          []Step
		expectError    bool
		expectedLevels int
		errorMsg       string
	}{
		{
			name: "no dependencies",
			steps: []Step{
				{Name: "step1", Type: "llm"},
				{Name: "step2", Type: "llm"},
			},
			expectError:    false,
			expectedLevels: 1, // All steps can run in parallel
		},
		{
			name: "linear dependencies",
			steps: []Step{
				{Name: "step1", Type: "llm"},
				{Name: "step2", Type: "llm", DependsOn: []string{"step1"}},
				{Name: "step3", Type: "llm", DependsOn: []string{"step2"}},
			},
			expectError:    false,
			expectedLevels: 3, // Each step depends on the previous
		},
		{
			name: "parallel with final step",
			steps: []Step{
				{Name: "step1", Type: "llm"},
				{Name: "step2", Type: "llm"},
				{Name: "final", Type: "llm", DependsOn: []string{"step1", "step2"}},
			},
			expectError:    false,
			expectedLevels: 2, // step1,step2 parallel, then final
		},
		{
			name: "circular dependency",
			steps: []Step{
				{Name: "step1", Type: "llm", DependsOn: []string{"step2"}},
				{Name: "step2", Type: "llm", DependsOn: []string{"step1"}},
			},
			expectError: true,
			errorMsg:    "circular dependency",
		},
		{
			name: "self dependency",
			steps: []Step{
				{Name: "step1", Type: "llm", DependsOn: []string{"step1"}},
			},
			expectError: true,
			errorMsg:    "circular dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, err := engine.buildDependencyGraph(tt.steps)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !containsError(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(graph) != tt.expectedLevels {
					t.Errorf("Expected %d levels, got %d", tt.expectedLevels, len(graph))
				}

				// Verify all steps are included
				stepCount := 0
				for _, level := range graph {
					stepCount += len(level)
				}
				if stepCount != len(tt.steps) {
					t.Errorf("Expected %d total steps, got %d", len(tt.steps), stepCount)
				}
			}
		})
	}
}

func TestEvaluateStepConditions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	toolRegistry, _ := NewToolRegistry(map[string]Tool{}, &Security{Enabled: false}, logger)
	llmClient, _ := NewLLMClient(LLMConfig{Provider: "openai", Model: "gpt-4"}, logger)
	validator, _ := NewValidator(Validation{Enabled: false}, logger)

	engine, _ := NewWorkflowEngine([]Workflow{}, toolRegistry, llmClient, validator, logger)

	execCtx := &ExecutionContext{
		Data:      make(map[string]interface{}),
		Variables: make(map[string]string),
	}

	previousResults := map[string]*StepResult{
		"step1": {
			StepName: "step1",
			Success:  true,
			Output:   "success result",
		},
		"step2": {
			StepName: "step2",
			Success:  true,
			Output:   map[string]interface{}{"response": "user_said_yes"},
		},
	}

	tests := []struct {
		name        string
		conditions  []StepCondition
		expected    bool
		expectError bool
	}{
		{
			name:       "no conditions",
			conditions: []StepCondition{},
			expected:   true,
		},
		{
			name: "equals condition - match",
			conditions: []StepCondition{
				{Field: "step1", Operator: "equals", Value: "success result"},
			},
			expected: true,
		},
		{
			name: "equals condition - no match",
			conditions: []StepCondition{
				{Field: "step1", Operator: "equals", Value: "different result"},
			},
			expected: false,
		},
		{
			name: "contains condition - match",
			conditions: []StepCondition{
				{Field: "step1", Operator: "contains", Value: "success"},
			},
			expected: true,
		},
		{
			name: "not_empty condition - match",
			conditions: []StepCondition{
				{Field: "step1", Operator: "not_empty", Value: ""},
			},
			expected: true,
		},
		{
			name: "empty condition - no match (field exists)",
			conditions: []StepCondition{
				{Field: "step1", Operator: "empty", Value: ""},
			},
			expected: false,
		},
		{
			name: "empty condition - match (field doesn't exist)",
			conditions: []StepCondition{
				{Field: "nonexistent", Operator: "empty", Value: ""},
			},
			expected: true,
		},
		{
			name: "map output response extraction",
			conditions: []StepCondition{
				{Field: "step2", Operator: "equals", Value: "user_said_yes"},
			},
			expected: true,
		},
		{
			name: "unsupported operator",
			conditions: []StepCondition{
				{Field: "step1", Operator: "unsupported", Value: "test"},
			},
			expected:    false,
			expectError: true,
		},
		{
			name: "multiple conditions - all match",
			conditions: []StepCondition{
				{Field: "step1", Operator: "contains", Value: "success"},
				{Field: "step1", Operator: "not_empty", Value: ""},
			},
			expected: true,
		},
		{
			name: "multiple conditions - one fails",
			conditions: []StepCondition{
				{Field: "step1", Operator: "contains", Value: "success"},
				{Field: "step1", Operator: "equals", Value: "different"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.evaluateStepConditions(tt.conditions, previousResults, execCtx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestEvaluateSingleCondition(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	toolRegistry, _ := NewToolRegistry(map[string]Tool{}, &Security{Enabled: false}, logger)
	llmClient, _ := NewLLMClient(LLMConfig{Provider: "openai", Model: "gpt-4"}, logger)
	validator, _ := NewValidator(Validation{Enabled: false}, logger)

	engine, _ := NewWorkflowEngine([]Workflow{}, toolRegistry, llmClient, validator, logger)

	execCtx := &ExecutionContext{
		Data:      make(map[string]interface{}),
		Variables: make(map[string]string),
	}

	previousResults := map[string]*StepResult{
		"test_step": {
			StepName: "test_step",
			Success:  true,
			Output:   "test value",
		},
	}

	tests := []struct {
		name        string
		condition   StepCondition
		expected    bool
		expectError bool
	}{
		{
			name:      "equals - true",
			condition: StepCondition{Field: "test_step", Operator: "equals", Value: "test value"},
			expected:  true,
		},
		{
			name:      "equals - false",
			condition: StepCondition{Field: "test_step", Operator: "equals", Value: "different"},
			expected:  false,
		},
		{
			name:      "not_equals - true",
			condition: StepCondition{Field: "test_step", Operator: "not_equals", Value: "different"},
			expected:  true,
		},
		{
			name:      "contains - true",
			condition: StepCondition{Field: "test_step", Operator: "contains", Value: "test"},
			expected:  true,
		},
		{
			name:      "contains - false",
			condition: StepCondition{Field: "test_step", Operator: "contains", Value: "missing"},
			expected:  false,
		},
		{
			name:      "not_contains - true",
			condition: StepCondition{Field: "test_step", Operator: "not_contains", Value: "missing"},
			expected:  true,
		},
		{
			name:      "not_empty - true",
			condition: StepCondition{Field: "test_step", Operator: "not_empty", Value: ""},
			expected:  true,
		},
		{
			name:      "empty - false (field exists)",
			condition: StepCondition{Field: "test_step", Operator: "empty", Value: ""},
			expected:  false,
		},
		{
			name:      "empty - true (field missing)",
			condition: StepCondition{Field: "missing_step", Operator: "empty", Value: ""},
			expected:  true,
		},
		{
			name:        "unsupported operator",
			condition:   StepCondition{Field: "test_step", Operator: "regex", Value: ".*"},
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.evaluateSingleCondition(tt.condition, previousResults, execCtx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestWorkflowEngineExecute(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	toolRegistry, _ := NewToolRegistry(map[string]Tool{}, &Security{Enabled: false}, logger)
	llmClient, _ := NewLLMClient(LLMConfig{Provider: "openai", Model: "gpt-4"}, logger)
	validator, _ := NewValidator(Validation{Enabled: false}, logger)

	engine, _ := NewWorkflowEngine([]Workflow{}, toolRegistry, llmClient, validator, logger)

	tests := []struct {
		name        string
		workflow    *Workflow
		expectError bool
		errorMsg    string
	}{
		{
			name: "simple workflow with single step",
			workflow: &Workflow{
				Name: "simple-workflow",
				Steps: []Step{
					{
						Name: "test-step",
						Type: "condition", // Use a step type that won't fail
					},
				},
			},
			expectError: false,
		},
		{
			name: "workflow with dependencies",
			workflow: &Workflow{
				Name: "dependency-workflow",
				Steps: []Step{
					{
						Name: "step1",
						Type: "condition",
					},
					{
						Name:      "step2",
						Type:      "condition",
						DependsOn: []string{"step1"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "workflow with circular dependency",
			workflow: &Workflow{
				Name: "circular-workflow",
				Steps: []Step{
					{
						Name:      "step1",
						Type:      "condition",
						DependsOn: []string{"step2"},
					},
					{
						Name:      "step2",
						Type:      "condition",
						DependsOn: []string{"step1"},
					},
				},
			},
			expectError: true,
			errorMsg:    "circular dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execCtx := &ExecutionContext{
				Context:     context.Background(),
				SessionID:   "test-session",
				StartTime:   time.Now(),
				Data:        make(map[string]interface{}),
				Variables:   make(map[string]string),
				StepResults: make(map[string]*StepResult),
				Metrics:     &ExecutionMetrics{},
			}

			result, err := engine.Execute(context.Background(), tt.workflow, execCtx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !containsError(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected result to be returned when no error")
				}

				// Check metrics were updated
				if execCtx.Metrics.TotalSteps != len(tt.workflow.Steps) {
					t.Errorf("Expected total steps %d, got %d", len(tt.workflow.Steps), execCtx.Metrics.TotalSteps)
				}
			}
		})
	}
}

func TestExecuteStep(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	toolRegistry, _ := NewToolRegistry(map[string]Tool{}, &Security{Enabled: false}, logger)
	llmClient, _ := NewLLMClient(LLMConfig{Provider: "openai", Model: "gpt-4"}, logger)
	validator, _ := NewValidator(Validation{Enabled: false}, logger)

	engine, _ := NewWorkflowEngine([]Workflow{}, toolRegistry, llmClient, validator, logger)

	execCtx := &ExecutionContext{
		Context:     context.Background(),
		SessionID:   "test-session",
		StartTime:   time.Now(),
		Data:        make(map[string]interface{}),
		Variables:   make(map[string]string),
		StepResults: make(map[string]*StepResult),
		Metrics:     &ExecutionMetrics{},
	}

	previousResults := make(map[string]*StepResult)

	tests := []struct {
		name        string
		step        Step
		expectError bool
		skipReason  string
	}{
		{
			name: "condition step",
			step: Step{
				Name: "test-condition",
				Type: "condition",
			},
			expectError: false,
		},
		{
			name: "loop step",
			step: Step{
				Name: "test-loop",
				Type: "loop",
			},
			expectError: false,
		},
		{
			name: "parallel step",
			step: Step{
				Name: "test-parallel",
				Type: "parallel",
			},
			expectError: false,
		},
		{
			name: "unsupported step type",
			step: Step{
				Name: "unsupported",
				Type: "unknown_type",
			},
			expectError: true,
		},
		{
			name: "step with conditions not met",
			step: Step{
				Name: "conditional-step",
				Type: "condition",
				Conditions: []StepCondition{
					{Field: "nonexistent", Operator: "equals", Value: "required_value"},
				},
			},
			expectError: false,
			skipReason:  "conditions not met",
		},
		{
			name: "step with retry",
			step: Step{
				Name: "retry-step",
				Type: "condition",
				Retry: RetryConfig{
					MaxAttempts: 3,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.executeStep(context.Background(), tt.step, execCtx, previousResults)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected result to be returned")
				}
				if result.StepName != tt.step.Name {
					t.Errorf("Expected step name '%s', got '%s'", tt.step.Name, result.StepName)
				}

				// Check if step was skipped due to conditions
				if tt.skipReason != "" {
					if skipped, ok := result.Metadata["skipped"].(bool); !ok || !skipped {
						t.Error("Expected step to be skipped due to conditions")
					}
				} else {
					// Step should have succeeded for supported types
					if !result.Success && tt.step.Type != "unknown_type" {
						t.Errorf("Expected step to succeed, but got error: %v", result.Error)
					}
				}

				// Check execution time was recorded
				if result.ExecutionTime < 0 {
					t.Error("Expected positive execution time")
				}
			}
		})
	}
}
