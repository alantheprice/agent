package generic

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Transformer interface defines data transformation operations
type Transformer interface {
	Transform(input interface{}, params map[string]interface{}) (interface{}, error)
	ValidateParams(params map[string]interface{}) error
	Name() string
	Description() string
}

// TransformRegistry manages available transformers
type TransformRegistry struct {
	transformers map[string]Transformer
	logger       *slog.Logger
}

// NewTransformRegistry creates a new transform registry with built-in transformers
func NewTransformRegistry(logger *slog.Logger) *TransformRegistry {
	registry := &TransformRegistry{
		transformers: make(map[string]Transformer),
		logger:       logger,
	}

	// Register built-in transformers
	registry.registerBuiltinTransformers()

	return registry
}

// GetTransformer gets a transformer by name
func (tr *TransformRegistry) GetTransformer(name string) (Transformer, bool) {
	transformer, exists := tr.transformers[name]
	return transformer, exists
}

// RegisterTransformer registers a new transformer
func (tr *TransformRegistry) RegisterTransformer(transformer Transformer) {
	tr.transformers[transformer.Name()] = transformer
}

// ListTransformers returns all available transformer names
func (tr *TransformRegistry) ListTransformers() []string {
	var names []string
	for name := range tr.transformers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// registerBuiltinTransformers registers all built-in transformers
func (tr *TransformRegistry) registerBuiltinTransformers() {
	tr.RegisterTransformer(&LineExtractor{})
	tr.RegisterTransformer(&JSONParser{})
	tr.RegisterTransformer(&Aggregator{})
	tr.RegisterTransformer(&DataFilter{})
	tr.RegisterTransformer(&TextFormatter{})
	tr.RegisterTransformer(&DataMerger{})
	tr.RegisterTransformer(&Deduplicator{})
	tr.RegisterTransformer(&DataSorter{})
	tr.RegisterTransformer(&RegexExtractor{})
	tr.RegisterTransformer(&StringProcessor{})
}

// Built-in Transformers

// LineExtractor extracts lines matching patterns
type LineExtractor struct{}

func (le *LineExtractor) Name() string        { return "extract_lines" }
func (le *LineExtractor) Description() string { return "Extract lines matching a regex pattern" }

func (le *LineExtractor) ValidateParams(params map[string]interface{}) error {
	if _, ok := params["pattern"]; !ok {
		return fmt.Errorf("pattern parameter is required")
	}
	return nil
}

func (le *LineExtractor) Transform(input interface{}, params map[string]interface{}) (interface{}, error) {
	inputStr, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("input must be string, got %T", input)
	}

	pattern, ok := params["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern must be string, got %T", params["pattern"])
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	lines := strings.Split(inputStr, "\n")
	var matchedLines []string

	for _, line := range lines {
		if regex.MatchString(line) {
			matchedLines = append(matchedLines, line)
		}
	}

	// Return based on mode parameter
	mode, _ := params["mode"].(string)
	if mode == "count" {
		return len(matchedLines), nil
	} else if mode == "first" && len(matchedLines) > 0 {
		return matchedLines[0], nil
	} else if mode == "joined" {
		delimiter, _ := params["delimiter"].(string)
		if delimiter == "" {
			delimiter = "\n"
		}
		return strings.Join(matchedLines, delimiter), nil
	}

	return matchedLines, nil
}

// JSONParser parses JSON from text
type JSONParser struct{}

func (jp *JSONParser) Name() string        { return "parse_json" }
func (jp *JSONParser) Description() string { return "Parse JSON from string input" }

func (jp *JSONParser) ValidateParams(params map[string]interface{}) error {
	return nil
}

