import { test, expect } from "@playwright/test";
import {
  br8kout, click, desktopRow, golden, gridText, offline, press, running, sampleScreen, waitForApp, waitForArt,
} from "./screen";

// Desktop (mouse and keyboard) library tests. Touch is in touch.spec.ts.
test.skip(({ hasTouch }) => hasTouch, "desktop only");

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
  await press(page, "Enter");

  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
});

test("a coming-soon game can't be started", async ({ page }) => {
  await offline(page);
  await openLibrary(page);
  await search(page, "clock program");
  await press(page, "Enter");
  await page.waitForTimeout(1500);

  expect(gridText(await sampleScreen(page))).toContain("?"); // still the drawer
});

test("a downloaded ROM is cached and plays offline after a reload", async ({ page }) => {
  await offline(page, { "/roms/br8kout.ch8": "br8kout.ch8" });
  await openLibrary(page);
  await search(page, "br8kout");
  await press(page, "Enter");
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
  await press(page, "Enter");
  await page.waitForTimeout(2000);

  expect(gridText(await sampleScreen(page))).toContain("?");
});

test("settings persist per ROM across a reload", async ({ page }) => {
  await offline(page);
  await page.goto("/?rom=game/glitchGhost&shader=clean");
  await waitForApp(page);
  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
  await waitForArt(page); // decoding the art blocks the app: let it finish first

  // Settings rows: TV (with more than one skin), Platform, Speed, …
  await press(page, "F2");
  for (let i = 0; i < 2; i++) {
    await press(page, "ArrowDown"); // to Speed
  }
  await press(page, "ArrowRight"); // 200 → 300
  await press(page, "Escape");

  const key = "chippie.overrides/2cdcb3c29a5f013a991db5909ca8e18e27b3c42b";
  const stored = () => page.evaluate((k) => localStorage.getItem(k), key);
  await expect.poll(stored, { intervals: [300] }).toBe('{"tickrate":300}');

  // Reset to recommended clears it again.
  await page.reload();
  await waitForApp(page);
  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
  await waitForArt(page);
  await press(page, "F2");
  for (let i = 0; i < 10; i++) {
    await press(page, "ArrowDown"); // to "Reset to recommended"
  }
  await press(page, "Enter");

  await expect.poll(stored, { intervals: [300] }).toBeNull();
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
// internal/app/drop_test.go; a real drag and drop is a manual check.
