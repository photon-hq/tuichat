import { useEffect, useState } from "react";
import {
  ensureImageTransmitted,
  isSupportedImageMime,
  supportsKittyGraphics,
} from "../kitty";
import type { LogEntry, Store } from "../store";
import { linkify } from "./linkify";
import { formatBytes, formatTime, roleColor, theme } from "./theme";

interface MessageItemProps {
  entry: LogEntry;
  store: Store;
}

export function MessageItem({ entry, store }: MessageItemProps) {
  const { content, role, timestamp, reactions, replyTo, attachmentPath } =
    entry;
  const prefix = theme.prefix[role].padEnd(5, " ");
  const color = roleColor(role);
  const replyToEntry = replyTo
    ? store.getSnapshot().entries.find((e) => e.id === replyTo)
    : undefined;

  return (
    <box style={{ flexDirection: "column", marginBottom: 0 }}>
      {replyToEntry ? (
        <text>
          <span style={{ fg: theme.colors.border }}>{"  ┌─ "}</span>
          <span style={{ fg: roleColor(replyToEntry.role) }}>
            {theme.prefix[replyToEntry.role]}
          </span>
          <span style={{ fg: theme.colors.border }}>{": "}</span>
          <span style={{ fg: theme.colors.system }}>
            {quoteOf(replyToEntry)}
          </span>
        </text>
      ) : null}

      <ContentLine
        color={color}
        prefix={prefix}
        timestamp={timestamp}
        content={content}
        attachmentPath={attachmentPath}
        store={store}
      />

      {reactions.length > 0 ? (
        <text>
          <span style={{ fg: theme.colors.border }}>{"       "}</span>
          <span style={{ fg: theme.colors.system }}>{reactions.join(" ")}</span>
        </text>
      ) : null}
    </box>
  );
}

function quoteOf(entry: LogEntry): string {
  const c = entry.content;
  if (c.type === "text") return trimTo(c.text, 60);
  if (c.type === "attachment") return `[attachment: ${c.name}]`;
  return "[custom]";
}

const trimTo = (s: string, n: number) =>
  s.length <= n ? s : `${s.slice(0, n - 1)}…`;

interface ContentLineProps {
  color: string;
  prefix: string;
  timestamp: Date;
  content: LogEntry["content"];
  attachmentPath?: string;
  store: Store;
}

function ContentLine({
  color,
  prefix,
  timestamp,
  content,
  attachmentPath,
  store,
}: ContentLineProps) {
  const timeLabel = `[${formatTime(timestamp)}] `;

  switch (content.type) {
    case "text":
      return (
        <text>
          <span style={{ fg: theme.colors.timestamp }}>{timeLabel}</span>
          <span style={{ fg: color }}>{prefix}</span>
          <span style={{ fg: theme.colors.border }}>{" › "}</span>
          {linkify(content.text, {
            text: theme.colors.input,
            link: theme.colors.prompt,
          })}
        </text>
      );
    case "attachment":
      return (
        <AttachmentLine
          color={color}
          prefix={prefix}
          timeLabel={timeLabel}
          content={content}
          attachmentPath={attachmentPath}
          store={store}
        />
      );
    case "custom":
      return (
        <box style={{ flexDirection: "column" }}>
          <text>
            <span style={{ fg: theme.colors.timestamp }}>{timeLabel}</span>
            <span style={{ fg: color }}>{prefix}</span>
            <span style={{ fg: theme.colors.border }}>{" › "}</span>
            <span style={{ fg: theme.colors.custom }}>{"[custom]"}</span>
          </text>
          <text>
            <span style={{ fg: theme.colors.border }}>{"       "}</span>
            <span style={{ fg: theme.colors.system }}>
              {safeStringify(content.raw)}
            </span>
          </text>
        </box>
      );
    default:
      return null;
  }
}

interface AttachmentLineProps {
  color: string;
  prefix: string;
  timeLabel: string;
  content: Extract<LogEntry["content"], { type: "attachment" }>;
  attachmentPath?: string;
  store: Store;
}

function AttachmentLine({
  color,
  prefix,
  timeLabel,
  content,
  attachmentPath,
  store,
}: AttachmentLineProps) {
  const [resolvedSize, setResolvedSize] = useState<number | undefined>(
    content.size
  );

  useEffect(() => {
    if (resolvedSize !== undefined) return;
    let cancelled = false;
    content
      .read()
      .then((buf) => {
        if (cancelled) return;
        setResolvedSize(buf.byteLength);
      })
      .catch(() => {
        if (cancelled) return;
        store.appendSystem(`failed to read attachment "${content.name}"`);
      });
    return () => {
      cancelled = true;
    };
  }, [content, resolvedSize, store]);

  const sizeLabel =
    resolvedSize !== undefined ? ` ${formatBytes(resolvedSize)}` : "";

  const canPreview =
    supportsKittyGraphics() && isSupportedImageMime(content.mimeType);

  useEffect(() => {
    if (!canPreview) return;
    ensureImageTransmitted(content.name, content.read, 40, 10).catch(() => {
      // best-effort
    });
  }, [canPreview, content.name, content.read]);

  const onOver = canPreview
    ? () =>
        store.setHoveredPreview({
          cacheKey: content.name,
          name: content.name,
          read: content.read,
        })
    : undefined;
  const onOut = canPreview ? () => store.setHoveredPreview(null) : undefined;

  return (
    <box
      style={{ flexDirection: "column" }}
      onMouseOver={onOver}
      onMouseOut={onOut}
    >
      <text>
        <span style={{ fg: theme.colors.timestamp }}>{timeLabel}</span>
        <span style={{ fg: color }}>{prefix}</span>
        <span style={{ fg: theme.colors.border }}>{" › "}</span>
        <span style={{ fg: theme.colors.attachment }}>
          {`[${canPreview ? "image" : "attachment"}: `}
        </span>
        {attachmentPath ? (
          <a
            href={`file://${attachmentPath}`}
            style={{ fg: theme.colors.attachment, attributes: 4 }}
          >
            {content.name}
          </a>
        ) : (
          <span style={{ fg: theme.colors.attachment }}>{content.name}</span>
        )}
        <span style={{ fg: theme.colors.attachment }}>
          {`${sizeLabel}]`}
        </span>
        {canPreview ? (
          <span style={{ fg: theme.colors.system }}>
            {"  (hover to preview)"}
          </span>
        ) : null}
      </text>
    </box>
  );
}

function safeStringify(raw: unknown): string {
  try {
    return JSON.stringify(raw, null, 2);
  } catch {
    return String(raw);
  }
}
