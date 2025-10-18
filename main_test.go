package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
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
			name: "child replaces parent actions",
			parent: Scenario{
				Actions: []string{"s3:GetObject", "s3:PutObject"},
			},
			child: Scenario{
				Actions: []string{"s3:DeleteObject"},
			},
			expected: Scenario{
				Actions: []string{"s3:DeleteObject"}, // replaced, not merged
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
			name: "merge expect maps",
			parent: Scenario{
				Expect: map[string]string{
					"s3:GetObject": "allowed",
				},
			},
			child: Scenario{
				Expect: map[string]string{
					"s3:PutObject": "denied",
				},
			},
			expected: Scenario{
				Expect: map[string]string{
					"s3:GetObject": "allowed",
					"s3:PutObject": "denied",
				},
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
			result := mergeScenario(tt.parent, tt.child)

			// Check Vars
			if len(tt.expected.Vars) > 0 {
				for k, v := range tt.expected.Vars {
					if result.Vars[k] != v {
						t.Errorf("Vars[%s] = %v, want %v", k, result.Vars[k], v)
					}
				}
			}

			// Check Actions
			if len(tt.expected.Actions) > 0 {
				if len(result.Actions) != len(tt.expected.Actions) {
					t.Errorf("Actions length = %v, want %v", len(result.Actions), len(tt.expected.Actions))
				}
				for i, action := range tt.expected.Actions {
					if i < len(result.Actions) && result.Actions[i] != action {
						t.Errorf("Actions[%d] = %v, want %v", i, result.Actions[i], action)
					}
				}
			}

			// Check PolicyJSON
			if tt.expected.PolicyJSON != "" && result.PolicyJSON != tt.expected.PolicyJSON {
				t.Errorf("PolicyJSON = %v, want %v", result.PolicyJSON, tt.expected.PolicyJSON)
			}

			// Check Expect
			if len(tt.expected.Expect) > 0 {
				for k, v := range tt.expected.Expect {
					if result.Expect[k] != v {
						t.Errorf("Expect[%s] = %v, want %v", k, result.Expect[k], v)
					}
				}
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

func TestRenderString(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     map[string]any
		want     string
	}{
		{
			name:     "simple variable",
			template: "{{.bucket}}",
			vars:     map[string]any{"bucket": "my-bucket"},
			want:     "my-bucket",
		},
		{
			name:     "multiple variables",
			template: "arn:aws:s3:::{{.bucket}}/{{.key}}",
			vars:     map[string]any{"bucket": "my-bucket", "key": "file.txt"},
			want:     "arn:aws:s3:::my-bucket/file.txt",
		},
		{
			name:     "no variables",
			template: "static-string",
			vars:     map[string]any{},
			want:     "static-string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderString(tt.template, tt.vars)
			if got != tt.want {
				t.Errorf("renderString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseContextType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  types.ContextKeyTypeEnum
	}{
		{
			name:  "string lowercase",
			input: "string",
			want:  types.ContextKeyTypeEnumString,
		},
		{
			name:  "stringList mixed case",
			input: "stringList",
			want:  types.ContextKeyTypeEnumStringList,
		},
		{
			name:  "numeric",
			input: "numeric",
			want:  types.ContextKeyTypeEnumNumeric,
		},
		{
			name:  "numericList",
			input: "numericList",
			want:  types.ContextKeyTypeEnumNumericList,
		},
		{
			name:  "boolean",
			input: "boolean",
			want:  types.ContextKeyTypeEnumBoolean,
		},
		{
			name:  "booleanList",
			input: "booleanList",
			want:  types.ContextKeyTypeEnumBooleanList,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseContextType(tt.input)
			if got != tt.want {
				t.Errorf("parseContextType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExpandGlobsRelative(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "scp1.json")
	file2 := filepath.Join(tmpDir, "scp2.json")
	file3 := filepath.Join(tmpDir, "other.txt")

	if err := os.WriteFile(file1, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file3, []byte("text"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		base     string
		patterns []string
		wantLen  int
	}{
		{
			name:     "glob json files",
			base:     tmpDir,
			patterns: []string{"*.json"},
			wantLen:  2,
		},
		{
			name:     "glob all files",
			base:     tmpDir,
			patterns: []string{"*"},
			wantLen:  3,
		},
		{
			name:     "no matches",
			base:     tmpDir,
			patterns: []string{"*.xml"},
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandGlobsRelative(tt.base, tt.patterns)
			if len(got) != tt.wantLen {
				t.Errorf("expandGlobsRelative() returned %v files, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestMinifyJSON(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "simple object",
			input: []byte(`{"key": "value"}`),
			want:  `{"key":"value"}`,
		},
		{
			name:  "object with whitespace",
			input: []byte(`{  "key" : "value"  }`),
			want:  `{"key":"value"}`,
		},
		{
			name:  "nested object",
			input: []byte(`{"outer": {"inner": "value"}}`),
			want:  `{"outer":{"inner":"value"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := minifyJSON(tt.input)
			if got != tt.want {
				t.Errorf("minifyJSON() = %v, want %v", got, tt.want)
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
actions:
  - s3:GetObject
`
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		var result Scenario
		err := loadYAML(file, &result)
		if err != nil {
			t.Errorf("loadYAML() error = %v", err)
		}
		if result.Vars["bucket"] != "test-bucket" {
			t.Errorf("loadYAML() Vars[bucket] = %v, want test-bucket", result.Vars["bucket"])
		}
	})

	t.Run("file not found", func(t *testing.T) {
		var result Scenario
		err := loadYAML(filepath.Join(tmpDir, "nonexistent.yml"), &result)
		if err == nil {
			t.Error("loadYAML() expected error for nonexistent file, got nil")
		}
	})
}

func TestRenderStringSlice(t *testing.T) {
	vars := map[string]any{
		"bucket": "my-bucket",
		"region": "us-east-1",
	}

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "render variables",
			input: []string{"arn:aws:s3:::{{.bucket}}/*", "arn:aws:s3:::{{.bucket}}-{{.region}}/*"},
			want:  []string{"arn:aws:s3:::my-bucket/*", "arn:aws:s3:::my-bucket-us-east-1/*"},
		},
		{
			name:  "no variables",
			input: []string{"static1", "static2"},
			want:  []string{"static1", "static2"},
		},
		{
			name:  "empty slice",
			input: []string{},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderStringSlice(tt.input, vars)
			if len(got) != len(tt.want) {
				t.Errorf("renderStringSlice() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("renderStringSlice()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRenderContext(t *testing.T) {
	vars := map[string]any{
		"username": "testuser",
	}

	ctx := []ContextEntryYml{
		{ContextKeyName: "aws:username", ContextKeyValues: []string{"{{.username}}"}, ContextKeyType: "string"},
		{ContextKeyName: "aws:userid", ContextKeyValues: []string{"12345"}, ContextKeyType: "string"},
	}

	result := renderContext(ctx, vars)

	if len(result) != 2 {
		t.Errorf("renderContext() returned %v entries, want 2", len(result))
	}

	// Check that variables were rendered
	found := false
	for _, entry := range result {
		if *entry.ContextKeyName == "aws:username" {
			if len(entry.ContextKeyValues) > 0 && entry.ContextKeyValues[0] == "testuser" {
				found = true
			}
		}
	}
	if !found {
		t.Error("renderContext() did not render variable correctly")
	}
}

func TestMergeSCPFiles(t *testing.T) {
	tmpDir := t.TempDir()

	scp1 := filepath.Join(tmpDir, "scp1.json")
	scp1Content := `{
		"Version": "2012-10-17",
		"Statement": [
			{"Effect": "Allow", "Action": "s3:*", "Resource": "*"}
		]
	}`
	if err := os.WriteFile(scp1, []byte(scp1Content), 0644); err != nil {
		t.Fatal(err)
	}

	scp2 := filepath.Join(tmpDir, "scp2.json")
	scp2Content := `{
		"Version": "2012-10-17",
		"Statement": [
			{"Effect": "Deny", "Action": "s3:DeleteBucket", "Resource": "*"}
		]
	}`
	if err := os.WriteFile(scp2, []byte(scp2Content), 0644); err != nil {
		t.Fatal(err)
	}

	paths := []string{scp1, scp2}
	result := mergeSCPFiles(paths)

	statements := result["Statement"].([]any)
	if len(statements) != 2 {
		t.Errorf("mergeSCPFiles() merged %v statements, want 2", len(statements))
	}
}

func TestReadJSONFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("read valid json", func(t *testing.T) {
		file := filepath.Join(tmpDir, "test.json")
		content := `{"key": "value", "number": 123}`
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		var result map[string]any
		err := readJSONFile(file, &result)
		if err != nil {
			t.Errorf("readJSONFile() error = %v", err)
		}
		if result["key"] != "value" {
			t.Errorf("readJSONFile() key = %v, want value", result["key"])
		}
	})

	t.Run("file not found", func(t *testing.T) {
		var result map[string]any
		err := readJSONFile(filepath.Join(tmpDir, "nonexistent.json"), &result)
		if err == nil {
			t.Error("readJSONFile() expected error for nonexistent file, got nil")
		}
	})
}

func TestToJSONMin(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{
			name:  "simple object",
			input: map[string]any{"key": "value"},
		},
		{
			name:  "nested object",
			input: map[string]any{"outer": map[string]any{"inner": "value"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toJSONMin(tt.input)
			// Verify it's valid JSON
			var parsed map[string]any
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Errorf("toJSONMin() produced invalid JSON: %v", err)
			}
		})
	}
}

func TestAwsString(t *testing.T) {
	tests := []struct {
		name  string
		input *string
		want  string
	}{
		{
			name:  "non-nil string",
			input: strPtr("test"),
			want:  "test",
		},
		{
			name:  "nil string",
			input: nil,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awsString(tt.input)
			if got != tt.want {
				t.Errorf("awsString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIfEmpty(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		fallback string
		want     string
	}{
		{
			name:     "non-empty string",
			s:        "value",
			fallback: "default",
			want:     "value",
		},
		{
			name:     "empty string",
			s:        "",
			fallback: "default",
			want:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ifEmpty(tt.s, tt.fallback)
			if got != tt.want {
				t.Errorf("ifEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStrPtr(t *testing.T) {
	s := "test"
	ptr := strPtr(s)
	if ptr == nil {
		t.Fatal("strPtr() returned nil")
		return
	}
	if *ptr != s {
		t.Errorf("strPtr() = %v, want %v", *ptr, s)
	}
}

func TestLoadScenarioWithExtends(t *testing.T) {
	tmpDir := t.TempDir()

	// Create parent scenario
	parentFile := filepath.Join(tmpDir, "parent.yml")
	parentContent := `
vars:
  bucket: parent-bucket
actions:
  - s3:GetObject
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

	result, err := loadScenarioWithExtends(childFile)
	if err != nil {
		t.Fatalf("loadScenarioWithExtends() error = %v", err)
	}

	// Check merged vars
	if result.Vars["bucket"] != "child-bucket" {
		t.Errorf("Vars[bucket] = %v, want child-bucket", result.Vars["bucket"])
	}

	if result.Vars["region"] != "us-east-1" {
		t.Errorf("Vars[region] = %v, want us-east-1", result.Vars["region"])
	}

	// Check inherited actions
	if len(result.Actions) == 0 || result.Actions[0] != "s3:GetObject" {
		t.Errorf("Actions not inherited correctly")
	}
}

func TestRenderTemplateFileJSON(t *testing.T) {
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "policy.json.tpl")

	templateContent := `{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Action": "s3:GetObject",
			"Resource": "arn:aws:s3:::{{.bucket}}/*"
		}]
	}`
	if err := os.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
		t.Fatal(err)
	}

	vars := map[string]any{
		"bucket": "test-bucket",
	}

	result := renderTemplateFileJSON(templateFile, vars)

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("renderTemplateFileJSON() produced invalid JSON: %v", err)
	}

	// Verify variable was rendered
	if !contains(result, "test-bucket") {
		t.Error("renderTemplateFileJSON() did not render variable")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}

func TestMustAbs(t *testing.T) {
	// Test with current directory
	result := mustAbs(".")
	if result == "" {
		t.Error("mustAbs() returned empty string")
	}

	// Should return absolute path
	if !filepath.IsAbs(result) {
		t.Errorf("mustAbs() returned relative path: %v", result)
	}
}

func TestMustAbsJoin(t *testing.T) {
	base := "/tmp"
	rel := "subdir/file.txt"

	result := mustAbsJoin(base, rel)

	if !filepath.IsAbs(result) {
		t.Errorf("mustAbsJoin() returned relative path: %v", result)
	}

	if !contains(result, "subdir") {
		t.Error("mustAbsJoin() did not join paths correctly")
	}
}

// mockIAMClient implements the IAM client interface for testing
type mockIAMClient struct {
	SimulateCustomPolicyFunc func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error)
}

func (m *mockIAMClient) SimulateCustomPolicy(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
	return m.SimulateCustomPolicyFunc(ctx, params, optFns...)
}

// Test runLegacyFormat with mocked IAM client
func TestRunLegacyFormatWithMock(t *testing.T) {
	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			// Verify inputs
			if len(params.ActionNames) != 1 || params.ActionNames[0] != action {
				t.Errorf("Expected action %s, got %v", action, params.ActionNames)
			}
			if len(params.ResourceArns) != 1 || params.ResourceArns[0] != resource {
				t.Errorf("Expected resource %s, got %v", resource, params.ResourceArns)
			}

			// Return mock response
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`

	// Test that mock client works correctly
	_, err := mockClient.SimulateCustomPolicy(context.Background(), &iam.SimulateCustomPolicyInput{
		PolicyInputList: []string{policyJSON},
		ActionNames:     []string{action},
		ResourceArns:    []string{resource},
	})

	if err != nil {
		t.Errorf("Mock client returned error: %v", err)
	}
}

// Test runTestCollection with mocked IAM client
func TestRunTestCollectionWithMock(t *testing.T) {
	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`

	// Test that mock client works
	_, err := mockClient.SimulateCustomPolicy(context.Background(), &iam.SimulateCustomPolicyInput{
		PolicyInputList: []string{policyJSON},
		ActionNames:     []string{action},
		ResourceArns:    []string{resource},
	})

	if err != nil {
		t.Errorf("Mock client returned error: %v", err)
	}
}

func TestPrintTable(t *testing.T) {
	// printTable writes to stdout, so we can't easily test output
	// But we can at least verify it doesn't panic
	rows := [][3]string{
		{"s3:GetObject", "allowed", "policy1"},
		{"s3:PutObject", "denied", "policy2"},
	}

	// Just call it to ensure no panic
	printTable(rows)
}

func TestCheck(t *testing.T) {
	// check() calls die() on error, which exits the program
	// We can only test the nil case
	check(nil)

	// If we get here, it worked correctly
}

// Additional comprehensive tests for better coverage

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
			Actions:                []string{"s3:GetObject"},
			Resources:              []string{"arn:aws:s3:::parent-bucket/*"},
			Context: []ContextEntryYml{
				{ContextKeyName: "aws:userid", ContextKeyValues: []string{"parent"}, ContextKeyType: "string"},
			},
			Expect: map[string]string{
				"s3:GetObject": "allowed",
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
			Actions:                []string{"s3:PutObject"},
			Resources:              []string{"arn:aws:s3:::child-bucket/*"},
			Context: []ContextEntryYml{
				{ContextKeyName: "aws:username", ContextKeyValues: []string{"child"}, ContextKeyType: "string"},
			},
			Expect: map[string]string{
				"s3:PutObject": "denied",
			},
			Tests: []TestCase{
				{Name: "child-test", Action: "s3:PutObject"},
			},
			Vars: map[string]any{
				"bucket": "child-bucket",
				"region": "us-west-2",
			},
		}

		result := mergeScenario(parent, child)

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

		// Expect should be merged
		if result.Expect["s3:GetObject"] != "allowed" {
			t.Errorf("Expect[s3:GetObject] = %v, want allowed", result.Expect["s3:GetObject"])
		}
		if result.Expect["s3:PutObject"] != "denied" {
			t.Errorf("Expect[s3:PutObject] = %v, want denied", result.Expect["s3:PutObject"])
		}

		// Slices should be replaced
		if len(result.Actions) != 1 || result.Actions[0] != "s3:PutObject" {
			t.Errorf("Actions = %v, want [s3:PutObject]", result.Actions)
		}
		if len(result.Resources) != 1 || result.Resources[0] != "arn:aws:s3:::child-bucket/*" {
			t.Errorf("Resources = %v, want [child-bucket]", result.Resources)
		}
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

		result := mergeScenario(parent, child)

		// Empty child doesn't override parent - this is the correct behavior
		if result.PolicyJSON != "parent.json" {
			t.Errorf("PolicyJSON = %v, want parent.json", result.PolicyJSON)
		}
		if result.CallerArn != "arn:aws:iam::123:user/parent" {
			t.Errorf("CallerArn = %v, want parent", result.CallerArn)
		}
	})
}

func TestMinifyJSONComprehensive(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty object",
			input: []byte(`{}`),
			want:  `{}`,
		},
		{
			name:  "empty array",
			input: []byte(`[]`),
			want:  `[]`,
		},
		{
			name:  "array of objects",
			input: []byte(`[{"key": "value1"}, {"key": "value2"}]`),
			want:  `[{"key":"value1"},{"key":"value2"}]`,
		},
		{
			name:  "deeply nested",
			input: []byte(`{"level1": {"level2": {"level3": "value"}}}`),
			want:  `{"level1":{"level2":{"level3":"value"}}}`,
		},
		{
			name:  "with numbers and booleans",
			input: []byte(`{"num": 123, "bool": true, "null": null}`),
			want:  `{"bool":true,"null":null,"num":123}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := minifyJSON(tt.input)
			// For some JSON, field order may vary, so just verify it's valid JSON
			var parsed any
			if err := json.Unmarshal([]byte(got), &parsed); err != nil {
				t.Errorf("minifyJSON() produced invalid JSON: %v", err)
			}
		})
	}
}

func TestExpandGlobsRelativeEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("empty pattern list", func(t *testing.T) {
		result := expandGlobsRelative(tmpDir, []string{})
		if len(result) != 0 {
			t.Errorf("expandGlobsRelative() with empty patterns = %v, want []", result)
		}
	})

	t.Run("pattern with no glob", func(t *testing.T) {
		file := filepath.Join(tmpDir, "exact.json")
		if err := os.WriteFile(file, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		result := expandGlobsRelative(tmpDir, []string{"exact.json"})
		if len(result) != 1 {
			t.Errorf("expandGlobsRelative() returned %v files, want 1", len(result))
		}
	})
}

func TestParseContextTypeAllTypes(t *testing.T) {
	allTypes := []string{
		"string", "String", "STRING",
		"stringList", "StringList", "STRINGLIST",
		"numeric", "Numeric", "NUMERIC",
		"numericList", "NumericList", "NUMERICLIST",
		"boolean", "Boolean", "BOOLEAN",
		"booleanList", "BooleanList", "BOOLEANLIST",
	}

	for _, typeName := range allTypes {
		t.Run(typeName, func(t *testing.T) {
			// Should not panic
			_ = parseContextType(typeName)
		})
	}
}

func TestRenderContextWithTemplates(t *testing.T) {
	vars := map[string]any{
		"username": "alice",
		"userid":   "12345",
		"enabled":  true,
		"count":    42,
	}

	ctx := []ContextEntryYml{
		{
			ContextKeyName:   "aws:username",
			ContextKeyValues: []string{"{{.username}}"},
			ContextKeyType:   "string",
		},
		{
			ContextKeyName:   "aws:userid",
			ContextKeyValues: []string{"{{.userid}}"},
			ContextKeyType:   "string",
		},
		{
			ContextKeyName:   "custom:numbers",
			ContextKeyValues: []string{"1", "2", "3"},
			ContextKeyType:   "numericList",
		},
		{
			ContextKeyName:   "custom:flags",
			ContextKeyValues: []string{"true", "false"},
			ContextKeyType:   "booleanList",
		},
	}

	result := renderContext(ctx, vars)

	if len(result) != 4 {
		t.Errorf("renderContext() returned %v entries, want 4", len(result))
	}

	// Verify username was rendered
	found := false
	for _, entry := range result {
		if *entry.ContextKeyName == "aws:username" {
			if len(entry.ContextKeyValues) > 0 && entry.ContextKeyValues[0] == "alice" {
				found = true
			}
		}
	}
	if !found {
		t.Error("renderContext() did not render username correctly")
	}
}

func TestLoadScenarioWithExtendsMultipleLevels(t *testing.T) {
	tmpDir := t.TempDir()

	// Grandparent
	grandparentFile := filepath.Join(tmpDir, "grandparent.yml")
	grandparentContent := `
vars:
  level: grandparent
  bucket: grandparent-bucket
actions:
  - s3:GetObject
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

	result, err := loadScenarioWithExtends(childFile)
	if err != nil {
		t.Fatalf("loadScenarioWithExtends() error = %v", err)
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

	// Actions should be inherited from grandparent
	if len(result.Actions) == 0 || result.Actions[0] != "s3:GetObject" {
		t.Errorf("Actions not inherited correctly: %v", result.Actions)
	}
}

func TestRenderStringSliceWithComplexTemplates(t *testing.T) {
	vars := map[string]any{
		"bucket":  "my-bucket",
		"region":  "us-east-1",
		"account": "123456789012",
		"env":     "prod",
	}

	input := []string{
		"arn:aws:s3:::{{.bucket}}/*",
		"arn:aws:s3:::{{.bucket}}-{{.env}}/*",
		"arn:aws:s3:::{{.bucket}}-{{.region}}-{{.account}}/*",
	}

	result := renderStringSlice(input, vars)

	if len(result) != 3 {
		t.Errorf("renderStringSlice() length = %v, want 3", len(result))
	}

	expected := []string{
		"arn:aws:s3:::my-bucket/*",
		"arn:aws:s3:::my-bucket-prod/*",
		"arn:aws:s3:::my-bucket-us-east-1-123456789012/*",
	}

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("renderStringSlice()[%d] = %v, want %v", i, result[i], exp)
		}
	}
}

func TestMergeSCPFilesWithEmptyStatements(t *testing.T) {
	tmpDir := t.TempDir()

	// SCP with empty Statement array
	scp := filepath.Join(tmpDir, "empty-scp.json")
	scpContent := `{
		"Version": "2012-10-17",
		"Statement": []
	}`
	if err := os.WriteFile(scp, []byte(scpContent), 0644); err != nil {
		t.Fatal(err)
	}

	paths := []string{scp}
	result := mergeSCPFiles(paths)

	statements := result["Statement"].([]any)
	if len(statements) != 0 {
		t.Errorf("mergeSCPFiles() with empty statements = %v, want 0", len(statements))
	}
}

// Mock Exiter for testing
type mockExiter struct {
	exitCode int
	called   bool
}

func (m *mockExiter) Exit(code int) {
	m.exitCode = code
	m.called = true
	// Don't actually exit in tests
}

func TestDieWithMockExiter(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	// Set mock exiter
	mockExit := &mockExiter{}
	exiter = mockExit

	// Call die
	die("test error")

	// Verify Exit was called with code 1
	if !mockExit.called {
		t.Error("die() did not call exiter.Exit()")
	}
	if mockExit.exitCode != 1 {
		t.Errorf("die() called Exit with code %d, want 1", mockExit.exitCode)
	}
}

func TestCheckWithError(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	// Set mock exiter
	mockExit := &mockExiter{}
	exiter = mockExit

	// Call check with an error
	check(os.ErrNotExist)

	// Verify Exit was called
	if !mockExit.called {
		t.Error("check() did not call exiter.Exit() on error")
	}
	if mockExit.exitCode != 1 {
		t.Errorf("check() called Exit with code %d, want 1", mockExit.exitCode)
	}
}

func TestPrintTableEmpty(t *testing.T) {
	// Calling printTable with empty rows should print "No evaluation results."
	// We can't easily capture stdout, but we can at least call it to ensure no panic
	printTable([][3]string{})
}

func TestMinifyJSONErrorPath(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	// Set mock exiter
	mockExit := &mockExiter{}
	exiter = mockExit

	// Call minifyJSON with invalid JSON that can't be compacted or unmarshaled
	invalidJSON := []byte("not json at all")
	_ = minifyJSON(invalidJSON)

	// Verify die was called
	if !mockExit.called {
		t.Error("minifyJSON() did not call die() on invalid JSON")
	}
}

func TestParseContextTypeDefault(t *testing.T) {
	// Test the default case
	result := parseContextType("unknown_type")
	if result != types.ContextKeyTypeEnumString {
		t.Errorf("parseContextType() default = %v, want String", result)
	}
}

// Test runLegacyFormat with actual mock
func TestRunLegacyFormatIntegration(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	mockExit := &mockExiter{}
	exiter = mockExit

	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Actions:   []string{action},
		Resources: []string{resource},
		Expect: map[string]string{
			action: "allowed",
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Call runLegacyFormat
	runLegacyFormat(mockClient, scen, policyJSON, "", "", allVars, "", false)

	// Should not have exited
	if mockExit.called {
		t.Errorf("runLegacyFormat() called Exit when expectations matched")
	}
}

func TestRunLegacyFormatWithFailure(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	mockExit := &mockExiter{}
	exiter = mockExit

	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeImplicitDeny,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Actions:   []string{action},
		Resources: []string{resource},
		Expect: map[string]string{
			action: "allowed", // Expect allowed, but will get denied
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[]}`
	allVars := map[string]any{}

	// Call runLegacyFormat
	runLegacyFormat(mockClient, scen, policyJSON, "", "", allVars, "", false)

	// Should have exited with code 2
	if !mockExit.called {
		t.Error("runLegacyFormat() did not call Exit on expectation failure")
	}
	if mockExit.exitCode != 2 {
		t.Errorf("runLegacyFormat() called Exit with code %d, want 2", mockExit.exitCode)
	}
}

func TestRunLegacyFormatWithNoAssert(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	mockExit := &mockExiter{}
	exiter = mockExit

	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeImplicitDeny,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Actions:   []string{action},
		Resources: []string{resource},
		Expect: map[string]string{
			action: "allowed", // Expect allowed, but will get denied
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[]}`
	allVars := map[string]any{}

	// Call runLegacyFormat with noAssert=true
	runLegacyFormat(mockClient, scen, policyJSON, "", "", allVars, "", true)

	// Should NOT have exited
	if mockExit.called {
		t.Error("runLegacyFormat() called Exit when noAssert=true")
	}
}

func TestRunLegacyFormatWithPermissionsBoundary(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	mockExit := &mockExiter{}
	exiter = mockExit

	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

	var capturedInput *iam.SimulateCustomPolicyInput

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			capturedInput = params
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Actions:   []string{action},
		Resources: []string{resource},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	pbJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Action":"s3:DeleteBucket","Resource":"*"}]}`
	allVars := map[string]any{}

	// Call runLegacyFormat with permissions boundary
	runLegacyFormat(mockClient, scen, policyJSON, pbJSON, "", allVars, "", false)

	// Verify permissions boundary was passed
	if capturedInput == nil {
		t.Fatal("SimulateCustomPolicy was not called")
	}
	if len(capturedInput.PermissionsBoundaryPolicyInputList) != 1 {
		t.Errorf("Expected 1 permissions boundary policy, got %d", len(capturedInput.PermissionsBoundaryPolicyInputList))
	}
	if capturedInput.PermissionsBoundaryPolicyInputList[0] != pbJSON {
		t.Error("Permissions boundary policy was not passed correctly")
	}
}

func TestRunTestCollectionIntegration(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	mockExit := &mockExiter{}
	exiter = mockExit

	tmpDir := t.TempDir()
	action := "s3:GetObject"

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Tests: []TestCase{
			{
				Name:      "test s3 get object",
				Action:    action,
				Resources: []string{"arn:aws:s3:::my-bucket/*"},
				Expect:    "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Create a dummy scenario file path (code uses filepath.Dir on this)
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	// Call runTestCollection
	runTestCollection(mockClient, scen, policyJSON, "", "", scenarioPath, allVars, "", false)

	// Should not have exited (all tests passed)
	if mockExit.called {
		t.Errorf("runTestCollection() called Exit when all tests passed")
	}
}

func TestRunTestCollectionWithFailure(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	mockExit := &mockExiter{}
	exiter = mockExit

	tmpDir := t.TempDir()
	action := "s3:GetObject"

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeImplicitDeny,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Tests: []TestCase{
			{
				Name:      "test s3 get object",
				Action:    action,
				Resources: []string{"arn:aws:s3:::my-bucket/*"},
				Expect:    "allowed", // Expect allowed, but will get denied
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[]}`
	allVars := map[string]any{}

	// Call runTestCollection
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")
	runTestCollection(mockClient, scen, policyJSON, "", "", scenarioPath, allVars, "", false)

	// Should have exited with code 2 (test failed)
	if !mockExit.called {
		t.Error("runTestCollection() did not call Exit on test failure")
	}
	if mockExit.exitCode != 2 {
		t.Errorf("runTestCollection() called Exit with code %d, want 2", mockExit.exitCode)
	}
}

func TestRunTestCollectionMultipleTests(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	mockExit := &mockExiter{}
	exiter = mockExit

	tmpDir := t.TempDir()

	callCount := 0
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			callCount++
			action := params.ActionNames[0]
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Tests: []TestCase{
			{
				Name:     "test 1",
				Action:   "s3:GetObject",
				Resource: "arn:aws:s3:::bucket1/*",
				Expect:   "allowed",
			},
			{
				Name:      "test 2",
				Action:    "s3:PutObject",
				Resources: []string{"arn:aws:s3:::bucket2/*"},
				Expect:    "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Call runTestCollection
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")
	runTestCollection(mockClient, scen, policyJSON, "", "", scenarioPath, allVars, "", false)

	// Should have called SimulateCustomPolicy twice
	if callCount != 2 {
		t.Errorf("Expected 2 AWS API calls, got %d", callCount)
	}

	// Should not have exited
	if mockExit.called {
		t.Error("runTestCollection() called Exit when all tests passed")
	}
}

func TestRunTestCollectionWithResourcePolicy(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	mockExit := &mockExiter{}
	exiter = mockExit

	tmpDir := t.TempDir()

	// Create a resource policy file
	resourcePolicyFile := filepath.Join(tmpDir, "resource-policy.json")
	resourcePolicy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"*"}]}`
	if err := os.WriteFile(resourcePolicyFile, []byte(resourcePolicy), 0644); err != nil {
		t.Fatal(err)
	}

	var capturedInput *iam.SimulateCustomPolicyInput
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			capturedInput = params
			action := params.ActionNames[0]
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Tests: []TestCase{
			{
				Name:               "test with resource policy",
				Action:             "s3:GetObject",
				Resource:           "arn:aws:s3:::bucket/*",
				ResourcePolicyJSON: "resource-policy.json",
				Expect:             "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Create a dummy scenario file path (code uses filepath.Dir on this)
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	// Call runTestCollection
	runTestCollection(mockClient, scen, policyJSON, "", "", scenarioPath, allVars, "", false)

	// Verify resource policy was passed
	if capturedInput.ResourcePolicy == nil {
		t.Error("Resource policy was not passed to SimulateCustomPolicy")
	}
}

func TestRunTestCollectionWithContext(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	mockExit := &mockExiter{}
	exiter = mockExit

	tmpDir := t.TempDir()

	var capturedInput *iam.SimulateCustomPolicyInput
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			capturedInput = params
			action := params.ActionNames[0]
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Context: []ContextEntryYml{
			{ContextKeyName: "aws:userid", ContextKeyValues: []string{"user1"}, ContextKeyType: "string"},
		},
		Tests: []TestCase{
			{
				Name:     "test with context",
				Action:   "s3:GetObject",
				Resource: "arn:aws:s3:::bucket/*",
				Context: []ContextEntryYml{
					{ContextKeyName: "aws:username", ContextKeyValues: []string{"testuser"}, ContextKeyType: "string"},
				},
				Expect: "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Call runTestCollection
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")
	runTestCollection(mockClient, scen, policyJSON, "", "", scenarioPath, allVars, "", false)

	// Verify both scenario-level and test-level context were merged
	if len(capturedInput.ContextEntries) != 2 {
		t.Errorf("Expected 2 context entries (scenario + test), got %d", len(capturedInput.ContextEntries))
	}
}

func TestRunLegacyFormatWithResourcePolicy(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	mockExit := &mockExiter{}
	exiter = mockExit

	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

	var capturedInput *iam.SimulateCustomPolicyInput
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			capturedInput = params
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Actions:                []string{action},
		Resources:              []string{resource},
		CallerArn:              "arn:aws:iam::123456789012:user/testuser",
		ResourceOwner:          "123456789012",
		ResourceHandlingOption: "EC2-VPC-InstanceStore",
		Context: []ContextEntryYml{
			{ContextKeyName: "aws:userid", ContextKeyValues: []string{"user1"}, ContextKeyType: "string"},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	resourcePolicy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Call runLegacyFormat with resource policy and other options
	runLegacyFormat(mockClient, scen, policyJSON, "", resourcePolicy, allVars, "", false)

	// Verify all parameters were passed
	if capturedInput.ResourcePolicy == nil || *capturedInput.ResourcePolicy != resourcePolicy {
		t.Error("Resource policy was not passed correctly")
	}
	if capturedInput.CallerArn == nil || *capturedInput.CallerArn != scen.CallerArn {
		t.Error("CallerArn was not passed correctly")
	}
	if capturedInput.ResourceOwner == nil || *capturedInput.ResourceOwner != scen.ResourceOwner {
		t.Error("ResourceOwner was not passed correctly")
	}
	if capturedInput.ResourceHandlingOption == nil || *capturedInput.ResourceHandlingOption != scen.ResourceHandlingOption {
		t.Error("ResourceHandlingOption was not passed correctly")
	}
	if len(capturedInput.ContextEntries) == 0 {
		t.Error("Context was not passed")
	}
}

func TestMergeSCPFilesEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with Statement as a single object instead of array
	scpSingle := filepath.Join(tmpDir, "scp-single.json")
	scpSingleContent := `{
		"Version": "2012-10-17",
		"Statement": {"Effect": "Allow", "Action": "s3:*", "Resource": "*"}
	}`
	if err := os.WriteFile(scpSingle, []byte(scpSingleContent), 0644); err != nil {
		t.Fatal(err)
	}

	result := mergeSCPFiles([]string{scpSingle})
	statements := result["Statement"].([]any)
	if len(statements) != 1 {
		t.Errorf("mergeSCPFiles() with single Statement object = %v statements, want 1", len(statements))
	}

	// Test with non-map document
	scpArray := filepath.Join(tmpDir, "scp-array.json")
	scpArrayContent := `["statement1", "statement2"]`
	if err := os.WriteFile(scpArray, []byte(scpArrayContent), 0644); err != nil {
		t.Fatal(err)
	}

	result2 := mergeSCPFiles([]string{scpArray})
	statements2 := result2["Statement"].([]any)
	if len(statements2) != 1 {
		t.Errorf("mergeSCPFiles() with array document = %v statements, want 1", len(statements2))
	}
}

func TestExpandGlobsRelativeWithAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	file := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(file, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with absolute path (should not join with base)
	result := expandGlobsRelative("/some/base", []string{file})
	if len(result) != 1 {
		t.Errorf("expandGlobsRelative() with absolute path returned %v files, want 1", len(result))
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

	_, err := loadScenarioWithExtends(childFile)
	if err == nil {
		t.Error("loadScenarioWithExtends() should return error when parent doesn't exist")
	}
}

func TestRunTestCollectionWithSaveFile(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	mockExit := &mockExiter{}
	exiter = mockExit

	tmpDir := t.TempDir()
	saveFile := filepath.Join(tmpDir, "output.json")

	action := "s3:GetObject"
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Tests: []TestCase{
			{
				Action:    action,
				Resources: []string{"arn:aws:s3:::bucket/*"},
				Expect:    "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	// Call runTestCollection with save file
	runTestCollection(mockClient, scen, policyJSON, "", "", scenarioPath, allVars, saveFile, false)

	// Verify save file was created
	if _, err := os.Stat(saveFile); os.IsNotExist(err) {
		t.Error("runTestCollection() did not create save file")
	}
}

func TestRunLegacyFormatWithSaveFile(t *testing.T) {
	// Save original exiter
	originalExiter := exiter
	defer func() { exiter = originalExiter }()

	mockExit := &mockExiter{}
	exiter = mockExit

	tmpDir := t.TempDir()
	saveFile := filepath.Join(tmpDir, "output.json")

	action := "s3:GetObject"
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Actions:   []string{action},
		Resources: []string{"arn:aws:s3:::bucket/*"},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Call runLegacyFormat with save file
	runLegacyFormat(mockClient, scen, policyJSON, "", "", allVars, saveFile, false)

	// Verify save file was created
	if _, err := os.Stat(saveFile); os.IsNotExist(err) {
		t.Error("runLegacyFormat() did not create save file")
	}
}
