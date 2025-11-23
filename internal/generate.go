package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// GenerateConfig holds configuration for the generate command
type GenerateConfig struct {
	URL         string // AWS documentation URL
	BaseURL     string // LLM API base URL
	APIKey      string // LLM API key
	Model       string // LLM model name
	OutputDir   string // Output directory for generated files
	NoEnrich    bool   // Skip enrichment step
	Quiet       bool   // Suppress progress output
	UserPrompt  string // User's custom requirements/constraints
	Concurrency int    // Number of parallel batch requests (default 3)
}

// GenerateOutput holds the output of the generate command
type GenerateOutput struct {
	ScrapedData *ScrapedIAMData `json:"scraped_data"`
	Policy      json.RawMessage `json:"policy"`
	PolicyFile  string          `json:"policy_file"`
	ScrapedFile string          `json:"scraped_file"`
	DocsFile    string          `json:"docs_file"`
	ServiceName string          `json:"service_name"`
}

// RunGenerate executes the generate command
func RunGenerate(cfg GenerateConfig, writer io.Writer) (*GenerateOutput, error) {
	// Create progress reporter
	var progress ProgressReporter
	var consoleProgress *ConsoleProgress
	if !cfg.Quiet {
		consoleProgress = NewConsoleProgress(writer)
		consoleProgress.Start()
		progress = consoleProgress
	}

	// Step 1: Scrape the AWS documentation
	scrapedData, err := ScrapeIAMDocumentation(cfg.URL, progress)
	if err != nil {
		if consoleProgress != nil {
			consoleProgress.Error(fmt.Sprintf("Scraping failed: %v", err))
		}
		return nil, fmt.Errorf("failed to scrape documentation: %w", err)
	}

	if progress != nil {
		progress.SetStatus(fmt.Sprintf("Found %d actions for %s", len(scrapedData.Actions), scrapedData.ServicePrefix))
	}

	// Step 2: Create LLM client and generate policy
	llmClient := NewLLMClient(cfg.BaseURL, cfg.APIKey, cfg.Model)

	// Optional: Enrich action descriptions with security context
	if !cfg.NoEnrich {
		_ = llmClient.EnrichActionDescriptions(scrapedData, progress)
	}

	// Set concurrency (default to 3 if not specified)
	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 3
	}

	// Generate the security-focused policy
	policyJSON, err := llmClient.GenerateSecurityPolicy(scrapedData, progress, cfg.UserPrompt, concurrency)
	if err != nil {
		if consoleProgress != nil {
			consoleProgress.Error(fmt.Sprintf("Policy generation failed: %v", err))
		}
		return nil, fmt.Errorf("failed to generate policy: %w", err)
	}

	// Step 3: Write output files
	if progress != nil {
		progress.SetStatus("Writing output files...")
	}

	output := &GenerateOutput{
		ScrapedData: scrapedData,
		ServiceName: scrapedData.ServicePrefix,
	}

	// Parse policy JSON for pretty-printing
	var policyData any
	if err := json.Unmarshal([]byte(policyJSON), &policyData); err != nil {
		return nil, fmt.Errorf("invalid policy JSON: %w", err)
	}
	prettyPolicy, _ := json.MarshalIndent(policyData, "", "  ")
	output.Policy = prettyPolicy

	// Determine output directory
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = "."
	}
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write scraped data file
	scrapedFileName := fmt.Sprintf("%s-iam-reference.json", scrapedData.ServicePrefix)
	scrapedFilePath := filepath.Join(outputDir, scrapedFileName)
	scrapedJSON, _ := json.MarshalIndent(scrapedData, "", "  ")
	if err := os.WriteFile(scrapedFilePath, scrapedJSON, 0600); err != nil {
		return nil, fmt.Errorf("failed to write scraped data: %w", err)
	}
	output.ScrapedFile = scrapedFilePath

	// Write policy file
	policyFileName := fmt.Sprintf("%s-full-access-policy.json", scrapedData.ServicePrefix)
	policyFilePath := filepath.Join(outputDir, policyFileName)
	if err := os.WriteFile(policyFilePath, prettyPolicy, 0600); err != nil {
		return nil, fmt.Errorf("failed to write policy: %w", err)
	}
	output.PolicyFile = policyFilePath

	// Generate documentation for the policy
	if progress != nil {
		progress.SetStatus("Generating policy documentation...")
	}
	docsMarkdown, err := llmClient.GeneratePolicyDocumentation(scrapedData, string(prettyPolicy))
	if err != nil {
		// Non-fatal - continue without docs
		if progress != nil {
			progress.SetStatus("Documentation generation skipped (error)")
		}
	} else {
		docsFileName := fmt.Sprintf("%s-policy-documentation.md", scrapedData.ServicePrefix)
		docsFilePath := filepath.Join(outputDir, docsFileName)
		if err := os.WriteFile(docsFilePath, []byte(docsMarkdown), 0600); err != nil {
			return nil, fmt.Errorf("failed to write documentation: %w", err)
		}
		output.DocsFile = docsFilePath
	}

	// Complete
	if consoleProgress != nil {
		consoleProgress.Done(fmt.Sprintf("Generated policy for %s", scrapedData.ServiceName))
	}

	// Print summary
	if !cfg.Quiet {
		fmt.Fprintf(writer, "\n\033[1mGeneration Complete\033[0m\n")
		fmt.Fprintf(writer, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(writer, "Service:           %s\n", scrapedData.ServiceName)
		fmt.Fprintf(writer, "Service Prefix:    %s\n", scrapedData.ServicePrefix)
		fmt.Fprintf(writer, "Actions Found:     %d\n", len(scrapedData.Actions))
		fmt.Fprintf(writer, "Condition Keys:    %d\n", len(scrapedData.ConditionKeys))
		fmt.Fprintf(writer, "Resource Types:    %d\n", len(scrapedData.ResourceTypes))
		fmt.Fprintf(writer, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(writer, "\033[32mOutput Files:\033[0m\n")
		fmt.Fprintf(writer, "  IAM Reference:   %s\n", scrapedFilePath)
		fmt.Fprintf(writer, "  Policy:          %s\n", policyFilePath)
		if output.DocsFile != "" {
			fmt.Fprintf(writer, "  Documentation:   %s\n", output.DocsFile)
		}
		fmt.Fprintf(writer, "\n")

		// Show action summary by access level
		accessLevels := make(map[string]int)
		for _, action := range scrapedData.Actions {
			level := action.AccessLevel
			if level == "" {
				level = "Unknown"
			}
			accessLevels[level]++
		}
		fmt.Fprintf(writer, "\033[1mActions by Access Level:\033[0m\n")
		for level, count := range accessLevels {
			fmt.Fprintf(writer, "  %-20s %d\n", level+":", count)
		}
	}

	return output, nil
}

// ValidateGenerateConfig validates the generate configuration
func ValidateGenerateConfig(cfg GenerateConfig) error {
	if cfg.URL == "" {
		return fmt.Errorf("--url is required: AWS documentation URL")
	}
	if !strings.Contains(cfg.URL, "docs.aws.amazon.com/service-authorization") {
		return fmt.Errorf("invalid URL: must be an AWS service authorization reference page\nExample: https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazonbedrock.html")
	}
	if cfg.BaseURL == "" {
		return fmt.Errorf("--base-url is required: OpenAI-compatible API base URL")
	}
	if cfg.Model == "" {
		return fmt.Errorf("--model is required: LLM model name")
	}
	return nil
}
