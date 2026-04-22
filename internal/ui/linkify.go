package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var urlRegex = regexp.MustCompile(`https?://[^\s<>()\[\]{}]+`)

// WrapOSC8 wraps `text` so terminals with OSC 8 support render it as a clickable hyperlink.
func WrapOSC8(url, text string) string {
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// LinkifyText scans a string for URLs, styles them with `linkStyle`, and styles
// non-URL runs with `textStyle`. Returns a single styled string.
func LinkifyText(text string, textStyle, linkStyle lipgloss.Style) string {
	matches := urlRegex.FindAllStringIndex(text, -1)
	if matches == nil {
		return textStyle.Render(text)
	}
	var b strings.Builder
	last := 0
	for _, m := range matches {
		if m[0] > last {
			b.WriteString(textStyle.Render(text[last:m[0]]))
		}
		urlStr := text[m[0]:m[1]]
		b.WriteString(linkStyle.Render(WrapOSC8(urlStr, urlStr)))
		last = m[1]
	}
	if last < len(text) {
		b.WriteString(textStyle.Render(text[last:]))
	}
	return b.String()
}
