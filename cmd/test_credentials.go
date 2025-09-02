package cmd

import (
	"fmt"
	"os"

	"github.com/alantheprice/agent/pkg/providers/config"
	"github.com/spf13/cobra"
)

// testCredentialsCmd represents the test-credentials command
var testCredentialsCmd = &cobra.Command{
	Use:   "test-credentials",
	Short: "Test the credential management system",
	Long: `Tests the credential management system by checking various scenarios:
- Environment variable detection
- Credentials file reading
- Missing credential handling`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🧪 Testing Credential Management System")
		fmt.Println("=====================================")

		// Test cases
		testCases := []struct {
			provider    string
			description string
		}{
			{"deepinfra", "Provider with credentials file entry"},
			{"groq", "Provider with empty credentials file entry"},
			{"ollama", "Provider with empty credentials + no env var"},
		}

		for i, tc := range testCases {
			fmt.Printf("\n%d. %s (%s)\n", i+1, tc.description, tc.provider)
			fmt.Println(fmt.Sprintf("%s", fmt.Sprintf("%*s", len(tc.description)+len(tc.provider)+5, "-")))
			
			// Check environment variable first
			providersConfig, err := config.LoadProvidersConfig()
			if err != nil {
				fmt.Printf("❌ Error loading config: %v\n", err)
				continue
			}

			provider, exists := providersConfig.Providers[tc.provider]
			if !exists {
				fmt.Printf("❌ Provider not found in configuration\n")
				continue
			}

			fmt.Printf("Environment variable: %s\n", provider.APIKeyEnv)
			
			if provider.APIKeyEnv != "" {
				envValue := os.Getenv(provider.APIKeyEnv)
				if envValue != "" {
					fmt.Printf("✅ Found in environment: %s...\n", envValue[:min(10, len(envValue))])
				} else {
					fmt.Printf("❌ Not found in environment\n")
				}
			}

			// Check credentials file
			apiKey := config.GetAPIKeyForProvider(tc.provider)
			if apiKey != "" {
				fmt.Printf("✅ Found in credentials: %s...\n", apiKey[:min(10, len(apiKey))])
			} else {
				fmt.Printf("❌ Not found in credentials file\n")
			}

			// Test the combined result
			finalKey := config.GetAPIKeyForProvider(tc.provider)
			if finalKey != "" {
				fmt.Printf("🎯 Final result: ✅ Available\n")
			} else {
				fmt.Printf("🎯 Final result: ❌ Missing (would prompt user if interactive)\n")
			}
		}

		// Test credentials file status
		fmt.Printf("\n📁 Credentials File Status\n")
		fmt.Println("==========================")
		
		credentialsPath, err := getCredentialsPath()
		if err != nil {
			fmt.Printf("❌ Error getting credentials path: %v\n", err)
		} else {
			fmt.Printf("Path: %s\n", credentialsPath)
			
			if _, err := os.Stat(credentialsPath); err == nil {
				fmt.Printf("Status: ✅ Exists\n")
				
				// Show file permissions
				info, err := os.Stat(credentialsPath)
				if err == nil {
					fmt.Printf("Permissions: %s\n", info.Mode().String())
				}
			} else {
				fmt.Printf("Status: ❌ Missing (would be created on first use)\n")
			}
		}

		fmt.Printf("\n💡 Usage Tips:\n")
		fmt.Printf("   • Set environment variables for temporary use: export DEEPINFRA_API_KEY=your-key\n")
		fmt.Printf("   • Use 'setup-provider <name>' for interactive credential setup\n")
		fmt.Printf("   • Use 'seed-keys' to import from ~/.ledit/api_keys.json\n")
		fmt.Printf("   • Credentials are stored securely in ~/.agents/credentials.json\n")

		return nil
	},
}

// getCredentialsPath duplicates the function for testing (normally we'd import it)
func getCredentialsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return fmt.Sprintf("%s/.agents/credentials.json", homeDir), nil
}

func init() {
	rootCmd.AddCommand(testCredentialsCmd)
}