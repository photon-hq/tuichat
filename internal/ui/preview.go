package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/photon-hq/tuichat/internal/kitty"
	"github.com/photon-hq/tuichat/internal/store"
)

const (
	PreviewCols = 40
	PreviewRows = 10
)

// RenderPreview returns a bordered floating panel showing the image for
// `preview`. Image ID is looked up via `kitty.EnsureTransmitted` during UI
// rendering — callers must pass the already-resolved id. Returns empty if
// not supported.
func RenderPreview(theme Theme, preview *store.HoveredPreview, imageID uint32) string {
	if preview == nil {
		return ""
	}
	headerStyle := lipgloss.NewStyle().Foreground(theme.AttachmentColor).Width(PreviewCols + 2)
	header := headerStyle.Render("📎 " + preview.Name)

	var body string
	if imageID > 0 {
		fg := kitty.HexColor(imageID)
		rows := kitty.PlaceholderRows(PreviewCols, PreviewRows)
		styledRows := make([]string, len(rows))
		for i, row := range rows {
			styledRows[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(row)
		}
		body = strings.Join(styledRows, "\n")
	} else {
		body = lipgloss.NewStyle().Foreground(theme.SystemColor).Render("[loading image…]")
	}

	panel := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderColor).
		Background(theme.SuggestionBG).
		Padding(0, 1).
		Render(panel)
}
