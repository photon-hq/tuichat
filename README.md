# tuichat

A standalone, language-agnostic rich TUI chat subprocess for [Spectrum](https://docs.photon.codes/spectrum-ts) agents. Powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) + [Bubble Zone](https://github.com/lrstanley/bubblezone) + [Mosaic](https://github.com/charmbracelet/x/tree/main/mosaic). Speaks [JSON-RPC 2.0 over TCP](./PROTOCOL.md) so any Spectrum SDK adapter (TypeScript, Python, Go, Rust, …) can drive it with no runtime dependencies.

## Why a separate process?

Every Spectrum SDK needs a polished terminal UI, and reimplementing one in each language would waste effort. tuichat is the shared UI — a small compiled binary downloaded on first use by the SDK's adapter. The adapter handles provider-shape conversion; tuichat handles rendering, input, images, keybindings.

## Features

- Multi-chat sidebar with click-to-switch + `Ctrl+N` / `Ctrl+J` / `Ctrl+K` navigation
- Slash-command palette with `/new`, `/help`, plus adapter-registered commands
- Drag-and-drop file attachments (bracketed-paste + quoted-path fallback)
- Inline image preview: Kitty graphics protocol where available (Kitty, Ghostty, WezTerm), half-block Mosaic fallback elsewhere
- OSC 8 hyperlinks for URLs in messages + attachment filenames
- 2000-entry scrollback cap per chat with "… N older messages dropped" marker
- Scrollable log, sticky-bottom, typing indicator, per-chat input drafts

## Install

Download the pre-built binary for your platform from the [latest release](https://github.com/photon-hq/tuichat/releases/latest):

- `tuichat-darwin-arm64` (Apple Silicon)
- `tuichat-darwin-x64` (Intel macOS)
- `tuichat-linux-x64`
- `tuichat-linux-arm64`
- `tuichat-windows-x64.exe`

Verify with `SHA256SUMS` in the same release, `chmod +x`, move to `PATH`.

If you hit a Gatekeeper warning on first run on macOS: `xattr -d com.apple.quarantine /path/to/tuichat`. Notarization coming soon.

You normally don't run `tuichat` directly — your Spectrum SDK's terminal adapter spawns it for you.

## Usage (for humans, mostly for debugging)

```sh
# Start an adapter somewhere that listens on 127.0.0.1:12345
# Then:
tuichat --connect 127.0.0.1:12345
```

See [PROTOCOL.md](./PROTOCOL.md) for the wire contract.

## Build from source

Requires Go ≥ 1.24 and [Task](https://taskfile.dev/) (`brew install go-task`).

```sh
task build          # dist/tuichat (dev build, with symbols)
task build:release  # dist/tuichat (stripped)
task cross          # dist/tuichat-<target> for all 5 targets
task check          # fmt + vet + build
```

## Environment variables

- `TUICHAT_DISABLE_IMAGES=1` — disable all inline image preview (Kitty + Mosaic both).
- `TUICHAT_DEBUG_IMAGES=1` — append Kitty APC emissions to a debug log.
- `TUICHAT_DEBUG_LOG=<path>` — override the debug log path (default `/tmp/tuichat-images.log`).

## Contributing

The `ts-reference/` directory preserves the earlier TypeScript + OpenTUI implementation as a spec-of-behavior for the Go port. It's not built or tested; ignore when developing.

## License

MIT.
