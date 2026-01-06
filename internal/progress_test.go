package internal

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewConsoleProgress(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	if progress == nil {
		t.Fatal("NewConsoleProgress() returned nil")
	}
	if progress.writer != &buf {
		t.Error("writer not set correctly")
	}
	if progress.startTime.IsZero() {
		t.Error("startTime should be set")
	}
}

func TestConsoleProgressSetStatus(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	progress.SetStatus("Testing status")

	if progress.status != "Testing status" {
		t.Errorf("status = %q, want 'Testing status'", progress.status)
	}

	output := buf.String()
	if !strings.Contains(output, "Testing status") {
		t.Errorf("output = %q, want to contain 'Testing status'", output)
	}
}

func TestConsoleProgressSetStatusWithBar(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// First set progress to create bar
	progress.SetProgress(1, 10)

	// Then set status
	progress.SetStatus("New status")

	if progress.status != "New status" {
		t.Errorf("status = %q, want 'New status'", progress.status)
	}
}

func TestConsoleProgressSetProgress(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Test with valid progress
	progress.SetProgress(5, 10)

	if progress.total != 10 {
		t.Errorf("total = %d, want 10", progress.total)
	}
	if progress.bar == nil {
		t.Error("bar should be created after SetProgress")
	}
}

func TestConsoleProgressSetProgressZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Test with zero total - should be no-op
	progress.SetProgress(0, 0)

	if progress.total != 0 {
		t.Errorf("total = %d, want 0 (unchanged)", progress.total)
	}
	if progress.bar != nil {
		t.Error("bar should not be created for zero total")
	}
}

func TestConsoleProgressSetProgressNegativeTotal(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Test with negative total - should be no-op
	progress.SetProgress(0, -1)

	if progress.total != 0 {
		t.Errorf("total = %d, want 0 (unchanged)", progress.total)
	}
	if progress.bar != nil {
		t.Error("bar should not be created for negative total")
	}
}

func TestConsoleProgressSetProgressChangingTotal(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Set initial progress
	progress.SetProgress(1, 10)
	firstBar := progress.bar

	// Change total - should create new bar
	progress.SetProgress(1, 20)

	if progress.total != 20 {
		t.Errorf("total = %d, want 20", progress.total)
	}
	// Note: progressbar.ProgressBar doesn't expose an easy way to compare
	// but the bar should be recreated. We verify the total changed.
	_ = firstBar
}

func TestConsoleProgressDone(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	progress.Done("All done")

	output := buf.String()
	if !strings.Contains(output, "All done") {
		t.Errorf("output = %q, want to contain 'All done'", output)
	}
	// Should contain checkmark
	if !strings.Contains(output, "✓") {
		t.Errorf("output = %q, want to contain '✓'", output)
	}
}

func TestConsoleProgressDoneWithBar(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Create bar first
	progress.SetProgress(5, 10)

	progress.Done("Completed")

	output := buf.String()
	if !strings.Contains(output, "Completed") {
		t.Errorf("output = %q, want to contain 'Completed'", output)
	}
}

func TestConsoleProgressError(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	progress.Error("Something failed")

	output := buf.String()
	if !strings.Contains(output, "Something failed") {
		t.Errorf("output = %q, want to contain 'Something failed'", output)
	}
	// Should contain X mark
	if !strings.Contains(output, "✗") {
		t.Errorf("output = %q, want to contain '✗'", output)
	}
}

func TestConsoleProgressErrorWithBar(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Create bar first
	progress.SetProgress(5, 10)

	progress.Error("Error occurred")

	output := buf.String()
	if !strings.Contains(output, "Error occurred") {
		t.Errorf("output = %q, want to contain 'Error occurred'", output)
	}
}

func TestConsoleProgressStop(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Create bar first
	progress.SetProgress(5, 10)

	// Should not panic
	progress.Stop()
}

func TestConsoleProgressStopNoBar(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Should not panic even without bar
	progress.Stop()
}

func TestConsoleProgressStart(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Start should not panic
	progress.Start()

	// Bar should not be created yet
	if progress.bar != nil {
		t.Error("bar should not be created by Start()")
	}
}

func TestProgressReporterInterface(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Verify ConsoleProgress implements ProgressReporter
	var reporter ProgressReporter = progress
	reporter.SetStatus("test")
	reporter.SetProgress(1, 2)
}

func TestConsoleProgressStatusWithoutBar(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Set status multiple times without bar
	progress.SetStatus("Status 1")
	progress.SetStatus("Status 2")
	progress.SetStatus("Status 3")

	output := buf.String()
	// Last status should be visible
	if !strings.Contains(output, "Status 3") {
		t.Errorf("output = %q, want to contain 'Status 3'", output)
	}
}

func TestConsoleProgressElapsedTime(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Let some time pass
	progress.Done("Done")

	output := buf.String()
	// Should contain "took" with some duration
	if !strings.Contains(output, "took") {
		t.Errorf("output = %q, want to contain 'took'", output)
	}
}

func TestConsoleProgressSetProgressWithStatus(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	// Set status first
	progress.SetStatus("Processing")

	// Then set progress - should use status as bar description
	progress.SetProgress(1, 10)

	if progress.total != 10 {
		t.Errorf("total = %d, want 10", progress.total)
	}
	if progress.status != "Processing" {
		t.Errorf("status = %q, want 'Processing'", progress.status)
	}
}

func TestConsoleProgressFullWorkflow(t *testing.T) {
	var buf bytes.Buffer
	progress := NewConsoleProgress(&buf)

	progress.Start()
	progress.SetStatus("Starting...")
	progress.SetProgress(0, 5)
	progress.SetProgress(1, 5)
	progress.SetStatus("In progress...")
	progress.SetProgress(2, 5)
	progress.SetProgress(3, 5)
	progress.SetProgress(4, 5)
	progress.SetProgress(5, 5)
	progress.Done("All tasks complete")

	output := buf.String()
	if !strings.Contains(output, "All tasks complete") {
		t.Errorf("output = %q, want to contain 'All tasks complete'", output)
	}
}
