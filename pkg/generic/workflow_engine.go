package generic

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

// WorkflowEngine executes workflows
type WorkflowEngine struct {
	toolRegistry      *ToolRegistry
	llmClient         *LLMClient
	validator         *Validator
	templateEngine    *TemplateEngine
	transformPipeline *TransformPipeline
	logger            *slog.Logger
}

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine(workflows []Workflow, toolRegistry *ToolRegistry, llmClient *LLMClient, validator *Validator, logger *slog.Logger) (*WorkflowEngine, error) {
	templateEngine := NewTemplateEngine(logger)
	transformRegistry := NewTransformRegistry(logger)
	transformPipeline := NewTransformPipeline(transformRegistry, templateEngine, logger)

	return &WorkflowEngine{
		toolRegistry:      toolRegistry,
		llmClient:         llmClient,
		validator:         validator,
		templateEngine:    templateEngine,
		transformPipeline: transformPipeline,
		logger:            logger,
	}, nil
}

// Execute executes a workflow
func (we *WorkflowEngine) Execute(ctx context.Context, workflow *Workflow, execCtx *ExecutionContext) (interface{}, error) {
	we.logger.Info("Starting workflow execution", "workflow", workflow.Name)

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		we.logger.Info("Workflow execution completed",
			"workflow", workflow.Name,
			"duration", duration)
	}()

	// Build dependency graph
	dependencyGraph, err := we.buildDependencyGraph(workflow.Steps)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Execute steps in dependency order
	we.logger.Debug("Dependency graph", "total_levels", len(dependencyGraph))
	for i, level := range dependencyGraph {
		stepNames := make([]string, len(level))
		for j, step := range level {
			stepNames[j] = step.Name
		}
		we.logger.Debug("Dependency level", "level", i, "steps", stepNames)
	}

	executedSteps := make(map[string]*StepResult)
	for i, stepGroup := range dependencyGraph {
		we.logger.Debug("Executing dependency level", "level", i, "step_count", len(stepGroup))
		// Steps in the same group can be executed in parallel
		if len(stepGroup) == 1 {
			// Single step execution
			result, err := we.executeStep(ctx, stepGroup[0], execCtx, executedSteps)
			if err != nil {
				if !stepGroup[0].ContinueOnError {
					return nil, fmt.Errorf("step %s failed: %w", stepGroup[0].Name, err)
				}
				we.logger.Warn("Step failed but continuing", "step", stepGroup[0].Name, "error", err)
			}
			executedSteps[stepGroup[0].Name] = result
		} else {
			// Parallel execution (simplified - real implementation would use goroutines)
			for _, step := range stepGroup {
				result, err := we.executeStep(ctx, step, execCtx, executedSteps)
				if err != nil {
					if !step.ContinueOnError {
						return nil, fmt.Errorf("step %s failed: %w", step.Name, err)
					}
					we.logger.Warn("Step failed but continuing", "step", step.Name, "error", err)
				}
				executedSteps[step.Name] = result
			}
		}
	}

	// Prepare final result
	results := make(map[string]interface{})
	for name, result := range executedSteps {
		if result.Success {
			results[name] = result.Output
		}
	}

	// Update execution context metrics
	execCtx.Metrics.TotalSteps = len(workflow.Steps)
	for _, result := range executedSteps {
		if result.Success {
			execCtx.Metrics.SuccessfulSteps++
		} else {
			execCtx.Metrics.FailedSteps++
		}
	}

	return results, nil
}

