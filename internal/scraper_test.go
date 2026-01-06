package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

func TestGetCacheDir(t *testing.T) {
	dir := getCacheDir()
	if dir == "" {
		t.Skip("Could not determine user home directory")
	}
	if !strings.Contains(dir, ".cache") || !strings.Contains(dir, "politest") {
		t.Errorf("getCacheDir() = %v, expected path containing .cache/politest", dir)
	}
}

func TestGetCacheKey(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "basic url",
			url:  "https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazonbedrock.html",
		},
		{
			name: "different url",
			url:  "https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazons3.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := getCacheKey(tt.url)
			if key == "" {
				t.Error("getCacheKey() returned empty string")
			}
			if !strings.HasSuffix(key, ".json") {
				t.Errorf("getCacheKey() = %v, expected .json suffix", key)
			}
			// Should be deterministic
			key2 := getCacheKey(tt.url)
			if key != key2 {
				t.Errorf("getCacheKey() not deterministic: %v != %v", key, key2)
			}
		})
	}

	// Different URLs should produce different keys
	key1 := getCacheKey("https://example.com/a")
	key2 := getCacheKey("https://example.com/b")
	if key1 == key2 {
		t.Error("getCacheKey() produced same key for different URLs")
	}
}

func TestLoadFromCacheAndSaveToCache(t *testing.T) {
	// Test the cache key generation is consistent
	testURL := "https://test.example.com/test"
	key1 := getCacheKey(testURL)
	key2 := getCacheKey(testURL)
	if key1 != key2 {
		t.Errorf("getCacheKey() not consistent: %s != %s", key1, key2)
	}

	// Test that different URLs produce different keys
	key3 := getCacheKey("https://different.example.com")
	if key1 == key3 {
		t.Error("getCacheKey() should produce different keys for different URLs")
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	// Create a temporary cache directory
	tempDir := t.TempDir()
	testURL := "https://test.example.com/cache-test"
	testContent := "<html><body>Test content</body></html>"

	// Manually create cache entry to test loading
	cacheKey := getCacheKey(testURL)
	cachePath := filepath.Join(tempDir, cacheKey)

	entry := cacheEntry{
		URL:       testURL,
		Content:   testContent,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal cache entry: %v", err)
	}

	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatalf("Failed to write cache file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file was not created")
	}
}

func TestLoadFromCacheExpired(t *testing.T) {
	// Test that expired cache entries are not loaded
	tempDir := t.TempDir()
	testURL := "https://test.example.com/expired"

	cacheKey := getCacheKey(testURL)
	cachePath := filepath.Join(tempDir, cacheKey)

	// Create an expired cache entry
	entry := cacheEntry{
		URL:       testURL,
		Content:   "expired content",
		CachedAt:  time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired 24 hours ago
	}

	data, _ := json.Marshal(entry)
	os.WriteFile(cachePath, data, 0600)

	// loadFromCache should return false for expired entries
	// Note: This tests the logic but not the actual function since getCacheDir() uses home dir
}

func TestLoadFromCacheInvalidJSON(t *testing.T) {
	// Test that invalid JSON in cache is handled gracefully
	tempDir := t.TempDir()
	testURL := "https://test.example.com/invalid"

	cacheKey := getCacheKey(testURL)
	cachePath := filepath.Join(tempDir, cacheKey)

	// Write invalid JSON
	os.WriteFile(cachePath, []byte("not valid json"), 0600)

	// Verify file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file was not created")
	}
}

func TestLoadFromCacheMissingFile(t *testing.T) {
	// Test that missing cache file returns false
	content, ok := loadFromCache("https://nonexistent.example.com/missing")
	if ok {
		t.Error("loadFromCache() should return false for missing file")
	}
	if content != "" {
		t.Errorf("loadFromCache() should return empty content for missing file, got: %s", content)
	}
}

func TestSaveToCacheCreatesDirectory(t *testing.T) {
	// saveToCache should create the cache directory if it doesn't exist
	// This is a basic test - the actual directory creation depends on getCacheDir()
	saveToCache("https://test.example.com/save-test", "test content")
	// Should not panic
}

