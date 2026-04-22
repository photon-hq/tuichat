import { defineConfig } from "tsup";

export default defineConfig({
  entry: { index: "src/index.ts" },
  format: ["esm"],
  dts: true,
  clean: true,
  outDir: "dist",
  target: "esnext",
  external: ["spectrum-ts", "zod", "@photon-ai/tuichat"],
});
