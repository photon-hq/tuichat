package ui

import (
	"encoding/base64"

	"github.com/photon-hq/tuichat/internal/kitty"
	"github.com/photon-hq/tuichat/internal/protocol"
	"github.com/photon-hq/tuichat/internal/store"
)

func richlinkLabel(c protocol.Content) string {
	if c.Title != "" {
		return c.Title
	}
	if c.Url != "" {
		return c.Url
	}
	return ""
}

func entrySupportsPreview(e store.LogEntry) bool {
	switch e.Content.Type {
	case "attachment":
		return kitty.SupportedMimeType(e.Content.MimeType)
	case "richlink":
		if e.Content.Cover == nil || !kitty.SupportedMimeType(e.Content.Cover.MimeType) {
			return false
		}
		raw, err := base64.StdEncoding.DecodeString(e.Content.Cover.Bytes)
		return err == nil && len(raw) > 0
	default:
		return false
	}
}

func previewForEntry(e store.LogEntry) (*store.HoveredPreview, bool) {
	switch e.Content.Type {
	case "attachment":
		if !entrySupportsPreview(e) {
			return nil, false
		}
		return &store.HoveredPreview{
			CacheKey: attachmentPreviewCacheKey(e),
			Name:     e.Content.Name,
			Path:     e.AttachmentPath,
		}, true
	case "richlink":
		if !entrySupportsPreview(e) {
			return nil, false
		}
		raw, err := base64.StdEncoding.DecodeString(e.Content.Cover.Bytes)
		if err != nil || len(raw) == 0 {
			return nil, false
		}
		name := richlinkLabel(e.Content)
		if name == "" {
			name = "link preview"
		}
		return &store.HoveredPreview{
			CacheKey: e.ID + ":richlink-cover",
			Name:     name,
			Bytes:    raw,
		}, true
	default:
		return nil, false
	}
}

func attachmentPreviewCacheKey(e store.LogEntry) string {
	if e.AttachmentPath != "" {
		return e.AttachmentPath + ":" + e.Content.Name
	}
	if e.ID != "" {
		return e.ID + ":attachment"
	}
	return e.Content.Name
}