func (jp *JSONParser) Transform(input interface{}, params map[string]interface{}) (interface{}, error) {
	inputStr, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("input must be string, got %T", input)
	}

	var result interface{}
	err := json.Unmarshal([]byte(inputStr), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

// Aggregator performs count, sum, average operations
type Aggregator struct{}

func (ag *Aggregator) Name() string { return "aggregate" }
func (ag *Aggregator) Description() string {
	return "Perform aggregation operations (count, sum, average)"
}

func (ag *Aggregator) ValidateParams(params map[string]interface{}) error {
	operation, ok := params["operation"].(string)
	if !ok {
		return fmt.Errorf("operation parameter is required")
	}

	validOps := []string{"count", "sum", "average", "min", "max"}
	for _, op := range validOps {
		if operation == op {
			return nil
		}
	}

	return fmt.Errorf("invalid operation: %s. Valid operations: %v", operation, validOps)
}

func (ag *Aggregator) Transform(input interface{}, params map[string]interface{}) (interface{}, error) {
	operation := params["operation"].(string)
	field, _ := params["field"].(string)

	v := reflect.ValueOf(input)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("input must be array or slice, got %T", input)
	}

	switch operation {
	case "count":
		return v.Len(), nil

	case "sum", "average":
		var sum float64
		var count int

		for i := 0; i < v.Len(); i++ {
			item := v.Index(i).Interface()
			var val float64

			if field != "" {
				// Extract field from object
				fieldVal, err := ag.extractField(item, field)
				if err != nil {
					continue
				}
				val, _ = ag.toFloat(fieldVal)
			} else {
				val, _ = ag.toFloat(item)
			}

			sum += val
			count++
		}

		if operation == "sum" {
			return sum, nil
		} else {
			if count == 0 {
				return 0.0, nil
			}
			return sum / float64(count), nil
		}

	case "min", "max":
		if v.Len() == 0 {
			return nil, nil
		}

		var minVal, maxVal float64
		initialized := false

		for i := 0; i < v.Len(); i++ {
			item := v.Index(i).Interface()
			var val float64

			if field != "" {
				fieldVal, err := ag.extractField(item, field)
				if err != nil {
					continue
				}
				val, _ = ag.toFloat(fieldVal)
			} else {
				val, _ = ag.toFloat(item)
			}

			if !initialized {
				minVal = val
				maxVal = val
				initialized = true
			} else {
				if val < minVal {
					minVal = val
				}
				if val > maxVal {
					maxVal = val
				}
			}
		}

		if operation == "min" {
			return minVal, nil
		} else {
			return maxVal, nil
		}
	}

	return nil, fmt.Errorf("unsupported operation: %s", operation)
}

func (ag *Aggregator) extractField(obj interface{}, field string) (interface{}, error) {
	if m, ok := obj.(map[string]interface{}); ok {
		if val, exists := m[field]; exists {
			return val, nil
		}
	}

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Struct {
		fieldVal := v.FieldByName(field)
		if fieldVal.IsValid() {
			return fieldVal.Interface(), nil
		}
	}

	return nil, fmt.Errorf("field %s not found", field)
}

func (ag *Aggregator) toFloat(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

// DataFilter filters data by conditions
type DataFilter struct{}

func (df *DataFilter) Name() string        { return "filter_data" }
func (df *DataFilter) Description() string { return "Filter array/slice by field conditions" }

func (df *DataFilter) ValidateParams(params map[string]interface{}) error {
	if _, ok := params["condition"]; !ok {
		return fmt.Errorf("condition parameter is required")
	}
	return nil
}

func (df *DataFilter) Transform(input interface{}, params map[string]interface{}) (interface{}, error) {
	v := reflect.ValueOf(input)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("input must be array or slice, got %T", input)
	}

	condition := params["condition"].(string)
	field, _ := params["field"].(string)

	var result []interface{}

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()

		if df.matchesCondition(item, field, condition) {
			result = append(result, item)
		}
	}

	return result, nil
}

