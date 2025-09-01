package generic

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
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

	// Note: Result storage moved to after post-transforms complete

	// Execute with retry logic
	maxAttempts := 1
	if step.Retry.MaxAttempts > 0 {
		maxAttempts = step.Retry.MaxAttempts
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			we.logger.Info("Retrying step", "step", step.Name, "attempt", attempt)
			
			// Implement exponential backoff with jitter
			backoffDuration := time.Duration(attempt-1) * time.Second * time.Duration(1<<uint(attempt-2))
			if backoffDuration > 30*time.Second {
				backoffDuration = 30 * time.Second // Cap at 30 seconds
			}
			
			// Add jitter (random delay up to 1 second)
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			totalDelay := backoffDuration + jitter
			
			we.logger.Debug("Applying backoff delay", "delay", totalDelay, "attempt", attempt)
			
			select {
			case <-time.After(totalDelay):
				// Continue with retry
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
			}
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
		case "llm_with_tools":
			output, err = we.executeLLMWithToolsStep(ctx, step, execCtx, previousResults)
		case "display":
			output, err = we.executeDisplayStep(ctx, step, execCtx, previousResults)
		case "script":
			output, err = we.executeScriptStep(ctx, step, execCtx, previousResults)
		case "condition":
			output, err = we.executeConditionStep(ctx, step, execCtx, previousResults)
		case "loop":
			output, err = we.executeLoopStep(ctx, step, execCtx, previousResults)
			// Add loop metadata
			if err == nil && output != nil {
				if loopResult, ok := output.(*LoopResult); ok {
					result.Metadata["iterations"] = loopResult.Iterations
					result.Metadata["break_reason"] = loopResult.BreakReason
				}
			}
		case "parallel":
			output, err = we.executeParallelStep(ctx, step, execCtx, previousResults)
		default:
			err = fmt.Errorf("unsupported step type: %s", step.Type)
		}

		if err == nil {
			result.Success = true
			result.Output = output

			// Execute post-transforms
			postErr := we.transformPipeline.ExecutePostTransforms(step, result, previousResults, execCtx)
			if postErr != nil {
				we.logger.Warn("Post-transform failed", "step", step.Name, "error", postErr)
				// Don't fail the step for post-transform errors, just log them
			}

			// Set execution time and store result in execution context AFTER post-transforms complete
			// This ensures dependent steps have access to post-transform data
			result.ExecutionTime = time.Since(startTime)
			execCtx.StepResults[step.Name] = result

			return result, nil
		}

		lastErr = err
		we.logger.Warn("Step attempt failed", "step", step.Name, "attempt", attempt, "error", err)
	}

	result.Success = false
	result.Error = lastErr
	result.ExecutionTime = time.Since(startTime)

	// Store failed result in execution context for completeness
	execCtx.StepResults[step.Name] = result

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

	// Check for system prompt in step config
	var response *LLMResponse
	
	if systemPrompt, ok := step.Config["system_prompt"].(string); ok && systemPrompt != "" {
		// Render system prompt template if provided
		renderedSystemPrompt, err := we.templateEngine.RenderTemplate(systemPrompt, previousResults, execCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to render system prompt template: %w", err)
		}
		response, err = we.llmClient.CompleteWithSystem(ctx, renderedSystemPrompt, renderedPrompt)
	} else {
		response, err = we.llmClient.Complete(ctx, renderedPrompt)
	}
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

	// Check for system prompt in step config
	var response *LLMResponse
	
	if systemPrompt, ok := step.Config["system_prompt"].(string); ok && systemPrompt != "" {
		// Render system prompt template if provided
		renderedSystemPrompt, err := we.templateEngine.RenderTemplate(systemPrompt, previousResults, execCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to render system prompt template: %w", err)
		}
		response, err = we.llmClient.CompleteWithSystem(ctx, renderedSystemPrompt, renderedPrompt)
	} else {
		response, err = we.llmClient.Complete(ctx, renderedPrompt)
	}
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

// LLMWithToolsConfig represents configuration for LLM with tools step
type LLMWithToolsConfig struct {
	MaxToolCalls    int      `json:"max_tool_calls"`
	AllowedTools    []string `json:"allowed_tools"`
	AllowedPaths    []string `json:"allowed_paths"`
	MaxFileSize     int      `json:"max_file_size"`
	FailOnToolError bool     `json:"fail_on_tool_error"`
}

