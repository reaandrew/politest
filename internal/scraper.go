package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// cacheEntry stores cached HTML content with metadata
type cacheEntry struct {
	URL       string    `json:"url"`
	Content   string    `json:"content"`
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

const cacheDuration = 24 * time.Hour

// getCacheDir returns the cache directory path
func getCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "politest")
}

// getCacheKey generates a cache key from URL
func getCacheKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])[:16] + ".json"
}

// loadFromCache attempts to load cached content for a URL
func loadFromCache(url string) (string, bool) {
	cacheDir := getCacheDir()
	if cacheDir == "" {
		return "", false
	}

	cachePath := filepath.Join(cacheDir, getCacheKey(url))
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return "", false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return "", false
	}

	// Check if cache is still valid
	if time.Now().After(entry.ExpiresAt) {
		return "", false
	}

	return entry.Content, true
}

// saveToCache saves content to cache
func saveToCache(url, content string) {
	cacheDir := getCacheDir()
	if cacheDir == "" {
		return
	}

	// Create cache directory if needed
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		return
	}

	entry := cacheEntry{
		URL:       url,
		Content:   content,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(cacheDuration),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	cachePath := filepath.Join(cacheDir, getCacheKey(url))
	_ = os.WriteFile(cachePath, data, 0600)
}

// IAMAction represents a scraped IAM action from AWS documentation
type IAMAction struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	AccessLevel      string   `json:"access_level"`
	ResourceTypes    []string `json:"resource_types,omitempty"`
	ConditionKeys    []string `json:"condition_keys,omitempty"`
	DependentActions []string `json:"dependent_actions,omitempty"`
}

// IAMConditionKey represents a condition key from AWS documentation
type IAMConditionKey struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// IAMResourceType represents a resource type from AWS documentation
type IAMResourceType struct {
	Name          string   `json:"name"`
	ARN           string   `json:"arn"`
	ConditionKeys []string `json:"condition_keys,omitempty"`
}

// ScrapedIAMData contains all scraped IAM data from a service documentation page
type ScrapedIAMData struct {
	ServiceName   string            `json:"service_name"`
	ServicePrefix string            `json:"service_prefix"`
	Actions       []IAMAction       `json:"actions"`
	ConditionKeys []IAMConditionKey `json:"condition_keys"`
	ResourceTypes []IAMResourceType `json:"resource_types"`
	SourceURL     string            `json:"source_url"`
}

// ScrapeIAMDocumentation scrapes IAM actions, conditions, and resources from an AWS documentation page
func ScrapeIAMDocumentation(url string, progress ProgressReporter) (*ScrapedIAMData, error) {
	// Validate URL format
	if !strings.Contains(url, "docs.aws.amazon.com/service-authorization") {
		return nil, fmt.Errorf("invalid URL: must be an AWS service authorization reference page (e.g., https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazonbedrock.html)")
	}

	var htmlContent string

	// Check cache first
	if cached, ok := loadFromCache(url); ok {
		if progress != nil {
			progress.SetStatus("Using cached documentation (valid for 24h)...")
		}
		htmlContent = cached
	} else {
		if progress != nil {
			progress.SetStatus("Fetching AWS documentation page...")
		}

		// Fetch the page
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP error: %s", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		htmlContent = string(body)

		// Save to cache
		saveToCache(url, htmlContent)
	}

	if progress != nil {
		progress.SetStatus("Parsing HTML content...")
	}

	return parseIAMDocumentation(htmlContent, url, progress)
}

// parseIAMDocumentation parses the HTML content of an AWS IAM documentation page
func parseIAMDocumentation(htmlContent, sourceURL string, progress ProgressReporter) (*ScrapedIAMData, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	data := &ScrapedIAMData{
		SourceURL: sourceURL,
	}

	// Extract service name and prefix from the page
	data.ServiceName, data.ServicePrefix = extractServiceInfo(doc)
	if data.ServicePrefix == "" {
		return nil, fmt.Errorf("could not extract service prefix from page - is this a valid AWS service authorization page?")
	}

	if progress != nil {
		progress.SetStatus("Extracting IAM actions...")
	}

	// Find and parse the actions table
	data.Actions = extractActions(doc, progress)

	if progress != nil {
		progress.SetStatus("Extracting condition keys...")
	}

	// Find and parse the condition keys table
	data.ConditionKeys = extractConditionKeys(doc)

	if progress != nil {
		progress.SetStatus("Extracting resource types...")
	}

	// Find and parse the resource types table
	data.ResourceTypes = extractResourceTypes(doc)

	if len(data.Actions) == 0 {
		return nil, fmt.Errorf("no IAM actions found on page - verify the URL is correct")
	}

	return data, nil
}

