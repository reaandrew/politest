// cmd: go run . --scenario scenarios/athena_primary.yml --save /tmp/resp.json
// cmd: go run . generate --url https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazonbedrock.html --base-url http://localhost:3000 --model gpt-4 --api-key xxx
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"politest/internal"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

// Build-time variables injected via -ldflags
var (
	version   = "dev"             // Semantic version (e.g., "v1.0.0")
	gitCommit = "unknown"         // Full git commit SHA
	buildDate = "unknown"         // ISO 8601 build timestamp
	goVersion = runtime.Version() // Go compiler version
)

// PrintVersion outputs version information to stdout
func PrintVersion() {
	fmt.Printf("politest %s\n", version)
	fmt.Printf("  commit:     %s\n", gitCommit)
	fmt.Printf("  built:      %s\n", buildDate)
	fmt.Printf("  go version: %s\n", goVersion)
}

// simulationPrep holds the prepared simulation data before AWS execution
type simulationPrep struct {
	scenario            *internal.Scenario
	policyJSON          string
	permissionsBoundary string
	resourcePolicyJSON  string
	variables           map[string]any
	absScenarioPath     string
	sourceMap           *internal.PolicySourceMap
}

// prepareSimulation loads and prepares all data for simulation WITHOUT contacting AWS
// This function is AWS-free and safe for unit testing
func prepareSimulation(scenarioPath string, noWarn, debug, strictPolicy bool, debugWriter io.Writer) (*simulationPrep, error) {
	if scenarioPath == "" {
		return nil, fmt.Errorf("missing --scenario\nUsage: politest --scenario <path> [--save <path>] [--no-assert] [--no-warn] [--debug]")
	}

	absScenario, err := filepath.Abs(scenarioPath)
	if err != nil {
		return nil, err
	}

	if debug {
		fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading scenario from: %s\n", absScenario)
	}

	scen, err := internal.LoadScenarioWithExtends(absScenario)
	if err != nil {
		return nil, err
	}

	if debug && scen.Extends != "" {
		fmt.Fprintf(debugWriter, "🔍 DEBUG: Scenario extends: %s\n", scen.Extends)
	}

	// Build vars: vars_file (if present), then inline vars override
	allVars := map[string]any{}
	if scen.VarsFile != "" {
		base := filepath.Dir(absScenario)
		vf := internal.MustAbsJoin(base, scen.VarsFile)
		if debug {
			fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading variables from: %s\n", vf)
		}
		vmap := map[string]any{}
		if err := internal.LoadYAML(vf, &vmap); err != nil {
			return nil, err
		}
		for k, v := range vmap {
			allVars[k] = v
		}
	}
	for k, v := range scen.Vars {
		allVars[k] = v
	}

	if debug && len(allVars) > 0 {
		fmt.Fprintf(debugWriter, "🔍 DEBUG: Variables available:\n")
		for k, v := range allVars {
			fmt.Fprintf(debugWriter, "  - %s = %v\n", k, v)
		}
	}

	// Policy document: template or pre-rendered JSON
	var policyJSON string
	var identityPolicyPath string
	switch {
	case scen.PolicyJSON != "" && scen.PolicyTemplate != "":
		return nil, fmt.Errorf("provide only one of 'policy_json' or 'policy_template'")
	case scen.PolicyJSON != "":
		base := filepath.Dir(absScenario)
		p := internal.MustAbsJoin(base, scen.PolicyJSON)
		identityPolicyPath = p
		if debug {
			fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading policy from: %s\n", p)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var policyData any
		if err := json.Unmarshal(b, &policyData); err != nil {
			return nil, fmt.Errorf("invalid JSON in policy file %s: %v", p, err)
		}
		policyJSON = internal.ToJSONPretty(policyData)
	case scen.PolicyTemplate != "":
		base := filepath.Dir(absScenario)
		tplPath := internal.MustAbsJoin(base, scen.PolicyTemplate)
		identityPolicyPath = tplPath
		if debug {
			fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading policy template from: %s\n", tplPath)
		}
		policyJSON = internal.RenderTemplateFileJSON(tplPath, allVars)
	default:
		return nil, fmt.Errorf("scenario must include 'policy_json' or 'policy_template'")
	}

	// Validate IAM fields if --strict-policy flag is set
	if strictPolicy {
		if err := internal.ValidateIAMFields(policyJSON); err != nil {
			return nil, fmt.Errorf("identity policy validation failed:\n%v", err)
		}
	}

	// Always strip non-IAM fields before sending to AWS
	policyJSON = internal.StripNonIAMFields(policyJSON)

	if debug {
		fmt.Fprintf(debugWriter, "🔍 DEBUG: Rendered policy (pretty-printed):\n%s\n", policyJSON)
	}

	// Process identity policy with source tracking (inject tracking Sids)
	policyJSONWithTracking, identitySourceMap := internal.ProcessIdentityPolicyWithSourceMap(policyJSON, identityPolicyPath)
	policyJSON = policyJSONWithTracking

	// Merge SCPs (permissions boundary) with source tracking
	var pbJSON string
	var scpSourceMap map[string]*internal.PolicySource
	if len(scen.SCPPaths) > 0 {
		files := internal.ExpandGlobsRelative(filepath.Dir(absScenario), scen.SCPPaths)
		if debug {
			fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading SCP/RCP files:\n")
			for _, f := range files {
				fmt.Fprintf(debugWriter, "  - %s\n", f)
			}
		}
		merged, sourceMap := internal.MergeSCPFilesWithSourceMap(files)
		scpSourceMap = sourceMap
		pbJSON = internal.ToJSONPretty(merged)

		// Validate IAM fields if --strict-policy flag is set
		if strictPolicy {
			if err := internal.ValidateIAMFields(pbJSON); err != nil {
				return nil, fmt.Errorf("SCP/RCP validation failed:\n%v", err)
			}
		}

		// Always strip non-IAM fields before sending to AWS
		pbJSON = internal.StripNonIAMFields(pbJSON)

		// Warn that SCP simulation is an approximation (unless suppressed)
		if !noWarn {
			internal.WarnSCPSimulation()
		}
	}

	// Resource policy: template or pre-rendered JSON
	var resourcePolicyJSON string
	switch {
	case scen.ResourcePolicyJSON != "" && scen.ResourcePolicyTemplate != "":
		return nil, fmt.Errorf("provide only one of 'resource_policy_json' or 'resource_policy_template'")
	case scen.ResourcePolicyJSON != "":
		base := filepath.Dir(absScenario)
		p := internal.MustAbsJoin(base, scen.ResourcePolicyJSON)
		if debug {
			fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading resource policy from: %s\n", p)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var resourcePolicyData any
		if err := json.Unmarshal(b, &resourcePolicyData); err != nil {
			return nil, fmt.Errorf("invalid JSON in resource policy file %s: %v", p, err)
		}
		resourcePolicyJSON = internal.ToJSONPretty(resourcePolicyData)
	case scen.ResourcePolicyTemplate != "":
		base := filepath.Dir(absScenario)
		tplPath := internal.MustAbsJoin(base, scen.ResourcePolicyTemplate)
		if debug {
			fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading resource policy template from: %s\n", tplPath)
		}
		resourcePolicyJSON = internal.RenderTemplateFileJSON(tplPath, allVars)
	}

	// Validate and strip resource policy if present
	if resourcePolicyJSON != "" {
		// Validate IAM fields if --strict-policy flag is set
		if strictPolicy {
			if err := internal.ValidateIAMFields(resourcePolicyJSON); err != nil {
				return nil, fmt.Errorf("resource policy validation failed:\n%v", err)
			}
		}

		// Always strip non-IAM fields before sending to AWS
		resourcePolicyJSON = internal.StripNonIAMFields(resourcePolicyJSON)
	}

	if debug && resourcePolicyJSON != "" {
		fmt.Fprintf(debugWriter, "🔍 DEBUG: Rendered resource policy (pretty-printed):\n%s\n", resourcePolicyJSON)
	}

	// Validate tests exist
	if len(scen.Tests) == 0 {
		return nil, fmt.Errorf("scenario must include 'tests' array with at least one test case")
	}

	// Build source map for tracking policy origins
	if scpSourceMap == nil {
		scpSourceMap = make(map[string]*internal.PolicySource)
	}
	if identitySourceMap == nil {
		identitySourceMap = make(map[string]*internal.PolicySource)
	}
	sourceMap := &internal.PolicySourceMap{
		Identity:               identitySourceMap,
		PermissionsBoundary:    scpSourceMap,
		PermissionsBoundaryRaw: pbJSON,
		IdentityPolicyRaw:      policyJSON,
		ResourcePolicyRaw:      resourcePolicyJSON,
	}

	// Track resource policy source if available
	if scen.ResourcePolicyJSON != "" {
		base := filepath.Dir(absScenario)
		policyPath := internal.MustAbsJoin(base, scen.ResourcePolicyJSON)
		sourceMap.ResourcePolicy = &internal.PolicySource{
			FilePath: policyPath,
		}
	} else if scen.ResourcePolicyTemplate != "" {
		base := filepath.Dir(absScenario)
		policyPath := internal.MustAbsJoin(base, scen.ResourcePolicyTemplate)
		sourceMap.ResourcePolicy = &internal.PolicySource{
			FilePath: policyPath,
		}
	}

	return &simulationPrep{
		scenario:            scen,
		policyJSON:          policyJSON,
		permissionsBoundary: pbJSON,
		resourcePolicyJSON:  resourcePolicyJSON,
		variables:           allVars,
		absScenarioPath:     absScenario,
		sourceMap:           sourceMap,
	}, nil
}

// run contains the main application logic and returns an error instead of calling Die()
func run(scenarioPath, savePath string, noAssert, noWarn, debug, strictPolicy, showMatchedSuccess bool, testFilter string, debugWriter io.Writer) error {
	// Prepare simulation data (AWS-free)
	prep, err := prepareSimulation(scenarioPath, noWarn, debug, strictPolicy, debugWriter)
	if err != nil {
		return err
	}

	// AWS client setup
	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return err
	}
	client := iam.NewFromConfig(awsCfg)

	// Build simulator configuration
	simCfg := internal.SimulatorConfig{
		PolicyJSON:          prep.policyJSON,
		PermissionsBoundary: prep.permissionsBoundary,
		ResourcePolicyJSON:  prep.resourcePolicyJSON,
		ScenarioPath:        prep.absScenarioPath,
		Variables:           prep.variables,
		SavePath:            savePath,
		NoAssert:            noAssert,
		ShowMatchedSuccess:  showMatchedSuccess,
		SourceMap:           prep.sourceMap,
		TestFilter:          testFilter,
	}

	// Run tests
	internal.RunTestCollection(client, prep.scenario, simCfg)
	return nil
}

// cliFlags holds the parsed command-line flags
type cliFlags struct {
	scenarioPath       string
	savePath           string
	noAssert           bool
	noWarn             bool
	showVersion        bool
	debug              bool
	strictPolicy       bool
	showMatchedSuccess bool
	tests              string // comma-separated list of test names to run
}

// parseFlags parses command-line arguments and returns flags or error
func parseFlags(args []string) (*cliFlags, []string, error) {
	fs := flag.NewFlagSet("politest", flag.ContinueOnError)

	flags := &cliFlags{}

	fs.StringVar(&flags.scenarioPath, "scenario", "", "Path to scenario YAML")
	fs.StringVar(&flags.savePath, "save", "", "Path to save raw JSON response")
	fs.BoolVar(&flags.noAssert, "no-assert", false, "Do not fail on expectation mismatches")
	fs.BoolVar(&flags.noWarn, "no-warn", false, "Suppress SCP/RCP simulation approximation warning")
	fs.BoolVar(&flags.debug, "debug", false, "Show debug output (files loaded, variables, rendered policies)")
	fs.BoolVar(&flags.showMatchedSuccess, "show-matched-success", false, "Show matched statements for passing tests")
	fs.BoolVar(&flags.showVersion, "version", false, "Show version information and exit")
	fs.StringVar(&flags.tests, "test", "", "Comma-separated list of test names to run (runs all if empty)")
	fs.BoolVar(&flags.strictPolicy, "strict-policy", false, "Fail if policies contain non-IAM fields")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	return flags, fs.Args(), nil
}

// validateArgs checks for unknown positional arguments
func validateArgs(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown arguments: %v\nUse -h or --help for usage information", args)
	}
	return nil
}

