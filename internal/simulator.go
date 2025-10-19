package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// RunLegacyFormat executes policy simulation in legacy format (actions + resources + expect map)
func RunLegacyFormat(client IAMSimulator, scen *Scenario, cfg SimulatorConfig) {
	// Render actions/resources/context with Go templates
	actions := RenderStringSlice(scen.Actions, cfg.Variables)
	resources := RenderStringSlice(scen.Resources, cfg.Variables)
	ctxEntries, err := RenderContext(scen.Context, cfg.Variables)
	Check(err)

	// Build and execute AWS API call
	input := buildSimulateInput(cfg, scen, actions, resources, ctxEntries)
	resp, err := client.SimulateCustomPolicy(context.Background(), input)
	Check(err)

	// Process and display results
	_ = processAndDisplayResults(resp)

	// Save raw JSON if requested
	saveResponseIfRequested(cfg.SavePath, resp)

	// Check expectations against ALL evaluation results (not just the map)
	// This ensures we catch failures when the same action is tested on multiple resources
	checkExpectationsAgainstAllResults(scen.Expect, resp.EvaluationResults, cfg.NoAssert)
}

// buildSimulateInput creates the IAM simulation input from scenario and config
func buildSimulateInput(cfg SimulatorConfig, scen *Scenario, actions, resources []string, ctxEntries []types.ContextEntry) *iam.SimulateCustomPolicyInput {
	input := &iam.SimulateCustomPolicyInput{
		PolicyInputList: []string{cfg.PolicyJSON},
		ActionNames:     actions,
		ResourceArns:    resources,
		ContextEntries:  ctxEntries,
	}
	if cfg.PermissionsBoundary != "" {
		input.PermissionsBoundaryPolicyInputList = []string{cfg.PermissionsBoundary}
	}
	if cfg.ResourcePolicyJSON != "" {
		input.ResourcePolicy = &cfg.ResourcePolicyJSON
	}
	if scen.CallerArn != "" {
		rendered := RenderString(scen.CallerArn, cfg.Variables)
		input.CallerArn = &rendered
	}
	if scen.ResourceOwner != "" {
		rendered := RenderString(scen.ResourceOwner, cfg.Variables)
		input.ResourceOwner = &rendered
	}
	if scen.ResourceHandlingOption != "" {
		input.ResourceHandlingOption = &scen.ResourceHandlingOption
	}
	return input
}

// processAndDisplayResults processes evaluation results and displays them in a table
// When multiple resources are tested with the same action, returns the decision map
// with action as key. For multiple resources with same action, the last result is stored
// but all results are displayed in the table.
func processAndDisplayResults(resp *iam.SimulateCustomPolicyOutput) map[string]string {
	rows := make([][3]string, 0, len(resp.EvaluationResults))
	evals := map[string]string{}
	for _, r := range resp.EvaluationResults {
		act := AwsString(r.EvalActionName)
		res := AwsString(r.EvalResourceName)
		dec := string(r.EvalDecision)

		// Store decision in map (will be overwritten if same action appears multiple times)
		evals[act] = dec

		// Display with resource name if available
		actionDisplay := act
		if res != "" && res != "*" {
			actionDisplay = fmt.Sprintf("%s on %s", act, res)
		}

		detail := extractMatchedStatements(r.MatchedStatements)
		rows = append(rows, [3]string{actionDisplay, dec, detail})
	}
	PrintTable(rows)
	return evals
}

// extractMatchedStatements extracts source policy IDs from matched statements
func extractMatchedStatements(matched []types.Statement) string {
	if len(matched) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(matched))
	for _, m := range matched {
		if m.SourcePolicyId != nil {
			parts = append(parts, AwsString(m.SourcePolicyId))
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, ",")
	}
	return "-"
}

// saveResponseIfRequested saves the API response to a file if savePath is provided
// Uses 0600 permissions to restrict access to the current user only, as the response
// may contain internal resource names or account IDs
func saveResponseIfRequested(savePath string, resp any) {
	if savePath != "" {
		b, _ := json.MarshalIndent(resp, "", "  ")
		Check(os.WriteFile(savePath, b, 0o600))
		fmt.Printf("\nSaved raw response → %s (permissions: 0600)\n", savePath)
	}
}

