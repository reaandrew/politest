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
	pass := evaluateTestResult(resp, test, action, resources, cfg)
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
func evaluateTestResult(resp *iam.SimulateCustomPolicyOutput, test TestCase, action string, resources []string, cfg SimulatorConfig) bool {
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
		fmt.Printf("  ✓ PASS: %s (matched: %s)\n", decision, detail)
		if cfg.ShowMatchedSuccess {
			displayMatchedStatements(result.MatchedStatements, cfg)
		}
		fmt.Println()
		return true
	}

	printTestFailure(test, action, resources, decision, detail, result.MatchedStatements, cfg)
	return false
}

// printTestFailure prints a formatted failure message with matched statement details
func printTestFailure(test TestCase, action string, resources []string, decision, detail string, matchedStatements []types.Statement, cfg SimulatorConfig) {
	fmt.Printf("  ✗ FAIL:\n")
	fmt.Printf("    Expected: %s\n", test.Expect)
	fmt.Printf("    Action:   %s\n", action)

	// Display resources
	if len(resources) == 0 {
		fmt.Printf("    Resource: *\n")
	} else if len(resources) == 1 {
		fmt.Printf("    Resource: %s\n", resources[0])
	} else {
		fmt.Printf("    Resources:\n")
		for _, res := range resources {
			fmt.Printf("      - %s\n", res)
		}
	}

	// Display context keys if present
	if len(test.Context) > 0 {
		fmt.Printf("    Context:\n")
		for _, ctx := range test.Context {
			if len(ctx.ContextKeyValues) == 1 {
				fmt.Printf("      %s = %s\n", ctx.ContextKeyName, ctx.ContextKeyValues[0])
			} else {
				fmt.Printf("      %s = [%s]\n", ctx.ContextKeyName, strings.Join(ctx.ContextKeyValues, ", "))
			}
		}
	}

	fmt.Printf("    Got:      %s\n", decision)

	// Display matched statements with source information
	displayMatchedStatements(matchedStatements, cfg)
	fmt.Println()
}

// displayMatchedStatements shows detailed information about matched policy statements
func displayMatchedStatements(matchedStatements []types.Statement, cfg SimulatorConfig) {
	if len(matchedStatements) == 0 || cfg.SourceMap == nil {
		return
	}

	fmt.Println("  Matched statements:")
	for _, stmt := range matchedStatements {
		displaySingleStatement(stmt, cfg)
	}
}

// displaySingleStatement displays a single matched statement with source information
func displaySingleStatement(stmt types.Statement, cfg SimulatorConfig) {
	if stmt.SourcePolicyId == nil {
		return
	}

	sourcePolicyID := *stmt.SourcePolicyId
	var source *PolicySource

	// Determine which policy this statement came from
	switch {
	case strings.HasPrefix(sourcePolicyID, "PolicyInputList"):
		// Look up specific identity policy statement by extracting Sid
		policyJSON := cfg.SourceMap.IdentityPolicyRaw
		if stmt.StartPosition != nil && stmt.EndPosition != nil && policyJSON != "" {
			stmtJSON := extractStatementFromPolicy(policyJSON, stmt.StartPosition, stmt.EndPosition)
			if trackingSid := extractSidFromJSON(stmtJSON); trackingSid != "" {
				if src, ok := cfg.SourceMap.Identity[trackingSid]; ok {
					source = src
				}
			}
		}
	case strings.HasPrefix(sourcePolicyID, "PermissionsBoundaryPolicyInputList"):
		// Look up specific SCP statement by extracting Sid
		policyJSON := cfg.SourceMap.PermissionsBoundaryRaw
		if stmt.StartPosition != nil && stmt.EndPosition != nil && policyJSON != "" {
			stmtJSON := extractStatementFromPolicy(policyJSON, stmt.StartPosition, stmt.EndPosition)
			if trackingSid := extractSidFromJSON(stmtJSON); trackingSid != "" {
				if src, ok := cfg.SourceMap.PermissionsBoundary[trackingSid]; ok {
					source = src
				}
			}
		}
	case strings.HasPrefix(sourcePolicyID, "ResourcePolicy"):
		source = cfg.SourceMap.ResourcePolicy
	default:
		// Unknown source
		fmt.Printf("    • %s (unknown source)\n", sourcePolicyID)
		return
	}

	// Display header with Sid if available
	if source != nil && source.Sid != "" {
		fmt.Printf("    • %s (Sid: %s)\n", sourcePolicyID, source.Sid)
	} else {
		fmt.Printf("    • %s\n", sourcePolicyID)
	}

	// Display source file path with line numbers
	if source != nil && source.FilePath != "" {
		if source.StartLine > 0 && source.EndLine > 0 {
			fmt.Printf("      Source: %s:%d-%d\n", source.FilePath, source.StartLine, source.EndLine)
		} else {
			fmt.Printf("      Source: %s\n", source.FilePath)
		}

		// Display statement with context from source file
		displayStatementWithContext(source)
	}
}