// ToolExecution represents a single tool execution
type ToolExecution struct {
	Tool    string                 `json:"tool"`
	Params  map[string]interface{} `json:"params"`
	Result  string                 `json:"result"`
	Success bool                   `json:"success"`
	Error   string                 `json:"error,omitempty"`
}

// LLMWithToolsResult represents the result of LLM with tools execution
type LLMWithToolsResult struct {
	FinalResponse  string          `json:"final_response"`
	ToolExecutions []ToolExecution `json:"tool_executions"`
	ToolCallsUsed  int             `json:"tool_calls_used"`
	TotalTokens    int             `json:"total_tokens"`
	TotalCost      float64         `json:"total_cost"`
}

// executeLLMWithToolsStep executes an LLM step with tool access and proper controls
func (we *WorkflowEngine) executeLLMWithToolsStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	prompt, ok := step.Config["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt not specified in step config")
	}

	// Parse configuration with defaults
	config := LLMWithToolsConfig{
		MaxToolCalls:    3,     // Default max tool calls to prevent loops
		MaxFileSize:     10240, // 10KB default
		FailOnToolError: true,  // Fail by default on tool errors
	}

	// Override with step config if provided
	if configMap, ok := step.Config["tool_config"].(map[string]interface{}); ok {
		if maxCalls, ok := configMap["max_tool_calls"].(float64); ok {
			config.MaxToolCalls = int(maxCalls)
		}
		if maxSize, ok := configMap["max_file_size"].(float64); ok {
			config.MaxFileSize = int(maxSize)
		}
		if failOnError, ok := configMap["fail_on_tool_error"].(bool); ok {
			config.FailOnToolError = failOnError
		}
		if tools, ok := configMap["allowed_tools"].([]interface{}); ok {
			for _, tool := range tools {
				if toolStr, ok := tool.(string); ok {
					config.AllowedTools = append(config.AllowedTools, toolStr)
				}
			}
		}
		if paths, ok := configMap["allowed_paths"].([]interface{}); ok {
			for _, path := range paths {
				if pathStr, ok := path.(string); ok {
					config.AllowedPaths = append(config.AllowedPaths, pathStr)
				}
			}
		}
	}

	// Template rendering for prompt with context variables
	renderedPrompt, err := we.templateEngine.RenderTemplate(prompt, previousResults, execCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt template: %w", err)
	}

	// Execute LLM with tools in a controlled manner
	result, err := we.executeLLMWithToolsControlled(ctx, renderedPrompt, config, execCtx)
	if err != nil {
		return nil, fmt.Errorf("LLM with tools execution failed: %w", err)
	}

	// Log the result instead of using fmt.Println
	we.logger.Info("LLM with tools step completed",
		"step", step.Name,
		"tool_calls_made", result.ToolCallsUsed,
		"response_length", len(result.FinalResponse))

	// Display the response (keeping display for now but using structured logging)
	fmt.Println("=== LLM ANALYSIS WITH TOOLS ===")
	fmt.Println()
	fmt.Print(result.FinalResponse)
	fmt.Println()
	if len(result.ToolExecutions) > 0 {
		fmt.Println("=== TOOL EXECUTIONS ===")
		for _, execution := range result.ToolExecutions {
			fmt.Printf("Tool: %s\nResult: %s\n\n", execution.Tool, execution.Result)
		}
	}
	fmt.Println("=== END ANALYSIS ===")
	fmt.Println()

	// Update metrics
	execCtx.Metrics.LLMTokensUsed += result.TotalTokens
	execCtx.Metrics.LLMCost += result.TotalCost

	return result.FinalResponse, nil
}

