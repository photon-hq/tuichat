import type { ChatState, Store } from "../store";
import { theme } from "./theme";

interface SidebarProps {
  chats: readonly ChatState[];
  activeChatId: string | null;
  store: Store;
}

const SIDEBAR_WIDTH = 20;

function trim(s: string, n: number): string {
  return s.length <= n ? s : `${s.slice(0, n - 1)}…`;
}

export function Sidebar({ chats, activeChatId, store }: SidebarProps) {
  return (
    <box
      style={{
        width: SIDEBAR_WIDTH,
        flexDirection: "column",
        flexShrink: 0,
        border: ["right"],
        borderColor: theme.colors.border,
      }}
    >
      <box
        style={{
          height: 1,
          paddingLeft: 1,
          paddingRight: 1,
        }}
      >
        <text>
          <span style={{ fg: theme.colors.system }}>{"Chats"}</span>
        </text>
      </box>

      <box style={{ flexDirection: "column", flexShrink: 0 }}>
        {chats.map((chat) => {
          const selected = chat.id === activeChatId;
          return (
            <box
              key={chat.id}
              style={{
                height: 1,
                paddingLeft: 1,
                paddingRight: 1,
              }}
              onMouseDown={() => store.setActiveChat(chat.id)}
            >
              <text>
                <span
                  style={{
                    fg: selected ? theme.colors.prompt : theme.colors.system,
                  }}
                >
                  {selected ? "› " : "  "}
                </span>
                <span
                  style={{
                    fg: selected ? theme.colors.user : theme.colors.input,
                  }}
                >
                  {trim(chat.id, SIDEBAR_WIDTH - 4)}
                </span>
              </text>
            </box>
          );
        })}
      </box>

      <box style={{ flexGrow: 1 }} />

      <box
        style={{
          height: 1,
          paddingLeft: 1,
          paddingRight: 1,
        }}
      >
        <text>
          <span style={{ fg: theme.colors.system }}>{"Ctrl+N new"}</span>
        </text>
      </box>
      <box
        style={{
          height: 1,
          paddingLeft: 1,
          paddingRight: 1,
        }}
      >
        <text>
          <span style={{ fg: theme.colors.system }}>{"Ctrl+J/K ↕"}</span>
        </text>
      </box>
    </box>
  );
}
