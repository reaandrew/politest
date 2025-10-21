package internal

// Scenario represents a complete test scenario loaded from YAML
type Scenario struct {
	Extends                string            `yaml:"extends"`                  // optional
	VarsFile               string            `yaml:"vars_file"`                // optional
	Vars                   map[string]any    `yaml:"vars"`                     // optional
	PolicyTemplate         string            `yaml:"policy_template"`          // OR
	PolicyJSON             string            `yaml:"policy_json"`              // mutually exclusive
	ResourcePolicyTemplate string            `yaml:"resource_policy_template"` // optional resource-based policy template
	ResourcePolicyJSON     string            `yaml:"resource_policy_json"`     // optional resource-based policy
	CallerArn              string            `yaml:"caller_arn"`               // optional IAM principal ARN to simulate as
	ResourceOwner          string            `yaml:"resource_owner"`           // optional account ARN that owns resources
	ResourceHandlingOption string            `yaml:"resource_handling_option"` // optional EC2 scenario (EC2-VPC-InstanceStore, etc)
	SCPPaths               []string          `yaml:"scp_paths"`                // optional
	Context                []ContextEntryYml `yaml:"context"`                  // optional
	Tests                  []TestCase        `yaml:"tests"`                    // required - array of test cases
}

// TestCase represents a single test case in the new collection format
type TestCase struct {
	Name                   string            `yaml:"name"`                     // descriptive test name
	Action                 string            `yaml:"action"`                   // single action to test (use this OR actions, not both)
	Actions                []string          `yaml:"actions"`                  // multiple actions to test with same resource/context (use this OR action, not both)
	Resource               string            `yaml:"resource"`                 // single resource ARN (optional, can use Resources for multiple)
	Resources              []string          `yaml:"resources"`                // multiple resources (alternative to Resource)
	Context                []ContextEntryYml `yaml:"context"`                  // optional context for this specific test
	ResourcePolicyTemplate string            `yaml:"resource_policy_template"` // optional resource policy template for this test
	ResourcePolicyJSON     string            `yaml:"resource_policy_json"`     // optional resource policy for this test
	CallerArn              string            `yaml:"caller_arn"`               // optional caller ARN override for this test
	ResourceOwner          string            `yaml:"resource_owner"`           // optional resource owner override for this test
	ResourceHandlingOption string            `yaml:"resource_handling_option"` // optional EC2 scenario override for this test
	Expect                 string            `yaml:"expect"`                   // expected decision: allowed, explicitDeny, implicitDeny
}

// ContextEntryYml represents a context key-value pair from YAML
type ContextEntryYml struct {
	ContextKeyName   string   `yaml:"ContextKeyName"`
	ContextKeyValues []string `yaml:"ContextKeyValues"`
	ContextKeyType   string   `yaml:"ContextKeyType"` // string, stringList, numeric, etc.
}

// SimulatorConfig holds configuration for running policy simulations
type SimulatorConfig struct {
	PolicyJSON          string
	PermissionsBoundary string
	ResourcePolicyJSON  string
	ScenarioPath        string // Only used by RunTestCollection
	Variables           map[string]any
	SavePath            string
	NoAssert            bool
	SourceMap           *PolicySourceMap // Tracks where statements came from
}

// PolicySourceMap tracks the origin of policy statements
type PolicySourceMap struct {
	Identity               *PolicySource            // Identity policy source
	PermissionsBoundary    map[string]*PolicySource // Map of Sid -> source for SCP/RCP statements
	ResourcePolicy         *PolicySource            // Resource policy source (scenario-level)
	PermissionsBoundaryRaw string                   // Raw merged JSON sent to AWS
	IdentityPolicyRaw      string                   // Raw identity policy JSON sent to AWS
	ResourcePolicyRaw      string                   // Raw resource policy JSON sent to AWS
}

// PolicySource tracks where a policy or statement originated
type PolicySource struct {
	FilePath string // Original file path
	Sid      string // Statement ID
	Index    int    // Statement index in original file
}
