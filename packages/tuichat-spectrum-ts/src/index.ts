import type { ChildProcess } from "node:child_process";
import { spawn } from "node:child_process";
import { connect, type Socket } from "node:net";
import { createInterface, type Interface as Readline } from "node:readline";
import type { Readable } from "node:stream";
import z from "zod";
import {
  type AnyPlatformDef,
  type Content as SpectrumContent,
  definePlatform,
  type Platform,
} from "spectrum-ts";
import {
  getBinaryPath,
  MessageDecoder,
  type ProtocolContent,
  type ReadyBanner,
  RpcSession,
} from "tuichat";

interface ProtocolMessageNotification {
  id: string;
  spaceId: string;
  senderId: string;
  content: ProtocolContent;
  timestamp: string;
}

const commandSchema = z.object({
  name: z.string().regex(/^\/[A-Za-z0-9_-]+$/, "command must start with /"),
  description: z.string().optional(),
});

const PLAIN_SPACE_ID = "terminal-tui";

type AdapterClient = RichAdapterClient | PlainAdapterClient;

interface RichAdapterClient {
  readonly mode: "rich";
  readonly proc: ChildProcess & { stdout: Readable };
  readonly socket: Socket;
  readonly session: RpcSession;
  readonly messages: AsyncIterable<ProtocolMessageNotification>;
}

interface PlainAdapterClient {
  readonly mode: "plain";
  readonly readline: Readline;
  readonly messages: AsyncIterable<ProtocolMessageNotification>;
  close: () => void;
}

function isInteractive(): boolean {
  if (process.env.TUICHAT_FORCE_PLAIN === "1") return false;
  if (process.env.TUICHAT_FORCE_TUI === "1") return true;
  return Boolean(process.stdout.isTTY && process.stdin.isTTY);
}

async function spawnAndConnect(options: {
  commands?: { name: string; description?: string }[];
}): Promise<RichAdapterClient> {
  const binary = getBinaryPath();
  const proc = spawn("bun", [binary], {
    stdio: ["inherit", "pipe", "inherit"],
  }) as ChildProcess & { stdout: Readable };

  const banner = await readReadyBanner(proc);
  const socket = await dial("127.0.0.1", banner.port);
  const session = new RpcSession(socket);

  const pushQueue: ProtocolMessageNotification[] = [];
  const waiters: Array<
    (v: IteratorResult<ProtocolMessageNotification>) => void
  > = [];
  let closed = false;

  const asyncIter: AsyncIterable<ProtocolMessageNotification> = {
    [Symbol.asyncIterator]() {
      return {
        next(): Promise<IteratorResult<ProtocolMessageNotification>> {
          if (closed && pushQueue.length === 0) {
            return Promise.resolve({ value: undefined, done: true });
          }
          const buffered = pushQueue.shift();
          if (buffered) {
            return Promise.resolve({ value: buffered, done: false });
          }
          return new Promise((resolve) => {
            waiters.push(resolve);
          });
        },
      };
    },
  };

  session.handleNotifications((method, params) => {
    if (method !== "message") return;
    const msg = params as ProtocolMessageNotification;
    const waiter = waiters.shift();
    if (waiter) waiter({ value: msg, done: false });
    else pushQueue.push(msg);
  });

  session.onClosed(() => {
    closed = true;
    while (waiters.length > 0) {
      waiters.shift()?.({ value: undefined, done: true });
    }
  });

  await session.request("initialize", {
    commands: options.commands,
    clientInfo: { name: "tuichat-spectrum-ts", version: "0.1.0" },
  });

  return { mode: "rich", proc, socket, session, messages: asyncIter };
}

async function readReadyBanner(
  proc: ChildProcess & { stdout: Readable }
): Promise<ReadyBanner> {
  return new Promise<ReadyBanner>((resolve, reject) => {
    let buf = "";
    const onData = (chunk: Buffer) => {
      buf += chunk.toString("utf8");
      const nl = buf.indexOf("\n");
      if (nl < 0) return;
      const line = buf.slice(0, nl);
      try {
        const banner = JSON.parse(line) as ReadyBanner;
        if (!banner.ready || typeof banner.port !== "number") {
          throw new Error(`bad banner: ${line}`);
        }
        proc.stdout.off("data", onData);
        resolve(banner);
      } catch (err) {
        reject(err);
      }
    };
    proc.stdout.on("data", onData);
    proc.once("exit", (code) => {
      reject(new Error(`tuichat exited before ready (code=${code})`));
    });
  });
}

