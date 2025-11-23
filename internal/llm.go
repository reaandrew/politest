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
		progress.SetStatus("Assembling final policy...")
	}

	// Assemble final policy
	policy := assembleFinalPolicy(allStatements)
	return policy, nil
}

func getBatchSystemPrompt(userPrompt string) string {
	basePrompt := `You are an AWS IAM security expert. Generate IAM policy statements for the specified actions.

CRITICAL: Output ONLY a valid JSON array of Statement objects - no markdown, no explanations, no code fences.

IMPORTANT - Efficiently group actions to minimize statement count:
1. Combine ALL actions with the same access level and resource requirements into ONE statement
2. Use wildcards (e.g., "service:Get*", "service:List*", "service:Describe*") where actions share common prefixes
3. Only create separate statements when:
   - Different resource types are required
   - Different conditions are needed (e.g., MFA for write operations)
   - Logical security boundaries exist (read vs write vs admin)
4. Target: Aim for 3-8 statements per batch, NOT one statement per action

Statement requirements:
- Use descriptive Sid values (e.g., "AllowBedrockReadOperations", "AllowBedrockModelInvocation")
- Apply aws:SecureTransport condition for network security
- Use MFA conditions for destructive/sensitive operations (Delete*, Update*, Put*)
- Use specific resource ARNs where the resource type is clear

IMPORTANT - Use these EXACT placeholder variables (not example values like vpce-1a2b3c4d):
- ${AWS::AccountId} - AWS account ID
- ${AWS::Region} - AWS region
- ${VpcEndpointId} - VPC endpoint ID (e.g., for aws:sourceVpce condition)
- ${VpcId} - VPC ID
- ${OrgId} - AWS Organization ID
- ${OrgPath} - AWS Organization path
- ${PrincipalTag/Department} - Principal tag value
- ${ResourceTag/Environment} - Resource tag value

The policy should be suitable for highly regulated environments (UK Government, banks, public sector).

Example of EFFICIENT grouping:
[
  {
    "Sid": "AllowReadAndListOperations",
    "Effect": "Allow",
    "Action": ["service:Get*", "service:List*", "service:Describe*"],
    "Resource": "*",
    "Condition": {"Bool": {"aws:SecureTransport": "true"}}
  },
  {
    "Sid": "AllowWriteOperationsWithMFA",
    "Effect": "Allow",
    "Action": ["service:Create*", "service:Update*", "service:Put*"],
    "Resource": "*",
    "Condition": {
      "Bool": {"aws:SecureTransport": "true", "aws:MultiFactorAuthPresent": "true"}
    }
  }
]`

	if userPrompt != "" {
		basePrompt += "\n\n## Additional Requirements from User:\n" + userPrompt
	}

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

func getSecurityPolicySystemPrompt() string {
	return `You are an AWS IAM security expert specializing in creating secure, compliant IAM policies for highly regulated environments such as UK Government, public sector organizations, financial institutions, and banks.

Your task is to generate a "full access" IAM policy that:
1. Grants comprehensive permissions for the specified AWS service
2. Follows security best practices and least-privilege principles where possible
3. Includes appropriate condition keys for enhanced security
4. Is suitable for production use in highly regulated environments
5. Includes conditions that enforce:
   - Encryption requirements where applicable (e.g., aws:SecureTransport)
   - VPC endpoint restrictions where supported
   - MFA requirements for sensitive operations
   - Tag-based access control where appropriate
   - Request tagging requirements
   - Source IP or VPC restrictions (as placeholders)

Output ONLY valid JSON - no markdown, no explanations, just the IAM policy document.
The policy should be practical and usable, not overly restrictive to the point of being unusable.
Include comments as "Sid" values to explain each statement's purpose.`
}

func buildPolicyGenerationPrompt(data *ScrapedIAMData) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Generate a comprehensive, security-focused IAM policy for the AWS service: %s (prefix: %s)\n\n", data.ServiceName, data.ServicePrefix))

	sb.WriteString("## Available Actions:\n")

	// Group actions by access level for better LLM comprehension
	// and limit to key representative actions if there are too many
	actionsByLevel := make(map[string][]IAMAction)
	for _, action := range data.Actions {
		level := action.AccessLevel
		if level == "" {
			level = "Unknown"
		}
		actionsByLevel[level] = append(actionsByLevel[level], action)
	}

	// Include all actions, grouped by access level
	accessLevelOrder := []string{"Read", "List", "Write", "Permissions management", "Tagging", "Unknown"}
	for _, level := range accessLevelOrder {
		actions, ok := actionsByLevel[level]
		if !ok || len(actions) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("\n### %s Actions (%d):\n", level, len(actions)))
		for _, action := range actions {
			sb.WriteString(fmt.Sprintf("- %s:%s: %s\n", data.ServicePrefix, action.Name, action.Description))
		}
	}

	sb.WriteString("\n## Available Condition Keys:\n")
	for _, key := range data.ConditionKeys {
		sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", key.Name, key.Type, key.Description))
	}

	sb.WriteString("\n## Resource Types:\n")
	for _, resource := range data.ResourceTypes {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", resource.Name, resource.ARN))
	}

	sb.WriteString(`

## Requirements:
1. Create a policy that grants full access to this service for authorized users
2. Group related actions into logical statements with descriptive Sids
3. Apply appropriate conditions for security:
   - Use aws:SecureTransport where network security matters
   - Use service-specific condition keys for fine-grained control
   - Consider MFA requirements for destructive/sensitive operations
   - Include resource-level permissions where possible (avoid * when specific resources can be used)
4. The policy should be suitable for:
   - UK Government (GDS, NCSC guidelines)
   - Financial institutions (PCI-DSS, SOX compliance considerations)
   - Public sector organizations
5. Include placeholder values like ${AWS::AccountId}, ${AWS::Region} for account-specific values
6. Output valid JSON only, no markdown code fences`)

	return sb.String()
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
