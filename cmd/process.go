package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alantheprice/agent-template/pkg/generic"
	"github.com/spf13/cobra"
)

var (
	createExample bool
	resume        bool
	statePath     string
	noProgress    bool
	dryRun        bool
	skipPrompt    bool
	model         string
	debug         bool
	verbose       bool
)

// processCmd represents the process command
var processCmd = &cobra.Command{
	Use:   "process [process-file]",
	Short: "Execute a multi-agent orchestration process",
	Long: `Multi-Agent Process Mode:
	- Loads a process file defining agents, steps, and dependencies
	- Coordinates multiple agents with specialized personas (e.g., frontend developer, backend architect, QA engineer)
	- Executes steps in dependency order
	- Tracks progress and agent status
	- Supports budget controls and cost management per agent

	Examples:
	  agent-template process process.json
	  agent-template process --create-example process.json
	  agent-template process --dry-run process.json`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle create-example flag
		if createExample {
			out := "process.json"
			if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
				out = args[0]
			}
			if err := createExampleProcessFile(out); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating example process file: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Require process file
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "Error: process file required\n")
			fmt.Fprintf(os.Stderr, "Usage: %s process <process-file>\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "   or: %s process --create-example <output-file>\n", os.Args[0])
			os.Exit(1)
		}

		input := args[0]

		// Dry-run validation
		if dryRun {
			if err := validateProcessOnly(input); err != nil {
				fmt.Fprintf(os.Stderr, "Process validation failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ Process file is valid")
			return
		}

		// Execute multi-agent process
		if err := runMultiAgentProcess(input); err != nil {
			fmt.Fprintf(os.Stderr, "Multi-agent process failed: %v\n", err)
			os.Exit(1)
		}
	},
}

// runMultiAgentProcess executes a multi-agent orchestration process using generic framework
func runMultiAgentProcess(processFilePath string) error {
	fmt.Printf("🚀 Starting generic agent process\n")
	fmt.Printf("Config file: %s\n", processFilePath)

	// Load agent config
	config, err := generic.LoadConfig(processFilePath)
	if err != nil {
		return fmt.Errorf("failed to load agent config: %w", err)
	}

	// Create logger with configurable level
	logLevel := slog.LevelWarn // Default to warn level (minimal output)

	// Check command line flags first
	if debug {
		logLevel = slog.LevelDebug
	} else if verbose {
		logLevel = slog.LevelInfo
	} else {
		// Check environment variables as fallback
		if debugFlag := os.Getenv("DEBUG"); debugFlag == "true" || debugFlag == "1" {
			logLevel = slog.LevelDebug
		} else if verboseFlag := os.Getenv("VERBOSE"); verboseFlag == "true" || verboseFlag == "1" {
			logLevel = slog.LevelInfo
		}
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	// Create and execute agent
	agent, err := generic.NewAgent(config, logger)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Execute with default input
	input := "Execute the configured workflow"
	if err := agent.Execute(input); err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	fmt.Println("✅ Generic agent process completed successfully")
	return nil
}

// createExampleProcessFile creates an example agent configuration file
func createExampleProcessFile(filePath string) error {
	fmt.Printf("📝 Creating example agent config file: %s\n", filePath)

	// Create a minimal example config using the generic framework
	config := &generic.AgentConfig{
		Agent: generic.AgentInfo{
			Name:        "Example Agent",
			Description: "A simple example agent",
			Version:     "1.0.0",
		},
		LLM: generic.LLMConfig{
			Provider:    "openai",
			Model:       "gpt-4",
			Temperature: 0.7,
			MaxTokens:   4096,
		},
		Workflows: []generic.Workflow{
			{
				Name:        "example_workflow",
				Description: "An example workflow",
				Steps: []generic.Step{
					{
						Name: "hello_world",
						Type: "llm",
						Config: map[string]interface{}{
							"prompt": "Say hello world!",
						},
					},
				},
			},
		},
	}

	if err := generic.SaveConfig(config, filePath); err != nil {
		return fmt.Errorf("failed to create example config file: %w", err)
	}

	fmt.Printf("✅ Example config file created successfully\n")
	fmt.Printf("You can now edit this file and run: %s process %s\n", os.Args[0], filePath)
	return nil
}

// validateProcessOnly loads and validates an agent config file without executing
func validateProcessOnly(processFilePath string) error {
	fmt.Printf("🔎 Validating agent config file: %s\n", processFilePath)

	// Try to load and validate the config using the generic framework
	_, err := generic.LoadConfig(processFilePath)
	return err
}

func init() {
	processCmd.Flags().StringVarP(&model, "model", "m", "", "Model to use for orchestration and editing")
	processCmd.Flags().BoolVar(&skipPrompt, "skip-prompt", false, "Skip the confirmation prompt and proceed with the plan")
	processCmd.Flags().BoolVar(&createExample, "create-example", false, "Create an example process file instead of executing")
	processCmd.Flags().BoolVar(&resume, "resume", false, "Resume from a previous orchestration state if compatible")
	processCmd.Flags().StringVar(&statePath, "state", "", "Path to orchestration state file (default .ledit/orchestration_state.json)")
	processCmd.Flags().BoolVar(&noProgress, "no-progress", false, "Suppress progress table output during orchestration")
	processCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate process file without executing")
	processCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")
	processCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
}
