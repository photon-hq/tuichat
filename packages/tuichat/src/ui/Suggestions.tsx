import type { CommandDef } from "../store";
import { theme } from "./theme";

interface SuggestionsProps {
  matches: readonly CommandDef[];
  selectedIndex: number;
}

const MAX_VISIBLE = 5;

export function Suggestions({ matches, selectedIndex }: SuggestionsProps) {
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

  return (
    <box
      style={{
        flexDirection: "column",
        flexShrink: 0,
        height: totalRows,
        backgroundColor: theme.colors.suggestionBg,
      }}
    >
      {visible.map((cmd, i) => {
        const isSelected = start + i === selected;
        const rowBg = isSelected
          ? theme.colors.suggestionSelectedBg
          : theme.colors.suggestionBg;
        const name = cmd.name.padEnd(longestName, " ");
        return (
          <box
            key={cmd.name}
            style={{
              height: 1,
              backgroundColor: rowBg,
              paddingLeft: 1,
              paddingRight: 1,
            }}
          >
            <text>
              <span
                style={{
                  fg: isSelected ? theme.colors.prompt : theme.colors.system,
                  bg: rowBg,
                }}
              >
                {isSelected ? "› " : "  "}
              </span>
              <span
                style={{
                  fg: isSelected ? theme.colors.user : theme.colors.input,
                  bg: rowBg,
                }}
              >
                {name}
              </span>
              {cmd.description ? (
                <>
                  <span style={{ fg: theme.colors.border, bg: rowBg }}>
                    {"  — "}
                  </span>
                  <span style={{ fg: theme.colors.system, bg: rowBg }}>
                    {cmd.description}
                  </span>
                </>
              ) : null}
            </text>
          </box>
        );
      })}
      {overflow > 0 ? (
        <box
          style={{
            height: 1,
            backgroundColor: theme.colors.suggestionBg,
            paddingLeft: 1,
            paddingRight: 1,
          }}
        >
          <text>
            <span
              style={{
                fg: theme.colors.system,
                bg: theme.colors.suggestionBg,
              }}
            >
              {`  +${overflow} more…`}
            </span>
          </text>
        </box>
      ) : null}
    </box>
  );
}
