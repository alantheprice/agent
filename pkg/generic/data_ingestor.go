package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/alantheprice/agent/pkg/embedding"
)

// DataIngestor handles ingesting data from various sources
type DataIngestor struct {
	sources              []DataSource
	embeddingsConfig     *EmbeddingConfig
	logger               *slog.Logger
	embeddingDataSources map[string]*embedding.EmbeddingDataSource
}

// IngestedData represents data from a source
type IngestedData struct {
	Source   string                 `json:"source"`
	Type     string                 `json:"type"`
	Data     interface{}            `json:"data"`
	Metadata map[string]interface{} `json:"metadata"`
}

// NewDataIngestor creates a new data ingestor
func NewDataIngestor(sources []DataSource, embeddingsConfig *EmbeddingConfig, logger *slog.Logger) (*DataIngestor, error) {
	return &DataIngestor{
		sources:              sources,
		embeddingsConfig:     embeddingsConfig,
		logger:               logger,
		embeddingDataSources: make(map[string]*embedding.EmbeddingDataSource),
	}, nil
}

// IngestAll ingests data from all configured sources
func (di *DataIngestor) IngestAll(ctx context.Context) ([]IngestedData, error) {
	var results []IngestedData

	for _, source := range di.sources {
		di.logger.Info("Ingesting data", "source", source.Name, "type", source.Type)

		data, err := di.ingestSource(ctx, source)
		if err != nil {
			di.logger.Error("Failed to ingest data", "source", source.Name, "error", err)
			continue
		}

		results = append(results, *data)
	}

	return results, nil
}

// ingestSource ingests data from a single source
func (di *DataIngestor) ingestSource(ctx context.Context, source DataSource) (*IngestedData, error) {
	switch source.Type {
	case "file":
		return di.ingestFile(ctx, source)
	case "directory":
		return di.ingestDirectory(ctx, source)
	case "api":
		return di.ingestAPI(ctx, source)
	case "web":
		return di.ingestWeb(ctx, source)
	case "stdin":
		return di.ingestStdin(ctx, source)
	case "embedding":
		return di.ingestEmbedding(ctx, source)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", source.Type)
	}
}

// ingestFile reads data from a file
func (di *DataIngestor) ingestFile(ctx context.Context, source DataSource) (*IngestedData, error) {
	path, ok := source.Config["path"].(string)
	if !ok {
		return nil, fmt.Errorf("file path not specified")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Apply preprocessing if configured
	processedContent, err := di.applyPreprocessing(content, source.Preprocessing)
	if err != nil {
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}

	return &IngestedData{
		Source: source.Name,
		Type:   source.Type,
		Data:   processedContent,
		Metadata: map[string]interface{}{
			"path": path,
			"size": len(content),
		},
	}, nil
}

// ingestDirectory reads data from a directory
func (di *DataIngestor) ingestDirectory(ctx context.Context, source DataSource) (*IngestedData, error) {
	path, ok := source.Config["path"].(string)
	if !ok {
		return nil, fmt.Errorf("directory path not specified")
	}

	recursive, _ := source.Config["recursive"].(bool)
	var files []string

	if recursive {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, filePath)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory %s: %w", path, err)
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				files = append(files, filepath.Join(path, entry.Name()))
			}
		}
	}

	var fileContents []map[string]interface{}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			di.logger.Warn("Failed to read file", "file", file, "error", err)
			continue
		}

		processedContent, err := di.applyPreprocessing(content, source.Preprocessing)
		if err != nil {
			di.logger.Warn("Preprocessing failed", "file", file, "error", err)
			continue
		}

		fileContents = append(fileContents, map[string]interface{}{
			"path":    file,
			"content": processedContent,
		})
	}

	return &IngestedData{
		Source: source.Name,
		Type:   source.Type,
		Data:   fileContents,
		Metadata: map[string]interface{}{
			"path":       path,
			"file_count": len(files),
			"recursive":  recursive,
		},
	}, nil
}

