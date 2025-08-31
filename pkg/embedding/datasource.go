package embedding

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EmbeddingDataSource implements a data source that indexes content into vector storage
type EmbeddingDataSource struct {
	generator *EmbeddingGenerator
	vectorDB  *VectorDB
	config    EmbeddingDataSourceConfig
}

// EmbeddingDataSourceConfig contains configuration for embedding data sources
type EmbeddingDataSourceConfig struct {
	StorageDir      string                 `json:"storage_dir,omitempty"`
	Provider        string                 `json:"provider,omitempty"`         // embedding provider (openai, deepinfra)
	Model           string                 `json:"model,omitempty"`            // embedding model
	APIKey          string                 `json:"api_key,omitempty"`          // API key for provider
	SourcePaths     []string               `json:"source_paths,omitempty"`     // paths to index
	FilePatterns    []string               `json:"file_patterns,omitempty"`    // file patterns to include
	ExcludePatterns []string               `json:"exclude_patterns,omitempty"` // patterns to exclude
	ChunkSize       int                    `json:"chunk_size,omitempty"`       // text chunk size for large files
	RefreshInterval string                 `json:"refresh_interval,omitempty"` // how often to refresh embeddings
	Metadata        map[string]interface{} `json:"metadata,omitempty"`         // additional metadata
}

// NewEmbeddingDataSource creates a new embedding data source
func NewEmbeddingDataSource(config EmbeddingDataSourceConfig) (*EmbeddingDataSource, error) {
	// Set defaults
	if config.StorageDir == "" {
		config.StorageDir = ".agent/embeddings"
	}
	if config.Provider == "" {
		config.Provider = "openai"
	}
	if config.ChunkSize == 0 {
		config.ChunkSize = 1000 // Default chunk size
	}

	// Create embedding generator
	generator := NewEmbeddingGenerator()

	// Register providers based on available API keys
	if config.APIKey != "" {
		switch config.Provider {
		case "openai":
			generator.RegisterProvider("openai", NewOpenAIProvider(config.APIKey))
		case "deepinfra":
			generator.RegisterProvider("deepinfra", NewDeepInfraProvider(config.APIKey))
		default:
			return nil, fmt.Errorf("unsupported embedding provider: %s", config.Provider)
		}
	}

	// Create vector database
	vectorDB := NewVectorDB(config.StorageDir)

	return &EmbeddingDataSource{
		generator: generator,
		vectorDB:  vectorDB,
		config:    config,
	}, nil
}

// IngestData indexes content from configured sources into the vector database
func (eds *EmbeddingDataSource) IngestData(ctx context.Context) (map[string]interface{}, error) {
	// Load existing embeddings
	if err := eds.vectorDB.LoadAll(); err != nil {
		return nil, fmt.Errorf("failed to load existing embeddings: %w", err)
	}

	stats := map[string]interface{}{
		"files_processed":  0,
		"files_skipped":    0,
		"files_errored":    0,
		"total_embeddings": eds.vectorDB.Count(),
	}

	// Process each source path
	for _, sourcePath := range eds.config.SourcePaths {
		if err := eds.processPath(ctx, sourcePath, stats); err != nil {
			return stats, fmt.Errorf("failed to process path %s: %w", sourcePath, err)
		}
	}

	stats["final_embeddings"] = eds.vectorDB.Count()
	return stats, nil
}

// SearchContent searches for content similar to the query
func (eds *EmbeddingDataSource) SearchContent(ctx context.Context, query string, limit int, minSimilarity float64) ([]*Embedding, []float64, error) {
	// Generate embedding for query
	queryVector, err := eds.generator.GenerateEmbedding(query, eds.config.Provider, eds.config.Model)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search vector database
	return eds.vectorDB.Search(queryVector, limit, minSimilarity)
}

// processPath processes a single path (file or directory)
func (eds *EmbeddingDataSource) processPath(ctx context.Context, sourcePath string, stats map[string]interface{}) error {
	fileInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat path %s: %w", sourcePath, err)
	}

	if fileInfo.IsDir() {
		return eds.processDirectory(ctx, sourcePath, stats)
	}

	return eds.processFile(ctx, sourcePath, stats)
}

// processDirectory processes all files in a directory
func (eds *EmbeddingDataSource) processDirectory(ctx context.Context, dirPath string, stats map[string]interface{}) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check context for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if file should be processed
		if eds.shouldProcessFile(path) {
			return eds.processFile(ctx, path, stats)
		}

		stats["files_skipped"] = stats["files_skipped"].(int) + 1
		return nil
	})
}

