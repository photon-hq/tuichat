# tuichat

A high-performance TUI chat provider for [Spectrum-TS](https://github.com/photon-hq/spectrum-ts) agents, powered by [OpenTUI React](https://opentui.com/).

A richer alternative to the built-in `terminal` provider — multi-chat sidebar, drag-drop attachments, floating image previews (Kitty graphics), slash-command palette, mouse hover. Falls back to plain readline when stdout isn't a TTY so agents still work over pipes and in CI.

## Install

```bash
bun add tuichat
```

`spectrum-ts` and `zod` are peer dependencies.

## Usage

```ts
import { Spectrum } from "spectrum-ts";
import { tuichat } from "tuichat";

const app = await Spectrum({
  providers: [
    tuichat.config({
      commands: [
        { name: "/attach", description: "send a demo attachment" },
      ],
    }),
  ],
});

for await (const [space, message] of app.messages) {
  if (message.content.type !== "text") continue;
  await space.responding(async () => {
    await space.send(`echo: ${message.content.text}`);
  });
}
```

Each Spectrum `space` becomes a chat in the sidebar. Create chats by pressing `Ctrl+N`, typing `/new`, or having the agent call `tuichat(app).space({ id: "whatever" })`.

## Keybindings

| Key                | Action                                  |
| ------------------ | --------------------------------------- |
| `Ctrl+N`           | New chat                                |
| `Ctrl+J` / `Ctrl+K`| Cycle chats (down / up)                 |
| `Ctrl+L`           | Clear active chat (marker only)         |
| `Ctrl+D`           | Toggle OpenTUI debug overlay            |
| `Ctrl+C`           | Exit                                    |
| `Tab`              | Complete slash command                  |
| `Esc`              | Cancel input + drop pending attachments |
| Drag a file in     | Attach (or paste the quoted path)       |
| Hover image row    | Floating preview (Kitty/Ghostty only)   |

Type `/help` inside the TUI to see the same reference.

## Slash commands

| Command | Behavior                                         |
| ------- | ------------------------------------------------ |
| `/new`  | Start a new chat (same as `Ctrl+N`)              |
| `/help` | Dump keybindings + env vars into the active chat |

Plus any commands you register via `tuichat.config({ commands: […] })`; these reach your agent's `for await` loop as normal text messages.

## Environment variables

| Variable                     | Effect                                                              |
| ---------------------------- | ------------------------------------------------------------------- |
| `TUICHAT_FORCE_TUI=1`        | Force rich TUI even without a TTY (tests, tmux/screen edge cases)   |
| `TUICHAT_FORCE_PLAIN=1`      | Force plain readline mode                                           |
| `TUICHAT_QUIET=1`            | Silence the plain-mode startup banner on stderr                     |
| `TUICHAT_DISABLE_IMAGES=1`   | Disable Kitty graphics image previews                               |
| `TUICHAT_DEBUG_IMAGES=1`     | Log APC sequences to `/tmp/tuichat-images.log` (or the override below) |
| `TUICHAT_DEBUG_LOG=<path>`   | Override the debug log path                                         |

## Image previews

Works in terminals that implement the Kitty graphics protocol with unicode placeholders — Kitty, Ghostty, and recent WezTerm. Detected automatically from `TERM`, `KITTY_WINDOW_ID`, `TERM_PROGRAM=ghostty`, `TERM_PROGRAM=WezTerm`, or `GHOSTTY_RESOURCES_DIR`. Images are uploaded once per attachment (cached by name) and rendered into a floating 40×10 cell panel on hover; the placeholder characters flow through OpenTUI's framebuffer as regular text, so the image never fights the renderer.

Other terminals fall back to a plain `[image: name.png 1.2MB]` label. Agent-sent attachments get spooled to `<tmpdir>/tuichat/<hash>.<ext>` so their filename is also an OSC 8 hyperlink that opens in the OS default viewer.

## Non-TTY behavior

When `stdout.isTTY` is false (piped output, CI, SSH without a PTY), tuichat prints a one-line banner to stderr and switches to plain readline:

- Each stdin line becomes one text message to the hardcoded `"tuichat"` space
- `actions.send` writes to stdout, mirroring the upstream `terminal` provider
- `startTyping` / `stopTyping` / reactions become no-ops
- No sidebar, no image preview, no drag-drop

Set `TUICHAT_QUIET=1` to suppress the banner.

## Limitations

- **Single instance per process.** `createTuichatClient` throws if a second one tries to start; OpenTUI can't share the alternate screen buffer.
- **Scrollback is capped** at 2000 entries per chat; older messages are dropped with a marker at the top of the log.
- **No persistence.** Chats live only for the process lifetime.
- **Streaming.** Spectrum models each `send()` as a discrete message, so streamed token responses become many entries. If you're streaming, buffer agent-side before calling `send`.

## License

MIT
