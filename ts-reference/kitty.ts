const APC_START = "\x1b_G";
const APC_END = "\x1b\\";
const PLACEHOLDER_CHAR = "\u{10EEEE}";

const ROW_COL_DIACRITICS = [
  0x0305, 0x030d, 0x030e, 0x0310, 0x0312, 0x033d, 0x033e, 0x033f, 0x0346,
  0x034a, 0x034b, 0x034c, 0x0350, 0x0351, 0x0352, 0x0357, 0x035b, 0x0363,
  0x0364, 0x0365, 0x0366, 0x0367, 0x0368, 0x0369, 0x036a, 0x036b, 0x036c,
  0x036d, 0x036e, 0x036f, 0x0483, 0x0484, 0x0485, 0x0486, 0x0487, 0x0592,
  0x0593, 0x0594, 0x0595, 0x0597, 0x0598, 0x0599, 0x059c, 0x059d, 0x059e,
  0x059f, 0x05a0, 0x05a1, 0x05a8, 0x05a9, 0x05ab, 0x05ac, 0x05af, 0x05c4,
  0x0610, 0x0611, 0x0612, 0x0613, 0x0614, 0x0615, 0x0616, 0x0617, 0x0657,
  0x0658, 0x0659, 0x065a, 0x065b, 0x065d, 0x065e, 0x06d6, 0x06d7, 0x06d8,
  0x06d9, 0x06da, 0x06db, 0x06dc, 0x06df, 0x06e0, 0x06e1, 0x06e2, 0x06e4,
  0x06e7, 0x06e8, 0x06eb, 0x06ec, 0x0730, 0x0732, 0x0733, 0x0735, 0x0736,
  0x073a, 0x073d, 0x073f, 0x0740, 0x0741, 0x0743, 0x0745, 0x0747, 0x0749,
  0x074a, 0x07eb, 0x07ec, 0x07ed, 0x07ee, 0x07ef, 0x07f0, 0x07f1, 0x07f3,
  0x0816, 0x0817, 0x0818, 0x0819, 0x081b, 0x081c, 0x081d, 0x081e, 0x081f,
  0x0820, 0x0821, 0x0822, 0x0823, 0x0825, 0x0826, 0x0827, 0x0829, 0x082a,
  0x082b, 0x082c, 0x082d, 0x0951, 0x0953, 0x0954, 0x0f82, 0x0f83, 0x0f86,
  0x0f87, 0x135d, 0x135e, 0x135f, 0x17dd, 0x193a, 0x1a17, 0x1a75, 0x1a76,
  0x1a77, 0x1a78, 0x1a79, 0x1a7a, 0x1a7b, 0x1a7c, 0x1b6b, 0x1b6d, 0x1b6e,
  0x1b6f, 0x1b70, 0x1b71, 0x1b72, 0x1b73, 0x1cd0, 0x1cd1, 0x1cd2, 0x1cda,
  0x1cdb, 0x1ce0, 0x1dc0, 0x1dc1, 0x1dc3, 0x1dc4, 0x1dc5, 0x1dc6, 0x1dc7,
  0x1dc8, 0x1dc9, 0x1dcb, 0x1dcc, 0x1dd1, 0x1dd2, 0x1dd3, 0x1dd4, 0x1dd5,
  0x1dd6, 0x1dd7, 0x1dd8, 0x1dd9, 0x1dda, 0x1ddb, 0x1ddc, 0x1ddd, 0x1dde,
  0x1ddf, 0x1de0, 0x1de1, 0x1de2, 0x1de3, 0x1de4, 0x1de5, 0x1de6, 0x1dfe,
  0x20d0, 0x20d1, 0x20d4, 0x20d5, 0x20d6, 0x20d7, 0x20db, 0x20dc, 0x20e1,
  0x20e7, 0x20e9, 0x20f0, 0x2cef, 0x2cf0, 0x2cf1, 0x2de0, 0x2de1, 0x2de2,
  0x2de3, 0x2de4, 0x2de5, 0x2de6, 0x2de7, 0x2de8, 0x2de9, 0x2dea, 0x2deb,
  0x2dec, 0x2ded, 0x2dee, 0x2def, 0x2df0, 0x2df1, 0x2df2, 0x2df3, 0x2df4,
  0x2df5, 0x2df6, 0x2df7, 0x2df8, 0x2df9, 0x2dfa, 0x2dfb, 0x2dfc, 0x2dfd,
  0x2dfe, 0x2dff, 0xa66f, 0xa67c, 0xa67d, 0xa6f0, 0xa6f1, 0xa8e0, 0xa8e1,
  0xa8e2, 0xa8e3, 0xa8e4, 0xa8e5, 0xa8e6, 0xa8e7, 0xa8e8, 0xa8e9, 0xa8ea,
  0xa8eb, 0xa8ec, 0xa8ed, 0xa8ee, 0xa8ef, 0xa8f0, 0xa8f1, 0xaab0,
];