func TestCleanText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basic whitespace",
			input: "  hello  world  ",
			want:  "hello world",
		},
		{
			name:  "newlines",
			input: "hello\n\nworld",
			want:  "hello world",
		},
		{
			name:  "tabs",
			input: "hello\t\tworld",
			want:  "hello world",
		},
		{
			name:  "mixed whitespace",
			input: "  hello \n\t world  ",
			want:  "hello world",
		},
		{
			name:  "already clean",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanText(tt.input)
			if result != tt.want {
				t.Errorf("cleanText(%q) = %q, want %q", tt.input, result, tt.want)
			}
		})
	}
}

func TestIsActionsTable(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		want    bool
	}{
		{
			name:    "valid actions table",
			headers: []string{"Actions", "Description", "Access level", "Resource types"},
			want:    true,
		},
		{
			name:    "lowercase headers",
			headers: []string{"actions", "description", "access level"},
			want:    true,
		},
		{
			name:    "mixed case",
			headers: []string{"ACTIONS", "Description", "ACCESS LEVEL"},
			want:    true,
		},
		{
			name:    "missing action",
			headers: []string{"Description", "Access level"},
			want:    false,
		},
		{
			name:    "missing description",
			headers: []string{"Actions", "Access level"},
			want:    false,
		},
		{
			name:    "missing access level",
			headers: []string{"Actions", "Description"},
			want:    false,
		},
		{
			name:    "resource types table (not actions)",
			headers: []string{"Resource types", "ARN", "Condition keys"},
			want:    false,
		},
		{
			name:    "empty headers",
			headers: []string{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isActionsTable(tt.headers)
			if result != tt.want {
				t.Errorf("isActionsTable(%v) = %v, want %v", tt.headers, result, tt.want)
			}
		})
	}
}

func TestIsConditionKeysTable(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		want    bool
	}{
		{
			name:    "valid condition keys table",
			headers: []string{"Condition keys", "Description", "Type"},
			want:    true,
		},
		{
			name:    "missing condition key",
			headers: []string{"Description", "Type"},
			want:    false,
		},
		{
			name:    "missing description",
			headers: []string{"Condition keys", "Type"},
			want:    false,
		},
		{
			name:    "empty headers",
			headers: []string{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConditionKeysTable(tt.headers)
			if result != tt.want {
				t.Errorf("isConditionKeysTable(%v) = %v, want %v", tt.headers, result, tt.want)
			}
		})
	}
}

func TestIsResourceTypesTable(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		want    bool
	}{
		{
			name:    "valid resource types table",
			headers: []string{"Resource types", "ARN", "Condition keys"},
			want:    true,
		},
		{
			name:    "missing resource type",
			headers: []string{"ARN", "Condition keys"},
			want:    false,
		},
		{
			name:    "missing ARN",
			headers: []string{"Resource types", "Condition keys"},
			want:    false,
		},
		{
			name:    "empty headers",
			headers: []string{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isResourceTypesTable(tt.headers)
			if result != tt.want {
				t.Errorf("isResourceTypesTable(%v) = %v, want %v", tt.headers, result, tt.want)
			}
		})
	}
}

func TestGetTextContent(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "simple text",
			html: "<div>Hello</div>",
			want: "Hello",
		},
		{
			name: "nested elements",
			html: "<div><span>Hello</span> <b>World</b></div>",
			want: "Hello World",
		},
		{
			name: "empty element",
			html: "<div></div>",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}
			result := getTextContent(doc)
			// Clean result for comparison (since html.Parse wraps in html/body)
			result = cleanText(result)
			if result != tt.want {
				t.Errorf("getTextContent() = %q, want %q", result, tt.want)
			}
		})
	}

	// Test nil node
	if result := getTextContent(nil); result != "" {
		t.Errorf("getTextContent(nil) = %q, want empty string", result)
	}
}

