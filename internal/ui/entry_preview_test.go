package ui

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/photon-hq/tuichat/internal/protocol"
	"github.com/photon-hq/tuichat/internal/store"
)

func TestPreviewForEntryRichlinkCover(t *testing.T) {
	coverBytes := []byte{0x89, 0x50, 0x4e, 0x47}
	entry := store.LogEntry{
		ID: "msg-1",
		Content: protocol.Content{
			Type:  "richlink",
			Url:   "https://example.com/article",
			Title: "Example article",
			Cover: &protocol.Cover{
				MimeType: "image/png",
				Bytes:    base64.StdEncoding.EncodeToString(coverBytes),
			},
		},
	}

	preview, ok := previewForEntry(entry)
	if !ok {
		t.Fatal("previewForEntry() ok = false, want true")
	}
	if preview.CacheKey != "msg-1:richlink-cover" {
		t.Fatalf("CacheKey = %q, want %q", preview.CacheKey, "msg-1:richlink-cover")
	}
	if preview.Name != "Example article" {
		t.Fatalf("Name = %q, want %q", preview.Name, "Example article")
	}
	if !bytes.Equal(preview.Bytes, coverBytes) {
		t.Fatalf("Bytes = %v, want %v", preview.Bytes, coverBytes)
	}
}

func TestPreviewForEntryRichlinkCoverRejectsInvalidPreview(t *testing.T) {
	tests := []struct {
		name    string
		content protocol.Content
	}{
		{
			name: "missing cover",
			content: protocol.Content{
				Type: "richlink",
				Url:  "https://example.com/article",
			},
		},
		{
			name: "unsupported mime",
			content: protocol.Content{
				Type: "richlink",
				Url:  "https://example.com/article",
				Cover: &protocol.Cover{
					MimeType: "text/plain",
					Bytes:    base64.StdEncoding.EncodeToString([]byte("hello")),
				},
			},
		},
		{
			name: "invalid base64",
			content: protocol.Content{
				Type: "richlink",
				Url:  "https://example.com/article",
				Cover: &protocol.Cover{
					MimeType: "image/png",
					Bytes:    "not-base64",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := previewForEntry(store.LogEntry{ID: "msg-1", Content: tt.content}); ok {
				t.Fatal("previewForEntry() ok = true, want false")
			}
		})
	}
}