// displayStatementWithContext reads the source file and displays the statement lines
func displayStatementWithContext(source *PolicySource) {
	if source.StartLine == 0 || source.EndLine == 0 {
		return
	}

	// Read source file
	content, err := os.ReadFile(source.FilePath)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return
	}

	fmt.Println()
	// Display only the statement lines (StartLine to EndLine)
	for i := source.StartLine - 1; i < source.EndLine; i++ { // -1 for 0-based array indexing
		if i >= 0 && i < len(lines) {
			fmt.Printf("      %d: %s\n", i+1, lines[i])
		}
	}
}

// extractStatementFromPolicy extracts a statement JSON from policy using line/column positions
func extractStatementFromPolicy(policyJSON string, start, end *types.Position) string {
	if start == nil || end == nil {
		return ""
	}

	lines := strings.Split(policyJSON, "\n")
	if len(lines) == 0 {
		return ""
	}

	startLine := int(start.Line) - 1 // AWS uses 1-based line numbers
	endLine := int(end.Line) - 1
	startCol := int(start.Column) - 1 // AWS uses 1-based column numbers
	endCol := int(end.Column) - 1

	if startLine < 0 || startLine >= len(lines) || endLine < 0 || endLine >= len(lines) {
		return ""
	}

	var extracted string
	if startLine == endLine {
		// Single line
		if startCol >= 0 && endCol <= len(lines[startLine]) {
			extracted = lines[startLine][startCol:endCol]
		}
	} else {
		// Multi-line
		var builder strings.Builder
		// First line (from startCol to end)
		if startCol >= 0 && startCol < len(lines[startLine]) {
			builder.WriteString(lines[startLine][startCol:])
			builder.WriteString("\n")
		}
		// Middle lines (complete lines)
		for i := startLine + 1; i < endLine; i++ {
			if i < len(lines) {
				builder.WriteString(lines[i])
				builder.WriteString("\n")
			}
		}
		// Last line (from start to endCol)
		if endLine < len(lines) && endCol >= 0 && endCol <= len(lines[endLine]) {
			builder.WriteString(lines[endLine][:endCol])
		}
		extracted = builder.String()
	}

	// Clean up: trim whitespace and leading commas that AWS includes
	extracted = strings.TrimSpace(extracted)
	extracted = strings.TrimPrefix(extracted, ",")
	extracted = strings.TrimSpace(extracted)

	return extracted
}

// extractSidFromJSON extracts the Sid field from a statement JSON string
func extractSidFromJSON(stmtJSON string) string {
	var stmt map[string]any
	if err := json.Unmarshal([]byte(stmtJSON), &stmt); err != nil {
		return ""
	}
	if sid, ok := stmt["Sid"]; ok {
		if sidStr, ok := sid.(string); ok {
			return sidStr
		}
	}
	return ""
}

// printTestSummary prints the final test summary
func printTestSummary(passCount, failCount int) {
	fmt.Printf("========================================\n")
	fmt.Printf("Test Results: %d passed, %d failed\n", passCount, failCount)
	fmt.Printf("========================================\n")
}
