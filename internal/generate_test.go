package internal

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateGenerateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     GenerateConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: GenerateConfig{
				URL:     "https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazons3.html",
				BaseURL: "https://api.openai.com",
				Model:   "gpt-4",
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			cfg: GenerateConfig{
				BaseURL: "https://api.openai.com",
				Model:   "gpt-4",
			},
			wantErr: true,
			errMsg:  "--url is required",
		},
		{
			name: "invalid URL (not AWS)",
			cfg: GenerateConfig{
				URL:     "https://example.com/not-aws",
				BaseURL: "https://api.openai.com",
				Model:   "gpt-4",
			},
			wantErr: true,
			errMsg:  "invalid URL",
		},
		{
			name: "missing BaseURL",
			cfg: GenerateConfig{
				URL:   "https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazons3.html",
				Model: "gpt-4",
			},
			wantErr: true,
			errMsg:  "--base-url is required",
		},
		{
			name: "missing Model",
			cfg: GenerateConfig{
				URL:     "https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazons3.html",
				BaseURL: "https://api.openai.com",
			},
			wantErr: true,
			errMsg:  "--model is required",
		},
		{
			name:    "all fields empty",
			cfg:     GenerateConfig{},
			wantErr: true,
			errMsg:  "--url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGenerateConfig(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("ValidateGenerateConfig() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateGenerateConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateGenerateConfig() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGenerateConfigDefaults(t *testing.T) {
	cfg := GenerateConfig{
		URL:     "https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazons3.html",
		BaseURL: "https://api.openai.com",
		Model:   "gpt-4",
	}

	// Test default values
	if cfg.Concurrency != 0 {
		t.Errorf("Default Concurrency = %d, want 0", cfg.Concurrency)
	}
	if cfg.NoEnrich != false {
		t.Error("Default NoEnrich should be false")
	}
	if cfg.Quiet != false {
		t.Error("Default Quiet should be false")
	}
	if cfg.GenerateSCP != false {
		t.Error("Default GenerateSCP should be false")
	}
}

func TestGenerateOutput(t *testing.T) {
	output := GenerateOutput{
		ScrapedData: &ScrapedIAMData{
			ServiceName:   "Amazon S3",
			ServicePrefix: "s3",
		},
		Policy:      json.RawMessage(`{"Version": "2012-10-17"}`),
		PolicyFile:  "/tmp/policy.json",
		ScrapedFile: "/tmp/scraped.json",
		DocsFile:    "/tmp/docs.md",
		ServiceName: "s3",
	}

	if output.ServiceName != "s3" {
		t.Errorf("ServiceName = %q, want s3", output.ServiceName)
	}
	if output.ScrapedData == nil {
		t.Error("ScrapedData should not be nil")
	}
	if output.ScrapedData.ServicePrefix != "s3" {
		t.Errorf("ScrapedData.ServicePrefix = %q, want s3", output.ScrapedData.ServicePrefix)
	}
}

func TestRunGenerateInvalidURLValidation(t *testing.T) {
	// RunGenerate should fail with invalid URL due to ScrapeIAMDocumentation validation
	var buf bytes.Buffer
	cfg := GenerateConfig{
		URL:       "https://example.com/not-aws",
		BaseURL:   "http://unused",
		Model:     "test-model",
		OutputDir: t.TempDir(),
		Quiet:     true,
	}

	_, err := RunGenerate(cfg, &buf)
	if err == nil {
		t.Error("RunGenerate() expected error for invalid URL, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid URL") {
		t.Errorf("RunGenerate() error = %v, want error about invalid URL", err)
	}
}

// Note: Integration tests for RunGenerate require mocking the entire HTTP layer
// including the AWS documentation URL validation in ScrapeIAMDocumentation.
// For comprehensive testing, use integration tests with real AWS documentation pages.
