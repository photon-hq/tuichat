import type { Socket } from "node:net";
import { encodeMessage, MessageDecoder } from "./codec";
import {
  ERROR_CODES,
  type RpcMessage,
  type RpcNotification,
  type RpcRequest,
  type RpcResponse,
} from "./types";

export type RequestHandler = (
  method: string,
  params: unknown
) => Promise<unknown> | unknown;

export type NotificationHandler = (method: string, params: unknown) => void;

export class RpcSession {
  private decoder = new MessageDecoder();
  private nextId = 1;
  private pending = new Map<
    number | string,
    { resolve: (v: unknown) => void; reject: (err: unknown) => void }
  >();

  private onRequest: RequestHandler | null = null;
  private onNotification: NotificationHandler | null = null;
  private onClose: (() => void) | null = null;
  private closed = false;

  constructor(private socket: Socket) {
    socket.on("data", (chunk: Buffer) => this.handleChunk(chunk));
    socket.on("error", () => this.shutdown());
    socket.on("close", () => this.shutdown());
  }

  handleRequests(handler: RequestHandler): void {
    this.onRequest = handler;
  }

  handleNotifications(handler: NotificationHandler): void {
    this.onNotification = handler;
  }

  onClosed(handler: () => void): void {
    this.onClose = handler;
  }

  async request<TResult = unknown>(
    method: string,
    params?: unknown
  ): Promise<TResult> {
    if (this.closed) throw new Error("session closed");
    const id = this.nextId++;
    const msg: RpcRequest = { jsonrpc: "2.0", id, method, params };
    return new Promise<TResult>((resolve, reject) => {
      this.pending.set(id, {
        resolve: (v) => resolve(v as TResult),
        reject,
      });
      this.write(msg);
    });
  }

  notify(method: string, params?: unknown): void {
    if (this.closed) return;
    const msg: RpcNotification = { jsonrpc: "2.0", method, params };
    this.write(msg);
  }

  close(): void {
    this.shutdown();
  }

  private write(msg: RpcMessage): void {
    try {
      this.socket.write(encodeMessage(msg));
    } catch {
      this.shutdown();
    }
  }

  private handleChunk(chunk: Buffer): void {
    let messages: RpcMessage[];
    try {
      messages = this.decoder.push(chunk);
    } catch (err) {
      this.shutdown();
      throw err;
    }
    for (const msg of messages) {
      this.dispatch(msg);
    }
  }

  private dispatch(msg: RpcMessage): void {
    if ("id" in msg && "method" in msg) {
      this.handleIncomingRequest(msg as RpcRequest);
    } else if ("id" in msg) {
      this.handleIncomingResponse(msg as RpcResponse);
    } else if ("method" in msg) {
      const notif = msg as RpcNotification;
      try {
        this.onNotification?.(notif.method, notif.params);
      } catch {
        // swallow; notifications have no response
      }
    }
  }

  private async handleIncomingRequest(req: RpcRequest): Promise<void> {
    if (!this.onRequest) {
      this.respondError(req.id, ERROR_CODES.methodNotFound, `no handler`);
      return;
    }
    try {
      const result = await this.onRequest(req.method, req.params);
      this.write({
        jsonrpc: "2.0",
        id: req.id,
        result: result ?? null,
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      this.respondError(req.id, ERROR_CODES.serverError, message);
    }
  }

  private handleIncomingResponse(res: RpcResponse): void {
    const pending = this.pending.get(res.id);
    if (!pending) return;
    this.pending.delete(res.id);
    if (res.error) pending.reject(new Error(res.error.message));
    else pending.resolve(res.result);
  }

  private respondError(
    id: number | string,
    code: number,
    message: string
  ): void {
    this.write({ jsonrpc: "2.0", id, error: { code, message } });
  }

  private shutdown(): void {
    if (this.closed) return;
    this.closed = true;
    for (const [, pending] of this.pending) {
      pending.reject(new Error("session closed"));
    }
    this.pending.clear();
    try {
      this.socket.end();
    } catch {
      // best-effort
    }
    try {
      this.socket.destroy();
    } catch {
      // best-effort
    }
    this.onClose?.();
  }
}