// extractServiceInfo extracts the service name and prefix from the page
func extractServiceInfo(doc *html.Node) (string, string) {
	var serviceName, servicePrefix string

	// Strategy 1: Look for pattern "(service prefix: xxx)" in the full text
	// The prefix is often in a <code> tag after "service prefix:"
	fullText := getTextContent(doc)
	prefixPatterns := []string{
		`service prefix:\s*` + "`" + `?(\w+)` + "`" + `?`,
		`\(service prefix:\s*(\w+)\)`,
		`service prefix:\s*(\w+)`,
	}
	for _, pattern := range prefixPatterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		if matches := re.FindStringSubmatch(fullText); len(matches) > 1 {
			servicePrefix = strings.TrimSpace(matches[1])
			break
		}
	}

	// Strategy 2: Look for code elements that follow "service prefix:" text
	if servicePrefix == "" {
		var foundServicePrefix bool
		var walkForPrefix func(*html.Node)
		walkForPrefix = func(n *html.Node) {
			if foundServicePrefix {
				return
			}
			if n.Type == html.TextNode && strings.Contains(strings.ToLower(n.Data), "service prefix:") {
				// Look for the next code element sibling
				for sib := n.NextSibling; sib != nil; sib = sib.NextSibling {
					if sib.Type == html.ElementNode && sib.Data == "code" {
						servicePrefix = cleanText(getTextContent(sib))
						foundServicePrefix = true
						return
					}
					if sib.Type == html.ElementNode {
						// Check inside the element for code
						codes := findElements(sib, "code")
						if len(codes) > 0 {
							servicePrefix = cleanText(getTextContent(codes[0]))
							foundServicePrefix = true
							return
						}
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkForPrefix(c)
			}
		}
		walkForPrefix(doc)
	}

	// Try to extract from title or h1
	var extractTitle func(*html.Node)
	extractTitle = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "title" || n.Data == "h1") {
			serviceName = getTextContent(n)
			// Clean up common patterns
			serviceName = strings.TrimPrefix(serviceName, "Actions, resources, and condition keys for ")
			serviceName = strings.TrimSuffix(serviceName, " - Service Authorization Reference")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractTitle(c)
		}
	}
	extractTitle(doc)

	// If prefix not found, try to extract from action names
	if servicePrefix == "" {
		var findPrefix func(*html.Node)
		findPrefix = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "td" {
				text := getTextContent(n)
				if strings.Contains(text, ":") {
					parts := strings.Split(text, ":")
					if len(parts) == 2 && len(parts[0]) > 0 && len(parts[0]) < 30 {
						servicePrefix = parts[0]
						return
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				findPrefix(c)
			}
		}
		findPrefix(doc)
	}

	return serviceName, servicePrefix
}

// extractActions extracts all IAM actions from the actions table
func extractActions(doc *html.Node, progress ProgressReporter) []IAMAction {
	var actions []IAMAction

	// Find all tables and look for the actions table
	tables := findElements(doc, "table")

	for _, table := range tables {
		// Check if this is the actions table by looking at headers
		headers := extractTableHeaders(table)
		if !isActionsTable(headers) {
			continue
		}

		rows := findElements(table, "tr")
		total := len(rows) - 1 // Exclude header row
		current := 0

		for _, row := range rows {
			cells := findElements(row, "td")
			if len(cells) < 3 {
				continue // Skip header rows or malformed rows
			}

			action := parseActionRow(cells)
			if action.Name != "" {
				actions = append(actions, action)
			}

			current++
			if progress != nil && total > 0 {
				progress.SetProgress(current, total)
			}
		}
	}

	return actions
}

// isActionsTable checks if the table headers indicate an actions table
func isActionsTable(headers []string) bool {
	hasAction := false
	hasDescription := false
	hasAccessLevel := false

	for _, h := range headers {
		h = strings.ToLower(h)
		if strings.Contains(h, "action") {
			hasAction = true
		}
		if strings.Contains(h, "description") {
			hasDescription = true
		}
		if strings.Contains(h, "access") || strings.Contains(h, "level") {
			hasAccessLevel = true
		}
	}

	return hasAction && hasDescription && hasAccessLevel
}

