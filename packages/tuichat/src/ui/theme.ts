import type { Role } from "../store";

export const theme = {
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
    prompt: "#7dd3fc",
  },
  prefix: {
    user: "you",
    agent: "agent",
    system: "sys",
  } satisfies Record<Role, string>,
};

export const formatTime = (d: Date): string => {
  const hh = d.getHours().toString().padStart(2, "0");
  const mm = d.getMinutes().toString().padStart(2, "0");
  const ss = d.getSeconds().toString().padStart(2, "0");
  return `${hh}:${mm}:${ss}`;
};

export const roleColor = (role: Role): string => theme.colors[role];

export const formatBytes = (bytes: number): string => {
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}kB`;
  return `${(bytes / 1024 / 1024).toFixed(1)}MB`;
};
