package internal

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

type mockIAMClient struct {
	SimulateCustomPolicyFunc func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error)
}

func (m *mockIAMClient) SimulateCustomPolicy(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
	return m.SimulateCustomPolicyFunc(ctx, params, optFns...)
}

// Test runTestCollection with mocked IAM client
func TestRunTestCollectionWithMock(t *testing.T) {
	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`

	// Test that mock client works
	_, err := mockClient.SimulateCustomPolicy(context.Background(), &iam.SimulateCustomPolicyInput{
		PolicyInputList: []string{policyJSON},
		ActionNames:     []string{action},
		ResourceArns:    []string{resource},
	})

	if err != nil {
		t.Errorf("Mock client returned error: %v", err)
	}
}

func TestRunTestCollectionIntegration(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	tmpDir := t.TempDir()
	action := "s3:GetObject"

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Tests: []TestCase{
			{
				Name:      "test s3 get object",
				Action:    action,
				Resources: []string{"arn:aws:s3:::my-bucket/*"},
				Expect:    "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Create a dummy scenario file path (code uses filepath.Dir on this)
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	// Call runTestCollection
	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should not have exited (all tests passed)
	if mockExit.called {
		t.Errorf("RunTestCollection() called Exit when all tests passed")
	}
}

func TestRunTestCollectionWithShowMatchedSuccess(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	tmpDir := t.TempDir()
	action := "s3:GetObject"

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName:    &action,
						EvalDecision:      types.PolicyEvaluationDecisionTypeAllowed,
						MatchedStatements: []types.Statement{},
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Tests: []TestCase{
			{
				Name:      "test s3 get object",
				Action:    action,
				Resources: []string{"arn:aws:s3:::my-bucket/*"},
				Expect:    "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[]}`
	allVars := map[string]any{}

	// Call runTestCollection with ShowMatchedSuccess enabled
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")
	RunTestCollection(mockClient, scen, SimulatorConfig{
		PolicyJSON:         policyJSON,
		ScenarioPath:       scenarioPath,
		Variables:          allVars,
		ShowMatchedSuccess: true,
	})

	// Should not have exited (all tests passed)
	if mockExit.called {
		t.Errorf("RunTestCollection() called Exit when all tests passed")
	}
}

