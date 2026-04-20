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

export async function createTuichatClient(
  options?: CreateTuichatClientOptions
): Promise<TuichatClient> {
  const store = createStore({ commands: options?.commands });

  const renderer = await createCliRenderer({
    exitOnCtrlC: false,
  });

  const root = createRoot(renderer);
  root.render(createElement(MountedApp, { store }));

  return {
    store,
    renderer,
    root: root as { unmount(): void },
  };
}

export async function destroyTuichatClient(client: TuichatClient): Promise<void> {
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
