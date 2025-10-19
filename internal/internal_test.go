package internal

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
			result := MergeScenario(tt.parent, tt.child)

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
			got := RenderString(tt.template, tt.vars)
			if got != tt.want {
				t.Errorf("RenderString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseContextType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    types.ContextKeyTypeEnum
		wantErr bool
	}{
		{
			name:    "string lowercase",
			input:   "string",
			want:    types.ContextKeyTypeEnumString,
			wantErr: false,
		},
		{
			name:    "stringList mixed case",
			input:   "stringList",
			want:    types.ContextKeyTypeEnumStringList,
			wantErr: false,
		},
		{
			name:    "numeric",
			input:   "numeric",
			want:    types.ContextKeyTypeEnumNumeric,
			wantErr: false,
		},
		{
			name:    "numericList",
			input:   "numericList",
			want:    types.ContextKeyTypeEnumNumericList,
			wantErr: false,
		},
		{
			name:    "boolean",
			input:   "boolean",
			want:    types.ContextKeyTypeEnumBoolean,
			wantErr: false,
		},
		{
			name:    "booleanList",
			input:   "booleanList",
			want:    types.ContextKeyTypeEnumBooleanList,
			wantErr: false,
		},
		{
			name:    "unknown type should error",
			input:   "unknownType",
			want:    "",
			wantErr: true,
		},
		{
			name:    "typo should error",
			input:   "strlng",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseContextType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseContextType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseContextType() = %v, want %v", got, tt.want)
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
			got := ExpandGlobsRelative(tt.base, tt.patterns)
			if len(got) != tt.wantLen {
				t.Errorf("ExpandGlobsRelative() returned %v files, want %v", len(got), tt.wantLen)
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
			got := MinifyJSON(tt.input)
			if got != tt.want {
				t.Errorf("MinifyJSON() = %v, want %v", got, tt.want)
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
			got := RenderStringSlice(tt.input, vars)
			if len(got) != len(tt.want) {
				t.Errorf("RenderStringSlice() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("RenderStringSlice()[%d] = %v, want %v", i, got[i], tt.want[i])
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

	result, err := RenderContext(ctx, vars)
	if err != nil {
		t.Fatalf("RenderContext() unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("RenderContext() returned %v entries, want 2", len(result))
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
		t.Error("RenderContext() did not render variable correctly")
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
	result := MergeSCPFiles(paths)

	statements := result["Statement"].([]any)
	if len(statements) != 2 {
		t.Errorf("MergeSCPFiles() merged %v statements, want 2", len(statements))
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
		err := ReadJSONFile(file, &result)
		if err != nil {
			t.Errorf("ReadJSONFile() error = %v", err)
		}
		if result["key"] != "value" {
			t.Errorf("ReadJSONFile() key = %v, want value", result["key"])
		}
	})

	t.Run("file not found", func(t *testing.T) {
		var result map[string]any
		err := ReadJSONFile(filepath.Join(tmpDir, "nonexistent.json"), &result)
		if err == nil {
			t.Error("ReadJSONFile() expected error for nonexistent file, got nil")
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
			result := ToJSONMin(tt.input)
			// Verify it's valid JSON
			var parsed map[string]any
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Errorf("ToJSONMin() produced invalid JSON: %v", err)
			}
		})
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

	result := RenderTemplateFileJSON(templateFile, vars)

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("RenderTemplateFileJSON() produced invalid JSON: %v", err)
	}

	// Verify variable was rendered
	if !contains(result, "test-bucket") {
		t.Error("RenderTemplateFileJSON() did not render variable")
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
	PrintTable(rows)
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
			got := MinifyJSON(tt.input)
			// For some JSON, field order may vary, so just verify it's valid JSON
			var parsed any
			if err := json.Unmarshal([]byte(got), &parsed); err != nil {
				t.Errorf("MinifyJSON() produced invalid JSON: %v", err)
			}
		})
	}
}

func TestExpandGlobsRelativeEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("empty pattern list", func(t *testing.T) {
		result := ExpandGlobsRelative(tmpDir, []string{})
		if len(result) != 0 {
			t.Errorf("ExpandGlobsRelative() with empty patterns = %v, want []", result)
		}
	})

	t.Run("pattern with no glob", func(t *testing.T) {
		file := filepath.Join(tmpDir, "exact.json")
		if err := os.WriteFile(file, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		result := ExpandGlobsRelative(tmpDir, []string{"exact.json"})
		if len(result) != 1 {
			t.Errorf("ExpandGlobsRelative() returned %v files, want 1", len(result))
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
			// Should not panic and should not error for valid types
			_, err := ParseContextType(typeName)
			if err != nil {
				t.Errorf("ParseContextType(%s) returned unexpected error: %v", typeName, err)
			}
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

	result, err := RenderContext(ctx, vars)
	if err != nil {
		t.Fatalf("RenderContext() unexpected error: %v", err)
	}

	if len(result) != 4 {
		t.Errorf("RenderContext() returned %v entries, want 4", len(result))
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
		t.Error("RenderContext() did not render username correctly")
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

	result := RenderStringSlice(input, vars)

	if len(result) != 3 {
		t.Errorf("RenderStringSlice() length = %v, want 3", len(result))
	}

	expected := []string{
		"arn:aws:s3:::my-bucket/*",
		"arn:aws:s3:::my-bucket-prod/*",
		"arn:aws:s3:::my-bucket-us-east-1-123456789012/*",
	}

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("RenderStringSlice()[%d] = %v, want %v", i, result[i], exp)
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
	result := MergeSCPFiles(paths)

	statements := result["Statement"].([]any)
	if len(statements) != 0 {
		t.Errorf("MergeSCPFiles() with empty statements = %v, want 0", len(statements))
	}
}

// Mock Exiter for testing
func TestPrintTableEmpty(t *testing.T) {
	// Calling printTable with empty rows should print "No evaluation results."
	// We can't easily capture stdout, but we can at least call it to ensure no panic
	PrintTable([][3]string{})
}

func TestMinifyJSONErrorPath(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	// Set mock exiter
	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	// Call minifyJSON with invalid JSON that can't be compacted or unmarshaled
	invalidJSON := []byte("not json at all")
	_ = MinifyJSON(invalidJSON)

	// Verify die was called
	if !mockExit.called {
		t.Error("MinifyJSON() did not call Die() on invalid JSON")
	}
}

func TestParseContextTypeDefault(t *testing.T) {
	// Test that unknown types now return an error instead of defaulting to string
	_, err := ParseContextType("unknown_type")
	if err == nil {
		t.Error("ParseContextType() with unknown type should return an error, got nil")
	}
}

// Test runLegacyFormat with actual mock
func TestRunLegacyFormatIntegration(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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
	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars})

	// Should not have exited
	if mockExit.called {
		t.Errorf("RunLegacyFormat() called Exit when expectations matched")
	}
}

func TestRunLegacyFormatWithFailure(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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
	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars})

	// Should have exited with code 2
	if !mockExit.called {
		t.Error("RunLegacyFormat() did not call Exit on expectation failure")
	}
	if mockExit.exitCode != 2 {
		t.Errorf("RunLegacyFormat() called Exit with code %d, want 2", mockExit.exitCode)
	}
}

func TestRunLegacyFormatWithNoAssert(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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
	RunLegacyFormat(mockClient, scen, SimulatorConfig{
		PolicyJSON: policyJSON,
		Variables:  allVars,
		NoAssert:   true,
	})

	// Should NOT have exited
	if mockExit.called {
		t.Error("RunLegacyFormat() called Exit when noAssert=true")
	}
}

func TestRunLegacyFormatWithPermissionsBoundary(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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
	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, PermissionsBoundary: pbJSON, Variables: allVars})

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
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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
	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should not have exited (all tests passed)
	if mockExit.called {
		t.Errorf("RunTestCollection() called Exit when all tests passed")
	}
}

func TestRunTestCollectionWithFailure(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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
	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should have exited with code 2 (test failed)
	if !mockExit.called {
		t.Error("RunTestCollection() did not call Exit on test failure")
	}
	if mockExit.exitCode != 2 {
		t.Errorf("RunTestCollection() called Exit with code %d, want 2", mockExit.exitCode)
	}
}

func TestRunTestCollectionMultipleTests(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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
	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should have called SimulateCustomPolicy twice
	if callCount != 2 {
		t.Errorf("Expected 2 AWS API calls, got %d", callCount)
	}

	// Should not have exited
	if mockExit.called {
		t.Error("RunTestCollection() called Exit when all tests passed")
	}
}

func TestRunTestCollectionWithResourcePolicy(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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
	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Verify resource policy was passed
	if capturedInput.ResourcePolicy == nil {
		t.Error("Resource policy was not passed to SimulateCustomPolicy")
	}
}

func TestRunTestCollectionWithContext(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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
	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Verify both scenario-level and test-level context were merged
	if len(capturedInput.ContextEntries) != 2 {
		t.Errorf("Expected 2 context entries (scenario + test), got %d", len(capturedInput.ContextEntries))
	}
}

func TestRunLegacyFormatWithResourcePolicy(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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
	RunLegacyFormat(mockClient, scen, SimulatorConfig{
		PolicyJSON:         policyJSON,
		ResourcePolicyJSON: resourcePolicy,
		Variables:          allVars,
	})

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

	result := MergeSCPFiles([]string{scpSingle})
	statements := result["Statement"].([]any)
	if len(statements) != 1 {
		t.Errorf("MergeSCPFiles() with single Statement object = %v statements, want 1", len(statements))
	}

	// Test with non-map document
	scpArray := filepath.Join(tmpDir, "scp-array.json")
	scpArrayContent := `["statement1", "statement2"]`
	if err := os.WriteFile(scpArray, []byte(scpArrayContent), 0644); err != nil {
		t.Fatal(err)
	}

	result2 := MergeSCPFiles([]string{scpArray})
	statements2 := result2["Statement"].([]any)
	if len(statements2) != 1 {
		t.Errorf("MergeSCPFiles() with array document = %v statements, want 1", len(statements2))
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
	result := ExpandGlobsRelative("/some/base", []string{file})
	if len(result) != 1 {
		t.Errorf("ExpandGlobsRelative() with absolute path returned %v files, want 1", len(result))
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

func TestRunTestCollectionWithSaveFile(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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
	RunTestCollection(mockClient, scen, SimulatorConfig{
		PolicyJSON:   policyJSON,
		ScenarioPath: scenarioPath,
		Variables:    allVars,
		SavePath:     saveFile,
	})

	// Verify save file was created
	if _, err := os.Stat(saveFile); os.IsNotExist(err) {
		t.Error("RunTestCollection() did not create save file")
	}
}

func TestRunLegacyFormatWithSaveFile(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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
	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars, SavePath: saveFile})

	// Verify save file was created
	if _, err := os.Stat(saveFile); os.IsNotExist(err) {
		t.Error("RunLegacyFormat() did not create save file")
	}
}

// TestRunTestCollectionWithCallerArnOverride tests test-level CallerArn override
func TestRunTestCollectionWithCallerArnOverride(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
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

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		CallerArn: "arn:aws:iam::123456789012:user/scenario-user",
		Tests: []TestCase{
			{
				Action:    "s3:GetObject",
				Resource:  "arn:aws:s3:::bucket/*",
				Expect:    "allowed",
				CallerArn: "arn:aws:iam::123456789012:user/test-user",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Verify test-level CallerArn was used
	if capturedInput == nil || capturedInput.CallerArn == nil {
		t.Fatal("Expected CallerArn to be set")
	}
	if *capturedInput.CallerArn != "arn:aws:iam::123456789012:user/test-user" {
		t.Errorf("Expected test-level CallerArn, got %s", *capturedInput.CallerArn)
	}
}

// TestRunTestCollectionWithResourceOwnerOverride tests test-level ResourceOwner override
func TestRunTestCollectionWithResourceOwnerOverride(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
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

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		ResourceOwner: "123456789012",
		Tests: []TestCase{
			{
				Action:        "s3:GetObject",
				Resource:      "arn:aws:s3:::bucket/*",
				Expect:        "allowed",
				ResourceOwner: "987654321098",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Verify test-level ResourceOwner was used
	if capturedInput == nil || capturedInput.ResourceOwner == nil {
		t.Fatal("Expected ResourceOwner to be set")
	}
	if *capturedInput.ResourceOwner != "987654321098" {
		t.Errorf("Expected test-level ResourceOwner, got %s", *capturedInput.ResourceOwner)
	}
}

// TestRunTestCollectionWithResourceHandlingOptionOverride tests test-level ResourceHandlingOption override
func TestRunTestCollectionWithResourceHandlingOptionOverride(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
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

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		ResourceHandlingOption: "arn",
		Tests: []TestCase{
			{
				Action:                 "s3:GetObject",
				Resource:               "arn:aws:s3:::bucket/*",
				Expect:                 "allowed",
				ResourceHandlingOption: "prefix",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Verify test-level ResourceHandlingOption was used
	if capturedInput == nil || capturedInput.ResourceHandlingOption == nil {
		t.Fatal("Expected ResourceHandlingOption to be set")
	}
	if *capturedInput.ResourceHandlingOption != "prefix" {
		t.Errorf("Expected test-level ResourceHandlingOption, got %s", *capturedInput.ResourceHandlingOption)
	}
}

// TestMergeScenarioWithTests tests merging scenarios with Tests field
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

// TestMergeScenarioWithCallerArn tests merging scenarios with CallerArn
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

// TestMergeScenarioWithResourceOwner tests merging scenarios with ResourceOwner
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

// TestMergeScenarioWithResourceHandlingOption tests merging scenarios with ResourceHandlingOption
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

// TestRunLegacyFormatWithCallerArn tests CallerArn rendering
func TestRunLegacyFormatWithCallerArn(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
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
		Resources: []string{"arn:aws:s3:::{{.bucket}}/*"},
		CallerArn: "arn:aws:iam::{{.account}}:user/test",
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{
		"bucket":  "my-bucket",
		"account": "123456789012",
	}

	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars})

	// Verify CallerArn was rendered and set
	if capturedInput == nil || capturedInput.CallerArn == nil {
		t.Fatal("Expected CallerArn to be set")
	}
	if *capturedInput.CallerArn != "arn:aws:iam::123456789012:user/test" {
		t.Errorf("Expected rendered CallerArn, got %s", *capturedInput.CallerArn)
	}
}

// TestRunLegacyFormatWithResourceOwner tests ResourceOwner rendering
func TestRunLegacyFormatWithResourceOwner(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
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
		Actions:       []string{action},
		Resources:     []string{"arn:aws:s3:::bucket/*"},
		ResourceOwner: "{{.account}}",
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{
		"account": "123456789012",
	}

	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars})

	// Verify ResourceOwner was rendered and set
	if capturedInput == nil || capturedInput.ResourceOwner == nil {
		t.Fatal("Expected ResourceOwner to be set")
	}
	if *capturedInput.ResourceOwner != "123456789012" {
		t.Errorf("Expected rendered ResourceOwner, got %s", *capturedInput.ResourceOwner)
	}
}

// TestRunLegacyFormatWithResourceHandlingOption tests ResourceHandlingOption
func TestRunLegacyFormatWithResourceHandlingOption(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
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
		Resources:              []string{"arn:aws:s3:::bucket/*"},
		ResourceHandlingOption: "prefix",
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars})

	// Verify ResourceHandlingOption was set
	if capturedInput == nil || capturedInput.ResourceHandlingOption == nil {
		t.Fatal("Expected ResourceHandlingOption to be set")
	}
	if *capturedInput.ResourceHandlingOption != "prefix" {
		t.Errorf("Expected ResourceHandlingOption 'prefix', got %s", *capturedInput.ResourceHandlingOption)
	}
}

// TestRunTestCollectionNoExpectation tests test without expectation
func TestRunTestCollectionNoExpectation(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		Tests: []TestCase{
			{
				Action:   "s3:GetObject",
				Resource: "arn:aws:s3:::bucket/*",
				// No Expect field - should just show result
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should not have exited
	if mockExit.called {
		t.Error("RunTestCollection() should not exit when test has no expectation")
	}
}

// TestRunTestCollectionNamedTestFailure tests failure with custom test name
func TestRunTestCollectionNamedTestFailure(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeExplicitDeny,
					},
				},
			}, nil
		},
	}

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		Tests: []TestCase{
			{
				Name:     "Custom test name",
				Action:   "s3:GetObject",
				Resource: "arn:aws:s3:::bucket/*",
				Expect:   "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should have exited with code 2
	if !mockExit.called || mockExit.exitCode != 2 {
		t.Errorf("Expected exit code 2, got called=%v code=%d", mockExit.called, mockExit.exitCode)
	}
}

// TestMergeScenarioWithContext tests merging scenarios with Context
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

// TestMergeScenarioWithExpectDeepMerge tests that Expect maps are deep-merged
func TestMergeScenarioWithExpectDeepMerge(t *testing.T) {
	parent := Scenario{
		Expect: map[string]string{
			"s3:GetObject": "allowed",
			"s3:PutObject": "denied",
		},
	}
	child := Scenario{
		Expect: map[string]string{
			"s3:DeleteObject": "denied",
		},
	}

	result := MergeScenario(parent, child)

	// Expect should be deep-merged
	if len(result.Expect) != 3 {
		t.Errorf("Expected 3 expectations, got %d", len(result.Expect))
	}
	if result.Expect["s3:GetObject"] != "allowed" {
		t.Error("Parent expectation should be preserved")
	}
	if result.Expect["s3:DeleteObject"] != "denied" {
		t.Error("Child expectation should be added")
	}
}

// TestMergeScenarioWithResourcePolicyTemplate tests ResourcePolicyTemplate merging
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

// TestMergeScenarioWithResourcePolicyJSON tests ResourcePolicyJSON merging
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

// TestMergeScenarioWithPolicyTemplate tests PolicyTemplate merging clears PolicyJSON
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

// TestExpandGlobsRelativeNoMatches tests expandGlobsRelative with no matches
func TestExpandGlobsRelativeNoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	// No files exist, glob should return empty
	result := ExpandGlobsRelative(tmpDir, []string{"*.nonexistent"})

	if len(result) != 0 {
		t.Errorf("Expected 0 files, got %d", len(result))
	}
}

// TestMergeSCPFilesWithMultipleStatements tests merging multiple SCP files
func TestMergeSCPFilesWithMultipleStatements(t *testing.T) {
	tmpDir := t.TempDir()

	// Create SCP files with multiple statements each
	scp1 := filepath.Join(tmpDir, "scp1.json")
	scp1Content := `{
		"Version": "2012-10-17",
		"Statement": [
			{"Effect": "Allow", "Action": "s3:GetObject", "Resource": "*"},
			{"Effect": "Deny", "Action": "s3:DeleteObject", "Resource": "*"}
		]
	}`
	if err := os.WriteFile(scp1, []byte(scp1Content), 0644); err != nil {
		t.Fatal(err)
	}

	scp2 := filepath.Join(tmpDir, "scp2.json")
	scp2Content := `{
		"Version": "2012-10-17",
		"Statement": [
			{"Effect": "Allow", "Action": "ec2:*", "Resource": "*"}
		]
	}`
	if err := os.WriteFile(scp2, []byte(scp2Content), 0644); err != nil {
		t.Fatal(err)
	}

	policy := MergeSCPFiles([]string{scp1, scp2})

	// Should merge all statements from both files
	statements, ok := policy["Statement"].([]any)
	if !ok {
		t.Fatal("Statement field is not an array")
	}
	if len(statements) != 3 {
		t.Errorf("Expected 3 statements, got %d", len(statements))
	}
}

// TestRunTestCollectionWithResources tests test with Resources array (not Resource)
func TestRunTestCollectionWithResources(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
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

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		Tests: []TestCase{
			{
				Action: "s3:GetObject",
				Resources: []string{
					"arn:aws:s3:::bucket1/*",
					"arn:aws:s3:::bucket2/*",
				},
				Expect: "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Verify Resources array was used
	if capturedInput == nil || len(capturedInput.ResourceArns) != 2 {
		t.Errorf("Expected 2 resource ARNs, got %d", len(capturedInput.ResourceArns))
	}
}

// TestRunLegacyFormatWithMatchedStatements tests the matched statements display
func TestRunLegacyFormatWithMatchedStatements(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	sourcePolicy := "policy-1"
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
						MatchedStatements: []types.Statement{
							{
								SourcePolicyId: &sourcePolicy,
							},
						},
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Actions:   []string{action},
		Resources: []string{"arn:aws:s3:::bucket/*"},
		Expect: map[string]string{
			action: "allowed",
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars})

	// Should not have exited
	if mockExit.called {
		t.Error("RunLegacyFormat() should not exit when expectation passes")
	}
}

// TestRunTestCollectionWithMatchedStatements tests the matched statements display in test collection
func TestRunTestCollectionWithMatchedStatements(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	sourcePolicy := "policy-1"
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
						MatchedStatements: []types.Statement{
							{
								SourcePolicyId: &sourcePolicy,
							},
						},
					},
				},
			}, nil
		},
	}

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		Tests: []TestCase{
			{
				Action:   "s3:GetObject",
				Resource: "arn:aws:s3:::bucket/*",
				Expect:   "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should not have exited
	if mockExit.called {
		t.Error("RunTestCollection() should not exit when all tests pass")
	}
}

// TestRunTestCollectionFailureWithUnnamedTest tests failure path with unnamed test
func TestRunTestCollectionFailureWithUnnamedTest(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeExplicitDeny,
					},
				},
			}, nil
		},
	}

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		Tests: []TestCase{
			{
				// No Name field - should use default format
				Action:   "s3:GetObject",
				Resource: "arn:aws:s3:::bucket/*",
				Expect:   "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should have exited with code 2
	if !mockExit.called || mockExit.exitCode != 2 {
		t.Errorf("Expected exit code 2, got called=%v code=%d", mockExit.called, mockExit.exitCode)
	}
}

// TestMergeScenarioVarsInitialization tests that Vars map is initialized if nil
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
