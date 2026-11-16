import * as esbuild from "esbuild";
import { cpSync, readdirSync, rmSync, mkdirSync } from "node:fs";
import { join } from "node:path";

const out = "../server/web/dist";
const watch = process.argv.includes("--watch");

mkdirSync(out, { recursive: true });
for (const f of readdirSync(out)) {
  if (f !== ".gitkeep") rmSync(join(out, f), { recursive: true, force: true });
}
cpSync("static", out, { recursive: true });

const options = {
  entryPoints: ["src/player.ts", "src/admin.ts", "src/display.ts", "src/help.ts"],
  bundle: true,
  outdir: out,
  target: "es2020",
  minify: !watch,
  logLevel: "info",
};

if (watch) {
  const ctx = await esbuild.context(options);
  await ctx.watch();
  console.log("watching (static/ not watched, rerun to copy)");
} else {
  await esbuild.build(options);
}