// executeStep executes a single workflow step
func (we *WorkflowEngine) executeStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (*StepResult, error) {
	// Check conditions before executing
	if len(step.Conditions) > 0 {
		conditionsMet, err := we.evaluateStepConditions(step.Conditions, previousResults, execCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate step conditions for %s: %w", step.Name, err)
		}
		if !conditionsMet {
			we.logger.Info("Step conditions not met, skipping", "step", step.Name)
			return &StepResult{
				StepName:      step.Name,
				Success:       true,
				Output:        "skipped - conditions not met",
				ExecutionTime: 0,
				Metadata:      map[string]interface{}{"skipped": true},
			}, nil
		}
	}

	we.logger.Info("Executing step", "step", step.Name, "type", step.Type)

	startTime := time.Now()
	result := &StepResult{
		StepName:      step.Name,
		ExecutionTime: 0,
		Metadata:      make(map[string]interface{}),
	}

	// Execute pre-transforms (context transforms)
	err := we.transformPipeline.ExecutePreTransforms(step, previousResults, execCtx)
	if err != nil {
		result.Success = false
		result.Error = err
		return result, fmt.Errorf("pre-transform failed: %w", err)
	}

	defer func() {
		result.ExecutionTime = time.Since(startTime)
		execCtx.StepResults[step.Name] = result
	}()

	// Execute with retry logic
	maxAttempts := 1
	if step.Retry.MaxAttempts > 0 {
		maxAttempts = step.Retry.MaxAttempts
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			we.logger.Info("Retrying step", "step", step.Name, "attempt", attempt)
			// TODO: Implement backoff strategy
		}

		var output interface{}
		var err error

		switch step.Type {
		case "tool":
			output, err = we.executeToolStep(ctx, step, execCtx, previousResults)
		case "llm":
			output, err = we.executeLLMStep(ctx, step, execCtx, previousResults)
		case "llm_display":
			output, err = we.executeLLMDisplayStep(ctx, step, execCtx, previousResults)
		case "script":
			output, err = we.executeScriptStep(ctx, step, execCtx, previousResults)
		case "condition":
			output, err = we.executeConditionStep(ctx, step, execCtx, previousResults)
		case "loop":
			output, err = we.executeLoopStep(ctx, step, execCtx, previousResults)
		case "parallel":
			output, err = we.executeParallelStep(ctx, step, execCtx, previousResults)
		default:
			err = fmt.Errorf("unsupported step type: %s", step.Type)
		}

		if err == nil {
			result.Success = true
			result.Output = output
			result.ExecutionTime = time.Since(startTime)

			// Execute post-transforms
			postErr := we.transformPipeline.ExecutePostTransforms(step, result, previousResults, execCtx)
			if postErr != nil {
				we.logger.Warn("Post-transform failed", "step", step.Name, "error", postErr)
				// Don't fail the step for post-transform errors, just log them
			}

			return result, nil
		}

		lastErr = err
		we.logger.Warn("Step attempt failed", "step", step.Name, "attempt", attempt, "error", err)
	}

	result.Success = false
	result.Error = lastErr
	result.ExecutionTime = time.Since(startTime)
	return result, lastErr
}

// executeToolStep executes a tool step
func (we *WorkflowEngine) executeToolStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	toolName, ok := step.Config["tool"].(string)
	if !ok {
		return nil, fmt.Errorf("tool name not specified in step config")
	}

	tool, exists := we.toolRegistry.GetTool(toolName)
	if !exists {
		return nil, fmt.Errorf("tool %s not found", toolName)
	}

	// Prepare parameters by merging step config and context data
	params := make(map[string]interface{})

	// Add step config parameters with template processing
	if stepParams, ok := step.Config["params"].(map[string]interface{}); ok {
		for k, v := range stepParams {
			// Process string values through template engine
			if stringVal, ok := v.(string); ok {
				renderedVal, err := we.templateEngine.RenderTemplate(stringVal, previousResults, execCtx)
				if err != nil {
					we.logger.Warn("Failed to render template in tool parameter", "step", step.Name, "param", k, "error", err)
					params[k] = v // Use original value if template fails
				} else {
					params[k] = renderedVal
				}
			} else {
				params[k] = v
			}
		}
	}

	// Add context data
	for k, v := range execCtx.Data {
		params[k] = v
	}

	return tool.Execute(ctx, params)
}

// executeLLMStep executes an LLM step
func (we *WorkflowEngine) executeLLMStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	prompt, ok := step.Config["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt not specified in step config")
	}

	// Template rendering for prompt with context variables
	renderedPrompt, err := we.templateEngine.RenderTemplate(prompt, previousResults, execCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt template: %w", err)
	}

	// TODO: Add system prompt handling

	response, err := we.llmClient.Complete(ctx, renderedPrompt)
	if err != nil {
		return nil, err
	}

	// Update metrics
	execCtx.Metrics.LLMTokensUsed += response.TokensUsed
	execCtx.Metrics.LLMCost += response.Cost

	return response.Content, nil
}

