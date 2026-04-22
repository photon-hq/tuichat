import { createCliRenderer, type CliRenderer } from "@opentui/core";
import { createRoot } from "@opentui/react";
import { createElement } from "react";
import { MountedApp } from "./ui/App";
import { type CommandDef, createStore, type Store } from "./store";

export interface TuichatClient {
  store: Store;
  renderer: CliRenderer;
  root: { unmount(): void };
}

export interface CreateTuichatClientOptions {
  commands?: readonly CommandDef[];
}

let activeInstance: TuichatClient | null = null;

export async function createTuichatClient(
  options?: CreateTuichatClientOptions
): Promise<TuichatClient> {
  if (activeInstance) {
    throw new Error(
      "tuichat: a client is already active in this process. Only one tuichat renderer can run at a time."
    );
  }

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
    const msg =
      error instanceof Error ? error.stack ?? error.message : String(error);
    process.stderr.write(`[tuichat] fatal render error:\n${msg}\n`);
    process.exit(1);
  };

  const root = createRoot(renderer);
  root.render(createElement(MountedApp, { store, onFatalError: handleFatal }));

  const client: TuichatClient = {
    store,
    renderer,
    root: root as { unmount(): void },
  };
  activeInstance = client;
  return client;
}

export async function destroyTuichatClient(
  client: TuichatClient
): Promise<void> {
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
  if (activeInstance === client) {
    activeInstance = null;
  }
}
