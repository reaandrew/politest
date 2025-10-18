package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/iam"
)

// RunLegacyFormat executes policy simulation in legacy format (actions + resources + expect map)
func RunLegacyFormat(client IAMSimulator, scen *Scenario, policyJSON, pbJSON, resourcePolicyJSON string, allVars map[string]any, savePath string, noAssert bool) {
	// Render actions/resources/context with Go templates
	actions := RenderStringSlice(scen.Actions, allVars)
	resources := RenderStringSlice(scen.Resources, allVars)
	ctxEntries := RenderContext(scen.Context, allVars)

	// AWS call
	input := &iam.SimulateCustomPolicyInput{
		PolicyInputList: []string{policyJSON},
		ActionNames:     actions,
		ResourceArns:    resources,
		ContextEntries:  ctxEntries,
	}
	if pbJSON != "" {
		input.PermissionsBoundaryPolicyInputList = []string{pbJSON}
	}
	if resourcePolicyJSON != "" {
		input.ResourcePolicy = &resourcePolicyJSON
	}
	if scen.CallerArn != "" {
		rendered := RenderString(scen.CallerArn, allVars)
		input.CallerArn = &rendered
	}
	if scen.ResourceOwner != "" {
		rendered := RenderString(scen.ResourceOwner, allVars)
		input.ResourceOwner = &rendered
	}
	if scen.ResourceHandlingOption != "" {
		input.ResourceHandlingOption = &scen.ResourceHandlingOption
	}

	resp, err := client.SimulateCustomPolicy(context.Background(), input)
	Check(err)

	// Print table
	rows := make([][3]string, 0, len(resp.EvaluationResults))
	evals := map[string]string{}
	for _, r := range resp.EvaluationResults {
		act := AwsString(r.EvalActionName)
		dec := string(r.EvalDecision)
		evals[act] = dec
		detail := "-"
		if len(r.MatchedStatements) > 0 {
			parts := make([]string, 0, len(r.MatchedStatements))
			for _, m := range r.MatchedStatements {
				if m.SourcePolicyId != nil {
					parts = append(parts, AwsString(m.SourcePolicyId))
				}
			}
			if len(parts) > 0 {
				detail = strings.Join(parts, ",")
			}
		}
		rows = append(rows, [3]string{act, dec, detail})
	}
	PrintTable(rows)

	// Save raw JSON if requested
	if savePath != "" {
		b, _ := json.MarshalIndent(resp, "", "  ")
		Check(os.WriteFile(savePath, b, 0o644))
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
			fmt.Printf("  - %s: expected %s, got %s\n", f[0], f[1], IfEmpty(f[2], "<missing>"))
		}
		GlobalExiter.Exit(2)
	}
}

