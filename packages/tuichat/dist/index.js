var __require = /* @__PURE__ */ ((x) => typeof require !== "undefined" ? require : typeof Proxy !== "undefined" ? new Proxy(x, {
  get: (a, b) => (typeof require !== "undefined" ? require : a)[b]
}) : x)(function(x) {
  if (typeof require !== "undefined") return require.apply(this, arguments);
  throw Error('Dynamic require of "' + x + '" is not supported');
});

// src/provider.ts
import z from "zod";
import { definePlatform } from "spectrum-ts";

// src/client.ts
import { createCliRenderer } from "@opentui/core";
import { createRoot } from "@opentui/react";
import { createElement } from "react";

// src/ui/App.tsx
import {
  useCallback,
  useEffect as useEffect3,
  useMemo,
  useRef,
  useState as useState3,
  useSyncExternalStore
} from "react";
import { useKeyboard, useRenderer } from "@opentui/react";
import { attachment } from "spectrum-ts";

// src/drop.ts
import { existsSync, statSync } from "fs";
import { basename } from "path";
function parseDroppedPath(raw) {
  let s = raw.trim();
  if (s.length === 0) return null;
  if (s.startsWith("file://")) {
    try {
      s = decodeURIComponent(s.slice(7));
    } catch {
      return null;
    }
  }
  if (s.startsWith("'") && s.endsWith("'") || s.startsWith('"') && s.endsWith('"')) {
    s = s.slice(1, -1);
  }
  s = s.replace(/\\(.)/g, "$1");
  if (/[\r\n]/.test(s)) return null;
  return s;
}
function resolveDroppedAttachment(raw) {
  const path = parseDroppedPath(raw);
  if (!path) return null;
  try {
    if (!existsSync(path)) return null;
    const st = statSync(path);
    if (!st.isFile()) return null;
    return {
      path,
      name: basename(path),
      size: st.size
    };
  } catch {
    return null;
  }
}
function extractDroppedPaths(value) {
  const attachments = [];
  const quoted = /(['"])((?:(?!\1).)+)\1/g;
  let cleaned = value;
  let match;
  const toRemove = [];
  while ((match = quoted.exec(value)) !== null) {
    const full = match[0];
    const inner = match[2];
    const resolved = resolveDroppedAttachment(inner);
    if (resolved) {
      attachments.push(resolved);
      toRemove.push(full);
    }
  }
  for (const r of toRemove) {
    cleaned = cleaned.replace(r, "");
  }
  if (attachments.length > 0) {
    cleaned = cleaned.replace(/\s+/g, " ").trim();
  }
  return { cleaned, attachments };
}

// src/ui/theme.ts
var theme = {
  title: "tuichat",
  colors: {
    user: "#7dd3fc",
    agent: "#a78bfa",
    system: "#6b7280",
    timestamp: "#4b5563",
    border: "#374151",
    suggestionBg: "#1f2937",
    suggestionSelectedBg: "#374151",
    typing: "#f59e0b",
    attachment: "#34d399",
    custom: "#f472b6",
    input: "#e5e7eb",
    prompt: "#7dd3fc"
  },
  prefix: {
    user: "you",
    agent: "agent",
    system: "sys"
  }
};
var formatTime = (d) => {
  const hh = d.getHours().toString().padStart(2, "0");
  const mm = d.getMinutes().toString().padStart(2, "0");
  const ss = d.getSeconds().toString().padStart(2, "0");
  return `${hh}:${mm}:${ss}`;
};
var roleColor = (role) => theme.colors[role];
var formatBytes = (bytes) => {
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}kB`;
  return `${(bytes / 1024 / 1024).toFixed(1)}MB`;
};

// src/ui/Attachments.tsx
import { jsx } from "@opentui/react/jsx-runtime";
function Attachments({ pending }) {
  if (pending.length === 0) return null;
  return /* @__PURE__ */ jsx(
    "box",
    {
      style: {
        height: 1,
        flexShrink: 0,
        flexDirection: "row",
        backgroundColor: theme.colors.suggestionBg,
        paddingLeft: 1,
        paddingRight: 1
      },
      children: /* @__PURE__ */ jsx("text", { children: pending.map((a, i) => {
        const size = a.size !== void 0 ? ` ${formatBytes(a.size)}` : "";
        const label = `\u{1F4CE} ${a.name}${size}`;
        const sep = i > 0 ? "  " : "";
        return /* @__PURE__ */ jsx(
          "span",
          {
            style: {
              fg: theme.colors.attachment,
              bg: theme.colors.suggestionBg
            },
            children: sep + label
          },
          a.path
        );
      }) })
    }
  );
}

// src/ui/KittyImage.tsx
import { useEffect, useState } from "react";

// src/kitty.ts
var APC_START = "\x1B_G";
var APC_END = "\x1B\\";
var PLACEHOLDER_CHAR = "\u{10EEEE}";
var ROW_COL_DIACRITICS = [
  773,
  781,
  782,
  784,
  786,
  829,
  830,
  831,
  838,
  842,
  843,
  844,
  848,
  849,
  850,
  855,
  859,
  867,
  868,
  869,
  870,
  871,
  872,
  873,
  874,
  875,
  876,
  877,
  878,
  879,
  1155,
  1156,
  1157,
  1158,
  1159,
  1426,
  1427,
  1428,
  1429,
  1431,
  1432,
  1433,
  1436,
  1437,
  1438,
  1439,
  1440,
  1441,
  1448,
  1449,
  1451,
  1452,
  1455,
  1476,
  1552,
  1553,
  1554,
  1555,
  1556,
  1557,
  1558,
  1559,
  1623,
  1624,
  1625,
  1626,
  1627,
  1629,
  1630,
  1750,
  1751,
  1752,
  1753,
  1754,
  1755,
  1756,
  1759,
  1760,
  1761,
  1762,
  1764,
  1767,
  1768,
  1771,
  1772,
  1840,
  1842,
  1843,
  1845,
  1846,
  1850,
  1853,
  1855,
  1856,
  1857,
  1859,
  1861,
  1863,
  1865,
  1866,
  2027,
  2028,
  2029,
  2030,
  2031,
  2032,
  2033,
  2035,
  2070,
  2071,
  2072,
  2073,
  2075,
  2076,
  2077,
  2078,
  2079,
  2080,
  2081,
  2082,
  2083,
  2085,
  2086,
  2087,
  2089,
  2090,
  2091,
  2092,
  2093,
  2385,
  2387,
  2388,
  3970,
  3971,
  3974,
  3975,
  4957,
  4958,
  4959,
  6109,
  6458,
  6679,
  6773,
  6774,
  6775,
  6776,
  6777,
  6778,
  6779,
  6780,
  7019,
  7021,
  7022,
  7023,
  7024,
  7025,
  7026,
  7027,
  7376,
  7377,
  7378,
  7386,
  7387,
  7392,
  7616,
  7617,
  7619,
  7620,
  7621,
  7622,
  7623,
  7624,
  7625,
  7627,
  7628,
  7633,
  7634,
  7635,
  7636,
  7637,
  7638,
  7639,
  7640,
  7641,
  7642,
  7643,
  7644,
  7645,
  7646,
  7647,
  7648,
  7649,
  7650,
  7651,
  7652,
  7653,
  7654,
  7678,
  8400,
  8401,
  8404,
  8405,
  8406,
  8407,
  8411,
  8412,
  8417,
  8423,
  8425,
  8432,
  11503,
  11504,
  11505,
  11744,
  11745,
  11746,
  11747,
  11748,
  11749,
  11750,
  11751,
  11752,
  11753,
  11754,
  11755,
  11756,
  11757,
  11758,
  11759,
  11760,
  11761,
  11762,
  11763,
  11764,
  11765,
  11766,
  11767,
  11768,
  11769,
  11770,
  11771,
  11772,
  11773,
  11774,
  11775,
  42607,
  42620,
  42621,
  42736,
  42737,
  43232,
  43233,
  43234,
  43235,
  43236,
  43237,
  43238,
  43239,
  43240,
  43241,
  43242,
  43243,
  43244,
  43245,
  43246,
  43247,
  43248,
  43249,
  43696
];
var IMAGE_MIME_TO_FORMAT = {
  "image/png": 100,
  "image/jpeg": 100,
  "image/jpg": 100,
  "image/gif": 100,
  "image/webp": 100,
  "image/bmp": 100
};
function supportsKittyGraphics() {
  const env = process.env;
  if (env.TUICHAT_DISABLE_IMAGES === "1") return false;
  if (env.KITTY_WINDOW_ID) return true;
  if (env.TERM === "xterm-kitty") return true;
  if (env.TERM_PROGRAM === "ghostty") return true;
  if (env.TERM_PROGRAM === "WezTerm") return true;
  if (env.GHOSTTY_RESOURCES_DIR) return true;
  return false;
}
function isSupportedImageMime(mimeType) {
  return mimeType in IMAGE_MIME_TO_FORMAT;
}
var MAX_CHUNK = 4096;
var nextImageId = 1;
function allocateImageId() {
  const id = nextImageId;
  nextImageId = nextImageId % 16777215 + 1;
  return id;
}
var imageCache = /* @__PURE__ */ new Map();
function ensureImageTransmitted(cacheKey, read, cols, rows) {
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
function debugLog(msg) {
  if (process.env.TUICHAT_DEBUG_IMAGES !== "1") return;
  try {
    const path = process.env.TUICHAT_DEBUG_LOG ?? "/tmp/tuichat-images.log";
    const { appendFileSync } = __require("fs");
    appendFileSync(path, `${(/* @__PURE__ */ new Date()).toISOString()} ${msg}
`);
  } catch {
  }
}
function writeAPC(controls, payload) {
  const full = payload !== void 0 ? `${APC_START}${controls};${payload}${APC_END}` : `${APC_START}${controls}${APC_END}`;
  debugLog(`APC ${controls}${payload ? ` (payload=${payload.length}b)` : ""}`);
  process.stdout.write(full);
}
function transmitImageBytes(bytes, imageId, cols, rows) {
  const base64 = Buffer.from(bytes).toString("base64");
  let offset = 0;
  let first = true;
  while (offset < base64.length) {
    const chunk = base64.slice(offset, offset + MAX_CHUNK);
    const last = offset + MAX_CHUNK >= base64.length;
    const controls = first ? `a=t,i=${imageId},f=100,q=2,m=${last ? 0 : 1}` : `q=2,m=${last ? 0 : 1}`;
    writeAPC(controls, chunk);
    offset += MAX_CHUNK;
    first = false;
  }
  writeAPC(`a=p,U=1,i=${imageId},c=${cols},r=${rows},q=2`);
}
function imageIdToRgb(imageId) {
  return {
    r: imageId >> 16 & 255,
    g: imageId >> 8 & 255,
    b: imageId & 255
  };
}
function imageIdToHexColor(imageId) {
  const { r, g, b } = imageIdToRgb(imageId);
  const toHex = (n) => n.toString(16).padStart(2, "0");
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
}
function buildPlaceholderRows(cols, rows) {
  const safeRows = Math.min(rows, ROW_COL_DIACRITICS.length);
  const safeCols = Math.min(cols, ROW_COL_DIACRITICS.length);
  const out = [];
  for (let r = 0; r < safeRows; r++) {
    const rowDiac = String.fromCodePoint(ROW_COL_DIACRITICS[r]);
    let line = "";
    for (let c = 0; c < safeCols; c++) {
      const colDiac = String.fromCodePoint(ROW_COL_DIACRITICS[c]);
      line += PLACEHOLDER_CHAR + rowDiac + colDiac;
    }
    out.push(line);
  }
  return out;
}

// src/ui/KittyImage.tsx
import { jsx as jsx2 } from "@opentui/react/jsx-runtime";
function KittyImage({
  read,
  cacheKey,
  cols = 40,
  rows = 10
}) {
  const [imageId, setImageId] = useState(null);
  const [error, setError] = useState(null);
  useEffect(() => {
    let cancelled = false;
    ensureImageTransmitted(cacheKey, read, cols, rows).then((id) => {
      if (!cancelled) setImageId(id);
    }).catch((err) => {
      if (!cancelled) {
        setError(err instanceof Error ? err.message : String(err));
      }
    });
    return () => {
      cancelled = true;
    };
  }, [cacheKey, read, cols, rows]);
  if (error) {
    return /* @__PURE__ */ jsx2("text", { children: /* @__PURE__ */ jsx2("span", { style: { fg: theme.colors.system }, children: `[image preview failed: ${error}]` }) });
  }
  if (imageId === null) {
    return /* @__PURE__ */ jsx2("text", { children: /* @__PURE__ */ jsx2("span", { style: { fg: theme.colors.system }, children: "[loading image\u2026]" }) });
  }
  const fg = imageIdToHexColor(imageId);
  const lines = buildPlaceholderRows(cols, rows);
  return /* @__PURE__ */ jsx2("box", { style: { flexDirection: "column", height: lines.length, width: cols }, children: lines.map((line, i) => /* @__PURE__ */ jsx2("text", { children: /* @__PURE__ */ jsx2("span", { style: { fg }, children: line }) }, i)) });
}

// src/ui/MessageItem.tsx
import { useEffect as useEffect2, useState as useState2 } from "react";

// src/ui/linkify.tsx
import { jsx as jsx3 } from "@opentui/react/jsx-runtime";
var URL_REGEX = /https?:\/\/[^\s<>()\[\]{}]+/g;
var OSC8_START = "\x1B]8;;";
var OSC8_ST = "\x1B\\";
function wrapOSC8(url, text) {
  return `${OSC8_START}${url}${OSC8_ST}${text}${OSC8_START}${OSC8_ST}`;
}
function linkify(text, colors) {
  const out = [];
  let last = 0;
  let i = 0;
  URL_REGEX.lastIndex = 0;
  let match;
  while ((match = URL_REGEX.exec(text)) !== null) {
    if (match.index > last) {
      out.push(
        /* @__PURE__ */ jsx3("span", { style: { fg: colors.text }, children: text.slice(last, match.index) }, `t${i++}`)
      );
    }
    out.push(
      /* @__PURE__ */ jsx3(
        "span",
        {
          style: { fg: colors.link, attributes: 8 },
          children: wrapOSC8(match[0], match[0])
        },
        `l${i++}`
      )
    );
    last = match.index + match[0].length;
  }
  if (last < text.length) {
    out.push(
      /* @__PURE__ */ jsx3("span", { style: { fg: colors.text }, children: text.slice(last) }, `t${i++}`)
    );
  }
  return out;
}

// src/ui/MessageItem.tsx
import { jsx as jsx4, jsxs } from "@opentui/react/jsx-runtime";
function MessageItem({
  entry,
  entries,
  chatId,
  store
}) {
  const { content, role, timestamp, reactions, replyTo, attachmentPath } = entry;
  const prefix = theme.prefix[role].padEnd(5, " ");
  const color = roleColor(role);
  const replyToEntry = replyTo ? entries.find((e) => e.id === replyTo) : void 0;
  return /* @__PURE__ */ jsxs("box", { style: { flexDirection: "column", marginBottom: 0 }, children: [
    replyToEntry ? /* @__PURE__ */ jsxs("text", { children: [
      /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.border }, children: "  \u250C\u2500 " }),
      /* @__PURE__ */ jsx4("span", { style: { fg: roleColor(replyToEntry.role) }, children: theme.prefix[replyToEntry.role] }),
      /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.border }, children: ": " }),
      /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.system }, children: quoteOf(replyToEntry) })
    ] }) : null,
    /* @__PURE__ */ jsx4(
      ContentLine,
      {
        color,
        prefix,
        timestamp,
        content,
        attachmentPath,
        chatId,
        store
      }
    ),
    reactions.length > 0 ? /* @__PURE__ */ jsxs("text", { children: [
      /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.border }, children: "       " }),
      /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.system }, children: reactions.join(" ") })
    ] }) : null
  ] });
}
function quoteOf(entry) {
  const c = entry.content;
  if (c.type === "text") return trimTo(c.text, 60);
  if (c.type === "attachment") return `[attachment: ${c.name}]`;
  return "[custom]";
}
var trimTo = (s, n) => s.length <= n ? s : `${s.slice(0, n - 1)}\u2026`;
function ContentLine({
  color,
  prefix,
  timestamp,
  content,
  attachmentPath,
  chatId,
  store
}) {
  const timeLabel = `[${formatTime(timestamp)}] `;
  switch (content.type) {
    case "text":
      return /* @__PURE__ */ jsxs("text", { children: [
        /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.timestamp }, children: timeLabel }),
        /* @__PURE__ */ jsx4("span", { style: { fg: color }, children: prefix }),
        /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.border }, children: " \u203A " }),
        linkify(content.text, {
          text: theme.colors.input,
          link: theme.colors.prompt
        })
      ] });
    case "attachment":
      return /* @__PURE__ */ jsx4(
        AttachmentLine,
        {
          color,
          prefix,
          timeLabel,
          content,
          attachmentPath,
          chatId,
          store
        }
      );
    case "custom":
      return /* @__PURE__ */ jsxs("box", { style: { flexDirection: "column" }, children: [
        /* @__PURE__ */ jsxs("text", { children: [
          /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.timestamp }, children: timeLabel }),
          /* @__PURE__ */ jsx4("span", { style: { fg: color }, children: prefix }),
          /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.border }, children: " \u203A " }),
          /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.custom }, children: "[custom]" })
        ] }),
        /* @__PURE__ */ jsxs("text", { children: [
          /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.border }, children: "       " }),
          /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.system }, children: safeStringify(content.raw) })
        ] })
      ] });
    default:
      return null;
  }
}
function AttachmentLine({
  color,
  prefix,
  timeLabel,
  content,
  attachmentPath,
  chatId,
  store
}) {
  const [resolvedSize, setResolvedSize] = useState2(
    content.size
  );
  useEffect2(() => {
    if (resolvedSize !== void 0) return;
    let cancelled = false;
    content.read().then((buf) => {
      if (cancelled) return;
      setResolvedSize(buf.byteLength);
    }).catch(() => {
      if (cancelled) return;
      store.appendSystem(
        chatId,
        `failed to read attachment "${content.name}"`
      );
    });
    return () => {
      cancelled = true;
    };
  }, [content, resolvedSize, store]);
  const sizeLabel = resolvedSize !== void 0 ? ` ${formatBytes(resolvedSize)}` : "";
  const canPreview = supportsKittyGraphics() && isSupportedImageMime(content.mimeType);
  useEffect2(() => {
    if (!canPreview) return;
    ensureImageTransmitted(content.name, content.read, 40, 10).catch(() => {
    });
  }, [canPreview, content.name, content.read]);
  const onOver = canPreview ? () => store.setHoveredPreview({
    cacheKey: content.name,
    name: content.name,
    read: content.read
  }) : void 0;
  const onOut = canPreview ? () => store.setHoveredPreview(null) : void 0;
  return /* @__PURE__ */ jsx4(
    "box",
    {
      style: { flexDirection: "column" },
      onMouseOver: onOver,
      onMouseOut: onOut,
      children: /* @__PURE__ */ jsxs("text", { children: [
        /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.timestamp }, children: timeLabel }),
        /* @__PURE__ */ jsx4("span", { style: { fg: color }, children: prefix }),
        /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.border }, children: " \u203A " }),
        /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.attachment }, children: `[${canPreview ? "image" : "attachment"}: ` }),
        attachmentPath ? /* @__PURE__ */ jsx4(
          "span",
          {
            style: { fg: theme.colors.attachment, attributes: 8 },
            children: wrapOSC8(`file://${attachmentPath}`, content.name)
          }
        ) : /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.attachment }, children: content.name }),
        /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.attachment }, children: `${sizeLabel}]` }),
        canPreview ? /* @__PURE__ */ jsx4("span", { style: { fg: theme.colors.system }, children: "  (hover to preview)" }) : null
      ] })
    }
  );
}
function safeStringify(raw) {
  try {
    return JSON.stringify(raw, null, 2);
  } catch {
    return String(raw);
  }
}

