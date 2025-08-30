package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/alantheprice/ledit/pkg/generic"
	"github.com/spf13/cobra"
)

var (
	configFile string
	logLevel   string
)

var rootCmd = &cobra.Command{
	Use:   "generic-agent",
	Short: "Generic AI Agent Framework",
	Long: `A generic, configurable AI agent framework that can be customized through JSON configuration files.
	
The agent supports:
- Configurable data ingestion from multiple sources
- Flexible workflow execution with step dependencies
- Pluggable tool system
- Multiple LLM provider support
- Configurable output processing and validation`,
	Run: runAgent,
}

var runCmd = &cobra.Command{
	Use:   "run [input]",
	Short: "Run the agent with input",
	Args:  cobra.MinimumNArgs(1),
	Run:   runAgent,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration file",
	Run:   validateConfig,
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Print the JSON schema for agent configuration",
	Run:   printSchema,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "agent.json", "Configuration file path")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(schemaCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