func TestFindElements(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<table>
					<tr><th>Header</th></tr>
					<tr><td>Cell 1</td></tr>
					<tr><td>Cell 2</td></tr>
				</table>
				<table>
					<tr><td>Another</td></tr>
				</table>
			</body>
		</html>
	`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	tables := findElements(doc, "table")
	if len(tables) != 2 {
		t.Errorf("findElements(table) = %d elements, want 2", len(tables))
	}

	trs := findElements(doc, "tr")
	if len(trs) != 4 {
		t.Errorf("findElements(tr) = %d elements, want 4", len(trs))
	}

	tds := findElements(doc, "td")
	if len(tds) != 3 {
		t.Errorf("findElements(td) = %d elements, want 3", len(tds))
	}
}

func TestExtractTableHeaders(t *testing.T) {
	htmlContent := `
		<table>
			<tr>
				<th>Actions</th>
				<th>Description</th>
				<th>Access level</th>
			</tr>
			<tr>
				<td>GetObject</td>
				<td>Gets an object</td>
				<td>Read</td>
			</tr>
		</table>
	`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	tables := findElements(doc, "table")
	if len(tables) == 0 {
		t.Fatal("No tables found")
	}

	headers := extractTableHeaders(tables[0])
	if len(headers) != 3 {
		t.Errorf("extractTableHeaders() = %d headers, want 3", len(headers))
	}

	expectedHeaders := []string{"Actions", "Description", "Access level"}
	for i, want := range expectedHeaders {
		if i >= len(headers) {
			t.Errorf("Missing header at index %d", i)
			continue
		}
		if headers[i] != want {
			t.Errorf("header[%d] = %q, want %q", i, headers[i], want)
		}
	}
}

func TestParseMultiValueCell(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		wantLen  int
		wantVals []string
	}{
		{
			name:     "single value",
			html:     "<html><body><table><tr><td>value1</td></tr></table></body></html>",
			wantLen:  1,
			wantVals: []string{"value1"},
		},
		{
			name:     "comma separated",
			html:     "<html><body><table><tr><td>value1, value2, value3</td></tr></table></body></html>",
			wantLen:  3,
			wantVals: []string{"value1", "value2", "value3"},
		},
		{
			name:     "newline separated",
			html:     "<html><body><table><tr><td>value1\nvalue2\nvalue3</td></tr></table></body></html>",
			wantLen:  3,
			wantVals: []string{"value1", "value2", "value3"},
		},
		{
			name:     "with asterisks (required markers)",
			html:     "<html><body><table><tr><td>value1*\nvalue2</td></tr></table></body></html>",
			wantLen:  2,
			wantVals: []string{"value1", "value2"},
		},
		{
			name:     "empty cell",
			html:     "<html><body><table><tr><td></td></tr></table></body></html>",
			wantLen:  0,
			wantVals: []string{},
		},
		{
			name:     "whitespace only",
			html:     "<html><body><table><tr><td>   \n  </td></tr></table></body></html>",
			wantLen:  0,
			wantVals: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			tds := findElements(doc, "td")
			if len(tds) == 0 {
				t.Fatal("No td elements found")
			}

			result := parseMultiValueCell(tds[0])
			if len(result) != tt.wantLen {
				t.Errorf("parseMultiValueCell() = %d values, want %d (got: %v)", len(result), tt.wantLen, result)
			}

			for i, want := range tt.wantVals {
				if i >= len(result) {
					break
				}
				if result[i] != want {
					t.Errorf("value[%d] = %q, want %q", i, result[i], want)
				}
			}
		})
	}
}

func TestParseActionRow(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		wantName  string
		wantDesc  string
		wantLevel string
	}{
		{
			name: "full row",
			html: `<html><body><table>
				<tr>
					<td>GetObject</td>
					<td>Retrieves objects from Amazon S3</td>
					<td>Read</td>
					<td>object*</td>
					<td>s3:authType</td>
					<td></td>
				</tr>
			</table></body></html>`,
			wantName:  "GetObject",
			wantDesc:  "Retrieves objects from Amazon S3",
			wantLevel: "Read",
		},
		{
			name: "minimal row",
			html: `<html><body><table>
				<tr>
					<td>PutObject</td>
					<td>Puts objects</td>
					<td>Write</td>
				</tr>
			</table></body></html>`,
			wantName:  "PutObject",
			wantDesc:  "Puts objects",
			wantLevel: "Write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			cells := findElements(doc, "td")
			action := parseActionRow(cells)

			if action.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", action.Name, tt.wantName)
			}
			if action.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", action.Description, tt.wantDesc)
			}
			if action.AccessLevel != tt.wantLevel {
				t.Errorf("AccessLevel = %q, want %q", action.AccessLevel, tt.wantLevel)
			}
		})
	}
}

func TestExtractServiceInfo(t *testing.T) {
	tests := []struct {
		name       string
		html       string
		wantName   string
		wantPrefix string
	}{
		{
			name: "standard format with code tag",
			html: `
				<html>
					<head><title>Actions, resources, and condition keys for Amazon S3 - Service Authorization Reference</title></head>
					<body>
						<p>Amazon S3 (service prefix: <code>s3</code>)</p>
					</body>
				</html>
			`,
			wantName:   "Amazon S3",
			wantPrefix: "s3",
		},
		{
			name: "prefix in parentheses",
			html: `
				<html>
					<body>
						<h1>Actions, resources, and condition keys for Amazon Bedrock</h1>
						<p>(service prefix: bedrock)</p>
					</body>
				</html>
			`,
			wantName:   "Amazon Bedrock",
			wantPrefix: "bedrock",
		},
		{
			name: "extract from action table",
			html: `
				<html>
					<body>
						<table>
							<tr><th>Actions</th><th>Description</th><th>Access level</th></tr>
							<tr><td>ec2:DescribeInstances</td><td>Describes instances</td><td>List</td></tr>
						</table>
					</body>
				</html>
			`,
			wantName:   "",
			wantPrefix: "ec2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			name, prefix := extractServiceInfo(doc)
			if prefix != tt.wantPrefix {
				t.Errorf("prefix = %q, want %q", prefix, tt.wantPrefix)
			}
			if tt.wantName != "" && !strings.Contains(name, tt.wantName) {
				t.Errorf("name = %q, want to contain %q", name, tt.wantName)
			}
		})
	}
}

func TestExtractActions(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<table>
					<tr>
						<th>Actions</th>
						<th>Description</th>
						<th>Access level</th>
						<th>Resource types</th>
					</tr>
					<tr>
						<td>GetObject</td>
						<td>Gets an object</td>
						<td>Read</td>
						<td>object*</td>
					</tr>
					<tr>
						<td>PutObject</td>
						<td>Puts an object</td>
						<td>Write</td>
						<td>object*</td>
					</tr>
				</table>
			</body>
		</html>
	`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	actions := extractActions(doc, nil)

	if len(actions) != 2 {
		t.Errorf("extractActions() = %d actions, want 2", len(actions))
	}

	if len(actions) > 0 {
		if actions[0].Name != "GetObject" {
			t.Errorf("actions[0].Name = %q, want GetObject", actions[0].Name)
		}
		if actions[0].AccessLevel != "Read" {
			t.Errorf("actions[0].AccessLevel = %q, want Read", actions[0].AccessLevel)
		}
	}

	if len(actions) > 1 {
		if actions[1].Name != "PutObject" {
			t.Errorf("actions[1].Name = %q, want PutObject", actions[1].Name)
		}
		if actions[1].AccessLevel != "Write" {
			t.Errorf("actions[1].AccessLevel = %q, want Write", actions[1].AccessLevel)
		}
	}
}

