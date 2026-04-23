// Package ui is the Bubbletea UI for tuichat. The Model owns the Store and
// delegates rendering to the sibling files (sidebar.go, messagelog.go, etc.).
package ui

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/mosaic"
	zone "github.com/lrstanley/bubblezone"
	"github.com/mattn/go-runewidth"

	"github.com/photon-hq/tuichat/internal/drop"
	"github.com/photon-hq/tuichat/internal/kitty"
	"github.com/photon-hq/tuichat/internal/protocol"
	"github.com/photon-hq/tuichat/internal/store"
)

// Zone IDs used with bubblezone for click/hover routing.
const (
	ZoneSidebarRowPrefix = "sidebar-row:"
	ZoneAttachmentPrefix = "attachment:" // per-log-entry image chip
	ZoneMessagePrefix    = "msg:"        // click a message to select it
	ZoneReplyButton      = "action-reply"
	ZoneReactButton      = "action-react"
	ZoneReactionPrefix   = "reaction:" // each quick-pick emoji in the picker
	ZoneLogToggle        = "log-toggle"
	ZoneLogEntryPrefix   = "log-entry:" // click a log line to expand/collapse
)

// LogColumnWidth is the fixed width (in cells) of the right-hand system log
// panel. Includes a 1-cell border on the left.
const LogColumnWidth = 40

// reactionPicks is the fixed set of quick-pick reactions shown in the picker.
var reactionPicks = []string{"👍", "❤️", "😂", "😮", "😢", "🎉"}

// StoreChangedMsg is sent whenever the RPC server mutates the store and the UI
// needs to re-render. The server's pump goroutine sends these via Program.Send.
type StoreChangedMsg struct{}

// helpLines are dumped into the active chat when user types `/help`.
var helpLines = []string{
	"tuichat — keybindings",
	"  Ctrl+N         new chat",
	"  Ctrl+J / K     cycle chats (down / up)",
	"  Ctrl+L         clear active chat",
	"  Ctrl+`         toggle system-log panel",
	"  Ctrl+C         exit",
	"  Tab            complete slash command",
	"  Esc            cancel input / exit select / drop attachments",
	"  ↑ / ↓          (empty input) select a message to reply/react",
	"  r              reply to selected message",
	"  e              react to selected message (emoji picker)",
	"  1-6 / ←→ Enter emoji picker: pick / move / confirm",
	"  drag file      attach (or paste its path)",
	"  click message  select (then ↩ reply or 🙂 react)",
	"  click image    toggle floating preview (Kitty/Ghostty)",
	"slash commands",
	"  /new           start a new chat",
	"  /help          this message",
	"environment variables",
	"  TUICHAT_DISABLE_IMAGES=1   disable Kitty graphics image previews",
	"  TUICHAT_DEBUG_IMAGES=1     log APC sequences to /tmp/tuichat-images.log",
}

// Model is the top-level Bubbletea model.
type Model struct {
	Store     *store.Store
	theme     Theme
	input     textinput.Model
	log       viewport.Model
	width     int
	height    int
	prefix    string
	tabIndex  int
	ready     bool

	// Message-select / action state — per active chat. The UI enters select
	// mode on ↑/↓ or message click, lets the user target one entry, then
	// either ↩ replies, 🙂 reacts, or Esc cancels.
	selecting   bool
	selectedID  string
	replyingTo  string // non-empty → next submit is a quoted reply
	reactingTo  string // non-empty → emoji picker open for this entry
	reactionIdx int

	// Right-hand system log panel. Open by default; toggled via a title-bar
	// button or Ctrl+Backtick.
	logVisible   bool
	expandedLogs map[string]bool // log entry IDs currently shown fully-wrapped
}

// NewModel builds an initialized Model. Caller is expected to wire a RPC server
// that holds the same store pointer.
func NewModel(s *store.Store) *Model {
	in := textinput.New()
	in.Placeholder = "type a message and press enter…"
	in.Prompt = ""
	in.Focus()
	in.CharLimit = 10000

	vp := viewport.New(80, 20)
	return &Model{
		Store:        s,
		theme:        DefaultTheme,
		input:        in,
		log:          vp,
		logVisible:   true,
		expandedLogs: map[string]bool{},
	}
}