// src/ui/Sidebar.tsx
import { jsx as jsx5, jsxs as jsxs2 } from "@opentui/react/jsx-runtime";
var SIDEBAR_WIDTH = 20;
function trim(s, n) {
  return s.length <= n ? s : `${s.slice(0, n - 1)}\u2026`;
}
function Sidebar({ chats, activeChatId, store }) {
  return /* @__PURE__ */ jsxs2(
    "box",
    {
      style: {
        width: SIDEBAR_WIDTH,
        flexDirection: "column",
        flexShrink: 0,
        border: ["right"],
        borderColor: theme.colors.border
      },
      children: [
        /* @__PURE__ */ jsx5(
          "box",
          {
            style: {
              height: 1,
              paddingLeft: 1,
              paddingRight: 1
            },
            children: /* @__PURE__ */ jsx5("text", { children: /* @__PURE__ */ jsx5("span", { style: { fg: theme.colors.system }, children: "Chats" }) })
          }
        ),
        /* @__PURE__ */ jsx5("box", { style: { flexDirection: "column", flexShrink: 0 }, children: chats.map((chat) => {
          const selected = chat.id === activeChatId;
          return /* @__PURE__ */ jsx5(
            "box",
            {
              style: {
                height: 1,
                paddingLeft: 1,
                paddingRight: 1
              },
              onMouseDown: () => store.setActiveChat(chat.id),
              children: /* @__PURE__ */ jsxs2("text", { children: [
                /* @__PURE__ */ jsx5(
                  "span",
                  {
                    style: {
                      fg: selected ? theme.colors.prompt : theme.colors.system
                    },
                    children: selected ? "\u203A " : "  "
                  }
                ),
                /* @__PURE__ */ jsx5(
                  "span",
                  {
                    style: {
                      fg: selected ? theme.colors.user : theme.colors.input
                    },
                    children: trim(chat.id, SIDEBAR_WIDTH - 4)
                  }
                )
              ] })
            },
            chat.id
          );
        }) }),
        /* @__PURE__ */ jsx5("box", { style: { flexGrow: 1 } }),
        /* @__PURE__ */ jsx5(
          "box",
          {
            style: {
              height: 1,
              paddingLeft: 1,
              paddingRight: 1
            },
            children: /* @__PURE__ */ jsx5("text", { children: /* @__PURE__ */ jsx5("span", { style: { fg: theme.colors.system }, children: "Ctrl+N new" }) })
          }
        ),
        /* @__PURE__ */ jsx5(
          "box",
          {
            style: {
              height: 1,
              paddingLeft: 1,
              paddingRight: 1
            },
            children: /* @__PURE__ */ jsx5("text", { children: /* @__PURE__ */ jsx5("span", { style: { fg: theme.colors.system }, children: "Ctrl+J/K \u2195" }) })
          }
        )
      ]
    }
  );
}

