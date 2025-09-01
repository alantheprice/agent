package deepinfra

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alantheprice/agent/pkg/interfaces"
	"github.com/alantheprice/agent/pkg/interfaces/types"
)

// Provider implements the DeepInfra LLM provider (OpenAI-compatible)
type Provider struct {
	config     *types.ProviderConfig
	httpClient *http.Client
}

// Factory implements the ProviderFactory interface for DeepInfra
type Factory struct{}

// GetName returns the provider name
func (f *Factory) GetName() string {
	return "deepinfra"
}

// Create creates a new DeepInfra provider instance
func (f *Factory) Create(config *types.ProviderConfig) (interfaces.LLMProvider, error) {
	if err := f.Validate(config); err != nil {
		return nil, err
	}

	return &Provider{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}, nil
}

// Validate validates the DeepInfra provider configuration
func (f *Factory) Validate(config *types.ProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required for DeepInfra provider")
	}

	if config.Model == "" {
		return fmt.Errorf("model is required for DeepInfra provider")
	}

	// Set defaults - DeepInfra uses OpenAI-compatible API
	if config.BaseURL == "" {
		config.BaseURL = "https://api.deepinfra.com/v1/openai"
	}

	if config.Timeout == 0 {
		config.Timeout = 60 // 60 seconds default
	}

	return nil
}

// GetName returns the provider name
func (p *Provider) GetName() string {
	return "deepinfra"
}

// GetModels returns available models for DeepInfra
func (p *Provider) GetModels(ctx context.Context) ([]types.ModelInfo, error) {
	// Common DeepInfra models
	return []types.ModelInfo{
		{
			Name:           "deepseek-ai/DeepSeek-V3.1",
			Provider:       "deepinfra",
			MaxTokens:      32768,
			SupportsTools:  true,
			SupportsImages: false,
		},
		{
			Name:           "meta-llama/Meta-Llama-3.1-70B-Instruct",
			Provider:       "deepinfra",
			MaxTokens:      32768,
			SupportsTools:  true,
			SupportsImages: false,
		},
		{
			Name:           "microsoft/WizardLM-2-8x22B",
			Provider:       "deepinfra",
			MaxTokens:      65536,
			SupportsTools:  true,
			SupportsImages: false,
		},
	}, nil
}

// GenerateResponse generates a response from DeepInfra
func (p *Provider) GenerateResponse(ctx context.Context, messages []types.Message, options types.RequestOptions) (string, *types.ResponseMetadata, error) {
	requestBody, err := p.buildRequest(messages, options)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build request: %w", err)
	}

	startTime := time.Now()
	resp, err := p.makeRequest(ctx, requestBody)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("DeepInfra API returned status %d: %s", resp.StatusCode, string(responseData))
	}

	var apiResponse OpenAIResponse
	if err := json.Unmarshal(responseData, &apiResponse); err != nil {
		return "", nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices returned from DeepInfra API")
	}

	content := apiResponse.Choices[0].Message.Content

	// Build metadata
	metadata := &types.ResponseMetadata{
		TokenUsage: types.TokenUsage{
			PromptTokens:     apiResponse.Usage.PromptTokens,
			CompletionTokens: apiResponse.Usage.CompletionTokens,
			TotalTokens:      apiResponse.Usage.TotalTokens,
		},
		Model:    p.config.Model,
		Provider: "deepinfra",
		Duration: time.Since(startTime),
	}

	return content, metadata, nil
}

