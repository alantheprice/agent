package generic

import (
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TemplateFunction represents a built-in template function
type TemplateFunction func(args []interface{}) (interface{}, error)

// TemplateEngine handles advanced template rendering with dot notation and functions
type TemplateEngine struct {
	logger    *slog.Logger
	functions map[string]TemplateFunction
}

// NewTemplateEngine creates a new template engine with built-in functions
func NewTemplateEngine(logger *slog.Logger) *TemplateEngine {
	te := &TemplateEngine{
		logger:    logger,
		functions: make(map[string]TemplateFunction),
	}

	// Register built-in functions
	te.registerBuiltinFunctions()

	return te
}

// RenderTemplate renders a template with enhanced context access
func (te *TemplateEngine) RenderTemplate(template string, stepResults map[string]*StepResult, execCtx *ExecutionContext) (string, error) {
	rendered := template

	// Find all template expressions: {expression}
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		fullMatch := match[0]                     // {expression}
		expression := strings.TrimSpace(match[1]) // expression

		// Resolve the expression
		value, err := te.resolveExpression(expression, stepResults, execCtx)
		if err != nil {
			te.logger.Error("Failed to resolve template expression", "expression", expression, "error", err)
			continue // Leave unresolved expressions as-is
		}

		// Convert to string
		valueStr := te.formatValue(value)

		te.logger.Debug("Template substitution successful",
			"expression", expression,
			"value_type", fmt.Sprintf("%T", value),
			"value_length", len(valueStr),
			"sample_value", func() string {
				if len(valueStr) > 100 {
					return valueStr[:100] + "..."
				}
				return valueStr
			}())

		// Replace in template
		rendered = strings.ReplaceAll(rendered, fullMatch, valueStr)
	}

	return rendered, nil
}

// resolveExpression resolves a template expression to a value
func (te *TemplateEngine) resolveExpression(expression string, stepResults map[string]*StepResult, execCtx *ExecutionContext) (interface{}, error) {
	// Check if it's a function call: function(args...)
	if strings.Contains(expression, "(") && strings.HasSuffix(expression, ")") {
		return te.resolveFunction(expression, stepResults, execCtx)
	}

	// Check if it's a dot notation path: step.field.subfield
	if strings.Contains(expression, ".") {
		return te.resolveDotNotation(expression, stepResults, execCtx)
	}

	// Check if it's an array access: step[0] or step[key]
	if strings.Contains(expression, "[") && strings.Contains(expression, "]") {
		return te.resolveArrayAccess(expression, stepResults, execCtx)
	}

	// Simple step name reference
	return te.resolveSimpleReference(expression, stepResults, execCtx)
}

// resolveDotNotation resolves dot notation paths like "step.field.subfield"
func (te *TemplateEngine) resolveDotNotation(path string, stepResults map[string]*StepResult, execCtx *ExecutionContext) (interface{}, error) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	// Get the root value
	rootValue, err := te.resolveSimpleReference(parts[0], stepResults, execCtx)
	if err != nil {
		return nil, err
	}

	// Navigate through the path
	current := rootValue
	for i := 1; i < len(parts); i++ {
		current, err = te.getField(current, parts[i])
		if err != nil {
			return nil, fmt.Errorf("failed to access field '%s' in path '%s': %w", parts[i], path, err)
		}
	}

	return current, nil
}

// resolveArrayAccess resolves array access like "step[0]" or "step[key]"
func (te *TemplateEngine) resolveArrayAccess(expression string, stepResults map[string]*StepResult, execCtx *ExecutionContext) (interface{}, error) {
	// Parse the expression: base[index]
	re := regexp.MustCompile(`^([^[]+)\[([^\]]*)\]$`)
	matches := re.FindStringSubmatch(expression)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid array access syntax: %s", expression)
	}

	basePath := matches[1]
	indexStr := matches[2]

	// Get the base value (might be dot notation)
	var baseValue interface{}
	var err error

	if strings.Contains(basePath, ".") {
		baseValue, err = te.resolveDotNotation(basePath, stepResults, execCtx)
	} else {
		baseValue, err = te.resolveSimpleReference(basePath, stepResults, execCtx)
	}

	if err != nil {
		return nil, err
	}

	// Handle wildcard access [*]
	if indexStr == "*" {
		return te.handleWildcardAccess(baseValue)
	}

	// Handle slice access [1:3]
	if strings.Contains(indexStr, ":") {
		return te.handleSliceAccess(baseValue, indexStr)
	}

	// Handle single index access
	return te.getArrayElement(baseValue, indexStr)
}

