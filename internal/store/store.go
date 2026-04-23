// Package store holds tuichat's reactive multi-chat state.
//
// The store is mutated from the UI's Bubbletea Update loop and from the
// protocol server's RPC handlers. Both happen on the tea.Program dispatch,
// so no mutex is needed in that call path. External goroutines (e.g. pump
// reading user input to emit notifications) acquire the mutex explicitly.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"github.com/photon-hq/tuichat/internal/protocol"
)

const ScrollbackCap = 2000

// SystemChatID is the pinned, always-present chat that receives agent-side
// console output forwarded via the `log` protocol notification.
const SystemChatID = "__system__"

type Role string

const (
	RoleUser   Role = "user"
	RoleAgent  Role = "agent"
	RoleSystem Role = "system"
)

type LogEntry struct {
	ID             string
	Role           Role
	Content        protocol.Content
	Timestamp      time.Time
	ReplyTo        string
	Reactions      []string
	AttachmentPath string
}

type PendingAttachment struct {
	Path string
	Name string
	Size int64
}

type ChatState struct {
	ID                 string
	Entries            []LogEntry
	DroppedCount       int
	Typing             bool
	PendingAttachments []PendingAttachment
	InputDraft         string
	LastActivityAt     time.Time
	CreatedAt          time.Time
}

type CommandDef = protocol.CommandDef

type HoveredPreview struct {
	CacheKey string
	Name     string
	Path     string // for file-backed previews
	Bytes    []byte // for in-memory previews
}

// Store holds all chat state plus ephemeral UI state (hovered preview).
type Store struct {
	mu             sync.Mutex
	chats          map[string]*ChatState
	activeChatID   string
	commands       []CommandDef
	hovered        *HoveredPreview
	nextChatIndex  int
	eventQueue     []UserEvent
	eventClosed    bool
	eventWaiters   []chan UserEvent
}

// UserEventKind discriminates what the pump goroutine will forward to the
// adapter — a user-authored message, or a reaction on an existing message.
type UserEventKind int

const (
	EventMessage UserEventKind = iota
	EventReaction
)

// UserEvent is a single outbound event drained by the server's pump.
type UserEvent struct {
	Kind      UserEventKind
	ChatID    string
	OwnID     string           // EventMessage: id the adapter will see on this message
	Content   protocol.Content // EventMessage
	ReplyTo   string           // EventMessage (empty = not a reply)
	MessageID string           // EventReaction: the target message id
	Reaction  string           // EventReaction
}

func New(commands []CommandDef) *Store {
	return &Store{
		chats:         map[string]*ChatState{},
		commands:      commands,
		nextChatIndex: 1,
	}
}

func (s *Store) Commands() []CommandDef {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]CommandDef, len(s.commands))
	copy(out, s.commands)
	return out
}

func (s *Store) SetCommands(cmds []CommandDef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands = append([]CommandDef(nil), cmds...)
}

// SortedChats returns user chats ordered by lastActivityAt descending. The
// pinned system chat (agent-side console output) is excluded — it's surfaced
// in a dedicated right-hand log column, not the sidebar.
func (s *Store) SortedChats() []ChatState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sortedLocked()
}

// SystemEntries returns a snapshot of the system chat's log entries, or nil
// if no log has been captured yet.
func (s *Store) SystemEntries() []LogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.chats[SystemChatID]
	if !ok || len(c.Entries) == 0 {
		return nil
	}
	out := make([]LogEntry, len(c.Entries))
	copy(out, c.Entries)
	return out
}

func (s *Store) ActiveChat() (ChatState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeChatID == "" {
		return ChatState{}, false
	}
	c, ok := s.chats[s.activeChatID]
	if !ok {
		return ChatState{}, false
	}
	return *c, true
}

func (s *Store) ActiveChatID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeChatID
}

