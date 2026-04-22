import { readFile } from "node:fs/promises";
import type { Content as InternalContent } from "../content";
import { spoolAttachment } from "../spool";
import type { Content as ProtocolContent } from "./types";

export function protocolToInternal(p: ProtocolContent): InternalContent {
  if (p.type === "text" || p.type === "custom" || p.type === "contact") {
    return p;
  }
  if (p.type === "attachment" || p.type === "voice") {
    const hasPath = typeof p.path === "string" && p.path.length > 0;
    const hasBytes = typeof p.bytes === "string" && p.bytes.length > 0;
    let path = hasPath ? p.path : undefined;
    let resolvedBytes: Buffer | undefined;

    if (!hasPath && hasBytes) {
      resolvedBytes = Buffer.from(p.bytes!, "base64");
      const name = (p.type === "attachment" ? p.name : p.name) ?? "attachment";
      const spooled = spoolAttachment(name, resolvedBytes);
      if (spooled) path = spooled;
    }

    const read = async (): Promise<Buffer> => {
      if (resolvedBytes) return resolvedBytes;
      if (path) {
        const buf = await readFile(path);
        resolvedBytes = buf;
        return buf;
      }
      throw new Error("attachment has neither path nor bytes");
    };

    if (p.type === "attachment") {
      return {
        type: "attachment",
        name: p.name,
        mimeType: p.mimeType,
        size: p.size,
        read,
        path,
      };
    }
    return {
      type: "voice",
      name: p.name,
      mimeType: p.mimeType,
      size: p.size,
      read,
      path,
    };
  }
  // Exhaustive — TS sees all cases above.
  throw new Error("unhandled content type");
}

export async function internalToProtocol(
  i: InternalContent
): Promise<ProtocolContent> {
  if (i.type === "text" || i.type === "custom" || i.type === "contact") {
    return i;
  }
  if (i.type === "attachment" || i.type === "voice") {
    const base = {
      name: i.name,
      mimeType: i.mimeType,
      size: i.size,
    };
    if (i.path) {
      return i.type === "attachment"
        ? { type: "attachment", ...base, name: i.name, path: i.path }
        : { type: "voice", ...base, name: i.name, path: i.path };
    }
    const bytes = (await i.read()).toString("base64");
    return i.type === "attachment"
      ? { type: "attachment", ...base, name: i.name, bytes }
      : { type: "voice", ...base, name: i.name, bytes };
  }
  throw new Error("unhandled content type");
}
