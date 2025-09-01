package llm

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/alantheprice/agent/pkg/interfaces"
	"github.com/alantheprice/agent/pkg/interfaces/types"
)

// MockProviderFactory is a mock implementation of ProviderFactory for testing
type MockProviderFactory struct {
	name string
}

func (m *MockProviderFactory) GetName() string {
	return m.name
}

func (m *MockProviderFactory) Create(config *types.ProviderConfig) (interfaces.LLMProvider, error) {
	return &MockProvider{name: m.name}, nil
}

func (m *MockProviderFactory) Validate(config *types.ProviderConfig) error {
	return nil
}

// MockProvider is a mock implementation of LLMProvider for testing
type MockProvider struct {
	name string
}

func (m *MockProvider) GetName() string {
	return m.name
}

func (m *MockProvider) GenerateResponse(ctx context.Context, messages []types.Message, options types.RequestOptions) (string, *types.ResponseMetadata, error) {
	return "mock response", &types.ResponseMetadata{
		Model:    options.Model,
		Provider: m.name,
		TokenUsage: types.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 10,
			TotalTokens:      20,
		},
	}, nil
}

func (m *MockProvider) GenerateResponseStream(ctx context.Context, messages []types.Message, options types.RequestOptions, writer io.Writer) (*types.ResponseMetadata, error) {
	writer.Write([]byte("mock stream response"))
	return &types.ResponseMetadata{
		Model:    options.Model,
		Provider: m.name,
		TokenUsage: types.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 10,
			TotalTokens:      20,
		},
	}, nil
}

func (m *MockProvider) IsAvailable(ctx context.Context) error {
	return nil
}

func (m *MockProvider) EstimateTokens(messages []types.Message) (int, error) {
	return len(messages) * 10, nil
}

func (m *MockProvider) CalculateCost(usage types.TokenUsage) float64 {
	return float64(usage.TotalTokens) * 0.001
}

func (m *MockProvider) GetModels(ctx context.Context) ([]types.ModelInfo, error) {
	return []types.ModelInfo{
		{
			Name:           "mock-model",
			Provider:       m.name,
			MaxTokens:      1000,
			SupportsTools:  false,
			SupportsImages: false,
		},
	}, nil
}

// ErrorProvider is a test provider that throws proper errors instead of mocking
type ErrorProvider struct {
	name string
}

func (m *ErrorProvider) GetName() string {
	return m.name
}

func (m *ErrorProvider) GenerateResponse(ctx context.Context, messages []types.Message, options types.RequestOptions) (string, *types.ResponseMetadata, error) {
	return "", nil, fmt.Errorf("%s LLM provider not implemented - real API integration required", m.name)
}

func (m *ErrorProvider) GenerateResponseStream(ctx context.Context, messages []types.Message, options types.RequestOptions, writer io.Writer) (*types.ResponseMetadata, error) {
	return nil, fmt.Errorf("%s LLM provider streaming not implemented - real API integration required", m.name)
}

func (m *ErrorProvider) IsAvailable(ctx context.Context) error {
	return fmt.Errorf("%s LLM provider not available - real API integration required", m.name)
}

func (m *ErrorProvider) EstimateTokens(messages []types.Message) (int, error) {
	return 0, fmt.Errorf("%s LLM provider token estimation not implemented - real API integration required", m.name)
}

func (m *ErrorProvider) CalculateCost(usage types.TokenUsage) float64 {
	return 0.0 // Can't calculate cost without real provider
}

func (m *ErrorProvider) GetModels(ctx context.Context) ([]types.ModelInfo, error) {
	return nil, fmt.Errorf("%s LLM provider model listing not implemented - real API integration required", m.name)
}

func TestNewFactory(t *testing.T) {
	registry := NewRegistry()
	factory := NewFactory(registry)

	if factory == nil {
		t.Error("Expected factory to be created")
	}

	if factory.registry != registry {
		t.Error("Expected factory to use provided registry")
	}
}

