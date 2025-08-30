package generic

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// OutputWriter handles writing agent output to various destinations
type OutputWriter struct {
	outputs []Output
	logger  *slog.Logger
}

// NewOutputWriter creates a new output writer
func NewOutputWriter(outputs []Output, logger *slog.Logger) (*OutputWriter, error) {
	return &OutputWriter{
		outputs: outputs,
		logger:  logger,
	}, nil
}

// WriteAll writes output to all configured destinations
func (ow *OutputWriter) WriteAll(data interface{}, execCtx *ExecutionContext) error {
	for _, output := range ow.outputs {
		ow.logger.Info("Writing output", "output", output.Name, "type", output.Type)

		if err := ow.writeOutput(output, data, execCtx); err != nil {
			ow.logger.Error("Failed to write output", "output", output.Name, "error", err)
			continue
		}
	}
	return nil
}

// writeOutput writes to a specific output destination
func (ow *OutputWriter) writeOutput(output Output, data interface{}, execCtx *ExecutionContext) error {
	// Apply filters if configured
	filteredData, err := ow.applyFilters(data, output.Filter)
	if err != nil {
		return fmt.Errorf("failed to apply filters: %w", err)
	}

	// Format the data
	formattedData, err := ow.formatData(filteredData, output)
	if err != nil {
		return fmt.Errorf("failed to format data: %w", err)
	}

	switch output.Type {
	case "file":
		return ow.writeToFile(formattedData, output, execCtx)
	case "console":
		return ow.writeToConsole(formattedData, output, execCtx)
	case "api":
		return ow.writeToAPI(formattedData, output, execCtx)
	case "database":
		return ow.writeToDatabase(formattedData, output, execCtx)
	case "webhook":
		return ow.writeToWebhook(formattedData, output, execCtx)
	default:
		return fmt.Errorf("unsupported output type: %s", output.Type)
	}
}

// applyFilters applies output filters
func (ow *OutputWriter) applyFilters(data interface{}, filter OutputFilter) (interface{}, error) {
	if len(filter.Include) == 0 && len(filter.Exclude) == 0 && filter.Transform == "" {
		return data, nil
	}

	// TODO: Implement sophisticated filtering logic
	// For now, just return the data as-is
	return data, nil
}

// formatData formats data according to the output configuration
func (ow *OutputWriter) formatData(data interface{}, output Output) ([]byte, error) {
	format := "json" // default format
	if formatStr, ok := output.Config["format"].(string); ok {
		format = formatStr
	}

	switch format {
	case "json":
		return json.MarshalIndent(data, "", "  ")
	case "yaml":
		// TODO: Implement YAML formatting
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, err
		}
		return jsonData, nil
	case "text":
		return []byte(fmt.Sprintf("%v", data)), nil
	case "markdown":
		return ow.formatAsMarkdown(data)
	case "csv":
		// TODO: Implement CSV formatting
		return []byte(fmt.Sprintf("%v", data)), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// formatAsMarkdown formats data as markdown
func (ow *OutputWriter) formatAsMarkdown(data interface{}) ([]byte, error) {
	var builder strings.Builder

	builder.WriteString("# Agent Output\n\n")

	if dataMap, ok := data.(map[string]interface{}); ok {
		for key, value := range dataMap {
			builder.WriteString(fmt.Sprintf("## %s\n\n", key))
			builder.WriteString(fmt.Sprintf("```\n%v\n```\n\n", value))
		}
	} else {
		builder.WriteString(fmt.Sprintf("```\n%v\n```\n\n", data))
	}

	return []byte(builder.String()), nil
}

// writeToFile writes output to a file
func (ow *OutputWriter) writeToFile(data []byte, output Output, execCtx *ExecutionContext) error {
	path, ok := output.Config["path"].(string)
	if !ok {
		return fmt.Errorf("file path not specified")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check append mode
	append, _ := output.Config["append"].(bool)
	flags := os.O_CREATE | os.O_WRONLY
	if append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// writeToConsole writes output to console
func (ow *OutputWriter) writeToConsole(data []byte, output Output, execCtx *ExecutionContext) error {
	fmt.Printf("=== %s ===\n", output.Name)
	fmt.Printf("%s\n", data)
	fmt.Println("================")
	return nil
}

// writeToAPI writes output to an API endpoint
func (ow *OutputWriter) writeToAPI(data []byte, output Output, execCtx *ExecutionContext) error {
	// TODO: Implement API writing
	ow.logger.Info("Would write to API", "output", output.Name, "size", len(data))
	return nil
}

// writeToDatabase writes output to a database
func (ow *OutputWriter) writeToDatabase(data []byte, output Output, execCtx *ExecutionContext) error {
	// TODO: Implement database writing
	ow.logger.Info("Would write to database", "output", output.Name, "size", len(data))
	return nil
}

// writeToWebhook writes output to a webhook
func (ow *OutputWriter) writeToWebhook(data []byte, output Output, execCtx *ExecutionContext) error {
	// TODO: Implement webhook writing
	ow.logger.Info("Would write to webhook", "output", output.Name, "size", len(data))
	return nil
}
