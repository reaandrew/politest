package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPrintVersion(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call PrintVersion
	PrintVersion()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that output contains expected version information
	if !strings.Contains(output, "politest") {
		t.Errorf("Version output missing 'politest': %s", output)
	}

	if !strings.Contains(output, "commit:") {
		t.Errorf("Version output missing 'commit:': %s", output)
	}

	if !strings.Contains(output, "built:") {
		t.Errorf("Version output missing 'built:': %s", output)
	}

	if !strings.Contains(output, "go version:") {
		t.Errorf("Version output missing 'go version:': %s", output)
	}

	// Check that default values are present
	if !strings.Contains(output, "dev") {
		t.Errorf("Version output should contain default 'dev' version: %s", output)
	}

	if !strings.Contains(output, "unknown") {
		t.Errorf("Version output should contain default 'unknown' values: %s", output)
	}

	// Verify format has proper indentation
	lines := strings.Split(output, "\n")
	if len(lines) < 4 {
		t.Errorf("Expected at least 4 lines of output, got %d", len(lines))
	}

	// Check that go version is runtime.Version()
	if !strings.Contains(output, "go1.") {
		t.Errorf("Expected output to contain Go version starting with 'go1.': %s", output)
	}
}

func TestVersionVariablesDefaults(t *testing.T) {
	// Test that version variables have correct default values
	if version != "dev" {
		t.Errorf("Expected version to be 'dev', got '%s'", version)
	}

	if gitCommit != "unknown" {
		t.Errorf("Expected gitCommit to be 'unknown', got '%s'", gitCommit)
	}

	if buildDate != "unknown" {
		t.Errorf("Expected buildDate to be 'unknown', got '%s'", buildDate)
	}

	// goVersion should be runtime.Version()
	if !strings.HasPrefix(goVersion, "go") {
		t.Errorf("Expected goVersion to start with 'go', got '%s'", goVersion)
	}
}

func TestVersionFlag(t *testing.T) {
	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", "politest-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("politest-test")

	// Run with --version flag
	cmd := exec.Command("./politest-test", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run --version: %v", err)
	}

	outputStr := string(output)

	// Check that output contains expected version information
	if !strings.Contains(outputStr, "politest") {
		t.Errorf("Version output missing 'politest': %s", outputStr)
	}

	if !strings.Contains(outputStr, "commit:") {
		t.Errorf("Version output missing 'commit:': %s", outputStr)
	}

	if !strings.Contains(outputStr, "built:") {
		t.Errorf("Version output missing 'built:': %s", outputStr)
	}

	if !strings.Contains(outputStr, "go version:") {
		t.Errorf("Version output missing 'go version:': %s", outputStr)
	}

	// Verify exit code is 0
	if cmd.ProcessState.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", cmd.ProcessState.ExitCode())
	}
}

func TestMissingScenarioFlag(t *testing.T) {
	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", "politest-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("politest-test")

	// Run without --scenario flag
	cmd := exec.Command("./politest-test")
	output, err := cmd.CombinedOutput()

	// Should exit with non-zero code
	if err == nil {
		t.Error("Expected non-zero exit code when --scenario is missing")
	}

	outputStr := string(output)

	// Check for error message
	if !strings.Contains(outputStr, "scenario") {
		t.Errorf("Error message should mention 'scenario': %s", outputStr)
	}
}

func TestUnknownArguments(t *testing.T) {
	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", "politest-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("politest-test")

	// Run with unknown positional arguments
	cmd := exec.Command("./politest-test", "--scenario", "test.yml", "unknown", "args")
	output, err := cmd.CombinedOutput()

	// Should exit with non-zero code
	if err == nil {
		t.Error("Expected non-zero exit code when unknown arguments are provided")
	}

	outputStr := string(output)

	// Check for error message about unknown arguments
	if !strings.Contains(outputStr, "unknown arguments") {
		t.Errorf("Error message should mention 'unknown arguments': %s", outputStr)
	}
}

func TestRunMissingScenario(t *testing.T) {
	// Test run() with empty scenario path
	var buf bytes.Buffer
	err := run("", "", false, false, false, &buf)
	if err == nil {
		t.Error("Expected error when scenario path is empty")
	}

	if !strings.Contains(err.Error(), "missing --scenario") {
		t.Errorf("Expected error message to contain 'missing --scenario', got: %v", err)
	}
}

