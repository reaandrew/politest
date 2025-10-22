package internal

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ExpandGlobsRelative expands glob patterns relative to a base directory
func ExpandGlobsRelative(base string, patterns []string) []string {
	var files []string
	seen := map[string]struct{}{}
	for _, pat := range patterns {
		p := MustAbsJoin(base, pat)
		matches, _ := filepath.Glob(p)
		// If literal file exists but glob found nothing, include it
		if len(matches) == 0 {
			if _, err := os.Stat(p); err == nil {
				matches = []string{p}
			}
		}
		sort.Strings(matches)
		for _, m := range matches {
			key := MustAbs(m)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			files = append(files, key)
		}
	}
	return files
}

// MergeSCPFiles merges multiple SCP JSON files into a single policy document
func MergeSCPFiles(files []string) map[string]any {
	merged, _ := MergeSCPFilesWithSourceMap(files)
	return merged
}

// MergeSCPFilesWithSourceMap merges multiple SCP JSON files and tracks statement origins with line numbers
func MergeSCPFilesWithSourceMap(files []string) (map[string]any, map[string]*PolicySource) {
	statements := []any{}
	sourceMap := make(map[string]*PolicySource)

	for _, f := range files {
		// Read the original file content for line number tracking
		fileContent, err := os.ReadFile(f)
		Check(err)

		var doc any
		Check(ReadJSONFile(f, &doc))

		var stmtsToAdd []any
		switch t := doc.(type) {
		case map[string]any:
			if st, ok := t["Statement"]; ok {
				// Extract statements from policy document
				if stmtArray, ok := st.([]any); ok {
					stmtsToAdd = stmtArray
				} else {
					stmtsToAdd = []any{st}
				}
			} else {
				// assume it's a single statement object
				stmtsToAdd = []any{t}
			}
		default:
			// treat as a statement-ish blob
			stmtsToAdd = []any{t}
		}

		// Inject unique Sid and track source for each statement
		for idx, stmt := range stmtsToAdd {
			if stmtMap, ok := stmt.(map[string]any); ok {
				// Create unique Sid from file path and statement index
				relPath := filepath.Base(f) // Use basename to keep Sids readable
				trackingSid := "scp:" + relPath + "#stmt:" + strconv.Itoa(idx)

				// Store original Sid if it exists
				originalSid := ""
				if existingSid, ok := stmtMap["Sid"]; ok {
					if sidStr, ok := existingSid.(string); ok {
						originalSid = sidStr
					}
				}

				// Find line numbers for this statement in the original file
				startLine, endLine := findStatementLineNumbers(string(fileContent), stmtMap, idx)

				// Inject our tracking Sid
				stmtMap["Sid"] = trackingSid

				// Track source
				sourceMap[trackingSid] = &PolicySource{
					FilePath:  f,
					Sid:       originalSid,
					Index:     idx,
					StartLine: startLine,
					EndLine:   endLine,
				}

				statements = append(statements, stmtMap)
			} else {
				statements = append(statements, stmt)
			}
		}
	}

	merged := map[string]any{
		"Version":   "2012-10-17",
		"Statement": statements,
	}

	return merged, sourceMap
}

