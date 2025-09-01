package generic

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"
)

// TransformPipeline handles execution of data transforms
type TransformPipeline struct {
	registry       *TransformRegistry
	templateEngine *TemplateEngine
	logger         *slog.Logger
}

// NewTransformPipeline creates a new transform pipeline
func NewTransformPipeline(registry *TransformRegistry, templateEngine *TemplateEngine, logger *slog.Logger) *TransformPipeline {
	return &TransformPipeline{
		registry:       registry,
		templateEngine: templateEngine,
		logger:         logger,
	}
}

// ExecutePreTransforms executes context transforms before step execution
func (tp *TransformPipeline) ExecutePreTransforms(step Step, stepResults map[string]*StepResult, execCtx *ExecutionContext) error {
	if len(step.ContextTransforms) == 0 {
		return nil
	}

	tp.logger.Debug("Executing pre-transforms", "step", step.Name, "transform_count", len(step.ContextTransforms))

	for i, transform := range step.ContextTransforms {
		err := tp.executeTransform(transform, stepResults, execCtx, fmt.Sprintf("%s_pre_%d", step.Name, i))
		if err != nil {
			return fmt.Errorf("pre-transform %d failed: %w", i, err)
		}
	}

	return nil
}

// ExecutePostTransforms executes transforms after step execution
func (tp *TransformPipeline) ExecutePostTransforms(step Step, stepResult *StepResult, stepResults map[string]*StepResult, execCtx *ExecutionContext) error {
	if len(step.PostTransforms) == 0 {
		return nil
	}

	tp.logger.Debug("Executing post-transforms", "step", step.Name, "transform_count", len(step.PostTransforms))

	// Add the current step result to available results for post-transforms
	stepResults[step.Name] = stepResult

	for i, transform := range step.PostTransforms {
		err := tp.executeTransform(transform, stepResults, execCtx, fmt.Sprintf("%s_post_%d", step.Name, i))
		if err != nil {
			return fmt.Errorf("post-transform %d failed: %w", i, err)
		}
	}

	return nil
}

// executeTransform executes a single transform
func (tp *TransformPipeline) executeTransform(transform Transform, stepResults map[string]*StepResult, execCtx *ExecutionContext, transformID string) error {
	tp.logger.Debug("Executing transform",
		"id", transformID,
		"name", transform.Name,
		"source", transform.Source,
		"transform", transform.Transform,
		"store_as", transform.StoreAs)

	// Check condition if specified
	if transform.Condition != "" {
		conditionMet, err := tp.evaluateCondition(transform.Condition, stepResults, execCtx)
		if err != nil {
			return fmt.Errorf("failed to evaluate condition '%s': %w", transform.Condition, err)
		}
		if !conditionMet {
			tp.logger.Debug("Transform condition not met, skipping", "id", transformID, "condition", transform.Condition)
			return nil
		}
	}

	// Resolve source data using template engine
	sourceData, err := tp.resolveSource(transform.Source, stepResults, execCtx)
	if err != nil {
		return fmt.Errorf("failed to resolve source '%s': %w", transform.Source, err)
	}

	// Get transformer
	transformer, exists := tp.registry.GetTransformer(transform.Transform)
	if !exists {
		return fmt.Errorf("transformer '%s' not found", transform.Transform)
	}

	// Validate parameters
	err = transformer.ValidateParams(transform.Params)
	if err != nil {
		return fmt.Errorf("invalid parameters for transformer '%s': %w", transform.Transform, err)
	}

	// Execute transformation
	result, err := transformer.Transform(sourceData, transform.Params)
	if err != nil {
		return fmt.Errorf("transformation '%s' failed: %w", transform.Transform, err)
	}

	// Store result
	if transform.StoreAs != "" {
		execCtx.Data[transform.StoreAs] = result
		tp.logger.Debug("Transform result stored",
			"id", transformID,
			"store_as", transform.StoreAs,
			"result_type", fmt.Sprintf("%T", result))
	}

	return nil
}

// resolveSource resolves source data using template expressions
func (tp *TransformPipeline) resolveSource(source string, stepResults map[string]*StepResult, execCtx *ExecutionContext) (interface{}, error) {
	// Use template engine to resolve the source expression
	sourceTemplate := "{" + source + "}"
	resolved, err := tp.templateEngine.RenderTemplate(sourceTemplate, stepResults, execCtx)
	if err != nil {
		return nil, err
	}

	// If the resolved template equals the original template, it means the expression wasn't found
	// In this case, try direct resolution
	if resolved == sourceTemplate {
		// Try to resolve directly from step results or context
		if stepName, field, hasField := tp.parseFieldAccess(source); hasField {
			if result, exists := stepResults[stepName]; exists && result.Success {
				return tp.getFieldValue(result.Output, field)
			}
		}

		// Try direct context access
		if value, exists := execCtx.Data[source]; exists {
			return value, nil
		}

		return nil, fmt.Errorf("source '%s' not found", source)
	}

	return resolved, nil
}

// parseFieldAccess parses expressions like "step.field" into (step, field, true)
func (tp *TransformPipeline) parseFieldAccess(expr string) (string, string, bool) {
	parts := strings.Split(expr, ".")
	if len(parts) >= 2 {
		return parts[0], strings.Join(parts[1:], "."), true
	}
	return expr, "", false
}

// getFieldValue extracts field value from structured data
func (tp *TransformPipeline) getFieldValue(data interface{}, field string) (interface{}, error) {
	if m, ok := data.(map[string]interface{}); ok {
		if value, exists := m[field]; exists {
			return value, nil
		}
		return nil, fmt.Errorf("field '%s' not found in map", field)
	}

	// Use reflection for struct field access
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		fieldVal := v.FieldByName(field)
		if fieldVal.IsValid() {
			return fieldVal.Interface(), nil
		}
		return nil, fmt.Errorf("field '%s' not found in struct", field)
	}

	return nil, fmt.Errorf("cannot access field '%s' on type %T", field, data)
}

// evaluateCondition evaluates a condition expression
func (tp *TransformPipeline) evaluateCondition(condition string, stepResults map[string]*StepResult, execCtx *ExecutionContext) (bool, error) {
	// Simple condition evaluation - can be extended
	conditionTemplate := "{" + condition + "}"
	result, err := tp.templateEngine.RenderTemplate(conditionTemplate, stepResults, execCtx)
	if err != nil {
		return false, err
	}

	// Convert result to boolean
	switch strings.ToLower(result) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no", "":
		return false, nil
	default:
		// Check if it's a non-empty string or non-zero number
		if result != conditionTemplate && result != "" {
			return true, nil
		}
		return false, nil
	}
}
