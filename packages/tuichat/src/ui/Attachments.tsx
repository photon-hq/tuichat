import type { PendingAttachment } from "../store";
import { formatBytes } from "./theme";
import { theme } from "./theme";

interface AttachmentsProps {
  pending: readonly PendingAttachment[];
}

export function Attachments({ pending }: AttachmentsProps) {
  if (pending.length === 0) return null;

  return (
    <box
      style={{
        height: 1,
        flexShrink: 0,
        flexDirection: "row",
        backgroundColor: theme.colors.suggestionBg,
        paddingLeft: 1,
        paddingRight: 1,
      }}
    >
      <text>
        {pending.map((a, i) => {
          const size = a.size !== undefined ? ` ${formatBytes(a.size)}` : "";
          const label = `📎 ${a.name}${size}`;
          const sep = i > 0 ? "  " : "";
          return (
            <span
              key={a.path}
              style={{
                fg: theme.colors.attachment,
                bg: theme.colors.suggestionBg,
              }}
            >
              {sep + label}
            </span>
          );
        })}
      </text>
    </box>
  );
}
