// Playwright UI smoke + screenshot tests for the dashboard. The /api responses
// are mocked, so these run without an Android SDK or emulator.

import { test, expect } from "@playwright/test";

const inventory = {
  backend: "android CLI (mock)",
  entries: [
    { name: "medium_phone", avd: "medium_phone", serial: "emulator-5554", running: true, isEmu: true },
    { name: "Pixel_Tablet", avd: "Pixel_Tablet", running: false, isEmu: true },
  ],
};
const doctorChecks = [
  { Name: "adb", OK: true, Detail: "/usr/bin/adb" },
  { Name: "android CLI", OK: true, Detail: "/usr/local/bin/android" },
];
const meta = { mine: ["com.example.app"], processes: ["com.android.systemui", "com.example.app"] };

test.beforeEach(async ({ page }) => {
  await page.route("**/api/inventory", (r) => r.fulfill({ json: inventory }));
  await page.route("**/api/doctor", (r) => r.fulfill({ json: doctorChecks }));
  await page.route("**/api/meta**", (r) => r.fulfill({ json: meta }));
  await page.route("**/api/logcat**", (r) => r.fulfill({ status: 200, body: "" }));
});

test("dashboard renders with modals hidden", async ({ page }) => {
  await page.goto("/");

  await expect(page.locator("#backend")).toContainText("mock");
  await expect(page.locator(".device", { hasText: "medium_phone" })).toBeVisible();
  await expect(page.locator(".device", { hasText: "Pixel_Tablet" })).toBeVisible();

  // Regression guard: modals must stay hidden on load (a CSS cascade bug once
  // left them displayed over the page).
  await expect(page.locator("#doctor-modal")).toBeHidden();
  await expect(page.locator("#shot-modal")).toBeHidden();

  await page.screenshot({ path: "test-results/dashboard.png" });
});

test("doctor modal opens and closes", async ({ page }) => {
  await page.goto("/");
  await page.locator("#doctor-btn").click();
  await expect(page.locator("#doctor-modal")).toBeVisible();
  await expect(page.locator("#doctor-list")).toContainText("adb");
  await page.locator("#doctor-close").click();
  await expect(page.locator("#doctor-modal")).toBeHidden();
});

test("filter autocomplete suggests keys and package:mine", async ({ page }) => {
  await page.goto("/");
  const filter = page.locator("#filter");

  await filter.click();
  await expect(page.locator("#ac")).toBeVisible();
  await expect(page.locator("#ac li", { hasText: "package:" })).toBeVisible();

  await filter.fill("package:");
  await expect(page.locator("#ac li", { hasText: "package:mine" })).toBeVisible();

  await page.screenshot({ path: "test-results/autocomplete.png" });
});
