package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// EmbeddingProvider defines the interface for embedding generation providers
type EmbeddingProvider interface {
	GenerateEmbedding(text string, model string) ([]float64, error)
	GetDefaultModel() string
	GetName() string
}

// OpenAIEmbeddingRequest represents a request to an OpenAI-compatible embeddings API
type OpenAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// OpenAIEmbeddingResponse represents a response from an OpenAI-compatible embeddings API
type OpenAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// OpenAIProvider implements embedding generation using OpenAI API
type OpenAIProvider struct {
	APIKey  string
	BaseURL string
}

// NewOpenAIProvider creates a new OpenAI embedding provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		APIKey:  apiKey,
		BaseURL: "https://api.openai.com/v1",
	}
}

func (p *OpenAIProvider) GetName() string {
	return "openai"
}

func (p *OpenAIProvider) GetDefaultModel() string {
	return "text-embedding-ada-002"
}

func (p *OpenAIProvider) GenerateEmbedding(text string, model string) ([]float64, error) {
	if model == "" {
		model = p.GetDefaultModel()
	}

	reqData := OpenAIEmbeddingRequest{
		Model: model,
		Input: []string{text},
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response OpenAIEmbeddingResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	return response.Data[0].Embedding, nil
}

// DeepInfraProvider implements embedding generation using DeepInfra API
type DeepInfraProvider struct {
	APIKey  string
	BaseURL string
}

// NewDeepInfraProvider creates a new DeepInfra embedding provider
func NewDeepInfraProvider(apiKey string) *DeepInfraProvider {
	return &DeepInfraProvider{
		APIKey:  apiKey,
		BaseURL: "https://api.deepinfra.com/v1/openai",
	}
}

func (p *DeepInfraProvider) GetName() string {
	return "deepinfra"
}

func (p *DeepInfraProvider) GetDefaultModel() string {
	return "sentence-transformers/all-MiniLM-L6-v2"
}

func (p *DeepInfraProvider) GenerateEmbedding(text string, model string) ([]float64, error) {
	if model == "" {
		model = p.GetDefaultModel()
	}

	reqData := OpenAIEmbeddingRequest{
		Model: model,
		Input: []string{text},
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response OpenAIEmbeddingResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	return response.Data[0].Embedding, nil
}

// EmbeddingGenerator manages embedding generation with multiple providers
type EmbeddingGenerator struct {
	providers map[string]EmbeddingProvider
}

// NewEmbeddingGenerator creates a new embedding generator
func NewEmbeddingGenerator() *EmbeddingGenerator {
	return &EmbeddingGenerator{
		providers: make(map[string]EmbeddingProvider),
	}
}

// RegisterProvider registers an embedding provider
func (g *EmbeddingGenerator) RegisterProvider(name string, provider EmbeddingProvider) {
	g.providers[name] = provider
}

// GenerateEmbedding generates an embedding using the specified provider and model
func (g *EmbeddingGenerator) GenerateEmbedding(text, providerName, model string) ([]float64, error) {
	if providerName == "" {
		providerName = "openai" // Default provider
	}

	provider, exists := g.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("embedding provider %s not found", providerName)
	}

	return provider.GenerateEmbedding(text, model)
}

// GetProviders returns a list of available provider names
func (g *EmbeddingGenerator) GetProviders() []string {
	providers := make([]string, 0, len(g.providers))
	for name := range g.providers {
		providers = append(providers, name)
	}
	return providers
}

// CreateEmbedding creates a new embedding from text content
func (g *EmbeddingGenerator) CreateEmbedding(id, embeddingType, source, content string, metadata map[string]interface{}, providerName, model string) (*Embedding, error) {
	vector, err := g.GenerateEmbedding(content, providerName, model)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Estimate token count (rough approximation)
	tokenCount := len(strings.Fields(content))

	return &Embedding{
		ID:          id,
		Type:        embeddingType,
		Source:      source,
		Content:     content,
		Vector:      vector,
		Metadata:    metadata,
		TokenCount:  tokenCount,
		LastUpdated: time.Now(),
	}, nil
}

// ParseProviderModel parses a provider:model string format
func ParseProviderModel(providerModel string) (provider, model string) {
	if providerModel == "" {
		return "openai", ""
	}

	parts := strings.SplitN(providerModel, ":", 2)
	provider = parts[0]
	if len(parts) > 1 {
		model = parts[1]
	}

	return provider, model
}
