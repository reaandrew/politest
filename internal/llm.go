package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// LLMClient provides an interface to OpenAI-compatible LLM APIs
type LLMClient struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

// ChatMessage represents a message in the chat completion format
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents the request body for chat completions
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
}

// ChatCompletionResponse represents the response from chat completions
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// NewLLMClient creates a new LLM client
func NewLLMClient(baseURL, apiKey, model string) *LLMClient {
	// Ensure base URL doesn't have trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &LLMClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Client: &http.Client{
			Timeout: 5 * time.Minute, // LLM calls can take a while for large prompts
		},
	}
}

// batchResult holds the result of processing a single batch
type batchResult struct {
	index      int
	statements []json.RawMessage
	err        error
}

// GenerateSecurityPolicy generates a security-focused IAM policy using the LLM
// It processes actions in batches with parallel execution for speed
func (c *LLMClient) GenerateSecurityPolicy(data *ScrapedIAMData, progress ProgressReporter, userPrompt string, concurrency int) (string, error) {
	const batchSize = 30 // Process 30 actions at a time

	if concurrency <= 0 {
		concurrency = 3
	}

	// Group actions by access level for logical batching
	actionsByLevel := make(map[string][]IAMAction)
	for _, action := range data.Actions {
		level := action.AccessLevel
		if level == "" {
			level = "Unknown"
		}
		actionsByLevel[level] = append(actionsByLevel[level], action)
	}

	// Create batches
	var batches [][]IAMAction
	accessLevelOrder := []string{"Read", "List", "Write", "Permissions management", "Tagging", "Unknown"}
	var currentBatch []IAMAction

	for _, level := range accessLevelOrder {
		actions := actionsByLevel[level]
		for _, action := range actions {
			currentBatch = append(currentBatch, action)
			if len(currentBatch) >= batchSize {
				batches = append(batches, currentBatch)
				currentBatch = nil
			}
		}
	}
	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	if progress != nil {
		progress.SetStatus(fmt.Sprintf("Generating policy in %d batches (concurrency: %d)...", len(batches), concurrency))
	}

	// Process batches in parallel with limited concurrency
	results := make(chan batchResult, len(batches))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	var completedMu sync.Mutex
	completed := 0

	for i, batch := range batches {
		wg.Add(1)
		go func(idx int, b []IAMAction) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			prompt := buildBatchPolicyPrompt(data, b, idx+1, len(batches), userPrompt)

			// Note: Some APIs (like OpenWebUI) don't support system messages properly
			// So we prepend the system instructions to the user message
			fullPrompt := getBatchSystemPrompt(userPrompt) + "\n\n---\n\n" + prompt

			response, err := c.ChatCompletion([]ChatMessage{
				{
					Role:    "user",
					Content: fullPrompt,
				},
			})

			if err != nil {
				results <- batchResult{index: idx, err: fmt.Errorf("batch %d: %w", idx+1, err)}
				return
			}

			// Extract statements from response
			statements := extractStatementsFromResponse(response)

			// Update progress
			completedMu.Lock()
			completed++
			if progress != nil {
				progress.SetStatus(fmt.Sprintf("Completed %d of %d batches...", completed, len(batches)))
				progress.SetProgress(completed, len(batches))
			}
			completedMu.Unlock()

			results <- batchResult{index: idx, statements: statements}
		}(i, batch)
	}

	// Wait for all batches to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results in order
	batchResults := make([]batchResult, len(batches))
	for result := range results {
		if result.err != nil {
			return "", result.err
		}
		batchResults[result.index] = result
	}

	// Sort by index and assemble statements
	sort.Slice(batchResults, func(i, j int) bool {
		return batchResults[i].index < batchResults[j].index
	})

	var allStatements []json.RawMessage
	for _, br := range batchResults {
		allStatements = append(allStatements, br.statements...)
	}

	if progress != nil {
		progress.SetProgress(len(batches), len(batches))
		progress.SetStatus("Consolidating policy statements...")
	}

	// Consolidate: dedupe exact matches, then LLM-merge similar statements
	consolidated, err := c.ConsolidatePolicy(allStatements, progress, concurrency)
	if err != nil {
		// Non-fatal: fall back to unconsolidated statements
		consolidated = allStatements
	}

	if progress != nil {
		progress.SetStatus("Assembling final policy...")
	}

	// Assemble final policy
	policy := assembleFinalPolicy(consolidated)
	return policy, nil
}

