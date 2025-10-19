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

// Test runLegacyFormat with mocked IAM client
func TestRunLegacyFormatWithMock(t *testing.T) {
	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

	mockClient := &mockIAMClient{
		SimulateCustomPolicyFunc: func(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error) {
			// Verify inputs
			if len(params.ActionNames) != 1 || params.ActionNames[0] != action {
				t.Errorf("Expected action %s, got %v", action, params.ActionNames)
			}
			if len(params.ResourceArns) != 1 || params.ResourceArns[0] != resource {
				t.Errorf("Expected resource %s, got %v", resource, params.ResourceArns)
			}

			// Return mock response
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

	// Test that mock client works correctly
	_, err := mockClient.SimulateCustomPolicy(context.Background(), &iam.SimulateCustomPolicyInput{
		PolicyInputList: []string{policyJSON},
		ActionNames:     []string{action},
		ResourceArns:    []string{resource},
	})

	if err != nil {
		t.Errorf("Mock client returned error: %v", err)
	}
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

// Test runLegacyFormat with actual mock
func TestRunLegacyFormatIntegration(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

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

	scen := &Scenario{
		Actions:   []string{action},
		Resources: []string{resource},
		Expect: map[string]string{
			action: "allowed",
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Call runLegacyFormat
	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars})

	// Should not have exited
	if mockExit.called {
		t.Errorf("RunLegacyFormat() called Exit when expectations matched")
	}
}

func TestRunLegacyFormatWithFailure(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

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
		Actions:   []string{action},
		Resources: []string{resource},
		Expect: map[string]string{
			action: "allowed", // Expect allowed, but will get denied
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[]}`
	allVars := map[string]any{}

	// Call runLegacyFormat
	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars})

	// Should have exited with code 2
	if !mockExit.called {
		t.Error("RunLegacyFormat() did not call Exit on expectation failure")
	}
	if mockExit.exitCode != 2 {
		t.Errorf("RunLegacyFormat() called Exit with code %d, want 2", mockExit.exitCode)
	}
}

func TestRunLegacyFormatWithNoAssert(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

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
		Actions:   []string{action},
		Resources: []string{resource},
		Expect: map[string]string{
			action: "allowed", // Expect allowed, but will get denied
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[]}`
	allVars := map[string]any{}

	// Call runLegacyFormat with noAssert=true
	RunLegacyFormat(mockClient, scen, SimulatorConfig{
		PolicyJSON: policyJSON,
		Variables:  allVars,
		NoAssert:   true,
	})

	// Should NOT have exited
	if mockExit.called {
		t.Error("RunLegacyFormat() called Exit when noAssert=true")
	}
}

func TestRunLegacyFormatWithPermissionsBoundary(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

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

	scen := &Scenario{
		Actions:   []string{action},
		Resources: []string{resource},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	pbJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Action":"s3:DeleteBucket","Resource":"*"}]}`
	allVars := map[string]any{}

	// Call runLegacyFormat with permissions boundary
	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, PermissionsBoundary: pbJSON, Variables: allVars})

	// Verify permissions boundary was passed
	if capturedInput == nil {
		t.Fatal("SimulateCustomPolicy was not called")
	}
	if len(capturedInput.PermissionsBoundaryPolicyInputList) != 1 {
		t.Errorf("Expected 1 permissions boundary policy, got %d", len(capturedInput.PermissionsBoundaryPolicyInputList))
	}
	if capturedInput.PermissionsBoundaryPolicyInputList[0] != pbJSON {
		t.Error("Permissions boundary policy was not passed correctly")
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

func TestRunLegacyFormatWithResourcePolicy(t *testing.T) {
	// Save original exiter
	originalExiter := GlobalExiter
	defer func() { GlobalExiter = originalExiter }()

	mockExit := &mockExiter{}
	GlobalExiter = mockExit

	action := "s3:GetObject"
	resource := "arn:aws:s3:::my-bucket/*"

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

	scen := &Scenario{
		Actions:                []string{action},
		Resources:              []string{resource},
		CallerArn:              "arn:aws:iam::123456789012:user/testuser",
		ResourceOwner:          "123456789012",
		ResourceHandlingOption: "EC2-VPC-InstanceStore",
		Context: []ContextEntryYml{
			{ContextKeyName: "aws:userid", ContextKeyValues: []string{"user1"}, ContextKeyType: "string"},
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	resourcePolicy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Call runLegacyFormat with resource policy and other options
	RunLegacyFormat(mockClient, scen, SimulatorConfig{
		PolicyJSON:         policyJSON,
		ResourcePolicyJSON: resourcePolicy,
		Variables:          allVars,
	})

	// Verify all parameters were passed
	if capturedInput.ResourcePolicy == nil || *capturedInput.ResourcePolicy != resourcePolicy {
		t.Error("Resource policy was not passed correctly")
	}
	if capturedInput.CallerArn == nil || *capturedInput.CallerArn != scen.CallerArn {
		t.Error("CallerArn was not passed correctly")
	}
	if capturedInput.ResourceOwner == nil || *capturedInput.ResourceOwner != scen.ResourceOwner {
		t.Error("ResourceOwner was not passed correctly")
	}
	if capturedInput.ResourceHandlingOption == nil || *capturedInput.ResourceHandlingOption != scen.ResourceHandlingOption {
		t.Error("ResourceHandlingOption was not passed correctly")
	}
	if len(capturedInput.ContextEntries) == 0 {
		t.Error("Context was not passed")
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

func TestRunLegacyFormatWithSaveFile(t *testing.T) {
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
		Actions:   []string{action},
		Resources: []string{"arn:aws:s3:::bucket/*"},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	// Call runLegacyFormat with save file
	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars, SavePath: saveFile})

	// Verify save file was created
	if _, err := os.Stat(saveFile); os.IsNotExist(err) {
		t.Error("RunLegacyFormat() did not create save file")
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

// TestRunLegacyFormatWithCallerArn tests CallerArn rendering
func TestRunLegacyFormatWithCallerArn(t *testing.T) {
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

	scen := &Scenario{
		Actions:   []string{action},
		Resources: []string{"arn:aws:s3:::{{.bucket}}/*"},
		CallerArn: "arn:aws:iam::{{.account}}:user/test",
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{
		"bucket":  "my-bucket",
		"account": "123456789012",
	}

	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars})

	// Verify CallerArn was rendered and set
	if capturedInput == nil || capturedInput.CallerArn == nil {
		t.Fatal("Expected CallerArn to be set")
	}
	if *capturedInput.CallerArn != "arn:aws:iam::123456789012:user/test" {
		t.Errorf("Expected rendered CallerArn, got %s", *capturedInput.CallerArn)
	}
}

// TestRunLegacyFormatWithResourceOwner tests ResourceOwner rendering
func TestRunLegacyFormatWithResourceOwner(t *testing.T) {
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

	scen := &Scenario{
		Actions:       []string{action},
		Resources:     []string{"arn:aws:s3:::bucket/*"},
		ResourceOwner: "{{.account}}",
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{
		"account": "123456789012",
	}

	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars})

	// Verify ResourceOwner was rendered and set
	if capturedInput == nil || capturedInput.ResourceOwner == nil {
		t.Fatal("Expected ResourceOwner to be set")
	}
	if *capturedInput.ResourceOwner != "123456789012" {
		t.Errorf("Expected rendered ResourceOwner, got %s", *capturedInput.ResourceOwner)
	}
}

// TestRunLegacyFormatWithResourceHandlingOption tests ResourceHandlingOption
func TestRunLegacyFormatWithResourceHandlingOption(t *testing.T) {
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

	scen := &Scenario{
		Actions:                []string{action},
		Resources:              []string{"arn:aws:s3:::bucket/*"},
		ResourceHandlingOption: "prefix",
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars})

	// Verify ResourceHandlingOption was set
	if capturedInput == nil || capturedInput.ResourceHandlingOption == nil {
		t.Fatal("Expected ResourceHandlingOption to be set")
	}
	if *capturedInput.ResourceHandlingOption != "prefix" {
		t.Errorf("Expected ResourceHandlingOption 'prefix', got %s", *capturedInput.ResourceHandlingOption)
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

// TestRunLegacyFormatWithMatchedStatements tests the matched statements display
func TestRunLegacyFormatWithMatchedStatements(t *testing.T) {
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

	scen := &Scenario{
		Actions:   []string{action},
		Resources: []string{"arn:aws:s3:::bucket/*"},
		Expect: map[string]string{
			action: "allowed",
		},
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	allVars := map[string]any{}

	RunLegacyFormat(mockClient, scen, SimulatorConfig{PolicyJSON: policyJSON, Variables: allVars})

	// Should not have exited
	if mockExit.called {
		t.Error("RunLegacyFormat() should not exit when expectation passes")
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