// NewChat creates a chat with a generated id, sets it active, returns id.
func (s *Store) NewChat() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.generateIDLocked()
	s.chats[id] = emptyChat(id)
	s.activeChatID = id
	return id
}

// EnsureChat creates a chat with the given id if absent; sets active iff none active.
func (s *Store) EnsureChat(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[id]; !ok {
		s.chats[id] = emptyChat(id)
		if s.activeChatID == "" {
			s.activeChatID = id
		}
	}
}

func (s *Store) SetActiveChat(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[id]; !ok {
		return false
	}
	s.activeChatID = id
	return true
}

func (s *Store) CycleActiveChat(delta int) {
	s.mu.Lock()
	sorted := s.sortedLocked()
	if len(sorted) == 0 {
		s.mu.Unlock()
		return
	}
	idx := -1
	for i, c := range sorted {
		if c.ID == s.activeChatID {
			idx = i
			break
		}
	}
	if idx < 0 {
		idx = 0
	}
	next := ((idx+delta)%len(sorted) + len(sorted)) % len(sorted)
	s.activeChatID = sorted[next].ID
	s.mu.Unlock()
}

func (s *Store) sortedLocked() []ChatState {
	out := make([]ChatState, 0, len(s.chats))
	for _, c := range s.chats {
		if c.ID == SystemChatID {
			continue
		}
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastActivityAt.After(out[j].LastActivityAt)
	})
	return out
}

func (s *Store) AppendAgent(chatID string, content protocol.Content, replyTo, attachmentPath string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureChatLocked(chatID)
	return s.appendLocked(chatID, RoleAgent, content, replyTo, attachmentPath)
}

func (s *Store) AppendUser(chatID string, content protocol.Content, attachmentPath string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureChatLocked(chatID)
	return s.appendLocked(chatID, RoleUser, content, "", attachmentPath)
}

// AppendUserReply is like AppendUser but records that the entry quotes an
// earlier message, so the UI renders a quote stub above it.
func (s *Store) AppendUserReply(chatID string, content protocol.Content, replyTo, attachmentPath string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureChatLocked(chatID)
	return s.appendLocked(chatID, RoleUser, content, replyTo, attachmentPath)
}

func (s *Store) AppendSystem(chatID, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureChatLocked(chatID)
	s.appendLocked(chatID, RoleSystem, protocol.Content{Type: "text", Text: text}, "", "")
}

// AppendLog appends agent-side console output to the pinned system chat,
// creating it lazily on the first call. Does not steal focus from the active
// chat — the system chat only becomes active if the user navigates to it.
func (s *Store) AppendLog(level, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[SystemChatID]; !ok {
		s.chats[SystemChatID] = emptyChat(SystemChatID)
	}
	body := "[" + level + "] " + text
	s.appendLocked(SystemChatID, RoleSystem, protocol.Content{Type: "text", Text: body}, "", "")
}

func (s *Store) SetTyping(chatID string, v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.chats[chatID]; ok {
		c.Typing = v
	}
}

func (s *Store) React(chatID, messageID, emoji string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.chats[chatID]
	if !ok {
		return
	}
	for i := range c.Entries {
		if c.Entries[i].ID == messageID {
			c.Entries[i].Reactions = append(c.Entries[i].Reactions, emoji)
			return
		}
	}
}

func (s *Store) SetInputDraft(chatID, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.chats[chatID]; ok {
		c.InputDraft = value
	}
}

func (s *Store) AddPendingAttachment(chatID string, a PendingAttachment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.chats[chatID]
	if !ok {
		return
	}
	for _, existing := range c.PendingAttachments {
		if existing.Path == a.Path {
			return
		}
	}
	c.PendingAttachments = append(c.PendingAttachments, a)
}

func (s *Store) ClearPendingAttachments(chatID string) []PendingAttachment {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.chats[chatID]
	if !ok {
		return nil
	}
	out := c.PendingAttachments
	c.PendingAttachments = nil
	return out
}

