import { test, expect, type BrowserContext } from "@playwright/test";
import { spawn, type ChildProcess } from "node:child_process";
import { mkdir, mkdtemp, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import os from "node:os";
test.describe.configure({ mode: "serial" });
const origin = "http://127.0.0.1:18896";
let server: ChildProcess, state: any, videoID: string, fixture: string;
test.beforeAll(async ({ browser }) => {
  fixture = await mkdtemp(path.join(os.tmpdir(), "mytube-browser-"));
  const config = {
    listen: "127.0.0.1:18896",
    public_url: origin,
    data_dir: path.join(fixture, "data"),
    mode: "local",
    public_uploads: true,
    ffmpeg: process.env.MYTUBE_TEST_FFMPEG || "",
    ffprobe: process.env.MYTUBE_TEST_FFPROBE || "",
    transcode: !!process.env.MYTUBE_TEST_FFMPEG,
  };
  await writeFile(path.join(fixture, "config.json"), JSON.stringify(config));
  server = spawn(
    path.resolve(process.env.MYTUBE_TEST_EXE || "../dist/MyTube.exe"),
    ["--config", path.join(fixture, "config.json"), "--no-browser"],
    { windowsHide: true, stdio: ["ignore", "pipe", "pipe"] },
  );
  const login = await new Promise<string>((resolve, reject) => {
    const timer = setTimeout(
      () => reject(Error("Native server did not provide local sign-in")),
      20000,
    );
    let err = "";
    server.stderr!.on("data", (d) => {
      err += d.toString();
    });
    server.stdout!.on("data", (d) => {
      const m = d
        .toString()
        .match(/http:\/\/127\.0\.0\.1:18896\/auth\/local\?token=[a-f0-9]+/);
      if (m) {
        clearTimeout(timer);
        resolve(m[0]);
      }
    });
    server.on("exit", (code) => {
      if (code) reject(Error(`Server exited ${code}: ${err}`));
    });
    server.on("error", reject);
  });
  const context = await browser.newContext();
  const p = await context.newPage();
  await p.goto(login);
  state = await context.storageState();
  await context.close();
});
test.afterAll(() => server?.kill());
test.beforeEach(async ({ context, page }) => {
  await context.addCookies(state.cookies);
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Find your next favourite" }),
  ).toBeVisible();
});
async function api(
  context: BrowserContext,
  p: string,
  method = "GET",
  data?: any,
  headers?: any,
) {
  const me = await (await context.request.get(origin + "/api/me")).json();
  const r = await context.request.fetch(origin + "/api/" + p, {
    method,
    data,
    headers: { Origin: origin, "X-CSRF-Token": me.csrf, ...headers },
  });
  expect(r.ok(), await r.text()).toBeTruthy();
  return r.json();
}
test("create a channel, resumable native upload, direct playback and persistent social actions", async ({
  page,
  context,
}) => {
  await page.getByRole("button", { name: "Your channel", exact: true }).click();
  await page
    .getByLabel("Channel name", { exact: true })
    .fill("Studio Field Notes");
  await page
    .getByLabel("Handle (3–40 letters, numbers, _ or -)")
    .fill("field-notes");
  await page
    .getByLabel("About your channel")
    .fill("Small observations, shared with curiosity.");
  await page.getByRole("button", { name: "Save channel", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Studio Field Notes" }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Create", exact: true })
    .first()
    .click();
  await expect(
    page.getByRole("heading", { name: "Upload your videos" }),
  ).toBeVisible();
  await page
    .getByLabel("Select videos")
    .setInputFiles(path.resolve("../internal/app/testdata/sample.mp4"));
  await expect(
    page.getByText("Uploaded — processing", { exact: true }),
  ).toBeVisible();
  const videos = await api(context, "videos?view=studio");
  expect(videos.length).toBe(1);
  videoID = videos[0].id;
  await page
    .getByRole("button", { name: "Creator Studio", exact: true })
    .first()
    .click();
  await expect(
    page.getByRole("button", { name: "sample", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Edit", exact: true }).click();
  await page
    .getByLabel("Title", { exact: true })
    .fill("Finding a little colour in the everyday");
  await page
    .getByLabel("Description", { exact: true })
    .fill(
      "A short original test sequence.\nMade in our own corner of the internet.",
    );
  await page
    .getByLabel("Tags, separated by commas")
    .fill("colour, creativity, field notes");
  await page.getByLabel("Category", { exact: true }).selectOption("Film");
  await page.getByLabel("Visibility", { exact: true }).selectOption("public");
  await page.getByRole("button", { name: "Save details", exact: true }).click();
  await expect(
    page.getByRole("button", {
      name: "Finding a little colour in the everyday",
      exact: true,
    }),
  ).toBeVisible();
  await page
    .getByRole("button", {
      name: "Finding a little colour in the everyday",
      exact: true,
    })
    .click();
  await expect(
    page.getByRole("heading", {
      name: "Finding a little colour in the everyday",
    }),
  ).toBeVisible();
  await page.waitForFunction(() => {
    const v = document.querySelector("video");
    return v && v.readyState >= 1;
  });
  const dimensions = await page
    .locator("video")
    .evaluate((v: HTMLVideoElement) => [v.videoWidth, v.videoHeight]);
  expect(dimensions).toEqual([640, 360]);
  await page.locator("video").evaluate((v: HTMLVideoElement) => v.play());
  await page.waitForFunction(
    () => document.querySelector("video")!.currentTime > 0.2,
  );
  await page.locator("video").evaluate((v: HTMLVideoElement) => v.pause());
  await page
    .getByLabel("Write a comment")
    .fill("A good place to start <script>safe text</script>");
  await page.getByRole("button", { name: "Post", exact: true }).click();
  await expect(
    page.getByText("A good place to start <script>safe text</script>", {
      exact: true,
    }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page
    .getByRole("button", { name: "Watch Later", exact: true })
    .last()
    .click();
  await page.reload();
  await expect(
    page.getByText("A good place to start <script>safe text</script>", {
      exact: true,
    }),
  ).toBeVisible();
  expect((await api(context, "videos/" + videoID)).later).toBe(true);
});
test("optional HLS rendition decodes in Chrome",async({page,context})=>{test.skip(!process.env.MYTUBE_TEST_FFMPEG,'Set helper paths for the real HLS browser check');await expect.poll(async()=> (await api(context,'videos/'+videoID)).hls,{timeout:30000}).toBe(true);await page.goto('/#watch/'+videoID);await page.waitForFunction(()=>{const video=document.querySelector('video');return video&&video.readyState>=2});await expect(page.getByLabel('Playback quality')).toContainText('360p');await page.locator('video').evaluate((v:HTMLVideoElement)=>v.play());await page.waitForFunction(()=>document.querySelector('video')!.currentTime>.2);await page.locator('video').evaluate((v:HTMLVideoElement)=>v.pause())});

test("playlists, indexed search and settings", async ({ page, context }) => {
  await page.getByRole("button", { name: "Playlists", exact: true }).click();
  await page.getByRole("button", { name: "New playlist", exact: true }).click();
  await page.getByLabel("Title", { exact: true }).fill("Weekend discoveries");
  await page
    .getByRole("button", { name: "Create playlist", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Weekend discoveries", exact: true }),
  ).toBeVisible();
  const playlists = await api(context, "playlists");
  await api(context, `playlists/${playlists[0].id}/videos`, "POST", {
    video_id: videoID,
  });
  await page
    .getByRole("button", { name: "Weekend discoveries", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Finding a little colour in the everyday",
      exact: true,
    }),
  ).toBeVisible();
  await page.getByLabel("Search videos", { exact: true }).fill("colour");
  await page.getByLabel("Search videos", { exact: true }).press("Enter");
  await expect(
    page.getByRole("heading", { name: "Results for “colour”" }),
  ).toBeVisible();
  await expect(page.locator(".video-card")).toHaveCount(1);
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Account & privacy" }),
  ).toBeVisible();
  await page.getByLabel("Personalise recommendations").uncheck();
  await expect
    .poll(async () => (await api(context, "me")).personalize)
    .toBe(false);
});
test("private originals and derivatives stay hidden to anonymous browsers", async ({
  browser,
  context,
}) => {
  await api(context, "videos/" + videoID, "PUT", {
    title: "Finding a little colour in the everyday",
    description: "Testing permission boundaries",
    tags: "colour",
    category: "Film",
    visibility: "private",
    chapters: [],
  });
  const anon = await browser.newContext();
  for (const f of [
    "",
    "/media/original",
    "/media/poster.jpg",
    "/media/master.m3u8",
    "/media/q360-00000.ts",
    "/media/captions.vtt",
  ]) {
    expect(
      (await anon.request.get(origin + "/api/videos/" + videoID + f)).status(),
    ).toBe(404);
  }
  await anon.close();
  await api(context, "videos/" + videoID, "PUT", {
    title: "Finding a little colour in the everyday",
    description: "Testing permission boundaries",
    tags: "colour",
    category: "Film",
    visibility: "public",
    chapters: [],
  });
});
test("desktop, mobile and dark screenshots; no horizontal clipping", async ({
  page,
  context,
}) => {
  await mkdir("../docs/screenshots", { recursive: true });
  await page.goto("/");
  await expect(page.locator(".video-card")).toHaveCount(1);
  await page.screenshot({
    path: "../docs/screenshots/library-desktop.png",
    fullPage: true,
  });
  await page.goto("/#watch/" + videoID);
  await expect(page.locator("video")).toBeVisible();
  await page.screenshot({
    path: "../docs/screenshots/watch-desktop.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.locator(".bottom-nav")).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: "../docs/screenshots/watch-mobile.png",
    fullPage: true,
  });
  await page.goto("/");
  await page.screenshot({
    path: "../docs/screenshots/library-mobile.png",
    fullPage: true,
  });
  await page.evaluate(() => {
    document.documentElement.dataset.theme = "dark";
  });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.screenshot({
    path: "../docs/screenshots/library-dark.png",
    fullPage: true,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await api(context, "history", "DELETE");
});

test('first-run wizard writes a managed configuration without storing a client secret',async({browser})=>{const root=await mkdtemp(path.join(os.tmpdir(),'mytube-setup-'));const config=path.join(root,'config.json');const child=spawn(path.resolve(process.env.MYTUBE_TEST_EXE||'../dist/MyTube.exe'),['--no-browser','--config',config,'--data',path.join(root,'data'),'--port','18901'],{windowsHide:true,stdio:['ignore','pipe','pipe']});const context=await browser.newContext();try{const url=await new Promise<string>((resolve,reject)=>{const timeout=setTimeout(()=>reject(Error('Setup startup timeout')),15000);child.stdout!.on('data',d=>{const m=d.toString().match(/http:\/\/127\.0\.0\.1:18901\/auth\/local\?token=[a-f0-9]+/);if(m){clearTimeout(timeout);resolve(m[0])}});child.on('error',reject)});const p=await context.newPage();await p.goto(url);await expect(p.getByRole('heading',{name:'Make My Tube yours'})).toBeVisible();await p.getByRole('button',{name:'Set up My Tube',exact:true}).click();await p.getByLabel('MyAccounts issuer URL').fill('http://localhost:12345');await p.getByLabel('MyAccounts client ID').fill('setup-fixture');await p.getByLabel('Administrator MyAccounts subject (exact sub claim)').fill('fixture-owner');await p.getByRole('button',{name:'Save setup',exact:true}).click();await expect(p.getByText('Setup saved. Restart MyTube.exe to begin using your configuration.')).toBeVisible();const saved=JSON.parse(await readFile(config,'utf8'));expect(saved.mode).toBe('managed');expect(saved.admin_subjects).toEqual(['fixture-owner']);expect(saved.oidc_client_secret).toBeUndefined()}finally{child.kill();await context.close()}});
