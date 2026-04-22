import { connect } from "node:net";
import {
  createTuichatClient,
  destroyTuichatClient,
  type TuichatClient,
} from "./client";
import type { Content as InternalContent } from "./content";
import { internalToProtocol, protocolToInternal } from "./protocol/convert";
import { RpcSession } from "./protocol/session";
import {
  ERROR_CODES,
  type EnsureSpaceParams,
  type InitializeParams,
  type InitializeResult,
  PROTOCOL_VERSION,
  type ReactParams,
  type ReplyParams,
  type SendParams,
  type SendResult,
  type TypingParams,
} from "./protocol/types";

const SERVER_VERSION = "0.1.0";

function iso(): string {
  return new Date().toISOString();
}

function newId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `id-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

interface ActiveSession {
  session: RpcSession;
  client: TuichatClient | null;
}

async function pumpUserInput(active: ActiveSession): Promise<void> {
  const { client, session } = active;
  if (!client) return;
  while (true) {
    const next = await client.store.nextUserInput();
    if (next.done) return;
    try {
      const protocolContent = await internalToProtocol(next.value.content);
      session.notify("message", {
        id: newId(),
        spaceId: next.value.chatId,
        senderId: "terminal-tui-user",
        content: protocolContent,
        timestamp: iso(),
      });
    } catch {
      // best-effort; drop this message on error
    }
  }
}

function installRpcHandlers(active: ActiveSession): void {
  const { session } = active;
  let initialized = false;

  session.handleRequests(async (method, rawParams) => {
    if (method === "initialize") {
      if (initialized) throw new Error("already initialized");
      const params = (rawParams ?? {}) as InitializeParams;
      const client = await createTuichatClient({ commands: params.commands });
      active.client = client;
      initialized = true;
      void pumpUserInput(active);
      const result: InitializeResult = {
        protocolVersion: PROTOCOL_VERSION,
        serverInfo: { name: "tuichat", version: SERVER_VERSION },
      };
      return result;
    }

    if (!initialized || !active.client) {
      throw new Error("not initialized");
    }
    const client = active.client;

    if (method === "shutdown") {
      await destroyTuichatClient(client);
      setImmediate(() => process.exit(0));
      return null;
    }

    const store = client.store;

    switch (method) {
      case "send": {
        const params = rawParams as SendParams;
        const internal: InternalContent = protocolToInternal(params.content);
        store.ensureChat(params.spaceId);
        store.appendAgent(params.spaceId, internal, {
          attachmentPath:
            internal.type === "attachment" || internal.type === "voice"
              ? internal.path
              : undefined,
        });
        const result: SendResult = { id: newId(), timestamp: iso() };
        return result;
      }
      case "replyToMessage": {
        const params = rawParams as ReplyParams;
        const internal: InternalContent = protocolToInternal(params.content);
        store.ensureChat(params.spaceId);
        store.appendAgent(params.spaceId, internal, {
          replyTo: params.messageId,
          attachmentPath:
            internal.type === "attachment" || internal.type === "voice"
              ? internal.path
              : undefined,
        });
        const result: SendResult = { id: newId(), timestamp: iso() };
        return result;
      }
      case "startTyping": {
        const params = rawParams as TypingParams;
        store.setTyping(params.spaceId, true);
        return null;
      }
      case "stopTyping": {
        const params = rawParams as TypingParams;
        store.setTyping(params.spaceId, false);
        return null;
      }
      case "reactToMessage": {
        const params = rawParams as ReactParams;
        store.react(params.spaceId, params.messageId, params.reaction);
        return null;
      }
      case "ensureSpace": {
        const params = rawParams as EnsureSpaceParams;
        store.ensureChat(params.id);
        return null;
      }
      default: {
        const err: Error & { code?: number } = new Error(
          `unknown method: ${method}`
        );
        err.code = ERROR_CODES.methodNotFound;
        throw err;
      }
    }
  });
}

export interface ServerOptions {
  host: string;
  port: number;
}

export interface ServerHandle {
  close: () => Promise<void>;
}

export async function startServer(
  options: ServerOptions
): Promise<ServerHandle> {
  const socket = await new Promise<import("node:net").Socket>(
    (resolve, reject) => {
      const s = connect({ host: options.host, port: options.port }, () => {
        s.off("error", reject);
        resolve(s);
      });
      s.once("error", reject);
    }
  );

  const session = new RpcSession(socket);
  const active: ActiveSession = { session, client: null };
  installRpcHandlers(active);

  session.onClosed(() => {
    if (active.client) {
      void destroyTuichatClient(active.client);
    }
    setImmediate(() => process.exit(0));
  });

  return {
    close: async () => {
      session.close();
    },
  };
}