// resolveFunction resolves function calls like "len(step.items)"
func (te *TemplateEngine) resolveFunction(expression string, stepResults map[string]*StepResult, execCtx *ExecutionContext) (interface{}, error) {
	// Parse function call: function(arg1, arg2, ...)
	re := regexp.MustCompile(`^(\w+)\((.*)\)$`)
	matches := re.FindStringSubmatch(expression)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid function syntax: %s", expression)
	}

	funcName := matches[1]
	argsStr := strings.TrimSpace(matches[2])

	// Get the function
	fn, exists := te.functions[funcName]
	if !exists {
		return nil, fmt.Errorf("unknown function: %s", funcName)
	}

	// Parse arguments
	var args []interface{}
	if argsStr != "" {
		argParts := te.parseArguments(argsStr)
		for _, argStr := range argParts {
			argStr = strings.TrimSpace(argStr)

			// Resolve argument (could be another expression)
			argValue, err := te.resolveExpression(argStr, stepResults, execCtx)
			if err != nil {
				// Try as literal string if expression resolution fails
				argValue = te.parseLiteral(argStr)
			}
			args = append(args, argValue)
		}
	}

	// Call the function
	return fn(args)
}

// resolveSimpleReference resolves simple step names or context keys
func (te *TemplateEngine) resolveSimpleReference(name string, stepResults map[string]*StepResult, execCtx *ExecutionContext) (interface{}, error) {
	// Debug logging for step resolution
	te.logger.Debug("Resolving simple reference",
		"name", name,
		"available_steps", func() []string {
			var steps []string
			for stepName := range stepResults {
				steps = append(steps, stepName)
			}
			return steps
		}())

	// Check step results first
	if result, exists := stepResults[name]; exists && result.Success && result.Output != nil {
		te.logger.Debug("Found step result",
			"name", name,
			"output_type", fmt.Sprintf("%T", result.Output),
			"output_preview", func() string {
				if str, ok := result.Output.(string); ok {
					return str[:min(100, len(str))]
				}
				if m, ok := result.Output.(map[string]interface{}); ok {
					return fmt.Sprintf("map with keys: %v", func() []string {
						var keys []string
						for k := range m {
							keys = append(keys, k)
						}
						return keys
					}())
				}
				return fmt.Sprintf("%v", result.Output)
			}())

		// For shell command results, we need to return the whole result as a structured object
		// so that dot notation can access .output, .command, .success fields
		if resultMap, ok := result.Output.(map[string]interface{}); ok {
			// Return the whole result map to enable dot notation access
			return resultMap, nil
		}
		return result.Output, nil
	}

	// Check execution context data
	if value, exists := execCtx.Data[name]; exists {
		return value, nil
	}

	te.logger.Debug("Reference not found", "name", name)
	return nil, fmt.Errorf("reference not found: %s", name)
}

// getField gets a field from an object using reflection
func (te *TemplateEngine) getField(obj interface{}, fieldName string) (interface{}, error) {
	if obj == nil {
		return nil, fmt.Errorf("cannot access field '%s' on nil object", fieldName)
	}

	// Handle map access
	if m, ok := obj.(map[string]interface{}); ok {
		if value, exists := m[fieldName]; exists {
			return value, nil
		}
		return nil, fmt.Errorf("field '%s' not found in map", fieldName)
	}

	// Handle struct access using reflection
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("cannot access field '%s' on non-struct type %T", fieldName, obj)
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		// Try case-insensitive search
		for i := 0; i < v.NumField(); i++ {
			if strings.EqualFold(v.Type().Field(i).Name, fieldName) {
				field = v.Field(i)
				break
			}
		}
	}

	if !field.IsValid() {
		return nil, fmt.Errorf("field '%s' not found in struct %T", fieldName, obj)
	}

	return field.Interface(), nil
}

