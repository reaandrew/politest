package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewLLMClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		apiKey  string
		model   string
		wantURL string
	}{
		{
			name:    "basic creation",
			baseURL: "https://api.example.com",
			apiKey:  "test-key",
			model:   "gpt-4",
			wantURL: "https://api.example.com",
		},
		{
			name:    "strips trailing slash",
			baseURL: "https://api.example.com/",
			apiKey:  "test-key",
			model:   "gpt-4",
			wantURL: "https://api.example.com",
		},
		{
			name:    "empty api key",
			baseURL: "https://api.example.com",
			apiKey:  "",
			model:   "gpt-4",
			wantURL: "https://api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewLLMClient(tt.baseURL, tt.apiKey, tt.model)
			if client.BaseURL != tt.wantURL {
				t.Errorf("BaseURL = %v, want %v", client.BaseURL, tt.wantURL)
			}
			if client.APIKey != tt.apiKey {
				t.Errorf("APIKey = %v, want %v", client.APIKey, tt.apiKey)
			}
			if client.Model != tt.model {
				t.Errorf("Model = %v, want %v", client.Model, tt.model)
			}
			if client.Client == nil {
				t.Error("HTTP client is nil")
			}
		})
	}
}

func TestExtractStatementsFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantLen  int
	}{
		{
			name:     "plain JSON array",
			response: `[{"Sid": "Test", "Effect": "Allow"}]`,
			wantLen:  1,
		},
		{
			name:     "JSON array with multiple statements",
			response: `[{"Sid": "Test1"}, {"Sid": "Test2"}, {"Sid": "Test3"}]`,
			wantLen:  3,
		},
		{
			name:     "markdown code fence json",
			response: "```json\n[{\"Sid\": \"Test\"}]\n```",
			wantLen:  1,
		},
		{
			name:     "markdown code fence without json tag",
			response: "```\n[{\"Sid\": \"Test\"}]\n```",
			wantLen:  1,
		},
		{
			name:     "JSON array with surrounding text",
			response: "Here is your policy:\n[{\"Sid\": \"Test\"}]\nEnd of policy.",
			wantLen:  1,
		},
		{
			name:     "empty array",
			response: "[]",
			wantLen:  0,
		},
		{
			name:     "invalid JSON",
			response: "not valid json",
			wantLen:  0,
		},
		{
			name:     "whitespace around JSON",
			response: "  \n  [{\"Sid\": \"Test\"}]  \n  ",
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractStatementsFromResponse(tt.response)
			if len(result) != tt.wantLen {
				t.Errorf("extractStatementsFromResponse() returned %d statements, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestAssembleFinalPolicy(t *testing.T) {
	tests := []struct {
		name       string
		statements []json.RawMessage
		wantFields []string
	}{
		{
			name:       "single statement",
			statements: []json.RawMessage{json.RawMessage(`{"Sid": "Test", "Effect": "Allow"}`)},
			wantFields: []string{"Version", "Statement"},
		},
		{
			name:       "multiple statements",
			statements: []json.RawMessage{json.RawMessage(`{"Sid": "Test1"}`), json.RawMessage(`{"Sid": "Test2"}`)},
			wantFields: []string{"Version", "Statement"},
		},
		{
			name:       "empty statements",
			statements: []json.RawMessage{},
			wantFields: []string{"Version", "Statement"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleFinalPolicy(tt.statements)
			if result == "" {
				t.Error("assembleFinalPolicy() returned empty string")
				return
			}

			var policy map[string]any
			if err := json.Unmarshal([]byte(result), &policy); err != nil {
				t.Errorf("assembleFinalPolicy() returned invalid JSON: %v", err)
				return
			}

			for _, field := range tt.wantFields {
				if _, ok := policy[field]; !ok {
					t.Errorf("assembleFinalPolicy() missing field %s", field)
				}
			}

			if policy["Version"] != "2012-10-17" {
				t.Errorf("assembleFinalPolicy() Version = %v, want 2012-10-17", policy["Version"])
			}
		})
	}
}

func TestDedupeStatements(t *testing.T) {
	tests := []struct {
		name       string
		statements []json.RawMessage
		wantLen    int
	}{
		{
			name: "no duplicates",
			statements: []json.RawMessage{
				json.RawMessage(`{"Sid": "Test1"}`),
				json.RawMessage(`{"Sid": "Test2"}`),
			},
			wantLen: 2,
		},
		{
			name: "exact duplicates",
			statements: []json.RawMessage{
				json.RawMessage(`{"Sid": "Test1"}`),
				json.RawMessage(`{"Sid": "Test1"}`),
			},
			wantLen: 1,
		},
		{
			name: "duplicates with different whitespace",
			statements: []json.RawMessage{
				json.RawMessage(`{"Sid":"Test1"}`),
				json.RawMessage(`{ "Sid" : "Test1" }`),
			},
			wantLen: 1,
		},
		{
			name: "mix of duplicates and unique",
			statements: []json.RawMessage{
				json.RawMessage(`{"Sid": "Test1"}`),
				json.RawMessage(`{"Sid": "Test2"}`),
				json.RawMessage(`{"Sid": "Test1"}`),
				json.RawMessage(`{"Sid": "Test3"}`),
			},
			wantLen: 3,
		},
		{
			name:       "empty input",
			statements: []json.RawMessage{},
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dedupeStatements(tt.statements)
			if len(result) != tt.wantLen {
				t.Errorf("dedupeStatements() returned %d statements, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestGroupStatementsBySid(t *testing.T) {
	tests := []struct {
		name       string
		statements []json.RawMessage
		wantGroups map[string]int
	}{
		{
			name: "single statement per sid",
			statements: []json.RawMessage{
				json.RawMessage(`{"Sid": "Test1"}`),
				json.RawMessage(`{"Sid": "Test2"}`),
			},
			wantGroups: map[string]int{"Test1": 1, "Test2": 1},
		},
		{
			name: "multiple statements per sid",
			statements: []json.RawMessage{
				json.RawMessage(`{"Sid": "Test1", "Effect": "Allow"}`),
				json.RawMessage(`{"Sid": "Test1", "Effect": "Deny"}`),
				json.RawMessage(`{"Sid": "Test2"}`),
			},
			wantGroups: map[string]int{"Test1": 2, "Test2": 1},
		},
		{
			name: "statements without sid",
			statements: []json.RawMessage{
				json.RawMessage(`{"Effect": "Allow"}`),
				json.RawMessage(`{"Effect": "Deny"}`),
			},
			wantGroups: map[string]int{"_no_sid": 2},
		},
		{
			name:       "empty input",
			statements: []json.RawMessage{},
			wantGroups: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := groupStatementsBySid(tt.statements)
			for sid, count := range tt.wantGroups {
				if len(result[sid]) != count {
					t.Errorf("groupStatementsBySid() sid %s has %d statements, want %d", sid, len(result[sid]), count)
				}
			}
		})
	}
}

func TestGetBatchSystemPrompt(t *testing.T) {
	tests := []struct {
		name         string
		userPrompt   string
		wantContains []string
	}{
		{
			name:       "empty user prompt",
			userPrompt: "",
			wantContains: []string{
				"AWS IAM security expert",
				"CRITICAL: Output ONLY a valid JSON array",
			},
		},
		{
			name:       "with user prompt",
			userPrompt: "Only include read actions",
			wantContains: []string{
				"AWS IAM security expert",
				"USER REQUIREMENTS (PRIMARY",
				"Only include read actions",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBatchSystemPrompt(tt.userPrompt)
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("getBatchSystemPrompt() missing expected content: %s", want)
				}
			}
		})
	}
}

func TestBuildBatchPolicyPrompt(t *testing.T) {
	data := &ScrapedIAMData{
		ServiceName:   "Amazon S3",
		ServicePrefix: "s3",
		ConditionKeys: []IAMConditionKey{
			{Name: "s3:prefix", Type: "String"},
		},
	}

	batch := []IAMAction{
		{Name: "GetObject", Description: "Get an object", AccessLevel: "Read"},
		{Name: "PutObject", Description: "Put an object", AccessLevel: "Write"},
	}

	result := buildBatchPolicyPrompt(data, batch, 1, 3, "test prompt")

	expectedContains := []string{
		"s3",
		"Batch 1 of 3",
		"GetObject",
		"PutObject",
		"Read Actions",
		"Write Actions",
		"s3:prefix",
	}

	for _, want := range expectedContains {
		if !strings.Contains(result, want) {
			t.Errorf("buildBatchPolicyPrompt() missing expected content: %s", want)
		}
	}
}

func TestBuildEnrichmentPrompt(t *testing.T) {
	data := &ScrapedIAMData{
		ServicePrefix: "s3",
		Actions: []IAMAction{
			{Name: "GetObject", AccessLevel: "Read", Description: "Get an object"},
			{Name: "DeleteObject", AccessLevel: "Write", Description: "Delete an object"},
		},
	}

	result := buildEnrichmentPrompt(data)

	expectedContains := []string{
		"s3",
		"GetObject",
		"DeleteObject",
		"security risk level",
		"JSON",
	}

	for _, want := range expectedContains {
		if !strings.Contains(result, want) {
			t.Errorf("buildEnrichmentPrompt() missing expected content: %s", want)
		}
	}
}

func TestExtractJSONFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantJSON bool
	}{
		{
			name:     "plain JSON object",
			response: `{"actions": []}`,
			wantJSON: true,
		},
		{
			name:     "JSON with markdown fence",
			response: "```json\n{\"actions\": []}\n```",
			wantJSON: true,
		},
		{
			name:     "JSON with plain fence",
			response: "```\n{\"actions\": []}\n```",
			wantJSON: true,
		},
		{
			name:     "JSON embedded in text",
			response: "Here is the result: {\"actions\": []} end.",
			wantJSON: true,
		},
		{
			name:     "invalid JSON",
			response: "not valid json",
			wantJSON: false,
		},
		{
			name:     "empty string",
			response: "",
			wantJSON: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromResponse(tt.response)
			if tt.wantJSON && result == "" {
				t.Error("extractJSONFromResponse() returned empty string, expected JSON")
			}
			if !tt.wantJSON && result != "" {
				t.Errorf("extractJSONFromResponse() = %v, expected empty string", result)
			}
		})
	}
}

func TestParseEnrichmentResponse(t *testing.T) {
	data := &ScrapedIAMData{
		ServicePrefix: "s3",
		Actions: []IAMAction{
			{Name: "GetObject", AccessLevel: "Read"},
			{Name: "DeleteObject", AccessLevel: "Write"},
		},
	}

	response := `{"actions": [
		{"name": "GetObject", "risk": "Low", "security_note": "Read only"},
		{"name": "DeleteObject", "risk": "High", "security_note": "Destructive action"}
	]}`

	// Should not panic
	parseEnrichmentResponse(response, data)

	// Test with invalid JSON
	parseEnrichmentResponse("invalid", data)

	// Test with empty response
	parseEnrichmentResponse("", data)
}

func TestChatCompletion(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		wantErr        bool
		wantContent    string
	}{
		{
			name: "successful response",
			serverResponse: `{
				"choices": [{
					"message": {"role": "assistant", "content": "Hello world"}
				}]
			}`,
			serverStatus: http.StatusOK,
			wantErr:      false,
			wantContent:  "Hello world",
		},
		{
			name:           "server error",
			serverResponse: `{"error": {"message": "Server error"}}`,
			serverStatus:   http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name: "api error in response",
			serverResponse: `{
				"error": {"message": "Invalid API key", "type": "auth_error"}
			}`,
			serverStatus: http.StatusOK,
			wantErr:      true,
		},
		{
			name:           "empty choices",
			serverResponse: `{"choices": []}`,
			serverStatus:   http.StatusOK,
			wantErr:        true,
		},
		{
			name: "alternative response format (OpenWebUI)",
			serverResponse: `{
				"message": {"content": "Alt response"}
			}`,
			serverStatus: http.StatusOK,
			wantErr:      false,
			wantContent:  "Alt response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type: application/json")
				}
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("expected Authorization header")
				}

				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := NewLLMClient(server.URL, "test-key", "test-model")
			result, err := client.ChatCompletion([]ChatMessage{
				{Role: "user", Content: "Hello"},
			})

			if tt.wantErr && err == nil {
				t.Error("ChatCompletion() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ChatCompletion() unexpected error: %v", err)
			}
			if !tt.wantErr && result != tt.wantContent {
				t.Errorf("ChatCompletion() = %v, want %v", result, tt.wantContent)
			}
		})
	}
}

func TestChatCompletionRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// First two attempts fail with 500
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "temporary error"}`))
			return
		}
		// Third attempt succeeds
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "success"}}]}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "test-model")
	result, err := client.ChatCompletion([]ChatMessage{
		{Role: "user", Content: "Hello"},
	})

	if err != nil {
		t.Errorf("ChatCompletion() unexpected error after retries: %v", err)
	}
	if result != "success" {
		t.Errorf("ChatCompletion() = %v, want success", result)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestDoRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "test"}}]}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "test-model")
	result, err := client.doRequest(server.URL+"/v1/chat/completions", []byte(`{}`))

	if err != nil {
		t.Errorf("doRequest() unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("doRequest() = %v, want test", result)
	}
}

func TestDoRequestNoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not have Authorization header
		if r.Header.Get("Authorization") != "" {
			t.Error("expected no Authorization header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "test"}}]}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "", "test-model")
	_, err := client.doRequest(server.URL+"/v1/chat/completions", []byte(`{}`))

	if err != nil {
		t.Errorf("doRequest() unexpected error: %v", err)
	}
}

func TestGenerateSecurityPolicy(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		// Return a simple policy statement for each batch
		w.Write([]byte(`{"choices": [{"message": {"content": "[{\"Sid\": \"Test\", \"Effect\": \"Allow\", \"Action\": [\"s3:GetObject\"], \"Resource\": \"*\"}]"}}]}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "test-model")

	data := &ScrapedIAMData{
		ServiceName:   "Amazon S3",
		ServicePrefix: "s3",
		Actions: []IAMAction{
			{Name: "GetObject", AccessLevel: "Read", Description: "Get object"},
		},
	}

	result, err := client.GenerateSecurityPolicy(data, nil, "", 1)

	if err != nil {
		t.Errorf("GenerateSecurityPolicy() unexpected error: %v", err)
	}
	if result == "" {
		t.Error("GenerateSecurityPolicy() returned empty string")
	}

	var policy map[string]any
	if err := json.Unmarshal([]byte(result), &policy); err != nil {
		t.Errorf("GenerateSecurityPolicy() returned invalid JSON: %v", err)
	}
}

func TestConsolidatePolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "{\"Sid\": \"Merged\", \"Effect\": \"Allow\"}"}}]}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "test-model")

	statements := []json.RawMessage{
		json.RawMessage(`{"Sid": "Test", "Effect": "Allow", "Action": ["s3:GetObject"]}`),
		json.RawMessage(`{"Sid": "Test", "Effect": "Allow", "Action": ["s3:PutObject"]}`),
	}

	result, err := client.ConsolidatePolicy(statements, nil, 1)

	if err != nil {
		t.Errorf("ConsolidatePolicy() unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Error("ConsolidatePolicy() returned empty result")
	}
}

func TestConsolidatePolicyNoDuplicates(t *testing.T) {
	client := NewLLMClient("http://unused", "", "test-model")

	statements := []json.RawMessage{
		json.RawMessage(`{"Sid": "Test1", "Effect": "Allow"}`),
		json.RawMessage(`{"Sid": "Test2", "Effect": "Deny"}`),
	}

	result, err := client.ConsolidatePolicy(statements, nil, 1)

	if err != nil {
		t.Errorf("ConsolidatePolicy() unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("ConsolidatePolicy() returned %d statements, want 2", len(result))
	}
}

func TestGenerateSCP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "{\"Version\": \"2012-10-17\", \"Statement\": []}"}}]}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "test-model")

	data := &ScrapedIAMData{
		ServiceName:   "Amazon S3",
		ServicePrefix: "s3",
	}

	result, err := client.GenerateSCP(data, `{"Version": "2012-10-17"}`, "test prompt")

	if err != nil {
		t.Errorf("GenerateSCP() unexpected error: %v", err)
	}
	if result == "" {
		t.Error("GenerateSCP() returned empty string")
	}

	var policy map[string]any
	if err := json.Unmarshal([]byte(result), &policy); err != nil {
		t.Errorf("GenerateSCP() returned invalid JSON: %v", err)
	}
}

