package generic

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// WorkflowEngine executes workflows
type WorkflowEngine struct {
	toolRegistry *ToolRegistry
	llmClient    *LLMClient
	logger       *slog.Logger
}

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine(workflows []Workflow, toolRegistry *ToolRegistry, llmClient *LLMClient, logger *slog.Logger) (*WorkflowEngine, error) {
	return &WorkflowEngine{
		toolRegistry: toolRegistry,
		llmClient:    llmClient,
		logger:       logger,
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
	executedSteps := make(map[string]*StepResult)
	for _, stepGroup := range dependencyGraph {
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
	startTime := time.Now()
	we.logger.Info("Executing step", "step", step.Name, "type", step.Type)

	result := &StepResult{
		StepName: step.Name,
		Success:  false,
		Metadata: make(map[string]interface{}),
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
			return result, nil
		}

		lastErr = err
		we.logger.Warn("Step attempt failed", "step", step.Name, "attempt", attempt, "error", err)
	}

	result.Error = lastErr
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

	// Add step config parameters
	if stepParams, ok := step.Config["params"].(map[string]interface{}); ok {
		for k, v := range stepParams {
			params[k] = v
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

	// TODO: Template rendering for prompt with context variables
	// TODO: Add system prompt handling

	response, err := we.llmClient.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Update metrics
	execCtx.Metrics.LLMTokensUsed += response.TokensUsed
	execCtx.Metrics.LLMCost += response.Cost

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

// buildDependencyGraph builds a dependency graph for workflow steps
func (we *WorkflowEngine) buildDependencyGraph(steps []Step) ([][]Step, error) {
	// Simple implementation - in practice, this would do proper topological sorting
	var graph [][]Step

	// Group steps by their dependencies
	independentSteps := make([]Step, 0)
	dependentSteps := make([]Step, 0)

	for _, step := range steps {
		if len(step.DependsOn) == 0 {
			independentSteps = append(independentSteps, step)
		} else {
			dependentSteps = append(dependentSteps, step)
		}
	}

	if len(independentSteps) > 0 {
		graph = append(graph, independentSteps)
	}

	if len(dependentSteps) > 0 {
		graph = append(graph, dependentSteps)
	}

	return graph, nil
}