// generateFlags holds the parsed flags for the generate command
type generateFlags struct {
	url         string
	baseURL     string
	apiKey      string
	model       string
	outputDir   string
	noEnrich    bool
	quiet       bool
	prompt      string
	concurrency int
}

// parseGenerateFlags parses command-line arguments for the generate command
func parseGenerateFlags(args []string) (*generateFlags, error) {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)

	flags := &generateFlags{}

	fs.StringVar(&flags.url, "url", "", "AWS IAM documentation URL (required)")
	fs.StringVar(&flags.baseURL, "base-url", "", "OpenAI-compatible API base URL (required)")
	fs.StringVar(&flags.apiKey, "api-key", "", "API key for LLM service")
	fs.StringVar(&flags.model, "model", "", "LLM model name (required)")
	fs.StringVar(&flags.outputDir, "output", ".", "Output directory for generated files")
	fs.BoolVar(&flags.noEnrich, "no-enrich", false, "Skip action description enrichment")
	fs.BoolVar(&flags.quiet, "quiet", false, "Suppress progress output")
	fs.StringVar(&flags.prompt, "prompt", "", "Custom requirements/constraints to include in LLM prompt")
	fs.IntVar(&flags.concurrency, "concurrency", 3, "Number of parallel batch requests")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: politest generate [options]\n\n")
		fmt.Fprintf(os.Stderr, "Generate security-focused IAM policies from AWS documentation.\n\n")
		fmt.Fprintf(os.Stderr, "This command scrapes AWS IAM documentation pages to extract action definitions,\n")
		fmt.Fprintf(os.Stderr, "condition keys, and resource types, then uses an LLM to generate a comprehensive\n")
		fmt.Fprintf(os.Stderr, "security-focused policy suitable for regulated environments.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  politest generate \\\n")
		fmt.Fprintf(os.Stderr, "    --url https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazonbedrock.html \\\n")
		fmt.Fprintf(os.Stderr, "    --base-url http://localhost:3000 \\\n")
		fmt.Fprintf(os.Stderr, "    --model gpt-4 \\\n")
		fmt.Fprintf(os.Stderr, "    --api-key $OPENAI_API_KEY\n")
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return flags, nil
}

