package internal

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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

// MergeSCPFilesWithSourceMap merges multiple SCP JSON files and tracks statement origins
func MergeSCPFilesWithSourceMap(files []string) (map[string]any, map[string]*PolicySource) {
	statements := []any{}
	sourceMap := make(map[string]*PolicySource)

	for _, f := range files {
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
				sid := "scp:" + relPath + "#stmt:" + strconv.Itoa(idx)

				// Store original Sid if it exists
				originalSid := ""
				if existingSid, ok := stmtMap["Sid"]; ok {
					if sidStr, ok := existingSid.(string); ok {
						originalSid = sidStr
					}
				}

				// Inject our tracking Sid
				stmtMap["Sid"] = sid

				// Track source
				sourceMap[sid] = &PolicySource{
					FilePath: f,
					Sid:      originalSid,
					Index:    idx,
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
