// cmd: go run . --scenario scenarios/athena_primary.yml --save /tmp/resp.json
package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	"politest/internal"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

func main() {
	var scenarioPath string
	var savePath string
	var noAssert bool

	flag.StringVar(&scenarioPath, "scenario", "", "Path to scenario YAML")
	flag.StringVar(&savePath, "save", "", "Path to save raw JSON response")
	flag.BoolVar(&noAssert, "no-assert", false, "Do not fail on expectation mismatches")
	flag.Parse()

	if scenarioPath == "" {
		internal.Die("missing --scenario")
	}

	absScenario, err := filepath.Abs(scenarioPath)
	internal.Check(err)

	scen, err := internal.LoadScenarioWithExtends(absScenario)
	internal.Check(err)

	// Build vars: vars_file (if present), then inline vars override
	allVars := map[string]any{}
	if scen.VarsFile != "" {
		base := filepath.Dir(absScenario)
		vf := internal.MustAbsJoin(base, scen.VarsFile)
		vmap := map[string]any{}
		internal.Check(internal.LoadYAML(vf, &vmap))
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
		internal.Die("provide only one of 'policy_json' or 'policy_template'")
	case scen.PolicyJSON != "":
		base := filepath.Dir(absScenario)
		p := internal.MustAbsJoin(base, scen.PolicyJSON)
		b, err := os.ReadFile(p)
		internal.Check(err)
		policyJSON = internal.MinifyJSON(b)
	case scen.PolicyTemplate != "":
		base := filepath.Dir(absScenario)
		tplPath := internal.MustAbsJoin(base, scen.PolicyTemplate)
		policyJSON = internal.RenderTemplateFileJSON(tplPath, allVars)
	default:
		internal.Die("scenario must include 'policy_json' or 'policy_template'")
	}

	// Merge SCPs (permissions boundary)
	var pbJSON string
	if len(scen.SCPPaths) > 0 {
		files := internal.ExpandGlobsRelative(filepath.Dir(absScenario), scen.SCPPaths)
		merged := internal.MergeSCPFiles(files)
		pbJSON = internal.ToJSONMin(merged)
	}

	// Resource policy: template or pre-rendered JSON
	var resourcePolicyJSON string
	switch {
	case scen.ResourcePolicyJSON != "" && scen.ResourcePolicyTemplate != "":
		internal.Die("provide only one of 'resource_policy_json' or 'resource_policy_template'")
	case scen.ResourcePolicyJSON != "":
		base := filepath.Dir(absScenario)
		p := internal.MustAbsJoin(base, scen.ResourcePolicyJSON)
		b, err := os.ReadFile(p)
		internal.Check(err)
		resourcePolicyJSON = internal.MinifyJSON(b)
	case scen.ResourcePolicyTemplate != "":
		base := filepath.Dir(absScenario)
		tplPath := internal.MustAbsJoin(base, scen.ResourcePolicyTemplate)
		resourcePolicyJSON = internal.RenderTemplateFileJSON(tplPath, allVars)
	}

	// AWS client setup
	cfg, err := config.LoadDefaultConfig(context.Background())
	internal.Check(err)
	client := iam.NewFromConfig(cfg)

	// Determine format and run tests
	if len(scen.Tests) > 0 {
		// New format: collection of named tests
		internal.RunTestCollection(client, scen, policyJSON, pbJSON, resourcePolicyJSON, absScenario, allVars, savePath, noAssert)
	} else {
		// Legacy format: actions + resources + expect map
		internal.RunLegacyFormat(client, scen, policyJSON, pbJSON, resourcePolicyJSON, allVars, savePath, noAssert)
	}
}
