package cmd

import (
	"fmt"

	"github.com/alantheprice/agent/pkg/interfaces/types"
	"github.com/alantheprice/agent/pkg/providers/config"
	"github.com/alantheprice/agent/pkg/providers/llm"
	"github.com/spf13/cobra"
)

// testProviderCmd represents the test-provider command
var testProviderCmd = &cobra.Command{
	Use:   "test-provider [provider-name]",
	Short: "Test a specific provider configuration",
	Long:  `Test a specific provider by creating a configuration and validating it.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerName := args[0]

		fmt.Printf("Testing provider: %s\n", providerName)
		fmt.Println("========================")

		// Get API key
		apiKey := config.GetAPIKeyForProvider(providerName)
		if apiKey == "" {
			return fmt.Errorf("no API key found for provider: %s", providerName)
		}

		// Load provider definition
		providersConfig, err := config.LoadProvidersConfig()
		if err != nil {
			return fmt.Errorf("failed to load provider configuration: %w", err)
		}

		providerDef, exists := providersConfig.Providers[providerName]
		if !exists {
			return fmt.Errorf("provider '%s' not found in configuration", providerName)
		}

		// Create provider configuration
		providerConfig := &types.ProviderConfig{
			Name:        providerName,
			BaseURL:     providerDef.BaseURL,
			APIKey:      apiKey,
			Model:       providerDef.DefaultModel,
			Enabled:     true,
			Temperature: 0.7,
			MaxTokens:   providerDef.Capabilities.MaxTokens,
		}

		fmt.Printf("Configuration:\n")
		fmt.Printf("  Name: %s\n", providerConfig.Name)
		fmt.Printf("  Base URL: %s\n", providerConfig.BaseURL)
		fmt.Printf("  Model: %s\n", providerConfig.Model)
		fmt.Printf("  Max Tokens: %d\n", providerConfig.MaxTokens)
		fmt.Printf("  API Key: %s...\n", apiKey[:min(10, len(apiKey))])

		// Test factory creation
		factory := llm.NewGlobalFactory()

		fmt.Printf("\nValidating configuration...\n")
		if err := factory.ValidateProviderConfig(providerConfig); err != nil {
			fmt.Printf("❌ Validation failed: %v\n", err)
			return err
		}

		fmt.Printf("✅ Configuration is valid\n")

		// Try to create provider instance
		fmt.Printf("\nCreating provider instance...\n")
		provider, err := factory.CreateProvider(providerConfig)
		if err != nil {
			fmt.Printf("❌ Provider creation failed: %v\n", err)
			return err
		}

		fmt.Printf("✅ Provider instance created successfully\n")
		fmt.Printf("Provider type: %T\n", provider)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(testProviderCmd)
}