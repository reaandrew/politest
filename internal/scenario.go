package internal

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadScenarioWithExtends loads a scenario and recursively merges parent scenarios
func LoadScenarioWithExtends(absPath string) (*Scenario, error) {
	var s Scenario
	if err := LoadYAML(absPath, &s); err != nil {
		return nil, err
	}
	if s.Extends == "" {
		return &s, nil
	}
	base := filepath.Dir(absPath)
	parent := MustAbsJoin(base, s.Extends)

	ps, err := LoadScenarioWithExtends(parent)
	if err != nil {
		return nil, err
	}
	merged := MergeScenario(*ps, s) // child overrides parent
	return &merged, nil
}

// MergeScenario merges two scenarios with child overriding parent
func MergeScenario(a, b Scenario) Scenario {
	// simple field-wise merge: b overrides a; maps deep-merged
	out := a
	mergePolicyFields(&out, b)
	mergeSliceFields(&out, b)
	mergeMapFields(&out, b)
	mergeResourcePolicyFields(&out, b)
	mergeSimulationFields(&out, b)
	return out
}

// mergePolicyFields merges policy-related fields from b into out
func mergePolicyFields(out *Scenario, b Scenario) {
	if b.VarsFile != "" {
		out.VarsFile = b.VarsFile
	}
	if b.PolicyTemplate != "" {
		out.PolicyTemplate = b.PolicyTemplate
		out.PolicyJSON = "" // ensure mutual exclusivity
	}
	if b.PolicyJSON != "" {
		out.PolicyJSON = b.PolicyJSON
		out.PolicyTemplate = ""
	}
	if len(b.SCPPaths) > 0 {
		out.SCPPaths = b.SCPPaths
	}
}

// mergeSliceFields merges slice-based fields from b into out
func mergeSliceFields(out *Scenario, b Scenario) {
	if len(b.Context) > 0 {
		out.Context = b.Context
	}
	if len(b.Tests) > 0 {
		out.Tests = b.Tests
	}
}

// mergeMapFields merges map-based fields from b into out
func mergeMapFields(out *Scenario, b Scenario) {
	if out.Vars == nil {
		out.Vars = map[string]any{}
	}
	for k, v := range b.Vars {
		out.Vars[k] = v
	}
}

// mergeResourcePolicyFields merges resource policy fields from b into out
func mergeResourcePolicyFields(out *Scenario, b Scenario) {
	if b.ResourcePolicyTemplate != "" {
		out.ResourcePolicyTemplate = b.ResourcePolicyTemplate
		out.ResourcePolicyJSON = ""
	}
	if b.ResourcePolicyJSON != "" {
		out.ResourcePolicyJSON = b.ResourcePolicyJSON
		out.ResourcePolicyTemplate = ""
	}
}

// mergeSimulationFields merges simulation-related fields from b into out
func mergeSimulationFields(out *Scenario, b Scenario) {
	if b.CallerArn != "" {
		out.CallerArn = b.CallerArn
	}
	if b.ResourceOwner != "" {
		out.ResourceOwner = b.ResourceOwner
	}
	if b.ResourceHandlingOption != "" {
		out.ResourceHandlingOption = b.ResourceHandlingOption
	}
}

// LoadYAML loads and unmarshals a YAML file
func LoadYAML(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, v)
}
