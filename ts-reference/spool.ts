import { createHash } from "node:crypto";
import { mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { basename, extname, join } from "node:path";

const SPOOL_DIR = join(tmpdir(), "tuichat");
const spoolCache = new Map<string, string>();
let ensured = false;

function ensureSpoolDir(): void {
  if (ensured) return;
  try {
    mkdirSync(SPOOL_DIR, { recursive: true });
  } catch {
    // best-effort
  }
  ensured = true;
}

function safeExt(name: string): string {
  const ext = extname(name).toLowerCase();
  if (!ext || !/^\.[A-Za-z0-9]+$/.test(ext)) return "";
  return ext;
}

export function spoolAttachment(
  name: string,
  bytes: Buffer | Uint8Array
): string {
  ensureSpoolDir();
  const hash = createHash("sha1")
    .update(name)
    .update(Buffer.from(bytes))
    .digest("hex")
    .slice(0, 16);
  const key = `${hash}-${basename(name)}`;
  const cached = spoolCache.get(key);
  if (cached) return cached;
  const filename = `${hash}${safeExt(name)}`;
  const path = join(SPOOL_DIR, filename);
  try {
    writeFileSync(path, Buffer.from(bytes));
    spoolCache.set(key, path);
    return path;
  } catch {
    return "";
  }
}
