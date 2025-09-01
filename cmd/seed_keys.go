package cmd

import (
	"fmt"

	"github.com/alantheprice/agent/pkg/providers/config"
	"github.com/spf13/cobra"
)

// seedKeysCmd represents the seed-keys command
var seedKeysCmd = &cobra.Command{
	Use:   "seed-keys",
	Short: "Seed API keys from ~/.ledit/api_keys.json",
	Long: `Seeds API keys from the legacy ~/.ledit/api_keys.json file into the new 
provider configuration system. This is typically run once during migration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Seeding API keys from ~/.ledit/api_keys.json...")
		
		if err := config.SeedAPIKeysFromLedit(); err != nil {
			return fmt.Errorf("failed to seed API keys: %w", err)
		}

		// Show current provider status
		fmt.Println("\nProvider status after seeding:")
		providers, err := config.ListProviders()
		if err != nil {
			return fmt.Errorf("failed to list providers: %w", err)
		}

		for name, hasKey := range providers {
			status := "❌ No API key"
			if hasKey {
				status = "✅ Ready"
			}
			fmt.Printf("  %s: %s\n", name, status)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(seedKeysCmd)
}