func (s *Store) SetHoveredPreview(p *HoveredPreview) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hovered = p
}

func (s *Store) HoveredPreview() *HoveredPreview {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hovered
}

// PushUserInput queues a user-originated message. Safe to call from any goroutine.
// ownID should match the store entry's ID so agents that call `message.reply(...)`
// target the same id the renderer uses for quote lookup.
func (s *Store) PushUserInput(chatID, ownID string, content protocol.Content) {
	s.pushEvent(UserEvent{Kind: EventMessage, ChatID: chatID, OwnID: ownID, Content: content})
}

// PushUserReply queues a user-originated message that quotes an earlier one.
func (s *Store) PushUserReply(chatID, ownID string, content protocol.Content, replyTo string) {
	s.pushEvent(UserEvent{Kind: EventMessage, ChatID: chatID, OwnID: ownID, Content: content, ReplyTo: replyTo})
}

// PushUserReaction queues an emoji reaction against a previously-seen message.
func (s *Store) PushUserReaction(chatID, messageID, reaction string) {
	s.pushEvent(UserEvent{Kind: EventReaction, ChatID: chatID, MessageID: messageID, Reaction: reaction})
}

func (s *Store) pushEvent(evt UserEvent) {
	s.mu.Lock()
	if s.eventClosed {
		s.mu.Unlock()
		return
	}
	if len(s.eventWaiters) > 0 {
		ch := s.eventWaiters[0]
		s.eventWaiters = s.eventWaiters[1:]
		s.mu.Unlock()
		ch <- evt
		return
	}
	s.eventQueue = append(s.eventQueue, evt)
	s.mu.Unlock()
}

// NextUserEvent blocks until an event is available. Returns (_, false) when closed.
func (s *Store) NextUserEvent() (UserEvent, bool) {
	s.mu.Lock()
	if s.eventClosed {
		s.mu.Unlock()
		return UserEvent{}, false
	}
	if len(s.eventQueue) > 0 {
		evt := s.eventQueue[0]
		s.eventQueue = s.eventQueue[1:]
		s.mu.Unlock()
		return evt, true
	}
	ch := make(chan UserEvent, 1)
	s.eventWaiters = append(s.eventWaiters, ch)
	s.mu.Unlock()

	evt, ok := <-ch
	if !ok {
		return UserEvent{}, false
	}
	return evt, true
}

func (s *Store) CloseInput() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.eventClosed {
		return
	}
	s.eventClosed = true
	for _, ch := range s.eventWaiters {
		close(ch)
	}
	s.eventWaiters = nil
}

// --- helpers ---

func emptyChat(id string) *ChatState {
	now := time.Now()
	return &ChatState{
		ID:             id,
		LastActivityAt: now,
		CreatedAt:      now,
	}
}

func (s *Store) ensureChatLocked(id string) {
	if _, ok := s.chats[id]; !ok {
		s.chats[id] = emptyChat(id)
		if s.activeChatID == "" {
			s.activeChatID = id
		}
	}
}

func (s *Store) appendLocked(chatID string, role Role, content protocol.Content, replyTo, attachmentPath string) string {
	c := s.chats[chatID]
	entry := LogEntry{
		ID:             newID(),
		Role:           role,
		Content:        content,
		Timestamp:      time.Now(),
		ReplyTo:        replyTo,
		AttachmentPath: attachmentPath,
	}
	c.Entries = append(c.Entries, entry)
	if len(c.Entries) > ScrollbackCap {
		over := len(c.Entries) - ScrollbackCap
		c.Entries = c.Entries[over:]
		c.DroppedCount += over
	}
	c.LastActivityAt = entry.Timestamp
	return entry.ID
}

func (s *Store) generateIDLocked() string {
	for {
		id := "chat-" + itoa(s.nextChatIndex)
		s.nextChatIndex++
		if _, exists := s.chats[id]; !exists {
			return id
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
