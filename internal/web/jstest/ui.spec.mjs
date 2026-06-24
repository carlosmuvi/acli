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

test("selecting a device collapses the panel; the toggle restores it", async ({ page }) => {
  await page.goto("/");
  const main = page.locator("main");
  const panel = page.locator("#devices-panel");
  const toggle = page.locator("#toggle-devices");

  // Starts expanded.
  await expect(panel).toBeVisible();
  await expect(main).not.toHaveClass(/devices-collapsed/);

  // Opening logcat on a running device collapses the list.
  await page
    .locator(".device", { hasText: "medium_phone" })
    .getByRole("button", { name: "Logcat" })
    .click();
  await expect(main).toHaveClass(/devices-collapsed/);
  await expect(panel).toBeHidden();
  await expect(toggle).toHaveAttribute("aria-pressed", "true");

  // The header toggle brings it back, and toggles off again.
  await toggle.click();
  await expect(panel).toBeVisible();
  await expect(toggle).toHaveAttribute("aria-pressed", "false");
  await toggle.click();
  await expect(panel).toBeHidden();
});

test("a lone running device auto-opens logcat and collapses the panel", async ({ page }) => {
  // Single running entry → unambiguous target, so open it without a click.
  await page.route("**/api/inventory", (r) =>
    r.fulfill({ json: { backend: "mock", entries: [
      { name: "solo_phone", avd: "solo_phone", serial: "emulator-5554", running: true, isEmu: true },
    ] } }),
  );
  await page.goto("/");

  await expect(page.locator("#logcat-title")).toHaveText("Logcat ▸ solo_phone");
  await expect(page.locator("main")).toHaveClass(/devices-collapsed/);
  await expect(page.locator("#devices-panel")).toBeHidden();
});

test("doctor modal opens and closes", async ({ page }) => {
  await page.goto("/");
  await page.locator("#doctor-btn").click();
  await expect(page.locator("#doctor-modal")).toBeVisible();
  await expect(page.locator("#doctor-list")).toContainText("adb");
  await page.locator("#doctor-close").click();
  await expect(page.locator("#doctor-modal")).toBeHidden();
});

test("long logcat lines wrap only in the message column", async ({ page }) => {
  // Regression guard: the fixed time/level/tag columns are flex items and once
  // got squeezed by a long message until they broke mid-token (e.g. the
  // timestamp split into "18:42:01" / ".141"). Render a line the way lineNode
  // does and assert only .msg wraps.
  await page.setViewportSize({ width: 760, height: 600 });
  await page.goto("/");

  await page.evaluate(() => {
    const longMsg =
      "at com.google.android.chimera.IntentOperation.onHandleIntent(" +
      ":com.google.android.gms@262233035@26.22.33 (260400-932714197):1) " +
      "and a great deal more text to force the message onto several lines";
    const line = document.createElement("div");
    line.className = "line lvl-E";
    line.innerHTML =
      '<span class="t">18:42:01.141</span>' +
      '<span class="lv">E</span>' +
      '<span class="tag">constellation:</span>' +
      `<span class="msg">${longMsg}</span>`;
    document.querySelector("#log").append(line);
  });

  // Each column is a (blockified) flex item, so height is the tell: a column
  // kept on one line is one line tall, a wrapped one is taller. The single-char
  // level span is the one-line reference. Without the fix this fails two ways:
  // align-items:stretch makes every column as tall as the wrapped message (so
  // .msg is no longer the tallest), and the unconstrained .t/.tag wrap too.
  const h = await page.evaluate(() => {
    const q = (sel) => document.querySelector("#log .line " + sel).getBoundingClientRect().height;
    return { t: q(".t"), lv: q(".lv"), tag: q(".tag"), msg: q(".msg") };
  });
  expect(h.t).toBeCloseTo(h.lv, 0); // time stays on one line
  expect(h.tag).toBeCloseTo(h.lv, 0); // tag stays on one line
  expect(h.msg).toBeGreaterThan(h.lv * 1.5); // only the message wraps

  // And the timestamp text must survive intact (not broken across boxes).
  await expect(page.locator("#log .line .t")).toHaveText("18:42:01.141");
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
