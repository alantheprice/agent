package generic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
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

// NewLLMClient creates a new LLM client
func NewLLMClient(config LLMConfig, logger *slog.Logger) (*LLMClient, error) {
	// Resolve API key from environment if not provided
	if config.APIKey == "" {
		config.APIKey = getAPIKeyFromEnv(config.Provider)
	}

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
