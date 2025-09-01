package cmd

import (
	"fmt"

	"github.com/alantheprice/agent/pkg/providers/config"
	"github.com/spf13/cobra"
)

// testEmbeddingCmd represents the test-embedding command
var testEmbeddingCmd = &cobra.Command{
	Use:   "test-embedding [provider-name]",
	Short: "Test embedding model configuration for a provider",
	Long:  `Test the embedding model configuration for a specific provider.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerName := "deepinfra" // default
		if len(args) > 0 {
			providerName = args[0]
		}

		fmt.Printf("Testing embedding configuration for: %s\n", providerName)
		fmt.Println("==========================================")

		// Load provider configuration
		providersConfig, err := config.LoadProvidersConfig()
		if err != nil {
			return fmt.Errorf("failed to load provider configuration: %w", err)
		}

		provider, exists := providersConfig.Providers[providerName]
		if !exists {
			return fmt.Errorf("provider '%s' not found in configuration", providerName)
		}

		fmt.Printf("Provider: %s\n", provider.Name)
		fmt.Printf("Base URL: %s\n", provider.BaseURL)
		fmt.Printf("Default LLM Model: %s\n", provider.DefaultModel)
		
		if provider.DefaultEmbeddingModel != "" {
			fmt.Printf("Default Embedding Model: %s\n", provider.DefaultEmbeddingModel)
		} else {
			fmt.Printf("Default Embedding Model: Not configured\n")
		}

		fmt.Printf("Supports Embeddings: %v\n", provider.Capabilities.SupportsEmbeddings)

		if len(provider.SupportedEmbeddingModels) > 0 {
			fmt.Printf("Supported Embedding Models:\n")
			for i, model := range provider.SupportedEmbeddingModels {
				marker := "  "
				if model == provider.DefaultEmbeddingModel {
					marker = "* " // Mark default model
				}
				fmt.Printf("%s%s\n", marker, model)
				if i >= 4 { // Limit display
					remaining := len(provider.SupportedEmbeddingModels) - 5
					if remaining > 0 {
						fmt.Printf("  ... and %d more\n", remaining)
					}
					break
				}
			}
		}

		// Test the helper function
		embeddingModel := config.GetEmbeddingModelForProvider(providerName)
		if embeddingModel != "" {
			fmt.Printf("\nHelper function result: %s\n", embeddingModel)
		} else {
			fmt.Printf("\nHelper function result: No embedding model configured\n")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(testEmbeddingCmd)
}