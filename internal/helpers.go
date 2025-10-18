package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// GlobalExiter is the global exiter instance used by helper functions
var GlobalExiter Exiter = DefaultExiter

// AwsString safely dereferences an AWS string pointer
func AwsString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// IfEmpty returns rep if s is empty, otherwise returns s
func IfEmpty(s, rep string) string {
	if s == "" {
		return rep
	}
	return s
}

// StrPtr returns a pointer to the given string
func StrPtr(s string) *string {
	return &s
}

// MustAbs returns the absolute path or dies on error
func MustAbs(p string) string {
	ap, err := filepath.Abs(p)
	Check(err)
	return ap
}

// MustAbsJoin joins base and rel, resolving to absolute path
func MustAbsJoin(base, rel string) string {
	joined := rel
	if !filepath.IsAbs(rel) {
		joined = filepath.Join(base, rel)
	}
	return MustAbs(joined)
}

// Check panics with die if err is not nil
func Check(err error) {
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			Die("file not found: %v", err)
		}
		Die("%v", err)
	}
}

// Die prints an error message and exits with code 1
func Die(f string, a ...any) {
	fmt.Fprintf(os.Stderr, f+"\n", a...)
	GlobalExiter.Exit(1)
}