func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles Bubbletea messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layoutInner()
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case StoreChangedMsg:
		m.refreshViewport()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.onInputChange()
	return m, cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Bracketed-paste events come through as a KeyMsg with Paste=true.
	if msg.Paste {
		return m.handlePaste(string(msg.Runes))
	}

	key := msg.String()

	// Reaction picker has priority — it intercepts digits/arrows/enter/esc.
	if m.reactingTo != "" {
		return m.handleReactionKey(key)
	}

	// Always-available shortcuts
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+n":
		m.clearModalState()
		id := m.Store.NewChat()
		_ = id
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

func (m *Model) handleReactionKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.reactingTo = ""
		m.reactionIdx = 0
		m.refreshViewport()
		return m, nil
	case "left":
		if m.reactionIdx > 0 {
			m.reactionIdx--
		}
		return m, nil
	case "right":
		if m.reactionIdx < len(reactionPicks)-1 {
			m.reactionIdx++
		}
		return m, nil
	case "enter":
		m.submitReaction(reactionPicks[m.reactionIdx])
		return m, nil
	}
	if len(key) == 1 && key >= "1" && key <= "6" {
		idx := int(key[0] - '1')
		if idx < len(reactionPicks) {
			m.submitReaction(reactionPicks[idx])
		}
	}
	return m, nil
}

func (m *Model) submitReaction(emoji string) {
	chatID := m.Store.ActiveChatID()
	targetID := m.reactingTo
	m.reactingTo = ""
	m.reactionIdx = 0
	if chatID == "" || targetID == "" {
		m.refreshViewport()
		return
	}
	m.Store.React(chatID, targetID, emoji)
	m.Store.PushUserReaction(chatID, targetID, emoji)
	m.refreshViewport()
}

func (m *Model) canSelectMessages() bool {
	chat, ok := m.Store.ActiveChat()
	if !ok {
		return false
	}
	if chat.ID == store.SystemChatID || len(chat.Entries) == 0 {
		return false
	}
	// When the user is typing, arrow keys should stay with the textinput for
	// cursor movement. Only hijack them when the input is empty.
	if !m.selecting && m.input.Value() != "" {
		return false
	}
	return true
}

// moveSelection activates select mode on first press, then walks through
// message IDs in the active chat. Direction: -1 up (older), +1 down (newer).
func (m *Model) moveSelection(delta int) {
	chat, ok := m.Store.ActiveChat()
	if !ok || len(chat.Entries) == 0 {
		return
	}
	if !m.selecting {
		m.selecting = true
		// First press selects the most recent message regardless of direction.
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

// clearModalState drops any cross-message modes (select, reply, react). Used
// when switching chats — the targets only make sense in their original chat.
func (m *Model) clearModalState() {
	m.selecting = false
	m.selectedID = ""
	m.replyingTo = ""
	m.reactingTo = ""
	m.reactionIdx = 0
}

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
	// Not a file path; just append to input value, also scan for embedded quoted paths.
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

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
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
	}
	return m, nil
}

func (m *Model) handleSubmit() (tea.Model, tea.Cmd) {
	id := m.Store.ActiveChatID()
	if id == "" || id == store.SystemChatID {
		return m, nil
	}
	raw := strings.TrimSpace(m.input.Value())

	if raw == "/new" {
		m.Store.SetInputDraft(id, "")
		m.Store.NewChat()
		m.input.SetValue("")
		m.prefix = ""
		m.tabIndex = 0
		m.refreshViewport()
		return m, nil
	}

	if raw == "/help" {
		for _, line := range helpLines {
			m.Store.AppendSystem(id, line)
		}
		m.Store.SetInputDraft(id, "")
		m.input.SetValue("")
		m.prefix = ""
		m.tabIndex = 0
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
		var msgID string
		if replyTo != "" {
			msgID = m.Store.AppendUserReply(id, content, replyTo, a.Path)
			m.Store.PushUserReply(id, msgID, content, replyTo)
			replyTo = "" // only first outgoing carries the reply quote
		} else {
			msgID = m.Store.AppendUser(id, content, a.Path)
			m.Store.PushUserInput(id, msgID, content)
		}
	}
	if raw != "" {
		content := protocol.Content{Type: "text", Text: raw}
		var msgID string
		if replyTo != "" {
			msgID = m.Store.AppendUserReply(id, content, replyTo, "")
			m.Store.PushUserReply(id, msgID, content, replyTo)
		} else {
			msgID = m.Store.AppendUser(id, content, "")
			m.Store.PushUserInput(id, msgID, content)
		}
		_ = msgID
	}

	m.Store.SetInputDraft(id, "")
	m.input.SetValue("")
	m.prefix = ""
	m.tabIndex = 0
	m.refreshViewport()
	return m, nil
}

