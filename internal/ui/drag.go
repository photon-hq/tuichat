package ui

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
	"go.dalton.dog/bubbleup"
)

// Drag-select + OSC-52 copy. Two panes can host a selection (the chat
// viewport and the system log column); pane-specific coord mappers and
// plain-text mirrors live below.

// clearDragSelection resets drag state (selection + in-progress + origin).
// Does not touch the copy-confirmation bubble — that runs its own 3s timer.
func (m *Model) clearDragSelection() {
	m.dragActive = false
	m.dragInProgress = false
	m.dragMoved = false
	m.dragStartRow = 0
	m.dragStartCol = 0
	m.dragEndRow = 0
	m.dragEndCol = 0
}

// --- chat viewport coord mappers ---

// inMessageLogRect returns true if screen coords (x,y) fall inside the chat
// column viewport — the area that participates in chat-pane drag-select.
func (m *Model) inMessageLogRect(x, y int) bool {
	// Screen row 0 is the title bar; viewport rows start at 1.
	if y < 1 || y > m.log.Height {
		return false
	}
	colMin := SidebarWidth + 1
	colMax := SidebarWidth + m.chatAreaWidth()
	return x >= colMin && x <= colMax
}

// screenRowToContentRow maps screen Y to an index into plainLogLines,
// accounting for viewport scroll offset and clamping to the content range.
func (m *Model) screenRowToContentRow(y int) int {
	row := (y - 1) + m.log.YOffset
	if row < 0 {
		row = 0
	}
	if n := len(m.plainLogLines); n > 0 && row > n-1 {
		row = n - 1
	}
	return row
}

// screenColToContentCol maps screen X to a display-cell column within the
// chat column's inner content area (0 = first cell of a message line).
func (m *Model) screenColToContentCol(x int) int {
	col := x - (SidebarWidth + 1)
	if col < 0 {
		col = 0
	}
	return col
}

// --- log-column coord mappers ---

// Log-column geometry. renderLogColumn paints:
//
//	[border 1 cell] [pad 1] [ header ] [pad 1]   ← screen row 0
//	[border]        [pad 1] [ ─rule─ ] [pad 1]   ← screen row 1
//	[border]        [pad 1] [ body-0 ] [pad 1]   ← screen row 2
//	[border]        [pad 1] [ body-N ] [pad 1]
//
// Body rows are the only area we let users drag-select; header and rule are
// static UI decoration.
const (
	sysLogFirstBodyScreenRow = 2
	sysLogPadCells           = 1
	sysLogBorderCells        = 1
)

// inSystemLogRect returns true if (x,y) sits over a body row in the log
// column's content area. Header, rule, border, and outer padding all
// return false.
func (m *Model) inSystemLogRect(x, y int) bool {
	if !m.logVisible {
		return false
	}
	colStart := m.width - LogColumnWidth + sysLogBorderCells + sysLogPadCells - 1
	colEnd := m.width - sysLogPadCells - 1
	if x < colStart || x > colEnd {
		return false
	}
	if y < sysLogFirstBodyScreenRow {
		return false
	}
	if len(m.plainSystemLogLines) == 0 {
		return false
	}
	// Body region is the viewport's visible Height, regardless of how much
	// content lives in the full buffer.
	if y >= sysLogFirstBodyScreenRow+m.logPanel.Height {
		return false
	}
	return true
}

func (m *Model) screenRowToSystemLogRow(y int) int {
	// Offset by the log panel's scroll position so a drag on the visible
	// row maps to the right content row regardless of how far the user has
	// scrolled.
	row := (y - sysLogFirstBodyScreenRow) + m.logPanel.YOffset
	if row < 0 {
		row = 0
	}
	if n := len(m.plainSystemLogLines); n > 0 && row > n-1 {
		row = n - 1
	}
	return row
}

func (m *Model) screenColToSystemLogCol(x int) int {
	colStart := m.width - LogColumnWidth + sysLogBorderCells + sysLogPadCells - 1
	col := x - colStart
	if col < 0 {
		col = 0
	}
	return col
}

// --- selection extraction helpers ---

