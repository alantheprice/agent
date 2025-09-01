package generic

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alantheprice/agent-template/pkg/embedding"
)

// GenericTool represents a tool that can be executed by the agent
type GenericTool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools                map[string]GenericTool
	config               map[string]Tool
	security             *Security
	logger               *slog.Logger
	embeddingDataSources map[string]*embedding.EmbeddingDataSource
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
		tools:                make(map[string]GenericTool),
		config:               toolConfigs,
		security:             security,
		logger:               logger,
		embeddingDataSources: make(map[string]*embedding.EmbeddingDataSource),
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

	tr.tools["embedding_ingest"] = &BuiltinTool{
		name:        "embedding_ingest",
		description: "Build embeddings for workspace files",
		executor:    tr.executeEmbeddingIngest,
	}

	tr.tools["embedding_search"] = &BuiltinTool{
		name:        "embedding_search",
		description: "Search files using semantic similarity",
		executor:    tr.executeEmbeddingSearch,
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

// SetEmbeddingDataSources sets the embedding data sources for tools to use
func (tr *ToolRegistry) SetEmbeddingDataSources(dataSources map[string]*embedding.EmbeddingDataSource) {
	tr.embeddingDataSources = dataSources
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

	// Security check: validate path is allowed
	if !tr.isPathAllowed(path) {
		return nil, fmt.Errorf("path '%s' is not allowed by security configuration", path)
	}

	// Read directory
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory '%s': %w", path, err)
	}

	// Convert to string list with file info
	var files []map[string]interface{}
	for _, entry := range entries {
		fileInfo, err := entry.Info()
		if err != nil {
			tr.logger.Debug("Failed to get file info", "file", entry.Name(), "error", err)
			continue
		}

		files = append(files, map[string]interface{}{
			"name":        entry.Name(),
			"type":        getFileType(entry),
			"size":        fileInfo.Size(),
			"modified":    fileInfo.ModTime(),
			"permissions": fileInfo.Mode().String(),
		})
	}

	return map[string]interface{}{
		"path":       path,
		"file_count": len(files),
		"files":      files,
	}, nil
}

// getFileType determines the type of a directory entry
func getFileType(entry os.DirEntry) string {
	if entry.IsDir() {
		return "directory"
	}
	if info, err := entry.Info(); err == nil {
		mode := info.Mode()
		if mode&os.ModeSymlink != 0 {
			return "symlink"
		}
		if mode&os.ModeCharDevice != 0 {
			return "char_device"
		}
		if mode&os.ModeDevice != 0 {
			return "device"
		}
		if mode&os.ModeNamedPipe != 0 {
			return "pipe"
		}
		if mode&os.ModeSocket != 0 {
			return "socket"
		}
	}
	return "file"
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
				// Handle stream closed gracefully with default response
				defaultResponse, hasDefault := params["default_response"].(string)
				if hasDefault {
					inputChan <- defaultResponse
				} else {
					errorChan <- fmt.Errorf("input stream closed")
				}
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

	// Parse JSON string into interface{}
	var parsed interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return map[string]interface{}{
		"parsed_data": parsed,
		"type":        fmt.Sprintf("%T", parsed),
		"success":     true,
	}, nil
}

func (tr *ToolRegistry) executeJSONFormat(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	data := params["data"]
	if data == nil {
		return nil, fmt.Errorf("data parameter is required")
	}

	// Get formatting options
	indent := "  " // default indent
	if indentParam, ok := params["indent"].(string); ok {
		indent = indentParam
	}

	compact := false
	if compactParam, ok := params["compact"].(bool); ok {
		compact = compactParam
	}

	// Format JSON
	var jsonBytes []byte
	var err error

	if compact {
		jsonBytes, err = json.Marshal(data)
	} else {
		jsonBytes, err = json.MarshalIndent(data, "", indent)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to format JSON: %w", err)
	}

	return map[string]interface{}{
		"json":     string(jsonBytes),
		"compact":  compact,
		"indent":   indent,
		"size":     len(jsonBytes),
		"success":  true,
	}, nil
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

// isPathAllowed checks if a path is allowed (simpler version of validateFilePath)
func (tr *ToolRegistry) isPathAllowed(path string) bool {
	err := tr.validateFilePath(path)
	return err == nil
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

// Embedding tool implementations

func (tr *ToolRegistry) executeEmbeddingIngest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	sourceName, ok := params["source_name"].(string)
	if !ok {
		return nil, fmt.Errorf("source_name parameter is required and must be a string")
	}

	tr.logger.Info("Executing embedding ingest", "source_name", sourceName)

	// Check if embedding data source exists
	embeddingDataSource, exists := tr.embeddingDataSources[sourceName]
	if !exists {
		return nil, fmt.Errorf("embedding data source '%s' not found", sourceName)
	}

	// Trigger re-ingestion of data
	stats, err := embeddingDataSource.IngestData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ingest embedding data for source '%s': %w", sourceName, err)
	}

	return map[string]interface{}{
		"source_name":        sourceName,
		"status":             "completed",
		"files_processed":    stats["files_processed"],
		"embeddings_created": stats["embeddings_created"],
		"message":            fmt.Sprintf("Embedding ingest completed for source: %s", sourceName),
		"stats":              stats,
	}, nil
}

func (tr *ToolRegistry) executeEmbeddingSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required and must be a string")
	}

	sourceName, ok := params["source_name"].(string)
	if !ok {
		return nil, fmt.Errorf("source_name parameter is required and must be a string")
	}

	limit := 3 // default
	if limitParam, ok := params["limit"].(float64); ok {
		limit = int(limitParam)
	}

	minSimilarity := 0.3 // default
	if simParam, ok := params["min_similarity"].(float64); ok {
		minSimilarity = simParam
	}

	tr.logger.Info("Executing embedding search", "query", query, "source_name", sourceName, "limit", limit)

	// Check if embedding data source exists
	embeddingDataSource, exists := tr.embeddingDataSources[sourceName]
	if !exists {
		return nil, fmt.Errorf("embedding data source '%s' not found", sourceName)
	}

	// Perform semantic search using the embedding data source
	searchResults, similarities, err := embeddingDataSource.SearchContent(ctx, query, limit, minSimilarity)
	if err != nil {
		return nil, fmt.Errorf("failed to search embeddings for source '%s': %w", sourceName, err)
	}

	// Convert results to the expected format
	var results []map[string]interface{}
	for i, result := range searchResults {
		similarity := 0.0
		if i < len(similarities) {
			similarity = similarities[i]
		}
		results = append(results, map[string]interface{}{
			"file_path":   result.Source,
			"content":     result.Content,
			"similarity":  similarity,
			"metadata":    result.Metadata,
			"type":        result.Type,
			"id":          result.ID,
		})
	}

	return map[string]interface{}{
		"query":          query,
		"source_name":    sourceName,
		"limit":          limit,
		"min_similarity": minSimilarity,
		"results":        results,
		"total_results":  len(results),
		"message":        fmt.Sprintf("Found %d matching results for query: %s", len(results), query),
	}, nil
}