// checkExpectationsAgainstAllResults checks expectations against ALL evaluation results
// This is critical for legacy mode with multiple resources, as it ensures we catch
// failures even when the same action is tested on multiple resources with different outcomes
func checkExpectationsAgainstAllResults(expect map[string]string, results []types.EvaluationResult, noAssert bool) {
	if len(expect) == 0 {
		return // No expectations to check
	}

	failures := []string{}
	for _, r := range results {
		action := AwsString(r.EvalActionName)
		resource := AwsString(r.EvalResourceName)
		decision := string(r.EvalDecision)

		// Check if we have an expectation for this action
		if wantDecision, ok := expect[action]; ok {
			if !strings.EqualFold(decision, wantDecision) {
				resourceStr := "*"
				if resource != "" && resource != "*" {
					resourceStr = resource
				}
				failMsg := fmt.Sprintf("%s on %s: expected %s, got %s", action, resourceStr, wantDecision, decision)
				failures = append(failures, failMsg)
			}
		}
	}

	if len(failures) > 0 && !noAssert {
		fmt.Println("\nExpectation failures:")
		for _, msg := range failures {
			fmt.Printf("  - %s\n", msg)
		}
		GlobalExiter.Exit(2)
	}
}

// RunTestCollection executes policy simulation in test collection format
func RunTestCollection(client IAMSimulator, scen *Scenario, cfg SimulatorConfig) {
	passCount := 0
	failCount := 0
	var allResponses []*iam.SimulateCustomPolicyOutput

	// Expand tests with actions array into individual tests
	expandedTests := expandTestsWithActions(scen.Tests)

	fmt.Printf("Running %d test(s)...\n\n", len(expandedTests))

	for i, test := range expandedTests {
		pass, resp := runSingleTest(client, scen, cfg, test, i)
		allResponses = append(allResponses, resp)
		if pass {
			passCount++
		} else {
			failCount++
		}
	}

	printTestSummary(passCount, failCount)
	saveResponseIfRequested(cfg.SavePath, allResponses)

	if failCount > 0 && !cfg.NoAssert {
		GlobalExiter.Exit(2)
	}
}

// expandTestsWithActions expands tests that use actions array into individual tests
func expandTestsWithActions(tests []TestCase) []TestCase {
	var expanded []TestCase

	for _, test := range tests {
		// Validation: cannot have both action and actions
		if test.Action != "" && len(test.Actions) > 0 {
			Die("test '%s': cannot specify both 'action' and 'actions'", test.Name)
		}

		// If actions array is provided, expand into multiple tests
		if len(test.Actions) > 0 {
			for _, action := range test.Actions {
				expandedTest := test
				expandedTest.Action = action
				expandedTest.Actions = nil // Clear actions array
				expanded = append(expanded, expandedTest)
			}
		} else if test.Action != "" {
			// Single action - use as-is
			expanded = append(expanded, test)
		} else {
			// No action specified
			Die("test '%s': must specify either 'action' or 'actions'", test.Name)
		}
	}

	return expanded
}

// runSingleTest executes a single test case and returns pass/fail status and response
func runSingleTest(client IAMSimulator, scen *Scenario, cfg SimulatorConfig, test TestCase, index int) (bool, *iam.SimulateCustomPolicyOutput) {
	resources := prepareTestResources(test, cfg.Variables)
	action := RenderString(test.Action, cfg.Variables)
	testName := getTestName(test, action, resources)

	fmt.Printf("[%d/%d] %s\n", index+1, len(scen.Tests), testName)

	// Build test input
	ctxEntries, err := mergeContextEntries(scen.Context, test.Context, cfg.Variables)
	Check(err)
	testResourcePolicy := resolveResourcePolicy(test, cfg, index)
	input := buildTestInput(cfg, action, resources, ctxEntries, testResourcePolicy)
	applyTestOverrides(input, scen, test, cfg.Variables)

	// Execute test
	resp, err := client.SimulateCustomPolicy(context.Background(), input)
	Check(err)

	// Evaluate result
	pass := evaluateTestResult(resp, test, action, resources)
	return pass, resp
}

// prepareTestResources determines and renders resources for a test
func prepareTestResources(test TestCase, vars map[string]any) []string {
	if test.Resource != "" {
		return []string{RenderString(test.Resource, vars)}
	}
	if len(test.Resources) > 0 {
		return RenderStringSlice(test.Resources, vars)
	}
	return nil
}

// getTestName generates a test name if not provided
func getTestName(test TestCase, action string, resources []string) string {
	if test.Name != "" {
		return test.Name
	}
	resourceStr := "*"
	if len(resources) > 0 {
		resourceStr = resources[0]
	}
	return fmt.Sprintf("%s on %s", action, resourceStr)
}

// mergeContextEntries merges scenario-level and test-level context
func mergeContextEntries(scenCtx, testCtx []ContextEntryYml, vars map[string]any) ([]types.ContextEntry, error) {
	ctxEntries, err := RenderContext(scenCtx, vars)
	if err != nil {
		return nil, err
	}
	if len(testCtx) > 0 {
		testCtxRendered, err := RenderContext(testCtx, vars)
		if err != nil {
			return nil, err
		}
		ctxEntries = append(ctxEntries, testCtxRendered...)
	}
	return ctxEntries, nil
}

