package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone"
	"github.com/mattn/go-runewidth"

	"github.com/photon-hq/tuichat/internal/store"
)

// The right-hand system-log panel: header, rule, tail of entries, with
// click-to-expand-per-entry and drag-select highlight when in the log pane.
//
// Layout painted by renderLogColumn:
//
//	[border 1 cell] [pad 1] [ "system log"  [×] ] [pad 1]   ← screen row 0
//	[border]        [pad 1] [     ─rule─        ] [pad 1]   ← screen row 1
//	[border]        [pad 1] [     body-0        ] [pad 1]   ← screen row 2
//	[border]        [pad 1] [     body-N        ] [pad 1]

// renderLogEntryLines renders a single system-log entry — one truncated line
// when collapsed, wrapped multi-line when expanded. Every line of the entry
// is wrapped in a single bubblezone so clicks anywhere on the block toggle
// expansion.
func (m *Model) renderLogEntryLines(e store.LogEntry, width int) []string {
	zoneID := ZoneLogEntryPrefix + e.ID
	text := e.Content.Text

	if !m.expandedLogs[e.ID] {
		body := text
		if runewidth.StringWidth(body) > width {
			body = runewidth.Truncate(body, width, "…")
		}
		return []string{zone.Mark(zoneID, renderLogText(m.theme, body))}
	}

	// Expanded: level token on the first line, continuation lines indented.
	level, rest := parseLogLevel(text)
	levelColor := m.theme.SystemColor
	switch level {
	case "error":
		levelColor = lipgloss.Color("#f87171")
	case "warn":
		levelColor = lipgloss.Color("#fbbf24")
	case "info":
		levelColor = lipgloss.Color("#60a5fa")
	case "debug":
		levelColor = lipgloss.Color("#9ca3af")
	}
	levelStyle := lipgloss.NewStyle().Foreground(levelColor).Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(m.theme.SystemColor)

	prefix := "[" + level + "]"
	prefixW := runewidth.StringWidth(prefix) + 1 // trailing space
	avail := width - prefixW
	if avail < 10 {
		avail = 10
	}
	chunks := wrapByWidth(rest, avail)
	if len(chunks) == 0 {
		chunks = []string{""}
	}
	indent := strings.Repeat(" ", prefixW)

	out := make([]string, 0, len(chunks))
	for i, c := range chunks {
		if i == 0 {
			out = append(out, levelStyle.Render(prefix)+" "+bodyStyle.Render(c))
		} else {
			out = append(out, indent+bodyStyle.Render(c))
		}
	}
	// Mark the entire block as one zone so clicks on continuation lines
	// collapse the entry just like clicks on the first line. bubblezone
	// tracks a multi-row bounding box when the start/end markers sit on
	// different rows.
	marked := zone.Mark(zoneID, strings.Join(out, "\n"))
	return strings.Split(marked, "\n")
}

// parseLogLevel splits a "[level] rest" string. Returns ("log", text) when
// the text doesn't carry a level prefix.
func parseLogLevel(text string) (string, string) {
	if !strings.HasPrefix(text, "[") {
		return "log", text
	}
	end := strings.Index(text, "] ")
	if end < 0 {
		return "log", text
	}
	return text[1:end], text[end+2:]
}

// wrapByWidth splits a plain string into lines each at most `w` visible
// cells wide. Rune-aware, not word-aware — fine for JSON and other log
// payloads that dominate the system log.
func wrapByWidth(s string, w int) []string {
	if w <= 0 {
		return []string{s}
	}
	var out []string
	var line strings.Builder
	lineW := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if rw == 0 {
			rw = 1
		}
		if lineW+rw > w && lineW > 0 {
			out = append(out, line.String())
			line.Reset()
			lineW = 0
		}
		line.WriteRune(r)
		lineW += rw
	}
	if line.Len() > 0 {
		out = append(out, line.String())
	}
	return out
}

// renderLogColumn renders the right-hand system-log panel. Height must match
// the combined height of titleBar + messages viewport + typing line so the
// input below spans both columns cleanly.
func (m *Model) renderLogColumn(height int) string {
	innerW := LogColumnWidth - 3 // 1 border + 2 padding
	if innerW < 8 {
		innerW = 8
	}

	closeBtn := zone.Mark(ZoneLogToggle,
		lipgloss.NewStyle().Foreground(m.theme.PromptColor).Render("[×]"),
	)
	headerLabel := lipgloss.NewStyle().
		Foreground(m.theme.SystemColor).
		Render("system log")
	// header: "system log" + right-aligned close button, padded to innerW
	labelW := runewidth.StringWidth("system log")
	btnW := runewidth.StringWidth("[×]")
	pad := innerW - labelW - btnW
	if pad < 1 {
		pad = 1
	}
	header := headerLabel + strings.Repeat(" ", pad) + closeBtn
	rule := lipgloss.NewStyle().Foreground(m.theme.BorderColor).
		Render(strings.Repeat("─", innerW))

	// Body: tail of entries that fits. Walk newest→oldest, render each to 1
	// or more lines, stop once we've filled bodyHeight. Reverse for display.
	entries := m.Store.SystemEntries()
	bodyHeight := height - 3 /*header + rule + bottom margin*/
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	rendered := make([]string, 0, bodyHeight)
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Content.Type != "text" {
			continue
		}
		lines := m.renderLogEntryLines(e, innerW)
		for j := len(lines) - 1; j >= 0; j-- {
			rendered = append(rendered, lines[j])
			if len(rendered) >= bodyHeight {
				break
			}
		}
		if len(rendered) >= bodyHeight {
			break
		}
	}
	for l, r := 0, len(rendered)-1; l < r; l, r = l+1, r-1 {
		rendered[l], rendered[r] = rendered[r], rendered[l]
	}
	for len(rendered) < bodyHeight {
		rendered = append([]string{""}, rendered...)
	}

	// Capture plain-text mirror of body rows for drag-select extraction.
	m.plainSystemLogLines = make([]string, len(rendered))
	for i, s := range rendered {
		m.plainSystemLogLines[i] = ansi.Strip(s)
	}

	// Overlay reverse-video on the selected span when drag-select is live in
	// the system-log pane. Mirrors the chat-pane branch in refreshViewport.
	if m.dragActive && m.dragPane == paneSystemLog {
		loRow, loCol, hiRow, hiCol := m.selectionRange()
		if loRow < 0 {
			loRow = 0
		}
		if hiRow >= len(rendered) {
			hiRow = len(rendered) - 1
		}
		reverse := lipgloss.NewStyle().Reverse(true)
		for r := loRow; r <= hiRow && r < len(m.plainSystemLogLines); r++ {
			plain := m.plainSystemLogLines[r]
			lineW := runewidth.StringWidth(plain)
			from, to := 0, lineW
			if r == loRow {
				from = loCol
			}
			if r == hiRow {
				to = hiCol
			}
			if from > lineW {
				from = lineW
			}
			if to > lineW {
				to = lineW
			}
			if from >= to {
				continue
			}
			left := ansi.Cut(rendered[r], 0, from)
			mid := reverse.Render(plainSliceByCols(plain, from, to))
			right := ansi.Cut(rendered[r], to, lineW)
			rendered[r] = left + mid + right
		}
	}

	inner := strings.Join(append([]string{header, rule}, rendered...), "\n")
	return lipgloss.NewStyle().
		Width(LogColumnWidth).
		Height(height).
		MaxWidth(LogColumnWidth).
		BorderStyle(lipgloss.Border{Left: "│"}).
		BorderLeft(true).
		BorderForeground(m.theme.BorderColor).
		Padding(0, 1).
		Render(inner)
}
