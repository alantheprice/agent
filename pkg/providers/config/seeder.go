package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
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
	credentialsPath, err := getCredentialsPath()
	if err != nil {
		return err
	}

	var apiKeys APIKeys

	// Try to read existing file, if it doesn't exist, create default structure
	if data, err := ioutil.ReadFile(credentialsPath); err == nil {
		if err := json.Unmarshal(data, &apiKeys); err != nil {
			return fmt.Errorf("failed to parse existing API credentials: %w", err)
		}
	} else {
		// Initialize with empty structure
		apiKeys = APIKeys{
			APIKeys:     make(map[string]string),
			Description: "API keys for LLM providers. Keys are loaded from environment variables or this file.",
		}

		// Create the .agents directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(credentialsPath), 0700); err != nil {
			return fmt.Errorf("failed to create credentials directory: %w", err)
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

		// Write updated API credentials
		data, err := json.MarshalIndent(apiKeys, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal API credentials: %w", err)
		}

		if err := ioutil.WriteFile(credentialsPath, data, 0600); err != nil {
			return fmt.Errorf("failed to write API credentials: %w", err)
		}

		fmt.Printf("Successfully seeded API keys from %s to %s\n", legacyPath, credentialsPath)
	} else {
		fmt.Println("No API keys found to seed")
	}

	return nil
}

// getCredentialsPath returns the path to the credentials file
func getCredentialsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".agents", "credentials.json"), nil
}

