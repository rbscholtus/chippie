import { test, expect } from "@playwright/test";
import { golden, gridText, sampleScreen, waitForApp } from "./screen";

// Software rendering in headless Chromium is slow, so the tests poll for
// the expected picture instead of waiting a fixed time (README §15.3).

test("the IBM logo test ROM renders exactly", async ({ page }) => {
  await page.goto("/?rom=test/2-ibm-logo&shader=clean");
  await waitForApp(page);

  const want = golden("2-ibm-logo");
  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toBe(want);
});

test("the CHIP-8 logo test ROM renders exactly", async ({ page }) => {
  await page.goto("/?rom=test/1-chip8-logo&shader=clean");
  await waitForApp(page);

  const want = golden("1-chip8-logo");
  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toBe(want);
});

test("switch on, pick a bundled game, and it runs", async ({ page }) => {
  await page.goto("/?shader=clean");
  await waitForApp(page);

  // The loader is gone, so the click reaches the canvas and switches on.
  await page.mouse.click(10, 10);
  // Glitch Ghost is the first entry; Enter plays it.
  await page.waitForTimeout(500);
  await page.keyboard.press("Enter");

  // Running: the screen shows only the emulated picture (pure black and
  // white, no overlay) and the game has drawn something.
  await expect
    .poll(async () => {
      const text = gridText(await sampleScreen(page));
      return !text.includes("?") && (text.match(/#/g) ?? []).length > 100;
    }, { intervals: [500] })
    .toBe(true);
});

test("a coming-soon game can't be started", async ({ page }) => {
  await page.goto("/?shader=clean");
  await waitForApp(page);

  await page.keyboard.press("Space"); // switch on
  await page.waitForTimeout(500);
  await page.keyboard.press("ArrowDown");
  await page.waitForTimeout(300);
  await page.keyboard.press("ArrowDown"); // Turnover '77 (SUPER-CHIP)
  await page.waitForTimeout(300);
  await page.keyboard.press("Enter");
  await page.waitForTimeout(2000);

  // Still in the picker: its panel and text cover the screen, so the
  // picture isn't pure black and white.
  expect(gridText(await sampleScreen(page))).toContain("?");
});
