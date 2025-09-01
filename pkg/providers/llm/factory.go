package llm

import (
	"fmt"
	"strings"

	"github.com/alantheprice/agent/pkg/interfaces"
	"github.com/alantheprice/agent/pkg/interfaces/types"
	"github.com/alantheprice/agent/pkg/providers/config"
)

// Factory provides convenient methods for creating providers
type Factory struct {
	registry *Registry
}

// NewFactory creates a new provider factory
func NewFactory(registry *Registry) *Factory {
	return &Factory{
		registry: registry,
	}
}

// CreateProvider creates a provider instance from configuration
func (f *Factory) CreateProvider(config *types.ProviderConfig) (interfaces.LLMProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("provider configuration is required")
	}

	if config.Name == "" {
		return nil, fmt.Errorf("provider name is required")
	}

	if !config.Enabled {
		return nil, fmt.Errorf("provider '%s' is disabled", config.Name)
	}

	// Normalize provider name
	providerName := strings.ToLower(config.Name)

	return f.registry.GetProvider(providerName, config)
}

// CreateProviderByName creates a provider with minimal configuration
func (f *Factory) CreateProviderByName(name, model, apiKey string) (interfaces.LLMProvider, error) {
	providerConfig := &types.ProviderConfig{
		Name:    name,
		Model:   model,
		APIKey:  apiKey,
		Enabled: true,
	}

	// Load provider definition to get base URL and other settings
	if providerDef, err := f.getProviderDefinition(name); err == nil {
		providerConfig.BaseURL = providerDef.BaseURL
		if model == "" {
			providerConfig.Model = providerDef.DefaultModel
		}
		if apiKey == "" {
			providerConfig.APIKey = config.GetAPIKeyForProvider(name)
		}
	}

	return f.CreateProvider(providerConfig)
}

// GetAvailableProviders returns a list of available provider names
func (f *Factory) GetAvailableProviders() []string {
	// Try to get from configuration first
	if providersConfig, err := config.LoadProvidersConfig(); err == nil {
		providers := make([]string, 0, len(providersConfig.Providers))
		for name, provider := range providersConfig.Providers {
			if provider.Enabled {
				providers = append(providers, name)
			}
		}
		return providers
	}

	// Fallback to registry
	return f.registry.ListProviders()
}

// ValidateProviderConfig validates a provider configuration
func (f *Factory) ValidateProviderConfig(config *types.ProviderConfig) error {
	if config == nil {
		return fmt.Errorf("provider configuration is required")
	}

	if config.Name == "" {
		return fmt.Errorf("provider name is required")
	}

	providerName := strings.ToLower(config.Name)
	return f.registry.ValidateConfig(providerName, config)
}

// AutoDetectProvider attempts to auto-detect the best available provider
func (f *Factory) AutoDetectProvider(configs []*types.ProviderConfig) (*types.ProviderConfig, error) {
	if len(configs) == 0 {
		// If no configs provided, try to load from provider definitions
		return f.autoDetectFromConfiguration()
	}

	// Load priority order from configuration
	providersConfig, err := config.LoadProvidersConfig()
	var priorityOrder []string
	if err == nil {
		priorityOrder = providersConfig.PriorityOrder
	} else {
		// Fallback to hardcoded priority
		priorityOrder = []string{"deepinfra", "deepseek", "cerebras", "gemini", "groq", "ollama", "openai"}
	}

	// First, try providers in priority order
	for _, preferredProvider := range priorityOrder {
		for _, config := range configs {
			if strings.ToLower(config.Name) == preferredProvider && config.Enabled {
				if err := f.ValidateProviderConfig(config); err == nil {
					return config, nil
				}
			}
		}
	}

	// If no priority provider works, try any enabled provider
	for _, config := range configs {
		if config.Enabled {
			if err := f.ValidateProviderConfig(config); err == nil {
				return config, nil
			}
		}
	}

	return nil, fmt.Errorf("no valid provider configurations found")
}

// CreateMultipleProviders creates multiple providers from configurations
func (f *Factory) CreateMultipleProviders(configs []*types.ProviderConfig) (map[string]interfaces.LLMProvider, error) {
	providers := make(map[string]interfaces.LLMProvider)
	errors := make([]string, 0)

	for _, config := range configs {
		if !config.Enabled {
			continue
		}

		provider, err := f.CreateProvider(config)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to create provider '%s': %v", config.Name, err))
			continue
		}

		providers[config.Name] = provider
	}

	if len(providers) == 0 {
		if len(errors) > 0 {
			return nil, fmt.Errorf("failed to create any providers: %s", strings.Join(errors, "; "))
		}
		return nil, fmt.Errorf("no enabled providers found")
	}

	return providers, nil
}

