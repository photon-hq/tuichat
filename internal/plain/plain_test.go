package plain

import (
	"testing"

	"github.com/photon-hq/tuichat/internal/protocol"
)

func TestPlainFormatRichlink(t *testing.T) {
	tests := []struct {
		name    string
		content protocol.Content
		want    string
	}{
		{
			name:    "url only",
			content: protocol.Content{Type: "richlink", Url: "https://example.com/x"},
			want:    "[link] https://example.com/x",
		},
		{
			name:    "title without summary",
			content: protocol.Content{Type: "richlink", Url: "https://example.com/x", Title: "Hello"},
			want:    "[link] Hello",
		},
		{
			name:    "title with summary",
			content: protocol.Content{Type: "richlink", Url: "https://example.com/x", Title: "Hello", Summary: "world"},
			want:    "[link] Hello — world",
		},
		{
			name:    "summary without title falls back to url label",
			content: protocol.Content{Type: "richlink", Url: "https://example.com/x", Summary: "world"},
			want:    "[link] https://example.com/x — world",
		},
		{
			name:    "missing url and title renders bracket only",
			content: protocol.Content{Type: "richlink"},
			want:    "[link] ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := plainFormat(tt.content); got != tt.want {
				t.Errorf("plainFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}
