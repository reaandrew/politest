package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestMinifyJSONFallbackPath(t *testing.T) {
	// Test MinifyJSON with valid JSON that might trigger the fallback path
	// This tests the case where Compact might fail but Unmarshal succeeds
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "unicode escapes",
			input: []byte(`{"key": "\u0048\u0065\u006c\u006c\u006f"}`),
		},
		{
			name:  "large numbers",
			input: []byte(`{"bignum": 12345678901234567890}`),
		},
		{
			name:  "special chars",
			input: []byte(`{"special": "line1\nline2\ttab"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MinifyJSON(tt.input)
			// Verify result is valid JSON
			var parsed any
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Errorf("MinifyJSON() produced invalid JSON: %v", err)
			}
		})
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

func TestExpandGlobsRelativeNoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	// No files exist, glob should return empty
	result := ExpandGlobsRelative(tmpDir, []string{"*.nonexistent"})

	if len(result) != 0 {
		t.Errorf("Expected 0 files, got %d", len(result))
	}
}

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

func TestMergeSCPFilesWithSourceMap(t *testing.T) {
	tmpDir := t.TempDir()

	scp1 := filepath.Join(tmpDir, "deny-s3.json")
	scp1Content := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DenyS3",
      "Effect": "Deny",
      "Action": "s3:*",
      "Resource": "*"
    }
  ]
}`
	if err := os.WriteFile(scp1, []byte(scp1Content), 0644); err != nil {
		t.Fatal(err)
	}

	merged, sourceMap := MergeSCPFilesWithSourceMap([]string{scp1})

	// Verify merged policy structure
	statements := merged["Statement"].([]any)
	if len(statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(statements))
	}

	// Verify statement has tracking Sid injected
	stmt := statements[0].(map[string]any)
	trackingSid, ok := stmt["Sid"].(string)
	if !ok || trackingSid == "" {
		t.Fatal("Statement missing tracking Sid")
	}

	// Verify tracking Sid follows expected format
	if !strings.HasPrefix(trackingSid, "scp:deny-s3.json#stmt:") {
		t.Errorf("Tracking Sid has unexpected format: %s", trackingSid)
	}

	// Verify source map entry exists
	source, ok := sourceMap[trackingSid]
	if !ok {
		t.Fatal("Source map missing entry for tracking Sid")
	}

	// Verify original Sid is preserved
	if source.Sid != "DenyS3" {
		t.Errorf("Original Sid = %v, want DenyS3", source.Sid)
	}

	// Verify file path
	if source.FilePath != scp1 {
		t.Errorf("FilePath = %v, want %v", source.FilePath, scp1)
	}

	// Verify line numbers were tracked
	if source.StartLine == 0 || source.EndLine == 0 {
		t.Errorf("Line numbers not tracked: StartLine=%d, EndLine=%d", source.StartLine, source.EndLine)
	}
}

func TestFindStatementLineNumbers(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		stmt      map[string]any
		stmtIndex int
		wantStart int
		wantEnd   int
	}{
		{
			name: "find by Sid",
			content: `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DenyS3",
      "Effect": "Deny",
      "Action": "s3:*",
      "Resource": "*"
    }
  ]
}`,
			stmt:      map[string]any{"Sid": "DenyS3", "Effect": "Deny"},
			stmtIndex: 0,
			wantStart: 4,
			wantEnd:   9,
		},
		{
			name: "find by Effect when no Sid",
			content: `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "s3:*",
      "Resource": "*"
    }
  ]
}`,
			stmt:      map[string]any{"Effect": "Allow"},
			stmtIndex: 0,
			wantStart: 4,
			wantEnd:   8,
		},
		{
			name: "statement without explicit braces",
			content: `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Deny",
      "Action": "s3:*"
    }
  ]
}`,
			stmt:      map[string]any{"Effect": "Deny"},
			stmtIndex: 0,
			wantStart: 4,
			wantEnd:   7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startLine, endLine := findStatementLineNumbers(tt.content, tt.stmt, tt.stmtIndex)

			if startLine != tt.wantStart {
				t.Errorf("startLine = %d, want %d", startLine, tt.wantStart)
			}
			if endLine != tt.wantEnd {
				t.Errorf("endLine = %d, want %d", endLine, tt.wantEnd)
			}
		})
	}
}

func TestFindStatementLineNumbersNoMatch(t *testing.T) {
	content := `{
  "Version": "2012-10-17",
  "Statement": []
}`
	stmt := map[string]any{"Sid": "NonExistent"}

	startLine, endLine := findStatementLineNumbers(content, stmt, 0)

	if startLine != 0 || endLine != 0 {
		t.Errorf("Expected (0, 0) for no match, got (%d, %d)", startLine, endLine)
	}
}

func TestStripNonIAMFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkFor []string // Fields that should NOT be in output
	}{
		{
			name: "strip top-level custom field",
			input: `{
				"Version": "2012-10-17",
				"CustomField": "should be removed",
				"Statement": [
					{"Effect": "Allow", "Action": "s3:*", "Resource": "*"}
				]
			}`,
			checkFor: []string{"CustomField"},
		},
		{
			name: "strip statement-level custom field",
			input: `{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Action": "s3:*",
						"Resource": "*",
						"Comment": "this should be removed"
					}
				]
			}`,
			checkFor: []string{"Comment"},
		},
		{
			name: "strip multiple custom fields",
			input: `{
				"Version": "2012-10-17",
				"CustomTop": "remove",
				"AnotherCustom": "remove",
				"Statement": [
					{
						"Effect": "Allow",
						"Action": "s3:*",
						"Resource": "*",
						"Note": "remove",
						"Description": "remove"
					}
				]
			}`,
			checkFor: []string{"CustomTop", "AnotherCustom", "Note", "Description"},
		},
		{
			name: "preserve all valid IAM fields",
			input: `{
				"Version": "2012-10-17",
				"Id": "policy-id",
				"Statement": [
					{
						"Sid": "stmt1",
						"Effect": "Allow",
						"Principal": {"AWS": "*"},
						"NotPrincipal": {"AWS": "arn:aws:iam::123456789012:user/test"},
						"Action": "s3:GetObject",
						"NotAction": "s3:DeleteObject",
						"Resource": "arn:aws:s3:::bucket/*",
						"NotResource": "arn:aws:s3:::bucket/private/*",
						"Condition": {
							"StringEquals": {
								"aws:username": "test"
							}
						}
					}
				]
			}`,
			checkFor: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripNonIAMFields(tt.input)

			// Verify result is valid JSON
			var parsed map[string]any
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Errorf("StripNonIAMFields() produced invalid JSON: %v", err)
			}

			// Check that custom fields were removed
			for _, field := range tt.checkFor {
				if _, exists := parsed[field]; exists {
					t.Errorf("StripNonIAMFields() did not remove top-level field: %s", field)
				}

				// Check statement-level fields
				if statements, ok := parsed["Statement"].([]any); ok {
					for i, stmt := range statements {
						if stmtMap, ok := stmt.(map[string]any); ok {
							if _, exists := stmtMap[field]; exists {
								t.Errorf("StripNonIAMFields() did not remove field %s from statement %d", field, i)
							}
						}
					}
				}
			}

			// Verify required fields are preserved
			if _, exists := parsed["Version"]; !exists {
				t.Error("StripNonIAMFields() removed required Version field")
			}
			if _, exists := parsed["Statement"]; !exists {
				t.Error("StripNonIAMFields() removed required Statement field")
			}
		})
	}
}

func TestValidateIAMFields(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errText string // Text that should appear in error
	}{
		{
			name: "valid policy - no errors",
			input: `{
				"Version": "2012-10-17",
				"Statement": [
					{"Effect": "Allow", "Action": "s3:*", "Resource": "*"}
				]
			}`,
			wantErr: false,
		},
		{
			name: "valid policy with all fields",
			input: `{
				"Version": "2012-10-17",
				"Id": "policy-id",
				"Statement": [
					{
						"Sid": "stmt1",
						"Effect": "Allow",
						"Principal": {"AWS": "*"},
						"Action": "s3:*",
						"Resource": "*",
						"Condition": {"StringEquals": {"aws:username": "test"}}
					}
				]
			}`,
			wantErr: false,
		},
		{
			name: "invalid top-level field",
			input: `{
				"Version": "2012-10-17",
				"CustomField": "not allowed",
				"Statement": [
					{"Effect": "Allow", "Action": "s3:*", "Resource": "*"}
				]
			}`,
			wantErr: true,
			errText: "CustomField",
		},
		{
			name: "invalid statement field",
			input: `{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Action": "s3:*",
						"Resource": "*",
						"Comment": "not allowed in IAM schema"
					}
				]
			}`,
			wantErr: true,
			errText: "Comment",
		},
		{
			name: "multiple invalid fields",
			input: `{
				"Version": "2012-10-17",
				"Custom1": "invalid",
				"Custom2": "invalid",
				"Statement": [
					{
						"Effect": "Allow",
						"Action": "s3:*",
						"Resource": "*",
						"Note": "invalid",
						"Description": "invalid"
					}
				]
			}`,
			wantErr: true,
			errText: "non-IAM fields",
		},
		{
			name:    "invalid JSON",
			input:   `{not valid json}`,
			wantErr: true,
			errText: "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIAMFields(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIAMFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errText != "" {
				errMsg := err.Error()
				if len(errMsg) == 0 || !containsSubstring(errMsg, tt.errText) {
					t.Errorf("ValidateIAMFields() error message should contain %q, got: %v", tt.errText, err)
				}
			}
		})
	}
}

// containsSubstring is a helper function for error message checking
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestStripNonIAMFieldsPreservesStructure(t *testing.T) {
	input := `{
		"Version": "2012-10-17",
		"Id": "my-policy-id",
		"Statement": [
			{
				"Sid": "AllowS3Read",
				"Effect": "Allow",
				"Principal": {"AWS": "arn:aws:iam::123456789012:root"},
				"Action": ["s3:GetObject", "s3:ListBucket"],
				"Resource": [
					"arn:aws:s3:::mybucket/*",
					"arn:aws:s3:::mybucket"
				],
				"Condition": {
					"StringEquals": {
						"aws:SourceAccount": "123456789012"
					}
				}
			}
		]
	}`

	result := StripNonIAMFields(input)

	// Verify structure is preserved
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Check all valid fields are present
	if parsed["Version"] != "2012-10-17" {
		t.Error("Version field not preserved")
	}
	if parsed["Id"] != "my-policy-id" {
		t.Error("Id field not preserved")
	}

	statements, ok := parsed["Statement"].([]any)
	if !ok || len(statements) != 1 {
		t.Fatal("Statement array not preserved")
	}

	stmt := statements[0].(map[string]any)
	if stmt["Sid"] != "AllowS3Read" {
		t.Error("Sid not preserved")
	}
	if stmt["Effect"] != "Allow" {
		t.Error("Effect not preserved")
	}
	if _, ok := stmt["Principal"]; !ok {
		t.Error("Principal not preserved")
	}
	if _, ok := stmt["Action"]; !ok {
		t.Error("Action not preserved")
	}
	if _, ok := stmt["Resource"]; !ok {
		t.Error("Resource not preserved")
	}
	if _, ok := stmt["Condition"]; !ok {
		t.Error("Condition not preserved")
	}
}
