// cmd: go run . --scenario scenarios/athena_primary.yml --save /tmp/resp.json
package main

import (
	"context"
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
func prepareSimulation(scenarioPath string, noWarn, debug bool, debugWriter io.Writer) (*simulationPrep, error) {
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
		policyJSON = internal.MinifyJSON(b)
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

	if debug {
		fmt.Fprintf(debugWriter, "🔍 DEBUG: Rendered policy (minified):\n%s\n", policyJSON)
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
		pbJSON = internal.ToJSONMin(merged)

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
		resourcePolicyJSON = internal.MinifyJSON(b)
	case scen.ResourcePolicyTemplate != "":
		base := filepath.Dir(absScenario)
		tplPath := internal.MustAbsJoin(base, scen.ResourcePolicyTemplate)
		if debug {
			fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading resource policy template from: %s\n", tplPath)
		}
		resourcePolicyJSON = internal.RenderTemplateFileJSON(tplPath, allVars)
	}

	if debug && resourcePolicyJSON != "" {
		fmt.Fprintf(debugWriter, "🔍 DEBUG: Rendered resource policy (minified):\n%s\n", resourcePolicyJSON)
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
func run(scenarioPath, savePath string, noAssert, noWarn, debug, showMatchedSuccess bool, debugWriter io.Writer) error {
	// Prepare simulation data (AWS-free)
	prep, err := prepareSimulation(scenarioPath, noWarn, debug, debugWriter)
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
	showMatchedSuccess bool
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

// realMain contains the full main logic and returns an exit code
// This allows testing without calling os.Exit
func realMain(args []string) int {
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
	if err := run(flags.scenarioPath, flags.savePath, flags.noAssert, flags.noWarn, flags.debug, flags.showMatchedSuccess, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(realMain(os.Args[1:]))
}
