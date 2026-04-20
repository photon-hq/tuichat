import { useEffect, useState } from "react";
import {
  buildPlaceholderRows,
  ensureImageTransmitted,
  imageIdToHexColor,
} from "../kitty";
import { theme } from "./theme";

interface KittyImageProps {
  read: () => Promise<Buffer>;
  cacheKey: string;
  cols?: number;
  rows?: number;
}

export function KittyImage({
  read,
  cacheKey,
  cols = 40,
  rows = 10,
}: KittyImageProps) {
  const [imageId, setImageId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    ensureImageTransmitted(cacheKey, read, cols, rows)
      .then((id) => {
        if (!cancelled) setImageId(id);
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [cacheKey, read, cols, rows]);

  if (error) {
    return (
      <text>
        <span style={{ fg: theme.colors.system }}>
          {`[image preview failed: ${error}]`}
        </span>
      </text>
    );
  }

  if (imageId === null) {
    return (
      <text>
        <span style={{ fg: theme.colors.system }}>{"[loading image…]"}</span>
      </text>
    );
  }

  const fg = imageIdToHexColor(imageId);
  const lines = buildPlaceholderRows(cols, rows);

  return (
    <box style={{ flexDirection: "column", height: lines.length, width: cols }}>
      {lines.map((line, i) => (
        <text key={i}>
          <span style={{ fg }}>{line}</span>
        </text>
      ))}
    </box>
  );
}
