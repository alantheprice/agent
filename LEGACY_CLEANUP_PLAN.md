# Legacy Cleanup Plan - Agent Template Project

## Overview

This document identifies all legacy components from the original `ledit` coding-specific agent that should be removed or modified to align with the new generic agent framework vision.

## Analysis Summary

The project currently contains:
- **Generic Agent Framework** (`pkg/generic/`, `cmd/generic/`, `examples/configs/`) - **KEEP**
- **Multi-Agent Orchestration** (`pkg/orchestration/`, `cmd/process.go`) - **KEEP** 
- **Legacy Ledit Commands** (`cmd/*.go` except `process.go`) - **REMOVE/MODIFY**
- **Legacy Packages** (most of `pkg/` except generic & orchestration) - **REMOVE**
- **Legacy Documentation** (most docs) - **REMOVE/REPLACE**

---

## 🗑️ FILES AND DIRECTORIES TO REMOVE

### Root Directory Files
```
❌ REMOVE:
- coverage.html (test coverage from legacy tests)
- run_adhoc_tests.sh (legacy test runner)
- test_e2e.sh (legacy e2e test script)
- test_run_output.txt (legacy test output)
- test_runner.py (Python test runner for legacy tests)
- AGENT_FLOW_TODOS (outdated planning document)
- prompt (appears to be a binary or temp file)
- generic-agent (compiled binary - should be in .gitignore)

✅ KEEP:
- main.go (entry point)
- go.mod, go.sum (dependencies)
- .gitignore, LICENSE (project files)
- PROJECT_STATE.md, E2E_TESTS_COMPLETED.md (documentation)
```

### CMD Directory - Legacy Commands
```
❌ REMOVE ENTIRE FILES:
- cmd/agent.go (legacy agent with coding-specific logic)
- cmd/code.go (coding-specific code generation)
- cmd/commit.go (git commit message generation)
- cmd/fix.go (code fixing functionality)
- cmd/question.go (workspace Q&A functionality)
- cmd/review_staged.go (code review functionality)
- cmd/log.go (change log functionality)
- cmd/rollback.go (rollback functionality)
- cmd/exec.go (command execution)
- cmd/ignore.go (ignore file management)
- cmd/init.go (project initialization)
- cmd/insights.go (project insights)
- cmd/pricing.go (cost calculation)
- cmd/ui.go (TUI interface)
- cmd/base.go (base command framework)
- cmd/process_state.go (process state management)
- cmd/process_validate.go (process validation)
- cmd/.ledit/ (configuration directory)

✅ KEEP:
- cmd/process.go (multi-agent orchestration - CORE to new vision)
- cmd/root.go (BUT NEEDS MAJOR MODIFICATION - see below)
- cmd/generic/ (generic agent binary)

⚠️  MODIFY:
- cmd/root.go (remove all legacy command registrations, keep only process)
```

### PKG Directory - Legacy Packages

```
❌ REMOVE ENTIRE DIRECTORIES:
- pkg/agent/ (legacy agent implementation with coding-specific logic)
- pkg/editor/ (code editing functionality)
- pkg/git/ (git operations)
- pkg/boundaries/ (domain boundaries for legacy architecture)
- pkg/adapters/ (legacy adapter pattern implementation)
- pkg/common/ (legacy utilities)
- pkg/filesystem/ (file system operations for code editing)
- pkg/index/ (code indexing functionality)
- pkg/llmclient/ (legacy LLM client)
- pkg/math/ (math utilities for similarity)
- pkg/parser/ (code parsing)
- pkg/performance/ (performance benchmarks)
- pkg/project/ (project goals functionality)
- pkg/prompts/ (legacy prompts)
- pkg/text/ (text processing utilities)
- pkg/utils/ (legacy utilities)
- pkg/workspaceinfo/ (workspace information for coding)

✅ KEEP:
- pkg/generic/ (core generic agent framework)
- pkg/orchestration/ (multi-agent orchestration system)
- pkg/interfaces/ (MAY NEED REVIEW - if generic enough)

❓ REVIEW NEEDED:
- pkg/interfaces/ (check if truly generic or coding-specific)
```

