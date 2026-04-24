package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/photon-hq/tuichat/internal/protocol"
	"github.com/photon-hq/tuichat/internal/store"
)

// handleSubmit fires on Enter in the chat-input field. Handles `/new` and
// `/help` intercepts, then ships any pending attachments + the typed text
// as store appends and protocol notifications. If a reply is pending
// (m.replyingTo), the first outgoing item carries the reply-quote target.
func (m *Model) handleSubmit() (tea.Model, tea.Cmd) {
	id := m.Store.ActiveChatID()
	if id == "" || id == store.SystemChatID {
		return m, nil
	}
	raw := strings.TrimSpace(m.input.Value())

	if raw == "/new" {
		m.Store.SetInputDraft(id, "")
		m.Store.NewChat()
		m.resetInput()
		m.refreshViewport()
		return m, nil
	}

	if raw == "/help" {
		for _, line := range helpLines {
			m.Store.AppendSystem(id, line)
		}
		m.Store.SetInputDraft(id, "")
		m.resetInput()
		m.refreshViewport()
		return m, nil
	}

	pending := m.Store.ClearPendingAttachments(id)
	if raw == "" && len(pending) == 0 {
		return m, nil
	}

	replyTo := m.replyingTo
	m.replyingTo = ""

	for _, a := range pending {
		content := protocol.Content{
			Type:     "attachment",
			Name:     a.Name,
			MimeType: guessMime(a.Name),
			Path:     a.Path,
		}
		if a.Size > 0 {
			size := a.Size
			content.Size = &size
		}
		if replyTo != "" {
			msgID := m.Store.AppendUserReply(id, content, replyTo, a.Path)
			m.Store.PushUserReply(id, msgID, content, replyTo)
			replyTo = "" // only the first outgoing carries the reply quote
		} else {
			msgID := m.Store.AppendUser(id, content, a.Path)
			m.Store.PushUserInput(id, msgID, content)
		}
	}
	if raw != "" {
		content := protocol.Content{Type: "text", Text: raw}
		if replyTo != "" {
			msgID := m.Store.AppendUserReply(id, content, replyTo, "")
			m.Store.PushUserReply(id, msgID, content, replyTo)
		} else {
			msgID := m.Store.AppendUser(id, content, "")
			m.Store.PushUserInput(id, msgID, content)
		}
	}

	m.Store.SetInputDraft(id, "")
	m.resetInput()
	m.refreshViewport()
	return m, nil
}

// resetInput clears the input field and completion state. Used after submit
// and after slash-command intercepts.
func (m *Model) resetInput() {
	m.input.SetValue("")
	m.prefix = ""
	m.tabIndex = 0
}
