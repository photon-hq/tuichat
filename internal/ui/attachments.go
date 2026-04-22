package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/photon-hq/tuichat/internal/store"
)

// RenderAttachmentChips returns a 1-line row of chips listing pending attachments.
// Empty string when there's nothing pending (so callers can skip allocating a row).
func RenderAttachmentChips(theme Theme, pending []store.PendingAttachment, width int) string {
	if len(pending) == 0 {
		return ""
	}
	bg := theme.SuggestionBG
	parts := make([]string, 0, len(pending))
	for _, a := range pending {
		label := "📎 " + a.Name
		if a.Size > 0 {
			label += " " + formatBytes(a.Size)
		}
		parts = append(parts, label)
	}
	line := " " + strings.Join(parts, "  ")
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(theme.AttachmentColor).
		Width(width).
		Render(line)
}