// getProviderDefinition loads a provider definition from the config
func (f *Factory) getProviderDefinition(name string) (*config.ProviderDefinition, error) {
	providersConfig, err := config.LoadProvidersConfig()
	if err != nil {
		return nil, err
	}

	providerDef, exists := providersConfig.Providers[name]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found in configuration", name)
	}

	return &providerDef, nil
}

// autoDetectFromConfiguration automatically detects the best provider from config
func (f *Factory) autoDetectFromConfiguration() (*types.ProviderConfig, error) {
	providersConfig, err := config.LoadProvidersConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load providers configuration: %w", err)
	}

	// Try providers in priority order
	for _, providerName := range providersConfig.PriorityOrder {
		providerDef, exists := providersConfig.Providers[providerName]
		if !exists || !providerDef.Enabled {
			continue
		}

		// Check if API key is available
		apiKey := config.GetAPIKeyForProvider(providerName)
		if apiKey == "" {
			continue
		}

		// Create configuration from provider definition
		providerConfig := &types.ProviderConfig{
			Name:        providerName,
			BaseURL:     providerDef.BaseURL,
			APIKey:      apiKey,
			Model:       providerDef.DefaultModel,
			Enabled:     true,
			Temperature: 0.7,
			MaxTokens:   providerDef.Capabilities.MaxTokens,
		}

		// Validate the configuration
		if err := f.ValidateProviderConfig(providerConfig); err == nil {
			return providerConfig, nil
		}
	}

	return nil, fmt.Errorf("no valid provider configurations found in system configuration")
}

// GetProviderCapabilities returns the capabilities of a provider
func (f *Factory) GetProviderCapabilities(providerName string) (*ProviderCapabilities, error) {
	// Try to load from configuration first
	if providerDef, err := f.getProviderDefinition(providerName); err == nil {
		return &ProviderCapabilities{
			Name:            providerDef.Name,
			SupportsTools:   providerDef.Capabilities.SupportsTools,
			SupportsImages:  providerDef.Capabilities.SupportsImages,
			SupportsStream:  providerDef.Capabilities.SupportsStream,
			MaxTokens:       providerDef.Capabilities.MaxTokens,
			SupportedModels: providerDef.SupportedModels,
		}, nil
	}

	// Fallback to hardcoded capabilities
	capabilities := getDefaultCapabilities(providerName)
	if capabilities == nil {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	return capabilities, nil
}

// ProviderCapabilities describes what a provider supports
type ProviderCapabilities struct {
	Name            string   `json:"name"`
	SupportsTools   bool     `json:"supports_tools"`
	SupportsImages  bool     `json:"supports_images"`
	SupportsStream  bool     `json:"supports_stream"`
	MaxTokens       int      `json:"max_tokens"`
	SupportedModels []string `json:"supported_models"`
}

// getDefaultCapabilities returns default capabilities for known providers
func getDefaultCapabilities(providerName string) *ProviderCapabilities {
	switch strings.ToLower(providerName) {
	case "openai":
		return &ProviderCapabilities{
			Name:            "openai",
			SupportsTools:   true,
			SupportsImages:  true,
			SupportsStream:  true,
			MaxTokens:       128000,
			SupportedModels: []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"},
		}
	case "gemini":
		return &ProviderCapabilities{
			Name:            "gemini",
			SupportsTools:   true,
			SupportsImages:  true,
			SupportsStream:  true,
			MaxTokens:       32768,
			SupportedModels: []string{"gemini-pro", "gemini-pro-vision"},
		}
	case "ollama":
		return &ProviderCapabilities{
			Name:            "ollama",
			SupportsTools:   false,
			SupportsImages:  false,
			SupportsStream:  true,
			MaxTokens:       4096,
			SupportedModels: []string{"llama2", "codellama", "mistral"},
		}
	case "groq":
		return &ProviderCapabilities{
			Name:            "groq",
			SupportsTools:   true,
			SupportsImages:  false,
			SupportsStream:  true,
			MaxTokens:       32768,
			SupportedModels: []string{"llama-3.1-70b", "mixtral-8x7b"},
		}
	default:
		return nil
	}
}

// NewGlobalFactory creates a factory using the global registry
func NewGlobalFactory() *Factory {
	return NewFactory(GetGlobalRegistry())
}