// LoadAPIKeys loads API keys from the credentials file
func LoadAPIKeys() (*APIKeys, error) {
	credentialsPath, err := getCredentialsPath()
	if err != nil {
		return nil, err
	}
	
	data, err := ioutil.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read API credentials from %s: %w", credentialsPath, err)
	}

	var apiKeys APIKeys
	if err := json.Unmarshal(data, &apiKeys); err != nil {
		return nil, fmt.Errorf("failed to parse API credentials: %w", err)
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

// GetAPIKeyForProvider gets the API key for a provider, with automatic credential management
func GetAPIKeyForProvider(providerName string) string {
	return GetAPIKeyForProviderWithPrompt(providerName, false)
}

// GetAPIKeyForProviderWithPrompt gets the API key for a provider, optionally prompting user
func GetAPIKeyForProviderWithPrompt(providerName string, allowPrompt bool) string {
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

	// Try to load from credentials file
	apiKeys, err := LoadAPIKeys()
	if err != nil {
		// Credentials file doesn't exist, create it if prompting is allowed
		if allowPrompt {
			return handleMissingCredentials(providerName, provider.Name)
		}
		return ""
	}

	// Check if provider key exists in credentials file
	if key, exists := apiKeys.APIKeys[providerName]; exists && key != "" {
		return key
	}

	// Key is missing or empty, prompt user if allowed
	if allowPrompt {
		return promptAndSaveAPIKey(providerName, provider.Name)
	}

	return ""
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

// SetAPIKey sets an API key in the credentials file
func SetAPIKey(providerName, apiKey string) error {
	credentialsPath, err := getCredentialsPath()
	if err != nil {
		return err
	}

	apiKeys, err := LoadAPIKeys()
	if err != nil {
		// Create new structure if file doesn't exist
		apiKeys = &APIKeys{
			APIKeys:     make(map[string]string),
			Description: "API keys for LLM providers. Keys are loaded from environment variables or this file.",
		}

		// Create the .agents directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(credentialsPath), 0700); err != nil {
			return fmt.Errorf("failed to create credentials directory: %w", err)
		}
	}

	apiKeys.APIKeys[providerName] = apiKey
	apiKeys.LastUpdated = time.Now().Format(time.RFC3339)

	data, err := json.MarshalIndent(apiKeys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal API credentials: %w", err)
	}

	if err := ioutil.WriteFile(credentialsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write API credentials: %w", err)
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

// handleMissingCredentials handles the case when credentials file doesn't exist
func handleMissingCredentials(providerName, displayName string) string {
	fmt.Printf("🔐 Credentials file not found. Setting up credentials for %s...\n", displayName)
	
	// Create empty credentials structure
	credentialsPath, err := getCredentialsPath()
	if err != nil {
		fmt.Printf("❌ Error getting credentials path: %v\n", err)
		return ""
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(credentialsPath), 0700); err != nil {
		fmt.Printf("❌ Error creating credentials directory: %v\n", err)
		return ""
	}

	// Prompt for API key
	apiKey := promptForAPIKey(providerName, displayName)
	if apiKey == "" {
		return ""
	}

	// Create new credentials file with the key
	apiKeys := &APIKeys{
		APIKeys: map[string]string{
			"openai":     "",
			"gemini":     "",
			"ollama":     "",
			"deepinfra":  "",
			"groq":       "",
			"cerebras":   "",
			"deepseek":   "",
			"github":     "",
			"lambda-ai":  "",
			"jinai":      "",
		},
		LastUpdated: time.Now().Format(time.RFC3339),
		Description: "API keys for LLM providers. Keys are loaded from environment variables or this file.",
	}

	// Set the specific key
	apiKeys.APIKeys[providerName] = apiKey

	// Save to file
	if err := saveCredentials(apiKeys); err != nil {
		fmt.Printf("❌ Error saving credentials: %v\n", err)
		return ""
	}

	fmt.Printf("✅ API key saved successfully to %s\n", credentialsPath)
	return apiKey
}

// promptAndSaveAPIKey prompts user for API key and saves it
func promptAndSaveAPIKey(providerName, displayName string) string {
	apiKey := promptForAPIKey(providerName, displayName)
	if apiKey == "" {
		return ""
	}

	// Save the key
	if err := SetAPIKey(providerName, apiKey); err != nil {
		fmt.Printf("❌ Error saving API key: %v\n", err)
		return ""
	}

	fmt.Printf("✅ API key saved successfully for %s\n", displayName)
	return apiKey
}

// promptForAPIKey prompts the user to enter an API key
func promptForAPIKey(providerName, displayName string) string {
	fmt.Printf("\n🔑 API key for %s (%s) is required.\n", displayName, providerName)
	
	// Show helpful information about where to get the key
	switch providerName {
	case "openai":
		fmt.Println("   Get your key at: https://platform.openai.com/api-keys")
	case "gemini":
		fmt.Println("   Get your key at: https://makersuite.google.com/app/apikey")
	case "deepinfra":
		fmt.Println("   Get your key at: https://deepinfra.com/dash/api_keys")
	case "groq":
		fmt.Println("   Get your key at: https://console.groq.com/keys")
	case "cerebras":
		fmt.Println("   Get your key at: https://cloud.cerebras.ai/platform")
	case "deepseek":
		fmt.Println("   Get your key at: https://platform.deepseek.com/api_keys")
	case "github":
		fmt.Println("   Create a personal access token at: https://github.com/settings/tokens")
	}

	fmt.Printf("\nEnter your %s API key (input will be hidden): ", displayName)

	// Read password-style input (hidden)
	apiKey, err := readPassword()
	if err != nil {
		fmt.Printf("\n❌ Error reading API key: %v\n", err)
		return ""
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		fmt.Println("\n❌ No API key entered. Skipping configuration.")
		return ""
	}

	// Basic validation
	if len(apiKey) < 10 {
		fmt.Println("\n⚠️  Warning: API key seems too short. Please verify it's correct.")
	}

	return apiKey
}

// readPassword reads a password from stdin without echoing
func readPassword() (string, error) {
	fd := int(syscall.Stdin)
	if term.IsTerminal(fd) {
		bytePassword, err := term.ReadPassword(fd)
		if err != nil {
			return "", err
		}
		return string(bytePassword), nil
	} else {
		// Fallback for non-terminal input
		reader := bufio.NewReader(os.Stdin)
		password, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(password), nil
	}
}

// saveCredentials saves the API keys to the credentials file
func saveCredentials(apiKeys *APIKeys) error {
	credentialsPath, err := getCredentialsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(apiKeys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal API credentials: %w", err)
	}

	if err := ioutil.WriteFile(credentialsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write API credentials: %w", err)
	}

	return nil
}