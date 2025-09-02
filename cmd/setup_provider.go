package cmd

import (
	"fmt"

	"github.com/alantheprice/agent/pkg/providers/config"
	"github.com/spf13/cobra"
)

// setupProviderCmd represents the setup-provider command
var setupProviderCmd = &cobra.Command{
	Use:   "setup-provider [provider-name]",
	Short: "Interactively setup credentials for a provider",
	Long: `Interactively setup API credentials for a specific provider. This command
will prompt you to enter the API key if it's not found in environment variables
or the credentials file.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerName := args[0]

		fmt.Printf("Setting up credentials for provider: %s\n", providerName)
		fmt.Println("=" + fmt.Sprintf("%*s", len(providerName)+36, "="))

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

		// Check current status
		currentKey := config.GetAPIKeyForProvider(providerName)
		if currentKey != "" {
			fmt.Printf("Current Status: ✅ API key already configured\n")
			fmt.Printf("Key preview: %s...\n", currentKey[:min(10, len(currentKey))])
			
			fmt.Print("\nDo you want to update the existing API key? (y/N): ")
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Setup cancelled.")
				return nil
			}
		} else {
			fmt.Printf("Current Status: ❌ No API key configured\n")
		}

		// Use the interactive credential setup
		apiKey := config.GetAPIKeyForProviderWithPrompt(providerName, true)
		
		if apiKey != "" {
			fmt.Printf("\n✅ Successfully configured credentials for %s\n", provider.Name)
			fmt.Printf("You can now use this provider in your agent configurations.\n")
			
			// Test the provider
			fmt.Printf("\nTo test this provider, run: ./agent test-provider %s\n", providerName)
		} else {
			fmt.Printf("\n❌ Failed to configure credentials for %s\n", provider.Name)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupProviderCmd)
}