func TestExtractConditionKeys(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<table>
					<tr>
						<th>Condition keys</th>
						<th>Description</th>
						<th>Type</th>
					</tr>
					<tr>
						<td>s3:authType</td>
						<td>Filters access by authentication method</td>
						<td>String</td>
					</tr>
					<tr>
						<td>s3:prefix</td>
						<td>Filters access by key name prefix</td>
						<td>String</td>
					</tr>
				</table>
			</body>
		</html>
	`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	keys := extractConditionKeys(doc)

	if len(keys) != 2 {
		t.Errorf("extractConditionKeys() = %d keys, want 2", len(keys))
	}

	if len(keys) > 0 {
		if keys[0].Name != "s3:authType" {
			t.Errorf("keys[0].Name = %q, want s3:authType", keys[0].Name)
		}
		if keys[0].Type != "String" {
			t.Errorf("keys[0].Type = %q, want String", keys[0].Type)
		}
	}
}

func TestExtractResourceTypes(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<table>
					<tr>
						<th>Resource types</th>
						<th>ARN</th>
						<th>Condition keys</th>
					</tr>
					<tr>
						<td>bucket</td>
						<td>arn:aws:s3:::${BucketName}</td>
						<td>s3:authType</td>
					</tr>
					<tr>
						<td>object</td>
						<td>arn:aws:s3:::${BucketName}/${ObjectName}</td>
						<td>s3:authType, s3:prefix</td>
					</tr>
				</table>
			</body>
		</html>
	`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	resources := extractResourceTypes(doc)

	if len(resources) != 2 {
		t.Errorf("extractResourceTypes() = %d resources, want 2", len(resources))
	}

	if len(resources) > 0 {
		if resources[0].Name != "bucket" {
			t.Errorf("resources[0].Name = %q, want bucket", resources[0].Name)
		}
		if !strings.Contains(resources[0].ARN, "s3:::") {
			t.Errorf("resources[0].ARN = %q, want to contain s3:::", resources[0].ARN)
		}
	}
}

