package generic

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
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
