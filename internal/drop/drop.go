// Package drop parses dragged-in file paths from bracketed-paste / raw input.
// Handles terminal-specific quoting (iTerm2 single-quotes, shell backslash-escapes, file:// URIs).
package drop

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	singleQuoted = regexp.MustCompile(`'([^']+)'`)
	doubleQuoted = regexp.MustCompile(`"([^"]+)"`)
)

// ParsePath normalizes a single dropped-path string. Returns "" if not a valid file.
func ParsePath(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "file://") {
		if u, err := url.Parse(s); err == nil && u.Path != "" {
			s = u.Path
		}
	}
	if (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) ||
		(strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)) {
		s = s[1 : len(s)-1]
	}
	// unescape shell backslash quoting (e.g. "\ ")
	s = regexp.MustCompile(`\\(.)`).ReplaceAllString(s, "$1")
	if strings.ContainsAny(s, "\r\n") {
		return ""
	}
	info, err := os.Stat(s)
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}
	return s
}

// Extracted holds the result of scanning a longer pasted string for any quoted
// file paths. Matches get removed from `Cleaned`; existing files become `Paths`.
type Extracted struct {
	Cleaned string
	Paths   []PathInfo
}

type PathInfo struct {
	Path string
	Name string
	Size int64
}

// Extract scans `value` for quoted substrings that resolve to files on disk.
// Useful for terminals that don't use bracketed paste (e.g. VS Code) and ship
// the quoted path as regular typed input.
func Extract(value string) Extracted {
	out := Extracted{Cleaned: value}
	toRemove := make([]string, 0, 4)
	for _, re := range []*regexp.Regexp{singleQuoted, doubleQuoted} {
		for _, m := range re.FindAllStringSubmatchIndex(value, -1) {
			full := value[m[0]:m[1]]
			inner := value[m[2]:m[3]]
			path := ParsePath(inner)
			if path == "" {
				continue
			}
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			toRemove = append(toRemove, full)
			out.Paths = append(out.Paths, PathInfo{
				Path: path,
				Name: filepath.Base(path),
				Size: info.Size(),
			})
		}
	}
	for _, r := range toRemove {
		out.Cleaned = strings.Replace(out.Cleaned, r, "", 1)
	}
	if len(out.Paths) > 0 {
		out.Cleaned = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(out.Cleaned, " "))
	}
	return out
}