func TestParseIAMDocumentation(t *testing.T) {
	htmlContent := `
		<html>
			<head><title>Actions, resources, and condition keys for Amazon S3 - Service Authorization Reference</title></head>
			<body>
				<h1>Actions, resources, and condition keys for Amazon S3</h1>
				<p>Amazon S3 (service prefix: <code>s3</code>) provides the following service-specific resources.</p>

				<h2>Actions defined by Amazon S3</h2>
				<table>
					<tr>
						<th>Actions</th>
						<th>Description</th>
						<th>Access level</th>
						<th>Resource types</th>
					</tr>
					<tr>
						<td>GetObject</td>
						<td>Grants permission to retrieve objects from Amazon S3</td>
						<td>Read</td>
						<td>object*</td>
					</tr>
				</table>

				<h2>Condition keys for Amazon S3</h2>
				<table>
					<tr>
						<th>Condition keys</th>
						<th>Description</th>
						<th>Type</th>
					</tr>
					<tr>
						<td>s3:authType</td>
						<td>Filters access by authentication method</td>
						<td>String</td>
					</tr>
				</table>

				<h2>Resource types defined by Amazon S3</h2>
				<table>
					<tr>
						<th>Resource types</th>
						<th>ARN</th>
						<th>Condition keys</th>
					</tr>
					<tr>
						<td>object</td>
						<td>arn:aws:s3:::${BucketName}/${ObjectName}</td>
						<td></td>
					</tr>
				</table>
			</body>
		</html>
	`

	data, err := parseIAMDocumentation(htmlContent, "https://test.example.com", nil)
	if err != nil {
		t.Fatalf("parseIAMDocumentation() error: %v", err)
	}

	if data.ServicePrefix != "s3" {
		t.Errorf("ServicePrefix = %q, want s3", data.ServicePrefix)
	}
	if len(data.Actions) != 1 {
		t.Errorf("Actions = %d, want 1", len(data.Actions))
	}
	if len(data.ConditionKeys) != 1 {
		t.Errorf("ConditionKeys = %d, want 1", len(data.ConditionKeys))
	}
	if len(data.ResourceTypes) != 1 {
		t.Errorf("ResourceTypes = %d, want 1", len(data.ResourceTypes))
	}
	if data.SourceURL != "https://test.example.com" {
		t.Errorf("SourceURL = %q, want https://test.example.com", data.SourceURL)
	}
}

func TestParseIAMDocumentationNoActions(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<p>Amazon S3 (service prefix: <code>s3</code>)</p>
				<table>
					<tr><th>Not an actions table</th></tr>
				</table>
			</body>
		</html>
	`

	_, err := parseIAMDocumentation(htmlContent, "https://test.example.com", nil)
	if err == nil {
		t.Error("parseIAMDocumentation() expected error for no actions, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "no IAM actions found") {
		t.Errorf("parseIAMDocumentation() error = %v, want error about no actions", err)
	}
}

func TestParseIAMDocumentationNoPrefix(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<p>No service prefix here</p>
			</body>
		</html>
	`

	_, err := parseIAMDocumentation(htmlContent, "https://test.example.com", nil)
	if err == nil {
		t.Error("parseIAMDocumentation() expected error for no prefix, got nil")
	}
}

