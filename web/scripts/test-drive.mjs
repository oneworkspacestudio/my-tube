import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { mkdtemp, mkdir, writeFile, readFile } from "node:fs/promises";
import { randomBytes, createHash } from "node:crypto";
import path from "node:path";
import os from "node:os";
import { once } from "node:events";
assert(
  process.env.MYTUBE_TEST_DRIVE_EXE,
  "Set MYTUBE_TEST_DRIVE_EXE to a reviewed FileBrowser Quantum executable",
);
const root = await mkdtemp(path.join(os.tmpdir(), "mytube-drive-"));
const files = path.join(root, "files");
await mkdir(files);
const sample = await readFile(
  new URL("../../internal/app/testdata/sample.mp4", import.meta.url),
);
await writeFile(path.join(files, "source.mp4"), sample);
const driveURL = "http://127.0.0.1:18899",
  tubeURL = "http://127.0.0.1:18900";
const pass = randomBytes(32).toString("hex");
let drive, tube;
let driveLog = "";
const conf = {
  integrations: {
    media: {
      ffmpegPath: process.env.MYTUBE_TEST_HELPER_DIR || "",
      convert: {
        videoPreview: {
          mp4: false,
          webm: false,
          mov: false,
          avi: false,
          mkv: false,
          flv: false,
          wmv: false,
        },
        imagePreview: { heic: false, jpeg: false },
      },
    },
  },
  http: { listen: "127.0.0.1", port: 18899, baseURL: "/" },
  server: {
    disableUpdateCheck: true,
    database: { path: path.join(root, "drive.sqlite") },
    cacheDir: path.join(root, "cache"),
    sources: [
      {
        name: "My Drive",
        path: files,
        config: {
          defaultEnabled: true,
          defaultUserScope: "/",
          createUserDir: false,
          defaultPermissions: {
            view: true,
            download: true,
            create: true,
            modify: false,
            delete: false,
          },
        },
      },
    ],
  },
  auth: {
    methods: {
      password: {
        enabled: true,
        signup: true,
        adminUsername: "fixture-admin",
        adminPassword: pass,
      },
    },
  },
  userDefaults: {
    account: {
      permissions: { admin: false, api: true, share: false, realtime: false },
    },
  },
};
await writeFile(path.join(root, "drive.yaml"), JSON.stringify(conf));
await writeFile(
  path.join(root, "tube.json"),
  JSON.stringify({
    listen: "127.0.0.1:18900",
    public_url: tubeURL,
    mode: "local",
    data_dir: path.join(root, "tube"),
    drive_url: driveURL,
  }),
);
async function ready(url, child) {
  for (let i = 0; i < 600; i++) {
    if (child.exitCode !== null)
      throw Error("Fixture exited: " + driveLog.slice(-2000));
    try {
      const r = await fetch(url);
      if (r.ok) return;
    } catch {}
    await new Promise((r) => setTimeout(r, 100));
  }
  throw Error("Fixture startup timeout: " + driveLog.slice(-2000));
}
try {
  drive = spawn(
    process.env.MYTUBE_TEST_DRIVE_EXE,
    ["--config", path.join(root, "drive.yaml"), "--no-input", "run"],
    { cwd: root, windowsHide: true, stdio: ["ignore", "pipe", "pipe"] },
  );
  drive.stdout.on("data", (d) => (driveLog += d.toString()));
  drive.stderr.on("data", (d) => (driveLog += d.toString()));
  await ready(driveURL + "/", drive);
  let response = await fetch(driveURL + "/api/auth/signup", {
    method: "POST",
    headers: { "Content-Type": "application/json", Origin: driveURL },
    body: JSON.stringify({ username: "tube-fixture", password: pass }),
  });
  assert.equal(response.status, 201, await response.text());
  response = await fetch(driveURL + "/api/auth/login?username=tube-fixture", {
    method: "POST",
    headers: { "X-Password": encodeURIComponent(pass), Origin: driveURL },
  });
  assert.equal(response.status, 200);
  const token = (await response.text()).trim().replace(/^"|"$/g, "");
  assert(token.split(".").length === 3);
  response = await fetch(driveURL + "/api/users?username=self", {
    headers: { Authorization: "Bearer " + token },
  });
  assert.equal(response.status, 200);
  const owner = await response.json();
  assert.equal(owner.username, "tube-fixture");
  assert.equal(owner.permissions.admin, false);
  tube = spawn(
    path.resolve(process.env.MYTUBE_TEST_EXE || "../dist/MyTube.exe"),
    ["--config", path.join(root, "tube.json"), "--no-browser"],
    { windowsHide: true, stdio: ["ignore", "pipe", "pipe"] },
  );
  const url = await new Promise((resolve, reject) => {
    const t = setTimeout(
      () => reject(Error("My Tube fixture startup timeout")),
      20000,
    );
    tube.stdout.on("data", (d) => {
      const m = d
        .toString()
        .match(/http:\/\/127\.0\.0\.1:18900\/auth\/local\?token=[a-f0-9]+/);
      if (m) {
        clearTimeout(t);
        resolve(m[0]);
      }
    });
    tube.on("error", reject);
  });
  response = await fetch(url, { redirect: "manual" });
  const cookie = response.headers.get("set-cookie").split(";")[0];
  const me = await (
    await fetch(tubeURL + "/api/me", { headers: { Cookie: cookie } })
  ).json();
  const headers = {
    Cookie: cookie,
    Origin: tubeURL,
    "X-CSRF-Token": me.csrf,
    "Content-Type": "application/json",
    "X-Drive-Token": token,
  };
  async function api(p, method = "POST", data = {}) {
    const r = await fetch(tubeURL + "/api/" + p, {
      method,
      headers,
      body: JSON.stringify(data),
    });
    const text = await r.text();
    assert(r.ok, text);
    return JSON.parse(text);
  }
  await api("channel", "POST", {
    name: "Drive fixture",
    handle: "drive-fixture",
    visibility: "public",
  });
  const listing = await api("drive/list", "POST", {
    source: "My Drive",
    path: "/",
  });
  assert(JSON.stringify(listing).includes("source.mp4"));
  const video = await api("drive/import", "POST", {
    source: "My Drive",
    path: "/source.mp4",
    visibility: "private",
    size: sample.length,
  });
  await api(`videos/${video.id}/drive-export`, "POST", {
    source: "My Drive",
    path: "/exported.mp4",
  });
  const exported = await readFile(path.join(files, "exported.mp4"));
  assert.equal(
    createHash("sha256").update(exported).digest("hex"),
    createHash("sha256").update(sample).digest("hex"),
  );
  assert.equal(
    (await fetch(tubeURL + `/api/videos/${video.id}/media/original`)).status,
    404,
  );
  response = await fetch(tubeURL + `/api/videos/${video.id}/drive-export`, {
    method: "POST",
    headers,
    body: JSON.stringify({ source: "My Drive", path: "/exported.mp4" }),
  });
  assert.equal(response.status, 403);
  assert.deepEqual(await readFile(path.join(files, "source.mp4")), sample);
  console.log(
    "PASS: real Quantum non-admin user, scoped listing, import, export, no overwrite, original preservation and private playback permissions",
  );
} finally {
  for (const p of [tube, drive])
    if (p && p.exitCode === null) {
      const ended = once(p, "close");
      p.kill();
      await ended;
    }
}
