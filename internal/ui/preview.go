package ui

import (
	"image"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/mosaic"

	"github.com/photon-hq/tuichat/internal/kitty"
	"github.com/photon-hq/tuichat/internal/store"
)

const (
	PreviewCols = 40
	PreviewRows = 10
)

// RenderPreview returns a bordered floating panel showing the image for
// `preview`. Image ID is looked up via `kitty.EnsureTransmitted` during UI
// rendering — callers must pass the already-resolved id. Returns empty if
// not supported.
func RenderPreview(theme Theme, preview *store.HoveredPreview, imageID uint32) string {
	if preview == nil {
		return ""
	}
	headerStyle := lipgloss.NewStyle().Foreground(theme.AttachmentColor).Width(PreviewCols + 2)
	header := headerStyle.Render("📎 " + preview.Name)

	var body string
	if imageID > 0 {
		fg := kitty.HexColor(imageID)
		rows := kitty.PlaceholderRows(PreviewCols, PreviewRows)
		styledRows := make([]string, len(rows))
		for i, row := range rows {
			styledRows[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(row)
		}
		body = strings.Join(styledRows, "\n")
	} else {
		body = lipgloss.NewStyle().Foreground(theme.SystemColor).Render("[loading image…]")
	}

	panel := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderColor).
		Background(theme.SuggestionBG).
		Padding(0, 1).
		Render(panel)
}

// renderPreviewPanel picks the best renderer for the hovered attachment:
// Kitty graphics if the terminal supports it (transmits once, reuses by id),
// otherwise a half-block fallback via charmbracelet/x/mosaic. Falls through
// to the placeholder "loading image…" panel if we can't even decode bytes.
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

// loadPreviewBytes returns the raw bytes for a hovered attachment — either
// the in-memory bytes carried by HoveredPreview, or re-read from disk.
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

// bytesReader wraps a []byte as an io.Reader without allocating a bytes.Buffer.
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
