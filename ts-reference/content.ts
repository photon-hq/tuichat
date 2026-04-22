export interface ContactName {
  formatted?: string;
  first?: string;
  last?: string;
  middle?: string;
  prefix?: string;
  suffix?: string;
}

export type Content =
  | { type: "text"; text: string }
  | {
      type: "attachment";
      name: string;
      mimeType: string;
      size?: number;
      read: () => Promise<Buffer>;
      path?: string;
    }
  | {
      type: "voice";
      name?: string;
      mimeType: string;
      size?: number;
      read: () => Promise<Buffer>;
      path?: string;
    }
  | { type: "contact"; name?: ContactName; vcard?: string }
  | { type: "custom"; raw: unknown };
