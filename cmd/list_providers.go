package cmd

import (
	"fmt"

	"github.com/alantheprice/agent/pkg/providers/config"
	"github.com/alantheprice/agent/pkg/providers/llm"
	"github.com/spf13/cobra"
)

// listProvidersCmd represents the list-providers command
var listProvidersCmd = &cobra.Command{
	Use:   "list-providers",
	Short: "List all available LLM providers",
	Long:  `Lists all available LLM providers with their configuration and status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Available LLM Providers:")
		fmt.Println("=======================")

		// Load provider configurations
		providersConfig, err := config.LoadProvidersConfig()
		if err != nil {
			return fmt.Errorf("failed to load provider configuration: %w", err)
		}

		factory := llm.NewGlobalFactory()

		// Show providers in priority order
		for _, providerName := range providersConfig.PriorityOrder {
			provider, exists := providersConfig.Providers[providerName]
			if !exists {
				continue
			}

			// Check API key availability
			apiKey := config.GetAPIKeyForProvider(providerName)
			hasKey := apiKey != ""

			// Get capabilities
			caps, _ := factory.GetProviderCapabilities(providerName)

			// Status indicators
			enabledStatus := "❌"
			if provider.Enabled {
				enabledStatus = "✅"
			}

			keyStatus := "❌"
			if hasKey {
				keyStatus = "✅"
			}

			fmt.Printf("\n%s (%s)\n", provider.Name, providerName)
			fmt.Printf("  Enabled: %s\n", enabledStatus)
			fmt.Printf("  API Key: %s\n", keyStatus)
			fmt.Printf("  Base URL: %s\n", provider.BaseURL)
			fmt.Printf("  Default Model: %s\n", provider.DefaultModel)
			
			if caps != nil {
				fmt.Printf("  Capabilities:\n")
				fmt.Printf("    Tools: %v\n", caps.SupportsTools)
				fmt.Printf("    Images: %v\n", caps.SupportsImages)
				fmt.Printf("    Streaming: %v\n", caps.SupportsStream)
				fmt.Printf("    Max Tokens: %d\n", caps.MaxTokens)
				fmt.Printf("    Models: %v\n", provider.SupportedModels[:min(3, len(provider.SupportedModels))])
				if len(provider.SupportedModels) > 3 {
					fmt.Printf("      ... and %d more\n", len(provider.SupportedModels)-3)
				}
			}
		}

		// Show auto-detection result
		fmt.Println("\n" + "Auto-Detection Result:")
		fmt.Println("=====================")
		
		if bestConfig, err := factory.AutoDetectProvider(nil); err == nil {
			fmt.Printf("Best available provider: %s (%s)\n", bestConfig.Name, bestConfig.Model)
		} else {
			fmt.Printf("No providers available: %v\n", err)
		}

		return nil
	},
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func init() {
	rootCmd.AddCommand(listProvidersCmd)
}