// parseActionRow parses a single row from the actions table
func parseActionRow(cells []*html.Node) IAMAction {
	action := IAMAction{}

	if len(cells) >= 1 {
		action.Name = cleanText(getTextContent(cells[0]))
	}
	if len(cells) >= 2 {
		action.Description = cleanText(getTextContent(cells[1]))
	}
	if len(cells) >= 3 {
		action.AccessLevel = cleanText(getTextContent(cells[2]))
	}
	if len(cells) >= 4 {
		action.ResourceTypes = parseMultiValueCell(cells[3])
	}
	if len(cells) >= 5 {
		action.ConditionKeys = parseMultiValueCell(cells[4])
	}
	if len(cells) >= 6 {
		action.DependentActions = parseMultiValueCell(cells[5])
	}

	return action
}

// extractConditionKeys extracts condition keys from the condition keys table
func extractConditionKeys(doc *html.Node) []IAMConditionKey {
	var keys []IAMConditionKey

	tables := findElements(doc, "table")

	for _, table := range tables {
		headers := extractTableHeaders(table)
		if !isConditionKeysTable(headers) {
			continue
		}

		rows := findElements(table, "tr")
		for _, row := range rows {
			cells := findElements(row, "td")
			if len(cells) < 2 {
				continue
			}

			key := IAMConditionKey{}
			if len(cells) >= 1 {
				key.Name = cleanText(getTextContent(cells[0]))
			}
			if len(cells) >= 2 {
				key.Description = cleanText(getTextContent(cells[1]))
			}
			if len(cells) >= 3 {
				key.Type = cleanText(getTextContent(cells[2]))
			}

			if key.Name != "" {
				keys = append(keys, key)
			}
		}
	}

	return keys
}

// isConditionKeysTable checks if headers indicate a condition keys table
func isConditionKeysTable(headers []string) bool {
	hasCondition := false
	hasDescription := false

	for _, h := range headers {
		h = strings.ToLower(h)
		if strings.Contains(h, "condition") && strings.Contains(h, "key") {
			hasCondition = true
		}
		if strings.Contains(h, "description") {
			hasDescription = true
		}
	}

	return hasCondition && hasDescription
}

// extractResourceTypes extracts resource types from the resource types table
func extractResourceTypes(doc *html.Node) []IAMResourceType {
	var resources []IAMResourceType

	tables := findElements(doc, "table")

	for _, table := range tables {
		headers := extractTableHeaders(table)
		if !isResourceTypesTable(headers) {
			continue
		}

		rows := findElements(table, "tr")
		for _, row := range rows {
			cells := findElements(row, "td")
			if len(cells) < 2 {
				continue
			}

			resource := IAMResourceType{}
			if len(cells) >= 1 {
				resource.Name = cleanText(getTextContent(cells[0]))
			}
			if len(cells) >= 2 {
				resource.ARN = cleanText(getTextContent(cells[1]))
			}
			if len(cells) >= 3 {
				resource.ConditionKeys = parseMultiValueCell(cells[2])
			}

			if resource.Name != "" {
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// isResourceTypesTable checks if headers indicate a resource types table
func isResourceTypesTable(headers []string) bool {
	hasResource := false
	hasARN := false

	for _, h := range headers {
		h = strings.ToLower(h)
		if strings.Contains(h, "resource") && strings.Contains(h, "type") {
			hasResource = true
		}
		if strings.Contains(h, "arn") {
			hasARN = true
		}
	}

	return hasResource && hasARN
}

// Helper functions

func findElements(n *html.Node, tag string) []*html.Node {
	var elements []*html.Node
	var find func(*html.Node)
	find = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag {
			elements = append(elements, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(n)
	return elements
}

func extractTableHeaders(table *html.Node) []string {
	var headers []string
	ths := findElements(table, "th")
	for _, th := range ths {
		headers = append(headers, cleanText(getTextContent(th)))
	}
	return headers
}

func getTextContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	var text strings.Builder
	var extract func(*html.Node)
	extract = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(n)
	return text.String()
}

func cleanText(s string) string {
	// Remove excessive whitespace
	s = strings.TrimSpace(s)
	// Replace multiple spaces/newlines with single space
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")
	return s
}

func parseMultiValueCell(cell *html.Node) []string {
	var values []string
	text := getTextContent(cell)

	// Split by common delimiters
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == ','
	})

	for _, part := range parts {
		part = cleanText(part)
		// Remove asterisks (used to mark required resources)
		part = strings.TrimSuffix(part, "*")
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}

	return values
}
