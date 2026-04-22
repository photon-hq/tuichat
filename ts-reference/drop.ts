import { existsSync, statSync } from "node:fs";
import { basename } from "node:path";
import type { PendingAttachment } from "./store";

export function parseDroppedPath(raw: string): string | null {
  let s = raw.trim();
  if (s.length === 0) return null;

  if (s.startsWith("file://")) {
    try {
      s = decodeURIComponent(s.slice(7));
    } catch {
      return null;
    }
  }

  if (
    (s.startsWith("'") && s.endsWith("'")) ||
    (s.startsWith('"') && s.endsWith('"'))
  ) {
    s = s.slice(1, -1);
  }

  s = s.replace(/\\(.)/g, "$1");

  if (/[\r\n]/.test(s)) return null;

  return s;
}

export function resolveDroppedAttachment(raw: string): PendingAttachment | null {
  const path = parseDroppedPath(raw);
  if (!path) return null;
  try {
    if (!existsSync(path)) return null;
    const st = statSync(path);
    if (!st.isFile()) return null;
    return {
      path,
      name: basename(path),
      size: st.size,
    };
  } catch {
    return null;
  }
}

export interface ExtractedDrops {
  cleaned: string;
  attachments: PendingAttachment[];
}

export function extractDroppedPaths(value: string): ExtractedDrops {
  const attachments: PendingAttachment[] = [];
  const quoted = /(['"])((?:(?!\1).)+)\1/g;
  let cleaned = value;
  let match: RegExpExecArray | null;
  const toRemove: string[] = [];
  while ((match = quoted.exec(value)) !== null) {
    const full = match[0];
    const inner = match[2]!;
    const resolved = resolveDroppedAttachment(inner);
    if (resolved) {
      attachments.push(resolved);
      toRemove.push(full);
    }
  }
  for (const r of toRemove) {
    cleaned = cleaned.replace(r, "");
  }
  if (attachments.length > 0) {
    cleaned = cleaned.replace(/\s+/g, " ").trim();
  }
  return { cleaned, attachments };
}
