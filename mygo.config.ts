import { defineConfig } from "mygo-cli";

export default defineConfig({
  name: "SBGen",
  slug: "sbgen",
  identifier: "dev.sivalabs.sbgen",
  // The release workflow sets SBGEN_VERSION from the tag (v1.2.3 → 1.2.3).
  version: process.env.SBGEN_VERSION || "0.0.1",
  copyright: "© 2027 SivaLabs",
  // resources/icon.png is the icon of every platform; macOS 26 prefers the
  // layered Icon Composer icon (outside resources, which ship as they are).
  macos: { icon: "AppIcon.icon" },
  // `mygo dev` runs devCommand and loads devUrl; `mygo build` runs
  // buildCommand and embeds frontendDist into the app.
  devUrl: "http://localhost:5173",
  devCommand: "bun run dev:web",
  buildCommand: "bun run build:web",
  frontendDist: "dist",
  bindings: "src/mygo.ts",
  out: "build",
});