### Internal Directory
```
❌ REMOVE:
- internal/ (entire directory - appears to be legacy domain structure)
```

### Documentation
```
❌ REMOVE LEGACY DOCS:
- docs/GETTING_STARTED.md (ledit-specific getting started)
- docs/CHEATSHEET.md (ledit command cheatsheet)
- docs/EDITING_OPTIMIZATION.md (code editing optimization)
- docs/ROLLBACK_IMPLEMENTATION.md (rollback features)
- docs/TIPS_AND_TRICKS.md (ledit tips)
- docs/AGENT_CONTROL_FLOW.md (legacy agent flow)
- docs/AGENT_V2_TODO.md (legacy planning)
- docs/AGENT_WORKFLOW_PLAYBOOK.md (legacy workflows)

✅ KEEP:
- docs/MULTI_AGENT_ORCHESTRATION.md (core to new vision)
- docs/templatized_agent/ (appears to be generic agent documentation)

⚠️  NEEDS MAJOR UPDATES:
- docs/GETTING_STARTED.md (rewrite for generic agent framework)
```

---

## 📝 FILES THAT NEED MODIFICATION

### cmd/root.go
**Current Issues:**
- References to `ledit` branding throughout
- Registers 12 legacy commands (code, agent, commit, fix, etc.)
- Legacy help text and descriptions

**Required Changes:**
```go
// BEFORE (legacy):
Use: "ledit",
Short: "AI-powered code editor and orchestrator",
Long: `Ledit is a command-line tool that leverages Large Language Models...
Available commands:
  code     - Generate/edit code based on instructions
  agent    - AI agent mode (analyzes intent and decides actions)
  ...`

rootCmd.AddCommand(agentCmd)
rootCmd.AddCommand(codeCmd.GetCommand())
rootCmd.AddCommand(commitCmd)
// ... 10 more legacy commands

// AFTER (generic):
Use: "agent-template",
Short: "Generic AI Agent Framework",
Long: `A configurable agent template system for creating specialized AI agents...`

rootCmd.AddCommand(processCmd)  // Only keep process command
```

### go.mod
**Current Issues:**
- Module name still references `ledit`

**Required Changes:**
```go
// BEFORE:
module github.com/alantheprice/ledit

// AFTER:
module github.com/alantheprice/agent-template
```

### main.go
**Required Changes:**
- Update import paths when go.mod changes
- Ensure it only loads the generic framework

---

## 🔄 ARCHITECTURAL CHANGES NEEDED

### 1. Primary Binary Focus
**Current State**: Two binaries
- `main.go` → `ledit` (legacy)
- `cmd/generic/main.go` → `generic-agent` (new)

**Target State**: Single binary
- `main.go` → `agent-template` (generic framework only)
- Remove `cmd/generic/main.go` (consolidate into main)

### 2. Command Structure Simplification
**Current**: 13 commands
```
process, agent, code, commit, fix, question, review, log, rollback, exec, ignore, init, insights, pricing, ui
```

**Target**: 1-2 commands
```
process (multi-agent orchestration)
[optional: run (single agent execution)]
```

### 3. Package Dependencies Cleanup
**Current Dependencies to Remove:**
- All editor-related packages
- Git operation packages
- File system manipulation packages
- Code parsing and indexing
- Legacy prompt templates
- Workspace analysis tools

**Dependencies to Keep:**
- Multi-agent orchestration
- Generic agent framework
- LLM provider abstractions (if generic enough)
- Configuration management

---

## 📊 IMPACT ANALYSIS

### Size Reduction Estimate
- **Files to Remove**: ~80+ files
- **Directories to Remove**: ~15 directories
- **Code Reduction**: ~70-80% of current codebase
- **Command Reduction**: ~90% of current commands

