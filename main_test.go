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