func (df *DataFilter) matchesCondition(item interface{}, field, condition string) bool {
	var checkValue interface{} = item

	if field != "" {
		if m, ok := item.(map[string]interface{}); ok {
			if val, exists := m[field]; exists {
				checkValue = val
			} else {
				return false
			}
		}
	}

	checkStr := fmt.Sprintf("%v", checkValue)

	// Simple condition matching - can be extended
	if strings.HasPrefix(condition, "contains:") {
		search := strings.TrimPrefix(condition, "contains:")
		return strings.Contains(checkStr, search)
	} else if strings.HasPrefix(condition, "equals:") {
		search := strings.TrimPrefix(condition, "equals:")
		return checkStr == search
	} else if strings.HasPrefix(condition, "not_empty") {
		return checkStr != "" && checkStr != "0" && checkStr != "false"
	}

	// Default: check if item contains condition string
	return strings.Contains(checkStr, condition)
}

// TextFormatter formats text with templates
type TextFormatter struct{}

func (tf *TextFormatter) Name() string        { return "format_text" }
func (tf *TextFormatter) Description() string { return "Format text using template strings" }

func (tf *TextFormatter) ValidateParams(params map[string]interface{}) error {
	if _, ok := params["template"]; !ok {
		return fmt.Errorf("template parameter is required")
	}
	return nil
}

func (tf *TextFormatter) Transform(input interface{}, params map[string]interface{}) (interface{}, error) {
	template := params["template"].(string)

	// Replace {input} with the actual input
	inputStr := fmt.Sprintf("%v", input)
	result := strings.ReplaceAll(template, "{input}", inputStr)

	// Support additional replacements if input is a map
	if m, ok := input.(map[string]interface{}); ok {
		for key, value := range m {
			placeholder := "{" + key + "}"
			valueStr := fmt.Sprintf("%v", value)
			result = strings.ReplaceAll(result, placeholder, valueStr)
		}
	}

	return result, nil
}

// DataMerger merges multiple data sources
type DataMerger struct{}

func (dm *DataMerger) Name() string        { return "merge_data" }
func (dm *DataMerger) Description() string { return "Merge multiple data sources" }

func (dm *DataMerger) ValidateParams(params map[string]interface{}) error {
	return nil
}

func (dm *DataMerger) Transform(input interface{}, params map[string]interface{}) (interface{}, error) {
	// For arrays, concatenate
	if v := reflect.ValueOf(input); v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		return input, nil // Pass through - merging happens at pipeline level
	}

	// For maps, merge with additional data
	if m, ok := input.(map[string]interface{}); ok {
		result := make(map[string]interface{})

		// Copy original data
		for k, v := range m {
			result[k] = v
		}

		// Add additional fields from params
		if additional, ok := params["additional"].(map[string]interface{}); ok {
			for k, v := range additional {
				result[k] = v
			}
		}

		return result, nil
	}

	return input, nil
}

// Deduplicator removes duplicate entries
type Deduplicator struct{}

func (dd *Deduplicator) Name() string        { return "deduplicate" }
func (dd *Deduplicator) Description() string { return "Remove duplicate entries from array" }

func (dd *Deduplicator) ValidateParams(params map[string]interface{}) error {
	return nil
}

func (dd *Deduplicator) Transform(input interface{}, params map[string]interface{}) (interface{}, error) {
	v := reflect.ValueOf(input)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("input must be array or slice, got %T", input)
	}

	seen := make(map[string]bool)
	var result []interface{}
	field, _ := params["field"].(string)

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()

		var key string
		if field != "" {
			if m, ok := item.(map[string]interface{}); ok {
				if val, exists := m[field]; exists {
					key = fmt.Sprintf("%v", val)
				}
			}
		} else {
			key = fmt.Sprintf("%v", item)
		}

		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}

	return result, nil
}

// DataSorter sorts arrays by fields
type DataSorter struct{}

func (ds *DataSorter) Name() string        { return "sort_data" }
func (ds *DataSorter) Description() string { return "Sort array by field or value" }

func (ds *DataSorter) ValidateParams(params map[string]interface{}) error {
	return nil
}