func TestCreateProvider(t *testing.T) {
	registry := NewRegistry()

	// Register a mock provider for testing
	mockFactory := &MockProviderFactory{name: "openai"}
	registry.Register(mockFactory)

	factory := NewFactory(registry)

	tests := []struct {
		name        string
		config      *types.ProviderConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "provider configuration is required",
		},
		{
			name: "empty name",
			config: &types.ProviderConfig{
				Name:    "",
				Enabled: true,
			},
			expectError: true,
			errorMsg:    "provider name is required",
		},
		{
			name: "disabled provider",
			config: &types.ProviderConfig{
				Name:    "openai",
				Enabled: false,
			},
			expectError: true,
			errorMsg:    "is disabled",
		},
		{
			name: "unknown provider",
			config: &types.ProviderConfig{
				Name:    "nonexistent",
				Model:   "some-model",
				APIKey:  "test-key",
				Enabled: true,
			},
			expectError: true,
			errorMsg:    "is not registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProvider(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !containsError(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				if provider != nil {
					t.Error("Expected nil provider when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if provider == nil {
					t.Error("Expected provider to be created when no error")
				}
			}
		})
	}
}

func TestCreateProviderByName(t *testing.T) {
	registry := NewRegistry()
	factory := NewFactory(registry)

	tests := []struct {
		name        string
		provName    string
		model       string
		apiKey      string
		expectError bool
	}{
		{
			name:        "empty provider name",
			provName:    "",
			model:       "gpt-4",
			apiKey:      "test-key",
			expectError: true,
		},
		{
			name:        "unknown provider",
			provName:    "nonexistent",
			model:       "some-model",
			apiKey:      "test-key",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProviderByName(tt.provName, tt.model, tt.apiKey)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if provider != nil {
					t.Error("Expected nil provider when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if provider == nil {
					t.Error("Expected provider to be created when no error")
				}
			}
		})
	}
}

func TestGetAvailableProviders(t *testing.T) {
	registry := NewRegistry()

	// Register some mock providers
	mockFactory1 := &MockProviderFactory{name: "openai"}
	mockFactory2 := &MockProviderFactory{name: "gemini"}
	mockFactory3 := &MockProviderFactory{name: "ollama"}
	registry.Register(mockFactory1)
	registry.Register(mockFactory2)
	registry.Register(mockFactory3)

	factory := NewFactory(registry)

	providers := factory.GetAvailableProviders()

	// Should have at least some providers registered
	if len(providers) == 0 {
		t.Error("Expected at least one available provider")
	}

	// Check for common providers
	expectedProviders := []string{"openai", "gemini", "ollama"}
	for _, expected := range expectedProviders {
		found := false
		for _, actual := range providers {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected provider '%s' to be available", expected)
		}
	}
}

func TestValidateProviderConfig(t *testing.T) {
	registry := NewRegistry()
	factory := NewFactory(registry)

	tests := []struct {
		name        string
		config      *types.ProviderConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "provider configuration is required",
		},
		{
			name: "empty name",
			config: &types.ProviderConfig{
				Name: "",
			},
			expectError: true,
			errorMsg:    "provider name is required",
		},
		{
			name: "unknown provider",
			config: &types.ProviderConfig{
				Name: "nonexistent",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := factory.ValidateProviderConfig(tt.config)

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
			}
		})
	}
}