func getBatchSystemPrompt(userPrompt string) string {
	basePrompt := `You are an AWS IAM security expert generating IAM policy statements.

CRITICAL: Output ONLY a valid JSON array of Statement objects - no markdown, no explanations, no code fences.
If no statements should be generated for this batch, output an empty array: []`

	// User requirements are PRIMARY - they override default behavior
	if userPrompt != "" {
		basePrompt += `

## USER REQUIREMENTS (PRIMARY - follow these first):
` + userPrompt + `

IMPORTANT: The user requirements above take precedence. If the user specifies:
- Which actions to include → ONLY generate statements for those actions, skip everything else
- Which actions to exclude → Do NOT generate statements for those actions
- Specific resources or ARNs → Use those exact resources, not wildcards
- To rely on implicit deny → Do NOT generate Allow statements for actions the user wants denied
- SCP or "SCP will handle" → Do NOT include SecureTransport conditions or deny statements - those go in the SCP

If none of the actions in this batch match what the user wants, return an empty array [].`
	}

	basePrompt += `

## POLICY GENERATION GUIDELINES:

Efficiently group actions to minimize statement count:
1. Combine actions with the same access level and resource requirements into ONE statement
2. Use wildcards (e.g., "service:Get*", "service:List*", "service:Describe*") where actions share common prefixes
3. Only create separate statements when:
   - Different resource types are required
   - Different conditions are needed
   - Logical security boundaries exist (read vs write vs admin)

Statement requirements:
- Use descriptive Sid values (e.g., "AllowS3ReadOperations", "AllowBedrockModelInvocation")
- Apply aws:SecureTransport condition for network security
- Use MFA conditions for destructive/sensitive operations (Delete*, Update*, Put*) unless user says otherwise
- Use specific resource ARNs where the resource type is clear

Use these placeholder variables where appropriate:
- ${AWS::AccountId} - AWS account ID
- ${AWS::Region} - AWS region
- ${VpcEndpointId} - VPC endpoint ID
- ${OrgId} - AWS Organization ID

The policy should be suitable for regulated environments.`

	return basePrompt
}

func buildBatchPolicyPrompt(data *ScrapedIAMData, batch []IAMAction, batchNum, totalBatches int, userPrompt string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Generate IAM policy statements for AWS service: %s (prefix: %s)\n", data.ServiceName, data.ServicePrefix))
	sb.WriteString(fmt.Sprintf("Batch %d of %d\n\n", batchNum, totalBatches))

	// Group actions by access level to help LLM understand grouping
	actionsByLevel := make(map[string][]IAMAction)
	for _, action := range batch {
		level := action.AccessLevel
		if level == "" {
			level = "Unknown"
		}
		actionsByLevel[level] = append(actionsByLevel[level], action)
	}

	sb.WriteString("## Actions to include (grouped by access level):\n")
	for level, actions := range actionsByLevel {
		sb.WriteString(fmt.Sprintf("\n### %s Actions (%d):\n", level, len(actions)))
		for _, action := range actions {
			sb.WriteString(fmt.Sprintf("- %s:%s: %s\n", data.ServicePrefix, action.Name, action.Description))
		}
	}

	if len(data.ConditionKeys) > 0 {
		sb.WriteString("\n## Available Condition Keys:\n")
		for i, key := range data.ConditionKeys {
			if i >= 10 {
				sb.WriteString(fmt.Sprintf("... and %d more\n", len(data.ConditionKeys)-10))
				break
			}
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", key.Name, key.Type))
		}
	}

	sb.WriteString("\nREMEMBER: Group actions efficiently! Use wildcards and combine similar actions. Output ONLY a JSON array.")

	return sb.String()
}

func extractStatementsFromResponse(response string) []json.RawMessage {
	response = strings.TrimSpace(response)

	// Remove markdown code fences if present
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	// Try to parse as array of statements
	var statements []json.RawMessage
	if err := json.Unmarshal([]byte(response), &statements); err == nil {
		return statements
	}

	// Try to find array in response
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")
	if start >= 0 && end > start {
		candidate := response[start : end+1]
		if err := json.Unmarshal([]byte(candidate), &statements); err == nil {
			return statements
		}
	}

	return nil
}

func assembleFinalPolicy(statements []json.RawMessage) string {
	policy := map[string]any{
		"Version":   "2012-10-17",
		"Statement": statements,
	}

	result, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return ""
	}
	return string(result)
}

