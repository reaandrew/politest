// cmd: go run . --scenario scenarios/athena_primary.yml --save /tmp/resp.json
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"gopkg.in/yaml.v3"
)

type Scenario struct {
	Extends        string            `yaml:"extends"`         // optional
	VarsFile       string            `yaml:"vars_file"`       // optional
	Vars           map[string]any    `yaml:"vars"`            // optional
	PolicyTemplate string            `yaml:"policy_template"` // OR
	PolicyJSON     string            `yaml:"policy_json"`     // mutually exclusive
	SCPPaths       []string          `yaml:"scp_paths"`       // optional
	Actions        []string          `yaml:"actions"`         // required if you want to simulate
	Resources      []string          `yaml:"resources"`       // optional
	Context        []ContextEntryYml `yaml:"context"`         // optional
	Expect         map[string]string `yaml:"expect"`          // optional (action -> decision)
}

type ContextEntryYml struct {
	ContextKeyName   string   `yaml:"ContextKeyName"`
	ContextKeyValues []string `yaml:"ContextKeyValues"`
	ContextKeyType   string   `yaml:"ContextKeyType"` // string, stringList, numeric, etc.
}

func main() {
	var scenarioPath string
	var savePath string
	var noAssert bool

	flag.StringVar(&scenarioPath, "scenario", "", "Path to scenario YAML")
	flag.StringVar(&savePath, "save", "", "Path to save raw JSON response")
	flag.BoolVar(&noAssert, "no-assert", false, "Do not fail on expectation mismatches")
	flag.Parse()

	if scenarioPath == "" {
		die("missing --scenario")
	}

	absScenario, err := filepath.Abs(scenarioPath)
	check(err)

	scen, err := loadScenarioWithExtends(absScenario)
	check(err)

	// Build vars: vars_file (if present), then inline vars override
	allVars := map[string]any{}
	if scen.VarsFile != "" {
		base := filepath.Dir(absScenario)
		vf := mustAbsJoin(base, scen.VarsFile)
		vmap := map[string]any{}
		check(loadYAML(vf, &vmap))
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
		die("provide only one of 'policy_json' or 'policy_template'")
	case scen.PolicyJSON != "":
		base := filepath.Dir(absScenario)
		p := mustAbsJoin(base, scen.PolicyJSON)
		b, err := os.ReadFile(p)
		check(err)
		policyJSON = minifyJSON(b)
	case scen.PolicyTemplate != "":
		base := filepath.Dir(absScenario)
		tplPath := mustAbsJoin(base, scen.PolicyTemplate)
		policyJSON = renderTemplateFileJSON(tplPath, allVars)
	default:
		die("scenario must include 'policy_json' or 'policy_template'")
	}

	// Merge SCPs (permissions boundary)
	var pbJSON string
	if len(scen.SCPPaths) > 0 {
		files := expandGlobsRelative(filepath.Dir(absScenario), scen.SCPPaths)
		merged := mergeSCPFiles(files)
		pbJSON = toJSONMin(merged)
	}

	// Render actions/resources/context with Go templates
	actions := renderStringSlice(scen.Actions, allVars)
	resources := renderStringSlice(scen.Resources, allVars)
	ctxEntries := renderContext(scen.Context, allVars)

	// AWS call
	cfg, err := config.LoadDefaultConfig(context.Background())
	check(err)
	client := iam.NewFromConfig(cfg)

	input := &iam.SimulateCustomPolicyInput{
		PolicyInputList: []string{policyJSON},
		ActionNames:     actions,
		ResourceArns:    resources,
		ContextEntries:  ctxEntries,
	}
	if pbJSON != "" {
		input.PermissionsBoundaryPolicyInputList = []string{pbJSON}
	}

	resp, err := client.SimulateCustomPolicy(context.Background(), input)
	check(err)

	// Print table
	rows := make([][3]string, 0, len(resp.EvaluationResults))
	evals := map[string]string{}
	for _, r := range resp.EvaluationResults {
		act := awsString(r.EvalActionName)
		dec := string(r.EvalDecision)
		evals[act] = dec
		detail := "-"
		if len(r.MatchedStatements) > 0 {
			parts := make([]string, 0, len(r.MatchedStatements))
			for _, m := range r.MatchedStatements {
				if m.SourcePolicyId != nil {
					parts = append(parts, awsString(m.SourcePolicyId))
				}
			}
			if len(parts) > 0 {
				detail = strings.Join(parts, ",")
			}
		}
		rows = append(rows, [3]string{act, dec, detail})
	}
	printTable(rows)

	// Save raw JSON if requested
	if savePath != "" {
		b, _ := json.MarshalIndent(resp, "", "  ")
		check(os.WriteFile(savePath, b, 0o644))
		fmt.Printf("\nSaved raw response → %s\n", savePath)
	}

	// Expectations
	failures := [][3]string{}
	for action, want := range scen.Expect {
		got := evals[action]
		if !strings.EqualFold(got, want) {
			failures = append(failures, [3]string{action, want, got})
		}
	}
	if len(failures) > 0 && !noAssert {
		fmt.Println("\nExpectation failures:")
		for _, f := range failures {
			fmt.Printf("  - %s: expected %s, got %s\n", f[0], f[1], ifEmpty(f[2], "<missing>"))
		}
		os.Exit(2)
	}
}

