package internal

import "testing"

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

func TestPrintTableEmpty(t *testing.T) {
	// Calling printTable with empty rows should print "No evaluation results."
	// We can't easily capture stdout, but we can at least call it to ensure no panic
	PrintTable([][3]string{})
}
