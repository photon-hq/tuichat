#!/usr/bin/env bun
import { startServer } from "../server";

function parseArgs(argv: string[]): { host: string; port: number } {
  let connectArg: string | undefined;
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i]!;
    if (arg === "--connect") {
      connectArg = argv[++i];
    } else if (arg.startsWith("--connect=")) {
      connectArg = arg.slice("--connect=".length);
    } else if (arg === "--help" || arg === "-h") {
      process.stderr.write(
        "tuichat — rich TUI chat subprocess for Spectrum adapters\n" +
          "Usage: tuichat --connect HOST:PORT\n" +
          "Dials back into the adapter, which must be listening on the given address.\n"
      );
      process.exit(0);
    }
  }
  if (!connectArg) {
    process.stderr.write(
      "tuichat: --connect HOST:PORT is required. See --help.\n"
    );
    process.exit(2);
  }
  const idx = connectArg.lastIndexOf(":");
  if (idx < 0) {
    process.stderr.write(`tuichat: invalid --connect value: ${connectArg}\n`);
    process.exit(2);
  }
  const host = connectArg.slice(0, idx);
  const port = Number.parseInt(connectArg.slice(idx + 1), 10);
  if (!host || Number.isNaN(port)) {
    process.stderr.write(`tuichat: invalid --connect value: ${connectArg}\n`);
    process.exit(2);
  }
  return { host, port };
}

async function main(): Promise<void> {
  const args = parseArgs(process.argv.slice(2));
  await startServer(args);
}

main().catch((err) => {
  process.stderr.write(
    `[tuichat] fatal: ${err instanceof Error ? err.stack ?? err.message : String(err)}\n`
  );
  process.exit(1);
});
