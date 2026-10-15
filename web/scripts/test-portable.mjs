import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { mkdtemp, readFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import os from "node:os";
import path from "node:path";
import { once } from "node:events";
const root = await mkdtemp(path.join(os.tmpdir(), "mytube-portable-"));
const exe = path.resolve(process.env.MYTUBE_TEST_EXE || "../dist/MyTube.exe");
const origin = "http://127.0.0.1:18897";
let running;
async function start(data, prefix = "") {
  const process = spawn(
    exe,
    [
      "--local",
      "--no-browser",
      "--port",
      "18897",
      "--data",
      data,
      ...(prefix ? ["--public-url", origin + prefix] : []),
    ],
    { windowsHide: true, stdio: ["ignore", "pipe", "pipe"] },
  );
  running = process;
  const url = await new Promise((resolve, reject) => {
    const t = setTimeout(() => reject(Error("Native startup timeout")), 20000);
    let err = "";
    process.stderr.on("data", (d) => (err += d.toString()));
    process.stdout.on("data", (d) => {
      const m = d
        .toString()
        .match(/http:\/\/127\.0\.0\.1:18897[^\s]+token=[a-f0-9]+/);
      if (m) {
        clearTimeout(t);
        resolve(m[0]);
      }
    });
    process.on("error", reject);
    process.on("exit", (code) => {
      if (code) {
        clearTimeout(t);
        reject(Error(`Native server exited ${code}: ${err}`));
      }
    });
  });
  const response = await fetch(url, { redirect: "manual" });
  assert.equal(response.status, 303);
  const cookie = response.headers.get("set-cookie").split(";")[0];
  const me = await (
    await fetch(origin + prefix + "/api/me", { headers: { Cookie: cookie } })
  ).json();
  return { process, cookie, csrf: me.csrf, base: origin + prefix };
}
async function stop(s) {
  const p = s.process;
  if (p.exitCode !== null) return;
  const finished = once(p, "close");
  p.kill();
  await finished;
  running = null;
}
async function api(s, p, method = "GET", body, headers = {}) {
  const r = await fetch(s.base + "/api/" + p, {
    method,
    body:
      body === undefined
        ? undefined
        : Buffer.isBuffer(body)
          ? body
          : JSON.stringify(body),
    headers: {
      Cookie: s.cookie,
      Origin: origin,
      "X-CSRF-Token": s.csrf,
      "Content-Type": "application/json",
      ...headers,
    },
  });
  const data = await r.json();
  assert(r.ok, JSON.stringify(data));
  return data;
}
async function command(args) {
  const p = spawn(exe, args, {
    windowsHide: true,
    stdio: ["ignore", "pipe", "pipe"],
  });
  let output = "";
  p.stderr.on("data", (d) => (output += d.toString()));
  const [code] = await once(p, "close");
  assert.equal(code, 0, output);
}
try {
  const data = path.join(root, "original");
  let s = await start(data);
  await api(s, "channel", "POST", {
    name: "Portable channel",
    handle: "portable-channel",
    visibility: "public",
  });
  const video = await readFile(
    new URL("../../internal/app/testdata/sample.mp4", import.meta.url),
  );
  const u = await api(s, "uploads", "POST", {
    name: "portable.mp4",
    size: video.length,
    visibility: "private",
  });
  await api(s, "uploads/" + u.id, "PATCH", video.subarray(0, 2048), {
    "Upload-Offset": "0",
  });
  await stop(s);
  s = await start(data);
  const checkpoint = await api(s, "uploads/" + u.id);
  assert.equal(checkpoint.offset, 2048);
  await api(s, "uploads/" + u.id, "PATCH", video.subarray(2048), {
    "Upload-Offset": "2048",
  });
  const v = await api(s, "uploads/" + u.id + "/complete", "POST", {});
  const original = await fetch(s.base + `/api/videos/${v.id}/media/original`, {
    headers: { Cookie: s.cookie, Range: "bytes=0-1023" },
  });
  assert.equal(original.status, 206);
  assert.equal((await original.arrayBuffer()).byteLength, 1024);
  const duplicate = spawn(
    exe,
    ["--local", "--no-browser", "--port", "18898", "--data", data],
    { windowsHide: true, stdio: "ignore" },
  );
  const [duplicateCode] = await once(duplicate, "close");
  assert.notEqual(
    duplicateCode,
    0,
    "Duplicate instance acquired the same data",
  );
  await stop(s);
  const zip = path.join(root, "backup.zip");
  await command(["--local", "--data", data, "--file", zip, "backup"]);
  const restored = path.join(root, "restored");
  await command(["--local", "--data", restored, "--file", zip, "restore"]);
  s = await start(restored, "/my-tube");
  const found = await api(s, "videos/" + v.id);
  assert.equal(found.title, "portable");
  const bytes = Buffer.from(
    await (
      await fetch(s.base + `/api/videos/${v.id}/media/original`, {
        headers: { Cookie: s.cookie },
      })
    ).arrayBuffer(),
  );
  assert.equal(
    createHash("sha256").update(bytes).digest("hex"),
    createHash("sha256").update(video).digest("hex"),
  );
  const html = await (await fetch(s.base + "/")).text();
  assert(html.includes('content="/my-tube/"'));
  const asset = html.match(/src="\.\/assets\/([^"\s]+)"/);
  assert(asset);
  assert.equal((await fetch(s.base + "/assets/" + asset[1])).status, 200);
  await stop(s);
  console.log(
    "PASS: native forced restart, upload resume, media persistence, duplicate lock, backup/restore, original integrity and Workspace prefix assets",
  );
} finally {
  if (running && running.exitCode === null) running.kill();
}