func TestRunTestCollectionWithFailure(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	tmpDir := t.TempDir()
	action := "s3:GetObject"

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeImplicitDeny,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Tests: []TestCase{
			{
				Name:      "test s3 get object",
				Action:    action,
				Resources: []string{"arn:aws:s3:::my-bucket/*"},
				Expect:    "allowed", // Expect allowed, but will get denied
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[]}`
	allVars := map[string]any{}

	// Call runTestCollection
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")
	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should have exited with code 2 (test failed)
	if !mockExit.called {
		t.Error("RunTestCollection() did not call Exit on test failure")
	}
	if mockExit.exitCode != 2 {
		t.Errorf("RunTestCollection() called Exit with code %d, want 2", mockExit.exitCode)
	}
}

func TestRunTestCollectionMultipleTests(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	tmpDir := t.TempDir()

	callCount := 0
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			callCount++
			action := params.ActionNames[0]
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Tests: []TestCase{
			{
				Name:     "test 1",
				Action:   "s3:GetObject",
				Resource: "arn:aws:s3:::bucket1/*",
				Expect:   "allowed",
			},
			{
				Name:      "test 2",
				Action:    "s3:PutObject",
				Resources: []string{"arn:aws:s3:::bucket2/*"},
				Expect:    "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Call runTestCollection
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")
	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should have called SimulateCustomPolicy twice
	if callCount != 2 {
		t.Errorf("Expected 2 AWS API calls, got %d", callCount)
	}

	// Should not have exited
	if mockExit.called {
		t.Error("RunTestCollection() called Exit when all tests passed")
	}
}

func TestRunTestCollectionWithResourcePolicy(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	tmpDir := t.TempDir()

	// Create a resource policy file
	resourcePolicyFile := filepath.Join(tmpDir, "resource-policy.json")
	resourcePolicy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"*"}]}`
	if err := os.WriteFile(resourcePolicyFile, []byte(resourcePolicy), 0644); err != nil {
		t.Fatal(err)
	}

	var capturedInput *iam.SimulateCustomPolicyInput
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			capturedInput = params
			action := params.ActionNames[0]
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Tests: []TestCase{
			{
				Name:               "test with resource policy",
				Action:             "s3:GetObject",
				Resource:           "arn:aws:s3:::bucket/*",
				ResourcePolicyJSON: "resource-policy.json",
				Expect:             "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Create a dummy scenario file path (code uses filepath.Dir on this)
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	// Call runTestCollection
	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Verify resource policy was passed
	if capturedInput.ResourcePolicy == nil {
		t.Error("Resource policy was not passed to SimulateCustomPolicy")
	}
}

func TestRunTestCollectionWithContext(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	tmpDir := t.TempDir()

	var capturedInput *iam.SimulateCustomPolicyInput
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			capturedInput = params
			action := params.ActionNames[0]
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Context: []ContextEntryYml{
			{ContextKeyName: "aws:userid", ContextKeyValues: []string{"user1"}, ContextKeyType: "string"},
		},
		Tests: []TestCase{
			{
				Name:     "test with context",
				Action:   "s3:GetObject",
				Resource: "arn:aws:s3:::bucket/*",
				Context: []ContextEntryYml{
					{ContextKeyName: "aws:username", ContextKeyValues: []string{"testuser"}, ContextKeyType: "string"},
				},
				Expect: "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Call runTestCollection
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")
	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Verify both scenario-level and test-level context were merged
	if len(capturedInput.ContextEntries) != 2 {
		t.Errorf("Expected 2 context entries (scenario + test), got %d", len(capturedInput.ContextEntries))
	}
}

func TestRunTestCollectionWithSaveFile(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	tmpDir := t.TempDir()
	saveFile := filepath.Join(tmpDir, "output.json")

	action := "s3:GetObject"
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	scen := &Scenario{
		Tests: []TestCase{
			{
				Action:    action,
				Resources: []string{"arn:aws:s3:::bucket/*"},
				Expect:    "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	// Call runTestCollection with save file
	RunTestCollection(mockClient, scen, SimulatorConfig{
		PolicyJSON:   policyJSON,
		ScenarioPath: scenarioPath,
		Variables:    allVars,
		SavePath:     saveFile,
	})

	// Verify save file was created
	if _, err := os.Stat(saveFile); os.IsNotExist(err) {
		t.Error("RunTestCollection() did not create save file")
	}
}

// TestRunTestCollectionWithCallerArnOverride tests test-level CallerArn override
func TestRunTestCollectionWithCallerArnOverride(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	var capturedInput *iam.SimulateCustomPolicyInput
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			capturedInput = params
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		CallerArn: "arn:aws:iam::123456789012:user/scenario-user",
		Tests: []TestCase{
			{
				Action:    "s3:GetObject",
				Resource:  "arn:aws:s3:::bucket/*",
				Expect:    "allowed",
				CallerArn: "arn:aws:iam::123456789012:user/test-user",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Verify test-level CallerArn was used
	if capturedInput == nil || capturedInput.CallerArn == nil {
		t.Fatal("Expected CallerArn to be set")
	}
	if *capturedInput.CallerArn != "arn:aws:iam::123456789012:user/test-user" {
		t.Errorf("Expected test-level CallerArn, got %s", *capturedInput.CallerArn)
	}
}

// TestRunTestCollectionWithResourceOwnerOverride tests test-level ResourceOwner override
func TestRunTestCollectionWithResourceOwnerOverride(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	var capturedInput *iam.SimulateCustomPolicyInput
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			capturedInput = params
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		ResourceOwner: "123456789012",
		Tests: []TestCase{
			{
				Action:        "s3:GetObject",
				Resource:      "arn:aws:s3:::bucket/*",
				Expect:        "allowed",
				ResourceOwner: "987654321098",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Verify test-level ResourceOwner was used
	if capturedInput == nil || capturedInput.ResourceOwner == nil {
		t.Fatal("Expected ResourceOwner to be set")
	}
	if *capturedInput.ResourceOwner != "987654321098" {
		t.Errorf("Expected test-level ResourceOwner, got %s", *capturedInput.ResourceOwner)
	}
}

// TestRunTestCollectionWithResourceHandlingOptionOverride tests test-level ResourceHandlingOption override
func TestRunTestCollectionWithResourceHandlingOptionOverride(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	var capturedInput *iam.SimulateCustomPolicyInput
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			capturedInput = params
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		ResourceHandlingOption: "arn",
		Tests: []TestCase{
			{
				Action:                 "s3:GetObject",
				Resource:               "arn:aws:s3:::bucket/*",
				Expect:                 "allowed",
				ResourceHandlingOption: "prefix",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Verify test-level ResourceHandlingOption was used
	if capturedInput == nil || capturedInput.ResourceHandlingOption == nil {
		t.Fatal("Expected ResourceHandlingOption to be set")
	}
	if *capturedInput.ResourceHandlingOption != "prefix" {
		t.Errorf("Expected test-level ResourceHandlingOption, got %s", *capturedInput.ResourceHandlingOption)
	}
}

// TestRunTestCollectionNoExpectation tests test without expectation
func TestRunTestCollectionNoExpectation(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		Tests: []TestCase{
			{
				Action:   "s3:GetObject",
				Resource: "arn:aws:s3:::bucket/*",
				// No Expect field - should just show result
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should not have exited
	if mockExit.called {
		t.Error("RunTestCollection() should not exit when test has no expectation")
	}
}

// TestRunTestCollectionNamedTestFailure tests failure with custom test name
func TestRunTestCollectionNamedTestFailure(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeExplicitDeny,
					},
				},
			}, nil
		},
	}

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		Tests: []TestCase{
			{
				Name:     "Custom test name",
				Action:   "s3:GetObject",
				Resource: "arn:aws:s3:::bucket/*",
				Expect:   "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should have exited with code 2
	if !mockExit.called || mockExit.exitCode != 2 {
		t.Errorf("Expected exit code 2, got called=%v code=%d", mockExit.called, mockExit.exitCode)
	}
}

// TestRunTestCollectionWithResources tests test with Resources array (not Resource)
func TestRunTestCollectionWithResources(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	var capturedInput *iam.SimulateCustomPolicyInput
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			capturedInput = params
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		Tests: []TestCase{
			{
				Action: "s3:GetObject",
				Resources: []string{
					"arn:aws:s3:::bucket1/*",
					"arn:aws:s3:::bucket2/*",
				},
				Expect: "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Verify Resources array was used
	if capturedInput == nil || len(capturedInput.ResourceArns) != 2 {
		t.Errorf("Expected 2 resource ARNs, got %d", len(capturedInput.ResourceArns))
	}
}

// TestRunTestCollectionWithMatchedStatements tests the matched statements display in test collection
func TestRunTestCollectionWithMatchedStatements(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	sourcePolicy := "policy-1"
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
						MatchedStatements: []types.Statement{
							{
								SourcePolicyId: &sourcePolicy,
							},
						},
					},
				},
			}, nil
		},
	}

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		Tests: []TestCase{
			{
				Action:   "s3:GetObject",
				Resource: "arn:aws:s3:::bucket/*",
				Expect:   "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should not have exited
	if mockExit.called {
		t.Error("RunTestCollection() should not exit when all tests pass")
	}
}

// TestRunTestCollectionFailureWithUnnamedTest tests failure path with unnamed test
func TestRunTestCollectionFailureWithUnnamedTest(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeExplicitDeny,
					},
				},
			}, nil
		},
	}

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		Tests: []TestCase{
			{
				// No Name field - should use default format
				Action:   "s3:GetObject",
				Resource: "arn:aws:s3:::bucket/*",
				Expect:   "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should have exited with code 2
	if !mockExit.called || mockExit.exitCode != 2 {
		t.Errorf("Expected exit code 2, got called=%v code=%d", mockExit.called, mockExit.exitCode)
	}
}

func TestMergeContextEntriesWithInvalidType(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	// Set mock exiter
	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	// Create context with invalid type
	scenCtx := []ContextEntryYml{
		{
			ContextKeyName:   "aws:test",
			ContextKeyValues: []string{"value"},
			ContextKeyType:   "invalid_type",
		},
	}

	vars := map[string]any{}

	// This should trigger an error from ParseContextType
	_, err := mergeContextEntries(scenCtx, nil, vars)

	if err == nil {
		t.Error("mergeContextEntries() should return error for invalid context type")
	}
}

func TestMergeContextEntriesWithTestContextError(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	// Set mock exiter
	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	// Valid scenario context
	scenCtx := []ContextEntryYml{
		{
			ContextKeyName:   "aws:test1",
			ContextKeyValues: []string{"value1"},
			ContextKeyType:   "string",
		},
	}

	// Invalid test context
	testCtx := []ContextEntryYml{
		{
			ContextKeyName:   "aws:test2",
			ContextKeyValues: []string{"value2"},
			ContextKeyType:   "invalid_type",
		},
	}

	vars := map[string]any{}

	// This should trigger an error from ParseContextType on test context
	_, err := mergeContextEntries(scenCtx, testCtx, vars)

	if err == nil {
		t.Error("mergeContextEntries() should return error for invalid test context type")
	}
}

func TestResolveResourcePolicyConflict(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	// Set mock exiter
	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	// Create test case with both ResourcePolicyJSON and ResourcePolicyTemplate
	test := TestCase{
		Action:                 "s3:GetObject",
		Resource:               "arn:aws:s3:::bucket/*",
		ResourcePolicyJSON:     "policy.json",
		ResourcePolicyTemplate: "policy.json.tpl",
	}

	cfg := SimulatorConfig{
		ScenarioPath: scenarioPath,
		Variables:    map[string]any{},
	}

	// This should trigger Die() for conflicting resource policies
	_ = resolveResourcePolicy(test, cfg, 0)

	// Verify Die was called
	if !mockExit.called {
		t.Error("resolveResourcePolicy() did not call Die() when both resource_policy_json and resource_policy_template are set")
	}
	if mockExit.exitCode != 1 {
		t.Errorf("resolveResourcePolicy() called Exit with code %d, want 1", mockExit.exitCode)
	}
}

func TestResolveResourcePolicyWithJSON(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	// Create a resource policy JSON file
	policyFile := filepath.Join(tmpDir, "resource-policy.json")
	policyContent := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": "arn:aws:iam::123456789012:root"},
				"Action": "s3:GetObject",
				"Resource": "arn:aws:s3:::bucket/*"
			}
		]
	}`
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	test := TestCase{
		Action:             "s3:GetObject",
		Resource:           "arn:aws:s3:::bucket/*",
		ResourcePolicyJSON: "resource-policy.json",
	}

	cfg := SimulatorConfig{
		ScenarioPath: scenarioPath,
		Variables:    map[string]any{},
	}

	result := resolveResourcePolicy(test, cfg, 0)

	// Verify result is valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("resolveResourcePolicy() produced invalid JSON: %v", err)
	}

	// Verify it's minified (no whitespace)
	if strings.Contains(result, "\n") || strings.Contains(result, "  ") {
		t.Error("resolveResourcePolicy() should return minified JSON")
	}
}

func TestResolveResourcePolicyWithTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	// Create a resource policy template file
	templateFile := filepath.Join(tmpDir, "resource-policy.json.tpl")
	templateContent := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": "arn:aws:iam::{{.account}}:root"},
				"Action": "s3:GetObject",
				"Resource": "arn:aws:s3:::{{.bucket}}/*"
			}
		]
	}`
	if err := os.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
		t.Fatal(err)
	}

	test := TestCase{
		Action:                 "s3:GetObject",
		Resource:               "arn:aws:s3:::test-bucket/*",
		ResourcePolicyTemplate: "resource-policy.json.tpl",
	}

	cfg := SimulatorConfig{
		ScenarioPath: scenarioPath,
		Variables: map[string]any{
			"account": "123456789012",
			"bucket":  "test-bucket",
		},
	}

	result := resolveResourcePolicy(test, cfg, 0)

	// Verify result is valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("resolveResourcePolicy() produced invalid JSON: %v", err)
	}

	// Verify template variables were rendered
	if !strings.Contains(result, "123456789012") {
		t.Error("resolveResourcePolicy() did not render account variable")
	}
	if !strings.Contains(result, "test-bucket") {
		t.Error("resolveResourcePolicy() did not render bucket variable")
	}
}

func TestResolveResourcePolicyWithScenarioDefault(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	// Test case with no resource policy specified - should use scenario default
	test := TestCase{
		Action:   "s3:GetObject",
		Resource: "arn:aws:s3:::bucket/*",
	}

	scenarioResourcePolicy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:*","Resource":"*"}]}`

	cfg := SimulatorConfig{
		ScenarioPath:       scenarioPath,
		ResourcePolicyJSON: scenarioResourcePolicy,
		Variables:          map[string]any{},
	}

	result := resolveResourcePolicy(test, cfg, 0)

	if result != scenarioResourcePolicy {
		t.Errorf("resolveResourcePolicy() = %v, want scenario default %v", result, scenarioResourcePolicy)
	}
}

func TestExpandTestsWithActions(t *testing.T) {
	tests := []struct {
		name    string
		input   []TestCase
		wantLen int
		wantErr bool
	}{
		{
			name: "single action - no expansion",
			input: []TestCase{
				{
					Name:     "Test 1",
					Action:   "s3:GetObject",
					Resource: "arn:aws:s3:::bucket/*",
					Expect:   "allowed",
				},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "actions array - expands to multiple tests",
			input: []TestCase{
				{
					Name: "Multiple actions",
					Actions: []string{
						"s3:GetObject",
						"s3:PutObject",
						"s3:ListBucket",
					},
					Resource: "arn:aws:s3:::bucket/*",
					Expect:   "allowed",
				},
			},
			wantLen: 3,
			wantErr: false,
		},
		{
			name: "mix of single and multiple actions",
			input: []TestCase{
				{
					Name:     "Single",
					Action:   "s3:GetObject",
					Resource: "arn:aws:s3:::bucket/*",
					Expect:   "allowed",
				},
				{
					Name: "Multiple",
					Actions: []string{
						"s3:PutObject",
						"s3:DeleteObject",
					},
					Resource: "arn:aws:s3:::bucket/*",
					Expect:   "explicitDeny",
				},
			},
			wantLen: 3, // 1 + 2
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandTestsWithActions(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("expandTestsWithActions() returned %d tests, want %d", len(result), tt.wantLen)
			}

			// Verify all expanded tests have Action set (not Actions)
			for i, test := range result {
				if test.Action == "" {
					t.Errorf("expanded test[%d] has empty Action", i)
				}
				if len(test.Actions) > 0 {
					t.Errorf("expanded test[%d] still has Actions array", i)
				}
			}
		})
	}
}

func TestExpandTestsWithActionsBothActionAndActions(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	// Set mock exiter
	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	tests := []TestCase{
		{
			Name:     "Invalid - both action and actions",
			Action:   "s3:GetObject",
			Actions:  []string{"s3:PutObject"},
			Resource: "arn:aws:s3:::bucket/*",
		},
	}

	// This should trigger Die()
	_ = expandTestsWithActions(tests)

	if !mockExit.called {
		t.Error("expandTestsWithActions() did not call Die() when both action and actions are specified")
	}
	if mockExit.exitCode != 1 {
		t.Errorf("expandTestsWithActions() called Exit with code %d, want 1", mockExit.exitCode)
	}
}

func TestExpandTestsWithActionsNoAction(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	// Set mock exiter
	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	tests := []TestCase{
		{
			Name:     "Invalid - no action or actions",
			Resource: "arn:aws:s3:::bucket/*",
		},
	}

	// This should trigger Die()
	_ = expandTestsWithActions(tests)

	if !mockExit.called {
		t.Error("expandTestsWithActions() did not call Die() when neither action nor actions are specified")
	}
	if mockExit.exitCode != 1 {
		t.Errorf("expandTestsWithActions() called Exit with code %d, want 1", mockExit.exitCode)
	}
}

func TestRunTestCollectionWithActionsArray(t *testing.T) {
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	callCount := 0
	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			callCount++
			action := params.ActionNames[0]
			return &iam.SimulateCustomPolicyOutput{
				EvaluationResults: []types.EvaluationResult{
					{
						EvalActionName: &action,
						EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
					},
				},
			}, nil
		},
	}

	tmpDir := t.TempDir()
	scenarioPath := filepath.Join(tmpDir, "scenario.yml")

	scen := &Scenario{
		Tests: []TestCase{
			{
				Name: "Multiple actions test",
				Actions: []string{
					"s3:GetObject",
					"s3:PutObject",
					"s3:ListBucket",
				},
				Resource: "arn:aws:s3:::bucket/*",
				Expect:   "allowed",
			},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunTestCollection(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, ScenarioPath: scenarioPath, Variables: allVars})

	// Should have called AWS API 3 times (one for each action)
	if callCount != 3 {
		t.Errorf("Expected 3 AWS API calls, got %d", callCount)
	}

	// Should not have exited
	if mockExit.called {
		t.Errorf("RunTestCollection() exited unexpectedly with code %d", mockExit.exitCode)
	}
}

func TestExtractSidFromJSON(t *testing.T) {
	tests := []struct {
		name     string
		stmtJSON string
		want     string
	}{
		{
			name:     "valid statement with Sid",
			stmtJSON: `{"Sid": "DenyS3", "Effect": "Deny", "Action": "s3:*", "Resource": "*"}`,
			want:     "DenyS3",
		},
		{
			name:     "statement without Sid",
			stmtJSON: `{"Effect": "Allow", "Action": "s3:*", "Resource": "*"}`,
			want:     "",
		},
		{
			name:     "invalid JSON",
			stmtJSON: `not valid json`,
			want:     "",
		},
		{
			name:     "Sid is not a string",
			stmtJSON: `{"Sid": 123, "Effect": "Deny"}`,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSidFromJSON(tt.stmtJSON)
			if got != tt.want {
				t.Errorf("extractSidFromJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractStatementFromPolicy(t *testing.T) {
	policyJSON := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DenyS3",
      "Effect": "Deny",
      "Action": "s3:*",
      "Resource": "*"
    }
  ]
}`

	tests := []struct {
		name      string
		start     *types.Position
		end       *types.Position
		wantEmpty bool
	}{
		{
			name:  "extract multi line statement",
			start: &types.Position{Line: 4, Column: 5},
			end:   &types.Position{Line: 9, Column: 6},
		},
		{
			name:      "nil start position",
			start:     nil,
			end:       &types.Position{Line: 5, Column: 10},
			wantEmpty: true,
		},
		{
			name:      "nil end position",
			start:     &types.Position{Line: 5, Column: 10},
			end:       nil,
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractStatementFromPolicy(policyJSON, tt.start, tt.end)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("extractStatementFromPolicy() = %v, want empty string", result)
				}
			} else {
				if result == "" {
					t.Error("extractStatementFromPolicy() returned empty string, want non-empty")
				}
			}
		})
	}
}

func TestPrintTestFailureWithSingleResource(t *testing.T) {
	// Capture stdout to verify output
	test := TestCase{
		Action:   "s3:GetObject",
		Resource: "arn:aws:s3:::bucket/*",
		Expect:   "allowed",
	}

	cfg := SimulatorConfig{}

	// This will print to stdout - we're just verifying it doesn't crash
	printTestFailure(test, "s3:GetObject", []string{"arn:aws:s3:::bucket/*"}, "explicitDeny", "policy1", []types.Statement{}, cfg)
}

func TestPrintTestFailureWithMultipleResources(t *testing.T) {
	test := TestCase{
		Action: "s3:ListBucket",
		Resources: []string{
			"arn:aws:s3:::bucket1",
			"arn:aws:s3:::bucket2",
			"arn:aws:s3:::bucket3",
		},
		Expect: "allowed",
	}

	cfg := SimulatorConfig{}

	printTestFailure(test, "s3:ListBucket", []string{"arn:aws:s3:::bucket1", "arn:aws:s3:::bucket2", "arn:aws:s3:::bucket3"}, "explicitDeny", "policy1", []types.Statement{}, cfg)
}

func TestPrintTestFailureWithContext(t *testing.T) {
	test := TestCase{
		Action:   "s3:PutObject",
		Resource: "arn:aws:s3:::secure-bucket/*",
		Context: []ContextEntryYml{
			{ContextKeyName: "aws:SourceIp", ContextKeyValues: []string{"10.0.1.50"}},
			{ContextKeyName: "aws:MultiFactorAuthPresent", ContextKeyValues: []string{"true"}},
		},
		Expect: "allowed",
	}

	cfg := SimulatorConfig{}

	printTestFailure(test, "s3:PutObject", []string{"arn:aws:s3:::secure-bucket/*"}, "explicitDeny", "policy1", []types.Statement{}, cfg)
}

func TestPrintTestFailureWithMultipleContextValues(t *testing.T) {
	test := TestCase{
		Action:   "s3:GetObject",
		Resource: "arn:aws:s3:::bucket/*",
		Context: []ContextEntryYml{
			{ContextKeyName: "aws:PrincipalTag/Department", ContextKeyValues: []string{"Engineering", "Sales", "Marketing"}},
		},
		Expect: "allowed",
	}

	cfg := SimulatorConfig{}

	printTestFailure(test, "s3:GetObject", []string{"arn:aws:s3:::bucket/*"}, "implicitDeny", "policy1", []types.Statement{}, cfg)
}

func TestEvaluateTestResultNoEvaluationResults(t *testing.T) {
	resp := &iam.SimulateCustomPolicyOutput{
		EvaluationResults: []types.EvaluationResult{},
	}

	test := TestCase{
		Action: "s3:GetObject",
		Expect: "allowed",
	}

	cfg := SimulatorConfig{}

	result := evaluateTestResult(resp, test, "s3:GetObject", []string{"arn:aws:s3:::bucket/*"}, cfg)
	if result {
		t.Error("Expected evaluateTestResult to return false when no evaluation results")
	}
}

func TestEvaluateTestResultNoExpectation(t *testing.T) {
	action := "s3:GetObject"
	resp := &iam.SimulateCustomPolicyOutput{
		EvaluationResults: []types.EvaluationResult{
			{
				EvalActionName:    &action,
				EvalDecision:      types.PolicyEvaluationDecisionTypeAllowed,
				MatchedStatements: []types.Statement{},
			},
		},
	}

	test := TestCase{
		Action: "s3:GetObject",
		Expect: "", // No expectation
	}

	cfg := SimulatorConfig{}

	result := evaluateTestResult(resp, test, "s3:GetObject", []string{"arn:aws:s3:::bucket/*"}, cfg)
	if !result {
		t.Error("Expected evaluateTestResult to return true when no expectation is set")
	}
}

func TestEvaluateTestResultWithShowMatchedSuccessFalse(t *testing.T) {
	action := "s3:GetObject"
	resp := &iam.SimulateCustomPolicyOutput{
		EvaluationResults: []types.EvaluationResult{
			{
				EvalActionName:    &action,
				EvalDecision:      types.PolicyEvaluationDecisionTypeAllowed,
				MatchedStatements: []types.Statement{},
			},
		},
	}

	test := TestCase{
		Action: "s3:GetObject",
		Expect: "allowed",
	}

	cfg := SimulatorConfig{
		ShowMatchedSuccess: false,
	}

	result := evaluateTestResult(resp, test, "s3:GetObject", []string{"arn:aws:s3:::bucket/*"}, cfg)
	if !result {
		t.Error("Expected evaluateTestResult to return true when test passes")
	}
}

func TestEvaluateTestResultWithShowMatchedSuccessTrue(t *testing.T) {
	action := "s3:GetObject"
	resp := &iam.SimulateCustomPolicyOutput{
		EvaluationResults: []types.EvaluationResult{
			{
				EvalActionName:    &action,
				EvalDecision:      types.PolicyEvaluationDecisionTypeAllowed,
				MatchedStatements: []types.Statement{},
			},
		},
	}

	test := TestCase{
		Action: "s3:GetObject",
		Expect: "allowed",
	}

	cfg := SimulatorConfig{
		ShowMatchedSuccess: true,
	}

	result := evaluateTestResult(resp, test, "s3:GetObject", []string{"arn:aws:s3:::bucket/*"}, cfg)
	if !result {
		t.Error("Expected evaluateTestResult to return true when test passes")
	}
}

func TestEvaluateTestResultMismatch(t *testing.T) {
	action := "s3:GetObject"
	resp := &iam.SimulateCustomPolicyOutput{
		EvaluationResults: []types.EvaluationResult{
			{
				EvalActionName:    &action,
				EvalDecision:      types.PolicyEvaluationDecisionTypeImplicitDeny,
				MatchedStatements: []types.Statement{},
			},
		},
	}

	test := TestCase{
		Action: "s3:GetObject",
		Expect: "allowed",
	}

	cfg := SimulatorConfig{}

	result := evaluateTestResult(resp, test, "s3:GetObject", []string{"arn:aws:s3:::bucket/*"}, cfg)
	if result {
		t.Error("Expected evaluateTestResult to return false when test fails")
	}
}

func TestPrintTestDetailsWithSingleResource(t *testing.T) {
	// Capture stdout to verify output
	test := TestCase{
		Action:   "s3:GetObject",
		Resource: "arn:aws:s3:::bucket/*",
		Expect:   "allowed",
	}

	cfg := SimulatorConfig{}

	// This will print to stdout - we're just verifying it doesn't crash
	printTestDetails(test, "s3:GetObject", []string{"arn:aws:s3:::bucket/*"}, "allowed", []types.Statement{}, cfg)
}

func TestPrintTestDetailsWithNoResources(t *testing.T) {
	test := TestCase{
		Action: "iam:ListUsers",
		Expect: "allowed",
	}

	cfg := SimulatorConfig{}

	printTestDetails(test, "iam:ListUsers", []string{}, "allowed", []types.Statement{}, cfg)
}

func TestPrintTestSuccessWithSingleResource(t *testing.T) {
	// Capture stdout to verify output
	test := TestCase{
		Action:   "s3:GetObject",
		Resource: "arn:aws:s3:::bucket/*",
		Expect:   "allowed",
	}

	cfg := SimulatorConfig{}

	// This will print to stdout - we're just verifying it doesn't crash
	printTestSuccess(test, "s3:GetObject", []string{"arn:aws:s3:::bucket/*"}, "allowed", "policy1", []types.Statement{}, cfg)
}

func TestPrintTestSuccessWithMultipleResources(t *testing.T) {
	test := TestCase{
		Action: "s3:ListBucket",
		Resources: []string{
			"arn:aws:s3:::bucket1",
			"arn:aws:s3:::bucket2",
			"arn:aws:s3:::bucket3",
		},
		Expect: "allowed",
	}

	cfg := SimulatorConfig{}

	printTestSuccess(test, "s3:ListBucket", []string{"arn:aws:s3:::bucket1", "arn:aws:s3:::bucket2", "arn:aws:s3:::bucket3"}, "allowed", "policy1", []types.Statement{}, cfg)
}

func TestPrintTestSuccessWithContext(t *testing.T) {
	test := TestCase{
		Action:   "s3:PutObject",
		Resource: "arn:aws:s3:::secure-bucket/*",
		Context: []ContextEntryYml{
			{ContextKeyName: "aws:SourceIp", ContextKeyValues: []string{"10.0.1.50"}},
			{ContextKeyName: "aws:MultiFactorAuthPresent", ContextKeyValues: []string{"true"}},
		},
		Expect: "allowed",
	}

	cfg := SimulatorConfig{}

	printTestSuccess(test, "s3:PutObject", []string{"arn:aws:s3:::secure-bucket/*"}, "allowed", "policy1", []types.Statement{}, cfg)
}

func TestPrintTestSuccessWithMultipleContextValues(t *testing.T) {
	test := TestCase{
		Action:   "s3:GetObject",
		Resource: "arn:aws:s3:::bucket/*",
		Context: []ContextEntryYml{
			{ContextKeyName: "aws:PrincipalTag/Department", ContextKeyValues: []string{"Engineering", "Sales", "Marketing"}},
		},
		Expect: "allowed",
	}

	cfg := SimulatorConfig{}

	printTestSuccess(test, "s3:GetObject", []string{"arn:aws:s3:::bucket/*"}, "allowed", "policy1", []types.Statement{}, cfg)
}

func TestPrintTestSuccessWithSourceMap(t *testing.T) {
	tmpDir := t.TempDir()

	policyFile := filepath.Join(tmpDir, "policy.json")
	policyContent := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowS3",
      "Effect": "Allow",
      "Action": "s3:*",
      "Resource": "*"
    }
  ]
}`
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	trackingSid := "identity#stmt:0"
	sourceMap := &PolicySourceMap{
		Identity: map[string]*PolicySource{
			trackingSid: {
				FilePath:  policyFile,
				Sid:       "AllowS3",
				StartLine: 4,
				EndLine:   9,
			},
		},
		IdentityPolicyRaw: `{"Version":"2012-10-17","Statement":[{"Sid":"identity#stmt:0","Effect":"Allow","Action":"s3:*","Resource":"*"}]}`,
	}

	test := TestCase{
		Action:   "s3:GetObject",
		Resource: "arn:aws:s3:::bucket/*",
		Expect:   "allowed",
	}

	cfg := SimulatorConfig{
		SourceMap: sourceMap,
	}

	sourcePolicyID := "PolicyInputList.1"
	line4, col5 := int32(4), int32(5)
	line9, col6 := int32(9), int32(6)
	matchedStmts := []types.Statement{
		{
			SourcePolicyId: &sourcePolicyID,
			StartPosition:  &types.Position{Line: line4, Column: col5},
			EndPosition:    &types.Position{Line: line9, Column: col6},
		},
	}

	printTestSuccess(test, "s3:GetObject", []string{"arn:aws:s3:::bucket/*"}, "allowed", "policy1", matchedStmts, cfg)
}

func TestPrintTestDetailsWithSourceMap(t *testing.T) {
	tmpDir := t.TempDir()

	policyFile := filepath.Join(tmpDir, "policy.json")
	policyContent := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowS3",
      "Effect": "Allow",
      "Action": "s3:*",
      "Resource": "*"
    }
  ]
}`
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	trackingSid := "identity#stmt:0"
	sourceMap := &PolicySourceMap{
		Identity: map[string]*PolicySource{
			trackingSid: {
				FilePath:  policyFile,
				Sid:       "AllowS3",
				StartLine: 4,
				EndLine:   9,
			},
		},
		IdentityPolicyRaw: `{"Version":"2012-10-17","Statement":[{"Sid":"identity#stmt:0","Effect":"Allow","Action":"s3:*","Resource":"*"}]}`,
	}

	test := TestCase{
		Action:   "s3:GetObject",
		Resource: "arn:aws:s3:::bucket/*",
		Expect:   "allowed",
		Context: []ContextEntryYml{
			{ContextKeyName: "aws:SourceIp", ContextKeyValues: []string{"10.0.1.50"}},
		},
	}

	cfg := SimulatorConfig{
		SourceMap: sourceMap,
	}

	sourcePolicyID := "PolicyInputList.1"
	line4, col5 := int32(4), int32(5)
	line9, col6 := int32(9), int32(6)
	matchedStmts := []types.Statement{
		{
			SourcePolicyId: &sourcePolicyID,
			StartPosition:  &types.Position{Line: line4, Column: col5},
			EndPosition:    &types.Position{Line: line9, Column: col6},
		},
	}

	printTestDetails(test, "s3:GetObject", []string{"arn:aws:s3:::bucket/*"}, "allowed", matchedStmts, cfg)
}

func TestProcessIdentityPolicyWithSourceMap(t *testing.T) {
	tmpDir := t.TempDir()

	policyFile := filepath.Join(tmpDir, "policy.json")
	policyContent := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowS3GetObject",
      "Effect": "Allow",
      "Action": "s3:GetObject",
      "Resource": "*"
    },
    {
      "Sid": "AllowS3PutObject",
      "Effect": "Allow",
      "Action": "s3:PutObject",
      "Resource": "*"
    }
  ]
}`
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Read and minify the policy
	policyBytes, err := os.ReadFile(policyFile)
	if err != nil {
		t.Fatal(err)
	}
	policyJSON := MinifyJSON(policyBytes)

	// Process the policy
	modifiedJSON, sourceMap := ProcessIdentityPolicyWithSourceMap(policyJSON, policyFile)

	// Verify source map was created
	if len(sourceMap) != 2 {
		t.Errorf("Expected 2 entries in source map, got %d", len(sourceMap))
	}

	// Verify tracking Sids were injected
	if !strings.Contains(modifiedJSON, "identity#stmt:0") {
		t.Error("Expected tracking Sid 'identity#stmt:0' in modified JSON")
	}
	if !strings.Contains(modifiedJSON, "identity#stmt:1") {
		t.Error("Expected tracking Sid 'identity#stmt:1' in modified JSON")
	}

	// Verify source info for first statement
	if source, ok := sourceMap["identity#stmt:0"]; ok {
		if source.Sid != "AllowS3GetObject" {
			t.Errorf("Expected original Sid 'AllowS3GetObject', got '%s'", source.Sid)
		}
		if source.FilePath != policyFile {
			t.Errorf("Expected file path '%s', got '%s'", policyFile, source.FilePath)
		}
		if source.StartLine == 0 {
			t.Error("Expected non-zero start line")
		}
	} else {
		t.Error("Expected source map entry for 'identity#stmt:0'")
	}
}

func TestDisplayMatchedStatementsWithSourceMap(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a source file
	scpFile := filepath.Join(tmpDir, "deny-s3.json")
	scpContent := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DenyS3",
      "Effect": "Deny",
      "Action": "s3:*",
      "Resource": "*"
    }
  ]
}`
	if err := os.WriteFile(scpFile, []byte(scpContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Build source map
	sourceMap := &PolicySourceMap{
		PermissionsBoundary: map[string]*PolicySource{
			"scp:deny-s3.json#stmt:0": {
				FilePath:  scpFile,
				Sid:       "DenyS3",
				Index:     0,
				StartLine: 4,
				EndLine:   9,
			},
		},
		PermissionsBoundaryRaw: `{"Version":"2012-10-17","Statement":[{"Sid":"scp:deny-s3.json#stmt:0","Effect":"Deny","Action":"s3:*","Resource":"*"}]}`,
	}

	cfg := SimulatorConfig{
		SourceMap: sourceMap,
	}

	sourcePolicyID := "PermissionsBoundaryPolicyInputList.1"
	line4, col5 := int32(4), int32(5)
	line9, col6 := int32(9), int32(6)

	matchedStatements := []types.Statement{
		{
			SourcePolicyId: &sourcePolicyID,
			StartPosition:  &types.Position{Line: line4, Column: col5},
			EndPosition:    &types.Position{Line: line9, Column: col6},
		},
	}

	// This will print to stdout - we're verifying it doesn't crash and exercises the code
	displayMatchedStatements(matchedStatements, cfg)
}

func TestDisplaySingleStatementIdentityPolicy(t *testing.T) {
	tmpDir := t.TempDir()

	policyFile := filepath.Join(tmpDir, "policy.json")
	policyContent := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowS3",
      "Effect": "Allow",
      "Action": "s3:*",
      "Resource": "*"
    }
  ]
}`
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	trackingSid := "identity#stmt:0"
	sourceMap := &PolicySourceMap{
		Identity: map[string]*PolicySource{
			trackingSid: {
				FilePath:  policyFile,
				Sid:       "AllowS3",
				StartLine: 4,
				EndLine:   9,
			},
		},
		IdentityPolicyRaw: `{"Version":"2012-10-17","Statement":[{"Sid":"identity#stmt:0","Effect":"Allow","Action":"s3:*","Resource":"*"}]}`,
	}

	cfg := SimulatorConfig{
		SourceMap: sourceMap,
	}

	sourcePolicyID := "PolicyInputList.1"
	line4, col5 := int32(4), int32(5)
	line9, col6 := int32(9), int32(6)
	stmt := types.Statement{
		SourcePolicyId: &sourcePolicyID,
		StartPosition:  &types.Position{Line: line4, Column: col5},
		EndPosition:    &types.Position{Line: line9, Column: col6},
	}

	displaySingleStatement(stmt, cfg)
}

