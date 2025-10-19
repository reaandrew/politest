package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
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
