// Package spool writes agent-sent attachment bytes to a temp file so the UI
// can point OSC 8 hyperlinks and Kitty image preview at a real path.
package spool

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

var spoolDir = filepath.Join(os.TempDir(), "tuichat")

// Attachment writes bytes to <tmp>/tuichat/<hash><ext> keyed by sha1(name+bytes).
// Subsequent calls with the same inputs reuse the file.
func Attachment(name string, bytes []byte) string {
	if err := os.MkdirAll(spoolDir, 0o755); err != nil {
		return ""
	}
	h := sha1.New()
	h.Write([]byte(name))
	h.Write(bytes)
	sum := hex.EncodeToString(h.Sum(nil))[:16]
	ext := safeExt(name)
	path := filepath.Join(spoolDir, sum+ext)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		return ""
	}
	return path
}

func safeExt(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return ""
	}
	for _, r := range ext {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '.') {
			return ""
		}
	}
	return strings.ToLower(ext)
}