// executeLLMWithToolsControlled executes LLM with proper tool controls and security
func (we *WorkflowEngine) executeLLMWithToolsControlled(ctx context.Context, prompt string, config LLMWithToolsConfig, execCtx *ExecutionContext) (*LLMWithToolsResult, error) {
	result := &LLMWithToolsResult{
		ToolExecutions: []ToolExecution{},
		ToolCallsUsed:  0,
	}

	// Get initial LLM response
	response, err := we.llmClient.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("initial LLM call failed: %w", err)
	}

	result.TotalTokens += response.TokensUsed
	result.TotalCost += response.Cost
	result.FinalResponse = response.Content

	// Check if the LLM is requesting tool usage using improved detection
	if we.shouldExecuteToolsImproved(response.Content) && result.ToolCallsUsed < config.MaxToolCalls {
		we.logger.Info("LLM response indicates tool usage needed", "response_sample", response.Content[:min(200, len(response.Content))])

		// Execute tools based on LLM response with security controls
		toolResults, err := we.executeToolsSecurely(ctx, response.Content, config, execCtx)
		if err != nil {
			if config.FailOnToolError {
				return nil, fmt.Errorf("tool execution failed: %w", err)
			}
			we.logger.Warn("Tool execution failed but continuing", "error", err)
			result.ToolExecutions = append(result.ToolExecutions, ToolExecution{
				Tool:    "unknown",
				Success: false,
				Error:   err.Error(),
			})
		} else {
			result.ToolExecutions = append(result.ToolExecutions, toolResults...)
		}

		result.ToolCallsUsed = len(result.ToolExecutions)

		// If we have tool results, get follow-up response from LLM
		if len(result.ToolExecutions) > 0 {
			toolResultText := we.formatToolResults(result.ToolExecutions)
			followUpPrompt := fmt.Sprintf("%s\n\nTool execution results:\n%s\n\nBased on these results, please provide your final analysis:",
				prompt, toolResultText)

			followUpResponse, err := we.llmClient.Complete(ctx, followUpPrompt)
			if err != nil {
				we.logger.Error("Follow-up LLM call failed", "error", err)
				// Don't fail the entire step, just log and use original response
			} else {
				result.FinalResponse = followUpResponse.Content
				result.TotalTokens += followUpResponse.TokensUsed
				result.TotalCost += followUpResponse.Cost
			}
		}
	}

	return result, nil
}

// shouldExecuteToolsImproved determines if the LLM response indicates tool usage with better heuristics
func (we *WorkflowEngine) shouldExecuteToolsImproved(response string) bool {
	// More sophisticated pattern matching with context awareness
	response = strings.ToLower(response)

	// Look for explicit tool usage intentions
	toolIndicators := []string{
		"let me check",
		"i'll examine",
		"let me look at",
		"i need to verify",
		"let me search",
		"i should read",
		"let me find",
		"i'll investigate",
	}

	fileOperations := []string{
		"read the file",
		"examine the code",
		"look at the implementation",
		"check the source",
		"verify the code",
	}

	// Check for tool indicators
	for _, indicator := range toolIndicators {
		if strings.Contains(response, indicator) {
			return true
		}
	}

	// Check for file operation intentions
	for _, operation := range fileOperations {
		if strings.Contains(response, operation) {
			return true
		}
	}

	return false
}

// executeToolsSecurely executes tools with proper security controls
func (we *WorkflowEngine) executeToolsSecurely(ctx context.Context, response string, config LLMWithToolsConfig, execCtx *ExecutionContext) ([]ToolExecution, error) {
	var executions []ToolExecution
	response = strings.ToLower(response)

	// Security: Only execute if we have allowed tools configured, or use safe defaults
	allowedTools := config.AllowedTools
	if len(allowedTools) == 0 {
		// Safe defaults
		allowedTools = []string{"read_file", "list_files"}
	}

	// Check for file reading requests with security controls
	if strings.Contains(response, "read") || strings.Contains(response, "file") || strings.Contains(response, "code") {
		if we.isToolAllowed("read_file", allowedTools) {
			files := we.determineFilesToRead(response, config.AllowedPaths)

			for _, file := range files {
				if !we.isPathAllowed(file, config.AllowedPaths) {
					we.logger.Warn("File access denied by security policy", "file", file)
					continue
				}

				execution := we.executeReadFileTool(ctx, file, config.MaxFileSize)
				executions = append(executions, execution)

				if len(executions) >= config.MaxToolCalls {
					break
				}
			}
		}
	}

	// Check for directory listing requests with security controls
	if strings.Contains(response, "list") || strings.Contains(response, "directory") {
		if we.isToolAllowed("list_files", allowedTools) {
			directories := we.determineDirectoriesToList(response, config.AllowedPaths)

			for _, dir := range directories {
				if !we.isPathAllowed(dir, config.AllowedPaths) {
					we.logger.Warn("Directory access denied by security policy", "directory", dir)
					continue
				}

				execution := we.executeListFilesTool(ctx, dir)
				executions = append(executions, execution)

				if len(executions) >= config.MaxToolCalls {
					break
				}
			}
		}
	}

	return executions, nil
}

