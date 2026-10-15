import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./tests",
  workers: 1,
  timeout: 60000,
  expect: { timeout: 15000 },
  use: {
    baseURL: "http://127.0.0.1:18896",
    viewport: { width: 1440, height: 1000 },
    launchOptions: {
      executablePath:
        process.env.MYTUBE_CHROME ||
        "C:/Program Files/Google/Chrome/Application/chrome.exe",
    },
  },
  reporter: [
    ["list"],
    ["json", { outputFile: "../test-results/browser.json" }],
  ],
  outputDir: "../test-results/playwright",
});