// selectionRange normalizes drag endpoints so (loRow,loCol) is always before
// (hiRow,hiCol) in reading order.
func (m *Model) selectionRange() (loRow, loCol, hiRow, hiCol int) {
	loRow, loCol = m.dragStartRow, m.dragStartCol
	hiRow, hiCol = m.dragEndRow, m.dragEndCol
	if hiRow < loRow || (hiRow == loRow && hiCol < loCol) {
		loRow, loCol, hiRow, hiCol = hiRow, hiCol, loRow, loCol
	}
	return
}

// plainSliceByCols returns the substring of `s` covering display-cell
// columns [from, to). Respects rune widths (emoji, CJK = 2 cells).
func plainSliceByCols(s string, from, to int) string {
	if from >= to {
		return ""
	}
	var out strings.Builder
	col := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if rw == 0 {
			rw = 1
		}
		if col >= to {
			break
		}
		if col >= from {
			out.WriteRune(r)
		}
		col += rw
	}
	return out.String()
}

// --- motion + release ---

func (m *Model) handleDragMotion(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	var row, col int
	switch m.dragPane {
	case paneSystemLog:
		row = m.screenRowToSystemLogRow(msg.Y)
		col = m.screenColToSystemLogCol(msg.X)
	default:
		row = m.screenRowToContentRow(msg.Y)
		col = m.screenColToContentCol(msg.X)
	}
	if row == m.dragEndRow && col == m.dragEndCol && m.dragMoved {
		return m, nil
	}
	if row != m.dragStartRow || col != m.dragStartCol {
		m.dragMoved = true
		m.dragActive = true
	}
	m.dragEndRow = row
	m.dragEndCol = col
	m.refreshViewport()
	return m, nil
}

func (m *Model) handleDragRelease(_ tea.MouseMsg) (tea.Model, tea.Cmd) {
	m.dragInProgress = false
	if !m.dragMoved {
		// Click, not drag — replay the originating press through the
		// zone-match cascade so click-to-expand, click-to-select-message
		// etc. keep working.
		pressX, pressY := m.dragPressX, m.dragPressY
		m.clearDragSelection()
		synthetic := tea.MouseMsg{
			X:      pressX,
			Y:      pressY,
			Button: tea.MouseButtonLeft,
			Action: tea.MouseActionPress,
		}
		return m.dispatchClick(synthetic)
	}

	// Pick the right source of plain lines for this pane.
	var source []string
	switch m.dragPane {
	case paneSystemLog:
		source = m.plainSystemLogLines
	default:
		source = m.plainLogLines
	}

	loRow, loCol, hiRow, hiCol := m.selectionRange()
	if loRow < 0 {
		loRow = 0
	}
	if hiRow >= len(source) {
		hiRow = len(source) - 1
	}

	// Build the plain-text payload. For each row, slice by display columns,
	// and trim trailing whitespace so we don't paste bottom-align padding.
	parts := make([]string, 0, hiRow-loRow+1)
	for r := loRow; r <= hiRow; r++ {
		line := source[r]
		lineW := runewidth.StringWidth(line)
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
		parts = append(parts, strings.TrimRight(plainSliceByCols(line, from, to), " \t"))
	}
	text := strings.Join(parts, "\n")
	rowCount := hiRow - loRow + 1
	label := "copied 1 line to clipboard"
	if rowCount != 1 {
		label = fmt.Sprintf("copied %d lines to clipboard", rowCount)
	}

	m.refreshViewport()
	return m, tea.Batch(
		emitOSC52(text),
		m.alert.NewAlertCmd(bubbleup.InfoKey, label),
	)
}

// emitOSC52 returns a Cmd that writes the OSC-52 clipboard escape to stdout.
// Supported by Ghostty, iTerm2, Kitty, Alacritty, WezTerm, recent
// Terminal.app; silent no-op on terminals that don't. Altscreen is active,
// but OSC escapes are out-of-band and don't disturb the rendered frame.
func emitOSC52(text string) tea.Cmd {
	return func() tea.Msg {
		enc := base64.StdEncoding.EncodeToString([]byte(text))
		_, _ = os.Stdout.WriteString("\x1b]52;c;" + enc + "\x07")
		return nil
	}
}