// src/ui/Suggestions.tsx
import { Fragment, jsx as jsx6, jsxs as jsxs3 } from "@opentui/react/jsx-runtime";
var MAX_VISIBLE = 5;
function Suggestions({ matches, selectedIndex }) {
  if (matches.length === 0) return null;
  const selected = selectedIndex % matches.length;
  const start = Math.max(0, selected - (MAX_VISIBLE - 1));
  const visible = matches.slice(start, start + MAX_VISIBLE);
  const longestName = visible.reduce(
    (max, m) => Math.max(max, m.name.length),
    0
  );
  const overflow = matches.length - visible.length - start;
  const totalRows = visible.length + (overflow > 0 ? 1 : 0);
  return /* @__PURE__ */ jsxs3(
    "box",
    {
      style: {
        flexDirection: "column",
        flexShrink: 0,
        height: totalRows,
        backgroundColor: theme.colors.suggestionBg
      },
      children: [
        visible.map((cmd, i) => {
          const isSelected = start + i === selected;
          const rowBg = isSelected ? theme.colors.suggestionSelectedBg : theme.colors.suggestionBg;
          const name = cmd.name.padEnd(longestName, " ");
          return /* @__PURE__ */ jsx6(
            "box",
            {
              style: {
                height: 1,
                backgroundColor: rowBg,
                paddingLeft: 1,
                paddingRight: 1
              },
              children: /* @__PURE__ */ jsxs3("text", { children: [
                /* @__PURE__ */ jsx6(
                  "span",
                  {
                    style: {
                      fg: isSelected ? theme.colors.prompt : theme.colors.system,
                      bg: rowBg
                    },
                    children: isSelected ? "\u203A " : "  "
                  }
                ),
                /* @__PURE__ */ jsx6(
                  "span",
                  {
                    style: {
                      fg: isSelected ? theme.colors.user : theme.colors.input,
                      bg: rowBg
                    },
                    children: name
                  }
                ),
                cmd.description ? /* @__PURE__ */ jsxs3(Fragment, { children: [
                  /* @__PURE__ */ jsx6("span", { style: { fg: theme.colors.border, bg: rowBg }, children: "  \u2014 " }),
                  /* @__PURE__ */ jsx6("span", { style: { fg: theme.colors.system, bg: rowBg }, children: cmd.description })
                ] }) : null
              ] })
            },
            cmd.name
          );
        }),
        overflow > 0 ? /* @__PURE__ */ jsx6(
          "box",
          {
            style: {
              height: 1,
              backgroundColor: theme.colors.suggestionBg,
              paddingLeft: 1,
              paddingRight: 1
            },
            children: /* @__PURE__ */ jsx6("text", { children: /* @__PURE__ */ jsx6(
              "span",
              {
                style: {
                  fg: theme.colors.system,
                  bg: theme.colors.suggestionBg
                },
                children: `  +${overflow} more\u2026`
              }
            ) })
          }
        ) : null
      ]
    }
  );
}