### Breaking Changes
- Complete API change (no backward compatibility)
- All `ledit` commands will be removed
- Configuration format changes
- Import path changes

### Benefits
- Dramatically simplified codebase
- Clear separation of concerns
- Faster compilation and testing
- Easier to understand and contribute to
- Aligns with generic agent vision

---

## 🏗️ IMPLEMENTATION PLAN

### Phase 1: Preparation
1. **Backup Current State**: Tag current version as `v1.0.0-ledit-final`
2. **Update Documentation**: Create migration guide for existing users
3. **Test Coverage**: Ensure new e2e tests cover all kept functionality

### Phase 2: Core Cleanup
1. **Remove Legacy Commands**: Delete all cmd/*.go files except process.go and root.go
2. **Remove Legacy Packages**: Delete all obsolete pkg/ directories
3. **Simplify root.go**: Remove all command registrations except process
4. **Update go.mod**: Change module name to agent-template

### Phase 3: Consolidation  
1. **Merge Binaries**: Move cmd/generic/main.go logic into main.go
2. **Update Imports**: Fix all import paths after module rename
3. **Remove Internal**: Delete internal/ directory
4. **Clean Documentation**: Remove legacy docs, update kept docs

### Phase 4: Testing & Validation
1. **Build Testing**: Ensure clean compilation
2. **E2E Testing**: Run full e2e test suite
3. **Example Validation**: Test all example configurations
4. **Integration Testing**: Verify multi-agent orchestration works

### Phase 5: Finalization
1. **Documentation**: Update all documentation for new structure
2. **README**: Create new README focused on generic agent framework
3. **Examples**: Ensure all examples work with new structure
4. **Release**: Tag as `v2.0.0-generic-framework`

---

## ⚠️ RISKS AND CONSIDERATIONS

### Major Risks
1. **Complete Breaking Change**: No backward compatibility
2. **User Migration**: Existing ledit users will need to migrate
3. **Feature Loss**: All coding-specific functionality removed
4. **Integration Dependencies**: Other systems depending on ledit commands

### Mitigation Strategies  
1. **Clear Communication**: Announce breaking changes well in advance
2. **Migration Tooling**: Provide tools/docs for migrating to generic framework
3. **Versioning**: Use clear version tags to separate legacy from generic
4. **Parallel Support**: Consider maintaining legacy branch temporarily

### Success Criteria
- [x] Clean compilation with no errors
- [x] E2E tests pass with new structure  
- [x] Multi-agent orchestration works correctly
- [x] Example configurations execute successfully
- [x] Documentation accurately reflects new architecture
- [x] Significantly reduced codebase complexity

---

## 📋 EXECUTION CHECKLIST

### Before Starting
- [ ] Tag current state as backup
- [ ] Document current functionality for migration guide
- [ ] Ensure e2e tests are comprehensive

### File Removal Phase
- [ ] Remove legacy cmd/ files (keep only process.go, root.go)
- [ ] Remove legacy pkg/ directories (keep generic/, orchestration/)
- [ ] Remove internal/ directory
- [ ] Remove legacy documentation
- [ ] Remove legacy example files
- [ ] Remove root directory legacy files

### Modification Phase  
- [ ] Simplify cmd/root.go (remove legacy commands)
- [ ] Update go.mod module name
- [ ] Consolidate main.go and cmd/generic/main.go
- [ ] Fix all import statements
- [ ] Update remaining documentation

### Validation Phase
- [ ] Build compiles cleanly
- [ ] E2E tests pass
- [ ] Example configurations work
- [ ] Multi-agent orchestration functional
- [ ] No legacy references in codebase

### Finalization Phase
- [ ] Create new README
- [ ] Update all documentation
- [ ] Create migration guide
- [ ] Tag new version
- [ ] Update project descriptions

---

**This cleanup plan represents a complete transformation from a coding-specific agent to a generic agent template framework. The changes are extensive but necessary to achieve the project's new vision.**