// Helper functions for security and tool execution
func (we *WorkflowEngine) isToolAllowed(tool string, allowedTools []string) bool {
	if len(allowedTools) == 0 {
		return false // Default deny if no tools specified
	}
	for _, allowed := range allowedTools {
		if allowed == tool {
			return true
		}
	}
	return false
}

func (we *WorkflowEngine) isPathAllowed(path string, allowedPaths []string) bool {
	if len(allowedPaths) == 0 {
		// Default safe paths if none specified
		safePaths := []string{"pkg/", "./pkg/", "cmd/", "./cmd/", "examples/", "./examples/"}
		allowedPaths = safePaths
	}

	for _, allowed := range allowedPaths {
		if strings.HasPrefix(path, allowed) {
			return true
		}
	}
	return false
}

func (we *WorkflowEngine) determineFilesToRead(response string, allowedPaths []string) []string {
	// Safe defaults for key files to examine
	defaultFiles := []string{
		"pkg/generic/workflow_engine.go",
		"pkg/generic/template_engine.go",
		"pkg/generic/tool_registry.go",
	}

	// TODO: Implement more sophisticated file path extraction from LLM response
	// For now, return safe defaults
	var validFiles []string
	for _, file := range defaultFiles {
		if we.isPathAllowed(file, allowedPaths) {
			validFiles = append(validFiles, file)
		}
	}

	return validFiles
}

func (we *WorkflowEngine) determineDirectoriesToList(response string, allowedPaths []string) []string {
	// Safe defaults for directories to list
	defaultDirs := []string{"pkg/generic", "cmd"}

	var validDirs []string
	for _, dir := range defaultDirs {
		if we.isPathAllowed(dir, allowedPaths) {
			validDirs = append(validDirs, dir)
		}
	}

	return validDirs
}

func (we *WorkflowEngine) executeReadFileTool(ctx context.Context, filePath string, maxSize int) ToolExecution {
	tool, exists := we.toolRegistry.GetTool("read_file")
	if !exists {
		return ToolExecution{
			Tool:    "read_file",
			Success: false,
			Error:   "read_file tool not available",
		}
	}

	params := map[string]interface{}{
		"path":     filePath,
		"max_size": maxSize,
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		return ToolExecution{
			Tool:    "read_file",
			Params:  params,
			Success: false,
			Error:   err.Error(),
		}
	}

	// Format the result for display
	var resultText string
	if resultMap, ok := result.(map[string]interface{}); ok {
		if content, hasContent := resultMap["content"].(string); hasContent {
			if len(content) > 2000 {
				content = content[:2000] + "\n... (truncated)"
			}
			resultText = fmt.Sprintf("Content of %s:\n%s", filePath, content)
		}
	}

	return ToolExecution{
		Tool:    "read_file",
		Params:  params,
		Result:  resultText,
		Success: true,
	}
}

func (we *WorkflowEngine) executeListFilesTool(ctx context.Context, directory string) ToolExecution {
	tool, exists := we.toolRegistry.GetTool("list_files")
	if !exists {
		return ToolExecution{
			Tool:    "list_files",
			Success: false,
			Error:   "list_files tool not available",
		}
	}

	params := map[string]interface{}{
		"path": directory,
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		return ToolExecution{
			Tool:    "list_files",
			Params:  params,
			Success: false,
			Error:   err.Error(),
		}
	}

	resultText := fmt.Sprintf("Files in %s: %v", directory, result)

	return ToolExecution{
		Tool:    "list_files",
		Params:  params,
		Result:  resultText,
		Success: true,
	}
}