// executeLLMDisplayStep executes an LLM step and displays the output to the user
func (we *WorkflowEngine) executeLLMDisplayStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	prompt, ok := step.Config["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt not specified in step config")
	}

	// Template rendering for prompt with context variables
	renderedPrompt, err := we.templateEngine.RenderTemplate(prompt, previousResults, execCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt template: %w", err)
	}

	// TODO: Add system prompt handling

	response, err := we.llmClient.Complete(ctx, renderedPrompt)
	if err != nil {
		return nil, err
	}

	// Update metrics
	execCtx.Metrics.LLMTokensUsed += response.TokensUsed
	execCtx.Metrics.LLMCost += response.Cost

	// Display the LLM response to the user
	fmt.Println("=== LLM ANALYSIS RESULTS ===")
	fmt.Println()
	fmt.Print(response.Content)
	fmt.Println()
	fmt.Println("=== END ANALYSIS RESULTS ===")
	fmt.Println()

	return response.Content, nil
}

// executeConditionStep executes a condition step
func (we *WorkflowEngine) executeConditionStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	// TODO: Implement condition evaluation
	return true, nil
}

// executeLoopStep executes a loop step
func (we *WorkflowEngine) executeLoopStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	// TODO: Implement loop execution
	return "loop completed", nil
}

// executeParallelStep executes parallel steps
func (we *WorkflowEngine) executeParallelStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	// TODO: Implement parallel execution
	return "parallel execution completed", nil
}

// buildDependencyGraph builds a dependency graph for workflow steps using topological sorting
func (we *WorkflowEngine) buildDependencyGraph(steps []Step) ([][]Step, error) {
	// Create a map for quick step lookup
	stepMap := make(map[string]Step)
	for _, step := range steps {
		stepMap[step.Name] = step
	}

	// Track in-degree (number of dependencies) for each step
	inDegree := make(map[string]int)
	for _, step := range steps {
		inDegree[step.Name] = len(step.DependsOn)
	}

	var graph [][]Step
	remaining := make(map[string]Step)
	for _, step := range steps {
		remaining[step.Name] = step
	}

	// Process steps in dependency order
	for len(remaining) > 0 {
		// Find steps with no remaining dependencies
		currentLevel := make([]Step, 0)
		for name, step := range remaining {
			if inDegree[name] == 0 {
				currentLevel = append(currentLevel, step)
			}
		}

		// If no steps can be processed, we have a circular dependency
		if len(currentLevel) == 0 {
			remainingNames := make([]string, 0, len(remaining))
			for name := range remaining {
				remainingNames = append(remainingNames, name)
			}
			return nil, fmt.Errorf("circular dependency detected among steps: %v", remainingNames)
		}

		// Remove processed steps and update dependencies
		for _, step := range currentLevel {
			delete(remaining, step.Name)

			// Reduce in-degree for steps that depend on this one
			for otherName := range remaining {
				otherStep := remaining[otherName]
				for _, dep := range otherStep.DependsOn {
					if dep == step.Name {
						inDegree[otherName]--
					}
				}
			}
		}

		graph = append(graph, currentLevel)
	}

	return graph, nil
}

