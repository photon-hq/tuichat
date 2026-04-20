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
    resolve: async () => ({ id: "tuichat" }),
  },

  lifecycle: {
    createClient: async ({ config }): Promise<TuichatClient> => {
      return await createTuichatClient({ commands: config.commands });
    },

    destroyClient: async ({ client }) => {
      await destroyTuichatClient(client);
    },
  },

  events: {
    async *messages({ client }) {
      const { store } = client;
      while (true) {
        const result = await store.nextUserInput();
        if (result.done) return;
        yield {
          id: crypto.randomUUID(),
          content: result.value,
          sender: { id: "tuichat-user" },
          space: { id: "tuichat" },
          timestamp: new Date(),
        };
      }
    },
  },

  actions: {
    send: async ({ client, content }) => {
      client.store.appendAgent(content);
    },

    startTyping: async ({ client }) => {
      client.store.setTyping(true);
    },

    stopTyping: async ({ client }) => {
      client.store.setTyping(false);
    },

    reactToMessage: async ({ client, messageId, reaction }) => {
      client.store.react(messageId, reaction);
    },

    replyToMessage: async ({ client, messageId, content }) => {
      client.store.appendAgent(content, { replyTo: messageId });
    },
  },
});
