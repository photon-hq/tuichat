import type { ReactNode } from "react";

const URL_REGEX = /https?:\/\/[^\s<>()\[\]{}]+/g;

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
      <a
        key={`l${i++}`}
        href={match[0]}
        style={{ fg: colors.link, attributes: 4 }}
      >
        {match[0]}
      </a>
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
