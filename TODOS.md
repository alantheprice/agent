# Implementation Plan: Enhanced Context Flow System

## Current State Analysis

### Working Features ✅
- Basic string template substitution (`{step_name}`)
- Dependency-based workflow execution
- Global context storage in `ExecutionContext`
- Multi-LLM provider support
- Tool registry with built-in tools
- JSON-driven configuration

### Current Limitations 🔴
- **Limited Data Access**: Only full step output, no structured field access
- **No Data Transformation**: Raw data passes directly between steps
- **Poor Context Management**: No scoping, inheritance, or conditional flow
- **Analysis Chain Gaps**: No built-in patterns for multi-stage reasoning
- **Template Engine Constraints**: Basic string replacement only

## Phase 1: Enhanced Template Engine 🎯

**Goal**: Upgrade template system to support structured data access and built-in functions

### Task 1.1: Dot Notation Access
**Priority**: High  
**Estimated Effort**: 4-6 hours

**Implementation**:
```go
// pkg/generic/template_engine.go (new file)
type TemplateEngine struct {
    resolver *DataResolver
    functions map[string]TemplateFunction
}

// Support: {step.field.subfield}
func (te *TemplateEngine) ResolveDotNotation(path string, context map[string]interface{}) (interface{}, error)
```

**Changes Required**:
- [ ] Create new `template_engine.go` file
- [ ] Implement dot notation parser
- [ ] Add path resolution for nested data structures
- [ ] Update `renderTemplate` in `workflow_engine.go` to use new engine
- [ ] Add comprehensive tests

### Task 1.2: Array Access & Indexing
**Priority**: High  
**Estimated Effort**: 3-4 hours

**Implementation**:
```go
// Support: {step.items[0]}, {step.files[*].name}
func (te *TemplateEngine) ResolveArrayAccess(expression string, context map[string]interface{}) (interface{}, error)
```

**Changes Required**:
- [ ] Add array index parsing (`[0]`, `[*]`)
- [ ] Implement wildcard array operations
- [ ] Handle out-of-bounds access gracefully
- [ ] Add slice notation support (`[1:3]`)

### Task 1.3: Built-in Template Functions
**Priority**: Medium  
**Estimated Effort**: 6-8 hours

**Implementation**:
```go
// Support: {len(step.items)}, {join(step.lines, '\n')}, {filter(step.items, 'active')}
type TemplateFunction func(args []interface{}) (interface{}, error)

var BuiltinFunctions = map[string]TemplateFunction{
    "len":    LengthFunction,
    "join":   JoinFunction,
    "filter": FilterFunction,
    "map":    MapFunction,
    "first":  FirstFunction,
    "last":   LastFunction,
}
```

**Functions to Implement**:
- [ ] `len()` - Get length of arrays/strings
- [ ] `join()` - Join array elements with delimiter
- [ ] `filter()` - Filter arrays by predicate
- [ ] `map()` - Transform array elements
- [ ] `first()` - Get first element
- [ ] `last()` - Get last element
- [ ] `contains()` - Check if array/string contains value
- [ ] `split()` - Split strings into arrays

### Task 1.4: Type-Safe Template Rendering
**Priority**: Medium  
**Estimated Effort**: 4-5 hours

**Changes Required**:
- [ ] Add type checking for template expressions
- [ ] Implement error handling for type mismatches
- [ ] Add template validation at configuration load time
- [ ] Create template debugging/preview mode

## Phase 2: Context Transformers 🔄

**Goal**: Add pre/post-processing capabilities for data transformation between steps

### Task 2.1: Transform Configuration Schema
**Priority**: High  
**Estimated Effort**: 3-4 hours

**JSON Schema Addition**:
```json
{
  "context_transforms": [
    {
      "source": "git_diff.output",
      "transform": "extract_lines",
      "params": {"pattern": "^\\+"},
      "store_as": "added_lines"
    },
    {
      "source": "code_review",
      "transform": "aggregate", 
      "params": {"operation": "count", "field": "issues"},
      "store_as": "issue_count"
    }
  ]
}
```

