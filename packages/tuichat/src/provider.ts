import z from "zod";
import type { Content } from "spectrum-ts";
import { definePlatform } from "spectrum-ts";
import {
  createTuichatClient,
  destroyTuichatClient,
  type TuichatClient,
} from "./client";
import { spoolAttachment } from "./spool";

async function resolveAttachmentPath(
  content: Content
): Promise<string | undefined> {
  if (content.type !== "attachment") return undefined;
  try {
    const bytes = await content.read();
    const path = spoolAttachment(content.name, bytes);
    return path || undefined;
  } catch {
    return undefined;
  }
}

function plainFormat(content: Content): string {
  if (content.type === "text") return content.text;
  if (content.type === "attachment")
    return `[attachment: ${content.name}${content.size !== undefined ? ` (${content.size} bytes)` : ""}]`;
  if (content.type === "custom") return `[custom] ${JSON.stringify(content.raw)}`;
  return "";
}

const commandSchema = z.object({
  name: z.string().regex(/^\/[A-Za-z0-9_-]+$/, "command must start with /"),
  description: z.string().optional(),
});

const PLAIN_SPACE_ID = "tuichat";

export const tuichat = definePlatform("tuichat", {
  config: z.object({
    commands: z.array(commandSchema).optional(),
  }),

  user: {
    resolve: async () => ({ id: "tuichat-user" }),
  },

  space: {
    params: z.object({ id: z.string().optional() }),
    resolve: async (ctx) => {
      const client = ctx.client as TuichatClient;
      if (client.mode === "plain") {
        return { id: ctx.input.params?.id ?? PLAIN_SPACE_ID };
      }
      const id = ctx.input.params?.id ?? client.store.newChat();
      client.store.ensureChat(id);
      return { id };
    },
  },

  lifecycle: {
    createClient: async ({ config }): Promise<TuichatClient> => {
      return await createTuichatClient({ commands: config.commands });
    },

    destroyClient: async ({ client }) => {
      await destroyTuichatClient(client as TuichatClient);
    },
  },

  events: {
    async *messages(ctx) {
      const client = ctx.client as TuichatClient;
      if (client.mode === "plain") {
        for await (const line of client.lines) {
          yield {
            id: crypto.randomUUID(),
            content: { type: "text" as const, text: line },
            sender: { id: "tuichat-user" },
            space: { id: PLAIN_SPACE_ID },
            timestamp: new Date(),
          };
        }
        return;
      }
      const { store } = client;
      while (true) {
        const result = await store.nextUserInput();
        if (result.done) return;
        yield {
          id: crypto.randomUUID(),
          content: result.value.content,
          sender: { id: "tuichat-user" },
          space: { id: result.value.chatId },
          timestamp: new Date(),
        };
      }
    },
  },

  actions: {
    send: async (ctx) => {
      const client = ctx.client as TuichatClient;
      if (client.mode === "plain") {
        process.stdout.write(`${plainFormat(ctx.content)}\n`);
        return;
      }
      client.store.ensureChat(ctx.space.id);
      const attachmentPath = await resolveAttachmentPath(ctx.content);
      client.store.appendAgent(ctx.space.id, ctx.content, { attachmentPath });
    },

    startTyping: async (ctx) => {
      const client = ctx.client as TuichatClient;
      if (client.mode === "plain") return;
      client.store.setTyping(ctx.space.id, true);
    },

    stopTyping: async (ctx) => {
      const client = ctx.client as TuichatClient;
      if (client.mode === "plain") return;
      client.store.setTyping(ctx.space.id, false);
    },

    reactToMessage: async (ctx) => {
      const client = ctx.client as TuichatClient;
      if (client.mode === "plain") return;
      client.store.react(ctx.space.id, ctx.messageId, ctx.reaction);
    },

    replyToMessage: async (ctx) => {
      const client = ctx.client as TuichatClient;
      if (client.mode === "plain") {
        process.stdout.write(`${plainFormat(ctx.content)}\n`);
        return;
      }
      client.store.ensureChat(ctx.space.id);
      const attachmentPath = await resolveAttachmentPath(ctx.content);
      client.store.appendAgent(ctx.space.id, ctx.content, {
        replyTo: ctx.messageId,
        attachmentPath,
      });
    },
  },
});