// resolveResourcePolicy determines the resource policy for a test
func resolveResourcePolicy(test TestCase, cfg SimulatorConfig, testIndex int) string {
	testResourcePolicy := cfg.ResourcePolicyJSON
	switch {
	case test.ResourcePolicyJSON != "" && test.ResourcePolicyTemplate != "":
		Die("test %d: provide only one of 'resource_policy_json' or 'resource_policy_template'", testIndex+1)
	case test.ResourcePolicyJSON != "":
		base := filepath.Dir(cfg.ScenarioPath)
		p := MustAbsJoin(base, test.ResourcePolicyJSON)
		b, err := os.ReadFile(p)
		Check(err)
		testResourcePolicy = MinifyJSON(b)
	case test.ResourcePolicyTemplate != "":
		base := filepath.Dir(cfg.ScenarioPath)
		tplPath := MustAbsJoin(base, test.ResourcePolicyTemplate)
		testResourcePolicy = RenderTemplateFileJSON(tplPath, cfg.Variables)
	}
	return testResourcePolicy
}

// buildTestInput creates the IAM simulation input for a single test
func buildTestInput(cfg SimulatorConfig, action string, resources []string, ctxEntries []types.ContextEntry, resourcePolicy string) *iam.SimulateCustomPolicyInput {
	input := &iam.SimulateCustomPolicyInput{
		PolicyInputList: []string{cfg.PolicyJSON},
		ActionNames:     []string{action},
		ResourceArns:    resources,
		ContextEntries:  ctxEntries,
	}
	if cfg.PermissionsBoundary != "" {
		input.PermissionsBoundaryPolicyInputList = []string{cfg.PermissionsBoundary}
	}
	if resourcePolicy != "" {
		input.ResourcePolicy = &resourcePolicy
	}
	return input
}

// applyTestOverrides applies test-level overrides to the simulation input
func applyTestOverrides(input *iam.SimulateCustomPolicyInput, scen *Scenario, test TestCase, vars map[string]any) {
	// Caller ARN
	callerArn := scen.CallerArn
	if test.CallerArn != "" {
		callerArn = test.CallerArn
	}
	if callerArn != "" {
		rendered := RenderString(callerArn, vars)
		input.CallerArn = &rendered
	}

	// Resource Owner
	resourceOwner := scen.ResourceOwner
	if test.ResourceOwner != "" {
		resourceOwner = test.ResourceOwner
	}
	if resourceOwner != "" {
		rendered := RenderString(resourceOwner, vars)
		input.ResourceOwner = &rendered
	}

	// Resource Handling Option
	resourceHandlingOption := scen.ResourceHandlingOption
	if test.ResourceHandlingOption != "" {
		resourceHandlingOption = test.ResourceHandlingOption
	}
	if resourceHandlingOption != "" {
		input.ResourceHandlingOption = &resourceHandlingOption
	}
}

// evaluateTestResult checks the API response against expectations and prints result
func evaluateTestResult(resp *iam.SimulateCustomPolicyOutput, test TestCase, action string, resources []string) bool {
	if len(resp.EvaluationResults) == 0 {
		fmt.Printf("  ✗ FAIL: no evaluation results returned\n\n")
		return false
	}

	result := resp.EvaluationResults[0]
	decision := string(result.EvalDecision)
	detail := extractMatchedStatements(result.MatchedStatements)

	if test.Expect == "" {
		fmt.Printf("  → Result: %s (matched: %s)\n\n", decision, detail)
		return true
	}

	if strings.EqualFold(decision, test.Expect) {
		fmt.Printf("  ✓ PASS: %s (matched: %s)\n\n", decision, detail)
		return true
	}

	printTestFailure(test, action, resources, decision, detail)
	return false
}

// printTestFailure prints a formatted failure message
func printTestFailure(test TestCase, action string, resources []string, decision, detail string) {
	if test.Name == "" {
		resourceStr := "*"
		if len(resources) > 0 {
			resourceStr = resources[0]
		}
		fmt.Printf("  ✗ FAIL: %s on %s failed: expected %s, got %s\n\n", action, resourceStr, test.Expect, decision)
	} else {
		fmt.Printf("  ✗ FAIL: expected %s, got %s (matched: %s)\n\n", test.Expect, decision, detail)
	}
}

// printTestSummary prints the final test summary
func printTestSummary(passCount, failCount int) {
	fmt.Printf("========================================\n")
	fmt.Printf("Test Results: %d passed, %d failed\n", passCount, failCount)
	fmt.Printf("========================================\n")
}
