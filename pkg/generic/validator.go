package generic

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

// Validator validates agent output
type Validator struct {
	config Validation
	logger *slog.Logger
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// NewValidator creates a new validator
func NewValidator(config Validation, logger *slog.Logger) (*Validator, error) {
	return &Validator{
		config: config,
		logger: logger,
	}, nil
}

// Validate validates the given data against configured rules
func (v *Validator) Validate(data interface{}) error {
	if !v.config.Enabled {
		return nil
	}

	v.logger.Info("Validating output", "rules", len(v.config.Rules))

	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	for _, rule := range v.config.Rules {
		if err := v.validateRule(data, rule, result); err != nil {
			v.logger.Error("Validation rule failed", "rule", rule.Name, "error", err)
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Rule '%s': %v", rule.Name, err))
		}
	}

	if !result.Valid {
		return fmt.Errorf("validation failed: %v", result.Errors)
	}

	if len(result.Warnings) > 0 {
		v.logger.Warn("Validation completed with warnings", "warnings", result.Warnings)
	}

	return nil
}

// validateRule validates data against a specific rule
func (v *Validator) validateRule(data interface{}, rule ValidationRule, result *ValidationResult) error {
	switch rule.Type {
	case "schema":
		return v.validateSchema(data, rule.Config, result)
	case "regex":
		return v.validateRegex(data, rule.Config, result)
	case "custom":
		return v.validateCustom(data, rule.Config, result)
	default:
		return fmt.Errorf("unsupported validation type: %s", rule.Type)
	}
}

// validateSchema validates data against a JSON schema
func (v *Validator) validateSchema(data interface{}, config map[string]interface{}, result *ValidationResult) error {
	schema, ok := config["schema"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("schema not specified or invalid")
	}

	// TODO: Implement proper JSON schema validation
	// For now, just do basic type checking
	if requiredType, ok := schema["type"].(string); ok {
		dataType := getDataType(data)
		if dataType != requiredType {
			return fmt.Errorf("expected type %s, got %s", requiredType, dataType)
		}
	}

	// Check required fields if it's an object
	if dataMap, ok := data.(map[string]interface{}); ok {
		if required, ok := schema["required"].([]interface{}); ok {
			for _, field := range required {
				if fieldName, ok := field.(string); ok {
					if _, exists := dataMap[fieldName]; !exists {
						return fmt.Errorf("required field '%s' is missing", fieldName)
					}
				}
			}
		}
	}

	return nil
}

// validateRegex validates data against a regular expression
func (v *Validator) validateRegex(data interface{}, config map[string]interface{}, result *ValidationResult) error {
	pattern, ok := config["pattern"].(string)
	if !ok {
		return fmt.Errorf("regex pattern not specified")
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}

	dataStr := fmt.Sprintf("%v", data)
	if !regex.MatchString(dataStr) {
		return fmt.Errorf("data does not match pattern '%s'", pattern)
	}

	return nil
}

// validateCustom validates data using custom logic
func (v *Validator) validateCustom(data interface{}, config map[string]interface{}, result *ValidationResult) error {
	// TODO: Implement custom validation logic
	// This could involve calling external validators, custom functions, etc.

	validatorType, ok := config["validator"].(string)
	if !ok {
		return fmt.Errorf("custom validator type not specified")
	}

	switch validatorType {
	case "not_empty":
		return v.validateNotEmpty(data, config, result)
	case "length_range":
		return v.validateLengthRange(data, config, result)
	case "value_range":
		return v.validateValueRange(data, config, result)
	default:
		return fmt.Errorf("unsupported custom validator: %s", validatorType)
	}
}

// validateNotEmpty checks that data is not empty
func (v *Validator) validateNotEmpty(data interface{}, config map[string]interface{}, result *ValidationResult) error {
	if data == nil {
		return fmt.Errorf("data is nil")
	}

	switch d := data.(type) {
	case string:
		if d == "" {
			return fmt.Errorf("string is empty")
		}
	case []interface{}:
		if len(d) == 0 {
			return fmt.Errorf("array is empty")
		}
	case map[string]interface{}:
		if len(d) == 0 {
			return fmt.Errorf("object is empty")
		}
	}

	return nil
}

// validateLengthRange checks that data length is within a specified range
func (v *Validator) validateLengthRange(data interface{}, config map[string]interface{}, result *ValidationResult) error {
	var length int

	switch d := data.(type) {
	case string:
		length = len(d)
	case []interface{}:
		length = len(d)
	case map[string]interface{}:
		length = len(d)
	default:
		return fmt.Errorf("length validation not supported for type %T", data)
	}

	if min, ok := config["min"].(float64); ok {
		if length < int(min) {
			return fmt.Errorf("length %d is less than minimum %d", length, int(min))
		}
	}

	if max, ok := config["max"].(float64); ok {
		if length > int(max) {
			return fmt.Errorf("length %d is greater than maximum %d", length, int(max))
		}
	}

	return nil
}

// validateValueRange checks that numeric data is within a specified range
func (v *Validator) validateValueRange(data interface{}, config map[string]interface{}, result *ValidationResult) error {
	var value float64

	switch d := data.(type) {
	case int:
		value = float64(d)
	case float64:
		value = d
	case json.Number:
		var err error
		value, err = d.Float64()
		if err != nil {
			return fmt.Errorf("cannot convert to number: %w", err)
		}
	default:
		return fmt.Errorf("value range validation not supported for type %T", data)
	}

	if min, ok := config["min"].(float64); ok {
		if value < min {
			return fmt.Errorf("value %f is less than minimum %f", value, min)
		}
	}

	if max, ok := config["max"].(float64); ok {
		if value > max {
			return fmt.Errorf("value %f is greater than maximum %f", value, max)
		}
	}

	return nil
}

// getDataType returns the type of data as a string
func getDataType(data interface{}) string {
	switch data.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case int, int64, float64, json.Number:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

// SecurityContext represents the security context for script validation
type SecurityContext struct {
	IsTrustedSource bool
	AllowedCommands []string
	BlockedCommands []string
	MaxFileSize     int64
}

// ScriptValidationResult represents the result of script security validation
type ScriptValidationResult struct {
	IsSecure        bool
	Violations      []string
	Warnings        []string
	SanitizedScript string
}

// ValidateScript performs comprehensive security validation on scripts
func (v *Validator) ValidateScript(script string, context SecurityContext) (*ScriptValidationResult, error) {
	result := &ScriptValidationResult{
		IsSecure:        true,
		Violations:      []string{},
		Warnings:        []string{},
		SanitizedScript: script,
	}

	// Skip strict validation for trusted sources, but still run basic checks
	if context.IsTrustedSource {
		v.logger.Info("Validating script from trusted source")
		return v.validateTrustedScript(script, context, result)
	}

	v.logger.Info("Validating script from untrusted source with strict security")
	return v.validateUntrustedScript(script, context, result)
}

// validateTrustedScript performs basic validation on trusted scripts
func (v *Validator) validateTrustedScript(script string, context SecurityContext, result *ScriptValidationResult) (*ScriptValidationResult, error) {
	lines := strings.Split(script, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for extremely dangerous commands even in trusted scripts
		if v.containsExtremelyDangerousCommand(line) {
			result.IsSecure = false
			result.Violations = append(result.Violations,
				fmt.Sprintf("Line %d: Extremely dangerous command detected: %s", i+1, line))
		}

		// Check for potential security issues and warn (but don't fail for trusted sources)
		if v.containsPotentiallyDangerousCommand(line) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Line %d: Potentially dangerous command: %s", i+1, line))
		}
	}

	return result, nil
}

// validateUntrustedScript performs strict validation on untrusted scripts
func (v *Validator) validateUntrustedScript(script string, context SecurityContext, result *ScriptValidationResult) (*ScriptValidationResult, error) {
	lines := strings.Split(script, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for dangerous commands
		if v.containsDangerousCommand(line) {
			result.IsSecure = false
			result.Violations = append(result.Violations,
				fmt.Sprintf("Line %d: Dangerous command detected: %s", i+1, line))
		}

		// Check for path traversal attempts
		if v.containsPathTraversal(line) {
			result.IsSecure = false
			result.Violations = append(result.Violations,
				fmt.Sprintf("Line %d: Path traversal detected: %s", i+1, line))
		}

		// Check for network operations
		if v.containsNetworkOperation(line) {
			result.IsSecure = false
			result.Violations = append(result.Violations,
				fmt.Sprintf("Line %d: Network operation detected: %s", i+1, line))
		}

		// Check for system modifications
		if v.containsSystemModification(line) {
			result.IsSecure = false
			result.Violations = append(result.Violations,
				fmt.Sprintf("Line %d: System modification detected: %s", i+1, line))
		}

		// Check against custom blocked commands
		if v.containsBlockedCommand(line, context.BlockedCommands) {
			result.IsSecure = false
			result.Violations = append(result.Violations,
				fmt.Sprintf("Line %d: Blocked command detected: %s", i+1, line))
		}
	}

	// Additional checks for file size limits
	if context.MaxFileSize > 0 && int64(len(script)) > context.MaxFileSize {
		result.IsSecure = false
		result.Violations = append(result.Violations,
			fmt.Sprintf("Script exceeds maximum allowed size of %d bytes", context.MaxFileSize))
	}

	return result, nil
}

// containsExtremelyDangerousCommand checks for commands that should never be allowed
func (v *Validator) containsExtremelyDangerousCommand(line string) bool {
	extremelyDangerous := []string{
		"rm -rf /",
		"rm -rf /*",
		"dd if=/dev/zero of=/dev/",
		"mkfs.",
		"fdisk",
		":(){ :|:& };:", // Fork bomb
		"> /dev/",       // Direct device writes
	}

	lowerLine := strings.ToLower(line)
	for _, cmd := range extremelyDangerous {
		if strings.Contains(lowerLine, strings.ToLower(cmd)) {
			return true
		}
	}
	return false
}

// containsPotentiallyDangerousCommand checks for commands that might be risky
func (v *Validator) containsPotentiallyDangerousCommand(line string) bool {
	potentiallyDangerous := []string{
		"rm ", "sudo ", "chmod ", "chown ", "mv ", "cp /",
	}

	lowerLine := strings.ToLower(line)
	for _, cmd := range potentiallyDangerous {
		if strings.Contains(lowerLine, strings.ToLower(cmd)) {
			return true
		}
	}
	return false
}

// containsDangerousCommand checks for dangerous commands in untrusted scripts
func (v *Validator) containsDangerousCommand(line string) bool {
	dangerous := []string{
		"rm ", "sudo ", "su ", "chmod ", "chown ", "passwd ",
		"useradd ", "userdel ", "groupadd ", "groupdel ",
		"mount ", "umount ", "fdisk ", "mkfs.", "fsck",
		"iptables ", "ufw ", "firewall-cmd ",
		"crontab ", "systemctl ", "service ",
		"kill ", "killall ", "pkill ",
		"eval ", "exec ", "source ", ". ",
		"curl ", "wget ", "nc ", "netcat ", "telnet ",
		"ssh ", "scp ", "rsync ", "ftp ", "sftp ",
	}

	lowerLine := strings.ToLower(line)
	for _, cmd := range dangerous {
		if strings.Contains(lowerLine, strings.ToLower(cmd)) {
			return true
		}
	}
	return false
}

// containsPathTraversal checks for path traversal attempts
func (v *Validator) containsPathTraversal(line string) bool {
	patterns := []string{
		"../",
		"..\\",
		"/etc/",
		"/proc/",
		"/sys/",
		"/dev/",
		"/root/",
		"/boot/",
	}

	lowerLine := strings.ToLower(line)
	for _, pattern := range patterns {
		if strings.Contains(lowerLine, pattern) {
			return true
		}
	}
	return false
}

// containsNetworkOperation checks for network-related operations
func (v *Validator) containsNetworkOperation(line string) bool {
	networkOps := []string{
		"curl ", "wget ", "nc ", "netcat ", "telnet ",
		"ssh ", "scp ", "rsync ", "ftp ", "sftp ",
		"ping ", "nmap ", "netstat ", "ss ",
	}

	lowerLine := strings.ToLower(line)
	for _, op := range networkOps {
		if strings.Contains(lowerLine, strings.ToLower(op)) {
			return true
		}
	}
	return false
}

// containsSystemModification checks for system modification commands
func (v *Validator) containsSystemModification(line string) bool {
	systemMods := []string{
		"systemctl ", "service ", "crontab ",
		"mount ", "umount ", "swapon ", "swapoff ",
		"modprobe ", "insmod ", "rmmod ",
		"iptables ", "ufw ", "firewall-cmd ",
	}

	lowerLine := strings.ToLower(line)
	for _, mod := range systemMods {
		if strings.Contains(lowerLine, strings.ToLower(mod)) {
			return true
		}
	}
	return false
}

// containsBlockedCommand checks against custom blocked commands
func (v *Validator) containsBlockedCommand(line string, blockedCommands []string) bool {
	if len(blockedCommands) == 0 {
		return false
	}

	lowerLine := strings.ToLower(line)
	for _, blocked := range blockedCommands {
		if strings.Contains(lowerLine, strings.ToLower(blocked)) {
			return true
		}
	}
	return false
}

// CreateSecureTempFile creates a temporary file with secure permissions
func (v *Validator) CreateSecureTempFile(content, prefix string) (string, error) {
	// Create temp file with secure permissions (600 - owner read/write only)
	tempFile, err := os.CreateTemp("", prefix+"*.sh")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Set secure permissions
	if err := os.Chmod(tempFile.Name(), 0600); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to set secure permissions: %w", err)
	}

	// Write content
	if _, err := tempFile.WriteString(content); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write content: %w", err)
	}

	return tempFile.Name(), nil
}

// CleanupTempFile safely removes a temporary file
func (v *Validator) CleanupTempFile(filepath string) error {
	if filepath == "" {
		return nil
	}

	// Verify it's actually a temp file before removing
	if !strings.Contains(filepath, os.TempDir()) {
		return fmt.Errorf("refusing to delete non-temp file: %s", filepath)
	}

	return os.Remove(filepath)
}
