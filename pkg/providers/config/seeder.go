package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// APIKeys represents the structure of the API keys file
type APIKeys struct {
	APIKeys     map[string]string `json:"api_keys"`
	LastUpdated string            `json:"last_updated"`
	Description string            `json:"description"`
}

// ProviderDefinition represents a provider definition from providers.json
type ProviderDefinition struct {
	Name                    string               `json:"name"`
	BaseURL                 string               `json:"base_url"`
	APIKeyEnv               string               `json:"api_key_env"`
	SupportedModels         []string             `json:"supported_models"`
	SupportedEmbeddingModels []string            `json:"supported_embedding_models,omitempty"`
	Capabilities            ProviderCapabilities `json:"capabilities"`
	DefaultModel            string               `json:"default_model"`
	DefaultEmbeddingModel   string               `json:"default_embedding_model,omitempty"`
	Enabled                 bool                 `json:"enabled"`
}

// ProviderCapabilities represents provider capabilities
type ProviderCapabilities struct {
	SupportsTools      bool `json:"supports_tools"`
	SupportsImages     bool `json:"supports_images"`
	SupportsStream     bool `json:"supports_stream"`
	SupportsEmbeddings bool `json:"supports_embeddings,omitempty"`
	MaxTokens          int  `json:"max_tokens"`
}

// ProvidersConfig represents the providers.json structure
type ProvidersConfig struct {
	Providers       map[string]ProviderDefinition `json:"providers"`
	DefaultProvider string                        `json:"default_provider"`
	PriorityOrder   []string                      `json:"priority_order"`
}

// LegacyAPIKeys represents the structure from ~/.ledit/api_keys.json
type LegacyAPIKeys map[string]string

// SeedAPIKeysFromLedit seeds API keys from the legacy ~/.ledit/api_keys.json file
func SeedAPIKeysFromLedit() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Read legacy API keys
	legacyPath := filepath.Join(homeDir, ".ledit", "api_keys.json")
	legacyData, err := ioutil.ReadFile(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("legacy API keys file not found at %s", legacyPath)
		}
		return fmt.Errorf("failed to read legacy API keys: %w", err)
	}

	var legacyKeys LegacyAPIKeys
	if err := json.Unmarshal(legacyData, &legacyKeys); err != nil {
		return fmt.Errorf("failed to parse legacy API keys: %w", err)
	}

	// Read current API secrets structure
	secretsPath := filepath.Join("configs", "api_secrets.json")
	var apiKeys APIKeys

	// Try to read existing file, if it doesn't exist, create default structure
	if data, err := ioutil.ReadFile(secretsPath); err == nil {
		if err := json.Unmarshal(data, &apiKeys); err != nil {
			return fmt.Errorf("failed to parse existing API secrets: %w", err)
		}
	} else {
		// Initialize with empty structure
		apiKeys = APIKeys{
			APIKeys:     make(map[string]string),
			Description: "API keys for LLM providers. Keys are loaded from environment variables or this file.",
		}
	}

	// Map legacy keys to new format
	keyMapping := map[string]string{
		"openai":     "openai",
		"gemini":     "gemini",
		"deepinfra":  "deepinfra",
		"cerebras":   "cerebras",
		"deepseek":   "deepseek",
		"github":     "github",
		"JinaAI":     "jinai",
		"lambda-ai":  "lambda-ai",
	}

	// Update API keys from legacy file
	updated := false
	for legacyKey, newKey := range keyMapping {
		if value, exists := legacyKeys[legacyKey]; exists && value != "" {
			apiKeys.APIKeys[newKey] = value
			updated = true
		}
	}

	// Ensure all providers have entries (even if empty)
	requiredKeys := []string{
		"openai", "gemini", "ollama", "deepinfra", "groq", 
		"cerebras", "deepseek", "github", "lambda-ai", "jinai",
	}
	
	for _, key := range requiredKeys {
		if _, exists := apiKeys.APIKeys[key]; !exists {
			apiKeys.APIKeys[key] = ""
		}
	}

	if updated {
		apiKeys.LastUpdated = time.Now().Format(time.RFC3339)

		// Write updated API secrets
		data, err := json.MarshalIndent(apiKeys, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal API secrets: %w", err)
		}

		if err := ioutil.WriteFile(secretsPath, data, 0600); err != nil {
			return fmt.Errorf("failed to write API secrets: %w", err)
		}

		fmt.Printf("Successfully seeded API keys from %s\n", legacyPath)
	} else {
		fmt.Println("No API keys found to seed")
	}

	return nil
}

// LoadAPIKeys loads API keys from the secrets file
func LoadAPIKeys() (*APIKeys, error) {
	secretsPath := filepath.Join("configs", "api_secrets.json")
	
	data, err := ioutil.ReadFile(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read API secrets: %w", err)
	}

	var apiKeys APIKeys
	if err := json.Unmarshal(data, &apiKeys); err != nil {
		return nil, fmt.Errorf("failed to parse API secrets: %w", err)
	}

	return &apiKeys, nil
}

// LoadProvidersConfig loads the providers configuration
func LoadProvidersConfig() (*ProvidersConfig, error) {
	configPath := filepath.Join("configs", "providers.json")
	
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read providers config: %w", err)
	}

	var config ProvidersConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse providers config: %w", err)
	}

	return &config, nil
}

// GetAPIKeyForProvider gets the API key for a provider, checking environment variables first
func GetAPIKeyForProvider(providerName string) string {
	// Load providers config to get the env var name
	config, err := LoadProvidersConfig()
	if err != nil {
		return ""
	}

	provider, exists := config.Providers[providerName]
	if !exists {
		return ""
	}

	// Check environment variable first
	if provider.APIKeyEnv != "" {
		if envKey := os.Getenv(provider.APIKeyEnv); envKey != "" {
			return envKey
		}
	}

	// Fallback to secrets file
	apiKeys, err := LoadAPIKeys()
	if err != nil {
		return ""
	}

	return apiKeys.APIKeys[providerName]
}

// GetEmbeddingModelForProvider gets the default embedding model for a provider
func GetEmbeddingModelForProvider(providerName string) string {
	// Load providers config to get the embedding model
	config, err := LoadProvidersConfig()
	if err != nil {
		return ""
	}

	provider, exists := config.Providers[providerName]
	if !exists {
		return ""
	}

	return provider.DefaultEmbeddingModel
}

// SetAPIKey sets an API key in the secrets file
func SetAPIKey(providerName, apiKey string) error {
	apiKeys, err := LoadAPIKeys()
	if err != nil {
		// Create new structure if file doesn't exist
		apiKeys = &APIKeys{
			APIKeys:     make(map[string]string),
			Description: "API keys for LLM providers. Keys are loaded from environment variables or this file.",
		}
	}

	apiKeys.APIKeys[providerName] = apiKey
	apiKeys.LastUpdated = time.Now().Format(time.RFC3339)

	secretsPath := filepath.Join("configs", "api_secrets.json")
	data, err := json.MarshalIndent(apiKeys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal API secrets: %w", err)
	}

	if err := ioutil.WriteFile(secretsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write API secrets: %w", err)
	}

	return nil
}

// ListProviders returns a list of available providers with their status
func ListProviders() (map[string]bool, error) {
	config, err := LoadProvidersConfig()
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for name, provider := range config.Providers {
		hasKey := GetAPIKeyForProvider(name) != ""
		result[name] = provider.Enabled && hasKey
	}

	return result, nil
}