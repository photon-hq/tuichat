# Testing tuichat end-to-end

No local GitHub Release exists yet, so the spectrum-ts adapter would try to download `v0.1.0` and fail. Use the `TUICHAT_BINARY` env var to point it at your local build instead.

## One-time setup

You need the spectrum-ts fork checked out alongside this repo:

```
/Volumes/Mac/tuichat/            ← this repo (Go binary + Taskfile + PROTOCOL.md)
/Volumes/Mac/spectrum-ts-fork/   ← photon-hq/spectrum-ts fork with `feat/terminal-provider-tui`
```

Install fork deps once:

```sh
cd /Volumes/Mac/spectrum-ts-fork && bun install
```

## Build the binary

```sh
cd /Volumes/Mac/tuichat
task build     # dist/tuichat (dev, with symbols)
# or
task build:release  # stripped + trimpath, matches what CI will ship
```

## Rich TUI mode — interactive smoke test

Run the fork's echo example in a real terminal:

```sh
cd /Volumes/Mac/spectrum-ts-fork/examples/basic
TUICHAT_BINARY=/Volumes/Mac/tuichat/dist/tuichat bun run index.ts
```

You should see the full TUI: sidebar with `chat-1`, title bar, input at the bottom. Type, hit Enter, the agent echoes.

Try:
- `Ctrl+N` — new chat (sidebar grows).
- `Ctrl+J` / `Ctrl+K` — cycle chats.
- `/he<Tab>` — `/help` autocompletes from the palette.
- `/help<Enter>` — keybindings dumped into the log.
- `/new<Enter>` — shortcut alias for `Ctrl+N`.
- Drag a file onto the window — appears as a chip above the input.
- Drop a `.png` / `.jpg` — once sent, clicking the chip opens a floating preview (Kitty/Ghostty use native graphics, other terminals get the half-block Mosaic fallback).
- Hover or Cmd+click a URL in a text message — clickable via OSC 8.
- `Ctrl+C` — clean exit, terminal restored.

## Plain mode — non-TTY smoke test

Same binary auto-detects non-TTY and runs a readline shim:

```sh
echo "hello world" | TUICHAT_BINARY=/Volumes/Mac/tuichat/dist/tuichat bun run /Volumes/Mac/spectrum-ts-fork/examples/basic/index.ts
```

Prints `echo: hello world`. Set `TUICHAT_QUIET=1` to silence the "non-TTY detected" stderr banner.

## Env overrides (useful during dev)

| Variable | Meaning |
| --- | --- |
| `TUICHAT_BINARY=/path/to/tuichat` | Skip the version-pinned download; use a local binary directly. |
| `TUICHAT_VERSION=0.1.0` | Override the version the adapter tries to download. |
| `TUICHAT_FORCE_TUI=1` | Force rich mode even without a TTY (useful for `tmux`-inside-something-weird cases). |
| `TUICHAT_FORCE_PLAIN=1` | Force plain readline mode. |
| `TUICHAT_QUIET=1` | Silence the "non-TTY detected" stderr banner. |
| `TUICHAT_DISABLE_IMAGES=1` | Disable inline image preview (Kitty + Mosaic both). |
| `TUICHAT_DEBUG_IMAGES=1` | Append Kitty APC emissions to `/tmp/tuichat-images.log` (or `TUICHAT_DEBUG_LOG` override). |

## Cross-compile the full matrix locally

Matches what CI will produce (pre-UPX, pre-codesign):

```sh
cd /Volumes/Mac/tuichat
task cross
ls -lh dist/
```

## What's _not_ testable locally right now

- **macOS codesigning** — CI-only; runs on `macos-14` with the org-level `APP_PRIVATE_KEY` + `DEVELOPER_ID_INSTALLER_NAME` secrets.
- **UPX compression** — CI applies this to Linux/Windows builds only.
- **Download flow** from GitHub Releases — requires an actual `v0.1.0` release. First real tag push will verify this.