// dedupeStatements removes exact duplicate statements using hash comparison
func dedupeStatements(statements []json.RawMessage) []json.RawMessage {
	seen := make(map[string]bool)
	var result []json.RawMessage

	for _, stmt := range statements {
		// Normalize JSON by unmarshaling and remarshaling with sorted keys
		var parsed map[string]any
		if err := json.Unmarshal(stmt, &parsed); err != nil {
			// Keep unparseable statements as-is
			result = append(result, stmt)
			continue
		}

		// Remarshal to get consistent key ordering
		normalized, err := json.Marshal(parsed)
		if err != nil {
			result = append(result, stmt)
			continue
		}

		key := string(normalized)
		if !seen[key] {
			seen[key] = true
			result = append(result, stmt)
		}
	}

	return result
}

// groupStatementsBySid groups statements by their Sid value
func groupStatementsBySid(statements []json.RawMessage) map[string][]json.RawMessage {
	groups := make(map[string][]json.RawMessage)

	for _, stmt := range statements {
		var parsed map[string]any
		if err := json.Unmarshal(stmt, &parsed); err != nil {
			// Put unparseable statements in a special group
			groups["_unparseable"] = append(groups["_unparseable"], stmt)
			continue
		}

		sid, ok := parsed["Sid"].(string)
		if !ok || sid == "" {
			sid = "_no_sid"
		}

		groups[sid] = append(groups[sid], stmt)
	}

	return groups
}

