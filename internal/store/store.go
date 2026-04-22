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
	inputQueue     []userInput
	inputClosed    bool
	inputWaiters   []chan userInput
}

type userInput struct {
	ChatID  string
	Content protocol.Content
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

// SortedChats returns chats ordered by lastActivityAt descending.
func (s *Store) SortedChats() []ChatState {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ChatState, 0, len(s.chats))
	for _, c := range s.chats {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastActivityAt.After(out[j].LastActivityAt)
	})
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

func (s *Store) AppendSystem(chatID, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureChatLocked(chatID)
	s.appendLocked(chatID, RoleSystem, protocol.Content{Type: "text", Text: text}, "", "")
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

// PushUserInput queues a user-originated message to be emitted as a protocol
// `message` notification. Safe to call from any goroutine.
func (s *Store) PushUserInput(chatID string, content protocol.Content) {
	s.mu.Lock()
	if s.inputClosed {
		s.mu.Unlock()
		return
	}
	item := userInput{ChatID: chatID, Content: content}
	if len(s.inputWaiters) > 0 {
		ch := s.inputWaiters[0]
		s.inputWaiters = s.inputWaiters[1:]
		s.mu.Unlock()
		ch <- item
		return
	}
	s.inputQueue = append(s.inputQueue, item)
	s.mu.Unlock()
}

// NextUserInput blocks until a user input is available. Returns (_, false) when closed.
func (s *Store) NextUserInput() (string, protocol.Content, bool) {
	s.mu.Lock()
	if s.inputClosed {
		s.mu.Unlock()
		return "", protocol.Content{}, false
	}
	if len(s.inputQueue) > 0 {
		item := s.inputQueue[0]
		s.inputQueue = s.inputQueue[1:]
		s.mu.Unlock()
		return item.ChatID, item.Content, true
	}
	ch := make(chan userInput, 1)
	s.inputWaiters = append(s.inputWaiters, ch)
	s.mu.Unlock()

	item, ok := <-ch
	if !ok {
		return "", protocol.Content{}, false
	}
	return item.ChatID, item.Content, true
}

func (s *Store) CloseInput() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inputClosed {
		return
	}
	s.inputClosed = true
	for _, ch := range s.inputWaiters {
		close(ch)
	}
	s.inputWaiters = nil
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