// ---------- YAML load + extends ----------

func loadScenarioWithExtends(absPath string) (*Scenario, error) {
	var s Scenario
	if err := loadYAML(absPath, &s); err != nil {
		return nil, err
	}
	if s.Extends == "" {
		return &s, nil
	}
	base := filepath.Dir(absPath)
	parent := mustAbsJoin(base, s.Extends)

	ps, err := loadScenarioWithExtends(parent)
	if err != nil {
		return nil, err
	}
	merged := mergeScenario(*ps, s) // child overrides parent
	return &merged, nil
}

func mergeScenario(a, b Scenario) Scenario {
	// simple field-wise merge: b overrides a; maps deep-merged
	out := a
	if b.VarsFile != "" {
		out.VarsFile = b.VarsFile
	}
	if b.PolicyTemplate != "" {
		out.PolicyTemplate = b.PolicyTemplate
		out.PolicyJSON = "" // ensure mutual exclusivity
	}
	if b.PolicyJSON != "" {
		out.PolicyJSON = b.PolicyJSON
		out.PolicyTemplate = ""
	}
	if len(b.SCPPaths) > 0 {
		out.SCPPaths = b.SCPPaths
	}
	if len(b.Actions) > 0 {
		out.Actions = b.Actions
	}
	if len(b.Resources) > 0 {
		out.Resources = b.Resources
	}
	if len(b.Context) > 0 {
		out.Context = b.Context
	}
	if len(b.Expect) > 0 {
		if out.Expect == nil {
			out.Expect = map[string]string{}
		}
		for k, v := range b.Expect {
			out.Expect[k] = v
		}
	}
	if out.Vars == nil {
		out.Vars = map[string]any{}
	}
	for k, v := range b.Vars {
		out.Vars[k] = v
	}
	return out
}

func loadYAML(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, v)
}

// ---------- templating & rendering ----------

func renderStringSlice(in []string, vars map[string]any) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, renderTemplateString(s, vars))
	}
	return out
}

func renderTemplateFileJSON(path string, vars map[string]any) string {
	tplText, err := os.ReadFile(path)
	check(err)
	tpl := template.Must(template.New(filepath.Base(path)).Option("missingkey=error").Parse(string(tplText)))
	var buf bytes.Buffer
	check(tpl.Execute(&buf, vars))
	// Validate and minify JSON
	return minifyJSON(buf.Bytes())
}

func renderTemplateString(s string, vars map[string]any) string {
	tpl := template.Must(template.New("inline").Option("missingkey=error").Parse(s))
	var buf bytes.Buffer
	check(tpl.Execute(&buf, vars))
	return buf.String()
}

func renderContext(in []ContextEntryYml, vars map[string]any) []iamtypes.ContextEntry {
	out := make([]iamtypes.ContextEntry, 0, len(in))
	for _, e := range in {
		values := make([]string, 0, len(e.ContextKeyValues))
		for _, v := range e.ContextKeyValues {
			values = append(values, renderTemplateString(v, vars))
		}
		out = append(out, iamtypes.ContextEntry{
			ContextKeyName:   strPtr(e.ContextKeyName),
			ContextKeyType:   parseContextType(e.ContextKeyType),
			ContextKeyValues: values,
		})
	}
	return out
}

