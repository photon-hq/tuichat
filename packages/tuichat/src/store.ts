import type { Content } from "spectrum-ts";

export interface CommandDef {
  name: string;
  description?: string;
}

export type Role = "user" | "agent" | "system";

export interface LogEntry {
  id: string;
  role: Role;
  content: Content;
  timestamp: Date;
  replyTo?: string;
  reactions: string[];
  attachmentPath?: string;
}

export interface PendingAttachment {
  path: string;
  name: string;
  size?: number;
}

export interface HoveredPreview {
  cacheKey: string;
  name: string;
  read: () => Promise<Buffer>;
}

export interface ChatState {
  id: string;
  entries: readonly LogEntry[];
  droppedCount: number;
  typing: boolean;
  pendingAttachments: readonly PendingAttachment[];
  inputDraft: string;
  lastActivityAt: Date;
  createdAt: Date;
}

export interface Snapshot {
  chats: readonly ChatState[];
  activeChatId: string | null;
  commands: readonly CommandDef[];
  hoveredPreview: HoveredPreview | null;
}

interface UserInputItem {
  chatId: string;
  content: Content;
}

type Listener = () => void;

interface PendingInput {
  resolve: (value: IteratorResult<UserInputItem>) => void;
}

export interface Store {
  subscribe(listener: Listener): () => void;
  getSnapshot(): Snapshot;

  newChat(): string;
  ensureChat(id: string): void;
  setActiveChat(id: string): void;
  cycleActiveChat(delta: 1 | -1): void;

  appendAgent(
    chatId: string,
    content: Content,
    opts?: { replyTo?: string; attachmentPath?: string }
  ): string;
  appendUser(
    chatId: string,
    content: Content,
    opts?: { attachmentPath?: string }
  ): string;
  appendSystem(chatId: string, text: string): void;
  setTyping(chatId: string, value: boolean): void;
  react(chatId: string, messageId: string, emoji: string): void;
  patchEntry(chatId: string, messageId: string, patch: Partial<LogEntry>): void;

  pushUserInput(chatId: string, content: Content): void;
  nextUserInput(): Promise<IteratorResult<UserInputItem>>;
  closeInput(): void;

  addPendingAttachment(chatId: string, att: PendingAttachment): void;
  removePendingAttachment(chatId: string, index: number): void;
  clearPendingAttachments(chatId: string): void;

  setInputDraft(chatId: string, value: string): void;
  setHoveredPreview(preview: HoveredPreview | null): void;
}