// ProcessIdentityPolicyWithSourceMap processes an identity policy JSON and returns it with tracking Sids injected
// and a source map for each statement
func ProcessIdentityPolicyWithSourceMap(policyJSON string, filePath string) (string, map[string]*PolicySource) {
	// Read the original file content for line number tracking
	fileContent, err := os.ReadFile(filePath)
	Check(err)

	// Parse the policy JSON
	var policy map[string]any
	if err := json.Unmarshal([]byte(policyJSON), &policy); err != nil {
		Die("invalid JSON in identity policy: %v", err)
	}

	sourceMap := make(map[string]*PolicySource)

	// Extract statements
	var statements []any
	if st, ok := policy["Statement"]; ok {
		if stmtArray, ok := st.([]any); ok {
			statements = stmtArray
		} else {
			statements = []any{st}
		}
	} else {
		// No statements to track
		return policyJSON, sourceMap
	}

	// Process each statement to inject tracking Sids
	for idx, stmt := range statements {
		if stmtMap, ok := stmt.(map[string]any); ok {
			// Create tracking Sid
			trackingSid := "identity#stmt:" + strconv.Itoa(idx)

			// Store original Sid if it exists
			originalSid := ""
			if existingSid, ok := stmtMap["Sid"]; ok {
				if sidStr, ok := existingSid.(string); ok {
					originalSid = sidStr
				}
			}

			// Find line numbers for this statement
			startLine, endLine := findStatementLineNumbers(string(fileContent), stmtMap, idx)

			// Inject tracking Sid
			stmtMap["Sid"] = trackingSid

			// Track source
			sourceMap[trackingSid] = &PolicySource{
				FilePath:  filePath,
				Sid:       originalSid,
				Index:     idx,
				StartLine: startLine,
				EndLine:   endLine,
			}
		}
	}

	// Re-serialize the modified policy
	modifiedJSON, err := json.Marshal(policy)
	Check(err)

	return string(modifiedJSON), sourceMap
}

// findStatementLineNumbers finds the line numbers where a statement appears in the source file
func findStatementLineNumbers(fileContent string, stmt map[string]any, stmtIndex int) (int, int) {
	lines := strings.Split(fileContent, "\n")

	// Try to find the statement's Sid or Effect as a marker
	var searchKey, searchValue string
	if sid, ok := stmt["Sid"].(string); ok && sid != "" {
		searchKey = "Sid"
		searchValue = sid
	} else if effect, ok := stmt["Effect"].(string); ok {
		searchKey = "Effect"
		searchValue = effect
	}

	if searchKey == "" {
		return 0, 0
	}

	// Search for the key-value pair in the file
	searchPattern := "\"" + searchKey + "\"" + ":" // Look for "Sid": or "Effect":
	var startLine int

	for i, line := range lines {
		if strings.Contains(line, searchPattern) && strings.Contains(line, searchValue) {
			// Found the statement - since Sids are unique in a file, just use the first match
			// (stmtIndex is only needed when the same Sid appears multiple times, like in merged SCPs)

			// Search backwards for opening brace
			for j := i; j >= 0; j-- {
				trimmed := strings.TrimSpace(lines[j])
				if trimmed == "{" {
					startLine = j + 1 // 1-based line numbers
					break
				}
			}

			// Search forwards for closing brace
			braceCount := 0
			foundOpen := false
			for j := startLine - 1; j < len(lines); j++ {
				for _, ch := range lines[j] {
					if ch == '{' {
						braceCount++
						foundOpen = true
					} else if ch == '}' {
						braceCount--
						if foundOpen && braceCount == 0 {
							return startLine, j + 1 // 1-based line numbers
						}
					}
				}
			}
			// If we get here, we found the Sid but couldn't find boundaries
			break
		}
	}

	return 0, 0
}

// ReadJSONFile reads a JSON file and decodes it into v
func ReadJSONFile(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	return dec.Decode(v)
}

// MinifyJSON compacts JSON bytes into a minified string
func MinifyJSON(b []byte) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		// try to marshal if text was formatted
		var anyv any
		if e2 := json.Unmarshal(b, &anyv); e2 == nil {
			out, _ := json.Marshal(anyv)
			return string(out)
		}
		Die("invalid JSON produced by template: %v", err)
	}
	return buf.String()
}

// ToJSONMin marshals a value to minified JSON string
func ToJSONMin(v any) string {
	b, err := json.Marshal(v)
	Check(err)
	return string(b)
}

// ToJSONPretty marshals a value to pretty-printed JSON string with 2-space indentation
// This makes AWS API error messages reference line numbers instead of column numbers,
// significantly improving debuggability
func ToJSONPretty(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	Check(err)
	return string(b)
}
