# tuichat protocol

Language-agnostic JSON-RPC 2.0 over TCP/localhost. The adapter (per-language Spectrum binding) spawns the `tuichat` binary, reads its one-line ready banner from stdout to discover the port, connects, and drives the TUI via this protocol.

## Transport

- **Socket**: TCP on `127.0.0.1`, OS-assigned ephemeral port (tuichat binds `:0`).
- **Framing**: LSP-style headers.
  ```
  Content-Length: <byte-length-of-body>\r\n
  \r\n
  <body>
  ```
  `body` is UTF-8 encoded JSON-RPC 2.0. Other `Content-*` headers are ignored.

## Lifecycle

1. Adapter spawns `tuichat` as a subprocess (no args required).
2. tuichat binds `127.0.0.1:0`, then prints exactly one line to stdout:
   ```json
   {"ready":true,"port":54321,"protocolVersion":"1"}
   ```
   Followed by a single `\n`. Adapter must read one line and parse JSON before doing anything else with the subprocess's stdout (after this, tuichat may take over stdout for the TUI alternate screen).
3. Adapter dials `127.0.0.1:<port>`.
4. Adapter sends `initialize` request; tuichat responds. TUI renderer boots.
5. Request/notification traffic flows either direction.
6. Adapter sends `shutdown` request, then closes the socket, then `SIGTERM`s the subprocess.

If the socket closes without `shutdown`, tuichat exits non-zero.

## JSON-RPC envelope

Requests (either direction, expect a response):
```json
{"jsonrpc":"2.0","id":1,"method":"send","params":{...}}
```

Responses:
```json
{"jsonrpc":"2.0","id":1,"result":{...}}
{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"..."}}
```

Notifications (no response expected; no `id`):
```json
{"jsonrpc":"2.0","method":"message","params":{...}}
```

Error codes follow JSON-RPC 2.0 conventions plus:
- `-32099` unsupported content type

## Content shape

Content payloads are JSON objects discriminated on `type`. Bytes are base64 unless a local filesystem path is available (user drag-drop), in which case `path` is sent instead and the peer is expected to be on the same machine.

```ts
type Content =
  | { type: "text"; text: string }
  | { type: "attachment"; name: string; mimeType: string; size?: number; bytes?: string /* base64 */; path?: string }
  | { type: "voice"; name?: string; mimeType: string; size?: number; bytes?: string; path?: string }
  | { type: "contact"; name?: { formatted?: string; first?: string; last?: string; middle?: string; prefix?: string; suffix?: string }; vcard?: string }
  | { type: "custom"; raw: unknown };
```

Exactly one of `bytes` or `path` must be present for `attachment`/`voice`.

## Methods (client → server)

### `initialize`
Request.
```ts
params: {
  commands?: { name: string; description?: string }[];
  clientInfo?: { name: string; version?: string };
}
result: {
  protocolVersion: "1";
  serverInfo: { name: "tuichat"; version: string };
}
```
Must be the first request. No other request is accepted before `initialize` resolves.

### `send`
Request. Display a message from the agent in a chat.
```ts
params: { spaceId: string; content: Content }
result: { id: string; timestamp: string /* ISO 8601 */ }
```
If `spaceId` does not exist, tuichat creates the chat automatically.

### `startTyping`
Request. Show the typing indicator in a chat.
```ts
params: { spaceId: string }
result: null
```

### `stopTyping`
Request. Hide the typing indicator.
```ts
params: { spaceId: string }
result: null
```

### `reactToMessage`
Request. Attach an emoji reaction to a previously-sent message.
```ts
params: { spaceId: string; messageId: string; reaction: string }
result: null
```

### `replyToMessage`
Request. Send a message as a quoted reply to an earlier one.
```ts
params: { spaceId: string; messageId: string; content: Content }
result: { id: string; timestamp: string }
```

### `ensureSpace`
Request. Create a chat if it doesn't exist, without sending anything.
```ts
params: { id: string }
result: null
```

### `shutdown`
Request. Graceful teardown. Adapter should close the socket after the response.
```ts
params: null
result: null
```

## Notifications (server → client)

### `message`
Notification. Fires whenever the user submits in any chat.
```ts
params: {
  id: string;
  spaceId: string;
  senderId: string;      // always "terminal-tui-user"
  content: Content;
  timestamp: string;
}
```

### `spaceCreated`
Notification. Fires when the user creates a new chat (Ctrl+N or `/new`). Adapter may track this to keep its own chat-id mirror.
```ts
params: { id: string; createdAt: string }
```

## Versioning

`protocolVersion` on the ready banner and on `initialize` result. Breaking changes bump to `"2"`. Adapters should refuse to connect if they see a major version they don't understand.
