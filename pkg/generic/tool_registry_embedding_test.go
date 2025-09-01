package generic

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

func TestEmbeddingTools(t *testing.T) {
	// Create a basic tool registry for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create empty configs - we're just testing the tool existence and basic functionality
	toolConfigs := make(map[string]Tool)
	toolConfigs["embedding_ingest"] = Tool{Enabled: true}
	toolConfigs["embedding_search"] = Tool{Enabled: true}

	security := &Security{}

	registry, err := NewToolRegistry(toolConfigs, security, logger)
	if err != nil {
		t.Fatalf("Failed to create tool registry: %v", err)
	}

	t.Run("EmbeddingIngestTool", func(t *testing.T) {
		tool, exists := registry.GetTool("embedding_ingest")
		if !exists {
			t.Fatal("embedding_ingest tool not found")
		}

		params := map[string]interface{}{
			"source_name": "test_source",
		}

		result, err := tool.Execute(context.Background(), params)
		if err != nil {
			t.Fatalf("Tool execution failed: %v", err)
		}

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("Result is not a map")
		}

		if resultMap["source_name"] != "test_source" {
			t.Errorf("Expected source_name 'test_source', got %v", resultMap["source_name"])
		}

		if resultMap["status"] != "completed" {
			t.Errorf("Expected status 'completed', got %v", resultMap["status"])
		}
	})

	t.Run("EmbeddingSearchTool", func(t *testing.T) {
		tool, exists := registry.GetTool("embedding_search")
		if !exists {
			t.Fatal("embedding_search tool not found")
		}

		params := map[string]interface{}{
			"query":          "test query",
			"source_name":    "test_source",
			"limit":          float64(3),
			"min_similarity": 0.5,
		}

		result, err := tool.Execute(context.Background(), params)
		if err != nil {
			t.Fatalf("Tool execution failed: %v", err)
		}

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("Result is not a map")
		}

		if resultMap["query"] != "test query" {
			t.Errorf("Expected query 'test query', got %v", resultMap["query"])
		}

		if resultMap["limit"] != 3 {
			t.Errorf("Expected limit 3, got %v", resultMap["limit"])
		}

		results, ok := resultMap["results"].([]map[string]interface{})
		if !ok {
			t.Fatal("Results is not a slice of maps")
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// Check structure of first result
		if len(results) > 0 {
			firstResult := results[0]
			if _, ok := firstResult["file_path"]; !ok {
				t.Error("First result missing file_path")
			}
			if _, ok := firstResult["similarity"]; !ok {
				t.Error("First result missing similarity")
			}
			if _, ok := firstResult["content_preview"]; !ok {
				t.Error("First result missing content_preview")
			}
		}
	})

	t.Run("EmbeddingSearchMissingQuery", func(t *testing.T) {
		tool, exists := registry.GetTool("embedding_search")
		if !exists {
			t.Fatal("embedding_search tool not found")
		}

		params := map[string]interface{}{
			"source_name": "test_source",
			// Missing required "query" parameter
		}

		_, err := tool.Execute(context.Background(), params)
		if err == nil {
			t.Fatal("Expected error for missing query parameter")
		}

		if err.Error() != "query parameter is required and must be a string" {
			t.Errorf("Unexpected error message: %v", err.Error())
		}
	})
}