func (we *WorkflowEngine) formatToolResults(executions []ToolExecution) string {
	var results []string
	for _, execution := range executions {
		if execution.Success {
			results = append(results, fmt.Sprintf("Tool: %s\n%s", execution.Tool, execution.Result))
		} else {
			results = append(results, fmt.Sprintf("Tool: %s (FAILED)\nError: %s", execution.Tool, execution.Error))
		}
	}
	return strings.Join(results, "\n\n")
}

// executeToolsBasedOnLLMResponse executes tools based on LLM response analysis
func (we *WorkflowEngine) executeToolsBasedOnLLMResponse(ctx context.Context, response string, execCtx *ExecutionContext, previousResults map[string]*StepResult) (string, error) {
	results := []string{}

	// Simple pattern matching for common operations
	// In a full implementation, this would use proper function calling

	// Check for file reading requests
	if strings.Contains(strings.ToLower(response), "read") && strings.Contains(strings.ToLower(response), "file") {
		// Extract potential file paths or suggest reading key files
		keyFiles := []string{
			"pkg/generic/workflow_engine.go",
			"pkg/generic/template_engine.go",
			"pkg/generic/tool_registry.go",
		}

		for _, file := range keyFiles {
			if tool, exists := we.toolRegistry.GetTool("read_file"); exists {
				params := map[string]interface{}{
					"path":     file,
					"max_size": 10240, // 10KB limit
				}
				result, err := tool.Execute(ctx, params)
				if err != nil {
					results = append(results, fmt.Sprintf("Failed to read %s: %v", file, err))
				} else {
					if resultMap, ok := result.(map[string]interface{}); ok {
						if content, hasContent := resultMap["content"].(string); hasContent {
							// Truncate content for display
							if len(content) > 2000 {
								content = content[:2000] + "\n... (truncated)"
							}
							results = append(results, fmt.Sprintf("Content of %s:\n%s", file, content))
						}
					}
				}
			}
		}
	}

	// Check for directory listing requests
	if strings.Contains(strings.ToLower(response), "list") || strings.Contains(strings.ToLower(response), "directory") {
		if tool, exists := we.toolRegistry.GetTool("list_files"); exists {
			params := map[string]interface{}{
				"path": "pkg/generic",
			}
			result, err := tool.Execute(ctx, params)
			if err != nil {
				results = append(results, fmt.Sprintf("Failed to list files: %v", err))
			} else {
				results = append(results, fmt.Sprintf("Files in pkg/generic/: %v", result))
			}
		}
	}

	if len(results) == 0 {
		return "No specific tool executions were triggered based on the response.", nil
	}

	return strings.Join(results, "\n\n"), nil
}

// executeDisplayStep executes a display step that shows static text to the user
func (we *WorkflowEngine) executeDisplayStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	text, ok := step.Config["text"].(string)
	if !ok {
		// Fall back to prompt for backward compatibility
		text, ok = step.Config["prompt"].(string)
		if !ok {
			return nil, fmt.Errorf("text or prompt not specified in display step config")
		}
	}

	// Template rendering for text with context variables
	renderedText, err := we.templateEngine.RenderTemplate(text, previousResults, execCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to render display text template: %w", err)
	}

	// Display the text to the user
	fmt.Print(renderedText)
	fmt.Println()

	return renderedText, nil
}

// executeConditionStep executes a condition step
func (we *WorkflowEngine) executeConditionStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	// Get condition expression from config
	conditionExpr, ok := step.Config["condition"].(string)
	if !ok {
		return false, fmt.Errorf("condition parameter is required for condition step")
	}

	// Render the condition template with current context
	renderedCondition, err := we.templateEngine.RenderTemplate(conditionExpr, previousResults, execCtx)
	if err != nil {
		return false, fmt.Errorf("failed to render condition template: %w", err)
	}

	// Simple condition evaluation - check for basic conditions
	result := we.evaluateSimpleCondition(renderedCondition, previousResults, execCtx)
	
	we.logger.Debug("Condition evaluation", 
		"condition", conditionExpr, 
		"rendered", renderedCondition,
		"result", result)

	return result, nil
}

