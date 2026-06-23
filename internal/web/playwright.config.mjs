import { defineConfig } from "@playwright/test";

// UI smoke tests for the dashboard. They serve the static assets locally and
// mock the /api responses, so they need no Android SDK or emulator.
export default defineConfig({
  testDir: "./jstest",
  testMatch: /.*\.spec\.mjs/,
  reporter: "line",
  use: {
    baseURL: "http://localhost:7099",
    screenshot: "only-on-failure",
  },
  webServer: {
    command: "node jstest/serve.mjs 7099",
    port: 7099,
    reuseExistingServer: !process.env.CI,
  },
});
