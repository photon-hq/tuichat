package protocol

import (
	"encoding/json"
	"testing"
)

func TestContentUnmarshalRichlinkCover(t *testing.T) {
	var content Content
	err := json.Unmarshal([]byte(`{
		"type": "richlink",
		"url": "https://example.com/article",
		"title": "Example",
		"cover": {
			"mimeType": "image/png",
			"bytes": "aGVsbG8="
		}
	}`), &content)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if content.Cover == nil {
		t.Fatal("Cover = nil, want value")
	}
	if content.Cover.MimeType != "image/png" {
		t.Fatalf("Cover.MimeType = %q, want %q", content.Cover.MimeType, "image/png")
	}
	if content.Cover.Bytes != "aGVsbG8=" {
		t.Fatalf("Cover.Bytes = %q, want %q", content.Cover.Bytes, "aGVsbG8=")
	}
}
