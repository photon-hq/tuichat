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
import type { CommandDef, Store } from "../store";
import { Attachments } from "./Attachments";
import { KittyImage } from "./KittyImage";
import { MessageItem } from "./MessageItem";
import { Suggestions } from "./Suggestions";
import { theme } from "./theme";

interface AppProps {
  store: Store;
}

function filterCommands(
  commands: readonly CommandDef[],
  prefix: string
): CommandDef[] {
  if (!prefix.startsWith("/")) return [];
  const lower = prefix.toLowerCase();
  return commands.filter((c) => c.name.toLowerCase().startsWith(lower));
}

export function App({ store }: AppProps) {
  const renderer = useRenderer();
  const snapshot = useSyncExternalStore(store.subscribe, store.getSnapshot);
  const [inputValue, setInputValue] = useState("");
  const [prefix, setPrefix] = useState("");
  const [tabIndex, setTabIndex] = useState(0);

  const inputRef = useRef(inputValue);
  inputRef.current = inputValue;
  const prefixRef = useRef(prefix);
  prefixRef.current = prefix;
  const tabIndexRef = useRef(tabIndex);
  tabIndexRef.current = tabIndex;

  const commands = snapshot.commands;
  const matches = useMemo(
    () => filterCommands(commands, prefix),
    [commands, prefix]
  );

  const handleInput = useCallback(
    (value: string) => {
      const { cleaned, attachments } = extractDroppedPaths(value);
      if (attachments.length > 0) {
        for (const a of attachments) {
          store.addPendingAttachment(a);
        }
      }
      setInputValue(cleaned);
      setPrefix(cleaned);
      setTabIndex(0);
    },
    [store]
  );

  const handleSubmit = useCallback(() => {
    const text = inputRef.current.trim();
    const pendings = store.getSnapshot().pendingAttachments;
    if (text.length === 0 && pendings.length === 0) return;

    (async () => {
      for (const p of pendings) {
        try {
          const content = await attachment(p.path, { name: p.name }).build();
          store.appendUser(content, { attachmentPath: p.path });
          store.pushUserInput(content);
        } catch {
          store.appendSystem(`failed to attach "${p.name}"`);
        }
      }

      if (text.length > 0) {
        const content = { type: "text" as const, text };
        store.appendUser(content);
        store.pushUserInput(content);
      }
    })();

    store.clearPendingAttachments();
    setInputValue("");
    setPrefix("");
    setTabIndex(0);
  }, [store]);

  useKeyboard((key) => {
    if (key.ctrl && key.name === "c") {
      process.kill(process.pid, "SIGINT");
      return;
    }
    if (key.ctrl && key.name === "l") {
      store.appendSystem("screen cleared");
      return;
    }
    if (key.ctrl && key.name === "k") {
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
      store.clearPendingAttachments();
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
      const raw = new TextDecoder().decode(event.bytes);
      const resolved = resolveDroppedAttachment(raw);
      if (!resolved) return;
      event.preventDefault();
      event.stopPropagation();
      store.addPendingAttachment(resolved);
    };
    renderer.keyInput.on("paste", handler);
    return () => {
      renderer.keyInput.off("paste", handler);
    };
  }, [renderer, store]);

  const showSuggestions = matches.length > 0 && prefix.startsWith("/");
  const suggestionRows = showSuggestions
    ? Math.min(matches.length, 5) + (matches.length > 5 ? 1 : 0)
    : 0;
  const attachmentRows = snapshot.pendingAttachments.length > 0 ? 1 : 0;
  const inputContainerHeight = 2 + attachmentRows + suggestionRows + 1;

  const hovered = snapshot.hoveredPreview;

  return (
    <box style={{ flexDirection: "column", flexGrow: 1, height: "100%" }}>
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

      <box
        style={{
          height: 1,
          paddingLeft: 1,
          paddingRight: 1,
          backgroundColor: theme.colors.border,
        }}
      >
        <text>
          <span style={{ fg: theme.colors.user }}>{` ${theme.title} `}</span>
          <span style={{ fg: theme.colors.system }}>
            {" — Ctrl+C exit · Ctrl+L clear · Tab complete · Esc cancel · drop files to attach"}
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
        {snapshot.entries.map((entry) => (
          <MessageItem key={entry.id} entry={entry} store={store} />
        ))}
      </scrollbox>

      <box
        style={{
          height: 1,
          paddingLeft: 1,
          paddingRight: 1,
        }}
      >
        <text>
          {snapshot.typing ? (
            <span style={{ fg: theme.colors.typing }}>
              {"● agent is typing…"}
            </span>
          ) : (
            <span style={{ fg: theme.colors.system }}>{" "}</span>
          )}
        </text>
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
        {snapshot.pendingAttachments.length > 0 ? (
          <Attachments pending={snapshot.pendingAttachments} />
        ) : null}
        {showSuggestions ? (
          <Suggestions matches={matches} selectedIndex={tabIndex} />
        ) : null}
        <box style={{ height: 1, flexShrink: 0 }}>
          <input
            focused
            placeholder="type a message and press enter…"
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
  );
}

export function MountedApp({ store }: AppProps) {
  useEffect(() => {
    return () => {
      store.closeInput();
    };
  }, [store]);
  return <App store={store} />;
}
