import { createCliRenderer, type CliRenderer } from "@opentui/core";
import { createRoot } from "@opentui/react";
import { createInterface, type Interface as Readline } from "node:readline";
import { createElement } from "react";
import { MountedApp } from "./ui/App";
import { type CommandDef, createStore, type Store } from "./store";

export type TuichatMode = "tui" | "plain";

export interface TuichatTuiClient {
  mode: "tui";
  store: Store;
  renderer: CliRenderer;
  root: { unmount(): void };
}

export interface TuichatPlainClient {
  mode: "plain";
  readline: Readline;
  pushLine: (line: string) => void;
  lines: AsyncIterable<string>;
  close: () => void;
}

export type TuichatClient = TuichatTuiClient | TuichatPlainClient;

export interface CreateTuichatClientOptions {
  commands?: readonly CommandDef[];
}

let activeInstance: TuichatClient | null = null;

function isInteractive(): boolean {
  if (process.env.TUICHAT_FORCE_PLAIN === "1") return false;
  if (process.env.TUICHAT_FORCE_TUI === "1") return true;
  return Boolean(process.stdout.isTTY && process.stdin.isTTY);
}

async function createPlainClient(): Promise<TuichatPlainClient> {
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
  let closed = false;
  rl.on("SIGINT", () => {
    rl.close();
    process.kill(process.pid, "SIGINT");
  });

  const lines: string[] = [];
  const waiters: Array<{
    resolve: (v: IteratorResult<string>) => void;
  }> = [];

  const pump = async () => {
    for await (const line of rl) {
      if (closed) break;
      const waiter = waiters.shift();
      if (waiter) waiter.resolve({ value: line, done: false });
      else lines.push(line);
    }
    closed = true;
    while (waiters.length > 0) {
      waiters.shift()?.resolve({ value: undefined, done: true });
    }
  };
  void pump();

  const iter: AsyncIterable<string> = {
    [Symbol.asyncIterator]() {
      return {
        next(): Promise<IteratorResult<string>> {
          if (closed && lines.length === 0) {
            return Promise.resolve({ value: undefined, done: true });
          }
          const buffered = lines.shift();
          if (buffered !== undefined) {
            return Promise.resolve({ value: buffered, done: false });
          }
          return new Promise((resolve) => {
            waiters.push({ resolve });
          });
        },
      };
    },
  };

  return {
    mode: "plain",
    readline: rl,
    pushLine: (line) => {
      const waiter = waiters.shift();
      if (waiter) waiter.resolve({ value: line, done: false });
      else lines.push(line);
    },
    lines: iter,
    close: () => {
      if (closed) return;
      closed = true;
      rl.close();
      while (waiters.length > 0) {
        waiters.shift()?.resolve({ value: undefined, done: true });
      }
    },
  };
}

async function createRichClient(
  options?: CreateTuichatClientOptions
): Promise<TuichatTuiClient> {
  const store = createStore({ commands: options?.commands });
  store.newChat();

  const renderer = await createCliRenderer({
    exitOnCtrlC: false,
  });

  const handleFatal = (error: unknown) => {
    try {
      (renderer as unknown as { destroy?: () => void }).destroy?.();
    } catch {
      // best-effort
    }
    const msg = error instanceof Error ? error.stack ?? error.message : String(error);
    process.stderr.write(`[tuichat] fatal render error:\n${msg}\n`);
    process.exit(1);
  };

  const root = createRoot(renderer);
  root.render(
    createElement(MountedApp, { store, onFatalError: handleFatal })
  );

  return {
    mode: "tui",
    store,
    renderer,
    root: root as { unmount(): void },
  };
}

export async function createTuichatClient(
  options?: CreateTuichatClientOptions
): Promise<TuichatClient> {
  if (activeInstance) {
    throw new Error(
      "tuichat: a client is already active in this process. Only one tuichat provider can run at a time."
    );
  }
  const client = isInteractive()
    ? await createRichClient(options)
    : await createPlainClient();
  activeInstance = client;
  return client;
}

export async function destroyTuichatClient(
  client: TuichatClient
): Promise<void> {
  if (client.mode === "plain") {
    client.close();
  } else {
    client.store.closeInput();
    try {
      client.root.unmount();
    } catch {
      // best-effort unmount
    }
    try {
      (client.renderer as unknown as { destroy?: () => void }).destroy?.();
    } catch {
      // best-effort destroy
    }
  }
  if (activeInstance === client) {
    activeInstance = null;
  }
}
