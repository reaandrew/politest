# politest Architecture

## Overview

politest is a single-binary Go CLI tool for testing AWS IAM policies using AWS's `SimulateCustomPolicy` API. The tool provides a YAML-based DSL for defining policy test scenarios with support for templating, inheritance, and Service Control Policy (SCP) merging.

**Core Purpose:** Enable infrastructure-as-code testing for IAM policies before deployment.

## High-Level Design

```
┌─────────────────────────────────────────────────────────────────┐
│                         politest CLI                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Scenario Loader                              │
│  - Loads YAML scenario files                                    │
│  - Resolves 'extends' inheritance (recursive)                   │
│  - Merges parent/child scenarios                                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Template Renderer                              │
│  - Renders Go text/template in policies, actions, resources     │
│  - Substitutes variables from vars_file + inline vars           │
│  - Supports both JSON and template formats                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Policy Simulator                              │
│  - Calls AWS IAM SimulateCustomPolicy API                       │
│  - Supports two test formats:                                   │
│    * Legacy: actions + resources + expect map                   │
│    * Collection: array of named test cases                      │
│  - Merges SCPs as permissions boundary                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Result Evaluator                               │
│  - Compares actual vs expected decisions                        │
│  - Prints formatted output (table or test results)              │
│  - Exits with code 2 on assertion failures                      │
└─────────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. Entry Point (`main.go`)

**Responsibilities:**
- Parse command-line flags (`--scenario`, `--save`, `--no-assert`)
- Load scenario file with inheritance
- Build variables map (vars_file + inline vars)
- Resolve policy documents (template or JSON)
- Merge SCP files into permissions boundary
- Initialize AWS IAM client
- Dispatch to appropriate simulator (legacy or collection format)

**Key Functions:**
- `main()` - Entry point, orchestrates entire flow

### 2. Scenario Management (`internal/scenario.go`)

**Responsibilities:**
- Load YAML scenario files
- Resolve `extends` inheritance recursively
- Merge parent and child scenarios with override semantics

**Key Types:**
```go
type Scenario struct {
    Extends                string            // Parent scenario path
    VarsFile               string            // External variables file
    Vars                   map[string]any    // Inline variables
    PolicyTemplate         string            // Go template path
    PolicyJSON             string            // Pre-rendered JSON path
    ResourcePolicyTemplate string            // Resource policy template
    ResourcePolicyJSON     string            // Resource policy JSON
    SCPPaths               []string          // Glob patterns for SCPs
    Actions                []string          // Actions to test (legacy)
    Resources              []string          // Resources to test (legacy)
    Context                []ContextEntryYml // IAM context keys
    Expect                 map[string]string // Expected results (legacy)
    Tests                  []TestCase        // Named test cases (collection)
    CallerArn              string            // Principal ARN
    ResourceOwner          string            // Resource owner account
    ResourceHandlingOption string            // EC2 scenario type
}

type TestCase struct {
    Name                   string            // Descriptive name
    Action                 string            // Single action
    Resource               string            // Single resource
    Resources              []string          // Multiple resources
    Context                []ContextEntryYml // Test-specific context
    ResourcePolicyTemplate string            // Test-level resource policy
    ResourcePolicyJSON     string            // Test-level resource policy
    CallerArn              string            // Test-level caller override
    ResourceOwner          string            // Test-level owner override
    ResourceHandlingOption string            // Test-level EC2 scenario
    Expect                 string            // Expected decision
}
```

**Key Functions:**
- `LoadScenarioWithExtends(path)` - Loads scenario with recursive parent resolution
- `MergeScenario(parent, child)` - Merges two scenarios with child override
- Helper functions: `mergePolicyFields`, `mergeSliceFields`, `mergeMapFields`, etc.

### 3. Template Rendering (`internal/template.go`)

**Responsibilities:**
- Render Go text/template strings with variable substitution
- Support for policies, actions, resources, and context values
- Minify JSON output

**Key Functions:**
- `RenderString(template, vars)` - Renders single template string
- `RenderStringSlice(templates, vars)` - Renders slice of strings
- `RenderContext(entries, vars)` - Renders IAM context entries
- `RenderTemplateFileJSON(path, vars)` - Renders template file to minified JSON

**Template Syntax:**
```yaml
vars:
  bucket: "my-bucket"
  account: "123456789012"