// ConsolidateStatementGroup uses LLM to merge similar statements into one
func (c *LLMClient) ConsolidateStatementGroup(sid string, statements []json.RawMessage) (json.RawMessage, error) {
	// Build a compact representation of the statements
	var stmtStrings []string
	for _, stmt := range statements {
		stmtStrings = append(stmtStrings, string(stmt))
	}

	prompt := fmt.Sprintf(`Merge these %d IAM policy statements with Sid "%s" into a SINGLE optimized statement.

Statements to merge:
%s

Rules:
1. Combine all Actions into one array (remove duplicates)
2. If Resources differ, use the most permissive (prefer "*" if any use it)
3. Merge Conditions intelligently (combine values for same condition keys)
4. Keep the same Sid name
5. Output ONLY the single merged JSON statement object - no markdown, no explanation

Merged statement:`, len(statements), sid, strings.Join(stmtStrings, "\n"))

	response, err := c.ChatCompletion([]ChatMessage{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, err
	}

	// Clean up response
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	// Find JSON object
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start >= 0 && end > start {
		response = response[start : end+1]
	}

	// Validate it's valid JSON
	var validated json.RawMessage
	if err := json.Unmarshal([]byte(response), &validated); err != nil {
		return nil, fmt.Errorf("invalid JSON from LLM: %w", err)
	}

	return validated, nil
}

// ConsolidatePolicy deduplicates and consolidates policy statements
// Step 1: Remove exact duplicates (programmatic)
// Step 2: Group by Sid
// Step 3: Use LLM to merge groups with multiple statements
func (c *LLMClient) ConsolidatePolicy(statements []json.RawMessage, progress ProgressReporter, concurrency int) ([]json.RawMessage, error) {
	if progress != nil {
		progress.SetStatus("Consolidating policy statements...")
	}

	// Step 1: Exact dedup
	deduped := dedupeStatements(statements)
	if progress != nil {
		progress.SetStatus(fmt.Sprintf("Removed %d exact duplicates (%d → %d statements)",
			len(statements)-len(deduped), len(statements), len(deduped)))
	}

	// Step 2: Group by Sid
	groups := groupStatementsBySid(deduped)

	// Count groups needing consolidation
	var groupsToMerge []string
	for sid, stmts := range groups {
		if len(stmts) > 1 && sid != "_unparseable" && sid != "_no_sid" {
			groupsToMerge = append(groupsToMerge, sid)
		}
	}

	if len(groupsToMerge) == 0 {
		// No merging needed, return deduped statements
		var result []json.RawMessage
		for _, stmts := range groups {
			result = append(result, stmts...)
		}
		return result, nil
	}

	if progress != nil {
		progress.SetStatus(fmt.Sprintf("Merging %d statement groups with LLM...", len(groupsToMerge)))
	}

	// Step 3: Merge groups in parallel
	if concurrency <= 0 {
		concurrency = 3
	}

	type mergeResult struct {
		sid  string
		stmt json.RawMessage
		err  error
	}

	results := make(chan mergeResult, len(groupsToMerge))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, sid := range groupsToMerge {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			merged, err := c.ConsolidateStatementGroup(s, groups[s])
			results <- mergeResult{sid: s, stmt: merged, err: err}
		}(sid)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	mergedGroups := make(map[string]json.RawMessage)
	for result := range results {
		if result.err != nil {
			// On error, keep original statements for this group
			continue
		}
		mergedGroups[result.sid] = result.stmt
	}

	// Assemble final statement list
	var finalStatements []json.RawMessage
	for sid, stmts := range groups {
		if merged, ok := mergedGroups[sid]; ok {
			// Use merged statement
			finalStatements = append(finalStatements, merged)
		} else {
			// Use original statements (single stmt groups, or merge failed)
			finalStatements = append(finalStatements, stmts...)
		}
	}

	if progress != nil {
		progress.SetStatus(fmt.Sprintf("Consolidated to %d statements", len(finalStatements)))
	}

	return finalStatements, nil
}

// EnrichActionDescriptions uses the LLM to add security context to actions
func (c *LLMClient) EnrichActionDescriptions(data *ScrapedIAMData, progress ProgressReporter) error {
	if progress != nil {
		progress.SetStatus("Enriching action descriptions with security context...")
	}

	// Build a prompt asking for security considerations for each action
	prompt := buildEnrichmentPrompt(data)

	// Merge system instructions into user message (some APIs don't support system role)
	fullPrompt := "You are an AWS security expert. Provide concise security considerations for IAM actions. Respond in JSON format.\n\n---\n\n" + prompt

	response, err := c.ChatCompletion([]ChatMessage{
		{
			Role:    "user",
			Content: fullPrompt,
		},
	})
	if err != nil {
		// Non-fatal - we can continue without enrichment
		return nil
	}

	// Parse and merge enrichment data
	parseEnrichmentResponse(response, data)
	return nil
}

// GeneratePolicyDocumentation generates a Markdown documentation file for the policy
func (c *LLMClient) GeneratePolicyDocumentation(data *ScrapedIAMData, policyJSON string) (string, error) {
	prompt := fmt.Sprintf(`Generate comprehensive Markdown documentation for the following IAM policy.

## Service Information
- Service Name: %s
- Service Prefix: %s
- Total Actions Available: %d

## The Policy to Document
%s

## Documentation Requirements

Create a well-structured Markdown document that includes:

1. **Overview Section**: Brief description of what this policy provides access to

2. **Statement Documentation**: For EACH statement in the policy:
   - **Statement ID (Sid)**: The identifier
   - **Purpose**: Clear explanation of what this statement allows
   - **Actions Covered**: List the actions and briefly explain what they do
   - **Resource Scope**: Explain what resources are affected
   - **Conditions**: Explain any conditions and their security implications
   - **Security Notes**: Any security considerations for this statement

3. **Variables to Configure**: List ALL placeholder variables found in the policy (like ${AWS::AccountId}, ${VpcEndpointId}, etc.) with:
   - Variable name
   - Description of what value to substitute
   - Example value format
   - Where to find/determine the correct value

4. **Security Summary**:
   - Key security controls in place
   - Recommendations for further hardening
   - Compliance considerations (mention relevance to UK Gov, financial sector)

5. **Usage Notes**:
   - When to use this policy
   - What roles/users it's suitable for
   - Any prerequisites

Output ONLY the Markdown content, no code fences around it.`, data.ServiceName, data.ServicePrefix, len(data.Actions), policyJSON)

	response, err := c.ChatCompletion([]ChatMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate documentation: %w", err)
	}

	// Clean up response - remove any markdown code fences if present
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```markdown") {
		response = strings.TrimPrefix(response, "```markdown")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```md") {
		response = strings.TrimPrefix(response, "```md")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	return response, nil
}

// GenerateSCP generates a Service Control Policy based on the identity policy and user requirements
func (c *LLMClient) GenerateSCP(data *ScrapedIAMData, identityPolicyJSON string, userPrompt string) (string, error) {
	prompt := fmt.Sprintf(`Generate a Service Control Policy (SCP) to complement the following identity policy.

## Service Information
- Service Name: %s
- Service Prefix: %s

## Identity Policy (already created)
%s

## User Requirements
%s

## SCP Generation Guidelines

Use the "deny all except allowlist" pattern with NotAction. This is the standard enterprise SCP pattern.

CRITICAL STRUCTURE - Use NotAction to deny everything EXCEPT allowed actions:

{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DenyAllExceptAllowedActions",
      "Effect": "Deny",
      "NotAction": [
        "service:AllowedAction1",
        "service:AllowedAction2",
        "service:Get*",
        "service:List*",
        "service:Describe*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "DenyInsecureTransport",
      "Effect": "Deny",
      "Action": "service:*",
      "Resource": "*",
      "Condition": {
        "Bool": {
          "aws:SecureTransport": "false"
        }
      }
    },
    {
      "Sid": "DenyOutsideAllowedRegions",
      "Effect": "Deny",
      "Action": "service:*",
      "Resource": "*",
      "Condition": {
        "StringNotEqualsIfExists": {
          "aws:RequestedRegion": ["eu-west-2"]
        }
      }
    },
    {
      "Sid": "DenyGlobalCrossRegionInference",
      "Effect": "Deny",
      "Action": ["service:InvokeModel", "service:InvokeModelWithResponseStream"],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "aws:RequestedRegion": "unspecified"
        }
      }
    }
  ]
}

Guidelines:
1. Extract the ALLOWED actions from the identity policy and put them in NotAction
2. Include Get*, List*, Describe* wildcards in NotAction for read operations
3. Add DenyInsecureTransport statement (deny when aws:SecureTransport = false)
4. Add region restriction statement using StringNotEqualsIfExists
5. Add global CRIS denial for invoke actions (aws:RequestedRegion = "unspecified")
6. Keep statements minimal - the NotAction pattern handles most denials in one statement

Output ONLY valid JSON for an SCP policy document. No markdown, no explanations.`, data.ServiceName, data.ServicePrefix, identityPolicyJSON, userPrompt)

	response, err := c.ChatCompletion([]ChatMessage{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate SCP: %w", err)
	}

	// Clean up response
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	// Find JSON object
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start >= 0 && end > start {
		response = response[start : end+1]
	}

	// Validate JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(response), &js); err != nil {
		return "", fmt.Errorf("invalid SCP JSON: %w", err)
	}

	return response, nil
}

// GenerateCombinedDocumentation generates documentation covering both identity policy and SCP
func (c *LLMClient) GenerateCombinedDocumentation(data *ScrapedIAMData, identityPolicyJSON, scpJSON string) (string, error) {
	prompt := fmt.Sprintf(`Generate comprehensive Markdown documentation for the following IAM Identity Policy and Service Control Policy (SCP).

## Service Information
- Service Name: %s
- Service Prefix: %s
- Total Actions Available: %d

## Identity Policy
%s

## Service Control Policy (SCP)
%s

## Documentation Requirements

Create a well-structured Markdown document that includes:

1. **Overview Section**:
   - Brief description of the overall access control strategy
   - Explain the separation between identity policy (what users CAN do) and SCP (org-wide guardrails)

2. **Identity Policy Documentation**:
   - Purpose: What this policy allows
   - Statement-by-statement breakdown
   - Actions, resources, and conditions explained
   - Who should have this policy attached

3. **SCP Documentation**:
   - Purpose: What org-wide guardrails this provides
   - Statement-by-statement breakdown
   - Why these denies are at the SCP level (cannot be bypassed)
   - Which OUs/accounts this should be applied to

4. **Deployment Guide**:
   - Step 1: Deploy SCP to AWS Organizations (specify OU or account targets)
   - Step 2: Attach identity policy to IAM roles/users
   - Testing recommendations
   - Rollback procedures

5. **Variables to Configure**:
   - List ALL placeholder variables from BOTH policies
   - Description, example values, and where to find them

6. **Security Summary**:
   - Defense in depth explanation (identity + SCP layers)
   - Key security controls in place
   - Compliance considerations (UK Gov, financial sector, regulated environments)

7. **Troubleshooting**:
   - Common access denied scenarios
   - How to diagnose SCP vs identity policy denials
   - CloudTrail event patterns to look for

Output ONLY the Markdown content, no code fences around it.`, data.ServiceName, data.ServicePrefix, len(data.Actions), identityPolicyJSON, scpJSON)

	response, err := c.ChatCompletion([]ChatMessage{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate combined documentation: %w", err)
	}

	// Clean up response
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```markdown") {
		response = strings.TrimPrefix(response, "```markdown")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```md") {
		response = strings.TrimPrefix(response, "```md")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	return response, nil
}

// ChatCompletion sends a chat completion request and returns the response content
// It includes retry logic for transient network failures
func (c *LLMClient) ChatCompletion(messages []ChatMessage) (string, error) {
	reqBody := ChatCompletionRequest{
		Model:       c.Model,
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   4096,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.BaseURL + "/v1/chat/completions"

	// Retry logic for transient failures
	var lastErr error
	maxAttempts := 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			backoff := time.Duration(attempt*attempt) * time.Second // Exponential backoff: 4s, 9s, 16s, 25s
			time.Sleep(backoff)
		}

		result, err := c.doRequest(url, jsonBody)
		if err == nil {
			return result, nil
		}
		lastErr = err

		// Retry on network errors and server errors (5xx)
		errStr := err.Error()
		isRetryable := strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "EOF") ||
			strings.Contains(errStr, "HTTP 500") ||
			strings.Contains(errStr, "HTTP 502") ||
			strings.Contains(errStr, "HTTP 503") ||
			strings.Contains(errStr, "HTTP 504")

		if isRetryable && attempt < maxAttempts {
			continue
		}
		return "", err
	}
	return "", fmt.Errorf("after %d attempts: %w", maxAttempts, lastErr)
}