async function dial(host: string, port: number): Promise<Socket> {
  return new Promise<Socket>((resolve, reject) => {
    const sock = connect({ host, port }, () => {
      sock.off("error", reject);
      resolve(sock);
    });
    sock.once("error", reject);
  });
}

function createPlainClient(): PlainAdapterClient {
  if (process.env.TUICHAT_QUIET !== "1") {
    process.stderr.write(
      "[tuichat] non-TTY detected — running in plain readline mode. " +
        "Rich TUI features (sidebar, image preview, drag-drop) are disabled. " +
        "Set TUICHAT_FORCE_TUI=1 to force TUI, TUICHAT_QUIET=1 to silence this notice.\n"
    );
  }
  const rl = createInterface({
    input: process.stdin,
    output: process.stdout,
  });
  rl.on("SIGINT", () => {
    rl.close();
    process.kill(process.pid, "SIGINT");
  });

  const buffer: ProtocolMessageNotification[] = [];
  const waiters: Array<
    (v: IteratorResult<ProtocolMessageNotification>) => void
  > = [];
  let closed = false;

  (async () => {
    for await (const line of rl) {
      if (closed) break;
      const msg: ProtocolMessageNotification = {
        id: crypto.randomUUID(),
        spaceId: PLAIN_SPACE_ID,
        senderId: "terminal-tui-user",
        content: { type: "text", text: line },
        timestamp: new Date().toISOString(),
      };
      const waiter = waiters.shift();
      if (waiter) waiter({ value: msg, done: false });
      else buffer.push(msg);
    }
    closed = true;
    while (waiters.length > 0) {
      waiters.shift()?.({ value: undefined, done: true });
    }
  })();

  const iter: AsyncIterable<ProtocolMessageNotification> = {
    [Symbol.asyncIterator]() {
      return {
        next(): Promise<IteratorResult<ProtocolMessageNotification>> {
          if (closed && buffer.length === 0) {
            return Promise.resolve({ value: undefined, done: true });
          }
          const buffered = buffer.shift();
          if (buffered) {
            return Promise.resolve({ value: buffered, done: false });
          }
          return new Promise((resolve) => {
            waiters.push(resolve);
          });
        },
      };
    },
  };

  return {
    mode: "plain",
    readline: rl,
    messages: iter,
    close: () => {
      if (closed) return;
      closed = true;
      rl.close();
      while (waiters.length > 0) {
        waiters.shift()?.({ value: undefined, done: true });
      }
    },
  };
}

async function spectrumToProtocol(
  content: SpectrumContent
): Promise<ProtocolContent> {
  if (content.type === "text" || content.type === "custom") return content;
  if (content.type === "attachment") {
    const buf = await content.read();
    return {
      type: "attachment",
      name: content.name,
      mimeType: content.mimeType,
      size: content.size,
      bytes: buf.toString("base64"),
    };
  }
  throw new Error(
    `unsupported content type for terminal-tui: ${(content as { type: string }).type}`
  );
}

function protocolToSpectrum(p: ProtocolContent): SpectrumContent {
  if (p.type === "text" || p.type === "custom") return p as SpectrumContent;
  if (p.type === "attachment") {
    const bytes = p.bytes ? Buffer.from(p.bytes, "base64") : undefined;
    const path = p.path;
    return {
      type: "attachment",
      name: p.name,
      mimeType: p.mimeType,
      size: p.size,
      read: async () => {
        if (bytes) return bytes;
        if (path) {
          const { readFile } = await import("node:fs/promises");
          return await readFile(path);
        }
        throw new Error("attachment has neither path nor bytes");
      },
      stream: async () => {
        if (bytes) {
          return new ReadableStream({
            start(ctrl) {
              ctrl.enqueue(new Uint8Array(bytes));
              ctrl.close();
            },
          });
        }
        if (path) {
          const { createReadStream } = await import("node:fs");
          const { Readable } = await import("node:stream");
          return Readable.toWeb(
            createReadStream(path)
          ) as ReadableStream<Uint8Array>;
        }
        throw new Error("attachment has neither path nor bytes");
      },
    } as SpectrumContent;
  }
  return { type: "custom", raw: p } as SpectrumContent;
}