const newId = (): string => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `id-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
};

const SCROLLBACK_CAP = 2000;

function emptyChatState(id: string): ChatState {
  const now = new Date();
  return {
    id,
    entries: [],
    droppedCount: 0,
    typing: false,
    pendingAttachments: [],
    inputDraft: "",
    lastActivityAt: now,
    createdAt: now,
  };
}

function sortChats(chats: readonly ChatState[]): readonly ChatState[] {
  return [...chats].sort(
    (a, b) => b.lastActivityAt.getTime() - a.lastActivityAt.getTime()
  );
}

export function createStore(options?: {
  commands?: readonly CommandDef[];
}): Store {
  const chats = new Map<string, ChatState>();
  let activeChatId: string | null = null;
  let hoveredPreview: HoveredPreview | null = null;
  const commands: readonly CommandDef[] = options?.commands ?? [];
  let nextChatIndex = 1;

  const inputBuffer: UserInputItem[] = [];
  const waiters: PendingInput[] = [];
  let inputClosed = false;

  let snapshot: Snapshot = {
    chats: [],
    activeChatId: null,
    commands,
    hoveredPreview,
  };
  const listeners = new Set<Listener>();

  const commit = () => {
    snapshot = {
      chats: sortChats(Array.from(chats.values())),
      activeChatId,
      commands,
      hoveredPreview,
    };
    for (const l of listeners) l();
  };

  const updateChat = (
    id: string,
    updater: (chat: ChatState) => ChatState
  ): boolean => {
    const chat = chats.get(id);
    if (!chat) return false;
    chats.set(id, updater(chat));
    return true;
  };

  const generateChatId = (): string => {
    while (chats.has(`chat-${nextChatIndex}`)) nextChatIndex += 1;
    const id = `chat-${nextChatIndex}`;
    nextChatIndex += 1;
    return id;
  };

  const ensureChatInternal = (id: string): boolean => {
    if (chats.has(id)) return false;
    chats.set(id, emptyChatState(id));
    return true;
  };

  const append = (
    chatId: string,
    role: Role,
    content: Content,
    opts?: { replyTo?: string; attachmentPath?: string }
  ): string => {
    const entryId = newId();
    const entry: LogEntry = {
      id: entryId,
      role,
      content,
      timestamp: new Date(),
      replyTo: opts?.replyTo,
      reactions: [],
      attachmentPath: opts?.attachmentPath,
    };
    ensureChatInternal(chatId);
    updateChat(chatId, (c) => {
      const next = [...c.entries, entry];
      const overflow = Math.max(0, next.length - SCROLLBACK_CAP);
      return {
        ...c,
        entries: overflow > 0 ? next.slice(overflow) : next,
        droppedCount: c.droppedCount + overflow,
        lastActivityAt: entry.timestamp,
      };
    });
    commit();
    return entryId;
  };

  return {
    subscribe(listener) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },

    getSnapshot() {
      return snapshot;
    },

    newChat() {
      const id = generateChatId();
      ensureChatInternal(id);
      activeChatId = id;
      commit();
      return id;
    },

    ensureChat(id) {
      const created = ensureChatInternal(id);
      if (created) {
        if (activeChatId === null) activeChatId = id;
        commit();
      }
    },

    setActiveChat(id) {
      if (!chats.has(id)) return;
      if (activeChatId === id) return;
      activeChatId = id;
      commit();
    },

    cycleActiveChat(delta) {
      const sorted = sortChats(Array.from(chats.values()));
      if (sorted.length === 0) return;
      const idx = sorted.findIndex((c) => c.id === activeChatId);
      const nextIdx =
        idx < 0 ? 0 : (idx + delta + sorted.length) % sorted.length;
      const next = sorted[nextIdx]!;
      if (next.id === activeChatId) return;
      activeChatId = next.id;
      commit();
    },

    appendAgent(chatId, content, opts) {
      return append(chatId, "agent", content, {
        replyTo: opts?.replyTo,
        attachmentPath: opts?.attachmentPath,
      });
    },

    appendUser(chatId, content, opts) {
      return append(chatId, "user", content, {
        attachmentPath: opts?.attachmentPath,
      });
    },

    appendSystem(chatId, text) {
      append(chatId, "system", { type: "text", text });
    },

    setTyping(chatId, value) {
      const chat = chats.get(chatId);
      if (!chat || chat.typing === value) return;
      updateChat(chatId, (c) => ({ ...c, typing: value }));
      commit();
    },

    react(chatId, messageId, emoji) {
      const chat = chats.get(chatId);
      if (!chat) return;
      const idx = chat.entries.findIndex((e) => e.id === messageId);
      if (idx < 0) return;
      const entry = chat.entries[idx]!;
      updateChat(chatId, (c) => ({
        ...c,
        entries: [
          ...c.entries.slice(0, idx),
          { ...entry, reactions: [...entry.reactions, emoji] },
          ...c.entries.slice(idx + 1),
        ],
      }));
      commit();
    },

    patchEntry(chatId, messageId, patch) {
      const chat = chats.get(chatId);
      if (!chat) return;
      const idx = chat.entries.findIndex((e) => e.id === messageId);
      if (idx < 0) return;
      const entry = chat.entries[idx]!;
      updateChat(chatId, (c) => ({
        ...c,
        entries: [
          ...c.entries.slice(0, idx),
          { ...entry, ...patch },
          ...c.entries.slice(idx + 1),
        ],
      }));
      commit();
    },

    pushUserInput(chatId, content) {
      if (inputClosed) return;
      const item: UserInputItem = { chatId, content };
      const waiter = waiters.shift();
      if (waiter) {
        waiter.resolve({ value: item, done: false });
      } else {
        inputBuffer.push(item);
      }
    },

    nextUserInput() {
      if (inputClosed) {
        return Promise.resolve<IteratorResult<UserInputItem>>({
          value: undefined,
          done: true,
        });
      }
      const buffered = inputBuffer.shift();
      if (buffered !== undefined) {
        return Promise.resolve<IteratorResult<UserInputItem>>({
          value: buffered,
          done: false,
        });
      }
      return new Promise<IteratorResult<UserInputItem>>((resolve) => {
        waiters.push({ resolve });
      });
    },

    closeInput() {
      if (inputClosed) return;
      inputClosed = true;
      while (waiters.length > 0) {
        const w = waiters.shift();
        w?.resolve({ value: undefined, done: true });
      }
    },

    addPendingAttachment(chatId, att) {
      const chat = chats.get(chatId);
      if (!chat) return;
      if (chat.pendingAttachments.some((a) => a.path === att.path)) return;
      updateChat(chatId, (c) => ({
        ...c,
        pendingAttachments: [...c.pendingAttachments, att],
      }));
      commit();
    },

    removePendingAttachment(chatId, index) {
      const chat = chats.get(chatId);
      if (!chat) return;
      if (index < 0 || index >= chat.pendingAttachments.length) return;
      updateChat(chatId, (c) => ({
        ...c,
        pendingAttachments: [
          ...c.pendingAttachments.slice(0, index),
          ...c.pendingAttachments.slice(index + 1),
        ],
      }));
      commit();
    },

    clearPendingAttachments(chatId) {
      const chat = chats.get(chatId);
      if (!chat || chat.pendingAttachments.length === 0) return;
      updateChat(chatId, (c) => ({ ...c, pendingAttachments: [] }));
      commit();
    },

    setInputDraft(chatId, value) {
      const chat = chats.get(chatId);
      if (!chat || chat.inputDraft === value) return;
      updateChat(chatId, (c) => ({ ...c, inputDraft: value }));
      commit();
    },

    setHoveredPreview(preview) {
      if (
        hoveredPreview === null &&
        preview === null
      ) {
        return;
      }
      if (
        hoveredPreview !== null &&
        preview !== null &&
        hoveredPreview.cacheKey === preview.cacheKey
      ) {
        return;
      }
      hoveredPreview = preview;
      commit();
    },
  };
}

