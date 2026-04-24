package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone"
	"github.com/mattn/go-runewidth"

	"github.com/photon-hq/tuichat/internal/kitty"
	"github.com/photon-hq/tuichat/internal/store"
)

// View rendering root + the per-region renderers that compose the frame:
// title bar, input container, chat viewport refresh, message-entries zone
// marking, and the small bg-reapply helper that keeps the selection fill
// connected across styled sub-segments.

// View renders the full frame. Composition:
//
//	[sidebar] [title  ──── log header ] [×]
//	          [messages][rule          ]
//	          [  ...    ][log body     ]
//	          [typing   ]
//	          [input spans chat + log column ]
//
// Floating preview + copy-confirmation toast overlay onto the frame last.
func (m *Model) View() string {
	if !m.ready {
		return ""
	}
	chat, hasActive := m.Store.ActiveChat()
	chats := m.Store.SortedChats()

	sidebar := zoneMarkSidebar(m.theme, chats, m.Store.ActiveChatID(), m.height)

	activeID := ""
	if hasActive {
		activeID = chat.ID
	}
	titleBar := m.renderTitleBar(activeID)

	typingLine := ""
	if hasActive && chat.Typing {
		typingLine = lipgloss.NewStyle().Foreground(m.theme.TypingColor).Render(" ● agent is typing…")
	}

	inputContainer := m.renderInputContainer(chat, hasActive)

	chatCol := lipgloss.JoinVertical(lipgloss.Left, titleBar, m.log.View(), typingLine)

	// Top half: chat column plus, when enabled, the system-log column on the
	// right. The input container spans both below.
	var topRow string
	if m.logVisible {
		topHeight := 1 /*title*/ + m.log.Height + 1 /*typing*/
		logCol := m.renderLogColumn(topHeight)
		topRow = lipgloss.JoinHorizontal(lipgloss.Top, chatCol, logCol)
	} else {
		topRow = chatCol
	}

	rightCol := lipgloss.JoinVertical(lipgloss.Left, topRow, inputContainer)
	frame := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightCol)

	// Floating preview overlay (top-right).
	if preview := m.Store.HoveredPreview(); preview != nil {
		if panel := m.renderPreviewPanel(preview); panel != "" {
			frame = overlayAt(frame, panel, m.width-PreviewCols-6, 1)
		}
	}

	// Resolve bubblezone markers first, then let bubbleup paint its alert
	// overlay. The alert is a full-frame overlay that leaves layout intact.
	return m.alert.Render(zone.Scan(frame))
}

// refreshViewport rebuilds the chat viewport content from the active chat,
// captures a plain-text mirror for drag-select extraction, and overlays
// reverse-video when a chat-pane drag-selection is active.
func (m *Model) refreshViewport() {
	chat, ok := m.Store.ActiveChat()
	if !ok {
		m.log.SetContent("")
		m.plainLogLines = nil
		return
	}
	inner := m.logInnerWidth()
	content := m.zoneMarkEntries(m.theme, chat, inner)

	// Bottom-align: if content fits in the viewport, prepend blank lines so
	// the newest message sits at the bottom next to the input.
	if content != "" && m.log.Height > 0 {
		lines := strings.Count(content, "\n") + 1
		if lines < m.log.Height {
			content = strings.Repeat("\n", m.log.Height-lines) + content
		}
	}

	// Capture a plain-text mirror for drag-select extraction. Bubblezone
	// markers are CSI-like sequences (\x1b[<n>z) that ansi.Strip removes
	// alongside styling escapes, giving us rows that align 1:1 with the
	// viewport's.
	m.plainLogLines = strings.Split(ansi.Strip(content), "\n")

	// Overlay reverse-video on the selected span when a chat-pane drag is
	// active. Preserve styling outside the selection via ansi.Cut; the
	// selected span itself renders plain + reverse (original colors are lost
	// during the drag, restored on clear). System-log drags are handled
	// separately inside renderLogColumn.
	if m.dragActive && m.dragPane == paneChat {
		styledLines := strings.Split(content, "\n")
		loRow, loCol, hiRow, hiCol := m.selectionRange()
		if loRow < 0 {
			loRow = 0
		}
		if hiRow >= len(styledLines) {
			hiRow = len(styledLines) - 1
		}
		reverse := lipgloss.NewStyle().Reverse(true)
		for r := loRow; r <= hiRow && r < len(m.plainLogLines); r++ {
			plain := m.plainLogLines[r]
			lineW := runewidth.StringWidth(plain)
			from, to := 0, lineW
			if r == loRow {
				from = loCol
			}
			if r == hiRow {
				to = hiCol
			}
			if from > lineW {
				from = lineW
			}
			if to > lineW {
				to = lineW
			}
			if from >= to {
				continue
			}
			left := ansi.Cut(styledLines[r], 0, from)
			mid := reverse.Render(plainSliceByCols(plain, from, to))
			right := ansi.Cut(styledLines[r], to, lineW)
			styledLines[r] = left + mid + right
		}
		content = strings.Join(styledLines, "\n")
	}

	m.log.SetContent(content)
	m.log.GotoBottom()
}