// evaluateSimpleCondition performs basic condition evaluation
func (we *WorkflowEngine) evaluateSimpleCondition(condition string, previousResults map[string]*StepResult, execCtx *ExecutionContext) bool {
	condition = strings.TrimSpace(condition)
	
	// Handle basic boolean values
	switch strings.ToLower(condition) {
	case "true", "yes", "1":
		return true
	case "false", "no", "0", "":
		return false
	}
	
	// Handle simple string comparisons
	if strings.Contains(condition, "==") {
		parts := strings.Split(condition, "==")
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			return left == right
		}
	}
	
	if strings.Contains(condition, "!=") {
		parts := strings.Split(condition, "!=")
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			return left != right
		}
	}
	
	// Check for "contains" operation
	if strings.Contains(condition, " contains ") {
		parts := strings.Split(condition, " contains ")
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			return strings.Contains(left, right)
		}
	}
	
	// Default: treat non-empty string as true
	return condition != ""
}

// LoopConfig represents configuration for loop steps
type LoopConfig struct {
	MaxIterations int                  `json:"max_iterations"`
	BreakOn       []LoopBreakCondition `json:"break_on"`
	Steps         []Step               `json:"steps"`
	OutputVar     string               `json:"output_var"`
}

// LoopBreakCondition defines when to exit a loop
type LoopBreakCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// LoopResult represents the result of loop execution
type LoopResult struct {
	Iterations  int                    `json:"iterations"`
	FinalResult interface{}            `json:"final_result"`
	BreakReason string                 `json:"break_reason"`
	StepResults map[string]interface{} `json:"step_results"`
}

// executeLoopStep executes a loop step with break conditions
func (we *WorkflowEngine) executeLoopStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	// Parse loop configuration
	config, err := we.parseLoopConfig(step.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid loop configuration: %w", err)
	}

	we.logger.Info("Starting loop execution",
		"step", step.Name,
		"max_iterations", config.MaxIterations,
		"break_conditions", len(config.BreakOn))

	result := &LoopResult{
		Iterations:  0,
		StepResults: make(map[string]interface{}),
		BreakReason: "max_iterations_reached",
	}

	// Execute loop iterations
	for iteration := 0; iteration < config.MaxIterations; iteration++ {
		result.Iterations = iteration + 1
		we.logger.Debug("Loop iteration starting", "iteration", result.Iterations)

		// Create iteration context with current loop variables
		iterationCtx := we.createIterationContext(execCtx, result, iteration)

		// Execute loop steps in sequence
		iterationResults := make(map[string]*StepResult)
		for _, loopStep := range config.Steps {
			// Execute step with iteration context
			stepResult, err := we.executeStep(ctx, loopStep, iterationCtx, previousResults)
			if err != nil {
				if !loopStep.ContinueOnError {
					return nil, fmt.Errorf("loop step %s failed on iteration %d: %w", loopStep.Name, result.Iterations, err)
				}
				we.logger.Warn("Loop step failed but continuing", "step", loopStep.Name, "iteration", result.Iterations, "error", err)
			}

			iterationResults[loopStep.Name] = stepResult

			// Store result for break condition evaluation
			if stepResult.Success {
				result.StepResults[loopStep.Name] = stepResult.Output
			}
		}

		// Check break conditions after each iteration
		shouldBreak, breakReason, err := we.evaluateLoopBreakConditions(config.BreakOn, iterationResults, iterationCtx)
		if err != nil {
			we.logger.Warn("Failed to evaluate break conditions", "error", err)
		} else if shouldBreak {
			result.BreakReason = breakReason
			we.logger.Info("Loop breaking early", "reason", breakReason, "iteration", result.Iterations)
			break
		}

		// Update execution context with latest results
		for name, stepResult := range iterationResults {
			previousResults[fmt.Sprintf("%s_iter_%d", name, iteration)] = stepResult
		}
	}

	// Set final result - use the specified output variable or latest step result
	if config.OutputVar != "" && result.StepResults[config.OutputVar] != nil {
		result.FinalResult = result.StepResults[config.OutputVar]
	} else {
		result.FinalResult = result.StepResults
	}

	we.logger.Info("Loop execution completed",
		"step", step.Name,
		"iterations", result.Iterations,
		"break_reason", result.BreakReason)

	return result, nil
}

