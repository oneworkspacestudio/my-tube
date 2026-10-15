import { readFile, readdir, writeFile, mkdir } from "node:fs/promises";
import { spawnSync } from "node:child_process";
import path from "node:path";
const root = path.resolve(import.meta.dirname, "..");
const go = process.argv[2] || process.env.MYTUBE_GO || "go";
const listed = spawnSync(go, ["list", "-m", "-json", "all"], {
  cwd: root,
  encoding: "utf8",
  windowsHide: true,
});
if (listed.status !== 0)
  throw new Error("Cannot inventory Go dependencies: " + listed.stderr);
const modules = JSON.parse(
  "[" + listed.stdout.trim().replace(/}\s*{/g, "},{") + "]",
);
const sections = [
  "Third-party license notices for my-tube\nGenerated from the pinned build inputs.\nApplication license: AGPL-3.0-or-later (see LICENSE).\nSee NOTICE for application attribution. FFmpeg helpers are not redistributed.",
];
let count = 0;
async function append(label, dir, license = "See package source") {
  let entries;
  try {
    entries = await readdir(dir);
  } catch {
    return;
  }
  const names = entries.filter((name) =>
    /^(licen[sc]e|copying|notice)(\.[a-z0-9-]+)?$/i.test(name),
  );
  const text = [];
  for (const name of names) {
    try {
      text.push(name + "\n" + (await readFile(path.join(dir, name), "utf8")));
    } catch {}
  }
  sections.push(
    "\n" +
      "=".repeat(72) +
      "\n" +
      label +
      "\nDeclared license: " +
      license +
      "\n" +
      (text.join("\n\n") ||
        "No standalone license file shipped in this package. Consult its upstream source and declared license."),
  );
  count++;
}
for (const mod of modules) {
  if (mod.Main || !mod.Dir) continue;
  await append(mod.Path + " " + mod.Version, mod.Dir);
}
const lock = JSON.parse(
  await readFile(path.join(root, "web/package-lock.json"), "utf8"),
);
for (const [rel, entry] of Object.entries(lock.packages)) {
  if (!rel || entry.dev) continue;
  const dir = path.join(root, "web", rel);
  let pkg;
  try {
    pkg = JSON.parse(await readFile(path.join(dir, "package.json"), "utf8"));
  } catch {
    continue;
  }
  await append(
    pkg.name + " " + pkg.version,
    dir,
    typeof pkg.license === "string"
      ? pkg.license
      : JSON.stringify(pkg.license || "See package source"),
  );
}
await mkdir(path.join(root, "dist"), { recursive: true });
await writeFile(
  path.join(root, "dist/THIRD-PARTY-LICENSES.txt"),
  sections.join("\n\n"),
);
console.log(`Collected license notices for ${count} installed dependencies`);
