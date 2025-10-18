package internal

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/iam"
)

// Exiter interface allows os.Exit to be mocked for testing
type Exiter interface {
	Exit(code int)
}

// RealExiter is the production implementation that calls os.Exit
type RealExiter struct{}

// Exit calls os.Exit with the given code
func (RealExiter) Exit(code int) {
	os.Exit(code)
}

// DefaultExiter is the default exiter used by the application
var DefaultExiter Exiter = RealExiter{}

// IAMSimulator interface allows IAM client to be mocked for testing
type IAMSimulator interface {
	SimulateCustomPolicy(ctx context.Context, params *iam.SimulateCustomPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulateCustomPolicyOutput, error)
}