// getArrayElement gets an element from an array or slice
func (te *TemplateEngine) getArrayElement(obj interface{}, indexStr string) (interface{}, error) {
	if obj == nil {
		return nil, fmt.Errorf("cannot index nil object")
	}

	v := reflect.ValueOf(obj)

	// Handle slices and arrays
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			return nil, fmt.Errorf("invalid array index: %s", indexStr)
		}

		if index < 0 || index >= v.Len() {
			return nil, fmt.Errorf("array index %d out of bounds (length %d)", index, v.Len())
		}

		return v.Index(index).Interface(), nil
	}

	// Handle maps
	if v.Kind() == reflect.Map {
		key := reflect.ValueOf(indexStr)
		value := v.MapIndex(key)
		if !value.IsValid() {
			return nil, fmt.Errorf("key '%s' not found in map", indexStr)
		}
		return value.Interface(), nil
	}

	return nil, fmt.Errorf("cannot index type %T", obj)
}

// handleWildcardAccess handles wildcard array access [*]
func (te *TemplateEngine) handleWildcardAccess(obj interface{}) (interface{}, error) {
	if obj == nil {
		return nil, fmt.Errorf("cannot perform wildcard access on nil object")
	}

	v := reflect.ValueOf(obj)

	// For arrays/slices, return all elements
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		result := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = v.Index(i).Interface()
		}
		return result, nil
	}

	// For maps, return all values
	if v.Kind() == reflect.Map {
		result := make([]interface{}, 0, v.Len())
		for _, key := range v.MapKeys() {
			result = append(result, v.MapIndex(key).Interface())
		}
		return result, nil
	}

	return nil, fmt.Errorf("wildcard access not supported for type %T", obj)
}

// handleSliceAccess handles slice access like [1:3]
func (te *TemplateEngine) handleSliceAccess(obj interface{}, sliceStr string) (interface{}, error) {
	parts := strings.Split(sliceStr, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid slice syntax: %s", sliceStr)
	}

	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("slice access not supported for type %T", obj)
	}

	// Parse start and end indices
	var start, end int
	var err error

	if parts[0] == "" {
		start = 0
	} else {
		start, err = strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid start index: %s", parts[0])
		}
	}

	if parts[1] == "" {
		end = v.Len()
	} else {
		end, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid end index: %s", parts[1])
		}
	}

	// Validate bounds
	if start < 0 || start > v.Len() {
		return nil, fmt.Errorf("start index %d out of bounds", start)
	}
	if end < 0 || end > v.Len() {
		return nil, fmt.Errorf("end index %d out of bounds", end)
	}
	if start > end {
		return nil, fmt.Errorf("start index %d greater than end index %d", start, end)
	}

	// Create slice
	result := make([]interface{}, end-start)
	for i := start; i < end; i++ {
		result[i-start] = v.Index(i).Interface()
	}

	return result, nil
}