// parseLoopConfig parses the loop configuration from step config
func (we *WorkflowEngine) parseLoopConfig(config map[string]interface{}) (*LoopConfig, error) {
	loopConfig := &LoopConfig{
		MaxIterations: 10, // Default max iterations
		BreakOn:       []LoopBreakCondition{},
		Steps:         []Step{},
	}

	// Parse max iterations - handle both int and float64 types
	if maxIterFloat, ok := config["max_iterations"].(float64); ok {
		loopConfig.MaxIterations = int(maxIterFloat)
	} else if maxIterInt, ok := config["max_iterations"].(int); ok {
		loopConfig.MaxIterations = maxIterInt
	}

	// Validate max iterations bounds
	if loopConfig.MaxIterations <= 0 {
		return nil, fmt.Errorf("max_iterations must be greater than 0")
	}
	if loopConfig.MaxIterations > 100 {
		return nil, fmt.Errorf("max_iterations cannot exceed 100")
	}

	// Parse output variable
	if outputVar, ok := config["output_var"].(string); ok {
		loopConfig.OutputVar = outputVar
	}

	// Parse break conditions
	if breakConditions, ok := config["break_on"].([]interface{}); ok {
		for _, conditionInterface := range breakConditions {
			if conditionMap, ok := conditionInterface.(map[string]interface{}); ok {
				condition := LoopBreakCondition{}
				if field, ok := conditionMap["field"].(string); ok {
					condition.Field = field
				}
				if operator, ok := conditionMap["operator"].(string); ok {
					condition.Operator = operator
				}
				if value, ok := conditionMap["value"].(string); ok {
					condition.Value = value
				}
				loopConfig.BreakOn = append(loopConfig.BreakOn, condition)
			}
		}
	}

	// Parse loop steps
	if stepsInterface, ok := config["steps"].([]interface{}); ok {
		for _, stepInterface := range stepsInterface {
			if stepMap, ok := stepInterface.(map[string]interface{}); ok {
				step := Step{}
				if name, ok := stepMap["name"].(string); ok {
					step.Name = name
				}
				if stepType, ok := stepMap["type"].(string); ok {
					step.Type = stepType
				}
				if stepConfig, ok := stepMap["config"].(map[string]interface{}); ok {
					step.Config = stepConfig
				}
				// Parse other step fields as needed...
				loopConfig.Steps = append(loopConfig.Steps, step)
			}
		}
	}

	if len(loopConfig.Steps) == 0 {
		return nil, fmt.Errorf("loop must have at least one step")
	}

	return loopConfig, nil
}

// createIterationContext creates an execution context for a loop iteration
func (we *WorkflowEngine) createIterationContext(baseCtx *ExecutionContext, loopResult *LoopResult, iteration int) *ExecutionContext {
	// Create a new context that inherits from base context
	iterationCtx := &ExecutionContext{
		Data:        make(map[string]interface{}),
		StepResults: baseCtx.StepResults,
		Metrics:     baseCtx.Metrics,
	}

	// Copy base context data
	for k, v := range baseCtx.Data {
		iterationCtx.Data[k] = v
	}

	// Add loop-specific variables
	iterationCtx.Data["loop_iteration"] = iteration
	iterationCtx.Data["loop_iterations_completed"] = loopResult.Iterations

	// Add previous loop results
	for k, v := range loopResult.StepResults {
		iterationCtx.Data[fmt.Sprintf("prev_%s", k)] = v
	}

	return iterationCtx
}

