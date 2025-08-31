package embedding

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Embedding represents a vector embedding for any entity (document, code, text, etc.)
type Embedding struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "document", "code", "text", "chunk"
	Source      string                 `json:"source"`
	Content     string                 `json:"content,omitempty"`
	Vector      []float64              `json:"vector"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	TokenCount  int                    `json:"token_count"`
	LastUpdated time.Time              `json:"last_updated"`
}

// VectorDB represents an in-memory vector database with disk persistence
type VectorDB struct {
	embeddings map[string]*Embedding
	storageDir string
	mu         sync.RWMutex
}

// NewVectorDB creates a new vector database with the specified storage directory
func NewVectorDB(storageDir string) *VectorDB {
	if storageDir == "" {
		storageDir = ".agent/embeddings"
	}
	return &VectorDB{
		embeddings: make(map[string]*Embedding),
		storageDir: storageDir,
	}
}

// getEmbeddingFilePath returns the file path for a given embedding ID
func (vdb *VectorDB) getEmbeddingFilePath(id string) string {
	// Sanitize ID for use as filename
	sanitizedID := strings.ReplaceAll(id, "/", "_")
	sanitizedID = strings.ReplaceAll(sanitizedID, ":", "-")
	sanitizedID = strings.ReplaceAll(sanitizedID, " ", "_")
	return filepath.Join(vdb.storageDir, sanitizedID+".json")
}

// Add adds an embedding to the database and persists it to disk
func (vdb *VectorDB) Add(embedding *Embedding) error {
	vdb.mu.Lock()
	defer vdb.mu.Unlock()

	// Add to in-memory storage
	vdb.embeddings[embedding.ID] = embedding

	// Persist to disk
	return vdb.saveEmbedding(embedding)
}

// Get retrieves an embedding by ID
func (vdb *VectorDB) Get(id string) (*Embedding, bool) {
	vdb.mu.RLock()
	defer vdb.mu.RUnlock()

	emb, exists := vdb.embeddings[id]
	return emb, exists
}

// Remove removes an embedding from the database and deletes its file
func (vdb *VectorDB) Remove(id string) error {
	vdb.mu.Lock()
	defer vdb.mu.Unlock()

	// Remove from memory
	delete(vdb.embeddings, id)

	// Remove from disk
	filePath := vdb.getEmbeddingFilePath(id)
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove embedding file %s: %w", filePath, err)
	}

	return nil
}

// LoadAll loads all embeddings from disk into memory
func (vdb *VectorDB) LoadAll() error {
	vdb.mu.Lock()
	defer vdb.mu.Unlock()

	// Clear existing embeddings
	vdb.embeddings = make(map[string]*Embedding)

	// Check if storage directory exists
	if _, err := os.Stat(vdb.storageDir); os.IsNotExist(err) {
		return nil // No embeddings to load
	}

	// Read all embedding files
	files, err := os.ReadDir(vdb.storageDir)
	if err != nil {
		return fmt.Errorf("failed to read embeddings directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(vdb.storageDir, file.Name())
			embedding, err := vdb.loadEmbedding(filePath)
			if err != nil {
				// Log warning but continue with other embeddings
				fmt.Printf("Warning: failed to load embedding %s: %v\n", filePath, err)
				continue
			}
			vdb.embeddings[embedding.ID] = embedding
		}
	}

	return nil
}

// GetAll returns all embeddings currently in memory
func (vdb *VectorDB) GetAll() []*Embedding {
	vdb.mu.RLock()
	defer vdb.mu.RUnlock()

	embeddings := make([]*Embedding, 0, len(vdb.embeddings))
	for _, emb := range vdb.embeddings {
		embeddings = append(embeddings, emb)
	}

	return embeddings
}

// Search finds the top K most similar embeddings to the query vector
func (vdb *VectorDB) Search(queryVector []float64, topK int, minSimilarity float64) ([]*Embedding, []float64, error) {
	vdb.mu.RLock()
	defer vdb.mu.RUnlock()

	if len(vdb.embeddings) == 0 {
		return nil, nil, nil
	}

	type result struct {
		embedding *Embedding
		score     float64
	}

	var results []result

	for _, emb := range vdb.embeddings {
		score, err := CosineSimilarity(queryVector, emb.Vector)
		if err != nil {
			// Skip embeddings that can't be compared
			continue
		}

		// Only include results above minimum similarity threshold
		if score >= minSimilarity {
			results = append(results, result{embedding: emb, score: score})
		}
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Take top K
	if topK > len(results) {
		topK = len(results)
	}

	if topK <= 0 {
		return nil, nil, nil
	}

	topResults := results[:topK]
	embeddings := make([]*Embedding, len(topResults))
	scores := make([]float64, len(topResults))

	for i, res := range topResults {
		embeddings[i] = res.embedding
		scores[i] = res.score
	}

	return embeddings, scores, nil
}

// Count returns the number of embeddings in the database
func (vdb *VectorDB) Count() int {
	vdb.mu.RLock()
	defer vdb.mu.RUnlock()
	return len(vdb.embeddings)
}

// Clear removes all embeddings from memory and disk
func (vdb *VectorDB) Clear() error {
	vdb.mu.Lock()
	defer vdb.mu.Unlock()

	// Clear memory
	vdb.embeddings = make(map[string]*Embedding)

	// Remove storage directory
	return os.RemoveAll(vdb.storageDir)
}

// saveEmbedding persists an embedding to disk
func (vdb *VectorDB) saveEmbedding(emb *Embedding) error {
	filePath := vdb.getEmbeddingFilePath(emb.ID)

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create embeddings directory: %w", err)
	}

	// Marshal and write to file
	data, err := json.MarshalIndent(emb, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write embedding file: %w", err)
	}

	return nil
}

// loadEmbedding loads an embedding from disk
func (vdb *VectorDB) loadEmbedding(filePath string) (*Embedding, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedding file: %w", err)
	}

	var emb Embedding
	if err := json.Unmarshal(data, &emb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
	}

	return &emb, nil
}
