import { existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

export { startServer, type ServerHandle, type ServerOptions } from "./server";

/**
 * Returns the absolute path to the tuichat binary suitable for `spawn(["bun", path])`.
 * Works whether the package is consumed as TS source (Bun workspace) or compiled dist (npm).
 */
export function getBinaryPath(): string {
  const here = fileURLToPath(import.meta.url);
  const root = dirname(here);
  const tsPath = join(root, "bin", "tuichat.ts");
  if (existsSync(tsPath)) return tsPath;
  return join(root, "bin", "tuichat.js");
}
export type { Content } from "./content";
export type {
  ChatState,
  CommandDef,
  HoveredPreview,
  LogEntry,
  PendingAttachment,
  Role,
  Snapshot,
  Store,
} from "./store";
export {
  PROTOCOL_VERSION,
  type Content as ProtocolContent,
  type InitializeParams,
  type InitializeResult,
  type MessageNotification,
  type ReadyBanner,
  type SendParams,
  type SendResult,
} from "./protocol/types";
export { encodeMessage, MessageDecoder } from "./protocol/codec";
export { RpcSession } from "./protocol/session";
