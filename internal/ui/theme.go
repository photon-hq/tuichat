package ui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Title                string
	UserColor            lipgloss.Color
	AgentColor           lipgloss.Color
	SystemColor          lipgloss.Color
	TimestampColor       lipgloss.Color
	BorderColor          lipgloss.Color
	SuggestionBG         lipgloss.Color
	SuggestionSelectedBG lipgloss.Color
	TypingColor          lipgloss.Color
	AttachmentColor      lipgloss.Color
	CustomColor          lipgloss.Color
	InputColor           lipgloss.Color
	PromptColor          lipgloss.Color
}

var DefaultTheme = Theme{
	Title:                "tuichat",
	UserColor:            lipgloss.Color("#7dd3fc"),
	AgentColor:           lipgloss.Color("#a78bfa"),
	SystemColor:          lipgloss.Color("#6b7280"),
	TimestampColor:       lipgloss.Color("#4b5563"),
	BorderColor:          lipgloss.Color("#374151"),
	SuggestionBG:         lipgloss.Color("#1f2937"),
	SuggestionSelectedBG: lipgloss.Color("#374151"),
	TypingColor:          lipgloss.Color("#f59e0b"),
	AttachmentColor:      lipgloss.Color("#34d399"),
	CustomColor:          lipgloss.Color("#f472b6"),
	InputColor:           lipgloss.Color("#e5e7eb"),
	PromptColor:          lipgloss.Color("#7dd3fc"),
}

func (t Theme) RolePrefix(role string) string {
	switch role {
	case "user":
		return "you  "
	case "agent":
		return "agent"
	case "system":
		return "sys  "
	}
	return "     "
}

func (t Theme) RoleColor(role string) lipgloss.Color {
	switch role {
	case "user":
		return t.UserColor
	case "agent":
		return t.AgentColor
	}
	return t.SystemColor
}
