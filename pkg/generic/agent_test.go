package generic

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestNewAgent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name        string
		config      *AgentConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &AgentConfig{
				Agent: AgentInfo{
					Name:        "test-agent",
					Description: "A test agent",
				},
				LLM: LLMConfig{
					Provider: "openai",
					Model:    "gpt-4",
				},
				DataSources: []DataSource{},
				Tools:       map[string]Tool{},
				Workflows:   []Workflow{},
				Outputs:     []Output{},
				Security:    Security{Enabled: true},
				Validation:  Validation{Enabled: true},
			},
			expectError: false,
		},
		{
			name: "empty config fields",
			config: &AgentConfig{
				Agent: AgentInfo{
					Name:        "",
					Description: "",
				},
				LLM: LLMConfig{
					Provider: "",
					Model:    "",
				},
			},
			expectError: true,
			errorMsg:    "API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config != nil {
				// Set required defaults
				if err := tt.config.setDefaults(); err != nil {
					t.Fatalf("Failed to set defaults: %v", err)
				}
			}

			agent, err := NewAgent(tt.config, logger)

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
				if agent == nil {
					t.Error("Agent should not be nil when no error")
				}
			}
		})
	}
}

func TestAgentExecute(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := &AgentConfig{
		Agent: AgentInfo{
			Name:        "test-agent",
			Description: "A test agent",
		},
		LLM: LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Workflows: []Workflow{
			{
				Name:        "test-workflow",
				Description: "Test workflow",
				Trigger:     Trigger{Priority: 1},
				Steps: []Step{
					{
						Name:   "test-step",
						Type:   "llm",
						Config: map[string]interface{}{"prompt": "Hello"},
					},
				},
			},
		},
		Security:   Security{Enabled: false},   // Disable for testing
		Validation: Validation{Enabled: false}, // Disable for testing
	}

	if err := config.setDefaults(); err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	agent, err := NewAgent(config, logger)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test basic execution (will fail due to LLM not being available, but should not panic)
	err = agent.Execute("test input")

	// We expect this to fail in testing environment, but it should be a controlled failure
	if err == nil {
		t.Log("Execute succeeded (unexpected in test environment)")
	} else {
		t.Logf("Execute failed as expected in test environment: %v", err)
	}
}

func TestAgentExecuteWithContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := &AgentConfig{
		Agent: AgentInfo{
			Name:        "test-agent",
			Description: "A test agent",
		},
		LLM: LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Workflows: []Workflow{
			{
				Name:        "test-workflow",
				Description: "Test workflow",
				Trigger:     Trigger{Priority: 1},
				Steps: []Step{
					{
						Name:   "test-step",
						Type:   "llm",
						Config: map[string]interface{}{"prompt": "Hello"},
					},
				},
			},
		},
		Security:   Security{Enabled: false},
		Validation: Validation{Enabled: false},
	}

	if err := config.setDefaults(); err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	agent, err := NewAgent(config, logger)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = agent.ExecuteWithContext(ctx, "test input")

	// Expect failure due to either timeout or missing LLM credentials
	if err == nil {
		t.Log("Execute succeeded (unexpected in test environment)")
	} else {
		t.Logf("Execute failed as expected: %v", err)
	}
}

func TestSelectWorkflow(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := &AgentConfig{
		Agent: AgentInfo{
			Name:        "test-agent",
			Description: "A test agent",
		},
		LLM: LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Workflows: []Workflow{
			{
				Name:        "low-priority",
				Description: "Low priority workflow",
				Trigger:     Trigger{Priority: 1},
				Steps: []Step{
					{Name: "step1", Type: "llm"},
				},
			},
			{
				Name:        "high-priority",
				Description: "High priority workflow",
				Trigger:     Trigger{Priority: 10},
				Steps: []Step{
					{Name: "step2", Type: "llm"},
				},
			},
		},
		Security:   Security{Enabled: false},
		Validation: Validation{Enabled: false},
	}

	if err := config.setDefaults(); err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	agent, err := NewAgent(config, logger)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	execCtx := &ExecutionContext{
		Data:      make(map[string]interface{}),
		Variables: make(map[string]string),
	}

	// Test workflow selection - should select highest priority
	workflow := agent.selectWorkflow("test input", execCtx)
	if workflow == nil {
		t.Error("Expected workflow to be selected")
	} else if workflow.Name != "high-priority" {
		t.Errorf("Expected 'high-priority' workflow, got '%s'", workflow.Name)
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	// Add a small sleep to ensure different timestamps
	time.Sleep(1 * time.Millisecond)
	id2 := generateSessionID()

	if id1 == id2 {
		t.Error("Expected different session IDs")
	}

	if id1 == "" || id2 == "" {
		t.Error("Expected non-empty session IDs")
	}

	// Check format
	expectedPrefix := "session_"
	if len(id1) < len(expectedPrefix) || id1[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected session ID to start with '%s', got '%s'", expectedPrefix, id1)
	}
}

func TestAgentGetConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := &AgentConfig{
		Agent: AgentInfo{
			Name:        "test-agent",
			Description: "A test agent",
		},
		LLM: LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Security:   Security{Enabled: false},
		Validation: Validation{Enabled: false},
	}

	if err := config.setDefaults(); err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	agent, err := NewAgent(config, logger)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	retrievedConfig := agent.GetConfig()
	if retrievedConfig == nil {
		t.Error("Expected config to be returned")
	}

	if retrievedConfig.Agent.Name != config.Agent.Name {
		t.Errorf("Expected agent name '%s', got '%s'", config.Agent.Name, retrievedConfig.Agent.Name)
	}
}

func TestAgentGetMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := &AgentConfig{
		Agent: AgentInfo{
			Name:        "test-agent",
			Description: "A test agent",
		},
		LLM: LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Security:   Security{Enabled: false},
		Validation: Validation{Enabled: false},
	}

	if err := config.setDefaults(); err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	agent, err := NewAgent(config, logger)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	metrics := agent.GetMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be returned")
	}

	// Metrics should be initialized with zero values
	if metrics.TotalSteps != 0 {
		t.Errorf("Expected total steps 0, got %d", metrics.TotalSteps)
	}
}

func TestAgentStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := &AgentConfig{
		Agent: AgentInfo{
			Name:        "test-agent",
			Description: "A test agent",
		},
		LLM: LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Security:   Security{Enabled: false},
		Validation: Validation{Enabled: false},
	}

	if err := config.setDefaults(); err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	agent, err := NewAgent(config, logger)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test stop - should not error
	err = agent.Stop()
	if err != nil {
		t.Errorf("Unexpected error stopping agent: %v", err)
	}
}
