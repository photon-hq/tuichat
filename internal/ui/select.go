package ui

import (
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/photon-hq/tuichat/internal/store"
)

// Message-select mode — arrow-key or click-targeted highlighting of a single
// entry, with an inline action row underneath that exposes reply / react.
// Reply banner rendering (shown above the input when composing a reply)
// also lives here since it reads `m.replyingTo`.

// canSelectMessages reports whether the current UI state is willing to
// redirect arrow keys to message navigation. System chat is read-only, and
// we don't hijack arrows away from the text input when the user is typing.
func (m *Model) canSelectMessages() bool {
	chat, ok := m.Store.ActiveChat()
	if !ok {
		return false
	}
	if chat.ID == store.SystemChatID || len(chat.Entries) == 0 {
		return false
	}
	if !m.selecting && m.input.Value() != "" {
		return false
	}
	return true
}

// moveSelection enters select mode on first press, then walks through message
// IDs in the active chat. Direction: -1 up (older), +1 down (newer).
func (m *Model) moveSelection(delta int) {
	chat, ok := m.Store.ActiveChat()
	if !ok || len(chat.Entries) == 0 {
		return
	}
	if !m.selecting {
		m.selecting = true
		// First press always lands on the most recent message.
		m.selectedID = chat.Entries[len(chat.Entries)-1].ID
		m.refreshViewport()
		return
	}
	idx := -1
	for i, e := range chat.Entries {
		if e.ID == m.selectedID {
			idx = i
			break
		}
	}
	if idx < 0 {
		m.selectedID = chat.Entries[len(chat.Entries)-1].ID
		m.refreshViewport()
		return
	}
	next := idx + delta
	if next < 0 {
		next = 0
	}
	if next > len(chat.Entries)-1 {
		next = len(chat.Entries) - 1
	}
	m.selectedID = chat.Entries[next].ID
	m.refreshViewport()
}

func (m *Model) exitSelectMode() {
	m.selecting = false
	m.selectedID = ""
}

// --- rendering ---

// renderReplyBanner draws the "↩ replying to: ..." hint above the input
// whenever a reply is pending.
func (m *Model) renderReplyBanner(chat store.ChatState, width int) string {
	quote := ""
	for _, e := range chat.Entries {
		if e.ID != m.replyingTo {
			continue
		}
		switch e.Content.Type {
		case "text":
			quote = truncateRunes(e.Content.Text, 60)
		case "attachment":
			quote = "[attachment: " + e.Content.Name + "]"
		case "voice":
			quote = "[voice]"
		case "contact":
			quote = "[contact]"
		case "custom":
			quote = "[custom]"
		case "richlink":
			label := e.Content.Title
			if label == "" {
				label = e.Content.Url
			}
			quote = "[link] " + label
		}
		break
	}
	if quote == "" {
		// Target message was dropped off scrollback or is otherwise missing.
		m.replyingTo = ""
		return ""
	}
	arrow := lipgloss.NewStyle().Foreground(m.theme.PromptColor).Render("↩ ")
	label := lipgloss.NewStyle().Foreground(m.theme.SystemColor).Render("replying to: ")
	body := lipgloss.NewStyle().Foreground(m.theme.InputColor).Render("\"" + quote + "\"")
	hint := lipgloss.NewStyle().Foreground(m.theme.SystemColor).Render("  (Esc to cancel)")
	line := arrow + label + body + hint
	return lipgloss.NewStyle().MaxWidth(width).Render(line)
}

// renderActionRow draws the `↩ reply (r)` / `🙂 react (e)` button row that
// appears beneath a selected message. Buttons use the suggestion-selected
// background so they stand out as chips against the SuggestionBG section
// fill that wraps the selected entry.
func renderActionRow(theme Theme) string {
	btn := func(label string, zoneID string) string {
		st := lipgloss.NewStyle().
			Foreground(theme.PromptColor).
			Background(theme.SuggestionSelectedBG).
			Padding(0, 1)
		return zone.Mark(zoneID, st.Render(label))
	}
	row := btn("↩ reply (r)", ZoneReplyButton) + "  " + btn("🙂 react (e)", ZoneReactButton)
	hint := lipgloss.NewStyle().Foreground(theme.SystemColor).Render("   ↑/↓ move · Esc cancel")
	return lipgloss.NewStyle().PaddingLeft(2).Render(row + hint)
}