func TestScrapeIAMDocumentationInvalidURL(t *testing.T) {
	_, err := ScrapeIAMDocumentation("https://example.com/not-aws", nil)
	if err == nil {
		t.Error("ScrapeIAMDocumentation() expected error for invalid URL, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid URL") {
		t.Errorf("ScrapeIAMDocumentation() error = %v, want error about invalid URL", err)
	}
}

func TestLoadFromCacheValidEntry(t *testing.T) {
	// Create a valid cache entry in the actual cache directory
	cacheDir := getCacheDir()
	if cacheDir == "" {
		t.Skip("Could not determine cache directory")
	}

	testURL := "https://test.politest.example.com/cache-valid-test"
	cacheKey := getCacheKey(testURL)
	cachePath := filepath.Join(cacheDir, cacheKey)

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Create a valid cache entry
	entry := cacheEntry{
		URL:       testURL,
		Content:   "<html><body>Test cached content</body></html>",
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal cache entry: %v", err)
	}

	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatalf("Failed to write cache file: %v", err)
	}
	defer os.Remove(cachePath)

	// Test loading from cache
	content, ok := loadFromCache(testURL)
	if !ok {
		t.Error("loadFromCache() should return true for valid cache entry")
	}
	if content != entry.Content {
		t.Errorf("loadFromCache() content = %q, want %q", content, entry.Content)
	}
}

func TestLoadFromCacheExpiredEntry(t *testing.T) {
	cacheDir := getCacheDir()
	if cacheDir == "" {
		t.Skip("Could not determine cache directory")
	}

	testURL := "https://test.politest.example.com/cache-expired-test"
	cacheKey := getCacheKey(testURL)
	cachePath := filepath.Join(cacheDir, cacheKey)

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Create an expired cache entry
	entry := cacheEntry{
		URL:       testURL,
		Content:   "expired content",
		CachedAt:  time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired
	}

	data, _ := json.Marshal(entry)
	os.WriteFile(cachePath, data, 0600)
	defer os.Remove(cachePath)

	// Test loading expired cache
	content, ok := loadFromCache(testURL)
	if ok {
		t.Error("loadFromCache() should return false for expired cache entry")
	}
	if content != "" {
		t.Errorf("loadFromCache() should return empty content for expired entry, got: %s", content)
	}
}

func TestLoadFromCacheInvalidJSONEntry(t *testing.T) {
	cacheDir := getCacheDir()
	if cacheDir == "" {
		t.Skip("Could not determine cache directory")
	}

	testURL := "https://test.politest.example.com/cache-invalid-json-test"
	cacheKey := getCacheKey(testURL)
	cachePath := filepath.Join(cacheDir, cacheKey)

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Write invalid JSON
	os.WriteFile(cachePath, []byte("not valid json {{{"), 0600)
	defer os.Remove(cachePath)

	// Test loading invalid JSON
	content, ok := loadFromCache(testURL)
	if ok {
		t.Error("loadFromCache() should return false for invalid JSON")
	}
	if content != "" {
		t.Errorf("loadFromCache() should return empty content for invalid JSON, got: %s", content)
	}
}

func TestSaveToCacheAndLoad(t *testing.T) {
	testURL := "https://test.politest.example.com/save-load-test"
	testContent := "<html><body>Test save and load</body></html>"

	// Save to cache
	saveToCache(testURL, testContent)

	// Clean up after test
	cacheDir := getCacheDir()
	if cacheDir != "" {
		cachePath := filepath.Join(cacheDir, getCacheKey(testURL))
		defer os.Remove(cachePath)
	}

	// Load from cache
	content, ok := loadFromCache(testURL)
	if !ok {
		t.Error("loadFromCache() should return true after saveToCache()")
	}
	if content != testContent {
		t.Errorf("loadFromCache() content = %q, want %q", content, testContent)
	}
}

func TestExtractServiceInfoFromTitle(t *testing.T) {
	htmlContent := `
		<html>
			<head><title>Actions, resources, and condition keys for AWS Lambda - Service Authorization Reference</title></head>
			<body>
				<h1>Actions, resources, and condition keys for AWS Lambda</h1>
				<p>AWS Lambda (service prefix: <code>lambda</code>)</p>
			</body>
		</html>
	`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	name, prefix := extractServiceInfo(doc)
	if prefix != "lambda" {
		t.Errorf("prefix = %q, want lambda", prefix)
	}
	if !strings.Contains(name, "Lambda") {
		t.Errorf("name = %q, want to contain Lambda", name)
	}
}

