#!/usr/bin/env bun
import { startServer } from "../server";

function parseArgs(argv: string[]): { host?: string } {
  const out: { host?: string } = {};
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i]!;
    if (arg === "--host") {
      out.host = argv[++i];
    } else if (arg.startsWith("--host=")) {
      out.host = arg.slice("--host=".length);
    } else if (arg === "--help" || arg === "-h") {
      process.stderr.write(
        "tuichat — rich TUI chat subprocess for Spectrum adapters\n" +
          "Usage: tuichat [--host 127.0.0.1]\n" +
          "Binds an ephemeral TCP port and prints {\"ready\":true,\"port\":N} on stdout.\n"
      );
      process.exit(0);
    }
  }
  return out;
}

async function main(): Promise<void> {
  const args = parseArgs(process.argv.slice(2));
  await startServer({ host: args.host });
}

main().catch((err) => {
  process.stderr.write(
    `[tuichat] fatal: ${err instanceof Error ? err.stack ?? err.message : String(err)}\n`
  );
  process.exit(1);
});
