package generic

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			configJSON: `{
				"agent": {
					"name": "test-agent",
					"description": "A test agent",
					"version": "1.0.0"
				},
				"llm": {
					"provider": "openai",
					"model": "gpt-4",
					"temperature": 0.7,
					"max_tokens": 4096
				},
				"workflows": [{
					"name": "test-workflow",
					"description": "Test workflow",
					"steps": [{
						"name": "test-step",
						"type": "llm",
						"config": {
							"prompt": "Hello"
						}
					}]
				}]
			}`,
			expectError: false,
		},
		{
			name: "missing agent name",
			configJSON: `{
				"agent": {
					"description": "A test agent"
				},
				"llm": {
					"provider": "openai",
					"model": "gpt-4"
				}
			}`,
			expectError: true,
			errorMsg:    "agent name is required",
		},
		{
			name: "missing llm provider",
			configJSON: `{
				"agent": {
					"name": "test-agent",
					"description": "A test agent"
				},
				"llm": {
					"model": "gpt-4"
				}
			}`,
			expectError: true,
			errorMsg:    "LLM provider is required",
		},
		{
			name: "invalid timeout format",
			configJSON: `{
				"agent": {
					"name": "test-agent",
					"description": "A test agent",
					"timeout": "invalid"
				},
				"llm": {
					"provider": "openai",
					"model": "gpt-4"
				}
			}`,
			expectError: true,
			errorMsg:    "invalid timeout format",
		},
		{
			name: "workflow without steps",
			configJSON: `{
				"agent": {
					"name": "test-agent",
					"description": "A test agent"
				},
				"llm": {
					"provider": "openai",
					"model": "gpt-4"
				},
				"workflows": [{
					"name": "empty-workflow",
					"steps": []
				}]
			}`,
			expectError: true,
			errorMsg:    "at least one step is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.json")

			err := os.WriteFile(configPath, []byte(tt.configJSON), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config file: %v", err)
			}

			// Load config
			config, err := LoadConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !containsError(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if config == nil {
					t.Error("Config should not be nil when no error")
				}
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	config := &AgentConfig{
		Agent: AgentInfo{
			Name:        "test-agent",
			Description: "A test agent",
			Version:     "1.0.0",
		},
		LLM: LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
	}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "saved_config.json")

	// Save config
	err := SaveConfig(config, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load and verify content
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Agent.Name != config.Agent.Name {
		t.Errorf("Expected agent name %s, got %s", config.Agent.Name, loadedConfig.Agent.Name)
	}
}

func TestConfigDefaults(t *testing.T) {
	config := &AgentConfig{
		Agent: AgentInfo{
			Name:        "test-agent",
			Description: "A test agent",
		},
		LLM: LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
	}

	err := config.setDefaults()
	if err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	// Test defaults
	if config.Agent.Version != "1.0.0" {
		t.Errorf("Expected default version '1.0.0', got '%s'", config.Agent.Version)
	}

	if config.Agent.MaxIterations != 10 {
		t.Errorf("Expected default max_iterations 10, got %d", config.Agent.MaxIterations)
	}

	if config.Agent.Timeout != "5m" {
		t.Errorf("Expected default timeout '5m', got '%s'", config.Agent.Timeout)
	}

	if config.LLM.Temperature != 0.7 {
		t.Errorf("Expected default temperature 0.7, got %f", config.LLM.Temperature)
	}

	if config.LLM.MaxTokens != 4096 {
		t.Errorf("Expected default max_tokens 4096, got %d", config.LLM.MaxTokens)
	}

	if config.Environment.WorkspaceRoot != "." {
		t.Errorf("Expected default workspace_root '.', got '%s'", config.Environment.WorkspaceRoot)
	}

	if config.Environment.LogLevel != "info" {
		t.Errorf("Expected default log_level 'info', got '%s'", config.Environment.LogLevel)
	}

	if !config.Security.Enabled {
		t.Error("Expected security to be enabled by default")
	}

	if !config.Validation.Enabled {
		t.Error("Expected validation to be enabled by default")
	}
}

