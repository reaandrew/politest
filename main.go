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

// run contains the main application logic and returns an error instead of calling Die()
func run(scenarioPath, savePath string, noAssert, noWarn, debug bool, debugWriter io.Writer) error {
	if scenarioPath == "" {
		return fmt.Errorf("missing --scenario\nUsage: politest --scenario <path> [--save <path>] [--no-assert] [--no-warn] [--debug]")
	}

	absScenario, err := filepath.Abs(scenarioPath)
	if err != nil {
		return err
	}

	if debug {
		fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading scenario from: %s\n", absScenario)
	}

	scen, err := internal.LoadScenarioWithExtends(absScenario)
	if err != nil {
		return err
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
			return err
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
	switch {
	case scen.PolicyJSON != "" && scen.PolicyTemplate != "":
		return fmt.Errorf("provide only one of 'policy_json' or 'policy_template'")
	case scen.PolicyJSON != "":
		base := filepath.Dir(absScenario)
		p := internal.MustAbsJoin(base, scen.PolicyJSON)
		if debug {
			fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading policy from: %s\n", p)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		policyJSON = internal.MinifyJSON(b)
	case scen.PolicyTemplate != "":
		base := filepath.Dir(absScenario)
		tplPath := internal.MustAbsJoin(base, scen.PolicyTemplate)
		if debug {
			fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading policy template from: %s\n", tplPath)
		}
		policyJSON = internal.RenderTemplateFileJSON(tplPath, allVars)
	default:
		return fmt.Errorf("scenario must include 'policy_json' or 'policy_template'")
	}

	if debug {
		fmt.Fprintf(debugWriter, "🔍 DEBUG: Rendered policy (minified):\n%s\n", policyJSON)
	}

	// Merge SCPs (permissions boundary)
	var pbJSON string
	if len(scen.SCPPaths) > 0 {
		files := internal.ExpandGlobsRelative(filepath.Dir(absScenario), scen.SCPPaths)
		if debug {
			fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading SCP/RCP files:\n")
			for _, f := range files {
				fmt.Fprintf(debugWriter, "  - %s\n", f)
			}
		}
		merged := internal.MergeSCPFiles(files)
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
		return fmt.Errorf("provide only one of 'resource_policy_json' or 'resource_policy_template'")
	case scen.ResourcePolicyJSON != "":
		base := filepath.Dir(absScenario)
		p := internal.MustAbsJoin(base, scen.ResourcePolicyJSON)
		if debug {
			fmt.Fprintf(debugWriter, "🔍 DEBUG: Loading resource policy from: %s\n", p)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
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

	// AWS client setup
	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return err
	}
	client := iam.NewFromConfig(awsCfg)

	// Build simulator configuration
	simCfg := internal.SimulatorConfig{
		PolicyJSON:          policyJSON,
		PermissionsBoundary: pbJSON,
		ResourcePolicyJSON:  resourcePolicyJSON,
		ScenarioPath:        absScenario,
		Variables:           allVars,
		SavePath:            savePath,
		NoAssert:            noAssert,
	}

	// Validate and run tests
	if len(scen.Tests) == 0 {
		return fmt.Errorf("scenario must include 'tests' array with at least one test case")
	}
	internal.RunTestCollection(client, scen, simCfg)
	return nil
}

// cliFlags holds the parsed command-line flags
type cliFlags struct {
	scenarioPath string
	savePath     string
	noAssert     bool
	noWarn       bool
	showVersion  bool
	debug        bool
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
	if err := run(flags.scenarioPath, flags.savePath, flags.noAssert, flags.noWarn, flags.debug, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(realMain(os.Args[1:]))
}
