package embedding

import (
	"os"
	"testing"
)

func TestAPIKeyResolution(t *testing.T) {
	// Test environment variable resolution
	originalKey := os.Getenv("DEEPINFRA_API_KEY")

	// Set a test API key
	testKey := "test-api-key-123"
	os.Setenv("DEEPINFRA_API_KEY", testKey)
	defer func() {
		if originalKey != "" {
			os.Setenv("DEEPINFRA_API_KEY", originalKey)
		} else {
			os.Unsetenv("DEEPINFRA_API_KEY")
		}
	}()

	config := EmbeddingDataSourceConfig{
		Provider:     "deepinfra",
		Model:        "sentence-transformers/all-MiniLM-L6-v2",
		APIKey:       "", // Empty - should get from env
		SourcePaths:  []string{"./"},
		FilePatterns: []string{"*.md"},
		ChunkSize:    1000,
	}

	_, err := NewEmbeddingDataSource(config)
	if err != nil {
		t.Fatalf("Expected no error creating embedding data source, got: %v", err)
	}

	t.Logf("Successfully created embedding data source with API key from environment")
}

func TestAPIKeyResolutionMissing(t *testing.T) {
	// Test missing API key
	originalKey := os.Getenv("DEEPINFRA_API_KEY")
	os.Unsetenv("DEEPINFRA_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("DEEPINFRA_API_KEY", originalKey)
		}
	}()

	config := EmbeddingDataSourceConfig{
		Provider:     "deepinfra",
		Model:        "sentence-transformers/all-MiniLM-L6-v2",
		APIKey:       "", // Empty - should get from env
		SourcePaths:  []string{"./"},
		FilePatterns: []string{"*.md"},
		ChunkSize:    1000,
	}

	_, err := NewEmbeddingDataSource(config)
	if err == nil {
		t.Fatal("Expected error when API key is missing")
	}

	expectedError := "no API key found for provider deepinfra"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		(len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
