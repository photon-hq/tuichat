package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/photon-hq/tuichat/internal/drop"
	"github.com/photon-hq/tuichat/internal/store"
)

// Keyboard input routing, paste handling, and chat-input text bookkeeping
// (draft preservation, slash-completion). Chat submission, reaction, and
// message-select flows live in their own files.

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Reaction picker takes highest priority — Cmd+V inside the emoji text
	// field would otherwise leak into the chat input below.
	if m.reactingTo != "" {
		if m.emojiInputMode {
			return m.handleEmojiInputKey(msg, key)
		}
		// Picker strip doesn't consume pastes.
		if msg.Paste {
			return m, nil
		}
		return m.handleReactionKey(msg, key)
	}

	// Bracketed-paste events come through as a KeyMsg with Paste=true.
	if msg.Paste {
		return m.handlePaste(string(msg.Runes))
	}

	// Always-available shortcuts.
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+n":
		m.clearModalState()
		m.Store.NewChat()
		m.syncInputFromDraft()
		m.refreshViewport()
		return m, nil
	case "ctrl+j":
		m.clearModalState()
		m.Store.CycleActiveChat(1)
		m.syncInputFromDraft()
		m.refreshViewport()
		return m, nil
	case "ctrl+k":
		m.clearModalState()
		m.Store.CycleActiveChat(-1)
		m.syncInputFromDraft()
		m.refreshViewport()
		return m, nil
	case "ctrl+l":
		if id := m.Store.ActiveChatID(); id != "" && id != store.SystemChatID {
			m.Store.AppendSystem(id, "screen cleared")
			m.refreshViewport()
		}
		return m, nil
	case "ctrl+`":
		m.logVisible = !m.logVisible
		m.layoutInner()
		return m, nil
	case "tab":
		m.cycleSlashCompletion()
		return m, nil
	case "up":
		if m.canSelectMessages() {
			m.moveSelection(-1)
			return m, nil
		}
	case "down":
		if m.canSelectMessages() {
			m.moveSelection(1)
			return m, nil
		}
	case "r":
		if m.selecting && m.selectedID != "" {
			m.replyingTo = m.selectedID
			m.exitSelectMode()
			m.refreshViewport()
			return m, nil
		}
	case "e":
		if m.selecting && m.selectedID != "" {
			m.reactingTo = m.selectedID
			m.reactionIdx = 0
			m.selecting = false
			m.refreshViewport()
			return m, nil
		}
	case "esc":
		if m.selecting || m.replyingTo != "" {
			m.exitSelectMode()
			m.replyingTo = ""
			m.refreshViewport()
			return m, nil
		}
		if id := m.Store.ActiveChatID(); id != "" {
			m.Store.ClearPendingAttachments(id)
			m.Store.SetInputDraft(id, "")
		}
		m.input.SetValue("")
		m.prefix = ""
		m.tabIndex = 0
		return m, nil
	case "enter":
		return m.handleSubmit()
	}

	// Selection is strictly a read-only mode; typing any other key exits it
	// back to normal editing rather than silently swallowing keystrokes.
	if m.selecting {
		m.exitSelectMode()
		m.refreshViewport()
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.onInputChange()
	return m, cmd
}

// handlePaste turns bracketed-paste events into either a pending-attachment
// (if the pasted content is a filesystem path) or regular input text with
// any quoted paths extracted. System chat never accepts pasted content.
func (m *Model) handlePaste(raw string) (tea.Model, tea.Cmd) {
	id := m.Store.ActiveChatID()
	if id == "" || id == store.SystemChatID {
		return m, nil
	}
	path := drop.ParsePath(raw)
	if path != "" {
		if info := statSafe(path); info != nil {
			m.Store.AddPendingAttachment(id, store.PendingAttachment{
				Path: path,
				Name: fileBase(path),
				Size: info.Size(),
			})
			m.refreshViewport()
			return m, nil
		}
	}
	// Not a file path. Append to the input value and also scan for embedded
	// quoted paths (VS Code / older terminals that pass drag-drop as raw
	// keystrokes).
	current := m.input.Value() + raw
	ex := drop.Extract(current)
	for _, p := range ex.Paths {
		m.Store.AddPendingAttachment(id, store.PendingAttachment{
			Path: p.Path,
			Name: p.Name,
			Size: p.Size,
		})
	}
	m.input.SetValue(ex.Cleaned)
	m.onInputChange()
	m.refreshViewport()
	return m, nil
}

// onInputChange runs after every text-input update — saves the draft to the
// active chat, extracts quoted filesystem paths as pending attachments, and
// resets slash-completion cycling.
func (m *Model) onInputChange() {
	id := m.Store.ActiveChatID()

	// Fallback for terminals that deliver drag-drop as raw keystrokes rather
	// than bracketed paste: scan the current input for quoted paths that
	// resolve to real files and promote them to pending attachments.
	if id != "" && id != store.SystemChatID {
		if ex := drop.Extract(m.input.Value()); len(ex.Paths) > 0 {
			for _, p := range ex.Paths {
				m.Store.AddPendingAttachment(id, store.PendingAttachment{
					Path: p.Path,
					Name: p.Name,
					Size: p.Size,
				})
			}
			m.input.SetValue(ex.Cleaned)
		}
	}

	m.prefix = m.input.Value()
	m.tabIndex = 0
	if id != "" {
		m.Store.SetInputDraft(id, m.prefix)
	}
}

// syncInputFromDraft restores the per-chat input draft when switching chats.
func (m *Model) syncInputFromDraft() {
	chat, ok := m.Store.ActiveChat()
	if !ok {
		m.input.SetValue("")
		m.prefix = ""
		return
	}
	m.input.SetValue(chat.InputDraft)
	m.prefix = chat.InputDraft
	m.tabIndex = 0
}

// cycleSlashCompletion cycles through slash-command autocompletion matches
// on each Tab press.
func (m *Model) cycleSlashCompletion() {
	matches := FilterCommands(m.Store.Commands(), m.prefix)
	if len(matches) == 0 {
		return
	}
	pick := matches[m.tabIndex%len(matches)]
	m.input.SetValue(pick.Name)
	m.tabIndex++
}
