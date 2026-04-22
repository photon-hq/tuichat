import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { useKeyboard, useRenderer } from "@opentui/react";
import { attachment } from "spectrum-ts";
import { extractDroppedPaths, resolveDroppedAttachment } from "../drop";
import type { ChatState, CommandDef, Store } from "../store";
import { Attachments } from "./Attachments";
import { ErrorBoundary } from "./ErrorBoundary";
import { KittyImage } from "./KittyImage";
import { MessageItem } from "./MessageItem";
import { Sidebar } from "./Sidebar";
import { Suggestions } from "./Suggestions";
import { theme } from "./theme";

interface AppProps {
  store: Store;
}

const RESERVED_COMMANDS: CommandDef[] = [
  { name: "/new", description: "start a new chat" },
  { name: "/help", description: "show keybindings and env vars" },
];

function filterCommands(
  commands: readonly CommandDef[],
  prefix: string
): CommandDef[] {
  if (!prefix.startsWith("/")) return [];
  const lower = prefix.toLowerCase();
  const merged = [...RESERVED_COMMANDS, ...commands];
  return merged.filter((c) => c.name.toLowerCase().startsWith(lower));
}

const HELP_LINES = [
  "tuichat — keybindings",
  "  Ctrl+N         new chat",
  "  Ctrl+J / K     cycle chats (down / up)",
  "  Ctrl+L         clear active chat (adds a system marker)",
  "  Ctrl+D         toggle debug overlay",
  "  Ctrl+C         exit",
  "  Tab            complete slash command",
  "  Esc            cancel input + drop pending attachments",
  "  drag file      attach (or paste its path)",
  "  hover image    floating preview (Kitty/Ghostty)",
  "slash commands",
  "  /new           start a new chat",
  "  /help          this message",
  "environment variables",
  "  TUICHAT_FORCE_TUI=1        force rich TUI even without a TTY",
  "  TUICHAT_FORCE_PLAIN=1      force plain readline mode",
  "  TUICHAT_QUIET=1            silence the plain-mode startup banner",
  "  TUICHAT_DISABLE_IMAGES=1   disable Kitty graphics image previews",
  "  TUICHAT_DEBUG_IMAGES=1     log APC sequences to /tmp/tuichat-images.log",
  "  TUICHAT_DEBUG_LOG=<path>   override the debug log path",
];

const EMPTY_PENDING: readonly never[] = [];

function activeChat(
  chats: readonly ChatState[],
  activeId: string | null
): ChatState | null {
  if (!activeId) return null;
  return chats.find((c) => c.id === activeId) ?? null;
}