// evaluateLoopBreakConditions evaluates whether loop should break
func (we *WorkflowEngine) evaluateLoopBreakConditions(conditions []LoopBreakCondition, stepResults map[string]*StepResult, execCtx *ExecutionContext) (bool, string, error) {
	if len(conditions) == 0 {
		return false, "", nil
	}

	for _, condition := range conditions {
		// Get the field value from step results or context
		var fieldValue interface{}

		if result, exists := stepResults[condition.Field]; exists && result.Success {
			fieldValue = result.Output

			// Handle tool outputs that return maps (like ask_user)
			if outputMap, ok := fieldValue.(map[string]interface{}); ok {
				if response, hasResponse := outputMap["response"]; hasResponse {
					fieldValue = response
				}
			}
		} else if value, exists := execCtx.Data[condition.Field]; exists {
			fieldValue = value
		} else {
			continue // Field not found, check next condition
		}

		// Convert to string for comparison
		fieldStr := fmt.Sprintf("%v", fieldValue)

		// Evaluate condition
		matched := false
		switch condition.Operator {
		case "equals":
			matched = fieldStr == condition.Value
		case "not_equals":
			matched = fieldStr != condition.Value
		case "contains":
			matched = strings.Contains(fieldStr, condition.Value)
		case "not_contains":
			matched = !strings.Contains(fieldStr, condition.Value)
		default:
			we.logger.Warn("Unknown loop break condition operator", "operator", condition.Operator)
			continue
		}

		if matched {
			reason := fmt.Sprintf("condition met: %s %s %s", condition.Field, condition.Operator, condition.Value)
			return true, reason, nil
		}
	}

	return false, "", nil
}

// executeParallelStep executes parallel steps
func (we *WorkflowEngine) executeParallelStep(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	// Get parallel steps from config
	parallelStepsConfig, ok := step.Config["steps"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("steps parameter is required for parallel step")
	}

	// Convert to Step structs
	var parallelSteps []Step
	for i, stepInterface := range parallelStepsConfig {
		stepMap, ok := stepInterface.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid step configuration at index %d", i)
		}

		parallelStep := Step{}
		if name, ok := stepMap["name"].(string); ok {
			parallelStep.Name = name
		} else {
			parallelStep.Name = fmt.Sprintf("parallel_%d", i)
		}
		if stepType, ok := stepMap["type"].(string); ok {
			parallelStep.Type = stepType
		}
		if config, ok := stepMap["config"].(map[string]interface{}); ok {
			parallelStep.Config = config
		}

		parallelSteps = append(parallelSteps, parallelStep)
	}

	// Execute steps in parallel using goroutines
	type parallelResult struct {
		index  int
		name   string
		result interface{}
		err    error
	}

	resultChan := make(chan parallelResult, len(parallelSteps))
	
	// Start all parallel steps
	for i, parallelStep := range parallelSteps {
		go func(index int, step Step) {
			defer func() {
				if r := recover(); r != nil {
					resultChan <- parallelResult{
						index: index,
						name:  step.Name,
						err:   fmt.Errorf("panic in parallel step: %v", r),
					}
				}
			}()

			result, err := we.executeStepByType(ctx, step, execCtx, previousResults)
			resultChan <- parallelResult{
				index:  index,
				name:   step.Name,
				result: result,
				err:    err,
			}
		}(i, parallelStep)
	}

	// Collect results
	results := make(map[string]interface{})
	var errors []string
	
	for i := 0; i < len(parallelSteps); i++ {
		select {
		case result := <-resultChan:
			if result.err != nil {
				errors = append(errors, fmt.Sprintf("Step '%s': %v", result.name, result.err))
			} else {
				results[result.name] = result.result
			}
		case <-ctx.Done():
			return nil, fmt.Errorf("parallel execution cancelled: %w", ctx.Err())
		}
	}

	response := map[string]interface{}{
		"results":   results,
		"completed": len(results),
		"total":     len(parallelSteps),
	}

	if len(errors) > 0 {
		response["errors"] = errors
		return response, fmt.Errorf("some parallel steps failed: %v", errors)
	}

	return response, nil
}

// executeStepByType executes a step based on its type (helper for parallel execution)
func (we *WorkflowEngine) executeStepByType(ctx context.Context, step Step, execCtx *ExecutionContext, previousResults map[string]*StepResult) (interface{}, error) {
	switch step.Type {
	case "tool":
		return we.executeToolStep(ctx, step, execCtx, previousResults)
	case "llm":
		return we.executeLLMStep(ctx, step, execCtx, previousResults)
	case "llm_display":
		return we.executeLLMDisplayStep(ctx, step, execCtx, previousResults)
	case "display":
		return we.executeDisplayStep(ctx, step, execCtx, previousResults)
	case "condition":
		return we.executeConditionStep(ctx, step, execCtx, previousResults)
	default:
		return nil, fmt.Errorf("unsupported step type for parallel execution: %s", step.Type)
	}
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
