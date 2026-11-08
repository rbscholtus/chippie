import { test, expect } from "@playwright/test";
import { click, golden, gridText, offline, sampleScreen, waitForApp } from "./screen";

// Software rendering in headless Chromium is slow, so the tests poll for
// the expected picture instead of waiting a fixed time (README §15.3).

test.beforeEach(async ({ page }) => offline(page));

test("the IBM logo test ROM renders exactly", async ({ page }) => {
  await page.goto("/?rom=test/2-ibm-logo&shader=clean");
  await waitForApp(page);

  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toBe(golden("2-ibm-logo"));
});

test("the CHIP-8 logo test ROM renders exactly, by SHA-1", async ({ page }) => {
  await page.goto("/?rom=30f27e5cee5b325fd1681ee98a14de60bfbe951f&shader=clean");
  await waitForApp(page);

  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toBe(golden("1-chip8-logo"));
});

test("switching on opens the library", async ({ page }) => {
  await page.goto("/?shader=clean");
  await waitForApp(page);

  // The loader is gone, so the click reaches the canvas and switches on.
  await click(page, 10, 10);

  // The drawer covers the picture, so it's no longer plain black.
  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toContain("?");
});