// parseArguments parses function arguments, handling nested expressions
func (te *TemplateEngine) parseArguments(argsStr string) []string {
	var args []string
	var current strings.Builder
	var parenCount int
	var inQuotes bool
	var quoteChar rune

	for _, r := range argsStr {
		switch r {
		case '"', '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = r
			} else if r == quoteChar {
				inQuotes = false
			}
			current.WriteRune(r)
		case '(':
			if !inQuotes {
				parenCount++
			}
			current.WriteRune(r)
		case ')':
			if !inQuotes {
				parenCount--
			}
			current.WriteRune(r)
		case ',':
			if !inQuotes && parenCount == 0 {
				args = append(args, current.String())
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// parseLiteral parses a literal value (string, number, boolean)
func (te *TemplateEngine) parseLiteral(str string) interface{} {
	str = strings.TrimSpace(str)

	// String literal
	if (strings.HasPrefix(str, "'") && strings.HasSuffix(str, "'")) ||
		(strings.HasPrefix(str, "\"") && strings.HasSuffix(str, "\"")) {
		return str[1 : len(str)-1]
	}

	// Boolean literal
	if str == "true" {
		return true
	}
	if str == "false" {
		return false
	}

	// Number literal
	if num, err := strconv.ParseFloat(str, 64); err == nil {
		if float64(int(num)) == num {
			return int(num)
		}
		return num
	}

	// Return as string if nothing else matches
	return str
}

// formatValue formats a value for string substitution
func (te *TemplateEngine) formatValue(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case []interface{}:
		// For arrays, join with newlines
		var parts []string
		for _, item := range v {
			parts = append(parts, te.formatValue(item))
		}
		return strings.Join(parts, "\n")
	case map[string]interface{}:
		// For maps, format as key=value pairs
		var parts []string
		for key, val := range v {
			parts = append(parts, fmt.Sprintf("%s=%s", key, te.formatValue(val)))
		}
		return strings.Join(parts, "\n")
	default:
		return fmt.Sprintf("%v", value)
	}
}

// registerBuiltinFunctions registers all built-in template functions
func (te *TemplateEngine) registerBuiltinFunctions() {
	te.functions["len"] = te.lenFunction
	te.functions["join"] = te.joinFunction
	te.functions["filter"] = te.filterFunction
	te.functions["map"] = te.mapFunction
	te.functions["first"] = te.firstFunction
	te.functions["last"] = te.lastFunction
	te.functions["contains"] = te.containsFunction
	te.functions["split"] = te.splitFunction
	te.functions["add"] = te.addFunction
	te.functions["subtract"] = te.subtractFunction
	te.functions["multiply"] = te.multiplyFunction
	te.functions["divide"] = te.divideFunction
	te.functions["timestamp"] = te.timestampFunction
}

// Built-in template functions implementation

// Math functions
func (te *TemplateEngine) addFunction(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("add() expects 2 arguments, got %d", len(args))
	}

	a, err := te.toFloat64(args[0])
	if err != nil {
		return nil, fmt.Errorf("first argument to add(): %w", err)
	}

	b, err := te.toFloat64(args[1])
	if err != nil {
		return nil, fmt.Errorf("second argument to add(): %w", err)
	}

	result := a + b
	if a == float64(int(a)) && b == float64(int(b)) {
		return int(result), nil
	}
	return result, nil
}

func (te *TemplateEngine) subtractFunction(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("subtract() expects 2 arguments, got %d", len(args))
	}

	a, err := te.toFloat64(args[0])
	if err != nil {
		return nil, fmt.Errorf("first argument to subtract(): %w", err)
	}

	b, err := te.toFloat64(args[1])
	if err != nil {
		return nil, fmt.Errorf("second argument to subtract(): %w", err)
	}

	result := a - b
	if a == float64(int(a)) && b == float64(int(b)) {
		return int(result), nil
	}
	return result, nil
}

func (te *TemplateEngine) multiplyFunction(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("multiply() expects 2 arguments, got %d", len(args))
	}

	a, err := te.toFloat64(args[0])
	if err != nil {
		return nil, fmt.Errorf("first argument to multiply(): %w", err)
	}

	b, err := te.toFloat64(args[1])
	if err != nil {
		return nil, fmt.Errorf("second argument to multiply(): %w", err)
	}

	result := a * b
	if a == float64(int(a)) && b == float64(int(b)) {
		return int(result), nil
	}
	return result, nil
}

func (te *TemplateEngine) divideFunction(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("divide() expects 2 arguments, got %d", len(args))
	}

	a, err := te.toFloat64(args[0])
	if err != nil {
		return nil, fmt.Errorf("first argument to divide(): %w", err)
	}

	b, err := te.toFloat64(args[1])
	if err != nil {
		return nil, fmt.Errorf("second argument to divide(): %w", err)
	}

	if b == 0 {
		return nil, fmt.Errorf("division by zero")
	}

	return a / b, nil
}

