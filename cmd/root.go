//go:build !windows

package cmd

import (
	"github.com/alantheprice/agent/pkg/providers"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "agent",
	Short: "Generic AI Agent Framework",
	Long: `A generic, configurable AI agent framework that can be customized 
through JSON configuration files to create specialized AI agents.

The framework supports:
  • Multi-agent orchestration with complex dependency management
  • Configurable data ingestion from multiple sources  
  • Flexible workflow execution with step dependencies
  • Multiple LLM provider support (OpenAI, Gemini, Ollama, etc.)
  • Pluggable tool system and validation rules
  • Budget controls and cost management per agent
  
Available commands:
  process  - Execute multi-agent orchestration processes

Examples:
  agent process my-workflow.json
  agent process --create-example example.json`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Register all default providers
	providers.MustRegisterDefaultProviders()
	
	// Add the process command - the core of the generic agent framework
	rootCmd.AddCommand(processCmd)
}
