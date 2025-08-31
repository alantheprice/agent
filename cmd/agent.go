package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/alantheprice/agent-template/pkg/generic"
	"github.com/spf13/cobra"
)

var (
	configFile string
	logLevel   string
)

// runCmd represents the direct agent execution command
var runCmd = &cobra.Command{
	Use:   "run [input]",
	Short: "Run a single agent with input",
	Long: `Direct agent execution mode:
	- Loads a single agent configuration file
	- Executes the agent with provided input
	- Supports all generic agent framework features
	
	Examples:
	  agent-template run "Process this data" --config agent.json
	  agent-template run --config my-agent.json`,
	Run: runAgent,
}

// validateCmd validates an agent configuration
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate an agent configuration file",
	Long: `Validates an agent configuration file without executing it.
	Checks syntax, required fields, and configuration consistency.
	
	Examples:
	  agent-template validate --config agent.json`,
	Run: validateConfig,
}

// schemaCmd prints the JSON schema
var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Print the JSON schema for agent configuration",
	Long: `Prints information about the JSON schema for agent configuration files.
	Use this schema to validate your configuration files with tools like ajv.`,
	Run: printSchema,
}

func init() {
	// Add flags to all generic agent commands
	runCmd.Flags().StringVarP(&configFile, "config", "c", "agent.json", "Configuration file path")
	runCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")

	validateCmd.Flags().StringVarP(&configFile, "config", "c", "agent.json", "Configuration file path")
	validateCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")

	schemaCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")

	// Add commands to root
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(schemaCmd)
}

func runAgent(cmd *cobra.Command, args []string) {
	// Set up logging
	logger := setupLogger()

	// Load configuration
	config, err := generic.LoadConfig(configFile)
	if err != nil {
		logger.Error("Failed to load configuration", "error", err, "config_file", configFile)
		os.Exit(1)
	}

	logger.Info("Configuration loaded successfully", "agent", config.Agent.Name)

	// Create agent
	agent, err := generic.NewAgent(config, logger)
	if err != nil {
		logger.Error("Failed to create agent", "error", err)
		os.Exit(1)
	}

	// Determine input
	var input string
	if len(args) > 0 {
		input = args[0]
	} else {
		// Interactive mode - read from stdin or prompt
		fmt.Print("Enter input: ")
		fmt.Scanln(&input)
	}

	// Execute agent
	logger.Info("Starting agent execution", "input", input)
	if err := agent.Execute(input); err != nil {
		logger.Error("Agent execution failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Agent execution completed successfully")
}

func validateConfig(cmd *cobra.Command, args []string) {
	logger := setupLogger()

	config, err := generic.LoadConfig(configFile)
	if err != nil {
		logger.Error("Configuration validation failed", "error", err, "config_file", configFile)
		os.Exit(1)
	}

	logger.Info("Configuration is valid", "config_file", configFile, "agent", config.Agent.Name)
	fmt.Printf("✅ Configuration is valid\n")
	fmt.Printf("Agent: %s\n", config.Agent.Name)
	fmt.Printf("Description: %s\n", config.Agent.Description)
	fmt.Printf("LLM Provider: %s\n", config.LLM.Provider)
	fmt.Printf("Model: %s\n", config.LLM.Model)

	if len(config.Workflows) > 0 {
		fmt.Printf("Workflows: %d\n", len(config.Workflows))
	}

	if len(config.DataSources) > 0 {
		fmt.Printf("Data Sources: %d\n", len(config.DataSources))
	}

	if len(config.Tools) > 0 {
		fmt.Printf("Tools: %d\n", len(config.Tools))
	}
}

func printSchema(cmd *cobra.Command, args []string) {
	// This would print the JSON schema
	// For now, just point to the schema file
	fmt.Printf("Agent configuration JSON schema is available at: schemas/agent-config.json\n")
	fmt.Printf("Use it to validate your configuration files with tools like ajv or jsonschema.\n")
}

func setupLogger() *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}