func (m *Model) onInputChange() {
	id := m.Store.ActiveChatID()

	// Fallback for terminals that deliver drag-drop as raw keystrokes rather
	// than bracketed paste (VS Code, some older terminals): scan the current
	// input for quoted paths that resolve to real files and promote them to
	// pending attachments.
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

func (m *Model) cycleSlashCompletion() {
	matches := FilterCommands(m.Store.Commands(), m.prefix)
	if len(matches) == 0 {
		return
	}
	pick := matches[m.tabIndex%len(matches)]
	m.input.SetValue(pick.Name)
	m.tabIndex++
}

func (m *Model) refreshViewport() {
	chat, ok := m.Store.ActiveChat()
	if !ok {
		m.log.SetContent("")
		return
	}
	inner := m.logInnerWidth()
	content := m.zoneMarkEntries(m.theme, chat, inner)
	// Bottom-align: if content fits in the viewport, prepend blank lines so
	// the newest message sits at the bottom next to the input, with empty
	// space above rather than below.
	if content != "" && m.log.Height > 0 {
		lines := strings.Count(content, "\n") + 1
		if lines < m.log.Height {
			content = strings.Repeat("\n", m.log.Height-lines) + content
		}
	}
	m.log.SetContent(content)
	m.log.GotoBottom()
}

// chatAreaWidth is the width of the chat column (messages + typing line).
// Shrinks when the right-hand log panel is visible.
func (m *Model) chatAreaWidth() int {
	w := m.width - SidebarWidth
	if m.logVisible {
		w -= LogColumnWidth
	}
	if w < 20 {
		w = 20
	}
	return w
}

// inputWidth is the width of the input container, which always spans the full
// main area (chat column + log column when open) below the sidebar.
func (m *Model) inputWidth() int {
	w := m.width - SidebarWidth - 2 // 2 for rounded-border left/right
	if w < 10 {
		w = 10
	}
	return w
}

func (m *Model) logInnerWidth() int {
	w := m.chatAreaWidth() - 2 // 2 for input-box padding
	if w < 20 {
		w = 20
	}
	return w
}

func (m *Model) layoutInner() {
	if m.width == 0 || m.height == 0 {
		return
	}
	logWidth := m.chatAreaWidth() - 2
	if logWidth < 20 {
		logWidth = 20
	}
	logHeight := m.height - 1 /*title*/ - 1 /*typing*/ - 3 /*input container*/
	// We may grow the input container when attachments / suggestions appear, but
	// we reflow on the fly in View().
	if logHeight < 3 {
		logHeight = 3
	}
	m.log.Width = logWidth
	m.log.Height = logHeight
	m.input.Width = m.inputWidth() - 2
	m.refreshViewport()
}

// View renders the full frame.
func (m *Model) View() string {
	if !m.ready {
		return ""
	}
	chat, hasActive := m.Store.ActiveChat()
	chats := m.Store.SortedChats()

	sidebarHeight := m.height
	sidebar := zoneMarkSidebar(m.theme, chats, m.Store.ActiveChatID(), sidebarHeight)

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

	// Stacked rows in the chat column: title + messages + typing line.
	chatCol := lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		m.log.View(),
		typingLine,
	)

	// Top half: chat column plus, when enabled, the system-log column on the
	// right. The input container spans across both below.
	var topRow string
	if m.logVisible {
		// topHeight = title(1) + log.Height + typing(1) — we render the log
		// panel to the same vertical extent so borders align.
		topHeight := 1 + m.log.Height + 1
		logCol := m.renderLogColumn(topHeight)
		topRow = lipgloss.JoinHorizontal(lipgloss.Top, chatCol, logCol)
	} else {
		topRow = chatCol
	}

	rightCol := lipgloss.JoinVertical(lipgloss.Left, topRow, inputContainer)
	frame := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightCol)

	// Floating preview overlay (top-right).
	if preview := m.Store.HoveredPreview(); preview != nil {
		panel := m.renderPreviewPanel(preview)
		if panel != "" {
			frame = overlayAt(frame, panel, m.width-PreviewCols-6, 1)
		}
	}

	return zone.Scan(frame)
}

