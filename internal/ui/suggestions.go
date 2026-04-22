package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/photon-hq/tuichat/internal/store"
)

const maxSuggestionsVisible = 5

// Reserved library-level commands; always appear in the palette.
var ReservedCommands = []store.CommandDef{
	{Name: "/new", Description: "start a new chat"},
	{Name: "/help", Description: "show keybindings and env vars"},
}

func FilterCommands(all []store.CommandDef, prefix string) []store.CommandDef {
	if !strings.HasPrefix(prefix, "/") {
		return nil
	}
	merged := append(append([]store.CommandDef{}, ReservedCommands...), all...)
	lower := strings.ToLower(prefix)
	out := merged[:0]
	for _, c := range merged {
		if strings.HasPrefix(strings.ToLower(c.Name), lower) {
			out = append(out, c)
		}
	}
	return out
}

// RenderSuggestions builds the palette panel. `selectedIndex` may be any int;
// it gets modulo'd to the match count.
func RenderSuggestions(theme Theme, matches []store.CommandDef, selectedIndex, innerWidth int) string {
	if len(matches) == 0 {
		return ""
	}
	selected := ((selectedIndex % len(matches)) + len(matches)) % len(matches)

	start := 0
	if selected >= maxSuggestionsVisible {
		start = selected - (maxSuggestionsVisible - 1)
	}
	end := start + maxSuggestionsVisible
	if end > len(matches) {
		end = len(matches)
	}
	visible := matches[start:end]

	longestName := 0
	for _, c := range visible {
		if len(c.Name) > longestName {
			longestName = len(c.Name)
		}
	}

	rows := make([]string, 0, len(visible)+1)
	for i, c := range visible {
		isSel := start+i == selected
		bg := theme.SuggestionBG
		if isSel {
			bg = theme.SuggestionSelectedBG
		}
		rowStyle := lipgloss.NewStyle().Background(bg).Width(innerWidth)
		arrowColor := theme.SystemColor
		nameColor := theme.InputColor
		if isSel {
			arrowColor = theme.PromptColor
			nameColor = theme.UserColor
		}
		arrow := "  "
		if isSel {
			arrow = "› "
		}
		arrowStyled := lipgloss.NewStyle().Background(bg).Foreground(arrowColor).Render(arrow)
		nameStyled := lipgloss.NewStyle().Background(bg).Foreground(nameColor).Render(padRight(c.Name, longestName))
		var desc string
		if c.Description != "" {
			dashStyled := lipgloss.NewStyle().Background(bg).Foreground(theme.BorderColor).Render("  — ")
			descStyled := lipgloss.NewStyle().Background(bg).Foreground(theme.SystemColor).Render(c.Description)
			desc = dashStyled + descStyled
		}
		rows = append(rows, rowStyle.Render(" "+arrowStyled+nameStyled+desc))
	}
	overflow := len(matches) - end
	if overflow > 0 {
		moreStyle := lipgloss.NewStyle().Background(theme.SuggestionBG).
			Foreground(theme.SystemColor).Width(innerWidth)
		rows = append(rows, moreStyle.Render("   +"+itoa(overflow)+" more…"))
	}
	return strings.Join(rows, "\n")
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
