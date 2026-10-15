import { readFile, writeFile } from "node:fs/promises";
import { parse, compileReact } from "@codedia/parser";
import {format} from "prettier";
for (const name of ["Tube", "Dialog"]) {
  const s = await readFile(
    new URL(`../src/${name}.aui`, import.meta.url),
    "utf8",
  );
  await writeFile(
    new URL(`../src/${name}.generated.tsx`, import.meta.url),
    `// @ts-nocheck -- Generated dynamic contracts from ${name}.aui; edit the AUI source.\n` +
      await format(compileReact(parse(s)),{parser:"typescript"}),
  );
}