// src/ui/App.tsx
import { jsx as jsx7, jsxs as jsxs4 } from "@opentui/react/jsx-runtime";
function filterCommands(commands, prefix) {
  if (!prefix.startsWith("/")) return [];
  const lower = prefix.toLowerCase();
  return commands.filter((c) => c.name.toLowerCase().startsWith(lower));
}
var EMPTY_PENDING = [];
function activeChat(chats, activeId) {
  if (!activeId) return null;
  return chats.find((c) => c.id === activeId) ?? null;
}
function App({ store }) {
  const renderer = useRenderer();
  const snapshot = useSyncExternalStore(store.subscribe, store.getSnapshot);
  const active = activeChat(snapshot.chats, snapshot.activeChatId);
  const activeId = active?.id ?? null;
  const [inputValue, setInputValue] = useState3(active?.inputDraft ?? "");
  const [prefix, setPrefix] = useState3(active?.inputDraft ?? "");
  const [tabIndex, setTabIndex] = useState3(0);
  const inputRef = useRef(inputValue);
  inputRef.current = inputValue;
  const prefixRef = useRef(prefix);
  prefixRef.current = prefix;
  const tabIndexRef = useRef(tabIndex);
  tabIndexRef.current = tabIndex;
  const activeIdRef = useRef(activeId);
  activeIdRef.current = activeId;
  useEffect3(() => {
    const draft = active?.inputDraft ?? "";
    setInputValue(draft);
    setPrefix(draft);
    setTabIndex(0);
  }, [activeId]);
  const commands = snapshot.commands;
  const matches = useMemo(
    () => filterCommands(commands, prefix),
    [commands, prefix]
  );
  const handleInput = useCallback(
    (value) => {
      const currentId = activeIdRef.current;
      if (!currentId) return;
      const { cleaned, attachments } = extractDroppedPaths(value);
      if (attachments.length > 0) {
        for (const a of attachments) {
          store.addPendingAttachment(currentId, a);
        }
      }
      setInputValue(cleaned);
      setPrefix(cleaned);
      setTabIndex(0);
      store.setInputDraft(currentId, cleaned);
    },
    [store]
  );
  const handleSubmit = useCallback(() => {
    const currentId = activeIdRef.current;
    if (!currentId) return;
    const text = inputRef.current.trim();
    if (text === "/new") {
      store.setInputDraft(currentId, "");
      store.newChat();
      setInputValue("");
      setPrefix("");
      setTabIndex(0);
      return;
    }
    const chatSnapshot = store.getSnapshot().chats.find((c) => c.id === currentId);
    const pendings = chatSnapshot?.pendingAttachments ?? EMPTY_PENDING;
    if (text.length === 0 && pendings.length === 0) return;
    (async () => {
      for (const p of pendings) {
        try {
          const content = await attachment(p.path, { name: p.name }).build();
          store.appendUser(currentId, content, { attachmentPath: p.path });
          store.pushUserInput(currentId, content);
        } catch {
          store.appendSystem(currentId, `failed to attach "${p.name}"`);
        }
      }
      if (text.length > 0) {
        const content = { type: "text", text };
        store.appendUser(currentId, content);
        store.pushUserInput(currentId, content);
      }
    })();
    store.clearPendingAttachments(currentId);
    store.setInputDraft(currentId, "");
    setInputValue("");
    setPrefix("");
    setTabIndex(0);
  }, [store]);
  useKeyboard((key) => {
    if (key.ctrl && key.name === "c") {
      process.kill(process.pid, "SIGINT");
      return;
    }
    if (key.ctrl && key.name === "n") {
      store.newChat();
      return;
    }
    if (key.ctrl && key.name === "j") {
      store.cycleActiveChat(1);
      return;
    }
    if (key.ctrl && key.name === "k") {
      store.cycleActiveChat(-1);
      return;
    }
    if (key.ctrl && key.name === "l") {
      const id = activeIdRef.current;
      if (id) store.appendSystem(id, "screen cleared");
      return;
    }
    if (key.ctrl && key.name === "d") {
      renderer?.toggleDebugOverlay();
      return;
    }
    if (key.name === "tab") {
      const current = filterCommands(commands, prefixRef.current);
      if (current.length === 0) return;
      const idx = tabIndexRef.current % current.length;
      const picked = current[idx];
      setInputValue(picked.name);
      setTabIndex((t) => t + 1);
      return;
    }
    if (key.name === "escape") {
      const id = activeIdRef.current;
      if (id) {
        store.clearPendingAttachments(id);
        store.setInputDraft(id, "");
      }
      setInputValue("");
      setPrefix("");
      setTabIndex(0);
      return;
    }
  });
  useEffect3(() => {
    if (!renderer) return;
    const handler = (event) => {
      const currentId = activeIdRef.current;
      if (!currentId) return;
      const raw = new TextDecoder().decode(event.bytes);
      const resolved = resolveDroppedAttachment(raw);
      if (!resolved) return;
      event.preventDefault();
      event.stopPropagation();
      store.addPendingAttachment(currentId, resolved);
    };
    renderer.keyInput.on("paste", handler);
    return () => {
      renderer.keyInput.off("paste", handler);
    };
  }, [renderer, store]);
  const entries = active?.entries ?? [];
  const typing = active?.typing ?? false;
  const pendingAttachments = active?.pendingAttachments ?? EMPTY_PENDING;
  const showSuggestions = matches.length > 0 && prefix.startsWith("/");
  const suggestionRows = showSuggestions ? Math.min(matches.length, 5) + (matches.length > 5 ? 1 : 0) : 0;
  const attachmentRows = pendingAttachments.length > 0 ? 1 : 0;
  const inputContainerHeight = 2 + attachmentRows + suggestionRows + 1;
  const hovered = snapshot.hoveredPreview;
  return /* @__PURE__ */ jsxs4("box", { style: { flexDirection: "row", flexGrow: 1, height: "100%" }, children: [
    hovered ? /* @__PURE__ */ jsxs4(
      "box",
      {
        style: {
          position: "absolute",
          top: 2,
          right: 2,
          flexDirection: "column",
          backgroundColor: theme.colors.suggestionBg,
          border: true,
          borderStyle: "rounded",
          borderColor: theme.colors.border,
          paddingLeft: 1,
          paddingRight: 1,
          zIndex: 100
        },
        children: [
          /* @__PURE__ */ jsx7("box", { style: { height: 1 }, children: /* @__PURE__ */ jsx7("text", { children: /* @__PURE__ */ jsx7("span", { style: { fg: theme.colors.attachment }, children: `\u{1F4CE} ${hovered.name}` }) }) }),
          /* @__PURE__ */ jsx7(
            KittyImage,
            {
              read: hovered.read,
              cacheKey: hovered.cacheKey,
              cols: 40,
              rows: 10
            }
          )
        ]
      }
    ) : null,
    /* @__PURE__ */ jsx7(
      Sidebar,
      {
        chats: snapshot.chats,
        activeChatId: snapshot.activeChatId,
        store
      }
    ),
    /* @__PURE__ */ jsxs4(
      "box",
      {
        style: {
          flexDirection: "column",
          flexGrow: 1,
          height: "100%"
        },
        children: [
          /* @__PURE__ */ jsx7(
            "box",
            {
              style: {
                height: 1,
                paddingLeft: 1,
                paddingRight: 1,
                backgroundColor: theme.colors.border
              },
              children: /* @__PURE__ */ jsxs4("text", { children: [
                /* @__PURE__ */ jsx7("span", { style: { fg: theme.colors.user }, children: ` ${theme.title}${activeId ? ` \xB7 ${activeId}` : ""} ` }),
                /* @__PURE__ */ jsx7("span", { style: { fg: theme.colors.system }, children: " \u2014 Ctrl+N new \xB7 Ctrl+J/K nav \xB7 Ctrl+C exit \xB7 Ctrl+L clear \xB7 Tab complete \xB7 Esc cancel" })
              ] })
            }
          ),
          /* @__PURE__ */ jsx7(
            "scrollbox",
            {
              focused: true,
              style: {
                flexGrow: 1,
                paddingLeft: 1,
                paddingRight: 1,
                stickyScroll: true,
                stickyStart: "bottom",
                contentOptions: {
                  flexDirection: "column",
                  justifyContent: "flex-end",
                  minHeight: "100%"
                },
                scrollbarOptions: {
                  visible: true
                }
              },
              children: activeId ? entries.map((entry) => /* @__PURE__ */ jsx7(
                MessageItem,
                {
                  entry,
                  entries,
                  chatId: activeId,
                  store
                },
                entry.id
              )) : null
            }
          ),
          /* @__PURE__ */ jsx7(
            "box",
            {
              style: {
                height: 1,
                paddingLeft: 1,
                paddingRight: 1
              },
              children: /* @__PURE__ */ jsx7("text", { children: typing ? /* @__PURE__ */ jsx7("span", { style: { fg: theme.colors.typing }, children: "\u25CF agent is typing\u2026" }) : /* @__PURE__ */ jsx7("span", { style: { fg: theme.colors.system }, children: " " }) })
            }
          ),
          /* @__PURE__ */ jsxs4(
            "box",
            {
              style: {
                border: true,
                borderStyle: "rounded",
                borderColor: theme.colors.border,
                flexDirection: "column",
                flexShrink: 0,
                height: inputContainerHeight,
                paddingLeft: 1,
                paddingRight: 1
              },
              children: [
                pendingAttachments.length > 0 ? /* @__PURE__ */ jsx7(Attachments, { pending: pendingAttachments }) : null,
                showSuggestions ? /* @__PURE__ */ jsx7(Suggestions, { matches, selectedIndex: tabIndex }) : null,
                /* @__PURE__ */ jsx7("box", { style: { height: 1, flexShrink: 0 }, children: /* @__PURE__ */ jsx7(
                  "input",
                  {
                    focused: true,
                    placeholder: activeId ? "type a message and press enter\u2026" : "Ctrl+N to start a new chat",
                    value: inputValue,
                    onInput: handleInput,
                    onSubmit: handleSubmit,
                    style: {
                      textColor: theme.colors.input,
                      placeholderColor: theme.colors.system,
                      cursorColor: theme.colors.prompt
                    }
                  }
                ) })
              ]
            }
          )
        ]
      }
    )
  ] });
}
function MountedApp({ store }) {
  useEffect3(() => {
    return () => {
      store.closeInput();
    };
  }, [store]);
  return /* @__PURE__ */ jsx7(App, { store });
}