// renderLogEntryLines renders a single log entry. Collapsed entries get one
// truncated line; expanded entries wrap to as many lines as the body needs,
// with continuation lines indented to align past the `[level]` prefix. Every
// line is wrapped in the same click zone so clicking any of them toggles the
// entry's expansion state.
func (m *Model) renderLogEntryLines(e store.LogEntry, width int) []string {
	zoneID := ZoneLogEntryPrefix + e.ID
	text := e.Content.Text

	if !m.expandedLogs[e.ID] {
		body := text
		if runewidth.StringWidth(body) > width {
			body = runewidth.Truncate(body, width, "…")
		}
		return []string{zone.Mark(zoneID, renderLogText(m.theme, body))}
	}

	// Expanded: render the level token, then wrap the body.
	level, rest := parseLogLevel(text)
	levelColor := m.theme.SystemColor
	switch level {
	case "error":
		levelColor = lipgloss.Color("#f87171")
	case "warn":
		levelColor = lipgloss.Color("#fbbf24")
	case "info":
		levelColor = lipgloss.Color("#60a5fa")
	case "debug":
		levelColor = lipgloss.Color("#9ca3af")
	}
	levelStyle := lipgloss.NewStyle().Foreground(levelColor).Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(m.theme.SystemColor)

	prefix := "[" + level + "]"
	prefixW := runewidth.StringWidth(prefix) + 1 // trailing space
	avail := width - prefixW
	if avail < 10 {
		avail = 10
	}
	chunks := wrapByWidth(rest, avail)
	if len(chunks) == 0 {
		chunks = []string{""}
	}
	indent := strings.Repeat(" ", prefixW)

	out := make([]string, 0, len(chunks))
	for i, c := range chunks {
		if i == 0 {
			out = append(out, levelStyle.Render(prefix)+" "+bodyStyle.Render(c))
		} else {
			out = append(out, indent+bodyStyle.Render(c))
		}
	}
	// Mark the whole block as a single zone so clicks on continuation lines
	// register the same as clicks on the first line. bubblezone tracks a
	// multi-row bounding box when start/end markers sit on different rows.
	marked := zone.Mark(zoneID, strings.Join(out, "\n"))
	return strings.Split(marked, "\n")
}

// parseLogLevel splits a "[level] rest" string. Returns ("log", text) if the
// text doesn't carry a level prefix.
func parseLogLevel(text string) (string, string) {
	if !strings.HasPrefix(text, "[") {
		return "log", text
	}
	end := strings.Index(text, "] ")
	if end < 0 {
		return "log", text
	}
	return text[1:end], text[end+2:]
}

// wrapByWidth splits a plain string into lines so each is at most `w` visible
// cells wide. Rune-aware, not word-aware — good enough for JSON and other log
// payloads that dominate the system log.
func wrapByWidth(s string, w int) []string {
	if w <= 0 {
		return []string{s}
	}
	var out []string
	var line strings.Builder
	lineW := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if rw == 0 {
			rw = 1
		}
		if lineW+rw > w && lineW > 0 {
			out = append(out, line.String())
			line.Reset()
			lineW = 0
		}
		line.WriteRune(r)
		lineW += rw
	}
	if line.Len() > 0 {
		out = append(out, line.String())
	}
	return out
}

