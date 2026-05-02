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

func TestPreviewForEntryAttachmentCacheKeyUsesPath(t *testing.T) {
	entry := store.LogEntry{
		ID:             "msg-1",
		AttachmentPath: "/tmp/tuichat-a/image.png",
		Content: protocol.Content{
			Type:     "attachment",
			Name:     "image.png",
			MimeType: "image/png",
		},
	}

	preview, ok := previewForEntry(entry)
	if !ok {
		t.Fatal("previewForEntry() ok = false, want true")
	}
	if preview.CacheKey != "/tmp/tuichat-a/image.png:image.png" {
		t.Fatalf("CacheKey = %q, want %q", preview.CacheKey, "/tmp/tuichat-a/image.png:image.png")
	}
	if preview.Name != "image.png" {
		t.Fatalf("Name = %q, want %q", preview.Name, "image.png")
	}
	if preview.Path != "/tmp/tuichat-a/image.png" {
		t.Fatalf("Path = %q, want %q", preview.Path, "/tmp/tuichat-a/image.png")
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
			if entrySupportsPreview(store.LogEntry{ID: "msg-1", Content: tt.content}) {
				t.Fatal("entrySupportsPreview() = true, want false")
			}
			if _, ok := previewForEntry(store.LogEntry{ID: "msg-1", Content: tt.content}); ok {
				t.Fatal("previewForEntry() ok = true, want false")
			}
		})
	}
}