// doRequest performs a single HTTP request
func (c *LLMClient) doRequest(url string, jsonBody []byte) (string, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, bodyStr)
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w (body: %s)", err, string(body))
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		// Check if there's a different response structure (some APIs use different formats)
		var altResp map[string]any
		if err := json.Unmarshal(body, &altResp); err == nil {
			// Check for OpenWebUI specific response formats
			if msg, ok := altResp["message"].(map[string]any); ok {
				if content, ok := msg["content"].(string); ok {
					return content, nil
				}
			}
			// Check for error field
			if errMsg, ok := altResp["error"].(string); ok {
				return "", fmt.Errorf("API error: %s", errMsg)
			}
			if detail, ok := altResp["detail"].(string); ok {
				return "", fmt.Errorf("API error: %s", detail)
			}
		}
		// Log truncated response for debugging
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		return "", fmt.Errorf("no choices in response (body: %s)", bodyStr)
	}

	return chatResp.Choices[0].Message.Content, nil
}

func buildEnrichmentPrompt(data *ScrapedIAMData) string {
	var sb strings.Builder

	sb.WriteString("For each of the following IAM actions, provide a brief security risk level (Low/Medium/High/Critical) and a one-line security consideration. Respond as JSON with format: {\"actions\": [{\"name\": \"ActionName\", \"risk\": \"Level\", \"security_note\": \"Note\"}]}\n\n")

	sb.WriteString(fmt.Sprintf("Service: %s\n\n", data.ServicePrefix))

	for _, action := range data.Actions {
		sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", action.Name, action.AccessLevel, action.Description))
	}

	return sb.String()
}