func TestAutoDetectProvider(t *testing.T) {
	registry := NewRegistry()

	// Register mock providers
	mockFactory1 := &MockProviderFactory{name: "openai"}
	mockFactory2 := &MockProviderFactory{name: "gemini"}
	registry.Register(mockFactory1)
	registry.Register(mockFactory2)

	factory := NewFactory(registry)

	tests := []struct {
		name        string
		configs     []*types.ProviderConfig
		expectError bool
		expected    string
	}{
		{
			name:        "empty configs",
			configs:     []*types.ProviderConfig{},
			expectError: true,
		},
		{
			name: "single valid config",
			configs: []*types.ProviderConfig{
				{
					Name:    "openai",
					Model:   "gpt-4",
					APIKey:  "test-key",
					Enabled: true,
				},
			},
			expectError: false,
			expected:    "openai",
		},
		{
			name: "multiple configs - priority order",
			configs: []*types.ProviderConfig{
				{
					Name:    "ollama",
					Model:   "llama2",
					Enabled: true,
				},
				{
					Name:    "openai",
					Model:   "gpt-4",
					APIKey:  "test-key",
					Enabled: true,
				},
			},
			expectError: false,
			expected:    "openai", // OpenAI has higher priority than Ollama
		},
		{
			name: "all disabled",
			configs: []*types.ProviderConfig{
				{
					Name:    "openai",
					Model:   "gpt-4",
					APIKey:  "test-key",
					Enabled: false,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := factory.AutoDetectProvider(tt.configs)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if config != nil {
					t.Error("Expected nil config when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if config == nil {
					t.Error("Expected config to be returned when no error")
				} else if config.Name != tt.expected {
					t.Errorf("Expected provider '%s', got '%s'", tt.expected, config.Name)
				}
			}
		})
	}
}

func TestCreateMultipleProviders(t *testing.T) {
	registry := NewRegistry()

	// Register mock providers
	mockFactory1 := &MockProviderFactory{name: "openai"}
	mockFactory2 := &MockProviderFactory{name: "gemini"}
	registry.Register(mockFactory1)
	registry.Register(mockFactory2)

	factory := NewFactory(registry)

	tests := []struct {
		name        string
		configs     []*types.ProviderConfig
		expectError bool
		minExpected int
	}{
		{
			name: "single valid provider",
			configs: []*types.ProviderConfig{
				{
					Name:    "openai",
					Model:   "gpt-4",
					APIKey:  "test-key",
					Enabled: true,
				},
			},
			expectError: false,
			minExpected: 1,
		},
		{
			name: "mixed valid and disabled",
			configs: []*types.ProviderConfig{
				{
					Name:    "openai",
					Model:   "gpt-4",
					APIKey:  "test-key",
					Enabled: true,
				},
				{
					Name:    "gemini",
					Model:   "gemini-pro",
					Enabled: false, // Disabled
				},
			},
			expectError: false,
			minExpected: 1, // Only enabled ones
		},
		{
			name: "all disabled",
			configs: []*types.ProviderConfig{
				{
					Name:    "openai",
					Model:   "gpt-4",
					APIKey:  "test-key",
					Enabled: false,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers, err := factory.CreateMultipleProviders(tt.configs)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if len(providers) != 0 {
					t.Error("Expected empty providers map when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(providers) < tt.minExpected {
					t.Errorf("Expected at least %d providers, got %d", tt.minExpected, len(providers))
				}
			}
		})
	}
}

func TestGetProviderCapabilities(t *testing.T) {
	registry := NewRegistry()
	factory := NewFactory(registry)

	tests := []struct {
		name         string
		providerName string
		expectError  bool
		expectedName string
	}{
		{
			name:         "openai capabilities",
			providerName: "openai",
			expectError:  false,
			expectedName: "openai",
		},
		{
			name:         "gemini capabilities",
			providerName: "gemini",
			expectError:  false,
			expectedName: "gemini",
		},
		{
			name:         "ollama capabilities",
			providerName: "ollama",
			expectError:  false,
			expectedName: "ollama",
		},
		{
			name:         "unknown provider",
			providerName: "nonexistent",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities, err := factory.GetProviderCapabilities(tt.providerName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if capabilities != nil {
					t.Error("Expected nil capabilities when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if capabilities == nil {
					t.Error("Expected capabilities to be returned")
				} else {
					if capabilities.Name != tt.expectedName {
						t.Errorf("Expected capabilities name '%s', got '%s'", tt.expectedName, capabilities.Name)
					}
					// Basic capability checks
					if len(capabilities.SupportedModels) == 0 {
						t.Error("Expected at least one supported model")
					}
					if capabilities.MaxTokens <= 0 {
						t.Error("Expected positive max tokens")
					}
				}
			}
		})
	}
}

func TestNewGlobalFactory(t *testing.T) {
	factory := NewGlobalFactory()

	if factory == nil {
		t.Error("Expected global factory to be created")
	}

	if factory.registry == nil {
		t.Error("Expected global factory to have a registry")
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
