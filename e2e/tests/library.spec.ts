import { test, expect } from "@playwright/test";
import {
  br8kout, click, desktopRow, golden, gridText, offline, sampleScreen, waitForApp,
} from "./screen";

// Desktop (mouse and keyboard) library tests. Touch is in touch.spec.ts.
test.skip(({ hasTouch }) => hasTouch, "desktop only");

// A game is running when the screen shows only the emulated picture (pure
// black and white, no overlay) and it has drawn something.
async function running(page) {
  const text = gridText(await sampleScreen(page));
  return !text.includes("?") && (text.match(/#/g) ?? []).length > 20;
}

async function openLibrary(page, url = "/?shader=clean") {
  await page.goto(url);
  await waitForApp(page);
  await click(page, 10, 10); // switch on
  await page.waitForTimeout(500);
}

async function search(page, text: string) {
  await page.keyboard.type(text, { delay: 120 });
  await page.waitForTimeout(400);
}

test("typing searches, Enter plays a bundled game", async ({ page }) => {
  await offline(page);
  await openLibrary(page);
  await search(page, "glitch");
  await page.keyboard.press("Enter");

  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
});

test("a coming-soon game can't be started", async ({ page }) => {
  await offline(page);
  await openLibrary(page);
  await search(page, "red october");
  await page.keyboard.press("Enter");
  await page.waitForTimeout(1500);

  expect(gridText(await sampleScreen(page))).toContain("?"); // still the drawer
});

test("a downloaded ROM is cached and plays offline after a reload", async ({ page }) => {
  await offline(page, { "/roms/br8kout.ch8": "br8kout.ch8" });
  await openLibrary(page);
  await search(page, "br8kout");
  await page.keyboard.press("Enter");
  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);

  // Now nothing can be downloaded: the ROM must come from IndexedDB.
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await offline(page);
  await page.goto(`/?rom=${br8kout}&shader=clean`);
  await waitForApp(page);

  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
});

test("a ROM that can't be downloaded keeps the library open", async ({ page }) => {
  await offline(page); // Br8kout's download fails
  await openLibrary(page);
  await search(page, "br8kout");
  await page.keyboard.press("Enter");
  await page.waitForTimeout(2000);

  expect(gridText(await sampleScreen(page))).toContain("?");
});

test("settings persist per ROM across a reload", async ({ page }) => {
  await offline(page);
  await page.goto("/?rom=game/glitchGhost&shader=clean");
  await waitForApp(page);
  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);

  await page.keyboard.press("F2");
  await page.waitForTimeout(300);
  await page.keyboard.press("ArrowDown"); // Speed
  await page.waitForTimeout(300);
  await page.keyboard.press("ArrowRight"); // 200 → 300
  await page.waitForTimeout(300);
  await page.keyboard.press("Escape");
  await page.waitForTimeout(300);

  const key = "chippie.overrides/2cdcb3c29a5f013a991db5909ca8e18e27b3c42b";
  expect(await page.evaluate((k) => localStorage.getItem(k), key)).toBe('{"tickrate":300}');

  // Reset to recommended clears it again.
  await page.reload();
  await waitForApp(page);
  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
  await page.keyboard.press("F2");
  await page.waitForTimeout(300);
  for (let i = 0; i < 9; i++) {
    await page.keyboard.press("ArrowDown"); // to "Reset to recommended"
    await page.waitForTimeout(100);
  }
  await page.keyboard.press("Enter");
  await page.waitForTimeout(300);

  expect(await page.evaluate((k) => localStorage.getItem(k), key)).toBeNull();
});

test("load a ROM from this device", async ({ page }) => {
  await offline(page);
  await openLibrary(page);

  const chooser = page.waitForEvent("filechooser");
  await click(page, 100, desktopRow(0)); // "Load a ROM from this device…"
  await (await chooser).setFiles("../third_party/chip8-test-suite/bin/2-ibm-logo.ch8");

  await expect.poll(async () => gridText(await sampleScreen(page)), { intervals: [500] }).toBe(golden("2-ibm-logo"));
});

// Dropping a file isn't tested here: Ebitengine reads drops through
// webkitGetAsEntry(), which returns null for a DataTransfer made in a
// script. Chippie's side (reading the dropped file) is unit-tested in
// app/drop_test.go; a real drag and drop is a manual check.
