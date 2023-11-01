#!/bin/env bun
import fs from "node:fs/promises";
import path from "node:path";
import { Argv } from "./argv";
import { buildPages } from "./build-pages";
import { registerGlobalFunctions } from "./router";

async function main() {
  const argv = new Argv("bldr");

  const build = argv.registerCommand(
    "build",
    {
      src: "string",
      outdir: "string",
    },
    async (args) => {
      const srcFile = path.resolve(args.src);
      const outDir = path.resolve(args.outdir);

      registerGlobalFunctions();

      const mod = await import("file://" + srcFile);

      if (typeof mod.default !== "function") {
        throw new Error(
          `Module ${srcFile} does not export a default function.`
        );
      }

      const App = mod.default;

      const pages = await buildPages(<App />);

      console.log("Saving results to filesystem...");
      await Promise.all(
        pages.map(async (page) => {
          const outfilePath = path.join(outDir, page.route) + ".html";
          const basedir = path.dirname(outfilePath);
          await fs.mkdir(basedir, { recursive: true });
          await Bun.write(outfilePath, page.html);
        })
      );

      console.log("Done.");
    }
  );

  build.setDescription(
    "Generates static HTML files from the given JSX component."
  );
  argv.run();
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
