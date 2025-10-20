package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

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

func TestParseContextTypeDefault(t *testing.T) {
	// Test that unknown types now return an error instead of defaulting to string
	_, err := ParseContextType("unknown_type")
	if err == nil {
		t.Error("ParseContextType() with unknown type should return an error, got nil")
	}
}

func TestPreprocessTemplate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "dollar variable",
			input: "$ACCOUNT_ID",
			want:  "{{.ACCOUNT_ID}}",
		},
		{
			name:  "angle bracket variable",
			input: "<ACCOUNT_NO>",
			want:  "{{.ACCOUNT_NO}}",
		},
		{
			name:  "mixed variables",
			input: "arn:aws:s3:::$BUCKET_NAME/<PREFIX>/*",
			want:  "arn:aws:s3:::{{.BUCKET_NAME}}/{{.PREFIX}}/*",
		},
		{
			name:  "no variables",
			input: "static string",
			want:  "static string",
		},
		{
			name:  "existing Go template",
			input: "{{.existing}}",
			want:  "{{.existing}}",
		},
		{
			name:  "all three formats",
			input: "$ENV_VAR and <CUSTOM_VAR> and {{.go_var}}",
			want:  "{{.ENV_VAR}} and {{.CUSTOM_VAR}} and {{.go_var}}",
		},
		{
			name:  "lowercase variable names",
			input: "$account_id and <bucket_name>",
			want:  "{{.account_id}} and {{.bucket_name}}",
		},
		{
			name:  "mixed case variable names",
			input: "$MyVar and <AnotherVar>",
			want:  "{{.MyVar}} and {{.AnotherVar}}",
		},
		{
			name:  "variable with underscores",
			input: "$AWS_ACCOUNT_ID and <DNS_HUB_DEV>",
			want:  "{{.AWS_ACCOUNT_ID}} and {{.DNS_HUB_DEV}}",
		},
		{
			name:  "variable with numbers",
			input: "$VAR123 and <ACCOUNT_NO_2>",
			want:  "{{.VAR123}} and {{.ACCOUNT_NO_2}}",
		},
		{
			name:  "dollar sign not followed by valid var",
			input: "$123 $-invalid",
			want:  "$123 $-invalid",
		},
		{
			name:  "angle brackets not valid var",
			input: "<123> <-invalid>",
			want:  "<123> <-invalid>",
		},
		{
			name:  "dollar brace variable",
			input: "${ACCOUNT_ID}",
			want:  "{{.ACCOUNT_ID}}",
		},
		{
			name:  "multiple dollar brace variables",
			input: "${BUCKET}-${REGION}-${ACCOUNT}",
			want:  "{{.BUCKET}}-{{.REGION}}-{{.ACCOUNT}}",
		},
		{
			name:  "dollar brace in ARN",
			input: "arn:aws:s3:::${BUCKET_NAME}/*",
			want:  "arn:aws:s3:::{{.BUCKET_NAME}}/*",
		},
		{
			name:  "mixed dollar formats",
			input: "$SIMPLE and ${BRACED}",
			want:  "{{.SIMPLE}} and {{.BRACED}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PreprocessTemplate(tt.input)
			if got != tt.want {
				t.Errorf("PreprocessTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderStringWithDollarVars(t *testing.T) {
	vars := map[string]any{
		"ACCOUNT_ID": "123456789012",
		"REGION":     "us-east-1",
		"BUCKET":     "my-bucket",
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "dollar variable",
			template: "$ACCOUNT_ID",
			want:     "123456789012",
		},
		{
			name:     "dollar variable in ARN",
			template: "arn:aws:s3:::$BUCKET/*",
			want:     "arn:aws:s3:::my-bucket/*",
		},
		{
			name:     "multiple dollar variables",
			template: "$BUCKET-$REGION-$ACCOUNT_ID",
			want:     "my-bucket-us-east-1-123456789012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderString(tt.template, vars)
			if got != tt.want {
				t.Errorf("RenderString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderStringWithAngleVars(t *testing.T) {
	vars := map[string]any{
		"ACCOUNT_NO": "123456789012",
		"REGION":     "us-east-1",
		"PREFIX":     "data",
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "angle variable",
			template: "<ACCOUNT_NO>",
			want:     "123456789012",
		},
		{
			name:     "angle variable in ARN",
			template: "arn:aws:iam::<ACCOUNT_NO>:role/MyRole",
			want:     "arn:aws:iam::123456789012:role/MyRole",
		},
		{
			name:     "multiple angle variables",
			template: "<PREFIX>-<REGION>-<ACCOUNT_NO>",
			want:     "data-us-east-1-123456789012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderString(tt.template, vars)
			if got != tt.want {
				t.Errorf("RenderString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderStringWithMixedVars(t *testing.T) {
	vars := map[string]any{
		"ACCOUNT_ID":  "123456789012",
		"BUCKET_NAME": "my-bucket",
		"region":      "us-east-1",
		"custom_var":  "value",
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "all three formats",
			template: "$ACCOUNT_ID-<BUCKET_NAME>-{{.region}}",
			want:     "123456789012-my-bucket-us-east-1",
		},
		{
			name:     "complex ARN with all formats",
			template: "arn:aws:s3:::<BUCKET_NAME>-$ACCOUNT_ID/{{.custom_var}}/*",
			want:     "arn:aws:s3:::my-bucket-123456789012/value/*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderString(tt.template, vars)
			if got != tt.want {
				t.Errorf("RenderString() = %q, want %q", got, tt.want)
			}
		})
	}
}
