package orchestration

import (
	"testing"

	"github.com/alantheprice/agent-template/pkg/orchestration/types"
)

func TestBuildStepDependencies(t *testing.T) {
	tests := []struct {
		name     string
		steps    []types.OrchestrationStep
		expected map[string][]string
	}{
		{
			name:     "no steps",
			steps:    []types.OrchestrationStep{},
			expected: map[string][]string{},
		},
		{
			name: "no dependencies",
			steps: []types.OrchestrationStep{
				{ID: "step1"},
				{ID: "step2"},
			},
			expected: map[string][]string{
				"step1": {},
				"step2": {},
			},
		},
		{
			name: "simple dependency",
			steps: []types.OrchestrationStep{
				{ID: "step1"},
				{ID: "step2", DependsOn: []string{"step1"}},
			},
			expected: map[string][]string{
				"step1": {},
				"step2": {"step1"},
			},
		},
		{
			name: "multiple dependencies",
			steps: []types.OrchestrationStep{
				{ID: "step1"},
				{ID: "step2"},
				{ID: "step3", DependsOn: []string{"step1", "step2"}},
			},
			expected: map[string][]string{
				"step1": {},
				"step2": {},
				"step3": {"step1", "step2"},
			},
		},
		{
			name: "chain of dependencies",
			steps: []types.OrchestrationStep{
				{ID: "step1"},
				{ID: "step2", DependsOn: []string{"step1"}},
				{ID: "step3", DependsOn: []string{"step2"}},
			},
			expected: map[string][]string{
				"step1": {},
				"step2": {"step1"},
				"step3": {"step2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildStepDependencies(tt.steps)

			// Check that the result has the right keys
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d entries, got %d", len(tt.expected), len(result))
			}

			// Check each dependency list
			for stepID, expectedDeps := range tt.expected {
				actualDeps, exists := result[stepID]
				if !exists {
					t.Errorf("Expected step '%s' to exist in result", stepID)
					continue
				}

				if len(actualDeps) != len(expectedDeps) {
					t.Errorf("Step '%s': expected %d dependencies, got %d", stepID, len(expectedDeps), len(actualDeps))
					continue
				}

				// Check each dependency
				for _, expectedDep := range expectedDeps {
					found := false
					for _, actualDep := range actualDeps {
						if actualDep == expectedDep {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Step '%s': expected dependency '%s' not found", stepID, expectedDep)
					}
				}
			}
		})
	}
}