// renderTitleBar paints the top bar over the chat column. When the log panel
// is hidden we extend rightward so the `[log ▸]` toggle stays reachable;
// when open, the log column has its own `[×]` and the title bar stops at
// the chat column edge.
func (m *Model) renderTitleBar(activeID string) string {
	rightWidth := m.chatAreaWidth()
	if !m.logVisible {
		rightWidth = m.width - SidebarWidth - 1 // -1 for sidebar border
	}
	if rightWidth <= 0 {
		return ""
	}

	title := " " + m.theme.Title
	if activeID != "" {
		title += " · " + activeID
	}
	title += " "
	rest := " — Ctrl+N new · Ctrl+J/K nav · Ctrl+C exit · Ctrl+L clear · Tab complete"

	titleStyle := lipgloss.NewStyle().
		Foreground(m.theme.UserColor).
		Background(m.theme.BorderColor)
	subStyle := lipgloss.NewStyle().
		Foreground(m.theme.SystemColor).
		Background(m.theme.BorderColor)
	btnStyle := lipgloss.NewStyle().
		Foreground(m.theme.PromptColor).
		Background(m.theme.BorderColor)

	// Only show the title-bar toggle when the log is hidden.
	var btnText string
	if !m.logVisible {
		btnText = " [log ▸] "
	}
	btnW := runewidth.StringWidth(btnText)

	// Pre-truncate by visible width so the terminal doesn't hard-wrap.
	titleW := runewidth.StringWidth(title)
	if titleW+btnW > rightWidth {
		title = runewidth.Truncate(title, rightWidth-btnW, "")
		rest = ""
	} else {
		avail := rightWidth - titleW - btnW
		if runewidth.StringWidth(rest) > avail {
			rest = runewidth.Truncate(rest, avail, "")
		}
	}
	titleW = runewidth.StringWidth(title)
	restW := runewidth.StringWidth(rest)
	gap := rightWidth - titleW - restW - btnW
	if gap < 0 {
		gap = 0
	}

	rendered := titleStyle.Render(title) + subStyle.Render(rest) +
		subStyle.Render(strings.Repeat(" ", gap))
	if btnText != "" {
		rendered += zone.Mark(ZoneLogToggle, btnStyle.Render(btnText))
	}

	return lipgloss.NewStyle().
		Background(m.theme.BorderColor).
		Width(rightWidth).
		MaxWidth(rightWidth).
		Render(rendered)
}

