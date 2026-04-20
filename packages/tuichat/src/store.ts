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

export interface Snapshot {
  entries: readonly LogEntry[];
  typing: boolean;
  commands: readonly CommandDef[];
  pendingAttachments: readonly PendingAttachment[];
  hoveredPreview: HoveredPreview | null;
}

type Listener = () => void;

interface PendingInput {
  resolve: (value: IteratorResult<Content>) => void;
  reject: (err: unknown) => void;
}

export interface Store {
  subscribe(listener: Listener): () => void;
  getSnapshot(): Snapshot;
  appendAgent(content: Content, opts?: { replyTo?: string }): string;
  appendUser(content: Content): string;
  appendSystem(text: string): void;
  setTyping(value: boolean): void;
  react(messageId: string, emoji: string): void;
  patchEntry(id: string, patch: Partial<LogEntry>): void;
  pushUserInput(content: Content): void;
  nextUserInput(): Promise<IteratorResult<Content>>;
  closeInput(): void;
  addPendingAttachment(att: PendingAttachment): void;
  removePendingAttachment(index: number): void;
  clearPendingAttachments(): void;
  setHoveredPreview(preview: HoveredPreview | null): void;
}

const newId = (): string => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `id-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
};

export function createStore(options?: {
  commands?: readonly CommandDef[];
}): Store {
  let entries: LogEntry[] = [];
  let typing = false;
  let pendingAttachments: PendingAttachment[] = [];
  let hoveredPreview: HoveredPreview | null = null;
  const commands: readonly CommandDef[] = options?.commands ?? [];
  let snapshot: Snapshot = {
    entries,
    typing,
    commands,
    pendingAttachments,
    hoveredPreview,
  };
  const listeners = new Set<Listener>();

  const inputBuffer: Content[] = [];
  const waiters: PendingInput[] = [];
  let inputClosed = false;

  const commit = () => {
    snapshot = {
      entries,
      typing,
      commands,
      pendingAttachments,
      hoveredPreview,
    };
    for (const l of listeners) l();
  };

  const append = (role: Role, content: Content, replyTo?: string): string => {
    const id = newId();
    entries = [
      ...entries,
      { id, role, content, timestamp: new Date(), replyTo, reactions: [] },
    ];
    commit();
    return id;
  };

  return {
    subscribe(listener) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },

    getSnapshot() {
      return snapshot;
    },

    appendAgent(content, opts) {
      return append("agent", content, opts?.replyTo);
    },

    appendUser(content) {
      return append("user", content);
    },

    appendSystem(text) {
      append("system", { type: "text", text });
    },

    setTyping(value) {
      if (typing === value) return;
      typing = value;
      commit();
    },

    react(messageId, emoji) {
      const idx = entries.findIndex((e) => e.id === messageId);
      if (idx < 0) return;
      const entry = entries[idx]!;
      entries = [
        ...entries.slice(0, idx),
        { ...entry, reactions: [...entry.reactions, emoji] },
        ...entries.slice(idx + 1),
      ];
      commit();
    },

    patchEntry(id, patch) {
      const idx = entries.findIndex((e) => e.id === id);
      if (idx < 0) return;
      const entry = entries[idx]!;
      entries = [
        ...entries.slice(0, idx),
        { ...entry, ...patch },
        ...entries.slice(idx + 1),
      ];
      commit();
    },

    pushUserInput(content) {
      if (inputClosed) return;
      const waiter = waiters.shift();
      if (waiter) {
        waiter.resolve({ value: content, done: false });
      } else {
        inputBuffer.push(content);
      }
    },

    nextUserInput() {
      if (inputClosed) {
        return Promise.resolve<IteratorResult<Content>>({
          value: undefined,
          done: true,
        });
      }
      const buffered = inputBuffer.shift();
      if (buffered !== undefined) {
        return Promise.resolve<IteratorResult<Content>>({
          value: buffered,
          done: false,
        });
      }
      return new Promise<IteratorResult<Content>>((resolve, reject) => {
        waiters.push({ resolve, reject });
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

    addPendingAttachment(att) {
      if (pendingAttachments.some((a) => a.path === att.path)) return;
      pendingAttachments = [...pendingAttachments, att];
      commit();
    },

    removePendingAttachment(index) {
      if (index < 0 || index >= pendingAttachments.length) return;
      pendingAttachments = [
        ...pendingAttachments.slice(0, index),
        ...pendingAttachments.slice(index + 1),
      ];
      commit();
    },

    clearPendingAttachments() {
      if (pendingAttachments.length === 0) return;
      pendingAttachments = [];
      commit();
    },

    setHoveredPreview(preview) {
      if (
        (hoveredPreview?.cacheKey ?? null) === (preview?.cacheKey ?? null) &&
        hoveredPreview === preview
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