// GenerateResponseStream generates a streaming response from DeepInfra  
func (p *Provider) GenerateResponseStream(ctx context.Context, messages []types.Message, options types.RequestOptions, writer io.Writer) (*types.ResponseMetadata, error) {
	options.Stream = true

	requestBody, err := p.buildRequest(messages, options)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	startTime := time.Now()
	resp, err := p.makeRequest(ctx, requestBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("DeepInfra API returned status %d: %s", resp.StatusCode, string(body))
	}

	var totalTokens int
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var streamResponse OpenAIStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResponse); err != nil {
				return nil, fmt.Errorf("failed to parse stream data: %w", err)
			}

			if len(streamResponse.Choices) > 0 {
				content := streamResponse.Choices[0].Delta.Content
				if content != "" {
					_, err := writer.Write([]byte(content))
					if err != nil {
						return nil, fmt.Errorf("failed to write stream content: %w", err)
					}
					// Rough token estimation
					totalTokens += len(strings.Fields(content))
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("stream reading error: %w", err)
	}

	metadata := &types.ResponseMetadata{
		TokenUsage: types.TokenUsage{
			PromptTokens:     0, // Not available in stream
			CompletionTokens: totalTokens,
			TotalTokens:      totalTokens,
		},
		Model:    p.config.Model,
		Provider: "deepinfra",
		Duration: time.Since(startTime),
	}

	return metadata, nil
}

// IsAvailable checks if the DeepInfra provider is available
func (p *Provider) IsAvailable(ctx context.Context) error {
	// Simple health check - try to make a minimal request
	messages := []types.Message{{Role: "user", Content: "test"}}
	options := types.RequestOptions{MaxTokens: 1}

	_, _, err := p.GenerateResponse(ctx, messages, options)
	return err
}

// buildRequest builds the request body for DeepInfra API (OpenAI format)
func (p *Provider) buildRequest(messages []types.Message, options types.RequestOptions) ([]byte, error) {
	// Convert to OpenAI format
	openAIMessages := make([]OpenAIMessage, len(messages))
	for i, msg := range messages {
		openAIMessages[i] = OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	request := OpenAIRequest{
		Model:       p.config.Model,
		Messages:    openAIMessages,
		MaxTokens:   options.MaxTokens,
		Temperature: options.Temperature,
		Stream:      options.Stream,
	}

	// Set defaults if not provided
	if request.MaxTokens == 0 {
		request.MaxTokens = 1000
	}
	if request.Temperature == 0 {
		request.Temperature = 0.7
	}

	return json.Marshal(request)
}

// makeRequest makes the HTTP request to DeepInfra API
func (p *Provider) makeRequest(ctx context.Context, requestBody []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	return p.httpClient.Do(req)
}

// OpenAI API request/response structures (compatible with DeepInfra)
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
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
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAIStreamResponse struct {
	Choices []OpenAIStreamChoice `json:"choices"`
}

type OpenAIStreamChoice struct {
	Delta        OpenAIDelta `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

type OpenAIDelta struct {
	Content string `json:"content"`
}

// EstimateTokens provides a rough estimate of token count
func (p *Provider) EstimateTokens(messages []types.Message) (int, error) {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content) + len(msg.Role) + 10 // Add some overhead
	}

	// Rough estimate: ~4 characters per token
	return totalChars / 4, nil
}

// CalculateCost calculates the cost for given token usage based on DeepInfra pricing
func (p *Provider) CalculateCost(usage types.TokenUsage) float64 {
	// DeepInfra pricing (approximate, competitive pricing)
	var inputCostPer1K, outputCostPer1K float64

	model := p.config.Model
	if strings.Contains(model, "deepseek") {
		inputCostPer1K = 0.0014  // $0.0014 per 1K prompt tokens
		outputCostPer1K = 0.0028 // $0.0028 per 1K completion tokens
	} else if strings.Contains(model, "llama") {
		inputCostPer1K = 0.0007  // $0.0007 per 1K prompt tokens
		outputCostPer1K = 0.0014 // $0.0014 per 1K completion tokens
	} else {
		// Default pricing for other models
		inputCostPer1K = 0.001
		outputCostPer1K = 0.002
	}

	inputCost := float64(usage.PromptTokens) * inputCostPer1K / 1000.0
	outputCost := float64(usage.CompletionTokens) * outputCostPer1K / 1000.0

	return inputCost + outputCost
}
