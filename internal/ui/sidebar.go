package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/photon-hq/tuichat/internal/store"
)

const SidebarWidth = 20

// RenderSidebar returns the left pane. Height is the full interior height
// so the "Ctrl+N new / Ctrl+J/K ↕" hints pin to the bottom via a spacer.
func RenderSidebar(theme Theme, chats []store.ChatState, activeID string, height int) string {
	header := lipgloss.NewStyle().
		Foreground(theme.SystemColor).
		PaddingLeft(1).
		Render("Chats")

	rowStyle := lipgloss.NewStyle().PaddingLeft(1).Width(SidebarWidth - 1)
	selectedArrowStyle := lipgloss.NewStyle().Foreground(theme.PromptColor)
	selectedNameStyle := lipgloss.NewStyle().Foreground(theme.UserColor)
	unselectedArrowStyle := lipgloss.NewStyle().Foreground(theme.SystemColor)
	unselectedNameStyle := lipgloss.NewStyle().Foreground(theme.InputColor)

	rows := make([]string, 0, len(chats)+1)
	rows = append(rows, header)
	for _, c := range chats {
		selected := c.ID == activeID
		var arrow, name string
		id := c.ID
		if len(id) > SidebarWidth-4 {
			id = id[:SidebarWidth-5] + "…"
		}
		if selected {
			arrow = selectedArrowStyle.Render("› ")
			name = selectedNameStyle.Render(id)
		} else {
			arrow = unselectedArrowStyle.Render("  ")
			name = unselectedNameStyle.Render(id)
		}
		rows = append(rows, rowStyle.Render(arrow+name))
	}

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.SystemColor).
		PaddingLeft(1).
		Width(SidebarWidth - 1)
	hints := []string{
		hintStyle.Render("Ctrl+N new"),
		hintStyle.Render("Ctrl+J/K ↕"),
	}

	// Compose: header + rows, a spacer to push hints to the bottom, then hints.
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
		Height(height).
		BorderStyle(lipgloss.Border{Right: "│"}).
		BorderRight(true).
		BorderForeground(theme.BorderColor).
		Render(column)
}