// executeScriptStep executes a script step with security validation
func (we *WorkflowEngine) executeScriptStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	script, ok := step.Config["script"].(string)
	if !ok {
		return nil, fmt.Errorf("script not specified in step config")
	}

	// Determine if this is a trusted source (config-defined) or untrusted (LLM-generated)
	isTrustedSource := false
	if source, ok := step.Config["source"].(string); ok && source == "config" {
		isTrustedSource = true
	}

	// Create security context
	securityContext := SecurityContext{
		IsTrustedSource: isTrustedSource,
		MaxFileSize:     10 * 1024, // 10KB limit for scripts
	}

	// Add custom blocked commands if specified
	if blocked, ok := step.Config["blocked_commands"].([]interface{}); ok {
		for _, cmd := range blocked {
			if cmdStr, ok := cmd.(string); ok {
				securityContext.BlockedCommands = append(securityContext.BlockedCommands, cmdStr)
			}
		}
	}

	we.logger.Info("Validating script",
		"step", step.Name,
		"trusted_source", isTrustedSource,
		"script_length", len(script))

	// Validate script security
	validationResult, err := we.validator.ValidateScript(script, securityContext)
	if err != nil {
		return nil, fmt.Errorf("script validation failed: %w", err)
	}

	if !validationResult.IsSecure {
		we.logger.Error("Script failed security validation",
			"step", step.Name,
			"violations", validationResult.Violations)
		return nil, fmt.Errorf("script security validation failed: %v", validationResult.Violations)
	}

	if len(validationResult.Warnings) > 0 {
		we.logger.Warn("Script validation warnings",
			"step", step.Name,
			"warnings", validationResult.Warnings)
	}

	// Create secure temporary file
	tempFile, err := we.validator.CreateSecureTempFile(validationResult.SanitizedScript, "agent-script-")
	if err != nil {
		return nil, fmt.Errorf("failed to create secure temp file: %w", err)
	}
	defer func() {
		if cleanupErr := we.validator.CleanupTempFile(tempFile); cleanupErr != nil {
			we.logger.Error("Failed to cleanup temp file", "file", tempFile, "error", cleanupErr)
		}
	}()

	// Execute script with timeout and resource limits
	timeout := 30 * time.Second
	if configTimeout, ok := step.Config["timeout"].(float64); ok {
		timeout = time.Duration(configTimeout) * time.Second
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use bash to execute the script
	cmd := exec.CommandContext(ctxWithTimeout, "bash", tempFile)

	// Set environment variables from execution context
	env := os.Environ()
	for k, v := range execCtx.Data {
		if strVal, ok := v.(string); ok {
			env = append(env, fmt.Sprintf("AGENT_%s=%s", k, strVal))
		}
	}
	cmd.Env = env

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		we.logger.Error("Script execution failed",
			"step", step.Name,
			"error", err,
			"output", string(output))
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	we.logger.Info("Script executed successfully",
		"step", step.Name,
		"output_length", len(output))

	return string(output), nil
}

// evaluateStepConditions evaluates conditions for a step to determine if it should execute
func (we *WorkflowEngine) evaluateStepConditions(conditions []StepCondition, previousResults map[string]*StepResult, execCtx *ExecutionContext) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}

	// All conditions must be met (AND logic)
	for _, condition := range conditions {
		met, err := we.evaluateSingleCondition(condition, previousResults, execCtx)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate condition %s %s %s: %w", condition.Field, condition.Operator, condition.Value, err)
		}
		if !met {
			we.logger.Debug("Condition not met", "field", condition.Field, "operator", condition.Operator, "value", condition.Value)
			return false, nil
		}
	}

	return true, nil
}

// evaluateSingleCondition evaluates a single condition
func (we *WorkflowEngine) evaluateSingleCondition(condition StepCondition, previousResults map[string]*StepResult, execCtx *ExecutionContext) (bool, error) {
	// Get the field value from previous results
	var fieldValue interface{}

	// Check if the field exists in previous results
	if result, exists := previousResults[condition.Field]; exists {
		fieldValue = result.Output

		// Handle tool outputs that return maps (like ask_user)
		if outputMap, ok := fieldValue.(map[string]interface{}); ok {
			if response, hasResponse := outputMap["response"]; hasResponse {
				fieldValue = response
			}
		}
	} else {
		// Field doesn't exist, treat as empty
		fieldValue = ""
	}

	// Convert to string for comparison
	fieldStr := fmt.Sprintf("%v", fieldValue)

	// Log condition evaluation for debugging
	we.logger.Debug("Evaluating condition",
		"field", condition.Field,
		"operator", condition.Operator,
		"expected", condition.Value,
		"actual", fieldStr)

	// Evaluate based on operator
	result := false
	switch condition.Operator {
	case "equals":
		result = fieldStr == condition.Value
	case "not_equals":
		result = fieldStr != condition.Value
	case "contains":
		result = strings.Contains(fieldStr, condition.Value)
	case "not_contains":
		result = !strings.Contains(fieldStr, condition.Value)
	case "empty":
		result = fieldStr == ""
	case "not_empty":
		result = fieldStr != ""
	default:
		return false, fmt.Errorf("unsupported operator: %s", condition.Operator)
	}

	we.logger.Debug("Condition evaluation result",
		"field", condition.Field,
		"result", result)

	return result, nil
}

// Legacy renderTemplate function - now handled by TemplateEngine
// This is kept for backward compatibility if needed