func TestGeneratePolicyDocumentation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "# Policy Documentation\n\nThis is the documentation."}}]}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "test-model")

	data := &ScrapedIAMData{
		ServiceName:   "Amazon S3",
		ServicePrefix: "s3",
		Actions:       []IAMAction{{Name: "GetObject"}},
	}

	result, err := client.GeneratePolicyDocumentation(data, `{"Version": "2012-10-17"}`)

	if err != nil {
		t.Errorf("GeneratePolicyDocumentation() unexpected error: %v", err)
	}
	if !strings.Contains(result, "Policy Documentation") {
		t.Error("GeneratePolicyDocumentation() missing expected content")
	}
}

func TestGenerateCombinedDocumentation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"choices\": [{\"message\": {\"content\": \"# Combined Documentation\"}}]}"))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "test-model")

	data := &ScrapedIAMData{
		ServiceName:   "Amazon S3",
		ServicePrefix: "s3",
		Actions:       []IAMAction{{Name: "GetObject"}},
	}

	result, err := client.GenerateCombinedDocumentation(data, `{"Version": "2012-10-17"}`, `{"Version": "2012-10-17"}`)

	if err != nil {
		t.Errorf("GenerateCombinedDocumentation() unexpected error: %v", err)
	}
	if !strings.Contains(result, "Combined Documentation") {
		t.Error("GenerateCombinedDocumentation() missing expected content")
	}
}

func TestEnrichActionDescriptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "{\"actions\": [{\"name\": \"GetObject\", \"risk\": \"Low\", \"security_note\": \"Read only\"}]}"}}]}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "test-model")

	data := &ScrapedIAMData{
		ServicePrefix: "s3",
		Actions: []IAMAction{
			{Name: "GetObject", AccessLevel: "Read"},
		},
	}

	err := client.EnrichActionDescriptions(data, nil)

	if err != nil {
		t.Errorf("EnrichActionDescriptions() unexpected error: %v", err)
	}
}

func TestConsolidateStatementGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "{\"Sid\": \"Merged\", \"Effect\": \"Allow\", \"Action\": [\"s3:GetObject\", \"s3:PutObject\"]}"}}]}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "test-model")

	statements := []json.RawMessage{
		json.RawMessage(`{"Sid": "Test", "Effect": "Allow", "Action": ["s3:GetObject"]}`),
		json.RawMessage(`{"Sid": "Test", "Effect": "Allow", "Action": ["s3:PutObject"]}`),
	}

	result, err := client.ConsolidateStatementGroup("Test", statements)

	if err != nil {
		t.Errorf("ConsolidateStatementGroup() unexpected error: %v", err)
	}
	if result == nil {
		t.Error("ConsolidateStatementGroup() returned nil")
	}
}