func extractJSONFromResponse(response string) string {
	// Try to extract JSON from the response
	response = strings.TrimSpace(response)

	// Remove markdown code fences if present
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	// Validate it's proper JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(response), &js); err != nil {
		// Try to find JSON object in the response
		start := strings.Index(response, "{")
		end := strings.LastIndex(response, "}")
		if start >= 0 && end > start {
			candidate := response[start : end+1]
			if err := json.Unmarshal([]byte(candidate), &js); err == nil {
				return candidate
			}
		}
		return ""
	}

	return response
}

func parseEnrichmentResponse(response string, data *ScrapedIAMData) {
	// Try to parse the enrichment response and update actions
	// This is best-effort - we don't fail if it doesn't work
	type EnrichmentData struct {
		Actions []struct {
			Name         string `json:"name"`
			Risk         string `json:"risk"`
			SecurityNote string `json:"security_note"`
		} `json:"actions"`
	}

	jsonStr := extractJSONFromResponse(response)
	if jsonStr == "" {
		return
	}

	var enrichment EnrichmentData
	if err := json.Unmarshal([]byte(jsonStr), &enrichment); err != nil {
		return
	}

	// Build lookup map
	enrichmentMap := make(map[string]struct {
		Risk         string
		SecurityNote string
	})
	for _, e := range enrichment.Actions {
		enrichmentMap[e.Name] = struct {
			Risk         string
			SecurityNote string
		}{e.Risk, e.SecurityNote}
	}

	// Note: In a full implementation, we'd add fields to IAMAction for this
	// For now, we could append to the description or add new fields
	_ = enrichmentMap
}