const IMAGE_MIME_TO_FORMAT: Record<string, number> = {
  "image/png": 100,
  "image/jpeg": 100,
  "image/jpg": 100,
  "image/gif": 100,
  "image/webp": 100,
  "image/bmp": 100,
};

export function supportsKittyGraphics(): boolean {
  const env = process.env;
  if (env.TUICHAT_DISABLE_IMAGES === "1") return false;
  if (env.KITTY_WINDOW_ID) return true;
  if (env.TERM === "xterm-kitty") return true;
  if (env.TERM_PROGRAM === "ghostty") return true;
  if (env.TERM_PROGRAM === "WezTerm") return true;
  if (env.GHOSTTY_RESOURCES_DIR) return true;
  return false;
}

export function isSupportedImageMime(mimeType: string): boolean {
  return mimeType in IMAGE_MIME_TO_FORMAT;
}

const MAX_CHUNK = 4096;
let nextImageId = 1;

export function allocateImageId(): number {
  const id = nextImageId;
  nextImageId = (nextImageId % 16777215) + 1;
  return id;
}

const imageCache = new Map<string, Promise<number>>();

export function ensureImageTransmitted(
  cacheKey: string,
  read: () => Promise<Buffer>,
  cols: number,
  rows: number
): Promise<number> {
  const existing = imageCache.get(cacheKey);
  if (existing) return existing;
  const promise = (async () => {
    const id = allocateImageId();
    const buf = await read();
    transmitImageBytes(buf, id, cols, rows);
    return id;
  })();
  imageCache.set(cacheKey, promise);
  promise.catch(() => {
    imageCache.delete(cacheKey);
  });
  return promise;
}

function debugLog(msg: string): void {
  if (process.env.TUICHAT_DEBUG_IMAGES !== "1") return;
  try {
    const path = process.env.TUICHAT_DEBUG_LOG ?? "/tmp/tuichat-images.log";
    const { appendFileSync } = require("node:fs") as typeof import("node:fs");
    appendFileSync(path, `${new Date().toISOString()} ${msg}\n`);
  } catch {
    // best-effort
  }
}

function writeAPC(controls: string, payload?: string): void {
  const full =
    payload !== undefined
      ? `${APC_START}${controls};${payload}${APC_END}`
      : `${APC_START}${controls}${APC_END}`;
  debugLog(`APC ${controls}${payload ? ` (payload=${payload.length}b)` : ""}`);
  process.stdout.write(full);
}

export function transmitImageBytes(
  bytes: Buffer | Uint8Array,
  imageId: number,
  cols: number,
  rows: number
): void {
  const base64 = Buffer.from(bytes).toString("base64");
  let offset = 0;
  let first = true;
  while (offset < base64.length) {
    const chunk = base64.slice(offset, offset + MAX_CHUNK);
    const last = offset + MAX_CHUNK >= base64.length;
    const controls = first
      ? `a=t,i=${imageId},f=100,q=2,m=${last ? 0 : 1}`
      : `q=2,m=${last ? 0 : 1}`;
    writeAPC(controls, chunk);
    offset += MAX_CHUNK;
    first = false;
  }
  writeAPC(`a=p,U=1,i=${imageId},c=${cols},r=${rows},q=2`);
}

export function imageIdToRgb(imageId: number): {
  r: number;
  g: number;
  b: number;
} {
  return {
    r: (imageId >> 16) & 0xff,
    g: (imageId >> 8) & 0xff,
    b: imageId & 0xff,
  };
}

export function imageIdToHexColor(imageId: number): string {
  const { r, g, b } = imageIdToRgb(imageId);
  const toHex = (n: number) => n.toString(16).padStart(2, "0");
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
}

export function buildPlaceholderRows(cols: number, rows: number): string[] {
  const safeRows = Math.min(rows, ROW_COL_DIACRITICS.length);
  const safeCols = Math.min(cols, ROW_COL_DIACRITICS.length);
  const out: string[] = [];
  for (let r = 0; r < safeRows; r++) {
    const rowDiac = String.fromCodePoint(ROW_COL_DIACRITICS[r]!);
    let line = "";
    for (let c = 0; c < safeCols; c++) {
      const colDiac = String.fromCodePoint(ROW_COL_DIACRITICS[c]!);
      line += PLACEHOLDER_CHAR + rowDiac + colDiac;
    }
    out.push(line);
  }
  return out;
}

export function deleteImage(imageId: number): void {
  try {
    writeAPC(`a=d,d=I,i=${imageId},q=2`);
  } catch {
    // best-effort
  }
}