export function App({ store }: AppProps) {
  const renderer = useRenderer();
  const snapshot = useSyncExternalStore(store.subscribe, store.getSnapshot);
  const active = activeChat(snapshot.chats, snapshot.activeChatId);
  const activeId = active?.id ?? null;

  const [inputValue, setInputValue] = useState(active?.inputDraft ?? "");
  const [prefix, setPrefix] = useState(active?.inputDraft ?? "");
  const [tabIndex, setTabIndex] = useState(0);

  const inputRef = useRef(inputValue);
  inputRef.current = inputValue;
  const prefixRef = useRef(prefix);
  prefixRef.current = prefix;
  const tabIndexRef = useRef(tabIndex);
  tabIndexRef.current = tabIndex;
  const activeIdRef = useRef(activeId);
  activeIdRef.current = activeId;

  useEffect(() => {
    const draft = active?.inputDraft ?? "";
    setInputValue(draft);
    setPrefix(draft);
    setTabIndex(0);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeId]);

  const commands = snapshot.commands;
  const matches = useMemo(
    () => filterCommands(commands, prefix),
    [commands, prefix]
  );

  const handleInput = useCallback(
    (value: string) => {
      const currentId = activeIdRef.current;
      if (!currentId) return;
      const { cleaned, attachments } = extractDroppedPaths(value);
      if (attachments.length > 0) {
        for (const a of attachments) {
          store.addPendingAttachment(currentId, a);
        }
      }
      setInputValue(cleaned);
      setPrefix(cleaned);
      setTabIndex(0);
      store.setInputDraft(currentId, cleaned);
    },
    [store]
  );

  const handleSubmit = useCallback(() => {
    const currentId = activeIdRef.current;
    if (!currentId) return;
    const text = inputRef.current.trim();

    if (text === "/new") {
      store.setInputDraft(currentId, "");
      store.newChat();
      setInputValue("");
      setPrefix("");
      setTabIndex(0);
      return;
    }

    if (text === "/help") {
      store.setInputDraft(currentId, "");
      for (const line of HELP_LINES) {
        store.appendSystem(currentId, line);
      }
      setInputValue("");
      setPrefix("");
      setTabIndex(0);
      return;
    }

    const chatSnapshot = store
      .getSnapshot()
      .chats.find((c) => c.id === currentId);
    const pendings = chatSnapshot?.pendingAttachments ?? EMPTY_PENDING;
    if (text.length === 0 && pendings.length === 0) return;

    (async () => {
      for (const p of pendings) {
        try {
          const content = await attachment(p.path, { name: p.name }).build();
          store.appendUser(currentId, content, { attachmentPath: p.path });
          store.pushUserInput(currentId, content);
        } catch {
          store.appendSystem(currentId, `failed to attach "${p.name}"`);
        }
      }

      if (text.length > 0) {
        const content = { type: "text" as const, text };
        store.appendUser(currentId, content);
        store.pushUserInput(currentId, content);
      }
    })();

    store.clearPendingAttachments(currentId);
    store.setInputDraft(currentId, "");
    setInputValue("");
    setPrefix("");
    setTabIndex(0);
  }, [store]);

  useKeyboard((key) => {
    if (key.ctrl && key.name === "c") {
      process.kill(process.pid, "SIGINT");
      return;
    }
    if (key.ctrl && key.name === "n") {
      store.newChat();
      return;
    }
    if (key.ctrl && key.name === "j") {
      store.cycleActiveChat(1);
      return;
    }
    if (key.ctrl && key.name === "k") {
      store.cycleActiveChat(-1);
      return;
    }
    if (key.ctrl && key.name === "l") {
      const id = activeIdRef.current;
      if (id) store.appendSystem(id, "screen cleared");
      return;
    }
    if (key.ctrl && key.name === "d") {
      renderer?.toggleDebugOverlay();
      return;
    }

    if (key.name === "tab") {
      const current = filterCommands(commands, prefixRef.current);
      if (current.length === 0) return;
      const idx = tabIndexRef.current % current.length;
      const picked = current[idx]!;
      setInputValue(picked.name);
      setTabIndex((t) => t + 1);
      return;
    }

    if (key.name === "escape") {
      const id = activeIdRef.current;
      if (id) {
        store.clearPendingAttachments(id);
        store.setInputDraft(id, "");
      }
      setInputValue("");
      setPrefix("");
      setTabIndex(0);
      return;
    }
  });

  useEffect(() => {
    if (!renderer) return;
    const handler = (event: {
      bytes: Uint8Array;
      preventDefault: () => void;
      stopPropagation: () => void;
    }) => {
      const currentId = activeIdRef.current;
      if (!currentId) return;
      const raw = new TextDecoder().decode(event.bytes);
      const resolved = resolveDroppedAttachment(raw);
      if (!resolved) return;
      event.preventDefault();
      event.stopPropagation();
      store.addPendingAttachment(currentId, resolved);
    };
    renderer.keyInput.on("paste", handler);
    return () => {
      renderer.keyInput.off("paste", handler);
    };
  }, [renderer, store]);

  const entries = active?.entries ?? [];
  const typing = active?.typing ?? false;
  const pendingAttachments = active?.pendingAttachments ?? EMPTY_PENDING;

  const showSuggestions = matches.length > 0 && prefix.startsWith("/");
  const suggestionRows = showSuggestions
    ? Math.min(matches.length, 5) + (matches.length > 5 ? 1 : 0)
    : 0;
  const attachmentRows = pendingAttachments.length > 0 ? 1 : 0;
  const inputContainerHeight = 2 + attachmentRows + suggestionRows + 1;

  const hovered = snapshot.hoveredPreview;

  return (
    <box style={{ flexDirection: "row", flexGrow: 1, height: "100%" }}>
      {hovered ? (
        <box
          style={{
            position: "absolute",
            top: 2,
            right: 2,
            flexDirection: "column",
            backgroundColor: theme.colors.suggestionBg,
            border: true,
            borderStyle: "rounded",
            borderColor: theme.colors.border,
            paddingLeft: 1,
            paddingRight: 1,
            zIndex: 100,
          }}
        >
          <box style={{ height: 1 }}>
            <text>
              <span style={{ fg: theme.colors.attachment }}>
                {`📎 ${hovered.name}`}
              </span>
            </text>
          </box>
          <KittyImage
            read={hovered.read}
            cacheKey={hovered.cacheKey}
            cols={40}
            rows={10}
          />
        </box>
      ) : null}

      <Sidebar
        chats={snapshot.chats}
        activeChatId={snapshot.activeChatId}
        store={store}
      />

      <box
        style={{
          flexDirection: "column",
          flexGrow: 1,
          height: "100%",
        }}
      >
        <box
          style={{
            height: 1,
            paddingLeft: 1,
            paddingRight: 1,
            backgroundColor: theme.colors.border,
          }}
        >
          <text>
            <span style={{ fg: theme.colors.user }}>
              {` ${theme.title}${activeId ? ` · ${activeId}` : ""} `}
            </span>
            <span style={{ fg: theme.colors.system }}>
              {" — Ctrl+N new · Ctrl+J/K nav · Ctrl+C exit · Ctrl+L clear · Tab complete · Esc cancel"}
            </span>
          </text>
        </box>

        <scrollbox
          focused
          style={{
            flexGrow: 1,
            paddingLeft: 1,
            paddingRight: 1,
            stickyScroll: true,
            stickyStart: "bottom",
            contentOptions: {
              flexDirection: "column",
              justifyContent: "flex-end",
              minHeight: "100%",
            },
            scrollbarOptions: {
              visible: true,
            },
          }}
        >
          {active && active.droppedCount > 0 ? (
            <text>
              <span style={{ fg: theme.colors.system }}>
                {`… ${active.droppedCount} older message${active.droppedCount === 1 ? "" : "s"} dropped`}
              </span>
            </text>
          ) : null}
          {activeId
            ? entries.map((entry) => (
                <MessageItem
                  key={entry.id}
                  entry={entry}
                  entries={entries}
                  chatId={activeId}
                  store={store}
                />
              ))
            : null}
        </scrollbox>

        <box
          style={{
            height: 1,
            paddingLeft: 1,
            paddingRight: 1,
            shouldFill: false,
          }}
        >
          {typing ? (
            <text>
              <span style={{ fg: theme.colors.typing }}>
                {"● agent is typing…"}
              </span>
            </text>
          ) : null}
        </box>

        <box
          style={{
            border: true,
            borderStyle: "rounded",
            borderColor: theme.colors.border,
            flexDirection: "column",
            flexShrink: 0,
            height: inputContainerHeight,
            paddingLeft: 1,
            paddingRight: 1,
          }}
        >
          {pendingAttachments.length > 0 ? (
            <Attachments pending={pendingAttachments} />
          ) : null}
          {showSuggestions ? (
            <Suggestions matches={matches} selectedIndex={tabIndex} />
          ) : null}
          <box style={{ height: 1, flexShrink: 0 }}>
            <input
              focused
              placeholder={
                activeId
                  ? "type a message and press enter…"
                  : "Ctrl+N to start a new chat"
              }
              value={inputValue}
              onInput={handleInput}
              onSubmit={handleSubmit}
              style={{
                textColor: theme.colors.input,
                placeholderColor: theme.colors.system,
                cursorColor: theme.colors.prompt,
              }}
            />
          </box>
        </box>
      </box>
    </box>
  );
}

interface MountedAppProps extends AppProps {
  onFatalError?: (error: unknown) => void;
}

export function MountedApp({ store, onFatalError }: MountedAppProps) {
  useEffect(() => {
    return () => {
      store.closeInput();
    };
  }, [store]);
  return (
    <ErrorBoundary
      onError={(error) => {
        onFatalError?.(error);
      }}
    >
      <App store={store} />
    </ErrorBoundary>
  );
}
