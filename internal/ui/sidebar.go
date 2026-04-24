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

// zoneMarkSidebar renders the sidebar with each chat row wrapped in a
// bubblezone marker so clicks can route to that chat. Layout:
//
//	Chats
//	> chat-1
//	  chat-2
//	  ...
//	<spacer pushing hints to the bottom>
//	Ctrl+N new
//	Ctrl+J/K ↕
func zoneMarkSidebar(theme Theme, chats []store.ChatState, activeID string, height int) string {
	header := lipgloss.NewStyle().Foreground(theme.SystemColor).PaddingLeft(1).Render("Chats")
	rows := []string{header}
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
		rows = append(rows, lipgloss.NewStyle().PaddingLeft(1).Width(SidebarWidth-1).Render(marked))
	}
	hintStyle := lipgloss.NewStyle().Foreground(theme.SystemColor).PaddingLeft(1).Width(SidebarWidth - 1)
	hints := []string{
		hintStyle.Render("Ctrl+N new"),
		hintStyle.Render("Ctrl+J/K ↕"),
	}
	top := strings.Join(rows, "\n")
	bottom := strings.Join(hints, "\n")
	used := strings.Count(top, "\n") + 1 + strings.Count(bottom, "\n") + 1
	spacer := ""
	if height > used {
		spacer = strings.Repeat("\n", height-used)
	}
	column := top + "\n" + spacer + bottom
	return lipgloss.NewStyle().
		Width(SidebarWidth).
		MaxWidth(SidebarWidth + 1). // +1 for the right border
		Height(height).
		BorderStyle(lipgloss.Border{Right: "│"}).
		BorderRight(true).
		BorderForeground(theme.BorderColor).
		Render(column)
}
