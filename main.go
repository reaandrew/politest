// cmd: go run . --scenario scenarios/athena_primary.yml --save /tmp/resp.json
package main

import (
	"context"
	"flag"
	"fmt"
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
func run(scenarioPath, savePath string, noAssert, noWarn bool) error {
	if scenarioPath == "" {
		return fmt.Errorf("missing --scenario\nUsage: politest --scenario <path> [--save <path>] [--no-assert] [--no-warn]")
	}

	absScenario, err := filepath.Abs(scenarioPath)
	if err != nil {
		return err
	}

	scen, err := internal.LoadScenarioWithExtends(absScenario)
	if err != nil {
		return err
	}

	// Build vars: vars_file (if present), then inline vars override
	allVars := map[string]any{}
	if scen.VarsFile != "" {
		base := filepath.Dir(absScenario)
		vf := internal.MustAbsJoin(base, scen.VarsFile)
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

	// Policy document: template or pre-rendered JSON
	var policyJSON string
	switch {
	case scen.PolicyJSON != "" && scen.PolicyTemplate != "":
		return fmt.Errorf("provide only one of 'policy_json' or 'policy_template'")
	case scen.PolicyJSON != "":
		base := filepath.Dir(absScenario)
		p := internal.MustAbsJoin(base, scen.PolicyJSON)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		policyJSON = internal.MinifyJSON(b)
	case scen.PolicyTemplate != "":
		base := filepath.Dir(absScenario)
		tplPath := internal.MustAbsJoin(base, scen.PolicyTemplate)
		policyJSON = internal.RenderTemplateFileJSON(tplPath, allVars)
	default:
		return fmt.Errorf("scenario must include 'policy_json' or 'policy_template'")
	}

	// Merge SCPs (permissions boundary)
	var pbJSON string
	if len(scen.SCPPaths) > 0 {
		files := internal.ExpandGlobsRelative(filepath.Dir(absScenario), scen.SCPPaths)
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
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		resourcePolicyJSON = internal.MinifyJSON(b)
	case scen.ResourcePolicyTemplate != "":
		base := filepath.Dir(absScenario)
		tplPath := internal.MustAbsJoin(base, scen.ResourcePolicyTemplate)
		resourcePolicyJSON = internal.RenderTemplateFileJSON(tplPath, allVars)
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

func main() {
	var scenarioPath string
	var savePath string
	var noAssert bool
	var noWarn bool
	var showVersion bool

	flag.StringVar(&scenarioPath, "scenario", "", "Path to scenario YAML")
	flag.StringVar(&savePath, "save", "", "Path to save raw JSON response")
	flag.BoolVar(&noAssert, "no-assert", false, "Do not fail on expectation mismatches")
	flag.BoolVar(&noWarn, "no-warn", false, "Suppress SCP/RCP simulation approximation warning")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.Parse()

	// Handle --version flag
	if showVersion {
		PrintVersion()
		os.Exit(0)
	}

	// Check for unknown flags
	if flag.NArg() > 0 {
		internal.Die("unknown arguments: %v\nUse -h or --help for usage information", flag.Args())
	}

	// Run main logic
	if err := run(scenarioPath, savePath, noAssert, noWarn); err != nil {
		internal.Die("%v", err)
	}
}
