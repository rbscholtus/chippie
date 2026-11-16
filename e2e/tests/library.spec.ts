import { readFileSync } from "node:fs";
import { test, expect } from "@playwright/test";
import {
  br8kout, click, desktopRow, golden, gridText, offline, running, sampleScreen, waitForApp,
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
  await page.keyboard.press("Enter");

  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
});

test("Chippie's own game, Depot, is in the library and plays offline", async ({ page }) => {
  await offline(page);
  await openLibrary(page);
  await search(page, "depot");
  await page.keyboard.press("Enter");

  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
});

test("Space and the arrow keys play a game with a keys hint (Depot)", async ({ page }) => {
  await offline(page);
  await openLibrary(page);
  await search(page, "depot");
  await page.keyboard.press("Enter");
  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);

  // Depot's 128×64 screen: (0, 2) is always background. On the title the
  // hatch wall's row (y 8) is dark at the left edge; in a game it's wall.
  // On the dock, the courier's middle row (y 56) covers x 60-64.
  const pixel = async (x: number, y: number) => {
    const g = await sampleScreen(page, 128, 64);
    return g.rgb[y * 128 + x].join();
  };
  const background = await pixel(0, 2);
  expect(await pixel(2, 8)).toBe(background);

  await page.keyboard.down("Space"); // Depot's "a" is its start key, 6
  await expect.poll(() => pixel(2, 8), { intervals: [200] }).not.toBe(background);
  await page.keyboard.up("Space");
  await expect.poll(() => pixel(62, 56), { intervals: [200] }).not.toBe(background);

  await page.keyboard.down("ArrowLeft"); // left is 7: one hop of 4 px
  await page.waitForTimeout(100);
  await page.keyboard.up("ArrowLeft");
  await expect.poll(() => pixel(62, 56), { intervals: [200] }).toBe(background);
});

test("Depot's best score comes back after a reload (saved flag registers)", async ({ page }) => {
  const bundled = JSON.parse(readFileSync(new URL("../../assets/roms/bundled.json", import.meta.url), "utf8"));
  const depot: string = bundled.find((e: { id: string }) => e.id === "depot").sha1;

  // The title's best-score digits: x 67-86, y 31-35 of Depot's 128×64 screen.
  const digits = async () => {
    const g = await sampleScreen(page, 128, 64);
    const rows: string[] = [];
    for (let y = 31; y <= 35; y++) {
      rows.push(g.rgb.slice(y * 128 + 67, y * 128 + 87).map((c) => c.join()).join(" "));
    }
    return rows.join("\n");
  };
  const title = async () => {
    await page.goto(`/?rom=${depot}&shader=clean`);
    await waitForApp(page);
    await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
    await page.waitForTimeout(300); // the title takes three frames
    return digits();
  };

  await offline(page);
  const fresh = await title(); // BEST 0000

  // What the app saves after a best score of 0130 (app/flags.go): the
  // marker, then the digits two to a byte.
  await page.evaluate(([k, v]) => localStorage.setItem(k, v), [`chippie.flags/${depot}`, "d3900130000000000000000000000000"]);
  expect(await title()).not.toBe(fresh);
});

test("a coming-soon game can't be started", async ({ page }) => {
  await offline(page);
  await openLibrary(page);
  await search(page, "clock program");
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

  // Settings rows: TV (with more than one skin), Platform, Speed, …
  await page.keyboard.press("F2");
  await page.waitForTimeout(300);
  for (let i = 0; i < 2; i++) {
    await page.keyboard.press("ArrowDown"); // to Speed
    await page.waitForTimeout(100);
  }
  await page.keyboard.press("ArrowRight"); // 200 → 300
  await page.waitForTimeout(300);
  await page.keyboard.press("Escape");
  await page.waitForTimeout(300);

  const key = "chippie.overrides/2cdcb3c29a5f013a991db5909ca8e18e27b3c42b";
  // Polled: under load the app can be a few frames behind the key presses.
  const stored = () => page.evaluate((k) => localStorage.getItem(k), key);
  await expect.poll(stored).toBe('{"tickrate":300}');

  // Reset to recommended clears it again.
  await page.reload();
  await waitForApp(page);
  await expect.poll(() => running(page), { intervals: [500] }).toBe(true);
  await page.keyboard.press("F2");
  await page.waitForTimeout(300);
  for (let i = 0; i < 10; i++) {
    await page.keyboard.press("ArrowDown"); // to "Reset to recommended"
    await page.waitForTimeout(100);
  }
  await page.keyboard.press("Enter");
  await page.waitForTimeout(300);

  await expect.poll(stored).toBeNull();
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