func parseContextType(t string) iamtypes.ContextKeyTypeEnum {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "string":
		return iamtypes.ContextKeyTypeEnumString
	case "stringlist":
		return iamtypes.ContextKeyTypeEnumStringList
	case "numeric":
		return iamtypes.ContextKeyTypeEnumNumeric
	case "numericlist":
		return iamtypes.ContextKeyTypeEnumNumericList
	case "boolean":
		return iamtypes.ContextKeyTypeEnumBoolean
	case "booleanlist":
		return iamtypes.ContextKeyTypeEnumBooleanList
	default:
		return iamtypes.ContextKeyTypeEnumString
	}
}

// ---------- SCP merge ----------

func expandGlobsRelative(base string, patterns []string) []string {
	var files []string
	seen := map[string]struct{}{}
	for _, pat := range patterns {
		p := mustAbsJoin(base, pat)
		matches, _ := filepath.Glob(p)
		// If literal file exists but glob found nothing, include it
		if len(matches) == 0 {
			if _, err := os.Stat(p); err == nil {
				matches = []string{p}
			}
		}
		sort.Strings(matches)
		for _, m := range matches {
			key := mustAbs(m)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			files = append(files, key)
		}
	}
	return files
}

func mergeSCPFiles(files []string) map[string]any {
	statements := []any{}
	for _, f := range files {
		var doc any
		check(readJSONFile(f, &doc))
		switch t := doc.(type) {
		case map[string]any:
			if st, ok := t["Statement"]; ok {
				switch sv := st.(type) {
				case []any:
					statements = append(statements, sv...)
				default:
					statements = append(statements, sv)
				}
			} else {
				// assume it's a single statement object
				statements = append(statements, t)
			}
		default:
			// treat as a statement-ish blob
			statements = append(statements, t)
		}
	}
	return map[string]any{
		"Version":   "2012-10-17",
		"Statement": statements,
	}
}

func readJSONFile(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	return dec.Decode(v)
}

// ---------- utils ----------

func minifyJSON(b []byte) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		// try to marshal if text was formatted
		var anyv any
		if e2 := json.Unmarshal(b, &anyv); e2 == nil {
			out, _ := json.Marshal(anyv)
			return string(out)
		}
		die("invalid JSON produced by template: %v", err)
	}
	return buf.String()
}

func toJSONMin(v any) string {
	b, err := json.Marshal(v)
	check(err)
	return string(b)
}

func printTable(rows [][3]string) {
	if len(rows) == 0 {
		fmt.Println("No evaluation results.")
		return
	}
	// simple fixed-width columns
	w1, w2 := 6, 8
	for _, r := range rows {
		if len(r[0]) > w1 {
			w1 = len(r[0])
		}
		if len(r[1]) > w2 {
			w2 = len(r[1])
		}
	}
	fmt.Printf("%-*s  %-*s  %s\n", w1, "Action", w2, "Decision", "Matched (details)")
	fmt.Printf("%s  %s  %s\n", strings.Repeat("-", w1), strings.Repeat("-", w2), strings.Repeat("-", 40))
	for _, r := range rows {
		fmt.Printf("%-*s  %-*s  %s\n", w1, r[0], w2, r[1], r[2])
	}
}

func awsString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ifEmpty(s, rep string) string {
	if s == "" {
		return rep
	}
	return s
}

func strPtr(s string) *string { return &s }

func mustAbs(p string) string {
	ap, err := filepath.Abs(p)
	check(err)
	return ap
}

func mustAbsJoin(base, rel string) string {
	joined := rel
	if !filepath.IsAbs(rel) {
		joined = filepath.Join(base, rel)
	}
	return mustAbs(joined)
}

func check(err error) {
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			die("file not found: %v", err)
		}
		die("%v", err)
	}
}

func die(f string, a ...any) {
	fmt.Fprintf(os.Stderr, f+"\n", a...)
	os.Exit(1)
}
