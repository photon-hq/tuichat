package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/rivo/uniseg"
	"go.dalton.dog/bubbleup"

	"github.com/photon-hq/tuichat/internal/store"
)

// Reaction picker: 6 quick-pick emojis + an "other" slot that swaps in a
// textinput so the user can type or paste any emoji.

// reactionPicks is the fixed set of quick-pick reactions shown in the
// picker strip.
var reactionPicks = []string{"👍", "❤️", "😂", "😮", "😢", "🎉"}

// reactionOtherIdx is the picker slot index that triggers the custom-emoji
// text input. Equals len(reactionPicks).
var reactionOtherIdx = len(reactionPicks) // 6

// --- key handling ---

func (m *Model) handleReactionKey(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	// Custom-emoji text-input submode intercepts everything except Enter
	// (submit) and Esc (back to picker).
	if m.emojiInputMode {
		return m.handleEmojiInputKey(msg, key)
	}

	switch key {
	case "esc":
		m.cancelReactionPicker()
		return m, nil
	case "left":
		if m.reactionIdx > 0 {
			m.reactionIdx--
		}
		return m, nil
	case "right":
		if m.reactionIdx < reactionOtherIdx {
			m.reactionIdx++
		}
		return m, nil
	case "enter":
		if m.reactionIdx == reactionOtherIdx {
			m.enterEmojiInputMode()
			return m, textinput.Blink
		}
		m.submitReaction(reactionPicks[m.reactionIdx])
		return m, nil
	}
	// Digit shortcuts: 1-6 fire a quick emoji, 7 opens the "other" field.
	if len(key) == 1 && key >= "1" && key <= "7" {
		idx := int(key[0] - '1')
		if idx == reactionOtherIdx {
			m.enterEmojiInputMode()
			return m, textinput.Blink
		}
		if idx < len(reactionPicks) {
			m.submitReaction(reactionPicks[idx])
		}
	}
	return m, nil
}

func (m *Model) handleEmojiInputKey(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		// Bail out of the text field back to the picker strip.
		m.emojiInputMode = false
		m.emojiInput.SetValue("")
		m.emojiInput.Blur()
		m.refreshViewport()
		return m, nil
	case "enter":
		text := strings.TrimSpace(m.emojiInput.Value())
		if text == "" {
			return m, nil
		}
		if !isSingleEmoji(text) {
			// Reject plain text / multi-emoji input. Keep the field open
			// so the user can correct; flash an error toast with a hint.
			m.emojiInput.SetValue("")
			return m, m.alert.NewAlertCmd(
				bubbleup.ErrorKey,
				"only one emoji is allowed",
			)
		}
		m.emojiInputMode = false
		m.emojiInput.SetValue("")
		m.emojiInput.Blur()
		m.submitReaction(text)
		return m, nil
	}
	var cmd tea.Cmd
	m.emojiInput, cmd = m.emojiInput.Update(msg)
	return m, cmd
}

// isSingleEmoji validates that the string is a single user-perceived
// character (one grapheme cluster) made entirely of non-ASCII codepoints.
// Catches plain text, multiple emojis, "emoji word", etc. Uses
// rivo/uniseg for grapheme-cluster counting so ZWJ / skin-tone /
// variation-selector sequences still count as one emoji.
func isSingleEmoji(s string) bool {
	if s == "" {
		return false
	}
	if uniseg.GraphemeClusterCount(s) != 1 {
		return false
	}
	for _, r := range s {
		if r < 0x80 {
			return false
		}
	}
	return true
}

// --- state helpers ---

func (m *Model) enterEmojiInputMode() {
	m.emojiInputMode = true
	m.emojiInput.SetValue("")
	m.emojiInput.Focus()
	m.refreshViewport()
}

func (m *Model) cancelReactionPicker() {
	m.reactingTo = ""
	m.reactionIdx = 0
	m.emojiInputMode = false
	m.emojiInput.SetValue("")
	m.emojiInput.Blur()
	m.refreshViewport()
}

func (m *Model) submitReaction(emoji string) {
	chatID := m.Store.ActiveChatID()
	targetID := m.reactingTo
	m.reactingTo = ""
	m.reactionIdx = 0
	m.emojiInputMode = false
	m.emojiInput.SetValue("")
	m.emojiInput.Blur()
	if chatID == "" || targetID == "" {
		m.refreshViewport()
		return
	}
	m.Store.React(chatID, targetID, emoji, store.RoleUser)
	m.Store.PushUserReaction(chatID, targetID, emoji)
	m.refreshViewport()
}

// --- rendering ---

func (m *Model) renderReactionPicker(width int) string {
	// In custom-emoji submode, replace the picker strip with the text field.
	if m.emojiInputMode {
		label := lipgloss.NewStyle().
			Foreground(m.theme.PromptColor).
			Bold(true).
			Render("emoji › ")
		field := lipgloss.NewStyle().
			Foreground(m.theme.InputColor).
			Render(m.emojiInput.View())
		hint := lipgloss.NewStyle().
			Foreground(m.theme.SystemColor).
			Render("   Enter submit · Esc back")
		return lipgloss.NewStyle().MaxWidth(width).Render(label + field + hint)
	}

	var parts []string
	for i, emoji := range reactionPicks {
		label := fmt.Sprintf(" %d %s ", i+1, emoji)
		var st lipgloss.Style
		if i == m.reactionIdx {
			st = lipgloss.NewStyle().
				Background(m.theme.SuggestionSelectedBG).
				Foreground(m.theme.InputColor).
				Bold(true)
		} else {
			st = lipgloss.NewStyle().Foreground(m.theme.InputColor)
		}
		cell := zone.Mark(ZoneReactionPrefix+emoji, st.Render(label))
		parts = append(parts, cell)
	}
	// Trailing "other" cell — clickable, keyboard-reachable via digit 7 or
	// arrow-right past the last emoji.
	otherLabel := fmt.Sprintf(" %d other ", reactionOtherIdx+1)
	var otherStyle lipgloss.Style
	if m.reactionIdx == reactionOtherIdx {
		otherStyle = lipgloss.NewStyle().
			Background(m.theme.SuggestionSelectedBG).
			Foreground(m.theme.InputColor).
			Bold(true)
	} else {
		otherStyle = lipgloss.NewStyle().Foreground(m.theme.InputColor)
	}
	parts = append(parts, zone.Mark(ZoneReactionOther, otherStyle.Render(otherLabel)))

	strip := strings.Join(parts, " ")
	hint := lipgloss.NewStyle().Foreground(m.theme.SystemColor).Render("  Enter pick · Esc cancel")
	return lipgloss.NewStyle().MaxWidth(width).Render(strip + hint)
}