// RunTestCollection executes policy simulation in test collection format
func RunTestCollection(client IAMSimulator, scen *Scenario, policyJSON, pbJSON, resourcePolicyJSON, absScenario string, allVars map[string]any, savePath string, noAssert bool) {
	passCount := 0
	failCount := 0
	var allResponses []*iam.SimulateCustomPolicyOutput

	fmt.Printf("Running %d test(s)...\n\n", len(scen.Tests))

	for i, test := range scen.Tests {
		// Determine resources for this test
		var resources []string
		if test.Resource != "" {
			resources = []string{RenderString(test.Resource, allVars)}
		} else if len(test.Resources) > 0 {
			resources = RenderStringSlice(test.Resources, allVars)
		}

		// Render action
		action := RenderString(test.Action, allVars)

		// Generate test name if not provided
		testName := test.Name
		if testName == "" {
			// Default format: "action on resource"
			resourceStr := "*"
			if len(resources) > 0 {
				resourceStr = resources[0]
			}
			testName = fmt.Sprintf("%s on %s", action, resourceStr)
		}

		fmt.Printf("[%d/%d] %s\n", i+1, len(scen.Tests), testName)

		// Merge context: scenario-level + test-level
		ctxEntries := RenderContext(scen.Context, allVars)
		if len(test.Context) > 0 {
			testCtx := RenderContext(test.Context, allVars)
			ctxEntries = append(ctxEntries, testCtx...)
		}

		// Determine resource policy: test-level overrides scenario-level
		testResourcePolicy := resourcePolicyJSON
		switch {
		case test.ResourcePolicyJSON != "" && test.ResourcePolicyTemplate != "":
			Die("test %d: provide only one of 'resource_policy_json' or 'resource_policy_template'", i+1)
		case test.ResourcePolicyJSON != "":
			base := filepath.Dir(absScenario)
			p := MustAbsJoin(base, test.ResourcePolicyJSON)
			b, err := os.ReadFile(p)
			Check(err)
			testResourcePolicy = MinifyJSON(b)
		case test.ResourcePolicyTemplate != "":
			base := filepath.Dir(absScenario)
			tplPath := MustAbsJoin(base, test.ResourcePolicyTemplate)
			testResourcePolicy = RenderTemplateFileJSON(tplPath, allVars)
		}

		// AWS call
		input := &iam.SimulateCustomPolicyInput{
			PolicyInputList: []string{policyJSON},
			ActionNames:     []string{action},
			ResourceArns:    resources,
			ContextEntries:  ctxEntries,
		}
		if pbJSON != "" {
			input.PermissionsBoundaryPolicyInputList = []string{pbJSON}
		}
		if testResourcePolicy != "" {
			input.ResourcePolicy = &testResourcePolicy
		}

		// Test-level overrides for caller ARN, resource owner, and handling option
		callerArn := scen.CallerArn
		if test.CallerArn != "" {
			callerArn = test.CallerArn
		}
		if callerArn != "" {
			rendered := RenderString(callerArn, allVars)
			input.CallerArn = &rendered
		}

		resourceOwner := scen.ResourceOwner
		if test.ResourceOwner != "" {
			resourceOwner = test.ResourceOwner
		}
		if resourceOwner != "" {
			rendered := RenderString(resourceOwner, allVars)
			input.ResourceOwner = &rendered
		}

		resourceHandlingOption := scen.ResourceHandlingOption
		if test.ResourceHandlingOption != "" {
			resourceHandlingOption = test.ResourceHandlingOption
		}
		if resourceHandlingOption != "" {
			input.ResourceHandlingOption = &resourceHandlingOption
		}

		resp, err := client.SimulateCustomPolicy(context.Background(), input)
		Check(err)
		allResponses = append(allResponses, resp)

		// Check result
		if len(resp.EvaluationResults) == 0 {
			fmt.Printf("  ✗ FAIL: no evaluation results returned\n\n")
			failCount++
			continue
		}

		result := resp.EvaluationResults[0]
		decision := string(result.EvalDecision)

		// Get matched statements for details
		detail := "-"
		if len(result.MatchedStatements) > 0 {
			parts := make([]string, 0, len(result.MatchedStatements))
			for _, m := range result.MatchedStatements {
				if m.SourcePolicyId != nil {
					parts = append(parts, AwsString(m.SourcePolicyId))
				}
			}
			if len(parts) > 0 {
				detail = strings.Join(parts, ",")
			}
		}

		// Check expectation
		if test.Expect != "" {
			if strings.EqualFold(decision, test.Expect) {
				fmt.Printf("  ✓ PASS: %s (matched: %s)\n\n", decision, detail)
				passCount++
			} else {
				// Format failure message
				if test.Name == "" {
					// Standard format: "action on resource failed: expected X, got Y"
					resourceStr := "*"
					if len(resources) > 0 {
						resourceStr = resources[0]
					}
					fmt.Printf("  ✗ FAIL: %s on %s failed: expected %s, got %s\n\n", action, resourceStr, test.Expect, decision)
				} else {
					fmt.Printf("  ✗ FAIL: expected %s, got %s (matched: %s)\n\n", test.Expect, decision, detail)
				}
				failCount++
			}
		} else {
			// No expectation, just show result
			fmt.Printf("  → Result: %s (matched: %s)\n\n", decision, detail)
			passCount++
		}
	}

	// Summary
	fmt.Printf("========================================\n")
	fmt.Printf("Test Results: %d passed, %d failed\n", passCount, failCount)
	fmt.Printf("========================================\n")

	// Save raw JSON if requested
	if savePath != "" {
		b, _ := json.MarshalIndent(allResponses, "", "  ")
		Check(os.WriteFile(savePath, b, 0o644))
		fmt.Printf("\nSaved raw responses → %s\n", savePath)
	}

	// Exit with error if any failures and not no-assert
	if failCount > 0 && !noAssert {
		GlobalExiter.Exit(2)
	}
}
