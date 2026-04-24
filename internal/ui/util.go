package ui

import (
	"os"
	"strings"
)

// fileBase returns the base name of a path without importing `filepath` —
// kept tiny so we can stay in the stdlib pool this package already uses.
func fileBase(p string) string {
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// statSafe returns file info (or nil) without propagating errors. Callers
// only need the size + existence check, so a nil return says "not a file".
func statSafe(p string) interface {
	Size() int64
} {
	info, err := os.Stat(p)
	if err != nil {
		return nil
	}
	return info
}

// guessMime maps common filename extensions to MIME types. Falls back to
// application/octet-stream for unrecognized suffixes so downstream MIME-
// aware paths (e.g. Kitty image preview gating) can still check the type.
func guessMime(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".bmp"):
		return "image/bmp"
	case strings.HasSuffix(lower, ".txt"):
		return "text/plain"
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
