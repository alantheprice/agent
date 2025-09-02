package cmd

import (
	"fmt"
	"strings"

	"github.com/alantheprice/agent/pkg/providers/config"
	"github.com/spf13/cobra"
)

// listModelsCmd represents the list-models command
var listModelsCmd = &cobra.Command{
	Use:   "list-models [provider-name]",
	Short: "List available models for a provider",
	Long: `Lists all available models for a specific provider, showing both LLM models 
and embedding models (if supported). If no provider is specified, lists models 
for the default provider.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine provider name
		providerName := ""
		if len(args) > 0 {
			providerName = args[0]
		} else {
			// Use default provider
			providersConfig, err := config.LoadProvidersConfig()
			if err != nil {
				return fmt.Errorf("failed to load provider configuration: %w", err)
			}
			providerName = providersConfig.DefaultProvider
		}

		fmt.Printf("Available Models for %s\n", strings.Title(providerName))
		fmt.Println(strings.Repeat("=", len(providerName)+22))

		// Load provider configuration
		providersConfig, err := config.LoadProvidersConfig()
		if err != nil {
			return fmt.Errorf("failed to load provider configuration: %w", err)
		}

		provider, exists := providersConfig.Providers[providerName]
		if !exists {
			return fmt.Errorf("provider '%s' not found in configuration", providerName)
		}

		// Check if API key is available
		apiKey := config.GetAPIKeyForProvider(providerName)
		hasKey := apiKey != ""

		fmt.Printf("Provider: %s\n", provider.Name)
		fmt.Printf("Base URL: %s\n", provider.BaseURL)
		fmt.Printf("API Key: ")
		if hasKey {
			fmt.Printf("✅ Available\n")
		} else {
			fmt.Printf("❌ Missing\n")
		}
		fmt.Printf("Status: ")
		if provider.Enabled && hasKey {
			fmt.Printf("✅ Ready\n")
		} else {
			fmt.Printf("❌ Not Ready\n")
		}

		// List LLM models
		if len(provider.SupportedModels) > 0 {
			fmt.Printf("\n🤖 LLM Models (%d available):\n", len(provider.SupportedModels))
			for i, model := range provider.SupportedModels {
				marker := "  "
				if model == provider.DefaultModel {
					marker = "* " // Mark default model with asterisk
					fmt.Printf("%s%s (default)\n", marker, model)
				} else {
					fmt.Printf("%s%s\n", marker, model)
				}
				
				// Add some spacing every 5 models for readability
				if (i+1)%5 == 0 && i+1 < len(provider.SupportedModels) {
					fmt.Println()
				}
			}
		} else {
			fmt.Printf("\n🤖 LLM Models: None configured\n")
		}

		// List embedding models if supported
		if provider.Capabilities.SupportsEmbeddings && len(provider.SupportedEmbeddingModels) > 0 {
			fmt.Printf("\n🔍 Embedding Models (%d available):\n", len(provider.SupportedEmbeddingModels))
			for i, model := range provider.SupportedEmbeddingModels {
				marker := "  "
				if model == provider.DefaultEmbeddingModel {
					marker = "* " // Mark default model with asterisk
					fmt.Printf("%s%s (default)\n", marker, model)
				} else {
					fmt.Printf("%s%s\n", marker, model)
				}
				
				// Add some spacing every 5 models for readability
				if (i+1)%5 == 0 && i+1 < len(provider.SupportedEmbeddingModels) {
					fmt.Println()
				}
			}
		} else if provider.Capabilities.SupportsEmbeddings {
			fmt.Printf("\n🔍 Embedding Models: None configured\n")
		}

		// Show capabilities
		fmt.Printf("\n💡 Capabilities:\n")
		fmt.Printf("  Tools Support: %v\n", provider.Capabilities.SupportsTools)
		fmt.Printf("  Image Support: %v\n", provider.Capabilities.SupportsImages)
		fmt.Printf("  Streaming: %v\n", provider.Capabilities.SupportsStream)
		fmt.Printf("  Embeddings: %v\n", provider.Capabilities.SupportsEmbeddings)
		fmt.Printf("  Max Tokens: %d\n", provider.Capabilities.MaxTokens)

		// Usage examples
		fmt.Printf("\n📋 Usage Examples:\n")
		fmt.Printf("  Test this provider: ./agent test-provider %s\n", providerName)
		if provider.Capabilities.SupportsEmbeddings {
			fmt.Printf("  Test embeddings:    ./agent test-embedding %s\n", providerName)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listModelsCmd)
}