package internal

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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
	statements := []any{}
	for _, f := range files {
		var doc any
		Check(ReadJSONFile(f, &doc))
		switch t := doc.(type) {
		case map[string]any:
			if st, ok := t["Statement"]; ok {
				switch sv := st.(type) {
				case []any:
					statements = append(statements, sv...)
				default:
					statements = append(statements, sv)
				}
			} else {
				// assume it's a single statement object
				statements = append(statements, t)
			}
		default:
			// treat as a statement-ish blob
			statements = append(statements, t)
		}
	}
	return map[string]any{
		"Version":   "2012-10-17",
		"Statement": statements,
	}
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