// src/store.ts
var newId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `id-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
};
function emptyChatState(id) {
  const now = /* @__PURE__ */ new Date();
  return {
    id,
    entries: [],
    typing: false,
    pendingAttachments: [],
    inputDraft: "",
    lastActivityAt: now,
    createdAt: now
  };
}
function sortChats(chats) {
  return [...chats].sort(
    (a, b) => b.lastActivityAt.getTime() - a.lastActivityAt.getTime()
  );
}
function createStore(options) {
  const chats = /* @__PURE__ */ new Map();
  let activeChatId = null;
  let hoveredPreview = null;
  const commands = options?.commands ?? [];
  let nextChatIndex = 1;
  const inputBuffer = [];
  const waiters = [];
  let inputClosed = false;
  let snapshot = {
    chats: [],
    activeChatId: null,
    commands,
    hoveredPreview
  };
  const listeners = /* @__PURE__ */ new Set();
  const commit = () => {
    snapshot = {
      chats: sortChats(Array.from(chats.values())),
      activeChatId,
      commands,
      hoveredPreview
    };
    for (const l of listeners) l();
  };
  const updateChat = (id, updater) => {
    const chat = chats.get(id);
    if (!chat) return false;
    chats.set(id, updater(chat));
    return true;
  };
  const generateChatId = () => {
    while (chats.has(`chat-${nextChatIndex}`)) nextChatIndex += 1;
    const id = `chat-${nextChatIndex}`;
    nextChatIndex += 1;
    return id;
  };
  const ensureChatInternal = (id) => {
    if (chats.has(id)) return false;
    chats.set(id, emptyChatState(id));
    return true;
  };
  const append = (chatId, role, content, opts) => {
    const entryId = newId();
    const entry = {
      id: entryId,
      role,
      content,
      timestamp: /* @__PURE__ */ new Date(),
      replyTo: opts?.replyTo,
      reactions: [],
      attachmentPath: opts?.attachmentPath
    };
    ensureChatInternal(chatId);
    updateChat(chatId, (c) => ({
      ...c,
      entries: [...c.entries, entry],
      lastActivityAt: entry.timestamp
    }));
    commit();
    return entryId;
  };
  return {
    subscribe(listener) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    getSnapshot() {
      return snapshot;
    },
    newChat() {
      const id = generateChatId();
      ensureChatInternal(id);
      activeChatId = id;
      commit();
      return id;
    },
    ensureChat(id) {
      const created = ensureChatInternal(id);
      if (created) {
        if (activeChatId === null) activeChatId = id;
        commit();
      }
    },
    setActiveChat(id) {
      if (!chats.has(id)) return;
      if (activeChatId === id) return;
      activeChatId = id;
      commit();
    },
    cycleActiveChat(delta) {
      const sorted = sortChats(Array.from(chats.values()));
      if (sorted.length === 0) return;
      const idx = sorted.findIndex((c) => c.id === activeChatId);
      const nextIdx = idx < 0 ? 0 : (idx + delta + sorted.length) % sorted.length;
      const next = sorted[nextIdx];
      if (next.id === activeChatId) return;
      activeChatId = next.id;
      commit();
    },
    appendAgent(chatId, content, opts) {
      return append(chatId, "agent", content, { replyTo: opts?.replyTo });
    },
    appendUser(chatId, content, opts) {
      return append(chatId, "user", content, {
        attachmentPath: opts?.attachmentPath
      });
    },
    appendSystem(chatId, text) {
      append(chatId, "system", { type: "text", text });
    },
    setTyping(chatId, value) {
      const chat = chats.get(chatId);
      if (!chat || chat.typing === value) return;
      updateChat(chatId, (c) => ({ ...c, typing: value }));
      commit();
    },
    react(chatId, messageId, emoji) {
      const chat = chats.get(chatId);
      if (!chat) return;
      const idx = chat.entries.findIndex((e) => e.id === messageId);
      if (idx < 0) return;
      const entry = chat.entries[idx];
      updateChat(chatId, (c) => ({
        ...c,
        entries: [
          ...c.entries.slice(0, idx),
          { ...entry, reactions: [...entry.reactions, emoji] },
          ...c.entries.slice(idx + 1)
        ]
      }));
      commit();
    },
    patchEntry(chatId, messageId, patch) {
      const chat = chats.get(chatId);
      if (!chat) return;
      const idx = chat.entries.findIndex((e) => e.id === messageId);
      if (idx < 0) return;
      const entry = chat.entries[idx];
      updateChat(chatId, (c) => ({
        ...c,
        entries: [
          ...c.entries.slice(0, idx),
          { ...entry, ...patch },
          ...c.entries.slice(idx + 1)
        ]
      }));
      commit();
    },
    pushUserInput(chatId, content) {
      if (inputClosed) return;
      const item = { chatId, content };
      const waiter = waiters.shift();
      if (waiter) {
        waiter.resolve({ value: item, done: false });
      } else {
        inputBuffer.push(item);
      }
    },
    nextUserInput() {
      if (inputClosed) {
        return Promise.resolve({
          value: void 0,
          done: true
        });
      }
      const buffered = inputBuffer.shift();
      if (buffered !== void 0) {
        return Promise.resolve({
          value: buffered,
          done: false
        });
      }
      return new Promise((resolve) => {
        waiters.push({ resolve });
      });
    },
    closeInput() {
      if (inputClosed) return;
      inputClosed = true;
      while (waiters.length > 0) {
        const w = waiters.shift();
        w?.resolve({ value: void 0, done: true });
      }
    },
    addPendingAttachment(chatId, att) {
      const chat = chats.get(chatId);
      if (!chat) return;
      if (chat.pendingAttachments.some((a) => a.path === att.path)) return;
      updateChat(chatId, (c) => ({
        ...c,
        pendingAttachments: [...c.pendingAttachments, att]
      }));
      commit();
    },
    removePendingAttachment(chatId, index) {
      const chat = chats.get(chatId);
      if (!chat) return;
      if (index < 0 || index >= chat.pendingAttachments.length) return;
      updateChat(chatId, (c) => ({
        ...c,
        pendingAttachments: [
          ...c.pendingAttachments.slice(0, index),
          ...c.pendingAttachments.slice(index + 1)
        ]
      }));
      commit();
    },
    clearPendingAttachments(chatId) {
      const chat = chats.get(chatId);
      if (!chat || chat.pendingAttachments.length === 0) return;
      updateChat(chatId, (c) => ({ ...c, pendingAttachments: [] }));
      commit();
    },
    setInputDraft(chatId, value) {
      const chat = chats.get(chatId);
      if (!chat || chat.inputDraft === value) return;
      updateChat(chatId, (c) => ({ ...c, inputDraft: value }));
      commit();
    },
    setHoveredPreview(preview) {
      if (hoveredPreview === null && preview === null) {
        return;
      }
      if (hoveredPreview !== null && preview !== null && hoveredPreview.cacheKey === preview.cacheKey) {
        return;
      }
      hoveredPreview = preview;
      commit();
    }
  };
}

// src/client.ts
async function createTuichatClient(options) {
  const store = createStore({ commands: options?.commands });
  store.newChat();
  const renderer = await createCliRenderer({
    exitOnCtrlC: false
  });
  const root = createRoot(renderer);
  root.render(createElement(MountedApp, { store }));
  return {
    store,
    renderer,
    root
  };
}
async function destroyTuichatClient(client) {
  client.store.closeInput();
  try {
    client.root.unmount();
  } catch {
  }
  try {
    client.renderer.destroy?.();
  } catch {
  }
}

// src/provider.ts
var commandSchema = z.object({
  name: z.string().regex(/^\/[A-Za-z0-9_-]+$/, "command must start with /"),
  description: z.string().optional()
});
var tuichat = definePlatform("tuichat", {
  config: z.object({
    commands: z.array(commandSchema).optional()
  }),
  user: {
    resolve: async () => ({ id: "tuichat-user" })
  },
  space: {
    params: z.object({ id: z.string().optional() }),
    resolve: async (ctx) => {
      const client = ctx.client;
      const id = ctx.input.params?.id ?? client.store.newChat();
      client.store.ensureChat(id);
      return { id };
    }
  },
  lifecycle: {
    createClient: async ({ config }) => {
      return await createTuichatClient({ commands: config.commands });
    },
    destroyClient: async ({ client }) => {
      await destroyTuichatClient(client);
    }
  },
  events: {
    async *messages(ctx) {
      const client = ctx.client;
      const { store } = client;
      while (true) {
        const result = await store.nextUserInput();
        if (result.done) return;
        yield {
          id: crypto.randomUUID(),
          content: result.value.content,
          sender: { id: "tuichat-user" },
          space: { id: result.value.chatId },
          timestamp: /* @__PURE__ */ new Date()
        };
      }
    }
  },
  actions: {
    send: async (ctx) => {
      const client = ctx.client;
      client.store.ensureChat(ctx.space.id);
      client.store.appendAgent(ctx.space.id, ctx.content);
    },
    startTyping: async (ctx) => {
      const client = ctx.client;
      client.store.setTyping(ctx.space.id, true);
    },
    stopTyping: async (ctx) => {
      const client = ctx.client;
      client.store.setTyping(ctx.space.id, false);
    },
    reactToMessage: async (ctx) => {
      const client = ctx.client;
      client.store.react(ctx.space.id, ctx.messageId, ctx.reaction);
    },
    replyToMessage: async (ctx) => {
      const client = ctx.client;
      client.store.ensureChat(ctx.space.id);
      client.store.appendAgent(ctx.space.id, ctx.content, {
        replyTo: ctx.messageId
      });
    }
  }
});
export {
  tuichat
};
