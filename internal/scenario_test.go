package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeScenario(t *testing.T) {
	tests := []struct {
		name     string
		parent   Scenario
		child    Scenario
		expected Scenario
	}{
		{
			name: "child overrides parent vars",
			parent: Scenario{
				Vars: map[string]any{
					"bucket": "parent-bucket",
					"region": "us-east-1",
				},
			},
			child: Scenario{
				Vars: map[string]any{
					"bucket": "child-bucket",
				},
			},
			expected: Scenario{
				Vars: map[string]any{
					"bucket": "child-bucket", // overridden
					"region": "us-east-1",    // inherited
				},
			},
		},
		{
			name: "child overrides policy fields",
			parent: Scenario{
				PolicyJSON:     "parent.json",
				PolicyTemplate: "parent.tpl",
			},
			child: Scenario{
				PolicyJSON: "child.json",
			},
			expected: Scenario{
				PolicyJSON: "child.json",
			},
		},
		{
			name: "merge context arrays",
			parent: Scenario{
				Context: []ContextEntryYml{
					{ContextKeyName: "aws:userid", ContextKeyValues: []string{"parent"}, ContextKeyType: "string"},
				},
			},
			child: Scenario{
				Context: []ContextEntryYml{
					{ContextKeyName: "aws:username", ContextKeyValues: []string{"child"}, ContextKeyType: "string"},
				},
			},
			expected: Scenario{
				Context: []ContextEntryYml{
					{ContextKeyName: "aws:username", ContextKeyValues: []string{"child"}, ContextKeyType: "string"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeScenario(tt.parent, tt.child)

			// Check Vars
			if len(tt.expected.Vars) > 0 {
				for k, v := range tt.expected.Vars {
					if result.Vars[k] != v {
						t.Errorf("Vars[%s] = %v, want %v", k, result.Vars[k], v)
					}
				}
			}

			// Check PolicyJSON
			if tt.expected.PolicyJSON != "" && result.PolicyJSON != tt.expected.PolicyJSON {
				t.Errorf("PolicyJSON = %v, want %v", result.PolicyJSON, tt.expected.PolicyJSON)
			}

			// Check Context - slices are replaced, not merged
			if len(tt.expected.Context) > 0 {
				if len(result.Context) != len(tt.expected.Context) {
					t.Errorf("Context length = %v, want %v", len(result.Context), len(tt.expected.Context))
				}
			}
		})
	}
}

func TestLoadYAML(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("load valid yaml", func(t *testing.T) {
		file := filepath.Join(tmpDir, "test.yml")
		content := `
vars:
  bucket: test-bucket
tests:
  - action: s3:GetObject
    resource: "*"
`
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		var result Scenario
		err := LoadYAML(file, &result)
		if err != nil {
			t.Errorf("LoadYAML() error = %v", err)
		}
		if result.Vars["bucket"] != "test-bucket" {
			t.Errorf("LoadYAML() Vars[bucket] = %v, want test-bucket", result.Vars["bucket"])
		}
	})

	t.Run("file not found", func(t *testing.T) {
		var result Scenario
		err := LoadYAML(filepath.Join(tmpDir, "nonexistent.yml"), &result)
		if err == nil {
			t.Error("LoadYAML() expected error for nonexistent file, got nil")
		}
	})
}

func TestLoadScenarioWithExtends(t *testing.T) {
	tmpDir := t.TempDir()

	// Create parent scenario
	parentFile := filepath.Join(tmpDir, "parent.yml")
	parentContent := `
vars:
  bucket: parent-bucket
tests:
  - action: s3:GetObject
    resource: "*"
`
	if err := os.WriteFile(parentFile, []byte(parentContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create child scenario
	childFile := filepath.Join(tmpDir, "child.yml")
	childContent := `
extends: parent.yml
vars:
  bucket: child-bucket
  region: us-east-1
`
	if err := os.WriteFile(childFile, []byte(childContent), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadScenarioWithExtends(childFile)
	if err != nil {
		t.Fatalf("LoadScenarioWithExtends() error = %v", err)
	}

	// Check merged vars
	if result.Vars["bucket"] != "child-bucket" {
		t.Errorf("Vars[bucket] = %v, want child-bucket", result.Vars["bucket"])
	}

	if result.Vars["region"] != "us-east-1" {
		t.Errorf("Vars[region] = %v, want us-east-1", result.Vars["region"])
	}

	// Check inherited tests
	if len(result.Tests) == 0 || result.Tests[0].Action != "s3:GetObject" {
		t.Errorf("Tests not inherited correctly")
	}
}

func TestMergeScenarioComprehensive(t *testing.T) {
	t.Run("all fields populated", func(t *testing.T) {
		parent := Scenario{
			VarsFile:               "parent-vars.yml",
			PolicyJSON:             "parent-policy.json",
			PolicyTemplate:         "parent-policy.tpl",
			ResourcePolicyJSON:     "parent-resource.json",
			ResourcePolicyTemplate: "parent-resource.tpl",
			CallerArn:              "arn:aws:iam::123:user/parent",
			ResourceOwner:          "123456789012",
			ResourceHandlingOption: "EC2-VPC-InstanceStore",
			SCPPaths:               []string{"parent-scp1.json", "parent-scp2.json"},
			Context: []ContextEntryYml{
				{ContextKeyName: "aws:userid", ContextKeyValues: []string{"parent"}, ContextKeyType: "string"},
			},
			Tests: []TestCase{
				{Name: "parent-test", Action: "s3:GetObject"},
			},
			Vars: map[string]any{
				"bucket": "parent-bucket",
				"env":    "prod",
			},
		}

		child := Scenario{
			VarsFile:               "child-vars.yml",
			PolicyJSON:             "child-policy.json",
			PolicyTemplate:         "",
			ResourcePolicyJSON:     "",
			ResourcePolicyTemplate: "child-resource.tpl",
			CallerArn:              "arn:aws:iam::456:user/child",
			ResourceOwner:          "456789012345",
			ResourceHandlingOption: "EC2-Classic",
			SCPPaths:               []string{"child-scp.json"},
			Context: []ContextEntryYml{
				{ContextKeyName: "aws:username", ContextKeyValues: []string{"child"}, ContextKeyType: "string"},
			},
			Tests: []TestCase{
				{Name: "child-test", Action: "s3:PutObject"},
			},
			Vars: map[string]any{
				"bucket": "child-bucket",
				"region": "us-west-2",
			},
		}

		result := MergeScenario(parent, child)

		// String fields should be overridden by child
		if result.VarsFile != "child-vars.yml" {
			t.Errorf("VarsFile = %v, want child-vars.yml", result.VarsFile)
		}
		if result.PolicyJSON != "child-policy.json" {
			t.Errorf("PolicyJSON = %v, want child-policy.json", result.PolicyJSON)
		}
		if result.PolicyTemplate != "" {
			t.Errorf("PolicyTemplate = %v, want empty", result.PolicyTemplate)
		}
		if result.ResourcePolicyJSON != "" {
			t.Errorf("ResourcePolicyJSON = %v, want empty", result.ResourcePolicyJSON)
		}
		if result.ResourcePolicyTemplate != "child-resource.tpl" {
			t.Errorf("ResourcePolicyTemplate = %v, want child-resource.tpl", result.ResourcePolicyTemplate)
		}
		if result.CallerArn != "arn:aws:iam::456:user/child" {
			t.Errorf("CallerArn = %v, want child", result.CallerArn)
		}
		if result.ResourceOwner != "456789012345" {
			t.Errorf("ResourceOwner = %v, want child", result.ResourceOwner)
		}
		if result.ResourceHandlingOption != "EC2-Classic" {
			t.Errorf("ResourceHandlingOption = %v, want EC2-Classic", result.ResourceHandlingOption)
		}

		// Vars should be merged
		if result.Vars["bucket"] != "child-bucket" {
			t.Errorf("Vars[bucket] = %v, want child-bucket", result.Vars["bucket"])
		}
		if result.Vars["env"] != "prod" {
			t.Errorf("Vars[env] = %v, want prod", result.Vars["env"])
		}
		if result.Vars["region"] != "us-west-2" {
			t.Errorf("Vars[region] = %v, want us-west-2", result.Vars["region"])
		}

		// Slices should be replaced
		if len(result.SCPPaths) != 1 || result.SCPPaths[0] != "child-scp.json" {
			t.Errorf("SCPPaths = %v, want [child-scp.json]", result.SCPPaths)
		}
		if len(result.Tests) != 1 || result.Tests[0].Name != "child-test" {
			t.Errorf("Tests = %v, want [child-test]", result.Tests)
		}
	})

	t.Run("empty child keeps parent values", func(t *testing.T) {
		parent := Scenario{
			PolicyJSON: "parent.json",
			CallerArn:  "arn:aws:iam::123:user/parent",
		}
		child := Scenario{}

		result := MergeScenario(parent, child)

		// Empty child doesn't override parent - this is the correct behavior
		if result.PolicyJSON != "parent.json" {
			t.Errorf("PolicyJSON = %v, want parent.json", result.PolicyJSON)
		}
		if result.CallerArn != "arn:aws:iam::123:user/parent" {
			t.Errorf("CallerArn = %v, want parent", result.CallerArn)
		}
	})
}

func TestLoadScenarioWithExtendsMultipleLevels(t *testing.T) {
	tmpDir := t.TempDir()

	// Grandparent
	grandparentFile := filepath.Join(tmpDir, "grandparent.yml")
	grandparentContent := `
vars:
  level: grandparent
  bucket: grandparent-bucket
tests:
  - action: s3:GetObject
    resource: "*"
`
	if err := os.WriteFile(grandparentFile, []byte(grandparentContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Parent extends grandparent
	parentFile := filepath.Join(tmpDir, "parent.yml")
	parentContent := `
extends: grandparent.yml
vars:
  level: parent
  region: us-east-1
`
	if err := os.WriteFile(parentFile, []byte(parentContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Child extends parent
	childFile := filepath.Join(tmpDir, "child.yml")
	childContent := `
extends: parent.yml
vars:
  level: child
  env: prod
`
	if err := os.WriteFile(childFile, []byte(childContent), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadScenarioWithExtends(childFile)
	if err != nil {
		t.Fatalf("LoadScenarioWithExtends() error = %v", err)
	}

	// Check all vars are merged correctly
	if result.Vars["level"] != "child" {
		t.Errorf("Vars[level] = %v, want child", result.Vars["level"])
	}
	if result.Vars["bucket"] != "grandparent-bucket" {
		t.Errorf("Vars[bucket] = %v, want grandparent-bucket", result.Vars["bucket"])
	}
	if result.Vars["region"] != "us-east-1" {
		t.Errorf("Vars[region] = %v, want us-east-1", result.Vars["region"])
	}
	if result.Vars["env"] != "prod" {
		t.Errorf("Vars[env] = %v, want prod", result.Vars["env"])
	}

	// Tests should be inherited from grandparent
	if len(result.Tests) == 0 || result.Tests[0].Action != "s3:GetObject" {
		t.Errorf("Tests not inherited correctly: %v", result.Tests)
	}
}

func TestLoadScenarioWithExtendsError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a scenario that extends a non-existent file
	childFile := filepath.Join(tmpDir, "child.yml")
	childContent := `
extends: nonexistent.yml
vars:
  bucket: child-bucket
`
	if err := os.WriteFile(childFile, []byte(childContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadScenarioWithExtends(childFile)
	if err == nil {
		t.Error("LoadScenarioWithExtends() should return error when parent doesn't exist")
	}
}

func TestMergeScenarioWithTests(t *testing.T) {
	parent := Scenario{
		Tests: []TestCase{
			{Action: "s3:GetObject", Expect: "allowed"},
		},
	}
	child := Scenario{
		Tests: []TestCase{
			{Action: "s3:PutObject", Expect: "denied"},
		},
	}

	result := MergeScenario(parent, child)

	// Tests should be replaced, not merged
	if len(result.Tests) != 1 {
		t.Errorf("Expected 1 test, got %d", len(result.Tests))
	}
	if result.Tests[0].Action != "s3:PutObject" {
		t.Errorf("Expected child test, got %s", result.Tests[0].Action)
	}
}

func TestMergeScenarioWithCallerArn(t *testing.T) {
	parent := Scenario{
		CallerArn: "arn:aws:iam::123456789012:user/parent",
	}
	child := Scenario{
		CallerArn: "arn:aws:iam::123456789012:user/child",
	}

	result := MergeScenario(parent, child)

	if result.CallerArn != "arn:aws:iam::123456789012:user/child" {
		t.Errorf("Expected child CallerArn, got %s", result.CallerArn)
	}
}

func TestMergeScenarioWithResourceOwner(t *testing.T) {
	parent := Scenario{
		ResourceOwner: "123456789012",
	}
	child := Scenario{
		ResourceOwner: "987654321098",
	}

	result := MergeScenario(parent, child)

	if result.ResourceOwner != "987654321098" {
		t.Errorf("Expected child ResourceOwner, got %s", result.ResourceOwner)
	}
}

func TestMergeScenarioWithResourceHandlingOption(t *testing.T) {
	parent := Scenario{
		ResourceHandlingOption: "arn",
	}
	child := Scenario{
		ResourceHandlingOption: "prefix",
	}

	result := MergeScenario(parent, child)

	if result.ResourceHandlingOption != "prefix" {
		t.Errorf("Expected child ResourceHandlingOption, got %s", result.ResourceHandlingOption)
	}
}

func TestMergeScenarioWithContext(t *testing.T) {
	parent := Scenario{
		Context: []ContextEntryYml{
			{ContextKeyName: "aws:userid", ContextKeyValues: []string{"parent-user"}, ContextKeyType: "string"},
		},
	}
	child := Scenario{
		Context: []ContextEntryYml{
			{ContextKeyName: "aws:userid", ContextKeyValues: []string{"child-user"}, ContextKeyType: "string"},
		},
	}

	result := MergeScenario(parent, child)

	// Context should be replaced, not merged
	if len(result.Context) != 1 {
		t.Errorf("Expected 1 context entry, got %d", len(result.Context))
	}
	if result.Context[0].ContextKeyValues[0] != "child-user" {
		t.Errorf("Expected child context, got %s", result.Context[0].ContextKeyValues[0])
	}
}

func TestMergeScenarioWithResourcePolicyTemplate(t *testing.T) {
	parent := Scenario{
		ResourcePolicyJSON: "parent.json",
	}
	child := Scenario{
		ResourcePolicyTemplate: "child.json.tpl",
	}

	result := MergeScenario(parent, child)

	if result.ResourcePolicyTemplate != "child.json.tpl" {
		t.Errorf("Expected child ResourcePolicyTemplate, got %s", result.ResourcePolicyTemplate)
	}
	if result.ResourcePolicyJSON != "" {
		t.Error("ResourcePolicyJSON should be cleared when ResourcePolicyTemplate is set")
	}
}

func TestMergeScenarioWithResourcePolicyJSON(t *testing.T) {
	parent := Scenario{
		ResourcePolicyTemplate: "parent.json.tpl",
	}
	child := Scenario{
		ResourcePolicyJSON: "child.json",
	}

	result := MergeScenario(parent, child)

	if result.ResourcePolicyJSON != "child.json" {
		t.Errorf("Expected child ResourcePolicyJSON, got %s", result.ResourcePolicyJSON)
	}
	if result.ResourcePolicyTemplate != "" {
		t.Error("ResourcePolicyTemplate should be cleared when ResourcePolicyJSON is set")
	}
}

func TestMergeScenarioWithPolicyTemplate(t *testing.T) {
	parent := Scenario{
		PolicyJSON: "parent.json",
	}
	child := Scenario{
		PolicyTemplate: "child.json.tpl",
	}

	result := MergeScenario(parent, child)

	if result.PolicyTemplate != "child.json.tpl" {
		t.Errorf("Expected child PolicyTemplate, got %s", result.PolicyTemplate)
	}
	if result.PolicyJSON != "" {
		t.Error("PolicyJSON should be cleared when PolicyTemplate is set")
	}
}

func TestMergeScenarioVarsInitialization(t *testing.T) {
	parent := Scenario{
		// Vars is nil
	}
	child := Scenario{
		Vars: map[string]any{
			"key": "value",
		},
	}

	result := MergeScenario(parent, child)

	if result.Vars == nil {
		t.Error("Vars should be initialized")
	}
	if result.Vars["key"] != "value" {
		t.Error("Child var should be present")
	}
}
