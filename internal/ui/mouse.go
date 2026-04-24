package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/photon-hq/tuichat/internal/kitty"
	"github.com/photon-hq/tuichat/internal/store"
)

// handleMouse is the mouse-event entry point. It classifies the event into
// one of four flows:
//
//  1. Drag-in-progress — motion/release route to the drag handlers
//     regardless of the reported Button (terminals often drop Button to
//     None during held-drag motion).
//  2. Mouse wheel — forwards to the viewport under the cursor (chat, log
//     panel, or sidebar) so each pane scrolls independently.
//  3. Left press inside a drag-capable pane — starts a new drag.
//  4. Everything else — falls through to the zone cascade.
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.dragInProgress {
		switch msg.Action {
		case tea.MouseActionMotion:
			return m.handleDragMotion(msg)
		case tea.MouseActionRelease:
			return m.handleDragRelease(msg)
		}
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		return m.handleWheel(msg)

	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}

		// Press inside the chat-column viewport starts a chat-pane drag.
		if m.inMessageLogRect(msg.X, msg.Y) {
			m.startDrag(msg, paneChat,
				m.screenRowToContentRow(msg.Y),
				m.screenColToContentCol(msg.X))
			return m, nil
		}

		// Press inside the log-column body starts a system-log drag.
		if m.inSystemLogRect(msg.X, msg.Y) {
			m.startDrag(msg, paneSystemLog,
				m.screenRowToSystemLogRow(msg.Y),
				m.screenColToSystemLogCol(msg.X))
			return m, nil
		}

		// Press outside any drag-capable region clears any persisted
		// selection and falls through to the zone cascade.
		if m.dragActive {
			m.clearDragSelection()
			m.refreshViewport()
		}
		return m.dispatchClick(msg)
	}
	return m, nil
}

// handleWheel forwards a mouse-wheel event to the viewport under the cursor
// — chat column, system-log panel body, or sidebar list — so each pane
// scrolls independently. Wheel events outside any scrollable pane are a
// no-op.
func (m *Model) handleWheel(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.inMessageLogRect(msg.X, msg.Y):
		m.log, _ = m.log.Update(msg)
	case m.inSystemLogRect(msg.X, msg.Y):
		m.logPanel, _ = m.logPanel.Update(msg)
	case m.inSidebarRect(msg.X, msg.Y):
		m.sidebar, _ = m.sidebar.Update(msg)
	}
	return m, nil
}

// inSidebarRect returns true if (x,y) sits over the scrollable middle of
// the sidebar — between the "Chats" header row and the hint rows at the
// bottom. Used to route wheel events to the sidebar viewport.
func (m *Model) inSidebarRect(x, y int) bool {
	if x < 0 || x >= SidebarWidth {
		return false
	}
	if y < 1 {
		return false
	}
	if y >= 1+m.sidebar.Height {
		return false
	}
	return true
}

// startDrag initializes drag-select state for a fresh press. Clearing a
// prior persisted selection (and re-rendering to remove its highlight)
// happens here first; the press might still turn into a simple click, in
// which case handleDragRelease replays it through dispatchClick.
func (m *Model) startDrag(msg tea.MouseMsg, pane dragPane, row, col int) {
	if m.dragActive {
		m.clearDragSelection()
		m.refreshViewport()
	}
	m.dragInProgress = true
	m.dragMoved = false
	m.dragPane = pane
	m.dragPressX = msg.X
	m.dragPressY = msg.Y
	m.dragStartRow = row
	m.dragStartCol = col
	m.dragEndRow = row
	m.dragEndCol = col
}

// dispatchClick runs the zone-match cascade against the given press
// MouseMsg. Called directly on non-drag-rect presses and also on release-
// without-motion (by handleDragRelease) so click-to-expand, click-to-
// select-message, etc. keep working.
func (m *Model) dispatchClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Toggle log panel.
	if zone.Get(ZoneLogToggle).InBounds(msg) {
		m.logVisible = !m.logVisible
		m.layoutInner()
		return m, nil
	}

	// Toggle expand/collapse of a specific log entry.
	if m.logVisible {
		for _, e := range m.Store.SystemEntries() {
			if zone.Get(ZoneLogEntryPrefix + e.ID).InBounds(msg) {
				if m.expandedLogs[e.ID] {
					delete(m.expandedLogs, e.ID)
				} else {
					m.expandedLogs[e.ID] = true
				}
				return m, nil
			}
		}
	}

	// Sidebar clicks → switch active chat.
	for _, chat := range m.Store.SortedChats() {
		zoneID := ZoneSidebarRowPrefix + chat.ID
		if zone.Get(zoneID).InBounds(msg) {
			m.clearModalState()
			m.Store.SetActiveChat(chat.ID)
			m.syncInputFromDraft()
			m.refreshViewport()
			return m, nil
		}
	}

	// Reaction picker cells (only meaningful while reacting).
	if m.reactingTo != "" {
		for _, emoji := range reactionPicks {
			if zone.Get(ZoneReactionPrefix + emoji).InBounds(msg) {
				m.submitReaction(emoji)
				return m, nil
			}
		}
		if zone.Get(ZoneReactionOther).InBounds(msg) {
			m.enterEmojiInputMode()
			return m, textinput.Blink
		}
	}

	// Action row buttons on the currently-selected message.
	if m.selecting && m.selectedID != "" {
		if zone.Get(ZoneReplyButton).InBounds(msg) {
			m.replyingTo = m.selectedID
			m.exitSelectMode()
			m.refreshViewport()
			return m, nil
		}
		if zone.Get(ZoneReactButton).InBounds(msg) {
			m.reactingTo = m.selectedID
			m.reactionIdx = 0
			m.selecting = false
			m.refreshViewport()
			return m, nil
		}
	}

	// Attachment chip clicks → toggle preview.
	if active, ok := m.Store.ActiveChat(); ok {
		for i := range active.Entries {
			e := active.Entries[i]
			if e.Content.Type != "attachment" || !kitty.SupportedMimeType(e.Content.MimeType) {
				continue
			}
			zoneID := ZoneAttachmentPrefix + e.ID
			if !zone.Get(zoneID).InBounds(msg) {
				continue
			}
			if hovered := m.Store.HoveredPreview(); hovered != nil && hovered.CacheKey == e.Content.Name {
				m.Store.SetHoveredPreview(nil)
			} else {
				m.Store.SetHoveredPreview(&store.HoveredPreview{
					CacheKey: e.Content.Name,
					Name:     e.Content.Name,
					Path:     e.AttachmentPath,
				})
			}
			return m, nil
		}
	}

	// Message body clicks → select that message (or toggle off if it was
	// already the selection). System chat is read-only — no selection.
	if active, ok := m.Store.ActiveChat(); ok && active.ID != store.SystemChatID {
		for i := range active.Entries {
			e := active.Entries[i]
			if !zone.Get(ZoneMessagePrefix + e.ID).InBounds(msg) {
				continue
			}
			if m.selecting && m.selectedID == e.ID {
				m.exitSelectMode()
			} else {
				m.selecting = true
				m.selectedID = e.ID
			}
			m.refreshViewport()
			return m, nil
		}
	}

	return m, nil
}