**Changes Required**:
- [ ] Update `config.go` with transform structs
- [ ] Add schema validation for transforms
- [ ] Create transform execution pipeline

### Task 2.2: Built-in Transformers
**Priority**: High  
**Estimated Effort**: 8-10 hours

**Transformers to Implement**:
```go
// pkg/generic/transformers/
type Transformer interface {
    Transform(input interface{}, params map[string]interface{}) (interface{}, error)
    ValidateParams(params map[string]interface{}) error
}

var BuiltinTransformers = map[string]Transformer{
    "extract_lines":  &LineExtractor{},
    "parse_json":     &JSONParser{},
    "aggregate":      &Aggregator{},
    "filter_data":    &DataFilter{},
    "format_text":    &TextFormatter{},
}
```

**Transformers to Build**:
- [ ] `extract_lines` - Extract lines matching patterns
- [ ] `parse_json` - Parse JSON from text
- [ ] `aggregate` - Count, sum, average operations
- [ ] `filter_data` - Filter data by conditions
- [ ] `format_text` - Format text with templates
- [ ] `merge_data` - Merge multiple data sources
- [ ] `deduplicate` - Remove duplicate entries
- [ ] `sort_data` - Sort arrays by fields

### Task 2.3: Transform Execution Pipeline
**Priority**: High  
**Estimated Effort**: 5-6 hours

**Implementation**:
```go
// pkg/generic/transform_pipeline.go (new file)
type TransformPipeline struct {
    transforms []Transform
    context    *ExecutionContext
}

func (tp *TransformPipeline) Execute(stepName string, stepResult *StepResult) error
```

**Changes Required**:
- [ ] Create transform execution pipeline
- [ ] Integrate with workflow engine
- [ ] Add transform error handling
- [ ] Support transform dependencies

## Phase 3: Analysis Chain Primitives 🧠

**Goal**: Built-in patterns for multi-stage reasoning and context accumulation

### Task 3.1: Analysis Chain Configuration
**Priority**: Medium  
**Estimated Effort**: 4-5 hours

**JSON Schema Addition**:
```json
{
  "analysis_chain": {
    "input": "code_changes",
    "stages": [
      {
        "type": "security_scan",
        "prompt_template": "security_analysis.md",
        "store_as": "security_issues"
      },
      {
        "type": "quality_analysis", 
        "depends_on": ["security_scan"],
        "store_as": "quality_metrics"
      },
      {
        "type": "synthesis",
        "inputs": ["security_issues", "quality_metrics"],
        "store_as": "final_analysis"
      }
    ]
  }
}
```

### Task 3.2: Analysis Stage Primitives
**Priority**: Medium  
**Estimated Effort**: 10-12 hours

**Built-in Analysis Types**:
- [ ] `security_scan` - Security vulnerability analysis
- [ ] `quality_analysis` - Code quality assessment
- [ ] `performance_analysis` - Performance bottleneck detection
- [ ] `synthesis` - Multi-input reasoning and conclusions
- [ ] `classification` - Categorize and label data
- [ ] `sentiment_analysis` - Analyze sentiment/tone
- [ ] `summarization` - Generate summaries
- [ ] `comparison` - Compare multiple inputs

### Task 3.3: Context Accumulation Strategies
**Priority**: Medium  
**Estimated Effort**: 6-7 hours

**Accumulation Patterns**:
```go
type AccumulationStrategy interface {
    Accumulate(previous interface{}, current interface{}) (interface{}, error)
}

var AccumulationStrategies = map[string]AccumulationStrategy{
    "append":     &AppendStrategy{},
    "merge":      &MergeStrategy{},
    "aggregate":  &AggregateStrategy{},
    "latest":     &LatestStrategy{},
}
```

## Phase 4: Advanced Context Management 🏗️

**Goal**: Context scoping, inheritance, and conditional flow control

### Task 4.1: Context Scoping System
**Priority**: Low  
**Estimated Effort**: 8-10 hours

