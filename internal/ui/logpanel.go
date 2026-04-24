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
// input below spans both columns cleanly. The body is rendered into
// m.logPanel (viewport.Model) so the user can scroll through history.
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
	labelW := runewidth.StringWidth("system log")
	btnW := runewidth.StringWidth("[×]")
	pad := innerW - labelW - btnW
	if pad < 1 {
		pad = 1
	}
	header := headerLabel + strings.Repeat(" ", pad) + closeBtn
	rule := lipgloss.NewStyle().Foreground(m.theme.BorderColor).
		Render(strings.Repeat("─", innerW))

	// Render ALL entries — viewport clips to its Height and tracks scroll.
	entries := m.Store.SystemEntries()
	rendered := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Content.Type != "text" {
			continue
		}
		rendered = append(rendered, m.renderLogEntryLines(e, innerW)...)
	}

	// Capture plain-text mirror of body rows for drag-select extraction.
	// Full content (not just visible tail) — the selection mapper adds
	// m.logPanel.YOffset when translating screen coords to content rows.
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

	// Auto-scroll to the bottom on new entries iff the user was already
	// parked at the bottom. If they've scrolled back to read older entries,
	// respect that position — don't snap them to the tail.
	wasAtBottom := m.logPanel.AtBottom()
	m.logPanel.SetContent(strings.Join(rendered, "\n"))
	if wasAtBottom {
		m.logPanel.GotoBottom()
	}

	inner := strings.Join([]string{header, rule, m.logPanel.View()}, "\n")
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
