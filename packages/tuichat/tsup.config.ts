import { defineConfig } from "tsup";

export default defineConfig({
  entry: {
    index: "src/index.ts",
  },
  format: ["esm"],
  dts: true,
  splitting: true,
  clean: true,
  outDir: "dist",
  target: "esnext",
  external: ["spectrum-ts", "zod", "react", "@opentui/core", "@opentui/react"],
});
