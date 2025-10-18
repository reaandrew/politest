package internal

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// RenderStringSlice renders a slice of strings using template variables
func RenderStringSlice(in []string, vars map[string]any) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, RenderTemplateString(s, vars))
	}
	return out
}

// RenderTemplateFileJSON reads a template file, renders it, and returns minified JSON
func RenderTemplateFileJSON(path string, vars map[string]any) string {
	tplText, err := os.ReadFile(path)
	Check(err)
	tpl := template.Must(template.New(filepath.Base(path)).Option("missingkey=error").Parse(string(tplText)))
	var buf bytes.Buffer
	Check(tpl.Execute(&buf, vars))
	// Validate and minify JSON
	return MinifyJSON(buf.Bytes())
}

// RenderTemplateString renders a template string with the given variables
func RenderTemplateString(s string, vars map[string]any) string {
	tpl := template.Must(template.New("inline").Option("missingkey=error").Parse(s))
	var buf bytes.Buffer
	Check(tpl.Execute(&buf, vars))
	return buf.String()
}

// RenderString is an alias for RenderTemplateString
func RenderString(s string, vars map[string]any) string {
	return RenderTemplateString(s, vars)
}

// RenderContext converts YAML context entries to IAM context entries with rendering
func RenderContext(in []ContextEntryYml, vars map[string]any) []iamtypes.ContextEntry {
	out := make([]iamtypes.ContextEntry, 0, len(in))
	for _, e := range in {
		values := make([]string, 0, len(e.ContextKeyValues))
		for _, v := range e.ContextKeyValues {
			values = append(values, RenderTemplateString(v, vars))
		}
		out = append(out, iamtypes.ContextEntry{
			ContextKeyName:   StrPtr(e.ContextKeyName),
			ContextKeyType:   ParseContextType(e.ContextKeyType),
			ContextKeyValues: values,
		})
	}
	return out
}

// ParseContextType converts a string to IAM context key type enum
func ParseContextType(t string) iamtypes.ContextKeyTypeEnum {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "string":
		return iamtypes.ContextKeyTypeEnumString
	case "stringlist":
		return iamtypes.ContextKeyTypeEnumStringList
	case "numeric":
		return iamtypes.ContextKeyTypeEnumNumeric
	case "numericlist":
		return iamtypes.ContextKeyTypeEnumNumericList
	case "boolean":
		return iamtypes.ContextKeyTypeEnumBoolean
	case "booleanlist":
		return iamtypes.ContextKeyTypeEnumBooleanList
	default:
		return iamtypes.ContextKeyTypeEnumString
	}
}