// runGenerate runs the generate command
func runGenerate(flags *generateFlags) error {
	cfg := internal.GenerateConfig{
		URL:         flags.url,
		BaseURL:     flags.baseURL,
		APIKey:      flags.apiKey,
		Model:       flags.model,
		OutputDir:   flags.outputDir,
		NoEnrich:    flags.noEnrich,
		Quiet:       flags.quiet,
		UserPrompt:  flags.prompt,
		Concurrency: flags.concurrency,
	}

	if err := internal.ValidateGenerateConfig(cfg); err != nil {
		return err
	}

	_, err := internal.RunGenerate(cfg, os.Stdout)
	return err
}

// printUsage prints the main usage information
func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: politest <command> [options]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  (default)    Run IAM policy simulations from scenario files\n")
	fmt.Fprintf(os.Stderr, "  generate     Generate security-focused IAM policies from AWS documentation\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Run 'politest <command> -h' for more information on a command.\n\n")
	fmt.Fprintf(os.Stderr, "Simulation Options (default command):\n")
	fmt.Fprintf(os.Stderr, "  -scenario string    Path to scenario YAML\n")
	fmt.Fprintf(os.Stderr, "  -save string        Path to save raw JSON response\n")
	fmt.Fprintf(os.Stderr, "  -no-assert          Do not fail on expectation mismatches\n")
	fmt.Fprintf(os.Stderr, "  -no-warn            Suppress SCP/RCP simulation approximation warning\n")
	fmt.Fprintf(os.Stderr, "  -debug              Show debug output\n")
	fmt.Fprintf(os.Stderr, "  -version            Show version information\n")
}

// realMain contains the full main logic and returns an exit code
// This allows testing without calling os.Exit
func realMain(args []string) int {
	// Check for subcommands
	if len(args) > 0 {
		switch args[0] {
		case "generate":
			genFlags, err := parseGenerateFlags(args[1:])
			if err != nil {
				if err == flag.ErrHelp {
					return 0
				}
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 1
			}
			if err := runGenerate(genFlags); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 1
			}
			return 0
		case "help", "-h", "--help":
			printUsage()
			return 0
		}
	}

	// Default: run simulation command
	flags, remainingArgs, err := parseFlags(args)
	if err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		return 1
	}

	// Handle --version flag
	if flags.showVersion {
		PrintVersion()
		return 0
	}

	// Validate no unknown arguments
	if err := validateArgs(remainingArgs); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	// Run main logic
	if err := run(flags.scenarioPath, flags.savePath, flags.noAssert, flags.noWarn, flags.debug, flags.strictPolicy, flags.showMatchedSuccess, flags.tests, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(realMain(os.Args[1:]))
}