resources:
  - "arn:aws:s3:::{{.bucket}}/*"
  - "arn:aws:s3:::{{.bucket}}"

policy_template: |
  {
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::{{.bucket}}/*",
      "Condition": {
        "StringEquals": {
          "aws:PrincipalAccount": "{{.account}}"
        }
      }
    }]
  }
```

### 4. Policy Merging (`internal/policy.go`)

**Responsibilities:**
- Expand glob patterns for SCP files
- Merge multiple SCP JSON files into single policy document
- Minify JSON for AWS API

**Key Functions:**
- `ExpandGlobsRelative(base, patterns)` - Expands glob patterns to file paths
- `MergeSCPFiles(files)` - Merges SCP files into single policy
- `MinifyJSON(bytes)` - Compacts JSON
- `ReadJSONFile(path, v)` - Reads and decodes JSON

**SCP Merging Logic:**
```go
// Merges statements from multiple files:
// scp/deny-root.json    → Statement 1
// scp/require-mfa.json  → Statement 2
// Result: { "Statement": [Statement1, Statement2] }
```

### 5. Policy Simulation (`internal/simulator.go`)

**Responsibilities:**
- Execute IAM policy simulations via AWS API
- Support both legacy and collection test formats
- Build AWS API input structures
- Evaluate results against expectations
- Format output (tables or test results)

**Key Functions:**

#### Legacy Format:
- `RunLegacyFormat(client, scenario, config)` - Executes legacy format tests
- `buildSimulateInput()` - Builds IAM API input
- `processAndDisplayResults()` - Formats results as table
- `checkExpectationsAndExit()` - Validates expectations

#### Collection Format:
- `RunTestCollection(client, scenario, config)` - Executes test collection
- `runSingleTest()` - Executes one test case
- `prepareTestResources()` - Resolves test resources
- `mergeContextEntries()` - Merges scenario + test context
- `resolveResourcePolicy()` - Determines resource policy for test
- `evaluateTestResult()` - Checks result vs expectation
- `printTestSummary()` - Prints pass/fail summary

**Configuration:**
```go
type SimulatorConfig struct {
    PolicyJSON          string         // Identity-based policy
    PermissionsBoundary string         // Merged SCPs
    ResourcePolicyJSON  string         // Resource-based policy
    ScenarioPath        string         // For resolving relative paths
    Variables           map[string]any // Template variables
    SavePath            string         // Save raw AWS response
    NoAssert            bool           // Skip assertion failures
}
```

### 6. Helper Utilities (`internal/helpers.go`)

**Responsibilities:**
- Error handling with exit codes
- Path resolution
- AWS string pointer handling
- Table formatting

**Key Functions:**
- `Check(err)` - Exits on error
- `Die(format, args...)` - Formatted error exit
- `MustAbs(path)` - Absolute path or die
- `AwsString(*string)` - Safely dereference AWS string pointers
- `PrintTable(rows)` - ASCII table output

### 7. Interfaces (`internal/interfaces.go`)

**Responsibilities:**
- Define mockable interfaces for testing
- Enable dependency injection

**Key Interfaces:**
```go
type IAMSimulator interface {
    SimulateCustomPolicy(ctx, params, opts) (*Output, error)
}

type Exiter interface {
    Exit(code int)
}
```

## Data Flow

### Complete Request Flow

```
1. CLI Arguments
   └─> --scenario scenarios/test.yml
       --save /tmp/response.json
       --no-assert

2. Load Scenario
   └─> LoadScenarioWithExtends()
       ├─> Reads test.yml
       ├─> Finds "extends: base.yml"
       ├─> Recursively loads base.yml
       └─> MergeScenario(base, test)

3. Build Variables
   └─> Load vars_file: vars.yml
       └─> Merge with inline vars
           Result: { bucket: "prod", account: "123..." }

4. Render Policy
   └─> PolicyTemplate specified
       ├─> Load template file
       ├─> RenderTemplateFileJSON(template, vars)
       └─> Output: Minified JSON

5. Merge SCPs
   └─> scp_paths: ["scp/*.json"]
       ├─> ExpandGlobsRelative() → [scp/deny-root.json, scp/require-mfa.json]
       ├─> MergeSCPFiles()
       └─> Output: Single policy with merged statements

6. Render Test Data
   └─> Actions: ["s3:GetObject"]
       Resources: ["arn:aws:s3:::{{.bucket}}/*"]
       ├─> RenderStringSlice(actions, vars)
       ├─> RenderStringSlice(resources, vars)
       └─> RenderContext(context, vars)

7. Call AWS API
   └─> Build SimulateCustomPolicyInput
       ├─> PolicyInputList: [identity policy]
       ├─> PermissionsBoundaryPolicyInputList: [merged SCPs]
       ├─> ResourcePolicy: resource policy
       ├─> ActionNames: rendered actions
       ├─> ResourceArns: rendered resources
       └─> ContextEntries: rendered context

8. Evaluate Results
   └─> AWS returns EvaluationResults
       ├─> Extract decision for each action
       ├─> Compare vs expect map
       └─> Print results

9. Output
   └─> Legacy: ASCII table
       Collection: Test results with ✓/✗
       Save: Raw JSON response to file

10. Exit
    └─> Exit 0: All tests passed
        Exit 2: Assertion failures
        Exit 1: Runtime errors
```

## Test Formats

### Legacy Format

**Use Case:** Simple batch testing of multiple actions against resources

```yaml
actions:
  - "s3:GetObject"
  - "s3:PutObject"
resources:
  - "arn:aws:s3:::my-bucket/*"
expect:
  "s3:GetObject": "allowed"
  "s3:PutObject": "explicitDeny"
```

**Output:**
```
Action          Decision        Matched
s3:GetObject    allowed         PolicyId1
s3:PutObject    explicitDeny    PolicyId2
```

### Collection Format

**Use Case:** Complex scenarios with named tests, per-test resources, and custom configurations

```yaml
tests:
  - name: "Read access allowed"
    action: "s3:GetObject"
    resource: "arn:aws:s3:::my-bucket/*"
    expect: "allowed"

  - name: "Write access denied"
    action: "s3:PutObject"
    resource: "arn:aws:s3:::my-bucket/*"
    expect: "explicitDeny"
```

**Output:**
```
Running 2 test(s)...

[1/2] Read access allowed
  ✓ PASS: allowed (matched: PolicyId1)

[2/2] Write access denied
  ✓ PASS: explicitDeny (matched: PolicyId2)

========================================
Test Results: 2 passed, 0 failed
========================================
```

## AWS Integration

### IAM SimulateCustomPolicy API

**API Endpoint:** `iam:SimulateCustomPolicy`

**Input:**
```go
SimulateCustomPolicyInput{
    PolicyInputList: []string{identityPolicy},
    PermissionsBoundaryPolicyInputList: []string{scpMerged},
    ResourcePolicy: &resourcePolicy,
    ActionNames: []string{"s3:GetObject"},
    ResourceArns: []string{"arn:aws:s3:::bucket/*"},
    ContextEntries: []ContextEntry{...},
    CallerArn: &principalArn,
    ResourceOwner: &ownerAccountArn,
    ResourceHandlingOption: &ec2Scenario,
}
```

**Output:**
```go
SimulateCustomPolicyOutput{
    EvaluationResults: []EvaluationResult{
        {
            EvalActionName: "s3:GetObject",
            EvalDecision: "allowed",
            MatchedStatements: []Statement{
                {SourcePolicyId: "PolicyId1"},
            },
        },
    },
}
```

**Decisions:**
- `allowed` - Action is allowed
- `explicitDeny` - Explicitly denied by a Deny statement
- `implicitDeny` - No Allow statement matches (default deny)

### Context Keys

IAM condition context keys with type mapping:

```yaml
context:
  - ContextKeyName: "aws:SourceIp"
    ContextKeyValues: ["203.0.113.0/24"]
    ContextKeyType: "string"

  - ContextKeyName: "s3:x-amz-acl"
    ContextKeyValues: ["private", "public-read"]
    ContextKeyType: "stringList"

  - ContextKeyName: "aws:CurrentTime"
    ContextKeyValues: ["2024-01-01T00:00:00Z"]
    ContextKeyType: "date"
```

**Supported Types:**
- `string`, `stringList`
- `numeric`, `numericList`
- `boolean`, `booleanList`
- `date`, `dateList`

**Not Supported:**
- `ipAddress`, `ipAddressList` (AWS SDK v2 for Go limitation)

## File Structure

```
politest/
├── main.go                 # Entry point, orchestration
├── internal/               # Internal packages
│   ├── scenario.go         # Scenario loading & merging
│   ├── template.go         # Template rendering
│   ├── policy.go           # SCP merging, JSON utilities
│   ├── simulator.go        # AWS API integration, test execution
│   ├── helpers.go          # Utilities, error handling
│   ├── interfaces.go       # Mockable interfaces
│   ├── types.go            # Type definitions
│   └── internal_test.go    # Comprehensive unit tests (95.1% coverage)
├── test/                   # Integration tests
│   ├── scenarios/          # Test scenario files
│   ├── policies/           # Test IAM policies
│   ├── scp/                # Service Control Policies
│   ├── rcp/                # Resource Control Policies
│   └── run-tests.sh        # Integration test runner
├── .github/workflows/      # CI/CD pipelines
└── docs/                   # Documentation
```

## Design Decisions

### Why Single Binary?

- **Simplicity:** No runtime dependencies
- **Portability:** Works on Linux, macOS, Windows
- **Fast:** No interpreter overhead
- **CI-Friendly:** Easy to install in pipelines

### Why YAML for Scenarios?

- **Human-Readable:** Easy to write and review
- **Structured:** Enforces schema via Go structs
- **Version Control Friendly:** Clean diffs
- **Standard:** Widely adopted in IaC tools

### Why Go text/template?

- **Standard Library:** No external dependencies
- **Familiar:** Same syntax as Kubernetes, Helm
- **Type-Safe:** Validated at runtime
- **Powerful:** Supports conditionals, loops, functions

### Why Two Test Formats?

**Legacy Format:**
- Simpler for batch testing
- Less verbose for many actions
- Backwards compatible

**Collection Format:**
- Better for complex scenarios
- Named tests for clarity
- Per-test configuration
- Better CI output

Both formats are supported indefinitely.

### Why Internal Package?

- **Encapsulation:** Implementation details hidden
- **Testing:** Easier to mock interfaces
- **Organization:** Logical separation of concerns
- **Go Convention:** Standard Go project layout

## Performance Characteristics

### Time Complexity

- Scenario loading: O(depth) where depth = inheritance levels
- Template rendering: O(n) where n = template size
- SCP merging: O(m) where m = number of SCP files
- AWS API call: O(1) per test (network latency)
- Legacy format: O(a × r) where a = actions, r = resources
- Collection format: O(t) where t = number of tests

### Memory Usage

- Minimal: Processes one scenario at a time
- No persistent state
- Garbage collected after each run

### AWS API Limits

- **Read-only:** No write capacity limits
- **Rate Limits:** Standard IAM API limits apply
- **No Quotas:** SimulateCustomPolicy has no quota

## Extension Points

### Adding New Features

1. **New Scenario Field:**
   - Add to `Scenario` or `TestCase` struct in `types.go`
   - Update `MergeScenario` logic in `scenario.go`
   - Update simulator input builders in `simulator.go`

2. **New Template Function:**
   - Add to `template.go`
   - Register in template.FuncMap
   - Document in README

3. **New Output Format:**
   - Add flag in `main.go`
   - Add formatter function
   - Call from simulator functions

### Testing Strategy

- **Unit Tests:** Mock AWS API via `IAMSimulator` interface
- **Integration Tests:** Real AWS API calls with credentials
- **Coverage:** 95.1% on internal package
- **CI:** Runs on every commit

## Security Considerations

- **No Credential Storage:** Uses AWS SDK credential chain
- **Read-Only API:** SimulateCustomPolicy doesn't modify state
- **Template Safety:** Go templates are NOT sandboxed (use trusted scenarios only)
- **Secret Scanning:** GitGuardian scans every commit
- **Dependency Scanning:** govulncheck on every push
- **SAST:** Semgrep and SonarCloud analysis

## Future Enhancements

Potential additions (not committed):

- Interactive mode for scenario creation
- Policy diff tool (compare before/after)
- Web UI for visualizing results
- Policy optimization suggestions
- Batch scenario execution
- Plugin system for custom validators
- Support for AWS Organizations policies

---

For implementation details, see [CLAUDE.md](../CLAUDE.md)
For contributing guidelines, see [CONTRIBUTING.md](../CONTRIBUTING.md)
