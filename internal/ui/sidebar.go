package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/photon-hq/tuichat/internal/store"
)

// SidebarWidth is the fixed width of the chat-list column, not including
// its right border.
const SidebarWidth = 20

// renderSidebar renders the sidebar with each chat row wrapped in a
// bubblezone marker so clicks can route to that chat. The middle of the
// column is a viewport.Model that scrolls when there are more chats than
// fit. Layout:
//
//	Chats              ← header (row 0, static)
//	> chat-1           ┐
//	  chat-2           │ rows fill m.sidebar (viewport)
//	  ...              │
//	                   ┘
//	Ctrl+N new         ← hint row (static, anchored)
//	Ctrl+J/K ↕         ← hint row (static, anchored)
func (m *Model) renderSidebar(chats []store.ChatState, activeID string, height int) string {
	theme := m.theme

	header := lipgloss.NewStyle().Foreground(theme.SystemColor).PaddingLeft(1).Render("Chats")

	// Build each chat row as a zone-marked line; feed them all into the
	// sidebar viewport. Keeping the selected row visible on activeID change
	// is a nice-to-have but not wired yet — viewport.YOffset stays where
	// the user last scrolled.
	rowStyle := lipgloss.NewStyle().PaddingLeft(1).Width(SidebarWidth - 1)
	rows := make([]string, 0, len(chats))
	for _, c := range chats {
		selected := c.ID == activeID
		arrowStyle := lipgloss.NewStyle().Foreground(theme.SystemColor)
		nameStyle := lipgloss.NewStyle().Foreground(theme.InputColor)
		if selected {
			arrowStyle = lipgloss.NewStyle().Foreground(theme.PromptColor)
			nameStyle = lipgloss.NewStyle().Foreground(theme.UserColor)
		}
		name := c.ID
		if len(name) > SidebarWidth-4 {
			name = name[:SidebarWidth-5] + "…"
		}
		arrow := "  "
		if selected {
			arrow = "› "
		}
		raw := arrowStyle.Render(arrow) + nameStyle.Render(name)
		marked := zone.Mark(ZoneSidebarRowPrefix+c.ID, raw)
		rows = append(rows, rowStyle.Render(marked))
	}
	m.sidebar.SetContent(strings.Join(rows, "\n"))

	hintStyle := lipgloss.NewStyle().Foreground(theme.SystemColor).PaddingLeft(1).Width(SidebarWidth - 1)
	hints := []string{
		hintStyle.Render("Ctrl+N new"),
		hintStyle.Render("Ctrl+J/K ↕"),
	}

	column := strings.Join(append(
		[]string{header, m.sidebar.View()},
		hints...,
	), "\n")

	return lipgloss.NewStyle().
		Width(SidebarWidth).
		MaxWidth(SidebarWidth + 1). // +1 for the right border
		Height(height).
		BorderStyle(lipgloss.Border{Right: "│"}).
		BorderRight(true).
		BorderForeground(theme.BorderColor).
		Render(column)
}