func TestDisplaySingleStatementResourcePolicy(t *testing.T) {
	tmpDir := t.TempDir()

	policyFile := filepath.Join(tmpDir, "resource-policy.json")
	policyContent := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DenyDelete",
      "Effect": "Deny",
      "Principal": {"AWS": "*"},
      "Action": "s3:DeleteObject",
      "Resource": "*"
    }
  ]
}`
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	sourceMap := &PolicySourceMap{
		ResourcePolicy: &PolicySource{
			FilePath:  policyFile,
			Sid:       "DenyDelete",
			StartLine: 4,
			EndLine:   10,
		},
		ResourcePolicyRaw: `{"Version":"2012-10-17","Statement":[{"Sid":"DenyDelete","Effect":"Deny","Principal":{"AWS":"*"},"Action":"s3:DeleteObject","Resource":"*"}]}`,
	}

	cfg := SimulatorConfig{
		SourceMap: sourceMap,
	}

	sourcePolicyID := "ResourcePolicy.1"
	stmt := types.Statement{
		SourcePolicyId: &sourcePolicyID,
	}

	displaySingleStatement(stmt, cfg)
}

func TestDisplaySingleStatementUnknownSource(t *testing.T) {
	cfg := SimulatorConfig{
		SourceMap: &PolicySourceMap{},
	}

	sourcePolicyID := "UnknownPolicyType.1"
	stmt := types.Statement{
		SourcePolicyId: &sourcePolicyID,
	}

	displaySingleStatement(stmt, cfg)
}

func TestDisplaySingleStatementNilSourcePolicyId(t *testing.T) {
	cfg := SimulatorConfig{
		SourceMap: &PolicySourceMap{},
	}

	stmt := types.Statement{
		SourcePolicyId: nil,
	}

	// Should return early without printing
	displaySingleStatement(stmt, cfg)
}

func TestDisplayStatementWithContextValidSource(t *testing.T) {
	tmpDir := t.TempDir()

	sourceFile := filepath.Join(tmpDir, "test-policy.json")
	sourceContent := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "TestStatement",
      "Effect": "Allow",
      "Action": "s3:GetObject",
      "Resource": "*"
    }
  ]
}`
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatal(err)
	}

	source := &PolicySource{
		FilePath:  sourceFile,
		Sid:       "TestStatement",
		StartLine: 4,
		EndLine:   9,
	}

	displayStatementWithContext(source)
}

