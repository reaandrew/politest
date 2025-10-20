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

// RunTestCollection executes policy simulation in test collection format
func RunTestCollection(client IAMSimulator, scen *Scenario, cfg SimulatorConfig) {
	passCount := 0
	failCount := 0
	var allResponses []*iam.SimulateCustomPolicyOutput

	// Expand tests with actions array into individual tests
	expandedTests := expandTestsWithActions(scen.Tests)

	fmt.Printf("Running %d test(s)...\n\n", len(expandedTests))

	for i, test := range expandedTests {
		pass, resp := runSingleTest(client, scen, cfg, test, i, len(expandedTests))
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
func runSingleTest(client IAMSimulator, scen *Scenario, cfg SimulatorConfig, test TestCase, index int, totalTests int) (bool, *iam.SimulateCustomPolicyOutput) {
	resources := prepareTestResources(test, cfg.Variables)
	action := RenderString(test.Action, cfg.Variables)
	testName := getTestName(test, action, resources)

	fmt.Printf("[%d/%d] %s\n", index+1, totalTests, testName)

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