// processFile processes a single file
func (eds *EmbeddingDataSource) processFile(ctx context.Context, filePath string, stats map[string]interface{}) error {
	// Check if embedding already exists and is up-to-date
	embeddingID := fmt.Sprintf("file:%s", filePath)
	if existing, exists := eds.vectorDB.Get(embeddingID); exists {
		fileInfo, err := os.Stat(filePath)
		if err == nil && !fileInfo.ModTime().After(existing.LastUpdated) {
			// File hasn't changed, skip
			return nil
		}
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		stats["files_errored"] = stats["files_errored"].(int) + 1
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	contentStr := string(content)

	// Create metadata
	metadata := make(map[string]interface{})
	for k, v := range eds.config.Metadata {
		metadata[k] = v
	}
	metadata["file_path"] = filePath
	metadata["file_size"] = len(content)
	metadata["file_extension"] = filepath.Ext(filePath)

	// If file is large, chunk it
	if len(contentStr) > eds.config.ChunkSize {
		return eds.processFileInChunks(ctx, filePath, contentStr, metadata, stats)
	}

	// Process entire file as single embedding
	embedding, err := eds.generator.CreateEmbedding(
		embeddingID,
		"file",
		filePath,
		contentStr,
		metadata,
		eds.config.Provider,
		eds.config.Model,
	)
	if err != nil {
		stats["files_errored"] = stats["files_errored"].(int) + 1
		return fmt.Errorf("failed to create embedding for file %s: %w", filePath, err)
	}

	if err := eds.vectorDB.Add(embedding); err != nil {
		stats["files_errored"] = stats["files_errored"].(int) + 1
		return fmt.Errorf("failed to add embedding to database: %w", err)
	}

	stats["files_processed"] = stats["files_processed"].(int) + 1
	return nil
}

// processFileInChunks processes large files by splitting them into chunks
func (eds *EmbeddingDataSource) processFileInChunks(ctx context.Context, filePath, content string, metadata map[string]interface{}, stats map[string]interface{}) error {
	chunks := eds.chunkText(content, eds.config.ChunkSize)

	for i, chunk := range chunks {
		// Check context for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		chunkID := fmt.Sprintf("file:%s:chunk:%d", filePath, i)

		// Add chunk-specific metadata
		chunkMetadata := make(map[string]interface{})
		for k, v := range metadata {
			chunkMetadata[k] = v
		}
		chunkMetadata["chunk_index"] = i
		chunkMetadata["total_chunks"] = len(chunks)

		embedding, err := eds.generator.CreateEmbedding(
			chunkID,
			"chunk",
			filePath,
			chunk,
			chunkMetadata,
			eds.config.Provider,
			eds.config.Model,
		)
		if err != nil {
			stats["files_errored"] = stats["files_errored"].(int) + 1
			return fmt.Errorf("failed to create embedding for chunk %d of file %s: %w", i, filePath, err)
		}

		if err := eds.vectorDB.Add(embedding); err != nil {
			stats["files_errored"] = stats["files_errored"].(int) + 1
			return fmt.Errorf("failed to add chunk embedding to database: %w", err)
		}
	}

	stats["files_processed"] = stats["files_processed"].(int) + 1
	return nil
}

// chunkText splits text into chunks of approximately the specified size
func (eds *EmbeddingDataSource) chunkText(text string, chunkSize int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	words := strings.Fields(text)
	currentChunk := []string{}
	currentSize := 0

	for _, word := range words {
		wordSize := len(word) + 1 // +1 for space

		if currentSize+wordSize > chunkSize && len(currentChunk) > 0 {
			// Current chunk is full, start new chunk
			chunks = append(chunks, strings.Join(currentChunk, " "))
			currentChunk = []string{word}
			currentSize = len(word)
		} else {
			currentChunk = append(currentChunk, word)
			currentSize += wordSize
		}
	}

	// Add remaining chunk
	if len(currentChunk) > 0 {
		chunks = append(chunks, strings.Join(currentChunk, " "))
	}

	return chunks
}

// shouldProcessFile determines if a file should be processed based on patterns
func (eds *EmbeddingDataSource) shouldProcessFile(filePath string) bool {
	fileName := filepath.Base(filePath)
	fileExt := filepath.Ext(filePath)

	// Check exclude patterns first
	for _, pattern := range eds.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return false
		}
		if matched, _ := filepath.Match(pattern, fileExt); matched {
			return false
		}
	}

	// If no include patterns specified, include all (except excluded)
	if len(eds.config.FilePatterns) == 0 {
		return true
	}

	// Check include patterns
	for _, pattern := range eds.config.FilePatterns {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, fileExt); matched {
			return true
		}
	}

	return false
}

// GetEmbeddings returns all embeddings in the database
func (eds *EmbeddingDataSource) GetEmbeddings() []*Embedding {
	return eds.vectorDB.GetAll()
}

// RefreshEmbeddings re-indexes all content if refresh interval has passed
func (eds *EmbeddingDataSource) RefreshEmbeddings(ctx context.Context) error {
	if eds.config.RefreshInterval == "" {
		return nil // No refresh configured
	}

	refreshInterval, err := time.ParseDuration(eds.config.RefreshInterval)
	if err != nil {
		return fmt.Errorf("invalid refresh interval: %w", err)
	}

	// Check if refresh is needed based on last update times
	embeddings := eds.vectorDB.GetAll()
	if len(embeddings) > 0 {
		var lastUpdate time.Time
		for _, emb := range embeddings {
			if emb.LastUpdated.After(lastUpdate) {
				lastUpdate = emb.LastUpdated
			}
		}

		if time.Since(lastUpdate) < refreshInterval {
			return nil // Not time to refresh yet
		}
	}

	// Perform refresh
	_, err = eds.IngestData(ctx)
	return err
}
