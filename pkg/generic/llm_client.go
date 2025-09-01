package generic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LLMClient handles interactions with LLM providers
type LLMClient struct {
	config LLMConfig
	logger *slog.Logger
}

// LLMResponse represents a response from the LLM
type LLMResponse struct {
	Content    string                 `json:"content"`
	TokensUsed int                    `json:"tokens_used"`
	Cost       float64                `json:"cost"`
	Model      string                 `json:"model"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// GetConfig returns the LLM configuration
func (llm *LLMClient) GetConfig() LLMConfig {
	return llm.config
}

// NewLLMClient creates a new LLM client
func NewLLMClient(config LLMConfig, logger *slog.Logger) (*LLMClient, error) {
	// Resolve API key from layered config first, then environment if not provided
	if config.APIKey == "" {
		config.APIKey = getAPIKeyFromConfig(config.Provider)
		logger.Debug("API key from config", "provider", config.Provider, "found", config.APIKey != "")
		if config.APIKey == "" {
			config.APIKey = getAPIKeyFromEnv(config.Provider)
			logger.Debug("API key from env", "provider", config.Provider, "found", config.APIKey != "")
		}

		// If still no API key, prompt user to enter one
		if config.APIKey == "" {
			logger.Info("No API key found, prompting user", "provider", config.Provider)
			var err error
			config.APIKey, err = promptForAPIKey(config.Provider)
			if err != nil {
				return nil, fmt.Errorf("failed to get API key for %s: %w", config.Provider, err)
			}
		}
	}

	// Validate we have an API key
	if config.APIKey == "" {
		return nil, fmt.Errorf("no API key available for provider %s", config.Provider)
	}

	logger.Info("LLM client initialized", "provider", config.Provider, "model", config.Model, "has_api_key", config.APIKey != "")

	return &LLMClient{
		config: config,
		logger: logger,
	}, nil
}

// getAPIKeyFromEnv gets API key from environment variables based on provider
func getAPIKeyFromEnv(provider string) string {
	// Common environment variable patterns for different providers
	envVars := map[string][]string{
		"openai":    {"OPENAI_API_KEY"},
		"anthropic": {"ANTHROPIC_API_KEY", "CLAUDE_API_KEY"},
		"gemini":    {"GEMINI_API_KEY", "GOOGLE_API_KEY"},
		"deepinfra": {"DEEPINFRA_API_KEY", "DEEPINFRA_TOKEN"},
		"groq":      {"GROQ_API_KEY"},
		"ollama":    {}, // Ollama typically doesn't use API keys
	}

	if envNames, exists := envVars[strings.ToLower(provider)]; exists {
		for _, envName := range envNames {
			if apiKey := os.Getenv(envName); apiKey != "" {
				return apiKey
			}
		}
	}

	// Fallback: try generic API_KEY environment variable
	return os.Getenv("API_KEY")
}

// getAPIKeyFromConfig gets API key from layered configuration
func getAPIKeyFromConfig(provider string) string {
	// Import the layered config package
	layeredConfig := newLayeredProvider()
	if layeredConfig == nil {
		return ""
	}

	// Load config from files
	layeredConfig.ReloadConfig()

	// Try to get provider config
	providerConfig, err := layeredConfig.GetProviderConfig(provider)
	if err != nil {
		return ""
	}

	return providerConfig.APIKey
}

// newLayeredProvider creates a layered config provider (helper function)
func newLayeredProvider() configProvider {
	// This is a simplified version - in practice you'd import the actual provider
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".ledit", "config.json")

	// Load config from file if it exists
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}

	return &simpleConfigProvider{config: config}
}

// Simple config provider interface for our needs
type configProvider interface {
	GetProviderConfig(providerName string) (*providerConfig, error)
	ReloadConfig() error
}

// Simple provider config struct
type providerConfig struct {
	APIKey string `json:"api_key"`
	Name   string `json:"name"`
	Model  string `json:"model"`
}

// Simple implementation for loading from .ledit/config.json
type simpleConfigProvider struct {
	config map[string]interface{}
}

func (s *simpleConfigProvider) ReloadConfig() error {
	return nil // Already loaded in constructor
}

func (s *simpleConfigProvider) GetProviderConfig(providerName string) (*providerConfig, error) {
	// Look for providers.{providerName} in config
	providersKey := "providers"
	if providers, exists := s.config[providersKey]; exists {
		if providersMap, ok := providers.(map[string]interface{}); ok {
			if providerData, exists := providersMap[providerName]; exists {
				if providerMap, ok := providerData.(map[string]interface{}); ok {
					config := &providerConfig{Name: providerName}
					if apiKey, exists := providerMap["api_key"]; exists {
						if keyStr, ok := apiKey.(string); ok {
							config.APIKey = keyStr
						}
					}
					if model, exists := providerMap["model"]; exists {
						if modelStr, ok := model.(string); ok {
							config.Model = modelStr
						}
					}
					return config, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("provider %s not found in config", providerName)
}

// promptForAPIKey prompts the user to enter an API key for the given provider
func promptForAPIKey(provider string) (string, error) {
	fmt.Printf("API key for %s not found. Please enter your %s API key: ", provider, provider)

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to read API key")
	}

	apiKey := strings.TrimSpace(scanner.Text())
	if apiKey == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}

	// Save the API key to config
	if err := saveAPIKeyToConfig(provider, apiKey); err != nil {
		// Log warning but don't fail - the key can still be used for this session
		fmt.Printf("Warning: failed to save API key to config: %v\n", err)
	} else {
		fmt.Printf("API key saved to ~/.ledit/config.json\n")
	}

	return apiKey, nil
}

// saveAPIKeyToConfig saves an API key to the configuration file
func saveAPIKeyToConfig(provider, apiKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".ledit")
	configPath := filepath.Join(configDir, "config.json")

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load existing config or create new one
	var config map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		// Create new config structure
		config = map[string]interface{}{
			"providers": map[string]interface{}{},
			"agent": map[string]interface{}{
				"max_retries":      3,
				"default_provider": provider,
			},
		}
	}

	// Ensure providers section exists
	providers, ok := config["providers"].(map[string]interface{})
	if !ok {
		providers = make(map[string]interface{})
		config["providers"] = providers
	}

	// Update or create provider config
	providerConfig, ok := providers[provider].(map[string]interface{})
	if !ok {
		providerConfig = make(map[string]interface{})
		providers[provider] = providerConfig
	}

	// Set the API key
	providerConfig["api_key"] = apiKey

	// Add default model if not present
	if _, hasModel := providerConfig["model"]; !hasModel {
		switch provider {
		case "deepinfra":
			providerConfig["model"] = "deepseek-ai/DeepSeek-V3.1"
			providerConfig["base_url"] = "https://api.deepinfra.com/v1/openai"
		case "openai":
			providerConfig["model"] = "gpt-4"
		case "anthropic":
			providerConfig["model"] = "claude-3-sonnet-20240229"
		case "gemini":
			providerConfig["model"] = "gemini-pro"
		}
	}

	// Write updated config back to file
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Chat sends a chat message to the LLM
func (llm *LLMClient) Chat(ctx context.Context, messages []Message) (*LLMResponse, error) {
	llm.logger.Info("Sending chat request to LLM",
		"provider", llm.config.Provider,
		"model", llm.config.Model,
		"message_count", len(messages))

	// TODO: Implement actual LLM provider integrations
	switch llm.config.Provider {
	case "openai":
		return llm.chatOpenAI(ctx, messages)
	case "anthropic":
		return llm.chatAnthropic(ctx, messages)
	case "gemini":
		return llm.chatGemini(ctx, messages)
	case "ollama":
		return llm.chatOllama(ctx, messages)
	case "deepinfra":
		return llm.chatDeepInfra(ctx, messages)
	case "groq":
		return llm.chatGroq(ctx, messages)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", llm.config.Provider)
	}
}

// Complete generates a completion from a prompt
func (llm *LLMClient) Complete(ctx context.Context, prompt string) (*LLMResponse, error) {
	messages := []Message{
		{Role: "user", Content: prompt},
	}
	return llm.Chat(ctx, messages)
}

// CompleteWithSystem generates a completion with a system prompt
func (llm *LLMClient) CompleteWithSystem(ctx context.Context, systemPrompt, userPrompt string) (*LLMResponse, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	return llm.Chat(ctx, messages)
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Provider-specific implementations (placeholders for now)
func (llm *LLMClient) chatOpenAI(ctx context.Context, messages []Message) (*LLMResponse, error) {
	// TODO: Implement OpenAI API integration
	return &LLMResponse{
		Content:    "Placeholder response from OpenAI",
		TokensUsed: 100,
		Cost:       0.002,
		Model:      llm.config.Model,
		Metadata:   map[string]interface{}{"provider": "openai"},
	}, nil
}

func (llm *LLMClient) chatAnthropic(ctx context.Context, messages []Message) (*LLMResponse, error) {
	// TODO: Implement Anthropic API integration
	return &LLMResponse{
		Content:    "Placeholder response from Anthropic",
		TokensUsed: 120,
		Cost:       0.003,
		Model:      llm.config.Model,
		Metadata:   map[string]interface{}{"provider": "anthropic"},
	}, nil
}

func (llm *LLMClient) chatGemini(ctx context.Context, messages []Message) (*LLMResponse, error) {
	// TODO: Implement Gemini API integration
	return &LLMResponse{
		Content:    "Placeholder response from Gemini",
		TokensUsed: 80,
		Cost:       0.001,
		Model:      llm.config.Model,
		Metadata:   map[string]interface{}{"provider": "gemini"},
	}, nil
}

func (llm *LLMClient) chatOllama(ctx context.Context, messages []Message) (*LLMResponse, error) {
	// TODO: Implement Ollama API integration
	return &LLMResponse{
		Content:    "Placeholder response from Ollama",
		TokensUsed: 90,
		Cost:       0.0, // Ollama is typically free
		Model:      llm.config.Model,
		Metadata:   map[string]interface{}{"provider": "ollama"},
	}, nil
}

func (llm *LLMClient) chatDeepInfra(ctx context.Context, messages []Message) (*LLMResponse, error) {
	return llm.callOpenAICompatibleAPI(ctx, messages, "https://api.deepinfra.com/v1/openai", "deepinfra")
}

func (llm *LLMClient) chatGroq(ctx context.Context, messages []Message) (*LLMResponse, error) {
	// TODO: Implement Groq API integration
	return &LLMResponse{
		Content:    "Placeholder response from Groq",
		TokensUsed: 95,
		Cost:       0.001,
		Model:      llm.config.Model,
		Metadata:   map[string]interface{}{"provider": "groq"},
	}, nil
}

// callOpenAICompatibleAPI makes a call to an OpenAI-compatible API
func (llm *LLMClient) callOpenAICompatibleAPI(ctx context.Context, messages []Message, baseURL, providerName string) (*LLMResponse, error) {
	// Log the API key status (without revealing the key)
	llm.logger.Debug("Making API call", "provider", providerName, "baseURL", baseURL, "has_api_key", llm.config.APIKey != "", "api_key_length", len(llm.config.APIKey))
	// Convert messages to OpenAI format
	openaiMessages := make([]OpenAIMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	request := OpenAIRequest{
		Model:    llm.config.Model,
		Messages: openaiMessages,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+llm.config.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to %s: %w", providerName, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(responseBody))
	}

	var apiResponse OpenAIResponse
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// Calculate simple cost estimate (this would be provider-specific in reality)
	cost := float64(apiResponse.Usage.TotalTokens) * 0.002 / 1000 // rough estimate

	return &LLMResponse{
		Content:    apiResponse.Choices[0].Message.Content,
		TokensUsed: apiResponse.Usage.TotalTokens,
		Cost:       cost,
		Model:      llm.config.Model,
		Metadata:   map[string]interface{}{"provider": providerName},
	}, nil
}

// OpenAI API types for compatibility
type OpenAIRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

type OpenAIChoice struct {
	Message OpenAIMessage `json:"message"`
}

type OpenAIUsage struct {
	TotalTokens int `json:"total_tokens"`
}