// ingestAPI reads data from an API endpoint
func (di *DataIngestor) ingestAPI(ctx context.Context, source DataSource) (*IngestedData, error) {
	url, ok := source.Config["url"].(string)
	if !ok {
		return nil, fmt.Errorf("API URL not specified")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers if configured
	if headers, ok := source.Config["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if headerValue, ok := v.(string); ok {
				req.Header.Set(k, headerValue)
			}
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	processedContent, err := di.applyPreprocessing(body, source.Preprocessing)
	if err != nil {
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}

	return &IngestedData{
		Source: source.Name,
		Type:   source.Type,
		Data:   processedContent,
		Metadata: map[string]interface{}{
			"url":            url,
			"status_code":    resp.StatusCode,
			"content_length": len(body),
		},
	}, nil
}

// ingestWeb scrapes data from a web page
func (di *DataIngestor) ingestWeb(ctx context.Context, source DataSource) (*IngestedData, error) {
	// This is a simplified web scraper - in practice, you'd use a proper library
	return di.ingestAPI(ctx, source) // Reuse API logic for now
}

// ingestStdin reads data from standard input
func (di *DataIngestor) ingestStdin(ctx context.Context, source DataSource) (*IngestedData, error) {
	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}

	processedContent, err := di.applyPreprocessing(content, source.Preprocessing)
	if err != nil {
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}

	return &IngestedData{
		Source: source.Name,
		Type:   source.Type,
		Data:   processedContent,
		Metadata: map[string]interface{}{
			"size": len(content),
		},
	}, nil
}

// applyPreprocessing applies preprocessing steps to data
func (di *DataIngestor) applyPreprocessing(data []byte, steps []ProcessingStep) (interface{}, error) {
	result := interface{}(string(data))

	for _, step := range steps {
		var err error
		switch step.Type {
		case "filter":
			result, err = di.applyFilter(result, step.Config)
		case "transform":
			result, err = di.applyTransform(result, step.Config)
		case "validate":
			result, err = di.applyValidation(result, step.Config)
		case "extract":
			result, err = di.applyExtraction(result, step.Config)
		default:
			return nil, fmt.Errorf("unsupported preprocessing step: %s", step.Type)
		}

		if err != nil {
			return nil, fmt.Errorf("preprocessing step %s failed: %w", step.Type, err)
		}
	}

	return result, nil
}

// applyFilter filters data based on criteria
func (di *DataIngestor) applyFilter(data interface{}, config map[string]interface{}) (interface{}, error) {
	// Simple filter implementation - could be much more sophisticated
	if dataStr, ok := data.(string); ok {
		if includes, ok := config["includes"].([]interface{}); ok {
			for _, include := range includes {
				if includeStr, ok := include.(string); ok {
					if !strings.Contains(dataStr, includeStr) {
						return "", nil // Filter out
					}
				}
			}
		}

		if excludes, ok := config["excludes"].([]interface{}); ok {
			for _, exclude := range excludes {
				if excludeStr, ok := exclude.(string); ok {
					if strings.Contains(dataStr, excludeStr) {
						return "", nil // Filter out
					}
				}
			}
		}
	}

	return data, nil
}

// applyTransform transforms data
func (di *DataIngestor) applyTransform(data interface{}, config map[string]interface{}) (interface{}, error) {
	// Simple transform implementation
	if dataStr, ok := data.(string); ok {
		if transformType, ok := config["type"].(string); ok {
			switch transformType {
			case "uppercase":
				return strings.ToUpper(dataStr), nil
			case "lowercase":
				return strings.ToLower(dataStr), nil
			case "trim":
				return strings.TrimSpace(dataStr), nil
			}
		}
	}

	return data, nil
}

