import type { RpcMessage } from "./types";

const HEADER_TERMINATOR = Buffer.from("\r\n\r\n");
const CONTENT_LENGTH_PREFIX = "content-length:";

export function encodeMessage(message: RpcMessage): Buffer {
  const body = Buffer.from(JSON.stringify(message), "utf8");
  const header = Buffer.from(`Content-Length: ${body.byteLength}\r\n\r\n`);
  return Buffer.concat([header, body]);
}

export class MessageDecoder {
  private buffer: Buffer = Buffer.alloc(0);

  push(chunk: Buffer): RpcMessage[] {
    this.buffer = this.buffer.length === 0 ? chunk : Buffer.concat([this.buffer, chunk]);
    const out: RpcMessage[] = [];
    while (true) {
      const next = this.tryReadOne();
      if (!next) break;
      out.push(next);
    }
    return out;
  }

  private tryReadOne(): RpcMessage | null {
    const headerEnd = this.buffer.indexOf(HEADER_TERMINATOR);
    if (headerEnd < 0) return null;

    const headerText = this.buffer.subarray(0, headerEnd).toString("utf8");
    let contentLength = -1;
    for (const line of headerText.split("\r\n")) {
      const lower = line.toLowerCase();
      if (lower.startsWith(CONTENT_LENGTH_PREFIX)) {
        const value = line.slice(CONTENT_LENGTH_PREFIX.length).trim();
        contentLength = Number.parseInt(value, 10);
        if (Number.isNaN(contentLength) || contentLength < 0) {
          throw new Error(`invalid Content-Length header: ${value}`);
        }
      }
    }
    if (contentLength < 0) {
      throw new Error("missing Content-Length header");
    }

    const bodyStart = headerEnd + HEADER_TERMINATOR.length;
    const bodyEnd = bodyStart + contentLength;
    if (this.buffer.length < bodyEnd) return null;

    const body = this.buffer.subarray(bodyStart, bodyEnd).toString("utf8");
    this.buffer = this.buffer.subarray(bodyEnd);

    try {
      return JSON.parse(body) as RpcMessage;
    } catch (err) {
      throw new Error(
        `failed to parse JSON-RPC body: ${err instanceof Error ? err.message : String(err)}`
      );
    }
  }
}