// renderLogColumn renders the right-hand system log panel. Height must match
// the combined height of titleBar + messages viewport + typing line so the
// input below spans both columns cleanly.
func (m *Model) renderLogColumn(height int) string {
	innerW := LogColumnWidth - 3 // 1 border + 2 padding
	if innerW < 8 {
		innerW = 8
	}

	closeBtn := zone.Mark(ZoneLogToggle,
		lipgloss.NewStyle().Foreground(m.theme.PromptColor).Render("[×]"),
	)
	headerLabel := lipgloss.NewStyle().
		Foreground(m.theme.SystemColor).
		Render("system log")
	// header: "system log" + right-aligned close button, padded to innerW
	labelW := runewidth.StringWidth("system log")
	btnW := runewidth.StringWidth("[×]")
	pad := innerW - labelW - btnW
	if pad < 1 {
		pad = 1
	}
	header := headerLabel + strings.Repeat(" ", pad) + closeBtn
	rule := lipgloss.NewStyle().Foreground(m.theme.BorderColor).
		Render(strings.Repeat("─", innerW))

	// Body: tail of system entries that fit. Collapsed entries render on one
	// line (truncated with `…`); clicking an entry toggles its expansion to
	// fully wrapped multi-line display.
	entries := m.Store.SystemEntries()
	bodyHeight := height - 3 /*header + rule + bottom margin*/
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// Walk newest→oldest, render each entry to 1 or more lines, stop once we
	// have enough to fill bodyHeight. Reverse so display order is old→new.
	rendered := make([]string, 0, bodyHeight)
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Content.Type != "text" {
			continue
		}
		lines := m.renderLogEntryLines(e, innerW)
		for j := len(lines) - 1; j >= 0; j-- {
			rendered = append(rendered, lines[j])
			if len(rendered) >= bodyHeight {
				break
			}
		}
		if len(rendered) >= bodyHeight {
			break
		}
	}
	// Reverse.
	for l, r := 0, len(rendered)-1; l < r; l, r = l+1, r-1 {
		rendered[l], rendered[r] = rendered[r], rendered[l]
	}
	for len(rendered) < bodyHeight {
		rendered = append([]string{""}, rendered...)
	}

	inner := strings.Join(append([]string{header, rule}, rendered...), "\n")
	return lipgloss.NewStyle().
		Width(LogColumnWidth).
		Height(height).
		MaxWidth(LogColumnWidth).
		BorderStyle(lipgloss.Border{Left: "│"}).
		BorderLeft(true).
		BorderForeground(m.theme.BorderColor).
		Padding(0, 1).
		Render(inner)
}


func (m *Model) renderTitleBar(activeID string) string {
	// When the log column is open it has its own [×] button, so the title bar
	// only spans the chat column. When the log is hidden we extend the title
	// bar all the way right so the re-open toggle stays visible.
	rightWidth := m.chatAreaWidth()
	if !m.logVisible {
		rightWidth = m.width - SidebarWidth
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

	// Only show the title-bar toggle when the log is hidden — when open the
	// column's own [×] handles closing.
	var btnText string
	if !m.logVisible {
		btnText = " [log ▸] "
	}
	btnW := runewidth.StringWidth(btnText)

	// Pre-truncate by visible width so the terminal doesn't hard-wrap us.
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

func (m *Model) renderReactionPicker(width int) string {
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
	strip := strings.Join(parts, " ")
	hint := lipgloss.NewStyle().Foreground(m.theme.SystemColor).Render("  Enter pick · Esc cancel")
	return lipgloss.NewStyle().MaxWidth(width).Render(strip + hint)
}

func (m *Model) renderPreviewPanel(preview *store.HoveredPreview) string {
	bytes := loadPreviewBytes(preview)
	if bytes == nil {
		return RenderPreview(m.theme, preview, 0)
	}

	if kitty.Supported() {
		id, err := kitty.EnsureTransmitted(os.Stdout, preview.CacheKey, bytes, PreviewCols, PreviewRows)
		if err == nil {
			return RenderPreview(m.theme, preview, id)
		}
	}

	// Fallback: half-block rasterization via charmbracelet/x/mosaic.
	img, _, err := image.Decode(bytesReader(bytes))
	if err != nil {
		return RenderPreview(m.theme, preview, 0)
	}
	mo := mosaic.New().Width(PreviewCols).Height(PreviewRows * 2)
	rendered := mo.Render(img)
	header := lipgloss.NewStyle().Foreground(m.theme.AttachmentColor).Render("📎 " + preview.Name)
	panel := lipgloss.JoinVertical(lipgloss.Left, header, rendered)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderColor).
		Padding(0, 1).
		Render(panel)
}

func loadPreviewBytes(preview *store.HoveredPreview) []byte {
	if preview == nil {
		return nil
	}
	if len(preview.Bytes) > 0 {
		return preview.Bytes
	}
	if preview.Path == "" {
		return nil
	}
	data, err := os.ReadFile(preview.Path)
	if err != nil {
		return nil
	}
	return data
}

func bytesReader(b []byte) io.Reader {
	return &byteReader{data: b}
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// --- helpers used above ---

func zoneMarkSidebar(theme Theme, chats []store.ChatState, activeID string, height int) string {
	// Render sidebar row-by-row so we can zone.Mark each clickable row, then
	// stitch together inside the same outer frame RenderSidebar would build.
	// Simpler alternative: mark each rendered row after RenderSidebar by
	// searching for its ID; but we just rebuild inline here for clarity.
	//
	// We keep using the layout from sidebar.go (header + rows + spacer + hints)
	// but intercept row rendering with zone marks.
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
		MaxWidth(SidebarWidth + 1). // + 1 for the right border
		Height(height).
		BorderStyle(lipgloss.Border{Right: "│"}).
		BorderRight(true).
		BorderForeground(theme.BorderColor).
		Render(column)
}

func (m *Model) zoneMarkEntries(theme Theme, chat store.ChatState, width int) string {
	lines := make([]string, 0, len(chat.Entries)+4)
	if chat.DroppedCount > 0 {
		msg := fmt.Sprintf("… %d older messages dropped", chat.DroppedCount)
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.SystemColor).Render(msg))
	}
	for i := range chat.Entries {
		e := chat.Entries[i]
		rendered := renderEntry(theme, e, chat.Entries, width)
		if e.ID == m.selectedID && m.selecting {
			rendered = lipgloss.NewStyle().
				Background(theme.SuggestionBG).
				Render(rendered)
		}
		if e.Content.Type == "attachment" && kitty.SupportedMimeType(e.Content.MimeType) {
			rendered = zone.Mark(ZoneAttachmentPrefix+e.ID, rendered)
		}
		// Wrap every entry with a click-zone so mouse users can select it.
		rendered = zone.Mark(ZoneMessagePrefix+e.ID, rendered)
		lines = append(lines, rendered)
		if e.ID == m.selectedID && m.selecting {
			lines = append(lines, renderActionRow(theme))
		}
	}
	return strings.Join(lines, "\n")
}