func TestDisplayStatementWithContextNoLineNumbers(t *testing.T) {
	source := &PolicySource{
		FilePath:  "/some/path",
		Sid:       "TestSid",
		StartLine: 0,
		EndLine:   0,
	}

	// Should return early without trying to read file
	displayStatementWithContext(source)
}

func TestDisplayStatementWithContextFileReadError(t *testing.T) {
	source := &PolicySource{
		FilePath:  "/nonexistent/path/file.json",
		Sid:       "TestSid",
		StartLine: 1,
		EndLine:   5,
	}

	// Should return early on file read error
	displayStatementWithContext(source)
}

func TestDisplayMatchedStatementsNoSourceMap(t *testing.T) {
	cfg := SimulatorConfig{
		SourceMap: nil,
	}

	sourcePolicyID := "PolicyInputList.1"
	matchedStatements := []types.Statement{
		{
			SourcePolicyId: &sourcePolicyID,
		},
	}

	// Should return early without source map
	displayMatchedStatements(matchedStatements, cfg)
}

func TestDisplayMatchedStatementsEmpty(t *testing.T) {
	cfg := SimulatorConfig{
		SourceMap: &PolicySourceMap{},
	}

	matchedStatements := []types.Statement{}

	// Should return early with empty statements
	displayMatchedStatements(matchedStatements, cfg)
}
