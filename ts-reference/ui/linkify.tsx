import type { ReactNode } from "react";

const URL_REGEX = /https?:\/\/[^\s<>()\[\]{}]+/g;

const OSC8_START = "\x1b]8;;";
const OSC8_ST = "\x1b\\";

export function wrapOSC8(url: string, text: string): string {
  return `${OSC8_START}${url}${OSC8_ST}${text}${OSC8_START}${OSC8_ST}`;
}

export interface LinkifyColors {
  text: string;
  link: string;
}

export function linkify(text: string, colors: LinkifyColors): ReactNode[] {
  const out: ReactNode[] = [];
  let last = 0;
  let i = 0;
  URL_REGEX.lastIndex = 0;
  let match: RegExpExecArray | null;
  while ((match = URL_REGEX.exec(text)) !== null) {
    if (match.index > last) {
      out.push(
        <span key={`t${i++}`} style={{ fg: colors.text }}>
          {text.slice(last, match.index)}
        </span>
      );
    }
    out.push(
      <span
        key={`l${i++}`}
        style={{ fg: colors.link, attributes: 8 }}
      >
        {wrapOSC8(match[0], match[0])}
      </span>
    );
    last = match.index + match[0].length;
  }
  if (last < text.length) {
    out.push(
      <span key={`t${i++}`} style={{ fg: colors.text }}>
        {text.slice(last)}
      </span>
    );
  }
  return out;
}