// ingestEmbedding indexes content into vector storage
func (di *DataIngestor) ingestEmbedding(ctx context.Context, source DataSource) (*IngestedData, error) {
	// Extract embedding configuration from source config with centralized defaults
	var embeddingConfig embedding.EmbeddingDataSourceConfig

	// Use centralized embedding config as defaults
	if di.embeddingsConfig != nil {
		embeddingConfig.Provider = di.embeddingsConfig.Provider
		embeddingConfig.Model = di.embeddingsConfig.Model
		embeddingConfig.APIKey = di.embeddingsConfig.APIKey
		embeddingConfig.ChunkSize = di.embeddingsConfig.ChunkSize
	}

	// Map config to embedding config struct (override defaults if specified)
	if storageDir, ok := source.Config["storage_dir"].(string); ok {
		embeddingConfig.StorageDir = storageDir
	}
	if provider, ok := source.Config["provider"].(string); ok {
		embeddingConfig.Provider = provider
	}
	if model, ok := source.Config["model"].(string); ok {
		embeddingConfig.Model = model
	}
	if apiKey, ok := source.Config["api_key"].(string); ok {
		embeddingConfig.APIKey = apiKey
	}
	if sourcePathsI, ok := source.Config["source_paths"].([]interface{}); ok {
		sourcePaths := make([]string, len(sourcePathsI))
		for i, path := range sourcePathsI {
			if pathStr, ok := path.(string); ok {
				sourcePaths[i] = pathStr
			}
		}
		embeddingConfig.SourcePaths = sourcePaths
	}
	if filePatternsI, ok := source.Config["file_patterns"].([]interface{}); ok {
		filePatterns := make([]string, len(filePatternsI))
		for i, pattern := range filePatternsI {
			if patternStr, ok := pattern.(string); ok {
				filePatterns[i] = patternStr
			}
		}
		embeddingConfig.FilePatterns = filePatterns
	}
	if excludePatternsI, ok := source.Config["exclude_patterns"].([]interface{}); ok {
		excludePatterns := make([]string, len(excludePatternsI))
		for i, pattern := range excludePatternsI {
			if patternStr, ok := pattern.(string); ok {
				excludePatterns[i] = patternStr
			}
		}
		embeddingConfig.ExcludePatterns = excludePatterns
	}
	if chunkSize, ok := source.Config["chunk_size"].(float64); ok {
		embeddingConfig.ChunkSize = int(chunkSize)
	} else if embeddingConfig.ChunkSize == 0 {
		embeddingConfig.ChunkSize = 1000 // fallback default
	}
	if refreshInterval, ok := source.Config["refresh_interval"].(string); ok {
		embeddingConfig.RefreshInterval = refreshInterval
	}
	if metadata, ok := source.Config["metadata"].(map[string]interface{}); ok {
		embeddingConfig.Metadata = metadata
	}

	// Create embedding data source
	embeddingDataSource, err := embedding.NewEmbeddingDataSource(embeddingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding data source: %w", err)
	}

	// Store the embedding data source for later use by tools
	di.embeddingDataSources[source.Name] = embeddingDataSource

	// Ingest data into embeddings
	stats, err := embeddingDataSource.IngestData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ingest embedding data: %w", err)
	}

	// Get all embeddings for the result
	embeddings := embeddingDataSource.GetEmbeddings()

	return &IngestedData{
		Source:   source.Name,
		Type:     source.Type,
		Data:     embeddings,
		Metadata: stats,
	}, nil
}

// GetEmbeddingDataSources returns the embedding data sources created during ingestion
func (di *DataIngestor) GetEmbeddingDataSources() map[string]*embedding.EmbeddingDataSource {
	return di.embeddingDataSources
}

// applyValidation validates data
func (di *DataIngestor) applyValidation(data interface{}, config map[string]interface{}) (interface{}, error) {
	// Simple validation - could check format, schema, etc.
	if format, ok := config["format"].(string); ok {
		if dataStr, ok := data.(string); ok {
			switch format {
			case "json":
				var temp interface{}
				if err := json.Unmarshal([]byte(dataStr), &temp); err != nil {
					return nil, fmt.Errorf("invalid JSON: %w", err)
				}
				return temp, nil
			}
		}
	}

	return data, nil
}

// applyExtraction extracts specific parts of data
func (di *DataIngestor) applyExtraction(data interface{}, config map[string]interface{}) (interface{}, error) {
	// Simple extraction implementation
	if dataStr, ok := data.(string); ok {
		if pattern, ok := config["pattern"].(string); ok {
			// Could use regex or other extraction methods
			if strings.Contains(dataStr, pattern) {
				// Simple substring extraction
				start := strings.Index(dataStr, pattern)
				if start != -1 {
					return dataStr[start:], nil
				}
			}
		}
	}

	return data, nil
}
