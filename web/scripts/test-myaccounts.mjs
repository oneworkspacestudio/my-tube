import assert from "node:assert/strict";
import { randomBytes, randomUUID } from "node:crypto";
import { spawn } from "node:child_process";
import { chromium } from "@playwright/test";
const issuer = process.env.MYTUBE_TEST_ISSUER,
  origin = process.env.MYTUBE_TEST_ORIGIN,
  password = process.env.MYTUBE_TEST_PASSWORD;
assert(
  issuer?.startsWith("http://localhost:") &&
    origin?.startsWith("http://127.0.0.1:") &&
    password,
);
const browser = await chromium.launch({
  executablePath:
    process.env.MYTUBE_CHROME ||
    "C:/Program Files/Google/Chrome/Application/chrome.exe",
});
let mytube;
try {
  const admin = await browser.newContext();
  async function provider(path, body) {
    const r = await admin.request.fetch(issuer + path, {
      method: body ? "POST" : "GET",
      data: body,
      headers: { Origin: issuer },
    });
    assert.equal(r.status(), 200);
    const result = await r.json();
    assert.equal(
      result.status,
      "ok",
      "Provider operation failed: " + path.split("?")[0],
    );
    return result.data;
  }
  const base = {
    organization: "built-in",
    username: "admin@example.com",
    password,
    application: "app-built-in",
    signinMethod: "Password",
    type: "login",
  };
  await provider("/api/login", base);
  const firstPassword = randomBytes(32).toString("hex"),
    otherPassword = randomBytes(32).toString("hex");
  for (const [name, pass] of [
    ["mytube-primary", firstPassword],
    ["mytube-isolation", otherPassword],
  ])
    await provider("/api/add-user", {
      owner: "myaccounts",
      name,
      displayName: name,
      email: name + "@example.test",
      emailVerified: false,
      type: "normal-user",
      isAdmin: false,
      passwordType: "plain",
      password: pass,
      signupApplication: "myaccounts",
      createdTime: new Date().toISOString(),
      properties: {},
    });
  const relyingUser = {
    ...base,
    organization: "myaccounts",
    username: "mytube-primary",
    password: firstPassword,
  };
  const template = await provider("/api/get-application?id=admin/myaccounts");
  const clientId = "mytube-test-" + randomBytes(8).toString("hex"),
    clientSecret = randomBytes(32).toString("hex");
  await provider("/api/add-application", {
    ...template,
    name: clientId,
    displayName: "My Tube disposable test",
    clientId,
    clientSecret,
    redirectUris: [origin + "/auth/callback"],
    grantTypes: ["authorization_code"],
    enableAutoSignin: false,
    enableSigninSession: true,
    enableSignUp: false,
    promptPage: [],
    providers: [],
    scopes: [],
  });
  mytube = spawn(
    process.env.MYTUBE_TEST_EXE,
    [
      "--no-browser",
      "--port",
      new URL(origin).port,
      "--data",
      process.env.MYTUBE_TEST_DATA,
    ],
    {
      windowsHide: true,
      env: {
        ...process.env,
        MYTUBE_OIDC_ISSUER: issuer,
        MYTUBE_OIDC_CLIENT_ID: clientId,
        MYTUBE_OIDC_CLIENT_SECRET: clientSecret,
      },
      stdio: "ignore",
    },
  );
  for (let i = 0; i < 100; i++) {
    try {
      if ((await fetch(origin + "/healthz")).ok) break;
    } catch {}
    await new Promise((r) => setTimeout(r, 100));
  }
  async function signIn(context, credentials = relyingUser) {
    const start = await context.request.get(origin + "/auth/login", {
      maxRedirects: 0,
    });
    assert.equal(start.status(), 302);
    const q = new URL(start.headers().location).searchParams;
    assert.equal(q.get("code_challenge_method"), "S256");
    const request = new URLSearchParams({
      clientId,
      responseType: "code",
      redirectUri: origin + "/auth/callback",
      scope: q.get("scope"),
      state: q.get("state"),
      nonce: q.get("nonce"),
      code_challenge: q.get("code_challenge"),
      code_challenge_method: "S256",
    });
    const code = await provider("/api/login?" + request, {
      ...credentials,
      type: "code",
      application: clientId,
    });
    const page = await context.newPage();
    await page.goto(
      origin +
        "/auth/callback?" +
        new URLSearchParams({ code, state: q.get("state") }),
    );
    const me = await (await context.request.get(origin + "/api/me")).json();
    assert(me.user?.id, "My Tube rejected the real MyAccounts callback");
    await page.close();
    return me;
  }
  const me = await signIn(admin);
  const headers = { Origin: origin, "X-CSRF-Token": me.csrf };
  let response = await admin.request.post(origin + "/api/channel", {
    data: {
      name: "Provider channel",
      handle: "provider-channel",
      visibility: "public",
    },
    headers,
  });
  assert.equal(response.status(), 200);
  const sample = await (
    await import("node:fs/promises")
  ).readFile(
    new URL("../../internal/app/testdata/sample.mp4", import.meta.url),
  );
  response = await admin.request.post(origin + "/api/uploads", {
    data: {
      name: "oidc.mp4",
      size: sample.length,
      title: "Real provider video",
      visibility: "private",
    },
    headers,
  });
  assert.equal(response.status(), 201);
  const upload = await response.json();
  response = await admin.request.patch(origin + "/api/uploads/" + upload.id, {
    data: sample,
    headers: { ...headers, "Upload-Offset": "0" },
  });
  assert.equal(response.status(), 200);
  response = await admin.request.post(
    origin + "/api/uploads/" + upload.id + "/complete",
    { data: {}, headers },
  );
  assert.equal(response.status(), 201);
  const n = await response.json();
  const second = await browser.newContext();
  const same = await signIn(second);
  assert.equal(
    me.user.id,
    same.user.id,
    "Subject mapping changed between sessions",
  );
  const other = await browser.newContext();
  const otherMe = await signIn(other, {
    ...relyingUser,
    username: "mytube-isolation",
    password: otherPassword,
  });
  assert.notEqual(
    otherMe.user.id,
    me.user.id,
    "Different provider subjects were merged",
  );
  assert.equal(
    (await other.request.get(origin + "/api/videos/" + n.id)).status(),
    404,
    "Cross-user video access was allowed",
  );
  const unauthenticated = await browser.newContext();
  assert.equal(
    (
      await unauthenticated.request.get(origin + "/api/videos/" + n.id)
    ).status(),
    404,
  );
  assert.equal(
    (
      await admin.request.post(origin + "/api/logout", {
        headers: { Origin: origin, "X-CSRF-Token": me.csrf },
      })
    ).status(),
    200,
  );
  assert.equal(
    (await admin.request.get(origin + "/api/videos/" + n.id)).status(),
    404,
  );
  assert.equal(
    (await second.request.get(origin + "/api/videos/" + n.id)).status(),
    200,
  );
  console.log(
    "PASS: real provider callback, stable identity, distinct-subject video isolation, unauthenticated refusal and local logout",
  );
} finally {
  mytube?.kill();
  await browser.close();
}