function plainFormat(content: ProtocolContent): string {
  if (content.type === "text") return content.text;
  if (content.type === "attachment") {
    const size = content.size !== undefined ? ` (${content.size} bytes)` : "";
    return `[attachment: ${content.name}${size}]`;
  }
  if (content.type === "voice") {
    const size = content.size !== undefined ? ` (${content.size} bytes)` : "";
    return `[voice: ${content.name ?? "audio"}${size}]`;
  }
  if (content.type === "contact") {
    const nm = content.name?.formatted ?? content.name?.first ?? "contact";
    return `[contact: ${nm}]`;
  }
  if (content.type === "custom") return `[custom] ${JSON.stringify(content.raw)}`;
  return "";
}

let nextChatIndex = 1;
const knownChats = new Set<string>();

function generateChatId(): string {
  while (knownChats.has(`chat-${nextChatIndex}`)) nextChatIndex += 1;
  const id = `chat-${nextChatIndex}`;
  nextChatIndex += 1;
  knownChats.add(id);
  return id;
}

export const terminalTui = definePlatform("terminal-tui", {
  config: z.object({
    commands: z.array(commandSchema).optional(),
  }),

  user: {
    resolve: async () => ({ id: "terminal-tui-user" }),
  },

  space: {
    params: z.object({ id: z.string().optional() }),
    resolve: async (ctx) => {
      const client = ctx.client as AdapterClient;
      if (client.mode === "plain") {
        return { id: ctx.input.params?.id ?? PLAIN_SPACE_ID };
      }
      const id = ctx.input.params?.id ?? generateChatId();
      knownChats.add(id);
      await client.session.request("ensureSpace", { id });
      return { id };
    },
  },

  lifecycle: {
    createClient: async ({ config }): Promise<AdapterClient> => {
      if (!isInteractive()) return createPlainClient();
      return await spawnAndConnect({ commands: config.commands });
    },

    destroyClient: async ({ client }) => {
      const c = client as AdapterClient;
      if (c.mode === "plain") {
        c.close();
        return;
      }
      try {
        await c.session.request("shutdown");
      } catch {
        // best-effort
      }
      c.session.close();
      try {
        c.proc.kill("SIGTERM");
      } catch {
        // best-effort
      }
    },
  },

  events: {
    async *messages(ctx) {
      const client = ctx.client as AdapterClient;
      for await (const msg of client.messages) {
        knownChats.add(msg.spaceId);
        yield {
          id: msg.id,
          content: protocolToSpectrum(msg.content),
          sender: { id: msg.senderId },
          space: { id: msg.spaceId },
          timestamp: new Date(msg.timestamp),
        };
      }
    },
  },

  actions: {
    send: async (ctx) => {
      const client = ctx.client as AdapterClient;
      const content = await spectrumToProtocol(ctx.content);
      if (client.mode === "plain") {
        process.stdout.write(`${plainFormat(content)}\n`);
        return;
      }
      await client.session.request("send", {
        spaceId: ctx.space.id,
        content,
      });
    },

    startTyping: async (ctx) => {
      const client = ctx.client as AdapterClient;
      if (client.mode === "plain") return;
      await client.session.request("startTyping", {
        spaceId: ctx.space.id,
      });
    },

    stopTyping: async (ctx) => {
      const client = ctx.client as AdapterClient;
      if (client.mode === "plain") return;
      await client.session.request("stopTyping", {
        spaceId: ctx.space.id,
      });
    },

    reactToMessage: async (ctx) => {
      const client = ctx.client as AdapterClient;
      if (client.mode === "plain") return;
      await client.session.request("reactToMessage", {
        spaceId: ctx.space.id,
        messageId: ctx.messageId,
        reaction: ctx.reaction,
      });
    },

    replyToMessage: async (ctx) => {
      const client = ctx.client as AdapterClient;
      const content = await spectrumToProtocol(ctx.content);
      if (client.mode === "plain") {
        process.stdout.write(`${plainFormat(content)}\n`);
        return;
      }
      await client.session.request("replyToMessage", {
        spaceId: ctx.space.id,
        messageId: ctx.messageId,
        content,
      });
    },
  },
});

export const terminalTuiProvider: Platform<AnyPlatformDef> =
  terminalTui as unknown as Platform<AnyPlatformDef>;

export type { ReadyBanner } from "tuichat";
export { MessageDecoder };
