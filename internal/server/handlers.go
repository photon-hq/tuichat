// Package server wires incoming JSON-RPC requests into the store,
// and pumps user-input from the store out as `message` notifications.
package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/photon-hq/tuichat/internal/protocol"
	"github.com/photon-hq/tuichat/internal/spool"
	"github.com/photon-hq/tuichat/internal/store"
	"github.com/photon-hq/tuichat/internal/ui"
	"github.com/photon-hq/tuichat/internal/version"
)

// Server holds the references needed to handle incoming RPC + push outgoing notifs.
type Server struct {
	Store   *store.Store
	Session *protocol.Session
	Program *tea.Program
}

// HandleRequest dispatches a request method to the store or returns an error.
func (s *Server) HandleRequest(method string, raw json.RawMessage) (any, error) {
	switch method {
	case "initialize":
		var p protocol.InitializeParams
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("initialize params: %w", err)
			}
		}
		// Re-seed the store's command list in case the adapter passed commands.
		s.Store.SetCommands(p.Commands)
		s.Store.NewChat()
		if s.Program != nil {
			s.Program.Send(ui.StoreChangedMsg{})
		}
		return protocol.InitializeResult{
			ProtocolVersion: protocol.Version,
			ServerInfo: protocol.ServerInfo{
				Name:    "tuichat",
				Version: version.Version,
			},
		}, nil

	case "send":
		var p protocol.SendParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		attachmentPath := resolveAttachmentPath(p.Content)
		s.Store.EnsureChat(p.SpaceID)
		s.Store.AppendAgent(p.SpaceID, resolveContent(p.Content, attachmentPath), "", attachmentPath)
		s.notifyStoreChanged()
		return protocol.SendResult{ID: newID(), Timestamp: iso()}, nil

	case "replyToMessage":
		var p protocol.ReplyParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		attachmentPath := resolveAttachmentPath(p.Content)
		s.Store.EnsureChat(p.SpaceID)
		s.Store.AppendAgent(p.SpaceID, resolveContent(p.Content, attachmentPath), p.MessageID, attachmentPath)
		s.notifyStoreChanged()
		return protocol.SendResult{ID: newID(), Timestamp: iso()}, nil

	case "startTyping":
		var p protocol.TypingParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		s.Store.SetTyping(p.SpaceID, true)
		s.notifyStoreChanged()
		return nil, nil

	case "stopTyping":
		var p protocol.TypingParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		s.Store.SetTyping(p.SpaceID, false)
		s.notifyStoreChanged()
		return nil, nil

	case "reactToMessage":
		var p protocol.ReactParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		s.Store.React(p.SpaceID, p.MessageID, p.Reaction)
		s.notifyStoreChanged()
		return nil, nil

	case "ensureSpace":
		var p protocol.EnsureSpaceParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		s.Store.EnsureChat(p.ID)
		s.notifyStoreChanged()
		return nil, nil

	case "shutdown":
		if s.Program != nil {
			s.Program.Quit()
		}
		s.Session.Close()
		return nil, nil
	}

	return nil, errors.New("unknown method: " + method)
}

// PumpUserInput blocks reading from the store's input queue and emits a
// `message` notification for each user-submitted input. Runs in a goroutine.
func (s *Server) PumpUserInput() {
	for {
		chatID, content, ok := s.Store.NextUserInput()
		if !ok {
			return
		}
		_ = s.Session.Notify("message", protocol.MessageNotification{
			ID:        newID(),
			SpaceID:   chatID,
			SenderID:  "terminal-tui-user",
			Content:   outgoingContent(content),
			Timestamp: iso(),
		})
	}
}

func (s *Server) notifyStoreChanged() {
	if s.Program != nil {
		s.Program.Send(ui.StoreChangedMsg{})
	}
}

// --- content conversion helpers ---

// resolveAttachmentPath writes agent-sent attachment bytes to tmpdir, returns
// the resulting path. For attachments already carrying a path, returns that.
func resolveAttachmentPath(c protocol.Content) string {
	if c.Type != "attachment" && c.Type != "voice" {
		return ""
	}
	if c.Path != "" {
		return c.Path
	}
	if c.Bytes == "" {
		return ""
	}
	raw, err := decodeBase64(c.Bytes)
	if err != nil {
		return ""
	}
	return spool.Attachment(c.Name, raw)
}

// resolveContent normalizes incoming content — populates Path so MessageItem
// can use the unified attachmentPath without re-decoding bytes.
func resolveContent(c protocol.Content, path string) protocol.Content {
	if path != "" && (c.Type == "attachment" || c.Type == "voice") {
		c.Path = path
		c.Bytes = "" // don't hold duplicate bytes in memory
	}
	return c
}

// outgoingContent prepares content for a `message` notification to the adapter.
// When the user drops a file, we have a path; send it without base64'ing bytes.
// For text/custom/contact, just pass through.
func outgoingContent(c protocol.Content) protocol.Content {
	return c
}

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func iso() string {
	return time.Now().UTC().Format(time.RFC3339)
}
