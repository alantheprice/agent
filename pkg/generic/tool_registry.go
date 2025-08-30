package generic

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// GenericTool represents a tool that can be executed by the agent
type GenericTool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools  map[string]GenericTool
	config map[string]Tool
	logger *slog.Logger
}

// BuiltinTool represents a built-in tool implementation
type BuiltinTool struct {
	name        string
	description string
	executor    func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(toolConfigs map[string]Tool, logger *slog.Logger) (*ToolRegistry, error) {
	registry := &ToolRegistry{
		tools:  make(map[string]GenericTool),
		config: toolConfigs,
		logger: logger,
	}

	// Register built-in tools
	registry.registerBuiltinTools()

	return registry, nil
}

// registerBuiltinTools registers the built-in tools
func (tr *ToolRegistry) registerBuiltinTools() {
	// File operations
	tr.tools["read_file"] = &BuiltinTool{
		name:        "read_file",
		description: "Read contents of a file",
		executor:    tr.executeReadFile,
	}

	tr.tools["write_file"] = &BuiltinTool{
		name:        "write_file",
		description: "Write content to a file",
		executor:    tr.executeWriteFile,
	}

	tr.tools["list_files"] = &BuiltinTool{
		name:        "list_files",
		description: "List files in a directory",
		executor:    tr.executeListFiles,
	}

	// Shell operations
	tr.tools["shell_command"] = &BuiltinTool{
		name:        "shell_command",
		description: "Execute a shell command",
		executor:    tr.executeShellCommand,
	}

	// User interaction
	tr.tools["ask_user"] = &BuiltinTool{
		name:        "ask_user",
		description: "Ask user for input",
		executor:    tr.executeAskUser,
	}

	// Data processing
	tr.tools["json_parse"] = &BuiltinTool{
		name:        "json_parse",
		description: "Parse JSON data",
		executor:    tr.executeJSONParse,
	}

	tr.tools["json_format"] = &BuiltinTool{
		name:        "json_format",
		description: "Format data as JSON",
		executor:    tr.executeJSONFormat,
	}
}

// GetTool returns a tool by name
func (tr *ToolRegistry) GetTool(name string) (GenericTool, bool) {
	// Check if tool is enabled in config
	if config, exists := tr.config[name]; exists && !config.Enabled {
		return nil, false
	}

	tool, exists := tr.tools[name]
	return tool, exists
}

// RegisterTool registers a new tool
func (tr *ToolRegistry) RegisterTool(name string, tool GenericTool) {
	tr.tools[name] = tool
}

// ListTools returns all available tools
func (tr *ToolRegistry) ListTools() []string {
	var tools []string
	for name := range tr.tools {
		// Only include enabled tools
		if config, exists := tr.config[name]; !exists || config.Enabled {
			tools = append(tools, name)
		}
	}
	return tools
}

// Tool interface implementations
func (bt *BuiltinTool) Name() string {
	return bt.name
}

func (bt *BuiltinTool) Description() string {
	return bt.description
}

func (bt *BuiltinTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return bt.executor(ctx, params)
}

// Built-in tool executors
func (tr *ToolRegistry) executeReadFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required and must be a string")
	}

	// Security checks
	if err := tr.validateFilePath(path); err != nil {
		return nil, fmt.Errorf("path validation failed: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", path)
	}

	// Read the file with size limit
	maxSize := int64(10 * 1024 * 1024) // 10MB default limit
	if maxSizeParam, ok := params["max_size"].(float64); ok {
		maxSize = int64(maxSizeParam)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Check file size
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	if stat.Size() > maxSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", stat.Size(), maxSize)
	}

	// Read content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return map[string]interface{}{
		"path":    path,
		"content": string(content),
		"size":    len(content),
		"success": true,
	}, nil
}

func (tr *ToolRegistry) executeWriteFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required and must be a string")
	}

	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter is required and must be a string")
	}

	// Security checks
	if err := tr.validateFilePath(path); err != nil {
		return nil, fmt.Errorf("path validation failed: %w", err)
	}

	// Create directory if needed
	if createDir, ok := params["create_directories"].(bool); ok && createDir {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Create backup if requested
	if createBackup, ok := params["create_backup"].(bool); ok && createBackup {
		if _, err := os.Stat(path); err == nil {
			backupPath := path + ".backup"
			if err := tr.copyFile(path, backupPath); err != nil {
				tr.logger.Warn("Failed to create backup", "path", path, "error", err)
			}
		}
	}

	// Write the file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return map[string]interface{}{
		"path":          path,
		"bytes_written": len(content),
		"success":       true,
	}, nil
}

func (tr *ToolRegistry) executeListFiles(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		path = "." // Default to current directory
	}

	// TODO: Implement actual directory listing
	// TODO: Add security checks

	return []string{fmt.Sprintf("Files in %s would be listed here", path)}, nil
}

func (tr *ToolRegistry) executeShellCommand(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	command, ok := params["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command parameter is required and must be a string")
	}

	// TODO: Add security checks against allowed commands
	// TODO: Implement actual shell execution with timeout
	// TODO: Add proper error handling

	return fmt.Sprintf("Would execute: %s", command), nil
}

func (tr *ToolRegistry) executeAskUser(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	question, ok := params["question"].(string)
	if !ok {
		return nil, fmt.Errorf("question parameter is required and must be a string")
	}

	// TODO: Implement actual user interaction
	// TODO: Add timeout handling

	return fmt.Sprintf("User response to: %s", question), nil
}

func (tr *ToolRegistry) executeJSONParse(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	jsonStr, ok := params["json"].(string)
	if !ok {
		return nil, fmt.Errorf("json parameter is required and must be a string")
	}

	// TODO: Implement actual JSON parsing

	return fmt.Sprintf("Parsed JSON from: %s", jsonStr), nil
}

func (tr *ToolRegistry) executeJSONFormat(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	data := params["data"]
	if data == nil {
		return nil, fmt.Errorf("data parameter is required")
	}

	// TODO: Implement actual JSON formatting

	return fmt.Sprintf("JSON formatted data: %v", data), nil
}

// Security helper methods

// validateFilePath checks if a file path is allowed based on security configuration
func (tr *ToolRegistry) validateFilePath(path string) error {
	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("directory traversal not allowed: %s", path)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// TODO: Check against allowed_paths from security configuration
	// For now, allow paths in current directory and subdirectories
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if !strings.HasPrefix(absPath, currentDir) {
		return fmt.Errorf("path outside allowed directories: %s", absPath)
	}

	return nil
}

// copyFile creates a backup copy of a file
func (tr *ToolRegistry) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}