// renderInputContainer builds the bottom input area. Stacks (in order) a
// reply banner, reaction picker, attachment chips, slash-command
// suggestions, and finally the text input itself — inside a single
// rounded-border box spanning the full width under the sidebar.
func (m *Model) renderInputContainer(chat store.ChatState, hasActive bool) string {
	innerWidth := m.inputWidth() - 2 /*padding*/
	if innerWidth < 10 {
		innerWidth = 10
	}

	var rows []string

	if hasActive && m.replyingTo != "" {
		if banner := m.renderReplyBanner(chat, innerWidth); banner != "" {
			rows = append(rows, banner)
		}
	}
	if hasActive && m.reactingTo != "" {
		rows = append(rows, m.renderReactionPicker(innerWidth))
	}

	if hasActive && len(chat.PendingAttachments) > 0 {
		chips := RenderAttachmentChips(m.theme, chat.PendingAttachments, innerWidth)
		rows = append(rows, chips)
	}

	matches := FilterCommands(m.Store.Commands(), m.prefix)
	if strings.HasPrefix(m.prefix, "/") && len(matches) > 0 {
		panel := RenderSuggestions(m.theme, matches, m.tabIndex, innerWidth)
		if panel != "" {
			rows = append(rows, panel)
		}
	}

	inputStyle := lipgloss.NewStyle().Foreground(m.theme.InputColor)
	rows = append(rows, inputStyle.Render(m.input.View()))

	inner := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderColor).
		Padding(0, 1).
		Width(m.inputWidth()).
		Render(inner)
}

// zoneMarkEntries wraps each rendered chat entry with its click-zone and
// handles the selected-entry treatment (full-width SuggestionBG fill +
// inline action row). A dim-color rule is inserted between every pair of
// entries to give them breathing room without a blank blank-line spacer.
func (m *Model) zoneMarkEntries(theme Theme, chat store.ChatState, width int) string {
	blocks := make([]string, 0, len(chat.Entries)+1)
	if chat.DroppedCount > 0 {
		msg := fmt.Sprintf("… %d older messages dropped", chat.DroppedCount)
		blocks = append(blocks, lipgloss.NewStyle().Foreground(theme.SystemColor).Render(msg))
	}
	selectedBG := lipgloss.NewStyle().
		Background(theme.SuggestionBG).
		Width(m.chatAreaWidth())
	// Opening escape for SuggestionBG — re-injected after every inner
	// \x1b[0m reset inside already-styled entries so the fill doesn't drop
	// out when a child segment (timestamp, role label, body) finishes
	// rendering.
	bgOpen := extractBgOpenSeq(theme.SuggestionBG)
	for i := range chat.Entries {
		e := chat.Entries[i]
		rendered := renderEntry(theme, e, chat.Entries, width)
		selected := e.ID == m.selectedID && m.selecting
		if selected {
			rendered = strings.ReplaceAll(rendered, "\x1b[0m", "\x1b[0m"+bgOpen)
			rendered = selectedBG.Render(rendered)
		}
		if e.Content.Type == "attachment" && kitty.SupportedMimeType(e.Content.MimeType) {
			rendered = zone.Mark(ZoneAttachmentPrefix+e.ID, rendered)
		}
		// Every entry gets a click-zone so mouse-select works.
		rendered = zone.Mark(ZoneMessagePrefix+e.ID, rendered)
		// The action row stays attached to the selected entry (no blank
		// gap) so it reads as one continuous highlighted section. The same
		// bg-reapply trick is applied to keep button gaps + hint text on
		// the fill.
		if selected {
			row := strings.ReplaceAll(renderActionRow(theme), "\x1b[0m", "\x1b[0m"+bgOpen)
			rendered = rendered + "\n" + selectedBG.Render(row)
		}
		blocks = append(blocks, rendered)
	}
	// Solid dim-color rule between entries — spans the chat column so it
	// runs edge-to-edge against the log-column border.
	ruleWidth := m.chatAreaWidth()
	if ruleWidth < 10 {
		ruleWidth = 10
	}
	rule := lipgloss.NewStyle().Foreground(theme.TimestampColor).Render(
		strings.Repeat("─", ruleWidth))
	sep := "\n" + rule + "\n"
	return strings.Join(blocks, sep)
}

// extractBgOpenSeq returns the ANSI escape prefix lipgloss emits for a given
// background color — the bytes up to (but not including) the first content
// character. Used to re-apply the outer background after inner \x1b[0m
// resets, since lipgloss doesn't patch those itself.
func extractBgOpenSeq(color lipgloss.Color) string {
	const sentinel = "\x01"
	sample := lipgloss.NewStyle().Background(color).Render(sentinel)
	if idx := strings.Index(sample, sentinel); idx > 0 {
		return sample[:idx]
	}
	return ""
}