func TestRunInvalidScenarioFile(t *testing.T) {
	// Test run() with non-existent scenario file
	var buf bytes.Buffer
	err := run("/nonexistent/scenario.yml", "", false, false, false, &buf)
	if err == nil {
		t.Error("Expected error when scenario file does not exist")
	}
}

func TestRunConflictingPolicyFields(t *testing.T) {
	// Create a temporary scenario file with conflicting policy fields
	tmpDir := t.TempDir()
	scenarioPath := tmpDir + "/scenario.yml"

	scenarioContent := `policy_json: "policy.json"
policy_template: "policy.tpl"
tests:
  - action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"
`

	if err := os.WriteFile(scenarioPath, []byte(scenarioContent), 0600); err != nil {
		t.Fatalf("Failed to create scenario file: %v", err)
	}

	var buf bytes.Buffer
	err := run(scenarioPath, "", false, false, false, &buf)
	if err == nil {
		t.Error("Expected error when both policy_json and policy_template are specified")
	}

	if !strings.Contains(err.Error(), "provide only one") {
		t.Errorf("Expected error about conflicting fields, got: %v", err)
	}
}

func TestRunMissingPolicyFields(t *testing.T) {
	// Create a temporary scenario file without policy fields
	tmpDir := t.TempDir()
	scenarioPath := tmpDir + "/scenario.yml"

	scenarioContent := `tests:
  - action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"
`

	if err := os.WriteFile(scenarioPath, []byte(scenarioContent), 0600); err != nil {
		t.Fatalf("Failed to create scenario file: %v", err)
	}

	var buf bytes.Buffer
	err := run(scenarioPath, "", false, false, false, &buf)
	if err == nil {
		t.Error("Expected error when neither policy_json nor policy_template is specified")
	}

	if !strings.Contains(err.Error(), "policy_json") && !strings.Contains(err.Error(), "policy_template") {
		t.Errorf("Expected error about missing policy fields, got: %v", err)
	}
}

func TestRunEmptyTests(t *testing.T) {
	// Create a temporary scenario file with no tests
	tmpDir := t.TempDir()
	scenarioPath := tmpDir + "/scenario.yml"
	policyPath := tmpDir + "/policy.json"

	policyContent := `{"Version": "2012-10-17", "Statement": []}`
	scenarioContent := `policy_json: "policy.json"
tests: []
`

	if err := os.WriteFile(policyPath, []byte(policyContent), 0600); err != nil {
		t.Fatalf("Failed to create policy file: %v", err)
	}

	if err := os.WriteFile(scenarioPath, []byte(scenarioContent), 0600); err != nil {
		t.Fatalf("Failed to create scenario file: %v", err)
	}

	var buf bytes.Buffer
	err := run(scenarioPath, "", false, false, false, &buf)
	if err == nil {
		t.Error("Expected error when tests array is empty")
	}

	if !strings.Contains(err.Error(), "tests") {
		t.Errorf("Expected error about empty tests array, got: %v", err)
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantScenario string
		wantSave     string
		wantNoAssert bool
		wantNoWarn   bool
		wantVersion  bool
		wantDebug    bool
		wantErr      bool
	}{
		{
			name:         "all flags",
			args:         []string{"--scenario", "test.yml", "--save", "out.json", "--no-assert", "--no-warn", "--version", "--debug"},
			wantScenario: "test.yml",
			wantSave:     "out.json",
			wantNoAssert: true,
			wantNoWarn:   true,
			wantVersion:  true,
			wantDebug:    true,
		},
		{
			name:         "only scenario",
			args:         []string{"--scenario", "test.yml"},
			wantScenario: "test.yml",
		},
		{
			name:         "short flags",
			args:         []string{"-scenario", "test.yml", "-version"},
			wantScenario: "test.yml",
			wantVersion:  true,
		},
		{
			name: "no flags",
			args: []string{},
		},
		{
			name:         "debug flag",
			args:         []string{"--scenario", "test.yml", "--debug"},
			wantScenario: "test.yml",
			wantDebug:    true,
		},
		{
			name:      "debug without scenario",
			args:      []string{"--debug"},
			wantDebug: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, _, err := parseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if flags.scenarioPath != tt.wantScenario {
				t.Errorf("scenarioPath = %v, want %v", flags.scenarioPath, tt.wantScenario)
			}
			if flags.savePath != tt.wantSave {
				t.Errorf("savePath = %v, want %v", flags.savePath, tt.wantSave)
			}
			if flags.noAssert != tt.wantNoAssert {
				t.Errorf("noAssert = %v, want %v", flags.noAssert, tt.wantNoAssert)
			}
			if flags.noWarn != tt.wantNoWarn {
				t.Errorf("noWarn = %v, want %v", flags.noWarn, tt.wantNoWarn)
			}
			if flags.showVersion != tt.wantVersion {
				t.Errorf("showVersion = %v, want %v", flags.showVersion, tt.wantVersion)
			}
			if flags.debug != tt.wantDebug {
				t.Errorf("debug = %v, want %v", flags.debug, tt.wantDebug)
			}
		})
	}
}

