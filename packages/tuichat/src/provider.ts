import z from "zod";
import { definePlatform } from "spectrum-ts";
import {
  createTuichatClient,
  destroyTuichatClient,
  type TuichatClient,
} from "./client";

const commandSchema = z.object({
  name: z.string().regex(/^\/[A-Za-z0-9_-]+$/, "command must start with /"),
  description: z.string().optional(),
});

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
      client.store.ensureChat(ctx.space.id);
      client.store.appendAgent(ctx.space.id, ctx.content);
    },

    startTyping: async (ctx) => {
      const client = ctx.client as TuichatClient;
      client.store.setTyping(ctx.space.id, true);
    },

    stopTyping: async (ctx) => {
      const client = ctx.client as TuichatClient;
      client.store.setTyping(ctx.space.id, false);
    },

    reactToMessage: async (ctx) => {
      const client = ctx.client as TuichatClient;
      client.store.react(ctx.space.id, ctx.messageId, ctx.reaction);
    },

    replyToMessage: async (ctx) => {
      const client = ctx.client as TuichatClient;
      client.store.ensureChat(ctx.space.id);
      client.store.appendAgent(ctx.space.id, ctx.content, {
        replyTo: ctx.messageId,
      });
    },
  },
});
