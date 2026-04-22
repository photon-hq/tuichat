package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/photon-hq/tuichat/internal/store"
)

func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1fkB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(n)/1024/1024)
}

// RenderEntries builds the scrollable log content (one big string).
func RenderEntries(theme Theme, chat store.ChatState, width int) string {
	lines := make([]string, 0, len(chat.Entries)+2)
	if chat.DroppedCount > 0 {
		msg := fmt.Sprintf("… %d older messages dropped", chat.DroppedCount)
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.SystemColor).Render(msg))
	}
	for i := range chat.Entries {
		lines = append(lines, renderEntry(theme, chat.Entries[i], chat.Entries, width))
	}
	return strings.Join(lines, "\n")
}

func renderEntry(theme Theme, e store.LogEntry, allEntries []store.LogEntry, width int) string {
	role := string(e.Role)
	prefix := theme.RolePrefix(role)
	color := theme.RoleColor(role)
	ts := e.Timestamp.Format("15:04:05")
	tsPart := lipgloss.NewStyle().Foreground(theme.TimestampColor).Render("[" + ts + "] ")
	rolePart := lipgloss.NewStyle().Foreground(color).Render(prefix)
	sep := lipgloss.NewStyle().Foreground(theme.BorderColor).Render(" › ")

	var body string
	switch e.Content.Type {
	case "text":
		textStyle := lipgloss.NewStyle().Foreground(theme.InputColor)
		linkStyle := lipgloss.NewStyle().Foreground(theme.PromptColor).Underline(true)
		body = LinkifyText(e.Content.Text, textStyle, linkStyle)
	case "attachment":
		body = renderAttachmentLabel(theme, e, "attachment")
	case "voice":
		body = renderAttachmentLabel(theme, e, "voice")
	case "contact":
		body = lipgloss.NewStyle().Foreground(theme.AttachmentColor).Render("[contact]")
	case "custom":
		attachStyle := lipgloss.NewStyle().Foreground(theme.CustomColor)
		body = attachStyle.Render("[custom] ") +
			lipgloss.NewStyle().Foreground(theme.SystemColor).Render(safeStringify(e.Content.Raw))
	default:
		return ""
	}

	main := tsPart + rolePart + sep + body

	if e.ReplyTo != "" {
		quoted := findEntryQuote(allEntries, e.ReplyTo)
		if quoted != "" {
			rStyle := lipgloss.NewStyle().Foreground(theme.BorderColor)
			qStyle := lipgloss.NewStyle().Foreground(theme.SystemColor)
			quoteLine := rStyle.Render("  ┌─ ") + qStyle.Render(quoted)
			main = quoteLine + "\n" + main
		}
	}

	if len(e.Reactions) > 0 {
		reactStyle := lipgloss.NewStyle().Foreground(theme.SystemColor)
		main += "\n" + lipgloss.NewStyle().Foreground(theme.BorderColor).Render("       ") + reactStyle.Render(strings.Join(e.Reactions, " "))
	}

	_ = width // not used for hard-wrapping yet; terminal handles soft wrap
	return main
}

func renderAttachmentLabel(theme Theme, e store.LogEntry, kind string) string {
	style := lipgloss.NewStyle().Foreground(theme.AttachmentColor)
	name := e.Content.Name
	if name == "" {
		name = "attachment"
	}
	size := ""
	if e.Content.Size != nil {
		size = " " + formatBytes(*e.Content.Size)
	}
	label := fmt.Sprintf("[%s: ", kind)
	tail := size + "]"
	if e.AttachmentPath != "" {
		href := "file://" + e.AttachmentPath
		wrapped := WrapOSC8(href, name)
		return style.Render(label) + style.Render(wrapped) + style.Render(tail)
	}
	return style.Render(label + name + tail)
}

func findEntryQuote(entries []store.LogEntry, id string) string {
	for i := range entries {
		if entries[i].ID != id {
			continue
		}
		e := entries[i]
		c := e.Content
		switch c.Type {
		case "text":
			return truncateRunes(c.Text, 60)
		case "attachment":
			return "[attachment: " + c.Name + "]"
		case "voice":
			name := c.Name
			if name == "" {
				name = "audio"
			}
			return "[voice: " + name + "]"
		case "contact":
			return "[contact]"
		case "custom":
			return "[custom]"
		}
	}
	return ""
}

func truncateRunes(s string, max int) string {
	if len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max-1]) + "…"
}

func safeStringify(raw []byte) string {
	if len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}