**JSON Schema Addition**:
```json
{
  "context_scope": {
    "inherit": ["git_diff", "user_preferences"],
    "private": ["api_keys", "temp_data"],
    "export": ["final_analysis"],
    "readonly": ["system_info"]
  }
}
```

### Task 4.2: Conditional Context Flow
**Priority**: Low  
**Estimated Effort**: 6-8 hours

**Implementation**:
```json
{
  "context_conditions": [
    {
      "if": "{security_issues.critical_count} > 0",
      "then": {"include": ["detailed_security_report"]},
      "else": {"exclude": ["security_details"]}
    }
  ]
}
```

### Task 4.3: Context Validation & Type System
**Priority**: Low  
**Estimated Effort**: 8-10 hours

**Features**:
- [ ] Context type definitions
- [ ] Runtime validation
- [ ] Type conversion/coercion
- [ ] Context debugging tools

## Implementation Schedule

### Sprint 1 (Week 1-2): Enhanced Template Engine
- [x] ~~Document architecture and create todos~~
- [ ] Task 1.1: Dot notation access
- [ ] Task 1.2: Array access & indexing  
- [ ] Task 1.3: Built-in template functions
- [ ] Task 1.4: Type-safe rendering

### Sprint 2 (Week 3-4): Context Transformers
- [ ] Task 2.1: Transform configuration schema
- [ ] Task 2.2: Built-in transformers
- [ ] Task 2.3: Transform execution pipeline

### Sprint 3 (Week 5-6): Analysis Chain Primitives  
- [ ] Task 3.1: Analysis chain configuration
- [ ] Task 3.2: Analysis stage primitives
- [ ] Task 3.3: Context accumulation strategies

### Sprint 4 (Week 7-8): Advanced Context Management
- [ ] Task 4.1: Context scoping system
- [ ] Task 4.2: Conditional context flow
- [ ] Task 4.3: Context validation & type system

## Testing Strategy

### Unit Tests
- [ ] Template engine with complex data structures
- [ ] All built-in transformers
- [ ] Analysis chain execution
- [ ] Context scoping rules

### Integration Tests  
- [ ] End-to-end workflow with enhanced context
- [ ] Multi-step analysis chains
- [ ] Error handling and validation
- [ ] Performance with large context data

### Example Configurations
- [ ] Update git workflow to use enhanced context
- [ ] Create data analysis pipeline example
- [ ] Build complex web scraping workflow
- [ ] Multi-agent orchestration with context sharing

## Success Criteria

### Phase 1 Success ✅
- Template expressions like `{git_diff.files[0].name}` work correctly
- Built-in functions like `{len(issues)}` are supported
- Type validation catches errors at configuration time

### Phase 2 Success ✅  
- Data transforms between steps without custom code
- Built-in transformers handle common data operations
- Transform errors are handled gracefully

### Phase 3 Success ✅
- Analysis chains execute multi-stage reasoning
- Context accumulates properly across stages
- Built-in analysis patterns work out of the box

### Phase 4 Success ✅
- Context scoping prevents data leaks
- Conditional flow works based on context state
- Type system catches errors early

## Risk Assessment

### High Risk 🔴
- **Breaking Changes**: Template syntax changes may break existing configurations
- **Performance Impact**: Complex context operations may slow execution  
- **Complexity Creep**: Too many features may make system hard to use

### Medium Risk 🟡
- **Backward Compatibility**: Need migration path for existing configurations
- **Documentation**: Complex features need clear examples and docs
- **Testing Coverage**: Many edge cases to test

### Low Risk 🟢
- **Implementation Time**: Well-defined tasks with clear scope
- **Integration**: Changes are mostly additive to existing system

## Mitigation Strategies

- **Versioned Configuration Schema**: Support both old and new syntax
- **Performance Monitoring**: Add metrics for context operations
- **Gradual Rollout**: Release phases incrementally  
- **Extensive Examples**: Document all new features with working examples
- **Migration Guide**: Clear upgrade path for existing configurations