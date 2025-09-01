package generic

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GenericTool represents a tool that can be executed by the agent
type GenericTool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools    map[string]GenericTool
	config   map[string]Tool
	security *Security
	logger   *slog.Logger
}

// BuiltinTool represents a built-in tool implementation
type BuiltinTool struct {
	name        string
	description string
	executor    func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(toolConfigs map[string]Tool, security *Security, logger *slog.Logger) (*ToolRegistry, error) {
	registry := &ToolRegistry{
		tools:    make(map[string]GenericTool),
		config:   toolConfigs,
		security: security,
		logger:   logger,
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

	// Git operations
	tr.tools["git_status"] = &BuiltinTool{
		name:        "git_status",
		description: "Get git repository status",
		executor:    tr.executeGitStatus,
	}

	tr.tools["git_diff"] = &BuiltinTool{
		name:        "git_diff",
		description: "Get git diff for staged changes",
		executor:    tr.executeGitDiff,
	}

	tr.tools["git_commit"] = &BuiltinTool{
		name:        "git_commit",
		description: "Execute git commit with message",
		executor:    tr.executeGitCommit,
	}
}

// GetTool returns a tool by name
func (tr *ToolRegistry) GetTool(name string) (GenericTool, bool) {
	// Check if tool is enabled in config
	if config, exists := tr.config[name]; exists && !config.Enabled {
		tr.logger.Debug("Tool disabled in config", "tool", name)
		return nil, false
	}

	// Look up tool in registry
	tool, exists := tr.tools[name]
	if !exists {
		tr.logger.Debug("Tool not found in registry", "tool", name, "available_tools", getMapKeys(tr.tools))
		return nil, false
	}

	tr.logger.Debug("Tool found", "tool", name)
	return tool, true
}

// Helper function to get map keys for debugging
func getMapKeys(m map[string]GenericTool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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

	tr.logger.Debug("Executing shell command", "command", command)

	// Set timeout
	timeout := 30 * time.Second
	if timeoutParam, ok := params["timeout"].(float64); ok {
		timeout = time.Duration(timeoutParam) * time.Second
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute the command using bash -c
	cmd := exec.CommandContext(ctxWithTimeout, "bash", "-c", command)

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	result := map[string]interface{}{
		"command": command,
		"output":  string(output),
		"success": err == nil,
	}

	if err != nil {
		tr.logger.Error("Shell command failed",
			"command", command,
			"error", err,
			"output", string(output))
		result["error"] = err.Error()
		// Return result with error info, don't fail completely
		return result, nil
	}

	tr.logger.Debug("Shell command executed successfully",
		"command", command,
		"output_length", len(output))

	return result, nil
}

func (tr *ToolRegistry) executeAskUser(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	question, ok := params["question"].(string)
	if !ok {
		return nil, fmt.Errorf("question parameter is required and must be a string")
	}

	// Display the question to the user
	fmt.Print(question + " ")

	// Read user input with timeout support
	timeout := 300 * time.Second // Default 5 minutes
	if timeoutParam, ok := params["timeout"].(float64); ok {
		timeout = time.Duration(timeoutParam) * time.Second
	}

	// Create a channel to receive the user input
	inputChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			inputChan <- strings.TrimSpace(scanner.Text())
		} else {
			if err := scanner.Err(); err != nil {
				errorChan <- err
			} else {
				errorChan <- fmt.Errorf("input stream closed")
			}
		}
	}()

	// Wait for input or timeout
	select {
	case input := <-inputChan:
		return map[string]interface{}{
			"response": input,
			"success":  true,
		}, nil
	case err := <-errorChan:
		return nil, fmt.Errorf("failed to read user input: %w", err)
	case <-time.After(timeout):
		return nil, fmt.Errorf("user input timeout after %v", timeout)
	case <-ctx.Done():
		return nil, fmt.Errorf("operation cancelled: %w", ctx.Err())
	}
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

// Git tool implementations

func (tr *ToolRegistry) executeGitStatus(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Execute git status command using shell_command
	return tr.executeShellCommand(ctx, map[string]interface{}{
		"command": "git status --porcelain",
		"timeout": 30.0,
	})
}

func (tr *ToolRegistry) executeGitDiff(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Default to staged changes, but allow customization
	command := "git diff --staged"

	if diffType, ok := params["type"].(string); ok {
		switch diffType {
		case "staged":
			command = "git diff --staged"
		case "unstaged":
			command = "git diff"
		case "all":
			command = "git diff HEAD"
		}
	}

	return tr.executeShellCommand(ctx, map[string]interface{}{
		"command": command,
		"timeout": 30.0,
	})
}

func (tr *ToolRegistry) executeGitCommit(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	message, ok := params["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message parameter is required and must be a string")
	}

	// Clean and validate the commit message
	message = strings.TrimSpace(message)
	if message == "" {
		return nil, fmt.Errorf("commit message cannot be empty")
	}

	// Execute git commit using shell_command
	command := fmt.Sprintf("git commit -m %q", message)

	result, err := tr.executeShellCommand(ctx, map[string]interface{}{
		"command": command,
		"timeout": 30.0,
	})

	if err != nil {
		return result, err
	}

	// If successful, also get the commit hash
	hashResult, _ := tr.executeShellCommand(ctx, map[string]interface{}{
		"command": "git rev-parse HEAD",
		"timeout": 10.0,
	})

	// Enhance the result with commit hash if available
	if resultMap, ok := result.(map[string]interface{}); ok {
		if hashMap, ok := hashResult.(map[string]interface{}); ok {
			if hashOutput, ok := hashMap["output"].(string); ok {
				resultMap["commit_hash"] = strings.TrimSpace(hashOutput)
			}
		}
	}

	return result, nil
}