func (ds *DataSorter) Transform(input interface{}, params map[string]interface{}) (interface{}, error) {
	v := reflect.ValueOf(input)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("input must be array or slice, got %T", input)
	}

	// Convert to []interface{} for sorting
	items := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		items[i] = v.Index(i).Interface()
	}

	field, _ := params["field"].(string)
	order, _ := params["order"].(string)
	if order == "" {
		order = "asc"
	}

	sort.Slice(items, func(i, j int) bool {
		var valI, valJ interface{}

		if field != "" {
			valI = ds.extractValue(items[i], field)
			valJ = ds.extractValue(items[j], field)
		} else {
			valI = items[i]
			valJ = items[j]
		}

		result := ds.compareValues(valI, valJ)
		if order == "desc" {
			return result > 0
		}
		return result < 0
	})

	return items, nil
}

func (ds *DataSorter) extractValue(item interface{}, field string) interface{} {
	if m, ok := item.(map[string]interface{}); ok {
		if val, exists := m[field]; exists {
			return val
		}
	}
	return item
}

func (ds *DataSorter) compareValues(a, b interface{}) int {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	// Try numeric comparison first
	if aNum, errA := strconv.ParseFloat(aStr, 64); errA == nil {
		if bNum, errB := strconv.ParseFloat(bStr, 64); errB == nil {
			if aNum < bNum {
				return -1
			} else if aNum > bNum {
				return 1
			}
			return 0
		}
	}

	// Fall back to string comparison
	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

// RegexExtractor extracts data using regex groups
type RegexExtractor struct{}

func (re *RegexExtractor) Name() string        { return "regex_extract" }
func (re *RegexExtractor) Description() string { return "Extract data using regex capture groups" }

func (re *RegexExtractor) ValidateParams(params map[string]interface{}) error {
	if _, ok := params["pattern"]; !ok {
		return fmt.Errorf("pattern parameter is required")
	}
	return nil
}

func (re *RegexExtractor) Transform(input interface{}, params map[string]interface{}) (interface{}, error) {
	inputStr, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("input must be string, got %T", input)
	}

	pattern := params["pattern"].(string)
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	matches := regex.FindAllStringSubmatch(inputStr, -1)
	if len(matches) == 0 {
		return []string{}, nil
	}

	mode, _ := params["mode"].(string)
	if mode == "first" && len(matches) > 0 && len(matches[0]) > 1 {
		return matches[0][1], nil // Return first capture group of first match
	}

	var result []interface{}
	for _, match := range matches {
		if len(match) > 1 {
			if len(match) == 2 {
				result = append(result, match[1]) // Single capture group
			} else {
				result = append(result, match[1:]) // Multiple capture groups as array
			}
		}
	}

	return result, nil
}

// StringProcessor performs string operations
type StringProcessor struct{}

func (sp *StringProcessor) Name() string        { return "string_process" }
func (sp *StringProcessor) Description() string { return "Process strings (trim, case, replace)" }

func (sp *StringProcessor) ValidateParams(params map[string]interface{}) error {
	if _, ok := params["operation"]; !ok {
		return fmt.Errorf("operation parameter is required")
	}
	return nil
}

func (sp *StringProcessor) Transform(input interface{}, params map[string]interface{}) (interface{}, error) {
	inputStr, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("input must be string, got %T", input)
	}

	operation := params["operation"].(string)

	switch operation {
	case "trim":
		return strings.TrimSpace(inputStr), nil
	case "lower":
		return strings.ToLower(inputStr), nil
	case "upper":
		return strings.ToUpper(inputStr), nil
	case "title":
		return strings.Title(inputStr), nil
	case "replace":
		old, _ := params["old"].(string)
		new, _ := params["new"].(string)
		return strings.ReplaceAll(inputStr, old, new), nil
	case "split":
		delimiter, _ := params["delimiter"].(string)
		if delimiter == "" {
			delimiter = "\n"
		}
		parts := strings.Split(inputStr, delimiter)
		result := make([]interface{}, len(parts))
		for i, part := range parts {
			result[i] = part
		}
		return result, nil
	case "length":
		return len(inputStr), nil
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}