func TestGetTimeout(t *testing.T) {
	tests := []struct {
		timeout  string
		expected time.Duration
	}{
		{"5m", 5 * time.Minute},
		{"30s", 30 * time.Second},
		{"1h", time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.timeout, func(t *testing.T) {
			config := &AgentConfig{
				Agent: AgentInfo{Timeout: tt.timeout},
			}

			duration := config.GetTimeout()
			if duration != tt.expected {
				t.Errorf("Expected timeout %v, got %v", tt.expected, duration)
			}
		})
	}
}

func TestGetWorkflow(t *testing.T) {
	config := &AgentConfig{
		Workflows: []Workflow{
			{Name: "workflow1", Description: "First workflow"},
			{Name: "workflow2", Description: "Second workflow"},
		},
	}

	// Test existing workflow
	workflow := config.GetWorkflow("workflow1")
	if workflow == nil {
		t.Error("Expected to find workflow1")
	} else if workflow.Name != "workflow1" {
		t.Errorf("Expected workflow name 'workflow1', got '%s'", workflow.Name)
	}

	// Test non-existing workflow
	workflow = config.GetWorkflow("nonexistent")
	if workflow != nil {
		t.Error("Expected nil for non-existing workflow")
	}
}

func TestGetTool(t *testing.T) {
	config := &AgentConfig{
		Tools: map[string]Tool{
			"file_writer": {
				Enabled: true,
				Config:  map[string]interface{}{"path": "/tmp"},
			},
			"web_scraper": {
				Enabled: false,
			},
		},
	}

	// Test existing tool
	tool, exists := config.GetTool("file_writer")
	if !exists {
		t.Error("Expected tool to exist")
	}
	if !tool.Enabled {
		t.Error("Expected tool to be enabled")
	}

	// Test non-existing tool
	_, exists = config.GetTool("nonexistent")
	if exists {
		t.Error("Expected tool to not exist")
	}
}

func TestIsToolEnabled(t *testing.T) {
	config := &AgentConfig{
		Tools: map[string]Tool{
			"enabled_tool":  {Enabled: true},
			"disabled_tool": {Enabled: false},
		},
	}

	if !config.IsToolEnabled("enabled_tool") {
		t.Error("Expected enabled_tool to be enabled")
	}

	if config.IsToolEnabled("disabled_tool") {
		t.Error("Expected disabled_tool to be disabled")
	}

	if config.IsToolEnabled("nonexistent") {
		t.Error("Expected nonexistent tool to be disabled")
	}
}

func TestValidateConfig(t *testing.T) {
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
					Name:        "test",
					Description: "test agent",
					Timeout:     "5m",
				},
				LLM: LLMConfig{
					Provider: "openai",
					Model:    "gpt-4",
				},
				Workflows: []Workflow{
					{
						Name:  "test-workflow",
						Steps: []Step{{Name: "step1", Type: "llm"}},
					},
				},
			},
			expectError: false,
		},
		{
			name: "unnamed workflow",
			config: &AgentConfig{
				Agent: AgentInfo{
					Name:        "test",
					Description: "test agent",
					Timeout:     "5m",
				},
				LLM: LLMConfig{
					Provider: "openai",
					Model:    "gpt-4",
				},
				Workflows: []Workflow{
					{
						Name:  "",
						Steps: []Step{{Name: "step1", Type: "llm"}},
					},
				},
			},
			expectError: true,
			errorMsg:    "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !containsError(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function to check if error message contains expected text
func containsError(errorMsg, expectedSubstring string) bool {
	return len(expectedSubstring) > 0 && len(errorMsg) > 0 &&
		(errorMsg == expectedSubstring ||
			len(errorMsg) >= len(expectedSubstring) &&
				errorMsg[:len(expectedSubstring)] == expectedSubstring ||
			containsSubstring(errorMsg, expectedSubstring))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
