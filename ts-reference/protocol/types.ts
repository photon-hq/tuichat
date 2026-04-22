export const PROTOCOL_VERSION = "1" as const;

export type Content =
  | { type: "text"; text: string }
  | {
      type: "attachment";
      name: string;
      mimeType: string;
      size?: number;
      bytes?: string;
      path?: string;
    }
  | {
      type: "voice";
      name?: string;
      mimeType: string;
      size?: number;
      bytes?: string;
      path?: string;
    }
  | {
      type: "contact";
      name?: {
        formatted?: string;
        first?: string;
        last?: string;
        middle?: string;
        prefix?: string;
        suffix?: string;
      };
      vcard?: string;
    }
  | { type: "custom"; raw: unknown };

export interface CommandDef {
  name: string;
  description?: string;
}

export interface SendResult {
  id: string;
  timestamp: string;
}

export interface InitializeParams {
  commands?: CommandDef[];
  clientInfo?: { name: string; version?: string };
}

export interface InitializeResult {
  protocolVersion: typeof PROTOCOL_VERSION;
  serverInfo: { name: "tuichat"; version: string };
}

export interface SendParams {
  spaceId: string;
  content: Content;
}

export interface TypingParams {
  spaceId: string;
}

export interface ReactParams {
  spaceId: string;
  messageId: string;
  reaction: string;
}

export interface ReplyParams {
  spaceId: string;
  messageId: string;
  content: Content;
}

export interface EnsureSpaceParams {
  id: string;
}

export interface MessageNotification {
  id: string;
  spaceId: string;
  senderId: string;
  content: Content;
  timestamp: string;
}

export interface SpaceCreatedNotification {
  id: string;
  createdAt: string;
}

export interface RpcRequest {
  jsonrpc: "2.0";
  id: number | string;
  method: string;
  params?: unknown;
}

export interface RpcNotification {
  jsonrpc: "2.0";
  method: string;
  params?: unknown;
}

export interface RpcResponse {
  jsonrpc: "2.0";
  id: number | string;
  result?: unknown;
  error?: { code: number; message: string; data?: unknown };
}

export type RpcMessage = RpcRequest | RpcNotification | RpcResponse;

export const ERROR_CODES = {
  parseError: -32700,
  invalidRequest: -32600,
  methodNotFound: -32601,
  invalidParams: -32602,
  internalError: -32603,
  serverError: -32000,
  unsupportedContent: -32099,
} as const;