func TestParseFlagsWithRemainingArgs(t *testing.T) {
	flags, remaining, err := parseFlags([]string{"--scenario", "test.yml", "extra", "args"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if flags.scenarioPath != "test.yml" {
		t.Errorf("Expected scenario 'test.yml', got %v", flags.scenarioPath)
	}

	if len(remaining) != 2 {
		t.Errorf("Expected 2 remaining args, got %d", len(remaining))
	}

	if remaining[0] != "extra" || remaining[1] != "args" {
		t.Errorf("Expected remaining args [extra args], got %v", remaining)
	}
}

func TestValidateArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args - valid",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "one arg - invalid",
			args:    []string{"extra"},
			wantErr: true,
		},
		{
			name:    "multiple args - invalid",
			args:    []string{"extra", "args"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateArgs() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && !strings.Contains(err.Error(), "unknown arguments") {
				t.Errorf("Expected error to contain 'unknown arguments', got: %v", err)
			}
		})
	}
}

func TestRealMainVersion(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := realMain([]string{"--version"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	if !strings.Contains(output, "politest") {
		t.Errorf("Expected version output to contain 'politest', got: %s", output)
	}
}

func TestRealMainUnknownArgs(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := realMain([]string{"--scenario", "test.yml", "extra"})

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	if !strings.Contains(output, "unknown arguments") {
		t.Errorf("Expected error about unknown arguments, got: %s", output)
	}
}

func TestRealMainMissingScenario(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := realMain([]string{})

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	if !strings.Contains(output, "missing --scenario") {
		t.Errorf("Expected error about missing scenario, got: %s", output)
	}
}

func TestRealMainInvalidScenario(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := realMain([]string{"--scenario", "/nonexistent/file.yml"})

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
}

func TestRunDebugOutputEnabled(t *testing.T) {
	// Create a minimal valid scenario with policy_json
	tmpDir := t.TempDir()
	scenarioPath := tmpDir + "/scenario.yml"
	policyPath := tmpDir + "/policy.json"

	policyContent := `{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": "s3:GetObject",
    "Resource": "*"
  }]
}`

	scenarioContent := `policy_json: "policy.json"
tests:
  - action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"
`

	if err := os.WriteFile(policyPath, []byte(policyContent), 0600); err != nil {
		t.Fatalf("Failed to create policy file: %v", err)
	}

	if err := os.WriteFile(scenarioPath, []byte(scenarioContent), 0600); err != nil {
		t.Fatalf("Failed to create scenario file: %v", err)
	}

	// Use bytes.Buffer to capture debug output
	var debugBuf bytes.Buffer

	// Run with debug=true (will fail at AWS call, but we only care about debug output)
	_ = run(scenarioPath, "", false, false, true, &debugBuf)

	output := debugBuf.String()

	// Verify debug output is present
	if !strings.Contains(output, "🔍 DEBUG: Loading scenario from:") {
		t.Errorf("Expected debug output for scenario loading, got: %s", output)
	}

	if !strings.Contains(output, "🔍 DEBUG: Loading policy from:") {
		t.Errorf("Expected debug output for policy loading, got: %s", output)
	}

	if !strings.Contains(output, "🔍 DEBUG: Rendered policy (minified):") {
		t.Errorf("Expected debug output for rendered policy, got: %s", output)
	}
}

func TestRunDebugOutputDisabled(t *testing.T) {
	// Create a minimal valid scenario with policy_json
	tmpDir := t.TempDir()
	scenarioPath := tmpDir + "/scenario.yml"
	policyPath := tmpDir + "/policy.json"

	policyContent := `{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": "s3:GetObject",
    "Resource": "*"
  }]
}`

	scenarioContent := `policy_json: "policy.json"
tests:
  - action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"
`

	if err := os.WriteFile(policyPath, []byte(policyContent), 0600); err != nil {
		t.Fatalf("Failed to create policy file: %v", err)
	}

	if err := os.WriteFile(scenarioPath, []byte(scenarioContent), 0600); err != nil {
		t.Fatalf("Failed to create scenario file: %v", err)
	}

	// Use bytes.Buffer to capture debug output
	var debugBuf bytes.Buffer

	// Run with debug=false (will fail at AWS call, but we only care about debug output)
	_ = run(scenarioPath, "", false, false, false, &debugBuf)

	output := debugBuf.String()

	// Verify NO debug output is present
	if strings.Contains(output, "🔍 DEBUG:") {
		t.Errorf("Expected no debug output when debug=false, but found: %s", output)
	}
}

func TestRunDebugWithTemplate(t *testing.T) {
	// Create scenario with policy_template and vars
	tmpDir := t.TempDir()
	scenarioPath := tmpDir + "/scenario.yml"
	policyPath := tmpDir + "/policy.json.tpl"
	varsPath := tmpDir + "/vars.yml"

	policyContent := `{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": "{{.action}}",
    "Resource": "{{.resource}}"
  }]
}`

	varsContent := `action: "s3:GetObject"
resource: "arn:aws:s3:::mybucket/*"
`

	scenarioContent := `policy_template: "policy.json.tpl"
vars_file: "vars.yml"
tests:
  - action: "s3:GetObject"
    resource: "arn:aws:s3:::mybucket/*"
    expect: "allowed"
`

	if err := os.WriteFile(policyPath, []byte(policyContent), 0600); err != nil {
		t.Fatalf("Failed to create policy template: %v", err)
	}

	if err := os.WriteFile(varsPath, []byte(varsContent), 0600); err != nil {
		t.Fatalf("Failed to create vars file: %v", err)
	}

	if err := os.WriteFile(scenarioPath, []byte(scenarioContent), 0600); err != nil {
		t.Fatalf("Failed to create scenario file: %v", err)
	}

	// Use bytes.Buffer to capture debug output
	var debugBuf bytes.Buffer

	// Run with debug=true
	_ = run(scenarioPath, "", false, false, true, &debugBuf)

	output := debugBuf.String()

	// Verify template-specific debug output
	if !strings.Contains(output, "🔍 DEBUG: Loading variables from:") {
		t.Errorf("Expected debug output for variables loading, got: %s", output)
	}

	if !strings.Contains(output, "🔍 DEBUG: Variables available:") {
		t.Errorf("Expected debug output for variables list, got: %s", output)
	}

	if !strings.Contains(output, "🔍 DEBUG: Loading policy template from:") {
		t.Errorf("Expected debug output for policy template loading, got: %s", output)
	}
}

func TestRunDebugWithResourcePolicy(t *testing.T) {
	// Create scenario with resource policy to test resource policy debug output
	tmpDir := t.TempDir()
	scenarioPath := tmpDir + "/scenario.yml"
	policyPath := tmpDir + "/policy.json"
	resourcePolicyPath := tmpDir + "/resource-policy.json"

	policyContent := `{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": "s3:*",
    "Resource": "*"
  }]
}`

	resourcePolicyContent := `{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {"AWS": "*"},
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::bucket/*"
  }]
}`

	scenarioContent := `policy_json: "policy.json"
resource_policy_json: "resource-policy.json"
caller_arn: "arn:aws:iam::123456789012:user/test"
tests:
  - action: "s3:GetObject"
    resource: "arn:aws:s3:::bucket/*"
    expect: "allowed"
`

	if err := os.WriteFile(policyPath, []byte(policyContent), 0600); err != nil {
		t.Fatalf("Failed to create policy file: %v", err)
	}

	if err := os.WriteFile(resourcePolicyPath, []byte(resourcePolicyContent), 0600); err != nil {
		t.Fatalf("Failed to create resource policy file: %v", err)
	}

	if err := os.WriteFile(scenarioPath, []byte(scenarioContent), 0600); err != nil {
		t.Fatalf("Failed to create scenario file: %v", err)
	}

	// Use bytes.Buffer to capture debug output
	var debugBuf bytes.Buffer

	// Run with debug=true
	_ = run(scenarioPath, "", false, false, true, &debugBuf)

	output := debugBuf.String()

	// Verify resource policy debug output
	if !strings.Contains(output, "🔍 DEBUG: Loading resource policy from:") {
		t.Errorf("Expected debug output for resource policy loading, got: %s", output)
	}

	if !strings.Contains(output, "🔍 DEBUG: Rendered resource policy (minified):") {
		t.Errorf("Expected debug output for rendered resource policy, got: %s", output)
	}
}
