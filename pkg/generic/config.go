package generic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AgentConfig represents the complete agent configuration
type AgentConfig struct {
	Agent       AgentInfo       `json:"agent" validate:"required"`
	LLM         LLMConfig       `json:"llm" validate:"required"`
	DataSources []DataSource    `json:"data_sources,omitempty"`
	Workflows   []Workflow      `json:"workflows,omitempty"`
	Tools       map[string]Tool `json:"tools,omitempty"`
	Outputs     []Output        `json:"outputs,omitempty"`
	Environment Environment     `json:"environment,omitempty"`
	Security    Security        `json:"security,omitempty"`
	Validation  Validation      `json:"validation,omitempty"`
}

// AgentInfo contains basic agent metadata
type AgentInfo struct {
	Name          string   `json:"name" validate:"required"`
	Description   string   `json:"description" validate:"required"`
	Version       string   `json:"version"`
	Goals         []string `json:"goals,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
	MaxIterations int      `json:"max_iterations"`
	Timeout       string   `json:"timeout"`
	Interactive   bool     `json:"interactive"`
}

// LLMConfig contains LLM provider configuration
type LLMConfig struct {
	Provider          string                 `json:"provider" validate:"required"`
	Model             string                 `json:"model" validate:"required"`
	Temperature       float64                `json:"temperature"`
	MaxTokens         int                    `json:"max_tokens"`
	SystemPrompt      string                 `json:"system_prompt,omitempty"`
	SpecializedModels map[string]string      `json:"specialized_models,omitempty"`
	ProviderConfig    map[string]interface{} `json:"provider_config,omitempty"`
	APIKey            string                 `json:"api_key,omitempty"` // Can be set directly or via environment variable
}

// DataSource defines a data ingestion source
type DataSource struct {
	Name          string                 `json:"name" validate:"required"`
	Type          string                 `json:"type" validate:"required"`
	Config        map[string]interface{} `json:"config,omitempty"`
	Preprocessing []ProcessingStep       `json:"preprocessing,omitempty"`
}

// ProcessingStep defines a data processing step
type ProcessingStep struct {
	Type   string                 `json:"type" validate:"required"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// Workflow defines an execution workflow
type Workflow struct {
	Name        string     `json:"name" validate:"required"`
	Description string     `json:"description,omitempty"`
	Trigger     Trigger    `json:"trigger,omitempty"`
	Steps       []Step     `json:"steps" validate:"required"`
	Output      OutputSpec `json:"output,omitempty"`
}

// Trigger defines when a workflow should execute
type Trigger struct {
	Conditions []string `json:"conditions,omitempty"`
	Priority   int      `json:"priority"`
}

// Step defines a workflow step
type Step struct {
	Name              string                 `json:"name" validate:"required"`
	Type              string                 `json:"type" validate:"required"`
	Config            map[string]interface{} `json:"config,omitempty"`
	DependsOn         []string               `json:"depends_on,omitempty"`
	Retry             RetryConfig            `json:"retry,omitempty"`
	ContinueOnError   bool                   `json:"continue_on_error"`
	ContextTransforms []Transform            `json:"context_transforms,omitempty"`
	PostTransforms    []Transform            `json:"post_transforms,omitempty"`
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts int    `json:"max_attempts"`
	Backoff     string `json:"backoff"`
}

// Transform defines a data transformation operation
type Transform struct {
	Name      string                 `json:"name,omitempty"`
	Source    string                 `json:"source" validate:"required"`
	Transform string                 `json:"transform" validate:"required"`
	Params    map[string]interface{} `json:"params,omitempty"`
	StoreAs   string                 `json:"store_as,omitempty"`
	Condition string                 `json:"condition,omitempty"`
}

// OutputSpec defines workflow output configuration
type OutputSpec struct {
	Format      string `json:"format,omitempty"`
	Destination string `json:"destination,omitempty"`
	Template    string `json:"template,omitempty"`
}

// Tool defines a tool configuration
type Tool struct {
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Permissions []string               `json:"permissions,omitempty"`
	Timeout     string                 `json:"timeout,omitempty"`
}

// Output defines an output processor configuration
type Output struct {
	Name   string                 `json:"name" validate:"required"`
	Type   string                 `json:"type" validate:"required"`
	Config map[string]interface{} `json:"config,omitempty"`
	Filter OutputFilter           `json:"filter,omitempty"`
}

// OutputFilter defines output filtering
type OutputFilter struct {
	Include   []string `json:"include,omitempty"`
	Exclude   []string `json:"exclude,omitempty"`
	Transform string   `json:"transform,omitempty"`
}

// Environment defines runtime environment configuration
type Environment struct {
	Variables     map[string]string `json:"variables,omitempty"`
	WorkspaceRoot string            `json:"workspace_root"`
	TempDir       string            `json:"temp_dir,omitempty"`
	LogLevel      string            `json:"log_level"`
	Cache         CacheConfig       `json:"cache,omitempty"`
	Limits        ResourceLimits    `json:"limits,omitempty"`
}

// CacheConfig defines caching behavior
type CacheConfig struct {
	Enabled   bool   `json:"enabled"`
	TTL       string `json:"ttl"`
	Directory string `json:"directory"`
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	MaxMemory        string  `json:"max_memory,omitempty"`
	MaxCPU           float64 `json:"max_cpu,omitempty"`
	MaxFiles         int     `json:"max_files,omitempty"`
	MaxExecutionTime string  `json:"max_execution_time,omitempty"`
}

// Security defines security configuration
type Security struct {
	Enabled         bool     `json:"enabled"`
	AllowedPaths    []string `json:"allowed_paths,omitempty"`
	BlockedPaths    []string `json:"blocked_paths,omitempty"`
	AllowedCommands []string `json:"allowed_commands,omitempty"`
	RequireApproval bool     `json:"require_approval"`
	MaxFileSize     string   `json:"max_file_size"`
}

// Validation defines output validation configuration
type Validation struct {
	Enabled   bool             `json:"enabled"`
	Rules     []ValidationRule `json:"rules,omitempty"`
	OnFailure string           `json:"on_failure"`
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	Name   string                 `json:"name" validate:"required"`
	Type   string                 `json:"type" validate:"required"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// LoadConfig loads agent configuration from file
func LoadConfig(filePath string) (*AgentConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	if err := config.setDefaults(); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// SaveConfig saves agent configuration to file
func SaveConfig(config *AgentConfig, filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// setDefaults sets default values for configuration
func (c *AgentConfig) setDefaults() error {
	// Agent defaults
	if c.Agent.Version == "" {
		c.Agent.Version = "1.0.0"
	}
	if c.Agent.MaxIterations == 0 {
		c.Agent.MaxIterations = 10
	}
	if c.Agent.Timeout == "" {
		c.Agent.Timeout = "5m"
	}

	// LLM defaults
	if c.LLM.Temperature == 0 {
		c.LLM.Temperature = 0.7
	}
	if c.LLM.MaxTokens == 0 {
		c.LLM.MaxTokens = 4096
	}

	// Environment defaults
	if c.Environment.WorkspaceRoot == "" {
		c.Environment.WorkspaceRoot = "."
	}
	if c.Environment.LogLevel == "" {
		c.Environment.LogLevel = "info"
	}

	// Cache defaults
	if c.Environment.Cache.TTL == "" {
		c.Environment.Cache.TTL = "1h"
	}
	if c.Environment.Cache.Directory == "" {
		c.Environment.Cache.Directory = ".agent/cache"
	}

	// Security defaults
	c.Security.Enabled = true
	if c.Security.MaxFileSize == "" {
		c.Security.MaxFileSize = "10MB"
	}

	// Validation defaults
	c.Validation.Enabled = true
	if c.Validation.OnFailure == "" {
		c.Validation.OnFailure = "warn"
	}

	return nil
}

// validate validates the configuration
func (c *AgentConfig) validate() error {
	if c.Agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if c.Agent.Description == "" {
		return fmt.Errorf("agent description is required")
	}
	if c.LLM.Provider == "" {
		return fmt.Errorf("LLM provider is required")
	}
	if c.LLM.Model == "" {
		return fmt.Errorf("LLM model is required")
	}

	// Validate timeout format
	if _, err := time.ParseDuration(c.Agent.Timeout); err != nil {
		return fmt.Errorf("invalid timeout format: %w", err)
	}

	// Validate workflows
	for i, workflow := range c.Workflows {
		if workflow.Name == "" {
			return fmt.Errorf("workflow %d: name is required", i)
		}
		if len(workflow.Steps) == 0 {
			return fmt.Errorf("workflow %s: at least one step is required", workflow.Name)
		}
	}

	return nil
}

// GetTimeout returns the agent timeout as a duration
func (c *AgentConfig) GetTimeout() time.Duration {
	duration, _ := time.ParseDuration(c.Agent.Timeout)
	return duration
}

// GetWorkflow returns a workflow by name
func (c *AgentConfig) GetWorkflow(name string) *Workflow {
	for _, workflow := range c.Workflows {
		if workflow.Name == name {
			return &workflow
		}
	}
	return nil
}

// GetTool returns a tool configuration by name
func (c *AgentConfig) GetTool(name string) (*Tool, bool) {
	tool, exists := c.Tools[name]
	return &tool, exists
}

// IsToolEnabled checks if a tool is enabled
func (c *AgentConfig) IsToolEnabled(name string) bool {
	tool, exists := c.Tools[name]
	return exists && tool.Enabled
}
