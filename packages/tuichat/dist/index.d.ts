import * as spectrum_ts from 'spectrum-ts';
import { Content as Content$1 } from 'spectrum-ts';
import z__default from 'zod';
import { CliRenderer } from '@opentui/core';

declare const contentSchema: z__default.ZodDiscriminatedUnion<[z__default.ZodObject<{
    type: z__default.ZodLiteral<"text">;
    text: z__default.ZodString;
}, z__default.core.$strip>, z__default.ZodObject<{
    type: z__default.ZodLiteral<"custom">;
    raw: z__default.ZodUnknown;
}, z__default.core.$strip>, z__default.ZodObject<{
    type: z__default.ZodLiteral<"attachment">;
    name: z__default.ZodString;
    mimeType: z__default.ZodString;
    size: z__default.ZodOptional<z__default.ZodNumber>;
    read: z__default.ZodFunction<z__default.ZodTuple<readonly [], null>, z__default.ZodPromise<z__default.ZodCustom<Buffer<ArrayBufferLike>, Buffer<ArrayBufferLike>>>>;
    stream: z__default.ZodFunction<z__default.ZodTuple<readonly [], null>, z__default.ZodPromise<z__default.ZodCustom<ReadableStream<unknown>, ReadableStream<unknown>>>>;
}, z__default.core.$strip>], "type">;
type Content = z__default.infer<typeof contentSchema>;
interface ContentBuilder {
    build(): Promise<Content>;
}
type ContentInput = string | ContentBuilder;

interface Space<_Def = unknown> {
    readonly __platform: string;
    readonly id: string;
    responding<T>(fn: () => T | Promise<T>): Promise<T>;
    send(...content: [ContentInput, ...ContentInput[]]): Promise<void>;
    startTyping(): Promise<void>;
    stopTyping(): Promise<void>;
}

interface User {
    readonly __platform: string;
    readonly id: string;
}

type ResolvedSpace = Pick<Space, "id">;
type ResolvedUser = Pick<User, "id">;
type ProviderMessage<TSender extends ResolvedUser = ResolvedUser, TSpace extends ResolvedSpace = ResolvedSpace, TExtra extends object = Record<never, never>> = {
    id: string;
    content: Content;
    sender: TSender;
    space: TSpace;
    timestamp?: Date;
} & TExtra;

interface CommandDef {
    name: string;
    description?: string;
}
type Role = "user" | "agent" | "system";
interface LogEntry {
    id: string;
    role: Role;
    content: Content$1;
    timestamp: Date;
    replyTo?: string;
    reactions: string[];
    attachmentPath?: string;
}
interface PendingAttachment {
    path: string;
    name: string;
    size?: number;
}
interface HoveredPreview {
    cacheKey: string;
    name: string;
    read: () => Promise<Buffer>;
}
interface Snapshot {
    entries: readonly LogEntry[];
    typing: boolean;
    commands: readonly CommandDef[];
    pendingAttachments: readonly PendingAttachment[];
    hoveredPreview: HoveredPreview | null;
}
type Listener = () => void;
interface Store {
    subscribe(listener: Listener): () => void;
    getSnapshot(): Snapshot;
    appendAgent(content: Content$1, opts?: {
        replyTo?: string;
    }): string;
    appendUser(content: Content$1, opts?: {
        attachmentPath?: string;
    }): string;
    appendSystem(text: string): void;
    setTyping(value: boolean): void;
    react(messageId: string, emoji: string): void;
    patchEntry(id: string, patch: Partial<LogEntry>): void;
    pushUserInput(content: Content$1): void;
    nextUserInput(): Promise<IteratorResult<Content$1>>;
    closeInput(): void;
    addPendingAttachment(att: PendingAttachment): void;
    removePendingAttachment(index: number): void;
    clearPendingAttachments(): void;
    setHoveredPreview(preview: HoveredPreview | null): void;
}

interface TuichatClient {
    store: Store;
    renderer: CliRenderer;
    root: {
        unmount(): void;
    };
}

declare const tuichat: spectrum_ts.Platform<spectrum_ts.PlatformDef<"tuichat", z__default.ZodObject<{
    commands: z__default.ZodOptional<z__default.ZodArray<z__default.ZodObject<{
        name: z__default.ZodString;
        description: z__default.ZodOptional<z__default.ZodString>;
    }, z__default.core.$strip>>>;
}, z__default.core.$strip>, z__default.ZodType<object, unknown, z__default.core.$ZodTypeInternals<object, unknown>> | undefined, z__default.ZodType<object, unknown, z__default.core.$ZodTypeInternals<object, unknown>> | undefined, z__default.ZodType<object, unknown, z__default.core.$ZodTypeInternals<object, unknown>> | undefined, TuichatClient, {
    id: string;
}, {
    id: string;
}, undefined, ProviderMessage<{
    id: string;
}, {
    id: string;
}, Record<never, never>>, {
    messages({ client }: {
        client: TuichatClient;
        config: {
            commands?: {
                name: string;
                description?: string | undefined;
            }[] | undefined;
        };
    }): AsyncGenerator<{
        id: `${string}-${string}-${string}-${string}-${string}`;
        content: {
            type: "text";
            text: string;
        } | {
            type: "custom";
            raw: unknown;
        } | {
            type: "attachment";
            name: string;
            mimeType: string;
            read: z__default.core.$InferOuterFunctionType<z__default.ZodTuple<readonly [], null>, z__default.ZodPromise<z__default.ZodCustom<Buffer<ArrayBufferLike>, Buffer<ArrayBufferLike>>>>;
            stream: z__default.core.$InferOuterFunctionType<z__default.ZodTuple<readonly [], null>, z__default.ZodPromise<z__default.ZodCustom<ReadableStream<unknown>, ReadableStream<unknown>>>>;
            size?: number | undefined;
        };
        sender: {
            id: string;
        };
        space: {
            id: string;
        };
        timestamp: Date;
    }, void, any>;
}>> & Readonly<Record<never, never>>;

export { type CommandDef, type LogEntry, type PendingAttachment, type Role, type Snapshot, type Store, type TuichatClient, tuichat };
