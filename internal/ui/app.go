// Package ui is the Bubbletea UI for tuichat. The Model owns the Store and
// delegates rendering to the sibling files (sidebar.go, messagelog.go, etc.).
package ui

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"go.dalton.dog/bubbleup"

	"github.com/photon-hq/tuichat/internal/store"
)

// copyAlertDuration is how long the copy confirmation bubble sits on screen.
const copyAlertDuration = 3 * time.Second

// dragPane identifies which scrollable region owns an active drag-selection.
type dragPane int

const (
	paneChat dragPane = iota
	paneSystemLog
)

// Zone IDs used with bubblezone for click/hover routing.
const (
	ZoneSidebarRowPrefix = "sidebar-row:"
	ZoneAttachmentPrefix = "attachment:" // per-log-entry image chip
	ZoneMessagePrefix    = "msg:"        // click a message to select it
	ZoneReplyButton      = "action-reply"
	ZoneReactButton      = "action-react"
	ZoneReactionPrefix   = "reaction:" // each quick-pick emoji in the picker
	ZoneReactionOther    = "reaction-other"
	ZoneLogToggle        = "log-toggle"
	ZoneLogEntryPrefix   = "log-entry:" // click a log line to expand/collapse
)

// LogColumnWidth is the fixed width (in cells) of the right-hand system log
// panel. Includes a 1-cell border on the left.
const LogColumnWidth = 40

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
	selecting       bool
	selectedID      string
	replyingTo      string // non-empty → next submit is a quoted reply
	reactingTo      string // non-empty → emoji picker open for this entry
	reactionIdx     int
	emojiInputMode  bool            // picker's "other" slot opened a text field
	emojiInput      textinput.Model // where the user types a custom emoji

	// Right-hand system log panel. Open by default; toggled via a title-bar
	// button or Ctrl+Backtick.
	logVisible   bool
	expandedLogs map[string]bool // log entry IDs currently shown fully-wrapped

	// Drag-select state. Rows are indices into whichever pane's plain-lines
	// buffer the drag lives in; cols are display-cell columns into that row.
	//   dragActive     — a selection is visible and highlighted
	//   dragInProgress — mouse button currently held; motion updates end
	//   dragMoved      — any motion fired between press and release
	//   dragPane       — which pane the drag lives in
	dragActive          bool
	dragInProgress      bool
	dragMoved           bool
	dragPane            dragPane
	dragStartRow        int
	dragStartCol        int
	dragEndRow          int
	dragEndCol          int
	dragPressX          int // screen coords of the originating press — used
	dragPressY          int // to dispatch a click-not-drag on release.
	plainLogLines       []string // ANSI-stripped chat viewport content
	plainSystemLogLines []string // ANSI-stripped log column content (body only)

	// Copy-confirmation bubble (library-driven).
	alert bubbleup.AlertModel
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

	emoji := textinput.New()
	emoji.Placeholder = "type / paste an emoji…"
	emoji.Prompt = ""
	emoji.CharLimit = 16 // plenty for a single emoji + ZWJ sequence

	// bubbleup AlertModel: fixed-width bubble at the top-right, using
	// Unicode prefixes so it renders cleanly without requiring a Nerd Font.
	alert := bubbleup.NewAlertModel(40, false, copyAlertDuration).
		WithPosition(bubbleup.TopRightPosition).
		WithUnicodePrefix()

	return &Model{
		Store:        s,
		theme:        DefaultTheme,
		input:        in,
		log:          vp,
		emojiInput:   emoji,
		logVisible:   true,
		expandedLogs: map[string]bool{},
		alert:        alert,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.alert.Init())
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
		// Any keystroke clears a persisted drag-selection.
		if m.dragActive {
			m.clearDragSelection()
			m.refreshViewport()
		}
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case StoreChangedMsg:
		// New content shifts row indices; clear any persisted drag-selection
		// so the highlighted range doesn't point at something else now.
		if m.dragActive {
			m.clearDragSelection()
		}
		m.refreshViewport()
		return m, m.forwardToAlert(msg)
	}

	// Any other message (incl. bubbleup's internal tick) needs to reach the
	// alert model so it can drive its own animation + dismissal timer.
	alertCmd := m.forwardToAlert(msg)
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.onInputChange()
	return m, tea.Batch(cmd, alertCmd)
}

// forwardToAlert routes a message through the bubbleup AlertModel and
// captures both its updated state and any command it wants run.
func (m *Model) forwardToAlert(msg tea.Msg) tea.Cmd {
	out, cmd := m.alert.Update(msg)
	if updated, ok := out.(bubbleup.AlertModel); ok {
		m.alert = updated
	}
	return cmd
}


// clearModalState drops any cross-message modes (select, reply, react, drag-
// selection). Used when switching chats — the targets only make sense in
// their original chat.
func (m *Model) clearModalState() {
	m.selecting = false
	m.selectedID = ""
	m.replyingTo = ""
	m.reactingTo = ""
	m.reactionIdx = 0
	m.emojiInputMode = false
	m.emojiInput.SetValue("")
	m.emojiInput.Blur()
	m.clearDragSelection()
}