func renderActionRow(theme Theme) string {
	btn := func(label string, zoneID string) string {
		st := lipgloss.NewStyle().
			Foreground(theme.PromptColor).
			Background(theme.SuggestionBG).
			Padding(0, 1)
		return zone.Mark(zoneID, st.Render(label))
	}
	row := btn("↩ reply (r)", ZoneReplyButton) + "  " + btn("🙂 react (e)", ZoneReactButton)
	hint := lipgloss.NewStyle().Foreground(theme.SystemColor).Render("   ↑/↓ move · Esc cancel")
	return lipgloss.NewStyle().PaddingLeft(2).Render(row + hint)
}

// overlayAt lets us paint `overlay` onto `base` at (x, y). Lipgloss doesn't
// have a native "position absolute" composite, so we do a best-effort grid
// merge: replace characters at the target rows/cols with the overlay's runes.
func overlayAt(base, overlay string, x, y int) string {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	for i, ov := range overlayLines {
		row := y + i
		if row >= len(baseLines) {
			break
		}
		// lipgloss.PlaceHorizontal won't help here because base lines carry
		// ANSI styling. For a first pass we just trust the overlay to be
		// shorter than the terminal and replace the tail.
		baseLines[row] = mergeLineAtCol(baseLines[row], ov, x)
	}
	return strings.Join(baseLines, "\n")
}

// mergeLineAtCol appends `overlay` to the end of `line` padded with spaces
// so the leftmost rune of the overlay lands near column `col`. Approximate —
// underlying base cells behind the overlay remain but get truncated by
// terminal scrollback on overlap. Acceptable for MVP; revisit when we need
// perfect compositing.
func mergeLineAtCol(line, overlay string, col int) string {
	plain := lipgloss.NewStyle().Width(col).Render(line)
	return plain + overlay
}

// fileBase returns the base name of a path without importing filepath (keeps this file lean).
func fileBase(p string) string {
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
}

func statSafe(p string) interface {
	Size() int64
} {
	info, err := os.Stat(p)
	if err != nil {
		return nil
	}
	return info
}

func guessMime(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".bmp"):
		return "image/bmp"
	case strings.HasSuffix(lower, ".txt"):
		return "text/plain"
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
