package generic

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Agent represents a generic AI agent
type Agent struct {
	config       *AgentConfig
	logger       *slog.Logger
	dataIngestor *DataIngestor
	toolRegistry *ToolRegistry
	llmClient    *LLMClient
	workflow     *WorkflowEngine
	outputWriter *OutputWriter
	validator    *Validator
}

// ExecutionContext holds context for agent execution
type ExecutionContext struct {
	Context     context.Context
	SessionID   string
	StartTime   time.Time
	Data        map[string]interface{}
	Variables   map[string]string
	StepResults map[string]*StepResult
	Metrics     *ExecutionMetrics
}

// StepResult holds the result of a workflow step
type StepResult struct {
	StepName      string
	Success       bool
	Output        interface{}
	Error         error
	ExecutionTime time.Duration
	Metadata      map[string]interface{}
}

// ExecutionMetrics tracks execution statistics
type ExecutionMetrics struct {
	TotalSteps         int
	SuccessfulSteps    int
	FailedSteps        int
	TotalExecutionTime time.Duration
	LLMTokensUsed      int
	LLMCost            float64
	DataProcessed      int64
}

// NewAgent creates a new generic agent
func NewAgent(config *AgentConfig, logger *slog.Logger) (*Agent, error) {
	agent := &Agent{
		config: config,
		logger: logger,
	}

	// Initialize components
	var err error

	// Data ingestion
	agent.dataIngestor, err = NewDataIngestor(config.DataSources, &config.Embeddings, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create data ingestor: %w", err)
	}

	// Tool registry
	agent.toolRegistry, err = NewToolRegistry(config.Tools, &config.Security, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool registry: %w", err)
	}

	// LLM client
	agent.llmClient, err = NewLLMClient(config.LLM, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Validator (create before workflow engine as it's needed)
	agent.validator, err = NewValidator(config.Validation, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	// Workflow engine
	agent.workflow, err = NewWorkflowEngine(config.Workflows, agent.toolRegistry, agent.llmClient, agent.validator, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow engine: %w", err)
	}

	// Output writer
	agent.outputWriter, err = NewOutputWriter(config.Outputs, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create output writer: %w", err)
	}

	return agent, nil
}

// Execute runs the agent with the given input
func (a *Agent) Execute(input string) error {
	ctx := context.Background()
	return a.ExecuteWithContext(ctx, input)
}

// ExecuteWithContext runs the agent with context
func (a *Agent) ExecuteWithContext(ctx context.Context, input string) error {
	startTime := time.Now()
	sessionID := generateSessionID()

	execCtx := &ExecutionContext{
		Context:     ctx,
		SessionID:   sessionID,
		StartTime:   startTime,
		Data:        make(map[string]interface{}),
		Variables:   make(map[string]string),
		StepResults: make(map[string]*StepResult),
		Metrics:     &ExecutionMetrics{},
	}

	// Add environment variables to context
	for k, v := range a.config.Environment.Variables {
		execCtx.Variables[k] = v
	}

	a.logger.Info("Starting agent execution",
		"agent", a.config.Agent.Name,
		"session_id", sessionID,
		"input", input)

	defer func() {
		execCtx.Metrics.TotalExecutionTime = time.Since(startTime)
		a.logger.Info("Agent execution completed",
			"session_id", sessionID,
			"duration", execCtx.Metrics.TotalExecutionTime,
			"steps_total", execCtx.Metrics.TotalSteps,
			"steps_successful", execCtx.Metrics.SuccessfulSteps,
			"steps_failed", execCtx.Metrics.FailedSteps,
			"tokens_used", execCtx.Metrics.LLMTokensUsed,
			"cost", execCtx.Metrics.LLMCost)
	}()

	// Step 1: Data ingestion
	if len(a.config.DataSources) > 0 {
		a.logger.Info("Starting data ingestion", "sources", len(a.config.DataSources))
		data, err := a.dataIngestor.IngestAll(ctx)
		if err != nil {
			return fmt.Errorf("data ingestion failed: %w", err)
		}
		execCtx.Data["ingested_data"] = data
		execCtx.Metrics.DataProcessed = int64(len(data))
	}

	// Step 2: Add input to context
	execCtx.Data["input"] = input

	// Step 3: Execute workflows
	if len(a.config.Workflows) == 0 {
		return fmt.Errorf("no workflows defined")
	}

	// Find the appropriate workflow
	workflow := a.selectWorkflow(input, execCtx)
	if workflow == nil {
		return fmt.Errorf("no suitable workflow found for input")
	}

	a.logger.Info("Executing workflow", "workflow", workflow.Name)

	result, err := a.workflow.Execute(ctx, workflow, execCtx)
	if err != nil {
		return fmt.Errorf("workflow execution failed: %w", err)
	}

	// Step 4: Validate output if validation is enabled
	if a.config.Validation.Enabled {
		a.logger.Info("Validating output")
		if err := a.validator.Validate(result); err != nil {
			switch a.config.Validation.OnFailure {
			case "stop":
				return fmt.Errorf("validation failed: %w", err)
			case "warn":
				a.logger.Warn("Validation failed", "error", err)
			case "retry":
				// TODO: Implement retry logic
				a.logger.Warn("Validation failed, retry not implemented", "error", err)
			}
		}
	}

	// Step 5: Write output
	if len(a.config.Outputs) > 0 {
		a.logger.Info("Writing output", "processors", len(a.config.Outputs))
		if err := a.outputWriter.WriteAll(result, execCtx); err != nil {
			return fmt.Errorf("output writing failed: %w", err)
		}
	}

	return nil
}

// selectWorkflow selects the appropriate workflow based on input and context
func (a *Agent) selectWorkflow(input string, execCtx *ExecutionContext) *Workflow {
	var selectedWorkflow *Workflow
	highestPriority := -1

	for _, workflow := range a.config.Workflows {
		// Simple workflow selection - in a real implementation, this would be more sophisticated
		if workflow.Trigger.Priority > highestPriority {
			selectedWorkflow = &workflow
			highestPriority = workflow.Trigger.Priority
		}
	}

	// If no workflow with priority is found, use the first one
	if selectedWorkflow == nil && len(a.config.Workflows) > 0 {
		selectedWorkflow = &a.config.Workflows[0]
	}

	return selectedWorkflow
}

// Stop gracefully stops the agent
func (a *Agent) Stop() error {
	a.logger.Info("Stopping agent", "agent", a.config.Agent.Name)

	// TODO: Implement graceful shutdown
	// - Cancel running workflows
	// - Cleanup resources
	// - Save state if needed

	return nil
}

// GetConfig returns the agent configuration
func (a *Agent) GetConfig() *AgentConfig {
	return a.config
}

// GetMetrics returns execution metrics
func (a *Agent) GetMetrics() *ExecutionMetrics {
	// This would typically be maintained across executions
	return &ExecutionMetrics{}
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}