func (te *TemplateEngine) toFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to number", val)
	}
}

func (te *TemplateEngine) lenFunction(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("len() expects 1 argument, got %d", len(args))
	}

	v := reflect.ValueOf(args[0])
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return v.Len(), nil
	default:
		return nil, fmt.Errorf("len() not supported for type %T", args[0])
	}
}

func (te *TemplateEngine) joinFunction(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("join() expects 2 arguments, got %d", len(args))
	}

	// First argument should be array/slice
	v := reflect.ValueOf(args[0])
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("join() first argument must be array or slice, got %T", args[0])
	}

	// Second argument should be delimiter string
	delimiter, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("join() second argument must be string, got %T", args[1])
	}

	// Convert array elements to strings and join
	var parts []string
	for i := 0; i < v.Len(); i++ {
		parts = append(parts, te.formatValue(v.Index(i).Interface()))
	}

	return strings.Join(parts, delimiter), nil
}

func (te *TemplateEngine) filterFunction(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("filter() expects 2 arguments, got %d", len(args))
	}

	// First argument should be array/slice
	v := reflect.ValueOf(args[0])
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("filter() first argument must be array or slice, got %T", args[0])
	}

	// Second argument is filter predicate (simplified - just check if field exists)
	predicate := fmt.Sprintf("%v", args[1])

	var result []interface{}
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		if te.matchesPredicate(item, predicate) {
			result = append(result, item)
		}
	}

	return result, nil
}

func (te *TemplateEngine) mapFunction(args []interface{}) (interface{}, error) {
	// Simplified map function - just returns the array as-is for now
	if len(args) != 1 {
		return nil, fmt.Errorf("map() expects 1 argument, got %d", len(args))
	}
	return args[0], nil
}

func (te *TemplateEngine) firstFunction(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("first() expects 1 argument, got %d", len(args))
	}

	v := reflect.ValueOf(args[0])
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("first() argument must be array or slice, got %T", args[0])
	}

	if v.Len() == 0 {
		return nil, nil
	}

	return v.Index(0).Interface(), nil
}

func (te *TemplateEngine) lastFunction(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("last() expects 1 argument, got %d", len(args))
	}

	v := reflect.ValueOf(args[0])
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("last() argument must be array or slice, got %T", args[0])
	}

	if v.Len() == 0 {
		return nil, nil
	}

	return v.Index(v.Len() - 1).Interface(), nil
}

func (te *TemplateEngine) containsFunction(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("contains() expects 2 arguments, got %d", len(args))
	}

	haystack := fmt.Sprintf("%v", args[0])
	needle := fmt.Sprintf("%v", args[1])

	return strings.Contains(haystack, needle), nil
}

func (te *TemplateEngine) splitFunction(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("split() expects 2 arguments, got %d", len(args))
	}

	str, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("split() first argument must be string, got %T", args[0])
	}

	delimiter, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("split() second argument must be string, got %T", args[1])
	}

	parts := strings.Split(str, delimiter)
	result := make([]interface{}, len(parts))
	for i, part := range parts {
		result[i] = part
	}

	return result, nil
}

// matchesPredicate is a simplified predicate matcher
func (te *TemplateEngine) matchesPredicate(item interface{}, predicate string) bool {
	// Simplified implementation - just check if the item contains the predicate string
	itemStr := te.formatValue(item)
	return strings.Contains(itemStr, predicate)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// timestampFunction generates a timestamp
func (te *TemplateEngine) timestampFunction(args []interface{}) (interface{}, error) {
	// Optional format argument, defaults to RFC3339
	format := "2006-01-02 15:04:05"

	if len(args) > 0 {
		if formatStr, ok := args[0].(string); ok {
			format = formatStr
		}
	}

	return time.Now().Format(format), nil
}