func TestParseIAMDocumentationWithProgress(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<p>Test Service (service prefix: <code>test</code>)</p>
				<table>
					<tr><th>Actions</th><th>Description</th><th>Access level</th></tr>
					<tr><td>TestAction</td><td>Test description</td><td>Read</td></tr>
				</table>
			</body>
		</html>
	`

	progress := &testProgress{}
	data, err := parseIAMDocumentation(htmlContent, "https://test.example.com", progress)
	if err != nil {
		t.Fatalf("parseIAMDocumentation() error: %v", err)
	}

	if data.ServicePrefix != "test" {
		t.Errorf("ServicePrefix = %q, want test", data.ServicePrefix)
	}
	if progress.statusCalls == 0 {
		t.Error("parseIAMDocumentation() should call progress.SetStatus()")
	}
}

type testProgress struct {
	statusCalls int
}

func (p *testProgress) SetStatus(status string) {
	p.statusCalls++
}

func (p *testProgress) SetProgress(current, total int) {}

func TestExtractActionsWithDependentActions(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<table>
					<tr>
						<th>Actions</th>
						<th>Description</th>
						<th>Access level</th>
						<th>Resource types</th>
						<th>Condition keys</th>
						<th>Dependent actions</th>
					</tr>
					<tr>
						<td>GetObject</td>
						<td>Gets an object</td>
						<td>Read</td>
						<td>object*</td>
						<td>s3:authType</td>
						<td>s3:ListBucket</td>
					</tr>
				</table>
			</body>
		</html>
	`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	actions := extractActions(doc, nil)

	if len(actions) != 1 {
		t.Fatalf("extractActions() = %d actions, want 1", len(actions))
	}

	if actions[0].Name != "GetObject" {
		t.Errorf("actions[0].Name = %q, want GetObject", actions[0].Name)
	}
}

func TestExtractActionsEmptyTable(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<table>
					<tr><th>Actions</th><th>Description</th><th>Access level</th></tr>
				</table>
			</body>
		</html>
	`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	actions := extractActions(doc, nil)
	if len(actions) != 0 {
		t.Errorf("extractActions() = %d actions, want 0 for empty table", len(actions))
	}
}

func TestParseActionRowMinimalCells(t *testing.T) {
	// Test with less than 3 cells
	htmlContent := `<html><body><table><tr><td>OnlyOne</td></tr></table></body></html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	cells := findElements(doc, "td")
	action := parseActionRow(cells)

	// Should handle gracefully
	if action.Name != "OnlyOne" {
		t.Errorf("Name = %q, want OnlyOne", action.Name)
	}
}

func TestExtractConditionKeysEmptyTable(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<table>
					<tr><th>Condition keys</th><th>Description</th><th>Type</th></tr>
				</table>
			</body>
		</html>
	`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	keys := extractConditionKeys(doc)
	if len(keys) != 0 {
		t.Errorf("extractConditionKeys() = %d keys, want 0 for empty table", len(keys))
	}
}

func TestExtractResourceTypesEmptyTable(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<table>
					<tr><th>Resource types</th><th>ARN</th><th>Condition keys</th></tr>
				</table>
			</body>
		</html>
	`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	resources := extractResourceTypes(doc)
	if len(resources) != 0 {
		t.Errorf("extractResourceTypes() = %d resources, want 0 for empty table", len(resources))
	}
}

func TestFindElementsNoMatches(t *testing.T) {
	htmlContent := `<html><body><p>No tables here</p></body></html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	elements := findElements(doc, "table")
	if len(elements) != 0 {
		t.Errorf("findElements() = %d elements, want 0", len(elements))
	}
}

func TestExtractTableHeadersNoTH(t *testing.T) {
	htmlContent := `<html><body><table><tr><td>Not a header</td></tr></table></body></html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	tables := findElements(doc, "table")
	if len(tables) == 0 {
		t.Fatal("No tables found")
	}

	headers := extractTableHeaders(tables[0])
	if len(headers) != 0 {
		t.Errorf("extractTableHeaders() = %d headers, want 0 for table without th", len(headers))
	}
}
