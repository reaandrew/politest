package internal

import (
	"os"
	"path/filepath"
	"testing"
)

// mockExiter is a test double for the Exiter interface
type mockExiter struct {
	exitCode int
	called   bool
}

func (m *mockExiter) Exit(code int) {
	m.exitCode = code
	m.called = true
	// Don't actually exit in tests
}

func TestAwsString(t *testing.T) {
	tests := []struct {
		name  string
		input *string
		want  string
	}{
		{
			name:  "non-nil string",
			input: StrPtr("test"),
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
			got := AwsString(tt.input)
			if got != tt.want {
				t.Errorf("AwsString() = %v, want %v", got, tt.want)
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
			got := IfEmpty(tt.s, tt.fallback)
			if got != tt.want {
				t.Errorf("IfEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStrPtr(t *testing.T) {
	s := "test"
	ptr := StrPtr(s)
	if ptr == nil {
		t.Fatal("StrPtr() returned nil")
		return
	}
	if *ptr != s {
		t.Errorf("StrPtr() = %v, want %v", *ptr, s)
	}
}

func TestMustAbs(t *testing.T) {
	// Test with current directory
	result := MustAbs(".")
	if result == "" {
		t.Error("MustAbs() returned empty string")
	}

	// Should return absolute path
	if !filepath.IsAbs(result) {
		t.Errorf("MustAbs() returned relative path: %v", result)
	}
}

func TestMustAbsJoin(t *testing.T) {
	base := "/tmp"
	rel := "subdir/file.txt"

	result := MustAbsJoin(base, rel)

	if !filepath.IsAbs(result) {
		t.Errorf("MustAbsJoin() returned relative path: %v", result)
	}

	if !contains(result, "subdir") {
		t.Error("MustAbsJoin() did not join paths correctly")
	}
}

func TestCheck(t *testing.T) {
	// Check() calls Die() on error, which exits the program
	// We can only test the nil case
	Check(nil)

	// If we get here, it worked correctly
}

func TestDieWithMockExiter(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	// Set mock exiter
	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	// Call die
	Die("test error")

	// Verify Exit was called with code 1
	if !mockExit.called {
		t.Error("Die() did not call GlobalExiter.Exit()")
	}
	if mockExit.exitCode != 1 {
		t.Errorf("Die() called Exit with code %d, want 1", mockExit.exitCode)
	}
}

func TestCheckWithError(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	// Set mock exiter
	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	// Call check with an error
	Check(os.ErrNotExist)

	// Verify Exit was called
	if !mockExit.called {
		t.Error("Check() did not call GlobalExiter.Exit() on error")
	}
	if mockExit.exitCode != 1 {
		t.Errorf("Check() called Exit with code %d, want 1", mockExit.exitCode)
	}
}

func TestWarnSCPSimulation(t *testing.T) {
	// This function just prints to stderr, we can call it to ensure it doesn't panic
	WarnSCPSimulation()
	// If we get here without panic, test passes
}

// Helper function for tests
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
