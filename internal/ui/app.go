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

	"github.com/photon-hq/tuichat/internal/drop"
	"github.com/photon-hq/tuichat/internal/kitty"
	"github.com/photon-hq/tuichat/internal/protocol"
	"github.com/photon-hq/tuichat/internal/store"
)

// Zone IDs used with bubblezone for click/hover routing.
const (
	ZoneSidebarRowPrefix = "sidebar-row:"
	ZoneAttachmentPrefix = "attachment:" // per-log-entry image chip
)

// StoreChangedMsg is sent whenever the RPC server mutates the store and the UI
// needs to re-render. The server's pump goroutine sends these via Program.Send.
type StoreChangedMsg struct{}

// helpLines are dumped into the active chat when user types `/help`.
var helpLines = []string{
	"tuichat — keybindings",
	"  Ctrl+N         new chat",
	"  Ctrl+J / K     cycle chats (down / up)",
	"  Ctrl+L         clear active chat",
	"  Ctrl+C         exit",
	"  Tab            complete slash command",
	"  Esc            cancel input + drop pending attachments",
	"  drag file      attach (or paste its path)",
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
		Store: s,
		theme: DefaultTheme,
		input: in,
		log:   vp,
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

	// Always-available shortcuts
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+n":
		id := m.Store.NewChat()
		_ = id
		m.syncInputFromDraft()
		m.refreshViewport()
		return m, nil
	case "ctrl+j":
		m.Store.CycleActiveChat(1)
		m.syncInputFromDraft()
		m.refreshViewport()
		return m, nil
	case "ctrl+k":
		m.Store.CycleActiveChat(-1)
		m.syncInputFromDraft()
		m.refreshViewport()
		return m, nil
	case "ctrl+l":
		if id := m.Store.ActiveChatID(); id != "" {
			m.Store.AppendSystem(id, "screen cleared")
			m.refreshViewport()
		}
		return m, nil
	case "tab":
		m.cycleSlashCompletion()
		return m, nil
	case "esc":
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

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.onInputChange()
	return m, cmd
}

func (m *Model) handlePaste(raw string) (tea.Model, tea.Cmd) {
	id := m.Store.ActiveChatID()
	if id == "" {
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
		// Sidebar clicks → switch active chat.
		for _, chat := range m.Store.SortedChats() {
			zoneID := ZoneSidebarRowPrefix + chat.ID
			if zone.Get(zoneID).InBounds(msg) {
				m.Store.SetActiveChat(chat.ID)
				m.syncInputFromDraft()
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
	}
	return m, nil
}

func (m *Model) handleSubmit() (tea.Model, tea.Cmd) {
	id := m.Store.ActiveChatID()
	if id == "" {
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
		m.Store.AppendUser(id, content, a.Path)
		m.Store.PushUserInput(id, content)
	}
	if raw != "" {
		content := protocol.Content{Type: "text", Text: raw}
		m.Store.AppendUser(id, content, "")
		m.Store.PushUserInput(id, content)
	}

	m.Store.SetInputDraft(id, "")
	m.input.SetValue("")
	m.prefix = ""
	m.tabIndex = 0
	m.refreshViewport()
	return m, nil
}

func (m *Model) onInputChange() {
	m.prefix = m.input.Value()
	m.tabIndex = 0
	if id := m.Store.ActiveChatID(); id != "" {
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
	content := zoneMarkEntries(m.theme, chat, inner)
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

func (m *Model) logInnerWidth() int {
	w := m.width - SidebarWidth - 2 // 2 for input-box padding
	if w < 20 {
		w = 20
	}
	return w
}

func (m *Model) layoutInner() {
	if m.width == 0 || m.height == 0 {
		return
	}
	logWidth := m.width - SidebarWidth - 2
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
	m.input.Width = logWidth - 2
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

	rightCol := lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		m.log.View(),
		typingLine,
		inputContainer,
	)

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

func (m *Model) renderTitleBar(activeID string) string {
	titleColor := lipgloss.NewStyle().Foreground(m.theme.UserColor)
	subColor := lipgloss.NewStyle().Foreground(m.theme.SystemColor)
	bar := " " + m.theme.Title
	if activeID != "" {
		bar += " · " + activeID
	}
	bar += " "
	rest := " — Ctrl+N new · Ctrl+J/K nav · Ctrl+C exit · Ctrl+L clear · Tab complete · Esc cancel"
	rightWidth := m.width - SidebarWidth
	if rightWidth < 0 {
		rightWidth = 0
	}
	rendered := titleColor.Background(m.theme.BorderColor).Render(bar) +
		subColor.Background(m.theme.BorderColor).Render(rest)
	return lipgloss.NewStyle().
		Background(m.theme.BorderColor).
		Width(rightWidth).
		Render(rendered)
}

func (m *Model) renderInputContainer(chat store.ChatState, hasActive bool) string {
	innerWidth := m.width - SidebarWidth - 2 /*rounded-border left/right*/ - 2 /*padding*/
	if innerWidth < 10 {
		innerWidth = 10
	}
	var rows []string

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
		Width(m.width - SidebarWidth - 2).
		Render(inner)
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
		id := c.ID
		if len(id) > SidebarWidth-4 {
			id = id[:SidebarWidth-5] + "…"
		}
		arrow := "  "
		if selected {
			arrow = "› "
		}
		raw := arrowStyle.Render(arrow) + nameStyle.Render(id)
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
		Height(height).
		BorderStyle(lipgloss.Border{Right: "│"}).
		BorderRight(true).
		BorderForeground(theme.BorderColor).
		Render(column)
}

func zoneMarkEntries(theme Theme, chat store.ChatState, width int) string {
	lines := make([]string, 0, len(chat.Entries)+2)
	if chat.DroppedCount > 0 {
		msg := fmt.Sprintf("… %d older messages dropped", chat.DroppedCount)
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.SystemColor).Render(msg))
	}
	for i := range chat.Entries {
		rendered := renderEntry(theme, chat.Entries[i], chat.Entries, width)
		if chat.Entries[i].Content.Type == "attachment" &&
			kitty.SupportedMimeType(chat.Entries[i].Content.MimeType) {
			rendered = zone.Mark(ZoneAttachmentPrefix+chat.Entries[i].ID, rendered)
		}
		lines = append(lines, rendered)
	}
	return strings.Join(lines, "\n")
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
