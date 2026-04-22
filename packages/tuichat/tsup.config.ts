import { defineConfig } from "tsup";

export default defineConfig({
  entry: {
    index: "src/index.ts",
    "bin/tuichat": "src/bin/tuichat.ts",
  },
  format: ["esm"],
  dts: { entry: "src/index.ts" },
  splitting: true,
  clean: true,
  outDir: "dist",
  target: "esnext",
  external: ["react", "@opentui/core", "@opentui/react"],
});
