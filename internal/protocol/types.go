// Package protocol describes the JSON-RPC 2.0 message types tuichat speaks
// over TCP with its adapter. See PROTOCOL.md for the wire contract.
package protocol

import "encoding/json"

const Version = "1"

// Content is the discriminated union used in messages. The zero-value fields
// are omitted on the wire via omitempty.
type Content struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	Name     string          `json:"name,omitempty"`
	MimeType string          `json:"mimeType,omitempty"`
	Size     *int64          `json:"size,omitempty"`
	Bytes    string          `json:"bytes,omitempty"` // base64-encoded
	Path     string          `json:"path,omitempty"`
	Raw      json.RawMessage `json:"raw,omitempty"`
	// Contact fields deliberately passed through via Raw on the client side
	// when needed. We don't model contact/voice extensively in Go today —
	// they fall through as custom-like payloads.
}

type CommandDef struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type InitializeParams struct {
	Commands   []CommandDef `json:"commands,omitempty"`
	ClientInfo *ClientInfo  `json:"clientInfo,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"`
	ServerInfo      ServerInfo `json:"serverInfo"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type SendParams struct {
	SpaceID string  `json:"spaceId"`
	Content Content `json:"content"`
}

type TypingParams struct {
	SpaceID string `json:"spaceId"`
}

type ReactParams struct {
	SpaceID   string `json:"spaceId"`
	MessageID string `json:"messageId"`
	Reaction  string `json:"reaction"`
}

type ReplyParams struct {
	SpaceID   string  `json:"spaceId"`
	MessageID string  `json:"messageId"`
	Content   Content `json:"content"`
}

type EnsureSpaceParams struct {
	ID string `json:"id"`
}

type SendResult struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
}

type MessageNotification struct {
	ID        string      `json:"id"`
	SpaceID   string      `json:"spaceId"`
	SenderID  string      `json:"senderId"`
	Content   Content     `json:"content"`
	Timestamp string      `json:"timestamp"`
	ReplyTo   *MessageRef `json:"replyTo,omitempty"`
}

// MessageRef points at a previously-seen message. Used when a user submits a
// reply in the TUI — we echo the target messageId back to the adapter so the
// agent knows which of its messages is being replied to.
type MessageRef struct {
	MessageID string `json:"messageId"`
}

// ReactionNotification fires when the user reacts to a message in the TUI.
// Server → client (binary → adapter).
type ReactionNotification struct {
	SpaceID   string `json:"spaceId"`
	MessageID string `json:"messageId"`
	Reaction  string `json:"reaction"`
	SenderID  string `json:"senderId"`
	Timestamp string `json:"timestamp"`
}

// LogNotification carries agent-side console output into the UI. Client → server.
// Level is one of: "log", "info", "warn", "error", "debug".
type LogNotification struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// RPC envelope types. ID is any (number or string) per JSON-RPC 2.0.

type Request struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Notification struct {
	Jsonrpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RpcError       `json:"error,omitempty"`
}

type RpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error codes.
const (
	CodeParseError         = -32700
	CodeInvalidRequest     = -32600
	CodeMethodNotFound     = -32601
	CodeInvalidParams      = -32602
	CodeInternalError      = -32603
	CodeServerError        = -32000
	CodeUnsupportedContent = -32099
)
