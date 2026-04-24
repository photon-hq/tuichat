package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Layout math for the three-column frame:
//
//   [sidebar | border] [chat column] [border | log column]
//                     (1 cell)       (1 cell)
//
// Widths below carve the terminal along those boundaries. Helpers are all
// methods on *Model so they can read the live window size + `logVisible`.

// chatAreaWidth is the width of the messages column, accounting for the
// sidebar's right border and (when the log panel is open) the log column's
// left border.
func (m *Model) chatAreaWidth() int {
	w := m.width - SidebarWidth - 1
	if m.logVisible {
		w -= LogColumnWidth + 1
	}
	if w < 20 {
		w = 20
	}
	return w
}

// inputWidth is the width of the input container (message composer). The
// input always spans chat + log column below the sidebar, regardless of
// whether the log panel is visible.
func (m *Model) inputWidth() int {
	// -1 sidebar border, -2 rounded-border left/right of the input container.
	w := m.width - SidebarWidth - 1 - 2
	if w < 10 {
		w = 10
	}
	return w
}

// logInnerWidth is the message-rendering width inside the chat viewport.
// Two cells of slack for padding around styled content.
func (m *Model) logInnerWidth() int {
	w := m.chatAreaWidth() - 2
	if w < 20 {
		w = 20
	}
	return w
}

// layoutInner re-sizes the chat viewport + message input to match the
// current terminal dimensions. Called on WindowSizeMsg and whenever the log
// panel is toggled.
func (m *Model) layoutInner() {
	if m.width == 0 || m.height == 0 {
		return
	}
	logWidth := m.chatAreaWidth()
	if logWidth < 20 {
		logWidth = 20
	}
	logHeight := m.height - 1 /*title*/ - 1 /*typing*/ - 3 /*input container*/
	if logHeight < 3 {
		logHeight = 3
	}
	m.log.Width = logWidth
	m.log.Height = logHeight
	m.input.Width = m.inputWidth() - 2
	m.refreshViewport()
}

// overlayAt paints `overlay` onto `base` at (x, y). Lipgloss has no native
// absolute-positioning composite, so we do a best-effort grid merge: for
// each row of the overlay we splice it into the corresponding base row at
// the given x offset.
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
		baseLines[row] = mergeLineAtCol(baseLines[row], ov, x)
	}
	return strings.Join(baseLines, "\n")
}

// mergeLineAtCol appends `overlay` to the end of `line` padded with spaces
// so the leftmost rune of the overlay lands near column `col`. Approximate
// (no ANSI-aware cell replacement), but good enough for the floating
// preview + toast overlays we paint today.
func mergeLineAtCol(line, overlay string, col int) string {
	plain := lipgloss.NewStyle().Width(col).Render(line)
	return plain + overlay
}
